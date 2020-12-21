package main

import (
	"github.com/incognitochain/incognito-chain/blockchain"
)

// activeNetParams is a pointer to the parameters specific to the
// currently active network.
var activeNetParams = &mainNetParams

// component is used to group parameters for various networks such as the main
// network and test networks.
type params struct {
	*blockchain.Params
	rpcPort string
	wsPort  string
}

var mainNetParams = params{
	Params:  &blockchain.ChainMainParam,
	rpcPort: MainnetRpcServerPort,
	wsPort:  MainnetWsServerPort,
}

var testNetParams = params{
	Params:  &blockchain.ChainTestParam,
	rpcPort: TestnetRpcServerPort,
	wsPort:  TestnetWsServerPort,
}

var testNet2Params = params{
	Params:  &blockchain.ChainTest2Param,
	rpcPort: Testnet2RpcServerPort,
	wsPort:  Testnet2WsServerPort,
}

// netName returns the name used when referring to a coin network.
func netName(chainParams *params) string {
	return chainParams.Name
}
