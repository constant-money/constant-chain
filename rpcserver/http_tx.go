package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/incognitochain/incognito-chain/wallet"
)

// handleCreateTransaction handles createtransaction commands.
func (httpServer *HttpServer) handleCreateRawTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	txHash, txBytes, txShardID, err := httpServer.txService.CreateRawTransaction(createRawTxParam, nil)
	if err != nil {
		// return hex for a new tx
		return nil, err
	}

	result := jsonresult.NewCreateTransactionResult(txHash, common.EmptyString, txBytes, txShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawConvertVer1ToVer2Transaction(params interface{}, closeChan <-chan struct{}) (*jsonresult.CreateTransactionResult, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateRawConvertVer1ToVer2Transaction params: %+v", params)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxSwitchVer1ToVer2Param(params)
	if errNewParam != nil {
		fmt.Println("Bean error")
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	txHash, txBytes, txShardID, err := httpServer.txService.CreateRawConvertVer1ToVer2Transaction(createRawTxParam)
	if err != nil {
		fmt.Println("TxHash, TxBytes, txShardID error")
		// return hex for a new tx
		return nil, err
	}

	result := jsonresult.NewCreateTransactionResult(txHash, common.EmptyString, txBytes, txShardID)
	Logger.log.Debugf("handleCreateRawConvertVer1ToVer2Transaction result: %+v", result)

	return &result, nil
}

// handleSendTransaction implements the sendtransaction command.
// Parameter #1—a serialized transaction to broadcast
// Parameter #2–whether to allow high fees
// Result—a TXID or error Message
func (httpServer *HttpServer) handleSendRawTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	base58CheckData, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("base58 check data is invalid"))
	}

	txMsg, txHash, LastBytePubKeySender, err := httpServer.txService.SendRawTransaction(base58CheckData)
	if err != nil {
		return nil, err
	}

	err2 := httpServer.config.Server.PushMessageToAll(txMsg)
	if err2 == nil {
		Logger.log.Info("handleSendRawTransaction broadcast message to all successfully")
		httpServer.config.TxMemPool.MarkForwardedTransaction(*txHash)
	} else {
		Logger.log.Errorf("handleSendRawTransaction broadcast message to all with error %+v", err2)
	}

	result := jsonresult.NewCreateTransactionResult(txHash, common.EmptyString, nil, common.GetShardIDFromLastByte(LastBytePubKeySender))
	return result, nil
}

func (httpServer *HttpServer) handleCreateConvertCoinVer1ToVer2Transaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateConvertCoinVer1ToVer2Transaction params: %+v", params)
	tx, err := httpServer.handleCreateRawConvertVer1ToVer2Transaction(params, closeChan)
	if err != nil {
		Logger.log.Debugf("handleCreateConvertCoinVer1ToVer2Transaction result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		Logger.log.Debugf("handleCreateConvertCoinVer1ToVer2Transaction result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, tx.ShardID)
	Logger.log.Debugf("handleCreateConvertCoinVer1ToVer2Transaction result: %+v", result)
	return result, nil
}

// handleCreateAndSendTx - RPC creates transaction and send to network
func (httpServer *HttpServer) handleCreateAndSendTx(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTransaction(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, tx.ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleGetTransactionHashByReceiver(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	paymentAddress, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payment address"))
	}

	result, err := httpServer.txService.GetTransactionHashByReceiver(paymentAddress)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	return result, nil
}

// Get tx hash by receiver in paging fashion
func (httpServer *HttpServer) handleGetTransactionHashByReceiverV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 3 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 element"))
	}

	paymentAddress, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payment address"))
	}

	skip, ok := arrayParams[1].(float64)
	if !ok || skip < 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("skip"))
	}

	limit, ok := arrayParams[2].(float64)
	if !ok || limit < 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("limit"))
	}

	txHashsByShards, err := httpServer.txService.GetTransactionHashByReceiverV2(paymentAddress, uint(skip), uint(limit))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	txHashs := []common.Hash{}
	for _, txHashsByShard := range txHashsByShards {
		txHashs = append(txHashs, txHashsByShard...)
	}
	result := struct {
		Skip uint
		Limit uint
		TxHashs []common.Hash
	}{
		uint(skip),
		uint(limit),
		txHashs,
	}
	return result, nil
}

