package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pkg/errors"
	"gopherconkorea-2023-wss-demo/bluesky"
	"gopherconkorea-2023-wss-demo/firehose"
)

const TableName = "WebSocketConnections"

func HandleRequest(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Capture the connectionId from the event
	connectionID := event.RequestContext.ConnectionID

	// Retrieve the connection details from DynamoDB
	connDetails, err := GetConnection(ctx, connectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("error retrieving connection details: %v", err)
	}

	fmt.Println("Retrieved connection details:", connDetails)

	cfg := firehose.FirehoseConfig{
		Authed:       true,
		MinFollowers: 0,
		Likes:        false,
		Save:         false,
	}

	blueskyClient, err := bluesky.Dial(ctx, "https://bsky.social")
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}
	defer blueskyClient.Close()

	err = blueskyClient.Login(ctx, "golangkorea.bsky.social", "DEMO")
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	err = firehose.RunWebSocket(cfg, blueskyClient, connectionID)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(HandleRequest)
}

type Connection struct {
	ConnectionID string
}

func GetConnection(ctx context.Context, connectionID string) (*Connection, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	resp, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"connectionId": &types.AttributeValueMemberS{Value: connectionID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting item: %v", err)
	}

	if len(resp.Item) == 0 {
		return nil, errors.New("no connection found")
	}

	connectionIdAttr, ok := resp.Item["connectionId"]
	if !ok || connectionIdAttr == nil {
		return nil, errors.New("connectionId not found or is nil")
	}

	connection := &Connection{
		ConnectionID: connectionIdAttr.(*types.AttributeValueMemberS).Value,
	}

	return connection, nil
}
