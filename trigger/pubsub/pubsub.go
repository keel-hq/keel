package pubsub

import (
	"encoding/json"
	"fmt"
	"time"

	"net"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	log "github.com/sirupsen/logrus"
)

// PubsubSubscriber is Google Cloud pubsub based subscriber
type PubsubSubscriber struct {
	providers provider.Providers

	project    string
	disableAck bool

	client *pubsub.Client
}

// pubsubImplementer - pubsub implementer
type pubsubImplementer interface {
	Subscription(id string) *pubsub.Subscription
	Receive(ctx context.Context, f func(context.Context, *Message)) error
}

// Opts - subscriber options
type Opts struct {
	ProjectID string
	Providers provider.Providers
}

// WithKeepAliveDialer - required so connections aren't dropped
// https://github.com/GoogleCloudPlatform/google-cloud-go/issues/500
func WithKeepAliveDialer() grpc.DialOption {
	return grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
		d := net.Dialer{Timeout: timeout, KeepAlive: time.Duration(10 * time.Second)}
		return d.Dial("tcp", addr)
	})
}

// NewPubsubSubscriber - create new pubsub subscriber
func NewPubsubSubscriber(opts *Opts) (*PubsubSubscriber, error) {
	clientOption := option.WithGRPCDialOption(WithKeepAliveDialer())
	client, err := pubsub.NewClient(context.Background(), opts.ProjectID, clientOption)
	if err != nil {
		return nil, err
	}

	return &PubsubSubscriber{
		project:   opts.ProjectID,
		providers: opts.Providers,
		client:    client,
	}, nil
}

// Message - expected message from gcr
type Message struct {
	Action string `json:"action,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

func (s *PubsubSubscriber) ensureTopic(ctx context.Context, id string) error {
	topic := s.client.Topic(id)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check whether topic exists, error: %s", err)
	}

	if exists {
		log.WithFields(log.Fields{
			"topic": id,
		}).Debug("trigger.pubsub: topic exists")
		return nil
	}

	_, err = s.client.CreateTopic(ctx, id)
	return err
}

func (s *PubsubSubscriber) ensureSubscription(ctx context.Context, subscriptionID, topicID string) error {
	sub := s.client.Subscription(subscriptionID)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check whether subscription exists, error: %s", err)
	}
	if exists {
		log.WithFields(log.Fields{
			"subscription": subscriptionID,
			"topic":        topicID,
		}).Debug("trigger.pubsub: subscription exists")
		return nil
	}

	_, err = s.client.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{
		Topic:       s.client.Topic(topicID),
		AckDeadline: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create subscription %s, error: %s", subscriptionID, err)
	}
	return nil
}

// Subscribe - initiate PubsubSubscriber
func (s *PubsubSubscriber) Subscribe(ctx context.Context, topic, subscription string) error {
	// ensuring that topic exists
	err := s.ensureTopic(ctx, topic)
	if err != nil {
		return err
	}

	err = s.ensureSubscription(ctx, subscription, topic)
	if err != nil {
		return err
	}

	sub := s.client.Subscription(subscription)
	log.WithFields(log.Fields{
		"topic":        topic,
		"subscription": subscription,
	}).Info("trigger.pubsub: subscribing for events...")
	err = sub.Receive(ctx, s.callback)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.pubsub: got error while subscribing")
	}
	return err
}

func (s *PubsubSubscriber) callback(ctx context.Context, msg *pubsub.Message) {
	// disable ack, useful for testing
	if !s.disableAck {
		defer msg.Ack()
	}

	var decoded Message
	err := json.Unmarshal(msg.Data, &decoded)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.pubsub: failed to decode message")
		return
	}

	// we only care about "INSERT" (push) events
	if decoded.Action != "INSERT" {
		return
	}

	if decoded.Tag == "" {
		return
	}

	ref, err := image.Parse(decoded.Tag)

	// imageName, parsedVersion, err := version.GetImageNameAndVersion(decoded.Tag)
	if err != nil {
		log.WithFields(log.Fields{
			"action": decoded.Action,
			"tag":    decoded.Tag,
			"error":  err,
		}).Warn("trigger.pubsub: failed to parse image name")
		return
	}

	// sending event to the providers
	log.WithFields(log.Fields{
		"action":     decoded.Action,
		"tag":        ref.Tag(),
		"image_name": ref.Name(),
	}).Debug("trigger.pubsub: got message")
	event := types.Event{
		Repository: types.Repository{Name: ref.Repository(), Tag: ref.Tag()},
		CreatedAt:  time.Now(),
	}

	s.providers.Submit(event)
}
