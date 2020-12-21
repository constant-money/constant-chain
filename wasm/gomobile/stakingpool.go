package gomobile

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"golang.org/x/crypto/sha3"
	"strconv"
)

const (
	BigIntSize        = 32 // bytes
	HashSize          = 32 // bytes
)

var sPK []byte

func handleError(msg string) error {
	println(msg)
	return errors.New(msg)
}

type Hash [HashSize]byte

func hashH(b []byte) Hash {
	return Hash(sha3.Sum256(b))
}

// String returns the Hash as the hexadecimal string of the byte-reversed hash.
func (hashObj Hash) String() string {
	for i := 0; i < HashSize/2; i++ {
		hashObj[i], hashObj[HashSize-1-i] = hashObj[HashSize-1-i], hashObj[i]
	}
	return hex.EncodeToString(hashObj[:])
}

func GetSignPublicKey(args string) (string, error) {
	// parse meta data
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	println("paramMaps:", paramMaps)

	data, ok := paramMaps["data"].(map[string]interface{})
	if !ok {
		return "", handleError("Invalid data param")
	}

	privateKey, ok := data["privateKey"].(string)
	if !ok {
		return "", handleError("Invalid private key")
	}

	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		return "", handleError("Invalid private key")
	}
	senderSK := keyWallet.KeySet.PrivateKey

	sk := new(privacy.Scalar).FromBytesS(senderSK[:BigIntSize])
	r := new(privacy.Scalar).FromBytesS(senderSK[BigIntSize:])
	sigKey := new(privacy.SchnorrPrivateKey)
	sigKey.Set(sk, r)

	sigPubKey := sigKey.GetPublicKey().GetPublicKey().ToBytesS()
	sigPubKeyStringEncode := hex.EncodeToString(sigPubKey)

	return sigPubKeyStringEncode, nil
}

func SignPoolWithdraw(args string) (string, error) {
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	println("paramMaps:", paramMaps)

	data, ok := paramMaps["data"].(map[string]interface{})
	if !ok {
		return "", handleError("Invalid data param")
	}

	privateKey, ok := data["privateKey"].(string)
	if !ok {
		return "", handleError("Invalid private key")
	}

	paymentAddress, ok := data["paymentAddress"].(string)
	if !ok {
		return "", handleError("Invalid payment address")
	}

	amount, ok := data["amount"].(string)
	if !ok {
		return "", handleError("Invalid amount")
	}

	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		return "", handleError("Invalid private key")
	}
	senderSK := keyWallet.KeySet.PrivateKey

	sk := new(privacy.Scalar).FromBytesS(senderSK[:BigIntSize])
	r := new(privacy.Scalar).FromBytesS(senderSK[BigIntSize:])
	sigKey := new(privacy.SchnorrPrivateKey)
	sigKey.Set(sk, r)

	if err != nil {
		return "", handleError("Get sigPublicKey error")
	}

	message := paymentAddress
	uintAmount, _ := strconv.ParseUint(amount, 10, 64)
	message += strconv.FormatUint(uintAmount, 10)

	inBytes := []byte(message)
	hash := hashH(inBytes)

	signature, err := sigKey.Sign(hash[:])
	if err != nil {
		println(err)
		return "", handleError("Sign error")
	}

	return hex.EncodeToString(signature.Bytes()), nil
}

func VerifySign(signEncode string, signPublicKeyEncode string, amount string, paymentAddress string) (bool, error) {
	signPublicKey, err := hex.DecodeString(signPublicKeyEncode)

	if err != nil {
		return false, handleError("Can not decode sign public key")
	}

	verifyKey := new(privacy.SchnorrPublicKey)
	sigPublicKey, err := new(privacy.Point).FromBytesS(signPublicKey)

	if err != nil {
		return false, handleError("Get sigPublicKey error")
	}
	verifyKey.Set(sigPublicKey)

	sign, err := hex.DecodeString(signEncode)
	signature := new(privacy.SchnSignature)
	err = signature.SetBytes(sign)
	if err != nil {
		return false, handleError("Sig set bytes error")
	}

	message := paymentAddress
	uintAmount, _ := strconv.ParseUint(amount, 10, 64)
	message += strconv.FormatUint(uintAmount, 10)

	inBytes := []byte(message)
	hash := hashH(inBytes)

	res := verifyKey.Verify(signature, hash[:])

	return res, nil
}

//func main() {
//	data := "{\"data\":{\"privateKey\":\"112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43\",\"paymentAddress\":\"12RpXjtmvPYyGkZ5Cy2fBQPsNzE2RLe1BJsuNbx8y3x49Uo7xnAShWYLovXJ7yb86xevrsDAAhtXThg3GqZ1n1ri5y9J2qamyFhs2By\",\"amount\":\"1000000000\"}}"
//	result, err := SignPoolWithdraw(data)
//	println("Result")
//	println(result)
//	println("Error")
//	println(err)
//	//sPK := []byte{3,184,101,162,74,169,116,19,130,110,152,65,168,94,128,156,23,139,86,107,36,54,197,44,50,56,41,18,157,224,6,135}
//	//sk, _ := hex.DecodeString("a5997dc882406cb0bed6213a33c7b058b985d208c195666ac019e3919f9a640b13998cb510a5c1459933546480ed69b9260f4f067d9cfd17f332edb487e97b0d")
//	//println(printlnskString)
//	//VerifySign(string(sk), string(sPK))
//}