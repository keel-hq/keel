package hipchat

import (
	"fmt"

	"github.com/keel-hq/keel/types"
)

func (b *Bot) RequestApproval(req *types.Approval) error {
	msg := fmt.Sprintf(ApprovalRequiredTempl,
		req.Message, req.Identifier, req.Identifier,
		req.VotesReceived, req.VotesRequired, req.Delta(), req.Identifier,
		req.Provider.String())
	return b.postMessage(formatAsSnippet(msg))
}

func (b *Bot) ReplyToApproval(approval *types.Approval) error {
	switch approval.Status() {
	case types.ApprovalStatusPending:
		msg := fmt.Sprintf(VoteReceivedTempl,
			approval.VotesReceived, approval.VotesRequired, approval.Delta(), approval.Identifier)
		b.postMessage(formatAsSnippet(msg))
	case types.ApprovalStatusRejected:
		msg := fmt.Sprintf(ChangeRejectedTempl,
			approval.Status().String(), approval.VotesReceived, approval.VotesRequired,
			approval.Delta(), approval.Identifier)
		b.postMessage(formatAsSnippet(msg))
	case types.ApprovalStatusApproved:
		msg := fmt.Sprintf(UpdateApprovedTempl,
			approval.VotesReceived, approval.VotesRequired, approval.Delta(), approval.Identifier)
		b.postMessage(formatAsSnippet(msg))
	}
	return nil
}
