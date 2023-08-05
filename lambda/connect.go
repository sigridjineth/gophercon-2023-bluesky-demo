package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"gopherconkorea-2023-wss-demo/bluesky"
	"gopherconkorea-2023-wss-demo/firehose"
)

const TableName = "WebSocketConnections"

func HandleRequest(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	cfg := firehose.FirehoseConfig{
		Authed:       true,
		MinFollowers: 0,
		Likes:        false,
		Save:         false,
	}

	blueskyClient, err := bluesky.Dial(ctx, "https://bsky.social")
	if err != nil {
		return err
	}
	defer blueskyClient.Close()

	err = blueskyClient.Login(ctx, "golangkorea.bsky.social", "DEMO")
	if err != nil {
		return err
	}

	// Save connection ID to DynamoDB
	dynamoCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	dynamoClient := dynamodb.NewFromConfig(dynamoCfg)
	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item: map[string]types.AttributeValue{
			"connectionId": &types.AttributeValueMemberS{Value: event.RequestContext.ConnectionID},
		},
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "OK",
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
