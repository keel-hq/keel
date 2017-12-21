package hipchat

import (
	"bytes"
	"encoding/xml"
	"errors"
	"time"

	"github.com/daneharrigan/hipchat/xmpp"
)

const (
	defaultAuthType = "plain" // or "oauth"
	defaultConf     = "conf.hipchat.com"
	defaultDomain   = "chat.hipchat.com"
	defaultHost     = "chat.hipchat.com"
)

// A Client represents the connection between the application to the HipChat
// service.
type Client struct {
	AuthType string
	Username string
	Password string
	Resource string
	Id       string

	// private
	mentionNames    map[string]string
	connection      *xmpp.Conn
	receivedUsers   chan []*User
	receivedRooms   chan []*Room
	receivedMessage chan *Message
	host            string
	domain          string
	conf            string
}

// A Message represents a message received from HipChat.
type Message struct {
	From        string
	To          string
	Body        string
	MentionName string
}

// A User represents a member of the HipChat service.
type User struct {
	Email       string
	Id          string
	Name        string
	MentionName string
}

// A Room represents a room in HipChat the Client can join to communicate with
// other members..
type Room struct {
	Id              string
	LastActive      string
	Name            string
	NumParticipants string
	Owner           string
	Privacy         string
	RoomId          string
	Topic           string
}

// NewClient creates a new Client connection from the user name, password and
// resource passed to it. It uses default host URL and conf URL.
func NewClient(user, pass, resource, authType string) (*Client, error) {
	return NewClientWithServerInfo(user, pass, resource, authType, defaultHost, defaultDomain, defaultConf)
}

// NewClientWithServerInfo creates a new Client connection from the user name, password,
// resource, host URL and conf URL passed to it.
func NewClientWithServerInfo(user, pass, resource, authType, host, domain, conf string) (*Client, error) {
	connection, err := xmpp.Dial(host)

	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(pass)); err != nil {
		return nil, err
	}

	c := &Client{
		AuthType: authType,
		Username: user,
		Password: b.String(),
		Resource: resource,
		Id:       user + "@" + domain,

		// private
		connection:      connection,
		mentionNames:    make(map[string]string),
		receivedUsers:   make(chan []*User),
		receivedRooms:   make(chan []*Room),
		receivedMessage: make(chan *Message),
		host:            host,
		domain:          domain,
		conf:            conf,
	}

	if err != nil {
		return c, err
	}

	err = c.authenticate()
	if err != nil {
		return c, err
	}

	go c.listen()
	return c, nil
}

// Messages returns a read-only channel of Message structs. After joining a
// room, messages will be sent on the channel.
func (c *Client) Messages() <-chan *Message {
	return c.receivedMessage
}

// Rooms returns a channel of Room slices
func (c *Client) Rooms() <-chan []*Room {
	return c.receivedRooms
}

// Users returns a channel of User slices
func (c *Client) Users() <-chan []*User {
	return c.receivedUsers
}

// Status sends a string to HipChat to indicate whether the client is available
// to chat, away or idle.
func (c *Client) Status(s string) {
	c.connection.Presence(c.Id, s)
}

// Join accepts the room id and the name used to display the client in the
// room.
func (c *Client) Join(roomId, resource string) {
	c.connection.MUCPresence(roomId+"/"+resource, c.Id)
}

// Part accepts the room id to part.
func (c *Client) Part(roomId, name string) {
	c.connection.MUCPart(roomId + "/" + name)
}

// Say accepts a room id, the name of the client in the room, and the message
// body and sends the message to the HipChat room.
func (c *Client) Say(roomId, name, body string) {
	c.connection.MUCSend("groupchat", roomId, c.Id+"/"+name, body)
}

// PrivSay accepts a client id, the name of the client, and the message
// body and sends the private message to the HipChat
// user.
func (c *Client) PrivSay(user, name, body string) {
	c.connection.MUCSend("chat", user, c.Id+"/"+name, body)
}

// KeepAlive is meant to run as a goroutine. It sends a single whitespace
// character to HipChat every 60 seconds. This keeps the connection from
// idling after 150 seconds.
func (c *Client) KeepAlive() {
	for _ = range time.Tick(60 * time.Second) {
		c.connection.KeepAlive()
	}
}

// RequestRooms will send an outgoing request to get
// the room information for all rooms
func (c *Client) RequestRooms() {
	c.connection.Discover(c.Id, c.conf)
}

// RequestUsers will send an outgoing request to get
// the user information for all users
func (c *Client) RequestUsers() {
	c.connection.Roster(c.Id, c.domain)
}

func (c *Client) authenticate() error {
	var errStr string
	c.connection.Stream(c.Id, c.host)
	for {
		element, err := c.connection.Next()
		if err != nil {
			return err
		}

		switch element.Name.Local + element.Name.Space {
		case "stream" + xmpp.NsStream:
			features := c.connection.Features()
			if features.StartTLS != nil {
				c.connection.StartTLS()
			} else {
				for _, m := range features.Mechanisms {
					if m == "PLAIN"  && c.AuthType == "plain" {
						c.connection.Auth(c.Username, c.Password, c.Resource)
					} else if m == "X-HIPCHAT-OAUTH2" && c.AuthType == "oauth" {
						c.connection.Oauth(c.Password, c.Resource)
					}
				}
			}
		case "proceed" + xmpp.NsTLS:
			c.connection.UseTLS(c.host)
			c.connection.Stream(c.Id, c.host)
		case "iq" + xmpp.NsJabberClient:
			for _, attr := range element.Attr {
				if attr.Name.Local == "type" && attr.Value == "result" {
					return nil // authenticated
				}
			}
			return errors.New("could not authenticate")

		// oauth:
		case "failure" + xmpp.NsSASL:
			errStr = "Unable to authenticate:"
		case "invalid-authzid" + xmpp.NsSASL:
			errStr += " no identity provided"
		case "not-authorized" + xmpp.NsSASL:
			errStr += " token not authorized"
			return errors.New(errStr)
		case "success" + xmpp.NsSASL:
			return nil
		}
	}

	return errors.New("unexpectedly ended auth loop")
}

func (c *Client) listen() {
	for {
		element, err := c.connection.Next()
		if err != nil {
			return
		}

		switch element.Name.Local + element.Name.Space {
		case "iq" + xmpp.NsJabberClient: // rooms and rosters
			query := c.connection.Query()
			switch query.XMLName.Space {
			case xmpp.NsDisco:
				items := make([]*Room, len(query.Items))
				for i, item := range query.Items {
					items[i] = &Room{Id: item.Jid,
						LastActive:      item.LastActive,
						NumParticipants: item.NumParticipants,
						Name:            item.Name,
						Owner:           item.Owner,
						Privacy:         item.Privacy,
						RoomId:          item.RoomId,
						Topic:           item.Topic,
					}
				}
				c.receivedRooms <- items
			case xmpp.NsIqRoster:
				items := make([]*User, len(query.Items))
				for i, item := range query.Items {
					items[i] = &User{Email: item.Email,
						Id:          item.Jid,
						Name:        item.Name,
						MentionName: item.MentionName,
					}
				}
				c.receivedUsers <- items
			}
		case "message" + xmpp.NsJabberClient:
			attr := xmpp.ToMap(element.Attr)

			c.receivedMessage <- &Message{
				From: attr["from"],
				To:   attr["to"],
				Body: c.connection.Body(),
			}
		}
	}
	panic("unreachable")
}
