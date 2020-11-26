package incognitokey

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"

	lru "github.com/hashicorp/golang-lru"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/bridgesig"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
)

type CommitteePublicKey struct {
	IncPubKey    privacy.PublicKey
	MiningPubKey map[string][]byte
}

func (pubKey *CommitteePublicKey) IsEqualMiningPubKey(consensusName string, k *CommitteePublicKey) bool {
	u, _ := pubKey.GetMiningKey(consensusName)
	b, _ := k.GetMiningKey(consensusName)
	return reflect.DeepEqual(u, b)
}

func NewCommitteePublicKey() *CommitteePublicKey {
	return &CommitteePublicKey{
		IncPubKey:    privacy.PublicKey{},
		MiningPubKey: make(map[string][]byte),
	}
}
func (pubKey *CommitteePublicKey) CheckSanityData() bool {
	if (len(pubKey.IncPubKey) != common.PublicKeySize) ||
		(len(pubKey.MiningPubKey[common.BlsConsensus]) != common.BLSPublicKeySize) ||
		(len(pubKey.MiningPubKey[common.BridgeConsensus]) != common.BriPublicKeySize) {
		return false
	}
	return true
}

func (pubKey *CommitteePublicKey) FromString(keyString string) error {
	keyBytes, ver, err := base58.Base58Check{}.Decode(keyString)
	if (ver != common.ZeroByte) || (err != nil) {
		return NewCashecError(B58DecodePubKeyErr, errors.New(ErrCodeMessage[B58DecodePubKeyErr].Message))
	}
	err = json.Unmarshal(keyBytes, pubKey)
	if err != nil {
		return NewCashecError(JSONError, errors.New(ErrCodeMessage[JSONError].Message))
	}
	return nil
}

func NewCommitteeKeyFromSeed(seed, incPubKey []byte) (CommitteePublicKey, error) {
	CommitteePublicKey := new(CommitteePublicKey)
	CommitteePublicKey.IncPubKey = incPubKey
	CommitteePublicKey.MiningPubKey = map[string][]byte{}
	_, blsPubKey := blsmultisig.KeyGen(seed)
	blsPubKeyBytes := blsmultisig.PKBytes(blsPubKey)
	CommitteePublicKey.MiningPubKey[common.BlsConsensus] = blsPubKeyBytes
	_, briPubKey := bridgesig.KeyGen(seed)
	briPubKeyBytes := bridgesig.PKBytes(&briPubKey)
	CommitteePublicKey.MiningPubKey[common.BridgeConsensus] = briPubKeyBytes
	return *CommitteePublicKey, nil
}

func (pubKey *CommitteePublicKey) FromBytes(keyBytes []byte) error {
	err := json.Unmarshal(keyBytes, pubKey)
	if err != nil {
		return NewCashecError(JSONError, err)
	}
	return nil
}

func (pubKey *CommitteePublicKey) RawBytes() ([]byte, error) {
	keys := make([]string, 0, len(pubKey.MiningPubKey))
	for k := range pubKey.MiningPubKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := pubKey.IncPubKey
	for _, k := range keys {
		res = append(res, pubKey.MiningPubKey[k]...)
	}
	return res, nil
}

func (pubKey *CommitteePublicKey) Bytes() ([]byte, error) {
	res, err := json.Marshal(pubKey)
	if err != nil {
		return []byte{0}, NewCashecError(JSONError, err)
	}
	return res, nil
}

func (pubKey *CommitteePublicKey) GetNormalKey() []byte {
	return pubKey.IncPubKey
}

func (pubKey *CommitteePublicKey) GetMiningKey(schemeName string) ([]byte, error) {
	allKey := map[string][]byte{}
	var ok bool
	allKey[schemeName], ok = pubKey.MiningPubKey[schemeName]
	if !ok {
		return nil, errors.New("this schemeName doesn't exist")
	}
	allKey[common.BridgeConsensus], ok = pubKey.MiningPubKey[common.BridgeConsensus]
	if !ok {
		return nil, errors.New("this lightweight schemeName doesn't exist")
	}
	result, err := json.Marshal(allKey)
	if err != nil {
		return nil, err
	}
	return result, nil
}

