package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
)

// whoever can send this type of tx
type ContractingRequest struct {
	BurnerAddress privacy.PaymentAddress
	BurnedAmount  uint64 // must be equal to vout value
	TokenID       common.Hash
	MetadataBase
}

type ContractingReqAction struct {
	Meta    ContractingRequest `json:"meta"`
	TxReqID common.Hash        `json:"txReqId"`
}

func NewContractingRequest(
	burnerAddress privacy.PaymentAddress,
	burnedAmount uint64,
	tokenID common.Hash,
	metaType int,
) (*ContractingRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	contractingReq := &ContractingRequest{
		TokenID:       tokenID,
		BurnedAmount:  burnedAmount,
		BurnerAddress: burnerAddress,
	}
	contractingReq.MetadataBase = metadataBase
	return contractingReq, nil
}

func (cReq ContractingRequest) Hash() *common.Hash {
	record := cReq.MetadataBase.Hash().String()
	record += cReq.BurnerAddress.String()
	record += cReq.TokenID.String()
	record += string(cReq.BurnedAmount)

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}