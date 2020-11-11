package blockchain

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain/committeestate"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/blockchain/btc"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/metadata"
)

type duplicateKeyStakeInstruction struct {
	instructions []*instruction.StakeInstruction
}

func (inst *duplicateKeyStakeInstruction) add(newInst *duplicateKeyStakeInstruction) {
	inst.instructions = append(inst.instructions, newInst.instructions...)
}

type shardInstruction struct {
	stakeInstructions         []*instruction.StakeInstruction
	unstakeInstructions       []*instruction.UnstakeInstruction
	swapInstructions          map[byte][]*instruction.SwapInstruction
	stopAutoStakeInstructions []*instruction.StopAutoStakeInstruction
}

func (shardInstruction *shardInstruction) add(newShardInstruction *shardInstruction) {
	shardInstruction.stakeInstructions = append(shardInstruction.stakeInstructions, newShardInstruction.stakeInstructions...)
	shardInstruction.unstakeInstructions = append(shardInstruction.unstakeInstructions, newShardInstruction.unstakeInstructions...)
	shardInstruction.stopAutoStakeInstructions = append(shardInstruction.stopAutoStakeInstructions, newShardInstruction.stopAutoStakeInstructions...)
	for shardID, swapInstructions := range newShardInstruction.swapInstructions {
		shardInstruction.swapInstructions[shardID] = append(shardInstruction.swapInstructions[shardID], swapInstructions...)
	}
}

