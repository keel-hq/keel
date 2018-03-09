package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/keel-hq/keel/provider/helm"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"

	"k8s.io/api/core/v1"

	log "github.com/sirupsen/logrus"
)

// const dockerConfigJSONKey = ".dockerconfigjson"
const dockerConfigKey = ".dockercfg"

const dockerConfigJSONKey = ".dockerconfigjson"

// common errors
var (
	ErrNamespaceNotSpecified = errors.New("namespace not specified")
	ErrSecretsNotSpecified   = errors.New("no secrets were specified")
)

// Getter - generic secret getter interface
type Getter interface {
	Get(image *types.TrackedImage) (*types.Credentials, error)
}

// DefaultGetter - default kubernetes secret getter implementation
type DefaultGetter struct {
	kubernetesImplementer kubernetes.Implementer
}

// NewGetter - create new default getter
func NewGetter(implementer kubernetes.Implementer) *DefaultGetter {
	return &DefaultGetter{
		kubernetesImplementer: implementer,
	}
}

// Get - get secret for tracked image
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
		}).Debug("secrets.defaultGetter.lookupSecrets: pod secrets found")
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
		}).Debug("secrets.defaultGetter.lookupSecrets: no secrets for image found")
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

		dockerCfg := make(DockerCfg)

		switch secret.Type {
		case v1.SecretTypeDockercfg:
			secretDataBts, ok := secret.Data[dockerConfigKey]
			if !ok {
				log.WithFields(log.Fields{
					"image":      image.Image.Repository(),
					"namespace":  image.Namespace,
					"secret_ref": secretRef,
					"type":       secret.Type,
					"data":       secret.Data,
				}).Warn("secrets.defaultGetter: secret is missing key '.dockerconfig', ensure that key exists")
				continue
			}
			dockerCfg, err = decodeSecret(secretDataBts)
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
		case v1.SecretTypeDockerConfigJson:
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

			dockerCfg, err = decodeJSONSecret(secretDataBts)
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

		default:
			log.WithFields(log.Fields{
				"image":      image.Image.Repository(),
				"namespace":  image.Namespace,
				"secret_ref": secretRef,
				"type":       secret.Type,
			}).Warn("secrets.defaultGetter: supplied secret is not kubernetes.io/dockercfg, ignoring")
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
				}).Debug("secrets.defaultGetter: secret looked up successfully")

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
	if strings.HasPrefix(registry, "http://") || strings.HasPrefix(registry, "https://") {
		u, err := url.Parse(registry)
		if err != nil {
			return "", err
		}
		return u.Hostname(), nil
	}

	return registry, nil
}

func decodeSecret(data []byte) (DockerCfg, error) {
	var cfg DockerCfg
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func decodeJSONSecret(data []byte) (DockerCfg, error) {
	var cfg DockerCfgJSON
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg.Auths, nil
}

// DockerCfgJSON - secret structure when dockerconfigjson is used
type DockerCfgJSON struct {
	Auths DockerCfg `json:"auths"`
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
