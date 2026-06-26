package notification

import (
	"context"
	"fmt"
	"testing"

	"github.com/keel-hq/keel/types"
)

type fakeSender struct {
	sent *types.EventNotification

	shouldConfigure bool
	shouldError     error
}

func (s *fakeSender) Configure(*Config) (bool, error) {
	return s.shouldConfigure, nil
}

func (s *fakeSender) Send(event types.EventNotification) error {
	s.sent = &event
	fmt.Println("sending event")
	return s.shouldError
}

func TestSend(t *testing.T) {
	sndr := New(context.Background())

	sndr.Configure(&Config{
		Level:    types.LevelDebug,
		Attempts: 1,
	})

	fs := &fakeSender{
		shouldConfigure: true,
		shouldError:     nil,
	}

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

	if fs.sent.Message != "foo" {
		t.Errorf("unexpected notification message: %s", fs.sent.Message)
	}

	if fs.sent.Level != types.LevelInfo {
		t.Errorf("unexpected level: %s", fs.sent.Level)
	}
}

// test when configured level is higher than the event
func TestSendLevelNotificationA(t *testing.T) {
	sndr := New(context.Background())

	sndr.Configure(&Config{
		Level:    types.LevelInfo,
		Attempts: 1,
	})

	fs := &fakeSender{
		shouldConfigure: true,
		shouldError:     nil,
	}

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

	if fs.sent != nil {
		t.Errorf("didn't expect to find sent even for this level")
	}
}

// event level is higher than the configured
func TestSendLevelNotificationB(t *testing.T) {
	sndr := New(context.Background())

	sndr.Configure(&Config{
		Level:    types.LevelInfo,
		Attempts: 1,
	})

	fs := &fakeSender{
		shouldConfigure: true,
		shouldError:     nil,
	}

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

	if fs.sent.Message != "foo" {
		t.Errorf("unexpected notification message: %s", fs.sent.Message)
	}

	if fs.sent.Level != types.LevelSuccess {
		t.Errorf("unexpected level: %s", fs.sent.Level)
	}
}

func TestSendLevelNotificationC(t *testing.T) {
	sndr := New(context.Background())

	sndr.Configure(&Config{
		Level:    types.LevelDebug,
		Attempts: 1,
	})

	fs := &fakeSender{
		shouldConfigure: true,
		shouldError:     nil,
	}

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

	if fs.sent.Message != "foo" {
		t.Errorf("unexpected notification message: %s", fs.sent.Message)
	}

	if fs.sent.Level != types.LevelDebug {
		t.Errorf("unexpected level: %s", fs.sent.Level)
	}
}

// TestSendFailedSenderDoesNotReturnError verifies that when a sender
// fails all attempts, Send() logs the failure but does NOT return an error.
// This is critical: a broken webhook must never block the update pipeline.
func TestSendFailedSenderDoesNotReturnError(t *testing.T) {
	sndr := New(context.Background())

	sndr.Configure(&Config{
		Level:    types.LevelDebug,
		Attempts: 1,
	})

	fs := &fakeSender{
		shouldConfigure: true,
		shouldError:     fmt.Errorf("webhook endpoint unreachable"),
	}

	RegisterSender("failingSender", fs)
	defer sndr.UnregisterSender("failingSender")

	err := sndr.Send(types.EventNotification{
		Level:   types.LevelInfo,
		Type:    types.NotificationDeploymentUpdate,
		Message: "update successful",
	})

	if err != nil {
		t.Errorf("expected no error from Send() when sender fails, got: %s", err)
	}
}

// TestSendFailingSenderDoesNotBlockOtherSenders verifies that when one
// sender fails, the remaining senders still receive the notification.
func TestSendFailingSenderDoesNotBlockOtherSenders(t *testing.T) {
	sndr := New(context.Background())

	sndr.Configure(&Config{
		Level:    types.LevelDebug,
		Attempts: 1,
	})

	brokenSender := &fakeSender{
		shouldConfigure: true,
		shouldError:     fmt.Errorf("webhook endpoint unreachable"),
	}

	healthySender := &fakeSender{
		shouldConfigure: true,
		shouldError:     nil,
	}

	RegisterSender("brokenSender", brokenSender)
	defer sndr.UnregisterSender("brokenSender")
	RegisterSender("healthySender", healthySender)
	defer sndr.UnregisterSender("healthySender")

	err := sndr.Send(types.EventNotification{
		Level:   types.LevelInfo,
		Type:    types.NotificationDeploymentUpdate,
		Message: "update successful",
	})

	if err != nil {
		t.Errorf("expected no error from Send(), got: %s", err)
	}

	if healthySender.sent == nil {
		t.Error("expected healthy sender to receive the notification, but it did not")
	}

	if healthySender.sent != nil && healthySender.sent.Message != "update successful" {
		t.Errorf("unexpected message on healthy sender: %s", healthySender.sent.Message)
	}
}
