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

	var latest bool
	prefix := "" // TODO: probably make it customazible
	if version == "latest" {
		//don't update tags to a specific version if some are labeled as latest
		//i.e. deployment with tag "latest" -> gets tag "0.0.1" or other version
		version = "999.999.999" 
		latest = true
	}
	if strings.HasPrefix(version, "v") {
		prefix = "v"
	}
	v, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}
	return &types.Version{
		Major:      v.Major(),
		Minor:      v.Minor(),
		Patch:      v.Patch(),
		PreRelease: string(v.Prerelease()),
		Metadata:   v.Metadata(),
		Latest:   	latest,
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

	if policy == types.PolicyTypeLatest {
		if current.Latest {
			return true, nil
		}
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
