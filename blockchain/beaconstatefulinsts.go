package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/instruction"
	"math/big"
	"sort"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
)

// build instructions at beacon chain before syncing to shards
func (blockchain *BlockChain) collectStatefulActions(
	shardBlockInstructions [][]string,
) [][]string {
	// stateful instructions are dependently processed with results of instructioins before them in shards2beacon blocks
	statefulInsts := [][]string{}
	for _, inst := range shardBlockInstructions {
		if len(inst) < 2 {
			continue
		}
		if instruction.IsConsensusInstruction(inst[0]) {
			continue
		}

		metaType, err := strconv.Atoi(inst[0])
		if err != nil {
			Logger.log.Error(err)
			continue
		}
		switch metaType {
		case metadata.IssuingRequestMeta,
			metadata.IssuingETHRequestMeta,
			metadata.PDEContributionMeta,
			metadata.PDETradeRequestMeta,
			metadata.PDEWithdrawalRequestMeta,
			metadata.PDEFeeWithdrawalRequestMeta,
			metadata.PDEPRVRequiredContributionRequestMeta,
			metadata.PDECrossPoolTradeRequestMeta,
			metadata.PortalCustodianDepositMeta,
			metadata.PortalUserRegisterMeta,
			metadata.PortalUserRequestPTokenMeta,
			metadata.PortalExchangeRatesMeta,
			metadata.RelayingBNBHeaderMeta,
			metadata.RelayingBTCHeaderMeta,
			metadata.PortalCustodianWithdrawRequestMeta,
			metadata.PortalRedeemRequestMeta,
			metadata.PortalRequestUnlockCollateralMeta,
			metadata.PortalLiquidateCustodianMeta,
			metadata.PortalRequestWithdrawRewardMeta,
			metadata.PortalRedeemLiquidateExchangeRatesMeta,
			metadata.PortalLiquidationCustodianDepositMetaV2,
			metadata.PortalLiquidationCustodianDepositResponseMeta,
			metadata.PortalReqMatchingRedeemMeta,
			metadata.PortalTopUpWaitingPortingRequestMeta:
			statefulInsts = append(statefulInsts, inst)

		default:
			continue
		}
	}
	return statefulInsts
}

func groupPDEActionsByShardID(
	pdeActionsByShardID map[byte][][]string,
	action []string,
	shardID byte,
) map[byte][][]string {
	_, found := pdeActionsByShardID[shardID]
	if !found {
		pdeActionsByShardID[shardID] = [][]string{action}
	} else {
		pdeActionsByShardID[shardID] = append(pdeActionsByShardID[shardID], action)
	}
	return pdeActionsByShardID
}

