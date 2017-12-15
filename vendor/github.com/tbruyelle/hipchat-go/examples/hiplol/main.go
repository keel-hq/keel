package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tbruyelle/hipchat-go/hipchat"
)

var (
	token  = flag.String("token", "", "The HipChat AuthToken")
	roomId = flag.String("room", "", "The HipChat room id")
	test   = flag.Bool("t", false, "Enable auth_test parameter")
)

func main() {
	flag.Parse()
	if *token == "" || *roomId == "" {
		flag.PrintDefaults()
		return
	}
	hipchat.AuthTest = *test

	c := hipchat.NewClient(*token)

	notifRq := &hipchat.NotificationRequest{Message: "Hey there!"}

	resp, err := c.Room.Notification(*roomId, notifRq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during room notification %q\n", err)
		fmt.Fprintf(os.Stderr, "Server returns %+v\n", resp)
		return
	}

	if hipchat.AuthTest {
		_, ok := hipchat.AuthTestResponse["success"]
		fmt.Println("Authentification succeed :", ok)
	} else {
		fmt.Println("Lol sent !")
	}
}
