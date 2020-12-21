package tx_ver2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/wallet"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"math"
	"sort"
	"strconv"
)

type TxTokenDataVersion2 struct {
	PropertyID     common.Hash
	PropertyName   string
	PropertySymbol string
	SigPubKey      []byte `json:"SigPubKey,omitempty"` // 33 bytes
	Sig            []byte `json:"Sig,omitempty"`       //
	Proof          privacy.Proof

	Type     int
	Mintable bool
}

func (td TxTokenDataVersion2) Hash() (*common.Hash, error) {
	// leave out signature & its public key when hashing tx
	td.Sig = []byte{}
	td.SigPubKey = []byte{}
	inBytes, err := json.Marshal(td)

	if err != nil {
		return nil, err
	}
	// after this returns, tx is restored since the receiver is not a pointer
	hash := common.HashH(inBytes)
	return &hash, nil
}

func makeTxToken(txPRV *Tx, pubkey, sig []byte, proof privacy.Proof) *Tx {
	result := &Tx{
		TxBase: tx_generic.TxBase{
			Version:              txPRV.Version,
			Type:                 txPRV.Type,
			LockTime:             txPRV.LockTime,
			Fee:                  0,
			PubKeyLastByteSender: txPRV.PubKeyLastByteSender,
			Metadata:             nil,
		},
	}
	var clonedInfo []byte = nil
	var err error
	if txPRV.Info != nil {
		clonedInfo = make([]byte, len(txPRV.Info))
		copy(clonedInfo, txPRV.Info)
	}
	var clonedProof privacy.Proof = nil
	// feed the type to parse proof 
	proofType := txPRV.Type
	if proofType == common.TxTokenConversionType {
		proofType = common.TxConversionType
	}
	if proof != nil {
		clonedProof, err = utils.ParseProof(proof, txPRV.Version, proofType)
		if err != nil {
			jsb, _ := json.Marshal(proof)
			utils.Logger.Log.Errorf("Cannot parse proof %s using version %v - type %v", string(jsb), txPRV.Version, txPRV.Type)
			return nil
		}
	}
	var clonedSig []byte = nil
	if sig != nil {
		clonedSig = make([]byte, len(sig))
		copy(clonedSig, sig)
	}
	var clonedPk []byte = nil
	if pubkey != nil {
		clonedPk = make([]byte, len(pubkey))
		copy(clonedPk, pubkey)
	}
	result.Info = clonedInfo
	result.Proof = clonedProof
	result.Sig = clonedSig
	result.SigPubKey = clonedPk
	result.Info = clonedInfo

	return result
}

type TxToken struct {
	Tx             Tx                  `json:"Tx"`
	TokenData      TxTokenDataVersion2 `json:"TxTokenPrivacyData"`
	cachedTxNormal *Tx
}
func (tx *TxToken) Hash() *common.Hash {
	firstHash := tx.Tx.Hash()
	secondHash, err := tx.TokenData.Hash()
	if err != nil {
		return nil
	}
	result := common.HashH(append(firstHash[:], secondHash[:]...))
	return &result
}
func (td TxTokenDataVersion2) ToCompatTokenData(ttx metadata.Transaction) tx_generic.TxTokenData {
	return tx_generic.TxTokenData{
		TxNormal:       ttx,
		PropertyID:     td.PropertyID,
		PropertyName:   td.PropertyName,
		PropertySymbol: td.PropertySymbol,
		Type:           td.Type,
		Mintable:       td.Mintable,
		Amount:         0,
	}
}
func decomposeTokenData(td tx_generic.TxTokenData) (*TxTokenDataVersion2, *Tx, error) {
	result := TxTokenDataVersion2{
		PropertyID:     td.PropertyID,
		PropertyName:   td.PropertyName,
		PropertySymbol: td.PropertySymbol,
		Type:           td.Type,
		Mintable:       td.Mintable,
	}
	tx, ok := td.TxNormal.(*Tx)
	if !ok {
		return nil, nil, errors.New("Error while casting a transaction to v2")
	}
	return &result, tx, nil
}
func (tx *TxToken) GetTxBase() metadata.Transaction {
	return &tx.Tx
}
func (tx *TxToken) SetTxBase(inTx metadata.Transaction) error {
	temp, ok := inTx.(*Tx)
	if !ok {
		return errors.New("Cannot set TxBase : wrong type")
	}
	tx.Tx = *temp
	return nil
}
func (tx *TxToken) GetTxNormal() metadata.Transaction {
	if tx.cachedTxNormal != nil {
		return tx.cachedTxNormal
	}
	result := makeTxToken(&tx.Tx, tx.TokenData.SigPubKey, tx.TokenData.Sig, tx.TokenData.Proof)
	// tx.cachedTxNormal = result
	return result
}
func (tx *TxToken) SetTxNormal(inTx metadata.Transaction) error {
	temp, ok := inTx.(*Tx)
	if !ok {
		return utils.NewTransactionErr(utils.UnexpectedError, errors.New("Cannot set TxNormal : wrong type"))
	}
	tx.TokenData.SigPubKey = temp.SigPubKey
	tx.TokenData.Sig = temp.Sig
	tx.TokenData.Proof = temp.Proof
	tx.cachedTxNormal = temp
	return nil
}

