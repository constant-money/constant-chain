package consensus

import "github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"

type MiningState struct {
	Role    string
	Layer   string
	ChainID int
	Index   int32 // Index of this public key in list committee PublicKey
}

type Validator struct {
	MiningKey   signatureschemes.MiningKey
	PrivateSeed string
	State       MiningState
}
