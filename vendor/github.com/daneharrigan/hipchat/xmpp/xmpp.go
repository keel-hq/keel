package xmpp

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
)

const (
	NsJabberClient = "jabber:client"
	NsStream       = "http://etherx.jabber.org/streams"
	NsIqAuth       = "jabber:iq:auth"
	NsIqRoster     = "jabber:iq:roster"
	NsSASL         = "urn:ietf:params:xml:ns:xmpp-sasl"
	NsTLS          = "urn:ietf:params:xml:ns:xmpp-tls"
	NsDisco        = "http://jabber.org/protocol/disco#items"
	NsMuc          = "http://jabber.org/protocol/muc"

	xmlStream      = "<stream:stream from='%s' to='%s' version='1.0' xml:lang='en' xmlns='%s' xmlns:stream='%s'>"
	xmlStartTLS    = "<starttls xmlns='%s'/>"
	xmlIqSet       = "<iq type='set' id='%s'><query xmlns='%s'><username>%s</username><password>%s</password><resource>%s</resource></query></iq>"
	xmlIqGet       = "<iq from='%s' to='%s' id='%s' type='get'><query xmlns='%s'/></iq>"
	xmlOauth       = "<auth xmlns='http://hipchat.com' ver='22' mechanism='oauth2'>%s</auth>"
	xmlPresence    = "<presence from='%s'><show>%s</show></presence>"
	xmlMUCPart     = "<presence to='%s' type='unavailable'></presence>"
	xmlMUCPresence = "<presence id='%s' to='%s' from='%s'><x xmlns='%s'/></presence>"
	xmlMUCMessage  = "<message from='%s' id='%s' to='%s' type='%s'><body>%s</body></message>"
)

type required struct{}

type features struct {
	Auth       xml.Name  `xml:"auth"`
	XMLName    xml.Name  `xml:"features"`
	StartTLS   *required `xml:"starttls>required"`
	Mechanisms []string  `xml:"mechanisms>mechanism"`
}

type item struct {
	Email           string `xml:"email,attr"`
	Jid             string `xml:"jid,attr"`
	LastActive      string `xml:"x>last_active"`
	MentionName     string `xml:"mention_name,attr"`
	Name            string `xml:"name,attr"`
	NumParticipants string `xml:"x>num_participants"`
	Owner           string `xml:"x>owner"`
	Privacy         string `xml:"x>privacy"`
	RoomId          string `xml:"x>id"`
	Topic           string `xml:"x>topic"`
}

type query struct {
	XMLName xml.Name `xml:"query"`
	Items   []*item  `xml:"item"`
}

type body struct {
	Body string `xml:",innerxml"`
}

type Conn struct {
	incoming *xml.Decoder
	outgoing net.Conn
}

type Message struct {
	Jid         string
	MentionName string
	Body        string
}

func (c *Conn) Stream(jid, host string) {
	fmt.Fprintf(c.outgoing, xmlStream, jid, host, NsJabberClient, NsStream)
}

func (c *Conn) StartTLS() {
	fmt.Fprintf(c.outgoing, xmlStartTLS, NsTLS)
}

func (c *Conn) UseTLS(host string) {
	c.outgoing = tls.Client(c.outgoing, &tls.Config{ServerName: host})
	c.incoming = xml.NewDecoder(c.outgoing)
}

func (c *Conn) Auth(user, pass, resource string) {
	fmt.Fprintf(c.outgoing, xmlIqSet, id(), NsIqAuth, user, pass, resource)
}

func (c *Conn) Oauth(token, resource string) {
	msg := "\x00" + token + "\x00" + resource
	b64 := base64.StdEncoding.EncodeToString([]byte(msg))
	fmt.Fprintf(c.outgoing, xmlOauth, b64)
}

func (c *Conn) Features() *features {
	var f features
	c.incoming.DecodeElement(&f, nil)
	return &f
}

func (c *Conn) Next() (xml.StartElement, error) {
	for {
	var element xml.StartElement

		var err error
		var t xml.Token
		t, err = c.incoming.Token()
		if err != nil {
			return element, err
		}

		switch t := t.(type) {
		case xml.StartElement:
			element = t
			if element.Name.Local == "" {
				return element, errors.New("invalid xml response")
			}

			return element, nil
		}
	}
	panic("unreachable")
}

func (c *Conn) Discover(from, to string) {
	fmt.Fprintf(c.outgoing, xmlIqGet, from, to, id(), NsDisco)
}

func (c *Conn) Body() string {
	b := new(body)
	c.incoming.DecodeElement(b, nil)
	return b.Body
}

func (c *Conn) Query() *query {
	q := new(query)
	c.incoming.DecodeElement(q, nil)
	return q
}

func (c *Conn) Presence(jid, pres string) {
	fmt.Fprintf(c.outgoing, xmlPresence, jid, pres)
}

func (c *Conn) MUCPart(roomId string) {
	fmt.Fprintf(c.outgoing, xmlMUCPart, roomId)
}

func (c *Conn) MUCPresence(roomId, jid string) {
	fmt.Fprintf(c.outgoing, xmlMUCPresence, id(), roomId, jid, NsMuc)
}

func (c *Conn) MUCSend(mtype, to, from, body string) {
	fmt.Fprintf(c.outgoing, xmlMUCMessage, from, id(), to, mtype, html.EscapeString(body))
}

func (c *Conn) Roster(from, to string) {
	fmt.Fprintf(c.outgoing, xmlIqGet, from, to, id(), NsIqRoster)
}

func (c *Conn) KeepAlive() {
	fmt.Fprintf(c.outgoing, " ")
}

func Dial(host string) (*Conn, error) {
	c := new(Conn)
	outgoing, err := net.Dial("tcp", host+":5222")

	if err != nil {
		return c, err
	}

	c.outgoing = outgoing
	c.incoming = xml.NewDecoder(outgoing)

	return c, nil
}

func ToMap(attr []xml.Attr) map[string]string {
	m := make(map[string]string)
	for _, a := range attr {
		m[a.Name.Local] = a.Value
	}

	return m
}

func id() string {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)
}
