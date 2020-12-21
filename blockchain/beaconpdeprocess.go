package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"reflect"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
)

func (blockchain *BlockChain) processPDEInstructions(pdexStateDB *statedb.StateDB, beaconBlock *BeaconBlock) error {
	if !hasPDEInstruction(beaconBlock.Body.Instructions) {
		return nil
	}
	beaconHeight := beaconBlock.Header.Height - 1
	currentPDEState, err := InitCurrentPDEStateFromDB(pdexStateDB, beaconHeight)
	if err != nil {
		Logger.log.Error(err)
		return nil
	}
	backUpCurrentPDEState := new(CurrentPDEState)
	temp, err := json.Marshal(currentPDEState)
	if err != nil {
		return err
	}
	err = json.Unmarshal(temp, backUpCurrentPDEState)
	if err != nil {
		return err
	}
	for _, inst := range beaconBlock.Body.Instructions {
		if len(inst) < 2 {
			continue // Not error, just not PDE instruction
		}
		var err error
		switch inst[0] {
		case strconv.Itoa(metadata.PDEContributionMeta):
			err = blockchain.processPDEContributionV2(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDEPRVRequiredContributionRequestMeta):
			err = blockchain.processPDEContributionV2(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDETradeRequestMeta):
			err = blockchain.processPDETrade(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDECrossPoolTradeRequestMeta):
			err = blockchain.processPDECrossPoolTrade(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDEWithdrawalRequestMeta):
			err = blockchain.processPDEWithdrawal(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDEFeeWithdrawalRequestMeta):
			err = blockchain.processPDEFeeWithdrawal(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDETradingFeesDistributionMeta):
			err = blockchain.processPDETradingFeesDistribution(pdexStateDB, beaconHeight, inst, currentPDEState)
		}
		if err != nil {
			Logger.log.Error(err)
			return nil
		}
	}
	if reflect.DeepEqual(backUpCurrentPDEState, currentPDEState) {
		return nil
	}
	// store updated currentPDEState to leveldb with new beacon height
	err = storePDEStateToDB(pdexStateDB, beaconHeight+1, currentPDEState)
	if err != nil {
		Logger.log.Error(err)
	}
	return nil
}

