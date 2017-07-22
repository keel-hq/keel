package image

import (
	"strings"
)

// Reference is an opaque object that include identifier such as a name, tag, repository, registry, etc...
type Reference struct {
	named  Named
	tag    string
	scheme string // registry scheme, i.e. http, https
}

func (r Reference) String() string {
	return r.Name()
}

// Name returns the image's name. (ie: debian[:8.2])
func (r Reference) Name() string {
	return r.named.RemoteName() + r.tag
}

// ShortName returns the image's name (ie: debian)
func (r Reference) ShortName() string {
	return r.named.RemoteName()
}

// Tag returns the image's tag (or digest).
func (r Reference) Tag() string {
	if len(r.tag) > 1 {
		return r.tag[1:]
	}
	return ""
}

// Registry returns the image's registry. (ie: host[:port])
func (r Reference) Registry() string {
	return r.named.Hostname()
}

// Scheme returns registry's scheme. (ie: https)
func (r Reference) Scheme() string {
	return r.scheme
}

// Repository returns the image's repository. (ie: registry/name)
func (r Reference) Repository() string {
	return r.named.FullName()
}

// Remote returns the image's remote identifier. (ie: registry/name[:tag])
func (r Reference) Remote() string {
	return r.named.FullName() + r.tag
}

func clean(url string) (cleaned string, scheme string) {

	s := url

	if strings.HasPrefix(url, "http://") {
		scheme = "http"
		s = strings.Replace(url, "http://", "", 1)
	} else if strings.HasPrefix(url, "https://") {
		scheme = "https"
		s = strings.Replace(url, "https://", "", 1)
	}

	if scheme == "" {
		scheme = DefaultScheme
	}

	return s, scheme
}

// Parse returns a Reference from analyzing the given remote identifier.
func Parse(remote string) (*Reference, error) {

	cleaned, scheme := clean(remote)
	n, err := ParseNamed(cleaned)

	if err != nil {
		return nil, err
	}

	n = WithDefaultTag(n)

	var t string
	switch x := n.(type) {
	case Canonical:
		t = "@" + x.Digest().String()
	case NamedTagged:
		t = ":" + x.Tag()
	}

	return &Reference{named: n, tag: t, scheme: scheme}, nil
}

// ParseRepo - parses remote
// pretty much the same as Parse but better for testing
func ParseRepo(remote string) (*Repository, error) {

	cleaned, scheme := clean(remote)

	n, err := ParseNamed(cleaned)

	if err != nil {
		return nil, err
	}

	n = WithDefaultTag(n)

	var t string
	switch x := n.(type) {
	case Canonical:
		t = "@" + x.Digest().String()
	case NamedTagged:
		t = ":" + x.Tag()
	}

	ref := &Reference{named: n, tag: t, scheme: scheme}

	return &Repository{
		Name:       ref.Name(),
		Repository: ref.Repository(),
		Registry:   ref.Registry(),
		Remote:     ref.Remote(),
		ShortName:  ref.ShortName(),
		Tag:        ref.Tag(),
		Scheme:     ref.scheme,
	}, nil
}
