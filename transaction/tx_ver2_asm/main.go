//+build linux,386 wasm

package main

import (
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/gobridge"
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/internal"
)

func main() {
	c := make(chan struct{}, 0)

	gobridge.RegisterCallback("createTransaction", internal.CreateTransaction)
	// js.Global().Set("decompressCoins", js.FuncOf(decompressCoins))
	// js.Global().Set("cacheCoins", js.FuncOf(cacheCoins))
	gobridge.RegisterCallback("newKeySetFromPrivate", internal.NewKeySetFromPrivate)
	gobridge.RegisterCallback("decryptCoin", internal.DecryptCoin)
	gobridge.RegisterCallback("createCoin", internal.CreateCoin)
	gobridge.RegisterCallback("generateBLSKeyPairFromSeed", internal.GenerateBLSKeyPairFromSeed)
	println("WASM bind complete !")
	<-c
}

// func createTx(_ js.Value, args []js.Value) interface{} {
// 	if len(args)<2{
// 		println("Invalid parameters")
// 		return nil
// 	}
// 	result, err := internal.CreateTransaction(args[0].String())
// 	if err != nil {
// 		return nil
// 	}

// 	return result
// }

// func decompressCoins(_ js.Value, args []js.Value) interface{} {
// 	if len(args)<1{
// 		println("Invalid parameters")
// 		return nil
// 	}
// 	result, err := internal.DecompressCoins(args[0].String())
// 	if err != nil {
// 		return nil
// 	}

// 	return result
// }

// func cacheCoins(_ js.Value, args []js.Value) interface{} {
// 	if len(args)<2{
// 		println("Invalid parameters")
// 		return nil
// 	}
// 	result, err := internal.CacheCoins(args[0].String(), args[1].String())
// 	if err != nil {
// 		return nil
// 	}

// 	return result
// }

// func newKeySetFromPrivate(_ js.Value, args []js.Value) interface{}{
// 	if len(args)<1{
// 		println("Invalid parameters")
// 		return nil
// 	}
// 	result, err := internal.NewKeySetFromPrivate(args[0].String())
// 	if err != nil {
// 		return nil
// 	}

// 	return result
// }

// func decryptCoin(_ js.Value, args []js.Value) interface{}{
// 	if len(args)<1{
// 		println("Invalid parameters")
// 		return nil
// 	}
// 	result, err := internal.DecryptCoin(args[0].String())
// 	if err != nil {
// 		return nil
// 	}

// 	return result
// }

// func generateBLSKeyPairFromSeed(_ js.Value, args []js.Value) interface{} {
// 	if len(args)<1{
// 		println("Invalid parameters")
// 		return nil
// 	}
// 	result, err := internal.GenerateBLSKeyPairFromSeed(args[0].String())
// 	if err != nil {
// 		return nil
// 	}

// 	return result
// }