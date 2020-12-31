package peerv2

import (
	"reflect"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/peerv2/mocks"
	"github.com/incognitochain/incognito-chain/peerv2/rpcclient"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testHighwayAddress = "/ip4/0.0.0.0/tcp/7337/p2p/QmSPa4gxx6PRmoNRu6P2iFwEwmayaoLdR5By3i3MgM9gMv"
var testHighwayAddress2 = "/ip4/0.0.0.0/tcp/7338/p2p/Qmba4kphPTHc3bxsgXJ6aT5SvNT2FoCXq8pe4vHs7kVSZm"

func TestDiscoverHighWay(t *testing.T) {
	type args struct {
		discoverPeerAddress string
		shardsStr           []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	// x, err := DiscoverHighWay("0.0.0.0:9330", []string{"all"})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcClient := rpcclient.RPCClient{}
			got, err := rpcClient.DiscoverHighway(tt.args.discoverPeerAddress, tt.args.shardsStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiscoverHighWay() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DiscoverHighWay() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConnectAtStart makes sure connection is established at start-up time
func TestConnectAtStart(t *testing.T) {
	defer configTime()()
	h, net := setupHost()
	// net.On("Connectedness", mock.Anything).Return(network.NotConnected).Return(network.Connected)
	setupConnectedness(net, []network.Connectedness{network.NotConnected, network.Connected})
	var err error
	h.On("Connect", mock.Anything, mock.Anything).Return(err)

	hwAddrs := map[string][]rpcclient.HighwayAddr{"all": []rpcclient.HighwayAddr{rpcclient.HighwayAddr{Libp2pAddr: testHighwayAddress}}}
	discoverer := &mocks.HighwayDiscoverer{}
	discoverer.On("DiscoverHighway", mock.Anything, mock.Anything).Return(hwAddrs, nil)
	cm := &ConnManager{
		LocalHost:        &Host{Host: h},
		discoverer:       discoverer,
		stop:             make(chan int),
		registerRequests: make(chan peer.ID, 1),
		keeper:           NewAddrKeeper(),
	}
	go cm.keepHighwayConnection()
	time.Sleep(200 * time.Millisecond)
	close(cm.stop)

	assert.Equal(t, 1, len(cm.registerRequests), "not connect at startup")
}

// TestConnectWhenMaxedRetry checks if new highway is picked when failing to connect to old highway for some number of times
func TestConnectWhenMaxedRetry(t *testing.T) {
	defer configTime()()

	h, net := setupHost()
	setupConnectedness(net, []network.Connectedness{network.NotConnected, network.Connected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected, network.NotConnected})
	var err error
	h.On("Connect", mock.Anything, mock.Anything).Return(err)

	hwAddrs := map[string][]rpcclient.HighwayAddr{
		"all": []rpcclient.HighwayAddr{
			rpcclient.HighwayAddr{Libp2pAddr: testHighwayAddress},
			rpcclient.HighwayAddr{Libp2pAddr: testHighwayAddress2},
		},
	}
	discoverer := &mocks.HighwayDiscoverer{}
	discoverer.On("DiscoverHighway", mock.Anything, mock.Anything).Return(hwAddrs, nil).Times(10)
	cm := &ConnManager{
		LocalHost:        &Host{Host: h},
		discoverer:       discoverer,
		stop:             make(chan int),
		registerRequests: make(chan peer.ID, 1),
		keeper:           NewAddrKeeper(),
	}
	go cm.keepHighwayConnection()
	time.Sleep(1 * time.Second)
	close(cm.stop)

	discoverer.AssertNumberOfCalls(t, "DiscoverHighway", 2)
	assert.Equal(t, 1, len(cm.keeper.ignoreHWUntil))
}

// TestCheckConnectionAfterMaxedOut checks if we re-register when trying a new highway after maxing out a number of retries to the old one
func TestCheckConnectionAfterMaxedOut(t *testing.T) {
	h, net := setupHost()
	// Not x 6 -> Con
	setupConnectedness(
		net,
		[]network.Connectedness{
			network.NotConnected, network.NotConnected,
			network.NotConnected, network.NotConnected,
			network.NotConnected, network.NotConnected,
			network.NotConnected, network.NotConnected,
			network.NotConnected, network.NotConnected,
			network.NotConnected, network.NotConnected,
			network.NotConnected,
			network.Connected, network.Connected,
		},
	)
	var err error
	h.On("Connect", mock.Anything, mock.Anything).Return(err)

	cm := ConnManager{
		DiscoverPeersAddress: testHighwayAddress,
		LocalHost:            &Host{Host: h},
		registerRequests:     make(chan peer.ID, 5),
		keeper:               NewAddrKeeper(),
	}
	for i := 0; i < 8; i++ {
		cm.checkConnection(&peer.AddrInfo{})
	}

	assert.Equal(t, 1, len(cm.registerRequests), "not reconnect")
}

// TestReconnect checks if connection is re-established after being disconnected
func TestReconnect(t *testing.T) {
	h, net := setupHost()
	// Not -> Con -> Not -> Con
	setupConnectedness(
		net,
		[]network.Connectedness{
			network.NotConnected,
			network.Connected,
			network.NotConnected,
			network.Connected,
		},
	)
	var err error
	h.On("Connect", mock.Anything, mock.Anything).Return(err)

	cm := ConnManager{
		DiscoverPeersAddress: testHighwayAddress,
		LocalHost:            &Host{Host: h},
		registerRequests:     make(chan peer.ID, 5),
		keeper:               NewAddrKeeper(),
	}
	for i := 0; i < 4; i++ {
		maxed := cm.checkConnection(&peer.AddrInfo{})
		assert.False(t, maxed)
	}

	assert.Equal(t, 2, len(cm.registerRequests), "not reconnect")
}

func TestPeriodicManageSub(t *testing.T) {
	defer configTime()()

	sc := new(subscribeCounter)
	cm := ConnManager{
		Requester:        &BlockRequester{},
		stop:             make(chan int),
		registerRequests: make(chan peer.ID, 10),
		Subscriber:       sc,
		keeper:           NewAddrKeeper(),
	}
	go cm.manageRoleSubscription()
	time.Sleep(RegisterTimestep + 50*time.Millisecond)
	close(cm.stop)

	assert.Equal(t, 1, sc.normal, "not subbed")
}

func TestForcedSub(t *testing.T) {
	defer configTime()()

	sc := new(subscribeCounter)
	cm := ConnManager{
		Requester:        &BlockRequester{},
		stop:             make(chan int),
		registerRequests: make(chan peer.ID, 10),
		Subscriber:       sc,
		keeper:           NewAddrKeeper(),
	}
	cm.registerRequests <- peer.ID("") // Sent forced, must sub with forced = True next time
	go cm.manageRoleSubscription()
	time.Sleep(RegisterTimestep + 50*time.Millisecond)
	close(cm.stop)

	assert.Equal(t, 1, sc.forced, "not subbed")
}

type subscribeCounter struct {
	normal int
	forced int
}

func (subCounter *subscribeCounter) Subscribe(forced bool) error {
	if forced {
		subCounter.forced++
	} else {
		subCounter.normal++
	}
	return nil
}

func (subCounter *subscribeCounter) GetMsgToTopics() msgToTopics {
	return msgToTopics{}
}

func setupHost() (*mocks.Host, *mocks.Network) {
	net := &mocks.Network{}
	h := &mocks.Host{}
	h.On("Network").Return(net)
	h.On("ID").Return(peer.ID(""))
	return h, net
}

func setupConnectedness(net *mocks.Network, values []network.Connectedness) {
	idx := -1
	net.On("Connectedness", mock.Anything).Return(func(_ peer.ID) network.Connectedness {
		if idx+1 < len(values) {
			idx += 1
		}
		return values[idx]
	})
}

func configTime() func() {
	reconnectHighwayTimestep := ReconnectHighwayTimestep
	requesterDialTimestep := RequesterDialTimestep
	registerTimestep := RegisterTimestep
	ReconnectHighwayTimestep = 100 * time.Millisecond
	RequesterDialTimestep = 100 * time.Millisecond
	RegisterTimestep = 100 * time.Millisecond

	return func() {
		// Revert time configuration after a test is done
		ReconnectHighwayTimestep = reconnectHighwayTimestep
		RequesterDialTimestep = requesterDialTimestep
		RegisterTimestep = registerTimestep
	}
}

func init() {
	Logger.Init(common.NewBackend(nil).Logger("test", true))
}
