package committeestate

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/instruction"
)

//ShardCommitteeStateHash
type ShardCommitteeStateHashV2 struct {
	ShardCommitteeHash  common.Hash
	ShardSubstituteHash common.Hash
}

//ShardCommitteeStateV2
type ShardCommitteeStateV2 struct {
	shardCommittee     []incognitokey.CommitteePublicKey
	committeeFromBlock common.Hash //Committees From Beacon Block Hash

	mu *sync.RWMutex
}

//ShardCommitteeEngineV2
type ShardCommitteeEngineV2 struct {
	shardHeight                      uint64
	shardHash                        common.Hash
	shardID                          byte
	shardCommitteeStateV2            *ShardCommitteeStateV2
	uncommittedShardCommitteeStateV2 *ShardCommitteeStateV2
}

//NewShardCommitteeStateV2 is default constructor for ShardCommitteeStateV2 ...
//Output: pointer of ShardCommitteeStateV2 struct
func NewShardCommitteeStateV2() *ShardCommitteeStateV2 {
	return &ShardCommitteeStateV2{
		mu: new(sync.RWMutex),
	}
}

//NewShardCommitteeStateV2WithValue is constructor for ShardCommitteeStateV2 with value
//Output: pointer of ShardCommitteeStateV2 struct with value
func NewShardCommitteeStateV2WithValue(
	shardCommittee []incognitokey.CommitteePublicKey,
	committeeFromBlockHash common.Hash,
) *ShardCommitteeStateV2 {
	return &ShardCommitteeStateV2{
		shardCommittee:     incognitokey.DeepCopy(shardCommittee),
		committeeFromBlock: committeeFromBlockHash,
		mu:                 new(sync.RWMutex),
	}
}

//NewShardCommitteeEngineV1 is default constructor for ShardCommitteeEngineV2
//Output: pointer of ShardCommitteeEngineV2
func NewShardCommitteeEngineV2(shardHeight uint64,
	shardHash common.Hash, shardID byte, shardCommitteeStateV2 *ShardCommitteeStateV2) *ShardCommitteeEngineV2 {
	return &ShardCommitteeEngineV2{
		shardHeight:                      shardHeight,
		shardHash:                        shardHash,
		shardID:                          shardID,
		shardCommitteeStateV2:            shardCommitteeStateV2,
		uncommittedShardCommitteeStateV2: NewShardCommitteeStateV2(),
	}
}

//Clone ...
func (engine *ShardCommitteeEngineV2) Clone() ShardCommitteeEngine {
	finalCommitteeState := NewShardCommitteeStateV2()
	engine.shardCommitteeStateV2.clone(finalCommitteeState)
	engine.uncommittedShardCommitteeStateV2 = NewShardCommitteeStateV2()

	res := NewShardCommitteeEngineV2(
		engine.shardHeight,
		engine.shardHash,
		engine.shardID,
		finalCommitteeState,
	)
	return res
}

//clone ShardCommitteeStateV2 to new instance
func (s ShardCommitteeStateV2) clone(newCommitteeState *ShardCommitteeStateV2) {
	newCommitteeState.reset()
	newCommitteeState.shardCommittee = incognitokey.DeepCopy(s.shardCommittee)
	newCommitteeState.committeeFromBlock = s.committeeFromBlock
}

//reset : reset ShardCommitteeStateV2 to default value
func (s *ShardCommitteeStateV2) reset() {
	s.shardCommittee = make([]incognitokey.CommitteePublicKey, 0)
	s.committeeFromBlock = common.Hash{}
}

//Version ...
func (engine *ShardCommitteeEngineV2) Version() uint {
	return SLASHING_VERSION
}

//GetShardCommittee get shard committees
func (engine *ShardCommitteeEngineV2) GetShardCommittee() []incognitokey.CommitteePublicKey {
	return incognitokey.DeepCopy(engine.shardCommitteeStateV2.shardCommittee)
}

//GetShardSubstitute get shard pending validators
func (engine *ShardCommitteeEngineV2) GetShardSubstitute() []incognitokey.CommitteePublicKey {
	return []incognitokey.CommitteePublicKey{}
}

func (engine *ShardCommitteeEngineV2) CommitteeFromBlock() common.Hash {
	return engine.shardCommitteeStateV2.committeeFromBlock
}

