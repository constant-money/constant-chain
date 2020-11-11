package blsbft

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/metrics/monitor"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/consensus/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
)

type BLSBFT struct {
	Chain    ChainInterface
	Node     NodeInterface
	ChainKey string
	ChainID  int
	PeerID   string

	UserKeySet       *MiningKey
	BFTMessageCh     chan wire.MessageBFT
	ProposeMessageCh chan BFTPropose
	VoteMessageCh    chan BFTVote

	RoundData struct {
		TimeStart         time.Time
		Block             types.BlockInterface
		BlockHash         common.Hash
		BlockValidateData ValidationData
		lockVotes         sync.Mutex
		Votes             map[string]vote
		Round             int
		NextHeight        uint64
		State             string
		NotYetSendVote    bool
		Committee         []incognitokey.CommitteePublicKey
		CommitteeBLS      struct {
			StringList []string
			ByteList   []blsmultisig.PublicKey
		}
		LastProposerIndex int
	}
	Blocks         map[string]types.BlockInterface
	EarlyVotes     map[string]map[string]vote
	lockEarlyVotes sync.Mutex
	isOngoing      bool
	isStarted      bool
	StopCh         chan struct{}
	logger         common.Logger
}

func (e *BLSBFT) IsOngoing() bool {
	return e.isOngoing
}

func (e *BLSBFT) IsStarted() bool {
	return e.isStarted
}

func (e *BLSBFT) GetConsensusName() string {
	return consensusName
}

func (e *BLSBFT) GetChainKey() string {
	return e.ChainKey
}
func (e *BLSBFT) GetChainID() int {
	return e.ChainID
}

func (e *BLSBFT) Stop() error {
	if e.isStarted {
		e.logger.Info("stop bls-bft consensus for chain", e.ChainKey)
		close(e.StopCh)
		e.isStarted = false
		e.isOngoing = false
		return nil
	}
	return NewConsensusError(ConsensusAlreadyStoppedError, errors.New(e.ChainKey))
}

