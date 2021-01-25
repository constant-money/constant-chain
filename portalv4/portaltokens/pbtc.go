package portaltokens

import (
	"errors"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
)

type PortalBTCTokenProcessor struct {
	*PortalToken
}

func (p PortalBTCTokenProcessor) GetExpectedMemoForPorting(portingID string) string {
	return p.PortalToken.GetExpectedMemoForPorting(portingID)
}

func (p PortalBTCTokenProcessor) GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string {
	return p.PortalToken.GetExpectedMemoForRedeem(redeemID, custodianIncAddress)
}

func (p PortalBTCTokenProcessor) ParseAndVerifyProof(
	proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string) (bool, []*statedb.UTXO, uint64, error) {
	btcChain := bc.GetBTCHeaderChain()
	if btcChain == nil {
		Logger.log.Error("BTC relaying chain should not be null")
		return false, nil, 0, errors.New("BTC relaying chain should not be null")
	}
	// parse BTCProof in meta
	btcTxProof, err := btcrelaying.ParseBTCProofFromB64EncodeStr(proof)
	if err != nil {
		Logger.log.Errorf("PortingProof is invalid %v\n", err)
		return false, nil, 0, fmt.Errorf("PortingProof is invalid %v\n", err)
	}

	// verify tx with merkle proofs
	isValid, err := btcChain.VerifyTxWithMerkleProofs(btcTxProof)
	if !isValid || err != nil {
		Logger.log.Errorf("Verify btcTxProof failed %v", err)
		return false, nil, 0, fmt.Errorf("Verify btcTxProof failed %v", err)
	}

	// extract attached message from txOut's OP_RETURN
	btcAttachedMsg, err := btcrelaying.ExtractAttachedMsgFromTx(btcTxProof.BTCTx)
	if err != nil {
		Logger.log.Errorf("Could not extract attached message from BTC tx proof with err: %v", err)
		return false, nil, 0, fmt.Errorf("Could not extract attached message from BTC tx proof with err: %v", err)
	}
	if btcAttachedMsg != expectedMemo {
		Logger.log.Errorf("PortingId in the btc attached message is not matched with portingID in metadata")
		return false, nil, 0, fmt.Errorf("PortingId in the btc attached message %v is not matched with portingID in metadata %v", btcAttachedMsg, expectedMemo)
	}

	// check whether amount transfer in txBNB is equal porting amount or not
	// check receiver and amount in tx
	outputs := btcTxProof.BTCTx.TxOut
	totalValue := uint64(0)

	listUTXO := []*statedb.UTXO{}

	for idx, out := range outputs {
		addrStr, err := btcChain.ExtractPaymentAddrStrFromPkScript(out.PkScript)
		if err != nil {
			Logger.log.Errorf("[portal] ExtractPaymentAddrStrFromPkScript: could not extract payment address string from pkscript with err: %v\n", err)
			continue
		}
		if addrStr != expectedMultisigAddress {
			continue
		}

		totalValue += uint64(out.Value)

		var curUTXO statedb.UTXO
		curUTXO.SetTxHash(btcTxProof.BTCTx.TxHash().String())
		curUTXO.SetOutputIndex(idx)
		curUTXO.SetOutputAmount(uint64(out.Value))

		listUTXO = append(listUTXO, &curUTXO)
	}

	return true, listUTXO, totalValue, nil
}

func (p PortalBTCTokenProcessor) IsValidRemoteAddress(address string, bcr bMeta.ChainRetriever) (bool, error) {
	btcHeaderChain := bcr.GetBTCHeaderChain()
	if btcHeaderChain == nil {
		return false, nil
	}
	return btcHeaderChain.IsBTCAddressValid(address), nil
}

func (p PortalBTCTokenProcessor) GetChainID() string {
	return p.ChainID
}

func (p PortalBTCTokenProcessor) GetMinTokenAmount() uint64 {
	return p.MinTokenAmount
}

// TODO:
func (p PortalBTCTokenProcessor) CreateRawExternalTx() error {
	return nil
}

type BtcUTXO struct {
	TxID    chainhash.Hash
	TxIndex uint32
	Amount  uint64
}

func sortBtcUTXOsAscendingAmount(utxos []*BtcUTXO) {
	sort.SliceStable(utxos, func(i, j int) bool {
		return utxos[i].Amount <= utxos[j].Amount
	})
}

func getBalance(utxos []*BtcUTXO) uint64 {
	balance := uint64(0)
	for _, item := range utxos {
		balance += item.Amount
	}
	return balance
}

// findClosestUTXO returns the closest utxo that have amount is greater or equal to target amount
// utxos was sorted ascending by amount
func findClosestUTXO(utxos []*BtcUTXO, targetAmount uint64) (*BtcUTXO, error) {
	l := 0
	r := len(utxos) - 1

	if utxos[l].Amount >= targetAmount {
		return utxos[l], nil
	}

	if utxos[r].Amount < targetAmount {
		return nil, errors.New("There is no utxo that has amount greater or equal target amount")
	}

	for true {
		m := (l + r) / 2
		if m <= l || m >= r {
			break
		}

		if utxos[m].Amount > targetAmount {
			r = m
		} else if utxos[m].Amount == targetAmount {
			return utxos[m], nil
		} else {
			l = m
		}
	}

	return utxos[r], nil
}

// chooseUTXOs receives all utxos of sender and returns the list utxos for spending with amountTransfer and fee
// we choose utxos such that number of chosen utxos is at least.
func chooseUTXOs(utxos []*BtcUTXO, amountTransfer uint64, fee uint64) ([]*BtcUTXO, error) {
	// check balance is valid or not
	totalTransfer := amountTransfer + fee
	if totalTransfer == 0 {
		return nil, errors.New("Amount transfer and fee are zero")
	}
	balance := getBalance(utxos)
	if balance < totalTransfer {
		return nil, fmt.Errorf("Balance %v is insufficiently to transfer %v and fee %v", balance, amountTransfer, fee)
	}

	// sort list of utxos
	sortBtcUTXOsAscendingAmount(utxos)
	chosenUTXOs := []*BtcUTXO{}

	// check the greatest utxo
	if len(utxos) == 0 {
		return nil, errors.New("UTXOs is empty")
	}
	if utxos[len(utxos)-1].Amount >= totalTransfer {
		// find the closest utxo
		closestUTXO, err := findClosestUTXO(utxos, totalTransfer)
		if err != nil {
			return nil, err
		}

		chosenUTXOs = append(chosenUTXOs, closestUTXO)
	} else {
		// greedy
		actualTransfer := uint64(0)
		for i := len(utxos) - 1; i >= 0; i-- {
			chosenUTXOs = append(chosenUTXOs, utxos[i])
			actualTransfer += utxos[i].Amount
			if actualTransfer >= totalTransfer {
				break
			}
		}
	}

	return chosenUTXOs, nil
}
