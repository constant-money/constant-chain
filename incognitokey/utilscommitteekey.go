package incognitokey

import (
	"encoding/json"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
)

func ExtractPublickeysFromCommitteeKeyList(keyList []CommitteePublicKey, keyType string) ([]string, error) {
	result := []string{}
	for _, keySet := range keyList {
		key := keySet.GetMiningKeyBase58(keyType)
		if key != "" {
			result = append(result, key)
		}
	}
	return result, nil
}

func ExtractMiningPublickeysFromCommitteeKeyList(keyList []CommitteePublicKey, keyType string) ([]string, error) {
	result := []string{}
	for _, keySet := range keyList {
		key, err := keySet.GetMiningKey(keyType)
		if err != nil {
			return nil, err
		}
		if string(key) != "" {
			result = append(result, string(key))
		}
	}
	return result, nil
}

func CommitteeKeyListToString(keyList []CommitteePublicKey) ([]string, error) {
	result := []string{}
	for _, key := range keyList {
		keyStr, err := key.ToBase58()
		if err != nil {
			return nil, err
		}
		result = append(result, keyStr)
	}
	return result, nil
}

func CommitteeBase58KeyListToStruct(strKeyList []string) ([]CommitteePublicKey, error) {
	if len(strKeyList) == 0 {
		return []CommitteePublicKey{}, nil
	}
	if len(strKeyList) == 1 && len(strKeyList[0]) == 0 {
		return []CommitteePublicKey{}, nil
	}
	result := []CommitteePublicKey{}
	for _, key := range strKeyList {
		var keyStruct CommitteePublicKey
		if err := keyStruct.FromString(key); err != nil {
			return nil, err
		}
		result = append(result, keyStruct)
	}
	return result, nil
}
func IsInBase58ShortFormat(strKeyList []string) bool {
	tempStruct, err := CommitteeBase58KeyListToStruct(strKeyList)
	if err != nil {
		return false
	}
	tempString, err := CommitteeKeyListToString(tempStruct)
	if len(tempString) != len(strKeyList) {
		return false
	}
	for index, value := range tempString {
		if value != strKeyList[index] {
			return false
		}
	}
	return true
}

func ConvertToBase58ShortFormat(strKeyList []string) ([]string, error) {
	tempStruct, err := CommitteeBase58KeyListToStruct(strKeyList)
	if err != nil {
		return []string{}, err
	}
	tempString, err := CommitteeKeyListToString(tempStruct)
	if err != nil {
		return []string{}, err
	}
	return tempString, nil
}

/*func IsEqualCommitteeKey(keyString1 string, keyString2 string) bool {
	var pubKey1 CommitteePublicKey
	var pubKey2 CommitteePublicKey
	keyBytes1, ver, err := base58.Base58Check{}.Decode(keyString1)
	if (ver != common.ZeroByte) || (err != nil) {
		// errors.New("wrong input")
		return false
	}
	keyBytes2, ver, err := base58.Base58Check{}.Decode(keyString2)
	if (ver != common.ZeroByte) || (err != nil) {
		// return errors.New("wrong input")
		return false
	}
	err = json.Unmarshal(keyBytes1, pubKey1)
	if err != nil {
		// return errors.New("wrong input")
		return false
	}
	err = json.Unmarshal(keyBytes2, pubKey2)
	if err != nil {
		// return errors.New("wrong input")
		return false
	}
	if reflect.DeepEqual(pubKey1, pubKey2) {
		return true
	}
	return false
}*/

func IsOneMiner(keyString1 string, keyString2 string) bool {
	var pubKey1 CommitteePublicKey
	var pubKey2 CommitteePublicKey
	keyBytes1, ver, err := base58.Base58Check{}.Decode(keyString1)
	if (ver != common.ZeroByte) || (err != nil) {
		// errors.New("wrong input")
		return false
	}
	keyBytes2, ver, err := base58.Base58Check{}.Decode(keyString2)
	if (ver != common.ZeroByte) || (err != nil) {
		// return errors.New("wrong input")
		return false
	}
	err = json.Unmarshal(keyBytes1, pubKey1)
	if err != nil {
		// return errors.New("wrong input")
		return false
	}
	err = json.Unmarshal(keyBytes2, pubKey2)
	if err != nil {
		// return errors.New("wrong input")
		return false
	}
	if reflect.DeepEqual(pubKey1.MiningPubKey, pubKey2.MiningPubKey) {
		return true
	}
	return false
}

func GetValidStakeStructCommitteePublicKey(committees []CommitteePublicKey, stakers []CommitteePublicKey) []CommitteePublicKey {
	validStaker := []CommitteePublicKey{}
	for _, staker := range stakers {
		flag := true
		for _, committee := range committees {
			if !staker.IsValid(committee) {
				flag = false
				break
			}
		}
		if flag {
			validStaker = append(validStaker, staker)
		}
	}
	return validStaker
}

func CommitteeKeyListToStringList(keyList []CommitteePublicKey) []CommitteeKeyString {
	result := []CommitteeKeyString{}
	for _, key := range keyList {
		var keyMap CommitteeKeyString
		keyMap.IncPubKey = key.GetIncKeyBase58()
		keyMap.MiningPubKey = make(map[string]string)
		for keyType := range key.MiningPubKey {
			keyMap.MiningPubKey[keyType] = key.GetMiningKeyBase58(keyType)
		}
		result = append(result, keyMap)
	}
	return result
}

func IndexOfCommitteeKey(item CommitteePublicKey, list []CommitteePublicKey) int {
	for k, v := range list {
		if item.IsEqual(v) {
			return k
		}
	}
	return -1
}
