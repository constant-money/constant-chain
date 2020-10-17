package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

type IssuingETHResponse struct {
	MetadataBase
	RequestedTxID   common.Hash
	UniqETHTx       []byte
	ExternalTokenID []byte
	SharedRandom       []byte
}

type IssuingETHResAction struct {
	Meta       *IssuingETHResponse `json:"meta"`
	IncTokenID *common.Hash        `json:"incTokenID"`
}

func NewIssuingETHResponse(
	requestedTxID common.Hash,
	uniqETHTx []byte,
	externalTokenID []byte,
	metaType int,
) *IssuingETHResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &IssuingETHResponse{
		RequestedTxID:   requestedTxID,
		UniqETHTx:       uniqETHTx,
		ExternalTokenID: externalTokenID,
		MetadataBase:    metadataBase,
	}
}

func (iRes IssuingETHResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += string(iRes.UniqETHTx)
	record += string(iRes.ExternalTokenID)
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *IssuingETHResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}