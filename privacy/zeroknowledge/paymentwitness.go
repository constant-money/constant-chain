package zkp

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge/aggregaterange"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge/oneoutofmany"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge/serialnumbernoprivacy"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge/serialnumberprivacy"
)

// PaymentWitness contains all of witness for proving when spending coins
type PaymentWitness struct {
	privateKey          *privacy.Scalar
	inputCoins          []*privacy.InputCoin
	outputCoins         []*privacy.OutputCoin
	commitmentIndices   []uint64
	myCommitmentIndices []uint64

	oneOfManyWitness             []*oneoutofmany.OneOutOfManyWitness
	serialNumberWitness          []*serialnumberprivacy.SNPrivacyWitness
	serialNumberNoPrivacyWitness []*serialnumbernoprivacy.SNNoPrivacyWitness

	aggregatedRangeWitness *aggregaterange.AggregatedRangeWitness

	comOutputValue                 []*privacy.Point
	comOutputSerialNumberDerivator []*privacy.Point
	comOutputShardID               []*privacy.Point

	comInputSecretKey             *privacy.Point
	comInputValue                 []*privacy.Point
	comInputSerialNumberDerivator []*privacy.Point
	comInputShardID               *privacy.Point

	randSecretKey *privacy.Scalar
}

func (paymentWitness PaymentWitness) GetRandSecretKey() *privacy.Scalar {
	return paymentWitness.randSecretKey
}

type PaymentWitnessParam struct {
	HasPrivacy              bool
	PrivateKey              *privacy.Scalar
	InputCoins              []*privacy.InputCoin
	OutputCoins             []*privacy.OutputCoin
	PublicKeyLastByteSender byte
	Commitments             []*privacy.Point
	CommitmentIndices       []uint64
	MyCommitmentIndices     []uint64
	Fee                     uint64
}

