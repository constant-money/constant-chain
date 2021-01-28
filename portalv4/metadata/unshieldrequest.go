package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
	"reflect"
	"strconv"
)

type PortalUnshieldRequest struct {
	basemeta.MetadataBase
	IncAddressStr  string
	RemoteAddress  string
	TokenID        string
	UnshieldAmount uint64
}

type PortalUnshieldRequestAction struct {
	Meta    PortalUnshieldRequest
	TxReqID common.Hash
	ShardID byte
}

type PortalUnshieldRequestContent struct {
	IncAddressStr  string
	RemoteAddress  string
	TokenID        string
	UnshieldAmount uint64
	TxReqID        common.Hash
	ShardID        byte
}

type PortalUnshieldRequestStatus struct {
	IncAddressStr  string
	RemoteAddress  string
	TokenID        string
	UnshieldAmount uint64
	TxHash         string
	Status         int
}

func NewPortalUnshieldRequestStatus(incAddressStr, tokenID, remoteAddress string, burnAmount uint64, status int) *PortalUnshieldRequestStatus {
	return &PortalUnshieldRequestStatus{
		IncAddressStr:          incAddressStr,
		UnshieldAmount:         burnAmount,
		Status:                 status,
		TokenID:                tokenID,
		RemoteAddress:          remoteAddress,
	}
}

func NewPortalUnshieldRequest(metaType int, incAddressStr, tokenID, remoteAddress string, burnAmount uint64) (*PortalUnshieldRequest, error) {
	metadataBase := basemeta.MetadataBase{
		Type: metaType,
	}

	portalBurnPTokenReq := &PortalUnshieldRequest{
		IncAddressStr:        incAddressStr,
		UnshieldAmount:       burnAmount,
		RemoteAddress:        remoteAddress,
		TokenID:              tokenID,
	}

	portalBurnPTokenReq.MetadataBase = metadataBase

	return portalBurnPTokenReq, nil
}

func (burnReq PortalUnshieldRequest) ValidateTxWithBlockChain(
	txr basemeta.Transaction,
	chainRetriever basemeta.ChainRetriever,
	shardViewRetriever basemeta.ShardViewRetriever,
	beaconViewRetriever basemeta.BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (burnReq PortalUnshieldRequest) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, tx basemeta.Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	if tx.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(tx).String() == "*transaction.Tx" {
		return true, true, nil
	}

	// validate RedeemerIncAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(burnReq.IncAddressStr)
	if err != nil {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("Requester incognito address is invalid"))
	}
	incAddr := keyWallet.KeySet.PaymentAddress
	if len(incAddr.Pk) == 0 {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("Requester incognito address is invalid"))
	}
	if !bytes.Equal(tx.GetSigPubKey()[:], incAddr.Pk[:]) {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("Requester incognito address is not signer"))
	}

	// check tx type
	if tx.GetType() != common.TxCustomTokenPrivacyType {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("tx burn ptoken must be TxCustomTokenPrivacyType"))
	}

	if !tx.IsCoinsBurning(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight) {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("txprivacytoken in tx burn ptoken must be coin burning tx"))
	}

	// validate burning amount
	minAmount, err := chainRetriever.GetMinAmountPortalToken(burnReq.TokenID, beaconHeight)
	if err != nil {
		return false, false, fmt.Errorf("Error get min portal token amount: %v", err)
	}
	if burnReq.UnshieldAmount < minAmount {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, fmt.Errorf("burning amount should be larger or equal to %v", minAmount))
	}

	// validate value transfer of tx for redeem amount in ptoken
	if burnReq.UnshieldAmount != tx.CalculateTxValue() {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("burning amount should be equal to the tx value"))
	}

	// validate tokenID
	if burnReq.TokenID != tx.GetTokenID().String() {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("TokenID in metadata is not matched to tokenID in tx"))
	}
	// check tokenId is portal token or not
	if !chainRetriever.IsPortalToken(beaconHeight, burnReq.TokenID) {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("TokenID is not in portal tokens list"))
	}

	// validate RemoteAddress
	if len(burnReq.RemoteAddress) == 0 {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("Remote address is invalid"))
	}
	isValidRemoteAddress, err := chainRetriever.IsValidPortalRemoteAddress(burnReq.TokenID, burnReq.RemoteAddress, beaconHeight)
	if err != nil || !isValidRemoteAddress {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, fmt.Errorf("Remote address %v is not a valid address of tokenID %v - Error %v", burnReq.RemoteAddress, burnReq.TokenID, err))
	}

	return true, true, nil
}

func (burnReq PortalUnshieldRequest) ValidateMetadataByItself() bool {
	return burnReq.Type == basemeta.PortalBurnPTokenMeta
}

func (burnReq PortalUnshieldRequest) Hash() *common.Hash {
	record := burnReq.MetadataBase.Hash().String()
	record += burnReq.IncAddressStr
	record += burnReq.RemoteAddress
	record += strconv.FormatUint(burnReq.UnshieldAmount, 10)
	record += burnReq.TokenID

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (burnReq *PortalUnshieldRequest) BuildReqActions(tx basemeta.Transaction, chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	actionContent := PortalUnshieldRequestAction{
		Meta:    *burnReq,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(basemeta.PortalBurnPTokenMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (burnReq *PortalUnshieldRequest) CalculateSize() uint64 {
	return basemeta.CalculateSize(burnReq)
}
