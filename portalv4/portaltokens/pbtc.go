package portaltokens

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
)

type PortalBTCTokenProcessor struct {
	*PortalToken
}

func (p PortalBTCTokenProcessor) GetExpectedMemoForShielding(incAddress string) string {
	return p.PortalToken.GetExpectedMemoForShielding(incAddress)
}

func (p PortalBTCTokenProcessor) GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string {
	return p.PortalToken.GetExpectedMemoForRedeem(redeemID, custodianIncAddress)
}

func (p PortalBTCTokenProcessor) ConvertExternalToIncAmount(externalAmt uint64) uint64 {
	return externalAmt * 10
}

func (p PortalBTCTokenProcessor) ConvertIncToExternalAmount(incAmt uint64) uint64 {
	return incAmt / 10 // incAmt / 1^9 * 1^8
}

func (p PortalBTCTokenProcessor) parseAndVerifyProofBTCChain(
	proof string, btcChain *btcrelaying.BlockChain, expectedMemo string, expectedMultisigAddress string) (bool, []*statedb.UTXO, error) {
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
		Logger.log.Errorf("ShieldingIncAddress in the btc attached message is not matched with ShieldingIncAddress in metadata")
		return false, nil, fmt.Errorf("ShieldingIncAddress in the btc attached message %v is not matched with ShieldingIncAddress in metadata %v", btcAttachedMsg, expectedMemo)
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
			uint32(idx),
			uint64(out.Value),
		))
	}

	if len(listUTXO) == 0 || totalValue < p.GetMinTokenAmount() {
		Logger.log.Errorf("Shielding amount: %v is less than the minimum threshold: %v\n", totalValue, p.GetMinTokenAmount())
		return false, nil, fmt.Errorf("Shielding amount: %v is less than the minimum threshold: %v", totalValue, p.GetMinTokenAmount())
	}

	return true, listUTXO, nil
}

func (p PortalBTCTokenProcessor) ParseAndVerifyProof(
	proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string) (bool, []*statedb.UTXO, error) {
	btcChain := bc.GetBTCHeaderChain()
	return p.parseAndVerifyProofBTCChain(proof, btcChain, expectedMemo, expectedMultisigAddress)
}

func (p PortalBTCTokenProcessor) ParseAndVerifyUnshieldProof(
	proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string, expectPaymentInfo map[string]uint64, utxos []*statedb.UTXO) (bool, []*statedb.UTXO, error) {
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
		Logger.log.Errorf("Submit confirmed tx in the btc attached message is not matched with portingID in metadata")
		return false, nil, fmt.Errorf("Submit confirmed tx in the btc attached message %v is not matched with batchid in metadata %v", btcAttachedMsg, expectedMemo)
	}

	// verify spent outputs
	if len(btcTxProof.BTCTx.TxIn) < 1 {
		Logger.log.Errorf("Can not find the tx inputs in proof")
		return false, nil, fmt.Errorf("Submit confirmed tx: no tx inputs in proof")
	}
	input := btcTxProof.BTCTx.TxIn[0]
	isMached := false
	for _, v := range utxos {
		if v.GetTxHash() == input.PreviousOutPoint.Hash.String() && v.GetOutputIndex() == input.PreviousOutPoint.Index {
			isMached = true
			break
		}
	}
	if !isMached {
		Logger.log.Errorf("Submit confirmed: tx inputs from proof is diff utxos from unshield batch")
		return false, nil, fmt.Errorf("Submit confirmed tx: tx inputs from proof is diff utxos from unshield batch")
	}

	// check whether amount transfer in txBNB is equal porting amount or not
	// check receiver and amount in tx
	outputs := btcTxProof.BTCTx.TxOut

	for receiverAddress := range expectPaymentInfo {
		isMached = false
		for _, out := range outputs {
			addrStr, err := btcChain.ExtractPaymentAddrStrFromPkScript(out.PkScript)
			if err != nil {
				Logger.log.Errorf("[portal] ExtractPaymentAddrStrFromPkScript: could not extract payment address string from pkscript with err: %v\n", err)
				continue
			}
			if addrStr != receiverAddress {
				continue
			}
			isMached = true
			break
		}
		if !isMached {
			Logger.log.Error("BTC-TxProof is invalid")
			return false, nil, errors.New("BTC-TxProof is invalid")
		}
	}

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
			uint32(idx),
			uint64(out.Value),
		))
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

// generate multisig wallet address from seeds (seed is mining key of beacon validator in byte array)
func (p PortalBTCTokenProcessor) GenerateMultiSigWalletFromSeeds(bc bMeta.ChainRetriever, seeds [][]byte, numSigsRequired int) ([]byte, []string, string, error) {
	if len(seeds) < numSigsRequired || numSigsRequired < 0 {
		return nil, nil, "", errors.New("Invalid signature requirment")
	}
	bitcoinPrvKeys := make([]*btcec.PrivateKey, 0)
	bitcoinPrvKeyStrs := make([]string, 0) // btc private key hex encoded
	// create redeem script for 2 of 3 multi-sig
	builder := txscript.NewScriptBuilder()
	// add the minimum number of needed signatures
	builder.AddOp(byte(txscript.OP_1 - 1 + numSigsRequired))
	for _, seed := range seeds {
		BTCKeyBytes := pv4Common.GenBTCPrivateKey(seed)
		privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), BTCKeyBytes)
		// add the public key to redeem script
		builder.AddData(privKey.PubKey().SerializeCompressed())
		bitcoinPrvKeys = append(bitcoinPrvKeys, privKey)
		bitcoinPrvKeyStrs = append(bitcoinPrvKeyStrs, hex.EncodeToString(privKey.Serialize()))
	}
	// add the total number of public keys in the multi-sig script
	builder.AddOp(byte(txscript.OP_1 - 1 + len(seeds)))
	// add the check-multi-sig op-code
	builder.AddOp(txscript.OP_CHECKMULTISIG)
	// redeem script is the script program in the format of []byte
	redeemScript, err := builder.Script()
	if err != nil {
		return nil, nil, "", err
	}
	// generate multisig address
	multiAddress := btcutil.Hash160(redeemScript)
	addr, err := btcutil.NewAddressScriptHashFromHash(multiAddress, bc.GetBTCHeaderChain().GetChainParams())

	return redeemScript, bitcoinPrvKeyStrs, addr.String(), nil
}