func (blockchain *BlockChain) buildStatefulInstructions(
	stateDB *statedb.StateDB,
	statefulActionsByShardID map[byte][][]string,
	beaconHeight uint64,
	rewardForCustodianByEpoch map[common.Hash]uint64,
	portalParams PortalParams) [][]string {
	currentPDEState, err := InitCurrentPDEStateFromDB(stateDB, beaconHeight-1)
	if err != nil {
		Logger.log.Error(err)
	}

	currentPortalState, err := InitCurrentPortalStateFromDB(stateDB)
	if err != nil {
		Logger.log.Error(err)
	}

	pm := NewPortalManager()
	relayingHeaderState, err := blockchain.InitRelayingHeaderChainStateFromDB()
	if err != nil {
		Logger.log.Error(err)
	}

	accumulatedValues := &metadata.AccumulatedValues{
		UniqETHTxsUsed:   [][]byte{},
		DBridgeTokenPair: map[string][]byte{},
		CBridgeTokens:    []*common.Hash{},
	}
	instructions := [][]string{}

	// pde instructions
	pdeContributionActionsByShardID := map[byte][][]string{}
	pdePRVRequiredContributionActionsByShardID := map[byte][][]string{}
	pdeTradeActionsByShardID := map[byte][][]string{}
	pdeCrossPoolTradeActionsByShardID := map[byte][][]string{}
	pdeWithdrawalActionsByShardID := map[byte][][]string{}
	pdeFeeWithdrawalActionsByShardID := map[byte][][]string{}

	// portal instructions
	portalCustodianDepositActionsByShardID := map[byte][][]string{}
	portalUserReqPortingActionsByShardID := map[byte][][]string{}
	portalUserReqPTokenActionsByShardID := map[byte][][]string{}
	portalExchangeRatesActionsByShardID := map[byte][][]string{}
	portalRedeemReqActionsByShardID := map[byte][][]string{}
	portalCustodianWithdrawActionsByShardID := map[byte][][]string{}
	portalReqUnlockCollateralActionsByShardID := map[byte][][]string{}
	portalReqWithdrawRewardActionsByShardID := map[byte][][]string{}
	portalRedeemLiquidateExchangeRatesActionByShardID := map[byte][][]string{}
	portalLiquidationCustodianDepositActionByShardID := map[byte][][]string{}
	portalReqMatchingRedeemActionsByShardID := map[byte][][]string{}
	portalTopUpWaitingPortingActionsByShardID := map[byte][][]string{}

	var keys []int
	for k := range statefulActionsByShardID {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, value := range keys {
		shardID := byte(value)
		actions := statefulActionsByShardID[shardID]
		for _, action := range actions {
			metaType, err := strconv.Atoi(action[0])
			if err != nil {
				continue
			}
			contentStr := action[1]
			newInst := [][]string{}
			switch metaType {
			case metadata.IssuingRequestMeta:
				newInst, err = blockchain.buildInstructionsForIssuingReq(stateDB, contentStr, shardID, metaType, accumulatedValues)

			case metadata.IssuingETHRequestMeta:
				newInst, err = blockchain.buildInstructionsForIssuingETHReq(stateDB, contentStr, shardID, metaType, accumulatedValues)

			case metadata.PDEContributionMeta:
				pdeContributionActionsByShardID = groupPDEActionsByShardID(
					pdeContributionActionsByShardID,
					action,
					shardID,
				)
			case metadata.PDEPRVRequiredContributionRequestMeta:
				pdePRVRequiredContributionActionsByShardID = groupPDEActionsByShardID(
					pdePRVRequiredContributionActionsByShardID,
					action,
					shardID,
				)
			case metadata.PDETradeRequestMeta:
				pdeTradeActionsByShardID = groupPDEActionsByShardID(
					pdeTradeActionsByShardID,
					action,
					shardID,
				)
			case metadata.PDECrossPoolTradeRequestMeta:
				pdeCrossPoolTradeActionsByShardID = groupPDEActionsByShardID(
					pdeCrossPoolTradeActionsByShardID,
					action,
					shardID,
				)
			case metadata.PDEWithdrawalRequestMeta:
				pdeWithdrawalActionsByShardID = groupPDEActionsByShardID(
					pdeWithdrawalActionsByShardID,
					action,
					shardID,
				)
			case metadata.PDEFeeWithdrawalRequestMeta:
				pdeFeeWithdrawalActionsByShardID = groupPDEActionsByShardID(
					pdeFeeWithdrawalActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalCustodianDepositMeta:
				{
					portalCustodianDepositActionsByShardID = groupPortalActionsByShardID(
						portalCustodianDepositActionsByShardID,
						action,
						shardID,
					)
				}

			case metadata.PortalUserRegisterMeta:
				portalUserReqPortingActionsByShardID = groupPortalActionsByShardID(
					portalUserReqPortingActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalUserRequestPTokenMeta:
				portalUserReqPTokenActionsByShardID = groupPortalActionsByShardID(
					portalUserReqPTokenActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalExchangeRatesMeta:
				portalExchangeRatesActionsByShardID = groupPortalActionsByShardID(
					portalExchangeRatesActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalCustodianWithdrawRequestMeta:
				portalCustodianWithdrawActionsByShardID = groupPortalActionsByShardID(
					portalCustodianWithdrawActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalRedeemRequestMeta:
				portalRedeemReqActionsByShardID = groupPortalActionsByShardID(
					portalRedeemReqActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalRequestUnlockCollateralMeta:
				portalReqUnlockCollateralActionsByShardID = groupPortalActionsByShardID(
					portalReqUnlockCollateralActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalRequestWithdrawRewardMeta:
				portalReqWithdrawRewardActionsByShardID = groupPortalActionsByShardID(
					portalReqWithdrawRewardActionsByShardID,
					action,
					shardID,
				)

			case metadata.PortalRedeemLiquidateExchangeRatesMeta:
				portalRedeemLiquidateExchangeRatesActionByShardID = groupPortalActionsByShardID(
					portalRedeemLiquidateExchangeRatesActionByShardID,
					action,
					shardID,
				)
			case metadata.PortalLiquidationCustodianDepositMetaV2:
				portalLiquidationCustodianDepositActionByShardID = groupPortalActionsByShardID(
					portalLiquidationCustodianDepositActionByShardID,
					action,
					shardID,
				)
			case metadata.PortalReqMatchingRedeemMeta:
				portalReqMatchingRedeemActionsByShardID = groupPortalActionsByShardID(
					portalReqMatchingRedeemActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalTopUpWaitingPortingRequestMeta:
				portalTopUpWaitingPortingActionsByShardID = groupPortalActionsByShardID(
					portalTopUpWaitingPortingActionsByShardID,
					action,
					shardID,
				)
			case metadata.RelayingBNBHeaderMeta:
				pm.relayingChains[metadata.RelayingBNBHeaderMeta].putAction(action)
			case metadata.RelayingBTCHeaderMeta:
				pm.relayingChains[metadata.RelayingBTCHeaderMeta].putAction(action)
			default:
				continue
			}
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	pdeInsts, err := blockchain.handlePDEInsts(
		beaconHeight-1, currentPDEState,
		pdeContributionActionsByShardID,
		pdePRVRequiredContributionActionsByShardID,
		pdeTradeActionsByShardID,
		pdeCrossPoolTradeActionsByShardID,
		pdeWithdrawalActionsByShardID,
		pdeFeeWithdrawalActionsByShardID,
	)

	if err != nil {
		Logger.log.Error(err)
		return instructions
	}
	if len(pdeInsts) > 0 {
		instructions = append(instructions, pdeInsts...)
	}

	// handle portal instructions
	portalInsts, err := blockchain.handlePortalInsts(
		stateDB,
		beaconHeight-1,
		currentPortalState,
		portalCustodianDepositActionsByShardID,
		portalUserReqPortingActionsByShardID,
		portalUserReqPTokenActionsByShardID,
		portalExchangeRatesActionsByShardID,
		portalRedeemReqActionsByShardID,
		portalCustodianWithdrawActionsByShardID,
		portalReqUnlockCollateralActionsByShardID,
		portalRedeemLiquidateExchangeRatesActionByShardID,
		portalLiquidationCustodianDepositActionByShardID,
		portalTopUpWaitingPortingActionsByShardID,
		portalReqMatchingRedeemActionsByShardID,
		portalReqWithdrawRewardActionsByShardID,
		rewardForCustodianByEpoch,
		portalParams,
	)

	if err != nil {
		Logger.log.Error(err)
		return instructions
	}
	if len(portalInsts) > 0 {
		instructions = append(instructions, portalInsts...)
	}

	// handle relaying instructions
	relayingInsts := blockchain.handleRelayingInsts(relayingHeaderState, pm)
	if len(relayingInsts) > 0 {
		instructions = append(instructions, relayingInsts...)
	}

	return instructions
}

func isTradingFairContainsPRV(
	tokenIDToSellStr string,
	tokenIDToBuyStr string,
) bool {
	return tokenIDToSellStr == common.PRVCoinID.String() || tokenIDToBuyStr == common.PRVCoinID.String()
}

func isPoolPairExisting(
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
	token1IDStr string,
	token2IDStr string,
) bool {
	poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, token1IDStr, token2IDStr))
	poolPair, found := currentPDEState.PDEPoolPairs[poolPairKey]
	if !found || poolPair == nil || poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
		return false
	}
	return true
}

func calcTradeValue(
	pdePoolPair *rawdbv2.PDEPoolForPair,
	tokenIDStrToSell string,
	sellAmount uint64,
) (uint64, uint64, uint64) {
	tokenPoolValueToBuy := pdePoolPair.Token1PoolValue
	tokenPoolValueToSell := pdePoolPair.Token2PoolValue
	if pdePoolPair.Token1IDStr == tokenIDStrToSell {
		tokenPoolValueToSell = pdePoolPair.Token1PoolValue
		tokenPoolValueToBuy = pdePoolPair.Token2PoolValue
	}
	invariant := big.NewInt(0)
	invariant.Mul(new(big.Int).SetUint64(tokenPoolValueToSell), new(big.Int).SetUint64(tokenPoolValueToBuy))
	newTokenPoolValueToSell := big.NewInt(0)
	newTokenPoolValueToSell.Add(new(big.Int).SetUint64(tokenPoolValueToSell), new(big.Int).SetUint64(sellAmount))

	newTokenPoolValueToBuy := big.NewInt(0).Div(invariant, newTokenPoolValueToSell).Uint64()
	modValue := big.NewInt(0).Mod(invariant, newTokenPoolValueToSell)
	if modValue.Cmp(big.NewInt(0)) != 0 {
		newTokenPoolValueToBuy++
	}
	if tokenPoolValueToBuy <= newTokenPoolValueToBuy {
		return uint64(0), uint64(0), uint64(0)
	}
	return tokenPoolValueToBuy - newTokenPoolValueToBuy, newTokenPoolValueToBuy, newTokenPoolValueToSell.Uint64()
}

func prepareInfoForSorting(
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
	tradeAction metadata.PDECrossPoolTradeRequestAction,
) (uint64, uint64) {
	prvIDStr := common.PRVCoinID.String()
	tradeMeta := tradeAction.Meta
	sellAmount := tradeMeta.SellAmount
	tradingFee := tradeMeta.TradingFee
	if tradeMeta.TokenIDToSellStr == prvIDStr {
		return tradingFee, sellAmount
	}
	poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, prvIDStr, tradeMeta.TokenIDToSellStr))
	poolPair, _ := currentPDEState.PDEPoolPairs[poolPairKey]
	sellAmount, _, _ = calcTradeValue(poolPair, tradeMeta.TokenIDToSellStr, sellAmount)
	return tradingFee, sellAmount
}

func categorizeNSortPDECrossPoolTradeInstsByFee(
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
	pdeCrossPoolTradeActionsByShardID map[byte][][]string,
) ([]metadata.PDECrossPoolTradeRequestAction, []metadata.PDECrossPoolTradeRequestAction) {
	prvIDStr := common.PRVCoinID.String()
	tradableActions := []metadata.PDECrossPoolTradeRequestAction{}
	untradableActions := []metadata.PDECrossPoolTradeRequestAction{}
	var keys []int
	for k := range pdeCrossPoolTradeActionsByShardID {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, value := range keys {
		shardID := byte(value)
		actions := pdeCrossPoolTradeActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while decoding content string of pde trade action: %+v", err)
				continue
			}
			var crossPoolTradeRequestAction metadata.PDECrossPoolTradeRequestAction
			err = json.Unmarshal(contentBytes, &crossPoolTradeRequestAction)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while unmarshaling pde cross pool trade request action: %+v", err)
				continue
			}
			tradeMeta := crossPoolTradeRequestAction.Meta
			if (isTradingFairContainsPRV(tradeMeta.TokenIDToSellStr, tradeMeta.TokenIDToBuyStr) && !isPoolPairExisting(beaconHeight, currentPDEState, tradeMeta.TokenIDToSellStr, tradeMeta.TokenIDToBuyStr)) ||
				(!isTradingFairContainsPRV(tradeMeta.TokenIDToSellStr, tradeMeta.TokenIDToBuyStr) && (!isPoolPairExisting(beaconHeight, currentPDEState, prvIDStr, tradeMeta.TokenIDToSellStr) || !isPoolPairExisting(beaconHeight, currentPDEState, prvIDStr, tradeMeta.TokenIDToBuyStr))) {
				untradableActions = append(untradableActions, crossPoolTradeRequestAction)
				continue
			}
			tradableActions = append(tradableActions, crossPoolTradeRequestAction)
		}
	}

	// sort tradable actions by trading fee
	sort.SliceStable(tradableActions, func(i, j int) bool {
		firstTradingFee, firstSellAmount := prepareInfoForSorting(
			currentPDEState,
			beaconHeight,
			tradableActions[i],
		)
		secondTradingFee, secondSellAmount := prepareInfoForSorting(
			currentPDEState,
			beaconHeight,
			tradableActions[j],
		)
		// comparing a/b to c/d is equivalent with comparing a*d to c*b
		firstItemProportion := big.NewInt(0)
		firstItemProportion.Mul(
			new(big.Int).SetUint64(firstTradingFee),
			new(big.Int).SetUint64(secondSellAmount),
		)
		secondItemProportion := big.NewInt(0)
		secondItemProportion.Mul(
			new(big.Int).SetUint64(secondTradingFee),
			new(big.Int).SetUint64(firstSellAmount),
		)
		return firstItemProportion.Cmp(secondItemProportion) == 1
	})
	return tradableActions, untradableActions
}

func sortPDETradeInstsByFee(
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
	pdeTradeActionsByShardID map[byte][][]string,
) []metadata.PDETradeRequestAction {
	tradesByPairs := make(map[string][]metadata.PDETradeRequestAction)

	var keys []int
	for k := range pdeTradeActionsByShardID {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, value := range keys {
		shardID := byte(value)
		actions := pdeTradeActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while decoding content string of pde trade action: %+v", err)
				continue
			}
			var pdeTradeReqAction metadata.PDETradeRequestAction
			err = json.Unmarshal(contentBytes, &pdeTradeReqAction)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while unmarshaling pde trade action: %+v", err)
				continue
			}
			tradeMeta := pdeTradeReqAction.Meta
			poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, tradeMeta.TokenIDToBuyStr, tradeMeta.TokenIDToSellStr))
			tradesByPair, found := tradesByPairs[poolPairKey]
			if !found {
				tradesByPairs[poolPairKey] = []metadata.PDETradeRequestAction{pdeTradeReqAction}
			} else {
				tradesByPairs[poolPairKey] = append(tradesByPair, pdeTradeReqAction)
			}
		}
	}

	notExistingPairTradeActions := []metadata.PDETradeRequestAction{}
	sortedExistingPairTradeActions := []metadata.PDETradeRequestAction{}

	var ppKeys []string
	for k := range tradesByPairs {
		ppKeys = append(ppKeys, k)
	}
	sort.Strings(ppKeys)
	for _, poolPairKey := range ppKeys {
		tradeActions := tradesByPairs[poolPairKey]
		poolPair, found := currentPDEState.PDEPoolPairs[poolPairKey]
		if !found || poolPair == nil {
			notExistingPairTradeActions = append(notExistingPairTradeActions, tradeActions...)
			continue
		}
		if poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
			notExistingPairTradeActions = append(notExistingPairTradeActions, tradeActions...)
			continue
		}

		// sort trade actions by trading fee
		sort.Slice(tradeActions, func(i, j int) bool {
			// comparing a/b to c/d is equivalent with comparing a*d to c*b
			firstItemProportion := big.NewInt(0)
			firstItemProportion.Mul(
				new(big.Int).SetUint64(tradeActions[i].Meta.TradingFee),
				new(big.Int).SetUint64(tradeActions[j].Meta.SellAmount),
			)
			secondItemProportion := big.NewInt(0)
			secondItemProportion.Mul(
				new(big.Int).SetUint64(tradeActions[j].Meta.TradingFee),
				new(big.Int).SetUint64(tradeActions[i].Meta.SellAmount),
			)
			return firstItemProportion.Cmp(secondItemProportion) == 1
		})
		sortedExistingPairTradeActions = append(sortedExistingPairTradeActions, tradeActions...)
	}
	return append(sortedExistingPairTradeActions, notExistingPairTradeActions...)
}

