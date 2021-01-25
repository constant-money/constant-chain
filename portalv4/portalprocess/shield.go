package portalprocess

import (
	"encoding/base64"
	"encoding/json"
	"strconv"

	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/portal"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"github.com/incognitochain/incognito-chain/portalv4/metadata"
)

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
	// TODO: query btc txid to detect used or not
	return nil, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildReqPTokensInstV4(
	tokenID string,
	incogAddressStr string,
	PortingWalletAddress string,
	portingAmount uint64,
	portingProof string,
	portingUTXO []*statedb.UTXO,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	reqPTokenContent := metadata.PortalRequestPTokensContentV4{
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
	currentPortalState *CurrentPortalV4State,
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
	var actionData metadata.PortalRequestPTokensActionV4
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal request ptoken action: %+v", err)
		return [][]string{}, nil
	}
	meta := actionData.Meta

	rejectInst := buildReqPTokensInstV4(
		meta.TokenID,
		meta.IncogAddressStr,
		"",
		0,
		meta.PortingProof,
		[]*statedb.UTXO{},
		meta.Type,
		shardID,
		actionData.TxReqID,
		pCommon.PortalRequestRejectedChainStatus,
	)

	if currentPortalState == nil {
		Logger.log.Warn("Request PTokens: Current Portal state is null.")
		return [][]string{rejectInst}, nil
	}

	portalTokenProcessor := portalParams.PortalTokensV4[meta.TokenID]
	if portalTokenProcessor == nil {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		return [][]string{rejectInst}, nil
	}

	expectedMemo := portalTokenProcessor.GetExpectedMemoForPorting(meta.IncogAddressStr)
	// TODO: get this value from portal params
	expectedMultisigAddress := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"
	isValid, listUTXO, portingAmount, err := portalTokenProcessor.ParseAndVerifyProof(meta.PortingProof, bc, expectedMemo, expectedMultisigAddress)

	if !isValid || err != nil {
		Logger.log.Error("Parse proof and verify porting proof failed: %v", err)
		return [][]string{rejectInst}, nil
	}

	err = UpdateMultisigWalletsStateAfterUserRequestPToken(currentPortalState, meta.TokenID, expectedMultisigAddress, listUTXO)

	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while execute UpdateMultisigWalletsStateAfterUserRequestPToken: %+v", err)
		return [][]string{rejectInst}, nil
	}

	inst := buildReqPTokensInstV4(
		actionData.Meta.TokenID,
		actionData.Meta.IncogAddressStr,
		expectedMultisigAddress,
		portingAmount,
		actionData.Meta.PortingProof,
		listUTXO,
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
	currentPortalState *CurrentPortalV4State,
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
	var actionData metadata.PortalRequestPTokensContentV4
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error: %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		err = UpdateMultisigWalletsStateAfterUserRequestPToken(currentPortalState, actionData.TokenID, actionData.PortingWalletAddress, actionData.PortingUTXO)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while execute UpdateMultisigWalletsStateAfterUserRequestPToken: %+v", err)
			return nil
		}

		// track reqPToken status by txID into DB
		reqPTokenTrackData := metadata.PortalRequestPTokensStatusV4{
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
		reqPTokenTrackData := metadata.PortalRequestPTokensStatusV4{
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
