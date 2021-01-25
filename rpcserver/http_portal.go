package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	portalMeta "github.com/incognitochain/incognito-chain/portal/metadata"
	"github.com/incognitochain/incognito-chain/portal/portalprocess"
	"github.com/incognitochain/incognito-chain/portalv4/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

func (httpServer *HttpServer) checkEnablePortalV3() *rpcservice.RPCError {
	if !httpServer.config.ChainParams.EnablePortalV3 {
		return rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("This RPC wasn't supported on this chain version"))
	}
	return nil
}

/*
====== Portal state
*/
func (httpServer *HttpServer) handleGetPortalState(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	beaconHeight, err := common.AssertAndConvertStrToNumber(data["BeaconHeight"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	//beaconFeatureStateDB := httpServer.config.BlockChain.GetBeaconBestState().GetCopiedFeatureStateDB()
	beaconFeatureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalStateError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	beaconFeatureStateDB, err := statedb.NewWithPrefixTrie(beaconFeatureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))

	portalState, err := portalprocess.InitCurrentPortalStateFromDB(beaconFeatureStateDB)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalStateError, err)
	}

	beaconBlocks, err := httpServer.config.BlockChain.GetBeaconBlockByHeight(uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalStateError, err)
	}
	beaconBlock := beaconBlocks[0]

	type CurrentPortalState struct {
		WaitingPortingRequests     map[string]*statedb.WaitingPortingRequest `json:"WaitingPortingRequests"`
		WaitingRedeemRequests      map[string]*statedb.RedeemRequest         `json:"WaitingRedeemRequests"`
		MatchedRedeemRequests      map[string]*statedb.RedeemRequest         `json:"MatchedRedeemRequests"`
		CustodianPool              map[string]*statedb.CustodianState        `json:"CustodianPool"`
		FinalExchangeRatesState    *statedb.FinalExchangeRatesState          `json:"FinalExchangeRatesState"`
		LiquidationPool            map[string]*statedb.LiquidationPool       `json:"LiquidationPool"`
		LockedCollateralForRewards *statedb.LockedCollateralState            `json:"LockedCollateralForRewards"`
		BeaconTimeStamp            int64                                     `json:"BeaconTimeStamp"`
	}

	result := CurrentPortalState{
		BeaconTimeStamp:            beaconBlock.Header.Timestamp,
		WaitingPortingRequests:     portalState.WaitingPortingRequests,
		WaitingRedeemRequests:      portalState.WaitingRedeemRequests,
		MatchedRedeemRequests:      portalState.MatchedRedeemRequests,
		CustodianPool:              portalState.CustodianPoolState,
		FinalExchangeRatesState:    portalState.FinalExchangeRatesState,
		LiquidationPool:            portalState.LiquidationPool,
		LockedCollateralForRewards: portalState.LockedCollateralForRewards,
	}
	return result, nil
}

