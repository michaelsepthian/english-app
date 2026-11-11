// MIVA-SHARED: small application utilities.
package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
