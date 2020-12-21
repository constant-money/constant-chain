package peer

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"strings"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wire"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/protocol"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/patrickmn/go-cache"
)

// ConnState represents the state of the requested connection.
type ConnState uint8

// RemotePeer is present for libp2p node data
type Peer struct {
	messagePoolNew *cache.Cache

	// channel
	cStop           chan struct{}
	cDisconnectPeer chan *PeerConn
	cNewConn        chan *newPeerMsg
	cNewStream      chan *newStreamMsg
	cStopConn       chan struct{}

	// private field
	host             host.Host
	port             string
	config           Config
	targetAddress    ma.Multiaddr
	rawAddress       string
	peerID           peer.ID
	peerConns        map[string]*PeerConn
	peerConnsMtx     *sync.Mutex
	pendingPeers     map[string]*Peer
	pendingPeersMtx  *sync.Mutex
	publicKey        string
	publicKeyType    string
	listeningAddress common.SimpleAddr
	seed             int64
	protocolID       protocol.ID

	// public field
	HandleConnected    func(peerConn *PeerConn)
	HandleDisconnected func(peerConn *PeerConn)
	HandleFailed       func(peerConn *PeerConn)
}

// config is the struct to hold configuration options useful to RemotePeer.
type Config struct {
	MessageListeners MessageListeners
	// UserKeySet       *incognitokey.KeySet
	MaxOutPeers     int
	MaxInPeers      int
	MaxPeers        int
	ConsensusEngine interface {
		GetCurrentMiningPublicKey() (publickey string, keyType string)
		VerifyData(data []byte, sig string, publicKey string, consensusType string) error
		SignDataWithCurrentMiningKey(data []byte) (string, string, string, error)
	}
}

/*
// MessageListeners defines callback function pointers to invoke with message
// listeners for a peer. Any listener which is not set to a concrete callback
// during peer initialization is ignored. Execution of multiple message
// listeners occurs serially, so one callback blocks the execution of the next.
//
// NOTE: Unless otherwise documented, these listeners must NOT directly call any
// blocking calls (such as WaitForShutdown) on the peer instance since the input
// handler goroutine blocks until the callback has completed.  Doing so will
// result in a deadlock.
*/
type MessageListeners struct {
	OnTx             func(p *PeerConn, msg *wire.MessageTx)
	OnTxPrivacyToken func(p *PeerConn, msg *wire.MessageTxPrivacyToken)
	OnBlockShard     func(p *PeerConn, msg *wire.MessageBlockShard)
	OnBlockBeacon    func(p *PeerConn, msg *wire.MessageBlockBeacon)
	OnCrossShard     func(p *PeerConn, msg *wire.MessageCrossShard)
	OnGetBlockBeacon func(p *PeerConn, msg *wire.MessageGetBlockBeacon)
	OnGetBlockShard  func(p *PeerConn, msg *wire.MessageGetBlockShard)
	OnGetCrossShard  func(p *PeerConn, msg *wire.MessageGetCrossShard)
	OnVersion        func(p *PeerConn, msg *wire.MessageVersion)
	OnVerAck         func(p *PeerConn, msg *wire.MessageVerAck)
	OnGetAddr        func(p *PeerConn, msg *wire.MessageGetAddr)
	OnAddr           func(p *PeerConn, msg *wire.MessageAddr)

	//PBFT
	OnBFTMsg             func(p *PeerConn, msg wire.Message)
	OnPeerState          func(p *PeerConn, msg *wire.MessagePeerState)
	PushRawBytesToShard  func(p *PeerConn, msgBytes *[]byte, shard byte) error
	PushRawBytesToBeacon func(p *PeerConn, msgBytes *[]byte) error
	GetCurrentRoleShard  func() (string, *byte)
}

func (peerObj Peer) GetHost() host.Host {
	return peerObj.host
}

func (peerObj Peer) GetPort() string {
	return peerObj.port
}

func (peerObj Peer) GetConfig() Config {
	return peerObj.config
}

