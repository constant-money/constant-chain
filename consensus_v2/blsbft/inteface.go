package blsbft

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
	"time"
)

type NodeInterface interface {
	PushMessageToChain(msg wire.Message, chain common.ChainInterface) error
	GetMiningKeys() string
	GetPrivateKey() string
	GetUserMiningState() (role string, chainID int)
}

type ChainInterface interface {
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

	InsertAndBroadcastBlock(block common.BlockInterface) error
	CreateNewBlock(version int, proposer string, round int, startTime int64) (common.BlockInterface, error)
	// ValidateAndInsertBlock(block common.BlockInterface) error
	ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	ValidatePreSignBlock(block common.BlockInterface) error
	GetShardID() int

	//for new syncker
	GetBestViewHeight() uint64
	GetFinalViewHeight() uint64
	GetBestViewHash() string
	GetFinalViewHash() string
}
