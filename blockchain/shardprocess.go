package blockchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain/types"

	"github.com/incognitochain/incognito-chain/blockchain/committeestate"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/pkg/errors"
)

// VerifyPreSignShardBlock Verify Shard Block Before Signing
// Used for PBFT consensus
// this block doesn't have full information (incomplete block)
func (blockchain *BlockChain) VerifyPreSignShardBlock(
	shardBlock *types.ShardBlock,
	committees []incognitokey.CommitteePublicKey,
	shardID byte,
) error {
	//get view that block link to
	preHash := shardBlock.Header.PreviousBlockHash
	view := blockchain.ShardChain[int(shardID)].GetViewByHash(preHash)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if view == nil {
		blockchain.config.Syncker.SyncMissingShardBlock(ctx, "", shardID, preHash)
	}
	var checkShardUntilTimeout = func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return errors.New(fmt.Sprintf("ShardBlock %v link to wrong view (%s)", shardBlock.GetHeight(), preHash.String()))
			default:
				if blockchain.ShardChain[shardID].GetViewByHash(preHash) != nil {
					return nil
				}
				time.Sleep(time.Second)
			}
		}
	}
	if err := checkShardUntilTimeout(ctx); err != nil {
		return err
	}

	curView := view.(*ShardBestState)
	Logger.log.Infof("SHARD %+v | Verify ShardBlock for signing process %d, with hash %+v", shardID, shardBlock.Header.Height, *shardBlock.Hash())
	shardBestState := NewShardBestState()
	if err := shardBestState.cloneShardBestStateFrom(curView); err != nil {
		return err
	}

	// fetch beacon blocks
	previousBeaconHeight := shardBestState.BeaconHeight
	var checkBeaconUntilTimeout = func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return errors.New("Wait for beacon timeout")
			default:
				if shardBlock.Header.BeaconHeight <= blockchain.BeaconChain.GetFinalView().GetHeight() {
					return nil
				}
				time.Sleep(time.Second)
			}
		}
	}
	ctx, _ = context.WithTimeout(context.Background(), time.Second*5)
	if checkBeaconUntilTimeout(ctx) != nil {
		return errors.New(fmt.Sprintf("Beacon %d not ready, latest is %d", shardBlock.Header.BeaconHeight, blockchain.GetBeaconBestState().BeaconHeight))
	}

	beaconBlocks, err := FetchBeaconBlockFromHeight(blockchain, previousBeaconHeight+1, shardBlock.Header.BeaconHeight)
	if err != nil {
		return err
	}

	//========Get Committes For Processing Block
	if shardBestState.shardCommitteeEngine.Version() == committeestate.SELF_SWAP_SHARD_VERSION {
		committees = shardBestState.GetShardCommittee()
	}
	//

	//========Verify shardBlock only
	if err := blockchain.verifyPreProcessingShardBlock(
		shardBestState, shardBlock, beaconBlocks,
		shardID, true, committees); err != nil {
		return err
	}
	//========Verify shardBlock with previous best state

	// Verify shardBlock with previous best state
	// DO NOT verify agg signature in this function
	if err := shardBestState.verifyBestStateWithShardBlock(blockchain, shardBlock, false, shardID, committees); err != nil {
		return err
	}
	//========updateShardBestState best state with new shardBlock
	newBeststate, hashes, _, err := shardBestState.updateShardBestState(blockchain, shardBlock, beaconBlocks, committees)
	if err != nil {
		return err
	}
	shardBestState.shardCommitteeEngine.AbortUncommittedShardState()
	//========Post verififcation: verify new beaconstate with corresponding shardBlock
	if err := newBeststate.verifyPostProcessingShardBlock(shardBlock, shardID, hashes); err != nil {
		return err
	}
	Logger.log.Infof("SHARD %+v | Block %d, with hash %+v is VALID for 🖋 signing", shardID, shardBlock.GetHeight(), shardBlock.Hash().String())
	return nil
}

// InsertShardBlock Insert Shard Block into blockchain
// this block must have full information (complete block)
func (blockchain *BlockChain) InsertShardBlock(shardBlock *types.ShardBlock, shouldValidate bool) error {
	blockHash := shardBlock.Header.Hash()
	blockHeight := shardBlock.Header.Height
	shardID := shardBlock.Header.ShardID
	preHash := shardBlock.Header.PreviousBlockHash
	shouldValidate = true

	Logger.log.Infof("SHARD %+v | InsertShardBlock %+v with hash %+v \nPrev hash: %+v", shardID, blockHeight, blockHash, preHash)
	blockchain.ShardChain[int(shardID)].insertLock.Lock()
	defer blockchain.ShardChain[int(shardID)].insertLock.Unlock()
	//startTimeInsertShardBlock := time.Now()

	//check if view is committed
	checkView := blockchain.ShardChain[int(shardID)].GetViewByHash(blockHash)
	if checkView != nil {
		return nil
	}

	//get view that block link to
	preView := blockchain.ShardChain[int(shardID)].GetViewByHash(preHash)
	if preView == nil {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultMaxBlockSyncTime)
		defer cancel()
		blockchain.config.Syncker.SyncMissingShardBlock(ctx, "", shardID, preHash)
		return NewBlockChainError(InsertShardBlockError, fmt.Errorf("ShardBlock %v link to wrong view (%s)", blockHeight, preHash.String()))
	}
	curView := preView.(*ShardBestState)

	if blockHeight != curView.ShardHeight+1 {
		return NewBlockChainError(InsertShardBlockError, fmt.Errorf("Not expected height, current view height %+v, incomming block height %+v", curView.ShardHeight, blockHeight))
	}
	// fetch beacon blocks
	previousBeaconHeight := curView.BeaconHeight
	beaconBlocks, err := FetchBeaconBlockFromHeight(blockchain, previousBeaconHeight+1, shardBlock.Header.BeaconHeight)
	if err != nil {
		return NewBlockChainError(FetchBeaconBlocksError, err)
	}

	committees := []incognitokey.CommitteePublicKey{}
	if curView.shardCommitteeEngine.Version() == committeestate.SELF_SWAP_SHARD_VERSION {
		committees = curView.GetShardCommittee()
	} else {
		committees, err = blockchain.GetShardCommitteeFromBeaconHash(shardBlock.Header.CommitteeFromBlock, shardID)
		if err != nil {
			return err
		}
	}

	//====== TODO: @hung remove after testing sync testnet and mainnet
	// Verify Transaction
	// get beacon height from shard block
	beaconHeight := shardBlock.Header.BeaconHeight
	Logger.log.Infof("SHARD %+v | Verify Transaction From Block 🔍 %+v, block height %+v with hash %+v", len(shardBlock.Body.Transactions), shardID, blockHeight, blockHash)
	if err := blockchain.verifyTransactionFromNewBlock(shardID, shardBlock.Body.Transactions, int64(beaconHeight), curView); err != nil {
		return NewBlockChainError(TransactionFromNewBlockError, err)
	}
	//======

	if shouldValidate {
		Logger.log.Infof("SHARD %+v | Verify Pre Processing, block height %+v with hash %+v", shardID, blockHeight, blockHash)
		if err := blockchain.verifyPreProcessingShardBlock(curView, shardBlock, beaconBlocks, shardID, false, committees); err != nil {
			return err
		}
	} else {
		Logger.log.Infof("SHARD %+v | SKIP Verify Pre Processing, block height %+v with hash %+v", shardID, blockHeight, blockHash)
	}

	if shouldValidate {
		// Verify block with previous best state
		Logger.log.Infof("SHARD %+v | Verify BestState With Shard Block, block height %+v with hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, blockHash)
		if err := curView.verifyBestStateWithShardBlock(blockchain, shardBlock, true, shardID, committees); err != nil {
			return err
		}

	} else {
		Logger.log.Infof("SHARD %+v | SKIP Verify Best State With Shard Block, Shard Block Height %+v with hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, blockHash)
	}
	Logger.log.Infof("SHARD %+v | Update ShardBestState, block height %+v with hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, blockHash)

	newBestState, hashes, committeeChange, err := curView.updateShardBestState(blockchain, shardBlock, beaconBlocks, committees)
	if err != nil {
		return err
	}
	var err2 error
	defer func() {
		if err2 != nil {
			newBestState.shardCommitteeEngine.AbortUncommittedShardState()
		}
	}()

	//========Post verification: verify new beaconstate with corresponding block
	if shouldValidate {
		Logger.log.Infof("SHARD %+v | Verify Post Processing, block height %+v with hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, blockHash)
		if err2 = newBestState.verifyPostProcessingShardBlock(shardBlock, shardID, hashes); err != nil {
			return err2
		}
	} else {
		Logger.log.Infof("SHARD %+v | SKIP Verify Post Processing, block height %+v with hash %+v \n", shardID, blockHeight, blockHash)
	}

	Logger.log.Infof("SHARD %+v | Update Beacon Instruction, block height %+v with hash %+v \n", shardID, blockHeight, blockHash)
	err2 = blockchain.processSalaryInstructions(newBestState.rewardStateDB, beaconBlocks, shardID)
	if err2 != nil {
		return err2
	}

	Logger.log.Infof("SHARD %+v | Update Committee State Block Height %+v with hash %+v",
		newBestState.ShardID, shardBlock.Header.Height, blockHash)
	if err2 = newBestState.shardCommitteeEngine.Commit(hashes); err2 != nil {
		return err2
	}

	Logger.log.Infof("SHARD %+v | Store New Shard Block And Update Data, block height %+v with hash %+v \n", shardID, blockHeight, blockHash)
	err2 = blockchain.processStoreShardBlock(newBestState, shardBlock, committeeChange, beaconBlocks)
	if err2 != nil {
		return err2
	}

	blockchain.removeOldDataAfterProcessingShardBlock(shardBlock, shardID)
	go blockchain.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.NewShardblockTopic, shardBlock))
	go blockchain.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.ShardBeststateTopic, newBestState))
	Logger.log.Infof("SHARD %+v | Finish Insert new block %d, with hash %+v 🔗, "+
		"Found 🔎 %+v transactions, "+
		"%+v cross shard transactions, "+
		"%+v instruction",
		shardBlock.Header.ShardID, shardBlock.Header.Height, blockHash,
		len(shardBlock.Body.Transactions), len(shardBlock.Body.CrossTransactions), len(shardBlock.Body.Instructions))

	return nil
}

