package blockchain

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type committeeChange struct {
	nextEpochBeaconCandidateAdded      []incognitokey.CommitteePublicKey
	nextEpochBeaconCandidateRemoved    []incognitokey.CommitteePublicKey
	currentEpochBeaconCandidateAdded   []incognitokey.CommitteePublicKey
	currentEpochBeaconCandidateRemoved []incognitokey.CommitteePublicKey
	nextEpochShardCandidateAdded       []incognitokey.CommitteePublicKey
	nextEpochShardCandidateRemoved     []incognitokey.CommitteePublicKey
	currentEpochShardCandidateAdded    []incognitokey.CommitteePublicKey
	currentEpochShardCandidateRemoved  []incognitokey.CommitteePublicKey
	shardSubstituteAdded               map[byte][]incognitokey.CommitteePublicKey
	shardSubstituteRemoved             map[byte][]incognitokey.CommitteePublicKey
	shardCommitteeAdded                map[byte][]incognitokey.CommitteePublicKey
	shardCommitteeRemoved              map[byte][]incognitokey.CommitteePublicKey
	beaconSubstituteAdded              []incognitokey.CommitteePublicKey
	beaconSubstituteRemoved            []incognitokey.CommitteePublicKey
	beaconCommitteeAdded               []incognitokey.CommitteePublicKey
	beaconCommitteeRemoved             []incognitokey.CommitteePublicKey
	beaconCommitteeReplaced            [2][]incognitokey.CommitteePublicKey
	shardCommitteeReplaced             map[byte][2][]incognitokey.CommitteePublicKey
	stopAutoStaking                    []string
}

func newCommitteeChange() *committeeChange {
	committeeChange := &committeeChange{
		shardSubstituteAdded:    make(map[byte][]incognitokey.CommitteePublicKey),
		shardSubstituteRemoved:  make(map[byte][]incognitokey.CommitteePublicKey),
		shardCommitteeAdded:     make(map[byte][]incognitokey.CommitteePublicKey),
		shardCommitteeRemoved:   make(map[byte][]incognitokey.CommitteePublicKey),
		shardCommitteeReplaced:  make(map[byte][2][]incognitokey.CommitteePublicKey),
		beaconCommitteeReplaced: [2][]incognitokey.CommitteePublicKey{},
	}
	for i := 0; i < common.MaxShardNumber; i++ {
		shardID := byte(i)
		committeeChange.shardSubstituteAdded[shardID] = []incognitokey.CommitteePublicKey{}
		committeeChange.shardSubstituteRemoved[shardID] = []incognitokey.CommitteePublicKey{}
		committeeChange.shardCommitteeAdded[shardID] = []incognitokey.CommitteePublicKey{}
		committeeChange.shardCommitteeRemoved[shardID] = []incognitokey.CommitteePublicKey{}
		committeeChange.shardCommitteeReplaced[shardID] = [2][]incognitokey.CommitteePublicKey{}
	}
	return committeeChange
}