func (e *BLSBFT) Start() error {
	if e.isStarted {
		return NewConsensusError(ConsensusAlreadyStartedError, errors.New(e.ChainKey))
	}

	e.isStarted = true
	e.isOngoing = false
	e.StopCh = make(chan struct{})
	e.EarlyVotes = make(map[string]map[string]vote)
	e.Blocks = map[string]types.BlockInterface{}
	e.ProposeMessageCh = make(chan BFTPropose)
	e.VoteMessageCh = make(chan BFTVote)
	e.InitRoundData()

	e.logger.Info("start bls-bft consensus for chain", e.ChainKey)

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		fmt.Println("action")
		for { //actor loop
			select {
			case <-e.StopCh:
				e.logger.Info("Exit BFT")
				return
			case proposeMsg := <-e.ProposeMessageCh:
				block, err := e.Chain.UnmarshalBlock(proposeMsg.Block)
				if err != nil {
					e.logger.Info(err)
					continue
				}
				blockRoundKey := getRoundKey(block.GetHeight(), block.GetRound())
				e.logger.Info("receive block", blockRoundKey, getRoundKey(e.RoundData.NextHeight, e.RoundData.Round))
				if block.GetHeight() == e.RoundData.NextHeight {
					if e.RoundData.Round == block.GetRound() {
						if e.RoundData.Block == nil {
							e.Blocks[blockRoundKey] = block
							continue
						}
					} else {
						if e.RoundData.Round < block.GetRound() {
							e.Blocks[blockRoundKey] = block
							continue
						}
					}
					continue
				}
				if block.GetHeight() > e.RoundData.NextHeight {
					e.Blocks[blockRoundKey] = block
					continue
				}
			case msg := <-e.VoteMessageCh:
				e.logger.Info("Receive vote", msg.RoundKey, getRoundKey(e.RoundData.NextHeight, e.RoundData.Round))
				validatorIdx := common.IndexOfStr(msg.Validator, e.RoundData.CommitteeBLS.StringList)
				if validatorIdx == -1 {
					continue
				}
				height, round := parseRoundKey(msg.RoundKey)
				if height < e.RoundData.NextHeight {
					continue
				}
				if (height == e.RoundData.NextHeight) && (round < e.RoundData.Round) {
					continue
				}
				// roundKey := getRoundKey(e.RoundData.NextHeight, e.RoundData.Round)
				if (height == e.RoundData.NextHeight) && (round == e.RoundData.Round) {
					//validate single sig
					if !(new(common.Hash).IsEqual(&e.RoundData.BlockHash)) {
						e.RoundData.lockVotes.Lock()
						if _, ok := e.RoundData.Votes[msg.Validator]; !ok {
							// committeeArr := []incognitokey.CommitteePublicKey{}
							// committeeArr = append(committeeArr, e.RoundData.Committee...)
							e.RoundData.lockVotes.Unlock()
							go func(voteMsg BFTVote, blockHash common.Hash, committee []incognitokey.CommitteePublicKey) {
								if err := e.preValidateVote(blockHash.GetBytes(), &(voteMsg.Vote), committee[validatorIdx].MiningPubKey[common.BridgeConsensus]); err != nil {
									e.logger.Error(err)
									return
								}
								if len(voteMsg.Vote.BRI) != 0 {
									if err := validateSingleBriSig(&blockHash, voteMsg.Vote.BRI, committee[validatorIdx].MiningPubKey[common.BridgeConsensus]); err != nil {
										e.logger.Error(err)
										return
									}
								}
								go func() {
									voteCtnBytes, err := json.Marshal(voteMsg)
									if err != nil {
										e.logger.Error(NewConsensusError(UnExpectedError, err))
										return
									}
									msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
									msg.(*wire.MessageBFT).ChainKey = e.ChainKey
									msg.(*wire.MessageBFT).Content = voteCtnBytes
									msg.(*wire.MessageBFT).Type = MSG_VOTE
									// TODO uncomment here when switch to non-highway mode
									// e.Node.PushMessageToChain(msg, e.Chain)
								}()
								e.addVote(voteMsg)
							}(msg, e.RoundData.BlockHash, append([]incognitokey.CommitteePublicKey{}, e.RoundData.Committee...))
							continue
						} else {
							e.RoundData.lockVotes.Unlock()
							continue
						}
					}
				}
				e.addEarlyVote(msg)

			case <-ticker.C:

				monitor.SetGlobalParam("RoundKey", getRoundKey(e.RoundData.NextHeight, e.RoundData.Round), "Phase", e.RoundData.State)

				pubKey := e.UserKeySet.GetPublicKey()
				if common.IndexOfStr(pubKey.GetMiningKeyBase58(consensusName), e.RoundData.CommitteeBLS.StringList) == -1 {
					e.enterNewRound()
					continue
				}

				if !e.Chain.IsReady() {
					e.isOngoing = false
					//fmt.Println("CONSENSUS: ticker 1")
					continue
				}

				if !e.isInTimeFrame() || e.RoundData.State == "" {
					e.enterNewRound()
				}

				switch e.RoundData.State {
				case listenPhase:
					// timeout or vote nil?
					//fmt.Println("CONSENSUS: listen phase 1")
					if e.Chain.CurrentHeight() == e.RoundData.NextHeight {
						e.enterNewRound()
						continue
					}
					roundKey := getRoundKey(e.RoundData.NextHeight, e.RoundData.Round)
					if e.Blocks[roundKey] != nil {
						monitor.SetGlobalParam("ReceiveBlockTime", time.Since(e.RoundData.TimeStart).Seconds())
						//fmt.Println("CONSENSUS: listen phase 2")

						if err := e.Chain.ValidatePreSignBlock(e.Blocks[roundKey], []incognitokey.CommitteePublicKey{}); err != nil {
							delete(e.Blocks, roundKey)
							e.logger.Error(err)
							continue
						}

						if e.RoundData.Block == nil {
							// blockData, _ := json.Marshal(e.Blocks[roundKey])
							// msg, _ := MakeBFTProposeMsg(blockData, e.ChainKey, e.UserKeySet)
							// go e.Node.PushMessageToChain(msg, e.Chain)

							e.RoundData.Block = e.Blocks[roundKey]
							e.RoundData.BlockHash = *e.RoundData.Block.Hash()
							valData, err := DecodeValidationData(e.RoundData.Block.GetValidationField())
							if err != nil {
								e.logger.Error(err)
								continue
							}
							e.RoundData.BlockValidateData = *valData
							e.enterVotePhase()
						}
					}
				case votePhase:
					e.logger.Info("Case: In vote phase")
					if e.RoundData.NotYetSendVote {
						err := e.sendVote()
						if err != nil {
							e.logger.Error(err)
							continue
						}
					}
					if !(new(common.Hash).IsEqual(&e.RoundData.BlockHash)) && e.isHasMajorityVotes() {
						e.RoundData.lockVotes.Lock()
						aggSig, brigSigs, validatorIdx, err := combineVotes(e.RoundData.Votes, e.RoundData.CommitteeBLS.StringList)
						e.RoundData.lockVotes.Unlock()
						if err != nil {
							e.logger.Error(err)
							continue
						}

						e.RoundData.BlockValidateData.AggSig = aggSig
						e.RoundData.BlockValidateData.BridgeSig = brigSigs
						e.RoundData.BlockValidateData.ValidatiorsIdx = validatorIdx

						validationDataString, _ := EncodeValidationData(e.RoundData.BlockValidateData)
						e.RoundData.Block.(blockValidation).AddValidationField(validationDataString)

						//TODO: check issue invalid sig when swap
						//TODO 0xakk0r0kamui trace who is malicious node if ValidateCommitteeSig return false
						err = ValidateCommitteeSig(e.RoundData.Block, e.RoundData.Committee)
						if err != nil {
							e.logger.Error(err)
							e.logger.Errorf("e.RoundData.Block.GetValidationField()=%+v\n", e.RoundData.Block.GetValidationField())
							e.logger.Errorf("e.RoundData.Committee=%+v\n", e.RoundData.Committee)
							for _, member := range e.RoundData.Committee {
								e.logger.Errorf("member.MiningPubKey[%+v] %+v\n", consensusName, base58.Base58Check{}.Encode(member.MiningPubKey[consensusName], common.Base58Version))
							}
							continue
						}

						if err := e.Chain.InsertAndBroadcastBlock(e.RoundData.Block); err != nil {
							e.logger.Error(err)
							if blockchainError, ok := err.(*blockchain.BlockChainError); ok {
								if blockchainError.Code != blockchain.ErrCodeMessage[blockchain.DuplicateShardBlockError].Code {
									e.logger.Error(err)
								}
							}
							continue
						}
						monitor.SetGlobalParam("CommitTime", time.Since(time.Unix(e.Chain.GetLastBlockTimeStamp(), 0)).Seconds())
						// e.Node.PushMessageToAll()
						e.logger.Infof("Commit block (%d votes) %+v hash=%+v \n Wait for next round", len(e.RoundData.Votes), e.RoundData.Block.GetHeight(), e.RoundData.Block.Hash().String())
						e.enterNewRound()
					}
				}
			}
		}
	}()
	return nil
}

