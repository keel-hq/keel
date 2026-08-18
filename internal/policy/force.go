package policy

import (
	"sort"

	"github.com/Masterminds/semver"
	"github.com/keel-hq/keel/types"
)

type ForcePolicy struct {
	matchTag bool
}

func NewForcePolicy(matchTag bool) *ForcePolicy {
	return &ForcePolicy{
		matchTag: matchTag,
	}
}

// ShouldUpdate - the force policy accepts non-version tags (e.g. "latest")
// for which no ordering exists, but it must never downgrade a versioned
// image: when both tags parse as semantic versions the new tag has to be
// strictly newer. A re-push of the current tag (same tag, new digest) is
// still an update.
func (fp *ForcePolicy) ShouldUpdate(current, new string) (bool, error) {
	if fp.matchTag && current != new {
		return false, nil
	}

	if current != new {
		currentVersion, currentErr := semver.NewVersion(current)
		newVersion, newErr := semver.NewVersion(new)
		if currentErr == nil && newErr == nil {
			return newVersion.GreaterThan(currentVersion), nil
		}
	}

	return true, nil
}

// Filter - orders the candidate tags so the newest one is tried first, like
// the other policies do: semantic-version tags are sorted in descending
// order, non-version tags (e.g. "latest") have no ordering so they keep
// their relative order and trail the versioned tags. Registries return tag
// lists in creation order (oldest first), so leaving the list unsorted
// would make the oldest tag in the repository win
// (https://github.com/keel-hq/keel/issues/823).
func (fp *ForcePolicy) Filter(tags []string) []string {
	var versions []*semver.Version
	var others []string

	for _, tag := range tags {
		if v, err := semver.NewVersion(tag); err == nil {
			versions = append(versions, v)
		} else {
			others = append(others, tag)
		}
	}

	sort.Sort(sort.Reverse(semver.Collection(versions)))

	filtered := make([]string, 0, len(tags))
	for _, v := range versions {
		filtered = append(filtered, v.Original())
	}
	return append(filtered, others...)
}

func (fp *ForcePolicy) Name() string {
	return "force"
}

func (fp *ForcePolicy) Type() types.PolicyType { return types.PolicyTypeForce }

func (fp *ForcePolicy) KeepTag() bool { return fp.matchTag }
