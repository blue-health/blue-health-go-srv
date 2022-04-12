package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

func RandomBytes(n uint) ([]byte, error) {
	b := make([]byte, n)

	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to read crypto/rand: %w", err)
	}

	return b, nil
}

func RandomString() string {
	u := [16]byte(uuid.New())
	return hex.EncodeToString(u[:])
}
