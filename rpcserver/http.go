package rpcserver

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/incdb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

type HttpServer struct {
	started          int32
	shutdown         int32
	numClients       int32
	numSocketClients int32
	config           RpcServerConfig
	server           *http.Server
	statusLock       sync.RWMutex
	statusLines      map[int]string
	authSHA          []byte
	limitAuthSHA     []byte
	// channel
	cRequestProcessShutdown chan struct{}

	// service
	blockService      *rpcservice.BlockService
	outputCoinService *rpcservice.CoinService
	txMemPoolService  *rpcservice.TxMemPoolService
	networkService    *rpcservice.NetworkService
	txService         *rpcservice.TxService
	walletService     *rpcservice.WalletService
	portal            *rpcservice.PortalService
	synkerService     *rpcservice.SynkerService
}

func (httpServer *HttpServer) Init(config *RpcServerConfig) {
	httpServer.config = *config
	httpServer.statusLines = make(map[int]string)
	if config.RPCUser != "" && config.RPCPass != "" {
		login := config.RPCUser + ":" + config.RPCPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		httpServer.authSHA = common.HashB([]byte(auth))
	}
	if config.RPCLimitUser != "" && config.RPCLimitPass != "" {
		login := config.RPCLimitUser + ":" + config.RPCLimitPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		httpServer.limitAuthSHA = common.HashB([]byte(auth))
	}

	// init service
	httpServer.blockService = &rpcservice.BlockService{
		BlockChain: httpServer.config.BlockChain,
		DB:         httpServer.config.Database,
		MemCache:   httpServer.config.MemCache,
	}
	httpServer.outputCoinService = &rpcservice.CoinService{
		BlockChain: httpServer.config.BlockChain,
	}
	httpServer.txMemPoolService = &rpcservice.TxMemPoolService{
		TxMemPool: httpServer.config.TxMemPool,
	}
	httpServer.networkService = &rpcservice.NetworkService{
		ConnMgr: httpServer.config.ConnMgr,
	}
	httpServer.txService = &rpcservice.TxService{
		BlockChain:   httpServer.config.BlockChain,
		Wallet:       httpServer.config.Wallet,
		FeeEstimator: httpServer.config.FeeEstimator,
		TxMemPool:    httpServer.config.TxMemPool,
	}
	httpServer.walletService = &rpcservice.WalletService{
		Wallet:     httpServer.config.Wallet,
		BlockChain: httpServer.config.BlockChain,
	}
	httpServer.synkerService = &rpcservice.SynkerService{
		Synker: config.Syncker,
	}

	httpServer.portal = &rpcservice.PortalService{
		BlockChain: httpServer.config.BlockChain,
	}
}

// Start is used by rpcserver.go to start the rpc listener.
func (httpServer *HttpServer) Start() error {
	if atomic.LoadInt32(&httpServer.started) == 1 {
		return rpcservice.NewRPCError(rpcservice.AlreadyStartedError, nil)
	}
	httpServeMux := http.NewServeMux()
	httpServer.server = &http.Server{
		Handler: httpServeMux,
		// Timeout connections which don't complete the initial
		// handshake within the allowed timeframe.
		ReadTimeout: time.Second * rpcAuthTimeoutSeconds,
	}

	httpServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpServer.handleRequest(w, r)
	})
	for _, listen := range httpServer.config.HttpListenters {
		go func(listen net.Listener) {
			Logger.log.Infof("RPC Http server listening on %s", listen.Addr())
			go func() {
				err := httpServer.server.Serve(listen)
				if err != nil {
					Logger.log.Errorf("Close Http Listener %+v", err)
				}
			}()
			Logger.log.Infof("RPC Http listener done for %s", listen.Addr())
		}(listen)
	}
	atomic.StoreInt32(&httpServer.started, 1)
	return nil
}

// Stop is used by rpcserver.go to stop the rpc listener.
func (httpServer *HttpServer) Stop() {
	if atomic.AddInt32(&httpServer.shutdown, 1) != 1 {
		Logger.log.Info("RPC server is already in the process of shutting down")
	}
	Logger.log.Info("RPC server shutting down")
	if httpServer.started != 0 {
		err := httpServer.server.Close()
		fmt.Println(err)
	}
	for _, listen := range httpServer.config.HttpListenters {
		listen.Close()
	}
	Logger.log.Warn("RPC server shutdown complete")
	atomic.StoreInt32(&httpServer.started, 0)
	atomic.StoreInt32(&httpServer.shutdown, 1)
}

