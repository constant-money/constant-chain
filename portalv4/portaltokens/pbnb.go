package portaltokens

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/binance-chain/go-sdk/types/msg"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/relaying/bnb"
)

type PortalBNBTokenProcessor struct {
	*PortalToken
}

func (p PortalBNBTokenProcessor) IsValidRemoteAddress(address string, bcr bMeta.ChainRetriever) (bool, error) {
	return bnb.IsValidBNBAddress(address, p.ChainID), nil
}

func (p PortalBNBTokenProcessor) GetChainID() string {
	return p.ChainID
}

func (p PortalBNBTokenProcessor) GetMinTokenAmount() uint64 {
	return p.MinTokenAmount
}

func (p PortalBNBTokenProcessor) ParseAndVerifyProof(
	proof string,
	bc bMeta.ChainRetriever,
	expectedMemo string,
	expectedPaymentInfos map[string]uint64) (bool, error) {
	// parse BNBproof from string
	txProofBNB, err := bnb.ParseBNBProofFromB64EncodeStr(proof)
	if err != nil {
		Logger.log.Errorf("Error when decoding BNBProof %v\n", err)
		return false, fmt.Errorf("Error when decoding BNBProof %v\n", err)
	}

	// check minimum confirmations block of bnb proof
	latestBNBBlockHeight, err2 := bc.GetLatestBNBBlkHeight()
	if err2 != nil {
		Logger.log.Errorf("Can not get latest relaying bnb block height %v\n", err)
		return false, fmt.Errorf("Can not get latest relaying bnb block height %v\n", err)
	}
	if latestBNBBlockHeight < txProofBNB.BlockHeight+bnb.MinConfirmationsBlock {
		Logger.log.Errorf("Not enough min bnb confirmations block %v, latestBNBBlockHeight %v - txProofBNB.BlockHeight %v\n",
			bnb.MinConfirmationsBlock, latestBNBBlockHeight, txProofBNB.BlockHeight)
		return false, fmt.Errorf("Not enough min bnb confirmations block %v, latestBNBBlockHeight %v - txProofBNB.BlockHeight %v\n",
			bnb.MinConfirmationsBlock, latestBNBBlockHeight, txProofBNB.BlockHeight)
	}

	// get data hash of bnb block
	dataHash, err2 := bc.GetBNBDataHash(txProofBNB.BlockHeight)
	if err2 != nil {
		Logger.log.Errorf("Error when get data hash in blockHeight %v - %v\n",
			txProofBNB.BlockHeight, err2)
		return false, fmt.Errorf("Error when get data hash in blockHeight %v - %v\n",
			txProofBNB.BlockHeight, err2)
	}

	// verify txProofBNB with dataHash
	isValid, err := txProofBNB.Verify(dataHash)
	if !isValid || err != nil {
		Logger.log.Errorf("Verify txProofBNB failed %v", err)
		return false, fmt.Errorf("Verify txProofBNB failed %v", err)
	}

	// parse Tx from Data in txProofBNB
	txBNB, err := bnb.ParseTxFromData(txProofBNB.Proof.Data)
	if err != nil {
		Logger.log.Errorf("Data in BNBProof is invalid %v", err)
		return false, fmt.Errorf("Data in BNBProof is invalid %v", err)
	}

	// check memo attached in bnb transaction
	memo := txBNB.Memo
	if memo != expectedMemo {
		Logger.log.Errorf("Expected memo %v - but got %v", expectedMemo, memo)
		return false, fmt.Errorf("Expected memo %v - but got %v", expectedMemo, memo)
	}

	// check paymentInfos (receiver - amount)
	// check whether amount transfer in txBNB is equal redeem/porting amount or not
	// check receiver and amount in tx

	outputs := txBNB.Msgs[0].(msg.SendMsg).Outputs

	for receiverAddress, amount := range expectedPaymentInfos {
		amountNeedToBeTransferInBNB := ConvertIncPBNBAmountToExternalBNBAmount(int64(amount))

		isChecked := false
		for _, out := range outputs {
			addr, _ := bnb.GetAccAddressString(&out.Address, bc.GetBNBChainID(0))
			if addr != receiverAddress {
				continue
			}

			// calculate amount that was transferred to custodian's remote address
			amountTransfer := int64(0)
			for _, coin := range out.Coins {
				if coin.Denom == bnb.DenomBNB {
					amountTransfer += coin.Amount
					break
				}
			}
			if amountTransfer < amountNeedToBeTransferInBNB {
				Logger.log.Errorf("TxProof-BNB is invalid - Amount transfer to %s must be equal to or greater than %d, but got %d",
					addr, amountNeedToBeTransferInBNB, amountTransfer)
				return false, fmt.Errorf("TxProof-BNB is invalid - Amount transfer to %s must be equal to or greater than %d, but got %d",
					addr, amountNeedToBeTransferInBNB, amountTransfer)
			} else {
				isChecked = true
				break
			}
		}

		if !isChecked {
			Logger.log.Errorf("TxProof-BNB is invalid - Receiver address is invalid, expected %v",
				receiverAddress)
			return false, fmt.Errorf("TxProof-BNB is invalid - Receiver address is invalid, expected %v",
				receiverAddress)
		}
	}

	return true, nil
}

func (p PortalBNBTokenProcessor) GetExpectedMemoForShielding(portingID string) string {
	return p.PortalToken.GetExpectedMemoForShielding(portingID)
}

func (p PortalBNBTokenProcessor) GetExpectedMemoForRedeem(redeemID string, custodianAddress string) string {
	return p.PortalToken.GetExpectedMemoForRedeem(redeemID, custodianAddress)
}

// ConvertIncPBNBAmountToExternalBNBAmount converts amount in inc chain (decimal 9) to amount in bnb chain (decimal 8)
func ConvertIncPBNBAmountToExternalBNBAmount(incPBNBAmount int64) int64 {
	return incPBNBAmount / 10 // incPBNBAmount / 1^9 * 1^8
}

// TODO:
func (p PortalBNBTokenProcessor) CreateRawExternalTx(inputs []*statedb.UTXO, outputs []*OutputTx, memo string) (string, string, error){
	return "", "", nil
}

// TODO:
func (p PortalBNBTokenProcessor) ChooseUnshieldIDsFromCandidates(utxos map[string]*statedb.UTXO, waitingUnshieldReqs map[string]*statedb.WaitingUnshieldRequest) []*BroadcastTx {
	return nil
}