func checkIsBridgeTokenID(bridgeStateDB *statedb.StateDB, tokenID *common.Hash) error {
	isBridgeToken := false
	allBridgeTokensBytes, err := statedb.GetAllBridgeTokens(bridgeStateDB)
	if err != nil {
		return utils.NewTransactionErr(utils.TokenIDExistedError, err)
	}
	if len(allBridgeTokensBytes) > 0 {
		var allBridgeTokens []*rawdbv2.BridgeTokenInfo
		err = json.Unmarshal(allBridgeTokensBytes, &allBridgeTokens)
		if err != nil {
			return utils.NewTransactionErr(utils.TokenIDExistedError, err)
		}
		for _, bridgeTokens := range allBridgeTokens {
			if tokenID.IsEqual(bridgeTokens.TokenID) {
				isBridgeToken = true
				break
			}
		}
	}
	if !isBridgeToken {
		return utils.NewTransactionErr(utils.TokenIDExistedError, errors.New("invalid Token ID"))
	}
	return nil
}

// this signs only on the hash of the data in it
func (tx *Tx) proveToken(params *tx_generic.TxPrivacyInitParams) (bool, error) {
	utils.Logger.Log.Debugf("CREATING sub-TX (token)")
	if err := tx_generic.ValidateTxParams(params); err != nil {
		return false, err
	}

	// Init tx and params (tx and params will be changed)
	utils.Logger.Log.Warnf("init token with receivers : %v", params.PaymentInfo)
	if err := tx.InitializeTxAndParams(params); err != nil {
		return false, err
	}
	tx.SetType(common.TxCustomTokenPrivacyType)
	isBurning, err := tx.proveCA(params)
	if err != nil {
		return false, err
	}
	return isBurning, nil
}

func (txToken *TxToken) initToken(txNormal *Tx, params *tx_generic.TxTokenParams) error {
	txToken.TokenData.Type = params.TokenParams.TokenTxType
	txToken.TokenData.PropertyName = params.TokenParams.PropertyName
	txToken.TokenData.PropertySymbol = params.TokenParams.PropertySymbol
	txToken.TokenData.Mintable = params.TokenParams.Mintable

	switch params.TokenParams.TokenTxType {
	case utils.CustomTokenInit:
		{
			temp := txNormal
			temp.Proof = new(privacy.ProofV2)
			temp.Proof.Init()

			// set output coins; hash everything but commitment; save the hash to compute the new token ID later
			message := []byte{}
			if len(params.TokenParams.Receiver[0].Message) > 0 {
				if len(params.TokenParams.Receiver[0].Message) > privacy.MaxSizeInfoCoin {
					return utils.NewTransactionErr(utils.ExceedSizeInfoOutCoinError, nil)
				}
				message = params.TokenParams.Receiver[0].Message
			}
			tempPaymentInfo := &privacy.PaymentInfo{PaymentAddress: params.TokenParams.Receiver[0].PaymentAddress, Amount: params.TokenParams.Amount, Message: message}
			createdTokenCoin, errCoin := privacy.NewCoinFromPaymentInfo(tempPaymentInfo)
			if errCoin != nil {
				utils.Logger.Log.Errorf("Cannot create new coin based on payment info err %v", errCoin)
				return errCoin
			}
			if err := temp.Proof.SetOutputCoins([]privacy.Coin{createdTokenCoin}); err != nil {
				utils.Logger.Log.Errorf("Init customPrivacyToken cannot set outputCoins")
				return err
			}
			// the coin was copied onto the proof
			theCoinOnProof, ok := temp.Proof.GetOutputCoins()[0].(*privacy.CoinV2)
			if !ok {
				return utils.NewTransactionErr(utils.UnexpectedError, errors.New("coin should have been ver2"))
			}
			theCoinOnProof.SetCommitment(new(privacy.Point).Identity())
			hashInitToken, err := txToken.TokenData.Hash()
			if err != nil {
				utils.Logger.Log.Error(errors.New("can't hash this token data"))
				return utils.NewTransactionErr(utils.UnexpectedError, err)
			}

			temp.Sig = []byte{}
			temp.SigPubKey = []byte{}

			var plainTokenID *common.Hash
			if params.TokenParams.Mintable {
				propertyID, err := common.Hash{}.NewHashFromStr(params.TokenParams.PropertyID)
				if err != nil {
					return utils.NewTransactionErr(utils.TokenIDInvalidError, err, propertyID.String())
				}
				plainTokenID = propertyID
			} else {
				//NOTICE: @merman update PropertyID calculated from hash of tokendata and shardID
				newHashInitToken := common.HashH(append(hashInitToken.GetBytes(), params.ShardID))
				existed := statedb.PrivacyTokenIDExisted(params.TransactionStateDB, newHashInitToken)
				if existed {
					utils.Logger.Log.Error("INIT Tx Custom Token Privacy is Existed", newHashInitToken)
					return utils.NewTransactionErr(utils.TokenIDExistedError, errors.New("this token is existed in network"))
				}
				plainTokenID = &newHashInitToken
				utils.Logger.Log.Debugf("A new token privacy wil be issued with ID: %+v", newHashInitToken.String())
			}

			// set the unblinded asset tag
			err = theCoinOnProof.SetPlainTokenID(plainTokenID)
			if err != nil {
				return utils.NewTransactionErr(utils.UnexpectedError, err)
			}
			txToken.TokenData.PropertyID = *plainTokenID
			txToken.SetTxNormal(temp)
			return nil
		}
	case utils.CustomTokenTransfer:
		{
			propertyID, _ := common.TokenStringToHash(params.TokenParams.PropertyID)
			dbFacingTokenID := common.ConfidentialAssetID
			utils.Logger.Log.Debugf("Token %+v wil be transfered with", propertyID)

			// fee in pToken is not supported
			feeToken := uint64(0)
			txParams := tx_generic.NewTxPrivacyInitParams(
				params.SenderKey,
				params.TokenParams.Receiver,
				params.TokenParams.TokenInput,
				feeToken,
				params.HasPrivacyToken,
				params.TransactionStateDB,
				propertyID,
				nil,
				nil,
			)
			isBurning, err := txNormal.proveToken(txParams)
			if err != nil {
				return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
			}
			if isBurning {
				// show plain tokenID if this is a burning TX
				txToken.TokenData.PropertyID = *propertyID
			} else {
				// tokenID is already hidden in asset tags in coin, here we use the umbrella ID
				txToken.TokenData.PropertyID = dbFacingTokenID
			}
			txToken.SetTxNormal(txNormal)
			return nil
		}
	default:
		return utils.NewTransactionErr(utils.PrivacyTokenTxTypeNotHandleError, errors.New("can't handle this TokenTxType"))
	}
}

