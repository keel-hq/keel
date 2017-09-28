package cache

import (
	"context"
	"errors"
	"time"
)

// Cache - generic cache interface
// type Cache interface {
// 	Put(ctx context.Context, key string, value []byte) error
// 	Get(ctx context.Context, key string) (value []byte, err error)
// 	Delete(ctx context.Context, key string) error
// 	List(prefix string) ([][]byte, error)
// }
type Cache interface {
	Put(key string, value []byte) error
	Get(key string) (value []byte, err error)
	Delete(key string) error
	List(prefix string) (map[string][]byte, error)
}

type expirationContextKeyType int

const expirationContextKey expirationContextKeyType = 1

// SetContextExpiration - set cache expiration context
func SetContextExpiration(ctx context.Context, expiration time.Duration) context.Context {
	return context.WithValue(ctx, expirationContextKey, expiration)
}

// GetContextExpiration - gets expiration from context, returns it and also returns
// ok - true/false to indicate whether ctx value was found
func GetContextExpiration(ctx context.Context) (exp time.Duration, ok bool) {
	expiration := ctx.Value(expirationContextKey)
	if expiration != nil {
		return expiration.(time.Duration), true
	}
	return 0, false
}

var (
	ErrNotFound = errors.New("not found")
	ErrExpired  = errors.New("entry expired")
)
