# hipchat

This is a abstraction in golang to Hipchat's implementation of XMPP. It communicates over
TLS and requires zero knowledge of XML or the XMPP protocol.

* Examples [available here][1]
* Documentation [available here][2]

### bot building

Hipchat treats the "bot" resource differently from any other resource connected to their service. When connecting to Hipchat with a resource of "bot", a chat history will not be sent. Any other resource will receive a chat history.

### example/hello.go

```go
package main

import (
	"github.com/daneharrigan/hipchat"
)

func main() {
	user := "11111_22222"
	pass := "secret"
	resource := "bot"
	roomJid := "11111_room_name@conf.hipchat.com"
	fullName := "Some Bot"

	client, err := hipchat.NewClient(user, pass, resource)
	if err != nil {
		fmt.Printf("client error: %s\n", err)
		return
	}

	client.Status("chat")
	client.Join(roomJid, fullName)
	client.Say(roomJid, fullName, "Hello")
	select {}
}
```

[1]: https://github.com/daneharrigan/hipchat/tree/master/example
[2]: http://godoc.org/github.com/daneharrigan/hipchat