func (peerObj *Peer) SetConfig(config Config) {
	peerObj.config = config
}

func (peerObj Peer) GetTargetAddress() ma.Multiaddr {
	return peerObj.targetAddress
}

func (peerObj *Peer) SetTargetAddress(targetAddress ma.Multiaddr) {
	peerObj.targetAddress = targetAddress
}

func (peerObj Peer) GetRawAddress() string {
	return peerObj.rawAddress
}

func (peerObj *Peer) SetRawAddress(rawAddress string) {
	peerObj.rawAddress = rawAddress
}

func (peerObj Peer) GetPeerID() peer.ID {
	return peerObj.peerID
}

func (peerObj *Peer) SetPeerID(peerID peer.ID) {
	peerObj.peerID = peerID
}

func (peerObj Peer) GetPeerConns() map[string]*PeerConn {
	return peerObj.peerConns
}

func (peerObj *Peer) SetPeerConns(data map[string]*PeerConn) {
	if data == nil {
		data = make(map[string]*PeerConn)

	}
	peerObj.peerConns = data
}

func (peerObj Peer) GetPendingPeers() map[string]*Peer {
	return peerObj.pendingPeers
}

func (peerObj *Peer) SetPendingPeers(data map[string]*Peer) {
	if data == nil {
		data = make(map[string]*Peer)

	}
	peerObj.pendingPeers = data
}

func (peerObj Peer) GetPeerConnsMtx() *sync.Mutex {
	return peerObj.peerConnsMtx
}

func (peerObj *Peer) SetPeerConnsMtx(v *sync.Mutex) {
	peerObj.peerConnsMtx = v
}

// GetPublicKey return publicKey and keyType
func (peerObj Peer) GetPublicKey() (string, string) {
	return peerObj.publicKey, peerObj.publicKeyType
}

func (peerObj *Peer) SetPublicKey(publicKey string, keyType string) {
	peerObj.publicKeyType = keyType
	peerObj.publicKey = publicKey
}

func (peerObj Peer) GetListeningAddress() common.SimpleAddr {
	return peerObj.listeningAddress
}

func (peerObj *Peer) SetListeningAddress(v common.SimpleAddr) {
	peerObj.listeningAddress = v
}

func (peerObj *Peer) SetSeed(v int64) {
	peerObj.seed = v
}

func (peerObj *Peer) HashToPool(hash string) error {
	if peerObj.messagePoolNew == nil {
		peerObj.messagePoolNew = cache.New(messageLiveTime, messageCleanupInterval)
	}
	return peerObj.messagePoolNew.Add(hash, 1, messageLiveTime)
}

func (peerObj *Peer) CheckHashPool(hash string) bool {
	_, expiredT, exist := peerObj.messagePoolNew.GetWithExpiration(hash)
	if exist {
		if (expiredT != time.Time{}) {
			return true
		}
	}
	return false
}

