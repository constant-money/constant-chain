package blsbftv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/incognitochain/incognito-chain/multiview"

	signatureschemes2 "github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"

	lru "github.com/hashicorp/golang-lru"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/wire"
)

type BLSBFT_V2 struct {
	Chain    ChainInterface
	Node     NodeInterface
	ChainKey string
	ChainID  int
	PeerID   string

	UserKeySet   []signatureschemes2.MiningKey
	BFTMessageCh chan wire.MessageBFT
	isStarted    bool
	destroyCh    chan struct{}
	Logger       common.Logger

	currentTime      int64
	currentTimeSlot  int64
	proposeHistory   *lru.Cache
	ProposeMessageCh chan BFTPropose
	VoteMessageCh    chan BFTVote

	receiveBlockByHeight map[uint64][]*ProposeBlockInfo   //blockHeight -> blockInfo
	receiveBlockByHash   map[string]*ProposeBlockInfo     //blockHash -> blockInfo
	voteHistory          map[uint64]common.BlockInterface // bestview height (previsous height )-> block
}

func (e BLSBFT_V2) GetChainKey() string {
	return e.ChainKey
}

func (e BLSBFT_V2) GetChainID() int {
	return e.ChainID
}

func (e BLSBFT_V2) IsOngoing() bool {
	return e.isStarted
}

func (e BLSBFT_V2) IsStarted() bool {
	return e.isStarted
}

type ProposeBlockInfo struct {
	receiveTime time.Time
	block       common.BlockInterface
	votes       map[string]*BFTVote //pk->BFTVote
	isValid     bool
	hasNewVote  bool
	sendVote    bool
}

func (e *BLSBFT_V2) GetConsensusName() string {
	return common.BlsConsensus
}

func (e *BLSBFT_V2) Stop() error {
	if e.isStarted {
		e.Logger.Info("stop bls-bftv2 consensus for chain", e.ChainKey)
	}
	e.isStarted = false
	return nil
}

func (e *BLSBFT_V2) Start() error {
	if !e.isStarted {
		e.Logger.Info("start bls-bftv2 consensus for chain", e.ChainKey)
	}
	e.isStarted = true
	return nil
}

func (e *BLSBFT_V2) Destroy() {
	close(e.destroyCh)
}

