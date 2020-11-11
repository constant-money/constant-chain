package committeestate

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/pkg/errors"

	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/privacy"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type BeaconCommitteeStateV2 struct {
	beaconCommittee            []incognitokey.CommitteePublicKey
	shardCommittee             map[byte][]incognitokey.CommitteePublicKey
	shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
	shardCommonPool            []incognitokey.CommitteePublicKey
	numberOfAssignedCandidates int

	autoStake      map[string]bool                   // committee public key => true or false
	rewardReceiver map[string]privacy.PaymentAddress // incognito public key => reward receiver payment address
	stakingTx      map[string]common.Hash            // committee public key => reward receiver payment address

	mu *sync.RWMutex
}

type BeaconCommitteeEngineV2 struct {
	beaconHeight                      uint64
	beaconHash                        common.Hash
	finalBeaconCommitteeStateV2       *BeaconCommitteeStateV2
	uncommittedBeaconCommitteeStateV2 *BeaconCommitteeStateV2
}

func NewBeaconCommitteeEngineV2(beaconHeight uint64, beaconHash common.Hash, finalBeaconCommitteeStateV2 *BeaconCommitteeStateV2) *BeaconCommitteeEngineV2 {
	return &BeaconCommitteeEngineV2{
		beaconHeight:                      beaconHeight,
		beaconHash:                        beaconHash,
		finalBeaconCommitteeStateV2:       finalBeaconCommitteeStateV2,
		uncommittedBeaconCommitteeStateV2: NewBeaconCommitteeStateV2(),
	}
}

func NewBeaconCommitteeStateV2() *BeaconCommitteeStateV2 {
	return &BeaconCommitteeStateV2{
		shardCommittee:  make(map[byte][]incognitokey.CommitteePublicKey),
		shardSubstitute: make(map[byte][]incognitokey.CommitteePublicKey),
		autoStake:       make(map[string]bool),
		rewardReceiver:  make(map[string]privacy.PaymentAddress),
		stakingTx:       make(map[string]common.Hash),
		mu:              new(sync.RWMutex),
	}
}

func NewBeaconCommitteeStateV2WithValue(
	beaconCommittee []incognitokey.CommitteePublicKey,
	shardCommittee map[byte][]incognitokey.CommitteePublicKey,
	shardSubstitute map[byte][]incognitokey.CommitteePublicKey,
	shardCommonPool []incognitokey.CommitteePublicKey,
	numberOfAssignedCandidates int,
	autoStake map[string]bool,
	rewardReceiver map[string]privacy.PaymentAddress,
	stakingTx map[string]common.Hash,
	numberOfRound map[string]int,
) *BeaconCommitteeStateV2 {
	return &BeaconCommitteeStateV2{
		beaconCommittee:            beaconCommittee,
		shardCommittee:             shardCommittee,
		shardSubstitute:            shardSubstitute,
		shardCommonPool:            shardCommonPool,
		numberOfAssignedCandidates: numberOfAssignedCandidates,
		autoStake:                  autoStake,
		rewardReceiver:             rewardReceiver,
		stakingTx:                  stakingTx,
		mu:                         new(sync.RWMutex),
	}
}

func (b BeaconCommitteeStateV2) clone(newB *BeaconCommitteeStateV2) {
	newB.reset()
	newB.beaconCommittee = make([]incognitokey.CommitteePublicKey, len(b.beaconCommittee))
	copy(newB.beaconCommittee, b.beaconCommittee)
	newB.numberOfAssignedCandidates = b.numberOfAssignedCandidates
	newB.shardCommonPool = make([]incognitokey.CommitteePublicKey, len(b.shardCommonPool))
	copy(newB.shardCommonPool, b.shardCommonPool)

	for i, v := range b.shardCommittee {
		newB.shardCommittee[i] = make([]incognitokey.CommitteePublicKey, len(v))
		copy(newB.shardCommittee[i], v)
	}

	for i, v := range b.shardSubstitute {
		newB.shardSubstitute[i] = make([]incognitokey.CommitteePublicKey, len(v))
		copy(newB.shardSubstitute[i], v)
	}

	for k, v := range b.autoStake {
		newB.autoStake[k] = v
	}

	for k, v := range b.rewardReceiver {
		newB.rewardReceiver[k] = v
	}

	for k, v := range b.stakingTx {
		newB.stakingTx[k] = v
	}
}

func (b *BeaconCommitteeStateV2) reset() {
	b.beaconCommittee = []incognitokey.CommitteePublicKey{}
	b.shardCommonPool = []incognitokey.CommitteePublicKey{}
	b.shardCommittee = make(map[byte][]incognitokey.CommitteePublicKey)
	b.shardSubstitute = make(map[byte][]incognitokey.CommitteePublicKey)
	b.autoStake = make(map[string]bool)
	b.rewardReceiver = make(map[string]privacy.PaymentAddress)
	b.stakingTx = make(map[string]common.Hash)
}

//Clone :
func (engine *BeaconCommitteeEngineV2) Clone() BeaconCommitteeEngine {

	finalCommitteeState := NewBeaconCommitteeStateV2()
	engine.finalBeaconCommitteeStateV2.clone(finalCommitteeState)
	engine.uncommittedBeaconCommitteeStateV2 = NewBeaconCommitteeStateV2()

	res := NewBeaconCommitteeEngineV2(
		engine.beaconHeight,
		engine.beaconHash,
		finalCommitteeState,
	)

	return res
}

//Version :
func (engine BeaconCommitteeEngineV2) Version() uint {
	return SLASHING_VERSION
}

