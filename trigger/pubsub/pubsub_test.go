package pubsub

import (
	"encoding/json"
	"time"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/util/codecs"

	"testing"
)

type fakeClient struct {
}

func fakeDoneFunc(id string, done bool) {
	return
}

func TestCallback(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	sub := &PubsubSubscriber{disableAck: true, providers: providers}

	dataMsg := &Message{Action: "INSERT", Tag: "gcr.io/v2-namespace/hello-world:1.1.1"}
	data, _ := json.Marshal(dataMsg)

	msg := &pubsub.Message{Data: data}

	sub.callback(context.Background(), msg)

	if len(fp.submitted) == 0 {
		t.Fatalf("no events found in provider")
	}
	if fp.submitted[0].Repository.Name != "gcr.io/v2-namespace/hello-world" {
		t.Errorf("expected repo name %s but got %s", "gcr.io/v2-namespace/hello-world", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "1.1.1" {
		t.Errorf("expected repo tag %s but got %s", "1.1.1", fp.submitted[0].Repository.Tag)
	}

}
func TestCallbackTagNotSemver(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	sub := &PubsubSubscriber{disableAck: true, providers: providers}

	dataMsg := &Message{Action: "INSERT", Tag: "gcr.io/stemnapp/alpine-website:latest"}
	data, _ := json.Marshal(dataMsg)

	msg := &pubsub.Message{Data: data}

	sub.callback(context.Background(), msg)

	if len(fp.submitted) == 0 {
		t.Fatalf("no events found in provider")
	}
	if fp.submitted[0].Repository.Name != "gcr.io/stemnapp/alpine-website" {
		t.Errorf("expected repo name %s but got %s", "gcr.io/v2-namespace/hello-world", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "latest" {
		t.Errorf("expected repo tag %s but got %s", "latest", fp.submitted[0].Repository.Tag)
	}

}

func TestCallbackNoTag(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	sub := &PubsubSubscriber{disableAck: true, providers: providers}

	dataMsg := &Message{Action: "INSERT", Tag: "gcr.io/stemnapp/alpine-website"}
	data, _ := json.Marshal(dataMsg)

	msg := &pubsub.Message{Data: data}

	sub.callback(context.Background(), msg)

	if len(fp.submitted) == 0 {
		t.Fatalf("no events found in provider")
	}
	if fp.submitted[0].Repository.Name != "gcr.io/stemnapp/alpine-website" {
		t.Errorf("expected repo name %s but got %s", "gcr.io/v2-namespace/hello-world", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "latest" {
		t.Errorf("expected repo tag %s but got %s", "latest", fp.submitted[0].Repository.Tag)
	}
}
