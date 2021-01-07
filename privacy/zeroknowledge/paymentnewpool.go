package zkp

import (
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
)

func (proof *PaymentProof) LoadCommitmentFromStateDB(db *statedb.StateDB, tokenID *common.Hash, shardID byte) error {
	privacy.Logger.Log.Infof("[testperformance] LoadCommitmentFromStateDB, tokenID %v, shardID %v", tokenID.String(), shardID)
	cmInputSum := make([]*privacy.Point, len(proof.oneOfManyProof))
	for i := 0; i < len(proof.oneOfManyProof); i++ {
		// privacy.Logger.Log.Debugf("[TEST] input coins %v\n ShardID %v fee %v", i, shardID, fee)
		// privacy.Logger.Log.Debugf("[TEST] commitments indices %v\n", proof.commitmentIndices[i*privacy.CommitmentRingSize:i*privacy.CommitmentRingSize+8])
		// Verify for the proof one-out-of-N commitments is a commitment to the coins being spent
		// Calculate cm input sum
		cmInputSum[i] = new(privacy.Point).Add(proof.commitmentInputSecretKey, proof.commitmentInputValue[i])
		cmInputSum[i].Add(cmInputSum[i], proof.commitmentInputSND[i])
		cmInputSum[i].Add(cmInputSum[i], proof.commitmentInputShardID)

		// get commitments list from CommitmentIndices
		commitments := make([]*privacy.Point, privacy.CommitmentRingSize)
		for j := 0; j < privacy.CommitmentRingSize; j++ {
			index := proof.commitmentIndices[i*privacy.CommitmentRingSize+j]
			commitmentBytes, err := statedb.GetCommitmentByIndex(db, *tokenID, index, shardID)
			privacy.Logger.Log.Debugf("[TEST] commitment at index %v: %v\n", index, commitmentBytes)
			if err != nil {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF 1: Error when get commitment by index from database", index, err)
				return privacy.NewPrivacyErr(privacy.VerifyOneOutOfManyProofFailedErr, err)
			}
			recheckIndex, err := statedb.GetCommitmentIndex(db, *tokenID, commitmentBytes, shardID)
			if err != nil || recheckIndex.Uint64() != index {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF 2: Error when get commitment by index from database", index, err)
				return privacy.NewPrivacyErr(privacy.VerifyOneOutOfManyProofFailedErr, err)
			}
			commitments[j], err = new(privacy.Point).FromBytesS(commitmentBytes)
			if err != nil {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Cannot decompress commitment from database", index, err)
				return privacy.NewPrivacyErr(privacy.VerifyOneOutOfManyProofFailedErr, err)
			}
			commitments[j].Sub(commitments[j], cmInputSum[i])
			if err != nil {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Cannot sub commitment to sum of commitment inputs", index, err)
				return privacy.NewPrivacyErr(privacy.VerifyOneOutOfManyProofFailedErr, err)
			}
		}

		proof.oneOfManyProof[i].Statement.Commitments = commitments

	}
	return nil
}

// Validate all of conditions
func (proof PaymentProof) VerifySanityData(
	vEnv privacy.ValidationEnviroment,
) (
	bool,
	error,
) {
	senderSID := vEnv.ShardID()
	if IsNewZKP(vEnv.BeaconHeight()) {
		expectedCMShardID := privacy.PedCom.CommitAtIndex(
			new(privacy.Scalar).FromUint64(uint64(senderSID)),
			fixedRandomnessShardID,
			privacy.PedersenShardIDIndex,
		)
		if !privacy.IsPointEqual(expectedCMShardID, proof.GetCommitmentInputShardID()) {
			return false, errors.New("ComInputShardID must be committed with the fixed randomness")
		}
	}

	return false, nil
}

func (proof PaymentProof) VerifyV2(
	vEnv privacy.ValidationEnviroment,
	pubKey privacy.PublicKey,
	fee uint64,
	// shardID byte,
	// tokenID *common.Hash,
) (
	bool,
	error,
) {
	if !vEnv.IsPrivacy() {
		return proof.verifyNoPrivacyV2(pubKey, fee, vEnv)
	}
	return proof.verifyHasPrivacyV2(pubKey, fee, vEnv)
}