//GetBeaconHeight :
func (engine BeaconCommitteeEngineV2) GetBeaconHeight() uint64 {
	return engine.beaconHeight
}

//GetBeaconHash :
func (engine BeaconCommitteeEngineV2) GetBeaconHash() common.Hash {
	return engine.beaconHash
}

//GetBeaconCommittee :
func (engine BeaconCommitteeEngineV2) GetBeaconCommittee() []incognitokey.CommitteePublicKey {
	return engine.finalBeaconCommitteeStateV2.beaconCommittee
}

//GetBeaconSubstitute :
func (engine BeaconCommitteeEngineV2) GetBeaconSubstitute() []incognitokey.CommitteePublicKey {
	return []incognitokey.CommitteePublicKey{}
}

//GetCandidateShardWaitingForCurrentRandom :
func (engine BeaconCommitteeEngineV2) GetCandidateShardWaitingForCurrentRandom() []incognitokey.CommitteePublicKey {
	return engine.finalBeaconCommitteeStateV2.shardCommonPool[:engine.finalBeaconCommitteeStateV2.numberOfAssignedCandidates]
}

//GetCandidateBeaconWaitingForCurrentRandom :
func (engine BeaconCommitteeEngineV2) GetCandidateBeaconWaitingForCurrentRandom() []incognitokey.CommitteePublicKey {
	return []incognitokey.CommitteePublicKey{}
}

//GetCandidateShardWaitingForNextRandom :
func (engine BeaconCommitteeEngineV2) GetCandidateShardWaitingForNextRandom() []incognitokey.CommitteePublicKey {
	return engine.finalBeaconCommitteeStateV2.shardCommonPool[engine.finalBeaconCommitteeStateV2.numberOfAssignedCandidates:]
}

//GetCandidateBeaconWaitingForNextRandom :
func (engine BeaconCommitteeEngineV2) GetCandidateBeaconWaitingForNextRandom() []incognitokey.CommitteePublicKey {
	return []incognitokey.CommitteePublicKey{}
}

//GetOneShardCommittee :
func (engine BeaconCommitteeEngineV2) GetOneShardCommittee(shardID byte) []incognitokey.CommitteePublicKey {
	return engine.finalBeaconCommitteeStateV2.shardCommittee[shardID]
}

//GetShardCommittee :
func (engine BeaconCommitteeEngineV2) GetShardCommittee() map[byte][]incognitokey.CommitteePublicKey {
	engine.finalBeaconCommitteeStateV2.mu.RLock()
	defer engine.finalBeaconCommitteeStateV2.mu.RUnlock()
	shardCommittee := make(map[byte][]incognitokey.CommitteePublicKey)
	for k, v := range engine.finalBeaconCommitteeStateV2.shardCommittee {
		shardCommittee[k] = v
	}
	return shardCommittee
}

//GetOneShardSubstitute :
func (engine BeaconCommitteeEngineV2) GetOneShardSubstitute(shardID byte) []incognitokey.CommitteePublicKey {
	return engine.finalBeaconCommitteeStateV2.shardSubstitute[shardID]
}

//GetShardSubstitute :
func (engine BeaconCommitteeEngineV2) GetShardSubstitute() map[byte][]incognitokey.CommitteePublicKey {
	engine.finalBeaconCommitteeStateV2.mu.RLock()
	defer engine.finalBeaconCommitteeStateV2.mu.RUnlock()
	shardSubstitute := make(map[byte][]incognitokey.CommitteePublicKey)
	for k, v := range engine.finalBeaconCommitteeStateV2.shardSubstitute {
		shardSubstitute[k] = v
	}
	return shardSubstitute
}

//GetAutoStaking :
func (engine BeaconCommitteeEngineV2) GetAutoStaking() map[string]bool {
	engine.finalBeaconCommitteeStateV2.mu.RLock()
	defer engine.finalBeaconCommitteeStateV2.mu.RUnlock()
	autoStake := make(map[string]bool)
	for k, v := range engine.finalBeaconCommitteeStateV2.autoStake {
		autoStake[k] = v
	}
	return autoStake
}

func (engine BeaconCommitteeEngineV2) GetRewardReceiver() map[string]privacy.PaymentAddress {
	engine.finalBeaconCommitteeStateV2.mu.RLock()
	defer engine.finalBeaconCommitteeStateV2.mu.RUnlock()
	rewardReceiver := make(map[string]privacy.PaymentAddress)
	for k, v := range engine.finalBeaconCommitteeStateV2.rewardReceiver {
		rewardReceiver[k] = v
	}
	return rewardReceiver
}

func (engine BeaconCommitteeEngineV2) GetStakingTx() map[string]common.Hash {
	engine.finalBeaconCommitteeStateV2.mu.RLock()
	defer engine.finalBeaconCommitteeStateV2.mu.RUnlock()
	stakingTx := make(map[string]common.Hash)
	for k, v := range engine.finalBeaconCommitteeStateV2.stakingTx {
		stakingTx[k] = v
	}
	return stakingTx
}

func (engine *BeaconCommitteeEngineV2) GetAllCandidateSubstituteCommittee() []string {
	engine.finalBeaconCommitteeStateV2.mu.RLock()
	defer engine.finalBeaconCommitteeStateV2.mu.RUnlock()
	return engine.finalBeaconCommitteeStateV2.getAllCandidateSubstituteCommittee()
}

