package rpcservice

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy/key"

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
	TxMemPool    MempoolInterface
}

type MempoolInterface interface {
	ValidateSerialNumberHashH(serialNumber []byte) error
	MaybeAcceptTransaction(tx metadata.Transaction, beaconHeight int64) (*common.Hash, *mempool.TxDesc, error)
	GetTx(txHash *common.Hash) (metadata.Transaction, error)
	GetClonedPoolCandidate() map[common.Hash]string
	ListTxs() []string
	RemoveTx(txs []metadata.Transaction, isInBlock bool)
	TriggerCRemoveTxs(tx metadata.Transaction)
	MarkForwardedTransaction(txHash common.Hash)
	MaxFee() uint64
	ListTxsDetail() []metadata.Transaction
	Count() int
	Size() uint64
	SendTransactionToBlockGen()
}

type TxInfo struct {
	BlockHash common.Hash
	Tx        metadata.Transaction
	TxHash    string
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
func (txService TxService) chooseBestOutCoinsToSpent(outCoins []coin.PlainCoin, amount uint64) (resultOutputCoins []coin.PlainCoin, remainOutputCoins []coin.PlainCoin, totalResultOutputCoinAmount uint64, err error) {
	resultOutputCoins = make([]coin.PlainCoin, 0)
	remainOutputCoins = make([]coin.PlainCoin, 0)
	totalResultOutputCoinAmount = uint64(0)

	// either take the smallest coins, or a single largest one
	var outCoinOverLimit coin.PlainCoin
	outCoinsUnderLimit := make([]coin.PlainCoin, 0)
	for _, outCoin := range outCoins {
		if outCoin.GetValue() < amount {
			outCoinsUnderLimit = append(outCoinsUnderLimit, outCoin)
		} else if outCoinOverLimit == nil {
			outCoinOverLimit = outCoin
		} else if outCoinOverLimit.GetValue() > outCoin.GetValue() {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		} else {
			remainOutputCoins = append(remainOutputCoins, outCoinOverLimit)
			outCoinOverLimit = outCoin
		}
	}
	sort.Slice(outCoinsUnderLimit, func(i, j int) bool {
		return outCoinsUnderLimit[i].GetValue() < outCoinsUnderLimit[j].GetValue()
	})
	for _, outCoin := range outCoinsUnderLimit {
		if totalResultOutputCoinAmount < amount {
			totalResultOutputCoinAmount += outCoin.GetValue()
			resultOutputCoins = append(resultOutputCoins, outCoin)
		} else {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	if outCoinOverLimit != nil && (outCoinOverLimit.GetValue() > 2*amount || totalResultOutputCoinAmount < amount) {
		remainOutputCoins = append(remainOutputCoins, resultOutputCoins...)
		resultOutputCoins = []coin.PlainCoin{outCoinOverLimit}
		totalResultOutputCoinAmount = outCoinOverLimit.GetValue()
	} else if outCoinOverLimit != nil {
		remainOutputCoins = append(remainOutputCoins, outCoinOverLimit)
	}
	if totalResultOutputCoinAmount < amount {
		return resultOutputCoins, remainOutputCoins, totalResultOutputCoinAmount, errors.New("Not enough coin")
	} else {
		return resultOutputCoins, remainOutputCoins, totalResultOutputCoinAmount, nil
	}
}

func (txService TxService) filterMemPoolOutcoinsToSpent(coins []coin.PlainCoin) ([]coin.PlainCoin, error) {
	remainOutputCoins := make([]coin.PlainCoin, 0)
	for _, outCoin := range coins {
		if txService.TxMemPool.ValidateSerialNumberHashH(outCoin.GetKeyImage().ToBytesS()) == nil {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	return remainOutputCoins, nil
}

func (txService TxService) chooseCoinsTokenVer1ByKeySet(keySet *incognitokey.KeySet, tokenID *common.Hash,
	numBlock uint64, shardIDSender byte) ([]coin.PlainCoin, []*key.PaymentInfo, *RPCError) {
	// estimate fee according to 8 recent block
	if numBlock == 0 {
		numBlock = 1000
	}
	// get list outputcoins tx
	plainCoins, _, err := txService.BlockChain.GetListDecryptedOutputCoinsVer1ByKeyset(keySet, shardIDSender, tokenID)
	if err != nil {
		return nil, nil, NewRPCError(GetOutputCoinsVer1Error, err)
	}
	// remove out coin in mem pool
	plainCoins, err = txService.filterMemPoolOutcoinsToSpent(plainCoins)
	if err != nil {
		return nil, nil, NewRPCError(GetOutputCoinsVer1Error, err)
	}
	if len(plainCoins) == 0 {
		return nil, nil, NewRPCError(GetOutputCoinsVer1Error, errors.New("Have switched all token ver 1, there is no token ver 1 left"))
	}
	paymentInfos := coin.CreatePaymentInfosFromPlainCoinsAndAddress(plainCoins, keySet.PaymentAddress, []byte{})
	return plainCoins, paymentInfos, nil
}

// chooseOutsCoinByKeyset returns list of input coins native token to spent
func (txService TxService) chooseOutsCoinVer2ByKeyset(
	paymentInfos []*privacy.PaymentInfo,
	unitFeeNativeToken int64, numBlock uint64, keySet *incognitokey.KeySet, shardIDSender byte,
	hasPrivacy bool,
	metadataParam metadata.Metadata,
	privacyCustomTokenParams *transaction.TokenParam,
) ([]coin.PlainCoin, uint64, *RPCError) {
	// estimate fee according to 8 recent block
	if numBlock == 0 {
		numBlock = 1000
	}
	// calculate total amount to send
	totalAmmount := uint64(0)
	for _, receiver := range paymentInfos {
		totalAmmount += receiver.Amount
	}
	// get list outputcoins tx
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	plainCoins, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(keySet, shardIDSender, prvCoinID, false)
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	// remove out coin in mem pool
	plainCoins, err = txService.filterMemPoolOutcoinsToSpent(plainCoins)
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	if len(plainCoins) == 0 && totalAmmount > 0 {
		return nil, 0, NewRPCError(GetOutputCoinError, errors.New("not enough output coin"))
	}

	// Use Knapsack to get candiate output coin
	candidatePlainCoins, outCoins, candidateOutputCoinAmount, err := txService.chooseBestOutCoinsToSpent(plainCoins, totalAmmount)
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	// refund out put for sender
	overBalanceAmount := candidateOutputCoinAmount - totalAmmount
	if overBalanceAmount > 0 {
		// add more into output for estimate fee
		paymentInfos = append(paymentInfos, &privacy.PaymentInfo{
			PaymentAddress: keySet.PaymentAddress,
			Amount:         overBalanceAmount,
		})
	}
	// check real fee(nano PRV) per tx
	beaconState, err := txService.BlockChain.GetClonedBeaconBestState()
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	beaconHeight := beaconState.BeaconHeight
	realFee, _, _, err := txService.EstimateFee(2, unitFeeNativeToken, false, candidatePlainCoins,
		paymentInfos, shardIDSender, numBlock, hasPrivacy,
		metadataParam,
		privacyCustomTokenParams, int64(beaconHeight))
	if err != nil {
		return nil, 0, NewRPCError(RejectInvalidTxFeeError, err)
	}
	if totalAmmount == 0 && realFee == 0 {
		if metadataParam != nil {
			metadataType := metadataParam.GetType()
			switch metadataType {
			case metadata.WithDrawRewardRequestMeta:
				{
					return nil, realFee, nil
				}
			}
			return nil, realFee, NewRPCError(RejectInvalidTxFeeError, fmt.Errorf("totalAmmount: %+v, realFee: %+v", totalAmmount, realFee))
		}
		if privacyCustomTokenParams != nil {
			// for privacy token
			return nil, 0, nil
		}
	}
	needToPayFee := int64((totalAmmount + realFee) - candidateOutputCoinAmount)
	// if not enough to pay fee
	if needToPayFee > 0 {
		if len(outCoins) > 0 {
			candidateOutputCoinsForFee, _, _, err1 := txService.chooseBestOutCoinsToSpent(outCoins, uint64(needToPayFee))
			if err1 != nil {
				return nil, 0, NewRPCError(GetOutputCoinError, err1)
			}
			candidatePlainCoins = append(candidatePlainCoins, candidateOutputCoinsForFee...)
		}
	}
	return candidatePlainCoins, realFee, nil
}

func (txService TxService) chooseCoinsVer1ByKeyset(keySet *incognitokey.KeySet, unitFeeNativeToken int64,
	numBlock uint64, shardIDSender byte, metadataParam metadata.Metadata) ([]coin.PlainCoin, []*key.PaymentInfo, uint64, *RPCError) {
	// estimate fee according to 8 recent block
	if numBlock == 0 {
		numBlock = 1000
	}
	// get list outputcoins tx
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	plainCoins, _, err := txService.BlockChain.GetListDecryptedOutputCoinsVer1ByKeyset(keySet, shardIDSender, prvCoinID)
	if err != nil {
		return nil, nil, 0, NewRPCError(GetOutputCoinsVer1Error, err)
	}
	// remove out coin in mem pool
	plainCoins, err = txService.filterMemPoolOutcoinsToSpent(plainCoins)
	if err != nil {
		return nil, nil, 0, NewRPCError(GetOutputCoinsVer1Error, err)
	}
	if len(plainCoins) == 0 {
		return nil, nil, 0, NewRPCError(GetOutputCoinsVer1Error, errors.New("Have switched all coins ver 1, there is no coins ver 1 left"))
	}
	// check real fee(nano PRV) per tx
	beaconState, err := txService.BlockChain.GetClonedBeaconBestState()
	if err != nil {
		return nil, nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	beaconHeight := beaconState.BeaconHeight
	paymentInfos := coin.CreatePaymentInfosFromPlainCoinsAndAddress(plainCoins, keySet.PaymentAddress, []byte{})
	realFee, _, _, err := txService.EstimateFee(2, unitFeeNativeToken, false, plainCoins,
		paymentInfos, shardIDSender, numBlock, false,
		metadataParam,
		nil, int64(beaconHeight))
	if err != nil {
		return nil, nil, 0, NewRPCError(RejectInvalidTxFeeError, err)
	}
	if paymentInfos[0].Amount < realFee {
		return nil, nil, 0, NewRPCError(RejectInvalidTxFeeTooLargeError, err)
	}
	paymentInfos[0].Amount -= realFee

	// TODO From privacy team: I don't really understand this so comment it
	//if realFee == 0 {
	//	if metadataParam != nil {
	//		metadataType := metadataParam.GetType()
	//		switch metadataType {
	//		case metadata.WithDrawRewardRequestMeta:
	//			{
	//				return nil, nil, realFee, nil
	//			}
	//		}
	//		return nil, nil, realFee, NewRPCError(RejectInvalidTxFeeError, fmt.Errorf("totalAmmount: %+v, realFee: %+v", totalAmmount, realFee))
	//	}
	//	if privacyCustomTokenParams != nil {
	//		// for privacy token
	//		return nil, nil, 0, nil
	//	}
	//}

	return plainCoins, paymentInfos, realFee, nil
}

// chooseOutsCoinByKeyset returns list of input coins native token to spent
func (txService TxService) chooseOutsCoinByKeyset(
	paymentInfos []*privacy.PaymentInfo,
	unitFeeNativeToken int64, numBlock uint64, keySet *incognitokey.KeySet, shardIDSender byte,
	hasPrivacy bool,
	metadataParam metadata.Metadata,
	privacyCustomTokenParams *transaction.TokenParam,
) ([]coin.PlainCoin, uint64, *RPCError) {
	// estimate fee according to 8 recent block
	if numBlock == 0 {
		numBlock = 1000
	}
	// calculate total amount to send
	totalAmmount := uint64(0)
	for _, receiver := range paymentInfos {
		totalAmmount += receiver.Amount
	}
	// get list outputcoins tx
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	plainCoins, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(keySet, shardIDSender, prvCoinID, true)
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	// remove out coin in mem pool
	plainCoins, err = txService.filterMemPoolOutcoinsToSpent(plainCoins)
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	if len(plainCoins) == 0 && totalAmmount > 0 {
		return nil, 0, NewRPCError(GetOutputCoinError, errors.New("not enough output coin"))
	}

	// Use Knapsack to get candiate output coin
	candidatePlainCoins, outCoins, candidateOutputCoinAmount, err := txService.chooseBestOutCoinsToSpent(plainCoins, totalAmmount)
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	// refund out put for sender
	overBalanceAmount := candidateOutputCoinAmount - totalAmmount
	if overBalanceAmount > 0 {
		// add more into output for estimate fee
		paymentInfos = append(paymentInfos, &privacy.PaymentInfo{
			PaymentAddress: keySet.PaymentAddress,
			Amount:         overBalanceAmount,
		})
	}
	// check real fee(nano PRV) per tx
	beaconState, err := txService.BlockChain.GetClonedBeaconBestState()
	if err != nil {
		return nil, 0, NewRPCError(GetOutputCoinError, err)
	}
	beaconHeight := beaconState.BeaconHeight
	ver, err := transaction.GetTxVersionFromCoins(candidatePlainCoins)
	realFee, _, _, err := txService.EstimateFee(int(ver), unitFeeNativeToken, false, candidatePlainCoins,
		paymentInfos, shardIDSender, numBlock, hasPrivacy,
		metadataParam,
		privacyCustomTokenParams, int64(beaconHeight))
	if err != nil {
		return nil, 0, NewRPCError(RejectInvalidTxFeeError, err)
	}
	if totalAmmount == 0 && realFee == 0 {
		if metadataParam != nil {
			metadataType := metadataParam.GetType()
			switch metadataType {
			case metadata.WithDrawRewardRequestMeta:
				{
					return nil, realFee, nil
				}
			}
			return nil, realFee, NewRPCError(RejectInvalidTxFeeError, fmt.Errorf("totalAmmount: %+v, realFee: %+v", totalAmmount, realFee))
		}
		if privacyCustomTokenParams != nil {
			// for privacy token
			return nil, 0, nil
		}
	}
	needToPayFee := int64((totalAmmount + realFee) - candidateOutputCoinAmount)
	// if not enough to pay fee
	if needToPayFee > 0 {
		if len(outCoins) > 0 {
			candidateOutputCoinsForFee, _, _, err1 := txService.chooseBestOutCoinsToSpent(outCoins, uint64(needToPayFee))
			if err != nil {
				return nil, 0, NewRPCError(GetOutputCoinError, err1)
			}
			candidatePlainCoins = append(candidatePlainCoins, candidateOutputCoinsForFee...)
		}
	}
	return candidatePlainCoins, realFee, nil
}

// EstimateFee - estimate fee from tx data and return real full fee, fee per kb and real tx size
// if isGetPTokenFee == true: return fee for ptoken
// if isGetPTokenFee == false: return fee for native token
func (txService TxService) EstimateFee(
	version int,
	defaultFee int64,
	isGetPTokenFee bool,
	candidatePlainCoins []coin.PlainCoin,
	paymentInfos []*privacy.PaymentInfo, shardID byte,
	numBlock uint64, hasPrivacy bool,
	metadata metadata.Metadata,
	privacyCustomTokenParams *transaction.TokenParam,
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
	estimateTxSizeInKb = transaction.EstimateTxSize(transaction.NewEstimateTxSizeParam(version, len(candidatePlainCoins), len(paymentInfos), hasPrivacy, metadata, privacyCustomTokenParams, limitFee))
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
		beaconStateDB, err := txService.BlockChain.GetBestStateBeaconFeatureStateDBByHeight(uint64(beaconHeight), txService.BlockChain.GetBeaconChainDatabase())
		limitFeePTokenTmp, err := metadata.ConvertNativeTokenToPrivacyToken(limitFee, tokenId, beaconHeight, beaconStateDB)
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

func (txService TxService) BuildConvertV1ToV2Transaction(params *bean.CreateRawTxSwitchVer1ToVer2Param) (metadata.Transaction, *RPCError) {
	Logger.log.Infof("Convert V1 to V2 Transaction Params: \n %+v", params)
	// get output coins to spend and real fee
	inputCoins, paymentInfos, realFee, err1 := txService.chooseCoinsVer1ByKeyset(
		params.SenderKeySet, params.EstimateFeeCoinPerKb, 0, params.ShardIDSender, nil)
	if err1 != nil {
		return nil, err1
	}

	// init tx
	txConvertParams := transaction.NewTxConvertVer1ToVer2InitParams(
		&params.SenderKeySet.PrivateKey,
		paymentInfos,
		inputCoins,
		realFee,
		txService.BlockChain.GetBestStateShard(params.ShardIDSender).GetCopiedTransactionStateDB(),
		nil, // use for prv coin -> nil is valid
		nil,
		params.Info,
	)
	tx := new(transaction.TxVersion2)
	if err := transaction.InitConversion(tx, txConvertParams); err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}
	return tx, nil
}

func (txService TxService) BuildRawTransaction(params *bean.CreateRawTxParam, meta metadata.Metadata) (metadata.Transaction, *RPCError) {
	Logger.log.Infof("Build Raw Transaction Params: \n %+v", params)
	// get output coins to spend and real fee
	inputCoins, realFee, err1 := txService.chooseOutsCoinByKeyset(
		params.PaymentInfos, params.EstimateFeeCoinPerKb, 0,
		params.SenderKeySet, params.ShardIDSender, params.HasPrivacyCoin,
		meta, nil)
	if err1 != nil {
		return nil, err1
	}

	txPrivacyParams := transaction.NewTxPrivacyInitParams(
		&params.SenderKeySet.PrivateKey,
		params.PaymentInfos,
		inputCoins,
		realFee,
		params.HasPrivacyCoin,
		txService.BlockChain.GetBestStateShard(params.ShardIDSender).GetCopiedTransactionStateDB(),
		nil, // use for prv coin -> nil is valid
		meta,
		params.Info,
	)
	tx, err := transaction.NewTransactionFromParams(txPrivacyParams)
	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}
	if err := tx.Init(txPrivacyParams); err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}
	return tx, nil
}

func (txService TxService) CreateRawConvertVer1ToVer2Transaction(params *bean.CreateRawTxSwitchVer1ToVer2Param) (*common.Hash, []byte, byte, *RPCError) {
	tx, err := txService.BuildConvertV1ToV2Transaction(params)
	if err != nil {
		Logger.log.Critical(err)
		return nil, nil, byte(0), NewRPCError(CreateTxDataError, err)
	}
	txBytes, errJson := json.Marshal(tx)
	if errJson != nil {
		// return hex for a new tx
		return nil, nil, byte(0), NewRPCError(CreateTxDataError, errJson)
	}
	txShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	return tx.Hash(), txBytes, txShardID, nil
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
	tx, err := transaction.NewTransactionFromJsonBytes(rawTxBytes)
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
	hash, _, err := txService.TxMemPool.MaybeAcceptTransaction(tx, beaconHeigh)
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
	txMsg.(*wire.MessageTx).Transaction = tx
	return txMsg, hash, tx.GetSenderAddrLastByte(), nil
}

func (txService TxService) BuildTokenParam(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.TokenParam, *RPCError) {
	var privacyTokenParam *transaction.TokenParam
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

func (txService TxService) BuildPrivacyCustomTokenParam(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.TokenParam, map[common.Hash]transaction.TransactionToken, map[common.Hash]blockchain.CrossShardTokenPrivacyMetaData, *RPCError) {
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
	tokenParams := &transaction.TokenParam{
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
			outputTokens, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(senderKeySet, shardIDSender, tokenID, true)
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
			tokenParams.TokenInput = candidateOutputTokens
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

func (txService TxService) BuildTokenParamV2(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.TokenParam, *RPCError) {
	var privacyTokenParam *transaction.TokenParam
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

func (txService TxService) BuildPrivacyCustomTokenParamV2(tokenParamsRaw map[string]interface{}, senderKeySet *incognitokey.KeySet, shardIDSender byte) (*transaction.TokenParam, map[common.Hash]transaction.TransactionToken, map[common.Hash]blockchain.CrossShardTokenPrivacyMetaData, *RPCError) {
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
	tokenParams := &transaction.TokenParam{
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
			// We should get all one time address of user from start 0 to current
			// This code does not used on production anyway (only for testing purpose) so it should not affect performance
			outputTokens, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(senderKeySet, shardIDSender, tokenID, true)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			outputTokens, err = txService.filterMemPoolOutcoinsToSpent(outputTokens)
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			candidateTokens, _, _, err := txService.chooseBestOutCoinsToSpent(outputTokens, uint64(voutsAmount))
			if err != nil {
				return nil, nil, nil, NewRPCError(GetOutputCoinError, err)
			}
			tokenParams.TokenInput = candidateTokens
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

func (txService TxService) BuildRawConvertVer1ToVer2Token(params *bean.CreateRawTxTokenSwitchVer1ToVer2Param) (*common.Hash, []byte, byte, *RPCError) {
	Logger.log.Infof("Convert V1 to V2 Transaction Token Params: \n %+v", params)

	// get token coins to convert
	inputTokenCoins, paymentInfos, err1 := txService.chooseCoinsTokenVer1ByKeySet(
		params.SenderKeySet, params.TokenID, 0, params.ShardIDSender)
	if err1 != nil {
		return nil, nil, 0, err1
	}

	/******* choose output native coins(PRV) for fee *****/
	// TODO: Remember this start height 0
	paymentFeeInfo := []*privacy.PaymentInfo{}
	inputCoins, realFeePRV, errFeeCoins := txService.chooseOutsCoinVer2ByKeyset(paymentFeeInfo,
		params.EstimateFeeCoinPerKb, 0, params.SenderKeySet,
		params.ShardIDSender, false, nil, nil)
	if errFeeCoins != nil {
		return nil, nil, 0, errFeeCoins
	}

	fmt.Println("BuildRawConvertVer1ToVer2Token")
	fmt.Println("Real fee", realFeePRV)
	fmt.Println("Coins:", inputCoins)
	for i := 0; i < len(inputCoins); i += 1 {
		fmt.Println("Coin[", i, "] ver:", inputCoins[i].GetVersion())
		fmt.Println("Coin[", i, "] value:", inputCoins[i].GetValue())
	}
	fmt.Println("FeePaymentInfos", paymentFeeInfo)
	for i := 0; i < len(paymentFeeInfo); i += 1 {
		fmt.Println("paymentFeeInfo[", i, "] ver:", paymentFeeInfo[i].Amount)
	}
	fmt.Println("Token:")
	fmt.Println("inputTokenCoins: length =", len(inputTokenCoins))
	for i := 0; i < len(inputTokenCoins); i += 1 {
		fmt.Println("TokenCoin[", i, "] ver:", inputTokenCoins[i].GetVersion())
		fmt.Println("TokenCoin[", i, "] value:", inputTokenCoins[i].GetValue())
	}
	fmt.Println("paymentInfos: length =", len(paymentInfos))
	for i := 0; i < len(paymentInfos); i += 1 {
		fmt.Println("paymentInfos[", i, "] ver:", paymentInfos[i].Amount)
	}

	fmt.Println("Done logging=============")

	/******* END GET output coins native coins(PRV), which is used to create tx *****/
	beaconView := txService.BlockChain.BeaconChain.GetFinalViewState()

	// init tx
	txTokenConvertParams := transaction.NewTxTokenConvertVer1ToVer2InitParams(
		&params.SenderKeySet.PrivateKey,
		inputCoins,
		paymentFeeInfo,
		inputTokenCoins,
		paymentInfos,
		realFeePRV,
		txService.BlockChain.GetBestStateShard(params.ShardIDSender).GetCopiedTransactionStateDB(),
		beaconView.GetBeaconFeatureStateDB(),
		params.TokenID,
		nil,
		params.Info,
	)
	tx := new(transaction.TxTokenVersion2)
	if err := transaction.InitTokenConversion(tx, txTokenConvertParams); err != nil {
		return nil, nil, 0, NewRPCError(CreateTxDataError, err)
	}

	txBytes, err := json.Marshal(tx)
	if err != nil {
		// return hex for a new tx
		return nil, nil, byte(0), NewRPCError(CreateTxDataError, err)
	}
	txShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())

	return tx.Hash(), txBytes, txShardID, nil
}

// BuildRawCustomTokenTransaction ...
func (txService TxService) BuildRawPrivacyCustomTokenTransaction(params interface{}, metaData metadata.Metadata) (transaction.TransactionToken, *RPCError) {
	txParam, errParam := bean.NewCreateRawPrivacyTokenTxParam(params)
	if errParam != nil {
		return nil, NewRPCError(RPCInvalidParamsError, errParam)
	}
	tokenParamsRaw := txParam.TokenParamsRaw
	tokenParams, err := txService.BuildTokenParam(tokenParamsRaw, txParam.SenderKeySet, txParam.ShardIDSender)
	if err != nil {
		return nil, err
	}

	if tokenParams == nil {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("can not build token params for request"))
	}
	/******* START choose output native coins(PRV), which is used to create tx *****/
	var inputCoins []coin.PlainCoin
	realFeePRV := uint64(0)
	inputCoins, realFeePRV, err = txService.chooseOutsCoinByKeyset(txParam.PaymentInfos,
		txParam.EstimateFeeCoinPerKb, 0, txParam.SenderKeySet,
		txParam.ShardIDSender, txParam.HasPrivacyCoin, nil, tokenParams)

	if err != nil {
		return nil, err
	}

	if len(txParam.PaymentInfos) == 0 && realFeePRV == 0 {
		txParam.HasPrivacyCoin = false
	}
	/******* END GET output coins native coins(PRV), which is used to create tx *****/
	beaconView := txService.BlockChain.BeaconChain.GetFinalViewState()

	txTokenParams := transaction.NewTxTokenParams(&txParam.SenderKeySet.PrivateKey,
		txParam.PaymentInfos,
		inputCoins,
		realFeePRV,
		tokenParams,
		txService.BlockChain.GetBestStateShard(txParam.ShardIDSender).GetCopiedTransactionStateDB(),
		metaData,
		txParam.HasPrivacyCoin,
		txParam.HasPrivacyToken,
		txParam.ShardIDSender, txParam.Info,
		beaconView.GetBeaconFeatureStateDB())

	tx, errTx := transaction.NewTransactionTokenFromParams(txTokenParams)
	if errTx != nil {
		Logger.log.Errorf("Cannot create new transaction token from params, err %v", err)
		return nil, NewRPCError(CreateTxDataError, errTx)
	}
	errTx = tx.Init(txTokenParams)
	if errTx != nil {
		return nil, NewRPCError(CreateTxDataError, errTx)
	}

	return tx, nil
}

// BuildRawCustomTokenTransactionV2 ...
func (txService TxService) BuildRawPrivacyCustomTokenTransactionV2(params interface{}, metaData metadata.Metadata) (transaction.TransactionToken, *RPCError) {
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
	var inputCoins []coin.PlainCoin
	realFeePRV := uint64(0)
	inputCoins, realFeePRV, err = txService.chooseOutsCoinByKeyset(txParam.PaymentInfos,
		txParam.EstimateFeeCoinPerKb, 0, txParam.SenderKeySet,
		txParam.ShardIDSender, txParam.HasPrivacyCoin, nil, tokenParams)
	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if len(txParam.PaymentInfos) == 0 && realFeePRV == 0 {
		txParam.HasPrivacyCoin = false
	}
	/******* END GET output coins native coins(PRV), which is used to create tx *****/
	txTokenParams := transaction.NewTxTokenParams(&txParam.SenderKeySet.PrivateKey,
		txParam.PaymentInfos,
		inputCoins,
		realFeePRV,
		tokenParams,
		txService.BlockChain.GetBestStateShard(txParam.ShardIDSender).GetCopiedTransactionStateDB(),
		metaData,
		txParam.HasPrivacyCoin,
		txParam.HasPrivacyToken,
		txParam.ShardIDSender, txParam.Info,
		txService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB())
	tx, err := transaction.NewTransactionTokenFromParams(txTokenParams)
	if err != nil {
		Logger.log.Errorf("Cannot create new transaction token from params, err %v", err)
		return nil, NewRPCError(CreateTxDataError, err)
	}
	err = tx.Init(txTokenParams)
	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}
	return tx.(transaction.TransactionToken), nil
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

func (txService TxService) getTxsByHashs(txHashs []common.Hash, ch chan []TxInfo, tokenID common.Hash, pubKey []byte) {
	txInfos := []TxInfo{}
	for _, txHash := range txHashs {
		_, blockHash, _, _, txDetail, _ := txService.BlockChain.GetTransactionByHash(txHash)
		// filter by tokenID
		if bytes.Equal(tokenID.Bytes(), common.PRVCoinID.Bytes()) {
			// case 1: tokenID is PRVID, include normal Tx and privacyTx that have native output coins for public key
			if txDetail.GetType() == common.TxCustomTokenPrivacyType {
				txPToken, ok := txDetail.(transaction.TransactionToken)
				if !ok || txPToken == nil {
					continue
				}
				nativeProof := txPToken.GetTxBase().GetProof()
				if nativeProof == nil {
					continue
				}
				nativeOutputCoins := nativeProof.GetOutputCoins()
				hasNativeOutCoin := false
				for _, out := range nativeOutputCoins {
					if bytes.Equal(out.GetPublicKey().ToBytesS(), pubKey) {
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

			txPToken, ok := txDetail.(transaction.TransactionToken)
			if !ok || txPToken == nil {
				continue
			}
			if !bytes.Equal(txPToken.GetTokenID().Bytes(), tokenID.Bytes()) {
				continue
			}

			pTokenProof := txPToken.GetTxNormal().GetProof()
			if pTokenProof == nil {
				continue
			}
			pTokenOutputCoins := pTokenProof.GetOutputCoins()
			hasPTokenOutCoin := false
			for _, out := range pTokenOutputCoins {
				if bytes.Equal(out.GetPublicKey().ToBytesS(), pubKey) {
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
			Tx:        txDetail,
			TxHash: txHash.String(),
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
		chunk := allTxHashs[start:end]
		go txService.getTxsByHashs(chunk, ch, tokenIDHash, keySet.PaymentAddress.Pk)
	}

	txInfos := []TxInfo{}
	for i := 0; i < actualChunksNum; i++ {
		chunkedTxInfos := <-ch
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
	for _, info := range pagingTxInfos{
		txDetail, err := txService.GetTransactionByHash(info.TxHash)
		if err != nil {
			Logger.log.Error("cannot get tx detail from hash %v. Error %v", info.TxHash, err)
			continue
		}

		receivedAmounts, err := txService.DecryptOutputCoinByKeyByTransaction(&keySet, info.TxHash)
		if err != nil {
			Logger.log.Error("cannot decrypt output coins from hash %v. Error %v", info.TxHash, err)
			continue
		}

		inputSerialNumbers, err := txService.GetInputSerialNumberByTransaction(info.TxHash)
		if err != nil {
			Logger.log.Error("cannot get serial numbers from hash %v. Error %v", info.TxHash, err)
			continue
		}

		item := jsonresult.ReceivedTransactionV2{
			TxDetail:        txDetail,
			FromShardID:     txDetail.ShardID,
			ReceivedAmounts: receivedAmounts,
			InputSerialNumbers: inputSerialNumbers,
		}

		result.ReceivedTransactions = append(result.ReceivedTransactions, item)
		sort.Slice(result.ReceivedTransactions, func(i, j int) bool {
			return result.ReceivedTransactions[i].TxDetail.LockTime > result.ReceivedTransactions[j].TxDetail.LockTime
		})

	}
	return result, txNum, nil
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

		// Get List privacy custom token balance require user to input their secret key, so this is for testing purpose
		// So we need to query startHeight from 0
		outcoints, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.KeySet, shardIDSender, &tokenID, true)
		if err != nil {
			Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance result: %+v, err: %+v", nil, err)
			return jsonresult.ListCustomTokenBalance{}, NewRPCError(GetListPrivacyCustomTokenBalanceError, err)
		}
		for _, out := range outcoints {
			balance += out.GetValue()
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
		outcoints, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.KeySet, shardIDSender, tokenID, true)
		if err != nil {
			return jsonresult.ListCustomTokenBalance{}, NewRPCError(UnexpectedError, err)
		}
		for _, out := range outcoints {
			balance += out.GetValue()
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
		// Get balance privacy custom token should use 0 as start height, because we do not use this in production anyway
		// Get from 0 to get all the coins starting from begin to end
		outcoins, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.KeySet, shardIDSender, tokenID, true)
		if err != nil {
			Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v, err: %+v", nil, err)
			return uint64(0), NewRPCError(UnexpectedError, err)
		}
		for _, out := range outcoins {
			totalValue += out.GetValue()
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
						outcoints, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.KeySet, shardIDSender, tempTokenID, true)
						if err != nil {
							Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v, err: %+v", nil, err)
							return uint64(0), NewRPCError(UnexpectedError, err)
						}
						for _, out := range outcoints {
							totalValue += out.GetValue()
						}
					}
				}
			}
		}
	}

	return totalValue, nil
}

func (txService TxService) PrivacyCustomTokenDetail(tokenIDStr string) ([]common.Hash, *transaction.TxTokenData, error) {
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
	tokenData := &transaction.TxTokenData{}
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

	usableCoin := make([]coin.PlainCoin, len(outputs))
	for index, item := range outputs {
		out, err1 := jsonresult.NewOutcoinFromInterface(item)
		if err1 != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("outputs is invalid", out)))
		}
		valueBNTmp := big.Int{}
		valueBNTmp.SetString(out.Value, 10)

		outputCoin := new(coin.PlainCoinV1).Init()
		outputCoin.SetValue(valueBNTmp.Uint64())

		RandomnessInBytes, _, err := base58.Base58Check{}.Decode(out.Randomness)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("randomness output is invalid", out.Randomness)))
		}
		outputCoin.SetRandomness(new(privacy.Scalar).FromBytesS(RandomnessInBytes))

		SNDerivatorInBytes, _, err := base58.Base58Check{}.Decode(out.SNDerivator)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("snderivator output is invalid", out.SNDerivator)))
		}
		outputCoin.SetSNDerivator(new(privacy.Scalar).FromBytesS(SNDerivatorInBytes))

		CoinCommitmentBytes, _, err := base58.Base58Check{}.Decode(out.Commitment)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("coin commitment output is invalid", out.Commitment)))
		}
		CoinCommitment, err := new(privacy.Point).FromBytesS(CoinCommitmentBytes)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("coin commitment output is invalid", CoinCommitmentBytes)))
		}
		outputCoin.SetCommitment(CoinCommitment)

		PublicKeyBytes, _, err := base58.Base58Check{}.Decode(out.PublicKey)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("public key output is invalid", out.PublicKey)))
		}
		PublicKey, err := new(privacy.Point).FromBytesS(PublicKeyBytes)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("public key output is invalid", PublicKeyBytes)))
		}
		outputCoin.SetPublicKey(PublicKey)

		InfoBytes, _, err := base58.Base58Check{}.Decode(out.Info)
		if err != nil {
			return nil, nil, nil, NewRPCError(RPCInvalidParamsError, errors.New(fmt.Sprint("Info output is invalid", out.Info)))
		}
		outputCoin.SetInfo(InfoBytes)
		usableCoin[index] = outputCoin
	}
	commitmentIndexs, myCommitmentIndexs, commitments := txService.BlockChain.RandomCommitmentsProcess(usableCoin, 0, shardIDSender, tokenID)
	return commitmentIndexs, myCommitmentIndexs, commitments, nil
}

