package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

type PDEWithdrawalResponse struct {
	MetadataBase
	RequestedTxID common.Hash
	TokenIDStr    string
	SharedRandom       []byte
}

func NewPDEWithdrawalResponse(
	tokenIDStr string,
	requestedTxID common.Hash,
	metaType int,
) *PDEWithdrawalResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDEWithdrawalResponse{
		RequestedTxID: requestedTxID,
		TokenIDStr:    tokenIDStr,
		MetadataBase:  metadataBase,
	}
}
func (iRes PDEWithdrawalResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.TokenIDStr
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PDEWithdrawalResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}