// NewBlockBeacon create new beacon block:
// 1. Clone Current Best State
// 2. Build Essential Header Data:
//	- Version: Get Proper version value
//	- Height: Previous block height + 1
//	- Epoch: Increase Epoch if next height mod epoch is 1 (begin of new epoch), otherwise use current epoch value
//	- Round: Get Round Value from consensus
//	- Previous Block Hash: Get Current Best Block Hash
//	- Producer: Get producer value from round and current beacon committee
//	- Consensus type: get from beaacon best state
// 3. Build Body:
//	a. Build Reward Instruction:
//		- These instruction will only be built at the begining of each epoch (for previous committee)
//	b. Get Shard State and Instruction:
//		- These information will be extracted from all shard block, which got from shard to beacon pool
//	c. Create Instruction:
//		- Instruction created from beacon data
//		- Instruction created from shard instructions
// 4. Update Cloned Beacon Best State to Build Root Hash for Header
//	+ Beacon Root Hash will be calculated from new beacon best state (beacon best state after process by this new block)
//	+ Some data may changed if beacon best state is updated:
//		+ Beacon Committee, Pending Validator, Candidate List
//		+ Shard Committee, Pending Validator, Candidate List
// 5. Build Root Hash in Header
//	a. Beacon Committee and Validator Root Hash: Hash from Beacon Committee and Pending Validator
//	b. Beacon Caiddate Root Hash: Hash from Beacon candidate list
//	c. Shard Committee and Validator Root Hash: Hash from Shard Committee and Pending Validator
//	d. Shard Caiddate Root Hash: Hash from Shard candidate list
//	+ These Root Hash will be used to verify that, either Two arbitray Nodes have the same data
//		after they update beacon best state by new block.
//	e. ShardStateHash: shard states from blocks of all shard
//	f. InstructionHash: from instructions in beacon block body
//	g. InstructionMerkleRoot
func (blockchain *BlockChain) NewBlockBeacon(curView *BeaconBestState, version int, proposer string, round int, startTime int64) (*types.BeaconBlock, error) {
	Logger.log.Infof("⛏ Creating Beacon Block %+v", curView.BeaconHeight+1)
	//============Init Variable============
	var err error
	var epoch uint64
	beaconBlock := types.NewBeaconBlock()
	beaconBestState := NewBeaconBestState()
	rewardByEpochInstruction := [][]string{}
	// produce new block with current beststate
	err = beaconBestState.cloneBeaconBestStateFrom(curView)
	if err != nil {
		return nil, err
	}
	//======Build Header Essential Data=======
	beaconBlock.Header.Version = version
	beaconBlock.Header.Height = beaconBestState.BeaconHeight + 1
	if (beaconBestState.BeaconHeight+1)%blockchain.config.ChainParams.Epoch == 1 {
		epoch = beaconBestState.Epoch + 1
	} else {
		epoch = beaconBestState.Epoch
	}
	beaconBlock.Header.ConsensusType = beaconBestState.ConsensusAlgorithm
	beaconBlock.Header.Producer = proposer
	beaconBlock.Header.ProducerPubKeyStr = proposer
	beaconBlock.Header.Height = beaconBestState.BeaconHeight + 1
	beaconBlock.Header.Epoch = epoch
	beaconBlock.Header.Round = round
	beaconBlock.Header.PreviousBlockHash = beaconBestState.BestBlockHash
	BLogger.log.Infof("Producing block: %d (epoch %d)", beaconBlock.Header.Height, beaconBlock.Header.Epoch)
	//=====END Build Header Essential Data=====
	//============Build body===================
	portalParams := blockchain.GetPortalParams(beaconBlock.GetHeight())
	rewardForCustodianByEpoch := map[common.Hash]uint64{}

	if beaconBestState.BeaconHeight%blockchain.config.ChainParams.Epoch == 0 {
		featureStateDB := beaconBestState.GetBeaconFeatureStateDB()
		totalLockedCollateral, err := getTotalLockedCollateralInEpoch(featureStateDB)
		if err != nil {
			return nil, NewBlockChainError(GetTotalLockedCollateralError, err)
		}
		isSplitRewardForCustodian := totalLockedCollateral > 0
		percentCustodianRewards := portalParams.MaxPercentCustodianRewards
		if totalLockedCollateral < portalParams.MinLockCollateralAmountInEpoch {
			percentCustodianRewards = portalParams.MinPercentCustodianRewards
		}
		rewardByEpochInstruction, rewardForCustodianByEpoch, err = blockchain.buildRewardInstructionByEpoch(beaconBestState, beaconBlock.Header.Height, beaconBestState.Epoch, curView.GetBeaconRewardStateDB(), isSplitRewardForCustodian, percentCustodianRewards)
		if err != nil {
			return nil, NewBlockChainError(BuildRewardInstructionError, err)
		}
	}

	tempShardState, shardInstruction, duplicateKeyStakeInstruction,
		bridgeInstructions, acceptedRewardInstructions := blockchain.GetShardState(
		beaconBestState, rewardForCustodianByEpoch, portalParams)
	if err != nil {
		return nil, NewBlockChainError(GenerateShardStateError, err)
	}

	shardInstruction.compose()
	Logger.log.Infof("In NewBlockBeacon tempShardState: %+v", tempShardState)
	tempInstruction, err := beaconBestState.GenerateInstruction(
		beaconBlock.Header.Height, shardInstruction, duplicateKeyStakeInstruction,
		bridgeInstructions, acceptedRewardInstructions, blockchain.config.ChainParams.Epoch,
		blockchain.config.ChainParams.RandomTime, blockchain,
		tempShardState,
	)
	if err != nil {
		return nil, err
	}
	if len(rewardByEpochInstruction) != 0 {
		tempInstruction = append(tempInstruction, rewardByEpochInstruction...)
	}
	beaconBlock.Body.Instructions = tempInstruction
	beaconBlock.Body.ShardState = tempShardState
	if len(bridgeInstructions) > 0 {
		BLogger.log.Infof("Producer instructions: %+v", tempInstruction)
	}
	//============End Build Body================
	//============Update Beacon Best State================
	// Process new block with beststate

	_, hashes, _, incurredInstructions, err := beaconBestState.updateBeaconBestState(beaconBlock, blockchain)
	beaconBestState.beaconCommitteeEngine.AbortUncommittedBeaconState()

	if err != nil {
		return nil, err
	}

	tempInstruction = append(tempInstruction, incurredInstructions...)
	beaconBlock.Body.Instructions = tempInstruction
	if len(beaconBlock.Body.Instructions) != 0 {
		Logger.log.Info("Beacon Produce: Beacon Instruction", beaconBlock.Body.Instructions)
	}
	//============Build Header Hash=============
	// calculate hash
	// Shard state hash
	tempShardStateHash, err := generateHashFromShardState(tempShardState)
	if err != nil {
		Logger.log.Error(err)
		return nil, NewBlockChainError(GenerateShardStateError, err)
	}
	// Instruction Hash
	tempInstructionArr := []string{}
	for _, strs := range tempInstruction {
		tempInstructionArr = append(tempInstructionArr, strs...)
	}
	tempInstructionHash, err := generateHashFromStringArray(tempInstructionArr)
	if err != nil {
		Logger.log.Error(err)
		return nil, NewBlockChainError(GenerateInstructionHashError, err)
	}
	// Instruction merkle root
	flattenInsts, err := FlattenAndConvertStringInst(tempInstruction)
	if err != nil {
		return nil, NewBlockChainError(FlattenAndConvertStringInstError, err)
	}
	// add hash to header
	beaconBlock.Header.BeaconCommitteeAndValidatorRoot = hashes.BeaconCommitteeAndValidatorHash
	beaconBlock.Header.BeaconCandidateRoot = hashes.BeaconCandidateHash
	beaconBlock.Header.ShardCandidateRoot = hashes.ShardCandidateHash
	beaconBlock.Header.ShardCommitteeAndValidatorRoot = hashes.ShardCommitteeAndValidatorHash
	beaconBlock.Header.ShardStateHash = tempShardStateHash
	beaconBlock.Header.InstructionHash = tempInstructionHash
	beaconBlock.Header.AutoStakingRoot = hashes.AutoStakeHash
	copy(beaconBlock.Header.InstructionMerkleRoot[:], GetKeccak256MerkleRoot(flattenInsts))
	beaconBlock.Header.Timestamp = startTime
	//============END Build Header Hash=========

	return beaconBlock, nil
}

