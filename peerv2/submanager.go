package peerv2

import (
	"context"
	"fmt"
	"sort"

	"github.com/incognitochain/incognito-chain/common/consensus"
	"github.com/incognitochain/incognito-chain/incognitokey"

	pubsub "github.com/incognitochain/go-libp2p-pubsub"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/peerv2/proto"
	"github.com/incognitochain/incognito-chain/wire"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
)

type ConsensusData interface {
	GetOneValidator() *consensus.Validator
	GetOneValidatorForEachConsensusProcess() map[int]*consensus.Validator
}

// SubManager manages pubsub subscription of highway's topics
type SubManager struct {
	info
	messages chan *pubsub.Message // channel to put subscribed messages to

	registerer Registerer
	subscriber Subscriber

	topics msgToTopics
	subs   msgToTopics // mapping from message to topic's subscription

	rolehash string
}

type info struct {
	consensusData ConsensusData
	pubkey        string
	syncMode      string
	relayShard    []byte
	peerID        peer.ID
}

type Subscriber interface {
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
}

type Registerer interface {
	Register(context.Context, string, []string, []byte, peer.ID, string, int32) ([]*proto.MessageTopicPair, *proto.UserRole, error)
	Target() string
	UpdateTarget(peer.ID)
}

func NewSubManager(
	info info,
	subscriber Subscriber,
	registerer Registerer,
	messages chan *pubsub.Message,
) *SubManager {
	return &SubManager{
		info:       info,
		subscriber: subscriber,
		registerer: registerer,
		messages:   messages,
		//role:       newUserRole("dummyLayer", "dummyRole", -1000),
		topics: msgToTopics{},
		subs:   msgToTopics{},
	}
}

func (sub *SubManager) GetMsgToTopics() msgToTopics {
	return sub.subs // no need to make a copy since topics rarely changed (when role changed)
}

func (sub *SubManager) SetSyncMode(s string) {
	sub.syncMode = s
}

// Subscribe registers to proxy and save the list of new topics if needed
func (sub *SubManager) Subscribe(forced bool) error {
	rolehash := ""
	relayShardIDs := sub.relayShard
	var newTopics = make(msgToTopics)
	var err error
	shardIDs := []int{}
	nodePK, _ := new(incognitokey.CommitteePublicKey).ToBase58()

	if sub.syncMode == "" {
		newRole := sub.consensusData.GetOneValidatorForEachConsensusProcess()

		for _, sid := range relayShardIDs {
			if newRole[int(sid)] == nil {
				newRole[int(sid)] = &consensus.Validator{}
			}
		}

		//recalculate rolehash
		for k := range newRole {
			shardIDs = append(shardIDs, int(k))
		}
		sort.Ints(shardIDs)
		str := ""
		for _, chainID := range shardIDs {
			if newRole[chainID] != nil {
				str += fmt.Sprintf("%v-%v-%v", chainID, newRole[chainID].State.Role, newRole[chainID].State.Index)
			} else {
				str += fmt.Sprintf("%v-", chainID)
			}
		}
		rolehash = common.HashH([]byte(str)).String()

		//check if role hash is changed
		if rolehash == sub.rolehash && !forced { // Not forced => no need to subscribe when role stays the same
			return nil
		}

		Logger.Infof("Role changed %+v", rolehash)

		// Registering relay
		if len(relayShardIDs) == 0 {
			relayShardIDs = []byte{255}
		}

		validator := sub.consensusData.GetOneValidator()
		if validator != nil {
			nodePK = validator.MiningKey.GetPublicKeyBase58()
		} else {
			validator = &consensus.Validator{
				State: consensus.MiningState{
					Index: 0,
				},
			}
		}
		newTopics, _, err = sub.registerToProxy(
			nodePK,
			"",
			"",
			relayShardIDs,
			validator.State.Index,
		)
		if err != nil {
			return err
		}

		// Registering mining
		for chainID, validator := range newRole {
			if validator.PrivateSeed != "" {
				topics, _, err := sub.registerToProxy(
					validator.MiningKey.GetPublicKeyBase58(),
					validator.State.Layer,
					validator.State.Role,
					[]byte{byte(chainID)},
					validator.State.Index,
				)
				if err != nil {
					return err // Don't save new role and topics since we need to retry later
				}
				for msg, topic := range topics {
					newTopics[msg] = append(newTopics[msg], topic...)
				}
			}
		}
	} else if sub.syncMode == "netmonitor" {
		rolehash = common.HashH([]byte("netmonitor")).String()

		//check if role hash is changed
		if rolehash == sub.rolehash && !forced { // Not forced => no need to subscribe when role stays the same
			return nil
		}

		topics, _, err := sub.registerToProxy(
			nodePK,
			"",
			"netmonitor",
			[]byte{255},
			0,
		)
		if err != nil {
			return err // Don't save new role and topics since we need to retry later
		}
		for msg, topic := range topics {
			newTopics[msg] = append(newTopics[msg], topic...)
		}

	}

	Logger.Infof("newTopics %v", newTopics)
	Logger.Infof("sub topics %v", sub.topics)

	// Subscribing
	if err := sub.subscribeNewTopics(newTopics, sub.topics, forced); err != nil {
		Logger.Error(err)
		return err
	}

	sub.topics = newTopics
	sub.rolehash = rolehash
	return nil
}

type userRole struct {
	layer   string
	role    string
	shardID int
}

