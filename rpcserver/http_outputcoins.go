package rpcserver

import (
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

//handleListUnspentOutputCoins - use private key to get all tx which contains output coin of account
// by private key, it return full tx outputcoin with amount and receiver address in txs
//component:
//Parameter #1—the minimum number of confirmations an output must have
//Parameter #2—the maximum number of confirmations an output may have
//Parameter #3—the list priv-key which be used to view utxo which also includes the fromHeight of each key
//From height is used to efficiently fetch onetimeaddress outputCoins
func (httpServer *HttpServer) handleListUnspentOutputCoins(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 3 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	var min, max uint64

	if paramsArray[0] != nil {
		minParam, ok := paramsArray[0].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min param is invalid"))
		}
		min = uint64(minParam)
	}

	if paramsArray[1] != nil {
		maxParam, ok := paramsArray[1].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("max param is invalid"))
		}
		max = uint64(maxParam)
	}
	_ = min
	// _ = max

	listKeyParams := common.InterfaceSlice(paramsArray[2])
	if listKeyParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	tokenID := &common.Hash{}
	err1 := tokenID.SetBytes(common.PRVCoinID[:])
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err1)
	}
	if len(paramsArray) == 4 {
		tokenIDStr, ok := paramsArray[3].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
		}
		if tokenIDStr != "" {
			tokenIDHash, err2 := common.Hash{}.NewHashFromStr(tokenIDStr)
			if err2 != nil {
				return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
			}
			tokenID = tokenIDHash
		}
	}

	result, err := httpServer.outputCoinService.ListUnspentOutputCoinsByKey(listKeyParams, tokenID, max)
	if err != nil {
		return nil, err
	}

	return result, nil
}

//handleListOutputCoins - use readonly key to get all tx which contains output coin of account
// by private key, it return full tx outputcoin with amount and receiver address in txs
//component:
//Parameter #1—the minimum number of confirmations an output must have
//Parameter #2—the maximum number of confirmations an output may have
//Parameter #3—the list paymentaddress-readonlykey which be used to view list outputcoin
//Parameter #4 - optional - token id - default prv coin
func (httpServer *HttpServer) handleListOutputCoins(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 3 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	minTemp, ok := paramsArray[0].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min param is invalid"))
	}
	min := uint64(minTemp)

	maxTemp, ok := paramsArray[1].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("max param is invalid"))
	}
	max := uint64(maxTemp)

	_ = min
	// _ = max

	//#3: list key component
	listKeyParams := common.InterfaceSlice(paramsArray[2])
	if listKeyParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	//#4: optional token type - default prv coin
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(paramsArray) > 3 {
		var err1 error
		tokenIdParam, ok := paramsArray[3].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
		}

		tokenID, err1 = common.Hash{}.NewHashFromStr(tokenIdParam)
		if err1 != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err1)
		}
	}
	result, err1 := httpServer.outputCoinService.ListOutputCoinsByKey(listKeyParams, *tokenID, max)
	if err1 != nil {
		return nil, err1
	}
	return result, nil
}

func (httpServer *HttpServer) handleListOutputCoinsFromCache(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 3 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	minTemp, ok := paramsArray[0].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min param is invalid"))
	}
	min := int(minTemp)

	maxTemp, ok := paramsArray[1].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("max param is invalid"))
	}
	max := int(maxTemp)

	_ = min
	_ = max

	//#3: list key component
	listKeyParams := common.InterfaceSlice(paramsArray[2])
	if listKeyParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	//#4: optional token type - default prv coin
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(paramsArray) > 3 {
		var err1 error
		tokenIdParam, ok := paramsArray[3].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
		}

		tokenID, err1 = common.Hash{}.NewHashFromStr(tokenIdParam)
		if err1 != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err1)
		}
	}
	result, err1 := httpServer.outputCoinService.ListCachedOutputCoinsByKey(listKeyParams, *tokenID)
	if err1 != nil {
		return nil, err1
	}
	return result, nil
}

func (httpServer *HttpServer) handleListOutputCoinV2Idxs(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}
	listKeyParams := common.InterfaceSlice(paramsArray[0])
	if listKeyParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(paramsArray) > 1 {
		var err1 error
		tokenIdParam, ok := paramsArray[1].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
		}
		tokenID, err1 = common.Hash{}.NewHashFromStr(tokenIdParam)
		if err1 != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err1)
		}
	}
	result, err1 := httpServer.outputCoinService.ListOutputCoinV2Idxs(listKeyParams[0], *tokenID)
	if err1 != nil {
		return nil, err1
	}
	return result, nil
}

func (httpServer *HttpServer) handleListOutputCoinV2ByIdxs(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	listIndexParams := common.InterfaceSlice(paramsArray[0])
	if listIndexParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	shardIDParam, ok := paramsArray[1].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Shard ID component invalid"))
	}
	shardID := byte(shardIDParam)

	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(paramsArray) > 2 {
		var err1 error
		tokenIdParam, ok := paramsArray[2].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
		}

		tokenID, err1 = common.Hash{}.NewHashFromStr(tokenIdParam)
		if err1 != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err1)
		}
	}
	result, err1 := httpServer.outputCoinService.ListOutputCoinV2ByIdxs(listIndexParams, shardID, *tokenID)
	if err1 != nil {
		return nil, err1
	}
	return result, nil
}
