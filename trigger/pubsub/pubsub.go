package pubsub

import (
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/version"

	log "github.com/Sirupsen/logrus"
)

type Subscriber struct {
	providers map[string]provider.Provider

	project    string
	disableAck bool

	client *pubsub.Client
}

// pubsubImplementer - pubsub implementer
type pubsubImplementer interface {
	Subscription(id string) *pubsub.Subscription
	Receive(ctx context.Context, f func(context.Context, *Message)) error
}

type Opts struct {
	ProjectID string
	Providers map[string]provider.Provider
}

func NewSubscriber(opts *Opts) (*Subscriber, error) {
	client, err := pubsub.NewClient(context.Background(), opts.ProjectID)
	if err != nil {
		return nil, err
	}

	return &Subscriber{
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

func (s *Subscriber) ensureTopic(ctx context.Context, id string) error {
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

func (s *Subscriber) ensureSubscription(ctx context.Context, subscriptionID, topicID string) error {
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
	return fmt.Errorf("failed to create subscription %s, error: %s", subscriptionID, err)
}

// Subscribe - initiate subscriber
func (s *Subscriber) Subscribe(ctx context.Context, topic, subscription string) error {
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
	log.Info("trigger.pubsub: subscribing for events...")
	// err := sub.Receive(ctx, s.callback)
	err = sub.Receive(ctx, s.callback)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.pubsub: got error while subscribing")
	}
	return err
}

func (s *Subscriber) callback(ctx context.Context, msg *pubsub.Message) {
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

	if decoded.Tag == "" {
		return
	}

	imageName, parsedVersion, err := version.GetImageNameAndVersion(decoded.Tag)
	if err != nil {
		log.WithFields(log.Fields{
			"action": decoded.Action,
			"tag":    decoded.Tag,
			"error":  err,
		}).Warn("trigger.pubsub: failed to get name and version from image")
		return
	}

	// sending event to the providers
	log.WithFields(log.Fields{
		"action":  decoded.Action,
		"tag":     decoded.Tag,
		"version": parsedVersion.String(),
	}).Debug("trigger.pubsub: got message")
	event := types.Event{
		Repository: types.Repository{Name: imageName, Tag: parsedVersion.String()},
		CreatedAt:  time.Now(),
	}
	for _, p := range s.providers {
		err = p.Submit(event)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": p.GetName(),
				"version":  parsedVersion.String(),
				"image":    decoded.Tag,
			}).Error("trigger.pubsub: got error while submitting event")
		}
	}
}
