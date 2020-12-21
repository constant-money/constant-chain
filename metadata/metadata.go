package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy"

	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/relaying/bnb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
)

// Interface for all types of metadata in tx
type Metadata interface {
	GetType() int
	GetSig() []byte
	SetSig([]byte)
	ShouldSignMetaData() bool
	Hash() *common.Hash
	HashWithoutSig() *common.Hash
	CheckTransactionFee(Transaction, uint64, int64, *statedb.StateDB) bool
	ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error)
	ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error)
	ValidateMetadataByItself() bool
	BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error)
	CalculateSize() uint64
	VerifyMinerCreatedTxBeforeGettingInBlock(mintData *MintData, shardID byte, tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever) (bool, error)
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
	GetOTAHashH() map[common.Hash][]common.Hash
}

type ChainRetriever interface {
	GetETHRemoveBridgeSigEpoch() uint64
	GetStakingAmountShard() uint64
	GetCentralizedWebsitePaymentAddress(uint64) string
	GetBeaconHeightBreakPointBurnAddr() uint64
	GetBurningAddress(blockHeight uint64) string
	GetTransactionByHash(common.Hash) (byte, common.Hash, uint64, int, Transaction, error)
	ListPrivacyTokenAndBridgeTokenAndPRVByShardID(byte) ([]common.Hash, error)
	GetBNBChainID() string
	GetBTCChainID() string
	GetBTCHeaderChain() *btcrelaying.BlockChain
	GetPortalFeederAddress() string
	IsAfterNewZKPCheckPoint(beaconHeight uint64) bool
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
	GetTxFullBurnData() (bool, coin.Coin, coin.Coin, *common.Hash, error)
	ListOTAHashH() []common.Hash
	ListSerialNumbersHashH() []common.Hash
	String() string
	Hash() *common.Hash
	CalculateTxValue() uint64

	// =================== FUNCTION THAT CHECK STUFFS  ===================
	CheckTxVersion(int8) bool
	CheckAuthorizedSender([]byte) (bool, error)
	ShouldSignMetaData() bool
	IsSalaryTx() bool
	IsPrivacy() bool
	IsCoinsBurning(ChainRetriever, ShardViewRetriever, BeaconViewRetriever, uint64) bool

	// =================== FUNCTIONS THAT VALIDATE STUFFS ===================
	ValidateTxReturnStaking(stateDB *statedb.StateDB) bool
	ValidateTxSalary(*statedb.StateDB) (bool, error)
	ValidateTxWithCurrentMempool(MempoolRetriever) error
	ValidateSanityData(ChainRetriever, ShardViewRetriever, BeaconViewRetriever, uint64) (bool, error)
	ValidateTxWithBlockChain(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error
	ValidateDoubleSpendWithBlockchain(byte, *statedb.StateDB, *common.Hash) error
	ValidateTxByItself(map[string]bool, *statedb.StateDB, *statedb.StateDB, ChainRetriever, byte, ShardViewRetriever, BeaconViewRetriever) (bool, error)
	ValidateType() bool
	ValidateTransaction(map[string]bool, *statedb.StateDB, *statedb.StateDB, byte, *common.Hash) (bool, []privacy.Proof, error)
	VerifyMinerCreatedTxBeforeGettingInBlock(*MintData, byte, ChainRetriever, *AccumulatedValues, ShardViewRetriever, BeaconViewRetriever) (bool, error)

	// Init Transaction, the input should be params such as: TxPrivacyInitParams
	Init(interface{}) error
	// Verify the init function above, which verify zero knowledge proof and signatures
	Verify(map[string]bool, *statedb.StateDB, *statedb.StateDB, byte, *common.Hash) (bool, error)
}

type MintData struct {
	ReturnStaking  map[string]bool
	WithdrawReward map[string]bool
	Txs            []Transaction
	TxsUsed        []int
	Insts          [][]string
	InstsUsed      []int
}

func getPDEPoolPair(
	prvIDStr, tokenIDStr string,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (*rawdbv2.PDEPoolForPair, error) {
	var pdePoolForPair rawdbv2.PDEPoolForPair
	var err error
	poolPairBytes := []byte{}
	if beaconHeight == -1 {
		poolPairBytes, err = statedb.GetLatestPDEPoolForPair(stateDB, prvIDStr, tokenIDStr)
	} else {
		poolPairBytes, err = statedb.GetPDEPoolForPair(stateDB, uint64(beaconHeight), prvIDStr, tokenIDStr)
	}
	if err != nil {
		return nil, err
	}
	if len(poolPairBytes) == 0 {
		return nil, NewMetadataTxError(CouldNotGetExchangeRateError, fmt.Errorf("Could not find out pdePoolForPair with token ids: %s & %s", prvIDStr, tokenIDStr))
	}
	err = json.Unmarshal(poolPairBytes, &pdePoolForPair)
	if err != nil {
		return nil, err
	}
	return &pdePoolForPair, nil
}

func isPairValid(poolPair *rawdbv2.PDEPoolForPair, beaconHeight int64) bool {
	if poolPair == nil {
		return false
	}
	prvIDStr := common.PRVCoinID.String()
	if poolPair.Token1IDStr == prvIDStr &&
		poolPair.Token1PoolValue < uint64(common.MinTxFeesOnTokenRequirement) &&
		beaconHeight >= common.BeaconBlockHeighMilestoneForMinTxFeesOnTokenRequirement {
		return false
	}
	if poolPair.Token2IDStr == prvIDStr &&
		poolPair.Token2PoolValue < uint64(common.MinTxFeesOnTokenRequirement) &&
		beaconHeight >= common.BeaconBlockHeighMilestoneForMinTxFeesOnTokenRequirement {
		return false
	}
	return true
}

func convertValueBetweenCurrencies(
	amount uint64,
	currentCurrencyIDStr string,
	tokenID *common.Hash,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (float64, error) {
	prvIDStr := common.PRVCoinID.String()
	tokenIDStr := tokenID.String()
	pdePoolForPair, err := getPDEPoolPair(prvIDStr, tokenIDStr, beaconHeight, stateDB)
	if err != nil {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	if !isPairValid(pdePoolForPair, beaconHeight) {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, errors.New("PRV pool size on pdex is smaller minimum initial adding liquidity amount"))
	}
	invariant := float64(0)
	invariant = float64(pdePoolForPair.Token1PoolValue) * float64(pdePoolForPair.Token2PoolValue)
	if invariant == 0 {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	if pdePoolForPair.Token1IDStr == currentCurrencyIDStr {
		remainingValue := invariant / (float64(pdePoolForPair.Token1PoolValue) + float64(amount))
		if float64(pdePoolForPair.Token2PoolValue) <= remainingValue {
			return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
		}
		return float64(pdePoolForPair.Token2PoolValue) - remainingValue, nil
	}
	remainingValue := invariant / (float64(pdePoolForPair.Token2PoolValue) + float64(amount))
	if float64(pdePoolForPair.Token1PoolValue) <= remainingValue {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	return float64(pdePoolForPair.Token1PoolValue) - remainingValue, nil
}

// return error if there is no exchange rate between native token and privacy token
// beaconHeight = -1: get the latest beacon height
func ConvertNativeTokenToPrivacyToken(
	nativeTokenAmount uint64,
	tokenID *common.Hash,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (float64, error) {
	return convertValueBetweenCurrencies(
		nativeTokenAmount,
		common.PRVCoinID.String(),
		tokenID,
		beaconHeight,
		stateDB,
	)
}

// return error if there is no exchange rate between native token and privacy token
// beaconHeight = -1: get the latest beacon height
func ConvertPrivacyTokenToNativeToken(
	privacyTokenAmount uint64,
	tokenID *common.Hash,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (float64, error) {
	return convertValueBetweenCurrencies(
		privacyTokenAmount,
		tokenID.String(),
		tokenID,
		beaconHeight,
		stateDB,
	)
}

func IsValidRemoteAddress(
	bcr ChainRetriever,
	remoteAddress string,
	tokenID string,
	chainID string,
) bool {
	if tokenID == common.PortalBNBIDStr {
		return bnb.IsValidBNBAddress(remoteAddress, chainID)
	} else if tokenID == common.PortalBTCIDStr {
		btcHeaderChain := bcr.GetBTCHeaderChain()
		if btcHeaderChain == nil {
			return false
		}
		return btcHeaderChain.IsBTCAddressValid(remoteAddress)
	}
	return false
}

func GetChainIDByTokenID(tokenID string, chainRetriever ChainRetriever) string {
	if tokenID == common.PortalBNBIDStr {
		return chainRetriever.GetBNBChainID()
	} else if tokenID == common.PortalBTCIDStr {
		return chainRetriever.GetBTCChainID()
	}
	return ""
}
