package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

// PortalReqMatchingRedeem - portal custodian request matching redeem requests
// metadata - request matching redeem requests - create normal tx with this metadata
type PortalReqMatchingRedeem struct {
	MetadataBase
	CustodianAddressStr string
	RedeemID            string
}

// PortalReqMatchingRedeemAction - shard validator creates instruction that contain this action content
// it will be append to ShardToBeaconBlock
type PortalReqMatchingRedeemAction struct {
	Meta    PortalReqMatchingRedeem
	TxReqID common.Hash
	ShardID byte
}

// PortalReqMatchingRedeemContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalReqMatchingRedeemContent struct {
	CustodianAddressStr string
	RedeemID            string
	MatchingAmount      uint64
	IsFullCustodian     bool
	TxReqID             common.Hash
	ShardID             byte
}

// PortalReqMatchingRedeemStatus - Beacon tracks status of request matching redeem tx into db
type PortalReqMatchingRedeemStatus struct {
	CustodianAddressStr string
	RedeemID            string
	MatchingAmount      uint64
	Status              byte
}

func NewPortalReqMatchingRedeem(metaType int, custodianAddrStr string, redeemID string) (*PortalReqMatchingRedeem, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}
	custodianDepositMeta := &PortalReqMatchingRedeem{
		CustodianAddressStr: custodianAddrStr,
		RedeemID:            redeemID,
	}
	custodianDepositMeta.MetadataBase = metadataBase
	return custodianDepositMeta, nil
}

func (*PortalReqMatchingRedeem) ShouldSignMetaData() bool { return true }


func (req PortalReqMatchingRedeem) Hash() *common.Hash {
	record := req.MetadataBase.Hash().String()
	record += req.CustodianAddressStr
	record += req.RedeemID
	if req.Sig != nil && len(req.Sig) != 0 {
		record += string(req.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (req PortalReqMatchingRedeem) HashWithoutSig() *common.Hash {
	record := req.MetadataBase.Hash().String()
	record += req.CustodianAddressStr
	record += req.RedeemID

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}