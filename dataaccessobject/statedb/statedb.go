package statedb

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/trie"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
	"time"
)

// StateDBs within the incognito protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * State Object
type StateDB struct {
	db   DatabaseAccessWarper
	trie Trie
	//rawdb incdb.Database
	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects        map[common.Hash]StateObject
	stateObjectsPending map[common.Hash]struct{} // State objects finalized but not yet written to the trie
	stateObjectsDirty   map[common.Hash]struct{} // State objects modified in the current execution

	// DB error.
	// State objects are used by the consensus core which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Measurements gathered during execution for debugging purposes
	StateObjectReads   time.Duration
	StateObjectHashes  time.Duration
	StateObjectUpdates time.Duration
	StateObjectCommits time.Duration
}

//
//// New return a new statedb attach with a state root
//func New(root common.Hash, db DatabaseAccessWarper) (*StateDB, error) {
//	tr, err := db.OpenTrie(root)
//	if err != nil {
//		return nil, err
//	}
//	return &StateDB{
//		db:                  db,
//		trie:                tr,
//		stateObjects:        make(map[common.Hash]StateObject),
//		stateObjectsPending: make(map[common.Hash]struct{}),
//		stateObjectsDirty:   make(map[common.Hash]struct{}),
//	}, nil
//}
//
//// New return a new statedb attach with a state root
//func NewWithRawDB(root common.Hash, db DatabaseAccessWarper, rawdb incdb.Database) (*StateDB, error) {
//	tr, err := db.OpenTrie(root)
//	if err != nil {
//		return nil, err
//	}
//	return &StateDB{
//		db:                  db,
//		trie:                tr,
//		rawdb:               rawdb,
//		stateObjects:        make(map[common.Hash]StateObject),
//		stateObjectsPending: make(map[common.Hash]struct{}),
//		stateObjectsDirty:   make(map[common.Hash]struct{}),
//	}, nil
//}

// New return a new statedb attach with a state root
func NewWithPrefixTrie(root common.Hash, db DatabaseAccessWarper) (*StateDB, error) {
	tr, err := db.OpenPrefixTrie(root)
	if err != nil {
		return nil, err
	}
	metrics.EnabledExpensive = true
	return &StateDB{
		db:                  db,
		trie:                tr,
		stateObjects:        make(map[common.Hash]StateObject),
		stateObjectsPending: make(map[common.Hash]struct{}),
		stateObjectsDirty:   make(map[common.Hash]struct{}),
	}, nil
}

// setError remembers the first non-nil error it is called with.
func (stateDB *StateDB) setError(err error) {
	if stateDB.dbErr == nil {
		stateDB.dbErr = err
	}
}

// Error return statedb error
func (stateDB *StateDB) Error() error {
	return stateDB.dbErr
}

// Reset clears out all ephemeral state objects from the state db, but keeps
// the underlying state trie to avoid reloading data for the next operations.
func (stateDB *StateDB) Reset(root common.Hash) error {
	tr, err := stateDB.db.OpenPrefixTrie(root)
	if err != nil {
		return err
	}
	stateDB.trie = tr
	stateDB.stateObjects = make(map[common.Hash]StateObject)
	stateDB.stateObjectsPending = make(map[common.Hash]struct{})
	stateDB.stateObjectsDirty = make(map[common.Hash]struct{})
	return nil
}

func (stateDB *StateDB) ClearObjects() {
	stateDB.stateObjects = make(map[common.Hash]StateObject)
	stateDB.stateObjectsPending = make(map[common.Hash]struct{})
	stateDB.stateObjectsDirty = make(map[common.Hash]struct{})
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (stateDB *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	stateDB.markDeleteEmptyStateObject(deleteEmptyObjects)
	for addr := range stateDB.stateObjectsPending {
		obj := stateDB.stateObjects[addr]
		if obj.IsDeleted() {
			stateDB.deleteStateObject(obj)
		} else {
			stateDB.updateStateObject(obj)
		}
	}
	if len(stateDB.stateObjectsPending) > 0 {
		stateDB.stateObjectsPending = make(map[common.Hash]struct{})
	}
	// Track the amount of time wasted on hashing the account trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { stateDB.StateObjectHashes += time.Since(start) }(time.Now())
	}
	return stateDB.trie.Hash()
}
func (stateDB *StateDB) markDeleteEmptyStateObject(deleteEmptyObjects bool) {
	for _, object := range stateDB.stateObjects {
		if object.IsEmpty() {
			object.MarkDelete()
		}
	}
}

// Commit writes the state to the underlying in-memory trie database.
func (stateDB *StateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	// Finalize any pending changes and merge everything into the tries
	//if metrics.EnabledExpensive {
	//	defer func(start time.Time) {
	//		elapsed := time.Since(start)
	//		stateDB.StateObjectCommits += elapsed
	//		dataaccessobject.Logger.Log.Infof("StateDB commit and return root hash time %+v", elapsed)
	//	}(time.Now())
	//}
	stateDB.IntermediateRoot(deleteEmptyObjects)

	if len(stateDB.stateObjectsDirty) > 0 {
		stateDB.stateObjectsDirty = make(map[common.Hash]struct{})
	}
	// Write the account trie changes, measuing the amount of wasted time
	return stateDB.trie.Commit(func(leaf []byte, parent common.Hash) error {
		return nil
	})
}

// Database return current database access warper
func (stateDB *StateDB) Database() DatabaseAccessWarper {
	return stateDB.db
}

// Copy duplicate statedb and return new statedb instance
func (stateDB *StateDB) Copy() *StateDB {
	return &StateDB{
		db:                  stateDB.db,
		trie:                stateDB.db.CopyTrie(stateDB.trie),
		stateObjects:        make(map[common.Hash]StateObject),
		stateObjectsPending: make(map[common.Hash]struct{}),
		stateObjectsDirty:   make(map[common.Hash]struct{}),
	}
}

// Exist check existence of a state object in statedb
func (stateDB *StateDB) Exist(objectType int, stateObjectHash common.Hash) (bool, error) {
	value, err := stateDB.getStateObject(objectType, stateObjectHash)
	if err != nil {
		return false, err
	}
	return value != nil, nil
}

// Empty check a state object in statedb is empty or not
func (stateDB *StateDB) Empty(objectType int, stateObjectHash common.Hash) bool {
	stateObject, err := stateDB.getStateObject(objectType, stateObjectHash)
	return stateObject == nil || stateObject.IsEmpty() || err != nil
}

// ================================= STATE OBJECT =======================================
// getDeletedStateObject is similar to getStateObject, but instead of returning
// nil for a deleted state object, it returns the actual object with the deleted
// flag set. This is needed by the state journal to revert to the correct self-
// destructed object instead of wiping all knowledge about the state object.
func (stateDB *StateDB) getDeletedStateObject(objectType int, hash common.Hash) (StateObject, error) {
	// Prefer live objects if any is available
	if obj := stateDB.stateObjects[hash]; obj != nil {
		return obj, nil
	}
	// Track the amount of time wasted on loading the object from the database
	if metrics.EnabledExpensive {
		defer func(start time.Time) { stateDB.StateObjectReads += time.Since(start) }(time.Now())
	}
	// Load the object from the database
	enc, err := stateDB.trie.TryGet(hash[:])
	if len(enc) == 0 {
		stateDB.setError(err)
		return nil, nil
	}
	newValue := make([]byte, len(enc))
	copy(newValue, enc)
	// Insert into the live set
	obj, err := newStateObjectWithValue(stateDB, objectType, hash, newValue)
	if err != nil {
		return nil, err
	}
	stateDB.setStateObject(obj)
	return obj, nil
}

