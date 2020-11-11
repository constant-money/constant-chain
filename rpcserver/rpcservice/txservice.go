package rpcservice

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/mempool"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/incognitochain/incognito-chain/wire"
)

type TxService struct {
	BlockChain   *blockchain.BlockChain
	Wallet       *wallet.Wallet
	FeeEstimator map[byte]*mempool.FeeEstimator
	TxMemPool    *mempool.TxPool
}

type TxInfo struct {
	BlockHash common.Hash
	Tx metadata.Transaction
}

func (txService TxService) ListSerialNumbers(tokenID common.Hash, shardID byte) (map[string]struct{}, error) {
	transactionStateDB := txService.BlockChain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()
	return statedb.ListSerialNumber(transactionStateDB, tokenID, shardID)
}

func (txService TxService) ListSNDerivator(tokenID common.Hash, shardID byte) ([]big.Int, error) {
	transactionStateDB := txService.BlockChain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()
	resultInBytes, err := statedb.ListSNDerivator(transactionStateDB, tokenID)
	if err != nil {
		return nil, err
	}
	result := []big.Int{}
	for _, v := range resultInBytes {
		result = append(result, *(new(big.Int).SetBytes(v)))
	}
	return result, nil
}

func (txService TxService) ListCommitments(tokenID common.Hash, shardID byte) (map[string]uint64, error) {
	transactionStateDB := txService.BlockChain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()
	return statedb.ListCommitment(transactionStateDB, tokenID, shardID)
}

func (txService TxService) ListCommitmentIndices(tokenID common.Hash, shardID byte) (map[uint64]string, error) {
	transactionStateDB := txService.BlockChain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()
	return statedb.ListCommitmentIndices(transactionStateDB, tokenID, shardID)
}

func (txService TxService) HasSerialNumbers(paymentAddressStr string, serialNumbersStr []interface{}, tokenID common.Hash) ([]bool, error) {
	_, shardIDSender, err := GetKeySetFromPaymentAddressParam(paymentAddressStr)
	if err != nil {
		return nil, err
	}
	result := make([]bool, 0)
	for _, item := range serialNumbersStr {
		itemStr, okParam := item.(string)
		if !okParam {
			return nil, fmt.Errorf("Invalid serial number param, %+v", item)
		}
		serialNumber, _, err := base58.Base58Check{}.Decode(itemStr)
		if err != nil {
			return nil, fmt.Errorf("Decode serial number failed, %+v", itemStr)
		}
		transactionStateDB := txService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB()
		ok, err := statedb.HasSerialNumber(transactionStateDB, tokenID, serialNumber, shardIDSender)
		if ok && err == nil {
			// serial number in db
			result = append(result, true)
		} else {
			// serial number not in db
			result = append(result, false)
		}
	}

	return result, nil
}

func (txService TxService) HasSnDerivators(paymentAddressStr string, snDerivatorStr []interface{}, tokenID common.Hash) ([]bool, error) {
	_, shardIDSender, err := GetKeySetFromPaymentAddressParam(paymentAddressStr)
	if err != nil {
		return nil, err
	}
	result := make([]bool, 0)
	for _, item := range snDerivatorStr {
		itemStr, okParam := item.(string)
		if !okParam {
			return nil, errors.New("Invalid serial number derivator param")
		}
		snderivator, _, err := base58.Base58Check{}.Decode(itemStr)
		if err != nil {
			return nil, errors.New("Invalid serial number derivator param")
		}
		transactionStateDB := txService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB()
		ok, err := statedb.HasSNDerivator(transactionStateDB, tokenID, common.AddPaddingBigInt(new(big.Int).SetBytes(snderivator), common.BigIntSize))
		if ok && err == nil {
			// SnD in db
			result = append(result, true)
		} else {
			// SnD not in db
			result = append(result, false)
		}
	}
	return result, nil
}

