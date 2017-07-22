package pubsub

import (
	"golang.org/x/net/context"
	"sync"
	"time"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"

	"testing"
)

type fakeSubscriber struct {
	TimesSubscribed     int
	SubscribedTopicName string
	SubscribedSubName   string
}

func (s *fakeSubscriber) Subscribe(ctx context.Context, topic, subscription string) error {
	s.TimesSubscribed++
	s.SubscribedTopicName = topic
	s.SubscribedSubName = subscription
	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}
}

type fakeProvider struct {
	images    []*types.TrackedImage
	submitted []types.Event
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

func TestCheckDeployment(t *testing.T) {
	img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:    img,
				Provider: "fp",
			},
		},
	}
	providers := provider.New([]provider.Provider{fp})

	fs := &fakeSubscriber{}
	mng := &DefaultManager{
		providers:   providers,
		client:      fs,
		mu:          &sync.Mutex{},
		ctx:         context.Background(),
		subscribers: make(map[string]context.Context),
	}

	err := mng.scan(context.Background())
	if err != nil {
		t.Errorf("failed to scan: %s", err)
	}

	// sleeping a bit since our fake subscriber goes into a separate goroutine
	time.Sleep(100 * time.Millisecond)

	if fs.TimesSubscribed != 1 {
		t.Errorf("expected to find one subscription, found: %d", fs.TimesSubscribed)
	}

}