func (txService TxService) RandomCommitmentsAndPublicKeys(paymentAddressStr string, numOutputs int, tokenID *common.Hash) ([]uint64, [][]byte, [][]byte, [][]byte, *RPCError) {
	_, shardIDSender, err := GetKeySetFromPaymentAddressParam(paymentAddressStr)
	if err != nil {
		Logger.log.Debugf("handleRandomCommitmentsAndPublicKeys result: %+v, err: %+v", nil, err)
		return nil, nil, nil, nil, NewRPCError(UnexpectedError, err)
	}

	indices, publicKeys, commitments, assetTags, err := txService.BlockChain.RandomCommitmentsAndPublicKeysProcess(numOutputs, shardIDSender, tokenID)
	if err != nil {
		return nil, nil, nil, nil, NewRPCError(UnexpectedError, err)
	}
	return indices, publicKeys, commitments, assetTags, nil
}

func (txService TxService) SendRawPrivacyCustomTokenTransaction(base58CheckData string) (wire.Message, transaction.TransactionToken, *RPCError) {
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckData)
	if err != nil {
		Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, nil, NewRPCError(RPCInvalidParamsError, err)
	}
	tx, err := transaction.NewTransactionTokenFromJsonBytes(rawTxBytes)
	if err != nil {
		Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, nil, NewRPCError(RPCInvalidParamsError, err)
	}
	beaconHeigh := int64(-1)
	beaconBestState, err := txService.BlockChain.GetClonedBeaconBestState()
	if err == nil {
		beaconHeigh = int64(beaconBestState.BeaconHeight)
	}

	hash, _, err := txService.TxMemPool.MaybeAcceptTransaction(tx, beaconHeigh)
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

	txMsg.(*wire.MessageTxPrivacyToken).Transaction = tx

	return txMsg, tx, nil
}

