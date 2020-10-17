package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalCustodianWithdrawResponse struct {
	MetadataBase
	RequestStatus  string
	ReqTxID        common.Hash
	PaymentAddress string
	Amount         uint64
	SharedRandom       []byte
}

func NewPortalCustodianWithdrawResponse(
	requestStatus string,
	reqTxId common.Hash,
	paymentAddress string,
	amount uint64,
	metaType int,
) *PortalCustodianWithdrawResponse {
	metaDataBase := MetadataBase{Type: metaType}

	return &PortalCustodianWithdrawResponse{
		MetadataBase:   metaDataBase,
		RequestStatus:  requestStatus,
		ReqTxID:        reqTxId,
		PaymentAddress: paymentAddress,
		Amount:         amount,
	}
}


func (responseMeta PortalCustodianWithdrawResponse) Hash() *common.Hash {
	record := responseMeta.MetadataBase.Hash().String()
	record += responseMeta.RequestStatus
	record += responseMeta.ReqTxID.String()
	record += responseMeta.PaymentAddress
	record += strconv.FormatUint(responseMeta.Amount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (responseMeta *PortalCustodianWithdrawResponse) SetSharedRandom(r []byte) {
	responseMeta.SharedRandom = r
}