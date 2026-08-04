package policy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/keel-hq/keel/types"
)

// NewestPolicy filters tags by regexp but relies on the watcher to sort by image creation date.
// Configured via annotation: keel.sh/policy: "newest:<regexp>"
type NewestPolicy struct {
	policy string
	regexp *regexp.Regexp
}

func NewNewestPolicy(policy string) (*NewestPolicy, error) {
	if strings.Contains(policy, ":") {
		parts := strings.SplitN(policy, ":", 2)
		if len(parts) == 2 {
			rx, err := regexp.Compile(parts[1])
			if err != nil {
				return nil, fmt.Errorf("failed to parse regexp pattern, error: %s", err)
			}
			return &NewestPolicy{
				regexp: rx,
				policy: policy,
			}, nil
		}
	}
	return nil, fmt.Errorf("invalid newest policy: %s", policy)
}

func (p *NewestPolicy) ShouldUpdate(current, new string) (bool, error) {
	return p.regexp.MatchString(new), nil
}

// Filter returns matching tags without sorting — the watcher sorts by build date.
func (p *NewestPolicy) Filter(tags []string) []string {
	filtered := []string{}
	for _, tag := range tags {
		if p.regexp.MatchString(tag) {
			filtered = append(filtered, tag)
		}
	}
	return filtered
}

func (p *NewestPolicy) Name() string           { return p.policy }
func (p *NewestPolicy) Type() types.PolicyType { return types.PolicyTypeNewest }
func (p *NewestPolicy) KeepTag() bool          { return false }