func (httpServer *HttpServer) handleGetTransactionByReceiver(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramsArray := common.InterfaceSlice(params)
	keys, ok := paramsArray[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}

	// create a key set
	keySet := incognitokey.KeySet{}

	// get keyset only contain readonly-key by deserializing
	readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
	if ok {
		readonlyKey, err := wallet.Base58CheckDeserialize(readonlyKeyStr)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
		keySet.ReadonlyKey = readonlyKey.KeySet.ReadonlyKey
	}

	// get keyset only contain payment address by deserializing
	paymentAddressStr, ok := keys["PaymentAddress"].(string)
	if ok {
		paymentAddress, err := wallet.Base58CheckDeserialize(paymentAddressStr)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
		keySet.PaymentAddress = paymentAddress.KeySet.PaymentAddress
	}

	result, err := httpServer.txService.GetTransactionByReceiver(keySet)

	return result, err
}

func (httpServer *HttpServer) handleGetTransactionByReceiverV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramsArray := common.InterfaceSlice(params)
	keys, ok := paramsArray[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}

	// create a key set
	keySet := incognitokey.KeySet{}

	// get keyset only contain readonly-key by deserializing
	readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
	if ok {
		readonlyKey, err := wallet.Base58CheckDeserialize(readonlyKeyStr)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
		keySet.ReadonlyKey = readonlyKey.KeySet.ReadonlyKey
	}

	// get keyset only contain payment address by deserializing
	paymentAddressStr, ok := keys["PaymentAddress"].(string)
	if ok {
		paymentAddress, err := wallet.Base58CheckDeserialize(paymentAddressStr)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
		keySet.PaymentAddress = paymentAddress.KeySet.PaymentAddress
	}

	// tokenID
	tokenID := common.PRVIDStr
	tokenIDParam, ok := keys["TokenID"].(string)
	if ok && tokenIDParam != "" {
		tokenID = tokenIDParam
	}
	tokenIDHash, err1 := common.Hash{}.NewHashFromStr(tokenID)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("TokenID is invalid"))
	}

	skip, ok := keys["Skip"].(float64)
	if !ok || skip < 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("skip"))
	}

	limit, ok := keys["Limit"].(float64)
	if !ok || limit < 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("limit"))
	}
	receivedTxsList, total, err := httpServer.txService.GetTransactionByReceiverV2(keySet, uint(skip), uint(limit), *tokenIDHash)
	if err != nil {
		return nil, err
	}
	result := struct {
		Total uint
		Skip uint
		Limit uint
		ReceivedTransactions []jsonresult.ReceivedTransactionV2
	}{
		total,
		uint(skip),
		uint(limit),
		receivedTxsList.ReceivedTransactions,
	}
	return result, nil
}

// Get transaction by Hash
func (httpServer *HttpServer) handleGetTransactionByHash(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	// param #1: transaction Hash
	txHashStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Tx hash is invalid"))
	}
	return httpServer.txService.GetTransactionByHash(txHashStr)
}

// handleGetListPrivacyCustomTokenBalance - return list privacy token + balance for one account payment address
func (httpServer *HttpServer) handleGetListPrivacyCustomTokenBalance(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	privateKey, ok := arrayParams[0].(string)
	if len(privateKey) == 0 || !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param is invalid"))
	}

	result, err := httpServer.txService.GetListPrivacyCustomTokenBalance(privateKey)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// #1 param: privateKey string
// #2 param: tokenID
// handleGetListPrivacyCustomTokenBalance - return list privacy token + balance for one account payment address
func (httpServer *HttpServer) handleGetBalancePrivacyCustomToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}

	privateKey, ok := arrayParams[0].(string)
	if len(privateKey) == 0 || !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("private key is invalid"))
	}

	tokenID, ok := arrayParams[1].(string)
	if len(tokenID) == 0 || !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tokenID is invalid"))
	}

	totalValue, err2 := httpServer.txService.GetBalancePrivacyCustomToken(privateKey, tokenID)
	if err2 != nil {
		return nil, err2
	}

	return totalValue, nil
}

func (httpServer *HttpServer) handleListUnspentOutputTokens(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 3 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	var min, max int

	if paramsArray[0] != nil {
		minParam, ok := paramsArray[0].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min param is invalid"))
		}
		min = int(minParam)
	}

	if paramsArray[1] != nil {
		maxParam, ok := paramsArray[1].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("max param is invalid"))
		}
		max = int(maxParam)
	}
	_ = min
	_ = max

	listKeyParams := common.InterfaceSlice(paramsArray[2])
	if listKeyParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	result, err := httpServer.outputCoinService.ListUnspentOutputTokensByKey(listKeyParams)
	if err != nil {
		return nil, err
	}

	return result, nil
}


