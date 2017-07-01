package registry

import (
	"errors"

	// "github.com/docker/distribution/digest"
	// "github.com/docker/distribution/manifest"
	// "github.com/docker/libtrust"
	"github.com/rusenask/docker-registry-client/registry"
)

// errors
var (
	ErrTagNotSupplied = errors.New("tag not supplied")
)

type Repository struct {
	Name string
	Tags []string
}

type Client interface {
}

func New() *DefaultClient {
	return &DefaultClient{}
}

type DefaultClient struct {
}

type Opts struct {
	Registry, Name, Tag string
	Username, Password  string // if "" - anonymous
}

// Get - get repository
func (c *DefaultClient) Get(opts Opts) (*Repository, error) {

	repo := &Repository{}
	hub, err := registry.New(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return nil, err
	}

	tags, err := hub.Tags(opts.Name)
	if err != nil {
		return nil, err
	}
	repo.Tags = tags

	return repo, nil
}

// Digest - get digest for repo
func (c *DefaultClient) Digest(opts Opts) (digest string, err error) {
	if opts.Tag == "" {
		return "", ErrTagNotSupplied
	}

	hub, err := registry.New(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return
	}

	manifestDigest, err := hub.ManifestDigest(opts.Name, opts.Tag)
	if err != nil {
		return
	}

	return manifestDigest.String(), nil
}
