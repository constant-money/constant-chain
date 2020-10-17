package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalLiquidationCustodianDepositV2 struct {
	MetadataBase
	IncogAddressStr      string
	PTokenId             string
	DepositedAmount      uint64
	FreeCollateralAmount uint64
}

type PortalLiquidationCustodianDepositActionV2 struct {
	Meta    PortalLiquidationCustodianDepositV2
	TxReqID common.Hash
	ShardID byte
}

type PortalLiquidationCustodianDepositContentV2 struct {
	IncogAddressStr      string
	PTokenId             string
	DepositedAmount      uint64
	FreeCollateralAmount uint64
	TxReqID              common.Hash
	ShardID              byte
}

type LiquidationCustodianDepositStatusV2 struct {
	TxReqID              common.Hash
	IncogAddressStr      string
	PTokenId             string
	DepositAmount        uint64
	FreeCollateralAmount uint64
	Status               byte
}

func NewLiquidationCustodianDepositStatusV2(txReqID common.Hash, incogAddressStr string, PTokenId string, depositAmount uint64, freeCollateralAmount uint64, status byte) *LiquidationCustodianDepositStatusV2 {
	return &LiquidationCustodianDepositStatusV2{TxReqID: txReqID, IncogAddressStr: incogAddressStr, PTokenId: PTokenId, DepositAmount: depositAmount, FreeCollateralAmount: freeCollateralAmount, Status: status}
}

func NewPortalLiquidationCustodianDepositV2(metaType int, incognitoAddrStr string, pToken string, amount uint64, freeCollateralAmount uint64) (*PortalLiquidationCustodianDepositV2, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	custodianDepositMeta := &PortalLiquidationCustodianDepositV2{
		IncogAddressStr:      incognitoAddrStr,
		PTokenId:             pToken,
		DepositedAmount:      amount,
		FreeCollateralAmount: freeCollateralAmount,
	}
	custodianDepositMeta.MetadataBase = metadataBase
	return custodianDepositMeta, nil
}

func (custodianDeposit PortalLiquidationCustodianDepositV2) Hash() *common.Hash {
	record := custodianDeposit.MetadataBase.Hash().String()
	record += custodianDeposit.IncogAddressStr
	record += custodianDeposit.PTokenId
	record += strconv.FormatUint(custodianDeposit.DepositedAmount, 10)
	record += strconv.FormatUint(custodianDeposit.FreeCollateralAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}