func (engine *BeaconCommitteeEngineV2) compareHashes(hash1, hash2 *BeaconCommitteeStateHash) error {
	if !hash1.BeaconCommitteeAndValidatorHash.IsEqual(&hash2.BeaconCommitteeAndValidatorHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState,
			fmt.Errorf("Uncommitted BeaconCommitteeAndValidatorHash want value %+v but have %+v",
				hash1.BeaconCommitteeAndValidatorHash, hash2.BeaconCommitteeAndValidatorHash))
	}
	if !hash1.BeaconCandidateHash.IsEqual(&hash2.BeaconCandidateHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState,
			fmt.Errorf("Uncommitted BeaconCandidateHash want value %+v but have %+v",
				hash1.BeaconCandidateHash, hash2.BeaconCandidateHash))
	}
	if !hash1.ShardCandidateHash.IsEqual(&hash2.ShardCandidateHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState,
			fmt.Errorf("Uncommitted ShardCandidateHash want value %+v but have %+v",
				hash1.ShardCandidateHash, hash2.ShardCandidateHash))
	}
	if !hash1.ShardCommitteeAndValidatorHash.IsEqual(&hash2.ShardCommitteeAndValidatorHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState,
			fmt.Errorf("Uncommitted ShardCommitteeAndValidatorHash want value %+v but have %+v",
				hash1.ShardCommitteeAndValidatorHash, hash2.ShardCommitteeAndValidatorHash))
	}
	if !hash1.AutoStakeHash.IsEqual(&hash2.AutoStakeHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState,
			fmt.Errorf("Uncommitted AutoStakingHash want value %+v but have %+v",
				hash1.AutoStakeHash, hash2.AutoStakeHash))
	}
	return nil
}

func (engine *BeaconCommitteeEngineV2) Commit(hashes *BeaconCommitteeStateHash) error {
	if reflect.DeepEqual(engine.uncommittedBeaconCommitteeStateV2, NewBeaconCommitteeStateV2()) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, fmt.Errorf("%+v", engine.uncommittedBeaconCommitteeStateV2))
	}
	engine.uncommittedBeaconCommitteeStateV2.mu.Lock()
	defer engine.uncommittedBeaconCommitteeStateV2.mu.Unlock()
	engine.finalBeaconCommitteeStateV2.mu.Lock()
	defer engine.finalBeaconCommitteeStateV2.mu.Unlock()
	comparedHashes, err := engine.generateCommitteeHashes(engine.uncommittedBeaconCommitteeStateV2)
	if err != nil {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, err)
	}
	err = engine.compareHashes(comparedHashes, hashes)
	if err != nil {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, err)
	}
	engine.uncommittedBeaconCommitteeStateV2.clone(engine.finalBeaconCommitteeStateV2)
	engine.uncommittedBeaconCommitteeStateV2.reset()
	return nil
}

func (engine *BeaconCommitteeEngineV2) AbortUncommittedBeaconState() {
	engine.uncommittedBeaconCommitteeStateV2.mu.Lock()
	defer engine.uncommittedBeaconCommitteeStateV2.mu.Unlock()
	engine.uncommittedBeaconCommitteeStateV2.reset()
}

func (engine *BeaconCommitteeEngineV2) InitCommitteeState(env *BeaconCommitteeStateEnvironment) {
	engine.finalBeaconCommitteeStateV2.mu.Lock()
	defer engine.finalBeaconCommitteeStateV2.mu.Unlock()
	b := engine.finalBeaconCommitteeStateV2
	newBeaconCandidates := []incognitokey.CommitteePublicKey{}
	newShardCandidates := []incognitokey.CommitteePublicKey{}
	for _, inst := range env.BeaconInstructions {
		if len(inst) == 0 {
			continue
		}
		if inst[0] == instruction.STAKE_ACTION {
			stakeInstruction := instruction.ImportInitStakeInstructionFromString(inst)
			for index, candidate := range stakeInstruction.PublicKeyStructs {
				b.rewardReceiver[candidate.GetIncKeyBase58()] = stakeInstruction.RewardReceiverStructs[index]
				b.autoStake[stakeInstruction.PublicKeys[index]] = stakeInstruction.AutoStakingFlag[index]
				b.stakingTx[stakeInstruction.PublicKeys[index]] = stakeInstruction.TxStakeHashes[index]
			}
			if stakeInstruction.Chain == instruction.BEACON_INST {
				newBeaconCandidates = append(newBeaconCandidates, stakeInstruction.PublicKeyStructs...)
			} else {
				newShardCandidates = append(newShardCandidates, stakeInstruction.PublicKeyStructs...)
			}
			_ = statedb.StoreStakerInfoV1(
				env.ConsensusStateDB,
				stakeInstruction.PublicKeyStructs,
				b.rewardReceiver,
				b.autoStake,
				b.stakingTx,
			)
		}
	}
	b.beaconCommittee = append(b.beaconCommittee, newBeaconCandidates...)
	for shardID := 0; shardID < env.ActiveShards; shardID++ {
		b.shardCommittee[byte(shardID)] = append(b.shardCommittee[byte(shardID)], newShardCandidates[shardID*env.MinShardCommitteeSize:(shardID+1)*env.MinShardCommitteeSize]...)
	}
}

