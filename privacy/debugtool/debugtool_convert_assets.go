package debugtool

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/wallet"
	"time"
)

type Account struct {
	PrivateKey string
	Address string
}

var TestAccounts []*Account

func GenTestAccount(seed string, numberAccounts int) error {
	if len(seed) == 0 {
		seed = "hello world!"
	}
	TestAccounts = make([]*Account, numberAccounts)
	for i := 0; i < numberAccounts; i++ {
		keyWallet, err := wallet.NewMasterKey([]byte(fmt.Sprintf("%s-%v", seed, i)))
		if err != nil {
			fmt.Errorf("cannot create master key. Error %v\n", err)
			return err
		}
		TestAccounts[i] = &Account{
			PrivateKey: keyWallet.Base58CheckSerialize(wallet.PriKeyType),
			Address: keyWallet.Base58CheckSerialize(wallet.PaymentAddressType),
		}
	}
	return nil
}

func InitPrv(tool *DebugTool) error {
	b, err := tool.CreateAndSendTransaction()
	fmt.Println(string(b))
	return err
}

func SendPrv2TestAccounts(tool *DebugTool, value uint64)  {
	groupSize := 20
	for i:= 0; i< int(len(TestAccounts) / groupSize); i++ {
		receivers := make(map[string]uint64)
		for j := 0; j < groupSize; j++ {
			receivers[TestAccounts[i*groupSize +j].Address] = value
		}
		paymentInfo, _ := json.Marshal(receivers)
		query := fmt.Sprintf(`{
			"jsonrpc": "1.0",
			"method": "createandsendtransaction",
			"params": [
				"112t8roafGgHL1rhAP9632Yef3sx5k8xgp8cwK4MCJsCL1UWcxXvpzg97N4dwvcD735iKf31Q2ZgrAvKfVjeSUEvnzKJyyJD3GqqSZdxN4or",
				%v,
				1,
				1
			],
			"id": 1
		}`, string(paymentInfo))
		fmt.Println(query)
		res, err := tool.SendPostRequestWithQuery(query)
		fmt.Println(string(res), err)
		time.Sleep(10*time.Second)
	}
}