// Build prepares witnesses for all protocol need to be proved when create tx
// if hashPrivacy = false, witness includes spending key, input coins, output coins
// otherwise, witness includes all attributes in PaymentWitness struct
func (wit *PaymentWitness) Init(PaymentWitnessParam PaymentWitnessParam) *privacy.PrivacyError {

	hasPrivacy := PaymentWitnessParam.HasPrivacy
	privateKey := PaymentWitnessParam.PrivateKey
	inputCoins := PaymentWitnessParam.InputCoins
	outputCoins := PaymentWitnessParam.OutputCoins
	publicKeyLastByteSender := PaymentWitnessParam.PublicKeyLastByteSender
	commitments := PaymentWitnessParam.Commitments
	commitmentIndices := PaymentWitnessParam.CommitmentIndices
	myCommitmentIndices := PaymentWitnessParam.MyCommitmentIndices
	_ = PaymentWitnessParam.Fee

	if !hasPrivacy {
		for _, outCoin := range outputCoins {
			outCoin.CoinDetails.SetRandomness(privacy.RandomScalar())
			err := outCoin.CoinDetails.CommitAll()
			if err != nil {
				return privacy.NewPrivacyErr(privacy.CommitNewOutputCoinNoPrivacyErr, nil)
			}
		}
		wit.privateKey = privateKey
		wit.inputCoins = inputCoins
		wit.outputCoins = outputCoins

		if len(inputCoins) > 0 {
			publicKey := inputCoins[0].CoinDetails.GetPublicKey()

			wit.serialNumberNoPrivacyWitness = make([]*serialnumbernoprivacy.SNNoPrivacyWitness, len(inputCoins))
			for i := 0; i < len(inputCoins); i++ {
				/***** Build witness for proving that serial number is derived from the committed derivator *****/
				if wit.serialNumberNoPrivacyWitness[i] == nil {
					wit.serialNumberNoPrivacyWitness[i] = new(serialnumbernoprivacy.SNNoPrivacyWitness)
				}
				wit.serialNumberNoPrivacyWitness[i].Set(inputCoins[i].CoinDetails.GetSerialNumber(), publicKey, inputCoins[i].CoinDetails.GetSNDerivator(), wit.privateKey)
			}
		}

		return nil
	}

	wit.privateKey = privateKey
	wit.inputCoins = inputCoins
	wit.outputCoins = outputCoins
	wit.commitmentIndices = commitmentIndices
	wit.myCommitmentIndices = myCommitmentIndices

	numInputCoin := len(wit.inputCoins)
	numOutputCoin := len(wit.outputCoins)

	randInputSK := privacy.RandomScalar()
	// set rand sk for Schnorr signature
	wit.randSecretKey = new(privacy.Scalar).Set(randInputSK)

	cmInputSK := privacy.PedCom.CommitAtIndex(wit.privateKey, randInputSK, privacy.PedersenPrivateKeyIndex)
	wit.comInputSecretKey = new(privacy.Point).Set(cmInputSK)

	// from BCHeightBreakPointFixRandShardCM, we fixed the randomness for shardID commitment
	// instead of generating it randomly.
	//randInputShardID := privacy.RandomScalar()
	randInputShardID := privacy.FixedRandomnessShardID
	senderShardID := common.GetShardIDFromLastByte(publicKeyLastByteSender)
	wit.comInputShardID = privacy.PedCom.CommitAtIndex(new(privacy.Scalar).FromUint64(uint64(senderShardID)), randInputShardID, privacy.PedersenShardIDIndex)

	wit.comInputValue = make([]*privacy.Point, numInputCoin)
	wit.comInputSerialNumberDerivator = make([]*privacy.Point, numInputCoin)
	// It is used for proving 2 commitments commit to the same value (input)
	//cmInputSNDIndexSK := make([]*privacy.Point, numInputCoin)

	randInputValue := make([]*privacy.Scalar, numInputCoin)
	randInputSND := make([]*privacy.Scalar, numInputCoin)
	//randInputSNDIndexSK := make([]*big.Int, numInputCoin)

	// cmInputValueAll is sum of all input coins' value commitments
	cmInputValueAll := new(privacy.Point).Identity()
	randInputValueAll := new(privacy.Scalar).FromUint64(0)

	// Summing all commitments of each input coin into one commitment and proving the knowledge of its Openings
	cmInputSum := make([]*privacy.Point, numInputCoin)
	randInputSum := make([]*privacy.Scalar, numInputCoin)
	// randInputSumAll is sum of all randomess of coin commitments
	randInputSumAll := new(privacy.Scalar).FromUint64(0)

	wit.oneOfManyWitness = make([]*oneoutofmany.OneOutOfManyWitness, numInputCoin)
	wit.serialNumberWitness = make([]*serialnumberprivacy.SNPrivacyWitness, numInputCoin)

	commitmentTemps := make([][]*privacy.Point, numInputCoin)
	randInputIsZero := make([]*privacy.Scalar, numInputCoin)

	preIndex := 0

	for i, inputCoin := range wit.inputCoins {
		// tx only has fee, no output, Rand_Value_Input = 0
		if numOutputCoin == 0 {
			randInputValue[i] = new(privacy.Scalar).FromUint64(0)
		} else {
			randInputValue[i] = privacy.RandomScalar()
		}
		// commit each component of coin commitment
		randInputSND[i] = privacy.RandomScalar()

		wit.comInputValue[i] = privacy.PedCom.CommitAtIndex(new(privacy.Scalar).FromUint64(inputCoin.CoinDetails.GetValue()), randInputValue[i], privacy.PedersenValueIndex)
		wit.comInputSerialNumberDerivator[i] = privacy.PedCom.CommitAtIndex(inputCoin.CoinDetails.GetSNDerivator(), randInputSND[i], privacy.PedersenSndIndex)

		cmInputValueAll.Add(cmInputValueAll, wit.comInputValue[i])
		randInputValueAll.Add(randInputValueAll, randInputValue[i])

		/***** Build witness for proving one-out-of-N commitments is a commitment to the coins being spent *****/
		cmInputSum[i] = new(privacy.Point).Add(cmInputSK, wit.comInputValue[i])
		cmInputSum[i].Add(cmInputSum[i], wit.comInputSerialNumberDerivator[i])
		cmInputSum[i].Add(cmInputSum[i], wit.comInputShardID)

		randInputSum[i] = new(privacy.Scalar).Set(randInputSK)
		randInputSum[i].Add(randInputSum[i], randInputValue[i])
		randInputSum[i].Add(randInputSum[i], randInputSND[i])
		randInputSum[i].Add(randInputSum[i], randInputShardID)

		randInputSumAll.Add(randInputSumAll, randInputSum[i])

		// commitmentTemps is a list of commitments for protocol one-out-of-N
		commitmentTemps[i] = make([]*privacy.Point, privacy.CommitmentRingSize)

		randInputIsZero[i] = new(privacy.Scalar).FromUint64(0)
		randInputIsZero[i].Sub(inputCoin.CoinDetails.GetRandomness(), randInputSum[i])

		for j := 0; j < privacy.CommitmentRingSize; j++ {
			commitmentTemps[i][j] = new(privacy.Point).Sub(commitments[preIndex+j], cmInputSum[i])
		}

		if wit.oneOfManyWitness[i] == nil {
			wit.oneOfManyWitness[i] = new(oneoutofmany.OneOutOfManyWitness)
		}
		indexIsZero := myCommitmentIndices[i] % privacy.CommitmentRingSize

		wit.oneOfManyWitness[i].Set(commitmentTemps[i], randInputIsZero[i], indexIsZero)
		preIndex = privacy.CommitmentRingSize * (i + 1)
		// ---------------------------------------------------

		/***** Build witness for proving that serial number is derived from the committed derivator *****/
		if wit.serialNumberWitness[i] == nil {
			wit.serialNumberWitness[i] = new(serialnumberprivacy.SNPrivacyWitness)
		}
		stmt := new(serialnumberprivacy.SerialNumberPrivacyStatement)
		stmt.Set(inputCoin.CoinDetails.GetSerialNumber(), cmInputSK, wit.comInputSerialNumberDerivator[i])
		wit.serialNumberWitness[i].Set(stmt, privateKey, randInputSK, inputCoin.CoinDetails.GetSNDerivator(), randInputSND[i])
		// ---------------------------------------------------
	}

	randOutputValue := make([]*privacy.Scalar, numOutputCoin)
	randOutputSND := make([]*privacy.Scalar, numOutputCoin)
	cmOutputValue := make([]*privacy.Point, numOutputCoin)
	cmOutputSND := make([]*privacy.Point, numOutputCoin)

	cmOutputSum := make([]*privacy.Point, numOutputCoin)
	randOutputSum := make([]*privacy.Scalar, numOutputCoin)

	cmOutputSumAll := new(privacy.Point).Identity()

	// cmOutputValueAll is sum of all value coin commitments
	cmOutputValueAll := new(privacy.Point).Identity()

	randOutputValueAll := new(privacy.Scalar).FromUint64(0)

	randOutputShardID := make([]*privacy.Scalar, numOutputCoin)
	cmOutputShardID := make([]*privacy.Point, numOutputCoin)

	for i, outputCoin := range wit.outputCoins {
		if i == len(outputCoins)-1 {
			randOutputValue[i] = new(privacy.Scalar).Sub(randInputValueAll, randOutputValueAll)
		} else {
			randOutputValue[i] = privacy.RandomScalar()
		}

		randOutputSND[i] = privacy.RandomScalar()
		randOutputShardID[i] = privacy.RandomScalar()

		cmOutputValue[i] = privacy.PedCom.CommitAtIndex(new(privacy.Scalar).FromUint64(outputCoin.CoinDetails.GetValue()), randOutputValue[i], privacy.PedersenValueIndex)
		cmOutputSND[i] = privacy.PedCom.CommitAtIndex(outputCoin.CoinDetails.GetSNDerivator(), randOutputSND[i], privacy.PedersenSndIndex)

		receiverShardID := common.GetShardIDFromLastByte(outputCoins[i].CoinDetails.GetPubKeyLastByte())
		cmOutputShardID[i] = privacy.PedCom.CommitAtIndex(new(privacy.Scalar).FromUint64(uint64(receiverShardID)), randOutputShardID[i], privacy.PedersenShardIDIndex)

		randOutputSum[i] = new(privacy.Scalar).FromUint64(0)
		randOutputSum[i].Add(randOutputValue[i], randOutputSND[i])
		randOutputSum[i].Add(randOutputSum[i], randOutputShardID[i])

		cmOutputSum[i] = new(privacy.Point).Identity()
		cmOutputSum[i].Add(cmOutputValue[i], cmOutputSND[i])
		cmOutputSum[i].Add(cmOutputSum[i], outputCoins[i].CoinDetails.GetPublicKey())
		cmOutputSum[i].Add(cmOutputSum[i], cmOutputShardID[i])

		cmOutputValueAll.Add(cmOutputValueAll, cmOutputValue[i])
		randOutputValueAll.Add(randOutputValueAll, randOutputValue[i])

		// calculate final commitment for output coins
		outputCoins[i].CoinDetails.SetCoinCommitment(cmOutputSum[i])
		outputCoins[i].CoinDetails.SetRandomness(randOutputSum[i])

		cmOutputSumAll.Add(cmOutputSumAll, cmOutputSum[i])
	}

	// For Multi Range Protocol
	// proving each output value is less than vmax
	// proving sum of output values is less than vmax
	outputValue := make([]uint64, numOutputCoin)
	for i := 0; i < numOutputCoin; i++ {
		if outputCoins[i].CoinDetails.GetValue() > 0 {
			outputValue[i] = outputCoins[i].CoinDetails.GetValue()
		} else {
			return privacy.NewPrivacyErr(privacy.UnexpectedErr, errors.New("output coin's value is less than 0"))
		}
	}
	if wit.aggregatedRangeWitness == nil {
		wit.aggregatedRangeWitness = new(aggregaterange.AggregatedRangeWitness)
	}
	wit.aggregatedRangeWitness.Set(outputValue, randOutputValue)
	// ---------------------------------------------------

	// save partial commitments (value, input, shardID)
	wit.comOutputValue = cmOutputValue
	wit.comOutputSerialNumberDerivator = cmOutputSND
	wit.comOutputShardID = cmOutputShardID

	return nil
}

