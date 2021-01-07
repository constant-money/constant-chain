package transaction

import (
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
)

func (tx TxCustomTokenPrivacy) ValidateDoubleSpendWithBlockChain(
	stateDB *statedb.StateDB,
) (bool, error) {
	tokenID := tx.GetTokenID()
	shardID := byte(tx.valEnv.ShardID())
	txNormal := tx.TxPrivacyTokenData.TxNormal
	if tokenID == nil {
		return false, errors.Errorf("TokenID of tx %v is not valid", tx.Hash().String())
	}
	if txNormal.Proof != nil {
		for _, txInput := range txNormal.Proof.GetInputCoins() {
			serialNumber := txInput.CoinDetails.GetSerialNumber().ToBytesS()
			ok, err := statedb.HasSerialNumber(stateDB, *tokenID, serialNumber, shardID)
			if ok || err != nil {
				return false, errors.New("double spend")
			}
		}
		for i, txOutput := range txNormal.Proof.GetOutputCoins() {
			if ok, err := CheckSNDerivatorExistence(tokenID, txOutput.CoinDetails.GetSNDerivator(), stateDB); ok || err != nil {
				if err != nil {
					Logger.log.Error(err)
				}
				Logger.log.Errorf("snd existed: %d\n", i)
				return false, NewTransactionErr(SndExistedError, err, fmt.Sprintf("snd existed: %d\n", i))
			}
		}
	}
	return tx.Tx.ValidateDoubleSpendWithBlockChain(stateDB)
}

func (tx TxCustomTokenPrivacy) ValidateSanityDataByItSelf() (bool, error) {
	if tx.GetType() != common.TxCustomTokenPrivacyType {
		return false, NewTransactionErr(InvalidSanityDataPrivacyTokenError, errors.New("txCustomTokenPrivacy.Tx should have type tp"))
	}
	if tx.TxPrivacyTokenData.TxNormal.GetType() != common.TxNormalType {
		return false, NewTransactionErr(InvalidSanityDataPrivacyTokenError, errors.New("txCustomTokenPrivacy.TxNormal should have type n"))
	}
	meta := tx.Tx.Metadata
	if meta != nil {
		if !metadata.IsAvailableMetaInTxType(meta.GetType(), tx.GetType()) {
			return false, nil
		}
	}

	if tx.TxPrivacyTokenData.TxNormal.GetMetadata() != nil {
		return false, errors.Errorf("This tx field is just used for send token, can not have metadata")
	}
	if tx.TxPrivacyTokenData.PropertyID.String() == common.PRVIDStr {
		return false, NewTransactionErr(InvalidSanityDataPrivacyTokenError, errors.New("TokenID must not be equal PRVID"))
	}

	ok, err := tx.Tx.ValidateSanityDataByItSelf()
	if !ok || err != nil {
		return ok, err
	}

	ok, err = tx.TxPrivacyTokenData.TxNormal.ValidateSanityDataByItSelf()
	if !ok || err != nil {
		return ok, err
	}

	return true, nil
}

func (tx *TxCustomTokenPrivacy) ValidateSanityDataWithBlockchain(
	chainRetriever metadata.ChainRetriever,
	shardViewRetriever metadata.ShardViewRetriever,
	beaconViewRetriever metadata.BeaconViewRetriever,
	beaconHeight uint64,
) (
	bool,
	error,
) {
	// Validate SND???
	// Validate DoubleSpend???
	if tx.Metadata != nil {
		Logger.log.Debug("tx.Metadata.ValidateSanityData")
		isContinued, ok, err := tx.GetMetadata().ValidateSanityData(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight, tx)
		Logger.log.Debug("END tx.Metadata.ValidateSanityData")
		if err != nil || !ok || !isContinued {
			return ok, err
		}
	}
	return true, nil
}

