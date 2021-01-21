package portalprocess

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/portal"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	portalMeta "github.com/incognitochain/incognito-chain/portal/metadata"
)

/* =======
Portal Porting Request Processor
======= */
type portalPortingRequestProcessor struct {
	*portalInstProcessor
}

func (p *portalPortingRequestProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalPortingRequestProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalPortingRequestProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while decoding content string of portal porting request action: %+v", err)
		return nil, fmt.Errorf("Porting request: an error occurred while decoding content string of portal porting request action: %+v", err)
	}

	var actionData portalMeta.PortalUserRegisterActionV3
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while unmarshal portal porting request action: %+v", err)
		return nil, fmt.Errorf("Porting request: an error occurred while unmarshal portal porting request action: %+v", err)
	}

	optionalData := make(map[string]interface{})

	// Get porting request with uniqueID from stateDB
	isExistPortingID, err := statedb.IsPortingRequestIdExist(stateDB, []byte(actionData.Meta.UniqueRegisterId))
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while get porting request Id from DB: %+v", err)
		return nil, fmt.Errorf("Porting request: an error occurred while get porting request Id from DB: %+v", err)
	}

	optionalData["isExistPortingID"] = isExistPortingID
	return optionalData, nil
}

func buildRequestPortingInst(
	metaType int,
	shardID byte,
	reqStatus string,
	uniqueRegisterId string,
	incogAddressStr string,
	pTokenId string,
	registerAmount uint64,
	portingFee uint64,
	custodian []*statedb.MatchingPortingCustodianDetail,
	txReqID common.Hash,
	shardHeight uint64,
) []string {
	portingRequestContent := portalMeta.PortalPortingRequestContent{
		UniqueRegisterId: uniqueRegisterId,
		IncogAddressStr:  incogAddressStr,
		PTokenId:         pTokenId,
		RegisterAmount:   registerAmount,
		PortingFee:       portingFee,
		Custodian:        custodian,
		TxReqID:          txReqID,
		ShardID:          shardID,
		ShardHeight:      shardHeight,
	}

	portingRequestContentBytes, _ := json.Marshal(portingRequestContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		reqStatus,
		string(portingRequestContentBytes),
	}
}

