package privacy_conversion

import (
	"bytes"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/oneoutofmany"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/serialnumberprivacy"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	numTests       = 100
	numInputCoins  = 10
	numOutputCoins = 1
)

//HELPER FUNCTIONS
//Random functions
func RandPubKeyForShard(targetShardID byte) *operation.Point {
	for {
		pk := operation.RandomPoint()
		pkBytes := pk.ToBytesS()
		lastByte := pkBytes[len(pkBytes)-1]
		shardID := common.GetShardIDFromLastByte(lastByte)
		if shardID == targetShardID {
			return pk
		}
	}
}
func RandomPlainCoinList(numCoins int, shardID byte) ([]coin.PlainCoin, error) {
	res := make([]coin.PlainCoin, 0)

	for i := 0; i < numCoins; i++ {
		plainCoin := new(coin.PlainCoinV1)

		plainCoin.SetPublicKey(RandPubKeyForShard(shardID))
		plainCoin.SetValue(uint64(common.RandInt()))
		plainCoin.SetRandomness(operation.RandomScalar())
		plainCoin.SetSNDerivator(operation.RandomScalar())
		plainCoin.SetInfo(common.RandBytes(common.RandIntInterval(0, 511)))

		err := plainCoin.CommitAll()
		if err != nil {
			return nil, err
		}
		res = append(res, plainCoin)
	}

	return res, nil
}
func RandCoinList(numCoins int, shardID byte) ([]coin.Coin, error) {
	res := make([]coin.Coin, 0)

	for i := 0; i < numCoins; i++ {
		coinDetail := new(coin.PlainCoinV1)

		coinDetail.SetPublicKey(RandPubKeyForShard(shardID))
		coinDetail.SetValue(uint64(common.RandInt()))
		coinDetail.SetRandomness(operation.RandomScalar())
		coinDetail.SetSNDerivator(operation.RandomScalar())
		coinDetail.SetInfo(common.RandBytes(common.RandIntInterval(0, 511)))

		err := coinDetail.CommitAll()
		if err != nil {
			return nil, err
		}

		outCoin := new(coin.CoinV1).Init()
		outCoin.CoinDetails = coinDetail

		res = append(res, outCoin)
	}

	return res, nil
}
func RandOneOfManyProofs(numProofs int) []*oneoutofmany.OneOutOfManyProof {
	res := make([]*oneoutofmany.OneOutOfManyProof, 0)

	for i := 0; i < numProofs; i++ {
		oneProof := new(oneoutofmany.OneOutOfManyProof).Init()

		commitments := make([]*operation.Point, 0)
		for i := 0; i < 8; i++ {
			commitments = append(commitments, operation.RandomPoint())
		}
		cl := make([]*operation.Point, 3)
		ca := make([]*operation.Point, 3)
		cb := make([]*operation.Point, 3)
		cd := make([]*operation.Point, 3)
		f := make([]*operation.Scalar, 3)
		za := make([]*operation.Scalar, 3)
		zb := make([]*operation.Scalar, 3)
		for i := 0; i < 3; i++ {
			cl[i] = operation.RandomPoint()
			ca[i] = operation.RandomPoint()
			cb[i] = operation.RandomPoint()
			cd[i] = operation.RandomPoint()

			f[i] = operation.RandomScalar()
			za[i] = operation.RandomScalar()
			zb[i] = operation.RandomScalar()
		}
		zd := operation.RandomScalar()

		oneProof.Set(commitments, cl, ca, cb, cd, f, za, zb, zd)

		res = append(res, oneProof)
	}

	return res
}
func RandSerialNumberProofs(numInputs int) []*serialnumberprivacy.SNPrivacyProof {
	res := make([]*serialnumberprivacy.SNPrivacyProof, 0)

	for i := 0; i < numInputs; i++ {
		snProof := new(serialnumberprivacy.SNPrivacyProof).Init()

		SN := operation.RandomPoint()
		comSK := operation.RandomPoint()
		comInput := operation.RandomPoint()
		stmt := new(serialnumberprivacy.SerialNumberPrivacyStatement)
		stmt.Set(SN, comSK, comInput)

		tSK := operation.RandomPoint()
		tInput := operation.RandomPoint()
		tSN := operation.RandomPoint()

		zSK := operation.RandomScalar()
		zRSK := operation.RandomScalar()
		zInput := operation.RandomScalar()
		zRInput := operation.RandomScalar()

		snProof.Set(stmt, tSK, tInput, tSN, zSK, zRSK, zInput, zRInput)

		res = append(res, snProof)
	}

	return res
}
func RandPointList(numPoints int) []*operation.Point {
	res := make([]*operation.Point, 0)
	for i := 0; i < numPoints; i++ {
		res = append(res, operation.RandomPoint())
	}
	return res
}
func RandScalarList(numScalars int) []*operation.Scalar {
	res := make([]*operation.Scalar, 0)
	for i := 0; i < numScalars; i++ {
		res = append(res, operation.RandomScalar())
	}
	return res
}
func RandUint64List(num int) []uint64 {
	res := make([]uint64, 0)
	for i := 0; i < num; i++ {
		res = append(res, uint64(common.RandInt()))
	}

	return res
}
func RandConversionProof(senderShard, receiverShard byte) (*ConversionProof, error) {
	proof := new(ConversionProof)
	proof.Init()

	inputCoins, err := RandomPlainCoinList(numInputCoins, senderShard)
	if err != nil {
		return nil, err
	}
	err = proof.SetInputCoins(inputCoins)
	if err != nil {
		return nil, err
	}

	outputCoins, err := RandCoinList(1, receiverShard)
	if err != nil {
		return nil, err
	}
	err = proof.SetOutputCoins(outputCoins)
	if err != nil {
		return nil, err
	}

	oneOfManyProof := RandOneOfManyProofs(numInputCoins)
	proof.SetOneOfManyProof(oneOfManyProof)

	snProofs := RandSerialNumberProofs(numInputCoins)
	proof.SetSerialNumberProof(snProofs)

	comInputSK := operation.RandomPoint()
	proof.SetCommitmentInputSecretKey(comInputSK)

	comInputShardID := operation.RandomPoint()
	proof.SetCommitmentShardID(comInputShardID)

	comInputSND := RandPointList(numInputCoins)
	proof.SetCommitmentInputSND(comInputSND)

	comInputValue := RandPointList(numInputCoins)
	proof.SetCommitmentInputValue(comInputValue)

	commitmentIndices := RandUint64List(numInputCoins * 8)
	proof.SetCommitmentIndices(commitmentIndices)

	return proof, nil
}

