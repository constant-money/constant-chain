package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
)

// PortalRedeemRequest - portal user redeem requests to get public token by burning ptoken
// metadata - redeem request - create normal tx with this metadata
type PortalRedeemRequest struct {
	MetadataBase
	UniqueRedeemID        string
	TokenID               string // pTokenID in incognito chain
	RedeemAmount          uint64
	RedeemerIncAddressStr string
	RemoteAddress         string // btc/bnb/etc address
	RedeemFee             uint64 // redeem fee in PRV, 0.01% redeemAmount in PRV
}

// PortalRedeemRequestAction - shard validator creates instruction that contain this action content
type PortalRedeemRequestAction struct {
	Meta    PortalRedeemRequest
	TxReqID common.Hash
	ShardID byte
}

// PortalRedeemRequestContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalRedeemRequestContent struct {
	UniqueRedeemID          string
	TokenID                 string // pTokenID in incognito chain
	RedeemAmount            uint64
	RedeemerIncAddressStr   string
	RemoteAddress           string                                   // btc/bnb/etc address
	RedeemFee               uint64                                   // redeem fee in PRV, 0.01% redeemAmount in PRV
	MatchingCustodianDetail []*statedb.MatchingRedeemCustodianDetail // key: incAddressCustodian
	TxReqID                 common.Hash
	ShardID                 byte
}

// PortalRedeemRequestStatus - Beacon tracks status of redeem request into db
type PortalRedeemRequestStatus struct {
	Status                  byte
	UniqueRedeemID          string
	TokenID                 string // pTokenID in incognito chain
	RedeemAmount            uint64
	RedeemerIncAddressStr   string
	RemoteAddress           string                                   // btc/bnb/etc address
	RedeemFee               uint64                                   // redeem fee in PRV, 0.01% redeemAmount in PRV
	MatchingCustodianDetail []*statedb.MatchingRedeemCustodianDetail // key: incAddressCustodian
	TxReqID                 common.Hash
}

func NewPortalRedeemRequest(
	metaType int,
	uniqueRedeemID string,
	tokenID string,
	redeemAmount uint64,
	incAddressStr string,
	remoteAddr string,
	redeemFee uint64) (*PortalRedeemRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	requestPTokenMeta := &PortalRedeemRequest{
		UniqueRedeemID:        uniqueRedeemID,
		TokenID:               tokenID,
		RedeemAmount:          redeemAmount,
		RedeemerIncAddressStr: incAddressStr,
		RemoteAddress:         remoteAddr,
		RedeemFee:             redeemFee,
	}
	requestPTokenMeta.MetadataBase = metadataBase
	return requestPTokenMeta, nil
}

func (redeemReq PortalRedeemRequest) ValidateTxWithBlockChain(
	txr Transaction,
	chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (redeemReq PortalRedeemRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, txr Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	if txr.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(txr).String() == "*transaction.Tx" {
		if !txr.IsCoinsBurning(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight) {
			return false, false, errors.New("txnormal in tx redeem request must be coin burning tx")
		}
		// validate value transfer of tx for redeem fee in prv
		if redeemReq.RedeemFee != txr.CalculateTxValue() {
			return false, false, errors.New("redeem fee amount should be equal to the tx value")
		}
		return true, true, nil
	}

	// validate RedeemerIncAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(redeemReq.RedeemerIncAddressStr)
	if err != nil {
		return false, false, NewMetadataTxError(PortalRedeemRequestParamError, errors.New("Requester incognito address is invalid"))
	}

	if len(keyWallet.KeySet.PaymentAddress.Pk) == 0 {
		return false, false, NewMetadataTxError(PortalRedeemRequestParamError, errors.New("Requester incognito address is invalid"))
	}

	// check burning tx
	isBurned, burnCoin, burnedTokenID, err := txr.GetTxBurnData()
	if err != nil || !isBurned {
		return false, false, errors.New("Error This is not Tx Burn")
	}

	// check tx type
	if txr.GetType() != common.TxCustomTokenPrivacyType {
		return false, false, errors.New("tx redeem request must be TxCustomTokenPrivacyType")
	}

	// validate redeem amount
	minAmount := common.MinAmountPortalPToken[redeemReq.TokenID]
	if redeemReq.RedeemAmount < minAmount {
		return false, false, fmt.Errorf("redeem amount should be larger or equal to %v", minAmount)
	}

	// validate redeem fee
	if redeemReq.RedeemFee <= 0 {
		return false, false, errors.New("redeem fee should be larger than 0")
	}

	// validate value transfer of tx for redeem amount in ptoken
	if redeemReq.RedeemAmount != burnCoin.GetValue() {
		return false, false, errors.New("redeem amount should be equal to the tx value")
	}

	// validate tokenID
	if redeemReq.TokenID != burnedTokenID.String() {
		return false, false, NewMetadataTxError(PortalRedeemRequestParamError, errors.New("TokenID in metadata is not matched to tokenID in tx"))
	}
	// check tokenId is portal token or not
	if !common.IsPortalToken(redeemReq.TokenID) {
		return false, false, NewMetadataTxError(PortalRedeemRequestParamError, errors.New("TokenID is not in portal tokens list"))
	}

	//validate RemoteAddress
	if len(redeemReq.RemoteAddress) == 0 {
		return false, false, NewMetadataTxError(PortalRedeemRequestParamError, errors.New("Remote address is invalid"))
	}
	chainID := GetChainIDByTokenID(redeemReq.TokenID, chainRetriever)
	if !IsValidRemoteAddress(chainRetriever, redeemReq.RemoteAddress, redeemReq.TokenID, chainID) {
		return false, false, fmt.Errorf("Remote address %v is not a valid address of tokenID %v", redeemReq.RemoteAddress, redeemReq.TokenID)
	}

	return true, true, nil
}

func (redeemReq PortalRedeemRequest) ValidateMetadataByItself() bool {
	return redeemReq.Type == PortalRedeemRequestMeta
}

func (redeemReq PortalRedeemRequest) Hash() *common.Hash {
	record := redeemReq.MetadataBase.Hash().String()
	record += redeemReq.UniqueRedeemID
	record += redeemReq.TokenID
	record += strconv.FormatUint(redeemReq.RedeemAmount, 10)
	record += strconv.FormatUint(redeemReq.RedeemFee, 10)
	record += redeemReq.RedeemerIncAddressStr
	record += redeemReq.RemoteAddress
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (redeemReq *PortalRedeemRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PortalRedeemRequestAction{
		Meta:    *redeemReq,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(PortalRedeemRequestMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (redeemReq *PortalRedeemRequest) CalculateSize() uint64 {
	return calculateSize(redeemReq)
}
