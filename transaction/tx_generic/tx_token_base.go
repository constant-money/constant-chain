// Because txprivacytoken version 1 had a bug
// txprivacytoken in later version will not use the same base with txtokenversion1
// So we duplicate some code from ver1 to ver2 and not use any embedding

package tx_generic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction/utils"
)

type Tx = metadata.Transaction

type TxTokenBase struct {
	Tx
	TxTokenData TxTokenData `json:"TxTokenPrivacyData"`
	cachedHash  *common.Hash
}

type TxTokenParams struct {
	SenderKey          *privacy.PrivateKey
	PaymentInfo        []*privacy.PaymentInfo
	InputCoin          []privacy.PlainCoin
	FeeNativeCoin      uint64
	TokenParams        *TokenParam
	TransactionStateDB *statedb.StateDB
	BridgeStateDB      *statedb.StateDB
	MetaData           metadata.Metadata
	HasPrivacyCoin     bool
	HasPrivacyToken    bool
	ShardID            byte
	Info               []byte
}

// CustomTokenParamTx - use for rpc request json body
type TokenParam struct {
	PropertyID     string                 `json:"TokenID"`
	PropertyName   string                 `json:"TokenName"`
	PropertySymbol string                 `json:"TokenSymbol"`
	Amount         uint64                 `json:"TokenAmount"`
	TokenTxType    int                    `json:"TokenTxType"`
	Receiver       []*privacy.PaymentInfo `json:"TokenReceiver"`
	TokenInput     []privacy.PlainCoin       `json:"TokenInput"`
	Mintable       bool                   `json:"TokenMintable"`
	Fee            uint64                 `json:"TokenFee"`
}

func NewTxTokenParams(senderKey *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoin []privacy.PlainCoin,
	feeNativeCoin uint64,
	tokenParams *TokenParam,
	transactionStateDB *statedb.StateDB,
	metaData metadata.Metadata,
	hasPrivacyCoin bool,
	hasPrivacyToken bool,
	shardID byte,
	info []byte,
	bridgeStateDB *statedb.StateDB) *TxTokenParams {
	params := &TxTokenParams{
		ShardID:            shardID,
		PaymentInfo:        paymentInfo,
		MetaData:           metaData,
		TransactionStateDB: transactionStateDB,
		BridgeStateDB:      bridgeStateDB,
		FeeNativeCoin:      feeNativeCoin,
		HasPrivacyCoin:     hasPrivacyCoin,
		HasPrivacyToken:    hasPrivacyToken,
		InputCoin:          inputCoin,
		SenderKey:          senderKey,
		TokenParams:        tokenParams,
		Info:               info,
	}
	return params
}

// ========== Get/Set FUNCTION ============

func (txToken TxTokenBase) GetTxBase() metadata.Transaction    { return txToken.Tx }
func (txToken *TxTokenBase) SetTxBase(tx metadata.Transaction) error{ 
	txToken.Tx = tx 
	return nil
}
func (txToken TxTokenBase) GetTxNormal() metadata.Transaction    { return txToken.TxTokenData.TxNormal }
func (txToken *TxTokenBase) SetTxNormal(tx metadata.Transaction) error{ 
	txToken.TxTokenData.TxNormal = tx 
	return nil
}
func (txToken TxTokenBase) GetTxTokenData() TxTokenData { return txToken.TxTokenData }
func (txToken *TxTokenBase) SetTxTokenData(data TxTokenData)error { 
	txToken.TxTokenData = data
	return nil
}

func (txToken TxTokenBase) GetTxMintData() (bool, privacy.Coin, *common.Hash, error) {
	tokenID := txToken.TxTokenData.GetPropertyID()
	return GetTxMintData(txToken.TxTokenData.TxNormal, &tokenID)
}

func (txToken TxTokenBase) GetTxBurnData() (bool, privacy.Coin, *common.Hash, error) {
	tokenID := txToken.TxTokenData.GetPropertyID()
	isBurn, burnCoin, _, err := txToken.TxTokenData.TxNormal.GetTxBurnData()
	return isBurn, burnCoin, &tokenID, err
}

func (txToken TxTokenBase) GetTxFullBurnData() (bool, privacy.Coin, privacy.Coin, *common.Hash, error) {
	isBurnToken, burnToken, burnedTokenID, errToken :=  txToken.GetTxBurnData()
	isBurnPrv, burnPrv, _, errPrv := txToken.GetTxBase().GetTxBurnData()

	if errToken != nil && errPrv != nil {
		return false, nil, nil, nil, fmt.Errorf("%v and %v", errPrv, errToken)
	}

	return isBurnPrv || isBurnToken, burnPrv, burnToken, burnedTokenID, nil
}
// ========== CHECK FUNCTION ===========

