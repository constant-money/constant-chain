package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

type PDETradeResponse struct {
	MetadataBase
	TradeStatus   string
	RequestedTxID common.Hash
}

func NewPDETradeResponse(
	tradeStatus string,
	requestedTxID common.Hash,
	metaType int,
) *PDETradeResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDETradeResponse{
		TradeStatus:   tradeStatus,
		RequestedTxID: requestedTxID,
		MetadataBase:  metadataBase,
	}
}

func (iRes PDETradeResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.TradeStatus
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}