// GetShardState get Shard To Beacon Block
// Rule:
// 1. Shard To Beacon Blocks will be get from Shard To Beacon Pool (only valid block)
// 2. Process shards independently, for each shard:
//	a. Shard To Beacon Block List must be compatible with current shard state in beacon best state:
//  + Increased continuosly in height (10, 11, 12,...)
//	  Ex: Shard state in beacon best state has height 11 then shard to beacon block list must have first block in list with height 12
//  + Shard To Beacon Block List must have incremental height in list (10, 11, 12,... NOT 10, 12,...)
//  + Shard To Beacon Block List can be verify with and only with current shard committee in beacon best state
//  + DO NOT accept Shard To Beacon Block List that can have two arbitrary blocks that can be verify with two different committee set
//  + If in Shard To Beacon Block List have one block with Swap Instruction, then this block must be the last block in this list (or only block in this list)
// return param:
// 1. shard state
// 2. valid stake instruction
// 3. valid swap instruction
// 4. bridge instructions
// 5. accepted reward instructions
// 6. stop auto staking instructions
func (blockchain *BlockChain) GetShardState(
	beaconBestState *BeaconBestState,
	rewardForCustodianByEpoch map[common.Hash]uint64,
	portalParams PortalParams,
) (map[byte][]types.ShardState, *shardInstruction, *duplicateKeyStakeInstruction,
	[][]string, [][]string) {
	shardStates := make(map[byte][]types.ShardState)
	shardInstruction := &shardInstruction{
		swapInstructions: make(map[byte][]*instruction.SwapInstruction),
	}
	duplicateKeyStakeInstructions := &duplicateKeyStakeInstruction{}
	validStakePublicKeys := []string{}
	validUnstakePublicKeys := make(map[string]bool)
	//Get shard to beacon block from pool
	allShardBlocks := blockchain.GetShardBlockForBeaconProducer(beaconBestState.BestShardHeight)
	keys := []int{}
	for shardID, shardBlocks := range allShardBlocks {
		strs := fmt.Sprintf("GetShardState shardID: %+v, Height", shardID)
		for _, shardBlock := range shardBlocks {
			strs += fmt.Sprintf(" %d", shardBlock.Header.Height)
		}
		Logger.log.Info(strs)
		keys = append(keys, int(shardID))
	}
	sort.Ints(keys)
	//Shard block is a map ShardId -> array of shard block
	bridgeInstructions := [][]string{}
	acceptedRewardInstructions := [][]string{}
	statefulActionsByShardID := map[byte][][]string{}
	for _, v := range keys {
		shardID := byte(v)
		shardBlocks := allShardBlocks[shardID]
		for _, shardBlock := range shardBlocks {
			shardState, newShardInstruction, newDuplicateKeyStakeInstruction,
				bridgeInstruction, acceptedRewardInstruction, statefulActions := blockchain.GetShardStateFromBlock(
				beaconBestState, beaconBestState.BeaconHeight+1, shardBlock, shardID, true, validUnstakePublicKeys, validStakePublicKeys)
			shardStates[shardID] = append(shardStates[shardID], shardState[shardID])
			duplicateKeyStakeInstructions.add(newDuplicateKeyStakeInstruction)
			shardInstruction.add(newShardInstruction)
			bridgeInstructions = append(bridgeInstructions, bridgeInstruction...)
			acceptedRewardInstructions = append(acceptedRewardInstructions, acceptedRewardInstruction)
			tempValidStakePublicKeys := []string{}
			for _, v := range newShardInstruction.stakeInstructions {
				tempValidStakePublicKeys = append(tempValidStakePublicKeys, v.PublicKeys...)
			}
			validStakePublicKeys = append(validStakePublicKeys, tempValidStakePublicKeys...)
			// group stateful actions by shardID
			_, found := statefulActionsByShardID[shardID]
			if !found {
				statefulActionsByShardID[shardID] = statefulActions
			} else {
				statefulActionsByShardID[shardID] = append(statefulActionsByShardID[shardID], statefulActions...)
			}
		}
	}

	// build stateful instructions
	statefulInsts := blockchain.buildStatefulInstructions(beaconBestState.featureStateDB, statefulActionsByShardID, beaconBestState.BeaconHeight+1, rewardForCustodianByEpoch, portalParams)
	bridgeInstructions = append(bridgeInstructions, statefulInsts...)
	return shardStates, shardInstruction, duplicateKeyStakeInstructions, bridgeInstructions, acceptedRewardInstructions
}