func (p *portalPortingRequestProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portal.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while decoding content string of portal porting request action: %+v", err)
		return [][]string{}, nil
	}

	var actionData portalMeta.PortalUserRegisterActionV3
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while unmarshal portal porting request action: %+v", err)
		return [][]string{}, nil
	}

	rejectInst := buildRequestPortingInst(
		actionData.Meta.Type,
		shardID,
		pCommon.PortalRequestRejectedChainStatus,
		actionData.Meta.UniqueRegisterId,
		actionData.Meta.IncogAddressStr,
		actionData.Meta.PTokenId,
		actionData.Meta.RegisterAmount,
		actionData.Meta.PortingFee,
		nil,
		actionData.TxReqID,
		actionData.ShardHeight,
	)

	if currentPortalState == nil {
		Logger.log.Errorf("Porting request: Current Portal state is null")
		return [][]string{rejectInst}, nil
	}

	// check unique id from optionalData which get from statedb
	if optionalData == nil {
		Logger.log.Errorf("Porting request: optionalData is null")
		return [][]string{rejectInst}, nil
	}
	isExist, ok := optionalData["isExistPortingID"].(bool)
	if !ok {
		Logger.log.Errorf("Porting request: optionalData isExistPortingID is invalid")
		return [][]string{rejectInst}, nil
	}
	if isExist {
		Logger.log.Errorf("Porting request: Porting request id exist in db %v", actionData.Meta.UniqueRegisterId)
		return [][]string{rejectInst}, nil
	}

	waitingPortingRequestKey := statedb.GeneratePortalWaitingPortingRequestObjectKey(actionData.Meta.UniqueRegisterId)
	if _, ok := currentPortalState.WaitingPortingRequests[waitingPortingRequestKey.String()]; ok {
		Logger.log.Errorf("Porting request: Waiting porting request exist, key %v", waitingPortingRequestKey)
		return [][]string{rejectInst}, nil
	}

	// get exchange rates
	exchangeRatesState := currentPortalState.FinalExchangeRatesState
	if exchangeRatesState == nil {
		Logger.log.Errorf("Porting request, exchange rates not found")
		return [][]string{rejectInst}, nil
	}

	// validate porting fees
	exchangePortingFees, err := CalMinPortingFee(actionData.Meta.RegisterAmount, actionData.Meta.PTokenId, exchangeRatesState, portalParams)
	if err != nil {
		Logger.log.Errorf("Porting request: Calculate Porting fee is error %v", err)
		return [][]string{rejectInst}, nil
	}
	if actionData.Meta.PortingFee < exchangePortingFees {
		Logger.log.Errorf("Porting request: Porting fees is invalid: expected %v - get %v", exchangePortingFees, actionData.Meta.PortingFee)
		return [][]string{rejectInst}, nil
	}

	// pick-up custodians
	pickedCustodians, err := pickUpCustodianForPorting(
		actionData.Meta.RegisterAmount,
		actionData.Meta.PTokenId,
		currentPortalState.CustodianPoolState,
		currentPortalState.FinalExchangeRatesState,
		portalParams,
	)
	if err != nil || len(pickedCustodians) == 0 {
		Logger.log.Errorf("Porting request: an error occurred while picking up custodians for the porting request: %+v", err)
		return [][]string{rejectInst}, nil
	}

	// Update custodian state after finishing choosing enough custodians for the porting request
	for _, cus := range pickedCustodians {
		// update custodian state
		err := UpdateCustodianStateAfterMatchingPortingRequest(currentPortalState, cus, actionData.Meta.PTokenId)
		if err != nil {
			Logger.log.Errorf("Porting request: an error occurred while updating custodian state: %+v", err)
			return [][]string{rejectInst}, nil
		}
	}

	acceptInst := buildRequestPortingInst(
		actionData.Meta.Type,
		shardID,
		pCommon.PortalRequestAcceptedChainStatus,
		actionData.Meta.UniqueRegisterId,
		actionData.Meta.IncogAddressStr,
		actionData.Meta.PTokenId,
		actionData.Meta.RegisterAmount,
		actionData.Meta.PortingFee,
		pickedCustodians,
		actionData.TxReqID,
		actionData.ShardHeight,
	)

	newPortingRequestStateWaiting := statedb.NewWaitingPortingRequestWithValue(
		actionData.Meta.UniqueRegisterId,
		actionData.TxReqID,
		actionData.Meta.PTokenId,
		actionData.Meta.IncogAddressStr,
		actionData.Meta.RegisterAmount,
		pickedCustodians,
		actionData.Meta.PortingFee,
		beaconHeight+1,
		actionData.ShardHeight,
		actionData.ShardID,
	)

	keyWaitingPortingRequest := statedb.GeneratePortalWaitingPortingRequestObjectKey(actionData.Meta.UniqueRegisterId)
	currentPortalState.WaitingPortingRequests[keyWaitingPortingRequest.String()] = newPortingRequestStateWaiting

	return [][]string{acceptInst}, nil
}

