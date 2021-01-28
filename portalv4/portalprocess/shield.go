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
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"github.com/incognitochain/incognito-chain/portalv4/metadata"
	portalMeta "github.com/incognitochain/incognito-chain/portalv4/metadata"
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
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Shielding request: an error occurred while decoding content string of pToken request action: %+v", err)
		return nil, fmt.Errorf("Shielding request: an error occurred while decoding content string of pToken request action: %+v", err)
	}

	var actionData portalMeta.PortalRequestPTokensActionV4
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("Shielding request: an error occurred while unmarshal pToken request action: %+v", err)
		return nil, fmt.Errorf("Shielding request: an error occurred while unmarshal pToken request action: %+v", err)
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
	shieldingWalletAddress string,
	shieldingAmount uint64,
	shieldingExternalTxHash string,
	shieldingUTXO []*statedb.UTXO,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	reqPTokenContent := portalMeta.PortalRequestPTokensContentV4{
		TokenID:                 tokenID,
		IncogAddressStr:         incogAddressStr,
		ShieldingWalletAddress:  shieldingWalletAddress,
		ShieldingAmount:         shieldingAmount,
		ShieldingExternalTxHash: shieldingExternalTxHash,
		ShieldingUTXO:           shieldingUTXO,
		TxReqID:                 txReqID,
		ShardID:                 shardID,
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
	portalParams portalv4.PortalParams,
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
		"",
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

	// // check unique id from optionalData which get from statedb
	// if optionalData == nil {
	// 	Logger.log.Errorf("Shielding request: optionalData is null")
	// 	return [][]string{rejectInst}, nil
	// }
	// isExist, ok := optionalData["isExistExternalTxHash"].(bool)
	// if !ok {
	// 	Logger.log.Errorf("Shielding request: optionalData isExistExternalTxHash is invalid")
	// 	return [][]string{rejectInst}, nil
	// }
	// if isExist {
	// 	Logger.log.Errorf("Shielding request: Shielding proof exists in db %v", actionData.Meta.ShieldingProof)
	// 	return [][]string{rejectInst}, nil
	// }

	portalTokenProcessor := portalParams.PortalTokens[meta.TokenID]
	if portalTokenProcessor == nil {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		return [][]string{rejectInst}, nil
	}

	expectedMemo := portalTokenProcessor.GetExpectedMemoForShielding(meta.IncogAddressStr)
	// TODO: get this value from portal params
	expectedMultisigAddress := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"
	isValid, listUTXO, shieldingAmount, err := portalTokenProcessor.ParseAndVerifyProof(meta.ShieldingProof, bc, expectedMemo, expectedMultisigAddress)

	if !isValid || err != nil {
		Logger.log.Error("Parse proof and verify shielding proof failed: %v", err)
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
		shieldingAmount,
		listUTXO[0].GetTxHash(),
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
	var actionData metadata.PortalRequestPTokensContentV4
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error: %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		err = UpdateMultisigWalletsStateAfterUserRequestPToken(currentPortalState, actionData.TokenID, actionData.ShieldingWalletAddress, actionData.ShieldingUTXO)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while execute UpdateMultisigWalletsStateAfterUserRequestPToken: %+v", err)
			return nil
		}
		err = SaveShieldingExternalTxToStateDB(currentPortalState, actionData.TokenID, actionData.ShieldingExternalTxHash, actionData.IncogAddressStr, actionData.ShieldingAmount)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while execute SaveShieldingExternalTxHashToStateDB: %+v", err)
			return nil
		}

		// track reqPToken status by txID into DB
		reqPTokenTrackData := metadata.PortalRequestPTokensStatusV4{
			Status:                  pCommon.PortalRequestAcceptedStatus,
			TokenID:                 actionData.TokenID,
			IncogAddressStr:         actionData.IncogAddressStr,
			ShieldingWalletAddress:  actionData.ShieldingWalletAddress,
			ShieldingAmount:         actionData.ShieldingAmount,
			ShieldingExternalTxHash: actionData.ShieldingExternalTxHash,
			ShieldingUTXO:           actionData.ShieldingUTXO,
			TxReqID:                 actionData.TxReqID,
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
			Logger.log.Errorf("ERROR: Can not new hash from shielding incTokenID: %+v", err)
			return nil
		}
		updatingInfo, found := updatingInfoByTokenID[*incTokenID]
		if found {
			updatingInfo.CountUpAmt += actionData.ShieldingAmount
		} else {
			updatingInfo = bMeta.UpdatingInfo{
				CountUpAmt:      actionData.ShieldingAmount,
				DeductAmt:       0,
				TokenID:         *incTokenID,
				ExternalTokenID: nil,
				IsCentralized:   false,
			}
		}
		updatingInfoByTokenID[*incTokenID] = updatingInfo

	} else if reqStatus == pCommon.PortalRequestRejectedChainStatus {
		reqPTokenTrackData := metadata.PortalRequestPTokensStatusV4{
			Status:                  pCommon.PortalRequestRejectedStatus,
			TokenID:                 actionData.TokenID,
			IncogAddressStr:         actionData.IncogAddressStr,
			ShieldingWalletAddress:  actionData.ShieldingWalletAddress,
			ShieldingAmount:         actionData.ShieldingAmount,
			ShieldingExternalTxHash: actionData.ShieldingExternalTxHash,
			ShieldingUTXO:           actionData.ShieldingUTXO,
			TxReqID:                 actionData.TxReqID,
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