func (e *BLSBFT) enterProposePhase() {
	if !e.isInTimeFrame() || e.RoundData.State == proposePhase {
		return
	}
	e.setState(proposePhase)
	e.isOngoing = true
	block, err := e.createNewBlock()
	monitor.SetGlobalParam("CreateTime", time.Since(e.RoundData.TimeStart).Seconds())
	if err != nil {
		e.isOngoing = false
		e.logger.Error("can't create block", err)
		return
	}

	if e.Chain.CurrentHeight()+1 != block.GetHeight() {
		return
	}
	validationData := e.CreateValidationData(block)
	validationDataString, err := EncodeValidationData(validationData)
	if err != nil {
		e.logger.Errorf("Encode validation data failed %+v", err)
	}
	block.(blockValidation).AddValidationField(validationDataString)

	e.RoundData.Block = block
	e.RoundData.BlockHash = *block.Hash()
	e.RoundData.BlockValidateData = validationData

	blockData, _ := json.Marshal(e.RoundData.Block)
	msg, _ := MakeBFTProposeMsg(blockData, e.ChainKey, e.UserKeySet)
	// e.logger.Info("push block", time.Since(time1).Seconds())
	go e.Node.PushMessageToChain(msg, e.Chain)
	e.enterVotePhase()
}

func (e *BLSBFT) enterListenPhase() {
	if !e.isInTimeFrame() || e.RoundData.State == listenPhase {
		return
	}
	e.setState(listenPhase)
}

