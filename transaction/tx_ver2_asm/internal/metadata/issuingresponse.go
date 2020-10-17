package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

type IssuingResponse struct {
	MetadataBase
	RequestedTxID common.Hash
	SharedRandom       []byte
}

type IssuingResAction struct {
	IncTokenID *common.Hash `json:"incTokenID"`
}

func NewIssuingResponse(requestedTxID common.Hash, metaType int) *IssuingResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &IssuingResponse{
		RequestedTxID: requestedTxID,
		MetadataBase:  metadataBase,
	}
}

func (iRes IssuingResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *IssuingResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}