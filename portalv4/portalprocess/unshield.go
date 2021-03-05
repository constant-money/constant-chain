package portalprocess

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"github.com/incognitochain/incognito-chain/portalv4"
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"
	pv4Meta "github.com/incognitochain/incognito-chain/portalv4/metadata"
	"github.com/incognitochain/incognito-chain/portalv4/portaltokens"
)

/* =======
Portal Unshield Request Processor
======= */
type portalUnshieldRequestProcessor struct {
	*portalInstProcessor
}

func (p *portalUnshieldRequestProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalUnshieldRequestProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalUnshieldRequestProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal unshield request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while decoding content string of portal unshield request action: %+v", err)
	}
	var actionData pv4Meta.PortalUnshieldRequestAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal unshield request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while unmarshal portal unshield request action: %+v", err)
	}

	optionalData := make(map[string]interface{})

	// Get unshield request with unshieldID from stateDB
	unshieldRequestStatusBytes, err := statedb.GetPortalUnshieldRequestStatus(stateDB, actionData.TxReqID.String())
	if err != nil {
		Logger.log.Errorf("Unshield request: an error occurred while get unshield request by id from DB: %+v", err)
		return nil, fmt.Errorf("Unshield request: an error occurred while get unshield request by id from DB: %+v", err)
	}

	optionalData["isExistUnshieldID"] = len(unshieldRequestStatusBytes) > 0
	return optionalData, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildUnshieldRequestInst(
	tokenID string,
	redeemAmount uint64,
	incAddressStr string,
	remoteAddress string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	unshieldRequestContent := pv4Meta.PortalUnshieldRequestContent{
		TokenID:        tokenID,
		UnshieldAmount: redeemAmount,
		IncAddressStr:  incAddressStr,
		RemoteAddress:  remoteAddress,
		TxReqID:        txReqID,
		ShardID:        shardID,
	}
	unshieldRequestContentBytes, _ := json.Marshal(unshieldRequestContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(unshieldRequestContentBytes),
	}
}

func (p *portalUnshieldRequestProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalV4State *CurrentPortalV4State,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portalv4.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal unshield request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while decoding content string of portal unshield request action: %+v", err)
	}
	var actionData pv4Meta.PortalUnshieldRequestAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal unshield request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while unmarshal portal unshield request action: %+v", err)
	}

	if currentPortalV4State == nil {
		Logger.log.Warn("WARN - [Unshield Request]: Current Portal state V4 is null.")
		return [][]string{}, nil
	}

	meta := actionData.Meta
	rejectInst := buildUnshieldRequestInst(
		meta.TokenID,
		meta.UnshieldAmount,
		meta.IncAddressStr,
		meta.RemoteAddress,
		meta.Type,
		actionData.ShardID,
		actionData.TxReqID,
		pCommon.PortalRequestRejectedChainStatus,
	)

	unshieldID := actionData.TxReqID.String()
	tokenID := meta.TokenID

	// check unshieldID is existed waitingUnshield list or not
	wUnshieldReqsByTokenID := currentPortalV4State.WaitingUnshieldRequests[tokenID]
	if wUnshieldReqsByTokenID != nil {
		keyWaitingUnshieldRequestStr := statedb.GenerateWaitingUnshieldRequestObjectKey(tokenID, unshieldID).String()
		waitingUnshieldRequest := wUnshieldReqsByTokenID[keyWaitingUnshieldRequestStr]
		if waitingUnshieldRequest != nil {
			Logger.log.Errorf("[Unshield Request] unshieldID is existed in waiting unshield requests list %v\n", unshieldID)
			return [][]string{rejectInst}, nil
		}
	}

	// check unshieldID is existed in db or not
	if optionalData == nil {
		Logger.log.Errorf("[Unshield Request] optionalData is null")
		return [][]string{rejectInst}, nil
	}
	isExist, ok := optionalData["isExistUnshieldID"].(bool)
	if !ok {
		Logger.log.Errorf("[Unshield Request] optionalData isExistUnshieldID is invalid")
		return [][]string{rejectInst}, nil
	}
	if isExist {
		Logger.log.Errorf("[Unshield Request] UnshieldID exist in db %v", unshieldID)
		return [][]string{rejectInst}, nil
	}

	// build accept instruction
	newInst := buildUnshieldRequestInst(
		meta.TokenID,
		meta.UnshieldAmount,
		meta.IncAddressStr,
		meta.RemoteAddress,
		meta.Type,
		actionData.ShardID,
		actionData.TxReqID,
		pCommon.PortalRequestAcceptedChainStatus,
	)

	// add new waiting unshield request to waiting list
	UpdatePortalStateAfterUnshieldRequest(currentPortalV4State, unshieldID, meta.TokenID, meta.RemoteAddress, meta.UnshieldAmount, beaconHeight)

	return [][]string{newInst}, nil
}