//Commit commit committee state change in uncommittedShardCommitteeStateV2 struct
//	- Generate hash from uncommiteed
//	- Check validations of input hash
//	- clone uncommitted to commit
//	- reset uncommitted
func (engine *ShardCommitteeEngineV2) Commit(hashes *ShardCommitteeStateHash) error {
	if reflect.DeepEqual(engine.uncommittedShardCommitteeStateV2, NewShardCommitteeStateV2()) {
		return NewCommitteeStateError(ErrCommitShardCommitteeState, fmt.Errorf("%+v", engine.uncommittedShardCommitteeStateV2))
	}
	engine.uncommittedShardCommitteeStateV2.mu.Lock()
	defer engine.uncommittedShardCommitteeStateV2.mu.Unlock()
	engine.shardCommitteeStateV2.mu.Lock()
	defer engine.shardCommitteeStateV2.mu.Unlock()
	comparedHashes, err := engine.generateUncommittedCommitteeHashes()
	if err != nil {
		return NewCommitteeStateError(ErrCommitShardCommitteeState, err)
	}

	if !comparedHashes.ShardCommitteeHash.IsEqual(&hashes.ShardCommitteeHash) {
		return NewCommitteeStateError(ErrCommitShardCommitteeState, fmt.Errorf("Uncommitted ShardCommitteeHash want value %+v but have %+v",
			comparedHashes.ShardCommitteeHash, hashes.ShardCommitteeHash))
	}

	if !comparedHashes.ShardSubstituteHash.IsEqual(&hashes.ShardSubstituteHash) {
		return NewCommitteeStateError(ErrCommitShardCommitteeState, fmt.Errorf("Uncommitted ShardSubstituteHash want value %+v but have %+v",
			comparedHashes.ShardSubstituteHash, hashes.ShardSubstituteHash))
	}

	engine.uncommittedShardCommitteeStateV2.clone(engine.shardCommitteeStateV2)
	engine.uncommittedShardCommitteeStateV2.reset()
	return nil
}

//AbortUncommittedShardState reset data in uncommittedShardCommitteeStateV2 struct
func (engine *ShardCommitteeEngineV2) AbortUncommittedShardState() {
	engine.uncommittedShardCommitteeStateV2.mu.Lock()
	defer engine.uncommittedShardCommitteeStateV2.mu.Unlock()
	engine.uncommittedShardCommitteeStateV2.reset()
}

//InitCommitteeState init committee state at genesis block or anytime restore program
//	- call function processInstructionFromBeacon for process instructions received from beacon
//	- call function processShardBlockInstruction for process shard block instructions
func (engine *ShardCommitteeEngineV2) InitCommitteeState(env ShardCommitteeStateEnvironment) {
	engine.shardCommitteeStateV2.mu.Lock()
	defer engine.shardCommitteeStateV2.mu.Unlock()

	committeeChange := NewCommitteeChange()
	candidates := []string{}

	for _, beaconInstruction := range env.BeaconInstructions() {
		if beaconInstruction[0] == instruction.STAKE_ACTION {
			candidates = strings.Split(beaconInstruction[1], ",")
		}
	}

	newShardCandidateStructs := []incognitokey.CommitteePublicKey{}
	for _, candidate := range candidates {
		key := incognitokey.CommitteePublicKey{}
		err := key.FromBase58(candidate)
		if err != nil {
			panic(err)
		}
		newShardCandidateStructs = append(newShardCandidateStructs, key)
	}

	addedCommittees := []incognitokey.CommitteePublicKey{}
	addedCommittees = append(addedCommittees, newShardCandidateStructs[int(env.ShardID())*
		env.MinShardCommitteeSize():(int(env.ShardID())*env.MinShardCommitteeSize())+env.MinShardCommitteeSize()]...)

	engine.shardCommitteeStateV2.shardCommittee = incognitokey.DeepCopy(addedCommittees)
	committeeChange.ShardCommitteeAdded[env.ShardID()] = addedCommittees

}

//UpdateCommitteeState update committeState from valid data before
//	- call process instructions from beacon
//	- check conditions for epoch timestamp
//		+ process shard block instructions for key
//			+ process shard block instructions normally
//	- hash for checking commit later
//	- Only call once in new or insert block process
func (engine *ShardCommitteeEngineV2) UpdateCommitteeState(
	env ShardCommitteeStateEnvironment) (*ShardCommitteeStateHash, *CommitteeChange, error) {
	engine.uncommittedShardCommitteeStateV2.mu.Lock()
	defer engine.uncommittedShardCommitteeStateV2.mu.Unlock()
	engine.shardCommitteeStateV2.mu.RLock()
	engine.shardCommitteeStateV2.clone(engine.uncommittedShardCommitteeStateV2)
	engine.shardCommitteeStateV2.mu.RUnlock()

	newCommitteeState := engine.uncommittedShardCommitteeStateV2
	committeeChange, err := newCommitteeState.forceUpdateCommitteesFromBeacon(env, NewCommitteeChange())
	if err != nil {
		return nil, NewCommitteeChange(), NewCommitteeStateError(ErrUpdateCommitteeState, err)
	}
	hashes, err := engine.generateUncommittedCommitteeHashes()
	if err != nil {
		return nil, NewCommitteeChange(), NewCommitteeStateError(ErrUpdateCommitteeState, err)
	}

	res, _ := incognitokey.CommitteeKeyListToString(env.CommitteesFromBeaconView())
	Logger.log.Infof(">>>>>>>> \n "+
		"Height %+v, Committee From Block %+v \n"+
		"Committees %+v", env.ShardHeight(), env.CommitteesFromBlock(), res)

	return hashes, committeeChange, nil
}

