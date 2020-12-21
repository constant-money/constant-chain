package statedb

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"math/big"
	"reflect"
)

type OTACoinState struct {
	tokenID    common.Hash
	shardID    byte
	height     []byte
	outputCoin []byte
	index      *big.Int
}

func (ota OTACoinState) Index() *big.Int {
	return ota.index
}

func (ota *OTACoinState) SetIndex(index *big.Int) {
	ota.index = index
}

func (ota OTACoinState) OutputCoin() []byte {
	return ota.outputCoin
}

func (ota *OTACoinState) SetOutputCoin(outputCoin []byte) {
	ota.outputCoin = outputCoin
}

func (ota OTACoinState) TokenID() common.Hash {
	return ota.tokenID
}

func (ota *OTACoinState) SetTokenID(tokenID common.Hash) {
	ota.tokenID = tokenID
}

func (ota OTACoinState) ShardID() byte {
	return ota.shardID
}

func (ota *OTACoinState) SetShardID(shardID byte) {
	ota.shardID = shardID
}

func (ota OTACoinState) Height() []byte {
	return ota.height
}

func (ota *OTACoinState) SetHeight(height []byte) {
	ota.height = height
}

func (ota *OTACoinState) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		TokenID    common.Hash
		ShardID    byte
		Height     []byte
		OutputCoin []byte
		Index      *big.Int
	}{
		TokenID:    ota.tokenID,
		ShardID:    ota.shardID,
		Height:     ota.height,
		OutputCoin: ota.outputCoin,
		Index:      ota.index,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (ota *OTACoinState) UnmarshalJSON(data []byte) error {
	temp := struct {
		TokenID    common.Hash
		ShardID    byte
		Height     []byte
		OutputCoin []byte
		Index      *big.Int
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	ota.tokenID = temp.TokenID
	ota.shardID = temp.ShardID
	ota.height = temp.Height
	ota.outputCoin = temp.OutputCoin
	ota.index = temp.Index
	return nil
}

func NewOTACoinStateWithValue(tokenID common.Hash, shardID byte, height []byte, outputCoin []byte, index *big.Int) *OTACoinState {
	return &OTACoinState{tokenID: tokenID, shardID: shardID, height: height, outputCoin: outputCoin, index: index}
}

func NewOTACoinState() *OTACoinState {
	return &OTACoinState{}
}

type OTACoinObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version             int
	onetimeAddressHash  common.Hash
	onetimeAddressState *OTACoinState
	objectType          int
	deleted             bool

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newOTACoinObject(db *StateDB, hash common.Hash) *OTACoinObject {
	return &OTACoinObject{
		version:             defaultVersion,
		db:                  db,
		onetimeAddressHash:  hash,
		onetimeAddressState: NewOTACoinState(),
		objectType:          OTACoinObjectType,
		deleted:             false,
	}
}

func newOTACoinObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*OTACoinObject, error) {
	var newOTACoinState = NewOTACoinState()
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := json.Unmarshal(dataBytes, newOTACoinState)
		if err != nil {
			return nil, err
		}
	} else {
		newOTACoinState, ok = data.(*OTACoinState)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidOTACoinStateType, reflect.TypeOf(data))
		}
	}
	return &OTACoinObject{
		version:             defaultVersion,
		onetimeAddressHash:  key,
		onetimeAddressState: newOTACoinState,
		db:                  db,
		objectType:          OTACoinObjectType,
		deleted:             false,
	}, nil
}

func GenerateOTACoinObjectKey(tokenID common.Hash, shardID byte, height []byte, outputCoin []byte) common.Hash {
	// non-PRV coins will be indexed together
	if tokenID != common.PRVCoinID {
		tokenID = common.ConfidentialAssetID
	}
	prefixHash := GetOTACoinPrefix(tokenID, shardID, height)
	valueHash := common.HashH(outputCoin)
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (s OTACoinObject) GetVersion() int {
	return s.version
}

// setError remembers the first non-nil error it is called with.
func (s *OTACoinObject) SetError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s OTACoinObject) GetTrie(db DatabaseAccessWarper) Trie {
	return s.trie
}