// this signs on the hash of both sub TXs
func (tx *Tx) provePRV(params *tx_generic.TxPrivacyInitParams) ([]privacy.PlainCoin, []*privacy.CoinV2, error) {
	outputCoins, err := utils.NewCoinV2ArrayFromPaymentInfoArray(params.PaymentInfo, params.TokenID, params.StateDB)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot parse outputCoinV2 to outputCoins, error %v ", err)
		return nil, nil, err
	}

	// inputCoins is plainCoin because it may have coinV1 with coinV2
	inputCoins := params.InputCoins

	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, nil, false, params.PaymentInfo)
	if err != nil {
		utils.Logger.Log.Errorf("Error in privacy_v2.Prove, error %v ", err)
		return nil, nil, err
	}

	if tx.ShouldSignMetaData() {
		if err := tx.signMetadata(params.SenderSK); err != nil {
			utils.Logger.Log.Error("Cannot signOnMessage txMetadata in shouldSignMetadata")
			return nil, nil, err
		}
	}

	// Get Hash of the whole txToken then sign on it
	// message := common.HashH(append(tx.Hash()[:], hashedTokenMessage...))

	return inputCoins, outputCoins, nil
}

func (txToken *TxToken) initPRV(feeTx *Tx, params *tx_generic.TxPrivacyInitParams) ([]privacy.PlainCoin, []*privacy.CoinV2, error) {
	// txTokenDataHash, err := txToken.TokenData.Hash()
	// if err != nil {
	// 	utils.Logger.Log.Errorf("Cannot calculate txPrivacyTokenData Hash, err %v", err)
	// 	return nil, nil, err
	// }
	feeTx.SetType(common.TxCustomTokenPrivacyType)
	inps, outs, err := feeTx.provePRV(params)
	if err != nil {
		return nil, nil, utils.NewTransactionErr(utils.PrivacyTokenInitPRVError, err)
	}
	// override TxCustomTokenPrivacyType type
	// txToken.SetTxBase(feeTx)

	return inps, outs, nil
}

func (txToken *TxToken) Init(paramsInterface interface{}) error {
	params, ok := paramsInterface.(*tx_generic.TxTokenParams)
	if !ok {
		return errors.New("Cannot init TxCustomTokenPrivacy because params is not correct")
	}

	// Check validate params first, before creating tx token
	// Because there are some validation must be made first
	// Please dont change their order when you dont really understand
	txPrivacyParams := tx_generic.NewTxPrivacyInitParams(
		params.SenderKey,
		params.PaymentInfo,
		params.InputCoin,
		params.FeeNativeCoin,
		params.HasPrivacyCoin,
		params.TransactionStateDB,
		nil,
		params.MetaData,
		params.Info,
	)
	jsb, _ := json.Marshal(params.TokenParams)
	utils.Logger.Log.Infof("Create TX token v2 with token params %s", string(jsb))
	if err := tx_generic.ValidateTxParams(txPrivacyParams); err != nil {
		return err
	}
	// Init tx and params (tx and params will be changed)
	tx := new(Tx)
	if err := tx.InitializeTxAndParams(txPrivacyParams); err != nil {
		return err
	}

	// Check if this tx is nonPrivacyNonInput
	// Case 1: tx ptoken transfer with ptoken fee
	// Case 2: tx Reward
	// If it is non privacy non input then return
	if check, err := tx.IsNonPrivacyNonInput(txPrivacyParams); check {
		return err
	}

	// check tx size
	limitFee := uint64(0)
	estimateTxSizeParam := tx_generic.NewEstimateTxSizeParam(2, len(params.InputCoin), len(params.PaymentInfo),
		params.HasPrivacyCoin, nil, params.TokenParams, limitFee)
	if txSize := tx_generic.EstimateTxSize(estimateTxSizeParam); txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}

	// Init PRV Fee
	inps, outs, err := txToken.initPRV(tx, txPrivacyParams)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot init token ver2: err %v", err)
		return err
	}
	txn := makeTxToken(tx, nil, nil, nil)
	// Init, prove and sign(CA) Token
	if err := txToken.initToken(txn, params); err != nil {
		utils.Logger.Log.Errorf("Cannot init token ver2: err %v", err)
		return err
	}
	tdh, err := txToken.TokenData.Hash()
	if err != nil {
		return err
	}
	message := common.HashH(append(tx.Hash()[:], tdh[:]...))
	err = tx.signOnMessage(inps, outs, txPrivacyParams, message[:])
	if err != nil {
		return err
	}

	err = txToken.SetTxBase(tx)
	jsb, _ = json.Marshal(txToken)
	utils.Logger.Log.Warnf("TX Creation complete ! The resulting token transaction is : %s", string(jsb))
	if err != nil {
		return err
	}
	// check tx size
	txSize := txToken.GetTxActualSize()
	if txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}
	return nil
}

