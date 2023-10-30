package policy

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
)

// SemverPolicyType - policy type
type SemverPolicyType int

var (
	ErrNoMajorMinorPatchElementsFound = errors.New("No Major.Minor.Patch elements found")
)

// available policies
const (
	SemverPolicyTypeNone SemverPolicyType = iota
	SemverPolicyTypeAll
	SemverPolicyTypeMajor
	SemverPolicyTypeMinor
	SemverPolicyTypePatch
)

func (t SemverPolicyType) String() string {
	switch t {
	case SemverPolicyTypeNone:
		return "none"
	case SemverPolicyTypeAll:
		return "all"
	case SemverPolicyTypeMajor:
		return "major"
	case SemverPolicyTypeMinor:
		return "minor"
	case SemverPolicyTypePatch:
		return "patch"
	default:
		return ""
	}
}

func NewSemverPolicy(spt SemverPolicyType, matchPreRelease bool) *SemverPolicy {
	return &SemverPolicy{
		spt:             spt,
		matchPreRelease: matchPreRelease,
	}
}

type SemverPolicy struct {
	spt             SemverPolicyType
	matchPreRelease bool
}

func (sp *SemverPolicy) ShouldUpdate(current, new string) (bool, error) {
	return shouldUpdate(sp.spt, sp.matchPreRelease, current, new)
}

func (sp *SemverPolicy) Name() string {
	return sp.spt.String()
}

func (sp *SemverPolicy) Type() PolicyType { return PolicyTypeSemver }

func shouldUpdate(spt SemverPolicyType, matchPreRelease bool, current, new string) (bool, error) {
	if current == "latest" {
		return true, nil
	}

	parts := strings.SplitN(new, ".", 3)
	if len(parts) != 2 && len(parts) != 3 {
		return false, ErrNoMajorMinorPatchElementsFound
	}

	currentVersion, err := semver.NewVersion(current)
	if err != nil {
		return false, fmt.Errorf("failed to parse current version: %s", err)
	}

	newVersion, err := semver.NewVersion(new)
	if err != nil {
		return false, fmt.Errorf("failed to parse new version: %s", err)
	}

	// Do not enforce pre-release match when either:
	// - All policy
	// - matchPreRelease set to false
	if currentVersion.Prerelease() != newVersion.Prerelease() && spt != SemverPolicyTypeAll && matchPreRelease {
		return false, nil
	}

	// new version is not higher than current - do nothing
	if !currentVersion.LessThan(newVersion) {
		return false, nil
	}

	switch spt {
	case SemverPolicyTypeAll, SemverPolicyTypeMajor:
		return true, nil
	case SemverPolicyTypeMinor:
		return newVersion.Major() == currentVersion.Major(), nil
	case SemverPolicyTypePatch:
		return newVersion.Major() == currentVersion.Major() && newVersion.Minor() == currentVersion.Minor(), nil
	}
	return false, nil
}

func (sp *SemverPolicy) Filter(tags []string) []string {
	var versions []*semver.Version
	var filtered []string

	for _, t := range tags {
		if len(strings.SplitN(t, ".", 3)) < 2 {
			// Keep only X.Y.Z+ semver
			continue
		}
		v, err := semver.NewVersion(t)
		// Filter out non semver tags
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}

	sort.Slice(versions, func(i, j int) bool { return versions[j].LessThan(versions[i]) })

	for _, version := range versions {
		filtered = append(filtered, version.Original())
	}

	return filtered
}
