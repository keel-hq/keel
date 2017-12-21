package main

import (
	"fmt"
	"strings"

	"github.com/daneharrigan/hipchat"
)

func main() {
	user := "11111_22222"
	pass := "secret"
	resource := "bot"
	roomJid := "11111_room_name@conf.hipchat.com"
	fullName := "Some Bot"
	mentionName := "SomeBot"

	client, err := hipchat.NewClient(user, pass, resource)
	if err != nil {
		fmt.Printf("client error: %s\n", err)
		return
	}

	client.Status("chat")
	client.Join(roomJid, fullName)

	go client.KeepAlive()

	go func() {
		for {
			select {
			case message := <-client.Messages():
				if strings.HasPrefix(message.Body, "@"+mentionName) {
					client.Say(roomJid, fullName, "Hello")
				}
			}
		}
	}()
	select {}
}
