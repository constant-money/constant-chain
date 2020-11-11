package syncker

import (
	"context"

	"github.com/incognitochain/incognito-chain/blockchain/types"

	"github.com/incognitochain/incognito-chain/incdb"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type Server interface {
	//state
	GetChainParam() *blockchain.Params
	GetUserMiningState() (role string, chainID int)

	//network
	PublishNodeState(userLayer string, shardID int) error
	RequestBeaconBlocksViaStream(ctx context.Context, peerID string, from uint64, to uint64) (blockCh chan types.BlockInterface, err error)
	RequestShardBlocksViaStream(ctx context.Context, peerID string, fromSID int, from uint64, to uint64) (blockCh chan types.BlockInterface, err error)
	RequestCrossShardBlocksViaStream(ctx context.Context, peerID string, fromSID int, toSID int, heights []uint64) (blockCh chan types.BlockInterface, err error)
	RequestCrossShardBlocksByHashViaStream(ctx context.Context, peerID string, fromSID int, toSID int, hashes [][]byte) (blockCh chan types.BlockInterface, err error)
	RequestBeaconBlocksByHashViaStream(ctx context.Context, peerID string, hashes [][]byte) (blockCh chan types.BlockInterface, err error)
	RequestShardBlocksByHashViaStream(ctx context.Context, peerID string, fromSID int, hashes [][]byte) (blockCh chan types.BlockInterface, err error)
	//database
	FetchConfirmBeaconBlockByHeight(height uint64) (*types.BeaconBlock, error)
	GetBeaconChainDatabase() incdb.Database
	FetchNextCrossShard(fromSID, toSID int, currentHeight uint64) *NextCrossShardInfo
	//StoreBeaconHashConfirmCrossShardHeight(fromSID, toSID int, height uint64, beaconHash string) error
	//FetchBeaconBlockConfirmCrossShardHeight(fromSID, toSID int, height uint64) (*blockchain.BeaconBlock, error)
}

type BeaconChainInterface interface {
	Chain
	GetShardBestViewHash() map[byte]common.Hash
	GetShardBestViewHeight() map[byte]uint64
}

type ShardChainInterface interface {
	Chain
	GetCrossShardState() map[byte]uint64
}

type Chain interface {
	GetAllViewHash() []common.Hash
	GetBestViewHeight() uint64
	GetFinalViewHeight() uint64
	SetReady(bool)
	IsReady() bool
	GetBestViewHash() string
	GetFinalViewHash() string
	GetEpoch() uint64
	ValidateBlockSignatures(block types.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	//ValidateProducerPosition(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	GetCommittee() []incognitokey.CommitteePublicKey
	CurrentHeight() uint64
	InsertBlk(block types.BlockInterface, shouldValidate bool) error
	CheckExistedBlk(block types.BlockInterface) bool
	GetCommitteeByHeight(h uint64) ([]incognitokey.CommitteePublicKey, error)
	GetCommitteeV2(types.BlockInterface) ([]incognitokey.CommitteePublicKey, error) // Using only for stream blocks by gRPC
	CommitteeStateVersion() uint
}

const (
	BeaconPoolType = iota
	ShardPoolType
	CrossShardPoolType
)