//only run when init process
func (e *BLSBFT_V2) run() error {
	go func() {
		//init view maps
		ticker := time.Tick(200 * time.Millisecond)
		cleanMemTicker := time.Tick(5 * time.Minute)
		e.Logger.Info("init bls-bftv2 consensus for chain", e.ChainKey)

		for { //actor loop
			if !e.isStarted { //sleep if this process is not start
				time.Sleep(time.Second)
				continue
			}
			select {
			case <-e.destroyCh:
				e.Logger.Info("exit bls-bftv2 consensus for chain", e.ChainKey)
				return
			case proposeMsg := <-e.ProposeMessageCh:
				//fmt.Println("debug receive propose message", string(proposeMsg.Block))
				blockIntf, err := e.Chain.UnmarshalBlock(proposeMsg.Block)
				if err != nil || blockIntf == nil {
					e.Logger.Info(err)
					continue
				}
				block := blockIntf.(common.BlockInterface)
				blkHash := block.Hash().String()

				if _, ok := e.receiveBlockByHash[blkHash]; !ok {
					e.receiveBlockByHash[blkHash] = &ProposeBlockInfo{
						block:       block,
						votes:       make(map[string]*BFTVote),
						hasNewVote:  false,
						receiveTime: time.Now(),
					}
					e.Logger.Info(e.ChainKey, "Receive block ", block.Hash().String(), "height", block.GetHeight(), ",block timeslot ", common.CalculateTimeSlot(block.GetProposeTime()))
					e.receiveBlockByHeight[block.GetHeight()] = append(e.receiveBlockByHeight[block.GetHeight()], e.receiveBlockByHash[blkHash])
				} else {
					e.receiveBlockByHash[blkHash].block = block
				}

				if block.GetHeight() <= e.Chain.GetBestView().GetHeight() {
					e.Logger.Infof("%v Receive block create from old view - height %v. Rejected! Expect: %v", e.ChainKey, block.GetHeight(), e.Chain.GetBestView().GetHeight())
					continue
				}

				proposeView := e.Chain.GetViewByHash(block.GetPrevHash())
				if proposeView == nil {
					e.Logger.Infof("%v Request sync block from node %s from %s to %s", e.ChainKey, proposeMsg.PeerID, block.GetPrevHash().String(), block.GetPrevHash().String())
					e.Node.RequestMissingViewViaStream(proposeMsg.PeerID, [][]byte{block.GetPrevHash().Bytes()}, e.Chain.GetShardID(), e.Chain.GetChainName())
				}

			case voteMsg := <-e.VoteMessageCh:
				voteMsg.IsValid = 0
				if b, ok := e.receiveBlockByHash[voteMsg.BlockHash]; ok { //if receiveblock is already initiated
					if _, ok := b.votes[voteMsg.Validator]; !ok { // and not receive validatorA vote
						b.votes[voteMsg.Validator] = &voteMsg // store it
						vid, v := GetValidatorIndex(e.Chain.GetBestView(), voteMsg.Validator)
						if v != nil {
							vbase58, _ := v.ToBase58()
							e.Logger.Infof("%v Receive vote (%d) for block %s from validator %d %v", e.ChainKey, len(e.receiveBlockByHash[voteMsg.BlockHash].votes), voteMsg.BlockHash, vid, vbase58)
						} else {
							e.Logger.Infof("%v Receive vote (%d) for block from unknown validator", e.ChainKey, len(e.receiveBlockByHash[voteMsg.BlockHash].votes), voteMsg.BlockHash, voteMsg.Validator)
						}

						b.hasNewVote = true
					}
				} else {
					e.receiveBlockByHash[voteMsg.BlockHash] = &ProposeBlockInfo{
						votes:      make(map[string]*BFTVote),
						hasNewVote: true,
					}
					e.receiveBlockByHash[voteMsg.BlockHash].votes[voteMsg.Validator] = &voteMsg
					vid, v := GetValidatorIndex(e.Chain.GetBestView(), voteMsg.Validator)
					if v != nil {
						vbase58, _ := v.ToBase58()
						e.Logger.Infof("%v Receive vote (%d) for block %s from validator %d %v", e.ChainKey, len(e.receiveBlockByHash[voteMsg.BlockHash].votes), voteMsg.BlockHash, vid, vbase58)
					} else {
						e.Logger.Infof("%v Receive vote (%d) for block from unknown validator", e.ChainKey, len(e.receiveBlockByHash[voteMsg.BlockHash].votes), voteMsg.BlockHash, voteMsg.Validator)
					}
				}
			case <-cleanMemTicker:
				for h, _ := range e.receiveBlockByHeight {
					if h <= e.Chain.GetFinalView().GetHeight() {
						delete(e.receiveBlockByHeight, h)
					}
				}
				for h, _ := range e.voteHistory {
					if h <= e.Chain.GetFinalView().GetHeight() {
						delete(e.voteHistory, h)
					}
				}
				for h, proposeBlk := range e.receiveBlockByHash {
					if time.Now().Sub(proposeBlk.receiveTime) > time.Minute {
						delete(e.receiveBlockByHash, h)
					}
				}
			case <-ticker:
				if !e.Chain.IsReady() {
					continue
				}
				e.currentTime = time.Now().Unix()

				newTimeSlot := false
				if e.currentTimeSlot != common.CalculateTimeSlot(e.currentTime) {
					newTimeSlot = true
				}

				e.currentTimeSlot = common.CalculateTimeSlot(e.currentTime)
				bestView := e.Chain.GetBestView()

				/*
					Check for whether we should propose block
				*/
				proposerPk, _ := bestView.GetProposerByTimeSlot(e.currentTimeSlot, 2)
				var userProposeKey signatureschemes2.MiningKey
				shouldPropose := false
				shouldListen := true
				for _, userKey := range e.UserKeySet {
					userPk := userKey.GetPublicKey().GetMiningKeyBase58(common.BlsConsensus)
					if proposerPk.GetMiningKeyBase58(common.BlsConsensus) == userPk {
						shouldListen = false
						if common.CalculateTimeSlot(bestView.GetBlock().GetProposeTime()) != e.currentTimeSlot { // current timeslot is not add to view, and this user is proposer of this timeslot
							//using block hash as key of best view -> check if this best view we propose or not
							if _, ok := e.proposeHistory.Get(fmt.Sprintf("%s%d", e.currentTimeSlot)); !ok {
								shouldPropose = true
								userProposeKey = userKey
							}
						}
					}
				}

				if newTimeSlot { //for logging
					e.Logger.Infof("%v", e.ChainKey)
					e.Logger.Infof("%v ======================================================", e.ChainKey)
					e.Logger.Infof("%v", e.ChainKey)
					if shouldListen {
						e.Logger.Infof("%v TS: %v, LISTEN BLOCK %v, Round %v", e.ChainKey, common.CalculateTimeSlot(e.currentTime), bestView.GetHeight()+1, e.currentTimeSlot-common.CalculateTimeSlot(bestView.GetBlock().GetProposeTime()))
					}
					if shouldPropose {
						e.Logger.Infof("%v TS: %v, PROPOSE BLOCK %v, Round %v", e.ChainKey, common.CalculateTimeSlot(e.currentTime), bestView.GetHeight()+1, e.currentTimeSlot-common.CalculateTimeSlot(bestView.GetBlock().GetProposeTime()))
					}

				}

				if shouldPropose {
					e.proposeHistory.Add(fmt.Sprintf("%s%d", e.currentTimeSlot), 1)
					//Proposer Rule: check propose block connected to bestview(longest chain rule 1) and re-propose valid block with smallest timestamp (including already propose in the past) (rule 2)
					sort.Slice(e.receiveBlockByHeight[bestView.GetHeight()+1], func(i, j int) bool {
						return e.receiveBlockByHeight[bestView.GetHeight()+1][i].block.GetProduceTime() < e.receiveBlockByHeight[bestView.GetHeight()+1][j].block.GetProduceTime()
					})

					var proposeBlock common.BlockInterface = nil
					for _, v := range e.receiveBlockByHeight[bestView.GetHeight()+1] {
						if v.isValid {
							proposeBlock = v.block
							break
						}
					}

					//proposerPk: which include mining pubkey + incokey
					//userKey: only have minigkey
					if createdBlk, err := e.proposeBlock(userProposeKey, proposerPk, proposeBlock); err != nil {
						e.Logger.Critical(UnExpectedError, errors.New("can't propose block"))
						e.Logger.Critical(err)

					} else {
						e.Logger.Infof("%v proposer block %v round %v time slot %v blockTimeSlot %v with hash %v", e.ChainKey, createdBlk.GetHeight(), e.currentTimeSlot-common.CalculateTimeSlot(bestView.GetBlock().GetProposeTime()), e.currentTimeSlot, common.CalculateTimeSlot(createdBlk.GetProduceTime()), createdBlk.Hash().String())
					}
				}

				/*
					Check for valid block to vote
				*/
				validProposeBlock := []*ProposeBlockInfo{}
				//get all block that has height = bestview height  + 1(rule 2 & rule 3) (
				for h, proposeBlockInfo := range e.receiveBlockByHash {
					if proposeBlockInfo.block == nil {
						continue
					}
					// e.Logger.Infof("[Monitor] bestview height %v, finalview height %v, block height %v %v", bestViewHeight, e.Chain.GetFinalView().GetHeight(), proposeBlockInfo.block.GetHeight(), proposeBlockInfo.block.GetProduceTime())
					// check if propose block in current time
					// if e.currentTimeSlot == common.CalculateTimeSlot(proposeBlockInfo.block.GetProposeTime()) {
					validProposeBlock = append(validProposeBlock, proposeBlockInfo)
					// }

					if proposeBlockInfo.block.GetHeight() < e.Chain.GetFinalView().GetHeight() {
						delete(e.receiveBlockByHash, h)
					}
				}
				//rule 1: get history of vote for this height, vote if (round is lower than the vote before) or (round is equal but new proposer) or (there is no vote for this height yet)
				sort.Slice(validProposeBlock, func(i, j int) bool {
					return validProposeBlock[i].block.GetProduceTime() < validProposeBlock[j].block.GetProduceTime()
				})

				for _, v := range validProposeBlock {
					if v.sendVote {
						continue
					}

					blkCreateTimeSlot := common.CalculateTimeSlot(v.block.GetProduceTime())
					bestViewHeight := bestView.GetHeight()

					if lastVotedBlk, ok := e.voteHistory[bestViewHeight+1]; ok {
						if blkCreateTimeSlot < common.CalculateTimeSlot(lastVotedBlk.GetProduceTime()) { //blkCreateTimeSlot is smaller than voted block => vote for this blk
							e.validateAndVote(v)
						} else if blkCreateTimeSlot == common.CalculateTimeSlot(lastVotedBlk.GetProduceTime()) && common.CalculateTimeSlot(v.block.GetProposeTime()) > common.CalculateTimeSlot(lastVotedBlk.GetProposeTime()) { //blk is old block (same round), but new proposer(larger timeslot) => vote again
							e.validateAndVote(v)
						} //blkCreateTimeSlot is larger or equal than voted block => do nothing
					} else { //there is no vote for this height yet
						e.validateAndVote(v)
					}
				}

				/*
					Check for 2/3 vote to commit
				*/
				for k, v := range e.receiveBlockByHash {
					e.processIfBlockGetEnoughVote(k, v)
				}

			}
		}
	}()
	return nil
}

