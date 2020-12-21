package rpcserver

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/pkg/errors"
)

func (httpServer *HttpServer) handleCreateIssuingRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	constructor := metaConstructors[createAndSendIssuingRequest]
	return httpServer.createRawTxWithMetadata(params, closeChan, constructor)
}

func (httpServer *HttpServer) handleSendIssuingRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	return httpServer.handleSendRawTransaction(params, closeChan)
}

// handleCreateAndSendIssuingRequest for user to buy Constant (using USD) or BANK token (using USD/ETH) from DCB
func (httpServer *HttpServer) handleCreateAndSendIssuingRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	return httpServer.createAndSendTxWithMetadata(
		params,
		closeChan,
		(*HttpServer).handleCreateIssuingRequest,
		(*HttpServer).handleSendIssuingRequest,
	)
}

func (httpServer *HttpServer) handleCreateRawTxWithContractingReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 5 elements"))
	}

	// check privacy mode param
	if len(arrayParams) > 6 {
		privacyTemp, ok := arrayParams[6].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be valid"))
		}
		hasPrivacyToken := int(privacyTemp) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}

	senderPrivateKeyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("private key is invalid"))
	}

	tokenParamsRaw, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token param is invalid"))
	}

	tokenReceivers, ok := tokenParamsRaw["TokenReceivers"].(interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token receivers is invalid"))
	}

	tokenID, ok := tokenParamsRaw["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token ID is invalid"))
	}

	meta, err := rpcservice.NewContractingRequestMetadata(senderPrivateKeyParam, tokenReceivers, tokenID)
	if err != nil {
		return nil, err
	}

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

func (httpServer *HttpServer) handleCreateAndSendContractingRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithContractingReq(params, closeChan)
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

func (httpServer *HttpServer) handleCreateRawTxWithBurningReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	return processBurningReq(
		metadata.BurningRequestMetaV2,
		params,
		closeChan,
		httpServer,
	)
}

func (httpServer *HttpServer) handleCreateAndSendBurningRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithBurningReq(params, closeChan)
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

func (httpServer *HttpServer) handleCreateRawTxWithIssuingETHReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 5 elements"))
	}

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	meta, err := metadata.NewIssuingETHRequestFromMap(data)
	if err != nil {
		rpcErr := rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		Logger.log.Error(rpcErr)
		return nil, rpcErr
	}

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

func (httpServer *HttpServer) handleCreateAndSendTxWithIssuingETHReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithIssuingETHReq(params, closeChan)
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

func (httpServer *HttpServer) handleCheckETHHashIssued(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}
	data := arrayParams[0].(map[string]interface{})

	issued, err := httpServer.blockService.CheckETHHashIssued(data)
	if err != nil {
		return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return issued, nil
}

func (httpServer *HttpServer) handleGetAllBridgeTokens(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	allBridgeTokens, err := httpServer.blockService.GetAllBridgeTokens()
	if err != nil {
		return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return allBridgeTokens, nil
}

func (httpServer *HttpServer) handleGetETHHeaderByHash(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}
	ethBlockHash := arrayParams[0].(string)

	ethHeader, err := rpcservice.GetETHHeaderByHash(ethBlockHash)
	if err != nil {
		return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return ethHeader, nil
}

func (httpServer *HttpServer) handleGetBridgeReqWithStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}
	data := arrayParams[0].(map[string]interface{})

	status, err := httpServer.blockService.GetBridgeReqWithStatus(data["TxReqID"].(string))
	if err != nil {
		return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return status, nil
}

func processBurningReq(
	burningMetaType int,
	params interface{},
	closeChan <-chan struct{},
	httpServer *HttpServer,
) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 5 elements"))
	}

	if len(arrayParams) > 6 {
		privacyTemp, ok := arrayParams[6].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("The privacy mode must be valid "))
		}
		hasPrivacyToken := int(privacyTemp) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}

	senderPrivateKeyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("private key is invalid"))
	}

	tokenParamsRaw, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token param is invalid"))
	}

	tokenReceivers, ok := tokenParamsRaw["TokenReceivers"].(interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token receivers is invalid"))
	}

	tokenID, ok := tokenParamsRaw["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token ID is invalid"))
	}

	tokenName, ok := tokenParamsRaw["TokenName"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token name is invalid"))
	}

	remoteAddress, ok := tokenParamsRaw["RemoteAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("remote address is invalid"))
	}

	meta, err := rpcservice.NewBurningRequestMetadata(senderPrivateKeyParam, tokenReceivers, tokenID, tokenName, remoteAddress, burningMetaType, httpServer.GetBlockchain(), httpServer.GetBlockchain().BeaconChain.CurrentHeight())
	if err != nil {
		return nil, err
	}

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

func (httpServer *HttpServer) handleCreateRawTxWithBurningForDepositToSCReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	return processBurningReq(
		metadata.BurningForDepositToSCRequestMetaV2,
		params,
		closeChan,
		httpServer,
	)
}

func (httpServer *HttpServer) handleCreateAndSendBurningForDepositToSCRequest(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithBurningForDepositToSCReq(params, closeChan)
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
