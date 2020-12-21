package statedb

import (
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

func storeCommitteeObjectOneShardForTestConsensus(role int, initRoot common.Hash, shardID, from, to int) (common.Hash, map[common.Hash]*CommitteeState) {
	m := make(map[common.Hash]*CommitteeState)
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		panic(err)
	}
	tempCommitteePublicKey = tempCommitteePublicKey[from:to]
	for _, value := range tempCommitteePublicKey {
		key, _ := GenerateCommitteeObjectKeyWithRole(role, shardID, value)

		committeeState := NewCommitteeStateWithValue(shardID, role, value)
		m[key] = committeeState
	}
	sDB, err := NewWithPrefixTrie(initRoot, wrarperDB)
	if err != nil {
		panic(err)
	}
	for key, value := range m {
		sDB.SetStateObject(CommitteeObjectType, key, value)
	}
	rootHash, err := sDB.Commit(true)
	if err != nil {
		panic(err)
	}
	err = sDB.Database().TrieDB().Commit(rootHash, false)
	if err != nil {
		panic(err)
	}
	return rootHash, m
}

func storeAllConsensusStateObjectForTesting(initRoot common.Hash) (
	common.Hash,
	map[int][]incognitokey.CommitteePublicKey,
	map[int][]incognitokey.CommitteePublicKey,
	[]incognitokey.CommitteePublicKey,
	[]incognitokey.CommitteePublicKey,
	map[string]map[common.Hash]uint64,
	map[common.Hash]*RewardRequestState,
	map[string]uint8,
) {
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	shardIDs := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	wantMCommittee := make(map[int][]incognitokey.CommitteePublicKey)
	wantMSubstituteValidator := make(map[int][]incognitokey.CommitteePublicKey)
	wantNextEpochCandidate := []incognitokey.CommitteePublicKey{}
	wantCurrentEpochCandidate := []incognitokey.CommitteePublicKey{}
	wantMCommitteeReward := make(map[string]map[common.Hash]uint64)
	wantMRewardRequest := make(map[common.Hash]*RewardRequestState)
	wantMBlackListProducer := make(map[string]uint8)
	// Committee
	from, to := 0, 32
	wantCurrentValidatorM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShardForTestConsensus(CurrentValidator, rootHashes[index], id, from, to)
		from += 32
		to += 32
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantCurrentValidatorM[id] = append(wantCurrentValidatorM[id], v.CommitteePublicKey())
		}
	}

	tempRootHash := rootHashes[8]
	to = from + 8
	rootHashes = []common.Hash{tempRootHash}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShardForTestConsensus(SubstituteValidator, rootHashes[index], id, from, to)
		from += 8
		to += 8
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantMSubstituteValidator[id] = append(wantMSubstituteValidator[id], v.CommitteePublicKey())
		}
	}
	to = from + 80
	tempRootHash = rootHashes[8]
	tempRootHash, tempM := storeCommitteeObjectOneShardForTestConsensus(NextEpochShardCandidate, tempRootHash, CandidateShardID, from, to)
	for _, v := range tempM {
		wantNextEpochCandidate = append(wantNextEpochCandidate, v.CommitteePublicKey())
	}

	from += 80
	to += 80
	tempRootHash, tempM = storeCommitteeObjectOneShardForTestConsensus(CurrentEpochShardCandidate, tempRootHash, CandidateShardID, from, to)
	for _, v := range tempM {
		wantCurrentEpochCandidate = append(wantCurrentEpochCandidate, v.CommitteePublicKey())
	}
	tempStateDB, err := NewWithPrefixTrie(tempRootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	tempRootHash, _, wantMCommitteeReward = storeCommitteeReward(tempRootHash, wrarperDB)
	tempRootHash, wantMRewardRequest = storeRewardRequest(tempRootHash, wrarperDB, defaultMaxEpoch, shardIDs)
	tempRootHash, _, wantMBlackListProducer = storeBlackListProducer(tempRootHash, wrarperDB, 1, 460, 500)
	return tempRootHash, wantMCommittee, wantMSubstituteValidator, wantNextEpochCandidate, wantCurrentEpochCandidate, wantMCommitteeReward, wantMRewardRequest, wantMBlackListProducer
}