// updateStateObject writes the given object to the trie.
func (stateDB *StateDB) updateStateObject(obj StateObject) {
	// Track the amount of time wasted on updating the account from the trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { stateDB.StateObjectUpdates += time.Since(start) }(time.Now())
	}
	// Encode the account and update the account trie
	addr := obj.GetHash()
	data := obj.GetValueBytes()
	stateDB.setError(stateDB.trie.TryUpdate(addr[:], data))
}

// deleteStateObject removes the given object from the state trie.
func (stateDB *StateDB) deleteStateObject(obj StateObject) {
	// Track the amount of time wasted on deleting the account from the trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { stateDB.StateObjectUpdates += time.Since(start) }(time.Now())
	}
	// Delete the account from the trie
	addr := obj.GetHash()
	stateDB.setError(stateDB.trie.TryDelete(addr[:]))
}

// createStateObject creates a new state object. If there is an existing account with
// the given hash, it is overwritten and returned as the second return value.
func (stateDB *StateDB) createStateObject(objectType int, hash common.Hash) (newobj, prev StateObject, err error) {
	prev, err = stateDB.getDeletedStateObject(objectType, hash) // Note, prev might have been deleted, we need that!
	if err != nil {
		return nil, nil, err
	}
	newobj = newStateObject(stateDB, objectType, hash)
	stateDB.stateObjectsPending[hash] = struct{}{}
	stateDB.setStateObject(newobj)
	return newobj, prev, err
}

func (stateDB *StateDB) createStateObjectWithValue(objectType int, hash common.Hash, value interface{}) (newobj, prev StateObject, err error) {
	newobj, err = newStateObjectWithValue(stateDB, objectType, hash, value)
	if err != nil {
		return nil, nil, err
	}
	stateDB.stateObjectsPending[hash] = struct{}{}
	stateDB.setStateObject(newobj)
	return newobj, prev, err
}

// SetStateObject add new stateobject into statedb
func (stateDB *StateDB) SetStateObject(objectType int, key common.Hash, value interface{}) error {
	obj, err := stateDB.getOrNewStateObjectWithValue(objectType, key, value)
	if err != nil {
		return err
	}
	err = obj.SetValue(value)
	if err != nil {
		return err
	}
	stateDB.stateObjectsPending[key] = struct{}{}
	return nil
}

// MarkDeleteStateObject add new stateobject into statedb
func (stateDB *StateDB) MarkDeleteStateObject(objectType int, key common.Hash) bool {
	stateObject, err := stateDB.getStateObject(objectType, key)
	if err == nil && stateObject != nil {
		stateObject.MarkDelete()
		stateDB.stateObjectsPending[key] = struct{}{}
		return true
	}
	return false
}

// Retrieve a state object or create a new state object if nil.
func (stateDB *StateDB) getOrNewStateObject(objectType int, hash common.Hash) (StateObject, error) {
	stateObject, err := stateDB.getStateObject(objectType, hash)
	if err != nil {
		return nil, err
	}
	if stateObject == nil {
		stateObject, _, err = stateDB.createStateObject(objectType, hash)
		if err != nil {
			return nil, err
		}
	}
	return stateObject, nil
}

func (stateDB *StateDB) getOrNewStateObjectWithValue(objectType int, hash common.Hash, value interface{}) (StateObject, error) {
	stateObject, err := stateDB.getStateObject(objectType, hash)
	if err != nil {
		return nil, err
	}
	if stateObject == nil {
		stateObject, _, err = stateDB.createStateObjectWithValue(objectType, hash, value)
		if err != nil {
			return nil, err
		}
	}
	return stateObject, nil
}

// add state object into statedb struct
func (stateDB *StateDB) setStateObject(object StateObject) {
	key := object.GetHash()
	stateDB.stateObjects[key] = object
}

// getStateObject retrieves a state object given by the address, returning nil if
// the object is not found or was deleted in this execution context. If you need
// to differentiate between non-existent/just-deleted, use getDeletedStateObject.
func (stateDB *StateDB) getStateObject(objectType int, addr common.Hash) (StateObject, error) {
	if obj, err := stateDB.getDeletedStateObject(objectType, addr); obj != nil && !obj.IsDeleted() {
		return obj, nil
	} else if err != nil {
		return nil, err
	}
	return nil, nil
}

// FOR TEST ONLY
// do not use this function for build feature
func (stateDB *StateDB) GetStateObjectMapForTestOnly() map[common.Hash]StateObject {
	return stateDB.stateObjects
}

func (stateDB *StateDB) GetStateObjectPendingMapForTestOnly() map[common.Hash]struct{} {
	return stateDB.stateObjectsPending
}

// =================================     Test Object     ========================================
func (stateDB *StateDB) getTestObject(key common.Hash) ([]byte, error) {
	testObject, err := stateDB.getStateObject(TestObjectType, key)
	if err != nil {
		return []byte{}, err
	}
	if testObject != nil {
		return testObject.GetValueBytes(), nil
	}
	return []byte{}, nil
}