func (txService TxService) BuildRawDefragmentAccountTransaction(params interface{}, meta metadata.Metadata) (metadata.Transaction, *RPCError) {
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

	// Defragment account need to get all coins, so start from 0
	plainCoins, err := txService.BlockChain.TryGetAllOutputCoinsByKeyset(senderKeySet, shardIDSender, prvCoinID, true)
	if err != nil {
		return nil, NewRPCError(GetOutputCoinError, err)
	}
	// remove out coin in mem pool
	plainCoins, err = txService.filterMemPoolOutcoinsToSpent(plainCoins)
	if err != nil {
		return nil, NewRPCError(GetOutputCoinError, err)
	}
	plainCoins, amount := txService.calculateOutputCoinsByMinValue(plainCoins, maxVal, maxDefragmentQuantity)
	if len(plainCoins) == 0 {
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
	ver, err := transaction.GetTxVersionFromCoins(plainCoins)
	if err != nil {
		return nil, NewRPCError(GetOutputCoinError, err)
	}
	realFee, _, _, _ := txService.EstimateFee(int(ver),
		estimateFeeCoinPerKb, isGetPTokenFee, plainCoins, paymentInfos, shardIDSender, 8, hasPrivacyCoin, nil, nil, int64(beaconHeight))
	if len(plainCoins) == 0 {
		realFee = 0
	}
	if uint64(amount) < realFee {
		return nil, NewRPCError(GetOutputCoinError, err)
	}
	paymentInfo.Amount = uint64(amount) - realFee
	/******* END GET output native coins(PRV), which is used to create tx *****/
	// START create tx
	// missing flag for privacy
	// false by default

	txPrivacyParams := transaction.NewTxPrivacyInitParams(&senderKeySet.PrivateKey,
		paymentInfos,
		plainCoins,
		realFee,
		hasPrivacyCoin,
		txService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB(),
		nil, // use for prv coin -> nil is valid
		meta, nil,
	)
	tx, err := transaction.NewTransactionFromParams(txPrivacyParams)
	if err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}
	if err = tx.Init(txPrivacyParams); err != nil {
		return nil, NewRPCError(CreateTxDataError, err)
	}
	return tx, nil
}

