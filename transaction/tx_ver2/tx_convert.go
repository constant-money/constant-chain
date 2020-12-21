package tx_ver2

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/serialnumbernoprivacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"strconv"
	"time"
)

// ================ TX NORMAL CONVERSION =================

type TxConvertVer1ToVer2InitParams struct {
	senderSK    *privacy.PrivateKey
	paymentInfo []*privacy.PaymentInfo
	inputCoins  []privacy.PlainCoin
	fee         uint64
	stateDB     *statedb.StateDB
	tokenID     *common.Hash // default is nil -> use for prv coin
	metaData    metadata.Metadata
	info        []byte // 512 bytes
}

func NewTxConvertVer1ToVer2InitParams(senderSK *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoins []privacy.PlainCoin,
	fee uint64,
	stateDB *statedb.StateDB,
	tokenID *common.Hash, // default is nil -> use for prv coin
	metaData metadata.Metadata,
	info []byte) *TxConvertVer1ToVer2InitParams {
	// make sure info is not nil ; zero value for it is []byte{}

	if info==nil{
		info = []byte{}
	}

	return  &TxConvertVer1ToVer2InitParams{
		stateDB:     stateDB,
		tokenID:     tokenID,
		inputCoins:  inputCoins,
		fee:         fee,
		metaData:    metaData,
		paymentInfo: paymentInfo,
		senderSK:    senderSK,
		info:        info,
	}
}

func validateTxConvertVer1ToVer2Params (params *TxConvertVer1ToVer2InitParams) error {
	if len(params.inputCoins) > 255 {
		return utils.NewTransactionErr(utils.InputCoinIsVeryLargeError, nil, strconv.Itoa(len(params.inputCoins)))
	}
	if len(params.paymentInfo) > 254 {
		return utils.NewTransactionErr(utils.PaymentInfoIsVeryLargeError, nil, strconv.Itoa(len(params.paymentInfo)))
	}

	sumInput, sumOutput := uint64(0), uint64(0)
	for _, c := range params.inputCoins {
		if c.GetVersion() != 1 {
			err := errors.New("TxConversion should only have inputCoins version 1")
			return utils.NewTransactionErr(utils.InvalidInputCoinVersionErr, err)
		}

		//Verify if input coins have been concealed
		if c.GetRandomness() == nil || c.GetSNDerivator() == nil || c.GetPublicKey() == nil || c.GetCommitment() == nil {
			err := errors.New("input coins should not be concealed")
			return utils.NewTransactionErr(utils.InvalidInputCoinVersionErr, err)
		}
		sumInput += c.GetValue()
	}
	for _, c := range params.paymentInfo {
		sumOutput += c.Amount
	}
	if sumInput != sumOutput + params.fee {
		err := errors.New("TxConversion's sum input coin and output coin (with fee) is not the same")
		return utils.NewTransactionErr(utils.SumInputCoinsAndOutputCoinsError, err)
	}

	if params.tokenID == nil {
		// using default PRV
		params.tokenID = &common.Hash{}
		if err := params.tokenID.SetBytes(common.PRVCoinID[:]); err != nil {
			return utils.NewTransactionErr(utils.TokenIDInvalidError, err, params.tokenID.String())
		}
	}
	return nil
}

func initializeTxConversion(tx *Tx, params *TxConvertVer1ToVer2InitParams) error {
	var err error
	// Get Keyset from param
	senderKeySet :=  incognitokey.KeySet{}
	if err:= senderKeySet.InitFromPrivateKey(params.senderSK); err != nil {
		utils.Logger.Log.Errorf("Cannot parse Private Key. Err %v", err)
		return utils.NewTransactionErr(utils.PrivateKeySenderInvalidError, err)
	}

	// Tx: initialize some values
	tx.Fee = params.fee
	tx.Version = utils.TxConversionVersion12Number
	tx.Type = common.TxConversionType
	tx.Metadata = params.metaData
	tx.PubKeyLastByteSender = senderKeySet.PaymentAddress.Pk[len(senderKeySet.PaymentAddress.Pk)-1]

	if tx.LockTime == 0 {
		tx.LockTime = time.Now().Unix()
	}
	if tx.Info, err = tx_generic.GetTxInfo(params.info); err != nil {
		return err
	}
	return nil
}

