package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

type PortalCustodianDepositResponse struct {
	MetadataBase
	DepositStatus    string
	ReqTxID          common.Hash
	CustodianAddrStr string
	SharedRandom       []byte
}

func NewPortalCustodianDepositResponse(
	depositStatus string,
	reqTxID common.Hash,
	custodianAddressStr string,
	metaType int,
) *PortalCustodianDepositResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PortalCustodianDepositResponse{
		DepositStatus:    depositStatus,
		ReqTxID:          reqTxID,
		MetadataBase:     metadataBase,
		CustodianAddrStr: custodianAddressStr,
	}
}
func (iRes PortalCustodianDepositResponse) Hash() *common.Hash {
	record := iRes.DepositStatus
	record += iRes.ReqTxID.String()
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PortalCustodianDepositResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}