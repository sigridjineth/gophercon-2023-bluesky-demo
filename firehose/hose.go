package firehose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/api/label"
	"github.com/bluesky-social/indigo/events"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/repo"
	"github.com/bluesky-social/indigo/repomgr"
	"github.com/gorilla/websocket"
	"github.com/opentracing/opentracing-go/log"
	"gopherconkorea-2023-wss-demo/bluesky"
	"net/http"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type FirehoseConfig struct {
	Authed       bool
	MinFollowers int64
	Likes        bool
	Save         bool
}

func RunWebSocket(cfg FirehoseConfig, blueskyClient *bluesky.Client, connectionID string) error {
	// create a session
	region := "ap-northeast-2"
	stage := "gophercon"
	apiID := "1qn7t1vw4m"
	endpoint := fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/%s", apiID, region, stage)

	acfg, aerr := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: region,
				}, nil
			},
		)),
	)
	if aerr != nil {
		panic("configuration error, " + aerr.Error())
	}

	agClient := apigatewaymanagementapi.NewFromConfig(acfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer stop()

	arg := "wss://bsky.social/xrpc/com.atproto.sync.subscribeRepos"
	var err error

	fmt.Println("dialing: ", arg)
	d := websocket.DefaultDialer
	con, _, err := d.Dial(arg, http.Header{})
	if err != nil {
		return fmt.Errorf("dial failure: %w", err)
	}

	// Here, instead of printing the message, we are sending it back to the agClient
	message := fmt.Sprintf("dialing: %s", arg)
	fmt.Printf(connectionID)
	_, err = agClient.PostToConnection(context.TODO(), &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: &connectionID,
		Data:         []byte(message),
	})
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	fmt.Println("Stream Started", time.Now().Format(time.RFC3339))
	defer func() {
		fmt.Println("Stream Exited", time.Now().Format(time.RFC3339))
	}()

	go func() {
		<-ctx.Done()
		_ = con.Close()
	}()

	return events.HandleRepoStream(ctx, con, &events.RepoStreamCallbacks{
		RepoCommit: func(evt *comatproto.SyncSubscribeRepos_Commit) error {
			rr, err := repo.ReadRepoFromCar(ctx, bytes.NewReader(evt.Blocks))
			if err != nil {
				fmt.Println(err)
			} else {

				for _, op := range evt.Ops {
					ek := repomgr.EventKind(op.Action)
					switch ek {
					case repomgr.EvtKindCreateRecord, repomgr.EvtKindUpdateRecord:
						rc, rec, err := rr.GetRecord(ctx, op.Path)
						if err != nil {
							e := fmt.Errorf("getting record %s (%s) within seq %d for %s: %w", op.Path, *op.Cid, evt.Seq, evt.Repo, err)
							log.Error(e)
							return nil
						}
						if lexutil.LexLink(rc) != *op.Cid {
							return fmt.Errorf("mismatch in record and op cid: %s != %s", rc, *op.Cid)
						}
						banana := lexutil.LexiconTypeDecoder{
							Val: rec,
						}

						var pst = appbsky.FeedPost{}
						b, err := banana.MarshalJSON()
						if err != nil {
							fmt.Println(err)
						}
						err = json.Unmarshal(b, &pst)
						if err != nil {
							fmt.Println(err)
						}

						var userProfile *appbsky.ActorDefs_ProfileViewDetailed
						var replyUserProfile *appbsky.ActorDefs_ProfileViewDetailed
						userProfile, err = appbsky.ActorGetProfile(context.TODO(), blueskyClient.XrpcClient, evt.Repo)
						if pst.Reply != nil {
							replyUserProfile, err = appbsky.ActorGetProfile(context.TODO(), blueskyClient.XrpcClient, strings.Split(pst.Reply.Parent.Uri, "/")[2])
							if err != nil {
								fmt.Println(err)
							}
						}

						if pst.LexiconTypeID == "app.bsky.feed.post" {
							err := PrintPost(cfg.MinFollowers, pst, userProfile, replyUserProfile, nil, op.Path, connectionID, agClient)
							if err != nil {
								return err
							}
						} else if pst.LexiconTypeID == "app.bsky.feed.like" && cfg.Likes {
							var like = appbsky.FeedLike{}
							err = json.Unmarshal(b, &like)
							if err != nil {
								fmt.Println(err)
							}
							likedDid := strings.Split(like.Subject.Uri, "/")[2]
							rrb, err := comatproto.SyncGetRepo(ctx, blueskyClient.XrpcClient, likedDid, "", "")
							if err != nil {
								fmt.Println(err)
								continue
							}
							rr, err := repo.ReadRepoFromCar(ctx, bytes.NewReader(rrb))
							if err != nil {
								fmt.Println(err)
								continue
							}
							_, rec, err := rr.GetRecord(ctx, like.Subject.Uri[strings.LastIndex(like.Subject.Uri[:strings.LastIndex(like.Subject.Uri, "/")], "/")+1:])
							if err != nil {
								log.Error(err)
								return nil
							}
							banana := lexutil.LexiconTypeDecoder{
								Val: rec,
							}
							var pst = appbsky.FeedPost{}
							b, err := banana.MarshalJSON()
							if err != nil {
								fmt.Println(err)
							}
							err = json.Unmarshal(b, &pst)
							if err != nil {
								fmt.Println(err)
							}
							likedUserProfile, err := appbsky.ActorGetProfile(context.TODO(), blueskyClient.XrpcClient, likedDid)
							if err != nil {
								fmt.Println(err)
							}
							PrintPost(cfg.MinFollowers, pst, likedUserProfile, nil, userProfile, like.Subject.Uri[strings.LastIndex(like.Subject.Uri, "/")+1:], connectionID, agClient)
						}
					}
				}
			}
			return nil
		},
		RepoHandle: func(handle *comatproto.SyncSubscribeRepos_Handle) error {
			b, err := json.Marshal(handle)
			if err != nil {
				return err
			}
			fmt.Println("RepoHandle")
			fmt.Println(string(b))
			return nil
		},
		RepoInfo: func(info *comatproto.SyncSubscribeRepos_Info) error {

			b, err := json.Marshal(info)
			if err != nil {
				return err
			}
			fmt.Println("RepoInfo")
			fmt.Println(string(b))

			return nil
		},
		RepoMigrate: func(mig *comatproto.SyncSubscribeRepos_Migrate) error {
			b, err := json.Marshal(mig)
			if err != nil {
				return err
			}
			fmt.Println("RepoMigrate")
			fmt.Println(string(b))
			return nil
		},
		RepoTombstone: func(tomb *comatproto.SyncSubscribeRepos_Tombstone) error {
			b, err := json.Marshal(tomb)
			if err != nil {
				return err
			}
			fmt.Println("RepoTombstone")
			fmt.Println(string(b))
			return nil
		},
		LabelLabels: func(labels *label.SubscribeLabels_Labels) error {
			b, err := json.Marshal(labels)
			if err != nil {
				return err
			}
			fmt.Println("LabelLabels")
			fmt.Println(string(b))
			return nil
		},
		LabelInfo: func(info *label.SubscribeLabels_Info) error {
			b, err := json.Marshal(info)
			if err != nil {
				return err
			}
			fmt.Println("LabelInfo")
			fmt.Println(string(b))
			return nil
		},

		Error: func(errf *events.ErrorFrame) error {
			return fmt.Errorf("error frame: %s: %s", errf.Error, errf.Message)
		},
	})
}

