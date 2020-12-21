package rpcserver

import (
	"fmt"
	"strings"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/pkg/errors"
)

/*
handleGetMiningInfo - RPC returns various mining-related info
*/
func (httpServer *HttpServer) handleGetMiningInfo(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	result := jsonresult.NewGetMiningInfoResult(httpServer.config.TxMemPool.Count(), *httpServer.config.BlockChain, httpServer.config.ConsensusEngine, *httpServer.config.ChainParams, httpServer.config.Server.IsEnableMining())
	return result, nil
}

func (httpServer *HttpServer) handleEnableMining(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 element"))
	}

	enableParam, ok := arrayParams[0].(bool)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("EnableParam component invalid"))
	}

	// validator by validator key
	validatorKeyParam, ok := arrayParams[1].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Validator key component invalid"))
	}
	seed, _, err := base58.Base58Check{}.Decode(validatorKeyParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Validator key component invalid"))
	}
	committeePK, err := incognitokey.NewCommitteeKeyFromSeed(seed, []byte{})
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Validator key component invalid"))
	}
	blsPubkey := "bls:" + base58.Base58Check{}.Encode(committeePK.MiningPubKey[common.BlsConsensus], common.ZeroByte)
	allMiningKeyList := httpServer.config.ConsensusEngine.GetAllMiningPublicKeys()
	if ok, err := common.SliceExists(allMiningKeyList, blsPubkey); !ok || err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Validator key component invalid"))
	}

	return httpServer.config.Server.EnableMining(enableParam), nil
}

func (httpServer *HttpServer) handleGetChainMiningStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Chain ID empty"))
	}

	chainIDParam, ok := arrayParams[0].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Chain ID component invalid"))
	}
	return httpServer.config.Server.GetChainMiningStatus(int(chainIDParam)), nil
}

// handleGetPublicKeyRole - from bls consensus public key and get role in network
func (httpServer *HttpServer) handleGetPublicKeyRole(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	keyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}

	keyParts := strings.Split(keyParam, ":")
	if len(keyParts) != 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}
	keyType := keyParts[0]
	publicKey := keyParts[1]

	role, shardID := httpServer.config.Server.GetPublicKeyRole(publicKey, keyType)
	if role == -2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInternalError, errors.New("Can't get publickey role"))
	}
	// role: -1 notstake; 0 candidate; 1 committee
	result := &struct {
		Role    int
		ShardID int
	}{
		Role:    role,
		ShardID: shardID,
	}

	return result, nil
}

// handleGetValidatorKeyRole - get validator key, convert to bls consensus public key and get role in network
func (httpServer *HttpServer) handleGetValidatorKeyRole(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	keyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}

	privateSeedBytes, _, err := base58.Base58Check{}.Decode(keyParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	_, blsPubKey := blsmultisig.KeyGen(privateSeedBytes)
	blsPubkey := blsmultisig.PKBytes(blsPubKey)
	temp := common.BlsConsensus + ":" + base58.Base58Check{}.Encode(blsPubkey[:], common.ZeroByte)

	newParams := []interface{}{}
	newParams = append(newParams, temp)

	return httpServer.handleGetPublicKeyRole(newParams, closeChan)
}

func (httpServer *HttpServer) handleGetIncognitoPublicKeyRole(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	keyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}

	role, isBeacon, shardID := httpServer.config.Server.GetIncognitoPublicKeyRole(keyParam)
	if role == -2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInternalError, errors.New("Can't get publickey role"))
	}
	// role: -1 notstake; 0 candidate; 1 pending; 2 committee
	result := &struct {
		Role     int
		IsBeacon bool
		ShardID  int
	}{
		Role:     role,
		IsBeacon: isBeacon,
		ShardID:  shardID,
	}
	return result, nil
}

func (httpServer *HttpServer) handleGetMinerRewardFromMiningKey(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param empty"))
	}

	keyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}

	keyParts := strings.Split(keyParam, ":")
	if len(keyParts) != 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}
	keyType := keyParts[0]
	publicKey := keyParts[1]
	incPublicKey := httpServer.config.Server.GetMinerIncognitoPublickey(publicKey, keyType)
	if len(incPublicKey) != 32 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("incognito public key length error, expect 32 but get %+v", len(incPublicKey)))
	}
	rewardAmountResult, err := httpServer.blockService.GetMinerRewardFromMiningKey(incPublicKey)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	return rewardAmountResult, nil
}
