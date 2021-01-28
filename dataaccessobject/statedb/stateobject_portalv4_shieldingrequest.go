package statedb

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
)

type ShieldingRequest struct {
	incAddress string
	amount     uint64
}

func NewShieldingRequest() *ShieldingRequest {
	return &ShieldingRequest{}
}

func NewShieldingRequestWithValue(
	incAddress string,
	amount uint64,
) *ShieldingRequest {
	return &ShieldingRequest{
		incAddress: incAddress,
		amount:     amount,
	}
}

func (pr *ShieldingRequest) GetIncAddress() string {
	return pr.incAddress
}

func (pr *ShieldingRequest) SetIncAddress(incAddress string) {
	pr.incAddress = incAddress
}

func (pr *ShieldingRequest) GetAmount() uint64 {
	return pr.amount
}

func (pr *ShieldingRequest) SetAmount(amount uint64) {
	pr.amount = amount
}

func (pr *ShieldingRequest) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		IncAddress string
		Amount     uint64
	}{
		IncAddress: pr.incAddress,
		Amount:     pr.amount,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (pr *ShieldingRequest) UnmarshalJSON(data []byte) error {
	temp := struct {
		IncAddress string
		Amount     uint64
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	pr.incAddress = temp.IncAddress
	pr.amount = temp.Amount
	return nil
}

type ShieldingRequestsState struct {
	requests map[string]*ShieldingRequest // map key: external tx hash => Shielding request information
}

func NewShieldingRequestsState() *ShieldingRequestsState {
	return &ShieldingRequestsState{
		requests: map[string]*ShieldingRequest{},
	}
}

func NewShieldingRequestsStateWithValue(r map[string]*ShieldingRequest) *ShieldingRequestsState {
	return &ShieldingRequestsState{
		requests: r,
	}
}

func (ps *ShieldingRequestsState) SetShieldingRequest(externalTxHash string, request *ShieldingRequest) {
	ps.requests[externalTxHash] = request
}

func (ps ShieldingRequestsState) GetShieldingRequests() map[string]*ShieldingRequest {
	return ps.requests
}

func (ps *ShieldingRequestsState) SetShieldingRequests(r map[string]*ShieldingRequest) {
	ps.requests = r
}

func (ps *ShieldingRequestsState) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		Requests map[string]*ShieldingRequest
	}{
		Requests: ps.requests,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (ps *ShieldingRequestsState) UnmarshalJSON(data []byte) error {
	temp := struct {
		Requests map[string]*ShieldingRequest
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	ps.requests = temp.Requests
	return nil
}

type ShieldingRequestsStateObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version                    int
	ShieldingRequestsStateHash common.Hash
	ShieldingRequestsState     *ShieldingRequestsState
	objectType                 int
	deleted                    bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newShieldingRequestsStateObject(db *StateDB, hash common.Hash) *ShieldingRequestsStateObject {
	return &ShieldingRequestsStateObject{
		version:                    defaultVersion,
		db:                         db,
		ShieldingRequestsStateHash: hash,
		ShieldingRequestsState:     NewShieldingRequestsState(),
		objectType:                 PortalShieldingRequestsStateObjectType,
		deleted:                    false,
	}
}

func newShieldingRequestsStateObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*ShieldingRequestsStateObject, error) {
	var shieldingRequestsState = NewShieldingRequestsState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, shieldingRequestsState)
		if err != nil {
			return nil, err
		}
	} else {
		shieldingRequestsState, ok = data.(*ShieldingRequestsState)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidPortalShieldingRequestsStateType, reflect.TypeOf(data))
		}
	}
	return &ShieldingRequestsStateObject{
		version:                    defaultVersion,
		ShieldingRequestsStateHash: key,
		ShieldingRequestsState:     shieldingRequestsState,
		db:                         db,
		objectType:                 PortalShieldingRequestsStateObjectType,
		deleted:                    false,
	}, nil
}

func GenerateShieldingRequestsStateObjectKey(tokenIDStr string) common.Hash {
	prefixHash := PortalRequestPTokenStatusPrefixV4()
	valueHash := common.HashH([]byte(tokenIDStr))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t ShieldingRequestsStateObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *ShieldingRequestsStateObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t ShieldingRequestsStateObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *ShieldingRequestsStateObject) SetValue(data interface{}) error {
	newShieldingRequestsState, ok := data.(*ShieldingRequestsState)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidPortalShieldingRequestsStateType, reflect.TypeOf(data))
	}
	t.ShieldingRequestsState = newShieldingRequestsState
	return nil
}

func (t ShieldingRequestsStateObject) GetValue() interface{} {
	return t.ShieldingRequestsState
}

func (t ShieldingRequestsStateObject) GetValueBytes() []byte {
	ShieldingRequestsState, ok := t.GetValue().(*ShieldingRequestsState)
	if !ok {
		panic("wrong expected value type")
	}
	value, err := json.Marshal(ShieldingRequestsState)
	if err != nil {
		panic("failed to marshal multisigWallet state")
	}
	return value
}

func (t ShieldingRequestsStateObject) GetHash() common.Hash {
	return t.ShieldingRequestsStateHash
}

func (t ShieldingRequestsStateObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *ShieldingRequestsStateObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *ShieldingRequestsStateObject) Reset() bool {
	t.ShieldingRequestsState = NewShieldingRequestsState()
	return true
}

func (t ShieldingRequestsStateObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t ShieldingRequestsStateObject) IsEmpty() bool {
	temp := NewShieldingRequestsState()
	return reflect.DeepEqual(temp, t.ShieldingRequestsState) || t.ShieldingRequestsState == nil
}
