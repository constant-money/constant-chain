package statedb

import (
	"github.com/incognitochain/incognito-chain/incognitokey"
	"reflect"
	"testing"
)

func TestStateDB_GetNextEpochCandidateCommitteeState(t *testing.T) {
	rootHash, m := storeCommitteeObjectOneShard(NextEpochShardCandidate, emptyRoot, 0, 0, len(committeePublicKeys))
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	for key, want := range m {
		got, has, err := tempStateDB.getCommitteeState(key)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatalf("getCommitteeState want key %+v, value %+v but didn't get anything", key, want)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getCommitteeState want key %+v, value %+v but got value %+v ", key, want, got)
		}
	}
}

func TestStateDB_GetCurrentEpochCandidateCommitteeState(t *testing.T) {
	rootHash, m := storeCommitteeObjectOneShard(CurrentEpochShardCandidate, emptyRoot, 0, 0, len(committeePublicKeys))
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	for key, want := range m {
		got, has, err := tempStateDB.getCommitteeState(key)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatalf("getCommitteeState want key %+v, value %+v but didn't get anything", key, want)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getCommitteeState want key %+v, value %+v but got value %+v ", key, want, got)
		}
	}
}

func TestStateDB_GetAllCurrentEpochCandidateCommitteeKey512EightShard(t *testing.T) {
	from, to := 0, 512
	wantM := []incognitokey.CommitteePublicKey{}
	tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentEpochShardCandidate, emptyRoot, CandidateChainID, from, to)
	for _, v := range tempM {
		wantM = append(wantM, v.CommitteePublicKey())
	}
	tempStateDB, err := NewWithPrefixTrie(tempRootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllCandidateCommitteePublicKey(CurrentEpochShardCandidate)

	if len(gotM) != to-from {
		t.Fatalf("getAllCandidateCommitteePublicKey want key length %+v but got %+v", to-from, len(gotM))
	}
	for _, want := range wantM {
		flag := false
		for _, got := range gotM {
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
func TestStateDB_GetAllNextEpochCandidateCommitteeKey(t *testing.T) {
	from, to := 0, 512
	rootHash, m := storeCommitteeObjectOneShard(NextEpochShardCandidate, emptyRoot, 0, from, to)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllCandidateCommitteePublicKey(NextEpochShardCandidate)
	if len(gotM) != to-from {
		t.Fatalf("getAllCommitteeState want key length %+v but got %+v", to-from, len(gotM))
	}
	for _, want := range m {
		flag := false
		for _, got := range gotM {
			if reflect.DeepEqual(got.CommitteePublicKey(), want.CommitteePublicKey()) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCommitteeState want %+v but didn't get anything", want.CommitteePublicKey())
		}
	}
}

func TestStateDB_GetAllCurrentEpochCandidateCommitteeKey(t *testing.T) {
	from, to := 0, 512
	rootHash, m := storeCommitteeObjectOneShard(CurrentEpochShardCandidate, emptyRoot, 0, from, to)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllCandidateCommitteePublicKey(CurrentEpochShardCandidate)
	if len(gotM) != to-from {
		t.Fatalf("getAllCommitteeState want key length %+v but got %+v", to-from, len(gotM))
	}
	for _, want := range m {
		flag := false
		for _, got := range gotM {
			if reflect.DeepEqual(got.CommitteePublicKey(), want.CommitteePublicKey()) {
				flag = true
				break
			}
		}
		if !flag {
			t.Fatalf("getAllCommitteeState want %+v but didn't get anything", want.CommitteePublicKey())
		}
	}
}

func TestStateDB_GetAllNextEpochCandidateCommitteeKey512EightShard(t *testing.T) {
	from, to := 0, 512
	wantM := []incognitokey.CommitteePublicKey{}
	tempRootHash, tempM := storeCommitteeObjectOneShard(NextEpochShardCandidate, emptyRoot, CandidateChainID, from, to)
	for _, v := range tempM {
		wantM = append(wantM, v.CommitteePublicKey())
	}
	tempStateDB, err := NewWithPrefixTrie(tempRootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllCandidateCommitteePublicKey(NextEpochShardCandidate)

	if len(gotM) != to-from {
		t.Fatalf("getAllCandidateCommitteePublicKey want key length %+v but got %+v", to-from, len(gotM))
	}
	for _, want := range wantM {
		flag := false
		for _, got := range gotM {
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
