package statedb

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

func storeCommitteeObjectOneShard(role int, initRoot common.Hash, shardID, from, to int) (common.Hash, map[common.Hash]*CommitteeState) {
	m := make(map[common.Hash]*CommitteeState)
	m1 := make(map[common.Hash]*StakerInfo)
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		panic(err)
	}
	tempCommitteePublicKey = tempCommitteePublicKey[from:to]
	for _, value := range tempCommitteePublicKey {
		key, _ := GenerateCommitteeObjectKeyWithRole(role, shardID, value)
		committeeState := NewCommitteeStateWithValue(shardID, role, value)
		m[key] = committeeState
		keyBytes, err := value.RawBytes()
		if err != nil {
			panic(err)
		}
		key1 := GetStakerInfoKey(keyBytes)
		stakerInfo := NewStakerInfoWithValue(
			receiverPaymentAddressStructs[0],
			true,
			txHashes[0],
		)
		m1[key1] = stakerInfo
	}
	sDB, err := NewWithPrefixTrie(initRoot, wrarperDB)
	if err != nil {
		panic(err)
	}
	for key, value := range m {
		sDB.SetStateObject(CommitteeObjectType, key, value)
	}
	for key, value := range m1 {
		sDB.SetStateObject(StakerObjectType, key, value)
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

func TestStateDB_SetStateObjectCommitteeState(t *testing.T) {
	var shardID = 0
	var err error = nil
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		panic(err)
	}
	sampleCommittee := tempCommitteePublicKey[0]
	sampleCommittee2 := tempCommitteePublicKey[1]
	key, _ := GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, sampleCommittee)
	key2, _ := GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, sampleCommittee2)
	committeeState := NewCommitteeStateWithValue(shardID, CurrentValidator, sampleCommittee)
	sDB, err := NewWithPrefixTrie(emptyRoot, wrarperDB)
	if err != nil {
		panic(err)
	}
	err = sDB.SetStateObject(CommitteeObjectType, key, committeeState)
	if err != nil {
		t.Fatal(err)
	}
	err = sDB.SetStateObject(CommitteeObjectType, key, "committeeState")
	if err == nil {
		t.Fatal("expect error")
	}
	err = sDB.SetStateObject(CommitteeObjectType, key, []byte("committee state"))
	if err == nil {
		t.Fatal("expect error")
	}
	err = sDB.SetStateObject(CommitteeObjectType, key2, []byte("committee state"))
	if err == nil {
		t.Fatal("expect error")
	}
	stateObjects := sDB.GetStateObjectMapForTestOnly()
	if _, ok := stateObjects[key2]; ok {
		t.Fatalf("want nothing but got %+v", key2)
	}
	rootHash, err := sDB.Commit(true)
	if err != nil {
		t.Fatal(err)
	}
	err = sDB.Database().TrieDB().Commit(rootHash, false)
	if err != nil {
		t.Fatal(err)
	}
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err)
	}
	got, has, err := tempStateDB.getCommitteeState(key)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal(has)
	}
	if !reflect.DeepEqual(got, committeeState) {
		t.Fatalf("want value %+v but got %+v", committeeState, got)
	}

	got2, has2, err := tempStateDB.getCommitteeState(key2)
	if err != nil {
		t.Fatal(err)
	}
	if has2 {
		t.Fatal(has2)
	}
	if !reflect.DeepEqual(got2, NewCommitteeState()) {
		t.Fatalf("want value %+v but got %+v", NewCommitteeState(), got2)
	}
}

func TestStateDB_SetDuplicateStateObjectCommitteeState(t *testing.T) {
	var shardID = 0
	var err error = nil
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		panic(err)
	}
	sampleCommittee := tempCommitteePublicKey[0]
	key, _ := GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, sampleCommittee)
	committeeState := NewCommitteeStateWithValue(shardID, CurrentValidator, sampleCommittee)
	sDB, err := NewWithPrefixTrie(emptyRoot, wrarperDB)
	if err != nil {
		panic(err)
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
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err)
	}
	got, has, err := tempStateDB.getCommitteeState(key)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal(has)
	}
	if !reflect.DeepEqual(got, committeeState) {
		t.Fatalf("want value %+v but got %+v", committeeState, got)
	}
	err = tempStateDB.SetStateObject(CommitteeObjectType, key, committeeState)
	if err != nil {
		t.Fatal(err)
	}
	rootHash2, err := sDB.Commit(true)
	if err != nil {
		t.Fatal(err)
	}
	err = sDB.Database().TrieDB().Commit(rootHash2, false)
	if err != nil {
		t.Fatal(err)
	}
	if !rootHash.IsEqual(&rootHash2) {
		t.Fatalf("expect %+v equal to %+v", rootHash2, rootHash)
	}
}

