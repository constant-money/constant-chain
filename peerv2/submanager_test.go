package peerv2

import (
	"fmt"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/peerv2/mocks"
	"github.com/incognitochain/incognito-chain/peerv2/proto"
	"github.com/incognitochain/incognito-chain/wire"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetShardIDsNormalAutoMode(t *testing.T) {
	role := userRole{shardID: -2}
	nodeMode := common.NodeModeAuto
	relayShard := []byte{}
	shardIDs := getWantedShardIDs(role, nodeMode, relayShard)

	// Must return shardID 255 to be able to get beacon's topics
	assert.Equal(t, []byte{255}, shardIDs, "incorrect shardIDs")
}

func TestGetShardIDsValidatorAutoMode(t *testing.T) {
	role := userRole{shardID: 3}
	nodeMode := common.NodeModeAuto
	relayShard := []byte{}
	shardIDs := getWantedShardIDs(role, nodeMode, relayShard)

	// shardID = 3 is enough when we want msg blockbeacon
	assert.Equal(t, []byte{3}, shardIDs, "incorrect shardIDs")
}

func TestGetShardIDsRelayBeacon(t *testing.T) {
	role := userRole{shardID: -2}
	nodeMode := common.NodeModeRelay
	relayShard := []byte{}
	shardIDs := getWantedShardIDs(role, nodeMode, relayShard)

	assert.Equal(t, []byte{255}, shardIDs, "incorrect shardIDs")
}

func TestGetShardIDsRelayShards(t *testing.T) {
	role := userRole{shardID: -2}
	nodeMode := common.NodeModeRelay
	relayShard := []byte{1, 2, 5, 7}
	shardIDs := getWantedShardIDs(role, nodeMode, relayShard)

	assert.Equal(t, []byte{1, 2, 5, 7, 255}, shardIDs, "incorrect shardIDs")
}

func TestSubscribeNoChange(t *testing.T) {
	role := userRole{
		layer:   "",
		role:    "",
		shardID: -2,
	}
	consensusData := &mocks.ConsensusData{}
	consensusData.On("GetUserRole").Return(role.layer, role.role, role.shardID)
	sub := &SubManager{
		info: info{consensusData: consensusData},
		role: role,
	}
	forced := false
	err := sub.Subscribe(forced)
	assert.Nil(t, err)
}

func TestSubscribeRoleChanged(t *testing.T) {
	role := userRole{
		layer:   "",
		role:    "",
		shardID: -2,
	}
	registerer := &mocks.Registerer{}
	var pairs []*proto.MessageTopicPair
	err := fmt.Errorf("error preventing further advance")
	registerer.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pairs, &proto.UserRole{}, err)
	consensusData := &mocks.ConsensusData{}
	consensusData.On("GetUserRole").Once().Return(common.ShardRole, common.PendingRole, 1)
	sub := &SubManager{
		info: info{
			consensusData: consensusData,
			nodeMode:      common.NodeModeAuto,
			relayShard:    []byte{},
		},
		role:       role,
		registerer: registerer,
	}
	forced := false
	sub.Subscribe(forced)
	consensusData.AssertNumberOfCalls(t, "GetUserRole", 1)
}

func TestSubscribeForced(t *testing.T) {
	role := userRole{
		layer:   "",
		role:    "",
		shardID: -2,
	}
	registerer := &mocks.Registerer{}
	var pairs []*proto.MessageTopicPair
	err := fmt.Errorf("error preventing further advance")
	registerer.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pairs, &proto.UserRole{}, err)
	consensusData := &mocks.ConsensusData{}
	consensusData.On("GetUserRole").Return(role.layer, role.role, role.shardID)
	sub := &SubManager{
		info: info{
			consensusData: consensusData,
			nodeMode:      common.NodeModeAuto,
			relayShard:    []byte{},
		},
		role:       role,
		registerer: registerer,
	}
	forced := true
	sub.Subscribe(forced)
	consensusData.AssertNumberOfCalls(t, "GetUserRole", 1)
}

func TestGetMessage(t *testing.T) {
	testCases := []struct {
		desc    string
		mode    string
		layer   string
		shardID []byte
		out     []string
	}{
		{
			desc:  "Nodemode auto, normal role",
			mode:  common.NodeModeAuto,
			layer: "",
			out:   []string{wire.CmdBlockBeacon, wire.CmdTx, wire.CmdPrivacyCustomToken, wire.CmdPeerState},
		},
		{
			desc:    "Nodemode relay beacon",
			mode:    common.NodeModeRelay,
			layer:   "",
			shardID: []byte{255},
			out:     []string{wire.CmdBlockBeacon, wire.CmdTx, wire.CmdPrivacyCustomToken, wire.CmdPeerState},
		},
		{
			desc:    "Nodemode relay shards",
			mode:    common.NodeModeRelay,
			layer:   "",
			shardID: []byte{1, 2, 3},
			out:     []string{wire.CmdBlockBeacon, wire.CmdBlockShard, wire.CmdTx, wire.CmdPrivacyCustomToken, wire.CmdPeerState},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			msgs := getMessagesForLayer(tc.mode, tc.layer, tc.shardID)
			compareMsgs(t, tc.out, msgs)
		})
	}
}

