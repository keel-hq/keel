package tests

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	log "github.com/sirupsen/logrus"
)

var kubeConfig = flag.String("kubeconfig", "", "Path to Kubernetes config file")

func getKubeConfig() string {
	if *kubeConfig != "" {
		return *kubeConfig
	}

	return filepath.Join(os.Getenv("HOME"), ".kube", "config")
}

func getKubernetesClient() (*rest.Config, *kubernetes.Clientset) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfig())
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return config, clientSet
}

func createNamespaceForTest() string {
	_, clientset := getKubernetesClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "keel-e2e-test-",
		},
	}
	cns, err := clientset.CoreV1().Namespaces().Create(ns)
	if err != nil {
		panic(err)
	}

	log.Infof("test namespace '%s' created", cns.Name)

	return cns.Name
}

func deleteTestNamespace(namespace string) error {

	defer log.Infof("test namespace '%s' deleted", namespace)
	_, clientset := getKubernetesClient()
	deleteOptions := metav1.DeleteOptions{}
	return clientset.CoreV1().Namespaces().Delete(namespace, &deleteOptions)
}

func startKeel(ctx context.Context) error {

	log.Info("keel started")
	defer log.Info("keel stopped")

	cmd := "keel"
	args := []string{"--no-incluster", "--kubeconfig", getKubeConfig()}
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	go func() {
		<-ctx.Done()
		err := c.Process.Kill()
		if err != nil {
			log.Errorf("failed to kill keel process: %s", err)
		}
	}()

	return c.Run()
}
