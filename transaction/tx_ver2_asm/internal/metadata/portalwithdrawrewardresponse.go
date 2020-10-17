package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

// PortalRequestUnlockCollateral - portal custodian requests unlock collateral (after returning pubToken to user)
// metadata - custodian requests unlock collateral - create normal tx with this metadata
type PortalWithdrawRewardResponse struct {
	MetadataBase
	CustodianAddressStr string
	TokenID             common.Hash
	RewardAmount        uint64
	TxReqID             common.Hash
	SharedRandom       []byte
}

func NewPortalWithdrawRewardResponse(
	reqTxID common.Hash,
	custodianAddressStr string,
	tokenID common.Hash,
	rewardAmount uint64,
	metaType int,
) *PortalWithdrawRewardResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PortalWithdrawRewardResponse{
		MetadataBase:        metadataBase,
		CustodianAddressStr: custodianAddressStr,
		TokenID:             tokenID,
		RewardAmount:        rewardAmount,
		TxReqID:             reqTxID,
	}
}

func (iRes PortalWithdrawRewardResponse) Hash() *common.Hash {
	record := iRes.MetadataBase.Hash().String()
	record += iRes.TxReqID.String()
	record += iRes.CustodianAddressStr
	record += iRes.TokenID.String()
	record += strconv.FormatUint(iRes.RewardAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PortalWithdrawRewardResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}