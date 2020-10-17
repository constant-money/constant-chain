package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalLiquidationCustodianDepositResponse struct {
	MetadataBase
	DepositStatus    string
	ReqTxID          common.Hash
	CustodianAddrStr string
	DepositedAmount  uint64
	SharedRandom       []byte
}

func NewPortalLiquidationCustodianDepositResponse(
	depositStatus string,
	reqTxID common.Hash,
	custodianAddressStr string,
	depositedAmount uint64,
	metaType int,
) *PortalLiquidationCustodianDepositResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}

	return &PortalLiquidationCustodianDepositResponse{
		DepositStatus:    depositStatus,
		ReqTxID:          reqTxID,
		MetadataBase:     metadataBase,
		CustodianAddrStr: custodianAddressStr,
		DepositedAmount:  depositedAmount,
	}
}

func (iRes PortalLiquidationCustodianDepositResponse) Hash() *common.Hash {
	record := iRes.DepositStatus
	record += strconv.FormatUint(iRes.DepositedAmount, 10)
	record += iRes.ReqTxID.String()
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PortalLiquidationCustodianDepositResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}