func (stateDB *StateDB) getAllTestObjectList() ([]common.Hash, [][]byte) {
	temp := stateDB.trie.NodeIterator(nil)
	it := trie.NewIterator(temp)
	keys := []common.Hash{}
	values := [][]byte{}
	for it.Next() {
		key := stateDB.trie.GetKey(it.Key)
		newKey := make([]byte, len(key))
		copy(newKey, key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		keys = append(keys, common.BytesToHash(key))
		values = append(values, value)
	}
	return keys, values
}

func (stateDB *StateDB) getAllTestObjectMap() map[common.Hash][]byte {
	temp := stateDB.trie.NodeIterator(nil)
	it := trie.NewIterator(temp)
	m := make(map[common.Hash][]byte)
	for it.Next() {
		key := stateDB.trie.GetKey(it.Key)
		newKey := make([]byte, len(key))
		copy(newKey, key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		m[common.BytesToHash(key)] = newValue
	}
	return m
}

func (stateDB *StateDB) getByPrefixTestObjectList(prefix []byte) ([]common.Hash, [][]byte) {
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	keys := []common.Hash{}
	values := [][]byte{}
	for it.Next() {
		key := stateDB.trie.GetKey(it.Key)
		newKey := make([]byte, len(key))
		copy(newKey, key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		keys = append(keys, common.BytesToHash(key))
		values = append(values, value)
	}
	return keys, values
}

// ================================= Committee OBJECT =======================================
func (stateDB *StateDB) getCommitteeState(key common.Hash) (*CommitteeState, bool, error) {
	committeeStateObject, err := stateDB.getStateObject(CommitteeObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if committeeStateObject != nil {
		return committeeStateObject.GetValue().(*CommitteeState), true, nil
	}
	return NewCommitteeState(), false, nil
}

func (stateDB *StateDB) getStakerInfo(key common.Hash) (*StakerInfo, bool, error) {
	stakerObject, err := stateDB.getStateObject(StakerObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if stakerObject != nil {
		res, ok := stakerObject.GetValue().(*StakerInfo)
		if !ok {
			err = fmt.Errorf("Can not parse staker info")
		}
		return res, true, err
	}
	return NewStakerInfo(), false, nil
}

func (stateDB *StateDB) getStakerObject(key common.Hash) (*StateObject, bool, error) {
	stakerObject, err := stateDB.getStateObject(StakerObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if stakerObject == nil {
		return nil, false, nil
	}
	return &stakerObject, true, nil
}

func (stateDB *StateDB) getAllValidatorCommitteePublicKey(role int, ids []int) map[int][]*CommitteeState {
	if role != CurrentValidator && role != SubstituteValidator {
		panic("wrong expected role " + strconv.Itoa(role))
	}
	m := make(map[int][]*CommitteeState)
	for _, id := range ids {
		prefix := GetCommitteePrefixWithRole(role, id)
		temp := stateDB.trie.NodeIterator(prefix)
		it := trie.NewIterator(temp)
		for it.Next() {
			value := it.Value
			newValue := make([]byte, len(value))
			copy(newValue, value)
			committeeState := NewCommitteeState()
			err := json.Unmarshal(newValue, committeeState)
			if err != nil {
				panic("wrong value type")
			}
			m[committeeState.shardID] = append(m[committeeState.shardID], committeeState)
		}
	}
	return m
}

func (stateDB *StateDB) getAllCandidateCommitteePublicKey(role int) []*CommitteeState {
	if role != CurrentEpochShardCandidate && role != NextEpochShardCandidate {
		panic("wrong expected role " + strconv.Itoa(role))
	}
	list := []*CommitteeState{}
	prefix := GetCommitteePrefixWithRole(role, CandidateShardID)
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		committeeState := NewCommitteeState()
		err := committeeState.UnmarshalJSON(newValue)
		if err != nil {
			panic("wrong value type")
		}
		list = append(list, committeeState)
	}
	return list
}

func (stateDB *StateDB) getByShardIDCurrentValidatorState(shardID int) []*CommitteeState {
	committees := []*CommitteeState{}
	prefix := GetCommitteePrefixWithRole(CurrentValidator, shardID)
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		committeeState := NewCommitteeState()
		err := json.Unmarshal(newValue, committeeState)
		if err != nil {
			panic("wrong value type")
		}
		if committeeState.ShardID() != shardID {
			panic("wrong expected shard id")
		}
		committees = append(committees, committeeState)
	}
	return committees
}

func (stateDB *StateDB) getByShardIDSubstituteValidatorState(shardID int) []*CommitteeState {
	committees := []*CommitteeState{}
	prefix := GetCommitteePrefixWithRole(SubstituteValidator, shardID)
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		committeeState := NewCommitteeState()
		err := json.Unmarshal(newValue, committeeState)
		if err != nil {
			panic("wrong value type")
		}
		if committeeState.ShardID() != shardID {
			panic("wrong expected shard id")
		}
		committees = append(committees, committeeState)
	}
	return committees
}

// getAllCommitteeState return all data related to all committee roles
// return params #1: current validator
// return params #2: substitute validator
// return params #3: next epoch candidate
// return params #4: current epoch candidate
// return params #5: reward receiver map
// return params #6: auto staking map
func (stateDB *StateDB) getAllCommitteeState(ids []int) (
	currentValidator map[int][]*CommitteeState,
	substituteValidator map[int][]*CommitteeState,
	nextEpochShardCandidate []*CommitteeState,
	currentEpochShardCandidate []*CommitteeState,
	nextEpochBeaconCandidate []*CommitteeState,
	currentEpochBeaconCandidate []*CommitteeState,
) {
	currentValidator = make(map[int][]*CommitteeState)
	substituteValidator = make(map[int][]*CommitteeState)
	nextEpochShardCandidate = []*CommitteeState{}
	currentEpochShardCandidate = []*CommitteeState{}
	nextEpochBeaconCandidate = []*CommitteeState{}
	currentEpochBeaconCandidate = []*CommitteeState{}
	for _, shardID := range ids {
		// Current Validator
		prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, shardID)
		resCurrentValidator := stateDB.iterateWithCommitteeState(prefixCurrentValidator)
		tempCurrentValidator := []*CommitteeState{}
		for _, v := range resCurrentValidator {
			tempCurrentValidator = append(tempCurrentValidator, v)
		}
		currentValidator[shardID] = tempCurrentValidator
		// Substitute Validator
		prefixSubstituteValidator := GetCommitteePrefixWithRole(SubstituteValidator, shardID)
		resSubstituteValidator := stateDB.iterateWithCommitteeState(prefixSubstituteValidator)
		tempSubstituteValidator := []*CommitteeState{}
		for _, v := range resSubstituteValidator {
			tempSubstituteValidator = append(tempSubstituteValidator, v)
		}
		substituteValidator[shardID] = tempSubstituteValidator
	}
	// next epoch candidate
	prefixNextEpochCandidate := GetCommitteePrefixWithRole(NextEpochShardCandidate, -2)
	resNextEpochCandidate := stateDB.iterateWithCommitteeState(prefixNextEpochCandidate)
	for _, v := range resNextEpochCandidate {
		nextEpochShardCandidate = append(nextEpochShardCandidate, v)
	}
	// current epoch candidate
	prefixCurrentEpochCandidate := GetCommitteePrefixWithRole(CurrentEpochShardCandidate, -2)
	resCurrentEpochCandidate := stateDB.iterateWithCommitteeState(prefixCurrentEpochCandidate)
	for _, v := range resCurrentEpochCandidate {
		currentEpochShardCandidate = append(currentEpochShardCandidate, v)
	}

	// next epoch candidate
	prefixNextEpochBeaconCandidate := GetCommitteePrefixWithRole(NextEpochBeaconCandidate, -2)
	resNextEpochBeaconCandidate := stateDB.iterateWithCommitteeState(prefixNextEpochBeaconCandidate)
	for _, v := range resNextEpochBeaconCandidate {
		nextEpochBeaconCandidate = append(nextEpochBeaconCandidate, v)
	}
	// current epoch candidate
	prefixCurrentEpochBeaconCandidate := GetCommitteePrefixWithRole(CurrentEpochBeaconCandidate, -2)
	resCurrentEpochBeaconCandidate := stateDB.iterateWithCommitteeState(prefixCurrentEpochBeaconCandidate)
	for _, v := range resCurrentEpochBeaconCandidate {
		currentEpochBeaconCandidate = append(currentEpochBeaconCandidate, v)
	}
	return currentValidator, substituteValidator, nextEpochShardCandidate, currentEpochShardCandidate, nextEpochBeaconCandidate, currentEpochBeaconCandidate
}

func (stateDB *StateDB) IterateWithStaker(prefix []byte) []*StakerInfo {
	m := []*StakerInfo{}
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		committeeState := NewStakerInfo()
		err := json.Unmarshal(newValue, committeeState)
		if err != nil {
			panic(err)
		}
		m = append(m, committeeState)
	}
	return m
}

func (stateDB *StateDB) iterateWithCommitteeState(prefix []byte) []*CommitteeState {
	m := []*CommitteeState{}
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		committeeState := NewCommitteeState()
		err := json.Unmarshal(newValue, committeeState)
		if err != nil {
			panic(err)
		}
		m = append(m, committeeState)
	}
	return m
}

