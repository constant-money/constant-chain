package consensus_v2

import "github.com/incognitochain/incognito-chain/common"

type consensusLogger struct {
	Log common.Logger
}

func (consensusLogger *consensusLogger) Init(inst common.Logger) {
	consensusLogger.Log = inst
}

// Global instant to use
var Logger = consensusLogger{}
