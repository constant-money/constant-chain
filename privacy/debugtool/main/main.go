package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"math/big"
	"os"
	"strconv"
	"strings"

	//"os"
	//"strconv"

	"github.com/incognitochain/incognito-chain/privacy/debugtool"
	"github.com/incognitochain/incognito-chain/wallet"
)

func sendTx(tool *debugtool.DebugTool) {
	b, _ := tool.CreateAndSendTransaction()
	fmt.Println(string(b))
}

func InitToken(tool *debugtool.DebugTool, privKey, tokenName string) {
	fmt.Println("========== INIT TOKEN ==========")
	b, _ := tool.CreateAndSendPrivacyCustomTokenTransaction(privKey, tokenName)
	fmt.Println(string(b))
	fmt.Println("========== END INIT TOKEN ==========")
}

func ListTokens(tool *debugtool.DebugTool) *debugtool.ListCustomToken {
	fmt.Println("========== LIST ALL TOKEN ==========")
	b, _ := tool.ListPrivacyCustomToken()
	res := new(debugtool.ListCustomToken)
	_ = json.Unmarshal(b, res)
	fmt.Println("Number of Token: ", len(res.Result.ListCustomToken))
	if len(res.Result.ListCustomToken) > 0 {
		for _, token := range res.Result.ListCustomToken {
			fmt.Println("Token ", token.Name, token.ID)
		}
		fmt.Println("========== END LIST ALL TOKEN ==========")
		return res
	}
	fmt.Println("========== END LIST ALL TOKEN ==========")
	return nil
}

func GetOutputToken(tool *debugtool.DebugTool, privKey string, tokenID string, height uint64) {
	fmt.Println("========== GET OUTPUT TOKEN ==========")
	b, _ := tool.GetListOutputCoins(privKey, tokenID, height)
	fmt.Println(string(b))
	fmt.Println("========== END OUTPUT TOKEN ==========")
}

func GetUnspentOutputToken(tool *debugtool.DebugTool, privKey string, tokenID string) {
	fmt.Println("========== GET UNSPENT OUTPUT TOKEN ==========")
	b, _ := tool.GetListUnspentOutputTokens(privKey, tokenID)
	fmt.Println(string(b))
	fmt.Println("========== END UNSPENT OUTPUT TOKEN ==========")
}

func TransferToken(tool *debugtool.DebugTool, fromPrivKey, paymentAddress, tokenID, amount string) {
	fmt.Println("========== TRANSFER TOKEN ==========")
	b, _ := tool.TransferPrivacyCustomToken(fromPrivKey, paymentAddress, tokenID, amount)
	fmt.Println(string(b))
	fmt.Println("========== END TRANSFER TOKEN ==========")
}

func GetBalanceToken(tool *debugtool.DebugTool, privkey, tokenID string) {
	fmt.Println("========== GET TOKEN BALANCE ==========")
	b, _ := tool.GetBalancePrivacyCustomToken(privkey, tokenID)
	fmt.Println(string(b))
	fmt.Println("========== END GET TOKEN BALANCE ==========")
}

func privateKeyToPaymentAddress(privkey string) string {
	keyWallet, _ := wallet.Base58CheckDeserialize(privkey)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentAddStr := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	return paymentAddStr
}

func privateKeyToPublicKey(privkey string) []byte {
	keyWallet, err := wallet.Base58CheckDeserialize(privkey)
	if err != nil {
		panic(err)
	}
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	return keyWallet.KeySet.PaymentAddress.Pk
}

func ConvertCoinVersion(tool *debugtool.DebugTool, privKey string) {
	fmt.Println("========== CONVERT COIN ==========")
	b, _ := tool.SwitchCoinVersion(privKey)
	fmt.Println(string(b))
	fmt.Println("========== END CONVERT COIN ==========")
}

func ConvertTokenCoinVersion(tool *debugtool.DebugTool, privKey string, tokenID string) {
	fmt.Println("CONVERT TOKEN COIN")
	b, _ := tool.SwitchTokenCoinVersion(privKey, tokenID)
	fmt.Println(string(b))
	fmt.Println("END CONVERT TOKEN COIN")
}

func GetPRVOutPutCoin(tool *debugtool.DebugTool, privkey string, height uint64) {
	fmt.Println("========== GET PRV OUTPUT COIN ==========")
	b, _ := tool.GetListOutputCoins(privkey, common.PRVIDStr, height)
	fmt.Println(string(b))
	fmt.Println("========== END GET PRV OUTPUT COIN ==========")
}


func GetPRVOutPutCoinCached(tool *debugtool.DebugTool, privkey string) {
	fmt.Println("========== GET CACHED PRV OUTPUT COIN ==========")
	b, _ := tool.GetListOutputCoinsCached(privkey, common.PRVIDStr, uint64(0))
	fmt.Println(string(b))
	fmt.Println("========== END GET CACHED PRV OUTPUT COIN ==========")
}
func GetOutputTokenCached(tool *debugtool.DebugTool, privKey string, tokenID string) {
	fmt.Println("========== GET CACHED OUTPUT TOKEN ==========")
	b, _ := tool.GetListOutputCoinsCached(privKey, tokenID, uint64(0))
	fmt.Println(string(b))
	fmt.Println("========== END GET CACHED OUTPUT TOKEN ==========")
}

