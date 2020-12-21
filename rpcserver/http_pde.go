package rpcserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

type PDEWithdrawal struct {
	WithdrawalTokenIDStr string
	WithdrawerAddressStr string
	DeductingPoolValue   uint64
	DeductingShares      uint64
	PairToken1IDStr      string
	PairToken2IDStr      string
	TxReqID              common.Hash
	ShardID              byte
	Status               string
	BeaconHeight         uint64
}

type PDETrade struct {
	TraderAddressStr    string
	ReceivingTokenIDStr string
	ReceiveAmount       uint64
	Token1IDStr         string
	Token2IDStr         string
	ShardID             byte
	RequestedTxID       common.Hash
	Status              string
	BeaconHeight        uint64
}

type PDERefundedTradeV2 struct {
	TraderAddressStr string
	TokenIDStr       string
	ReceiveAmount    uint64
	ShardID          byte
	RequestedTxID    common.Hash
	Status           string
	BeaconHeight     uint64
}

type TradePath struct {
	TokenIDToBuyStr string
	ReceiveAmount   uint64
	SellAmount      uint64
	Token1IDStr     string
	Token2IDStr     string
}

type PDEAcceptedTradeV2 struct {
	TraderAddressStr string
	ShardID          byte
	RequestedTxID    common.Hash
	Status           string
	BeaconHeight     uint64
	TradePaths       []TradePath
}

type PDEContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
	ShardID               byte
	Status                string
	BeaconHeight          uint64
}

type PDEInfoFromBeaconBlock struct {
	PDEContributions    []*PDEContribution    `json:"PDEContributions"`
	PDETrades           []*PDETrade           `json:"PDETrades"`
	PDEWithdrawals      []*PDEWithdrawal      `json:"PDEWithdrawals"`
	PDEAcceptedTradesV2 []*PDEAcceptedTradeV2 `json:"PDEAcceptedTradesV2"`
	PDERefundedTradesV2 []*PDERefundedTradeV2 `json:"PDERefundedTradesV2"`
	BeaconTimeStamp     int64                 `json:"BeaconTimeStamp"`
}

type ConvertedPrice struct {
	FromTokenIDStr string
	ToTokenIDStr   string
	Amount         uint64
	Price          uint64
}

