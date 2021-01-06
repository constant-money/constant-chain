package metadata

import (

	"github.com/incognitochain/incognito-chain/common"
)

type PDEFeeWithdrawalResponse struct {
	MetadataBase
	RequestedTxID common.Hash
	SharedRandom       []byte
}

func NewPDEFeeWithdrawalResponse(
	requestedTxID common.Hash,
	metaType int,
) *PDEFeeWithdrawalResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDEFeeWithdrawalResponse{
		RequestedTxID: requestedTxID,
		MetadataBase:  metadataBase,
	}
}

func (iRes PDEFeeWithdrawalResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PDEFeeWithdrawalResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}