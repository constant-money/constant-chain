package portaltokens

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/blockcypher/gobcy"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
)

func insertUnshieldIDIntoStateDB(waitingUnshieldState map[string]*statedb.WaitingUnshieldRequest,
	tokenID string, remoteAddress string, unshieldID string, amount uint64, beaconHeight uint64) {
	key := statedb.GenerateWaitingUnshieldRequestObjectKey(tokenID, unshieldID).String()
	waitingUnshieldState[key] = statedb.NewWaitingUnshieldRequestStateWithValue(remoteAddress, amount, unshieldID, beaconHeight)
}

func insertUTXOIntoStateDB(utxos map[string]*statedb.UTXO, key string, amount uint64) {
	curUTXO := &statedb.UTXO{}
	curUTXO.SetOutputAmount(amount)
	utxos[key] = curUTXO
}

func printBroadcastTxs(t *testing.T, broadcastTxs []*BroadcastTx) {
	t.Logf("Len of broadcast txs: %v\n", len(broadcastTxs))
	for i, tx := range broadcastTxs {
		t.Logf("+ Broadcast Tx %v\n", i)
		for idx, utxo := range tx.UTXOs {
			t.Logf("++ UTXO %v: %v\n", idx, utxo.GetOutputAmount())
		}
		t.Logf("+ Unshield IDs: %v \n", tx.UnshieldIDs)
	}
}

func TestChooseUnshieldIDsFromCandidates(t *testing.T) {
	p := &PortalToken{}

	tokenID := "btc"
	waitingUnshieldState := map[string]*statedb.WaitingUnshieldRequest{}
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_1", "unshield_1", 1000, 1)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_2", "unshield_2", 500, 2)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_3", "unshield_3", 2000, 3)

	// Not enough UTXO
	utxos := map[string]*statedb.UTXO{}
	insertUTXOIntoStateDB(utxos, "utxo_1", 900)

	broadcastTxs := p.ChooseUnshieldIDsFromCandidates(utxos, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// Broadcast a part of unshield requests
	utxos = map[string]*statedb.UTXO{}
	insertUTXOIntoStateDB(utxos, "utxo_2", 1500)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// Broadcast all unshield requests
	utxos = map[string]*statedb.UTXO{}
	insertUTXOIntoStateDB(utxos, "utxo_3", 5000)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// First unshield request need multiple UTXOs
	waitingUnshieldState = map[string]*statedb.WaitingUnshieldRequest{}
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_4", "unshield_4", 2000, 4)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_5", "unshield_5", 1000, 5)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_6", "unshield_6", 1500, 6)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_7", "unshield_7", 10000, 7)

	utxos = map[string]*statedb.UTXO{}
	insertUTXOIntoStateDB(utxos, "utxo_4", 500)
	insertUTXOIntoStateDB(utxos, "utxo_5", 1600)
	insertUTXOIntoStateDB(utxos, "utxo_6", 1000)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// Broadcast multiple txs
	waitingUnshieldState = map[string]*statedb.WaitingUnshieldRequest{}
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_8", "unshield_8", 2000, 8)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_9", "unshield_9", 1000, 9)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_10", "unshield_10", 200, 10)
	insertUnshieldIDIntoStateDB(waitingUnshieldState, tokenID, "remoteAddr_11", "unshield_11", 100, 11)

	utxos = map[string]*statedb.UTXO{}
	insertUTXOIntoStateDB(utxos, "utxo_7", 150)
	insertUTXOIntoStateDB(utxos, "utxo_8", 150)
	insertUTXOIntoStateDB(utxos, "utxo_9", 1000)
	insertUTXOIntoStateDB(utxos, "utxo_10", 1600)
	insertUTXOIntoStateDB(utxos, "utxo_11", 1000)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)
}

func getBlockCypherAPI(networkName string) gobcy.API {
	//explicitly
	bc := gobcy.API{}
	bc.Token = "029727206f7e4c8fb19301e4629c5793"
	bc.Coin = "btc"        //options: "btc","bcy","ltc","doge"
	bc.Chain = networkName //depending on coin: "main","test3","test"
	return bc
}

