package transaction

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	zkp "github.com/incognitochain/incognito-chain/privacy/zeroknowledge"
	"github.com/incognitochain/incognito-chain/wallet"
)

// TxCustomTokenPrivacy is class tx which is inherited from P tx(supporting privacy) for fee
// and contain data(with supporting privacy format) to support issuing and transfer a custom token(token from end-user, look like erc-20)
// Dev or end-user can use this class tx to create an token type which use personal purpose
// TxCustomTokenPrivacy is an advance format of TxNormalToken
// so that user need to spend a lot fee to create this class tx
type TxCustomTokenPrivacy struct {
	Tx                                    // inherit from normal tx of P(supporting privacy) with a high fee to ensure that tx could contain a big data of privacy for token
	TxPrivacyTokenData TxPrivacyTokenData `json:"TxTokenPrivacyData"` // supporting privacy format
	// private field, not use for json parser, only use as temp variable
	cachedHash *common.Hash // cached hash data of tx
}

func (txCustomTokenPrivacy *TxCustomTokenPrivacy) UnmarshalJSON(data []byte) error {
	tx := Tx{}
	err := json.Unmarshal(data, &tx)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(PrivacyTokenPRVJsonError, err)
	}
	temp := &struct {
		TxTokenPrivacyData TxPrivacyTokenData
	}{}
	err = json.Unmarshal(data, &temp)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(PrivacyTokenJsonError, err)
	}
	TxTokenPrivacyDataJson, err := json.MarshalIndent(temp.TxTokenPrivacyData, "", "\t")
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(UnexpectedError, err)
	}
	err = json.Unmarshal(TxTokenPrivacyDataJson, &txCustomTokenPrivacy.TxPrivacyTokenData)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(PrivacyTokenJsonError, err)
	}
	txCustomTokenPrivacy.Tx = tx

	// TODO: hotfix, remove when fixed this issue
	if tx.Metadata != nil && tx.Metadata.GetType() == 81 {
		if txCustomTokenPrivacy.TxPrivacyTokenData.Amount == 37772966455153490 {
			txCustomTokenPrivacy.TxPrivacyTokenData.Amount = 37772966455153487
		}
	}

	return nil
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) String() string {
	// get hash of tx
	record := txCustomTokenPrivacy.Tx.Hash().String()
	// add more hash of tx custom token data privacy
	tokenPrivacyDataHash, _ := txCustomTokenPrivacy.TxPrivacyTokenData.Hash()
	record += tokenPrivacyDataHash.String()
	if txCustomTokenPrivacy.Metadata != nil {
		record += string(txCustomTokenPrivacy.Metadata.Hash()[:])
	}
	return record
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) JSONString() string {
	data, err := json.MarshalIndent(txCustomTokenPrivacy, "", "\t")
	if err != nil {
		Logger.log.Error(err)
		return ""
	}
	return string(data)
}

// Hash returns the hash of all fields of the transaction
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) Hash() *common.Hash {
	if txCustomTokenPrivacy.cachedHash != nil {
		return txCustomTokenPrivacy.cachedHash
	}
	// final hash
	hash := common.HashH([]byte(txCustomTokenPrivacy.String()))
	return &hash
}

// GetTxActualSize computes the virtual size of a given transaction
// size of this tx = (normal TxNormal size) + (custom token data size)
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTxActualSize() uint64 {
	normalTxSize := txCustomTokenPrivacy.Tx.GetTxActualSize()
	tokenDataSize := uint64(0)
	tokenDataSize += txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.GetTxActualSize()
	tokenDataSize += uint64(len(txCustomTokenPrivacy.TxPrivacyTokenData.PropertyName))
	tokenDataSize += uint64(len(txCustomTokenPrivacy.TxPrivacyTokenData.PropertySymbol))
	tokenDataSize += uint64(len(txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID))
	tokenDataSize += 4 // for TxPrivacyTokenData.Type
	tokenDataSize += 8 // for TxPrivacyTokenData.Amount
	meta := txCustomTokenPrivacy.Metadata
	if meta != nil {
		tokenDataSize += meta.CalculateSize()
	}

	return normalTxSize + uint64(math.Ceil(float64(tokenDataSize)/1024))
}

func (tx TxCustomTokenPrivacy) GetTxPrivacyTokenActualSize() uint64 {
	tokenDataSize := uint64(0)
	tokenDataSize += tx.TxPrivacyTokenData.TxNormal.GetTxActualSize()
	tokenDataSize += uint64(len(tx.TxPrivacyTokenData.PropertyName))
	tokenDataSize += uint64(len(tx.TxPrivacyTokenData.PropertySymbol))
	tokenDataSize += uint64(len(tx.TxPrivacyTokenData.PropertyID))
	tokenDataSize += 4 // for TxPrivacyTokenData.Type
	tokenDataSize += 8 // for TxPrivacyTokenData.Amount

	meta := tx.TxPrivacyTokenData.TxNormal.Metadata
	if meta != nil {
		tokenDataSize += meta.CalculateSize()
	}

	return uint64(math.Ceil(float64(tokenDataSize) / 1024))
}

