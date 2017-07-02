package version

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/rusenask/keel/types"
)

// ErrVersionTagMissing - tag missing error
var ErrVersionTagMissing = errors.New("version tag is missing")

// GetVersion - parse version
func GetVersion(version string) (*types.Version, error) {

	v, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}
	// TODO: probably make it customazible
	prefix := ""
	if strings.HasPrefix(version, "v") {
		prefix = "v"
	}

	return &types.Version{
		Major:      v.Major(),
		Minor:      v.Minor(),
		Patch:      v.Patch(),
		PreRelease: string(v.Prerelease()),
		Metadata:   v.Metadata(),
		Prefix:     prefix,
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
	case types.PolicyTypeAll:
		return true, nil
	case types.PolicyTypeMajor:
		return newVersion.Major() > currentVersion.Major(), nil
	case types.PolicyTypeMinor:
		return newVersion.Major() == currentVersion.Major() && newVersion.Minor() > currentVersion.Minor(), nil
	case types.PolicyTypePatch:
		return newVersion.Major() == currentVersion.Major() && newVersion.Minor() == currentVersion.Minor() && newVersion.Patch() > currentVersion.Patch(), nil
	}
	return false, nil
}
