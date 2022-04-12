package crypto

import (
	"crypto/sha256"
	"encoding/base64"
)

func Hash(b []byte) string {
	h := sha256.New()
	h.Write(b)

	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}
