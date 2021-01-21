package statedb

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
)

type UTXO struct {
	txHash       string
	outputIdx    []int
	outputAmount []uint64
}

type MultisigWalletState struct {
	listUTXO []UTXO
}

func NewMultisigWalletState() *MultisigWalletState {
	return &MultisigWalletState{
		listUTXO: []UTXO{},
	}
}

func NewMultisigWalletStateWithValue(listUTXO []UTXO) *MultisigWalletState {
	return &MultisigWalletState{
		listUTXO: listUTXO,
	}
}

func (ws MultisigWalletState) GetListUTXO() []UTXO {
	return ws.listUTXO
}

func (ws MultisigWalletState) SetListUTXO(l []UTXO) {
	ws.listUTXO = l
}

func (ws *MultisigWalletState) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (ws *MultisigWalletState) UnmarshalJSON(data []byte) error {
	return nil
}

type MultisigWalletStateObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version                 int
	multisigWalletStateHash common.Hash
	multisigWalletState     *MultisigWalletState
	objectType              int
	deleted                 bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newMultisigWalletStateObject(db *StateDB, hash common.Hash) *MultisigWalletStateObject {
	return &MultisigWalletStateObject{
		version:                 defaultVersion,
		db:                      db,
		multisigWalletStateHash: hash,
		multisigWalletState:     NewMultisigWalletState(),
		objectType:              PortalMultisigWalletObjectType,
		deleted:                 false,
	}
}

func newMultisigWalletObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*MultisigWalletStateObject, error) {
	var multisigWalletState = NewMultisigWalletState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, multisigWalletState)
		if err != nil {
			return nil, err
		}
	} else {
		multisigWalletState, ok = data.(*MultisigWalletState)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidPortalMultisigWalletStateType, reflect.TypeOf(data))
		}
	}
	return &MultisigWalletStateObject{
		version:                 defaultVersion,
		multisigWalletStateHash: key,
		multisigWalletState:     multisigWalletState,
		db:                      db,
		objectType:              PortalMultisigWalletObjectType,
		deleted:                 false,
	}, nil
}

func GenerateMultisigWalletStateObjectKey(walletAddress string, tokenIDStr string) common.Hash {
	prefixHash := GetPortalMultisigWalletStatePrefix()
	valueHash := common.HashH([]byte(walletAddress + "-" + tokenIDStr))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t MultisigWalletStateObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *MultisigWalletStateObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t MultisigWalletStateObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *MultisigWalletStateObject) SetValue(data interface{}) error {
	newMultisigWalletState, ok := data.(*MultisigWalletState)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidPortalMultisigWalletStateType, reflect.TypeOf(data))
	}
	t.multisigWalletState = newMultisigWalletState
	return nil
}

func (t MultisigWalletStateObject) GetValue() interface{} {
	return t.multisigWalletState
}

func (t MultisigWalletStateObject) GetValueBytes() []byte {
	multisigWalletState, ok := t.GetValue().(*MultisigWalletState)
	if !ok {
		panic("wrong expected value type")
	}
	value, err := json.Marshal(multisigWalletState)
	if err != nil {
		panic("failed to marshal multisigWallet state")
	}
	return value
}

func (t MultisigWalletStateObject) GetHash() common.Hash {
	return t.multisigWalletStateHash
}

func (t MultisigWalletStateObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *MultisigWalletStateObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *MultisigWalletStateObject) Reset() bool {
	t.multisigWalletState = NewMultisigWalletState()
	return true
}

func (t MultisigWalletStateObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t MultisigWalletStateObject) IsEmpty() bool {
	temp := NewMultisigWalletState()
	return reflect.DeepEqual(temp, t.multisigWalletState) || t.multisigWalletState == nil
}
