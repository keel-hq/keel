package main

import (
	"flag"
	"fmt"

	"github.com/tbruyelle/hipchat-go/hipchat"
)

var (
	token    = flag.String("token", "", "The HipChat AuthToken")
	roomId   = flag.String("room", "", "The HipChat room id")
	userId   = flag.String("user", "", "The HipChat user id")
	path     = flag.String("path", "", "The file path")
	message  = flag.String("message", "", "The message")
	filename = flag.String("filename", "", "The name of the file")
)

func main() {
	flag.Parse()
	if *token == "" || *path == "" || ((*roomId == "") && (*userId == "")) {
		flag.PrintDefaults()
		return
	}
	c := hipchat.NewClient(*token)

	shareFileRq := &hipchat.ShareFileRequest{Path: *path, Message: *message, Filename: *filename}

	if *roomId != "" {
		resp, err := c.Room.ShareFile(*roomId, shareFileRq)

		if err != nil {
			fmt.Printf("Error during room file share %q\n", err)
			fmt.Printf("Server returns %+v\n", resp)
			return
		}
	}

	if *userId != "" {
		resp, err := c.User.ShareFile(*userId, shareFileRq)

		if err != nil {
			fmt.Printf("Error during user file share %q\n", err)
			fmt.Printf("Server returns %+v\n", resp)
			return
		}
	}

	fmt.Println("File sent !")
}
