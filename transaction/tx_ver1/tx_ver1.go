package tx_ver1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
)

type Tx struct {
	tx_generic.TxBase
}

// ========== GET FUNCTION ============

func (tx *Tx) GetReceiverData() ([]privacy.Coin, error) {
	pubkeys := make([]*privacy.Point, 0)
	amounts := []uint64{}

	if tx.Proof != nil && len(tx.Proof.GetOutputCoins()) > 0 {
		for _, coin := range tx.Proof.GetOutputCoins() {
			coinPubKey := coin.GetPublicKey()
			added := false
			for i, key := range pubkeys {
				if bytes.Equal(coinPubKey.ToBytesS(), key.ToBytesS()) {
					added = true
					amounts[i] += coin.GetValue()
					break
				}
			}
			if !added {
				pubkeys = append(pubkeys, coinPubKey)
				amounts = append(amounts, coin.GetValue())
			}
		}
	}
	coins := make([]privacy.Coin, 0)
	for i := 0; i < len(pubkeys); i++ {
		coin := new(privacy.CoinV1).Init()
		coin.CoinDetails.SetPublicKey(pubkeys[i])
		coin.CoinDetails.SetValue(amounts[i])
		coins = append(coins, coin)
	}
	return coins, nil
}

// ========== CHECK FUNCTION ===========

func (tx *Tx) CheckAuthorizedSender(publicKey []byte) (bool, error) {
	sigPubKey := tx.GetSigPubKey()
	if bytes.Equal(sigPubKey, publicKey) {
		return true, nil
	} else {
		return false, nil
	}
}

