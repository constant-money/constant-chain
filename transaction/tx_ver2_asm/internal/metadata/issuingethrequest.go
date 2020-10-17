package metadata

import (
	"encoding/hex"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/pkg/errors"
)

type IssuingETHRequest struct {
	BlockHash  []byte
	TxIndex    uint
	ProofStrs  []string
	IncTokenID common.Hash
	MetadataBase
}

// type IssuingETHReqAction struct {
// 	Meta       IssuingETHRequest `json:"meta"`
// 	TxReqID    common.Hash       `json:"txReqId"`
// 	ETHReceipt *types.Receipt    `json:"ethReceipt"`
// }

// func ParseETHIssuingInstContent(instContentStr string) (*IssuingETHReqAction, error) {
// 	contentBytes, err := base64.StdEncoding.DecodeString(instContentStr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var issuingETHReqAction IssuingETHReqAction
// 	err = json.Unmarshal(contentBytes, &issuingETHReqAction)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &issuingETHReqAction, nil
// }

func NewIssuingETHRequest(
	blockHash []byte,
	txIndex uint,
	proofStrs []string,
	incTokenID common.Hash,
	metaType int,
) (*IssuingETHRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	issuingETHReq := &IssuingETHRequest{
		BlockHash:  blockHash,
		TxIndex:    txIndex,
		ProofStrs:  proofStrs,
		IncTokenID: incTokenID,
	}
	issuingETHReq.MetadataBase = metadataBase
	return issuingETHReq, nil
}

func NewIssuingETHRequestFromMap(
	data map[string]interface{},
) (*IssuingETHRequest, error) {
	blockHash := data["BlockHash"].([]byte)
	txIdx := uint(data["TxIndex"].(float64))
	proofsRaw := data["ProofStrs"].([]interface{})
	proofStrs := []string{}
	for _, item := range proofsRaw {
		proofStrs = append(proofStrs, item.(string))
	}

	incTokenID, err := common.Hash{}.NewHashFromStr(data["IncTokenID"].(string))
	if err != nil {
		return nil, errors.Errorf("TokenID incorrect")
	}

	req, _ := NewIssuingETHRequest(
		blockHash,
		txIdx,
		proofStrs,
		*incTokenID,
		IssuingETHRequestMeta,
	)
	return req, nil
}

func (iReq IssuingETHRequest) Hash() *common.Hash {
	record := hex.EncodeToString(iReq.BlockHash)
	record += string(iReq.TxIndex)
	proofStrs := iReq.ProofStrs
	for _, proofStr := range proofStrs {
		record += proofStr
	}
	record += iReq.MetadataBase.Hash().String()
	record += iReq.IncTokenID.String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}