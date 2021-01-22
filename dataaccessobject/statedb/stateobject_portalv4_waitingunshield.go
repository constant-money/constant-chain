package statedb

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"reflect"
)

type Unshield struct {
	remoteAddress string
	amount        uint64
}

type WaitingUnshield struct {
	unshields map[string]*Unshield // txid : Unshield struct
}

func (rq *WaitingUnshield) GetUnshield(unshiedID string) *Unshield {
	return rq.unshields[unshiedID]
}

func (rq *WaitingUnshield) GetUnshields() map[string]*Unshield {
	return rq.unshields
}

func (rq *WaitingUnshield) SetUnshield(unshiedID string, unshield *Unshield) {
	rq.unshields[unshiedID] = unshield
}

func (rq WaitingUnshield) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		Unshields map[string]*Unshield
	}{
		Unshields: rq.unshields,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (rq *WaitingUnshield) UnmarshalJSON(data []byte) error {
	temp := struct {
		Unshields map[string]*Unshield
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	rq.unshields = temp.Unshields
	return nil
}

func (us *Unshield) GetRemoteAddress() string {
	return us.remoteAddress
}

func (us *Unshield) GetAmount() uint64 {
	return us.amount
}

func (us *Unshield) SetRemoteAddress(remoteAddress string) {
	us.remoteAddress = remoteAddress
}

func (us *Unshield) SetAmount(amount uint64) {
	us.amount = amount
}

func (us Unshield) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		RemoteAddress string
		Amount        uint64
	}{
		RemoteAddress: us.remoteAddress,
		Amount:        us.amount,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (us *Unshield) UnmarshalJSON(data []byte) error {
	temp := struct {
		RemoteAddress string
		Amount        uint64
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	us.remoteAddress = temp.RemoteAddress
	us.amount = temp.Amount
	return nil
}

func NewLWaitingUnshieldState() *WaitingUnshield {
	return &WaitingUnshield{
		unshields: map[string]*Unshield{},
	}
}

func NewWaitingUnshieldStateWithValue(
	unshieldsInput map[string]*Unshield,
) *WaitingUnshield {
	return &WaitingUnshield{
		unshields: unshieldsInput,
	}
}

func NewUnshieldRequestDetailWithValue(
	remoteAddress string,
	amount uint64) *Unshield {
	return &Unshield{
		remoteAddress: remoteAddress,
		amount:        amount,
	}
}

func NewUnshield() *Unshield {
	return &Unshield{}
}

type WaitingUnshieldObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version                    int
	waitingWaitingUnshieldHash common.Hash
	waitingWaitingUnshield     *WaitingUnshield
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
		waitingWaitingUnshieldHash: hash,
		waitingWaitingUnshield:     NewLWaitingUnshieldState(),
		objectType:                 PortalWaitingUnshieldObjectType,
		deleted:                    false,
	}
}

func newWaitingUnshieldObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*WaitingUnshieldObject, error) {
	var content = NewLWaitingUnshieldState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, content)
		if err != nil {
			return nil, err
		}
	} else {
		content, ok = data.(*WaitingUnshield)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidUnshieldRequestType, reflect.TypeOf(data))
		}
	}
	return &WaitingUnshieldObject{
		version:                    defaultVersion,
		waitingWaitingUnshieldHash: key,
		waitingWaitingUnshield:     content,
		db:                         db,
		objectType:                 PortalWaitingUnshieldObjectType,
		deleted:                    false,
	}, nil
}

func GenerateWaitingWaitingUnshieldObjectKey(redeemID string) common.Hash {
	prefixHash := GetWaitingUnshieldRequestPrefix()
	valueHash := common.HashH([]byte(redeemID))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func GenerateMatchedWaitingUnshieldObjectKey(redeemID string) common.Hash {
	prefixHash := GetWaitingUnshieldRequestPrefix()
	valueHash := common.HashH([]byte(redeemID))
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
	WaitingUnshield, ok := data.(*WaitingUnshield)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidUnshieldRequestType, reflect.TypeOf(data))
	}
	t.waitingWaitingUnshield = WaitingUnshield
	return nil
}

func (t WaitingUnshieldObject) GetValue() interface{} {
	return t.waitingWaitingUnshield
}

func (t WaitingUnshieldObject) GetValueBytes() []byte {
	WaitingUnshield, ok := t.GetValue().(*WaitingUnshield)
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
	return t.waitingWaitingUnshieldHash
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
	t.waitingWaitingUnshield = NewLWaitingUnshieldState()
	return true
}

func (t WaitingUnshieldObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t WaitingUnshieldObject) IsEmpty() bool {
	temp := NewLWaitingUnshieldState()
	return reflect.DeepEqual(temp, t.waitingWaitingUnshield) || t.waitingWaitingUnshield == nil
}
