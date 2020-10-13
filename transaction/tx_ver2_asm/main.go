//+build linux,386 wasm

package main

import (
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/internal"
	"syscall/js"
)

var stopper chan int

func createTx(_ js.Value, _js_inputs []js.Value) interface{}{
	if len(_js_inputs)<2{
		println("Invalid parameters")
		return nil
	}
	result, err := internal.CreateTransaction(_js_inputs[0].String(), int64(_js_inputs[1].Int()))
	if err != nil {
		return nil
	}
	return result
}

func decompressCoins(_ js.Value, args []js.Value) interface{} {
	if len(args)<1{
		println("Invalid parameters")
		return nil
	}
	result, err := internal.DecompressCoins(args[0].String())
	if err != nil {
		return nil
	}

	return result
}

func cacheCoins(_ js.Value, args []js.Value) interface{} {
	if len(args)<2{
		println("Invalid parameters")
		return nil
	}
	result, err := internal.CacheCoins(args[0].String(), args[1].String())
	if err != nil {
		return nil
	}

	return result
}

func main() {
	stopper = make(chan int, 0)
	println("WASM resource loaded !")

	js.Global().Set("createTransaction", js.FuncOf(createTx))
	js.Global().Set("decompressCoins", js.FuncOf(decompressCoins))
	js.Global().Set("cacheCoins", js.FuncOf(cacheCoins))

	<-stopper
	println("Exited !")
}