// GetShardStateFromBlock get state (information) from shard-to-beacon block
// state will be presented as instruction
//	Return Params:
//	1. ShardState
//	2. Stake Instruction
//	3. Swap Instruction
//	4. Bridge Instruction
//	5. Accepted BlockReward Instruction
//	6. StopAutoStakingInstruction
func (blockchain *BlockChain) GetShardStateFromBlock(
	curView *BeaconBestState,
	newBeaconHeight uint64,
	shardBlock *types.ShardBlock,
	shardID byte,
	isProducer bool,
	validUnstakePublicKeys map[string]bool,
	validStakePublicKeys []string,
) (map[byte]types.ShardState, *shardInstruction, *duplicateKeyStakeInstruction,
	[][]string, []string, [][]string) {
	//Variable Declaration
	shardStates := make(map[byte]types.ShardState)
	duplicateKeyStakeInstruction := &duplicateKeyStakeInstruction{}
	bridgeInstructions := [][]string{}
	acceptedBlockRewardInfo := metadata.NewAcceptedBlockRewardInfo(shardID, shardBlock.Header.TotalTxsFee, shardBlock.Header.Height)
	acceptedRewardInstructions, err := acceptedBlockRewardInfo.GetStringFormat()
	if err != nil {
		// if err then ignore accepted reward instruction
		acceptedRewardInstructions = []string{}
	}
	//Get Shard State from Block
	shardState := types.ShardState{}
	shardState.CrossShard = make([]byte, len(shardBlock.Header.CrossShardBitMap))
	copy(shardState.CrossShard, shardBlock.Header.CrossShardBitMap)
	shardState.Hash = shardBlock.Header.Hash()
	shardState.Height = shardBlock.Header.Height
	shardStates[shardID] = shardState
	instructions, err := CreateShardInstructionsFromTransactionAndInstruction(shardBlock.Body.Transactions, blockchain, shardBlock.Header.ShardID)
	instructions = append(instructions, shardBlock.Body.Instructions...)

	shardInstruction := curView.preProcessInstructionsFromShardBlock(instructions, shardID)
	shardInstruction, duplicateKeyStakeInstruction = curView.
		processStakeInstructionFromShardBlock(shardInstruction, validStakePublicKeys)

	allCommitteeValidatorCandidate := []string{}
	if len(shardInstruction.stopAutoStakeInstructions) != 0 || len(shardInstruction.unstakeInstructions) != 0 {
		// avoid dead lock
		// if producer new block then lock beststate
		allCommitteeValidatorCandidate = curView.getAllCommitteeValidatorCandidateFlattenList()
	}

	shardInstruction = curView.processStopAutoStakeInstructionFromShardBlock(shardInstruction, allCommitteeValidatorCandidate)
	shardInstruction = curView.processUnstakeInstructionFromShardBlock(
		shardInstruction, allCommitteeValidatorCandidate, shardID, validUnstakePublicKeys)

	// Create bridge instruction
	if len(instructions) > 0 || shardBlock.Header.Height%10 == 0 {
		BLogger.log.Debugf("Included shardID %d, block %d, insts: %s", shardID, shardBlock.Header.Height, instructions)
	}
	bridgeInstructionForBlock, err := blockchain.buildBridgeInstructions(
		curView.GetBeaconFeatureStateDB(),
		shardID,
		instructions,
		newBeaconHeight,
	)
	if err != nil {
		BLogger.log.Errorf("Build bridge instructions failed: %s", err.Error())
	}
	// Pick instruction with shard committee's pubkeys to save to beacon block
	confirmInsts := pickBridgeSwapConfirmInst(instructions)
	if len(confirmInsts) > 0 {
		bridgeInstructionForBlock = append(bridgeInstructionForBlock, confirmInsts...)
		BLogger.log.Infof("Beacon block %d found bridge swap confirm stopAutoStakeInstruction in shard block %d: %s", newBeaconHeight, shardBlock.Header.Height, confirmInsts)
	}
	bridgeInstructions = append(bridgeInstructions, bridgeInstructionForBlock...)

	// Collect stateful actions
	statefulActions := blockchain.collectStatefulActions(instructions)
	Logger.log.Infof("Becon Produce: Got Shard Block %+v Shard %+v \n", shardBlock.Header.Height, shardID)
	return shardStates, shardInstruction, duplicateKeyStakeInstruction, bridgeInstructions, acceptedRewardInstructions, statefulActions
}

