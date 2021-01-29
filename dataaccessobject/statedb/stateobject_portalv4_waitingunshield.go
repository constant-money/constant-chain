package statedb

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
)

type WaitingUnshieldRequest struct {
	unshieldID string
	remoteAddress string
	amount        uint64
	beaconHeight uint64
}

func (us *WaitingUnshieldRequest) GetRemoteAddress() string {
	return us.remoteAddress
}

func (us *WaitingUnshieldRequest) SetRemoteAddress(remoteAddress string) {
	us.remoteAddress = remoteAddress
}

func (us *WaitingUnshieldRequest) GetAmount() uint64 {
	return us.amount
}

func (us *WaitingUnshieldRequest) SetAmount(amount uint64) {
	us.amount = amount
}

func (us *WaitingUnshieldRequest) GetUnshieldID() string {
	return us.unshieldID
}

func (us *WaitingUnshieldRequest) SetUnshieldID(unshieldID string) {
	us.unshieldID = unshieldID
}

func (us *WaitingUnshieldRequest) GetBeaconHeight() uint64 {
	return us.beaconHeight
}

func (us *WaitingUnshieldRequest) SetBeaconHeight(beaconHeight uint64) {
	us.beaconHeight = beaconHeight
}

func (us WaitingUnshieldRequest) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		RemoteAddress string
		Amount        uint64
		UnshieldID string
		BeaconHeight uint64
	}{
		RemoteAddress: us.remoteAddress,
		Amount:        us.amount,
		UnshieldID: us.unshieldID,
		BeaconHeight : us.beaconHeight,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (us *WaitingUnshieldRequest) UnmarshalJSON(data []byte) error {
	temp := struct {
		RemoteAddress string
		Amount        uint64
		UnshieldID string
		BeaconHeight uint64
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	us.remoteAddress = temp.RemoteAddress
	us.amount = temp.Amount
	return nil
}

func NewWaitingUnshieldRequestStateWithValue(
	remoteAddress string,
	amount uint64,
	unshieldID string,
	beaconHeight uint64) *WaitingUnshieldRequest {
	return &WaitingUnshieldRequest{
		remoteAddress: remoteAddress,
		amount:        amount,
		unshieldID: unshieldID,
		beaconHeight: beaconHeight,
	}
}

func NewWaitingUnshieldRequestState() *WaitingUnshieldRequest {
	return &WaitingUnshieldRequest{}
}

type WaitingUnshieldObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version                    int
	waitingUnshieldRequestHash common.Hash
	waitingUnshieldRequest     *WaitingUnshieldRequest
	objectType                 int
	deleted                    bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newWaitingUnshieldObject(db *StateDB, hash common.Hash) *WaitingUnshieldObject {
	return &WaitingUnshieldObject{
		version:                    defaultVersion,
		db:                         db,
		waitingUnshieldRequestHash: hash,
		waitingUnshieldRequest:     NewWaitingUnshieldRequestState(),
		objectType:                 PortalWaitingUnshieldObjectType,
		deleted:                    false,
	}
}

func newWaitingUnshieldObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*WaitingUnshieldObject, error) {
	var content = NewWaitingUnshieldRequestState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, content)
		if err != nil {
			return nil, err
		}
	} else {
		content, ok = data.(*WaitingUnshieldRequest)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidUnshieldRequestType, reflect.TypeOf(data))
		}
	}
	return &WaitingUnshieldObject{
		version:                    defaultVersion,
		waitingUnshieldRequestHash: key,
		waitingUnshieldRequest:     content,
		db:                         db,
		objectType:                 PortalWaitingUnshieldObjectType,
		deleted:                    false,
	}, nil
}

func GenerateWaitingUnshieldRequestObjectKey(tokenID string, unshieldID string) common.Hash {
	prefixHash := GetWaitingUnshieldRequestPrefix(tokenID)
	valueHash := common.HashH([]byte(unshieldID))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t WaitingUnshieldObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *WaitingUnshieldObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t WaitingUnshieldObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *WaitingUnshieldObject) SetValue(data interface{}) error {
	WaitingUnshield, ok := data.(*WaitingUnshieldRequest)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidUnshieldRequestType, reflect.TypeOf(data))
	}
	t.waitingUnshieldRequest = WaitingUnshield
	return nil
}

func (t WaitingUnshieldObject) GetValue() interface{} {
	return t.waitingUnshieldRequest
}

func (t WaitingUnshieldObject) GetValueBytes() []byte {
	WaitingUnshield, ok := t.GetValue().(*WaitingUnshieldRequest)
	if !ok {
		panic("wrong expected value type")
	}
	value, err := json.Marshal(WaitingUnshield)
	if err != nil {
		panic("failed to marshal redeem request")
	}
	return value
}

func (t WaitingUnshieldObject) GetHash() common.Hash {
	return t.waitingUnshieldRequestHash
}

func (t WaitingUnshieldObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *WaitingUnshieldObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *WaitingUnshieldObject) Reset() bool {
	t.waitingUnshieldRequest = NewWaitingUnshieldRequestState()
	return true
}

func (t WaitingUnshieldObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t WaitingUnshieldObject) IsEmpty() bool {
	temp := NewWaitingUnshieldRequestState()
	return reflect.DeepEqual(temp, t.waitingUnshieldRequest) || t.waitingUnshieldRequest == nil
}
