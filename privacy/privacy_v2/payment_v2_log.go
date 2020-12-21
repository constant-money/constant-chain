package privacy_v2

import (
	"github.com/incognitochain/incognito-chain/common"
	bp "github.com/incognitochain/incognito-chain/privacy/privacy_v2/bulletproofs"
)

type PaymentV2Logger struct {
	Log common.Logger
}

func (logger *PaymentV2Logger) Init(inst common.Logger) {
	logger.Log = inst
	bp.Logger.Init(inst)
}

const (
	ConversionProofVersion = 255
)

// Global instant to use
var Logger = PaymentV2Logger{}