/*
Init - init a peer with go libp2p
*/
func (peerObj *Peer) Init(protocolIDStr string) error {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if peerObj.seed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(peerObj.seed))
	}

	// Generate a key pair for this Host. We will use it
	// to obtain a valid Host Id.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return NewPeerError(PeerGenerateKeyPairError, err, peerObj)
	}

	ip := strings.Split(peerObj.listeningAddress.String(), ":")[0]
	if len(ip) == 0 {
		ip = localHost
	}
	Logger.log.Debug(ip)
	port := strings.Split(peerObj.listeningAddress.String(), ":")[1]
	net := peerObj.listeningAddress.Network()
	listeningAddressString := fmt.Sprintf("/%s/%s/tcp/%s", net, ip, port)
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listeningAddressString),
		libp2p.Identity(priv),
	}

	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return NewPeerError(CreateP2PNodeError, err, peerObj)
	}

	// Build Host multiaddress
	mulAddrStr := fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty())

	hostAddr, err := ma.NewMultiaddr(mulAddrStr)
	if err != nil {
		return NewPeerError(CreateP2PAddressError, err, peerObj)
	}

	// Now we can build a full multiaddress to reach this Host
	// by encapsulating both addresses:
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	rawAddress := fmt.Sprintf("%s%s", listeningAddressString, mulAddrStr)
	Logger.log.Debugf("I am listening on %s with PEER Id - %s", rawAddress, basicHost.ID().Pretty())
	pid, err := fullAddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		return NewPeerError(GetPeerIdFromProtocolError, err, peerObj)
	}
	peerID, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Print(err)
		return NewPeerError(GetPeerIdFromProtocolError, err, peerObj)
	}

	peerObj.rawAddress = rawAddress
	peerObj.host = basicHost
	peerObj.port = port
	peerObj.SetTargetAddress(fullAddr)
	peerObj.peerID = peerID
	peerObj.cStop = make(chan struct{}, 1)
	peerObj.cDisconnectPeer = make(chan *PeerConn)
	peerObj.cNewConn = make(chan *newPeerMsg)
	peerObj.cNewStream = make(chan *newStreamMsg)
	peerObj.cStopConn = make(chan struct{})

	peerObj.peerConnsMtx = &sync.Mutex{}
	peerObj.pendingPeersMtx = &sync.Mutex{}

	peerObj.protocolID = protocol.ID(protocolIDStr)
	if peerObj.protocolID == "" {
		return NewPeerError(GetPeerIdFromProtocolError, errors.New("Protocol of peer is empty"), peerObj)
	}
	return nil
}

// Start - start peer to begin waiting for connections from other peers
func (peerObj *Peer) Start() {
	Logger.log.Info("RemotePeer start")
	// ping to bootnode for test env
	Logger.log.Debug("Set stream handler and wait for connection from other peer")
	peerObj.host.SetStreamHandler(peerObj.protocolID, peerObj.pushStream)

	go peerObj.processConn()

	_, ok := <-peerObj.cStop
	if !ok { // stop
		close(peerObj.cStopConn)
		Logger.log.Warnf("PEER server shutdown complete %s", peerObj.peerID)
	}
}

// pushStream - handle function for peer to process a new stream  from connection of other peer
func (peerObj *Peer) pushStream(stream net.Stream) {
	go func(stream net.Stream) {
		newStreamMsg := newStreamMsg{
			stream: stream,
			cConn:  nil,
		}
		peerObj.cNewStream <- &newStreamMsg
	}(stream)
}

func (peerObj *Peer) PushConn(peer *Peer, cConn chan *PeerConn) {
	go func(peer *Peer, cConn chan *PeerConn) {
		newPeerMsg := newPeerMsg{
			peer:  peer,
			cConn: cConn,
		}
		peerObj.cNewConn <- &newPeerMsg
	}(peer, cConn)
}

// processConn - control all channel which correspond to connection and process
func (peerObj *Peer) processConn() {
	for {
		select {
		case <-peerObj.cStopConn:
			Logger.log.Critical("ProcessConn QUIT")
			return
		case newPeerMsg := <-peerObj.cNewConn:
			// fmt.Printf("CONNLog cNewConn Try to connect??? %v %v %v\n", newPeerMsg.peer.peerID.Pretty(), newPeerMsg.peer.rawAddress, newPeerMsg.peer.publicKey)
			Logger.log.Debugf("ProcessConn START CONN %s %s", newPeerMsg.peer.peerID.Pretty(), newPeerMsg.peer.rawAddress)
			cConn := make(chan *PeerConn)
			go func(peerObj *Peer) {
				peerConn, err := peerObj.handleNewConnectionOut(newPeerMsg.peer, cConn)
				if err != nil && peerConn == nil {
					Logger.log.Errorf("Fail in opening stream from PEER Id - %s with err: %s", peerObj.peerID.Pretty(), err.Error())
					// fmt.Printf("CONNLog Fail in opening stream from PEER Id - %s with err: %s\n", peerObj.peerID.Pretty(), err.Error())
				}
			}(peerObj)
			p := <-cConn
			if newPeerMsg.cConn != nil {
				newPeerMsg.cConn <- p
			}
			Logger.log.Debugf("ProcessConn END CONN %s %s", newPeerMsg.peer.peerID.Pretty(), newPeerMsg.peer.rawAddress)
			// fmt.Printf("CONNLog END CONN %s %s\n", newPeerMsg.peer.peerID.Pretty(), newPeerMsg.peer.rawAddress)
			continue
		case newStreamMsg := <-peerObj.cNewStream:
			remotePeerID := newStreamMsg.stream.Conn().RemotePeer()
			Logger.log.Debugf("ProcessConn START STREAM %s", remotePeerID.Pretty())
			cConn := make(chan *PeerConn)
			go peerObj.handleNewStreamIn(newStreamMsg.stream, cConn)
			p := <-cConn
			if newStreamMsg.cConn != nil {
				newStreamMsg.cConn <- p
			}
			Logger.log.Debugf("ProcessConn END STREAM %s", remotePeerID.Pretty())
			continue
		}
	}
}

