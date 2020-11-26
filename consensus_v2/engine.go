package consensus_v2

import (
	"fmt"
	"strings"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/consensus"
	"github.com/incognitochain/incognito-chain/consensus_v2/blsbft"
	blsbft2 "github.com/incognitochain/incognito-chain/consensus_v2/blsbftv2"
	blsbft3 "github.com/incognitochain/incognito-chain/consensus_v2/blsbftv3"
	signatureschemes2 "github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
)

type Engine struct {
	BFTProcess map[int]ConsensusInterface //chainID -> consensus
	validators []*consensus.Validator     //list of validator
	version    map[int]int                //chainID -> version

	consensusName string
	config        *EngineConfig
	IsEnabled     int //0 > stop, 1: running

	//legacy code -> single process
	userMiningPublicKeys *incognitokey.CommitteePublicKey
	userKeyListString    string
	currentMiningProcess ConsensusInterface
}

//just get role of first validator
//this function support NODE monitor (getmininginfo) which assumed running only 1 validator
func (s *Engine) GetUserRole() (string, string, int) {
	for _, validator := range s.validators {
		return validator.State.Layer, validator.State.Role, validator.State.ChainID
	}
	return "", "", -2
}

func (s *Engine) GetCurrentValidators() []*consensus.Validator {
	return s.validators
}

func (s *Engine) GetOneValidator() *consensus.Validator {
	if len(s.validators) > 0 {
		return s.validators[0]
	}
	return nil
}

func (s *Engine) GetOneValidatorForEachConsensusProcess() map[int]*consensus.Validator {
	chainValidator := make(map[int]*consensus.Validator)
	if len(s.validators) > 0 {
		for _, validator := range s.validators {
			if validator.State.ChainID != -2 {
				_, ok := chainValidator[validator.State.ChainID]
				if ok {
					if validator.State.Role == common.CommitteeRole {
						chainValidator[validator.State.ChainID] = validator
					}
				} else {
					chainValidator[validator.State.ChainID] = validator
				}
			}
		}
	}

	//fmt.Println("GetOneValidatorForEachConsensusProcess", chainValidator[1])
	return chainValidator
}

func (s *Engine) WatchCommitteeChange() {

	defer func() {
		time.AfterFunc(time.Second*3, s.WatchCommitteeChange)
	}()

	//check if enable
	if s.IsEnabled == 0 || s.config == nil {
		return
	}

	ValidatorGroup := make(map[int][]consensus.Validator)
	for _, validator := range s.validators {
		s.userMiningPublicKeys = validator.MiningKey.GetPublicKey()
		s.userKeyListString = validator.PrivateSeed
		role, chainID := s.config.Node.GetPubkeyMiningState(validator.MiningKey.GetPublicKey())
		//Logger.Log.Info(validator.miningKey.GetPublicKey().GetMiningKeyBase58(common.BlsConsensus))
		if chainID == -1 {
			validator.State = consensus.MiningState{role, "beacon", -1}
		} else if chainID > -1 {
			validator.State = consensus.MiningState{role, "shard", chainID}
		} else {
			if role != "" {
				validator.State = consensus.MiningState{role, "shard", -2}
			} else {
				validator.State = consensus.MiningState{role, "", -2}
			}
		}

		//group all validator as committee by chainID
		if role == common.CommitteeRole {
			//fmt.Println("Consensus", chainID, validator.PrivateSeed, validator.State)
			ValidatorGroup[chainID] = append(ValidatorGroup[chainID], *validator)
		}
	}

	miningProc := ConsensusInterface(nil)
	for chainID, validators := range ValidatorGroup {
		chainName := "beacon"
		if chainID >= 0 {
			chainName = fmt.Sprintf("shard-%d", chainID)
		}
		s.updateVersion(chainID)
		if _, ok := s.BFTProcess[chainID]; !ok {
			s.initProcess(chainID, chainName)
		} else { //if not run correct version => stop and init
			if s.version[chainID] == 1 {
				if _, ok := s.BFTProcess[chainID].(*blsbft.BLSBFT); !ok {
					s.BFTProcess[chainID].Stop()
					s.initProcess(chainID, chainName)
				}
			}
			if s.version[chainID] == 2 {
				if _, ok := s.BFTProcess[chainID].(*blsbft2.BLSBFT_V2); !ok {
					s.BFTProcess[chainID].Stop()
					s.initProcess(chainID, chainName)
				}
			}
			if s.version[chainID] == 3 {
				if _, ok := s.BFTProcess[chainID].(*blsbft3.BLSBFT_V3); !ok {
					s.BFTProcess[chainID].Stop()
					s.initProcess(chainID, chainName)
				}
			}
		}
		validatorMiningKey := []signatureschemes2.MiningKey{}
		for _, validator := range validators {
			validatorMiningKey = append(validatorMiningKey, validator.MiningKey)
		}
		s.BFTProcess[chainID].LoadUserKeys(validatorMiningKey)
		s.BFTProcess[chainID].Start()
		miningProc = s.BFTProcess[chainID]
	}

	for chainID, proc := range s.BFTProcess {
		if _, ok := ValidatorGroup[chainID]; !ok {
			proc.Stop()
		}
	}

	s.currentMiningProcess = miningProc
}

