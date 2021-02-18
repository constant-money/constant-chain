package portalprocess

import (
	"encoding/json"
	"fmt"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"github.com/incognitochain/incognito-chain/portalv4"
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"
	pv4Meta "github.com/incognitochain/incognito-chain/portalv4/metadata"
	"github.com/incognitochain/incognito-chain/portalv4/portaltokens"
	"strconv"
)

/* =======
Portal Unshield Request Batching Processor
======= */
type portalUnshieldBatchingProcessor struct {
	*portalInstProcessor
}

func (p *portalUnshieldBatchingProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalUnshieldBatchingProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalUnshieldBatchingProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	return nil, nil
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildUnshieldBatchingInst(
	batchID string,
	rawExtTx string,
	tokenID string,
	unshieldIDs []string,
	utxos map[string][]*statedb.UTXO,
	networkFee map[uint64]uint,
	metaType int,
	status string,
) []string {
	unshieldBatchContent :=pv4Meta.PortalUnshieldRequestBatchContent{
		BatchID:       batchID,
		RawExternalTx: rawExtTx,
		TokenID:       tokenID,
		UnshieldIDs:   unshieldIDs,
		UTXOs: utxos,
		NetworkFee: networkFee,
	}
	unshieldBatchContentBytes, _ := json.Marshal(unshieldBatchContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(-1),
		status,
		string(unshieldBatchContentBytes),
	}
}

// batchID is hash of current beacon height and unshieldIDs that processed
func getBatchID(beaconHeight uint64, unshieldIDs []string) string {
	dataBytes := []byte(fmt.Sprintf("%d", beaconHeight))
	for _, id := range unshieldIDs {
		dataBytes = append(dataBytes, []byte(id)...)
	}
	dataHash := common.HashH(dataBytes)
	return dataHash.String()
}

func (p *portalUnshieldBatchingProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalV4State *CurrentPortalV4State,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portalv4.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	if currentPortalV4State == nil {
		Logger.log.Warn("WARN - [Batch Unshield Request]: Current Portal state V4 is null.")
		return [][]string{}, nil
	}

	newInsts := [][]string{}
	wUnshieldRequests := currentPortalV4State.WaitingUnshieldRequests
	for tokenID, wReqs := range wUnshieldRequests {
		portalTokenProcessor := portalParams.PortalTokens[tokenID]
		if portalTokenProcessor == nil {
			Logger.log.Errorf("[Batch Unshield Request]: Portal token ID %v is null.", tokenID)
			continue
		}

		// use default unshield fee in nano ptoken
		feeUnshield := portalParams.FeeUnshields[tokenID]

		// choose waiting unshield IDs to process with current UTXOs
		utxos := currentPortalV4State.UTXOs[tokenID]
		broadCastTxs := portalTokenProcessor.ChooseUnshieldIDsFromCandidates(utxos, wReqs)

		// create raw external txs
		for _, bcTx := range broadCastTxs {
			totalFee := uint64(0)
			// prepare outputs for tx
			outputTxs := []*portaltokens.OutputTx{}
			for _, chosenUnshieldID := range bcTx.UnshieldIDs {
				keyWaitingUnshieldRequest := statedb.GenerateWaitingUnshieldRequestObjectKey(tokenID, chosenUnshieldID).String()
				wUnshieldReq := wUnshieldRequests[tokenID][keyWaitingUnshieldRequest]
				outputTxs = append(outputTxs, &portaltokens.OutputTx{
					ReceiverAddress: wUnshieldReq.GetRemoteAddress(),
					Amount:          portalTokenProcessor.ConvertIncToExternalAmount(wUnshieldReq.GetAmount() - feeUnshield),
				})
				totalFee += feeUnshield
			}

			// memo in tx: batchId: combine beacon height and list of unshieldIDs
			batchID := getBatchID(beaconHeight + 1, bcTx.UnshieldIDs)
			memo := batchID

			// create raw tx
			hexRawExtTxStr, _, err := portalTokenProcessor.CreateRawExternalTx(
				bcTx.UTXOs, outputTxs, portalTokenProcessor.ConvertIncToExternalAmount(totalFee), memo, bc)
			if err != nil {
				Logger.log.Errorf("[Batch Unshield Request]: Error when creating raw external tx %v", err)
				continue
			}

			// build new instruction with new raw external tx
			externalFees := map[uint64]uint{
				beaconHeight: uint(totalFee),
			}
			chosenUTXOs := map[string][]*statedb.UTXO {
				portalParams.MultiSigAddresses[tokenID]: bcTx.UTXOs,
			}
			newInst := buildUnshieldBatchingInst(batchID, hexRawExtTxStr, tokenID, bcTx.UnshieldIDs, chosenUTXOs, externalFees, bMeta.PortalUnshieldBatchingMeta, pv4Common.PortalRequestAcceptedChainStatus)
			newInsts = append(newInsts, newInst)

			// update current portal state
			// remove chosen waiting unshield requests from waiting list
			UpdatePortalStateAfterProcessBatchUnshieldRequest(
				currentPortalV4State, batchID, chosenUTXOs, externalFees, bcTx.UnshieldIDs, tokenID, beaconHeight + 1)
		}
	}
	return newInsts, nil
}

func (p *portalUnshieldBatchingProcessor) ProcessInsts(
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
	var actionData pv4Meta.PortalUnshieldRequestBatchContent
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal instruction content %v - Error %v\n", instructions[3], err)
		return nil
	}

	reqStatus := instructions[2]
	if reqStatus == pCommon.PortalRequestAcceptedChainStatus {
		// add new waiting unshield request to waiting list
		UpdatePortalStateAfterProcessBatchUnshieldRequest(
			currentPortalV4State, actionData.BatchID, actionData.UTXOs, actionData.NetworkFee, actionData.UnshieldIDs, actionData.TokenID, beaconHeight + 1)

		// todo: review
		// update bridge/portal token info
		//incTokenID, err := common.Hash{}.NewHashFromStr(actionData.TokenID)
		//if err != nil {
		//	Logger.log.Errorf("ERROR: Can not new hash from porting incTokenID: %+v", err)
		//	return nil
		//}
		//updatingInfo, found := updatingInfoByTokenID[*incTokenID]
		//if found {
		//	updatingInfo.DeductAmt += actionData.UnshieldAmount
		//} else {
		//	updatingInfo = bMeta.UpdatingInfo{
		//		CountUpAmt:      0,
		//		DeductAmt:       actionData.UnshieldAmount,
		//		TokenID:         *incTokenID,
		//		ExternalTokenID: nil,
		//		IsCentralized:   false,
		//	}
		//}
		//updatingInfoByTokenID[*incTokenID] = updatingInfo
	}

	return nil
}
