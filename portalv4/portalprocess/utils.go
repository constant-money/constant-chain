package portalprocess

import (
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type CurrentPortalV4State struct {
	WaitingUnshieldRequests   map[string]*statedb.WaitingUnshield      // key : hash(tokenID)
	WalletState               map[string]*statedb.MultisigWalletsState // key : hash(tokenID)
	UnshieldRequestsProcessed map[string]*statedb.ProcessUnshield      // key : hash(tokenID)
}