func InitConversion(tx *Tx, params *TxConvertVer1ToVer2InitParams) error {
	// validate again
	if err := validateTxConvertVer1ToVer2Params(params); err != nil {
		return err
	}
	if err := initializeTxConversion(tx, params); err != nil {
		return err
	}
	if err := proveConversion(tx, params); err != nil {
		return err
	}
	jsb, _ := json.Marshal(tx)
	utils.Logger.Log.Infof("Init conversion complete ! -> %s", string(jsb))
	txSize := tx.GetTxActualSize()
	if txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}
	return nil
}

func getOutputcoinsFromPaymentInfo(paymentInfos []*privacy.PaymentInfo, tokenID *common.Hash,  db *statedb.StateDB) ([]*privacy.CoinV2, error) {
	var err error
	isPRV := (tokenID==nil) || (*tokenID==common.PRVCoinID)
	c := make([]*privacy.CoinV2, len(paymentInfos))

	for i := 0; i < len(paymentInfos); i += 1 {
		if isPRV{
			c[i], err = utils.NewCoinUniqueOTABasedOnPaymentInfo(paymentInfos[i], tokenID, db)
			if err != nil {
				utils.Logger.Log.Errorf("TxConversion cannot create new coin unique OTA, got error %v", err)
				return nil, err
			}
		}else{
			createdCACoin, _, err := createUniqueOTACoinCA(paymentInfos[i], tokenID, db)
			if err!=nil{
				utils.Logger.Log.Errorf("TxConversion cannot create new CA coin - %v", err)
				return nil, err
			}
			createdCACoin.SetPlainTokenID(tokenID)
			c[i] = createdCACoin
		}
	}
	return c, nil
}

func proveConversion(tx *Tx, params *TxConvertVer1ToVer2InitParams) error {
	inputCoins := params.inputCoins
	outputCoins, err := getOutputcoinsFromPaymentInfo(params.paymentInfo, params.tokenID, params.stateDB)
	if err != nil {
		utils.Logger.Log.Errorf("TxConversion cannot get output coins from payment info got error %v", err)
		return err
	}
	lenInputs := len(inputCoins)
	serialnumberWitness := make([]*serialnumbernoprivacy.SNNoPrivacyWitness, lenInputs)
	for i := 0; i < len(inputCoins); i++ {
		/***** Build witness for proving that serial number is derived from the committed derivator *****/
		serialnumberWitness[i] = new(serialnumbernoprivacy.SNNoPrivacyWitness)
		serialnumberWitness[i].Set(inputCoins[i].GetKeyImage(), inputCoins[i].GetPublicKey(),
			inputCoins[i].GetSNDerivator(), new(privacy.Scalar).FromBytesS(*params.senderSK))
	}
	tx.Proof, err = privacy_v2.ProveConversion(inputCoins, outputCoins, serialnumberWitness)
	if err != nil {
		utils.Logger.Log.Errorf("Error in privacy_v2.Prove, error %v ", err)
		return err
	}

	// sign tx
	if tx.Sig, tx.SigPubKey, err = tx_generic.SignNoPrivacy(params.senderSK, tx.Hash()[:]); err != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.SignTxError, err)
	}
	return nil
}

func validateConversionVer1ToVer2(tx metadata.Transaction, db *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	if valid, err := tx_generic.VerifySigNoPrivacy(tx.GetSig(), tx.GetSigPubKey(), tx.Hash()[:]); !valid {
		if err != nil {
			utils.Logger.Log.Errorf("Error verifying signature conversion with tx hash %s: %+v \n", tx.Hash().String(), err)
			return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
		}
		utils.Logger.Log.Errorf("FAILED VERIFICATION SIGNATURE conversion with tx hash %s", tx.Hash().String())
		return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, fmt.Errorf("FAILED VERIFICATION SIGNATURE ver1 with tx hash %s", tx.Hash().String()))
	}
	proofConversion, ok := tx.GetProof().(*privacy_v2.ConversionProofVer1ToVer2)
	if !ok {
		utils.Logger.Log.Error("Error casting ConversionProofVer1ToVer2")
		return false, errors.New("Error casting ConversionProofVer1ToVer2")
	}

	//Verify that input coins have not been spent
	inputCoins := proofConversion.GetInputCoins()
	for i := 0; i < len(inputCoins); i++ {
		if ok, err := statedb.HasSerialNumber(db, *tokenID, inputCoins[i].GetKeyImage().ToBytesS(), shardID); ok || err != nil {
			if err != nil {
				errStr := fmt.Sprintf("TxConversion database serialNumber got error: %v", err)
				return false, errors.New(errStr)
			}
			return false, errors.New("TxConversion found existing serialNumber in database error")
		}
	}

	//Verify that output coins' one-time-address has not been obtained + not duplicate OTAs
	outputCoins := proofConversion.GetOutputCoins()
	mapOutputCoins := make(map[string]int)
	for i := 0; i < len(outputCoins); i++ {
		if ok, err := statedb.HasOnetimeAddress(db, *tokenID, outputCoins[i].GetPublicKey().ToBytesS()); ok || err != nil {
			if err != nil {
				errStr := fmt.Sprintf("TxConversion database onetimeAddress got error: %v", err)
				return false, errors.New(errStr)
			}
			return false, errors.New("TxConversion found existing one-time-address in database error")
		}
		dst := make([]byte, hex.EncodedLen(len(outputCoins[i].GetPublicKey().ToBytesS())))
		hex.Encode(dst, outputCoins[i].GetPublicKey().ToBytesS())
		mapOutputCoins[string(dst)] = i
	}
	if len(mapOutputCoins) != len(outputCoins) {
		return false, errors.New("TxConversion found duplicate one-time-address error")
	}


	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = false
	boolParams["isBatch"] = false

	//Verify the conversion proof
	valid, err := proofConversion.Verify(boolParams, tx.GetSigPubKey(), tx.GetTxFee(), shardID, tokenID, nil)
	if !valid {
		if err != nil {
			utils.Logger.Log.Error(err)
		}
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, err, tx.Hash().String())
	}

	utils.Logger.Log.Debugf("SUCCEEDED VERIFICATION PAYMENT PROOF")
	return true, nil
}

