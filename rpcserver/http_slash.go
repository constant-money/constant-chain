package rpcserver

import (
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

func (httpServer *HttpServer) handleGetProducersBlackList(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	//// beaconHeight := httpServer.config.BlockChain.BestState.Beacon.BeaconHeight
	//arrayParams := common.InterfaceSlice(params)
	//if arrayParams == nil || len(arrayParams) < 1 {
	//	return nil, nil
	//}
	//beaconHeightParam, ok := arrayParams[0].(float64)
	//if !ok {
	//	return nil, nil
	//}
	//beaconHeight := uint64(beaconHeightParam)
	//
	//producersBlackList, err := httpServer.blockService.GetProducersBlackList(beaconHeight)
	//if err != nil {
	//	return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	//}
	return nil, nil
}

func (httpServer *HttpServer) handleGetProducersBlackListDetail(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	// beaconHeight := httpServer.config.BlockChain.BestState.Beacon.BeaconHeight
	//arrayParams := common.InterfaceSlice(params)
	//if arrayParams == nil || len(arrayParams) < 1 {
	//	return nil, nil
	//}
	//
	//beaconHeightParam, ok := arrayParams[0].(float64)
	//if !ok {
	//	return nil, nil
	//}
	//beaconHeight := uint64(beaconHeightParam)
	//
	//producersBlackList, err := httpServer.blockService.GetProducersBlackList(beaconHeight)
	//if err != nil {
	//	return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	//}
	//type producerBlacklistString struct {
	//	IncPubKey    string
	//	MiningPubKey map[string]string
	//	Epochs       uint8
	//}
	//var result []producerBlacklistString
	//for k, v := range producersBlackList {
	//	var keySet incognitokey.CommitteePublicKey
	//	keySet.FromString(k)
	//
	//	var keyMap producerBlacklistString
	//	keyMap.IncPubKey = keySet.GetIncKeyBase58()
	//	keyMap.MiningPubKey = make(map[string]string)
	//	for keyType := range keySet.MiningPubKey {
	//		keyMap.MiningPubKey[keyType] = keySet.GetMiningKeyBase58(keyType)
	//	}
	//	keyMap.Epochs = v
	//	result = append(result, keyMap)
	//}

	return nil, nil
}
