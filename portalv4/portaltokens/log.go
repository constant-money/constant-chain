package portaltokens

import "github.com/incognitochain/incognito-chain/common"

type PortalTokenLogger struct {
	log common.Logger
}

func (portalTokenLogger *PortalTokenLogger) Init(inst common.Logger) {
	portalTokenLogger.log = inst
}

// Global instant to use
var Logger = PortalTokenLogger{}