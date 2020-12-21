package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

func (httpServer *HttpServer) createPortalExchangeRate(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Params should be not empty"))
	}

	if len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 5"))
	}

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}

	senderAddress, ok := data["SenderAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata SenderAddress is invalid"))
	}

	var exchangeRate = make([]*metadata.ExchangeRateInfo, 0)

	exchangeRateMap, ok := data["Rates"].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Rates is invalid"))
	}

	if len(exchangeRateMap) <= 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Rates is invalid"))
	}

	for pTokenID, value := range exchangeRateMap {
		if !common.IsPortalExchangeRateToken(pTokenID) {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("TokenID is not portal exchange rate token"))
		}

		amount, err := common.AssertAndConvertStrToNumber(value)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
		}

		if amount <= 0 {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Exchange rates should be larger than 0"))
		}

		exchangeRate = append(
			exchangeRate,
			&metadata.ExchangeRateInfo{
				PTokenID: pTokenID,
				Rate:     uint64(amount),
			})
	}

	meta, _ := metadata.NewPortalExchangeRates(
		metadata.PortalExchangeRatesMeta,
		senderAddress,
		exchangeRate,
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

func (httpServer *HttpServer) handleCreateAndSendPortalExchangeRates(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.createPortalExchangeRate(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalFinalExchangeRates(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Params should be not empty"))
	}

	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 1"))
	}

	// get meta data from params
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}

	beaconHeight, err := common.AssertAndConvertStrToNumber(data["BeaconHeight"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetFinalExchangeRatesError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetFinalExchangeRatesError, err)
	}

	result, err := httpServer.portal.GetFinalExchangeRates(stateDB, uint64(beaconHeight))

	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetFinalExchangeRatesError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleConvertExchangeRates(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Params should be not empty"))
	}

	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 1"))
	}

	// get meta data from params
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}

	valuePToken, err := common.AssertAndConvertStrToNumber(data["ValuePToken"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	tokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}

	beaconHeight, err := common.AssertAndConvertStrToNumber(data["BeaconHeight"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	_, err = httpServer.config.BlockChain.GetBeaconBlockByHeight(uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ConvertExchangeRatesError, err)
	}

	if !common.IsPortalToken(tokenID) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is not support"))
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ConvertExchangeRatesError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))

	result, err := httpServer.portal.ConvertExchangeRates(stateDB, tokenID, uint64(valuePToken))

	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ConvertExchangeRatesError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleGetPortingRequestFees(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Params should be not empty"))
	}

	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 1"))
	}

	// get meta data from params
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}

	valuePToken, err := common.AssertAndConvertStrToNumber(data["ValuePToken"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	tokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}

	beaconHeight, err := common.AssertAndConvertStrToNumber(data["BeaconHeight"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	if !common.IsPortalToken(tokenID) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is not support"))
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestFeesError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestFeesError, err)
	}

	portalParams := httpServer.config.BlockChain.GetPortalParams(uint64(beaconHeight))

	result, err := httpServer.portal.GetPortingFees(stateDB, tokenID, uint64(valuePToken), portalParams)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestFeesError, err)
	}

	return result, nil
}
