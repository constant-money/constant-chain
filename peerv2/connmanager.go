package peerv2

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"reflect"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/peerv2/proto"
	"github.com/incognitochain/incognito-chain/peerv2/rpcclient"
	"github.com/incognitochain/incognito-chain/peerv2/wrapper"
	"github.com/incognitochain/incognito-chain/wire"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var HighwayBeaconID = byte(255)

func NewConnManager(
	host *Host,
	dpa string,
	ikey *incognitokey.CommitteePublicKey,
	cd ConsensusData,
	dispatcher *Dispatcher,
	nodeMode string,
	relayShard []byte,
) *ConnManager {
	pubkey, _ := ikey.ToBase58()
	return &ConnManager{
		info: info{
			consensusData: cd,
			pubkey:        pubkey,
			relayShard:    relayShard,
			nodeMode:      nodeMode,
			peerID:        host.Host.ID(),
		},
		keeper:               NewAddrKeeper(),
		LocalHost:            host,
		DiscoverPeersAddress: dpa,
		discoverer:           new(rpcclient.RPCClient),
		disp:                 dispatcher,
		IsMasterNode:         false,
		registerRequests:     make(chan peer.ID, 100),
		stop:                 make(chan int),
	}
}

func (cm *ConnManager) PublishMessage(msg wire.Message) error {
	var topic string
	publishable := []string{wire.CmdBlockShard, wire.CmdBFT, wire.CmdBlockBeacon, wire.CmdTx, wire.CmdPrivacyCustomToken, wire.CmdPeerState, wire.CmdCrossShard}

	// msgCrossShard := msg.(wire.MessageCrossShard)
	msgType := msg.MessageType()
	subs := cm.subscriber.GetMsgToTopics()
	for _, p := range publishable {
		topic = ""
		if msgType == p {
			for _, availableTopic := range subs[msgType] {
				// Logger.Info("[hy]", availableTopic)
				if (availableTopic.Act == proto.MessageTopicPair_PUB) || (availableTopic.Act == proto.MessageTopicPair_PUBSUB) {
					topic = availableTopic.Name
					err := broadcastMessage(msg, topic, cm.ps)
					if err != nil {
						Logger.Errorf("Broadcast to topic %v error %v", topic, err)
						return err
					}
				}

			}
			if topic == "" {
				return errors.New("Can not find topic of this message type " + msgType + "for publish")
			}

			// return broadcastMessage(msg, topic, cm.ps)
		}
	}

	return nil
}

func (cm *ConnManager) PublishMessageToShard(msg wire.Message, shardID byte) error {
	publishable := []string{wire.CmdPeerState, wire.CmdBlockShard, wire.CmdCrossShard, wire.CmdBFT}
	msgType := msg.MessageType()
	subs := cm.subscriber.GetMsgToTopics()
	for _, p := range publishable {
		if msgType == p {
			// Get topic for mess
			for _, availableTopic := range subs[msgType] {
				Logger.Info(availableTopic)
				cID := GetCommitteeIDOfTopic(availableTopic.Name)
				if (byte(cID) == shardID) && ((availableTopic.Act == proto.MessageTopicPair_PUB) || (availableTopic.Act == proto.MessageTopicPair_PUBSUB)) {
					return broadcastMessage(msg, availableTopic.Name, cm.ps)
				}
			}
		}
	}

	Logger.Warn("Cannot publish message", msgType)
	return nil
}

func (cm *ConnManager) Start(ns NetSync) {
	// Pubsub
	var err error
	cm.ps, err = pubsub.NewFloodSub(context.Background(), cm.LocalHost.Host, pubsub.WithMaxMessageSize(common.MaxPSMsgSize))
	if err != nil {
		panic(err)
	}
	cm.messages = make(chan *pubsub.Message, 1000)

	// NOTE: must Connect after creating FloodSub
	go cm.keepHighwayConnection()

	cm.Requester = NewRequester(cm.LocalHost.GRPC)
	cm.subscriber = NewSubManager(cm.info, cm.ps, cm.Requester, cm.messages)
	cm.Provider = NewBlockProvider(cm.LocalHost.GRPC, ns)
	go cm.manageRoleSubscription()
	cm.process()
}

