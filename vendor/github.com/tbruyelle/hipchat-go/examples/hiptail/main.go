package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/tbruyelle/hipchat-go/hipchat"
)

const (
	maxMsgLen  = 128
	moreString = " [MORE]"
)

var (
	token  = flag.String("token", "", "The HipChat AuthToken")
	roomId = flag.String("room", "", "The HipChat room id")
)

func main() {
	flag.Parse()
	if *token == "" || *roomId == "" {
		flag.PrintDefaults()
		return
	}
	c := hipchat.NewClient(*token)
	hist, resp, err := c.Room.History(*roomId, &hipchat.HistoryOptions{})
	if err != nil {
		fmt.Printf("Error during room history req %q\n", err)
		fmt.Printf("Server returns %+v\n", resp)
		return
	}
	for _, m := range hist.Items {
		from := ""
		switch m.From.(type) {
		case string:
			from = m.From.(string)
		case map[string]interface{}:
			f := m.From.(map[string]interface{})
			from = f["name"].(string)
		}
		msg := m.Message
		if len(m.Message) > (maxMsgLen - len(moreString)) {
			msg = fmt.Sprintf("%s%s", strings.Replace(m.Message[:maxMsgLen], "\n", " - ", -1), moreString)
		}
		fmt.Printf("%s [%s]: %s\n", from, m.Date, msg)
	}
}