// checkSNDerivatorExistence return true if snd exists in snDerivators list
func checkSNDerivatorExistence(tokenID *common.Hash, snd *privacy.Scalar, stateDB *statedb.StateDB) (bool, error) {
	ok, err := statedb.HasSNDerivator(stateDB, *tokenID, snd.ToBytesS())
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ========== NORMAL INIT FUNCTIONS ==========

func parseCommitments(params *tx_generic.TxPrivacyInitParams, shardID byte) ([]uint64, []uint64, error) {
	var commitmentIndexs []uint64   // array index random of commitments in db
	var myCommitmentIndexs []uint64 // index in array index random of commitment in db

	if params.HasPrivacy {
		randomParams := tx_generic.NewRandomCommitmentsProcessParam(params.InputCoins, privacy.CommitmentRingSize, params.StateDB, shardID, params.TokenID)
		commitmentIndexs, myCommitmentIndexs, _ = tx_generic.RandomCommitmentsProcess(randomParams)

		// Check number of list of random commitments, list of random commitment indices
		if len(commitmentIndexs) != len(params.InputCoins)*privacy.CommitmentRingSize {
			return nil, nil, utils.NewTransactionErr(utils.RandomCommitmentError, nil)
		}

		if len(myCommitmentIndexs) != len(params.InputCoins) {
			return nil, nil, utils.NewTransactionErr(utils.RandomCommitmentError, errors.New("number of list my commitment indices must be equal to number of input coins"))
		}
	}
	return commitmentIndexs, myCommitmentIndexs, nil
}

func parseCommitmentProving(params *tx_generic.TxPrivacyInitParams, shardID byte, commitmentIndexs []uint64) ([]*privacy.Point, error) {
	commitmentProving := make([]*privacy.Point, len(commitmentIndexs))
	for i, cmIndex := range commitmentIndexs {
		temp, err := statedb.GetCommitmentByIndex(params.StateDB, *params.TokenID, cmIndex, shardID)
		if err != nil {
			utils.Logger.Log.Error(errors.New(fmt.Sprintf("can not get commitment from index=%d shardID=%+v", cmIndex, shardID)))
			return nil, utils.NewTransactionErr(utils.CanNotGetCommitmentFromIndexError, err, cmIndex, shardID)
		}
		commitmentProving[i] = new(privacy.Point)
		commitmentProving[i], err = commitmentProving[i].FromBytesS(temp)
		if err != nil {
			utils.Logger.Log.Error(errors.New(fmt.Sprintf("can not get commitment from index=%d shardID=%+v value=%+v", cmIndex, shardID, temp)))
			return nil, utils.NewTransactionErr(utils.CanNotDecompressCommitmentFromIndexError, err, cmIndex, shardID, temp)
		}
	}
	return commitmentProving, nil
}

func generateSndOut(paymentInfo []*privacy.PaymentInfo, tokenID *common.Hash, stateDB *statedb.StateDB) []*privacy.Scalar {
	ok := true
	sndOuts := make([]*privacy.Scalar, 0)
	for ok {
		for i := 0; i < len(paymentInfo); i++ {
			sndOut := privacy.RandomScalar()
			for {
				ok1, err := checkSNDerivatorExistence(tokenID, sndOut, stateDB)
				if err != nil {
					utils.Logger.Log.Error(err)
				}
				// if sndOut existed, then re-random it
				if ok1 {
					sndOut = privacy.RandomScalar()
				} else {
					break
				}
			}
			sndOuts = append(sndOuts, sndOut)
		}

		// if sndOuts has two elements that have same value, then re-generates it
		ok = privacy.CheckDuplicateScalarArray(sndOuts)
		if ok {
			sndOuts = make([]*privacy.Scalar, 0)
		}
	}
	return sndOuts
}

func parseOutputCoins(paymentInfo []*privacy.PaymentInfo, tokenID *common.Hash, stateDB *statedb.StateDB) ([]*privacy.CoinV1, error) {
	sndOuts := generateSndOut(paymentInfo, tokenID, stateDB)
	outputCoins := make([]*privacy.CoinV1, len(paymentInfo))
	for i, pInfo := range paymentInfo {
		outputCoins[i] = new(privacy.CoinV1)
		outputCoins[i].CoinDetails = new(privacy.PlainCoinV1)
		outputCoins[i].CoinDetails.SetValue(pInfo.Amount)
		if len(pInfo.Message) > 0 {
			if len(pInfo.Message) > privacy.MaxSizeInfoCoin {
				return nil, utils.NewTransactionErr(utils.ExceedSizeInfoOutCoinError, nil)
			}
		}
		outputCoins[i].CoinDetails.SetInfo(pInfo.Message)

		PK, err := new(privacy.Point).FromBytesS(pInfo.PaymentAddress.Pk)
		if err != nil {
			utils.Logger.Log.Error(errors.New(fmt.Sprintf("can not decompress public key from %+v", pInfo.PaymentAddress)))
			return nil, utils.NewTransactionErr(utils.DecompressPaymentAddressError, err, pInfo.PaymentAddress)
		}
		outputCoins[i].CoinDetails.SetPublicKey(PK)
		outputCoins[i].CoinDetails.SetSNDerivator(sndOuts[i])
	}
	return outputCoins, nil
}

func (tx *Tx) Init(paramsInterface interface{}) error {
	params, ok := paramsInterface.(*tx_generic.TxPrivacyInitParams)
	if !ok {
		return errors.New("params of tx Init is not TxPrivacyInitParam")
	}

	//utils.Logger.Log.Debugf("CREATING TX........\n")
	if err := tx_generic.ValidateTxParams(params); err != nil {
		return err
	}

	// Init tx and params (tx and params will be changed)
	if err := tx.InitializeTxAndParams(params); err != nil {
		return err
	}

	// Check if this tx is nonPrivacyNonInput
	// Case 1: tx ptoken transfer with ptoken fee
	// Case 2: tx Reward
	if check, err := tx.IsNonPrivacyNonInput(params); check {
		return err
	}

	if err := tx.prove(params); err != nil {
		return err
	}
	return nil
}

func (tx *Tx) initializePaymentWitnessParam(params *tx_generic.TxPrivacyInitParams) (*privacy.PaymentWitnessParam, error) {
	shardID := common.GetShardIDFromLastByte(tx.PubKeyLastByteSender)

	// get list of commitments for proving one-out-of-many from commitmentIndexs
	commitmentIndexs, myCommitmentIndexs, err := parseCommitments(params, shardID)
	if err != nil {
		return nil, err
	}
	commitmentProving, err := parseCommitmentProving(params, shardID, commitmentIndexs)
	if err != nil {
		return nil, err
	}
	outputCoins, err := parseOutputCoins(params.PaymentInfo, params.TokenID, params.StateDB)
	if err != nil {
		return nil, err
	}

	// prepare witness for proving
	paymentWitnessParam := privacy.PaymentWitnessParam{
		HasPrivacy:              params.HasPrivacy,
		PrivateKey:              new(privacy.Scalar).FromBytesS(*params.SenderSK),
		InputCoins:              params.InputCoins,
		OutputCoins:             outputCoins,
		PublicKeyLastByteSender: tx.PubKeyLastByteSender,
		Commitments:             commitmentProving,
		CommitmentIndices:       commitmentIndexs,
		MyCommitmentIndices:     myCommitmentIndexs,
		Fee:                     params.Fee,
	}
	return &paymentWitnessParam, nil
}

func (tx *Tx) proveAndSignCore(params *tx_generic.TxPrivacyInitParams, paymentWitnessParamPtr *privacy.PaymentWitnessParam) error {
	paymentWitnessParam := *paymentWitnessParamPtr
	witness := new(privacy.PaymentWitness)
	err := witness.Init(paymentWitnessParam)
	if err != nil {
		utils.Logger.Log.Error(err)
		jsonParam, _ := json.MarshalIndent(paymentWitnessParam, common.EmptyString, "  ")
		return utils.NewTransactionErr(utils.InitWithnessError, err, string(jsonParam))
	}

	paymentProof, err := witness.Prove(params.HasPrivacy, params.PaymentInfo)
	if err != nil {
		utils.Logger.Log.Error(err)
		jsonParam, _ := json.MarshalIndent(paymentWitnessParam, common.EmptyString, "  ")
		return utils.NewTransactionErr(utils.WithnessProveError, err, params.HasPrivacy, string(jsonParam))
	}
	tx.Proof = paymentProof

	// set private key for signing tx
	if params.HasPrivacy {
		randSK := witness.GetRandSecretKey()
		tx.SetPrivateKey(append(*params.SenderSK, randSK.ToBytesS()...))
	} else {
		tx.SetPrivateKey([]byte{})
		randSK := big.NewInt(0)
		tx.SetPrivateKey(append(*params.SenderSK, randSK.Bytes()...))
	}

	// sign tx
	signErr := tx.sign()
	if signErr != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.SignTxError, err)
	}
	return nil
}

