package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalLiquidationCustodianDeposit struct {
	MetadataBase
	IncogAddressStr        string
	PTokenId               string
	DepositedAmount        uint64
	FreeCollateralSelected bool
}

type PortalLiquidationCustodianDepositAction struct {
	Meta    PortalLiquidationCustodianDeposit
	TxReqID common.Hash
	ShardID byte
}

type PortalLiquidationCustodianDepositContent struct {
	IncogAddressStr        string
	PTokenId               string
	DepositedAmount        uint64
	FreeCollateralSelected bool
	TxReqID                common.Hash
	ShardID                byte
}

type LiquidationCustodianDepositStatus struct {
	TxReqID                common.Hash
	IncogAddressStr        string
	PTokenId               string
	DepositAmount          uint64
	FreeCollateralSelected bool
	Status                 byte
}

func NewLiquidationCustodianDepositStatus(txReqID common.Hash, incogAddressStr string, PTokenId string, depositAmount uint64, freeCollateralSelected bool, status byte) *LiquidationCustodianDepositStatus {
	return &LiquidationCustodianDepositStatus{TxReqID: txReqID, IncogAddressStr: incogAddressStr, PTokenId: PTokenId, DepositAmount: depositAmount, FreeCollateralSelected: freeCollateralSelected, Status: status}
}

func NewPortalLiquidationCustodianDeposit(metaType int, incognitoAddrStr string, pToken string, amount uint64, freeCollateralSelected bool) (*PortalLiquidationCustodianDeposit, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	custodianDepositMeta := &PortalLiquidationCustodianDeposit{
		IncogAddressStr:        incognitoAddrStr,
		PTokenId:               pToken,
		DepositedAmount:        amount,
		FreeCollateralSelected: freeCollateralSelected,
	}
	custodianDepositMeta.MetadataBase = metadataBase
	return custodianDepositMeta, nil
}
func (custodianDeposit PortalLiquidationCustodianDeposit) Hash() *common.Hash {
	record := custodianDeposit.MetadataBase.Hash().String()
	record += custodianDeposit.IncogAddressStr
	record += custodianDeposit.PTokenId
	record += strconv.FormatUint(custodianDeposit.DepositedAmount, 10)
	record += strconv.FormatBool(custodianDeposit.FreeCollateralSelected)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}