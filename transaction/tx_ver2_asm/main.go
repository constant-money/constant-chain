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
	
	gobridge.RegisterCallback("newKeySetFromPrivate", internal.NewKeySetFromPrivate)
	gobridge.RegisterCallback("decryptCoin", internal.DecryptCoin)
	gobridge.RegisterCallback("createCoin", internal.CreateCoin)
	gobridge.RegisterCallback("generateBLSKeyPairFromSeed", internal.GenerateBLSKeyPairFromSeed)
	gobridge.RegisterCallback("hybridEncrypt", internal.HybridEncrypt)
	gobridge.RegisterCallback("hybridDecrypt", internal.HybridDecrypt)

	// compat
	gobridge.RegisterCallback("initPrivacyTx", internal.CreateTransaction)
	gobridge.RegisterCallback("staking", internal.CreateTransaction)
	gobridge.RegisterCallback("stopAutoStaking", internal.CreateTransaction)
	gobridge.RegisterCallback("initPrivacyTokenTx", internal.CreateTransaction)
	gobridge.RegisterCallback("initBurningRequestTx", internal.CreateTransaction)
	gobridge.RegisterCallback("initWithdrawRewardTx", internal.CreateTransaction)

	gobridge.RegisterCallback("generateKeyFromSeed", internal.GenerateKeyFromSeed)
	gobridge.RegisterCallback("scalarMultBase", internal.ScalarMultBase)
	gobridge.RegisterCallback("randomScalars", internal.RandomScalars)
	gobridge.RegisterCallback("getSignPublicKey", internal.GetSignPublicKey)
	gobridge.RegisterCallback("signPoolWithdraw", internal.SignPoolWithdraw)
	gobridge.RegisterCallback("verifySign", internal.VerifySign)
	// gobridge.RegisterCallback("generateBLSKeyPairFromSeed", internal.GenerateBLSKeyPairFromSeed)
	gobridge.RegisterCallback("initPRVContributionTx", internal.CreateTransaction)
	gobridge.RegisterCallback("initPTokenContributionTx", internal.CreateTransaction)
	gobridge.RegisterCallback("initPRVTradeTx", internal.CreateTransaction)
	gobridge.RegisterCallback("initPTokenTradeTx", internal.CreateTransaction)
	gobridge.RegisterCallback("withdrawDexTx", internal.CreateTransaction)
	gobridge.RegisterCallback("hybridEncryptionASM", internal.HybridEncrypt)
	gobridge.RegisterCallback("hybridDecryptionASM", internal.HybridDecrypt)

	// not applicable
	// gobridge.RegisterCallback("deriveSerialNumber", internal.DeriveSerialNumber)

	println("WASM loading finished")
	<-c
}