package rpcserver

import (
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

/*
handleGetMempoolInfo - RPC returns information about the node's current txs memory pool
*/
func (httpServer *HttpServer) handleGetMempoolInfo(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	result := jsonresult.NewGetMempoolInfo(httpServer.config.TxMemPool)
	return result, nil
}

/*
handleGetRawMempool - RPC returns all transaction ids in memory pool as a json array of string transaction ids
Hint: use getmempoolentry to fetch a specific transaction from the mempool.
*/
func (httpServer *HttpServer) handleGetRawMempool(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	result := jsonresult.NewGetRawMempoolResult(httpServer.config.TxMemPool)
	return result, nil
}

/*
handleGetPendingTxsInBlockgen - RPC returns all transaction ids in blockgen
*/
func (httpServer *HttpServer) handleGetPendingTxsInBlockgen(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	result := jsonresult.NewGetPendingTxsInBlockgenResult(httpServer.config.Blockgen.GetPendingTxsV2(255))
	return result, nil
}

func (httpServer *HttpServer) handleGetNumberOfTxsInMempool(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	result := httpServer.txMemPoolService.GetNumberOfTxsInMempool()
	return result, nil
}

/*
handleMempoolEntry - RPC fetch a specific transaction from the mempool
*/
func (httpServer *HttpServer) handleMempoolEntry(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	// Param #1: hash string of tx(tx id)
	txIDParam, ok := params.(string)
	if !ok || txIDParam == "" {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("transaction id is invalid"))
	}

	txInPool, shardID, err := httpServer.txMemPoolService.MempoolEntry(txIDParam)
	if err != nil {
		return nil, err
	}

	tx, errM := jsonresult.NewTransactionDetail(txInPool, nil, 0, 0, shardID)
	if errM != nil {
		Logger.log.Error(errM)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errM)
	}
	tx.IsInMempool = true
	return tx, nil
}

// handleRemoveTxInMempool - try to remove tx from tx mempool
func (httpServer *HttpServer) handleRemoveTxInMempool(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrays := common.InterfaceSlice(params)
	if arrays == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param is invalid"))
	}

	result := []bool{}
	for _, txHashString := range arrays {
		txHash, ok := txHashString.(string)
		if ok {
			t, _ := httpServer.txMemPoolService.RemoveTxInMempool(txHash)
			result = append(result, t)
		} else {
			result = append(result, false)
		}
	}
	return result, nil
}