// ================================= Committee Reward OBJECT =======================================
func (stateDB *StateDB) getCommitteeRewardState(key common.Hash) (*CommitteeRewardState, bool, error) {
	committeeRewardObject, err := stateDB.getStateObject(CommitteeRewardObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if committeeRewardObject != nil {
		return committeeRewardObject.GetValue().(*CommitteeRewardState), true, nil
	}
	return NewCommitteeRewardState(), false, nil
}

func (stateDB *StateDB) getCommitteeRewardAmount(key common.Hash) (map[common.Hash]uint64, bool, error) {
	m := make(map[common.Hash]uint64)
	committeeRewardObject, err := stateDB.getStateObject(CommitteeRewardObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if committeeRewardObject != nil {
		temp := committeeRewardObject.GetValue().(*CommitteeRewardState)
		m = temp.reward
		return m, true, nil
	}
	return m, false, nil
}

func (stateDB *StateDB) getAllCommitteeReward() map[string]map[common.Hash]uint64 {
	m := make(map[string]map[common.Hash]uint64)
	prefix := GetCommitteeRewardPrefix()
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		committeeRewardState := NewCommitteeRewardState()
		err := json.Unmarshal(newValue, committeeRewardState)
		if err != nil {
			panic("wrong value type")
		}
		m[committeeRewardState.incognitoPublicKey] = committeeRewardState.reward
	}
	return m
}

func (stateDB *StateDB) getShardsCommitteeState(sIDs []int) (currentValidator map[int][]*CommitteeState) {
	currentValidator = make(map[int][]*CommitteeState)
	for _, shardID := range sIDs {
		// Current Validator
		prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, shardID)
		resCurrentValidator := stateDB.iterateWithCommitteeState(prefixCurrentValidator)
		tempCurrentValidator := []*CommitteeState{}
		for _, v := range resCurrentValidator {
			tempCurrentValidator = append(tempCurrentValidator, v)
		}
		currentValidator[shardID] = tempCurrentValidator
	}
	return currentValidator
}

func (stateDB *StateDB) getShardsCommitteeInfo(sIDs []int) (curValidatorInfo map[int][]*StakerInfo) {
	currentValidator := make(map[int][]*CommitteeState)
	curValidatorInfo = make(map[int][]*StakerInfo)
	for _, shardID := range sIDs {
		// Current Validator
		prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, shardID)
		resCurrentValidator := stateDB.iterateWithCommitteeState(prefixCurrentValidator)
		tempCurrentValidator := []*CommitteeState{}
		for _, v := range resCurrentValidator {
			tempCurrentValidator = append(tempCurrentValidator, v)
		}
		currentValidator[shardID] = tempCurrentValidator
		tempStakerInfos := []*StakerInfo{}
		for _, c := range currentValidator[shardID] {
			cPKBytes, _ := c.committeePublicKey.RawBytes()
			s, has, err := stateDB.getStakerInfo(GetStakerInfoKey(cPKBytes))
			if err != nil {
				panic(err)
			}
			if !has || s == nil {
				panic(errors.Errorf("Can not found staker info for this committee %v", c.committeePublicKey))
			}
			tempStakerInfos = append(tempStakerInfos, s)
		}
		curValidatorInfo[shardID] = tempStakerInfos
	}
	return curValidatorInfo
}

func (beaconConsensusStateDB *StateDB) GetAllStakingTX(ids []int) (map[string]string, error) {
	allStaker := []*CommitteeState{}
	mapStakingTx := map[string]string{}
	for _, shardID := range ids {
		// Current Validator
		prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, shardID)
		resCurrentValidator := beaconConsensusStateDB.iterateWithCommitteeState(prefixCurrentValidator)
		allStaker = append(allStaker, resCurrentValidator...)
		// Substitute Validator
		prefixSubstituteValidator := GetCommitteePrefixWithRole(SubstituteValidator, shardID)
		resSubstituteValidator := beaconConsensusStateDB.iterateWithCommitteeState(prefixSubstituteValidator)
		allStaker = append(allStaker, resSubstituteValidator...)
	}
	// next epoch candidate
	prefixNextEpochCandidate := GetCommitteePrefixWithRole(NextEpochShardCandidate, -2)
	resNextEpochCandidate := beaconConsensusStateDB.iterateWithCommitteeState(prefixNextEpochCandidate)
	allStaker = append(allStaker, resNextEpochCandidate...)
	// current epoch candidate
	prefixCurrentEpochCandidate := GetCommitteePrefixWithRole(CurrentEpochShardCandidate, -2)
	resCurrentEpochCandidate := beaconConsensusStateDB.iterateWithCommitteeState(prefixCurrentEpochCandidate)
	allStaker = append(allStaker, resCurrentEpochCandidate...)

	// next epoch candidate
	prefixNextEpochBeaconCandidate := GetCommitteePrefixWithRole(NextEpochBeaconCandidate, -2)
	resNextEpochBeaconCandidate := beaconConsensusStateDB.iterateWithCommitteeState(prefixNextEpochBeaconCandidate)
	allStaker = append(allStaker, resNextEpochBeaconCandidate...)
	// current epoch candidate
	prefixCurrentEpochBeaconCandidate := GetCommitteePrefixWithRole(CurrentEpochBeaconCandidate, -2)
	resCurrentEpochBeaconCandidate := beaconConsensusStateDB.iterateWithCommitteeState(prefixCurrentEpochBeaconCandidate)
	allStaker = append(allStaker, resCurrentEpochBeaconCandidate...)

	for _, v := range allStaker {
		pubKeyBytes, _ := v.committeePublicKey.RawBytes()
		key := GetStakerInfoKey(pubKeyBytes)
		stakerInfo, has, err := beaconConsensusStateDB.getStakerInfo(key)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		pKey, err := v.committeePublicKey.ToBase58()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		if (!has) || (stakerInfo == nil) {
			fmt.Println("No staker info")
			return nil, errors.Errorf("Can not found staker info for this committee public key %v", pKey)
		}
		if stakerInfo.txStakingID.String() != common.HashH([]byte{0}).String() {
			mapStakingTx[pKey] = stakerInfo.txStakingID.String()
		}
	}
	return mapStakingTx, nil
}

func (stateDB *StateDB) getMapAutoStaking(ids []int) (map[string]bool, error) {
	allStaker := []*CommitteeState{}
	mapAutoStaking := map[string]bool{}

	// Current Beacon Validator
	prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, -1)
	resCurrentValidator := stateDB.iterateWithCommitteeState(prefixCurrentValidator)
	allStaker = append(allStaker, resCurrentValidator...)
	// Substitute Beacon Validator
	prefixSubstituteValidator := GetCommitteePrefixWithRole(SubstituteValidator, -1)
	resSubstituteValidator := stateDB.iterateWithCommitteeState(prefixSubstituteValidator)
	allStaker = append(allStaker, resSubstituteValidator...)

	for _, shardID := range ids {
		// Current Shard Validator
		prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, shardID)
		resCurrentValidator := stateDB.iterateWithCommitteeState(prefixCurrentValidator)
		allStaker = append(allStaker, resCurrentValidator...)
		// Substitute Shard sValidator
		prefixSubstituteValidator := GetCommitteePrefixWithRole(SubstituteValidator, shardID)
		resSubstituteValidator := stateDB.iterateWithCommitteeState(prefixSubstituteValidator)
		allStaker = append(allStaker, resSubstituteValidator...)
	}
	// next epoch candidate
	prefixNextEpochCandidate := GetCommitteePrefixWithRole(NextEpochShardCandidate, -2)
	resNextEpochCandidate := stateDB.iterateWithCommitteeState(prefixNextEpochCandidate)
	allStaker = append(allStaker, resNextEpochCandidate...)
	// current epoch candidate
	prefixCurrentEpochCandidate := GetCommitteePrefixWithRole(CurrentEpochShardCandidate, -2)
	resCurrentEpochCandidate := stateDB.iterateWithCommitteeState(prefixCurrentEpochCandidate)
	allStaker = append(allStaker, resCurrentEpochCandidate...)

	// next epoch candidate
	prefixNextEpochBeaconCandidate := GetCommitteePrefixWithRole(NextEpochBeaconCandidate, -2)
	resNextEpochBeaconCandidate := stateDB.iterateWithCommitteeState(prefixNextEpochBeaconCandidate)
	allStaker = append(allStaker, resNextEpochBeaconCandidate...)
	// current epoch candidate
	prefixCurrentEpochBeaconCandidate := GetCommitteePrefixWithRole(CurrentEpochBeaconCandidate, -2)
	resCurrentEpochBeaconCandidate := stateDB.iterateWithCommitteeState(prefixCurrentEpochBeaconCandidate)
	allStaker = append(allStaker, resCurrentEpochBeaconCandidate...)
	for _, v := range allStaker {
		pubKeyBytes, _ := v.committeePublicKey.RawBytes()
		key := GetStakerInfoKey(pubKeyBytes)
		stakerInfo, has, err := stateDB.getStakerInfo(key)
		if err != nil {
			return nil, err
		}
		pKey, err := v.committeePublicKey.ToBase58()
		if err != nil {
			return nil, err
		}
		if (!has) || (stakerInfo == nil) {
			return nil, errors.Errorf("Can not found staker info for this committee public key %v", pKey)
		}
		if stakerInfo.txStakingID.String() != common.HashH([]byte{0}).String() {
			mapAutoStaking[pKey] = stakerInfo.autoStaking
		} else {
			mapAutoStaking[pKey] = false
		}
	}
	return mapAutoStaking, nil
}

