package rpcserver

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

var (
	netAddrs           []common.SimpleAddr
	errHttp            error
	httpServer         = &HttpServer{}
	rpcListener        = []string{"127.0.0.1:9334"}
	bc                 *blockchain.BlockChain
	pb                 = pubsub.NewPubSubManager()
	rpcConfig          = &RpcServerConfig{}
	user               = "admin"
	limitUser          = "admin@123"
	pass               = "autonomous"
	limitPass          = "autonomous@123"
	wrongUser          = "ad"
	wrongPass          = "au"
	header             = make(map[string][]string)
	getBlockchainInfo  = []byte{123, 10, 9, 34, 106, 115, 111, 110, 114, 112, 99, 34, 58, 32, 34, 49, 46, 48, 34, 44, 10, 32, 32, 32, 32, 34, 109, 101, 116, 104, 111, 100, 34, 58, 32, 34, 103, 101, 116, 98, 108, 111, 99, 107, 99, 104, 97, 105, 110, 105, 110, 102, 111, 34, 44, 10, 32, 32, 32, 32, 34, 112, 97, 114, 97, 109, 115, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 105, 100, 34, 58, 32, 49, 10, 125}
	testRPCBodyRequest = &JsonRequest{
		Jsonrpc: "1.0",
		Method:  "testrpcserver",
		Params:  "",
		Id:      1,
	}
	getBlockchainInfoBody = `{"jsonrpc": "1.0","method": "getblockchaininfo","params": "","id": 1}`
	testRpcServerString   = `{"jsonrpc": "1.0","method": "testrpcserver","params": "","id": 1}`
	writeBuf              = bufio.NewWriterSize(NewHijackerResponse(), 1000)
)

type HijackerResponse struct {
	Code          int
	RequestHeader http.Header
}
type FakeReader struct{}

func (h *HijackerResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:9335")
	return conn, bufio.NewReadWriter(&bufio.Reader{}, writeBuf), err
}
func (w *HijackerResponse) Write(data []byte) (n int, err error) {
	return len(data), nil
}
func (w *HijackerResponse) WriteString(data string) (n int, err error) {
	return len(data), nil
}
func (w *HijackerResponse) Header() http.Header {
	return w.RequestHeader
}
func (w *HijackerResponse) WriteHeader(statusCode int) {
	w.Code = statusCode
}
func NewHijackerResponse() *HijackerResponse {
	return &HijackerResponse{
		RequestHeader: make(map[string][]string),
	}
}
func (r *FakeReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("Error Reading")
}

var _ = func() (_ struct{}) {
	fmt.Println("This runs before init()!")
	bc = blockchain.NewBlockChain(&blockchain.Config{}, true)
	bc.IsTest = true
	netAddrs, _ = common.ParseListeners(rpcListener, "tcp")
	listeners := make([]net.Listener, 0, len(netAddrs))
	listenFunc := net.Listen
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			continue
		}
		listeners = append(listeners, listener)
	}
	rpcConfig.HttpListenters = listeners
	rpcConfig.PubSubManager = pb
	rpcConfig.BlockChain = bc
	rpcConfig.RPCMaxClients = 20
	rpcConfig.RPCUser = user
	rpcConfig.RPCPass = pass
	rpcConfig.RPCLimitUser = limitUser
	rpcConfig.RPCLimitPass = limitPass
	rpcConfig.DisableAuth = true
	httpServer.config = *rpcConfig
	Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