func (peerObj *Peer) connPending(peer *Peer) {
	peerObj.pendingPeersMtx.Lock()
	defer peerObj.pendingPeersMtx.Unlock()
	peerIDStr := peer.peerID.Pretty()
	peerObj.pendingPeers[peerIDStr] = peer
}

func (peerObj *Peer) connEstablished(peer *Peer) {
	peerObj.pendingPeersMtx.Lock()
	defer peerObj.pendingPeersMtx.Unlock()
	peerIDStr := peer.peerID.Pretty()
	_, ok := peerObj.pendingPeers[peerIDStr]
	if ok {
		delete(peerObj.pendingPeers, peerIDStr)
	}
}

func (peerObj *Peer) connCanceled(peer *Peer) {
	peerObj.peerConnsMtx.Lock()
	peerObj.pendingPeersMtx.Lock()
	defer func() {
		peerObj.peerConnsMtx.Unlock()
		peerObj.pendingPeersMtx.Unlock()
	}()
	peerIDStr := peer.peerID.Pretty()
	_, ok := peerObj.peerConns[peerIDStr]
	if ok {
		delete(peerObj.peerConns, peerIDStr)
	}
	peerObj.pendingPeers[peerIDStr] = peer
}

func (peerObj *Peer) countOfInboundConn() int {
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	ret := 0
	for _, peerConn := range peerObj.peerConns {
		if !peerConn.GetIsOutbound() {
			ret++
		}
	}
	return ret
}

func (peerObj *Peer) countOfOutboundConn() int {
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	ret := 0
	for _, peerConn := range peerObj.peerConns {
		if peerConn.GetIsOutbound() {
			ret++
		}
	}
	return ret
}

func (peerObj *Peer) GetPeerConnByPeerID(peerID string) *PeerConn {
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	peerConn, ok := peerObj.peerConns[peerID]
	if ok {
		return peerConn
	}
	return nil
}

func (peerObj *Peer) setPeerConn(peerConn *PeerConn) {
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	peerIDStr := peerConn.remotePeer.peerID.Pretty()
	internalConnPeer, ok := peerObj.peerConns[peerIDStr]
	if ok {
		if internalConnPeer.getIsConnected() {
			internalConnPeer.close()
		}
		Logger.log.Debugf("SetPeerConn and Remove %s %s", peerIDStr, internalConnPeer.remotePeer.rawAddress)
	}
	//fmt.Println("CONN: setPeerConn", peerIDStr)
	peerObj.peerConns[peerIDStr] = peerConn
}

func (peerObj *Peer) removePeerConn(peerConn *PeerConn) error {
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	peerIDStr := peerConn.remotePeer.peerID.Pretty()
	internalConnPeer, ok := peerObj.peerConns[peerIDStr]
	if ok {
		if internalConnPeer.getIsConnected() {
			internalConnPeer.close()
		}
		delete(peerObj.peerConns, peerIDStr)
		Logger.log.Debugf("RemovePeerConn %s %s", peerIDStr, peerConn.remotePeer.rawAddress)
		return nil
	} else {
		return NewPeerError(UnexpectedError, errors.New(fmt.Sprintf("Can not find %+v", peerIDStr)), nil)
	}
}

