package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
)

func buildWaitingContributionInst(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
) []string {
	waitingContribution := metadata.PDEWaitingContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
		TxReqID:               txReqID,
	}
	waitingContributionBytes, _ := json.Marshal(waitingContribution)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDEContributionWaitingChainStatus,
		string(waitingContributionBytes),
	}
}

func buildRefundContributionInst(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
) []string {
	refundContribution := metadata.PDERefundContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
		TxReqID:               txReqID,
		ShardID:               shardID,
	}
	refundContributionBytes, _ := json.Marshal(refundContribution)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDEContributionRefundChainStatus,
		string(refundContributionBytes),
	}
}

func buildMatchedContributionInst(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
) []string {
	matchedContribution := metadata.PDEMatchedContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
		TxReqID:               txReqID,
	}
	matchedContributionBytes, _ := json.Marshal(matchedContribution)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDEContributionMatchedChainStatus,
		string(matchedContributionBytes),
	}
}

func buildMatchedNReturnedContributionInst(
	pdeContributionPairID string,
	contributorAddressStr string,
	actualContributedAmount uint64,
	returnedContributedAmount uint64,
	tokenIDStr string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	actualWaitingContribAmount uint64,
) []string {
	matchedNReturnedContribution := metadata.PDEMatchedNReturnedContribution{
		PDEContributionPairID:      pdeContributionPairID,
		ContributorAddressStr:      contributorAddressStr,
		ActualContributedAmount:    actualContributedAmount,
		ReturnedContributedAmount:  returnedContributedAmount,
		TokenIDStr:                 tokenIDStr,
		ShardID:                    shardID,
		TxReqID:                    txReqID,
		ActualWaitingContribAmount: actualWaitingContribAmount,
	}
	matchedNReturnedContribBytes, _ := json.Marshal(matchedNReturnedContribution)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDEContributionMatchedNReturnedChainStatus,
		string(matchedNReturnedContribBytes),
	}
}

func isRightRatio(
	waitingContribution1 *rawdbv2.PDEContribution,
	waitingContribution2 *rawdbv2.PDEContribution,
	poolPair *rawdbv2.PDEPoolForPair,
) bool {
	if poolPair == nil {
		return true
	}
	if poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
		return true
	}
	if waitingContribution1.TokenIDStr == poolPair.Token1IDStr {
		expectedContribAmt := big.NewInt(0)
		expectedContribAmt.Mul(
			new(big.Int).SetUint64(waitingContribution1.Amount),
			new(big.Int).SetUint64(poolPair.Token2PoolValue),
		)
		expectedContribAmt.Div(
			expectedContribAmt,
			new(big.Int).SetUint64(poolPair.Token1PoolValue),
		)
		return expectedContribAmt.Uint64() == waitingContribution2.Amount
	}
	if waitingContribution1.TokenIDStr == poolPair.Token2IDStr {
		expectedContribAmt := big.NewInt(0)
		expectedContribAmt.Mul(
			new(big.Int).SetUint64(waitingContribution1.Amount),
			new(big.Int).SetUint64(poolPair.Token1PoolValue),
		)
		expectedContribAmt.Div(
			expectedContribAmt,
			new(big.Int).SetUint64(poolPair.Token2PoolValue),
		)
		return expectedContribAmt.Uint64() == waitingContribution2.Amount
	}
	return false
}

func computeActualContributedAmounts(
	waitingContribution1 *rawdbv2.PDEContribution,
	waitingContribution2 *rawdbv2.PDEContribution,
	poolPair *rawdbv2.PDEPoolForPair,
) (uint64, uint64, uint64, uint64) {
	if poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
		return waitingContribution1.Amount, 0, waitingContribution2.Amount, 0
	}
	if poolPair.Token1IDStr == waitingContribution1.TokenIDStr {
		// waitingAmtTemp = waitingContribution2.Amount * poolPair.Token1PoolValue / poolPair.Token2PoolValue
		contribution1Amt := big.NewInt(0)
		tempAmt := big.NewInt(0)
		tempAmt.Mul(
			new(big.Int).SetUint64(waitingContribution2.Amount),
			new(big.Int).SetUint64(poolPair.Token1PoolValue),
		)
		tempAmt.Div(
			tempAmt,
			new(big.Int).SetUint64(poolPair.Token2PoolValue),
		)
		if tempAmt.Uint64() > waitingContribution1.Amount {
			contribution1Amt = new(big.Int).SetUint64(waitingContribution1.Amount)
		} else {
			contribution1Amt = tempAmt
		}
		contribution2Amt := big.NewInt(0)
		contribution2Amt.Mul(
			contribution1Amt,
			new(big.Int).SetUint64(poolPair.Token2PoolValue),
		)
		contribution2Amt.Div(
			contribution2Amt,
			new(big.Int).SetUint64(poolPair.Token1PoolValue),
		)
		actualContribution1Amt := contribution1Amt.Uint64()
		actualContribution2Amt := contribution2Amt.Uint64()
		return actualContribution1Amt, waitingContribution1.Amount - actualContribution1Amt, actualContribution2Amt, waitingContribution2.Amount - actualContribution2Amt
	}
	if poolPair.Token1IDStr == waitingContribution2.TokenIDStr {
		// tempAmt = waitingContribution2.Amount * poolPair.Token1PoolValue / poolPair.Token2PoolValue
		contribution2Amt := big.NewInt(0)
		tempAmt := big.NewInt(0)
		tempAmt.Mul(
			new(big.Int).SetUint64(waitingContribution1.Amount),
			new(big.Int).SetUint64(poolPair.Token1PoolValue),
		)
		tempAmt.Div(
			tempAmt,
			new(big.Int).SetUint64(poolPair.Token2PoolValue),
		)
		if tempAmt.Uint64() > waitingContribution2.Amount {
			contribution2Amt = new(big.Int).SetUint64(waitingContribution2.Amount)
		} else {
			contribution2Amt = tempAmt
		}
		contribution1Amt := big.NewInt(0)
		contribution1Amt.Mul(
			contribution2Amt,
			new(big.Int).SetUint64(poolPair.Token2PoolValue),
		)
		contribution1Amt.Div(
			contribution1Amt,
			new(big.Int).SetUint64(poolPair.Token1PoolValue),
		)
		actualContribution1Amt := contribution1Amt.Uint64()
		actualContribution2Amt := contribution2Amt.Uint64()
		return actualContribution1Amt, waitingContribution1.Amount - actualContribution1Amt, actualContribution2Amt, waitingContribution2.Amount - actualContribution2Amt
	}
	return 0, 0, 0, 0
}