func PrintPost(mf int64, pst appbsky.FeedPost, userProfile, replyUserProfile, likingUserProfile *appbsky.ActorDefs_ProfileViewDetailed, postPath string, connectionID string, client *apigatewaymanagementapi.Client) error {
	if userProfile != nil && userProfile.FollowersCount != nil {
		var enoughfollowers bool
		if *userProfile.FollowersCount >= mf {
			enoughfollowers = true
		}
		if likingUserProfile != nil {
			if *likingUserProfile.FollowersCount >= mf {
				enoughfollowers = true
			}
		}
		if enoughfollowers {
			var rply, likedTxt string
			if pst.Reply != nil && replyUserProfile != nil && replyUserProfile.FollowersCount != nil {
				rply = " ➡️ " + replyUserProfile.Handle + ":" + strconv.Itoa(int(*userProfile.FollowersCount)) + "\n" //+ "https://staging.bsky.app/profile/" + strings.Split(pst.Reply.Parent.Uri, "/")[2] + "/post/" + path.Base(pst.Reply.Parent.Uri) + "\n"
			} else if likingUserProfile != nil {
				likedTxt = likingUserProfile.Handle + ":" + strconv.Itoa(int(*likingUserProfile.FollowersCount)) + " ❤️ "
				rply = ":\n"
			} else {
				rply = ":\n"
			}

			url := "https://staging.bsky.app/profile/" + userProfile.Handle + "/post/" + path.Base(postPath)
			fmtdstring := likedTxt + userProfile.Handle + ":" + strconv.Itoa(int(*userProfile.FollowersCount)) + rply + pst.Text + "\n" + url + "\n"
			fmt.Println(fmtdstring)

			// Send the post details to the client
			_, err := client.PostToConnection(context.TODO(), &apigatewaymanagementapi.PostToConnectionInput{
				ConnectionId: &connectionID,
				Data:         []byte(fmtdstring),
			})
			if err != nil {
				return fmt.Errorf("failed to post message: %w", err)
			}
		}
	}
	return nil
}
