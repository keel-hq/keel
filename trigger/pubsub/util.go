package pubsub

import (
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// MetadataEndpoint - default metadata server for gcloud pubsub
const MetadataEndpoint = "http://metadata/computeMetadata/v1/instance/attributes/cluster-name"

func containerRegistryURI(projectID, registry string) string {
	return registry + "%2F" + projectID
}

func containerRegistrySubName(projectID, topic string) string {
	cluster := "unknown"
	clusterName, err := clusterName(MetadataEndpoint)
	if err != nil {
		log.WithFields(log.Fields{
			"error":             err,
			"metadata_endpoint": MetadataEndpoint,
		}).Warn("trigger.pubsub.containerRegistrySubName: got error while retrieving cluster metadata, messages might be lost if more than one Keel instance is created")
	} else {
		cluster = clusterName
	}

	return "keel-" + cluster + "-" + projectID + "-" + topic
}

// https://cloud.google.com/compute/docs/storing-retrieving-metadata
func clusterName(metadataEndpoint string) (string, error) {
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// isGoogleContainerRegistry - we only care about gcr.io images,
// with other registries - we won't be able to receive events.
// Theoretically if someone publishes messages for updated images to
// google pubsub - we could turn this off
func isGoogleContainerRegistry(registry string) bool {
	return strings.Contains(registry, "gcr.io")
}