func (httpServer *HttpServer) handleCreateRawTxWithPRVContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	pdeContributionPairID, ok := data["PDEContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributorAddressStr, ok := data["ContributorAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmountData, ok := data["ContributedAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmount := uint64(contributedAmountData)
	tokenIDStr, ok := data["TokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	meta, _ := metadata.NewPDEContribution(
		pdeContributionPairID,
		contributorAddressStr,
		contributedAmount,
		tokenIDStr,
		metadata.PDEContributionMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPRVContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPRVContribution(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPTokenContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) >= 7 {
		hasPrivacyToken := int(arrayParams[6].(float64)) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}
	tokenParamsRaw := arrayParams[4].(map[string]interface{})

	pdeContributionPairID, ok := tokenParamsRaw["PDEContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributorAddressStr, ok := tokenParamsRaw["ContributorAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmountData, ok := tokenParamsRaw["ContributedAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmount := uint64(contributedAmountData)
	tokenIDStr := tokenParamsRaw["TokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	meta, _ := metadata.NewPDEContribution(
		pdeContributionPairID,
		contributorAddressStr,
		contributedAmount,
		tokenIDStr,
		metadata.PDEContributionMeta,
	)

	customTokenTx, rpcErr := httpServer.txService.BuildRawPrivacyCustomTokenTransaction(params, meta)
	if rpcErr != nil {
		Logger.log.Error(rpcErr)
		return nil, rpcErr
	}

	byteArrays, err2 := json.Marshal(customTokenTx)
	if err2 != nil {
		Logger.log.Error(err2)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            customTokenTx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPTokenContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPTokenContribution(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	// sendResult, err1 := httpServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	sendResult, err1 := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPRVTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tokenIDToBuyStr, ok := data["TokenIDToBuyStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tokenIDToSellStr, ok := data["TokenIDToSellStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	sellAmount := uint64(data["SellAmount"].(float64))
	traderAddressStr, ok := data["TraderAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	minAcceptableAmountData, ok := data["MinAcceptableAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid: minAcceptableAmountData"))
	}
	minAcceptableAmount := uint64(minAcceptableAmountData)
	tradingFeeData, ok := data["TradingFee"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tradingFee := uint64(tradingFeeData)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	traderOTAPublicKeyStr, traderOTAtxRandomStr, err := httpServer.txService.GenerateOTAFromPaymentAddress(traderAddressStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPDETradeRequest(
		tokenIDToBuyStr,
		tokenIDToSellStr,
		sellAmount,
		minAcceptableAmount,
		tradingFee,
		traderOTAPublicKeyStr,
		traderOTAtxRandomStr,
		metadata.PDETradeRequestMeta,
	)

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPRVTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPRVTradeReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPTokenTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) >= 7 {
		hasPrivacyToken := int(arrayParams[6].(float64)) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}
	tokenParamsRaw := arrayParams[4].(map[string]interface{})

	tokenIDToBuyStr, ok := tokenParamsRaw["TokenIDToBuyStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	tokenIDToSellStr, ok := tokenParamsRaw["TokenIDToSellStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	sellAmountData, ok := tokenParamsRaw["SellAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	sellAmount := uint64(sellAmountData)

	traderAddressStr, ok := tokenParamsRaw["TraderAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	minAcceptableAmountData, ok := tokenParamsRaw["MinAcceptableAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	minAcceptableAmount := uint64(minAcceptableAmountData)

	tradingFeeData, ok := tokenParamsRaw["TradingFee"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tradingFee := uint64(tradingFeeData)

	traderOTAPublicKeyStr, traderOTAtxRandomStr, err := httpServer.txService.GenerateOTAFromPaymentAddress(traderAddressStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPDETradeRequest(
		tokenIDToBuyStr,
		tokenIDToSellStr,
		sellAmount,
		minAcceptableAmount,
		tradingFee,
		traderOTAPublicKeyStr,
		traderOTAtxRandomStr,
		metadata.PDETradeRequestMeta,
	)

	customTokenTx, rpcErr := httpServer.txService.BuildRawPrivacyCustomTokenTransaction(params, meta)
	if rpcErr != nil {
		Logger.log.Error(rpcErr)
		return nil, rpcErr
	}

	byteArrays, err2 := json.Marshal(customTokenTx)
	if err2 != nil {
		Logger.log.Error(err2)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            customTokenTx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPTokenTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPTokenTradeReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	// sendResult, err1 := httpServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	sendResult, err1 := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithWithdrawalReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawerAddressStr, ok := data["WithdrawerAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken1IDStr, ok := data["WithdrawalToken1IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken2IDStr, ok := data["WithdrawalToken2IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalShareAmtData, ok := data["WithdrawalShareAmt"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	withdrawalShareAmt := uint64(withdrawalShareAmtData)

	meta, _ := metadata.NewPDEWithdrawalRequest(
		withdrawerAddressStr,
		withdrawalToken1IDStr,
		withdrawalToken2IDStr,
		withdrawalShareAmt,
		metadata.PDEWithdrawalRequestMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithWithdrawalReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithWithdrawalReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleGetPDEState(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Beacon height is invalid"))
	}
	beaconFeatureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, fmt.Errorf("Can't found ConsensusStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	beaconFeatureStateDB, err := statedb.NewWithPrefixTrie(beaconFeatureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	pdeState, err := blockchain.InitCurrentPDEStateFromDB(beaconFeatureStateDB, uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	beaconBlocks, err := httpServer.config.BlockChain.GetBeaconBlockByHeight(uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	beaconBlock := beaconBlocks[0]
	result := jsonresult.CurrentPDEState{
		BeaconTimeStamp:         beaconBlock.Header.Timestamp,
		PDEPoolPairs:            pdeState.PDEPoolPairs,
		PDEShares:               pdeState.PDEShares,
		WaitingPDEContributions: pdeState.WaitingPDEContributions,
		PDETradingFees:          pdeState.PDETradingFees,
	}
	return result, nil
}

func (httpServer *HttpServer) handleConvertNativeTokenToPrivacyToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	nativeTokenAmount, ok := data["NativeTokenAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenIDStr, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	beaconPdexStateDB, err := httpServer.config.BlockChain.GetBestStateBeaconFeatureStateDBByHeight(uint64(beaconHeight), httpServer.GetBeaconChainDatabase())
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	res, err := metadata.ConvertNativeTokenToPrivacyToken(
		uint64(nativeTokenAmount),
		tokenID,
		int64(beaconHeight),
		beaconPdexStateDB,
	)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return res, nil
}

func (httpServer *HttpServer) handleConvertPrivacyTokenToNativeToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	privacyTokenAmount, ok := data["PrivacyTokenAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenIDStr, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	beaconPdexStateDB, err := httpServer.config.BlockChain.GetBestStateBeaconFeatureStateDBByHeight(uint64(beaconHeight), httpServer.GetBeaconChainDatabase())
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	res, err := metadata.ConvertPrivacyTokenToNativeToken(
		uint64(privacyTokenAmount),
		tokenID,
		int64(beaconHeight),
		beaconPdexStateDB,
	)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return res, nil
}

func (httpServer *HttpServer) handleGetPDEContributionStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	contributionPairID, ok := data["ContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	status, err := httpServer.blockService.GetPDEStatus(rawdbv2.PDEContributionStatusPrefix, []byte(contributionPairID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPDEContributionStatusV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	contributionPairID, ok := data["ContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	contributionStatus, err := httpServer.blockService.GetPDEContributionStatus(rawdbv2.PDEContributionStatusPrefix, []byte(contributionPairID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return contributionStatus, nil
}

func (httpServer *HttpServer) handleGetPDETradeStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	txRequestIDStr, ok := data["TxRequestIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	txIDHash, err := common.Hash{}.NewHashFromStr(txRequestIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	status, err := httpServer.blockService.GetPDEStatus(rawdbv2.PDETradeStatusPrefix, txIDHash[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPDEWithdrawalStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	txRequestIDStr, ok := data["TxRequestIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	txIDHash, err := common.Hash{}.NewHashFromStr(txRequestIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	status, err := httpServer.blockService.GetPDEStatus(rawdbv2.PDEWithdrawalStatusPrefix, txIDHash[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPDEFeeWithdrawalStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	txRequestIDStr, ok := data["TxRequestIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	txIDHash, err := common.Hash{}.NewHashFromStr(txRequestIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	status, err := httpServer.blockService.GetPDEStatus(rawdbv2.PDEFeeWithdrawalStatusPrefix, txIDHash[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return status, nil
}

func parsePDEContributionInst(inst []string, beaconHeight uint64) (*PDEContribution, error) {
	status := inst[2]
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}
	if status == common.PDEContributionMatchedChainStatus {
		matchedContribContent := []byte(inst[3])
		var matchedContrib metadata.PDEMatchedContribution
		err := json.Unmarshal(matchedContribContent, &matchedContrib)
		if err != nil {
			return nil, err
		}
		return &PDEContribution{
			PDEContributionPairID: matchedContrib.PDEContributionPairID,
			ContributorAddressStr: matchedContrib.ContributorAddressStr,
			ContributedAmount:     matchedContrib.ContributedAmount,
			TokenIDStr:            matchedContrib.TokenIDStr,
			TxReqID:               matchedContrib.TxReqID,
			ShardID:               byte(shardID),
			Status:                common.PDEContributionMatchedChainStatus,
			BeaconHeight:          beaconHeight,
		}, nil
	}
	if status == common.PDEContributionRefundChainStatus {
		refundedContribContent := []byte(inst[3])
		var refundedContrib metadata.PDERefundContribution
		err := json.Unmarshal(refundedContribContent, &refundedContrib)
		if err != nil {
			return nil, err
		}
		return &PDEContribution{
			PDEContributionPairID: refundedContrib.PDEContributionPairID,
			ContributorAddressStr: refundedContrib.ContributorAddressStr,
			ContributedAmount:     refundedContrib.ContributedAmount,
			TokenIDStr:            refundedContrib.TokenIDStr,
			TxReqID:               refundedContrib.TxReqID,
			ShardID:               byte(shardID),
			Status:                common.PDEContributionRefundChainStatus,
			BeaconHeight:          beaconHeight,
		}, nil
	}
	return nil, nil
}

func parsePDETradeInst(inst []string, beaconHeight uint64) (*PDETrade, error) {
	status := inst[2]
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}
	if status == common.PDETradeRefundChainStatus {
		contentBytes, err := base64.StdEncoding.DecodeString(inst[3])
		if err != nil {
			return nil, err
		}
		var pdeTradeReqAction metadata.PDETradeRequestAction
		err = json.Unmarshal(contentBytes, &pdeTradeReqAction)
		if err != nil {
			return nil, err
		}
		tokenIDStrs := []string{pdeTradeReqAction.Meta.TokenIDToBuyStr, pdeTradeReqAction.Meta.TokenIDToSellStr}
		sort.Slice(tokenIDStrs, func(i, j int) bool {
			return tokenIDStrs[i] < tokenIDStrs[j]
		})
		return &PDETrade{
			TraderAddressStr:    pdeTradeReqAction.Meta.TraderAddressStr,
			ReceivingTokenIDStr: pdeTradeReqAction.Meta.TokenIDToSellStr,
			ReceiveAmount:       pdeTradeReqAction.Meta.SellAmount + pdeTradeReqAction.Meta.TradingFee,
			Token1IDStr:         tokenIDStrs[0],
			Token2IDStr:         tokenIDStrs[1],
			ShardID:             byte(shardID),
			RequestedTxID:       pdeTradeReqAction.TxReqID,
			Status:              "refunded",
			BeaconHeight:        beaconHeight,
		}, nil
	}
	if status == common.PDETradeAcceptedChainStatus {
		tradeAcceptedContentBytes := []byte(inst[3])
		var tradeAcceptedContent metadata.PDETradeAcceptedContent
		err := json.Unmarshal(tradeAcceptedContentBytes, &tradeAcceptedContent)
		if err != nil {
			return nil, err
		}
		tokenIDStrs := []string{tradeAcceptedContent.Token1IDStr, tradeAcceptedContent.Token2IDStr}
		sort.Slice(tokenIDStrs, func(i, j int) bool {
			return tokenIDStrs[i] < tokenIDStrs[j]
		})
		return &PDETrade{
			TraderAddressStr:    tradeAcceptedContent.TraderAddressStr,
			ReceivingTokenIDStr: tradeAcceptedContent.TokenIDToBuyStr,
			ReceiveAmount:       tradeAcceptedContent.ReceiveAmount,
			Token1IDStr:         tokenIDStrs[0],
			Token2IDStr:         tokenIDStrs[1],
			ShardID:             byte(shardID),
			RequestedTxID:       tradeAcceptedContent.RequestedTxID,
			Status:              "accepted",
			BeaconHeight:        beaconHeight,
		}, nil
	}
	return nil, nil
}

func parsePDERefundedTradeV2Inst(inst []string, beaconHeight uint64) (*PDERefundedTradeV2, error) {
	status := inst[2]
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}

	refundStatus := "fee-refund"
	if status == common.PDECrossPoolTradeSellingTokenRefundChainStatus {
		refundStatus = "selling-token-refund"
	}

	var pdeRefundCrossPoolTrade metadata.PDERefundCrossPoolTrade
	err = json.Unmarshal([]byte(inst[3]), &pdeRefundCrossPoolTrade)
	if err != nil {
		return nil, err
	}
	return &PDERefundedTradeV2{
		TraderAddressStr: pdeRefundCrossPoolTrade.TraderAddressStr,
		TokenIDStr:       pdeRefundCrossPoolTrade.TokenIDStr,
		ReceiveAmount:    pdeRefundCrossPoolTrade.Amount,
		ShardID:          byte(shardID),
		RequestedTxID:    pdeRefundCrossPoolTrade.TxReqID,
		Status:           refundStatus,
		BeaconHeight:     beaconHeight,
	}, nil
}

func parsePDEAcceptedTradeV2Inst(inst []string, beaconHeight uint64) (*PDEAcceptedTradeV2, error) {
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}

	var tradeAcceptedContents []metadata.PDECrossPoolTradeAcceptedContent
	err = json.Unmarshal([]byte(inst[3]), &tradeAcceptedContents)
	if err != nil {
		return nil, err
	}
	if len(tradeAcceptedContents) == 0 {
		return nil, nil
	}

	tradePaths := make([]TradePath, len(tradeAcceptedContents))
	for idx, tradeContent := range tradeAcceptedContents {
		tradePaths[idx] = TradePath{
			TokenIDToBuyStr: tradeContent.TokenIDToBuyStr,
			ReceiveAmount:   tradeContent.ReceiveAmount,
			Token1IDStr:     tradeContent.Token1IDStr,
			Token2IDStr:     tradeContent.Token2IDStr,
		}
		tradePaths[idx].SellAmount = tradeContent.Token2PoolValueOperation.Value
		if tradeContent.Token1PoolValueOperation.Operator == "+" {
			tradePaths[idx].SellAmount = tradeContent.Token1PoolValueOperation.Value
		}
	}

	receivingTokenIDStr := tradePaths[len(tradePaths)-1].TokenIDToBuyStr
	sellingTokenIDStr := tradePaths[0].Token1IDStr
	if sellingTokenIDStr == receivingTokenIDStr {
		sellingTokenIDStr = tradePaths[0].Token2IDStr
	}

	return &PDEAcceptedTradeV2{
		TraderAddressStr: tradeAcceptedContents[0].TraderAddressStr,
		ShardID:          byte(shardID),
		RequestedTxID:    tradeAcceptedContents[0].RequestedTxID,
		Status:           "accepted",
		BeaconHeight:     beaconHeight,
		TradePaths:       tradePaths,
	}, nil
}

func parsePDEWithdrawalInst(inst []string, beaconHeight uint64) (*PDEWithdrawal, error) {
	status := inst[2]
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}
	if status == common.PDEWithdrawalAcceptedChainStatus {
		withdrawalAcceptedContentBytes := []byte(inst[3])
		var withdrawalAcceptedContent metadata.PDEWithdrawalAcceptedContent
		err := json.Unmarshal(withdrawalAcceptedContentBytes, &withdrawalAcceptedContent)
		if err != nil {
			return nil, err
		}
		tokenIDStrs := []string{withdrawalAcceptedContent.PairToken1IDStr, withdrawalAcceptedContent.PairToken2IDStr}
		sort.Slice(tokenIDStrs, func(i, j int) bool {
			return tokenIDStrs[i] < tokenIDStrs[j]
		})
		return &PDEWithdrawal{
			WithdrawalTokenIDStr: withdrawalAcceptedContent.WithdrawalTokenIDStr,
			WithdrawerAddressStr: withdrawalAcceptedContent.WithdrawerAddressStr,
			DeductingPoolValue:   withdrawalAcceptedContent.DeductingPoolValue,
			DeductingShares:      withdrawalAcceptedContent.DeductingShares,
			PairToken1IDStr:      tokenIDStrs[0],
			PairToken2IDStr:      tokenIDStrs[1],
			TxReqID:              withdrawalAcceptedContent.TxReqID,
			ShardID:              byte(shardID),
			Status:               "accepted",
			BeaconHeight:         beaconHeight,
		}, nil
	}
	return nil, nil
}

func (httpServer *HttpServer) handleExtractPDEInstsFromBeaconBlock(
	params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError,
) {
	arrayParams := common.InterfaceSlice(params)
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Beacon height is invalid"))
	}

	bcHeight := uint64(beaconHeight)
	beaconBlocks, err := blockchain.FetchBeaconBlockFromHeight(
		httpServer.config.BlockChain,
		bcHeight,
		bcHeight,
	)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	if len(beaconBlocks) == 0 {
		return nil, nil
	}
	bcBlk := beaconBlocks[0]
	pdeInfoFromBeaconBlock := PDEInfoFromBeaconBlock{
		PDEContributions:    []*PDEContribution{},
		PDETrades:           []*PDETrade{},
		PDEAcceptedTradesV2: []*PDEAcceptedTradeV2{},
		PDERefundedTradesV2: []*PDERefundedTradeV2{},
		PDEWithdrawals:      []*PDEWithdrawal{},
		BeaconTimeStamp:     bcBlk.Header.Timestamp,
	}
	insts := bcBlk.Body.Instructions
	for _, inst := range insts {
		if len(inst) < 2 {
			continue // Not error, just not PDE instruction
		}
		switch inst[0] {
		case strconv.Itoa(metadata.PDEContributionMeta):
			pdeContrib, err := parsePDEContributionInst(inst, bcHeight)
			if err != nil || pdeContrib == nil {
				continue
			}
			pdeInfoFromBeaconBlock.PDEContributions = append(pdeInfoFromBeaconBlock.PDEContributions, pdeContrib)
		case strconv.Itoa(metadata.PDETradeRequestMeta):
			pdeTrade, err := parsePDETradeInst(inst, bcHeight)
			if err != nil || pdeTrade == nil {
				continue
			}
			pdeInfoFromBeaconBlock.PDETrades = append(pdeInfoFromBeaconBlock.PDETrades, pdeTrade)
		case strconv.Itoa(metadata.PDEWithdrawalRequestMeta):
			pdeWithdrawal, err := parsePDEWithdrawalInst(inst, bcHeight)
			if err != nil || pdeWithdrawal == nil {
				continue
			}
			pdeInfoFromBeaconBlock.PDEWithdrawals = append(pdeInfoFromBeaconBlock.PDEWithdrawals, pdeWithdrawal)

		case strconv.Itoa(metadata.PDECrossPoolTradeRequestMeta):
			if inst[2] == common.PDECrossPoolTradeAcceptedChainStatus {
				acceptedTradeV2, err := parsePDEAcceptedTradeV2Inst(inst, bcHeight)
				if err != nil || acceptedTradeV2 == nil {
					continue
				}
				pdeInfoFromBeaconBlock.PDEAcceptedTradesV2 = append(pdeInfoFromBeaconBlock.PDEAcceptedTradesV2, acceptedTradeV2)
			} else {
				refundedTradeV2, err := parsePDERefundedTradeV2Inst(inst, bcHeight)
				if err != nil || refundedTradeV2 == nil {
					continue
				}
				pdeInfoFromBeaconBlock.PDERefundedTradesV2 = append(pdeInfoFromBeaconBlock.PDERefundedTradesV2, refundedTradeV2)
			}
		}
	}
	return pdeInfoFromBeaconBlock, nil
}

func convertPrice(
	latestBcHeight uint64,
	toTokenIDStr string,
	fromTokenIDStr string,
	convertingAmt uint64,
	pdePoolPairs map[string]*rawdbv2.PDEPoolForPair,
) *ConvertedPrice {
	poolPairKey := rawdbv2.BuildPDEPoolForPairKey(
		latestBcHeight,
		toTokenIDStr,
		fromTokenIDStr,
	)
	poolPair, found := pdePoolPairs[string(poolPairKey)]
	if !found || poolPair == nil {
		return nil
	}
	if poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
		return nil
	}

	tokenPoolValueToBuy := poolPair.Token1PoolValue
	tokenPoolValueToSell := poolPair.Token2PoolValue
	if poolPair.Token1IDStr == fromTokenIDStr {
		tokenPoolValueToBuy = poolPair.Token2PoolValue
		tokenPoolValueToSell = poolPair.Token1PoolValue
	}

	invariant := big.NewInt(0)
	invariant.Mul(new(big.Int).SetUint64(tokenPoolValueToSell), new(big.Int).SetUint64(tokenPoolValueToBuy))

	newTokenPoolValueToSell := big.NewInt(0)
	newTokenPoolValueToSell.Add(new(big.Int).SetUint64(tokenPoolValueToSell), new(big.Int).SetUint64(convertingAmt))

	newTokenPoolValueToBuy := big.NewInt(0).Div(invariant, newTokenPoolValueToSell).Uint64()
	modValue := big.NewInt(0).Mod(invariant, newTokenPoolValueToSell)
	if modValue.Cmp(big.NewInt(0)) != 0 {
		newTokenPoolValueToBuy++
	}
	if tokenPoolValueToBuy <= newTokenPoolValueToBuy {
		return nil
	}
	return &ConvertedPrice{
		FromTokenIDStr: fromTokenIDStr,
		ToTokenIDStr:   toTokenIDStr,
		Amount:         convertingAmt,
		Price:          tokenPoolValueToBuy - newTokenPoolValueToBuy,
	}
}

func (httpServer *HttpServer) handleConvertPDEPrices(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	latestBeaconHeight := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight

	arrayParams := common.InterfaceSlice(params)
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	fromTokenIDStr, ok := data["FromTokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("FromTokenIDStr is invalid"))
	}
	toTokenIDStr, ok := data["ToTokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("ToTokenIDStr is invalid"))
	}
	amount, ok := data["Amount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Amount is invalid"))
	}
	convertingAmt := uint64(amount)
	if convertingAmt == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Amount is invalid"))
	}
	beaconFeatureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(latestBeaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, fmt.Errorf("Can't found ConsensusStateRootHash of beacon height %+v, error %+v", latestBeaconHeight, err))
	}
	beaconFeatureStateDB, err := statedb.NewWithPrefixTrie(beaconFeatureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	pdeState, err := blockchain.InitCurrentPDEStateFromDB(beaconFeatureStateDB, latestBeaconHeight)
	if err != nil || pdeState == nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	pdePoolPairs := pdeState.PDEPoolPairs
	results := []*ConvertedPrice{}
	if toTokenIDStr != "all" {
		convertedPrice := convertPrice(
			latestBeaconHeight,
			toTokenIDStr,
			fromTokenIDStr,
			convertingAmt,
			pdePoolPairs,
		)
		if convertedPrice == nil {
			return results, nil
		}
		return append(results, convertedPrice), nil
	}
	// compute price of "from" token against all tokens else
	for poolPairKey, poolPair := range pdePoolPairs {
		if !strings.Contains(poolPairKey, fromTokenIDStr) {
			continue
		}
		var convertedPrice *ConvertedPrice
		if poolPair.Token1IDStr == fromTokenIDStr {
			convertedPrice = convertPrice(
				latestBeaconHeight,
				poolPair.Token2IDStr,
				fromTokenIDStr,
				convertingAmt,
				pdePoolPairs,
			)
		} else if poolPair.Token2IDStr == fromTokenIDStr {
			convertedPrice = convertPrice(
				latestBeaconHeight,
				poolPair.Token1IDStr,
				fromTokenIDStr,
				convertingAmt,
				pdePoolPairs,
			)
		}
		if convertedPrice == nil {
			continue
		}
		results = append(results, convertedPrice)
	}
	return results, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPRVContributionV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	pdeContributionPairID, ok := data["PDEContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributorAddressStr, ok := data["ContributorAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmount, err := common.AssertAndConvertStrToNumber(data["ContributedAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	tokenIDStr, ok := data["TokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	meta, _ := metadata.NewPDEContribution(
		pdeContributionPairID,
		contributorAddressStr,
		contributedAmount,
		tokenIDStr,
		metadata.PDEPRVRequiredContributionRequestMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParamV2(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPRVContributionV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPRVContributionV2(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPTokenContributionV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) >= 7 {
		hasPrivacyToken := int(arrayParams[6].(float64)) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}
	tokenParamsRaw := arrayParams[4].(map[string]interface{})

	pdeContributionPairID, ok := tokenParamsRaw["PDEContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributorAddressStr, ok := tokenParamsRaw["ContributorAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmount, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["ContributedAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	tokenIDStr := tokenParamsRaw["TokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	meta, _ := metadata.NewPDEContribution(
		pdeContributionPairID,
		contributorAddressStr,
		contributedAmount,
		tokenIDStr,
		metadata.PDEPRVRequiredContributionRequestMeta,
	)

	customTokenTx, rpcErr := httpServer.txService.BuildRawPrivacyCustomTokenTransactionV2(params, meta)
	if rpcErr != nil {
		Logger.log.Error(rpcErr)
		return nil, rpcErr
	}

	byteArrays, err2 := json.Marshal(customTokenTx)
	if err2 != nil {
		Logger.log.Error(err2)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            customTokenTx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPTokenContributionV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPTokenContributionV2(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	// sendResult, err1 := httpServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	sendResult, err1 := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPRVCrossPoolTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tokenIDToBuyStr, ok := data["TokenIDToBuyStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tokenIDToSellStr, ok := data["TokenIDToSellStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	sellAmount, err := common.AssertAndConvertStrToNumber(data["SellAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	traderAddressStr, ok := data["TraderAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	minAcceptableAmount, err := common.AssertAndConvertStrToNumber(data["MinAcceptableAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	tradingFee, err := common.AssertAndConvertStrToNumber(data["TradingFee"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	traderOTAPublicKeyStr, traderOTAtxRandomStr, err := httpServer.txService.GenerateOTAFromPaymentAddress(traderAddressStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPDECrossPoolTradeRequest(
		tokenIDToBuyStr,
		tokenIDToSellStr,
		sellAmount,
		minAcceptableAmount,
		tradingFee,
		traderOTAPublicKeyStr,
		traderOTAtxRandomStr,
		metadata.PDECrossPoolTradeRequestMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParamV2(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPRVCrossPoolTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPRVCrossPoolTradeReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPTokenCrossPoolTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) >= 7 {
		hasPrivacyToken := int(arrayParams[6].(float64)) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}
	tokenParamsRaw := arrayParams[4].(map[string]interface{})

	tokenIDToBuyStr, ok := tokenParamsRaw["TokenIDToBuyStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	tokenIDToSellStr, ok := tokenParamsRaw["TokenIDToSellStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	sellAmount, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["SellAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	traderAddressStr, ok := tokenParamsRaw["TraderAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	minAcceptableAmount, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["MinAcceptableAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	tradingFee, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["TradingFee"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	traderOTAPublicKeyStr, traderOTAtxRandomStr, err := httpServer.txService.GenerateOTAFromPaymentAddress(traderAddressStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPDECrossPoolTradeRequest(
		tokenIDToBuyStr,
		tokenIDToSellStr,
		sellAmount,
		minAcceptableAmount,
		tradingFee,
		traderOTAPublicKeyStr,
		traderOTAtxRandomStr,
		metadata.PDECrossPoolTradeRequestMeta,
	)

	customTokenTx, rpcErr := httpServer.txService.BuildRawPrivacyCustomTokenTransactionV2(params, meta)
	if rpcErr != nil {
		Logger.log.Error(rpcErr)
		return nil, rpcErr
	}

	byteArrays, err2 := json.Marshal(customTokenTx)
	if err2 != nil {
		Logger.log.Error(err2)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            customTokenTx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPTokenCrossPoolTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPTokenCrossPoolTradeReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	// sendResult, err1 := httpServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	sendResult, err1 := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithWithdrawalReqV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawerAddressStr, ok := data["WithdrawerAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken1IDStr, ok := data["WithdrawalToken1IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken2IDStr, ok := data["WithdrawalToken2IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalShareAmt, err := common.AssertAndConvertStrToNumber(data["WithdrawalShareAmt"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPDEWithdrawalRequest(
		withdrawerAddressStr,
		withdrawalToken1IDStr,
		withdrawalToken2IDStr,
		withdrawalShareAmt,
		metadata.PDEWithdrawalRequestMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParamV2(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithWithdrawalReqV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithWithdrawalReqV2(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPDEFeeWithdrawalReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawerAddressStr, ok := data["WithdrawerAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken1IDStr, ok := data["WithdrawalToken1IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken2IDStr, ok := data["WithdrawalToken2IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalFeeAmt, err := common.AssertAndConvertStrToNumber(data["WithdrawalFeeAmt"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPDEFeeWithdrawalRequest(
		withdrawerAddressStr,
		withdrawalToken1IDStr,
		withdrawalToken2IDStr,
		withdrawalFeeAmt,
		metadata.PDEFeeWithdrawalRequestMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParamV2(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPDEFeeWithdrawalReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPDEFeeWithdrawalReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}
