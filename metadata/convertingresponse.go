package metadata

import (
	"bytes"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy/coin"
)

type ConvertingResponse struct {
	TxRequest common.Hash
	TokenID   common.Hash
	MetadataBase
}

func NewConvertingResponse(metaRequest *ConvertingRequest, reqID *common.Hash) (*ConvertingResponse, error) {
	metadataBase := MetadataBase{
		Type: ConvertingResponseMeta,
	}
	result := &ConvertingResponse{
		MetadataBase:    metadataBase,
		TxRequest:       *reqID,
		TokenID:         metaRequest.TokenID,
	}
	return result, nil
}

//ValidateSanityData performs the following verifications:
//	1. Check tx type and supported tokenID
//	2. Check the mintedToken and convertedToken are the same
func (iRes ConvertingResponse) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	//Step 1
	if tx.GetType() == common.TxRewardType && iRes.TokenID.String() != common.PRVIDStr {
		return false, false, NewMetadataTxError(ConvertingTokenIDError, fmt.Errorf("cannot mint token %v in a PRV transaction", iRes.TokenID.String()))
	}
	if tx.GetType() == common.TxCustomTokenPrivacyType && iRes.TokenID.String() == common.PRVIDStr {
		return false, false, NewMetadataTxError(ConvertingTokenIDError, fmt.Errorf("cannot mint PRV in a token transaction"))
	}

	//Step 2
	mintedTokenID := tx.GetTokenID()
	if mintedTokenID.String() != iRes.TokenID.String() {
		return false, false, NewMetadataTxError(ConvertingTokenIDError, fmt.Errorf("mintedToken and convertedToken mismatch: %v != %v", mintedTokenID.String(), iRes.TokenID.String()))
	}

	return false, true, nil
}

func (iRes ConvertingResponse) ValidateMetadataByItself() bool {
	return iRes.Type == ConvertingResponseMeta
}

func (iRes ConvertingResponse) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	return true, nil
}

func (iRes ConvertingResponse) Hash() *common.Hash {
	record := iRes.TxRequest.String()
	record = iRes.TokenID.String()
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes ConvertingResponse) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *MintData, shardID byte, tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever) (bool, error) {
	return true, nil
}

//ValidateTxResponse validates if a converting response is a reply to the converting request by the following checks:
//	1. Check if the req metadata type is valid
//	2. Check the burned and minted tokenIDs
//	3. Check the burned and minted value
//	4. Check the OTA/TxRandom in request metadata and OTA/TxRandom in minted coin
// TODO: reviews should double-check if the above validations are sufficient
func (iRes ConvertingResponse) ValidateTxResponse(txReq, txResp Transaction) (bool, error) {
	Logger.log.Infof("ValidateTxResponse for converting req(%v)/resp(%v)\n", txReq.Hash().String(), txResp.Hash().String())
	//Step 1
	if txReq.GetMetadataType() != ConvertingRequestMeta {
		return false, fmt.Errorf("txReq %v, type %v is not a converting request", txReq.Hash().String(), txReq.GetMetadataType())
	}

	//Step 2
	_, burnedCoin, burnedTokenID, err := txReq.GetTxBurnData()
	if err != nil {
		return false, fmt.Errorf("cannot get tx burned data of txReq %v: %v", txReq.Hash().String(), err)
	}
	_, mintedCoin, mintedTokenID, err := txResp.GetTxMintData()
	if err != nil {
		return false, fmt.Errorf("cannot get tx minted data of txResp %v: %v", txResp.Hash().String(), err)
	}
	if burnedTokenID.String() != mintedTokenID.String() {
		return false, fmt.Errorf("txReq mintedTokenID and txResp mintedTokenID mismatch: %v != %v", burnedTokenID.String(), mintedTokenID.String())
	}

	//Step 3
	if burnedCoin.GetValue() != mintedCoin.GetValue() {
		return false, fmt.Errorf("burned value (%v) and minted value (%v) mismatch", burnedCoin.GetValue(), mintedCoin.GetValue())
	}

	//Step 4
	tmpReqMeta := txReq.GetMetadata()
	reqMeta, ok := tmpReqMeta.(*ConvertingRequest)
	if !ok {
		return false, fmt.Errorf("metadata in txReq (%v) is not a converting request", txReq.Hash().String())
	}
	otaStr, txRandomStr := reqMeta.OTAStr, reqMeta.TxRandomStr
	recvPubKey, txRandom, err := coin.ParseOTAInfoFromString(otaStr, txRandomStr)
	if err != nil {
		return false, fmt.Errorf("cannot parse OTA params (%v, %v): %v", otaStr, txRandomStr, err)
	}
	if !bytes.Equal(recvPubKey.ToBytesS(), mintedCoin.GetPublicKey().ToBytesS()) {
		return false, fmt.Errorf("recvPubkey in txReq (%v) and public key of minted coin (%v) mismatch", recvPubKey.ToBytesS(), mintedCoin.GetPublicKey().ToBytesS())
	}
	if !bytes.Equal(txRandom.Bytes(), mintedCoin.GetTxRandom().Bytes()) {
		return false, fmt.Errorf("txRandom in txReq (%v) and txRandom of minted coin (%v) mismatch", txRandom, mintedCoin.GetTxRandom())
	}

	Logger.log.Infof("Finish ValidateTxResponse of pair req(%v)/resp(%v)\n", txReq.Hash().String(), txResp.Hash().String())

	return true, nil
}

