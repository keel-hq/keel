package tests

import (
	"fmt"
	"testing"

	"github.com/rusenask/k8s-kv/kv"

	"k8s.io/client-go/kubernetes"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const clusterConfig = ".kubeconfig"
const testingNamespace = "default"

func getImplementer(t *testing.T) (implementer core_v1.ConfigMapInterface) {
	cfg, err := clientcmd.BuildConfigFromFlags("", clusterConfig)
	if err != nil {
		t.Fatalf("failed to get config: %s", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	return client.ConfigMaps(testingNamespace)
}

func TestPut(t *testing.T) {

	impl := getImplementer(t)
	kv, err := kv.New(impl, "test", "testput")
	if err != nil {
		t.Fatalf("failed to create kv: %s", err)
	}
	defer kv.Teardown()

	err = kv.Put("key", []byte("val"))
	if err != nil {
		t.Errorf("failed to put: %s", err)
	}
}

func TestPutDirectoryKeys(t *testing.T) {
	impl := getImplementer(t)
	kv, err := kv.New(impl, "test", "testputdirectorykeys")
	if err != nil {
		t.Fatalf("failed to create kv: %s", err)
	}
	defer kv.Teardown()

	err = kv.Put("/somedir/key-here", []byte("val"))
	if err != nil {
		t.Errorf("failed to put: %s", err)
	}

	val, err := kv.Get("/somedir/key-here")
	if err != nil {
		t.Errorf("failed to get key: %s", err)
	}

	if string(val) != "val" {
		t.Errorf("unexpected return: %s", string(val))
	}
}

func TestGet(t *testing.T) {

	impl := getImplementer(t)
	kv, err := kv.New(impl, "test", "testget")
	if err != nil {
		t.Fatalf("failed to create kv: %s", err)
	}
	defer kv.Teardown()

	err = kv.Put("foo", []byte("bar"))
	if err != nil {
		t.Errorf("failed to put: %s", err)
	}

	// getting it back
	val, err := kv.Get("foo")
	if err != nil {
		t.Errorf("failed to get: %s", err)
	}

	if string(val) != "bar" {
		t.Errorf("expected 'bar' but got: '%s'", string(val))
	}

}

func TestDelete(t *testing.T) {

	impl := getImplementer(t)
	kvdb, err := kv.New(impl, "test", "testdelete")
	if err != nil {
		t.Fatalf("failed to create kv: %s", err)
	}
	defer kvdb.Teardown()

	err = kvdb.Put("foo", []byte("bar"))
	if err != nil {
		t.Errorf("failed to put: %s", err)
	}

	// getting it back
	val, err := kvdb.Get("foo")
	if err != nil {
		t.Errorf("failed to get: %s", err)
	}

	if string(val) != "bar" {
		t.Errorf("expected 'bar' but got: '%s'", string(val))
	}

	// deleting it
	err = kvdb.Delete("foo")
	if err != nil {
		t.Errorf("got error while deleting: %s", err)
	}

	_, err = kvdb.Get("foo")
	if err != kv.ErrNotFound {
		t.Errorf("expected to get an error on deleted key")
	}
}

func TestList(t *testing.T) {
	count := 3

	impl := getImplementer(t)
	kv, err := kv.New(impl, "test", "testlist")
	if err != nil {
		t.Fatalf("failed to create kv: %s", err)
	}
	defer kv.Teardown()

	for i := 0; i < count; i++ {
		err = kv.Put(fmt.Sprint(i), []byte(fmt.Sprintf("bar-%d", i)))
		if err != nil {
			t.Errorf("failed to put: %s", err)
		}
	}

	items, err := kv.List("")
	if err != nil {
		t.Fatalf("failed to list items, error: %s", err)
	}

	if len(items) != count {
		t.Errorf("expected %d items, got: %d", count, len(items))
	}

	if string(items["0"]) != "bar-0" {
		t.Errorf("unexpected value on '0': %s", items["0"])
	}
	if string(items["1"]) != "bar-1" {
		t.Errorf("unexpected value on '1': %s", items["1"])
	}
	if string(items["2"]) != "bar-2" {
		t.Errorf("unexpected value on '2': %s", items["2"])
	}

}

func TestListPrefix(t *testing.T) {
	impl := getImplementer(t)
	kv, err := kv.New(impl, "test", "testlistprefix")
	if err != nil {
		t.Fatalf("failed to create kv: %s", err)
	}
	defer kv.Teardown()

	err = kv.Put("aaa", []byte("aaa"))
	if err != nil {
		t.Errorf("failed to put key, error: %s", err)
	}
	err = kv.Put("aaaaa", []byte("aaa"))
	if err != nil {
		t.Errorf("failed to put key, error: %s", err)
	}
	err = kv.Put("aaaaaaa", []byte("aaa"))
	if err != nil {
		t.Errorf("failed to put key, error: %s", err)
	}

	err = kv.Put("bbb", []byte("bbb"))
	if err != nil {
		t.Errorf("failed to put key, error: %s", err)
	}
	err = kv.Put("bbbbb", []byte("bbb"))
	if err != nil {
		t.Errorf("failed to put key, error: %s", err)
	}
	err = kv.Put("bbbbbbb", []byte("bbb"))
	if err != nil {
		t.Errorf("failed to put key, error: %s", err)
	}

	items, err := kv.List("aaa")
	if err != nil {
		t.Fatalf("failed to list items, error: %s", err)
	}

	if len(items) != 3 {
		t.Errorf("expected %d items, got: %d", 3, len(items))
	}

	if string(items["aaa"]) != "aaa" {
		t.Errorf("unexpected value on 'aaa': %s", items["aaa"])
	}
	if string(items["aaaaa"]) != "aaa" {
		t.Errorf("unexpected value on 'aaaaa': %s", items["aaaaa"])
	}
	if string(items["aaaaaaa"]) != "aaa" {
		t.Errorf("unexpected value on 'aaaaaaa': %s", items["aaaaaaa"])
	}

}
