package helm

import (
	"fmt"
	"time"

	"github.com/keel-hq/keel/cache"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// namespace/release name/version
func getIdentifier(namespace, name, version string) string {
	return namespace + "/" + name + ":" + version
}

func (p *Provider) checkForApprovals(event *types.Event, plans []*UpdatePlan) (approvedPlans []*UpdatePlan) {
	approvedPlans = []*UpdatePlan{}
	for _, plan := range plans {
		approved, err := p.isApproved(event, plan)
		if err != nil {
			log.WithFields(log.Fields{
				"error":        err,
				"release_name": plan.Name,
				"namespace":    plan.Namespace,
			}).Error("provider.helm: failed to check approval status for deployment")
			continue
		}
		if approved {
			approvedPlans = append(approvedPlans, plan)
		}
	}
	return approvedPlans
}

func (p *Provider) isApproved(event *types.Event, plan *UpdatePlan) (bool, error) {
	if plan.Config.Approvals == 0 {
		return true, nil
	}

	identifier := getIdentifier(plan.Namespace, plan.Name, plan.NewVersion)

	// checking for existing approval
	existing, err := p.approvalManager.Get(identifier)
	if err != nil {
		if err == cache.ErrNotFound {

			// creating new one
			approval := &types.Approval{
				Provider:       types.ProviderTypeHelm,
				Identifier:     identifier,
				Event:          event,
				CurrentVersion: plan.CurrentVersion,
				NewVersion:     plan.NewVersion,
				VotesRequired:  plan.Config.Approvals,
				VotesReceived:  0,
				Rejected:       false,
				Deadline:       time.Now().Add(time.Duration(plan.Config.ApprovalDeadline) * time.Hour),
			}

			approval.Message = fmt.Sprintf("New image is available for release %s/%s (%s).",
				plan.Namespace,
				plan.Name,
				approval.Delta(),
			)

			return false, p.approvalManager.Create(approval)
		}

		return false, err
	}

	return existing.Status() == types.ApprovalStatusApproved, nil
}
