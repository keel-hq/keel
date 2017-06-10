package version

import (
	// "strconv"
	"errors"
	"fmt"
	"strings"

	"github.com/rusenask/keel/types"
	// "github.com/Masterminds/semver"
	"github.com/coreos/go-semver/semver"
	// log "github.com/Sirupsen/logrus"
)

var ErrVersionTagMissing = errors.New("version tag is missing")

// GetVersion - parse version
func GetVersion(version string) (*types.Version, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}

	return &types.Version{
		Major:      v.Major,
		Minor:      v.Minor,
		Patch:      v.Patch,
		PreRelease: string(v.PreRelease),
		Metadata:   v.Metadata,
	}, nil
}

// GetVersionFromImageName - get version from image name
func GetVersionFromImageName(name string) (*types.Version, error) {
	parts := strings.Split(name, ":")
	if len(parts) > 0 {
		return GetVersion(parts[1])
	}

	return nil, ErrVersionTagMissing
}

// ShouldUpdate - checks whether update is needed
func ShouldUpdate(current *types.Version, new *types.Version, policy types.PolicyType) (bool, error) {
	currentVersion, err := semver.NewVersion(current.String())
	if err != nil {
		return false, fmt.Errorf("failed to parse current version: %s", err)
	}
	newVersion, err := semver.NewVersion(new.String())
	if err != nil {
		return false, fmt.Errorf("failed to parse new version: %s", err)
	}

	// new version is not higher than current - do nothing
	if !currentVersion.LessThan(*newVersion) {
		return false, nil
	}

	switch policy {
	case types.PolicyTypeAll:
		return true, nil
	case types.PolicyTypeMajor:
		return newVersion.Major > currentVersion.Major, nil
	case types.PolicyTypeMinor:
		return newVersion.Major == currentVersion.Major && newVersion.Minor > currentVersion.Minor, nil
	case types.PolicyTypePatch:
		return newVersion.Major == currentVersion.Major && newVersion.Minor == currentVersion.Minor && newVersion.Patch > currentVersion.Patch, nil
	}
	return false, nil
}
