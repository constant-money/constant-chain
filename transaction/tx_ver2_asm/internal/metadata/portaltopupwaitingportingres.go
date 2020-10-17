package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

type PortalTopUpWaitingPortingResponse struct {
	MetadataBase
	DepositStatus string
	ReqTxID       common.Hash
	SharedRandom       []byte
}

func NewPortalTopUpWaitingPortingResponse(
	depositStatus string,
	reqTxID common.Hash,
	metaType int,
) *PortalTopUpWaitingPortingResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}

	return &PortalTopUpWaitingPortingResponse{
		DepositStatus: depositStatus,
		ReqTxID:       reqTxID,
		MetadataBase:  metadataBase,
	}
}

func (iRes PortalTopUpWaitingPortingResponse) Hash() *common.Hash {
	record := iRes.DepositStatus
	record += iRes.ReqTxID.String()
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PortalTopUpWaitingPortingResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}