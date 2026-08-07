package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	Digest string `json:"digest"`
	Tag    string `json:"tag,omitempty"`
}

func (s *PubsubSubscriber) ensureTopic(ctx context.Context, id string) error {
	name := fmt.Sprintf("projects/%s/topics/%s", s.project, id)
	_, err := s.client.TopicAdminClient.GetTopic(ctx, &pubsubpb.GetTopicRequest{Topic: name})
	if err == nil {
		log.WithField("topic", id).Debug("trigger.pubsub: topic exists")
		return nil
	}
	if status.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to check whether topic exists, error: %w", err)
	}

	_, err = s.client.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: name})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return err
}

func (s *PubsubSubscriber) ensureSubscription(ctx context.Context, subscriptionID, topicID string) error {
	name := fmt.Sprintf("projects/%s/subscriptions/%s", s.project, subscriptionID)
	topic := fmt.Sprintf("projects/%s/topics/%s", s.project, topicID)
	_, err := s.client.SubscriptionAdminClient.GetSubscription(ctx, &pubsubpb.GetSubscriptionRequest{Subscription: name})
	if err == nil {
		log.WithFields(log.Fields{"subscription": subscriptionID, "topic": topicID}).Debug("trigger.pubsub: subscription exists")
		return nil
	}
	if status.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to check whether subscription exists, error: %w", err)
	}

	_, err = s.client.SubscriptionAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name: name, Topic: topic, AckDeadlineSeconds: 10,
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to create subscription %s, error: %w", subscriptionID, err)
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

	sub := s.client.Subscriber(subscription)
	// Pub/Sub v1 used ten StreamingPull streams by default. Preserve that
	// concurrency explicitly because v2 defaults to one.
	sub.ReceiveSettings.NumGoroutines = 10
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
		Repository: types.Repository{
			Name:   ref.Repository(),
			Tag:    ref.Tag(),
			Digest: decoded.Digest,
		},
		CreatedAt: time.Now(),
	}

	s.providers.Submit(event)
}
