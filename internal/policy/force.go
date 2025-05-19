package policy

import "github.com/keel-hq/keel/types"

type ForcePolicy struct {
	matchTag bool
}

func NewForcePolicy(matchTag bool) *ForcePolicy {
	return &ForcePolicy{
		matchTag: matchTag,
	}
}

func (fp *ForcePolicy) ShouldUpdate(current, new string) (bool, error) {
	if fp.matchTag && current != new {
		return false, nil
	}
	return true, nil
}

func (fp *ForcePolicy) Filter(tags []string) []string {
	return append([]string{}, tags...)
}

func (fp *ForcePolicy) Name() string {
	return "force"
}

func (fp *ForcePolicy) Type() types.PolicyType { return types.PolicyTypeForce }
