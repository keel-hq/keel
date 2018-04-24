package aws

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	// "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/types"
)

func init() {
	credentialshelper.RegisterCredentialsHelper("aws", New())
}

// CredentialsHelper provides authorization to ECR.
// Authentication details: https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
// # Access Key ID
// AWS_ACCESS_KEY_ID=AKID
// AWS_ACCESS_KEY=AKID # only read if AWS_ACCESS_KEY_ID is not set.
// more on auth: https://stackoverflow.com/questions/41544554/how-to-run-aws-sdk-with-credentials-from-variables
type CredentialsHelper struct {
	enabled bool
}

// New creates a new instance of aws credentials helper
func New() *CredentialsHelper {
	ch := &CredentialsHelper{}
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_ACCESS_KEY") != "" {
		ch.enabled = true
	}

	return ch
}

// IsEnabled returns a bool whether this credentials helper is initialised or not
func (h *CredentialsHelper) IsEnabled() bool {
	return h.enabled
}

// GetCredentials - finds credentials
func (h *CredentialsHelper) GetCredentials(registry string) (*types.Credentials, error) {

	svc := ecr.New(session.New())

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
			fmt.Println(err.Error())
		}
		return nil, err
	}

	fmt.Println(result)

	for _, ad := range result.AuthorizationData {
		if *ad.ProxyEndpoint == registry {
			username, password, err := decodeBase64Secret(*ad.AuthorizationToken)
			if err != nil {
				return nil, fmt.Errorf("failed to decode authentication token: %s, error: %s", *ad.AuthorizationToken, err)
			}

			return &types.Credentials{
				Username: username,
				Password: password,
			}, nil
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