/*
====== Porting request
*/
func (httpServer *HttpServer) handleGetPortingRequestFees(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	valuePToken, err := common.AssertAndConvertStrToNumber(data["ValuePToken"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	beaconHeight, err := common.AssertAndConvertStrToNumber(data["BeaconHeight"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	tokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}
	if !httpServer.config.BlockChain.GetPortalParams(beaconHeight).IsPortalToken(tokenID) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID should be a portal token"))
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestFeesError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestFeesError, err)
	}
	// get final exchange rate from db
	finalExchangeRates, err := statedb.GetFinalExchangeRatesState(stateDB)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestFeesError, fmt.Errorf("Can't get final exchange rate of beacon height %+v, error %+v", beaconHeight, err))
	}

	portalParams := httpServer.config.BlockChain.GetPortalParams(uint64(beaconHeight))

	result, err := httpServer.portal.GetPortingFees(finalExchangeRates, tokenID, uint64(valuePToken), portalParams)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestFeesError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPortingRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	uniqueRegisterId, ok := data["UniqueRegisterId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata UniqueRegisterId is invalid"))
	}

	incogAddressStr, ok := data["IncogAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata IncogAddressStr is invalid"))
	}

	pTokenId, ok := data["PTokenId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PTokenId is invalid"))
	}

	if !httpServer.config.BlockChain.GetPortalParams(0).IsPortalToken(pTokenId) {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata public token is not supported currently"))
	}

	registerAmount, err := common.AssertAndConvertStrToNumber(data["RegisterAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	portingFee, err := common.AssertAndConvertStrToNumber(data["PortingFee"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	//check exchange rates
	meta, _ := portalMeta.NewPortalUserRegister(
		uniqueRegisterId,
		incogAddressStr,
		pTokenId,
		registerAmount,
		portingFee,
		basemeta.PortalRequestPortingMetaV3,
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

func (httpServer *HttpServer) handleCreateAndSendTxPortingRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithPortingRequest(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortingRequestStatusByTxID(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	txHash, ok := data["TxHash"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PortingRequestId is invalid"))
	}

	result, err := httpServer.portal.GetPortingRequestByByTxID(txHash)

	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleGetPortingRequestStatusByPortingId(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	portingId, ok := data["PortingId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Porting Id is invalid"))
	}

	result, err := httpServer.portal.GetPortingRequestByByPortingId(portingId)

	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortingRequestError, err)
	}

	return result, nil
}

/*
====== Request ptoken
*/
func (httpServer *HttpServer) handleCreateRawTxWithReqPToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 5"))
	}

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}
	uniquePortingID, ok := data["UniquePortingID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata UniquePortingID is invalid"))
	}
	tokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}
	//tokenIDHash, err := new(common.Hash).NewHashFromStr(tokenID)
	//if err != nil {
	//	return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Can not new TokenIDHash from TokenID"))
	//}
	incognitoAddress, ok := data["IncogAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata IncogAddressStr is invalid"))
	}
	portingAmount, err := common.AssertAndConvertStrToNumber(data["PortingAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	portingProof, ok := data["PortingProof"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PortingProof param is invalid"))
	}

	meta, _ := portalMeta.NewPortalRequestPTokens(
		basemeta.PortalUserRequestPTokenMeta,
		uniquePortingID,
		tokenID,
		incognitoAddress,
		portingAmount,
		portingProof,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithReqPToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithReqPToken(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalReqPTokenStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	status, err := httpServer.blockService.GetPortalReqPTokenStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqPTokenStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithReqPTokenV4(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 5"))
	}

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}
	tokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}
	//tokenIDHash, err := new(common.Hash).NewHashFromStr(tokenID)
	//if err != nil {
	//	return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Can not new TokenIDHash from TokenID"))
	//}
	incognitoAddress, ok := data["IncogAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata IncogAddressStr is invalid"))
	}

	portingProof, ok := data["PortingProof"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PortingProof param is invalid"))
	}

	meta, _ := metadata.NewPortalRequestPTokensV4(
		basemeta.PortalUserRequestPTokenMeta,
		tokenID,
		incognitoAddress,
		portingProof,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
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

func (httpServer *HttpServer) handleCreateAndSendTxWithReqPTokenV4(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithReqPTokenV4(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalReqPTokenStatusV4(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	status, err := httpServer.blockService.GetPortalReqPTokenStatusV4(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqPTokenStatusV4Error, err)
	}
	return status, nil
}

/*
====== Redeem request
*/
func (httpServer *HttpServer) handleCreateRawTxWithRedeemReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)

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
	tokenParamsRaw, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param metadata is invalid"))
	}

	uniqueRedeemID, ok := tokenParamsRaw["UniqueRedeemID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("UniqueRedeemID is invalid"))
	}

	redeemTokenID, ok := tokenParamsRaw["RedeemTokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("RedeemTokenID is invalid"))
	}

	redeemAmount, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["RedeemAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	redeemFee, err := common.AssertAndConvertStrToNumber(tokenParamsRaw["RedeemFee"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	redeemerIncAddressStr, ok := tokenParamsRaw["RedeemerIncAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("RedeemerIncAddressStr is invalid"))
	}

	remoteAddress, ok := tokenParamsRaw["RemoteAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("RemoteAddress is invalid"))
	}

	redeemerExternalAddress, ok := tokenParamsRaw["RedeemerExternalAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("RedeemerExternalAddress is invalid"))
	}
	if len(redeemerExternalAddress) >= 2 && redeemerExternalAddress[0] == '0' && (redeemerExternalAddress[1] == 'x' || redeemerExternalAddress[1] == 'X') {
		redeemerExternalAddress = redeemerExternalAddress[2:]
	}

	meta, _ := portalMeta.NewPortalRedeemRequest(basemeta.PortalRedeemRequestMetaV3, uniqueRedeemID,
		redeemTokenID, redeemAmount, redeemerIncAddressStr, remoteAddress, redeemFee, redeemerExternalAddress)

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

func (httpServer *HttpServer) handleCreateAndSendTxWithRedeemReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithRedeemReq(params, closeChan)
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

func (httpServer *HttpServer) handleGetReqRedeemStatusByRedeemID(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	status, err := httpServer.blockService.GetPortalRedeemReqStatus(redeemID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqRedeemStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetReqRedeemStatusByTxID(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	status, err := httpServer.blockService.GetPortalRedeemReqByTxIDStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqRedeemStatusError, err)
	}
	return status, nil
}

/*
====== Request matching waiting redeem request
*/
func (httpServer *HttpServer) handleCreateRawTxWithReqMatchingRedeem(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 5"))
	}

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}
	incognitoAddress, ok := data["CustodianAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata IncognitoAddress is invalid"))
	}
	redeemID, ok := data["RedeemID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata RemoteAddresses param is invalid"))
	}

	meta, _ := portalMeta.NewPortalReqMatchingRedeem(
		basemeta.PortalReqMatchingRedeemMeta,
		incognitoAddress,
		redeemID,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithReqMatchingRedeem(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithReqMatchingRedeem(params, closeChan)
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

func (httpServer *HttpServer) handleGetReqMatchingRedeemStatusByTxID(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	status, err := httpServer.blockService.GetReqMatchingRedeemByTxIDStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqMatchingRedeemStatusError, err)
	}
	return status, nil
}

