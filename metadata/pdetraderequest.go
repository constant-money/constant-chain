package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/wallet"
	"reflect"
	"strconv"
)

// PDETradeRequest - privacy dex trade
type PDETradeRequest struct {
	TokenIDToBuyStr     string
	TokenIDToSellStr    string
	SellAmount          uint64 // must be equal to vout value
	MinAcceptableAmount uint64
	TradingFee          uint64
	TraderAddressStr    string
	TxRandomStr         string
	MetadataBase
}

type PDETradeRequestAction struct {
	Meta    PDETradeRequest
	TxReqID common.Hash
	ShardID byte
}

type TokenPoolValueOperation struct {
	Operator string
	Value    uint64
}

type PDETradeAcceptedContent struct {
	TraderAddressStr         string
	TxRandomStr              string
	TokenIDToBuyStr          string
	ReceiveAmount            uint64
	Token1IDStr              string
	Token2IDStr              string
	Token1PoolValueOperation TokenPoolValueOperation
	Token2PoolValueOperation TokenPoolValueOperation
	ShardID                  byte
	RequestedTxID            common.Hash
}

func NewPDETradeRequest(
	tokenIDToBuyStr string,
	tokenIDToSellStr string,
	sellAmount uint64,
	minAcceptableAmount uint64,
	tradingFee uint64,
	traderAddressStr string,
	txRandomStr string,
	metaType int,
) (*PDETradeRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	pdeTradeRequest := &PDETradeRequest{
		TokenIDToBuyStr:     tokenIDToBuyStr,
		TokenIDToSellStr:    tokenIDToSellStr,
		SellAmount:          sellAmount,
		MinAcceptableAmount: minAcceptableAmount,
		TradingFee:          tradingFee,
		TraderAddressStr:    traderAddressStr,
		TxRandomStr:         txRandomStr,
	}
	pdeTradeRequest.MetadataBase = metadataBase
	return pdeTradeRequest, nil
}

func (pc PDETradeRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	// NOTE: verify supported tokens pair as needed
	return true, nil
}

func (pc PDETradeRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	if tx.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(tx).String() == "*transaction.Tx" {
		return true, true, nil
	}

	if len(pc.TxRandomStr) > 0 {
		_, _, err := coin.ParseOTAInfoFromString(pc.TraderAddressStr, pc.TxRandomStr)
		if err != nil {
			return false, false, err
		}
	} else {
		keyWallet, err := wallet.Base58CheckDeserialize(pc.TraderAddressStr)
		if err != nil {
			return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TraderAddressStr incorrect"))
		}
		traderAddr := keyWallet.KeySet.PaymentAddress
		if len(traderAddr.Pk) == 0 {
			return false, false, errors.New("Wrong request info's trader address")
		}
	}

	isBurned, burnCoin, burnedTokenID, err := tx.GetTxBurnData()
	if err != nil {
		return false, false, err
	}
	if !isBurned {
		return false, false, errors.New("Error This is not Tx Burn")
	}
	if (pc.SellAmount + pc.TradingFee) != burnCoin.GetValue() {
		return false, false, errors.New("Error Selling amount should be equal to the burned amount")
	}
	if pc.SellAmount > burnCoin.GetValue() || pc.TradingFee > burnCoin.GetValue() {
		return false, false, errors.New("Neither selling amount nor trading fee allows to be larger than the tx value")
	}

	tokenIDToSell, err := common.Hash{}.NewHashFromStr(pc.TokenIDToSellStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenIDToSellStr incorrect"))
	}
	if !bytes.Equal(burnedTokenID[:], tokenIDToSell[:]) {
		return false, false, errors.New("Wrong request info's token id, it should be equal to tx's token id.")
	}

	if tx.GetType() == common.TxNormalType && pc.TokenIDToSellStr != common.PRVCoinID.String() {
		return false, false, errors.New("With tx normal privacy, the tokenIDStr should be PRV, not custom token.")
	}

	if tx.GetType() == common.TxCustomTokenPrivacyType && pc.TokenIDToSellStr == common.PRVCoinID.String() {
		return false, false, errors.New("With tx custome token privacy, the tokenIDStr should not be PRV, but custom token.")
	}

	return true, true, nil
}

func (pc PDETradeRequest) ValidateMetadataByItself() bool {
	return pc.Type == PDETradeRequestMeta
}

func (pc PDETradeRequest) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.TokenIDToBuyStr
	record += pc.TokenIDToSellStr
	record += pc.TraderAddressStr
	if len(pc.TxRandomStr) > 0 {
		record += pc.TxRandomStr
	}
	record += strconv.FormatUint(pc.SellAmount, 10)
	record += strconv.FormatUint(pc.MinAcceptableAmount, 10)
	record += strconv.FormatUint(pc.TradingFee, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc *PDETradeRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PDETradeRequestAction{
		Meta:    *pc,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(PDETradeRequestMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (pc *PDETradeRequest) CalculateSize() uint64 {
	return calculateSize(pc)
}