func (blockchain *BlockChain) buildInstructionsForPDEContribution(
	contentStr string,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
	isPRVRequired bool,
) ([][]string, error) {
	if currentPDEState == nil {
		Logger.log.Warn("WARN - [buildInstructionsForPDEContribution]: Current PDE state is null.")
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			common.PDEContributionRefundChainStatus,
			contentStr,
		}
		return [][]string{inst}, nil
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde withdrawal action: %+v", err)
		return [][]string{}, nil
	}
	var pdeContributionAction metadata.PDEContributionAction
	err = json.Unmarshal(contentBytes, &pdeContributionAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde contribution action: %+v", err)
		return [][]string{}, nil
	}
	meta := pdeContributionAction.Meta
	waitingContribPairKey := string(rawdbv2.BuildWaitingPDEContributionKey(beaconHeight, meta.PDEContributionPairID))
	waitingContribution, found := currentPDEState.WaitingPDEContributions[waitingContribPairKey]
	if !found || waitingContribution == nil {
		currentPDEState.WaitingPDEContributions[waitingContribPairKey] = &rawdbv2.PDEContribution{
			ContributorAddressStr: meta.ContributorAddressStr,
			TokenIDStr:            meta.TokenIDStr,
			Amount:                meta.ContributedAmount,
			TxReqID:               pdeContributionAction.TxReqID,
		}
		inst := buildWaitingContributionInst(
			meta.PDEContributionPairID,
			meta.ContributorAddressStr,
			meta.ContributedAmount,
			meta.TokenIDStr,
			metaType,
			shardID,
			pdeContributionAction.TxReqID,
		)
		return [][]string{inst}, nil
	}
	if waitingContribution.TokenIDStr == meta.TokenIDStr ||
		waitingContribution.ContributorAddressStr != meta.ContributorAddressStr ||
		(isPRVRequired && waitingContribution.TokenIDStr != common.PRVIDStr && meta.TokenIDStr != common.PRVIDStr) {
		delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		refundInst1 := buildRefundContributionInst(
			meta.PDEContributionPairID,
			meta.ContributorAddressStr,
			meta.ContributedAmount,
			meta.TokenIDStr,
			metaType,
			shardID,
			pdeContributionAction.TxReqID,
		)
		refundInst2 := buildRefundContributionInst(
			meta.PDEContributionPairID,
			waitingContribution.ContributorAddressStr,
			waitingContribution.Amount,
			waitingContribution.TokenIDStr,
			metaType,
			shardID,
			waitingContribution.TxReqID,
		)
		return [][]string{refundInst1, refundInst2}, nil
	}
	// contributed to 2 diff sides of a pair and its a first contribution of this pair
	poolPairs := currentPDEState.PDEPoolPairs
	poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, waitingContribution.TokenIDStr, meta.TokenIDStr))
	poolPair, found := poolPairs[poolPairKey]
	incomingWaitingContribution := &rawdbv2.PDEContribution{
		ContributorAddressStr: meta.ContributorAddressStr,
		TokenIDStr:            meta.TokenIDStr,
		Amount:                meta.ContributedAmount,
		TxReqID:               pdeContributionAction.TxReqID,
	}

	if !found || poolPair == nil {
		delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		updateWaitingContributionPairToPoolV2(
			beaconHeight,
			waitingContribution,
			incomingWaitingContribution,
			currentPDEState,
		)
		matchedInst := buildMatchedContributionInst(
			meta.PDEContributionPairID,
			meta.ContributorAddressStr,
			meta.ContributedAmount,
			meta.TokenIDStr,
			metaType,
			shardID,
			pdeContributionAction.TxReqID,
		)
		return [][]string{matchedInst}, nil
	}

	// isRightRatio(waitingContribution, incomingWaitingContribution, poolPair)
	actualWaitingContribAmt, returnedWaitingContribAmt, actualIncomingWaitingContribAmt, returnedIncomingWaitingContribAmt := computeActualContributedAmounts(
		waitingContribution,
		incomingWaitingContribution,
		poolPair,
	)
	if actualWaitingContribAmt == 0 || actualIncomingWaitingContribAmt == 0 {
		delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		refundInst1 := buildRefundContributionInst(
			meta.PDEContributionPairID,
			meta.ContributorAddressStr,
			meta.ContributedAmount,
			meta.TokenIDStr,
			metaType,
			shardID,
			pdeContributionAction.TxReqID,
		)
		refundInst2 := buildRefundContributionInst(
			meta.PDEContributionPairID,
			waitingContribution.ContributorAddressStr,
			waitingContribution.Amount,
			waitingContribution.TokenIDStr,
			metaType,
			shardID,
			waitingContribution.TxReqID,
		)
		return [][]string{refundInst1, refundInst2}, nil
	}

	delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
	actualWaitingContrib := &rawdbv2.PDEContribution{
		ContributorAddressStr: waitingContribution.ContributorAddressStr,
		TokenIDStr:            waitingContribution.TokenIDStr,
		Amount:                actualWaitingContribAmt,
		TxReqID:               waitingContribution.TxReqID,
	}
	actualIncomingWaitingContrib := &rawdbv2.PDEContribution{
		ContributorAddressStr: meta.ContributorAddressStr,
		TokenIDStr:            meta.TokenIDStr,
		Amount:                actualIncomingWaitingContribAmt,
		TxReqID:               pdeContributionAction.TxReqID,
	}
	updateWaitingContributionPairToPoolV2(
		beaconHeight,
		actualWaitingContrib,
		actualIncomingWaitingContrib,
		currentPDEState,
	)
	matchedNReturnedInst1 := buildMatchedNReturnedContributionInst(
		meta.PDEContributionPairID,
		meta.ContributorAddressStr,
		actualIncomingWaitingContribAmt,
		returnedIncomingWaitingContribAmt,
		meta.TokenIDStr,
		metaType,
		shardID,
		pdeContributionAction.TxReqID,
		actualWaitingContribAmt,
	)
	matchedNReturnedInst2 := buildMatchedNReturnedContributionInst(
		meta.PDEContributionPairID,
		waitingContribution.ContributorAddressStr,
		actualWaitingContribAmt,
		returnedWaitingContribAmt,
		waitingContribution.TokenIDStr,
		metaType,
		shardID,
		waitingContribution.TxReqID,
		0,
	)
	return [][]string{matchedNReturnedInst1, matchedNReturnedInst2}, nil
}

