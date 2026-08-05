package notification

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/keel-hq/keel/types"
)

type fakeSender struct {
	sent chan types.EventNotification

	shouldConfigure bool
	shouldError     error
	started         chan struct{}
	block           <-chan struct{}
}

func (s *fakeSender) Configure(*Config) (bool, error) {
	return s.shouldConfigure, nil
}

func (s *fakeSender) Send(event types.EventNotification) error {
	if s.started != nil {
		select {
		case s.started <- struct{}{}:
		default:
		}
	}
	if s.block != nil {
		<-s.block
	}
	s.sent <- event
	return s.shouldError
}

func newFakeSender(sendError error) *fakeSender {
	return &fakeSender{
		sent:            make(chan types.EventNotification, 16),
		shouldConfigure: true,
		shouldError:     sendError,
	}
}

func waitForNotification(t *testing.T, sender *fakeSender) types.EventNotification {
	t.Helper()
	select {
	case event := <-sender.sent:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
		return types.EventNotification{}
	}
}

func TestSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sndr := New(ctx)

	sndr.Configure(&Config{
		Level:    types.LevelDebug,
		Attempts: 1,
	})

	fs := newFakeSender(nil)

	RegisterSender("fakeSender", fs)
	defer sndr.UnregisterSender("fakeSender")

	err := sndr.Send(types.EventNotification{
		Level:   types.LevelInfo,
		Type:    types.NotificationPreDeploymentUpdate,
		Message: "foo",
	})

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	sent := waitForNotification(t, fs)

	if sent.Message != "foo" {
		t.Errorf("unexpected notification message: %s", sent.Message)
	}

	if sent.Level != types.LevelInfo {
		t.Errorf("unexpected level: %s", sent.Level)
	}
}

// test when configured level is higher than the event
func TestSendLevelNotificationA(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sndr := New(ctx)

	sndr.Configure(&Config{
		Level:    types.LevelInfo,
		Attempts: 1,
	})

	fs := newFakeSender(nil)

	RegisterSender("fakeSender", fs)
	defer sndr.UnregisterSender("fakeSender")

	err := sndr.Send(types.EventNotification{
		Level:   types.LevelDebug,
		Type:    types.NotificationPreDeploymentUpdate,
		Message: "foo",
	})

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	select {
	case <-fs.sent:
		t.Errorf("didn't expect to find sent even for this level")
	default:
	}
}

// event level is higher than the configured
func TestSendLevelNotificationB(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sndr := New(ctx)

	sndr.Configure(&Config{
		Level:    types.LevelInfo,
		Attempts: 1,
	})

	fs := newFakeSender(nil)

	RegisterSender("fakeSender", fs)
	defer sndr.UnregisterSender("fakeSender")

	err := sndr.Send(types.EventNotification{
		Level:   types.LevelSuccess,
		Type:    types.NotificationPreDeploymentUpdate,
		Message: "foo",
	})

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	sent := waitForNotification(t, fs)

	if sent.Message != "foo" {
		t.Errorf("unexpected notification message: %s", sent.Message)
	}

	if sent.Level != types.LevelSuccess {
		t.Errorf("unexpected level: %s", sent.Level)
	}
}

func TestSendLevelNotificationC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sndr := New(ctx)

	sndr.Configure(&Config{
		Level:    types.LevelDebug,
		Attempts: 1,
	})

	fs := newFakeSender(nil)

	RegisterSender("fakeSender", fs)
	defer sndr.UnregisterSender("fakeSender")

	err := sndr.Send(types.EventNotification{
		Level:   types.LevelDebug,
		Type:    types.NotificationPreDeploymentUpdate,
		Message: "foo",
	})

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	sent := waitForNotification(t, fs)

	if sent.Message != "foo" {
		t.Errorf("unexpected notification message: %s", sent.Message)
	}

	if sent.Level != types.LevelDebug {
		t.Errorf("unexpected level: %s", sent.Level)
	}
}

func TestSendDoesNotBlockOnBrokenSender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sndr := New(ctx)
	sndr.Configure(&Config{
		Level:    types.LevelDebug,
		Attempts: 10,
	})

	blocked := make(chan struct{})
	defer close(blocked)
	brokenSender := newFakeSender(fmt.Errorf("webhook endpoint unreachable"))
	brokenSender.started = make(chan struct{}, 1)
	brokenSender.block = blocked
	healthySender := newFakeSender(nil)

	RegisterSender("brokenSender", brokenSender)
	t.Cleanup(func() { sndr.UnregisterSender("brokenSender") })
	RegisterSender("healthySender", healthySender)
	t.Cleanup(func() { sndr.UnregisterSender("healthySender") })

	done := make(chan error, 1)
	go func() {
		done <- sndr.Send(types.EventNotification{
			Level:   types.LevelInfo,
			Type:    types.NotificationDeploymentUpdate,
			Message: "update successful",
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error from Send(), got: %s", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Send() blocked on a notification sender")
	}

	select {
	case <-brokenSender.started:
	case <-time.After(time.Second):
		t.Fatal("broken sender was not attempted")
	}

	sent := waitForNotification(t, healthySender)
	if sent.Message != "update successful" {
		t.Errorf("unexpected message on healthy sender: %s", sent.Message)
	}

}