// UpdateCommitteeState New flow
// Store information from instructions into temp stateDB in env
// When all thing done and no problems, in commit function, we read data in statedb and update
func (engine *BeaconCommitteeEngineV2) UpdateCommitteeState(env *BeaconCommitteeStateEnvironment) (
	*BeaconCommitteeStateHash, *CommitteeChange, [][]string, error) {
	var err error
	incurredInstructions := [][]string{}
	returnStakingInstructions := make(map[byte]*instruction.ReturnStakeInstruction)

	engine.finalBeaconCommitteeStateV2.mu.RLock()
	engine.finalBeaconCommitteeStateV2.clone(engine.uncommittedBeaconCommitteeStateV2)
	env.unassignedCommonPool, err = engine.finalBeaconCommitteeStateV2.unassignedCommonPool()
	if err != nil {
		return nil, nil, nil, err
	}
	env.allSubstituteCommittees, err = engine.finalBeaconCommitteeStateV2.getAllSubstituteCommittees()
	if err != nil {
		return nil, nil, nil, err
	}
	env.allCandidateSubstituteCommittee = append(env.unassignedCommonPool, env.allSubstituteCommittees...)
	engine.finalBeaconCommitteeStateV2.mu.RUnlock()
	newB := engine.uncommittedBeaconCommitteeStateV2
	committeeChange := NewCommitteeChange()
	// snapshot shard common pool in beacon random time
	if env.IsBeaconRandomTime {
		newB.numberOfAssignedCandidates = SnapshotShardCommonPoolV2(
			newB.shardCommonPool,
			newB.shardCommittee,
			newB.shardSubstitute,
			env.NumberOfFixedShardBlockValidators,
			env.MinShardCommitteeSize,
		)
		Logger.log.Infof("Block %+v, Number of Snapshot to Assign Candidate %+v", env.BeaconHeight, newB.numberOfAssignedCandidates)
	}

	for _, inst := range env.BeaconInstructions {
		if len(inst) == 0 {
			continue
		}
		switch inst[0] {
		case instruction.STAKE_ACTION:
			stakeInstruction, err := instruction.ValidateAndImportStakeInstructionFromString(inst)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
			committeeChange, err = newB.processStakeInstruction(stakeInstruction, committeeChange, env)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
		case instruction.RANDOM_ACTION:
			randomInstruction, err := instruction.ValidateAndImportRandomInstructionFromString(inst)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
			committeeChange = newB.processAssignWithRandomInstruction(
				randomInstruction.BtcNonce, env.ActiveShards, committeeChange)
			Logger.log.Infof("Block %+v, Committee Change %+v", env.BeaconHeight, committeeChange.ShardSubstituteAdded)
		case instruction.STOP_AUTO_STAKE_ACTION:
			stopAutoStakeInstruction, err := instruction.ValidateAndImportStopAutoStakeInstructionFromString(inst)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
			committeeChange = newB.processStopAutoStakeInstruction(stopAutoStakeInstruction, env, committeeChange)
		case instruction.UNSTAKE_ACTION:
			unstakeInstruction, err := instruction.ValidateAndImportUnstakeInstructionFromString(inst)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
			committeeChange, returnStakingInstructions, err =
				newB.processUnstakeInstruction(unstakeInstruction, env, committeeChange, returnStakingInstructions)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
		case instruction.SWAP_SHARD_ACTION:
			swapShardInstruction, err := instruction.ValidateAndImportSwapShardInstructionFromString(inst)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
			committeeChange, returnStakingInstructions, err = newB.
				processSwapShardInstruction(swapShardInstruction, env, committeeChange, returnStakingInstructions)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
		}
	}

	hashes, err := engine.generateCommitteeHashes(engine.uncommittedBeaconCommitteeStateV2)
	if err != nil {
		return nil, nil, incurredInstructions, NewCommitteeStateError(ErrUpdateCommitteeState, err)
	}

	keys := []int{}
	for k, _ := range returnStakingInstructions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, key := range keys {
		shardID := byte(key)
		incurredInstructions = append(incurredInstructions, returnStakingInstructions[shardID].ToString())
	}

	return hashes, committeeChange, incurredInstructions, nil
}

// GenerateAllSwapShardInstructions generate swap shard instructions for all shard
// it also assigned swapped out committee back to substitute list if auto stake is true
// generate all swap shard instructions by only swap by the end of epoch (normally execution)
func (engine *BeaconCommitteeEngineV2) GenerateAllSwapShardInstructions(
	env *BeaconCommitteeStateEnvironment) (
	[]*instruction.SwapShardInstruction, error) {
	swapShardInstructions := []*instruction.SwapShardInstruction{}
	for i := 0; i < env.ActiveShards; i++ {
		shardID := byte(i)
		committees := engine.finalBeaconCommitteeStateV2.shardCommittee[shardID]
		substitutes := engine.finalBeaconCommitteeStateV2.shardSubstitute[shardID]
		if len(substitutes) == 0 {
			continue
		}
		tempCommittees, _ := incognitokey.CommitteeKeyListToString(committees)
		tempSubstitutes, _ := incognitokey.CommitteeKeyListToString(substitutes)

		swapShardInstruction, _, _, _ := createSwapShardInstructionV2(
			shardID,
			tempSubstitutes,
			tempCommittees,
			env.MinShardCommitteeSize,
			env.MaxShardCommitteeSize,
			instruction.SWAP_BY_END_EPOCH,
			env.NumberOfFixedShardBlockValidators,
		)

		if !swapShardInstruction.IsEmpty() {
			swapShardInstructions = append(swapShardInstructions, swapShardInstruction)
		} else {
			Logger.log.Infof("[swap-instructions] Generate empty instructions beacon hash: %s & height: %v \n", engine.beaconHash, engine.beaconHash)
		}
	}
	return swapShardInstructions, nil
}

