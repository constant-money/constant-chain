package portaltokens

import (
	"encoding/base64"
	"encoding/json"
	"sort"

	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type PortalTokenProcessor interface {
	IsValidRemoteAddress(address string, bcr bMeta.ChainRetriever) (bool, error)
	GetChainID() string
	GetMinTokenAmount() uint64

	GetExpectedMemoForPorting(incAddress string) string
	GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string
	ParseAndVerifyProof(
		proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string) (bool, []*statedb.UTXO, uint64, error)
	ChooseUnshieldIDsFromCandidates(utxos []*statedb.UTXO, unshieldIDs []string, waitingUnshieldState *statedb.WaitingUnshield) []*BroadcastTx

	CreateRawExternalTx() error
}

// set MinTokenAmount to avoid attacking with amount is less than smallest unit of cryptocurrency
// such as satoshi in BTC
type PortalToken struct {
	ChainID        string
	MinTokenAmount uint64 // minimum amount for porting/redeem
}

type BroadcastTx struct {
	UTXOs       []*statedb.UTXO
	UnshieldIDs []string
}

func (p PortalToken) GetExpectedMemoForPorting(incAddress string) string {
	type portingMemoStruct struct {
		IncAddress string `json:"PortingIncAddress"`
	}
	memoPorting := portingMemoStruct{IncAddress: incAddress}
	memoPortingBytes, _ := json.Marshal(memoPorting)
	memoPortingHashBytes := common.HashB(memoPortingBytes)
	memoPortingStr := base64.StdEncoding.EncodeToString(memoPortingHashBytes)
	return memoPortingStr
}

func (p PortalToken) GetExpectedMemoForRedeem(redeemID string, custodianAddress string) string {
	type redeemMemoStruct struct {
		RedeemID                  string `json:"RedeemID"`
		CustodianIncognitoAddress string `json:"CustodianIncognitoAddress"`
	}

	redeemMemo := redeemMemoStruct{
		RedeemID:                  redeemID,
		CustodianIncognitoAddress: custodianAddress,
	}
	redeemMemoBytes, _ := json.Marshal(redeemMemo)
	redeemMemoHashBytes := common.HashB(redeemMemoBytes)
	redeemMemoStr := base64.StdEncoding.EncodeToString(redeemMemoHashBytes)
	return redeemMemoStr
}

// Choose list of pairs (UTXOs and unshield IDs) for broadcast external transactions
func (p PortalToken) ChooseUnshieldIDsFromCandidates(utxos []*statedb.UTXO, unshieldIDs []string, waitingUnshieldState *statedb.WaitingUnshield) []*BroadcastTx {
	if len(utxos) == 0 || len(unshieldIDs) == 0 {
		return []*BroadcastTx{}
	}

	sort.SliceStable(utxos, func(i, j int) bool {
		return utxos[i].GetOutputAmount() < utxos[j].GetOutputAmount()
	})

	broadcastTxs := []*BroadcastTx{}
	utxo_idx := 0
	unshield_idx := 0
	for utxo_idx < len(utxos) && unshield_idx < len(unshieldIDs) {
		chosenUTXOs := []*statedb.UTXO{}
		chosenUnshieldIDs := []string{}

		cur_sum_amount := uint64(0)
		if utxos[utxo_idx].GetOutputAmount() >= waitingUnshieldState.GetUnshield(unshieldIDs[unshield_idx]).GetAmount() {
			// find the last unshield idx that the cummulative sum of unshield amount <= current utxo amount
			for unshield_idx < len(unshieldIDs) && cur_sum_amount+waitingUnshieldState.GetUnshield(unshieldIDs[unshield_idx]).GetAmount() <= utxos[utxo_idx].GetOutputAmount() {
				cur_sum_amount += waitingUnshieldState.GetUnshield(unshieldIDs[unshield_idx]).GetAmount()
				chosenUnshieldIDs = append(chosenUnshieldIDs, unshieldIDs[unshield_idx])
				unshield_idx += 1
			}
			chosenUTXOs = append(chosenUTXOs, utxos[utxo_idx])
			utxo_idx += 1
		} else {
			// find the first utxo idx that the cummulative sum of utxo amount >= current unshield amount
			for utxo_idx < len(utxos) && cur_sum_amount+utxos[utxo_idx].GetOutputAmount() < waitingUnshieldState.GetUnshield(unshieldIDs[unshield_idx]).GetAmount() {
				cur_sum_amount += utxos[utxo_idx].GetOutputAmount()
				chosenUTXOs = append(chosenUTXOs, utxos[utxo_idx])
				utxo_idx += 1
			}
			if utxo_idx < len(utxos) {
				chosenUTXOs = append(chosenUTXOs, utxos[utxo_idx])
				utxo_idx += 1
			} else {
				// not enough utxo for last unshield IDs
				break
			}
			chosenUnshieldIDs = append(chosenUnshieldIDs, unshieldIDs[unshield_idx])
			unshield_idx += 1
		}
		broadcastTxs = append(broadcastTxs, &BroadcastTx{
			UTXOs:       chosenUTXOs,
			UnshieldIDs: chosenUnshieldIDs,
		})
	}
	return broadcastTxs
}