func (tx *Tx) prove(params *tx_generic.TxPrivacyInitParams) error {
	// PrepareTransaction paymentWitness params
	paymentWitnessParamPtr, err := tx.initializePaymentWitnessParam(params)
	if err != nil {
		return err
	}
	return tx.proveAndSignCore(params, paymentWitnessParamPtr)
}

// func (tx *Tx) initializePaymentWitnessParamASM(params *TxPrivacyInitParamsForASM) (*privacy.PaymentWitnessParam, error) {
// 	// create SNDs for output coins
// 	txParams := &params.txParam
// 	outputCoins, err := parseOutputCoins(txParams.paymentInfo, txParams.tokenID, txParams.stateDB)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// get list of commitments for proving one-out-of-many from commitmentIndexs
// 	shardID := common.GetShardIDFromLastByte(tx.PubKeyLastByteSender)
// 	commitmentProving := make([]*operation.Point, len(params.commitmentBytes))
// 	for i, cmBytes := range params.commitmentBytes {
// 		commitmentProving[i] = new(operation.Point)
// 		commitmentProving[i], err = commitmentProving[i].FromBytesS(cmBytes)
// 		if err != nil {
// 			utils.Logger.Log.Error(errors.New(fmt.Sprintf("ASM: Can not get commitment from index=%d shardID=%+v value=%+v", params.commitmentIndices[i], shardID, cmBytes)))
// 			return nil, NewTransactionErr(CanNotDecompressCommitmentFromIndexError, err, params.commitmentIndices[i], shardID, cmBytes)
// 		}
// 	}

