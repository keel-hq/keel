package aws

import (
	"sync"
	"time"

	"github.com/keel-hq/keel/types"

	"testing"
)

func TestPutCreds(t *testing.T) {
	c := NewCache(time.Second * 5)

	creds := &types.Credentials{
		Username: "user-1",
		Password: "pass-1",
	}

	c.Put("reg1", creds)

	stored, err := c.Get("reg1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if stored.Username != "user-1" {
		t.Errorf("username mismatch: %s", stored.Username)
	}
	if stored.Password != "pass-1" {
		t.Errorf("password mismatch: %s", stored.Password)
	}
}

func TestExpiry(t *testing.T) {
	c := &Cache{
		creds: make(map[string]*item),
		mu:    &sync.RWMutex{},
		ttl:   time.Millisecond * 500,
		tick:  time.Millisecond * 100,
	}

	go c.expiryService()

	creds := &types.Credentials{
		Username: "user-1",
		Password: "pass-1",
	}

	c.Put("reg1", creds)

	time.Sleep(1100 * time.Millisecond)

	_, err := c.Get("reg1")
	if err == nil {
		t.Fatalf("expected to get an error about missing record")
	}

}
