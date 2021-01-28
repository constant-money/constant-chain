package portalprocess

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"github.com/incognitochain/incognito-chain/portalv4"
	pv4Meta "github.com/incognitochain/incognito-chain/portalv4/metadata"
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"
	"strconv"
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
	unshieldRequestContent :=pv4Meta.PortalUnshieldRequestContent{
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

//todo:
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

	if currentPortalV4State.WaitingUnshieldRequests[tokenID] == nil {
		currentPortalV4State.WaitingUnshieldRequests[tokenID] = statedb.NewWaitingUnshieldState()
	}
	wUnshieldReqsByTokenID := currentPortalV4State.WaitingUnshieldRequests[tokenID].GetUnshields()

	if currentPortalV4State.UnshieldRequestsProcessed[tokenID] == nil {
		currentPortalV4State.UnshieldRequestsProcessed[tokenID] = statedb.NewProcessUnshieldState()
	}
	processedUnshieldReqsByTokenID := currentPortalV4State.UnshieldRequestsProcessed[tokenID].GetUnshields()


	// check unshieldID is existed waitingUnshield list or not
	keyWaitingUnshieldRequestStr := statedb.GenerateWaitingWaitingUnshieldObjectKey(unshieldID).String()
	waitingUnshieldRequest := wUnshieldReqsByTokenID[keyWaitingUnshieldRequestStr]
	if waitingUnshieldRequest != nil {
		Logger.log.Errorf("[Unshield Request] unshieldID is existed in waiting unshield requests list %v\n", unshieldID)
		return [][]string{rejectInst}, nil
	}

	// check unshieldID is existed matched process Unshield request list or not
	keyProcessedUnshieldRequest := statedb.GenerateMatchedProcessUnshieldObjectKey(unshieldID).String()
	processedUnshieldRequest := processedUnshieldReqsByTokenID[keyProcessedUnshieldRequest]
	if processedUnshieldRequest != nil {
		Logger.log.Errorf("[Unshield Request] unshieldID is existed in processed unshield requests list %v\n", unshieldID)
		return [][]string{rejectInst}, nil
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
	UpdatePortalStateAfterUnshieldRequest(currentPortalV4State, unshieldID, meta.TokenID, meta.RemoteAddress, meta.UnshieldAmount)

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
		UpdatePortalStateAfterUnshieldRequest(currentPortalV4State, actionData.TxReqID.String(), actionData.TokenID, actionData.RemoteAddress, actionData.UnshieldAmount)

		// track status of redeem request by redeemID
		redeemRequestStatus := pv4Meta.PortalUnshieldRequestStatus{
			IncAddressStr:  actionData.IncAddressStr,
			RemoteAddress:  actionData.RemoteAddress,
			TokenID:        actionData.TokenID,
			UnshieldAmount: actionData.UnshieldAmount,
			TxHash:         actionData.TxReqID.String(),
			Status:         pv4Common.PortalUnshieldReqWaitingStatus,
		}
		redeemRequestStatusBytes, _ := json.Marshal(redeemRequestStatus)
		err := statedb.StorePortalRedeemRequestStatus(
			stateDB,
			actionData.TxReqID.String(),
			redeemRequestStatusBytes)
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