// GeneratePartPrivateKeyFromSeed generate private key from seed
// return the private key serialized in bytes array
func (p PortalBTCTokenProcessor) GeneratePrivateKeyFromSeed(seed []byte) ([]byte, error) {
	if len(seed) == 0 {
		return nil, errors.New("Invalid seed")
	}

	btcKeyBytes := pv4Common.GenBTCPrivateKey(seed)
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), btcKeyBytes)

	return privKey.Serialize(), nil
}

// CreateRawExternalTx creates raw btc transaction (not include signatures of beacon validator)
// TODO: networkFee
func (p PortalBTCTokenProcessor) CreateRawExternalTx(inputs []*statedb.UTXO, outputs []*OutputTx, networkFee uint64, memo string, bc bMeta.ChainRetriever) (string, string, error) {
	msgTx := wire.NewMsgTx(wire.TxVersion)

	// add TxIns into raw tx
	for _, in := range inputs {
		utxoHash, err := chainhash.NewHashFromStr(in.GetTxHash())
		if err != nil {
			Logger.log.Errorf("[CreateRawExternalTx-BTC] Error when new TxIn for tx: %v", err)
			return "", "", nil
		}
		outPoint := wire.NewOutPoint(utxoHash, in.GetOutputIndex())
		txIn := wire.NewTxIn(outPoint, nil, nil)
		msgTx.AddTxIn(txIn)
	}

	// add TxOuts into raw tx
	for _, out := range outputs {
		// adding the output to tx
		decodedAddr, err := btcutil.DecodeAddress(out.ReceiverAddress, bc.GetBTCHeaderChain().GetChainParams())
		if err != nil {
			Logger.log.Errorf("[CreateRawExternalTx-BTC] Error when decoding receiver address: %v", err)
			return "", "", err
		}
		destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
		if err != nil {
			Logger.log.Errorf("[CreateRawExternalTx-BTC] Error when new Address Script: %v", err)
			return "", "", err
		}

		// adding the destination address and the amount to the transaction
		redeemTxOut := wire.NewTxOut(int64(out.Amount), destinationAddrByte)
		msgTx.AddTxOut(redeemTxOut)
	}

	// add memo into raw tx
	script := append([]byte{txscript.OP_RETURN}, byte(len([]byte(memo))))
	script = append(script, []byte(memo)...)
	msgTx.AddTxOut(wire.NewTxOut(0, script))

	var rawTxBytes bytes.Buffer
	err := msgTx.Serialize(&rawTxBytes)
	if err != nil {
		Logger.log.Errorf("[CreateRawExternalTx-BTC] Error when serializing raw tx: %v", err)
		return "", "", err
	}

	hexRawTx := hex.EncodeToString(rawTxBytes.Bytes())
	return hexRawTx, msgTx.TxHash().String(), nil
}

//func (p PortalBTCTokenProcessor) ExtractRawTx(rawTxStr string) ([]*statedb.UTXO, uint, error){
//	rawTxBytes, err := hex.DecodeString(rawTxStr)
//	if err != nil {
//		Logger.log.Errorf("[ExtractRawTx-BTC] Error when decoding raw tx string: %v", err)
//		return nil, 0, err
//	}
//
//	msgTx := new(wire.MsgTx)
//	rawTxBuffer := bytes.NewBuffer(rawTxBytes)
//	err = msgTx.Deserialize(rawTxBuffer)
//	if err != nil {
//		Logger.log.Errorf("[ExtractRawTx-BTC] Error when deserializing raw tx bytes: %v", err)
//		return nil, 0, err
//	}
//
//	utxos := []*statedb.UTXO{}
//	for _, in := range msgTx.TxIn {
//		utxos = append(utxos, statedb.NewUTXOWithValue())
//	}
//}

//func (p PortalBTCTokenProcessor) ChooseUnshieldIDsFromCandidates(utxos map[string]*statedb.UTXO, waitingUnshieldReqs map[string]*statedb.WaitingUnshieldRequest) []*BroadcastTx {
//	return p.PortalToken.ChooseUnshieldIDsFromCandidates(utxos, waitingUnshieldReqs)
//}

func (p PortalBTCTokenProcessor) IsAcceptableTxSize(num_utxos int, num_unshield_id int) bool {
	// TODO: do experiments depend on external chain miner's habit
	A := 1
	B := 1
	C := 10
	return A*num_utxos+B*num_unshield_id <= C
}

// Choose list of pairs (UTXOs and unshield IDs) for broadcast external transactions
func (p PortalBTCTokenProcessor) ChooseUnshieldIDsFromCandidates(utxos map[string]*statedb.UTXO, waitingUnshieldReqs map[string]*statedb.WaitingUnshieldRequest) []*BroadcastTx {
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

	// convert unshield amount to external token amount
	wReqsArr := []unshieldItem{}
	for k, req := range waitingUnshieldReqs {
		wReqsArr = append(
			wReqsArr,
			unshieldItem{
				key: k,
				value: statedb.NewWaitingUnshieldRequestStateWithValue(
					req.GetRemoteAddress(), p.ConvertIncToExternalAmount(req.GetAmount()), req.GetUnshieldID(), req.GetBeaconHeight()),
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
