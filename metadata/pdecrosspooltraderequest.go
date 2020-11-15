package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wallet"
)

// TODO: Update error type to correct one
// PDECrossPoolTradeRequest - privacy dex cross pool trade
type PDECrossPoolTradeRequest struct {
	TokenIDToBuyStr     string
	TokenIDToSellStr    string
	SellAmount          uint64 // must be equal to vout value
	MinAcceptableAmount uint64
	TradingFee          uint64
	TraderAddressStr    string
	MetadataBase
}

type PDECrossPoolTradeRequestAction struct {
	Meta    PDECrossPoolTradeRequest
	TxReqID common.Hash
	ShardID byte
}

type PDECrossPoolTradeAcceptedContent struct {
	TraderAddressStr         string
	TokenIDToBuyStr          string
	ReceiveAmount            uint64
	Token1IDStr              string
	Token2IDStr              string
	Token1PoolValueOperation TokenPoolValueOperation
	Token2PoolValueOperation TokenPoolValueOperation
	ShardID                  byte
	RequestedTxID            common.Hash
	AddingFee                uint64
}

type PDERefundCrossPoolTrade struct {
	TraderAddressStr string
	TokenIDStr       string
	Amount           uint64
	ShardID          byte
	TxReqID          common.Hash
}

func NewPDECrossPoolTradeRequest(
	tokenIDToBuyStr string,
	tokenIDToSellStr string,
	sellAmount uint64,
	minAcceptableAmount uint64,
	tradingFee uint64,
	traderAddressStr string,
	metaType int,
) (*PDECrossPoolTradeRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	pdeCrossPoolTradeRequest := &PDECrossPoolTradeRequest{
		TokenIDToBuyStr:     tokenIDToBuyStr,
		TokenIDToSellStr:    tokenIDToSellStr,
		SellAmount:          sellAmount,
		MinAcceptableAmount: minAcceptableAmount,
		TradingFee:          tradingFee,
		TraderAddressStr:    traderAddressStr,
	}
	pdeCrossPoolTradeRequest.MetadataBase = metadataBase
	return pdeCrossPoolTradeRequest, nil
}

func (pc PDECrossPoolTradeRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	// NOTE: verify supported tokens pair as needed
	return true, nil
}

func (pc PDECrossPoolTradeRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	if tx.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(tx).String() == "*transaction.Tx" {
		return true, true, nil
	}

	keyWallet, err := wallet.Base58CheckDeserialize(pc.TraderAddressStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TraderAddressStr incorrect"))
	}
	traderAddr := keyWallet.KeySet.PaymentAddress

	if len(traderAddr.Pk) == 0 {
		return false, false, errors.New("Wrong request info's trader address")
	}

	if !bytes.Equal(tx.GetSigPubKey()[:], traderAddr.Pk[:]) {
		return false, false, errors.New("TraderAddress incorrect")
	}

	_, err = common.Hash{}.NewHashFromStr(pc.TokenIDToBuyStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenIDToBuyStr incorrect"))
	}

	if pc.TokenIDToSellStr == pc.TokenIDToBuyStr {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenIDToSellStr should be different from TokenIDToBuyStr"))
	}

	if tx.GetType() == common.TxNormalType {
		if pc.TokenIDToSellStr != common.PRVCoinID.String() {
			return false, false, errors.New("With tx normal privacy, the tokenIDStr should be PRV, not custom token")
		}
		if !tx.IsCoinsBurning(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight) {
			return false, false, errors.New("Must send coin to burning address")
		}
		txValue := tx.CalculateTxValue()
		if pc.SellAmount > txValue || pc.TradingFee > txValue {
			return false, false, errors.New("Neither selling amount nor trading fee allows to be larger than the tx value")
		}
		if (pc.SellAmount + pc.TradingFee) != txValue {
			return false, false, errors.New("Total of selling amount and trading fee should be equal to the tx value")
		}
	}

	if tx.GetType() == common.TxCustomTokenPrivacyType {
		if pc.TokenIDToSellStr == common.PRVCoinID.String() {
			return false, false, errors.New("With custom token privacy tx, the tokenIDStr should not be PRV, but custom token")
		}
		tokenIDToSell, err := common.Hash{}.NewHashFromStr(pc.TokenIDToSellStr)
		if err != nil {
			return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenIDToSellStr incorrect"))
		}
		if !bytes.Equal(tx.GetTokenID()[:], tokenIDToSell[:]) {
			return false, false, errors.New("Wrong request info's token id, it should be equal to tx's token id")
		}

		if pc.TradingFee == 0 {
			if !tx.IsCoinsBurning(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight) {
				return false, false, errors.New("Must send custom coin to burning address")
			}
			pTokenAmt := tx.CalculateTxValue()
			if pTokenAmt != pc.SellAmount {
				return false, false, errors.New("Sell amount should be equal to the burned pToken amount")
			}
		} else {
			if !tx.IsFullBurning(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight) {
				return false, false, errors.New("Must send coins to burning address")
			}
			prvAmt, pTokenAmt := tx.GetFullTxValues()
			if prvAmt != pc.TradingFee {
				return false, false, errors.New("Trading fee should be equal to the burned prv amount")
			}
			if pTokenAmt != pc.SellAmount {
				return false, false, errors.New("Sell amount should be equal to the burned pToken amount")
			}
		}
	}

	return true, true, nil
}

func (pc PDECrossPoolTradeRequest) ValidateMetadataByItself() bool {
	return pc.Type == PDECrossPoolTradeRequestMeta
}

func (pc PDECrossPoolTradeRequest) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.TokenIDToBuyStr
	record += pc.TokenIDToSellStr
	record += pc.TraderAddressStr
	record += strconv.FormatUint(pc.SellAmount, 10)
	record += strconv.FormatUint(pc.MinAcceptableAmount, 10)
	record += strconv.FormatUint(pc.TradingFee, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc *PDECrossPoolTradeRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PDECrossPoolTradeRequestAction{
		Meta:    *pc,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(pc.Type), actionContentBase64Str}
	return [][]string{action}, nil
}

func (pc *PDECrossPoolTradeRequest) CalculateSize() uint64 {
	return calculateSize(pc)
}