func (txToken *TxToken) InitTxTokenSalary(otaCoin *privacy.CoinV2, privKey *privacy.PrivateKey, stateDB *statedb.StateDB, metaData metadata.Metadata, coinID *common.Hash, coinName string) error {
	var err error
	// Check validate params
	txPrivacyParams := tx_generic.NewTxPrivacyInitParams(
		privKey, []*privacy.PaymentInfo{}, nil, 0, false, stateDB, nil, metaData, nil,
	)
	if err := tx_generic.ValidateTxParams(txPrivacyParams); err != nil {
		return err
	}
	// check tx size
	publicKeyBytes := otaCoin.GetPublicKey().ToBytesS()
	if txSize := tx_generic.EstimateTxSizeOfInitTokenSalary(2, publicKeyBytes, otaCoin.GetValue(), coinName, coinID); txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}

	// Create TxToken
	var propertyID [common.HashSize]byte
	copy(propertyID[:], coinID[:])
	txToken.TokenData.PropertyID = propertyID
	txToken.TokenData.Type = utils.CustomTokenInit
	txToken.TokenData.PropertyName = coinName
	txToken.TokenData.PropertySymbol = coinName
	txToken.TokenData.Mintable = true

	tempOutputCoin := []privacy.Coin{otaCoin}
	proof := new(privacy.ProofV2)
	proof.Init()
	if err = proof.SetOutputCoins(tempOutputCoin); err != nil {
		utils.Logger.Log.Errorf("Init customPrivacyToken cannot set outputCoins")
		return err
	}

	// Init tx fee params
	tx := new(Tx)
	if err := tx.InitializeTxAndParams(txPrivacyParams); err != nil {
		return err
	}
	tx.SetType(common.TxCustomTokenPrivacyType)
	tx.SetPrivateKey(*txPrivacyParams.SenderSK)
	temp := makeTxToken(tx, []byte{}, []byte{}, proof)
	txToken.SetTxNormal(temp)

	hashedTokenMessage, err := txToken.TokenData.Hash()
	if err != nil {
		return utils.NewTransactionErr(utils.SignTxError, err)
	}

	message := common.HashH(append(tx.Hash()[:], hashedTokenMessage[:]...))
	if tx.Sig, tx.SigPubKey, err = tx_generic.SignNoPrivacy(privKey, message[:]); err != nil {
		utils.Logger.Log.Error(errors.New(fmt.Sprintf("Cannot signOnMessage tx %v\n", err)))
		return utils.NewTransactionErr(utils.SignTxError, err)
	}

	txToken.SetTxBase(tx)
	return nil
}

func (tx *TxToken) ValidateTxSalary(db *statedb.StateDB) (bool, error) {
	return false, nil
}

func (txToken *TxToken) verifySig(transactionStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	// check input transaction
	txFee := &txToken.Tx
	if txFee.GetSig() == nil || txFee.GetSigPubKey() == nil {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("input transaction must be a signed one"))
	}
	var err error

	// Verify TxToken Salary: NonPrivacyNonInput
	if txFee.GetProof() == nil {
		hashedTokenMessage, err := txToken.TokenData.Hash()
		if err != nil {
			return false, err
		}
		message := common.HashH(append(txFee.Hash()[:], hashedTokenMessage[:]...))
		if valid, err := tx_generic.VerifySigNoPrivacy(txFee.GetSig(), txFee.GetSigPubKey(), message[:]); !valid {
			if err != nil {
				utils.Logger.Log.Debugf("Error verifying signature of tx: %+v", err)
				return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
			}
			return false, nil
		}
		return true, nil
	}

	// Reform Ring
	sumOutputCoinsWithFee := tx_generic.CalculateSumOutputsWithFee(txFee.GetProof().GetOutputCoins(), txFee.GetTxFee())
	ring, err := getRingFromSigPubKeyAndLastColumnCommitment(
		txFee.GetSigPubKey(), sumOutputCoinsWithFee,
		transactionStateDB, shardID, tokenID,
	)
	if err != nil {
		utils.Logger.Log.Errorf("Error when querying database to construct mlsag ring: %v ", err)
		return false, err
	}

	// Reform MLSAG Signature
	inputCoins := txFee.GetProof().GetInputCoins()
	keyImages := make([]*privacy.Point, len(inputCoins)+1)
	for i := 0; i < len(inputCoins); i += 1 {
		if inputCoins[i].GetKeyImage() == nil {
			utils.Logger.Log.Errorf("Error when reconstructing mlsagSignature: missing keyImage")
			return false, err
		}
		keyImages[i] = inputCoins[i].GetKeyImage()
	}
	// The last column is gone, so just fill in any value
	keyImages[len(inputCoins)] = privacy.RandomPoint()
	mlsagSignature, err := getMLSAGSigFromTxSigAndKeyImages(txFee.GetSig(), keyImages)
	if err != nil {
		utils.Logger.Log.Errorf("Error when reconstructing mlsagSignature: %v ", err)
		return false, err
	}

	txTokenDataHash, err := txToken.TokenData.Hash()
	if err != nil {
		utils.Logger.Log.Errorf("Error when getting txTokenData Hash: %v ", err)
		return false, err

	}
	message := common.HashH(append(txFee.Hash()[:], txTokenDataHash[:]...))
	return mlsag.Verify(mlsagSignature, ring, message[:])
}

