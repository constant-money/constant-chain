package rpcserver

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/pkg/errors"
	"io"
	"net"
	"os"
)

func (httpServer *HttpServer) handleSetBackup(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramArray, ok := params.([]interface{})
	if ok && len(paramArray) == 1 {
		setBackup, ok := paramArray[0].(bool)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("set backup is invalid"))
		}
		httpServer.config.ChainParams.IsBackup = setBackup
		return setBackup, nil
	}
	return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("no param"))
}

func (httpServer *HttpServer) handleGetLatestBackup(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramArray, ok := params.([]interface{})
	//fmt.Println("handleGetLatestBackup", paramArray)
	if ok && len(paramArray) == 1 {

		chainName, ok := paramArray[0].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("chainName is invalid"))
		}
		epoch, _ := httpServer.config.BlockChain.GetBeaconChainDatabase().LatestBackup(fmt.Sprintf("../../backup/%v", chainName))
		return struct {
			LatestEpoch int
		}{
			epoch,
		}, nil
	}

	return 0, nil
}

func (httpServer *HttpServer) handleDownloadBackup(conn net.Conn, params interface{}) {
	paramArray, ok := params.([]interface{})
	if ok && len(paramArray) >= 1 {
		chainName, ok := paramArray[0].(string)
		if !ok {
			return
		}
		var fd *os.File
		var err error
		if len(paramArray) == 1 {
			_, filepath := httpServer.config.BlockChain.GetBeaconChainDatabase().LatestBackup(fmt.Sprintf("../../backup/%v", chainName))
			fd, err = os.Open(filepath)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer fd.Close()
		} else if len(paramArray) == 2 {
			otherChain, ok := paramArray[1].(string)
			if !ok {
				return
			}
			_, filepath := httpServer.config.BlockChain.GetBeaconChainDatabase().LatestBackup(fmt.Sprintf("../../backup/%v", otherChain))
			fd, err = os.Open(filepath)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer fd.Close()
		}
		_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\n\r\n"))
		if err != nil {
			return
		}
		_, err = io.Copy(conn, fd)
		if err != nil {
			return
		}
	}
	return
}