func (e *BLSBFT) enterVotePhase() {
	e.logger.Info("enter voting phase")
	if !e.isInTimeFrame() || e.RoundData.State == votePhase {
		return
	}
	e.isOngoing = true
	e.setState(votePhase)
	err := e.sendVote()
	if err != nil {
		e.logger.Error(err)
	}
}

func (e *BLSBFT) enterNewRound() {
	//if chain is not ready,  return
	if !e.Chain.IsReady() {
		e.RoundData.State = ""
		return
	}
	//if already running a round for current timeframe
	if e.isInTimeFrame() && (e.RoundData.State != newround && e.RoundData.State != "") {
		fmt.Println("CONSENSUS", e.isInTimeFrame(), e.getCurrentRound(), e.getTimeSinceLastBlock().Seconds(), e.RoundData.State)
		return
	}

	e.isOngoing = false
	e.setState("")
	if e.waitForNextRound() {
		return
	}
	e.setState(newround)
	e.InitRoundData()
	e.logger.Info("")
	e.logger.Info("============================================")
	e.logger.Info("")
	pubKey := e.UserKeySet.GetPublicKey()
	if e.Chain.GetPubKeyCommitteeIndex(pubKey.GetMiningKeyBase58(consensusName)) == GetProposerIndexByRound(e.Chain.GetLastProposerIndex(), e.RoundData.Round, e.Chain.GetCommitteeSize()) {
		e.logger.Info("BFT: new round => PROPOSE", e.RoundData.NextHeight, e.RoundData.Round)
		e.enterProposePhase()
	} else {
		e.logger.Info("BFT: new round => LISTEN", e.RoundData.NextHeight, e.RoundData.Round)
		e.enterListenPhase()
	}

}

func (e *BLSBFT) addVote(voteMsg BFTVote) {
	e.RoundData.lockVotes.Lock()
	defer e.RoundData.lockVotes.Unlock()
	e.RoundData.Votes[voteMsg.Validator] = voteMsg.Vote
	e.logger.Warn("vote added...")
	return
}

func (e *BLSBFT) addEarlyVote(voteMsg BFTVote) {
	e.lockEarlyVotes.Lock()
	defer e.lockEarlyVotes.Unlock()
	if _, ok := e.EarlyVotes[voteMsg.RoundKey]; !ok {
		e.EarlyVotes[voteMsg.RoundKey] = make(map[string]vote)
	}
	e.EarlyVotes[voteMsg.RoundKey][voteMsg.Validator] = voteMsg.Vote
	return
}

func (e *BLSBFT) createNewBlock() (types.BlockInterface, error) {

	var errCh chan error
	var block types.BlockInterface = nil
	errCh = make(chan error)
	timeout := time.NewTimer(timeout / 2).C

	go func() {
		time1 := time.Now()
		var err error
		commitee := e.Chain.GetCommittee()
		pk := e.UserKeySet.GetPublicKey()
		base58Str, err := commitee[e.Chain.GetPubKeyCommitteeIndex(pk.GetMiningKeyBase58(consensusName))].ToBase58()
		if err != nil {
			e.logger.Error("UserKeySet is wrong", err)
			errCh <- err
			return
		}

		block, err = e.Chain.CreateNewBlock(1, base58Str, int(e.RoundData.Round), e.RoundData.TimeStart.Unix(), []incognitokey.CommitteePublicKey{}, common.Hash{})
		if block != nil {
			e.logger.Info("create block", block.GetHeight(), time.Since(time1).Seconds())
		} else {
			e.logger.Info("create block", time.Since(time1).Seconds())
		}

		time.AfterFunc(100*time.Millisecond, func() {
			select {
			case <-errCh:
			default:
			}
		})
		errCh <- err
	}()
	select {
	case err := <-errCh:
		return block, err
	case <-timeout:
		if block != nil {
			e.logger.Info("Create block has something wrong ", block.GetHeight())
		}
		return nil, NewConsensusError(BlockCreationError, errors.New("block creation timeout"))
	}
}

func NewInstance(chain ChainInterface, chainKey string, chainID int, node NodeInterface, logger common.Logger) *BLSBFT {
	var newInstance BLSBFT
	newInstance.Chain = chain
	newInstance.ChainKey = chainKey
	newInstance.ChainID = chainID
	newInstance.Node = node
	newInstance.logger = logger
	return &newInstance
}

//func init() {
//	consensus.RegisterConsensus(common.BlsConsensus, &BLSBFT{})
//}
