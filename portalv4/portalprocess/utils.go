package portalprocess

import (
	"errors"

	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type CurrentPortalV4State struct {
	WaitingUnshieldRequests   map[string]map[string]*statedb.WaitingUnshieldRequest        // tokenID : hash(tokenID || unshieldID) : value
	UTXOs                     map[string]map[string]*statedb.UTXO                          // tokenID : hash(tokenID || walletAddress || txHash || index): value
	ProcessedUnshieldRequests map[string]map[string]*statedb.ProcessedUnshieldRequestBatch // tokenID : hash(tokenID || batchID) : value
	ShieldingExternalTx       map[string]*statedb.ShieldingRequestsState                   // key : hash(tokenID)
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

// UpdateCustodianStateAfterMatchingPortingRequest updates current portal state after requesting ptoken
func UpdateMultisigWalletsStateAfterUserRequestPToken(currentPortalV4State *CurrentPortalV4State, tokenID string, listUTXO []*statedb.UTXO) {
	for _, utxo := range listUTXO {
		walletAddress := utxo.GetWalletAddress()
		txHash := utxo.GetTxHash()
		outputIdx := utxo.GetOutputIndex()
		outputAmount := utxo.GetOutputAmount()
		currentPortalV4State.UTXOs[tokenID][statedb.GenerateUTXOObjectKey(tokenID, walletAddress, txHash, outputIdx).String()] = statedb.NewUTXOWithValue(walletAddress, txHash, outputIdx, outputAmount)
	}
}

// UpdateCustodianStateAfterMatchingPortingRequest updates current portal state after requesting ptoken
func SaveShieldingExternalTxToStateDB(currentPortalV4State *CurrentPortalV4State, tokenID string, shieldingExternalTxHash string, incAddress string, amount uint64) error {
	externalTxHashState, ok := currentPortalV4State.ShieldingExternalTx[tokenID]
	if !ok {
		return errors.New("[SaveShieldingExternalTxToStateDB] TokenID not found")
	}
	requests := externalTxHashState.GetShieldingRequests()
	request := statedb.NewShieldingRequestWithValue(incAddress, amount)
	requests[shieldingExternalTxHash] = request
	currentPortalV4State.ShieldingExternalTx[tokenID].SetShieldingRequests(requests)
	return nil
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
