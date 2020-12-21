package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"sort"
)

func (httpServer *HttpServer) handleCreateRawTxWithCustodianDeposit(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
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
	remoteAddressesMap, ok := data["RemoteAddresses"].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata RemoteAddresses param is invalid"))
	}
	if len(remoteAddressesMap) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata RemoteAddresses must be at least one"))
	}
	remoteAddresses := make(map[string]string, 0)
	tokenIDKeys := make([]string, 0)
	for pTokenID, remoteAddress := range remoteAddressesMap {
		if !common.IsPortalToken(pTokenID) {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata public token is not supported currently"))
		}
		_, ok := remoteAddress.(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata RemoteAddresses is invalid"))
		}
		tokenIDKeys = append(tokenIDKeys, pTokenID)
	}
	sort.Strings(tokenIDKeys)
	for _, pTokenID := range tokenIDKeys {
		remoteAddresses[pTokenID] = remoteAddressesMap[pTokenID].(string)
	}

	depositedAmount, err := common.AssertAndConvertStrToNumber(data["DepositedAmount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPortalCustodianDeposit(
		metadata.PortalCustodianDepositMeta,
		incognitoAddress,
		remoteAddresses,
		depositedAmount,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithCustodianDeposit(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithCustodianDeposit(params, closeChan)
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

func (httpServer *HttpServer) handleCreateRawTxWithReqPToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	meta, _ := metadata.NewPortalRequestPTokens(
		metadata.PortalUserRequestPTokenMeta,
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

func (httpServer *HttpServer) handleGetPortalState(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	portalState, err := blockchain.InitCurrentPortalStateFromDB(beaconFeatureStateDB)
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

func (httpServer *HttpServer) handleGetPortalCustodianDepositStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	depositTxID, ok := data["DepositTxID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param DepositTxID is invalid"))
	}

	status, err := httpServer.blockService.GetCustodianDepositStatus(depositTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianDepositError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPortalReqPTokenStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

func (httpServer *HttpServer) handleCreateRawTxWithRedeemReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	meta, _ := metadata.NewPortalRedeemRequest(metadata.PortalRedeemRequestMeta, uniqueRedeemID,
		redeemTokenID, redeemAmount, redeemerIncAddressStr, remoteAddress, redeemFee)

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

func (httpServer *HttpServer) handleCustodianWithdrawRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	paymentAddress, ok := data["PaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata PaymentAddress is invalid"))
	}

	amount, err := common.AssertAndConvertStrToNumber(data["Amount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := metadata.NewPortalCustodianWithdrawRequest(
		metadata.PortalCustodianWithdrawRequestMeta,
		paymentAddress,
		amount,
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

func (httpServer *HttpServer) handleCreateAndSendCustodianWithdrawRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCustodianWithdrawRequest(params, closeChan)
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

func (httpServer *HttpServer) handleCreateRawTxWithReqUnlockCollateral(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	meta, _ := metadata.NewPortalRequestUnlockCollateral(
		metadata.PortalRequestUnlockCollateralMeta,
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

func (httpServer *HttpServer) handleGetCustodianWithdrawByTxId(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata param is invalid"))
	}

	txId, ok := data["TxId"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata txId is invalid"))
	}

	result, err := httpServer.portal.GetCustodianWithdrawByTxId(txId)

	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianWithdrawError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleGetPortalReqUnlockCollateralStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

func (httpServer *HttpServer) handleGetPortalReqRedeemStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

func (httpServer *HttpServer) handleGetPortalReqRedeemByTxIDStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

func (httpServer *HttpServer) handleGetCustodianLiquidationStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

func (httpServer *HttpServer) handleCreateRawTxWithReqWithdrawRewardPortal(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianAddressStr is invalid"))
	}
	tokenID := new(common.Hash)
	tokenIDStr, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))

	}
	var err error
	tokenID, err = tokenID.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata TokenID is invalid"))
	}

	meta, _ := metadata.NewPortalRequestWithdrawReward(
		metadata.PortalRequestWithdrawRewardMeta,
		incognitoAddress,
		*tokenID)

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

func (httpServer *HttpServer) handleCreateAndSendTxWithReqWithdrawRewardPortal(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithReqWithdrawRewardPortal(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalReward(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	incognitoAddress, ok := data["IncognitoAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("IncognitoAddress is invalid"))
	}

	latestBeaconHeight := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	beaconFeatureStateRootHash, err := httpServer.config.BlockChain.GetBeaconFeatureRootHash(httpServer.config.BlockChain.GetBeaconBestState(), latestBeaconHeight)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalRewardError, fmt.Errorf("Can't found FeatureStateRootHash of beacon height %+v, error %+v", incognitoAddress, err))
	}
	beaconFeatureStateDB, err := statedb.NewWithPrefixTrie(beaconFeatureStateRootHash, statedb.NewDatabaseAccessWarper(httpServer.GetBeaconChainDatabase()))

	portalState, err := blockchain.InitCurrentPortalStateFromDB(beaconFeatureStateDB)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPortalRewardError, err)
	}

	rewardAmount := map[string]uint64{}
	for _, cus := range portalState.CustodianPoolState {
		if cus.GetIncognitoAddress() == incognitoAddress {
			rewardAmount = cus.GetRewardAmount()
			break
		}
	}
	return rewardAmount, nil
}

func (httpServer *HttpServer) handleGetRequestWithdrawPortalRewardStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	status, err := httpServer.blockService.GetPortalRequestWithdrawRewardStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetRequestWithdrawRewardStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetRewardFeature(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param array must be at least one"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	featureName, ok := data["FeatureName"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param FeatureName is invalid"))
	}
	epoch, err := common.AssertAndConvertStrToNumber(data["Epoch"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	if epoch < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param Epoch must be greater or than 1"))
	}

	result, err := httpServer.blockService.GetRewardFeatureByFeatureName(featureName, uint64(epoch))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetRewardFeatureByFeatureNameError, err)
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithReqMatchingRedeem(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	meta, _ := metadata.NewPortalReqMatchingRedeem(
		metadata.PortalReqMatchingRedeemMeta,
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

func (httpServer *HttpServer) handleGetReqMatchingRedeemByTxIDStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
