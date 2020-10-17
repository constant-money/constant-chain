package metadata

import (
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

// PortalRedeemRequest - portal user redeem requests to get public token by burning ptoken
// metadata - redeem request - create normal tx with this metadata
type PortalRedeemRequest struct {
	MetadataBase
	UniqueRedeemID        string
	TokenID               string // pTokenID in incognito chain
	RedeemAmount          uint64
	RedeemerIncAddressStr string
	RemoteAddress         string // btc/bnb/etc address
	RedeemFee             uint64 // redeem fee in PRV, 0.01% redeemAmount in PRV
}

// PortalRedeemRequestAction - shard validator creates instruction that contain this action content
// it will be append to ShardToBeaconBlock
type PortalRedeemRequestAction struct {
	Meta    PortalRedeemRequest
	TxReqID common.Hash
	ShardID byte
}

type MatchingRedeemCustodianDetail struct {
	incAddress    string
	remoteAddress string
	amount        uint64
}

// PortalRedeemRequestContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalRedeemRequestContent struct {
	UniqueRedeemID          string
	TokenID                 string // pTokenID in incognito chain
	RedeemAmount            uint64
	RedeemerIncAddressStr   string
	RemoteAddress           string                                   // btc/bnb/etc address
	RedeemFee               uint64                                   // redeem fee in PRV, 0.01% redeemAmount in PRV
	MatchingCustodianDetail []*MatchingRedeemCustodianDetail // key: incAddressCustodian
	TxReqID                 common.Hash
	ShardID                 byte
}

// PortalRedeemRequestStatus - Beacon tracks status of redeem request into db
type PortalRedeemRequestStatus struct {
	Status                  byte
	UniqueRedeemID          string
	TokenID                 string // pTokenID in incognito chain
	RedeemAmount            uint64
	RedeemerIncAddressStr   string
	RemoteAddress           string                                   // btc/bnb/etc address
	RedeemFee               uint64                                   // redeem fee in PRV, 0.01% redeemAmount in PRV
	MatchingCustodianDetail []*MatchingRedeemCustodianDetail // key: incAddressCustodian
	TxReqID                 common.Hash
}

func NewPortalRedeemRequest(
	metaType int,
	uniqueRedeemID string,
	tokenID string,
	redeemAmount uint64,
	incAddressStr string,
	remoteAddr string,
	redeemFee uint64) (*PortalRedeemRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	requestPTokenMeta := &PortalRedeemRequest{
		UniqueRedeemID:        uniqueRedeemID,
		TokenID:               tokenID,
		RedeemAmount:          redeemAmount,
		RedeemerIncAddressStr: incAddressStr,
		RemoteAddress:         remoteAddr,
		RedeemFee:             redeemFee,
	}
	requestPTokenMeta.MetadataBase = metadataBase
	return requestPTokenMeta, nil
}

func (redeemReq PortalRedeemRequest) Hash() *common.Hash {
	record := redeemReq.MetadataBase.Hash().String()
	record += redeemReq.UniqueRedeemID
	record += redeemReq.TokenID
	record += strconv.FormatUint(redeemReq.RedeemAmount, 10)
	record += strconv.FormatUint(redeemReq.RedeemFee, 10)
	record += redeemReq.RedeemerIncAddressStr
	record += redeemReq.RemoteAddress
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}