func getNewShardCommittees(
	shardCommittees []string,
) ([]string, error) {
	return shardCommittees, nil
}
func (engine *ShardCommitteeEngineV2) GenerateSwapInstruction(env ShardCommitteeStateEnvironment) (*instruction.SwapInstruction, []string, []string, error) {
	shardCommittees, _ := incognitokey.CommitteeKeyListToString(engine.shardCommitteeStateV2.shardCommittee)
	return instruction.NewSwapInstruction(), []string{}, shardCommittees, nil
}

// processSwapShardInstruction: process swap shard instruction
func (s *ShardCommitteeStateV2) processSwapShardInstruction(
	swapShardInstruction *instruction.SwapShardInstruction,
	env ShardCommitteeStateEnvironment,
	committeeChange *CommitteeChange) (*CommitteeChange, error) {

	newCommitteeChange := committeeChange
	chainID := byte(swapShardInstruction.ChainID)
	tempSwapOutPublicKeys := swapShardInstruction.OutPublicKeyStructs
	tempSwapInPublicKeys := swapShardInstruction.InPublicKeyStructs
	numberFixedValidators := env.NumberOfFixedBlockValidators()

	// process list shard committees
	for _, v := range tempSwapOutPublicKeys {
		s.shardCommittee = append(s.shardCommittee[:numberFixedValidators], s.shardCommittee[numberFixedValidators+1:]...)
		newCommitteeChange.ShardCommitteeRemoved[chainID] = append(newCommitteeChange.ShardCommitteeRemoved[chainID], v)
	}
	s.shardCommittee = append(s.shardCommittee, tempSwapInPublicKeys...)
	newCommitteeChange.ShardCommitteeAdded[chainID] = append(newCommitteeChange.ShardCommitteeAdded[chainID], tempSwapInPublicKeys...)

	return newCommitteeChange, nil
}

func (s *ShardCommitteeStateV2) forceUpdateCommitteesFromBeacon(
	env ShardCommitteeStateEnvironment,
	committeeChange *CommitteeChange) (*CommitteeChange, error) {

	newCommitteeChange := committeeChange

	s.shardCommittee = incognitokey.DeepCopy(env.CommitteesFromBeaconView())
	s.committeeFromBlock = env.CommitteesFromBlock()
	return newCommitteeChange, nil
}

//ProcessInstructionFromBeacon : process instrucction from beacon
func (engine *ShardCommitteeEngineV2) ProcessInstructionFromBeacon(
	env ShardCommitteeStateEnvironment) (*CommitteeChange, error) {
	return NewCommitteeChange(), nil
}

//generateUncommittedCommitteeHashes generate hashes relate to uncommitted committees of struct ShardCommitteeEngineV2
//	append committees and subtitutes to struct and hash it
func (engine ShardCommitteeEngineV2) generateUncommittedCommitteeHashes() (*ShardCommitteeStateHash, error) {
	if reflect.DeepEqual(engine.uncommittedShardCommitteeStateV2, NewBeaconCommitteeStateV1()) {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, empty uncommitted state")
	}

	newCommitteeState := engine.uncommittedShardCommitteeStateV2

	committeesStr, err := incognitokey.CommitteeKeyListToString(newCommitteeState.shardCommittee)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}

	committeeHash, err := common.GenerateHashFromStringArray(committeesStr)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}

	substitutesStr, err := incognitokey.CommitteeKeyListToString([]incognitokey.CommitteePublicKey{})
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}

	substituteHash, err := common.GenerateHashFromStringArray(substitutesStr)
	if err != nil {
		return nil, fmt.Errorf("Generate Uncommitted Root Hash, error %+v", err)
	}

	return &ShardCommitteeStateHash{
		ShardCommitteeHash:  committeeHash,
		ShardSubstituteHash: substituteHash,
		CommitteeFromBlock:  newCommitteeState.committeeFromBlock,
	}, nil
}
