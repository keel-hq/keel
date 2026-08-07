package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var unsetenv = os.Unsetenv

func TestLoadDefaults(t *testing.T) {
	for _, name := range []string{
		"PUBSUB",
		"POLL",
		"PROJECT_ID",
		"CLUSTER_NAME",
		"XDG_DATA_HOME",
		"HELM3_PROVIDER",
		"UI_DIR",
	} {
		t.Setenv(name, "")
		require.NoError(t, unsetenv(name))
	}

	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.TriggerPubSub)
	require.True(t, cfg.TriggerPoll)
	require.Equal(t, "/data", cfg.DataDir)
	require.False(t, cfg.Helm3Provider)
	require.Equal(t, "www", cfg.UIDir)
}

func TestLoad(t *testing.T) {
	t.Setenv("PUBSUB", "true")
	t.Setenv("POLL", "false")
	t.Setenv("PROJECT_ID", "project")
	t.Setenv("CLUSTER_NAME", "cluster")
	t.Setenv("XDG_DATA_HOME", "/var/lib/keel")
	t.Setenv("HELM3_PROVIDER", "1")
	t.Setenv("UI_DIR", "/opt/keel/ui")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, Config{
		TriggerPubSub: true,
		TriggerPoll:   false,
		ProjectID:     "project",
		ClusterName:   "cluster",
		DataDir:       "/var/lib/keel",
		Helm3Provider: true,
		UIDir:         "/opt/keel/ui",
	}, cfg)
}
