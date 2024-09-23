package slack

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/slack-go/slack"

	"github.com/keel-hq/keel/extension/approval"
	"github.com/keel-hq/keel/pkg/store/sql"
	"github.com/keel-hq/keel/provider/kubernetes"

	"github.com/keel-hq/keel/approvals"
	b "github.com/keel-hq/keel/bot"

	// "github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/types"

	"testing"

	testutil "github.com/keel-hq/keel/util/testing"
)

var botMessagesChannel chan *b.BotMessage
var approvalsRespCh chan *b.ApprovalResponse

func New(name, token, channel string,
	k8sImplementer kubernetes.Implementer,
	approvalsManager approvals.Manager) *Bot {

	approvalsRespCh = make(chan *b.ApprovalResponse)
	botMessagesChannel = make(chan *b.BotMessage)

	slack := &Bot{}
	b.RegisterBot(name, slack)
	b.Run(k8sImplementer, approvalsManager)
	return slack
}

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

type postedMessage struct {
	channel string
	// text    string
	msg []slack.MsgOption
}

type fakeSlackImplementer struct {
	postedMessages []postedMessage
}

// func (i *fakeSlackImplementer) PostMessage(channel, text string, params slack.PostMessageParameters) (string, string, error) {
func (i *fakeSlackImplementer) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	i.postedMessages = append(i.postedMessages, postedMessage{
		channel: channelID,
		// text:    text,

		msg: options,
	})
	return "", "", nil
}

func newTestingUtils() (*sql.SQLStore, func()) {
	dir, err := ioutil.TempDir("", "whstoretest")
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

func TestBotRequest(t *testing.T) {

	os.Setenv(constants.EnvSlackBotToken, "")

	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}

	token := os.Getenv(constants.EnvSlackBotToken)
	if token == "" {
		t.Skip()
	}

	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	New("keel", token, "approvals", f8s, am)
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

	if len(fi.postedMessages) != 1 {
		t.Errorf("expected to find one message, but got: %d", len(fi.postedMessages))
	}
}

func TestProcessApprovedResponse(t *testing.T) {

	os.Setenv(constants.EnvSlackBotToken, "")

	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}

	token := os.Getenv(constants.EnvSlackBotToken)
	if token == "" {
		t.Skip()
	}

	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	New("keel", token, "approvals", f8s, am)
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

	if len(fi.postedMessages) != 1 {
		t.Errorf("expected to find one message")
	}
}

func TestProcessApprovalReply(t *testing.T) {

	os.Setenv(constants.EnvSlackBotToken, "")

	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}

	token := os.Getenv(constants.EnvSlackBotToken)
	if token == "" {
		t.Skip()
	}

	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	identifier := "k8s/project/repo:1.2.3"

	// creating initial approve request
	err := am.Create(&types.Approval{
		Identifier:     identifier,
		VotesRequired:  2,
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

	bot := New("keel", token, "approvals", f8s, am)
	defer b.Stop()

	time.Sleep(1 * time.Second)

	// approval resp
	bot.approvalsRespCh <- &b.ApprovalResponse{
		User:   "123",
		Status: types.ApprovalStatusApproved,
		Text:   fmt.Sprintf("%s %s", b.ApprovalResponseKeyword, identifier),
	}

	time.Sleep(1 * time.Second)

	updated, err := am.Get(identifier)
	if err != nil {
		t.Fatalf("failed to get approval, error: %s", err)
	}

	if updated.VotesReceived != 1 {
		t.Errorf("expected to find 1 received vote, found %d", updated.VotesReceived)
	}

	if updated.Status() != types.ApprovalStatusPending {
		t.Errorf("expected approval to be in status pending but got: %s", updated.Status())
	}

	if len(fi.postedMessages) != 1 {
		t.Errorf("expected to find one message, found: %d", len(fi.postedMessages))
	}

}

func TestProcessRejectedReply(t *testing.T) {

	os.Setenv(constants.EnvSlackBotToken, "")

	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}

	token := os.Getenv(constants.EnvSlackBotToken)
	if token == "" {
		t.Skip()
	}

	identifier := "k8s/project/repo:1.2.3"

	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})
	// creating initial approve request
	err := am.Create(&types.Approval{
		Identifier:     identifier,
		VotesRequired:  2,
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

	bot := New("keel", "random", "approvals", f8s, am)
	defer b.Stop()

	collector := approval.New()
	collector.Configure(am)

	time.Sleep(1 * time.Second)

	// approval resp
	bot.approvalsRespCh <- &b.ApprovalResponse{
		User:   "123",
		Status: types.ApprovalStatusRejected,
		Text:   fmt.Sprintf("%s %s", b.RejectResponseKeyword, identifier),
	}

	t.Logf("rejecting with: '%s'", fmt.Sprintf("%s %s", b.RejectResponseKeyword, identifier))

	time.Sleep(1 * time.Second)

	updated, err := am.Get(identifier)
	if err != nil {
		t.Fatalf("failed to get approval, error: %s", err)
	}

	if updated.VotesReceived != 0 {
		t.Errorf("expected to find 0 received vote, found %d", updated.VotesReceived)
	}

	if updated.Status() != types.ApprovalStatusRejected {
		t.Errorf("expected approval to be in status rejected but got: %s", updated.Status())
	}

	fmt.Println(updated.Status())

	if len(fi.postedMessages) != 1 {
		t.Errorf("expected to find one message, got: %d", len(fi.postedMessages))
	}

}

func TestIsApproval(t *testing.T) {

	event := &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "approvals",
			User:    "user-x",
		},
	}
	_, isApproval := b.IsApproval(event.User, "approve k8s/project/repo:1.2.3")

	if !isApproval {
		t.Errorf("event expected to be an approval")
	}
}
