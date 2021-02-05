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

	GetExpectedMemoForShielding(incAddress string) string
	GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string
	ParseAndVerifyProof(
		proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string) (bool, []*statedb.UTXO, error)
	GetExternalTxHashFromProof(proof string) (string, error)
	ChooseUnshieldIDsFromCandidates(utxos map[string]*statedb.UTXO, waitingUnshieldReqs map[string]*statedb.WaitingUnshieldRequest) []*BroadcastTx

	CreateRawExternalTx(inputs []*statedb.UTXO, outputs []*OutputTx, networkFee uint64, memo string, bc bMeta.ChainRetriever) (string, string, error)
	//ExtractRawTx(rawTxStr string) ([]*statedb.UTXO, uint)
}

// set MinTokenAmount to avoid attacking with amount is less than smallest unit of cryptocurrency
// such as satoshi in BTC
type PortalToken struct {
	ChainID        string
	MinTokenAmount uint64 // minimum amount for shielding/redeem
}

type BroadcastTx struct {
	UTXOs       []*statedb.UTXO
	UnshieldIDs []string
}

type OutputTx struct {
	ReceiverAddress string
	Amount          uint64
}

func (p PortalToken) GetExpectedMemoForShielding(incAddress string) string {
	type shieldingMemoStruct struct {
		IncAddress string `json:"ShieldingIncAddress"`
	}
	memoShielding := shieldingMemoStruct{IncAddress: incAddress}
	memoShieldingBytes, _ := json.Marshal(memoShielding)
	memoShieldingHashBytes := common.HashB(memoShieldingBytes)
	memoShieldingStr := base64.StdEncoding.EncodeToString(memoShieldingHashBytes)
	return memoShieldingStr
}

//todo:
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

func (p PortalToken) IsAcceptableTxSize(num_utxos int, num_unshield_id int) bool {
	// TODO: do experiments depend on external chain miner's habit
	A := 1
	B := 1
	C := 10
	return A*num_utxos+B*num_unshield_id <= C
}

// Choose list of pairs (UTXOs and unshield IDs) for broadcast external transactions
func (p PortalToken) ChooseUnshieldIDsFromCandidates(utxos map[string]*statedb.UTXO, waitingUnshieldReqs map[string]*statedb.WaitingUnshieldRequest) []*BroadcastTx {
	if len(utxos) == 0 || len(waitingUnshieldReqs) == 0 {
		return []*BroadcastTx{}
	}

	// descending sort utxo by value
	type utxoItem struct {
		key   string
		value *statedb.UTXO
	}
	utxosArr := []utxoItem{}
	for k, req := range utxos {
		utxosArr = append(
			utxosArr,
			utxoItem{
				key:   k,
				value: req,
			})
	}
	sort.SliceStable(utxosArr, func(i, j int) bool {
		return utxosArr[i].value.GetOutputAmount() > utxosArr[j].value.GetOutputAmount()
	})

	// ascending sort waitingUnshieldReqs by beaconHeight
	type unshieldItem struct {
		key   string
		value *statedb.WaitingUnshieldRequest
	}

	wReqsArr := []unshieldItem{}
	for k, req := range waitingUnshieldReqs {
		wReqsArr = append(
			wReqsArr,
			unshieldItem{
				key:   k,
				value: req,
			})
	}

	sort.SliceStable(wReqsArr, func(i, j int) bool {
		return wReqsArr[i].value.GetBeaconHeight() < wReqsArr[j].value.GetBeaconHeight()
	})

	broadcastTxs := []*BroadcastTx{}
	utxo_idx := 0
	unshield_idx := 0
	for utxo_idx < len(utxos) && unshield_idx < len(wReqsArr) {
		chosenUTXOs := []*statedb.UTXO{}
		chosenUnshieldIDs := []string{}

		cur_sum_amount := uint64(0)
		cnt := 0
		if utxosArr[utxo_idx].value.GetOutputAmount() >= wReqsArr[unshield_idx].value.GetAmount() {
			// find the last unshield idx that the cummulative sum of unshield amount <= current utxo amount
			for unshield_idx < len(wReqsArr) && cur_sum_amount+wReqsArr[unshield_idx].value.GetAmount() <= utxosArr[utxo_idx].value.GetOutputAmount() && p.IsAcceptableTxSize(1, cnt+1) {
				cur_sum_amount += wReqsArr[unshield_idx].value.GetAmount()
				chosenUnshieldIDs = append(chosenUnshieldIDs, wReqsArr[unshield_idx].value.GetUnshieldID())
				unshield_idx += 1
				cnt += 1
			}
			chosenUTXOs = append(chosenUTXOs, utxosArr[utxo_idx].value)
			utxo_idx += 1
		} else {
			// find the first utxo idx that the cummulative sum of utxo amount >= current unshield amount
			for utxo_idx < len(utxos) && cur_sum_amount+utxosArr[utxo_idx].value.GetOutputAmount() < wReqsArr[unshield_idx].value.GetAmount() {
				cur_sum_amount += utxosArr[utxo_idx].value.GetOutputAmount()
				chosenUTXOs = append(chosenUTXOs, utxosArr[utxo_idx].value)
				utxo_idx += 1
				cnt += 1
			}
			if utxo_idx < len(utxos) && p.IsAcceptableTxSize(cnt+1, 1) {
				// insert new unshield ids if the current utxos still has enough amount
				cur_sum_amount += utxosArr[utxo_idx].value.GetOutputAmount()
				chosenUTXOs = append(chosenUTXOs, utxosArr[utxo_idx].value)
				utxo_idx += 1
				cnt += 1

				new_cnt := 0
				target := cur_sum_amount
				cur_sum_amount = 0

				for unshield_idx < len(wReqsArr) && cur_sum_amount+wReqsArr[unshield_idx].value.GetAmount() <= target && p.IsAcceptableTxSize(cnt, new_cnt+1) {
					cur_sum_amount += wReqsArr[unshield_idx].value.GetAmount()
					chosenUnshieldIDs = append(chosenUnshieldIDs, wReqsArr[unshield_idx].value.GetUnshieldID())
					unshield_idx += 1
					new_cnt += 1
				}

			} else {
				// not enough utxo for last unshield IDs
				break
			}
		}
		broadcastTxs = append(broadcastTxs, &BroadcastTx{
			UTXOs:       chosenUTXOs,
			UnshieldIDs: chosenUnshieldIDs,
		})
	}
	return broadcastTxs
}