// handlePrivacyCustomTokenDetail - return list tx which relate to privacy custom token by token id
func (httpServer *HttpServer) handlePrivacyCustomTokenDetail(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("param must be an array at least 1 element"))
	}

	tokenIDTemp, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("tokenID is invalid"))
	}

	txs, tokenData, err := httpServer.txService.PrivacyCustomTokenDetail(tokenIDTemp)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	result := jsonresult.CustomToken{
		ListTxs:            []string{},
		ID:                 tokenIDTemp,
		Name:               tokenData.PropertyName,
		IsPrivacy:          true,
		Symbol:             tokenData.PropertySymbol,
		InitiatorPublicKey: "",
	}

	for _, tx := range txs {
		result.ListTxs = append(result.ListTxs, tx.String())
		if result.Name == "" {
			tx, err2 := httpServer.txService.GetTransactionByHash(tx.String())
			if err2 != nil {
				Logger.log.Error(err)
			} else {
				if tx.PrivacyCustomTokenName != "" {
					result.Name = tx.PrivacyCustomTokenName
					result.Symbol = tx.PrivacyCustomTokenSymbol
				}
			}
		}
	}

	return result, nil
}

// handleRandomCommitments - from input of outputcoin, random to create data for create new tx
func (httpServer *HttpServer) handleRandomCommitments(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 element"))
	}

	// #1: payment address
	paymentAddressStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("PaymentAddress is invalid"))
	}

	// #2: available inputCoin from old outputcoin
	outputs, ok := arrayParams[1].([]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("outputs is invalid"))
	}
	if len(outputs) == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("len of outputs must be greater than zero"))
	}

	//#3 - tokenID - default PRV
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 2 {
		tokenIDTemp, ok := arrayParams[2].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tokenID is invalid"))
		}
		tokenID, err = common.Hash{}.NewHashFromStr(tokenIDTemp)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
		}
	}

	commitmentIndexs, myCommitmentIndexs, commitments, err2 := httpServer.txService.RandomCommitments(paymentAddressStr, outputs, tokenID)
	if err2 != nil {
		return nil, err2
	}

	result := jsonresult.NewRandomCommitmentResult(commitmentIndexs, myCommitmentIndexs, commitments)
	return result, nil
}

// handleRandomCommitmentsAndPublicKey - returns a list of random commitments, public keys and indices for creating txver2
func (httpServer *HttpServer) handleRandomCommitmentsAndPublicKeys(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 element"))
	}

	// #1: payment address
	paymentAddressStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("PaymentAddress is invalid"))
	}

	// #2: Number of commitments
	numOutputs, ok := arrayParams[1].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Number of commitments is invalid"))
	}

	//#3 - tokenID - default PRV
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 2 {
		tokenIDTemp, ok := arrayParams[2].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tokenID is invalid"))
		}
		tokenID, err = common.Hash{}.NewHashFromStr(tokenIDTemp)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
		}
	}

	commitmentIndices, publicKeys, commitments, assetTags, err2 := httpServer.txService.RandomCommitmentsAndPublicKeys(paymentAddressStr, int(numOutputs), tokenID)
	if err2 != nil {
		return nil, err2
	}

	result := jsonresult.NewRandomCommitmentAndPublicKeyResult(commitmentIndices, publicKeys, commitments, assetTags)
	return result, nil
}

// handleListSerialNumbers - return list all serialnumber in shard for token ID
func (httpServer *HttpServer) handleListSerialNumbers(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
			}
		}
	}
	shardID := 0
	if len(arrayParams) > 1 {
		shardIDParam, ok := arrayParams[1].(float64)
		if ok {
			shardID = int(shardIDParam)
		}
	}
	result, err := httpServer.txService.ListSerialNumbers(*tokenID, byte(shardID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
	}
	return result, nil
}

// handleListSerialNumbers - return list all serialnumber in shard for token ID
func (httpServer *HttpServer) handleListSNDerivator(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
			}
		}
	}
	shardID := 0
	if len(arrayParams) > 1 {
		shardIDParam, ok := arrayParams[1].(float64)
		if ok {
			shardID = int(shardIDParam)
		}
	}
	result, err := httpServer.txService.ListSNDerivator(*tokenID, byte(shardID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
	}
	return result, nil
}

