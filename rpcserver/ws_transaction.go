package rpcserver

import (
	"errors"
	"reflect"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

func (wsServer *WsServer) handleSubscribePendingTransaction(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain 1 params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	txHashTemp, ok := arrayParams[0].(string)
	if !ok || txHashTemp == "" {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Invalid Tx Hash"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	txHash, err := common.Hash{}.NewHashFromStr(txHashTemp)
	if err != nil {
		err1 := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
		cResult <- RpcSubResult{Error: err1}
		return
	}
	// try to get transaction in database
	_, blockHash, _, index, tx, err := wsServer.config.BlockChain.GetTransactionByHash(*txHash)
	if err == nil {
		shardBlock, _, err := wsServer.config.BlockChain.GetShardBlockByHash(blockHash)
		if err == nil {
			res, err := jsonresult.NewTransactionDetail(tx, shardBlock.Hash(), shardBlock.Header.Height, index, shardBlock.Header.ShardID)
			cResult <- RpcSubResult{Result: res, Error: rpcservice.NewRPCError(rpcservice.UnexpectedError, err)}
			return
		}
	}
	// transaction not in database yet then subscribe new shard event block and watch
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewShardblockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe New Pending Transaction ", txHashTemp)
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewShardblockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				shardBlock, ok := msg.Value.(*blockchain.ShardBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.ShardBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				for index, tx := range shardBlock.Body.Transactions {
					if tx.Hash().IsEqual(txHash) {
						res, err := jsonresult.NewTransactionDetail(tx, shardBlock.Hash(), shardBlock.Header.Height, index, shardBlock.Header.ShardID)
						if err != nil {
							cResult <- RpcSubResult{Result: res, Error: rpcservice.NewRPCError(rpcservice.UnexpectedError, err)}
						} else {
							cResult <- RpcSubResult{Result: res, Error: nil}
						}
						return
					}
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe Pending Transaction " + txHashTemp}}
				return
			}
		}
	}
}
