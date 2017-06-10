package version

import (
	// "strconv"
	"errors"
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
