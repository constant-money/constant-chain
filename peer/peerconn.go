package peer

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wire"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/patrickmn/go-cache"
)

var maxMsgProcessPerTime *cache.Cache

func init() {
	maxMsgProcessPerTime = cache.New(1*time.Second, 1*time.Second)
}

type PeerConn struct {
	connState      ConnState
	stateMtx       sync.RWMutex
	verAckReceived bool

	// channel
	sendMessageQueue chan outMsg
	cDisconnect      chan struct{}
	cRead            chan struct{}
	cWrite           chan struct{}
	cClose           chan struct{}
	cMsgHash         map[string]chan bool

	isOutbound      bool
	isOutboundMtx   sync.Mutex
	isForceClose    bool
	isForceCloseMtx sync.Mutex
	readWriteStream *bufio.ReadWriter
	isConnected     bool
	isConnectedMtx  sync.Mutex
	retryCount      int32
	config          Config
	// remote peer info
	remotePeer       *Peer
	remotePeerID     peer.ID
	remoteRawAddress string
	listenerPeer     *Peer
	verValid         bool

	HandleConnected    func(peerConn *PeerConn)
	HandleDisconnected func(peerConn *PeerConn)
	HandleFailed       func(peerConn *PeerConn)

	isUnitTest bool // default = false, use for unit test
}

// Start GET/SET func
func (peerConn *PeerConn) GetIsOutbound() bool {
	peerConn.isOutboundMtx.Lock()
	defer peerConn.isOutboundMtx.Unlock()
	return peerConn.isOutbound
}

func (peerConn *PeerConn) setIsOutbound(isOutbound bool) {
	peerConn.isOutboundMtx.Lock()
	defer peerConn.isOutboundMtx.Unlock()
	peerConn.isOutbound = isOutbound
}

func (peerConn *PeerConn) getIsForceClose() bool {
	peerConn.isForceCloseMtx.Lock()
	defer peerConn.isForceCloseMtx.Unlock()
	return peerConn.isForceClose
}

func (peerConn *PeerConn) setIsForceClose(v bool) {
	peerConn.isForceCloseMtx.Lock()
	defer peerConn.isForceCloseMtx.Unlock()
	peerConn.isForceClose = v
}

func (peerConn *PeerConn) getIsConnected() bool {
	peerConn.isConnectedMtx.Lock()
	defer peerConn.isConnectedMtx.Unlock()
	return peerConn.isConnected
}

func (peerConn *PeerConn) setIsConnected(v bool) {
	peerConn.isConnectedMtx.Lock()
	defer peerConn.isConnectedMtx.Unlock()
	peerConn.isConnected = v
}

func (p *PeerConn) VerAckReceived() bool {
	return p.verAckReceived
}

// updateState updates the state of the connection request.
func (p *PeerConn) setConnState(connState ConnState) {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	p.connState = connState
}

func (p PeerConn) GetRemoteRawAddress() string {
	return p.remotePeer.rawAddress
}

func (p PeerConn) GetRemotePeer() *Peer {
	return p.remotePeer
}

func (p *PeerConn) SetRemotePeer(v *Peer) {
	p.remotePeer = v
}

func (p PeerConn) GetRemotePeerID() peer.ID {
	return p.remotePeerID
}

func (p *PeerConn) SetRemotePeerID(v peer.ID) {
	p.remotePeerID = v
}

func (p PeerConn) GetListenerPeer() *Peer {
	return p.listenerPeer
}

func (p *PeerConn) SetListenerPeer(v *Peer) {
	p.listenerPeer = v
}

func (p *PeerConn) SetVerValid(v bool) {
	p.verValid = v
}

// end GET/SET func

// readString - read data from received message on stream
// and convert to string format
func (peerConn *PeerConn) readString(rw *bufio.ReadWriter, delim byte, maxReadBytes int) (string, error) {
	buf := make([]byte, 0)
	bufL := 0
	// Loop to read byte to byte
	for {
		b, err := rw.ReadByte()
		if err != nil {
			return common.EmptyString, NewPeerError(ReadStringMessageError, err, nil)
		}
		// break byte buf after get a delim
		if b == delim {
			break
		}
		// continue add read byte to buf if not find a delim
		buf = append(buf, b)
		bufL++
		if bufL > maxReadBytes {
			return common.EmptyString, NewPeerError(LimitByteForMessageError, errors.New("limit bytes for message"), nil)
		}
	}

	// convert byte buf to string format
	return string(buf), nil
}

