package metadata

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/common"
)

type PortalFeeRefundResponse struct {
	MetadataBase
	PortingRequestStatus string
	ReqTxID              common.Hash
	SharedRandom       []byte
}

func NewPortalFeeRefundResponse(
	portingRequestStatus string,
	reqTxID common.Hash,
	metaType int,
) *PortalFeeRefundResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PortalFeeRefundResponse{
		PortingRequestStatus: portingRequestStatus,
		ReqTxID:              reqTxID,
		MetadataBase:         metadataBase,
	}
}

func (iRes PortalFeeRefundResponse) Hash() *common.Hash {
	record := iRes.PortingRequestStatus
	record += iRes.ReqTxID.String()
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
func parsePortingRequest(contentBytes []byte, shardID string) (string, common.Hash, string, uint64, error) {
	var portalPortingRequestContent PortalPortingRequestContent
	err := json.Unmarshal(contentBytes, &portalPortingRequestContent)
	if err != nil {
		return "", common.Hash{}, "", uint64(0), err
	}
	return shardID, portalPortingRequestContent.TxReqID, portalPortingRequestContent.IncogAddressStr, portalPortingRequestContent.PortingFee, nil
}

func parseValuesFromInst(inst []string) (string, common.Hash, string, uint64, error) {
	shardIDStr := inst[1]
	contentBytes := []byte(inst[3])
	return parsePortingRequest(contentBytes, shardIDStr)
}

func (iRes *PortalFeeRefundResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}