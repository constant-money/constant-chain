package transaction

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
)

func (tx Tx) ValidateDoubleSpendWithBlockChain(
	stateDB *statedb.StateDB,
) (bool, error) {
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return false, err
	}
	shardID := byte(tx.valEnv.ShardID())

	if tx.Proof != nil {
		for _, txInput := range tx.Proof.GetInputCoins() {
			serialNumber := txInput.CoinDetails.GetSerialNumber().ToBytesS()
			ok, err := statedb.HasSerialNumber(stateDB, *tokenID, serialNumber, shardID)
			if ok || err != nil {
				return false, errors.New("double spend")
			}
		}
		for i, txOutput := range tx.Proof.GetOutputCoins() {
			if ok, err := CheckSNDerivatorExistence(tokenID, txOutput.CoinDetails.GetSNDerivator(), stateDB); ok || err != nil {
				if err != nil {
					Logger.log.Error(err)
				}
				Logger.log.Errorf("snd existed: %d\n", i)
				return false, NewTransactionErr(SndExistedError, err, fmt.Sprintf("snd existed: %d\n", i))
			}
		}

	}
	return true, nil
}

func (tx Tx) ValidateSanityDataByItSelf() (bool, error) {
	switch tx.Type {
	case common.TxNormalType, common.TxRewardType, common.TxCustomTokenPrivacyType, common.TxReturnStakingType: //is valid
	default:
		return false, NewTransactionErr(RejectTxType, fmt.Errorf("wrong tx type with %s", tx.Type))
	}

	// check info field
	if len(tx.Info) > 512 {
		return false, NewTransactionErr(RejectTxInfoSize, fmt.Errorf("wrong tx info length %d bytes, only support info with max length <= %d bytes", len(tx.Info), 512))
	}
	if tx.Metadata != nil {
		if ((tx.Metadata.GetType() == metadata.ReturnStakingMeta) != (tx.valEnv.TxType() == common.TxReturnStakingType)) ||
			((tx.Metadata.GetType() == metadata.WithDrawRewardResponseMeta) != (tx.valEnv.TxType() == common.TxRewardType)) {
			return false, errors.Errorf("Not mismatch Type, txType: %v, metadataType %v", tx.valEnv.TxType(), tx.Metadata.GetType())
		}
	}

	// check tx size
	actualTxSize := tx.GetTxActualSize()
	if actualTxSize > common.MaxTxSize {
		//fmt.Print(actualTxSize, common.MaxTxSize)
		return false, NewTransactionErr(RejectTxSize, fmt.Errorf("tx size %d kB is too large", actualTxSize))
	}

	//check version
	if tx.Version > txVersion {
		return false, NewTransactionErr(RejectTxVersion, fmt.Errorf("tx version is %d. Wrong version tx. Only support for version >= %d", tx.Version, txVersion))
	}
	// check LockTime before now
	if int64(tx.LockTime) > time.Now().Unix() {
		return false, NewTransactionErr(RejectInvalidLockTime, fmt.Errorf("wrong tx locktime %d", tx.LockTime))
	}

	if len(tx.SigPubKey) != common.SigPubKeySize {
		return false, NewTransactionErr(RejectTxPublickeySigSize, fmt.Errorf("wrong tx Sig PK size %d", len(tx.SigPubKey)))
	}

	metaData := tx.GetMetadata()
	if metaData != nil {
		if !metaData.ValidateMetadataByItself() {
			return false, errors.Errorf("Metadata is not valid")
		}
	}

	if tx.GetProof() == nil {
		if metaData == nil {
			return false, NewTransactionErr(RejectTxType, fmt.Errorf("This tx %v has no proof, but metadata is nil", tx.Hash().String()))
		} else {
			metaType := metaData.GetType()
			if !metadata.NoInputNoOutput(metaType) {
				return false, NewTransactionErr(RejectTxType, fmt.Errorf("This tx %v has no proof, but metadata is invalid, metadata type %v", tx.Hash().String(), metaType))
			}
		}
	} else {
		proof := tx.GetProof()
		if len(proof.GetInputCoins()) == 0 {
			if metaData == nil {
				return false, NewTransactionErr(RejectTxType, fmt.Errorf("This tx %v has no input, but metadata is nil", tx.Hash().String()))
			} else {
				metaType := metaData.GetType()
				if !metadata.NoInputHasOutput(metaType) {
					return false, NewTransactionErr(RejectTxType, fmt.Errorf("This tx %v has no proof, but metadata is invalid, metadata type %v", tx.Hash().String(), metaType))
				}
			}
		}
		if len(proof.GetOutputCoins()) == 0 {
			if metaData == nil {
				return false, NewTransactionErr(RejectTxType, fmt.Errorf("This tx %v has no input, but metadata is nil", tx.Hash().String()))
			} else {
				metaType := metaData.GetType()
				if !metadata.HasInputNoOutput(metaType) {
					return false, NewTransactionErr(RejectTxType, fmt.Errorf("This tx %v has no proof, but metadata is invalid, metadata type %v", tx.Hash().String(), metaType))
				}
			}
		}
		// check sanity of Proof
		validateSanityOfProof, err := tx.validateSanityDataOfProofV2()
		if err != nil || !validateSanityOfProof {
			return false, err
		}
	}

	return true, nil
}

