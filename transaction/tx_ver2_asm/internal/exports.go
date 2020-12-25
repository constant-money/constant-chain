package internal

import (
	"syscall/js"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/hybridencryption"
	"github.com/incognitochain/incognito-chain/wallet"

	// "github.com/incognitochain/incognito-chain/metadata"
	"github.com/pkg/errors"
	// "math/big"
)

func CreateTransaction(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	var theirTime int64 = 0
	if len(jsInputs)>=2 && jsInputs[1].Type()==js.TypeNumber{
		theirTime = int64(jsInputs[1].Int())
	}

	params := &InitParamsAsm{}
	println("Before parse - TX parameters")
	println(args)
	err := json.Unmarshal([]byte(args), params)
	if err!=nil{
		println(err.Error())
		return "", err
	}
	println("After parse - TX parameters")
	thoseBytesAgain, _ := json.Marshal(params)
	println(string(thoseBytesAgain))

	var txJson []byte
	if params.TokenParams==nil{			
		tx := &Tx{}
		err = tx.InitASM(params, theirTime)

		if err != nil {
			println("Can not create tx: ", err.Error())
			return "", err
		}

		// serialize tx json
		txJson, err = json.Marshal(tx)
		if err != nil {
			println("Can not marshal tx: ", err)
			return "", err
		}
	}else{
		tx := &TxToken{}
		err = tx.InitASM(params, theirTime)

		if err != nil {
			println("Can not create tx: ", err.Error())
			return "", err
		}

		// serialize tx json
		txJson, err = json.Marshal(tx)
		if err != nil {
			println("Error marshalling tx: ", err)
			return "", err
		}
	}
	res := b58.Encode(txJson, common.ZeroByte)

	return res, nil
}

func CreateConvertTx(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	var theirTime int64 = 0
	if len(jsInputs)>=2 && jsInputs[1].Type()==js.TypeNumber{
		theirTime = int64(jsInputs[1].Int())
	}

	params := &InitParamsAsm{}
	// DEBUG
	println("Before parse - TX parameters")
	println(args)
	err := json.Unmarshal([]byte(args), params)
	if err!=nil{
		println(err.Error())
		return "", err
	}
	println("After parse - TX parameters")
	thoseBytesAgain, _ := json.Marshal(params)
	println(string(thoseBytesAgain))
	// END DEBUG

	var txJson []byte
	if params.TokenParams==nil{			
		tx := &Tx{}
		err = InitConversionASM(tx, params, theirTime)

		if err != nil {
			println("Can not create tx: ", err.Error())
			return "", err
		}

		// serialize tx json
		txJson, err = json.Marshal(tx)
		if err != nil {
			println("Can not marshal tx: ", err)
			return "", err
		}
	}else{
		tx := &TxToken{}
		err = InitTokenConversionASM(tx, params, theirTime)

		if err != nil {
			println("Can not create tx: ", err.Error())
			return "", err
		}

		// serialize tx json
		txJson, err = json.Marshal(tx)
		if err != nil {
			println("Error marshalling tx: ", err)
			return "", err
		}
	}
	res := b58.Encode(txJson, common.ZeroByte)

	return res, nil
}

func NewKeySetFromPrivate(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	skStr := jsInputs[0].String()

	var err error
	skHolder := struct{
		PrivateKey []byte `json:"PrivateKey"`
	}{}
	err = json.Unmarshal([]byte(skStr), &skHolder)
	if err!=nil{
		println(err.Error())
		return "", err
	}
	ks := &incognitokey.KeySet{}
	err = ks.InitFromPrivateKeyByte(skHolder.PrivateKey)
	if err!=nil{
		println(err.Error())
		return "", err
	}
	txJson, err := json.Marshal(ks)
	if err != nil {
		println("Error marshalling ket set: ", err)
		return "", err
	}

	return string(txJson), nil
}