func TestStateDB_GetCurrentValidatorCommitteeState(t *testing.T) {
	rootHash, m := storeCommitteeObjectOneShard(CurrentValidator, emptyRoot, 0, 0, len(committeePublicKeys))
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

func TestStateDB_GetAllCurrentValidatorCommitteeKey512OneShard(t *testing.T) {
	from, to := 0, 512
	rootHash, m := storeCommitteeObjectOneShard(CurrentValidator, emptyRoot, 0, from, to)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, []int{0})
	gotMShard0, ok := gotM[0]
	if !ok {
		t.Fatalf("getAllCommitteeState want shard %+v", 0)
	}
	if len(gotMShard0) != to-from {
		t.Fatalf("getAllCommitteeState want key length %+v but got %+v", to-from, len(gotM))
	}
	for _, want := range m {
		flag := false
		for _, got := range gotMShard0 {
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

func TestStateDB_GetAllCurrentValidatorCommitteeKey512EightShard(t *testing.T) {
	from, to := 0, 64
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentValidator, rootHashes[index], id, from, to)
		from += 64
		to += 64
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, ids)
	for _, id := range ids {
		temp, ok := gotM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 64 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 64, len(temp))
		}
	}
	for id, wants := range wantM {
		flag := false
		for _, want := range wants {
			for _, got := range gotM[id] {
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
}

func TestStateDB_GetAllCurrentValidatorCommitteePublicKey512EightShardMultipleRootHash(t *testing.T) {
	from, to := 0, 32
	maxHeight := 8
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	wantMs := []map[int][]incognitokey.CommitteePublicKey{}
	rootHashesByHeight := []common.Hash{}
	rootHashes := []common.Hash{emptyRoot}
	committeePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		t.Fatal(err)
	}
	committeePublicKey = committeePublicKey[0:512]
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentValidator, rootHashes[index], id, from, to)
		from += 32
		to += 32
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, ids)
	for _, id := range ids {
		temp, ok := gotM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 32 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 32, len(temp))
		}
	}
	for id, wants := range wantM {
		flag := false
		for _, want := range wants {
			for _, got := range gotM[id] {
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
	wantMs = append(wantMs, wantM)
	rootHashesByHeight = append(rootHashesByHeight, rootHashes[8])
	from = 256
	to = 260
	for i := 1; i < maxHeight; i++ {
		sDB, err := NewWithPrefixTrie(rootHashesByHeight[i-1], wrarperDB)
		if err != nil {
			t.Fatalf("height %+v, err %+v", i, err)
		}
		prevWantM := wantMs[i-1]
		newWantMState := make(map[common.Hash]*CommitteeState)
		for shardID, publicKeys := range prevWantM {
			for _, value := range publicKeys {
				key, _ := GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, value)
				committeeState := NewCommitteeStateWithValue(shardID, CurrentValidator, value)
				newWantMState[key] = committeeState
			}
		}
		newWantM := make(map[int][]incognitokey.CommitteePublicKey)
		newAddedMState := make(map[common.Hash]*CommitteeState)
		for _, shardID := range ids {
			tempCommitteePublicKey := committeePublicKey[from:to]
			for _, value := range tempCommitteePublicKey {
				key, _ := GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, value)
				committeeState := NewCommitteeStateWithValue(shardID, CurrentValidator, value)
				newAddedMState[key] = committeeState
			}
			from += 4
			to += 4
			prevWantMStateByShardID := make(map[common.Hash]*CommitteeState)
			for _, value := range prevWantM[shardID] {
				key, _ := GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, value)
				committeeState := NewCommitteeStateWithValue(shardID, CurrentValidator, value)
				prevWantMStateByShardID[key] = committeeState
			}
			count := 0
			for key, _ := range prevWantMStateByShardID {
				ok := sDB.MarkDeleteStateObject(CommitteeObjectType, key)
				if sDB.Error() != nil {
					t.Fatal(sDB.Error())
				}
				if !ok {
					t.Fatal("can't mark delete state object " + strconv.Itoa(i) + " " + strconv.Itoa(count))
				}
				delete(newWantMState, key)
				count++
				if count == 4 {
					break
				}
			}
			for k, v := range newAddedMState {
				err := sDB.SetStateObject(CommitteeObjectType, k, v)
				if err != nil {
					t.Fatal(err)
				}
				newWantMState[k] = v
			}
		}
		for _, v := range newWantMState {
			newWantM[v.ShardID()] = append(newWantM[v.ShardID()], v.CommitteePublicKey())
		}
		rootHash, err := sDB.Commit(true)
		if err != nil {
			t.Fatal(err)
		}
		err = sDB.Database().TrieDB().Commit(rootHash, false)
		if err != nil {
			t.Fatal(err)
		}
		rootHashesByHeight = append(rootHashesByHeight, rootHash)
		wantMs = append(wantMs, newWantM)
	}
	for index, rootHash := range rootHashesByHeight {
		tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
		if err != nil {
			t.Fatal(err)
		}
		gotM := tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, ids)
		for _, id := range ids {
			temp, ok := gotM[id]
			if !ok {
				t.Fatalf("getAllCommitteeState want shard %+v", id)
			}
			if len(temp) != 32 {
				t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 32, len(temp))
			}
		}
		for id, wants := range wantMs[index] {
			flag := false
			for _, want := range wants {
				for _, got := range gotM[id] {
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
	}
}

func TestStateDB_GetCurrentValidatorCommitteePublicKeyByShardIDState512EightShard(t *testing.T) {
	from, to := 0, 64
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentValidator, rootHashes[index], id, from, to)
		from += 64
		to += 64
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	for _, id := range ids {
		gotM := tempStateDB.getByShardIDCurrentValidatorState(id)
		if len(gotM) != 64 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 64, len(gotM))
		}
		for _, want := range wantM[id] {
			flag := false
			for _, got := range gotM {
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
}

// SUBSTITUTE VALIDATOR===============================================================
func TestStateDB_GetSubstituteValidatorCommitteeState(t *testing.T) {
	rootHash, m := storeCommitteeObjectOneShard(SubstituteValidator, emptyRoot, 0, 0, len(committeePublicKeys))
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

func TestStateDB_GetSubstituteValidatorCommitteePublicKeyByShardIDState512EightShard(t *testing.T) {
	from, to := 0, 64
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(SubstituteValidator, rootHashes[index], id, from, to)
		from += 64
		to += 64
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	for _, id := range ids {
		gotM := tempStateDB.getByShardIDSubstituteValidatorState(id)
		if len(gotM) != 64 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 64, len(gotM))
		}
		for _, want := range wantM[id] {
			flag := false
			for _, got := range gotM {
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
}

func TestStateDB_GetAllSubstituteValidatorCommitteeKey512OneShard(t *testing.T) {
	from, to := 0, 512
	rootHash, m := storeCommitteeObjectOneShard(SubstituteValidator, emptyRoot, 0, from, to)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllValidatorCommitteePublicKey(SubstituteValidator, []int{0})
	gotMShard0, ok := gotM[0]
	if !ok {
		t.Fatalf("getAllCommitteeState want shard %+v", 0)
	}
	if len(gotMShard0) != to-from {
		t.Fatalf("getAllCommitteeState want key length %+v but got %+v", to-from, len(gotM))
	}
	for _, want := range m {
		flag := false
		for _, got := range gotMShard0 {
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

func TestStateDB_GetAllSubstituteValidatorCommitteeKey512EightShard(t *testing.T) {
	from, to := 0, 64
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(SubstituteValidator, rootHashes[index], id, from, to)
		from += 64
		to += 64
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllValidatorCommitteePublicKey(SubstituteValidator, ids)
	for _, id := range ids {
		temp, ok := gotM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 64 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 64, len(temp))
		}
	}
	for id, wants := range wantM {
		flag := false
		for _, want := range wants {
			for _, got := range gotM[id] {
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
}

func TestStateDB_AllSubstituteValidatorCommitteePublicKey512EightShardMultipleRootHash(t *testing.T) {
	from, to := 0, 32
	maxHeight := 8
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	wantMs := []map[int][]incognitokey.CommitteePublicKey{}
	rootHashesByHeight := []common.Hash{}
	rootHashes := []common.Hash{emptyRoot}
	committeePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		t.Fatal(err)
	}
	committeePublicKey = committeePublicKey[0:512]
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(SubstituteValidator, rootHashes[index], id, from, to)
		from += 32
		to += 32
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err, tempStateDB)
	}
	gotM := tempStateDB.getAllValidatorCommitteePublicKey(SubstituteValidator, ids)
	for _, id := range ids {
		temp, ok := gotM[id]
		if !ok {
			t.Fatalf("getAllCommitteeState want shard %+v", id)
		}
		if len(temp) != 32 {
			t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 32, len(temp))
		}
	}
	for id, wants := range wantM {
		flag := false
		for _, want := range wants {
			for _, got := range gotM[id] {
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
	wantMs = append(wantMs, wantM)
	rootHashesByHeight = append(rootHashesByHeight, rootHashes[8])
	from = 256
	to = 260
	for i := 1; i < maxHeight; i++ {
		sDB, err := NewWithPrefixTrie(rootHashesByHeight[i-1], wrarperDB)
		if err != nil {
			t.Fatalf("height %+v, err %+v", i, err)
		}
		prevWantM := wantMs[i-1]
		newWantMState := make(map[common.Hash]*CommitteeState)
		for shardID, publicKeys := range prevWantM {
			for _, value := range publicKeys {
				key, _ := GenerateCommitteeObjectKeyWithRole(SubstituteValidator, shardID, value)
				committeeState := NewCommitteeStateWithValue(shardID, SubstituteValidator, value)
				newWantMState[key] = committeeState
			}
		}
		newWantM := make(map[int][]incognitokey.CommitteePublicKey)
		newAddedMState := make(map[common.Hash]*CommitteeState)
		for _, shardID := range ids {
			tempCommitteePublicKey := committeePublicKey[from:to]
			for _, value := range tempCommitteePublicKey {
				key, _ := GenerateCommitteeObjectKeyWithRole(SubstituteValidator, shardID, value)
				committeeState := NewCommitteeStateWithValue(shardID, SubstituteValidator, value)
				newAddedMState[key] = committeeState
			}
			from += 4
			to += 4
			prevWantMStateByShardID := make(map[common.Hash]*CommitteeState)
			for _, value := range prevWantM[shardID] {
				key, _ := GenerateCommitteeObjectKeyWithRole(SubstituteValidator, shardID, value)
				committeeState := NewCommitteeStateWithValue(shardID, SubstituteValidator, value)
				prevWantMStateByShardID[key] = committeeState
			}
			count := 0
			for key, _ := range prevWantMStateByShardID {
				ok := sDB.MarkDeleteStateObject(CommitteeObjectType, key)
				if sDB.Error() != nil {
					t.Fatal(sDB.Error())
				}
				if !ok {
					t.Fatal("can't mark delete state object " + strconv.Itoa(i) + " " + strconv.Itoa(count))
				}
				delete(newWantMState, key)
				count++
				if count == 4 {
					break
				}
			}
			for k, v := range newAddedMState {
				err := sDB.SetStateObject(CommitteeObjectType, k, v)
				if err != nil {
					t.Fatal(err)
				}
				newWantMState[k] = v
			}
		}
		for _, v := range newWantMState {
			newWantM[v.ShardID()] = append(newWantM[v.ShardID()], v.CommitteePublicKey())
		}
		rootHash, err := sDB.Commit(true)
		if err != nil {
			t.Fatal(err)
		}
		err = sDB.Database().TrieDB().Commit(rootHash, false)
		if err != nil {
			t.Fatal(err)
		}
		rootHashesByHeight = append(rootHashesByHeight, rootHash)
		wantMs = append(wantMs, newWantM)
	}
	for index, rootHash := range rootHashesByHeight {
		tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
		if err != nil {
			t.Fatal(err)
		}
		gotM := tempStateDB.getAllValidatorCommitteePublicKey(SubstituteValidator, ids)
		for _, id := range ids {
			temp, ok := gotM[id]
			if !ok {
				t.Fatalf("getAllCommitteeState want shard %+v", id)
			}
			if len(temp) != 32 {
				t.Fatalf("getAllCommitteeState want key length %+v but got %+v", 32, len(temp))
			}
		}
		for id, wants := range wantMs[index] {
			flag := false
			for _, want := range wants {
				for _, got := range gotM[id] {
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
	}
}

// SUBSTITUTE VALIDATOR===============================================================

func BenchmarkStateDB_GetCurrentValidatorCommitteePublicKey512EightShard(b *testing.B) {
	from, to := 0, 64
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentValidator, rootHashes[index], id, from, to)
		from += 64
		to += 64
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		for _, id := range ids {
			tempStateDB.getByShardIDCurrentValidatorState(id)
		}
	}
}

func BenchmarkStateDB_GetAllCurrentCandidateCommitteePublicKey512EightShard(b *testing.B) {
	from, to := 0, 64
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	wantM := make(map[int][]incognitokey.CommitteePublicKey)
	rootHashes := []common.Hash{emptyRoot}
	for index, id := range ids {
		tempRootHash, tempM := storeCommitteeObjectOneShard(CurrentValidator, rootHashes[index], id, from, to)
		from += 64
		to += 64
		rootHashes = append(rootHashes, tempRootHash)
		for _, v := range tempM {
			wantM[id] = append(wantM[id], v.CommitteePublicKey())
		}
	}
	tempStateDB, err := NewWithPrefixTrie(rootHashes[8], wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, ids)
	}
}
func BenchmarkStateDB_GetAllCurrentCandidateCommitteePublicKey512OneShard(b *testing.B) {
	from, to := 0, 512
	rootHash, _ := storeCommitteeObjectOneShard(CurrentValidator, emptyRoot, 0, from, to)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		tempStateDB.getAllValidatorCommitteePublicKey(CurrentValidator, []int{0})
	}
}

func BenchmarkStateDB_GetCommitteeState512OneShard(b *testing.B) {
	rootHash, m := storeCommitteeObjectOneShard(CurrentValidator, emptyRoot, 0, 0, 512)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		for key, want := range m {
			got, has, err := tempStateDB.getCommitteeState(key)
			if err != nil {
				panic(err)
			}
			if !has {
				panic(key)
			}
			if !reflect.DeepEqual(got, want) {
				panic(key)
			}
		}
	}
}

func BenchmarkStateDB_GetCommitteeState256OneShard(b *testing.B) {
	rootHash, m := storeCommitteeObjectOneShard(CurrentValidator, emptyRoot, 0, 0, 256)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		for key, want := range m {
			got, has, err := tempStateDB.getCommitteeState(key)
			if err != nil {
				panic(err)
			}
			if !has {
				panic(key)
			}
			if !reflect.DeepEqual(got, want) {
				panic(key)
			}
		}
	}
}

func BenchmarkStateDB_GetCommitteeState1In1(b *testing.B) {
	key := common.Hash{}
	shardID := 0
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		panic(err)
	}
	sampleCommitteePublicKey := tempCommitteePublicKey[0]
	key, _ = GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, sampleCommitteePublicKey)
	committeeState := NewCommitteeStateWithValue(shardID, CurrentValidator, sampleCommitteePublicKey)
	sDB, err := NewWithPrefixTrie(emptyRoot, wrarperDB)
	if err != nil {
		panic(err)
	}
	sDB.SetStateObject(CommitteeObjectType, key, committeeState)
	rootHash, err := sDB.Commit(true)
	if err != nil {
		panic(err)
	}
	err = sDB.Database().TrieDB().Commit(rootHash, false)
	if err != nil {
		panic(err)
	}
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		tempStateDB.getCommitteeState(key)
	}
}
func BenchmarkStateDB_GetCommitteeState1In64(b *testing.B) {
	key := common.Hash{}
	shardID := 0
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		panic(err)
	}
	sampleCommitteePublicKey := tempCommitteePublicKey[0]
	key, _ = GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, sampleCommitteePublicKey)
	rootHash, _ := storeCommitteeObjectOneShard(CurrentValidator, emptyRoot, 0, 0, 64)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		tempStateDB.getCommitteeState(key)
	}
}
func BenchmarkStateDB_GetCommitteeState1In256(b *testing.B) {
	key := common.Hash{}
	shardID := 0
	tempCommitteePublicKey, err := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	if err != nil {
		panic(err)
	}
	sampleCommitteePublicKey := tempCommitteePublicKey[0]
	key, _ = GenerateCommitteeObjectKeyWithRole(CurrentValidator, shardID, sampleCommitteePublicKey)
	rootHash, _ := storeCommitteeObjectOneShard(CurrentValidator, emptyRoot, 0, 0, 256)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		tempStateDB.getCommitteeState(key)
	}
}
