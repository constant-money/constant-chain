package metadata

import (
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

// PortalUserRegister - User register porting public tokens
type PortalUserRegister struct {
	MetadataBase
	UniqueRegisterId string //
	IncogAddressStr  string
	PTokenId         string
	RegisterAmount   uint64
	PortingFee       uint64
}

type PortalUserRegisterAction struct {
	Meta    PortalUserRegister
	TxReqID common.Hash
	ShardID byte
}

type MatchingPortingCustodianDetail struct {
	IncAddress             string
	RemoteAddress          string
	Amount                 uint64
	LockedAmountCollateral uint64
}

type PortalPortingRequestContent struct {
	UniqueRegisterId string
	IncogAddressStr  string
	PTokenId         string
	RegisterAmount   uint64
	PortingFee       uint64
	Custodian        []*MatchingPortingCustodianDetail
	TxReqID          common.Hash
	ShardID          byte
}

type PortingRequestStatus struct {
	UniquePortingID string
	TxReqID         common.Hash
	TokenID         string
	PorterAddress   string
	Amount          uint64
	Custodians      []*MatchingPortingCustodianDetail
	PortingFee      uint64
	Status          int
	BeaconHeight    uint64
}

func NewPortingRequestStatus(
	uniquePortingID string,
	txReqID common.Hash,
	tokenID string,
	porterAddress string,
	amount uint64,
	custodians []*MatchingPortingCustodianDetail,
	portingFee uint64,
	status int,
	beaconHeight uint64) *PortingRequestStatus {
	return &PortingRequestStatus{UniquePortingID: uniquePortingID, TxReqID: txReqID, TokenID: tokenID, PorterAddress: porterAddress, Amount: amount, Custodians: custodians, PortingFee: portingFee, Status: status, BeaconHeight: beaconHeight}
}

func NewPortalUserRegister(uniqueRegisterId string, incogAddressStr string, pTokenId string, registerAmount uint64, portingFee uint64, metaType int) (*PortalUserRegister, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}

	portalUserRegisterMeta := &PortalUserRegister{
		UniqueRegisterId: uniqueRegisterId,
		IncogAddressStr:  incogAddressStr,
		PTokenId:         pTokenId,
		RegisterAmount:   registerAmount,
		PortingFee:       portingFee,
	}

	portalUserRegisterMeta.MetadataBase = metadataBase

	return portalUserRegisterMeta, nil
}

func (portalUserRegister PortalUserRegister) Hash() *common.Hash {
	record := portalUserRegister.MetadataBase.Hash().String()
	record += portalUserRegister.UniqueRegisterId
	record += portalUserRegister.PTokenId
	record += portalUserRegister.IncogAddressStr
	record += strconv.FormatUint(portalUserRegister.RegisterAmount, 10)
	record += strconv.FormatUint(portalUserRegister.PortingFee, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}