// chooseBestOutCoinsToSpent returns list of unspent coins for spending with amount
func (txService TxService) chooseBestOutCoinsToSpent(outCoins []*privacy.OutputCoin, amount uint64) (resultOutputCoins []*privacy.OutputCoin, remainOutputCoins []*privacy.OutputCoin, totalResultOutputCoinAmount uint64, err error) {
	resultOutputCoins = make([]*privacy.OutputCoin, 0)
	remainOutputCoins = make([]*privacy.OutputCoin, 0)
	totalResultOutputCoinAmount = uint64(0)

	// either take the smallest coins, or a single largest one
	var outCoinOverLimit *privacy.OutputCoin
	outCoinsUnderLimit := make([]*privacy.OutputCoin, 0)
	for _, outCoin := range outCoins {
		if outCoin.CoinDetails.GetValue() < amount {
			outCoinsUnderLimit = append(outCoinsUnderLimit, outCoin)
		} else if outCoinOverLimit == nil {
			outCoinOverLimit = outCoin
		} else if outCoinOverLimit.CoinDetails.GetValue() > outCoin.CoinDetails.GetValue() {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		} else {
			remainOutputCoins = append(remainOutputCoins, outCoinOverLimit)
			outCoinOverLimit = outCoin
		}
	}
	sort.Slice(outCoinsUnderLimit, func(i, j int) bool {
		return outCoinsUnderLimit[i].CoinDetails.GetValue() < outCoinsUnderLimit[j].CoinDetails.GetValue()
	})
	for _, outCoin := range outCoinsUnderLimit {
		if totalResultOutputCoinAmount < amount {
			totalResultOutputCoinAmount += outCoin.CoinDetails.GetValue()
			resultOutputCoins = append(resultOutputCoins, outCoin)
		} else {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	if outCoinOverLimit != nil && (outCoinOverLimit.CoinDetails.GetValue() > 2*amount || totalResultOutputCoinAmount < amount) {
		remainOutputCoins = append(remainOutputCoins, resultOutputCoins...)
		resultOutputCoins = []*privacy.OutputCoin{outCoinOverLimit}
		totalResultOutputCoinAmount = outCoinOverLimit.CoinDetails.GetValue()
	} else if outCoinOverLimit != nil {
		remainOutputCoins = append(remainOutputCoins, outCoinOverLimit)
	}
	if totalResultOutputCoinAmount < amount {
		return resultOutputCoins, remainOutputCoins, totalResultOutputCoinAmount, errors.New("Not enough coin")
	} else {
		return resultOutputCoins, remainOutputCoins, totalResultOutputCoinAmount, nil
	}
}

func (txService TxService) filterMemPoolOutcoinsToSpent(outCoins []*privacy.OutputCoin) ([]*privacy.OutputCoin, error) {
	remainOutputCoins := make([]*privacy.OutputCoin, 0)
	for _, outCoin := range outCoins {
		if txService.TxMemPool.ValidateSerialNumberHashH(outCoin.CoinDetails.GetSerialNumber().ToBytesS()) == nil {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	return remainOutputCoins, nil
}

// chooseOutsCoinByKeyset returns list of input coins native token to spent
func (txService TxService) chooseOutsCoinByKeyset(
	paymentInfos []*privacy.PaymentInfo,
	unitFeeNativeToken int64, numBlock uint64, keySet *incognitokey.KeySet, shardIDSender byte,
	hasPrivacy bool,
	metadataParam metadata.Metadata,
	privacyCustomTokenParams *transaction.CustomTokenPrivacyParamTx,
	isGetFeePToken bool,
	unitFeePToken int64,
) ([]*privacy.InputCoin, uint64, *RPCError) {
	// estimate fee according to 8 recent block
	if numBlock == 0 {
		numBlock = 1000
	}
	// calculate total amount to send
	totalAmount := uint64(0)
	for _, receiver := range paymentInfos {
		totalAmount += receiver.Amount
	}

	// get output native coins for estimate fee
	candidateOutputCoins, remainOutCoins, candidateOutputCoinAmount, overBalanceAmount, err1 := txService.ChooseOutsCoinByKeysetForEstimateFee(
		paymentInfos, 0, keySet, shardIDSender)
	if err1 != nil {
		return nil, 0, NewRPCError(RejectInvalidTxFeeError, err1)
	}

	// refund out put for sender
	if overBalanceAmount > 0 {
		// add more into output for estimate fee
		paymentInfos = append(paymentInfos, &privacy.PaymentInfo{
			PaymentAddress: keySet.PaymentAddress,
			Amount:         overBalanceAmount,
		})
	}

	// check real fee(nano PRV) per tx
	beaconHeight := txService.BlockChain.GetBeaconBestState().BestBlock.GetHeight()
	realFee, _, _, err := txService.EstimateFee(unitFeeNativeToken, false, candidateOutputCoins,
		paymentInfos, shardIDSender, numBlock, hasPrivacy,
		metadataParam,
		privacyCustomTokenParams, int64(beaconHeight))
	if err != nil {
		return nil, 0, NewRPCError(RejectInvalidTxFeeError, err)
	}
	if totalAmount == 0 && realFee == 0 {
		if metadataParam != nil {
			metadataType := metadataParam.GetType()
			switch metadataType {
			case metadata.WithDrawRewardRequestMeta:
				{
					return nil, realFee, nil
				}
			}
			return nil, realFee, NewRPCError(RejectInvalidTxFeeError, fmt.Errorf("totalAmount: %+v, realFee: %+v", totalAmount, realFee))
		}
		if privacyCustomTokenParams != nil {
			// for privacy token
			return nil, 0, nil
		}
	}
	needToPayFee := int64((totalAmount + realFee) - candidateOutputCoinAmount)
	// if not enough to pay fee
	if totalAmount + realFee > candidateOutputCoinAmount {
		if len(remainOutCoins) > 0 {
			candidateOutputCoinsForFee, _, _, err1 := txService.chooseBestOutCoinsToSpent(remainOutCoins, uint64(needToPayFee))
			if err != nil {
				return nil, 0, NewRPCError(GetOutputCoinError, err1)
			}
			candidateOutputCoins = append(candidateOutputCoins, candidateOutputCoinsForFee...)
		} else {
			return nil, 0, NewRPCError(RejectInvalidTxFeeError, errors.New("not enough coin for native fee"))
		}
	}
	// convert to input coins
	inputCoins := transaction.ConvertOutputCoinToInputCoin(candidateOutputCoins)
	return inputCoins, realFee, nil
}

// chooseOutsCoinByKeyset returns list of input coins native token to estimate fee
func (txService TxService) ChooseOutsCoinByKeysetForEstimateFee(
	paymentInfos []*privacy.PaymentInfo, numBlock uint64,
	keySet *incognitokey.KeySet, shardIDSender byte,
) ([]*privacy.OutputCoin, []*privacy.OutputCoin, uint64, uint64, *RPCError) {
	// estimate fee according to 8 recent block
	if numBlock == 0 {
		numBlock = 1000
	}
	// calculate total amount to send
	totalAmount := uint64(0)
	for _, receiver := range paymentInfos {
		totalAmount += receiver.Amount
	}
	// get list outputcoins tx
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	outCoins, err := txService.BlockChain.GetListOutputCoinsByKeyset(keySet, shardIDSender, prvCoinID)
	if err != nil {
		return nil, nil, 0, 0, NewRPCError(GetOutputCoinError, err)
	}
	// remove out coin in mem pool
	outCoins, err = txService.filterMemPoolOutcoinsToSpent(outCoins)
	if err != nil {
		return nil, nil, 0, 0, NewRPCError(GetOutputCoinError, err)
	}
	if len(outCoins) == 0 && totalAmount > 0 {
		return nil, nil, 0, 0, NewRPCError(GetOutputCoinError, errors.New("not enough output coin"))
	}
	// Use Knapsack to get candiate output coin
	candidateOutputCoins, remainOutCoins, totalCandidateOutCoinsAmount, err := txService.chooseBestOutCoinsToSpent(outCoins, totalAmount)
	if err != nil {
		return nil, nil, 0, 0, NewRPCError(GetOutputCoinError, err)
	}

	overBalance := totalCandidateOutCoinsAmount - totalAmount
	return candidateOutputCoins, remainOutCoins, totalCandidateOutCoinsAmount, overBalance, nil
}

// EstimateFee - estimate fee from tx data and return real full fee, fee per kb and real tx size
// if isGetPTokenFee == true: return fee for ptoken
// if isGetPTokenFee == false: return fee for native token
func (txService TxService) EstimateFee(
	defaultFee int64,
	isGetPTokenFee bool,
	candidateOutputCoins []*privacy.OutputCoin,
	paymentInfos []*privacy.PaymentInfo, shardID byte,
	numBlock uint64, hasPrivacy bool,
	metadata metadata.Metadata,
	privacyCustomTokenParams *transaction.CustomTokenPrivacyParamTx,
	beaconHeight int64) (uint64, uint64, uint64, error) {
	if numBlock == 0 {
		numBlock = 1000
	}
	// check real fee(nano PRV) per tx
	var realFee uint64
	estimateFeeCoinPerKb := uint64(0)
	estimateTxSizeInKb := uint64(0)
	tokenId := &common.Hash{}
	if isGetPTokenFee {
		if privacyCustomTokenParams != nil {
			tokenId, _ = common.Hash{}.NewHashFromStr(privacyCustomTokenParams.PropertyID)
		}
	} else {
		tokenId = nil
	}
	estimateFeeCoinPerKb, err := txService.EstimateFeeWithEstimator(defaultFee, shardID, numBlock, tokenId, beaconHeight)
	if err != nil {
		return 0, 0, 0, err
	}
	if txService.Wallet != nil {
		estimateFeeCoinPerKb += uint64(txService.Wallet.GetConfig().IncrementalFee)
	}
	limitFee := uint64(0)
	if feeEstimator, ok := txService.FeeEstimator[shardID]; ok {
		limitFee = feeEstimator.GetLimitFeeForNativeToken()
	}
	estimateTxSizeInKb = transaction.EstimateTxSize(transaction.NewEstimateTxSizeParam(len(candidateOutputCoins), len(paymentInfos), hasPrivacy, metadata, privacyCustomTokenParams, limitFee))
	realFee = uint64(estimateFeeCoinPerKb) * uint64(estimateTxSizeInKb)
	return realFee, estimateFeeCoinPerKb, estimateTxSizeInKb, nil
}

// EstimateFeeWithEstimator - only estimate fee by estimator and return fee per kb
// if tokenID != nil: return fee per kb for pToken (return error if there is no exchange rate between pToken and native token)
// if tokenID == nil: return fee per kb for native token
func (txService TxService) EstimateFeeWithEstimator(defaultFee int64, shardID byte, numBlock uint64, tokenId *common.Hash, beaconHeight int64) (uint64, error) {
	if defaultFee == 0 {
		return uint64(defaultFee), nil
	}
	unitFee := uint64(0)
	if defaultFee == -1 {
		// estimate fee on the blocks before (in native token or in pToken)
		if _, ok := txService.FeeEstimator[shardID]; ok {
			temp, _ := txService.FeeEstimator[shardID].EstimateFee(numBlock, tokenId)
			unitFee = uint64(temp)
		}
	} else {
		// get default fee (in native token or in ptoken)
		unitFee = uint64(defaultFee)
	}
	// get limit fee for native token
	limitFee := uint64(0)
	if feeEstimator, ok := txService.FeeEstimator[shardID]; ok {
		limitFee = feeEstimator.GetLimitFeeForNativeToken()
	}
	if tokenId == nil {
		// check with limit fee
		if unitFee < limitFee {
			unitFee = limitFee
		}
		return unitFee, nil
	} else {
		// convert limit fee native token to limit fee ptoken
		limitFeePTokenTmp, err := metadata.ConvertNativeTokenToPrivacyToken(limitFee, tokenId, beaconHeight, txService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB())
		limitFeePToken := uint64(math.Ceil(limitFeePTokenTmp))
		if err != nil {
			return uint64(0), err
		}
		// check with limit fee ptoken
		if unitFee < limitFeePToken {
			unitFee = limitFeePToken
		}
		// add extra fee to make sure tx is confirmed
		// extra fee = unitFee * 10%
		unitFee += uint64(math.Ceil(float64(unitFee) * float64(0.1)))
		return unitFee, nil
	}
}

func (txService TxService) BuildRawTransaction(params *bean.CreateRawTxParam, meta metadata.Metadata) (*transaction.Tx, *RPCError) {
	// get output coins to spend and real fee
	inputCoins, realFee, err1 := txService.chooseOutsCoinByKeyset(
		params.PaymentInfos, params.EstimateFeeCoinPerKb, 0,
		params.SenderKeySet, params.ShardIDSender, params.HasPrivacyCoin,
		meta, nil, false, int64(0))
	if err1 != nil {
		return nil, err1
	}
	// init tx
	tx := transaction.Tx{}
	err := tx.Init(
		transaction.NewTxPrivacyInitParams(
			&params.SenderKeySet.PrivateKey,
			params.PaymentInfos,
			inputCoins,
			realFee,
			params.HasPrivacyCoin,
			txService.BlockChain.GetBestStateShard(params.ShardIDSender).GetCopiedTransactionStateDB(),
			nil, // use for prv coin -> nil is valid
			meta,
			params.Info,
		))
	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}
	return &tx, nil
}

func (txService TxService) CreateRawTransaction(params *bean.CreateRawTxParam, meta metadata.Metadata) (*common.Hash, []byte, byte, *RPCError) {
	var err error
	tx, err := txService.BuildRawTransaction(params, meta)
	if err.(*RPCError) != nil {
		Logger.log.Critical(err)
		return nil, nil, byte(0), NewRPCError(CreateTxDataError, err)
	}
	txBytes, err := json.Marshal(tx)
	if err != nil {
		// return hex for a new tx
		return nil, nil, byte(0), NewRPCError(CreateTxDataError, err)
	}
	txShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	return tx.Hash(), txBytes, txShardID, nil
}

func (txService TxService) SendRawTransaction(txB58Check string) (wire.Message, *common.Hash, byte, *RPCError) {
	// Decode base58check data of tx
	rawTxBytes, _, err := base58.Base58Check{}.Decode(txB58Check)
	if err != nil {
		Logger.log.Errorf("Send Raw Transaction Error: %+v", err)
		return nil, nil, byte(0), NewRPCError(Base58ChedkDataOfTxInvalid, err)
	}
	// Unmarshal from json data to object tx
	var tx transaction.Tx
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		Logger.log.Errorf("Send Raw Transaction Error: %+v", err)
		return nil, nil, byte(0), NewRPCError(JsonDataOfTxInvalid, err)
	}

	beaconHeigh := int64(-1)
	beaconBestState, err := txService.BlockChain.GetClonedBeaconBestState()
	if err == nil {
		beaconHeigh = int64(beaconBestState.BeaconHeight)
	} else {
		Logger.log.Errorf("Send Raw Transaction can not get beacon best state with error %+v", err)
	}
	// Try add tx in to mempool of node
	hash, _, err := txService.TxMemPool.MaybeAcceptTransaction(&tx, beaconHeigh)
	if err != nil {
		Logger.log.Errorf("Send Raw Transaction Error, try add tx into mempool of node: %+v", err)
		mempoolErr, ok := err.(*mempool.MempoolTxError)
		if ok {
			switch mempoolErr.Code {
			case mempool.ErrCodeMessage[mempool.RejectInvalidFee].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxFeeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidSize].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxSizeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidTxType].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxTypeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidTx].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectReplacementTxError].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectReplacementTx, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectDoubleSpendWithBlockchainTx].Code, mempool.ErrCodeMessage[mempool.RejectDoubleSpendWithMempoolTx].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectDoubleSpendTxError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectDuplicateTx].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectDuplicateTxInPoolError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectVersion].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectDuplicateTxInPoolError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectSanityTxLocktime].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectSanityTxLocktime, mempoolErr)
				}
			}
		}
		return nil, nil, byte(0), NewRPCError(TxPoolRejectTxError, err)
	}
	Logger.log.Debugf("New transaction hash: %+v \n", *hash)
	// Create tx message for broadcasting
	txMsg, err := wire.MakeEmptyMessage(wire.CmdTx)
	if err != nil {
		Logger.log.Errorf("Send Raw Transaction Error, Create tx message for broadcasting: %+v", err)
		return nil, nil, byte(0), NewRPCError(SendTxDataError, err)
	}
	txMsg.(*wire.MessageTx).Transaction = &tx
	return txMsg, hash, tx.PubKeyLastByteSender, nil
}

func (txService TxService) BuildTokenParam(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.CustomTokenPrivacyParamTx, *RPCError) {
	var privacyTokenParam *transaction.CustomTokenPrivacyParamTx
	var err *RPCError
	isPrivacy, ok := tokenParamsRaw["Privacy"].(bool)
	if !ok {
		return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Params %+v", tokenParamsRaw))
	}
	if !isPrivacy {
		// Check normal custom token param
	} else {
		// Check privacy custom token param
		privacyTokenParam, _, _, err = txService.BuildPrivacyCustomTokenParam(tokenParamsRaw, senderKeySet, shardIDSender)
		if err != nil {
			return nil, NewRPCError(BuildTokenParamError, err)
		}
	}
	return privacyTokenParam, nil
}

