package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
)

// PortalShieldingRequest - portal user requests ptoken (after sending pubToken to multisig wallet)
// metadata - portal user sends shielding request - create normal tx with this metadata
type PortalShieldingRequest struct {
	basemeta.MetadataBase
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	ShieldingProof  string
}

// PortalShieldingRequestAction - shard validator creates instruction that contain this action content
type PortalShieldingRequestAction struct {
	Meta    PortalShieldingRequest
	TxReqID common.Hash
	ShardID byte
}

// PortalShieldingRequestContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalShieldingRequestContent struct {
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	ProofHash       string
	ShieldingUTXO   []*statedb.UTXO
	TxReqID         common.Hash
	ShardID         byte
}

// PortalRequestPTokensStatus - Beacon tracks status of request ptokens into db
type PortalShieldingRequestStatus struct {
	Status          byte
	TokenID         string // pTokenID in incognito chain
	IncogAddressStr string
	ProofHash       string
	ShieldingUTXO   []*statedb.UTXO
	TxReqID         common.Hash
}

func NewPortalShieldingRequest(
	metaType int,
	tokenID string,
	incogAddressStr string,
	shieldingProof string) (*PortalShieldingRequest, error) {
	metadataBase := basemeta.MetadataBase{
		Type: metaType,
	}
	shieldingRequestMeta := &PortalShieldingRequest{
		TokenID:         tokenID,
		IncogAddressStr: incogAddressStr,
		ShieldingProof:  shieldingProof,
	}
	shieldingRequestMeta.MetadataBase = metadataBase
	return shieldingRequestMeta, nil
}

func (shieldingReq PortalShieldingRequest) ValidateTxWithBlockChain(
	txr basemeta.Transaction,
	chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (shieldingReq PortalShieldingRequest) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, txr basemeta.Transaction) (bool, bool, error) {
	// validate IncogAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(shieldingReq.IncogAddressStr)
	if err != nil {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRequestPTokenParamError, errors.New("Requester incognito address is invalid"))
	}
	incogAddr := keyWallet.KeySet.PaymentAddress
	if len(incogAddr.Pk) == 0 {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRequestPTokenParamError, errors.New("Requester incognito address is invalid"))
	}
	// let anyone can submit the proof
	//if !bytes.Equal(txr.GetSigPubKey()[:], incogAddr.Pk[:]) {
	//	return false, false, basemeta.NewMetadataTxError(basemeta.PortalRequestPTokenParamError, errors.New("Requester incognito address is not signer"))
	//}

	// check tx type
	if txr.GetType() != common.TxNormalType {
		return false, false, errors.New("tx custodian deposit must be TxNormalType")
	}

	// validate tokenID and shielding proof
	if !chainRetriever.IsPortalToken(beaconHeight, shieldingReq.TokenID) {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRequestPTokenParamError, errors.New("TokenID is not supported currently on Portal"))
	}

	return true, true, nil
}

func (shieldingReq PortalShieldingRequest) ValidateMetadataByItself() bool {
	return shieldingReq.Type == basemeta.PortalShieldingRequestMeta
}

func (shieldingReq PortalShieldingRequest) Hash() *common.Hash {
	record := shieldingReq.MetadataBase.Hash().String()
	record += shieldingReq.TokenID
	record += shieldingReq.IncogAddressStr
	record += shieldingReq.ShieldingProof
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (shieldingReq *PortalShieldingRequest) BuildReqActions(tx basemeta.Transaction, chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	actionContent := PortalShieldingRequestAction{
		Meta:    *shieldingReq,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(basemeta.PortalShieldingRequestMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (shieldingReq *PortalShieldingRequest) CalculateSize() uint64 {
	return basemeta.CalculateSize(shieldingReq)
}
