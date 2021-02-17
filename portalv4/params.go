package portalv4

import (
	"github.com/incognitochain/incognito-chain/portalv4/portaltokens"
	"time"
)

// todo: add more params for portal v4
type PortalParams struct {
	MultiSigAddresses       map[string]string
	MultiSigScriptHexEncode map[string]string
	PortalTokens            map[string]portaltokens.PortalTokenProcessor

	// for unshielding
	FeeUnshields map[string]uint64
	BatchNumBlks uint

	// for replacement
	PortalReplacementAddress   string
	MaxFeeForEachStep          uint
	TimeSpaceForFeeReplacement time.Duration
}