func (txToken TxToken) ValidateTxByItself(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, chainRetriever metadata.ChainRetriever, shardID byte, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever) (bool, error) {
	// check for proof, signature ...
	// isBatch = false
	hasPrivacyCoin, ok := boolParams["hasPrivacy"]
	if !ok {
		hasPrivacyCoin = false
	}
	valid, _, err := txToken.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, nil)
	if !valid {
		return false, err
	}

	valid, err = tx_generic.MdValidate(&txToken, hasPrivacyCoin, transactionStateDB, bridgeStateDB, shardID)
	if !valid {
		return false, err
	}
	return true, nil
}

func (txToken TxToken) ValidateTransaction(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, []privacy.Proof, error) {
	var err error
	jsb, _ := json.Marshal(txToken)
	utils.Logger.Log.Infof("Begin verifying token TX %s", string(jsb))
	txn := txToken.GetTxNormal()
	if tokenID, err = tx_generic.ParseTokenID(tokenID); err != nil {
		return false, nil, err
	}
	ok, err := txToken.verifySig(transactionStateDB, shardID, tokenID)
	if !ok {
		utils.Logger.Log.Errorf("FAILED VERIFICATION SIGNATURE ver2 (token) with tx hash %s: %+v \n", txToken.Hash().String(), err)
		return false, nil, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
	} else {
		// validate for pToken
		tokenID := txToken.TokenData.PropertyID
		switch txToken.TokenData.Type {
		case utils.CustomTokenInit:
			// handle PRV bullet proof ?
			if txToken.TokenData.Mintable {
				return true, nil, nil
			} else {
				// check exist token
				if statedb.PrivacyTokenIDExisted(transactionStateDB, tokenID) {
					return false, nil, errors.New("Cannot validate Tx Init Token. It is tx mint from User")
				}
				return true, nil, nil
			}
		case utils.CustomTokenTransfer:
			if txToken.GetType() == common.TxTokenConversionType {
				valid, err := validateConversionVer1ToVer2(txn, transactionStateDB, shardID, &tokenID)
				return valid, nil, err
			} else {
				isBatch, ok := boolParams["isBatch"]
				if !ok {
					isBatch = false
				}

				// for CA, bulletproof batching is not supported
				boolParams["isBatch"] = false
				boolParams["hasPrivacy"] = true
				resTxTokenData, _, err := txn.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, &tokenID)
				if err != nil {
					return resTxTokenData, nil, err
				}
				txFeeProof := txToken.Tx.GetProof()
				if txFeeProof == nil {
					return false, nil, errors.New("Missing proof for PRV")
				}

				boolParams["isBatch"] = isBatch
				boolParams["hasConfidentialAsset"] = false
				// when batch-verifying for PRV, bulletproof will be skipped here & verified with the whole batch
				bpValid, err := txFeeProof.Verify(boolParams, txToken.Tx.GetSigPubKey(), 0, shardID, &common.PRVCoinID, nil)
				resultProofs := []privacy.Proof{}
				if isBatch {
					resultProofs = append(resultProofs, txFeeProof)
				}

				return bpValid && resTxTokenData, resultProofs, err

			}
		default:
			return false, nil, errors.New("Cannot validate Tx Token. Unavailable type")
		}
	}
}

func (txToken TxToken) ValidateSanityData(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, error) {
	if txToken.GetType() != common.TxCustomTokenPrivacyType && txToken.GetType() != common.TxTokenConversionType {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, errors.New("txCustomTokenPrivacy.Tx should have type tp"))
	}
	txn, ok := txToken.GetTxNormal().(*Tx)
	if !ok || txn == nil {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, errors.New("TX token must have token component"))
	}
	if txToken.GetTxBase().GetProof() == nil && txn.GetProof() == nil {
		return false, errors.New("Tx Privacy Ver 2 must have a proof")
	}
	if txToken.GetTokenID().String() == common.PRVCoinID.String() {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, errors.New("cannot transfer PRV via txtoken"))
	}
	// validate metadata
	check, err := tx_generic.MdValidateSanity(&txToken, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if !check || err != nil {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, err)
	}
	check, err = tx_generic.ValidateSanity(txn, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if !check || err != nil {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, err)
	}
	check1, err1 := tx_generic.ValidateSanity(&txToken.Tx, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if !check1 || err1 != nil {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, err1)
	}
	return true, nil
}

func (txToken TxToken) GetTxActualSize() uint64 {
	jsb, err := json.Marshal(txToken)
	if err != nil {
		return 0
	}
	return uint64(math.Ceil(float64(len(jsb)) / 1024))
}

//-- OVERRIDE--
func (tx TxToken) GetVersion() int8 { return tx.Tx.Version }

func (tx *TxToken) SetVersion(version int8) { tx.Tx.Version = version }

func (tx TxToken) GetMetadataType() int {
	if tx.Tx.Metadata != nil {
		return tx.Tx.Metadata.GetType()
	}
	return metadata.InvalidMeta
}

func (tx TxToken) GetType() string { return tx.Tx.Type }

func (tx *TxToken) SetType(t string) { tx.Tx.Type = t }

func (tx TxToken) GetLockTime() int64 { return tx.Tx.LockTime }

func (tx *TxToken) SetLockTime(locktime int64) { tx.Tx.LockTime = locktime }

func (tx TxToken) GetSenderAddrLastByte() byte { return tx.Tx.PubKeyLastByteSender }

func (tx *TxToken) SetGetSenderAddrLastByte(b byte) { tx.Tx.PubKeyLastByteSender = b }

func (tx TxToken) GetTxFee() uint64 { return tx.Tx.Fee }

func (tx *TxToken) SetTxFee(fee uint64) { tx.Tx.Fee = fee }

func (tx TxToken) GetTxFeeToken() uint64 { return uint64(0) }

func (tx TxToken) GetInfo() []byte { return tx.Tx.Info }