/*
Handle all request to rpcserver
*/
func (httpServer *HttpServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	NewCorsHeader(w)
	r.Close = true

	// Limit the number of connections to max allowed.
	if httpServer.limitConnections(w, r.RemoteAddr) {
		return
	}

	if r.Method == "OPTIONS" || r.Method == "HEAD" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Keep track of the number of connected clients.
	done := make(chan int)
	//before := httpServer.numClients
	httpServer.IncrementClients()
	defer func() {
		httpServer.DecrementClients()
		//fmt.Println("RPCCON:", before, httpServer.numClients)
	}()
	// Check authentication for rpc user
	ok, isLimitUser, err := httpServer.checkAuth(r, true)
	if err != nil || !ok {
		Logger.log.Error(err)
		AuthFail(w)
		return
	}

	go func() {
		httpServer.ProcessRpcRequest(w, r, isLimitUser)
		done <- 1
	}()

	select {
	case <-done:
	case <-time.After(time.Second * rpcProcessTimeoutSeconds):
	}

}

/*
handles reading and responding to RPC messages.
*/

func (httpServer *HttpServer) ProcessRpcRequest(w http.ResponseWriter, r *http.Request, isLimitedUser bool) {
	if atomic.LoadInt32(&httpServer.shutdown) != 0 {
		return
	}

	if httpServer.config.RPCLimitRequestPerDay > 0 {
		// check limit request per day
		if httpServer.checkLimitRequestPerDay(r) {
			errMsg := "Reach limit request per day"
			Logger.log.Error(errMsg)
			errCode := http.StatusTooManyRequests
			http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
			return
		}
	}

	// Read and close the JSON-RPC request body from the caller.
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		errCode := http.StatusBadRequest
		http.Error(w, fmt.Sprintf("%d error reading JSON Message: %+v", errCode, err), errCode)
		return
	}
	// Unfortunately, the http server doesn't provide the ability to
	// change the read deadline for the new connection and having one breaks
	// long polling.  However, not having a read deadline on the initial
	// connection would mean clients can connect and idle forever.  Thus,
	// hijack the connecton from the HTTP server, clear the read deadline,
	// and handle writing the response manually.
	hj, ok := w.(http.Hijacker)
	if !ok {
		errMsg := "webserver doesn't support hijacking"
		Logger.log.Error(errMsg)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		Logger.log.Errorf("Failed to hijack HTTP connection: %s", err.Error())
		Logger.log.Error(err)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+err.Error(), errCode)
		return
	}
	defer conn.Close()
	defer buf.Flush()
	conn.SetReadDeadline(timeZeroVal)

	var jsonErr error
	var result interface{}
	var request *JsonRequest
	request, jsonErr = parseJsonRequest(body, r.Method)

	if jsonErr == nil {
		if request.Id == nil && !(httpServer.config.RPCQuirks && request.Jsonrpc == "") {
			return
		}

		if httpServer.config.RPCLimitRequestErrorPerHour > 0 {
			if httpServer.checkBlackListClientRequestErrorPerHour(r, request.Method) {
				errMsg := "Reach limit request error for method " + request.Method
				Logger.log.Error(errMsg)
				errCode := http.StatusTooManyRequests
				http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
				return
			}
		}

		// Setup a close notifier.  Since the connection is hijacked,
		// the CloseNotifer on the ResponseWriter is not available.
		closeChan := make(chan struct{}, 1)
		go func() {
			_, err := conn.Read(make([]byte, 1))
			if err != nil {
				close(closeChan)
			}
		}()

		// Check if the user is limited and set error if method unauthorized
		if !isLimitedUser {
			if function, ok := LimitedHttpHandler[request.Method]; ok {
				_ = function
				jsonErr = rpcservice.NewRPCError(rpcservice.RPCInvalidMethodPermissionError, errors.New(""))
			}
		}
		if jsonErr == nil {
			if request.Method == "downloadbackup" {
				httpServer.handleDownloadBackup(conn, request.Params)
				return
			}

			// Attempt to parse the JSON-RPC request into a known concrete
			// command.
			command := HttpHandler[request.Method]
			if command == nil {
				if isLimitedUser {
					command = LimitedHttpHandler[request.Method]
				} else {
					result = nil
					jsonErr = rpcservice.NewRPCError(rpcservice.RPCMethodNotFoundError, errors.New("Method not found: "+request.Method))
				}
			}
			if command != nil {
				result, jsonErr = command(httpServer, request.Params, closeChan)
			} else {
				jsonErr = rpcservice.NewRPCError(rpcservice.RPCMethodNotFoundError, errors.New("Method not found: "+request.Method))
			}
		}
	}

	if jsonErr.(*rpcservice.RPCError) != nil && r.Method != "OPTIONS" {
		if jsonErr.(*rpcservice.RPCError).Code == rpcservice.ErrCodeMessage[rpcservice.RPCParseError].Code {
			Logger.log.Errorf("RPC function process with err \n %+v", jsonErr)
			httpServer.writeHTTPResponseHeaders(r, w.Header(), http.StatusBadRequest, buf)
			httpServer.addBlackListClientRequestErrorPerHour(r, request.Method)
			return
		}

		// Logger.log.Errorf("RPC function process with err \n %+v", jsonErr)
		//fmt.Println(request.Method)
		if request.Method != getTransactionByHash {
			Logger.log.Errorf("RPC function process with err \n %+v", jsonErr)
		}
	}

	if jsonErr != nil && jsonErr.(*rpcservice.RPCError) != nil {
		httpServer.addBlackListClientRequestErrorPerHour(r, request.Method)
	}

	// Marshal the response.
	msg, err := createMarshalledResponse(request, result, jsonErr)
	if err != nil {
		Logger.log.Errorf("Failed to marshal reply: %s", err.Error())
		Logger.log.Error(err)
		return
	}

	// Write the response.
	// for testing only
	// w.WriteHeader(http.StatusOK)
	err = httpServer.writeHTTPResponseHeaders(r, w.Header(), http.StatusOK, buf)
	if err != nil {
		Logger.log.Error(err)
		return
	}
	if _, err := buf.Write(msg); err != nil {
		Logger.log.Errorf("Failed to write marshalled reply: %s", err.Error())
		Logger.log.Error(err)
	}

	// Terminate with newline to maintain compatibility with coin Core.
	if err := buf.WriteByte('\n'); err != nil {
		Logger.log.Errorf("Failed to append terminating newline to reply: %s", err.Error())
		Logger.log.Error(err)
	}
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	temp := ""
	if forwarded != "" {
		temp = forwarded
	} else {
		temp = r.RemoteAddr
	}
	if strings.Contains(temp, ":") {
		return strings.Split(temp, ":")[0]
	} else {
		return temp
	}
}

