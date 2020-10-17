package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

// PortalRedeemRequest - portal user redeem requests to get public token by burning ptoken
// metadata - redeem request - create normal tx with this metadata
type PortalExpiredWaitingPortingReq struct {
	MetadataBase
	UniquePortingID      string
	ExpiredByLiquidation bool
}

// PortalExpiredWaitingPortingReqContent - Beacon builds a new instruction with this content after detecting custodians run away
// It will be appended to beaconBlock
type PortalExpiredWaitingPortingReqContent struct {
	MetadataBase
	UniquePortingID      string
	ExpiredByLiquidation bool
	ShardID              byte
}

// PortalExpiredWaitingPortingReqStatus - Beacon tracks status of custodian liquidation into db
type PortalExpiredWaitingPortingReqStatus struct {
	Status               byte
	UniquePortingID      string
	ShardID              byte
	ExpiredByLiquidation bool
	ExpiredBeaconHeight  uint64
}

func NewPortalExpiredWaitingPortingReq(
	metaType int,
	uniquePortingID string,
	expiredByLiquidation bool,
) (*PortalExpiredWaitingPortingReq, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	liquidCustodianMeta := &PortalExpiredWaitingPortingReq{
		UniquePortingID:      uniquePortingID,
		ExpiredByLiquidation: expiredByLiquidation,
	}
	liquidCustodianMeta.MetadataBase = metadataBase
	return liquidCustodianMeta, nil
}

func (expiredPortingReq PortalExpiredWaitingPortingReq) Hash() *common.Hash {
	record := expiredPortingReq.MetadataBase.Hash().String()
	record += expiredPortingReq.UniquePortingID
	record += strconv.FormatBool(expiredPortingReq.ExpiredByLiquidation)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
