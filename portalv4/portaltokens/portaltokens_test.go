package portaltokens

import (
	"testing"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

func insertUnshieldIDIntoStateDB(waitingUnshieldState *statedb.WaitingUnshield, unshieldIDs []string, unshieldID string, amount uint64) []string {
	cur_unshield := &statedb.Unshield{}
	cur_unshield.SetAmount(amount)
	waitingUnshieldState.SetUnshield(unshieldID, cur_unshield)
	return append(unshieldIDs, unshieldID)
}

func insertUTXOIntoStateDB(utxos []*statedb.UTXO, amount uint64) []*statedb.UTXO {
	cur_utxo := &statedb.UTXO{}
	cur_utxo.SetOutputAmount(amount)
	return append(utxos, cur_utxo)
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

	waitingUnshieldState := statedb.NewWaitingUnshieldState()
	unshieldIDs := []string{}
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_1", 1000)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_2", 500)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_3", 2000)

	// Not enough UTXO
	utxos := []*statedb.UTXO{}
	utxos = insertUTXOIntoStateDB(utxos, 900)

	broadcastTxs := p.ChooseUnshieldIDsFromCandidates(utxos, unshieldIDs, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// Broadcast a part of unshield requests
	utxos = []*statedb.UTXO{}
	utxos = insertUTXOIntoStateDB(utxos, 1500)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, unshieldIDs, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// Broadcast all unshield requests
	utxos = []*statedb.UTXO{}
	utxos = insertUTXOIntoStateDB(utxos, 5000)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, unshieldIDs, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// First unshield request need multiple UTXOs
	waitingUnshieldState = statedb.NewWaitingUnshieldState()
	unshieldIDs = []string{}
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_4", 2000)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_5", 1000)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_6", 1500)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_7", 10000)

	utxos = []*statedb.UTXO{}
	utxos = insertUTXOIntoStateDB(utxos, 500)
	utxos = insertUTXOIntoStateDB(utxos, 1600)
	utxos = insertUTXOIntoStateDB(utxos, 1000)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, unshieldIDs, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)

	// Broadcast multiple txs
	waitingUnshieldState = statedb.NewWaitingUnshieldState()
	unshieldIDs = []string{}
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_8", 2000)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_9", 1000)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_10", 200)
	unshieldIDs = insertUnshieldIDIntoStateDB(waitingUnshieldState, unshieldIDs, "unshield_11", 100)

	utxos = []*statedb.UTXO{}
	utxos = insertUTXOIntoStateDB(utxos, 150)
	utxos = insertUTXOIntoStateDB(utxos, 150)
	utxos = insertUTXOIntoStateDB(utxos, 1000)
	utxos = insertUTXOIntoStateDB(utxos, 1600)
	utxos = insertUTXOIntoStateDB(utxos, 1000)

	broadcastTxs = p.ChooseUnshieldIDsFromCandidates(utxos, unshieldIDs, waitingUnshieldState)
	printBroadcastTxs(t, broadcastTxs)
}
