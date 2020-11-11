package rpcserver

import (
	"errors"
	"github.com/incognitochain/incognito-chain/blockchain/types"

	"reflect"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

func (wsServer *WsServer) handleSubcribeShardCandidateByPublickey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain 1 params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	candidate, ok := arrayParams[0].(string)
	if !ok {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Invalid Public Key"))
		cResult <- RpcSubResult{Error: err}
	}
	// try to get candidate from beacon beststate
	candidates, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetShardCandidate())
	if err != nil {
		panic(err)
	}
	if common.IndexOfStr(candidate, candidates) > -1 {
		cResult <- RpcSubResult{Result: true, Error: nil}
		return
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewBeaconBlockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe Candidate Role", candidate)
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewBeaconBlockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				_, ok := msg.Value.(*types.BeaconBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				// try to get candidate from beacon beststate
				candidates, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetShardCandidate())
				if err != nil {
					panic(err)
				}
				if common.IndexOfStr(candidate, candidates) > -1 {
					cResult <- RpcSubResult{Result: true, Error: nil}
					return
				} else {
					continue
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe Shard Candidate " + candidate}}
				return
			}
		}
	}
}

func (wsServer *WsServer) handleSubcribeShardPendingValidatorByPublickey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain 1 params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	validator, ok := arrayParams[0].(string)
	if !ok {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Invalid Public Key"))
		cResult <- RpcSubResult{Error: err}
	}
	// try to get validator from beacon beststate
	var err error
	allValidators := make(map[byte][]string)
	for shardID, committee := range wsServer.config.BlockChain.GetBeaconBestState().GetShardPendingValidator() {
		allValidators[shardID], err = incognitokey.CommitteeKeyListToString(committee)
		if err != nil {
			panic(err)
		}
	}
	for _, shardValidators := range allValidators {
		if common.IndexOfStr(validator, shardValidators) > -1 {
			cResult <- RpcSubResult{Result: true, Error: nil}
			return
		}
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewBeaconBlockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe Validator Role", validator)
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewBeaconBlockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				_, ok := msg.Value.(*types.BeaconBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				// try to get validator from beacon beststate
				allValidators := make(map[byte][]string)
				var err error
				for shardID, committee := range wsServer.config.BlockChain.GetBeaconBestState().GetShardPendingValidator() {
					allValidators[shardID], err = incognitokey.CommitteeKeyListToString(committee)
					if err != nil {
						panic(err)
					}
				}
				for _, shardValidators := range allValidators {
					if common.IndexOfStr(validator, shardValidators) > -1 {
						cResult <- RpcSubResult{Result: true, Error: nil}
						return
					}
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe Shard Validator " + validator}}
				return
			}
		}
	}
}

func (wsServer *WsServer) handleSubcribeShardCommitteeByPublickey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain 1 params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	committee, ok := arrayParams[0].(string)
	if !ok {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Invalid Public Key"))
		cResult <- RpcSubResult{Error: err}
	}
	// try to get committee from beacon beststate
	allCommittees := make(map[byte][]string)
	var err error
	for shardID, committee := range wsServer.config.BlockChain.GetBeaconBestState().GetShardCommittee() {
		allCommittees[shardID], err = incognitokey.CommitteeKeyListToString(committee)
		if err != nil {
			panic(err)
		}
	}
	for _, shardCommittees := range allCommittees {
		if common.IndexOfStr(committee, shardCommittees) > -1 {
			cResult <- RpcSubResult{Result: true, Error: nil}
			return
		}
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewBeaconBlockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe Committee Role", committee)
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewBeaconBlockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				_, ok := msg.Value.(*types.BeaconBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				// try to get committee from beacon beststate
				allCommittees := make(map[byte][]string)
				var err error
				for shardID, committee := range wsServer.config.BlockChain.GetBeaconBestState().GetShardCommittee() {
					allCommittees[shardID], err = incognitokey.CommitteeKeyListToString(committee)
					if err != nil {
						panic(err)
					}
				}
				for _, shardCommittees := range allCommittees {
					if common.IndexOfStr(committee, shardCommittees) > -1 {
						cResult <- RpcSubResult{Result: true, Error: nil}
						return
					}
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe Shard Committee " + committee}}
				return
			}
		}
	}
}