type tradeInfo struct {
	tokenIDToBuyStr         string
	tokenIDToSellStr        string
	sellAmount              uint64
	newTokenPoolValueToBuy  uint64
	newTokenPoolValueToSell uint64
	receiveAmount           uint64
}

func (blockchain *BlockChain) buildInstsForSortedTradableActions(
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
	sortedTradableActions []metadata.PDECrossPoolTradeRequestAction,
) ([][]string, map[string]uint64) {
	prvIDStr := common.PRVCoinID.String()
	tradableInsts := [][]string{}
	tradingFeeByPair := make(map[string]uint64)
	for _, tradeAction := range sortedTradableActions {
		tradeMeta := tradeAction.Meta
		var sequentialTrades []*tradeInfo
		if isTradingFairContainsPRV(tradeMeta.TokenIDToSellStr, tradeMeta.TokenIDToBuyStr) { // direct trade
			sequentialTrades = []*tradeInfo{
				&tradeInfo{
					tokenIDToBuyStr:  tradeMeta.TokenIDToBuyStr,
					tokenIDToSellStr: tradeMeta.TokenIDToSellStr,
					sellAmount:       tradeMeta.SellAmount,
				},
			}
		} else { // cross pool trade
			sequentialTrades = []*tradeInfo{
				&tradeInfo{
					tokenIDToBuyStr:  prvIDStr,
					tokenIDToSellStr: tradeMeta.TokenIDToSellStr,
					sellAmount:       tradeMeta.SellAmount,
				},
				&tradeInfo{
					tokenIDToBuyStr:  tradeMeta.TokenIDToBuyStr,
					tokenIDToSellStr: prvIDStr,
					sellAmount:       uint64(0),
				},
			}
		}
		newInsts, err := blockchain.buildInstructionsForPDECrossPoolTrade(
			sequentialTrades,
			tradeMeta.MinAcceptableAmount,
			tradeMeta.TradingFee,
			tradeAction.ShardID,
			metadata.PDECrossPoolTradeRequestMeta,
			currentPDEState,
			beaconHeight,
			tradeAction.Meta.TraderAddressStr,
			tradeAction.Meta.TxRandomStr,
			tradeAction.TxReqID,
			tradingFeeByPair,
		)
		if err != nil {
			Logger.log.Error(err)
			continue
		}
		if len(newInsts) > 0 {
			tradableInsts = append(tradableInsts, newInsts...)
		}
	}
	return tradableInsts, tradingFeeByPair
}

