package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
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

func (iRes ConvertingResponse) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	return false, true, nil
}

func (iRes ConvertingResponse) ValidateMetadataByItself() bool {
	return true
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

