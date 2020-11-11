package committeestate

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
)

type BeaconCommitteeStateV1 struct {
	beaconCommittee             []incognitokey.CommitteePublicKey
	beaconSubstitute            []incognitokey.CommitteePublicKey
	nextEpochShardCandidate     []incognitokey.CommitteePublicKey
	currentEpochShardCandidate  []incognitokey.CommitteePublicKey
	nextEpochBeaconCandidate    []incognitokey.CommitteePublicKey
	currentEpochBeaconCandidate []incognitokey.CommitteePublicKey
	shardCommittee              map[byte][]incognitokey.CommitteePublicKey
	shardSubstitute             map[byte][]incognitokey.CommitteePublicKey
	autoStake                   map[string]bool                   // committee public key => reward receiver payment address
	rewardReceiver              map[string]privacy.PaymentAddress // incognito public key => reward receiver payment address
	stakingTx                   map[string]common.Hash            // committee public key => reward receiver payment address

	mu *sync.RWMutex
}

type BeaconCommitteeEngineV1 struct {
	beaconHeight                      uint64
	beaconHash                        common.Hash
	beaconCommitteeStateV1            *BeaconCommitteeStateV1
	uncommittedBeaconCommitteeStateV1 *BeaconCommitteeStateV1
	version                           uint
}

func NewBeaconCommitteeEngineV1(
	beaconHeight uint64,
	beaconHash common.Hash,
	beaconCommitteeStateV1 *BeaconCommitteeStateV1) *BeaconCommitteeEngineV1 {
	return &BeaconCommitteeEngineV1{
		beaconHeight:                      beaconHeight,
		beaconHash:                        beaconHash,
		beaconCommitteeStateV1:            beaconCommitteeStateV1,
		uncommittedBeaconCommitteeStateV1: NewBeaconCommitteeStateV1(),
	}
}

func NewBeaconCommitteeStateV1WithValue(
	beaconCommittee []incognitokey.CommitteePublicKey,
	beaconSubstitute []incognitokey.CommitteePublicKey,
	nextEpochShardCandidate []incognitokey.CommitteePublicKey,
	currentEpochShardCandidate []incognitokey.CommitteePublicKey,
	nextEpochBeaconCandidate []incognitokey.CommitteePublicKey,
	currentEpochBeaconCandidate []incognitokey.CommitteePublicKey,
	shardCommittee map[byte][]incognitokey.CommitteePublicKey,
	shardSubstitute map[byte][]incognitokey.CommitteePublicKey,
	autoStake map[string]bool,
	rewardReceiver map[string]privacy.PaymentAddress,
	stakingTx map[string]common.Hash,
) *BeaconCommitteeStateV1 {
	return &BeaconCommitteeStateV1{
		beaconCommittee:             beaconCommittee,
		beaconSubstitute:            beaconSubstitute,
		nextEpochShardCandidate:     nextEpochShardCandidate,
		currentEpochShardCandidate:  currentEpochShardCandidate,
		nextEpochBeaconCandidate:    nextEpochBeaconCandidate,
		currentEpochBeaconCandidate: currentEpochBeaconCandidate,
		shardCommittee:              shardCommittee,
		shardSubstitute:             shardSubstitute,
		autoStake:                   autoStake,
		rewardReceiver:              rewardReceiver,
		stakingTx:                   stakingTx,
		mu:                          new(sync.RWMutex),
	}
}

func NewBeaconCommitteeStateEnvironment() *BeaconCommitteeStateEnvironment {
	return &BeaconCommitteeStateEnvironment{}
}

func NewBeaconCommitteeStateV1() *BeaconCommitteeStateV1 {
	return &BeaconCommitteeStateV1{
		shardCommittee:  make(map[byte][]incognitokey.CommitteePublicKey),
		shardSubstitute: make(map[byte][]incognitokey.CommitteePublicKey),
		autoStake:       make(map[string]bool),
		rewardReceiver:  make(map[string]privacy.PaymentAddress),
		stakingTx:       make(map[string]common.Hash),
		mu:              new(sync.RWMutex),
	}
}

