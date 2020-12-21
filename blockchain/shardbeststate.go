package blockchain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

// BestState houses information about the current best block and other info
// related to the state of the main chain as it exists from the point of view of
// the current best block.
//
// The BestSnapshot method can be used to obtain access to this information
// in a concurrent safe manner and the data will not be changed out from under
// the caller when chain state changes occur as the function name implies.
// However, the returned snapshot must be treated as immutable since it is
// shared by all callers.

type ShardRootHash struct {
	ConsensusStateDBRootHash   common.Hash
	TransactionStateDBRootHash common.Hash
	FeatureStateDBRootHash     common.Hash
	RewardStateDBRootHash      common.Hash
	SlashStateDBRootHash       common.Hash
}

type ShardBestState struct {
	BestBlockHash          common.Hash                       `json:"BestBlockHash"` // hash of block.
	BestBlock              *ShardBlock                       `json:"-"`             // block data
	BestBeaconHash         common.Hash                       `json:"BestBeaconHash"`
	BeaconHeight           uint64                            `json:"BeaconHeight"`
	ShardID                byte                              `json:"ShardID"`
	Epoch                  uint64                            `json:"Epoch"`
	ShardHeight            uint64                            `json:"ShardHeight"`
	MaxShardCommitteeSize  int                               `json:"MaxShardCommitteeSize"`
	MinShardCommitteeSize  int                               `json:"MinShardCommitteeSize"`
	ShardProposerIdx       int                               `json:"ShardProposerIdx"`
	ShardCommittee         []incognitokey.CommitteePublicKey `json:"-"`
	ShardPendingValidator  []incognitokey.CommitteePublicKey `json:"-"`
	BestCrossShard         map[byte]uint64                   `json:"BestCrossShard"` // Best cross shard block by heigh
	StakingTx              *MapStringString                  `json:"-"`
	NumTxns                uint64                            `json:"NumTxns"`                // The number of txns in the block.
	TotalTxns              uint64                            `json:"TotalTxns"`              // The total number of txns in the chain.
	TotalTxnsExcludeSalary uint64                            `json:"TotalTxnsExcludeSalary"` // for testing and benchmark
	ActiveShards           int                               `json:"ActiveShards"`
	ConsensusAlgorithm     string                            `json:"ConsensusAlgorithm"`
	// Number of blocks produced by producers in epoch
	NumOfBlocksByProducers map[string]uint64 `json:"NumOfBlocksByProducers"`
	BlockInterval          time.Duration
	BlockMaxCreateTime     time.Duration
	MetricBlockHeight      uint64
	//================================ StateDB Method
	// block height => root hash
	consensusStateDB           *statedb.StateDB
	ConsensusStateDBRootHash   common.Hash
	transactionStateDB         *statedb.StateDB
	TransactionStateDBRootHash common.Hash
	featureStateDB             *statedb.StateDB
	FeatureStateDBRootHash     common.Hash
	rewardStateDB              *statedb.StateDB
	RewardStateDBRootHash      common.Hash
	slashStateDB               *statedb.StateDB
	SlashStateDBRootHash       common.Hash
}

func (shardBestState *ShardBestState) GetCopiedConsensusStateDB() *statedb.StateDB {
	return shardBestState.consensusStateDB.Copy()
}

func (shardBestState *ShardBestState) GetCopiedTransactionStateDB() *statedb.StateDB {
	return shardBestState.transactionStateDB.Copy()
}

func (shardBestState *ShardBestState) GetCopiedFeatureStateDB() *statedb.StateDB {
	return shardBestState.featureStateDB.Copy()
}

func (shardBestState *ShardBestState) GetShardRewardStateDB() *statedb.StateDB {
	return shardBestState.rewardStateDB.Copy()
}

func (shardBestState *ShardBestState) GetHash() *common.Hash {
	return shardBestState.BestBlock.Hash()
}

func (shardBestState *ShardBestState) GetPreviousHash() *common.Hash {
	return &shardBestState.BestBlock.Header.PreviousBlockHash
}

func (shardBestState *ShardBestState) GetHeight() uint64 {
	return shardBestState.BestBlock.GetHeight()
}

func (shardBestState *ShardBestState) GetEpoch() uint64 {
	return shardBestState.Epoch
}

func (shardBestState *ShardBestState) GetBlockTime() int64 {
	return shardBestState.BestBlock.Header.Timestamp
}

// var bestStateShardMap = make(map[byte]*ShardBestState)

