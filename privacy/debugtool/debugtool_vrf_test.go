package debugtool

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/operation/curve25519"
	"math/big"
	"testing"
)

var x = privacy.RandomScalar()
var g = privacy.RandomPoint()
var pubKey = new(privacy.Point).ScalarMult(g, x)
var vrfWitness = NewVRFWitness(x, g)
var L = ConvertByte2BigInt(curve25519.L[:])
//L = 7237005577332262213973186563042994240857116359379907606001950938285454250989

func ConvertByte2BigInt(b []byte) *big.Int {
	bLE := b
	for i := 0; i < len(bLE)/2; i++ {
		bLE[i], bLE[len(bLE)-i-1] = bLE[len(bLE)-i-1], bLE[i]
	}
	return new(big.Int).SetBytes(bLE)
}


func GenerateRandomValue() *big.Int {
	msg := operation.RandomScalar().ToBytesS()
	y, pi, _ := vrfWitness.Compute(msg)
	isValid, err := pi.Verify(msg, g, pubKey, y)
	if err != nil || !isValid {
		panic("something went wrong ...")
	}
	return ConvertByte2BigInt(y.ToBytesS())
}

func GenerateOutputRange(inputRange []int64, total int64) []*big.Int {
	outputRange := make([]*big.Int, len(inputRange))
	tmpSum := int64(0)
	for i, input := range inputRange {
		tmpSum += input
		tmp1 := new(big.Int).Mul(L, new(big.Int).SetInt64(tmpSum))
		outputRange[i] = new(big.Int).Div(tmp1, new(big.Int).SetInt64(total))
	}
	outputRange[len(outputRange)-1] = L
	if tmpSum != total {
		panic("something went wrong with input")
	}
	return outputRange
}


func TestVRFProof_Verify(t *testing.T) {
	total := int64(404500)
	inputRange := []int64{25, 40, 100, 100, 200, 500, 1000, 2000, 3000, 21000, 376535}
	outputRange := GenerateOutputRange(inputRange, total)
	countPrizes := make([]int64, len(outputRange))
	for i := range countPrizes {
		fmt.Println(outputRange[i])
		countPrizes[i] = 0
	}

	for i:=0;i< int(total);i++{
		y := GenerateRandomValue()
		for index, li := range outputRange {
			if y.Cmp(li) == -1 {
				countPrizes[index] += 1
				break
			}
		}
		if i % 10000 == 0 {
			fmt.Println("Count: ", i)
			for i := range countPrizes {
				fmt.Printf("Range %v \t %v\n", len(countPrizes) - 1 -i, countPrizes[len(countPrizes) - 1 -i])
			}
		}
	}
	for i := range countPrizes {
		fmt.Printf("Range %v \t %v\n", len(countPrizes) - 1 -i, countPrizes[len(countPrizes) - 1 -i])
	}
}