// processInMessageString - this is sub-function of InMessageHandler
// after receiving a good message from stream,
// we need analyze it and process with corresponding message type
func (peerConn *PeerConn) processInMessageString(msgStr string) error {
	// Parse Message header from last 24 bytes header message
	jsonDecodeBytesRaw, err := hex.DecodeString(msgStr)
	if err != nil {
		return NewPeerError(HexDecodeMessageError, err, nil)
	}

	// cache message hash
	hashMsgRaw := common.HashH(jsonDecodeBytesRaw).String()
	if peerConn.listenerPeer != nil {
		if err := peerConn.listenerPeer.HashToPool(hashMsgRaw); err != nil {
			Logger.log.Error(err)
			return NewPeerError(HashToPoolError, err, nil)
		}
	}
	// unzip data before process
	jsonDecodeBytes, err := common.GZipToBytes(jsonDecodeBytesRaw)
	if err != nil {
		Logger.log.Error("Can not unzip from message")
		Logger.log.Error(err)
		return NewPeerError(UnzipMessageError, err, nil)
	}

	Logger.log.Debugf("In message content : %s", string(jsonDecodeBytes))

	// Parse Message body
	messageBody := jsonDecodeBytes[:len(jsonDecodeBytes)-wire.MessageHeaderSize]

	messageHeader := jsonDecodeBytes[len(jsonDecodeBytes)-wire.MessageHeaderSize:]

	// get cmd type in header message
	commandInHeader := bytes.Trim(messageHeader[:wire.MessageCmdTypeSize], "\x00")
	commandType := string(messageHeader[:len(commandInHeader)])
	// convert to particular message from message cmd type
	message, err := wire.MakeEmptyMessage(string(commandType))
	//fmt.Println("RECEIVE:", reflect.TypeOf(message))
	if err != nil {
		Logger.log.Error("Can not find particular message for message cmd type")
		Logger.log.Error(err)
		return NewPeerError(MessageTypeError, err, nil)
	}

	if len(jsonDecodeBytes) > message.MaxPayloadLength(wire.Version) {
		Logger.log.Errorf("Msg size exceed MsgType %s max size, size %+v | max allow is %+v \n", commandType, len(jsonDecodeBytes), message.MaxPayloadLength(1))
		return NewPeerError(MessageTypeError, err, nil)
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
							Logger.log.Error(err1)
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
						Logger.log.Error(err1)
					}
				}
				return NewPeerError(CheckForwardError, err, nil)
			}
		}
	}*/

	err = json.Unmarshal(messageBody, &message)
	if err != nil {
		Logger.log.Error("Can not parse struct from json message")
		Logger.log.Error(err)
		return NewPeerError(ParseJsonMessageError, err, nil)
	}
	realType := reflect.TypeOf(message)
	Logger.log.Debugf("Cmd message type of struct %s", realType.String())

	// cache message hash
	if peerConn.listenerPeer != nil {
		hashMsg := message.Hash()
		if err := peerConn.listenerPeer.HashToPool(hashMsg); err != nil {
			Logger.log.Error(err)
			return NewPeerError(CacheMessageHashError, err, nil)
		}
	}

	// process message for each of message type
	errProcessMessage := peerConn.processMessageForEachType(realType, message)
	if errProcessMessage != nil {
		Logger.log.Error(errProcessMessage)
	}

	// MONITOR INBOUND MESSAGE
	//storeInboundPeerMessage(message, time.Now().Unix(), peerConn.remotePeer.GetPeerID())
	return nil
}