func (blockchain *BlockChain) handlePDEInsts(
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
	pdeContributionActionsByShardID map[byte][][]string,
	pdePRVRequiredContributionActionsByShardID map[byte][][]string,
	pdeTradeActionsByShardID map[byte][][]string,
	pdeCrossPoolTradeActionsByShardID map[byte][][]string,
	pdeWithdrawalActionsByShardID map[byte][][]string,
	pdeFeeWithdrawalActionsByShardID map[byte][][]string,
) ([][]string, error) {
	instructions := [][]string{}

	// handle fee withdrawal
	var feeWRKeys []int
	for k := range pdeFeeWithdrawalActionsByShardID {
		feeWRKeys = append(feeWRKeys, int(k))
	}
	sort.Ints(feeWRKeys)
	for _, value := range feeWRKeys {
		shardID := byte(value)
		actions := pdeFeeWithdrawalActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPDEFeeWithdrawal(contentStr, shardID, metadata.PDEFeeWithdrawalRequestMeta, currentPDEState, beaconHeight)
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle trade
	sortedTradesActions := sortPDETradeInstsByFee(
		beaconHeight,
		currentPDEState,
		pdeTradeActionsByShardID,
	)
	for _, tradeAction := range sortedTradesActions {
		actionContentBytes, _ := json.Marshal(tradeAction)
		actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
		newInst, err := blockchain.buildInstructionsForPDETrade(actionContentBase64Str, tradeAction.ShardID, metadata.PDETradeRequestMeta, currentPDEState, beaconHeight)
		if err != nil {
			Logger.log.Error(err)
			continue
		}
		if len(newInst) > 0 {
			instructions = append(instructions, newInst...)
		}
	}

	// handle cross pool trade
	sortedTradableActions, untradableActions := categorizeNSortPDECrossPoolTradeInstsByFee(
		beaconHeight,
		currentPDEState,
		pdeCrossPoolTradeActionsByShardID,
	)
	tradableInsts, tradingFeeByPair := blockchain.buildInstsForSortedTradableActions(currentPDEState, beaconHeight, sortedTradableActions)
	untradableInsts := blockchain.buildInstsForUntradableActions(untradableActions)
	instructions = append(instructions, tradableInsts...)
	instructions = append(instructions, untradableInsts...)

	// calculate and build instruction for trading fees distribution
	tradingFeesDistInst := blockchain.buildInstForTradingFeesDist(currentPDEState, beaconHeight, tradingFeeByPair)
	if len(tradingFeesDistInst) > 0 {
		instructions = append(instructions, tradingFeesDistInst)
	}

	// handle withdrawal
	var wrKeys []int
	for k := range pdeWithdrawalActionsByShardID {
		wrKeys = append(wrKeys, int(k))
	}
	sort.Ints(wrKeys)
	for _, value := range wrKeys {
		shardID := byte(value)
		actions := pdeWithdrawalActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPDEWithdrawal(contentStr, shardID, metadata.PDEWithdrawalRequestMeta, currentPDEState, beaconHeight)
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle contribution
	var ctKeys []int
	for k := range pdeContributionActionsByShardID {
		ctKeys = append(ctKeys, int(k))
	}
	sort.Ints(ctKeys)
	for _, value := range ctKeys {
		shardID := byte(value)
		actions := pdeContributionActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPDEContribution(contentStr, shardID, metadata.PDEContributionMeta, currentPDEState, beaconHeight, false)
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle prv required contribution
	var prvRequiredContribKeys []int
	for k := range pdePRVRequiredContributionActionsByShardID {
		prvRequiredContribKeys = append(prvRequiredContribKeys, int(k))
	}
	sort.Ints(prvRequiredContribKeys)
	for _, value := range prvRequiredContribKeys {
		shardID := byte(value)
		actions := pdePRVRequiredContributionActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPDEContribution(contentStr, shardID, metadata.PDEPRVRequiredContributionRequestMeta, currentPDEState, beaconHeight, true)
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}
	return instructions, nil
}

// Portal
func groupPortalActionsByShardID(
	portalActionsByShardID map[byte][][]string,
	action []string,
	shardID byte,
) map[byte][][]string {
	_, found := portalActionsByShardID[shardID]
	if !found {
		portalActionsByShardID[shardID] = [][]string{action}
	} else {
		portalActionsByShardID[shardID] = append(portalActionsByShardID[shardID], action)
	}
	return portalActionsByShardID
}

func (blockchain *BlockChain) handlePortalInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	currentPortalState *CurrentPortalState,
	portalCustodianDepositActionsByShardID map[byte][][]string,
	portalUserRequestPortingActionsByShardID map[byte][][]string,
	portalUserRequestPTokenActionsByShardID map[byte][][]string,
	portalExchangeRatesActionsByShardID map[byte][][]string,
	portalRedeemReqActionsByShardID map[byte][][]string,
	portalCustodianWithdrawActionByShardID map[byte][][]string,
	portalReqUnlockCollateralActionsByShardID map[byte][][]string,
	portalRedeemLiquidateExchangeRatesActionByShardID map[byte][][]string,
	portalLiquidationCustodianDepositActionByShardID map[byte][][]string,
	portalTopUpWaitingPortingActionsByShardID map[byte][][]string,
	portalReqMatchingRedeemActionByShardID map[byte][][]string,
	portalReqWithdrawRewardActionsByShardID map[byte][][]string,
	rewardForCustodianByEpoch map[common.Hash]uint64,
	portalParams PortalParams,
) ([][]string, error) {
	instructions := [][]string{}
	newMatchedRedeemReqIDs := []string{}

	// auto-liquidation portal instructions
	portalLiquidationInsts, err := blockchain.autoCheckAndCreatePortalLiquidationInsts(
		beaconHeight,
		currentPortalState,
		portalParams,
	)
	if err != nil {
		Logger.log.Error(err)
	}
	if len(portalLiquidationInsts) > 0 {
		instructions = append(instructions, portalLiquidationInsts...)
	}

	// handle portal custodian deposit inst
	var custodianShardIDKeys []int
	for k := range portalCustodianDepositActionsByShardID {
		custodianShardIDKeys = append(custodianShardIDKeys, int(k))
	}

	sort.Ints(custodianShardIDKeys)
	for _, value := range custodianShardIDKeys {
		shardID := byte(value)
		actions := portalCustodianDepositActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForCustodianDeposit(
				contentStr,
				shardID,
				metadata.PortalCustodianDepositMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle portal user request porting inst
	var requestPortingShardIDKeys []int
	for k := range portalUserRequestPortingActionsByShardID {
		requestPortingShardIDKeys = append(requestPortingShardIDKeys, int(k))
	}

	sort.Ints(requestPortingShardIDKeys)
	for _, value := range requestPortingShardIDKeys {
		shardID := byte(value)
		actions := portalUserRequestPortingActionsByShardID[shardID]

		//check identity of porting request id
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPortingRequest(
				stateDB,
				contentStr,
				shardID,
				metadata.PortalUserRegisterMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}
	// handle portal user request ptoken inst
	var reqPTokenShardIDKeys []int
	for k := range portalUserRequestPTokenActionsByShardID {
		reqPTokenShardIDKeys = append(reqPTokenShardIDKeys, int(k))
	}

	sort.Ints(reqPTokenShardIDKeys)
	for _, value := range reqPTokenShardIDKeys {
		shardID := byte(value)
		actions := portalUserRequestPTokenActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForReqPTokens(
				stateDB,
				contentStr,
				shardID,
				metadata.PortalUserRequestPTokenMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle portal redeem req inst
	var redeemReqShardIDKeys []int
	for k := range portalRedeemReqActionsByShardID {
		redeemReqShardIDKeys = append(redeemReqShardIDKeys, int(k))
	}

	sort.Ints(redeemReqShardIDKeys)
	for _, value := range redeemReqShardIDKeys {
		shardID := byte(value)
		actions := portalRedeemReqActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForRedeemRequest(
				stateDB,
				contentStr,
				shardID,
				metadata.PortalRedeemRequestMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	//handle portal exchange rates
	var exchangeRatesShardIDKeys []int
	for k := range portalExchangeRatesActionsByShardID {
		exchangeRatesShardIDKeys = append(exchangeRatesShardIDKeys, int(k))
	}

	sort.Ints(exchangeRatesShardIDKeys)
	for _, value := range exchangeRatesShardIDKeys {
		shardID := byte(value)
		actions := portalExchangeRatesActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForExchangeRates(
				contentStr,
				shardID,
				metadata.PortalExchangeRatesMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	//handle portal custodian withdraw
	var portalCustodianWithdrawShardIDKeys []int
	for k := range portalCustodianWithdrawActionByShardID {
		portalCustodianWithdrawShardIDKeys = append(portalCustodianWithdrawShardIDKeys, int(k))
	}

	sort.Ints(portalCustodianWithdrawShardIDKeys)
	for _, value := range portalCustodianWithdrawShardIDKeys {
		shardID := byte(value)
		actions := portalCustodianWithdrawActionByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForCustodianWithdraw(
				contentStr,
				shardID,
				metadata.PortalCustodianWithdrawRequestMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle portal req unlock collateral inst
	var reqUnlockCollateralShardIDKeys []int
	for k := range portalReqUnlockCollateralActionsByShardID {
		reqUnlockCollateralShardIDKeys = append(reqUnlockCollateralShardIDKeys, int(k))
	}

	sort.Ints(reqUnlockCollateralShardIDKeys)
	for _, value := range reqUnlockCollateralShardIDKeys {
		shardID := byte(value)
		actions := portalReqUnlockCollateralActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForReqUnlockCollateral(
				stateDB,
				contentStr,
				shardID,
				metadata.PortalRequestUnlockCollateralMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle liquidation user redeem ptoken  exchange rates
	var redeemLiquidateExchangeRatesActionByShardIDKeys []int
	for k := range portalRedeemLiquidateExchangeRatesActionByShardID {
		redeemLiquidateExchangeRatesActionByShardIDKeys = append(redeemLiquidateExchangeRatesActionByShardIDKeys, int(k))
	}

	sort.Ints(redeemLiquidateExchangeRatesActionByShardIDKeys)
	for _, value := range redeemLiquidateExchangeRatesActionByShardIDKeys {
		shardID := byte(value)
		actions := portalRedeemLiquidateExchangeRatesActionByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForLiquidationRedeemPTokenExchangeRates(
				contentStr,
				shardID,
				metadata.PortalRedeemLiquidateExchangeRatesMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle portal  liquidation custodian deposit inst
	var portalLiquidationCustodianDepositActionByShardIDKeys []int
	for k := range portalLiquidationCustodianDepositActionByShardID {
		portalLiquidationCustodianDepositActionByShardIDKeys = append(portalLiquidationCustodianDepositActionByShardIDKeys, int(k))
	}

	sort.Ints(portalLiquidationCustodianDepositActionByShardIDKeys)
	for _, value := range portalLiquidationCustodianDepositActionByShardIDKeys {
		shardID := byte(value)
		actions := portalLiquidationCustodianDepositActionByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForLiquidationCustodianDeposit(
				contentStr,
				shardID,
				metadata.PortalLiquidationCustodianDepositMetaV2,
				currentPortalState,
				beaconHeight,
				portalParams,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle portal top up waiting porting inst
	var portalTopUpWaitingPortingActionsByShardIDKeys []int
	for k := range portalTopUpWaitingPortingActionsByShardID {
		portalTopUpWaitingPortingActionsByShardIDKeys = append(portalTopUpWaitingPortingActionsByShardIDKeys, int(k))
	}
	sort.Ints(portalTopUpWaitingPortingActionsByShardIDKeys)
	for _, value := range portalTopUpWaitingPortingActionsByShardIDKeys {
		shardID := byte(value)
		actions := portalTopUpWaitingPortingActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstsForTopUpWaitingPorting(
				contentStr,
				shardID,
				metadata.PortalTopUpWaitingPortingRequestMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
			)
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle portal req matching redeem inst
	var reqMatchRedeemShardIDKeys []int
	for k := range portalReqMatchingRedeemActionByShardID {
		reqMatchRedeemShardIDKeys = append(reqMatchRedeemShardIDKeys, int(k))
	}

	sort.Ints(reqMatchRedeemShardIDKeys)
	for _, value := range reqMatchRedeemShardIDKeys {
		shardID := byte(value)
		actions := portalReqMatchingRedeemActionByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			var newInst [][]string
			newInst, newMatchedRedeemReqIDs, err = blockchain.buildInstructionsForReqMatchingRedeem(
				stateDB,
				contentStr,
				shardID,
				metadata.PortalReqMatchingRedeemMeta,
				currentPortalState,
				beaconHeight,
				portalParams,
				newMatchedRedeemReqIDs,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// check and create instruction for picking more custodians for timeout waiting redeem requests
	var timeOutRedeemReqInsts [][]string
	timeOutRedeemReqInsts, newMatchedRedeemReqIDs, err = blockchain.checkAndPickMoreCustodianForWaitingRedeemRequest(
		beaconHeight,
		currentPortalState,
		newMatchedRedeemReqIDs,
	)
	if err != nil {
		Logger.log.Error(err)
	}
	if len(timeOutRedeemReqInsts) > 0 {
		instructions = append(instructions, timeOutRedeemReqInsts...)
	}

	// calculate rewards (include porting fee and redeem fee) for custodians and build instructions at beaconHeight
	portalRewardsInsts, err := blockchain.handlePortalRewardInsts(
		beaconHeight,
		currentPortalState,
		portalReqWithdrawRewardActionsByShardID,
		rewardForCustodianByEpoch,
		newMatchedRedeemReqIDs,
	)

	if err != nil {
		Logger.log.Error(err)
	}
	if len(portalRewardsInsts) > 0 {
		instructions = append(instructions, portalRewardsInsts...)
	}

	return instructions, nil
}

// Header relaying
func groupRelayingActionsByShardID(
	relayingActionsByShardID map[byte][][]string,
	action []string,
	shardID byte,
) map[byte][][]string {
	_, found := relayingActionsByShardID[shardID]
	if !found {
		relayingActionsByShardID[shardID] = [][]string{action}
	} else {
		relayingActionsByShardID[shardID] = append(relayingActionsByShardID[shardID], action)
	}
	return relayingActionsByShardID
}

func (blockchain *BlockChain) autoCheckAndCreatePortalLiquidationInsts(
	beaconHeight uint64,
	currentPortalState *CurrentPortalState,
	portalParams PortalParams) ([][]string, error) {
	insts := [][]string{}

	// check there is any waiting porting request timeout
	expiredWaitingPortingInsts, err := blockchain.checkAndBuildInstForExpiredWaitingPortingRequest(beaconHeight, currentPortalState, portalParams)
	if err != nil {
		Logger.log.Errorf("Error when check and build custodian liquidation %v\n", err)
	}
	if len(expiredWaitingPortingInsts) > 0 {
		insts = append(insts, expiredWaitingPortingInsts...)
	}
	Logger.log.Infof("There are %v instruction for expired waiting porting in portal\n", len(expiredWaitingPortingInsts))

	// case 1: check there is any custodian doesn't send public tokens back to user after TimeOutCustodianReturnPubToken
	// get custodian's collateral to return user
	custodianLiqInsts, err := blockchain.checkAndBuildInstForCustodianLiquidation(beaconHeight, currentPortalState, portalParams)
	if err != nil {
		Logger.log.Errorf("Error when check and build custodian liquidation %v\n", err)
	}
	if len(custodianLiqInsts) > 0 {
		insts = append(insts, custodianLiqInsts...)
	}
	Logger.log.Infof("There are %v instruction for custodian liquidation in portal\n", len(custodianLiqInsts))

	// case 2: check collateral's value (locked collateral amount) drops below MinRatio

	exchangeRatesLiqInsts, err := buildInstForLiquidationTopPercentileExchangeRates(beaconHeight, currentPortalState, portalParams)
	if err != nil {
		Logger.log.Errorf("Error when check and build exchange rates liquidation %v\n", err)
	}
	if len(exchangeRatesLiqInsts) > 0 {
		insts = append(insts, exchangeRatesLiqInsts...)
	}

	Logger.log.Infof("There are %v instruction for exchange rates liquidation in portal\n", len(exchangeRatesLiqInsts))

	return insts, nil
}

// handlePortalRewardInsts
// 1. Build instructions for request withdraw portal reward
// 2. Build instructions portal reward for each beacon block
func (blockchain *BlockChain) handlePortalRewardInsts(
	beaconHeight uint64,
	currentPortalState *CurrentPortalState,
	portalReqWithdrawRewardActionsByShardID map[byte][][]string,
	rewardForCustodianByEpoch map[common.Hash]uint64,
	newMatchedRedeemReqIDs []string,
) ([][]string, error) {
	instructions := [][]string{}

	// Build instructions portal reward for each beacon block
	portalRewardInsts, err := blockchain.buildPortalRewardsInsts(beaconHeight, currentPortalState, rewardForCustodianByEpoch, newMatchedRedeemReqIDs)
	if err != nil {
		Logger.log.Error(err)
	}
	if len(portalRewardInsts) > 0 {
		instructions = append(instructions, portalRewardInsts...)
	}

	// handle portal request withdraw reward inst
	var shardIDKeys []int
	for k := range portalReqWithdrawRewardActionsByShardID {
		shardIDKeys = append(shardIDKeys, int(k))
	}

	sort.Ints(shardIDKeys)
	for _, value := range shardIDKeys {
		shardID := byte(value)
		actions := portalReqWithdrawRewardActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForReqWithdrawPortalReward(
				contentStr,
				shardID,
				metadata.PortalRequestWithdrawRewardMeta,
				currentPortalState,
				beaconHeight,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	return instructions, nil
}