type TxPrivacyTokenInitParams struct {
	senderKey          *privacy.PrivateKey
	paymentInfo        []*privacy.PaymentInfo
	inputCoin          []*privacy.InputCoin
	feeNativeCoin      uint64
	tokenParams        *CustomTokenPrivacyParamTx
	transactionStateDB *statedb.StateDB
	bridgeStateDB      *statedb.StateDB
	metaData           metadata.Metadata
	hasPrivacyCoin     bool
	hasPrivacyToken    bool
	shardID            byte
	info               []byte
}

func NewTxPrivacyTokenInitParams(senderKey *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoin []*privacy.InputCoin,
	feeNativeCoin uint64,
	tokenParams *CustomTokenPrivacyParamTx,
	transactionStateDB *statedb.StateDB,
	metaData metadata.Metadata,
	hasPrivacyCoin bool,
	hasPrivacyToken bool,
	shardID byte,
	info []byte,
	bridgeStateDB *statedb.StateDB) *TxPrivacyTokenInitParams {
	params := &TxPrivacyTokenInitParams{
		shardID:            shardID,
		paymentInfo:        paymentInfo,
		metaData:           metaData,
		transactionStateDB: transactionStateDB,
		bridgeStateDB:      bridgeStateDB,
		feeNativeCoin:      feeNativeCoin,
		hasPrivacyCoin:     hasPrivacyCoin,
		hasPrivacyToken:    hasPrivacyToken,
		inputCoin:          inputCoin,
		senderKey:          senderKey,
		tokenParams:        tokenParams,
		info:               info,
	}
	return params
}

