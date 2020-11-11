package jsonresult

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
)

type GetBeaconBlockResult struct {
	Hash              string      `json:"Hash"`
	Height            uint64      `json:"Height"`
	BlockProducer     string      `json:"BlockProducer"`
	ValidationData    string      `json:"ValidationData"`
	ConsensusType     string      `json:"ConsensusType"`
	Version           int         `json:"Version"`
	Epoch             uint64      `json:"Epoch"`
	Round             int         `json:"Round"`
	Time              int64       `json:"Time"`
	PreviousBlockHash string      `json:"PreviousBlockHash"`
	NextBlockHash     string      `json:"NextBlockHash"`
	Instructions      [][]string  `json:"Instructions"`
	Size              uint64      `json:"Size"`
	ShardStates       interface{} `json:"ShardStates"`
}

type GetShardBlockResult struct {
	Hash               string             `json:"Hash"`
	ShardID            byte               `json:"ShardID"`
	Height             uint64             `json:"Height"`
	Confirmations      int64              `json:"Confirmations"`
	Version            int                `json:"Version"`
	TxRoot             string             `json:"TxRoot"`
	Time               int64              `json:"Time"`
	PreviousBlockHash  string             `json:"PreviousBlockHash"`
	NextBlockHash      string             `json:"NextBlockHash"`
	TxHashes           []string           `json:"TxHashes"`
	Txs                []GetBlockTxResult `json:"Txs"`
	BlockProducer      string             `json:"BlockProducer"`
	ValidationData     string             `json:"ValidationData"`
	ConsensusType      string             `json:"ConsensusType"`
	Data               string             `json:"Data"`
	BeaconHeight       uint64             `json:"BeaconHeight"`
	BeaconBlockHash    string             `json:"BeaconBlockHash"`
	Round              int                `json:"Round"`
	Epoch              uint64             `json:"Epoch"`
	Reward             uint64             `json:"Reward"`
	RewardBeacon       uint64             `json:"RewardBeacon"`
	Fee                uint64             `json:"Fee"`
	Size               uint64             `json:"Size"`
	CommitteeFromBlock common.Hash        `json:"CommitteeFromBlock"`
	Instruction        [][]string         `json:"Instruction"`
	CrossShardBitMap   []int              `json:"CrossShardBitMap"`
}

type GetBlockTxResult struct {
	Hash     string `json:"Hash"`
	Locktime int64  `json:"Locktime"`
	HexData  string `json:"HexData"`
}

func NewGetBlocksBeaconResult(block *types.BeaconBlock, size uint64, nextBlockHash string) *GetBeaconBlockResult {
	getBlockResult := &GetBeaconBlockResult{}
	getBlockResult.Version = block.Header.Version
	getBlockResult.Hash = block.Hash().String()
	getBlockResult.Height = block.Header.Height
	getBlockResult.BlockProducer = block.Header.Producer
	getBlockResult.ValidationData = block.ValidationData
	getBlockResult.ConsensusType = block.Header.ConsensusType
	getBlockResult.Epoch = block.Header.Epoch
	getBlockResult.Round = block.Header.Round
	getBlockResult.Time = block.Header.Timestamp
	getBlockResult.PreviousBlockHash = block.Header.PreviousBlockHash.String()
	getBlockResult.Instructions = block.Body.Instructions
	getBlockResult.Size = size
	getBlockResult.NextBlockHash = nextBlockHash
	getBlockResult.ShardStates = block.Body.ShardState
	return getBlockResult
}

func NewGetBlockResult(block *types.ShardBlock, size uint64, nextBlockHash string) *GetShardBlockResult {
	getBlockResult := &GetShardBlockResult{}
	getBlockResult.BlockProducer = block.Header.Producer
	getBlockResult.ValidationData = block.ValidationData
	getBlockResult.Hash = block.Hash().String()
	getBlockResult.PreviousBlockHash = block.Header.PreviousBlockHash.String()
	getBlockResult.Version = block.Header.Version
	getBlockResult.Height = block.Header.Height
	getBlockResult.Time = block.Header.Timestamp
	getBlockResult.ShardID = block.Header.ShardID
	getBlockResult.TxRoot = block.Header.TxRoot.String()
	getBlockResult.TxHashes = make([]string, 0)
	getBlockResult.Fee = uint64(0)
	getBlockResult.Size = size
	for _, tx := range block.Body.Transactions {
		getBlockResult.TxHashes = append(getBlockResult.TxHashes, tx.Hash().String())
		getBlockResult.Fee += tx.GetTxFee()
	}
	getBlockResult.BeaconHeight = block.Header.BeaconHeight
	getBlockResult.BeaconBlockHash = block.Header.BeaconHash.String()
	getBlockResult.Round = block.Header.Round
	getBlockResult.CrossShardBitMap = []int{}
	if len(block.Header.CrossShardBitMap) > 0 {
		for _, shardID := range block.Header.CrossShardBitMap {
			getBlockResult.CrossShardBitMap = append(getBlockResult.CrossShardBitMap, int(shardID))
		}
	}
	getBlockResult.Epoch = block.Header.Epoch
	if len(block.Body.Transactions) > 0 {
		for _, tx := range block.Body.Transactions {
			if tx.GetMetadataType() == metadata.ShardBlockReward {
				getBlockResult.Reward += tx.GetProof().GetOutputCoins()[0].CoinDetails.GetValue()
			} else if tx.GetMetadataType() == metadata.BeaconSalaryResponseMeta {
				getBlockResult.RewardBeacon += tx.GetProof().GetOutputCoins()[0].CoinDetails.GetValue()
			}
		}
	}
	getBlockResult.NextBlockHash = nextBlockHash
	getBlockResult.CommitteeFromBlock = block.Header.CommitteeFromBlock
	return getBlockResult
}

type GetViewResult struct {
	Hash              string `json:"Hash"`
	Height            uint64 `json:"Height"`
	PreviousBlockHash string `json:"PreviousBlockHash"`
	Round             uint64 `json:"Round"`
}
