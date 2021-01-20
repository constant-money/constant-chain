package rpcserver

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

// handleGetPortalWithdrawCollateralProof returns a proof of a tx withdraw external collaterals
func (httpServer *HttpServer) handleGetPortalWithdrawCollateralProof(
	params interface{},
	closeChan <-chan struct{},
) (interface{}, *rpcservice.RPCError) {
	checkPortalV3Error := httpServer.checkEnablePortalV3()
	if checkPortalV3Error != nil {
		return nil, checkPortalV3Error
	}

	listParams, ok := params.([]interface{})
	if !ok || len(listParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}
	data, ok := listParams[0].(map[string]interface{})
	if !ok || len(listParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an map[string]interface{}"))
	}
	txIDParam, ok := data["TxID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param TxID should be a string"))
	}
	txID, err := common.Hash{}.NewHashFromStr(txIDParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	// validate metaType is valid
	metaTypeParam, ok := data["MetadataType"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param MetadataType should be a number"))
	}

	// Get beacon block height from txID
	height, err := httpServer.portal.GetWithdrawCollateralConfirm(*txID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Get portal proof error: %v", err))
	}

	// get withdraw proof
	return retrieveIncProof(int(metaTypeParam), true, height, txID, httpServer)
}