// 	// prepare witness for proving
// 	paymentWitnessParam := privacy.PaymentWitnessParam{
// 		HasPrivacy:              params.txParam.hasPrivacy,
// 		PrivateKey:              new(operation.Scalar).FromBytesS(*params.txParam.senderSK),
// 		InputCoins:              params.txParam.inputCoins,
// 		OutputCoins:             outputCoins,
// 		PublicKeyLastByteSender: tx.PubKeyLastByteSender,
// 		Commitments:             commitmentProving,
// 		CommitmentIndices:       params.commitmentIndices,
// 		MyCommitmentIndices:     params.myCommitmentIndices,
// 		Fee:                     params.txParam.fee,
// 	}
// 	return &paymentWitnessParam, nil
// }

func (tx *Tx) sign() error {
	//Check input transaction
	if tx.Sig != nil {
		return utils.NewTransactionErr(utils.UnexpectedError, errors.New("input transaction must be an unsigned one"))
	}

	/****** using Schnorr signature *******/
	// sign with sigPrivKey
	// prepare private key for Schnorr
	sk := new(privacy.Scalar).FromBytesS(tx.GetPrivateKey()[:common.BigIntSize])
	r := new(privacy.Scalar).FromBytesS(tx.GetPrivateKey()[common.BigIntSize:])
	sigKey := new(privacy.SchnorrPrivateKey)
	sigKey.Set(sk, r)

	// save public key for verification signature tx
	tx.SigPubKey = sigKey.GetPublicKey().GetPublicKey().ToBytesS()

	// signing
	signature, err := sigKey.Sign(tx.Hash()[:])
	if err != nil {
		return err
	}

	// convert signature to byte array
	tx.Sig = signature.Bytes()

	return nil
}

func (tx *Tx) Sign(sigPrivakey []byte) error {//For testing-purpose only, remove when deploy
	if sigPrivakey != nil{
		tx.SetPrivateKey(sigPrivakey)
	}
	return tx.sign()
}

// func (tx *Tx) proveASM(params *TxPrivacyInitParamsForASM) error {
// 	paymentWitnessParamPtr, err := tx.initializePaymentWitnessParamASM(params)
// 	if err != nil {
// 		return err
// 	}
// 	return tx.proveAndSignCore(&params.txParam, paymentWitnessParamPtr)
// }

// ========== NORMAL VERIFY FUNCTIONS ==========

func (tx Tx) ValidateSanityData(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, error) {
	if check, err := tx_generic.ValidateSanity(&tx, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight); !check || err != nil {
		utils.Logger.Log.Errorf("Cannot check sanity of version, size, proof, type and info: err %v", err)
		return false, err
	}

	if check, err := tx_generic.MdValidateSanity(&tx, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight); !check || err != nil {
		utils.Logger.Log.Errorf("Cannot check sanity of metadata: err %v", err)
		return false, err
	}
	// Ver1 also validate size of sigpubkey
	if len(tx.SigPubKey) != common.SigPubKeySize {
		return false, utils.NewTransactionErr(utils.RejectTxPublickeySigSize, fmt.Errorf("wrong tx Sig PK size %d", len(tx.SigPubKey)))
	}
	return true, nil
}

func validateSndFromOutputCoin(outputCoins []*privacy.CoinV1) error {
	sndOutputs := make([]*privacy.Scalar, len(outputCoins))
	for i := 0; i < len(outputCoins); i++ {
		sndOutputs[i] = outputCoins[i].GetSNDerivator()
	}
	if privacy.CheckDuplicateScalarArray(sndOutputs) {
		utils.Logger.Log.Errorf("Duplicate output coins' snd\n")
		return utils.NewTransactionErr(utils.DuplicatedOutputSndError, errors.New("Duplicate output coins' snd\n"))
	}
	return nil
}

