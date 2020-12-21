package main

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wallet"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
)

type committeePubKey struct {
	Beacon []map[string]string            `json:"Beacon"`
	Shard  map[string][]map[string]string `json:"Shard"`
}

func main() {
	salt := []byte(os.Getenv("SALT"))

	// load nodes file
	oldNodeFilePath := os.Getenv("OLD_NODES")
	fmt.Println("Read file from " + oldNodeFilePath)
	oldNodeFile, err := ioutil.ReadFile(oldNodeFilePath)
	if err != nil {
		log.Printf("oldNodeFile.Get err   #%v ", err)
		return
	}
	contentNodesFile := make(map[string]interface{})
	err = yaml.Unmarshal(oldNodeFile, &contentNodesFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	// load old key committee pub file
	oldPubkeyPath := os.Getenv("OLD_PUBKEYS")
	fmt.Println("Read file from " + oldPubkeyPath)
	oldCommitteeKeyFile, err := ioutil.ReadFile(oldPubkeyPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	committeePubKey := committeePubKey{
		Beacon: []map[string]string{},
		Shard:  make(map[string][]map[string]string),
	}
	err = json.Unmarshal(oldCommitteeKeyFile, &committeePubKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println(contentNodesFile)
	//fmt.Println(committeePubKey)

	//---- beacon
	beacons := contentNodesFile["beacon"].(map[interface{}]interface{})
	// sort keys
	keys := []int{}
	for i, _ := range beacons {
		keys = append(keys, i.(int))
	}
	sort.Ints(keys)

	for _, i := range keys {
		beacon := beacons[i]
		beaconMap := beacon.(map[interface{}]interface{})
		_ = i
		key := beaconMap["KEY"].(string)
		validatorKeyByte, _, _ := base58.Base58Check{}.Decode(key)
		if len(salt) > 0 {
			validatorKeyByte = append(validatorKeyByte, salt...)
			validatorKeyByte = common.HashB(common.HashB(validatorKeyByte))
		}
		newKey := base58.Base58Check{}.Encode(validatorKeyByte, 0x0)
		beaconMap["KEY"] = newKey

		//
		oldPaymentAddress := committeePubKey.Beacon[i]["PaymentAddress"]
		k, _ := wallet.Base58CheckDeserialize(oldPaymentAddress)
		committeeKeyStr, _ := incognitokey.NewCommitteeKeyFromSeed(validatorKeyByte, k.KeySet.PaymentAddress.Pk)
		committeeKey, _ := incognitokey.CommitteeKeyListToString([]incognitokey.CommitteePublicKey{committeeKeyStr})
		beaconPublicKey := map[string]string{
			"PaymentAddress":     oldPaymentAddress,
			"CommitteePublicKey": committeeKey[0],
		}
		committeePubKey.Beacon[i] = beaconPublicKey

		fmt.Printf("New key beacon-%d: %v -> [validator key: %s] [committee pub key: %s] [payment address: %s] \n", i, key, beaconMap["KEY"], committeeKey[0], oldPaymentAddress)
	}

	//------ shard
	shardGroups := contentNodesFile["shard"].(map[interface{}]interface{})
	for shardId, shards := range shardGroups {
		shardsMap := shards.(map[interface{}]interface{})

		// sort keys
		keys := []int{}
		for i, _ := range shardsMap {
			keys = append(keys, i.(int))
		}
		sort.Ints(keys)

		for _, i := range keys {
			shard := shardsMap[i]
			shardMap := shard.(map[interface{}]interface{})
			key := shardMap["KEY"].(string)
			newValidatorKeyByte, _, _ := base58.Base58Check{}.Decode(key)
			if len(salt) > 0 {
				newValidatorKeyByte = append(newValidatorKeyByte, salt...)
				newValidatorKeyByte = common.HashB(common.HashB(newValidatorKeyByte))
			}
			newKey := base58.Base58Check{}.Encode(newValidatorKeyByte, 0x0)
			shardMap["KEY"] = string(newKey)
			//fmt.Printf("New key shard%d-%d: %s -> %s \n", shardId, i, key, shardMap["KEY"])

			//
			oldPaymentAddress := committeePubKey.Shard[strconv.Itoa(shardId.(int))][i]["PaymentAddress"]
			k, _ := wallet.Base58CheckDeserialize(oldPaymentAddress)
			committeeKeyStr, _ := incognitokey.NewCommitteeKeyFromSeed(newValidatorKeyByte, k.KeySet.PaymentAddress.Pk)
			committeeKey, _ := incognitokey.CommitteeKeyListToString([]incognitokey.CommitteePublicKey{committeeKeyStr})
			shardPublicKey := map[string]string{
				"PaymentAddress":     oldPaymentAddress,
				"CommitteePublicKey": committeeKey[0],
			}
			committeePubKey.Shard[strconv.Itoa(shardId.(int))][i] = shardPublicKey

			fmt.Printf("New key shard%d-%d: %s -> [validator key: %s] [committee pub key: %s] [payment address: %s] \n", shardId, i, key, shardMap["KEY"], committeeKey[0], oldPaymentAddress)
		}
	}

	fmt.Println()
	fmt.Println()
	fmt.Println()

	rewriteData, err := yaml.Marshal(contentNodesFile)
	//fmt.Println(string(rewriteData))
	outputFileNodeServer := os.Getenv("NEW_NODES")
	os.Remove(outputFileNodeServer)
	os.Create(outputFileNodeServer)
	err = ioutil.WriteFile(outputFileNodeServer, rewriteData, 0x666)
	if err != nil {
		fmt.Println(err)
	}

	publicData, err := json.MarshalIndent(committeePubKey, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	outputFileCommitteePublicKey := os.Getenv("NEW_PUBKEYS")
	os.Remove(outputFileCommitteePublicKey)
	os.Create(outputFileCommitteePublicKey)
	err = ioutil.WriteFile(outputFileCommitteePublicKey, publicData, 0x666)
	if err != nil {
		fmt.Println(err)
	}
}
