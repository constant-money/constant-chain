package blockchain

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain/btc"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
)

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
func (blockchain *BlockChain) NewBlockBeacon(curView *BeaconBestState, version int, proposer string, round int, startTime int64) (*BeaconBlock, error) {
	Logger.log.Infof("⛏ Creating Beacon Block %+v", curView.BeaconHeight+1)
	//============Init Variable============
	var err error
	var epoch uint64
	beaconBlock := NewBeaconBlock()
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

	if (beaconBestState.BeaconHeight+1)%blockchain.config.ChainParams.Epoch == 1 {
		featureStateDB := curView.GetBeaconFeatureStateDB()
		totalLockedCollateral, err := getTotalLockedCollateralInEpoch(featureStateDB)
		if err != nil {
			return nil, NewBlockChainError(GetTotalLockedCollateralError, err)
		}
		isSplitRewardForCustodian := totalLockedCollateral > 0
		percentCustodianRewards := portalParams.MaxPercentCustodianRewards
		if totalLockedCollateral < portalParams.MinLockCollateralAmountInEpoch {
			percentCustodianRewards = portalParams.MinPercentCustodianRewards
		}
		rewardByEpochInstruction, rewardForCustodianByEpoch, err = blockchain.buildRewardInstructionByEpoch(curView, beaconBlock.Header.Height, beaconBestState.Epoch, curView.GetBeaconRewardStateDB(), isSplitRewardForCustodian, percentCustodianRewards)
		if err != nil {
			return nil, NewBlockChainError(BuildRewardInstructionError, err)
		}
	}

	tempShardState, stakeInstructions, swapInstructions, bridgeInstructions, acceptedRewardInstructions, stopAutoStakingInstructions := blockchain.GetShardState(beaconBestState, rewardForCustodianByEpoch, portalParams)

	Logger.log.Infof("In NewBlockBeacon tempShardState: %+v", tempShardState)
	tempInstruction, err := beaconBestState.GenerateInstruction(
		beaconBlock.Header.Height, stakeInstructions, swapInstructions, stopAutoStakingInstructions,
		beaconBestState.CandidateShardWaitingForCurrentRandom, bridgeInstructions, acceptedRewardInstructions, blockchain.config.ChainParams.Epoch,
		blockchain.config.ChainParams.RandomTime, blockchain,
	)
	if err != nil {
		return nil, err
	}
	if len(rewardByEpochInstruction) != 0 {
		tempInstruction = append(tempInstruction, rewardByEpochInstruction...)
	}
	beaconBlock.Body.Instructions = tempInstruction
	beaconBlock.Body.ShardState = tempShardState
	if len(beaconBlock.Body.Instructions) != 0 {
		Logger.log.Info("Beacon Produce: Beacon Instruction", beaconBlock.Body.Instructions)
	}
	if len(bridgeInstructions) > 0 {
		BLogger.log.Infof("Producer instructions: %+v", tempInstruction)
	}
	//============End Build Body================
	//============Update Beacon Best State================
	// Process new block with beststate
	newBeaconBeststate, err := beaconBestState.updateBeaconBestState(beaconBlock, blockchain, newCommitteeChange())
	if err != nil {
		return nil, err
	}
	//============Build Header Hash=============
	// calculate hash
	// BeaconValidator root: beacon committee + beacon pending committee
	beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(newBeaconBeststate.BeaconCommittee)
	if err != nil {
		return nil, NewBlockChainError(UnExpectedError, err)
	}
	validatorArr := append([]string{}, beaconCommitteeStr...)

	beaconPendingValidatorStr, err := incognitokey.CommitteeKeyListToString(newBeaconBeststate.BeaconPendingValidator)
	if err != nil {
		return nil, NewBlockChainError(UnExpectedError, err)
	}
	validatorArr = append(validatorArr, beaconPendingValidatorStr...)
	tempBeaconCommitteeAndValidatorRoot, err := generateHashFromStringArray(validatorArr)
	if err != nil {
		return nil, NewBlockChainError(GenerateBeaconCommitteeAndValidatorRootError, err)
	}
	// BeaconCandidate root: beacon current candidate + beacon next candidate
	beaconCandidateArr := append(newBeaconBeststate.CandidateBeaconWaitingForCurrentRandom, newBeaconBeststate.CandidateBeaconWaitingForNextRandom...)

	beaconCandidateArrStr, err := incognitokey.CommitteeKeyListToString(beaconCandidateArr)
	if err != nil {
		return nil, NewBlockChainError(UnExpectedError, err)
	}
	tempBeaconCandidateRoot, err := generateHashFromStringArray(beaconCandidateArrStr)
	if err != nil {
		return nil, NewBlockChainError(GenerateBeaconCandidateRootError, err)
	}
	// Shard candidate root: shard current candidate + shard next candidate
	shardCandidateArr := append(newBeaconBeststate.CandidateShardWaitingForCurrentRandom, newBeaconBeststate.CandidateShardWaitingForNextRandom...)

	shardCandidateArrStr, err := incognitokey.CommitteeKeyListToString(shardCandidateArr)
	if err != nil {
		return nil, NewBlockChainError(UnExpectedError, err)
	}
	tempShardCandidateRoot, err := generateHashFromStringArray(shardCandidateArrStr)
	if err != nil {
		return nil, NewBlockChainError(GenerateShardCandidateRootError, err)
	}
	// Shard Validator root
	shardPendingValidator := make(map[byte][]string)
	for shardID, keys := range newBeaconBeststate.ShardPendingValidator {
		keysStr, err := incognitokey.CommitteeKeyListToString(keys)
		if err != nil {
			return nil, NewBlockChainError(UnExpectedError, err)
		}
		shardPendingValidator[shardID] = keysStr
	}
	shardCommittee := make(map[byte][]string)
	for shardID, keys := range newBeaconBeststate.ShardCommittee {
		keysStr, err := incognitokey.CommitteeKeyListToString(keys)
		if err != nil {
			return nil, NewBlockChainError(UnExpectedError, err)
		}
		shardCommittee[shardID] = keysStr
	}
	tempShardCommitteeAndValidatorRoot, err := generateHashFromMapByteString(shardPendingValidator, shardCommittee)
	if err != nil {
		return nil, NewBlockChainError(GenerateShardCommitteeAndValidatorRootError, err)
	}

	tempAutoStakingRoot, err := newBeaconBeststate.AutoStaking.GenerateHash()
	if err != nil {
		return nil, NewBlockChainError(AutoStakingRootHashError, err)
	}
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
	beaconBlock.Header.BeaconCommitteeAndValidatorRoot = tempBeaconCommitteeAndValidatorRoot
	beaconBlock.Header.BeaconCandidateRoot = tempBeaconCandidateRoot
	beaconBlock.Header.ShardCandidateRoot = tempShardCandidateRoot
	beaconBlock.Header.ShardCommitteeAndValidatorRoot = tempShardCommitteeAndValidatorRoot
	beaconBlock.Header.ShardStateHash = tempShardStateHash
	beaconBlock.Header.InstructionHash = tempInstructionHash
	beaconBlock.Header.AutoStakingRoot = tempAutoStakingRoot
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
func (blockchain *BlockChain) GetShardState(beaconBestState *BeaconBestState, rewardForCustodianByEpoch map[common.Hash]uint64, portalParams PortalParams) (map[byte][]ShardState, [][]string, map[byte][][]string, [][]string, [][]string, [][]string) {
	shardStates := make(map[byte][]ShardState)
	validStakeInstructions := [][]string{}
	validStakePublicKeys := []string{}
	validStopAutoStakingInstructions := [][]string{}
	validSwapInstructions := make(map[byte][][]string)
	//Get shard to beacon block from pool
	allShardBlocks := blockchain.GetShardBlockForBeaconProducer(beaconBestState.BestShardHeight)
	keys := []int{}
	for shardID, _ := range allShardBlocks {
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
		Logger.log.Infof("Beacon Producer Got %+v Shard Block from shard %+v: ", len(shardBlocks), shardID)
		for _, shardBlock := range shardBlocks {
			Logger.log.Info("Add shard block in shardstate", shardID, "height", shardBlock.GetHeight(), shardBlock.Hash().String())
			shardState, validStakeInstruction, tempValidStakePublicKeys, validSwapInstruction, bridgeInstruction, acceptedRewardInstruction, stopAutoStakingInstruction, statefulActions := blockchain.GetShardStateFromBlock(beaconBestState, beaconBestState.BeaconHeight+1, shardBlock, shardID, true, validStakePublicKeys)
			shardStates[shardID] = append(shardStates[shardID], shardState[shardID])
			validStakeInstructions = append(validStakeInstructions, validStakeInstruction...)
			validSwapInstructions[shardID] = append(validSwapInstructions[shardID], validSwapInstruction[shardID]...)
			bridgeInstructions = append(bridgeInstructions, bridgeInstruction...)
			acceptedRewardInstructions = append(acceptedRewardInstructions, acceptedRewardInstruction)
			validStopAutoStakingInstructions = append(validStopAutoStakingInstructions, stopAutoStakingInstruction...)
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
	return shardStates, validStakeInstructions, validSwapInstructions, bridgeInstructions, acceptedRewardInstructions, validStopAutoStakingInstructions
}

// GetShardStateFromBlock get state (information) from shard-to-beacon block
// state will be presented as instruction
//	Swap instruction format:
//	- ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "shard" "shardID"]
//	- ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "beacon"]
//	Stake instruction format:
//	- ["stake" "pubkey1,pubkey2,..." "shard" "txStakeHash1, txStakeHash2,..." "txStakeRewardReceiver1, txStakeRewardReceiver2,..." "flag1,flag2,..."]
//	- ["stake" "pubkey1,pubkey2,..." "beacon" "txStakeHash1, txStakeHash2,..." "txStakeRewardReceiver1, txStakeRewardReceiver2,..." "flag1,flag2,..."]
//	Stop Auto Staking instruction format:
//	- ["stopautostaking" "pubkey1,pubkey2,..."]
//	Return Params:
//	1. ShardState
//	2. Stake Instruction
//	3. Swap Instruction
//	4. Bridge Instruction
//	5. Accepted BlockReward Instruction
//	6. StopAutoStakingInstruction
func (blockchain *BlockChain) GetShardStateFromBlock(curView *BeaconBestState, newBeaconHeight uint64, shardBlock *ShardBlock, shardID byte, isProducer bool, validStakePublicKeys []string) (map[byte]ShardState, [][]string, []string, map[byte][][]string, [][]string, []string, [][]string, [][]string) {
	//Variable Declaration
	shardStates := make(map[byte]ShardState)
	stakeInstructions := [][]string{}
	swapInstructions := make(map[byte][][]string)
	stopAutoStakingInstructions := [][]string{}
	stopAutoStakingInstructionsFromBlock := [][]string{}
	stakeInstructionFromShardBlock := [][]string{}
	swapInstructionFromShardBlock := [][]string{}
	bridgeInstructions := [][]string{}
	stakeBeaconPublicKeys := []string{}
	stakeShardPublicKeys := []string{}
	stakeBeaconTx := []string{}
	stakeShardTx := []string{}
	stakeShardRewardReceiver := []string{}
	stakeBeaconRewardReceiver := []string{}
	stakeShardAutoStaking := []string{}
	stakeBeaconAutoStaking := []string{}
	stopAutoStakingPublicKeys := []string{}
	tempValidStakePublicKeys := []string{}
	acceptedBlockRewardInfo := metadata.NewAcceptedBlockRewardInfo(shardID, shardBlock.Header.TotalTxsFee, shardBlock.Header.Height)
	acceptedRewardInstructions, err := acceptedBlockRewardInfo.GetStringFormat()
	if err != nil {
		// if err then ignore accepted reward instruction
		acceptedRewardInstructions = []string{}
	}
	//Get Shard State from Block
	shardState := ShardState{}
	shardState.CrossShard = make([]byte, len(shardBlock.Header.CrossShardBitMap))
	copy(shardState.CrossShard, shardBlock.Header.CrossShardBitMap)
	shardState.Hash = shardBlock.Header.Hash()
	shardState.Height = shardBlock.Header.Height
	shardStates[shardID] = shardState
	instructions, err := CreateShardInstructionsFromTransactionAndInstruction(shardBlock.Body.Transactions, blockchain, shardBlock.Header.ShardID)
	instructions = append(instructions, shardBlock.Body.Instructions...)

	// extract instructions
	for _, instruction := range instructions {
		if len(instruction) > 0 {
			if instruction[0] == StakeAction {
				stakeInstructionFromShardBlock = append(stakeInstructionFromShardBlock, instruction)
			}
			if instruction[0] == SwapAction {
				//- ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "shard" "shardID"]
				//- ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "beacon"]
				// validate swap instruction
				// only allow shard to swap committee for it self
				if instruction[3] == "beacon" {
					continue
				}
				if instruction[3] == "shard" && len(instruction) != 6 && instruction[4] != strconv.Itoa(int(shardID)) {
					continue
				}
				swapInstructions[shardID] = append(swapInstructions[shardID], instruction)
			}
			if instruction[0] == StopAutoStake {
				if len(instruction) != 2 {
					continue
				}
				stopAutoStakingInstructionsFromBlock = append(stopAutoStakingInstructionsFromBlock, instruction)
			}
		}
	}
	if len(stakeInstructionFromShardBlock) != 0 {
		Logger.log.Info("Beacon Producer/ Process Stakers List ", stakeInstructionFromShardBlock)
	}
	if len(swapInstructions[shardID]) != 0 {
		Logger.log.Info("Beacon Producer/ Process Stakers List ", swapInstructionFromShardBlock)
	}
	// Process Stake Instruction form Shard Block
	// Validate stake instruction => extract only valid stake instruction
	for _, stakeInstruction := range stakeInstructionFromShardBlock {
		if len(stakeInstruction) != 6 {
			continue
		}
		var tempStakePublicKey []string
		newBeaconCandidate, newShardCandidate := getStakeValidatorArrayString(stakeInstruction)
		assignShard := true
		if !reflect.DeepEqual(newBeaconCandidate, []string{}) {
			tempStakePublicKey = make([]string, len(newBeaconCandidate))
			copy(tempStakePublicKey, newBeaconCandidate[:])
			assignShard = false
		} else {
			tempStakePublicKey = make([]string, len(newShardCandidate))
			copy(tempStakePublicKey, newShardCandidate[:])
		}
		// list of stake public keys and stake transaction and reward receiver must have equal length
		if len(tempStakePublicKey) != len(strings.Split(stakeInstruction[3], ",")) && len(strings.Split(stakeInstruction[3], ",")) != len(strings.Split(stakeInstruction[4], ",")) && len(strings.Split(stakeInstruction[4], ",")) != len(strings.Split(stakeInstruction[5], ",")) {
			continue
		}
		tempStakePublicKey = curView.GetValidStakers(tempStakePublicKey)
		tempStakePublicKey = common.GetValidStaker(stakeShardPublicKeys, tempStakePublicKey)
		tempStakePublicKey = common.GetValidStaker(stakeBeaconPublicKeys, tempStakePublicKey)
		tempStakePublicKey = common.GetValidStaker(validStakePublicKeys, tempStakePublicKey)
		if len(tempStakePublicKey) > 0 {
			if assignShard {
				stakeShardPublicKeys = append(stakeShardPublicKeys, tempStakePublicKey...)
				for i, v := range strings.Split(stakeInstruction[1], ",") {
					if common.IndexOfStr(v, tempStakePublicKey) > -1 {
						stakeShardTx = append(stakeShardTx, strings.Split(stakeInstruction[3], ",")[i])
						stakeShardRewardReceiver = append(stakeShardRewardReceiver, strings.Split(stakeInstruction[4], ",")[i])
						stakeShardAutoStaking = append(stakeShardAutoStaking, strings.Split(stakeInstruction[5], ",")[i])
					}
				}
			} else {
				stakeBeaconPublicKeys = append(stakeBeaconPublicKeys, tempStakePublicKey...)
				for i, v := range strings.Split(stakeInstruction[1], ",") {
					if common.IndexOfStr(v, tempStakePublicKey) > -1 {
						stakeBeaconTx = append(stakeBeaconTx, strings.Split(stakeInstruction[3], ",")[i])
						stakeBeaconRewardReceiver = append(stakeBeaconRewardReceiver, strings.Split(stakeInstruction[4], ",")[i])
						stakeBeaconAutoStaking = append(stakeBeaconAutoStaking, strings.Split(stakeInstruction[5], ",")[i])
					}
				}
			}
		}
	}
	if len(stakeShardPublicKeys) > 0 {
		tempValidStakePublicKeys = append(tempValidStakePublicKeys, stakeShardPublicKeys...)
		stakeInstructions = append(stakeInstructions, []string{StakeAction, strings.Join(stakeShardPublicKeys, ","), "shard", strings.Join(stakeShardTx, ","), strings.Join(stakeShardRewardReceiver, ","), strings.Join(stakeShardAutoStaking, ",")})
	}
	if len(stakeBeaconPublicKeys) > 0 {
		tempValidStakePublicKeys = append(tempValidStakePublicKeys, stakeBeaconPublicKeys...)
		stakeInstructions = append(stakeInstructions, []string{StakeAction, strings.Join(stakeBeaconPublicKeys, ","), "beacon", strings.Join(stakeBeaconTx, ","), strings.Join(stakeBeaconRewardReceiver, ","), strings.Join(stakeBeaconAutoStaking, ",")})
	}
	for _, instruction := range stopAutoStakingInstructionsFromBlock {
		allCommitteeValidatorCandidate := []string{}
		// avoid dead lock
		// if producer new block then lock beststate
		// if isProducer {
		// 	allCommitteeValidatorCandidate = curView.getAllCommitteeValidatorCandidateFlattenList()
		// } else {
		// if process block then do not lock beststate
		allCommitteeValidatorCandidate = curView.getAllCommitteeValidatorCandidateFlattenList()
		// }
		tempStopAutoStakingPublicKeys := strings.Split(instruction[1], ",")
		for _, tempStopAutoStakingPublicKey := range tempStopAutoStakingPublicKeys {
			if common.IndexOfStr(tempStopAutoStakingPublicKey, allCommitteeValidatorCandidate) > -1 {
				stopAutoStakingPublicKeys = append(stopAutoStakingPublicKeys, tempStopAutoStakingPublicKey)
			}
		}
	}
	if len(stopAutoStakingPublicKeys) > 0 {
		stopAutoStakingInstructions = append(stopAutoStakingInstructions, []string{StopAutoStake, strings.Join(stopAutoStakingPublicKeys, ",")})
	}
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
		BLogger.log.Infof("Beacon block %d found bridge swap confirm inst in shard block %d: %s", newBeaconHeight, shardBlock.Header.Height, confirmInsts)
	}
	bridgeInstructions = append(bridgeInstructions, bridgeInstructionForBlock...)

	// Collect stateful actions
	statefulActions := blockchain.collectStatefulActions(instructions)
	//Logger.log.Infof("Becon Produce: Got Shard Block %+v Shard %+v \n", shardBlock.Header.Height, shardID)
	return shardStates, stakeInstructions, tempValidStakePublicKeys, swapInstructions, bridgeInstructions, acceptedRewardInstructions, stopAutoStakingInstructions, statefulActions
}

//  GenerateInstruction generate instruction for new beacon block
//  Instruction Type:
//  - swap instruction format
//    + ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "shard" "shardID"]
//    + ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,..." "beacon"]
//  - random instruction format
//    + ["random" "{nonce}" "{blockheight}" "{timestamp}" "{bitcoinTimestamp}"]
//  - stake instruction format
//    + ["stake", "pubkey1,pubkey2,..." "shard" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,...", "flag1,flag2..."]
//    + ["stake", "pubkey1,pubkey2,..." "beacon" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,...", "flag1,flag2..."]
//  - assign instruction fomart
//    + ["assign" "shardCandidate1,shardCandidate2,..." "shard" "{shardID}"]
func (beaconBestState *BeaconBestState) GenerateInstruction(
	newBeaconHeight uint64,
	stakeInstructions [][]string,
	swapInstructions map[byte][][]string,
	stopAutoStakingInstructions [][]string,
	shardCandidates []incognitokey.CommitteePublicKey,
	bridgeInstructions [][]string,
	acceptedRewardInstructions [][]string,
	chainParamEpoch uint64,
	randomTime uint64,
	blockchain *BlockChain,
) ([][]string, error) {
	instructions := [][]string{}
	instructions = append(instructions, bridgeInstructions...)
	instructions = append(instructions, acceptedRewardInstructions...)
	//=======Swap
	// Shard Swap: both abnormal or normal swap
	var keys []int
	for k := range swapInstructions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		instructions = append(instructions, swapInstructions[byte(shardID)]...)
	}
	// Beacon normal swap
	if newBeaconHeight%chainParamEpoch == 0 {
		swapBeaconInstructions := []string{}
		beaconPendingValidatorStr, err := incognitokey.CommitteeKeyListToString(beaconBestState.BeaconPendingValidator)
		if err != nil {
			return [][]string{}, err
		}
		beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(beaconBestState.BeaconCommittee)
		if err != nil {
			return [][]string{}, err
		}
		//beaconSlashRootHash, err := blockchain.GetBeaconSlashRootHash(beaconBestState, newBeaconHeight-1)
		//if err != nil {
		//	return [][]string{}, err
		//}
		//beaconSlashStateDB, err := statedb.NewWithPrefixTrie(beaconSlashRootHash, statedb.NewDatabaseAccessWarper(blockchain.GetBeaconChainDatabase()))
		//producersBlackList, err := blockchain.getUpdatedProducersBlackList(beaconSlashStateDB, true, -1, beaconCommitteeStr, newBeaconHeight-1)
		//if err != nil {
		//	Logger.log.Error(err)
		//}
		producersBlackList := make(map[string]uint8)
		badProducersWithPunishment := blockchain.buildBadProducersWithPunishment(true, -1, beaconCommitteeStr)
		badProducersWithPunishmentBytes, err := json.Marshal(badProducersWithPunishment)
		if err != nil {
			Logger.log.Error(err)
		}
		if common.IndexOfUint64(newBeaconHeight/chainParamEpoch, blockchain.config.ChainParams.EpochBreakPointSwapNewKey) > -1 {
			epoch := newBeaconHeight / chainParamEpoch
			swapBeaconInstructions, _, beaconCommittee := CreateBeaconSwapActionForKeyListV2(blockchain.config.GenesisParams, beaconPendingValidatorStr, beaconCommitteeStr, beaconBestState.MinBeaconCommitteeSize, epoch)
			instructions = append(instructions, swapBeaconInstructions)
			beaconRootInst, _ := buildBeaconSwapConfirmInstruction(beaconCommittee, newBeaconHeight)
			instructions = append(instructions, beaconRootInst)
		} else {
			_, currentValidators, swappedValidator, beaconNextCommittee, err := SwapValidator(beaconPendingValidatorStr, beaconCommitteeStr, beaconBestState.MaxBeaconCommitteeSize, beaconBestState.MinBeaconCommitteeSize, blockchain.config.ChainParams.Offset, producersBlackList, blockchain.config.ChainParams.SwapOffset)
			if len(swappedValidator) > 0 || len(beaconNextCommittee) > 0 && err == nil {
				swapBeaconInstructions = append(swapBeaconInstructions, "swap")
				swapBeaconInstructions = append(swapBeaconInstructions, strings.Join(beaconNextCommittee, ","))
				swapBeaconInstructions = append(swapBeaconInstructions, strings.Join(swappedValidator, ","))
				swapBeaconInstructions = append(swapBeaconInstructions, "beacon")
				swapBeaconInstructions = append(swapBeaconInstructions, string(badProducersWithPunishmentBytes))
				instructions = append(instructions, swapBeaconInstructions)
				// Generate instruction storing validators pubkey and send to bridge
				beaconRootInst, _ := buildBeaconSwapConfirmInstruction(currentValidators, newBeaconHeight)
				instructions = append(instructions, beaconRootInst)
			}
		}
	}
	// Stake
	instructions = append(instructions, stakeInstructions...)
	// Stop Auto Staking
	instructions = append(instructions, stopAutoStakingInstructions...)
	// Random number for Assign Instruction
	if newBeaconHeight%chainParamEpoch > randomTime && !beaconBestState.IsGetRandomNumber {
		var err error
		var chainTimeStamp int64
		if !TestRandom {
			if newBeaconHeight%chainParamEpoch == chainParamEpoch-1 {
				startTime := time.Now()
				for {
					Logger.log.Criticalf("Block %+v, Enter final block of epoch but still no random number", newBeaconHeight)
					chainTimeStamp, err = blockchain.config.RandomClient.GetCurrentChainTimeStamp()
					if err != nil {
						Logger.log.Error(err)
					} else {
						if chainTimeStamp < beaconBestState.CurrentRandomTimeStamp {
							Logger.log.Infof("Final Block %+v in Epoch but still haven't found new random number", newBeaconHeight)
						} else {
							break
						}
					}
					if time.Since(startTime).Seconds() > beaconBestState.BlockMaxCreateTime.Seconds() {
						return [][]string{}, NewBlockChainError(GenerateInstructionError, fmt.Errorf("Get Current Chain Timestamp for New Block Height %+v Timeout", newBeaconHeight))
					}
					time.Sleep(100 * time.Millisecond)
				}
			} else {
				Logger.log.Criticalf("Block %+v, finding random number", newBeaconHeight)
				chainTimeStamp, err = blockchain.config.RandomClient.GetCurrentChainTimeStamp()
				if err != nil {
					Logger.log.Error(err)
				}
			}
		} else {
			chainTimeStamp = beaconBestState.CurrentRandomTimeStamp + 1
		}
		//==================================
		if err == nil && chainTimeStamp > beaconBestState.CurrentRandomTimeStamp {
			numberOfPendingValidator := make(map[byte]int)
			for i := 0; i < beaconBestState.ActiveShards; i++ {
				if pendingValidators, ok := beaconBestState.ShardPendingValidator[byte(i)]; ok {
					numberOfPendingValidator[byte(i)] = len(pendingValidators)
				} else {
					numberOfPendingValidator[byte(i)] = 0
				}
			}
			randomInstruction, rand, err := beaconBestState.generateRandomInstruction(beaconBestState.CurrentRandomTimeStamp, blockchain.config.RandomClient)
			if err != nil {
				return [][]string{}, err
			}
			instructions = append(instructions, randomInstruction)
			Logger.log.Infof("Beacon Producer found Random Instruction at Block Height %+v, %+v", randomInstruction, newBeaconHeight)
			shardCandidatesStr, err := incognitokey.CommitteeKeyListToString(shardCandidates)
			if err != nil {
				panic(err)
			}
			_, assignedCandidates := assignShardCandidate(shardCandidatesStr, numberOfPendingValidator, rand, blockchain.config.ChainParams.AssignOffset, beaconBestState.ActiveShards)
			var keys []int
			for k := range assignedCandidates {
				keys = append(keys, int(k))
			}
			sort.Ints(keys)
			for _, key := range keys {
				shardID := byte(key)
				candidates := assignedCandidates[shardID]
				Logger.log.Infof("Assign Candidate at Shard %+v: %+v", shardID, candidates)
				shardAssingInstruction := []string{AssignAction}
				shardAssingInstruction = append(shardAssingInstruction, strings.Join(candidates, ","))
				shardAssingInstruction = append(shardAssingInstruction, "shard")
				shardAssingInstruction = append(shardAssingInstruction, fmt.Sprintf("%v", shardID))
				instructions = append(instructions, shardAssingInstruction)
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
			strs           []string
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
		strs = append(strs, "random")
		strs = append(strs, strconv.Itoa(int(nonce)))
		strs = append(strs, strconv.Itoa(blockHeight))
		strs = append(strs, strconv.Itoa(int(timestamp)))
		strs = append(strs, strconv.Itoa(int(chainTimestamp)))
		return strs, int64(nonce), nil
	} else {
		ran := rand.New(rand.NewSource(timestamp))
		randInt := ran.Int()
		var strs []string
		reses := []string{strconv.Itoa(randInt), strconv.Itoa(int(timestamp)), strconv.Itoa(int(timestamp) + 1)}
		strs = append(strs, RandomAction)
		strs = append(strs, reses...)
		strs = append(strs, strconv.Itoa(int(timestamp)))
		return strs, int64(randInt), nil
	}
}
