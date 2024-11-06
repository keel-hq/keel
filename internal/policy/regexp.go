package policy

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// RegexpPolicy - regular expression based pattern
type RegexpPolicy struct {
	policy string
	regexp *regexp.Regexp
}

func NewRegexpPolicy(policy string) (*RegexpPolicy, error) {
	if strings.Contains(policy, ":") {
		parts := strings.SplitN(policy, ":", 2)
		if len(parts) == 2 {

			rx, err := regexp.Compile(parts[1])
			if err != nil {
				return nil, fmt.Errorf("failed to parse regexp pattern, error: %s", err)
			}

			return &RegexpPolicy{
				regexp: rx,
				policy: policy,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid regexp policy: %s", policy)
}

func (p *RegexpPolicy) ShouldUpdate(current, new string) (bool, error) {
	return p.regexp.MatchString(new), nil
}

func (p *RegexpPolicy) Filter(tags []string) []string {
	filtered := []string{}
	compare := p.regexp.SubexpIndex("compare")

	for _, tag := range tags {
		if p.regexp.MatchString(tag) {
			filtered = append(filtered, tag)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if compare != -1 {
			mi := p.regexp.FindStringSubmatch(filtered[i])
			mj := p.regexp.FindStringSubmatch(filtered[j])
			return mi[compare] > mj[compare]
		} else {
			return filtered[i] > filtered[j]
		}
	})

	return filtered
}

func (p *RegexpPolicy) Name() string     { return p.policy }
func (p *RegexpPolicy) Type() PolicyType { return PolicyTypeRegexp }
