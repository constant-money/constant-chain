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

func (p PortalBTCTokenProcessor) GetExpectedMemoForPorting(portingID string) string {
	return p.PortalToken.GetExpectedMemoForPorting(portingID)
}

func (p PortalBTCTokenProcessor) GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string {
	return p.PortalToken.GetExpectedMemoForRedeem(redeemID, custodianIncAddress)
}

func (p PortalBTCTokenProcessor) ParseAndVerifyProof(
	proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedPaymentInfos map[string]uint64) (bool, error) {
	btcChain := bc.GetBTCHeaderChain()
	if btcChain == nil {
		Logger.log.Error("BTC relaying chain should not be null")
		return false, errors.New("BTC relaying chain should not be null")
	}
	// parse BTCProof in meta
	btcTxProof, err := btcrelaying.ParseBTCProofFromB64EncodeStr(proof)
	if err != nil {
		Logger.log.Errorf("PortingProof is invalid %v\n", err)
		return false, fmt.Errorf("PortingProof is invalid %v\n", err)
	}

	// verify tx with merkle proofs
	isValid, err := btcChain.VerifyTxWithMerkleProofs(btcTxProof)
	if !isValid || err != nil {
		Logger.log.Errorf("Verify btcTxProof failed %v", err)
		return false, fmt.Errorf("Verify btcTxProof failed %v", err)
	}

	// extract attached message from txOut's OP_RETURN
	btcAttachedMsg, err := btcrelaying.ExtractAttachedMsgFromTx(btcTxProof.BTCTx)
	if err != nil {
		Logger.log.Errorf("Could not extract attached message from BTC tx proof with err: %v", err)
		return false, fmt.Errorf("Could not extract attached message from BTC tx proof with err: %v", err)
	}
	if btcAttachedMsg != expectedMemo {
		Logger.log.Errorf("PortingId in the btc attached message is not matched with portingID in metadata")
		return false, fmt.Errorf("PortingId in the btc attached message %v is not matched with portingID in metadata %v", btcAttachedMsg, expectedMemo)
	}

	// check whether amount transfer in txBNB is equal porting amount or not
	// check receiver and amount in tx
	outputs := btcTxProof.BTCTx.TxOut
	for receiverAddress, amount := range expectedPaymentInfos {
		amountNeedToBeTransferInBTC := btcrelaying.ConvertIncPBTCAmountToExternalBTCAmount(int64(amount))
		isChecked := false
		for _, out := range outputs {
			addrStr, err := btcChain.ExtractPaymentAddrStrFromPkScript(out.PkScript)
			if err != nil {
				Logger.log.Errorf("[portal] ExtractPaymentAddrStrFromPkScript: could not extract payment address string from pkscript with err: %v\n", err)
				continue
			}
			if addrStr != receiverAddress {
				continue
			}
			if out.Value < amountNeedToBeTransferInBTC {
				Logger.log.Errorf("BTC-TxProof is invalid - the transferred amount to %s must be equal to or greater than %d, but got %d", addrStr, amountNeedToBeTransferInBTC, out.Value)
				return false, fmt.Errorf("BTC-TxProof is invalid - the transferred amount to %s must be equal to or greater than %d, but got %d", addrStr, amountNeedToBeTransferInBTC, out.Value)
			} else {
				isChecked = true
				break
			}
		}
		if !isChecked {
			Logger.log.Error("BTC-TxProof is invalid")
			return false, errors.New("BTC-TxProof is invalid")
		}
	}

	return true, nil
}

func (p PortalBTCTokenProcessor) ParseAndVerifyProofV4(
	proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string, expectedAmount uint64) (bool, *statedb.UTXO, error) {
	btcChain := bc.GetBTCHeaderChain()
	if btcChain == nil {
		Logger.log.Error("BTC relaying chain should not be null")
		return false, nil, errors.New("BTC relaying chain should not be null")
	}
	// parse BTCProof in meta
	btcTxProof, err := btcrelaying.ParseBTCProofFromB64EncodeStr(proof)
	if err != nil {
		Logger.log.Errorf("PortingProof is invalid %v\n", err)
		return false, nil, fmt.Errorf("PortingProof is invalid %v\n", err)
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
		Logger.log.Errorf("PortingId in the btc attached message is not matched with portingID in metadata")
		return false, nil, fmt.Errorf("PortingId in the btc attached message %v is not matched with portingID in metadata %v", btcAttachedMsg, expectedMemo)
	}

	// check whether amount transfer in txBNB is equal porting amount or not
	// check receiver and amount in tx
	outputs := btcTxProof.BTCTx.TxOut
	amountNeedToBeTransferInBTC := btcrelaying.ConvertIncPBTCAmountToExternalBTCAmount(int64(expectedAmount))
	totalValue := int64(0)

	var curUTXO statedb.UTXO
	curUTXO.txHash = btcTxProof.BTCTx
	curUTXO.outputIdx = []int{}
	curUTXO.outputAmount = []uint64{}

	for idx, out := range outputs {
		addrStr, err := btcChain.ExtractPaymentAddrStrFromPkScript(out.PkScript)
		if err != nil {
			Logger.log.Errorf("[portal] ExtractPaymentAddrStrFromPkScript: could not extract payment address string from pkscript with err: %v\n", err)
			continue
		}
		if addrStr != expectedMultisigAddress {
			continue
		}

		totalValue += out.Value
		curUTXO.outputIdx = append(curUTXO.outputIdx, idx)
		curUTXO.outputAmount = append(curUTXO.outputAmount, out.Value)
	}

	if totalValue < amountNeedToBeTransferInBTC {
		Logger.log.Errorf("BTC-TxProof is invalid - the transferred amount to %s must be equal to or greater than %d, but got %d", expectedMultisigAddress, amountNeedToBeTransferInBTC, totalValue)
		return false, nil, fmt.Errorf("BTC-TxProof is invalid - the transferred amount to %s must be equal to or greater than %d, but got %d", expectedMultisigAddress, amountNeedToBeTransferInBTC, totalValue)
	}

	return true, &curUTXO, nil
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
