package rpcserver

import (
	"bytes"
	"encoding/hex"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

type GetSignedTxResult struct {
	SignedTx     string
	BeaconHeight uint64
}

func getRawSignedTxByHeight(
	httpServer *HttpServer,
	height uint64,
	rawTx string,
) (interface{}, *rpcservice.RPCError) {
	// Get beacon block
	beaconBlockQueried, err := getSingleBeaconBlockByHeight(httpServer.GetBlockchain(), height)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	block := &beaconBlock{BeaconBlock: beaconBlockQueried}
	portalV4Sig, err := block.ProtalV4Sigs(httpServer.config.ConsensusEngine)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	hexRawTx, err := hex.DecodeString(rawTx)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	buffer := bytes.NewReader(hexRawTx)
	redeemTx := wire.NewMsgTx(wire.TxVersion)
	err = redeemTx.Deserialize(buffer)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	signatures := make([]*txscript.ScriptBuilder, len(redeemTx.TxIn))
	for i := range signatures {
		signature := txscript.NewScriptBuilder()
		signature.AddOp(txscript.OP_FALSE)
		signatures[i] = signature
	}

	redeemTxHash := redeemTx.TxHash().String()
	var tokenID string
	for _, v := range portalV4Sig {
		if v.RawTxHash == redeemTxHash {
			if tokenID == "" {
				tokenID = v.TokenID
			}
			for i, v2 := range v.Sigs {
				if i >= len(signatures) {
					return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
				}
				signatures[i].AddData(v2)
			}
		}
	}
	if tokenID == "" {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	redeemScriptStr := httpServer.portal.BlockChain.GetMultiSigScriptHexEncode(height, tokenID)
	redeemScriptHex, err := hex.DecodeString(redeemScriptStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	for i, v := range signatures {
		v.AddData(redeemScriptHex)
		signatureScript, err := v.Script()
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
		redeemTx.TxIn[i].SignatureScript = signatureScript
	}

	var signedTx bytes.Buffer
	err = redeemTx.Serialize(&signedTx)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	hexSignedTx := hex.EncodeToString(signedTx.Bytes())

	return GetSignedTxResult{
		SignedTx:     hexSignedTx,
		BeaconHeight: height,
	}, nil
}