func hasPDEInstruction(instructions [][]string) bool {
	hasPDEXInstruction := false
	for _, inst := range instructions {
		if len(inst) < 2 {
			continue // Not error, just not PDE instruction
		}
		switch inst[0] {
		case strconv.Itoa(metadata.PDEContributionMeta):
			hasPDEXInstruction = true
			break
		case strconv.Itoa(metadata.PDETradeRequestMeta):
			hasPDEXInstruction = true
			break
		case strconv.Itoa(metadata.PDEWithdrawalRequestMeta):
			hasPDEXInstruction = true
			break
		case strconv.Itoa(metadata.PDEFeeWithdrawalRequestMeta):
			hasPDEXInstruction = true
			break
		case strconv.Itoa(metadata.PDEPRVRequiredContributionRequestMeta):
			hasPDEXInstruction = true
			break
		case strconv.Itoa(metadata.PDECrossPoolTradeRequestMeta):
			hasPDEXInstruction = true
			break
		case strconv.Itoa(metadata.PDETradingFeesDistributionMeta):
			hasPDEXInstruction = true
			break
		}
	}
	return hasPDEXInstruction
}
func (blockchain *BlockChain) processPDEContributionV2(pdexStateDB *statedb.StateDB, beaconHeight uint64, instruction []string, currentPDEState *CurrentPDEState) error {
	if currentPDEState == nil {
		Logger.log.Warn("WARN - [processPDEContribution]: Current PDE state is null.")
		return nil
	}
	if len(instruction) != 4 {
		return nil // skip the instruction
	}
	contributionStatus := instruction[2]
	if contributionStatus == common.PDEContributionWaitingChainStatus {
		var waitingContribution metadata.PDEWaitingContribution
		err := json.Unmarshal([]byte(instruction[3]), &waitingContribution)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling content string of pde waiting contribution instruction: %+v", err)
			return nil
		}
		waitingContribPairKey := string(rawdbv2.BuildWaitingPDEContributionKey(beaconHeight, waitingContribution.PDEContributionPairID))
		currentPDEState.WaitingPDEContributions[waitingContribPairKey] = &rawdbv2.PDEContribution{
			ContributorAddressStr: waitingContribution.ContributorAddressStr,
			TokenIDStr:            waitingContribution.TokenIDStr,
			Amount:                waitingContribution.ContributedAmount,
			TxReqID:               waitingContribution.TxReqID,
		}

		contribStatus := metadata.PDEContributionStatus{
			Status: byte(common.PDEContributionWaitingStatus),
		}

		contribStatusBytes, _ := json.Marshal(contribStatus)
		err = statedb.TrackPDEContributionStatus(
			pdexStateDB,
			rawdbv2.PDEContributionStatusPrefix,
			[]byte(waitingContribution.PDEContributionPairID),
			contribStatusBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde waiting contribution status: %+v", err)
			return nil
		}

	} else if contributionStatus == common.PDEContributionRefundChainStatus {
		var refundContribution metadata.PDERefundContribution
		err := json.Unmarshal([]byte(instruction[3]), &refundContribution)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling content string of pde refund contribution instruction: %+v", err)
			return nil
		}
		waitingContribPairKey := string(rawdbv2.BuildWaitingPDEContributionKey(beaconHeight, refundContribution.PDEContributionPairID))
		existingWaitingContribution, found := currentPDEState.WaitingPDEContributions[waitingContribPairKey]
		if found {
			currentPDEState.DeletedWaitingPDEContributions[waitingContribPairKey] = existingWaitingContribution
			delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		}
		contribStatus := metadata.PDEContributionStatus{
			Status: byte(common.PDEContributionRefundStatus),
		}
		contribStatusBytes, _ := json.Marshal(contribStatus)
		err = statedb.TrackPDEContributionStatus(
			pdexStateDB,
			rawdbv2.PDEContributionStatusPrefix,
			[]byte(refundContribution.PDEContributionPairID),
			contribStatusBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde refund contribution status: %+v", err)
			return nil
		}

	} else if contributionStatus == common.PDEContributionMatchedChainStatus {
		var matchedContribution metadata.PDEMatchedContribution
		err := json.Unmarshal([]byte(instruction[3]), &matchedContribution)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling content string of pde matched contribution instruction: %+v", err)
			return nil
		}
		waitingContribPairKey := string(rawdbv2.BuildWaitingPDEContributionKey(beaconHeight, matchedContribution.PDEContributionPairID))
		existingWaitingContribution, found := currentPDEState.WaitingPDEContributions[waitingContribPairKey]
		if !found || existingWaitingContribution == nil {
			Logger.log.Errorf("ERROR: could not find out existing waiting contribution with unique pair id: %s", matchedContribution.PDEContributionPairID)
			return nil
		}
		incomingWaitingContribution := &rawdbv2.PDEContribution{
			ContributorAddressStr: matchedContribution.ContributorAddressStr,
			TokenIDStr:            matchedContribution.TokenIDStr,
			Amount:                matchedContribution.ContributedAmount,
			TxReqID:               matchedContribution.TxReqID,
		}
		updateWaitingContributionPairToPoolV2(
			beaconHeight,
			existingWaitingContribution,
			incomingWaitingContribution,
			currentPDEState,
		)
		currentPDEState.DeletedWaitingPDEContributions[waitingContribPairKey] = existingWaitingContribution
		delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		contribStatus := metadata.PDEContributionStatus{
			Status: byte(common.PDEContributionAcceptedStatus),
		}
		contribStatusBytes, _ := json.Marshal(contribStatus)
		err = statedb.TrackPDEContributionStatus(
			pdexStateDB,
			rawdbv2.PDEContributionStatusPrefix,
			[]byte(matchedContribution.PDEContributionPairID),
			contribStatusBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde accepted contribution status: %+v", err)
			return nil
		}

	} else if contributionStatus == common.PDEContributionMatchedNReturnedChainStatus {
		var matchedNReturnedContrib metadata.PDEMatchedNReturnedContribution
		err := json.Unmarshal([]byte(instruction[3]), &matchedNReturnedContrib)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling content string of pde matched and returned contribution instruction: %+v", err)
			return nil
		}
		waitingContribPairKey := string(rawdbv2.BuildWaitingPDEContributionKey(beaconHeight, matchedNReturnedContrib.PDEContributionPairID))
		waitingContribution, found := currentPDEState.WaitingPDEContributions[waitingContribPairKey]
		if found && waitingContribution != nil {
			incomingWaitingContribution := &rawdbv2.PDEContribution{
				ContributorAddressStr: matchedNReturnedContrib.ContributorAddressStr,
				TokenIDStr:            matchedNReturnedContrib.TokenIDStr,
				Amount:                matchedNReturnedContrib.ActualContributedAmount,
				TxReqID:               matchedNReturnedContrib.TxReqID,
			}
			existingWaitingContribution := &rawdbv2.PDEContribution{
				ContributorAddressStr: waitingContribution.ContributorAddressStr,
				TokenIDStr:            waitingContribution.TokenIDStr,
				Amount:                matchedNReturnedContrib.ActualWaitingContribAmount,
				TxReqID:               waitingContribution.TxReqID,
			}
			updateWaitingContributionPairToPoolV2(
				beaconHeight,
				existingWaitingContribution,
				incomingWaitingContribution,
				currentPDEState,
			)
			currentPDEState.DeletedWaitingPDEContributions[waitingContribPairKey] = waitingContribution
			delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		}
		pdeStatusContentBytes, err := statedb.GetPDEContributionStatus(
			pdexStateDB,
			rawdbv2.PDEContributionStatusPrefix,
			[]byte(matchedNReturnedContrib.PDEContributionPairID),
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while getting pde contribution status: %+v", err)
			return nil
		}
		if len(pdeStatusContentBytes) == 0 {
			return nil
		}

		var contribStatus metadata.PDEContributionStatus
		err = json.Unmarshal(pdeStatusContentBytes, &contribStatus)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling pde contribution status: %+v", err)
			return nil
		}

		if contribStatus.Status != byte(common.PDEContributionMatchedNReturnedStatus) {
			contribStatus := metadata.PDEContributionStatus{
				Status:             byte(common.PDEContributionMatchedNReturnedStatus),
				TokenID1Str:        matchedNReturnedContrib.TokenIDStr,
				Contributed1Amount: matchedNReturnedContrib.ActualContributedAmount,
				Returned1Amount:    matchedNReturnedContrib.ReturnedContributedAmount,
			}
			contribStatusBytes, _ := json.Marshal(contribStatus)
			err := statedb.TrackPDEContributionStatus(
				pdexStateDB,
				rawdbv2.PDEContributionStatusPrefix,
				[]byte(matchedNReturnedContrib.PDEContributionPairID),
				contribStatusBytes,
			)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while tracking pde contribution status: %+v", err)
				return nil
			}
		} else {
			var contribStatus metadata.PDEContributionStatus
			err := json.Unmarshal(pdeStatusContentBytes, &contribStatus)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while unmarshaling pde contribution status: %+v", err)
				return nil
			}
			contribStatus.TokenID2Str = matchedNReturnedContrib.TokenIDStr
			contribStatus.Contributed2Amount = matchedNReturnedContrib.ActualContributedAmount
			contribStatus.Returned2Amount = matchedNReturnedContrib.ReturnedContributedAmount
			contribStatusBytes, _ := json.Marshal(contribStatus)
			err = statedb.TrackPDEContributionStatus(
				pdexStateDB,
				rawdbv2.PDEContributionStatusPrefix,
				[]byte(matchedNReturnedContrib.PDEContributionPairID),
				contribStatusBytes,
			)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while tracking pde contribution status: %+v", err)
				return nil
			}
		}
	}
	return nil
}

