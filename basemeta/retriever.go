package basemeta

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	zkp "github.com/incognitochain/incognito-chain/privacy/zeroknowledge"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
	"time"
)

// Interface for all types of metadata in tx
type Metadata interface {
	GetType() int
	Hash() *common.Hash
	CheckTransactionFee(Transaction, uint64, int64, *statedb.StateDB) bool
	ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error)
	ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error)
	ValidateMetadataByItself() bool
	BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error)
	CalculateSize() uint64
	VerifyMinerCreatedTxBeforeGettingInBlock(txsInBlock []Transaction, txsUsed []int, insts [][]string, instUsed []int, shardID byte, tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever) (bool, error)
	IsMinerCreatedMetaType() bool
}

// Interface for mempool which is used in metadata
type MempoolRetriever interface {
	GetSerialNumbersHashH() map[common.Hash][]common.Hash
	GetTxsInMem() map[common.Hash]TxDesc
}

type ChainRetriever interface {
	GetETHRemoveBridgeSigEpoch() uint64
	GetBCHeightBreakPointPortalV3() uint64
	GetStakingAmountShard() uint64
	GetCentralizedWebsitePaymentAddress(uint64) string
	GetBeaconHeightBreakPointBurnAddr() uint64
	GetBurningAddress(blockHeight uint64) string
	GetTransactionByHash(common.Hash) (byte, common.Hash, uint64, int, Transaction, error)
	ListPrivacyTokenAndBridgeTokenAndPRVByShardID(byte) ([]common.Hash, error)
	GetBNBChainID(beaconHeight uint64) string
	GetBTCChainID(beaconHeight uint64) string
	GetBTCHeaderChain() *btcrelaying.BlockChain
	GetPortalFeederAddress(beaconHeight uint64) string
	GetFixedRandomForShardIDCommitment(beaconHeight uint64) *privacy.Scalar
	GetSupportedCollateralTokenIDs(beaconHeight uint64) []string
	GetPortalETHContractAddrStr(beaconHeight uint64) string
	GetLatestBNBBlkHeight() (int64, error)
	GetBNBDataHash(blockHeight int64) ([]byte, error)
	CheckBlockTimeIsReached(recentBeaconHeight, beaconHeight, recentShardHeight, shardHeight uint64, duration time.Duration) bool
	IsSupportedTokenCollateralV3(beaconHeight uint64, externalTokenID string) bool
	IsPortalExchangeRateToken(beaconHeight uint64, tokenID string) bool
	IsPortalToken(beaconHeight uint64, tokenIDStr string) bool
	GetMinAmountPortalToken(tokenIDStr string, beaconHeight uint64) (uint64, error)
	IsValidPortalRemoteAddress(tokenIDStr string, remoteAddr string, beaconHeight uint64) (bool, error)
	GetPortalReplacementAddress(beaconHeight uint64) string
	CheckBlockTimeIsReachedByBeaconHeight(recentBeaconHeight, beaconHeight uint64, duration time.Duration) bool
}

type BeaconViewRetriever interface {
	GetAllCommitteeValidatorCandidate() (map[byte][]incognitokey.CommitteePublicKey, map[byte][]incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, error)
	GetAllCommitteeValidatorCandidateFlattenListFromDatabase() ([]string, error)
	GetAutoStakingList() map[string]bool
	GetAllBridgeTokens() ([]common.Hash, error)
	GetBeaconFeatureStateDB() *statedb.StateDB
	GetBeaconRewardStateDB() *statedb.StateDB
	GetBeaconSlashStateDB() *statedb.StateDB
}

type ShardViewRetriever interface {
	GetEpoch() uint64
	GetBeaconHeight() uint64
	GetStakingTx() map[string]string
	ListShardPrivacyTokenAndPRV() []common.Hash
	GetShardRewardStateDB() *statedb.StateDB
	GetCopiedFeatureStateDB() *statedb.StateDB
	GetHeight() uint64
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

// Interface for all type of transaction
type Transaction interface {
	// GET/SET FUNC
	GetMetadataType() int
	GetType() string
	GetLockTime() int64
	GetTxActualSize() uint64
	GetSenderAddrLastByte() byte
	GetTxFee() uint64
	GetTxFeeToken() uint64
	GetMetadata() Metadata
	SetMetadata(Metadata)
	GetInfo() []byte
	GetSender() []byte
	GetSigPubKey() []byte
	GetProof() *zkp.PaymentProof
	// Get receivers' data for tx
	GetReceivers() ([][]byte, []uint64)
	GetUniqueReceiver() (bool, []byte, uint64)
	GetTransferData() (bool, []byte, uint64, *common.Hash)
	// Get receivers' data for custom token tx (nil for normal tx)
	GetTokenReceivers() ([][]byte, []uint64)
	GetTokenUniqueReceiver() (bool, []byte, uint64)
	GetMetadataFromVinsTx(ChainRetriever, ShardViewRetriever, BeaconViewRetriever) (Metadata, error)
	GetTokenID() *common.Hash
	ListSerialNumbersHashH() []common.Hash
	Hash() *common.Hash
	// VALIDATE FUNC
	CheckTxVersion(int8) bool
	// CheckTransactionFee(minFeePerKbTx uint64) bool
	ValidateTxWithCurrentMempool(MempoolRetriever) error
	ValidateSanityData(ChainRetriever, ShardViewRetriever, BeaconViewRetriever, uint64) (bool, error)
	ValidateTxWithBlockChain(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error
	ValidateDoubleSpendWithBlockchain(byte, *statedb.StateDB, *common.Hash) error
	ValidateTxByItself(map[string]bool, *statedb.StateDB, *statedb.StateDB, ChainRetriever, byte, ShardViewRetriever, BeaconViewRetriever) (bool, error)
	ValidateType() bool
	ValidateTransaction(map[string]bool, *statedb.StateDB, *statedb.StateDB, byte, *common.Hash) (bool, error)
	VerifyMinerCreatedTxBeforeGettingInBlock([]Transaction, []int, [][]string, []int, byte, ChainRetriever, *AccumulatedValues, ShardViewRetriever, BeaconViewRetriever) (bool, error)
	IsPrivacy() bool
	IsCoinsBurning(ChainRetriever, ShardViewRetriever, BeaconViewRetriever, uint64) bool
	CalculateTxValue() uint64
	CalculateBurningTxValue(bcr ChainRetriever, retriever ShardViewRetriever, viewRetriever BeaconViewRetriever, beaconHeight uint64) (bool, uint64)
	IsSalaryTx() bool
	GetFullTxValues() (uint64, uint64)
	IsFullBurning(ChainRetriever, ShardViewRetriever, BeaconViewRetriever, uint64) bool
}

// Interface for Portal functions
type PortalRetriever interface {
	IsPortalMetadata(metaType int) bool
	ParseMetadata(meta interface{}) (Metadata, error)
}
