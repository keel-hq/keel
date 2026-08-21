package hipchat

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	h "github.com/daneharrigan/hipchat"

	"github.com/keel-hq/keel/approvals"
	b "github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/pkg/config"
	"github.com/keel-hq/keel/pkg/store/sql"

	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"

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
	mu             sync.Mutex
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
	i.mu.Lock()
	defer i.mu.Unlock()
	i.postedMessages = append(i.postedMessages, postedMessage{
		text:    body,
		channel: roomID,
	})
}

// snapshot returns a copy of the posted messages under lock, so tests can read
// them concurrently with the bot's async message handling goroutines.
func (i *fakeXmppImplementer) snapshot() []postedMessage {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := make([]postedMessage, len(i.postedMessages))
	copy(out, i.postedMessages)
	return out
}

// waitFor polls the condition until it holds or the deadline elapses, so async
// bot responses (posted via goroutines) are awaited deterministically instead
// of via fixed sleeps. On timeout it fails the test without panicking.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s waiting for condition", timeout)
		}
		time.Sleep(20 * time.Millisecond)
	}
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
	fakeBot := &Bot{hipchatClient: fi, name: "keel", approvalsChannel: "111111_approvals@conf.hipchat.com"}

	os.Setenv("HIPCHAT_APPROVALS_CHANNEL", "111111_approvals@conf.hipchat.com")
	os.Setenv("HIPCHAT_APPROVALS_BOT_NAME", "keel")
	os.Setenv("HIPCHAT_APPROVALS_USER_NAME", "111111_222222")
	os.Setenv("HIPCHAT_APPROVALS_PASSWORT", "pass")
	os.Setenv("HIPCHAT_CONNECTION_ATTEMPTS", "0")

	b.UnregisterBot("hipchat")
	b.RegisterBot("fakechat", fakeBot)
	b.Run(config.Config{Bots: config.BotConfig{Hipchat: config.HipchatBotConfig{ApprovalsChannel: "111111_approvals@conf.hipchat.com", ApprovalsBotName: "keel", ApprovalsUserName: "111111_222222", ApprovalsPassword: "pass", ConnectionAttempts: 0}}}, k8sImplementer, approvalsManager)
	return fakeBot
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func newTestingUtils() (*sql.SQLStore, func()) {
	dir, err := os.MkdirTemp("", "whstoretest")
	if err != nil {
		log.Fatal(err)
	}
	tmpfn := filepath.Join(dir, "gorm.db")
	// defer
	store, err := sql.New(sql.Opts{DatabaseType: "sqlite3", URI: tmpfn})
	if err != nil {
		log.Fatal(err)
	}

	teardown := func() {
		os.RemoveAll(dir) // clean up
	}

	return store, teardown
}

func TestHelpCommand(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeXmppImplementer{}
	fi.messages = make(chan *h.Message)

	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	NewBot(f8s, am, fi)
	defer b.Stop()

	waitFor(t, 5*time.Second, func() bool {
		msgs := fi.snapshot()
		return len(msgs) == 1 && strings.HasPrefix(msgs[0].text, "Keel bot was started")
	})
	msgs := fi.snapshot()
	if len(msgs) < 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if !strings.HasPrefix(msgs[0].text, "Keel bot was started") {
		t.Errorf("expected to find greeting message, but got: %s", msgs[0].text)
	}

	fi.messageFromChat("help")
	waitFor(t, 5*time.Second, func() bool { return len(fi.snapshot()) == 2 })

	msgs = fi.snapshot()
	if len(msgs) < 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if !strings.HasPrefix(msgs[1].text, "/code Here's a list of supported commands") {
		t.Errorf("expected to find help message, but got: %s", msgs[1].text)
	}
}

func TestBotAproval(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeXmppImplementer{}
	fi.messages = make(chan *h.Message)
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	NewBot(f8s, am, fi)
	defer b.Stop()

	// wait for the "Keel bot was started" greeting instead of a fixed sleep
	waitFor(t, 5*time.Second, func() bool {
		return len(fi.snapshot()) == 1 &&
			strings.HasPrefix(fi.snapshot()[0].text, "Keel bot was started")
	})

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

	// wait for the "Approval required!" message (goroutine-delivered)
	waitFor(t, 5*time.Second, func() bool {
		msgs := fi.snapshot()
		return len(msgs) == 2 && strings.HasPrefix(msgs[1].text, "/code Approval required!")
	})
	msgs := fi.snapshot()
	if !strings.HasPrefix(msgs[1].text, "/code Approval required!") {
		t.Errorf("expected to find help message, but got: %s", msgs[1].text)
	}

	// approve
	fi.messageFromChat("approve k8s/project/repo:1.2.3")

	// wait for the "Update approved!" message (goroutine-delivered)
	waitFor(t, 5*time.Second, func() bool {
		msgs := fi.snapshot()
		return len(msgs) == 3 && strings.HasPrefix(msgs[2].text, "/code Update approved!")
	})
	msgs = fi.snapshot()
	if !strings.HasPrefix(msgs[2].text, "/code Update approved!") {
		t.Errorf("expected to find message, but got: %s", msgs[2].text)
	}

	// get approvals
	fi.messageFromChat("get approvals")
	waitFor(t, 5*time.Second, func() bool { return len(fi.snapshot()) == 4 })

	msgs = fi.snapshot()
	if len(msgs) < 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	resp := trimSpaces(msgs[3].text)

	if !strings.Contains(resp, "k8s/project/repo:1.2.3 2.3.4 -> 3.4.5 1/1 false") {
		t.Errorf("expected to find message, but got: %s", resp)
	}
}

func TestBotReject(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeXmppImplementer{}
	fi.messages = make(chan *h.Message)
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	NewBot(f8s, am, fi)
	defer b.Stop()

	waitFor(t, 5*time.Second, func() bool {
		msgs := fi.snapshot()
		return len(msgs) == 1 && strings.HasPrefix(msgs[0].text, "Keel bot was started")
	})

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

	// wait for the "Approval required!" message (goroutine-delivered)
	waitFor(t, 5*time.Second, func() bool {
		msgs := fi.snapshot()
		return len(msgs) == 2 && strings.HasPrefix(msgs[1].text, "/code Approval required!")
	})
	msgs := fi.snapshot()
	if !strings.HasPrefix(msgs[1].text, "/code Approval required!") {
		t.Errorf("expected to find help message, but got: %s", msgs[1].text)
	}

	// reject
	fi.messageFromChat("reject k8s/project/repo:1.2.3")

	// wait for the "Change rejected" message (goroutine-delivered)
	waitFor(t, 5*time.Second, func() bool {
		msgs := fi.snapshot()
		return len(msgs) == 3 && strings.HasPrefix(msgs[2].text, "/code Change rejected")
	})
	msgs = fi.snapshot()
	if !strings.HasPrefix(msgs[2].text, "/code Change rejected") {
		t.Errorf("expected to find message, but got: %s", msgs[2].text)
	}

	// get approvals
	fi.messageFromChat("get approvals")
	waitFor(t, 5*time.Second, func() bool { return len(fi.snapshot()) == 4 })

	msgs = fi.snapshot()
	if len(msgs) < 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	resp := trimSpaces(msgs[3].text)

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