// Prove creates big proof
func (wit *PaymentWitness) Prove(hasPrivacy bool) (*PaymentProof, *privacy.PrivacyError) {
	proof := new(PaymentProof)
	proof.Init()

	proof.inputCoins = wit.inputCoins
	proof.outputCoins = wit.outputCoins
	proof.commitmentOutputValue = wit.comOutputValue
	proof.commitmentOutputSND = wit.comOutputSerialNumberDerivator
	proof.commitmentOutputShardID = wit.comOutputShardID

	proof.commitmentInputSecretKey = wit.comInputSecretKey
	proof.commitmentInputValue = wit.comInputValue
	proof.commitmentInputSND = wit.comInputSerialNumberDerivator
	proof.commitmentInputShardID = wit.comInputShardID
	proof.commitmentIndices = wit.commitmentIndices

	// if hasPrivacy == false, don't need to create the zero knowledge proof
	// proving user has spending key corresponding with public key in input coins
	// is proved by signing with spending key
	if !hasPrivacy {
		// Proving that serial number is derived from the committed derivator
		for i := 0; i < len(wit.inputCoins); i++ {
			snNoPrivacyProof, err := wit.serialNumberNoPrivacyWitness[i].Prove(nil)
			if err != nil {
				return nil, privacy.NewPrivacyErr(privacy.ProveSerialNumberNoPrivacyErr, err)
			}
			proof.serialNumberNoPrivacyProof = append(proof.serialNumberNoPrivacyProof, snNoPrivacyProof)
		}
		return proof, nil
	}

	// if hasPrivacy == true
	numInputCoins := len(wit.oneOfManyWitness)

	for i := 0; i < numInputCoins; i++ {
		// Proving one-out-of-N commitments is a commitment to the coins being spent
		oneOfManyProof, err := wit.oneOfManyWitness[i].Prove()
		if err != nil {
			return nil, privacy.NewPrivacyErr(privacy.ProveOneOutOfManyErr, err)
		}
		proof.oneOfManyProof = append(proof.oneOfManyProof, oneOfManyProof)

		// Proving that serial number is derived from the committed derivator
		serialNumberProof, err := wit.serialNumberWitness[i].Prove(nil)
		if err != nil {
			return nil, privacy.NewPrivacyErr(privacy.ProveSerialNumberPrivacyErr, err)
		}
		proof.serialNumberProof = append(proof.serialNumberProof, serialNumberProof)

	}
	var err error

	// Proving that each output values and sum of them does not exceed v_max
	// Update to bulletproof ver 2
	proof.aggregatedRangeProof, err = wit.aggregatedRangeWitness.Prove()
	if err != nil {
		fmt.Println("Error: ", err)
		return nil, privacy.NewPrivacyErr(privacy.ProveAggregatedRangeErr, err)
	}

	if len(proof.inputCoins) == 0 {
		proof.commitmentIndices = nil
		proof.commitmentInputSecretKey = nil
		proof.commitmentInputShardID = nil
		proof.commitmentInputSND = nil
		proof.commitmentInputValue = nil
	}

	if len(proof.outputCoins) == 0 {
		proof.commitmentOutputValue = nil
		proof.commitmentOutputSND = nil
		proof.commitmentOutputShardID = nil
	}

	//privacy.Logger.Log.Debug("Privacy log: PROVING DONE!!!")
	return proof, nil
}
