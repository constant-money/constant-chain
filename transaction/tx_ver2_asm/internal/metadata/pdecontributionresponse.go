package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

type PDEContributionResponse struct {
	MetadataBase
	ContributionStatus string
	RequestedTxID      common.Hash
	TokenIDStr         string
	SharedRandom       []byte
}

func NewPDEContributionResponse(
	contributionStatus string,
	requestedTxID common.Hash,
	tokenIDStr string,
	metaType int,
) *PDEContributionResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDEContributionResponse{
		ContributionStatus: contributionStatus,
		RequestedTxID:      requestedTxID,
		TokenIDStr:         tokenIDStr,
		MetadataBase:       metadataBase,
	}
}

func (iRes PDEContributionResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.TokenIDStr
	record += iRes.ContributionStatus
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PDEContributionResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}
