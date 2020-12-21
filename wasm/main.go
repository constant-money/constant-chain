//+build linux,386 wasm

package main

import (
	"github.com/incognitochain/incognito-chain/wasm/gomobile"
	"syscall/js"
)

//func aggregatedRangeProve(_ js.Value, args []js.Value) interface{} {
//	return gomobile.AggregatedRangeProve(args[0].String())
//}
//func oneOutOfManyProve(_ js.Value, args []js.Value) interface{} {
//	println("args.len :", len(args))
//	proof, err := gomobile.OneOutOfManyProve(args[0].String())
//	if err != nil {
//		return nil
//	}
//	return proof
//}

func generateBLSKeyPairFromSeed(_ js.Value, args []js.Value) interface{} {
	return gomobile.GenerateBLSKeyPairFromSeed(args[0].String())
}

func generateKeyFromSeed(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.GenerateKeyFromSeed(args[0].String())
	if err != nil {
		return nil
	}

	println("[Go] Result: ", result)

	return result
}

func scalarMultBase(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.ScalarMultBase(args[0].String())
	if err != nil {
		return nil
	}

	println("[Go] Result: ", result)

	return result
}

func deriveSerialNumber(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.DeriveSerialNumber(args[0].String())
	if err != nil {
		return nil
	}

	println("[Go] Result: ", result)

	return result
}

func randomScalars(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.RandomScalars(args[0].String())
	if err != nil {
		return nil
	}

	println("[Go] Result: ", result)

	return result
}

func initPrivacyTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitPrivacyTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func stopAutoStaking(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.StopAutoStaking(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func staking(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.Staking(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func initPrivacyTokenTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitPrivacyTokenTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func initBurningRequestTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitBurningRequestTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func initWithdrawRewardTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitWithdrawRewardTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func initPRVContributionTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitPRVContributionTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func initPTokenContributionTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitPTokenContributionTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func initPRVTradeTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitPRVTradeTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func initPTokenTradeTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.InitPTokenTradeTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func withdrawDexTx(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.WithdrawDexTx(args[0].String(), int64(args[1].Int()))
	if err != nil {
		return nil
	}

	return result
}

func hybridEncryptionASM(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.HybridEncryptionASM(args[0].String())
	if err != nil {
		return nil
	}

	return result
}

func hybridDecryptionASM(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.HybridDecryptionASM(args[0].String())
	if err != nil {
		return nil
	}

	return result
}

func getSignPublicKey(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.GetSignPublicKey(args[0].String())
	if err != nil {
		return nil
	}

	return result
}

func signPoolWithdraw(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.SignPoolWithdraw(args[0].String())
	if err != nil {
		return nil
	}

	return result
}

func verifySign(_ js.Value, args []js.Value) interface{} {
	result, err := gomobile.VerifySign(args[0].String(), args[1].String(), args[2].String(), args[3].String())
	if err != nil {
		return nil
	}

	return result
}

func main() {
	c := make(chan struct{}, 0)
	println("Hello WASM")
	//js.Global().Set("aggregatedRangeProve", js.FuncOf(aggregatedRangeProve))
	//js.Global().Set("oneOutOfManyProve", js.FuncOf(oneOutOfManyProve))

	js.Global().Set("initPrivacyTx", js.FuncOf(initPrivacyTx))
	js.Global().Set("staking", js.FuncOf(staking))
	js.Global().Set("stopAutoStaking", js.FuncOf(stopAutoStaking))
	js.Global().Set("initPrivacyTokenTx", js.FuncOf(initPrivacyTokenTx))
	js.Global().Set("initBurningRequestTx", js.FuncOf(initBurningRequestTx))
	js.Global().Set("initWithdrawRewardTx", js.FuncOf(initWithdrawRewardTx))
	js.Global().Set("deriveSerialNumber", js.FuncOf(deriveSerialNumber))

	js.Global().Set("generateKeyFromSeed", js.FuncOf(generateKeyFromSeed))
	js.Global().Set("scalarMultBase", js.FuncOf(scalarMultBase))
	js.Global().Set("randomScalars", js.FuncOf(randomScalars))
	js.Global().Set("generateBLSKeyPairFromSeed", js.FuncOf(generateBLSKeyPairFromSeed))

	js.Global().Set("initPRVContributionTx", js.FuncOf(initPRVContributionTx))
	js.Global().Set("initPTokenContributionTx", js.FuncOf(initPTokenContributionTx))
	js.Global().Set("initPRVTradeTx", js.FuncOf(initPRVTradeTx))
	js.Global().Set("initPTokenTradeTx", js.FuncOf(initPTokenTradeTx))
	js.Global().Set("withdrawDexTx", js.FuncOf(withdrawDexTx))

	js.Global().Set("hybridEncryptionASM", js.FuncOf(hybridEncryptionASM))
	js.Global().Set("hybridDecryptionASM", js.FuncOf(hybridDecryptionASM))

	js.Global().Set("getSignPublicKey", js.FuncOf(getSignPublicKey))
	js.Global().Set("signPoolWithdraw", js.FuncOf(signPoolWithdraw))
	js.Global().Set("verifySign", js.FuncOf(verifySign))

	<-c
}
