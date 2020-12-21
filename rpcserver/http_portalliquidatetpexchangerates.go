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

func (httpServer *HttpServer) handleGetLiquidationExchangeRatesPool(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	pTokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}

	if !common.IsPortalExchangeRateToken(pTokenID) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is not support"))
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetExchangeRatesLiquidationPoolError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetExchangeRatesLiquidationPoolError, err)
	}

	result, err := httpServer.portal.GetLiquidateExchangeRatesPool(stateDB, pTokenID)
	if err != nil {
		return result, rpcservice.NewRPCError(rpcservice.GetExchangeRatesLiquidationPoolError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleGetAmountNeededForCustodianDepositLiquidation(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	custodianAddress, ok := data["CustodianAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianAddress is invalid"))
	}

	pTokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}

	if !common.IsPortalToken(pTokenID) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is not support"))
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalStateError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetAmountNeededForCustodianDepositLiquidationError, err)
	}

	portalParam := httpServer.config.BlockChain.GetPortalParams(uint64(beaconHeight))

	result, err := httpServer.portal.CalculateAmountNeededCustodianDepositLiquidation(stateDB, custodianAddress, pTokenID, portalParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetAmountNeededForCustodianDepositLiquidationError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) createRawRedeemLiquidationExchangeRates(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Params should be not empty"))
	}

	if len(arrayParams) >= 7 {
		hasPrivacyTokenParam, ok := arrayParams[6].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("HasPrivacyToken is invalid"))
		}
		hasPrivacyToken := int(hasPrivacyTokenParam) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("The privacy mode must be disabled"))
		}
	}

	if len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 5"))
	}

	tokenParamsRaw, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param metadata is invalid"))
	}

	redeemTokenID, ok := tokenParamsRaw["RedeemTokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("RedeemTokenID is invalid"))
	}

	redeemAmount, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["RedeemAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	redeemerIncAddressStr, ok := tokenParamsRaw["RedeemerIncAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("RedeemerIncAddressStr is invalid"))
	}

	meta, _ := metadata.NewPortalRedeemLiquidateExchangeRates(metadata.PortalRedeemLiquidateExchangeRatesMeta, redeemTokenID, redeemAmount, redeemerIncAddressStr)

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

func (httpServer *HttpServer) handleCreateAndSendRedeemLiquidationExchangeRates(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.createRawRedeemLiquidationExchangeRates(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err1 := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) createLiquidationCustodianDeposit(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	incognitoAddress, ok := data["IncognitoAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata IncognitoAddress is invalid"))
	}

	pTokenId, ok := data["PTokenId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PTokenId param is invalid"))
	}

	if !common.IsPortalToken(pTokenId) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata public token is not supported currently"))
	}

	freeCollateralAmount, err := common.AssertAndConvertStrToNumber(data["FreeCollateralAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	depositedAmount, err := common.AssertAndConvertStrToNumber(data["DepositedAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPortalLiquidationCustodianDepositV2(
		metadata.PortalLiquidationCustodianDepositMetaV2,
		incognitoAddress,
		pTokenId,
		depositedAmount,
		freeCollateralAmount,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParamV2(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}
	// HasPrivacyCoin param is always false
	createRawTxParam.HasPrivacyCoin = false

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

func (httpServer *HttpServer) handleCreateAndSendLiquidationCustodianDeposit(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.createLiquidationCustodianDeposit(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err1 := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) createTopUpWaitingPorting(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	portingID, ok := data["PortingID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PortingID is invalid"))
	}

	incognitoAddress, ok := data["IncognitoAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata IncognitoAddress is invalid"))
	}

	pTokenId, ok := data["PTokenId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PTokenId param is invalid"))
	}

	if !common.IsPortalToken(pTokenId) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata public token is not supported currently"))
	}

	freeCollateralAmount, err := common.AssertAndConvertStrToNumber(data["FreeCollateralAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	depositedAmount, err := common.AssertAndConvertStrToNumber(data["DepositedAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPortalTopUpWaitingPortingRequest(
		metadata.PortalTopUpWaitingPortingRequestMeta,
		portingID,
		incognitoAddress,
		pTokenId,
		depositedAmount,
		freeCollateralAmount,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParamV2(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}
	// HasPrivacyCoin param is always false
	createRawTxParam.HasPrivacyCoin = false

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

func (httpServer *HttpServer) handleCreateAndSendTopUpWaitingPorting(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.createTopUpWaitingPorting(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err1 := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) handleGetPortalCustodianTopupStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	txID, ok := data["TxID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param TxID is invalid"))
	}

	status, err := httpServer.blockService.GetCustodianTopupStatus(txID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianTopupStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPortalCustodianTopupWaitingPortingStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	txID, ok := data["TxID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param TxID is invalid"))
	}

	status, err := httpServer.blockService.GetCustodianTopupWaitingPortingStatus(txID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianTopupWaitingPortingStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetAmountTopUpWaitingPorting(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	custodianAddr, ok := data["CustodianAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param CustodianAddress is invalid"))
	}

	result, err := httpServer.blockService.GetAmountTopUpWaitingPorting(custodianAddr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetAmountTopUpWaitingPortingError, err)
	}
	return result, nil
}

func (httpServer *HttpServer) handleGetReqRedeemFromLiquidationPoolByTxIDStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	reqTxID, ok := data["ReqTxID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param ReqTxID is invalid"))
	}
	status, err := httpServer.blockService.GetRedeemReqFromLiquidationPoolByTxIDStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqRedeemFromLiquidationPoolStatusError, err)
	}
	return status, nil
}