// Init -  build normal tx component and privacy custom token data
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) Init(params *TxPrivacyTokenInitParams) error {
	var err error
	// init data for tx PRV for fee
	normalTx := Tx{}
	err = normalTx.Init(NewTxPrivacyInitParams(
		params.senderKey,
		params.paymentInfo,
		params.inputCoin,
		params.feeNativeCoin,
		params.hasPrivacyCoin,
		params.transactionStateDB,
		nil,
		params.metaData,
		params.info))
	if err != nil {
		return NewTransactionErr(PrivacyTokenInitPRVError, err)
	}
	// override TxCustomTokenPrivacyType type
	normalTx.Type = common.TxCustomTokenPrivacyType
	txCustomTokenPrivacy.Tx = normalTx

	// check tx size
	limitFee := uint64(0)
	estimateTxSizeParam := NewEstimateTxSizeParam(len(params.inputCoin), len(params.paymentInfo),
		params.hasPrivacyCoin, nil, params.tokenParams, limitFee)
	if txSize := EstimateTxSize(estimateTxSizeParam); txSize > common.MaxTxSize {
		return NewTransactionErr(ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}

	// check action type and create privacy custom toke data
	var handled = false
	// Add token data component
	switch params.tokenParams.TokenTxType {
	case CustomTokenInit:
		// case init a new privacy custom token
		{
			handled = true
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.tokenParams.TokenTxType,
				PropertyName:   params.tokenParams.PropertyName,
				PropertySymbol: params.tokenParams.PropertySymbol,
				Amount:         params.tokenParams.Amount,
			}

			// issue token with data of privacy
			temp := Tx{}
			temp.Type = common.TxNormalType
			temp.Proof = new(zkp.PaymentProof)
			tempOutputCoin := make([]*privacy.OutputCoin, 1)
			tempOutputCoin[0] = new(privacy.OutputCoin)
			tempOutputCoin[0].CoinDetails = new(privacy.Coin)
			tempOutputCoin[0].CoinDetails.SetValue(params.tokenParams.Amount)
			PK, err := new(privacy.Point).FromBytesS(params.tokenParams.Receiver[0].PaymentAddress.Pk)
			if err != nil {
				return NewTransactionErr(DecompressPaymentAddressError, err)
			}
			tempOutputCoin[0].CoinDetails.SetPublicKey(PK)
			tempOutputCoin[0].CoinDetails.SetRandomness(privacy.RandomScalar())

			// set info coin for output coin
			if len(params.tokenParams.Receiver[0].Message) > 0 {
				if len(params.tokenParams.Receiver[0].Message) > privacy.MaxSizeInfoCoin {
					return NewTransactionErr(ExceedSizeInfoOutCoinError, nil)
				}
				tempOutputCoin[0].CoinDetails.SetInfo(params.tokenParams.Receiver[0].Message)
			}

			sndOut := privacy.RandomScalar()
			tempOutputCoin[0].CoinDetails.SetSNDerivator(sndOut)
			temp.Proof.SetOutputCoins(tempOutputCoin)

			// create coin commitment
			err = temp.Proof.GetOutputCoins()[0].CoinDetails.CommitAll()
			if err != nil {
				return NewTransactionErr(CommitOutputCoinError, err)
			}
			// get last byte
			lastByteSender := params.tokenParams.Receiver[0].PaymentAddress.Pk[len(params.tokenParams.Receiver[0].PaymentAddress.Pk)-1]
			temp.PubKeyLastByteSender = common.GetShardIDFromLastByte(lastByteSender)

			// sign Tx
			temp.SigPubKey = params.tokenParams.Receiver[0].PaymentAddress.Pk
			temp.sigPrivKey = *params.senderKey
			err = temp.signTx()
			if err != nil {
				Logger.log.Error(errors.New("can't sign this tx"))
				return NewTransactionErr(SignTxError, err)
			}

			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
			hashInitToken, err := txCustomTokenPrivacy.TxPrivacyTokenData.Hash()
			if err != nil {
				Logger.log.Error(errors.New("can't hash this token data"))
				return NewTransactionErr(UnexpectedError, err)
			}

			if params.tokenParams.Mintable {
				propertyID, err := common.Hash{}.NewHashFromStr(params.tokenParams.PropertyID)
				if err != nil {
					return NewTransactionErr(TokenIDInvalidError, err, propertyID.String())
				}
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = *propertyID
				txCustomTokenPrivacy.TxPrivacyTokenData.Mintable = true
			} else {
				//NOTICE: @merman update PropertyID calculated from hash of tokendata and shardID
				newHashInitToken := common.HashH(append(hashInitToken.GetBytes(), params.shardID))
				Logger.log.Debug("New Privacy Token %+v ", newHashInitToken)
				existed := statedb.PrivacyTokenIDExisted(params.transactionStateDB, newHashInitToken)
				if existed {
					Logger.log.Error("INIT Tx Custom Token Privacy is Existed", newHashInitToken)
					return NewTransactionErr(TokenIDExistedError, errors.New("this token is existed in network"))
				}
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = newHashInitToken
				Logger.log.Debugf("A new token privacy wil be issued with ID: %+v", txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID.String())
			}
		}
	case CustomTokenTransfer:
		{
			handled = true
			// make a transfering for privacy custom token
			// fee always 0 and reuse function of normal tx for custom token ID
			temp := Tx{}
			propertyID, _ := common.Hash{}.NewHashFromStr(params.tokenParams.PropertyID)
			existed := statedb.PrivacyTokenIDExisted(params.transactionStateDB, *propertyID)
			if !existed {
				isBridgeToken := false
				allBridgeTokensBytes, err := statedb.GetAllBridgeTokens(params.bridgeStateDB)
				if err != nil {
					return NewTransactionErr(TokenIDExistedError, err)
				}
				if len(allBridgeTokensBytes) > 0 {
					var allBridgeTokens []*rawdbv2.BridgeTokenInfo
					err = json.Unmarshal(allBridgeTokensBytes, &allBridgeTokens)
					if err != nil {
						return NewTransactionErr(TokenIDExistedError, err)
					}
					for _, bridgeTokens := range allBridgeTokens {
						if propertyID.IsEqual(bridgeTokens.TokenID) {
							isBridgeToken = true
							break
						}
					}
				}
				if !isBridgeToken {
					return NewTransactionErr(TokenIDExistedError, errors.New("invalid Token ID"))
				}
			}
			Logger.log.Debugf("Token %+v wil be transfered with", propertyID)
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.tokenParams.TokenTxType,
				PropertyName:   params.tokenParams.PropertyName,
				PropertySymbol: params.tokenParams.PropertySymbol,
				PropertyID:     *propertyID,
				Mintable:       params.tokenParams.Mintable,
			}
			err := temp.Init(NewTxPrivacyInitParams(params.senderKey,
				params.tokenParams.Receiver,
				params.tokenParams.TokenInput,
				params.tokenParams.Fee,
				params.hasPrivacyToken,
				params.transactionStateDB,
				propertyID,
				nil,
				nil))
			if err != nil {
				return NewTransactionErr(PrivacyTokenInitTokenDataError, err)
			}
			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
		}
	}
	if !handled {
		return NewTransactionErr(PrivacyTokenTxTypeNotHandleError, errors.New("can't handle this TokenTxType"))
	}
	return nil
}

// ValidateType - check type of tx
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateType() bool {
	return txCustomTokenPrivacy.Type == common.TxCustomTokenPrivacyType
}

