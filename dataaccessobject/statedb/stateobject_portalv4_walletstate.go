package statedb

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
)

type UTXO struct {
	txHash       string
	outputIdx    int
	outputAmount uint64
}

type MultisigWalletsState struct {
	wallets map[string][]UTXO
}

func NewMultisigWalletsState() *MultisigWalletsState {
	return &MultisigWalletsState{
		wallets: map[string][]UTXO{},
	}
}

func NewMultisigWalletsStateWithValue(w map[string][]UTXO) *MultisigWalletsState {
	return &MultisigWalletsState{
		wallets: w,
	}
}

func (ws MultisigWalletsState) GetWallets() map[string][]UTXO {
	return ws.wallets
}

func (ws MultisigWalletsState) SetWallets(w map[string][]UTXO) {
	ws.wallets = w
}

func (ws *MultisigWalletsState) MarshalJSON() ([]byte, error) {
	type TmpUTXO struct {
		TxHash       string
		OutputIdx    int
		OutputAmount uint64
	}
	temp := struct {
		Wallets map[string][]TmpUTXO
	}{}

	for wallet_address, list_utxo := range ws.wallets {
		temp.Wallets[wallet_address] = []TmpUTXO{}
		for _, utxo := range list_utxo {
			temp.Wallets[wallet_address] = append(temp.Wallets[wallet_address], TmpUTXO{
				TxHash:       utxo.txHash,
				OutputIdx:    utxo.outputIdx,
				OutputAmount: utxo.outputAmount,
			})
		}
	}
	data, err := json.Marshal(temp)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (ws *MultisigWalletsState) UnmarshalJSON(data []byte) error {
	type TmpUTXO struct {
		TxHash       string
		OutputIdx    int
		OutputAmount uint64
	}
	temp := struct {
		Wallets map[string][]TmpUTXO
	}{}

	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}

	ws.wallets = map[string][]UTXO{}
	for wallet_address, list_utxo := range temp.Wallets {
		ws.wallets[wallet_address] = []UTXO{}
		for _, utxo := range list_utxo {
			ws.wallets[wallet_address] = append(ws.wallets[wallet_address], UTXO{
				txHash:       utxo.TxHash,
				outputIdx:    utxo.OutputIdx,
				outputAmount: utxo.OutputAmount,
			})
		}
	}

	return nil
}

type MultisigWalletsStateObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version                  int
	MultisigWalletsStateHash common.Hash
	MultisigWalletsState     *MultisigWalletsState
	objectType               int
	deleted                  bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newMultisigWalletsStateObject(db *StateDB, hash common.Hash) *MultisigWalletsStateObject {
	return &MultisigWalletsStateObject{
		version:                  defaultVersion,
		db:                       db,
		MultisigWalletsStateHash: hash,
		MultisigWalletsState:     NewMultisigWalletsState(),
		objectType:               PortalMultisigWalletObjectType,
		deleted:                  false,
	}
}

func newMultisigWalletObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*MultisigWalletsStateObject, error) {
	var multisigWalletsState = NewMultisigWalletsState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, multisigWalletsState)
		if err != nil {
			return nil, err
		}
	} else {
		multisigWalletsState, ok = data.(*MultisigWalletsState)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidPortalMultisigWalletsStateType, reflect.TypeOf(data))
		}
	}
	return &MultisigWalletsStateObject{
		version:                  defaultVersion,
		MultisigWalletsStateHash: key,
		MultisigWalletsState:     multisigWalletsState,
		db:                       db,
		objectType:               PortalMultisigWalletObjectType,
		deleted:                  false,
	}, nil
}

func GenerateMultisigWalletsStateObjectKey(tokenIDStr string) common.Hash {
	prefixHash := GetPortalMultisigWalletsStatePrefix()
	valueHash := common.HashH([]byte(tokenIDStr))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t MultisigWalletsStateObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *MultisigWalletsStateObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t MultisigWalletsStateObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *MultisigWalletsStateObject) SetValue(data interface{}) error {
	newMultisigWalletsState, ok := data.(*MultisigWalletsState)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidPortalMultisigWalletsStateType, reflect.TypeOf(data))
	}
	t.MultisigWalletsState = newMultisigWalletsState
	return nil
}

func (t MultisigWalletsStateObject) GetValue() interface{} {
	return t.MultisigWalletsState
}

func (t MultisigWalletsStateObject) GetValueBytes() []byte {
	MultisigWalletsState, ok := t.GetValue().(*MultisigWalletsState)
	if !ok {
		panic("wrong expected value type")
	}
	value, err := json.Marshal(MultisigWalletsState)
	if err != nil {
		panic("failed to marshal multisigWallet state")
	}
	return value
}

func (t MultisigWalletsStateObject) GetHash() common.Hash {
	return t.MultisigWalletsStateHash
}

func (t MultisigWalletsStateObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *MultisigWalletsStateObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *MultisigWalletsStateObject) Reset() bool {
	t.MultisigWalletsState = NewMultisigWalletsState()
	return true
}

func (t MultisigWalletsStateObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t MultisigWalletsStateObject) IsEmpty() bool {
	temp := NewMultisigWalletsState()
	return reflect.DeepEqual(temp, t.MultisigWalletsState) || t.MultisigWalletsState == nil
}