// ================ TX TOKEN CONVERSION =================

type CustomTokenConversionParams struct {
	tokenID       *common.Hash
	tokenInputs   []privacy.PlainCoin
	tokenPayments []*privacy.PaymentInfo
}

type TxTokenConvertVer1ToVer2InitParams struct {
	senderSK      *privacy.PrivateKey
	feeInputs     []privacy.PlainCoin
	feePayments   []*privacy.PaymentInfo
	fee           uint64
	tokenParams   *CustomTokenConversionParams
	stateDB       *statedb.StateDB
	bridgeStateDB *statedb.StateDB
	metaData      metadata.Metadata
	info          []byte // 512 bytes
}

func NewTxTokenConvertVer1ToVer2InitParams(senderSK *privacy.PrivateKey,
	feeInputs []privacy.PlainCoin,
	feePayments []*privacy.PaymentInfo,
	tokenInputs []privacy.PlainCoin,
	tokenPayments []*privacy.PaymentInfo,
	fee uint64,
	stateDB *statedb.StateDB,
	bridgeStateDB *statedb.StateDB,
	tokenID *common.Hash, // tokenID of the conversion coin
	metaData metadata.Metadata,
	info []byte) *TxTokenConvertVer1ToVer2InitParams {

	if info == nil{
		info = []byte{}
	}


	tokenParams := &CustomTokenConversionParams{
		tokenID:       tokenID,
		tokenPayments: tokenPayments,
		tokenInputs:   tokenInputs,
	}
	return  &TxTokenConvertVer1ToVer2InitParams{
		stateDB:     stateDB,
		feeInputs:   feeInputs,
		fee:         fee,
		tokenParams: tokenParams,
		bridgeStateDB: bridgeStateDB,
		metaData:    metaData,
		feePayments: feePayments,
		senderSK:    senderSK,
		info:        info,
	}
}

func validateTxTokenConvertVer1ToVer2Params (params *TxTokenConvertVer1ToVer2InitParams) error {
	if len(params.feeInputs) > 255 {
		return errors.New("FeeInput is too large, feeInputs length = " + strconv.Itoa(len(params.feeInputs)))
	}
	if len(params.feePayments) > 255 {
		return errors.New("FeePayment is too large, feePayments length = " + strconv.Itoa(len(params.feePayments)))
	}
	if len(params.tokenParams.tokenPayments) > 255 {
		return errors.New("tokenPayments is too large, tokenPayments length = " + strconv.Itoa(len(params.tokenParams.tokenPayments)))
	}
	if len(params.tokenParams.tokenInputs) > 255 {
		return errors.New("tokenInputs length = " + strconv.Itoa(len(params.tokenParams.tokenInputs)))
	}

	for _, c := range params.feeInputs {
		if c.GetVersion() != utils.TxVersion2Number {
			return errors.New("TxConversion should only have fee input coins version 2")
		}
	}
	tokenParams := params.tokenParams
	if tokenParams.tokenID == nil {
		return utils.NewTransactionErr(utils.TokenIDInvalidError, errors.New("TxTokenConversion should have its tokenID not null"))
	}
	sumInput := uint64(0)
	for _, c := range tokenParams.tokenInputs {
		sumInput += c.GetValue()
	}
	if sumInput != tokenParams.tokenPayments[0].Amount {
		return utils.NewTransactionErr(utils.SumInputCoinsAndOutputCoinsError, errors.New("sumInput and sum TokenPayment amount is not equal"))
	}
	return nil
}