// ValidateTxWithCurrentMempool - validate for serrial number use in tx is double with other tx in mempool
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateTxWithCurrentMempool(mr metadata.MempoolRetriever) error {
	// check double snd outputs in mempool
	poolSNDOutputsHashH := mr.GetSNDOutputsHashH()
	duplicateSNDs := txCustomTokenPrivacy.validateDoubleSNDOutputsWithCurrentMempool(poolSNDOutputsHashH)
	if duplicateSNDs != nil {
		return duplicateSNDs
	}

	// check double spend
	poolSerialNumbersHashH := mr.GetSerialNumbersHashH()
	err := txCustomTokenPrivacy.validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(DoubleSpendError, err)
	}
	// TODO: will move this to mempool process
	if txCustomTokenPrivacy.TxPrivacyTokenData.Type == CustomTokenInit && txCustomTokenPrivacy.GetMetadata() == nil {
		initTokenID := txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID
		txsInMem := mr.GetTxsInMem()
		for _, tx := range txsInMem {
			// try parse to TxCustomTokenPrivacy
			privacyTokenTx, ok := tx.Tx.(*TxCustomTokenPrivacy)
			if ok && privacyTokenTx.TxPrivacyTokenData.Type == CustomTokenInit && privacyTokenTx.GetMetadata() == nil {
				// check > 1 tx init token by the same token ID
				if privacyTokenTx.TxPrivacyTokenData.PropertyID.IsEqual(&initTokenID) {
					return NewTransactionErr(TokenIDInvalidError, fmt.Errorf("had already tx for initing token ID %s in pool", privacyTokenTx.TxPrivacyTokenData.PropertyID.String()), privacyTokenTx.TxPrivacyTokenData.PropertyID.String())
				}
			}
		}
	}

	return nil
}