// BroadcastCommittee floods message to topic `chain_committee` for highways
// Only masternode actually does the broadcast, other's messages will be ignored by highway
func (cm *ConnManager) BroadcastCommittee(
	epoch uint64,
	newBeaconCommittee []incognitokey.CommitteePublicKey,
	newAllShardCommittee map[byte][]incognitokey.CommitteePublicKey,
	newAllShardPending map[byte][]incognitokey.CommitteePublicKey,
) {
	// NOTE: disabled feature, always return for now
	if !cm.IsMasterNode {
		return
	}

	Logger.Info("Broadcasting committee to highways!!!")
	cc := &incognitokey.ChainCommittee{
		Epoch:             epoch,
		BeaconCommittee:   newBeaconCommittee,
		AllShardCommittee: newAllShardCommittee,
		AllShardPending:   newAllShardPending,
	}
	data, err := cc.ToByte()
	if err != nil {
		Logger.Error(err)
		return
	}

	topic := "chain_committee"
	err = cm.ps.Publish(topic, data)
	if err != nil {
		Logger.Error(err)
	}
}

type ForcedSubscriber interface {
	Subscribe(forced bool) error
	GetMsgToTopics() msgToTopics
}

type ConnManager struct {
	info         // info of running node
	LocalHost    *Host
	subscriber   ForcedSubscriber
	disconnected int
	registered   bool

	DiscoverPeersAddress string
	IsMasterNode         bool

	ps               *pubsub.PubSub
	messages         chan *pubsub.Message // queue messages from all topics
	registerRequests chan peer.ID

	keeper     *AddrKeeper
	discoverer HighwayDiscoverer
	disp       *Dispatcher
	Requester  *BlockRequester
	Provider   *BlockProvider

	stop chan int
}

func (cm *ConnManager) PutMessage(msg *pubsub.Message) {
	cm.messages <- msg
}

func (cm *ConnManager) process() {
	for {
		select {
		case msg := <-cm.messages:
			err := cm.disp.processInMessageString(string(msg.Data))
			if err != nil {
				Logger.Warn(err)
			}
		}
	}
}

// keepHighwayConnection periodically checks liveliness of connection to highway
// and try to connect if it's not available.
func (cm *ConnManager) keepHighwayConnection() {
	// Init list of highways
	cm.keeper.Add(
		rpcclient.HighwayAddr{
			Libp2pAddr: "",
			RPCUrl:     cm.DiscoverPeersAddress,
		},
	)
	var currentHighway *rpcclient.HighwayAddr

	watchTimestep := time.NewTicker(ReconnectHighwayTimestep)
	refreshTimestep := time.NewTicker(UpdateHighwayListTimestep)
	defer watchTimestep.Stop()
	defer refreshTimestep.Stop()
	cm.disconnected = 1 // Init, to make first connection to highway
	pid := cm.LocalHost.Host.ID()

	refreshHighway := func() (*rpcclient.HighwayAddr, error) {
		newHighway, err := cm.keeper.ChooseHighway(cm.discoverer, pid)
		if err != nil {
			Logger.Errorf("Failed refreshing highway: %v", err)
			return currentHighway, err
		}
		return &newHighway, nil
	}

	for {
		select {
		case <-watchTimestep.C:
			if currentHighway == nil {
				var err error
				if currentHighway, err = refreshHighway(); err != nil || currentHighway == nil {
					continue
				}
			}

			addrInfo, err := getAddressInfo(currentHighway.Libp2pAddr)
			if err != nil || cm.checkConnection(addrInfo) {
				cm.keeper.IgnoreAddress(*currentHighway) // Not reconnect to this address for some time
				currentHighway = nil                     // Failed retries, connect to new highway next iteration
			}

		case <-refreshTimestep.C:
			currentHighway, _ = refreshHighway()

		case <-cm.stop:
			Logger.Info("Stop keeping connection to highway")
			break
		}
	}
}

