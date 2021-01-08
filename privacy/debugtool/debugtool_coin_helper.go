package debugtool

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/wallet"
	"math/big"
)

//OutCoinKey is used to retrieve output coins via RPC.
//
//Payment address must always be present. Other fields are optional
type OutCoinKey struct {
	paymentAddress string
	otaKey         string
	readonlyKey	   string
}

func (outCoinKey *OutCoinKey) SetOTAKey(otaKey string){
	outCoinKey.otaKey = otaKey
}

func (outCoinKey *OutCoinKey) SetPaymentAddress(paymentAddress string){
	outCoinKey.paymentAddress = paymentAddress
}

func (outCoinKey *OutCoinKey) SetReadonlyKey(readonlyKey string){
	outCoinKey.readonlyKey = readonlyKey
}

func NewOutCoinKey(paymentAddress, otaKey, readonlyKey string) *OutCoinKey {
	return &OutCoinKey{paymentAddress: paymentAddress, otaKey: otaKey, readonlyKey: readonlyKey}
}

func NewOutCoinKeyFromPrivateKey(privateKey string) (*OutCoinKey, error) {
	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil{
		return nil, err
	}

	err = keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	if err != nil{
		return nil, err
	}
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	otaSecretKey := keyWallet.Base58CheckSerialize(wallet.OTAKeyType)
	viewingKeyStr := keyWallet.Base58CheckSerialize(wallet.ReadonlyKeyType)

	return &OutCoinKey{paymentAddress: paymentAddStr, otaKey: otaSecretKey, readonlyKey: viewingKeyStr}, nil
}

func ParseCoinFromJsonResponse(b []byte) ([]jsonresult.ICoinInfo, []*big.Int, error){
	response, err := ParseResponse(b)
	fmt.Println(err)
	if err != nil{
		return nil, nil, err
	}

	var tmp jsonresult.ListOutputCoins
	err = json.Unmarshal(response.Result, &tmp)
	if err != nil {
		return nil, nil, err
	}

	resultOutCoins := make([]jsonresult.ICoinInfo, 0)
	listOutputCoins := tmp.Outputs
	for _, value := range listOutputCoins {
		for _, outCoin := range value {
			out, _, err := jsonresult.NewCoinFromJsonOutCoin(outCoin)
			if err != nil {
				return nil, nil, err
			}

			resultOutCoins = append(resultOutCoins, out)
		}
	}

	return resultOutCoins, nil, nil
}