func NewShardBestState() *ShardBestState {
	return &ShardBestState{}
}
func NewShardBestStateWithShardID(shardID byte) *ShardBestState {
	return &ShardBestState{ShardID: shardID}
}
func NewBestStateShardWithConfig(shardID byte, netparam *Params) *ShardBestState {
	bestStateShard := NewShardBestStateWithShardID(shardID)
	err := bestStateShard.BestBlockHash.SetBytes(make([]byte, 32))
	if err != nil {
		panic(err)
	}
	err = bestStateShard.BestBeaconHash.SetBytes(make([]byte, 32))
	if err != nil {
		panic(err)
	}
	bestStateShard.BestBlock = nil
	bestStateShard.ShardCommittee = []incognitokey.CommitteePublicKey{}
	bestStateShard.MaxShardCommitteeSize = netparam.MaxShardCommitteeSize
	bestStateShard.MinShardCommitteeSize = netparam.MinShardCommitteeSize
	bestStateShard.ShardPendingValidator = []incognitokey.CommitteePublicKey{}
	bestStateShard.ActiveShards = netparam.ActiveShards
	bestStateShard.BestCrossShard = make(map[byte]uint64)
	bestStateShard.StakingTx = NewMapStringString()
	bestStateShard.ShardHeight = 1
	bestStateShard.BeaconHeight = 1
	bestStateShard.BlockInterval = netparam.MinShardBlockInterval
	bestStateShard.BlockMaxCreateTime = netparam.MaxShardBlockCreation
	return bestStateShard
}

func (blockchain *BlockChain) GetBestStateShard(shardID byte) *ShardBestState {
	if blockchain.ShardChain[int(shardID)].multiView.GetBestView() == nil {
		return nil
	}
	return blockchain.ShardChain[int(shardID)].multiView.GetBestView().(*ShardBestState)
}