func SetNewListenerAddress() {
	rpcListener = []string{"127.0.0.1:9335"}
	netAddrs, _ = common.ParseListeners(rpcListener, "tcp")
	listeners := make([]net.Listener, 0, len(netAddrs))
	listenFunc := net.Listen
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			continue
		}
		listeners = append(listeners, listener)
	}
	rpcConfig.HttpListenters = listeners
}
func ResetHttpServer() {
	httpServer.numClients = 0
	httpServer.shutdown = 0
	httpServer.started = 0
	httpServer.config.RPCPass = ""
	httpServer.config.RPCUser = ""
	httpServer.config.RPCLimitPass = ""
	httpServer.config.RPCLimitUser = ""
	httpServer.config.DisableAuth = true
	httpServer.authSHA = []byte{}
	httpServer.statusLines = make(map[int]string)
	httpServer.limitAuthSHA = []byte{}
}
func TestHttpServerInit(t *testing.T) {
	httpServer.Init(rpcConfig)
	login := user + ":" + pass
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	if bytes.Compare(httpServer.authSHA, common.HashB([]byte(auth))) != 0 {
		t.Fatalf("Expect authSHA to be %+v but get %+v ", common.HashB([]byte(auth)), httpServer.authSHA)
	}
	limitLogin := limitUser + ":" + limitPass
	limitAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(limitLogin))
	if bytes.Compare(httpServer.limitAuthSHA, common.HashB([]byte(limitAuth))) != 0 {
		t.Fatalf("Expect authSHA to be %+v but get %+v ", common.HashB([]byte(limitAuth)), httpServer.limitAuthSHA)
	}
}
func TestHttpServerStart(t *testing.T) {
	ResetHttpServer()
	errHttp = httpServer.Start()
	if errHttp != nil {
		t.Fatalf("Expect no error but get %+v", errHttp)
	}
	errHttp = httpServer.Start()
	if errHttp == nil {
		t.Fatalf("Expect error %+v but get no error", errHttp)
	} else {
		if errHttp.(*rpcservice.RPCError).Code != rpcservice.ErrCodeMessage[rpcservice.AlreadyStartedError].Code {
			t.Fatalf("Expect %+v but get %+v", rpcservice.AlreadyStartedError, errHttp)
		}
	}
	value := atomic.LoadInt32(&httpServer.started)
	if value != 1 {
		t.Fatalf("Expect value to be 1 but get %+v", value)
	}
	httpServer.Stop()
}

