package slack

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/bot/formatter"
	"github.com/keel-hq/keel/types"
	"github.com/nlopes/slack"

	log "github.com/Sirupsen/logrus"
)

func (b *Bot) subscribeForApprovals() error {
	approvalsCh, err := b.approvalsManager.Subscribe(b.ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-b.ctx.Done():
			return nil
		case a := <-approvalsCh:
			err = b.requestApproval(a)
			if err != nil {
				log.WithFields(log.Fields{
					"error":    err,
					"approval": a.Identifier,
				}).Error("bot.subscribeForApprovals: approval request failed")
			}

		}
	}
}

// Request - request approval
func (b *Bot) requestApproval(req *types.Approval) error {
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
func (b *Bot) processApprovalResponses() error {

	for {
		select {
		case <-b.ctx.Done():
			return nil
		case resp := <-b.approvalsRespCh:

			switch resp.Status {
			case types.ApprovalStatusApproved:
				err := b.processApprovedResponse(resp)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("bot.processApprovalResponses: failed to process approval response message")
				}
			case types.ApprovalStatusRejected:
				err := b.processRejectedResponse(resp)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("bot.processApprovalResponses: failed to process approval reject response message")
				}
			}

		}
	}
}

func (b *Bot) processApprovedResponse(approvalResponse *bot.ApprovalResponse) error {
	trimmed := strings.TrimPrefix(approvalResponse.Text, bot.ApprovalResponseKeyword)
	identifiers := strings.Split(trimmed, " ")
	if len(identifiers) == 0 {
		return nil
	}

	for _, identifier := range identifiers {
		if identifier == "" {
			continue
		}
		approval, err := b.approvalsManager.Approve(identifier, approvalResponse.User)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: failed to approve")
			continue
		}

		err = b.replyToApproval(approval)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: got error while replying after processing approved approval")
		}

	}
	return nil
}

func (b *Bot) processRejectedResponse(approvalResponse *bot.ApprovalResponse) error {
	trimmed := strings.TrimPrefix(approvalResponse.Text, bot.RejectResponseKeyword)
	identifiers := strings.Split(trimmed, " ")
	if len(identifiers) == 0 {
		return nil
	}

	for _, identifier := range identifiers {
		approval, err := b.approvalsManager.Reject(identifier)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: failed to reject")
			continue
		}

		err = b.replyToApproval(approval)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: got error while replying after processing rejected approval")
		}

	}
	return nil
}

func (b *Bot) replyToApproval(approval *types.Approval) error {
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

func (b *Bot) approvalsResponse() string {
	approvals, err := b.approvalsManager.List()
	if err != nil {
		return fmt.Sprintf("got error while fetching approvals: %s", err)
	}

	if len(approvals) == 0 {
		return fmt.Sprintf("there are currently no request waiting to be approved.")
	}

	buf := &bytes.Buffer{}

	approvalCtx := formatter.Context{
		Output: buf,
		Format: formatter.NewApprovalsFormat(formatter.TableFormatKey, false),
	}
	err = formatter.ApprovalWrite(approvalCtx, approvals)

	if err != nil {
		return fmt.Sprintf("got error while formatting approvals: %s", err)
	}

	return buf.String()
}

func (b *Bot) removeApprovalHandler(identifier string) string {
	err := b.approvalsManager.Delete(identifier)
	if err != nil {
		return fmt.Sprintf("failed to remove '%s' approval: %s.", identifier, err)
	}
	return fmt.Sprintf("approval '%s' removed.", identifier)
}