func buildBTCBlockFromCypher(networkName string, blkHeight int) (*btcutil.Block, error) {
	bc := getBlockCypherAPI(networkName)
	cypherBlock, err := bc.GetBlock(blkHeight, "", nil)
	if err != nil {
		return nil, err
	}
	prevBlkHash, _ := chainhash.NewHashFromStr(cypherBlock.PrevBlock)
	merkleRoot, _ := chainhash.NewHashFromStr(cypherBlock.MerkleRoot)
	msgBlk := wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    int32(cypherBlock.Ver),
			PrevBlock:  *prevBlkHash,
			MerkleRoot: *merkleRoot,
			Timestamp:  cypherBlock.Time,
			Bits:       uint32(cypherBlock.Bits),
			Nonce:      uint32(cypherBlock.Nonce),
		},
		Transactions: []*wire.MsgTx{},
	}
	blk := btcutil.NewBlock(&msgBlk)
	blk.SetHeight(int32(blkHeight))
	return blk, nil
}

func setGenesisBlockToChainParams(networkName string, genesisBlkHeight int) (*chaincfg.Params, error) {
	blk, err := buildBTCBlockFromCypher(networkName, genesisBlkHeight)
	if err != nil {
		return nil, err
	}

	// chainParams := chaincfg.MainNetParams
	chainParams := chaincfg.TestNet3Params
	chainParams.GenesisBlock = blk.MsgBlock()
	chainParams.GenesisHash = blk.Hash()
	return &chainParams, nil
}

func tearDownRelayBTCHeadersTest(dbName string, t *testing.T) {
	testDbRoot := "btcdbs"
	t.Logf("Tearing down RelayBTCHeadersTest...")
	dbPath := filepath.Join(testDbRoot, dbName)
	os.RemoveAll(dbPath)
	os.RemoveAll(testDbRoot)
}

func TestParseAndVerifyProof(t *testing.T) {
	expectedMultisigAddress := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"

	networkName := "test3"
	genesisBlockHeight := int(1719640)

	chainParams, err := setGenesisBlockToChainParams(networkName, genesisBlockHeight)
	if err != nil {
		t.Errorf("Could not set genesis block to chain params with err: %v", err)
		return
	}

	dbName := "btc-blocks-test"
	btcChain1, err := btcrelaying.GetChainV2(dbName, chainParams, int32(genesisBlockHeight))
	defer tearDownRelayBTCHeadersTest(dbName, t)
	if err != nil {
		t.Errorf("Could not get chain instance with err: %v", err)
		return
	}

	for i := genesisBlockHeight + 1; i <= genesisBlockHeight+10; i++ {
		blk, err := buildBTCBlockFromCypher(networkName, i)
		if err != nil {
			t.Errorf("buildBTCBlockFromCypher fail on block %v: %v\n", i, err)
			return
		}
		isMainChain, isOrphan, err := btcChain1.ProcessBlockV2(blk, 0)
		if err != nil {
			t.Errorf("ProcessBlock fail on block %v: %v\n", i, err)
			return
		}
		if isOrphan {
			t.Errorf("ProcessBlock incorrectly returned block %v "+
				"is an orphan\n", i)
			return
		}
		t.Logf("Block %s (%d) is on main chain: %t\n", blk.Hash(), blk.Height(), isMainChain)
		time.Sleep(1 * time.Second)
	}

	t.Logf("Session 1: best block hash %s and block height %d\n", btcChain1.BestSnapshot().Hash.String(), btcChain1.BestSnapshot().Height)

	p := PortalBTCTokenProcessor{}

	shieldingIncAddress := "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci"
	expectedMemo := p.GetExpectedMemoForShielding(shieldingIncAddress)
	proof := ""

	p.ParseAndVerifyProof(proof, btcChain1, expectedMemo, expectedMultisigAddress)

	btcChain1.GetDB().Close()
}