func NewInstance(chain ChainInterface, chainKey string, chainID int, node NodeInterface, logger common.Logger) *BLSBFT_V2 {
	var err error
	var newInstance = new(BLSBFT_V2)
	newInstance.Chain = chain
	newInstance.ChainKey = chainKey
	newInstance.ChainID = chainID
	newInstance.Node = node
	newInstance.Logger = logger
	newInstance.destroyCh = make(chan struct{})
	newInstance.ProposeMessageCh = make(chan BFTPropose)
	newInstance.VoteMessageCh = make(chan BFTVote)
	newInstance.receiveBlockByHash = make(map[string]*ProposeBlockInfo)
	newInstance.receiveBlockByHeight = make(map[uint64][]*ProposeBlockInfo)
	newInstance.voteHistory = make(map[uint64]common.BlockInterface)
	newInstance.proposeHistory, err = lru.New(1000)
	if err != nil {
		panic(err) //must not error
	}
	newInstance.run()
	return newInstance
}

func GetValidatorIndex(view multiview.View, validator string) (int, *incognitokey.CommitteePublicKey) {
	for id, c := range view.GetCommittee() {
		if validator == c.GetMiningKeyBase58(common.BlsConsensus) {
			return id, &c
		}
	}
	return -1, nil
}

func (e *BLSBFT_V2) processIfBlockGetEnoughVote(blockHash string, v *ProposeBlockInfo) {
	//no vote
	if v.hasNewVote == false {
		return
	}

	//no block
	if v.block == nil {
		return
	}

	//already in chain
	view := e.Chain.GetViewByHash(*v.block.Hash())
	if view != nil {
		return
	}

	//not connected previous block
	view = e.Chain.GetViewByHash(v.block.GetPrevHash())
	if view == nil {
		return
	}

	validVote := 0
	errVote := 0
	for id, vote := range v.votes {
		dsaKey := []byte{}
		if vote.IsValid == 0 {
			cid, committeePk := GetValidatorIndex(view, vote.Validator)
			if committeePk != nil {
				dsaKey = committeePk.MiningPubKey[common.BridgeConsensus]
			} else {
				e.Logger.Error("Receive vote from nonCommittee member")
				continue
			}

			if len(dsaKey) == 0 {
				e.Logger.Error(fmt.Sprintf("Cannot find dsa key from vote of %d", cid))
				continue
			}

			err := vote.validateVoteOwner(dsaKey)
			if err != nil {
				e.Logger.Error(dsaKey)
				e.Logger.Error(err)
				panic(1)
				v.votes[id].IsValid = -1
				errVote++
			} else {
				v.votes[id].IsValid = 1
				validVote++
			}
		} else {
			validVote++
		}
	}
	//e.Logger.Debug(validVote, len(view.GetCommittee()), errVote)
	v.hasNewVote = false
	if validVote > 2*len(view.GetCommittee())/3 {
		e.Logger.Infof("%v Commit block %v , height: %v", e.ChainKey, blockHash, v.block.GetHeight())
		committeeBLSString, err := incognitokey.ExtractPublickeysFromCommitteeKeyList(view.GetCommittee(), common.BlsConsensus)
		// fmt.Println(committeeBLSString)
		if err != nil {
			e.Logger.Error(err)
			return
		}
		aggSig, brigSigs, validatorIdx, err := CombineVotes(v.votes, committeeBLSString)
		committeeBLSKeys := []blsmultisig.PublicKey{}
		for _, member := range view.GetCommittee() {
			committeeBLSKeys = append(committeeBLSKeys, member.MiningPubKey[common.BlsConsensus])
		}
		err2 := validateBLSSig(v.block.Hash(), aggSig, validatorIdx, committeeBLSKeys)
		e.Logger.Infof("[debugsig] Committee list %v, validatorIdx %v", committeeBLSString, validatorIdx)
		if err2 != nil {
			panic(err2)
		}

		if err != nil {
			e.Logger.Error(err)
			return
		}

		valData, err := DecodeValidationData(v.block.GetValidationField())
		if err != nil {
			e.Logger.Error(err)
			return
		}

		valData.AggSig = aggSig
		valData.BridgeSig = brigSigs
		valData.ValidatiorsIdx = validatorIdx
		validationDataString, _ := EncodeValidationData(*valData)
		e.Logger.Infof("%v Validation Data %v %v %v %v", e.ChainKey, aggSig, brigSigs, validatorIdx, validationDataString)
		if err := v.block.(blockValidation).AddValidationField(validationDataString); err != nil {
			e.Logger.Error(err)
			return
		}

		go e.Chain.InsertAndBroadcastBlock(v.block)
	}
}

