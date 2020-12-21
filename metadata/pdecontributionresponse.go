package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
)

type PDEContributionResponse struct {
	MetadataBase
	ContributionStatus string
	RequestedTxID      common.Hash
	TokenIDStr         string
	SharedRandom       []byte
}

func NewPDEContributionResponse(
	contributionStatus string,
	requestedTxID common.Hash,
	tokenIDStr string,
	metaType int,
) *PDEContributionResponse {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &PDEContributionResponse{
		ContributionStatus: contributionStatus,
		RequestedTxID:      requestedTxID,
		TokenIDStr:         tokenIDStr,
		MetadataBase:       metadataBase,
	}
}

func (iRes PDEContributionResponse) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, db *statedb.StateDB) bool {
	// no need to have fee for this tx
	return true
}

func (iRes PDEContributionResponse) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	// no need to validate tx with blockchain, just need to validate with requested tx (via RequestedTxID)
	return false, nil
}

func (iRes PDEContributionResponse) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	return false, true, nil
}

func (iRes PDEContributionResponse) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return iRes.Type == PDEContributionResponseMeta
}

func (iRes PDEContributionResponse) Hash() *common.Hash {
	record := iRes.RequestedTxID.String()
	record += iRes.TokenIDStr
	record += iRes.ContributionStatus
	record += iRes.MetadataBase.Hash().String()
	if iRes.SharedRandom != nil && len(iRes.SharedRandom) > 0 {
		record += string(iRes.SharedRandom)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PDEContributionResponse) CalculateSize() uint64 {
	return calculateSize(iRes)
}

func (iRes PDEContributionResponse) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *MintData, shardID byte, tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever) (bool, error) {
	idx := -1
	for i, inst := range mintData.Insts {
		if len(inst) < 4 { // this is not PDEContribution instruction
			continue
		}
		instMetaType := inst[0]
		if mintData.InstsUsed[i] > 0 ||
			instMetaType != strconv.Itoa(PDEContributionMeta) {
			continue
		}
		instContributionStatus := inst[2]
		if instContributionStatus != iRes.ContributionStatus || (instContributionStatus != common.PDEContributionRefundChainStatus && instContributionStatus != common.PDEContributionMatchedNReturnedChainStatus) {
			continue
		}

		var shardIDFromInst byte
		var txReqIDFromInst common.Hash
		var receiverAddrStrFromInst string
		var receivingAmtFromInst uint64
		var receivingTokenIDStr string

		if instContributionStatus == common.PDEContributionRefundChainStatus {
			contentBytes := []byte(inst[3])
			var refundContribution PDERefundContribution
			err := json.Unmarshal(contentBytes, &refundContribution)
			if err != nil {
				Logger.log.Error("WARNING - VALIDATION: an error occured while parsing refund contribution content: ", err)
				continue
			}
			shardIDFromInst = refundContribution.ShardID
			txReqIDFromInst = refundContribution.TxReqID
			receiverAddrStrFromInst = refundContribution.ContributorAddressStr
			receivingTokenIDStr = refundContribution.TokenIDStr
			receivingAmtFromInst = refundContribution.ContributedAmount

		} else { // matched and returned
			fmt.Println("[BUGLOG] Verify Metadata", tx.Hash().String(), iRes)
			contentBytes := []byte(inst[3])
			var matchedNReturnedContrib PDEMatchedNReturnedContribution
			err := json.Unmarshal(contentBytes, &matchedNReturnedContrib)
			if err != nil {
				Logger.log.Error("WARNING - VALIDATION: an error occured while parsing matched and returned contribution content: ", err)
				continue
			}
			shardIDFromInst = matchedNReturnedContrib.ShardID
			txReqIDFromInst = matchedNReturnedContrib.TxReqID
			receiverAddrStrFromInst = matchedNReturnedContrib.ContributorAddressStr
			receivingTokenIDStr = matchedNReturnedContrib.TokenIDStr
			receivingAmtFromInst = matchedNReturnedContrib.ReturnedContributedAmount
		}

		if !bytes.Equal(iRes.RequestedTxID[:], txReqIDFromInst[:]) ||
			shardID != shardIDFromInst {
			continue
		}

		key, err := wallet.Base58CheckDeserialize(receiverAddrStrFromInst)
		if err != nil {
			Logger.log.Info("WARNING - VALIDATION: an error occured while deserializing receiver address string: ", err)
			continue
		}

		isMinted, mintCoin, coinID, err := tx.GetTxMintData()
		if err != nil || !isMinted || coinID.String() != receivingTokenIDStr {
			continue
		}
		if ok := mintCoin.CheckCoinValid(key.KeySet.PaymentAddress, iRes.SharedRandom, receivingAmtFromInst); !ok {
			continue
		}

		idx = i
		fmt.Println("[BUGLOG] Verify Metadata --- OK")
		break
	}
	if idx == -1 { // not found the issuance request tx for this response
		return false, fmt.Errorf(fmt.Sprintf("no PDEContribution or PDEPRVRequiredContributionRequestMeta instruction found for PDEContributionResponse tx %s", tx.Hash().String()))
	}
	mintData.InstsUsed[idx] = 1
	return true, nil
}

func (iRes *PDEContributionResponse) SetSharedRandom(r []byte) {
	iRes.SharedRandom = r
}
