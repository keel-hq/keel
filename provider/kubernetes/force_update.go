package kubernetes

import (
	"fmt"
	"time"

	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

func (p *Provider) forceUpdate(deployment *v1beta1.Deployment) (err error) {

	gracePeriod := types.ParsePodTerminationGracePeriod(deployment.Annotations)
	selector := meta_v1.FormatLabelSelector(deployment.Spec.Selector)
	podDeleteDelay := types.ParsePodDeleteDelay(deployment.Annotations)

	// image tag didn't change, need to terminate pods
	podList, err := p.implementer.Pods(deployment.Namespace, selector)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"selector":   selector,
			"namespace":  deployment.Namespace,
			"deployment": deployment.Name,
		}).Error("provider.kubernetes: got error while looking for deployment pods")
		return err
	}

	for index, pod := range podList.Items {

		var gp int64

		if pod.DeletionGracePeriodSeconds != nil {
			gp = *pod.DeletionGracePeriodSeconds
		}
		if gracePeriod != 0 {
			gp = gracePeriod
		}

		log.WithFields(log.Fields{
			"selector":     selector,
			"pod":          pod.Name,
			"namespace":    deployment.Namespace,
			"deployment":   deployment.Name,
			"grace_period": fmt.Sprint(gp),
		}).Info("provider.kubernetes: deleting pod to force pull...")

		err = p.implementer.DeletePod(deployment.Namespace, pod.Name, &meta_v1.DeleteOptions{
			GracePeriodSeconds: &gp,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"selector":   selector,
				"pod":        pod.Name,
				"namespace":  deployment.Namespace,
				"deployment": deployment.Name,
			}).Error("provider.kubernetes: got error while deleting a pod")
		}

		// sleep between pod restarts but not if there aren't more left
		if index < len(podList.Items)-1 {
			time.Sleep(time.Duration(podDeleteDelay) * time.Second)
		}
	}

	return nil
}