func (e *BLSBFT_V2) validateAndVote(v *ProposeBlockInfo) error {
	//not connected
	e.Logger.Info(e.ChainKey, "validateAndVote")
	view := e.Chain.GetViewByHash(v.block.GetPrevHash())
	if view == nil {
		e.Logger.Error(e.ChainKey, "view is null")
		return errors.New("View not connect")
	}

	//TODO: using context to validate block
	_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := e.Chain.ValidatePreSignBlock(v.block); err != nil {
		e.Logger.Error(err)
		return err
	}

	//if valid then vote
	committeeBLSString, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(view.GetCommittee(), common.BlsConsensus)
	for _, userKey := range e.UserKeySet {
		pubKey := userKey.GetPublicKey()
		if common.IndexOfStr(pubKey.GetMiningKeyBase58(e.GetConsensusName()), committeeBLSString) != -1 {
			Vote, err := CreateVote(&userKey, v.block, e.Chain.GetBestView().GetCommittee())
			if err != nil {
				e.Logger.Error(err)
				return NewConsensusError(UnExpectedError, err)
			}

			msg, err := MakeBFTVoteMsg(Vote, e.ChainKey, e.currentTimeSlot, v.block.GetHeight())
			if err != nil {
				e.Logger.Error(err)
				return NewConsensusError(UnExpectedError, err)
			}

			v.isValid = true
			blkHeight := v.block.GetHeight()
			blkHash := v.block.Hash()
			// preHash := v.block.GetPrevHash()blkHash := v.block.Hash()
			e.voteHistory[v.block.GetHeight()] = v.block
			x := msg.(*wire.MessageBFT)
			var xVote BFTVote
			err = json.Unmarshal(x.Content, &xVote)
			if err != nil {
				e.Logger.Error(err)
			}
			idx := common.IndexOfStr(xVote.Validator, committeeBLSString)
			e.Logger.Infof("%v sending vote for block %v %v TS %v signature %v %v %v...", blkHeight, blkHash, xVote.TimeSlot, xVote.BLS, xVote.BRI, xVote.Validator, idx)
			v.sendVote = true
			go e.ProcessBFTMsg(msg.(*wire.MessageBFT))
			go e.Node.PushMessageToChain(msg, e.Chain)
		}
	}

	return nil
}