func (txToken *TxToken) initTokenConversion(txNormal *Tx, params *TxTokenConvertVer1ToVer2InitParams) error {
	txToken.TokenData.Type = utils.CustomTokenTransfer
	txToken.TokenData.PropertyName = ""
	txToken.TokenData.PropertySymbol = ""
	txToken.TokenData.Mintable = false
	txToken.TokenData.PropertyID = *params.tokenParams.tokenID

	existed := statedb.PrivacyTokenIDExisted(params.stateDB, *params.tokenParams.tokenID)
	if !existed {
		if err := checkIsBridgeTokenID(params.bridgeStateDB, params.tokenParams.tokenID); err != nil {
			return err
		}
	}

	txConvertParams := NewTxConvertVer1ToVer2InitParams(
		params.senderSK,
		params.tokenParams.tokenPayments,
		params.tokenParams.tokenInputs,
		0,
		params.stateDB,
		params.tokenParams.tokenID,
		nil,
		params.info,
	)
	
	if err := validateTxConvertVer1ToVer2Params(txConvertParams); err != nil {
		return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
	}
	if err := initializeTxConversion(txNormal, txConvertParams); err != nil {
		return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
	}
	txNormal.SetType(common.TxTokenConversionType)
	if err := proveConversion(txNormal, txConvertParams); err != nil {
		return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
	}
	// if err := InitConversion(txNormal, txConvertParams); err != nil {
	// 	return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
	// }
	err := txToken.SetTxNormal(txNormal)
	return err
}

func (txToken *TxToken) initPRVFeeConversion(feeTx *Tx, params *tx_generic.TxPrivacyInitParams) ([]privacy.PlainCoin, []*privacy.CoinV2, error) {
	// txTokenDataHash, err := txToken.TokenData.Hash()
	// if err != nil {
	// 	utils.Logger.Log.Errorf("Cannot calculate txPrivacyTokenData Hash, err %v", err)
	// 	return err
	// }
	feeTx.SetVersion(utils.TxConversionVersion12Number)
	feeTx.SetType(common.TxTokenConversionType)
	inps, outs, err := feeTx.provePRV(params)
	if err != nil {
		return nil, nil, utils.NewTransactionErr(utils.PrivacyTokenInitPRVError, err)
	}
	// override TxCustomTokenPrivacyType type
	
	// txToken.SetTxBase(feeTx)
	return inps, outs, nil
}

func InitTokenConversion(txToken *TxToken, params *TxTokenConvertVer1ToVer2InitParams) error {
	if err := validateTxTokenConvertVer1ToVer2Params(params); err != nil {
		return err
	}

	txPrivacyParams := tx_generic.NewTxPrivacyInitParams(
		params.senderSK, params.feePayments, params.feeInputs, params.fee,
		false, params.stateDB, nil, params.metaData, params.info,
	)
	if err := tx_generic.ValidateTxParams(txPrivacyParams); err != nil {
		return err
	}
	// Init tx and params (tx and params will be changed)
	tx := new(Tx)
	if err := tx.InitializeTxAndParams(txPrivacyParams); err != nil {
		return err
	}

	// Init PRV Fee
	inps, outs, err := txToken.initPRVFeeConversion(tx, txPrivacyParams)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot init token ver2: err %v", err)
		return err
	}
	txn := makeTxToken(tx, nil, nil, nil)
	// Init Token
	if err := txToken.initTokenConversion(txn, params); err != nil {
		utils.Logger.Log.Errorf("Cannot init token ver2: err %v", err)
		return err
	}
	tdh, err := txToken.TokenData.Hash()
	if err!=nil{
		return err
	}
	message := common.HashH(append(tx.Hash()[:], tdh[:]...))
	err = tx.signOnMessage(inps, outs, txPrivacyParams, message[:])
	if err!=nil{
		return err
	}
	err = txToken.SetTxBase(tx)
	if err!=nil{
		return err
	}
	txSize := txToken.GetTxActualSize()
	if txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}
	return nil
}