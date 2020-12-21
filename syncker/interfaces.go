package syncker

import (
	"context"

	"github.com/incognitochain/incognito-chain/incdb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type Network interface {
	//network
	RequestBeaconBlocksViaStream(ctx context.Context, peerID string, from uint64, to uint64) (blockCh chan common.BlockInterface, err error)
	RequestShardBlocksViaStream(ctx context.Context, peerID string, fromSID int, from uint64, to uint64) (blockCh chan common.BlockInterface, err error)
	RequestCrossShardBlocksViaStream(ctx context.Context, peerID string, fromSID int, toSID int, heights []uint64) (blockCh chan common.BlockInterface, err error)
	RequestCrossShardBlocksByHashViaStream(ctx context.Context, peerID string, fromSID int, toSID int, hashes [][]byte) (blockCh chan common.BlockInterface, err error)
	RequestBeaconBlocksByHashViaStream(ctx context.Context, peerID string, hashes [][]byte) (blockCh chan common.BlockInterface, err error)
	RequestShardBlocksByHashViaStream(ctx context.Context, peerID string, fromSID int, hashes [][]byte) (blockCh chan common.BlockInterface, err error)
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
	GetDatabase() incdb.Database
	GetAllViewHash() []common.Hash
	GetBestViewHeight() uint64
	GetFinalViewHeight() uint64
	SetReady(bool)
	IsReady() bool
	GetBestViewHash() string
	GetFinalViewHash() string
	GetEpoch() uint64
	ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	//ValidateProducerPosition(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	GetCommittee() []incognitokey.CommitteePublicKey
	CurrentHeight() uint64
	InsertBlk(block common.BlockInterface, shouldValidate bool) error
	CheckExistedBlk(block common.BlockInterface) bool
	GetCommitteeByHeight(h uint64) ([]incognitokey.CommitteePublicKey, error)
}

const (
	BeaconPoolType = iota
	ShardPoolType
	CrossShardPoolType
)
