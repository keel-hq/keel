package pubsub

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/pkg/store/sql"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	"testing"
)

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
			{
				Image:    img,
				Provider: "fp",
			},
		},
	}

	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)

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
