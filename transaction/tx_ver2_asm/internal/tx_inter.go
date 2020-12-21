package internal

import (
	"syscall/js"
	"encoding/json"

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
		err = tx.InitASM(params)

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
		err = tx.InitASM(params)

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
		err = InitConversionASM(tx, params)

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
		err = InitTokenConversionASM(tx, params)

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
	
	seed, _ := b64.DecodeString(args)
	privateKey, publicKey := blsmultisig.KeyGen(seed)
	keyPairBytes := []byte{}
	keyPairBytes = append(keyPairBytes, common.AddPaddingBigInt(privateKey, common.BigIntSize)...)
	keyPairBytes = append(keyPairBytes, blsmultisig.CmprG2(publicKey)...)
	keyPairEncode := b64.EncodeToString(keyPairBytes)
	return keyPairEncode, nil
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

