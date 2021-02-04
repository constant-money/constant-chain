package portalv4

import (
	"github.com/incognitochain/incognito-chain/portalv4/portaltokens"
	"time"
)

// todo: add more params for portal v4
type PortalParams struct {
	PortalTokens               map[string]portaltokens.PortalTokenProcessor
	PortalReplacementAddress   string
	MaxFeeForEachStep          uint
	TimeSpaceForFeeReplacement time.Duration
}