/*
====== Request unlock collaterals
*/
func (httpServer *HttpServer) handleCreateRawTxWithReqUnlockCollateral(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 5"))
	}

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}
	uniqueRedeemID, ok := data["UniqueRedeemID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata UniqueRedeemID is invalid"))
	}
	tokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}
	_, err := new(common.Hash).NewHashFromStr(tokenID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Can not new TokenIDHash from TokenID"))
	}
	incognitoAddress, ok := data["CustodianAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianAddressStr is invalid"))
	}
	redeemAmount, err := common.AssertAndConvertStrToNumber(data["RedeemAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	redeemProof, ok := data["RedeemProof"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata RedeemProof param is invalid"))
	}

	meta, _ := portalMeta.NewPortalRequestUnlockCollateral(
		basemeta.PortalRequestUnlockCollateralMetaV3,
		uniqueRedeemID,
		tokenID,
		incognitoAddress,
		redeemAmount,
		redeemProof,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithReqUnlockCollateral(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithReqUnlockCollateral(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalReqUnlockCollateralStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	status, err := httpServer.blockService.GetPortalReqUnlockCollateralStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqUnlockCollateralStatusError, err)
	}
	return status, nil
}

/*
====== Portal exchange rates
*/
func (httpServer *HttpServer) handleCreateRawTxWithPortalExchangeRate(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)
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

	var exchangeRate = make([]*portalMeta.ExchangeRateInfo, 0)
	exchangeRateMap, ok := data["Rates"].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Rates is invalid"))
	}
	if len(exchangeRateMap) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Rates is invalid"))
	}

	beaconHeight := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	for tokenID, value := range exchangeRateMap {
		tokenID = strings.ToLower(common.Remove0xPrefix(tokenID))
		if !httpServer.config.BlockChain.IsPortalExchangeRateToken(beaconHeight, tokenID) {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("TokenID is not portal exchange rate token"))
		}

		amount, err := common.AssertAndConvertStrToNumber(value)
		if err != nil || amount == 0 {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Value of exchange rate is invalid %v", err))
		}

		exchangeRate = append(
			exchangeRate,
			&portalMeta.ExchangeRateInfo{
				PTokenID: tokenID,
				Rate:     amount,
			})
	}

	meta, _ := portalMeta.NewPortalExchangeRates(
		basemeta.PortalExchangeRatesMeta,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithPortalExchangeRate(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithPortalExchangeRate(params, closeChan)
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
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("metadata BeaconHeight is invalid %v", err))
	}

	amount, err := common.AssertAndConvertStrToNumber(data["Amount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("metadata Amount is invalid %v", err))
	}

	tokenIDFrom := ""
	tokenIDFromParam, ok := data["TokenIDFrom"].(string)
	if ok {
		tokenIDFrom = strings.ToLower(tokenIDFromParam)
	}

	// default convert to USDT
	tokenIDTo := ""
	tokenIDToParam, ok := data["TokenIDTo"].(string)
	if ok {
		tokenIDTo = strings.ToLower(tokenIDToParam)
	}

	featureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), beaconHeight)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ConvertExchangeRatesError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", beaconHeight, err))
	}
	stateDB, err := statedb.NewWithPrefixTrie(featureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.config.BlockChain.GetBeaconChainDatabase()))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ConvertExchangeRatesError, fmt.Errorf("Can't get statedb of beacon height %+v, error %+v", beaconHeight, err))
	}
	// get final exchange rate from db
	finalExchangeRates, err := statedb.GetFinalExchangeRatesState(stateDB)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ConvertExchangeRatesError, fmt.Errorf("Can't get final exchange rate of beacon height %+v, error %+v", beaconHeight, err))
	}

	portalParam := httpServer.blockService.BlockChain.GetPortalParams(beaconHeight)

	result, err := httpServer.portal.ConvertExchangeRates(finalExchangeRates, portalParam, amount, tokenIDFrom, tokenIDTo)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ConvertExchangeRatesError, err)
	}

	return result, nil
}

/*
====== Custodian unlock collaterals
*/
func (httpServer *HttpServer) handleCreateRawTxWithPortalCusUnlockOverRateCollaterals(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least 5"))
	}

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}

	custodianAddress, ok := data["CustodianAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata SenderAddress is invalid"))
	}

	tokenID, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}
	_, err := new(common.Hash).NewHashFromStr(tokenID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata Can not new TokenIDHash from TokenID"))
	}

	meta, _ := portalMeta.NewPortalUnlockOverRateCollaterals(
		basemeta.PortalUnlockOverRateCollateralsMeta,
		custodianAddress,
		tokenID,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithPortalCusUnlockOverRateCollaterals(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithPortalCusUnlockOverRateCollaterals(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalReqUnlockOverRateCollateralStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	status, err := httpServer.blockService.GetPortalReqUnlockOverRateCollateralStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetReqUnlockCollateralStatusError, err)
	}
	return status, nil
}
