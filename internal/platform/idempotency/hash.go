package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
)

func RequestHash(method, path string, body []byte) string {
	sum := sha256.New()
	_, _ = sum.Write([]byte(method))
	_, _ = sum.Write([]byte{'\n'})
	_, _ = sum.Write([]byte(path))
	_, _ = sum.Write([]byte{'\n'})
	_, _ = sum.Write(body)
	return hex.EncodeToString(sum.Sum(nil))
}