func (tx *Tx) ValidateSanityDataWithBlockchain(
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
		isContinued, ok, err := tx.Metadata.ValidateSanityData(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight, tx)
		Logger.log.Debug("END tx.Metadata.ValidateSanityData")
		if err != nil || !ok || !isContinued {
			return ok, err
		}
	}
	return true, nil
}

// func (tx *Tx) ValidateSanityDataWithBlockchain(

func (tx *Tx) LoadCommitment(
	db *statedb.StateDB,
) error {
	prf := tx.Proof
	if prf == nil {
		meta := tx.GetMetadata()
		if meta == nil {
			return errors.Errorf("This tx has no proof and not a tx for pay fee or tx with metadata")
		}
		if meta != nil {
			if metadata.NoInputNoOutput(meta.GetType()) || metadata.NoInputHasOutput(meta.GetType()) {
				return nil
			} else {
				return errors.Errorf("Invalid tx")
			}
		}
	}
	tokenID := tx.GetTokenID()
	if tx.valEnv.IsPrivacy() {
		return tx.Proof.LoadCommitmentFromStateDB(db, tokenID, byte(tx.valEnv.ShardID()))
	} else {
		for i := 0; i < len(tx.Proof.GetInputCoins()); i++ {
			ok, err := tx.CheckCMExistence(
				tx.Proof.GetInputCoins()[i].CoinDetails.GetCoinCommitment().ToBytesS(),
				db,
				byte(tx.valEnv.ShardID()),
				tokenID,
			)
			if !ok || err != nil {
				if err != nil {
					Logger.log.Error(err)
				}
				return NewTransactionErr(InputCommitmentIsNotExistedError, err)
			}
		}
	}
	return nil
}