func (engine *BeaconCommitteeEngineV2) GenerateAssignInstruction(rand int64, assignOffset int, activeShards int) ([]*instruction.AssignInstruction, []string, map[byte][]string) {
	return []*instruction.AssignInstruction{}, []string{}, make(map[byte][]string)
}

func (engine *BeaconCommitteeEngineV2) BuildIncurredInstructions(env *BeaconCommitteeStateEnvironment) ([][]string, error) {
	newB := NewBeaconCommitteeStateV2()
	engine.finalBeaconCommitteeStateV2.clone(newB)
	committeeChange := NewCommitteeChange()

	incurredInstructions := [][]string{}
	if env == nil {
		return incurredInstructions, errors.New("Environment Variable Is Null")
	}
	if len(env.BeaconInstructions) == 0 {
		return incurredInstructions, nil
	}
	var err error
	returnStakingInstructions := make(map[byte]*instruction.ReturnStakeInstruction)
	env.unassignedCommonPool, err = newB.unassignedCommonPool()
	if err != nil {
		return incurredInstructions, err
	}
	env.allSubstituteCommittees, err = newB.getAllSubstituteCommittees()
	if err != nil {
		return incurredInstructions, err
	}
	for _, inst := range env.BeaconInstructions {
		switch inst[0] {
		case instruction.UNSTAKE_ACTION:
			unstakeInstruction, err := instruction.ValidateAndImportUnstakeInstructionFromString(inst)
			if err != nil {
				Logger.log.Errorf("SKIP unstake instruction %+v, error %+v", inst, err)
				return incurredInstructions, err
			}
			_, returnStakingInstructions, err =
				newB.processUnstakeInstruction(unstakeInstruction, env, committeeChange, returnStakingInstructions)
			if err != nil {
				return incurredInstructions, NewCommitteeStateError(ErrBuildIncurredInstruction, err)
			}
		case instruction.SWAP_SHARD_ACTION:
			swapShardInstruction, err := instruction.ValidateAndImportSwapShardInstructionFromString(inst)
			if err != nil {
				return incurredInstructions, err
			}
			committeeChange, returnStakingInstructions, err = newB.
				processSwapShardInstruction(swapShardInstruction, env, committeeChange, returnStakingInstructions)
			if err != nil {
				return incurredInstructions, err
			}
		}
	}

	keys := []int{}
	for k, _ := range returnStakingInstructions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, key := range keys {
		shardID := byte(key)
		incurredInstructions = append(incurredInstructions, returnStakingInstructions[shardID].ToString())
	}

	return incurredInstructions, nil
}

func SnapshotShardCommonPoolV2(
	shardCommonPool []incognitokey.CommitteePublicKey,
	shardCommittee map[byte][]incognitokey.CommitteePublicKey,
	shardSubstitute map[byte][]incognitokey.CommitteePublicKey,
	numberOfFixedValidator int,
	minCommitteeSize int,
) (numberOfAssignedCandidates int) {
	for k, v := range shardCommittee {
		assignPerShard := getSwapOutOffset(
			len(shardSubstitute[k]),
			len(v),
			numberOfFixedValidator,
			minCommitteeSize,
		)

		if assignPerShard == 0 {
			assignPerShard = len(v) / 3
		}

		numberOfAssignedCandidates += assignPerShard
		Logger.log.Infof("SnapshotShardCommonPoolV2 | Shard %+v, numberOfAssignedCandidates %+v", k, numberOfAssignedCandidates)
	}
	Logger.log.Infof("SnapshotShardCommonPoolV2 | Shard Common Pool Size %+v", len(shardCommonPool))

	if numberOfAssignedCandidates > len(shardCommonPool) {
		numberOfAssignedCandidates = len(shardCommonPool)
	}
	return numberOfAssignedCandidates
}

func (b *BeaconCommitteeStateV2) processStakeInstruction(
	stakeInstruction *instruction.StakeInstruction,
	committeeChange *CommitteeChange,
	env *BeaconCommitteeStateEnvironment,
) (*CommitteeChange, error) {
	var err error
	// var key string
	for index, candidate := range stakeInstruction.PublicKeyStructs {
		committeePublicKey := stakeInstruction.PublicKeys[index]
		b.rewardReceiver[candidate.GetIncKeyBase58()] = stakeInstruction.RewardReceiverStructs[index]
		b.autoStake[committeePublicKey] = stakeInstruction.AutoStakingFlag[index]
		b.stakingTx[committeePublicKey] = stakeInstruction.TxStakeHashes[index]
	}
	committeeChange.NextEpochShardCandidateAdded = append(committeeChange.NextEpochShardCandidateAdded, stakeInstruction.PublicKeyStructs...)
	b.shardCommonPool = append(b.shardCommonPool, stakeInstruction.PublicKeyStructs...)
	return committeeChange, err
}

func (b *BeaconCommitteeStateV2) processStopAutoStakeInstruction(
	stopAutoStakeInstruction *instruction.StopAutoStakeInstruction,
	env *BeaconCommitteeStateEnvironment,
	committeeChange *CommitteeChange,
) *CommitteeChange {
	for _, committeePublicKey := range stopAutoStakeInstruction.CommitteePublicKeys {
		if common.IndexOfStr(committeePublicKey, env.allCandidateSubstituteCommittee) == -1 {
			// if not found then delete auto staking data for this public key if present
			if _, ok := b.autoStake[committeePublicKey]; ok {
				delete(b.autoStake, committeePublicKey)
			}
		} else {
			// if found in committee list then turn off auto staking
			if _, ok := b.autoStake[committeePublicKey]; ok {
				committeeChange = b.stopAutoStake(committeePublicKey, committeeChange)
			}
		}
	}
	return committeeChange
}