func (p *portalPortingRequestProcessor) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalState *CurrentPortalState,
	portalParams portal.PortalParams,
	updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
) error {

	if currentPortalState == nil {
		Logger.log.Errorf("current portal state is nil")
		return nil
	}

	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	// parse instruction
	var portingRequestContent portalMeta.PortalPortingRequestContent
	err := json.Unmarshal([]byte(instructions[3]), &portingRequestContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occurred while unmarshaling content string of porting request contribution instruction: %+v", err)
		return nil
	}

	reqStatus := instructions[2]

	uniquePortingID := portingRequestContent.UniqueRegisterId
	txReqID := portingRequestContent.TxReqID
	tokenID := portingRequestContent.PTokenId
	porterAddress := portingRequestContent.IncogAddressStr
	amount := portingRequestContent.RegisterAmount
	custodiansDetail := portingRequestContent.Custodian
	portingFee := portingRequestContent.PortingFee
	shardHeight := portingRequestContent.ShardHeight
	shardId := portingRequestContent.ShardID

	switch reqStatus {
	case pCommon.PortalRequestAcceptedChainStatus:
		//verify custodian
		isCustodianAccepted := true
		for _, itemCustodian := range custodiansDetail {
			keyPortingRequestNewState := statedb.GenerateCustodianStateObjectKey(itemCustodian.IncAddress)
			keyPortingRequestNewStateStr := keyPortingRequestNewState.String()
			custodian, ok := currentPortalState.CustodianPoolState[keyPortingRequestNewStateStr]
			if !ok {
				Logger.log.Errorf("ERROR: Custodian not found")
				isCustodianAccepted = false
				break
			}

			if custodian.GetFreeCollateral() < itemCustodian.LockedAmountCollateral {
				Logger.log.Errorf("ERROR: Custodian is not enough PRV, free collateral %v < lock amount %v", custodian.GetFreeCollateral(), itemCustodian.LockedAmountCollateral)
				isCustodianAccepted = false
				break
			}

			continue
		}

		if isCustodianAccepted == false {
			Logger.log.Errorf("ERROR: Custodian not found")
			return nil
		}

		// new request
		newWaitingPortingRequestState := statedb.NewWaitingPortingRequestWithValue(
			uniquePortingID,
			txReqID,
			tokenID,
			porterAddress,
			amount,
			custodiansDetail,
			portingFee,
			beaconHeight+1,
			shardHeight,
			shardId,
		)

		newPortingRequestState := portalMeta.NewPortingRequestStatus(
			uniquePortingID,
			txReqID,
			tokenID,
			porterAddress,
			amount,
			custodiansDetail,
			portingFee,
			pCommon.PortalPortingReqWaitingStatus,
			beaconHeight+1,
			shardHeight,
			shardId,
		)

		newPortingTxRequestState := portalMeta.NewPortingRequestStatus(
			uniquePortingID,
			txReqID,
			tokenID,
			porterAddress,
			amount,
			custodiansDetail,
			portingFee,
			pCommon.PortalRequestAcceptedStatus,
			beaconHeight+1,
			shardHeight,
			shardId,
		)

		//save transaction
		newPortingTxRequestStatusBytes, _ := json.Marshal(newPortingTxRequestState)
		err = statedb.StorePortalPortingRequestByTxIDStatus(
			stateDB,
			txReqID.String(),
			newPortingTxRequestStatusBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occurred while store porting tx request item: %+v", err)
			return nil
		}

		//save success porting request
		newPortingRequestStatusBytes, _ := json.Marshal(newPortingRequestState)
		err = statedb.StorePortalPortingRequestStatus(
			stateDB,
			uniquePortingID,
			newPortingRequestStatusBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occurred while store porting request item: %+v", err)
			return nil
		}

		//save custodian state
		for _, itemCustodian := range custodiansDetail {
			//update custodian state
			_ = UpdateCustodianStateAfterMatchingPortingRequest(currentPortalState, itemCustodian, tokenID)
		}

		//save waiting request porting state
		keyWaitingPortingRequest := statedb.GeneratePortalWaitingPortingRequestObjectKey(portingRequestContent.UniqueRegisterId)
		Logger.log.Infof("Porting request, save waiting porting request with key %v", keyWaitingPortingRequest)
		currentPortalState.WaitingPortingRequests[keyWaitingPortingRequest.String()] = newWaitingPortingRequestState

		break
	case pCommon.PortalRequestRejectedChainStatus:
		txReqID := portingRequestContent.TxReqID

		newPortingRequest := portalMeta.NewPortingRequestStatus(
			uniquePortingID,
			txReqID,
			tokenID,
			porterAddress,
			amount,
			custodiansDetail,
			portingFee,
			pCommon.PortalRequestRejectedStatus,
			beaconHeight+1,
			shardHeight,
			shardId,
		)

		//save transaction
		newPortingTxRequestStatusBytes, _ := json.Marshal(newPortingRequest)
		err = statedb.StorePortalPortingRequestByTxIDStatus(
			stateDB,
			txReqID.String(),
			newPortingTxRequestStatusBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occurred while store porting request item: %+v", err)
			return nil
		}
		break
	}

	return nil
}