// validateDoubleSpendTxWithCurrentMempool - from proof of PRV and pToken,
// check serrial numbers is valid,
// not double spend with any tx in mempool
// this a private func -> call by ValidateTxWithCurrentMempool
func (txCustomTokenPrivacy TxCustomTokenPrivacy) validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH map[common.Hash][]common.Hash) error {
	// check proof of PRV and pToken
	if txCustomTokenPrivacy.Proof == nil && txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof == nil {
		return errors.New("empty tx")
	}

	// collect serial number for PRV
	temp := make(map[common.Hash]interface{})
	if txCustomTokenPrivacy.Proof != nil {
		for _, desc := range txCustomTokenPrivacy.Proof.GetInputCoins() {
			hash := common.HashH(desc.CoinDetails.GetSerialNumber().ToBytesS())
			temp[hash] = nil
		}
	}
	// collect serial number for pToken
	if txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof != nil {
		for _, desc := range txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof.GetInputCoins() {
			hash := common.HashH(desc.CoinDetails.GetSerialNumber().ToBytesS())
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

func (txCustomTokenPrivacy TxCustomTokenPrivacy) validateDoubleSNDOutputsWithCurrentMempool(poolSndOutputsHashH map[common.Hash][]common.Hash) error {
	// check proof of PRV and pToken
	if txCustomTokenPrivacy.Proof == nil && txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof == nil {
		return errors.New("empty tx")
	}

	// collect serial number for PRV
	temp := make(map[common.Hash]interface{})
	if txCustomTokenPrivacy.Proof != nil {
		for _, outputCoin := range txCustomTokenPrivacy.Proof.GetOutputCoins() {
			hash := common.HashH(outputCoin.CoinDetails.GetSNDerivator().ToBytesS())
			temp[hash] = nil
		}
	}
	// collect serial number for pToken
	if txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof != nil {
		for _, outputCoin := range txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins() {
			hash := common.HashH(outputCoin.CoinDetails.GetSNDerivator().ToBytesS())
			temp[hash] = nil
		}
	}

	// check with pool serial number in mempool
	for _, listSndOutputs := range poolSndOutputsHashH {
		for _, sndHash := range listSndOutputs {
			if _, ok := temp[sndHash]; ok {
				return fmt.Errorf("duplicate snd output with current mempool %v",
					sndHash.String())
			}
		}
	}
	return nil
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateTxWithBlockChain(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error {
	err := txCustomTokenPrivacy.ValidateDoubleSpendWithBlockchain(shardID, stateDB, nil)
	if err != nil {
		return NewTransactionErr(InvalidDoubleSpendPRVError, err)
	}
	err = txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.ValidateDoubleSpendWithBlockchain(shardID, stateDB, txCustomTokenPrivacy.GetTokenID())
	if err != nil {
		return NewTransactionErr(InvalidDoubleSpendPrivacyTokenError, err)
	}
	return nil
}

// ValidateSanityData - validate sanity data of PRV and pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateSanityData(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, error) {
	if txCustomTokenPrivacy.GetType() != common.TxCustomTokenPrivacyType{
		return false, NewTransactionErr(InvalidSanityDataPrivacyTokenError, errors.New("txCustomTokenPrivacy.Tx should have type tp"))
	}
	if txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.GetType() != common.TxNormalType{
		return false, NewTransactionErr(InvalidSanityDataPrivacyTokenError, errors.New("txCustomTokenPrivacy.TxNormal should have type n"))
	}
	meta := txCustomTokenPrivacy.Tx.Metadata
	if meta != nil {
		isContinued, ok, err := meta.ValidateSanityData(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight, &txCustomTokenPrivacy)
		if err != nil || !ok || !isContinued {
			return ok, err
		}
	}

	// validate sanity data for PRV
	//result, err := txCustomTokenPrivacy.Tx.validateNormalTxSanityData()
	result, err := txCustomTokenPrivacy.Tx.ValidateSanityData(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if err != nil {
		return result, NewTransactionErr(InvalidSanityDataPRVError, err)
	}
	// validate sanity for pToken

	// check tokenID in TxPrivacyTokenData
	if txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID.String() == common.PRVIDStr {
		return false, NewTransactionErr(InvalidSanityDataPrivacyTokenError, errors.New("TokenID must not be equal PRVID"))
	}

	//result, err = txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.validateNormalTxSanityData()
	result, err = txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.ValidateSanityData(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight)
	if err != nil {
		return result, NewTransactionErr(InvalidSanityDataPrivacyTokenError, err)
	}
	return result, nil
}

// ValidateTxByItself - validate tx by itself, check signature, proof,... and metadata
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateTxByItself(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, chainRetriever metadata.ChainRetriever, shardID byte, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever) (bool, error) {
	// check for proof, signature ...
	if ok, err := txCustomTokenPrivacy.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, nil); !ok {
		return false, err
	}
	// check for metadata
	if txCustomTokenPrivacy.Metadata != nil {
		validateMetadata := txCustomTokenPrivacy.Metadata.ValidateMetadataByItself()
		if !validateMetadata {
			return validateMetadata, NewTransactionErr(UnexpectedError, errors.New("Metadata is invalid"))
		}
		return validateMetadata, nil
	}
	return true, nil
}

// ValidateTransaction - verify proof, signature, ... of PRV and pToken
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) ValidateTransaction(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	// validate for PRV
	ok, err := txCustomTokenPrivacy.Tx.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, nil)
	if ok {
		// validate for pToken
		tokenID := txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID
		if txCustomTokenPrivacy.TxPrivacyTokenData.Type == CustomTokenInit {
			if txCustomTokenPrivacy.TxPrivacyTokenData.Mintable {
				return true, nil
			} else {
				// check exist token
				if statedb.PrivacyTokenIDExisted(transactionStateDB, tokenID) {
					return false, nil
				}
				return true, nil
			}
		} else {
			boolParams["hasPrivacy"] = txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.IsPrivacy()
			return txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, &tokenID)
		}
	}
	return false, err
}

// GetProof - return proof PRV of tx
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetProof() *zkp.PaymentProof {
	return txCustomTokenPrivacy.Proof
}

// VerifyMinerCreatedTxBeforeGettingInBlock
func (txCustomTokenPrivacy TxCustomTokenPrivacy) VerifyMinerCreatedTxBeforeGettingInBlock(txsInBlock []metadata.Transaction, txsUsed []int, insts [][]string, instsUsed []int, shardID byte, bcr metadata.ChainRetriever, accumulatedValues *metadata.AccumulatedValues, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever) (bool, error) {
	if !txCustomTokenPrivacy.TxPrivacyTokenData.Mintable {
		return true, nil
	}
	meta := txCustomTokenPrivacy.Metadata
	if meta == nil {
		Logger.log.Error("Mintable custom token must contain metadata")
		return false, nil
	}
	if !meta.IsMinerCreatedMetaType() {
		return false, nil
	}
	return meta.VerifyMinerCreatedTxBeforeGettingInBlock(txsInBlock, txsUsed, insts, instsUsed, shardID, &txCustomTokenPrivacy, bcr, accumulatedValues, nil, nil)
}

// GetTokenReceivers - return receivers in tx, who receive token
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTokenReceivers() ([][]byte, []uint64) {
	pubkeys := [][]byte{}
	amounts := []uint64{}
	// get proof pToken
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil {
		return pubkeys, amounts
	}
	// fetch payment info
	for _, coin := range proof.GetOutputCoins() {
		coinPubKey := coin.CoinDetails.GetPublicKey().ToBytesS()
		added := false
		// coinPubKey := vout.PaymentAddress.Pk
		for i, key := range pubkeys {
			if bytes.Equal(coinPubKey, key) {
				added = true
				amounts[i] += coin.CoinDetails.GetValue()
				break
			}
		}
		if !added {
			pubkeys = append(pubkeys, coinPubKey)
			amounts = append(amounts, coin.CoinDetails.GetValue())
		}
	}
	return pubkeys, amounts
}

// GetTokenUniqueReceiver
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTokenUniqueReceiver() (bool, []byte, uint64) {
	sender := []byte{}
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil {
		return false, []byte{}, 0
	}
	if len(proof.GetInputCoins()) > 0 && proof.GetInputCoins()[0].CoinDetails != nil {
		sender = proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	}
	pubkeys, amounts := txCustomTokenPrivacy.GetTokenReceivers()
	pubkey := []byte{}
	amount := uint64(0)
	count := 0
	for i, pk := range pubkeys {
		if !bytes.Equal(pk, sender) {
			pubkey = pk
			amount = amounts[i]
			count += 1
		}
	}
	return count == 1, pubkey, amount
}

// GetTransferData
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTransferData() (bool, []byte, uint64, *common.Hash) {
	unique, pk, amount := txCustomTokenPrivacy.GetTokenUniqueReceiver()
	return unique, pk, amount, &txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) CalculateBurningTxValue(bcr metadata.ChainRetriever, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, uint64) {
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil || len(proof.GetOutputCoins()) == 0 {
		return false, 0
	}
	//get burning address
	burningAddress := bcr.GetBurningAddress(beaconHeight)
	keyWalletBurningAccount, err := wallet.Base58CheckDeserialize(burningAddress)
	if err != nil {
		return false, 0
	}
	keysetBurningAccount := keyWalletBurningAccount.KeySet
	paymentAddressBurningAccount := keysetBurningAccount.PaymentAddress

	// check burning amount
	totalBurningAmount := uint64(0)
	for _, outCoin := range proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if bytes.Equal(outPKBytes, paymentAddressBurningAccount.Pk[:]) {
			totalBurningAmount += outCoin.CoinDetails.GetValue()
		}
	}
	if totalBurningAmount > 0 {
		return true, totalBurningAmount
	}
	return false, 0
}

