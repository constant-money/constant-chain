package rpcserver

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/incdb"

	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/pkg/errors"
)

type swapProof struct {
	inst []string

	instPath       []string
	instPathIsLeft []bool
	instRoot       string
	blkData        string
	signerSigs     []string
	sigIdxs        []int
}

type ConsensusEngine interface {
	ExtractBridgeValidationData(block common.BlockInterface) ([][]byte, []int, error)
}

// handleGetLatestBeaconSwapProof returns the latest proof of a change in bridge's committee
func (httpServer *HttpServer) handleGetLatestBeaconSwapProof(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	latestBlock := httpServer.config.BlockChain.GetBeaconBestState().BeaconHeight
	for i := latestBlock; i >= 1; i-- {
		params := []interface{}{float64(i)}
		proof, err := httpServer.handleGetBeaconSwapProof(params, closeChan)
		if err != nil {
			continue
		}
		return proof, nil
	}
	return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.Errorf("no swap proof found before block %d", latestBlock))
}

// handleGetBeaconSwapProof returns a proof of a new beacon committee (for a given bridge block height)
func (httpServer *HttpServer) handleGetBeaconSwapProof(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Infof("handleGetBeaconSwapProof params: %+v", params)
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
	beaconInstProof, _, errProof := getSwapProofOnBeacon(beaconHeigh, httpServer.config.BlockChain, httpServer.config.ConsensusEngine, metadata.BeaconSwapConfirmMeta)
	if errProof != nil {
		return nil, errProof
	}
	// Decode instruction to send to Ethereum without having to decode on client
	decodedInst, err := blockchain.DecodeInstruction(beaconInstProof.inst)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	inst := hex.EncodeToString(decodedInst)
	bridgeInstProof := &swapProof{}
	return buildProofResult(inst, beaconInstProof, bridgeInstProof, strconv.FormatUint(beaconHeigh, 10), ""), nil
}

