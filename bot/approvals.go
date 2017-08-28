package bot

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/nlopes/slack"
	"github.com/rusenask/keel/bot/formatter"
	"github.com/rusenask/keel/types"

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
				Value: req.Message,
				Short: false,
			},
			slack.AttachmentField{
				Title: "Required",
				Value: fmt.Sprint(req.VotesRequired),
				Short: true,
			},
			slack.AttachmentField{
				Title: "Current",
				Value: "0",
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

func (b *Bot) processApprovedResponse(approvalResponse *approvalResponse) error {
	trimmed := strings.TrimPrefix(approvalResponse.Text, approvalResponseKeyword)
	identifiers := strings.Split(trimmed, " ")
	if len(identifiers) == 0 {
		return nil
	}

	for _, identifier := range identifiers {
		if identifier == "" {
			continue
		}
		fmt.Println("approving: ", identifier)
		approval, err := b.approvalsManager.Approve(identifier)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: failed to approve")
			continue
		}

		fmt.Println("approved: ", identifier)

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

func (b *Bot) processRejectedResponse(approvalResponse *approvalResponse) error {
	trimmed := strings.TrimPrefix(approvalResponse.Text, rejectResponseKeyword)
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
	fmt.Println("replying")
	switch approval.Status() {
	case types.ApprovalStatusPending:
		b.postMessage(
			"Vote received",
			"All approvals received, thanks for voting!",
			types.LevelInfo.Color(),
			[]slack.AttachmentField{
				slack.AttachmentField{
					Title: "Vote received!",
					Value: "Waiting for remaining votes to proceed with update.",
					Short: false,
				},
				slack.AttachmentField{
					Title: "Delta",
					Value: approval.Delta(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Votes",
					Value: fmt.Sprintf("%d/%d", approval.VotesReceived, approval.VotesRequired),
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
					Title: "Change rejected",
					Value: "Change was rejected. Thanks for voting!",
					Short: false,
				},
				slack.AttachmentField{
					Title: "Status",
					Value: approval.Status().String(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Delta",
					Value: approval.Delta(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Votes",
					Value: fmt.Sprintf("%d/%d", approval.VotesReceived, approval.VotesRequired),
					Short: true,
				},
			})
	case types.ApprovalStatusApproved:
		b.postMessage(
			"Approval received",
			"All approvals received, thanks for voting!",
			types.LevelSuccess.Color(),
			[]slack.AttachmentField{
				slack.AttachmentField{
					Title: "Update approved!",
					Value: "All approvals received, thanks for voting!",
					Short: false,
				},
				slack.AttachmentField{
					Title: "Delta",
					Value: approval.Delta(),
					Short: true,
				},
				slack.AttachmentField{
					Title: "Votes",
					Value: fmt.Sprintf("%d/%d", approval.VotesReceived, approval.VotesRequired),
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
