package policy

import (
	"fmt"
	"strings"

	"github.com/ryanuber/go-glob"
)

type GlobPolicy struct {
	policy  string // original string
	pattern string // without prefix
}

func NewGlobPolicy(policy string) (*GlobPolicy, error) {
	if strings.Contains(policy, ":") {
		parts := strings.Split(policy, ":")
		if len(parts) == 2 {
			return &GlobPolicy{
				policy:  policy,
				pattern: parts[1],
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid glob policy: %s", policy)
}

func (p *GlobPolicy) ShouldUpdate(current, new string) (bool, error) {
	return glob.Glob(p.pattern, new), nil
}

func (p *GlobPolicy) Name() string     { return p.policy }
func (p *GlobPolicy) Type() PolicyType { return PolicyTypeGlob }
