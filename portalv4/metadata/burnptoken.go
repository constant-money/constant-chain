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

type PortalBurnPToken struct {
	basemeta.MetadataBase
	IncAddressStr        string
	RemoteAddress        string
	TokenID              string
	BurnAmount           uint64
	MinAcceptPubTokenAmt uint64
}

type PortalBurnPTokenAction struct {
	Meta    PortalBurnPToken
	TxReqID common.Hash
	ShardID byte
}

type PortalBurnPTokenContent struct {
	IncAddressStr          string
	RemoteAddress          string
	TokenID                string
	BurnAmount             uint64
	MinReceivedPubTokenAmt uint64
	TxReqID                common.Hash
	ShardID                byte
}

type PortalBurnPTokenRequestStatus struct {
	IncAddressStr          string
	RemoteAddress          string
	TokenID                string
	BurnAmount             uint64
	MinReceivedPubTokenAmt uint64
	TxHash                 string
	Status                 int
}

func NewPortalBurnPTokenRequestStatus(incAddressStr, tokenID, remoteAddress string, burnAmount uint64, minAmt uint64, status int) *PortalBurnPTokenRequestStatus {
	return &PortalBurnPTokenRequestStatus{
		IncAddressStr:          incAddressStr,
		BurnAmount:             burnAmount,
		Status:                 status,
		TokenID:                tokenID,
		RemoteAddress:          remoteAddress,
		MinReceivedPubTokenAmt: minAmt,
	}
}

func NewPortalBurnPToken(metaType int, incAddressStr, tokenID, remoteAddress string, burnAmount uint64, minAmt uint64) (*PortalBurnPToken, error) {
	metadataBase := basemeta.MetadataBase{
		Type: metaType,
	}

	portalBurnPTokenReq := &PortalBurnPToken{
		IncAddressStr:        incAddressStr,
		BurnAmount:           burnAmount,
		MinAcceptPubTokenAmt: minAmt,
		RemoteAddress:        remoteAddress,
		TokenID:              tokenID,
	}

	portalBurnPTokenReq.MetadataBase = metadataBase

	return portalBurnPTokenReq, nil
}

func (burnReq PortalBurnPToken) ValidateTxWithBlockChain(
	txr basemeta.Transaction,
	chainRetriever basemeta.ChainRetriever,
	shardViewRetriever basemeta.ShardViewRetriever,
	beaconViewRetriever basemeta.BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (burnReq PortalBurnPToken) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, tx basemeta.Transaction) (bool, bool, error) {
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
	if burnReq.BurnAmount < minAmount {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, fmt.Errorf("burning amount should be larger or equal to %v", minAmount))
	}

	// validate value transfer of tx for redeem amount in ptoken
	if burnReq.BurnAmount != tx.CalculateTxValue() {
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

	// validate MinAcceptPubTokenAmt
	if burnReq.MinAcceptPubTokenAmt == 0 {
		return false, false, NewPortalV4MetadataError(PortalBurnPTokenMetaError, errors.New("MinAcceptPubTokenAmt should be greater than zero"))
	}

	return true, true, nil
}

func (burnReq PortalBurnPToken) ValidateMetadataByItself() bool {
	return burnReq.Type == basemeta.PortalBurnPTokenMeta
}

func (burnReq PortalBurnPToken) Hash() *common.Hash {
	record := burnReq.MetadataBase.Hash().String()
	record += burnReq.IncAddressStr
	record += burnReq.RemoteAddress
	record += strconv.FormatUint(burnReq.BurnAmount, 10)
	record += strconv.FormatUint(burnReq.MinAcceptPubTokenAmt, 10)
	record += burnReq.TokenID

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (burnReq *PortalBurnPToken) BuildReqActions(tx basemeta.Transaction, chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	actionContent := PortalBurnPTokenAction{
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

func (burnReq *PortalBurnPToken) CalculateSize() uint64 {
	return basemeta.CalculateSize(burnReq)
}
