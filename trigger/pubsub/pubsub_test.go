package pubsub

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/provider"
)

type fakeClient struct {
}

func fakeDoneFunc(id string, done bool) {
	return
}

// TestWithKeepAliveDialer verifies that the dial option used by
// NewPubsubSubscriber still establishes the TCP transport Pub/Sub requires.
func TestWithKeepAliveDialer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	connected := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		close(connected)
		defer conn.Close()
		<-time.After(time.Second)
	}()

	client, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		WithKeepAliveDialer(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	client.Connect()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !client.WaitForStateChange(ctx, connectivity.Idle) {
		t.Fatalf("dialer did not establish a connection: %v", ctx.Err())
	}

	select {
	case <-connected:
	case <-ctx.Done():
		t.Fatalf("dialer did not reach the test listener: %v", ctx.Err())
	}
}

func TestCallback(t *testing.T) {

	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})
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
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})
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
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})
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
