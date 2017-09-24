# Kubernetes backed KV 

[![GoDoc](https://godoc.org/github.com/rusenask/k8s-kv/kv?status.svg)](https://godoc.org/github.com/rusenask/k8s-kv/kv)

Use Kubernetes config maps as key/value store! 
When to use k8s-kv:
* You have a simple application that has a need to store some configuration and you can't be bothered to set up EBS like volumes or use some fancy external KV store.
* You have a stateless application that suddenly got to store state and you are not into converting 
 it into full stateless app that will use a proper database.

When __not to__ use k8s-kv:
* You have a read/write heavy multi-node application (k8s-kv doesn't have cross-app locking).
* You want to store bigger values than 1MB. Even though k8s-kv uses compression for the data stored in bucket - it's wise to not try the limits. It's there because of the limit in Etcd. In this case use something else.


## Basics

Package API:

```
// Pyt key/value pair into the store
Put(key string, value []byte) error
// Get value of the specified key
Get(key string) (value []byte, err error)
// Delete key/value pair from the store
Delete(key string) error
// List all key/value pairs under specified prefix
List(prefix string) (data map[string][]byte, err error)
// Delete config map (results in deleted data)
Teardown() error
```

## Caveats

* Don't be silly, you can't put a lot of stuff here.

## Example

Usage example:

1. Get minikube or your favourite k8s environment running.

2. In your app you will probably want to use this: https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration

3. Get ConfigMaps interface and supply it to this lib:

```
package main

import (
	"fmt"

	"github.com/rusenask/k8s-kv/kv"

	"k8s.io/client-go/kubernetes"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// get ConfigMapInterface to access config maps in "default" namespace
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

	// getting acces to k8s-kv. "my-app" will become a label
	// for this config map, this way it's easier to manage configs 
	// "bucket1" will be config map's name and represent one entry in config maps list	
	kvdb, err := kv.New(impl, "my-app", "bucket1")
	if err != nil {
		panic(err)
	}

	// insert a key "foo" with value "hello kubernetes world"
	kvdb.Put("foo", []byte("hello kubernetes world"))

	// get value of key "foo"
	stored, _ := kvdb.Get("foo")

	fmt.Println(string(stored))
}
```