// ================================= Reward Request OBJECT =======================================
func (stateDB *StateDB) getRewardRequestState(key common.Hash) (*RewardRequestState, bool, error) {
	rewardRequestState, err := stateDB.getStateObject(RewardRequestObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if rewardRequestState != nil {
		return rewardRequestState.GetValue().(*RewardRequestState), true, nil
	}
	return NewRewardRequestState(), false, nil
}

func (stateDB *StateDB) getRewardRequestAmount(key common.Hash) (uint64, bool, error) {
	amount := uint64(0)
	rewardRequestObject, err := stateDB.getStateObject(RewardRequestObjectType, key)
	if err != nil {
		return amount, false, err
	}
	if rewardRequestObject != nil {
		temp := rewardRequestObject.GetValue().(*RewardRequestState)
		amount = temp.amount
		return amount, true, nil
	}
	return amount, false, nil
}

func (stateDB *StateDB) getAllRewardRequestState(epoch uint64) ([]common.Hash, []*RewardRequestState) {
	m := []*RewardRequestState{}
	keys := []common.Hash{}
	prefix := GetRewardRequestPrefix(epoch)
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		key := it.Key
		newKey := make([]byte, len(key))
		copy(newKey, key)
		keys = append(keys, common.BytesToHash(newKey))
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		rewardRequestState := NewRewardRequestState()
		err := json.Unmarshal(newValue, rewardRequestState)
		if err != nil {
			panic("wrong value type")
		}
		m = append(m, rewardRequestState)
	}
	return keys, m
}

// ================================= Black List Producer OBJECT =======================================
func (stateDB *StateDB) getBlackListProducerState(key common.Hash) (*BlackListProducerState, bool, error) {
	blackListProducerState, err := stateDB.getStateObject(BlackListProducerObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if blackListProducerState != nil {
		return blackListProducerState.GetValue().(*BlackListProducerState), true, nil
	}
	return NewBlackListProducerState(), false, nil
}

func (stateDB *StateDB) getBlackListProducerPunishedEpoch(key common.Hash) (uint8, bool, error) {
	duration := uint8(0)
	blackListProducerObject, err := stateDB.getStateObject(BlackListProducerObjectType, key)
	if err != nil {
		return duration, false, err
	}
	if blackListProducerObject != nil {
		temp := blackListProducerObject.GetValue().(*BlackListProducerState)
		duration = temp.punishedEpoches
		return duration, true, nil
	}
	return duration, false, nil
}

func (stateDB *StateDB) getAllBlackListProducerState() []*BlackListProducerState {
	blackListProducerStates := []*BlackListProducerState{}
	prefix := GetBlackListProducerPrefix()
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		blackListProducerState := NewBlackListProducerState()
		err := json.Unmarshal(newValue, blackListProducerState)
		if err != nil {
			panic("wrong value type")
		}
		blackListProducerStates = append(blackListProducerStates, blackListProducerState)
	}
	return blackListProducerStates
}

func (stateDB *StateDB) getAllProducerBlackList() map[string]uint8 {
	m := make(map[string]uint8)
	prefix := GetBlackListProducerPrefix()
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		blackListProducerState := NewBlackListProducerState()
		err := json.Unmarshal(newValue, blackListProducerState)
		if err != nil {
			panic("wrong value type")
		}
		m[blackListProducerState.producerCommitteePublicKey] = blackListProducerState.punishedEpoches
	}
	return m
}

func (stateDB *StateDB) getAllProducerBlackListState() map[common.Hash]uint8 {
	m := make(map[common.Hash]uint8)
	prefix := GetBlackListProducerPrefix()
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		key := it.Key
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		blackListProducerState := NewBlackListProducerState()
		err := json.Unmarshal(newValue, blackListProducerState)
		if err != nil {
			panic("wrong value type")
		}
		m[common.BytesToHash(key)] = blackListProducerState.punishedEpoches
	}
	return m
}

// ================================= Serial Number OBJECT =======================================
func (stateDB *StateDB) getSerialNumberState(key common.Hash) (*SerialNumberState, bool, error) {
	serialNumberState, err := stateDB.getStateObject(SerialNumberObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if serialNumberState != nil {
		return serialNumberState.GetValue().(*SerialNumberState), true, nil
	}
	return NewSerialNumberState(), false, nil
}

func (stateDB *StateDB) getAllSerialNumberByPrefix(tokenID common.Hash, shardID byte) [][]byte {
	serialNumberList := [][]byte{}
	prefix := GetSerialNumberPrefix(tokenID, shardID)
	temp := stateDB.trie.NodeIterator(prefix)
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		serialNumberState := NewSerialNumberState()
		err := json.Unmarshal(newValue, serialNumberState)
		if err != nil {
			panic("wrong value type")
		}
		serialNumberList = append(serialNumberList, serialNumberState.SerialNumber())
	}
	return serialNumberList
}