func TestHttpServerStop(t *testing.T) {
	ResetHttpServer()
	httpServer.config = *rpcConfig
	errHttp = httpServer.Start()
	if errHttp != nil {
		t.Fatalf("Expect no error but get %+v", errHttp)
	}
	httpServer.Stop()
	start := atomic.LoadInt32(&httpServer.started)
	if start != 0 {
		t.Fatalf("Expect value to be 0 but get %+v", start)
	}
	shutdown := atomic.LoadInt32(&httpServer.shutdown)
	if shutdown != 1 {
		t.Fatalf("Expect value to be 0 but get %+v", shutdown)
	}
}
func TestHttpServerDecrementClients(t *testing.T) {
	ResetHttpServer()
	httpServer.IncrementClients()
	numOfClient := atomic.LoadInt32(&httpServer.numClients)
	if numOfClient != 1 {
		t.Fatalf("Expect num client is 1 but get %+v", numOfClient)
	}
	httpServer.DecrementClients()
	numOfClient = atomic.LoadInt32(&httpServer.numClients)
	if numOfClient != 0 {
		t.Fatalf("Expect num client is 0 but get %+v", numOfClient)
	}
}
func TestHttpServerLimitConnections(t *testing.T) {
	ResetHttpServer()
	atomic.StoreInt32(&httpServer.numClients, int32(httpServer.config.RPCMaxClients))
	w := &httptest.ResponseRecorder{}
	if isOk := httpServer.limitConnections(w, ""); !isOk {
		t.Fatal("Expect limit connection but dont")
	}
	httpServer.numClients = int32(httpServer.config.RPCMaxClients) - 1
	if isOk := httpServer.limitConnections(w, ""); isOk {
		t.Fatal("Expect no limit connection but get limit")
	}
}
func TestHttpServerCheckAuth(t *testing.T) {
	r := &http.Request{
		Header: header,
	}
	ResetHttpServer()
	// disable auth
	if ok, isLimitUser, err := httpServer.checkAuth(r, true); !(err == nil && ok && isLimitUser) {
		t.Fatal("Expect no error because diable auth")
	}
	httpServer.Init(rpcConfig)
	httpServer.config.DisableAuth = false
	limitLogin := limitUser + ":" + limitPass
	login := user + ":" + pass
	r.Header["Authorization"] = []string{"Basic " + base64.StdEncoding.EncodeToString([]byte(limitLogin))}
	if ok, isLimitUser, err := httpServer.checkAuth(r, true); !(err == nil && ok && isLimitUser) {
		t.Fatal("Expect no error, pass auth and limited user", err, ok, isLimitUser)
	}
	r.Header["Authorization"] = []string{"Basic " + base64.StdEncoding.EncodeToString([]byte(login))}
	if ok, isLimitUser, err := httpServer.checkAuth(r, true); !(err == nil && ok && !isLimitUser) {
		t.Fatal("Expect no error, pass auth and limited user", err, ok, isLimitUser)
	}
	r.Header["Authorization"] = []string{}
	if ok, isLimitUser, err := httpServer.checkAuth(r, true); !(err != nil && !ok && !isLimitUser) {
		t.Fatal("Expect no error, pass auth and limited user", err, ok, isLimitUser)
	} else {
		if err.(*rpcservice.RPCError).Code != rpcservice.ErrCodeMessage[rpcservice.AuthFailError].Code {
			t.Fatalf("Expect %+v but get %+v", rpcservice.AuthFailError, err)
		}
	}
	if ok, isLimitUser, err := httpServer.checkAuth(r, false); !(err == nil && !ok && !isLimitUser) {
		t.Fatal("Expect no error, pass auth and limited user", err, ok, isLimitUser)
	}
	r.Header["Authorization"] = []string{wrongUser + ":" + wrongPass}
	if ok, isLimitUser, err := httpServer.checkAuth(r, true); !(err != nil && !ok && !isLimitUser) {
		t.Fatal("Expect no error, pass auth and limited user", err, ok, isLimitUser)
	} else {
		if err.(*rpcservice.RPCError).Code != rpcservice.ErrCodeMessage[rpcservice.AuthFailError].Code {
			t.Fatalf("Expect %+v but get %+v", rpcservice.AuthFailError, err)
		}
	}
}
func TestHttpServerProcessRpcRequest(t *testing.T) {
	ResetHttpServer()
	SetNewListenerAddress()
	httpServer.Init(rpcConfig)
	httpServer.Start()
	w := httptest.NewRecorder()
	testRPCBodyBytes, err := json.Marshal(testRPCBodyRequest)
	if err != nil {
		t.Fatalf("Expect to marshal Json Request %+v", testRPCBodyRequest)
	}
	r := &http.Request{
		Method:        "POST",
		Header:        header,
		Body:          ioutil.NopCloser(&FakeReader{}),
		ContentLength: int64(len(testRPCBodyBytes)),
	}
	r.Header.Set("content-type", "json")
	httpServer.ProcessRpcRequest(w, r, false)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expect code %+v but get %+v", http.StatusBadRequest, w.Code)
	}
	w = httptest.NewRecorder()
	r.Body = ioutil.NopCloser(strings.NewReader(testRpcServerString))
	// not hijack connection
	httpServer.ProcessRpcRequest(w, r, false)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expect code %+v but get %+v", http.StatusInternalServerError, w.Code)
	}
	hijackW := NewHijackerResponse()
	//server start => can dial connection => return connection and no error
	r.Body = ioutil.NopCloser(strings.NewReader(testRpcServerString))
	httpServer.ProcessRpcRequest(hijackW, r, false)
	if hijackW.Code == http.StatusBadRequest || hijackW.Code == http.StatusInternalServerError {
		t.Fatalf("Expect no bad status")
	}
	httpServer.Stop()
	httpServer.shutdown = 0
	httpServer.started = 0
	hijackW = NewHijackerResponse()
	httpServer.ProcessRpcRequest(hijackW, r, false)
	// no server => can not dial => no connection
	if hijackW.Code != http.StatusInternalServerError {
		t.Fatalf("Expect code %+v but get %+v", http.StatusInternalServerError, w.Code)
	}
}
