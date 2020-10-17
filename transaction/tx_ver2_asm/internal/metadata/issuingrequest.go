package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

// only centralized website can send this type of tx
type IssuingRequest struct {
	ReceiverAddress privacy.PaymentAddress
	DepositedAmount uint64
	TokenID         common.Hash
	TokenName       string
	MetadataBase
}

type IssuingReqAction struct {
	Meta    IssuingRequest `json:"meta"`
	TxReqID common.Hash    `json:"txReqId"`
}

type IssuingAcceptedInst struct {
	ShardID         byte                   `json:"shardId"`
	DepositedAmount uint64                 `json:"issuingAmount"`
	ReceiverAddr    privacy.PaymentAddress `json:"receiverAddrStr"`
	IncTokenID      common.Hash            `json:"incTokenId"`
	IncTokenName    string                 `json:"incTokenName"`
	TxReqID         common.Hash            `json:"txReqId"`
}

func ParseIssuingInstContent(instContentStr string) (*IssuingReqAction, error) {
	contentBytes, err := base64.StdEncoding.DecodeString(instContentStr)
	if err != nil {
		return nil, err
	}
	var issuingReqAction IssuingReqAction
	err = json.Unmarshal(contentBytes, &issuingReqAction)
	if err != nil {
		return nil, err
	}
	return &issuingReqAction, nil
}

func NewIssuingRequest(
	receiverAddress privacy.PaymentAddress,
	depositedAmount uint64,
	tokenID common.Hash,
	tokenName string,
	metaType int,
) (*IssuingRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}
	issuingReq := &IssuingRequest{
		ReceiverAddress: receiverAddress,
		DepositedAmount: depositedAmount,
		TokenID:         tokenID,
		TokenName:       tokenName,
	}
	issuingReq.MetadataBase = metadataBase
	return issuingReq, nil
}

func NewIssuingRequestFromMap(data map[string]interface{}) (Metadata, error) {
	tokenID, err := common.Hash{}.NewHashFromStr(data["TokenID"].(string))
	if err != nil {
		return nil, errors.New("TokenID incorrect")
	}

	tokenName, ok := data["TokenName"].(string)
	if !ok {
		return nil, errors.New("TokenName incorrect")
	}

	depositedAmount, ok := data["DepositedAmount"]
	if !ok {
		return nil, errors.New("DepositedAmount incorrect")
	}
	depositedAmountFloat, ok := depositedAmount.(float64)
	if !ok {
		return nil, errors.New("DepositedAmount incorrect")
	}
	depositedAmt := uint64(depositedAmountFloat)
	keyWallet, err := wallet.Base58CheckDeserialize(data["ReceiveAddress"].(string))
	if err != nil {
		return nil, errors.New("ReceiveAddress incorrect")
	}

	return NewIssuingRequest(
		keyWallet.KeySet.PaymentAddress,
		depositedAmt,
		*tokenID,
		tokenName,
		IssuingRequestMeta,
	)
}

func (*IssuingRequest) ShouldSignMetaData() bool { return true }

func (iReq IssuingRequest) Hash() *common.Hash {
	record := iReq.ReceiverAddress.String()
	record += iReq.TokenID.String()
	record += string(iReq.DepositedAmount)
	record += iReq.TokenName
	record += iReq.MetadataBase.Hash().String()
	if iReq.Sig != nil && len(iReq.Sig) != 0 {
		record += string(iReq.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iReq IssuingRequest) HashWithoutSig() *common.Hash {
	record := iReq.ReceiverAddress.String()
	record += iReq.TokenID.String()
	record += string(iReq.DepositedAmount)
	record += iReq.TokenName
	record += iReq.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}