package statedb

import (
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

func TestStoreAndGetBeaconCommittee(t *testing.T) {
	number := 20
	beaconCommittees := committeePublicKeys[:number]
	beaconCommitteesStruct, _ := incognitokey.CommitteeBase58KeyListToStruct(beaconCommittees)
	rewardReceiver := make(map[string]string)
	autoStaking := make(map[string]bool)
	for index, beaconCommittee := range beaconCommitteesStruct {
		incPublicKey := beaconCommittee.GetIncKeyBase58()
		paymentAddress := receiverPaymentAddress[index]
		rewardReceiver[incPublicKey] = paymentAddress
		autoStaking[beaconCommittees[index]] = true
	}
	sDB, _ := NewWithPrefixTrie(emptyRoot, wrarperDB)
	err := StoreBeaconCommittee(sDB, beaconCommitteesStruct)
	if err != nil {
		t.Fatal(err)
	}
	rootHash, _ := sDB.Commit(true)
	err1 := sDB.Database().TrieDB().Commit(rootHash, false)
	if err1 != nil {
		t.Fatal(err1)
	}
	gotBeaconCommitteeStruct := GetBeaconCommittee(sDB)
	if len(gotBeaconCommitteeStruct) != number {
		t.Fatalf("expect number of committee %+v, but got %+v", len(gotBeaconCommitteeStruct), number)
	}
	for index, wantC := range beaconCommitteesStruct {
		if !wantC.IsEqual(gotBeaconCommitteeStruct[index]) {
			t.Fatalf("want %+v, got %+v", wantC, gotBeaconCommitteeStruct[index])
		}
	}
}

func TestStoreAndGetShardCommittee(t *testing.T) {
	number := 20
	shardID := byte(0)
	shardCommittees := committeePublicKeys[:number]
	shardCommitteesStruct, _ := incognitokey.CommitteeBase58KeyListToStruct(shardCommittees)
	rewardReceiver := make(map[string]string)
	autoStaking := make(map[string]bool)
	for index, beaconCommittee := range shardCommitteesStruct {
		incPublicKey := beaconCommittee.GetIncKeyBase58()
		paymentAddress := receiverPaymentAddress[index]
		rewardReceiver[incPublicKey] = paymentAddress
		autoStaking[shardCommittees[index]] = true
	}
	sDB, _ := NewWithPrefixTrie(emptyRoot, wrarperDB)
	err := StoreOneShardCommittee(sDB, shardID, shardCommitteesStruct)
	if err != nil {
		t.Fatal(err)
	}
	rootHash, _ := sDB.Commit(true)
	err1 := sDB.Database().TrieDB().Commit(rootHash, false)
	if err1 != nil {
		t.Fatal(err1)
	}
	gotShardCommitteeStruct := GetOneShardCommittee(sDB, shardID)
	if len(gotShardCommitteeStruct) != number {
		t.Fatalf("expect number of committee %+v, but got %+v", len(gotShardCommitteeStruct), number)
	}
	for index, wantC := range shardCommitteesStruct {
		if !wantC.IsEqual(gotShardCommitteeStruct[index]) {
			t.Fatalf("want %+v, got %+v", wantC, gotShardCommitteeStruct[index])
		}
	}
}

func TestDeleteOneShardCommittee(t *testing.T) {
	number := 20
	split := 10
	shardID := byte(0)
	shardCommittees := committeePublicKeys[:number]
	shardCommitteesStruct, _ := incognitokey.CommitteeBase58KeyListToStruct(shardCommittees)
	rewardReceiver := make(map[string]string)
	autoStaking := make(map[string]bool)
	for index, beaconCommittee := range shardCommitteesStruct {
		incPublicKey := beaconCommittee.GetIncKeyBase58()
		paymentAddress := receiverPaymentAddress[index]
		rewardReceiver[incPublicKey] = paymentAddress
		autoStaking[shardCommittees[index]] = true
	}
	sDB, _ := NewWithPrefixTrie(emptyRoot, wrarperDB)
	err := StoreOneShardCommittee(sDB, shardID, shardCommitteesStruct)
	if err != nil {
		t.Fatal(err)
	}
	rootHash, _ := sDB.Commit(true)
	err1 := sDB.Database().TrieDB().Commit(rootHash, false)
	if err1 != nil {
		t.Fatal(err1)
	}
	deletedShardCommittee := shardCommitteesStruct[:split]
	remainedShardCommittee := shardCommitteesStruct[split:]
	err2 := DeleteOneShardCommittee(sDB, shardID, deletedShardCommittee)
	if err2 != nil {
		t.Fatal(err2)
	}
	rootHash2, _ := sDB.Commit(true)
	err3 := sDB.Database().TrieDB().Commit(rootHash2, false)
	if err3 != nil {
		t.Fatal(err3)
	}
	gotShardCommitteeStruct := GetOneShardCommittee(sDB, shardID)
	if len(gotShardCommitteeStruct) != len(remainedShardCommittee) {
		t.Fatalf("expect number of committee %+v, but got %+v", len(gotShardCommitteeStruct), len(remainedShardCommittee))
	}
	for index, wantC := range remainedShardCommittee {
		if !wantC.IsEqual(gotShardCommitteeStruct[index]) {
			t.Fatalf("want %+v, got %+v", wantC, gotShardCommitteeStruct[index])
		}
	}
	for _, wantC := range deletedShardCommittee {
		for _, gotC := range gotShardCommitteeStruct {
			if wantC.IsEqual(gotC) {
				t.Fatalf("want %+v, got %+v", wantC, gotC)
			}
		}
	}
}

func TestDeleteBeaconCommittee(t *testing.T) {
	number := 20
	split := 10
	beaconCommittees := committeePublicKeys[:number]
	beaconCommitteesStruct, _ := incognitokey.CommitteeBase58KeyListToStruct(beaconCommittees)
	rewardReceiver := make(map[string]string)
	autoStaking := make(map[string]bool)
	for index, beaconCommittee := range beaconCommitteesStruct {
		incPublicKey := beaconCommittee.GetIncKeyBase58()
		paymentAddress := receiverPaymentAddress[index]
		rewardReceiver[incPublicKey] = paymentAddress
		autoStaking[beaconCommittees[index]] = true
	}
	sDB, _ := NewWithPrefixTrie(emptyRoot, wrarperDB)
	err := StoreBeaconCommittee(sDB, beaconCommitteesStruct)
	if err != nil {
		t.Fatal(err)
	}
	rootHash, _ := sDB.Commit(true)
	err1 := sDB.Database().TrieDB().Commit(rootHash, false)
	if err1 != nil {
		t.Fatal(err1)
	}
	deletedBeaconCommittee := beaconCommitteesStruct[:split]
	remainedBeaconCommittee := beaconCommitteesStruct[split:]
	err2 := DeleteBeaconCommittee(sDB, deletedBeaconCommittee)
	if err2 != nil {
		t.Fatal(err2)
	}
	rootHash2, _ := sDB.Commit(true)
	err3 := sDB.Database().TrieDB().Commit(rootHash2, false)
	if err3 != nil {
		t.Fatal(err3)
	}
	gotBeaconCommitteeStruct := GetBeaconCommittee(sDB)
	if len(gotBeaconCommitteeStruct) != len(remainedBeaconCommittee) {
		t.Fatalf("expect number of committee %+v, but got %+v", len(gotBeaconCommitteeStruct), len(remainedBeaconCommittee))
	}
	for index, wantC := range remainedBeaconCommittee {
		if !wantC.IsEqual(gotBeaconCommitteeStruct[index]) {
			t.Fatalf("want %+v, got %+v", wantC, gotBeaconCommitteeStruct[index])
		}
	}
	for _, wantC := range deletedBeaconCommittee {
		for _, gotC := range gotBeaconCommitteeStruct {
			if wantC.IsEqual(gotC) {
				t.Fatalf("want %+v, got %+v", wantC, gotC)
			}
		}
	}
}

func TestStoreAndGetStakerInfo(t *testing.T) {
	number := 20
	shardID := byte(0)
	shardCommittees := committeePublicKeys[:number]
	shardCommitteesStruct, _ := incognitokey.CommitteeBase58KeyListToStruct(shardCommittees)
	rewardReceiver := make(map[string]privacy.PaymentAddress)
	autoStaking := make(map[string]bool)
	stakingTx := make(map[string]common.Hash)
	for index, beaconCommittee := range shardCommitteesStruct {
		incPublicKey := beaconCommittee.GetIncKeyBase58()
		paymentAddress := receiverPaymentAddress[index]
		wl, err := wallet.Base58CheckDeserialize(paymentAddress)
		if err != nil {
			t.Fatal(err)
		}
		rewardReceiver[incPublicKey] = wl.KeySet.PaymentAddress
		autoStaking[shardCommittees[index]] = true
		stakingTx[shardCommittees[index]] = common.HashH([]byte{0})
	}
	sDB, _ := NewWithPrefixTrie(emptyRoot, wrarperDB)
	err := StoreOneShardCommittee(sDB, shardID, shardCommitteesStruct)
	if err != nil {
		t.Fatal(err)
	}
	err = StoreStakerInfo(sDB, shardCommitteesStruct, rewardReceiver, autoStaking, stakingTx)
	if err != nil {
		t.Fatal(err)
	}
	rootHash, _ := sDB.Commit(true)
	err1 := sDB.Database().TrieDB().Commit(rootHash, false)
	if err1 != nil {
		t.Fatal(err1)
	}
	gotShardCommitteeStruct := GetOneShardCommittee(sDB, shardID)
	if len(gotShardCommitteeStruct) != number {
		t.Fatalf("expect number of committee %+v, but got %+v", len(gotShardCommitteeStruct), number)
	}
	for index, wantC := range shardCommitteesStruct {
		if !wantC.IsEqual(gotShardCommitteeStruct[index]) {
			t.Fatalf("want %+v, got %+v", wantC, gotShardCommitteeStruct[index])
		}
		cString, _ := wantC.ToBase58()
		s, ok, err := GetStakerInfo(sDB, cString)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("Can not get committee info")
		}
		if s == nil {
			t.Fatal("wtf")
		}
	}
}
