package tx_ver2

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"

)

type TxToken struct {
	tx_generic.TxTokenBase
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
	if err := tx.InitializeTxAndParams(params); err != nil {
		return false, err
	}
	isBurning, err := tx.proveCA(params)
	if err != nil {
		return false, err
	}
	return isBurning, nil
}

func (txToken *TxToken) initToken(params *tx_generic.TxTokenParams) error {
	txToken.TxTokenData.SetType(params.TokenParams.TokenTxType)
	txToken.TxTokenData.SetPropertyName(params.TokenParams.PropertyName)
	txToken.TxTokenData.SetPropertySymbol(params.TokenParams.PropertySymbol)
	txToken.TxTokenData.SetMintable(params.TokenParams.Mintable)

	var handled = false
	switch params.TokenParams.TokenTxType {
	case utils.CustomTokenInit:
		// case init a new privacy custom token
		{
			handled = true
			txToken.TxTokenData.SetAmount(params.TokenParams.Amount)

			message := []byte{}
			if len(params.TokenParams.Receiver[0].Message) > 0 {
				if len(params.TokenParams.Receiver[0].Message) > privacy.MaxSizeInfoCoin {
					return utils.NewTransactionErr(utils.ExceedSizeInfoOutCoinError, nil)
				}
				message = params.TokenParams.Receiver[0].Message
			}
			tempPaymentInfo := &privacy.PaymentInfo{PaymentAddress: params.TokenParams.Receiver[0].PaymentAddress, Amount: params.TokenParams.Amount, Message: message}
			c, errCoin := privacy.NewCoinFromPaymentInfo(tempPaymentInfo)
			if errCoin != nil {
				utils.Logger.Log.Errorf("Cannot create new coin based on payment info err %v", errCoin)
				return errCoin
			}

			temp := new(Tx)
			temp.SetVersion(utils.TxVersion2Number)
			temp.SetType(common.TxNormalType)
			temp.Proof = new(privacy.ProofV2)
			temp.Proof.Init()
			if err := temp.Proof.SetOutputCoins([]privacy.Coin{c}); err != nil {
				utils.Logger.Log.Errorf("Init customPrivacyToken cannot set outputCoins")
				return err
			}
			var err error
			if temp.PubKeyLastByteSender, err = params.InputCoin[0].GetShardID(); err != nil {
				return utils.NewTransactionErr(utils.GetShardIDByPublicKeyError, err)
			}
			//temp.sigPrivKey = *params.senderKey

			temp.Sig, _, err = tx_generic.SignNoPrivacy(params.SenderKey, temp.Hash()[:])
			if err != nil {
				utils.Logger.Log.Error(errors.New("can't signOnMessage this tx"))
				return utils.NewTransactionErr(utils.SignTxError, err)
			}
			temp.SigPubKey = params.TokenParams.Receiver[0].PaymentAddress.Pk
			txToken.TxTokenData.TxNormal = temp

			theCoin, _ := txToken.TxTokenData.TxNormal.GetProof().GetOutputCoins()[0].(*privacy.CoinV2)
			theCoin.SetCommitment(new(privacy.Point).Identity())
			hashInitToken, err := txToken.TxTokenData.Hash()
			if err != nil {
				utils.Logger.Log.Error(errors.New("can't hash this token data"))
				return utils.NewTransactionErr(utils.UnexpectedError, err)
			}
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
				utils.Logger.Log.Debug("New Privacy Token %+v ", newHashInitToken)
				existed := statedb.PrivacyTokenIDExisted(params.TransactionStateDB, newHashInitToken)
				if existed {
					utils.Logger.Log.Error("INIT Tx Custom Token Privacy is Existed", newHashInitToken)
					return utils.NewTransactionErr(utils.TokenIDExistedError, errors.New("this token is existed in network"))
				}
				plainTokenID = &newHashInitToken
				utils.Logger.Log.Debugf("A new token privacy wil be issued with ID: %+v", txToken.TxTokenData.PropertyID.String())
			}

			// fmt.Printf("While init token, its ID is %s\n", plainTokenID.String())
			// set the unblinded asset tag
			err = theCoin.SetPlainTokenID(plainTokenID)
			if err!=nil{
				return utils.NewTransactionErr(utils.UnexpectedError, err)
			}
			txToken.TxTokenData.SetPropertyID(*plainTokenID)
		}
	case utils.CustomTokenTransfer:
		{
			handled = true
			// make a transfering for privacy custom token
			// fee always 0 and reuse function of normal tx for custom token ID
			// propertyID, _ := common.TokenStringToHash(params.TokenParams.PropertyID)
			// existed := statedb.PrivacyTokenIDExisted(params.TransactionStateDB, *propertyID)
			// if !existed {
			// 	if err := checkIsBridgeTokenID(params.BridgeStateDB, propertyID); err != nil {
			// 		return err
			// 	}
			// }
			// fmt.Printf("Confidential Asset Transfer! Token ID is %s\n", params.TokenParams.PropertyID)
			propertyID, _ := common.TokenStringToHash(params.TokenParams.PropertyID)
			dbFacingTokenID := common.ConfidentialAssetID
			utils.Logger.Log.Debugf("Token %+v wil be transfered with", propertyID)

			// txToken.TxTokenData.SetPropertyID(*propertyID)
			txParams := tx_generic.NewTxPrivacyInitParams(
				params.SenderKey,
				params.TokenParams.Receiver,
				params.TokenParams.TokenInput,
				params.TokenParams.Fee,
				params.HasPrivacyToken,
				params.TransactionStateDB,
				propertyID,
				nil,
				nil,
			)
			txNormal := new(Tx)
			isBurning, err := txNormal.proveToken(txParams)
			if err != nil {
				return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
			}
			txToken.TxTokenData.TxNormal = txNormal
			// tokenID is already hidden in asset tags in coin, here we use the umbrella ID
			if isBurning{
				// show plain tokenID if this is a burning TX
				txToken.TxTokenData.SetPropertyID(*propertyID)
			}else{
				txToken.TxTokenData.SetPropertyID(dbFacingTokenID)
			}
		}
	}
	if !handled {
		return utils.NewTransactionErr(utils.PrivacyTokenTxTypeNotHandleError, errors.New("can't handle this TokenTxType"))
	}
	return nil
}

