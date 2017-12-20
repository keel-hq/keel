package hipchat

import (
	"os"
	"testing"
	"time"

	"github.com/glower/keel/approvals"
	"github.com/glower/keel/cache/memory"
	"github.com/glower/keel/types"
	"github.com/glower/keel/util/codecs"
	testutil "github.com/keel-hq/keel/util/testing"
)

type fakeSlackImplementer struct {
	postedMessages []postedMessage
}

type fakeProvider struct {
	submitted []types.Event
	images    []*types.TrackedImage
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
	// params  slack.PostMessageParameters
}

func TestBot(t *testing.T) {
	k8sImplementer := &testutil.FakeK8sImplementer{}

	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)

	os.Setenv("HIPCHAT_APPROVALS_CHANNEL", "701032_keel-bot@conf.hipchat.com")
	os.Setenv("HIPCHAT_APPROVALS_BOT_NAME", "Igor Komlew")
	os.Setenv("HIPCHAT_APPROVALS_USER_NAME", "701032_4966430")
	os.Setenv("HIPCHAT_APPROVALS_PASSWORT", "B10nadeL!tschi22")

	approvalsManager := approvals.New(mem, codecs.DefaultSerializer())

	Run(k8sImplementer, approvalsManager)
	select {}
}
