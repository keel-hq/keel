package slack

import (
	"fmt"

	"github.com/keel-hq/keel/types"
	"github.com/nlopes/slack"
)

// Request - request approval
func (b *Bot) RequestApproval(req *types.Approval) error {
	return b.postMessage(
		"Approval required",
		req.Message,
		types.LevelSuccess.Color(),
		[]slack.AttachmentField{
			slack.AttachmentField{
				Title: "Approval required!",
				Value: req.Message + "\n" + fmt.Sprintf("To vote for change type '%s approve %s' to reject it: '%s reject %s'.", b.name, req.Identifier, b.name, req.Identifier),
				Short: false,
			},
			slack.AttachmentField{
				Title: "Votes",
				Value: fmt.Sprintf("%d/%d", req.VotesReceived, req.VotesRequired),
				Short: true,
			},
			slack.AttachmentField{
				Title: "Delta",
				Value: req.Delta(),
				Short: true,
			},
			slack.AttachmentField{
				Title: "Identifier",
				Value: req.Identifier,
				Short: true,
			},
			slack.AttachmentField{
				Title: "Provider",
				Value: req.Provider.String(),
				Short: true,
			},
		})
}

func (b *Bot) ReplyToApproval(approval *types.Approval) error {
	switch approval.Status() {
	case types.ApprovalStatusPending:
		b.postMessage(
			"Vote received",
			"All approvals received, thanks for voting!",
			types.LevelInfo.Color(),
			[]slack.AttachmentField{
				slack.AttachmentField{
					Title: "vote received!",
					Value: "Waiting for remaining votes.",
					Short: false,
				},
				slack.AttachmentField{
					Title: "Votes",
					Value: fmt.Sprintf("%d/%d", approval.VotesReceived, approval.VotesRequired),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Delta",
					Value: approval.Delta(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Identifier",
					Value: approval.Identifier,
					Short: true,
				},
			})
	case types.ApprovalStatusRejected:
		b.postMessage(
			"Change rejected",
			"Change was rejected",
			types.LevelWarn.Color(),
			[]slack.AttachmentField{
				slack.AttachmentField{
					Title: "change rejected",
					Value: "Change was rejected.",
					Short: false,
				},
				slack.AttachmentField{
					Title: "Status",
					Value: approval.Status().String(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Votes",
					Value: fmt.Sprintf("%d/%d", approval.VotesReceived, approval.VotesRequired),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Delta",
					Value: approval.Delta(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Identifier",
					Value: approval.Identifier,
					Short: true,
				},
			})
	case types.ApprovalStatusApproved:
		b.postMessage(
			"approval received",
			"All approvals received, thanks for voting!",
			types.LevelSuccess.Color(),
			[]slack.AttachmentField{
				slack.AttachmentField{
					Title: "update approved!",
					Value: "All approvals received, thanks for voting!",
					Short: false,
				},
				slack.AttachmentField{
					Title: "Votes",
					Value: fmt.Sprintf("%d/%d", approval.VotesReceived, approval.VotesRequired),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Delta",
					Value: approval.Delta(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Identifier",
					Value: approval.Identifier,
					Short: true,
				},
			})
	}
	return nil
}