func (s *OTACoinObject) SetValue(data interface{}) error {
	newOnetimeAddressState, ok := data.(*OTACoinState)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidOTACoinStateType, reflect.TypeOf(data))
	}
	s.onetimeAddressState = newOnetimeAddressState
	return nil
}

func (s OTACoinObject) GetValue() interface{} {
	return s.onetimeAddressState
}

func (s OTACoinObject) GetValueBytes() []byte {
	onetimeAddressState, ok := s.GetValue().(*OTACoinState)
	if !ok {
		panic("wrong expected value type")
	}
	onetimeAddressStateBytes, err := json.Marshal(onetimeAddressState)
	if err != nil {
		panic(err.Error())
	}
	return onetimeAddressStateBytes
}

func (s OTACoinObject) GetHash() common.Hash {
	return s.onetimeAddressHash
}

func (s OTACoinObject) GetType() int {
	return s.objectType
}

// MarkDelete will delete an object in trie
func (s *OTACoinObject) MarkDelete() {
	s.deleted = true
}

// Reset serial number into default
func (s *OTACoinObject) Reset() bool {
	s.onetimeAddressState = NewOTACoinState()
	return true
}

func (s OTACoinObject) IsDeleted() bool {
	return s.deleted
}

// empty value or not
func (s OTACoinObject) IsEmpty() bool {
	temp := NewOTACoinState()
	return reflect.DeepEqual(temp, s.onetimeAddressState) || s.onetimeAddressState == nil
}

//========================================================== INDEX
type OTACoinIndexObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version          int
	otaCoinIndexHash common.Hash
	otaCoinHash      common.Hash
	objectType       int
	deleted          bool
	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newOTACoinIndexObject(db *StateDB, hash common.Hash) *OTACoinIndexObject {
	return &OTACoinIndexObject{
		version:          defaultVersion,
		db:               db,
		otaCoinIndexHash: hash,
		otaCoinHash:      common.Hash{},
		objectType:       OTACoinIndexObjectType,
		deleted:          false,
	}
}

func newOTACoinIndexObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*OTACoinIndexObject, error) {
	var newOnetimeAddressCoinIndexState = common.Hash{}
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		err := newOnetimeAddressCoinIndexState.SetBytes(dataBytes)
		if err != nil {
			return nil, err
		}
	} else {
		newOnetimeAddressCoinIndexState, ok = data.(common.Hash)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidHashType, reflect.TypeOf(data))
		}
	}
	return &OTACoinIndexObject{
		version:          defaultVersion,
		otaCoinIndexHash: key,
		otaCoinHash:      newOnetimeAddressCoinIndexState,
		db:               db,
		objectType:       OTACoinIndexObjectType,
		deleted:          false,
	}, nil
}

func GenerateOTACoinIndexObjectKey(tokenID common.Hash, shardID byte, index *big.Int) common.Hash {
	// non-PRV coins will be indexed together
	if tokenID != common.PRVCoinID {
		tokenID = common.ConfidentialAssetID
	}
	prefixHash := GetOTACoinIndexPrefix(tokenID, shardID)
	valueHash := common.HashH(index.Bytes())
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (s OTACoinIndexObject) GetVersion() int {
	return s.version
}

// setError remembers the first non-nil error it is called with.
func (s *OTACoinIndexObject) SetError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s OTACoinIndexObject) GetTrie(db DatabaseAccessWarper) Trie {
	return s.trie
}

func (s *OTACoinIndexObject) SetValue(data interface{}) error {
	newOTACoinIndexState, ok := data.(common.Hash)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidHashType, reflect.TypeOf(data))
	}
	s.otaCoinHash = newOTACoinIndexState
	return nil
}

func (s OTACoinIndexObject) GetValue() interface{} {
	return s.otaCoinHash
}

func (s OTACoinIndexObject) GetValueBytes() []byte {
	temp := s.GetValue().(common.Hash)
	return temp.Bytes()
}

func (s OTACoinIndexObject) GetHash() common.Hash {
	return s.otaCoinIndexHash
}

func (s OTACoinIndexObject) GetType() int {
	return s.objectType
}

