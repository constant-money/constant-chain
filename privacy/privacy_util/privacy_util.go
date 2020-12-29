package privacy_util

import (
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/curve25519"
	"math/big"

	"github.com/incognitochain/incognito-chain/common"
)

func ScalarToBigInt(sc *privacy.Scalar) *big.Int {
	keyR := privacy.Reverse(sc.GetKey())
	keyRByte := keyR.ToBytes()
	bi := new(big.Int).SetBytes(keyRByte[:])
	return bi
}

func BigIntToScalar(bi *big.Int) *privacy.Scalar {
	biByte := common.AddPaddingBigInt(bi, privacy.Ed25519KeySize)
	var key curve25519.Key
	key.FromBytes(SliceToArray(biByte))
	keyR := privacy.Reverse(key)
	sc, err := new(privacy.Scalar).SetKey(&keyR)
	if err != nil {
		return nil
	}
	return sc
}

// ConvertIntToBinary represents a integer number in binary array with little endian with size n
func ConvertIntToBinary(inum int, n int) []byte {
	binary := make([]byte, n)

	for i := 0; i < n; i++ {
		binary[i] = byte(inum % 2)
		inum = inum / 2
	}

	return binary
}

// ConvertIntToBinary represents a integer number in binary
func ConvertUint64ToBinary(number uint64, n int) []*privacy.Scalar {
	if number == 0 {
		res := make([]*privacy.Scalar, n)
		for i := 0; i < n; i++ {
			res[i] = new(privacy.Scalar).FromUint64(0)
		}
		return res
	}

	binary := make([]*privacy.Scalar, n)

	for i := 0; i < n; i++ {
		binary[i] = new(privacy.Scalar).FromUint64(number % 2)
		number = number / 2
	}
	return binary
}

// isOdd check a big int is odd or not
func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}

// padd1Div4 computes (p + 1) / 4
func padd1Div4(p *big.Int) (res *big.Int) {
	res = new(big.Int).Add(p, big.NewInt(1))
	res.Div(res, big.NewInt(4))
	return
}

// paddedAppend appends the src byte slice to dst, returning the new slice.
// If the length of the source is smaller than the passed size, leading zero
// bytes are appended to the dst slice before appending src.
func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}

func ConvertScalarArrayToBigIntArray(scalarArr []*privacy.Scalar) []*big.Int {
	res := make([]*big.Int, len(scalarArr))

	for i := 0; i < len(res); i++ {
		tmp := privacy.Reverse(scalarArr[i].GetKey())
		res[i] = new(big.Int).SetBytes(ArrayToSlice(tmp.ToBytes()))
	}

	return res
}

func SliceToArray(slice []byte) [privacy.Ed25519KeySize]byte {
	var array [privacy.Ed25519KeySize]byte
	copy(array[:], slice)
	return array
}

func ArrayToSlice(array [privacy.Ed25519KeySize]byte) []byte {
	var slice []byte
	slice = array[:]
	return slice
}
