package pubsub

import (
	"golang.org/x/net/context"
	"sync"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/types"

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

func TestCheckDeployment(t *testing.T) {
	fs := &fakeSubscriber{}
	mng := &DefaultManager{
		client:      fs,
		mu:          &sync.Mutex{},
		ctx:         context.Background(),
		subscribers: make(map[string]context.Context),
	}

	dep := &v1beta1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:      "dep-1",
			Namespace: "xxxx",
			Labels:    map[string]string{types.KeelPolicyLabel: "all"},
		},
		v1beta1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Image: "gcr.io/v2-namespace/hello-world:1.1.1",
						},
						v1.Container{
							Image: "gcr.io/v2-namespace/greetings-world:1.1.1",
						},
					},
				},
			},
		},
		v1beta1.DeploymentStatus{},
	}

	err := mng.checkDeployment(dep)
	if err != nil {
		t.Errorf("deployment check failed: %s", err)
	}

	// sleeping a bit since our fake subscriber goes into a separate goroutine
	time.Sleep(100 * time.Millisecond)

	if fs.TimesSubscribed != 1 {
		t.Errorf("expected to find one subscription, found: %d", fs.TimesSubscribed)
	}

}