func (blockchain *BlockChain) buildInstsForUntradableActions(
	untradableActions []metadata.PDECrossPoolTradeRequestAction,
) [][]string {
	untradableInsts := [][]string{}
	for _, tradeAction := range untradableActions {
		refundTradingFeeInst := buildCrossPoolTradeRefundInst(
			tradeAction.Meta.TraderAddressStr,
			common.PRVCoinID.String(),
			tradeAction.Meta.TradingFee,
			metadata.PDECrossPoolTradeRequestMeta,
			common.PDECrossPoolTradeFeeRefundChainStatus,
			tradeAction.ShardID,
			tradeAction.TxReqID,
		)
		untradableInsts = append(untradableInsts, refundTradingFeeInst)
		refundSellingTokenInst := buildCrossPoolTradeRefundInst(
			tradeAction.Meta.TraderAddressStr,
			tradeAction.Meta.TokenIDToSellStr,
			tradeAction.Meta.SellAmount,
			metadata.PDECrossPoolTradeRequestMeta,
			common.PDECrossPoolTradeSellingTokenRefundChainStatus,
			tradeAction.ShardID,
			tradeAction.TxReqID,
		)
		untradableInsts = append(untradableInsts, refundSellingTokenInst)
	}
	return untradableInsts
}

func buildCrossPoolTradeRefundInst(
	traderAddressStr string,
	tokenIDStr string,
	amount uint64,
	metaType int,
	status string,
	shardID byte,
	txReqID common.Hash,
) []string {
	refundCrossPoolTrade := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: traderAddressStr,
		TokenIDStr:       tokenIDStr,
		Amount:           amount,
		ShardID:          shardID,
		TxReqID:          txReqID,
	}
	refundCrossPoolTradeBytes, _ := json.Marshal(refundCrossPoolTrade)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(refundCrossPoolTradeBytes),
	}
}

