package docker

import "net/url"

func (r *Registry) Tags(repository string) (tags []string, err error) {
	url := r.url("/v2/%s/tags/list", repository)

	var response tagsResponse
	for {
		r.Logf("registry.tags url=%s repository=%s", url, repository)
		url, err = r.getPaginatedJSON(url, &response)
		switch err {
		case ErrNoMorePages:
			tags = append(tags, response.Tags...)
			return tags, nil
		case nil:
			tags = append(tags, response.Tags...)
			continue
		default:
			return nil, err
		}
	}
}

// TagsAfter lists the tags the registry returns after the given tag, using
// the `last` query parameter as a cursor. What "after" means depends on the
// registry: the distribution spec orders tags lexically, while some
// registries, ghcr.io among them, list them in push order.
func (r *Registry) TagsAfter(repository, after string) ([]string, error) {
	next := r.url("/v2/%s/tags/list?last=%s", repository, url.QueryEscape(after))

	var tags []string
	for {
		r.Logf("registry.tags url=%s repository=%s", next, repository)
		var response tagsResponse
		var err error
		next, err = r.getPaginatedJSON(next, &response)
		switch err {
		case ErrNoMorePages:
			return append(tags, response.Tags...), nil
		case nil:
			tags = append(tags, response.Tags...)
		default:
			return nil, err
		}
	}
}

// FirstTagsPage returns the first page of the repository's tag list.
func (r *Registry) FirstTagsPage(repository string) ([]string, error) {
	first := r.url("/v2/%s/tags/list", repository)
	r.Logf("registry.tags url=%s repository=%s", first, repository)

	var response tagsResponse
	_, err := r.getPaginatedJSON(first, &response)
	if err != nil && err != ErrNoMorePages {
		return nil, err
	}
	return response.Tags, nil
}
