package rawdbv2_test

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/incdb"
	_ "github.com/incognitochain/incognito-chain/incdb/lvdb"
	"io/ioutil"
	"os"
	"testing"
)

var (
	beaconBlocks       []*blockchain.BeaconBlock
	forkedBeaconBlock1 = blockchain.NewBeaconBlock()
	forkedBeaconBlock2 = blockchain.NewBeaconBlock()
	randomBeaconBlock1 = blockchain.NewBeaconBlock()
	randomBeaconBlock2 = blockchain.NewBeaconBlock()
	max                = 5
	db                 incdb.Database
)
var _ = func() (_ struct{}) {
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_rawdbv2")
	if err != nil {
		panic(err)
	}
	db, err = incdb.Open("leveldb", dbPath)
	if err != nil {
		panic(err)
	}
	for i := 0; i < max; i++ {
		beaconBlock := blockchain.NewBeaconBlock()
		beaconBlock.Header.Height = uint64(i)
		if i != 0 {
			beaconBlock.Header.PreviousBlockHash = beaconBlocks[i-1].Header.Hash()
		}
		beaconBlocks = append(beaconBlocks, beaconBlock)
	}

	forkedBeaconBlock1.Header.Height = 1
	forkedBeaconBlock1.Header.Version = 2
	forkedBeaconBlock2.Header.Height = 2
	forkedBeaconBlock2.Header.Version = 2
	forkedBeaconBlock2.Header.PreviousBlockHash = forkedBeaconBlock1.Header.Hash()

	randomBeaconBlock1.Header.Height = 10001
	randomBeaconBlock2.Header.Height = 10002
	return
}()

func resetDatabase() {
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_rawdbv2")
	if err != nil {
		panic(err)
	}
	db, err = incdb.Open("leveldb", dbPath)
	if err != nil {
		panic(err)
	}
}
func storeBeaconBlock() error {
	resetDatabase()
	for i := 0; i < max; i++ {
		err := rawdbv2.StoreBeaconBlockByHash(db, uint64(i), beaconBlocks[i].Header.Hash(), beaconBlocks[i])
		if err != nil {
			return err
		}
	}
	err := rawdbv2.StoreBeaconBlockByHash(db, forkedBeaconBlock1.Header.Height, forkedBeaconBlock1.Header.Hash(), forkedBeaconBlock1)
	if err != nil {
		return err
	}
	err1 := rawdbv2.StoreBeaconBlockByHash(db, forkedBeaconBlock2.Header.Height, forkedBeaconBlock2.Header.Hash(), forkedBeaconBlock2)
	if err1 != nil {
		return err1
	}
	for i := 0; i < max; i++ {
		err := rawdbv2.StoreBeaconBlockIndex(db, uint64(i), beaconBlocks[i].Header.Hash())
		if err != nil {
			return err
		}
	}
	err2 := rawdbv2.StoreBeaconBlockIndex(db, forkedBeaconBlock1.Header.Height, forkedBeaconBlock1.Header.Hash())
	if err2 != nil {
		return err2
	}
	err3 := rawdbv2.StoreBeaconBlockIndex(db, forkedBeaconBlock2.Header.Height, forkedBeaconBlock2.Header.Hash())
	if err3 != nil {
		return err3
	}
	return nil
}