func (p *portalUnshieldRequestProcessor) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalV4State *CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
) error {
	if currentPortalV4State == nil {
		Logger.log.Errorf("current portal state is nil")
		return nil
	}

	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	// unmarshal instructions content
	var actionData pv4Meta.PortalUnshieldRequestContent
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		// add new waiting unshield request to waiting list
		UpdatePortalStateAfterUnshieldRequest(currentPortalV4State, actionData.TxReqID.String(), actionData.TokenID, actionData.RemoteAddress, actionData.UnshieldAmount, beaconHeight)

		// track status of unshield request by unshieldID (txID)
		unshieldRequestStatus := pv4Meta.PortalUnshieldRequestStatus{
			IncAddressStr:  actionData.IncAddressStr,
			RemoteAddress:  actionData.RemoteAddress,
			TokenID:        actionData.TokenID,
			UnshieldAmount: actionData.UnshieldAmount,
			TxHash:         actionData.TxReqID.String(),
			Status:         pv4Common.PortalUnshieldReqWaitingStatus,
		}
		unshieldRequestStatusBytes, _ := json.Marshal(unshieldRequestStatus)
		err := statedb.StorePortalUnshieldRequestStatus(
			stateDB,
			actionData.TxReqID.String(),
			unshieldRequestStatusBytes)
		if err != nil {
			Logger.log.Errorf("[processPortalRedeemRequest] Error when storing status of redeem request by redeemID: %v\n", err)
			return nil
		}

		//todo: review
		// update bridge/portal token info
		incTokenID, err := common.Hash{}.NewHashFromStr(actionData.TokenID)
		if err != nil {
			Logger.log.Errorf("ERROR: Can not new hash from porting incTokenID: %+v", err)
			return nil
		}
		updatingInfo, found := updatingInfoByTokenID[*incTokenID]
		if found {
			updatingInfo.DeductAmt += actionData.UnshieldAmount
		} else {
			updatingInfo = bMeta.UpdatingInfo{
				CountUpAmt:      0,
				DeductAmt:       actionData.UnshieldAmount,
				TokenID:         *incTokenID,
				ExternalTokenID: nil,
				IsCentralized:   false,
			}
		}
		updatingInfoByTokenID[*incTokenID] = updatingInfo
	}

	return nil
}

/* =======
Portal Replacement Processor
======= */

type portalReplacementFeeRequestProcessor struct {
	*portalInstProcessor
}

func (p *portalReplacementFeeRequestProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalReplacementFeeRequestProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalReplacementFeeRequestProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Replace fee request: an error occurred while decoding content string of replace fee unshield request action: %+v", err)
		return nil, fmt.Errorf("Replace fee request: an error occurred while decoding content string of replace fee unshield request action: %+v", err)
	}

	var actionData pv4Meta.PortalReplacementFeeRequestAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("Replace fee: an error occurred while unmarshal replace fee unshield request action: %+v", err)
		return nil, fmt.Errorf("Replace fee: an error occurred while unmarshal replace fee unshield request action: %+v", err)
	}

	unshieldBatchBytes, err := statedb.GetPortalBatchUnshieldRequestStatus(stateDB, actionData.Meta.BatchID)
	if err != nil {
		Logger.log.Error("Can not get unshield batch: %v", err)
		return nil, err
	}

	var processedUnshieldRequestBatch pv4Meta.PortalUnshieldRequestBatchStatus
	err = json.Unmarshal(unshieldBatchBytes, &processedUnshieldRequestBatch)
	if err != nil {
		Logger.log.Errorf("Replace fee: an error occurred while unmarshal processedUnshieldRequestBatch status: %+v", err)
		return nil, fmt.Errorf("Replace fee: an error occurred while unmarshal processedUnshieldRequestBatch status: %+v", err)
	}

	var outputs []*portaltokens.OutputTx
	for _, v := range processedUnshieldRequestBatch.UnshieldIDs {
		unshieldBytes, err := statedb.GetPortalUnshieldRequestStatus(stateDB, v)
		if err != nil {
			Logger.log.Error("Can not get unshield batch: %v", err)
			return nil, err
		}
		var portalUnshieldRequestStatus pv4Meta.PortalUnshieldRequestStatus
		err = json.Unmarshal(unshieldBytes, &portalUnshieldRequestStatus)
		if err != nil {
			Logger.log.Errorf("Replace fee: an error occurred while unmarshal PortalUnshieldRequestStatus: %+v", err)
			return nil, fmt.Errorf("Replace fee: an error occurred while unmarshal PortalUnshieldRequestStatus: %+v", err)
		}
		outputs = append(outputs, &portaltokens.OutputTx{ReceiverAddress: portalUnshieldRequestStatus.RemoteAddress, Amount: portalUnshieldRequestStatus.UnshieldAmount})
	}

	optionalData := make(map[string]interface{})
	optionalData["outputs"] = outputs
	return optionalData, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildReplacementFeeRequestInst(
	tokenID string,
	incAddressStr string,
	fee uint,
	batchID string,
	metaType int,
	shardID byte,
	externalRawTx string,
	txReqID common.Hash,
	status string,
) []string {
	replacementRequestContent := pv4Meta.PortalReplacementFeeRequestContent{
		TokenID:       tokenID,
		IncAddressStr: incAddressStr,
		Fee:           fee,
		BatchID:       batchID,
		TxReqID:       txReqID,
		ExternalRawTx: externalRawTx,
	}
	replacementRequestContentBytes, _ := json.Marshal(replacementRequestContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(replacementRequestContentBytes),
	}
}

func (p *portalReplacementFeeRequestProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalV4State *CurrentPortalV4State,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portalv4.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal replacement fee request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while decoding content string of portal replacement fee request action: %+v", err)
	}
	var actionData pv4Meta.PortalReplacementFeeRequestAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal replacement fee request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while unmarshal portal replacement fee request action: %+v", err)
	}

	if currentPortalV4State == nil {
		Logger.log.Warn("WARN - [Unshield Request]: Current Portal state V4 is null.")
		return [][]string{}, nil
	}

	meta := actionData.Meta
	rejectInst := buildReplacementFeeRequestInst(
		meta.TokenID,
		meta.IncAddressStr,
		meta.Fee,
		meta.BatchID,
		meta.Type,
		actionData.ShardID,
		"",
		actionData.TxReqID,
		pCommon.PortalRequestRejectedChainStatus,
	)

	tokenIDStr := meta.TokenID
	keyUnshieldBatch := statedb.GenerateProcessedUnshieldRequestBatchObjectKey(tokenIDStr, meta.BatchID).String()
	unshieldBatch, ok := currentPortalV4State.ProcessedUnshieldRequests[tokenIDStr][keyUnshieldBatch]
	if !ok {
		Logger.log.Errorf("Error: Replace a non-exist unshield batch with tokenID: %v, batchid : %v.", tokenIDStr, meta.BatchID)
		return [][]string{rejectInst}, nil
	}
	latestBeaconHeight := GetMaxKeyValue(unshieldBatch.GetExternalFees())
	if latestBeaconHeight == 0 || !bc.CheckBlockTimeIsReachedByBeaconHeight(beaconHeight, latestBeaconHeight, portalParams.TimeSpaceForFeeReplacement) {
		Logger.log.Errorf("Error: Can not replace unshield batch with tokenID: %v, batchid : %v.", tokenIDStr, meta.BatchID)
		return [][]string{rejectInst}, nil
	}
	latestFee := unshieldBatch.GetExternalFees()[latestBeaconHeight]

	if meta.Fee < latestFee || meta.Fee-latestFee > portalParams.MaxFeeForEachStep {
		Logger.log.Errorf("Error: Replace unshield batch with invalid fee: %v", meta.Fee)
		return [][]string{rejectInst}, nil
	}

	portalTokenProcessor := portalParams.PortalTokens[tokenIDStr]
	multisigAddress := portalParams.MultiSigAddresses[tokenIDStr]
	if unshieldBatch.GetUTXOs() == nil || unshieldBatch.GetUTXOs()[multisigAddress] == nil {
		Logger.log.Errorf("Error: Can not get utxos from unshield batch with multisig address: %v", multisigAddress)
		return [][]string{rejectInst}, nil
	}
	hexRawExtTxStr, _, err := portalTokenProcessor.CreateRawExternalTx(unshieldBatch.GetUTXOs()[multisigAddress], optionalData["outputs"].([]*portaltokens.OutputTx), uint64(meta.Fee), meta.BatchID, bc)

	// build accept instruction
	newInst := buildReplacementFeeRequestInst(
		meta.TokenID,
		meta.IncAddressStr,
		meta.Fee,
		meta.BatchID,
		meta.Type,
		actionData.ShardID,
		hexRawExtTxStr,
		actionData.TxReqID,
		pCommon.PortalRequestAcceptedChainStatus,
	)

	// add new waiting unshield request to waiting list
	UpdatePortalStateAfterReplaceFeedRequest(currentPortalV4State, unshieldBatch, beaconHeight, meta.Fee, tokenIDStr, meta.BatchID)

	return [][]string{newInst}, nil
}

func (p *portalReplacementFeeRequestProcessor) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalV4State *CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
) error {
	if currentPortalV4State == nil {
		Logger.log.Errorf("current portal state is nil")
		return nil
	}

	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	// unmarshal instructions content
	var actionData pv4Meta.PortalReplacementFeeRequestContent
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	var unshieldBatchRequestStatus pv4Meta.PortalReplacementFeeRequestStatus

	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		// update unshield batch
		keyUnshieldBatch := statedb.GenerateProcessedUnshieldRequestBatchObjectKey(actionData.TokenID, actionData.BatchID).String()
		unshieldBatch := currentPortalV4State.ProcessedUnshieldRequests[actionData.TokenID][keyUnshieldBatch]
		UpdatePortalStateAfterReplaceFeedRequest(currentPortalV4State, unshieldBatch, beaconHeight, actionData.Fee, actionData.TokenID, actionData.BatchID)

		// track status of unshield batch request by batchID
		unshieldBatchRequestStatus = pv4Meta.PortalReplacementFeeRequestStatus{
			IncAddressStr: actionData.IncAddressStr,
			TokenID:       actionData.TokenID,
			BatchID:       actionData.BatchID,
			Fee:           actionData.Fee,
			ExternalRawTx: actionData.ExternalRawTx,
			BeaconHeight:  beaconHeight,
			TxHash:        actionData.TxReqID.String(),
			Status:        pCommon.PortalRequestAcceptedStatus,
		}
	} else if reqStatus == pCommon.PortalRequestRejectedChainStatus {

		unshieldBatchRequestStatus = pv4Meta.PortalReplacementFeeRequestStatus{
			IncAddressStr: actionData.IncAddressStr,
			TokenID:       actionData.TokenID,
			BatchID:       actionData.BatchID,
			ExternalRawTx: actionData.ExternalRawTx,
			BeaconHeight:  beaconHeight,
			Fee:           actionData.Fee,
			TxHash:        actionData.TxReqID.String(),
			Status:        pCommon.PortalRequestRejectedStatus,
		}
	} else {
		return nil
	}
	unshieldBatchStatusBytes, _ := json.Marshal(unshieldBatchRequestStatus)
	err = statedb.StorePortalUnshieldBatchReplacementRequestStatus(
		stateDB,
		actionData.TxReqID.String(),
		unshieldBatchStatusBytes)
	if err != nil {
		Logger.log.Errorf("[processPortalReplacementRequest] Error when storing status of replacement request: %v\n", err)
		return nil
	}

	return nil
}

