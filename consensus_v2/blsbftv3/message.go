package blsbftv3

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/wire"
)

const (
	MSG_PROPOSE    = "propose"
	MSG_VOTE       = "vote"
	MSG_REQUESTBLK = "getblk"
)

type BFTPropose struct {
	PeerID   string
	Block    json.RawMessage
	TimeSlot uint64
}

type BFTVote struct {
	//ViewHash     string
	PrevBlockHash string
	BlockHash     string
	Validator     string
	BLS           []byte
	BRI           []byte
	Confirmation  []byte
	isValid       int // 0 not process, 1 valid, -1 not valid
	TimeSlot      uint64
}

type BFTRequestBlock struct {
	BlockHash string
	PeerID    string
}

func MakeBFTProposeMsg(proposeCtn *BFTPropose, chainKey string, ts int64, height uint64) (wire.Message, error) {
	proposeCtnBytes, err := json.Marshal(proposeCtn)
	if err != nil {
		return nil, NewConsensusError(UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = proposeCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_PROPOSE
	msg.(*wire.MessageBFT).TimeSlot = ts
	msg.(*wire.MessageBFT).Timestamp = int64(height)
	msg.(*wire.MessageBFT).PeerID = proposeCtn.PeerID
	return msg, nil
}

func MakeBFTVoteMsg(vote *BFTVote, chainKey string, ts int64, height uint64) (wire.Message, error) {
	voteCtnBytes, err := json.Marshal(vote)
	if err != nil {
		return nil, NewConsensusError(UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = voteCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_VOTE
	msg.(*wire.MessageBFT).TimeSlot = ts
	msg.(*wire.MessageBFT).Timestamp = int64(height)
	return msg, nil
}

func MakeBFTRequestBlk(request BFTRequestBlock, peerID string, chainKey string) (wire.Message, error) {
	requestCtnBytes, err := json.Marshal(request)
	if err != nil {
		return nil, NewConsensusError(UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = requestCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_REQUESTBLK
	return msg, nil
}
