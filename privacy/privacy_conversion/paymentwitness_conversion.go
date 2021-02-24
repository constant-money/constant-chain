package privacy_conversion

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_util"
	zkp "github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/oneoutofmany"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/serialnumberprivacy"
)

// PaymentWitness contains all of witness for proving when spending coins
type ConversionWitness struct {
	privateKey          *operation.Scalar
	inputCoins          []*coin.PlainCoinV1
	outputCoins         []*coin.CoinV1
	commitmentIndices   []uint64
	myCommitmentIndices []uint64

	oneOfManyWitness    []*oneoutofmany.OneOutOfManyWitness
	serialNumberWitness []*serialnumberprivacy.SNPrivacyWitness

	comInputSecretKey             *operation.Point
	comInputSerialNumberDerivator []*operation.Point
	comInputValue                 []*operation.Point
	comInputShardID               *operation.Point

	randSecretKey *operation.Scalar
}

func (witness ConversionWitness) GetRandSecretKey() *operation.Scalar {
	return witness.randSecretKey
}

type ConversionWitnessParam struct {
	PrivateKey              *operation.Scalar
	InputCoins              []*coin.PlainCoinV1
	OutputCoins             []*coin.CoinV1
	PublicKeyLastByteSender byte
	Commitments             []*operation.Point
	CommitmentIndices       []uint64
	MyCommitmentIndices     []uint64
	Fee                     uint64
}

func (witness *ConversionWitness) Init(witnessParam ConversionWitnessParam) *errhandler.PrivacyError {
	_ = witnessParam.Fee
	witness.privateKey = witnessParam.PrivateKey
	witness.inputCoins = witnessParam.InputCoins
	witness.outputCoins = witnessParam.OutputCoins
	witness.commitmentIndices = witnessParam.CommitmentIndices
	witness.myCommitmentIndices = witnessParam.MyCommitmentIndices

	randInputSK := operation.RandomScalar()
	witness.randSecretKey = new(operation.Scalar).Set(randInputSK)

	numInputCoins := len(witness.inputCoins)

	cmInputSK := operation.PedCom.CommitAtIndex(witness.privateKey, randInputSK, operation.PedersenPrivateKeyIndex)
	witness.comInputSecretKey = new(operation.Point).Set(cmInputSK)

	randInputShardID := zkp.FixedRandomnessShardID
	senderShardID := common.GetShardIDFromLastByte(witnessParam.PublicKeyLastByteSender)
	witness.comInputShardID = operation.PedCom.CommitAtIndex(new(operation.Scalar).FromUint64(uint64(senderShardID)), randInputShardID, operation.PedersenShardIDIndex)

	witness.comInputValue = make([]*operation.Point, numInputCoins)
	witness.comInputSerialNumberDerivator = make([]*operation.Point, numInputCoins)

	randInputValue := make([]*operation.Scalar, numInputCoins)
	randInputSND := make([]*operation.Scalar, numInputCoins)

	// cmInputValueAll is sum of all input coins' value commitments
	cmInputValueAll := new(operation.Point).Identity()
	randInputValueAll := new(operation.Scalar).FromUint64(0)

	// Summing all commitments of each input coin into one commitment and proving the knowledge of its Openings
	cmInputSum := make([]*operation.Point, numInputCoins)
	randInputSum := make([]*operation.Scalar, numInputCoins)

	randInputSumAll := new(operation.Scalar).FromUint64(0)

	witness.oneOfManyWitness = make([]*oneoutofmany.OneOutOfManyWitness, numInputCoins)
	witness.serialNumberWitness = make([]*serialnumberprivacy.SNPrivacyWitness, numInputCoins)

	commitmentTemps := make([][]*operation.Point, numInputCoins)
	randInputIsZero := make([]*operation.Scalar, numInputCoins)

	preIndex := 0
	commitments := witnessParam.Commitments
	for i, inputCoin := range witness.inputCoins {
		if i == numInputCoins- 1 {
			randInputValue[i] = new(operation.Scalar).Sub(new(operation.Scalar).FromUint64(0), randInputValueAll)
		} else {
			randInputValue[i] = operation.RandomScalar()
		}

		randInputSND[i] = operation.RandomScalar()

		witness.comInputValue[i] = operation.PedCom.CommitAtIndex(new(operation.Scalar).FromUint64(inputCoin.GetValue()), randInputValue[i], operation.PedersenValueIndex)
		witness.comInputSerialNumberDerivator[i] = operation.PedCom.CommitAtIndex(inputCoin.GetSNDerivator(), randInputSND[i], operation.PedersenSndIndex)

		cmInputValueAll.Add(cmInputValueAll, witness.comInputValue[i])
		randInputValueAll.Add(randInputValueAll, randInputValue[i])

		/***** Build witness for proving one-out-of-N commitments is a commitment to the coins being spent *****/
		cmInputSum[i] = new(operation.Point).Add(cmInputSK, witness.comInputValue[i])
		cmInputSum[i].Add(cmInputSum[i], witness.comInputSerialNumberDerivator[i])
		cmInputSum[i].Add(cmInputSum[i], witness.comInputShardID)

		randInputSum[i] = new(operation.Scalar).Set(randInputSK)
		randInputSum[i].Add(randInputSum[i], randInputValue[i])
		randInputSum[i].Add(randInputSum[i], randInputSND[i])
		randInputSum[i].Add(randInputSum[i], randInputShardID)

		randInputSumAll.Add(randInputSumAll, randInputSum[i])

		// commitmentTemps is a list of commitments for protocol one-out-of-N
		commitmentTemps[i] = make([]*operation.Point, privacy_util.CommitmentRingSize)

		randInputIsZero[i] = new(operation.Scalar).FromUint64(0)
		randInputIsZero[i].Sub(inputCoin.GetRandomness(), randInputSum[i])

		for j := 0; j < privacy_util.CommitmentRingSize; j++ {
			commitmentTemps[i][j] = new(operation.Point).Sub(commitments[preIndex+j], cmInputSum[i])
		}

		if witness.oneOfManyWitness[i] == nil {
			witness.oneOfManyWitness[i] = new(oneoutofmany.OneOutOfManyWitness)
		}
		indexIsZero := witness.myCommitmentIndices[i] % privacy_util.CommitmentRingSize

		witness.oneOfManyWitness[i].Set(commitmentTemps[i], randInputIsZero[i], indexIsZero)
		preIndex = privacy_util.CommitmentRingSize * (i + 1)
		// ---------------------------------------------------

		/***** Build witness for proving that serial number is derived from the committed derivator *****/
		witness.serialNumberWitness[i] = new(serialnumberprivacy.SNPrivacyWitness)
		stmt := new(serialnumberprivacy.SerialNumberPrivacyStatement)
		stmt.Set(inputCoin.GetKeyImage(), cmInputSK, witness.comInputSerialNumberDerivator[i])
		witness.serialNumberWitness[i].Set(stmt, witness.privateKey, randInputSK, inputCoin.GetSNDerivator(), randInputSND[i])
	}

	return nil
}

