package gomobile

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
)

func InitParamCreatePrivacyTx(args string) (*transaction.TxPrivacyInitParamsForASM, error) {
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return nil, err
	}

	println("paramMaps:", paramMaps)

	// sender's private key
	senderSKParam, ok := paramMaps["senderSK"].(string)
	if !ok {
		println("Invalid sender private key!")
		return nil, errors.New("Invalid sender private key")
	}
	println("senderSKParam: %v\n", senderSKParam)

	keyWallet, err := wallet.Base58CheckDeserialize(senderSKParam)
	if err != nil {
		println("Error can not decode sender private key : %v\n", err)
		return nil, err
	}
	senderSK := keyWallet.KeySet.PrivateKey
	println("senderSK: ", senderSK)

	//get payment infos
	println(paramMaps["paramPaymentInfos"])
	paymentInfoParams, ok := paramMaps["paramPaymentInfos"].([]interface{})
	if !ok {
		println("Invalid payment info params!")
		return nil, errors.New("Invalid payment info params")
	}

	paymentInfo := make([]*privacy.PaymentInfo, 0)
	for i := 0; i < len(paymentInfoParams); i++ {
		tmp, ok := paymentInfoParams[i].(map[string]interface{})
		if !ok {
			println("Invalid payment info param!")
			return nil, errors.New("Invalid payment info param")
		}
		paymentAddrStr, ok := tmp["paymentAddressStr"].(string)
		if !ok {
			println("Invalid payment info param payment address string")
			return nil, errors.New("Invalid payment info param payment address string")
		}

		amount, err := common.AssertAndConvertStrToNumber(tmp["amount"])
		if err != nil {
			println("Invalid payment info param amount")
			return nil, errors.New("Invalid payment info param amount")
		}

		msgBytes := []byte{}
		if tmp["message"] != nil {
			msgB64Encode, ok := tmp["message"].(string)
			if !ok {
				println("Invalid payment info param amount")
				return nil, errors.New("Invalid payment info param amount")
			}

			if msgB64Encode != "" {
				msgBytes, err = base64.StdEncoding.DecodeString(msgB64Encode)
				if err != nil {
					println("Can not decode msg string in payment info for ptoken")
					return nil, errors.New("Can not decode msg string in payment info for ptoken")
				}
			}
		}

		paymentInfoTmp := new(privacy.PaymentInfo)
		keyWallet, err := wallet.Base58CheckDeserialize(paymentAddrStr)
		if err != nil {
			println("Error can not decode sender private key : %v\n", err)
			return nil, err
		}
		paymentInfoTmp.PaymentAddress = keyWallet.KeySet.PaymentAddress
		paymentInfoTmp.Amount = amount
		paymentInfoTmp.Message = msgBytes
		paymentInfo = append(paymentInfo, paymentInfoTmp)
	}

	//get fee
	fee, err := common.AssertAndConvertStrToNumber(paramMaps["fee"])
	if err != nil {
		println("Invalid fee param")
		return nil, errors.New("Invalid fee param")
	}
	println("fee: ", fee)

	// get has Privacy
	hasPrivacy, ok := paramMaps["isPrivacy"].(bool)
	if !ok {
		println("Invalid is privacy param!")
		return nil, errors.New("Invalid is privacy param")
	}
	println("hasPrivacy: ", hasPrivacy)

	// get has Privacy
	info, ok := paramMaps["info"].(string)
	if !ok {
		println("Invalid info param!")
		return nil, errors.New("Invalid info param")
	}
	infoBytes := []byte(info)
	println("infoBytes: ", infoBytes)

	inputCoinStrs, ok := paramMaps["inputCoinStrs"].([]interface{})
	if !ok {
		println("Invalid input coin string params!")
		return nil, errors.New("Invalid input coin string params")
	}
	println("inputCoinStrs: ", inputCoinStrs)

	inputCoins := make([]*coin.PlainCoinV1, len(inputCoinStrs))
	for i := 0; i < len(inputCoins); i++ {
		tmp, ok := inputCoinStrs[i].(map[string]interface{})
		if !ok {
			println("Invalid input coin string param!")
			return nil, errors.New("Invalid input coin string param")
		}
		coinObjTmp := new(privacy.CoinObject)
		coinObjTmp.PublicKey, ok = tmp["PublicKey"].(string)
		if !ok {
			println("Invalid input coin public key param!")
			return nil, errors.New("Invalid input coin public key param")
		}
		coinObjTmp.CoinCommitment, ok = tmp["CoinCommitment"].(string)
		if !ok {
			println("Invalid input coin coin commitment param!")
			return nil, errors.New("Invalid input coin coin commitment param")
		}
		coinObjTmp.SNDerivator, ok = tmp["SNDerivator"].(string)
		if !ok {
			println("Invalid input coin snderivator param!")
			return nil, errors.New("Invalid input coin snderivator param")
		}
		coinObjTmp.SerialNumber, ok = tmp["SerialNumber"].(string)
		if !ok {
			println("Invalid input coin serial number param!")
			return nil, errors.New("Invalid input coin serial number param")
		}
		coinObjTmp.Randomness, ok = tmp["Randomness"].(string)
		if !ok {
			println("Invalid input coin randomness param!")
			return nil, errors.New("Invalid input coin randomness param")
		}
		coinObjTmp.Value, ok = tmp["Value"].(string)
		if !ok {
			println("Invalid input coin value param!")
			return nil, errors.New("Invalid input coin value param")
		}
		coinObjTmp.Info, ok = tmp["Info"].(string)
		if !ok {
			println("Invalid input coin info param!")
			return nil, errors.New("Invalid input coin info param")
		}

		inputCoins[i] = new(coin.PlainCoinV1).Init()
		inputCoins[i].ParseCoinObjectToInputCoin(*coinObjTmp)
	}

	println("inputCoins: ", inputCoins)

	commitmentIndicesParam, ok := paramMaps["commitmentIndices"].([]interface{})
	if !ok {
		return nil, errors.New("invalid commitment indices param")
	}
	commitmentStrsParam, ok := paramMaps["commitmentStrs"].([]interface{})
	if !ok {
		return nil, errors.New("invalid commitment strings param")
	}

	myCommitmentIndicesParam, ok := paramMaps["myCommitmentIndices"].([]interface{})
	if !ok {
		return nil, errors.New("invalid my commitment indices param")
	}

	sndOutputsParam, ok := paramMaps["sndOutputs"].([]interface{})
	if !ok {
		return nil, errors.New("invalid snd outputs param")
	}

	println("sndOutputsParam: ", sndOutputsParam)

	commitmentIndices := make([]uint64, len(commitmentIndicesParam))
	commitmentStrs := make([]string, len(commitmentStrsParam))
	myCommitmentIndices := make([]uint64, len(myCommitmentIndicesParam))
	sndOutputs := make([]*privacy.Scalar, len(sndOutputsParam))

	commitmentBytes := make([][]byte, len(commitmentStrsParam))
	for i := 0; i < len(commitmentIndices); i++ {
		tmp, ok := commitmentIndicesParam[i].(float64)
		if !ok {
			return nil, errors.New("invalid commitment indices param")
		}
		commitmentIndices[i] = uint64(tmp)
		commitmentStrs[i], ok = commitmentStrsParam[i].(string)
		if !ok {
			return nil, errors.New("invalid commitment string param")
		}

		commitmentBytes[i], _, err = base58.Base58Check{}.Decode(commitmentStrs[i])
		if err != nil {
			return nil, nil
		}
	}

	for i := 0; i < len(myCommitmentIndices); i++ {
		tmp, ok := myCommitmentIndicesParam[i].(float64)
		if !ok {
			return nil, errors.New("invalid my commitment index param")
		}
		myCommitmentIndices[i] = uint64(tmp)
	}

	for i := 0; i < len(sndOutputs); i++ {
		println("sndOutputsParam[i].(string): ", sndOutputsParam[i].(string))
		tmp, _, err := base58.Base58Check{}.Decode(sndOutputsParam[i].(string))
		if err != nil {
			return nil, err
		}

		sndOutputs[i] = new(privacy.Scalar).FromBytesS(tmp)
	}

	paramCreateTx := transaction.NewTxPrivacyInitParamsForASM(&senderSK, paymentInfo, inputCoins, uint64(fee), hasPrivacy, nil, nil, infoBytes, commitmentIndices, commitmentBytes, myCommitmentIndices, sndOutputs)
	println("paramCreateTx: ", paramCreateTx)

	return paramCreateTx, nil
}

