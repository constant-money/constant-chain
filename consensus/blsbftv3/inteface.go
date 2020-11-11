package blsbftv3

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/multiview"
	"github.com/incognitochain/incognito-chain/wire"
	peer "github.com/libp2p/go-libp2p-peer"
)

type NodeInterface interface {
	PushMessageToChain(msg wire.Message, chain common.ChainInterface) error
	IsEnableMining() bool
	GetMiningKeys() string
	GetPrivateKey() string
	GetUserMiningState() (role string, chainID int)
	RequestMissingViewViaStream(peerID string, hashes [][]byte, fromCID int, chainName string) (err error)
	GetSelfPeerID() peer.ID
}

type ChainInterface interface {
	BestViewCommitteeFromBlock() common.Hash
	GetFinalView() multiview.View
	GetBestView() multiview.View
	GetEpoch() uint64
	GetChainName() string
	GetConsensusType() string
	GetLastBlockTimeStamp() int64
	GetMinBlkInterval() time.Duration
	GetMaxBlkCreateTime() time.Duration
	IsReady() bool
	SetReady(bool)
	GetActiveShardNumber() int
	CurrentHeight() uint64
	GetCommitteeSize() int
	IsBeaconChain() bool
	GetCommittee() []incognitokey.CommitteePublicKey
	GetPendingCommittee() []incognitokey.CommitteePublicKey
	GetPubKeyCommitteeIndex(string) int
	GetLastProposerIndex() int
	UnmarshalBlock(blockString []byte) (types.BlockInterface, error)
	CreateNewBlock(
		version int,
		proposer string,
		round int,
		startTime int64,
		committees []incognitokey.CommitteePublicKey,
		hash common.Hash) (types.BlockInterface, error)
	CreateNewBlockFromOldBlock(
		oldBlock types.BlockInterface,
		proposer string,
		startTime int64,
		committees []incognitokey.CommitteePublicKey,
		hash common.Hash) (types.BlockInterface, error)
	InsertAndBroadcastBlock(block types.BlockInterface) error
	// ValidateAndInsertBlock(block common.BlockInterface) error
	ValidateBlockSignatures(block types.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	ValidatePreSignBlock(block types.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	GetShardID() int

	//for new syncker
	GetBestViewHeight() uint64
	GetFinalViewHeight() uint64
	GetBestViewHash() string
	GetFinalViewHash() string
	GetViewByHash(hash common.Hash) multiview.View
}

//CommitteeChainHandler :
type CommitteeChainHandler interface {
	CommitteesFromViewHashForShard(hash common.Hash, shardID byte) ([]incognitokey.CommitteePublicKey, error)
	ProposerByTimeSlot(byte, int64, []incognitokey.CommitteePublicKey) incognitokey.CommitteePublicKey
	FinalView() multiview.View
}