func (b BeaconCommitteeStateV1) clone(newB *BeaconCommitteeStateV1) {
	newB.reset()
	newB.beaconCommittee = b.beaconCommittee
	newB.beaconSubstitute = b.beaconSubstitute
	newB.currentEpochShardCandidate = b.currentEpochShardCandidate
	newB.currentEpochBeaconCandidate = b.currentEpochBeaconCandidate
	newB.nextEpochShardCandidate = b.nextEpochShardCandidate
	newB.nextEpochBeaconCandidate = b.nextEpochBeaconCandidate
	for k, v := range b.shardCommittee {
		newB.shardCommittee[k] = v
	}
	for k, v := range b.shardSubstitute {
		newB.shardSubstitute[k] = v
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

func (b *BeaconCommitteeStateV1) reset() {
	b.beaconCommittee = []incognitokey.CommitteePublicKey{}
	b.beaconSubstitute = []incognitokey.CommitteePublicKey{}
	b.currentEpochShardCandidate = []incognitokey.CommitteePublicKey{}
	b.currentEpochBeaconCandidate = []incognitokey.CommitteePublicKey{}
	b.nextEpochShardCandidate = []incognitokey.CommitteePublicKey{}
	b.nextEpochBeaconCandidate = []incognitokey.CommitteePublicKey{}
	b.shardCommittee = make(map[byte][]incognitokey.CommitteePublicKey)
	b.shardSubstitute = make(map[byte][]incognitokey.CommitteePublicKey)
	b.autoStake = make(map[string]bool)
	b.rewardReceiver = make(map[string]privacy.PaymentAddress)
	b.stakingTx = make(map[string]common.Hash)
}

//Clone :
func (engine *BeaconCommitteeEngineV1) Clone() BeaconCommitteeEngine {
	finalCommitteeState := NewBeaconCommitteeStateV1()
	engine.beaconCommitteeStateV1.clone(finalCommitteeState)
	engine.uncommittedBeaconCommitteeStateV1 = NewBeaconCommitteeStateV1()

	res := NewBeaconCommitteeEngineV1(
		engine.beaconHeight,
		engine.beaconHash,
		finalCommitteeState,
	)

	return res
}

//Version :
func (engine BeaconCommitteeEngineV1) Version() uint {
	return SELF_SWAP_SHARD_VERSION
}

//GetBeaconHeight :
func (engine BeaconCommitteeEngineV1) GetBeaconHeight() uint64 {
	return engine.beaconHeight
}

//GetBeaconHash :
func (engine BeaconCommitteeEngineV1) GetBeaconHash() common.Hash {
	return engine.beaconHash
}

//GetBeaconCommittee :
func (engine BeaconCommitteeEngineV1) GetBeaconCommittee() []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.beaconCommittee
}

//GetBeaconSubstitute :
func (engine BeaconCommitteeEngineV1) GetBeaconSubstitute() []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.beaconSubstitute
}

//GetCandidateShardWaitingForCurrentRandom :
func (engine BeaconCommitteeEngineV1) GetCandidateShardWaitingForCurrentRandom() []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.currentEpochShardCandidate
}

//GetCandidateBeaconWaitingForCurrentRandom :
func (engine BeaconCommitteeEngineV1) GetCandidateBeaconWaitingForCurrentRandom() []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.currentEpochBeaconCandidate
}

//GetCandidateShardWaitingForNextRandom :
func (engine BeaconCommitteeEngineV1) GetCandidateShardWaitingForNextRandom() []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.nextEpochShardCandidate
}

//GetCandidateBeaconWaitingForNextRandom :
func (engine BeaconCommitteeEngineV1) GetCandidateBeaconWaitingForNextRandom() []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.nextEpochBeaconCandidate
}

//GetOneShardCommittee :
func (engine BeaconCommitteeEngineV1) GetOneShardCommittee(shardID byte) []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.shardCommittee[shardID]
}

//GetShardCommittee :
func (engine BeaconCommitteeEngineV1) GetShardCommittee() map[byte][]incognitokey.CommitteePublicKey {
	engine.beaconCommitteeStateV1.mu.RLock()
	defer engine.beaconCommitteeStateV1.mu.RUnlock()
	shardCommittee := make(map[byte][]incognitokey.CommitteePublicKey)
	for k, v := range engine.beaconCommitteeStateV1.shardCommittee {
		shardCommittee[k] = v
	}
	return shardCommittee
}

//GetOneShardSubstitute :
func (engine BeaconCommitteeEngineV1) GetOneShardSubstitute(shardID byte) []incognitokey.CommitteePublicKey {
	return engine.beaconCommitteeStateV1.shardSubstitute[shardID]
}

//GetShardSubstitute :
func (engine BeaconCommitteeEngineV1) GetShardSubstitute() map[byte][]incognitokey.CommitteePublicKey {
	engine.beaconCommitteeStateV1.mu.RLock()
	defer engine.beaconCommitteeStateV1.mu.RUnlock()
	shardSubstitute := make(map[byte][]incognitokey.CommitteePublicKey)
	for k, v := range engine.beaconCommitteeStateV1.shardSubstitute {
		shardSubstitute[k] = v
	}
	return shardSubstitute
}

//GetAutoStaking :
func (engine BeaconCommitteeEngineV1) GetAutoStaking() map[string]bool {
	engine.beaconCommitteeStateV1.mu.RLock()
	defer engine.beaconCommitteeStateV1.mu.RUnlock()
	autoStake := make(map[string]bool)
	for k, v := range engine.beaconCommitteeStateV1.autoStake {
		autoStake[k] = v
	}
	return autoStake
}

func (engine BeaconCommitteeEngineV1) GetRewardReceiver() map[string]privacy.PaymentAddress {
	engine.beaconCommitteeStateV1.mu.RLock()
	defer engine.beaconCommitteeStateV1.mu.RUnlock()
	rewardReceiver := make(map[string]privacy.PaymentAddress)
	for k, v := range engine.beaconCommitteeStateV1.rewardReceiver {
		rewardReceiver[k] = v
	}
	return rewardReceiver
}

func (engine BeaconCommitteeEngineV1) GetStakingTx() map[string]common.Hash {
	engine.beaconCommitteeStateV1.mu.RLock()
	defer engine.beaconCommitteeStateV1.mu.RUnlock()
	stakingTx := make(map[string]common.Hash)
	for k, v := range engine.beaconCommitteeStateV1.stakingTx {
		stakingTx[k] = v
	}
	return stakingTx
}

