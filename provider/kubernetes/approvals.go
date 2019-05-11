package kubernetes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/keel-hq/keel/pkg/store"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

func getApprovalIdentifier(resourceIdentifier, version string) string {
	return resourceIdentifier + ":" + version
}

// checkForApprovals - filters out deployments and only passes forward approved ones
func (p *Provider) checkForApprovals(event *types.Event, plans []*UpdatePlan) (approvedPlans []*UpdatePlan) {
	approvedPlans = []*UpdatePlan{}
	for _, plan := range plans {
		approved, err := p.isApproved(event, plan)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"name":      plan.Resource.Name,
				"namespace": plan.Resource.Namespace,
			}).Error("provider.kubernetes: failed to check approval status for deployment")
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
	return p.approvalManager.Archive(getApprovalIdentifier(plan.Resource.Identifier, plan.NewVersion))
}

func getInt(key string, labels map[string]string, annotations map[string]string) (int, error) {

	var (
		valStr string
		ok     bool
	)

	valStr, ok = labels[key]
	if ok {
		valInt, err := strconv.Atoi(valStr)
		if err != nil {
			return 0, err
		}
		return valInt, nil
	}

	valStr, ok = annotations[key]
	if ok {
		valInt, err := strconv.Atoi(valStr)
		if err != nil {
			return 0, err
		}
		return valInt, nil
	}

	return 0, nil
}

func (p *Provider) isApproved(event *types.Event, plan *UpdatePlan) (bool, error) {

	minApprovals, err := getInt(types.KeelMinimumApprovalsLabel, plan.Resource.GetLabels(), plan.Resource.GetAnnotations())
	if err != nil {
		return false, err
	}

	if minApprovals == 0 {
		return true, nil
	}

	// deadline
	deadline := types.KeelApprovalDeadlineDefault
	d, err := getInt(types.KeelApprovalDeadlineLabel, plan.Resource.GetLabels(), plan.Resource.GetAnnotations())
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"resource": plan.Resource.GetName(),
		}).Warn("failed to parse approvals deadline, using default value")
	} else if d != 0 {
		deadline = d
	}

	identifier := getApprovalIdentifier(plan.Resource.Identifier, plan.NewVersion)

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

			approval.Message = fmt.Sprintf("New image is available for resource %s/%s (%s).",
				plan.Resource.Namespace,
				plan.Resource.Name,
				approval.Delta(),
			)

			return false, p.approvalManager.Create(approval)
		}

		return false, err
	}

	// if event.Repository.Digest != "" && event.Repository.Digest != existing.Digest {
	// 	err = p.approvalManager.Reset(existing)
	// 	if err != nil {
	// 		return false, fmt.Errorf("failed to reset approval after changed digest, error %s", err)
	// 	}
	// 	return false, nil
	// }
	// log.WithFields(log.Fields{
	// 	"previous": existing.Digest,
	// 	"new":      event.Repository.Digest,
	// }).Info("digests match")

	return existing.Status() == types.ApprovalStatusApproved, nil
}