// handleListCommitments - return list all commitments in shard for token ID
func (httpServer *HttpServer) handleListCommitments(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
			}
		}
	}
	shardID := byte(0)
	if len(arrayParams) > 1 {
		shardIDParam, ok := arrayParams[1].(float64)
		if ok {
			shardID = byte(shardIDParam)
		}
	}
	result, err := httpServer.txService.ListCommitments(*tokenID, shardID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
	}
	return result, nil
}

// handleListCommitmentIndices - return list all commitment indices in shard for token ID
func (httpServer *HttpServer) handleListCommitmentIndices(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
			}
		}
	}
	shardID := byte(0)
	if len(arrayParams) > 1 {
		shardIDParam, ok := arrayParams[1].(float64)
		if ok {
			shardID = byte(shardIDParam)
		}
	}

	result, err := httpServer.txService.ListCommitmentIndices(*tokenID, shardID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
	}
	return result, nil
}

// handleHasSerialNumbers - check list serial numbers existed in db of node
func (httpServer *HttpServer) handleHasSerialNumbers(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}

	// #1: payment address
	paymentAddressStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("PaymentAddress is invalid"))
	}

	//#2: list serialnumbers in base58check encode string
	serialNumbersStr, ok := arrayParams[1].([]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
	}

	// #3: optional - token ID - default is prv coin
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 2 {
		tokenIDTemp, ok := arrayParams[2].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
		}
	}

	result, err := httpServer.txService.HasSerialNumbers(paymentAddressStr, serialNumbersStr, *tokenID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
	}

	return result, nil
}

