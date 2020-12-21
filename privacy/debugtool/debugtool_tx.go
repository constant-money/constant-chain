package debugtool

import (
	"encoding/json"
	"errors"
	"fmt"
	// "strconv"

	"github.com/incognitochain/incognito-chain/wallet"
	// "github.com/incognitochain/incognito-chain/privacy/coin"
	// "github.com/incognitochain/incognito-chain/transaction"
)

var privIndicator string = "1"

// Parse from byte to AutoTxByHash
func ParseAutoTxHashFromBytes(b []byte) (*AutoTxByHash, error) {
	data := new(AutoTxByHash)
	err := json.Unmarshal(b, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Query the RPC server then return the AutoTxByHash
func (this *DebugTool) getAutoTxByHash(txHash string) (*AutoTxByHash, error) {
	if len(this.url) == 0 {
		return nil, errors.New("Debugtool has not set mainnet or testnet")
	}
	query := fmt.Sprintf(`{
		"jsonrpc":"1.0",
		"method":"gettransactionbyhash",
		"params":["%s"],
		"id":1
	}`, txHash)
	b, err := this.SendPostRequestWithQuery(query)
	if err != nil {
		return nil, err
	}
	autoTx, txError := ParseAutoTxHashFromBytes(b)
	if txError != nil {
		return nil, err
	}
	return autoTx, nil
}

// Get only the proof of transaction requiring the txHash
func (this *DebugTool) GetProofTransactionByHash(txHash string) (string, error) {
	tx, err := this.getAutoTxByHash(txHash)
	if err != nil {
		return "", err
	}
	return tx.Result.Proof, nil
}

// Get only the Sig of transaction requiring the txHash
func (this *DebugTool) GetSigTransactionByHash(txHash string) (string, error) {
	tx, err := this.getAutoTxByHash(txHash)
	if err != nil {
		return "", err
	}
	return tx.Result.Sig, nil
}

// Get only the BlockHash of transaction requiring the txHash
func (this *DebugTool) GetBlockHashTransactionByHash(txHash string) (string, error) {
	tx, err := this.getAutoTxByHash(txHash)
	if err != nil {
		return "", err
	}
	return tx.Result.BlockHash, nil
}

// Get only the BlockHeight of transaction requiring the txHash
func (this *DebugTool) GetBlockHeightTransactionByHash(txHash string) (int, error) {
	tx, err := this.getAutoTxByHash(txHash)
	if err != nil {
		return -1, err
	}
	return tx.Result.BlockHeight, nil
}

// Get the whole result of rpc call 'gettransactionbyhash'
func (this *DebugTool) GetTransactionByHash(txHash string) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}
	query := fmt.Sprintf(`{
		"jsonrpc":"1.0",
		"method":"gettransactionbyhash",
		"params":["%s"],
		"id":1
	}`, txHash)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) CreateAndSendTransaction() ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}
	query := `{
		"jsonrpc": "1.0",
		"method": "createandsendtransaction",
		"params": [
			"112t8roafGgHL1rhAP9632Yef3sx5k8xgp8cwK4MCJsCL1UWcxXvpzg97N4dwvcD735iKf31Q2ZgrAvKfVjeSUEvnzKJyyJD3GqqSZdxN4or",
			{
				"12RuhVZQtGgYmCVzVi49zFZD7gR8SQx8Uuz8oHh6eSZ8PwB2MwaNE6Kkhd6GoykfkRnHNSHz1o2CzMiQBCyFPikHmjvvrZkLERuhcVE":200000000000000,
				"12RxDSnQVjPojzf7uju6dcgC2zkKkg85muvQh347S76wKSSsKPAqXkvfpSeJzyEH3PREHZZ6SKsXLkDZbs3BSqwEdxqprqih4VzANK9":200000000000000,
				"12S6m2LpzN17jorYnLb2ApNKaV2EVeZtd6unvrPT1GH8yHGCyjYzKbywweQDZ7aAkhD31gutYAgfQizb2JhJTgBb3AJ8aB4hyppm2ax":200000000000000,
				"12S42y9fq8xWXx1YpZ6KVDLGx6tLjeFWqbSBo6zGxwgVnPe1rMGxtLs87PyziCzYPEiRGdmwU1ewWFXwjLwog3X71K87ApNUrd3LQB3":200000000000000,
				"12S3yvTvWUJfubx3whjYLv23NtaNSwQMGWWScSaAkf3uQg8xdZjPFD4fG8vGvXjpRgrRioS5zuyzZbkac44rjBfs7mEdgoL4pwKu87u":200000000000000,
				"12S6mGbnS3Df5bGBaUfBTh56NRax4PvFPDhUnxvP9D6cZVjnTx9T4FsVdFT44pFE8KXTGYaHSAmb2MkpnUJzkrAe49EPHkBULM8N2ZJ":200000000000000,
				"12Rs5tQTYkWGzEdPNo2GRA1tjZ5aDCTYUyzXf6SJFq89QnY3US3ZzYSjWHVmmLUa6h8bdHHUuVYoR3iCVRoYDCNn1AfP6pxTz5YL8Aj":200000000000000,
				"12S33dTF3aVsuSxY7iniK3UULUYyLMZumExKm6DPfsqnNepGjgDZqkQCDp1Z7Te9dFKQp7G2WeeYqCr5vcDCfrA3id4x5UvL4yyLrrT":200000000000000
			},
			1,
			1
		],
		"id": 1
	}`
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) CreateAndSendTransactionFromAToB(privKeyA string, paymentAddress string, amount string) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	query := fmt.Sprintf(`{
		"jsonrpc": "1.0",
		"method": "createandsendtransaction",
		"params": [
			"%s",
			{
				"%s": %s
			},
			1,
			%s
		],
		"id": 1
	}`, privKeyA, paymentAddress, amount, privIndicator)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) GetListOutputCoins(privKeyStr, tokenID string, h uint64) ([]byte, error) {
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
	}`, paymentAddStr, otaSecretKey, viewingKeyStr, h, tokenID)

	//fmt.Println("==============")

	return this.SendPostRequestWithQuery(query)
}

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

func (this *DebugTool) GetListOutputTokens(privKeyStr, tokenID string) ([]byte, error) {
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
		"method": "listoutputcoins",
		"params": [
			0,
			999999,
			[
				{
			  "PaymentAddress": "%s",
			  "OTASecretKey":   "%s",
			  "ReadonlyKey": "%s",
			  "StartHeight": 0
				}
			],
			"%s"
		  ],
		"id": 1
	}`, paymentAddStr, otaSecretKey, viewingKeyStr, tokenID)

	//fmt.Println("==============")

	return this.SendPostRequestWithQuery(query)
}

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

