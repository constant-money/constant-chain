package portaltokens

import (
	"testing"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
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
