//+build linux,386 wasm

package main

import (
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/gobridge"
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/internal"
)

func main() {
	c := make(chan struct{}, 0)

	gobridge.RegisterCallback("createTransaction", internal.CreateTransaction)
	gobridge.RegisterCallback("createConvertTx", internal.CreateConvertTx)
	// js.Global().Set("decompressCoins", js.FuncOf(decompressCoins))
	// js.Global().Set("cacheCoins", js.FuncOf(cacheCoins))
	gobridge.RegisterCallback("newKeySetFromPrivate", internal.NewKeySetFromPrivate)
	gobridge.RegisterCallback("decryptCoin", internal.DecryptCoin)
	gobridge.RegisterCallback("createCoin", internal.CreateCoin)
	gobridge.RegisterCallback("generateBLSKeyPairFromSeed", internal.GenerateBLSKeyPairFromSeed)
	gobridge.RegisterCallback("hybridEncrypt", internal.HybridEncrypt)
	gobridge.RegisterCallback("hybridDecrypt", internal.HybridDecrypt)
	println("WASM bind complete !")
	<-c
}