func (this *DebugTool) GetBalanceByPrivatekey(privKeyStr string) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	query := fmt.Sprintf(`{
	   "jsonrpc":"1.0",
	   "method":"getbalancebyprivatekey",
	   "params":["%s"],
	   "id":1
	}`, privKeyStr)

	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) SubmitKey(privKeyStr string) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	query := fmt.Sprintf(`{
	   "jsonrpc":"1.0",
	   "method":"submitkey",
	   "params":["%s"],
	   "id":1
	}`, privKeyStr)

	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) CreateAndSendPrivacyCustomTokenTransaction(privKeyStr, tokenName string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

	query := fmt.Sprintf(`{
		"id": 1,
		"jsonrpc": "1.0",
		"method": "createandsendprivacycustomtokentransaction",
		"params": [
			"%s",
			{},
			5,
			1,
			{
				"Privacy": true,
				"TokenID": "",
				"TokenName": "%s",
				"TokenSymbol": "pTTT",
				"TokenFee": 0,
				"TokenTxType": 0,
				"TokenAmount": 1000000000000000000,
				"TokenReceivers": {
					"%s": 1000000000000000000
				}
			}
			]
	}`, privKeyStr, tokenName, paymentAddStr)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) ListPrivacyCustomToken() ([]byte, error) {
	query := `{
		"id": 1,
		"jsonrpc": "1.0",
		"method": "listprivacycustomtoken",
		"params": []
	}`
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) TransferPrivacyCustomToken(privKeyStrA string, paymentAddress string, tokenID string, amount string) ([]byte, error) {

	query := fmt.Sprintf(`{
		"id": 1,
		"jsonrpc": "1.0",
		"method": "createandsendprivacycustomtokentransaction",
		"params": [
			"%s",
			null,
			10,
			1,
			{
				"Privacy": true,
				"TokenID": "%s",
				"TokenName": "",
				"TokenSymbol": "",
				"TokenFee": 0,
				"TokenTxType": 1,
				"TokenAmount": 0,
				"TokenReceivers": {
					"%s": %s
				}
			}
			]
	}`, privKeyStrA, tokenID, paymentAddress, amount)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) PDEContributePRV(privKeyStr string, amount string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
				"id": 1,
				"jsonrpc": "1.0",
				"method": "createandsendtxwithprvcontribution",
				"params": [
					"%s",
					{
						"15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs": %s
					},
					-1,
					0,
					{
						"PDEContributionPairID": "newpair",
						"ContributorAddressStr": "%s",
						"ContributedAmount": %s,
						"TokenIDStr": "0000000000000000000000000000000000000000000000000000000000000004"
					}
				]
			}`, privKeyStr, amount, paymentAddStr, amount)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) PDEContributeToken(privKeyStr, tokenID, amount string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
				"id": 1,
				"jsonrpc": "1.0",
				"method": "createandsendtxwithptokencontribution",
				"params": [
					"%s",
					{},
					-1,
					0,
					{
						"Privacy": true,
						"TokenID": "%s",
						"TokenTxType": 1,
						"TokenName": "",
						"TokenSymbol": "",
						"TokenAmount": %s,
						"TokenReceivers": {
							"15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs": %s
						},
						"TokenFee": 0,
						"PDEContributionPairID": "newpair",
						"ContributorAddressStr": "%s",
						"ContributedAmount": %s,
						"TokenIDStr": "%s"
					},
					"",
					0
				]
			}`, privKeyStr, tokenID, amount, amount, paymentAddStr, amount, tokenID)
	return this.SendPostRequestWithQuery(query)
}
func (this *DebugTool) PDEWithdrawContribution(privKeyStr, tokenID1, tokenID2, amountShare string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
			"id": 1,
			"jsonrpc": "1.0",
			"method": "createandsendtxwithwithdrawalreq",
		   "params": [
				"%s",
				{
					"15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs": 0
				},
				5,
				-1,
				{
					"WithdrawalShareAmt": %s,
					"WithdrawalToken1IDStr": "%s",
					"WithdrawalToken2IDStr": "%s",
					"WithdrawerAddressStr": "%s"
				}
			]
		}`, privKeyStr, amountShare, tokenID1, tokenID2, paymentAddStr)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) PDETradePRV(privKeyStr, receiverToken, amount string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
			"id": 1,
			"jsonrpc": "1.0",
			"method": "createandsendtxwithprvtradereq",
			"params": [
				"%s",
				{
					"12RxahVABnAVCGP3LGwCn8jkQxgw7z1x14wztHzn455TTVpi1wBq9YGwkRMQg3J4e657AbAnCvYCJSdA9czBUNuCKwGSRQt55Xwz8WA": %s
				},
				-1,
				-1,
				{
					"TokenIDToBuyStr": "%s",
					"TokenIDToSellStr": "0000000000000000000000000000000000000000000000000000000000000004",
					"SellAmount": %s,
					"MinAcceptableAmount": 0,
					"TradingFee": 0,
					"TraderAddressStr": "%s"
				}
			]
		}`, privKeyStr, amount, receiverToken, amount, paymentAddStr)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) PDETradeToken(privKeyStr, sellToken, amount string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStr)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
			"id": 1,
			"jsonrpc": "1.0",
			"method": "createandsendtxwithptokentradereq",
			"params": [
				"%s",
				{},
				-1,
				0,
				{
					"Privacy": true,
					"TokenID": "%s",
					"TokenTxType": 1,
					"TokenName": "",
					"TokenSymbol": "",
					"TokenAmount": %s,
					"TokenReceivers": {
						"15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs": %s
					},
					"TokenFee": 0,
					"TokenIDToBuyStr": "0000000000000000000000000000000000000000000000000000000000000004",
					"TokenIDToSellStr": "%s",
					"SellAmount": %s,
					"MinAcceptableAmount":0,
					"TradingFee":0,
					"TraderAddressStr": "%s"
				},
				"",
				0
			]
		}`, privKeyStr, sellToken, amount, amount, sellToken, amount, paymentAddStr)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) GetBalancePrivacyCustomToken(privKeyStr string, tokenID string) ([]byte, error) {
	query := fmt.Sprintf(`{
		"id": 1,
		"jsonrpc": "1.0",
		"method": "getbalanceprivacycustomtoken",
		"params": [
			"%s",
			"%s"
		]
	}`, privKeyStr, tokenID)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) SwitchTokenCoinVersion(privKey string, tokenID string) ([]byte, error) {
	query := fmt.Sprintf(`{
		"jsonrpc": "1.0",
		"method": "createconvertcoinver1tover2txtoken",
		"params": [
			"%s",
			"%s",
			1
		],
		"id": 1
	}`, privKey, tokenID)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) SwitchCoinVersion(privKey string) ([]byte, error) {
	query := fmt.Sprintf(`{
		"jsonrpc": "1.0",
		"method": "createconvertcoinver1tover2transaction",
		"params": [
			"%s",
			1
		],
		"id": 1
	}`, privKey)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) Stake(privKey string, seed string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKey)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
	  "jsonrpc":"1.0",
	  "method":"createandsendstakingtransaction",
	  "params":[
			"%s",
			{
				"15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs": 1750000000000
			},
			5,
			0,
			{
				"StakingType": 63,
				"CandidatePaymentAddress": "%s",
				"PrivateSeed": "%s",
				"RewardReceiverPaymentAddress": "%s",
				"AutoReStaking": true
			}
	  ],
	  "id":1
	}`, privKey, paymentAddStr, seed, paymentAddStr)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) Unstake(privKey string, seed string) ([]byte, error) {
	//private key [4]
	//wrongPrivKey := "112t8rnXWRThUTJQgoyH6evV8w19dFZfKWpCh8rZpfymW9JTgKPEVQS44nDRPpsooJiGStHxu81m3HA84t9DBVobz8hgBKRMcz2hddPWNX9N"
	keyWallet, _ := wallet.Base58CheckDeserialize(privKey)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
		"id":1,
		"jsonrpc":"1.0",
		"method":"createandsendstopautostakingtransaction",
		"params": [
			"%s",
			{
				"15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs": 0
			},
			10,
			0,
			{
				"StopAutoStakingType" : 127,
				"CandidatePaymentAddress" : "%s",
				"PrivateSeed":"%s"
			}
		]
	}`, privKey, paymentAddStr, seed)
	return this.SendPostRequestWithQuery(query)
}

func (this *DebugTool) WithdrawReward(privKey string, tokenID string) ([]byte, error) {
	keyWallet, _ := wallet.Base58CheckDeserialize(privKey)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	query := fmt.Sprintf(`{
    "jsonrpc": "1.0",
    "method": "withdrawreward",
    "params": [
        "%s",
        {},
		10,
		0,
        {
            "PaymentAddress": "%s",
            "TokenID": "%s"
        }
    ],
    "id": 1
	}`, privKey, paymentAddStr, tokenID)
	return this.SendPostRequestWithQuery(query)
}

// func (this *DebugTool) CreateDoubleSpend(privKeyA string, privKeyB string, amount string, isPrivacy bool) ([]byte, error) {
// 	amountI,_ := strconv.Atoi(amount)
// 	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyB)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
// 	tx1, err := this.CreateRawTx(privKeyA, paymentAddStr, uint64(amountI), isPrivacy)

// 	keyWallet, _ = wallet.Base58CheckDeserialize(privKeyA)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr = keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
// 	tx2, err := this.CreateRawTx(privKeyA, paymentAddStr, uint64(amountI), isPrivacy)
// 	preJson := []string{EncodeBase58Check(tx1),EncodeBase58Check(tx2)}
// 	result, _ := json.Marshal(preJson)
// 	return result, err

// 	// if len(this.url) == 0 {
// 	// 	return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
// 	// }

// 	// keyWallet, _ := wallet.Base58CheckDeserialize(privKeyB)
// 	// keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	// paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

// 	// query := fmt.Sprintf(`{
// 	// 	"jsonrpc": "1.0",
// 	// 	"method": "testbuilddoublespend",
// 	// 	"params": [
// 	// 		"%s",
// 	// 		{
// 	// 			"%s": %s
// 	// 		},
// 	// 		1,
// 	// 		%s
// 	// 	],
// 	// 	"id": 1
// 	// }`, privKeyA, paymentAddStr, amount, privIndicator)
// 	// return this.SendPostRequestWithQuery(query)
// }

// func (this *DebugTool) CreateDuplicateInput(privKeyA string, privKeyB string, amount string, isPrivacy bool) ([]byte, error) {
// 	if len(this.url) == 0 {
// 		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
// 	}
// 	amountI,_ := strconv.Atoi(amount)
// 	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyB)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
// 	tx1j, err := this.CreateRawTx(privKeyA, paymentAddStr, uint64(amountI), isPrivacy)
// 	if err !=nil{
// 		return nil,err
// 	}
// 	tx1, err := transaction.NewTransactionFromJsonBytes(tx1j)
// 	if err!=nil{
// 		return nil,err
// 	}
// 	proof := tx1.GetProof()
// 	inputCoins := proof.GetInputCoins()
// 	clonedCoin := &coin.PlainCoinV1{}
// 	clonedCoin.SetBytes(inputCoins[0].Bytes())
// 	tx1.GetProof().SetInputCoins(append(inputCoins,clonedCoin))
// 	// transaction.TestResignTxV1(tx1)

// 	tx1j, _ = json.Marshal(tx1)

// 	preJson := []string{EncodeBase58Check(tx1j)}
// 	result, _ := json.Marshal(preJson)
// 	return result, err

// 	// query := fmt.Sprintf(`{
// 	// 	"jsonrpc": "1.0",
// 	// 	"method": "testbuildduplicateinput",
// 	// 	"params": [
// 	// 		"%s",
// 	// 		{
// 	// 			"%s": %s
// 	// 		},
// 	// 		1,
// 	// 		%s
// 	// 	],
// 	// 	"id": 1
// 	// }`, privKeyA, paymentAddStr, amount, privIndicator)
// 	// return this.SendPostRequestWithQuery(query)
// }

// func (this *DebugTool) CreateOutGtIn(privKeyA string, privKeyB string, amount string, isPrivacy bool) ([]byte, error) {
// 	if len(this.url) == 0 {
// 		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
// 	}
// 	amountI,_ := strconv.Atoi(amount)
// 	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyB)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	myOwnKeyWallet, _ := wallet.Base58CheckDeserialize(privKeyB)
// 	myOwnKeyWallet.KeySet.InitFromPrivateKey(&myOwnKeyWallet.KeySet.PrivateKey)
// 	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
// 	tx1j, err := this.CreateRawTx(privKeyA, paymentAddStr, uint64(amountI), isPrivacy)
// 	if err !=nil{
// 		return nil,err
// 	}
// 	tx1, err := transaction.NewTransactionFromJsonBytes(tx1j)
// 	if err!=nil{
// 		return nil,err
// 	}
// 	realFee := tx1.GetTxFee()
// 	tx1.SetTxFee(realFee + 1000)
// 	transaction.TestResignTxV1WithKey(tx1,[]byte(myOwnKeyWallet.KeySet.PrivateKey))

// 	tx1j, _ = json.Marshal(tx1)

// 	preJson := []string{EncodeBase58Check(tx1j)}
// 	result, _ := json.Marshal(preJson)
// 	return result, err

// 	// query := fmt.Sprintf(`{
// 	// 	"jsonrpc": "1.0",
// 	// 	"method": "testbuildoutgtin",
// 	// 	"params": [
// 	// 		"%s",
// 	// 		{
// 	// 			"%s": %s
// 	// 		},
// 	// 		1,
// 	// 		%s
// 	// 	],
// 	// 	"id": 1
// 	// }`, privKeyA, paymentAddStr, amount, privIndicator)
// 	// return this.SendPostRequestWithQuery(query)
// }

// func (this *DebugTool) CreateReceiverExists(privKeyA string, amount string) ([]byte, error) {
// 	if len(this.url) == 0 {
// 		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
// 	}

// 	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyA)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

// 	query := fmt.Sprintf(`{
// 		"jsonrpc": "1.0",
// 		"method": "testbuildreceiverexists",
// 		"params": [
// 			"%s",
// 			{
// 				"%s": %s
// 			},
// 			1,
// 			%s
// 		],
// 		"id": 1
// 	}`, privKeyA, paymentAddStr, amount, privIndicator)
// 	return this.SendPostRequestWithQuery(query)
// }

// func (this *DebugTool) CreateDoubleSpendToken(privKeyStrA string, privKeyStrB string, tokenID string, amount string, isPrivacy bool) ([]byte, error) {

// 	amountI,_ := strconv.Atoi(amount)
// 	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStrB)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
// 	tx1, err := this.CreateRawTxToken(privKeyStrA, tokenID, paymentAddStr, uint64(amountI), true)

// 	keyWallet, _ = wallet.Base58CheckDeserialize(privKeyStrA)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr = keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
// 	tx2, err := this.CreateRawTxToken(privKeyStrA, tokenID, paymentAddStr, uint64(amountI), true)

// 	preJson := []string{EncodeBase58Check(tx1),EncodeBase58Check(tx2)}
// 	result, _ := json.Marshal(preJson)
// 	return result, err

// 	// query := fmt.Sprintf(`{
// 	// 	"id": 1,
// 	// 	"jsonrpc": "1.0",
// 	// 	"method": "testbuilddoublespendtoken",
// 	// 	"params": [
// 	// 		"%s",
// 	// 		null,
// 	// 		10,
// 	// 		1,
// 	// 		{
// 	// 			"Privacy": true,
// 	// 			"TokenID": "%s",
// 	// 			"TokenName": "",
// 	// 			"TokenSymbol": "",
// 	// 			"TokenFee": 0,
// 	// 			"TokenTxType": 1,
// 	// 			"TokenAmount": 0,
// 	// 			"TokenReceivers": {
// 	// 				"%s": %s
// 	// 			}
// 	// 		}
// 	// 		]
// 	// }`, privKeyStrA, tokenID, paymentAddStr, amount)
// 	// return this.SendPostRequestWithQuery(query)
// }

// func (this *DebugTool) CreateDuplicateInputToken(privKeyStrA string, privKeyStrB string, tokenID string, amount string, isPrivacy bool) ([]byte, error) {
// 	if len(this.url) == 0 {
// 		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
// 	}
// 	amountI,_ := strconv.Atoi(amount)
// 	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStrB)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
// 	tx1j, err := this.CreateRawTxToken(privKeyStrA, tokenID, paymentAddStr, uint64(amountI), true)
// 	if err !=nil{
// 		return nil,err
// 	}
// 	tx1, err := transaction.NewTransactionTokenFromJsonBytes(tx1j)
// 	if err!=nil{
// 		return nil,err
// 	}
// 	proof := tx1.GetTxTokenData().TxNormal.GetProof()
// 	inputCoins := proof.GetInputCoins()
// 	clonedCoin := &coin.PlainCoinV1{}
// 	clonedCoin.SetBytes(inputCoins[0].Bytes())
// 	proof.SetInputCoins(append(inputCoins,clonedCoin))
// 	// transaction.TestResignTxV1(tx1)

// 	tx1j, _ = json.Marshal(tx1)

// 	preJson := []string{EncodeBase58Check(tx1j)}
// 	result, _ := json.Marshal(preJson)
// 	return result, err

// 	// query := fmt.Sprintf(`{
// 	// 	"id": 1,
// 	// 	"jsonrpc": "1.0",
// 	// 	"method": "testbuildduplicateinputtoken",
// 	// 	"params": [
// 	// 		"%s",
// 	// 		null,
// 	// 		10,
// 	// 		1,
// 	// 		{
// 	// 			"Privacy": true,
// 	// 			"TokenID": "%s",
// 	// 			"TokenName": "",
// 	// 			"TokenSymbol": "",
// 	// 			"TokenFee": 0,
// 	// 			"TokenTxType": 1,
// 	// 			"TokenAmount": 0,
// 	// 			"TokenReceivers": {
// 	// 				"%s": %s
// 	// 			}
// 	// 		}
// 	// 		]
// 	// }`, privKeyStrA, tokenID, paymentAddStr, amount)
// 	// return this.SendPostRequestWithQuery(query)
// }

// func (this *DebugTool) CreateReceiverExistsToken(privKeyStrA string, tokenID string, amount string) ([]byte, error) {
// 	keyWallet, _ := wallet.Base58CheckDeserialize(privKeyStrA)
// 	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

// 	query := fmt.Sprintf(`{
// 		"id": 1,
// 		"jsonrpc": "1.0",
// 		"method": "testbuildreceiverexiststoken",
// 		"params": [
// 			"%s",
// 			null,
// 			10,
// 			1,
// 			{
// 				"Privacy": true,
// 				"TokenID": "%s",
// 				"TokenName": "",
// 				"TokenSymbol": "",
// 				"TokenFee": 0,
// 				"TokenTxType": 1,
// 				"TokenAmount": 0,
// 				"TokenReceivers": {
// 					"%s": %s
// 				}
// 			}
// 			]
// 	}`, privKeyStrA, tokenID, paymentAddStr, amount)
// 	return this.SendPostRequestWithQuery(query)
// }