//calculateOutputCoinsByMinValue
func (txService TxService) calculateOutputCoinsByMinValue(outCoins []coin.PlainCoin, maxVal uint64, maxDefragmentQuantityTemp int) ([]coin.PlainCoin, uint64) {
	outCoinsTmp := make([]coin.PlainCoin, 0)
	amount := uint64(0)
	for _, outCoin := range outCoins {
		if outCoin.GetValue() <= maxVal {
			outCoinsTmp = append(outCoinsTmp, outCoin)
			amount += outCoin.GetValue()
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
	}

	txMsg.(*wire.MessageTx).Transaction = &tx

	return txMsg, hash, tx.PubKeyLastByteSender, nil
}*/

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

	result := jsonresult.ListReceivedTransaction{
		ReceivedTransactions: []jsonresult.ReceivedTransaction{},
	}

	for shardID, txHashs := range listTxsHash {
		for _, txHash := range txHashs {
			txDetail, err := txService.GetTransactionByHash(txHash.String())
			if err != nil {
				Logger.log.Error("cannot get tx detail from hash %v. Error %v", txHash.String(), err)
				continue
			}

			receivedAmounts, err := txService.DecryptOutputCoinByKeyByTransaction(&keySet, txHash.String())
			if err != nil {
				Logger.log.Error("cannot decrypt output coins from hash %v. Error %v", txHash.String(), err)
				continue
			}

			item := jsonresult.ReceivedTransaction{
				TxDetail:        txDetail,
				FromShardID:     shardID,
				ReceivedAmounts: receivedAmounts,
			}

			result.ReceivedTransactions = append(result.ReceivedTransactions, item)
			sort.Slice(result.ReceivedTransactions, func(i, j int) bool {
				return result.ReceivedTransactions[i].TxDetail.LockTime > result.ReceivedTransactions[j].TxDetail.LockTime
			})
		}
	}
	return &result, nil
}