//Coin-related function

//Proof-related functions
func CorruptProof(proofBytes []byte, numPositions int) []byte {
	n := len(proofBytes)
	res := make([]byte, n)
	copy(res, proofBytes)

	for i := 0; i < numPositions; i++ {
		positionToCorrupt := common.RandIntInterval(0, n-1)
		byteReplace := common.RandInt() % 256
		res[positionToCorrupt] = byte(byteReplace)
	}

	return res
}

//END HELPER FUNCTIONS

func TestConversionProof_SetBytes(t *testing.T) {
	for i := 0; i < numTests; i++ {
		if i%(numTests/100) == 0 {
			fmt.Printf("Test #%v\n", i)
		}
		shardIDSender := byte(common.RandInt() % common.MaxShardNumber)
		shardIDReceiver := byte(common.RandInt() % common.MaxShardNumber)

		proof, err := RandConversionProof(shardIDSender, shardIDReceiver)
		if err != nil {
			panic(err)
		}

		proofBytes := proof.Bytes()

		newProof := new(ConversionProof)
		err1 := newProof.SetBytes(proofBytes)
		if err1 != nil {
			panic(err1)
		}

		newProofBytes := newProof.Bytes()

		assert.Equal(t, true, bytes.Equal(proofBytes, newProofBytes), "proofBytes not equal")
	}
}

func TestConversionProof_UnmarshalJSON(t *testing.T) {
	for i := 0; i < numTests; i++ {
		if i%(numTests/100) == 0 {
			fmt.Printf("Test #%v\n", i)
		}
		shardIDSender := byte(common.RandInt() % common.MaxShardNumber)
		shardIDReceiver := byte(common.RandInt() % common.MaxShardNumber)

		proof, err := RandConversionProof(shardIDSender, shardIDReceiver)
		if err != nil {
			panic(err)
		}

		proofBytes, err := proof.MarshalJSON()
		if err != nil {
			panic(err)
		}

		newProof := new(ConversionProof)
		err = newProof.UnmarshalJSON(proofBytes)
		if err != nil {
			panic(err)
		}

		newProofBytes, err := newProof.MarshalJSON()
		if err != nil {
			panic(err)
		}

		assert.Equal(t, true, bytes.Equal(proofBytes, newProofBytes), "proofBytes not equal")
	}
}

func TestConversionProof_PanicSetBytes(t *testing.T) {
	start := time.Now()
	numAttempts := 100000

	for i := 0; i < numTests; i++ {
		if i%(numTests/100) == 0 {
			fmt.Printf("Test #%v, Time elapsed: %v\n", i, time.Since(start).String())
		}
		shardIDSender := byte(common.RandInt() % common.MaxShardNumber)
		shardIDReceiver := byte(common.RandInt() % common.MaxShardNumber)

		proof, err := RandConversionProof(shardIDSender, shardIDReceiver)
		if err != nil {
			panic(err)
		}

		proofBytes := proof.Bytes()
		lenProof := len(proofBytes)

		for j := 0; j < numAttempts; j++ {
			if j % (numAttempts/100) == 0 {
				fmt.Printf("Test #%v - Attempt #%v, Time elapsed: %v\n", i, j, time.Since(start).String())
			}

			numPositions := common.RandIntInterval(1, lenProof)
			res := CorruptProof(proofBytes, numPositions)

			newProof := new(ConversionProof)
			_ = newProof.SetBytes(res)
		}

	}
}