func NewConsensusEngine() *Engine {
	Logger.Log.Infof("CONSENSUS: NewConsensusEngine")
	engine := &Engine{
		BFTProcess:    make(map[int]ConsensusInterface),
		consensusName: common.BlsConsensus,
		version:       make(map[int]int),
	}
	return engine
}

func (engine *Engine) initProcess(chainID int, chainName string) {
	if engine.version[chainID] == 1 {
		if chainID == -1 {
			engine.BFTProcess[chainID] = blsbft.NewInstance(engine.config.Blockchain.BeaconChain, chainName, chainID, engine.config.Node, Logger.Log)
		} else {
			engine.BFTProcess[chainID] = blsbft.NewInstance(engine.config.Blockchain.ShardChain[chainID], chainName, chainID, engine.config.Node, Logger.Log)
		}
	} else if engine.version[chainID] == 2 {
		if chainID == -1 {
			engine.BFTProcess[chainID] = blsbft2.NewInstance(engine.config.Blockchain.BeaconChain, chainName, chainID, engine.config.Node, Logger.Log)
		} else {
			engine.BFTProcess[chainID] = blsbft2.NewInstance(engine.config.Blockchain.ShardChain[chainID], chainName, chainID, engine.config.Node, Logger.Log)
		}
	} else if engine.version[chainID] == 3 {
		if chainID == -1 {
			engine.BFTProcess[chainID] = blsbft3.NewInstance(
				engine.config.Blockchain.BeaconChain,
				engine.config.Blockchain.BeaconChain,
				chainName, chainID,
				engine.config.Node, Logger.Log)
		} else {
			engine.BFTProcess[chainID] = blsbft3.NewInstance(
				engine.config.Blockchain.ShardChain[chainID],
				engine.config.Blockchain.BeaconChain,
				chainName, chainID,
				engine.config.Node, Logger.Log)
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
		engine.version[chainID] = 2
	}

	if chainEpoch >= engine.config.Blockchain.GetConfig().ChainParams.ConsensusV3Epoch {
		engine.version[chainID] = 3
	}
}

func (engine *Engine) Init(config *EngineConfig) {
	engine.config = config
	go engine.WatchCommitteeChange()
}

func (engine *Engine) Start() error {
	defer Logger.Log.Infof("CONSENSUS: Start")

	if engine.config.Node.GetPrivateKey() != "" {
		privateSeed, err := engine.GenMiningKeyFromPrivateKey(engine.config.Node.GetPrivateKey())
		if err != nil {
			panic(err)
		}
		miningKey, err := GetMiningKeyFromPrivateSeed(privateSeed)
		if err != nil {
			panic(err)
		}
		engine.validators = []*consensus.Validator{&consensus.Validator{PrivateSeed: privateSeed, MiningKey: *miningKey}}
	} else if engine.config.Node.GetMiningKeys() != "" {
		keys := strings.Split(engine.config.Node.GetMiningKeys(), ",")
		engine.validators = []*consensus.Validator{}
		for _, key := range keys {
			miningKey, err := GetMiningKeyFromPrivateSeed(key)
			if err != nil {
				panic(err)
			}
			engine.validators = append(engine.validators, &consensus.Validator{PrivateSeed: key, MiningKey: *miningKey})
		}
		engine.validators = engine.validators[:1] //allow only 1 key
	}
	engine.IsEnabled = 1
	return nil
}

func (engine *Engine) Stop() error {
	Logger.Log.Infof("CONSENSUS: Stop")
	for _, BFTProcess := range engine.BFTProcess {
		BFTProcess.Stop()
	}
	engine.IsEnabled = 0
	return nil
}

func (engine *Engine) OnBFTMsg(msg *wire.MessageBFT) {
	for _, process := range engine.BFTProcess {
		if process.IsStarted() && process.GetChainKey() == msg.ChainKey {
			process.ProcessBFTMsg(msg)
		}
	}
}

func (engine *Engine) GetAllValidatorKeyState() map[string]consensus.MiningState {
	result := make(map[string]consensus.MiningState)
	for _, validator := range engine.validators {
		result[validator.PrivateSeed] = validator.State
	}
	return result
}

func (engine *Engine) IsCommitteeInShard(shardID byte) bool {
	if shard, ok := engine.BFTProcess[int(shardID)]; ok {
		return shard.IsStarted()
	}
	return false
}
