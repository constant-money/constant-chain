package metadata

import (

	"github.com/incognitochain/incognito-chain/common"
)

type PDECrossPoolTradeResponse struct {
	MetadataBase
	TradeStatus   string
	RequestedTxID common.Hash
}

func NewPDECrossPoolTradeResponse(
	tradeStatus string,
	requestedTxID common.Hash,
	metaType int,
) *PDECrossPoolTradeResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDECrossPoolTradeResponse{
		TradeStatus:   tradeStatus,
		RequestedTxID: requestedTxID,
		MetadataBase:  metadataBase,
	}
}

func (iRes PDECrossPoolTradeResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.TradeStatus
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

