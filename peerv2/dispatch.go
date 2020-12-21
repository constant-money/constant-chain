package peerv2

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/peer"
	"github.com/incognitochain/incognito-chain/wire"
	libp2p "github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
)

type Dispatcher struct {
	MessageListeners   *MessageListeners
	PublishableMessage []string
	BC                 *blockchain.BlockChain
	CurrentHWPeerID    libp2p.ID
}

//TODO hy parse msg here
// processInMessageString - this is sub-function of InMessageHandler
// after receiving a good message from stream,
// we need analyze it and process with corresponding message type
func (d *Dispatcher) processInMessageString(msgStr string) error {
	// NOTE: copy from peerConn.processInMessageString
	// Parse Message header from last 24 bytes header message
	jsonDecodeBytesRaw, err := hex.DecodeString(msgStr)
	if err != nil {
		return errors.Wrapf(err, "msgStr: %v", msgStr)
	}

	// TODO(0xbunyip): separate caching from peerConn
	// // cache message hash
	// hashMsgRaw := common.HashH(jsonDecodeBytesRaw).String()
	// if peerConn.listenerPeer != nil {
	// 	if err := peerConn.listenerPeer.HashToPool(hashMsgRaw); err != nil {
	// 		Logger.Error(err)
	// 		return NewPeerError(HashToPoolError, err, nil)
	// 	}
	// }
	// unzip data before process
	jsonDecodeBytes, err := common.GZipToBytes(jsonDecodeBytesRaw)
	if err != nil {
		return errors.WithStack(err)
	}

	// fmt.Printf("In message content : %s", string(jsonDecodeBytes))

	// Parse Message body
	messageBody := jsonDecodeBytes[:len(jsonDecodeBytes)-wire.MessageHeaderSize]

	messageHeader := jsonDecodeBytes[len(jsonDecodeBytes)-wire.MessageHeaderSize:]

	// get cmd type in header message
	commandInHeader := bytes.Trim(messageHeader[:wire.MessageCmdTypeSize], "\x00")
	commandType := string(messageHeader[:len(commandInHeader)])
	// convert to particular message from message cmd type
	message, err := wire.MakeEmptyMessage(string(commandType))
	if err != nil {
		return errors.WithStack(err)
	}

	if len(jsonDecodeBytes) > message.MaxPayloadLength(wire.Version) {
		return errors.Errorf("Message size too lagre %v, it must be less than %v", len(jsonDecodeBytes), message.MaxPayloadLength(wire.Version))
	}
	// check forward TODO
	/*if peerConn.config.MessageListeners.GetCurrentRoleShard != nil {
		cRole, cShard := peerConn.config.MessageListeners.GetCurrentRoleShard()
		if cShard != nil {
			fT := messageHeader[wire.MessageCmdTypeSize]
			if fT == MessageToShard {
				fS := messageHeader[wire.MessageCmdTypeSize+1]
				if *cShard != fS {
					if peerConn.config.MessageListeners.PushRawBytesToShard != nil {
						err1 := peerConn.config.MessageListeners.PushRawBytesToShard(peerConn, &jsonDecodeBytesRaw, *cShard)
						if err1 != nil {
							Logger.Error(err1)
						}
					}
					return NewPeerError(CheckForwardError, err, nil)
				}
			}
		}
		if cRole != "" {
			fT := messageHeader[wire.MessageCmdTypeSize]
			if fT == MessageToBeacon && cRole != "beacon" {
				if peerConn.config.MessageListeners.PushRawBytesToBeacon != nil {
					err1 := peerConn.config.MessageListeners.PushRawBytesToBeacon(peerConn, &jsonDecodeBytesRaw)
					if err1 != nil {
						Logger.Error(err1)
					}
				}
				return NewPeerError(CheckForwardError, err, nil)
			}
		}
	}*/

	err = json.Unmarshal(messageBody, &message)
	if err != nil {
		return errors.WithStack(err)
	}
	realType := reflect.TypeOf(message)
	// fmt.Printf("Cmd message type of struct %s", realType.String())

	// // cache message hash
	// if peerConn.listenerPeer != nil {
	// 	hashMsg := message.Hash()
	// 	if err := peerConn.listenerPeer.HashToPool(hashMsg); err != nil {
	// 		Logger.Error(err)
	// 		return NewPeerError(CacheMessageHashError, err, nil)
	// 	}
	// }

	// process message for each of message type
	errProcessMessage := d.processMessageForEachType(realType, message)
	if errProcessMessage != nil {
		return errors.WithStack(errProcessMessage)
	}

	// MONITOR INBOUND MESSAGE
	//storeInboundPeerMessage(message, time.Now().Unix(), peerConn.remotePeer.GetPeerID())
	return nil
}

