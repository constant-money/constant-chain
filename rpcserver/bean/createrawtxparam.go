package bean

import (
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

type CreateRawTxParam struct {
	SenderKeySet         *incognitokey.KeySet
	ShardIDSender        byte
	PaymentInfos         []*privacy.PaymentInfo
	EstimateFeeCoinPerKb int64
	HasPrivacyCoin       bool
	Info                 []byte
}

func GetKeySetFromPrivateKeyParams(privateKeyWalletStr string) (*incognitokey.KeySet, byte, error) {
	// deserialize to crate keywallet object which contain private key
	keyWallet, err := wallet.Base58CheckDeserialize(privateKeyWalletStr)
	if err != nil {
		return nil, byte(0), err
	}

	// fill paymentaddress and readonly key with privatekey
	err = keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	if err != nil {
		return nil, byte(0), err
	}

	if len(keyWallet.KeySet.PaymentAddress.Pk) == 0 {
		return nil, byte(0), errors.New("private key is not valid")
	}

	// calculate shard ID
	lastByte := keyWallet.KeySet.PaymentAddress.Pk[len(keyWallet.KeySet.PaymentAddress.Pk)-1]
	shardID := common.GetShardIDFromLastByte(lastByte)

	return &keyWallet.KeySet, shardID, nil
}

func NewCreateRawTxParam(params interface{}) (*CreateRawTxParam, error) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 3 {
		return nil, errors.New("not enough param")
	}

	// param #1: private key of sender
	senderKeyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, errors.New("sender private key is invalid")
	}
	senderKeySet, shardIDSender, err := GetKeySetFromPrivateKeyParams(senderKeyParam)
	if err != nil {
		return nil, err
	}

	// param #2: list receivers
	receivers := make(map[string]interface{})
	if arrayParams[1] != nil {
		receivers, ok = arrayParams[1].(map[string]interface{})
		if !ok {
			return nil, errors.New("receivers param is invalid")
		}
	}
	paymentInfos := make([]*privacy.PaymentInfo, 0)
	for paymentAddressStr, amount := range receivers {
		keyWalletReceiver, err := wallet.Base58CheckDeserialize(paymentAddressStr)
		if err != nil {
			return nil, err
		}
		if len(keyWalletReceiver.KeySet.PaymentAddress.Pk) == 0 {
			return nil, fmt.Errorf("payment info %+v is invalid", paymentAddressStr)
		}

		amountParam, ok := amount.(float64)
		if !ok {
			return nil, errors.New("amount payment address is invalid")
		}
		paymentInfo := &privacy.PaymentInfo{
			Amount:         uint64(amountParam),
			PaymentAddress: keyWalletReceiver.KeySet.PaymentAddress,
		}
		paymentInfos = append(paymentInfos, paymentInfo)
	}

	// param #3: estimation fee nano P per kb
	estimateFeeCoinPerKb, ok := arrayParams[2].(float64)
	if !ok {
		return nil, errors.New("estimate fee coin per kb is invalid")
	}

	// param #4: hasPrivacyCoin flag: 1 or -1
	// default: -1 (has no privacy) (if missing this param)
	hasPrivacyCoinParam := float64(-1)
	if len(arrayParams) > 3 {
		hasPrivacyCoinParam, ok = arrayParams[3].(float64)
		if !ok {
			return nil, errors.New("has privacy for tx is invalid")
		}
	}
	hasPrivacyCoin := int(hasPrivacyCoinParam) > 0

	// param #5: meta data (optional)
	// don't do anything

	// param#6: info (optional)
	info := []byte{}
	if len(arrayParams) > 5 {
		if arrayParams[5] != nil {
			infoStr, ok := arrayParams[5].(string)
			if !ok {
				return nil, errors.New("info is invalid")
			}
			info = []byte(infoStr)
		}
	}

	return &CreateRawTxParam{
		SenderKeySet:         senderKeySet,
		ShardIDSender:        shardIDSender,
		PaymentInfos:         paymentInfos,
		EstimateFeeCoinPerKb: int64(estimateFeeCoinPerKb),
		HasPrivacyCoin:       hasPrivacyCoin,
		Info:                 info,
	}, nil
}

func NewCreateRawTxParamV2(params interface{}) (*CreateRawTxParam, error) {
	arrayParams := common.InterfaceSlice(params)
	if len(arrayParams) < 3 {
		return nil, errors.New("not enough param")
	}

	// param #1: private key of sender
	senderKeyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, errors.New("sender private key is invalid")
	}
	senderKeySet, shardIDSender, err := GetKeySetFromPrivateKeyParams(senderKeyParam)
	if err != nil {
		return nil, err
	}

	// param #2: list receivers
	receivers := make(map[string]interface{})
	if arrayParams[1] != nil {
		receivers, ok = arrayParams[1].(map[string]interface{})
		if !ok {
			return nil, errors.New("receivers param is invalid")
		}
	}
	paymentInfos := make([]*privacy.PaymentInfo, 0)
	for paymentAddressStr, amount := range receivers {
		keyWalletReceiver, err := wallet.Base58CheckDeserialize(paymentAddressStr)
		if err != nil {
			return nil, err
		}
		if len(keyWalletReceiver.KeySet.PaymentAddress.Pk) == 0 {
			return nil, fmt.Errorf("payment info %+v is invalid", paymentAddressStr)
		}

		amountParam, err := common.AssertAndConvertStrToNumber(amount)
		if err != nil {
			return nil, err
		}

		paymentInfo := &privacy.PaymentInfo{
			Amount:         amountParam,
			PaymentAddress: keyWalletReceiver.KeySet.PaymentAddress,
		}
		paymentInfos = append(paymentInfos, paymentInfo)
	}

	// param #3: estimation fee nano P per kb
	estimateFeeCoinPerKb, ok := arrayParams[2].(float64)
	if !ok {
		return nil, errors.New("estimate fee coin per kb is invalid")
	}

	// param #4: hasPrivacyCoin flag: 1 or -1
	// default: -1 (has no privacy) (if missing this param)
	hasPrivacyCoinParam := float64(-1)
	if len(arrayParams) > 3 {
		hasPrivacyCoinParam, ok = arrayParams[3].(float64)
		if !ok {
			return nil, errors.New("has privacy for tx is invalid")
		}
	}
	hasPrivacyCoin := int(hasPrivacyCoinParam) > 0

	// param #5: meta data (optional)
	// don't do anything

	// param#6: info (optional)
	info := []byte{}
	if len(arrayParams) > 5 {
		if arrayParams[5] != nil {
			infoStr, ok := arrayParams[5].(string)
			if !ok {
				return nil, errors.New("info is invalid")
			}
			info = []byte(infoStr)
		}

	}

	return &CreateRawTxParam{
		SenderKeySet:         senderKeySet,
		ShardIDSender:        shardIDSender,
		PaymentInfos:         paymentInfos,
		EstimateFeeCoinPerKb: int64(estimateFeeCoinPerKb),
		HasPrivacyCoin:       hasPrivacyCoin,
		Info:                 info,
	}, nil
}
