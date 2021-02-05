package portalprocess

import (
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type CurrentPortalV4State struct {
	WaitingUnshieldRequests   map[string]map[string]*statedb.WaitingUnshieldRequest        // tokenID : hash(tokenID || unshieldID) : value
	UTXOs                     map[string]map[string]*statedb.UTXO                          // tokenID : hash(tokenID || walletAddress || txHash || index) : value
	ProcessedUnshieldRequests map[string]map[string]*statedb.ProcessedUnshieldRequestBatch // tokenID : hash(tokenID || batchID) : value
	ShieldingExternalTx       map[string]map[string]*statedb.ShieldingRequest              // tokenID : hash(tokenID || txHash) : value
}

//todo:
func InitCurrentPortalV4StateFromDB(
	stateDB *statedb.StateDB,
) (*CurrentPortalV4State, error) {
	var err error

	// load list of waiting unshielding requests
	waitingUnshieldRequests := map[string]map[string]*statedb.WaitingUnshieldRequest{}
	for _, tokenID := range pv4Common.PortalV4SupportedIncTokenIDs {
		waitingUnshieldRequests[tokenID], err = statedb.GetWaitingUnshieldRequestsByTokenID(stateDB, tokenID)
		if err != nil {
			return nil, err
		}
	}

	return &CurrentPortalV4State{
		WaitingUnshieldRequests:   waitingUnshieldRequests,
		UTXOs:                     nil,
		ProcessedUnshieldRequests: nil,
		ShieldingExternalTx:       nil,
	}, nil
}

func StorePortalV4StateToDB(
	stateDB *statedb.StateDB,
	currentPortalState *CurrentPortalV4State,
) error {
	var err error
	for _, tokenID := range pv4Common.PortalV4SupportedIncTokenIDs {
		err = statedb.StoreWaitingUnshieldRequests(stateDB, currentPortalState.WaitingUnshieldRequests[tokenID])
		if err != nil {
			return err
		}
	}

	return nil
}

func UpdatePortalStateAfterShieldingRequest(currentPortalV4State *CurrentPortalV4State, tokenID string, listUTXO []*statedb.UTXO) {
	if currentPortalV4State.UTXOs == nil {
		currentPortalV4State.UTXOs = map[string]map[string]*statedb.UTXO{}
	}
	if currentPortalV4State.UTXOs[tokenID] == nil {
		currentPortalV4State.UTXOs[tokenID] = map[string]*statedb.UTXO{}
	}
	for _, utxo := range listUTXO {
		walletAddress := utxo.GetWalletAddress()
		txHash := utxo.GetTxHash()
		outputIdx := utxo.GetOutputIndex()
		outputAmount := utxo.GetOutputAmount()
		currentPortalV4State.UTXOs[tokenID][statedb.GenerateUTXOObjectKey(tokenID, walletAddress, txHash, outputIdx).String()] = statedb.NewUTXOWithValue(walletAddress, txHash, outputIdx, outputAmount)
	}
}

func SaveShieldingExternalTxToStateDB(currentPortalV4State *CurrentPortalV4State, tokenID string, shieldingExternalTxHash string, incAddress string, amount uint64) {
	if currentPortalV4State.ShieldingExternalTx == nil {
		currentPortalV4State.ShieldingExternalTx = map[string]map[string]*statedb.ShieldingRequest{}
	}
	if currentPortalV4State.ShieldingExternalTx[tokenID] == nil {
		currentPortalV4State.ShieldingExternalTx[tokenID] = map[string]*statedb.ShieldingRequest{}
	}
	currentPortalV4State.ShieldingExternalTx[tokenID][statedb.GenerateShieldingRequestObjectKey(tokenID, shieldingExternalTxHash).String()] = statedb.NewShieldingRequestWithValue(shieldingExternalTxHash, incAddress, amount)
}

func UpdatePortalStateAfterUnshieldRequest(
	currentPortalV4State *CurrentPortalV4State,
	unshieldID string, tokenID string, remoteAddress string, unshieldAmt uint64, beaconHeight uint64) {
	if currentPortalV4State.WaitingUnshieldRequests == nil {
		currentPortalV4State.WaitingUnshieldRequests = map[string]map[string]*statedb.WaitingUnshieldRequest{}
	}
	if currentPortalV4State.WaitingUnshieldRequests[tokenID] == nil {
		currentPortalV4State.WaitingUnshieldRequests[tokenID] = map[string]*statedb.WaitingUnshieldRequest{}
	}

	keyWaitingUnshieldRequest := statedb.GenerateWaitingUnshieldRequestObjectKey(tokenID, unshieldID).String()
	waitingUnshieldRequest := statedb.NewWaitingUnshieldRequestStateWithValue(remoteAddress, unshieldAmt, unshieldID, beaconHeight)
	currentPortalV4State.WaitingUnshieldRequests[tokenID][keyWaitingUnshieldRequest] = waitingUnshieldRequest
}

func UpdatePortalStateAfterReplaceFeedRequest(
	currentPortalV4State *CurrentPortalV4State, unshieldBatch *statedb.ProcessedUnshieldRequestBatch, beaconHeight uint64, fee uint, tokenIDStr, batchIDStr string) {
	if currentPortalV4State.ProcessedUnshieldRequests == nil {
		currentPortalV4State.ProcessedUnshieldRequests = map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{}
	}
	if currentPortalV4State.ProcessedUnshieldRequests[tokenIDStr] == nil {
		currentPortalV4State.ProcessedUnshieldRequests[tokenIDStr] = map[string]*statedb.ProcessedUnshieldRequestBatch{}
	}
	keyWaitingReplacementRequest := statedb.GenerateProcessedUnshieldRequestBatchObjectKey(tokenIDStr, batchIDStr).String()
	fees := unshieldBatch.GetExternalFees()
	fees[beaconHeight] = fee
	waitingReplacementRequest := statedb.NewProcessedUnshieldRequestBatchWithValue(unshieldBatch.GetUnshieldRequests(), unshieldBatch.GetUTXOs(), fees)
	currentPortalV4State.ProcessedUnshieldRequests[tokenIDStr][keyWaitingReplacementRequest] = waitingReplacementRequest
}

// get latest beaconheight
func GetMaxKeyValue(input map[uint64]uint) (max uint64) {
	max = 0
	for k := range input {
		if k > max {
			max = k
		}
	}
	return max
}

func UpdatePortalStateAfterSubmitConfirmedTx(currentPortalV4State *CurrentPortalV4State, tokenIDStr, batchKey string) {
	delete(currentPortalV4State.ProcessedUnshieldRequests[tokenIDStr], batchKey)
}