/* =======
Portal Submit external unshield tx confirmed Processor V4
======= */

type portalSubmitConfirmedTxProcessor struct {
	*portalInstProcessor
}

func (p *portalSubmitConfirmedTxProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalSubmitConfirmedTxProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalSubmitConfirmedTxProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("SubmitConfirmed request: an error occurred while decoding content string of SubmitConfirmed unshield request action: %+v", err)
		return nil, fmt.Errorf("Replace fee request: an error occurred while decoding content string of SubmitConfirmed unshield request action: %+v", err)
	}

	var actionData pv4Meta.PortalSubmitConfirmedTxAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("SubmitConfirmed request: an error occurred while unmarshal SubmitConfirmed unshield request action: %+v", err)
		return nil, fmt.Errorf("SubmitConfirmed request: an error occurred while unmarshal SubmitConfirmed unshield request action: %+v", err)
	}

	unshieldBatchBytes, err := statedb.GetPortalBatchUnshieldRequestStatus(stateDB, actionData.Meta.BatchID)
	if err != nil {
		Logger.log.Error("Can not get unshield batch: %v", err)
		return nil, err
	}

	var processedUnshieldRequestBatch pv4Meta.PortalUnshieldRequestBatchStatus
	err = json.Unmarshal(unshieldBatchBytes, &processedUnshieldRequestBatch)
	if err != nil {
		Logger.log.Errorf("SubmitConfirmed request: an error occurred while unmarshal processedUnshieldRequestBatch status: %+v", err)
		return nil, fmt.Errorf("SubmitConfirmed request: an error occurred while unmarshal processedUnshieldRequestBatch status: %+v", err)
	}

	outputs := make(map[string]uint64, 0)
	for _, v := range processedUnshieldRequestBatch.UnshieldIDs {
		unshieldBytes, err := statedb.GetPortalUnshieldRequestStatus(stateDB, v)
		if err != nil {
			Logger.log.Error("Can not get unshield batch: %v", err)
			return nil, err
		}
		var portalUnshieldRequestStatus pv4Meta.PortalUnshieldRequestStatus
		err = json.Unmarshal(unshieldBytes, &portalUnshieldRequestStatus)
		if err != nil {
			Logger.log.Errorf("SubmitConfirmed: an error occurred while unmarshal PortalUnshieldRequestStatus: %+v", err)
			return nil, fmt.Errorf("SubmitConfirmed: an error occurred while unmarshal PortalUnshieldRequestStatus: %+v", err)
		}
		outputs[portalUnshieldRequestStatus.RemoteAddress] = portalUnshieldRequestStatus.UnshieldAmount
	}

	optionalData := make(map[string]interface{})
	optionalData["outputs"] = outputs

	return optionalData, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildSubmitConfirmedTxInst(
	tokenID string,
	unshieldProof string,
	utxos []*statedb.UTXO,
	batchID string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	replacementRequestContent := pv4Meta.PortalSubmitConfirmedTxContent{
		TokenID:       tokenID,
		UnshieldProof: unshieldProof,
		UTXOs:         utxos,
		BatchID:       batchID,
		TxReqID:       txReqID,
	}
	replacementRequestContentBytes, _ := json.Marshal(replacementRequestContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(replacementRequestContentBytes),
	}
}

func (p *portalSubmitConfirmedTxProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalV4State *CurrentPortalV4State,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portalv4.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal replacement fee request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while decoding content string of portal replacement fee request action: %+v", err)
	}
	var actionData pv4Meta.PortalSubmitConfirmedTxAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal replacement fee request action: %+v", err)
		return nil, fmt.Errorf("ERROR: an error occured while unmarshal portal replacement fee request action: %+v", err)
	}

	if currentPortalV4State == nil {
		Logger.log.Warn("WARN - [Unshield Request]: Current Portal state V4 is null.")
		return [][]string{}, nil
	}

	meta := actionData.Meta
	listUTXO := []*statedb.UTXO{}
	rejectInst := buildSubmitConfirmedTxInst(
		meta.TokenID,
		meta.UnshieldProof,
		listUTXO,
		meta.BatchID,
		meta.Type,
		actionData.ShardID,
		actionData.TxReqID,
		pCommon.PortalRequestRejectedChainStatus,
	)

	tokenIDStr := meta.TokenID
	batchIDStr := meta.BatchID
	keyUnshieldBatch := statedb.GenerateProcessedUnshieldRequestBatchObjectKey(tokenIDStr, batchIDStr).String()
	if currentPortalV4State.ProcessedUnshieldRequests == nil ||
		currentPortalV4State.ProcessedUnshieldRequests[tokenIDStr] == nil {
		Logger.log.Errorf("Error: currentPortalV4State.ProcessedUnshieldRequests not initialized yet")
		return [][]string{rejectInst}, nil
	}
	unshieldBatch, ok := currentPortalV4State.ProcessedUnshieldRequests[tokenIDStr][keyUnshieldBatch]
	if !ok {
		Logger.log.Errorf("Error: Submit non-exist unshield external transaction with tokenID: %v, batchid : %v.", tokenIDStr, batchIDStr)
		return [][]string{rejectInst}, nil
	}
	portalTokenProcessor := portalParams.PortalTokens[meta.TokenID]
	if portalTokenProcessor == nil {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		return [][]string{rejectInst}, nil
	}

	expectedMultisigAddress := portalParams.MultiSigAddresses[tokenIDStr]
	outputs := optionalData["outputs"].(map[string]uint64)
	if unshieldBatch.GetUTXOs() == nil || unshieldBatch.GetUTXOs()[expectedMultisigAddress] == nil {
		Logger.log.Errorf("Error submit external confirmed tx: can not get utxos of wallet address: %v", expectedMultisigAddress)
		return [][]string{rejectInst}, nil
	}
	isValid, listUTXO, err := portalTokenProcessor.ParseAndVerifyUnshieldProof(meta.UnshieldProof, bc, batchIDStr, expectedMultisigAddress, outputs, unshieldBatch.GetUTXOs()[expectedMultisigAddress])
	if !isValid || err != nil {
		Logger.log.Errorf("Unshield Proof is invalid")
		return [][]string{rejectInst}, nil
	}

	// build accept instruction
	newInst := buildSubmitConfirmedTxInst(
		meta.TokenID,
		meta.UnshieldProof,
		listUTXO,
		meta.BatchID,
		meta.Type,
		actionData.ShardID,
		actionData.TxReqID,
		pCommon.PortalRequestAcceptedChainStatus,
	)

	// remove unshield being processed and update status
	UpdatePortalStateAfterSubmitConfirmedTx(currentPortalV4State, tokenIDStr, keyUnshieldBatch)
	if len(listUTXO) > 0 {
		UpdatePortalStateUTXOs(currentPortalV4State, tokenIDStr, listUTXO)
	}

	return [][]string{newInst}, nil
}

