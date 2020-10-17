package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalRedeemLiquidateExchangeRatesResponse struct {
	MetadataBase
	RequestStatus    string
	ReqTxID          common.Hash
	RequesterAddrStr string
	RedeemAmount     uint64
	Amount           uint64
	TokenID          string
	SharedRandom       []byte
}

func NewPortalRedeemLiquidateExchangeRatesResponse(
	requestStatus string,
	reqTxID common.Hash,
	requesterAddressStr string,
	redeemAmount uint64,
	amount uint64,
	tokenID string,
	metaType int,
) *PortalRedeemLiquidateExchangeRatesResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PortalRedeemLiquidateExchangeRatesResponse{
		RequestStatus:    requestStatus,
		ReqTxID:          reqTxID,
		MetadataBase:     metadataBase,
		RequesterAddrStr: requesterAddressStr,
		RedeemAmount:     redeemAmount,
		Amount:           amount,
		TokenID:          tokenID,
	}
}

func (iRes PortalRedeemLiquidateExchangeRatesResponse) Hash() *common.Hash {
	record := iRes.MetadataBase.Hash().String()
	record += iRes.RequestStatus
	record += iRes.ReqTxID.String()
	record += iRes.RequesterAddrStr
	record += strconv.FormatUint(iRes.RedeemAmount, 10)
	record += strconv.FormatUint(iRes.Amount, 10)
	record += iRes.TokenID
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PortalRedeemLiquidateExchangeRatesResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}