// ================================= Commitment OBJECT =======================================
func (stateDB *StateDB) getCommitmentState(key common.Hash) (*CommitmentState, bool, error) {
	commitmentState, err := stateDB.getStateObject(CommitmentObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if commitmentState != nil {
		return commitmentState.GetValue().(*CommitmentState), true, nil
	}
	return NewCommitmentState(), false, nil
}

func (stateDB *StateDB) getCommitmentIndexState(key common.Hash) (*CommitmentState, bool, error) {
	commitmentIndexState, err := stateDB.getStateObject(CommitmentIndexObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if commitmentIndexState != nil {
		tempKey, ok := commitmentIndexState.GetValue().(common.Hash)
		if !ok {
			panic("wrong expected type")
		}
		commitmentState, err := stateDB.getDeletedStateObject(CommitmentObjectType, tempKey)
		if err != nil || commitmentState == nil {
			return NewCommitmentState(), false, nil
		}
		return commitmentState.GetValue().(*CommitmentState), true, nil
	}
	return NewCommitmentState(), false, nil
}

func (stateDB *StateDB) getCommitmentLengthState(key common.Hash) (*big.Int, bool, error) {
	commitmentLengthState, err := stateDB.getStateObject(CommitmentLengthObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if commitmentLengthState != nil {
		return commitmentLengthState.GetValue().(*big.Int), true, nil
	}
	return new(big.Int), false, nil
}

func (stateDB *StateDB) getAllCommitmentStateByPrefix(tokenID common.Hash, shardID byte) map[string]uint64 {
	temp := stateDB.trie.NodeIterator(GetCommitmentPrefix(tokenID, shardID))
	it := trie.NewIterator(temp)
	m := make(map[string]uint64)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		newCommitmentState := NewCommitmentState()
		err := json.Unmarshal(newValue, newCommitmentState)
		if err != nil {
			panic("wrong expect type")
		}
		commitmentString := base58.Base58Check{}.Encode(newCommitmentState.Commitment(), common.Base58Version)
		m[commitmentString] = newCommitmentState.Index().Uint64()
	}
	return m
}

// ================================= Output Coin OBJECT =======================================
func (stateDB *StateDB) getOutputCoinState(key common.Hash) (*OutputCoinState, bool, error) {
	outputCoinState, err := stateDB.getStateObject(OutputCoinObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if outputCoinState != nil {
		return outputCoinState.GetValue().(*OutputCoinState), true, nil
	}
	return NewOutputCoinState(), false, nil
}

func (stateDB *StateDB) getAllOutputCoinState(tokenID common.Hash, shardID byte, publicKey []byte) []*OutputCoinState {
	temp := stateDB.trie.NodeIterator(GetOutputCoinPrefix(tokenID, shardID, publicKey))
	it := trie.NewIterator(temp)
	outputCoins := []*OutputCoinState{}
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		newOutputCoin := NewOutputCoinState()
		err := json.Unmarshal(newValue, newOutputCoin)
		if err != nil {
			panic("wrong expect type")
		}
		outputCoins = append(outputCoins, newOutputCoin)
	}
	return outputCoins
}

// ================================= OnetimeAddress OBJECT =======================================

func (stateDB *StateDB) getOnetimeAddressState(key common.Hash) (*OnetimeAddressState, bool, error) {
	onetimeAddressState, err := stateDB.getStateObject(OnetimeAddressObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if onetimeAddressState != nil {
		return onetimeAddressState.GetValue().(*OnetimeAddressState), true, nil
	}
	return NewOnetimeAddressState(), false, nil
}

func (stateDB *StateDB) getOTACoinIndexState(key common.Hash) (*OTACoinState, bool, error) {
	otaCoinIndexState, err := stateDB.getStateObject(OTACoinIndexObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if otaCoinIndexState != nil {
		tempKey, ok := otaCoinIndexState.GetValue().(common.Hash)
		if !ok {
			panic("wrong expected type")
		}
		otaCoinState, err := stateDB.getDeletedStateObject(OTACoinObjectType, tempKey)
		if err != nil || otaCoinState == nil {
			return NewOTACoinState(), false, nil
		}
		return otaCoinState.GetValue().(*OTACoinState), true, nil
	}
	return NewOTACoinState(), false, nil
}

func (stateDB *StateDB) getOTACoinLengthState(key common.Hash) (*big.Int, bool, error) {
	otaCoinLengthState, err := stateDB.getStateObject(OTACoinLengthObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if otaCoinLengthState != nil {
		return otaCoinLengthState.GetValue().(*big.Int), true, nil
	}
	return new(big.Int), false, nil
}

func (stateDB *StateDB) getAllOTACoinsByPrefix(tokenID common.Hash, shardID byte, height []byte) []*OTACoinState {
	temp := stateDB.trie.NodeIterator(GetOTACoinPrefix(tokenID, shardID, height))
	it := trie.NewIterator(temp)
	onetimeAddresses := make([]*OTACoinState, 0)

	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		newOnetimeAddress := NewOTACoinState()
		err := json.Unmarshal(newValue, newOnetimeAddress)
		if err != nil {
			panic("wrong expect type")
		}
		onetimeAddresses = append(onetimeAddresses, newOnetimeAddress)
	}
	return onetimeAddresses
}

// ================================= SNDerivator OBJECT =======================================
func (stateDB *StateDB) getSNDerivatorState(key common.Hash) (*SNDerivatorState, bool, error) {
	sndState, err := stateDB.getStateObject(SNDerivatorObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if sndState != nil {
		return sndState.GetValue().(*SNDerivatorState), true, nil
	}
	return NewSNDerivatorState(), false, nil
}

func (stateDB *StateDB) getAllSNDerivatorStateByPrefix(tokenID common.Hash) [][]byte {
	temp := stateDB.trie.NodeIterator(GetSNDerivatorPrefix(tokenID))
	it := trie.NewIterator(temp)
	list := [][]byte{}
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		newSNDerivatorState := NewSNDerivatorState()
		err := json.Unmarshal(newValue, newSNDerivatorState)
		if err != nil {
			panic("wrong expect type")
		}
		list = append(list, newSNDerivatorState.Snd())
	}
	return list
}

// ================================= Token OBJECT =======================================
func (stateDB *StateDB) getTokenState(key common.Hash) (*TokenState, bool, error) {
	tokenState, err := stateDB.getStateObject(TokenObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if tokenState != nil {
		return tokenState.GetValue().(*TokenState), true, nil
	}
	return NewTokenState(), false, nil
}

func (stateDB *StateDB) getTokenTxs(tokenID common.Hash) []common.Hash {
	txs := []common.Hash{}
	temp := stateDB.trie.NodeIterator(GetTokenTransactionPrefix(tokenID))
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		tokenTransactionState := NewTokenTransactionState()
		err := json.Unmarshal(newValue, tokenTransactionState)
		if err != nil {
			panic("wrong expect type")
		}
		txs = append(txs, tokenTransactionState.TxHash())
	}
	return txs
}

func (stateDB *StateDB) getAllTokenWithTxs() map[common.Hash]*TokenState {
	temp := stateDB.trie.NodeIterator(GetTokenPrefix())
	it := trie.NewIterator(temp)
	tokenIDs := make(map[common.Hash]*TokenState)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		tokenState := NewTokenState()
		err := json.Unmarshal(newValue, tokenState)
		if err != nil {
			panic("wrong expect type")
		}
		tokenID := tokenState.TokenID()
		txs := stateDB.getTokenTxs(tokenID)
		tokenState.AddTxs(txs)
		tokenIDs[tokenID] = tokenState
	}
	return tokenIDs
}

func (stateDB *StateDB) getAllToken() map[common.Hash]*TokenState {
	temp := stateDB.trie.NodeIterator(GetTokenPrefix())
	it := trie.NewIterator(temp)
	tokenIDs := make(map[common.Hash]*TokenState)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		tokenState := NewTokenState()
		err := json.Unmarshal(newValue, tokenState)
		if err != nil {
			panic("wrong expect type")
		}
		tokenID := tokenState.TokenID()
		tokenIDs[tokenID] = tokenState
	}
	return tokenIDs
}

// ================================= PDE OBJECT =======================================
func (stateDB *StateDB) getAllWaitingPDEContributionState() []*WaitingPDEContributionState {
	waitingPDEContributionStates := []*WaitingPDEContributionState{}
	temp := stateDB.trie.NodeIterator(GetWaitingPDEContributionPrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		wc := NewWaitingPDEContributionState()
		err := json.Unmarshal(newValue, wc)
		if err != nil {
			panic("wrong expect type")
		}
		waitingPDEContributionStates = append(waitingPDEContributionStates, wc)
	}
	return waitingPDEContributionStates
}

func (stateDB *StateDB) getAllPDEPoolPairState() []*PDEPoolPairState {
	pdePoolPairStates := []*PDEPoolPairState{}
	temp := stateDB.trie.NodeIterator(GetPDEPoolPairPrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		pp := NewPDEPoolPairState()
		err := json.Unmarshal(newValue, pp)
		if err != nil {
			panic("wrong expect type")
		}
		pdePoolPairStates = append(pdePoolPairStates, pp)
	}
	return pdePoolPairStates
}

func (stateDB *StateDB) getPDEPoolPairState(key common.Hash) (*PDEPoolPairState, bool, error) {
	ppState, err := stateDB.getStateObject(PDEPoolPairObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if ppState != nil {
		return ppState.GetValue().(*PDEPoolPairState), true, nil
	}
	return NewPDEPoolPairState(), false, nil
}

func (stateDB *StateDB) getAllPDEShareState() []*PDEShareState {
	pdeShareStates := []*PDEShareState{}
	temp := stateDB.trie.NodeIterator(GetPDESharePrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		pp := NewPDEShareState()
		err := json.Unmarshal(newValue, pp)
		if err != nil {
			panic("wrong expect type")
		}
		pdeShareStates = append(pdeShareStates, pp)
	}
	return pdeShareStates
}

func (stateDB *StateDB) getAllPDETradingFeeState() []*PDETradingFeeState {
	pdeTradingFeeStates := []*PDETradingFeeState{}
	temp := stateDB.trie.NodeIterator(GetPDETradingFeePrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		pp := NewPDETradingFeeState()
		err := json.Unmarshal(newValue, pp)
		if err != nil {
			panic("wrong expect type")
		}
		pdeTradingFeeStates = append(pdeTradingFeeStates, pp)
	}
	return pdeTradingFeeStates
}

func (stateDB *StateDB) getAllPDEStatus() []*PDEStatusState {
	pdeStatusStates := []*PDEStatusState{}
	temp := stateDB.trie.NodeIterator(GetPDEStatusPrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		s := NewPDEStatusState()
		err := json.Unmarshal(newValue, s)
		if err != nil {
			panic("wrong expect type")
		}
		pdeStatusStates = append(pdeStatusStates, s)
	}
	return pdeStatusStates
}

func (stateDB *StateDB) getPDEStatusByKey(key common.Hash) (*PDEStatusState, bool, error) {
	pdeStatusState, err := stateDB.getStateObject(PDEStatusObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if pdeStatusState != nil {
		return pdeStatusState.GetValue().(*PDEStatusState), true, nil
	}
	return NewPDEStatusState(), false, nil
}

// ================================= Bridge OBJECT =======================================
func (stateDB *StateDB) getBridgeEthTxState(key common.Hash) (*BridgeEthTxState, bool, error) {
	ethTxState, err := stateDB.getStateObject(BridgeEthTxObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if ethTxState != nil {
		return ethTxState.GetValue().(*BridgeEthTxState), true, nil
	}
	return NewBridgeEthTxState(), false, nil
}

func (stateDB *StateDB) getBridgeTokenInfoState(key common.Hash) (*BridgeTokenInfoState, bool, error) {
	tokenInfoState, err := stateDB.getStateObject(BridgeTokenInfoObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if tokenInfoState != nil {
		return tokenInfoState.GetValue().(*BridgeTokenInfoState), true, nil
	}
	return NewBridgeTokenInfoState(), false, nil
}

func (stateDB *StateDB) getAllBridgeTokenInfoState(isCentralized bool) []*BridgeTokenInfoState {
	bridgeTokenInfoStates := []*BridgeTokenInfoState{}
	temp := stateDB.trie.NodeIterator(GetBridgeTokenInfoPrefix(isCentralized))
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		s := NewBridgeTokenInfoState()
		err := json.Unmarshal(newValue, s)
		if err != nil {
			panic("wrong expect type")
		}
		bridgeTokenInfoStates = append(bridgeTokenInfoStates, s)
	}
	return bridgeTokenInfoStates
}

func (stateDB *StateDB) getBridgeStatusState(key common.Hash) (*BridgeStatusState, bool, error) {
	statusState, err := stateDB.getStateObject(BridgeStatusObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if statusState != nil {
		return statusState.GetValue().(*BridgeStatusState), true, nil
	}
	return NewBridgeStatusState(), false, nil
}

// ================================= Burn OBJECT =======================================
func (stateDB *StateDB) getBurningConfirmState(key common.Hash) (*BurningConfirmState, bool, error) {
	burningConfirmState, err := stateDB.getStateObject(BurningConfirmObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if burningConfirmState != nil {
		return burningConfirmState.GetValue().(*BurningConfirmState), true, nil
	}
	return NewBurningConfirmState(), false, nil
}

// ================================= Portal OBJECT =======================================
func (stateDB *StateDB) getWaitingPortingRequests() map[string]*WaitingPortingRequest {
	waitingPortingRequest := make(map[string]*WaitingPortingRequest)
	temp := stateDB.trie.NodeIterator(GetPortalWaitingPortingRequestPrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		key := it.Key
		keyHash, _ := common.Hash{}.NewHash(key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		object := NewWaitingPortingRequest()
		err := json.Unmarshal(newValue, object)
		if err != nil {
			panic("wrong expect type")
		}
		waitingPortingRequest[keyHash.String()] = object
	}

	return waitingPortingRequest
}

func (stateDB *StateDB) getCustodianByKey(key common.Hash) (*CustodianState, bool, error) {
	custodianState, err := stateDB.getStateObject(CustodianStateObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if custodianState != nil {
		return custodianState.GetValue().(*CustodianState), true, nil
	}
	return NewCustodianState(), false, nil
}

func (stateDB *StateDB) getFinalExchangeRatesByKey(key common.Hash) (*FinalExchangeRatesState, bool, error) {
	finalExchangeRates, err := stateDB.getStateObject(PortalFinalExchangeRatesStateObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if finalExchangeRates != nil {
		return finalExchangeRates.GetValue().(*FinalExchangeRatesState), true, nil
	}
	return NewFinalExchangeRatesState(), false, nil
}

func (stateDB *StateDB) getLiquidateExchangeRatesPoolByKey(key common.Hash) (*LiquidationPool, bool, error) {
	liquidateExchangeRates, err := stateDB.getStateObject(PortalLiquidationPoolObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if liquidateExchangeRates != nil {
		return liquidateExchangeRates.GetValue().(*LiquidationPool), true, nil
	}
	return NewLiquidationPool(), false, nil
}

func (stateDB *StateDB) getLiquidateExchangeRatesPool() map[string]*LiquidationPool {
	liquidateExchangeRatesPoolList := make(map[string]*LiquidationPool)
	temp := stateDB.trie.NodeIterator(GetPortalLiquidationPoolPrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		key := it.Key
		keyHash, _ := common.Hash{}.NewHash(key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		object := NewLiquidationPool()
		err := json.Unmarshal(newValue, object)
		if err != nil {
			panic("wrong expect type")
		}
		liquidateExchangeRatesPoolList[keyHash.String()] = object
	}

	return liquidateExchangeRatesPoolList
}

func (stateDB *StateDB) getFinalExchangeRatesState() (*FinalExchangeRatesState, error) {
	key := GeneratePortalFinalExchangeRatesStateObjectKey()
	finalRates, err := stateDB.getStateObject(PortalFinalExchangeRatesStateObjectType, key)
	if err != nil {
		return nil, err
	}
	if finalRates != nil {
		return finalRates.GetValue().(*FinalExchangeRatesState), nil
	}
	return NewFinalExchangeRatesState(), nil
}

//B
func (stateDB *StateDB) getAllWaitingRedeemRequest() map[string]*RedeemRequest {
	waitingRedeemRequests := make(map[string]*RedeemRequest)
	temp := stateDB.trie.NodeIterator(GetWaitingRedeemRequestPrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		key := it.Key
		keyHash, _ := common.Hash{}.NewHash(key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		wr := NewRedeemRequest()
		err := json.Unmarshal(newValue, wr)
		if err != nil {
			panic("wrong expect type")
		}
		waitingRedeemRequests[keyHash.String()] = wr
	}
	return waitingRedeemRequests
}

func (stateDB *StateDB) getAllMatchedRedeemRequest() map[string]*RedeemRequest {
	matchedRedeemRequests := make(map[string]*RedeemRequest)
	temp := stateDB.trie.NodeIterator(GetMatchedRedeemRequestPrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		key := it.Key
		keyHash, _ := common.Hash{}.NewHash(key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		wr := NewRedeemRequest()
		err := json.Unmarshal(newValue, wr)
		if err != nil {
			panic("wrong expect type")
		}
		matchedRedeemRequests[keyHash.String()] = wr
	}
	return matchedRedeemRequests
}

func (stateDB *StateDB) getAllCustodianStatePool() map[string]*CustodianState {
	custodians := make(map[string]*CustodianState)
	temp := stateDB.trie.NodeIterator(GetPortalCustodianStatePrefix())
	it := trie.NewIterator(temp)
	for it.Next() {
		key := it.Key
		keyHash, _ := common.Hash{}.NewHash(key)
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		cus := NewCustodianState()
		err := json.Unmarshal(newValue, cus)
		if err != nil {
			panic("wrong expect type")
		}
		custodians[keyHash.String()] = cus
	}
	return custodians
}

func (stateDB *StateDB) getPortalRewards(beaconHeight uint64) []*PortalRewardInfo {
	portalRewards := make([]*PortalRewardInfo, 0)
	temp := stateDB.trie.NodeIterator(GetPortalRewardInfoStatePrefix(beaconHeight))
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		rewardInfo := NewPortalRewardInfo()
		err := json.Unmarshal(newValue, rewardInfo)
		if err != nil {
			panic("wrong expect type")
		}
		portalRewards = append(portalRewards, rewardInfo)
	}
	return portalRewards
}

func (stateDB *StateDB) getPortalStatusByKey(key common.Hash) (*PortalStatusState, bool, error) {
	portalStatusState, err := stateDB.getStateObject(PortalStatusObjectType, key)
	if err != nil {
		return nil, false, err
	}
	if portalStatusState != nil {
		return portalStatusState.GetValue().(*PortalStatusState), true, nil
	}
	return NewPortalStatusState(), false, nil
}

func (stateDB *StateDB) getLockedCollateralState() (*LockedCollateralState, bool, error) {
	key := GenerateLockedCollateralStateObjectKey()
	lockedCollateralState, err := stateDB.getStateObject(LockedCollateralStateObjectType, key)
	if err != nil {
		return nil, false, err
	}

	if lockedCollateralState != nil {
		return lockedCollateralState.GetValue().(*LockedCollateralState), true, nil
	}
	return NewLockedCollateralState(), false, nil
}

// ================================= Feature reward OBJECT =======================================
func (stateDB *StateDB) getFeatureRewardByFeatureName(featureName string, epoch uint64) (*RewardFeatureState, bool, error) {
	key := GenerateRewardFeatureStateObjectKey(featureName, epoch)
	rewardFeatureState, err := stateDB.getStateObject(RewardFeatureStateObjectType, key)
	if err != nil {
		return nil, false, err
	}

	if rewardFeatureState != nil {
		return rewardFeatureState.GetValue().(*RewardFeatureState), true, nil
	}
	return NewRewardFeatureState(), false, nil
}

func (stateDB *StateDB) getAllFeatureRewards(epoch uint64) (*RewardFeatureState, bool, error) {
	result := NewRewardFeatureState()

	temp := stateDB.trie.NodeIterator(GetRewardFeatureStatePrefix(epoch))
	it := trie.NewIterator(temp)
	for it.Next() {
		value := it.Value
		newValue := make([]byte, len(value))
		copy(newValue, value)
		rewardFeature := NewRewardFeatureState()
		err := json.Unmarshal(newValue, rewardFeature)
		if err != nil {
			panic("wrong expect type")
		}

		for tokenID, amount := range rewardFeature.totalRewards {
			result.AddTotalRewards(tokenID, amount)
		}
	}
	return result, true, nil
}

func (stateDB *StateDB) getAllStaker(ids []int) int {
	allStaker := []*CommitteeState{}

	// Current Beacon Validator
	prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, -1)
	resCurrentValidator := stateDB.iterateWithCommitteeState(prefixCurrentValidator)
	allStaker = append(allStaker, resCurrentValidator...)
	// Substitute Beacon Validator
	prefixSubstituteValidator := GetCommitteePrefixWithRole(SubstituteValidator, -1)
	resSubstituteValidator := stateDB.iterateWithCommitteeState(prefixSubstituteValidator)
	allStaker = append(allStaker, resSubstituteValidator...)

	for _, shardID := range ids {
		// Current Shard Validator
		prefixCurrentValidator := GetCommitteePrefixWithRole(CurrentValidator, shardID)
		resCurrentValidator := stateDB.iterateWithCommitteeState(prefixCurrentValidator)
		allStaker = append(allStaker, resCurrentValidator...)
		// Substitute Shard sValidator
		prefixSubstituteValidator := GetCommitteePrefixWithRole(SubstituteValidator, shardID)
		resSubstituteValidator := stateDB.iterateWithCommitteeState(prefixSubstituteValidator)
		allStaker = append(allStaker, resSubstituteValidator...)
	}
	// next epoch candidate
	prefixNextEpochCandidate := GetCommitteePrefixWithRole(NextEpochShardCandidate, -2)
	resNextEpochCandidate := stateDB.iterateWithCommitteeState(prefixNextEpochCandidate)
	allStaker = append(allStaker, resNextEpochCandidate...)
	// current epoch candidate
	prefixCurrentEpochCandidate := GetCommitteePrefixWithRole(CurrentEpochShardCandidate, -2)
	resCurrentEpochCandidate := stateDB.iterateWithCommitteeState(prefixCurrentEpochCandidate)
	allStaker = append(allStaker, resCurrentEpochCandidate...)

	// next epoch candidate
	prefixNextEpochBeaconCandidate := GetCommitteePrefixWithRole(NextEpochBeaconCandidate, -2)
	resNextEpochBeaconCandidate := stateDB.iterateWithCommitteeState(prefixNextEpochBeaconCandidate)
	allStaker = append(allStaker, resNextEpochBeaconCandidate...)
	// current epoch candidate
	prefixCurrentEpochBeaconCandidate := GetCommitteePrefixWithRole(CurrentEpochBeaconCandidate, -2)
	resCurrentEpochBeaconCandidate := stateDB.iterateWithCommitteeState(prefixCurrentEpochBeaconCandidate)
	allStaker = append(allStaker, resCurrentEpochBeaconCandidate...)
	return len(allStaker)
}