func getCommitmentsInDatabase(
	proof *privacy.ProofV1, hasPrivacy bool,
	pubKey privacy.PublicKey, transactionStateDB *statedb.StateDB,
	shardID byte, tokenID *common.Hash, isBatch bool) (*[][privacy.CommitmentRingSize]*privacy.Point, error) {

	// verify for input coins
	oneOfManyProof := proof.GetOneOfManyProof()
	commitmentIndices := proof.GetCommitmentIndices()
	commitmentInputSND := proof.GetCommitmentInputSND()
	commitmentInputValue := proof.GetCommitmentInputValue()
	commitmentInputShardID := proof.GetCommitmentInputShardID()
	commitmentInputSecretKey := proof.GetCommitmentInputSecretKey()

	const sz int = privacy.CommitmentRingSize
	commitments := make([][sz]*privacy.Point, len(oneOfManyProof))
	for i := 0; i < len(oneOfManyProof); i++ {
		cmInputSum := new(privacy.Point).Add(commitmentInputSecretKey, commitmentInputValue[i])
		cmInputSum.Add(cmInputSum, commitmentInputSND[i])
		cmInputSum.Add(cmInputSum, commitmentInputShardID)

		for j := 0; j < sz; j++ {
			index := commitmentIndices[i*privacy.CommitmentRingSize+j]
			commitmentBytes, err := statedb.GetCommitmentByIndex(transactionStateDB, *tokenID, index, shardID)
			if err != nil {
				utils.Logger.Log.Errorf("GetCommitmentInDatabase: Error when getCommitmentByIndex from database", index, err)
				return nil, utils.NewTransactionErr(utils.GetCommitmentsInDatabaseError, err)
			}
			recheckIndex, err := statedb.GetCommitmentIndex(transactionStateDB, *tokenID, commitmentBytes, shardID)
			if err != nil || recheckIndex.Uint64() != index {
				utils.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Error when getCommitmentIndex from database", index, err)
				return nil, utils.NewTransactionErr(utils.GetCommitmentsInDatabaseError, err)
			}

			commitments[i][j], err = new(privacy.Point).FromBytesS(commitmentBytes)
			if err != nil {
				utils.Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Cannot decompress commitment from database", index, err)
				return nil, errhandler.NewPrivacyErr(utils.VerifyOneOutOfManyProofFailedErr, err)
			}
			commitments[i][j].Sub(commitments[i][j], cmInputSum)
		}
	}
	return &commitments, nil
}

func (tx *Tx) verifySig() (bool, error) {
	// check input transaction
	if tx.Sig == nil || tx.SigPubKey == nil {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("input transaction must be an signed one"))
	}

	/****** verify Schnorr signature *****/
	// prepare Public key for verification
	verifyKey := new(privacy.SchnorrPublicKey)
	sigPublicKey, err := new(privacy.Point).FromBytesS(tx.SigPubKey)
	if err != nil {
		utils.Logger.Log.Error(err)
		return false, utils.NewTransactionErr(utils.DecompressSigPubKeyError, err)
	}
	verifyKey.Set(sigPublicKey)

	// convert signature from byte array to SchnorrSign
	signature := new(privacy.SchnSignature)
	if err = signature.SetBytes(tx.Sig); err != nil {
		utils.Logger.Log.Error(err)
		return false, utils.NewTransactionErr(utils.InitTxSignatureFromBytesError, err)
	}
	res := verifyKey.Verify(signature, tx.Hash()[:])
	return res, nil
}

