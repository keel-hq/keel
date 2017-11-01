package kubekv

import (
	"github.com/keel-hq/keel/cache"

	"github.com/rusenask/k8s-kv/kv"
)

type KubeKV struct {
	kv *kv.KV
}

func New(implementer kv.ConfigMapInterface, bucket string) (*KubeKV, error) {

	kvdb, err := kv.New(implementer, "keel", bucket)
	if err != nil {
		return nil, err
	}

	return &KubeKV{
		kv: kvdb,
	}, nil
}

func (k *KubeKV) Put(key string, value []byte) error {
	return k.kv.Put(key, value)
}

func (k *KubeKV) Get(key string) (value []byte, err error) {
	value, err = k.kv.Get(key)
	if err != nil {
		if err == kv.ErrNotFound {
			return []byte(""), cache.ErrNotFound
		}

	}
	return
}

func (k *KubeKV) Delete(key string) error {
	return k.kv.Delete(key)
}

func (k *KubeKV) List(prefix string) (map[string][]byte, error) {
	return k.kv.List(prefix)
}