func DecryptCoin(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	paramStr := jsInputs[0].String()

	var err error
	temp := &struct{
		Coin 	CoinInter
		KeySet 	string
	}{}
	err = json.Unmarshal([]byte(paramStr), temp)
	if err!=nil{
		return "", err
	}
	tempKw, err := wallet.Base58CheckDeserialize(temp.KeySet)
	if err!=nil{
		return "", err
	}
	ks := tempKw.KeySet
	var res CoinInter
	if temp.Coin.Version==2{
		c, _, err := temp.Coin.ToCoin()
		if err!=nil{
			return "", err
		}
		
		_, err = c.Decrypt(&ks)
		if err!=nil{
			println(err.Error())
			return "", err
		}
		res = GetCoinInter(c)
	}else if temp.Coin.Version==1{
		c, _, err := temp.Coin.ToCoinV1()
		if err!=nil{
			return "", err
		}
		
		pc, err := c.Decrypt(&ks)
		if err!=nil{
			println(err.Error())
			return "", err
		}
		res = GetCoinInter(pc)
	}
	
	res.Index = temp.Coin.Index
	resJson, err := json.Marshal(res)
	if err != nil {
		println("Error marshalling ket set: ", err)
		return "", err
	}
	return string(resJson), nil
}

func CreateCoin(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	paramStr := jsInputs[0].String()

	var err error
	temp := &struct{
		PaymentInfo 	printedPaymentInfo
		TokenID			string
	}{}
	err = json.Unmarshal([]byte(paramStr), temp)
	if err!=nil{
		return "", err
	}
	pInf, err := temp.PaymentInfo.To()
	if err!=nil{
		return "", err
	}
	var c *privacy.CoinV2
	if len(temp.TokenID)==0{
		c, err = privacy.NewCoinFromPaymentInfo(pInf)
		if err!=nil{
			println(err.Error())
			return "", err
		}
	}else{
		var tokenID common.Hash
		tokenID, _ = getTokenIDFromString(temp.TokenID)
		c, _, err = privacy.NewCoinCA(pInf, &tokenID)
		if err!=nil{
			println(err.Error())
			return "", err
		}
	}
	
	res := GetCoinInter(c)
	resJson, err := json.Marshal(res)
	if err != nil {
		println("Error marshalling ket set: ", err)
		return "", err
	}
	return string(resJson), nil
}

func GenerateBLSKeyPairFromSeed(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	seed, err := b64.DecodeString(args)
	if err != nil {
		return "", err
	}
	privateKey, publicKey := blsmultisig.KeyGen(seed)
	keyPairBytes := []byte{}
	keyPairBytes = append(keyPairBytes, common.AddPaddingBigInt(privateKey, common.BigIntSize)...)
	keyPairBytes = append(keyPairBytes, blsmultisig.CmprG2(publicKey)...)
	keyPairEncode := b64.EncodeToString(keyPairBytes)
	return keyPairEncode, nil
}

func GenerateKeyFromSeed(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	seed, err := b64.DecodeString(args)
	if err != nil {
		return "", err
	}
	key := privacy.GeneratePrivateKey(seed)
	res := b64.EncodeToString(key)
	return res, nil
}

func HybridEncrypt(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	
	raw, _ := b64.DecodeString(args)
	publicKeyBytes := raw[0:privacy.Ed25519KeySize]
	publicKeyPoint, err := new(privacy.Point).FromBytesS(publicKeyBytes)
	if err != nil {
		return "", errors.Errorf("Invalid public key encryption")
	}

	msgBytes := raw[privacy.Ed25519KeySize:]
	ciphertext, err := hybridencryption.HybridEncrypt(msgBytes, publicKeyPoint)
	if err != nil{
		return "", err
	}
	return b64.EncodeToString(ciphertext.Bytes()), nil
}

func HybridDecrypt(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	
	raw, _ := b64.DecodeString(args)
	privateKeyBytes := raw[0:privacy.Ed25519KeySize]
	privateKeyScalar := new(privacy.Scalar).FromBytesS(privateKeyBytes)

	ciphertextBytes := raw[privacy.Ed25519KeySize:]
	ciphertext := new(hybridencryption.HybridCipherText)
	ciphertext.SetBytes(ciphertextBytes)

	plaintextBytes, err := hybridencryption.HybridDecrypt(ciphertext, privateKeyScalar)
	if err != nil{
		return "", err
	}
	return b64.EncodeToString(plaintextBytes), nil
}

