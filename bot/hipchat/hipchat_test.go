package hipchat

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	h "github.com/daneharrigan/hipchat"

	"github.com/keel-hq/keel/approvals"
	b "github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"

	testutil "github.com/keel-hq/keel/util/testing"

	log "github.com/sirupsen/logrus"
)

type fakeProvider struct {
	submitted []types.Event
	images    []*types.TrackedImage
}

func (p *fakeProvider) Submit(event types.Event) error {
	p.submitted = append(p.submitted, event)
	return nil
}

func (p *fakeProvider) TrackedImages() ([]*types.TrackedImage, error) {
	return p.images, nil
}

func (p *fakeProvider) List() []string {
	return []string{"fakeprovider"}
}
func (p *fakeProvider) Stop() {
	return
}
func (p *fakeProvider) GetName() string {
	return "fp"
}

var botMessagesChannel chan *b.BotMessage
var approvalsRespCh chan *b.ApprovalResponse

type postedMessage struct {
	channel string
	text    string
}

type fakeXmppImplementer struct {
	postedMessages []postedMessage
	messages       chan *h.Message
}

func (i *fakeXmppImplementer) messageFromChat(message string) {
	i.messages <- &h.Message{
		Body: "@keel " + message,
		From: "111111_approvals@conf.hipchat.com/test",
		To:   "222222_333333@chat.hipchat.com/bot",
	}
}

func (i *fakeXmppImplementer) Say(roomID, name, body string) {
	i.postedMessages = append(i.postedMessages, postedMessage{
		text:    body,
		channel: roomID,
	})
}
func (i *fakeXmppImplementer) Status(s string) {
}
func (i *fakeXmppImplementer) Join(roomID, resource string) {
}
func (i *fakeXmppImplementer) KeepAlive() {
}
func (i *fakeXmppImplementer) Messages() <-chan *h.Message {
	return i.messages
}

func NewBot(k8sImplementer kubernetes.Implementer,
	approvalsManager approvals.Manager, fi XmppImplementer) *Bot {

	approvalsRespCh = make(chan *b.ApprovalResponse)
	botMessagesChannel = make(chan *b.BotMessage)
	fakeBot := &Bot{}
	fakeBot.hipchatClient = fi

	os.Setenv("HIPCHAT_APPROVALS_CHANNEL", "111111_approvals@conf.hipchat.com")
	os.Setenv("HIPCHAT_APPROVALS_BOT_NAME", "keel")
	os.Setenv("HIPCHAT_APPROVALS_USER_NAME", "111111_222222")
	os.Setenv("HIPCHAT_APPROVALS_PASSWORT", "pass")
	os.Setenv("HIPCHAT_CONNECTION_ATTEMPTS", "0")

	b.RegisterBot("fakechat", fakeBot)
	b.Run(k8sImplementer, approvalsManager)
	return fakeBot
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestHelpCommand(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeXmppImplementer{}
	fi.messages = make(chan *h.Message)
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())

	NewBot(f8s, am, fi)
	defer b.Stop()

	time.Sleep(1 * time.Second)

	if len(fi.postedMessages) != 1 {
		t.Errorf("expected to find 1 message, but got: %d", len(fi.postedMessages))
	}
	if !strings.HasPrefix(fi.postedMessages[0].text, "Keel bot was started") {
		t.Errorf("expected to find greeting message, but got: %s", fi.postedMessages[0].text)
	}

	fi.messageFromChat("help")
	time.Sleep(1 * time.Second)

	if len(fi.postedMessages) != 2 {
		t.Errorf("expected to find 2 messages, but got: %d", len(fi.postedMessages))
	}
	if !strings.HasPrefix(fi.postedMessages[1].text, "/code Here's a list of supported commands") {
		t.Errorf("expected to find help message, but got: %s", fi.postedMessages[1].text)
	}
}

