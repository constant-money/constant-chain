package portalprocess

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/incognitochain/incognito-chain/portalv4"
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"
	pv4Meta "github.com/incognitochain/incognito-chain/portalv4/metadata"
)

// PortalSig defines sigs of one beacon validator on unshield external tx
type PortalSig struct {
	TokenID   string
	RawTxHash string
	Sigs      [][]byte // array of sigs for all TxIn
}

//todo: update for other case need beacon sign on
func CheckAndSignPortalUnshieldExternalTx(seedKey []byte, insts [][]string, portalParam portalv4.PortalParams) ([]*PortalSig, error) {
	pSigs := []*PortalSig{}
	for _, inst := range insts {
		// only this meta type need beacon sign on raw tx
		if pv4Meta.IsRequireBeaconSigForPortalV4Meta(inst) || len(inst) < 4 {
			continue
		}

		// unmarshal instructions content
		var actionData pv4Meta.PortalUnshieldRequestBatchContent
		err := json.Unmarshal([]byte(inst[3]), &actionData)
		if err != nil {
			return nil, fmt.Errorf("[checkAndSignPortalV4] Can not unmarshal instruction content %v - Error %v\n", inst[3], err)
		}

		rawTxBytes, err := hex.DecodeString(actionData.RawExternalTx)
		if err != nil {
			return nil, fmt.Errorf("[checkAndSignPortalV4] Error when decoding raw tx string: %v", err)
		}

		//todo: using interface to handle creating raw tx
		if actionData.TokenID == pv4Common.PortalBTCIDStr {
			msgTx := new(btcwire.MsgTx)
			rawTxBuffer := bytes.NewBuffer(rawTxBytes)
			err = msgTx.Deserialize(rawTxBuffer)
			if err != nil {
				return nil, fmt.Errorf("[checkAndSignPortalV4] Error when deserializing raw tx bytes: %v", err)
			}

			sigs := [][]byte{}
			for i := range msgTx.TxIn {
				// signing the tx
				signature := txscript.NewScriptBuilder()
				signature.AddOp(txscript.OP_FALSE)

				// generate btc private key from seed: private key of bridge consensus
				btcPrivateKeySeed := seedKey
				btcPrivateKeyBytes, err := portalParam.PortalTokens[pv4Common.PortalBTCIDStr].GeneratePrivateKeyFromSeed(btcPrivateKeySeed)
				if err != nil {
					return nil, fmt.Errorf("[checkAndSignPortalV4] Error when generate btc private key from seed: %v", err)
				}
				btcPrivateKey := new(btcec.PrivateKey)
				btcPrivateKey.D.SetBytes(btcPrivateKeyBytes)
				multiSigScript, _ := hex.DecodeString(portalParam.MultiSigScriptHexEncode[pv4Common.PortalBTCIDStr])
				sig, err := txscript.RawTxInSignature(msgTx, i, multiSigScript, txscript.SigHashAll, btcPrivateKey)
				if err != nil {
					return nil, fmt.Errorf("[checkAndSignPortalV4] Error when signing on raw btc tx: %v", err)
				}
				sigs = append(sigs, sig)
			}
			pSigs = append(pSigs, &PortalSig{
				TokenID:   actionData.TokenID,
				RawTxHash: msgTx.TxHash().String(),
				Sigs:      sigs,
			})
		}

	}

	return pSigs, nil
}