// process message for each of message type
func (peerConn *PeerConn) processMessageForEachType(messageType reflect.Type, message wire.Message) error {
	switch messageType {
	case reflect.TypeOf(&wire.MessageTx{}):
		if peerConn.config.MessageListeners.OnTx != nil {
			peerConn.config.MessageListeners.OnTx(peerConn, message.(*wire.MessageTx))
		}
	case reflect.TypeOf(&wire.MessageTxPrivacyToken{}):
		if peerConn.config.MessageListeners.OnTxPrivacyToken != nil {
			peerConn.config.MessageListeners.OnTxPrivacyToken(peerConn, message.(*wire.MessageTxPrivacyToken))
		}
	case reflect.TypeOf(&wire.MessageBlockShard{}):
		if peerConn.config.MessageListeners.OnBlockShard != nil {
			peerConn.config.MessageListeners.OnBlockShard(peerConn, message.(*wire.MessageBlockShard))
		}
	case reflect.TypeOf(&wire.MessageBlockBeacon{}):
		if peerConn.config.MessageListeners.OnBlockBeacon != nil {
			peerConn.config.MessageListeners.OnBlockBeacon(peerConn, message.(*wire.MessageBlockBeacon))
		}
	case reflect.TypeOf(&wire.MessageCrossShard{}):
		if peerConn.config.MessageListeners.OnCrossShard != nil {
			peerConn.config.MessageListeners.OnCrossShard(peerConn, message.(*wire.MessageCrossShard))
		}
	case reflect.TypeOf(&wire.MessageGetBlockBeacon{}):
		if peerConn.config.MessageListeners.OnGetBlockBeacon != nil {
			peerConn.config.MessageListeners.OnGetBlockBeacon(peerConn, message.(*wire.MessageGetBlockBeacon))
		}
	case reflect.TypeOf(&wire.MessageGetBlockShard{}):
		if peerConn.config.MessageListeners.OnGetBlockShard != nil {
			peerConn.config.MessageListeners.OnGetBlockShard(peerConn, message.(*wire.MessageGetBlockShard))
		}
	case reflect.TypeOf(&wire.MessageGetCrossShard{}):
		if peerConn.config.MessageListeners.OnGetCrossShard != nil {
			peerConn.config.MessageListeners.OnGetCrossShard(peerConn, message.(*wire.MessageGetCrossShard))
		}
	case reflect.TypeOf(&wire.MessageVersion{}):
		if peerConn.config.MessageListeners.OnVersion != nil {
			versionMessage := message.(*wire.MessageVersion)
			peerConn.config.MessageListeners.OnVersion(peerConn, versionMessage)
		}
	case reflect.TypeOf(&wire.MessageVerAck{}):
		peerConn.verAckReceived = true
		if peerConn.config.MessageListeners.OnVerAck != nil {
			peerConn.config.MessageListeners.OnVerAck(peerConn, message.(*wire.MessageVerAck))
		}
	case reflect.TypeOf(&wire.MessageGetAddr{}):
		if peerConn.config.MessageListeners.OnGetAddr != nil {
			peerConn.config.MessageListeners.OnGetAddr(peerConn, message.(*wire.MessageGetAddr))
		}
	case reflect.TypeOf(&wire.MessageAddr{}):
		if peerConn.config.MessageListeners.OnGetAddr != nil {
			peerConn.config.MessageListeners.OnAddr(peerConn, message.(*wire.MessageAddr))
		}
	case reflect.TypeOf(&wire.MessageBFT{}):
		if peerConn.config.MessageListeners.OnBFTMsg != nil {
			peerConn.config.MessageListeners.OnBFTMsg(peerConn, message.(*wire.MessageBFT))
		}
	case reflect.TypeOf(&wire.MessagePeerState{}):
		if peerConn.config.MessageListeners.OnPeerState != nil {
			peerConn.config.MessageListeners.OnPeerState(peerConn, message.(*wire.MessagePeerState))
		}
	case reflect.TypeOf(&wire.MessageMsgCheck{}):
		err1 := peerConn.handleMsgCheck(message.(*wire.MessageMsgCheck))
		if err1 != nil {
			Logger.log.Error(err1)
		}
	case reflect.TypeOf(&wire.MessageMsgCheckResp{}):
		err1 := peerConn.handleMsgCheckResp(message.(*wire.MessageMsgCheckResp))
		if err1 != nil {
			Logger.log.Error(err1)
		}
	default:
		errorMessage := fmt.Sprintf("InMessageHandler Received unhandled message of type % from %v", messageType, peerConn)
		Logger.log.Error(errorMessage)
		return NewPeerError(UnhandleMessageTypeError, errors.New(errorMessage), nil)
	}
	return nil
}

