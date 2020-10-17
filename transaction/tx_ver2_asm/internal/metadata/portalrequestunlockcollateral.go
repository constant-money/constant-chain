package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

// PortalRequestUnlockCollateral - portal custodian requests unlock collateral (after returning pubToken to user)
// metadata - custodian requests unlock collateral - create normal tx with this metadata
type PortalRequestUnlockCollateral struct {
	MetadataBase
	UniqueRedeemID      string
	TokenID             string // pTokenID in incognito chain
	CustodianAddressStr string
	RedeemAmount        uint64
	RedeemProof         string
}

// PortalRequestUnlockCollateralAction - shard validator creates instruction that contain this action content
// it will be append to ShardToBeaconBlock
type PortalRequestUnlockCollateralAction struct {
	Meta    PortalRequestUnlockCollateral
	TxReqID common.Hash
	ShardID byte
}

// PortalRequestUnlockCollateralContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalRequestUnlockCollateralContent struct {
	UniqueRedeemID      string
	TokenID             string // pTokenID in incognito chain
	CustodianAddressStr string
	RedeemAmount        uint64
	UnlockAmount        uint64 // prv
	RedeemProof         string
	TxReqID             common.Hash
	ShardID             byte
}

// PortalRequestUnlockCollateralStatus - Beacon tracks status of request unlock collateral amount into db
type PortalRequestUnlockCollateralStatus struct {
	Status              byte
	UniqueRedeemID      string
	TokenID             string // pTokenID in incognito chain
	CustodianAddressStr string
	RedeemAmount        uint64
	UnlockAmount        uint64 // prv
	RedeemProof         string
	TxReqID             common.Hash
}

func NewPortalRequestUnlockCollateral(
	metaType int,
	uniqueRedeemID string,
	tokenID string,
	incogAddressStr string,
	redeemAmount uint64,
	redeemProof string) (*PortalRequestUnlockCollateral, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	requestPTokenMeta := &PortalRequestUnlockCollateral{
		UniqueRedeemID:      uniqueRedeemID,
		TokenID:             tokenID,
		CustodianAddressStr: incogAddressStr,
		RedeemAmount:        redeemAmount,
		RedeemProof:         redeemProof,
	}
	requestPTokenMeta.MetadataBase = metadataBase
	return requestPTokenMeta, nil
}


func (meta PortalRequestUnlockCollateral) Hash() *common.Hash {
	record := meta.MetadataBase.Hash().String()
	record += meta.UniqueRedeemID
	record += meta.TokenID
	record += meta.CustodianAddressStr
	record += strconv.FormatUint(meta.RedeemAmount, 10)
	record += meta.RedeemProof
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}