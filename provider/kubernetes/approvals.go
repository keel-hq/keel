package kubernetes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/keel-hq/keel/cache"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

func getIdentifier(namespace, name, version string) string {
	return namespace + "/" + name + ":" + version
}

// checkForApprovals - filters out deployments and only passes forward approved ones
func (p *Provider) checkForApprovals(event *types.Event, plans []*UpdatePlan) (approvedPlans []*UpdatePlan) {
	approvedPlans = []*UpdatePlan{}
	for _, plan := range plans {
		approved, err := p.isApproved(event, plan)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"deployment": plan.Deployment.Name,
				"namespace":  plan.Deployment.Namespace,
			}).Error("provider.kubernetes: failed to check approval status for deployment")
			continue
		}
		if approved {
			approvedPlans = append(approvedPlans, plan)
		}
	}
	return approvedPlans
}

func (p *Provider) isApproved(event *types.Event, plan *UpdatePlan) (bool, error) {
	labels := plan.Deployment.GetLabels()

	minApprovalsStr, ok := labels[types.KeelMinimumApprovalsLabel]
	if !ok {
		// no approvals required - passing
		return true, nil
	}

	minApprovals, err := strconv.Atoi(minApprovalsStr)
	if err != nil {
		return false, err
	}

	if minApprovals == 0 {
		return true, nil
	}

	deadline := types.KeelApprovalDeadlineDefault

	// deadline
	deadlineStr, ok := labels[types.KeelApprovalDeadlineLabel]
	if ok {
		d, err := strconv.Atoi(deadlineStr)
		if err == nil {
			deadline = d
		}
	}

	identifier := getIdentifier(plan.Deployment.Namespace, plan.Deployment.Name, plan.NewVersion)

	// checking for existing approval
	existing, err := p.approvalManager.Get(identifier)
	if err != nil {
		if err == cache.ErrNotFound {

			// creating new one
			approval := &types.Approval{
				Provider:       types.ProviderTypeKubernetes,
				Identifier:     identifier,
				Event:          event,
				CurrentVersion: plan.CurrentVersion,
				NewVersion:     plan.NewVersion,
				VotesRequired:  minApprovals,
				VotesReceived:  0,
				Rejected:       false,
				Deadline:       time.Now().Add(time.Duration(deadline) * time.Hour),
			}

			approval.Message = fmt.Sprintf("New image is available for deployment %s/%s (%s).",
				plan.Deployment.Namespace,
				plan.Deployment.Name,
				approval.Delta(),
			)

			fmt.Println("requesting approval, ns: ", plan.Deployment.Namespace)

			return false, p.approvalManager.Create(approval)
		}

		return false, err
	}

	return existing.Status() == types.ApprovalStatusApproved, nil
}
