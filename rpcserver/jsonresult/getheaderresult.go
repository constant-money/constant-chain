package jsonresult

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
)

type GetHeaderResult struct {
	BlockNum  int               `json:"Blocknum"`
	ShardID   byte              `json:"ShardID"`
	BlockHash string            `json:"Blockhash"`
	Header    types.ShardHeader `json:"Header"`
}

func NewHeaderResult(header types.ShardHeader, blockNum int, blockHash string, shardID byte) GetHeaderResult {
	result := GetHeaderResult{}
	result.Header = header
	result.BlockNum = blockNum
	result.BlockHash = blockHash
	result.ShardID = shardID
	return result
}
