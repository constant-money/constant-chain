package jsonresult

import (
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type committeeKeySetAutoStake struct {
	IncPubKey    string
	MiningPubKey map[string]string
	IsAutoStake  bool
}

type GetBeaconBestStateDetail struct {
	BestBlockHash                          common.Hash                                `json:"BestBlockHash"`         // The hash of the block.
	PreviousBestBlockHash                  common.Hash                                `json:"PreviousBestBlockHash"` // The hash of the block.
	BestShardHash                          map[byte]common.Hash                       `json:"BestShardHash"`
	BestShardHeight                        map[byte]uint64                            `json:"BestShardHeight"`
	Epoch                                  uint64                                     `json:"Epoch"`
	BeaconHeight                           uint64                                     `json:"BeaconHeight"`
	BeaconProposerIndex                    int                                        `json:"BeaconProposerIndex"`
	BeaconCommittee                        []incognitokey.CommitteeKeyString          `json:"BeaconCommittee"`
	BeaconPendingValidator                 []incognitokey.CommitteeKeyString          `json:"BeaconPendingValidator"`
	CandidateShardWaitingForCurrentRandom  []incognitokey.CommitteeKeyString          `json:"CandidateShardWaitingForCurrentRandom"` // snapshot shard candidate list, waiting to be shuffled in this current epoch
	CandidateBeaconWaitingForCurrentRandom []incognitokey.CommitteeKeyString          `json:"CandidateBeaconWaitingForCurrentRandom"`
	CandidateShardWaitingForNextRandom     []incognitokey.CommitteeKeyString          `json:"CandidateShardWaitingForNextRandom"` // shard candidate list, waiting to be shuffled in next epoch
	CandidateBeaconWaitingForNextRandom    []incognitokey.CommitteeKeyString          `json:"CandidateBeaconWaitingForNextRandom"`
	RewardReceiver                         map[string]string                          `json:"RewardReceiver"`        // key: incognito public key of committee, value: payment address reward receiver
	ShardCommittee                         map[byte][]incognitokey.CommitteeKeyString `json:"ShardCommittee"`        // current committee and validator of all shard
	ShardPendingValidator                  map[byte][]incognitokey.CommitteeKeyString `json:"ShardPendingValidator"` // pending candidate waiting for swap to get in committee of all shard
	AutoStaking                            []committeeKeySetAutoStake                 `json:"AutoStaking"`
	CurrentRandomNumber                    int64                                      `json:"CurrentRandomNumber"`
	CurrentRandomTimeStamp                 int64                                      `json:"CurrentRandomTimeStamp"` // random timestamp for this epoch
	IsGetRandomNumber                      bool                                       `json:"IsGetRandomNumber"`
	MaxBeaconCommitteeSize                 int                                        `json:"MaxBeaconCommitteeSize"`
	MinBeaconCommitteeSize                 int                                        `json:"MinBeaconCommitteeSize"`
	MaxShardCommitteeSize                  int                                        `json:"MaxShardCommitteeSize"`
	MinShardCommitteeSize                  int                                        `json:"MinShardCommitteeSize"`
	ActiveShards                           int                                        `json:"ActiveShards"`

	LastCrossShardState map[byte]map[byte]uint64 `json:"LastCrossShardState"`
	ShardHandle         map[byte]bool            `json:"ShardHandle"` // lock sync.RWMutex
}

func NewGetBeaconBestStateDetail(data *blockchain.BeaconBestState) *GetBeaconBestStateDetail {
	result := &GetBeaconBestStateDetail{
		BestBlockHash:          data.BestBlockHash,
		PreviousBestBlockHash:  data.PreviousBestBlockHash,
		Epoch:                  data.Epoch,
		BeaconHeight:           data.BeaconHeight,
		BeaconProposerIndex:    data.BeaconProposerIndex,
		CurrentRandomNumber:    data.CurrentRandomNumber,
		CurrentRandomTimeStamp: data.CurrentRandomTimeStamp,
		IsGetRandomNumber:      data.IsGetRandomNumber,
		MaxShardCommitteeSize:  data.MaxShardCommitteeSize,
		MinShardCommitteeSize:  data.MinShardCommitteeSize,
		MaxBeaconCommitteeSize: data.MaxBeaconCommitteeSize,
		MinBeaconCommitteeSize: data.MinBeaconCommitteeSize,
		ActiveShards:           data.ActiveShards,
	}
	result.BestShardHash = make(map[byte]common.Hash)
	for k, v := range data.BestShardHash {
		result.BestShardHash[k] = v
	}

	result.BestShardHeight = make(map[byte]uint64)
	for k, v := range data.BestShardHeight {
		result.BestShardHeight[k] = v
	}

	tempBeaconCommittee := incognitokey.CommitteeKeyListToStringList(data.BeaconCommittee)

	result.BeaconCommittee = make([]incognitokey.CommitteeKeyString, len(data.BeaconCommittee))
	copy(result.BeaconCommittee, tempBeaconCommittee)

	tempBeaconPendingValidator := incognitokey.CommitteeKeyListToStringList(data.BeaconPendingValidator)

	result.BeaconPendingValidator = make([]incognitokey.CommitteeKeyString, len(data.BeaconPendingValidator))
	copy(result.BeaconPendingValidator, tempBeaconPendingValidator)

	tempCandidateShardWaitingForCurrentRandom := incognitokey.CommitteeKeyListToStringList(data.CandidateShardWaitingForCurrentRandom)

	result.CandidateShardWaitingForCurrentRandom = make([]incognitokey.CommitteeKeyString, len(data.CandidateShardWaitingForCurrentRandom))
	copy(result.CandidateShardWaitingForCurrentRandom, tempCandidateShardWaitingForCurrentRandom)

	tempCandidateBeaconWaitingForCurrentRandom := incognitokey.CommitteeKeyListToStringList(data.CandidateBeaconWaitingForCurrentRandom)

	result.CandidateBeaconWaitingForCurrentRandom = make([]incognitokey.CommitteeKeyString, len(data.CandidateBeaconWaitingForCurrentRandom))
	copy(result.CandidateBeaconWaitingForCurrentRandom, tempCandidateBeaconWaitingForCurrentRandom)

	tempCandidateShardWaitingForNextRandom := incognitokey.CommitteeKeyListToStringList(data.CandidateShardWaitingForNextRandom)

	result.CandidateShardWaitingForNextRandom = make([]incognitokey.CommitteeKeyString, len(data.CandidateShardWaitingForNextRandom))
	copy(result.CandidateShardWaitingForNextRandom, tempCandidateShardWaitingForNextRandom)

	tempCandidateBeaconWaitingForNextRandom := incognitokey.CommitteeKeyListToStringList(data.CandidateBeaconWaitingForNextRandom)

	result.CandidateBeaconWaitingForNextRandom = make([]incognitokey.CommitteeKeyString, len(data.CandidateBeaconWaitingForNextRandom))
	copy(result.CandidateBeaconWaitingForNextRandom, tempCandidateBeaconWaitingForNextRandom)

	result.RewardReceiver = make(map[string]string)
	for k, v := range data.RewardReceiver {
		result.RewardReceiver[k] = v.String()
	}

	result.ShardCommittee = make(map[byte][]incognitokey.CommitteeKeyString)
	for k, v := range data.ShardCommittee {
		result.ShardCommittee[k] = make([]incognitokey.CommitteeKeyString, len(v))
		tempV := incognitokey.CommitteeKeyListToStringList(v)
		copy(result.ShardCommittee[k], tempV)
	}

	result.ShardPendingValidator = make(map[byte][]incognitokey.CommitteeKeyString)
	for k, v := range data.ShardPendingValidator {
		result.ShardPendingValidator[k] = make([]incognitokey.CommitteeKeyString, len(v))
		tempV := incognitokey.CommitteeKeyListToStringList(v)
		copy(result.ShardPendingValidator[k], tempV)
	}

	result.LastCrossShardState = make(map[byte]map[byte]uint64)
	for k1, v1 := range data.LastCrossShardState {
		result.LastCrossShardState[k1] = make(map[byte]uint64)
		for k2, v2 := range v1 {
			result.LastCrossShardState[k1][k2] = v2
		}
	}

	for k, v := range data.AutoStaking.GetMap() {
		var keySet incognitokey.CommitteePublicKey
		keySet.FromString(k)

		var keyMap committeeKeySetAutoStake
		keyMap.IncPubKey = keySet.GetIncKeyBase58()
		keyMap.MiningPubKey = make(map[string]string)
		for keyType := range keySet.MiningPubKey {
			keyMap.MiningPubKey[keyType] = keySet.GetMiningKeyBase58(keyType)
		}
		keyMap.IsAutoStake = v
		result.AutoStaking = append(result.AutoStaking, keyMap)

	}
	return result
}
