package metadata

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"reflect"
	"strconv"
)

type ConvertingRequest struct {
	OTAStr          string
	TxRandomStr     string
	TokenID         common.Hash
	ConvertedAmount uint64
	MetadataBase
}

type ConvertingRequestAction struct {
	Meta    ConvertingRequest
	TxReqID common.Hash
	ShardID byte
}

type ConvertingAcceptedContent struct {
	Address         string
	TxRandomStr     string
	ConvertedAmount uint64
	TokenID         common.Hash
	ShardID         byte
	RequestedTxID   common.Hash
}

func NewConvertingRequest(convertingAddress string, txRandomStr string, convertingAmount uint64, tokenID common.Hash, metaType int) (*ConvertingRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	pdeCrossPoolTradeRequest := &ConvertingRequest{
		OTAStr:          convertingAddress,
		TxRandomStr:     txRandomStr,
		TokenID:         tokenID,
		ConvertedAmount: convertingAmount,
	}
	pdeCrossPoolTradeRequest.MetadataBase = metadataBase
	return pdeCrossPoolTradeRequest, nil
}

func (req ConvertingRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	return true, nil
}

//ValidateSanityData performs the following verifications:
//	1. Check if the convertedAmount is not zero
//	2. Check tx type and supported tokenID
//	3. Check the burnedToken and convertedToken are the same
//	4. Check if the burnedAmount equals the convertedAmount
func (req ConvertingRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	if tx.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(tx).String() == "*transaction.Tx" {
		return true, true, nil
	}

	if req.ConvertedAmount == 0 {
		return false, false, NewMetadataTxError(ConvertingAmountError, fmt.Errorf("amount of a converting request cannot be 0"))
	}

	if tx.GetType() == common.TxConversionType && req.TokenID.String() != common.PRVIDStr {
		return false, false, NewMetadataTxError(ConvertingTokenIDError, fmt.Errorf("cannot convert token %v in a PRV transaction", req.TokenID.String()))
	}

	if tx.GetType() == common.TxTokenConversionType && req.TokenID.String() == common.PRVIDStr {
		return false, false, NewMetadataTxError(ConvertingTokenIDError, fmt.Errorf("cannot convert PRV in a token transaction"))
	}

	isBurned, burnedCoin, burnedTokenID, err := tx.GetTxBurnData()
	if err != nil || !isBurned {
		if err != nil {
			Logger.log.Errorf("not a burn transaction: %v\n", err)
		}
		return false, false, fmt.Errorf("not a burn transaction: %v", err)
	}

	if burnedTokenID.String() != req.TokenID.String() {
		return false, false, NewMetadataTxError(ConvertingTokenIDError, fmt.Errorf("burnedToken and convertedToken mismatch: %v != %v", burnedTokenID.String(), req.TokenID.String()))
	}

	if burnedCoin.GetValue() != req.ConvertedAmount {
		return false, false, NewMetadataTxError(ConvertingTokenIDError, fmt.Errorf("burnedAmount and convertedAmount mismatch: %v != %v", burnedCoin.GetValue(), req.ConvertedAmount))
	}

	return true, true, nil
}

func (req ConvertingRequest) ValidateMetadataByItself() bool {
	return req.Type == PDECrossPoolTradeRequestMeta
}

func (req ConvertingRequest) Hash() *common.Hash {
	record := req.MetadataBase.Hash().String()
	record += req.TokenID.String()
	record += req.OTAStr
	record += req.TxRandomStr
	record += strconv.FormatUint(req.ConvertedAmount, 10)

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (req *ConvertingRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	actionContent := ConvertingRequestAction{
		Meta:    *req,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}

	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(req.Type), actionContentBase64Str}
	return [][]string{action}, nil
}

func (req *ConvertingRequest) CalculateSize() uint64 {
	return calculateSize(req)
}
