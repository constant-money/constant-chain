package statedb

import (
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

func TestStateDB_TestChangeAutoStaking(t *testing.T) {
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		t.Fatal(err)
	}
	tempCommitteePublicKey = tempCommitteePublicKey[:1]
	key, _ := GenerateCommitteeObjectKeyWithRole(CurrentValidator, 0, tempCommitteePublicKey[0])
	committeeState := NewCommitteeStateWithValue(0, CurrentValidator, tempCommitteePublicKey[0])
	sDB, err := NewWithPrefixTrie(emptyRoot, wrarperDB)
	if err != nil {
		t.Fatal(err)
	}
	err = sDB.SetStateObject(CommitteeObjectType, key, committeeState)
	if err != nil {
		t.Fatal(err)
	}
	rootHash, err := sDB.Commit(true)
	if err != nil {
		t.Fatal(err)
	}
	err = sDB.Database().TrieDB().Commit(rootHash, false)
	if err != nil {
		t.Fatal(err)
	}
	committeeState1 := NewCommitteeStateWithValue(0, CurrentValidator, tempCommitteePublicKey[0])
	err = sDB.SetStateObject(CommitteeObjectType, key, committeeState1)
	rootHash2, err := sDB.Commit(true)
	if err != nil {
		t.Fatal(err)
	}
	err = sDB.Database().TrieDB().Commit(rootHash2, false)
	if err != nil {
		t.Fatal(err)
	}
	// _, newAutoStaking := GetRewardReceiverAndAutoStaking(sDB, []int{0})
	// log.Println(newAutoStaking)
}

func TestStateDB_GetMixCommitteePublicKey(t *testing.T) {
	from, to := 0, 32
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantCurrentValidatorM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentValidator, rootHashes[index], id, from, to)
		from += 32
		to += 32
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantCurrentValidatorM[id] = append(wantCurrentValidatorM[id], v.CommitteePublicKey())
		}
	}
	tempRootHash := rootHashes[8]
	wantSubstituteValidatorM := make(map[int][]incognitokey.CommitteePublicKey)
	to = from + 8
	rootHashes = []common.Hash{tempRootHash}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(SubstituteValidator, rootHashes[index], id, from, to)
		from += 8
		to += 8
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantSubstituteValidatorM[id] = append(wantSubstituteValidatorM[id], v.CommitteePublicKey())
		}
	}
	to = from + 80
	tempRootHash = rootHashes[8]
	wantNextEpochCandidateM := []incognitokey.CommitteePublicKey{}
	tempRootHash, tempM := storeCommitteeObjectOneShard(NextEpochShardCandidate, tempRootHash, CandidateShardID, from, to)
	for _, v := range tempM {
		wantNextEpochCandidateM = append(wantNextEpochCandidateM, v.CommitteePublicKey())
	}

	from += 80
	to += 80
	wantCurrentEpochCandidateM := []incognitokey.CommitteePublicKey{}
	tempRootHash, tempM = storeCommitteeObjectOneShard(CurrentEpochShardCandidate, tempRootHash, CandidateShardID, from, to)
	for _, v := range tempM {
		wantCurrentEpochCandidateM = append(wantCurrentEpochCandidateM, v.CommitteePublicKey())
	}

	tempStateDB, err := NewWithPrefixTrie(tempRootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}

	gotCurrentValidatorM := tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, ids)
	for _, id := range ids {
		temp, ok := gotCurrentValidatorM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 32 {
			t.Fatalf("getAllCommitteeState shard %+v want key length %+v but got %+v", id, 32, len(temp))
		}
	}
	for id, wants := range wantCurrentValidatorM {
		flag := false
		for _, want := range wants {
			for _, got := range gotCurrentValidatorM[id] {
				if reflect.DeepEqual(got.CommitteePublicKey(), want) {
					flag = true
					break
				}
			}
			if !flag {
				t.Fatalf("getAllCommitteeState shard %+v want %+v but didn't get anything", id, want)
			}
		}
	}

	gotSubstituteValidatorM := tempStateDB.getAllValidatorCommitteePublicKey(SubstituteValidator, ids)
	for _, id := range ids {
		temp, ok := gotSubstituteValidatorM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 8 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 8, len(temp))
		}
	}
	for id, wants := range wantSubstituteValidatorM {
		flag := false
		for _, want := range wants {
			for _, got := range gotSubstituteValidatorM[id] {
				if reflect.DeepEqual(got.CommitteePublicKey(), want) {
					flag = true
					break
				}
			}
			if !flag {
				t.Fatalf("getAllCommitteeState shard %+v want %+v but didn't get anything", id, want)
			}
		}
	}

	gotNextEpochCandidateM := tempStateDB.getAllCandidateCommitteePublicKey(NextEpochShardCandidate)
	if len(gotNextEpochCandidateM) != 80 {
		t.Fatalf("getAllCandidateCommitteePublicKey want key length %+v but got %+v", to-from, len(gotNextEpochCandidateM))
	}
	for _, want := range wantNextEpochCandidateM {
		flag := false
		for _, got := range gotNextEpochCandidateM {
			if reflect.DeepEqual(got.CommitteePublicKey(), want) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCandidateCommitteePublicKey want %+v but didn't get anything", want)
		}
	}

	gotCurrentEpochCandidateM := tempStateDB.getAllCandidateCommitteePublicKey(CurrentEpochShardCandidate)
	if len(gotNextEpochCandidateM) != 80 {
		t.Fatalf("getAllCandidateCommitteePublicKey want key length %+v but got %+v", to-from, len(gotCurrentEpochCandidateM))
	}
	for _, want := range wantCurrentEpochCandidateM {
		flag := false
		for _, got := range gotCurrentEpochCandidateM {
			if reflect.DeepEqual(got.CommitteePublicKey(), want) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCandidateCommitteePublicKey want %+v but didn't get anything", want)
		}
	}
}