// this signs on the hash of both sub TXs
func (tx *Tx) provePRV(params *tx_generic.TxPrivacyInitParams, hashedTokenMessage []byte) error {
	outputCoins, err := utils.NewCoinV2ArrayFromPaymentInfoArray(params.PaymentInfo, params.TokenID, params.StateDB)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot parse outputCoinV2 to outputCoins, error %v ", err)
		return err
	}

	// inputCoins is plainCoin because it may have coinV1 with coinV2
	inputCoins := params.InputCoins

	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, nil, false, params.PaymentInfo)
	if err != nil {
		utils.Logger.Log.Errorf("Error in privacy_v2.Prove, error %v ", err)
		return err
	}

	if tx.ShouldSignMetaData() {
		if err := tx.signMetadata(params.SenderSK); err != nil {
			utils.Logger.Log.Error("Cannot signOnMessage txMetadata in shouldSignMetadata")
			return err
		}
	}

	// Get Hash of the whole txToken then sign on it
	message := common.HashH(append(tx.Hash()[:], hashedTokenMessage...))
	err = tx.signOnMessage(inputCoins, outputCoins, params, message[:])
	return err
}

func (txToken *TxToken) initPRV(feeTx * Tx, params *tx_generic.TxPrivacyInitParams) error {
	txTokenDataHash, err := txToken.TxTokenData.Hash()
	if err != nil {
		utils.Logger.Log.Errorf("Cannot calculate txPrivacyTokenData Hash, err %v", err)
		return err
	}
	if err := feeTx.provePRV(params, txTokenDataHash[:]); err != nil {
		return utils.NewTransactionErr(utils.PrivacyTokenInitPRVError, err)
	}
	// override TxCustomTokenPrivacyType type
	feeTx.SetType(common.TxCustomTokenPrivacyType)
	txToken.Tx = feeTx

	return nil
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
	estimateTxSizeParam := tx_generic.NewEstimateTxSizeParam(len(params.InputCoin), len(params.PaymentInfo),
		params.HasPrivacyCoin, nil, params.TokenParams, limitFee)
	if txSize := tx_generic.EstimateTxSize(estimateTxSizeParam); txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}

	// Init Token first
	if err := txToken.initToken(params); err != nil {
		utils.Logger.Log.Errorf("Cannot init token ver2: err %v", err)
		return err
	}

	// Init PRV Fee on the whole transaction
	if err := txToken.initPRV(tx, txPrivacyParams); err != nil {
		utils.Logger.Log.Errorf("Cannot init token ver2: err %v", err)
		return err
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
	if txSize := tx_generic.EstimateTxSizeOfInitTokenSalary(publicKeyBytes, otaCoin.GetValue(), coinName, coinID); txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}

	// Create TxToken
	var propertyID [common.HashSize]byte
	copy(propertyID[:], coinID[:])
	txToken.TxTokenData.PropertyID = propertyID
	txToken.TxTokenData.SetType(utils.CustomTokenInit)
	txToken.TxTokenData.SetPropertyName(coinName)
	txToken.TxTokenData.SetPropertySymbol(coinName)
	txToken.TxTokenData.SetAmount(otaCoin.GetValue())
	txToken.TxTokenData.SetMintable( true)

	tempOutputCoin := []privacy.Coin{otaCoin}
	proof := new(privacy.ProofV2)
	proof.Init()
	if err = proof.SetOutputCoins(tempOutputCoin); err != nil {
		utils.Logger.Log.Errorf("Init customPrivacyToken cannot set outputCoins")
		return err
	}
	temp := new(Tx)
	temp.Version = utils.TxVersion2Number
	temp.Type = common.TxNormalType
	temp.Proof = proof
	temp.PubKeyLastByteSender = publicKeyBytes[len(publicKeyBytes)-1]
	// signOnMessage Tx
	//temp.sigPrivKey = *privKey
	if temp.Sig, temp.SigPubKey, err = tx_generic.SignNoPrivacy(privKey, temp.Hash()[:]); err != nil {
		utils.Logger.Log.Error(errors.New("can't signOnMessage this tx"))
		return utils.NewTransactionErr(utils.SignTxError, err)
	}
	temp.SigPubKey = otaCoin.GetPublicKey().ToBytesS()
	txToken.TxTokenData.TxNormal = temp

	// Init tx fee params
	tx := new(Tx)
	if err := tx.InitializeTxAndParams(txPrivacyParams); err != nil {
		return err
	}
	tx.SetType(common.TxCustomTokenPrivacyType)
	tx.SetPrivateKey(*txPrivacyParams.SenderSK)

	hashedTokenMessage := txToken.TxTokenData.TxNormal.Hash()

	message := common.HashH(append(tx.Hash()[:], hashedTokenMessage[:]...))
	if tx.Sig, tx.SigPubKey, err = tx_generic.SignNoPrivacy(privKey, message[:]); err != nil {
		utils.Logger.Log.Error(errors.New(fmt.Sprintf("Cannot signOnMessage tx %v\n", err)))
		return utils.NewTransactionErr(utils.SignTxError, err)
	}

	txToken.SetTxBase(tx)
	return nil
}

