package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"gophercon-2023-demo/blob"
	"gophercon-2023-demo/client"
	"log"
	"net/http"
)

func GetProfile(ctx context.Context, client *client.Client, handle string) (events.APIGatewayProxyResponse, error) {
	log.Printf("Fetching profile for handle: %s\n", handle)
	profile, err := client.FetchProfile(ctx, handle)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	profileJson, err := json.Marshal(profile)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(profileJson),
	}, nil
}

func GetAvatar(ctx context.Context, client *client.Client, handle string) (events.APIGatewayProxyResponse, error) {
	profile, err := client.FetchProfile(ctx, handle)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	err = profile.ResolveAvatar(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	bounds, err := json.Marshal(profile.Avatar.Bounds())
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{Body: string(bounds), StatusCode: http.StatusOK}, nil
}

func GetBanner(ctx context.Context, client *client.Client, handle string) (events.APIGatewayProxyResponse, error) {
	//handle := request.PathParameters["handle"]
	profile, err := client.FetchProfile(ctx, handle)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	err = profile.ResolveBanner(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       fmt.Sprintf("%v", profile.Banner.Bounds()),
	}

	return response, nil
}

func GetFollowersFull(ctx context.Context, client *client.Client, handle string) (events.APIGatewayProxyResponse, error) {
	//handle, ok := request.PathParameters["handle"]
	//if !ok {
	//	return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}, nil
	//}
	profile, err := client.FetchProfile(ctx, handle)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	err = profile.ResolveFollowers(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	jsonFollowers, _ := json.Marshal(profile.Followers)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonFollowers),
	}, nil
}

func GetFollowingFull(ctx context.Context, client *client.Client, handle string) (events.APIGatewayProxyResponse, error) {
	//handle, ok := request.PathParameters["handle"]
	//if !ok {
	//	return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}, nil
	//}

	profile, err := client.FetchProfile(ctx, handle)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	err = profile.ResolveFollowing(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	jsonFollowers, _ := json.Marshal(profile.Followers)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonFollowers),
	}, nil
}

func GetFollowersShort(ctx context.Context, client *client.Client, handle string) (events.APIGatewayProxyResponse, error) {
	//handle := request.PathParameters["handle"]
	profile, err := client.FetchProfile(ctx, handle)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}, nil
	}

	followerc, errc := profile.StreamFollowers(ctx)

	if err := <-errc; err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}, nil
	}

	var followersName []string
	for follower := range followerc {
		followersName = append(followersName, follower.Name)
	}

	response, err := json.Marshal(followersName)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Error marshalling response"}, nil
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(response)}, nil
}

func GetFollowingShort(ctx context.Context, client *client.Client, handle string) (events.APIGatewayProxyResponse, error) {
	profile, err := client.FetchProfile(ctx, handle)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}, nil
	}

	followingFull, errc := profile.StreamFollowing(ctx)

	if err := <-errc; err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}, nil
	}

	var followingName []string
	for following := range followingFull {
		followingName = append(followingName, following.Name)
	}

	response, err := json.Marshal(followingName)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Error marshalling response"}, nil
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: string(response)}, nil
}

func GetBlob(ctx context.Context, client *client.Client, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	did := request.PathParameters["did"]
	cid := request.PathParameters["cid"]

	blobRawResponse, err := blob.RetrieveBlob(string('.'), did, cid)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	blobJSON, err := json.Marshal(blobRawResponse)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{Body: string(blobJSON), StatusCode: http.StatusOK}, nil
}
