package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	portalMeta "github.com/incognitochain/incognito-chain/portal/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"strings"
)

/*
====== Portal liquidation pool
*/
func (httpServer *HttpServer) handleGetLiquidationExchangeRatesPool(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	if !httpServer.config.BlockChain.GetPortalParams(beaconHeight).IsPortalToken(pTokenID) {
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

/*
====== Redeem request from liquidation pool v3
*/
func (httpServer *HttpServer) createRawTxRedeemFromLiquidationPoolV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	redeemerExtAddressStr, ok := tokenParamsRaw["RedeemerExtAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("RedeemerExtAddressStr is invalid"))
	}

	meta, _ := portalMeta.NewPortalRedeemFromLiquidationPoolV3(
		basemeta.PortalRedeemFromLiquidationPoolMetaV3, redeemTokenID,
		redeemAmount, redeemerIncAddressStr, redeemerExtAddressStr)

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

func (httpServer *HttpServer) handleCreateAndSendTxRedeemFromLiquidationPoolV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.createRawTxRedeemFromLiquidationPoolV3(params, closeChan)
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

func (httpServer *HttpServer) handleGetReqRedeemFromLiquidationPoolByTxIDStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

func (httpServer *HttpServer) handleGetReqRedeemFromLiquidationPoolByTxIDStatusV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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
	status, err := httpServer.blockService.GetRedeemReqFromLiquidationPoolByTxIDStatusV3(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqRedeemFromLiquidationPoolStatusError, err)
	}
	return status, nil
}

