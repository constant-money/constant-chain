package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
)

type PDECrossPoolTradeResponse struct {
	MetadataBase
	TradeStatus   string
	RequestedTxID common.Hash
}

func NewPDECrossPoolTradeResponse(
	tradeStatus string,
	requestedTxID common.Hash,
	metaType int,
) *PDECrossPoolTradeResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDECrossPoolTradeResponse{
		TradeStatus:   tradeStatus,
		RequestedTxID: requestedTxID,
		MetadataBase:  metadataBase,
	}
}

func (iRes PDECrossPoolTradeResponse) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, db *statedb.StateDB) bool {
	// no need to have fee for this tx
	return true
}

func (iRes PDECrossPoolTradeResponse) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	// no need to validate tx with blockchain, just need to validate with requested tx (via RequestedTxID)
	return false, nil
}

func (iRes PDECrossPoolTradeResponse) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	return false, true, nil
}

func (iRes PDECrossPoolTradeResponse) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return iRes.Type == PDECrossPoolTradeResponseMeta
}

func (iRes PDECrossPoolTradeResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.TradeStatus
	record += iRes.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PDECrossPoolTradeResponse) CalculateSize() uint64 {
	return calculateSize(iRes)
}

func (iRes PDECrossPoolTradeResponse) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *MintData, shardID byte, tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever) (bool, error) {
	idx := -1

	for i, inst := range mintData.Insts  {
		if len(inst) < 4 { // this is not PDETradeRequest or PDECrossPoolTradeRequestMeta instruction
			continue
		}
		instMetaType := inst[0]
		if mintData.InstsUsed[i] > 0 ||
			instMetaType != strconv.Itoa(PDECrossPoolTradeRequestMeta) {
			continue
		}
		instTradeStatus := inst[2]
		if instTradeStatus != iRes.TradeStatus ||
			(instTradeStatus != common.PDECrossPoolTradeFeeRefundChainStatus &&
				instTradeStatus != common.PDECrossPoolTradeSellingTokenRefundChainStatus &&
				instTradeStatus != common.PDECrossPoolTradeAcceptedChainStatus) {
			continue
		}

		var shardIDFromInst byte
		var txReqIDFromInst common.Hash
		var receiverAddrStrFromInst string
		var receiverTxRandomFromInst string
		var receivingAmtFromInst uint64
		var receivingTokenIDStr string
		if instTradeStatus == common.PDECrossPoolTradeFeeRefundChainStatus ||
			instTradeStatus == common.PDECrossPoolTradeSellingTokenRefundChainStatus {
			contentBytes := []byte(inst[3])
			var pdeRefundCrossPoolTrade PDERefundCrossPoolTrade
			err := json.Unmarshal(contentBytes, &pdeRefundCrossPoolTrade)
			if err != nil {
				Logger.log.Error("WARNING - VALIDATION: an error occured while parsing pde refund cross pool trade content: ", err)
				continue
			}
			shardIDFromInst = pdeRefundCrossPoolTrade.ShardID
			txReqIDFromInst = pdeRefundCrossPoolTrade.TxReqID
			receiverAddrStrFromInst = pdeRefundCrossPoolTrade.TraderAddressStr
			receiverTxRandomFromInst = pdeRefundCrossPoolTrade.TxRandomStr
			receivingTokenIDStr = pdeRefundCrossPoolTrade.TokenIDStr
			receivingAmtFromInst = pdeRefundCrossPoolTrade.Amount
		} else { // trade accepted
			contentBytes := []byte(inst[3])
			var pdeCrossPoolTradeAcceptedContents []PDECrossPoolTradeAcceptedContent
			err := json.Unmarshal(contentBytes, &pdeCrossPoolTradeAcceptedContents)
			cLen := len(pdeCrossPoolTradeAcceptedContents)
			if err != nil || cLen == 0 {
				Logger.log.Error("WARNING - VALIDATION: an error occured while parsing pde cross pool trade accepted content: ", err)
				continue
			}
			lastPDETradeAcceptedContent := pdeCrossPoolTradeAcceptedContents[cLen-1]
			shardIDFromInst = lastPDETradeAcceptedContent.ShardID
			txReqIDFromInst = lastPDETradeAcceptedContent.RequestedTxID
			receiverAddrStrFromInst = lastPDETradeAcceptedContent.TraderAddressStr
			receiverTxRandomFromInst = lastPDETradeAcceptedContent.TxRandomStr
			receivingTokenIDStr = lastPDETradeAcceptedContent.TokenIDToBuyStr
			receivingAmtFromInst = lastPDETradeAcceptedContent.ReceiveAmount
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
		return false, fmt.Errorf(fmt.Sprintf("no PDECrossPoolTradeRequestMeta tx found for PDECrossPoolTradeResponse tx %s", tx.Hash().String()))
	}
	mintData.InstsUsed[idx] = 1
	return true, nil
}
