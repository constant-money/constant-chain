package wallet

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

// burnAddress1BytesDecode is a decoded bytes array of old burning address "15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs"
// burnAddress1BytesDecode,_, err := base58.Base58Check{}.Decode("15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs")
var burnAddress1BytesDecode = []byte{1, 32, 99, 183, 246, 161, 68, 172, 228, 222, 153, 9, 172, 39, 208, 245, 167, 79, 11, 2, 114, 65, 241, 69, 85, 40, 193, 104, 199, 79, 70, 4, 53, 0, 0, 163, 228, 236, 208}

// KeyWallet represents with bip32 standard
type KeyWallet struct {
	Depth       byte   // 1 bytes
	ChildNumber []byte // 4 bytes
	ChainCode   []byte // 32 bytes
	KeySet      incognitokey.KeySet
}

// NewMasterKey creates a new master extended PubKey from a Seed
// Seed is a bytes array which any size
func NewMasterKey(seed []byte) (*KeyWallet, error) {
	// Generate PubKey and chaincode
	hmacObj := hmac.New(sha512.New, []byte("Incognito Seed"))
	_, err := hmacObj.Write(seed)
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	intermediary := hmacObj.Sum(nil)

	// Split it into our PubKey and chain code
	keyBytes := intermediary[:32]  // use to create master private/public keypair
	chainCode := intermediary[32:] // be used with public PubKey (in keypair) for new Child keys

	keySet := (&incognitokey.KeySet{}).GenerateKey(keyBytes)

	// Create the PubKey struct
	key := &KeyWallet{
		ChainCode:   chainCode,
		KeySet:      *keySet,
		Depth:       0x00,
		ChildNumber: []byte{0x00, 0x00, 0x00, 0x00},
	}

	return key, nil
}

// NewChildKey derives a Child KeyWallet from a given parent as outlined by bip32
// 2 child keys is derived from one key and a same child index are the same
func (key *KeyWallet) NewChildKey(childIdx uint32) (*KeyWallet, error) {
	intermediary, err := key.getIntermediary(childIdx)
	if err != nil {
		return nil, NewWalletError(NewChildKeyError, err)
	}

	newSeed := []byte{}
	newSeed = append(newSeed[:], intermediary[:32]...)
	newKeyset := (&incognitokey.KeySet{}).GenerateKey(newSeed)
	// Create Child KeySet with data common to all both scenarios
	childKey := &KeyWallet{
		ChildNumber: common.Uint32ToBytes(childIdx),
		ChainCode:   intermediary[32:],
		Depth:       key.Depth + 1,
		KeySet:      *newKeyset,
	}

	return childKey, nil
}

// getIntermediary
func (key *KeyWallet) getIntermediary(childIdx uint32) ([]byte, error) {
	childIndexBytes := common.Uint32ToBytes(childIdx)

	var data []byte
	data = append(data, childIndexBytes...)

	hmacObj := hmac.New(sha512.New, key.ChainCode)
	_, err := hmacObj.Write(data)
	if err != nil {
		return nil, err
	}
	return hmacObj.Sum(nil), nil
}