func CreateVote(userKey *signatureschemes2.MiningKey, block common.BlockInterface, committees []incognitokey.CommitteePublicKey) (*BFTVote, error) {
	var Vote = new(BFTVote)
	bytelist := []blsmultisig.PublicKey{}
	selfIdx := 0
	userBLSPk := userKey.GetPublicKey().GetMiningKeyBase58(common.BlsConsensus)
	for i, v := range committees {
		if v.GetMiningKeyBase58(common.BlsConsensus) == userBLSPk {
			selfIdx = i
		}
		bytelist = append(bytelist, v.MiningPubKey[common.BlsConsensus])
	}

	blsSig, err := userKey.BLSSignData(block.Hash().GetBytes(), selfIdx, bytelist)
	if err != nil {

		return nil, NewConsensusError(UnExpectedError, err)
	}
	bridgeSig := []byte{}
	if metadata.HasBridgeInstructions(block.GetInstructions()) || metadata.HasPortalInstructions(block.GetInstructions()) {
		bridgeSig, err = userKey.BriSignData(block.Hash().GetBytes())
		if err != nil {
			return nil, NewConsensusError(UnExpectedError, err)
		}
	}
	Vote.BLS = blsSig
	Vote.BRI = bridgeSig
	Vote.BlockHash = block.Hash().String()

	userPk := userKey.GetPublicKey()
	Vote.Validator = userPk.GetMiningKeyBase58(common.BlsConsensus)
	Vote.PrevBlockHash = block.GetPrevHash().String()
	err = Vote.signVote(userKey)
	if err != nil {
		return nil, NewConsensusError(UnExpectedError, err)
	}
	return Vote, nil
}