func (cm *ConnManager) checkConnection(addrInfo *peer.AddrInfo) bool {
	net := cm.LocalHost.Host.Network()
	// Reconnect if not connected
	if net.Connectedness(addrInfo.ID) != network.Connected {
		cm.disconnected++
		cm.registered = false // Next time we connect to highway, we need to register again
		Logger.Info("Not connected to highway, connecting")
		ctx, cancel := context.WithTimeout(context.Background(), DialTimeout)
		defer cancel()
		if err := cm.LocalHost.Host.Connect(ctx, *addrInfo); err != nil {
			Logger.Errorf("Could not connect to highway: %v %v", err, addrInfo)
		}
		if cm.disconnected > MaxConnectionRetry {
			Logger.Error("Retry maxed out")
			cm.disconnected = 0 // Retry N times for next chosen highway
			return true
		}
	}

	if !cm.registered && net.Connectedness(addrInfo.ID) == network.Connected {
		// Register again since this might be a new highway
		Logger.Info("Connected to highway, sending register request")
		cm.registerRequests <- addrInfo.ID
		cm.disconnected = 0
		cm.registered = true
	}
	return false
}

// manageRoleSubscription: polling current role periodically and subscribe to relevant topics
func (cm *ConnManager) manageRoleSubscription() {
	forced := false // only subscribe when role changed or last forced subscribe failed
	hwID := peer.ID("")
	var err error
	registerTimestep := time.NewTicker(RegisterTimestep)
	defer registerTimestep.Stop()
	for {
		select {
		case <-registerTimestep.C:
			// Check if we are connecting to the target of registration (correct highway peerID)
			target := cm.Requester.Target()
			if hwID.Pretty() != target {
				cm.Requester.UpdateTarget(hwID)
				Logger.Errorf("Waiting to establish connection to highway: new highway = %v, current = %v", hwID.Pretty(), target)
				continue
			}

			err = cm.subscriber.Subscribe(forced)
			if err != nil {
				Logger.Errorf("Subscribe failed: forced = %v hwID = %s err = %+v", forced, hwID.String(), err)
			} else {
				forced = false
			}

		case newID := <-cm.registerRequests:
			Logger.Info("Received request to register")
			forced = true // register no matter if role changed or not

			// Close libp2p connection to old highway if we are connecting to new one
			if hwID != peer.ID("") && newID != hwID {
				if err := cm.LocalHost.Host.Network().ClosePeer(hwID); err != nil {
					Logger.Errorf("Failed closing connection to old highway: hwID = %s err = %v", hwID.String(), err)
				}
			}
			hwID = newID

		case <-cm.stop:
			Logger.Info("Stop managing role subscription")
			break
		}
	}
}

func encodeMessage(msg wire.Message) (string, error) {
	// NOTE: copy from peerConn.outMessageHandler
	// Create messageHex
	messageBytes, err := msg.JsonSerialize()
	if err != nil {
		Logger.Error("Can not serialize json format for messageHex:"+msg.MessageType(), err)
		return "", err
	}

	// Add 24 bytes headerBytes into messageHex
	headerBytes := make([]byte, wire.MessageHeaderSize)
	// add command type of message
	cmdType, messageErr := wire.GetCmdType(reflect.TypeOf(msg))
	if messageErr != nil {
		Logger.Error("Can not get cmd type for "+msg.MessageType(), messageErr)
		return "", err
	}
	copy(headerBytes[:], []byte(cmdType))
	// add forward type of message at 13st byte
	forwardType := byte('s')
	forwardValue := byte(0)
	copy(headerBytes[wire.MessageCmdTypeSize:], []byte{forwardType})
	copy(headerBytes[wire.MessageCmdTypeSize+1:], []byte{forwardValue})
	messageBytes = append(messageBytes, headerBytes...)
	// Logger.Infof("Encoded message TYPE %s CONTENT %s", cmdType, string(messageBytes))

	// zip data before send
	messageBytes, err = common.GZipFromBytes(messageBytes)
	if err != nil {
		Logger.Error("Can not gzip for messageHex:"+msg.MessageType(), err)
		return "", err
	}
	messageHex := hex.EncodeToString(messageBytes)
	//log.Debugf("Content in hex encode: %s", string(messageHex))
	// add end character to messageHex (delim '\n')
	// messageHex += "\n"
	return messageHex, nil
}