// getSwapProofOnBeacon finds in a given beacon block a committee swap instruction and returns its proof;
// returns rpcservice.RPCError if proof not found
func getSwapProofOnBeacon(
	height uint64,
	bc *blockchain.BlockChain,
	ce ConsensusEngine,
	meta int,
) (*swapProof, *blockchain.BeaconBlock, *rpcservice.RPCError) {
	// Get beacon block
	beaconBlocks, err := blockchain.FetchBeaconBlockFromHeight(bc, height, height)
	if len(beaconBlocks) == 0 {
		err := fmt.Errorf("cannot find beacon block with height %d", height)
		return nil, nil, rpcservice.NewRPCError(rpcservice.GetBeaconBlockByHeightError, err)
	}
	b := beaconBlocks[0]

	// Find bridge swap instruction in beacon block
	insts := b.Body.Instructions
	_, instID := findCommSwapInst(insts, meta)
	if instID < 0 {
		err := fmt.Errorf("cannot find bridge swap instruction in beacon block")
		return nil, nil, rpcservice.NewRPCError(rpcservice.NoSwapConfirmInst, err)
	}
	block := &beaconBlock{BeaconBlock: b}
	proof, err := buildProofForBlock(block, insts, instID, ce)
	if err != nil {
		return nil, nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return proof, b, nil
}

// getShardAndBeaconBlocks returns a shard block (with all of its instructions) and the included beacon blocks
func getShardAndBeaconBlocks(
	height uint64,
	bc *blockchain.BlockChain,
) (*blockchain.ShardBlock, []*blockchain.BeaconBlock, error) {
	bridgeID := byte(common.BridgeShardID)
	bridgeBlocks, err := bc.GetShardBlockByHeight(height, bridgeID)
	if err != nil {
		return nil, nil, err
	}
	if len(bridgeBlocks) == 0 {
		return nil, nil, fmt.Errorf("shard block bridgeID %+v, height %+v not found", bridgeID, height)
	}
	var bridgeBlock *blockchain.ShardBlock
	for _, temp := range bridgeBlocks {
		bridgeBlock = temp
	}
	beaconBlocks, err := getIncludedBeaconBlocks(
		bridgeBlock.Header.Height,
		bridgeBlock.Header.BeaconHeight,
		bridgeBlock.Header.ShardID,
		bc,
	)
	if err != nil {
		return nil, nil, err
	}
	bridgeInsts, err := extractInstsFromShardBlock(bridgeBlock, bc)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("Bridge Instruction", bridgeInsts)
	bridgeBlock.Body.Instructions = bridgeInsts
	return bridgeBlock, beaconBlocks, nil
}

type block interface {
	common.BlockInterface // to be able to get ValidationData from ConsensusEngine

	InstructionMerkleRoot() []byte
	MetaHash() []byte
	Sig(ce ConsensusEngine) ([][]byte, []int, error)
}

// buildProofForBlock builds a swapProof for an instruction in a block (beacon or shard)
func buildProofForBlock(
	blk block,
	insts [][]string,
	id int,
	ce ConsensusEngine,
) (*swapProof, error) {
	// Build merkle proof for instruction in bridge block
	instProof := buildInstProof(insts, id)

	// Get meta hash and block hash
	instRoot := hex.EncodeToString(blk.InstructionMerkleRoot())
	metaHash := blk.MetaHash()

	// Get sig data
	bSigs, sigIdxs, err := blk.Sig(ce)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	fmt.Printf("DeBridge log: sig: %d %x\n", len(bSigs[0]), bSigs[0])
	sigs := []string{}
	for _, s := range bSigs {
		sigs = append(sigs, hex.EncodeToString(s))
	}

	return &swapProof{
		inst:           insts[id],
		instPath:       instProof.getPath(),
		instPathIsLeft: instProof.left,
		instRoot:       instRoot,
		blkData:        hex.EncodeToString(metaHash[:]),
		signerSigs:     sigs,
		sigIdxs:        sigIdxs,
	}, nil
}

// getBeaconSwapProofOnBeacon finds in given beacon blocks a beacon committee swap instruction and returns its proof
func getBeaconSwapProofOnBeacon(
	inst []string,
	beaconBlocks []*blockchain.BeaconBlock,
	db incdb.Database,
	ce ConsensusEngine,
) (*swapProof, error) {
	// Get beacon block and check if it contains beacon swap instruction
	b, instID := findBeaconBlockWithInst(beaconBlocks, inst)
	if b == nil {
		return nil, fmt.Errorf("cannot find corresponding beacon block that includes swap instruction")
	}

	insts := b.Body.Instructions
	block := &beaconBlock{BeaconBlock: b}
	return buildProofForBlock(block, insts, instID, ce)
}

// getIncludedBeaconBlocks retrieves all beacon blocks included in a shard block
func getIncludedBeaconBlocks(
	shardHeight uint64,
	beaconHeight uint64,
	shardID byte,
	bc *blockchain.BlockChain,
) ([]*blockchain.BeaconBlock, error) {
	prevShardBlocks, err := bc.GetShardBlockByHeight(shardHeight-1, shardID)
	if err != nil {
		return nil, err
	}
	var previousShardBlock *blockchain.ShardBlock
	for _, temp := range prevShardBlocks {
		previousShardBlock = temp
	}
	beaconBlocks, err := blockchain.FetchBeaconBlockFromHeight(
		bc,
		previousShardBlock.Header.BeaconHeight+1,
		beaconHeight,
	)
	if err != nil {
		return nil, err
	}
	return beaconBlocks, nil
}

// extractInstsFromShardBlock returns all instructions in a shard block as a slice of []string
func extractInstsFromShardBlock(
	shardBlock *blockchain.ShardBlock,
	//beaconBlocks []*blockchain.BeaconBlock,
	bc *blockchain.BlockChain,
) ([][]string, error) {
	instructions, err := blockchain.CreateShardInstructionsFromTransactionAndInstruction(
		shardBlock.Body.Transactions,
		bc,
		shardBlock.Header.ShardID,
		//	&shardBlock.Header.ProducerAddress,
		//	shardBlock.Header.Height,
		//	beaconBlocks,
		//	shardBlock.Header.BeaconHeight,
	)
	if err != nil {
		return nil, err
	}
	shardInsts := append(instructions, shardBlock.Body.Instructions...)
	return shardInsts, nil
}

// findCommSwapInst finds a swap instruction in a list, returns it along with its index
func findCommSwapInst(insts [][]string, meta int) ([]string, int) {
	for i, inst := range insts {
		if strconv.Itoa(meta) == inst[0] {
			BLogger.log.Debug("CommSwap inst:", inst)
			return inst, i
		}
	}
	return nil, -1
}

type keccak256MerkleProof struct {
	path [][]byte
	left []bool
}

// getPath encodes the path of merkle proof as string and returns
func (p *keccak256MerkleProof) getPath() []string {
	path := make([]string, len(p.path))
	for i, h := range p.path {
		path[i] = hex.EncodeToString(h)
	}
	return path
}

// buildProof builds a merkle proof for one element in a merkle tree
func buildProofFromTree(merkles [][]byte, id int) *keccak256MerkleProof {
	path, left := blockchain.GetKeccak256MerkleProofFromTree(merkles, id)
	return &keccak256MerkleProof{path: path, left: left}
}

// buildProof receives a list of data (as bytes) and returns a merkle proof for one element in the list
func buildProof(data [][]byte, id int) *keccak256MerkleProof {
	merkles := blockchain.BuildKeccak256MerkleTree(data)
	BLogger.log.Debugf("BuildProof: %x", merkles[id])
	BLogger.log.Debugf("BuildProof merkles: %x", merkles)
	return buildProofFromTree(merkles, id)
}

// buildInstProof receives a list of instructions (as string) and returns a merkle proof for one instruction in the list
func buildInstProof(insts [][]string, id int) *keccak256MerkleProof {
	flattenInsts, err := blockchain.FlattenAndConvertStringInst(insts)
	if err != nil {
		BLogger.log.Errorf("Cannot flatten instructions: %+v", err)
		return nil
	}
	BLogger.log.Debugf("insts: %v", insts)
	return buildProof(flattenInsts, id)
}

type beaconBlock struct {
	*blockchain.BeaconBlock
}

func (bb *beaconBlock) InstructionMerkleRoot() []byte {
	return bb.Header.InstructionMerkleRoot[:]
}

func (bb *beaconBlock) MetaHash() []byte {
	h := bb.Header.MetaHash()
	return h[:]
}

func (bb *beaconBlock) Sig(ce ConsensusEngine) ([][]byte, []int, error) {
	return ce.ExtractBridgeValidationData(bb)
}

type shardBlock struct {
	*blockchain.ShardBlock
}

func (sb *shardBlock) InstructionMerkleRoot() []byte {
	return sb.Header.InstructionMerkleRoot[:]
}

func (sb *shardBlock) MetaHash() []byte {
	h := sb.Header.MetaHash()
	return h[:]
}

func (sb *shardBlock) Sig(ce ConsensusEngine) ([][]byte, []int, error) {
	return ce.ExtractBridgeValidationData(sb)
}

// findBeaconBlockWithInst finds a beacon block with a specific instruction and the instruction's index; nil if not found
func findBeaconBlockWithInst(beaconBlocks []*blockchain.BeaconBlock, inst []string) (*blockchain.BeaconBlock, int) {
	for _, b := range beaconBlocks {
		for k, blkInst := range b.Body.Instructions {
			diff := false
			for i, part := range blkInst {
				if part != inst[i] {
					diff = true
					break
				}
			}
			if !diff {
				return b, k
			}
		}
	}
	return nil, -1
}

func buildProofResult(
	decodedInst string,
	beaconInstProof *swapProof,
	bridgeInstProof *swapProof,
	beaconHeight string,
	bridgeHeight string,
) jsonresult.GetInstructionProof {
	r := jsonresult.GetInstructionProof{
		Instruction:  decodedInst,
		BeaconHeight: beaconHeight,
		BridgeHeight: bridgeHeight,

		BeaconInstPath:       beaconInstProof.instPath,
		BeaconInstPathIsLeft: beaconInstProof.instPathIsLeft,
		BeaconInstRoot:       beaconInstProof.instRoot,
		BeaconBlkData:        beaconInstProof.blkData,
		BeaconSigs:           beaconInstProof.signerSigs,
		BeaconSigIdxs:        beaconInstProof.sigIdxs,
	}
	if bridgeInstProof != nil {
		r.BridgeInstPath = bridgeInstProof.instPath
		r.BridgeInstPathIsLeft = bridgeInstProof.instPathIsLeft
		r.BridgeInstRoot = bridgeInstProof.instRoot
		r.BridgeBlkData = bridgeInstProof.blkData
		r.BridgeSigs = bridgeInstProof.signerSigs
		r.BridgeSigIdxs = bridgeInstProof.sigIdxs
	}
	return r
}
