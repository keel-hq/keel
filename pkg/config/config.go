package config

import "github.com/kelseyhightower/envconfig"

// Config contains Keel's application configuration loaded from environment variables.
type Config struct {
	TriggerPubSub bool   `envconfig:"PUBSUB" default:"false"`
	TriggerPoll   bool   `envconfig:"POLL" default:"true"`
	ProjectID     string `envconfig:"PROJECT_ID"`
	ClusterName   string `envconfig:"CLUSTER_NAME"`
	DataDir       string `envconfig:"XDG_DATA_HOME" default:"/data"`
	Helm3Provider bool   `envconfig:"HELM3_PROVIDER" default:"false"`
	UIDir         string `envconfig:"UI_DIR" default:"www"`
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