func GetPRVBalance(tool *debugtool.DebugTool, privkey string) {
	fmt.Println("========== GET PRV BALANCE ==========")
	b, _ := tool.GetBalanceByPrivatekey(privkey)
	fmt.Println(string(b))
	fmt.Println("========== END GET PRV BALANCE ==========")
}

func SubmitOTAKey(tool *debugtool.DebugTool, privkey string) {
	fmt.Println("========== SUBMIT KEY ==========")
	b, _ := tool.SubmitKey(privkey)
	fmt.Println(string(b))
	fmt.Println("========== END SUBMIT KEY ==========")
}

func GetRawMempool(tool *debugtool.DebugTool) {
	fmt.Println("========== GET RAW MEMPOOL ==========")
	b, _ := tool.GetRawMempool()
	fmt.Println(string(b))
	fmt.Println("========== END GET RAW MEMPOOL ==========")
}

func GetTxByHash(tool *debugtool.DebugTool, txHash string) {
	fmt.Println("========== GET TX BY HASH ==========")
	b, _ := tool.GetTransactionByHash(txHash)
	fmt.Println(string(b))
	fmt.Println("========== END GET TX BY HASH ==========")
}

func Staking(tool *debugtool.DebugTool, privKey string, privSeed string) {
	fmt.Println("========== STAKING  ==========")
	b, _ := tool.Stake(privKey, privSeed)
	fmt.Println(string(b))
	fmt.Println("========== END STAKING ==========")
}

func UnStaking(tool *debugtool.DebugTool, privKey string, privSeed string) {
	fmt.Println("========== UN-STAKING  ==========")
	b, _ := tool.Unstake(privKey, privSeed)
	fmt.Println(string(b))
	fmt.Println("========== END UN-STAKING ==========")
}

func WithdrawReward(tool *debugtool.DebugTool, privKey string, tokenID string) {
	fmt.Println("========== WITHDRAW REWARD  ==========")
	b, _ := tool.WithdrawReward(privKey, tokenID)
	fmt.Println(string(b))
	fmt.Println("========== END WITHDRAW REWARD  ==========")
}

func SwitchPort(newPort string) *debugtool.DebugTool {
	tool := new(debugtool.DebugTool).InitLocal(newPort)
	return tool
}

func TransferPRV(tool *debugtool.DebugTool, fromPrivKey, paymentAddress, amount string) {
	fmt.Println("========== TRANSFER PRV  ==========")
	b, _ := tool.CreateAndSendTransactionFromAToB(fromPrivKey, paymentAddress, amount)
	fmt.Println(string(b))
	fmt.Println("========== END TRANSFER PRV  ==========")
}

func GenKeySet(b []byte) (string, string, string) {
	if b == nil {
		b = privacy.RandomScalar().ToBytesS()
	}

	seed := operation.HashToScalar(b).ToBytesS()

	keyWallet, err := wallet.NewMasterKey(seed)
	if err != nil {
		return "", "", ""
	}

	privateKey := keyWallet.Base58CheckSerialize(wallet.PriKeyType)
	paymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	readOnly := keyWallet.Base58CheckSerialize(wallet.ReadonlyKeyType)

	return privateKey, paymentAddress, readOnly
}

func GenKeySetApp() (string, string, string) {
	for i := 0; i < 10000; i++ {
		seed := common.RandBytes(32)

		keyWallet, err := wallet.NewMasterKey(seed)
		if err != nil {
			return "", "", ""
		}

		privateKey := keyWallet.Base58CheckSerialize(wallet.PriKeyType)
		if privateKey[:3] == "112" {
			paymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
			readOnly := keyWallet.Base58CheckSerialize(wallet.ReadonlyKeyType)
			return privateKey, paymentAddress, readOnly
		}

	}

	return "", "", ""

}

// func DoubleSpendPRV(tool *debugtool.DebugTool, fromPrivKey, toPrivKey, amount string) {
// 	fmt.Println("========== TRANSFER PRV (DOUBLE SPEND - TEST) ==========")
// 	b, _ := tool.CreateDoubleSpend(fromPrivKey, toPrivKey, amount)
// 	fmt.Println(string(b))
// 	fmt.Println("========== END TRANSFER PRV (DOUBLE SPEND - TEST) ==========")
// }

// func DoubleSpendToken(tool *debugtool.DebugTool, fromPrivKey, toPrivKey, tokenID string, amount string) {
// 	fmt.Println("========== TRANSFER PRV (DOUBLE SPEND TOKEN - TEST) ==========")
// 	b, _ := tool.CreateDoubleSpendToken(fromPrivKey, toPrivKey, tokenID, amount)
// 	fmt.Println(string(b))
// 	fmt.Println("========== END TRANSFER PRV (DOUBLE SPEND TOKEN - TEST) ==========")
// }

