package registry

import (
	"errors"
	"hash/fnv"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/keel-hq/keel/registry/docker"
	"github.com/keel-hq/keel/types"

	drc "github.com/rusenask/docker-registry-client/registry"
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
	Digests(opts Opts) ([]string, error)
	Platforms(opts Opts) ([]types.Platform, error)
}

// New - new registry client
func New() *DefaultClient {
	insecure := false
	if os.Getenv(EnvInsecure) == "true" {
		insecure = true
	}
	return &DefaultClient{
		mu:         &sync.Mutex{},
		registries: make(map[uint32]*docker.Registry),
		insecure:   insecure,
	}
}

// DefaultClient - default client implementation
type DefaultClient struct {
	// a map of registries to reuse for polling
	mu         *sync.Mutex
	registries map[uint32]*docker.Registry
	insecure   bool
}

// Opts - registry client opts. If username & password are not supplied
// it will try to authenticate as anonymous
type Opts struct {
	Registry, Name, Tag string
	Username, Password  string // if "" - anonymous

	// After, when set, makes Get list only the tags pushed after this tag
	// where the registry lists tags in push order. Registries that list tags
	// lexically get a full listing instead.
	After string
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

func (c *DefaultClient) getRegistryClient(registryAddress, username, password string) (*docker.Registry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var r *docker.Registry

	h := hash(registryAddress + username + password)
	r, ok := c.registries[h]
	if ok {
		return r, nil
	}

	url := strings.TrimSuffix(registryAddress, "/")
	if os.Getenv(EnvInsecure) == "true" {
		r = docker.NewInsecure(url, username, password)
	} else {
		r = docker.New(url, username, password)
	}

	r.Logf = LogFormatter

	c.registries[h] = r

	return r, nil
}

// Get - get repository
func (c *DefaultClient) Get(opts Opts) (*Repository, error) {

	// fallback to HTTP if the registry doesn't speak HTTPS https://github.com/keel-hq/keel/issues/331
INIT_CLIENT:
	hub, err := c.getRegistryClient(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return nil, err
	}

	var tags []string
	if opts.After != "" {
		tags, err = tagsAfter(hub, opts.Name, opts.After)
	} else {
		tags, err = hub.Tags(opts.Name)
	}
	if err != nil {
		if strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") && strings.HasPrefix(opts.Registry, "https://") && c.insecure {
			opts.Registry = strings.Replace(opts.Registry, "https://", "http://", 1)
			goto INIT_CLIENT
		}
		return nil, err
	}
	repo := &Repository{
		Tags: tags,
	}

	return repo, nil
}

// tagsAfter lists the tags pushed after the given tag. The `last` cursor only
// means "pushed after" on registries that list tags in push order, so the
// cursor result is used only when the listing order is proven; otherwise
// every tag is listed, which is what Get does without a cursor.
func tagsAfter(hub *docker.Registry, name, after string) ([]string, error) {
	tags, err := hub.TagsAfter(name, after)
	if err != nil {
		return nil, err
	}

	if len(tags) > 0 {
		// A lexically ordered listing only returns tags that sort after
		// the cursor, so any tag sorting before it proves push order.
		for _, tag := range tags {
			if tag < after {
				return tags, nil
			}
		}
		log.WithFields(log.Fields{
			"repository": name,
			"after":      after,
		}).Info("registry.Get: registry appears to list tags lexically, listing all tags")
		return hub.Tags(name)
	}

	// An empty page means either nothing was pushed after the tag, or the
	// registry does not know the tag, or the tag sorts last in a lexical
	// listing. Only the first case means there is nothing newer.
	if _, err := hub.ManifestDigest(name, after); err != nil {
		var statusErr *drc.HttpStatusError
		if !errors.As(err, &statusErr) || statusErr.Response.StatusCode != http.StatusNotFound {
			return nil, err
		}
		log.WithFields(log.Fields{
			"repository": name,
			"after":      after,
		}).Info("registry.Get: current tag not found in registry, listing all tags")
		return hub.Tags(name)
	}

	first, err := hub.FirstTagsPage(name)
	if err != nil {
		return nil, err
	}
	if sort.StringsAreSorted(first) {
		log.WithFields(log.Fields{
			"repository": name,
			"after":      after,
		}).Info("registry.Get: registry appears to list tags lexically, listing all tags")
		return hub.Tags(name)
	}
	return nil, nil
}

// Digest - get digest for repo
func (c *DefaultClient) Digest(opts Opts) (string, error) {
	if opts.Tag == "" {
		return "", ErrTagNotSupplied
	}

	// fallback to HTTP if the registry doesn't speak HTTPS https://github.com/keel-hq/keel/issues/331
INIT_CLIENT:
	hub, err := c.getRegistryClient(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return "", err
	}

	manifestDigest, err := hub.ManifestDigest(opts.Name, opts.Tag)
	if err != nil {
		if strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") && strings.HasPrefix(opts.Registry, "https://") && c.insecure {
			opts.Registry = strings.Replace(opts.Registry, "https://", "http://", 1)
			goto INIT_CLIENT
		}
		return "", err
	}

	return manifestDigest.String(), nil
}

// Digests returns every digest that identifies a tag: the digest reported for
// the tag plus, for a multi-platform image, the digests of the per-platform
// manifests a node can actually be running.
func (c *DefaultClient) Digests(opts Opts) ([]string, error) {
	if opts.Tag == "" {
		return nil, ErrTagNotSupplied
	}

	// fallback to HTTP if the registry doesn't speak HTTPS https://github.com/keel-hq/keel/issues/331
INIT_CLIENT:
	hub, err := c.getRegistryClient(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return nil, err
	}

	digests, err := hub.ManifestDigests(opts.Name, opts.Tag)
	if err != nil {
		if strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") && strings.HasPrefix(opts.Registry, "https://") && c.insecure {
			opts.Registry = strings.Replace(opts.Registry, "https://", "http://", 1)
			goto INIT_CLIENT
		}
		return nil, err
	}

	return digests, nil
}

// Platforms returns the platforms supported by a tagged image manifest.
func (c *DefaultClient) Platforms(opts Opts) ([]types.Platform, error) {
	if opts.Tag == "" {
		return nil, ErrTagNotSupplied
	}

INIT_CLIENT:
	hub, err := c.getRegistryClient(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return nil, err
	}

	platforms, err := hub.ManifestPlatforms(opts.Name, opts.Tag)
	if err != nil {
		if strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") && strings.HasPrefix(opts.Registry, "https://") && c.insecure {
			opts.Registry = strings.Replace(opts.Registry, "https://", "http://", 1)
			goto INIT_CLIENT
		}
		return nil, err
	}
	return platforms, nil
}