func compareMsgs(t *testing.T, exp, msgs []string) {
	assert.Equal(t, len(exp), len(msgs))
	for _, e := range exp {
		ok := false
		for _, m := range msgs {
			if e == m {
				ok = true
				break
			}
		}
		assert.True(t, ok, "msg %s not found", e)
	}
}

func TestSubscribeNewTopics(t *testing.T) {
	subscription := &pubsub.Subscription{}
	testCases := []struct {
		desc       string
		newTopics  msgToTopics
		subscribed msgToTopics
		forced     bool
		subCalled  int
		subsLen    int
	}{
		{
			desc: "Already subscribed all topics",
			newTopics: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc"}},
				wire.CmdTx:          []Topic{Topic{Name: "xyz"}, Topic{Name: "ijk"}},
			},
			subscribed: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc"}},
				wire.CmdTx:          []Topic{Topic{Name: "xyz"}, Topic{Name: "ijk"}},
			},
			subCalled: 0,
			subsLen:   2,
		},
		{
			desc: "Subscribe to new topics",
			newTopics: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc"}},
				wire.CmdTx:          []Topic{Topic{Name: "xyz"}, Topic{Name: "ijk"}},
			},
			subscribed: msgToTopics{},
			subCalled:  3,
			subsLen:    2,
		},
		{
			desc: "Subscribe ignore PUB topics",
			newTopics: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc"}},
				wire.CmdTx:          []Topic{Topic{Name: "xyz", Act: proto.MessageTopicPair_PUB}, Topic{Name: "ijk"}},
			},
			subscribed: msgToTopics{},
			subCalled:  2,
			subsLen:    2,
		},
		{
			desc: "Unsubscribe old topics",
			newTopics: msgToTopics{
				wire.CmdBlockBeacon: []Topic{},
				wire.CmdTx:          []Topic{Topic{Name: "xyz"}},
			},
			subscribed: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc", Sub: subscription}},
				wire.CmdTx:          []Topic{Topic{Name: "ijk", Sub: subscription}},
			},
			subCalled: 1,
			subsLen:   1,
		},
		{
			desc: "Unsubscribe ignore PUB topics",
			newTopics: msgToTopics{
				wire.CmdBlockBeacon: []Topic{},
				wire.CmdTx:          []Topic{},
			},
			subscribed: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc", Act: proto.MessageTopicPair_PUB, Sub: nil}}, // Sub == nil => panic if trying to unsub
				wire.CmdTx:          []Topic{Topic{Name: "ijk", Act: proto.MessageTopicPair_PUB, Sub: nil}},
			},
			subCalled: 0,
			subsLen:   0,
		},
		{
			desc: "Forced subscribe old topics",
			newTopics: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc"}},
				wire.CmdTx:          []Topic{Topic{Name: "xyz"}, Topic{Name: "ijk"}},
			},
			subscribed: msgToTopics{
				wire.CmdBlockBeacon: []Topic{Topic{Name: "abc"}},
				wire.CmdTx:          []Topic{Topic{Name: "xyz"}, Topic{Name: "ijk"}},
			},
			forced:    true,
			subCalled: 3,
			subsLen:   2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			subscription := &pubsub.Subscription{}
			subscriber := &mocks.Subscriber{}
			var err error
			subscriber.On("Subscribe", mock.Anything).Return(subscription, err)
			sub := &SubManager{subs: tc.subscribed, subscriber: subscriber}

			err = sub.subscribeNewTopics(tc.newTopics, tc.subscribed, tc.forced)
			assert.Nil(t, err)
			subscriber.AssertNumberOfCalls(t, "Subscribe", tc.subCalled)
			assert.Equal(t, tc.subsLen, len(sub.subs))
		})
	}
}

func TestRegisterToProxy(t *testing.T) {
	registerer := &mocks.Registerer{}
	pairs := []*proto.MessageTopicPair{
		&proto.MessageTopicPair{
			Message: "msg",
			Topic:   []string{"t1", "t2"},
			Act:     []proto.MessageTopicPair_Action{proto.MessageTopicPair_PUB, proto.MessageTopicPair_SUB},
		},
	}
	pRole := &proto.UserRole{
		Layer: "abc",
		Role:  "xyz",
		Shard: int32(123),
	}
	var err error
	registerer.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pairs, pRole, err)

	sub := &SubManager{
		info: info{
			nodeMode: common.NodeModeAuto,
			peerID:   peer.ID(""),
		},
		registerer: registerer,
	}

	topics, role, err := sub.registerToProxy("", "", "", []byte{1})
	assert.Nil(t, err)

	expRole := userRole{
		layer:   "abc",
		role:    "xyz",
		shardID: 123,
	}
	assert.Equal(t, expRole, role)

	if assert.Len(t, topics, 1) {
		if tp, ok := topics[pairs[0].Message]; assert.True(t, ok) {
			assert.Equal(t, pairs[0].Topic[0], tp[0].Name)
			assert.Equal(t, pairs[0].Topic[1], tp[1].Name)
			assert.Equal(t, pairs[0].Act[0], tp[0].Act)
			assert.Equal(t, pairs[0].Act[1], tp[1].Act)
		}
	}
}
