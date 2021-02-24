package privacy_conversion

import (
	"github.com/incognitochain/incognito-chain/common"
	bp "github.com/incognitochain/incognito-chain/privacy/privacy_v2/bulletproofs"
)

type PaymentConversionLogger struct {
	Log common.Logger
}

func (logger *PaymentConversionLogger) Init(inst common.Logger) {
	logger.Log = inst
	bp.Logger.Init(inst)
}

const (
	ConversionProofVersion = 255
)

// Global instant to use
var Logger = PaymentConversionLogger{}
