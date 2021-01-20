package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/basemeta"
	portalMeta "github.com/incognitochain/incognito-chain/portal/metadata"
	"github.com/incognitochain/incognito-chain/portal/portalprocess"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"sort"
	"strings"
)

/*
====== Custodian deposit collateral (PRV)
*/
func (httpServer *HttpServer) handleCreateRawTxWithCustodianDeposit(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
		if !httpServer.config.BlockChain.GetPortalParams(0).IsPortalToken(pTokenID) {
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

	meta, _ := portalMeta.NewPortalCustodianDeposit(
		basemeta.PortalCustodianDepositMeta,
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
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

func (httpServer *HttpServer) handleGetPortalCustodianDepositStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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


/*
====== Custodian deposit collateral v3 (ETH/ERC20)
*/
func (httpServer *HttpServer) handleCreateRawTxWithCustodianDepositV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	meta, err := portalMeta.NewPortalCustodianDepositV3FromMap(data, httpServer.config.BlockChain.GetPortalParams(0))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

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

func (httpServer *HttpServer) handleCreateAndSendTxWithCustodianDepositV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithCustodianDepositV3(params, closeChan)
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

func (httpServer *HttpServer) handleGetPortalCustodianDepositStatusV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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
	depositTxID, ok := data["DepositTxID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param DepositTxID is invalid"))
	}

	status, err := httpServer.blockService.GetCustodianDepositStatusV3(depositTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianDepositV3Error, err)
	}
	return status, nil
}


/*
====== Custodian request withdraw collateral (PRV)
*/
func (httpServer *HttpServer) handleCreateRawTxWithCustodianWithdrawRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	paymentAddress, ok := data["CustodianIncAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianIncAddress is invalid"))
	}

	amount, err := common.AssertAndConvertStrToNumber(data["Amount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := portalMeta.NewPortalCustodianWithdrawRequest(
		basemeta.PortalCustodianWithdrawRequestMeta,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithCustodianWithdrawRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithCustodianWithdrawRequest(params, closeChan)
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

func (httpServer *HttpServer) handleGetCustodianWithdrawRequestStatusByTxId(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	result, err := httpServer.portal.GetCustodianWithdrawRequestStatusByTxId(txId)

	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianWithdrawError, err)
	}

	return result, nil
}


/*
====== Custodian request withdraw collateral v3 (ETH/ERC20)
*/
func (httpServer *HttpServer) handleCreateRawTxWithCustodianWithdrawRequestV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	custodianIncAddress, ok := data["CustodianIncAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianIncAddress is invalid"))
	}

	custodianExtAddress, ok := data["CustodianExtAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianExtAddress is invalid"))
	}
	// remove 0x prefix (if there)
	custodianExtAddress = common.Remove0xPrefix(custodianExtAddress)

	externalTokenID, ok := data["ExternalTokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata CustodianIncAddress is invalid"))
	}
	externalTokenID = strings.ToLower(common.Remove0xPrefix(externalTokenID))

	amount, err := common.AssertAndConvertStrToNumber(data["Amount"])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	meta, _ := portalMeta.NewPortalCustodianWithdrawRequestV3(
		basemeta.PortalCustodianWithdrawRequestMetaV3,
		custodianIncAddress,
		custodianExtAddress,
		externalTokenID,
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

func (httpServer *HttpServer) handleCreateAndSendTxWithCustodianWithdrawRequestV3(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	data, err := httpServer.handleCreateRawTxWithCustodianWithdrawRequestV3(params, closeChan)
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

func (httpServer *HttpServer) handleGetCustodianWithdrawRequestStatusV3ByTxId(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

	result, err := httpServer.portal.GetCustodianWithdrawRequestStatusV3ByTxId(txId)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetCustodianWithdrawError, err)
	}

	return result, nil
}


/*
====== Portal rewards
*/
func (httpServer *HttpServer) handleCreateRawTxWithReqWithdrawRewardPortal(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	meta, _ := portalMeta.NewPortalRequestWithdrawReward(
		basemeta.PortalRequestWithdrawRewardMeta,
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
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

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

func (httpServer *HttpServer) handleGetRequestWithdrawPortalRewardStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	status, err := httpServer.blockService.GetPortalRequestWithdrawRewardStatus(reqTxID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetRequestWithdrawRewardStatusError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPortalReward(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

	portalState, err := portalprocess.InitCurrentPortalStateFromDB(beaconFeatureStateDB)
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

func (httpServer *HttpServer) handleGetRewardFeature(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
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

/*
====== Check external txHash submitted or not
*/
func (httpServer *HttpServer) handleCheckPortalExternalHashSubmitted(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param[0] must be a map[string]interface"))
	}

	issued, err := httpServer.blockService.CheckPortalExternalTxSubmitted(data)
	if err != nil {
		return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return issued, nil
}