func (blockchain *BlockChain) processPDETrade(pdexStateDB *statedb.StateDB, beaconHeight uint64, instruction []string, currentPDEState *CurrentPDEState) error {
	if len(instruction) != 4 {
		return nil // skip the instruction
	}
	if instruction[2] == common.PDETradeRefundChainStatus {
		contentBytes, err := base64.StdEncoding.DecodeString(instruction[3])
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while decoding content string of pde trade instruction: %+v", err)
			return nil
		}
		var pdeTradeReqAction metadata.PDETradeRequestAction
		err = json.Unmarshal(contentBytes, &pdeTradeReqAction)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling pde trade instruction: %+v", err)
			return nil
		}
		err = statedb.TrackPDEStatus(
			pdexStateDB,
			rawdbv2.PDETradeStatusPrefix,
			pdeTradeReqAction.TxReqID[:],
			byte(common.PDETradeRefundStatus),
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde refund trade status: %+v", err)
		}
		return nil
	}
	var pdeTradeAcceptedContent metadata.PDETradeAcceptedContent
	err := json.Unmarshal([]byte(instruction[3]), &pdeTradeAcceptedContent)
	if err != nil {
		Logger.log.Errorf("WARNING: an error occured while unmarshaling PDETradeAcceptedContent: %+v", err)
		return nil
	}

	pdePoolForPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, pdeTradeAcceptedContent.Token1IDStr, pdeTradeAcceptedContent.Token2IDStr))
	pdePoolForPair, found := currentPDEState.PDEPoolPairs[pdePoolForPairKey]
	if !found || pdePoolForPair == nil {
		Logger.log.Errorf("WARNING: could not find out pdePoolForPair with token ids: %s & %s", pdeTradeAcceptedContent.Token1IDStr, pdeTradeAcceptedContent.Token2IDStr)
		return nil
	}

	if pdeTradeAcceptedContent.Token1PoolValueOperation.Operator == "+" {
		pdePoolForPair.Token1PoolValue += pdeTradeAcceptedContent.Token1PoolValueOperation.Value
		pdePoolForPair.Token2PoolValue -= pdeTradeAcceptedContent.Token2PoolValueOperation.Value
	} else {
		pdePoolForPair.Token1PoolValue -= pdeTradeAcceptedContent.Token1PoolValueOperation.Value
		pdePoolForPair.Token2PoolValue += pdeTradeAcceptedContent.Token2PoolValueOperation.Value
	}
	err = statedb.TrackPDEStatus(
		pdexStateDB,
		rawdbv2.PDETradeStatusPrefix,
		pdeTradeAcceptedContent.RequestedTxID[:],
		byte(common.PDETradeAcceptedStatus),
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while tracking pde accepted trade status: %+v", err)
	}
	return nil
}