// handleHasSerialNumbers - check list serial numbers existed in db of node
func (httpServer *HttpServer) handleHasSnDerivators(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}

	// #1: payment address
	paymentAddressStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("paymentAddress is invalid"))
	}

	//#2: list serialnumbers in base58check encode string
	snDerivatorStr, ok := arrayParams[1].([]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("snDerivatorStr is invalid"))
	}

	// #3: optional - token ID - default is prv coin
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 2 {
		tokenIDTemp, ok := arrayParams[1].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("tokenID is invalid"))
		}
		tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
	}
	result, err := httpServer.txService.HasSnDerivators(paymentAddressStr, snDerivatorStr, *tokenID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleCreateRawConvertCoinVer1ToVer2TxToken(params interface{}, closeChan <-chan struct{}) (*jsonresult.CreateTransactionResult, *rpcservice.RPCError){
	Logger.log.Debugf("handleCreateRawConvertVer1ToVer2Transaction params: %+v", params)

	// create new param to build raw tx from param interface
	createRawTxTokenParam, errNewParam := bean.NewCreateRawPrivacyTokenTxConversionVer1To2Param(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	txHash, txBytes, txShardID, err := httpServer.txService.BuildRawConvertVer1ToVer2Token(createRawTxTokenParam)
	if err != nil {
		// return hex for a new tx
		return nil, err
	}

	result := jsonresult.NewCreateTransactionResult(txHash, common.EmptyString, txBytes, txShardID)
	Logger.log.Debugf("handleCreateRawConvertVer1ToVer2Transaction result: %+v", result)
	return &result, nil
}

func (httpServer *HttpServer) handleCreateConvertCoinVer1ToVer2TxToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateConvertCoinVer1ToVer2TxToken params: %+v", params)

	tx, errTx := httpServer.handleCreateRawConvertCoinVer1ToVer2TxToken(params, closeChan)
	if errTx != nil {
		return nil, errTx
	}
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	_, err := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err != nil {
		Logger.log.Errorf("handleCreateConvertCoinVer1ToVer2TxToken result: %+v, err: %+v", nil, err)
		return nil, err
	}
	return tx, nil
}

// handleCreateRawCustomTokenTransaction - handle create a custom token command and return in hex string format.
func (httpServer *HttpServer) handleCreateRawPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	tx, err := httpServer.txService.BuildRawPrivacyCustomTokenTransaction(params, nil)
	if err != nil {
		Logger.log.Error(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	byteArrays, errJson := json.Marshal(tx)
	if errJson != nil {
		Logger.log.Error(errJson)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, errJson)
	}

	tokenData := tx.GetTxTokenData()
	result := jsonresult.CreateTransactionTokenResult{
		ShardID:         common.GetShardIDFromLastByte(tx.GetTxBase().GetSenderAddrLastByte()),
		TxID:            tx.Hash().String(),
		TokenID:         tokenData.PropertyID.String(),
		TokenName:       tokenData.PropertyName,
		TokenAmount:     tokenData.Amount,
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

// handleSendRawTransaction...
func (httpServer *HttpServer) handleSendRawPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	base58CheckData, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param is invalid"))
	}

	txMsg, tx, err1 := httpServer.txService.SendRawPrivacyCustomTokenTransaction(base58CheckData)
	if err1 != nil {
		return nil, err1
	}

	err := httpServer.config.Server.PushMessageToAll(txMsg)
	//Mark forwarded message
	if err == nil {
		httpServer.config.TxMemPool.MarkForwardedTransaction(*tx.Hash())
	}
	tokenData := tx.GetTxTokenData()
	result := jsonresult.CreateTransactionTokenResult{
		TxID:        tx.Hash().String(),
		TokenID:     tokenData.PropertyID.String(),
		TokenName:   tokenData.PropertyName,
		TokenAmount: tokenData.Amount,
		ShardID:     common.GetShardIDFromLastByte(tx.GetTxBase().GetSenderAddrLastByte()),
	}
	return result, nil
}

// handleCreateAndSendCustomTokenTransaction - create and send a tx which process on a custom token look like erc-20 on eth
func (httpServer *HttpServer) handleCreateAndSendPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawPrivacyCustomTokenTransaction(params, closeChan)
	if err != nil {
		return nil, err
	}
	tx := data.(jsonresult.CreateTransactionTokenResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	_, err = httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err != nil {
		Logger.log.Errorf("handleCreateAndSendPrivacyCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, err
	}
	return tx, nil
}

// handleCreateRawStakingTransaction handles create staking
func (httpServer *HttpServer) handleCreateRawStakingTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	// get component

	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 5 element"))
	}

	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	keyWallet := new(wallet.KeyWallet)
	keyWallet.KeySet = *createRawTxParam.SenderKeySet
	funderPaymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

	// prepare meta data
	data, ok := paramsArray[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Data For Staking Transaction %+v", paramsArray[4]))
	}

	stakingType, ok := data["StakingType"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Staking Type For Staking Transaction %+v", data["StakingType"]))
	}

	candidatePaymentAddress, ok := data["CandidatePaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Producer Payment Address for Staking Transaction %+v", data["CandidatePaymentAddress"]))
	}

	// Get private seed, a.k.a mining key
	privateSeed, ok := data["PrivateSeed"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Private Seed For Staking Transaction %+v", data["PrivateSeed"]))
	}
	privateSeedBytes, ver, errDecode := base58.Base58Check{}.Decode(privateSeed)
	if (errDecode != nil) || (ver != common.ZeroByte) {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("Decode privateseed failed!"))
	}

	//Get RewardReceiver Payment Address
	rewardReceiverPaymentAddress, ok := data["RewardReceiverPaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Reward Receiver Payment Address For Staking Transaction %+v", data["RewardReceiverPaymentAddress"]))
	}

	//Get auto staking flag
	autoReStaking, ok := data["AutoReStaking"].(bool)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid auto restaking flag %+v", data["AutoReStaking"]))
	}

	// Get candidate publickey
	candidateWallet, err := wallet.Base58CheckDeserialize(candidatePaymentAddress)
	if err != nil || candidateWallet == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Base58CheckDeserialize candidate Payment Address failed"))
	}
	pk := candidateWallet.KeySet.PaymentAddress.Pk

	committeePK, err := incognitokey.NewCommitteeKeyFromSeed(privateSeedBytes, pk)
	if err != nil {
		Logger.log.Critical(err)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Cannot get payment address"))
	}

	committeePKBytes, err := committeePK.Bytes()
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Cannot import key set"))
	}

	stakingMetadata, err := metadata.NewStakingMetadata(
		int(stakingType), funderPaymentAddress, rewardReceiverPaymentAddress,
		httpServer.config.ChainParams.StakingAmountShard,
		base58.Base58Check{}.Encode(committeePKBytes, common.ZeroByte), autoReStaking)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	txID, txBytes, txShardID, err := httpServer.txService.CreateRawTransaction(createRawTxParam, stakingMetadata)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	result := jsonresult.CreateTransactionResult{
		TxID:            txID.String(),
		Base58CheckData: base58.Base58Check{}.Encode(txBytes, common.ZeroByte),
		ShardID:         txShardID,
	}
	return result, nil
}

