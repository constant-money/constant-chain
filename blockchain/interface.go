package blockchain

import (
	"context"
	"github.com/incognitochain/incognito-chain/multiview"

	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/pubsub"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
)

type TxPool interface {
	// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
	// LastUpdated() time.Time
	// MiningDescs returns a slice of mining descriptors for all the
	// transactions in the source pool.
	// MiningDescs() []*metadata.TxDesc
	// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
	HaveTransaction(hash *common.Hash) bool
	// RemoveTx remove tx from tx resource
	RemoveTx(txs []metadata.Transaction, isInBlock bool)
	RemoveCandidateList([]string)
	EmptyPool() bool
	MaybeAcceptTransactionForBlockProducing(metadata.Transaction, int64, *ShardBestState) (*metadata.TxDesc, error)
	MaybeAcceptBatchTransactionForBlockProducing(byte, []metadata.Transaction, int64, *ShardBestState) ([]*metadata.TxDesc, error)
	MaybeAcceptSalaryTransactionForBlockProducing(byte, metadata.Transaction, int64, *ShardBestState) (*metadata.TxDesc, error)
	//CheckTransactionFee
	// CheckTransactionFee(tx metadata.Transaction) (uint64, error)
	// Check tx validate by it self
	// ValidateTxByItSelf(tx metadata.Transaction) bool
}

type FeeEstimator interface {
	RegisterBlock(block *ShardBlock) error
}

type ConsensusEngine interface {
	ValidateProducerPosition(blk common.BlockInterface, lastProposerIdx int, committee []incognitokey.CommitteePublicKey, minCommitteeSize int) error
	ValidateProducerSig(block common.BlockInterface, consensusType string) error
	ValidateBlockCommitteSig(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	// GetCurrentMiningPublicKey() (string, string)
	// GetCurrentValidators() []*consensus.Validator
	// GetOneValidatorForEachConsensusProcess() map[int]*consensus.Validator
	// GetMiningPublicKeyByConsensus(consensusName string) (string, error)
	// GetUserRole() (string, string, int)
	// CommitteeChange(chainName string)
}

type Server interface {
	// PublishNodeState() error

	// PushMessageGetBlockBeaconByHeight(from uint64, to uint64) error
	// PushMessageGetBlockBeaconByHash(blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error
	// PushMessageGetBlockBeaconBySpecificHeight(heights []uint64, getFromPool bool) error

	// PushMessageGetBlockShardByHeight(shardID byte, from uint64, to uint64) error
	// PushMessageGetBlockShardByHash(shardID byte, blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error
	// PushMessageGetBlockShardBySpecificHeight(shardID byte, heights []uint64, getFromPool bool) error

	// PushMessageGetBlockCrossShardByHash(fromShard byte, toShard byte, blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error
	// UpdateConsensusState(role string, userPbk string, currentShard *byte, beaconCommittee []string, shardCommittee map[byte][]string)
	PushBlockToAll(block common.BlockInterface, isBeacon bool) error
}

type Highway interface {
	BroadcastCommittee(uint64, []incognitokey.CommitteePublicKey, map[byte][]incognitokey.CommitteePublicKey, map[byte][]incognitokey.CommitteePublicKey)
}

type Syncker interface {
	GetCrossShardBlocksForShardProducer(toShard byte, list map[byte][]uint64) map[byte][]interface{}
	GetCrossShardBlocksForShardValidator(toShard byte, list map[byte][]uint64) (map[byte][]interface{}, error)
	SyncMissingBeaconBlock(ctx context.Context, peerID string, fromHash common.Hash)
	SyncMissingShardBlock(ctx context.Context, peerID string, sid byte, fromHash common.Hash)
}

type Pubsub interface {
	PublishMessage(message *pubsub.Message)
}

type ChainInterface interface {
	GetBestView() multiview.View
	GetFinalView() multiview.View
}
