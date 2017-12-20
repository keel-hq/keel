package hipchat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func TestRoomGet(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, `
		{
			"id":1,
			"name":"n",
			"links":{"self":"s"},
			"Participants":[
				{"Name":"n1"},
				{"Name":"n2"}
			],
			"Owner":{"Name":"n1"}
		}`)
	})
	want := &Room{
		ID:           1,
		Name:         "n",
		Links:        RoomLinks{Links: Links{Self: "s"}},
		Participants: []User{{Name: "n1"}, {Name: "n2"}},
		Owner:        User{Name: "n1"},
	}

	room, _, err := client.Room.Get("1")
	if err != nil {
		t.Fatalf("Room.Get returns an error %v", err)
	}
	if !reflect.DeepEqual(want, room) {
		t.Errorf("Room.Get returned %+v, want %+v", room, want)
	}
}

func TestRoomList(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"start-index":      "1",
			"max-results":      "10",
			"include-private":  "true",
			"include-archived": "true",
		})
		fmt.Fprintf(w, `
		{
			"items": [{"id":1,"name":"n"}],
			"startIndex":1,
			"maxResults":1,
			"links":{"Self":"s"}
		}`)
	})
	want := &Rooms{Items: []Room{{ID: 1, Name: "n"}}, StartIndex: 1, MaxResults: 1, Links: PageLinks{Links: Links{Self: "s"}}}
	opt := &RoomsListOptions{ListOptions{1, 10}, true, true}
	rooms, _, err := client.Room.List(opt)
	if err != nil {
		t.Fatalf("Room.List returns an error %v", err)
	}
	if !reflect.DeepEqual(want, rooms) {
		t.Errorf("Room.List returned %+v, want %+v", rooms, want)
	}
}

