package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
	"strconv"
)

// PortalRequestPTokens - portal user requests ptoken (after sending pubToken to custodians)
// metadata - user requests ptoken - create normal tx with this metadata
type PortalRequestPTokens struct {
	MetadataBase
	UniquePortingID string
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	PortingAmount   uint64
	PortingProof    string
}

// PortalRequestPTokensAction - shard validator creates instruction that contain this action content
type PortalRequestPTokensAction struct {
	Meta    PortalRequestPTokens
	TxReqID common.Hash
	ShardID byte
}

// PortalRequestPTokensContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalRequestPTokensContent struct {
	UniquePortingID string
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	PortingAmount   uint64
	PortingProof    string
	TxReqID         common.Hash
	ShardID         byte
}

// PortalRequestPTokensStatus - Beacon tracks status of request ptokens into db
type PortalRequestPTokensStatus struct {
	Status          byte
	UniquePortingID string
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	PortingAmount   uint64
	PortingProof    string
	TxReqID         common.Hash
}

func NewPortalRequestPTokens(
	metaType int,
	uniquePortingID string,
	tokenID string,
	incogAddressStr string,
	portingAmount uint64,
	portingProof string) (*PortalRequestPTokens, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	requestPTokenMeta := &PortalRequestPTokens{
		UniquePortingID: uniquePortingID,
		TokenID:         tokenID,
		IncogAddressStr: incogAddressStr,
		PortingAmount:   portingAmount,
		PortingProof:    portingProof,
	}
	requestPTokenMeta.MetadataBase = metadataBase
	return requestPTokenMeta, nil
}

func (reqPToken PortalRequestPTokens) ValidateTxWithBlockChain(
	txr Transaction,
	chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (reqPToken PortalRequestPTokens) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, txr Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	//if txr.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(txr).String() == "*transaction.Tx" {
	//	return true, true, nil
	//}

	// validate IncogAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(reqPToken.IncogAddressStr)
	if err != nil {
		return false, false, NewMetadataTxError(PortalRequestPTokenParamError, errors.New("Requester incognito address is invalid"))
	}
	if len(keyWallet.KeySet.PaymentAddress.Pk) == 0 {
		return false, false, NewMetadataTxError(PortalRequestPTokenParamError, errors.New("Requester incognito address is invalid"))
	}

	// check tx type
	if txr.GetType() != common.TxNormalType {
		return false, false, errors.New("tx ptoken request must be TxNormalType")
	}

	// validate amount deposit
	if reqPToken.PortingAmount == 0 {
		return false, false, errors.New("porting amount should be larger than 0")
	}

	// validate tokenID and porting proof
	if !common.IsPortalToken(reqPToken.TokenID) {
		return false, false, NewMetadataTxError(PortalRequestPTokenParamError, errors.New("TokenID is not supported currently on Portal"))
	}

	return true, true, nil
}

func (reqPToken PortalRequestPTokens) ValidateMetadataByItself() bool {
	return reqPToken.Type == PortalUserRequestPTokenMeta
}

func (reqPToken PortalRequestPTokens) Hash() *common.Hash {
	record := reqPToken.MetadataBase.Hash().String()
	record += reqPToken.UniquePortingID
	record += reqPToken.TokenID
	record += reqPToken.IncogAddressStr
	record += strconv.FormatUint(reqPToken.PortingAmount, 10)
	record += reqPToken.PortingProof
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (reqPToken *PortalRequestPTokens) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PortalRequestPTokensAction{
		Meta:    *reqPToken,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(PortalUserRequestPTokenMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (reqPToken *PortalRequestPTokens) CalculateSize() uint64 {
	return calculateSize(reqPToken)
}
