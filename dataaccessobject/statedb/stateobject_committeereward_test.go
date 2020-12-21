package statedb

import (
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
)

func storeCommitteeReward(initRoot common.Hash, warperDB DatabaseAccessWarper) (common.Hash, map[common.Hash]*CommitteeRewardState, map[string]map[common.Hash]uint64) {
	mState := make(map[common.Hash]*CommitteeRewardState)
	wantM := make(map[string]map[common.Hash]uint64)
	for index, value := range incognitoPublicKeys {
		key, _ := GenerateCommitteeRewardObjectKey(value)
		reward := testGenerateTokenMapWithAmount()
		rewardReceiverState := NewCommitteeRewardStateWithValue(reward, incognitoPublicKeys[index])
		mState[key] = rewardReceiverState
		wantM[value] = reward
	}
	sDB, err := NewWithPrefixTrie(initRoot, warperDB)
	if err != nil {
		panic(err)
	}
	for key, value := range mState {
		sDB.SetStateObject(CommitteeRewardObjectType, key, value)
	}
	rootHash, err := sDB.Commit(true)
	if err != nil {
		panic(err)
	}
	err = sDB.Database().TrieDB().Commit(rootHash, false)
	if err != nil {
		panic(err)
	}
	return rootHash, mState, wantM
}

func TestStateDB_GetAllCommitteeRewardState(t *testing.T) {
	rootHash, wantM, _ := storeCommitteeReward(emptyRoot, wrarperDB)
	tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
	if err != nil || tempStateDB == nil {
		t.Fatal(err)
	}
	for k, v := range wantM {
		gotM, has, err := tempStateDB.getCommitteeRewardState(k)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatal(has)
		}
		if !reflect.DeepEqual(v, gotM) {
			t.Fatalf("want %+v but got %+v", v, gotM)
		}
	}
}

func TestStateDB_StoreAndGetRewardReceiver(t *testing.T) {
	var err error = nil
	key, _ := GenerateCommitteeRewardObjectKey(incognitoPublicKeys[0])
	key2, _ := GenerateCommitteeRewardObjectKey(incognitoPublicKeys[1])
	rewardReceiverState := NewCommitteeRewardStateWithValue(testGenerateTokenMapWithAmount(), incognitoPublicKeys[0])
	sDB, err := NewWithPrefixTrie(emptyRoot, wrarperDB)
	if err != nil {
		panic(err)
	}
	err = sDB.SetStateObject(CommitteeRewardObjectType, key, rewardReceiverState)
	if err != nil {
		t.Fatal(err)
	}
	err = sDB.SetStateObject(CommitteeRewardObjectType, key, "committee reward")
	if err == nil {
		t.Fatal("expect error")
	}
	err = sDB.SetStateObject(CommitteeRewardObjectType, key, []byte("committee reward"))
	if err == nil {
		t.Fatal("expect error")
	}
	err = sDB.SetStateObject(CommitteeRewardObjectType, key2, []byte("committee reward"))
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
	got, has, err := tempStateDB.getCommitteeRewardState(key)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal(has)
	}
	if !reflect.DeepEqual(got, rewardReceiverState) {
		t.Fatalf("want value %+v but got %+v", rewardReceiverState, got)
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

func TestStateDB_GetAllRewardReceiverStateMultipleRootHash(t *testing.T) {
	offset := 9
	maxHeight := int(len(incognitoPublicKeys) / offset)
	rootHashes := []common.Hash{emptyRoot}
	wantMs := []map[string]map[common.Hash]uint64{}
	for i := 0; i < maxHeight; i++ {
		sDB, err := NewWithPrefixTrie(rootHashes[i], wrarperDB)
		if err != nil || sDB == nil {
			t.Fatal(err)
		}
		tempKeys := incognitoPublicKeys[i*9 : (i+1)*9]
		tempM := make(map[string]map[common.Hash]uint64)
		prevWantM := make(map[string]map[common.Hash]uint64)
		if i != 0 {
			prevWantM = wantMs[i-1]
		}
		for k, v := range prevWantM {
			tempM[k] = v
		}
		for _, value := range tempKeys {
			key, _ := GenerateCommitteeRewardObjectKey(value)
			reward := testGenerateTokenMapWithAmount()
			rewardReceiverState := NewCommitteeRewardStateWithValue(reward, value)
			err := sDB.SetStateObject(CommitteeRewardObjectType, key, rewardReceiverState)
			if err != nil {
				t.Fatal(err)
			}
			tempM[value] = reward
		}
		rootHash, err := sDB.Commit(true)
		if err != nil {
			t.Fatal(err)
		}
		err = sDB.Database().TrieDB().Commit(rootHash, false)
		if err != nil {
			t.Fatal(err)
		}
		wantMs = append(wantMs, tempM)
		rootHashes = append(rootHashes, rootHash)
	}
	for index, rootHash := range rootHashes[1:] {
		wantM := wantMs[index]
		tempStateDB, err := NewWithPrefixTrie(rootHash, wrarperDB)
		if err != nil || tempStateDB == nil {
			t.Fatal(err)
		}
		gotM := tempStateDB.getAllCommitteeReward()
		for k, v1 := range gotM {
			if v2, ok := wantM[k]; !ok {
				t.Fatalf("want %+v but get nothing", k)
			} else {
				if !reflect.DeepEqual(v2, v1) {
					t.Fatalf("want %+v but got %+v", v2, v1)
				}
			}
		}
	}
}
