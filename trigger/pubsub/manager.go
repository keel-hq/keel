package pubsub

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

// DefaultManager - subscription manager
type DefaultManager struct {
	implementer kubernetes.Implementer

	client *Subscriber
	// existing subscribers
	mu *sync.Mutex
	// a map of GCR URIs and subscribers to those URIs
	// i.e. key could be something like: gcr.io%2Fmy-project
	subscribers map[string]context.Context

	// projectID is required to correctly set GCR subscriptions
	projectID string

	// root context
	ctx context.Context
}

func NewDefaultManager(projectID string, implementer kubernetes.Implementer, subClient *Subscriber) *DefaultManager {
	return &DefaultManager{
		implementer: implementer,
		client:      subClient,
		projectID:   projectID,
		subscribers: make(map[string]context.Context),
		mu:          &sync.Mutex{},
	}
}

func (s *DefaultManager) Start(ctx context.Context) error {
	// setting root context
	s.ctx = ctx
	for _ = range time.Tick(10 * time.Second) {
		select {
		case <-ctx.Done():
			return nil
		default:
			log.Info("performing scan")
			err := s.scan(ctx)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("trigger.pubsub.manager: scan failed")
			}
		}
	}

	return nil
}

func (s *DefaultManager) scan(ctx context.Context) error {
	deploymentLists, err := s.deployments()
	if err != nil {
		return err
	}

	for _, deploymentList := range deploymentLists {
		for _, deployment := range deploymentList.Items {
			labels := deployment.GetLabels()
			_, ok := labels[types.KeelPolicyLabel]
			// if no keel policy is set - skipping this deployment
			if !ok {
				continue
			}

			err = s.checkDeployment(&deployment)
			if err != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"deployment": deployment.Name,
					"namespace":  deployment.Namespace,
				}).Error("trigger.pubsub.manager: failed to check deployment subscription status")
			}
		}
	}

	return nil
}

func (s *DefaultManager) subscribed(gcrURI string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.subscribers[gcrURI]
	return ok
}

func (s *DefaultManager) addSubscription(ctx context.Context, gcrURI string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[gcrURI] = ctx
}

func (s *DefaultManager) removeSubscription(gcrURI string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, gcrURI)
}

// checkDeployment - gets deployment image and checks whether we have appropriate topic
// and subscription for this deployment
func (s *DefaultManager) checkDeployment(deployment *v1beta1.Deployment) error {

	for _, c := range deployment.Spec.Template.Spec.Containers {
		// registry host
		registry := extractContainerRegistryURI(c.Image)

		if !isGoogleContainerRegistry(registry) {
			log.Infof("registry %s is not a GCR, skipping", registry)
			continue
		}

		// uri
		gcrURI := containerRegistryURI(s.projectID, registry)

		// existing sub
		ok := s.subscribed(gcrURI)
		if !ok {
			// create sub in a separate goroutine since client.Subscribe is a blocking call
			go func() {
				ctx, cancel := context.WithCancel(s.ctx)
				defer cancel()
				s.addSubscription(ctx, gcrURI)
				defer s.removeSubscription(gcrURI)
				subName := containerRegistrySubName(s.projectID, gcrURI)
				err := s.client.Subscribe(s.ctx, gcrURI, subName)
				if err != nil {
					log.WithFields(log.Fields{
						"error":             err,
						"gcr_uri":           gcrURI,
						"subscription_name": subName,
					}).Error("trigger.pubsub.manager: failed to create subscription")
				}
			}()
		} else {
			log.WithFields(log.Fields{
				"registry":   registry,
				"gcr_uri":    gcrURI,
				"deployment": deployment.Name,
				"image_name": c.Image,
			}).Info("trigger.pubsub.manager: existing subscription for deployment's image found")
		}

	}

	return nil
}

func (s *DefaultManager) deployments() ([]*v1beta1.DeploymentList, error) {
	// namespaces := p.client.Namespaces()
	deployments := []*v1beta1.DeploymentList{}

	n, err := s.implementer.Namespaces()
	if err != nil {
		return nil, err
	}

	for _, n := range n.Items {
		l, err := s.implementer.Deployments(n.GetName())
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"namespace": n.GetName(),
			}).Error("trigger.pubsub.manager: failed to list deployments")
			continue
		}
		deployments = append(deployments, l)
	}

	return deployments, nil
}
