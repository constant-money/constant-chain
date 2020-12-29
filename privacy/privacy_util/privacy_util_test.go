package privacy_util

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/stretchr/testify/assert"
)

var _ = func() (_ struct{}) {
	fmt.Println("This runs before init()!")
	Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

func TestUtilsRandBytes(t *testing.T) {
	data := []int{
		0,
		10,
		45,
		100,
		1000,
	}

	for _, item := range data {
		res := RandBytes(item)
		fmt.Printf("Res: %v\n", res)
		assert.Equal(t, item, len(res))
	}
}

func TestUtilsConvertIntToBinary(t *testing.T) {
	data := []struct {
		number int
		size   int
		binary []byte
	}{
		{64, 8, []byte{0, 0, 0, 0, 0, 0, 1, 0}},
		{100, 10, []byte{0, 0, 1, 0, 0, 1, 1, 0, 0, 0}},
		{1, 8, []byte{1, 0, 0, 0, 0, 0, 0, 0}},
	}

	for _, item := range data {
		res := ConvertIntToBinary(item.number, item.size)
		assert.Equal(t, item.binary, res)
	}
}

// func TestUtilsConvertBigIntToBinary(t *testing.T) {
// 	data := []struct {
// 		number *big.Int
// 		size   int
// 		binary []*big.Int
// 	}{
// 		{new(big.Int).FromUint64(uint64(64)), 8, []*big.Int{new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(1), new(big.Int).SetInt64(0)}},
// 		{new(big.Int).FromUint64(uint64(100)), 10, []*big.Int{new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(1), new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(1), new(big.Int).SetInt64(1), new(big.Int).SetInt64(0), new(big.Int).SetInt64(0), new(big.Int).SetInt64(0)}},
// 	}

// 	for _, item := range data {
// 		res := ConvertBigIntToBinary(item.number, item.size)
// 		assert.Equal(t, item.binary, res)
// 	}
// }

func TestUtilsAddPaddingBigInt(t *testing.T) {
	data := []struct {
		number *big.Int
		size   int
	}{
		{new(big.Int).SetBytes(RandBytes(12)), common.BigIntSize},
		{new(big.Int).SetBytes(RandBytes(42)), 50},
		{new(big.Int).SetBytes(RandBytes(0)), 10},
	}

	for _, item := range data {
		res := common.AddPaddingBigInt(item.number, item.size)
		assert.Equal(t, item.size, len(res))
	}
}

func TestUtilsIntToByteArr(t *testing.T) {
	data := []struct {
		number int
		bytes  []byte
	}{
		{12345, []byte{48, 57}},
		{123, []byte{0, 123}},
		{0, []byte{0, 0}},
	}

	for _, item := range data {
		res := common.IntToBytes(item.number)
		assert.Equal(t, item.bytes, res)

		number := common.BytesToInt(res)
		assert.Equal(t, item.number, number)
	}
}

func TestInterface(t *testing.T) {
	a := make(map[string]interface{})
	a["x"] = "10"

	value, ok := a["y"].(string)
	if !ok {
		fmt.Printf("Param is invalid\n")
	}

	value2, ok := a["y"]
	if !ok {
		fmt.Printf("Param is invalid\n")
	}

	value3, ok := a["x"].(string)
	if !ok {
		fmt.Printf("Param is invalid\n")
	}

	fmt.Printf("Value: %v\n", value)
	fmt.Printf("Value2: %v\n", value2)
	fmt.Printf("Value2: %v\n", value3)
}

func TestFee(t *testing.T) {
	inValue := uint64(50000)
	outValue1 := uint64(23000)
	fee := -1
	fee2 := uint64(fee)
	outValue2 := int64(inValue - outValue1 - fee2)

	fmt.Printf("Fee uint64: %v\n", uint64(fee))
	fmt.Printf("outValue2: %v\n", outValue2)

	comInputValueSum := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenValueIndex], new(operation.Scalar).FromUint64(uint64(inValue)))

	comOutputValue1 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenValueIndex], new(operation.Scalar).FromUint64(uint64(outValue1)))

	comOutputValue2 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenValueIndex], new(operation.Scalar).FromUint64(uint64(outValue2)))

	comOutputValueSum := new(operation.Point).Add(comOutputValue1, comOutputValue2)

	comFee := new(operation.Point)
	if fee2 > 0 {
		fmt.Printf("fee2 > 0\n")
		comFee = comFee.ScalarMult(operation.PedCom.G[operation.PedersenValueIndex], new(operation.Scalar).FromUint64(uint64(fee2)))
	}

	tmp1 := new(operation.Point).Add(comOutputValueSum, comFee)

	if operation.IsPointEqual(tmp1, comInputValueSum) {
		fmt.Printf("Equal\n")
	} else {
		fmt.Printf(" Not Equal\n")
	}

	//fee := -10
	//output := -9
	//aUint64 := uint64(a)
	//bUint64 := uint64(b)
	//
	//fmt.Printf("aUint64: %v\n", aUint64)
	//fmt.Printf("bUint64: %v\n", bUint64)

	//comOutputValueSum.Add(comOutputValueSum, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenValueIndex], new(privacy.Scalar).FromUint64(uint64(fee))))
}

func TestEncryptByXorOperator(t *testing.T) {
	v := new(big.Int).SetUint64(100)

	randomness := operation.RandomScalar()
	randomnessBytes := randomness.ToBytesS()

	// encrypt
	ciphertext := v.Uint64()

	for i := 0; i < 4; i++ {
		randSlice := randomnessBytes[i*8 : i*8+8]
		randSliceUint64 := new(big.Int).SetBytes(randSlice).Uint64()
		ciphertext = ciphertext ^ randSliceUint64
	}
	fmt.Printf("ciphertext %v\n", ciphertext)

	// decrypt
	plaintext := ciphertext
	for i := 0; i < 4; i++ {
		randSlice := randomnessBytes[i*8 : i*8+8]
		randSliceUint64 := new(big.Int).SetBytes(randSlice).Uint64()
		plaintext = plaintext ^ randSliceUint64
	}
	fmt.Printf("plaintext %v\n", plaintext)
}
