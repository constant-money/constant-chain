package statedb

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"reflect"
)

type ProcessUnshieldDetail struct {
	unshieldsID []string
	utxos       map[string][]*UTXO // map key (wallet address => list utxos)
	txHash      []string
	fee         uint
}

type ProcessUnshield struct {
	unshields map[string]*ProcessUnshieldDetail // external tx id => ProcessUnshieldDetail
}

func (rq *ProcessUnshield) GetUnshield(unshiedID string) *ProcessUnshieldDetail {
	return rq.unshields[unshiedID]
}

func (rq *ProcessUnshield) GetUnshields() map[string]*ProcessUnshieldDetail {
	return rq.unshields
}

func (rq *ProcessUnshield) SetUnshield(unshiedID string, unshield *ProcessUnshieldDetail) {
	rq.unshields[unshiedID] = unshield
}

func (rq ProcessUnshield) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		Unshields map[string]*ProcessUnshieldDetail
	}{
		Unshields: rq.unshields,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (rq *ProcessUnshield) UnmarshalJSON(data []byte) error {
	temp := struct {
		Unshields map[string]*ProcessUnshieldDetail
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	rq.unshields = temp.Unshields
	return nil
}

func (us *ProcessUnshieldDetail) GetUTXOs() map[string][]*UTXO {
	return us.utxos
}

func (us *ProcessUnshieldDetail) SetUTXOs(newUTXOs map[string][]*UTXO) {
	us.utxos = newUTXOs
}

func (us *ProcessUnshieldDetail) GetTXHash() []string {
	return us.txHash
}

func (us *ProcessUnshieldDetail) SetTXHash(btcTxHashs []string) {
	us.txHash = btcTxHashs
}

func (us *ProcessUnshieldDetail) GetUnshieldRequests() []string {
	return us.unshieldsID
}

func (us *ProcessUnshieldDetail) SetUnshieldRequests(usRequests []string) {
	us.unshieldsID = usRequests
}

func (us *ProcessUnshieldDetail) GetUnshieldFee() uint {
	return us.fee
}

func (us *ProcessUnshieldDetail) SetUnshieldFee(fee uint) {
	us.fee = fee
}

func (rq ProcessUnshieldDetail) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		UnshieldsID []string
		UTXOs       map[string][]*UTXO
		BtcTxHash   []string
		Fee         uint
	}{
		UnshieldsID: rq.unshieldsID,
		UTXOs:       rq.utxos,
		BtcTxHash:   rq.txHash,
		Fee:         rq.fee,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (rq *ProcessUnshieldDetail) UnmarshalJSON(data []byte) error {
	temp := struct {
		UnshieldsID []string
		UTXOs       map[string][]*UTXO
		TxHash      []string
		Fee         uint
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	rq.unshieldsID = temp.UnshieldsID
	rq.utxos = temp.UTXOs
	rq.txHash = temp.TxHash
	rq.fee = temp.Fee
	return nil
}

func (ut *UTXO) getTxHash() string {
	return ut.txHash
}

func (ut *UTXO) setTxHash(txHash string) {
	ut.txHash = txHash
}

func (ut *UTXO) getIndex() int {
	return ut.outputIdx
}

func (ut *UTXO) setIndex(index int) {
	ut.outputIdx = index
}

func (ut *UTXO) getAmount() uint64 {
	return ut.outputAmount
}

func (ut *UTXO) setAmount(amount uint64) {
	ut.outputAmount = amount
}

func (ut UTXO) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		TxHash       string
		OutputIdx    int
		OutputAmount uint64
	}{
		TxHash:       ut.txHash,
		OutputIdx:    ut.outputIdx,
		OutputAmount: ut.outputAmount,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (ut *UTXO) UnmarshalJSON(data []byte) error {
	temp := struct {
		TxHash       string
		OutputIdx    int
		OutputAmount uint64
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	ut.outputAmount = temp.OutputAmount
	ut.outputIdx = temp.OutputIdx
	ut.txHash = temp.TxHash
	return nil
}

func NewProcessUnshieldState() *ProcessUnshield {
	return &ProcessUnshield{
		unshields: map[string]*ProcessUnshieldDetail{},
	}
}

func NewProcessUnshieldStateWithValue(
	unshieldsInput map[string]*ProcessUnshieldDetail,
) *ProcessUnshield {
	return &ProcessUnshield{
		unshields: unshieldsInput,
	}
}

func NewProcessUnshieldDetailWithValue(
	unshieldsIDInput []string,
	utxosInput map[string][]*UTXO,
	txHashInput []string,
	walletAddrsInput []string) *ProcessUnshieldDetail {
	return &ProcessUnshieldDetail{
		unshieldsID: unshieldsIDInput,
		utxos:       utxosInput,
		txHash:      txHashInput,
	}
}

func NewProcessUnshieldDetail() *ProcessUnshieldDetail {
	return &ProcessUnshieldDetail{}
}

func NewProcessUnshieldUTXOWithValue(
	txHash string,
	index int,
	amount uint64) *UTXO {
	return &UTXO{
		txHash:       txHash,
		outputIdx:    index,
		outputAmount: amount,
	}
}

func NewProcessUnshieldUTXO() *UTXO {
	return &UTXO{}
}

type ProcessUnshieldObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version                    int
	waitingProcessUnshieldHash common.Hash
	waitingProcessUnshield     *ProcessUnshield
	objectType                 int
	deleted                    bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newProcessUnshieldObject(db *StateDB, hash common.Hash) *ProcessUnshieldObject {
	return &ProcessUnshieldObject{
		version:                    defaultVersion,
		db:                         db,
		waitingProcessUnshieldHash: hash,
		waitingProcessUnshield:     NewProcessUnshieldState(),
		objectType:                 PortalUnshieldProcessedObjectType,
		deleted:                    false,
	}
}

func newProcessUnshieldObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*ProcessUnshieldObject, error) {
	var content = NewProcessUnshieldState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, content)
		if err != nil {
			return nil, err
		}
	} else {
		content, ok = data.(*ProcessUnshield)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidUnshieldRequestProcessedType, reflect.TypeOf(data))
		}
	}
	return &ProcessUnshieldObject{
		version:                    defaultVersion,
		waitingProcessUnshieldHash: key,
		waitingProcessUnshield:     content,
		db:                         db,
		objectType:                 PortalUnshieldProcessedObjectType,
		deleted:                    false,
	}, nil
}

