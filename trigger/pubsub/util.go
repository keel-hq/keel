package pubsub

import (
	"io"
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"
)

// MetadataEndpoint - default metadata server for gcloud pubsub
const MetadataEndpoint = "http://metadata/computeMetadata/v1/instance/attributes/cluster-name"

func containerRegistrySubName(clusterName, projectID, topic string) string {

	if clusterName == "" {
		var err error
		clusterName, err = getClusterName(MetadataEndpoint)
		if err != nil {
			clusterName = "unknown"
			log.WithFields(log.Fields{
				"error":             err,
				"metadata_endpoint": MetadataEndpoint,
			}).Warn("trigger.pubsub.containerRegistrySubName: got error while retrieving cluster metadata, messages might be lost if more than one Keel instance is created")
		}
	}

	return "keel-" + clusterName + "-" + projectID + "-" + topic
}

// https://cloud.google.com/compute/docs/storing-retrieving-metadata
func getClusterName(metadataEndpoint string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, metadataEndpoint, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Metadata-Flavor", "Google")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// isGoogleArtifactRegistry - we only care about gcr.io and pkg.dev images,
// with other registries - we won't be able to receive events.
// Theoretically if someone publishes messages for updated images to
// google pubsub - we could turn this off
func isGoogleArtifactRegistry(registry string) bool {
	matched, err := regexp.MatchString(`(gcr\.io|pkg\.dev)`, registry)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("trigger.pubsub.isGoogleArtifactRegistry: got error while checking if registry is gcr")
		return false
	}
	return matched
}
