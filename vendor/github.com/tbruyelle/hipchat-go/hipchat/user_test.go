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

func TestUserShareFile(t *testing.T) {
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

	mux.HandleFunc("/user/1/share/file", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		body, _ := ioutil.ReadAll(r.Body)

		if string(body) != want {
			t.Errorf("Request body \n%+v\n,want \n\n%+v", string(body), want)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	args := &ShareFileRequest{Path: tempFile.Name(), Message: "Hello there", Filename: "hipfile"}
	_, err = client.User.ShareFile("1", args)
	if err != nil {
		t.Fatalf("User.ShareFile returns an error %v", err)
	}
}

func TestUserMessage(t *testing.T) {
	setup()
	defer teardown()

	args := &MessageRequest{Message: "m", MessageFormat: "text"}

	mux.HandleFunc("/user/@FirstL/message", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		v := new(MessageRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, args) {
			t.Errorf("Request body %+v, want %+v", v, args)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.User.Message("@FirstL", args)
	if err != nil {
		t.Fatalf("User.Message returns an error %v", err)
	}
}

func TestUserView(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/user/@FirstL", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, `
			{
				"created": "2013-11-07T17:57:11+00:00",
				"email": "user@example.com",
				"group": {
					"id": 1234,
					"links": {
						"self": "https://api.hipchat.com/v2/group/1234"
					},
					"name": "Example"
				},
				"id": 1,
				"is_deleted": false,
				"is_group_admin": true,
				"is_guest": false,
				"last_active": "1421029691",
				"links": {
					"self": "https://api.hipchat.com/v2/user/1"
				},
				"mention_name": "FirstL",
				"name": "First Last",
				"photo_url": "https://bitbucket-assetroot.s3.amazonaws.com/c/photos/2014/Mar/02/hipchat-pidgin-theme-logo-571708621-0_avatar.png",
				"presence": {
						"client": {
							"type": "http://hipchat.com/client/mac",
							"version": "151"
						},
						"is_online": true,
						"show": "chat"
				},
				"timezone": "America/New_York",
				"title": "Test user",
				"xmpp_jid": "1@chat.hipchat.com"
			}`)
	})
	want := &User{XmppJid: "1@chat.hipchat.com",
		IsDeleted:    false,
		Name:         "First Last",
		LastActive:   "1421029691",
		Title:        "Test user",
		Presence:     UserPresence{Show: "chat", IsOnline: true},
		Created:      "2013-11-07T17:57:11+00:00",
		ID:           1,
		MentionName:  "FirstL",
		IsGroupAdmin: true,
		Timezone:     "America/New_York",
		IsGuest:      false,
		Email:        "user@example.com",
		PhotoURL:     "https://bitbucket-assetroot.s3.amazonaws.com/c/photos/2014/Mar/02/hipchat-pidgin-theme-logo-571708621-0_avatar.png",
		Links:        Links{Self: "https://api.hipchat.com/v2/user/1"}}

	user, _, err := client.User.View("@FirstL")
	if err != nil {
		t.Fatalf("User.View returns an error %v", err)
	}
	if !reflect.DeepEqual(want, user) {
		t.Errorf("User.View returned %+v, want %+v", user, want)
	}
}

func TestUserList(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"start-index":     "1",
			"max-results":     "100",
			"include-guests":  "true",
			"include-deleted": "true",
		})
		fmt.Fprintf(w, `
            {
              "items": [
                {
                  "id": 1,
                  "links": {
                    "self": "https:\/\/api.hipchat.com\/v2\/user\/1"
                  },
                  "mention_name": "FirstL",
                  "name": "First Last"
                }
              ],
              "startIndex": 0,
              "maxResults": 100,
              "links": {
                "self": "https:\/\/api.hipchat.com\/v2\/user"
              }
            }`)
	})
	want := []User{
		{
			ID:          1,
			Name:        "First Last",
			MentionName: "FirstL",
			Links:       Links{Self: "https://api.hipchat.com/v2/user/1"},
		},
	}

	opt := &UserListOptions{
		ListOptions{StartIndex: 1, MaxResults: 100},
		true, true,
	}

	users, _, err := client.User.List(opt)
	if err != nil {
		t.Fatalf("User.List returned an error %v", err)
	}
	if !reflect.DeepEqual(want, users) {
		t.Errorf("User.List returned %+v, want %+v", users, want)
	}
}

func TestUserUpdate(t *testing.T) {
	setup()
	defer teardown()

	pArgs := UpdateUserPresenceRequest{Status: "status", Show: UserPresenceShowDnd}
	userArgs := &UpdateUserRequest{Name: "n", Presence: pArgs, MentionName: "mn", Email: "e"}

	mux.HandleFunc("/user/@FirstL", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		v := new(UpdateUserRequest)
		json.NewDecoder(r.Body).Decode(v)

		if !reflect.DeepEqual(v, userArgs) {
			t.Errorf("Request body %+v, want %+v", v, userArgs)
		}
	})

	_, err := client.User.Update("@FirstL", userArgs)
	if err != nil {
		t.Fatalf("User.Update returns an error %v", err)
	}
}
