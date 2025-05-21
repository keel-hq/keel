package dockerhelper

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

const defaultCacheTTL = 30 * time.Minute

var helperBinary = os.Getenv("DOCKER_CREDENTIALS_HELPER")

type DockerSecret struct {
	Username string `json:"Username"`
	Secret   string `json:"Secret"`
}

type CredentialsHelper struct {
	executor executor
	cache    *ttlcache.Cache[string, *types.Credentials]
	path     string
	enabled  bool
}

type executor interface {
	Run(string, string) ([]byte, error)
}

type executorImpl struct{}

func init() {
	credentialshelper.RegisterCredentialsHelper("dockerhelper", New())
}

func (e *executorImpl) Run(path, input string) ([]byte, error) {
	cmd := exec.Command(path, "get")
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.WithError(err).Error("credentialshelper.dockerhelper: failed to get stdin pipe")
		return nil, err
	}
	stdin.Write([]byte(input))
	stdin.Close()
	secrets, err := cmd.Output()
	if err != nil {
		log.WithError(err).Error("credentialshelper.dockerhelper: failed to get credentials")
		return nil, err
	}
	return secrets, nil
}

// New creates a new docker credentials helper.
func New() *CredentialsHelper {
	ch := &CredentialsHelper{}
	if helperBinary == "" {
		return ch
	}
	// Look up the binary at DOCKER_CREDENTIALS_HELPER.
	path, err := exec.LookPath(helperBinary)
	if err != nil {
		log.WithError(err).Error("credentialshelper.dockerhelper: failed to find DOCKER_CREDENTIALS_HELPER")
		return ch
	}
	ch.path = path
	ch.executor = &executorImpl{}
	ch.cache = ttlcache.New(
		ttlcache.WithTTL[string, *types.Credentials](defaultCacheTTL),
		ttlcache.WithDisableTouchOnHit[string, *types.Credentials](),
	)
	ch.enabled = true
	return ch
}

// IsEnabled returns a bool whether this credentials helper has been initialized.
func (h *CredentialsHelper) IsEnabled() bool {
	return h.enabled
}

// GetCredentials - finds credentials.
func (h *CredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	if !h.enabled {
		return nil, fmt.Errorf("not initialized")
	}
	registry := image.Image.Registry()
	creds := h.cache.Get(registry)
	if creds != nil {
		log.WithField("registry", registry).Debug("credentialshelper.dockerhelper: cache hit")
		return creds.Value(), nil
	}
	// Run the credentials helper at h.path and pass the registry via stdin.
	secrets, err := h.executor.Run(h.path, registry)
	if err != nil {
		return nil, err
	}
	dockerSecret := DockerSecret{}
	if err = json.Unmarshal(secrets, &dockerSecret); err != nil {
		log.WithError(err).Error("credentialshelper.dockerhelper: failed to unmarshal credentials")
		return nil, err
	}
	crds := &types.Credentials{
		Username: dockerSecret.Username,
		Password: dockerSecret.Secret,
	}
	log.WithField("registry", registry).Debug("credentialshelper.dockerhelper: cache miss")
	h.cache.Set(registry, crds, 0)
	return crds, nil
}
