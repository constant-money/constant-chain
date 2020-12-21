package statedb

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"reflect"
)

type SerialNumberState struct {
	tokenID      common.Hash
	shardID      byte
	serialNumber []byte
}

func (s SerialNumberState) SerialNumber() []byte {
	return s.serialNumber
}

func (s *SerialNumberState) SetSerialNumber(serialNumber []byte) {
	s.serialNumber = serialNumber
}

func (s SerialNumberState) ShardID() byte {
	return s.shardID
}

func (s *SerialNumberState) SetShardID(shardID byte) {
	s.shardID = shardID
}

func (s SerialNumberState) TokenID() common.Hash {
	return s.tokenID
}

func (s *SerialNumberState) SetTokenID(tokenID common.Hash) {
	s.tokenID = tokenID
}

func NewSerialNumberState() *SerialNumberState {
	return &SerialNumberState{}
}

func NewSerialNumberStateWithValue(tokenID common.Hash, shardID byte, serialNumber []byte) *SerialNumberState {
	return &SerialNumberState{tokenID: tokenID, shardID: shardID, serialNumber: serialNumber}
}

func (s SerialNumberState) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		TokenID      common.Hash
		ShardID      byte
		SerialNumber []byte
	}{
		TokenID:      s.tokenID,
		ShardID:      s.shardID,
		SerialNumber: s.serialNumber,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (s *SerialNumberState) UnmarshalJSON(data []byte) error {
	temp := struct {
		TokenID      common.Hash
		ShardID      byte
		SerialNumber []byte
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	s.tokenID = temp.TokenID
	s.shardID = temp.ShardID
	s.serialNumber = temp.SerialNumber
	return nil
}

type SerialNumberObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version           int
	serialNumberHash  common.Hash
	serialNumberState *SerialNumberState
	objectType        int
	deleted           bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newSerialNumberObject(db *StateDB, hash common.Hash) *SerialNumberObject {
	return &SerialNumberObject{
		version:           defaultVersion,
		db:                db,
		serialNumberHash:  hash,
		serialNumberState: NewSerialNumberState(),
		objectType:        SerialNumberObjectType,
		deleted:           false,
	}
}

func newSerialNumberObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*SerialNumberObject, error) {
	var newSerialNumberState = NewSerialNumberState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, newSerialNumberState)
		if err != nil {
			return nil, err
		}
	} else {
		newSerialNumberState, ok = data.(*SerialNumberState)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidSerialNumberStateType, reflect.TypeOf(data))
		}
	}
	return &SerialNumberObject{
		version:           defaultVersion,
		serialNumberHash:  key,
		serialNumberState: newSerialNumberState,
		db:                db,
		objectType:        SerialNumberObjectType,
		deleted:           false,
	}, nil
}

func GenerateSerialNumberObjectKey(tokenID common.Hash, shardID byte, serialNumber []byte) common.Hash {
	prefixHash := GetSerialNumberPrefix(tokenID, shardID)
	valueHash := common.HashH(serialNumber)
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (s SerialNumberObject) GetVersion() int {
	return s.version
}

// setError remembers the first non-nil error it is called with.
func (s *SerialNumberObject) SetError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s SerialNumberObject) GetTrie(db DatabaseAccessWarper) Trie {
	return s.trie
}

func (s *SerialNumberObject) SetValue(data interface{}) error {
	newSerialNumberState, ok := data.(*SerialNumberState)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidSerialNumberStateType, reflect.TypeOf(data))
	}
	s.serialNumberState = newSerialNumberState
	return nil
}

func (s SerialNumberObject) GetValue() interface{} {
	return s.serialNumberState
}

func (s SerialNumberObject) GetValueBytes() []byte {
	serialNumberState, ok := s.GetValue().(*SerialNumberState)
	if !ok {
		panic("wrong expected value type")
	}
	serialNumberStateBytes, err := json.Marshal(serialNumberState)
	if err != nil {
		panic(err.Error())
	}
	return serialNumberStateBytes
}

func (s SerialNumberObject) GetHash() common.Hash {
	return s.serialNumberHash
}

func (s SerialNumberObject) GetType() int {
	return s.objectType
}

// MarkDelete will delete an object in trie
func (s *SerialNumberObject) MarkDelete() {
	s.deleted = true
}

// Reset serial number into default
func (s *SerialNumberObject) Reset() bool {
	s.serialNumberState = NewSerialNumberState()
	return true
}

func (s SerialNumberObject) IsDeleted() bool {
	return s.deleted
}

// empty value or not
func (s SerialNumberObject) IsEmpty() bool {
	temp := NewSerialNumberState()
	return reflect.DeepEqual(temp, s.serialNumberState) || s.serialNumberState == nil
}