// handleCreateAndSendStakingTx - RPC creates staking transaction and send to network
func (httpServer *HttpServer) handleCreateAndSendStakingTx(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	var err error
	data, err := httpServer.handleCreateRawStakingTransaction(params, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, tx.ShardID)
	return result, nil
}

// handleCreateRawStopAutoStakingTransaction - RPC create stop auto stake tx
func (httpServer *HttpServer) handleCreateRawStopAutoStakingTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 5 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 5 element"))
	}

	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	keyWallet := new(wallet.KeyWallet)
	keyWallet.KeySet = *createRawTxParam.SenderKeySet
	funderPaymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	_ = funderPaymentAddress

	//Get data to create meta data
	data, ok := paramsArray[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Staking Type For Staking Transaction %+v", paramsArray[4]))
	}

	//Get staking type
	stopAutoStakingType, ok := data["StopAutoStakingType"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Staking Type For Staking Transaction %+v", data["StopAutoStakingType"]))
	}

	//Get Candidate Payment Address
	candidatePaymentAddress, ok := data["CandidatePaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Producer Payment Address for Staking Transaction %+v", data["CandidatePaymentAddress"]))
	}
	// Get private seed, a.k.a mining key
	privateSeed, ok := data["PrivateSeed"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Private Seed for Staking Transaction %+v", data["PrivateSeed"]))
	}
	privateSeedBytes, ver, err := base58.Base58Check{}.Decode(privateSeed)
	if (err != nil) || (ver != common.ZeroByte) {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("Decode privateseed failed!"))
	}

	// Get candidate publickey
	candidateWallet, err := wallet.Base58CheckDeserialize(candidatePaymentAddress)
	if err != nil || candidateWallet == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Base58CheckDeserialize candidate Payment Address failed"))
	}
	pk := candidateWallet.KeySet.PaymentAddress.Pk

	committeePK, err := incognitokey.NewCommitteeKeyFromSeed(privateSeedBytes, pk)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	committeePKBytes, err := committeePK.Bytes()
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	stakingMetadata, err := metadata.NewStopAutoStakingMetadata(int(stopAutoStakingType), base58.Base58Check{}.Encode(committeePKBytes, common.ZeroByte))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	txID, txBytes, txShardID, err := httpServer.txService.CreateRawTransaction(createRawTxParam, stakingMetadata)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	result := jsonresult.CreateTransactionResult{
		TxID:            txID.String(),
		Base58CheckData: base58.Base58Check{}.Encode(txBytes, common.ZeroByte),
		ShardID:         txShardID,
	}
	return result, nil
}

// handleCreateRawStopAutoStakingTransaction - RPC create and send stop auto stake tx to network
func (httpServer *HttpServer) handleCreateAndSendStopAutoStakingTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	var err error
	data, err := httpServer.handleCreateRawStopAutoStakingTransaction(params, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData

	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, tx.ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleDecryptOutputCoinByKeyOfTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramsArray := common.InterfaceSlice(params)

	txIdParam, ok := paramsArray[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tx id param is invalid"))
	}
	txId, err1 := common.Hash{}.NewHashFromStr(txIdParam)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tx id param is invalid"))
	}

	keys, ok := paramsArray[1].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}
	// get keyset only contain readonly-key by deserializing(optional)
	var readonlyKey *wallet.KeyWallet
	var err error
	readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
	if !ok || readonlyKeyStr == "" {
		//return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid readonly key"))
		Logger.log.Info("ReadonlyKey is optional")
	} else {
		readonlyKey, err = wallet.Base58CheckDeserialize(readonlyKeyStr)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
	}

	// get keyset only contain pub-key by deserializing(required)
	paymentAddressStr, ok := keys["PaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("invalid payment address"))
	}
	paymentAddressKey, err := wallet.Base58CheckDeserialize(paymentAddressStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	// create a key set
	keySet := &incognitokey.KeySet{
		PaymentAddress: paymentAddressKey.KeySet.PaymentAddress,
	}
	// readonly key is optional
	if readonlyKey != nil && len(readonlyKey.KeySet.ReadonlyKey.Rk) > 0 {
		keySet.ReadonlyKey = readonlyKey.KeySet.ReadonlyKey
	}

	result, err2 := httpServer.txService.DecryptOutputCoinByKeyByTransaction(keySet, txId.String())

	return result, err2
}