func TestRoomGetStatistics(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room/1/statistics", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, `
		{
			"messages_sent":1,
			"last_active":"2016-02-29T09:03:53+00:00"
		}`)
	})
	want := &RoomStatistics{
		MessagesSent: 1,
		LastActive:   "2016-02-29T09:03:53+00:00",
	}

	roomStatistics, _, err := client.Room.GetStatistics("1")
	if err != nil {
		t.Fatalf("Room.GetStatistics returns an error %v", err)
	}
	if !reflect.DeepEqual(want, roomStatistics) {
		t.Errorf("Room.GetStatistics returned %+v, want %+v", roomStatistics, want)
	}
}

func TestRoomNotification(t *testing.T) {
	setup()
	defer teardown()

	args := &NotificationRequest{Color: "red", Message: "m", MessageFormat: "text"}

	mux.HandleFunc("/room/1/notification", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		v := new(NotificationRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Room.Notification("1", args)
	if err != nil {
		t.Fatalf("Room.Notification returns an error %v", err)
	}
}

func TestRoomMessage(t *testing.T) {
	setup()
	defer teardown()

	args := &RoomMessageRequest{Message: "m"}

	mux.HandleFunc("/room/1/message", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		v := new(RoomMessageRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Room.Message("1", args)
	if err != nil {
		t.Fatalf("Room.Message returns an error %v", err)
	}
}

func TestRoomShareFile(t *testing.T) {
	setup()
	defer teardown()

	tempFile, err := ioutil.TempFile(os.TempDir(), "hipfile")
	tempFile.WriteString("go gophers")
	defer os.Remove(tempFile.Name())

	want := "--hipfileboundary\n" +
		"Content-Type: application/json; charset=UTF-8\n" +
		"Content-Disposition: attachment; name=\"metadata\"\n\n" +
		"{\"message\": \"Hello there\"}\n" +
		"--hipfileboundary\n" +
		"Content-Type:  charset=UTF-8\n" +
		"Content-Transfer-Encoding: base64\n" +
		"Content-Disposition: attachment; name=file; filename=hipfile\n\n" +
		"Z28gZ29waGVycw==\n" +
		"--hipfileboundary\n"

	mux.HandleFunc("/room/1/share/file", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		body, _ := ioutil.ReadAll(r.Body)

		if string(body) != want {
			t.Errorf("Request body \n%+v\n,want \n\n%+v", string(body), want)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	args := &ShareFileRequest{Path: tempFile.Name(), Message: "Hello there", Filename: "hipfile"}
	_, err = client.Room.ShareFile("1", args)
	if err != nil {
		t.Fatalf("Room.ShareFile returns an error %v", err)
	}
}

func TestRoomCreate(t *testing.T) {
	setup()
	defer teardown()

	args := &CreateRoomRequest{Name: "n", Topic: "t"}

	mux.HandleFunc("/room", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		v := new(CreateRoomRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
		fmt.Fprintf(w, `{"id":1,"links":{"self":"s"}}`)
	})
	want := &Room{ID: 1, Links: RoomLinks{Links: Links{Self: "s"}}}

	room, _, err := client.Room.Create(args)
	if err != nil {
		t.Fatalf("Room.Create returns an error %v", err)
	}
	if !reflect.DeepEqual(room, want) {
		t.Errorf("Room.Create returns %+v, want %+v", room, want)
	}
}

func TestRoomDelete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Room.Delete("1")
	if err != nil {
		t.Fatalf("Room.Delete returns an error %v", err)
	}
}

func TestRoomUpdate(t *testing.T) {
	setup()
	defer teardown()

	args := &UpdateRoomRequest{Name: "n", Topic: "t"}

	mux.HandleFunc("/room/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		v := new(UpdateRoomRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
	})

	_, err := client.Room.Update("1", args)
	if err != nil {
		t.Fatalf("Room.Update returns an error %v", err)
	}
}

func TestRoomHistory(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room/1/history", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"start-index": "1",
			"max-results": "100",
			"date":        "date",
			"timezone":    "tz",
			"reverse":     "true",
		})
		fmt.Fprintf(w, `
		{
      "items": [
          {
              "date": "2014-11-23T21:23:49.807578+00:00",
              "from": "Test Testerson",
              "id": "f058e668-c9c0-4cd5-9ca5-e2c42b06f3ed",
              "mentions": [],
              "message": "Hey there!",
              "message_format": "html",
              "type": "notification"
          }
      ],
      "links": {
          "self": "https://api.hipchat.com/v2/room/1/history"
      },
      "maxResults": 100,
      "startIndex": 0
		}`)
	})

	opt := &HistoryOptions{
		ListOptions{1, 100}, "date", "tz", true,
	}
	hist, _, err := client.Room.History("1", opt)
	if err != nil {
		t.Fatalf("Room.History returns an error %v", err)
	}

	want := &History{Items: []Message{{Date: "2014-11-23T21:23:49.807578+00:00", From: "Test Testerson", ID: "f058e668-c9c0-4cd5-9ca5-e2c42b06f3ed", Mentions: []User{}, Message: "Hey there!", MessageFormat: "html", Type: "notification"}}, StartIndex: 0, MaxResults: 100, Links: PageLinks{Links: Links{Self: "https://api.hipchat.com/v2/room/1/history"}}}
	if !reflect.DeepEqual(want, hist) {
		t.Errorf("Room.History returned %+v, want %+v", hist, want)
	}
}

func TestRoomLatest(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/room/1/history/latest", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"max-results": "100",
			"timezone":    "tz",
			"not-before":  "notbefore",
		})
		fmt.Fprintf(w, `
		{
      "items": [
          {
              "date": "2014-11-23T21:23:49.807578+00:00",
              "from": "Test Testerson",
              "id": "f058e668-c9c0-4cd5-9ca5-e2c42b06f3ed",
              "mentions": [],
              "message": "Hey there!",
              "message_format": "html",
              "type": "notification"
          }
      ],
      "links": {
          "self": "https://api.hipchat.com/v2/room/1/history/latest"
      },
      "maxResults": 100
		}`)
	})

	opt := &LatestHistoryOptions{
		100, "tz", "notbefore",
	}
	hist, _, err := client.Room.Latest("1", opt)
	if err != nil {
		t.Fatalf("Room.Latest returns an error %v", err)
	}
	want := &History{Items: []Message{{Date: "2014-11-23T21:23:49.807578+00:00", From: "Test Testerson", ID: "f058e668-c9c0-4cd5-9ca5-e2c42b06f3ed", Mentions: []User{}, Message: "Hey there!", MessageFormat: "html", Type: "notification"}}, MaxResults: 100, Links: PageLinks{Links: Links{Self: "https://api.hipchat.com/v2/room/1/history/latest"}}}
	if !reflect.DeepEqual(want, hist) {
		t.Errorf("Room.Latest returned %+v, want %+v", hist, want)
	}
}
func TestRoomGlanceCreate(t *testing.T) {
	setup()
	defer teardown()

	args := &GlanceRequest{
		Key:      "abc",
		Name:     GlanceName{Value: "Test Glance"},
		Target:   "target",
		QueryURL: "qu",
		Icon:     Icon{URL: "i", URL2x: "i"},
	}

	mux.HandleFunc(fmt.Sprintf("/room/1/extension/glance/%s", args.Key), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		v := new(GlanceRequest)
		json.NewDecoder(r.Body).Decode(v)
		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Room.CreateGlance("1", args)
	if err != nil {
		t.Fatalf("Room.CreateGlance returns an error %v", err)
	}
}

func TestRoomGlanceDelete(t *testing.T) {
	setup()
	defer teardown()

	args := &GlanceRequest{
		Key: "abc",
	}

	mux.HandleFunc(fmt.Sprintf("/room/1/extension/glance/%s", args.Key), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Room.DeleteGlance("1", args)
	if err != nil {
		t.Fatalf("Room.DeleteGlance returns an error %v", err)
	}
}

func TestRoomGlanceUpdate(t *testing.T) {
	setup()
	defer teardown()

	args := &GlanceUpdateRequest{
		Glance: []*GlanceUpdate{
			&GlanceUpdate{
				Key: "abc",
				Content: GlanceContent{
					Status: GlanceStatus{Type: "lozenge", Value: AttributeValue{Type: "default", Label: "something"}},
					Label:  AttributeValue{Type: "html", Value: "hello"},
				},
			},
		},
	}

	mux.HandleFunc("/addon/ui/room/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		v := new(GlanceUpdateRequest)
		json.NewDecoder(r.Body).Decode(v)
		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Room.UpdateGlance("1", args)
	if err != nil {
		t.Fatalf("Room.UpdateGlance returns an error %v", err)
	}
}

func TestSetTopic(t *testing.T) {
	setup()
	defer teardown()

	args := &SetTopicRequest{Topic: "t"}

	mux.HandleFunc("/room/1/topic", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		v := new(SetTopicRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
	})

	_, err := client.Room.SetTopic("1", "t")
	if err != nil {
		t.Fatalf("Room.SetTopic returns an error %v", err)
	}
}

func TestInvite(t *testing.T) {
	setup()
	defer teardown()

	args := &InviteRequest{Reason: "r"}

	mux.HandleFunc("/room/1/invite/user", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		v := new(InviteRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
	})

	_, err := client.Room.Invite("1", "user", "r")
	if err != nil {
		t.Fatalf("Room.Invite returns an error %v", err)
	}
}

func TestCardDescriptionJSONEncodeWithString(t *testing.T) {
	description := CardDescription{Value: "This is a test"}
	expected := `"This is a test"`

	encoded, err := json.Marshal(description)
	if err != nil {
		t.Errorf("Encoding of CardDescription failed")
	}

	if string(encoded) != expected {
		t.Fatalf("Encoding of CardDescription failed: %s", encoded)
	}
}

func TestCardDescriptionJSONDecodeWithString(t *testing.T) {
	encoded := []byte(`"This is a test"`)
	expected := CardDescription{Format: "", Value: "This is a test"}

	var actual CardDescription

	err := json.Unmarshal(encoded, &actual)
	if err != nil {
		t.Errorf("Decoding of CardDescription failed: %v", err)
	}

	if actual.Value != expected.Value {
		t.Fatalf("Unexpected CardDescription.Value: %v", actual.Value)
	}

	if actual.Format != expected.Format {
		t.Fatalf("Unexpected CardDescription.Format: %v", actual.Format)
	}
}

func TestCardDescriptionJSONEncodeWithObject(t *testing.T) {
	description := CardDescription{Format: "html", Value: "<strong>This is a test</strong>"}
	expected := `{"format":"html","value":"\u003cstrong\u003eThis is a test\u003c/strong\u003e"}`

	encoded, err := json.Marshal(description)
	if err != nil {
		t.Errorf("Encoding of CardDescription failed")
	}

	if string(encoded) != expected {
		t.Fatalf("Encoding of CardDescription failed: %s", encoded)
	}
}

func TestCardDescriptionJSONDecodeWithObject(t *testing.T) {
	encoded := []byte(`{"format":"html","value":"\u003cstrong\u003eThis is a test\u003c/strong\u003e"}`)
	expected := CardDescription{Format: "html", Value: "<strong>This is a test</strong>"}

	var actual CardDescription

	err := json.Unmarshal(encoded, &actual)
	if err != nil {
		t.Errorf("Decoding of CardDescription failed: %v", err)
	}

	if actual.Value != expected.Value {
		t.Fatalf("Unexpected CardDescription.Value: %v", actual.Value)
	}

	if actual.Format != expected.Format {
		t.Fatalf("Unexpected CardDescription.Format: %v", actual.Format)
	}
}

func TestGlanceUpdateRequestJSONEncodeWithString(t *testing.T) {
	gr := GlanceUpdateRequest{
		Glance: []*GlanceUpdate{
			&GlanceUpdate{
				Key: "abc",
				Content: GlanceContent{
					Status: GlanceStatus{Type: "lozenge", Value: AttributeValue{Type: "default", Label: "something"}},
					Label:  AttributeValue{Type: "html", Value: "hello"},
				},
			},
		},
	}
	expected := `{"glance":[{"key":"abc","content":{"status":{"type":"lozenge","value":{"type":"default","label":"something"}},"label":{"type":"html","value":"hello"}}}]}`

	encoded, err := json.Marshal(gr)
	if err != nil {
		t.Errorf("Encoding of GlanceUpdateRequest failed")
	}

	if string(encoded) != expected {
		t.Fatalf("Encoding of GlanceUpdateRequest failed: %s", encoded)
	}
}

func TestGlanceContentJSONEncodeWithString(t *testing.T) {
	gcTests := []struct {
		gc       GlanceContent
		expected string
	}{
		{
			GlanceContent{
				Status: GlanceStatus{Type: "lozenge", Value: AttributeValue{Type: "default", Label: "something"}},
				Label:  AttributeValue{Type: "html", Value: "hello"},
			},
			`{"status":{"type":"lozenge","value":{"type":"default","label":"something"}},"label":{"type":"html","value":"hello"}}`,
		},
	}

	for _, tt := range gcTests {
		encoded, err := json.Marshal(tt.gc)
		if err != nil {
			t.Errorf("Encoding of GlanceContent failed")
		}

		if string(encoded) != tt.expected {
			t.Fatalf("Encoding of GlanceContent failed: %s", encoded)
		}
	}
}

func TestGlanceContentJSONDecodeWithObject(t *testing.T) {
	gcTests := []struct {
		gc      GlanceContent
		encoded string
	}{
		{
			GlanceContent{
				Status: GlanceStatus{Type: "lozenge", Value: AttributeValue{Type: "default", Label: "something"}},
				Label:  AttributeValue{Type: "html", Value: "hello"},
			},
			`{"status":{"type":"lozenge","value":{"type":"default","label":"something"}},"label":{"type":"html","value":"hello"}}`,
		},
	}

	for _, tt := range gcTests {
		var actual GlanceContent

		err := json.Unmarshal([]byte(tt.encoded), &actual)
		if err != nil {
			t.Errorf("Decoding of GlanceContent failed: %v", err)
		}

		if actual.Status != tt.gc.Status {
			t.Fatalf("Unexpected GlanceContent.Status: %v", actual.Status)
		}

		if actual.Label != tt.gc.Label {
			t.Fatalf("Unexpected GlanceStatus.Label: %v", actual.Label)
		}

		if actual.Metadata != tt.gc.Metadata {
			t.Fatalf("Unexpected GlanceStatus.Metadata %v", actual.Metadata)
		}
	}
}

func TestGlanceStatusJSONEncodeWithString(t *testing.T) {
	gsTests := []struct {
		gs       GlanceStatus
		expected string
	}{
		{GlanceStatus{Type: "lozenge", Value: AttributeValue{Type: "default", Label: "something"}},
			`{"type":"lozenge","value":{"type":"default","label":"something"}}`},
		{GlanceStatus{Type: "icon", Value: Icon{URL: "z", URL2x: "x"}},
			`{"type":"icon","value":{"url":"z","url@2x":"x"}}`},
	}

	for _, tt := range gsTests {
		encoded, err := json.Marshal(tt.gs)
		if err != nil {
			t.Errorf("Encoding of GlanceStatus failed")
		}

		if string(encoded) != tt.expected {
			t.Fatalf("Encoding of GlanceStatus failed: %s", encoded)
		}
	}
}

func TestGlanceStatusJSONDecodeWithObject(t *testing.T) {
	gsTests := []struct {
		gs      GlanceStatus
		encoded string
	}{
		{GlanceStatus{Type: "lozenge", Value: AttributeValue{Type: "default", Label: "something"}},
			`{"type":"lozenge","value":{"type":"default","label":"something"}}`},
		{GlanceStatus{Type: "icon", Value: Icon{URL: "z", URL2x: "x"}},
			`{"type":"icon","value":{"url":"z","url@2x":"x"}}`},
	}

	for _, tt := range gsTests {
		var actual GlanceStatus

		err := json.Unmarshal([]byte(tt.encoded), &actual)
		if err != nil {
			t.Errorf("Decoding of GlanceStatus failed: %v", err)
		}

		if actual.Type != tt.gs.Type {
			t.Fatalf("Unexpected GlanceStatus.Type: %v", actual.Type)
		}

		if actual.Value != tt.gs.Value {
			t.Fatalf("Unexpected GlanceStatus.Value: %v", actual.Value)
		}
	}
}
