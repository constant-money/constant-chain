package blsbft

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/wire"
)

const (
	MSG_PROPOSE = "propose"
	MSG_VOTE    = "vote"
)

type BFTPropose struct {
	Block json.RawMessage
}

type BFTVote struct {
	RoundKey  string
	Validator string
	Vote      vote
}

func MakeBFTProposeMsg(block []byte, chainKey string, userKeySet *signatureschemes.MiningKey) (wire.Message, error) {
	var proposeCtn BFTPropose
	proposeCtn.Block = block
	proposeCtnBytes, err := json.Marshal(proposeCtn)
	if err != nil {
		return nil, NewConsensusError(UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = proposeCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_PROPOSE
	return msg, nil
}

func MakeBFTVoteMsg(userPublicKey string, chainKey, roundKey string, vote vote) (wire.Message, error) {
	var voteCtn BFTVote
	voteCtn.RoundKey = roundKey
	voteCtn.Validator = userPublicKey
	voteCtn.Vote = vote
	voteCtnBytes, err := json.Marshal(voteCtn)
	if err != nil {
		return nil, NewConsensusError(UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = voteCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_VOTE
	return msg, nil
}

//TODO merman
func (e *BLSBFT) ProcessBFTMsg(msg *wire.MessageBFT) {
	switch msg.Type {
	case MSG_PROPOSE:
		var msgPropose BFTPropose
		err := json.Unmarshal(msg.Content, &msgPropose)
		if err != nil {
			fmt.Println(err)
			return
		}
		e.ProposeMessageCh <- msgPropose
	case MSG_VOTE:
		var msgVote BFTVote
		err := json.Unmarshal(msg.Content, &msgVote)
		if err != nil {
			fmt.Println(err)
			return
		}
		e.VoteMessageCh <- msgVote
	default:
		e.logger.Critical("???")
		return
	}
}

func (e *BLSBFT) confirmVote(Vote *vote, userKey signatureschemes.MiningKey) error {
	data := e.RoundData.Block.Hash().GetBytes()
	data = append(data, Vote.BLS...)
	data = append(data, Vote.BRI...)
	data = common.HashB(data)
	var err error
	Vote.Confirmation, err = userKey.BriSignData(data)
	return err
}

func (e *BLSBFT) preValidateVote(blockHash []byte, Vote *vote, candidate []byte) error {
	data := []byte{}
	data = append(data, blockHash...)
	data = append(data, Vote.BLS...)
	data = append(data, Vote.BRI...)
	dataHash := common.HashH(data)
	err := validateSingleBriSig(&dataHash, Vote.Confirmation, candidate)
	return err
}

func (e *BLSBFT) sendVote() error {
	var Vote vote
	for _, userKey := range e.UserKeySet {
		pubKey := userKey.GetPublicKey()
		if common.IndexOfStr(pubKey.GetMiningKeyBase58(consensusName), e.RoundData.CommitteeBLS.StringList) != -1 {
			selfIdx := common.IndexOfStr(pubKey.GetMiningKeyBase58(consensusName), e.RoundData.CommitteeBLS.StringList)
			blsSig, err := userKey.BLSSignData(e.RoundData.Block.Hash().GetBytes(), selfIdx, e.RoundData.CommitteeBLS.ByteList)
			if err != nil {
				return NewConsensusError(UnExpectedError, err)
			}
			bridgeSig := []byte{}
			if metadata.HasBridgeInstructions(e.RoundData.Block.GetInstructions()) {
				bridgeSig, err = userKey.BriSignData(e.RoundData.Block.Hash().GetBytes())
				if err != nil {
					return NewConsensusError(UnExpectedError, err)
				}
			}

			Vote.BLS = blsSig
			Vote.BRI = bridgeSig

			//TODO hy
			err = e.confirmVote(&Vote, userKey)
			if err != nil {
				return NewConsensusError(UnExpectedError, err)
			}
			key := userKey.GetPublicKey()

			msg, err := MakeBFTVoteMsg(key.GetMiningKeyBase58(consensusName), e.ChainKey, getRoundKey(e.RoundData.NextHeight, e.RoundData.Round), Vote)
			if err != nil {
				return NewConsensusError(UnExpectedError, err)
			}
			e.RoundData.Votes[pubKey.GetMiningKeyBase58(consensusName)] = Vote
			e.logger.Info("sending vote...", getRoundKey(e.RoundData.NextHeight, e.RoundData.Round))
			go e.Node.PushMessageToChain(msg, e.Chain)
		}
	}

	e.RoundData.NotYetSendVote = false
	return nil
}
