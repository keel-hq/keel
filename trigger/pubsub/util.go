package pubsub

import (
	"strings"
)

// "gcr.io/v2-namespace/hello-world:1.1"
func extractContainerRegistryURI(imageName string) string {
	parts := strings.Split(imageName, "/")
	return parts[0]
}

func containerRegistryURI(projectID, registry string) string {
	return registry + "%2F" + projectID
}

func containerRegistrySubName(projectID, topic string) string {
	return "keel-" + projectID + "-" + topic
}

func isGoogleContainerRegistry(registry string) bool {
	return strings.Contains(registry, "gcr.io")
}