func (tx *TxToken) SetInfo(info []byte) { tx.Tx.Info = info }

// not supported
func (tx TxToken) GetSigPubKey() []byte           { return []byte{} }
func (tx *TxToken) SetSigPubKey(sigPubkey []byte) {}
func (tx TxToken) GetSig() []byte                 { return []byte{} }
func (tx *TxToken) SetSig(sig []byte)             {}
func (tx TxToken) GetProof() privacy.Proof        { return nil }
func (tx *TxToken) SetProof(proof privacy.Proof)  {}
func (tx TxToken) GetCachedActualSize() *uint64 {
	return nil
}
func (tx *TxToken) SetCachedActualSize(sz *uint64) {}

func (tx TxToken) GetCachedHash() *common.Hash {
	return nil
}
func (tx *TxToken) SetCachedHash(h *common.Hash) {}
func (tx *TxToken) Verify(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	return false, nil
}

func (tx TxToken) GetTokenID() *common.Hash { return &tx.TokenData.PropertyID }

func (tx TxToken) GetMetadata() metadata.Metadata { return tx.Tx.Metadata }

func (tx *TxToken) SetMetadata(meta metadata.Metadata) { tx.Tx.Metadata = meta }
func (tx TxToken) GetPrivateKey() []byte {
	return tx.Tx.GetPrivateKey()
}
func (tx *TxToken) SetPrivateKey(sk []byte) {
	tx.Tx.SetPrivateKey(sk)
}

func (tx TxToken) GetReceivers() ([][]byte, []uint64) {
	return nil, nil
}

func (tx TxToken) ListSerialNumbersHashH() []common.Hash {
	result := []common.Hash{}
	if tx.Tx.GetProof() != nil {
		for _, d := range tx.Tx.GetProof().GetInputCoins() {
			hash := common.HashH(d.GetKeyImage().ToBytesS())
			result = append(result, hash)
		}
	}
	if tx.GetTxNormal().GetProof() != nil {
		for _, d := range tx.GetTxNormal().GetProof().GetInputCoins() {
			hash := common.HashH(d.GetKeyImage().ToBytesS())
			result = append(result, hash)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

func (tx TxToken) String() string {
	jsb, err := json.Marshal(tx)
	if err != nil {
		return ""
	}
	return string(jsb)
	// record := strconv.Itoa(int(tx.Tx.Version))
	// record += strconv.FormatInt(tx.Tx.LockTime, 10)
	// record += strconv.FormatUint(tx.Tx.Fee, 10)
	// if tx.Proof != nil {
	// 	record += base64.StdEncoding.EncodeToString(tx.Tx.Proof.Bytes())
	// }
	// if tx.Metadata != nil {
	// 	metadataHash := tx.Metadata.Hash()
	// 	record += metadataHash.String()
	// }
	// return record
}
func (tx *TxToken) CalculateTxValue() uint64 {
	proof := tx.GetTxNormal().GetProof()
	if proof == nil {
		return 0
	}
	if proof.GetOutputCoins() == nil || len(proof.GetOutputCoins()) == 0 {
		return 0
	}
	if proof.GetInputCoins() == nil || len(proof.GetInputCoins()) == 0 { // coinbase tx
		txValue := uint64(0)
		for _, outCoin := range proof.GetOutputCoins() {
			txValue += outCoin.GetValue()
		}
		return txValue
	}

	if tx.GetTxNormal().IsPrivacy() {
		return 0
	}

	senderPKBytes := proof.GetInputCoins()[0].GetPublicKey().ToBytesS()
	txValue := uint64(0)
	for _, outCoin := range proof.GetOutputCoins() {
		outPKBytes := outCoin.GetPublicKey().ToBytesS()
		if bytes.Equal(senderPKBytes, outPKBytes) {
			continue
		}
		txValue += outCoin.GetValue()
	}
	return txValue
}

func (tx TxToken) CheckTxVersion(maxTxVersion int8) bool {
	return !(tx.Tx.Version > maxTxVersion)
}
func (tx TxToken) ShouldSignMetaData() bool {
	if tx.Tx.GetMetadata() == nil {
		return false
	}
	return tx.Tx.GetMetadata().ShouldSignMetaData()
}

func (tx TxToken) IsSalaryTx() bool {
	if tx.Tx.GetType() != common.TxRewardType {
		return false
	}
	if len(tx.TokenData.Proof.GetInputCoins()) > 0 {
		return false
	}
	return true
}

func (tx TxToken) IsPrivacy() bool {
	// In the case of NonPrivacyNonInput, we do not have proof
	if tx.Tx.Proof == nil {
		return false
	}
	return tx.Tx.Proof.IsPrivacy()
}

func (txToken *TxToken) IsCoinsBurning(bcr metadata.ChainRetriever, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) bool {
	// get proof of pToken
	proof := txToken.GetTxNormal().GetProof()
	if proof == nil || len(proof.GetOutputCoins()) == 0 {
		return false
	}
	return txToken.GetTxNormal().IsCoinsBurning(bcr, retriever, viewRetriever, beaconHeight)
}

func (txToken *TxToken) CheckAuthorizedSender(pk []byte) (bool, error) {
	return txToken.Tx.CheckAuthorizedSender(pk)
}

func (tx *TxToken) GetReceiverData() ([]privacy.Coin, error) {
	if tx.Tx.Proof != nil && len(tx.Tx.Proof.GetOutputCoins()) > 0 {
		return tx.Tx.Proof.GetOutputCoins(), nil
	}
	return nil, nil
}

func (txToken *TxToken) GetTransferData() (bool, []byte, uint64, *common.Hash) {
	pubkeys, amounts := txToken.GetTxNormal().GetReceivers()
	if len(pubkeys) == 0 {
		utils.Logger.Log.Error("GetTransferData receive 0 output, it should has exactly 1 output")
		return false, nil, 0, &txToken.TokenData.PropertyID
	}
	if len(pubkeys) > 1 {
		utils.Logger.Log.Error("GetTransferData receiver: More than 1 receiver")
		return false, nil, 0, &txToken.TokenData.PropertyID
	}
	return true, pubkeys[0], amounts[0], &txToken.TokenData.PropertyID
}

func (txToken TxToken) ValidateType() bool {
	return txToken.Tx.GetType() == common.TxCustomTokenPrivacyType
}

func (txToken *TxToken) ValidateTxWithCurrentMempool(mr metadata.MempoolRetriever) error {
	//Validate duplicate OTA
	poolOTAHashH := mr.GetOTAHashH()
	err := txToken.validateDuplicateOTAsWithCurrentMempool(poolOTAHashH)
	if err != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.DoubleSpendError, err)
	}

	//Validate duplicate serial numbers
	poolSerialNumbersHashH := mr.GetSerialNumbersHashH()
	err = txToken.validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH)
	if err != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.DoubleSpendError, err)
	}
	// TODO: will move this to mempool process
	if txToken.TokenData.Type == utils.CustomTokenInit && txToken.Tx.GetMetadata() == nil {
		initTokenID := txToken.TokenData.PropertyID
		txsInMem := mr.GetTxsInMem()
		for _, tx := range txsInMem {
			// try parse to TxTokenBase
			var tokenTx, ok = tx.Tx.(tx_generic.TransactionToken)
			if ok {
				txTokenData := tokenTx.GetTxTokenData()
				if txTokenData.Type == utils.CustomTokenInit && tokenTx.GetMetadata() == nil {
					// check > 1 tx init token by the same token ID
					if txTokenData.PropertyID.IsEqual(&initTokenID) {
						return utils.NewTransactionErr(utils.TokenIDInvalidError, fmt.Errorf("had already tx for initing token ID %s in pool", txTokenData.PropertyID.String()), txTokenData.PropertyID.String())
					}
				}
			}
		}
	}
	return nil
}