// MarkDelete will delete an object in trie
func (s *OTACoinIndexObject) MarkDelete() {
	s.deleted = true
}

// Reset serial number into default
func (s *OTACoinIndexObject) Reset() bool {
	s.otaCoinHash = common.Hash{}
	return true
}

func (s OTACoinIndexObject) IsDeleted() bool {
	return s.deleted
}

// empty value or not
func (s OTACoinIndexObject) IsEmpty() bool {
	temp := common.Hash{}
	return reflect.DeepEqual(temp, s.otaCoinHash)
}

//========================================================== Length
type OTACoinLengthObject struct {
	db *StateDB
	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	version           int
	otaCoinLengthHash common.Hash
	otaCoinLength     *big.Int
	objectType        int
	deleted           bool
	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
}

func newOTACoinLengthObject(db *StateDB, hash common.Hash) *OTACoinLengthObject {
	return &OTACoinLengthObject{
		version:           defaultVersion,
		db:                db,
		otaCoinLengthHash: hash,
		otaCoinLength:     new(big.Int).SetUint64(0),
		objectType:        OTACoinLengthObjectType,
		deleted:           false,
	}
}

func newOTACoinLengthObjectWithValue(db *StateDB, key common.Hash, data interface{}) (*OTACoinLengthObject, error) {
	var newOTACoinLengthValue = new(big.Int)
	var ok bool
	var dataBytes []byte
	if dataBytes, ok = data.([]byte); ok {
		newOTACoinLengthValue.SetBytes(dataBytes)
	} else {
		newOTACoinLengthValue, ok = data.(*big.Int)
		if !ok {
			return nil, fmt.Errorf("%+v, got type %+v", ErrInvalidBigIntType, reflect.TypeOf(data))
		}
	}
	return &OTACoinLengthObject{
		version:           defaultVersion,
		otaCoinLengthHash: key,
		otaCoinLength:     newOTACoinLengthValue,
		db:                db,
		objectType:        OTACoinLengthObjectType,
		deleted:           false,
	}, nil
}

func GenerateOTACoinLengthObjectKey(tokenID common.Hash, shardID byte) common.Hash {
	// non-PRV coins will be indexed together
	if tokenID != common.PRVCoinID {
		tokenID = common.ConfidentialAssetID
	}
	prefixHash := GetOTACoinLengthPrefix()
	valueHash := common.HashH(append(tokenID[:], shardID))
	return common.BytesToHash(append(prefixHash, valueHash[:][:prefixKeyLength]...))
}

func (s OTACoinLengthObject) GetVersion() int {
	return s.version
}

// setError remembers the first non-nil error it is called with.
func (s *OTACoinLengthObject) SetError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s OTACoinLengthObject) GetTrie(db DatabaseAccessWarper) Trie {
	return s.trie
}

func (s *OTACoinLengthObject) SetValue(data interface{}) error {
	newOTACoinLengthValue, ok := data.(*big.Int)
	if !ok {
		return fmt.Errorf("%+v, got type %+v", ErrInvalidBigIntType, reflect.TypeOf(data))
	}
	s.otaCoinLength = newOTACoinLengthValue
	return nil
}

func (s OTACoinLengthObject) GetValue() interface{} {
	return s.otaCoinLength
}

func (s OTACoinLengthObject) GetValueBytes() []byte {
	if s.GetValue().(*big.Int).Uint64() == 0 {
		return []byte{0}
	}
	return s.GetValue().(*big.Int).Bytes()
}

func (s OTACoinLengthObject) GetHash() common.Hash {
	return s.otaCoinLengthHash
}

func (s OTACoinLengthObject) GetType() int {
	return s.objectType
}

// MarkDelete will delete an object in trie
func (s *OTACoinLengthObject) MarkDelete() {
	s.deleted = true
}

// Reset serial number into default
func (s *OTACoinLengthObject) Reset() bool {
	s.otaCoinLength = new(big.Int).SetUint64(0)
	return true
}

func (s OTACoinLengthObject) IsDeleted() bool {
	return s.deleted
}

// empty value or not
func (s OTACoinLengthObject) IsEmpty() bool {
	return s.otaCoinLength == nil
}
