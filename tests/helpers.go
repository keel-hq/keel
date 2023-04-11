package tests

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		ObjectMeta: meta_v1.ObjectMeta{
			GenerateName: "keel-e2e-test-",
		},
	}
	createOptions := meta_v1.CreateOptions{}
	cns, err := clientset.CoreV1().Namespaces().Create(context.Background(), ns, createOptions)
	if err != nil {
		panic(err)
	}

	log.Infof("test namespace '%s' created", cns.Name)

	return cns.Name
}

func deleteTestNamespace(namespace string) error {

	defer log.Infof("test namespace '%s' deleted", namespace)
	_, clientset := getKubernetesClient()
	deleteOptions := meta_v1.DeleteOptions{}
	return clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, deleteOptions)
}

type KeelCmd struct {
	cmd *exec.Cmd

	env []string
}

func (kc *KeelCmd) Start(ctx context.Context) error {

	log.Info("keel started")

	cmd := "keel"
	args := []string{"--no-incluster", "--kubeconfig", getKubeConfig()}
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = []string{
		"DEBUG=true",
	}
	c.Env = append(c.Env, kc.env...)

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	kc.cmd = c

	err := c.Run()
	if err != nil {
		if strings.Contains(err.Error(), "killed") {
			return nil
		}
	}
	return err
}

func (kc *KeelCmd) Stop() error {
	defer log.Info("keel stopped")
	return kc.cmd.Process.Kill()
}

func waitFor(ctx context.Context, kcs *kubernetes.Clientset, namespace, name string, desired string) error {
	last := ""
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("expected '%s', got: '%s'", desired, last)
		default:
			updated, err := kcs.AppsV1().Deployments(namespace).Get(context.Background(), name, meta_v1.GetOptions{})
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			if updated.Spec.Template.Spec.Containers[0].Image != desired {
				time.Sleep(500 * time.Millisecond)
				last = updated.Spec.Template.Spec.Containers[0].Image
				continue
			}
			return nil
		}
	}
}