func (blockchain *BlockChain) buildInstructionsForPDECrossPoolTrade(
	sequentialTrades []*tradeInfo,
	minAcceptableAmount uint64,
	tradingFee uint64,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
	traderAddressStr string,
	txRandomStr string,
	txReqID common.Hash,
	tradingFeeByPair map[string]uint64,
) ([][]string, error) {
	if currentPDEState == nil ||
		(currentPDEState.PDEPoolPairs == nil || len(currentPDEState.PDEPoolPairs) == 0) {
		refundTradingFeeInst := buildCrossPoolTradeRefundInst(
			traderAddressStr,
			common.PRVCoinID.String(),
			tradingFee,
			metaType,
			common.PDECrossPoolTradeFeeRefundChainStatus,
			shardID,
			txReqID,
		)
		refundSellingTokenInst := buildCrossPoolTradeRefundInst(
			traderAddressStr,
			sequentialTrades[0].tokenIDToSellStr,
			sequentialTrades[0].sellAmount,
			metaType,
			common.PDECrossPoolTradeSellingTokenRefundChainStatus,
			shardID,
			txReqID,
		)
		return [][]string{refundTradingFeeInst, refundSellingTokenInst}, nil
	}

	amt := sequentialTrades[0].sellAmount
	for _, tradeInf := range sequentialTrades {
		tradeInf.sellAmount = amt
		pairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, tradeInf.tokenIDToBuyStr, tradeInf.tokenIDToSellStr))
		pdePoolPair, _ := currentPDEState.PDEPoolPairs[pairKey]
		newAmt, newTokenPoolValueToBuy, newTokenPoolValueToSell := calcTradeValue(pdePoolPair, tradeInf.tokenIDToSellStr, amt)
		amt = newAmt
		tradeInf.newTokenPoolValueToBuy = newTokenPoolValueToBuy
		tradeInf.newTokenPoolValueToSell = newTokenPoolValueToSell
		tradeInf.receiveAmount = amt
	}

	if minAcceptableAmount > amt {
		refundTradingFeeInst := buildCrossPoolTradeRefundInst(
			traderAddressStr,
			common.PRVCoinID.String(),
			tradingFee,
			metaType,
			common.PDECrossPoolTradeFeeRefundChainStatus,
			shardID,
			txReqID,
		)
		refundSellingTokenInst := buildCrossPoolTradeRefundInst(
			traderAddressStr,
			sequentialTrades[0].tokenIDToSellStr,
			sequentialTrades[0].sellAmount,
			metaType,
			common.PDECrossPoolTradeSellingTokenRefundChainStatus,
			shardID,
			txReqID,
		)
		return [][]string{refundTradingFeeInst, refundSellingTokenInst}, nil
	}

	tradeAcceptedContents := []metadata.PDECrossPoolTradeAcceptedContent{}
	proportionalFee := tradingFee / uint64(len(sequentialTrades))
	for idx, tradeInf := range sequentialTrades {
		// update current pde state on mem
		pairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, tradeInf.tokenIDToBuyStr, tradeInf.tokenIDToSellStr))
		pdePoolPair, _ := currentPDEState.PDEPoolPairs[pairKey]

		pdePoolPair.Token1PoolValue = tradeInf.newTokenPoolValueToBuy
		pdePoolPair.Token2PoolValue = tradeInf.newTokenPoolValueToSell
		if pdePoolPair.Token1IDStr == tradeInf.tokenIDToSellStr {
			pdePoolPair.Token1PoolValue = tradeInf.newTokenPoolValueToSell
			pdePoolPair.Token2PoolValue = tradeInf.newTokenPoolValueToBuy
		}

		// build trade accepted contents
		pdeTradeAcceptedContent := metadata.PDECrossPoolTradeAcceptedContent{
			TraderAddressStr: traderAddressStr,
			TxRandomStr:      txRandomStr,
			TokenIDToBuyStr:  tradeInf.tokenIDToBuyStr,
			ReceiveAmount:    tradeInf.receiveAmount,
			Token1IDStr:      pdePoolPair.Token1IDStr,
			Token2IDStr:      pdePoolPair.Token2IDStr,
			ShardID:          shardID,
			RequestedTxID:    txReqID,
		}
		pdeTradeAcceptedContent.Token1PoolValueOperation = metadata.TokenPoolValueOperation{
			Operator: "-",
			Value:    tradeInf.receiveAmount,
		}
		pdeTradeAcceptedContent.Token2PoolValueOperation = metadata.TokenPoolValueOperation{
			Operator: "+",
			Value:    tradeInf.sellAmount,
		}
		if pdePoolPair.Token1IDStr == tradeInf.tokenIDToSellStr {
			pdeTradeAcceptedContent.Token1PoolValueOperation = metadata.TokenPoolValueOperation{
				Operator: "+",
				Value:    tradeInf.sellAmount,
			}
			pdeTradeAcceptedContent.Token2PoolValueOperation = metadata.TokenPoolValueOperation{
				Operator: "-",
				Value:    tradeInf.receiveAmount,
			}
		}

		addingFee := proportionalFee
		if idx == len(sequentialTrades)-1 {
			addingFee = tradingFee - uint64(len(sequentialTrades)-1)*proportionalFee
		}
		pdeTradeAcceptedContent.AddingFee = addingFee
		sKey := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, tradeInf.tokenIDToBuyStr, tradeInf.tokenIDToSellStr, ""))
		tradingFeeByPair[sKey] += addingFee
		tradeAcceptedContents = append(tradeAcceptedContents, pdeTradeAcceptedContent)
	}

	pdeTradeAcceptedContentsBytes, err := json.Marshal(tradeAcceptedContents)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while marshaling pdeTradeAcceptedContents: %+v", err)
		return [][]string{}, nil
	}
	inst := []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDECrossPoolTradeAcceptedChainStatus,
		string(pdeTradeAcceptedContentsBytes),
	}
	return [][]string{inst}, nil
}

