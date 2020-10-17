package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/pkg/errors"
	"strconv"
)

type WithDrawRewardResponse struct {
	MetadataBase
	TxRequest       *common.Hash
	TokenID         common.Hash
	RewardPublicKey []byte
	SharedRandom    []byte
	Version         int
}

func NewWithDrawRewardResponse(metaRequest *WithDrawRewardRequest, reqID *common.Hash) (*WithDrawRewardResponse, error) {
	metadataBase := MetadataBase{
		Type: WithDrawRewardResponseMeta,
	}
	result := &WithDrawRewardResponse{
		MetadataBase:    metadataBase,
		TxRequest:       reqID,
		TokenID:         metaRequest.TokenID,
		RewardPublicKey: metaRequest.PaymentAddress.Pk[:],
	}
	result.Version = metaRequest.Version
	if ok, err := common.SliceExists(AcceptedWithdrawRewardRequestVersion, result.Version); !ok || err != nil {
		return nil, errors.Errorf("Invalid version %d", result.Version)
	}
	return result, nil
}

func (withDrawRewardResponse WithDrawRewardResponse) Hash() *common.Hash {
	if withDrawRewardResponse.Version == 1 {
		if withDrawRewardResponse.TxRequest == nil {
			return &common.Hash{}
		}
		bArr := append(withDrawRewardResponse.TxRequest.GetBytes(), withDrawRewardResponse.TokenID.GetBytes()...)
		version := strconv.Itoa(withDrawRewardResponse.Version)
		if len(withDrawRewardResponse.SharedRandom) != 0 {
			bArr = append(bArr, withDrawRewardResponse.SharedRandom...)
		}
		if len(withDrawRewardResponse.RewardPublicKey) != 0 {
			bArr = append(bArr, withDrawRewardResponse.RewardPublicKey...)
		}

		bArr = append(bArr, []byte(version)...)
		txResHash := common.HashH(bArr)
		return &txResHash
	} else {
		return withDrawRewardResponse.TxRequest
	}
}

func (withDrawRewardResponse *WithDrawRewardResponse) SetSharedRandom(r []byte) {
	withDrawRewardResponse.SharedRandom = r
}
