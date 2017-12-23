package bot

import (
	"bytes"
	"fmt"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/bot/formatter"
)

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

func RemoveApprovalHandler(identifier string, approvalsManager approvals.Manager) string {
	err := approvalsManager.Delete(identifier)
	if err != nil {
		return fmt.Sprintf("failed to remove '%s' approval: %s.", identifier, err)
	}
	return fmt.Sprintf("approval '%s' removed.", identifier)
}
