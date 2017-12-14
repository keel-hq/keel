# hipchat-go

Go client library for the [HipChat API v2](https://www.hipchat.com/docs/apiv2).

[![GoDoc](https://godoc.org/github.com/tbruyelle/hipchat-go/hipchat?status.svg)](https://godoc.org/github.com/tbruyelle/hipchat-go/hipchat)
[![Build Status](https://travis-ci.org/tbruyelle/hipchat-go.svg??branch=master)](https://travis-ci.org/tbruyelle/hipchat-go)

Currently only a small part of the API is implemented, so pull requests are welcome.

### Usage

```go
import "github.com/tbruyelle/hipchat-go/hipchat"
```

Build a new client, then use the `client.Room` service to spam all the rooms you have access to (not recommended):

```go
c := hipchat.NewClient("<your AuthToken here>")

opt := &hipchat.RoomsListOptions{IncludePrivate:  true, IncludeArchived: true}
rooms, _, err := c.Room.List(opt)
if err != nil {
	panic(err)
}

notifRq := &hipchat.NotificationRequest{Message: "Hey there!"}

for _, room := range rooms.Items {
	_, err := c.Room.Notification(room.Name, notifRq)
	if err != nil {
		panic(err)
	}
}
```

### Testing the auth token

HipChat allows to [test the auth token](https://www.hipchat.com/docs/apiv2/auth#auth_test) by adding the `auth_test=true` param, into any API endpoints.

You can do this with `hipchat-go` by setting the global var `hipchat.AuthTest`. Because the server response will be different from the one defined in the API endpoint, you need to check another global var `AuthTestReponse` to see if the authentication succeeds.

```go
hipchat.AuthTest = true

client.Room.Get(42)

_, ok := hipchat.AuthTestResponse["success"]
fmt.Println("Authentification succeed :", ok)
// Dont forget to reset the variable, or every other API calls
// will be impacted.
hipchat.AuthTest = false
```

---
The code architecture is hugely inspired by [google/go-github](http://github.com/google/go-github).