func (txToken *TxToken) validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH map[common.Hash][]common.Hash) error {
	// check proof of PRV and pToken
	if txToken.Tx.GetProof() == nil && txToken.GetTxNormal().GetProof() == nil {
		return errors.New("empty tx")
	}

	// collect serial number for PRV
	temp := make(map[common.Hash]interface{})
	if txToken.Tx.GetProof() != nil {
		for _, desc := range txToken.Tx.GetProof().GetInputCoins() {
			hash := common.HashH(desc.GetKeyImage().ToBytesS())
			temp[hash] = nil
		}
	}
	// collect serial number for pToken
	txNormalProof := txToken.GetTxNormal().GetProof()
	if txNormalProof != nil {
		for _, desc := range txNormalProof.GetInputCoins() {
			hash := common.HashH(desc.GetKeyImage().ToBytesS())
			temp[hash] = nil
		}
	}

	// check with pool serial number in mempool
	for _, listSerialNumbers := range poolSerialNumbersHashH {
		for _, serialNumberHash := range listSerialNumbers {
			if _, ok := temp[serialNumberHash]; ok {
				return errors.New("double spend")
			}
		}
	}
	return nil
}

func (txToken *TxToken) validateDuplicateOTAsWithCurrentMempool(poolOTAHashH map[common.Hash][]common.Hash) error {
	if txToken.GetTxBase().GetProof() == nil {
		return nil
	}
	temp := make(map[common.Hash]interface{})
	for _, outputCoin := range txToken.GetTxBase().GetProof().GetOutputCoins() {
		//Skip coins sent to the burning address
		if wallet.IsPublicKeyBurningAddress(outputCoin.GetPublicKey().ToBytesS()){
			continue
		}
		hash := common.HashH(outputCoin.GetPublicKey().ToBytesS())
		temp[hash] = nil
	}

	if txToken.GetTxNormal().GetProof() == nil {
		return nil
	}

	for _, outputCoin := range txToken.GetTxNormal().GetProof().GetOutputCoins() {
		//Skip coins sent to the burning address
		if wallet.IsPublicKeyBurningAddress(outputCoin.GetPublicKey().ToBytesS()){
			continue
		}
		hash := common.HashH(outputCoin.GetPublicKey().ToBytesS())
		temp[hash] = nil
	}

	for _, listOTAs := range poolOTAHashH {
		for _, otaHash := range listOTAs {
			if _, ok := temp[otaHash]; ok {
				return fmt.Errorf("duplicate OTA with current mempool %v",
					otaHash.String())
			}
		}
	}
	return nil
}

func (txToken *TxToken) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *metadata.MintData, shardID byte, bcr metadata.ChainRetriever, accumulatedValues *metadata.AccumulatedValues, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever) (bool, error) {
	if !txToken.TokenData.Mintable {
		return true, nil
	}
	meta := txToken.Tx.GetMetadata()
	if meta == nil {
		utils.Logger.Log.Error("Mintable custom token must contain metadata")
		return false, nil
	}
	if !meta.IsMinerCreatedMetaType() {
		return false, nil
	}
	return meta.VerifyMinerCreatedTxBeforeGettingInBlock(mintData, shardID, txToken, bcr, accumulatedValues, retriever, viewRetriever)
}

func (txToken *TxToken) GetTxTokenData() tx_generic.TxTokenData {
	return txToken.TokenData.ToCompatTokenData(txToken.GetTxNormal())
}