func (txService TxService) BuildPrivacyCustomTokenParam(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.CustomTokenPrivacyParamTx, map[common.Hash]transaction.TxCustomTokenPrivacy, map[common.Hash]types.CrossShardTokenPrivacyMetaData, *RPCError) {
	property, ok := tokenParamsRaw["TokenID"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token ID, Params %+v ", tokenParamsRaw))
	}
	_, ok = tokenParamsRaw["TokenReceivers"]
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Token Receiver is invalid"))
	}
	tokenName, ok := tokenParamsRaw["TokenName"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Name, Params %+v ", tokenParamsRaw))
	}
	tokenSymbol, ok := tokenParamsRaw["TokenSymbol"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Symbol, Params %+v ", tokenParamsRaw))
	}
	tokenTxType, ok := tokenParamsRaw["TokenTxType"].(float64)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Tx Type, Params %+v ", tokenParamsRaw))
	}
	tokenAmount, ok := tokenParamsRaw["TokenAmount"].(float64)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Amout, Params %+v ", tokenParamsRaw))
	}
	tokenFee, ok := tokenParamsRaw["TokenFee"].(float64)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Fee, Params %+v ", tokenParamsRaw))
	}
	if tokenTxType == transaction.CustomTokenInit {
		tokenFee = 0
	}
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID:     property,
		PropertyName:   tokenName,
		PropertySymbol: tokenSymbol,
		TokenTxType:    int(tokenTxType),
		Amount:         uint64(tokenAmount),
		TokenInput:     nil,
		Fee:            uint64(tokenFee),
	}
	voutsAmount := int64(0)
	var err1 error

	tokenParams.Receiver, voutsAmount, err1 = transaction.CreateCustomTokenPrivacyReceiverArray(tokenParamsRaw["TokenReceivers"])
	if err1 != nil {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, err1)
	}
	voutsAmount += int64(tokenFee)
	// get list custom token
	switch tokenParams.TokenTxType {
	case transaction.CustomTokenTransfer:
		{
			tokenID, err := common.Hash{}.NewHashFromStr(tokenParams.PropertyID)
			if err != nil {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
			}
			isExisted := statedb.PrivacyTokenIDExisted(txService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB(), *tokenID)
			if !isExisted {
				var isBridgeToken bool
				_, allBridgeTokens, err := txService.BlockChain.GetAllBridgeTokens()
				if err != nil {
					return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
				}
				for _, bridgeToken := range allBridgeTokens {
					if bridgeToken.TokenID.IsEqual(tokenID) {
						isBridgeToken = true
						break
					}
				}
				if !isBridgeToken {
					// totally invalid token
					return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
				}
				//return nil, nil, nil, NewRPCError(BuildPrivacyTokenParamError, err)
			}
			outputTokens, err := txService.BlockChain.GetListOutputCoinsByKeyset(senderKeySet, shardIDSender, tokenID)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			outputTokens, err = txService.filterMemPoolOutcoinsToSpent(outputTokens)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			candidateOutputTokens, _, _, err := txService.chooseBestOutCoinsToSpent(outputTokens, uint64(voutsAmount))
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			intputToken := transaction.ConvertOutputCoinToInputCoin(candidateOutputTokens)
			tokenParams.TokenInput = intputToken
		}
	case transaction.CustomTokenInit:
		{
			if len(tokenParams.Receiver) == 0 {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong receiver"))
			}
			if tokenParams.Receiver[0].Amount != tokenParams.Amount { // Init with wrong max amount of custom token
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong max amount of property"))
			}
			if tokenParams.PropertyName == "" {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong name of property"))
			}
			if tokenParams.PropertySymbol == "" {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong symbol of property"))
			}
		}
	}
	return tokenParams, nil, nil, nil
}

func (txService TxService) BuildTokenParamV2(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.CustomTokenPrivacyParamTx, *RPCError) {
	var privacyTokenParam *transaction.CustomTokenPrivacyParamTx
	var err *RPCError
	isPrivacy, ok := tokenParamsRaw["Privacy"].(bool)
	if !ok {
		return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Params %+v", tokenParamsRaw))
	}
	if !isPrivacy {
		// Check normal custom token param
	} else {
		// Check privacy custom token param
		privacyTokenParam, _, _, err = txService.BuildPrivacyCustomTokenParamV2(tokenParamsRaw, senderKeySet, shardIDSender)
		if err != nil {
			return nil, NewRPCError(BuildTokenParamError, err)
		}
	}
	return privacyTokenParam, nil
}

func (txService TxService) BuildPrivacyCustomTokenParamV2(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.CustomTokenPrivacyParamTx, map[common.Hash]transaction.TxCustomTokenPrivacy, map[common.Hash]types.CrossShardTokenPrivacyMetaData, *RPCError) {
	property, ok := tokenParamsRaw["TokenID"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token ID, Params %+v ", tokenParamsRaw))
	}
	_, ok = tokenParamsRaw["TokenReceivers"]
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Token Receiver is invalid"))
	}
	tokenName, ok := tokenParamsRaw["TokenName"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Name, Params %+v ", tokenParamsRaw))
	}
	tokenSymbol, ok := tokenParamsRaw["TokenSymbol"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Symbol, Params %+v ", tokenParamsRaw))
	}
	tokenTxType, ok := tokenParamsRaw["TokenTxType"].(float64)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Tx Type, Params %+v ", tokenParamsRaw))
	}

	tokenAmount, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["TokenAmount"])
	if err != nil {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Amout - error: %+v ", err))
	}

	tokenFee, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["TokenFee"])
	if err != nil {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Fee - error: %+v ", err))
	}

	if tokenTxType == transaction.CustomTokenInit {
		tokenFee = 0
	}
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID:     property,
		PropertyName:   tokenName,
		PropertySymbol: tokenSymbol,
		TokenTxType:    int(tokenTxType),
		Amount:         uint64(tokenAmount),
		TokenInput:     nil,
		Fee:            uint64(tokenFee),
	}
	voutsAmount := int64(0)
	var err1 error

	tokenParams.Receiver, voutsAmount, err1 = transaction.CreateCustomTokenPrivacyReceiverArrayV2(tokenParamsRaw["TokenReceivers"])
	if err1 != nil {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, err1)
	}
	voutsAmount += int64(tokenFee)
	// get list custom token
	switch tokenParams.TokenTxType {
	case transaction.CustomTokenTransfer:
		{
			tokenID, err := common.Hash{}.NewHashFromStr(tokenParams.PropertyID)
			if err != nil {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
			}
			isExisted := statedb.PrivacyTokenIDExisted(txService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB(), *tokenID)
			if !isExisted {
				var isBridgeToken bool
				_, allBridgeTokens, err := txService.BlockChain.GetAllBridgeTokens()
				if err != nil {
					return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
				}
				for _, bridgeToken := range allBridgeTokens {
					if bridgeToken.TokenID.IsEqual(tokenID) {
						isBridgeToken = true
						break
					}
				}
				if !isBridgeToken {
					// totally invalid token
					return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
				}
				//return nil, nil, nil, NewRPCError(BuildPrivacyTokenParamError, err)
			}
			outputTokens, err := txService.BlockChain.GetListOutputCoinsByKeyset(senderKeySet, shardIDSender, tokenID)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			outputTokens, err = txService.filterMemPoolOutcoinsToSpent(outputTokens)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			candidateOutputTokens, _, _, err := txService.chooseBestOutCoinsToSpent(outputTokens, uint64(voutsAmount))
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			intputToken := transaction.ConvertOutputCoinToInputCoin(candidateOutputTokens)
			tokenParams.TokenInput = intputToken
		}
	case transaction.CustomTokenInit:
		{
			if len(tokenParams.Receiver) == 0 {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong receiver"))
			}
			if tokenParams.Receiver[0].Amount != tokenParams.Amount { // Init with wrong max amount of custom token
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong max amount of property"))
			}
			if tokenParams.PropertyName == "" {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong name of property"))
			}
			if tokenParams.PropertySymbol == "" {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Init with wrong symbol of property"))
			}
		}
	}
	return tokenParams, nil, nil, nil
}

// BuildRawCustomTokenTransaction ...
func (txService TxService) BuildRawPrivacyCustomTokenTransaction(params interface{}, metaData metadata.Metadata) (*transaction.TxCustomTokenPrivacy, *RPCError) {
	txParam, errParam := bean.NewCreateRawPrivacyTokenTxParam(params)
	if errParam != nil {
		return nil, NewRPCError(RPCInvalidParamsError, errParam)
	}
	tokenParamsRaw := txParam.TokenParamsRaw
	var err error
	tokenParams, err := txService.BuildTokenParam(tokenParamsRaw, txParam.SenderKeySet, txParam.ShardIDSender)

	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if tokenParams == nil {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("can not build token params for request"))
	}
	/******* START choose output native coins(PRV), which is used to create tx *****/
	var inputCoins []*privacy.InputCoin
	realFeePRV := uint64(0)
	inputCoins, realFeePRV, err = txService.chooseOutsCoinByKeyset(txParam.PaymentInfos,
		txParam.EstimateFeeCoinPerKb, 0, txParam.SenderKeySet,
		txParam.ShardIDSender, txParam.HasPrivacyCoin, nil, tokenParams, txParam.IsGetPTokenFee, txParam.UnitPTokenFee)
	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if len(txParam.PaymentInfos) == 0 && realFeePRV == 0 {
		txParam.HasPrivacyCoin = false
	}
	/******* END GET output coins native coins(PRV), which is used to create tx *****/
	beaconView := txService.BlockChain.BeaconChain.GetFinalViewState()
	tx := &transaction.TxCustomTokenPrivacy{}
	err = tx.Init(
		transaction.NewTxPrivacyTokenInitParams(&txParam.SenderKeySet.PrivateKey,
			txParam.PaymentInfos,
			inputCoins,
			realFeePRV,
			tokenParams,
			txService.BlockChain.GetBestStateShard(txParam.ShardIDSender).GetCopiedTransactionStateDB(),
			metaData,
			txParam.HasPrivacyCoin,
			txParam.HasPrivacyToken,
			txParam.ShardIDSender, txParam.Info,
			beaconView.GetBeaconFeatureStateDB()))
	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}

	return tx, nil
}

// BuildRawCustomTokenTransactionV2 ...
func (txService TxService) BuildRawPrivacyCustomTokenTransactionV2(params interface{}, metaData metadata.Metadata) (*transaction.TxCustomTokenPrivacy, *RPCError) {
	txParam, errParam := bean.NewCreateRawPrivacyTokenTxParamV2(params)
	if errParam != nil {
		return nil, NewRPCError(RPCInvalidParamsError, errParam)
	}
	tokenParamsRaw := txParam.TokenParamsRaw
	var err error
	tokenParams, err := txService.BuildTokenParamV2(tokenParamsRaw, txParam.SenderKeySet, txParam.ShardIDSender)

	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if tokenParams == nil {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("can not build token params for request"))
	}
	/******* START choose output native coins(PRV), which is used to create tx *****/
	var inputCoins []*privacy.InputCoin
	realFeePRV := uint64(0)
	inputCoins, realFeePRV, err = txService.chooseOutsCoinByKeyset(txParam.PaymentInfos,
		txParam.EstimateFeeCoinPerKb, 0, txParam.SenderKeySet,
		txParam.ShardIDSender, txParam.HasPrivacyCoin, nil, tokenParams, txParam.IsGetPTokenFee, txParam.UnitPTokenFee)
	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if len(txParam.PaymentInfos) == 0 && realFeePRV == 0 {
		txParam.HasPrivacyCoin = false
	}
	/******* END GET output coins native coins(PRV), which is used to create tx *****/

	tx := &transaction.TxCustomTokenPrivacy{}
	err = tx.Init(
		transaction.NewTxPrivacyTokenInitParams(&txParam.SenderKeySet.PrivateKey,
			txParam.PaymentInfos,
			inputCoins,
			realFeePRV,
			tokenParams,
			txService.BlockChain.GetBestStateShard(txParam.ShardIDSender).GetCopiedTransactionStateDB(),
			metaData,
			txParam.HasPrivacyCoin,
			txParam.HasPrivacyToken,
			txParam.ShardIDSender, txParam.Info,
			txService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()))
	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}

	return tx, nil
}