func (e *BLSBFT_V2) proposeBlock(userMiningKey signatureschemes2.MiningKey, proposerPk incognitokey.CommitteePublicKey, block common.BlockInterface) (common.BlockInterface, error) {
	time1 := time.Now()
	b58Str, _ := proposerPk.ToBase58()
	var err error
	if block == nil {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, (time.Duration(common.TIMESLOT)*time.Second)/2)
		defer cancel()
		//block, _ = e.Chain.CreateNewBlock(ctx, e.currentTimeSlot, e.UserKeySet.GetPublicKeyBase58())
		//e.Logger.Info("debug CreateNewBlock")
		block, err = e.Chain.CreateNewBlock(2, b58Str, 1, e.currentTime)
	} else {
		//e.Logger.Info("debug CreateNewBlockFromOldBlock")
		block, err = e.Chain.CreateNewBlockFromOldBlock(block, b58Str, e.currentTime)
		//b58Str, _ := proposerPk.ToBase58()
		//block = e.voteHistory[e.Chain.GetBestViewHeight()+1]
	}
	if err != nil {
		return nil, NewConsensusError(BlockCreationError, err)
	}

	if block != nil {
		e.Logger.Infof("%v create block %v hash %v, propose time %v, produce time %v", e.ChainKey, block.GetHeight(), block.Hash().String(), block.(common.BlockInterface).GetProposeTime(), block.(common.BlockInterface).GetProduceTime())
	} else {
		e.Logger.Infof("%v create block fail, time: %v", e.ChainKey, time.Since(time1).Seconds())
		return nil, NewConsensusError(BlockCreationError, errors.New("block is nil"))
	}

	var validationData ValidationData
	validationData.ProducerBLSSig, _ = userMiningKey.BriSignData(block.Hash().GetBytes())
	validationDataString, _ := EncodeValidationData(validationData)
	block.(blockValidation).AddValidationField(validationDataString)
	blockData, _ := json.Marshal(block)
	var proposeCtn = new(BFTPropose)
	proposeCtn.Block = blockData
	pKey := userMiningKey.GetPublicKey()
	proposeCtn.PeerID = pKey.GetMiningKeyBase58("bls")
	msg, _ := MakeBFTProposeMsg(proposeCtn, e.ChainKey, e.currentTimeSlot, block.GetHeight())
	e.Logger.Infof("[debugbft] BFT msg pubkey %v", msg.(*wire.MessageBFT).PeerID)
	// go e.ProcessBFTMsg(msg.(*wire.MessageBFT))
	go e.Node.PushMessageToChain(msg, e.Chain)

	return block, nil
}

