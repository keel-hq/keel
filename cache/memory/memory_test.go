package memory

import (
	"log"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	err := c.Put("a", []byte("b"))
	if err != nil {
		t.Errorf("failed to SET a key, got error: %s", err)
	}

	val, err := c.Get("a")
	if err != nil {
		t.Errorf("failed to GET a key, got error: %s", err)
	}

	if string(val) != "b" {
		log.Panicf("value %v", val)
	}

	cc := c.Copy()
	if len(cc) != 1 {
		t.Errorf("expected 1 item, got: %d", len(cc))
	}
}

func TestCacheDel(t *testing.T) {
	c := NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	err := c.Put("a", []byte("b"))
	if err != nil {
		t.Errorf("failed to SET a key, got error: %s", err)
	}

	val, err := c.Get("a")
	if err != nil {
		t.Errorf("failed to GET a key, got error: %s", err)
	}

	if string(val) != "b" {
		log.Panicf("value %v", val)
	}

	err = c.Delete("a")
	if err != nil {
		t.Errorf("faield to delete entry, got error: %s", err)
	}

	_, err = c.Get("a")
	if err == nil {
		t.Errorf("expected to get an error after deletion, but got nil")
	}
}

func TestCacheExpiration(t *testing.T) {
	c := NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	err := c.Put("a", []byte("b"))
	if err != nil {
		t.Errorf("failed to SET a key, got error: %s", err)
	}

	val, err := c.Get("a")
	if err != nil {
		t.Errorf("failed to GET a key, got error: %s", err)
	}

	if string(val) != "b" {
		log.Panicf("value %v", val)
	}

	time.Sleep(200 * time.Millisecond)

	_, err = c.Get("a")
	if err == nil {
		t.Errorf("expected to get an error after deletion, but got nil")
	}
}
