package mempool

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/databasemp"
	"github.com/incognitochain/incognito-chain/incdb"
	_ "github.com/incognitochain/incognito-chain/incdb/lvdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/stretchr/testify/assert"
)

var (
	db               incdb.Database
	dbp              databasemp.DatabaseInterface
	bc               *blockchain.BlockChain
	pbMempool        = pubsub.NewPubSubManager()
	tp               = &TxPool{}
	feeEstimator     = make(map[byte]*FeeEstimator)
	cPendingTxs      = make(chan metadata.Transaction, 1000)
	cRemoveTxs       = make(chan metadata.Transaction, 1000)
	privateKeyShard0 = []string{
		"112t8rnYXEN3RubmoYcHhxpfmeM7X733n49nVbLMFvzesvL26QUiZpEH42sxYNSg2e7ovYDXiZ28wezDMrEBFgjeYs13rKRwNBaJRfEFH7Xj",
		"112t8rna6GNo5uGGkevGwgYgEFxeESDqPsY3zFjFBscaVadvsWqCWcU3r4jhVdZUnetEWuXGJB4jKybqdJvxVULjWG1qELMPuqR28J54TRUU",
		"112t8rnaE7Mwp3ydP3bLUFjgw4RR92dLRb5bZUAm9D4Cmb8nDy13fo1nZPTzk8Muyr4M6LHD3rLMpgUqCXvrFq2fAfgFDyWfUUXDXegBxyz9",
		"112t8rndvGZoqvhyiyrBNqsNoYPbYvouvbErABRgj2YKEt8d7kKkwCAMJaWbM1LtWJtmAdqN6KcUekRFqbp11jExb1ex4ToH5hp9XcsBi3YZ",
		"112t8rndVbFQpRHNbcq6Dd7ggMcDkZsKmxcU5smoTDQMLGAn4Uh81gEQ41XWuCKeamMcDfaivXHbkgHqqzPHtY69ah4tDKofERiWaPNpiY1m",
		"112t8rnikzAix9cimXJ5SJ2Cv8FrHjFjrprHjG412v17gvuGLHvDLT3rJeccrrTGgXz83Xq9HuHBsGt3sMxEW8kkLXfCU9pddcXBpADjvFjo",
		"112t8rnkkKBJWU3UnBhaz2HueYtrdudHnRpaQDMawy1m1Sxg7CuLqzKMpFiVsrPAUzZFkgoP3ZSEzFN9CwtdobBbQ44L8vo6du73m51Ekixi",
		"112t8rnmYmbvf4cVPS3FuBSUy39yPuono7jEx124bUsZ6YH848eAbjfmvCCD5JXwp4wM24wtnAwkFDcSJMjg4cY2a9b9f3NanKfBbe1tCqbH",
		"112t8rnmkcf4fANohyLkLRiPTSZr5YXpfrhZkthiEh66Ly8kA3mSeJyUBArzMXD3CVbXLajKUidMypgSUmtbbsRvMohVPubFVQnfxTfJPt1M",
		"112t8rnmz7yKZC853NR7fiUxpds2LRNHmWzaMEzsWevjaaHFsun335R9L5Hf3DibHkzSjUv2o6SrMRdmrLr9rMSWbxnUdZ9EXXvh3myxSGrF",
	}
	miningSeedShard0 = []string{
		"12N3BfpMP18LxLLTcGDEKEpVrciuuFqniVbovDqXweyH2tysx2Z",
		"1KrNte7w4Wy4igBehAZpH63KSpAwgLkR4YcGo6xzCKTP34jDxu",
		"1Vs1hubckf1NoGzmVANZT7QPjo3HyCXB6UWx722uhE285LrvGD",
		"1ywGtB7MASWBSWYrdvLFkMQFq6NLcbNY7w7jtY45Grvve16TQj",
		"12QQCaNNEBk3z8gumUvxFeFmLGLQgyUuAwUY4z5rLQuLL8amri1",
		"12oYwtqXsy3aKGQ3ewNTUEs3FmfwhFrHHhR2ajjkP1UPKZ2azCX",
		"1PLo6quoqNW754vPocS7jgwxjH8vbHoCpviAMrXSGp5JRUWJBF",
		"1dzj2DxqgQphjPcy3bc4QM3mivJ25essyZfqer9S1wdEu8AoXV",
		"12ocMcMRmamq82Vi7Z7wsP56U6svnbKJVZKP8KN8MnenMTMkAZC",
		"19N5JLGtxvXAZhR9ra8oTRGMXPh5noR7EZoUWUjgRRxQgshfhT",
	}
	/*publicKeyShard0 = []string{
		"185azaQSWjk4WUhVKavjdbvzLNaFRQH9BABLVh7LVZsMEAQbE2b",
		"15P9cpSnqFDHZXdZ5gA7poUuWwNH9JnMbu8yvTLWJypsryYR7Zk",
		"15Fepa7m851vTema3fPQTbyiuxa289dLfL73tDuJYYLCd9hssmf",
		"17odhKmHJhy537BzSPyLH7f2u6TAadnoko6MdHzpC1xvzVTJo87",
		"151vzKx6AaQs8Jw5Q8PefGSPu3E16w2E2tSRXd1tEyM1qUA4H1r",
		"173mJJm5Yuw3VDPeZhvwC4GALgQon1ZucTi5MRgtXLuKRg3MEPL",
		"18NBXkSazuSk9C5gTwafehZ1L8LeXwCLpNHNaiatYXwVNGGH36t",
		"15SwUShJ8MkG7rzAckV2KR4kFUWmDeZf99rU5bE2RKuAPYhMxeR",
		"161VJhDDWsyM6ATC8cnmPjqPEc4kYk1suKcwdLNzr3MVTQrPjBd",
		"18JBEghYX65CWTZLdnw4TPEz4S2hNgYgyxfaep3oruq6UTakrJV",
	}*/
	stakingPublicKey = "121VhftSAygpEJZ6i9jGkLm73VBfy9sVFpajamtqFcTu1NAE6ok2Cym9q4z5X8VXchEhsxcRcppLCmbAdhTdNNNaLySwFa82noFgf8fkv4HrFkZuXUdVM5Ut4RsBiWQ2kUHUsiyEac7HV3Kx19w1MoEUHMsffJDEyMe96cEEzUPFKNq2qLQ5DFZvnkZDiDga8rTBb2UWFnf6Y6QuEw8X18oCbzJFqBMbbpr8USTXxW56ruYFvY5LhvsPPYjqbH2vtnBzqErD786kb1FSrGYyXPz7KXqTKS6B3tuMwmN6Uzs4kHbfZqhnxsX5sqGk6J5KpzaKuHywLHmZmUtzPUKsGTjADCQXxKhydbgYJkFLRt8qJaTm5EjV8WSQ6yTPDs2AsAS7jG8P1sNnKtrEbA2bfY688d1snZwKFncLswgSgPjzTvHE"
	// public key of 112t8rndVbFQpRHNbcq6Dd7ggMcDkZsKmxcU5smoTDQMLGAn4Uh81gEQ41XWuCKeamMcDfaivXHbkgHqqzPHtY69ah4tDKofERiWaPNpiY1m at 4

	receiverPaymentAddress1 = "12RsQ3xTPfebpiomuGbBgFFHMDaPSnmtbvEkipTBiSj6qmQ8ya5BMKUY28oQaSht7vjzrTk4TvZDwwve1eqEFXeW29FgAZFXU9PRkNU"
	receiverPaymentAddress2 = "12S4TkmzjyV2vdQuaj5WxxUjJ4epXjkXmAvGoYiiPSfHHe5JS2GPTbkA9TpYq6aCpM1BnoQQY4pMp7xD3W5eMHjTy7XoztFxEPaqHpD"
	tokenID                 = "1a871b83b0724955e0f9331eea059f0e7c83c44985088b561835cf5add4a3810"
	normalTokenID           = "275ef76d02bf242d5da17baba45a9b6c9bd4c9211738800f2d4ffe938c8fa525"
	maxAmount               = 1750000000000 * 4
	normalTranferAmount     = 50
	commonFee               = int64(10)
	higherFee               = int64(math.Round(float64(commonFee)*defaultReplaceFeeRatio)) + 1
	lowerFee                = int64(math.Round(float64(commonFee)/defaultReplaceFeeRatio - 2))
	noFee                   = int64(0)
	defaultTokenFee         = float64(5)
	defaultTokenParams      = make(map[string]interface{})
	defaultTokenReceiver    = make(map[string]interface{})
)
var _ = func() (_ struct{}) {
	go pbMempool.Start()
	for i := 0; i < 255; i++ {
		shardID := byte(i)
		feeEstimator[shardID] = NewFeeEstimator(
			DefaultEstimateFeeMaxRollback,
			DefaultEstimateFeeMinRegisteredBlocks,
			1)
	}
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_")
	if err != nil {
		log.Fatalf("failed to create temp dir: %+v", err)
	}
	log.Println(dbPath)
	db, err = incdb.Open("leveldb", filepath.Join("./", dbPath))
	if err != nil {
		log.Fatal("Could not open database connection", err)
	}
	dbPath2, err := ioutil.TempDir(os.TempDir(), "test_")
	if err != nil {
		log.Fatalf("failed to create temp dir: %+v", err)
	}
	log.Println(dbPath2)
	dbp, err = databasemp.Open("leveldbmempool", filepath.Join("./", dbPath2))
	if err != nil {
		log.Fatal("Could not open persist database connection", err)
	}
	bc = blockchain.NewBlockChain(&blockchain.Config{
		DataBase:      db,
		PubSubManager: pbMempool,
		ChainParams:   &blockchain.ChainTestParam,
		MemCache:      memcache.New(),
	}, true)
	bc.BestState = &blockchain.BestState{
		Beacon: &blockchain.BeaconBestState{},
		Shard:  make(map[byte]*blockchain.ShardBestState),
	}
	for i := 0; i < 255; i++ {
		shardID := byte(i)
		bc.BestState.Shard[shardID] = &blockchain.ShardBestState{
			BestBlock: &types.ShardBlock{
				Header: types.ShardHeader{
					Height: 1,
				},
			},
		}
	}
	if err != nil {
		panic("Could not init blockchain")
	}
	tp.Init(&Config{
		DataBase:          db,
		DataBaseMempool:   dbp,
		BlockChain:        bc,
		PubSubManager:     pbMempool,
		IsLoadFromMempool: false,
		PersistMempool:    false,
		FeeEstimator:      feeEstimator,
		ChainParams:       &blockchain.ChainTestParam,
	})
	tp.CPendingTxs = nil
	tp.CRemoveTxs = nil
	var transactions []metadata.Transaction
	for _, privateKey := range privateKeyShard0 {
		txs := initTx(strconv.Itoa(maxAmount), privateKey, db)
		transactions = append(transactions, txs...)
	}
	err = tp.config.BlockChain.CreateAndSaveTxViewPointFromBlock(&types.ShardBlock{
		Header: types.ShardHeader{ShardID: 0},
		Body: types.ShardBody{
			Transactions: transactions,
		},
	}, &[]incdb.BatchData{})
	transactions = []metadata.Transaction{}
	for _, privateKey := range privateKeyShard0 {
		txs := initTx(strconv.Itoa(maxAmount), privateKey, db)
		transactions = append(transactions, txs...)
	}
	err = tp.config.BlockChain.CreateAndSaveTxViewPointFromBlock(&types.ShardBlock{
		Header: types.ShardHeader{ShardID: 0},
		Body: types.ShardBody{
			Transactions: transactions,
		},
	}, &[]incdb.BatchData{})
	if err != nil {
		fmt.Println("Can not fetch transaction")
		return
	}
	defaultTokenParams["TokenID"] = ""
	defaultTokenParams["TokenName"] = "ABCD123"
	defaultTokenParams["TokenSymbol"] = "ABCDF123"
	defaultTokenParams["TokenAmount"] = float64(1000)
	defaultTokenParams["TokenTxType"] = float64(0)
	defaultTokenReceiver[receiverPaymentAddress1] = float64(1000)
	defaultTokenParams["TokenReceivers"] = defaultTokenReceiver
	defaultTokenParams["TokenFee"] = defaultTokenFee
	// token id custom token: 1a871b83b0724955e0f9331eea059f0e7c83c44985088b561835cf5add4a3810
	Logger.Init(common.NewBackend(nil).Logger("test", true))
	privacy.Logger.Init(common.NewBackend(nil).Logger("test", true))
	transaction.Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

func ResetMempoolTest() {
	tp.pool = make(map[common.Hash]*TxDesc)
	tp.poolSerialNumbersHashList = make(map[common.Hash][]common.Hash)
	tp.poolSerialNumberHash = make(map[common.Hash]common.Hash)
	tp.poolCandidate = make(map[common.Hash]string)
	tp.duplicateTxs = make(map[common.Hash]uint64)
	tp.RoleInCommittees = -1
	tp.IsBlockGenStarted = false
	tp.IsUnlockMempool = false
	_, subChanRole, _ := tp.config.PubSubManager.RegisterNewSubscriber(pubsub.ShardRoleTopic)
	tp.config.RoleInCommitteesEvent = subChanRole
	tp.IsTest = false
	tp.CPendingTxs = cPendingTxs
	tp.CRemoveTxs = cRemoveTxs
	tp.config.DataBaseMempool.Reset()
}
func initTx(amount string, privateKey string, db incdb.Database) []metadata.Transaction {
	var initTxs []metadata.Transaction
	var initAmount, _ = strconv.Atoi(amount) // amount init
	testUserkeyList := []string{
		privateKey,
	}
	for _, val := range testUserkeyList {
		testUserKey, _ := wallet.Base58CheckDeserialize(val)
		testUserKey.KeySet.InitFromPrivateKey(&testUserKey.KeySet.PrivateKey)
		testSalaryTX := transaction.Tx{}
		testSalaryTX.InitTxSalary(uint64(initAmount), &testUserKey.KeySet.PaymentAddress, &testUserKey.KeySet.PrivateKey,
			db,
			nil,
		)
		initTxs = append(initTxs, &testSalaryTX)
	}
	return initTxs
}

// chooseBestOutCoinsToSpent returns list of unspent coins for spending with amount
func chooseBestOutCoinsToSpent(outCoins []*privacy.OutputCoin, amount uint64) (resultOutputCoins []*privacy.OutputCoin, remainOutputCoins []*privacy.OutputCoin, totalResultOutputCoinAmount uint64, err error) {
	resultOutputCoins = make([]*privacy.OutputCoin, 0)
	remainOutputCoins = make([]*privacy.OutputCoin, 0)
	totalResultOutputCoinAmount = uint64(0)

	// either take the smallest coins, or a single largest one
	var outCoinOverLimit *privacy.OutputCoin
	outCoinsUnderLimit := make([]*privacy.OutputCoin, 0)

	for _, outCoin := range outCoins {
		if outCoin.CoinDetails.GetValue() < amount {
			outCoinsUnderLimit = append(outCoinsUnderLimit, outCoin)
		} else if outCoinOverLimit == nil {
			outCoinOverLimit = outCoin
		} else if outCoinOverLimit.CoinDetails.GetValue() > outCoin.CoinDetails.GetValue() {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		} else {
			remainOutputCoins = append(remainOutputCoins, outCoinOverLimit)
			outCoinOverLimit = outCoin
		}
	}

	sort.Slice(outCoinsUnderLimit, func(i, j int) bool {
		return outCoinsUnderLimit[i].CoinDetails.GetValue() < outCoinsUnderLimit[j].CoinDetails.GetValue()
	})

	for _, outCoin := range outCoinsUnderLimit {
		if totalResultOutputCoinAmount < amount {
			totalResultOutputCoinAmount += outCoin.CoinDetails.GetValue()
			resultOutputCoins = append(resultOutputCoins, outCoin)
		} else {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}

	if outCoinOverLimit != nil && (outCoinOverLimit.CoinDetails.GetValue() > 2*amount || totalResultOutputCoinAmount < amount) {
		remainOutputCoins = append(remainOutputCoins, resultOutputCoins...)
		resultOutputCoins = []*privacy.OutputCoin{outCoinOverLimit}
		totalResultOutputCoinAmount = outCoinOverLimit.CoinDetails.GetValue()
	} else if outCoinOverLimit != nil {
		remainOutputCoins = append(remainOutputCoins, outCoinOverLimit)
	}

	if totalResultOutputCoinAmount < amount {
		return resultOutputCoins, remainOutputCoins, totalResultOutputCoinAmount, errors.New("Not enough coin")
	} else {
		return resultOutputCoins, remainOutputCoins, totalResultOutputCoinAmount, nil
	}
}
func CreateAndSaveTestNormalTransaction(privateKey string, fee int64, hasPrivacyCoin bool, amount int) metadata.Transaction {
	// get sender key set from private key
	senderKeySet, _ := wallet.Base58CheckDeserialize(privateKey)
	senderKeySet.KeySet.InitFromPrivateKey(&senderKeySet.KeySet.PrivateKey)
	lastByte := senderKeySet.KeySet.PaymentAddress.Pk[len(senderKeySet.KeySet.PaymentAddress.Pk)-1]
	shardIDSender := common.GetShardIDFromLastByte(lastByte)
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	receiversPaymentAddressStrParam := make(map[string]interface{})
	receiversPaymentAddressStrParam[receiverPaymentAddress2] = amount
	paymentInfos := make([]*privacy.PaymentInfo, 0)
	for paymentAddressStr, amount := range receiversPaymentAddressStrParam {
		keyWalletReceiver, _ := wallet.Base58CheckDeserialize(paymentAddressStr)
		paymentInfo := &privacy.PaymentInfo{
			Amount:         uint64(amount.(int)),
			PaymentAddress: keyWalletReceiver.KeySet.PaymentAddress,
		}
		paymentInfos = append(paymentInfos, paymentInfo)
	}
	estimateFeeCoinPerKb := fee
	totalAmmount := uint64(0)
	for _, receiver := range paymentInfos {
		totalAmmount += receiver.Amount
	}

	outCoins, err := tp.config.BlockChain.GetListOutputCoinsByKeyset(&senderKeySet.KeySet, shardIDSender, prvCoinID)
	if err != nil {
		fmt.Println("Can't create transaction", err)
		return nil
	}
	remainOutputCoins := make([]*privacy.OutputCoin, 0)
	for _, outCoin := range outCoins {
		if tp.ValidateSerialNumberHashH(outCoin.CoinDetails.GetSerialNumber().ToBytesS()) == nil {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	if len(outCoins) == 0 && totalAmmount > 0 {
		fmt.Println("Can't create transaction")
		return nil
	}
	candidateOutputCoins, outCoins, candidateOutputCoinAmount, err := chooseBestOutCoinsToSpent(outCoins, totalAmmount)
	if err != nil {
		fmt.Println("Can't create transaction", err)
		return nil
	}

	estimateTxSizeInKb := transaction.EstimateTxSize(transaction.NewEstimateTxSizeParam(len(candidateOutputCoins), len(paymentInfos), hasPrivacyCoin, nil, nil, 0))
	realFee := uint64(estimateFeeCoinPerKb) * uint64(estimateTxSizeInKb)
	needToPayFee := int64((totalAmmount + realFee) - candidateOutputCoinAmount)
	// if not enough to pay fee
	if needToPayFee > 0 {
		if len(outCoins) > 0 {
			candidateOutputCoinsForFee, _, _, err := chooseBestOutCoinsToSpent(outCoins, uint64(needToPayFee))
			if err != nil {
				fmt.Println("Can't create transaction", err)
				return nil
			}
			candidateOutputCoins = append(candidateOutputCoins, candidateOutputCoinsForFee...)
		}
	}
	// convert to inputcoins
	inputCoins := transaction.ConvertOutputCoinToInputCoin(candidateOutputCoins)
	tx := transaction.Tx{}
	err1 := tx.Init(
		transaction.NewTxPrivacyInitParams(&senderKeySet.KeySet.PrivateKey,
			paymentInfos,
			inputCoins,
			realFee,
			hasPrivacyCoin,
			db,
			nil, // use for prv coin -> nil is valid
			nil,
			[]byte{}))
	if err1 != nil {
		panic("no tx found")
	}
	return &tx
}
func CreateAndSaveTestStakingTransaction(privateKey string, privateSeed string, fee int64, isBeacon bool) metadata.Transaction {
	// get sender key set from private key
	hasPrivacyCoin := false
	senderKeySet, _ := wallet.Base58CheckDeserialize(privateKey)
	senderKeySet.KeySet.InitFromPrivateKey(&senderKeySet.KeySet.PrivateKey)
	lastByte := senderKeySet.KeySet.PaymentAddress.Pk[len(senderKeySet.KeySet.PaymentAddress.Pk)-1]
	shardIDSender := common.GetShardIDFromLastByte(lastByte)

	receiversPaymentAddressStrParam := make(map[string]interface{})
	if isBeacon {
		receiversPaymentAddressStrParam[common.BurningAddress] = tp.config.ChainParams.StakingAmountShard * 3
	} else {
		receiversPaymentAddressStrParam[common.BurningAddress] = tp.config.ChainParams.StakingAmountShard
	}
	paymentInfos := make([]*privacy.PaymentInfo, 0)
	for paymentAddressStr, amount := range receiversPaymentAddressStrParam {
		keyWalletReceiver, _ := wallet.Base58CheckDeserialize(paymentAddressStr)
		paymentInfo := &privacy.PaymentInfo{
			Amount:         amount.(uint64),
			PaymentAddress: keyWalletReceiver.KeySet.PaymentAddress,
		}
		paymentInfos = append(paymentInfos, paymentInfo)
	}
	estimateFeeCoinPerKb := fee
	totalAmmount := uint64(0)
	for _, receiver := range paymentInfos {
		totalAmmount += receiver.Amount
	}
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	outCoins, err := tp.config.BlockChain.GetListOutputCoinsByKeyset(&senderKeySet.KeySet, shardIDSender, prvCoinID)
	if err != nil {
		fmt.Println("Can't create transaction", err)
		return nil
	}
	remainOutputCoins := make([]*privacy.OutputCoin, 0)
	for _, outCoin := range outCoins {
		if tp.ValidateSerialNumberHashH(outCoin.CoinDetails.GetSerialNumber().ToBytesS()) == nil {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	if len(outCoins) == 0 && totalAmmount > 0 {
		fmt.Println("Can't create transaction")
		return nil
	}
	candidateOutputCoins, outCoins, candidateOutputCoinAmount, err := chooseBestOutCoinsToSpent(outCoins, totalAmmount)
	if err != nil {
		fmt.Println("Can't create transaction", err)
		return nil
	}
	paymentAddress, _ := senderKeySet.Serialize(wallet.PaymentAddressType)

	privateSeedBytes, ver, errDecode := base58.Base58Check{}.Decode(privateSeed)
	if (errDecode != nil) || (ver != common.ZeroByte) {
		return nil
	}
	committeePK, err := incognitokey.NewCommitteeKeyFromSeed(privateSeedBytes, senderKeySet.KeySet.PaymentAddress.Pk)
	if err != nil {
		return nil
	}
	committeePKBytes, err := committeePK.Bytes()
	if err != nil {
		return nil
	}
	committeePKBase58 := base58.Base58Check{}.Encode(committeePKBytes, common.ZeroByte)
	var stakingMetadata *metadata.StakingMetadata
	if isBeacon {
		stakingMetadata, _ = metadata.NewStakingMetadata(64, base58.Base58Check{}.Encode(paymentAddress, common.ZeroByte), base58.Base58Check{}.Encode(paymentAddress, common.ZeroByte), tp.config.ChainParams.StakingAmountShard, committeePKBase58, true)
	} else {
		stakingMetadata, _ = metadata.NewStakingMetadata(63, base58.Base58Check{}.Encode(paymentAddress, common.ZeroByte), base58.Base58Check{}.Encode(paymentAddress, common.ZeroByte), tp.config.ChainParams.StakingAmountShard, committeePKBase58, true)
	}
	estimateTxSizeInKb := transaction.EstimateTxSize(transaction.NewEstimateTxSizeParam(len(candidateOutputCoins), len(paymentInfos), hasPrivacyCoin, stakingMetadata, nil, 0))
	realFee := uint64(estimateFeeCoinPerKb) * uint64(estimateTxSizeInKb)
	needToPayFee := int64((totalAmmount + realFee) - candidateOutputCoinAmount)
	// if not enough to pay fee
	if needToPayFee > 0 {
		if len(outCoins) > 0 {
			candidateOutputCoinsForFee, _, _, err := chooseBestOutCoinsToSpent(outCoins, uint64(needToPayFee))
			if err != nil {
				fmt.Println("Can't create transaction", err)
				return nil
			}
			candidateOutputCoins = append(candidateOutputCoins, candidateOutputCoinsForFee...)
		}
	}
	// convert to inputcoins
	inputCoins := transaction.ConvertOutputCoinToInputCoin(candidateOutputCoins)
	tx := transaction.Tx{}
	err1 := tx.Init(
		transaction.NewTxPrivacyInitParams(&senderKeySet.KeySet.PrivateKey,
			paymentInfos,
			inputCoins,
			realFee,
			hasPrivacyCoin,
			db,
			nil, // use for prv coin -> nil is valid
			stakingMetadata,
			[]byte{}))
	if err1 != nil {
		panic("no tx found")
	}
	return &tx
}
func CreateAndSaveTestInitCustomTokenTransactionPrivacy(privateKey string, fee int64, tokenParamsRaw map[string]interface{}, hasPrivacyCoin bool) metadata.Transaction {
	// get sender key set from private key
	senderKeySet, _ := wallet.Base58CheckDeserialize(privateKey)
	senderKeySet.KeySet.InitFromPrivateKey(&senderKeySet.KeySet.PrivateKey)
	lastByte := senderKeySet.KeySet.PaymentAddress.Pk[len(senderKeySet.KeySet.PaymentAddress.Pk)-1]
	shardIDSender := common.GetShardIDFromLastByte(lastByte)

	receiversPaymentAddressStrParam := make(map[string]interface{})
	receiversPaymentAddressStrParam[receiverPaymentAddress2] = 50
	paymentInfos := make([]*privacy.PaymentInfo, 0)
	for paymentAddressStr, amount := range receiversPaymentAddressStrParam {
		keyWalletReceiver, _ := wallet.Base58CheckDeserialize(paymentAddressStr)
		paymentInfo := &privacy.PaymentInfo{
			Amount:         uint64(amount.(int)),
			PaymentAddress: keyWalletReceiver.KeySet.PaymentAddress,
		}
		paymentInfos = append(paymentInfos, paymentInfo)
	}
	estimateFeeCoinPerKb := fee
	totalAmmount := uint64(0)
	for _, receiver := range paymentInfos {
		totalAmmount += receiver.Amount
	}
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	outCoins, err := tp.config.BlockChain.GetListOutputCoinsByKeyset(&senderKeySet.KeySet, shardIDSender, prvCoinID)
	if err != nil {
		fmt.Println("Can't create transaction", err)
		return nil
	}
	remainOutputCoins := make([]*privacy.OutputCoin, 0)
	for _, outCoin := range outCoins {
		if tp.ValidateSerialNumberHashH(outCoin.CoinDetails.GetSerialNumber().ToBytesS()) == nil {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	if len(outCoins) == 0 && totalAmmount > 0 {
		fmt.Println("Can't create transaction")
		return nil
	}
	candidateOutputCoins, outCoins, candidateOutputCoinAmount, err := chooseBestOutCoinsToSpent(outCoins, totalAmmount)
	if err != nil {
		fmt.Println("Can't create transaction", err)
		return nil
	}
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID:     tokenParamsRaw["TokenID"].(string),
		PropertyName:   tokenParamsRaw["TokenName"].(string),
		PropertySymbol: tokenParamsRaw["TokenSymbol"].(string),
		TokenTxType:    int(tokenParamsRaw["TokenTxType"].(float64)),
		Amount:         uint64(tokenParamsRaw["TokenAmount"].(float64)),
		TokenInput:     nil,
		Fee:            uint64(tokenParamsRaw["TokenFee"].(float64)),
	}
	tokenParams.Receiver, _, _ = transaction.CreateCustomTokenPrivacyReceiverArray(tokenParamsRaw["TokenReceivers"])
	estimateTxSizeInKb := transaction.EstimateTxSize(transaction.NewEstimateTxSizeParam(len(candidateOutputCoins), len(paymentInfos), hasPrivacyCoin, nil, tokenParams, 0))
	realFee := uint64(estimateFeeCoinPerKb) * uint64(estimateTxSizeInKb)
	needToPayFee := int64((totalAmmount + realFee) - candidateOutputCoinAmount)
	// if not enough to pay fee
	if needToPayFee > 0 {
		if len(outCoins) > 0 {
			candidateOutputCoinsForFee, _, _, err := chooseBestOutCoinsToSpent(outCoins, uint64(needToPayFee))
			if err != nil {
				fmt.Println("Can't create transaction", err)
				return nil
			}
			candidateOutputCoins = append(candidateOutputCoins, candidateOutputCoinsForFee...)
		}
	}
	// convert to inputcoins
	inputCoins := transaction.ConvertOutputCoinToInputCoin(candidateOutputCoins)
	tx := &transaction.TxCustomTokenPrivacy{}
	err1 := tx.Init(
		transaction.NewTxPrivacyTokenInitParams(&senderKeySet.KeySet.PrivateKey,
			nil,
			inputCoins,
			realFee,
			tokenParams,
			db,
			nil,
			hasPrivacyCoin,
			true,
			shardIDSender,
			[]byte{}))
	fmt.Println(tx.TxPrivacyTokenData.PropertyID.String())
	if err1 != nil {
		panic("no tx found")
	}
	return tx
}
func TestTxPoolStart(t *testing.T) {
	ResetMempoolTest()
	cQuit := make(chan struct{})
	go tp.Start(cQuit)
	if tp.RoleInCommittees != -1 {
		t.Fatal("Expect role is -1 but get ", tp.RoleInCommittees)
	}
	go tp.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.ShardRoleTopic, int(0)))
	now := time.Now()
	for {
		if tp.RoleInCommittees == 0 {
			close(cQuit)
			return
		}
		<-time.Tick(100 * time.Millisecond)
		if time.Since(now).Seconds() > time.Duration(10*time.Second).Seconds() {
			t.Fatal("Fail to get role from pubsub")
		}
	}

}
func TestTxPoolCheckRelayShard(t *testing.T) {
	ResetMempoolTest()
	tp.config.RelayShards = []byte{}
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], 10, false, normalTranferAmount)
	if isOK := tp.checkRelayShard(tx1); isOK {
		t.Fatalf("Expect false but get true")
	}
	tp.config.RelayShards = []byte{0, 1}
	if isOK := tp.checkRelayShard(tx1); !isOK {
		t.Fatalf("Expect true but get false")
	}
	tp.config.RelayShards = []byte{1, 0}
	if isOK := tp.checkRelayShard(tx1); !isOK {
		t.Fatalf("Expect true but get false")
	}
	tp.config.RelayShards = []byte{0}
	if isOK := tp.checkRelayShard(tx1); !isOK {
		t.Fatalf("Expect true but get false")
	}
}
func TestTxPoolCheckPublicKeyRole(t *testing.T) {
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], 10, false, normalTranferAmount)
	tp.RoleInCommittees = -1
	if isOK := tp.checkPublicKeyRole(tx1); isOK {
		t.Fatalf("Expect false but get true")
	}
	tp.RoleInCommittees = 1
	if isOK := tp.checkPublicKeyRole(tx1); isOK {
		t.Fatalf("Expect false but get true")
	}
	tp.RoleInCommittees = 0
	if isOK := tp.checkPublicKeyRole(tx1); !isOK {
		t.Fatalf("Expect true but get false")
	}

}
func TestTxPoolInitChannelMempool(t *testing.T) {
	tp.CPendingTxs = nil
	tp.CRemoveTxs = nil
	if tp.CPendingTxs != nil && tp.CRemoveTxs != nil {
		t.Fatal("Expect nil channel but get", tp.CPendingTxs, tp.CRemoveTxs)
	} else {
		tp.InitChannelMempool(cPendingTxs, cRemoveTxs)
		if tp.CPendingTxs == nil {
			t.Fatalf("Expect %+v channel but get nil", tp.CPendingTxs)
		}
		if tp.CRemoveTxs == nil {
			t.Fatalf("Expect %+v channel but get nil", tp.CRemoveTxs)
		}
	}
}
func TestTxPoolGetTxsInMem(t *testing.T) {
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], commonFee, false, normalTranferAmount)
	tx2 := CreateAndSaveTestNormalTransaction(privateKeyShard0[1], commonFee, false, normalTranferAmount)
	tx3 := CreateAndSaveTestNormalTransaction(privateKeyShard0[2], commonFee, false, normalTranferAmount)
	txDesc1 := createTxDescMempool(tx1, 1, uint64(commonFee), 0)
	txDesc2 := createTxDescMempool(tx2, 1, uint64(commonFee), 0)
	txDesc3 := createTxDescMempool(tx3, 1, uint64(commonFee), 0)
	tp.pool[*tx1.Hash()] = txDesc1
	tp.pool[*tx2.Hash()] = txDesc2
	tp.pool[*tx3.Hash()] = txDesc3
	txs := tp.GetTxsInMem()
	if len(txs) != 3 {
		t.Fatalf("Expect 3 transaction from mempool but get %+v", len(txs))
	}
}
func TestTxPoolGetSerialNumbersHashH(t *testing.T) {
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], commonFee, false, normalTranferAmount)
	tx2 := CreateAndSaveTestNormalTransaction(privateKeyShard0[1], commonFee, false, normalTranferAmount)
	tx3 := CreateAndSaveTestNormalTransaction(privateKeyShard0[2], commonFee, false, normalTranferAmount)
	tp.poolSerialNumbersHashList[*tx1.Hash()] = tx1.ListSerialNumbersHashH()
	tp.poolSerialNumbersHashList[*tx2.Hash()] = tx2.ListSerialNumbersHashH()
	tp.poolSerialNumbersHashList[*tx3.Hash()] = tx3.ListSerialNumbersHashH()
	serialNumberList := tp.GetSerialNumbersHashH()
	if !reflect.DeepEqual(serialNumberList, tp.poolSerialNumbersHashList) {
		t.Fatalf("Something wrong with return serial list")
	}
}
func TestTxPoolIsTxInPool(t *testing.T) {
	ResetMempoolTest()
	tx := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], commonFee, false, normalTranferAmount)
	if tp.isTxInPool(tx.Hash()) {
		t.Fatalf("Expect %+v to be NOT in pool", *tx.Hash())
	}
	txDesc := createTxDescMempool(tx, 1, uint64(commonFee), 0)
	tp.pool[*tx.Hash()] = txDesc
	tp.poolSerialNumbersHashList[*tx.Hash()] = tx.ListSerialNumbersHashH()
	if !tp.isTxInPool(tx.Hash()) {
		t.Fatalf("Expect %+v to be in pool", *tx.Hash())
	}
}
func TestTxPoolAddTx(t *testing.T) {
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], commonFee, false, normalTranferAmount)
	tx2 := CreateAndSaveTestNormalTransaction(privateKeyShard0[1], commonFee, false, normalTranferAmount)
	tx3 := CreateAndSaveTestNormalTransaction(privateKeyShard0[2], commonFee, false, normalTranferAmount)
	txDesc1 := createTxDescMempool(tx1, 1, 10, 0)
	txDesc2 := createTxDescMempool(tx2, 1, 10, 0)
	txDesc3 := createTxDescMempool(tx3, 1, 10, 0)
	txStakingShard := CreateAndSaveTestStakingTransaction(privateKeyShard0[4], miningSeedShard0[4], commonFee, false)
	txStakingBeacon := CreateAndSaveTestStakingTransaction(privateKeyShard0[4], miningSeedShard0[4], commonFee, true)
	tx6 := CreateAndSaveTestNormalTransaction(privateKeyShard0[5], commonFee, true, 50)
	tp.addTx(txDesc1, false)
	tp.addTx(txDesc2, false)
	tp.addTx(txDesc3, false)
	if len(tp.pool) != 3 {
		t.Fatalf("Expect 3 transaction from mempool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 3 {
		t.Fatalf("Expect 3 transaction from mempool but get %+v", len(tp.poolSerialNumbersHashList))
	}
	tp.addTx(createTxDescMempool(tx6, 1, 10, 10), true)
	tp.addTx(createTxDescMempool(txStakingShard, 1, 10, 10), false)
	if len(tp.pool) != 5 {
		t.Fatalf("Expect 5 transaction from mempool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 5 {
		t.Fatalf("Expect 5 transaction from mempool but get %+v", len(tp.poolSerialNumbersHashList))
	}
	for _, v := range tp.poolCandidate {
		candidate := incognitokey.CommitteePublicKey{}
		candidate.FromString(v)
		_ = candidate
	}
	for _, v := range tp.poolCandidate {
		candidate := incognitokey.CommitteePublicKey{}
		candidate.FromString(v)
		_ = candidate
	}
	if common.IndexOfStrInHashMap(stakingPublicKey, tp.poolCandidate) < 0 {
		t.Fatalf("Expect %+v in pool but get %+v", stakingPublicKey, tp.poolCandidate)
	}
	if len(tp.poolCandidate) != 1 {
		t.Fatalf("Expect 1 but get %+v", len(tp.poolCandidate))
	}
	ResetMempoolTest()
	tp.addTx(txDesc1, true)
	tp.addTx(txDesc2, true)
	tp.addTx(txDesc3, true)
	if len(tp.pool) != 3 {
		t.Fatalf("Expect 3 transaction from mempool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 3 {
		t.Fatalf("Expect 3 transaction from mempool but get %+v", len(tp.poolSerialNumbersHashList))
	}
	tp.addTx(createTxDescMempool(tx6, 1, 10, 10), true)
	tp.addTx(createTxDescMempool(txStakingBeacon, 1, 10, 10), true)
	if len(tp.pool) != 5 {
		t.Fatalf("Expect 6 transaction from mempool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 5 {
		t.Fatalf("Expect 6 transaction from mempool but get %+v", len(tp.poolSerialNumbersHashList))
	}
	if common.IndexOfStrInHashMap(stakingPublicKey, tp.poolCandidate) < 0 {
		t.Fatalf("Expect %+v in pool but get %+v", stakingPublicKey, tp.poolCandidate)
	}
	if len(tp.poolCandidate) != 1 {
		t.Fatalf("Expect 1 but get %+v", len(tp.poolCandidate))
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx1.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx1.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx2.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx2.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx3.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx3.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx6.Hash()); !isOk && err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx6.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(txStakingBeacon.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", txStakingBeacon.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(txStakingShard.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", txStakingShard.Hash())
	}
}
func TestTxPoolValidateTransaction(t *testing.T) {
	ResetMempoolTest()
	senderKeySet, _ := wallet.Base58CheckDeserialize(privateKeyShard0[0])
	senderKeySet.KeySet.InitFromPrivateKey(&senderKeySet.KeySet.PrivateKey)
	lastByte := senderKeySet.KeySet.PaymentAddress.Pk[len(senderKeySet.KeySet.PaymentAddress.Pk)-1]
	shardIDSender := common.GetShardIDFromLastByte(lastByte)
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	sum := uint64(0)
	outCoins, _ := tp.config.BlockChain.GetListOutputCoinsByKeyset(&senderKeySet.KeySet, shardIDSender, prvCoinID)
	for _, outCoin := range outCoins {
		hash := common.HashH(outCoin.CoinDetails.GetSerialNumber().ToBytesS())
		log.Println("Serial Number: ", hash)
		log.Println("Serial Number value: ", outCoin.CoinDetails.GetValue())
		sum += outCoin.CoinDetails.GetValue()
	}
	log.Println("Sum:", sum)
	salaryTx := initTx("100", privateKeyShard0[0], db)
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], commonFee, false, maxAmount)
	tx1Replace := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], higherFee, false, maxAmount)
	tx1DoubleSpend := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], lowerFee, false, 1)
	// get sender key set from private key
	tx1ReplaceFailed := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], lowerFee, false, maxAmount)
	txInitCustomTokenPrivacy := CreateAndSaveTestInitCustomTokenTransactionPrivacy(privateKeyShard0[0], commonFee, defaultTokenParams, false)
	txInitCustomTokenPrivacyReplace := CreateAndSaveTestInitCustomTokenTransactionPrivacy(privateKeyShard0[0], higherFee, defaultTokenParams, false)
	txInitCustomTokenPrivacyReplaceFailed := CreateAndSaveTestInitCustomTokenTransactionPrivacy(privateKeyShard0[0], lowerFee, defaultTokenParams, false)
	tx2 := CreateAndSaveTestNormalTransaction(privateKeyShard0[1], commonFee, false, normalTranferAmount)
	tx3 := CreateAndSaveTestNormalTransaction(privateKeyShard0[2], commonFee, false, normalTranferAmount)
	tx4 := CreateAndSaveTestNormalTransaction(privateKeyShard0[3], noFee, false, normalTranferAmount)
	tx5 := CreateAndSaveTestNormalTransaction(privateKeyShard0[4], commonFee, false, normalTranferAmount)
	txStakingShard := CreateAndSaveTestStakingTransaction(privateKeyShard0[4], miningSeedShard0[4], commonFee, false)
	//txStakingBeacon := CreateAndSaveTestStakingTransaction(privateKeyShard0[4], miningSeedShard0[4], commonFee, true)
	txDesc1 := createTxDescMempool(tx1, 1, tx1.GetTxFee(), tx1.GetTxFeeToken())
	txDesc1CustomTokenPrivacy := createTxDescMempool(txInitCustomTokenPrivacy, 1, txInitCustomTokenPrivacy.GetTxFee(), txInitCustomTokenPrivacy.GetTxFeeToken())
	// Check condition 1: Sanity - Max version error
	ResetMempoolTest()
	tx1.(*transaction.Tx).Version = 2
	err1 := tp.validateTransaction(tx1, 0)
	if err1 == nil {
		t.Fatal("Expect max version error error but no error")
	} else {
		if err1.(*MempoolTxError).Code != ErrCodeMessage[RejectSanityTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err1)
		}
	}
	tx1.(*transaction.Tx).Version = 1
	// Check condition 1: Size - Invalid size error
	ResetMempoolTest()
	common.MaxTxSize = 0
	common.MaxBlockSize = 2000
	err2 := tp.validateTransaction(tx2, 0)
	if err2 == nil {
		t.Fatal("Expect size error error but no error")
	} else {
		if err2.(*MempoolTxError).Code != ErrCodeMessage[RejectSanityTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err2)
		}
	}
	common.MaxTxSize = 100
	common.MaxBlockSize = 2000
	// Check Condition 1: Sanity Validate type
	ResetMempoolTest()
	tx3.(*transaction.Tx).Type = "abc"
	err3 := tp.validateTransaction(tx3, 0)
	if err3 == nil {
		t.Fatal("Expect type error error but no error")
	} else {
		if err3.(*MempoolTxError).Code != ErrCodeMessage[RejectSanityTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err3)
		}
	}
	tx3.(*transaction.Tx).Type = common.TxNormalType
	// Check Condition 1: Sanity Validate type
	ResetMempoolTest()
	tempLockTime := tx4.(*transaction.Tx).LockTime
	tx4.(*transaction.Tx).LockTime = time.Now().Unix() + 1000000
	err4 := tp.validateTransaction(tx4, 0)
	if err4 == nil {
		t.Fatal("Expect type error error but no error")
	} else {
		if err4.(*MempoolTxError).Code != ErrCodeMessage[RejectSanityTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err4)
		}
	}
	tx4.(*transaction.Tx).LockTime = tempLockTime
	// Check Condition 1: Sanity Validate Info Length
	ResetMempoolTest()
	tempByte := []byte{}
	for i := 0; i < 514; i++ {
		tempByte = append(tempByte, byte(i))
	}
	tx4.(*transaction.Tx).Info = tempByte
	err5 := tp.validateTransaction(tx4, 0)
	if err5 == nil {
		t.Fatal("Expect type error error but no error")
	} else {
		if err5.(*MempoolTxError).Code != ErrCodeMessage[RejectSanityTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err5)
		}
	}
	tx4.(*transaction.Tx).Info = []byte{}
	// Check condition 2: tx exist in pool
	tp.pool[*tx1.Hash()] = txDesc1
	tp.poolSerialNumbersHashList[*tx1.Hash()] = tx1.ListSerialNumbersHashH()
	err6 := tp.validateTransaction(tx1, 0)
	if err6 == nil {
		t.Fatal("Expect reject duplicate error but no error")
	} else {
		if err6.(*MempoolTxError).Code != ErrCodeMessage[RejectDuplicateTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectDuplicateTx], err6)
		}
	}
	// Check Condition 3: Salary Transaction
	ResetMempoolTest()
	err7 := tp.validateTransaction(salaryTx[0], 0)
	if err7 == nil {
		t.Fatal("Expect salary error error but no error")
	} else {
		if err7.(*MempoolTxError).Code != ErrCodeMessage[RejectSalaryTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSalaryTx], err7)
		}
	}
	// Check Condition 4: Validate fee
	ResetMempoolTest()
	err8 := tp.validateTransaction(tx4, 0)
	if err8 == nil {
		t.Fatal("Expect fee error error but no error")
	} else {
		if err8.(*MempoolTxError).Code != ErrCodeMessage[RejectInvalidFee].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectInvalidFee], err8)
		}
	}
	tx5.(*transaction.Tx).Type = common.TxNormalType
	// Check Condition 5: replace (normal tx)
	ResetMempoolTest()
	tp.addTx(txDesc1, false)
	err9 := tp.validateTransaction(tx1Replace, 0)
	if err9 != nil {
		t.Fatal("Expect no error error but get ", err9)
	}
	// Check Condition 5: Check replace with mempool (normal tx)
	ResetMempoolTest()
	tp.addTx(txDesc1, false)
	err91 := tp.validateTransaction(tx1ReplaceFailed, 0)
	if err91 == nil {
		t.Fatal("Expect replace fail error in mempool error error but no error")
	} else {
		if err91.(*MempoolTxError).Code != ErrCodeMessage[RejectReplacementTxError].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectReplacementTxError], err91)
		}
	}
	// Check Condition 5: replace (custom token privacy tx)
	ResetMempoolTest()
	tp.addTx(txDesc1CustomTokenPrivacy, false)
	err92 := tp.validateTransaction(txInitCustomTokenPrivacyReplace, 0)
	if err92 != nil {
		t.Fatal("Expect no error error but get ", err92)
	}
	// Check Condition 5: Check replace with mempool (custom token privacy tx)
	ResetMempoolTest()
	tp.addTx(txDesc1CustomTokenPrivacy, false)
	err93 := tp.validateTransaction(txInitCustomTokenPrivacyReplaceFailed, 0)
	if err93 == nil {
		t.Fatal("Expect replace fail error in mempool error error but no error")
	} else {
		if err93.(*MempoolTxError).Code != ErrCodeMessage[RejectReplacementTxError].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectReplacementTxError], err93)
		}
	}
	// Check Condition 5: Check double spend with mempool
	ResetMempoolTest()
	tp.addTx(txDesc1, false)
	log.Println("Tx 1 Serial Number Hash:", tx1.ListSerialNumbersHashH())
	log.Println("Tx 1 replaced Number Hash:", tx1Replace.ListSerialNumbersHashH())
	log.Println("Tx 1 replaced failed Number Hash:", tx1ReplaceFailed.ListSerialNumbersHashH())
	log.Println("Tx 1 double spend Serial Number Hash:", tx1DoubleSpend.ListSerialNumbersHashH())
	err10 := tp.validateTransaction(tx1DoubleSpend, 0)
	if err10 == nil {
		t.Fatal("Expect double spend error in mempool error error but no error")
	} else {
		if err10.(*MempoolTxError).Code != ErrCodeMessage[RejectDoubleSpendWithMempoolTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectDoubleSpendWithMempoolTx], err10)
		}
	}
	// check Condition 6: validate by it self
	ResetMempoolTest()
	err = tp.config.BlockChain.CreateAndSaveTxViewPointFromBlock(&types.ShardBlock{
		Header: types.ShardHeader{ShardID: 0},
		Body: types.ShardBody{
			Transactions: []metadata.Transaction{tx1},
		},
	}, &[]incdb.BatchData{})
	if err != nil {
		t.Fatalf("Expect no error but get %+v", err)
	}
	// snd existed
	err11 := tp.validateTransaction(tx1, 0)
	if err11 == nil {
		t.Fatal("Expect double spend with blockchain error error but no error")
	} else {
		if err11.(*MempoolTxError).Code != ErrCodeMessage[RejectInvalidTx].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectInvalidTx], err11)
		}
	}
	// check Condition 7: Check double spend with blockchain
	// check Condition 9: Check Init Custom Token
	ResetMempoolTest()
	tp.poolCandidate[*txStakingShard.Hash()] = stakingPublicKey
	err13 := tp.validateTransaction(txStakingShard, 0)
	if err13 == nil {
		t.Fatal("Expect duplicate staking pubkey error error but no error")
	} else {
		if err13.(*MempoolTxError).Code != ErrCodeMessage[RejectDuplicateStakePubkey].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectDuplicateStakePubkey], err)
		}
	}
	err13 = tp.validateTransaction(txStakingShard, 0)
	if err13 == nil {
		t.Fatal("Expect duplicate staking pubkey error error but no error")
	} else {
		if err13.(*MempoolTxError).Code != ErrCodeMessage[RejectDuplicateStakePubkey].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectDuplicateStakePubkey], err)
		}
	}
	ResetMempoolTest()
	// Pass all case
	err14 := tp.validateTransaction(txStakingShard, 0)
	if err14 != nil {
		t.Fatal("Expect no err but get ", err14)
	}
	err14 = tp.validateTransaction(tx3, 0)
	if err14 != nil {
		t.Fatal("Expect no err but get ", err14)
	}
}
func TestTxPoolmayBeAcceptTransaction(t *testing.T) {
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], commonFee, false, normalTranferAmount)
	tx2 := CreateAndSaveTestNormalTransaction(privateKeyShard0[1], commonFee, false, normalTranferAmount)
	tx3 := CreateAndSaveTestNormalTransaction(privateKeyShard0[2], commonFee, false, normalTranferAmount)
	txStakingBeacon := CreateAndSaveTestStakingTransaction(privateKeyShard0[4], miningSeedShard0[4], commonFee, true)
	tx6 := CreateAndSaveTestNormalTransaction(privateKeyShard0[5], commonFee, true, 50)
	_, _, err1 := tp.maybeAcceptTransaction(tx1, false, true, 0)
	if err1 != nil {
		t.Fatal("Expect no error but get ", err1)
	}
	_, _, err2 := tp.maybeAcceptTransaction(tx2, false, true, 0)
	if err2 != nil {
		t.Fatal("Expect no error but get ", err2)
	}
	_, _, err3 := tp.maybeAcceptTransaction(tx3, false, true, 0)
	if err3 != nil {
		t.Fatal("Expect no error but get ", err3)
	}
	/* can not stake beacon
	_, _, err5 := tp.maybeAcceptTransaction(txStakingBeacon, false, true)
	if err5 != nil {
		t.Fatal("Expect no error but get ", err5)
	}*/
	_, _, err6 := tp.maybeAcceptTransaction(tx6, false, true, 0)
	if err6 != nil {
		t.Fatal("Expect no error but get ", err6)
	}
	if len(tp.pool) != 4 {
		t.Fatalf("Expect 4 transaction from mempool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 4 {
		t.Fatalf("Expect 4 transaction from mempool but get %+v", len(tp.poolSerialNumbersHashList))
	}
	/*if common.IndexOfStrInHashMap(stakingPublicKey, tp.poolCandidate) < 0 {
		t.Fatalf("Expect %+v in pool but get %+v", stakingPublicKey, tp.poolCandidate)
	}
	if len(tp.poolCandidate) != 1 {
		t.Fatalf("Expect 1 but get %+v", len(tp.poolCandidate))
	}*/
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx1.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx1.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx2.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx2.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx3.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx3.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx6.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx6.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(txStakingBeacon.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", txStakingBeacon.Hash())
	}
	// persist mempool
	ResetMempoolTest()
	tp.maybeAcceptTransaction(tx1, true, true, 0)
	tp.maybeAcceptTransaction(tx2, true, true, 0)
	tp.maybeAcceptTransaction(tx3, true, true, 0)
	tp.maybeAcceptTransaction(txStakingBeacon, true, true, 0)
	tp.maybeAcceptTransaction(tx6, true, true, 0)
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx1.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx1.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx2.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx2.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx3.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx3.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx6.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", tx6.Hash())
	}
	/*if isOk, err := tp.config.DataBaseMempool.HasTransaction(txStakingBeacon.Hash()); !isOk || err != nil {
		t.Fatalf("Expect tx hash %+v in database mempool but counter err", txStakingBeacon.Hash())
	}*/

	tx1Data, err := tp.getTransactionFromDatabaseMempool(tx1.Hash())
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, tx1Data)
	assert.Equal(t, tx1.Hash(), tx1Data.Desc.Tx.Hash())

	tx1Data, err = tp.getTransactionFromDatabaseMempool(&common.Hash{})
	assert.NotEqual(t, nil, err)

	err = tp.removeTransactionFromDatabaseMP(tx1.Hash())
	assert.Equal(t, nil, err)
	isOk, err := tp.config.DataBaseMempool.HasTransaction(tx1.Hash())
	assert.Equal(t, nil, err)
	assert.Equal(t, false, isOk)

	tp.config.TxLifeTime = 100000000000
	listTx, err := tp.loadDatabaseMP()
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(listTx))

	tp.config.IsLoadFromMempool = false
	err = tp.LoadOrResetDatabaseMempool()
	assert.Equal(t, nil, err)

	tp.config.IsLoadFromMempool = true
	err = tp.LoadOrResetDatabaseMempool()
	assert.Equal(t, nil, err)

	list := tp.ListTxs()
	assert.Equal(t, 4, len(list))

	c := tp.Count()
	assert.Equal(t, 4, c)

	has := tp.HaveTransaction(tx1.Hash())
	assert.Equal(t, true, has)

	max := tp.Size()
	assert.NotEqual(t, 0, max)

	fee := tp.MaxFee()
	assert.Equal(t, uint64(30), uint64(fee))

	/*tp.LockPool()
	tp.UnlockPool()*/

	pool := tp.GetPool()
	assert.NotEqual(t, nil, pool)

	mining := tp.MiningDescs()
	assert.NotEqual(t, nil, mining)
	assert.Equal(t, 4, len(mining))

	tx1Temp, err := tp.GetTx(tx1.Hash())
	assert.Equal(t, nil, err)
	assert.Equal(t, tx1.Hash(), tx1Temp.Hash())

	tp.removeTx(tx1)
	_, err = tp.GetTx(tx1.Hash())
	assert.NotEqual(t, nil, err)
}
func TestTxPoolRemoveTx(t *testing.T) {
	// no persist mempool
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], 10, false, normalTranferAmount)
	tx2 := CreateAndSaveTestNormalTransaction(privateKeyShard0[1], 10, false, normalTranferAmount)
	tx3 := CreateAndSaveTestNormalTransaction(privateKeyShard0[2], 10, false, normalTranferAmount)
	txStakingBeacon := CreateAndSaveTestStakingTransaction(privateKeyShard0[4], miningSeedShard0[4], commonFee, true)
	tx6 := CreateAndSaveTestNormalTransaction(privateKeyShard0[5], commonFee, true, 50)
	txs := []metadata.Transaction{tx1, tx2, tx3, txStakingBeacon, tx6}
	tp.maybeAcceptTransaction(tx1, false, true, 0)
	tp.maybeAcceptTransaction(tx2, false, true, 0)
	tp.maybeAcceptTransaction(tx3, false, true, 0)
	tp.maybeAcceptTransaction(txStakingBeacon, false, true, 0) // this is fail because can not stake beacon now
	tp.maybeAcceptTransaction(tx6, false, true, 0)
	if len(tp.pool) != 4 {
		t.Fatalf("Expect 4 transaction from pool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 4 {
		t.Fatalf("Expect 4 transaction from poolSerialNumbersHashList but get %+v", len(tp.poolSerialNumbersHashList))
	}
	if len(tp.poolSerialNumberHash) != 4 {
		t.Fatalf("Expect 4 transaction from poolSerialNumberHash but get %+v", len(tp.poolSerialNumberHash))
	}
	if common.IndexOfStrInHashMap(stakingPublicKey, tp.poolCandidate) >= 0 { // because can not stake to beacon
		t.Fatalf("Expect %+v in pool but get %+v", stakingPublicKey, tp.poolCandidate)
	}
	if len(tp.poolCandidate) != 0 { // because can not stake beacon
		t.Fatalf("Expect 0 but get %+v", len(tp.poolCandidate))
	}
	tp.RemoveTx(txs, true)
	if len(tp.pool) != 0 {
		t.Fatalf("Expect 0 transaction from mempool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 0 {
		t.Fatalf("Expect 0 transaction from mempool but get %+v", len(tp.poolSerialNumbersHashList))
	}
	if len(tp.poolSerialNumberHash) != 0 {
		t.Fatalf("Expect 0 transaction from mempool but get %+v", len(tp.poolSerialNumberHash))
	}
	if common.IndexOfStrInHashMap(stakingPublicKey, tp.poolCandidate) >= 0 {
		t.Fatalf("Expect %+v in pool but get %+v", stakingPublicKey, tp.poolCandidate)
	}
	if len(tp.poolCandidate) != 0 { // beacause can not stake to beacon
		t.Fatalf("Expect 0 but get %+v", len(tp.poolCandidate))
	}
	tp.RemoveCandidateList([]string{stakingPublicKey})
	if len(tp.poolCandidate) != 0 {
		t.Fatalf("Expect 0 but get %+v", len(tp.poolCandidate))
	}
	if common.IndexOfStrInHashMap(stakingPublicKey, tp.poolCandidate) > 0 {
		t.Fatalf("Expect %+v NOT in pool but get %+v", stakingPublicKey, tp.poolCandidate)
	}
	// no persist mempool
	ResetMempoolTest()
	tp.config.PersistMempool = true
	tp.maybeAcceptTransaction(tx1, true, true, 0)
	tp.maybeAcceptTransaction(tx2, true, true, 0)
	tp.maybeAcceptTransaction(tx3, true, true, 0)
	tp.maybeAcceptTransaction(txStakingBeacon, true, true, 0)
	tp.maybeAcceptTransaction(tx6, true, true, 0)
	tp.RemoveTx(txs, true)
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx1.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx1.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx2.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx2.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx3.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx3.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(tx6.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", tx6.Hash())
	}
	if isOk, err := tp.config.DataBaseMempool.HasTransaction(txStakingBeacon.Hash()); isOk && err == nil {
		t.Fatalf("Expect tx hash %+v NOT in database mempool but counter err", txStakingBeacon.Hash())
	}
}
func TestTxPoolMaybeAcceptTransaction(t *testing.T) {
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], 10, false, normalTranferAmount)
	// test relay shard and role in committeess
	tp.config.RelayShards = []byte{}
	tp.RoleInCommittees = -1
	_, _, err1 := tp.MaybeAcceptTransaction(tx1, 0)
	if err1 == nil {
		t.Fatal("Expect unexpected transaction error error but no error")
	} else {
		if err1.(*MempoolTxError).Code != ErrCodeMessage[UnexpectedTransactionError].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err)
		}
	}
	// test size of mempool
	tp.config.RelayShards = []byte{0}
	_, _, err2 := tp.MaybeAcceptTransaction(tx1, 0)
	if err2 == nil {
		t.Fatal("Expect max pool size error error but no error")
	} else {
		if err2.(*MempoolTxError).Code != ErrCodeMessage[MaxPoolSizeError].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err)
		}
	}
	tp.RoleInCommittees = 0
	_, _, err3 := tp.MaybeAcceptTransaction(tx1, 0)
	if err3 == nil {
		t.Fatal("Expect max pool size error error but no error")
	} else {
		if err3.(*MempoolTxError).Code != ErrCodeMessage[MaxPoolSizeError].Code {
			t.Fatalf("Expect Error %+v but get %+v", ErrCodeMessage[RejectSanityTx], err)
		}
	}
	tp.config.MaxTx = 1
	_, _, err4 := tp.MaybeAcceptTransaction(tx1, 0)
	if err4 != nil {
		t.Fatal("Expect no error but get ", err4)
	}
	ResetMempoolTest()
	tp.config.MaxTx = 1
	tp.IsBlockGenStarted = true
	tp.IsUnlockMempool = true
	tp.config.RelayShards = []byte{0}
	tp.RoleInCommittees = 0
	// test push transaction to block gen
	_, _, err5 := tp.MaybeAcceptTransaction(tx1, 0)
	if err5 != nil {
		t.Fatal("Expect no error but get ", err5)
	}
	go func() {
		tx := <-cPendingTxs
		if !tx.Hash().IsEqual(tx1.Hash()) {
			t.Fatalf("Expect get %+v but get %+v ", tx1.Hash(), tx.Hash())
		}
	}()
}
func TestTxPoolMarkForwardedTransaction(t *testing.T) {
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], 10, false, normalTranferAmount)
	txHash1, txDesc1, err := tp.maybeAcceptTransaction(tx1, false, true, 0)
	if err != nil {
		t.Fatal("Expect no error but get ", err)
	}
	tp.MarkForwardedTransaction(*txHash1)
	if !txDesc1.IsFowardMessage {
		t.Fatal("Tx Should be marked as forwarded already")
	}
}
func TestTxPoolEmptyPool(t *testing.T) {
	ResetMempoolTest()
	tx1 := CreateAndSaveTestNormalTransaction(privateKeyShard0[0], 10, false, normalTranferAmount)
	tx2 := CreateAndSaveTestNormalTransaction(privateKeyShard0[1], 10, false, normalTranferAmount)
	tx3 := CreateAndSaveTestNormalTransaction(privateKeyShard0[2], 10, false, normalTranferAmount)
	txStakingBeacon := CreateAndSaveTestStakingTransaction(privateKeyShard0[4], miningSeedShard0[4], commonFee, true)
	tx6 := CreateAndSaveTestNormalTransaction(privateKeyShard0[5], commonFee, true, 50)
	tp.maybeAcceptTransaction(tx1, true, true, 0)
	tp.maybeAcceptTransaction(tx2, true, true, 0)
	tp.maybeAcceptTransaction(tx3, true, true, 0)
	tp.maybeAcceptTransaction(txStakingBeacon, true, true, 0) // this is fail because can not stake beacon now
	tp.maybeAcceptTransaction(tx6, true, true, 0)
	if len(tp.pool) != 4 {
		t.Fatalf("Expect 4 transaction from mempool but get %+v", len(tp.pool))
	}
	if len(tp.poolSerialNumbersHashList) != 4 {
		t.Fatalf("Expect 4 transaction from mempool but get %+v", len(tp.poolSerialNumbersHashList))
	}
	if len(tp.poolCandidate) != 0 { // because can not stake beacon
		t.Fatalf("Expect 0 but get %+v", len(tp.poolCandidate))
	}
	tp.EmptyPool()

	if len(tp.pool) != 0 {
		t.Fatal("Can't empty pool")
	}
	if len(tp.poolSerialNumbersHashList) != 0 {
		t.Fatal("Can't empty pool serial number")
	}
	if len(tp.poolCandidate) != 0 {
		t.Fatal("Can't empty candidate pool")
	}
}
