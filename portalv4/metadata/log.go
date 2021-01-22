package metadata

import "github.com/incognitochain/incognito-chain/common"

type PortalMetaDataLogger struct {
	log common.Logger
}

func (portalMetaLogger *PortalMetaDataLogger) Init(inst common.Logger) {
	portalMetaLogger.log = inst
}

// Global instant to use
var Logger = PortalMetaDataLogger{}