//GenerateInstruction generate instruction for new beacon block
func (beaconBestState *BeaconBestState) GenerateInstruction(
	newBeaconHeight uint64,
	shardInstruction *shardInstruction,
	duplicateKeyStakeInstruction *duplicateKeyStakeInstruction,
	bridgeInstructions [][]string,
	acceptedRewardInstructions [][]string,
	chainParamEpoch uint64,
	randomTime uint64,
	blockchain *BlockChain,
	shardsState map[byte][]types.ShardState,
) ([][]string, error) {
	instructions := [][]string{}
	instructions = append(instructions, bridgeInstructions...)
	instructions = append(instructions, acceptedRewardInstructions...)
	//=======Swap
	// Shard Swap: both abnormal or normal swap
	var keys []int
	for k := range shardInstruction.swapInstructions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		for _, tempSwapInstruction := range shardInstruction.swapInstructions[byte(shardID)] {
			instructions = append(instructions, tempSwapInstruction.ToString())
		}
	}
	// Stake
	for _, stakeInstruction := range shardInstruction.stakeInstructions {
		instructions = append(instructions, stakeInstruction.ToString())
	}
	// Stop Auto Stake
	for _, stopAutoStakeInstruction := range shardInstruction.stopAutoStakeInstructions {
		instructions = append(instructions, stopAutoStakeInstruction.ToString())
	}
	// Unstake
	for _, unstakeInstruction := range shardInstruction.unstakeInstructions {
		instructions = append(instructions, unstakeInstruction.ToString())
	}

	// Duplicate Staking Instruction
	for _, stakeInstruction := range duplicateKeyStakeInstruction.instructions {
		percentReturns := make([]uint, len(stakeInstruction.PublicKeys))
		for i, _ := range percentReturns {
			percentReturns[i] = 100
		}
		if len(stakeInstruction.TxStakes) > 0 {
			txHash, err := common.Hash{}.NewHashFromStr(stakeInstruction.TxStakes[0])
			if err != nil {
				return [][]string{}, err
			}
			shardID, _, _, _, _, err := blockchain.GetTransactionByHash(*txHash)
			if err != nil {
				return [][]string{}, err
			}
			returnStakingIns := instruction.NewReturnStakeInsWithValue(
				stakeInstruction.PublicKeys,
				shardID,
				stakeInstruction.TxStakes,
			)
			instructions = append(instructions, returnStakingIns.ToString())
		}
	}
	// Random number for Assign Instruction
	if newBeaconHeight%chainParamEpoch > randomTime && !beaconBestState.IsGetRandomNumber {
		var err error
		var chainTimeStamp int64
		if !TestRandom {
			chainTimeStamp, err = blockchain.getChainTimeStamp(newBeaconHeight, chainParamEpoch, beaconBestState.CurrentRandomTimeStamp, beaconBestState.BlockMaxCreateTime)
			if err != nil {
				return [][]string{}, err
			}
		} else {
			chainTimeStamp = beaconBestState.CurrentRandomTimeStamp + 1
		}
		//==================================
		if chainTimeStamp > beaconBestState.CurrentRandomTimeStamp {
			randomInstruction, randomNumber, err := beaconBestState.generateRandomInstruction(beaconBestState.CurrentRandomTimeStamp, blockchain.config.RandomClient)
			if err != nil {
				return [][]string{}, err
			}
			instructions = append(instructions, randomInstruction)
			Logger.log.Infof("Beacon Producer found Random Instruction at Block Height %+v, %+v", randomInstruction, newBeaconHeight)
			assignInstructions, _, _ := beaconBestState.beaconCommitteeEngine.GenerateAssignInstruction(randomNumber, blockchain.config.ChainParams.AssignOffset, beaconBestState.ActiveShards)
			for _, assignInstruction := range assignInstructions {
				instructions = append(instructions, assignInstruction.ToString())
			}
			Logger.log.Info("assignInstructions:", assignInstructions)
		}
	}
	// Generate swap shard instruction at block height %chainParamEpoch == 0
	if newBeaconHeight < blockchain.config.ChainParams.UpgradeCommitteeEngineV2Height {
		if newBeaconHeight%chainParamEpoch == 0 {
			BeaconCommittee := beaconBestState.GetBeaconCommittee()
			beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(BeaconCommittee)
			if err != nil {
				Logger.log.Error(err)
			}
			if common.IndexOfUint64(newBeaconHeight/chainParamEpoch, blockchain.config.ChainParams.EpochBreakPointSwapNewKey) > -1 {
				epoch := newBeaconHeight / chainParamEpoch
				swapBeaconInstructions, beaconCommittee := CreateBeaconSwapActionForKeyListV2(blockchain.config.GenesisParams, beaconCommitteeStr, beaconBestState.MinBeaconCommitteeSize, epoch)
				instructions = append(instructions, swapBeaconInstructions)
				beaconRootInst, _ := buildBeaconSwapConfirmInstruction(beaconCommittee, newBeaconHeight)
				instructions = append(instructions, beaconRootInst)
			}
		}
	} else {
		if newBeaconHeight%chainParamEpoch == 1 {
			// Generate request shard swap instruction, only available after upgrade to BeaconCommitteeEngineV2
			env := beaconBestState.NewBeaconCommitteeStateEnvironment(blockchain.config.ChainParams)
			env.LatestShardsState = shardsState
			swapShardInstructions, err := beaconBestState.beaconCommitteeEngine.GenerateAllSwapShardInstructions(env)
			if err != nil {
				return [][]string{}, err
			}
			for _, swapShardInstruction := range swapShardInstructions {
				instructions = append(instructions, swapShardInstruction.ToString())
			}
		}
	}

	return instructions, nil
}

// ["random" "{nonce}" "{blockheight}" "{timestamp}" "{bitcoinTimestamp}"]
func (beaconBestState *BeaconBestState) generateRandomInstruction(timestamp int64, randomClient btc.RandomClient) ([]string, int64, error) {
	if !TestRandom {
		var (
			blockHeight    int
			chainTimestamp int64
			nonce          int64
			err            error
		)
		startTime := time.Now()
		for {
			Logger.log.Debug("GetNonceByTimestamp", timestamp)
			blockHeight, chainTimestamp, nonce, err = randomClient.GetNonceByTimestamp(startTime, beaconBestState.BlockMaxCreateTime, timestamp)
			if err == nil {
				break
			} else {
				Logger.log.Error("generateRandomInstruction", err)
			}
			if time.Since(startTime).Seconds() > beaconBestState.BlockMaxCreateTime.Seconds() {
				return []string{}, -1, NewBlockChainError(GenerateInstructionError, fmt.Errorf("Get Random Number By Timestmap %+v Timeout", timestamp))
			}
			time.Sleep(time.Millisecond * 500)
		}
		randomInstruction := instruction.NewRandomInstruction().
			SetBtcBlockHeight(blockHeight).
			SetCheckPointTime(timestamp).
			SetNonce(nonce).
			SetBtcBlockTime(chainTimestamp)
		return randomInstruction.ToString(), nonce, nil
	} else {
		ran := rand.New(rand.NewSource(timestamp))
		randInt := ran.Int63()
		randomInstruction := instruction.NewRandomInstruction().
			SetBtcBlockHeight(int(timestamp)).
			SetCheckPointTime(timestamp).
			SetBtcBlockTime(timestamp + 1).
			SetNonce(randInt)
		return randomInstruction.ToString(), randInt, nil
	}
}