func (blockchain *BlockChain) buildInstructionsForPDETrade(
	contentStr string,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) ([][]string, error) {
	if currentPDEState == nil ||
		(currentPDEState.PDEPoolPairs == nil || len(currentPDEState.PDEPoolPairs) == 0) {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			common.PDETradeRefundChainStatus,
			contentStr,
		}
		return [][]string{inst}, nil
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde trade instruction: %+v", err)
		return [][]string{}, nil
	}
	var pdeTradeReqAction metadata.PDETradeRequestAction
	err = json.Unmarshal(contentBytes, &pdeTradeReqAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde trade instruction: %+v", err)
		return [][]string{}, nil
	}
	pairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, pdeTradeReqAction.Meta.TokenIDToBuyStr, pdeTradeReqAction.Meta.TokenIDToSellStr))

	pdePoolPair, found := currentPDEState.PDEPoolPairs[pairKey]
	if !found || (pdePoolPair.Token1PoolValue == 0 || pdePoolPair.Token2PoolValue == 0) {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			common.PDETradeRefundChainStatus,
			contentStr,
		}
		return [][]string{inst}, nil
	}
	// trade accepted
	tokenPoolValueToBuy := pdePoolPair.Token1PoolValue
	tokenPoolValueToSell := pdePoolPair.Token2PoolValue
	if pdePoolPair.Token1IDStr == pdeTradeReqAction.Meta.TokenIDToSellStr {
		tokenPoolValueToSell = pdePoolPair.Token1PoolValue
		tokenPoolValueToBuy = pdePoolPair.Token2PoolValue
	}
	invariant := big.NewInt(0)
	invariant.Mul(new(big.Int).SetUint64(tokenPoolValueToSell), new(big.Int).SetUint64(tokenPoolValueToBuy))
	fee := pdeTradeReqAction.Meta.TradingFee
	newTokenPoolValueToSell := big.NewInt(0)
	newTokenPoolValueToSell.Add(new(big.Int).SetUint64(tokenPoolValueToSell), new(big.Int).SetUint64(pdeTradeReqAction.Meta.SellAmount))

	newTokenPoolValueToBuy := big.NewInt(0).Div(invariant, newTokenPoolValueToSell).Uint64()
	modValue := big.NewInt(0).Mod(invariant, newTokenPoolValueToSell)
	if modValue.Cmp(big.NewInt(0)) != 0 {
		newTokenPoolValueToBuy++
	}
	if tokenPoolValueToBuy <= newTokenPoolValueToBuy {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			common.PDETradeRefundChainStatus,
			contentStr,
		}
		return [][]string{inst}, nil
	}

	receiveAmt := tokenPoolValueToBuy - newTokenPoolValueToBuy
	if pdeTradeReqAction.Meta.MinAcceptableAmount > receiveAmt {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			common.PDETradeRefundChainStatus,
			contentStr,
		}
		return [][]string{inst}, nil
	}

	// update current pde state on mem
	newTokenPoolValueToSell.Add(newTokenPoolValueToSell, new(big.Int).SetUint64(fee))

	pdePoolPair.Token1PoolValue = newTokenPoolValueToBuy
	pdePoolPair.Token2PoolValue = newTokenPoolValueToSell.Uint64()
	if pdePoolPair.Token1IDStr == pdeTradeReqAction.Meta.TokenIDToSellStr {
		pdePoolPair.Token1PoolValue = newTokenPoolValueToSell.Uint64()
		pdePoolPair.Token2PoolValue = newTokenPoolValueToBuy
	}

	pdeTradeAcceptedContent := metadata.PDETradeAcceptedContent{
		TraderAddressStr: pdeTradeReqAction.Meta.TraderAddressStr,
		TxRandomStr:      pdeTradeReqAction.Meta.TxRandomStr,
		TokenIDToBuyStr:  pdeTradeReqAction.Meta.TokenIDToBuyStr,
		ReceiveAmount:    receiveAmt,
		Token1IDStr:      pdePoolPair.Token1IDStr,
		Token2IDStr:      pdePoolPair.Token2IDStr,
		ShardID:          shardID,
		RequestedTxID:    pdeTradeReqAction.TxReqID,
	}
	pdeTradeAcceptedContent.Token1PoolValueOperation = metadata.TokenPoolValueOperation{
		Operator: "-",
		Value:    receiveAmt,
	}
	pdeTradeAcceptedContent.Token2PoolValueOperation = metadata.TokenPoolValueOperation{
		Operator: "+",
		Value:    pdeTradeReqAction.Meta.SellAmount + fee,
	}
	if pdePoolPair.Token1IDStr == pdeTradeReqAction.Meta.TokenIDToSellStr {
		pdeTradeAcceptedContent.Token1PoolValueOperation = metadata.TokenPoolValueOperation{
			Operator: "+",
			Value:    pdeTradeReqAction.Meta.SellAmount + fee,
		}
		pdeTradeAcceptedContent.Token2PoolValueOperation = metadata.TokenPoolValueOperation{
			Operator: "-",
			Value:    receiveAmt,
		}
	}
	pdeTradeAcceptedContentBytes, err := json.Marshal(pdeTradeAcceptedContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while marshaling pdeTradeAcceptedContent: %+v", err)
		return [][]string{}, nil
	}
	inst := []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDETradeAcceptedChainStatus,
		string(pdeTradeAcceptedContentBytes),
	}
	return [][]string{inst}, nil
}

func buildPDEWithdrawalAcceptedInst(
	wdMeta metadata.PDEWithdrawalRequest,
	shardID byte,
	metaType int,
	withdrawalTokenIDStr string,
	deductingPoolValue uint64,
	deductingShares uint64,
	txReqID common.Hash,
) ([]string, error) {
	wdAcceptedContent := metadata.PDEWithdrawalAcceptedContent{
		WithdrawalTokenIDStr: withdrawalTokenIDStr,
		WithdrawerAddressStr: wdMeta.WithdrawerAddressStr,
		DeductingPoolValue:   deductingPoolValue,
		DeductingShares:      deductingShares,
		PairToken1IDStr:      wdMeta.WithdrawalToken1IDStr,
		PairToken2IDStr:      wdMeta.WithdrawalToken2IDStr,
		TxReqID:              txReqID,
		ShardID:              shardID,
	}
	wdAcceptedContentBytes, err := json.Marshal(wdAcceptedContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while marshaling PDEWithdrawalAcceptedContent: %+v", err)
		return []string{}, nil
	}
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDEWithdrawalAcceptedChainStatus,
		string(wdAcceptedContentBytes),
	}, nil
}