func (tx *Tx) ValidateTxCorrectness(
// transactionStateDB *statedb.StateDB,
) (
	bool,
	error,
) {
	if ok, err := tx.VerifySigTx(); (!ok) || (err != nil) {
		return ok, err
	}

	// Todo Moving out
	Logger.log.Debugf("VALIDATING TX........\n")
	// if tx.IsSalaryTx() {
	// 	return tx.ValidateTxSalary(transactionStateDB)
	// }
	// hasPrivacy := tx.IsPrivacy()
	var valid bool
	var err error

	//Todo find out how to validate safer
	if tx.GetType() == common.TxReturnStakingType {
		return true, nil //
	}

	if tx.Proof != nil {
		// if tokenID == nil {
		// 	tokenID = &common.Hash{}
		// 	err := tokenID.SetBytes(common.PRVCoinID[:])
		// 	if err != nil {
		// 		Logger.log.Error(err)
		// 		return false, NewTransactionErr(TokenIDInvalidError, err, tokenID.String())
		// 	}
		// }

		/*----------- TODO Moving out --------------

		// if !tx.valEnv.IsPrivacy() {
			// Check input coins' commitment is exists in cm list (Database)

		// }
		------------------------------------------ */
		// Verify the payment proof

		valid, err = tx.Proof.VerifyV2(tx.valEnv, tx.SigPubKey, tx.Fee)
		if !valid {
			if err != nil {
				Logger.log.Error(err)
			}
			Logger.log.Error("FAILED VERIFICATION PAYMENT PROOF")
			err1, ok := err.(*privacy.PrivacyError)
			if ok {
				// parse error detail
				if err1.Code == privacy.ErrCodeMessage[privacy.VerifyOneOutOfManyProofFailedErr].Code {
					// if isNewTransaction {
					// 	return false, NewTransactionErr(VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
					// } else {
					// for old txs which be get from sync block or validate new block
					if tx.LockTime <= ValidateTimeForOneoutOfManyProof {
						// only verify by sign on block because of issue #504(that mean we should pass old tx, which happen before this issue)
						return true, nil
					} else {
						return false, NewTransactionErr(VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
					}
					// }
				}
			}
			return false, NewTransactionErr(TxProofVerifyFailError, err, tx.Hash().String())
		} else {
			Logger.log.Debugf("SUCCESSED VERIFICATION PAYMENT PROOF ")
		}
	}
	//@UNCOMMENT: metrics time
	//elapsed := time.Since(start)
	//Logger.log.Debugf("Validation normal tx %+v in %s time \n", *tx.Hash(), elapsed)

	return true, nil
}

func (txN Tx) checkDuplicateInput() (bool, error) {
	if len(txN.Proof.GetInputCoins()) > 255 {
		return false, errors.New("Input coins in tx are very large:" + strconv.Itoa(len(txN.Proof.GetInputCoins())))
	}

	// check doubling a input coin in tx
	serialNumbers := make(map[[privacy.Ed25519KeySize]byte]struct{})
	for i, inCoin := range txN.Proof.GetInputCoins() {
		snBytes := inCoin.CoinDetails.GetSerialNumber().ToBytes()
		if _, ok := serialNumbers[snBytes]; ok {
			Logger.log.Errorf("Double input in tx - txId %v - index %v", txN.Hash().String(), i)
			return false, errors.New("double input in tx")
		}
		serialNumbers[snBytes] = struct{}{}
	}
	return true, nil
}

func (txN Tx) checkDuplicateOutput() (bool, error) {
	if len(txN.Proof.GetOutputCoins()) > 255 {
		return false, errors.New("Output coins in tx are very large:" + strconv.Itoa(len(txN.Proof.GetOutputCoins())))
	}

	sndOutputs := make(map[[privacy.Ed25519KeySize]byte]struct{})
	for i, output := range txN.Proof.GetOutputCoins() {
		sndBytes := output.CoinDetails.GetSNDerivator().ToBytes()
		if _, ok := sndOutputs[sndBytes]; ok {
			Logger.log.Errorf("Double output in tx - txId %v - index %v", txN.Hash().String(), i)
			return false, NewTransactionErr(DuplicatedOutputSndError, errors.New("Duplicate output coins' snd\n"))
		}
		sndOutputs[sndBytes] = struct{}{}
	}
	return true, nil
}

func (txN Tx) validateInputPrivacy() (bool, error) {
	prf := txN.Proof
	cmInputSK := prf.GetCommitmentInputSecretKey()
	if !cmInputSK.PointValid() {
		return false, errors.New("validate sanity ComInputSK of proof failed")
	}
	sigPubKeyPoint, err := new(privacy.Point).FromBytesS(txN.GetSigPubKey())
	if err != nil {
		Logger.log.Errorf("SigPubKey is invalid - txId %v", txN.Hash().String())
		return false, errors.New("SigPubKey is invalid")
	}
	if !privacy.IsPointEqual(cmInputSK, sigPubKeyPoint) {
		Logger.log.Errorf("SigPubKey is not equal to commitment of private key - txId %v", txN.Hash().String())
		return false, errors.New("SigPubKey is not equal to commitment of private key")
	}
	if !prf.GetCommitmentInputShardID().PointValid() {
		return false, errors.New("validate sanity ComInputShardID of proof failed")
	}
	cmInputSNDs := prf.GetCommitmentInputSND()
	cmInputValue := prf.GetCommitmentInputValue()
	if (len(cmInputSNDs) != len(cmInputValue)) || (len(cmInputSNDs) != len(prf.GetInputCoins())) {
		return false, errors.Errorf("Len Commitment input SND %v and Commitment input value %v and len input coins %v is not equal", len(cmInputSNDs), len(cmInputValue), len(prf.GetInputCoins()))
	}
	for i, iCoin := range prf.GetInputCoins() {
		if !iCoin.CoinDetails.GetSerialNumber().PointValid() {
			return false, errors.New("validate sanity Serial number of input coin failed")
		}
		if iCoin.CoinDetails.GetCoinCommitment() != nil {
			return false, errors.New("CoinCommitment of input coin is not nil")
		}
		if iCoin.CoinDetails.GetPublicKey() != nil {
			return false, errors.New("PublicKey of input coin is not nil")
		}
		if iCoin.CoinDetails.GetRandomness() != nil {
			return false, errors.New("Randomness of input coin is not nil")
		}
		if iCoin.CoinDetails.GetSNDerivator() != nil {
			return false, errors.New("SNDerivator of input coin is not nil")
		}

		if !cmInputValue[i].PointValid() {
			return false, errors.New("validate sanity ComInputValue of proof failed")
		}
		if !cmInputSNDs[i].PointValid() {
			return false, errors.New("validate sanity ComInputValue of proof failed")
		}
	}

	return true, nil
}

func (txN Tx) validateInputNoPrivacy() (bool, error) {
	prf := txN.Proof
	inputCoins := prf.GetInputCoins()
	sigPubKeyPoint, err := new(privacy.Point).FromBytesS(txN.GetSigPubKey())
	if err != nil {
		Logger.log.Errorf("SigPubKey is invalid - txId %v", txN.Hash().String())
		return false, errors.New("SigPubKey is invalid")
	}
	for _, iCoin := range inputCoins {
		if !iCoin.CoinDetails.GetSerialNumber().PointValid() {
			return false, errors.New("validate sanity Serial number of input coin failed")
		}
		if !iCoin.CoinDetails.GetCoinCommitment().PointValid() {
			return false, errors.New("validate sanity CoinCommitment of input coin failed")
		}
		if !iCoin.CoinDetails.GetPublicKey().PointValid() {
			return false, errors.New("validate sanity PublicKey of input coin failed")
		}
		if !iCoin.CoinDetails.GetRandomness().ScalarValid() {
			return false, errors.New("validate sanity Randomness of input coin failed")
		}
		if !iCoin.CoinDetails.GetSNDerivator().ScalarValid() {
			return false, errors.New("validate sanity SNDerivator of input coin failed")
		}
		if !privacy.IsPointEqual(iCoin.CoinDetails.GetPublicKey(), sigPubKeyPoint) {
			Logger.log.Errorf("SigPubKey is not equal to public key of input coins - txId %v", txN.Hash().String())
			return false, errors.New("SigPubKey is not equal to public key of input coins")
		}
	}

	return true, nil
}

func (txN Tx) validateOutputPrivacy() (bool, error) {
	prf := txN.Proof
	cmOutputValue := prf.GetCommitmentOutputValue()
	cmOutSNDs := prf.GetCommitmentOutputSND()
	cmOutSIDs := prf.GetCommitmentOutputShardID()

	for i, oCoin := range prf.GetOutputCoins() {
		if !oCoin.CoinDetails.GetPublicKey().PointValid() {
			return false, errors.New("validate sanity Public key of output coin failed")
		}
		if !oCoin.CoinDetails.GetCoinCommitment().PointValid() {
			return false, errors.New("validate sanity Coin commitment of output coin failed")
		}
		if !oCoin.CoinDetails.GetSNDerivator().ScalarValid() {
			return false, errors.New("validate sanity SNDerivator of output coin failed")
		}

		if oCoin.CoinDetails.GetRandomness() != nil {
			return false, errors.New("Randomness of output coin is not nil")
		}
		if oCoin.CoinDetails.GetSerialNumber() != nil {
			return false, errors.New("SerialNumber of output coin is not nil")
		}

		if !cmOutSIDs[i].PointValid() {
			return false, errors.New("validate sanity ComOutputShardID of proof failed")
		}
		if !cmOutSNDs[i].PointValid() {
			return false, errors.New("validate sanity ComOutputSND of proof failed")
		}
		if !cmOutputValue[i].PointValid() {
			return false, errors.New("validate sanity ComOutputValue of proof failed")
		}
	}

	return true, nil
}

func (txN Tx) validateOutputNoPrivacy() (bool, error) {
	prf := txN.Proof
	for _, oCoin := range prf.GetOutputCoins() {
		if !oCoin.CoinDetails.GetCoinCommitment().PointValid() {
			return false, errors.New("validate sanity CoinCommitment of output coin failed")
		}
		if !oCoin.CoinDetails.GetPublicKey().PointValid() {
			return false, errors.New("validate sanity PublicKey of output coin failed")
		}
		if !oCoin.CoinDetails.GetRandomness().ScalarValid() {
			return false, errors.New("validate sanity Randomness of output coin failed")
		}
		if !oCoin.CoinDetails.GetSNDerivator().ScalarValid() {
			return false, errors.New("validate sanity SNDerivator of output coin failed")
		}
		if oCoin.CoinDetails.GetSerialNumber() != nil {
			return false, errors.New("SerialNumber of output coin is not nil")
		}
	}
	return true, nil
}

func (txN Tx) validatePrivacyZKPSanityWithInput() (bool, error) {
	prf := txN.Proof
	inputCoins := prf.GetInputCoins()
	oomProofs := prf.GetOneOfManyProof()
	snProofs := prf.GetSerialNumberProof()
	snNoPrivacyProofs := prf.GetSerialNumberNoPrivacyProof()
	if len(snNoPrivacyProofs) != 0 {
		return false, errors.Errorf("This is privacy tx, no NoPrivacy ZKP")
	}
	cmInputSNDs := prf.GetCommitmentInputSND()
	cmInputSK := prf.GetCommitmentInputSecretKey()
	if (len(inputCoins) != len(snProofs)) || (len(inputCoins) != len(oomProofs)) {
		return false, errors.New("the number of input coins must be equal to the number of serialnumber proofs and the number of one-of-many proofs")
	}
	for _, oomPrf := range oomProofs {
		if !oomPrf.ValidateSanity() {
			return false, errors.New("validate sanity One out of many proof failed")
		}
	}
	for i, snProof := range snProofs {
		if !snProof.ValidateSanity() {
			return false, errors.New("validate sanity Serial number proof failed")
		}
		if !privacy.IsPointEqual(snProof.GetComSK(), cmInputSK) {
			Logger.log.Errorf("ComSK in SNproof is not equal to commitment of private key - txId %v", txN.Hash().String())
			return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("comSK of SNProof %v is not comSK of input coins", i))
		}
		if !privacy.IsPointEqual(snProof.GetComInput(), cmInputSNDs[i]) {
			Logger.log.Errorf("cmSND in SNproof is not equal to commitment of input's SND - txId %v", txN.Hash().String())
			return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("cmSND in SNproof %v is not equal to commitment of input's SND", i))
		}
		if !privacy.IsPointEqual(inputCoins[i].CoinDetails.GetSerialNumber(), snProof.GetSN()) {
			Logger.log.Errorf("SN in SNProof is not equal to SN of input coin - txId %v", txN.Hash().String())
			return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("SN in SNProof %v is not equal to SN of input coin", i))
		}
	}
	return true, nil
}