/* =======
Portal Request Ptoken Processor
======= */

type portalRequestPTokenProcessor struct {
	*portalInstProcessor
}

func (p *portalRequestPTokenProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalRequestPTokenProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalRequestPTokenProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	return nil, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildReqPTokensInst(
	uniquePortingID string,
	tokenID string,
	incogAddressStr string,
	portingAmount uint64,
	portingProof string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	reqPTokenContent := portalMeta.PortalRequestPTokensContent{
		UniquePortingID: uniquePortingID,
		TokenID:         tokenID,
		IncogAddressStr: incogAddressStr,
		PortingAmount:   portingAmount,
		PortingProof:    portingProof,
		TxReqID:         txReqID,
		ShardID:         shardID,
	}
	reqPTokenContentBytes, _ := json.Marshal(reqPTokenContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(reqPTokenContentBytes),
	}
}

func (p *portalRequestPTokenProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portal.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal request ptoken action: %+v", err)
		return [][]string{}, nil
	}
	var actionData portalMeta.PortalRequestPTokensAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal request ptoken action: %+v", err)
		return [][]string{}, nil
	}
	meta := actionData.Meta

	rejectInst := buildReqPTokensInst(
		meta.UniquePortingID,
		meta.TokenID,
		meta.IncogAddressStr,
		meta.PortingAmount,
		meta.PortingProof,
		meta.Type,
		shardID,
		actionData.TxReqID,
		pCommon.PortalRequestRejectedChainStatus,
	)

	if currentPortalState == nil {
		Logger.log.Warn("Request PTokens: Current Portal state is null.")
		return [][]string{rejectInst}, nil
	}

	// check meta.UniquePortingID is in waiting PortingRequests list in portal state or not
	portingID := meta.UniquePortingID
	keyWaitingPortingRequestStr := statedb.GeneratePortalWaitingPortingRequestObjectKey(portingID).String()
	waitingPortingRequest := currentPortalState.WaitingPortingRequests[keyWaitingPortingRequestStr]
	if waitingPortingRequest == nil {
		Logger.log.Errorf("PortingID is not existed in waiting porting requests list")
		return [][]string{rejectInst}, nil
	}

	// check tokenID
	if meta.TokenID != waitingPortingRequest.TokenID() {
		Logger.log.Errorf("TokenID is not correct in portingID req")
		return [][]string{rejectInst}, nil
	}

	// check porting amount
	if meta.PortingAmount != waitingPortingRequest.Amount() {
		Logger.log.Errorf("PortingAmount is not correct in portingID req")
		return [][]string{rejectInst}, nil
	}

	// check requester address
	if meta.IncogAddressStr != waitingPortingRequest.PorterAddress() {
		Logger.log.Errorf("IncogAddressStr is not correct in waiting portingID req")
		return [][]string{rejectInst}, nil
	}

	portalTokenProcessor := portalParams.PortalTokens[meta.TokenID]
	if portalTokenProcessor == nil {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		return [][]string{rejectInst}, nil
	}

	expectedMemo := portalTokenProcessor.GetExpectedMemoForPorting(meta.UniquePortingID)
	expectedPaymentInfos := map[string]uint64{}
	for _, matchCus := range waitingPortingRequest.Custodians() {
		expectedPaymentInfos[matchCus.RemoteAddress] = matchCus.Amount
	}
	isValid, err := portalTokenProcessor.ParseAndVerifyProof(meta.PortingProof, bc, expectedMemo, expectedPaymentInfos)
	if !isValid || err != nil {
		Logger.log.Error("Parse proof and verify porting proof failed: %v", err)
		return [][]string{rejectInst}, nil
	}

	// update holding public token for custodians
	for _, cusDetail := range waitingPortingRequest.Custodians() {
		custodianKey := statedb.GenerateCustodianStateObjectKey(cusDetail.IncAddress)
		UpdateCustodianStateAfterUserRequestPToken(currentPortalState, custodianKey.String(), waitingPortingRequest.TokenID(), cusDetail.Amount)
	}

	inst := buildReqPTokensInst(
		actionData.Meta.UniquePortingID,
		actionData.Meta.TokenID,
		actionData.Meta.IncogAddressStr,
		actionData.Meta.PortingAmount,
		actionData.Meta.PortingProof,
		actionData.Meta.Type,
		shardID,
		actionData.TxReqID,
		pCommon.PortalRequestAcceptedChainStatus,
	)

	// remove waiting porting request from currentPortalState
	deleteWaitingPortingRequest(currentPortalState, keyWaitingPortingRequestStr)
	return [][]string{inst}, nil
}