func (p *portalSubmitConfirmedTxProcessor) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalV4State *CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
) error {
	if currentPortalV4State == nil {
		Logger.log.Errorf("current portal state is nil")
		return nil
	}

	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	// unmarshal instructions content
	var actionData pv4Meta.PortalSubmitConfirmedTxContent
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	var portalSubmitConfirmedStatus pv4Meta.PortalSubmitConfirmedTxStatus

	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		// update unshield batch
		keyUnshieldBatch := statedb.GenerateProcessedUnshieldRequestBatchObjectKey(actionData.TokenID, actionData.BatchID).String()
		unshieldRequests := currentPortalV4State.ProcessedUnshieldRequests[actionData.TokenID][keyUnshieldBatch].GetUnshieldRequests()
		UpdatePortalStateAfterSubmitConfirmedTx(currentPortalV4State, actionData.TokenID, keyUnshieldBatch)
		if len(actionData.UTXOs) > 0 {
			UpdatePortalStateUTXOs(currentPortalV4State, actionData.TokenID, actionData.UTXOs)
		}
		// track status of unshield batch request by batchID
		portalSubmitConfirmedStatus = pv4Meta.PortalSubmitConfirmedTxStatus{
			TokenID:       actionData.TokenID,
			BatchID:       actionData.BatchID,
			UTXOs:         actionData.UTXOs,
			UnshieldProof: actionData.UnshieldProof,
			TxHash:        actionData.TxReqID.String(),
			Status:        pCommon.PortalRequestAcceptedStatus,
		}

		// update unshield list to completed
		for _, v := range unshieldRequests {
			unshieldRequestBytes, err := statedb.GetPortalUnshieldRequestStatus(stateDB, v)
			if err != nil {
				Logger.log.Errorf("[processPortalSubmitConfirmedTx] Error when query unshield tx by unshieldID: %v\n err: %v", v, err)
				return nil
			}
			var unshielRequest pv4Meta.PortalUnshieldRequestStatus
			err = json.Unmarshal(unshieldRequestBytes, &unshielRequest)
			if err != nil {
				Logger.log.Errorf("Can not unmarshal instruction content %v - Error %v\n", unshieldRequestBytes, err)
				return nil
			}

			unshieldRequestStatus := pv4Meta.PortalUnshieldRequestStatus{
				IncAddressStr:  unshielRequest.IncAddressStr,
				RemoteAddress:  unshielRequest.RemoteAddress,
				TokenID:        unshielRequest.TokenID,
				UnshieldAmount: unshielRequest.UnshieldAmount,
				TxHash:         unshielRequest.TxHash,
				Status:         pv4Common.PortalUnshieldReqCompletedStatus,
			}
			redeemRequestStatusBytes, _ := json.Marshal(unshieldRequestStatus)
			err = statedb.StorePortalUnshieldRequestStatus(
				stateDB,
				actionData.TxReqID.String(),
				redeemRequestStatusBytes)
			if err != nil {
				Logger.log.Errorf("[processPortalSubmitConfirmedTx] Error store completed unshield request unshieldID: %v\n err: %v", v, err)
				return nil
			}
		}

	} else if reqStatus == pCommon.PortalRequestRejectedChainStatus {
		portalSubmitConfirmedStatus = pv4Meta.PortalSubmitConfirmedTxStatus{
			TokenID:       actionData.TokenID,
			BatchID:       actionData.BatchID,
			UTXOs:         actionData.UTXOs,
			UnshieldProof: actionData.UnshieldProof,
			TxHash:        actionData.TxReqID.String(),
			Status:        pCommon.PortalRequestRejectedStatus,
		}
	} else {
		return nil
	}
	portalSubmitConfirmedStatusBytes, _ := json.Marshal(portalSubmitConfirmedStatus)
	err = statedb.StorePortalSubmitConfirmedTxRequestStatus(
		stateDB,
		actionData.TxReqID.String(),
		portalSubmitConfirmedStatusBytes)
	if err != nil {
		Logger.log.Errorf("[processPortalReplacementRequest] Error when storing status of replacement request: %v\n", err)
		return nil
	}

	return nil
}
