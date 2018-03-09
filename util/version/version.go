package version

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// ErrVersionTagMissing - tag missing error
var ErrVersionTagMissing = errors.New("version tag is missing")

// ErrInvalidSemVer is returned a version is found to be invalid when
// being parsed.
var ErrInvalidSemVer = errors.New("invalid semantic version")

// MustParse - must parse version, if fails - panics
func MustParse(version string) *types.Version {
	ver, err := GetVersion(version)
	if err != nil {
		panic(err)
	}
	return ver
}

// GetVersion - parse version
func GetVersion(version string) (*types.Version, error) {

	v, err := semver.NewVersion(version)
	if err != nil {
		if err == semver.ErrInvalidSemVer {
			return nil, ErrInvalidSemVer
		}
		return nil, err
	}

	return &types.Version{
		Major:      v.Major(),
		Minor:      v.Minor(),
		Patch:      v.Patch(),
		PreRelease: string(v.Prerelease()),
		Metadata:   v.Metadata(),
		Original:   v.Original(),
	}, nil
}

// GetVersionFromImageName - get version from image name
func GetVersionFromImageName(name string) (*types.Version, error) {
	parts := strings.Split(name, ":")
	if len(parts) > 1 {
		return GetVersion(parts[1])
	}

	return nil, ErrVersionTagMissing
}

// GetImageNameAndVersion - get name and version
func GetImageNameAndVersion(name string) (string, *types.Version, error) {
	parts := strings.Split(name, ":")
	if len(parts) > 0 {
		v, err := GetVersion(parts[1])
		if err != nil {
			return "", nil, err
		}

		return parts[0], v, nil
	}

	return "", nil, ErrVersionTagMissing
}

// NewAvailable - takes version and current tags. Checks whether there is a new version in the list of tags
// and returns it as well as newAvailable bool
func NewAvailable(current string, tags []string) (newVersion string, newAvailable bool, err error) {

	currentVersion, err := semver.NewVersion(current)
	if err != nil {
		return "", false, err
	}

	if len(tags) == 0 {
		return "", false, nil
	}

	var vs []*semver.Version
	for _, r := range tags {
		v, err := semver.NewVersion(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"tag":   r,
			}).Debug("failed to parse tag")
			continue

		}

		vs = append(vs, v)
	}

	if len(vs) == 0 {
		log.Debug("no versions available")
		return "", false, nil
	}

	sort.Sort(sort.Reverse(semver.Collection(vs)))

	if currentVersion.LessThan(vs[0]) {
		return vs[0].Original(), true, nil
	}
	return "", false, nil
}

// ShouldUpdate - checks whether update is needed
func ShouldUpdate(current *types.Version, new *types.Version, policy types.PolicyType) (bool, error) {
	if policy == types.PolicyTypeForce {
		return true, nil
	}

	currentVersion, err := semver.NewVersion(current.String())
	if err != nil {
		return false, fmt.Errorf("failed to parse current version: %s", err)
	}
	newVersion, err := semver.NewVersion(new.String())
	if err != nil {
		return false, fmt.Errorf("failed to parse new version: %s", err)
	}

	// new version is not higher than current - do nothing
	if !currentVersion.LessThan(newVersion) {
		return false, nil
	}

	switch policy {
	case types.PolicyTypeAll, types.PolicyTypeMajor:
		return true, nil
	case types.PolicyTypeMinor:
		return newVersion.Major() == currentVersion.Major(), nil
	case types.PolicyTypePatch:
		return newVersion.Major() == currentVersion.Major() && newVersion.Minor() == currentVersion.Minor(), nil
	}
	return false, nil
}
