package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/incognitochain/incognito-chain/dataaccessobject"
	relaying "github.com/incognitochain/incognito-chain/relaying/bnb"
	btcRelaying "github.com/incognitochain/incognito-chain/relaying/btc"

	"github.com/incognitochain/incognito-chain/syncker"

	"github.com/incognitochain/incognito-chain/addrmanager"
	"github.com/incognitochain/incognito-chain/blockchain"
	main2 "github.com/incognitochain/incognito-chain/blockchain/btc"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/connmanager"
	consensus "github.com/incognitochain/incognito-chain/consensus_v2"
	"github.com/incognitochain/incognito-chain/databasemp"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/mempool"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/netsync"
	"github.com/incognitochain/incognito-chain/peer"
	"github.com/incognitochain/incognito-chain/peerv2"
	"github.com/incognitochain/incognito-chain/peerv2/wrapper"

	//privacy "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/rpcserver"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/trie"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/jrick/logrotate/rotator"
	"github.com/incognitochain/incognito-chain/privacy"
)

var (
	// logRotator is one of the logging outputs.  It should be closed on
	// application shutdown.
	logRotator *rotator.Rotator

	backendLog             = common.NewBackend(logWriter{})
	addrManagerLoger       = backendLog.Logger("Address log", true)
	connManagerLogger      = backendLog.Logger("Connection Manager log", true)
	mainLogger             = backendLog.Logger("Server log", false)
	rpcLogger              = backendLog.Logger("RPC log", false)
	rpcServiceLogger       = backendLog.Logger("RPC service log", false)
	rpcServiceBridgeLogger = backendLog.Logger("RPC service DeBridge log", false)
	netsyncLogger          = backendLog.Logger("Netsync log", false)
	peerLogger             = backendLog.Logger("Peer log", true)
	dbLogger               = backendLog.Logger("Database log", false)
	dbmpLogger             = backendLog.Logger("Mempool Persistence DB log", false)
	walletLogger           = backendLog.Logger("Wallet log", false)
	blockchainLogger       = backendLog.Logger("BlockChain log", false)
	consensusLogger        = backendLog.Logger("Consensus log", false)
	mempoolLogger          = backendLog.Logger("Mempool log", false)
	transactionLogger      = backendLog.Logger("Transaction log", false)
	privacyLogger          = backendLog.Logger("Privacy log", false)
	randomLogger           = backendLog.Logger("RandomAPI log", false)
	bridgeLogger           = backendLog.Logger("DeBridge log", false)
	metadataLogger         = backendLog.Logger("Metadata log", false)
	trieLogger             = backendLog.Logger("Trie log", false)
	peerv2Logger           = backendLog.Logger("Peerv2 log", false)
	relayingLogger         = backendLog.Logger("Relaying log", false)
	wrapperLogger          = backendLog.Logger("Wrapper log", false)
	daov2Logger            = backendLog.Logger("DAO log", false)
	btcRelayingLogger      = backendLog.Logger("BTC relaying log", false)
	synckerLogger          = backendLog.Logger("Syncker log ", false)
	privacyV1Logger        = backendLog.Logger("Privacy V1 log ", false)
	privacyV2Logger        = backendLog.Logger("Privacy V2 log ", false)
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	logRotator.Write(p)
	return len(p), nil
}

func init() {
	// for main thread
	Logger.Init(mainLogger)

	// for other components
	connmanager.Logger.Init(connManagerLogger)
	addrmanager.Logger.Init(addrManagerLoger)
	rpcserver.Logger.Init(rpcLogger)
	rpcservice.Logger.Init(rpcServiceLogger)
	rpcservice.BLogger.Init(rpcServiceBridgeLogger)
	netsync.Logger.Init(netsyncLogger)
	peer.Logger.Init(peerLogger)
	incdb.Logger.Init(dbLogger)
	wallet.Logger.Init(walletLogger)
	blockchain.Logger.Init(blockchainLogger)
	consensus.Logger.Init(consensusLogger)
	mempool.Logger.Init(mempoolLogger)
	main2.Logger.Init(randomLogger)
	transaction.Logger.Init(transactionLogger)
	//privacy.Logger.Init(privacyLogger)
	databasemp.Logger.Init(dbmpLogger)
	blockchain.BLogger.Init(bridgeLogger)
	rpcserver.BLogger.Init(bridgeLogger)
	metadata.Logger.Init(metadataLogger)
	trie.Logger.Init(trieLogger)
	peerv2.Logger.Init(peerv2Logger)
	relaying.Logger.Init(relayingLogger)
	wrapper.Logger.Init(wrapperLogger)
	dataaccessobject.Logger.Init(daov2Logger)
	btcRelaying.Logger.Init(btcRelayingLogger)
	syncker.Logger.Init(synckerLogger)
	privacy.LoggerV1.Init(privacyV1Logger)
	privacy.LoggerV2.Init(privacyV2Logger)
}

// subsystemLoggers maps each subsystem identifier to its associated logger.
var subsystemLoggers = map[string]common.Logger{
	"MAIN": mainLogger,

	"AMGR":              addrManagerLoger,
	"CMGR":              connManagerLogger,
	"RPCS":              rpcLogger,
	"RPCSservice":       rpcServiceLogger,
	"RPCSbridgeservice": rpcServiceBridgeLogger,
	"NSYN":              netsyncLogger,
	"PEER":              peerLogger,
	"DABA":              dbLogger,
	"WALL":              walletLogger,
	"BLOC":              blockchainLogger,
	"CONS":              consensusLogger,
	"MEMP":              mempoolLogger,
	"RAND":              randomLogger,
	"TRAN":              transactionLogger,
	"PRIV":              privacyLogger,
	"DBMP":              dbmpLogger,
	"DEBR":              bridgeLogger,
	"META":              metadataLogger,
	"TRIE":              trieLogger,
	"PEERV2":            peerv2Logger,
	"DAO":               daov2Logger,
	"BTCRELAYING":       btcRelayingLogger,
	"SYNCKER":           synckerLogger,
}

// initLogRotator initializes the logging rotater to write logs to logFile and
// create roll files in the same directory.  It must be called before the
// package-global log rotater variables are used.
func initLogRotator(logFile string) {
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(common.ExitByLogging)
	}
	r, err := rotator.New(logFile, 10*1024, false, 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(common.ExitByLogging)
	}

	logRotator = r
}

// setLogLevel sets the logging level for provided subsystem.  Invalid
// subsystems are ignored.  Uninitialized subsystems are dynamically created as
// needed.
func setLogLevel(subsystemID string, logLevel string) {
	// Ignore invalid subsystems.
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

	// Defaults to info if the log level is invalid.
	level, _ := common.LevelFromString(logLevel)
	logger.SetLevel(level)
}

// setLogLevels sets the log level for all subsystem loggers to the passed
// level.  It also dynamically creates the subsystem loggers as needed, so it
// can be used to initialize the logging system.
func setLogLevels(logLevel string) {
	// Configure all sub-systems with the new logging level.  Dynamically
	// create loggers as needed.
	for subsystemID := range subsystemLoggers {
		setLogLevel(subsystemID, logLevel)
	}
}

type MainLogger struct {
	log common.Logger
}

func (mainLogger *MainLogger) Init(inst common.Logger) {
	mainLogger.log = inst
}

// Global instant to use
var Logger = MainLogger{}
