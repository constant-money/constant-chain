package instruction

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

var key1 = "121VhftSAygpEJZ6i9jGkEKLMQTKTiiHzeUfeuhpQCcLZtys8FazpWwytpHebkAwgCxvqgUUF13fcSMtp5dgV1YkbRMj3z42TW2EebzAaiGg2DkGPodckN2UsbqhVDibpMgJUHVkLXardemfLdgUqWGtymdxaaRyPM38BAZcLpo2pAjxKv5vG5Uh9zHMkn7ZHtdNHmBmhG8B46UeiGBXYTwhyMe9KGS83jCMPAoUwHhTEXj5qQh6586dHjVxwEkRzp7SKn9iG1FFWdJ97xEkP2ezAapNQ46quVrMggcHFvoZofs1xdd4o5vAmPKnPTZtGTKunFiTWGnpSG9L6r5QpcmapqvRrK5SiuFhNM5DqgzUeHBb7fTfoiWd2N29jkbTGSq8CPUSjx3zdLR9sZguvPdnAA8g25cFPGSZt8aEnFJoPRzM"
var key2 = "121VhftSAygpEJZ6i9jGkEqPGAXcmKffwMbzpwxnEfzJxen4oZKPukWAUBbqvV5xPnowZ2eQmAj2mEebG2oexebQPh1MPFC6vEZAk6i7AiRPrZmfaRrRVrBp4WXnVJmL3xK4wzTfkR2rZkhUmSZm112TTyhDNkDQSaBGJkexrPbryqUygazCA2eyo6LnK5qs7jz2RhhsWqUTQ3sQJUuFcYdf2pSnYwhqZqphDCSRizDHeysaua5L7LwS8fY7KZHhPgTuFjvUWWnWSRTmV8u1dTY5kcmMdDZsPiyN9WfqjgVoTFNALjFG8U4GMvzV3kKwVVjuPMsM2XqyPDVpdNQUgLnv2bJS8Tr22A9NgF1FQfWyAny1DYyY3N5H3tfCggsybzZXzrbYPPgokvEynac91y8hPkRdgKW1e7FHzuBnEisPuKzy"
var key3 = "121VhftSAygpEJZ6i9jGkGLcYhJBeaJTGY5aFjqQA2WwyxU69Utrviuy9AJ3ATkeEyigVGScQUZw22cD1HeFKiyASYAs82WEamujt3nefYA9FPhURBpRTn6jDmGKUdb4QNbs7HVCJkRRaL9aktg1yaQaZE8TJFg2UeE9tBqUdmvD8fy36aDCYM5W86jaTVCXeEJQWPxUunP2EEL3e283PJ8zqPeBkpoFvkvhB28Hk3oRDeCCTC7QhbaV18ayKeToYqAxoUMBBihanfA33ixeX1daeKpajLCgDZ6jrfphwdYwQbf7dMcZ2NVvQ1a5JUCTJUZypwgKRt8tnTAKCowt2L1KNGP4NJJZm61cfHAGbKRyG9QxCJgK2SdMKsKPVefZSc9LbVaB7VeBby5LHxvMoCD7bN7g1HYRp4BX9n1fZJUeEkVa"
var key4 = "121VhftSAygpEJZ6i9jGkDjJj7e2cfgQvrLsPsmLhGMmGD9U9Knffa1MZAw79EijnpueVfTStN2VYt5jRqEr2DTjVqzUinwHVKWH4Tg4szHUntiBdWeqzNC4E8iiwC9Y2KtcRr3hBkpfqvyuBvchigatrigRvFVWu8H2RQqjvopLL51DQ4LFD87L9Zgj9HhasMeyr6f37yirs47JgtGs4BM7EhhpM5zD3TCsFabPphtwDKnfuLMaGzoAw5fM8zEXvdLMuohk96oayjdYothncdtZom17DxB1Mmw535eEjxBwz9ELoZRKk3LYiheSd4xGN9QsxrT2WnZCTd8B5QktARte5S91QYvRMixKC8UEuovQhXt8jMZNkq7CmMeXoybfYdmNaAHuqbY1QeUT2AgaqPho4ay3z5eeKRhnB28H18RGWQ1L"

