package rpcserver

import (
	"errors"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/incognitochain/incognito-chain/common"
)

type WsServer struct {
	started      int32
	shutdown     int32
	numWsClients int32
	config       RpcServerConfig
	server       *http.Server
	statusLock   sync.RWMutex
	authSHA      []byte
	limitAuthSHA []byte
	// channel
	cRequestProcessShutdown chan struct{}

	blockService *rpcservice.BlockService
}
type RpcSubResult struct {
	Result interface{}
	Error  *rpcservice.RPCError
}

// Manage All Subcription from one socket connection
type SubcriptionManager struct {
	wsMtx          sync.RWMutex
	subMtx         sync.RWMutex
	subRequestList map[string]map[common.Hash]chan struct{} // String: Subcription Method, Hash: hash from Subcription Params
	ws             *websocket.Conn
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (wsServer *WsServer) Init(config *RpcServerConfig) {
	wsServer.config = *config

	// init service
	wsServer.blockService = &rpcservice.BlockService{
		BlockChain: wsServer.config.BlockChain,
		DB:         wsServer.config.Database,
		MemCache:   wsServer.config.MemCache,
	}
}

func NewSubscriptionManager(ws *websocket.Conn) *SubcriptionManager {
	return &SubcriptionManager{
		subRequestList: make(map[string]map[common.Hash]chan struct{}),
		ws:             ws,
	}
}

// Start is used by rpcserver.go to start the rpc listener.
func (wsServer *WsServer) Start() error {
	if atomic.AddInt32(&wsServer.started, 1) != 1 {
		return rpcservice.NewRPCError(rpcservice.AlreadyStartedError, nil)
	}
	wsServeMux := http.NewServeMux()
	wsServer.server = &http.Server{
		Handler: wsServeMux,
		// Timeout connections which don't complete the initial
		// handshake within the allowed timeframe.
		ReadTimeout: time.Second * rpcAuthTimeoutSeconds,
	}
	wsServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		wsServer.handleWsRequest(w, r)
	})
	for _, listen := range wsServer.config.WsListenters {
		go func(listen net.Listener) {
			Logger.log.Infof("RPC Websocket server listening on %s", listen.Addr())
			go wsServer.server.Serve(listen)
			Logger.log.Infof("RPC Websocket listener done for %s", listen.Addr())
		}(listen)
	}
	wsServer.started = 1
	return nil
}

// Stop is used by rpcserver.go to stop the rpc listener.
func (wsServer *WsServer) Stop() {
	if atomic.AddInt32(&wsServer.shutdown, 1) != 1 {
		Logger.log.Info("RPC server is already in the process of shutting down")
	}
	Logger.log.Info("RPC server shutting down")
	if wsServer.started != 0 {
		wsServer.server.Close()
	}
	for _, listen := range wsServer.config.HttpListenters {
		listen.Close()
	}
	Logger.log.Warn("RPC server shutdown complete")
	wsServer.started = 0
	wsServer.shutdown = 1
}

/*
Handle all ws request to rpcserver
*/
// @NOTICE: no auth for this version yet
func (wsServer *WsServer) handleWsRequest(w http.ResponseWriter, r *http.Request) {
	if wsServer.limitWsConnections(w, r.RemoteAddr) {
		return
	}
	// Keep track of the number of connected clients.
	wsServer.IncrementWsClients()
	defer wsServer.DecrementWsClients()

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	wsServer.ProcessRpcWsRequest(ws)
}

func (wsServer *WsServer) limitWsConnections(w http.ResponseWriter, remoteAddr string) bool {
	if int(atomic.LoadInt32(&wsServer.numWsClients)+1) > wsServer.config.RPCMaxWSClients {
		Logger.log.Infof("Max RPC Web Socket exceeded [%d] - "+
			"disconnecting client %s", wsServer.config.RPCMaxClients,
			remoteAddr)
		http.Error(w, "503 Too busy.  Try again later.",
			http.StatusServiceUnavailable)
		return true
	}
	return false
}

func (wsServer *WsServer) IncrementWsClients() {
	atomic.AddInt32(&wsServer.numWsClients, 1)
}

func (wsServer *WsServer) DecrementWsClients() {
	atomic.AddInt32(&wsServer.numWsClients, -1)
}

func (wsServer *WsServer) ProcessRpcWsRequest(ws *websocket.Conn) {
	if atomic.LoadInt32(&wsServer.shutdown) != 0 {
		return
	}
	defer ws.Close()
	// one sub manager will manage connection and subcription with one client (one websocket connection)
	subManager := NewSubscriptionManager(ws)
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				Logger.log.Infof("Websocket Connection Closed from client %+v \n", ws.RemoteAddr())
				return
			} else {
				Logger.log.Info("Websocket Connection from Client %+v counter error %+v \n", ws.RemoteAddr(), err)
				continue
			}
		}
		Logger.log.Infof("Handle Websocket Connection from Client %+v ", ws.RemoteAddr())
		subRequest, jsonErr := parseSubcriptionRequest(msg)
		if jsonErr == nil {
			if subRequest.Type == 0 {
				go wsServer.subscribe(subManager, subRequest, msgType)
			}
			if subRequest.Type == 1 {
				go wsServer.unsubscribe(subManager, subRequest, msgType)
				if err != nil {

				}
			}
		} else {
			Logger.log.Errorf("RPC function process with err \n %+v", jsonErr)
		}
	}
}

