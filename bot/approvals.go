package bot

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/bot/formatter"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

type BotRequestApproval func(req *types.Approval) error
type BotReplyApproval func(approval *types.Approval) error

func (bm *BotManager) SubscribeForApprovals(ctx context.Context, approval BotRequestApproval) error {
	approvalsCh, err := bm.approvalsManager.Subscribe(ctx)
	if err != nil {
		log.Errorf("bot.subscribeForApprovals(): %s", err.Error())
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case a := <-approvalsCh:
			err = approval(a)
			if err != nil {
				log.WithFields(log.Fields{
					"error":    err,
					"approval": a.Identifier,
				}).Error("bot.subscribeForApprovals: approval request failed")
			}
		}
	}
}

func (bm *BotManager) ProcessApprovalResponses(ctx context.Context, reply BotReplyApproval) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case resp := <-bm.approvalsRespCh:
			switch resp.Status {
			case types.ApprovalStatusApproved:
				err := bm.processApprovedResponse(resp, reply)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("bot.processApprovalResponses: failed to process approval response message")
				}
			case types.ApprovalStatusRejected:
				err := bm.processRejectedResponse(resp, reply)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("bot.processApprovalResponses: failed to process approval reject response message")
				}
			}
		}
	}
}

func (bm *BotManager) processApprovedResponse(approvalResponse *ApprovalResponse, reply BotReplyApproval) error {
	trimmed := strings.TrimPrefix(approvalResponse.Text, ApprovalResponseKeyword)
	identifiers := strings.Split(trimmed, " ")
	if len(identifiers) == 0 {
		return nil
	}

	for _, identifier := range identifiers {
		if identifier == "" {
			continue
		}
		approval, err := bm.approvalsManager.Approve(identifier, approvalResponse.User)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: failed to approve")
			continue
		}

		err = reply(approval)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: got error while replying after processing approved approval")
		}
	}
	return nil
}

func (bm *BotManager) processRejectedResponse(approvalResponse *ApprovalResponse, reply BotReplyApproval) error {
	trimmed := strings.TrimPrefix(approvalResponse.Text, RejectResponseKeyword)
	identifiers := strings.Split(trimmed, " ")
	if len(identifiers) == 0 {
		return nil
	}

	for _, identifier := range identifiers {
		approval, err := bm.approvalsManager.Reject(identifier)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: failed to reject")
			continue
		}

		err = reply(approval)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"identifier": identifier,
			}).Error("bot.processApprovedResponse: got error while replying after processing rejected approval")
		}
	}
	return nil
}

func ApprovalsResponse(approvalsManager approvals.Manager) string {
	approvals, err := approvalsManager.List()
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

func IsApproval(eventUser string, eventText string) (resp *ApprovalResponse, ok bool) {
	if strings.HasPrefix(strings.ToLower(eventText), ApprovalResponseKeyword) {
		return &ApprovalResponse{
			User:   eventUser,
			Status: types.ApprovalStatusApproved,
			Text:   eventText,
		}, true
	}

	if strings.HasPrefix(strings.ToLower(eventText), RejectResponseKeyword) {
		return &ApprovalResponse{
			User:   eventUser,
			Status: types.ApprovalStatusRejected,
			Text:   eventText,
		}, true
	}

	return nil, false
}

func RemoveApprovalHandler(identifier string, approvalsManager approvals.Manager) string {
	err := approvalsManager.Delete(identifier)
	if err != nil {
		return fmt.Sprintf("failed to remove '%s' approval: %s.", identifier, err)
	}
	return fmt.Sprintf("approval '%s' removed.", identifier)
}