// LoadCommitment do something
func (tx *TxCustomTokenPrivacy) LoadCommitment(
	db *statedb.StateDB,
) error {
	embededTx := tx.Tx
	normalTx := tx.TxPrivacyTokenData.TxNormal
	if embededTx.valEnv.IsPrivacy() {
		tokenID := embededTx.GetTokenID()
		prf := embededTx.Proof
		if prf != nil {
			err := prf.LoadCommitmentFromStateDB(db, tokenID, byte(tx.valEnv.ShardID()))
			if err != nil {
				return err
			}
		}
		// return tx.Proof.LoadCommitmentFromStateDB(db, tokenID, byte(tx.valEnv.ShardID()))
	}
	if normalTx.valEnv.IsPrivacy() {
		tokenID := tx.GetTokenID()
		prf := embededTx.Proof
		if prf != nil {
			err := prf.LoadCommitmentFromStateDB(db, tokenID, byte(tx.valEnv.ShardID()))
			if err != nil {
				return err
			}
		} else {
			return errors.Errorf("Normal tx of Tx CustomeTokenPrivacy can not has no input no outputs")
		}
	}
	return nil
}

func (tx *TxCustomTokenPrivacy) ValidateTxCorrectness(
// transactionStateDB *statedb.StateDB,
) (
	bool,
	error,
) {
	if ok, err := tx.VerifySigTx(); (!ok) || (err != nil) {
		return ok, err
	}

	ok, err := tx.TxPrivacyTokenData.TxNormal.ValidateTxCorrectness()
	if (!ok) || (err != nil) {
		return ok, err
	}
	return tx.Tx.ValidateTxCorrectness()
	//@UNCOMMENT: metrics time
	//elapsed := time.Since(start)
	//Logger.log.Debugf("Validation normal tx %+v in %s time \n", *tx.Hash(), elapsed)

	return true, nil
}