func GenerateWaitingProcessUnshieldObjectKey(redeemID string) common.Hash {
	prefixHash := GetUnshieldRequestProcessedPrefix()
	valueHash := common.HashH([]byte(redeemID))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func GenerateMatchedProcessUnshieldObjectKey(redeemID string) common.Hash {
	prefixHash := GetUnshieldRequestProcessedPrefix()
	valueHash := common.HashH([]byte(redeemID))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t ProcessUnshieldObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *ProcessUnshieldObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t ProcessUnshieldObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *ProcessUnshieldObject) SetValue(data interface{}) error {
	ProcessUnshield, ok := data.(*ProcessUnshield)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidUnshieldRequestProcessedType, reflect.TypeOf(data))
	}
	t.waitingProcessUnshield = ProcessUnshield
	return nil
}

func (t ProcessUnshieldObject) GetValue() interface{} {
	return t.waitingProcessUnshield
}

func (t ProcessUnshieldObject) GetValueBytes() []byte {
	ProcessUnshield, ok := t.GetValue().(*ProcessUnshield)
	if !ok {
		panic("wrong expected value type")
	}
	value, err := json.Marshal(ProcessUnshield)
	if err != nil {
		panic("failed to marshal redeem request")
	}
	return value
}

func (t ProcessUnshieldObject) GetHash() common.Hash {
	return t.waitingProcessUnshieldHash
}

func (t ProcessUnshieldObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *ProcessUnshieldObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *ProcessUnshieldObject) Reset() bool {
	t.waitingProcessUnshield = NewProcessUnshieldState()
	return true
}

func (t ProcessUnshieldObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t ProcessUnshieldObject) IsEmpty() bool {
	temp := NewProcessUnshieldState()
	return reflect.DeepEqual(temp, t.waitingProcessUnshield) || t.waitingProcessUnshield == nil
}