func lookupIp(host string) string {
	addr, err := net.LookupIP(host)
	if err != nil {
		return ""
	} else {
		return addr[0].String()
	}
}

func lookupAddress(ip string) string {
	host, err := net.LookupAddr(ip)
	if err != nil {
		return ""
	} else {
		return host[0]
	}
}

func (httpServer *HttpServer) checkBlackListClientRequestErrorPerHour(r *http.Request, method string) bool {
	if httpServer.config.RPCLimitRequestErrorPerHour == 0 {
		return false
	}
	inBlackList := false
	remoteAddress := getIP(r)
	remoteAddressKey := append([]byte("rpc-blacklist-"), []byte(remoteAddress)...)
	remoteAddressKey = append(remoteAddressKey, []byte(method)...)

	requestCountInByte, _ := httpServer.config.MemCache.Get(remoteAddressKey)
	//if err1 != nil {
	//	Logger.log.Infof("Can not get limit request per day for %s err:%+v", remoteAddress, err1)
	//}
	if requestCountInByte != nil {
		requestCount := common.BytesToInt(requestCountInByte)
		if requestCount > httpServer.config.RPCLimitRequestErrorPerHour {
			// only accept RPCLimitRequestErrorPerHour error request in 1 hour
			inBlackList = true
		}
	}

	return inBlackList
}

