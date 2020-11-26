package blsmultisig

import (
	"math/big"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"golang.org/x/crypto/sha3"
)

// G1P2I is Point to Big Int
func G1P2I(point *bn256.G1) *big.Int {
	pnByte := CmprG1(point)
	res := big.NewInt(0)
	res.SetBytes(pnByte)
	for res.Cmp(bn256.Order) != -1 {
		pnByte = Hash4Bls(pnByte)
		res.SetBytes(pnByte)
	}
	return res
}

// B2I convert byte array to big int which belong to Fp
func B2I(bytes []byte) *big.Int {
	res := big.NewInt(0)
	res.SetBytes(bytes)
	for res.Cmp(bn256.Order) != -1 {
		bytes = Hash4Bls(bytes)
		res.SetBytes(bytes)
	}
	return res
}

// I2G1P is Big Int to Point, in BLS-BFT, it called H0
func I2G1P(bigInt *big.Int) *bn256.G1 {
	x := big.NewInt(0)
	x.Set(bigInt)
	for i := 0; ; i++ {
		res, err := xCoor2G1P(x, x.Bit(0) == 1)
		if err == nil {
			return res
		}
		x.SetBytes(Hash4Bls(x.Bytes()))
	}
}

// B2G1P is Bytes to Point, in BLS-BFT, it also called H0
func B2G1P(bytes []byte) *bn256.G1 {
	x := big.NewInt(0)
	x.SetBytes(Hash4Bls(bytes))
	return I2G1P(x)
}

// Hash4Bls is Hash function for calculate block hash
// this is different from hash function for calculate transaction hash
func Hash4Bls(data []byte) []byte {
	hashMachine := sha3.NewLegacyKeccak256()
	hashMachine.Write(data)
	return hashMachine.Sum(nil)
}
