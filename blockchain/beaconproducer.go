package blockchain

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"

	"github.com/incognitochain/incognito-chain/incognitokey"

	"github.com/incognitochain/incognito-chain/blockchain/committeestate"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
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

func newShardInstruction() *shardInstruction {
	return &shardInstruction{
		swapInstructions: make(map[byte][]*instruction.SwapInstruction),
	}
}

func (shardInstruction *shardInstruction) add(newShardInstruction *shardInstruction) {
	shardInstruction.stakeInstructions = append(shardInstruction.stakeInstructions, newShardInstruction.stakeInstructions...)
	shardInstruction.unstakeInstructions = append(shardInstruction.unstakeInstructions, newShardInstruction.unstakeInstructions...)
	shardInstruction.stopAutoStakeInstructions = append(shardInstruction.stopAutoStakeInstructions, newShardInstruction.stopAutoStakeInstructions...)
	for shardID, swapInstructions := range newShardInstruction.swapInstructions {
		shardInstruction.swapInstructions[shardID] = append(shardInstruction.swapInstructions[shardID], swapInstructions...)
	}
}

// NewBlockBeacon create new beacon block
func (blockchain *BlockChain) NewBlockBeacon(curView *BeaconBestState, version int, proposer string, round int, startTime int64) (*types.BeaconBlock, error) {
	Logger.log.Infof("⛏ Creating Beacon Block %+v", curView.BeaconHeight+1)
	var err error
	var epoch uint64
	newBeaconBlock := types.NewBeaconBlock()
	copiedCurView := NewBeaconBestState()

	err = copiedCurView.cloneBeaconBestStateFrom(curView)
	if err != nil {
		return nil, err
	}

	if (copiedCurView.BeaconHeight+1)%blockchain.config.ChainParams.Epoch == 1 {
		epoch = copiedCurView.Epoch + 1
	} else {
		epoch = copiedCurView.Epoch
	}

	newBeaconBlock.Header = types.NewBeaconHeader(
		version,
		copiedCurView.BeaconHeight+1,
		epoch,
		round,
		startTime,
		copiedCurView.BestBlockHash,
		copiedCurView.ConsensusAlgorithm,
		proposer,
		proposer,
	)

	BLogger.log.Infof("Producing block: %d (epoch %d)", newBeaconBlock.Header.Height, newBeaconBlock.Header.Epoch)
	//=====END Build Header Essential Data=====
	portalParams := blockchain.GetPortalParams(newBeaconBlock.GetHeight())
	allShardBlocks := blockchain.GetShardBlockForBeaconProducer(copiedCurView.BestShardHeight)

	instructions, shardStates, err := blockchain.GenerateBeaconBlockBody(
		newBeaconBlock,
		copiedCurView,
		portalParams,
		allShardBlocks,
	)
	if err != nil {
		return nil, NewBlockChainError(GenerateInstructionError, err)
	}

	newBeaconBlock.Body = types.NewBeaconBody(shardStates, instructions)

	// Process new block with new view
	_, hashes, _, incurredInstructions, err := copiedCurView.updateBeaconBestState(newBeaconBlock, blockchain)
	if err != nil {
		return nil, err
	}
	copiedCurView.beaconCommitteeEngine.AbortUncommittedBeaconState()

	instructions = append(instructions, incurredInstructions...)
	newBeaconBlock.Body.SetInstructions(instructions)
	if len(newBeaconBlock.Body.Instructions) != 0 {
		Logger.log.Info("Beacon Produce: Beacon Instruction", newBeaconBlock.Body.Instructions)
	}

	// calculate hash
	tempInstructionArr := []string{}
	for _, strs := range instructions {
		tempInstructionArr = append(tempInstructionArr, strs...)
	}
	instructionHash, err := generateHashFromStringArray(tempInstructionArr)
	if err != nil {
		return nil, NewBlockChainError(GenerateInstructionHashError, err)
	}
	shardStatesHash, err := generateHashFromShardState(shardStates)
	if err != nil {
		return nil, NewBlockChainError(GenerateShardStateError, err)
	}
	// Instruction merkle root
	flattenInsts, err := FlattenAndConvertStringInst(instructions)
	if err != nil {
		return nil, NewBlockChainError(FlattenAndConvertStringInstError, err)
	}
	// add hash to header
	newBeaconBlock.Header.AddBeaconHeaderHash(
		instructionHash,
		shardStatesHash,
		GetKeccak256MerkleRoot(flattenInsts),
		hashes.BeaconCommitteeAndValidatorHash,
		hashes.BeaconCandidateHash,
		hashes.ShardCandidateHash,
		hashes.ShardCommitteeAndValidatorHash,
		hashes.AutoStakeHash,
	)

	return newBeaconBlock, nil
}

