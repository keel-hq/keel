package main

import (
	"fmt"

	"github.com/daneharrigan/hipchat"
)

func main() {
	user := "11111_22222"
	pass := "secret"
	resource := "bot"

	client, err := hipchat.NewClient(user, pass, resource)
	if err != nil {
		fmt.Printf("client error: %s\n", err)
		return
	}

	client.RequestUsers()

	select {
	case users := <-client.Users():
		for _, user := range users {
			if user.Id == client.Id {
				fmt.Printf("name: %s\n", user.Name)
				fmt.Printf("mention: %s\n", user.MentionName)
			}
		}
	}
}