func (wsServer *WsServer) subscribe(subManager *SubcriptionManager, subRequest *SubcriptionRequest, msgType int) {
	var cResult chan RpcSubResult
	var closeChan = make(chan struct{})
	defer func() {
		close(closeChan)
	}()
	var jsonErr error
	request := subRequest.JsonRequest
	if request.Id == nil && !(wsServer.config.RPCQuirks && request.Jsonrpc == "") {
		return
	}
	// Attempt to parse the JSON-RPC request into a known concrete command.
	command := WsHandler[request.Method]
	if command == nil {
		jsonErr = rpcservice.NewRPCError(rpcservice.RPCMethodNotFoundError, errors.New("Method"+request.Method+"Not found"))
		Logger.log.Errorf("RPC from client %+v error %+v", subManager.ws.RemoteAddr(), jsonErr)
		//Notify user, method not found
		res, err := createMarshalledSubResponse(subRequest, nil, jsonErr)
		if err != nil {
			Logger.log.Errorf("Failed to marshal reply: %s", err.Error())
			return
		}
		subManager.wsMtx.Lock()
		if err := subManager.ws.WriteMessage(msgType, res); err != nil {
			Logger.log.Errorf("Failed to write reply message: %+v", err)
			subManager.wsMtx.Unlock()
			return
		}
		subManager.wsMtx.Unlock()
		return
	} else {
		cResult = make(chan RpcSubResult)
		// push this subscription to subscription list
		err := AddSubscription(subManager, subRequest, closeChan)
		if err != nil {
			Logger.log.Errorf("Json Params Hash Error %+v, Closing Websocket from Client %+v \n", err, subManager.ws.RemoteAddr())
			close(cResult)
			return
		}
		// Run RPC websocket method
		go command(wsServer, request.Params, subRequest.Subcription, cResult, closeChan)
		// when rpc method has result, it will deliver it to this channel
		for subResult := range cResult {
			result := subResult.Result
			jsonErr := subResult.Error
			res, err := createMarshalledSubResponse(subRequest, result, jsonErr)
			if err != nil {
				Logger.log.Errorf("Failed to marshal reply: %s", err.Error())
				break
			}
			subManager.wsMtx.Lock()
			if err := subManager.ws.WriteMessage(msgType, res); err != nil {
				Logger.log.Errorf("Failed to write reply message: %+v", err)
				subManager.wsMtx.Unlock()
				break
			}
			subManager.wsMtx.Unlock()
		}
		return
	}
}

func (wsServer *WsServer) unsubscribe(subManager *SubcriptionManager, subRequest *SubcriptionRequest, msgType int) {
	subManager.subMtx.Lock()
	defer subManager.subMtx.Unlock()
	var done = true
	var jsonErr error
	hash, err := common.HashArrayInterface(subRequest.JsonRequest.Params)
	if err != nil {
		done = false
	}
	if paramsList, ok := subManager.subRequestList[subRequest.JsonRequest.Method]; ok {
		if closeCh, ok := paramsList[hash]; ok {
			closeCh <- struct{}{}
			delete(paramsList, hash)
			return
		} else {
			done = false
		}
	} else {
		done = false
	}
	if !done {
		if err != nil {
			jsonErr = rpcservice.NewRPCError(rpcservice.UnsubcribeError, err)
		} else {
			jsonErr = rpcservice.NewRPCError(rpcservice.UnsubcribeError, errors.New("No Subcription Found"))
		}
		res, err := createMarshalledSubResponse(subRequest, nil, jsonErr)
		if err != nil {
			Logger.log.Errorf("Failed to marshal reply: %s", err.Error())
		}
		subManager.wsMtx.Lock()
		if err := subManager.ws.WriteMessage(msgType, res); err != nil {
			Logger.log.Errorf("Failed to write reply message: %+v", err)
			subManager.wsMtx.Unlock()
		}
		subManager.wsMtx.Unlock()
	}
}
func RemoveSubcription(subManager *SubcriptionManager, subRequest *SubcriptionRequest) error {
	subManager.subMtx.Lock()
	defer subManager.subMtx.Unlock()
	hash, err := common.HashArrayInterface(subRequest.JsonRequest.Params)
	if err != nil {
		return err
	}
	if paramsList, ok := subManager.subRequestList[subRequest.JsonRequest.Method]; ok {
		if _, ok := paramsList[hash]; ok {
			delete(paramsList, hash)
		}
	}
	return nil
}

func AddSubscription(subManager *SubcriptionManager, subRequest *SubcriptionRequest, closeChan chan struct{}) error {
	subManager.subMtx.Lock()
	defer subManager.subMtx.Unlock()
	hash, err := common.HashArrayInterface(subRequest.JsonRequest.Params)
	if err != nil {
		return err
	}
	if _, ok := subManager.subRequestList[subRequest.JsonRequest.Method]; !ok {
		subManager.subRequestList[subRequest.JsonRequest.Method] = make(map[common.Hash]chan struct{})
	}
	subManager.subRequestList[subRequest.JsonRequest.Method][hash] = closeChan
	return nil
}
