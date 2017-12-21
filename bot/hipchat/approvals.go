package hipchat

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/bot/formatter"
	"github.com/keel-hq/keel/types"

	log "github.com/Sirupsen/logrus"
)

func (b *Bot) subscribeForApprovals() error {
	approvalsCh, err := b.approvalsManager.Subscribe(b.ctx)
	if err != nil {
		log.Errorf("hipchat.subscribeForApprovals(): %s", err.Error())
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
	msg := fmt.Sprintf(`Approval required!
		%s
		To vote for change type '%s approve %s'
		To reject it: '%s reject %s'
			Votes: %d/%d
			Delta: %s
			Identifier: %s
			Provider: %s`,
		req.Message, b.mentionName, req.Identifier, b.mentionName, req.Identifier,
		req.VotesReceived, req.VotesRequired, req.Delta(), req.Identifier,
		req.Provider.String())
	return b.postMessage(msg)
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
		msg := fmt.Sprintf(`Vote received
			Waiting for remaining votes!
				Votes: %d/%d
				Delta: %s
				Identifier: %s`,
			approval.VotesReceived, approval.VotesRequired, approval.Delta(), approval.Identifier)
		b.postMessage(msg)
	case types.ApprovalStatusRejected:
		msg := fmt.Sprintf(`change rejected
			Change was rejected.
				Status: %s
				Votes: %d/%d
				Delta: %s
				Identifier: %s`,
			approval.Status().String(), approval.VotesReceived, approval.VotesRequired,
			approval.Delta(), approval.Identifier)
		b.postMessage(msg)
	case types.ApprovalStatusApproved:
		msg := fmt.Sprintf(`Update approved!
			All approvals received, thanks for voting!
				Votes: %d/%d
				Delta: %s
				Identifier: %s`,
			approval.VotesReceived, approval.VotesRequired, approval.Delta(), approval.Identifier)
		b.postMessage(msg)
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
