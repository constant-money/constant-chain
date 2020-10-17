package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
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