func (witness *ConversionWitness) Prove() (*ConversionProof, *errhandler.PrivacyError) {
	proof := new(ConversionProof)
	proof.Init()

	proof.inputCoins = witness.inputCoins
	proof.outputCoins = witness.outputCoins

	proof.comInputSecretKey = witness.comInputSecretKey
	proof.comInputValue = witness.comInputValue
	proof.comInputSND = witness.comInputSerialNumberDerivator
	proof.comInputShardID = witness.comInputShardID
	proof.comIndices = witness.commitmentIndices

	numInputCoins := len(witness.oneOfManyWitness)

	for i := 0; i < numInputCoins; i++ {
		// Proving one-out-of-N commitments is a commitment to the coins being spent
		oneOfManyProof, err := witness.oneOfManyWitness[i].Prove()
		if err != nil {
			return nil, errhandler.NewPrivacyErr(errhandler.ProveOneOutOfManyErr, err)
		}
		proof.oneOfManyProof = append(proof.oneOfManyProof, oneOfManyProof)

		// Proving that serial number is derived from the committed derivator
		serialNumberProof, err := witness.serialNumberWitness[i].Prove(nil)
		if err != nil {
			return nil, errhandler.NewPrivacyErr(errhandler.ProveSerialNumberPrivacyErr, err)
		}
		proof.serialNumberProof = append(proof.serialNumberProof, serialNumberProof)
	}
	if len(proof.outputCoins) == 0 {
		return nil, errhandler.NewPrivacyErr(errhandler.UnexpectedErr, errors.New("require one output coin"))
	}

	for i := 0; i < len(proof.GetInputCoins()); i++ {
		err := proof.inputCoins[i].ConcealOutputCoin(nil)
		if err != nil {
			return nil, errhandler.NewPrivacyErr(errhandler.UnexpectedErr, fmt.Errorf("conceal input coin %v error: %v", i, err))
		}
	}
	return proof, nil
}