func TestStateDB_GetAllConsensusStateObject(t *testing.T) {
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	rootHash, wantMCommittee, wantMSubstituteValidator, wantNextEpochCandidate, wantCurrentEpochCandidate, wantMCommitteeReward, wantMRewardRequest, wantMBlackListProducer := storeAllConsensusStateObjectForTesting(emptyRoot)
	// GOT to verify
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err)
	}
	gotMCommittee := tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, ids)
	for _, id := range ids {
		temp, ok := gotMCommittee[id]
		if !ok {
			t.Fatalf("getAllValidatorCommitteePublicKey want shard %+v", id)
		}
		if len(temp) != 32 {
			t.Fatalf("getAllValidatorCommitteePublicKey want key length %+v but got %+v", 32, len(temp))
		}
	}
	for id, wants := range wantMCommittee {
		flag := false
		for _, want := range wants {
			for _, got := range gotMCommittee[id] {
				if reflect.DeepEqual(got, want) {
					flag = true
					break
				}
			}
			if !flag {
				t.Fatalf("getAllValidatorCommitteePublicKey shard %+v want %+v but didn't get anything", id, want)
			}
		}
	}

	gotMSubstituteValidator := tempStateDB.getAllValidatorCommitteePublicKey(SubstituteValidator, ids)
	for _, id := range ids {
		temp, ok := gotMSubstituteValidator[id]
		if !ok {
			t.Fatalf("getAllValidatorCommitteePublicKey want shard %+v", id)
		}
		if len(temp) != 8 {
			t.Fatalf("getAllValidatorCommitteePublicKey want key length %+v but got %+v", 8, len(temp))
		}
	}
	for id, wants := range wantMSubstituteValidator {
		flag := false
		for _, want := range wants {
			for _, got := range gotMSubstituteValidator[id] {
				if reflect.DeepEqual(got.CommitteePublicKey(), want) {
					flag = true
					break
				}
			}
			if !flag {
				t.Fatalf("getAllValidatorCommitteePublicKey shard %+v want %+v but didn't get anything", id, want)
			}
		}
	}

	gotNextEpochCandidate := tempStateDB.getAllCandidateCommitteePublicKey(NextEpochShardCandidate)
	if len(gotNextEpochCandidate) != 80 {
		t.Fatalf("getAllCandidateCommitteePublicKey want key length %+v but got %+v", 80, len(gotNextEpochCandidate))
	}
	for id, want := range wantNextEpochCandidate {
		flag := false
		for _, got := range gotNextEpochCandidate {
			if reflect.DeepEqual(got.CommitteePublicKey(), want) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCandidateCommitteePublicKey shard %+v want %+v but didn't get anything", id, want)
		}
	}

	gotCurrentEpochCandidate := tempStateDB.getAllCandidateCommitteePublicKey(CurrentEpochShardCandidate)
	if len(gotCurrentEpochCandidate) != 80 {
		t.Fatalf("getAllCandidateCommitteePublicKey want key length %+v but got %+v", 80, len(gotCurrentEpochCandidate))
	}
	for id, want := range wantCurrentEpochCandidate {
		flag := false
		for _, got := range gotCurrentEpochCandidate {
			if reflect.DeepEqual(got.CommitteePublicKey(), want) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCandidateCommitteePublicKey %+v want %+v but didn't get anything", id, want)
		}
	}

	gotMCommitteeReward := tempStateDB.getAllCommitteeReward()
	for k1, v1 := range wantMCommitteeReward {
		if v2, ok := gotMCommitteeReward[k1]; !ok {
			t.Fatalf("want %+v but got nothing", k1)
		} else {
			for k11, v11 := range v1 {
				if v22, ok := v2[k11]; !ok {
					t.Fatalf("want %+v but got nothing", k11)
				} else {
					if v11 != v22 {
						t.Fatalf("want %+v but got %+v", v11, v22)
					}
				}
			}
		}
	}
	for k, v := range wantMRewardRequest {
		gotRewardRequest, has, err := tempStateDB.getRewardRequestAmount(k)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatal(has)
		}
		if v.Amount() != gotRewardRequest {
			t.Fatalf("want %+v but got %+v", v.Amount(), gotRewardRequest)
		}
	}
	gotMBlackListProducer := tempStateDB.getAllProducerBlackList()
	for k, v := range wantMBlackListProducer {
		if v2, ok := gotMBlackListProducer[k]; !ok {
			t.Fatalf("want %+v but got nothing", k)
		} else if v != v2 {
			t.Fatalf("want %+v but got %+v", v, v2)
		}
	}
}

func BenchmarkStateDB_GetAllCommitteeRewardInFullData(b *testing.B) {
	rootHash, _, _, _, _, _, _, _ := storeAllConsensusStateObjectForTesting(emptyRoot)
	// GOT to verify
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		tempStateDB.getAllCommitteeReward()
	}
}

func BenchmarkStateDB_GetAllAutoStakingInFullData(b *testing.B) {
	rootHash, _, _, _, _, _, _, _ := storeAllConsensusStateObjectForTesting(emptyRoot)
	// GOT to verify
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		tempStateDB.getAllCommitteeState([]int{0, 1, 2, 3, 4, 5, 6, 7, 8})
	}
}
func BenchmarkStateDB_GetAllCommitteeInFullData(b *testing.B) {
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	rootHash, _, _, _, _, _, _, _ := storeAllConsensusStateObjectForTesting(emptyRoot)
	// GOT to verify
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, ids)
	}
}