func deductPDEAmountsV2(
	wdMeta metadata.PDEWithdrawalRequest,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) *DeductingAmountsByWithdrawal {
	var deductingAmounts *DeductingAmountsByWithdrawal
	pairKey := string(rawdbv2.BuildPDEPoolForPairKey(
		beaconHeight,
		wdMeta.WithdrawalToken1IDStr, wdMeta.WithdrawalToken2IDStr,
	))
	pdePoolPair, found := currentPDEState.PDEPoolPairs[pairKey]
	if !found || pdePoolPair == nil {
		return deductingAmounts
	}
	shareForWithdrawerKey := string(rawdbv2.BuildPDESharesKeyV2(
		beaconHeight,
		wdMeta.WithdrawalToken1IDStr, wdMeta.WithdrawalToken2IDStr, wdMeta.WithdrawerAddressStr,
	))
	currentSharesForWithdrawer, found := currentPDEState.PDEShares[shareForWithdrawerKey]
	if !found || currentSharesForWithdrawer == 0 {
		return deductingAmounts
	}

	totalSharesForPairPrefix := string(rawdbv2.BuildPDESharesKeyV2(
		beaconHeight,
		wdMeta.WithdrawalToken1IDStr, wdMeta.WithdrawalToken2IDStr, "",
	))
	totalSharesForPair := big.NewInt(0)
	for shareKey, shareAmt := range currentPDEState.PDEShares {
		if strings.Contains(shareKey, totalSharesForPairPrefix) {
			totalSharesForPair.Add(totalSharesForPair, new(big.Int).SetUint64(shareAmt))
		}
	}
	if totalSharesForPair.Cmp(big.NewInt(0)) == 0 {
		return deductingAmounts
	}
	wdSharesForWithdrawer := wdMeta.WithdrawalShareAmt
	if wdSharesForWithdrawer > currentSharesForWithdrawer {
		wdSharesForWithdrawer = currentSharesForWithdrawer
	}
	if wdSharesForWithdrawer == 0 {
		return deductingAmounts
	}

	deductingAmounts = &DeductingAmountsByWithdrawal{}
	deductingPoolValueToken1 := big.NewInt(0)
	deductingPoolValueToken1.Mul(new(big.Int).SetUint64(pdePoolPair.Token1PoolValue), new(big.Int).SetUint64(wdSharesForWithdrawer))
	deductingPoolValueToken1.Div(deductingPoolValueToken1, totalSharesForPair)
	if pdePoolPair.Token1PoolValue < deductingPoolValueToken1.Uint64() {
		pdePoolPair.Token1PoolValue = 0
	} else {
		pdePoolPair.Token1PoolValue -= deductingPoolValueToken1.Uint64()
	}
	deductingAmounts.Token1IDStr = pdePoolPair.Token1IDStr
	deductingAmounts.PoolValue1 = deductingPoolValueToken1.Uint64()

	deductingPoolValueToken2 := big.NewInt(0)
	deductingPoolValueToken2.Mul(new(big.Int).SetUint64(pdePoolPair.Token2PoolValue), new(big.Int).SetUint64(wdSharesForWithdrawer))
	deductingPoolValueToken2.Div(deductingPoolValueToken2, totalSharesForPair)
	if pdePoolPair.Token2PoolValue < deductingPoolValueToken2.Uint64() {
		pdePoolPair.Token2PoolValue = 0
	} else {
		pdePoolPair.Token2PoolValue -= deductingPoolValueToken2.Uint64()
	}
	deductingAmounts.Token2IDStr = pdePoolPair.Token2IDStr
	deductingAmounts.PoolValue2 = deductingPoolValueToken2.Uint64()

	if currentPDEState.PDEShares[shareForWithdrawerKey] < wdSharesForWithdrawer {
		currentPDEState.PDEShares[shareForWithdrawerKey] = 0
	} else {
		currentPDEState.PDEShares[shareForWithdrawerKey] -= wdSharesForWithdrawer
	}
	deductingAmounts.Shares = wdSharesForWithdrawer
	return deductingAmounts
}

func (blockchain *BlockChain) buildInstructionsForPDEWithdrawal(
	contentStr string,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) ([][]string, error) {
	if currentPDEState == nil {
		Logger.log.Warn("WARN - [buildInstructionsForPDEWithdrawal]: Current PDE state is null.")
		return [][]string{}, nil
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde withdrawal action: %+v", err)
		return [][]string{}, nil
	}
	var pdeWithdrawalRequestAction metadata.PDEWithdrawalRequestAction
	err = json.Unmarshal(contentBytes, &pdeWithdrawalRequestAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde withdrawal request action: %+v", err)
		return [][]string{}, nil
	}
	wdMeta := pdeWithdrawalRequestAction.Meta
	deductingAmounts := deductPDEAmountsV2(
		wdMeta,
		currentPDEState,
		beaconHeight,
	)

	if deductingAmounts == nil {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			common.PDEWithdrawalRejectedChainStatus,
			contentStr,
		}
		return [][]string{inst}, nil
	}

	insts := [][]string{}
	inst1, err := buildPDEWithdrawalAcceptedInst(
		wdMeta,
		shardID,
		metaType,
		deductingAmounts.Token1IDStr,
		deductingAmounts.PoolValue1,
		deductingAmounts.Shares,
		pdeWithdrawalRequestAction.TxReqID,
	)
	if err != nil {
		return [][]string{}, nil
	}
	insts = append(insts, inst1)
	inst2, err := buildPDEWithdrawalAcceptedInst(
		wdMeta,
		shardID,
		metaType,
		deductingAmounts.Token2IDStr,
		deductingAmounts.PoolValue2,
		0,
		pdeWithdrawalRequestAction.TxReqID,
	)
	if err != nil {
		return [][]string{}, nil
	}
	insts = append(insts, inst2)
	return insts, nil
}

