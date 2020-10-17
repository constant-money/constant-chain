package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

// PortalRedeemRequest - portal user redeem requests to get public token by burning ptoken
// metadata - redeem request - create normal tx with this metadata
type PortalLiquidateCustodian struct {
	MetadataBase
	UniqueRedeemID           string
	TokenID                  string // pTokenID in incognito chain
	RedeemPubTokenAmount     uint64
	MintedCollateralAmount   uint64 // minted PRV amount for sending back to users
	RedeemerIncAddressStr    string
	CustodianIncAddressStr   string
	LiquidatedByExchangeRate bool
}

// PortalLiquidateCustodianContent - Beacon builds a new instruction with this content after detecting custodians run away
// It will be appended to beaconBlock
type PortalLiquidateCustodianContent struct {
	MetadataBase
	UniqueRedeemID                 string
	TokenID                        string // pTokenID in incognito chain
	RedeemPubTokenAmount           uint64
	LiquidatedCollateralAmount     uint64 // minted PRV amount for sending back to users
	RemainUnlockAmountForCustodian uint64
	RedeemerIncAddressStr          string
	CustodianIncAddressStr         string
	LiquidatedByExchangeRate       bool
	ShardID                        byte
}

// PortalLiquidateCustodianStatus - Beacon tracks status of custodian liquidation into db
type PortalLiquidateCustodianStatus struct {
	Status                         byte
	UniqueRedeemID                 string
	TokenID                        string // pTokenID in incognito chain
	RedeemPubTokenAmount           uint64
	LiquidatedCollateralAmount     uint64 // minted PRV amount for sending back to users
	RemainUnlockAmountForCustodian uint64
	RedeemerIncAddressStr          string
	CustodianIncAddressStr         string
	LiquidatedByExchangeRate       bool
	ShardID                        byte
	LiquidatedBeaconHeight         uint64
}

func NewPortalLiquidateCustodian(
	metaType int,
	uniqueRedeemID string,
	tokenID string,
	redeemAmount uint64,
	mintedCollateralAmount uint64,
	redeemerIncAddressStr string,
	custodianIncAddressStr string,
	liquidatedByExchangeRate bool) (*PortalLiquidateCustodian, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	liquidCustodianMeta := &PortalLiquidateCustodian{
		UniqueRedeemID:           uniqueRedeemID,
		TokenID:                  tokenID,
		RedeemPubTokenAmount:     redeemAmount,
		MintedCollateralAmount:   mintedCollateralAmount,
		RedeemerIncAddressStr:    redeemerIncAddressStr,
		CustodianIncAddressStr:   custodianIncAddressStr,
		LiquidatedByExchangeRate: liquidatedByExchangeRate,
	}
	liquidCustodianMeta.MetadataBase = metadataBase
	return liquidCustodianMeta, nil
}

func (liqCustodian PortalLiquidateCustodian) Hash() *common.Hash {
	record := liqCustodian.MetadataBase.Hash().String()
	record += liqCustodian.UniqueRedeemID
	record += liqCustodian.TokenID
	record += strconv.FormatUint(liqCustodian.RedeemPubTokenAmount, 10)
	record += strconv.FormatUint(liqCustodian.MintedCollateralAmount, 10)
	record += liqCustodian.RedeemerIncAddressStr
	record += liqCustodian.CustodianIncAddressStr
	record += strconv.FormatBool(liqCustodian.LiquidatedByExchangeRate)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