func TestStateDB_GetMixCommitteeState(t *testing.T) {
	from, to := 0, 32
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantCurrentValidatorM := make(map[int][]incognitokey.CommitteePublicKey)
	wantSubstituteValidatorM := make(map[int][]incognitokey.CommitteePublicKey)
	wantNextEpochCandidateM := []incognitokey.CommitteePublicKey{}
	wantCurrentEpochCandidateM := []incognitokey.CommitteePublicKey{}

	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentValidator, rootHashes[index], id, from, to)
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
		tempRootHash, tempM := storeCommitteeObjectOneShard(SubstituteValidator, rootHashes[index], id, from, to)
		from += 8
		to += 8
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantSubstituteValidatorM[id] = append(wantSubstituteValidatorM[id], v.CommitteePublicKey())
		}
	}
	to = from + 80
	tempRootHash = rootHashes[8]
	tempRootHash, tempM := storeCommitteeObjectOneShard(NextEpochShardCandidate, tempRootHash, CandidateShardID, from, to)
	for _, v := range tempM {
		wantNextEpochCandidateM = append(wantNextEpochCandidateM, v.CommitteePublicKey())
	}

	from += 80
	to += 80
	tempRootHash, tempM = storeCommitteeObjectOneShard(CurrentEpochShardCandidate, tempRootHash, CandidateShardID, from, to)
	for _, v := range tempM {
		wantCurrentEpochCandidateM = append(wantCurrentEpochCandidateM, v.CommitteePublicKey())
	}

	tempStateDB, err := NewWithPrefixTrie(tempRootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}

	gotCurrentValidatorM, gotSubstituteValidatorM, gotNextEpochCandidateM, gotCurrentEpochCandidateM, _, _ := tempStateDB.getAllCommitteeState(ids)
	for _, id := range ids {
		temp, ok := gotCurrentValidatorM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 32 {
			t.Fatalf("getAllCommitteeState shard %+v want key length %+v but got %+v", id, 32, len(temp))
		}
	}
	for id, wants := range wantCurrentValidatorM {
		flag := false
		for _, want := range wants {
			for _, got := range gotCurrentValidatorM[id] {
				if reflect.DeepEqual(got.CommitteePublicKey(), want) {
					flag = true
					break
				}
			}
			if !flag {
				t.Fatalf("getAllCommitteeState shard %+v want %+v but didn't get anything", id, want)
			}
		}
	}

	for _, id := range ids {
		temp, ok := gotSubstituteValidatorM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 8 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 8, len(temp))
		}
	}
	for id, wants := range wantSubstituteValidatorM {
		flag := false
		for _, want := range wants {
			for _, got := range gotSubstituteValidatorM[id] {
				if reflect.DeepEqual(got.CommitteePublicKey(), want) {
					flag = true
					break
				}
			}
			if !flag {
				t.Fatalf("getAllCommitteeState shard %+v want %+v but didn't get anything", id, want)
			}
		}
	}

	if len(gotNextEpochCandidateM) != 80 {
		t.Fatalf("getAllCommitteeState want key length %+v but got %+v", to-from, len(gotNextEpochCandidateM))
	}
	for _, want := range wantNextEpochCandidateM {
		flag := false
		for _, got := range gotNextEpochCandidateM {
			if reflect.DeepEqual(got.CommitteePublicKey(), want) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCommitteeState want %+v but didn't get anything", want)
		}
	}

	if len(gotNextEpochCandidateM) != 80 {
		t.Fatalf("getAllCommitteeState want key length %+v but got %+v", to-from, len(gotCurrentEpochCandidateM))
	}
	for _, want := range wantCurrentEpochCandidateM {
		flag := false
		for _, got := range gotCurrentEpochCandidateM {
			if reflect.DeepEqual(got.CommitteePublicKey(), want) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCommitteeState want %+v but didn't get anything", want)
		}
	}

}
