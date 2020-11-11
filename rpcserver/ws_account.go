package rpcserver

import (
	"errors"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/incognitochain/incognito-chain/wallet"
	"reflect"
)

var (
	ErrParseTransaction = errors.New("Parse transaction failed")
)

func (wsServer *WsServer) handleSubcribeCrossOutputCoinByPrivateKey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain ONE params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	privateKey := arrayParams[0].(string)
	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	err = keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewShardblockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe New Shard Block")
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewShardblockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				shardBlock, ok := msg.Value.(*types.ShardBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				m := make(map[byte]uint64)
				for senderShardID, crossTransactions := range shardBlock.Body.CrossTransactions {
					for _, crossTransaction := range crossTransactions {
						for _, crossOutputCoin := range crossTransaction.OutputCoin {
							processedOutputCoin := blockchain.DecryptOutputCoinByKey(wsServer.config.BlockChain.GetBestStateShard(shardBlock.Header.ShardID).GetCopiedTransactionStateDB(), &crossOutputCoin, &keyWallet.KeySet, &common.PRVCoinID, senderShardID)
							if processedOutputCoin == nil {
								Logger.log.Errorf("processedOutputCoin is nil!")
								continue
							}
							if value, ok := m[senderShardID]; ok {
								value += processedOutputCoin.CoinDetails.GetValue()
								m[senderShardID] = value
							} else {
								if processedOutputCoin.CoinDetails != nil {
									m[senderShardID] = processedOutputCoin.CoinDetails.GetValue()
								}
							}
						}
					}
				}
				if len(m) != 0 {
					for senderShardID, value := range m {
						cResult <- RpcSubResult{Result: jsonresult.CrossOutputCoinResult{
							SenderShardID:   senderShardID,
							ReceiverShardID: shardBlock.Header.ShardID,
							BlockHeight:     shardBlock.Header.Height,
							BlockHash:       shardBlock.Header.Hash().String(),
							PaymentAddress:  keyWallet.Base58CheckSerialize(wallet.PaymentAddressType),
							Value:           value,
						}, Error: nil}
					}
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe New Beacon Block"}}
				return
			}
		}
	}
}

func (wsServer *WsServer) handleSubcribeCrossCustomTokenPrivacyByPrivateKey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain ONE params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	privateKey, ok := arrayParams[0].(string)
	if !ok {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Params is invalid"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	err = keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewShardblockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe New Shard Block")
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewShardblockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				shardBlock, ok := msg.Value.(*types.ShardBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				m := make(map[byte]map[common.Hash]uint64)
				for senderShardID, crossTransactions := range shardBlock.Body.CrossTransactions {
					for _, crossTransaction := range crossTransactions {
						for _, crossTokenPrivacyData := range crossTransaction.TokenPrivacyData {
							for _, crossOutputCoin := range crossTokenPrivacyData.OutputCoin {
								processedOutputCoin := blockchain.DecryptOutputCoinByKey(wsServer.config.BlockChain.GetBestStateShard(shardBlock.Header.ShardID).GetCopiedTransactionStateDB(), &crossOutputCoin, &keyWallet.KeySet, &common.PRVCoinID, senderShardID)
								if processedOutputCoin != nil {
									if m[senderShardID] == nil {
										m[senderShardID] = make(map[common.Hash]uint64)
									}
									if value, ok := m[senderShardID][crossTokenPrivacyData.PropertyID]; ok {
										value += processedOutputCoin.CoinDetails.GetValue()
										m[senderShardID][crossTokenPrivacyData.PropertyID] = value
									} else {
										m[senderShardID][crossTokenPrivacyData.PropertyID] = processedOutputCoin.CoinDetails.GetValue()
									}
								}
							}
						}
					}
				}
				if len(m) != 0 {
					for senderShardID, tokenIDValue := range m {
						for tokenID, value := range tokenIDValue {
							cResult <- RpcSubResult{Result: jsonresult.CrossCustomTokenPrivacyResult{
								SenderShardID:   senderShardID,
								ReceiverShardID: shardBlock.Header.ShardID,
								BlockHeight:     shardBlock.Header.Height,
								BlockHash:       shardBlock.Header.Hash().String(),
								PaymentAddress:  keyWallet.Base58CheckSerialize(wallet.PaymentAddressType),
								TokenID:         tokenID.String(),
								Value:           value,
							}, Error: nil}
						}
					}
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe New Beacon Block"}}
				return
			}
		}
	}
}