type shareInfo struct {
	shareKey string
	shareAmt uint64
}

type tradingFeeForContributorByPair struct {
	ContributorAddressStr string
	FeeAmt                uint64
	Token1IDStr           string
	Token2IDStr           string
}

func (blockchain *BlockChain) buildInstForTradingFeesDist(
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
	tradingFeeByPair map[string]uint64,
) []string {
	feesForContributorsByPair := []*tradingFeeForContributorByPair{}
	pdeShares := currentPDEState.PDEShares

	var keys []string
	for k := range tradingFeeByPair {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, sKey := range keys {
		feeAmt := tradingFeeByPair[sKey]
		allSharesByPair := []shareInfo{}
		totalSharesOfPair := big.NewInt(0)

		var shareKeys []string
		for shareKey := range pdeShares {
			shareKeys = append(shareKeys, shareKey)
		}
		sort.Strings(shareKeys)
		for _, shareKey := range shareKeys {
			shareAmt := pdeShares[shareKey]
			if strings.Contains(shareKey, sKey) {
				allSharesByPair = append(allSharesByPair, shareInfo{shareKey: shareKey, shareAmt: shareAmt})
				totalSharesOfPair.Add(totalSharesOfPair, new(big.Int).SetUint64(shareAmt))
			}
		}

		accumFees := big.NewInt(0)
		totalFees := new(big.Int).SetUint64(feeAmt)
		for idx, sInfo := range allSharesByPair {
			feeForContributor := big.NewInt(0)
			if idx == len(allSharesByPair)-1 {
				feeForContributor.Sub(totalFees, accumFees)
			} else {
				feeForContributor.Mul(totalFees, new(big.Int).SetUint64(sInfo.shareAmt))
				feeForContributor.Div(feeForContributor, totalSharesOfPair)
			}

			parts := strings.Split(sInfo.shareKey, "-")
			partsLen := len(parts)
			if partsLen < 5 {
				continue
			}
			feesForContributorsByPair = append(
				feesForContributorsByPair,
				&tradingFeeForContributorByPair{
					ContributorAddressStr: parts[partsLen-1],
					FeeAmt:                feeForContributor.Uint64(),
					Token2IDStr:           parts[partsLen-2],
					Token1IDStr:           parts[partsLen-3],
				},
			)
			accumFees.Add(accumFees, feeForContributor)
		}
	}
	if len(feesForContributorsByPair) == 0 {
		return []string{}
	}

	feesForContributorsByPairBytes, err := json.Marshal(feesForContributorsByPair)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while marshaling feesForContributorsByPair: %+v", err)
		return []string{}
	}
	return []string{
		strconv.Itoa(metadata.PDETradingFeesDistributionMeta),
		"",
		"",
		string(feesForContributorsByPairBytes),
	}
}

func (blockchain *BlockChain) buildInstructionsForPDEFeeWithdrawal(
	contentStr string,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) ([][]string, error) {
	if currentPDEState == nil {
		Logger.log.Warn("WARN - [buildInstructionsForPDEFeeWithdrawal]: Current PDE state is null.")
		return [][]string{}, nil
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde withdrawal action: %+v", err)
		return [][]string{}, nil
	}
	var pdeFeeWithdrawalRequestAction metadata.PDEFeeWithdrawalRequestAction
	err = json.Unmarshal(contentBytes, &pdeFeeWithdrawalRequestAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde fee withdrawal request action: %+v", err)
		return [][]string{}, nil
	}
	wdMeta := pdeFeeWithdrawalRequestAction.Meta
	pdeTradingFees := currentPDEState.PDETradingFees
	tradingFeeKey := string(rawdbv2.BuildPDETradingFeeKey(
		beaconHeight,
		wdMeta.WithdrawalToken1IDStr,
		wdMeta.WithdrawalToken2IDStr,
		wdMeta.WithdrawerAddressStr,
	))
	withdrawableFee, found := pdeTradingFees[tradingFeeKey]
	if !found || withdrawableFee < wdMeta.WithdrawalFeeAmt {
		rejectedInst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			common.PDEFeeWithdrawalRejectedChainStatus,
			contentStr,
		}
		return [][]string{rejectedInst}, nil
	}
	pdeTradingFees[tradingFeeKey] -= wdMeta.WithdrawalFeeAmt
	acceptedInst := []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PDEFeeWithdrawalAcceptedChainStatus,
		contentStr,
	}
	return [][]string{acceptedInst}, nil
}