// InMessageHandler - Handle all in-coming message
// We receive a message with stream connection  of peer-to-peer
// convert to string data
// check type object which map with string data
// call corresponding function to process message
func (peerConn *PeerConn) inMessageHandler(rw *bufio.ReadWriter) error {
	peerConn.setIsConnected(true)
	for {
		Logger.log.Debugf("PEER %s (address: %s) Reading stream", peerConn.remotePeer.GetPeerID().Pretty(), peerConn.remotePeer.GetRawAddress())

		str, errR := peerConn.readString(rw, delimMessageByte, spamMessageSize)
		if errR != nil {
			// we has an error when read stream message an can not parse to string data
			peerConn.setIsConnected(false)
			Logger.log.Error("---------------------------------------------------------------------")
			Logger.log.Errorf("InMessageHandler ERROR %s %s", peerConn.remotePeerID.Pretty(), peerConn.remotePeer.GetRawAddress())
			Logger.log.Error(errR)
			Logger.log.Errorf("InMessageHandler QUIT")
			Logger.log.Error("---------------------------------------------------------------------")
			close(peerConn.cWrite)
			return errR
		}

		if str != delimMessageStr {
			// Get an good message, make an process to do something on it
			if !peerConn.isUnitTest {
				// not use for unit test -> call go routine for process
				count := maxMsgProcessPerTime.ItemCount()
				if count > 10000 {
					fmt.Println("DEBUG: maxMsgProcessPerTime exceed 10000")
					continue
				}
				maxMsgProcessPerTime.Add(str, nil, 1*time.Second)
				peerConn.processInMessageString(str)
			} else {
				// not use for unit test -> not call go routine for process
				// and break for loop
				peerConn.processInMessageString(str)
				return nil
			}
		}
	}
}

// OutMessageHandler handles the queuing of outgoing data for the peer. This runs as
// a muxer for various sources of input so we can ensure that server and peer
// handlers will not block on us sending a message.  That data is then passed on
// to outHandler to be actually written.
func (peerConn *PeerConn) outMessageHandler(rw *bufio.ReadWriter) {
	for {
		select {
		case outMsg := <-peerConn.sendMessageQueue:
			{
				//fmt.Println("SEND:", reflect.TypeOf(outMsg.message))

				var sendString string
				if outMsg.rawBytes != nil && len(*outMsg.rawBytes) > 0 {
					Logger.log.Debugf("OutMessageHandler with raw bytes")
					message := hex.EncodeToString(*outMsg.rawBytes)
					message += delimMessageStr
					sendString = message
					Logger.log.Debugf("Send a messageHex raw bytes to %s", peerConn.remotePeer.GetPeerID().Pretty())
				} else {
					// Create and send messageHex
					messageBytes, err := outMsg.message.JsonSerialize()
					if err != nil {
						Logger.log.Error("Can not serialize json format for messageHex:" + outMsg.message.MessageType())
						Logger.log.Error(err)
						continue
					}

					// Add 24 bytes headerBytes into messageHex
					headerBytes := make([]byte, wire.MessageHeaderSize)
					// add command type of message
					cmdType, messageErr := wire.GetCmdType(reflect.TypeOf(outMsg.message))
					if messageErr != nil {
						Logger.log.Error("Can not get cmd type for " + outMsg.message.MessageType())
						Logger.log.Error(messageErr)
						continue
					}
					copy(headerBytes[:], []byte(cmdType))
					// add forward type of message at 13st byte
					copy(headerBytes[wire.MessageCmdTypeSize:], []byte{outMsg.forwardType})
					if outMsg.forwardValue != nil {
						// add forward value at 14st byte
						copy(headerBytes[wire.MessageCmdTypeSize+1:], []byte{*outMsg.forwardValue})
					}
					messageBytes = append(messageBytes, headerBytes...)
					Logger.log.Debugf("OutMessageHandler TYPE %s CONTENT %s", cmdType, string(messageBytes))

					// zip data before send
					messageBytes, err = common.GZipFromBytes(messageBytes)
					if err != nil {
						Logger.log.Error("Can not gzip for messageHex:" + outMsg.message.MessageType())
						Logger.log.Error(err)
						continue
					}
					messageHex := hex.EncodeToString(messageBytes)
					//Logger.log.Debugf("Content in hex encode: %s", string(messageHex))
					// add end character to messageHex (delim '\n')
					messageHex += delimMessageStr

					// send on p2p stream
					Logger.log.Debugf("Send a messageHex %s to %s", outMsg.message.MessageType(), peerConn.remotePeer.GetPeerID().Pretty())
					sendString = messageHex
				}
				//// MONITOR OUTBOUND MESSAGE
				//if outMsg.message != nil {
				//storeOutboundPeerMessage(outMsg.message, time.Now().Unix(), peerConn.remotePeer.GetPeerID())
				//}

				_, err := rw.Writer.WriteString(sendString)
				if err != nil {
					Logger.log.Critical("OutMessageHandler WriteString error", err)
					continue
				}
				err = rw.Writer.Flush()
				if err != nil {
					Logger.log.Critical("OutMessageHandler Flush error", err)
					continue
				}
				continue
			}
		case <-peerConn.cWrite:
			Logger.log.Debugf("OutMessageHandler QUIT %s %s", peerConn.remotePeerID.Pretty(), peerConn.remotePeer.GetRawAddress())
			peerConn.setIsConnected(false)
			close(peerConn.cDisconnect)
			if peerConn.HandleDisconnected != nil {
				go peerConn.HandleDisconnected(peerConn)
			}

			return
		}
	}
}

