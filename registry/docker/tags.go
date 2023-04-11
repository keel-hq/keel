package docker

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