// process message for each of message type
func (d *Dispatcher) processMessageForEachType(messageType reflect.Type, message wire.Message) error {
	// NOTE: copy from peerConn.processInMessageString
	Logger.Debugf("Processing msgType %s", message.MessageType())
	peerConn := &peer.PeerConn{}
	peerConn.SetRemotePeerID(d.CurrentHWPeerID)
	//fmt.Printf("[stream2] %v\n", peerConn.GetRemotePeerID())
	switch messageType {
	case reflect.TypeOf(&wire.MessageTx{}):
		if d.MessageListeners.OnTx != nil {
			d.MessageListeners.OnTx(peerConn, message.(*wire.MessageTx))
		}
	case reflect.TypeOf(&wire.MessageTxPrivacyToken{}):
		if d.MessageListeners.OnTxPrivacyToken != nil {
			d.MessageListeners.OnTxPrivacyToken(peerConn, message.(*wire.MessageTxPrivacyToken))
		}
	case reflect.TypeOf(&wire.MessageBlockShard{}):
		// Logger.Infof("Processing msgContent %+v", message.(*wire.MessageBlockShard).Block)
		if d.MessageListeners.OnBlockShard != nil {
			d.MessageListeners.OnBlockShard(peerConn, message.(*wire.MessageBlockShard))
		}
	case reflect.TypeOf(&wire.MessageBlockBeacon{}):
		// Logger.Infof("Processing msgContent %+v", message.(*wire.MessageBlockBeacon).Block)
		if d.MessageListeners.OnBlockBeacon != nil {
			d.MessageListeners.OnBlockBeacon(peerConn, message.(*wire.MessageBlockBeacon))
		}
	case reflect.TypeOf(&wire.MessageCrossShard{}):
		// Logger.Infof("Processing msgContent %+v", message.(*wire.MessageCrossShard).Block)
		if d.MessageListeners.OnCrossShard != nil {
			d.MessageListeners.OnCrossShard(peerConn, message.(*wire.MessageCrossShard))
		}
	case reflect.TypeOf(&wire.MessageGetBlockBeacon{}):
		if d.MessageListeners.OnGetBlockBeacon != nil {
			d.MessageListeners.OnGetBlockBeacon(peerConn, message.(*wire.MessageGetBlockBeacon))
		}
	case reflect.TypeOf(&wire.MessageGetBlockShard{}):
		if d.MessageListeners.OnGetBlockShard != nil {
			d.MessageListeners.OnGetBlockShard(peerConn, message.(*wire.MessageGetBlockShard))
		}
	case reflect.TypeOf(&wire.MessageGetCrossShard{}):
		if d.MessageListeners.OnGetCrossShard != nil {
			d.MessageListeners.OnGetCrossShard(peerConn, message.(*wire.MessageGetCrossShard))
		}
	case reflect.TypeOf(&wire.MessageVersion{}):
		if d.MessageListeners.OnVersion != nil {
			d.MessageListeners.OnVersion(peerConn, message.(*wire.MessageVersion))
		}
	case reflect.TypeOf(&wire.MessageVerAck{}):
		// d.verAckReceived = true
		if d.MessageListeners.OnVerAck != nil {
			d.MessageListeners.OnVerAck(peerConn, message.(*wire.MessageVerAck))
		}
	case reflect.TypeOf(&wire.MessageGetAddr{}):
		if d.MessageListeners.OnGetAddr != nil {
			d.MessageListeners.OnGetAddr(peerConn, message.(*wire.MessageGetAddr))
		}
	case reflect.TypeOf(&wire.MessageAddr{}):
		if d.MessageListeners.OnGetAddr != nil {
			d.MessageListeners.OnAddr(peerConn, message.(*wire.MessageAddr))
		}
	case reflect.TypeOf(&wire.MessageBFT{}):
		if d.MessageListeners.OnBFTMsg != nil {
			d.MessageListeners.OnBFTMsg(peerConn, message.(*wire.MessageBFT))
		}
	case reflect.TypeOf(&wire.MessagePeerState{}):
		if d.MessageListeners.OnPeerState != nil {
			d.MessageListeners.OnPeerState(peerConn, message.(*wire.MessagePeerState))
		}

	// case reflect.TypeOf(&wire.MessageMsgCheck{}):
	// 	err1 := peerConn.handleMsgCheck(message.(*wire.MessageMsgCheck))
	// 	if err1 != nil {
	// 		Logger.Error(err1)
	// 	}
	// case reflect.TypeOf(&wire.MessageMsgCheckResp{}):
	// 	err1 := peerConn.handleMsgCheckResp(message.(*wire.MessageMsgCheckResp))
	// 	if err1 != nil {
	// 		Logger.Error(err1)
	// 	}
	default:
		return errors.Errorf("InMessageHandler Received unhandled message of type % from %v", messageType, peerConn)
	}
	return nil
}

type MessageListeners struct {
	OnTx             func(p *peer.PeerConn, msg *wire.MessageTx)
	OnTxPrivacyToken func(p *peer.PeerConn, msg *wire.MessageTxPrivacyToken)
	OnBlockShard     func(p *peer.PeerConn, msg *wire.MessageBlockShard)
	OnBlockBeacon    func(p *peer.PeerConn, msg *wire.MessageBlockBeacon)
	OnCrossShard     func(p *peer.PeerConn, msg *wire.MessageCrossShard)
	OnGetBlockBeacon func(p *peer.PeerConn, msg *wire.MessageGetBlockBeacon)
	OnGetBlockShard  func(p *peer.PeerConn, msg *wire.MessageGetBlockShard)
	OnGetCrossShard  func(p *peer.PeerConn, msg *wire.MessageGetCrossShard)
	OnVersion        func(p *peer.PeerConn, msg *wire.MessageVersion)
	OnVerAck         func(p *peer.PeerConn, msg *wire.MessageVerAck)
	OnGetAddr        func(p *peer.PeerConn, msg *wire.MessageGetAddr)
	OnAddr           func(p *peer.PeerConn, msg *wire.MessageAddr)

	//PBFT
	OnBFTMsg    func(p *peer.PeerConn, msg wire.Message)
	OnPeerState func(p *peer.PeerConn, msg *wire.MessagePeerState)
}