func (proof PaymentProof) verifyNoPrivacyV2(
	pubKey privacy.PublicKey,
	fee uint64,
	// shardID byte,
	// tokenID *common.Hash,
	vEnv privacy.ValidationEnviroment,
) (
	bool,
	error,
) {
	var sumInputValue, sumOutputValue uint64
	sumInputValue = 0
	sumOutputValue = 0

	pubKeyLastByteSender := pubKey[len(pubKey)-1]
	senderShardID := common.GetShardIDFromLastByte(pubKeyLastByteSender)
	cmShardIDSender := new(privacy.Point)
	cmShardIDSender.ScalarMult(privacy.PedCom.G[privacy.PedersenShardIDIndex], new(privacy.Scalar).FromBytes([privacy.Ed25519KeySize]byte{senderShardID}))

	isNewZKP := IsNewZKP(vEnv.BeaconHeight())

	for i := 0; i < len(proof.inputCoins); i++ {
		if isNewZKP {
			// Check input coins' Serial number is created from input coins' input and sender's spending key
			valid, err := proof.serialNumberNoPrivacyProof[i].Verify(nil)
			if !valid {
				privacy.Logger.Log.Errorf("Verify serial number no privacy proof failed")
				return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, err)
			}
		} else {
			// Check input coins' Serial number is created from input coins' input and sender's spending key
			valid, err := proof.serialNumberNoPrivacyProof[i].VerifyOld(nil)
			if !valid {
				privacy.Logger.Log.Errorf("Verify serial number no privacy proof failed")
				return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, err)
			}
		}

		// Check input coins' cm is calculated correctly
		cmSK := proof.inputCoins[i].CoinDetails.GetPublicKey()
		cmValue := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenValueIndex], new(privacy.Scalar).FromUint64(proof.inputCoins[i].CoinDetails.GetValue()))
		cmSND := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenSndIndex], proof.inputCoins[i].CoinDetails.GetSNDerivator())
		cmRandomness := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], proof.inputCoins[i].CoinDetails.GetRandomness())
		cmTmp := new(privacy.Point).Add(cmSK, cmValue)
		cmTmp.Add(cmTmp, cmSND)
		cmTmp.Add(cmTmp, cmShardIDSender)
		cmTmp.Add(cmTmp, cmRandomness)

		if !privacy.IsPointEqual(cmTmp, proof.inputCoins[i].CoinDetails.GetCoinCommitment()) {
			privacy.Logger.Log.Errorf("Input coins %v commitment wrong!\n", i)
			return false, privacy.NewPrivacyErr(privacy.VerifyCoinCommitmentInputFailedErr, nil)
		}

		// Calculate sum of input values
		sumInputValue += proof.inputCoins[i].CoinDetails.GetValue()
	}

	for i := 0; i < len(proof.outputCoins); i++ {
		// Check output coins' cm is calculated correctly
		shardID := common.GetShardIDFromLastByte(proof.outputCoins[i].CoinDetails.GetPubKeyLastByte())
		cmSK := proof.outputCoins[i].CoinDetails.GetPublicKey()
		cmValue := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenValueIndex], new(privacy.Scalar).FromUint64(proof.outputCoins[i].CoinDetails.GetValue()))
		cmSND := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenSndIndex], proof.outputCoins[i].CoinDetails.GetSNDerivator())
		cmShardID := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenShardIDIndex], new(privacy.Scalar).FromBytes([privacy.Ed25519KeySize]byte{shardID}))
		cmRandomness := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], proof.outputCoins[i].CoinDetails.GetRandomness())

		cmTmp := new(privacy.Point).Add(cmSK, cmValue)
		cmTmp.Add(cmTmp, cmSND)
		cmTmp.Add(cmTmp, cmShardID)
		cmTmp.Add(cmTmp, cmRandomness)

		if !privacy.IsPointEqual(cmTmp, proof.outputCoins[i].CoinDetails.GetCoinCommitment()) {
			privacy.Logger.Log.Errorf("Output coins %v commitment wrong!\n", i)
			return false, privacy.NewPrivacyErr(privacy.VerifyCoinCommitmentOutputFailedErr, nil)
		}
	}

	//Calculate sum of output values and check overflow output's value
	if len(proof.outputCoins) > 0 {
		sumOutputValue = proof.outputCoins[0].CoinDetails.GetValue()
		for i := 1; i < len(proof.outputCoins); i++ {
			outValue := proof.outputCoins[i].CoinDetails.GetValue()
			sumTmp := sumOutputValue + outValue
			if sumTmp < sumOutputValue || sumTmp < outValue {
				return false, privacy.NewPrivacyErr(privacy.UnexpectedErr, fmt.Errorf("Overflow output value %v\n", outValue))
			}
			sumOutputValue += outValue
		}
	}

	// check overflow fee value
	tmp := sumOutputValue + fee
	if tmp < sumOutputValue || tmp < fee {
		return false, privacy.NewPrivacyErr(privacy.UnexpectedErr, fmt.Errorf("Overflow fee value %v\n", fee))
	}

	// check if sum of input values equal sum of output values
	if sumInputValue != sumOutputValue+fee {
		privacy.Logger.Log.Debugf("sumInputValue: %v\n", sumInputValue)
		privacy.Logger.Log.Debugf("sumOutputValue: %v\n", sumOutputValue)
		privacy.Logger.Log.Debugf("fee: %v\n", fee)
		privacy.Logger.Log.Errorf("Sum of inputs is not equal sum of output!\n")
		return false, privacy.NewPrivacyErr(privacy.VerifyAmountNoPrivacyFailedErr, nil)
	}
	return true, nil
}