func (engine *BeaconCommitteeEngineV1) GetAllCandidateSubstituteCommittee() []string {
	engine.beaconCommitteeStateV1.mu.RLock()
	defer engine.beaconCommitteeStateV1.mu.RUnlock()
	return engine.beaconCommitteeStateV1.getAllCandidateSubstituteCommittee()
}

//Commit :
func (engine *BeaconCommitteeEngineV1) Commit(hashes *BeaconCommitteeStateHash) error {
	if reflect.DeepEqual(engine.uncommittedBeaconCommitteeStateV1, NewBeaconCommitteeStateV1()) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, fmt.Errorf("%+v", engine.uncommittedBeaconCommitteeStateV1))
	}
	engine.uncommittedBeaconCommitteeStateV1.mu.Lock()
	defer engine.uncommittedBeaconCommitteeStateV1.mu.Unlock()
	engine.beaconCommitteeStateV1.mu.Lock()
	defer engine.beaconCommitteeStateV1.mu.Unlock()
	comparedHashes, err := engine.generateUncommittedCommitteeHashes()
	if err != nil {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, err)
	}
	if !comparedHashes.BeaconCommitteeAndValidatorHash.IsEqual(&hashes.BeaconCommitteeAndValidatorHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, fmt.Errorf("Uncommitted BeaconCommitteeAndValidatorHash want value %+v but have %+v", comparedHashes.BeaconCommitteeAndValidatorHash, hashes.BeaconCommitteeAndValidatorHash))
	}
	if !comparedHashes.BeaconCandidateHash.IsEqual(&hashes.BeaconCandidateHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, fmt.Errorf("Uncommitted BeaconCandidateHash want value %+v but have %+v", comparedHashes.BeaconCandidateHash, hashes.BeaconCandidateHash))
	}
	if !comparedHashes.ShardCandidateHash.IsEqual(&hashes.ShardCandidateHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, fmt.Errorf("Uncommitted ShardCandidateHash want value %+v but have %+v", comparedHashes.ShardCandidateHash, hashes.ShardCandidateHash))
	}
	if !comparedHashes.ShardCommitteeAndValidatorHash.IsEqual(&hashes.ShardCommitteeAndValidatorHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, fmt.Errorf("Uncommitted ShardCommitteeAndValidatorHash want value %+v but have %+v", comparedHashes.ShardCommitteeAndValidatorHash, hashes.ShardCommitteeAndValidatorHash))
	}
	if !comparedHashes.AutoStakeHash.IsEqual(&hashes.AutoStakeHash) {
		return NewCommitteeStateError(ErrCommitBeaconCommitteeState, fmt.Errorf("Uncommitted AutoStakingHash want value %+v but have %+v", comparedHashes.AutoStakeHash, hashes.AutoStakeHash))
	}
	engine.uncommittedBeaconCommitteeStateV1.clone(engine.beaconCommitteeStateV1)
	engine.uncommittedBeaconCommitteeStateV1.reset()
	return nil
}

//AbortUncommittedBeaconState :
func (engine *BeaconCommitteeEngineV1) AbortUncommittedBeaconState() {
	engine.uncommittedBeaconCommitteeStateV1.mu.Lock()
	defer engine.uncommittedBeaconCommitteeStateV1.mu.Unlock()
	engine.uncommittedBeaconCommitteeStateV1.reset()
}

//InitCommitteeState :
func (engine *BeaconCommitteeEngineV1) InitCommitteeState(env *BeaconCommitteeStateEnvironment) {
	engine.beaconCommitteeStateV1.mu.Lock()
	defer engine.beaconCommitteeStateV1.mu.Unlock()
	b := engine.beaconCommitteeStateV1
	newBeaconCandidates := []incognitokey.CommitteePublicKey{}
	newShardCandidates := []incognitokey.CommitteePublicKey{}
	for _, inst := range env.BeaconInstructions {
		if len(inst) == 0 {
			continue
		}
		if inst[0] == instruction.STAKE_ACTION {
			stakeInstruction := instruction.ImportInitStakeInstructionFromString(inst)
			tempNewBeaconCandidates, tempNewShardCandidates, _ := b.processStakeInstruction(stakeInstruction, env)
			newBeaconCandidates = append(newBeaconCandidates, tempNewBeaconCandidates...)
			newShardCandidates = append(newShardCandidates, tempNewShardCandidates...)
		}
	}
	b.beaconCommittee = append(b.beaconCommittee, newBeaconCandidates...)
	for shardID := 0; shardID < env.ActiveShards; shardID++ {
		b.shardCommittee[byte(shardID)] = append(b.shardCommittee[byte(shardID)], newShardCandidates[shardID*env.MinShardCommitteeSize:(shardID+1)*env.MinShardCommitteeSize]...)
	}
}

