package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

// PortalRequestPTokens - portal user requests ptoken (after sending pubToken to custodians)
// metadata - user requests ptoken - create normal tx with this metadata
type PortalRequestPTokens struct {
	MetadataBase
	UniquePortingID string
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	PortingAmount   uint64
	PortingProof    string
}

// PortalRequestPTokensAction - shard validator creates instruction that contain this action content
// it will be append to ShardToBeaconBlock
type PortalRequestPTokensAction struct {
	Meta    PortalRequestPTokens
	TxReqID common.Hash
	ShardID byte
}

// PortalRequestPTokensContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalRequestPTokensContent struct {
	UniquePortingID string
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	PortingAmount   uint64
	PortingProof    string
	TxReqID         common.Hash
	ShardID         byte
}

// PortalRequestPTokensStatus - Beacon tracks status of request ptokens into db
type PortalRequestPTokensStatus struct {
	Status          byte
	UniquePortingID string
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	PortingAmount   uint64
	PortingProof    string
	TxReqID         common.Hash
}

func NewPortalRequestPTokens(
	metaType int,
	uniquePortingID string,
	tokenID string,
	incogAddressStr string,
	portingAmount uint64,
	portingProof string) (*PortalRequestPTokens, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	requestPTokenMeta := &PortalRequestPTokens{
		UniquePortingID: uniquePortingID,
		TokenID:         tokenID,
		IncogAddressStr: incogAddressStr,
		PortingAmount:   portingAmount,
		PortingProof:    portingProof,
	}
	requestPTokenMeta.MetadataBase = metadataBase
	return requestPTokenMeta, nil
}

func (reqPToken PortalRequestPTokens) Hash() *common.Hash {
	record := reqPToken.MetadataBase.Hash().String()
	record += reqPToken.UniquePortingID
	record += reqPToken.TokenID
	record += reqPToken.IncogAddressStr
	record += strconv.FormatUint(reqPToken.PortingAmount, 10)
	record += reqPToken.PortingProof
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}