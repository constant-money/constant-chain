package metadata

import "github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

type PortalUnshieldRequestBatchContent struct {
	BatchID       string // Hash(beaconHeight || UnshieldIDs)
	RawExternalTx string
	TokenID       string
	UnshieldIDs   []string
	UTXOs         map[string][]*statedb.UTXO
	NetworkFee    map[uint64]uint
}

type PortalUnshieldRequestBatchStatus struct {
	BatchID       string // Hash(beaconHeight || UnshieldIDs)
	RawExternalTx string
	BeaconHeight  uint64
	TokenID       string
	UnshieldIDs   []string
	UTXOs         map[string][]*statedb.UTXO
	NetworkFee    map[uint64]uint
	Status        int
}