func (httpServer *HttpServer) addBlackListClientRequestErrorPerHour(r *http.Request, method string) {
	if httpServer.config.RPCLimitRequestErrorPerHour == 0 {
		return
	}
	// pink list method
	switch method {
	case getBeaconSwapProof, getLatestBeaconSwapProof, getLatestBridgeSwapProof, getBurnProof, getTransactionByHash, getBridgeReqWithStatus:
		return
	}

	remoteAddress := getIP(r)
	remoteAddressKey := append([]byte("rpc-blacklist-"), []byte(remoteAddress)...)
	remoteAddressKey = append(remoteAddressKey, []byte(method)...)

	Logger.log.Infof("Update limit request error per hour for %s on method %s", remoteAddress, method)

	requestCountInByte, _ := httpServer.config.MemCache.Get(remoteAddressKey)
	//if err1 != nil {
	//	Logger.log.Infof("Can not get limit request error per hour for %s err:%+v", remoteAddress, err1)
	//}
	if requestCountInByte != nil {
		requestCount := common.BytesToInt(requestCountInByte)
		requestCount += 1
		requestCountInByte = common.IntToBytes(requestCount)
		httpServer.config.MemCache.Put(remoteAddressKey, requestCountInByte)
	} else {
		requestCount := 1
		requestCountInByte = common.IntToBytes(requestCount)
		err := httpServer.config.MemCache.PutExpired(remoteAddressKey, requestCountInByte, 1*60*60*1000) // cache in 1 hour
		if err != nil {
			Logger.log.Errorf("Can not update limit error request per hour for %s err:%+v", remoteAddress, err)
		}
	}
}

func (httpServer *HttpServer) checkLimitRequestPerDay(r *http.Request) bool {
	if httpServer.config.RPCLimitRequestPerDay == 0 {
		return false
	}
	remoteAddress := getIP(r)
	remoteAddressKey := []byte(remoteAddress)
	requestCountInByte, _ := httpServer.config.MemCache.Get(remoteAddressKey)
	//if err != nil {
	//Logger.log.Info("Can not get limit request per day for %s err:%+v", remoteAddress, err)
	//}
	reachLimit := false
	if requestCountInByte != nil {
		requestCount := common.BytesToInt(requestCountInByte)
		requestCount += 1
		if requestCount > httpServer.config.RPCLimitRequestPerDay {
			reachLimit = true
		}
		requestCountInByte = common.IntToBytes(requestCount)
		httpServer.config.MemCache.Put(remoteAddressKey, requestCountInByte)
	} else {
		requestCount := 1
		requestCountInByte = common.IntToBytes(requestCount)
		err := httpServer.config.MemCache.PutExpired(remoteAddressKey, requestCountInByte, 24*60*60*1000) // cache 1 day
		if err != nil {
			Logger.log.Error("Can not update limit request per day for %s err:%+v", remoteAddress, err)
		}
	}
	return reachLimit
}

// checkAuth checks the HTTP Basic authentication supplied by a wallet
// or RPC client in the HTTP request r.  If the supplied authentication
// does not match the username and password expected, a non-nil error is
// returned.
//
// This check is time-constant.
//
// The first bool return value signifies auth success (true if successful) and
// the second bool return value specifies whether the user can change the state
// of the server (true) or whether the user is limited (false). The second is
// always false if the first is.
func (httpServer *HttpServer) checkAuth(r *http.Request, require bool) (bool, bool, error) {
	if httpServer.config.DisableAuth {
		return true, true, nil
	}
	authhdr := r.Header["Authorization"]
	if len(authhdr) <= 0 {
		if require {
			Logger.log.Warnf("RPC authentication failure from %s",
				r.RemoteAddr)
			return false, false, rpcservice.NewRPCError(rpcservice.AuthFailError, nil)
		}

		return false, false, nil
	}

	authsha := common.HashB([]byte(authhdr[0]))

	// Check for limited auth first as in environments with limited users, those
	// are probably expected to have a higher volume of calls
	limitcmp := subtle.ConstantTimeCompare(authsha[:], httpServer.limitAuthSHA[:])
	if limitcmp == 1 {
		return true, true, nil
	}

	// Check for admin-level auth
	cmp := subtle.ConstantTimeCompare(authsha[:], httpServer.authSHA[:])
	if cmp == 1 {
		return true, false, nil
	}

	// JsonRequest's auth doesn't match either user
	Logger.log.Warnf("RPC authentication failure from %s", r.RemoteAddr)
	return false, false, rpcservice.NewRPCError(rpcservice.AuthFailError, nil)
}

