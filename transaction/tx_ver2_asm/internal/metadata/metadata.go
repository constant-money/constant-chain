package metadata

import (

	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/common"
)

// Interface for all types of metadata in tx
type Metadata interface {
	GetType() int
	GetSig() []byte
	SetSig([]byte)
	ShouldSignMetaData() bool
	Hash() *common.Hash
	HashWithoutSig() *common.Hash
	IsMinerCreatedMetaType() bool
}

// This is tx struct which is really saved in tx mempool
type TxDesc struct {
	// Tx is the transaction associated with the entry.
	Tx Transaction

	// Height is the best block's height when the entry was added to the the source pool.
	Height uint64

	// Fee is the total fee the transaction associated with the entry pays.
	Fee uint64

	// FeeToken is the total token fee the transaction associated with the entry pays.
	// FeeToken is zero if tx is PRV transaction
	FeeToken uint64

	// FeePerKB is the fee the transaction pays in coin per 1000 bytes.
	FeePerKB int32
}

// Interface for mempool which is used in metadata
type MempoolRetriever interface {
	GetSerialNumbersHashH() map[common.Hash][]common.Hash
	GetTxsInMem() map[common.Hash]TxDesc
}
// Interface for all type of transaction
type Transaction interface {
	// GET/SET FUNCTION
	GetVersion() int8
	SetVersion(int8)
	GetMetadataType() int
	GetType() string
	SetType(string)
	GetLockTime() int64
	SetLockTime(int64)
	GetSenderAddrLastByte() byte
	SetGetSenderAddrLastByte(byte)
	GetTxFee() uint64
	SetTxFee(uint64)
	GetTxFeeToken() uint64
	GetInfo() []byte
	SetInfo([]byte)
	GetSigPubKey() []byte
	SetSigPubKey([]byte)
	GetSig() []byte
	SetSig([]byte)
	GetProof() privacy.Proof
	SetProof(privacy.Proof)
	GetTokenID() *common.Hash
	GetMetadata() Metadata
	SetMetadata(Metadata)

	// =================== FUNCTIONS THAT GET STUFF AND REQUIRE SOME CODING ===================
	GetTxActualSize() uint64
	GetReceivers() ([][]byte, []uint64)
	GetTransferData() (bool, []byte, uint64, *common.Hash)

	GetReceiverData() ([]coin.Coin, error)
	GetTxMintData() (bool, coin.Coin, *common.Hash, error)
	GetTxBurnData() (bool, coin.Coin, *common.Hash, error)

	ListSerialNumbersHashH() []common.Hash
	String() string
	Hash() *common.Hash
	CalculateTxValue() uint64
	// Init Transaction, the input should be params such as: TxPrivacyInitParams
	Init(interface{}) error
}

type MintData struct {
	ReturnStaking  map[string]bool
	WithdrawReward map[string]bool
	Txs            []Transaction
	TxsUsed        []int
	Insts          [][]string
	InstsUsed      []int
}