func (txService TxService) GetTransactionHashByReceiver(paymentAddressParam string) (map[byte][]common.Hash, error) {
	var keySet *incognitokey.KeySet

	if paymentAddressParam != "" {
		senderKey, err := wallet.Base58CheckDeserialize(paymentAddressParam)
		if err != nil {
			return nil, errors.New("payment address is invalid")
		}

		keySet = &senderKey.KeySet
	} else {
		return nil, errors.New("payment address is invalid")
	}

	return txService.BlockChain.GetTransactionHashByReceiver(keySet)
}

// GetTransactionHashByReceiverV2 gets tx hashes by receiver in paging fashion
func (txService TxService) GetTransactionHashByReceiverV2(
	paymentAddressParam string,
	skip, limit uint,
) (map[byte][]common.Hash, error) {
	var keySet *incognitokey.KeySet

	if paymentAddressParam != "" {
		senderKey, err := wallet.Base58CheckDeserialize(paymentAddressParam)
		if err != nil {
			return nil, errors.New("payment address is invalid")
		}

		keySet = &senderKey.KeySet
	} else {
		return nil, errors.New("payment address is invalid")
	}

	return txService.BlockChain.GetTransactionHashByReceiverV2(keySet, skip, limit)
}

func (txService TxService) GetTransactionByHash(txHashStr string) (*jsonresult.TransactionDetail, *RPCError) {
	txHash, err := common.Hash{}.NewHashFromStr(txHashStr)
	if err != nil {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("tx hash is invalid"))
	}
	Logger.log.Infof("Get Transaction By Hash %+v", *txHash)

	shardID, blockHash, blockHeight, index, tx, err := txService.BlockChain.GetTransactionByHash(*txHash)
	if err != nil {
		// maybe tx is still in tx mempool -> check mempool
		tx, errM := txService.TxMemPool.GetTx(txHash)
		if errM != nil {
			return nil, NewRPCError(TxNotExistedInMemAndBLockError, errors.New("Tx is not existed in block or mempool"))
		}
		shardIDTemp := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
		result, errM := jsonresult.NewTransactionDetail(tx, nil, 0, 0, shardIDTemp)
		if errM != nil {
			return nil, NewRPCError(UnexpectedError, errM)
		}
		result.IsInMempool = true
		return result, nil
	}

	result, err := jsonresult.NewTransactionDetail(tx, &blockHash, blockHeight, index, shardID)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}
	result.IsInBlock = true
	Logger.log.Debugf("handleGetTransactionByHash result: %+v", result)
	return result, nil
}

func (txService TxService) ListPrivacyCustomToken() (map[common.Hash]*statedb.TokenState, error) {
	tokenStates, err := txService.BlockChain.ListAllPrivacyCustomTokenAndPRV()
	if err != nil {
		return tokenStates, err
	}
	delete(tokenStates, common.PRVCoinID)
	return tokenStates, nil
}

func (txService TxService) GetPrivacyTokenWithTxs(tokenID common.Hash) (*statedb.TokenState, error) {
	for _, i := range txService.BlockChain.GetShardIDs() {
		shardID := byte(i)
		tokenState, has, err := txService.BlockChain.GetPrivacyTokenState(tokenID, shardID)
		if err != nil {
			return tokenState, err
		}
		if has {
			return tokenState, nil
		}
	}
	return nil, nil
}

func (txService TxService) GetListPrivacyCustomTokenBalance(privateKey string) (jsonresult.ListCustomTokenBalance, *RPCError) {
	result := jsonresult.ListCustomTokenBalance{ListCustomTokenBalance: []jsonresult.CustomTokenBalance{}}
	resultM := make(map[string]jsonresult.CustomTokenBalance)
	account, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance result: %+v, err: %+v", nil, err)
		return jsonresult.ListCustomTokenBalance{}, NewRPCError(GetListPrivacyCustomTokenBalanceError, err)
	}
	err = account.KeySet.InitFromPrivateKey(&account.KeySet.PrivateKey)
	if err != nil {
		Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance result: %+v, err: %+v", nil, err)
		return jsonresult.ListCustomTokenBalance{}, NewRPCError(GetListPrivacyCustomTokenBalanceError, err)
	}
	result.PaymentAddress = account.Base58CheckSerialize(wallet.PaymentAddressType)
	lastByte := account.KeySet.PaymentAddress.Pk[len(account.KeySet.PaymentAddress.Pk)-1]
	shardIDSender := common.GetShardIDFromLastByte(lastByte)
	tokenStates, err := txService.ListPrivacyCustomToken()
	if err != nil {
		Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance result: %+v, err: %+v", nil, err)
		return jsonresult.ListCustomTokenBalance{}, NewRPCError(GetListPrivacyCustomTokenBalanceError, err)
	}
	for tokenID, tokenState := range tokenStates {
		item := jsonresult.CustomTokenBalance{}
		item.Name = tokenState.PropertyName()
		item.Symbol = tokenState.PropertySymbol()
		item.TokenID = tokenState.TokenID().String()
		item.TokenImage = common.Render([]byte(item.TokenID))
		balance := uint64(0)
		// get balance for accountName in wallet
		prvCoinID := &common.Hash{}
		err := prvCoinID.SetBytes(common.PRVCoinID[:])
		if err != nil {
			return jsonresult.ListCustomTokenBalance{}, NewRPCError(TokenIsInvalidError, err)
		}
		outcoints, err := txService.BlockChain.GetListOutputCoinsByKeyset(&account.KeySet, shardIDSender, &tokenID)
		if err != nil {
			Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance result: %+v, err: %+v", nil, err)
			return jsonresult.ListCustomTokenBalance{}, NewRPCError(GetListPrivacyCustomTokenBalanceError, err)
		}
		for _, out := range outcoints {
			balance += out.CoinDetails.GetValue()
		}
		item.Amount = balance
		if item.Amount == 0 {
			continue
		}
		item.IsPrivacy = true
		resultM[item.TokenID] = item
	}
	// bridge token
	_, allBridgeTokens, err := txService.BlockChain.GetAllBridgeTokens()
	if err != nil {
		return jsonresult.ListCustomTokenBalance{}, NewRPCError(GetListPrivacyCustomTokenBalanceError, err)
	}
	for _, bridgeToken := range allBridgeTokens {
		bridgeTokenID := bridgeToken.TokenID.String()
		if tokenInfo, ok := resultM[bridgeTokenID]; ok {
			tokenInfo.IsBridgeToken = true
			resultM[bridgeTokenID] = tokenInfo
			continue
		}
		item := jsonresult.CustomTokenBalance{}
		item.Name = ""
		item.Symbol = ""
		item.TokenID = bridgeToken.TokenID.String()
		item.TokenImage = common.Render([]byte(item.TokenID))
		tokenID := bridgeToken.TokenID
		balance := uint64(0)
		// get balance for accountName in wallet
		lastByte := account.KeySet.PaymentAddress.Pk[len(account.KeySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		prvCoinID := &common.Hash{}
		err := prvCoinID.SetBytes(common.PRVCoinID[:])
		if err != nil {
			return jsonresult.ListCustomTokenBalance{}, NewRPCError(TokenIsInvalidError, err)
		}
		outcoints, err := txService.BlockChain.GetListOutputCoinsByKeyset(&account.KeySet, shardIDSender, tokenID)
		if err != nil {
			return jsonresult.ListCustomTokenBalance{}, NewRPCError(UnexpectedError, err)
		}
		for _, out := range outcoints {
			balance += out.CoinDetails.GetValue()
		}
		item.Amount = balance
		if item.Amount == 0 {
			continue
		}
		item.IsPrivacy = true
		item.IsBridgeToken = true
		resultM[item.TokenID] = item
	}
	for _, v := range resultM {
		result.ListCustomTokenBalance = append(result.ListCustomTokenBalance, v)
	}
	return result, nil
}

func (txService TxService) GetBalancePrivacyCustomToken(privateKey string, tokenIDStr string) (uint64, *RPCError) {
	var totalValue uint64 = 0
	account, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v, err: %+v", nil, err)
		return uint64(0), NewRPCError(UnexpectedError, err)
	}
	err = account.KeySet.InitFromPrivateKey(&account.KeySet.PrivateKey)
	if err != nil {
		Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v, err: %+v", nil, err)
		return uint64(0), NewRPCError(UnexpectedError, err)
	}
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return uint64(0), NewRPCError(UnexpectedError, err)
	}
	isExisted := false
	for _, i := range txService.BlockChain.GetShardIDs() {
		shardID := byte(i)
		isExisted = txService.BlockChain.PrivacyCustomTokenIDExistedV2(tokenID, shardID)
		if isExisted {
			break
		}
	}
	if isExisted {
		lastByte := account.KeySet.PaymentAddress.Pk[len(account.KeySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		outcoints, err := txService.BlockChain.GetListOutputCoinsByKeyset(&account.KeySet, shardIDSender, tokenID)
		if err != nil {
			Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v, err: %+v", nil, err)
			return uint64(0), NewRPCError(UnexpectedError, err)
		}
		for _, out := range outcoints {
			totalValue += out.CoinDetails.GetValue()
		}
	}
	if totalValue == 0 {
		// bridge token
		allBridgeTokensBytes, err := statedb.GetAllBridgeTokens(txService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB())
		if err != nil {
			return 0, NewRPCError(UnexpectedError, err)
		}
		if len(allBridgeTokensBytes) > 0 {
			var allBridgeTokens []*rawdbv2.BridgeTokenInfo
			err = json.Unmarshal(allBridgeTokensBytes, &allBridgeTokens)
			if err != nil {
				return 0, NewRPCError(UnexpectedError, err)
			}
			if len(allBridgeTokens) > 0 {
				for _, bridgeToken := range allBridgeTokens {
					tempTokenID := bridgeToken.TokenID
					if tokenID.IsEqual(tempTokenID) {
						lastByte := account.KeySet.PaymentAddress.Pk[len(account.KeySet.PaymentAddress.Pk)-1]
						shardIDSender := common.GetShardIDFromLastByte(lastByte)
						outcoints, err := txService.BlockChain.GetListOutputCoinsByKeyset(&account.KeySet, shardIDSender, tempTokenID)
						if err != nil {
							Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v, err: %+v", nil, err)
							return uint64(0), NewRPCError(UnexpectedError, err)
						}
						for _, out := range outcoints {
							totalValue += out.CoinDetails.GetValue()
						}
					}
				}
			}
		}
	}

	return totalValue, nil
}

func (txService TxService) PrivacyCustomTokenDetail(tokenIDStr string) ([]common.Hash, *transaction.TxPrivacyTokenData, error) {
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		Logger.log.Debugf("handlePrivacyCustomTokenDetail result: %+v, err: %+v", nil, err)
		return nil, nil, err
	}
	tokenStates, err := txService.GetPrivacyTokenWithTxs(*tokenID)
	if err != nil {
		return nil, nil, err
	}
	if tokenStates == nil {
		return nil, nil, errors.New("Token not found")
	}
	tokenData := &transaction.TxPrivacyTokenData{}
	txs := []common.Hash{}
	tokenData.PropertyName = tokenStates.PropertyName()
	tokenData.PropertySymbol = tokenStates.PropertySymbol()
	txs = append(txs, tokenStates.InitTx())
	txs = append(txs, tokenStates.Txs()...)
	return txs, tokenData, nil
}