// handleNewConnectionOut - main process when receiving a new peer connection,
// this mean we want to connect out to other peer
func (peerObj *Peer) handleNewConnectionOut(otherPeer *Peer, cConn chan *PeerConn) (*PeerConn, error) {
	Logger.log.Debugf("Opening stream to PEER Id - %s", otherPeer.rawAddress)

	otherPeerID := otherPeer.peerID
	peerIDStr := otherPeerID.Pretty()
	peerObj.peerConnsMtx.Lock()
	_, ok := peerObj.peerConns[peerIDStr]
	peerObj.peerConnsMtx.Unlock()
	if ok {
		Logger.log.Debugf("Checked Existed PEER Id - %s", otherPeer.rawAddress)

		if cConn != nil {
			cConn <- nil
		}
		return nil, nil
	}

	if peerIDStr == peerObj.peerID.Pretty() {
		Logger.log.Debugf("Checked My peerObj PEER Id - %s", otherPeer.rawAddress)
		//peerObj.newPeerConnectionMutex.Unlock()

		if cConn != nil {
			cConn <- nil
		}
		return nil, nil
	}

	if peerObj.countOfOutboundConn() >= peerObj.config.MaxOutPeers && peerObj.config.MaxOutPeers > 0 && !ok {
		Logger.log.Debugf("Checked Max Outbound Connection PEER Id - %s", otherPeer.rawAddress)

		//push to pending peers
		peerObj.connPending(otherPeer)

		if cConn != nil {
			cConn <- nil
		}
		return nil, nil
	}

	stream, err := peerObj.host.NewStream(context.Background(), otherPeerID, peerObj.protocolID)
	Logger.log.Debug(otherPeer, stream, err)
	if err != nil {
		if cConn != nil {
			cConn <- nil
		}
		return nil, NewPeerError(OpeningStreamP2PError, err, peerObj)
	}

	remotePeerID := stream.Conn().RemotePeer()

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	peerConn := PeerConn{
		isOutbound:         true, // we are connecting to remote peer -> this is an outbound peer
		remotePeer:         otherPeer,
		remotePeerID:       remotePeerID,
		remoteRawAddress:   otherPeer.rawAddress,
		listenerPeer:       peerObj,
		config:             peerObj.config,
		readWriteStream:    rw,
		cDisconnect:        make(chan struct{}),
		cClose:             make(chan struct{}),
		cRead:              make(chan struct{}),
		cWrite:             make(chan struct{}),
		cMsgHash:           make(map[string]chan bool),
		sendMessageQueue:   make(chan outMsg),
		HandleConnected:    peerObj.handleConnected,
		HandleDisconnected: peerObj.handleDisconnected,
		HandleFailed:       peerObj.handleFailed,
	}

	go peerConn.inMessageHandler(rw)
	go peerConn.outMessageHandler(rw)

	peerObj.setPeerConn(&peerConn)
	defer func() {
		stream.Close()
		err := peerObj.removePeerConn(&peerConn)
		Logger.log.Error(err)
	}()

	peerConn.retryCount = 0
	peerConn.setConnState(connEstablished)

	go peerObj.handleConnected(&peerConn)

	if cConn != nil {
		cConn <- &peerConn
	}

	for {
		select {
		case <-peerConn.cDisconnect:
			Logger.log.Warnf("NewPeerConnection Disconnected Stream PEER Id %s", peerConn.remotePeerID.Pretty())
			return &peerConn, nil
		case <-peerConn.cClose:
			Logger.log.Warnf("NewPeerConnection closed stream PEER Id %s", peerConn.remotePeerID.Pretty())
			go func() {
				_, ok := <-peerConn.cDisconnect
				if !ok {
					Logger.log.Debugf("NewPeerConnection disconnected after closed stream PEER Id %s", peerConn.remotePeerID.Pretty())
					return
				}
			}()
			return &peerConn, nil
		}
	}
	return &peerConn, nil
}