func (blockchain *BlockChain) processPDECrossPoolTrade(pdexStateDB *statedb.StateDB, beaconHeight uint64, instruction []string, currentPDEState *CurrentPDEState) error {
	if len(instruction) != 4 {
		return nil // skip the instruction
	}
	if instruction[2] == common.PDECrossPoolTradeFeeRefundChainStatus ||
		instruction[2] == common.PDECrossPoolTradeSellingTokenRefundChainStatus {
		contentBytes := []byte(instruction[3])
		var pdeRefundCrossPoolTrade metadata.PDERefundCrossPoolTrade
		err := json.Unmarshal(contentBytes, &pdeRefundCrossPoolTrade)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling pde refund cross pool trade instruction: %+v", err)
			return nil
		}
		err = statedb.TrackPDEStatus(
			pdexStateDB,
			rawdbv2.PDETradeStatusPrefix,
			pdeRefundCrossPoolTrade.TxReqID[:],
			byte(common.PDECrossPoolTradeRefundStatus),
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde refund trade status: %+v", err)
		}
		return nil
	}
	// trade accepted
	var pdeTradeAcceptedContents []metadata.PDECrossPoolTradeAcceptedContent
	err := json.Unmarshal([]byte(instruction[3]), &pdeTradeAcceptedContents)
	if err != nil {
		Logger.log.Errorf("WARNING: an error occured while unmarshaling PDETradeAcceptedContents: %+v", err)
		return nil
	}

	if len(pdeTradeAcceptedContents) == 0 {
		Logger.log.Error("WARNING: There is no pde cross pool trade accepted content.")
		return nil
	}

	for _, pdeTradeAcceptedContent := range pdeTradeAcceptedContents {
		pdePoolForPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, pdeTradeAcceptedContent.Token1IDStr, pdeTradeAcceptedContent.Token2IDStr))
		pdePoolForPair, found := currentPDEState.PDEPoolPairs[pdePoolForPairKey]
		if !found || pdePoolForPair == nil {
			Logger.log.Errorf("WARNING: could not find out pdePoolForPair with token ids: %s & %s", pdeTradeAcceptedContent.Token1IDStr, pdeTradeAcceptedContent.Token2IDStr)
			return nil
		}

		if pdeTradeAcceptedContent.Token1PoolValueOperation.Operator == "+" {
			pdePoolForPair.Token1PoolValue += pdeTradeAcceptedContent.Token1PoolValueOperation.Value
			pdePoolForPair.Token2PoolValue -= pdeTradeAcceptedContent.Token2PoolValueOperation.Value
		} else {
			pdePoolForPair.Token1PoolValue -= pdeTradeAcceptedContent.Token1PoolValueOperation.Value
			pdePoolForPair.Token2PoolValue += pdeTradeAcceptedContent.Token2PoolValueOperation.Value
		}
	}
	err = statedb.TrackPDEStatus(
		pdexStateDB,
		rawdbv2.PDETradeStatusPrefix,
		pdeTradeAcceptedContents[0].RequestedTxID[:],
		byte(common.PDECrossPoolTradeAcceptedStatus),
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while tracking pde accepted trade status: %+v", err)
	}
	return nil
}