// Todo decoupling this function
func (txN TxCustomTokenPrivacy) validateSanityDataOfProofV2() (bool, error) {
	if txN.Proof != nil {
		if len(txN.Proof.GetInputCoins()) > 255 {
			return false, errors.New("Input coins in tx are very large:" + strconv.Itoa(len(txN.Proof.GetInputCoins())))
		}

		if len(txN.Proof.GetOutputCoins()) > 255 {
			return false, errors.New("Output coins in tx are very large:" + strconv.Itoa(len(txN.Proof.GetOutputCoins())))
		}

		// check doubling a input coin in tx
		serialNumbers := make(map[common.Hash]bool)
		for i, inCoin := range txN.Proof.GetInputCoins() {
			hashSN := common.HashH(inCoin.CoinDetails.GetSerialNumber().ToBytesS())
			if serialNumbers[hashSN] {
				Logger.log.Errorf("Double input in tx - txId %v - index %v", txN.Hash().String(), i)
				return false, errors.New("double input in tx")
			}
			serialNumbers[hashSN] = true
		}

		sndOutputs := make([]*privacy.Scalar, len(txN.Proof.GetOutputCoins()))
		for i, output := range txN.Proof.GetOutputCoins() {
			sndOutputs[i] = output.CoinDetails.GetSNDerivator()
		}
		if privacy.CheckDuplicateScalarArray(sndOutputs) {
			Logger.log.Errorf("Duplicate output coins' snd\n")
			return false, NewTransactionErr(DuplicatedOutputSndError, errors.New("Duplicate output coins' snd\n"))
		}

		isPrivacy := txN.IsPrivacy()

		if isPrivacy {
			// check cmValue of output coins is equal to comValue in Bulletproof
			cmValueOfOutputCoins := txN.Proof.GetCommitmentOutputValue()
			cmValueInBulletProof := txN.Proof.GetAggregatedRangeProof().GetCmValues()
			if len(cmValueOfOutputCoins) != len(cmValueInBulletProof) {
				return false, errors.New("invalid cmValues in Bullet proof")
			}

			if len(txN.Proof.GetInputCoins()) != len(txN.Proof.GetSerialNumberProof()) || len(txN.Proof.GetInputCoins()) != len(txN.Proof.GetOneOfManyProof()) {
				return false, errors.New("the number of input coins must be equal to the number of serialnumber proofs and the number of one-of-many proofs")
			}

			for i := 0; i < len(cmValueOfOutputCoins); i++ {
				if !privacy.IsPointEqual(cmValueOfOutputCoins[i], cmValueInBulletProof[i]) {
					Logger.log.Errorf("cmValue in Bulletproof is not equal to commitment of output's Value - txId %v", txN.Hash().String())
					return false, fmt.Errorf("cmValue %v in Bulletproof is not equal to commitment of output's Value", i)
				}
			}

			if !txN.Proof.GetAggregatedRangeProof().ValidateSanity() {
				return false, errors.New("validate sanity Aggregated range proof failed")
			}

			for i := 0; i < len(txN.Proof.GetOneOfManyProof()); i++ {
				if !txN.Proof.GetOneOfManyProof()[i].ValidateSanity() {
					return false, errors.New("validate sanity One out of many proof failed")
				}
			}

			cmInputSNDs := txN.Proof.GetCommitmentInputSND()
			cmInputSK := txN.Proof.GetCommitmentInputSecretKey()
			for i := 0; i < len(txN.Proof.GetSerialNumberProof()); i++ {
				// check cmSK of input coin is equal to comSK in serial number proof
				if !privacy.IsPointEqual(cmInputSK, txN.Proof.GetSerialNumberProof()[i].GetComSK()) {
					Logger.log.Errorf("ComSK in SNproof is not equal to commitment of private key - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("comSK of SNProof %v is not comSK of input coins", i))
				}

				// check cmSND of input coins is equal to comInputSND in serial number proof
				if !privacy.IsPointEqual(cmInputSNDs[i], txN.Proof.GetSerialNumberProof()[i].GetComInput()) {
					Logger.log.Errorf("cmSND in SNproof is not equal to commitment of input's SND - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("cmSND in SNproof %v is not equal to commitment of input's SND", i))
				}

				// check SN of input coins is equal to the corresponding SN in serial number proof
				if !privacy.IsPointEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber(), txN.Proof.GetSerialNumberProof()[i].GetSN()) {
					Logger.log.Errorf("SN in SNProof is not equal to SN of input coin - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("SN in SNProof %v is not equal to SN of input coin", i))
				}

				if !txN.Proof.GetSerialNumberProof()[i].ValidateSanity() {
					return false, errors.New("validate sanity Serial number proof failed")
				}
			}

			// check input coins with privacy
			for i := 0; i < len(txN.Proof.GetInputCoins()); i++ {
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber().PointValid() {
					return false, errors.New("validate sanity Serial number of input coin failed")
				}
			}
			// check output coins with privacy
			for i := 0; i < len(txN.Proof.GetOutputCoins()); i++ {
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetPublicKey().PointValid() {
					return false, errors.New("validate sanity Public key of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetCoinCommitment().PointValid() {
					return false, errors.New("validate sanity Coin commitment of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetSNDerivator().ScalarValid() {
					return false, errors.New("validate sanity SNDerivator of output coin failed")
				}
			}
			// check ComInputSK
			if !txN.Proof.GetCommitmentInputSecretKey().PointValid() {
				return false, errors.New("validate sanity ComInputSK of proof failed")
			}

			// check SigPubKey
			sigPubKeyPoint, err := new(privacy.Point).FromBytesS(txN.GetSigPubKey())
			if err != nil {
				Logger.log.Errorf("SigPubKey is invalid - txId %v", txN.Hash().String())
				return false, errors.New("SigPubKey is invalid")
			}
			if !privacy.IsPointEqual(cmInputSK, sigPubKeyPoint) {
				Logger.log.Errorf("SigPubKey is not equal to commitment of private key - txId %v", txN.Hash().String())
				return false, errors.New("SigPubKey is not equal to commitment of private key")
			}

			// check ComInputValue
			for i := 0; i < len(txN.Proof.GetCommitmentInputValue()); i++ {
				if !txN.Proof.GetCommitmentInputValue()[i].PointValid() {
					return false, errors.New("validate sanity ComInputValue of proof failed")
				}
			}
			//check ComInputSND
			for i := 0; i < len(txN.Proof.GetCommitmentInputSND()); i++ {
				if !txN.Proof.GetCommitmentInputSND()[i].PointValid() {
					return false, errors.New("validate sanity ComInputSND of proof failed")
				}
			}

			//check ComInputShardID
			if !txN.Proof.GetCommitmentInputShardID().PointValid() {
				return false, errors.New("validate sanity ComInputShardID of proof failed")
			}

			ok, err := txN.Proof.VerifySanityData(txN.valEnv)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}

			// check ComOutputShardID
			for i := 0; i < len(txN.Proof.GetCommitmentOutputShardID()); i++ {
				if !txN.Proof.GetCommitmentOutputShardID()[i].PointValid() {
					return false, errors.New("validate sanity ComOutputShardID of proof failed")
				}
			}
			//check ComOutputSND
			for i := 0; i < len(txN.Proof.GetCommitmentOutputShardID()); i++ {
				if !txN.Proof.GetCommitmentOutputShardID()[i].PointValid() {
					return false, errors.New("validate sanity ComOutputSND of proof failed")
				}
			}
			//check ComOutputValue
			for i := 0; i < len(txN.Proof.GetCommitmentOutputValue()); i++ {
				if !txN.Proof.GetCommitmentOutputValue()[i].PointValid() {
					return false, errors.New("validate sanity ComOutputValue of proof failed")
				}
			}
			if len(txN.Proof.GetCommitmentIndices()) != len(txN.Proof.GetInputCoins())*privacy.CommitmentRingSize {
				return false, errors.New("validate sanity CommitmentIndices of proof failed")

			}
		}

		if !isPrivacy {
			// check SigPubKey
			sigPubKeyPoint, err := new(privacy.Point).FromBytesS(txN.GetSigPubKey())
			if err != nil {
				Logger.log.Errorf("SigPubKey is invalid - txId %v", txN.Hash().String())
				return false, errors.New("SigPubKey is invalid")
			}
			inputCoins := txN.Proof.GetInputCoins()

			if len(inputCoins) != len(txN.Proof.GetSerialNumberNoPrivacyProof()) {
				return false, errors.New("the number of input coins must be equal to the number of serialnumbernoprivacy proofs")
			}

			for i := 0; i < len(inputCoins); i++ {
				// check PublicKey of input coin is equal to SigPubKey
				if !privacy.IsPointEqual(inputCoins[i].CoinDetails.GetPublicKey(), sigPubKeyPoint) {
					Logger.log.Errorf("SigPubKey is not equal to public key of input coins - txId %v", txN.Hash().String())
					return false, errors.New("SigPubKey is not equal to public key of input coins")
				}
			}

			for i := 0; i < len(txN.Proof.GetSerialNumberNoPrivacyProof()); i++ {
				// check PK of input coin is equal to vKey in serial number proof
				if !privacy.IsPointEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetPublicKey(), txN.Proof.GetSerialNumberNoPrivacyProof()[i].GetVKey()) {
					Logger.log.Errorf("VKey in SNNoPrivacyProof is not equal public key of sender - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("VKey of SNNoPrivacyProof %v is not public key of sender", i))
				}

				// check SND of input coins is equal to SND in serial number no privacy proof
				if !privacy.IsScalarEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetSNDerivator(), txN.Proof.GetSerialNumberNoPrivacyProof()[i].GetInput()) {
					Logger.log.Errorf("SND in SNNoPrivacyProof is not equal to input's SND - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("SND in SNNoPrivacyProof %v is not equal to input's SND", i))
				}

				// check SND of input coins is equal to SND in serial number no privacy proof
				if !privacy.IsPointEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber(), txN.Proof.GetSerialNumberNoPrivacyProof()[i].GetOutput()) {
					Logger.log.Errorf("SN in SNNoPrivacyProof is not equal to SN in input coin - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("SN in SNNoPrivacyProof %v is not equal to SN in input coin", i))
				}

				if !txN.Proof.GetSerialNumberNoPrivacyProof()[i].ValidateSanity() {
					return false, errors.New("validate sanity Serial number no privacy proof failed")
				}
			}
			// check input coins without privacy
			for i := 0; i < len(txN.Proof.GetInputCoins()); i++ {
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetCoinCommitment().PointValid() {
					return false, errors.New("validate sanity CoinCommitment of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetPublicKey().PointValid() {
					return false, errors.New("validate sanity PublicKey of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber().PointValid() {
					return false, errors.New("validate sanity Serial number of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetRandomness().ScalarValid() {
					return false, errors.New("validate sanity Randomness of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetSNDerivator().ScalarValid() {
					return false, errors.New("validate sanity SNDerivator of input coin failed")
				}
			}

			// check output coins without privacy
			for i := 0; i < len(txN.Proof.GetOutputCoins()); i++ {
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetCoinCommitment().PointValid() {
					return false, errors.New("validate sanity CoinCommitment of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetPublicKey().PointValid() {
					return false, errors.New("validate sanity PublicKey of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetRandomness().ScalarValid() {
					return false, errors.New("validate sanity Randomness of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetSNDerivator().ScalarValid() {
					return false, errors.New("validate sanity SNDerivator of output coin failed")
				}
			}
		}
	}
	return true, nil
}
