package metadata

import (
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

// PDEContribution - privacy dex contribution
type PDEContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64 // must be equal to vout value
	TokenIDStr            string
	MetadataBase
}

type PDEContributionAction struct {
	Meta    PDEContribution
	TxReqID common.Hash
	ShardID byte
}

type PDEWaitingContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
}

type PDERefundContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
	ShardID               byte
}

type PDEMatchedContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
}

type PDEMatchedNReturnedContribution struct {
	PDEContributionPairID      string
	ContributorAddressStr      string
	ActualContributedAmount    uint64
	ReturnedContributedAmount  uint64
	TokenIDStr                 string
	ShardID                    byte
	TxReqID                    common.Hash
	ActualWaitingContribAmount uint64
}

type PDEContributionStatus struct {
	Status             byte
	TokenID1Str        string
	Contributed1Amount uint64
	Returned1Amount    uint64
	TokenID2Str        string
	Contributed2Amount uint64
	Returned2Amount    uint64
}

func NewPDEContribution(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
) (*PDEContribution, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	pdeContribution := &PDEContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
	}
	pdeContribution.MetadataBase = metadataBase
	return pdeContribution, nil
}

func (pc PDEContribution) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.PDEContributionPairID
	record += pc.ContributorAddressStr
	record += pc.TokenIDStr
	record += strconv.FormatUint(pc.ContributedAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}