// IsCoinsBurning - checking this is a burning pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) IsCoinsBurning(bcr metadata.ChainRetriever, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) bool {
	// get proof of pToken
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil || len(proof.GetOutputCoins()) == 0 {
		return false
	}
	//  validate receiver with burning address
	senderPKBytes := []byte{}
	if len(proof.GetInputCoins()) > 0 {
		coinDetails := proof.GetInputCoins()[0].CoinDetails
		if coinDetails == nil {
			Logger.log.Errorf("coin details is nil\n")
			return false
		}
		publicKey := coinDetails.GetPublicKey()
		if publicKey == nil {
			Logger.log.Errorf("public key is nil\n")
			return false
		}
		senderPKBytes = publicKey.ToBytesS()
	}

	//get burning address
	burningAddress := bcr.GetBurningAddress(beaconHeight)
	keyWalletBurningAccount, err := wallet.Base58CheckDeserialize(burningAddress)
	if err != nil {
		Logger.log.Errorf("Can not deserialize burn address: %v\n", burningAddress)
		return false
	}

	keysetBurningAccount := keyWalletBurningAccount.KeySet
	paymentAddressBurningAccount := keysetBurningAccount.PaymentAddress
	for _, outCoin := range proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if !bytes.Equal(senderPKBytes, outPKBytes) && !bytes.Equal(outPKBytes, paymentAddressBurningAccount.Pk[:]) {
			return false
		}
	}
	return true
}