func (txN Tx) validateNoPrivacyZKPSanityWithInput() (bool, error) {
	prf := txN.Proof
	inputCoins := prf.GetInputCoins()
	snProofs := txN.Proof.GetSerialNumberNoPrivacyProof()
	oomProofs := prf.GetOneOfManyProof()
	snPrivacyProofs := prf.GetSerialNumberProof()
	if (len(snPrivacyProofs) != 0) || (len(oomProofs) != 0) {
		return false, errors.Errorf("This is tx no privacy, no privacy zkp")
	}
	for i, snPrf := range snProofs {
		if !snPrf.ValidateSanity() {
			return false, errors.New("validate sanity Serial number no privacy proof failed")
		}
		if !privacy.IsPointEqual(inputCoins[i].CoinDetails.GetPublicKey(), snPrf.GetVKey()) {
			Logger.log.Errorf("VKey in SNNoPrivacyProof is not equal public key of sender - txId %v", txN.Hash().String())
			return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("VKey of SNNoPrivacyProof %v is not public key of sender", i))
		}
		if !privacy.IsScalarEqual(inputCoins[i].CoinDetails.GetSNDerivator(), snPrf.GetInput()) {
			Logger.log.Errorf("SND in SNNoPrivacyProof is not equal to input's SND - txId %v", txN.Hash().String())
			return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("SND in SNNoPrivacyProof %v is not equal to input's SND", i))
		}
		if !privacy.IsPointEqual(inputCoins[i].CoinDetails.GetSerialNumber(), snPrf.GetOutput()) {
			Logger.log.Errorf("SN in SNNoPrivacyProof is not equal to SN in input coin - txId %v", txN.Hash().String())
			return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("SN in SNNoPrivacyProof %v is not equal to SN in input coin", i))
		}
	}
	return true, nil
}

