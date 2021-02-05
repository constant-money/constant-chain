package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"strconv"
)

type PortalSubmitConfirmedTxRequest struct {
	basemeta.MetadataBase
	TokenID       string // pTokenID in incognito chain
	UnshieldProof string
	BatchID       string
}

type PortalSubmitConfirmedTxAction struct {
	Meta    PortalSubmitConfirmedTxRequest
	TxReqID common.Hash
	ShardID byte
}

type PortalSubmitConfirmedTxContent struct {
	TokenID       string
	UnshieldProof string
	UTXOs         []*statedb.UTXO
	BatchID       string
	TxReqID       common.Hash
	ShardID       byte
}

type PortalSubmitConfirmedTxStatus struct {
	TokenID       string
	UnshieldProof string
	UTXOs         []*statedb.UTXO
	BatchID       string
	TxHash        string
	Status        int
}

func NewPortalSubmitConfirmedTxStatus(unshieldProof, tokenID, batchID string, UTXOs []*statedb.UTXO, status int) *PortalSubmitConfirmedTxStatus {
	return &PortalSubmitConfirmedTxStatus{
		TokenID:       tokenID,
		BatchID:       batchID,
		UnshieldProof: unshieldProof,
		UTXOs:         UTXOs,
		Status:        status,
	}
}

func NewPortalSubmitConfirmedTxRequest(metaType int, unshieldProof, tokenID, batchID string) (*PortalSubmitConfirmedTxRequest, error) {
	metadataBase := basemeta.MetadataBase{
		Type: metaType,
	}

	portalUnshieldReq := &PortalSubmitConfirmedTxRequest{
		TokenID:       tokenID,
		BatchID:       batchID,
		UnshieldProof: unshieldProof,
	}

	portalUnshieldReq.MetadataBase = metadataBase

	return portalUnshieldReq, nil
}

func (r PortalSubmitConfirmedTxRequest) ValidateTxWithBlockChain(
	txr basemeta.Transaction,
	chainRetriever basemeta.ChainRetriever,
	shardViewRetriever basemeta.ShardViewRetriever,
	beaconViewRetriever basemeta.BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (r PortalSubmitConfirmedTxRequest) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, tx basemeta.Transaction) (bool, bool, error) {
	// check tx type
	if tx.GetType() != common.TxNormalType {
		return false, false, NewPortalV4MetadataError(PortalSubmitConfirmedTxRequestMetaError, errors.New("tx replace transaction must be TxNormalType"))
	}

	// check tokenId is portal token or not
	if !chainRetriever.IsPortalToken(beaconHeight, r.TokenID) {
		return false, false, NewPortalV4MetadataError(PortalSubmitConfirmedTxRequestMetaError, errors.New("TokenID is not in portal tokens list"))
	}

	return true, true, nil
}

func (r PortalSubmitConfirmedTxRequest) ValidateMetadataByItself() bool {
	return r.Type == basemeta.PortalSubmitConfirmedTxMeta
}

func (r PortalSubmitConfirmedTxRequest) Hash() *common.Hash {
	record := r.MetadataBase.Hash().String()
	record += r.TokenID
	record += r.BatchID
	record += r.UnshieldProof

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (r *PortalSubmitConfirmedTxRequest) BuildReqActions(tx basemeta.Transaction, chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	actionContent := PortalSubmitConfirmedTxAction{
		Meta:    *r,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(basemeta.PortalSubmitConfirmedTxMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (r *PortalSubmitConfirmedTxRequest) CalculateSize() uint64 {
	return basemeta.CalculateSize(r)
}