/*
====== Topup collateral (PRV)
*/
func (httpServer *HttpServer) createCustodianTopup(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	if !httpServer.config.BlockChain.GetPortalParams(0).IsPortalToken(pTokenId) {
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

	meta, _ := portalMeta.NewPortalLiquidationCustodianDepositV2(
		basemeta.PortalCustodianTopupMetaV2,
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

func (httpServer *HttpServer) handleCreateAndSendCustodianTopup(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.createCustodianTopup(params, closeChan)
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
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	if !httpServer.config.BlockChain.GetPortalParams(0).IsPortalToken(pTokenId) {
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

	meta, _ := portalMeta.NewPortalTopUpWaitingPortingRequest(
		basemeta.PortalTopUpWaitingPortingRequestMeta,
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
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

/*
====== Topup collateral v3 (ETH/ERC20)
*/
func (httpServer *HttpServer) createCustodianTopupV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	if !httpServer.config.BlockChain.GetPortalParams(0).IsPortalToken(pTokenId) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata public token is not supported currently"))
	}

	collateralTokenId, ok := data["CollateralTokenId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CollateralTokenId param is invalid"))
	}
	collateralTokenId = strings.ToLower(common.Remove0xPrefix(collateralTokenId))
	beaconHeight := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	if !httpServer.config.BlockChain.IsSupportedTokenCollateralV3(beaconHeight, collateralTokenId) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata collateral tokenID is not supported currently"))
	}

	freeCollateralAmount, err := common.AssertAndConvertStrToNumber(data["FreeCollateralAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	depositedAmount, err := common.AssertAndConvertStrToNumber(data["DepositedAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	blockHashStr, ok := data["BlockHash"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata BlockHash should be a string"))
	}
	blockHash := eCommon.HexToHash(blockHashStr)

	txIndexFloat, ok := data["TxIndex"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TxIndex should be a number"))
	}
	txIdx := uint(txIndexFloat)

	proofsRaw, ok := data["ProofStrs"].([]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata ProofStrs should be an array of string"))
	}
	proofStrs := []string{}
	for _, item := range proofsRaw {
		proofStrs = append(proofStrs, item.(string))
	}

	meta, _ := portalMeta.NewPortalLiquidationCustodianDepositV3(
		basemeta.PortalCustodianTopupMetaV3,
		incognitoAddress,
		pTokenId,
		collateralTokenId,
		depositedAmount,
		freeCollateralAmount,
		blockHash,
		txIdx,
		proofStrs,
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

func (httpServer *HttpServer) handleCreateAndSendCustodianTopupV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.createCustodianTopupV3(params, closeChan)
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

func (httpServer *HttpServer) createTopUpWaitingPortingV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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
	if !httpServer.config.BlockChain.GetPortalParams(0).IsPortalToken(pTokenId) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata public token is not supported currently"))
	}

	collateralTokenId, ok := data["CollateralTokenId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CollateralTokenId param is invalid"))
	}
	collateralTokenId = strings.ToLower(common.Remove0xPrefix(collateralTokenId))
	beaconHeight := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	if !httpServer.config.BlockChain.IsSupportedTokenCollateralV3(beaconHeight, collateralTokenId) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata collateral tokenID is not supported currently"))
	}

	freeCollateralAmount, err := common.AssertAndConvertStrToNumber(data["FreeCollateralAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	depositedAmount, err := common.AssertAndConvertStrToNumber(data["DepositedAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	blockHashStr, ok := data["BlockHash"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata BlockHash should be a string"))
	}
	blockHash := eCommon.HexToHash(blockHashStr)

	txIndexFloat, ok := data["TxIndex"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TxIndex should be a number"))
	}
	txIdx := uint(txIndexFloat)

	proofsRaw, ok := data["ProofStrs"].([]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata ProofStrs should be an array of string"))
	}
	proofStrs := []string{}
	for _, item := range proofsRaw {
		proofStrs = append(proofStrs, item.(string))
	}

	meta, _ := portalMeta.NewPortalTopUpWaitingPortingRequestV3(
		basemeta.PortalTopUpWaitingPortingRequestMetaV3,
		incognitoAddress,
		pTokenId,
		collateralTokenId,
		depositedAmount,
		freeCollateralAmount,
		portingID,
		blockHash,
		txIdx,
		proofStrs,
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

func (httpServer *HttpServer) handleCreateAndSendTopUpWaitingPortingV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.createTopUpWaitingPortingV3(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalCustodianTopupStatusV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	status, err := httpServer.blockService.GetCustodianTopupStatusV3(txID)

	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianTopupStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPortalCustodianTopupWaitingPortingStatusV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	status, err := httpServer.blockService.GetCustodianTopupWaitingPortingStatusV3(txID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianTopupWaitingPortingStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetTopupAmountForCustodianState(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	// default get best beacon height
	beaconHeight := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	beaconHeightParam, err := common.AssertAndConvertStrToNumber(data["BeaconHeight"])
	if err == nil || beaconHeightParam > 0 {
		beaconHeight = beaconHeightParam
	}

	custodianAddress, ok := data["CustodianAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianAddress is invalid"))
	}

	portalTokenID, ok := data["PortalTokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PortalTokenID is invalid"))
	}
	if !httpServer.config.BlockChain.GetPortalParams(beaconHeight).IsPortalToken(portalTokenID) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PortalTokenID is not support"))
	}

	collateralTokenID := common.PRVIDStr
	collateralTokenIDParam, ok := data["CollateralTokenID"].(string)
	if ok && collateralTokenIDParam != "" {
		collateralTokenID = strings.ToLower(collateralTokenIDParam)
	}
	if !httpServer.config.BlockChain.IsSupportedTokenCollateralV3(beaconHeight, collateralTokenID) && collateralTokenID != common.PRVIDStr {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CollateralTokenID is not supported"))
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), beaconHeight)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalStateError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetAmountNeededForCustodianDepositLiquidationError, err)
	}

	portalParam := httpServer.config.BlockChain.GetPortalParams(beaconHeight)

	result, err := httpServer.portal.CalculateTopupAmountForCustodianState(stateDB, custodianAddress, portalTokenID, collateralTokenID, portalParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetAmountNeededForCustodianDepositLiquidationError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleGetAmountTopUpWaitingPorting(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	// default get best beacon height
	beaconHeight := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	beaconHeightParam, err := common.AssertAndConvertStrToNumber(data["BeaconHeight"])
	if err == nil || beaconHeightParam > 0 {
		beaconHeight = beaconHeightParam
	}

	beaconFeatureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), beaconHeight)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalRewardError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	beaconFeatureStateDB, err := statedb.NewWithPrefixTrie(beaconFeatureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.GetBeaconChainDatabase()))

	// default collateralTokenID is PRV
	collateralTokenID := common.PRVIDStr
	collateralTokenIDParam, ok := data["CollateralTokenID"].(string)
	if ok && collateralTokenIDParam != "" {
		collateralTokenID = strings.ToLower(collateralTokenIDParam)
	}
	if !httpServer.config.BlockChain.IsSupportedTokenCollateralV3(beaconHeight, collateralTokenID) && collateralTokenID != common.PRVIDStr {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CollateralTokenID is not supported"))
	}

	result, err := httpServer.blockService.GetAmountTopUpWaitingPorting(custodianAddr, collateralTokenID, beaconHeight, beaconFeatureStateDB)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetAmountTopUpWaitingPortingError, err)
	}
	return result, nil
}


func (httpServer *HttpServer) handleGetCustodianLiquidationStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	redeemID, ok := data["RedeemID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param RedeemID is invalid"))
	}

	custodianAddress, ok := data["CustodianIncAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param CustodianIncAddress is invalid"))
	}
	status, err := httpServer.blockService.GetPortalLiquidationCustodianStatus(redeemID, custodianAddress)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqRedeemStatusError, err)
	}
	return status, nil
}
