package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const keelForwardURL = "http://127.0.0.1:19300"

type E2ESuite struct {
	suite.Suite
	cfg           e2eConfig
	client        *kubernetes.Clientset
	portForward   *exec.Cmd
	testNamespace string
}

func TestE2ESuite(t *testing.T) {
	if os.Getenv("KEEL_E2E_RUN_ID") == "" {
		t.Skip("run through make e2e")
	}
	suite.Run(t, new(E2ESuite))
}

func (s *E2ESuite) SetupSuite() {
	var err error
	s.cfg, err = loadE2EConfig()
	require.NoError(s.T(), err)

	restConfig, err := clientcmd.BuildConfigFromFlags("", s.cfg.kubeconfig)
	require.NoError(s.T(), err)
	s.client, err = kubernetes.NewForConfig(restConfig)
	require.NoError(s.T(), err)
	require.NoError(s.T(), createKeelResources(context.Background(), s.client, s.cfg))

	deploymentCtx, deploymentCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer deploymentCancel()
	require.NoError(s.T(), waitForDeploymentAvailable(deploymentCtx, s.client, s.cfg.systemNamespace(), "keel"))

	logPath := filepath.Join(s.cfg.artifactDir, "port-forward.log")
	logFile, err := os.Create(logPath)
	require.NoError(s.T(), err)
	s.portForward = exec.Command(s.cfg.kubectl, "kubectl", "--kubeconfig", s.cfg.kubeconfig,
		"--namespace", s.cfg.systemNamespace(), "port-forward", "service/keel", "19300:9300")
	s.portForward.Env = append(os.Environ(), "K3S_DATA_DIR="+s.cfg.k3sDataDir)
	s.portForward.Stdout = logFile
	s.portForward.Stderr = logFile
	require.NoError(s.T(), s.portForward.Start())
	require.NoError(s.T(), os.WriteFile(filepath.Join(filepath.Dir(s.cfg.kubeconfig), "port-forward.pid"),
		[]byte(fmt.Sprintf("%d\n", s.portForward.Process.Pid)), 0o600))

	httpClient := &http.Client{Timeout: 2 * time.Second}
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer healthCancel()
	require.NoError(s.T(), waitForHTTPReady(healthCtx, httpClient, keelForwardURL+"/healthz"))
}

func (s *E2ESuite) TearDownSuite() {
	if s.portForward != nil && s.portForward.Process != nil {
		_ = s.portForward.Process.Signal(os.Interrupt)
		_, _ = s.portForward.Process.Wait()
	}
	if s.client == nil || s.cfg.runID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_ = s.client.RbacV1().ClusterRoleBindings().Delete(ctx, "keel-e2e-"+s.cfg.runID, metav1.DeleteOptions{})
	_ = s.client.RbacV1().ClusterRoles().Delete(ctx, "keel-e2e-"+s.cfg.runID, metav1.DeleteOptions{})
	_ = s.client.CoreV1().Namespaces().Delete(ctx, s.cfg.systemNamespace(), metav1.DeleteOptions{})
	require.Eventually(s.T(), func() bool {
		_, err := s.client.CoreV1().Namespaces().Get(context.Background(), s.cfg.systemNamespace(), metav1.GetOptions{})
		return err != nil
	}, 60*time.Second, time.Second, "Keel suite namespace was not deleted")
}