func (p *portalRequestPTokenProcessor) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalState *CurrentPortalState,
	portalParams portal.PortalParams,
	updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
) error {
	if currentPortalState == nil {
		Logger.log.Errorf("current portal state is nil")
		return nil
	}

	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	// unmarshal instructions content
	var actionData portalMeta.PortalRequestPTokensContent
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error: %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		waitingPortingReqKey := statedb.GeneratePortalWaitingPortingRequestObjectKey(actionData.UniquePortingID)
		waitingPortingReqKeyStr := waitingPortingReqKey.String()
		waitingPortingReq := currentPortalState.WaitingPortingRequests[waitingPortingReqKeyStr]

		// update holding public token for custodians
		for _, cusDetail := range waitingPortingReq.Custodians() {
			custodianKey := statedb.GenerateCustodianStateObjectKey(cusDetail.IncAddress)
			UpdateCustodianStateAfterUserRequestPToken(currentPortalState, custodianKey.String(), waitingPortingReq.TokenID(), cusDetail.Amount)
		}

		// remove portingRequest from waitingPortingRequests
		deleteWaitingPortingRequest(currentPortalState, waitingPortingReqKeyStr)
		statedb.DeleteWaitingPortingRequest(stateDB, waitingPortingReq.UniquePortingID())
		// make sure user can not re-use proof for other portingID
		// update status of porting request with portingID

		//update new status of porting request
		portingRequestState, err := statedb.GetPortalPortingRequestStatus(stateDB, actionData.UniquePortingID)
		if err != nil {
			Logger.log.Errorf("Has an error occurred while get porting request status: %+v", err)
			return nil
		}

		var portingRequestStatus portalMeta.PortingRequestStatus
		err = json.Unmarshal(portingRequestState, &portingRequestStatus)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occurred while unmarshal PortingRequestStatus: %+v", err)
			return nil
		}

		portingRequestStatus.Status = pCommon.PortalPortingReqSuccessStatus
		newPortingRequestStatusBytes, _ := json.Marshal(portingRequestStatus)
		err = statedb.StorePortalPortingRequestStatus(
			stateDB,
			actionData.UniquePortingID,
			newPortingRequestStatusBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occurred while store porting request item status: %+v", err)
			return nil
		}
		//end

		// track reqPToken status by txID into DB
		reqPTokenTrackData := portalMeta.PortalRequestPTokensStatus{
			Status:          pCommon.PortalRequestAcceptedStatus,
			UniquePortingID: actionData.UniquePortingID,
			TokenID:         actionData.TokenID,
			IncogAddressStr: actionData.IncogAddressStr,
			PortingAmount:   actionData.PortingAmount,
			PortingProof:    actionData.PortingProof,
			TxReqID:         actionData.TxReqID,
		}
		reqPTokenTrackDataBytes, _ := json.Marshal(reqPTokenTrackData)
		err = statedb.StoreRequestPTokenStatus(
			stateDB,
			actionData.TxReqID.String(),
			reqPTokenTrackDataBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking request ptoken tx: %+v", err)
			return nil
		}

		// update bridge/portal token info
		incTokenID, err := common.Hash{}.NewHashFromStr(actionData.TokenID)
		if err != nil {
			Logger.log.Errorf("ERROR: Can not new hash from porting incTokenID: %+v", err)
			return nil
		}
		updatingInfo, found := updatingInfoByTokenID[*incTokenID]
		if found {
			updatingInfo.CountUpAmt += actionData.PortingAmount
		} else {
			updatingInfo = bMeta.UpdatingInfo{
				CountUpAmt:      actionData.PortingAmount,
				DeductAmt:       0,
				TokenID:         *incTokenID,
				ExternalTokenID: nil,
				IsCentralized:   false,
			}
		}
		updatingInfoByTokenID[*incTokenID] = updatingInfo

	} else if reqStatus == pCommon.PortalRequestRejectedChainStatus {
		reqPTokenTrackData := portalMeta.PortalRequestPTokensStatus{
			Status:          pCommon.PortalRequestRejectedStatus,
			UniquePortingID: actionData.UniquePortingID,
			TokenID:         actionData.TokenID,
			IncogAddressStr: actionData.IncogAddressStr,
			PortingAmount:   actionData.PortingAmount,
			PortingProof:    actionData.PortingProof,
			TxReqID:         actionData.TxReqID,
		}
		reqPTokenTrackDataBytes, _ := json.Marshal(reqPTokenTrackData)
		err = statedb.StoreRequestPTokenStatus(
			stateDB,
			actionData.TxReqID.String(),
			reqPTokenTrackDataBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking request ptoken tx: %+v", err)
			return nil
		}
	}

	return nil
}

