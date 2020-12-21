package blsbftv2

import (
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
	GetCommittee() []incognitokey.CommitteePublicKey
	GetPendingCommittee() []incognitokey.CommitteePublicKey
	GetPubKeyCommitteeIndex(string) int
	GetLastProposerIndex() int
	UnmarshalBlock(blockString []byte) (common.BlockInterface, error)
	CreateNewBlock(version int, proposer string, round int, startTime int64) (common.BlockInterface, error)
	CreateNewBlockFromOldBlock(oldBlock common.BlockInterface, proposer string, startTime int64) (common.BlockInterface, error)
	InsertAndBroadcastBlock(block common.BlockInterface) error
	// ValidateAndInsertBlock(block common.BlockInterface) error
	ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	ValidatePreSignBlock(block common.BlockInterface) error
	GetShardID() int

	//for new syncker
	GetBestViewHeight() uint64
	GetFinalViewHeight() uint64
	GetBestViewHash() string
	GetFinalViewHash() string

	GetViewByHash(hash common.Hash) multiview.View
}
