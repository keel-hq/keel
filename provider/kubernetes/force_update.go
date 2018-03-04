package kubernetes

import (
	"fmt"

	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

func (p *Provider) forceUpdate(deployment *v1beta1.Deployment) (err error) {

	gracePeriod := types.ParsePodTerminationGracePeriod(deployment.Annotations)
	selector := meta_v1.FormatLabelSelector(deployment.Spec.Selector)

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

	for _, pod := range podList.Items {

		log.WithFields(log.Fields{
			"selector":     selector,
			"pod":          pod.Name,
			"namespace":    deployment.Namespace,
			"deployment":   deployment.Name,
			"grace_period": fmt.Sprint(gracePeriod),
		}).Info("provider.kubernetes: deleting pod to force pull...")

		err = p.implementer.DeletePod(deployment.Namespace, pod.Name, &meta_v1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"selector":   selector,
				"pod":        pod.Name,
				"namespace":  deployment.Namespace,
				"deployment": deployment.Name,
			}).Error("provider.kubernetes: got error while deleting a pod")
			continue
		}

	}

	return nil
}
