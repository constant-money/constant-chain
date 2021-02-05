package statedb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

type UTXO struct {
	walletAddress string
	txHash        string
	outputIdx     uint32
	outputAmount  uint64
}

func NewUTXO() *UTXO {
	return &UTXO{}
}

func NewUTXOWithValue(
	walletAddress string,
	txHash string,
	outputIdx uint32,
	outputAmount uint64,
) *UTXO {
	return &UTXO{
		walletAddress: walletAddress,
		txHash:        txHash,
		outputAmount:  outputAmount,
		outputIdx:     outputIdx,
	}
}

func (uo *UTXO) GetWalletAddress() string {
	return uo.walletAddress
}

func (uo *UTXO) SetWalletAddress(address string) {
	uo.walletAddress = address
}

func (uo *UTXO) GetTxHash() string {
	return uo.txHash
}

func (uo *UTXO) SetTxHash(txHash string) {
	uo.txHash = txHash
}

func (uo *UTXO) GetOutputAmount() uint64 {
	return uo.outputAmount
}

func (uo *UTXO) SetOutputAmount(amount uint64) {
	uo.outputAmount = amount
}

func (uo *UTXO) GetOutputIndex() uint32 {
	return uo.outputIdx
}

func (uo *UTXO) SetOutputIndex(index uint32) {
	uo.outputIdx = index
}

func (uo *UTXO) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		WalletAddress string
		TxHash        string
		OutputIdx     uint32
		OutputAmount  uint64
	}{
		WalletAddress: uo.walletAddress,
		TxHash:        uo.txHash,
		OutputIdx:     uo.outputIdx,
		OutputAmount:  uo.outputAmount,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (uo *UTXO) UnmarshalJSON(data []byte) error {
	temp := struct {
		WalletAddress string
		TxHash        string
		OutputIdx     uint32
		OutputAmount  uint64
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	uo.walletAddress = temp.WalletAddress
	uo.txHash = temp.TxHash
	uo.outputIdx = temp.OutputIdx
	uo.outputAmount = temp.OutputAmount
	return nil
}

type UTXOObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version    int
	utxoHash   common.Hash
	utxo       *UTXO
	objectType int
	deleted    bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newUTXOObject(db *StateDB, hash common.Hash) *UTXOObject {
	return &UTXOObject{
		version:    defaultVersion,
		db:         db,
		utxoHash:   hash,
		utxo:       NewUTXO(),
		objectType: PortalUTXOObjectType,
		deleted:    false,
	}
}

func newUTXOObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*UTXOObject, error) {
	var content = NewUTXO()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, content)
		if err != nil {
			return nil, err
		}
	} else {
		content, ok = data.(*UTXO)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidPortalUTXOType, reflect.TypeOf(data))
		}
	}
	return &UTXOObject{
		version:    defaultVersion,
		utxoHash:   key,
		utxo:       content,
		db:         db,
		objectType: PortalUTXOObjectType,
		deleted:    false,
	}, nil
}

func GenerateUTXOObjectKey(tokenID string, walletAddress string, txHash string, outputIdx uint32) common.Hash {
	prefixHash := GetWaitingUnshieldRequestPrefix(tokenID)
	value := append([]byte(walletAddress), []byte(txHash)...)
	value = append(value, []byte(strconv.Itoa(int(outputIdx)))...)
	valueHash := common.HashH(value)
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t UTXOObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *UTXOObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t UTXOObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *UTXOObject) SetValue(data interface{}) error {
	utxo, ok := data.(*UTXO)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidPortalUTXOType, reflect.TypeOf(data))
	}
	t.utxo = utxo
	return nil
}

func (t UTXOObject) GetValue() interface{} {
	return t.utxo
}

func (t UTXOObject) GetValueBytes() []byte {
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

func (t UTXOObject) GetHash() common.Hash {
	return t.utxoHash
}

func (t UTXOObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *UTXOObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *UTXOObject) Reset() bool {
	t.utxo = NewUTXO()
	return true
}

func (t UTXOObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t UTXOObject) IsEmpty() bool {
	temp := NewWaitingUnshieldRequestState()
	return reflect.DeepEqual(temp, t.utxo) || t.utxo == nil
}