func (txService TxService) RandomCommitments(paymentAddressStr string, outputs []interface{}, tokenID *common.Hash) ([]uint64, []uint64, [][]byte, *RPCError) {
	_, shardIDSender, err := GetKeySetFromPaymentAddressParam(paymentAddressStr)
	if err != nil {
		Logger.log.Debugf("handleRandomCommitments result: %+v, err: %+v", nil, err)
		return nil, nil, nil, NewRPCError(UnexpectedError, err)
	}

	usableOutputCoins := []*privacy.OutputCoin{}
	for _, item := range outputs {
		out, err1 := jsonresult.NewOutcoinFromInterface(item)
		if err1 != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("outputs is invalid", out)))
		}
		valueBNTmp := big.Int{}
		valueBNTmp.SetString(out.Value, 10)

		outputCoin := new(privacy.OutputCoin).Init()
		outputCoin.CoinDetails.SetValue(valueBNTmp.Uint64())

		RandomnessInBytes, _, err := base58.Base58Check{}.Decode(out.Randomness)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("randomness output is invalid", out.Randomness)))
		}
		outputCoin.CoinDetails.SetRandomness(new(privacy.Scalar).FromBytesS(RandomnessInBytes))

		SNDerivatorInBytes, _, err := base58.Base58Check{}.Decode(out.SNDerivator)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("snderivator output is invalid", out.SNDerivator)))
		}
		outputCoin.CoinDetails.SetSNDerivator(new(privacy.Scalar).FromBytesS(SNDerivatorInBytes))

		CoinCommitmentBytes, _, err := base58.Base58Check{}.Decode(out.CoinCommitment)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("coin commitment output is invalid", out.CoinCommitment)))
		}
		CoinCommitment, err := new(privacy.Point).FromBytesS(CoinCommitmentBytes)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("coin commitment output is invalid", CoinCommitmentBytes)))
		}
		outputCoin.CoinDetails.SetCoinCommitment(CoinCommitment)

		PublicKeyBytes, _, err := base58.Base58Check{}.Decode(out.PublicKey)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("public key output is invalid", out.PublicKey)))
		}
		PublicKey, err := new(privacy.Point).FromBytesS(PublicKeyBytes)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("public key output is invalid", PublicKeyBytes)))
		}
		outputCoin.CoinDetails.SetPublicKey(PublicKey)

		InfoBytes, _, err := base58.Base58Check{}.Decode(out.Info)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("Info output is invalid", out.Info)))
		}
		outputCoin.CoinDetails.SetInfo(InfoBytes)

		usableOutputCoins = append(usableOutputCoins, outputCoin)
	}
	usableInputCoins := transaction.ConvertOutputCoinToInputCoin(usableOutputCoins)

	commitmentIndexs, myCommitmentIndexs, commitments := txService.BlockChain.RandomCommitmentsProcess(usableInputCoins, 0, shardIDSender, tokenID)
	return commitmentIndexs, myCommitmentIndexs, commitments, nil
}

