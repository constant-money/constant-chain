package bridgesig

import (
	"crypto/ecdsa"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func KeyGen(seed []byte) (ecdsa.PrivateKey, ecdsa.PublicKey) {
	priKey := new(ecdsa.PrivateKey)
	priKey.Curve = ethcrypto.S256()
	priKey.D = B2ImN(seed)
	priKey.PublicKey.X, priKey.PublicKey.Y = priKey.Curve.ScalarBaseMult(priKey.D.Bytes())
	return *priKey, priKey.PublicKey
}

func SKBytes(priKey *ecdsa.PrivateKey) []byte {
	return ethcrypto.FromECDSA(priKey)
}

func PKBytes(pubKey *ecdsa.PublicKey) []byte {
	return ethcrypto.CompressPubkey(pubKey)
}
