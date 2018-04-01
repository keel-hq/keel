package registry

import (
	"errors"
	"hash/fnv"
	"os"
	"strings"
	"sync"

	"github.com/rusenask/docker-registry-client/registry"

	log "github.com/sirupsen/logrus"
)

// EnvInsecure - uses insecure registry client to skip cert verification
const EnvInsecure = "INSECURE_REGISTRY"

// errors
var (
	ErrTagNotSupplied = errors.New("tag not supplied")
)

// Repository - holds repository related info
type Repository struct {
	Name string
	Tags []string // available tags
}

// Client - generic docker registry client
type Client interface {
	Get(opts Opts) (*Repository, error)
	Digest(opts Opts) (string, error)
}

// New - new registry client
func New() *DefaultClient {
	return &DefaultClient{
		mu:         &sync.Mutex{},
		registries: make(map[uint32]*registry.Registry),
	}
}

// DefaultClient - default client implementation
type DefaultClient struct {
	// a map of registries to reuse for polling
	mu         *sync.Mutex
	registries map[uint32]*registry.Registry
}

// Opts - registry client opts. If username & password are not supplied
// it will try to authenticate as anonymous
type Opts struct {
	Registry, Name, Tag string
	Username, Password  string // if "" - anonymous
}

// LogFormatter - formatter callback passed into registry client
func LogFormatter(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func (c *DefaultClient) getRegistryClient(registryAddress, username, password string) (*registry.Registry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var r *registry.Registry

	h := hash(registryAddress + username + password)
	r, ok := c.registries[h]
	if ok {
		return r, nil
	}

	url := strings.TrimSuffix(registryAddress, "/")
	if os.Getenv(EnvInsecure) == "true" {
		r = registry.NewInsecure(url, username, password)
	} else {
		r = registry.New(url, username, password)
	}

	r.Logf = LogFormatter

	c.registries[h] = r

	return r, nil
}

// Get - get repository
func (c *DefaultClient) Get(opts Opts) (*Repository, error) {

	hub, err := c.getRegistryClient(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return nil, err
	}

	tags, err := hub.Tags(opts.Name)
	if err != nil {
		return nil, err
	}
	repo := &Repository{
		Tags: tags,
	}

	return repo, nil
}

// Digest - get digest for repo
func (c *DefaultClient) Digest(opts Opts) (string, error) {
	if opts.Tag == "" {
		return "", ErrTagNotSupplied
	}

	hub, err := c.getRegistryClient(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return "", err
	}

	manifestDigest, err := hub.ManifestDigest(opts.Name, opts.Tag)
	if err != nil {
		return "", err
	}

	return manifestDigest.String(), nil
}