var txHash1 = "d368a4c2c199aa949b00c0d634b0d4a54853ab27c1ee21b3dfa08e33b395cd63"
var txHash2 = "6a71a0c1f382492739614831866417900f888d23203cb8fd1ee0ad40e7b0b59e"
var txHash3 = "dcef9d320572067b933364481b7669f2311b2d94bd83d585d32dc9252371704a"
var txHash4 = "3b0e8bb7693c9a1313b4e1e0983a397739e69b7b4e82abfe8842f66b4d83b24b"

var paymentAddress1 = "12Rtc3sbfHTTSqmS8efnhgb7Rc6ineoQCwJyX63MMRK4HF6JGo51GJp5rk25QfviU7GPjyptT9q3JguQmDEG3uKpPUDEY5CSUJtttfU"
var paymentAddress2 = "12RuC7GdG1P89A5KjgrStACHjQj79Ka5fZNREbawP7rxfYQWzxqe3Yuq1saN6zAVJquZKKUJf9ZfddmKPJEb8ZZFWHDRnx6VBqRyuAd"
var paymentAddress3 = "12RvMjD8h6L68j7AjM6stP5qwuZvFCBnQtXCdtKCypNxqgpT3yZ11d5upFz6y5autbmLdK3ip2UyyyjmVizzgj8ChRVsujiFeagcyVM"
var paymentAddress4 = "12Rt5Zm9a3DxxmTPoui7aNkpcsV2yLyimRBjunXpocu8ky1pm7o8mn9EJBkpGNaSvGnWMpPPLABhqixUBAznjnWCAR9c8WKKJfsgq4J"

var incKey1, incKey2, incKey3, incKey4 *incognitokey.CommitteePublicKey
var incTxHash1, incTxHash2, incTxHash3, incTxHash4 *common.Hash
var incPaymentAddress1, incPaymentAddress2, incPaymentAddress3, incPaymentAddress4 *privacy.PaymentAddress

func initTxHash() {
	var err error
	incTxHash1, err = common.Hash{}.NewHashFromStr(txHash1)
	if err != nil {
		panic(err)
	}
	incTxHash2, err = common.Hash{}.NewHashFromStr(txHash2)
	if err != nil {
		panic(err)
	}
	incTxHash3, err = common.Hash{}.NewHashFromStr(txHash3)
	if err != nil {
		panic(err)
	}
	incTxHash4, err = common.Hash{}.NewHashFromStr(txHash4)
	if err != nil {
		panic(err)
	}
}

//initPublicKey init incognito public key for testing by base 58 string
func initPublicKey() {
	incKey1 = new(incognitokey.CommitteePublicKey)
	incKey2 = new(incognitokey.CommitteePublicKey)
	incKey3 = new(incognitokey.CommitteePublicKey)
	incKey4 = new(incognitokey.CommitteePublicKey)

	err := incKey1.FromBase58(key1)
	if err != nil {
		panic(err)
	}

	err = incKey2.FromBase58(key2)
	if err != nil {
		panic(err)
	}

	err = incKey3.FromBase58(key3)
	if err != nil {
		panic(err)
	}

	err = incKey4.FromBase58(key4)
	if err != nil {
		panic(err)
	}
}

func initPaymentAddress() {
	incPaymentAddress1 = new(privacy.PaymentAddress)
	incPaymentAddress2 = new(privacy.PaymentAddress)
	incPaymentAddress3 = new(privacy.PaymentAddress)
	incPaymentAddress4 = new(privacy.PaymentAddress)

	wl, _ := wallet.Base58CheckDeserialize(paymentAddress1)
	*incPaymentAddress1 = wl.KeySet.PaymentAddress

	w2, _ := wallet.Base58CheckDeserialize(paymentAddress2)
	*incPaymentAddress2 = w2.KeySet.PaymentAddress

	w3, _ := wallet.Base58CheckDeserialize(paymentAddress3)
	*incPaymentAddress3 = w3.KeySet.PaymentAddress

	w4, _ := wallet.Base58CheckDeserialize(paymentAddress4)
	*incPaymentAddress4 = w4.KeySet.PaymentAddress
}