func (b *BeaconCommitteeStateV2) processAssignWithRandomInstruction(
	rand int64, activeShards int, committeeChange *CommitteeChange) *CommitteeChange {
	newCommitteeChange := committeeChange
	candidates, _ := incognitokey.CommitteeKeyListToString(b.shardCommonPool[:b.numberOfAssignedCandidates])
	newCommitteeChange = b.assign(candidates, rand, activeShards, newCommitteeChange)
	newCommitteeChange.NextEpochShardCandidateRemoved = append(newCommitteeChange.NextEpochShardCandidateRemoved, b.shardCommonPool[:b.numberOfAssignedCandidates]...)
	b.shardCommonPool = b.shardCommonPool[b.numberOfAssignedCandidates:]
	b.numberOfAssignedCandidates = 0
	return newCommitteeChange
}

func (b *BeaconCommitteeStateV2) assign(
	candidates []string, rand int64, activeShards int, committeeChange *CommitteeChange) *CommitteeChange {
	numberOfValidator := make([]int, activeShards)

	for i := 0; i < activeShards; i++ {
		numberOfValidator[byte(i)] += len(b.shardSubstitute[byte(i)])
		numberOfValidator[byte(i)] += len(b.shardCommittee[byte(i)])
	}

	assignedCandidates := assignShardCandidateV2(candidates, numberOfValidator, rand)
	for shardID, tempCandidates := range assignedCandidates {
		candidates, _ := incognitokey.CommitteeBase58KeyListToStruct(tempCandidates)
		b.shardSubstitute[shardID] = append(b.shardSubstitute[shardID], candidates...)
		committeeChange.ShardSubstituteAdded[shardID] = append(committeeChange.ShardSubstituteAdded[shardID], candidates...)
	}
	return committeeChange
}

//processSwapShardInstruction update committees state by swap shard instruction
// Process single swap shard instruction for and update committee state
func (b *BeaconCommitteeStateV2) processSwapShardInstruction(
	swapShardInstruction *instruction.SwapShardInstruction,
	env *BeaconCommitteeStateEnvironment, committeeChange *CommitteeChange,
	returnStakingInstructions map[byte]*instruction.ReturnStakeInstruction,
) (
	*CommitteeChange, map[byte]*instruction.ReturnStakeInstruction, error) {

	var err error
	chainID := byte(swapShardInstruction.ChainID)
	newCommitteeChange := committeeChange
	tempSwapOutPublicKeys := swapShardInstruction.OutPublicKeyStructs
	tempSwapInPublicKeys := swapShardInstruction.InPublicKeyStructs
	numberFixedValidators := env.NumberOfFixedShardBlockValidators

	// process list shard committees
	for _, v := range tempSwapOutPublicKeys {
		if !v.IsEqual(b.shardCommittee[chainID][numberFixedValidators]) {
			return nil, returnStakingInstructions, errors.New("Swap Out Not Valid In List Committees Public Key")
		}
		b.shardCommittee[chainID] = append(b.shardCommittee[chainID][:numberFixedValidators],
			b.shardCommittee[chainID][numberFixedValidators+1:]...)
		newCommitteeChange.ShardCommitteeRemoved[chainID] = append(newCommitteeChange.ShardCommitteeRemoved[chainID], v)
	}
	b.shardCommittee[chainID] = append(b.shardCommittee[chainID], tempSwapInPublicKeys...)

	// process list shard pool
	for _, v := range tempSwapInPublicKeys {
		if !v.IsEqual(b.shardSubstitute[chainID][0]) {
			return nil, returnStakingInstructions, errors.New("Swap In Not Valid In List Subtitutes Public Key")
		}
		b.shardSubstitute[chainID] = b.shardSubstitute[chainID][1:]
		newCommitteeChange.ShardSubstituteRemoved[chainID] = append(newCommitteeChange.ShardSubstituteRemoved[chainID], v)
	}
	newCommitteeChange.ShardCommitteeAdded[chainID] = append(newCommitteeChange.ShardCommitteeAdded[chainID], tempSwapInPublicKeys...)

	// process after swap for assign old committees to current shard pool
	newCommitteeChange, returnStakingInstructions, err = b.processAfterSwap(env,
		swapShardInstruction.OutPublicKeys,
		swapShardInstruction.OutPublicKeyStructs,
		chainID,
		newCommitteeChange,
		returnStakingInstructions,
	)

	if err != nil {
		return nil, returnStakingInstructions, err
	}

	return newCommitteeChange, returnStakingInstructions, nil
}