func (blockchain *BlockChain) processPDEWithdrawal(pdexStateDB *statedb.StateDB, beaconHeight uint64, instruction []string, currentPDEState *CurrentPDEState) error {
	if len(instruction) != 4 {
		return nil // skip the instruction
	}
	if instruction[2] == common.PDEWithdrawalRejectedChainStatus {
		contentBytes, err := base64.StdEncoding.DecodeString(instruction[3])
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while decoding content string of pde withdrawal action: %+v", err)
			return nil
		}
		var pdeWithdrawalRequestAction metadata.PDEWithdrawalRequestAction
		err = json.Unmarshal(contentBytes, &pdeWithdrawalRequestAction)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while unmarshaling pde withdrawal request action: %+v", err)
			return nil
		}
		err = statedb.TrackPDEStatus(
			pdexStateDB,
			rawdbv2.PDEWithdrawalStatusPrefix,
			pdeWithdrawalRequestAction.TxReqID[:],
			byte(common.PDEWithdrawalRejectedStatus),
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde rejected withdrawal status: %+v", err)
		}
		return nil
	}

	var wdAcceptedContent metadata.PDEWithdrawalAcceptedContent
	err := json.Unmarshal([]byte(instruction[3]), &wdAcceptedContent)
	if err != nil {
		Logger.log.Errorf("WARNING: an error occured while unmarshaling PDEWithdrawalAcceptedContent: %+v", err)
		return nil
	}

	// update pde pool pair
	pdePoolForPairKey := string(rawdbv2.BuildPDEPoolForPairKey(
		beaconHeight,
		wdAcceptedContent.PairToken1IDStr,
		wdAcceptedContent.PairToken2IDStr,
	))
	pdePoolForPair, found := currentPDEState.PDEPoolPairs[pdePoolForPairKey]
	if !found || pdePoolForPair == nil {
		Logger.log.Errorf("WARNING: could not find out pdePoolForPair with token ids: %s & %s", wdAcceptedContent.PairToken1IDStr, wdAcceptedContent.PairToken2IDStr)
		return nil
	}
	if pdePoolForPair.Token1IDStr == wdAcceptedContent.WithdrawalTokenIDStr {
		pdePoolForPair.Token1PoolValue -= wdAcceptedContent.DeductingPoolValue
	} else {
		pdePoolForPair.Token2PoolValue -= wdAcceptedContent.DeductingPoolValue
	}

	// update pde shares
	deductSharesForWithdrawal(
		beaconHeight,
		wdAcceptedContent.PairToken1IDStr, wdAcceptedContent.PairToken2IDStr,
		wdAcceptedContent.WithdrawerAddressStr,
		wdAcceptedContent.DeductingShares,
		currentPDEState,
	)

	err = statedb.TrackPDEStatus(
		pdexStateDB,
		rawdbv2.PDEWithdrawalStatusPrefix,
		wdAcceptedContent.TxReqID[:],
		byte(common.PDEWithdrawalAcceptedStatus),
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while tracking pde accepted withdrawal status: %+v", err)
	}
	return nil
}

