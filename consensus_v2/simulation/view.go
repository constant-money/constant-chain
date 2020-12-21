package main

import (
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type State struct {
	block     common.BlockInterface
	committee []incognitokey.CommitteePublicKey
}

func (s *State) GetHash() *common.Hash {
	return s.block.Hash()
}

func (s *State) GetPreviousHash() *common.Hash {
	hash := s.block.GetPrevHash()
	return &hash
}

func (s *State) GetHeight() uint64 {
	return s.block.GetHeight()
}

func (s *State) GetCommittee() []incognitokey.CommitteePublicKey {
	return s.committee
}

func (s *State) GetProposerByTimeSlot(ts int64, version int) incognitokey.CommitteePublicKey {
	id := blockchain.GetProposerByTimeSlot(ts, len(s.committee))
	return s.committee[id]
}

func (s *State) GetBlock() common.BlockInterface {
	return s.block
}
