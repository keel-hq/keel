package main

import (
	"fmt"

	"github.com/rusenask/k8s-kv/kv"

	"k8s.io/client-go/kubernetes"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func getImplementer() (implementer core_v1.ConfigMapInterface) {
	cfg, err := clientcmd.BuildConfigFromFlags("", ".kubeconfig") // in your app you could replace it with in-cluster-config
	if err != nil {
		panic(err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	return client.ConfigMaps("default")
}

func main() {
	impl := getImplementer()

	kvdb, err := kv.New(impl, "my-app", "bucket1")
	if err != nil {
		panic(err)
	}

	kvdb.Put("foo", []byte("hello kubernetes world"))

	stored, _ := kvdb.Get("foo")

	fmt.Println(string(stored))
}