func (txN Tx) validatePrivacyZKPSanityWithOutput() (bool, error) {
	cmValueOfOutputCoins := txN.Proof.GetCommitmentOutputValue()
	rangeProof := txN.Proof.GetAggregatedRangeProof()
	if rangeProof == nil {
		return false, errors.Errorf("Invalid range proof, it can not be nil")
	}
	cmValueInBulletProof := rangeProof.GetCmValues()
	if len(cmValueOfOutputCoins) != len(cmValueInBulletProof) {
		return false, errors.New("invalid cmValues in Bullet proof")
	}

	for i, cmValue := range cmValueOfOutputCoins {
		if !privacy.IsPointEqual(cmValue, cmValueInBulletProof[i]) {
			Logger.log.Errorf("cmValue in Bulletproof is not equal to commitment of output's Value - txId %v", txN.Hash().String())
			return false, fmt.Errorf("cmValue %v in Bulletproof is not equal to commitment of output's Value", i)
		}
	}

	if !rangeProof.ValidateSanity() {
		return false, errors.New("validate sanity Aggregated range proof failed")
	}

	return true, nil
}

func (txN Tx) validateNoPrivacyZKPSanityWithOutput() (bool, error) {
	rangeProof := txN.Proof.GetAggregatedRangeProof()
	if rangeProof != nil {
		return false, errors.Errorf("This field must be nil")
	}
	return true, nil
}

