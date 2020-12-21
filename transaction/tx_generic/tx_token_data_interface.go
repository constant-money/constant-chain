package tx_generic

import (
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
)

// Interface for Transaction Transfer Token
type TransactionToken interface {
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
	GetMetadata() metadata.Metadata
	SetMetadata(metadata.Metadata)

	GetTxTokenData() TxTokenData
	SetTxTokenData(TxTokenData) error
	GetTxBase() metadata.Transaction
	SetTxBase(metadata.Transaction) error
	GetTxNormal() metadata.Transaction
	SetTxNormal(metadata.Transaction) error

	// =================== FUNCTIONS THAT GET STUFF AND REQUIRE SOME CODING ===================
	GetTxActualSize() uint64
	GetReceivers() ([][]byte, []uint64)
	GetTransferData() (bool, []byte, uint64, *common.Hash)

	GetReceiverData() ([]privacy.Coin, error)
	GetTxMintData() (bool, privacy.Coin, *common.Hash, error)
	GetTxBurnData() (bool, privacy.Coin, *common.Hash, error)
	ListOTAHashH() []common.Hash
	GetTxFullBurnData() (bool, privacy.Coin, privacy.Coin, *common.Hash, error)
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
	IsCoinsBurning(metadata.ChainRetriever, metadata.ShardViewRetriever, metadata.BeaconViewRetriever, uint64) bool

	// =================== FUNCTIONS THAT VALIDATE STUFFS ===================
	ValidateTxReturnStaking(stateDB *statedb.StateDB) bool
	ValidateTxSalary(*statedb.StateDB) (bool, error)
	ValidateTxWithCurrentMempool(metadata.MempoolRetriever) error
	ValidateSanityData(metadata.ChainRetriever, metadata.ShardViewRetriever, metadata.BeaconViewRetriever, uint64) (bool, error)
	ValidateTxWithBlockChain(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error
	ValidateDoubleSpendWithBlockchain(byte, *statedb.StateDB, *common.Hash) error
	ValidateTxByItself(map[string]bool, *statedb.StateDB, *statedb.StateDB, metadata.ChainRetriever, byte, metadata.ShardViewRetriever, metadata.BeaconViewRetriever) (bool, error)
	ValidateType() bool
	ValidateTransaction(map[string]bool, *statedb.StateDB, *statedb.StateDB, byte, *common.Hash) (bool, []privacy.Proof, error)
	VerifyMinerCreatedTxBeforeGettingInBlock(*metadata.MintData, byte, metadata.ChainRetriever, *metadata.AccumulatedValues, metadata.ShardViewRetriever, metadata.BeaconViewRetriever) (bool, error)

	// Init Transaction, the input should be params such as: TxPrivacyInitParams
	Init(interface{}) error
	// Verify the init function above, which verify zero knowledge proof and signatures
	Verify(map[string]bool, *statedb.StateDB, *statedb.StateDB, byte, *common.Hash) (bool, error)
}


type TxTokenData struct {
	// TxNormal is the normal transaction, it will never be token transaction
	TxNormal       metadata.Transaction
	PropertyID     common.Hash // = hash of TxCustomTokenprivacy data
	PropertyName   string
	PropertySymbol string

	Type     int    // action type
	Mintable bool   // default false
	Amount   uint64 // init amount
}

func (txData TxTokenData) GetPropertyID() common.Hash { return txData.PropertyID }
func (txData TxTokenData) GetPropertyName() string { return txData.PropertyName }
func (txData TxTokenData) GetPropertySymbol() string { return txData.PropertySymbol }
func (txData TxTokenData) GetType() int { return txData.Type }
func (txData TxTokenData) IsMintable() bool { return txData.Mintable }
func (txData TxTokenData) GetAmount() uint64 { return txData.Amount }


func (txData *TxTokenData) SetPropertyID(propID common.Hash) { txData.PropertyID = propID }
func (txData *TxTokenData) SetPropertyName(propertyName string) { txData.PropertyName = propertyName }
func (txData *TxTokenData) SetPropertySymbol(propertySymbol string) { txData.PropertySymbol = propertySymbol }
func (txData *TxTokenData) SetType(t int) { txData.Type = t }
func (txData *TxTokenData) SetMintable(mintable bool) { txData.Mintable = mintable }
func (txData *TxTokenData) SetAmount(amount uint64) { txData.Amount = amount }

func (txData TxTokenData) String() string {
	record := txData.PropertyName
	record += txData.PropertySymbol
	record += fmt.Sprintf("%d", txData.Amount)
	if txData.TxNormal.GetProof() != nil {
		inputCoins := txData.TxNormal.GetProof().GetInputCoins()
		outputCoins := txData.TxNormal.GetProof().GetOutputCoins()
		for _, out := range outputCoins {
			publicKeyBytes := []byte{}
			if out.GetPublicKey() != nil {
				publicKeyBytes = out.GetPublicKey().ToBytesS()
			}
			record += string(publicKeyBytes)
			record += strconv.FormatUint(out.GetValue(), 10)
		}
		for _, in := range inputCoins {
			publicKeyBytes := []byte{}
			if in.GetPublicKey() != nil {
				publicKeyBytes = in.GetPublicKey().ToBytesS()
			}
			record += string(publicKeyBytes)
			if in.GetValue() > 0 {
				record += strconv.FormatUint(in.GetValue(), 10)
			}
		}
	}
	return record
}

func (txData TxTokenData) Hash() (*common.Hash, error) {
	point := privacy.HashToPoint([]byte(txData.String()))
	hash := new(common.Hash)
	err := hash.SetBytes(point.ToBytesS())
	if err != nil {
		return nil, err
	}
	return hash, nil
}

