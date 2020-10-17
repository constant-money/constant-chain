package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalRedeemLiquidateExchangeRates struct {
	MetadataBase
	TokenID               string // pTokenID in incognito chain
	RedeemAmount          uint64
	RedeemerIncAddressStr string
}

type PortalRedeemLiquidateExchangeRatesAction struct {
	Meta    PortalRedeemLiquidateExchangeRates
	TxReqID common.Hash
	ShardID byte
}

type PortalRedeemLiquidateExchangeRatesContent struct {
	TokenID               string // pTokenID in incognito chain
	RedeemAmount          uint64
	RedeemerIncAddressStr string
	TxReqID               common.Hash
	ShardID               byte
	TotalPTokenReceived   uint64
}

type RedeemLiquidateExchangeRatesStatus struct {
	TxReqID             common.Hash
	TokenID             string
	RedeemerAddress     string
	RedeemAmount        uint64
	Status              byte
	TotalPTokenReceived uint64
}

func NewRedeemLiquidateExchangeRatesStatus(txReqID common.Hash, tokenID string, redeemerAddress string, redeemAmount uint64, status byte, totalPTokenReceived uint64) *RedeemLiquidateExchangeRatesStatus {
	return &RedeemLiquidateExchangeRatesStatus{TxReqID: txReqID, TokenID: tokenID, RedeemerAddress: redeemerAddress, RedeemAmount: redeemAmount, Status: status, TotalPTokenReceived: totalPTokenReceived}
}

func NewPortalRedeemLiquidateExchangeRates(
	metaType int,
	tokenID string,
	redeemAmount uint64,
	incAddressStr string,
) (*PortalRedeemLiquidateExchangeRates, error) {
	metadataBase := MetadataBase{Type: metaType}

	portalRedeemLiquidateExchangeRates := &PortalRedeemLiquidateExchangeRates{
		TokenID:               tokenID,
		RedeemAmount:          redeemAmount,
		RedeemerIncAddressStr: incAddressStr,
	}

	portalRedeemLiquidateExchangeRates.MetadataBase = metadataBase

	return portalRedeemLiquidateExchangeRates, nil
}


func (redeemReq PortalRedeemLiquidateExchangeRates) Hash() *common.Hash {
	record := redeemReq.MetadataBase.Hash().String()
	record += redeemReq.TokenID
	record += strconv.FormatUint(redeemReq.RedeemAmount, 10)
	record += redeemReq.RedeemerIncAddressStr
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}