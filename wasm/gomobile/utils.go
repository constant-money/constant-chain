package gomobile

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
	"strconv"
)

// GenerateBLSKeyPairFromSeed generates BLS key pair from seed
func GenerateBLSKeyPairFromSeed(args string) string {
	// convert seed from string to bytes array
	//fmt.Printf("args: %v\n", args)
	seed, _ := base64.StdEncoding.DecodeString(args)
	//fmt.Printf("bls seed: %v\n", seed)

	// generate  bls key
	privateKey, publicKey := blsmultisig.KeyGen(seed)

	// append key pair to one bytes array
	keyPairBytes := []byte{}
	keyPairBytes = append(keyPairBytes, common.AddPaddingBigInt(privateKey, common.BigIntSize)...)
	keyPairBytes = append(keyPairBytes, blsmultisig.CmprG2(publicKey)...)

	//  base64.StdEncoding.EncodeToString()
	keyPairEncode := base64.StdEncoding.EncodeToString(keyPairBytes)

	return keyPairEncode
}

// args: seed
func GenerateKeyFromSeed(seedB64Encoded string) (string, error) {
	seed, err := base64.StdEncoding.DecodeString(seedB64Encoded)
	if err != nil {
		return "", err
	}

	println("[Go] Seed: ", seed)

	key := privacy.GeneratePrivateKey(seed)
	println("[Go] key: ", key)

	res := base64.StdEncoding.EncodeToString(key)
	println("[Go] res: ", res)
	return res, nil
}

func ScalarMultBase(scalarB64Encode string) (string, error) {
	scalar, err := base64.StdEncoding.DecodeString(scalarB64Encode)
	if err != nil {
		return "", nil
	}

	point := new(privacy.Point).ScalarMultBase(new(privacy.Scalar).FromBytesS(scalar))
	res := base64.StdEncoding.EncodeToString(point.ToBytesS())
	return res, nil
}

func DeriveSerialNumber(args string) (string, error) {
	// parse data
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	privateKeyStr, ok := paramMaps["privateKey"].(string)
	if !ok {
		println("Invalid private key")
		return "", errors.New("Invalid private key")
	}

	keyWallet, err := wallet.Base58CheckDeserialize(privateKeyStr)
	if err != nil {
		println("Can not decode private key")
		return "", errors.New("Can not decode private key")
	}
	privateKeyScalar := new(privacy.Scalar).FromBytesS(keyWallet.KeySet.PrivateKey)

	snds, ok := paramMaps["snds"].([]interface{})
	if !ok {
		println("Invalid list of serial number derivator")
		return "", errors.New("Invalid list of serial number derivator")

	}
	sndScalars := make([]*privacy.Scalar, len(snds))

	for i := 0; i < len(snds); i++ {
		tmp, ok := snds[i].(string)
		println("tmp: ", tmp)
		if !ok {
			println("Invalid serial number derivator")
			return "", errors.New("Invalid serial number derivator")

		}
		sndBytes, _, err := base58.Base58Check{}.Decode(tmp)
		println("sndBytes: ", sndBytes)
		if err != nil {
			println("Can not decode serial number derivator")
			return "", errors.New("Can not decode serial number derivator")
		}
		sndScalars[i] = new(privacy.Scalar).FromBytesS(sndBytes)
	}

	// calculate serial number and return result

	serialNumberPoint := make([]*privacy.Point, len(sndScalars))
	serialNumberStr := make([]string, len(serialNumberPoint))

	serialNumberBytes := make([]byte, 0)

	for i := 0; i < len(sndScalars); i++ {
		serialNumberPoint[i] = new(privacy.Point).Derive(privacy.PedCom.G[privacy.PedersenPrivateKeyIndex], privateKeyScalar, sndScalars[i])
		println("serialNumberPoint[i]: ", serialNumberPoint[i])

		serialNumberStr[i] = base58.Base58Check{}.Encode(serialNumberPoint[i].ToBytesS(), 0x00)
		println("serialNumberStr[i]: ", serialNumberStr[i])
		serialNumberBytes = append(serialNumberBytes, serialNumberPoint[i].ToBytesS()...)
	}

	result := base64.StdEncoding.EncodeToString(serialNumberBytes)

	return result, nil
}

func RandomScalars(n string) (string, error) {
	nInt, err := strconv.ParseUint(n, 10, 64)
	println("nInt: ", nInt)
	if err != nil {
		return "", nil
	}

	scalars := make([]byte, 0)
	for i := 0; i < int(nInt); i++ {
		scalars = append(scalars, privacy.RandomScalar().ToBytesS()...)
	}

	res := base64.StdEncoding.EncodeToString(scalars)

	println("res scalars: ", res)

	return res, nil
}

// plaintextB64Encode = base64Encode(public key bytes || msg)
// returns base64Encode(ciphertextBytes)
func HybridEncryptionASM(dataB64Encode string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(dataB64Encode)
	if err != nil {
		return "", nil
	}

	publicKeyBytes := data[0:privacy.Ed25519KeySize]
	publicKeyPoint, err := new(privacy.Point).FromBytesS(publicKeyBytes)
	if err != nil {
		return "", errors.New("Invalid public key encryption")
	}

	msgBytes := data[privacy.Ed25519KeySize:]

	ciphertext, err := privacy.HybridEncrypt(msgBytes, publicKeyPoint)
	if err != nil {
		return "", err
	}
	res := base64.StdEncoding.EncodeToString(ciphertext.Bytes())
	return res, nil
}

// plaintextB64Encode = base64Encode(private key || ciphertext)
// returns base64Encode(plaintextBytes)
func HybridDecryptionASM(dataB64Encode string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(dataB64Encode)
	if err != nil {
		return "", nil
	}

	privateKeyBytes := data[0:privacy.Ed25519KeySize]
	privateKeyScalar := new(privacy.Scalar).FromBytesS(privateKeyBytes)

	ciphertextBytes := data[privacy.Ed25519KeySize:]
	ciphertext := new(privacy.HybridCipherText)
	ciphertext.SetBytes(ciphertextBytes)

	plaintextBytes, err := privacy.HybridDecrypt(ciphertext, privateKeyScalar)
	if err != nil {
		return "", err
	}
	res := base64.StdEncoding.EncodeToString(plaintextBytes)
	return res, nil
}