func (txService TxService) SendRawPrivacyCustomTokenTransaction(base58CheckData string) (wire.Message, *transaction.TxCustomTokenPrivacy, *RPCError) {
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckData)
	if err != nil {
		Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, nil, NewRPCError(RPCInvalidParamsError, err)
	}

	tx := transaction.TxCustomTokenPrivacy{}
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, nil, NewRPCError(RPCInvalidParamsError, err)
	}

	beaconHeigh := int64(-1)
	beaconBestState, err := txService.BlockChain.GetClonedBeaconBestState()
	if err == nil {
		beaconHeigh = int64(beaconBestState.BeaconHeight)
	}
	hash, _, err := txService.TxMemPool.MaybeAcceptTransaction(&tx, beaconHeigh)
	if err != nil {
		Logger.log.Errorf("txService.SendRawPrivacyCustomTokenTransaction Try add tx into mempool of node with err: %+v", err)
		mempoolErr, ok := err.(*mempool.MempoolTxError)
		if ok {
			switch mempoolErr.Code {
			case mempool.ErrCodeMessage[mempool.RejectInvalidFee].Code:
				{
					return nil, nil, NewRPCError(RejectInvalidTxFeeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidSize].Code:
				{
					return nil, nil, NewRPCError(RejectInvalidTxSizeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidTxType].Code:
				{
					return nil, nil, NewRPCError(RejectInvalidTxTypeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidTx].Code:
				{
					return nil, nil, NewRPCError(RejectInvalidTxError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectReplacementTxError].Code:
				{
					return nil, nil, NewRPCError(RejectReplacementTx, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectDoubleSpendWithBlockchainTx].Code, mempool.ErrCodeMessage[mempool.RejectDoubleSpendWithMempoolTx].Code:
				{
					return nil, nil, NewRPCError(RejectDoubleSpendTxError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectDuplicateTx].Code:
				{
					return nil, nil, NewRPCError(RejectDuplicateTxInPoolError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectVersion].Code:
				{
					return nil, nil, NewRPCError(RejectDuplicateTxInPoolError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectSanityTxLocktime].Code:
				{
					return nil, nil, NewRPCError(RejectSanityTxLocktime, mempoolErr)
				}
			}
		}
		return nil, nil, NewRPCError(TxPoolRejectTxError, err)
	}

	Logger.log.Debugf("there is hash of transaction: %s\n", hash.String())

	txMsg, err := wire.MakeEmptyMessage(wire.CmdPrivacyCustomToken)
	if err != nil {
		Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, nil, NewRPCError(UnexpectedError, err)
	}

	txMsg.(*wire.MessageTxPrivacyToken).Transaction = &tx

	return txMsg, &tx, nil
}

func (txService TxService) BuildRawDefragmentAccountTransaction(params interface{}, meta metadata.Metadata) (*transaction.Tx, *RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 4 {
		return nil, NewRPCError(RPCInvalidParamsError, nil)
	}

	senderKeyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("senderKeyParam is invalid"))
	}

	maxValTemp, ok := arrayParams[1].(float64)
	if !ok {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("maxVal is invalid"))
	}
	maxVal := uint64(maxValTemp)

	estimateFeeCoinPerKbtemp, ok := arrayParams[2].(float64)
	if !ok {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("estimateFeeCoinPerKb is invalid"))
	}
	estimateFeeCoinPerKb := int64(estimateFeeCoinPerKbtemp)

	// param #4: hasPrivacyCoin flag: 1 or -1
	hasPrivacyCoinParam := arrayParams[3].(float64)
	hasPrivacyCoin := int(hasPrivacyCoinParam) > 0

	maxDefragmentQuantity := 32
	if len(arrayParams) >= 5 {
		maxDefragmentQuantityTemp, ok := arrayParams[4].(float64)
		if !ok {
			maxDefragmentQuantityTemp = 32
		}
		if maxDefragmentQuantityTemp > 32 || maxDefragmentQuantityTemp <= 0 {
			maxDefragmentQuantityTemp = 32
		}
		maxDefragmentQuantity = int(maxDefragmentQuantityTemp)
	}
	/********* END Fetch all component to *******/

	// param #1: private key of sender
	senderKeySet, shardIDSender, err := GetKeySetFromPrivateKeyParams(senderKeyParam)
	if err != nil {
		return nil, NewRPCError(InvalidSenderPrivateKeyError, err)
	}

	prvCoinID := &common.Hash{}
	err1 := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err1 != nil {
		return nil, NewRPCError(TokenIsInvalidError, err1)
	}
	outCoins, err := txService.BlockChain.GetListOutputCoinsByKeyset(senderKeySet, shardIDSender, prvCoinID)
	if err != nil {
		return nil, NewRPCError(GetOutputCoinError, err)
	}
	// remove out coin in mem pool
	outCoins, err = txService.filterMemPoolOutcoinsToSpent(outCoins)
	if err != nil {
		return nil, NewRPCError(GetOutputCoinError, err)
	}
	outCoins, amount := txService.calculateOutputCoinsByMinValue(outCoins, maxVal, maxDefragmentQuantity)
	if len(outCoins) == 0 {
		return nil, NewRPCError(GetOutputCoinError, nil)
	}
	paymentInfo := &privacy.PaymentInfo{
		Amount:         uint64(amount),
		PaymentAddress: senderKeySet.PaymentAddress,
		Message:        []byte{},
	}
	paymentInfos := []*privacy.PaymentInfo{paymentInfo}
	// check real fee(nano PRV) per tx
	isGetPTokenFee := false
	beaconState := txService.BlockChain.GetBeaconBestState()
	beaconHeight := beaconState.BeaconHeight
	realFee, _, _, _ := txService.EstimateFee(
		estimateFeeCoinPerKb, isGetPTokenFee, outCoins, paymentInfos, shardIDSender, 8, hasPrivacyCoin, nil, nil, int64(beaconHeight))
	if len(outCoins) == 0 {
		realFee = 0
	}
	if uint64(amount) < realFee {
		return nil, NewRPCError(GetOutputCoinError, err)
	}
	paymentInfo.Amount = uint64(amount) - realFee
	inputCoins := transaction.ConvertOutputCoinToInputCoin(outCoins)
	/******* END GET output native coins(PRV), which is used to create tx *****/
	// START create tx
	// missing flag for privacy
	// false by default
	tx := transaction.Tx{}
	err = tx.Init(
		transaction.NewTxPrivacyInitParams(&senderKeySet.PrivateKey,
			paymentInfos,
			inputCoins,
			realFee,
			hasPrivacyCoin,
			txService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB(),
			nil, // use for prv coin -> nil is valid
			meta, nil))
	// END create tx

	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}

	return &tx, nil
}

//calculateOutputCoinsByMinValue
func (txService TxService) calculateOutputCoinsByMinValue(outCoins []*privacy.OutputCoin, maxVal uint64, maxDefragmentQuantityTemp int) ([]*privacy.OutputCoin, uint64) {
	outCoinsTmp := make([]*privacy.OutputCoin, 0)
	amount := uint64(0)
	for _, outCoin := range outCoins {
		if outCoin.CoinDetails.GetValue() <= maxVal {
			outCoinsTmp = append(outCoinsTmp, outCoin)
			amount += outCoin.CoinDetails.GetValue()
			if len(outCoinsTmp) >= maxDefragmentQuantityTemp {
				break
			}
		}
	}
	return outCoinsTmp, amount
}

/*func (txService TxService) SendRawTxWithMetadata(txBase58CheckData string) (wire.Message, *common.Hash, byte, *RPCError) {
	// Decode base58check data of tx
	rawTxBytes, _, err := base58.Base58Check{}.Decode(txBase58CheckData)
	if err != nil {
		Logger.log.Errorf("txService.SendRawTxWithMetadata fail with err: %+v", err)
		return nil, nil, byte(0), NewRPCError(Base58ChedkDataOfTxInvalid, err)
	}

	// Unmarshal from json data to object tx
	tx := transaction.Tx{}
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		Logger.log.Errorf("txService.SendRawTxWithMetadata fail with err: %+v", err)
		return nil, nil, byte(0), NewRPCError(JsonDataOfTxInvalid, err)
	}

	beaconHeigh := int64(-1)
	beaconBestState, err := txService.BlockChain.BestState.GetClonedBeaconBestState()
	if err == nil {
		beaconHeigh = int64(beaconBestState.BeaconHeight)
	} else {
		Logger.log.Errorf("txService.SendRawTransaction can not get beacon best state with error %+v", err)
	}

	// Try add tx in to mempool of node
	hash, _, err := txService.TxMemPool.MaybeAcceptTransaction(&tx, beaconHeigh)
	if err != nil {
		Logger.log.Errorf("txService.SendRawTxWithMetadata Try add tx into mempool of node with err: %+v", err)
		mempoolErr, ok := err.(*mempool.MempoolTxError)
		if ok {
			switch mempoolErr.Code {
			case mempool.ErrCodeMessage[mempool.RejectInvalidFee].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxFeeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidSize].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxSizeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidTxType].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxTypeError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectInvalidTx].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectInvalidTxError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectDoubleSpendWithBlockchainTx].Code, mempool.ErrCodeMessage[mempool.RejectDoubleSpendWithMempoolTx].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectDoubleSpendTxError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectDuplicateTx].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectDuplicateTxInPoolError, mempoolErr)
				}
			case mempool.ErrCodeMessage[mempool.RejectVersion].Code:
				{
					return nil, nil, byte(0), NewRPCError(RejectDuplicateTxInPoolError, mempoolErr)
				}
			}
		}
		return nil, nil, byte(0), NewRPCError(TxPoolRejectTxError, err)
	}

	Logger.log.Debugf("there is hash of transaction: %s\n", hash.String())

	// Create tx message for broadcasting
	txMsg, err := wire.MakeEmptyMessage(wire.CmdTx)
	if err != nil {
		Logger.log.Errorf("txService.SendRawTxWithMetadata Create tx message for broadcasting with err: %+v", err)
		return nil, nil, byte(0), NewRPCError(SendTxDataError, err)

	txMsg.(*wire.MessageTx).Transaction = &tx

	return txMsg, hash, tx.PubKeyLastByteSender, nil
}*/


func (txService TxService) buildTxInfosFromTxHashs(
	listTxsHash map[byte][]common.Hash,
	keySet incognitokey.KeySet,
) *jsonresult.ListReceivedTransaction {
	result := jsonresult.ListReceivedTransaction{
		ReceivedTransactions: []jsonresult.ReceivedTransaction{},
	}
	for shardID, txHashs := range listTxsHash {
		for _, txHash := range txHashs {
			item := jsonresult.ReceivedTransaction{
				FromShardID:     shardID,
				ReceivedAmounts: map[common.Hash]jsonresult.ReceivedInfo{},
			}
			if len(keySet.ReadonlyKey.Rk) != 0 {
				_, blockHash, _, _, txDetail, _ := txService.BlockChain.GetTransactionByHash(txHash)
				item.LockTime = time.Unix(txDetail.GetLockTime(), 0).Format(common.DateOutputFormat)
				item.Info = base58.Base58Check{}.Encode(txDetail.GetInfo(), common.ZeroByte)
				item.BlockHash = blockHash.String()
				item.Hash = txDetail.Hash().String()

				txType := txDetail.GetType()
				item.Type = txType
				switch item.Type {
				case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType:
					{
						normalTx := txDetail.(*transaction.Tx)
						item.Version = normalTx.Version
						item.IsPrivacy = normalTx.IsPrivacy()
						item.Fee = normalTx.Fee

						proof := normalTx.GetProof()
						if proof != nil {
							outputs := proof.GetOutputCoins()
							for _, output := range outputs {
								if bytes.Equal(output.CoinDetails.GetPublicKey().ToBytesS(), keySet.PaymentAddress.Pk) {
									temp := &privacy.OutputCoin{
										CoinDetails:          output.CoinDetails,
										CoinDetailsEncrypted: output.CoinDetailsEncrypted,
									}
									if temp.CoinDetailsEncrypted != nil && !temp.CoinDetailsEncrypted.IsNil() {
										// try to decrypt to get more data
										err := temp.Decrypt(keySet.ReadonlyKey)
										if err != nil {
											Logger.log.Error(err)
											continue
										}
									}
									info := jsonresult.ReceivedInfo{
										CoinDetails: jsonresult.ReceivedCoin{
											Info:      base58.Base58Check{}.Encode(temp.CoinDetails.GetInfo(), common.ZeroByte),
											PublicKey: base58.Base58Check{}.Encode(temp.CoinDetails.GetPublicKey().ToBytesS(), common.ZeroByte),
											Value:     temp.CoinDetails.GetValue(),
										},
									}
									if temp.CoinDetailsEncrypted != nil {
										info.CoinDetailsEncrypted = base58.Base58Check{}.Encode(temp.CoinDetailsEncrypted.Bytes(), common.ZeroByte)
									}
									item.ReceivedAmounts[common.PRVCoinID] = info
								}
							}
						}
					}
				case common.TxCustomTokenPrivacyType:
					{
						privacyTokenTx := txDetail.(*transaction.TxCustomTokenPrivacy)
						item.Version = privacyTokenTx.Version
						item.IsPrivacy = privacyTokenTx.IsPrivacy()
						item.PrivacyCustomTokenIsPrivacy = privacyTokenTx.TxPrivacyTokenData.TxNormal.IsPrivacy()
						item.Fee = privacyTokenTx.Fee
						item.PrivacyCustomTokenFee = privacyTokenTx.TxPrivacyTokenData.TxNormal.Fee
						item.PrivacyCustomTokenID = privacyTokenTx.TxPrivacyTokenData.PropertyID.String()
						item.PrivacyCustomTokenName = privacyTokenTx.TxPrivacyTokenData.PropertyName
						item.PrivacyCustomTokenSymbol = privacyTokenTx.TxPrivacyTokenData.PropertySymbol

						// prv proof
						proof := privacyTokenTx.GetProof()
						if proof != nil {
							outputs := proof.GetOutputCoins()
							for _, output := range outputs {
								if bytes.Equal(output.CoinDetails.GetPublicKey().ToBytesS(), keySet.PaymentAddress.Pk) {
									temp := &privacy.OutputCoin{
										CoinDetails:          output.CoinDetails,
										CoinDetailsEncrypted: output.CoinDetailsEncrypted,
									}
									if temp.CoinDetailsEncrypted != nil && !temp.CoinDetailsEncrypted.IsNil() {
										// try to decrypt to get more data
										err := temp.Decrypt(keySet.ReadonlyKey)
										if err != nil {
											Logger.log.Error(err)
											continue
										}
									}
									info := jsonresult.ReceivedInfo{
										CoinDetails: jsonresult.ReceivedCoin{
											Info:      base58.Base58Check{}.Encode(temp.CoinDetails.GetInfo(), common.ZeroByte),
											PublicKey: base58.Base58Check{}.Encode(temp.CoinDetails.GetPublicKey().ToBytesS(), common.ZeroByte),
											Value:     temp.CoinDetails.GetValue(),
										},
									}
									if temp.CoinDetailsEncrypted != nil {
										info.CoinDetailsEncrypted = base58.Base58Check{}.Encode(temp.CoinDetailsEncrypted.Bytes(), common.ZeroByte)
									}
									item.ReceivedAmounts[common.PRVCoinID] = info
								}
							}
						}

						// token proof
						proof = privacyTokenTx.TxPrivacyTokenData.TxNormal.GetProof()
						if proof != nil {
							outputs := proof.GetOutputCoins()
							for _, output := range outputs {
								if bytes.Equal(output.CoinDetails.GetPublicKey().ToBytesS(), keySet.PaymentAddress.Pk) {
									temp := &privacy.OutputCoin{
										CoinDetails:          output.CoinDetails,
										CoinDetailsEncrypted: output.CoinDetailsEncrypted,
									}
									if temp.CoinDetailsEncrypted != nil && !temp.CoinDetailsEncrypted.IsNil() {
										// try to decrypt to get more data
										err := temp.Decrypt(keySet.ReadonlyKey)
										if err != nil {
											Logger.log.Error(err)
											continue
										}
									}
									info := jsonresult.ReceivedInfo{
										CoinDetails: jsonresult.ReceivedCoin{
											Info:      base58.Base58Check{}.Encode(temp.CoinDetails.GetInfo(), common.ZeroByte),
											PublicKey: base58.Base58Check{}.Encode(temp.CoinDetails.GetPublicKey().ToBytesS(), common.ZeroByte),
											Value:     temp.CoinDetails.GetValue(),
										},
									}
									if temp.CoinDetailsEncrypted != nil {
										info.CoinDetailsEncrypted = base58.Base58Check{}.Encode(temp.CoinDetailsEncrypted.Bytes(), common.ZeroByte)
									}
									item.ReceivedAmounts[privacyTokenTx.TxPrivacyTokenData.PropertyID] = info
								}
							}
						}
					}
				}
			}
			result.ReceivedTransactions = append(result.ReceivedTransactions, item)
			sort.Slice(result.ReceivedTransactions, func(i, j int) bool {
				return result.ReceivedTransactions[i].LockTime > result.ReceivedTransactions[j].LockTime
			})
		}
	}
	return &result
}

// GetTransactionByReceiver - from keyset of receiver, we can get list tx hash which be sent to receiver
// if this keyset contain payment-addr, we can detect tx hash
// if this keyset contain viewing key, we can detect amount in tx, but can not know output in tx is spent
// because this is monitoring output to get received tx -> can not know this is a returned amount tx
func (txService TxService) GetTransactionByReceiver(keySet incognitokey.KeySet) (*jsonresult.ListReceivedTransaction, *RPCError) {
	if len(keySet.PaymentAddress.Pk) == 0 {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("Missing payment address"))
	}
	listTxsHash, err := txService.BlockChain.GetTransactionHashByReceiver(&keySet)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, errors.New("Can not find any tx"))
	}
	result := txService.buildTxInfosFromTxHashs(listTxsHash, keySet)
	return result, nil
}