// Serialize receives keyType and serializes key which has keyType to bytes array
// and append 4-byte checksum into bytes array
func (key *KeyWallet) Serialize(keyType byte) ([]byte, error) {
	// Write fields to buffer in order
	buffer := new(bytes.Buffer)
	buffer.WriteByte(keyType)
	if keyType == PriKeyType {
		buffer.WriteByte(key.Depth)
		buffer.Write(key.ChildNumber)
		buffer.Write(key.ChainCode)

		// Private keys should be prepended with a single null byte
		keyBytes := make([]byte, 0)
		keyBytes = append(keyBytes, byte(len(key.KeySet.PrivateKey))) // set length
		keyBytes = append(keyBytes, key.KeySet.PrivateKey[:]...)      // set pri-key
		buffer.Write(keyBytes)
	} else if keyType == PaymentAddressType {
		keyBytes := make([]byte, 0)
		keyBytes = append(keyBytes, byte(len(key.KeySet.PaymentAddress.Pk))) // set length PaymentAddress
		keyBytes = append(keyBytes, key.KeySet.PaymentAddress.Pk[:]...)      // set PaymentAddress

		keyBytes = append(keyBytes, byte(len(key.KeySet.PaymentAddress.Tk))) // set length Pkenc
		keyBytes = append(keyBytes, key.KeySet.PaymentAddress.Tk[:]...)      // set Pkenc

		if len(key.KeySet.PaymentAddress.OTAPublic) > 0 {
			keyBytes = append(keyBytes, byte(len(key.KeySet.PaymentAddress.OTAPublic))) // set length OTAPublicKey
			keyBytes = append(keyBytes, key.KeySet.PaymentAddress.OTAPublic[:]...)      // set OTAPublicKey
		}

		buffer.Write(keyBytes)
	} else if keyType == ReadonlyKeyType {
		keyBytes := make([]byte, 0)
		keyBytes = append(keyBytes, byte(len(key.KeySet.ReadonlyKey.Pk))) // set length PaymentAddress
		keyBytes = append(keyBytes, key.KeySet.ReadonlyKey.Pk[:]...)      // set PaymentAddress

		keyBytes = append(keyBytes, byte(len(key.KeySet.ReadonlyKey.Rk))) // set length Skenc
		keyBytes = append(keyBytes, key.KeySet.ReadonlyKey.Rk[:]...)      // set Pkenc
		buffer.Write(keyBytes)
	} else if keyType == OTAKeyType {
		keyBytes := make([]byte, 0)
		keyBytes = append(keyBytes, byte(len(key.KeySet.OTAKey.GetPublicSpend().ToBytesS()))) // set length publicSpend
		keyBytes = append(keyBytes, key.KeySet.OTAKey.GetPublicSpend().ToBytesS()[:]...)      // set publicSpend

		keyBytes = append(keyBytes, byte(len(key.KeySet.OTAKey.GetOTASecretKey().ToBytesS()))) // set length OTASecretKey
		keyBytes = append(keyBytes, key.KeySet.OTAKey.GetOTASecretKey().ToBytesS()[:]...)      // set OTASecretKey
		buffer.Write(keyBytes)
	}else{
		return []byte{}, NewWalletError(InvalidKeyTypeErr, nil)
	}

	// Append the standard doublesha256 checksum
	serializedKey, err := key.addChecksumToBytes(buffer.Bytes())
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}

	return serializedKey, nil
}

func (key KeyWallet) addChecksumToBytes(data []byte) ([]byte, error) {
	checksum := base58.ChecksumFirst4Bytes(data)
	return append(data, checksum...), nil
}

// Base58CheckSerialize encodes the key corresponding to keyType in KeySet
// in the standard Incognito base58 encoding
// It returns the encoding string of the key
func (key *KeyWallet) Base58CheckSerialize(keyType byte) string {
	serializedKey, err := key.Serialize(keyType)
	if err != nil {
		return ""
	}

	return base58.Base58Check{}.Encode(serializedKey, common.ZeroByte)
}