/* =======
Portal Request Ptoken Processor V4
======= */

type portalRequestPTokenProcessorV4 struct {
	*portalInstProcessor
}

func (p *portalRequestPTokenProcessorV4) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalRequestPTokenProcessorV4) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalRequestPTokenProcessorV4) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	return nil, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildReqPTokensInstV4(
	tokenID string,
	incogAddressStr string,
	portingAmount uint64,
	portingProof string,
	portingUTXO statedb.UTXO,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	reqPTokenContent := portalMeta.PortalRequestPTokensContentV4{
		TokenID:         tokenID,
		IncogAddressStr: incogAddressStr,
		PortingAmount:   portingAmount,
		PortingProof:    portingProof,
		PortingUTXO:     portingUTXO,
		TxReqID:         txReqID,
		ShardID:         shardID,
	}
	reqPTokenContentBytes, _ := json.Marshal(reqPTokenContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(reqPTokenContentBytes),
	}
}

func (p *portalRequestPTokenProcessorV4) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portal.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal request ptoken action: %+v", err)
		return [][]string{}, nil
	}
	var actionData portalMeta.PortalRequestPTokensAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal request ptoken action: %+v", err)
		return [][]string{}, nil
	}
	meta := actionData.Meta

	rejectInst := buildReqPTokensInstV4(
		meta.TokenID,
		meta.IncogAddressStr,
		meta.PortingAmount,
		meta.PortingProof,
		statedb.UTXO{},
		meta.Type,
		shardID,
		actionData.TxReqID,
		pCommon.PortalRequestRejectedChainStatus,
	)

	if currentPortalState == nil {
		Logger.log.Warn("Request PTokens: Current Portal state is null.")
		return [][]string{rejectInst}, nil
	}

	portalTokenProcessor := portalParams.PortalTokens[meta.TokenID]
	if portalTokenProcessor == nil {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		return [][]string{rejectInst}, nil
	}

	expectedMemo := portalTokenProcessor.GetExpectedMemoForPorting(meta.IncogAddressStr)
	expectedMultisigAddress := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"
	expectedAmount := meta.PortingAmount
	isValid, UTXO, err := portalTokenProcessor.ParseAndVerifyProofV4(meta.PortingProof, bc, expectedMemo, expectedMultisigAddress, expectedAmount)
	if !isValid || err != nil {
		Logger.log.Error("Parse proof and verify porting proof failed: %v", err)
		return [][]string{rejectInst}, nil
	}

	multisigWalletKey := statedb.GenerateMultisigWalletStateObjectKey(expectedMultisigAddress, meta.TokenID)
	UpdateMultisigWalletStateAfterUserRequestPToken(currentPortalState, multisigWalletKey.String(), *UTXO)

	inst := buildReqPTokensInstV4(
		actionData.Meta.TokenID,
		actionData.Meta.IncogAddressStr,
		actionData.Meta.PortingAmount,
		actionData.Meta.PortingProof,
		*UTXO,
		actionData.Meta.Type,
		shardID,
		actionData.TxReqID,
		pCommon.PortalRequestAcceptedChainStatus,
	)
	return [][]string{inst}, nil
}

