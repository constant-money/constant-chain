package common

import (
	"crypto/ed25519"
)

func GenBTCPrivateKey(IncKeyBytes []byte) []byte {
	BTCKeyBytes := ed25519.NewKeyFromSeed(IncKeyBytes)[32:]
	return BTCKeyBytes
}
