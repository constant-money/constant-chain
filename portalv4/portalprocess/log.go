package portalprocess

import "github.com/incognitochain/incognito-chain/common"

type PortalInstLogger struct {
	log common.Logger
}

func (portalInstLogger *PortalInstLogger) Init(inst common.Logger) {
	portalInstLogger.log = inst
}

// Global instant to use
var Logger = PortalInstLogger{}
