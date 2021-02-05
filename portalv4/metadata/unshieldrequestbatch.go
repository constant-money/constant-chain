package metadata

import "github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

type PortalUnshieldRequestBatchContent struct {
	BatchID       string // beaconHeight || Hash(RawExternalTx)
	RawExternalTx string
	TokenID       string
	UnshieldIDs   []string
	UTXOs         map[string][]*statedb.UTXO
	NetworkFee    map[uint64]uint
}