//UpdateCommitteeState :
func (engine *BeaconCommitteeEngineV1) UpdateCommitteeState(env *BeaconCommitteeStateEnvironment) (
	*BeaconCommitteeStateHash, *CommitteeChange, [][]string, error) {
	engine.uncommittedBeaconCommitteeStateV1.mu.Lock()
	defer engine.uncommittedBeaconCommitteeStateV1.mu.Unlock()
	engine.beaconCommitteeStateV1.mu.RLock()
	engine.beaconCommitteeStateV1.clone(engine.uncommittedBeaconCommitteeStateV1)
	var err error
	incurredInstructions := [][]string{}
	engine.beaconCommitteeStateV1.mu.RUnlock()
	newB := engine.uncommittedBeaconCommitteeStateV1
	committeeChange := NewCommitteeChange()
	newBeaconCandidates := []incognitokey.CommitteePublicKey{}
	newShardCandidates := []incognitokey.CommitteePublicKey{}
	for _, inst := range env.BeaconInstructions {
		if len(inst) == 0 {
			continue
		}
		tempNewBeaconCandidates, tempNewShardCandidates := []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}
		switch inst[0] {
		case instruction.STAKE_ACTION:
			stakeInstruction, err := instruction.ValidateAndImportStakeInstructionFromString(inst)
			if err != nil {
				Logger.log.Errorf("SKIP stake instruction %+v, error %+v", inst, err)
				continue
			}
			tempNewBeaconCandidates, tempNewShardCandidates, err = newB.processStakeInstruction(stakeInstruction, env)
			if err != nil {
				Logger.log.Errorf("SKIP stake instruction %+v, error %+v", inst, err)
				continue
			}
		case instruction.SWAP_ACTION:
			swapInstruction, err := instruction.ValidateAndImportSwapInstructionFromString(inst)
			if err != nil {
				Logger.log.Errorf("SKIP swap instruction %+v, error %+v", inst, err)
				continue
			}
			tempNewBeaconCandidates, tempNewShardCandidates, err = newB.processSwapInstruction(swapInstruction, env, committeeChange)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
		case instruction.STOP_AUTO_STAKE_ACTION:
			stopAutoStakeInstruction, err := instruction.ValidateAndImportStopAutoStakeInstructionFromString(inst)
			if err != nil {
				Logger.log.Errorf("SKIP stop auto stake instruction %+v, error %+v", inst, err)
			}
			newB.processStopAutoStakeInstruction(stopAutoStakeInstruction, env, committeeChange)
		}
		if len(tempNewBeaconCandidates) > 0 {
			newBeaconCandidates = append(newBeaconCandidates, tempNewBeaconCandidates...)
		}
		if len(tempNewShardCandidates) > 0 {
			newShardCandidates = append(newShardCandidates, tempNewShardCandidates...)
		}
	}
	newB.nextEpochBeaconCandidate = append(newB.nextEpochBeaconCandidate, newBeaconCandidates...)
	committeeChange.NextEpochBeaconCandidateAdded = append(committeeChange.NextEpochBeaconCandidateAdded, newBeaconCandidates...)
	newB.nextEpochShardCandidate = append(newB.nextEpochShardCandidate, newShardCandidates...)
	committeeChange.NextEpochShardCandidateAdded = append(committeeChange.NextEpochShardCandidateAdded, newShardCandidates...)
	if env.IsBeaconRandomTime {
		committeeChange.CurrentEpochShardCandidateAdded = newB.nextEpochShardCandidate
		newB.currentEpochShardCandidate = newB.nextEpochShardCandidate
		committeeChange.CurrentEpochBeaconCandidateAdded = newB.nextEpochBeaconCandidate
		newB.currentEpochBeaconCandidate = newB.nextEpochBeaconCandidate
		Logger.log.Debug("Beacon Process: CandidateShardWaitingForCurrentRandom: ", newB.currentEpochShardCandidate)
		Logger.log.Debug("Beacon Process: CandidateBeaconWaitingForCurrentRandom: ", newB.currentEpochBeaconCandidate)
		// reset candidate list
		committeeChange.NextEpochShardCandidateRemoved = newB.nextEpochShardCandidate
		newB.nextEpochShardCandidate = []incognitokey.CommitteePublicKey{}
		committeeChange.NextEpochBeaconCandidateRemoved = newB.nextEpochBeaconCandidate
		newB.nextEpochBeaconCandidate = []incognitokey.CommitteePublicKey{}
	}
	if env.IsFoundRandomNumber {
		numberOfShardSubstitutes := make(map[byte]int)
		for shardID, shardSubstitute := range newB.shardSubstitute {
			numberOfShardSubstitutes[shardID] = len(shardSubstitute)
		}
		shardCandidatesStr, err := incognitokey.CommitteeKeyListToString(newB.currentEpochShardCandidate)
		if err != nil {
			return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
		}
		remainShardCandidatesStr, assignedCandidates := assignShardCandidate(shardCandidatesStr, numberOfShardSubstitutes, env.RandomNumber, env.AssignOffset, env.ActiveShards)
		remainShardCandidates, err := incognitokey.CommitteeBase58KeyListToStruct(remainShardCandidatesStr)
		if err != nil {
			return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
		}
		committeeChange.NextEpochShardCandidateAdded = append(committeeChange.NextEpochShardCandidateAdded, remainShardCandidates...)
		// append remain candidate into shard waiting for next random list
		newB.nextEpochShardCandidate = append(newB.nextEpochShardCandidate, remainShardCandidates...)
		// assign candidate into shard pending validator list
		for shardID, candidateListStr := range assignedCandidates {
			candidateList, err := incognitokey.CommitteeBase58KeyListToStruct(candidateListStr)
			if err != nil {
				return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
			}
			committeeChange.ShardSubstituteAdded[shardID] = candidateList
			newB.shardSubstitute[shardID] = append(newB.shardSubstitute[shardID], candidateList...)
		}
		committeeChange.CurrentEpochShardCandidateRemoved = newB.currentEpochShardCandidate
		// delete CandidateShardWaitingForCurrentRandom list
		newB.currentEpochShardCandidate = []incognitokey.CommitteePublicKey{}
		// shuffle CandidateBeaconWaitingForCurrentRandom with current random number
		newBeaconSubstitute, err := ShuffleCandidate(newB.currentEpochBeaconCandidate, env.RandomNumber)
		if err != nil {
			return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
		}
		committeeChange.CurrentEpochBeaconCandidateRemoved = newB.currentEpochBeaconCandidate
		newB.currentEpochBeaconCandidate = []incognitokey.CommitteePublicKey{}
		committeeChange.BeaconSubstituteAdded = newBeaconSubstitute
		newB.beaconSubstitute = append(newB.beaconSubstitute, newBeaconSubstitute...)
	}

	committeeChange, err = newB.processUnstakeChange(committeeChange, env)
	if err != nil {
		return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
	}

	err = newB.processAutoStakingChange(committeeChange, env)
	if err != nil {
		return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
	}
	hashes, err := engine.generateUncommittedCommitteeHashes()
	if err != nil {
		return nil, nil, nil, NewCommitteeStateError(ErrUpdateCommitteeState, err)
	}
	return hashes, committeeChange, incurredInstructions, nil
}

