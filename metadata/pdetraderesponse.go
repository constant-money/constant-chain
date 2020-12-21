package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
)

type PDETradeResponse struct {
	MetadataBase
	TradeStatus   string
	RequestedTxID common.Hash
}

func NewPDETradeResponse(
	tradeStatus string,
	requestedTxID common.Hash,
	metaType int,
) *PDETradeResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDETradeResponse{
		TradeStatus:   tradeStatus,
		RequestedTxID: requestedTxID,
		MetadataBase:  metadataBase,
	}
}

func (iRes PDETradeResponse) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, db *statedb.StateDB) bool {
	// no need to have fee for this tx
	return true
}

func (iRes PDETradeResponse) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	// no need to validate tx with blockchain, just need to validate with requested tx (via RequestedTxID)
	return false, nil
}

func (iRes PDETradeResponse) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	return false, true, nil
}

func (iRes PDETradeResponse) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return iRes.Type == PDETradeResponseMeta
}

func (iRes PDETradeResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.TradeStatus
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PDETradeResponse) CalculateSize() uint64 {
	return calculateSize(iRes)
}

func (iRes PDETradeResponse) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *MintData, shardID byte, tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever) (bool, error) {
	idx := -1

	for i, inst := range mintData.Insts {
		if len(inst) < 4 { // this is not PDETradeRequest instruction
			continue
		}
		instMetaType := inst[0]
		if mintData.InstsUsed[i] > 0 ||
			instMetaType != strconv.Itoa(PDETradeRequestMeta) {
			continue
		}
		instTradeStatus := inst[2]
		if instTradeStatus != iRes.TradeStatus || (instTradeStatus != common.PDETradeRefundChainStatus && instTradeStatus != common.PDETradeAcceptedChainStatus) {
			continue
		}

		var shardIDFromInst byte
		var txReqIDFromInst common.Hash
		var receiverAddrStrFromInst string
		var receiverTxRandomFromInst string
		var receivingAmtFromInst uint64
		var receivingTokenIDStr string
		if instTradeStatus == common.PDETradeRefundChainStatus {
			contentBytes, err := base64.StdEncoding.DecodeString(inst[3])
			if err != nil {
				Logger.log.Error("WARNING - VALIDATION: an error occured while parsing instruction content: ", err)
				continue
			}
			var pdeTradeRequestAction PDETradeRequestAction
			err = json.Unmarshal(contentBytes, &pdeTradeRequestAction)
			if err != nil {
				Logger.log.Error("WARNING - VALIDATION: an error occured while parsing instruction content: ", err)
				continue
			}
			shardIDFromInst = pdeTradeRequestAction.ShardID
			txReqIDFromInst = pdeTradeRequestAction.TxReqID
			receiverAddrStrFromInst = pdeTradeRequestAction.Meta.TraderAddressStr
			receiverTxRandomFromInst = pdeTradeRequestAction.Meta.TxRandomStr
			receivingTokenIDStr = pdeTradeRequestAction.Meta.TokenIDToSellStr
			receivingAmtFromInst = pdeTradeRequestAction.Meta.SellAmount + pdeTradeRequestAction.Meta.TradingFee
		} else { // trade accepted
			contentBytes := []byte(inst[3])
			var pdeTradeAcceptedContent PDETradeAcceptedContent
			err := json.Unmarshal(contentBytes, &pdeTradeAcceptedContent)
			if err != nil {
				Logger.log.Error("WARNING - VALIDATION: an error occured while parsing instruction content: ", err)
				continue
			}
			shardIDFromInst = pdeTradeAcceptedContent.ShardID
			txReqIDFromInst = pdeTradeAcceptedContent.RequestedTxID
			receiverAddrStrFromInst = pdeTradeAcceptedContent.TraderAddressStr
			receiverTxRandomFromInst = pdeTradeAcceptedContent.TxRandomStr
			receivingTokenIDStr = pdeTradeAcceptedContent.TokenIDToBuyStr
			receivingAmtFromInst = pdeTradeAcceptedContent.ReceiveAmount
		}

		if !bytes.Equal(iRes.RequestedTxID[:], txReqIDFromInst[:]) ||
			shardID != shardIDFromInst {
			continue
		}

		isMinted, mintCoin, assetID, err := tx.GetTxMintData()
		if err != nil {
			Logger.log.Error("ERROR - VALIDATION: an error occured while get tx mint data: ", err)
			continue
		}
		if !isMinted {
			Logger.log.Info("WARNING - VALIDATION: this is not Tx Mint: ")
			continue
		}
		pk := mintCoin.GetPublicKey().ToBytesS()

		paidAmount := mintCoin.GetValue()
		if len(receiverTxRandomFromInst) > 0 {
			publicKey, txRandom, err := coin.ParseOTAInfoFromString(receiverAddrStrFromInst, receiverTxRandomFromInst)
			if err != nil {
				Logger.log.Errorf("Wrong request info's txRandom - Cannot set txRandom from bytes: %+v", err)
				continue
			}

			txR := mintCoin.(*coin.CoinV2).GetTxRandom()
			if !bytes.Equal(publicKey.ToBytesS(), pk[:]) ||
				receivingAmtFromInst != paidAmount ||
				!bytes.Equal(txR[:], txRandom[:]) ||
				receivingTokenIDStr != assetID.String() {
				continue
			}
		} else {
			key, err := wallet.Base58CheckDeserialize(receiverAddrStrFromInst)
			if err != nil {
				Logger.log.Info("WARNING - VALIDATION: an error occured while deserializing receiver address string: ", err)
				continue
			}

			if !bytes.Equal(key.KeySet.PaymentAddress.Pk[:], pk[:]) ||
				receivingAmtFromInst != paidAmount ||
				receivingTokenIDStr != assetID.String() {
				continue
			}
		}
		idx = i
		break
	}
	if idx == -1 { // not found the issuance request tx for this response
		return false, fmt.Errorf(fmt.Sprintf("no PDETradeRequest or PDECrossPoolTradeRequestMeta tx found for PDETradeResponse tx %s", tx.Hash().String()))
	}
	mintData.InstsUsed[idx] = 1
	return true, nil
}