// handleNewStreamIn - this mean we have other peer want to be connect to us(an inbound peer)
// we need to create data about this inbound peer and handle our inbound stream
func (peerObj *Peer) handleNewStreamIn(stream net.Stream, cDone chan *PeerConn) error {
	// Remember to close the stream when we are done.
	defer stream.Close()
	// fmt.Printf("\n\n\n\\n\nSomeone connecting to meeeeeeeeeeeeeeeeeeeeeeeeee \n")
	peerConfig := peerObj.config
	if peerObj.countOfInboundConn() >= peerConfig.MaxInPeers && peerConfig.MaxInPeers > 0 {
		Logger.log.Debugf("Max RemotePeer Inbound Connection")
		fmt.Printf("\n\n\n\\n\nMax RemotePeer Inbound Connection\n\n\n\n\n\n\n")
		if cDone != nil {
			close(cDone)
		}
		return NewPeerError(HandleNewStreamError, errors.New("Max RemotePeer Inbound Connection"), peerObj)
	}

	remotePeerID := stream.Conn().RemotePeer()
	Logger.log.Debugf("PEER %s Received a new stream from OTHER PEER with Id %s", peerObj.host.ID().String(), remotePeerID.Pretty())
	//fmt.Printf("\n\n\n\\n\nReceived a new stream from OTHER PEER with Id %s %s \n\n\n\n\n\n\n", peerObj.host.ID().String(), remotePeerID.Pretty())
	peerObj.peerConnsMtx.Lock()
	_, ok := peerObj.peerConns[remotePeerID.Pretty()]
	peerObj.peerConnsMtx.Unlock()
	if ok {
		Logger.log.Debugf("Received a new stream existed PEER Id - %s", remotePeerID.Pretty())

		if cDone != nil {
			close(cDone)
		}
		return NewPeerError(HandleNewStreamError, errors.New(fmt.Sprintf("Received a new stream existed PEER Id - %s", remotePeerID.Pretty())), nil)
	}

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	peerConn := PeerConn{
		isOutbound:   false, // we are connected from remote peer -> this is an inbound peer
		listenerPeer: peerObj,
		remotePeer: &Peer{
			peerID:     remotePeerID,
			rawAddress: stream.Conn().RemoteMultiaddr().String(),
		},
		config:             peerConfig,
		remotePeerID:       remotePeerID,
		readWriteStream:    rw,
		cDisconnect:        make(chan struct{}),
		cClose:             make(chan struct{}),
		cRead:              make(chan struct{}),
		cWrite:             make(chan struct{}),
		cMsgHash:           make(map[string]chan bool),
		sendMessageQueue:   make(chan outMsg),
		HandleConnected:    peerObj.handleConnected,
		HandleDisconnected: peerObj.handleDisconnected,
		HandleFailed:       peerObj.handleFailed,
	}

	peerObj.setPeerConn(&peerConn)

	go peerConn.inMessageHandler(rw)
	go peerConn.outMessageHandler(rw)

	peerConn.retryCount = 0
	peerConn.setConnState(connEstablished)

	go peerObj.handleConnected(&peerConn)

	if cDone != nil {
		close(cDone)
	}

	defer func() {
		stream.Close()
		err := peerObj.removePeerConn(&peerConn)
		Logger.log.Error(err)
	}()

	for {
		select {
		case <-peerConn.cDisconnect:
			Logger.log.Debugf("HandleStream disconnected stream PEER Id %s", peerConn.remotePeerID.Pretty())
			return nil
		case <-peerConn.cClose:
			Logger.log.Debugf("HandleStream closed stream PEER Id %s", peerConn.remotePeerID.Pretty())
			go func() {
				_, ok := <-peerConn.cDisconnect
				if !ok {
					Logger.log.Debugf("HandleStream disconnected after closed stream PEER Id %s", peerConn.remotePeerID.Pretty())
					return
				}
			}()
			return nil
		}
	}
}