// func DoCreateDuplicateInputTx(tool *debugtool.DebugTool, fromPrivKey, toPrivKey, amount string) {
// 	fmt.Println("========== TRANSFER PRV (DUP INPUT - TEST) ==========")
// 	b, _ := tool.CreateDuplicateInput(fromPrivKey, toPrivKey, amount)
// 	fmt.Println(string(b))
// 	fmt.Println("========== END TRANSFER PRV (DUP INPUT - TEST) ==========")
// }

// func DoCreateDuplicateInputTokenTx(tool *debugtool.DebugTool, fromPrivKey, toPrivKey, tokenID string, amount string) {
// 	fmt.Println("========== TRANSFER PRV (DUPLICATE INPUT TOKEN - TEST) ==========")
// 	b, _ := tool.CreateDuplicateInputToken(fromPrivKey, toPrivKey, tokenID, amount)
// 	fmt.Println(string(b))
// 	fmt.Println("========== END TRANSFER PRV (DUPLICATE INPUT TOKEN - TEST) ==========")
// }

// func DoCreateOutGtInTx(tool *debugtool.DebugTool, fromPrivKey, toPrivKey, amount string) {
// 	fmt.Println("========== TRANSFER PRV (OUT > IN - TEST) ==========")
// 	b, _ := tool.CreateOutGtIn(fromPrivKey, toPrivKey, amount)
// 	fmt.Println(string(b))
// 	fmt.Println("========== END TRANSFER PRV (OUT > IN - TEST) ==========")
// }

// func DoCreateReceiverExistsTx(tool *debugtool.DebugTool, fromPrivKey, amount string) {
// 	fmt.Println("========== TRANSFER PRV (OTA EXISTS - TEST) ==========")
// 	b, _ := tool.CreateReceiverExists(fromPrivKey, amount)
// 	fmt.Println(string(b))
// 	fmt.Println("========== END TRANSFER PRV (OTA EXISTS - TEST) ==========")
// }

// func DoCreateReceiverExistsTokenTx(tool *debugtool.DebugTool, fromPrivKey, tokenID string, amount string) {
// 	fmt.Println("========== TRANSFER PRV (OTA EXISTS TOKEN - TEST) ==========")
// 	b, _ := tool.CreateReceiverExistsToken(fromPrivKey, tokenID, amount)
// 	fmt.Println(string(b))
// 	fmt.Println("========== END TRANSFER PRV (OTA EXISTS TOKEN - TEST) ==========")
// }

// PDE
func PDEContributePRV(tool *debugtool.DebugTool, privKey, amount string) {
	fmt.Println("========== PDE CONTRIBUTE PRV  ==========")
	b, _ := tool.PDEContributePRV(privKey, amount)
	fmt.Println(string(b))
	fmt.Println("========== END PDE CONTRIBUTE PRV  ==========")
}
func PDEContributeToken(tool *debugtool.DebugTool, privKey, tokenID, amount string) {
	fmt.Println("========== PDE CONTRIBUTE TOKEN  ==========")
	b, _ := tool.PDEContributeToken(privKey, tokenID, amount)
	fmt.Println(string(b))
	fmt.Println("========== END PDE CONTRIBUTE TOKEN  ==========")
}

func PDEWithdrawContribution(tool *debugtool.DebugTool, privKey, tokenID1, tokenID2, amountShare string) {
	fmt.Println("========== PDE WITHDRAW   ==========")
	b, _ := tool.PDEWithdrawContribution(privKey, tokenID1, tokenID2, amountShare)
	fmt.Println(string(b))
	fmt.Println("========== END PDE WITHDRAW  ==========")
}

func PDETradePRV(tool *debugtool.DebugTool, privKey, token, amount string) {
	fmt.Println("========== PDE TRADE PRV   ==========")
	b, _ := tool.PDETradePRV(privKey, token, amount)
	fmt.Println(string(b))
	fmt.Println("========== END TRADE PRV  ==========")
}

func PDETradeToken(tool *debugtool.DebugTool, privKey, token, amount string) {
	fmt.Println("========== PDE TRADE TOKEN   ==========")
	b, _ := tool.PDETradeToken(privKey, token, amount)
	fmt.Println(string(b))
	fmt.Println("========== END PDE TRADE TOKEN  ==========")
}

// func sendRawTxNoPrivacy(tool *debugtool.DebugTool, privKey, tokenID, paymentString string, txType int64){
// 	fmt.Println("========== FAKE TRANSACTION ==========")
// 	if len(paymentString) < 2{
// 		keyWallet, _ := wallet.Base58CheckDeserialize(privKey)
// 		keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
// 		paymentString = keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

// 	}
// 	b, _ := tool.SendTxNoPrivacyFake(privKey, tokenID, paymentString, txType, 1)
// 	fmt.Println(string(b))
// 	fmt.Println("========== FAKE TRANSACTION FINISH ==========")
// }

// func sendRawTxPrivacy(tool *debugtool.DebugTool, privKey, tokenID, paymentString string, txType int64){
// 	fmt.Println("========== FAKE TRANSACTION ==========")
// 	b, err := tool.SendTxPrivacyFake(privKey, tokenID, paymentString, txType, 1)
// 	//if err != nil{
// 	//	return
// 	//}
// 	fmt.Println("err =", err)
// 	fmt.Println(string(b))
// 	fmt.Println("========== FAKE TRANSACTION FINISH ==========")
// }

