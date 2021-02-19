package statedb

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
)

type ShieldingRequest struct {
	externalTxHash string
	incAddress     string
	amount         uint64
}

func NewShieldingRequest() *ShieldingRequest {
	return &ShieldingRequest{}
}

func NewShieldingRequestWithValue(
	externalTxHash string,
	incAddress string,
	amount uint64,
) *ShieldingRequest {
	return &ShieldingRequest{
		externalTxHash: externalTxHash,
		incAddress:     incAddress,
		amount:         amount,
	}
}

func (pr *ShieldingRequest) GetExternalTxHash() string {
	return pr.externalTxHash
}

func (pr *ShieldingRequest) SetExternalTxHash(txHash string) {
	pr.externalTxHash = txHash
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
		ExternalTxHash string
		IncAddress     string
		Amount         uint64
	}{
		ExternalTxHash: pr.externalTxHash,
		IncAddress:     pr.incAddress,
		Amount:         pr.amount,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (pr *ShieldingRequest) UnmarshalJSON(data []byte) error {
	temp := struct {
		ExternalTxHash string
		IncAddress     string
		Amount         uint64
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	pr.incAddress = temp.IncAddress
	pr.amount = temp.Amount
	return nil
}

type ShieldingRequestObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version              int
	ShieldingRequestHash common.Hash
	ShieldingRequest     *ShieldingRequest
	objectType           int
	deleted              bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newShieldingRequestObject(db *StateDB, hash common.Hash) *ShieldingRequestObject {
	return &ShieldingRequestObject{
		version:              defaultVersion,
		db:                   db,
		ShieldingRequestHash: hash,
		ShieldingRequest:     NewShieldingRequest(),
		objectType:           PortalShieldingRequestObjectType,
		deleted:              false,
	}
}

func newShieldingRequestObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*ShieldingRequestObject, error) {
	var shieldingRequestsState = NewShieldingRequest()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, shieldingRequestsState)
		if err != nil {
			return nil, err
		}
	} else {
		shieldingRequestsState, ok = data.(*ShieldingRequest)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidPortalShieldingRequestType, reflect.TypeOf(data))
		}
	}
	return &ShieldingRequestObject{
		version:              defaultVersion,
		ShieldingRequestHash: key,
		ShieldingRequest:     shieldingRequestsState,
		db:                   db,
		objectType:           PortalShieldingRequestObjectType,
		deleted:              false,
	}, nil
}

func GenerateShieldingRequestObjectKey(tokenIDStr string, proofTxHash string) common.Hash {
	prefixHash := GetShieldingRequestPrefix(tokenIDStr)
	valueHash := common.HashH([]byte(proofTxHash))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (t ShieldingRequestObject) GetVersion() int {
	return t.version
}

// setError remembers the first non-nil error it is called with.
func (t *ShieldingRequestObject) SetError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t ShieldingRequestObject) GetTrie(db DatabaseAccessWarper) Trie {
	return t.trie
}

func (t *ShieldingRequestObject) SetValue(data interface{}) error {
	newShieldingRequest, ok := data.(*ShieldingRequest)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidPortalShieldingRequestType, reflect.TypeOf(data))
	}
	t.ShieldingRequest = newShieldingRequest
	return nil
}

func (t ShieldingRequestObject) GetValue() interface{} {
	return t.ShieldingRequest
}

func (t ShieldingRequestObject) GetValueBytes() []byte {
	ShieldingRequest, ok := t.GetValue().(*ShieldingRequest)
	if !ok {
		panic("wrong expected value type")
	}
	value, err := json.Marshal(ShieldingRequest)
	if err != nil {
		panic("failed to marshal multisigWallet state")
	}
	return value
}

func (t ShieldingRequestObject) GetHash() common.Hash {
	return t.ShieldingRequestHash
}

func (t ShieldingRequestObject) GetType() int {
	return t.objectType
}

// MarkDelete will delete an object in trie
func (t *ShieldingRequestObject) MarkDelete() {
	t.deleted = true
}

// reset all shard committee value into default value
func (t *ShieldingRequestObject) Reset() bool {
	t.ShieldingRequest = NewShieldingRequest()
	return true
}

func (t ShieldingRequestObject) IsDeleted() bool {
	return t.deleted
}

// value is either default or nil
func (t ShieldingRequestObject) IsEmpty() bool {
	temp := NewShieldingRequest()
	return reflect.DeepEqual(temp, t.ShieldingRequest) || t.ShieldingRequest == nil
}