// processAfterSwap process swapped out committee public key
// if number of round is less than MAX_NUMBER_OF_ROUND go back to THAT shard pool, and increase number of round
// if number of round is equal to or greater than MAX_NUMBER_OF_ROUND
// - auto stake is false then remove completely out of any committee, candidate, substitute list
// - auto stake is true then using assignment rule v2 to assign this committee public key
func (b *BeaconCommitteeStateV2) processAfterSwap(
	env *BeaconCommitteeStateEnvironment,
	outPublicKeys []string,
	outPublicKeyStructs []incognitokey.CommitteePublicKey,
	shardID byte,
	committeeChange *CommitteeChange,
	returnStakingInstructions map[byte]*instruction.ReturnStakeInstruction,
) (*CommitteeChange, map[byte]*instruction.ReturnStakeInstruction, error) {
	candidates := []string{}
	newCommitteeChange := committeeChange
	for index, outPublicKey := range outPublicKeys {
		stakerInfo, has, err := statedb.GetStakerInfo(env.ConsensusStateDB, outPublicKey)
		if err != nil {
			return newCommitteeChange, returnStakingInstructions, err
		}
		if !has {
			return newCommitteeChange, returnStakingInstructions, errors.Errorf("Can not found info of this public key %v", outPublicKey)
		}
		if stakerInfo.AutoStaking() {
			candidates = append(candidates, outPublicKey)
		} else {
			returnStakingInstructions, err = b.buildReturnStakingInstructionAndDeleteStakerInfo(
				returnStakingInstructions,
				outPublicKeyStructs[index],
				outPublicKey,
				stakerInfo,
			)
			if err != nil {
				return newCommitteeChange, returnStakingInstructions, err
			}
		}
	}

	newCommitteeChange = b.assign(candidates, env.RandomNumber, env.ActiveShards, newCommitteeChange)
	return newCommitteeChange, returnStakingInstructions, nil
}

//processUnstakeInstruction : process unstake instruction from beacon block
func (b *BeaconCommitteeStateV2) processUnstakeInstruction(
	unstakeInstruction *instruction.UnstakeInstruction,
	env *BeaconCommitteeStateEnvironment,
	committeeChange *CommitteeChange,
	returnStakingInstructions map[byte]*instruction.ReturnStakeInstruction,
) (*CommitteeChange, map[byte]*instruction.ReturnStakeInstruction, error) {
	newCommitteeChange := committeeChange
	indexNextEpochShardCandidate := make(map[string]int)
	for i, v := range b.shardCommonPool {
		key, err := v.ToBase58()
		if err != nil {
			return newCommitteeChange, returnStakingInstructions, err
		}
		indexNextEpochShardCandidate[key] = i
	}
	for index, publicKey := range unstakeInstruction.CommitteePublicKeys {
		if common.IndexOfStr(publicKey, env.unassignedCommonPool) == -1 {
			if common.IndexOfStr(publicKey, env.allSubstituteCommittees) != -1 {
				// if found in committee list then turn off auto staking
				if _, ok := b.autoStake[publicKey]; ok {
					newCommitteeChange = b.stopAutoStake(publicKey, newCommitteeChange)
				}
			}
		} else {
			indexCandidate := indexNextEpochShardCandidate[publicKey]
			b.shardCommonPool = append(b.shardCommonPool[:indexCandidate], b.shardCommonPool[indexCandidate+1:]...)
			stakerInfo, has, err := statedb.GetStakerInfo(env.ConsensusStateDB, publicKey)
			if err != nil {
				return newCommitteeChange, returnStakingInstructions, err
			}
			if !has {
				return newCommitteeChange, returnStakingInstructions, errors.New("Can't find staker info")
			}

			newCommitteeChange.NextEpochShardCandidateRemoved =
				append(newCommitteeChange.NextEpochShardCandidateRemoved, unstakeInstruction.CommitteePublicKeysStruct[index])

			returnStakingInstructions, err = b.buildReturnStakingInstructionAndDeleteStakerInfo(
				returnStakingInstructions,
				unstakeInstruction.CommitteePublicKeysStruct[index],
				publicKey,
				stakerInfo,
			)
			if err != nil {
				return newCommitteeChange, returnStakingInstructions, errors.New("Can't find staker info")
			}
			newCommitteeChange.Unstake = append(newCommitteeChange.Unstake, publicKey)
		}
	}

	return newCommitteeChange, returnStakingInstructions, nil
}

func (engine *BeaconCommitteeEngineV2) generateCommitteeHashes(state *BeaconCommitteeStateV2) (*BeaconCommitteeStateHash, error) {
	if reflect.DeepEqual(state, NewBeaconCommitteeStateV2()) {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, empty uncommitted state")
	}
	newB := state
	// beacon committee
	beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(newB.beaconCommittee)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	validatorArr := append([]string{}, beaconCommitteeStr...)

	tempBeaconCommitteeAndValidatorHash, err := common.GenerateHashFromStringArray(validatorArr)
	// Shard candidate root: shard current candidate + shard next candidate
	shardNextEpochCandidateStr, err := incognitokey.CommitteeKeyListToString(newB.shardCommonPool)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	tempShardCandidateHash, err := common.GenerateHashFromStringArray(shardNextEpochCandidateStr)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	// Shard Validator root
	shardPendingValidator := make(map[byte][]string)
	for shardID, keys := range newB.shardSubstitute {
		keysStr, err := incognitokey.CommitteeKeyListToString(keys)
		if err != nil {
			return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
		}
		shardPendingValidator[shardID] = keysStr
	}
	shardCommittee := make(map[byte][]string)
	for shardID, keys := range newB.shardCommittee {
		keysStr, err := incognitokey.CommitteeKeyListToString(keys)
		if err != nil {
			return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
		}
		shardCommittee[shardID] = keysStr
	}
	tempShardCommitteeAndValidatorHash, err := common.GenerateHashFromMapByteString(shardPendingValidator, shardCommittee)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	tempAutoStakingHash, err := common.GenerateHashFromMapStringBool(newB.autoStake)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	hashes := &BeaconCommitteeStateHash{
		BeaconCommitteeAndValidatorHash: tempBeaconCommitteeAndValidatorHash,
		ShardCandidateHash:              tempShardCandidateHash,
		ShardCommitteeAndValidatorHash:  tempShardCommitteeAndValidatorHash,
		AutoStakeHash:                   tempAutoStakingHash,
	}
	return hashes, nil
}

