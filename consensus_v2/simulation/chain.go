package main

import (
	"encoding/json"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/multiview"
)

type Chain struct {
	multiview *multiview.MultiView
	chainID   int
	chainName string
}

func NewChain(chainID int, chainName string, committee []incognitokey.CommitteePublicKey) *Chain {
	c := new(Chain)
	c.chainID = chainID
	c.chainName = chainName
	c.multiview = multiview.NewMultiView()
	state := &State{
		NewBlock(1, 1, "Genesis", common.Hash{}),
		committee,
	}
	c.multiview.AddView(state)
	return c
}

func (c *Chain) GetFinalView() multiview.View {
	return c.multiview.GetFinalView()
}

func (c *Chain) GetBestView() multiview.View {
	return c.multiview.GetBestView()
}

func (c *Chain) GetChainName() string {
	return c.chainName
}

func (s *Chain) IsReady() bool {
	return true
}

func (s *Chain) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	blk := &blockchain.ShardBlock{}
	json.Unmarshal(blockString, blk)
	return blk, nil
}

func (c *Chain) CreateNewBlock(version int, proposer string, round int, startTime int64) (common.BlockInterface, error) {
	newBlock := NewBlock(c.GetBestView().GetHeight()+1, time.Now().Unix(), proposer, *c.GetBestView().GetHash())
	return newBlock, nil
}

func (c *Chain) CreateNewBlockFromOldBlock(oldBlock common.BlockInterface, proposer string, startTime int64) (common.BlockInterface, error) {
	//TODO: must using the old block data, and timestamp
	oldBlock.(*blockchain.ShardBlock).Header.Proposer = proposer
	oldBlock.(*blockchain.ShardBlock).Header.ProposeTime = time.Now().Unix()
	// newBlock := NewBlock(c.GetBestView().GetHeight()+1, oldBlock.GetProduceTime(), proposer, *c.GetBestView().GetHash())
	return oldBlock, nil
}

func (s *Chain) InsertAndBroadcastBlock(block common.BlockInterface) error {
	state := &State{
		block,
		s.multiview.GetBestView().GetCommittee(),
	}
	s.multiview.AddView(state)
	return nil
}

func (s *Chain) ValidatePreSignBlock(block common.BlockInterface) error {
	return nil
}

func (s *Chain) GetShardID() int {
	return s.chainID
}

func (c Chain) GetViewByHash(hash common.Hash) multiview.View {
	return c.multiview.GetViewByHash(hash)
}
