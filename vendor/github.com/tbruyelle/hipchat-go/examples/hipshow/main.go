package main

import (
	"flag"
	"fmt"

	"github.com/tbruyelle/hipchat-go/hipchat"
)

var (
	token           = flag.String("token", "", "The HipChat AuthToken")
	maxResults      = flag.Int("maxResults", 5, "Max results per request")
	includePrivate  = flag.Bool("includePrivate", false, "Include private rooms?")
	includeArchived = flag.Bool("includeArchived", false, "Include archived rooms?")
)

func main() {
	flag.Parse()
	if *token == "" {
		flag.PrintDefaults()
		return
	}
	c := hipchat.NewClient(*token)
	startIndex := 0
	totalRequests := 0
	var allRooms []hipchat.Room

	for {
		opt := &hipchat.RoomsListOptions{
			ListOptions:     hipchat.ListOptions{StartIndex: startIndex, MaxResults: *maxResults},
			IncludePrivate:  *includePrivate,
			IncludeArchived: *includeArchived}

		rooms, resp, err := c.Room.List(opt)

		if err != nil {
			fmt.Printf("Error during room list req %q\n", err)
			fmt.Printf("Server returns %+v\n", resp)
			return
		}

		totalRequests++

		allRooms = append(allRooms, rooms.Items...)
		if rooms.Links.Next != "" {
			startIndex += *maxResults
		} else {
			break
		}
	}

	fmt.Printf("Your group has %d rooms, it took %d requests to retrieve all of them:\n",
		len(allRooms), totalRequests)
	for _, r := range allRooms {
		fmt.Printf("%d %s \n", r.ID, r.Name)
	}
}
