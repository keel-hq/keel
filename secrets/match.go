package secrets

import (
	"net/url"
	"strings"
)

func registryMatches(imageRegistry, secretRegistry string) bool {

	if imageRegistry == secretRegistry {
		return true
	}

	imageRegistry = stripScheme(imageRegistry)
	secretRegistry = stripScheme(secretRegistry)

	if imageRegistry == secretRegistry {
		return true
	}

	// checking domains only
	if domainOnly(imageRegistry) == domainOnly(secretRegistry) {
		return true
	}

	// stripping any paths
	irh, err := url.Parse("https://" + imageRegistry)
	if err != nil {
		return false
	}
	srh, err := url.Parse("https://" + secretRegistry)
	if err != nil {
		return false
	}

	if irh.Hostname() == srh.Hostname() {
		return true
	}

	return false
}

func stripScheme(url string) string {

	if strings.HasPrefix(url, "http://") {
		return strings.TrimPrefix(url, "http://")
	}
	if strings.HasPrefix(url, "https://") {
		return strings.TrimPrefix(url, "https://")
	}
	return url
}