// checkMessageHashBeforeSend - pre-process message before pushing it into Send Queue
func (peerConn *PeerConn) checkMessageHashBeforeSend(hash string) bool {
	numRetries := 0
BeginCheckHashMessage:
	numRetries++
	bTimeOut := false
	// new model for received response
	peerConn.cMsgHash[hash] = make(chan bool)
	cTimeOut := make(chan struct{})
	bCheck := false
	// send msg for check has
	go func() {
		msgCheck, err := wire.MakeEmptyMessage(wire.CmdMsgCheck)
		if err != nil {
			Logger.log.Error("checkMessageHashBeforeSend error", err)
			return
		}
		msgCheck.(*wire.MessageMsgCheck).HashStr = hash
		peerConn.QueueMessageWithEncoding(msgCheck, nil, MessageToPeer, nil)
	}()
	// set time out for check message
	go func() {
		_, ok := <-time.NewTimer(maxTimeoutCheckHashMessage * time.Second).C
		if !ok {
			if cTimeOut != nil {
				Logger.log.Debugf("checkMessageHashBeforeSend TIMER time out %s", hash)
				bTimeOut = true
				close(cTimeOut)
			}
			return
		}
	}()
	Logger.log.Debugf("checkMessageHashBeforeSend WAIT result check hash %s", hash)
	select {
	case bCheck = <-peerConn.cMsgHash[hash]:
		Logger.log.Debugf("checkMessageHashBeforeSend RECEIVED hash %s bAccept %s", hash, bCheck)
		cTimeOut = nil
		break
	case <-cTimeOut:
		Logger.log.Debugf("checkMessageHashBeforeSend RECEIVED time out %d", numRetries)
		cTimeOut = nil
		bTimeOut = true
		break
	}
	if cTimeOut == nil {
		delete(peerConn.cMsgHash, hash)
	}
	Logger.log.Debugf("checkMessageHashBeforeSend FINISHED check hash %s %s", hash, bCheck)
	if bTimeOut && numRetries < maxRetriesCheckHashMessage {
		goto BeginCheckHashMessage
	}
	return bCheck
}