// CalculateTxValue - get tx value for pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) CalculateTxValue() uint64 {
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil {
		return 0
	}
	if proof.GetOutputCoins() == nil || len(proof.GetOutputCoins()) == 0 {
		return 0
	}
	if proof.GetInputCoins() == nil || len(proof.GetInputCoins()) == 0 { // coinbase tx
		txValue := uint64(0)
		for _, outCoin := range proof.GetOutputCoins() {
			txValue += outCoin.CoinDetails.GetValue()
		}
		return txValue
	}

	if txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.IsPrivacy() {
		return 0
	}

	senderPKBytes := proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	txValue := uint64(0)
	for _, outCoin := range proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if bytes.Equal(senderPKBytes, outPKBytes) {
			continue
		}
		txValue += outCoin.CoinDetails.GetValue()
	}
	return txValue
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) ListSerialNumbersHashH() []common.Hash {
	tx := txCustomTokenPrivacy.Tx
	result := []common.Hash{}
	if tx.Proof != nil {
		for _, d := range tx.Proof.GetInputCoins() {
			hash := common.HashH(d.CoinDetails.GetSerialNumber().ToBytesS())
			result = append(result, hash)
		}
	}
	customTokenPrivacy := txCustomTokenPrivacy.TxPrivacyTokenData
	if customTokenPrivacy.TxNormal.Proof != nil {
		for _, d := range customTokenPrivacy.TxNormal.Proof.GetInputCoins() {
			hash := common.HashH(d.CoinDetails.GetSerialNumber().ToBytesS())
			result = append(result, hash)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) ListSNDOutputsHashH() []common.Hash {
	// tx normal
	tx := txCustomTokenPrivacy.Tx
	result := []common.Hash{}
	if tx.Proof != nil {
		for _, outputCoin := range tx.Proof.GetOutputCoins() {
			hash := common.HashH(outputCoin.CoinDetails.GetSNDerivator().ToBytesS())
			result = append(result, hash)
		}
	}
	// tx ptoken data
	customTokenPrivacy := txCustomTokenPrivacy.TxPrivacyTokenData
	if customTokenPrivacy.TxNormal.Proof != nil {
		for _, outputCoin := range customTokenPrivacy.TxNormal.Proof.GetOutputCoins() {
			hash := common.HashH(outputCoin.CoinDetails.GetSNDerivator().ToBytesS())
			result = append(result, hash)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

// GetSigPubKey - return sig pubkey for pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetSigPubKey() []byte {
	return txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.SigPubKey
}

// GetTxFeeToken - return Token Fee use to pay for privacy token Tx
func (tx TxCustomTokenPrivacy) GetTxFeeToken() uint64 {
	return tx.TxPrivacyTokenData.TxNormal.Fee
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTokenID() *common.Hash {
	return &txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID
}

// GetTxFee - return fee PRV of Tx which contain privacy token Tx
func (tx TxCustomTokenPrivacy) GetTxFee() uint64 {
	return tx.Tx.GetTxFee()
}

type TxPrivacyTokenInitParamsForASM struct {
	//senderKey       *privacy.PrivateKey
	//paymentInfo     []*privacy.PaymentInfo
	//inputCoin       []*privacy.InputCoin
	//feeNativeCoin   uint64
	//tokenParams     *CustomTokenPrivacyParamTx
	//transactionStateDB              database.DatabaseInterface
	//metaData        metadata.Metadata
	//hasPrivacyCoin  bool
	//hasPrivacyToken bool
	//shardID         byte
	//info            []byte

	txParam                           TxPrivacyTokenInitParams
	commitmentIndicesForNativeToken   []uint64
	commitmentBytesForNativeToken     [][]byte
	myCommitmentIndicesForNativeToken []uint64
	sndOutputsForNativeToken          []*privacy.Scalar

	commitmentIndicesForPToken   []uint64
	commitmentBytesForPToken     [][]byte
	myCommitmentIndicesForPToken []uint64
	sndOutputsForPToken          []*privacy.Scalar
}

func (param *TxPrivacyTokenInitParamsForASM) SetMetaData(meta metadata.Metadata) {
	param.txParam.metaData = meta
}

func NewTxPrivacyTokenInitParamsForASM(
	senderKey *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoin []*privacy.InputCoin,
	feeNativeCoin uint64,
	tokenParams *CustomTokenPrivacyParamTx,
	metaData metadata.Metadata,
	hasPrivacyCoin bool,
	hasPrivacyToken bool,
	shardID byte,
	info []byte,
	commitmentIndicesForNativeToken []uint64,
	commitmentBytesForNativeToken [][]byte,
	myCommitmentIndicesForNativeToken []uint64,
	sndOutputsForNativeToken []*privacy.Scalar,

	commitmentIndicesForPToken []uint64,
	commitmentBytesForPToken [][]byte,
	myCommitmentIndicesForPToken []uint64,
	sndOutputsForPToken []*privacy.Scalar) *TxPrivacyTokenInitParamsForASM {

	txParam := NewTxPrivacyTokenInitParams(senderKey, paymentInfo, inputCoin, feeNativeCoin, tokenParams, nil, metaData, hasPrivacyCoin, hasPrivacyToken, shardID, info, nil)
	params := &TxPrivacyTokenInitParamsForASM{
		txParam:                           *txParam,
		commitmentIndicesForNativeToken:   commitmentIndicesForNativeToken,
		commitmentBytesForNativeToken:     commitmentBytesForNativeToken,
		myCommitmentIndicesForNativeToken: myCommitmentIndicesForNativeToken,
		sndOutputsForNativeToken:          sndOutputsForNativeToken,

		commitmentIndicesForPToken:   commitmentIndicesForPToken,
		commitmentBytesForPToken:     commitmentBytesForPToken,
		myCommitmentIndicesForPToken: myCommitmentIndicesForPToken,
		sndOutputsForPToken:          sndOutputsForPToken,
	}
	return params
}

// Init -  build normal tx component and privacy custom token data
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) InitForASM(params *TxPrivacyTokenInitParamsForASM, serverTime int64) error {
	var err error
	// init data for tx PRV for fee
	normalTx := Tx{}
	err = normalTx.InitForASM(NewTxPrivacyInitParamsForASM(
		params.txParam.senderKey,
		params.txParam.paymentInfo,
		params.txParam.inputCoin,
		params.txParam.feeNativeCoin,
		params.txParam.hasPrivacyCoin,
		nil,
		params.txParam.metaData,
		params.txParam.info,
		params.commitmentIndicesForNativeToken,
		params.commitmentBytesForNativeToken,
		params.myCommitmentIndicesForNativeToken,
		params.sndOutputsForNativeToken,
	), serverTime)
	if err != nil {
		return NewTransactionErr(PrivacyTokenInitPRVError, err)
	}

	// override TxCustomTokenPrivacyType type
	normalTx.Type = common.TxCustomTokenPrivacyType
	txCustomTokenPrivacy.Tx = normalTx

	// check action type and create privacy custom toke data
	var handled = false
	// Add token data component
	switch params.txParam.tokenParams.TokenTxType {
	case CustomTokenInit:
		// case init a new privacy custom token
		{
			handled = true
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.txParam.tokenParams.TokenTxType,
				PropertyName:   params.txParam.tokenParams.PropertyName,
				PropertySymbol: params.txParam.tokenParams.PropertySymbol,
				Amount:         params.txParam.tokenParams.Amount,
			}

			// issue token with data of privacy
			temp := Tx{}
			temp.Type = common.TxNormalType
			temp.Proof = new(zkp.PaymentProof)
			tempOutputCoin := make([]*privacy.OutputCoin, 1)
			tempOutputCoin[0] = new(privacy.OutputCoin)
			tempOutputCoin[0].CoinDetails = new(privacy.Coin)
			tempOutputCoin[0].CoinDetails.SetValue(params.txParam.tokenParams.Amount)
			PK, err := new(privacy.Point).FromBytesS(params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk)
			if err != nil {
				return NewTransactionErr(DecompressPaymentAddressError, err)
			}
			tempOutputCoin[0].CoinDetails.SetPublicKey(PK)
			tempOutputCoin[0].CoinDetails.SetRandomness(privacy.RandomScalar())

			// set info coin for output coin
			if len(params.txParam.tokenParams.Receiver[0].Message) > 0 {
				if len(params.txParam.tokenParams.Receiver[0].Message) > privacy.MaxSizeInfoCoin {
					return NewTransactionErr(ExceedSizeInfoOutCoinError, nil)
				}
				tempOutputCoin[0].CoinDetails.SetInfo(params.txParam.tokenParams.Receiver[0].Message)
			}

			sndOut := privacy.RandomScalar()
			tempOutputCoin[0].CoinDetails.SetSNDerivator(sndOut)
			temp.Proof.SetOutputCoins(tempOutputCoin)

			// create coin commitment
			err = temp.Proof.GetOutputCoins()[0].CoinDetails.CommitAll()
			if err != nil {
				return NewTransactionErr(CommitOutputCoinError, err)
			}
			// get last byte
			lastByteSender := params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk[len(params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk)-1]
			temp.PubKeyLastByteSender = common.GetShardIDFromLastByte(lastByteSender)

			// sign Tx
			temp.SigPubKey = params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk
			temp.sigPrivKey = *params.txParam.senderKey
			err = temp.signTx()
			if err != nil {
				Logger.log.Error(errors.New("can't sign this tx"))
				return NewTransactionErr(SignTxError, err)
			}

			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
			hashInitToken, err := txCustomTokenPrivacy.TxPrivacyTokenData.Hash()
			if err != nil {
				Logger.log.Error(errors.New("can't hash this token data"))
				return NewTransactionErr(UnexpectedError, err)
			}

			if params.txParam.tokenParams.Mintable {
				propertyID, err := common.Hash{}.NewHashFromStr(params.txParam.tokenParams.PropertyID)
				if err != nil {
					return NewTransactionErr(TokenIDInvalidError, err, propertyID.String())
				}
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = *propertyID
				txCustomTokenPrivacy.TxPrivacyTokenData.Mintable = true
			} else {
				//NOTICE: @merman update PropertyID calculated from hash of tokendata and shardID
				newHashInitToken := common.HashH(append(hashInitToken.GetBytes(), params.txParam.shardID))
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = newHashInitToken
			}
		}
	case CustomTokenTransfer:
		{
			handled = true
			// make a transfering for privacy custom token
			// fee always 0 and reuse function of normal tx for custom token ID
			temp := Tx{}
			propertyID, _ := common.Hash{}.NewHashFromStr(params.txParam.tokenParams.PropertyID)
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.txParam.tokenParams.TokenTxType,
				PropertyName:   params.txParam.tokenParams.PropertyName,
				PropertySymbol: params.txParam.tokenParams.PropertySymbol,
				PropertyID:     *propertyID,
				Mintable:       params.txParam.tokenParams.Mintable,
			}
			err := temp.InitForASM(NewTxPrivacyInitParamsForASM(
				params.txParam.senderKey,
				params.txParam.tokenParams.Receiver,
				params.txParam.tokenParams.TokenInput,
				params.txParam.tokenParams.Fee,
				params.txParam.hasPrivacyToken,
				propertyID,
				nil,
				params.txParam.info,
				params.commitmentIndicesForPToken,
				params.commitmentBytesForPToken,
				params.myCommitmentIndicesForPToken,
				params.sndOutputsForPToken,
			), serverTime)
			if err != nil {
				return NewTransactionErr(PrivacyTokenInitTokenDataError, err)
			}
			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
		}
	}

	if !handled {
		return NewTransactionErr(PrivacyTokenTxTypeNotHandleError, errors.New("can't handle this TokenTxType"))
	}
	return nil
}

// GetFullTxValues returns both prv and ptoken values
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetFullTxValues() (uint64, uint64) {
	return txCustomTokenPrivacy.Tx.CalculateTxValue(), txCustomTokenPrivacy.CalculateTxValue()
}

// IsFullBurning returns whether the tx is full burning tx
func (txCustomTokenPrivacy TxCustomTokenPrivacy) IsFullBurning(
	bcr metadata.ChainRetriever,
	retriever metadata.ShardViewRetriever,
	viewRetriever metadata.BeaconViewRetriever,
	beaconHeight uint64,
) bool {
	return txCustomTokenPrivacy.Tx.IsCoinsBurning(bcr, retriever, viewRetriever, beaconHeight) && txCustomTokenPrivacy.IsCoinsBurning(bcr, retriever, viewRetriever, beaconHeight)
}