func (p *portalRequestPTokenProcessorV4) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalState *CurrentPortalState,
	portalParams portal.PortalParams,
	updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
) error {
	if currentPortalState == nil {
		Logger.log.Errorf("current portal state is nil")
		return nil
	}

	// ?: len of instructions
	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	// unmarshal instructions content
	var actionData portalMeta.PortalRequestPTokensContentV4
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error: %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		expectedMultisigAddress := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"
		multisigWalletKey := statedb.GenerateMultisigWalletStateObjectKey(expectedMultisigAddress, actionData.TokenID)
		UpdateMultisigWalletStateAfterUserRequestPToken(currentPortalState, multisigWalletKey.String(), actionData.PortingUTXO)

		// track reqPToken status by txID into DB
		reqPTokenTrackData := portalMeta.PortalRequestPTokensStatusV4{
			Status:          pCommon.PortalRequestAcceptedStatus,
			TokenID:         actionData.TokenID,
			IncogAddressStr: actionData.IncogAddressStr,
			PortingAmount:   actionData.PortingAmount,
			PortingProof:    actionData.PortingProof,
			PortingUTXO:     actionData.PortingUTXO,
			TxReqID:         actionData.TxReqID,
		}
		reqPTokenTrackDataBytes, _ := json.Marshal(reqPTokenTrackData)
		err = statedb.StoreRequestPTokenStatus(
			stateDB,
			actionData.TxReqID.String(),
			reqPTokenTrackDataBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking request ptoken tx: %+v", err)
			return nil
		}

		// update bridge/portal token info
		incTokenID, err := common.Hash{}.NewHashFromStr(actionData.TokenID)
		if err != nil {
			Logger.log.Errorf("ERROR: Can not new hash from porting incTokenID: %+v", err)
			return nil
		}
		updatingInfo, found := updatingInfoByTokenID[*incTokenID]
		if found {
			updatingInfo.CountUpAmt += actionData.PortingAmount
		} else {
			updatingInfo = bMeta.UpdatingInfo{
				CountUpAmt:      actionData.PortingAmount,
				DeductAmt:       0,
				TokenID:         *incTokenID,
				ExternalTokenID: nil,
				IsCentralized:   false,
			}
		}
		updatingInfoByTokenID[*incTokenID] = updatingInfo

	} else if reqStatus == pCommon.PortalRequestRejectedChainStatus {
		reqPTokenTrackData := portalMeta.PortalRequestPTokensStatusV4{
			Status:          pCommon.PortalRequestRejectedStatus,
			TokenID:         actionData.TokenID,
			IncogAddressStr: actionData.IncogAddressStr,
			PortingAmount:   actionData.PortingAmount,
			PortingProof:    actionData.PortingProof,
			PortingUTXO:     actionData.PortingUTXO,
			TxReqID:         actionData.TxReqID,
		}
		reqPTokenTrackDataBytes, _ := json.Marshal(reqPTokenTrackData)
		err = statedb.StoreRequestPTokenStatus(
			stateDB,
			actionData.TxReqID.String(),
			reqPTokenTrackDataBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking request ptoken tx: %+v", err)
			return nil
		}
	}

	return nil
}