func TestStoreBeaconBlock(t *testing.T) {
	resetDatabase()
	for i := 0; i < max; i++ {
		err := rawdbv2.StoreBeaconBlockByHash(db, uint64(i), beaconBlocks[i].Header.Hash(), beaconBlocks[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	err := rawdbv2.StoreBeaconBlockByHash(db, forkedBeaconBlock1.Header.Height, forkedBeaconBlock1.Header.Hash(), forkedBeaconBlock1)
	if err != nil {
		t.Fatal(err)
	}
	err1 := rawdbv2.StoreBeaconBlockByHash(db, forkedBeaconBlock2.Header.Height, forkedBeaconBlock2.Header.Hash(), forkedBeaconBlock2)
	if err1 != nil {
		t.Fatal(err1)
	}
}

func TestStoreBeaconBlockIndex(t *testing.T) {
	resetDatabase()
	for i := 0; i < max; i++ {
		err := rawdbv2.StoreBeaconBlockIndex(db, uint64(i), beaconBlocks[i].Header.Hash())
		if err != nil {
			t.Fatal(err)
		}
	}
	err := rawdbv2.StoreBeaconBlockIndex(db, forkedBeaconBlock1.Header.Height, forkedBeaconBlock1.Header.Hash())
	if err != nil {
		t.Fatal(err)
	}
	err1 := rawdbv2.StoreBeaconBlockIndex(db, forkedBeaconBlock2.Header.Height, forkedBeaconBlock2.Header.Hash())
	if err1 != nil {
		t.Fatal(err1)
	}
}

func TestHasBeaconBlock(t *testing.T) {
	err := storeBeaconBlock()
	if err != nil {
		t.Fatal(err)
	}
	has, err := rawdbv2.HasBeaconBlock(db, randomBeaconBlock1.Header.Hash())
	if has {
		t.Fatalf("Want block %+v but got nothing", randomBeaconBlock1.Header.Hash())
	}
	if err != nil {
		t.Fatal(err)
	}
	has1, err := rawdbv2.HasBeaconBlock(db, randomBeaconBlock2.Header.Hash())
	if has1 {
		t.Fatalf("Want block %+v but got nothing", randomBeaconBlock2.Header.Hash())
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestHasBeaconBlockForked(t *testing.T) {
	err := storeBeaconBlock()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < max; i++ {
		has, err := rawdbv2.HasBeaconBlock(db, beaconBlocks[i].Header.Hash())
		if !has {
			t.Fatalf("Want block %+v but got nothing", beaconBlocks[i].Header.Hash())
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	has, err := rawdbv2.HasBeaconBlock(db, forkedBeaconBlock1.Header.Hash())
	if !has {
		t.Fatalf("Want block %+v but got nothing", forkedBeaconBlock1.Header.Hash())
	}
	if err != nil {
		t.Fatal(err)
	}
	has1, err := rawdbv2.HasBeaconBlock(db, forkedBeaconBlock2.Header.Hash())
	if !has1 {
		t.Fatalf("Want block %+v but got nothing", forkedBeaconBlock2.Header.Hash())
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetBeaconBlockByHash(t *testing.T) {
	err := storeBeaconBlock()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < max; i++ {
		res, err := rawdbv2.GetBeaconBlockByHash(db, beaconBlocks[i].Header.Hash())
		if err != nil {
			t.Fatal(err)
		}
		beaconBlock := blockchain.NewBeaconBlock()
		err = json.Unmarshal(res, beaconBlock)
		if err != nil {
			t.Fatal(err)
		}
		if beaconBlock.Header.Height != uint64(i) {
			t.Fatalf("want height %+v but got %+v", i, beaconBlock.Header.Height)
		}
		if beaconBlock.Header.Hash().String() != beaconBlocks[i].Header.Hash().String() {
			t.Fatalf("want hash %+v but got %+v", beaconBlocks[i].Header.Hash(), beaconBlock.Header.Hash())
		}
	}
	res1, err1 := rawdbv2.GetBeaconBlockByHash(db, forkedBeaconBlock1.Header.Hash())
	if err1 != nil {
		t.Fatal(err1)
	}
	beaconBlock1 := blockchain.NewBeaconBlock()
	err1 = json.Unmarshal(res1, beaconBlock1)
	if err1 != nil {
		t.Fatal(err1)
	}
	if beaconBlock1.Header.Height != uint64(1) {
		t.Fatalf("want height %+v but got %+v", 1, beaconBlock1.Header.Height)
	}
	if beaconBlock1.Header.Hash().String() != forkedBeaconBlock1.Header.Hash().String() {
		t.Fatalf("want hash %+v but got %+v", forkedBeaconBlock1.Header.Hash(), beaconBlock1.Header.Hash())
	}
	res, err2 := rawdbv2.GetBeaconBlockByHash(db, forkedBeaconBlock2.Header.Hash())
	if err2 != nil {
		t.Fatal(err2)
	}
	beaconBlock2 := blockchain.NewBeaconBlock()
	err2 = json.Unmarshal(res, beaconBlock2)
	if err2 != nil {
		t.Fatal(err2)
	}
	if beaconBlock2.Header.Height != uint64(2) {
		t.Fatalf("want height %+v but got %+v", 2, beaconBlock2.Header.Height)
	}
	if beaconBlock2.Header.Hash().String() != forkedBeaconBlock2.Header.Hash().String() {
		t.Fatalf("want hash %+v but got %+v", forkedBeaconBlock2.Header.Hash(), beaconBlock2.Header.Hash())
	}
}

func TestGetBeaconBlockByIndex(t *testing.T) {
	err := storeBeaconBlock()
	if err != nil {
		t.Fatal(err)
	}
	tempBeaconBlockHeight1, err := rawdbv2.GetBeaconBlockByIndex(db, 1)
	count1 := 0
	for h, data := range tempBeaconBlockHeight1 {
		beaconBlock := blockchain.NewBeaconBlock()
		err = json.Unmarshal(data, beaconBlock)
		if err != nil {
			t.Fatal(err)
		}
		if beaconBlock.Header.Height != uint64(1) {
			t.Fatalf("want height %+v but got %+v", 1, beaconBlock.Header.Height)
		}
		if beaconBlock.Header.Hash().String() != h.String() {
			t.Fatalf("want hash %+v but got %+v", beaconBlock.Header.Hash(), h)
		}
		count1++
	}
	if count1 != 2 {
		t.Fatalf("want %+v but got %+v block", 2, count1)
	}

	tempBeaconBlockHeight2, err := rawdbv2.GetBeaconBlockByIndex(db, 2)
	count2 := 0
	for h, data := range tempBeaconBlockHeight2 {
		beaconBlock := blockchain.NewBeaconBlock()
		err = json.Unmarshal(data, beaconBlock)
		if err != nil {
			t.Fatal(err)
		}
		if beaconBlock.Header.Height != uint64(2) {
			t.Fatalf("want height %+v but got %+v", 2, beaconBlock.Header.Height)
		}
		if beaconBlock.Header.Hash().String() != h.String() {
			t.Fatalf("want hash %+v but got %+v", beaconBlock.Header.Hash(), h)
		}
		count2++
	}
	if count2 != 2 {
		t.Fatalf("want %+v but got %+v block", 2, count2)
	}

	for i := 3; i < max; i++ {
		tempBeaconBlockHeight, err := rawdbv2.GetBeaconBlockByIndex(db, uint64(i))
		count := 0
		for h, data := range tempBeaconBlockHeight {
			beaconBlock := blockchain.NewBeaconBlock()
			err = json.Unmarshal(data, beaconBlock)
			if err != nil {
				t.Fatal(err)
			}
			if beaconBlock.Header.Height != uint64(i) {
				t.Fatalf("want height %+v but got %+v", i, beaconBlock.Header.Height)
			}
			if beaconBlock.Header.Hash().String() != h.String() {
				t.Fatalf("want hash %+v but got %+v", beaconBlock.Header.Hash(), h)
			}
			count++
		}
		if count != 1 {
			t.Fatalf("want %+v but got %+v block", 1, count2)
		}
	}
}

func TestGetIndexOfBeaconBlock(t *testing.T) {

}