func (engine *BeaconCommitteeEngineV1) GenerateAssignInstruction(rand int64, assignOffset int, activeShards int) ([]*instruction.AssignInstruction, []string, map[byte][]string) {
	candidates, _ := incognitokey.CommitteeKeyListToString(engine.beaconCommitteeStateV1.currentEpochShardCandidate)
	numberOfPendingValidator := make(map[byte]int)
	shardPendingValidator := engine.beaconCommitteeStateV1.shardSubstitute
	for i := 0; i < activeShards; i++ {
		if pendingValidators, ok := shardPendingValidator[byte(i)]; ok {
			numberOfPendingValidator[byte(i)] = len(pendingValidators)
		} else {
			numberOfPendingValidator[byte(i)] = 0
		}
	}
	assignedCandidates := make(map[byte][]string)
	remainShardCandidates := []string{}
	shuffledCandidate := shuffleShardCandidate(candidates, rand)
	for _, candidate := range shuffledCandidate {
		shardID := calculateCandidateShardID(candidate, rand, activeShards)
		if numberOfPendingValidator[shardID]+1 > assignOffset {
			remainShardCandidates = append(remainShardCandidates, candidate)
			continue
		} else {
			assignedCandidates[shardID] = append(assignedCandidates[shardID], candidate)
			numberOfPendingValidator[shardID] += 1
		}
	}
	var keys []int
	for k := range assignedCandidates {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	instructions := []*instruction.AssignInstruction{}
	for _, key := range keys {
		shardID := byte(key)
		candidates := assignedCandidates[shardID]
		Logger.log.Infof("Assign Candidate at Shard %+v: %+v", shardID, candidates)
		shardAssignInstruction := instruction.NewAssignInstructionWithValue(int(shardID), candidates)
		instructions = append(instructions, shardAssignInstruction)
	}
	return instructions, remainShardCandidates, assignedCandidates
}

// GenerateAllSwapShardInstructions do nothing
func (b *BeaconCommitteeEngineV1) GenerateAllSwapShardInstructions(env *BeaconCommitteeStateEnvironment) (
	[]*instruction.SwapShardInstruction, error) {
	return []*instruction.SwapShardInstruction{}, nil
}

//BuildIncurredInstructions : BuildIncurredInstruction from instructions
func (engine BeaconCommitteeEngineV1) BuildIncurredInstructions(
	env *BeaconCommitteeStateEnvironment) (
	[][]string, error) {
	incurredInstructions := [][]string{}
	return incurredInstructions, nil
}

func (b *BeaconCommitteeStateV1) processStakeInstruction(
	stakeInstruction *instruction.StakeInstruction,
	env *BeaconCommitteeStateEnvironment,
) ([]incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, error) {
	newBeaconCandidates := []incognitokey.CommitteePublicKey{}
	newShardCandidates := []incognitokey.CommitteePublicKey{}
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
	err := statedb.StoreStakerInfoV1(
		env.ConsensusStateDB,
		stakeInstruction.PublicKeyStructs,
		b.rewardReceiver,
		b.autoStake,
		b.stakingTx,
	)
	if err != nil {
		return newBeaconCandidates, newShardCandidates, err
	}
	return newBeaconCandidates, newShardCandidates, nil
}

func (b *BeaconCommitteeStateV1) processStopAutoStakeInstruction(
	stopAutoStakeInstruction *instruction.StopAutoStakeInstruction,
	env *BeaconCommitteeStateEnvironment,
	committeeChange *CommitteeChange,
) {
	for _, committeePublicKey := range stopAutoStakeInstruction.CommitteePublicKeys {
		if common.IndexOfStr(committeePublicKey, b.getAllCandidateSubstituteCommittee()) == -1 {
			// if not found then delete auto staking data for this public key if present
			if _, ok := b.autoStake[committeePublicKey]; ok {
				delete(b.autoStake, committeePublicKey)
			}
		} else {
			// if found in committee list then turn off auto staking
			if _, ok := b.autoStake[committeePublicKey]; ok {
				b.autoStake[committeePublicKey] = false
				committeeChange.StopAutoStake = append(committeeChange.StopAutoStake, committeePublicKey)
			}
		}
	}
}

func (b *BeaconCommitteeStateV1) processSwapInstruction(
	swapInstruction *instruction.SwapInstruction,
	env *BeaconCommitteeStateEnvironment,
	committeeChange *CommitteeChange,
) ([]incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, error) {
	newBeaconCandidates := []incognitokey.CommitteePublicKey{}
	newShardCandidates := []incognitokey.CommitteePublicKey{}
	if common.IndexOfUint64(env.BeaconHeight/env.ParamEpoch, env.EpochBreakPointSwapNewKey) > -1 || swapInstruction.IsReplace {
		err := b.processReplaceInstruction(swapInstruction, committeeChange, env)
		if err != nil {
			return newBeaconCandidates, newShardCandidates, err
		}
	} else {
		Logger.log.Debug("Swap Instruction In Public Keys", swapInstruction.InPublicKeys)
		Logger.log.Debug("Swap Instruction Out Public Keys", swapInstruction.OutPublicKeys)
		if swapInstruction.ChainID != instruction.BEACON_CHAIN_ID {
			shardID := byte(swapInstruction.ChainID)
			// delete in public key out of sharding pending validator list
			if len(swapInstruction.InPublicKeys) > 0 {
				shardSubstituteStr, err := incognitokey.CommitteeKeyListToString(b.shardSubstitute[shardID])
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				tempShardSubstitute, err := removeValidatorV1(shardSubstituteStr, swapInstruction.InPublicKeys)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				// update shard pending validator
				committeeChange.ShardSubstituteRemoved[shardID] = append(committeeChange.ShardSubstituteRemoved[shardID], swapInstruction.InPublicKeyStructs...)
				b.shardSubstitute[shardID], err = incognitokey.CommitteeBase58KeyListToStruct(tempShardSubstitute)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				// add new public key to committees
				committeeChange.ShardCommitteeAdded[shardID] = append(committeeChange.ShardCommitteeAdded[shardID], swapInstruction.InPublicKeyStructs...)
				b.shardCommittee[shardID] = append(b.shardCommittee[shardID], swapInstruction.InPublicKeyStructs...)
			}
			// delete out public key out of current committees
			if len(swapInstruction.OutPublicKeys) > 0 {
				//for _, value := range outPublickeyStructs {
				//	delete(b,cue.GetIncKeyBase58(
				//}
				shardCommitteeStr, err := incognitokey.CommitteeKeyListToString(b.shardCommittee[shardID])
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				tempShardCommittees, err := removeValidatorV1(shardCommitteeStr, swapInstruction.OutPublicKeys)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				// remove old public key in shard committee update shard committee
				committeeChange.ShardCommitteeRemoved[shardID] = append(committeeChange.ShardCommitteeRemoved[shardID], swapInstruction.OutPublicKeyStructs...)
				b.shardCommittee[shardID], err = incognitokey.CommitteeBase58KeyListToStruct(tempShardCommittees)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				// Check auto stake in out public keys list
				// if auto staking not found or flag auto stake is false then do not re-stake for this out public key
				// if auto staking flag is true then system will automatically add this out public key to current candidate list
				for index, outPublicKey := range swapInstruction.OutPublicKeys {
					stakerInfo, has, err := statedb.GetStakerInfo(env.ConsensusStateDB, outPublicKey)
					if err != nil {
						panic(err)
					}
					if !has {
						panic(errors.Errorf("Can not found info of this public key %v", outPublicKey))
					}
					if stakerInfo.AutoStaking() {
						newShardCandidates = append(newShardCandidates, swapInstruction.OutPublicKeyStructs[index])
					} else {
						delete(b.rewardReceiver, swapInstruction.OutPublicKeyStructs[index].GetIncKeyBase58())
						delete(b.autoStake, outPublicKey)
					}
				}
			}
		} else {
			if len(swapInstruction.InPublicKeys) > 0 {
				beaconSubstituteStr, err := incognitokey.CommitteeKeyListToString(b.beaconSubstitute)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				tempBeaconSubstitute, err := removeValidatorV1(beaconSubstituteStr, swapInstruction.InPublicKeys)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				// update beacon pending validator
				committeeChange.BeaconSubstituteRemoved = append(committeeChange.BeaconSubstituteRemoved, swapInstruction.InPublicKeyStructs...)
				b.beaconSubstitute, err = incognitokey.CommitteeBase58KeyListToStruct(tempBeaconSubstitute)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				// add new public key to beacon committee
				committeeChange.BeaconCommitteeAdded = append(committeeChange.BeaconCommitteeAdded, swapInstruction.InPublicKeyStructs...)
				b.beaconCommittee = append(b.beaconCommittee, swapInstruction.InPublicKeyStructs...)
			}
			if len(swapInstruction.OutPublicKeys) > 0 {
				// delete reward receiver
				//for _, value := range swapInstruction.OutPublicKeyStructs {
				//	delete(b,cue.GetIncKeyBase58(
				//}
				beaconCommitteeStrs, err := incognitokey.CommitteeKeyListToString(b.beaconCommittee)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				tempBeaconCommittees, err := removeValidatorV1(beaconCommitteeStrs, swapInstruction.OutPublicKeys)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				// remove old public key in beacon committee and update beacon best state
				committeeChange.BeaconCommitteeRemoved = append(committeeChange.BeaconCommitteeRemoved, swapInstruction.OutPublicKeyStructs...)
				b.beaconCommittee, err = incognitokey.CommitteeBase58KeyListToStruct(tempBeaconCommittees)
				if err != nil {
					return newBeaconCandidates, newShardCandidates, err
				}
				for index, outPublicKey := range swapInstruction.OutPublicKeys {
					stakerInfo, has, err := statedb.GetStakerInfo(env.ConsensusStateDB, outPublicKey)
					if err != nil {
						panic(err)
					}
					if !has {
						panic(errors.Errorf("Can not found info of this public key %v", outPublicKey))
					}
					if stakerInfo.AutoStaking() {
						newBeaconCandidates = append(newBeaconCandidates, swapInstruction.OutPublicKeyStructs[index])
					} else {
						delete(b.rewardReceiver, swapInstruction.OutPublicKeyStructs[index].GetIncKeyBase58())
						delete(b.autoStake, outPublicKey)
					}
				}
			}
		}
	}
	return newBeaconCandidates, newShardCandidates, nil
}

func (b *BeaconCommitteeStateV1) processReplaceInstruction(
	swapInstruction *instruction.SwapInstruction,
	committeeChange *CommitteeChange,
	env *BeaconCommitteeStateEnvironment,
) error {
	removedCommittee := len(swapInstruction.InPublicKeys)
	if swapInstruction.ChainID == instruction.BEACON_CHAIN_ID {
		committeeChange.BeaconCommitteeReplaced[common.REPLACE_OUT] = append(committeeChange.BeaconCommitteeReplaced[common.REPLACE_OUT], swapInstruction.OutPublicKeyStructs...)
		// add new public key to committees
		committeeChange.BeaconCommitteeReplaced[common.REPLACE_IN] = append(committeeChange.BeaconCommitteeReplaced[common.REPLACE_IN], swapInstruction.InPublicKeyStructs...)
		remainedBeaconCommittees := b.beaconCommittee[removedCommittee:]
		b.beaconCommittee = append(swapInstruction.InPublicKeyStructs, remainedBeaconCommittees...)
	} else {
		shardID := byte(swapInstruction.ChainID)
		committeeReplace := [2][]incognitokey.CommitteePublicKey{}
		// update shard COMMITTEE
		committeeReplace[common.REPLACE_OUT] = append(committeeReplace[common.REPLACE_OUT], swapInstruction.OutPublicKeyStructs...)
		// add new public key to committees
		committeeReplace[common.REPLACE_IN] = append(committeeReplace[common.REPLACE_IN], swapInstruction.InPublicKeyStructs...)
		committeeChange.ShardCommitteeReplaced[shardID] = committeeReplace
		remainedShardCommittees := b.shardCommittee[shardID][removedCommittee:]
		b.shardCommittee[shardID] = append(swapInstruction.InPublicKeyStructs, remainedShardCommittees...)
	}
	for index := 0; index < removedCommittee; index++ {
		delete(b.autoStake, swapInstruction.OutPublicKeys[index])
		delete(b.rewardReceiver, swapInstruction.OutPublicKeyStructs[index].GetIncKeyBase58())
		b.autoStake[swapInstruction.InPublicKeys[index]] = false
		b.rewardReceiver[swapInstruction.InPublicKeyStructs[index].GetIncKeyBase58()] = swapInstruction.NewRewardReceiverStructs[index]
		b.stakingTx[swapInstruction.InPublicKeys[index]] = common.HashH([]byte{0})
	}
	err := statedb.StoreStakerInfoV1(
		env.ConsensusStateDB,
		swapInstruction.InPublicKeyStructs,
		b.rewardReceiver,
		b.autoStake,
		b.stakingTx,
	)
	return err
}

func (engine BeaconCommitteeEngineV1) generateUncommittedCommitteeHashes() (*BeaconCommitteeStateHash, error) {
	if reflect.DeepEqual(engine.uncommittedBeaconCommitteeStateV1, NewBeaconCommitteeStateV1()) {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, empty uncommitted state")
	}
	newB := engine.uncommittedBeaconCommitteeStateV1
	// beacon committee
	beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(newB.beaconCommittee)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	validatorArr := append([]string{}, beaconCommitteeStr...)

	beaconPendingValidatorStr, err := incognitokey.CommitteeKeyListToString(newB.beaconSubstitute)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	validatorArr = append(validatorArr, beaconPendingValidatorStr...)
	tempBeaconCommitteeAndValidatorHash, err := common.GenerateHashFromStringArray(validatorArr)
	// beacon candidate: current candidate + next candidate
	// BeaconCandidate root: beacon current candidate + beacon next candidate
	beaconCandidateArr := append(newB.currentEpochBeaconCandidate, newB.nextEpochBeaconCandidate...)
	beaconCandidateArrStr, err := incognitokey.CommitteeKeyListToString(beaconCandidateArr)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	tempBeaconCandidateHash, err := common.GenerateHashFromStringArray(beaconCandidateArrStr)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	// Shard candidate root: shard current candidate + shard next candidate
	shardCandidateArr := append(newB.currentEpochShardCandidate, newB.nextEpochShardCandidate...)
	shardCandidateArrStr, err := incognitokey.CommitteeKeyListToString(shardCandidateArr)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}
	tempShardCandidateHash, err := common.GenerateHashFromStringArray(shardCandidateArrStr)
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
		BeaconCandidateHash:             tempBeaconCandidateHash,
		ShardCandidateHash:              tempShardCandidateHash,
		ShardCommitteeAndValidatorHash:  tempShardCommitteeAndValidatorHash,
		AutoStakeHash:                   tempAutoStakingHash,
	}
	return hashes, nil
}

