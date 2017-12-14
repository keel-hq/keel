package hipchat

import (
	// "encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestWebhookList(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room/1/webhook", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"max-results": "100",
			"start-index": "1",
		})
		fmt.Fprintf(w, `
		{
			"items":[
			  {"name":"a", "key": "a", "pattern":"a", "event":"message_received", "url":"h", "id":1, "links":{"self":"s"}},
				{"name":"b", "key": "b", "pattern":"b", "event":"message_received", "url":"h", "id":2, "links":{"self":"s"}}
			],
			"links":{"self":"s", "prev":"a", "next":"b"},
			"startIndex":0,
			"maxResults":10
		}`)
	})

	want := &WebhookList{
		Webhooks: []Webhook{
			{
				Name:    "a",
				Key:     "a",
				Pattern: "a",
				Event:   "message_received",
				URL:     "h",
				ID:      1,
				Links:   Links{Self: "s"},
			},
			{
				Name:    "b",
				Key:     "b",
				Pattern: "b",
				Event:   "message_received",
				URL:     "h",
				ID:      2,
				Links:   Links{Self: "s"},
			},
		},
		StartIndex: 0,
		MaxResults: 10,
		Links:      PageLinks{Links: Links{Self: "s"}, Prev: "a", Next: "b"},
	}

	opt := &ListWebhooksOptions{ListOptions{1, 100}}

	actual, _, err := client.Room.ListWebhooks("1", opt)
	if err != nil {
		t.Fatalf("Room.ListWebhooks returns an error %v", err)
	}
	if !reflect.DeepEqual(want, actual) {
		t.Errorf("Room.ListWebhooks returned %+v, want %+v", actual, want)
	}
}

func TestWebhookDelete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room/1/webhook/2", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Room.DeleteWebhook("1", "2")
	if err != nil {
		t.Fatalf("Room.Update returns an error %v", err)
	}
}
