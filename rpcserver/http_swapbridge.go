package rpcserver

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/pkg/errors"
)

// handleGetLatestBridgeSwapProof returns the latest proof of a change in bridge's committee
func (httpServer *HttpServer) handleGetLatestBridgeSwapProof(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	latestBlock := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	for i := latestBlock; i >= 1; i-- {
		params := []interface{}{float64(i)}
		proof, err := httpServer.handleGetBridgeSwapProof(params, closeChan)
		if err != nil {
			continue
		}
		return proof, nil
	}
	return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.Errorf("no swap proof found before block %d", latestBlock))
}

// handleGetBridgeSwapProof returns a proof of a new bridge committee (for a given beacon block height)
func (httpServer *HttpServer) handleGetBridgeSwapProof(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Infof("handleGetBridgeSwapProof params: %+v", params)
	listParams, ok := params.([]interface{})
	if !ok || len(listParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	heightParam, ok := listParams[0].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("height param is invalid"))
	}
	beaconHeigh := uint64(heightParam)
	// Get proof of instruction on beacon
	beaconInstProof, beaconBlock, errProof := getSwapProofOnBeacon(beaconHeigh, httpServer.config.BlockChain, httpServer.config.ConsensusEngine, metadata.BridgeSwapConfirmMeta)
	if errProof != nil {
		return nil, errProof
	}

	// Get proof of instruction on bridge
	bridgeInstProof, bridgeShardBlockHeight, err := getBridgeSwapProofOnBridge(beaconBlock, httpServer.GetBlockchain(), httpServer.config.ConsensusEngine)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	// Decode instruction to send to Ethereum without having to decode on client
	decodedInst, err := blockchain.DecodeInstruction(beaconInstProof.inst)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	inst := hex.EncodeToString(decodedInst)

	return buildProofResult(inst, beaconInstProof, bridgeInstProof, strconv.FormatUint(beaconBlock.Header.Height, 10), strconv.FormatUint(bridgeShardBlockHeight, 10)), nil
}

// getBridgeSwapProofOnBridge finds a bridge committee swap instruction in a bridge block and returns its proof; the bridge block must be included in a given beaconBlock
func getBridgeSwapProofOnBridge(
	beaconBlock *blockchain.BeaconBlock,
	bc *blockchain.BlockChain,
	ce ConsensusEngine,
) (*swapProof, uint64, error) {
	// Get bridge block and check if it contains bridge swap instruction
	bridgeShardBlock, instID, err := findBridgeBlockWithInst(beaconBlock, bc)
	if err != nil {
		return nil, 0, err
	}
	insts := bridgeShardBlock.Body.Instructions
	block := &shardBlock{ShardBlock: bridgeShardBlock}
	swapProof, err := buildProofForBlock(block, insts, instID, ce)
	return swapProof, bridgeShardBlock.Header.Height, err
}

// findBridgeBlockWithInst traverses all shard blocks included in a beacon block and returns the one containing a bridge swap instruction
func findBridgeBlockWithInst(
	beaconBlock *blockchain.BeaconBlock,
	bc *blockchain.BlockChain,
) (*blockchain.ShardBlock, int, error) {
	bridgeID := byte(common.BridgeShardID)
	for _, state := range beaconBlock.Body.ShardState[bridgeID] {
		bridgeBlock, _, err := getShardAndBeaconBlocks(state.Height, bc)
		if err != nil {
			return nil, 0, err
		}

		_, bridgeInstID := findCommSwapInst(bridgeBlock.Body.Instructions, metadata.BridgeSwapConfirmMeta)
		BLogger.log.Debugf("Finding swap bridge inst in bridge block %d %d", state.Height, bridgeInstID)
		if bridgeInstID >= 0 {
			return bridgeBlock, bridgeInstID, nil
		}
	}

	return nil, 0, fmt.Errorf("cannot find bridge swap instruction in bridge block")
}
