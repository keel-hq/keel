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
