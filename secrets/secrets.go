package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"

	v1 "k8s.io/api/core/v1"

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
	defaultDockerConfig   DockerCfg // default configuration supplied by optional environment variable
}

// NewGetter - create new default getter
func NewGetter(implementer kubernetes.Implementer, defaultDockerConfig DockerCfg) *DefaultGetter {

	// initialising empty configuration
	if defaultDockerConfig == nil {
		defaultDockerConfig = make(DockerCfg)
	}

	return &DefaultGetter{
		kubernetesImplementer: implementer,
		defaultDockerConfig:   defaultDockerConfig,
	}
}

// Get - get secret for tracked image
func (g *DefaultGetter) Get(image *types.TrackedImage) (*types.Credentials, error) {
	if image.Namespace == "" {
		return nil, ErrNamespaceNotSpecified
	}

	// checking in default creds
	creds, found := g.lookupDefaultDockerConfig(image)
	if found {
		return creds, nil
	}

	if len(image.Secrets) == 0 {
		return nil, ErrSecretsNotSpecified
	}

	return g.getCredentialsFromSecret(image)
}

func (g *DefaultGetter) lookupDefaultDockerConfig(image *types.TrackedImage) (*types.Credentials, bool) {
	return credentialsFromConfig(image, g.defaultDockerConfig)
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
	secretFound := false

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

		var dockerCfg DockerCfg

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
			secretFound = true
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
			secretFound = true

			dockerCfg, err = DecodeDockerCfgJson(secretDataBts)
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
			secretFound = true

		default:
			log.WithFields(log.Fields{
				"image":      image.Image.Repository(),
				"namespace":  image.Namespace,
				"secret_ref": secretRef,
				"type":       secret.Type,
			}).Warn("secrets.defaultGetter: supplied secret is not kubernetes.io/dockercfg, ignoring")
			continue
		}

		creds, found := credentialsFromConfig(image, dockerCfg)
		if found {
			return creds, nil
		} else {
			log.WithFields(log.Fields{
				"secret_ref": secretRef,
				"image":      image.Image.String(),
			}).Warn("secrets.defaultGetter: registry not found among secrets")
		}
	}

	if secretFound {
		log.WithFields(log.Fields{
			"namespace": image.Namespace,
			"provider":  image.Provider,
			"registry":  image.Image.Registry(),
			"image":     image.Image.Repository(),
			"secrets":   image.Secrets,
		}).Warn("secrets.defaultGetter.lookupSecrets: secret found but couldn't detect authentication for the desired registry")
	} else if len(image.Secrets) > 0 {
		log.WithFields(log.Fields{
			"namespace": image.Namespace,
			"provider":  image.Provider,
			"registry":  image.Image.Registry(),
			"image":     image.Image.Repository(),
			"secrets":   image.Secrets,
		}).Errorf("secrets.defaultGetter.lookupSecrets: docker credentials were not found among secrets, is secret in the namespace '%s'?", image.Namespace)
	}

	return credentials, nil
}

func credentialsFromConfig(image *types.TrackedImage, cfg DockerCfg) (*types.Credentials, bool) {
	credentials := &types.Credentials{}
	found := false

	imageRegistry := image.Image.Registry()

	// looking for our registry
	for registry, auth := range cfg {
		if registryMatches(imageRegistry, registry) {
			if auth.Username != "" && auth.Password != "" {
				credentials.Username = auth.Username
				credentials.Password = auth.Password
			} else if auth.Auth != "" {
				username, password, err := decodeBase64Secret(auth.Auth)
				if err != nil {
					log.WithFields(log.Fields{
						"image":     image.Image.Repository(),
						"namespace": image.Namespace,
						"registry":  registry,
						"error":     err,
					}).Error("secrets.defaultGetter: failed to decode auth secret")
					continue
				}
				credentials.Username = username
				credentials.Password = password
				found = true
			} else {
				log.WithFields(log.Fields{
					"image":     image.Image.Repository(),
					"namespace": image.Namespace,
					"registry":  registry,
				}).Warn("secrets.defaultGetter: secret doesn't have username, password and base64 encoded auth, skipping")
				continue
			}

			log.WithFields(log.Fields{
				"namespace": image.Namespace,
				"provider":  image.Provider,
				"registry":  image.Image.Registry(),
				"image":     image.Image.Repository(),
			}).Debug("secrets.defaultGetter: secret looked up successfully")

			return credentials, true
		}
	}
	return credentials, found
}

func decodeBase64Secret(authSecret string) (username, password string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(authSecret)
	if err != nil {
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected auth secret format")
	}

	return parts[0], parts[1], nil
}

func EncodeBase64Secret(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
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

func domainOnly(registry string) string {
	if strings.Contains(registry, ":") {
		return strings.Split(registry, ":")[0]
	}

	return registry
}

func decodeSecret(data []byte) (DockerCfg, error) {
	var cfg DockerCfg
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func DecodeDockerCfgJson(data []byte) (DockerCfg, error) {
	// var cfg DockerCfg
	var cfg DockerCfgJSON
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg.Auths, nil
}

func EncodeDockerCfgJson(cfg *DockerCfg) ([]byte, error) {
	return json.Marshal(&DockerCfgJSON{
		Auths: *cfg,
	})
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