func (blockchain *BlockChain) getChainTimeStamp(
	newBeaconHeight uint64,
	chainParamEpoch uint64,
	currentRandomTimeStamp int64,
	blockMaxCreateTime time.Duration,
) (int64, error) {
	var chainTimeStamp int64
	var err error
	if newBeaconHeight%chainParamEpoch == chainParamEpoch-1 {
		startTime := time.Now()
		for {
			Logger.log.Criticalf("Block %+v, Enter final block of epoch but still no random number", newBeaconHeight)
			chainTimeStamp, err = blockchain.config.RandomClient.GetCurrentChainTimeStamp()
			if err != nil {
				Logger.log.Error(err)
			} else {
				if chainTimeStamp < currentRandomTimeStamp {
					Logger.log.Infof("Final Block %+v in Epoch but still haven't found new random number", newBeaconHeight)
				} else {
					break
				}
			}
			if time.Since(startTime).Seconds() > blockMaxCreateTime.Seconds() {
				return 0, NewBlockChainError(GenerateInstructionError, fmt.Errorf("Get Current Chain Timestamp for New Block Height %+v Timeout", newBeaconHeight))
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		Logger.log.Criticalf("Block %+v, finding random number", newBeaconHeight)
		chainTimeStamp, err = blockchain.config.RandomClient.GetCurrentChainTimeStamp()
		if err != nil {
			Logger.log.Error(err)
		}
		return 0, nil
	}
	return chainTimeStamp, nil
}

func CreateBeaconSwapActionForKeyListV2(
	genesisParam *GenesisParams,
	beaconCommittees []string,
	minCommitteeSize int,
	epoch uint64,
) ([]string, []string) {
	swapInstruction, newBeaconCommittees := GetBeaconSwapInstructionKeyListV2(genesisParam, epoch)
	remainBeaconCommittees := beaconCommittees[minCommitteeSize:]
	return swapInstruction, append(newBeaconCommittees, remainBeaconCommittees...)
}

func (beaconBestState *BeaconBestState) postProcessIncurredInstructions(instructions [][]string) error {

	for _, inst := range instructions {
		switch inst[0] {
		case instruction.RETURN_ACTION:
			returnStakingIns, err := instruction.ValidateAndImportReturnStakingInstructionFromString(inst)
			if err != nil {
				return err
			}
			err = statedb.DeleteStakerInfo(beaconBestState.consensusStateDB, returnStakingIns.PublicKeysStruct)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (beaconBestState *BeaconBestState) preProcessInstructionsFromShardBlock(instructions [][]string, shardID byte) *shardInstruction {
	res := &shardInstruction{
		swapInstructions: make(map[byte][]*instruction.SwapInstruction),
	}
	// extract instructions
	for _, inst := range instructions {
		if len(inst) > 0 {
			if inst[0] == instruction.STAKE_ACTION {
				if err := instruction.ValidateStakeInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Stake Instruction Error %+v", err)
					continue
				}
				tempStakeInstruction := instruction.ImportStakeInstructionFromString(inst)
				res.stakeInstructions = append(res.stakeInstructions, tempStakeInstruction)
			}
			if inst[0] == instruction.SWAP_ACTION {
				// validate swap instruction
				// only allow shard to swap committee for it self
				if err := instruction.ValidateSwapInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Swap Instruction Error %+v", err)
					continue
				}
				tempSwapInstruction := instruction.ImportSwapInstructionFromString(inst)
				res.swapInstructions[shardID] = append(res.swapInstructions[shardID], tempSwapInstruction)
			}
			if inst[0] == instruction.STOP_AUTO_STAKE_ACTION {
				if err := instruction.ValidateStopAutoStakeInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Stop Auto Stake Instruction Error %+v", err)
					continue
				}
				tempStopAutoStakeInstruction := instruction.ImportStopAutoStakeInstructionFromString(inst)
				res.stopAutoStakeInstructions = append(res.stopAutoStakeInstructions, tempStopAutoStakeInstruction)
			}
			if inst[0] == instruction.UNSTAKE_ACTION {
				if err := instruction.ValidateUnstakeInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Stop Auto Stake Instruction Error %+v", err)
					continue
				}
				tempUnstakeInstruction := instruction.ImportUnstakeInstructionFromString(inst)
				res.unstakeInstructions = append(res.unstakeInstructions, tempUnstakeInstruction)
			}
		}
	}

	if len(res.stakeInstructions) != 0 {
		Logger.log.Info("Beacon Producer/ Process Stakers List ", res.stakeInstructions)
	}
	if len(res.swapInstructions[shardID]) != 0 {
		Logger.log.Info("Beacon Producer/ Process Stakers List ", res.swapInstructions[shardID])
	}

	return res
}

func (beaconBestState *BeaconBestState) processStakeInstructionFromShardBlock(
	shardInstructions *shardInstruction, validStakePublicKeys []string) (
	*shardInstruction, *duplicateKeyStakeInstruction) {

	duplicateKeyStakeInstruction := &duplicateKeyStakeInstruction{}
	newShardInstructions := shardInstructions
	stakeInstructions := []*instruction.StakeInstruction{}
	stakeShardPublicKeys := []string{}
	stakeShardTx := []string{}
	stakeShardRewardReceiver := []string{}
	stakeShardAutoStaking := []bool{}
	tempValidStakePublicKeys := []string{}

	// Process Stake Instruction form Shard Block
	// Validate stake instruction => extract only valid stake instruction
	for _, stakeInstruction := range shardInstructions.stakeInstructions {
		tempStakePublicKey := stakeInstruction.PublicKeys
		duplicateStakePublicKeys := []string{}
		// list of stake public keys and stake transaction and reward receiver must have equal length

		tempStakePublicKey = beaconBestState.GetValidStakers(tempStakePublicKey)
		tempStakePublicKey = common.GetValidStaker(stakeShardPublicKeys, tempStakePublicKey)
		tempStakePublicKey = common.GetValidStaker(validStakePublicKeys, tempStakePublicKey)

		if len(tempStakePublicKey) > 0 {
			stakeShardPublicKeys = append(stakeShardPublicKeys, tempStakePublicKey...)
			for i, v := range stakeInstruction.PublicKeys {
				if common.IndexOfStr(v, tempStakePublicKey) > -1 {
					stakeShardTx = append(stakeShardTx, stakeInstruction.TxStakes[i])
					stakeShardRewardReceiver = append(stakeShardRewardReceiver, stakeInstruction.RewardReceivers[i])
					stakeShardAutoStaking = append(stakeShardAutoStaking, stakeInstruction.AutoStakingFlag[i])
				}
			}
		}

		if beaconBestState.beaconCommitteeEngine.Version() == committeestate.SLASHING_VERSION &&
			(len(stakeInstruction.PublicKeys) != len(tempStakePublicKey)) {
			duplicateStakePublicKeys = common.DifferentElementStrings(stakeInstruction.PublicKeys, tempStakePublicKey)
			if len(duplicateStakePublicKeys) > 0 {
				stakingTxs := []string{}
				autoStaking := []bool{}
				rewardReceivers := []string{}
				for i, v := range stakeInstruction.PublicKeys {
					if common.IndexOfStr(v, duplicateStakePublicKeys) > -1 {
						stakingTxs = append(stakingTxs, stakeInstruction.TxStakes[i])
						rewardReceivers = append(rewardReceivers, stakeInstruction.RewardReceivers[i])
						autoStaking = append(autoStaking, stakeInstruction.AutoStakingFlag[i])
					}
				}
				duplicateStakeInstruction := instruction.NewStakeInstructionWithValue(
					duplicateStakePublicKeys,
					stakeInstruction.Chain,
					stakingTxs,
					rewardReceivers,
					autoStaking,
				)
				duplicateKeyStakeInstruction.instructions = append(duplicateKeyStakeInstruction.instructions, duplicateStakeInstruction)
			}
		}
	}

	if len(stakeShardPublicKeys) > 0 {
		tempValidStakePublicKeys = append(tempValidStakePublicKeys, stakeShardPublicKeys...)
		tempStakeShardInstruction := instruction.NewStakeInstructionWithValue(
			stakeShardPublicKeys,
			instruction.SHARD_INST,
			stakeShardTx, stakeShardRewardReceiver,
			stakeShardAutoStaking,
		)
		stakeInstructions = append(stakeInstructions, tempStakeShardInstruction)
		validStakePublicKeys = append(validStakePublicKeys, stakeShardPublicKeys...)
	}

	newShardInstructions.stakeInstructions = stakeInstructions
	return newShardInstructions, duplicateKeyStakeInstruction
}

func (beaconBestState *BeaconBestState) processStopAutoStakeInstructionFromShardBlock(
	shardInstructions *shardInstruction, allCommitteeValidatorCandidate []string) *shardInstruction {

	stopAutoStakingPublicKeys := []string{}
	stopAutoStakeInstructions := []*instruction.StopAutoStakeInstruction{}

	for _, stopAutoStakeInstruction := range shardInstructions.stopAutoStakeInstructions {
		for _, tempStopAutoStakingPublicKey := range stopAutoStakeInstruction.CommitteePublicKeys {
			if common.IndexOfStr(tempStopAutoStakingPublicKey, allCommitteeValidatorCandidate) > -1 {
				stopAutoStakingPublicKeys = append(stopAutoStakingPublicKeys, tempStopAutoStakingPublicKey)
			}
		}
	}

	if len(stopAutoStakingPublicKeys) > 0 {
		tempStopAutoStakeInstruction := instruction.NewStopAutoStakeInstructionWithValue(stopAutoStakingPublicKeys)
		stopAutoStakeInstructions = append(stopAutoStakeInstructions, tempStopAutoStakeInstruction)
	}

	shardInstructions.stopAutoStakeInstructions = stopAutoStakeInstructions
	return shardInstructions
}

func (beaconBestState *BeaconBestState) processUnstakeInstructionFromShardBlock(
	shardInstructions *shardInstruction,
	allCommitteeValidatorCandidate []string,
	shardID byte,
	validUnstakePublicKeys map[string]bool) *shardInstruction {
	unstakePublicKeys := []string{}
	unstakeInstructions := []*instruction.UnstakeInstruction{}

	for _, unstakeInstruction := range shardInstructions.unstakeInstructions {
		for _, tempUnstakePublicKey := range unstakeInstruction.CommitteePublicKeys {
			// TODO: @hung check why only one transaction but it saied duplciate unstake instruction
			if _, ok := validUnstakePublicKeys[tempUnstakePublicKey]; ok {
				Logger.log.Errorf("SHARD %v | UNSTAKE duplicated unstake instruction %+v ", shardID, tempUnstakePublicKey)
				continue
			}
			if common.IndexOfStr(tempUnstakePublicKey, allCommitteeValidatorCandidate) > -1 {
				unstakePublicKeys = append(unstakePublicKeys, tempUnstakePublicKey)
			}
			validUnstakePublicKeys[tempUnstakePublicKey] = true
		}
	}
	if len(unstakePublicKeys) > 0 {
		tempUnstakeInstruction := instruction.NewUnstakeInstructionWithValue(unstakePublicKeys)
		tempUnstakeInstruction.SetCommitteePublicKeys(unstakePublicKeys)
		unstakeInstructions = append(unstakeInstructions, tempUnstakeInstruction)
	}

	shardInstructions.unstakeInstructions = unstakeInstructions
	return shardInstructions

}

func (shardInstruction *shardInstruction) compose() {
	stakeInstruction := &instruction.StakeInstruction{}
	unstakeInstruction := &instruction.UnstakeInstruction{}
	stopAutoStakeInstruction := &instruction.StopAutoStakeInstruction{}

	for _, v := range shardInstruction.stakeInstructions {
		if v.IsEmpty() {
			continue
		}
		stakeInstruction.PublicKeys = append(stakeInstruction.PublicKeys, v.PublicKeys...)
		stakeInstruction.PublicKeyStructs = append(stakeInstruction.PublicKeyStructs, v.PublicKeyStructs...)
		stakeInstruction.TxStakeHashes = append(stakeInstruction.TxStakeHashes, v.TxStakeHashes...)
		stakeInstruction.TxStakes = append(stakeInstruction.TxStakes, v.TxStakes...)
		stakeInstruction.RewardReceivers = append(stakeInstruction.RewardReceivers, v.RewardReceivers...)
		stakeInstruction.RewardReceiverStructs = append(stakeInstruction.RewardReceiverStructs, v.RewardReceiverStructs...)
		stakeInstruction.Chain = v.Chain
		stakeInstruction.AutoStakingFlag = append(stakeInstruction.AutoStakingFlag, v.AutoStakingFlag...)
	}

	for _, v := range shardInstruction.unstakeInstructions {
		if v.IsEmpty() {
			continue
		}
		unstakeInstruction.CommitteePublicKeys = append(unstakeInstruction.CommitteePublicKeys, v.CommitteePublicKeys...)
		unstakeInstruction.CommitteePublicKeysStruct = append(unstakeInstruction.CommitteePublicKeysStruct, v.CommitteePublicKeysStruct...)
	}

	for _, v := range shardInstruction.stopAutoStakeInstructions {
		if v.IsEmpty() {
			continue
		}
		stopAutoStakeInstruction.CommitteePublicKeys = append(stopAutoStakeInstruction.CommitteePublicKeys, v.CommitteePublicKeys...)
	}

	if !stakeInstruction.IsEmpty() {
		shardInstruction.stakeInstructions = []*instruction.StakeInstruction{}
		shardInstruction.stakeInstructions = append(shardInstruction.stakeInstructions, stakeInstruction)
	}
	if !unstakeInstruction.IsEmpty() {
		shardInstruction.unstakeInstructions = []*instruction.UnstakeInstruction{}
		shardInstruction.unstakeInstructions = append(shardInstruction.unstakeInstructions, unstakeInstruction)
	}
	if !stopAutoStakeInstruction.IsEmpty() {
		shardInstruction.stopAutoStakeInstructions = []*instruction.StopAutoStakeInstruction{}
		shardInstruction.stopAutoStakeInstructions = append(shardInstruction.stopAutoStakeInstructions, stopAutoStakeInstruction)
	}
}
