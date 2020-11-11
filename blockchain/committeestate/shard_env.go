package committeestate

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
)

//ShardEnvBuilder : Interface for building shard environment
type ShardEnvBuilder interface {
	BuildShardInstructions(instructions [][]string) ShardEnvBuilder
	BuildBeaconBlockHash(blockHash common.Hash) ShardEnvBuilder
	BuildShardHeight(height uint64) ShardEnvBuilder
	BuildShardBlockHash(blockHash common.Hash) ShardEnvBuilder
	BuildTxs(txs []metadata.Transaction) ShardEnvBuilder
	BuildBeaconInstructions(instructions [][]string) ShardEnvBuilder
	BuildBeaconHeight(height uint64) ShardEnvBuilder
	BuildChainParamEpoch(chainParamEpoch uint64) ShardEnvBuilder
	BuildEpochBreakPointSwapNewKey(epochBreakPointSwapNewKey []uint64) ShardEnvBuilder
	BuildShardID(id byte) ShardEnvBuilder
	BuildMaxShardCommitteeSize(maxShardCommitteeSize int) ShardEnvBuilder
	BuildMinShardCommitteeSize(minShardCommitteeSize int) ShardEnvBuilder
	BuildOffset(offset int) ShardEnvBuilder
	BuildSwapOffset(swapOffset int) ShardEnvBuilder
	BuildStakingTx(stakingTx map[string]string) ShardEnvBuilder
	BuildNumberOfFixedBlockValidators(int) ShardEnvBuilder
	BuildCommitteesFromBlock(common.Hash) ShardEnvBuilder
	BuildCommitteesFromBeaconView([]incognitokey.CommitteePublicKey) ShardEnvBuilder
	Build() ShardCommitteeStateEnvironment
}

//NewShardEnvBuilder :
func NewShardEnvBuilder() ShardEnvBuilder {
	return &shardCommitteeStateEnvironment{}
}

// ShardCommitteeStateEnvironment :
type ShardCommitteeStateEnvironment interface {
	ShardHeight() uint64
	ShardBlockHash() common.Hash
	BeaconBlockHash() common.Hash
	Txs() []metadata.Transaction
	BeaconInstructions() [][]string
	ShardInstructions() [][]string
	BeaconHeight() uint64
	ChainParamEpoch() uint64
	EpochBreakPointSwapNewKey() []uint64
	ShardID() byte
	MaxShardCommitteeSize() int
	MinShardCommitteeSize() int
	Offset() int
	SwapOffset() int
	StakingTx() map[string]string
	CommitteesFromBlock() common.Hash
	NumberOfFixedBlockValidators() int
	CommitteesFromBeaconView() []incognitokey.CommitteePublicKey // This Field Is Only Use For Swap Committee
}

//shardCommitteeStateEnvironment :
type shardCommitteeStateEnvironment struct {
	shardHeight                  uint64
	shardBlockHash               common.Hash
	beaconBlockHash              common.Hash
	shardInstructions            [][]string
	beaconInstructions           [][]string
	txs                          []metadata.Transaction
	beaconHeight                 uint64
	chainParamEpoch              uint64
	epochBreakPointSwapNewKey    []uint64
	shardID                      byte
	maxShardCommitteeSize        int
	minShardCommitteeSize        int
	offset                       int
	swapOffset                   int
	stakingTx                    map[string]string
	numberOfFixedBlockValidators int
	committeesFromBlock          common.Hash
	committeesFromBeaconView     []incognitokey.CommitteePublicKey
}

//BuildCommitteesFromBeacon :
func (env *shardCommitteeStateEnvironment) BuildCommitteesFromBeaconView(committees []incognitokey.CommitteePublicKey) ShardEnvBuilder {
	env.committeesFromBeaconView = committees
	return env
}

//BuildCommitteeFromBlock :
func (env *shardCommitteeStateEnvironment) BuildCommitteesFromBlock(hash common.Hash) ShardEnvBuilder {
	env.committeesFromBlock = hash
	return env
}

//BuildShardHeight :
func (env *shardCommitteeStateEnvironment) BuildShardHeight(height uint64) ShardEnvBuilder {
	env.shardHeight = height
	return env
}

//BuildBeaconBlockHash :
func (env *shardCommitteeStateEnvironment) BuildBeaconBlockHash(blockHash common.Hash) ShardEnvBuilder {
	env.beaconBlockHash = blockHash
	return env
}

//BuildShardBlockHash :
func (env *shardCommitteeStateEnvironment) BuildShardBlockHash(blockHash common.Hash) ShardEnvBuilder {
	env.shardBlockHash = blockHash
	return env
}

//BuildTxs :
func (env *shardCommitteeStateEnvironment) BuildTxs(txs []metadata.Transaction) ShardEnvBuilder {
	env.txs = txs
	return env
}

//BuildShardInstructions :
func (env *shardCommitteeStateEnvironment) BuildShardInstructions(instructions [][]string) ShardEnvBuilder {
	env.shardInstructions = instructions
	return env
}

//BuildBeaconInstructions :
func (env *shardCommitteeStateEnvironment) BuildBeaconInstructions(instructions [][]string) ShardEnvBuilder {
	env.beaconInstructions = instructions
	return env
}

