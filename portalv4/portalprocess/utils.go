package portalprocess

import (
	"errors"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type CurrentPortalV4State struct {
	WaitingUnshieldRequests   map[string]*statedb.WaitingUnshield      // key : hash(tokenID)
	WalletsState              map[string]*statedb.MultisigWalletsState // key : hash(tokenID)
	UnshieldRequestsProcessed map[string]*statedb.ProcessUnshield      // key : hash(tokenID)
}

//todo:
func InitCurrentPortalV4StateFromDB(
	stateDB *statedb.StateDB,
) (*CurrentPortalV4State, error) {
	return &CurrentPortalV4State{
		WaitingUnshieldRequests:   nil,
		WalletsState:              nil,
		UnshieldRequestsProcessed: nil,
	}, nil
}

func CloneMultisigWallet(wallets map[string]*statedb.MultisigWalletsState) map[string]*statedb.MultisigWalletsState {
	newWallets := make(map[string]*statedb.MultisigWalletsState, len(wallets))
	for key, wallet := range wallets {
		newWallets[key] = statedb.NewMultisigWalletsStateWithValue(
			wallet.GetWallets(),
		)
	}
	return newWallets
}

// UpdateCustodianStateAfterMatchingPortingRequest updates current portal state after requesting ptoken
func UpdateMultisigWalletsStateAfterUserRequestPToken(currentPortalV4State *CurrentPortalV4State, tokenID string, walletAddress string, listUTXO []*statedb.UTXO) error {
	walletsState, ok := currentPortalV4State.WalletsState[tokenID]
	if !ok {
		return errors.New("[UpdateMultisigWalletsStateAfterUserRequestPToken] MultisigWallet not found")
	}
	wallets := walletsState.GetWallets()
	_, found := wallets[walletAddress]
	if !found {
		wallets[walletAddress] = []*statedb.UTXO{}
	}
	wallets[walletAddress] = append(wallets[walletAddress], listUTXO...)
	currentPortalV4State.WalletsState[tokenID].SetWallets(wallets)
	return nil
}

func UpdatePortalStateAfterUnshieldRequest(
	currentPortalV4State *CurrentPortalV4State,
	unshieldID string, tokenID string, remoteAddress string, unshieldAmt uint64) {
	if currentPortalV4State.WaitingUnshieldRequests == nil {
		currentPortalV4State.WaitingUnshieldRequests = map[string]*statedb.WaitingUnshield{}
	}

	if currentPortalV4State.WaitingUnshieldRequests[tokenID] == nil {
		currentPortalV4State.WaitingUnshieldRequests[tokenID] = statedb.NewWaitingUnshieldState()
	}

	currentPortalV4State.WaitingUnshieldRequests[tokenID].SetUnshield(
		statedb.GenerateWaitingWaitingUnshieldObjectKey(unshieldID).String(),
		statedb.NewUnshieldRequestDetailWithValue(remoteAddress, unshieldAmt),
	)
}