func (b *BeaconCommitteeStateV1) getAllCandidateSubstituteCommittee() []string {
	res := []string{}
	for _, committee := range b.shardCommittee {
		committeeStr, err := incognitokey.CommitteeKeyListToString(committee)
		if err != nil {
			panic(err)
		}
		res = append(res, committeeStr...)
	}
	for _, substitute := range b.shardSubstitute {
		substituteStr, err := incognitokey.CommitteeKeyListToString(substitute)
		if err != nil {
			panic(err)
		}
		res = append(res, substituteStr...)
	}
	beaconCommittee := b.beaconCommittee
	beaconCommitteeStr, err := incognitokey.CommitteeKeyListToString(beaconCommittee)
	if err != nil {
		panic(err)
	}
	res = append(res, beaconCommitteeStr...)
	beaconSubstitute := b.beaconSubstitute
	beaconSubstituteStr, err := incognitokey.CommitteeKeyListToString(beaconSubstitute)
	if err != nil {
		panic(err)
	}
	res = append(res, beaconSubstituteStr...)
	candidateBeaconWaitingForCurrentRandom := b.currentEpochBeaconCandidate
	candidateBeaconWaitingForCurrentRandomStr, err := incognitokey.CommitteeKeyListToString(candidateBeaconWaitingForCurrentRandom)
	if err != nil {
		panic(err)
	}
	res = append(res, candidateBeaconWaitingForCurrentRandomStr...)
	candidateBeaconWaitingForNextRandom := b.nextEpochBeaconCandidate
	candidateBeaconWaitingForNextRandomStr, err := incognitokey.CommitteeKeyListToString(candidateBeaconWaitingForNextRandom)
	if err != nil {
		panic(err)
	}
	res = append(res, candidateBeaconWaitingForNextRandomStr...)
	candidateShardWaitingForCurrentRandom := b.currentEpochShardCandidate
	candidateShardWaitingForCurrentRandomStr, err := incognitokey.CommitteeKeyListToString(candidateShardWaitingForCurrentRandom)
	if err != nil {
		panic(err)
	}
	res = append(res, candidateShardWaitingForCurrentRandomStr...)
	candidateShardWaitingForNextRandom := b.nextEpochShardCandidate
	candidateShardWaitingForNextRandomStr, err := incognitokey.CommitteeKeyListToString(candidateShardWaitingForNextRandom)
	if err != nil {
		panic(err)
	}
	res = append(res, candidateShardWaitingForNextRandomStr...)
	return res
}

