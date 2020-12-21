package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus_v2/blsbftv2"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
	libp2p "github.com/libp2p/go-libp2p-peer"
)

var mapNodeID = make(map[string]string)
var mapLock = sync.Mutex{}

type Node struct {
	id              string
	consensusEngine *blsbftv2.BLSBFT_V2
	chain           *Chain
	nodeList        []*Node
}

type logWriter struct {
	NodeID string
	fd     *os.File
}

func (s logWriter) Write(p []byte) (n int, err error) {
	s.fd.Write(p)
	return len(p), nil
}

func NewNode(committeePkStruct []incognitokey.CommitteePublicKey, miningKey *signatureschemes.MiningKey, index int, testname string) *Node {
	name := fmt.Sprintf("%s_log%d", testname, index)
	fd, err := os.OpenFile(fmt.Sprintf("%s.log", name), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	fd.Truncate(0)
	backendLog := common.NewBackend(logWriter{
		NodeID: name,
		fd:     fd,
	})
	consensusLogger := backendLog.Logger("Consensus", false)
	consensusLogger.SetLevel(1)
	chainViewLogger := backendLog.Logger("ShardView", false)
	chainViewLogger.SetLevel(1)

	node := Node{id: fmt.Sprintf("%d", index)}
	node.chain = NewChain(0, "shard0", committeePkStruct)

	node.consensusEngine = &blsbftv2.BLSBFT_V2{
		Chain:    node.chain,
		Node:     &node,
		ChainKey: "shard",
		PeerID:   name,
		Logger:   consensusLogger,
	}
	node.consensusEngine.LoadUserKeys([]signatureschemes.MiningKey{*miningKey})
	return &node
}

func (s *Node) PushMessageToChain(msg wire.Message, chain common.ChainInterface) error {
	time.AfterFunc(time.Millisecond*100, func() {
		if msg.(*wire.MessageBFT).Type == "propose" {
			timeSlot := uint64(msg.(*wire.MessageBFT).TimeSlot)
			if timeSlot > GetSimulation().maxTimeSlot {
				os.Exit(0)
			}
			pComm := GetSimulation().scenario.proposeComm
			if senderComm, ok := pComm[timeSlot]; ok {
				for i, c := range s.nodeList {
					if s.id == c.id {
						continue
					}
					if senderComm[i] == 0 {
						fmt.Printf("Don'trrrrrr send propose from %v to %v at timeslot %v\n", s.id, c.id, timeSlot-GetSimulation().startTimeSlot+1)
						// c.consensusEngine.ProcessBFTMsg(msg.(*wire.MessageBFT))
					} else {
						fmt.Printf("Send propose from %v to %v at timeslot %v\n", s.id, c.id, timeSlot-GetSimulation().startTimeSlot+1)
						c.consensusEngine.ProcessBFTMsg(msg.(*wire.MessageBFT))
					}
				}
			} else {
				for _, c := range s.nodeList {
					if s.id == c.id {
						continue
					}
					fmt.Printf("Send* propose from %v to %v at timeslot %v\n", s.id, c.id, timeSlot-GetSimulation().startTimeSlot+1)
					c.consensusEngine.ProcessBFTMsg(msg.(*wire.MessageBFT))
				}
			}
			return
		}

		if msg.(*wire.MessageBFT).Type == "vote" {
			vComm := GetSimulation().scenario.voteComm
			timeSlot := uint64(msg.(*wire.MessageBFT).TimeSlot)
			if timeSlot > GetSimulation().maxTimeSlot {
				os.Exit(0)
			}
			if comm, ok := vComm[timeSlot]; ok {
				for i, c := range s.nodeList {
					if s.id == c.id {
						continue
					}
					if senderComm, ok := comm[s.id]; ok {
						if senderComm[i] == 0 {
							fmt.Printf("Don't send vote from %v to %v at timeslot %v\n", s.id, c.id, timeSlot-GetSimulation().startTimeSlot+1)
							// c.consensusEngine.ProcessBFTMsg(msg.(*wire.MessageBFT))
						} else {
							fmt.Printf("Send vote from %v to %v at timeslot %v\n", s.id, c.id, timeSlot-GetSimulation().startTimeSlot+1)
							c.consensusEngine.ProcessBFTMsg(msg.(*wire.MessageBFT))
						}
					} else {
						fmt.Printf("Send vote from %v to %v at timeslot %v\n", s.id, c.id, timeSlot-GetSimulation().startTimeSlot+1)
						c.consensusEngine.ProcessBFTMsg(msg.(*wire.MessageBFT))
					}

				}
			} else {
				for _, c := range s.nodeList {
					if s.id == c.id {
						continue
					}
					fmt.Printf("Send* vote from %v to %v at timeslot %v\n", s.id, c.id, timeSlot-GetSimulation().startTimeSlot+1)
					c.consensusEngine.ProcessBFTMsg(msg.(*wire.MessageBFT))
				}
			}
			return
		}
	})

	return nil
}

func (s *Node) RequestMissingViewViaStream(peerID string, hashes [][]byte, fromCID int, chainName string) (err error) {
	str := []string{}
	for _, h := range hashes {
		pH, _ := common.Hash{}.NewHash(h)
		str = append(str, pH.String())
	}
	for _, c := range s.nodeList {
		if s.id == c.id {
			continue
		}
		if c.id == mapNodeID[peerID] {
			for _, h := range hashes {
				pH, _ := common.Hash{}.NewHash(h)
				v := c.chain.multiview.GetViewByHash(*pH)
				s.chain.multiview.AddView(v)
			}
		}
	}
	fmt.Printf("Node %v RequestMissingViewViaStream from %v hash %v \n", s.id, mapNodeID[peerID], strings.Join(str, ","))
	return nil
}

func (s *Node) GetSelfPeerID() libp2p.ID {
	return libp2p.ID(s.id)
}

func (s *Node) Start() {
	fmt.Printf("Node %s log is %s, peerID: %v \n", s.id, fmt.Sprintf("log%s.log", s.id), libp2p.ID(s.id).String())
	mapLock.Lock()
	defer mapLock.Unlock()
	mapNodeID[libp2p.ID(s.id).String()] = s.id
	s.consensusEngine.Start()
}

func (s *Node) Stop() {
	mapLock.Lock()
	defer mapLock.Unlock()
	delete(mapNodeID, libp2p.ID(s.id).String())
	s.consensusEngine.Stop()
}