// GenerateBeaconBlockBody generate beacon instructions and shard states
func (blockchain *BlockChain) GenerateBeaconBlockBody(
	newBeaconBlock *types.BeaconBlock,
	curView *BeaconBestState,
	portalParams PortalParams,
	allShardBlocks map[byte][]*types.ShardBlock,
) ([][]string, map[byte][]types.ShardState, error) {
	bridgeInstructions := [][]string{}
	acceptedRewardInstructions := [][]string{}
	statefulActionsByShardID := map[byte][][]string{}
	shardStates := make(map[byte][]types.ShardState)
	shardInstruction := newShardInstruction()
	duplicateKeyStakeInstructions := &duplicateKeyStakeInstruction{}
	validStakePublicKeys := []string{}
	validUnstakePublicKeys := make(map[string]bool)
	rewardForCustodianByEpoch := map[common.Hash]uint64{}
	rewardByEpochInstruction := [][]string{}

	if curView.BeaconHeight%blockchain.config.ChainParams.Epoch == 0 {
		featureStateDB := curView.GetBeaconFeatureStateDB()
		totalLockedCollateral, err := getTotalLockedCollateralInEpoch(featureStateDB)
		if err != nil {
			return nil, nil, NewBlockChainError(GetTotalLockedCollateralError, err)
		}

		isSplitRewardForCustodian := totalLockedCollateral > 0
		percentCustodianRewards := portalParams.MaxPercentCustodianRewards
		if totalLockedCollateral < portalParams.MinLockCollateralAmountInEpoch {
			percentCustodianRewards = portalParams.MinPercentCustodianRewards
		}
		rewardByEpochInstruction, rewardForCustodianByEpoch, err = blockchain.buildRewardInstructionByEpoch(
			curView,
			newBeaconBlock.Header.Height,
			curView.Epoch,
			isSplitRewardForCustodian,
			percentCustodianRewards,
		)
		if err != nil {
			return nil, nil, NewBlockChainError(BuildRewardInstructionError, err)
		}
	}

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
	for _, v := range keys {
		shardID := byte(v)
		shardBlocks := allShardBlocks[shardID]
		for _, shardBlock := range shardBlocks {
			shardState, newShardInstruction, newDuplicateKeyStakeInstruction,
				bridgeInstruction, acceptedRewardInstruction, statefulActions := blockchain.GetShardStateFromBlock(
				curView, curView.BeaconHeight+1, shardBlock, shardID, validUnstakePublicKeys, validStakePublicKeys)
			shardStates[shardID] = append(shardStates[shardID], shardState[shardID])
			duplicateKeyStakeInstructions.add(newDuplicateKeyStakeInstruction)
			shardInstruction.add(newShardInstruction)
			bridgeInstructions = append(bridgeInstructions, bridgeInstruction...)
			acceptedRewardInstructions = append(acceptedRewardInstructions, acceptedRewardInstruction)
			// group stateful actions by shardID
			for _, v := range newShardInstruction.stakeInstructions {
				validStakePublicKeys = append(validStakePublicKeys, v.PublicKeys...)
			}
			_, found := statefulActionsByShardID[shardID]
			if !found {
				statefulActionsByShardID[shardID] = statefulActions
			} else {
				statefulActionsByShardID[shardID] = append(statefulActionsByShardID[shardID], statefulActions...)
			}
		}
	}

	// build stateful instructions
	statefulInsts := blockchain.buildStatefulInstructions(
		curView.featureStateDB,
		statefulActionsByShardID,
		newBeaconBlock.Header.Height,
		rewardForCustodianByEpoch,
		portalParams,
	)
	bridgeInstructions = append(bridgeInstructions, statefulInsts...)

	shardInstruction.compose()

	instructions, err := curView.GenerateInstruction(
		newBeaconBlock.Header.Height, shardInstruction, duplicateKeyStakeInstructions,
		bridgeInstructions, acceptedRewardInstructions, blockchain.config.ChainParams.Epoch,
		blockchain.config.ChainParams.RandomTime, blockchain,
		shardStates,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(bridgeInstructions) > 0 {
		BLogger.log.Infof("Producer instructions: %+v", instructions)
	}

	if len(rewardByEpochInstruction) != 0 {
		instructions = append(instructions, rewardByEpochInstruction...)
	}

	return instructions, shardStates, nil
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

	instructions, err := CreateShardInstructionsFromTransactionAndInstruction(
		shardBlock.Body.Transactions, blockchain, shardID)
	instructions = append(instructions, shardBlock.Body.Instructions...)

	shardInstruction := curView.preProcessInstructionsFromShardBlock(instructions, shardID)
	shardInstruction, duplicateKeyStakeInstruction = curView.
		processStakeInstructionFromShardBlock(shardInstruction, validStakePublicKeys)

	allCommitteeValidatorCandidate := []string{}
	if len(shardInstruction.stopAutoStakeInstructions) != 0 || len(shardInstruction.unstakeInstructions) != 0 {
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
func (curView *BeaconBestState) GenerateInstruction(
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

	// Stake
	for _, stakeInstruction := range shardInstruction.stakeInstructions {
		instructions = append(instructions, stakeInstruction.ToString())
	}

	// Duplicate Staking Instruction
	for _, stakeInstruction := range duplicateKeyStakeInstruction.instructions {
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

	// Random number for Assign Instruction
	if newBeaconHeight%chainParamEpoch > randomTime && !curView.IsGetRandomNumber {
		randomInstruction, randomNumber := curView.generateRandomInstruction()
		instructions = append(instructions, randomInstruction)
		Logger.log.Infof("Beacon Producer found Random Instruction at Block Height %+v, %+v", randomInstruction, newBeaconHeight)
		assignInstructions, _, _ := curView.beaconCommitteeEngine.GenerateAssignInstruction(
			randomNumber,
			blockchain.config.ChainParams.AssignOffset,
			curView.ActiveShards,
		)
		for _, assignInstruction := range assignInstructions {
			instructions = append(instructions, assignInstruction.ToString())
		}
		Logger.log.Info("assignInstructions:", assignInstructions)
	}

	// Generate swap shard instruction at block height %chainParamEpoch == 0
	if curView.CommitteeEngineVersion() == committeestate.SELF_SWAP_SHARD_VERSION {
		if newBeaconHeight%chainParamEpoch == 0 {
			BeaconCommittee := curView.GetBeaconCommittee()
			beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(BeaconCommittee)
			if err != nil {
				Logger.log.Error(err)
			}
			if common.IndexOfUint64(newBeaconHeight/chainParamEpoch, blockchain.config.ChainParams.EpochBreakPointSwapNewKey) > -1 {
				epoch := newBeaconHeight / chainParamEpoch
				swapBeaconInstructions, beaconCommittee := CreateBeaconSwapActionForKeyListV2(blockchain.config.GenesisParams, beaconCommitteeStr, curView.MinBeaconCommitteeSize, epoch)
				instructions = append(instructions, swapBeaconInstructions)
				beaconRootInst, _ := buildBeaconSwapConfirmInstruction(beaconCommittee, newBeaconHeight)
				instructions = append(instructions, beaconRootInst)
			}
		}
	} else if curView.CommitteeEngineVersion() == committeestate.SLASHING_VERSION {
		if newBeaconHeight%chainParamEpoch == 1 {
			// Generate request shard swap instruction, only available after upgrade to BeaconCommitteeEngineV2
			env := curView.NewBeaconCommitteeStateEnvironment(blockchain.config.ChainParams)
			env.LatestShardsState = shardsState
			swapShardInstructions, err := curView.beaconCommitteeEngine.GenerateAllSwapShardInstructions(env)
			if err != nil {
				return [][]string{}, err
			}
			for _, swapShardInstruction := range swapShardInstructions {
				if !swapShardInstruction.IsEmpty() {
					instructions = append(instructions, swapShardInstruction.ToString())
				}

			}
		}
	}

	// Stop Auto Stake
	for _, stopAutoStakeInstruction := range shardInstruction.stopAutoStakeInstructions {
		instructions = append(instructions, stopAutoStakeInstruction.ToString())
	}

	// Unstake
	for _, unstakeInstruction := range shardInstruction.unstakeInstructions {
		instructions = append(instructions, unstakeInstruction.ToString())
	}

	return instructions, nil
}

// ["random" "{nonce}" "{blockheight}" "{timestamp}" "{bitcoinTimestamp}"]
func (curView *BeaconBestState) generateRandomInstruction() ([]string, int64) {
	res := []byte{}
	bestBeaconBlockHash := curView.BestBlockHash
	res = append(res, bestBeaconBlockHash.Bytes()...)
	for i := 0; i < curView.ActiveShards; i++ {
		shardID := byte(i)
		bestShardBlockHash := curView.BestShardHash[shardID]
		res = append(res, bestShardBlockHash.Bytes()...)
	}

	bigInt := new(big.Int)
	bigInt = bigInt.SetBytes(res)
	randomNumber := int64(bigInt.Uint64())
	randomInstruction := []string{instruction.RANDOM_ACTION, strconv.FormatInt(randomNumber, 10), "", "", ""}

	return randomInstruction, randomNumber
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

func (beaconBestState *BeaconBestState) preProcessInstructionsFromShardBlock(instructions [][]string, shardID byte) *shardInstruction {
	shardInstruction := newShardInstruction()
	// extract instructions

	waitingValidatorsList, err := incognitokey.CommitteeKeyListToString(beaconBestState.beaconCommitteeEngine.GetCandidateShardWaitingForNextRandom())
	if err != nil {
		return shardInstruction
	}

	for _, inst := range instructions {
		if len(inst) > 0 {
			if inst[0] == instruction.STAKE_ACTION {
				if err := instruction.ValidateStakeInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Stake Instruction Error %+v", err)
					continue
				}
				tempStakeInstruction := instruction.ImportStakeInstructionFromString(inst)
				shardInstruction.stakeInstructions = append(shardInstruction.stakeInstructions, tempStakeInstruction)
			}
			if inst[0] == instruction.SWAP_ACTION {
				// validate swap instruction
				// only allow shard to swap committee for it self
				if err := instruction.ValidateSwapInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Swap Instruction Error %+v", err)
					continue
				}
				tempSwapInstruction := instruction.ImportSwapInstructionFromString(inst)
				shardInstruction.swapInstructions[shardID] = append(shardInstruction.swapInstructions[shardID], tempSwapInstruction)
			}
			if inst[0] == instruction.STOP_AUTO_STAKE_ACTION {
				if err := instruction.ValidateStopAutoStakeInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Stop Auto Stake Instruction Error %+v", err)
					continue
				}
				tempStopAutoStakeInstruction := instruction.ImportStopAutoStakeInstructionFromString(inst)
				for i := 0; i < len(tempStopAutoStakeInstruction.CommitteePublicKeys); i++ {
					v := tempStopAutoStakeInstruction.CommitteePublicKeys[i]
					check, ok := beaconBestState.GetAutoStakingList()[v]
					if !ok {
						Logger.log.Errorf("Committee %s is not found or has already been unstaked:", v)
					}
					if !ok || !check {
						tempStopAutoStakeInstruction.DeleteSingleElement(i)
						i--
					}
				}
				if len(tempStopAutoStakeInstruction.CommitteePublicKeys) != 0 {
					shardInstruction.stopAutoStakeInstructions = append(shardInstruction.stopAutoStakeInstructions, tempStopAutoStakeInstruction)
				}
			}
			if inst[0] == instruction.UNSTAKE_ACTION {
				if err := instruction.ValidateUnstakeInstructionSanity(inst); err != nil {
					Logger.log.Errorf("SKIP Stop Auto Stake Instruction Error %+v", err)
					continue
				}
				tempUnstakeInstruction := instruction.ImportUnstakeInstructionFromString(inst)
				for i := 0; i < len(tempUnstakeInstruction.CommitteePublicKeys); i++ {
					v := tempUnstakeInstruction.CommitteePublicKeys[i]
					index := common.IndexOfStr(v, waitingValidatorsList)
					if index == -1 {
						check, ok := beaconBestState.GetAutoStakingList()[v]
						if !ok {
							Logger.log.Errorf("[unstaking] Committee %s is not found or has already been unstaked:", v)
						}
						if !ok || !check {
							tempUnstakeInstruction.DeleteSingleElement(i)
							i--
						}
					}
				}
				if len(tempUnstakeInstruction.CommitteePublicKeys) != 0 {
					shardInstruction.unstakeInstructions = append(shardInstruction.unstakeInstructions, tempUnstakeInstruction)
				}
			}
		}
	}

	if len(shardInstruction.stakeInstructions) != 0 {
		Logger.log.Info("Beacon Producer/ Process Stakers List ", shardInstruction.stakeInstructions)
	}
	if len(shardInstruction.swapInstructions[shardID]) != 0 {
		Logger.log.Info("Beacon Producer/ Process Swap List ", shardInstruction.swapInstructions[shardID])
	}

	return shardInstruction
}

func (beaconBestState *BeaconBestState) processStakeInstructionFromShardBlock(
	shardInstructions *shardInstruction, validStakePublicKeys []string) (
	*shardInstruction, *duplicateKeyStakeInstruction) {

	duplicateKeyStakeInstruction := &duplicateKeyStakeInstruction{}
	newShardInstructions := shardInstructions
	newStakeInstructions := []*instruction.StakeInstruction{}
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
		newStakeInstructions = append(newStakeInstructions, tempStakeShardInstruction)
	}

	newShardInstructions.stakeInstructions = newStakeInstructions
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
	unstakeKeys := map[string]bool{}

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
		for _, key := range v.CommitteePublicKeys {
			unstakeKeys[key] = true
		}
		unstakeInstruction.CommitteePublicKeys = append(unstakeInstruction.CommitteePublicKeys, v.CommitteePublicKeys...)
		unstakeInstruction.CommitteePublicKeysStruct = append(unstakeInstruction.CommitteePublicKeysStruct, v.CommitteePublicKeysStruct...)
	}

	for _, v := range shardInstruction.stopAutoStakeInstructions {
		if v.IsEmpty() {
			continue
		}

		committeePublicKeys := []string{}
		for _, key := range v.CommitteePublicKeys {
			if !unstakeKeys[key] {
				committeePublicKeys = append(committeePublicKeys, key)
			}
		}

		stopAutoStakeInstruction.CommitteePublicKeys = append(stopAutoStakeInstruction.CommitteePublicKeys, committeePublicKeys...)
	}

	shardInstruction.stakeInstructions = []*instruction.StakeInstruction{}
	shardInstruction.unstakeInstructions = []*instruction.UnstakeInstruction{}
	shardInstruction.stopAutoStakeInstructions = []*instruction.StopAutoStakeInstruction{}

	if !stakeInstruction.IsEmpty() {
		shardInstruction.stakeInstructions = append(shardInstruction.stakeInstructions, stakeInstruction)
	}

	if !unstakeInstruction.IsEmpty() {
		shardInstruction.unstakeInstructions = append(shardInstruction.unstakeInstructions, unstakeInstruction)
	}

	if !stopAutoStakeInstruction.IsEmpty() {
		shardInstruction.stopAutoStakeInstructions = append(shardInstruction.stopAutoStakeInstructions, stopAutoStakeInstruction)
	}
}
