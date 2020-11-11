package instruction

import "github.com/incognitochain/incognito-chain/common"

type instructionLogger struct {
	Log common.Logger
}

func (i *instructionLogger) Init(inst common.Logger) {
	i.Log = inst
}

// Global instant to use
var Logger = instructionLogger{}
