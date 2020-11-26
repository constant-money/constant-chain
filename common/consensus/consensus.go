package consensus

import "github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"

type MiningState struct {
	Role    string
	Layer   string
	ChainID int
}

type Validator struct {
	MiningKey   signatureschemes.MiningKey
	PrivateSeed string
	State       MiningState
}