func (txService TxService) buildTxDetails(
	txInfos []TxInfo,
	keySet incognitokey.KeySet,
) []jsonresult.ReceivedTransactionV2 {
	txDetails := []jsonresult.ReceivedTransactionV2{}
	for _, txInfo := range txInfos {
		item := jsonresult.ReceivedTransactionV2{
			ReceivedAmounts: map[common.Hash][]jsonresult.ReceivedInfo{},
			InputSerialNumbers: map[common.Hash][]string{},
		}
		if len(keySet.ReadonlyKey.Rk) != 0 {
			_, blockHash, _, _, txDetail, _ := txService.BlockChain.GetTransactionByHash(*txInfo.Tx.Hash())
			item.LockTime = time.Unix(txDetail.GetLockTime(), 0).Format(common.DateOutputFormat)
			item.Info = base58.Base58Check{}.Encode(txDetail.GetInfo(), common.ZeroByte)
			item.BlockHash = blockHash.String()
			item.Hash = txDetail.Hash().String()

			txType := txDetail.GetType()
			item.Type = txType
			switch item.Type {
			case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType:
				{
					normalTx, ok := txDetail.(*transaction.Tx)
					if !ok || normalTx == nil {
						continue
					}
					item.Version = normalTx.Version
					item.IsPrivacy = normalTx.IsPrivacy()
					item.Fee = normalTx.Fee

					// marshal metadata
					if txDetail.GetMetadata() != nil {
						metaData, _ := json.MarshalIndent(txDetail.GetMetadata(), "", "\t")
						item.Metadata = string(metaData)
					}

					proof := normalTx.GetProof()
					if proof != nil {
						// add list input coins' serial number
						inputCoins := proof.GetInputCoins()
						for _, in := range inputCoins {
							item.InputSerialNumbers[common.PRVCoinID] = append(item.InputSerialNumbers[common.PRVCoinID],
								base58.Base58Check{}.Encode(in.CoinDetails.GetSerialNumber().ToBytesS(), common.ZeroByte))
						}

						outputs := proof.GetOutputCoins()
						for _, output := range outputs {
							if bytes.Equal(output.CoinDetails.GetPublicKey().ToBytesS(), keySet.PaymentAddress.Pk) {
								temp := &privacy.OutputCoin{
									CoinDetails:          output.CoinDetails,
									CoinDetailsEncrypted: output.CoinDetailsEncrypted,
								}
								if temp.CoinDetailsEncrypted != nil && !temp.CoinDetailsEncrypted.IsNil() {
									// try to decrypt to get more data
									err := temp.Decrypt(keySet.ReadonlyKey)
									if err != nil {
										Logger.log.Error(err)
										continue
									}
								}
								info := jsonresult.ReceivedInfo{
									CoinDetails: jsonresult.ReceivedCoin{
										Info:      base58.Base58Check{}.Encode(temp.CoinDetails.GetInfo(), common.ZeroByte),
										PublicKey: base58.Base58Check{}.Encode(temp.CoinDetails.GetPublicKey().ToBytesS(), common.ZeroByte),
										Value:     temp.CoinDetails.GetValue(),
									},
								}
								if temp.CoinDetailsEncrypted != nil {
									info.CoinDetailsEncrypted = base58.Base58Check{}.Encode(temp.CoinDetailsEncrypted.Bytes(), common.ZeroByte)
								}
								item.ReceivedAmounts[common.PRVCoinID] = append(item.ReceivedAmounts[common.PRVCoinID], info)
							}
						}
					}
				}
			case common.TxCustomTokenPrivacyType:
				{
					privacyTokenTx, ok := txDetail.(*transaction.TxCustomTokenPrivacy)
					if !ok || privacyTokenTx == nil {
						continue
					}
					item.Version = privacyTokenTx.Version
					item.IsPrivacy = privacyTokenTx.IsPrivacy()
					item.PrivacyCustomTokenIsPrivacy = privacyTokenTx.TxPrivacyTokenData.TxNormal.IsPrivacy()
					item.Fee = privacyTokenTx.Fee
					item.PrivacyCustomTokenFee = privacyTokenTx.TxPrivacyTokenData.TxNormal.Fee
					item.PrivacyCustomTokenID = privacyTokenTx.TxPrivacyTokenData.PropertyID.String()
					item.PrivacyCustomTokenName = privacyTokenTx.TxPrivacyTokenData.PropertyName
					item.PrivacyCustomTokenSymbol = privacyTokenTx.TxPrivacyTokenData.PropertySymbol

					// marshal metadata
					if txDetail.GetMetadata() != nil {
						metaData, _ := json.MarshalIndent(txDetail.GetMetadata(), "", "\t")
						item.Metadata = string(metaData)
					}

					// prv proof
					proof := privacyTokenTx.GetProof()
					if proof != nil {
						// list serial number of native input coins
						nativeInputCoins := proof.GetInputCoins()
						for _, in := range nativeInputCoins {
							item.InputSerialNumbers[common.PRVCoinID] = append(item.InputSerialNumbers[common.PRVCoinID],
								base58.Base58Check{}.Encode(in.CoinDetails.GetSerialNumber().ToBytesS(), common.ZeroByte))
						}

						outputs := proof.GetOutputCoins()
						for _, output := range outputs {
							if bytes.Equal(output.CoinDetails.GetPublicKey().ToBytesS(), keySet.PaymentAddress.Pk) {
								temp := &privacy.OutputCoin{
									CoinDetails:          output.CoinDetails,
									CoinDetailsEncrypted: output.CoinDetailsEncrypted,
								}
								if temp.CoinDetailsEncrypted != nil && !temp.CoinDetailsEncrypted.IsNil() {
									// try to decrypt to get more data
									err := temp.Decrypt(keySet.ReadonlyKey)
									if err != nil {
										Logger.log.Error(err)
										continue
									}
								}
								info := jsonresult.ReceivedInfo{
									CoinDetails: jsonresult.ReceivedCoin{
										Info:      base58.Base58Check{}.Encode(temp.CoinDetails.GetInfo(), common.ZeroByte),
										PublicKey: base58.Base58Check{}.Encode(temp.CoinDetails.GetPublicKey().ToBytesS(), common.ZeroByte),
										Value:     temp.CoinDetails.GetValue(),
									},
								}
								if temp.CoinDetailsEncrypted != nil {
									info.CoinDetailsEncrypted = base58.Base58Check{}.Encode(temp.CoinDetailsEncrypted.Bytes(), common.ZeroByte)
								}
								item.ReceivedAmounts[common.PRVCoinID] = append(item.ReceivedAmounts[common.PRVCoinID], info)
							}
						}
					}

					// token proof
					proof = privacyTokenTx.TxPrivacyTokenData.TxNormal.GetProof()
					if proof != nil {
						// list serial number of ptoken input coins
						ptokenInputCoins := proof.GetInputCoins()
						for _, in := range ptokenInputCoins {
							item.InputSerialNumbers[privacyTokenTx.TxPrivacyTokenData.PropertyID] = append(item.InputSerialNumbers[privacyTokenTx.TxPrivacyTokenData.PropertyID],
								base58.Base58Check{}.Encode(in.CoinDetails.GetSerialNumber().ToBytesS(), common.ZeroByte))
						}

						outputs := proof.GetOutputCoins()
						for _, output := range outputs {
							if bytes.Equal(output.CoinDetails.GetPublicKey().ToBytesS(), keySet.PaymentAddress.Pk) {
								temp := &privacy.OutputCoin{
									CoinDetails:          output.CoinDetails,
									CoinDetailsEncrypted: output.CoinDetailsEncrypted,
								}
								if temp.CoinDetailsEncrypted != nil && !temp.CoinDetailsEncrypted.IsNil() {
									// try to decrypt to get more data
									err := temp.Decrypt(keySet.ReadonlyKey)
									if err != nil {
										Logger.log.Error(err)
										continue
									}
								}
								info := jsonresult.ReceivedInfo{
									CoinDetails: jsonresult.ReceivedCoin{
										Info:      base58.Base58Check{}.Encode(temp.CoinDetails.GetInfo(), common.ZeroByte),
										PublicKey: base58.Base58Check{}.Encode(temp.CoinDetails.GetPublicKey().ToBytesS(), common.ZeroByte),
										Value:     temp.CoinDetails.GetValue(),
									},
								}
								if temp.CoinDetailsEncrypted != nil {
									info.CoinDetailsEncrypted = base58.Base58Check{}.Encode(temp.CoinDetailsEncrypted.Bytes(), common.ZeroByte)
								}
								item.ReceivedAmounts[privacyTokenTx.TxPrivacyTokenData.PropertyID] = append(item.ReceivedAmounts[privacyTokenTx.TxPrivacyTokenData.PropertyID], info)
							}
						}
					}
				}
			}
		}
		txDetails = append(txDetails, item)
	}
	return txDetails
}

func (txService TxService) getTxsByHashs(txHashs []common.Hash, ch chan []TxInfo, tokenID common.Hash, pubKey []byte) {
	txInfos := []TxInfo{}
	for _, txHash := range txHashs {
		_, blockHash, _, _, txDetail, _ := txService.BlockChain.GetTransactionByHash(txHash)
		// filter by tokenID
		if bytes.Equal(tokenID.Bytes(), common.PRVCoinID.Bytes()) {
			// case 1: tokenID is PRVID, include normal Tx and privacyTx that have native output coins for public key
			if txDetail.GetType() == common.TxCustomTokenPrivacyType {
				txPToken, ok := txDetail.(*transaction.TxCustomTokenPrivacy)
				if !ok || txPToken == nil {
					continue
				}
				nativeProof := txPToken.GetProof()
				if nativeProof == nil {
					continue
				}
				nativeOutputCoins := nativeProof.GetOutputCoins()
				hasNativeOutCoin := false
				for _, out := range nativeOutputCoins {
					if bytes.Equal(out.CoinDetails.GetPublicKey().ToBytesS(), pubKey) {
						hasNativeOutCoin = true
						break
					}
				}
				if !hasNativeOutCoin {
					continue
				}
			}
		} else {
			// case 2: tokenID is others, include privacyTx that have ptoken output coins for public key
			if txDetail.GetType() != common.TxCustomTokenPrivacyType {
				continue
			}

			txPToken, ok := txDetail.(*transaction.TxCustomTokenPrivacy)
			if !ok || txPToken == nil {
				continue
			}
			if !bytes.Equal(txPToken.GetTokenID().Bytes(), tokenID.Bytes()) {
				continue
			}

			pTokenProof := txPToken.TxPrivacyTokenData.TxNormal.GetProof()
			if pTokenProof == nil {
				continue
			}
			pTokenOutputCoins := pTokenProof.GetOutputCoins()
			hasPTokenOutCoin := false
			for _, out := range pTokenOutputCoins {
				if bytes.Equal(out.CoinDetails.GetPublicKey().ToBytesS(), pubKey) {
					hasPTokenOutCoin = true
					break
				}
			}
			if !hasPTokenOutCoin {
				continue
			}
		}

		txInfo := TxInfo{
			BlockHash: blockHash,
			Tx: txDetail,
		}
		txInfos = append(txInfos, txInfo)
	}
	ch <- txInfos
}

