package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/bridgesig"
)

const (
	heartbeatInterval = 10
	heartbeatTimeout  = 120
)

type peer struct {
	id            string
	rawAddress    string
	publicKey     string
	publickeyType string
	firstPing     time.Time
	lastPing      time.Time
}

// rpcServer provides a concurrent safe RPC server to a bootnode server.
type RpcServer struct {
	peers    map[string]*peer // list peers which are still pinging to bootnode continuously
	peersMtx sync.Mutex
	server   *rpc.Server
	Config   RpcServerConfig // config for RPC server
}

type RpcServerConfig struct {
	Port int // rpc port
}

func (rpcServer *RpcServer) Init(config *RpcServerConfig) {
	// get config and init list Peers
	rpcServer.Config = *config
	rpcServer.peers = make(map[string]*peer)
	rpcServer.server = rpc.NewServer()
	// start go routin hertbeat to check invalid peers
	go rpcServer.PeerHeartBeat(heartbeatTimeout)
}

// Start - create handler and add into rpc server
// Listen and serve rpc server with config port
func (rpcServer *RpcServer) Start() error {
	handler := &Handler{rpcServer}
	rpcServer.server.Register(handler)
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", rpcServer.Config.Port))
	if e != nil {
		log.Fatal("listen error:", e)
		return e
	}
	rpcServer.server.Accept(l)
	l.Close()
	return nil
}

// AddOrUpdatePeer - push a connected peer in to list of mem or update an old peer node
func (rpcServer *RpcServer) AddOrUpdatePeer(rawAddress string, publicKeyType string, publicKeyMining string, signDataB58 string) error {
	rpcServer.peersMtx.Lock()
	defer rpcServer.peersMtx.Unlock()
	if signDataB58 != "" && publicKeyMining != "" && rawAddress != "" {

		sigByte, _, err := base58.Base58Check{}.Decode(signDataB58)
		if err != nil {
			return err
		}
		publicKeyByte := []byte(publicKeyMining)
		miningKey := map[string][]byte{}
		if publicKeyType == common.BlsConsensus {
			err := json.Unmarshal(publicKeyByte, &miningKey)
			if err != nil {
				return err
			}
			ecdsaKeyByte, ok := miningKey[common.BridgeConsensus]
			if !ok {
				return errors.New("ECDSA Public key not found")
			}
			valid, err := bridgesig.Verify(ecdsaKeyByte, []byte(rawAddress), sigByte) //, []int{0}, []blsmultisig.PublicKey{publicKeyByte})
			if valid {
				// blsKeyByte, ok := miningKey[common.BlsConsensus]
				// if !ok {
				// 	return errors.New("BLS Public key not found")
				// }
				rpcServer.peers[publicKeyMining] = &peer{
					id:            rpcServer.CombineID(rawAddress, publicKeyMining),
					rawAddress:    rawAddress,
					publicKey:     publicKeyMining,
					publickeyType: publicKeyType,
					firstPing:     time.Now().Local(),
					lastPing:      time.Now().Local(),
				}
			} else {
				log.Println("AddOrUpdatePeer error", err)
				return err
			}
		} else {
			log.Println("AddOrUpdatePeer error", err)
			return err
		}

	}
	return nil
}

// RemovePeerByPbk - remove peer from mem of bootnode
func (rpcServer *RpcServer) RemovePeerByPbk(publicKey string) {
	delete(rpcServer.peers, publicKey)
}

// CombineID - return string = rawAddress of peer + public key in base58check encode of node(run as committee)
// in case node is not running like a committee, we dont have public key of user who running node
// from this, we can check who is committee in network from bootnode if node provide data for bootnode about key
func (rpcServer *RpcServer) CombineID(rawAddress string, publicKey string) string {
	return rawAddress + publicKey
}

// PeerHeartBeat - loop forever after heartbeatInterval to check peers
// which are not connected to remove from bootnode
// use Last Ping time to compare with time.now
func (rpcServer *RpcServer) PeerHeartBeat(heartbeatTimeout int) {
	for {
		now := time.Now().Local()
		if len(rpcServer.peers) > 0 {
		loop:
			for publicKey, peer := range rpcServer.peers {
				if now.Sub(peer.lastPing).Seconds() > float64(heartbeatTimeout) {
					rpcServer.RemovePeerByPbk(publicKey)
					goto loop
				}
			}
		}
		time.Sleep(heartbeatInterval * time.Second)
	}
}
