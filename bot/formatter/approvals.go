package formatter

import (
	"fmt"
	"strconv"

	"github.com/keel-hq/keel/types"
)

// Formatter headers
const (
	defaultApprovalQuietFormat = "{{.Identifier}} {{.Delta}}"
	defaultApprovalTableFormat = "table {{.Identifier}}\t{{.Delta}}\t{{.Votes}}\t{{.Rejected}}\t{{.Provider}}\t{{.Created}}"

	ApprovalIdentifierHeader = "Identifier"
	ApprovalDeltaHeader      = "Delta"
	ApprovalVotesHeader      = "Votes"
	ApprovalRejectedHeader   = "Rejected"
	ApprovalProviderHeader   = "Provider"
	ApprovalCreatedHeader    = "Created"
)

// NewApprovalsFormat returns a format for use with a approval Context
func NewApprovalsFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultApprovalQuietFormat
		}
		return defaultApprovalTableFormat
	case RawFormatKey:
		if quiet {
			return `name: {{.Identifier}}`
		}
		return `name: {{.Identifier}}\n`
	}
	return Format(source)
}

// ApprovalWrite writes formatted approvals using the Context
func ApprovalWrite(ctx Context, approvals []*types.Approval) error {
	render := func(format func(subContext subContext) error) error {
		for _, approval := range approvals {
			if err := format(&ApprovalContext{v: *approval}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(&DeploymentContext{}, render)
}

// ApprovalContext - approval context is a container for each line
type ApprovalContext struct {
	HeaderContext
	v types.Approval
}

// MarshalJSON - marshal to json (inspect)
func (c *ApprovalContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func (c *ApprovalContext) Identifier() string {
	c.AddHeader(ApprovalIdentifierHeader)
	return c.v.Identifier
}

func (c *ApprovalContext) Delta() string {
	c.AddHeader(ApprovalDeltaHeader)
	return c.v.Delta()
}

func (c *ApprovalContext) Votes() string {
	c.AddHeader(ApprovalVotesHeader)
	return fmt.Sprintf("%d/%d", c.v.VotesReceived, c.v.VotesRequired)
}

func (c *ApprovalContext) Rejected() string {
	c.AddHeader(ApprovalRejectedHeader)
	return strconv.FormatBool(c.v.Rejected)
}

func (c *ApprovalContext) Provider() string {
	c.AddHeader(ApprovalProviderHeader)
	return c.v.Provider.String()
}

func (c *ApprovalContext) Created() string {
	c.AddHeader(ApprovalCreatedHeader)
	return c.v.CreatedAt.String()
}
