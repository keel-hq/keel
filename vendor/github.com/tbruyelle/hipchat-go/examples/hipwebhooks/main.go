package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/tbruyelle/hipchat-go/hipchat"
)

var (
	token  = flag.String("token", "", "The HipChat AuthToken")
	roomId = flag.String("room", "", "A specific Room ID")
	action = flag.String("action", "", "An action to take. Currently supported: create, delete")

	// delete
	webhookId = flag.String("webhook", "", "A specific Room ID")

	// create
	name    = flag.String("name", "", "With action: create, name for the new webhook")
	event   = flag.String("event", "", "With action: create, event for the new webhook (enter, exit, message, notification, topic_change)")
	pattern = flag.String("pattern", "", "With action: create, pattern for the new webhook")
	url     = flag.String("url", "", "With action: create, target URL for the new webhook")
)

func main() {
	flag.Parse()
	if *token == "" {
		flag.PrintDefaults()
		return
	}
	c := hipchat.NewClient(*token)

	if *action == "" {
		if *roomId == "" {
			// If no room is given, look up all rooms and all of their webhooks
			rooms, resp, err := c.Room.List()
			handleRequestError(resp, err)

			for _, room := range rooms.Items {
				fmt.Printf("%-25v%10v\n", room.Name, room.ID)

				hooks, resp, err := c.Room.ListWebhooks(room.ID, nil)
				handleRequestError(resp, err)

				for _, webhook := range hooks.Webhooks {
					fmt.Printf("  %v %v\t%v\t%v\t%v\n", webhook.Name, webhook.ID, webhook.Event, webhook.URL, webhook.Links.Self)
				}

				fmt.Println("---")
			}
		} else {
			// If room is given, just get the webhooks for that room
			hooks, resp, err := c.Room.ListWebhooks(*roomId, nil)
			handleRequestError(resp, err)

			for _, webhook := range hooks.Webhooks {
				fmt.Printf("  %v %v\t%v\t%v\t%v\n", webhook.Name, webhook.ID, webhook.Event, webhook.URL, webhook.Links.Self)
			}
		}
	} else if *action == "create" {
		if *roomId == "" {
			fmt.Println("roomId is required for webhook creation")
			flag.PrintDefaults()
			return
		}

		webhook, resp, err := c.Room.CreateWebhook(*roomId, &hipchat.CreateWebhookRequest{
			Name:    *name,
			Event:   "room_" + *event,
			Pattern: *pattern,
			URL:     *url,
		})
		handleRequestError(resp, err)
		fmt.Printf("%v\t%v\t%v\t%v\n", webhook.Name, webhook.Event, webhook.URL, webhook.Links.Self)
	} else if *action == "delete" {
		if *roomId == "" {
			fmt.Println("roomId is required for webhook deletion")
			flag.PrintDefaults()
			return
		}

		if *webhookId == "" {
			fmt.Println("webhookId is required for webhook deletion")
			flag.PrintDefaults()
			return
		}

		resp, err := c.Room.DeleteWebhook(*roomId, *webhookId)
		handleRequestError(resp, err)

		fmt.Println("Deleted webhook with id", *webhookId)
	}

}

func handleRequestError(resp *http.Response, err error) {
	if err != nil {
		if resp != nil {
			fmt.Printf("Request Failed:\n%+v\n", resp)
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("%+v\n", body)
		} else {
			fmt.Printf("Request failed, response is nil")
		}
		panic(err)
	}
}