func (b *BeaconCommitteeStateV1) processAutoStakingChange(committeeChange *CommitteeChange, env *BeaconCommitteeStateEnvironment) error {
	stopAutoStakingIncognitoKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeeChange.StopAutoStake)
	if err != nil {
		return err
	}
	err = statedb.StoreStakerInfoV1(
		env.ConsensusStateDB,
		stopAutoStakingIncognitoKey,
		b.rewardReceiver,
		b.autoStake,
		b.stakingTx,
	)
	return nil
}

func (b *BeaconCommitteeStateV1) processUnstakeChange(committeeChange *CommitteeChange, env *BeaconCommitteeStateEnvironment) (*CommitteeChange, error) {

	newCommitteeChange := committeeChange

	unstakingIncognitoKey, err := incognitokey.CommitteeBase58KeyListToStruct(newCommitteeChange.Unstake)
	if err != nil {
		return newCommitteeChange, err
	}
	err = statedb.StoreStakerInfoV1(
		env.ConsensusStateDB,
		unstakingIncognitoKey,
		b.rewardReceiver,
		b.autoStake,
		b.stakingTx,
	)

	return newCommitteeChange, err
}

//HasSwappedCommittees ...
func (engine *BeaconCommitteeEngineV1) HasSwappedCommittees(env *BeaconCommitteeStateEnvironment) (bool, error) {
	return false, nil
}

//ActiveShards ...
func (engine *BeaconCommitteeEngineV1) ActiveShards() int {
	return len(engine.beaconCommitteeStateV1.shardCommittee)
}
