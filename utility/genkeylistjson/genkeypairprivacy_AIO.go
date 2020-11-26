//+build !test

package main

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wallet"
	"io/ioutil"
	"os"
	"bufio"
)
// This script generate keylist.json file with private key in it

type AccountPub struct {
	PaymentAddress string
    PrivateKey  string
	CommitteePublicKey string
	ValidatorKey       string
}

const NUM_OF_SHARD = 2
const NUM_OF_FIX_VALIDATOR = 5
const NUM_OF_BEACON = 4
const NUM_OF_STAKER = 16
const SEED = "send nude"

func main() {
	mnemonicGen := wallet.MnemonicGenerator{}
	//Entropy, _ := mnemonicGen.NewEntropy(128)
	//Mnemonic, _ := mnemonicGen.NewMnemonic(Entropy)
	fmt.Printf("Input mnemonic seed (default: %s) :",SEED)
    reader := bufio.NewReader(os.Stdin)
    MNEMONIC, _ := reader.ReadString('\n')
    if MNEMONIC == "\n" {
        MNEMONIC = SEED
    }
	fmt.Printf("Mnemonic: %s\n", MNEMONIC)
	return
	Seed := mnemonicGen.NewSeed(MNEMONIC, "dmnkajdklawjdkjawk")
	saltForvalidatorKey := []byte{} // default is empty for original generate

	key, _ := wallet.NewMasterKey(Seed)
	var i = 0
	var j = 0

	listAcc := make(map[int][]AccountPub)
	beaconAcc := []AccountPub{}
	stakerAcc := []AccountPub{}

	for j = 0; j < NUM_OF_SHARD; j++ {
		listAcc[j] = []AccountPub{}
	}

	for { // generate shard validator accounts
		child, _ := key.NewChildKey(uint32(i))
		i++
		privAddr := child.Base58CheckSerialize(wallet.PriKeyType)
		paymentAddress := child.Base58CheckSerialize(wallet.PaymentAddressType)

		committeeValidatorKeyByte := common.HashB(common.HashB(child.KeySet.PrivateKey)) // old validator key
		if len(saltForvalidatorKey) > 0 {
			committeeValidatorKeyByte = append(committeeValidatorKeyByte, saltForvalidatorKey...)
			committeeValidatorKeyByte = common.HashB(common.HashB(committeeValidatorKeyByte))
		}

		committeeValidatorKeyBase58 := base58.Base58Check{}.Encode(committeeValidatorKeyByte, 0x0)
		committeeKeyStr, _ := incognitokey.NewCommitteeKeyFromSeed(committeeValidatorKeyByte, child.KeySet.PaymentAddress.Pk)
		committeeKey, _ := incognitokey.CommitteeKeyListToString([]incognitokey.CommitteePublicKey{committeeKeyStr})
		shardID := int(child.KeySet.PaymentAddress.Pk[len(child.KeySet.PaymentAddress.Pk)-1])
		if shardID >= NUM_OF_SHARD {
			continue
		}

		if len(listAcc[shardID]) < NUM_OF_FIX_VALIDATOR {
			listAcc[shardID] = append(listAcc[shardID], AccountPub{paymentAddress, privAddr, committeeKey[0], committeeValidatorKeyBase58})
		}

		shouldBreak := true
		for i, _ := range listAcc {
			if len(listAcc[i]) < NUM_OF_FIX_VALIDATOR {
				shouldBreak = false
			}
		}

		if shouldBreak {
			break
		}
	}

	for { // generate beacon accounts
		child, _ := key.NewChildKey(uint32(i))
		i++
		privAddr := child.Base58CheckSerialize(wallet.PriKeyType)
		paymentAddress := child.Base58CheckSerialize(wallet.PaymentAddressType)

		validatorKeyBytes := common.HashB(common.HashB(child.KeySet.PrivateKey)) // old validator key
		if len(saltForvalidatorKey) > 0 {
			validatorKeyBytes = append(validatorKeyBytes, saltForvalidatorKey...)
			validatorKeyBytes = common.HashB(common.HashB(validatorKeyBytes))
		}
		validatorKeyBase58 := base58.Base58Check{}.Encode(validatorKeyBytes, 0x0)
		committeeKeyStr, _ := incognitokey.NewCommitteeKeyFromSeed(validatorKeyBytes, child.KeySet.PaymentAddress.Pk)
		committeeKey, _ := incognitokey.CommitteeKeyListToString([]incognitokey.CommitteePublicKey{committeeKeyStr})
		if len(beaconAcc) < NUM_OF_BEACON {
			beaconAcc = append(beaconAcc, AccountPub{paymentAddress,privAddr, committeeKey[0], validatorKeyBase58})
		} else {
			break
		}

	}

	for { // generate staker accounts
		child, _ := key.NewChildKey(uint32(i))
		i++
		privAddr := child.Base58CheckSerialize(wallet.PriKeyType)
		paymentAddress := child.Base58CheckSerialize(wallet.PaymentAddressType)

		validatorKeyBytes := common.HashB(common.HashB(child.KeySet.PrivateKey)) // old validator key
		if len(saltForvalidatorKey) > 0 {
			validatorKeyBytes = append(validatorKeyBytes, saltForvalidatorKey...)
			validatorKeyBytes = common.HashB(common.HashB(validatorKeyBytes))
		}
		validatorKeyBase58 := base58.Base58Check{}.Encode(validatorKeyBytes, 0x0)
		committeeKeyStr, _ := incognitokey.NewCommitteeKeyFromSeed(validatorKeyBytes, child.KeySet.PaymentAddress.Pk)
		committeeKey, _ := incognitokey.CommitteeKeyListToString([]incognitokey.CommitteePublicKey{committeeKeyStr})
		if len(stakerAcc) < NUM_OF_STAKER {
			stakerAcc = append(stakerAcc, AccountPub{paymentAddress,privAddr, committeeKey[0], validatorKeyBase58})
		} else {
			break
		}

	}

	data, _ := json.MarshalIndent(struct {
        Beacon []AccountPub
		Shard  map[int][]AccountPub
		Staker []AccountPub
	}{
        beaconAcc,
		listAcc,
		stakerAcc,
	},"","   ")

	os.Remove("keylist.json")
	ioutil.WriteFile("keylist.json", data, 0x766)

}
