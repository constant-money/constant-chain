package consensus

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/metrics/monitor"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus/blsbft"
	blsbft2 "github.com/incognitochain/incognito-chain/consensus/blsbftv2"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/wire"
)

type Engine struct {
	BFTProcess           map[int]ConsensusInterface //chainID -> consensus
	userMiningPublicKeys map[string]*incognitokey.CommitteePublicKey
	userKeyListString    string
	consensusName        string
	currentMiningProcess ConsensusInterface
	config               *EngineConfig
	IsEnabled            int //0 > stop, 1: running

	curringMiningState struct {
		layer   string
		role    string
		chainID int
	}

	version int

	lock *sync.Mutex
}

func (engine *Engine) GetUserLayer() (string, int) {
	return engine.curringMiningState.layer, engine.curringMiningState.chainID
}

func (s *Engine) GetUserRole() (string, string, int) {
	layer := s.curringMiningState.layer
	if s.curringMiningState.role == common.WaitingRole {
		layer = "shard"
	}
	return layer, s.curringMiningState.role, s.curringMiningState.chainID
}

func (engine *Engine) IsOngoing(chainName string) bool {
	if engine.currentMiningProcess == nil {
		return false
	}
	return engine.currentMiningProcess.IsOngoing()
}

//TODO: remove all places use this function
// func (engine *Engine) CommitteeChange(chainName string) {
// 	return
// }

func (s *Engine) WatchCommitteeChange() {

	defer func() {
		time.AfterFunc(time.Second*3, s.WatchCommitteeChange)
	}()

	//check if enable
	if s.IsEnabled == 0 || s.config == nil {
		return
	}

	//extract role, layer, chainID
	role, chainID := s.config.Node.GetUserMiningState()
	if s.curringMiningState.role != role || s.curringMiningState.chainID != chainID {
		Logger.Log.Infof("Node state role: %v chainID: %v", role, chainID)
	}

	s.curringMiningState.chainID = chainID
	s.curringMiningState.role = role

	if chainID == -2 {
		s.curringMiningState.role = role
		s.curringMiningState.layer = ""
		s.NotifyBeaconRole(false)
		s.NotifyShardRole(-2)
	} else if chainID == -1 {
		s.curringMiningState.layer = "beacon"
		s.NotifyBeaconRole(true)
		s.NotifyShardRole(-1)
	} else if chainID >= 0 {
		s.curringMiningState.layer = "shard"
		s.NotifyBeaconRole(false)
		s.NotifyShardRole(chainID)
	} else {
		panic("User Mining State Error")
	}

	for _, BFTProcess := range s.BFTProcess {
		if role != "committee" || chainID != BFTProcess.GetChainID() {
			BFTProcess.Stop()
		}
	}

	monitor.SetGlobalParam("Role", s.curringMiningState.role)
	monitor.SetGlobalParam("Layer", s.curringMiningState.layer)
	monitor.SetGlobalParam("ShardID", s.curringMiningState.chainID)

	var miningProcess ConsensusInterface = nil
	//TODO: optimize - if in pending start to listen propose block, but not vote
	if role == "committee" {
		chainName := "beacon"

		if chainID >= 0 {
			chainName = fmt.Sprintf("shard-%d", chainID)
		}

		s.updateVersion(chainID)
		if _, ok := s.BFTProcess[chainID]; !ok {
			if len(s.config.Blockchain.ShardChain)-1 < chainID {
				panic("Chain " + chainName + " not available")
			}
			s.initProcess(chainID, chainName)
		} else { //if not run correct version => stop and init
			if s.version == 1 {
				if _, ok := s.BFTProcess[chainID].(*blsbft.BLSBFT); !ok {
					s.BFTProcess[chainID].Stop()
					s.initProcess(chainID, chainName)
				}
			}
			if s.version == 2 {
				if _, ok := s.BFTProcess[chainID].(*blsbft2.BLSBFT_V2); !ok {
					s.BFTProcess[chainID].Stop()
					s.initProcess(chainID, chainName)
				}
			}
		}

		s.BFTProcess[chainID].Start()
		miningProcess = s.BFTProcess[chainID]
		s.currentMiningProcess = s.BFTProcess[chainID]
		if err := s.LoadMiningKeys(s.userKeyListString); err != nil {
			panic(err)
		}

	}
	s.currentMiningProcess = miningProcess
}

