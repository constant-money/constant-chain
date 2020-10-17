   package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalLiquidationCustodianDepositResponseV2 struct {
	MetadataBase
	DepositStatus    string
	ReqTxID          common.Hash
	CustodianAddrStr string
	DepositedAmount  uint64
	SharedRandom       []byte
}

func NewPortalLiquidationCustodianDepositResponseV2(
	depositStatus string,
	reqTxID common.Hash,
	custodianAddressStr string,
	depositedAmount uint64,
	metaType int,
) *PortalLiquidationCustodianDepositResponseV2 {
	metadataBase := MetadataBase{
		Type: metaType,
	}

	return &PortalLiquidationCustodianDepositResponseV2{
		DepositStatus:    depositStatus,
		ReqTxID:          reqTxID,
		MetadataBase:     metadataBase,
		CustodianAddrStr: custodianAddressStr,
		DepositedAmount:  depositedAmount,
	}
}

func (iRes PortalLiquidationCustodianDepositResponseV2) Hash() *common.Hash {
	record := iRes.DepositStatus
	record += strconv.FormatUint(iRes.DepositedAmount, 10)
	record += iRes.ReqTxID.String()
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}


func (iRes *PortalLiquidationCustodianDepositResponseV2) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}