package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
	"strconv"
)

type PortalReplacementFeeRequest struct {
	basemeta.MetadataBase
	IncAddressStr string
	TokenID       string
	BatchID       string
	Fee           uint
}

type PortalReplacementFeeRequestAction struct {
	Meta    PortalReplacementFeeRequest
	TxReqID common.Hash
	ShardID byte
}

type PortalReplacementFeeRequestContent struct {
	IncAddressStr string
	TokenID       string
	BatchID       string
	Fee           uint
	ExternalRawTx string
	TxReqID       common.Hash
	ShardID       byte
}

type PortalReplacementFeeRequestStatus struct {
	IncAddressStr string
	TokenID       string
	BatchID       string
	Fee           uint
	TxHash        string
	ExternalRawTx string
	Status        int
}

func NewPortalReplacementFeeRequestStatus(incAddressStr, tokenID, batchID string, fee uint, externalRawTx string, status int) *PortalReplacementFeeRequestStatus {
	return &PortalReplacementFeeRequestStatus{
		IncAddressStr: incAddressStr,
		TokenID:       tokenID,
		BatchID:       batchID,
		Fee:           fee,
		ExternalRawTx: externalRawTx,
		Status:        status,
	}
}

func NewPortalReplacementFeeRequest(metaType int, incAddressStr, tokenID, batchID string, fee uint) (*PortalReplacementFeeRequest, error) {
	metadataBase := basemeta.MetadataBase{
		Type: metaType,
	}

	portalUnshieldReq := &PortalReplacementFeeRequest{
		IncAddressStr: incAddressStr,
		TokenID:       tokenID,
		BatchID:       batchID,
		Fee:           fee,
	}

	portalUnshieldReq.MetadataBase = metadataBase

	return portalUnshieldReq, nil
}

func (repl PortalReplacementFeeRequest) ValidateTxWithBlockChain(
	txr basemeta.Transaction,
	chainRetriever basemeta.ChainRetriever,
	shardViewRetriever basemeta.ShardViewRetriever,
	beaconViewRetriever basemeta.BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (repl PortalReplacementFeeRequest) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, tx basemeta.Transaction) (bool, bool, error) {

	if repl.IncAddressStr != chainRetriever.GetPortalReplacementAddress(beaconHeight) {
		return false, false, NewPortalV4MetadataError(PortalReplacementFeeRequestMetaError, errors.New("Requester incognito address is invalid"))
	}

	// validate IncAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(repl.IncAddressStr)
	if err != nil {
		return false, false, NewPortalV4MetadataError(PortalReplacementFeeRequestMetaError, errors.New("Requester incognito address is invalid"))
	}
	incAddr := keyWallet.KeySet.PaymentAddress
	if len(incAddr.Pk) == 0 {
		return false, false, NewPortalV4MetadataError(PortalReplacementFeeRequestMetaError, errors.New("Requester incognito address is invalid"))
	}

	if !bytes.Equal(tx.GetSigPubKey()[:], incAddr.Pk[:]) {
		return false, false, NewPortalV4MetadataError(PortalReplacementFeeRequestMetaError, errors.New("Requester incognito address is not signer"))
	}

	// check tx type
	if tx.GetType() != common.TxNormalType {
		return false, false, NewPortalV4MetadataError(PortalReplacementFeeRequestMetaError, errors.New("tx replace transaction must be TxNormalType"))
	}

	// check tokenId is portal token or not
	if !chainRetriever.IsPortalToken(beaconHeight, repl.TokenID) {
		return false, false, NewPortalV4MetadataError(PortalReplacementFeeRequestMetaError, errors.New("TokenID is not in portal tokens list"))
	}

	return true, true, nil
}

func (repl PortalReplacementFeeRequest) ValidateMetadataByItself() bool {
	return repl.Type == basemeta.PortalReplacementFeeRequestMeta
}

func (repl PortalReplacementFeeRequest) Hash() *common.Hash {
	record := repl.MetadataBase.Hash().String()
	record += repl.IncAddressStr
	record += repl.TokenID
	record += repl.BatchID
	record += strconv.FormatUint(uint64(repl.Fee), 10)

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (repl *PortalReplacementFeeRequest) BuildReqActions(tx basemeta.Transaction, chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	actionContent := PortalReplacementFeeRequestAction{
		Meta:    *repl,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(basemeta.PortalReplacementFeeRequestMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (repl *PortalReplacementFeeRequest) CalculateSize() uint64 {
	return basemeta.CalculateSize(repl)
}