func (b *BeaconCommitteeStateV2) getAllCandidateSubstituteCommittee() []string {
	res := []string{}
	for _, committee := range b.shardCommittee {
		shardCommitteeStr, err := incognitokey.CommitteeKeyListToString(committee)
		if err != nil {
			panic(err)
		}
		res = append(res, shardCommitteeStr...)
	}
	for _, substitute := range b.shardSubstitute {
		beaconSubstituteStr, err := incognitokey.CommitteeKeyListToString(substitute)
		if err != nil {
			panic(err)
		}
		res = append(res, beaconSubstituteStr...)
	}
	beaconCommittee := b.beaconCommittee
	beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(beaconCommittee)
	if err != nil {
		panic(err)
	}
	res = append(res, beaconCommitteeStr...)
	shardCandidates := b.shardCommonPool
	shardCandidatesStr, err := incognitokey.CommitteeKeyListToString(shardCandidates)
	if err != nil {
		panic(err)
	}
	res = append(res, shardCandidatesStr...)
	return res
}

//HasSwappedCommittees ...
func (engine *BeaconCommitteeEngineV2) HasSwappedCommittees(env *BeaconCommitteeStateEnvironment) (bool, error) {
	if env.BeaconHeight%env.ParamEpoch == 0 {
		for _, inst := range env.BeaconInstructions {
			switch inst[0] {
			case instruction.SWAP_SHARD_ACTION:
				err := instruction.ValidateSwapShardInstructionSanity(inst)
				if err != nil {
					return false, err
				}
				return true, nil
			}
		}
	}
	return false, nil
}

func (b *BeaconCommitteeStateV2) stopAutoStake(publicKey string, committeeChange *CommitteeChange) *CommitteeChange {
	b.autoStake[publicKey] = false
	committeeChange.StopAutoStake = append(committeeChange.StopAutoStake, publicKey)
	return committeeChange
}

func (b *BeaconCommitteeStateV2) unassignedCommonPool() ([]string, error) {
	commonPoolValidators := []string{}
	candidateShardWaitingForNextRandomStr, err := incognitokey.CommitteeKeyListToString(b.shardCommonPool[b.numberOfAssignedCandidates:])
	if err != nil {
		return nil, err
	}
	commonPoolValidators = append(commonPoolValidators, candidateShardWaitingForNextRandomStr...)
	return commonPoolValidators, nil
}

func (b *BeaconCommitteeStateV2) getAllSubstituteCommittees() ([]string, error) {
	validators := []string{}

	for _, committee := range b.shardCommittee {
		committeeStr, err := incognitokey.CommitteeKeyListToString(committee)
		if err != nil {
			return nil, err
		}
		validators = append(validators, committeeStr...)
	}
	for _, substitute := range b.shardSubstitute {
		substituteStr, err := incognitokey.CommitteeKeyListToString(substitute)
		if err != nil {
			return nil, err
		}
		validators = append(validators, substituteStr...)
	}

	beaconCommittee := b.beaconCommittee
	beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(beaconCommittee)
	if err != nil {
		return nil, err
	}
	validators = append(validators, beaconCommitteeStr...)
	candidateShardWaitingForCurrentRandomStr, err := incognitokey.CommitteeKeyListToString(b.shardCommonPool[:b.numberOfAssignedCandidates])
	if err != nil {
		return nil, err
	}
	validators = append(validators, candidateShardWaitingForCurrentRandomStr...)

	return validators, nil
}

//ActiveShards ...
func (engine *BeaconCommitteeEngineV2) ActiveShards() int {
	return len(engine.finalBeaconCommitteeStateV2.shardCommittee)
}

func (b *BeaconCommitteeStateV2) buildReturnStakingInstructionAndDeleteStakerInfo(
	returnStakingInstructions map[byte]*instruction.ReturnStakeInstruction,
	committeePublicKeyStruct incognitokey.CommitteePublicKey,
	publicKey string,
	stakerInfo *statedb.StakerInfo,
) (map[byte]*instruction.ReturnStakeInstruction, error) {
	returnStakingInstructions = buildReturnStakingInstruction(
		returnStakingInstructions,
		publicKey,
		stakerInfo.ShardID(),
		stakerInfo.TxStakingID().String(),
	)
	err := b.deleteStakerInfo(committeePublicKeyStruct, publicKey)
	if err != nil {
		return returnStakingInstructions, err
	}
	return returnStakingInstructions, nil
}

func buildReturnStakingInstruction(
	returnStakingInstructions map[byte]*instruction.ReturnStakeInstruction,
	publicKey string,
	shardID byte,
	txStake string,
) map[byte]*instruction.ReturnStakeInstruction {
	returnStakingInstruction, ok := returnStakingInstructions[shardID]
	if !ok {
		returnStakingInstruction = instruction.NewReturnStakeInsWithValue(
			[]string{publicKey},
			shardID,
			[]string{txStake},
		)
	} else {
		returnStakingInstruction.AddInTheSameShard(publicKey, txStake)
	}
	returnStakingInstructions[shardID] = returnStakingInstruction
	return returnStakingInstructions
}

func (b *BeaconCommitteeStateV2) deleteStakerInfo(
	committeePublicKeyStruct incognitokey.CommitteePublicKey,
	committeePublicKey string,
) error {
	delete(b.rewardReceiver, committeePublicKeyStruct.GetIncKeyBase58())
	delete(b.autoStake, committeePublicKey)
	delete(b.stakingTx, committeePublicKey)
	return nil
}
