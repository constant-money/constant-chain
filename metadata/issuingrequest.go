package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
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
		return nil, NewMetadataTxError(IssuingRequestDecodeInstructionError, err)
	}
	var issuingReqAction IssuingReqAction
	err = json.Unmarshal(contentBytes, &issuingReqAction)
	if err != nil {
		return nil, NewMetadataTxError(IssuingRequestUnmarshalJsonError, err)
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
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenID incorrect"))
	}

	tokenName, ok := data["TokenName"].(string)
	if !ok {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenName incorrect"))
	}

	depositedAmount, ok := data["DepositedAmount"]
	if !ok {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("DepositedAmount incorrect"))
	}
	depositedAmountFloat, ok := depositedAmount.(float64)
	if !ok {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("DepositedAmount incorrect"))
	}
	depositedAmt := uint64(depositedAmountFloat)
	keyWallet, err := wallet.Base58CheckDeserialize(data["ReceiveAddress"].(string))
	if err != nil {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("ReceiveAddress incorrect"))
	}

	return NewIssuingRequest(
		keyWallet.KeySet.PaymentAddress,
		depositedAmt,
		*tokenID,
		tokenName,
		IssuingRequestMeta,
	)
}

func (iReq IssuingRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	shardBlockBeaconHeight := shardViewRetriever.GetBeaconHeight()
	keySet, err := wallet.Base58CheckDeserialize(chainRetriever.GetCentralizedWebsitePaymentAddress(shardBlockBeaconHeight))
	if err != nil {
		return false, NewMetadataTxError(IssuingRequestValidateTxWithBlockChainError, errors.New("cannot get centralized website payment address"))
	}
	if ok, err := tx.CheckAuthorizedSender(keySet.KeySet.PaymentAddress.Pk); err != nil || !ok {
		fmt.Println("Check authorized sender fail:", ok, err)
		return false, NewMetadataTxError(IssuingRequestValidateTxWithBlockChainError, errors.New("the issuance request must be called by centralized website"))
	}

	// check this is a normal pToken
	if statedb.PrivacyTokenIDExisted(transactionStateDB, iReq.TokenID) {
		isBridgeToken, err := statedb.IsBridgeTokenExistedByType(beaconViewRetriever.GetBeaconFeatureStateDB(), iReq.TokenID, true)
		if !isBridgeToken {
			if err != nil {
				return false, NewMetadataTxError(InvalidMeta, err)
			} else {
				return false, NewMetadataTxError(InvalidMeta, errors.New("token is invalid"))
			}
		}
	}
	return true, nil
}

func (iReq IssuingRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	if len(iReq.ReceiverAddress.Pk) == 0 {
		return false, false, NewMetadataTxError(IssuingRequestValidateSanityDataError, errors.New("Wrong request info's receiver address"))
	}
	if iReq.DepositedAmount == 0 {
		return false, false, errors.New("Wrong request info's deposited amount")
	}
	if iReq.Type != IssuingRequestMeta {
		return false, false, NewMetadataTxError(IssuingRequestValidateSanityDataError, errors.New("Wrong request info's meta type"))
	}
	if iReq.TokenName == "" {
		return false, false, NewMetadataTxError(IssuingRequestValidateSanityDataError, errors.New("Wrong request info's token name"))
	}
	return true, true, nil
}

func (iReq IssuingRequest) ValidateMetadataByItself() bool {
	return iReq.Type == IssuingRequestMeta
}

func (*IssuingRequest) ShouldSignMetaData() bool { return true }

func (iReq IssuingRequest) Hash() *common.Hash {
	record := iReq.ReceiverAddress.String()
	record += iReq.TokenID.String()
	// TODO: @hung change to record += fmt.Sprint(iReq.DepositedAmount)
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

func (iReq *IssuingRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	txReqID := *(tx.Hash())
	actionContent := map[string]interface{}{
		"meta":    *iReq,
		"txReqId": txReqID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, NewMetadataTxError(IssuingRequestBuildReqActionsError, err)
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(IssuingRequestMeta), actionContentBase64Str}
	// track the request status to leveldb
	//err = statedb.TrackBridgeReqWithStatus(bcr.GetBeaconFeatureStateDB(), txReqID, byte(common.BridgeRequestProcessingStatus))
	//if err != nil {
	//	return [][]string{}, NewMetadataTxError(IssuingRequestBuildReqActionsError, err)
	//}
	return [][]string{action}, nil
}

func (iReq *IssuingRequest) CalculateSize() uint64 {
	return calculateSize(iReq)
}
