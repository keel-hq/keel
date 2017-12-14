package hipchat

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestEmoticonList(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/emoticon", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"start-index": "1",
			"max-results": "100",
			"type":        "type",
		})
		fmt.Fprintf(w, `{
			"items": [{"id":1, "url":"u", "shortcut":"s", "links":{"self":"s"}}],
			"startIndex": 1,
			"maxResults": 1,
			"links":{"self":"s", "prev":"p", "next":"n"}
		}`)
	})
	want := &Emoticons{
		Items:      []Emoticon{{ID: 1, URL: "u", Shortcut: "s", Links: Links{Self: "s"}}},
		StartIndex: 1,
		MaxResults: 1,
		Links:      PageLinks{Links: Links{Self: "s"}, Prev: "p", Next: "n"},
	}

	opt := &EmoticonsListOptions{ListOptions{1, 100}, "type"}
	emos, _, err := client.Emoticon.List(opt)
	if err != nil {
		t.Fatalf("Emoticon.List returned an error %v", err)
	}
	if !reflect.DeepEqual(want, emos) {
		t.Errorf("Emoticon.List returned %+v, want %+v", emos, want)
	}
}