func InitParamCreatePrivacyTokenTx(args string) (*transaction.TxPrivacyTokenInitParamsForASM, error) {
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return nil, err
	}

	println("paramMaps:", paramMaps)

	// sender's private key
	senderSKParam, ok := paramMaps["senderSK"].(string)
	if !ok {
		println("Invalid sender private key!")
		return nil, errors.New("Invalid sender private key")
	}
	println("senderSKParam: %v\n", senderSKParam)

	keyWallet, err := wallet.Base58CheckDeserialize(senderSKParam)
	if err != nil {
		println("Error can not decode sender private key : %v\n", err)
		return nil, err
	}
	senderSK := keyWallet.KeySet.PrivateKey
	println("senderSK: ", senderSK)

	keyWallet.KeySet.InitFromPrivateKeyByte(keyWallet.KeySet.PrivateKey)
	publicKey := keyWallet.KeySet.PaymentAddress.Pk
	shardID := common.GetShardIDFromLastByte(publicKey[len(publicKey)-1])
	println("shardID: ", shardID)

	//get payment infos
	println(paramMaps["paramPaymentInfos"])
	paymentInfoParams, ok := paramMaps["paramPaymentInfos"].([]interface{})
	if !ok {
		println("Invalid payment info params!")
		return nil, errors.New("Invalid payment info params")
	}

	paymentInfo := make([]*privacy.PaymentInfo, 0)
	for i := 0; i < len(paymentInfoParams); i++ {
		tmp, ok := paymentInfoParams[i].(map[string]interface{})
		if !ok {
			println("Invalid payment info param!")
			return nil, errors.New("Invalid payment info param")
		}
		paymentAddrStr, ok := tmp["paymentAddressStr"].(string)
		if !ok {
			println("Invalid payment info param payment address string")
			return nil, errors.New("Invalid payment info param payment address string")
		}

		amount, err := common.AssertAndConvertStrToNumber(tmp["amount"])
		if err != nil {
			println("Invalid payment info param amount")
			return nil, errors.New("Invalid payment info param amount")
		}

		msgBytes := []byte{}
		if tmp["message"] != nil {
			msgB64Encode, ok := tmp["message"].(string)
			if !ok {
				println("Invalid payment info param amount")
				return nil, errors.New("Invalid payment info param amount")
			}

			if msgB64Encode != "" {
				msgBytes, err = base64.StdEncoding.DecodeString(msgB64Encode)
				if err != nil {
					println("Can not decode msg string in payment info for ptoken")
					return nil, errors.New("Can not decode msg string in payment info for ptoken")
				}
			}
		}

		paymentInfoTmp := new(privacy.PaymentInfo)
		keyWallet, err := wallet.Base58CheckDeserialize(paymentAddrStr)
		if err != nil {
			println("Error can not decode sender private key : %v\n", err)
			return nil, err
		}
		paymentInfoTmp.PaymentAddress = keyWallet.KeySet.PaymentAddress
		paymentInfoTmp.Amount = amount
		paymentInfoTmp.Message = msgBytes
		paymentInfo = append(paymentInfo, paymentInfoTmp)
	}

	//get fee
	fee, err := common.AssertAndConvertStrToNumber(paramMaps["fee"])
	if err != nil {
		println("Invalid fee param")
		return nil, errors.New("Invalid fee param")
	}
	println("fee: ", fee)

	// get has Privacy
	hasPrivacy, ok := paramMaps["isPrivacy"].(bool)
	if !ok {
		println("Invalid is privacy param!")
		return nil, errors.New("Invalid is privacy param")
	}
	println("hasPrivacy: ", hasPrivacy)

	// get has Privacy for ptoken
	hasPrivacyForPToken, ok := paramMaps["isPrivacyForPToken"].(bool)
	if !ok {
		println("Invalid is privacy for ptoken param!")
		return nil, errors.New("Invalid is privacy for ptoken param")
	}
	println("hasPrivacyForPToken: ", hasPrivacyForPToken)

	// get info
	info, ok := paramMaps["info"].(string)
	if !ok {
		println("Invalid info param!")
		return nil, errors.New("Invalid info param")
	}
	infoBytes := []byte(info)
	println("infoBytes: ", infoBytes)

	inputCoinStrs, ok := paramMaps["inputCoinStrs"].([]interface{})
	if !ok {
		println("Invalid input coin string params!")
		return nil, errors.New("Invalid input coin string params")
	}
	println("inputCoinStrs: ", inputCoinStrs)

	inputCoins := make([]*coin.PlainCoinV1, len(inputCoinStrs))
	for i := 0; i < len(inputCoins); i++ {
		tmp, ok := inputCoinStrs[i].(map[string]interface{})
		if !ok {
			println("Invalid input coin string param!")
			return nil, errors.New("Invalid input coin string param")
		}
		coinObjTmp := new(privacy.CoinObject)
		coinObjTmp.PublicKey, ok = tmp["PublicKey"].(string)
		if !ok {
			println("Invalid input coin public key param!")
			return nil, errors.New("Invalid input coin public key param")
		}
		coinObjTmp.CoinCommitment, ok = tmp["CoinCommitment"].(string)
		if !ok {
			println("Invalid input coin coin commitment param!")
			return nil, errors.New("Invalid input coin coin commitment param")
		}
		coinObjTmp.SNDerivator, ok = tmp["SNDerivator"].(string)
		if !ok {
			println("Invalid input coin snderivator param!")
			return nil, errors.New("Invalid input coin snderivator param")
		}
		coinObjTmp.SerialNumber, ok = tmp["SerialNumber"].(string)
		if !ok {
			println("Invalid input coin serial number param!")
			return nil, errors.New("Invalid input coin serial number param")
		}
		coinObjTmp.Randomness, ok = tmp["Randomness"].(string)
		if !ok {
			println("Invalid input coin randomness param!")
			return nil, errors.New("Invalid input coin randomness param")
		}
		coinObjTmp.Value, ok = tmp["Value"].(string)
		if !ok {
			println("Invalid input coin value param!")
			return nil, errors.New("Invalid input coin value param")
		}
		coinObjTmp.Info, ok = tmp["Info"].(string)
		if !ok {
			println("Invalid input coin info param!")
			return nil, errors.New("Invalid input coin info param")
		}

		inputCoins[i] = new(coin.PlainCoinV1).Init()
		inputCoins[i].ParseCoinObjectToInputCoin(*coinObjTmp)
	}

	println("inputCoins: ", inputCoins)

	// for native token
	commitmentIndicesParamForNativeToken, ok := paramMaps["commitmentIndicesForNativeToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid commitment indices param")
	}
	commitmentStrsParamForNativeToken, ok := paramMaps["commitmentStrsForNativeToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid commitment strings param")
	}

	myCommitmentIndicesParamForNativeToken, ok := paramMaps["myCommitmentIndicesForNativeToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid my commitment indices param")
	}

	sndOutputsParamForNativeToken, ok := paramMaps["sndOutputsForNativeToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid snd outputs param")
	}

	println("sndOutputsParamForNativeToken: ", sndOutputsParamForNativeToken)

	commitmentIndicesForNativeToken := make([]uint64, len(commitmentIndicesParamForNativeToken))
	commitmentStrsForNativeToken := make([]string, len(commitmentStrsParamForNativeToken))
	myCommitmentIndicesForNativeToken := make([]uint64, len(myCommitmentIndicesParamForNativeToken))
	sndOutputsForNativeToken := make([]*privacy.Scalar, len(sndOutputsParamForNativeToken))

	commitmentBytesForNativeToken := make([][]byte, len(commitmentStrsParamForNativeToken))
	for i := 0; i < len(commitmentIndicesForNativeToken); i++ {
		tmp, ok := commitmentIndicesParamForNativeToken[i].(float64)
		if !ok {
			return nil, errors.New("invalid commitment indices for native token param")
		}
		commitmentIndicesForNativeToken[i] = uint64(tmp)
		commitmentStrsForNativeToken[i], ok = commitmentStrsParamForNativeToken[i].(string)
		if !ok {
			return nil, errors.New("invalid commitment string for native token param")
		}

		commitmentBytesForNativeToken[i], _, err = base58.Base58Check{}.Decode(commitmentStrsForNativeToken[i])
		if err != nil {
			return nil, nil
		}
	}

	for i := 0; i < len(myCommitmentIndicesForNativeToken); i++ {
		tmp, ok := myCommitmentIndicesParamForNativeToken[i].(float64)
		if !ok {
			return nil, errors.New("invalid my commitment index for native token param")
		}
		myCommitmentIndicesForNativeToken[i] = uint64(tmp)
	}

	for i := 0; i < len(sndOutputsForNativeToken); i++ {

		println("sndOutputsParamForNativeToken[i].(string): ", sndOutputsParamForNativeToken[i].(string))
		tmp, _, err := base58.Base58Check{}.Decode(sndOutputsParamForNativeToken[i].(string))
		if err != nil {
			return nil, nil
		}

		sndOutputsForNativeToken[i] = new(privacy.Scalar).FromBytesS(tmp)
	}

	// for privacy token
	commitmentIndicesParamForPToken, ok := paramMaps["commitmentIndicesForPToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid commitment indices for ptoken param")
	}
	commitmentStrsParamForPToken, ok := paramMaps["commitmentStrsForPToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid commitment strings for ptoken param")
	}

	myCommitmentIndicesParamForPToken, ok := paramMaps["myCommitmentIndicesForPToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid my commitment indices for ptoken param")
	}

	sndOutputsParamForPToken, ok := paramMaps["sndOutputsForPToken"].([]interface{})
	if !ok {
		return nil, errors.New("invalid snd outputs for ptoken param")
	}

	println("sndOutputsParamForPToken: ", sndOutputsParamForPToken)

	commitmentIndicesForPToken := make([]uint64, len(commitmentIndicesParamForPToken))
	commitmentStrsForPToken := make([]string, len(commitmentStrsParamForPToken))
	myCommitmentIndicesForPToken := make([]uint64, len(myCommitmentIndicesParamForPToken))
	sndOutputsForPToken := make([]*privacy.Scalar, len(sndOutputsParamForPToken))

	commitmentBytesForPToken := make([][]byte, len(commitmentStrsParamForPToken))
	for i := 0; i < len(commitmentIndicesForPToken); i++ {
		tmp, ok := commitmentIndicesParamForPToken[i].(float64)
		if !ok {
			return nil, errors.New("invalid commitment indices for privacy token param")
		}
		commitmentIndicesForPToken[i] = uint64(tmp)
		commitmentStrsForPToken[i] = commitmentStrsParamForPToken[i].(string)

		commitmentBytesForPToken[i], _, err = base58.Base58Check{}.Decode(commitmentStrsForPToken[i])
		if err != nil {
			return nil, err
		}
	}

	println("commitmentBytesForPToken: ", commitmentBytesForPToken)
	println("commitmentIndicesForPToken: ", commitmentIndicesForPToken)

	for i := 0; i < len(myCommitmentIndicesForPToken); i++ {
		tmp, ok := myCommitmentIndicesParamForPToken[i].(float64)
		if !ok {
			return nil, errors.New("invalid commitment indices for privacy token param")
		}
		myCommitmentIndicesForPToken[i] = uint64(tmp)
	}
	println("myCommitmentIndicesForPToken: ", myCommitmentIndicesForPToken)

	for i := 0; i < len(sndOutputsForPToken); i++ {
		tmp, _, err := base58.Base58Check{}.Decode(sndOutputsParamForPToken[i].(string))
		if err != nil {
			return nil, err
		}

		sndOutputsForPToken[i] = new(privacy.Scalar).FromBytesS(tmp)
	}
	println("sndOutputsForPToken: ", sndOutputsForPToken)

	// get privacy token param
	privacyTokenParam := new(transaction.TokenParam)
	pTokenParam, ok := paramMaps["privacyTokenParam"].(map[string]interface{})
	if !ok {
		println("Invalid privacy token param")
		return nil, errors.New("Invalid privacy token param")
	}
	privacyTokenParam.PropertyID, ok = pTokenParam["propertyID"].(string)
	if !ok {
		println("Invalid token ID param")
		return nil, errors.New("Invalid token ID param")
	}
	privacyTokenParam.PropertyName, ok = pTokenParam["propertyName"].(string)
	if !ok {
		println("Invalid token name param")
		return nil, errors.New("Invalid token name param")
	}
	privacyTokenParam.PropertySymbol, ok = pTokenParam["propertySymbol"].(string)
	if !ok {
		println("Invalid token symbol param")
		return nil, errors.New("Invalid token symbol param")
	}

	tmpAmount, err := common.AssertAndConvertStrToNumber(pTokenParam["amount"])
	if err != nil {
		println("Invalid amount param")
		return nil, errors.New("Invalid amount param")
	}
	privacyTokenParam.Amount = tmpAmount
	tmpTokenTxType, ok := pTokenParam["tokenTxType"].(float64)
	if !ok {
		println("Invalid token tx type param")
		return nil, errors.New("Invalid token tx type param")
	}
	privacyTokenParam.TokenTxType = int(tmpTokenTxType)
	tmpFeePToken, err := common.AssertAndConvertStrToNumber(pTokenParam["fee"])
	if err != nil {
		println("Invalid fee token param")
		return nil, errors.New("Invalid fee token param")
	}
	privacyTokenParam.Fee = tmpFeePToken
	paymentInfoForPTokenParam, ok := pTokenParam["paymentInfoForPToken"].([]interface{})
	if !ok {
		println("Invalid payment info params!")
		return nil, errors.New("Invalid payment info params")
	}

	paymentInfoForPToken := make([]*privacy.PaymentInfo, 0)
	for i := 0; i < len(paymentInfoForPTokenParam); i++ {
		tmp, ok := paymentInfoForPTokenParam[i].(map[string]interface{})
		if !ok {
			println("Invalid payment info param!")
			return nil, errors.New("Invalid payment info param")
		}
		paymentAddrStr, ok := tmp["paymentAddressStr"].(string)
		if !ok {
			println("Invalid payment info for ptoken param payment address string")
			return nil, errors.New("Invalid payment info for ptoken param payment address string")
		}
		amount, err := common.AssertAndConvertStrToNumber(tmp["amount"])
		if err != nil {
			println("Invalid payment info for ptoken param amount")
			return nil, errors.New("Invalid payment info for ptoken param amount")
		}
		msgBytes := []byte{}
		if tmp["message"] != nil {
			msgB64Encode, ok := tmp["message"].(string)
			if !ok {
				println("Invalid payment info for ptoken param amount")
				return nil, errors.New("Invalid payment info for ptoken param amount")
			}

			if msgB64Encode != "" {
				msgBytes, err = base64.StdEncoding.DecodeString(msgB64Encode)
				if err != nil {
					println("Can not decode msg string in payment info for ptoken")
					return nil, errors.New("Can not decode msg string in payment info for ptoken")
				}
			}
		}

		paymentInfoTmp := new(privacy.PaymentInfo)
		keyWallet, err := wallet.Base58CheckDeserialize(paymentAddrStr)
		if err != nil {
			println("Error can not decode sender private key : %v\n", err)
			return nil, err
		}
		paymentInfoTmp.PaymentAddress = keyWallet.KeySet.PaymentAddress
		println("PK receiver token: ", paymentInfoTmp.PaymentAddress.Pk)
		paymentInfoTmp.Amount = amount
		paymentInfoTmp.Message = msgBytes
		paymentInfoForPToken = append(paymentInfoForPToken, paymentInfoTmp)
	}

	privacyTokenParam.Receiver = paymentInfoForPToken

	tokenInputsParam, ok := pTokenParam["tokenInputs"].([]interface{})
	if !ok {
		println("Invalid token input coin string params!")
		return nil, errors.New("Invalid token input coin string params")
	}
	println("tokenInputs: ", tokenInputsParam)

	tokenInputs := make([]*coin.PlainCoinV1, len(tokenInputsParam))
	for i := 0; i < len(tokenInputs); i++ {
		tmp, ok := tokenInputsParam[i].(map[string]interface{})
		if !ok {
			println("Invalid input coin string param!")
			return nil, errors.New("Invalid input coin string param")
		}
		coinObjTmp := new(privacy.CoinObject)
		coinObjTmp.PublicKey, ok = tmp["PublicKey"].(string)
		if !ok {
			println("Invalid input coin public key param!")
			return nil, errors.New("Invalid input coin public key param")
		}
		coinObjTmp.CoinCommitment, ok = tmp["CoinCommitment"].(string)
		if !ok {
			println("Invalid input coin coin commitment param!")
			return nil, errors.New("Invalid input coin coin commitment param")
		}
		coinObjTmp.SNDerivator, ok = tmp["SNDerivator"].(string)
		if !ok {
			println("Invalid input coin snderivator param!")
			return nil, errors.New("Invalid input coin snderivator param")
		}
		coinObjTmp.SerialNumber, ok = tmp["SerialNumber"].(string)
		if !ok {
			println("Invalid input coin serial number param!")
			return nil, errors.New("Invalid input coin serial number param")
		}
		coinObjTmp.Randomness, ok = tmp["Randomness"].(string)
		if !ok {
			println("Invalid input coin randomness param!")
			return nil, errors.New("Invalid input coin randomness param")
		}
		coinObjTmp.Value, ok = tmp["Value"].(string)
		if !ok {
			println("Invalid input coin value param!")
			return nil, errors.New("Invalid input coin value param")
		}
		coinObjTmp.Info, ok = tmp["Info"].(string)
		if !ok {
			println("Invalid input coin info param!")
			return nil, errors.New("Invalid input coin info param")
		}

		tokenInputs[i] = new(coin.PlainCoinV1).Init()
		tokenInputs[i].ParseCoinObjectToInputCoin(*coinObjTmp)
	}

	println("tokenInputs: ", tokenInputs)
	privacyTokenParam.TokenInput = tokenInputs

	println("privacyTokenParam: ", len(privacyTokenParam.Receiver))
	println("privacyTokenParam.PropertyName: ", privacyTokenParam.PropertyName)
	println("privacyTokenParam.PropertySymbol: ", privacyTokenParam.PropertySymbol)
	println("privacyTokenParam.TokenInput: ", len(privacyTokenParam.TokenInput))
	println("privacyTokenParam.TokenTxType: ", privacyTokenParam.TokenTxType)
	println("privacyTokenParam.Amount: ", privacyTokenParam.Amount)
	//println("privacyTokenParam.PropertySymbol: ", len(privacyTokenParam.PropertySymbol))

	paramCreateTx := transaction.NewTxPrivacyTokenInitParamsForASM(
		&senderSK, paymentInfo, inputCoins, fee, privacyTokenParam, nil, hasPrivacy, hasPrivacyForPToken, shardID, infoBytes,
		commitmentIndicesForNativeToken, commitmentBytesForNativeToken, myCommitmentIndicesForNativeToken, sndOutputsForNativeToken,
		commitmentIndicesForPToken, commitmentBytesForPToken, myCommitmentIndicesForPToken, sndOutputsForPToken)
	println("paramCreateTx: ", paramCreateTx)

	return paramCreateTx, nil
}