func (tx *Tx) Verify(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	var err error
	if valid, err := tx.verifySig(); !valid {
		if err != nil {
			utils.Logger.Log.Errorf("Error verifying signature ver1 with tx hash %s: %+v \n", tx.Hash().String(), err)
			return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
		}
		utils.Logger.Log.Errorf("FAILED VERIFICATION SIGNATURE ver1 with tx hash %s", tx.Hash().String())
		return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, fmt.Errorf("FAILED VERIFICATION SIGNATURE ver1 with tx hash %s", tx.Hash().String()))
	}

	hasPrivacy, ok := boolParams["hasPrivacy"]
	if !ok {
		hasPrivacy = false
	}
	isNewTransaction, ok := boolParams["isNewTransaction"]
	if !ok {
		isNewTransaction = false
	}
	isBatch, ok := boolParams["isBatch"]
	if !ok {
		isBatch = false
	}

	if tx.Proof == nil {
		return true, nil
	}

	tokenID, err = tx_generic.ParseTokenID(tokenID)
	if err != nil {
		return false, err
	}
	inputCoins := tx.Proof.GetInputCoins()
	outputCoins := tx.Proof.GetOutputCoins()
	outputCoinsAsV1 := make([]*privacy.CoinV1, len(outputCoins))
	for i := 0; i < len(outputCoins); i += 1 {
		c, ok := outputCoins[i].(*privacy.CoinV1)
		if !ok{
			return false, utils.NewTransactionErr(utils.UnexpectedError, nil, fmt.Sprintf("Error when casting a coin to ver1"))
		}
		outputCoinsAsV1[i] = c
	}
	if err := validateSndFromOutputCoin(outputCoinsAsV1); err != nil {
		return false, err
	}

	if isNewTransaction {
		for i := 0; i < len(outputCoins); i++ {
			// Check output coins' SND is not exists in SND list (Database)
			if ok, err := checkSNDerivatorExistence(tokenID, outputCoins[i].GetSNDerivator(), transactionStateDB); ok || err != nil {
				if err != nil {
					utils.Logger.Log.Error(err)
				}
				utils.Logger.Log.Errorf("snd existed: %d\n", i)
				return false, utils.NewTransactionErr(utils.SndExistedError, err, fmt.Sprintf("snd existed: %d\n", i))
			}
		}
	}
	if !hasPrivacy {
		// Check input coins' commitment is exists in cm list (Database)
		for i := 0; i < len(inputCoins); i++ {
			if inputCoins[i].GetCommitment()==nil{
				return false, utils.NewTransactionErr(utils.InputCommitmentIsNotExistedError, err)
			}
			ok, err := statedb.HasCommitment(transactionStateDB, *tokenID, inputCoins[i].GetCommitment().ToBytesS(), shardID)
			if !ok || err != nil {
				if err != nil {
					utils.Logger.Log.Error(err)
				}
				return false, utils.NewTransactionErr(utils.InputCommitmentIsNotExistedError, err)
			}
		}
	}
	// Verify the payment proof
	txProofV1, ok := tx.Proof.(*privacy.ProofV1)
	if !ok{
		return false, utils.NewTransactionErr(utils.RejectTxVersion, errors.New("Wrong proof version"))
	}

	commitments, err := getCommitmentsInDatabase(txProofV1, hasPrivacy, tx.SigPubKey, transactionStateDB, shardID, tokenID, isBatch)
	if err != nil {
		return false, err
	}

	if valid, err := tx.Proof.Verify(boolParams, tx.SigPubKey, tx.Fee, shardID, tokenID, commitments); !valid {
		if err != nil {
			utils.Logger.Log.Error(err)
		}
		utils.Logger.Log.Error("FAILED VERIFICATION PAYMENT PROOF")
		err1, ok := err.(*privacy.PrivacyError)
		if ok {
			// parse error detail
			if err1.Code == privacy.ErrCodeMessage[errhandler.VerifyOneOutOfManyProofFailedErr].Code {
				if isNewTransaction {
					return false, utils.NewTransactionErr(utils.VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
				} else {
					// for old txs which be get from sync block or validate new block
					if tx.LockTime <= utils.ValidateTimeForOneoutOfManyProof {
						// only verify by sign on block because of issue #504(that mean we should pass old tx, which happen before this issue)
						return true, nil
					} else {
						return false, utils.NewTransactionErr(utils.VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
					}
				}
			}
		}
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, err, tx.Hash().String())
	}
	utils.Logger.Log.Debugf("SUCCESSED VERIFICATION PAYMENT PROOF ")
	return true, nil
}

