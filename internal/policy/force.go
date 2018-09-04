package policy

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

func (fp *ForcePolicy) Name() string {
	return "force"
}

func (fp *ForcePolicy) Type() PolicyType { return PolicyTypeForce }