func GetListRandomCommitments(tool *debugtool.DebugTool, privKey, tokenID string) {
	fmt.Println("========== GET LIST RANDOM COMMITMENTS ==========")
	outCoins, _, err := tool.GetPlainOutputCoin(privKey, tokenID)
	if err != nil {
		panic(err)
	}

	keyWallet, _ := wallet.Base58CheckDeserialize(privKey)
	keyWallet.KeySet.InitFromPrivateKey(&keyWallet.KeySet.PrivateKey)
	paymentString := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

	if err != nil {
		panic(err)
	}

	b, err := tool.GetRandomCommitment(common.PRVIDStr, paymentString, outCoins)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))

	fmt.Println("========== GET LIST RANDOM COMMITMENT FINISH ==========")
}

func GetListRandomCommitmentsAndPublicKeys(tool *debugtool.DebugTool, paymentAddress, tokenID string, numOutputs int) {
	fmt.Println("========== GET LIST RANDOM COMMITMENTS AND PUBLIC KEYS ==========")

	b, err := tool.GetRandomCommitmentsAndPublicKeys(tokenID, paymentAddress, numOutputs)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))

	fmt.Println("========== GET LIST RANDOM COMMITMENTS AND PUBLIC KEYS FINISH ==========")
}

//Blockchain
func GetBlockchainInfo(tool *debugtool.DebugTool) {
	fmt.Println("========== GET BLOCKCHAIN INFO ==========")
	b, _ := tool.GetBlockchainInfo()
	fmt.Println(string(b))
	fmt.Println("========== END GET BLOCKCHAIN INFO ==========")
}

func GetShardIDFromPrivateKey(privateKey string) byte {
	pubkey := privateKeyToPublicKey(privateKey)
	return common.GetShardIDFromLastByte(pubkey[len(pubkey)-1])
}