func (txToken TxTokenBase) CheckAuthorizedSender(publicKey []byte) (bool, error) {
	sigPubKey := txToken.TxTokenData.TxNormal.GetSigPubKey()
	if bytes.Equal(sigPubKey, publicKey) {
		return true, nil
	} else {
		return false, nil
	}
}

func (txToken TxTokenBase) IsSalaryTx() bool {
	if txToken.GetType() != common.TxRewardType {
		return false
	}
	if txToken.GetProof() != nil {
		return false
	}
	if len(txToken.TxTokenData.TxNormal.GetProof().GetInputCoins()) > 0 {
		return false
	}
	return true
}

// ==========  PARSING JSON FUNCTIONS ==========

func (txToken TxTokenBase) MarshalJSON() ([]byte, error) {
	type TemporaryTxToken struct {
		TxBase
		TxTokenData TxTokenData `json:"TxTokenPrivacyData"`
	}
	tempTx := TemporaryTxToken{}
	tempTx.TxTokenData = txToken.GetTxTokenData()
	tx := txToken.GetTxBase()
	if tx == nil {
		return nil, errors.New("Cannot unmarshal transaction: txfee cannot be nil")
	}
	tempTx.TxBase.SetVersion(tx.GetVersion())
	tempTx.TxBase.SetType(tx.GetType())
	tempTx.TxBase.SetLockTime(tx.GetLockTime())
	tempTx.TxBase.SetTxFee(tx.GetTxFee())
	tempTx.TxBase.SetInfo(tx.GetInfo())
	tempTx.TxBase.SetSigPubKey(tx.GetSigPubKey())
	tempTx.TxBase.SetSig(tx.GetSig())
	tempTx.TxBase.SetProof(tx.GetProof())
	tempTx.TxBase.SetGetSenderAddrLastByte(tx.GetSenderAddrLastByte())
	tempTx.TxBase.SetMetadata(tx.GetMetadata())
	tempTx.TxBase.SetGetSenderAddrLastByte(tx.GetSenderAddrLastByte())

	return json.Marshal(tempTx)
}

func (txToken TxTokenBase) String() string {
	// get hash of tx
	record := txToken.Tx.Hash().String()
	// add more hash of tx custom token data privacy
	tokenPrivacyDataHash, _ := txToken.TxTokenData.Hash()
	record += tokenPrivacyDataHash.String()

	meta := txToken.GetMetadata()
	if meta != nil {
		record += string(meta.Hash()[:])
	}
	return record
}

func (txToken TxTokenBase) JSONString() string {
	data, err := json.MarshalIndent(txToken, "", "\t")
	if err != nil {
		utils.Logger.Log.Error(err)
		return ""
	}
	return string(data)
}

// =================== FUNCTIONS THAT GET STUFF ===================

// Hash returns the hash of all fields of the transaction
func (txToken *TxTokenBase) Hash() *common.Hash {
	if txToken.cachedHash != nil {
		return txToken.cachedHash
	}
	// final hash
	hash := common.HashH([]byte(txToken.String()))
	txToken.cachedHash = &hash
	return &hash
}

// Get SigPubKey of ptoken
func (txToken TxTokenBase) GetSigPubKey() []byte {
	return txToken.TxTokenData.TxNormal.GetSigPubKey()
}

// GetTxFeeToken - return Token Fee use to pay for privacy token Tx
func (txToken TxTokenBase) GetTxFeeToken() uint64 {
	return txToken.TxTokenData.TxNormal.GetTxFee()
}

func (txToken TxTokenBase) GetTokenID() *common.Hash {
	return &txToken.TxTokenData.PropertyID
}

func (txToken TxTokenBase) GetTransferData() (bool, []byte, uint64, *common.Hash) {
	pubkeys, amounts := txToken.TxTokenData.TxNormal.GetReceivers()
	if len(pubkeys) == 0 {
		utils.Logger.Log.Error("GetTransferData receive 0 output, it should has exactly 1 output")
		return false, nil, 0, &txToken.TxTokenData.PropertyID
	}
	if len(pubkeys) > 1 {
		utils.Logger.Log.Error("GetTransferData receiver: More than 1 receiver")
		return false, nil, 0, &txToken.TxTokenData.PropertyID
	}
	return true, pubkeys[0], amounts[0], &txToken.TxTokenData.PropertyID
}

