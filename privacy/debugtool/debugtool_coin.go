package debugtool

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/wallet"
	"math/big"
)


func (tool *DebugTool) GetOutputCoins(outCoinKey *OutCoinKey, tokenID string, height uint64) ([]jsonresult.ICoinInfo, []*big.Int, error) {
	b, err := tool.GetListOutputCoins(outCoinKey, tokenID, height)
	if err != nil{
		return nil, nil, err
	}

	//fmt.Println(string(b))

	return ParseCoinFromJsonResponse(b)
}

func (tool *DebugTool) CheckCoinsSpent(outCoinKey *OutCoinKey, tokenID string, snList []string) ([]bool, error){
	b, err := tool.HasSerialNumber(outCoinKey.paymentAddress, tokenID, snList)
	if err != nil{
		return []bool{}, err
	}

	response, err := ParseResponse(b)
	if err != nil{
		return []bool{}, err
	}

	var tmp []bool
	err = json.Unmarshal(response.Result, &tmp)
	if err != nil {
		return []bool{}, err
	}

	if len(tmp) != len(snList){
		return []bool{}, errors.New(fmt.Sprintf("Length of result and length of snList mismathc: len(Result) = %v, len(snList) = %v", len(tmp), len(snList)))
	}

	return []bool{}, nil
}

//===================== OUTPUT COINS RPC =====================//

//GetListOutputCoins retrieves list of output coins of an OutCoinKey and returns the result in raw json bytes.
func (this *DebugTool) GetListOutputCoins(outCoinKey *OutCoinKey, tokenID string, h uint64) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	query := fmt.Sprintf(`{
		"jsonrpc": "1.0",
		"method": "listoutputcoins",
		"params": [
			0,
			999999,
			[
				{
			  "PaymentAddress": "%s",
			  "OTASecretKey": "%s",
			  "ReadonlyKey" : "%s",
			  "StartHeight": %d
				}
			],
		  "%s"
		  ],
		"id": 1
	}`, outCoinKey.paymentAddress, outCoinKey.otaKey, outCoinKey.readonlyKey, h, tokenID)

	return this.SendPostRequestWithQuery(query)
}

//GetListOutputCoinsCached retrieves list of output coins (which have been cached at the fullnode) of an OutCoinKey and returns the result in raw json bytes.
func (this *DebugTool) GetListOutputCoinsCached(privKeyStr, tokenID string, h uint64) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	otaSecretKey := keyWallet.Base58CheckSerialize(wallet.OTAKeyType)
	viewingKeyStr := keyWallet.Base58CheckSerialize(wallet.ReadonlyKeyType)

	query := fmt.Sprintf(`{
		"jsonrpc": "1.0",
		"method": "listoutputcoinsfromcache",
		"params": [
			0,
			999999,
			[
				{
			  "PaymentAddress": "%s",
			  "OTASecretKey": "%s",
			  "ReadonlyKey" : "%s",
			  "StartHeight": %d
				}
			],
		  "%s"
		  ],
		"id": 1
	}`, paymentAddStr, otaSecretKey, viewingKeyStr, h, tokenID)

	//fmt.Println("==============")

	return this.SendPostRequestWithQuery(query)
}

//ListUnspentOutputCoins retrieves list of output coins of an OutCoinKey and returns the result in raw json bytes.
//
//NOTE: PrivateKey must be supplied.
func (this *DebugTool) ListUnspentOutputCoins(privKeyStr string) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	query := fmt.Sprintf(`{
	   "jsonrpc":"1.0",
	   "method":"listunspentoutputcoins",
	   "params":[
		  0,
		  999999,
		  [
			 {
				"PrivateKey":"%s",
				"StartHeight": 0
			 }

		  ]
	   ],
	   "id":1
	}`, privKeyStr)

	return this.SendPostRequestWithQuery(query)
}

//ListUnspentOutputCoins retrieves list of output tokens of an OutCoinKey and returns the result in raw json bytes.
//
//NOTE: PrivateKey must be supplied.
func (this *DebugTool) GetListUnspentOutputTokens(privKeyStr, tokenID string) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)

	query := fmt.Sprintf(`{
	   "jsonrpc":"1.0",
	   "method":"listunspentoutputtokens",
	   "params":[
		  0,
		  999999,
		  [
			 {
				"PrivateKey":"%s",
				"StartHeight": 0,
				"tokenID" : "%s"
			 }

		  ]
	   ],
	   "id":1
	}`, privKeyStr, tokenID)

	return this.SendPostRequestWithQuery(query)
}

//ListPrivacyCustomToken lists all tokens currently present on the blockchain
func (this *DebugTool) ListPrivacyCustomToken() ([]byte, error) {
	query := `{
		"id": 1,
		"jsonrpc": "1.0",
		"method": "listprivacycustomtoken",
		"params": []
	}`
	return this.SendPostRequestWithQuery(query)
}

//HasSerialNumber checks if the provided serial numbers have been spent or not
func (tool *DebugTool) HasSerialNumber(paymentAddress, tokenID string, snList []string) ([]byte, error){
	query := fmt.Sprintf(`{
		"jsonrpc": "1.0",
		"method": "hasserialnumbers",
		"params": [
			%s,
			%v,
			%s,
		"id": 1
	}`, paymentAddress, snList, tokenID)

	return tool.SendPostRequestWithQuery(query)

}

//===================== END OF OUTPUT COINS RPC =====================//


//func (tool *DebugTool) GetUnspentOutputCoins(priva)