// Deserialize receives a byte array and deserializes into KeySet
// because data contains keyType and serialized data of corresponding key
// it returns KeySet just contain corresponding key
func deserialize(data []byte) (*KeyWallet, error) {
	var key = &KeyWallet{}
	if len(data) < 2 {
		return nil, NewWalletError(InvalidKeyTypeErr, nil)
	}
	keyType := data[0]
	if keyType == PriKeyType {
		if len(data) != privKeySerializedBytesLen {
			return nil, NewWalletError(InvalidSeserializedKey, nil)
		}

		key.Depth = data[1]
		key.ChildNumber = data[2:6]
		key.ChainCode = data[6:38]
		keyLength := int(data[38])
		key.KeySet.PrivateKey = make([]byte, keyLength)
		copy(key.KeySet.PrivateKey[:], data[39:39+keyLength])
		err := key.KeySet.InitFromPrivateKey(&key.KeySet.PrivateKey)
		if err != nil {
			return nil, err
		}
	} else if keyType == PaymentAddressType {
		if !bytes.Equal(burnAddress1BytesDecode, data){
			if len(data) != paymentAddrSerializedBytesLen && len(data) != paymentAddrSerializedBytesLen + 1 +operation.Ed25519KeySize{
				return nil, NewWalletError(InvalidSeserializedKey, errors.New("length ota public key not valid: " + string(len(data))))
			}
		}
		apkKeyLength := int(data[1])
		pkencKeyLength := int(data[apkKeyLength+2])
		key.KeySet.PaymentAddress.Pk = make([]byte, apkKeyLength)
		key.KeySet.PaymentAddress.Tk = make([]byte, pkencKeyLength)
		copy(key.KeySet.PaymentAddress.Pk[:], data[2:2+apkKeyLength])
		copy(key.KeySet.PaymentAddress.Tk[:], data[3+apkKeyLength:3+apkKeyLength+pkencKeyLength])
		//Deserialize OTAPublic Key
		if len(data) > paymentAddrSerializedBytesLen{
			otapkLength := int(data[apkKeyLength+pkencKeyLength+3])
			if otapkLength != operation.Ed25519KeySize {
				return nil, NewWalletError(InvalidSeserializedKey, errors.New("length ota public key not valid: " + string(otapkLength)))
			}
			key.KeySet.PaymentAddress.OTAPublic = append([]byte{}, data[apkKeyLength+pkencKeyLength+4:apkKeyLength+pkencKeyLength+otapkLength+4]...)
		}

	} else if keyType == ReadonlyKeyType {
		if len(data) != readOnlyKeySerializedBytesLen {
			return nil, NewWalletError(InvalidSeserializedKey, nil)
		}

		apkKeyLength := int(data[1])
		if len(data) < apkKeyLength+3 {
			return nil, NewWalletError(InvalidKeyTypeErr, nil)
		}
		skencKeyLength := int(data[apkKeyLength+2])
		key.KeySet.ReadonlyKey.Pk = make([]byte, apkKeyLength)
		key.KeySet.ReadonlyKey.Rk = make([]byte, skencKeyLength)
		copy(key.KeySet.ReadonlyKey.Pk[:], data[2:2+apkKeyLength])
		copy(key.KeySet.ReadonlyKey.Rk[:], data[3+apkKeyLength:3+apkKeyLength+skencKeyLength])
	}else if keyType == OTAKeyType {
		if len(data) != otaKeySerializedBytesLen{
			return nil, NewWalletError(InvalidSeserializedKey, nil)
		}

		pkKeyLength := int(data[1])
		if len(data) < pkKeyLength+3 {
			return nil, NewWalletError(InvalidKeyTypeErr, nil)
		}
		skKeyLength := int(data[pkKeyLength+2])

		key.KeySet.OTAKey.SetPublicSpend(data[2:2+pkKeyLength])
		key.KeySet.OTAKey.SetOTASecretKey(data[3+pkKeyLength :3+pkKeyLength+skKeyLength])
	} else{
		return nil, NewWalletError(InvalidKeyTypeErr, errors.New("cannot detect key type"))
	}

	// validate checksum
	cs1 := base58.ChecksumFirst4Bytes(data[0 : len(data)-4])
	cs2 := data[len(data)-4:]
	for i := range cs1 {
		if cs1[i] != cs2[i] {
			return nil, NewWalletError(InvalidChecksumErr, nil)
		}
	}
	return key, nil
}

// Base58CheckDeserialize deserializes a KeySet encoded in base58 encoding
// because data contains keyType and serialized data of corresponding key
// it returns KeySet just contain corresponding key
func Base58CheckDeserialize(data string) (*KeyWallet, error) {
	b, _, err := base58.Base58Check{}.Decode(data)
	if err != nil {
		return nil, err
	}
	return deserialize(b)
}
