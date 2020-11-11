package metadata

import (
	"encoding/json"
	"strconv"

	// "errors"

	"github.com/incognitochain/incognito-chain/common"
)

const (
	SHARD_REWARD_INST = "shardRewardInst"
)

type ShardBlockRewardInfo struct {
	ShardReward map[common.Hash]uint64
	Epoch       uint64
}

type AcceptedBlockRewardInfo struct {
	ShardID          byte
	TxsFee           map[common.Hash]uint64
	ShardBlockHeight uint64
}

func BuildInstForShardReward(reward map[common.Hash]uint64, epoch uint64, shardID byte) ([][]string, error) {
	resIns := [][]string{}
	shardBlockRewardInfo := ShardBlockRewardInfo{
		Epoch:       epoch,
		ShardReward: reward,
	}

	contentStr, err := json.Marshal(shardBlockRewardInfo)
	if err != nil {
		return nil, err
	}

	returnedInst := []string{
		strconv.Itoa(ShardBlockRewardRequestMeta),
		strconv.Itoa(int(shardID)),
		SHARD_REWARD_INST,
		string(contentStr),
	}
	resIns = append(resIns, returnedInst)
	return resIns, nil
}

func NewShardBlockRewardInfoFromString(inst string) (*ShardBlockRewardInfo, error) {
	Ins := &ShardBlockRewardInfo{}
	err := json.Unmarshal([]byte(inst), Ins)
	if err != nil {
		return nil, err
	}
	return Ins, nil
}

func NewAcceptedBlockRewardInfo(
	shardID byte,
	txsFee map[common.Hash]uint64,
	shardBlockHeight uint64,
) *AcceptedBlockRewardInfo {
	return &AcceptedBlockRewardInfo{
		ShardID:          shardID,
		TxsFee:           txsFee,
		ShardBlockHeight: shardBlockHeight,
	}
}

func NewAcceptedBlockRewardInfoFromStr(
	inst string,
) (*AcceptedBlockRewardInfo, error) {
	Ins := &AcceptedBlockRewardInfo{}
	err := json.Unmarshal([]byte(inst), Ins)
	if err != nil {
		return nil, err
	}
	return Ins, nil
}

func (blockRewardInfo *AcceptedBlockRewardInfo) GetStringFormat() ([]string, error) {
	content, err := json.Marshal(blockRewardInfo)
	if err != nil {
		return nil, err
	}
	return []string{
		strconv.Itoa(AcceptedBlockRewardInfoMeta),
		strconv.Itoa(BeaconOnly),
		string(content),
	}, nil
}
