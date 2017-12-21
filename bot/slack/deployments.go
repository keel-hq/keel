package slack

import (
	"bytes"
	"fmt"

	"github.com/keel-hq/keel/bot/formatter"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	log "github.com/Sirupsen/logrus"
)

// Filter - deployment filter
type Filter struct {
	Namespace string
	All       bool // keel or not
}

// deployments - gets all deployments
func (b *Bot) deployments() ([]v1beta1.Deployment, error) {
	deploymentLists := []*v1beta1.DeploymentList{}

	n, err := b.k8sImplementer.Namespaces()
	if err != nil {
		return nil, err
	}

	for _, n := range n.Items {
		l, err := b.k8sImplementer.Deployments(n.GetName())
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"namespace": n.GetName(),
			}).Error("provider.kubernetes: failed to list deployments")
			continue
		}
		deploymentLists = append(deploymentLists, l)
	}

	impacted := []v1beta1.Deployment{}

	for _, deploymentList := range deploymentLists {
		for _, deployment := range deploymentList.Items {
			impacted = append(impacted, deployment)
		}
	}

	return impacted, nil
}

func (b *Bot) deploymentsResponse(filter Filter) string {
	deps, err := b.deployments()
	if err != nil {
		return fmt.Sprintf("got error while fetching deployments: %s", err)
	}
	log.Debugf("%d deployments fetched, formatting", len(deps))
	buf := &bytes.Buffer{}

	DeploymentCtx := formatter.Context{
		Output: buf,
		Format: formatter.NewDeploymentsFormat(formatter.TableFormatKey, false),
	}
	err = formatter.DeploymentWrite(DeploymentCtx, convertToInternal(deps))

	if err != nil {
		return fmt.Sprintf(" got error while formatting deployments: %s", err)
	}

	return buf.String()
}

func convertToInternal(deployments []v1beta1.Deployment) []formatter.Deployment {
	formatted := []formatter.Deployment{}
	for _, d := range deployments {

		formatted = append(formatted, formatter.Deployment{
			Namespace:         d.Namespace,
			Name:              d.Name,
			Replicas:          d.Status.Replicas,
			AvailableReplicas: d.Status.AvailableReplicas,
			Images:            getImages(&d),
		})
	}
	return formatted
}

func getImages(deployment *v1beta1.Deployment) []string {
	var images []string
	for _, c := range deployment.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
	}

	return images
}