func (e *BLSBFT_V2) ProcessBFTMsg(msgBFT *wire.MessageBFT) {
	switch msgBFT.Type {
	case MSG_PROPOSE:
		var msgPropose BFTPropose
		err := json.Unmarshal(msgBFT.Content, &msgPropose)
		if err != nil {
			e.Logger.Error(err)
			return
		}
		msgPropose.PeerID = msgBFT.PeerID
		e.ProposeMessageCh <- msgPropose
	case MSG_VOTE:
		var msgVote BFTVote
		err := json.Unmarshal(msgBFT.Content, &msgVote)
		if err != nil {
			e.Logger.Error(err)
			return
		}
		e.VoteMessageCh <- msgVote
	default:
		e.Logger.Critical("Unknown BFT message type")
		return
	}
}

func (e *BLSBFT_V2) preValidateVote(blockHash []byte, Vote *BFTVote, candidate []byte) error {
	data := []byte{}
	data = append(data, blockHash...)
	data = append(data, Vote.BLS...)
	data = append(data, Vote.BRI...)
	dataHash := common.HashH(data)
	err := validateSingleBriSig(&dataHash, Vote.Confirmation, candidate)
	return err
}

func (s *BFTVote) signVote(key *signatureschemes2.MiningKey) error {
	data := []byte{}
	data = append(data, s.BlockHash...)
	data = append(data, s.BLS...)
	data = append(data, s.BRI...)
	data = common.HashB(data)
	var err error
	s.Confirmation, err = key.BriSignData(data)
	return err
}

func (s *BFTVote) validateVoteOwner(ownerPk []byte) error {
	data := []byte{}
	data = append(data, s.BlockHash...)
	data = append(data, s.BLS...)
	data = append(data, s.BRI...)
	dataHash := common.HashH(data)
	err := validateSingleBriSig(&dataHash, s.Confirmation, ownerPk)
	return err
}

func ExtractBridgeValidationData(block common.BlockInterface) ([][]byte, []int, error) {
	valData, err := DecodeValidationData(block.GetValidationField())
	if err != nil {
		return nil, nil, NewConsensusError(UnExpectedError, err)
	}
	return valData.BridgeSig, valData.ValidatiorsIdx, nil
}