func (proof PaymentProof) verifyHasPrivacyV2(
	pubKey privacy.PublicKey,
	fee uint64,
	// stateDB *statedb.StateDB,
	// shardID byte,
	// tokenID *common.Hash,
	vEnv privacy.ValidationEnviroment,
) (
	bool,
	error,
) {

	isNewZKP := IsNewZKP(vEnv.BeaconHeight())
	shardID := byte(vEnv.ShardID())
	for i := 0; i < len(proof.oneOfManyProof); i++ {
		privacy.Logger.Log.Debugf("[TEST] input coins %v\n ShardID %v fee %v", i, shardID, fee)
		privacy.Logger.Log.Debugf("[TEST] commitments indices %v\n", proof.commitmentIndices[i*privacy.CommitmentRingSize:i*privacy.CommitmentRingSize+8])
		// Verify for the proof one-out-of-N commitments is a commitment to the coins being spent

		if isNewZKP {
			valid, err := proof.oneOfManyProof[i].Verify()
			if !valid {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: One out of many failed")
				return false, privacy.NewPrivacyErr(privacy.VerifyOneOutOfManyProofFailedErr, err)
			}
			// Verify for the Proof that input coins' serial number is derived from the committed derivator
			valid, err = proof.serialNumberProof[i].Verify(nil)
			if !valid {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Serial number privacy failed")
				return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, err)
			}
		} else {
			valid, err := proof.oneOfManyProof[i].VerifyOld()
			if !valid {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: One out of many failed")
				return false, privacy.NewPrivacyErr(privacy.VerifyOneOutOfManyProofFailedErr, err)
			}
			// Verify for the Proof that input coins' serial number is derived from the committed derivator
			valid, err = proof.serialNumberProof[i].VerifyOld(nil)
			if !valid {
				privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Serial number privacy failed")
				return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, err)
			}
		}
	}

	// Check output coins' cm is calculated correctly
	for i := 0; i < len(proof.outputCoins); i++ {
		cmTmp := new(privacy.Point).Add(proof.outputCoins[i].CoinDetails.GetPublicKey(), proof.commitmentOutputValue[i])
		cmTmp.Add(cmTmp, proof.commitmentOutputSND[i])
		cmTmp.Add(cmTmp, proof.commitmentOutputShardID[i])

		if !privacy.IsPointEqual(cmTmp, proof.outputCoins[i].CoinDetails.GetCoinCommitment()) {
			privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Commitment for output coins are not computed correctly")
			return false, privacy.NewPrivacyErr(privacy.VerifyCoinCommitmentOutputFailedErr, nil)
		}
	}

	// Verify the proof that output values and sum of them do not exceed v_max
	// if !isBatch {
	valid, err := proof.aggregatedRangeProof.Verify()
	if !valid {
		privacy.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Multi-range failed")
		return false, privacy.NewPrivacyErr(privacy.VerifyAggregatedProofFailedErr, err)
	}
	// }

	// Verify the proof that sum of all input values is equal to sum of all output values
	comInputValueSum := new(privacy.Point).Identity()
	for i := 0; i < len(proof.commitmentInputValue); i++ {
		comInputValueSum.Add(comInputValueSum, proof.commitmentInputValue[i])
	}

	comOutputValueSum := new(privacy.Point).Identity()
	for i := 0; i < len(proof.commitmentOutputValue); i++ {
		comOutputValueSum.Add(comOutputValueSum, proof.commitmentOutputValue[i])
	}

	if fee > 0 {
		comOutputValueSum.Add(comOutputValueSum, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenValueIndex], new(privacy.Scalar).FromUint64(uint64(fee))))
	}

	privacy.Logger.Log.Infof("comInputValueSum: %v\n", comInputValueSum.ToBytesS())
	privacy.Logger.Log.Infof("comOutputValueSum: %v\n", comOutputValueSum.ToBytesS())

	if !privacy.IsPointEqual(comInputValueSum, comOutputValueSum) {
		privacy.Logger.Log.Debugf("comInputValueSum: ", comInputValueSum)
		privacy.Logger.Log.Debugf("comOutputValueSum: ", comOutputValueSum)
		privacy.Logger.Log.Error("VERIFICATION PAYMENT PROOF: Sum of input coins' value is not equal to sum of output coins' value")
		return false, privacy.NewPrivacyErr(privacy.VerifyAmountPrivacyFailedErr, nil)
	}

	return true, nil
}