func (txToken *TxToken) SetTxTokenData(data tx_generic.TxTokenData) error {
	td, txN, err := decomposeTokenData(data)
	if err == nil {
		txToken.TokenData = *td
		return txToken.SetTxNormal(txN)
	}
	return err
}

func (txToken *TxToken) GetTxMintData() (bool, privacy.Coin, *common.Hash, error) {
	tokenID := txToken.TokenData.PropertyID
	return tx_generic.GetTxMintData(txToken.GetTxNormal(), &tokenID)
}

func (txToken *TxToken) GetTxBurnData() (bool, privacy.Coin, *common.Hash, error) {
	tokenID := txToken.TokenData.PropertyID
	isBurn, burnCoin, _, err := txToken.GetTxNormal().GetTxBurnData()
	return isBurn, burnCoin, &tokenID, err
}

func (txToken *TxToken) GetTxFullBurnData() (bool, privacy.Coin, privacy.Coin, *common.Hash, error) {
	isBurnToken, burnToken, burnedTokenID, errToken :=  txToken.GetTxBurnData()
	isBurnPrv, burnPrv, _, errPrv := txToken.GetTxBase().GetTxBurnData()

	if errToken != nil && errPrv != nil {
		return false, nil, nil, nil, fmt.Errorf("%v and %v", errPrv, errToken)
	}

	return isBurnPrv || isBurnToken, burnToken, burnPrv, burnedTokenID, nil
}

func (tx *TxToken) ValidateDoubleSpendWithBlockchain(shardID byte, stateDB *statedb.StateDB, tokenID *common.Hash) error {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return err
	}
	if tokenID != nil {
		err := prvCoinID.SetBytes(tokenID.GetBytes())
		if err != nil {
			return err
		}
	}
	if tx.Tx.Proof == nil {
		return nil
	}
	err = tx.Tx.ValidateDoubleSpendWithBlockchain(shardID, stateDB, nil)
	if err != nil {
		return err
	}
	if tx.GetTxNormal().GetProof() == nil {
		return nil
	}
	err = tx.GetTxNormal().ValidateDoubleSpendWithBlockchain(shardID, stateDB, prvCoinID)
	return err
}

func (txToken *TxToken) ValidateTxWithBlockChain(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error {
	err := txToken.ValidateDoubleSpendWithBlockchain(shardID, stateDB, txToken.GetTokenID())
	return err
}

func (tx *TxToken) ValidateTxReturnStaking(stateDB *statedb.StateDB) bool { return true }

func (txToken *TxToken) UnmarshalJSON(data []byte) error {
	var err error
	type TxTokenHolder struct {
		Tx                 json.RawMessage
		TxTokenPrivacyData json.RawMessage
	}
	var holder TxTokenHolder
	if err = json.Unmarshal(data, &holder); err != nil {
		return err
	}

	if err = json.Unmarshal(holder.Tx, &txToken.Tx); err != nil {
		return err
	}

	switch txToken.Tx.Type {
	case common.TxTokenConversionType:
		if txToken.Tx.Version != utils.TxConversionVersion12Number {
			return utils.NewTransactionErr(utils.PrivacyTokenJsonError, errors.New("Error while unmarshalling TX token v2 : wrong proof version"))
		}
		txToken.TokenData.Proof = &privacy.ProofForConversion{}
		txToken.TokenData.Proof.Init()
	case common.TxCustomTokenPrivacyType:
		if txToken.Tx.Version != utils.TxVersion2Number {
			return utils.NewTransactionErr(utils.PrivacyTokenJsonError, errors.New("Error while unmarshalling TX token v2 : wrong proof version"))
		}
		txToken.TokenData.Proof = &privacy.ProofV2{}
		txToken.TokenData.Proof.Init()
	default:
		return utils.NewTransactionErr(utils.PrivacyTokenJsonError, errors.New("Error while unmarshalling TX token v2 : wrong proof type"))
	}

	err = json.Unmarshal(holder.TxTokenPrivacyData, &txToken.TokenData)
	if err != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.PrivacyTokenJsonError, err)
	}
	// proof := txToken.TokenData.Proof.(*privacy.ProofV2).GetAggregatedRangeProof().(*privacy.AggregatedRangeProofV2)
	// fmt.Printf("Unmarshalled proof into token data: %v\n", agg)
	txToken.cachedTxNormal = makeTxToken(&txToken.Tx, txToken.TokenData.SigPubKey, txToken.TokenData.Sig, txToken.TokenData.Proof)
	return nil
}

func (txToken TxToken) ListOTAHashH() []common.Hash {
	result := make([]common.Hash, 0)

	//Retrieve PRV output coins
	if txToken.GetTxBase().GetProof() != nil {
		for _, outputCoin := range txToken.GetTxBase().GetProof().GetOutputCoins() {
			//Discard coins sent to the burning address
			if wallet.IsPublicKeyBurningAddress(outputCoin.GetPublicKey().ToBytesS()){
				continue
			}
			hash := common.HashH(outputCoin.GetPublicKey().ToBytesS())
			result = append(result, hash)
		}
	}

	//Retrieve token output coins
	if txToken.GetTxNormal().GetProof() != nil {
		for _, outputCoin := range txToken.GetTxNormal().GetProof().GetOutputCoins() {
			//Discard coins sent to the burning address
			if wallet.IsPublicKeyBurningAddress(outputCoin.GetPublicKey().ToBytesS()){
				continue
			}
			hash := common.HashH(outputCoin.GetPublicKey().ToBytesS())
			result = append(result, hash)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