func (tx Tx) VerifyMinerCreatedTxBeforeGettingInBlock(mintdata *metadata.MintData,
		shardID byte, bcr metadata.ChainRetriever,
		accumulatedValues *metadata.AccumulatedValues,
		retriever metadata.ShardViewRetriever,
		viewRetriever metadata.BeaconViewRetriever) (bool, error) {
	return tx_generic.VerifyTxCreatedByMiner(&tx, mintdata, shardID, bcr, accumulatedValues, retriever, viewRetriever)
}

// ========== SALARY FUNCTIONS: INIT AND VALIDATE  ==========

func (tx *Tx) InitTxSalary(salary uint64, receiverAddr *privacy.PaymentAddress, privKey *privacy.PrivateKey, stateDB *statedb.StateDB, metaData metadata.Metadata,
) error {
	tx.Version = utils.TxVersion1Number
	tx.Type = common.TxRewardType
	if tx.LockTime == 0 {
		tx.LockTime = time.Now().Unix()
	}

	var err error
	// create new output coins with info: Pk, value, input, randomness, last byte pk, coin commitment
	tx.Proof = new(privacy.ProofV1)
	tempOutputCoin := make([]*privacy.CoinV1, 1)
	tempOutputCoin[0] = new(privacy.CoinV1).Init()
	publicKey, err := new(privacy.Point).FromBytesS(receiverAddr.Pk)
	if err != nil {
		return err
	}
	tempOutputCoin[0].CoinDetails.SetPublicKey(publicKey)
	tempOutputCoin[0].CoinDetails.SetValue(salary)
	tempOutputCoin[0].CoinDetails.SetRandomness(privacy.RandomScalar())

	sndOut := privacy.RandomScalar()
	for {
		tokenID := &common.Hash{}
		err := tokenID.SetBytes(common.PRVCoinID[:])
		if err != nil {
			return utils.NewTransactionErr(utils.TokenIDInvalidError, err, tokenID.String())
		}
		ok, err := checkSNDerivatorExistence(tokenID, sndOut, stateDB)
		if err != nil {
			return utils.NewTransactionErr(utils.SndExistedError, err)
		}
		if ok {
			sndOut = privacy.RandomScalar()
		} else {
			break
		}
	}
	tempOutputCoin[0].CoinDetails.SetSNDerivator(sndOut)
	if err = tempOutputCoin[0].CoinDetails.CommitAll(); err != nil {
		return utils.NewTransactionErr(utils.CommitOutputCoinError, err)
	}
	outputCoinsAsGeneric := make([]privacy.Coin, len(tempOutputCoin))
	for i := 0; i < len(tempOutputCoin); i += 1 {
		outputCoinsAsGeneric[i] = tempOutputCoin[i]
	}
	tx.Proof.SetOutputCoins(outputCoinsAsGeneric)
	tx.PubKeyLastByteSender = receiverAddr.Pk[len(receiverAddr.Pk)-1]

	// sign Tx
	tx.SigPubKey = receiverAddr.Pk
	tx.SetPrivateKey(*privKey)
	tx.SetMetadata(metaData)
	err = tx.sign()
	if err != nil {
		return utils.NewTransactionErr(utils.SignTxError, err)
	}
	return nil
}

