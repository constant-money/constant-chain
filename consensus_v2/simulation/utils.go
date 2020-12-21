package main

import (
	"bytes"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
)

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewEmptyBlock() common.BlockInterface {
	return &blockchain.ShardBlock{}
}

func NewBlock(height uint64, time int64, producer string, prev common.Hash) common.BlockInterface {
	return &blockchain.ShardBlock{
		Header: blockchain.ShardHeader{
			Version:           1,
			Height:            height,
			Round:             1,
			Epoch:             1,
			Timestamp:         time,
			PreviousBlockHash: prev,
			Producer:          producer,
			ProposeTime:       time,
			Proposer:          producer,
		},
		Body: blockchain.ShardBody{},
	}
}

func GetIndexOfBytes(b []byte, arr [][]byte) int {
	for i, item := range arr {
		if bytes.Equal(b, item) {
			return i
		}
	}
	return -1
}
