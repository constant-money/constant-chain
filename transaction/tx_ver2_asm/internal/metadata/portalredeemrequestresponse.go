package metadata

import (
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

type PortalRedeemRequestResponse struct {
	MetadataBase
	RequestStatus    string
	ReqTxID          common.Hash
	RequesterAddrStr string
	Amount           uint64
	IncTokenID       string
	SharedRandom       []byte
}

func NewPortalRedeemRequestResponse(
	requestStatus string,
	reqTxID common.Hash,
	requesterAddressStr string,
	amount uint64,
	tokenID string,
	metaType int,
) *PortalRedeemRequestResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PortalRedeemRequestResponse{
		RequestStatus:    requestStatus,
		ReqTxID:          reqTxID,
		MetadataBase:     metadataBase,
		RequesterAddrStr: requesterAddressStr,
		Amount:           amount,
		IncTokenID:       tokenID,
	}
}

func (iRes PortalRedeemRequestResponse) Hash() *common.Hash {
	record := iRes.MetadataBase.Hash().String()
	record += iRes.RequestStatus
	record += iRes.ReqTxID.String()
	record += iRes.RequesterAddrStr
	record += strconv.FormatUint(iRes.Amount, 10)
	record += iRes.IncTokenID
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PortalRedeemRequestResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}