func (txN Tx) validateSanityDataPrivacyProof() (bool, error) {
	if len(txN.Proof.GetInputCoins()) > 0 {
		ok, err := txN.validateInputPrivacy()
		if !ok || err != nil {
			return ok, err
		}
		ok, err = txN.validatePrivacyZKPSanityWithInput()
		if !ok || err != nil {
			return ok, err
		}
	}
	if len(txN.Proof.GetOutputCoins()) > 0 {
		ok, err := txN.validateOutputPrivacy()
		if !ok || err != nil {
			return ok, err
		}
		ok, err = txN.validatePrivacyZKPSanityWithOutput()
		if !ok || err != nil {
			return ok, err
		}
	}
	return true, nil
}

func (txN Tx) validateSanityDataNoPrivacyProof() (bool, error) {
	if len(txN.Proof.GetInputCoins()) > 0 {
		ok, err := txN.validateInputNoPrivacy()
		if !ok || err != nil {
			return ok, err
		}
		ok, err = txN.validateNoPrivacyZKPSanityWithInput()
		if !ok || err != nil {
			return ok, err
		}
	}
	if len(txN.Proof.GetOutputCoins()) > 0 {
		ok, err := txN.validateOutputNoPrivacy()
		if !ok || err != nil {
			return ok, err
		}
		ok, err = txN.validateNoPrivacyZKPSanityWithOutput()
		if !ok || err != nil {
			return ok, err
		}
	}
	return true, nil
}

// Todo decoupling this function
func (txN Tx) validateSanityDataOfProofV2() (bool, error) {
	if txN.Proof != nil {
		isPrivacy := txN.IsPrivacy()
		if isPrivacy {
			return txN.validateSanityDataPrivacyProof()
		}
		return txN.validateSanityDataNoPrivacyProof()
	}
	return false, nil
}
