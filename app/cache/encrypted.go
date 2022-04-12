package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/blue-health/blue-health-go-srv/app/crypto"
)

type Encrypted struct {
	inner Cache
	aes   *crypto.AES
}

func NewEncrypted(aes *crypto.AES, inner Cache) *Encrypted {
	return &Encrypted{aes: aes, inner: inner}
}

func (e *Encrypted) Get(ctx context.Context, key string) ([]byte, error) {
	var (
		i   []byte
		err error
	)

	if i, err = e.inner.Get(ctx, key); err == nil && i != nil {
		i, err = e.aes.Decrypt(i)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt data: %w", err)
		}
	}

	return i, err
}

func (e *Encrypted) Set(ctx context.Context, key string, value []byte, expiry time.Duration) error {
	b, err := e.aes.Encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	return e.inner.Set(ctx, key, b, expiry)
}

func (e *Encrypted) Del(ctx context.Context, key string) error {
	return e.inner.Del(ctx, key)
}
