package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
	"strconv"
)

type WithDrawRewardRequest struct {
	MetadataBase
	PaymentAddress privacy.PaymentAddress
	TokenID common.Hash
	Version int
}

func (withDrawRewardRequest WithDrawRewardRequest) Hash() *common.Hash {
	if withDrawRewardRequest.Version == 1 {
		bArr := append(withDrawRewardRequest.PaymentAddress.Bytes(), withDrawRewardRequest.TokenID.GetBytes()...)
		if withDrawRewardRequest.Sig != nil && len(withDrawRewardRequest.Sig) != 0 {
			bArr = append(bArr, withDrawRewardRequest.Sig...)
		}
		txReqHash := common.HashH(bArr)
		return &txReqHash
	} else {
		record := strconv.Itoa(withDrawRewardRequest.Type)
		data := []byte(record)
		hash := common.HashH(data)
		return &hash
	}
}

func (withDrawRewardRequest WithDrawRewardRequest) HashWithoutSig() *common.Hash {
	if withDrawRewardRequest.Version == 1 {
		bArr := append(withDrawRewardRequest.PaymentAddress.Bytes(), withDrawRewardRequest.TokenID.GetBytes()...)
		txReqHash := common.HashH(bArr)
		return &txReqHash
	} else {
		record := strconv.Itoa(withDrawRewardRequest.Type)
		data := []byte(record)
		hash := common.HashH(data)
		return &hash
	}
}

func (*WithDrawRewardRequest) ShouldSignMetaData() bool { return true }

func NewWithDrawRewardRequest(tokenIDStr string, paymentAddStr string, version float64, metaType int) (*WithDrawRewardRequest, error) {
	metadataBase := NewMetadataBase(metaType)
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, errors.New("token ID is invalid")
	}
	paymentAddWallet, err := wallet.Base58CheckDeserialize(paymentAddStr)
	if err != nil {
		return nil, errors.New("payment address is invalid")
	}
	ok, err := common.SliceExists(AcceptedWithdrawRewardRequestVersion, int(version));
	if !ok || err != nil {
		return nil, errors.Errorf("Invalid version %d", version)
	}

	withdrawRewardRequest := &WithDrawRewardRequest{
		MetadataBase: *metadataBase,
		TokenID:  *tokenID,
		PaymentAddress: paymentAddWallet.KeySet.PaymentAddress,
		Version: int(version),
	}
	return withdrawRewardRequest, nil
}

func (withDrawRewardRequest WithDrawRewardRequest) GetType() int {
	return withDrawRewardRequest.Type
}
