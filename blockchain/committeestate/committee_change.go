package committeestate

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type CommitteeChange struct {
	NextEpochBeaconCandidateAdded      []incognitokey.CommitteePublicKey
	NextEpochBeaconCandidateRemoved    []incognitokey.CommitteePublicKey
	CurrentEpochBeaconCandidateAdded   []incognitokey.CommitteePublicKey
	CurrentEpochBeaconCandidateRemoved []incognitokey.CommitteePublicKey
	NextEpochShardCandidateAdded       []incognitokey.CommitteePublicKey
	NextEpochShardCandidateRemoved     []incognitokey.CommitteePublicKey
	CurrentEpochShardCandidateAdded    []incognitokey.CommitteePublicKey
	CurrentEpochShardCandidateRemoved  []incognitokey.CommitteePublicKey
	ShardSubstituteAdded               map[byte][]incognitokey.CommitteePublicKey
	ShardSubstituteRemoved             map[byte][]incognitokey.CommitteePublicKey
	ShardCommitteeAdded                map[byte][]incognitokey.CommitteePublicKey
	ShardCommitteeRemoved              map[byte][]incognitokey.CommitteePublicKey
	BeaconSubstituteAdded              []incognitokey.CommitteePublicKey
	BeaconSubstituteRemoved            []incognitokey.CommitteePublicKey
	BeaconCommitteeAdded               []incognitokey.CommitteePublicKey
	BeaconCommitteeRemoved             []incognitokey.CommitteePublicKey
	BeaconCommitteeReplaced            [2][]incognitokey.CommitteePublicKey
	ShardCommitteeReplaced             map[byte][2][]incognitokey.CommitteePublicKey
	StopAutoStake                      []string
	Unstake                            []string
}

//GetStakerKeys ...
func (committeeChange *CommitteeChange) StakerKeys() []incognitokey.CommitteePublicKey {
	return committeeChange.NextEpochShardCandidateAdded
}

func (committeeChange *CommitteeChange) UnstakeKeys() []incognitokey.CommitteePublicKey {
	res := []incognitokey.CommitteePublicKey{}
	res, _ = incognitokey.CommitteeBase58KeyListToStruct(committeeChange.Unstake)
	return res
}

func (committeeChange *CommitteeChange) StopAutoStakeKeys() []incognitokey.CommitteePublicKey {
	res := []incognitokey.CommitteePublicKey{}
	res, _ = incognitokey.CommitteeBase58KeyListToStruct(committeeChange.StopAutoStake)
	return res
}

func NewCommitteeChange() *CommitteeChange {
	committeeChange := &CommitteeChange{
		ShardSubstituteAdded:    make(map[byte][]incognitokey.CommitteePublicKey),
		ShardSubstituteRemoved:  make(map[byte][]incognitokey.CommitteePublicKey),
		ShardCommitteeAdded:     make(map[byte][]incognitokey.CommitteePublicKey),
		ShardCommitteeRemoved:   make(map[byte][]incognitokey.CommitteePublicKey),
		ShardCommitteeReplaced:  make(map[byte][2][]incognitokey.CommitteePublicKey),
		BeaconCommitteeReplaced: [2][]incognitokey.CommitteePublicKey{},
	}
	for i := 0; i < common.MaxShardNumber; i++ {
		shardID := byte(i)
		committeeChange.ShardSubstituteAdded[shardID] = []incognitokey.CommitteePublicKey{}
		committeeChange.ShardSubstituteRemoved[shardID] = []incognitokey.CommitteePublicKey{}
		committeeChange.ShardCommitteeAdded[shardID] = []incognitokey.CommitteePublicKey{}
		committeeChange.ShardCommitteeRemoved[shardID] = []incognitokey.CommitteePublicKey{}
	}
	return committeeChange
}

func (committeeChange *CommitteeChange) clone(root *CommitteeChange) {
	for i, v := range root.BeaconCommitteeReplaced {
		committeeChange.BeaconCommitteeReplaced[i] = append(committeeChange.BeaconCommitteeReplaced[i], v...)
	}

	for i, v := range root.ShardCommitteeReplaced {
		// for index, value := range v {
		// 	committeeChange.ShardCommitteeReplaced[i][index] = append(committeeChange.ShardCommitteeReplaced[i][index], value...)
		// }
		committeeChange.ShardCommitteeReplaced[i] = v
	}

	for i, v := range root.ShardSubstituteAdded {
		committeeChange.ShardSubstituteAdded[i] = append(committeeChange.ShardSubstituteAdded[i], v...)
	}

	for i, v := range root.ShardSubstituteRemoved {
		committeeChange.ShardSubstituteRemoved[i] = append(committeeChange.ShardSubstituteRemoved[i], v...)
	}

	for i, v := range root.ShardCommitteeAdded {
		committeeChange.ShardCommitteeAdded[i] = append(committeeChange.ShardCommitteeAdded[i], v...)
	}

	for i, v := range root.ShardCommitteeRemoved {
		committeeChange.ShardCommitteeRemoved[i] = append(committeeChange.ShardCommitteeRemoved[i], v...)
	}

	committeeChange.StopAutoStake = append(committeeChange.StopAutoStake, root.StopAutoStake...)
	committeeChange.Unstake = append(committeeChange.Unstake, root.Unstake...)
	committeeChange.NextEpochBeaconCandidateAdded =
		append(committeeChange.NextEpochBeaconCandidateAdded, root.NextEpochBeaconCandidateAdded...)
	committeeChange.StopAutoStake = append(committeeChange.StopAutoStake, root.StopAutoStake...)
	committeeChange.NextEpochBeaconCandidateRemoved =
		append(committeeChange.NextEpochBeaconCandidateRemoved, root.NextEpochBeaconCandidateRemoved...)
	committeeChange.CurrentEpochBeaconCandidateAdded =
		append(committeeChange.CurrentEpochBeaconCandidateAdded, root.CurrentEpochBeaconCandidateAdded...)
	committeeChange.CurrentEpochBeaconCandidateRemoved =
		append(committeeChange.CurrentEpochBeaconCandidateRemoved, root.CurrentEpochBeaconCandidateRemoved...)
	committeeChange.NextEpochShardCandidateAdded =
		append(committeeChange.NextEpochShardCandidateAdded, root.NextEpochShardCandidateAdded...)
	committeeChange.NextEpochShardCandidateRemoved =
		append(committeeChange.NextEpochShardCandidateRemoved, root.NextEpochShardCandidateRemoved...)
	committeeChange.CurrentEpochShardCandidateAdded =
		append(committeeChange.CurrentEpochShardCandidateAdded, root.CurrentEpochShardCandidateAdded...)
	committeeChange.BeaconSubstituteAdded =
		append(committeeChange.BeaconSubstituteAdded, root.BeaconSubstituteAdded...)
	committeeChange.BeaconSubstituteRemoved =
		append(committeeChange.BeaconSubstituteRemoved, root.BeaconSubstituteRemoved...)
	committeeChange.BeaconCommitteeRemoved =
		append(committeeChange.BeaconCommitteeRemoved, root.BeaconCommitteeRemoved...)
	committeeChange.BeaconCommitteeAdded =
		append(committeeChange.BeaconCommitteeAdded, root.BeaconCommitteeAdded...)
}