func (txToken *TxToken) verifySig(transactionStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	// check input transaction
	txFee := txToken.Tx
	if txFee.GetSig() == nil || txFee.GetSigPubKey() == nil {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("input transaction must be a signed one"))
	}
	var err error

	// Verify TxToken Salary: NonPrivacyNonInput
	if txFee.GetProof() == nil {
		hashedTokenMessage := txToken.TxTokenData.TxNormal.Hash()
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
		if inputCoins[i].GetKeyImage()==nil {
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

	txTokenDataHash, err := txToken.TxTokenData.Hash()
	if err != nil {
		utils.Logger.Log.Errorf("Error when getting txTokenData Hash: %v ", err)
		return false, err

	}
	message := common.HashH(append(txFee.Hash()[:], txTokenDataHash[:]...))
	return mlsag.Verify(mlsagSignature, ring, message[:])
}

func (txToken TxToken) ValidateTxByItself(hasPrivacyCoin bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, chainRetriever metadata.ChainRetriever, shardID byte, isNewTransaction bool, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever) (bool, error) {
	// check for proof, signature ...
	if ok, err := txToken.ValidateTransaction(hasPrivacyCoin, transactionStateDB, bridgeStateDB, shardID, nil, false, isNewTransaction); !ok {
		return false, err
	}
	meta := txToken.GetMetadata()
	if meta != nil {
		validateMetadata := meta.ValidateMetadataByItself()
		if !validateMetadata {
			return validateMetadata, utils.NewTransactionErr(utils.UnexpectedError, errors.New("Metadata is invalid"))
		}
		return validateMetadata, nil
	}
	return true, nil
}

func (txToken TxToken) ValidateTransaction(hasPrivacyCoin bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash, isBatch bool, isNewTransaction bool) (bool, error) {
	var err error
	if tokenID, err = tx_generic.ParseTokenID(tokenID); err != nil {
		return false, err
	}
	ok, err := txToken.verifySig(transactionStateDB, shardID, tokenID)
	if !ok {
		utils.Logger.Log.Errorf("FAILED VERIFICATION SIGNATURE ver2 (token) with tx hash %s: %+v \n", txToken.Hash().String(), err)
		return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
	}else {
		// validate for pToken
		// TODO : decide which tokenID to put here when init, and when transfer
		tokenID := txToken.TxTokenData.PropertyID
		switch txToken.TxTokenData.Type {
		case utils.CustomTokenInit:
			if txToken.TxTokenData.Mintable {
				return true, nil
			} else {
				// check exist token
				if statedb.PrivacyTokenIDExisted(transactionStateDB, tokenID) {
					return false, errors.New("Cannot validate Tx Init Token. It is tx mint from User")
				}
				return true, nil
			}
		case utils.CustomTokenTransfer:
			if txToken.GetType() == common.TxTokenConversionType {
				return validateConversionVer1ToVer2(txToken.TxTokenData.TxNormal, transactionStateDB, shardID, &tokenID)
			} else {
				resTxTokenData, err :=  txToken.TxTokenData.TxNormal.ValidateTransaction(
					true,
					transactionStateDB, bridgeStateDB, shardID, &tokenID, isBatch, isNewTransaction)
				if err!= nil{
					return resTxTokenData, err
				}
				txFeeProof := txToken.Tx.GetProof()
				if txFeeProof == nil {
					return resTxTokenData, nil
				}
				resTxFee, err := txFeeProof.Verify(false, txToken.Tx.GetSigPubKey(), 0, shardID, &common.PRVCoinID, isBatch, nil)
				return resTxFee && resTxTokenData, err

			}
		default:
			return false, errors.New("Cannot validate Tx Token. Unavailable type")
		}
	}
}

func (txToken TxToken) ValidateSanityData(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, error) {
	if txToken.GetTxBase() == nil && txToken.TxTokenData.TxNormal == nil {
		return false, errors.New("Malform TX token")
	}
	if txToken.GetTxBase().GetProof() == nil && txToken.TxTokenData.TxNormal.GetProof() == nil {
		return false, errors.New("Tx Privacy Ver 2 must have a proof")
	}
	if txToken.GetTokenID().String() == common.PRVCoinID.String(){
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, errors.New("cannot transfer PRV via txtoken"))
	}
	// validate metadata
	check, err := tx_generic.ValidateSanityMetadata(&txToken, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if !check || err != nil {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, err)
	}
	// validate sanity for tx pToken + metadata
	check, err = tx_generic.ValidateSanityTxWithoutMetadata(txToken.TxTokenData.TxNormal, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if !check || err != nil {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, err)
	}
	// validate sanity for tx pToken + without metadata
	check1, err1 := tx_generic.ValidateSanityTxWithoutMetadata(txToken.Tx, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if !check1 || err1 != nil {
		return false, utils.NewTransactionErr(utils.InvalidSanityDataPrivacyTokenError, err1)
	}
	return true, nil
}

func (txToken TxToken) GetTxActualSize() uint64 {
	sizeTx := tx_generic.GetTxActualSizeInBytes(txToken.Tx)

	if &txToken.TxTokenData != nil {
		sizeTx += tx_generic.GetTxActualSizeInBytes(txToken.TxTokenData.TxNormal)
		sizeTx += uint64(len(txToken.TxTokenData.PropertyName))
		sizeTx += uint64(len(txToken.TxTokenData.PropertySymbol))
		sizeTx += uint64(len(txToken.TxTokenData.PropertyID))
		sizeTx += 4 // Type
		sizeTx += 1 // Mintable
		sizeTx += 8 // Amount
	}
	meta := txToken.GetMetadata()
	// fmt.Println(meta)
	if meta != nil {
		sizeTx += meta.CalculateSize()
	}

	result := uint64(math.Ceil(float64(sizeTx) / 1024))
	return result
}

func (txToken *TxToken) UnmarshalJSON(data []byte) error {
	var err error
	txToken.Tx = &Tx{}
	if err = json.Unmarshal(data, txToken.Tx); err != nil {
		return err
	}

	temp := &struct {
		TxTokenData tx_generic.TxTokenData `json:"TxTokenPrivacyData"`
	}{}
	temp.TxTokenData.TxNormal = &Tx{}
	err = json.Unmarshal(data, &temp)
	if err != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.PrivacyTokenJsonError, err)
	}
	txToken.TxTokenData = temp.TxTokenData
	if txToken.Tx.GetMetadata() != nil && txToken.Tx.GetMetadata().GetType() == 81 {
		if txToken.TxTokenData.Amount == 37772966455153490 {
			txToken.TxTokenData.Amount = 37772966455153487
		}
	}
	return nil
}