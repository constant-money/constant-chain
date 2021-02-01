package portaltokens

import (
	"errors"
	"fmt"

	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
)

type PortalBTCTokenProcessor struct {
	*PortalToken
}

func (p PortalBTCTokenProcessor) GetExpectedMemoForShielding(portingID string) string {
	return p.PortalToken.GetExpectedMemoForShielding(portingID)
}

func (p PortalBTCTokenProcessor) GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string {
	return p.PortalToken.GetExpectedMemoForRedeem(redeemID, custodianIncAddress)
}

func (p PortalBTCTokenProcessor) ParseAndVerifyProof(
	proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string) (bool, []*statedb.UTXO, error) {
	btcChain := bc.GetBTCHeaderChain()
	if btcChain == nil {
		Logger.log.Error("BTC relaying chain should not be null")
		return false, nil, errors.New("BTC relaying chain should not be null")
	}
	// parse BTCProof in meta
	btcTxProof, err := btcrelaying.ParseBTCProofFromB64EncodeStr(proof)
	if err != nil {
		Logger.log.Errorf("ShieldingProof is invalid %v\n", err)
		return false, nil, fmt.Errorf("ShieldingProof is invalid %v\n", err)
	}

	// verify tx with merkle proofs
	isValid, err := btcChain.VerifyTxWithMerkleProofs(btcTxProof)
	if !isValid || err != nil {
		Logger.log.Errorf("Verify btcTxProof failed %v", err)
		return false, nil, fmt.Errorf("Verify btcTxProof failed %v", err)
	}

	// extract attached message from txOut's OP_RETURN
	btcAttachedMsg, err := btcrelaying.ExtractAttachedMsgFromTx(btcTxProof.BTCTx)
	if err != nil {
		Logger.log.Errorf("Could not extract attached message from BTC tx proof with err: %v", err)
		return false, nil, fmt.Errorf("Could not extract attached message from BTC tx proof with err: %v", err)
	}
	if btcAttachedMsg != expectedMemo {
		Logger.log.Errorf("ShieldingId in the btc attached message is not matched with portingID in metadata")
		return false, nil, fmt.Errorf("ShieldingId in the btc attached message %v is not matched with portingID in metadata %v", btcAttachedMsg, expectedMemo)
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

		listUTXO = append(listUTXO, statedb.NewUTXOWithValue(
			addrStr,
			btcTxProof.BTCTx.TxHash().String(),
			idx,
			uint64(out.Value),
		))
	}

	if len(listUTXO) == 0 || totalValue < p.GetMinTokenAmount() {
		Logger.log.Errorf("Shielding amount: %v is less than the minimum threshold: %v\n", totalValue, p.GetMinTokenAmount())
		return false, nil, fmt.Errorf("Shielding amount: %v is less than the minimum threshold: %v", totalValue, p.GetMinTokenAmount())
	}

	return true, listUTXO, nil
}

func (p PortalBTCTokenProcessor) GetExternalTxHashFromProof(proof string) (string, error) {
	// parse BTCProof in meta
	btcTxProof, err := btcrelaying.ParseBTCProofFromB64EncodeStr(proof)
	if err != nil {
		Logger.log.Errorf("ShieldingProof is invalid %v\n", err)
		return "", fmt.Errorf("ShieldingProof is invalid %v\n", err)
	}

	return btcTxProof.BTCTx.TxHash().String(), nil
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

func (p PortalBTCTokenProcessor) ChooseUnshieldIDsFromCandidates(utxos []*statedb.UTXO, waitingUnshieldState map[string]*statedb.WaitingUnshieldRequest) []*BroadcastTx {
	return p.PortalToken.ChooseUnshieldIDsFromCandidates(utxos, waitingUnshieldState)
}