// updateNumOfBlocksByProducers updates number of blocks produced by producers to shard best state
func (shardBestState *ShardBestState) updateNumOfBlocksByProducers(shardBlock *types.ShardBlock) {
	isSwapInstContained := false
	for _, inst := range shardBlock.Body.Instructions {
		if len(inst) > 0 && inst[0] == instruction.SWAP_ACTION {
			isSwapInstContained = true
			break
		}
	}
	producer := shardBlock.GetProducerPubKeyStr()
	if isSwapInstContained {
		// reset number of blocks produced by producers
		shardBestState.NumOfBlocksByProducers = map[string]uint64{
			producer: 1,
		}
	} else {
		// Update number of blocks produced by producers in epoch
		numOfBlks, found := shardBestState.NumOfBlocksByProducers[producer]
		if !found {
			shardBestState.NumOfBlocksByProducers[producer] = 1
		} else {
			shardBestState.NumOfBlocksByProducers[producer] = numOfBlks + 1
		}
	}
}

// verifyPreProcessingShardBlock DOES NOT verify new block with best state
// DO NOT USE THIS with GENESIS BLOCK
// Verification condition:
//	- Producer Address is not empty
//	- ShardID: of received block same shardID
//	- Version: shard block version is one of pre-defined versions
//	- Parent (previous) block must be found in database ( current block point to an exist block in database )
//	- Height: parent block height + 1
//	- Epoch: blockHeight % Epoch ? Parent Epoch + 1 : Current Epoch
//	- Timestamp: block timestamp must be greater than previous block timestamp
//	- TransactionRoot: rebuild transaction root from txs in block and compare with transaction root in header
//	- ShardTxRoot: rebuild shard transaction root from txs in block and compare with shard transaction root in header
//	- CrossOutputCoinRoot: rebuild cross shard output root from cross output coin in block and compare with cross shard output coin
//		+ cross output coin must be re-created (from cross shard block) if verify block for signing
//	- InstructionRoot: rebuild instruction root from instructions and txs in block and compare with instruction root in header
//		+ instructions must be re-created (from beacon block and instruction) if verify block for signing
//	- InstructionMerkleRoot: rebuild instruction root from instructions and txs in block and compare with instruction root in header
//	- TotalTxFee: calculate tx token fee from all transaction in block then compare with header
//	- CrossShars: Verify Cross Shard Bitmap
//	- BeaconHeight: fetch beacon hash using beacon height in current shard block
//	- BeaconHash: compare beacon hash in database with beacon hash in shard block
//	- Verify swap instruction
//	- Validate transaction created from miner via instruction
//	- Validate Response Transaction From Transaction with Metadata
//	- ALL Transaction in block: see in verifyTransactionFromNewBlock
func (blockchain *BlockChain) verifyPreProcessingShardBlock(curView *ShardBestState,
	shardBlock *types.ShardBlock, beaconBlocks []*types.BeaconBlock,
	shardID byte, isPreSign bool, committees []incognitokey.CommitteePublicKey) error {
	startTimeVerifyPreProcessingShardBlock := time.Now()
	Logger.log.Debugf("SHARD %+v | Begin verifyPreProcessingShardBlock Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	if shardBlock.Header.ShardID != shardID {
		return NewBlockChainError(WrongShardIDError, fmt.Errorf("Expect receive shardBlock from Shard ID %+v but get %+v", shardID, shardBlock.Header.ShardID))
	}

	beaconHeight := curView.BeaconHeight
	for _, v := range beaconBlocks {
		if v.GetHeight() >= beaconHeight {
			beaconHeight = v.GetHeight()
		}
	}

	if curView.shardCommitteeEngine.Version() == committeestate.SLASHING_VERSION {
		committees = curView.GetShardCommittee()
		if beaconHeight <= curView.BeaconHeight {
			Logger.log.Info("Waiting For Beacon Produce Block beaconHeight %+v curView.BeaconHeight %+v",
				beaconHeight, curView.BeaconHeight)
			return NewBlockChainError(WrongBlockHeightError, errors.New("Waiting For Beacon Produce Block"))
		}
	}

	if shardBlock.Header.Height > curView.ShardHeight+1 {
		return NewBlockChainError(WrongBlockHeightError, fmt.Errorf("Expect shardBlock height %+v but get %+v", curView.ShardHeight+1, shardBlock.Header.Height))
	}
	// Verify parent hash exist or not
	previousBlockHash := shardBlock.Header.PreviousBlockHash
	previousShardBlockData, err := rawdbv2.GetShardBlockByHash(blockchain.GetShardChainDatabase(shardID), previousBlockHash)
	if err != nil {
		return NewBlockChainError(FetchPreviousBlockError, err)
	}

	previousShardBlock := types.ShardBlock{}
	err = json.Unmarshal(previousShardBlockData, &previousShardBlock)
	if err != nil {
		return NewBlockChainError(UnmashallJsonShardBlockError, err)
	}
	// Verify shardBlock height with parent shardBlock
	if previousShardBlock.Header.Height+1 != shardBlock.Header.Height {
		return NewBlockChainError(WrongBlockHeightError, fmt.Errorf("Expect receive shardBlock height %+v but get %+v", previousShardBlock.Header.Height+1, shardBlock.Header.Height))
	}
	// Verify timestamp with parent shardBlock
	if shardBlock.Header.Timestamp <= previousShardBlock.Header.Timestamp {
		return NewBlockChainError(WrongTimestampError, fmt.Errorf("Expect receive shardBlock has timestamp must be greater than %+v but get %+v", previousShardBlock.Header.Timestamp, shardBlock.Header.Timestamp))
	}
	// Verify transaction root
	txMerkleTree := Merkle{}.BuildMerkleTreeStore(shardBlock.Body.Transactions)
	txRoot := &common.Hash{}
	if len(txMerkleTree) > 0 {
		txRoot = txMerkleTree[len(txMerkleTree)-1]
	}
	if !bytes.Equal(shardBlock.Header.TxRoot.GetBytes(), txRoot.GetBytes()) && (blockchain.config.ChainParams.Net == Testnet && shardBlock.Header.Height != 487260 && shardBlock.Header.Height != 487261 && shardBlock.Header.Height != 494144) {
		return NewBlockChainError(TransactionRootHashError, fmt.Errorf("Expect transaction root hash %+v but get %+v", shardBlock.Header.TxRoot, txRoot))
	}
	// Verify ShardTx Root
	_, shardTxMerkleData := CreateShardTxRoot(shardBlock.Body.Transactions)
	shardTxRoot := shardTxMerkleData[len(shardTxMerkleData)-1]
	if !bytes.Equal(shardBlock.Header.ShardTxRoot.GetBytes(), shardTxRoot.GetBytes()) {
		return NewBlockChainError(ShardTransactionRootHashError, fmt.Errorf("Expect shard transaction root hash %+v but get %+v", shardBlock.Header.ShardTxRoot, shardTxRoot))
	}
	// Verify crossTransaction coin
	if !VerifyMerkleCrossTransaction(shardBlock.Body.CrossTransactions, shardBlock.Header.CrossTransactionRoot) {
		return NewBlockChainError(CrossShardTransactionRootHashError, fmt.Errorf("Expect cross shard transaction root hash %+v", shardBlock.Header.CrossTransactionRoot))
	}
	// Verify Action
	txInstructions, err := CreateShardInstructionsFromTransactionAndInstruction(shardBlock.Body.Transactions, blockchain, shardID)
	if err != nil {
		Logger.log.Error(err)
		return NewBlockChainError(ShardIntructionFromTransactionAndInstructionError, err)
	}

	if !isPreSign {
		totalInstructions := []string{}
		for _, value := range txInstructions {
			totalInstructions = append(totalInstructions, value...)
		}
		for _, value := range shardBlock.Body.Instructions {
			totalInstructions = append(totalInstructions, value...)
		}
		Logger.log.Infof("Total Instruction \n %+v ", totalInstructions)
		if hash, ok := verifyHashFromStringArray(totalInstructions, shardBlock.Header.InstructionsRoot); !ok {
			return NewBlockChainError(InstructionsHashError, fmt.Errorf("Expect instruction hash to be %+v but get %+v at block %+v hash %+v", shardBlock.Header.InstructionsRoot, hash, shardBlock.Header.Height, shardBlock.Hash().String()))
		}
	}
	totalTxsFee := make(map[common.Hash]uint64)
	for _, tx := range shardBlock.Body.Transactions {
		totalTxsFee[*tx.GetTokenID()] += tx.GetTxFee()
		txType := tx.GetType()
		if txType == common.TxCustomTokenPrivacyType {
			txCustomPrivacy := tx.(*transaction.TxCustomTokenPrivacy)
			totalTxsFee[*txCustomPrivacy.GetTokenID()] = txCustomPrivacy.GetTxFeeToken()
		}
	}
	tokenIDsfromTxs := make([]common.Hash, 0)
	for tokenID := range totalTxsFee {
		tokenIDsfromTxs = append(tokenIDsfromTxs, tokenID)
	}
	sort.Slice(tokenIDsfromTxs, func(i int, j int) bool {
		res, _ := tokenIDsfromTxs[i].Cmp(&tokenIDsfromTxs[j])
		return res == -1
	})
	tokenIDsfromBlock := make([]common.Hash, 0)
	for tokenID := range shardBlock.Header.TotalTxsFee {
		tokenIDsfromBlock = append(tokenIDsfromBlock, tokenID)
	}
	sort.Slice(tokenIDsfromBlock, func(i int, j int) bool {
		res, _ := tokenIDsfromBlock[i].Cmp(&tokenIDsfromBlock[j])
		return res == -1
	})
	if len(tokenIDsfromTxs) != len(tokenIDsfromBlock) {
		return NewBlockChainError(WrongBlockTotalFeeError, fmt.Errorf("Expect Total Fee to be equal, From Txs %+v, From Block %+v", len(tokenIDsfromTxs), len(tokenIDsfromBlock)))
	}
	for i, tokenID := range tokenIDsfromTxs {
		if !tokenIDsfromTxs[i].IsEqual(&tokenIDsfromBlock[i]) {
			return NewBlockChainError(WrongBlockTotalFeeError, fmt.Errorf("Expect Total Fee to be equal, From Txs %+v, From Block %+v", tokenIDsfromTxs[i], tokenIDsfromBlock[i]))
		}
		if totalTxsFee[tokenID] != shardBlock.Header.TotalTxsFee[tokenID] {
			return NewBlockChainError(WrongBlockTotalFeeError, fmt.Errorf("Expect Total Fee to be equal, From Txs %+v, From Block Header %+v", totalTxsFee[tokenID], shardBlock.Header.TotalTxsFee[tokenID]))
		}
	}
	// Verify Cross Shards
	crossShards := createCrossShardByteArray(shardBlock.Body.Transactions, shardID)
	if len(crossShards) != len(shardBlock.Header.CrossShardBitMap) {
		return NewBlockChainError(CrossShardBitMapError, fmt.Errorf("Expect number of cross shardID is %+v but get %+v", len(shardBlock.Header.CrossShardBitMap), len(crossShards)))
	}
	for index := range crossShards {
		if crossShards[index] != shardBlock.Header.CrossShardBitMap[index] {
			return NewBlockChainError(CrossShardBitMapError, fmt.Errorf("Expect Cross Shard Bitmap of shardID %+v is %+v but get %+v", index, shardBlock.Header.CrossShardBitMap[index], crossShards[index]))
		}
	}
	// Check if InstructionMerkleRoot is the root of merkle tree containing all instructions in this shardBlock
	flattenTxInsts, err := FlattenAndConvertStringInst(txInstructions)
	if err != nil {
		return NewBlockChainError(FlattenAndConvertStringInstError, fmt.Errorf("Instruction from Tx: %+v", err))
	}
	flattenInsts, err := FlattenAndConvertStringInst(shardBlock.Body.Instructions)
	if err != nil {
		return NewBlockChainError(FlattenAndConvertStringInstError, fmt.Errorf("Instruction from shardBlock body: %+v", err))
	}
	insts := append(flattenTxInsts, flattenInsts...) // Order of instructions must be the same as when creating new shard shardBlock
	root := GetKeccak256MerkleRoot(insts)
	if !bytes.Equal(root, shardBlock.Header.InstructionMerkleRoot[:]) {
		return NewBlockChainError(InstructionMerkleRootError, fmt.Errorf("Expect transaction merkle root to be %+v but get %+v", shardBlock.Header.InstructionMerkleRoot, string(root)))
	}
	//Get beacon hash by height in db
	//If hash not found then fail to verify
	beaconHash, err := rawdbv2.GetFinalizedBeaconBlockHashByIndex(blockchain.GetBeaconChainDatabase(), shardBlock.Header.BeaconHeight)
	if err != nil {
		return NewBlockChainError(FetchBeaconBlockHashError, err)
	}

	//Hash in db must be equal to hash in shard shardBlock
	newHash, err := common.Hash{}.NewHash(shardBlock.Header.BeaconHash.GetBytes())
	if err != nil {
		return NewBlockChainError(HashError, err)
	}

	if !newHash.IsEqual(beaconHash) {
		return NewBlockChainError(BeaconBlockNotCompatibleError, fmt.Errorf("Expect beacon shardBlock hash to be %+v but get %+v", beaconHash.String(), newHash.String()))
	}

	// Verify response transactions
	instsForValidations := [][]string{}
	instsForValidations = append(instsForValidations, shardBlock.Body.Instructions...)
	for _, beaconBlock := range beaconBlocks {
		instsForValidations = append(instsForValidations, beaconBlock.Body.Instructions...)
	}
	invalidTxs, err := blockchain.verifyMinerCreatedTxBeforeGettingInBlock(instsForValidations, shardBlock.Body.Transactions, shardID)
	if err != nil {
		return NewBlockChainError(TransactionCreatedByMinerError, err)
	}
	if len(invalidTxs) > 0 {
		return NewBlockChainError(TransactionCreatedByMinerError, fmt.Errorf("There are %d invalid txs", len(invalidTxs)))
	}
	err = blockchain.ValidateResponseTransactionFromTxsWithMetadata(shardBlock, curView)
	if err != nil {
		return NewBlockChainError(ResponsedTransactionWithMetadataError, err)
	}
	err = blockchain.ValidateResponseTransactionFromBeaconInstructions(curView, shardBlock, beaconBlocks, shardID)
	if err != nil {
		return NewBlockChainError(ResponsedTransactionWithMetadataError, err)
	}
	shardVerifyPreprocesingTimer.UpdateSince(startTimeVerifyPreProcessingShardBlock)
	// Get cross shard shardBlock from pool
	if isPreSign {
		err := blockchain.verifyPreProcessingShardBlockForSigning(curView, shardBlock, beaconBlocks, txInstructions, shardID, committees)
		if err != nil {
			return err
		}
	} else {
		if err := blockchain.config.ConsensusEngine.ValidateBlockCommitteSig(shardBlock, committees); err != nil {
			Logger.log.Errorf("Validate block %v shard %v with committee %v return error %v", shardBlock.GetHeight(), shardBlock.GetShardID(), committees, err)
			return err
		}
	}
	Logger.log.Debugf("SHARD %+v | Finish verifyPreProcessingShardBlock Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	return err
}

// VerifyPreProcessingShardBlockForSigning verify shard block before a validator signs new shard block
//	- Verify Transactions In New Block
//	- Generate Instruction (from beacon), create instruction root and compare instruction root with instruction root in header
//	- Get Cross Output Data from cross shard block (shard pool) and verify cross transaction hash
//	- Get Cross Tx Custom Token from cross shard block (shard pool) then verify
//
func (blockchain *BlockChain) verifyPreProcessingShardBlockForSigning(curView *ShardBestState,
	shardBlock *types.ShardBlock, beaconBlocks []*types.BeaconBlock,
	txInstructions [][]string, shardID byte, committees []incognitokey.CommitteePublicKey) error {
	var err error
	var isOldBeaconHeight = false
	startTimeVerifyPreProcessingShardBlockForSigning := time.Now()
	// Verify Transaction
	//get beacon height from shard block
	beaconHeight := shardBlock.Header.BeaconHeight
	if err := blockchain.verifyTransactionFromNewBlock(shardID, shardBlock.Body.Transactions, int64(beaconHeight), curView); err != nil {
		return NewBlockChainError(TransactionFromNewBlockError, err)
	}
	// Verify Instruction
	beaconInstructions := [][]string{}

	shardCommittee, err := incognitokey.CommitteeKeyListToString(committees)
	if err != nil {
		return err
	}

	shardPendingValidator := curView.GetShardPendingValidator()
	shardPendingValidatorStr := []string{}

	if curView != nil {
		var err error
		shardPendingValidatorStr, err = incognitokey.
			CommitteeKeyListToString(shardPendingValidator)
		if err != nil {
			return err
		}
	}

	beaconInstructions, _, err = blockchain.
		preProcessInstructionFromBeacon(beaconBlocks, curView.ShardID)
	if err != nil {
		return err
	}

	env := committeestate.
		NewShardEnvBuilder().
		BuildShardID(curView.ShardID).
		BuildBeaconInstructions(beaconInstructions).
		BuildNumberOfFixedBlockValidators(NumberOfFixedShardBlockValidators).
		BuildShardHeight(curView.ShardHeight).
		Build()

	committeeChange, err := curView.shardCommitteeEngine.ProcessInstructionFromBeacon(env)
	if err != nil {
		return err
	}
	curView.shardCommitteeEngine.AbortUncommittedShardState()

	instructions := [][]string{}
	if curView.BeaconHeight == shardBlock.Header.BeaconHeight {
		isOldBeaconHeight = true
	}

	shardPendingValidator, err = updateCommiteesWithAddedAndRemovedListValidator(shardPendingValidator,
		committeeChange.ShardSubstituteAdded[curView.ShardID],
		committeeChange.ShardSubstituteRemoved[curView.ShardID])

	if err != nil {
		return NewBlockChainError(ProcessInstructionFromBeaconError, err)
	}

	shardPendingValidatorStr, err = incognitokey.CommitteeKeyListToString(shardPendingValidator)
	if err != nil {
		return NewBlockChainError(ProcessInstructionFromBeaconError, err)
	}

	instructions, _, shardCommittee, err = blockchain.generateInstruction(curView, shardID,
		shardBlock.Header.BeaconHeight, isOldBeaconHeight, beaconBlocks, beaconInstructions,
		shardPendingValidatorStr, shardCommittee)
	if err != nil {
		return NewBlockChainError(GenerateInstructionError, err)
	}
	///
	totalInstructions := []string{}
	for _, value := range txInstructions {
		totalInstructions = append(totalInstructions, value...)
	}
	for _, value := range instructions {
		totalInstructions = append(totalInstructions, value...)
	}

	if len(totalInstructions) != 0 {
		Logger.log.Info("totalInstructions:", totalInstructions)
	}

	if hash, ok := verifyHashFromStringArray(totalInstructions, shardBlock.Header.InstructionsRoot); !ok {
		return NewBlockChainError(InstructionsHashError, fmt.Errorf("Expect instruction hash to be %+v but %+v", shardBlock.Header.InstructionsRoot, hash))
	}
	toShard := shardID
	var toShardAllCrossShardBlock = make(map[byte][]*types.CrossShardBlock)

	// blockchain.config.Syncker.GetCrossShardBlocksForShardValidator(toShard, list map[byte]common.Hash) map[byte][]interface{}
	crossShardRequired := make(map[byte][]uint64)
	for fromShard, crossTransactions := range shardBlock.Body.CrossTransactions {
		for _, crossTransaction := range crossTransactions {
			//fmt.Println("Crossshard from ", fromShard, crossTransaction.BlockHash)
			crossShardRequired[fromShard] = append(crossShardRequired[fromShard], crossTransaction.BlockHeight)
		}
	}
	crossShardBlksFromPool, err := blockchain.config.Syncker.GetCrossShardBlocksForShardValidator(toShard, crossShardRequired)
	if err != nil {
		return NewBlockChainError(CrossShardBlockError, fmt.Errorf("Unable to get required crossShard blocks from pool in time"))
	}
	for sid, v := range crossShardBlksFromPool {
		heightList := make([]uint64, len(v))
		for i, b := range v {
			toShardAllCrossShardBlock[sid] = append(toShardAllCrossShardBlock[sid], b.(*types.CrossShardBlock))
			heightList[i] = b.(*types.CrossShardBlock).GetHeight()
		}
		Logger.log.Infof("Shard %v, GetCrossShardBlocksForShardValidator from shard %v: %v", toShard, sid, heightList)

	}
	for fromShard, crossTransactions := range shardBlock.Body.CrossTransactions {
		toShardCrossShardBlocks := toShardAllCrossShardBlock[fromShard]
		// if !ok {

		// 	//blockchain.Synker.SyncBlkCrossShard(false, false, []common.Hash{}, heights, fromShard, shardID, "")
		// 	return NewBlockChainError(CrossShardBlockError, fmt.Errorf("Cross Shard Block From Shard %+v Not Found in Pool", fromShard))
		// }
		sort.SliceStable(toShardCrossShardBlocks[:], func(i, j int) bool {
			return toShardCrossShardBlocks[i].Header.Height < toShardCrossShardBlocks[j].Header.Height
		})
		//startHeight := blockchain.GetBestStateShard(toShard).BestCrossShard[fromShard]
		isValids := 0
		for _, crossTransaction := range crossTransactions {
			for index, toShardCrossShardBlock := range toShardCrossShardBlocks {
				//Compare block height and block hash
				if crossTransaction.BlockHeight == toShardCrossShardBlock.Header.Height {
					compareCrossTransaction := types.CrossTransaction{
						TokenPrivacyData: toShardCrossShardBlock.CrossTxTokenPrivacyData,
						OutputCoin:       toShardCrossShardBlock.CrossOutputCoin,
						BlockHash:        *toShardCrossShardBlock.Hash(),
						BlockHeight:      toShardCrossShardBlock.Header.Height,
					}
					targetHash := crossTransaction.Hash()
					hash := compareCrossTransaction.Hash()
					if !hash.IsEqual(&targetHash) {
						return NewBlockChainError(CrossTransactionHashError, fmt.Errorf("Cross Output Coin From New Block %+v not compatible with cross shard block in pool %+v", targetHash, hash))
					}
					if true {
						toShardCrossShardBlocks = toShardCrossShardBlocks[index:]
						isValids++
						break
					}
				}
			}
		}
		if len(crossTransactions) != isValids {
			return NewBlockChainError(CrossShardBlockError, fmt.Errorf("Can't not verify all cross shard block from shard %+v", fromShard))
		}
	}
	shardVerifyPreprocesingForPreSignTimer.UpdateSince(startTimeVerifyPreProcessingShardBlockForSigning)
	return nil
}

// verifyBestStateWithShardBlock will verify the validation of a block with some best state in cache or current best state
// Get beacon state of this block
// For example, new blockHeight is 91 then beacon state of this block must have height 90
// OR new block has previous has is beacon best block hash
//	- New Shard Block has parent (previous) hash is current shard state best block hash (compatible with current beststate)
//	- New Shard Block Height must be compatible with best shard state
//	- New Shard Block has beacon must higher or equal to beacon height of shard best state
func (shardBestState *ShardBestState) verifyBestStateWithShardBlock(blockchain *BlockChain,
	shardBlock *types.ShardBlock, isVerifySig bool,
	shardID byte, committees []incognitokey.CommitteePublicKey) error {
	startTimeVerifyBestStateWithShardBlock := time.Now()
	Logger.log.Debugf("SHARD %+v | Begin VerifyBestStateWithShardBlock Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	//verify producer via index

	if err := blockchain.config.ConsensusEngine.ValidateProducerPosition(shardBlock,
		shardBestState.ShardProposerIdx, committees,
		shardBestState.MinShardCommitteeSize); err != nil {
		return err
	}

	if shardBestState.shardCommitteeEngine.Version() == committeestate.SLASHING_VERSION {
		if !shardBestState.CommitteeFromBlock().IsZeroValue() {
			newCommitteesPubKeys, _ := incognitokey.CommitteeKeyListToString(committees)
			oldCommitteesPubKeys, _ := incognitokey.CommitteeKeyListToString(shardBestState.GetCommittee())
			//Logger.log.Infof("new Committee %+v \n old Committees %+v", newCommitteesPubKeys, oldCommitteesPubKeys)
			temp := common.DifferentElementStrings(oldCommitteesPubKeys, newCommitteesPubKeys)
			if len(temp) != 0 {
				oldBeaconBlock, _, err := blockchain.GetBeaconBlockByHash(shardBestState.CommitteeFromBlock())
				if err != nil {
					return err
				}
				newBeaconBlock, _, err := blockchain.GetBeaconBlockByHash(shardBlock.Header.CommitteeFromBlock)
				if err != nil {
					return err
				}
				if oldBeaconBlock.Header.Height >= newBeaconBlock.Header.Height {
					return NewBlockChainError(WrongBlockHeightError,
						fmt.Errorf("Height of New Shard Block's Committee From Block %+v is smaller than current Committee From Block View %+v",
							newBeaconBlock.Header.Hash(), oldBeaconBlock.Header.Hash()))
				}
			}
		}
	}

	// check with current final best state
	// shardBlock can only be insert if it match the current best state
	if !shardBestState.BestBlockHash.IsEqual(&shardBlock.Header.PreviousBlockHash) {
		return NewBlockChainError(ShardBestStateNotCompatibleError, fmt.Errorf("Current Best Block Hash %+v, New Shard Block %+v, Previous Block Hash of New Block %+v", shardBestState.BestBlockHash, shardBlock.Header.Height, shardBlock.Header.PreviousBlockHash))
	}
	if shardBestState.ShardHeight+1 != shardBlock.Header.Height {
		return NewBlockChainError(WrongBlockHeightError, fmt.Errorf("Shard Block height of new Shard Block should be %+v, but get %+v", shardBestState.ShardHeight+1, shardBlock.Header.Height))
	}
	if shardBlock.Header.BeaconHeight < shardBestState.BeaconHeight {
		return NewBlockChainError(ShardBestStateBeaconHeightNotCompatibleError, fmt.Errorf("Shard Block contain invalid beacon height, current beacon height %+v but get %+v ", shardBestState.BeaconHeight, shardBlock.Header.BeaconHeight))
	}
	shardVerifyWithBestStateTimer.UpdateSince(startTimeVerifyBestStateWithShardBlock)
	Logger.log.Debugf("SHARD %+v | Finish VerifyBestStateWithShardBlock Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	return nil
}

// updateShardBestState beststate with new shard block:
//	- New Previous Shard BlockHash
//	- New BestShardBlockHash
//	- New BestBeaconHash
//	- New Best Shard Block
//	- New Best Shard Height
//	- New Beacon Height
//	- ShardProposerIdx of new shard block
//	- Execute stake instruction, store staking transaction (if exist)
//	- Execute assign instruction, add new pending validator (if exist)
//	- Execute swap instruction, swap pending validator and committee (if exist)
func (oldBestState *ShardBestState) updateShardBestState(blockchain *BlockChain,
	shardBlock *types.ShardBlock,
	beaconBlocks []*types.BeaconBlock,
	committees []incognitokey.CommitteePublicKey) (
	*ShardBestState, *committeestate.ShardCommitteeStateHash, *committeestate.CommitteeChange, error) {
	var (
		err     error
		shardID = shardBlock.Header.ShardID
	)
	startTimeUpdateShardBestState := time.Now()
	Logger.log.Debugf("SHARD %+v | Begin update Beststate with new Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	shardBestState := NewShardBestState()
	if err := shardBestState.cloneShardBestStateFrom(oldBestState); err != nil {
		return nil, nil, nil, err
	}
	shardBestState.BestBlockHash = *shardBlock.Hash()
	shardBestState.BestBeaconHash = shardBlock.Header.BeaconHash
	shardBestState.BestBlock = shardBlock
	shardBestState.BestBlockHash = *shardBlock.Hash()
	shardBestState.ShardHeight = shardBlock.Header.Height
	shardBestState.Epoch = shardBlock.Header.Epoch
	shardBestState.BeaconHeight = shardBlock.Header.BeaconHeight
	shardBestState.TotalTxns += uint64(len(shardBlock.Body.Transactions))
	shardBestState.NumTxns = uint64(len(shardBlock.Body.Transactions))
	if shardBlock.Header.Height == 1 {
		shardBestState.ShardProposerIdx = 0
	} else {
		for i, v := range committees {
			b58Str, _ := v.ToBase58()
			if b58Str == shardBlock.Header.Producer {
				shardBestState.ShardProposerIdx = i
				break
			}
		}
	}
	//updateShardBestState best cross shard
	for shardID, crossShardBlock := range shardBlock.Body.CrossTransactions {
		shardBestState.BestCrossShard[shardID] = crossShardBlock[len(crossShardBlock)-1].BlockHeight
	}
	temp := 0
	for _, tx := range shardBlock.Body.Transactions {
		//detect transaction that's not salary
		if !tx.IsSalaryTx() {
			temp++
		}
	}
	shardBestState.TotalTxnsExcludeSalary += uint64(temp)
	beaconInstructions, stakingTx, err := blockchain.
		preProcessInstructionFromBeacon(beaconBlocks, shardBestState.ShardID)
	if err != nil {
		return nil, nil, nil, err
	}
	for stakePublicKey, txHash := range stakingTx {
		shardBestState.StakingTx.Set(stakePublicKey, txHash)
	}

	for _, beaconInstruction := range beaconInstructions {
		swapInstruction, err := instruction.ValidateAndImportSwapInstructionFromString(beaconInstruction)
		if err == nil {
			for _, v := range swapInstruction.OutPublicKeys {
				shardBestState.StakingTx.Remove(v)
				if txID, ok := shardBestState.StakingTx.Get(v); ok {
					if checkReturnStakingTxExistence(txID, shardBlock) {
						shardBestState.StakingTx.Remove(v)
					}
				}
			}
		}
	}

	env := committeestate.
		NewShardEnvBuilder().
		BuildBeaconHeight(shardBestState.BeaconHeight).
		BuildChainParamEpoch(blockchain.config.ChainParams.Epoch).
		BuildEpochBreakPointSwapNewKey(blockchain.config.ChainParams.EpochBreakPointSwapNewKey).
		BuildBeaconInstructions(beaconInstructions).
		BuildMaxShardCommitteeSize(shardBestState.MaxShardCommitteeSize).
		BuildNumberOfFixedBlockValidators(NumberOfFixedShardBlockValidators).
		BuildMinShardCommitteeSize(shardBestState.MinShardCommitteeSize).
		BuildOffset(blockchain.config.ChainParams.Offset).
		BuildShardBlockHash(shardBestState.BestBlockHash).
		BuildShardHeight(shardBestState.ShardHeight).
		BuildShardID(shardID).
		BuildStakingTx(make(map[string]string)).
		BuildSwapOffset(blockchain.config.ChainParams.SwapOffset).
		BuildTxs(shardBlock.Body.Transactions).
		BuildShardInstructions(shardBlock.Body.Instructions).
		BuildCommitteesFromBlock(shardBlock.Header.CommitteeFromBlock).
		BuildCommitteesFromBeaconView(committees).
		BuildShardHeight(shardBlock.Header.Height).
		Build()

	hashes, committeeChange, err := shardBestState.shardCommitteeEngine.UpdateCommitteeState(env)
	if err != nil {
		return nil, nil, nil, NewBlockChainError(UpdateShardCommitteeStateError, err)
	}
	shardUpdateBestStateTimer.UpdateSince(startTimeUpdateShardBestState)
	Logger.log.Debugf("SHARD %+v | Finish update Beststate with new Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	return shardBestState, hashes, committeeChange, nil
}

func (shardBestState *ShardBestState) initShardBestState(blockchain *BlockChain,
	db incdb.Database, genesisShardBlock *types.ShardBlock, genesisBeaconBlock *types.BeaconBlock) error {

	shardBestState.BestBeaconHash = *ChainTestParam.GenesisBeaconBlock.Hash()
	shardBestState.BestBlock = genesisShardBlock
	shardBestState.BestBlockHash = *genesisShardBlock.Hash()
	shardBestState.ShardHeight = genesisShardBlock.Header.Height
	shardBestState.Epoch = genesisShardBlock.Header.Epoch
	shardBestState.BeaconHeight = genesisShardBlock.Header.BeaconHeight
	shardBestState.TotalTxns += uint64(len(genesisShardBlock.Body.Transactions))
	shardBestState.NumTxns = uint64(len(genesisShardBlock.Body.Transactions))
	shardBestState.ShardProposerIdx = 0

	shardBestState.ConsensusAlgorithm = common.BlsConsensus
	shardBestState.NumOfBlocksByProducers = make(map[string]uint64)

	// Get all instructions from beacon here
	instructions, stakingTx, err := blockchain.
		preProcessInstructionFromBeacon([]*types.BeaconBlock{genesisBeaconBlock}, shardBestState.ShardID)
	if err != nil {
		return err
	}

	for stakePublicKey, txHash := range stakingTx {
		shardBestState.StakingTx.Set(stakePublicKey, txHash)
		//if err := statedb.StoreStakerInfoAtShardDB(shardBestState.consensusStateDB, stakePublicKey, txHash); err != nil {
		//	return err
		//}
	}

	env := committeestate.
		NewShardEnvBuilder().
		BuildBeaconHeight(shardBestState.BeaconHeight).
		BuildChainParamEpoch(shardBestState.Epoch).
		BuildEpochBreakPointSwapNewKey(blockchain.config.ChainParams.EpochBreakPointSwapNewKey).
		BuildBeaconInstructions(instructions).
		BuildNumberOfFixedBlockValidators(NumberOfFixedShardBlockValidators).
		BuildMaxShardCommitteeSize(shardBestState.MaxShardCommitteeSize).
		BuildMinShardCommitteeSize(shardBestState.MinShardCommitteeSize).
		BuildOffset(blockchain.config.ChainParams.Offset).
		BuildShardBlockHash(shardBestState.BestBlockHash).
		BuildShardHeight(shardBestState.ShardHeight).
		BuildShardID(shardBestState.ShardID).
		BuildStakingTx(make(map[string]string)).
		BuildSwapOffset(blockchain.config.ChainParams.SwapOffset).
		BuildTxs(genesisShardBlock.Body.Transactions).
		Build()

	shardBestState.shardCommitteeEngine.InitCommitteeState(env)

	//statedb===========================START
	dbAccessWarper := statedb.NewDatabaseAccessWarper(db)
	shardBestState.consensusStateDB, err = statedb.NewWithPrefixTrie(common.EmptyRoot, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.transactionStateDB, err = statedb.NewWithPrefixTrie(common.EmptyRoot, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.featureStateDB, err = statedb.NewWithPrefixTrie(common.EmptyRoot, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.rewardStateDB, err = statedb.NewWithPrefixTrie(common.EmptyRoot, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.slashStateDB, err = statedb.NewWithPrefixTrie(common.EmptyRoot, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.ConsensusStateDBRootHash = common.EmptyRoot
	shardBestState.SlashStateDBRootHash = common.EmptyRoot
	shardBestState.RewardStateDBRootHash = common.EmptyRoot
	shardBestState.FeatureStateDBRootHash = common.EmptyRoot
	shardBestState.TransactionStateDBRootHash = common.EmptyRoot
	//statedb===========================END
	return nil
}

// verifyPostProcessingShardBlock
//	- commitee root
//	- pending validator root
func (shardBestState *ShardBestState) verifyPostProcessingShardBlock(shardBlock *types.ShardBlock, shardID byte,
	hashes *committeestate.ShardCommitteeStateHash) error {
	if !hashes.ShardCommitteeHash.IsEqual(&shardBlock.Header.CommitteeRoot) {
		return NewBlockChainError(ShardCommitteeRootHashError, fmt.Errorf("Expect %+v but get %+v", shardBlock.Header.CommitteeRoot, hashes.ShardCommitteeHash))
	}

	if !hashes.ShardSubstituteHash.IsEqual(&shardBlock.Header.PendingValidatorRoot) {
		return NewBlockChainError(ShardPendingValidatorRootHashError, fmt.Errorf("Expect %+v but get %+v", shardBlock.Header.PendingValidatorRoot, hashes.ShardSubstituteHash))
	}

	startTimeVerifyPostProcessingShardBlock := time.Now()
	Logger.log.Debugf("SHARD %+v | Begin VerifyPostProcessing Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	//if hash, isOk := verifyHashFromMapStringString(shardBestState.StakingTx, shardBlock.Header.StakingTxRoot); !isOk {
	//	return NewBlockChainError(ShardPendingValidatorRootHashError, fmt.Errorf("Expect shard staking root hash to be %+v but get %+v", shardBlock.Header.StakingTxRoot, hash))
	//}
	shardVerifyPostProcessingTimer.UpdateSince(startTimeVerifyPostProcessingShardBlock)
	Logger.log.Debugf("SHARD %+v | Finish VerifyPostProcessing Block with height %+v at hash %+v", shardBlock.Header.ShardID, shardBlock.Header.Height, shardBlock.Hash())
	return nil
}

// Verify Transaction with these condition:
//	1. Validate tx version
//	2. Validate fee with tx size
//	3. Validate type of tx
//	4. Validate with other txs in block:
// 	- Normal Transaction:
// 	- Custom Tx:
//	4.1 Validate Init Custom Token
//	5. Validate sanity data of tx
//	6. Validate data in tx: privacy proof, metadata,...
//	7. Validate tx with blockchain: douple spend, ...
//	8. Check tx existed in block
//	9. Not accept a salary tx
//	10. Check duplicate staker public key in block
//	11. Check duplicate Init Custom Token in block
func (blockchain *BlockChain) verifyTransactionFromNewBlock(shardID byte, txs []metadata.Transaction, beaconHeight int64, curView *ShardBestState) error {
	if len(txs) == 0 {
		return nil
	}
	isEmpty := blockchain.config.TempTxPool.EmptyPool()
	if !isEmpty {
		panic("TempTxPool Is not Empty")
	}
	defer blockchain.config.TempTxPool.EmptyPool()
	listTxs := []metadata.Transaction{}
	for _, tx := range txs {
		if !tx.IsSalaryTx() {
			listTxs = append(listTxs, tx)
		}
	}
	_, err := blockchain.config.TempTxPool.MaybeAcceptBatchTransactionForBlockProducing(shardID, listTxs, beaconHeight, curView)
	if err != nil {
		Logger.log.Errorf("Batching verify transactions from new block err: %+v\n Trying verify one by one", err)
		for index, tx := range listTxs {
			if blockchain.config.TempTxPool.HaveTransaction(tx.Hash()) {
				continue
			}
			_, err1 := blockchain.config.TempTxPool.MaybeAcceptTransactionForBlockProducing(tx, beaconHeight, curView)
			if err1 != nil {
				Logger.log.Errorf("One by one verify txs at index %d error: %+v", index, err1)
				return NewBlockChainError(TransactionFromNewBlockError, fmt.Errorf("Transaction %+v, index %+v get %+v ", *tx.Hash(), index, err1))
			}
		}
	}
	return nil
}

// processStoreShardBlock Store All information after Insert
//	- Shard Block
//	- Shard Best State
//	- Transaction => UTXO, serial number, snd, commitment
//	- Cross Output Coin => UTXO, snd, commmitment
//	- Store transaction metadata:
//		+ Withdraw Metadata
//	- Store incoming cross shard block
//	- Store Burning Confirmation
//	- Update Mempool fee estimator
func (blockchain *BlockChain) processStoreShardBlock(newShardState *ShardBestState, shardBlock *types.ShardBlock, committeeChange *committeestate.CommitteeChange, beaconBlocks []*types.BeaconBlock) error {
	startTimeProcessStoreShardBlock := time.Now()
	shardID := shardBlock.Header.ShardID
	blockHeight := shardBlock.Header.Height
	blockHash := shardBlock.Header.Hash()

	Logger.log.Infof("SHARD %+v | Process store block height %+v at hash %+v", shardBlock.Header.ShardID, blockHeight, *shardBlock.Hash())
	if len(shardBlock.Body.CrossTransactions) != 0 {
		Logger.log.Critical("processStoreShardBlock/CrossTransactions	", shardBlock.Body.CrossTransactions)
	}

	if err := blockchain.CreateAndSaveTxViewPointFromBlock(shardBlock, newShardState.transactionStateDB); err != nil {
		return NewBlockChainError(FetchAndStoreTransactionError, err)
	}

	for index, tx := range shardBlock.Body.Transactions {
		if err := rawdbv2.StoreTransactionIndex(blockchain.GetShardChainDatabase(shardID), *tx.Hash(), shardBlock.Header.Hash(), index); err != nil {
			return NewBlockChainError(FetchAndStoreTransactionError, err)
		}
		// Process Transaction Metadata
		metaType := tx.GetMetadataType()
		if metaType == metadata.WithDrawRewardResponseMeta {
			_, publicKey, amountRes, coinID := tx.GetTransferData()
			err := statedb.RemoveCommitteeReward(newShardState.rewardStateDB, publicKey, amountRes, *coinID)
			if err != nil {
				return NewBlockChainError(RemoveCommitteeRewardError, err)
			}
		}
		Logger.log.Debugf("Transaction in block with hash", blockHash, "and index", index)
	}
	// Store Incomming Cross Shard
	if err := blockchain.CreateAndSaveCrossTransactionViewPointFromBlock(shardBlock, newShardState.transactionStateDB); err != nil {
		return NewBlockChainError(FetchAndStoreCrossTransactionError, err)
	}
	// Save result of BurningConfirm instruction to get proof later
	metas := []string{ // Burning v1: sig on both beacon and bridge
		strconv.Itoa(metadata.BurningConfirmMeta),
		strconv.Itoa(metadata.BurningConfirmForDepositToSCMeta),
	}
	err := blockchain.storeBurningConfirm(newShardState.featureStateDB, shardBlock.Body.Instructions, shardBlock.Header.Height, metas)
	if err != nil {
		return NewBlockChainError(StoreBurningConfirmError, err)
	}
	// Update bridge issuancstore sharde request status
	err = blockchain.updateBridgeIssuanceStatus(newShardState.featureStateDB, shardBlock)
	if err != nil {
		return NewBlockChainError(UpdateBridgeIssuanceStatusError, err)
	}
	// call FeeEstimator for processing
	if feeEstimator, ok := blockchain.config.FeeEstimator[shardBlock.Header.ShardID]; ok {
		err := feeEstimator.RegisterBlock(shardBlock)
		if err != nil {
			Logger.log.Debug(NewBlockChainError(RegisterEstimatorFeeError, err))
		}
	}
	if len(committeeChange.ShardCommitteeAdded[shardID]) > 0 || len(committeeChange.ShardSubstituteAdded[shardID]) > 0 {
		err = statedb.StoreOneShardCommittee(newShardState.consensusStateDB, shardID, committeeChange.ShardCommitteeAdded[shardID])
		if err != nil {
			return NewBlockChainError(StoreShardBlockError, err)
		}
		err = statedb.StoreOneShardSubstitutesValidator(newShardState.consensusStateDB, shardID, committeeChange.ShardSubstituteAdded[shardID])
		if err != nil {
			return NewBlockChainError(StoreShardBlockError, fmt.Errorf("can't get ConsensusStateRootHash of height %+v ,error %+v", newShardState.GetHeight(), err))
		}
	}
	err = statedb.ReplaceOneShardCommittee(newShardState.consensusStateDB, shardID, committeeChange.ShardCommitteeReplaced[shardID])
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	//err = statedb.DeleteOneShardCommittee(newShardState.consensusStateDB, shardID, removedCommittees)
	err = statedb.DeleteOneShardCommittee(newShardState.consensusStateDB, shardID, committeeChange.ShardCommitteeRemoved[shardID])
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	//err = statedb.DeleteOneShardSubstitutesValidator(newShardState.consensusStateDB, shardID, removedSubstitutesValidator)
	err = statedb.DeleteOneShardSubstitutesValidator(newShardState.consensusStateDB, shardID, committeeChange.ShardSubstituteRemoved[shardID])
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}

	// consensus root hash
	consensusRootHash, err := newShardState.consensusStateDB.Commit(true) // Store data to memory
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	err = newShardState.consensusStateDB.Database().TrieDB().Commit(consensusRootHash, false) // Save data to disk database
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}

	newShardState.ConsensusStateDBRootHash = consensusRootHash
	// transaction root hash
	transactionRootHash, err := newShardState.transactionStateDB.Commit(true)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	err = newShardState.transactionStateDB.Database().TrieDB().Commit(transactionRootHash, false)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	newShardState.TransactionStateDBRootHash = transactionRootHash
	// feature root hash
	featureRootHash, err := newShardState.featureStateDB.Commit(true)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	err = newShardState.featureStateDB.Database().TrieDB().Commit(featureRootHash, false)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	newShardState.FeatureStateDBRootHash = featureRootHash
	// reward root hash
	rewardRootHash, err := newShardState.rewardStateDB.Commit(true)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	err = newShardState.rewardStateDB.Database().TrieDB().Commit(rewardRootHash, false)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	newShardState.RewardStateDBRootHash = rewardRootHash
	// slash root hash
	slashRootHash, err := newShardState.slashStateDB.Commit(true)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	err = newShardState.slashStateDB.Database().TrieDB().Commit(slashRootHash, false)
	if err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	newShardState.consensusStateDB.ClearObjects()
	newShardState.transactionStateDB.ClearObjects()
	newShardState.featureStateDB.ClearObjects()
	newShardState.rewardStateDB.ClearObjects()
	newShardState.slashStateDB.ClearObjects()

	batchData := blockchain.GetShardChainDatabase(shardID).NewBatch()
	sRH := ShardRootHash{
		ConsensusStateDBRootHash:   consensusRootHash,
		FeatureStateDBRootHash:     featureRootHash,
		RewardStateDBRootHash:      rewardRootHash,
		SlashStateDBRootHash:       slashRootHash,
		TransactionStateDBRootHash: transactionRootHash,
	}

	if err := rawdbv2.StoreShardRootsHash(batchData, shardID, blockHash, sRH); err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}

	//statedb===========================END
	if err := rawdbv2.StoreShardBlock(batchData, blockHash, shardBlock); err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}
	finalView := blockchain.ShardChain[shardID].multiView.GetFinalView()
	blockchain.ShardChain[shardBlock.Header.ShardID].multiView.AddView(newShardState)
	newFinalView := blockchain.ShardChain[shardID].multiView.GetFinalView()

	storeBlock := newFinalView.GetBlock()

	for finalView == nil || storeBlock.GetHeight() > finalView.GetHeight() {
		err := rawdbv2.StoreFinalizedShardBlockHashByIndex(batchData, shardID, storeBlock.GetHeight(), *storeBlock.Hash())
		if err != nil {
			return NewBlockChainError(StoreBeaconBlockError, err)
		}
		if storeBlock.GetHeight() == 1 {
			break
		}
		prevHash := storeBlock.GetPrevHash()
		newFinalView = blockchain.ShardChain[shardID].multiView.GetViewByHash(prevHash)
		if newFinalView == nil {
			storeBlock, _, err = blockchain.GetShardBlockByHashWithShardID(prevHash, shardID)
			if err != nil {
				panic("Database is corrupt")
			}
		} else {
			storeBlock = newFinalView.GetBlock()
		}
	}

	err = blockchain.BackupShardViews(batchData, shardBlock.Header.ShardID)
	if err != nil {
		panic("Backup shard view error")
	}

	if err := batchData.Write(); err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}

	if !blockchain.config.ChainParams.IsBackup {
		return nil
	}

	backupPoint := false
	for _, bblk := range beaconBlocks {
		if (bblk.GetHeight()+1)%blockchain.config.ChainParams.Epoch == 0 {
			backupPoint = true
		}
	}

	if backupPoint {
		err := blockchain.GetShardChainDatabase(newShardState.ShardID).Backup(fmt.Sprintf("../../backup/shard%d/%d", newShardState.ShardID, newShardState.Epoch))
		if err != nil {
			blockchain.GetShardChainDatabase(newShardState.ShardID).RemoveBackup(fmt.Sprintf("../../backup/shard%d/%d", newShardState.ShardID, newShardState.Epoch))
		}
	}

	shardStoreBlockTimer.UpdateSince(startTimeProcessStoreShardBlock)
	Logger.log.Infof("SHARD %+v | 🔎 %d transactions in block height %+v \n", shardBlock.Header.ShardID, len(shardBlock.Body.Transactions), blockHeight)
	return nil
}

// removeOldDataAfterProcessingShardBlock remove outdate data from pool and beststate
//	- Remove Staking TX in Shard BestState from instruction
//	- Set Shard State for removing old Shard Block in Pool
//	- Remove Old Cross Shard Block
//	- Remove Init Tokens ID in Mempool
//	- Remove Candiates in Mempool
//	- Remove Transaction in Mempool and Block Generator
func (blockchain *BlockChain) removeOldDataAfterProcessingShardBlock(shardBlock *types.ShardBlock, shardID byte) {
	go func() {
		//Remove Candidate In pool
		candidates := []string{}
		for _, tx := range shardBlock.Body.Transactions {
			if blockchain.config.IsBlockGenStarted {
				blockchain.config.CRemovedTxs <- tx
			}
			if tx.GetMetadata() != nil {
				if tx.GetMetadata().GetType() == metadata.ShardStakingMeta || tx.GetMetadata().GetType() == metadata.BeaconStakingMeta {
					stakingMetadata, ok := tx.GetMetadata().(*metadata.StakingMetadata)
					if !ok {
						continue
					}
					candidates = append(candidates, stakingMetadata.CommitteePublicKey)
				}
			}
		}
		go blockchain.config.TxPool.RemoveCandidateList(candidates)
		//Remove tx out of pool
		go blockchain.config.TxPool.RemoveTx(shardBlock.Body.Transactions, true)
	}()
}

func (blockchain *BlockChain) GetShardCommitteeFromBeaconHash(
	committeeFromBlock common.Hash, shardID byte) ([]incognitokey.CommitteePublicKey, error) {
	_, _, err := blockchain.GetBeaconBlockByHash(committeeFromBlock)
	if err != nil {
		return []incognitokey.CommitteePublicKey{}, NewBlockChainError(CommitteeFromBlockNotFoundError, err)
	}

	bRH, err := GetBeaconRootsHashByBlockHash(blockchain.GetBeaconChainDatabase(), committeeFromBlock)
	if err != nil {
		return []incognitokey.CommitteePublicKey{}, NewBlockChainError(CommitteeFromBlockNotFoundError, err)
	}

	stateDB, err := statedb.NewWithPrefixTrie(
		bRH.ConsensusStateDBRootHash, statedb.NewDatabaseAccessWarper(blockchain.GetBeaconChainDatabase()))
	if err != nil {
		return []incognitokey.CommitteePublicKey{}, NewBlockChainError(CommitteeFromBlockNotFoundError, err)
	}
	committees := statedb.GetOneShardCommittee(stateDB, shardID)

	return committees, nil
}
