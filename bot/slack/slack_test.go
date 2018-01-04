package slack

import (
	"fmt"
	"os"
	"time"

	"github.com/nlopes/slack"

	"github.com/keel-hq/keel/extension/approval"
	"github.com/keel-hq/keel/provider/kubernetes"

	"github.com/keel-hq/keel/approvals"
	b "github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"

	"testing"

	testutil "github.com/keel-hq/keel/util/testing"
)

var botMessagesChannel chan *b.BotMessage
var approvalsRespCh chan *b.ApprovalResponse

func New(name, token, channel string,
	k8sImplementer kubernetes.Implementer,
	approvalsManager approvals.Manager, fi SlackImplementer) *Bot {

	approvalsRespCh = make(chan *b.ApprovalResponse)
	botMessagesChannel = make(chan *b.BotMessage)

	slack := &Bot{}
	b.RegisterBot(name, slack)
	b.Run(k8sImplementer, approvalsManager)
	slack.slackHTTPClient = fi
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
	text    string
	params  slack.PostMessageParameters
}

type fakeSlackImplementer struct {
	postedMessages []postedMessage
}

func (i *fakeSlackImplementer) PostMessage(channel, text string, params slack.PostMessageParameters) (string, string, error) {
	i.postedMessages = append(i.postedMessages, postedMessage{
		channel: channel,
		text:    text,
		params:  params,
	})
	return "", "", nil
}

func TestBotRequest(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)

	token := os.Getenv(constants.EnvSlackToken)
	if token == "" {
		t.Skip()
	}

	am := approvals.New(mem, codecs.DefaultSerializer())

	New("keel", token, "approvals", f8s, am, fi)
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
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)

	token := os.Getenv(constants.EnvSlackToken)
	if token == "" {
		t.Skip()
	}

	am := approvals.New(mem, codecs.DefaultSerializer())

	New("keel", token, "approvals", f8s, am, fi)
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
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)

	token := os.Getenv(constants.EnvSlackToken)
	if token == "" {
		t.Skip()
	}

	am := approvals.New(mem, codecs.DefaultSerializer())

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

	bot := New("keel", token, "approvals", f8s, am, fi)
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
		t.Errorf("expected to find one message")
	}

}

func TestProcessRejectedReply(t *testing.T) {
	f8s := &testutil.FakeK8sImplementer{}
	fi := &fakeSlackImplementer{}
	mem := memory.NewMemoryCache(100*time.Hour, 100*time.Hour, 100*time.Hour)

	token := os.Getenv(constants.EnvSlackToken)
	if token == "" {
		t.Skip()
	}

	identifier := "k8s/project/repo:1.2.3"

	am := approvals.New(mem, codecs.DefaultSerializer())
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

	bot := New("keel", "random", "approvals", f8s, am, fi)
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
		t.Errorf("expected to find one message")
	}

}

func TestIsApproval(t *testing.T) {
	// f8s := &testutil.FakeK8sImplementer{}
	// mem := memory.NewMemoryCache(100*time.Hour, 100*time.Hour, 100*time.Hour)
	//
	// identifier := "k8s/project/repo:1.2.3"
	//
	// am := approvals.New(mem, codecs.DefaultSerializer())
	// // creating initial approve request
	// err := am.Create(&types.Approval{
	// 	Identifier:     identifier,
	// 	VotesRequired:  2,
	// 	CurrentVersion: "2.3.4",
	// 	NewVersion:     "3.4.5",
	// 	Event: &types.Event{
	// 		Repository: types.Repository{
	// 			Name: "project/repo",
	// 			Tag:  "2.3.4",
	// 		},
	// 	},
	// })
	//
	// if err != nil {
	// 	t.Fatalf("unexpected error while creating : %s", err)
	// }
	// bot := New("keel", "random", "approvals", f8s, am, fi)
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
