package metadata

import (
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

type PortalTopUpWaitingPortingRequest struct {
	MetadataBase
	IncogAddressStr      string
	PortingID            string
	PTokenID             string
	DepositedAmount      uint64
	FreeCollateralAmount uint64
}

type PortalTopUpWaitingPortingRequestAction struct {
	Meta    PortalTopUpWaitingPortingRequest
	TxReqID common.Hash
	ShardID byte
}

type PortalTopUpWaitingPortingRequestContent struct {
	IncogAddressStr      string
	PortingID            string
	PTokenID             string
	DepositedAmount      uint64
	FreeCollateralAmount uint64
	TxReqID              common.Hash
	ShardID              byte
}

type PortalTopUpWaitingPortingRequestStatus struct {
	TxReqID              common.Hash
	IncogAddressStr      string
	PortingID            string
	PTokenID             string
	DepositAmount        uint64
	FreeCollateralAmount uint64
	Status               byte
}

func NewPortalTopUpWaitingPortingRequestStatus(
	txReqID common.Hash,
	portingID string,
	incogAddressStr string,
	pTokenID string,
	depositAmount uint64,
	freeCollateralAmount uint64,
	status byte,
) *PortalTopUpWaitingPortingRequestStatus {
	return &PortalTopUpWaitingPortingRequestStatus{
		TxReqID:              txReqID,
		PortingID:            portingID,
		IncogAddressStr:      incogAddressStr,
		PTokenID:             pTokenID,
		DepositAmount:        depositAmount,
		FreeCollateralAmount: freeCollateralAmount,
		Status:               status,
	}
}

func NewPortalTopUpWaitingPortingRequest(
	metaType int,
	portingID string,
	incogAddressStr string,
	pToken string,
	amount uint64,
	freeCollateralAmount uint64,
) (*PortalTopUpWaitingPortingRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	portalTopUpWaitingPortingRequestMeta := &PortalTopUpWaitingPortingRequest{
		PortingID:            portingID,
		IncogAddressStr:      incogAddressStr,
		PTokenID:             pToken,
		DepositedAmount:      amount,
		FreeCollateralAmount: freeCollateralAmount,
	}
	portalTopUpWaitingPortingRequestMeta.MetadataBase = metadataBase
	return portalTopUpWaitingPortingRequestMeta, nil
}

func (p PortalTopUpWaitingPortingRequest) Hash() *common.Hash {
	record := p.MetadataBase.Hash().String()
	record += p.PortingID
	record += p.IncogAddressStr
	record += p.PTokenID
	record += strconv.FormatUint(p.DepositedAmount, 10)
	record += strconv.FormatUint(p.FreeCollateralAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