func broadcastMessage(msg wire.Message, topic string, ps *pubsub.PubSub) error {
	// Encode message to string first
	messageHex, err := encodeMessage(msg)
	if err != nil {
		return err
	}

	// Broadcast
	Logger.Infof("Publishing to topic %s", topic)
	return ps.Publish(topic, []byte(messageHex))
}

type HighwayDiscoverer interface {
	DiscoverHighway(discoverPeerAddress string, shardsStr []string) (map[string][]rpcclient.HighwayAddr, error)
}

func (conn *ConnManager) RequestBeaconBlocksViaStream(ctx context.Context, peerID string, from uint64, to uint64) (blockCh chan types.BlockInterface, err error) {
	Logger.Infof("[SyncBeacon] from %v to %v ", from, to)
	req := &proto.BlockByHeightRequest{
		Type:         proto.BlkType_BlkBc,
		Specific:     false,
		Heights:      []uint64{from, to},
		From:         int32(HighwayBeaconID),
		To:           int32(HighwayBeaconID),
		SyncFromPeer: peerID,
	}
	return conn.requestBlocksViaStream(ctx, peerID, req)
}

func (conn *ConnManager) RequestShardBlocksViaStream(ctx context.Context, peerID string, fromSID int, from uint64, to uint64) (blockCh chan types.BlockInterface, err error) {
	Logger.Infof("[SyncShard] from %v to %v fromShard %v", from, to, fromSID)
	req := &proto.BlockByHeightRequest{
		Type:         proto.BlkType_BlkShard,
		Specific:     false,
		Heights:      []uint64{from, to},
		From:         int32(fromSID),
		To:           int32(fromSID),
		SyncFromPeer: peerID,
	}
	return conn.requestBlocksViaStream(ctx, peerID, req)
}

func (conn *ConnManager) RequestCrossShardBlocksViaStream(ctx context.Context, peerID string, fromSID int, toSID int, heights []uint64) (blockCh chan types.BlockInterface, err error) {
	Logger.Infof("[SyncXShard] heights %v fromShard %v toShard %v", heights, fromSID, toSID)
	req := &proto.BlockByHeightRequest{
		Type:         proto.BlkType_BlkXShard,
		Specific:     true,
		Heights:      heights,
		From:         int32(fromSID),
		To:           int32(toSID),
		SyncFromPeer: peerID,
	}
	return conn.requestBlocksViaStream(ctx, peerID, req)
}

func (conn *ConnManager) RequestCrossShardBlocksByHashViaStream(ctx context.Context, peerID string, fromSID int, toSID int, hashes [][]byte) (blockCh chan types.BlockInterface, err error) {
	req := &proto.BlockByHashRequest{
		Type:         proto.BlkType_BlkXShard,
		Hashes:       hashes,
		From:         int32(fromSID),
		To:           int32(toSID),
		SyncFromPeer: peerID,
	}
	return conn.requestBlocksByHashViaStream(ctx, peerID, req)
}

func (conn *ConnManager) RequestBeaconBlocksByHashViaStream(ctx context.Context, peerID string, hashes [][]byte) (blockCh chan types.BlockInterface, err error) {
	req := &proto.BlockByHashRequest{
		Type:         proto.BlkType_BlkBc,
		Hashes:       hashes,
		From:         int32(HighwayBeaconID),
		To:           int32(HighwayBeaconID),
		SyncFromPeer: peerID,
	}
	return conn.requestBlocksByHashViaStream(ctx, peerID, req)
}