//Comment the init function in blockchain/constants.go to run the debug tool.
func main() {
	privateKeys := []string{
		"112t8roafGgHL1rhAP9632Yef3sx5k8xgp8cwK4MCJsCL1UWcxXvpzg97N4dwvcD735iKf31Q2ZgrAvKfVjeSUEvnzKJyyJD3GqqSZdxN4or",
		"112t8rnZDRztVgPjbYQiXS7mJgaTzn66NvHD7Vus2SrhSAY611AzADsPFzKjKQCKWTgbkgYrCPo9atvSMoCf9KT23Sc7Js9RKhzbNJkxpJU6",
		"112t8rne7fpTVvSgZcSgyFV23FYEv3sbRRJZzPscRcTo8DsdZwstgn6UyHbnKHmyLJrSkvF13fzkZ4e8YD5A2wg8jzUZx6Yscdr4NuUUQDAt",
		"112t8rnXoBXrThDTACHx2rbEq7nBgrzcZhVZV4fvNEcGJetQ13spZRMuW5ncvsKA1KvtkauZuK2jV8pxEZLpiuHtKX3FkKv2uC5ZeRC8L6we",
		"112t8rnbcZ92v5omVfbXf1gu7j7S1xxr2eppxitbHfjAMHWdLLBjBcQSv1X1cKjarJLffrPGwBhqZzBvEeA9PhtKeM8ALWiWjhUzN5Fi6WVC",
		"112t8rnZUQXxcbayAZvyyZyKDhwVJBLkHuTKMhrS51nQZcXKYXGopUTj22JtZ8KxYQcak54KUQLhimv1GLLPFk1cc8JCHZ2JwxCRXGsg4gXU",
		"112t8rnXDS4cAjFVgCDEw4sWGdaqQSbKLRH1Hu4nUPBFPJdn29YgUei2KXNEtC8mhi1sEZb1V3gnXdAXjmCuxPa49rbHcH9uNaf85cnF3tMw",
		"112t8rnYoioTRNsM8gnUYt54ThWWrRnG4e1nRX147MWGbEazYP7RWrEUB58JLnBjKhh49FMS5o5ttypZucfw5dFYMAsgDUsHPa9BAasY8U1i",
		"112t8rnXtw6pWwowv88Ry4XxukFNLfbbY2PLh2ph38ixbCbZKwf9ZxVjd4s7jU3RSdKctC7gGZp9piy8nZoLqHwqDBWcsMHWsQg27S5WCdm4",
	}
	privateSeeds := []string{
		"12MZ4QiFoETNbdLKgRQWPMQMqsceWPKo71Jma9NzwvLTabpcDhn",
		"158ZGK5EHmoyrHEd8aA2HCaqbNQ4r45ZsnwL4Zh8mH8dueWHWs",
		"1mYRSzV7yigD7qNpuQwnyKeVMQcnenjSxAB1L8MEpDuT3RRbZc",
		"1G5Q9uGSxekPSgC1w1ZFaDJ8RxeYrekk2FtFLF33QCKNbg2V88",
		"1mYRSzV7yigD7qNpuQwnyKeVMQcnenjSxAB1L8MEpDuT3RRbZc",
		"12MZ4QiFoETNbdLKgRQWPMQMqsceWPKo71Jma9NzwvLTabpcDhn",
		"1G5Q9uGSxekPSgC1w1ZFaDJ8RxeYrekk2FtFLF33QCKNbg2V88",
		"1cQCTV1m33LxBKpNW2SisbuJfp5VcBSEau7PE5aD16gGLAN7eq",
		"1DRtHReKFQqPQzd689EsjivNgUScPBUwvbw8azgxaLRUBtmFL2",
		"12YCyPu6KyBToSaaQkw7hzbWkJnUi78DLkfvWokgi4dCtkbVusC",
	}

	tokenIDs := make(map[string]string)
	tokenIDs["USDT"] = "716fd1009e2a1669caacc36891e707bfdf02590f96ebd897548e8963c95ebac0"
	tokenIDs["BNB"] = "b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"
	tokenIDs["ETH"] = "ffd8d42dc40a8d166ea4848baf8b5f6e912ad79875f4373070b59392b1756c8f"
	tokenIDs["USDC"] = "1ff2da446abfebea3ba30385e2ca99b0f0bbeda5c6371f4c23c939672b429a42"
	tokenIDs["XMR"] = "c01e7dc1d1aba995c19b257412340b057f8ad1482ccb6a9bb0adce61afbf05d4"
	tokenIDs["BTC"] = "b832e5d3b1f01a4f0623f7fe91d6673461e1f5d37d91fe78c5c2e6183ff39696"

	//paymentKeys := []string{
	//	"",
	//	"12RuhVZQtGgYmCVzVi49zFZD7gR8SQx8Uuz8oHh6eSZ8PwB2MwaNE6Kkhd6GoykfkRnHNSHz1o2CzMiQBCyFPikHmjvvrZkLERuhcVE",
	//	"12RxDSnQVjPojzf7uju6dcgC2zkKkg85muvQh347S76wKSSsKPAqXkvfpSeJzyEH3PREHZZ6SKsXLkDZbs3BSqwEdxqprqih4VzANK9",
	//	"12S6m2LpzN17jorYnLb2ApNKaV2EVeZtd6unvrPT1GH8yHGCyjYzKbywweQDZ7aAkhD31gutYAgfQizb2JhJTgBb3AJ8aB4hyppm2ax"}

	tool := new(debugtool.DebugTool).InitLocal("9334")
	//tool := new(debugtool.DebugTool).InitMainnet()

	//tool := new(debugtool.DebugTool).InitDevNet()
	//InitToken(tool, privateKeys[0], "something")

	//a := ListTokens(tool)
	//tokenID := a.Result.ListCustomToken[0].ID
	//ConvertTokenCoinVersion(tool, privateKeys[0], tokenID)

	//TransferToken(tool, privateKeys[0], privateKeys[1], tokenID,"1000")

	//GetBalanceToken(tool, privateKeys[0], tokenID)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your choice (arguments separated by ONLY ONE space) and hit ENTER: ")
		text, _ := reader.ReadString('\n')
		args := strings.Split(text[:len(text)-1], " ")
		if len(args) < 1 {
			return
		}
		if args[0] == "port" {
			tool = SwitchPort(args[1])
		}
		if args[0] == "inittestnet" {
			tool = new(debugtool.DebugTool).InitTestnet()
		}
		if args[0] == "initdevnet" {
			tool = new(debugtool.DebugTool).InitDevNet()
		}
		if args[0] == "initmainnet" {
			tool = new(debugtool.DebugTool).InitMainnet()
		}
		if args[0] == "initlocal" {
			tool = new(debugtool.DebugTool).InitLocal(args[1])
		}

		if args[0] == "convert" {
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			ConvertCoinVersion(tool, privateKeys[index])
		}
		if args[0] == "send" {
			sendTx(tool)
		}
		if args[0] == "outcoin" {
			var privateKey string
			if len(args[1]) < 10 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					panic(err)
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}
			bi := big.NewInt(0)
			if len(args) >= 3 {
				_, ok := bi.SetString(args[2], 10)
				if !ok {
					continue
				}
			}

			GetPRVOutPutCoin(tool, privateKey, bi.Uint64())
		}
		if args[0] == "outcoincached" {
			var privateKey string
			if len(args[1]) < 10 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					panic(err)
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}

			GetPRVOutPutCoinCached(tool, privateKey)
		}
		if args[0] == "balance" {
			var privateKey string
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				if index >= int64(len(privateKeys)) {
					fmt.Println("Cannot find the private key")
					continue
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}

			GetPRVBalance(tool, privateKey)
		}
		if args[0] == "mempool" {
			GetRawMempool(tool)
		}
		if args[0] == "txhash" {
			GetTxByHash(tool, args[1])
		}
		if args[0] == "staking" {
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			Staking(tool, privateKeys[index], privateSeeds[index])
		}
		if args[0] == "unstaking" {
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			UnStaking(tool, privateKeys[index], privateSeeds[index])
		}

		if args[0] == "reward" {
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			tokenID := "0000000000000000000000000000000000000000000000000000000000000004"
			if args[2] != "" && len(args[2]) > 0 {
				tokenID = args[2]
			}
			WithdrawReward(tool, privateKeys[index], tokenID)
		}
		if args[0] == "transfer" {
			var privateKey string
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				if index >= int64(len(privateKeys)) {
					fmt.Println("Cannot find the private key")
					continue
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}

			var paymentAddress string
			if len(args[2]) < 4 {
				index, err := strconv.ParseInt(args[2], 10, 32)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				if index >= int64(len(privateKeys)) {
					fmt.Println("Cannot find the private key")
					continue
				}
				paymentAddress = privateKeyToPaymentAddress(privateKeys[index])
			} else {
				paymentAddress = args[2]
			}

			TransferPRV(tool, privateKey, paymentAddress, args[3])
		}

		if args[0] == "payment" {
			fmt.Println("Payment Address", privateKeyToPaymentAddress(args[1]))
		}
		if args[0] == "public" {
			fmt.Println("Public Key", privateKeyToPublicKey(args[1]))
		}

		//TOKEN RPC
		if args[0] == "inittoken" {
			if len(args) < 3 {
				panic("Not enough params for initToken")
			}
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			InitToken(tool, privateKeys[index], args[2])
		}
		if args[0] == "listtoken" {
			ListTokens(tool)
		}
		if args[0] == "converttoken" {
			if len(args) < 3 {
				panic("Not enough params for converttoken")
			}
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			tokenID := args[2]
			ConvertTokenCoinVersion(tool, privateKeys[index], tokenID)
		}
		if args[0] == "transfertoken" {
			if len(args) < 5 {
				panic("Not enough params for transfertoken")
			}
			var privateKey string
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				if index >= int64(len(privateKeys)) {
					fmt.Println("Cannot find the private key")
					continue
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}

			paymentAddress := args[2]
			if len(args[2]) < 3 {
				index, err := strconv.ParseInt(args[2], 10, 32)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				if index >= int64(len(privateKeys)) {
					fmt.Println("Cannot find the private key")
					continue
				}
				paymentAddress = privateKeyToPaymentAddress(privateKeys[index])
			}

			tokenID := args[3]
			if len(args[3]) < 10 {
				tokenID = tokenIDs[args[3]]
			}
			TransferToken(tool, privateKey, paymentAddress, tokenID, args[4])

		}
		if args[0] == "balancetoken" {
			var privateKey string
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				if index >= int64(len(privateKeys)) {
					fmt.Println("Cannot find the private key")
					continue
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}

			var tokenID string
			if len(args[2]) < 10 {
				tokenID = tokenIDs[args[2]]
			} else {
				tokenID = args[2]
			}

			GetBalanceToken(tool, privateKey, tokenID)
		}
		if args[0] == "outtoken" {
			if len(args) < 2 {
				panic("Not enough param for outtoken")
			}
			var privateKey = args[1]
			if len(args[1]) < 10 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					panic(err)
				}
				privateKey = privateKeys[index]
			}
			tokenID := common.PRVIDStr
			if len(args) > 2 {
				tokenID = args[2]
				if len(args[2]) < 10 {
					tokenID = tokenIDs[args[2]]
				}
			}
			bi := big.NewInt(0)
			if len(args) >= 4 {
				_, ok := bi.SetString(args[3], 10)
				if !ok {
					continue
				}
			}

			GetOutputToken(tool, privateKey, tokenID, bi.Uint64())
		}
		if args[0] == "outtokencached" {
			if len(args) < 2 {
				panic("Not enough param for outtoken")
			}
			var privateKey = args[1]
			if len(args[1]) < 10 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					panic(err)
				}
				privateKey = privateKeys[index]
			}
			tokenID := common.PRVIDStr
			if len(args) > 2 {
				tokenID = args[2]
				if len(args[2]) < 10 {
					tokenID = tokenIDs[args[2]]
				}
			}

			GetOutputTokenCached(tool, privateKey, tokenID)
		}
		if args[0] == "uot" {
			if len(args) < 2 {
				fmt.Println("Not enough param for unspentouttoken")
				continue
			}
			tokenID := common.PRVIDStr
			if len(args) > 2 {
				tokenID = args[2]
				if len(args[2]) < 10 {
					tokenID = tokenIDs[args[2]] //Make sure you have the right token name
				}
			}

			var privateKey string
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					fmt.Println(err)
					continue
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}

			GetUnspentOutputToken(tool, privateKey, tokenID)
		}
		// PDE
		if args[0] == "pdecontributeprv" {
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			PDEContributePRV(tool, privateKeys[index], args[2])
		}
		if args[0] == "pdecontributetoken" {
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			PDEContributeToken(tool, privateKeys[index], args[2], args[3])
		}
		if args[0] == "pdewithdraw" {
			index, err := strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			tokenID1 := args[2]
			if len(args[2]) < 10 {
				tokenID1 = tokenIDs[args[2]]
			}

			tokenID2 := args[3]
			if len(args[3]) < 10 {
				tokenID2 = tokenIDs[args[3]]
			}

			PDEWithdrawContribution(tool, privateKeys[index], tokenID1, tokenID2, args[4])
		}

		if args[0] == "pdetradeprv" {
			if len(args) < 3 {
				fmt.Println("Not enough param for pdetradeprv")
				continue
			}

			privateKey := args[1]
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					panic(err)
				}
				privateKey = privateKeys[index]
			}

			tokenID := args[2]
			if len(args[2]) < 10 {
				tokenID = tokenIDs[args[2]]
			}

			PDETradePRV(tool, privateKey, tokenID, args[3])
		}

		if args[0] == "pdetradetoken" {
			if len(args) < 3 {
				fmt.Println("Not enough param for pdetradeprv")
				continue
			}

			privateKey := args[1]
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					panic(err)
				}
				privateKey = privateKeys[index]
			}

			tokenID := args[2]
			if len(args[2]) < 10 {
				tokenID = tokenIDs[args[2]]
			}
			PDETradeToken(tool, privateKey, tokenID, args[3])
		}

		if args[0] == "sub" {
			var privateKey string
			if len(args[1]) < 3 {
				index, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				if index >= int64(len(privateKeys)) {
					fmt.Println("Cannot find the private key")
					continue
				}
				privateKey = privateKeys[index]
			} else {
				privateKey = args[1]
			}

			SubmitOTAKey(tool, privateKey)
		}
		// if args[0] == "doublespend" {
		// 	indexFrom, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	indexTo, err := strconv.ParseInt(args[2], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	DoubleSpendPRV(tool, privateKeys[indexFrom], privateKeys[indexTo], args[3])
		// }
		// if args[0] == "dupinput" {
		// 	indexFrom, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	indexTo, err := strconv.ParseInt(args[2], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	DoCreateDuplicateInputTx(tool, privateKeys[indexFrom], privateKeys[indexTo], args[3])
		// }
		// if args[0] == "outgtin" {
		// 	indexFrom, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	indexTo, err := strconv.ParseInt(args[2], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	DoCreateOutGtInTx(tool, privateKeys[indexFrom], privateKeys[indexTo], args[3])
		// }
		// if args[0] == "recvexists" {
		// 	indexFrom, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	DoCreateReceiverExistsTx(tool, privateKeys[indexFrom], args[2])
		// }
		// if args[0] == "doublespendtoken"{
		// 	if len(args) < 5 {
		// 		panic("Not enough params for transfertoken")
		// 	}
		// 	indexFrom, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	indexTo, err := strconv.ParseInt(args[2], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	DoubleSpendToken(tool, privateKeys[indexFrom], privateKeys[indexTo], args[3], args[4])
		// }
		// if args[0] == "dupinputtoken"{
		// 	if len(args) < 5 {
		// 		panic("Not enough params for transfertoken")
		// 	}
		// 	indexFrom, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	indexTo, err := strconv.ParseInt(args[2], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	DoCreateDuplicateInputTokenTx(tool, privateKeys[indexFrom], privateKeys[indexTo], args[3], args[4])
		// }
		// if args[0] == "recvexiststoken"{
		// 	if len(args) < 4 {
		// 		panic("Not enough params for transfertoken")
		// 	}
		// 	indexFrom, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	DoCreateReceiverExistsTokenTx(tool, privateKeys[indexFrom], args[2], args[3])
		// }

		// if args[0] == "sendraw"{
		// 	/*
		// 	args[1] = 0/1 => non-privacy/privacy transaction
		// 	args[2]: 0 - tx without signature; 1 - tx with bulletproof tampered; 2 - tx with snProof tampered; 3 - tx with one-of-many proof tampered
		// 	args[3]: sender index
		// 	args[4]: receiver address (index or full paymentAddress)
		// 	args[5]: tokenID (optional)
		// 	*/
		// 	if len(args) < 5 {
		// 		fmt.Println("Need at least 5 arguments")
		// 		continue
		// 	}
		// 	flagPrivacy, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil{
		// 		fmt.Println(err)
		// 		continue
		// 	}

		// 	txType, err := strconv.ParseInt(args[2], 10, 32)
		// 	if err != nil{
		// 		fmt.Println(err)
		// 		continue
		// 	}

		// 	idxSender, err := strconv.ParseInt(args[3], 10, 32)
		// 	if err != nil{
		// 		fmt.Println(err)
		// 		continue
		// 	}

		// 	var paymentAddress string
		// 	if len(args[4]) < 3{
		// 		idxReceiver, err := strconv.ParseInt(args[4], 10, 32)
		// 		if err != nil{
		// 			fmt.Println(err)
		// 			continue
		// 		}
		// 		paymentAddress = privateKeyToPaymentAddress(privateKeys[idxReceiver])
		// 	}else{
		// 		paymentAddress = args[3]
		// 	}

		// 	tokenID := common.PRVIDStr
		// 	if len(args) > 5 {
		// 		tokenID = args[5]
		// 	}

		// 	if flagPrivacy == 0{
		// 		sendRawTxNoPrivacy(tool, privateKeys[idxSender], tokenID, paymentAddress, txType)
		// 	}else{
		// 		sendRawTxPrivacy(tool, privateKeys[idxSender], tokenID, paymentAddress, txType)
		// 	}
		// }

		// if args[0] == "sendrawtoken"{
		// 	/*
		// 		args[1] = 0/1 => non-privacy/privacy transaction
		// 		args[2]: 0 - tx without signature; 1 - tx with bulletproof tampered; 2 - tx with snProof tampered; 3 - tx with one-of-many proof tampered
		// 		args[3]: sender index
		// 		args[4]: receiver address (index or full paymentAddress)
		// 		args[5]: tokenID (optional)
		// 	*/
		// 	if len(args) < 5 {
		// 		fmt.Println("Need at least 5 arguments")
		// 		continue
		// 	}
		// 	flagPrivacy, err := strconv.ParseInt(args[1], 10, 32)
		// 	if err != nil{
		// 		fmt.Println(err)
		// 		continue
		// 	}

		// 	txType, err := strconv.ParseInt(args[2], 10, 32)
		// 	if err != nil{
		// 		fmt.Println(err)
		// 		continue
		// 	}

		// 	idxSender, err := strconv.ParseInt(args[3], 10, 32)
		// 	if err != nil{
		// 		fmt.Println(err)
		// 		continue
		// 	}

		// 	var paymentAddress string
		// 	if len(args[4]) < 3{
		// 		idxReceiver, err := strconv.ParseInt(args[4], 10, 32)
		// 		if err != nil{
		// 			fmt.Println(err)
		// 			continue
		// 		}
		// 		paymentAddress = privateKeyToPaymentAddress(privateKeys[idxReceiver])
		// 	}else{
		// 		paymentAddress = args[3]
		// 	}

		// 	tokenID := common.PRVIDStr
		// 	if len(args) > 5 {
		// 		tokenID = args[5]
		// 	}

		// 	if flagPrivacy == 0{
		// 		sendRawTxNoPrivacy(tool, privateKeys[idxSender], tokenID, paymentAddress, txType)
		// 	}else{
		// 		sendRawTxPrivacy(tool, privateKeys[idxSender], tokenID, paymentAddress, txType)
		// 	}
		// }

		if args[0] == "cmtandpubkey" {
			if len(args) < 3 {
				fmt.Println("Need at least 3 arguments")
				continue
			}

			//#1 - paymentAddress
			paymentAddress := args[1]
			if len(args[1]) < 2 {
				idx, err := strconv.ParseInt(args[1], 10, 32)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if idx > int64(len(privateKeys)-1) {
					fmt.Println("Privatekey index out of range")
					continue
				}
				paymentAddress = privateKeyToPaymentAddress(privateKeyToPaymentAddress(privateKeys[idx]))
			}

			//#2 - numOutputs
			numOutputs, err := strconv.ParseInt(args[2], 10, 32)
			if err != nil {
				fmt.Println(err)
				continue
			}

			//#3- TokenID
			tokenID := common.PRVIDStr
			if len(args) > 3 {
				tokenID = args[3]
			}

			GetListRandomCommitmentsAndPublicKeys(tool, paymentAddress, tokenID, int(numOutputs))
		}

		if args[0] == "genkeyset" {
			privateKey, payment, _ := GenKeySet([]byte(args[1]))
			fmt.Println(privateKey, payment)
		}

		if args[0] == "testeqdl" {
			for i:=0;i<100;i++{
				x := privacy.RandomScalar()
				g := privacy.RandomPoint()
				h := privacy.RandomPoint()

				a := new(privacy.Point).ScalarMult(g, x)
				b := new(privacy.Point).ScalarMult(h, x)

				eqdlWitness := debugtool.NewEQDLWitness(x, g, h, a, b)


				msg := operation.RandomScalar().ToBytesS()
				pi := eqdlWitness.Prove(msg)

				//For testing purpose only, will be expanded to cover more cases
				isModified := (common.RandInt() % 5) == 1
				if isModified{
					msg = operation.RandomScalar().ToBytesS()
				}

				isValid, err := pi.Verify(msg, g, h, a, b)

				fmt.Printf("Step %d, isModified: %v, isValid: %v, err: %v\n", i, isModified, isValid, err)
			}

		}

		if args[0] == "testvrf" {
			for i:=0;i<100;i++{
				x := privacy.RandomScalar()
				g := privacy.RandomPoint()
				pubKey := new(privacy.Point).ScalarMult(g, x)


				vrfWitness := debugtool.NewVRFWitness(x, g)


				msg := operation.RandomScalar().ToBytesS()
				y, pi, _ := vrfWitness.Compute(msg)

				//For testing purpose only, will be expanded to cover more cases
				isModified := (common.RandInt() % 5) == 1
				if isModified{
					msg = operation.RandomScalar().ToBytesS()
				}

				isValid, err := pi.Verify(msg, g, pubKey, y)

				fmt.Printf("Step %d, output: %d, isModified: %v, isValid: %v, err: %v\n", i, y.ToUint64Little(), isModified, isValid, err)
			}

		}
	}
}
