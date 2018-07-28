package aws

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
	"regexp"

	// "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// AWSCredentialsExpiry specifies how long can we keep cached AWS credentials.
// This is required to reduce chance of hiting rate limits,
// more info here: https://docs.aws.amazon.com/AmazonECR/latest/userguide/service_limits.html
const AWSCredentialsExpiry = 2 * time.Hour
var registryRegxp *regexp.Regexp

func init() {
	credentialshelper.RegisterCredentialsHelper("aws", New())
	registryRegxp = regexp.MustCompile(`(?P<registryID>\d+)\.dkr\.ecr\.(?P<region>\S+)\.amazonaws\.com`)
}

// CredentialsHelper provides authorization to ECR.
// Authentication details: https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
// # Access Key ID
// AWS_ACCESS_KEY_ID=AKID
// AWS_ACCESS_KEY=AKID # only read if AWS_ACCESS_KEY_ID is not set.
// more on auth: https://stackoverflow.com/questions/41544554/how-to-run-aws-sdk-with-credentials-from-variables
type CredentialsHelper struct {
	enabled bool
	cache   *Cache
}

// New creates a new instance of aws credentials helper
func New() *CredentialsHelper {
	ch := &CredentialsHelper{}
	ch.enabled = true
	log.Infof("extension.credentialshelper.aws: enabled")
	ch.cache = NewCache(AWSCredentialsExpiry)

	return ch
}

// IsEnabled returns a bool whether this credentials helper is initialised or not
func (h *CredentialsHelper) IsEnabled() bool {
	return h.enabled
}

// GetCredentials - finds credentials
func (h *CredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {

	if !h.enabled {
		return nil, fmt.Errorf("not initialised")
	}

	registry := image.Image.Registry()

	_, region, err := parseRegistry(registry)
	if err != nil {
		return nil, err
	}

	cached, err := h.cache.Get(registry)
	if err == nil {
		return cached, nil
	}
	// fetch region from registry instead of env
	svc := ecr.New(session.New(), &aws.Config{
		Region: aws.String(region),
	})

	input := &ecr.GetAuthorizationTokenInput{}

	result, err := svc.GetAuthorizationToken(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				fmt.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				fmt.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.WithFields(log.Fields{
				"error": err,
			}).Error("credentialshelper.aws: failed to get authorization token")
		}
		return nil, err
	}

	for _, ad := range result.AuthorizationData {

		u, err := url.Parse(*ad.ProxyEndpoint)
		if err != nil {
			log.WithError(err).Errorf("credentialshelper.aws: failed to parse registry endpoint: %s", *ad.ProxyEndpoint)
			continue
		}

		log.WithFields(log.Fields{
			"current_registry": u.Host,
			"token":            *ad.AuthorizationToken,
			"registry":         registry,
		}).Debug("checking registry")
		if u.Host == registry {
			username, password, err := decodeBase64Secret(*ad.AuthorizationToken)
			if err != nil {
				return nil, fmt.Errorf("failed to decode authentication token: %s, error: %s", *ad.AuthorizationToken, err)
			}

			creds := &types.Credentials{
				Username: username,
				Password: password,
			}

			h.cache.Put(registry, creds)

			return creds, nil
		}
	}

	return nil, fmt.Errorf("not found")
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

func parseRegistry(registry string) (registryID string, region string, err error) {
	if !registryRegxp.MatchString(registry) {
		err = credentialshelper.ErrUnsupportedRegistry
		return
	}
	// parse registry with named regex, then put into map by name
	matches := registryRegxp.FindStringSubmatch(registry)
	registryParsed := make(map[string]string)
	for i, name := range registryRegxp.SubexpNames() {
		if i != 0 && name != "" {
			registryParsed[name] = matches[i]
		}
	}

	return registryParsed["registryID"], registryParsed["region"], nil
}

