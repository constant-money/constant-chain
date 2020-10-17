package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"strconv"
)

type PortalCustodianWithdrawRequest struct {
	MetadataBase
	PaymentAddress string
	Amount         uint64
}

type PortalCustodianWithdrawRequestAction struct {
	Meta    PortalCustodianWithdrawRequest
	TxReqID common.Hash
	ShardID byte
}

type PortalCustodianWithdrawRequestContent struct {
	PaymentAddress       string
	Amount               uint64
	RemainFreeCollateral uint64
	TxReqID              common.Hash
	ShardID              byte
}

type CustodianWithdrawRequestStatus struct {
	PaymentAddress                string
	Amount                        uint64
	Status                        int
	RemainCustodianFreeCollateral uint64
}

func NewCustodianWithdrawRequestStatus(paymentAddress string, amount uint64, status int, remainCustodianFreeCollateral uint64) *CustodianWithdrawRequestStatus {
	return &CustodianWithdrawRequestStatus{PaymentAddress: paymentAddress, Amount: amount, Status: status, RemainCustodianFreeCollateral: remainCustodianFreeCollateral}
}

func NewPortalCustodianWithdrawRequest(metaType int, paymentAddress string, amount uint64) (*PortalCustodianWithdrawRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}

	portalCustodianWithdrawReq := &PortalCustodianWithdrawRequest{
		PaymentAddress: paymentAddress,
		Amount:         amount,
	}

	portalCustodianWithdrawReq.MetadataBase = metadataBase

	return portalCustodianWithdrawReq, nil
}

func (*PortalCustodianWithdrawRequest) ShouldSignMetaData() bool { return true }


func (Withdraw PortalCustodianWithdrawRequest) Hash() *common.Hash {
	record := Withdraw.MetadataBase.Hash().String()
	record += Withdraw.PaymentAddress
	record += strconv.FormatUint(Withdraw.Amount, 10)
	if Withdraw.Sig != nil && len(Withdraw.Sig) != 0 {
		record += string(Withdraw.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (Withdraw PortalCustodianWithdrawRequest) HashWithoutSig() *common.Hash {
	record := Withdraw.MetadataBase.Hash().String()
	record += Withdraw.PaymentAddress
	record += strconv.FormatUint(Withdraw.Amount, 10)

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}