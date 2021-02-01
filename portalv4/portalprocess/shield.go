package portalprocess

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/portalv4"

	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	pCommon "github.com/incognitochain/incognito-chain/portalv4/common"
	"github.com/incognitochain/incognito-chain/portalv4/metadata"
	portalMeta "github.com/incognitochain/incognito-chain/portalv4/metadata"
)

/* =======
Portal Shielding Request Processor V4
======= */

type portalShieldingRequestProcessor struct {
	*portalInstProcessor
}

func (p *portalShieldingRequestProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalShieldingRequestProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalShieldingRequestProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Shielding request: an error occurred while decoding content string of pToken request action: %+v", err)
		return nil, fmt.Errorf("Shielding request: an error occurred while decoding content string of pToken request action: %+v", err)
	}

	var actionData portalMeta.PortalShieldingRequestAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("Shielding request: an error occurred while unmarshal shielding request action: %+v", err)
		return nil, fmt.Errorf("Shielding request: an error occurred while unmarshal shielding request action: %+v", err)
	}

	portalParams := portalv4.PortalParams{}
	portalTokenProcessor := portalParams.PortalTokens[actionData.Meta.TokenID]
	if portalTokenProcessor == nil {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		return nil, nil
	}

	externalTxHash, err := portalTokenProcessor.GetExternalTxHashFromProof(actionData.Meta.ShieldingProof)
	if err != nil {
		Logger.log.Error("Parse proof and verify shielding proof failed: %v", err)
		return nil, nil
	}

	isExistExternalTxHash, err := statedb.IsShieldingExternalTxHashExists(stateDB, actionData.Meta.TokenID, externalTxHash)
	if err != nil {
		Logger.log.Errorf("Shielding request: an error occurred while get pToken request proof from DB: %+v", err)
		return nil, fmt.Errorf("Shielding request: an error occurred while get pToken request proof from DB: %+v", err)
	}

	optionalData := make(map[string]interface{})
	optionalData["isExistExternalTxHash"] = isExistExternalTxHash
	return optionalData, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildReqPTokensInstV4(
	tokenID string,
	incogAddressStr string,
	shieldingUTXO []*statedb.UTXO,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	shieldingReqContent := portalMeta.PortalShieldingRequestContent{
		TokenID:         tokenID,
		IncogAddressStr: incogAddressStr,
		ShieldingUTXO:   shieldingUTXO,
		TxReqID:         txReqID,
		ShardID:         shardID,
	}
	shieldingReqContentBytes, _ := json.Marshal(shieldingReqContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(shieldingReqContentBytes),
	}
}

func (p *portalShieldingRequestProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalState *CurrentPortalV4State,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portalv4.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal shielding request action: %+v", err)
		return [][]string{}, nil
	}
	var actionData metadata.PortalShieldingRequestAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal shielding request action: %+v", err)
		return [][]string{}, nil
	}
	meta := actionData.Meta

	rejectInst := buildReqPTokensInstV4(
		meta.TokenID,
		meta.IncogAddressStr,
		[]*statedb.UTXO{},
		meta.Type,
		shardID,
		actionData.TxReqID,
		pCommon.PortalRequestRejectedChainStatus,
	)

	if currentPortalState == nil {
		Logger.log.Warn("Shielding Request: Current Portal state is null.")
		return [][]string{rejectInst}, nil
	}

	portalTokenProcessor := portalParams.PortalTokens[meta.TokenID]
	if portalTokenProcessor == nil {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		return [][]string{rejectInst}, nil
	}

	expectedMemo := portalTokenProcessor.GetExpectedMemoForShielding(meta.IncogAddressStr)
	// TODO: get this value from portal params
	expectedMultisigAddress := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"
	isValid, listUTXO, err := portalTokenProcessor.ParseAndVerifyProof(meta.ShieldingProof, bc, expectedMemo, expectedMultisigAddress)

	if !isValid || err != nil {
		Logger.log.Error("Parse proof and verify shielding proof failed: %v", err)
		return [][]string{rejectInst}, nil
	}

	UpdatePortalStateAfterShieldingRequest(currentPortalState, meta.TokenID, listUTXO)

	inst := buildReqPTokensInstV4(
		actionData.Meta.TokenID,
		actionData.Meta.IncogAddressStr,
		listUTXO,
		actionData.Meta.Type,
		shardID,
		actionData.TxReqID,
		pCommon.PortalRequestAcceptedChainStatus,
	)
	return [][]string{inst}, nil
}

func (p *portalShieldingRequestProcessor) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalState *CurrentPortalV4State,
	portalParams portalv4.PortalParams,
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
	var actionData metadata.PortalShieldingRequestContent
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error: %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		UpdatePortalStateAfterShieldingRequest(currentPortalState, actionData.TokenID, actionData.ShieldingUTXO)
		shieldingExternalTxHash := actionData.ShieldingUTXO[0].GetTxHash()
		shieldingAmount := uint64(0)
		for _, utxo := range actionData.ShieldingUTXO {
			shieldingAmount += utxo.GetOutputAmount()
		}

		SaveShieldingExternalTxToStateDB(currentPortalState, actionData.TokenID, shieldingExternalTxHash, actionData.IncogAddressStr, shieldingAmount)

		// track shieldingReq status by txID into DB
		shieldingReqTrackData := metadata.PortalShieldingRequestStatus{
			Status:          pCommon.PortalRequestAcceptedStatus,
			TokenID:         actionData.TokenID,
			IncogAddressStr: actionData.IncogAddressStr,
			ShieldingUTXO:   actionData.ShieldingUTXO,
			TxReqID:         actionData.TxReqID,
		}
		shieldingReqTrackDataBytes, _ := json.Marshal(shieldingReqTrackData)
		err = statedb.StoreShieldingRequestStatus(
			stateDB,
			actionData.TxReqID.String(),
			shieldingReqTrackDataBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking shielding request tx: %+v", err)
			return nil
		}

		// update bridge/portal token info
		incTokenID, err := common.Hash{}.NewHashFromStr(actionData.TokenID)
		if err != nil {
			Logger.log.Errorf("ERROR: Can not new hash from shielding incTokenID: %+v", err)
			return nil
		}
		updatingInfo, found := updatingInfoByTokenID[*incTokenID]
		if found {
			updatingInfo.CountUpAmt += shieldingAmount
		} else {
			updatingInfo = bMeta.UpdatingInfo{
				CountUpAmt:      shieldingAmount,
				DeductAmt:       0,
				TokenID:         *incTokenID,
				ExternalTokenID: nil,
				IsCentralized:   false,
			}
		}
		updatingInfoByTokenID[*incTokenID] = updatingInfo

	} else if reqStatus == pCommon.PortalRequestRejectedChainStatus {
		shieldingReqTrackData := metadata.PortalShieldingRequestStatus{
			Status:          pCommon.PortalRequestRejectedStatus,
			TokenID:         actionData.TokenID,
			IncogAddressStr: actionData.IncogAddressStr,
			ShieldingUTXO:   actionData.ShieldingUTXO,
			TxReqID:         actionData.TxReqID,
		}
		shieldingReqTrackDataBytes, _ := json.Marshal(shieldingReqTrackData)
		err = statedb.StoreShieldingRequestStatus(
			stateDB,
			actionData.TxReqID.String(),
			shieldingReqTrackDataBytes,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while tracking shielding request tx: %+v", err)
			return nil
		}
	}

	return nil
}
