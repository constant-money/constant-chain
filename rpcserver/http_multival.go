package rpcserver

import (
	"errors"

	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

func (httpServer *HttpServer) handleGetValKeyState(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	if httpServer.config.DisableAuth {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("Auth is no enable"))
	}
	states := httpServer.config.ConsensusEngine.GetAllValidatorKeyState()
	return states, nil
}
