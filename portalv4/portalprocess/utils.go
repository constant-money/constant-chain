package portalprocess

import (
	"errors"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type CurrentPortalV4State struct {
	WaitingUnshieldRequests   map[string]*statedb.WaitingUnshield      // key : hash(tokenID)
	WalletState               map[string]*statedb.MultisigWalletsState // key : hash(tokenID)
	UnshieldRequestsProcessed map[string]*statedb.ProcessUnshield      // key : hash(tokenID)
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
func UpdateMultisigWalletsStateAfterUserRequestPToken(currentPortalState *CurrentPortalV4State, tokenID string, multiWalletsKey string, utxo statedb.UTXO) error {
	walletState, ok := currentPortalState.WalletState[tokenID]
	if !ok {
		return errors.New("[UpdateMultisigWalletsStateAfterUserRequestPToken] MultisigWallet not found")
	}

	if walletState.GetWallets() == nil || walletState.GetWallets()[multiWalletsKey] == nil {
		return errors.New("[UpdateMultisigWalletsStateAfterUserRequestPToken] Can not get wallets")
	}

	curListUTXO := walletState.GetWallets()[multiWalletsKey]
	if curListUTXO == nil {
		curListUTXO = []statedb.UTXO{}
	}
	curListUTXO = append(curListUTXO, utxo)
	currentPortalState.WalletState[multiWalletsKey].SetWalletOutput(multiWalletsKey, curListUTXO)

	return nil
}
