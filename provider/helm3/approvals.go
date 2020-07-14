package helm3

import (
	"fmt"
	"time"

	"github.com/keel-hq/keel/pkg/store"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// namespace/release name:version
func getIdentifier(plan *UpdatePlan) string {
	return fmt.Sprintf("%s/%s:%s", plan.Namespace, plan.Name, plan.NewVersion)
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
				"version":      plan.NewVersion,
			}).Error("provider.helm3: failed to check approval status for deployment")
			continue
		}
		if approved {
			approvedPlans = append(approvedPlans, plan)
		}
	}
	return approvedPlans
}

// updateComplete is called after we successfully update resource
func (p *Provider) updateComplete(plan *UpdatePlan) error {
	return p.approvalManager.Archive(getIdentifier(plan))
}

func (p *Provider) isApproved(event *types.Event, plan *UpdatePlan) (bool, error) {
	if plan.Config.Approvals == 0 {
		return true, nil
	}

	identifier := getIdentifier(plan)

	// checking for existing approval
	existing, err := p.approvalManager.Get(identifier)
	if err != nil {
		if err == store.ErrRecordNotFound {

			// if approval doesn't exist and trigger wasn't existing approval fulfillment -
			// create a new one, otherwise if several deployments rely on the same image, it would just be
			// requesting approvals in a loop
			if event.TriggerName == types.TriggerTypeApproval.String() {
				return false, nil
			}

			if plan.Config.ApprovalDeadline == 0 {
				plan.Config.ApprovalDeadline = types.KeelApprovalDeadlineDefault
			}

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
