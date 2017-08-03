package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/rusenask/keel/provider/helm"
	"github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/types"

	"k8s.io/client-go/pkg/api/v1"

	log "github.com/Sirupsen/logrus"
)

// const dockerConfigJSONKey = ".dockerconfigjson"
const dockerConfigJSONKey = ".dockercfg"

var (
	ErrNamespaceNotSpecified = errors.New("namespace not specified")
	ErrSecretsNotSpecified   = errors.New("no secrets were specified")
)

type Getter interface {
	Get(image *types.TrackedImage) (*types.Credentials, error)
}

type DefaultGetter struct {
	kubernetesImplementer kubernetes.Implementer
}

func NewGetter(implementer kubernetes.Implementer) *DefaultGetter {
	return &DefaultGetter{
		kubernetesImplementer: implementer,
	}
}

func (g *DefaultGetter) Get(image *types.TrackedImage) (*types.Credentials, error) {
	if image.Namespace == "" {
		return nil, ErrNamespaceNotSpecified
	}

	switch image.Provider {
	case helm.ProviderName:
		// looking up secrets based on selector
		secrets, err := g.lookupSecrets(image)
		if err != nil {
			return nil, err
		}

		// populating secrets
		image.Secrets = secrets
	}

	return g.getCredentialsFromSecret(image)
}

func (g *DefaultGetter) lookupSecrets(image *types.TrackedImage) ([]string, error) {
	secrets := []string{}

	selector, ok := image.Meta["selector"]
	if !ok {
		// nothing
		return secrets, nil
	}

	podList, err := g.kubernetesImplementer.Pods(image.Namespace, selector)
	if err != nil {
		return secrets, err
	}

	for _, pod := range podList.Items {
		podSecrets := getPodImagePullSecrets(&pod)
		log.WithFields(log.Fields{
			"namespace":    image.Namespace,
			"provider":     image.Provider,
			"registry":     image.Image.Registry(),
			"image":        image.Image.Repository(),
			"pod_selector": selector,
			"secrets":      podSecrets,
		}).Info("secrets.defaultGetter.lookupSecrets: pod secrets found")
		secrets = append(secrets, podSecrets...)
	}

	if len(secrets) == 0 {
		log.WithFields(log.Fields{
			"namespace":    image.Namespace,
			"provider":     image.Provider,
			"registry":     image.Image.Registry(),
			"image":        image.Image.Repository(),
			"pod_selector": selector,
			"pods_checked": len(podList.Items),
		}).Info("secrets.defaultGetter.lookupSecrets: no secrets for image found")
	}

	return secrets, nil
}

func getPodImagePullSecrets(pod *v1.Pod) []string {
	var secrets []string
	for _, s := range pod.Spec.ImagePullSecrets {
		secrets = append(secrets, s.Name)
	}
	return secrets
}

func (g *DefaultGetter) getCredentialsFromSecret(image *types.TrackedImage) (*types.Credentials, error) {

	credentials := &types.Credentials{}

	for _, secretRef := range image.Secrets {
		secret, err := g.kubernetesImplementer.Secret(image.Namespace, secretRef)
		if err != nil {
			log.WithFields(log.Fields{
				"image":      image.Image.Repository(),
				"namespace":  image.Namespace,
				"secret_ref": secretRef,
				"error":      err,
			}).Warn("secrets.defaultGetter: failed to get secret")
			continue
		}

		if secret.Type != v1.SecretTypeDockercfg {
			log.WithFields(log.Fields{
				"image":      image.Image.Repository(),
				"namespace":  image.Namespace,
				"secret_ref": secretRef,
				"type":       secret.Type,
			}).Warn("secrets.defaultGetter: supplied secret is not kubernetes.io/dockercfg, ignoring")
			continue
		}

		secretDataBts, ok := secret.Data[dockerConfigJSONKey]
		if !ok {
			log.WithFields(log.Fields{
				"image":      image.Image.Repository(),
				"namespace":  image.Namespace,
				"secret_ref": secretRef,
				"type":       secret.Type,
				"data":       secret.Data,
			}).Warn("secrets.defaultGetter: secret is missing key '.dockerconfigjson', ensure that key exists")
			continue
		}
		dockerCfg, err := decodeSecret(secretDataBts)
		if err != nil {
			log.WithFields(log.Fields{
				"image":       image.Image.Repository(),
				"namespace":   image.Namespace,
				"secret_ref":  secretRef,
				"secret_data": string(secretDataBts),
				"error":       err,
			}).Error("secrets.defaultGetter: failed to decode secret")
			continue
		}

		// looking for our registry
		for registry, auth := range dockerCfg {
			h, err := hostname(registry)
			if err != nil {
				log.WithFields(log.Fields{
					"image":      image.Image.Repository(),
					"namespace":  image.Namespace,
					"registry":   registry,
					"secret_ref": secretRef,
					"error":      err,
				}).Error("secrets.defaultGetter: failed to parse hostname")
				continue
			}

			if h == image.Image.Registry() {
				if auth.Username != "" && auth.Password != "" {
					credentials.Username = auth.Username
					credentials.Password = auth.Password
				} else if auth.Auth != "" {
					username, password, err := decodeBase64Secret(auth.Auth)
					if err != nil {
						log.WithFields(log.Fields{
							"image":      image.Image.Repository(),
							"namespace":  image.Namespace,
							"registry":   registry,
							"secret_ref": secretRef,
							"error":      err,
						}).Error("secrets.defaultGetter: failed to decode auth secret")
						continue
					}
					credentials.Username = username
					credentials.Password = password
				} else {
					log.WithFields(log.Fields{
						"image":      image.Image.Repository(),
						"namespace":  image.Namespace,
						"registry":   registry,
						"secret_ref": secretRef,
						"error":      err,
					}).Warn("secrets.defaultGetter: secret doesn't have username, password and base64 encoded auth, skipping")
					continue
				}

				log.WithFields(log.Fields{
					"namespace": image.Namespace,
					"provider":  image.Provider,
					"registry":  image.Image.Registry(),
					"image":     image.Image.Repository(),
				}).Info("secrets.defaultGetter: secret looked up successfully")

				return credentials, nil
			}
		}
	}

	if len(image.Secrets) > 0 {
		log.WithFields(log.Fields{
			"namespace": image.Namespace,
			"provider":  image.Provider,
			"registry":  image.Image.Registry(),
			"image":     image.Image.Repository(),
			"secrets":   image.Secrets,
		}).Warn("secrets.defaultGetter.lookupSecrets: docker credentials were not found among secrets")
	}

	return credentials, nil
}

func decodeBase64Secret(authSecret string) (username, password string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(authSecret)
	if err != nil {
		return
	}

	parts := strings.Split(string(decoded), ":")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected auth secret format")
	}

	return parts[0], parts[1], nil
}

func hostname(registry string) (string, error) {
	u, err := url.Parse(registry)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

func decodeSecret(data []byte) (DockerCfg, error) {
	var cfg DockerCfg
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// DockerCfg - registry_name=auth
type DockerCfg map[string]*Auth

// Auth - auth
type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
}