func TestBotAproval(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeXmppImplementer{}
	fi.messages = make(chan *h.Message)
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())

	NewBot(f8s, am, fi)
	defer b.Stop()

	time.Sleep(1 * time.Second)

	err := am.Create(&types.Approval{
		Identifier:     "k8s/project/repo:1.2.3",
		VotesRequired:  1,
		CurrentVersion: "2.3.4",
		NewVersion:     "3.4.5",
		Event: &types.Event{
			Repository: types.Repository{
				Name: "project/repo",
				Tag:  "2.3.4",
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error while creating : %s", err)
	}

	time.Sleep(1 * time.Second)

	if len(fi.postedMessages) != 2 {
		t.Errorf("expected to find 2 message, but got: %d", len(fi.postedMessages))
	}
	if !strings.HasPrefix(fi.postedMessages[1].text, "/code Approval required!") {
		t.Errorf("expected to find help message, but got: %s", fi.postedMessages[1].text)
	}

	// approve
	fi.messageFromChat("approve k8s/project/repo:1.2.3")
	time.Sleep(1 * time.Second)

	if len(fi.postedMessages) != 3 {
		t.Errorf("expected to find 3 message, but got: %d", len(fi.postedMessages))
	}
	if !strings.HasPrefix(fi.postedMessages[2].text, "/code Update approved!") {
		t.Errorf("expected to find message, but got: %s", fi.postedMessages[2].text)
	}

	// get approvals
	fi.messageFromChat("get approvals")
	time.Sleep(1 * time.Second)
	if len(fi.postedMessages) != 4 {
		t.Errorf("expected to find 4 message, but got: %d", len(fi.postedMessages))
	}
	resp := trimSpaces(fi.postedMessages[3].text)

	if !strings.Contains(resp, "k8s/project/repo:1.2.3 2.3.4 -> 3.4.5 1/1 false") {
		t.Errorf("expected to find message, but got: %s", resp)
	}
}

func TestBotReject(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeXmppImplementer{}
	fi.messages = make(chan *h.Message)
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())

	NewBot(f8s, am, fi)
	defer b.Stop()

	time.Sleep(1 * time.Second)

	err := am.Create(&types.Approval{
		Identifier:     "k8s/project/repo:1.2.3",
		VotesRequired:  1,
		CurrentVersion: "2.3.4",
		NewVersion:     "3.4.5",
		Event: &types.Event{
			Repository: types.Repository{
				Name: "project/repo",
				Tag:  "2.3.4",
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error while creating : %s", err)
	}

	time.Sleep(1 * time.Second)

	if len(fi.postedMessages) != 2 {
		t.Errorf("expected to find 2 message, but got: %d", len(fi.postedMessages))
	}
	if !strings.HasPrefix(fi.postedMessages[1].text, "/code Approval required!") {
		t.Errorf("expected to find help message, but got: %s", fi.postedMessages[1].text)
	}

	// reject
	fi.messageFromChat("reject k8s/project/repo:1.2.3")
	time.Sleep(1 * time.Second)

	if len(fi.postedMessages) != 3 {
		t.Errorf("expected to find 3 message, but got: %d", len(fi.postedMessages))
	}
	if !strings.HasPrefix(fi.postedMessages[2].text, "/code Change rejected") {
		t.Errorf("expected to find message, but got: %s", fi.postedMessages[2].text)
	}

	// get approvals
	fi.messageFromChat("get approvals")
	time.Sleep(1 * time.Second)
	if len(fi.postedMessages) != 4 {
		t.Errorf("expected to find 4 message, but got: %d", len(fi.postedMessages))
	}
	resp := trimSpaces(fi.postedMessages[3].text)

	if !strings.Contains(resp, "k8s/project/repo:1.2.3 2.3.4 -> 3.4.5 0/1 true") {
		t.Errorf("expected to find message, but got: %s", resp)
	}
}

func trimSpaces(input string) string {
	reLeadcloseWhtsp := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
	reInsideWhtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	final := reLeadcloseWhtsp.ReplaceAllString(input, "")
	final = reInsideWhtsp.ReplaceAllString(final, " ")
	return final
}