func (tx Tx) ValidateTxSalary(
	db *statedb.StateDB,
) (bool, error) {
	// verify signature
	valid, err := tx.verifySig()
	if !valid {
		if err != nil {
			utils.Logger.Log.Debugf("Error verifying signature of tx: %+v", err)
			return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
		}
		return false, nil
	}

	// check whether output coin's input exists in input list or not
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return false, utils.NewTransactionErr(utils.TokenIDInvalidError, err, tokenID.String())
	}
	outputCoins := tx.Proof.GetOutputCoins()
	if len(outputCoins) != 1 {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("length outputCoins of proof is not 1"))
	}
	coin := outputCoins[0]
	if coin.GetPublicKey()==nil || coin.GetSNDerivator()==nil || coin.GetRandomness()==nil || coin.GetCommitment()==nil{
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("output coin is corrupted"))
	}
	if ok, err := checkSNDerivatorExistence(tokenID, coin.GetSNDerivator(), db); ok || err != nil {
		return false, err
	}

	// check output coin's coin commitment is calculated correctly
	shardID, err := coin.GetShardID()
	if err != nil {
		utils.Logger.Log.Errorf("Cannot get shardID from coin %v", err)
		return false, err
	}

	cmTmp2 := new(privacy.Point)
	cmTmp2.Add(coin.GetPublicKey(), new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenValueIndex], new(privacy.Scalar).FromUint64(uint64(coin.GetValue()))))
	cmTmp2.Add(cmTmp2, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenSndIndex], coin.GetSNDerivator()))
	cmTmp2.Add(cmTmp2, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenShardIDIndex], new(privacy.Scalar).FromUint64(uint64(shardID))))
	cmTmp2.Add(cmTmp2, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], coin.GetRandomness()))

	ok := privacy.IsPointEqual(cmTmp2, coin.GetCommitment())
	if !ok {
		return ok, utils.NewTransactionErr(utils.UnexpectedError, errors.New("check output coin's coin commitment isn't calculated correctly"))
	}
	return ok, nil
}

// =========== SHARED FUNCTIONS ==========

func (tx Tx) GetTxMintData() (bool, privacy.Coin, *common.Hash, error) { return tx_generic.GetTxMintData(&tx, &common.PRVCoinID) }

func (tx Tx) GetTxBurnData() (bool, privacy.Coin, *common.Hash, error) { return tx_generic.GetTxBurnData(&tx) }

func (tx Tx) GetTxFullBurnData() (bool, privacy.Coin, privacy.Coin, *common.Hash, error) {
	isBurn, burnedCoin, burnedTokenID, err := tx.GetTxBurnData()
	return isBurn, burnedCoin, nil, burnedTokenID, err
}


func (tx Tx) ValidateTxWithBlockChain(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error {
	err := tx_generic.MdValidateWithBlockChain(&tx, chainRetriever, shardViewRetriever, beaconViewRetriever, shardID, stateDB)
	if err!=nil{
		return err
	}
	return tx.TxBase.ValidateDoubleSpendWithBlockchain(shardID, stateDB, nil)
}

func (tx Tx) ValidateTransaction(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, []privacy.Proof, error) {
	switch tx.GetType() {
	case common.TxRewardType:
		valid, err := tx.ValidateTxSalary(transactionStateDB)
		return valid, nil, err
	case common.TxReturnStakingType:
		return tx.ValidateTxReturnStaking(transactionStateDB), nil, nil
	default:
		valid, err := tx.Verify(boolParams, transactionStateDB, bridgeStateDB, shardID, tokenID)
		resultProofs := []privacy.Proof{}
		hasPrivacy, ok := boolParams["hasPrivacy"]
		if !ok {
			hasPrivacy = false
		}
		isBatch, ok := boolParams["isBatch"]
		if !ok {
			isBatch = false
		}
		if isBatch && hasPrivacy{
			if tx.GetProof()!=nil{
				resultProofs = append(resultProofs, tx.GetProof())
			}
		}
		return valid, resultProofs, err
	}
}

func (tx Tx) ValidateTxByItself(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, chainRetriever metadata.ChainRetriever, shardID byte, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever) (bool, error) {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return false, err
	}
	valid, _, err := tx.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, prvCoinID)
	if !valid {
		return false, err
	}
	hasPrivacy, ok := boolParams["hasPrivacy"]
	if !ok {
		hasPrivacy = false
	}
	valid, err = tx_generic.MdValidate(&tx, hasPrivacy, transactionStateDB, bridgeStateDB, shardID)
	if !valid {
		return false, err
	}
	return true, nil
}

func (tx Tx) GetTxActualSize() uint64 {
	if tx.GetCachedActualSize() != nil {
		return *tx.GetCachedActualSize()
	}
	sizeTx := tx_generic.GetTxActualSizeInBytes(&tx)
	result := uint64(math.Ceil(float64(sizeTx) / 1024))
	tx.SetCachedActualSize(&result)
	return result
}