// QueueMessageWithEncoding adds the passed Incognito message to the peer send
// queue. This function is identical to QueueMessage, however it allows the
// caller to specify the wire encoding type that should be used when
// encoding/decoding blocks and transactions.
//
// This function is safe for concurrent access.
func (peerConn *PeerConn) QueueMessageWithEncoding(msg wire.Message, doneChan chan<- struct{}, forwardType byte, forwardValue *byte) {
	Logger.log.Debugf("QueueMessageWithEncoding %s %s", peerConn.remotePeer.GetPeerID().Pretty(), msg.MessageType())
	go func() {
		if peerConn.getIsConnected() {
			data, _ := msg.JsonSerialize()
			if len(data) >= heavyMessageSize && msg.MessageType() != wire.CmdMsgCheck && msg.MessageType() != wire.CmdMsgCheckResp {
				hash := msg.Hash()
				Logger.log.Debugf("QueueMessageWithEncoding HeavyMessageSize %s %s", hash, msg.MessageType())

				if peerConn.checkMessageHashBeforeSend(hash) {
					peerConn.sendMessageQueue <- outMsg{
						message:      msg,
						doneChan:     doneChan,
						forwardType:  forwardType,
						forwardValue: forwardValue,
					}
				}
			} else {
				peerConn.sendMessageQueue <- outMsg{
					message:      msg,
					doneChan:     doneChan,
					forwardType:  forwardType,
					forwardValue: forwardValue,
				}
			}
		}
	}()
}

// QueueMessageWithBytes -
func (peerConn *PeerConn) QueueMessageWithBytes(msgBytes *[]byte, doneChan chan<- struct{}) {
	Logger.log.Debugf("QueueMessageWithBytes from %s", peerConn.remotePeer.GetPeerID().Pretty())
	if msgBytes == nil || len(*msgBytes) <= 0 {
		return
	}
	go func() {
		if peerConn.getIsConnected() {
			if len(*msgBytes) >= heavyMessageSize+wire.MessageHeaderSize {
				hash := common.HashH(*msgBytes).String()
				Logger.log.Debugf("QueueMessageWithBytes HeavyMessageSize %s", hash)

				if peerConn.checkMessageHashBeforeSend(hash) {
					peerConn.sendMessageQueue <- outMsg{
						rawBytes: msgBytes,
						doneChan: doneChan,
					}
				}
			} else {
				peerConn.sendMessageQueue <- outMsg{
					rawBytes: msgBytes,
					doneChan: doneChan,
				}
			}
		}
	}()
}

// handleMsgCheck -
func (p *PeerConn) handleMsgCheck(msg *wire.MessageMsgCheck) error {
	Logger.log.Debugf("handleMsgCheck %s", msg.HashStr)
	msgResp, err := wire.MakeEmptyMessage(wire.CmdMsgCheckResp)
	if err != nil {
		Logger.log.Error("handleMsgCheck error", err)
		return NewPeerError(HandleMessageCheck, err, nil)
	}
	if p.listenerPeer.CheckHashPool(msg.HashStr) {
		msgResp.(*wire.MessageMsgCheckResp).HashStr = msg.HashStr
		msgResp.(*wire.MessageMsgCheckResp).Accept = false
	} else {
		msgResp.(*wire.MessageMsgCheckResp).HashStr = msg.HashStr
		msgResp.(*wire.MessageMsgCheckResp).Accept = true
	}
	p.QueueMessageWithEncoding(msgResp, nil, MessageToPeer, nil)
	return nil
}

// handleMsgCheckResp - check cMsgHash contain hash of message
func (p *PeerConn) handleMsgCheckResp(msg *wire.MessageMsgCheckResp) error {
	Logger.log.Debugf("handleMsgCheckResp %s", msg.HashStr)
	m, ok := p.cMsgHash[msg.HashStr]
	if ok {
		if !p.isUnitTest {
			// if not unit test -> send channel to process
			m <- msg.Accept
		}
		return nil
	} else {
		return NewPeerError(HandleMessageCheckResponse, errors.New(fmt.Sprintf("p.cMsgHash not contain %s", msg.HashStr)), nil)
	}
}

// Close - close peer connection by close channel
func (p *PeerConn) close() {
	if _, ok := <-p.cClose; ok {
		close(p.cClose)
	}
}

// ForceClose - set flag and close channel
func (p *PeerConn) ForceClose() {
	p.setIsForceClose(true)
	select {
	case <-p.cClose:
		return
	default:
		close(p.cClose)
	}
}
