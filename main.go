package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gophercon-2023-demo/client"
	bskyImpl "gophercon-2023-demo/lambda"
	"log"
	"net/http"
	"strings"
)

var (
	blueskyHandle = "golangkorea.bsky.social"
	blueskyAppkey = "DEMO"
)

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Request: %+v\n", request)
	authHeader := request.Headers["Authorization"]
	authParts := strings.Split(authHeader, " ")

	if len(authParts) != 2 || authParts[0] != "Bearer" || authParts[1] != blueskyAppkey {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusUnauthorized}, nil
	}

	client, err := client.Dial(ctx, client.ServerBskySocial)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}
	log.Printf("Logging in with handle: %s, appkey: %s\n", blueskyHandle, blueskyAppkey)

	err = client.Login(ctx, blueskyHandle, blueskyAppkey)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	handle := request.PathParameters["handle"]
	if handle == "" {
		handle = request.PathParameters["proxy"]
	}

	switch {
	case strings.HasPrefix(request.Path, "/profile"):
		return bskyImpl.GetProfile(ctx, client, handle)
	case strings.HasPrefix(request.Path, "/avatar"):
		return bskyImpl.GetAvatar(ctx, client, handle)
	case strings.HasPrefix(request.Path, "/banner"):
		return bskyImpl.GetBanner(ctx, client, handle)
	case strings.HasPrefix(request.Path, "/followers/full"):
		return bskyImpl.GetFollowersFull(ctx, client, handle)
	case strings.HasPrefix(request.Path, "/following/full"):
		return bskyImpl.GetFollowingFull(ctx, client, handle)
	case strings.HasPrefix(request.Path, "/followers/short"):
		return bskyImpl.GetFollowersShort(ctx, client, handle)
	case strings.HasPrefix(request.Path, "/following/short"):
		return bskyImpl.GetFollowingShort(ctx, client, handle)
	case strings.HasPrefix(request.Path, "/blob"):
		return bskyImpl.GetBlob(ctx, client, request)
	default:
		return events.APIGatewayProxyResponse{StatusCode: http.StatusNotFound}, nil
	}
}
