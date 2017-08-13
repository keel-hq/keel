package kubernetes

import (
	"fmt"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/cache"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

func getIdentifier(namespace, name string) string {
	return namespace + "/" + name
}

// checkForApprovals - filters out deployments and only passes forward approved ones
func (p *Provider) checkForApprovals(event *types.Event, deployments []v1beta1.Deployment) (approved []v1beta1.Deployment) {
	for _, deployment := range deployments {
		approvedDeployment, err := p.isApproved(event, deployment)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"deployment": deployment.Name,
				"namespace":  deployment.Namespace,
			}).Error("provider.kubernetes: failed to check approval status for deployment")
			continue
		}
		if approvedDeployment {
			approved = append(approved, deployment)
		}
	}
	return approved
}

func (p *Provider) isApproved(event *types.Event, deployment v1beta1.Deployment) (bool, error) {
	labels := deployment.GetLabels()

	minApprovals, ok := labels[types.KeelMinimumApprovalsLabel]
	if !ok {
		// no approvals required - passing
		return true, nil
	}

	if minApprovals == "0" {
		return true, nil
	}

	identifier := getIdentifier(deployment.Namespace, deployment.Name)

	// checking for existing approval
	existing, err := p.approvalManager.Get(types.ProviderTypeKubernetes, identifier)
	if err != nil {
		if err == cache.ErrNotFound {

			// creating new one
			approval := &types.Approval{
				Provider:   types.ProviderTypeKubernetes,
				Identifier: identifier,
				Event:      event,
				Message:    fmt.Sprintf("New image is available for deployment %s/%s"),
			}

			return false, p.approvalManager.Create(approval)
		}

		return false, err
	}

	return existing.Approved(), nil
}
