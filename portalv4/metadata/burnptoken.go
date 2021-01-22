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
	"strconv"
)

type PortalBurnPToken struct {
	basemeta.MetadataBase
	IncAddressStr string
	RemoteAddress string
	BurnAmount    uint64
	TokenID       string
}

type PortalBurnPTokenAction struct {
	Meta    PortalBurnPToken
	TxReqID common.Hash
	ShardID byte
}

type PortalBurnPTokenContent struct {
	IncAddressStr string
	RemoteAddress string
	BurnAmount    uint64
	TokenID       string
	TxReqID       common.Hash
	ShardID       byte
}

type PortalBurnPTokenRequestStatus struct {
	IncAddressStr string
	RemoteAddress string
	BurnAmount    uint64
	TokenID       string
	TxHash        string
	Status        int
}

func NewPortalBurnPTokenRequestStatus(incAddressStr, tokenID, remoteAddress string, burnAmount uint64, status int) *PortalBurnPTokenRequestStatus {
	return &PortalBurnPTokenRequestStatus{IncAddressStr: incAddressStr, BurnAmount: burnAmount, Status: status, TokenID: tokenID, RemoteAddress: remoteAddress}
}

func NewPortalBurnPToken(metaType int, incAddressStr, tokenID, remoteAddress string, burnAmount uint64) (*PortalBurnPToken, error) {
	metadataBase := basemeta.MetadataBase{
		Type: metaType,
	}

	portalBurnPTokenReq := &PortalBurnPToken{
		IncAddressStr: incAddressStr,
		BurnAmount:    burnAmount,
		RemoteAddress: remoteAddress,
		TokenID:       tokenID,
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
	// NOTE: verify supported tokens pair as needed
	return true, nil
}

func (burnReq PortalBurnPToken) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, tx basemeta.Transaction) (bool, bool, error) {
	// validate Payment address
	if len(burnReq.IncAddressStr) <= 0 {
		return false, false, errors.New("IncAddressStr should be not empty")
	}
	keyWallet, err := wallet.Base58CheckDeserialize(burnReq.IncAddressStr)
	if err != nil {
		return false, false, errors.New("IncAddressStr incorrect")
	}
	incogAddr := keyWallet.KeySet.PaymentAddress
	if len(incogAddr.Pk) == 0 {
		return false, false, errors.New("wrong incognito address")
	}
	if !bytes.Equal(tx.GetSigPubKey()[:], incogAddr.Pk[:]) {
		return false, false, errors.New("custodian incognito address is not signer tx")
	}

	// check tx type
	if tx.GetType() != common.TxNormalType {
		return false, false, errors.New("tx custodian deposit must be TxNormalType")
	}

	// validate tokenID
	if burnReq.TokenID != tx.GetTokenID().String() {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemRequestParamError, errors.New("TokenID in metadata is not matched to tokenID in tx"))
	}

	//validate RemoteAddress
	if len(burnReq.RemoteAddress) == 0 {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemRequestParamError, errors.New("Remote address is invalid"))
	}

	isValidRemoteAddress, err := chainRetriever.IsValidPortalRemoteAddress(burnReq.TokenID, burnReq.RemoteAddress, beaconHeight)
	if err != nil || !isValidRemoteAddress {
		return false, false, fmt.Errorf("Remote address %v is not a valid address of tokenID %v - Error %v", burnReq.RemoteAddress, burnReq.TokenID, err)
	}

	if !chainRetriever.IsPortalToken(beaconHeight, burnReq.TokenID) {
		return false, false, errors.New("token not supported")
	}

	// check burning tx
	if !tx.IsCoinsBurning(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight) {
		return false, false, errors.New("must send coin to burning address")
	}

	// check withdraw amount
	if burnReq.BurnAmount <= 0 {
		return false, false, errors.New("Burn amount should be larger than 0")
	}

	// validate redeem amount
	minAmount, err := chainRetriever.GetMinAmountPortalToken(burnReq.TokenID, beaconHeight)
	if err != nil {
		return false, false, fmt.Errorf("Error get min portal token amount: %v", err)
	}
	if burnReq.BurnAmount < minAmount {
		return false, false, fmt.Errorf("burn token amount should be larger or equal to %v", minAmount)
	}

	// validate redeem fee
	//if burnReq.RedeemFee <= 0 {
	//	return false, false, errors.New("redeem fee should be larger than 0")
	//}

	// validate value transfer of tx for redeem amount in ptoken
	if burnReq.BurnAmount != tx.CalculateTxValue() {
		return false, false, errors.New("burn amount should be equal to the tx value")
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
