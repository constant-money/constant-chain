package rawdbv2

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incdb"
)

// StoreShardBlock store block hash => block value and block index => block hash
// record1: prefix-shardid-index-hash => empty
// record2: prefix-hash => block value
func StoreShardBlock(db incdb.KeyValueWriter, hash common.Hash, v interface{}) error {
	keyHash := GetShardHashToBlockKey(hash)
	val, err := json.Marshal(v)
	if err != nil {
		return NewRawdbError(StoreShardBlockError, err)
	}
	if err := db.Put(keyHash, val); err != nil {
		return NewRawdbError(StoreShardBlockError, err)
	}
	return nil
}

func StoreFinalizedShardBlockHashByIndex(db incdb.KeyValueWriter, sid byte, index uint64, hash common.Hash) error {
	keyHash := GetShardIndexToBlockHashPrefix(sid, index)
	if err := db.Put(keyHash, hash.Bytes()); err != nil {
		return NewRawdbError(StoreShardBlockIndexError, err)
	}
	return nil
}

func GetFinalizedShardBlockHashByIndex(db incdb.KeyValueReader, sid byte, index uint64) (*common.Hash, error) {
	keyHash := GetShardIndexToBlockHashPrefix(sid, index)
	val, err := db.Get(keyHash)
	if err != nil {
		return nil, NewRawdbError(GetShardBlockByIndexError, err)
	}
	h, err := common.Hash{}.NewHash(val)
	if err != nil {
		return nil, NewRawdbError(GetShardBlockByIndexError, err)
	}
	return h, nil
}

func HasShardBlock(db incdb.KeyValueReader, hash common.Hash) (bool, error) {
	keyHash := GetShardHashToBlockKey(hash)
	if ok, err := db.Has(keyHash); err != nil {
		return false, NewRawdbError(HasShardBlockError, fmt.Errorf("has key %+v failed", keyHash))
	} else if ok {
		return true, nil
	}
	return false, nil
}

func GetShardBlockByHash(db incdb.KeyValueReader, hash common.Hash) ([]byte, error) {
	keyHash := GetShardHashToBlockKey(hash)
	if ok, err := db.Has(keyHash); err != nil {
		return []byte{}, NewRawdbError(GetShardBlockByHashError, fmt.Errorf("has key %+v failed", keyHash))
	} else if !ok {
		return []byte{}, NewRawdbError(GetShardBlockByHashError, fmt.Errorf("block %+v not exist", hash))
	}
	block, err := db.Get(keyHash)
	if err != nil {
		return nil, NewRawdbError(GetShardBlockByHashError, err)
	}
	ret := make([]byte, len(block))
	copy(ret, block)
	return ret, nil
}

func StoreShardBestState(db incdb.KeyValueWriter, shardID byte, v interface{}) error {
	key := GetShardBestStateKey(shardID)
	val, err := json.Marshal(v)
	if err != nil {
		return NewRawdbError(StoreShardBestStateError, err)
	}
	if err := db.Put(key, val); err != nil {
		return NewRawdbError(StoreShardBestStateError, err)
	}
	return nil
}

func GetShardBestState(db incdb.KeyValueReader, shardID byte) ([]byte, error) {
	key := GetShardBestStateKey(shardID)
	shardBestStateBytes, err := db.Get(key)
	if err != nil {
		return nil, NewRawdbError(StoreShardBestStateError, err)
	}
	return shardBestStateBytes, nil
}

// StoreFeeEstimator - Store data for FeeEstimator object
func StoreFeeEstimator(db incdb.KeyValueWriter, val []byte, shardID byte) error {
	key := GetFeeEstimatorPrefix(shardID)
	err := db.Put(key, val)
	if err != nil {
		return NewRawdbError(StoreFeeEstimatorError, err)
	}
	return nil
}

// GetFeeEstimator - Get data for FeeEstimator object as a json in byte format
func GetFeeEstimator(db incdb.KeyValueReader, shardID byte) ([]byte, error) {
	key := GetFeeEstimatorPrefix(shardID)
	res, err := db.Get(key)
	if err != nil {
		return nil, NewRawdbError(GetFeeEstimatorError, err)
	}
	return res, err
}
func StoreShardRootsHash(db incdb.KeyValueWriter, shardID byte, hash common.Hash, rootsHash interface{}) error {
	key := GetShardRootsHashKey(shardID, hash)
	b, _ := json.Marshal(rootsHash)
	err := db.Put(key, b)
	if err != nil {
		return NewRawdbError(StoreShardConsensusRootHashError, err)
	}
	return nil
}

func GetShardRootsHash(db incdb.KeyValueReader, shardID byte, hash common.Hash) ([]byte, error) {
	key := GetShardRootsHashKey(shardID, hash)
	return db.Get(key)
}
