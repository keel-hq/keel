package kv

import (
	"fmt"
	"testing"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type fakeImplementer struct {
	getcfgMap *v1.ConfigMap

	createdMap *v1.ConfigMap
	updatedMap *v1.ConfigMap

	deletedName    string
	deletedOptions *meta_v1.DeleteOptions
}

func (i *fakeImplementer) Get(name string, options meta_v1.GetOptions) (*v1.ConfigMap, error) {
	return i.getcfgMap, nil
}

func (i *fakeImplementer) Create(cfgMap *v1.ConfigMap) (*v1.ConfigMap, error) {
	i.createdMap = cfgMap
	return i.createdMap, nil
}

func (i *fakeImplementer) Update(cfgMap *v1.ConfigMap) (*v1.ConfigMap, error) {
	i.updatedMap = cfgMap
	return i.updatedMap, nil
}

func (i *fakeImplementer) Delete(name string, options *meta_v1.DeleteOptions) error {
	i.deletedName = name
	i.deletedOptions = options
	return nil
}

func TestGetMap(t *testing.T) {
	fi := &fakeImplementer{
		getcfgMap: &v1.ConfigMap{
			Data: map[string]string{
				"foo": "bar",
			},
		},
	}
	kv, err := New(fi, "app", "b1")
	if err != nil {
		t.Fatalf("failed to get kv: %s", err)
	}

	cfgMap, err := kv.getMap()
	if err != nil {
		t.Fatalf("failed to get map: %s", err)
	}

	if cfgMap.Data["foo"] != "bar" {
		t.Errorf("cfgMap.Data is missing expected key")
	}
}

func TestGet(t *testing.T) {

	im := map[string][]byte{
		"foo": []byte("bar"),
	}
	fi := &fakeImplementer{
		getcfgMap: &v1.ConfigMap{
			Data: map[string]string{},
		},
	}
	kv, err := New(fi, "app", "b1")
	if err != nil {
		t.Fatalf("failed to get kv: %s", err)
	}

	cfgMap, _ := kv.getMap()

	kv.saveInternalMap(cfgMap, im)

	val, err := kv.Get("foo")
	if err != nil {
		t.Fatalf("failed to get key: %s", err)
	}

	if string(val) != "bar" {
		t.Errorf("expected 'bar' but got: %s", string(val))
	}
}

func TestUpdate(t *testing.T) {

	im := map[string][]byte{
		"a": []byte("a-val"),
		"b": []byte("b-val"),
		"c": []byte("c-val"),
		"d": []byte("d-val"),
	}

	fi := &fakeImplementer{
		getcfgMap: &v1.ConfigMap{
			Data: map[string]string{},
		},
	}
	kv, err := New(fi, "app", "b1")
	if err != nil {
		t.Fatalf("failed to get kv: %s", err)
	}

	cfgMap, _ := kv.getMap()

	kv.saveInternalMap(cfgMap, im)

	err = kv.Put("b", []byte("updated"))
	if err != nil {
		t.Fatalf("failed to get key: %s", err)
	}

	updatedIm, err := decodeInternalMap(kv.serializer, fi.updatedMap.Data[dataKey])
	if err != nil {
		t.Fatalf("failed to decode internal map: %s", err)
	}

	if string(updatedIm["b"]) != "updated" {
		t.Errorf("b value was not updated")
	}

}

func TestEncodeInternal(t *testing.T) {
	serializer := DefaultSerializer()

	im := make(map[string][]byte)

	for i := 0; i < 100; i++ {
		im[fmt.Sprintf("foo-%d", i)] = []byte(fmt.Sprintf("some important data here %d", i))
	}

	encoded, err := encodeInternalMap(serializer, im)
	if err != nil {
		t.Fatalf("failed to encode map: %s", err)
	}

	decoded, err := decodeInternalMap(serializer, encoded)
	if err != nil {
		t.Fatalf("failed to decode map: %s", err)
	}

	if string(decoded["foo-1"]) != "some important data here 1" {
		t.Errorf("expected to find 'some important data here 1' but got: %s", string(decoded["foo-1"]))
	}

}