// AuthFail sends a Message back to the client if the http auth is rejected.
func AuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="RPC"`)
	http.Error(w, "401 Unauthorized.", http.StatusUnauthorized)
}
func NewCorsHeader(w http.ResponseWriter) {
	// Set CORS Header
	w.Header().Set("Connection", "close")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Device-Type, Device-Id, Authorization, Accept-Language, Access-Control-Allow-Headers, Access-Control-Allow-Credentials, Access-Control-Allow-Origin, Access-Control-Allow-Methods, *")
	w.Header().Set("Access-Control-Allow-Methods", "POST, PUT, GET, OPTIONS, DELETE")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

// writeHTTPResponseHeaders writes the necessary response headers prior to
// writing an HTTP body given a request to use for protocol negotiation, headers
// to write, a status Code, and a writer.
func (httpServer *HttpServer) writeHTTPResponseHeaders(req *http.Request, headers http.Header, code int, w io.Writer) error {
	_, err := io.WriteString(w, httpServer.httpStatusLine(req, code))
	if err != nil {
		return err
	}
	err = headers.Write(w)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "\r\n")
	return err
}

// httpStatusLine returns a response Status-Line (RFC 2616 Section 6.1)
// for the given request and response status Code.  This function was lifted and
// adapted from the standard library HTTP server Code since it's not exported.
func (httpServer *HttpServer) httpStatusLine(req *http.Request, code int) string {
	// Fast path:
	key := code
	proto11 := req.ProtoAtLeast(1, 1)
	if !proto11 {
		key = -key
	}
	httpServer.statusLock.RLock()
	line, ok := httpServer.statusLines[key]
	httpServer.statusLock.RUnlock()
	if ok {
		return line
	}

	// Slow path:
	proto := "HTTP/1.0"
	if proto11 {
		proto = "HTTP/1.1"
	}
	codeStr := strconv.Itoa(code)
	text := http.StatusText(code)
	if text != "" {
		line = proto + " " + codeStr + " " + text + "\r\n"
		httpServer.statusLock.Lock()
		httpServer.statusLines[key] = line
		httpServer.statusLock.Unlock()
	} else {
		text = "status Code " + codeStr
		line = proto + " " + codeStr + " " + text + "\r\n"
	}
	return line
}

// limitConnections responds with a 503 service unavailable and returns true if
// adding another client would exceed the maximum allow RPC clients.
//
// This function is safe for concurrent access.
func (httpServer *HttpServer) limitConnections(w http.ResponseWriter, remoteAddr string) bool {
	if int(atomic.LoadInt32(&httpServer.numClients)+1) > httpServer.config.RPCMaxClients {
		Logger.log.Infof("Max RPC clients exceeded [%d] - "+
			"disconnecting client %s", httpServer.config.RPCMaxClients,
			remoteAddr)
		http.Error(w, "503 Too busy.  Try again later.",
			http.StatusServiceUnavailable)
		return true
	}
	return false
}

// DecrementClients subtracts one from the number of connected RPC clients.
// Note this only applies to standard clients.
//
// This function is safe for concurrent access.
func (httpServer *HttpServer) DecrementClients() {
	atomic.AddInt32(&httpServer.numClients, -1)
}

// IncrementClients adds one to the number of connected RPC clients.  Note
// this only applies to standard clients.
//
// This function is safe for concurrent access.
func (httpServer *HttpServer) IncrementClients() {
	atomic.AddInt32(&httpServer.numClients, 1)
}

func (httpServer *HttpServer) GetBeaconChainDatabase() incdb.Database {
	return httpServer.config.Database[common.BeaconChainDataBaseID]
}

func (httpServer *HttpServer) GetShardChainDatabase(shardID byte) incdb.Database {
	return httpServer.config.Database[int(shardID)]
}

func (httpServer *HttpServer) GetBlockchain() *blockchain.BlockChain {
	return httpServer.config.BlockChain
}