// QueueMessageWithEncoding adds the passed Incognito message to the peer send
// queue. This function is identical to QueueMessage, however it allows the
// caller to specify the wire encoding type that should be used when
// encoding/decoding blocks and transactions.
//
// This function is safe for concurrent access.
func (peerObj *Peer) QueueMessageWithEncoding(msg wire.Message, doneChan chan<- struct{}, msgType byte, msgShard *byte) {
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	for _, peerConnection := range peerObj.peerConns {
		peerConnection.QueueMessageWithEncoding(msg, doneChan, msgType, msgShard)
	}
}

// Stop - stop all features of peer,
// not connect,
// not stream,
// not read and write message on stream
func (peerObj *Peer) Stop() {
	Logger.log.Warnf("Stopping PEER %s", peerObj.peerID.Pretty())

	peerObj.host.Close()
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	for _, peerConn := range peerObj.peerConns {
		peerConn.setConnState(connCanceled)
	}

	close(peerObj.cStop)
	Logger.log.Criticalf("PEER %s stopped", peerObj.peerID.Pretty())
}

// handleConnected - set established flag to a peer when being connected
func (peerObj *Peer) handleConnected(peerConn *PeerConn) {
	Logger.log.Debugf("handleConnected %s", peerConn.remotePeerID.Pretty())
	//fmt.Printf("handleConnected %s", peerConn.remotePeerID.Pretty())
	peerConn.retryCount = 0
	peerConn.setConnState(connEstablished)

	peerObj.connEstablished(peerConn.remotePeer)

	if peerObj.HandleConnected != nil {
		peerObj.HandleConnected(peerConn)
	}
}

// handleDisconnected - handle connected peer when it is disconnected, remove and retry connection
func (peerObj *Peer) handleDisconnected(peerConn *PeerConn) {
	Logger.log.Debugf("handleDisconnected %s", peerConn.remotePeerID.Pretty())
	peerConn.setConnState(connCanceled)
	if peerConn.GetIsOutbound() && !peerConn.getIsForceClose() {
		go peerObj.retryPeerConnection(peerConn)
	}

	if peerObj.HandleDisconnected != nil {
		peerObj.HandleDisconnected(peerConn)
	}
}

// handleFailed - handle when connecting peer failure
func (peerObj *Peer) handleFailed(peerConn *PeerConn) {
	Logger.log.Debugf("handleFailed %s", peerConn.remotePeerID.String())

	peerObj.connCanceled(peerConn.remotePeer)

	if peerObj.HandleFailed != nil {
		peerObj.HandleFailed(peerConn)
	}
}

// retryPeerConnection - retry to connect to peer when being disconnected
func (peerObj *Peer) retryPeerConnection(peerConn *PeerConn) {
	time.AfterFunc(retryConnDuration, func() {
		Logger.log.Debugf("Retry Zero RemotePeer Connection %s", peerConn.remoteRawAddress)
		peerConn.retryCount += 1

		if peerConn.retryCount < maxRetryConn {
			peerConn.setConnState(connPending)
			cConn := make(chan *PeerConn)
			peerConn.listenerPeer.PushConn(peerConn.remotePeer, cConn)
			p := <-cConn
			if p == nil {
				peerConn.retryCount++
				go peerObj.retryPeerConnection(peerConn)
			}
		}
	})
}

// GetPeerConnOfAll - return all Peer connection to other peers
func (peerObj *Peer) GetPeerConnOfAll() []*PeerConn {
	peerObj.peerConnsMtx.Lock()
	defer peerObj.peerConnsMtx.Unlock()
	peerConns := make([]*PeerConn, 0)
	for _, peerConn := range peerObj.peerConns {
		peerConns = append(peerConns, peerConn)
	}
	return peerConns
}