func newUserRole(layer, role string, shardID int) userRole {
	return userRole{
		layer:   layer,
		role:    role,
		shardID: shardID,
	}
}

// subscribeNewTopics subscribes to new topics and unsubcribes any topics that aren't needed anymore
func (sub *SubManager) subscribeNewTopics(newTopics, subscribed msgToTopics, forced bool) error {
	found := func(tName string, tmap msgToTopics) bool {
		for _, topicList := range tmap {
			for _, t := range topicList {
				if tName == t.Name {
					return true
				}
			}
		}
		return false
	}

	// Unsubscribe to old ones
	for m, topicList := range subscribed {
		for _, t := range topicList {
			if !forced && found(t.Name, newTopics) { // Unsub and sub again if forced subscribe
				continue
			}

			idx := -1
			for i, s := range sub.subs[m] {
				if s.Name == t.Name {
					if t.Act != proto.MessageTopicPair_PUB {
						Logger.Info("unsubscribing", m, t.Name)
						if s.Sub != nil {
							go s.Sub.Cancel()
						}
					}
					idx = i
					break
				}
			}

			if idx < 0 {
				continue
			}

			if len(sub.subs[m]) == 1 {
				delete(sub.subs, m)
			} else {
				sub.subs[m] = append(sub.subs[m][:idx], sub.subs[m][idx+1:]...)
			}
		}
	}

	// Subscribe to new topics
	for m, topicList := range newTopics {
		Logger.Infof("Process message %v and topic %v", m, topicList)
		for _, t := range topicList {
			if !forced && found(t.Name, subscribed) {
				Logger.Infof("Skip subscribed topic: %v %v", t.Name, subscribed)
				continue
			}

			if t.Act == proto.MessageTopicPair_PUB {
				sub.subs[m] = append(sub.subs[m], Topic{Name: t.Name, Sub: nil, Act: t.Act})
				Logger.Infof("Skip subscribing to PUB topic: %v", t.Name)
				continue
			}

			Logger.Infof("subscribing message: %v, topic: %v", m, t.Name)

			s, err := sub.subscriber.Subscribe(t.Name)
			if err != nil {
				return errors.WithStack(err)
			}
			sub.subs[m] = append(sub.subs[m], Topic{Name: t.Name, Sub: s, Act: t.Act})
			go processSubscriptionMessage(sub.messages, s)
		}
	}
	return nil
}

// processSubscriptionMessage listens to a topic and pushes all messages to a queue to be processed later
func processSubscriptionMessage(inbox chan *pubsub.Message, sub *pubsub.Subscription) {
	ctx := context.Background()
	for {
		// TODO(@0xbunyip): check if topic is unsubbed then return, otherwise just continue
		msg, err := sub.Next(ctx)
		if err != nil { // Subscription might have been cancelled
			Logger.Warn(err)
			return
		}

		inbox <- msg
	}
}

type Topic struct {
	Name string
	Sub  *pubsub.Subscription
	Act  proto.MessageTopicPair_Action
}

type msgToTopics map[string][]Topic // Message to topics

func (sub *SubManager) registerToProxy(
	pubkey string,
	layer string,
	role string,
	shardID []byte,
	pkIndex int32,
) (msgToTopics, userRole, error) {
	messagesWanted := getMessagesForLayer(layer, shardID)
	Logger.Infof("Registering: message: %v", messagesWanted)
	// Logger.Infof("Registering: nodeMode: %v", sub.nodeMode)
	Logger.Infof("Registering: layer: %v", layer)
	Logger.Infof("Registering: role: %v", role)
	Logger.Infof("Registering: shardID: %v", shardID)
	Logger.Infof("Registering: peerID: %v", sub.peerID.String())
	Logger.Infof("Registering: pubkey: %v", pubkey)

	pairs, topicRole, err := sub.registerer.Register(
		context.Background(),
		pubkey,
		messagesWanted,
		shardID,
		sub.peerID,
		role,
		pkIndex,
	)
	if err != nil {
		return nil, userRole{}, err
	}

	// Mapping from message to list of topics
	topics := msgToTopics{}
	for _, p := range pairs {
		for i, t := range p.Topic {
			topics[p.Message] = append(topics[p.Message], Topic{
				Name: t,
				Act:  p.Act[i],
			})
		}
	}
	r := userRole{
		layer:   topicRole.Layer,
		role:    topicRole.Role,
		shardID: int(topicRole.Shard),
	}
	return topics, r, nil
}

func getMessagesForLayer(layer string, shardID []byte) []string {
	if layer == common.ShardRole {
		return []string{
			wire.CmdBlockShard,
			wire.CmdBlockBeacon,
			wire.CmdBFT,
			wire.CmdPeerState,
			wire.CmdCrossShard,
			wire.CmdTx,
			wire.CmdPrivacyCustomToken,
		}
	} else if layer == common.BeaconRole {
		return []string{
			wire.CmdBlockBeacon,
			wire.CmdBFT,
			wire.CmdPeerState,
			wire.CmdBlockShard,
		}
	} else {
		containShard := false
		for _, s := range shardID {
			if s != HighwayBeaconID {
				containShard = true
			}
		}
		msgs := []string{
			wire.CmdBlockBeacon,
			wire.CmdPeerState,
			wire.CmdTx,
			wire.CmdPrivacyCustomToken,
		}
		if containShard {
			msgs = append(msgs, wire.CmdBlockShard)
		}
		return msgs
	}

	return []string{}
}
