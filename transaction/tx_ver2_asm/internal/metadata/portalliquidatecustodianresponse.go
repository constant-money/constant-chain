package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalLiquidateCustodianResponse struct {
	MetadataBase
	UniqueRedeemID         string
	MintedCollateralAmount uint64 // minted PRV amount for sending back to users
	RedeemerIncAddressStr  string
	CustodianIncAddressStr string
	SharedRandom       []byte
}

func NewPortalLiquidateCustodianResponse(
	uniqueRedeemID string,
	mintedAmount uint64,
	redeemerIncAddressStr string,
	custodianIncAddressStr string,
	metaType int,
) *PortalLiquidateCustodianResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PortalLiquidateCustodianResponse{
		MetadataBase:           metadataBase,
		UniqueRedeemID:         uniqueRedeemID,
		MintedCollateralAmount: mintedAmount,
		RedeemerIncAddressStr:  redeemerIncAddressStr,
		CustodianIncAddressStr: custodianIncAddressStr,
	}
}


func (iRes PortalLiquidateCustodianResponse) Hash() *common.Hash {
	record := iRes.UniqueRedeemID
	record += strconv.FormatUint(iRes.MintedCollateralAmount, 10)
	record += iRes.RedeemerIncAddressStr
	record += iRes.CustodianIncAddressStr
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
func (iRes *PortalLiquidateCustodianResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}