// CalculateBurnAmount - get tx value for pToken
func (txToken TxTokenBase) CalculateTxValue() uint64 {
	proof := txToken.TxTokenData.TxNormal.GetProof()
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

	if txToken.TxTokenData.TxNormal.IsPrivacy() {
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

func (txToken TxTokenBase) ListSerialNumbersHashH() []common.Hash {
	tx := txToken.Tx
	result := []common.Hash{}
	if tx.GetProof() != nil {
		for _, d := range tx.GetProof().GetInputCoins() {
			hash := common.HashH(d.GetKeyImage().ToBytesS())
			result = append(result, hash)
		}
	}
	customTokenPrivacy := txToken.TxTokenData
	if customTokenPrivacy.TxNormal.GetProof() != nil {
		for _, d := range customTokenPrivacy.TxNormal.GetProof().GetInputCoins() {
			hash := common.HashH(d.GetKeyImage().ToBytesS())
			result = append(result, hash)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

// GetTxFee - return fee PRV of Tx which contain privacy token Tx
func (txToken TxTokenBase) GetTxFee() uint64 {
	return txToken.Tx.GetTxFee()
}

// ================== NORMAL INIT FUNCTIONS ===================

func EstimateTxSizeOfInitTokenSalary(version int, publicKey []byte, amount uint64, coinName string, coinID *common.Hash) uint64 {
	receiver := &privacy.PaymentInfo{
		Amount: amount,
		PaymentAddress: privacy.PaymentAddress{
			Pk: publicKey,
			Tk: []byte{},
		},
	}
	propString := common.TokenHashToString(coinID)
	tokenParams := &TokenParam{
		PropertyID:     propString,
		PropertyName:   coinName,
		PropertySymbol: coinName,
		Amount:         amount,
		TokenTxType:    utils.CustomTokenInit,
		Receiver:       []*privacy.PaymentInfo{receiver},
		TokenInput:     []privacy.PlainCoin{},
		Mintable:       true,
	}
	estimateTxSizeParam := NewEstimateTxSizeParam(version, 0, 0, false, nil, tokenParams, uint64(0))
	return EstimateTxSize(estimateTxSizeParam)
}

// =================== FUNCTION THAT CHECK STUFFS  ===================

// IsCoinsBurning - checking this is a burning pToken
func (txToken TxTokenBase) IsCoinsBurning(bcr metadata.ChainRetriever, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) bool {
	// get proof of pToken
	proof := txToken.TxTokenData.TxNormal.GetProof()
	if proof == nil || len(proof.GetOutputCoins()) == 0 {
		return false
	}
	return txToken.TxTokenData.TxNormal.IsCoinsBurning(bcr, retriever, viewRetriever, beaconHeight)
}

// ========== VALIDATE FUNCTIONS ===========

func (txToken TxTokenBase) ValidateType() bool {
	return txToken.Tx.GetType() == common.TxCustomTokenPrivacyType
}

func (txToken TxTokenBase) ValidateTxWithCurrentMempool(mr metadata.MempoolRetriever) error {
	poolSerialNumbersHashH := mr.GetSerialNumbersHashH()
	err := txToken.validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH)
	if err != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.DoubleSpendError, err)
	}
	// TODO: will move this to mempool process
	if txToken.TxTokenData.Type == utils.CustomTokenInit && txToken.GetMetadata() == nil {
		initTokenID := txToken.TxTokenData.PropertyID
		txsInMem := mr.GetTxsInMem()
		for _, tx := range txsInMem {
			// try parse to TxTokenBase
			var tokenTx, ok = tx.Tx.(TransactionToken)
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

func (txToken TxTokenBase) validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH map[common.Hash][]common.Hash) error {
	// check proof of PRV and pToken
	if txToken.GetProof() == nil && txToken.TxTokenData.TxNormal.GetProof() == nil {
		return errors.New("empty tx")
	}

	// collect serial number for PRV
	temp := make(map[common.Hash]interface{})
	if txToken.GetProof() != nil {
		for _, desc := range txToken.GetProof().GetInputCoins() {
			hash := common.HashH(desc.GetKeyImage().ToBytesS())
			temp[hash] = nil
		}
	}
	// collect serial number for pToken
	txNormalProof := txToken.TxTokenData.TxNormal.GetProof()
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

func (txToken TxTokenBase) ValidateTxWithBlockChain(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error {
	err := txToken.ValidateDoubleSpendWithBlockchain(shardID, stateDB, nil)
	if err != nil {
		return utils.NewTransactionErr(utils.InvalidDoubleSpendPRVError, err)
	}
	err = txToken.TxTokenData.TxNormal.ValidateDoubleSpendWithBlockchain(shardID, stateDB, txToken.GetTokenID())
	// err = txToken.TxTokenData.TxNormal.ValidateDoubleSpendWithBlockchain(shardID, stateDB, &common.ConfidentialAssetID)
	if err != nil {
		return utils.NewTransactionErr(utils.InvalidDoubleSpendPrivacyTokenError, err)
	}
	return nil
}

// ValidateSanityData - validate sanity data of PRV and pToken


// VerifyMinerCreatedTxBeforeGettingInBlock
func (txToken TxTokenBase) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *metadata.MintData, shardID byte, bcr metadata.ChainRetriever, accumulatedValues *metadata.AccumulatedValues, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever) (bool, error) {
	if !txToken.TxTokenData.Mintable {
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
	return meta.VerifyMinerCreatedTxBeforeGettingInBlock(mintData, shardID, &txToken, bcr, accumulatedValues, retriever, viewRetriever)
}