func (conn *ConnManager) RequestShardBlocksByHashViaStream(ctx context.Context, peerID string, fromSID int, hashes [][]byte) (blockCh chan types.BlockInterface, err error) {
	req := &proto.BlockByHashRequest{
		Type:         proto.BlkType_BlkShard,
		Hashes:       hashes,
		From:         int32(fromSID),
		To:           int32(fromSID),
		SyncFromPeer: peerID,
	}
	return conn.requestBlocksByHashViaStream(ctx, peerID, req)
}

func (conn *ConnManager) requestBlocksViaStream(ctx context.Context, peerID string, req *proto.BlockByHeightRequest) (blockCh chan types.BlockInterface, err error) {
	Logger.Infof("[stream] Request Block type %v from peer %v from cID %v, [%v %v] ", req.Type, peerID, req.GetFrom(), req.Heights[0], req.Heights[len(req.Heights)-1])
	blockCh = make(chan types.BlockInterface, blockchain.DefaultMaxBlkReqPerPeer)
	stream, err := conn.Requester.StreamBlockByHeight(ctx, req)
	if err != nil {
		Logger.Errorf("[stream] %v", err)
		return nil, err
	}

	var closeChannel = func() {
		if blockCh != nil {
			close(blockCh)
			blockCh = nil
		}
	}

	go func(stream proto.HighwayService_StreamBlockByHeightClient, ctx context.Context) {
		for {
			blkData, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					Logger.Errorf("[stream] %v", err)
				}
				closeChannel()
				return
			}

			if len(blkData.Data) < 2 {
				Logger.Errorf("[stream] received empty blk")
				closeChannel()
				return
			}

			var newBlk types.BlockInterface = types.NewBeaconBlock()
			if req.Type == proto.BlkType_BlkShard {
				newBlk = types.NewShardBlock()
			} else if req.Type == proto.BlkType_BlkXShard {
				newBlk = types.NewCrossShardBlock()
			}

			err = wrapper.DeCom(blkData.Data[1:], newBlk)
			if err != nil {
				Logger.Errorf("[stream] %v", err)
				closeChannel()
				return
			}
			//fmt.Printf("[stream]: Receive %v block %v \n", req.GetType(), newBlk.GetHeight())
			select {
			case <-ctx.Done():
				closeChannel()
				return
			case blockCh <- newBlk:
			}
		}

	}(stream, ctx)

	return blockCh, nil
}

func (conn *ConnManager) requestBlocksByHashViaStream(ctx context.Context, peerID string, req *proto.BlockByHashRequest) (blockCh chan types.BlockInterface, err error) {
	Logger.Infof("SYNCKER Request Block by hash from peerID %v, from CID %v, total %v blocks", peerID, req.From, len(req.Hashes))
	blockCh = make(chan types.BlockInterface, blockchain.DefaultMaxBlkReqPerPeer)
	stream, err := conn.Requester.StreamBlockByHash(ctx, req)
	if err != nil {
		return nil, err
	}

	var closeChannel = func() {
		if blockCh != nil {
			close(blockCh)
			blockCh = nil
		}
	}

	go func(stream proto.HighwayService_StreamBlockByHashClient, ctx context.Context) {
		for {
			blkData, err := stream.Recv()
			if err != nil || err == io.EOF {
				closeChannel()
				return
			}

			if len(blkData.Data) < 2 {
				closeChannel()
				return
			}

			var newBlk types.BlockInterface = types.NewBeaconBlock()
			if req.Type == proto.BlkType_BlkShard {
				newBlk = types.NewShardBlock()
			} else if req.Type == proto.BlkType_BlkXShard {
				newBlk = types.NewCrossShardBlock()
			}

			err = wrapper.DeCom(blkData.Data[1:], newBlk)
			if err != nil {
				closeChannel()
				return
			}
			//fmt.Println("SYNCKER: Receive block ...", newBlk.GetHeight())
			select {
			case <-ctx.Done():
				closeChannel()
				return
			case blockCh <- newBlk:
			}
		}

	}(stream, ctx)

	return blockCh, nil
}
