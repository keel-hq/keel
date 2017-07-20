package poll

import (
	"context"
	"sync"
	"time"

	// "github.com/rusenask/keel/image"
	// "github.com/rusenask/keel/provider/helm"
	// "github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

// DefaultManager - default manager is responsible for scanning deployments and identifying
// deployments that have market
type DefaultManager struct {
	// kubernetes implementer
	// implementer kubernetes.Implementer

	providers provider.Providers

	// repository watcher
	watcher Watcher

	mu *sync.Mutex

	// scanTick - scan interval in seconds, defaults to 60 seconds
	scanTick int

	// root context
	ctx context.Context
}

// NewPollManager - new default poller
func NewPollManager(providers provider.Providers, watcher Watcher) *DefaultManager {
	return &DefaultManager{
		// implementer: implementer,
		// helmManager: helmManager,
		providers: providers,
		watcher:   watcher,
		mu:        &sync.Mutex{},
		scanTick:  55,
	}
}

// Start - start scanning deployment for changes
func (s *DefaultManager) Start(ctx context.Context) error {
	// setting root context
	s.ctx = ctx

	// initial scan
	err := s.scan(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.poll.manager: scan failed")
	}

	for _ = range time.Tick(time.Duration(s.scanTick) * time.Second) {
		select {
		case <-ctx.Done():
			return nil
		default:
			log.Debug("performing scan")
			err := s.scan(ctx)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("trigger.poll.manager: kubernetes scan failed")
			}
		}
	}

	return nil
}

func (s *DefaultManager) scan(ctx context.Context) error {
	trackedImages, err := s.providers.TrackedImages()
	if err != nil {
		return err
	}

	for _, trackedImage := range trackedImages {
		if trackedImage.Trigger != types.TriggerTypePoll {
			continue
		}

		err := s.watcher.Watch(trackedImage.Image.Remote(), trackedImage.PollSchedule, "", "")
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"schedule": trackedImage.PollSchedule,
				"image":    trackedImage.Image.Remote(),
			}).Error("trigger.poll.manager: failed to start watching repository")
			return err
		}
	}
	return nil
}

// func (s *DefaultManager) scanKubernetes(ctx context.Context) error {
// 	deploymentLists, err := s.deployments()
// 	if err != nil {
// 		return err
// 	}

// 	for _, deploymentList := range deploymentLists {
// 		for _, deployment := range deploymentList.Items {
// 			labels := deployment.GetLabels()

// 			// ignoring unlabelled deployments
// 			policy := policies.GetPolicy(labels)
// 			if policy == types.PolicyTypeNone {
// 				continue
// 			}

// 			// trigger type, we only care for "poll" type triggers
// 			trigger := policies.GetTriggerPolicy(labels)
// 			if trigger != types.TriggerTypePoll {
// 				continue
// 			}

// 			err = s.checkDeployment(&deployment)
// 			if err != nil {
// 				log.WithFields(log.Fields{
// 					"error":      err,
// 					"deployment": deployment.Name,
// 					"namespace":  deployment.Namespace,
// 				}).Error("trigger.poll.manager: failed to check deployment poll status")
// 			}
// 		}
// 	}
// 	return nil
// }

// // checkDeployment - checks whether we are already watching for this deployment
// func (s *DefaultManager) checkDeployment(deployment *v1beta1.Deployment) error {
// 	annotations := deployment.GetAnnotations()

// 	for _, c := range deployment.Spec.Template.Spec.Containers {

// 		schedule, ok := annotations[types.KeelPollScheduleAnnotation]
// 		if ok {
// 			_, err := cron.Parse(schedule)
// 			if err != nil {
// 				log.WithFields(log.Fields{
// 					"error":      err,
// 					"schedule":   schedule,
// 					"image":      c.Image,
// 					"deployment": deployment.Name,
// 					"namespace":  deployment.Namespace,
// 				}).Error("trigger.poll.manager: failed to parse poll schedule")
// 				return err
// 			}
// 		} else {
// 			schedule = types.KeelPollDefaultSchedule
// 		}

// 		err := s.watcher.Watch(c.Image, schedule, "", "")
// 		if err != nil {
// 			log.WithFields(log.Fields{
// 				"error":      err,
// 				"schedule":   schedule,
// 				"image":      c.Image,
// 				"deployment": deployment.Name,
// 				"namespace":  deployment.Namespace,
// 			}).Error("trigger.poll.manager: failed to start watching repository")
// 			return err
// 		}
// 		// continue
// 	}

// 	return nil
// }

// func (s *DefaultManager) deployments() ([]*v1beta1.DeploymentList, error) {
// 	// namespaces := p.client.Namespaces()
// 	deployments := []*v1beta1.DeploymentList{}

// 	n, err := s.implementer.Namespaces()
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, n := range n.Items {
// 		l, err := s.implementer.Deployments(n.GetName())
// 		if err != nil {
// 			log.WithFields(log.Fields{
// 				"error":     err,
// 				"namespace": n.GetName(),
// 			}).Error("trigger.pubsub.manager: failed to list deployments")
// 			continue
// 		}
// 		deployments = append(deployments, l)
// 	}

// 	return deployments, nil
// }