func NewConsensusEngine() *Engine {
	Logger.Log.Infof("CONSENSUS: NewConsensusEngine")

	engine := &Engine{
		BFTProcess:           make(map[int]ConsensusInterface),
		consensusName:        common.BlsConsensus,
		userMiningPublicKeys: make(map[string]*incognitokey.CommitteePublicKey),
		version:              1,
		lock:                 new(sync.Mutex),
	}
	return engine
}

func (engine *Engine) initProcess(chainID int, chainName string) {
	if engine.version == 1 {
		if chainID == -1 {
			engine.BFTProcess[chainID] = blsbft.NewInstance(engine.config.Blockchain.BeaconChain, chainName, chainID, engine.config.Node, Logger.Log)
		} else {
			engine.BFTProcess[chainID] = blsbft.NewInstance(engine.config.Blockchain.ShardChain[chainID], chainName, chainID, engine.config.Node, Logger.Log)
		}
	} else {
		if chainID == -1 {
			engine.BFTProcess[chainID] = blsbft2.NewInstance(engine.config.Blockchain.BeaconChain, chainName, chainID, engine.config.Node, Logger.Log)
		} else {
			engine.BFTProcess[chainID] = blsbft2.NewInstance(engine.config.Blockchain.ShardChain[chainID], chainName, chainID, engine.config.Node, Logger.Log)
		}
	}
}

func (engine *Engine) updateVersion(chainID int) {
	chainEpoch := uint64(1)
	if chainID == -1 {
		chainEpoch = engine.config.Blockchain.BeaconChain.GetEpoch()
	} else {
		chainEpoch = engine.config.Blockchain.ShardChain[chainID].GetEpoch()
	}

	if chainEpoch >= engine.config.Blockchain.GetConfig().ChainParams.ConsensusV2Epoch {
		engine.version = 2
	}
}

func (engine *Engine) Init(config *EngineConfig) {
	engine.config = config
	go engine.WatchCommitteeChange()
}

func (engine *Engine) Start() error {
	defer Logger.Log.Infof("CONSENSUS: Start")
	if engine.config.Node.GetPrivateKey() != "" {
		keyList, err := engine.GenMiningKeyFromPrivateKey(engine.config.Node.GetPrivateKey())
		if err != nil {
			panic(err)
		}
		engine.userKeyListString = keyList
	} else if engine.config.Node.GetMiningKeys() != "" {
		engine.userKeyListString = engine.config.Node.GetMiningKeys()
	}
	err := engine.LoadMiningKeys(engine.userKeyListString)
	if err != nil {
		panic(err)
	}
	engine.IsEnabled = 1
	return nil
}

func (engine *Engine) Stop() error {
	Logger.Log.Infof("CONSENSUS: Stop")
	for _, BFTProcess := range engine.BFTProcess {
		BFTProcess.Stop()
		engine.currentMiningProcess = nil
	}
	engine.IsEnabled = 0
	return nil
}

func (engine *Engine) OnBFTMsg(msg *wire.MessageBFT) {
	if engine.currentMiningProcess == nil {
		Logger.Log.Warnf("Current mining process still nil")
		return
	}
	if engine.currentMiningProcess.GetChainKey() == msg.ChainKey {
		engine.currentMiningProcess.ProcessBFTMsg(msg)
	}
}

func (engine *Engine) NotifyBeaconRole(beaconRole bool) {
	engine.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.BeaconRoleTopic, beaconRole))
}
func (engine *Engine) NotifyShardRole(shardRole int) {
	engine.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.ShardRoleTopic, shardRole))
}
