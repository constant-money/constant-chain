package common

import (
	"testing"

	"github.com/incognitochain/incognito-chain/wallet"
)

func TestGenBTCPrivateKey(t *testing.T) {
	IncKeyStr := "112t8rnXRDT21fsx5UYR1kGd8yjiygUS3tXfhcRfXy2nmJS3U39vkf76wbQsXguwhHwN2EtBF4YZJ8o1i7MMF9BsKngcgxfkCZBa5P3Fq9xp"
	keyWallet, err := wallet.Base58CheckDeserialize(IncKeyStr)
	if err != nil {
		return
	}
	IncKeyBytes := keyWallet.KeySet.PrivateKey
	BTCKeyBytes := GenBTCPrivateKey(IncKeyBytes)

	t.Logf("IncKey: %+v\n", IncKeyBytes)
	t.Logf("BTCKey: %+v\n", BTCKeyBytes)
}
