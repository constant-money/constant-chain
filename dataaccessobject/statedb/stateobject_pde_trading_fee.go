package statedb

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
)

type PDETradingFeeState struct {
	token1ID           string
	token2ID           string
	contributorAddress string
	amount             uint64
}

func (s PDETradingFeeState) Token1ID() string {
	return s.token1ID
}

func (s *PDETradingFeeState) SetToken1ID(token1ID string) {
	s.token1ID = token1ID
}

func (s PDETradingFeeState) Token2ID() string {
	return s.token2ID
}

func (s *PDETradingFeeState) SetToken2ID(token2ID string) {
	s.token2ID = token2ID
}

func (s PDETradingFeeState) Amount() uint64 {
	return s.amount
}

func (s *PDETradingFeeState) SetAmount(amount uint64) {
	s.amount = amount
}

func (s *PDETradingFeeState) ContributorAddress() string {
	return s.contributorAddress
}

func (s PDETradingFeeState) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		Token1ID           string
		Token2ID           string
		Amount             uint64
		ContributorAddress string
	}{
		Token1ID:           s.token1ID,
		Token2ID:           s.token2ID,
		Amount:             s.amount,
		ContributorAddress: s.contributorAddress,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (s *PDETradingFeeState) UnmarshalJSON(data []byte) error {
	temp := struct {
		Token1ID           string
		Token2ID           string
		Amount             uint64
		ContributorAddress string
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	s.token1ID = temp.Token1ID
	s.token2ID = temp.Token2ID
	s.amount = temp.Amount
	s.contributorAddress = temp.ContributorAddress
	return nil
}

func NewPDETradingFeeState() *PDETradingFeeState {
	return &PDETradingFeeState{}
}

func NewPDETradingFeeStateWithValue(token1ID string, token2ID string, contributorAddress string, amount uint64) *PDETradingFeeState {
	return &PDETradingFeeState{token1ID: token1ID, token2ID: token2ID, amount: amount, contributorAddress: contributorAddress}
}

type PDETradingFeeObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version            int
	pdeTradingFeeHash  common.Hash
	pdeTradingFeeState *PDETradingFeeState
	objectType         int
	deleted            bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newPDETradingFeeObject(db *StateDB, hash common.Hash) *PDETradingFeeObject {
	return &PDETradingFeeObject{
		version:            defaultVersion,
		db:                 db,
		pdeTradingFeeHash:  hash,
		pdeTradingFeeState: NewPDETradingFeeState(),
		objectType:         PDETradingFeeObjectType,
		deleted:            false,
	}
}

func newPDETradingFeeObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*PDETradingFeeObject, error) {
	var newPDETradingFeeState = NewPDETradingFeeState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, newPDETradingFeeState)
		if err != nil {
			return nil, err
		}
	} else {
		newPDETradingFeeState, ok = data.(*PDETradingFeeState)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidPDETradingFeeStateType, reflect.TypeOf(data))
		}
	}
	return &PDETradingFeeObject{
		version:            defaultVersion,
		pdeTradingFeeHash:  key,
		pdeTradingFeeState: newPDETradingFeeState,
		db:                 db,
		objectType:         PDETradingFeeObjectType,
		deleted:            false,
	}, nil
}

func GeneratePDETradingFeeObjectKey(token1ID, token2ID, contributorAddress string) common.Hash {
	prefixHash := GetPDETradingFeePrefix()
	valueHash := common.HashH([]byte(token1ID + token2ID + contributorAddress))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t PDETradingFeeObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *PDETradingFeeObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t PDETradingFeeObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *PDETradingFeeObject) SetValue(data interface{}) error {
	newPDETradingFeeState, ok := data.(*PDETradingFeeState)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidPDETradingFeeStateType, reflect.TypeOf(data))
	}
	t.pdeTradingFeeState = newPDETradingFeeState
	return nil
}

func (t PDETradingFeeObject) GetValue() interface{} {
	return t.pdeTradingFeeState
}

func (t PDETradingFeeObject) GetValueBytes() []byte {
	pdeTradingFeeState, ok := t.GetValue().(*PDETradingFeeState)
	if !ok {
		panic("wrong expected value type")
	}
	value, err := json.Marshal(pdeTradingFeeState)
	if err != nil {
		panic("failed to marshal pde trading fee state")
	}
	return value
}

func (t PDETradingFeeObject) GetHash() common.Hash {
	return t.pdeTradingFeeHash
}

func (t PDETradingFeeObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *PDETradingFeeObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *PDETradingFeeObject) Reset() bool {
	t.pdeTradingFeeState = NewPDETradingFeeState()
	return true
}

func (t PDETradingFeeObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t PDETradingFeeObject) IsEmpty() bool {
	temp := NewPDETradingFeeState()
	return reflect.DeepEqual(temp, t.pdeTradingFeeState) || t.pdeTradingFeeState == nil
}
