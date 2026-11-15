package ids

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns an id such as "veh_a1b2c3d4e5f6".
func New(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}