func (wsServer *WsServer) handleSubcribeBeaconCandidateByPublickey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain 1 params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	candidate, ok := arrayParams[0].(string)
	if !ok {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Invalid Public Key"))
		cResult <- RpcSubResult{Error: err}
	}
	// try to get candidate from beacon beststate
	candidates, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetBeaconCandidate())
	if err != nil {
		panic(err)
	}
	if common.IndexOfStr(candidate, candidates) > -1 {
		cResult <- RpcSubResult{Result: true, Error: nil}
		return
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewBeaconBlockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe Candidate Role", candidate)
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewBeaconBlockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				_, ok := msg.Value.(*types.BeaconBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				// try to get candidate from beacon beststate
				candidates, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetBeaconCandidate())
				if err != nil {
					panic(err)
				}
				if common.IndexOfStr(candidate, candidates) > -1 {
					cResult <- RpcSubResult{Result: true, Error: nil}
					return
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe Beacon Candidate " + candidate}}
				return
			}
		}
	}
}

func (wsServer *WsServer) handleSubcribeBeaconPendingValidatorByPublickey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain 1 params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	validator, ok := arrayParams[0].(string)
	if !ok {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Invalid Public Key"))
		cResult <- RpcSubResult{Error: err}
	}
	// try to get validator from beacon beststate
	validators, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetBeaconPendingValidator())
	if err != nil {
		panic(err)
	}
	if common.IndexOfStr(validator, validators) > -1 {
		cResult <- RpcSubResult{Result: true, Error: nil}
		return
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewBeaconBlockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe Pending Validator Role", validator)
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewBeaconBlockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				_, ok := msg.Value.(*types.BeaconBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				// try to get validator from beacon beststate
				validators, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetBeaconPendingValidator())
				if err != nil {
					panic(err)
				}
				if common.IndexOfStr(validator, validators) > -1 {
					cResult <- RpcSubResult{Result: true, Error: nil}
					return
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe Beacon Validator " + validator}}
				return
			}
		}
	}
}

func (wsServer *WsServer) handleSubcribeBeaconCommitteeByPublickey(params interface{}, subcription string, cResult chan RpcSubResult, closeChan <-chan struct{}) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) != 1 {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Methods should only contain 1 params"))
		cResult <- RpcSubResult{Error: err}
		return
	}
	committee, ok := arrayParams[0].(string)
	if !ok {
		err := rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Invalid Public Key"))
		cResult <- RpcSubResult{Error: err}
	}
	// try to get candidate from beacon beststate
	committees, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetBeaconCommittee())
	if err != nil {
		panic(err)
	}
	if common.IndexOfStr(committee, committees) > -1 {
		cResult <- RpcSubResult{Result: true, Error: nil}
		return
	}
	subId, subChan, err := wsServer.config.PubSubManager.RegisterNewSubscriber(pubsub.NewBeaconBlockTopic)
	if err != nil {
		err := rpcservice.NewRPCError(rpcservice.SubcribeError, err)
		cResult <- RpcSubResult{Error: err}
		return
	}
	defer func() {
		Logger.log.Info("Finish Subscribe Committee Role", committee)
		wsServer.config.PubSubManager.Unsubscribe(pubsub.NewBeaconBlockTopic, subId)
		close(cResult)
	}()
	for {
		select {
		case msg := <-subChan:
			{
				_, ok := msg.Value.(*types.BeaconBlock)
				if !ok {
					Logger.log.Errorf("Wrong Message Type from Pubsub Manager, wanted *blockchain.BeaconBlock, have %+v", reflect.TypeOf(msg.Value))
					continue
				}
				// try to get committee from beacon beststate
				committees, err := incognitokey.CommitteeKeyListToString(wsServer.config.BlockChain.GetBeaconBestState().GetBeaconCommittee())
				if err != nil {
					panic(err)
				}
				if common.IndexOfStr(committee, committees) > -1 {
					cResult <- RpcSubResult{Result: true, Error: nil}
					return
				}
			}
		case <-closeChan:
			{
				cResult <- RpcSubResult{Result: jsonresult.UnsubcribeResult{Message: "Unsubscribe Beacon Committee " + committee}}
				return
			}
		}
	}
}
