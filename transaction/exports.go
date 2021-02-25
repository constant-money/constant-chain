package transaction

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
)

const(
	NormalCoinType 						= utils.NormalCoinType
	CustomTokenPrivacyType 				= utils.CustomTokenPrivacyType
	CustomTokenInit 					= utils.CustomTokenInit
	CustomTokenTransfer					= utils.CustomTokenTransfer
	CustomTokenCrossShard				= utils.CustomTokenCrossShard
	CurrentTxVersion                 	= utils.CurrentTxVersion
	TxVersion0Number                 	= utils.TxVersion0Number
	TxVersion1Number                 	= utils.TxVersion1Number
	TxVersion2Number                 	= utils.TxVersion2Number
	TxConversionVersion12Number      	= utils.TxConversionNumber
	ValidateTimeForOneoutOfManyProof 	= utils.ValidateTimeForOneoutOfManyProof
	MaxSizeInfo   						= utils.MaxSizeInfo
	MaxSizeUint32 						= utils.MaxSizeUint32
	MaxSizeByte   						= utils.MaxSizeByte
)

type EstimateTxSizeParam 				= tx_generic.EstimateTxSizeParam
type TxPrivacyInitParams 				= tx_generic.TxPrivacyInitParams

func NewRandomCommitmentsProcessParam(usableInputCoins []privacy.PlainCoin, randNum int, stateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) *tx_generic.RandomCommitmentsProcessParam{
	return tx_generic.NewRandomCommitmentsProcessParam(usableInputCoins, randNum, stateDB, shardID, tokenID)
}

func RandomCommitmentsProcess(param *tx_generic.RandomCommitmentsProcessParam) (commitmentIndexs []uint64, myCommitmentIndexs []uint64, commitments [][]byte){
	return tx_generic.RandomCommitmentsProcess(param)
}

func NewTxTokenParams(senderKey *privacy.PrivateKey, paymentInfo []*privacy.PaymentInfo, inputCoin []privacy.PlainCoin,feeNativeCoin uint64, tokenParams *TokenParam, transactionStateDB *statedb.StateDB, metaData metadata.Metadata, hasPrivacyCoin bool,	hasPrivacyToken bool, shardID byte,	info []byte, bridgeStateDB *statedb.StateDB) *TxTokenParams{
	return tx_generic.NewTxTokenParams(senderKey, paymentInfo, inputCoin, feeNativeCoin, tokenParams, transactionStateDB, metaData, hasPrivacyCoin, hasPrivacyToken, shardID, info, bridgeStateDB)
}

func EstimateTxSize(estimateTxSizeParam *tx_generic.EstimateTxSizeParam) uint64 {
	return tx_generic.EstimateTxSize(estimateTxSizeParam)
}

func NewEstimateTxSizeParam(version, numInputCoins, numPayments int,
	hasPrivacy bool, metadata metadata.Metadata,
	privacyCustomTokenParams *TokenParam,
	limitFee uint64) *EstimateTxSizeParam{
	return tx_generic.NewEstimateTxSizeParam(version, numInputCoins, numPayments, hasPrivacy, metadata, privacyCustomTokenParams, limitFee)
}

func NewTxPrivacyInitParams(senderSK *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoins []privacy.PlainCoin,
	fee uint64,
	hasPrivacy bool,
	stateDB *statedb.StateDB,
	tokenID *common.Hash, // default is nil -> use for prv coin
	metaData metadata.Metadata,
	info []byte) *TxPrivacyInitParams {
	return tx_generic.NewTxPrivacyInitParams(senderSK, paymentInfo, inputCoins,	fee, hasPrivacy, stateDB, tokenID, metaData, info)
}

func GetTxVersionFromCoins(coins []privacy.PlainCoin) (int8, error){
	return tx_generic.GetTxVersionFromCoins(coins)
}