// GetTransactionByReceiverV2 - from keyset of receiver, we can get list tx hash which be sent to receiver
// if this keyset contain payment-addr, we can detect tx hash
// if this keyset contain viewing key, we can detect amount in tx, but can not know output in tx is spent
// because this is monitoring output to get received tx -> can not know this is a returned amount tx
func (txService TxService) GetTransactionByReceiverV2(
	keySet incognitokey.KeySet,
	skip, limit uint,
	tokenIDHash common.Hash,
) (*jsonresult.ListReceivedTransactionV2, uint, *RPCError) {
	result := &jsonresult.ListReceivedTransactionV2{
		ReceivedTransactions: []jsonresult.ReceivedTransactionV2{},
	}
	if len(keySet.PaymentAddress.Pk) == 0 {
		return nil, 0, NewRPCError(RPCInvalidParamsError, errors.New("Missing payment address"))
	}
	listTxsHash, err := txService.BlockChain.GetTransactionHashByReceiver(&keySet)
	if err != nil {
		return nil, 0, NewRPCError(UnexpectedError, errors.New("Cannot find any tx"))
	}

	allTxHashs := []common.Hash{}
	for _, txHashs := range listTxsHash {
		allTxHashs = append(allTxHashs, txHashs...)
	}
	totalTxHashs := len(allTxHashs)
	chunksNum := 32 // number of concurrent gorountines
	chunkSize := totalTxHashs / (chunksNum + 1)
	if chunkSize == 0 {
		chunkSize = 1
	}

	actualChunksNum := chunksNum
	if chunkSize == 1 {
		actualChunksNum = totalTxHashs
	}
	ch := make(chan []TxInfo)
	for i := 0; i < actualChunksNum; i++ {
		start := chunkSize * i
		end := start + chunkSize
		chunk := allTxHashs[start : end]
		go txService.getTxsByHashs(chunk, ch, tokenIDHash, keySet.PaymentAddress.Pk)
	}

	txInfos := []TxInfo{}
	for i := 0; i < actualChunksNum; i++ {
		chunkedTxInfos := <- ch
		txInfos = append(txInfos, chunkedTxInfos...)
	}

	sort.SliceStable(txInfos, func(i, j int) bool {
		return txInfos[i].Tx.GetLockTime() > txInfos[j].Tx.GetLockTime()
	})
	txNum := uint(len(txInfos))
	if skip >= txNum {
		return result, 0, nil
	}
	limit = skip + limit
	if limit > txNum {
		limit = txNum
	}
	pagingTxInfos := txInfos[skip:limit]
	txDetails := txService.buildTxDetails(pagingTxInfos, keySet)
	result.ReceivedTransactions = txDetails
	return result, txNum, nil
}

func (txService TxService) DecryptOutputCoinByKeyByTransaction(keyParam *incognitokey.KeySet, txHashStr string) (map[string]interface{}, *RPCError) {
	txHash, err := common.Hash{}.NewHashFromStr(txHashStr)
	if err != nil {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("tx hash is invalid"))
	}

	isInMempool := false
	_, _, _, _, tx, err := txService.BlockChain.GetTransactionByHash(*txHash)
	if err != nil {
		// maybe tx is still in tx mempool -> check mempool
		var errM error
		tx, errM = txService.TxMemPool.GetTx(txHash)
		if errM != nil {
			return nil, NewRPCError(TxNotExistedInMemAndBLockError, errors.New("Tx is not existed in block or mempool"))
		}
		isInMempool = true
	}

	results := make(map[string]interface{})
	results["IsMempool"] = isInMempool
	results[common.PRVCoinID.String()] = 0

	switch tx.GetType() {
	case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType:
		{
			tempTx := tx.(*transaction.Tx)
			prvOutputs, _ := txService.DecryptOutputCoinByKey(tempTx.Proof.GetOutputCoins(), keyParam)
			if len(prvOutputs) > 0 {
				totalPrvValue := uint64(0)
				for _, output := range prvOutputs {
					totalPrvValue += output.CoinDetails.GetValue()
				}
				results[common.PRVCoinID.String()] = totalPrvValue
			}
		}
	case common.TxCustomTokenPrivacyType:
		{
			tempTx := tx.(*transaction.TxCustomTokenPrivacy)
			outputOfPrv := tempTx.Proof.GetOutputCoins()
			if len(outputOfPrv) > 0 {
				prvOutputs, _ := txService.DecryptOutputCoinByKey(outputOfPrv, keyParam)
				if len(prvOutputs) > 0 {
					totalPrvValue := uint64(0)
					for _, output := range prvOutputs {
						totalPrvValue += output.CoinDetails.GetValue()
					}
					results[common.PRVCoinID.String()] = totalPrvValue
				}
			}

			results[tempTx.TxPrivacyTokenData.PropertyID.String()] = 0
			outputOfTokens := tempTx.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins()
			if len(outputOfTokens) > 0 {
				tokenOutput, _ := txService.DecryptOutputCoinByKey(outputOfTokens, keyParam)
				if len(tokenOutput) > 0 {
					totalTokenValue := uint64(0)
					for _, output := range tokenOutput {
						totalTokenValue += output.CoinDetails.GetValue()
					}
					results[tempTx.TxPrivacyTokenData.PropertyID.String()] = totalTokenValue
				}
			}
		}
	default:
		{
			return nil, NewRPCError(UnexpectedError, errors.New("tx type is invalid"))
		}
	}
	return results, nil
}
func (txService TxService) DecryptOutputCoinByKey(outCoints []*privacy.OutputCoin, keyset *incognitokey.KeySet) ([]*privacy.OutputCoin, *RPCError) {
	keyset.PrivateKey = nil // always nil
	results := make([]*privacy.OutputCoin, 0)
	for _, out := range outCoints {
		decryptedOut := blockchain.DecryptOutputCoinByKey(txService.BlockChain.GetBestStateTransactionStateDB(0), out, keyset, nil, 0)
		if decryptedOut == nil {
			continue
		} else {
			results = append(results, decryptedOut)
		}
	}

	return results, nil
}

func (txService TxService) BuildRawDefragmentPrivacyCustomTokenTransaction(params interface{}, metaData metadata.Metadata) (*transaction.TxCustomTokenPrivacy, *RPCError) {
	txParam, errParam := bean.NewCreateRawPrivacyTokenTxParam(params)
	if errParam != nil {
		return nil, NewRPCError(RPCInvalidParamsError, errParam)
	}
	tokenParamsRaw := txParam.TokenParamsRaw
	var err error
	tokenParams, err := txService.BuildDefragmentTokenParam(tokenParamsRaw, txParam.SenderKeySet, txParam.ShardIDSender)

	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if tokenParams == nil {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("can not build token params for request"))
	}
	/******* START choose output native coins(PRV), which is used to create tx *****/
	var inputCoins []*privacy.InputCoin
	realFeePRV := uint64(0)
	inputCoins, realFeePRV, err = txService.chooseOutsCoinByKeyset(txParam.PaymentInfos,
		txParam.EstimateFeeCoinPerKb, 0, txParam.SenderKeySet,
		txParam.ShardIDSender, txParam.HasPrivacyCoin, nil, tokenParams, txParam.IsGetPTokenFee, txParam.UnitPTokenFee)
	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if len(txParam.PaymentInfos) == 0 && realFeePRV == 0 {
		txParam.HasPrivacyCoin = false
	}
	/******* END GET output coins native coins(PRV), which is used to create tx *****/

	shardBestState, _ := txService.BlockChain.GetClonedAShardBestState(txParam.ShardIDSender)
	tx := &transaction.TxCustomTokenPrivacy{}
	err = tx.Init(
		transaction.NewTxPrivacyTokenInitParams(&txParam.SenderKeySet.PrivateKey,
			txParam.PaymentInfos,
			inputCoins,
			realFeePRV,
			tokenParams,
			shardBestState.GetCopiedTransactionStateDB(),
			metaData,
			txParam.HasPrivacyCoin,
			txParam.HasPrivacyToken,
			txParam.ShardIDSender, txParam.Info,
			txService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()))
	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}

	return tx, nil
}

func (txService TxService) BuildDefragmentTokenParam(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.CustomTokenPrivacyParamTx, *RPCError) {
	var privacyTokenParam *transaction.CustomTokenPrivacyParamTx
	var err *RPCError
	isPrivacy, ok := tokenParamsRaw["Privacy"].(bool)
	if !ok {
		return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Params %+v", tokenParamsRaw))
	}
	if !isPrivacy {
		// Check normal custom token param
	} else {
		// Check privacy custom token param
		privacyTokenParam, _, _, err = txService.BuildDefragmentPrivacyCustomTokenParam(tokenParamsRaw, senderKeySet, shardIDSender)
		if err != nil {
			return nil, NewRPCError(BuildTokenParamError, err)
		}
	}
	return privacyTokenParam, nil
}

func (txService TxService) BuildDefragmentPrivacyCustomTokenParam(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.CustomTokenPrivacyParamTx, map[common.Hash]transaction.TxCustomTokenPrivacy, map[common.Hash]types.CrossShardTokenPrivacyMetaData, *RPCError) {
	property, ok := tokenParamsRaw["TokenID"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token ID, Params %+v ", tokenParamsRaw))
	}
	tokenName, ok := tokenParamsRaw["TokenName"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Name, Params %+v ", tokenParamsRaw))
	}
	tokenSymbol, ok := tokenParamsRaw["TokenSymbol"].(string)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Symbol, Params %+v ", tokenParamsRaw))
	}
	tokenTxType, ok := tokenParamsRaw["TokenTxType"].(float64)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Tx Type, Params %+v ", tokenParamsRaw))
	}
	tokenAmount, ok := tokenParamsRaw["TokenAmount"].(float64)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Amout, Params %+v ", tokenParamsRaw))
	}
	tokenFee, ok := tokenParamsRaw["TokenFee"].(float64)
	if !ok {
		return nil, nil, nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Token Fee, Params %+v ", tokenParamsRaw))
	}
	if tokenTxType == transaction.CustomTokenInit {
		tokenFee = 0
	}
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID:     property,
		PropertyName:   tokenName,
		PropertySymbol: tokenSymbol,
		TokenTxType:    int(tokenTxType),
		Amount:         uint64(tokenAmount),
		TokenInput:     nil,
		Fee:            uint64(tokenFee),
	}

	maxDefragmentQuantity := 32

	// get list custom token
	switch tokenParams.TokenTxType {
	case transaction.CustomTokenTransfer:
		{
			tokenID, err := common.Hash{}.NewHashFromStr(tokenParams.PropertyID)
			if err != nil {
				return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
			}
			shardBestState, _ := txService.BlockChain.GetClonedAShardBestState(shardIDSender)
			isExisted := statedb.PrivacyTokenIDExisted(shardBestState.GetCopiedTransactionStateDB(), *tokenID)
			if !isExisted {
				var isBridgeToken bool
				_, allBridgeTokens, err := txService.BlockChain.GetAllBridgeTokens()
				if err != nil {
					return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
				}
				for _, bridgeToken := range allBridgeTokens {
					if bridgeToken.TokenID.IsEqual(tokenID) {
						isBridgeToken = true
						break
					}
				}
				if !isBridgeToken {
					// totally invalid token
					return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New("Invalid Token ID"))
				}
				//return nil, nil, nil, NewRPCError(BuildPrivacyTokenParamError, err)
			}
			outputTokens, err := txService.BlockChain.GetListOutputCoinsByKeyset(senderKeySet, shardIDSender, tokenID)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			outputTokens, err = txService.filterMemPoolOutcoinsToSpent(outputTokens)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}

			candidateOutputTokens, amount := txService.calculateOutputCoinsByMinValue(outputTokens, 10000*1e9, maxDefragmentQuantity)
			if len(candidateOutputTokens) == 0 {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			intputToken := transaction.ConvertOutputCoinToInputCoin(candidateOutputTokens)
			tokenParams.TokenInput = intputToken
			tokenParams.Receiver = []*privacy.PaymentInfo{{
				PaymentAddress: senderKeySet.PaymentAddress,
				Amount:         amount,
			}}
		}
	default:
		{
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, nil)
		}
	}
	return tokenParams, nil, nil, nil
}