//BuildBeaconHeight :
func (env *shardCommitteeStateEnvironment) BuildBeaconHeight(height uint64) ShardEnvBuilder {
	env.beaconHeight = height
	return env
}

//BuildChainParamEpoch :
func (env *shardCommitteeStateEnvironment) BuildChainParamEpoch(chainParamEpoch uint64) ShardEnvBuilder {
	env.chainParamEpoch = chainParamEpoch
	return env
}

//BuildEpochBreakPointSwapNewKey :
func (env *shardCommitteeStateEnvironment) BuildEpochBreakPointSwapNewKey(
	epochBreakPointSwapNewKey []uint64) ShardEnvBuilder {
	env.epochBreakPointSwapNewKey = epochBreakPointSwapNewKey
	return env
}

//BuildShardID :
func (env *shardCommitteeStateEnvironment) BuildShardID(id byte) ShardEnvBuilder {
	env.shardID = id
	return env
}

//BuildMaxShardCommitteeSize :
func (env *shardCommitteeStateEnvironment) BuildMaxShardCommitteeSize(maxShardCommitteeSize int) ShardEnvBuilder {
	env.maxShardCommitteeSize = maxShardCommitteeSize
	return env
}

//BuildMinShardCommitteeSize :
func (env *shardCommitteeStateEnvironment) BuildMinShardCommitteeSize(minShardCommitteeSize int) ShardEnvBuilder {
	env.minShardCommitteeSize = minShardCommitteeSize
	return env
}

//BuildOffset :
func (env *shardCommitteeStateEnvironment) BuildOffset(offset int) ShardEnvBuilder {
	env.offset = offset
	return env
}

//BuildSwapOffset :
func (env *shardCommitteeStateEnvironment) BuildSwapOffset(swapOffset int) ShardEnvBuilder {
	env.swapOffset = swapOffset
	return env
}

//BuildStakingTx :
func (env *shardCommitteeStateEnvironment) BuildStakingTx(stakingTx map[string]string) ShardEnvBuilder {
	env.stakingTx = stakingTx
	return env
}

func (env *shardCommitteeStateEnvironment) BuildNumberOfFixedBlockValidators(
	numberOfFixedBlockValidators int) ShardEnvBuilder {
	env.numberOfFixedBlockValidators = numberOfFixedBlockValidators
	return env
}

////

//ShardHeight :
func (env *shardCommitteeStateEnvironment) ShardHeight() uint64 {
	return env.shardHeight
}

//ShardBlockHash :
func (env *shardCommitteeStateEnvironment) ShardBlockHash() common.Hash {
	return env.shardBlockHash
}

//BeaconBlockHash :
func (env *shardCommitteeStateEnvironment) BeaconBlockHash() common.Hash {
	return env.beaconBlockHash
}

//Txs :
func (env *shardCommitteeStateEnvironment) Txs() []metadata.Transaction {
	return env.txs
}

//BeaconInstructions :
func (env *shardCommitteeStateEnvironment) BeaconInstructions() [][]string {
	return env.beaconInstructions
}

//ShardInstructions :
func (env *shardCommitteeStateEnvironment) ShardInstructions() [][]string {
	return env.shardInstructions
}

//BeaconHeight :
func (env *shardCommitteeStateEnvironment) BeaconHeight() uint64 {
	return env.beaconHeight
}

//ChainParamEpoch :
func (env *shardCommitteeStateEnvironment) ChainParamEpoch() uint64 {
	return env.chainParamEpoch
}

//EpochBreakPointSwapNewKey :
func (env *shardCommitteeStateEnvironment) EpochBreakPointSwapNewKey() []uint64 {
	return env.epochBreakPointSwapNewKey
}

//ShardID :
func (env *shardCommitteeStateEnvironment) ShardID() byte {
	return env.shardID
}

//MaxShardCommitteeSize :
func (env *shardCommitteeStateEnvironment) MaxShardCommitteeSize() int {
	return env.maxShardCommitteeSize
}

//MinShardCommitteeSize :
func (env *shardCommitteeStateEnvironment) MinShardCommitteeSize() int {
	return env.minShardCommitteeSize
}

//Offset :
func (env *shardCommitteeStateEnvironment) Offset() int {
	return env.offset
}

//SwapOffset :
func (env *shardCommitteeStateEnvironment) SwapOffset() int {
	return env.swapOffset
}

//StakingTx :
func (env *shardCommitteeStateEnvironment) StakingTx() map[string]string {
	return env.stakingTx
}

func (env *shardCommitteeStateEnvironment) NumberOfFixedBlockValidators() int {
	return env.numberOfFixedBlockValidators
}

func (env *shardCommitteeStateEnvironment) CommitteesFromBlock() common.Hash {
	return env.committeesFromBlock
}

func (env *shardCommitteeStateEnvironment) CommitteesFromBeaconView() []incognitokey.CommitteePublicKey {
	return env.committeesFromBeaconView
}

//Build :
func (env *shardCommitteeStateEnvironment) Build() ShardCommitteeStateEnvironment {
	return env
}