func storePDEPoolForPair(
	pdePoolForPairKey string,
	token1IDStr string,
	token1PoolValue uint64,
	token2IDStr string,
	token2PoolValue uint64,
	currentPDEState *CurrentPDEState,
) {
	pdePoolForPair := &rawdbv2.PDEPoolForPair{
		Token1IDStr:     token1IDStr,
		Token1PoolValue: token1PoolValue,
		Token2IDStr:     token2IDStr,
		Token2PoolValue: token2PoolValue,
	}
	currentPDEState.PDEPoolPairs[pdePoolForPairKey] = pdePoolForPair
}

func deductSharesForWithdrawal(
	beaconHeight uint64,
	token1IDStr string,
	token2IDStr string,
	withdrawerAddressStr string,
	amt uint64,
	currentPDEState *CurrentPDEState,
) {
	pdeShareKey := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, token1IDStr, token2IDStr, withdrawerAddressStr))
	adjustingAmt := uint64(0)
	currentAmt, found := currentPDEState.PDEShares[pdeShareKey]
	if found && amt <= currentAmt {
		adjustingAmt = currentAmt - amt
	}
	currentPDEState.PDEShares[pdeShareKey] = adjustingAmt
}

func (blockchain *BlockChain) processPDETradingFeesDistribution(
	pdexStateDB *statedb.StateDB,
	beaconHeight uint64,
	instruction []string,
	currentPDEState *CurrentPDEState,
) error {
	if len(instruction) != 4 {
		return nil // skip the instruction
	}
	var feesForContributorsByPair []*tradingFeeForContributorByPair
	err := json.Unmarshal([]byte(instruction[3]), &feesForContributorsByPair)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling trading fees for contribution by pair: %+v", err)
		return nil
	}

	pdeTradingFees := currentPDEState.PDETradingFees
	for _, item := range feesForContributorsByPair {
		tradingFeeKey := string(rawdbv2.BuildPDETradingFeeKey(
			beaconHeight,
			item.Token1IDStr,
			item.Token2IDStr,
			item.ContributorAddressStr,
		))
		pdeTradingFees[tradingFeeKey] += item.FeeAmt
	}
	return nil
}

func (blockchain *BlockChain) processPDEFeeWithdrawal(
	pdexStateDB *statedb.StateDB,
	beaconHeight uint64,
	instruction []string,
	currentPDEState *CurrentPDEState,
) error {
	if len(instruction) != 4 {
		return nil // skip the instruction
	}
	contentStr := instruction[3]
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde fee withdrawal action: %+v", err)
		return nil
	}
	var pdeFeeWithdrawalRequestAction metadata.PDEFeeWithdrawalRequestAction
	err = json.Unmarshal(contentBytes, &pdeFeeWithdrawalRequestAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde fee withdrawal request action: %+v", err)
		return nil
	}

	if instruction[2] == common.PDEFeeWithdrawalRejectedChainStatus {
		err = statedb.TrackPDEStatus(
			pdexStateDB,
			rawdbv2.PDEFeeWithdrawalStatusPrefix,
			pdeFeeWithdrawalRequestAction.TxReqID[:],
			byte(common.PDEFeeWithdrawalRejectedStatus),
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde rejected withdrawal status: %+v", err)
		}
		return nil
	}
	// fee withdrawal accepted
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
		Logger.log.Warnf("WARN: Could not withdraw trading fee due to insufficient amount or not existed trading fee key (%s)", tradingFeeKey)
		err = statedb.TrackPDEStatus(
			pdexStateDB,
			rawdbv2.PDEFeeWithdrawalStatusPrefix,
			pdeFeeWithdrawalRequestAction.TxReqID[:],
			byte(common.PDEFeeWithdrawalRejectedStatus),
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking pde rejected withdrawal status: %+v", err)
		}
		return nil
	}
	pdeTradingFees[tradingFeeKey] -= wdMeta.WithdrawalFeeAmt
	err = statedb.TrackPDEStatus(
		pdexStateDB,
		rawdbv2.PDEFeeWithdrawalStatusPrefix,
		pdeFeeWithdrawalRequestAction.TxReqID[:],
		byte(common.PDEFeeWithdrawalAcceptedStatus),
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while tracking pde rejected withdrawal status: %+v", err)
	}
	return nil
}