func ScalarMultBase(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	scalar, err := b64.DecodeString(args)
	if err != nil {
		return "", err
	}

	point := new(privacy.Point).ScalarMultBase(new(privacy.Scalar).FromBytesS(scalar))
	res := b64.EncodeToString(point.ToBytesS())
	return res, nil
}

func RandomScalars(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	num, err := strconv.ParseUint(args, 10, 64)
	if err != nil {
		return "", nil
	}

	var scalars []byte
	for i := 0; i < int(num); i++ {
		scalars = append(scalars, privacy.RandomScalar().ToBytesS()...)
	}

	res := b64.EncodeToString(scalars)
	return res, nil
}

func GetSignPublicKey(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	raw := []byte(args)
	var holder struct{
		Data struct{
			Sk string `json:"privateKey"`
		} `json:"data"`
	}

	err := json.Unmarshal(raw, &holder)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}
	privateKey := holder.Data.Sk
	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		return "", errors.Errorf("Invalid private key")
	}
	senderSK := keyWallet.KeySet.PrivateKey
	sk := new(privacy.Scalar).FromBytesS(senderSK[:HashSize])
	r := new(privacy.Scalar).FromBytesS(senderSK[HashSize:])
	sigKey := new(privacy.SchnorrPrivateKey)
	sigKey.Set(sk, r)
	sigPubKey := sigKey.GetPublicKey().GetPublicKey().ToBytesS()

	return hex.EncodeToString(sigPubKey), nil
}

func SignPoolWithdraw(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<1{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 1)
	}
	args := jsInputs[0].String()
	raw := []byte(args)
	var holder struct{
		Data struct{
			Sk string `json:"privateKey"`
			Amount string `json:"amount"`
			PaymentAddress string `json:"paymentAddress"`
		} `json:"data"`
	}

	err := json.Unmarshal(raw, &holder)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}
	privateKey := holder.Data.Sk
	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		return "", errors.Errorf("Invalid private key")
	}
	senderSK := keyWallet.KeySet.PrivateKey
	sk := new(privacy.Scalar).FromBytesS(senderSK[:HashSize])
	r := new(privacy.Scalar).FromBytesS(senderSK[HashSize:])
	sigKey := new(privacy.SchnorrPrivateKey)
	sigKey.Set(sk, r)

	message := holder.Data.PaymentAddress + holder.Data.Amount
	hashed := common.HashH([]byte(message))
	signature, err := sigKey.Sign(hashed[:])
	if err != nil {
		println(err.Error())
		return "", errors.Errorf("Sign error")
	}

	return hex.EncodeToString(signature.Bytes()), nil
}

// signEncode string, signPublicKeyEncode string, amount string, paymentAddress string
func VerifySign(_ js.Value, jsInputs []js.Value) (interface{}, error){
	if len(jsInputs)<4{
		return nil, errors.Errorf("Invalid number of parameters. Expected %d", 4)
	}
	temp, err := hex.DecodeString(jsInputs[1].String())
	if err != nil {
		return "", errors.Errorf("Can not decode sign public key")
	}
	sigPublicKey, err := new(privacy.Point).FromBytesS(temp)
	if err != nil {
		return "", errors.Errorf("Get sigPublicKey error")
	}
	verifyKey := new(privacy.SchnorrPublicKey)
	verifyKey.Set(sigPublicKey)

	temp, err = hex.DecodeString(jsInputs[0].String())
	signature := new(privacy.SchnSignature)
	err = signature.SetBytes(temp)
	if err != nil {
		return false, errors.Errorf("Sig set bytes error")
	}
	message := jsInputs[3].String() + jsInputs[2].String()
	hashed := common.HashH([]byte(message))
	res := verifyKey.Verify(signature, hashed[:])

	return res, nil
}