var GetMiningKeyBase58Cache, _ = lru.New(2000)
var ToBase58Cache, _ = lru.New(2000)

func (pubKey *CommitteePublicKey) GetMiningKeyBase58(schemeName string) string {
	b, _ := pubKey.RawBytes()
	key := schemeName + string(b)
	value, exist := GetMiningKeyBase58Cache.Get(key)
	if exist {
		return value.(string)
	}
	keyBytes, ok := pubKey.MiningPubKey[schemeName]
	if !ok {
		return ""
	}
	encodeData := base58.Base58Check{}.Encode(keyBytes, common.Base58Version)
	GetMiningKeyBase58Cache.Add(key, encodeData)
	return encodeData
}

func (pubKey *CommitteePublicKey) GetIncKeyBase58() string {
	return base58.Base58Check{}.Encode(pubKey.IncPubKey, common.Base58Version)
}

func (pubKey *CommitteePublicKey) ToBase58() (string, error) {
	if pubKey == nil {
		result, err := json.Marshal(pubKey)
		if err != nil {
			return "", err
		}
		return base58.Base58Check{}.Encode(result, common.Base58Version), nil
	}

	b, _ := pubKey.RawBytes()
	key := string(b)
	value, exist := ToBase58Cache.Get(key)
	if exist {
		return value.(string), nil
	}
	result, err := json.Marshal(pubKey)
	if err != nil {
		return "", err
	}
	encodeData := base58.Base58Check{}.Encode(result, common.Base58Version)
	ToBase58Cache.Add(key, encodeData)
	return encodeData, nil
}

func (pubKey *CommitteePublicKey) FromBase58(keyString string) error {
	keyBytes, ver, err := base58.Base58Check{}.Decode(keyString)
	if (ver != common.ZeroByte) || (err != nil) {
		return errors.New("wrong input")
	}
	return json.Unmarshal(keyBytes, pubKey)
}

type CommitteeKeyString struct {
	IncPubKey    string
	MiningPubKey map[string]string
}

func (committeePublicKey *CommitteePublicKey) IsValid(target CommitteePublicKey) bool {
	if bytes.Compare(committeePublicKey.IncPubKey[:], target.IncPubKey[:]) == 0 {
		return false
	}
	if committeePublicKey.MiningPubKey == nil || target.MiningPubKey == nil {
		return false
	}
	for key, value := range committeePublicKey.MiningPubKey {
		if targetValue, ok := target.MiningPubKey[key]; ok {
			if bytes.Compare(targetValue, value) == 0 {
				return false
			}
		}
	}
	return true
}

func (committeePublicKey *CommitteePublicKey) IsEqual(target CommitteePublicKey) bool {
	if bytes.Compare(committeePublicKey.IncPubKey[:], target.IncPubKey[:]) != 0 {
		return false
	}
	if committeePublicKey.MiningPubKey == nil && target.MiningPubKey != nil {
		return false
	}
	for key, value := range committeePublicKey.MiningPubKey {
		if targetValue, ok := target.MiningPubKey[key]; !ok {
			return false
		} else {
			if bytes.Compare(targetValue, value) != 0 {
				return false
			}
		}
	}
	return true
}

func DeepCopy(committeePublicKeys []CommitteePublicKey) []CommitteePublicKey {
	newCommitteePublicKeys := []CommitteePublicKey{}
	for _, committeePublicKey := range committeePublicKeys {
		res, _ := committeePublicKey.Bytes()
		newCommitteePublicKey := new(CommitteePublicKey)
		_ = newCommitteePublicKey.FromBytes(res)
		newCommitteePublicKeys = append(newCommitteePublicKeys, *newCommitteePublicKey)
	}
	return newCommitteePublicKeys
}