func (txService TxService) GetInputSerialNumberByTransaction(txHashStr string) (map[string][]string, *RPCError) {
	txHash, err := common.Hash{}.NewHashFromStr(txHashStr)
	if err != nil {
		return nil, NewRPCError(RPCInvalidParamsError, errors.New("tx hash is invalid"))
	}
	_, _, _, _, tx, err := txService.BlockChain.GetTransactionByHash(*txHash)
	if err != nil {
		// maybe tx is still in tx mempool -> check mempool
		var errM error
		tx, errM = txService.TxMemPool.GetTx(txHash)
		if errM != nil {
			return nil, NewRPCError(TxNotExistedInMemAndBLockError, errors.New("Tx is not existed in block or mempool"))
		}
	}
	results := make(map[string][]string)

	switch tx.GetType() {
	case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType, common.TxConversionType:
		{
			inputSerialNumbers, err := GetInputSerialnumber(tx.GetProof().GetInputCoins())
			if err != nil {
				return nil, NewRPCError(UnexpectedError, errors.New("cannot get input coins"))
			}
			results[common.PRVCoinID.String()] = inputSerialNumbers
		}
	case common.TxCustomTokenPrivacyType, common.TxTokenConversionType:
		{
			txToken, ok := tx.(transaction.TransactionToken)
			if !ok{
				return nil, NewRPCError(UnexpectedError, errors.New("tx type is invalid - cast failed"))
			}
			inputSerialNumbers, err := GetInputSerialnumber(txToken.GetTxBase().GetProof().GetInputCoins())
			if err != nil {
				return nil, NewRPCError(UnexpectedError, errors.New("cannot get input coins"))
			}
			results[common.PRVCoinID.String()] = inputSerialNumbers

			tempTx := txToken.GetTxNormal()
			tokenData := tx.(transaction.TransactionToken).GetTxTokenData()
			inputTokenSerialNumbers, err := GetInputSerialnumber(tempTx.GetProof().GetInputCoins())
			if err != nil {
				return nil, NewRPCError(UnexpectedError, errors.New("cannot get input tokens"))
			}
			results[tokenData.PropertyID.String()] = inputTokenSerialNumbers
		}
	default:
		{
			return nil, NewRPCError(UnexpectedError, errors.New("tx type is invalid"))
		}
	}
	return results, nil

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
	case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType, common.TxConversionType:
		{
			prvOutputs, _ := txService.DecryptOutputCoinByKey(tx.GetProof().GetOutputCoins(), keyParam)
			if len(prvOutputs) > 0 {
				totalPrvValue := uint64(0)
				for _, output := range prvOutputs {
					totalPrvValue += output.GetValue()
				}
				results[common.PRVCoinID.String()] = totalPrvValue
			}
		}
	case common.TxCustomTokenPrivacyType, common.TxTokenConversionType:
		{
			tempTx := tx.(transaction.TransactionToken)
			outputOfPrv := tempTx.GetTxBase().GetProof().GetOutputCoins()
			if len(outputOfPrv) > 0 {
				prvOutputs, _ := txService.DecryptOutputCoinByKey(outputOfPrv, keyParam)
				if len(prvOutputs) > 0 {
					totalPrvValue := uint64(0)
					for _, output := range prvOutputs {
						totalPrvValue += output.GetValue()
					}
					results[common.PRVCoinID.String()] = totalPrvValue
				}
			}

			tokenData := tempTx.GetTxTokenData()
			results[tokenData.PropertyID.String()] = 0
			outputOfTokens := tokenData.TxNormal.GetProof().GetOutputCoins()
			if len(outputOfTokens) > 0 {
				tokenOutput, _ := txService.DecryptOutputCoinByKey(outputOfTokens, keyParam)
				if len(tokenOutput) > 0 {
					totalTokenValue := uint64(0)
					for _, output := range tokenOutput {
						totalTokenValue += output.GetValue()
					}
					results[tokenData.PropertyID.String()] = totalTokenValue
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

func (txService TxService) DecryptOutputCoinByKey(outCoins []coin.Coin, keyset *incognitokey.KeySet) ([]coin.PlainCoin, *RPCError) {
	keyset.PrivateKey = nil // always nil
	results := make([]coin.PlainCoin, 0)
	for _, out := range outCoins {
		decryptedOut, err := blockchain.DecryptOutputCoinByKey(txService.BlockChain.GetBestStateTransactionStateDB(0), out, keyset, nil, 0)
		if err != nil {
			return nil, NewRPCError(UnexpectedError, err)
		}
		if decryptedOut == nil {
			continue
		} else {
			results = append(results, decryptedOut)
		}
	}

	return results, nil
}

func (TxService TxService) GenerateOTAFromPaymentAddress(paymentAddressStr string) (string, string, error) {
	keySet, _, err := GetKeySetFromPaymentAddressParam(paymentAddressStr)
	if err != nil {
		Logger.log.Errorf("GenerateOTAFromPaymentAddress Cannot get keyset from payment address. Error: %+v", err)
		return "", "", err
	}
	publickey, txRandom, err := coin.NewOTAFromReceiver(keySet.PaymentAddress)
	if err != nil {
		Logger.log.Errorf("GenerateOTAFromPaymentAddress Cannot generate OTA Coin from keyset. Error: %+v", err)
		return "", "", err
	}
	return base58.Base58Check{}.Encode(publickey.ToBytesS(), common.ZeroByte), base58.Base58Check{}.Encode(txRandom.Bytes(), common.ZeroByte), nil
}



func (txService TxService) BuildRawDefragmentPrivacyCustomTokenTransaction(params interface{}, metaData metadata.Metadata) (transaction.TransactionToken, *RPCError) {
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
	inputCoins, realFeePRV, err := txService.chooseOutsCoinByKeyset(txParam.PaymentInfos,
		txParam.EstimateFeeCoinPerKb, 0, txParam.SenderKeySet,
		txParam.ShardIDSender, txParam.HasPrivacyCoin, nil, tokenParams)
	if err.(*RPCError) != nil {
		return nil, err.(*RPCError)
	}

	if len(txParam.PaymentInfos) == 0 && realFeePRV == 0 {
		txParam.HasPrivacyCoin = false
	}
	/******* END GET output coins native coins(PRV), which is used to create tx *****/
	beaconView := txService.BlockChain.BeaconChain.GetFinalViewState()

	txTokenParams := transaction.NewTxTokenParams(&txParam.SenderKeySet.PrivateKey,
		txParam.PaymentInfos,
		inputCoins,
		realFeePRV,
		tokenParams,
		txService.BlockChain.GetBestStateShard(txParam.ShardIDSender).GetCopiedTransactionStateDB(),
		metaData,
		txParam.HasPrivacyCoin,
		txParam.HasPrivacyToken,
		txParam.ShardIDSender, txParam.Info,
		beaconView.GetBeaconFeatureStateDB())

	tx, errTx := transaction.NewTransactionTokenFromParams(txTokenParams)
	if errTx != nil {
		Logger.log.Errorf("Cannot create new transaction token from params, err %v", err)
		return nil, NewRPCError(CreateTxDataError, errTx)
	}
	errTx = tx.Init(txTokenParams)
	if errTx != nil {
		return nil, NewRPCError(CreateTxDataError, errTx)
	}
	return tx, nil
}