func (shardBestState *ShardBestState) InitStateRootHash(db incdb.Database, bc *BlockChain) error {
	var err error
	var dbAccessWarper = statedb.NewDatabaseAccessWarper(db)
	shardBestState.consensusStateDB, err = statedb.NewWithPrefixTrie(shardBestState.ConsensusStateDBRootHash, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.transactionStateDB, err = statedb.NewWithPrefixTrie(shardBestState.TransactionStateDBRootHash, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.featureStateDB, err = statedb.NewWithPrefixTrie(shardBestState.FeatureStateDBRootHash, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.rewardStateDB, err = statedb.NewWithPrefixTrie(shardBestState.RewardStateDBRootHash, dbAccessWarper)
	if err != nil {
		return err
	}
	shardBestState.slashStateDB, err = statedb.NewWithPrefixTrie(shardBestState.SlashStateDBRootHash, dbAccessWarper)
	if err != nil {
		return err
	}
	return nil
}

// Get role of a public key base on best state shard
func (shardBestState *ShardBestState) GetBytes() []byte {
	res := []byte{}
	res = append(res, shardBestState.BestBlockHash.GetBytes()...)
	res = append(res, shardBestState.BestBlock.Hash().GetBytes()...)
	res = append(res, shardBestState.BestBeaconHash.GetBytes()...)
	beaconHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(beaconHeightBytes, shardBestState.BeaconHeight)
	res = append(res, beaconHeightBytes...)
	res = append(res, shardBestState.ShardID)
	epochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBytes, shardBestState.Epoch)
	res = append(res, epochBytes...)
	shardHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(shardHeightBytes, shardBestState.ShardHeight)
	res = append(res, shardHeightBytes...)
	shardCommitteeSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(shardCommitteeSizeBytes, uint32(shardBestState.MaxShardCommitteeSize))
	res = append(res, shardCommitteeSizeBytes...)
	minShardCommitteeSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(minShardCommitteeSizeBytes, uint32(shardBestState.MinShardCommitteeSize))
	res = append(res, minShardCommitteeSizeBytes...)
	proposerIdxBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(proposerIdxBytes, uint32(shardBestState.ShardProposerIdx))
	res = append(res, proposerIdxBytes...)
	for _, value := range shardBestState.ShardCommittee {
		valueBytes, err := value.Bytes()
		if err != nil {
			return nil
		}
		res = append(res, valueBytes...)
	}
	for _, value := range shardBestState.ShardPendingValidator {
		valueBytes, err := value.Bytes()
		if err != nil {
			return nil
		}
		res = append(res, valueBytes...)
	}
	keys := []int{}
	for k := range shardBestState.BestCrossShard {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		value := shardBestState.BestCrossShard[byte(shardID)]
		valueBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(valueBytes, value)
		res = append(res, valueBytes...)
	}
	keystr := []string{}

	for _, k := range shardBestState.StakingTx.data {
		keystr = append(keystr, k)
	}
	sort.Strings(keystr)
	for _, key := range keystr {
		value := shardBestState.StakingTx.data[key]
		res = append(res, []byte(key)...)
		res = append(res, []byte(value)...)
	}
	numTxnsBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(numTxnsBytes, shardBestState.NumTxns)
	res = append(res, numTxnsBytes...)
	totalTxnsBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(totalTxnsBytes, shardBestState.TotalTxns)
	res = append(res, totalTxnsBytes...)
	activeShardsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(activeShardsBytes, uint32(shardBestState.ActiveShards))
	res = append(res, activeShardsBytes...)
	return res
}

func (shardBestState *ShardBestState) Hash() common.Hash {
	return common.HashH(shardBestState.GetBytes())
}

func (shardBestState *ShardBestState) SetMaxShardCommitteeSize(maxShardCommitteeSize int) bool {
	// check input params, below MinCommitteeSize failed to acheive consensus
	if maxShardCommitteeSize < MinCommitteeSize {
		return false
	}
	// max committee size can't be lower than current min committee size
	if maxShardCommitteeSize >= shardBestState.MinShardCommitteeSize {
		shardBestState.MaxShardCommitteeSize = maxShardCommitteeSize
		return true
	}
	return false
}

func (shardBestState *ShardBestState) SetMinShardCommitteeSize(minShardCommitteeSize int) bool {
	// check input params, below MinCommitteeSize failed to acheive consensus
	if minShardCommitteeSize < MinCommitteeSize {
		return false
	}
	// min committee size can't be greater than current min committee size
	if minShardCommitteeSize <= shardBestState.MaxShardCommitteeSize {
		shardBestState.MinShardCommitteeSize = minShardCommitteeSize
		return true
	}
	return false
}

//MarshalJSON - remember to use lock
func (shardBestState *ShardBestState) MarshalJSON() ([]byte, error) {
	type Alias ShardBestState
	b, err := json.Marshal(&struct {
		*Alias
	}{
		(*Alias)(shardBestState),
	})
	if err != nil {
		Logger.log.Error(err)
	}
	return b, err
}

func (shardBestState ShardBestState) GetShardHeight() uint64 {
	return shardBestState.ShardHeight
}

func (shardBestState ShardBestState) GetBeaconHeight() uint64 {
	return shardBestState.BeaconHeight
}

//cloneShardBestStateFrom - remember to use lock
func (shardBestState *ShardBestState) cloneShardBestStateFrom(target *ShardBestState) error {
	tempMarshal, err := json.Marshal(target)
	if err != nil {
		return NewBlockChainError(MashallJsonShardBestStateError, fmt.Errorf("Shard Best State %+v get %+v", target.ShardHeight, err))
	}
	err = json.Unmarshal(tempMarshal, shardBestState)
	if err != nil {
		return NewBlockChainError(UnmashallJsonShardBestStateError, fmt.Errorf("Clone Shard Best State %+v get %+v", target.ShardHeight, err))
	}
	if reflect.DeepEqual(*shardBestState, ShardBestState{}) {
		return NewBlockChainError(CloneShardBestStateError, fmt.Errorf("Shard Best State %+v clone failed", target.ShardHeight))
	}

	shardBestState.StakingTx = target.StakingTx.LazyCopy()

	shardBestState.consensusStateDB = target.consensusStateDB.Copy()
	shardBestState.transactionStateDB = target.transactionStateDB.Copy()
	shardBestState.featureStateDB = target.featureStateDB.Copy()
	shardBestState.rewardStateDB = target.rewardStateDB.Copy()
	shardBestState.slashStateDB = target.slashStateDB.Copy()

	shardBestState.BestBlock = target.BestBlock

	shardBestState.ShardCommittee = make([]incognitokey.CommitteePublicKey, len(target.ShardCommittee))
	for i, v := range target.ShardCommittee {
		shardBestState.ShardCommittee[i] = v
	}

	shardBestState.ShardPendingValidator = make([]incognitokey.CommitteePublicKey, len(target.ShardPendingValidator))
	for i, v := range target.ShardPendingValidator {
		shardBestState.ShardPendingValidator[i] = v
	}

	return nil
}

func (shardBestState *ShardBestState) GetStakingTx() map[string]string {
	m := make(map[string]string)
	for k, v := range shardBestState.StakingTx.data {
		m[k] = v
	}
	return m
}

func (shardBestState *ShardBestState) GetCommittee() []incognitokey.CommitteePublicKey {
	result := []incognitokey.CommitteePublicKey{}
	return append(result, shardBestState.ShardCommittee...)
}

func (shardBestState *ShardBestState) GetProposerByTimeSlot(ts int64, version int) incognitokey.CommitteePublicKey {
	id := GetProposerByTimeSlot(ts, shardBestState.MinShardCommitteeSize)
	return shardBestState.ShardCommittee[id]
}

func (shardBestState *ShardBestState) GetBlock() common.BlockInterface {
	return shardBestState.BestBlock
}

func GetProposerByTimeSlot(ts int64, committeeLen int) int {
	id := int(ts) % committeeLen
	return id
}

func (shardBestState *ShardBestState) GetShardCommittee() []incognitokey.CommitteePublicKey {
	return shardBestState.ShardCommittee
}

func (shardBestState *ShardBestState) GetShardPendingValidator() []incognitokey.CommitteePublicKey {
	return shardBestState.ShardPendingValidator
}

func (shardBestState *ShardBestState) ListShardPrivacyTokenAndPRV() []common.Hash {
	tokenIDs := []common.Hash{}
	tokenStates := statedb.ListPrivacyToken(shardBestState.GetCopiedTransactionStateDB())
	for k, _ := range tokenStates {
		tokenIDs = append(tokenIDs, k)
	}
	return tokenIDs
}
