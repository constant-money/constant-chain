package rpcserver

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/addrmanager"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/connmanager"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/mempool"
	"github.com/incognitochain/incognito-chain/netsync"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/syncker"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/incognitochain/incognito-chain/wire"
	peer2 "github.com/libp2p/go-libp2p-peer"
)

const (
	rpcAuthTimeoutSeconds    = 60
	rpcProcessTimeoutSeconds = 90
	RpcServerVersion         = "1.0"
)

// timeZeroVal is simply the zero value for a time.Time and is used to avoid
// creating multiple instances.
var timeZeroVal time.Time

// UsageFlag define flags that specify additional properties about the
// circumstances under which a command can be used.
type UsageFlag uint32

// rpcServer provides a concurrent safe RPC server to a chain server.
type RpcServer struct {
	HttpServer *HttpServer
	WsServer   *WsServer

	started          int32
	shutdown         int32
	numClients       int32
	numSocketClients int32
	config           RpcServerConfig
	RpcServer        *http.Server

	statusLock  sync.RWMutex
	statusLines map[int]string

	authSHA      []byte
	limitAuthSHA []byte

	// channel
	cRequestProcessShutdown chan struct{}
}

type RpcServerConfig struct {
	HttpListenters  []net.Listener
	WsListenters    []net.Listener
	ProtocolVersion string
	ChainParams     *blockchain.Params
	BlockChain      *blockchain.BlockChain
	Blockgen        *blockchain.BlockGenerator
	MemCache        *memcache.MemoryCache
	Database        map[int]incdb.Database
	Wallet          *wallet.Wallet
	ConnMgr         *connmanager.ConnManager
	AddrMgr         *addrmanager.AddrManager
	NodeMode        string
	NetSync         *netsync.NetSync
	Syncker         *syncker.SynckerManager
	Server          interface {
		// Push TxNormal Message
		PushMessageToAll(message wire.Message) error
		PushMessageToPeer(message wire.Message, id peer2.ID) error
		GetNodeRole() string
		// GetUserKeySet() *incognitokey.KeySet
		EnableMining(enable bool) error
		IsEnableMining() bool
		GetChainMiningStatus(chain int) string
		GetPublicKeyRole(publicKey string, keyType string) (int, int)
		GetIncognitoPublicKeyRole(publicKey string) (int, bool, int)
		GetMinerIncognitoPublickey(publicKey string, keyType string) []byte
	}
	ConsensusEngine interface {
		GetUserLayer() (string, int)
		GetUserRole() (string, string, int)
		GetCurrentMiningPublicKey() (publickey string, keyType string)
		GetAllMiningPublicKeys() []string
		ExtractBridgeValidationData(block types.BlockInterface) ([][]byte, []int, error)
	}
	TxMemPool                   *mempool.TxPool
	RPCMaxClients               int
	RPCMaxWSClients             int
	RPCLimitRequestPerDay       int
	RPCLimitRequestErrorPerHour int
	RPCQuirks                   bool
	// Authentication
	RPCUser      string
	RPCPass      string
	RPCLimitUser string
	RPCLimitPass string
	DisableAuth  bool
	// The fee estimator keeps track of how long transactions are left in
	// the mempool before they are mined into blocks.
	FeeEstimator map[byte]*mempool.FeeEstimator
	// IsMiningNode    bool   // flag mining node. True: mining, False: not mining
	MiningKeys    string // encode of mining key
	PubSubManager *pubsub.PubSubManager
}

func (rpcServer *RpcServer) Init(config *RpcServerConfig) {
	if len(config.HttpListenters) > 0 {
		rpcServer.HttpServer = &HttpServer{}
		rpcServer.HttpServer.Init(config)
	}
	if len(config.WsListenters) > 0 {
		rpcServer.WsServer = &WsServer{}
		rpcServer.WsServer.Init(config)
	}
}
func (rpcServer *RpcServer) Start() {
	if rpcServer.WsServer != nil {
		err := rpcServer.WsServer.Start()
		if err != nil {
			Logger.log.Error(err)
		}
	}
	if rpcServer.HttpServer != nil {
		err := rpcServer.HttpServer.Start()
		if err != nil {
			Logger.log.Error(err)
		}
	}
}
func (rpcServer *RpcServer) Stop() {
	if rpcServer.WsServer != nil {
		rpcServer.WsServer.Stop()
	}
	if rpcServer.HttpServer != nil {
		rpcServer.HttpServer.Stop()
	}
}

// RequestedProcessShutdown returns a channel that is sent to when an authorized
// RPC client requests the process to shutdown.  If the request can not be read
// immediately, it is dropped.
func (rpcServer *RpcServer) RequestedProcessShutdown() <-chan struct{} {
	return rpcServer.cRequestProcessShutdown
}
