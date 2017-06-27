package formatter

import (
	// "fmt"
	// "strings"
	log "github.com/Sirupsen/logrus"
)

// Deployment - internal deployment, used to better represent keel related info
type Deployment struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

const (
	defaultDeploymentQuietFormat = "{{.Name}}"
	defaultDeploymentTableFormat = "table {{.Namespace}}\t{{.Name}}"

	DeploymentNamespaceHeader = "NAMESPACE"
	DeploymentNameHeader      = "NAME"
)

// NewDeploymentsFormat returns a format for use with a deployment Context
func NewDeploymentsFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultDeploymentQuietFormat
		}
		return defaultDeploymentTableFormat
	case RawFormatKey:
		if quiet {
			return `name: {{.Name}}`
		}
		return `name: {{.Name}}\n`
	}
	return Format(source)
}

// DeploymentWrite writes formatted deployments using the Context
func DeploymentWrite(ctx Context, Deployments []Deployment) error {
	render := func(format func(subContext subContext) error) error {
		for _, deployment := range Deployments {
			log.WithFields(log.Fields{
				"name":      deployment.Name,
				"namespace": deployment.Namespace,
			}).Info("formatting deployment")
			if err := format(&DeploymentContext{v: deployment}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(&DeploymentContext{}, render)
}

type DeploymentContext struct {
	HeaderContext
	v Deployment
}

func (c *DeploymentContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func (c *DeploymentContext) Namespace() string {
	c.AddHeader(DeploymentNamespaceHeader)
	return c.v.Namespace
}
func (c *DeploymentContext) Name() string {
	c.AddHeader(DeploymentNameHeader)
	return c.v.Name
}
