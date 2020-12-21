package tx_ver1

import (
	"bytes"
	"fmt"
	"encoding/json"
	"unicode"
	"testing"
	"time"
	"os"
	"math/rand"
	"io/ioutil"
	// "math/big"
	// "strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/trie"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/metadata"
	zkp "github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/aggregatedrange"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/serialnumberprivacy"
	"github.com/stretchr/testify/assert"
)

var (
	// num of private keys
	maxPrivateKeys = 10
	minPrivateKeys = 1

	maxInputs = 10
	minInputs = 1

	maxTries = 100
	numOfLoops = 1
	hasPrivacyForPRV   bool = true
	hasPrivacyForToken bool = false
	shardID            byte = byte(0)
	numInputs = 5
	// must be 1
	numOutputs         = 1
	numTests           = 10
	unitFeeNativeToken = 100
)
var (
	warperDBStatedbTest statedb.DatabaseAccessWarper
	emptyRoot           = common.HexToHash(common.HexEmptyRoot)
	prefixA             = "serialnumber"
	prefixB             = "serialnumberderivator"
	prefixC             = "serial"
	prefixD             = "commitment"
	prefixE             = "outputcoin"
	keysA               = []common.Hash{}
	keysB               = []common.Hash{}
	keysC               = []common.Hash{}
	keysD               = []common.Hash{}
	keysE               = []common.Hash{}
	valuesA             = [][]byte{}
	valuesB             = [][]byte{}
	valuesC             = [][]byte{}
	valuesD             = [][]byte{}
	valuesE             = [][]byte{}

	limit100000 = 100000
	limit10000  = 10000
	limit1000   = 1000
	limit100    = 100
	limit1      = 1

	dummyDB *statedb.StateDB
	testDB * statedb.StateDB
	bridgeDB *statedb.StateDB
	dummyPrivateKeys []*key.PrivateKey
	keySets []*incognitokey.KeySet
	paymentInfo []*key.PaymentInfo
	activeLogger common.Logger
	inactiveLogger common.Logger
)

var _ = func() (_ struct{}) {
// initialize a `test` db in the OS's tempdir
// and with it, a db access wrapper that reads/writes our transactions
	fmt.Println("This runs before init()!")
	testLogFile, err := os.OpenFile("test-log.txt",os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)

	inactiveLogger = common.NewBackend(nil).Logger("test", true)
	activeLogger = common.NewBackend(testLogFile).Logger("test", false)
	activeLogger.SetLevel(common.LevelDebug)
	privacy.LoggerV1.Init(inactiveLogger)
	privacy.LoggerV2.Init(activeLogger)
	// can switch between the 2 loggers to mute logs as one wishes
	utils.Logger.Init(activeLogger)
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_statedb_")
	if err != nil {
		panic(err)
	}
	diskBD, _ := incdb.Open("leveldb", dbPath)
	warperDBStatedbTest = statedb.NewDatabaseAccessWarper(diskBD)
	dummyDB, _ = statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)
	testDB = dummyDB.Copy()
	bridgeDB  = dummyDB.Copy()
	trie.Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

func createSamplePlainCoinV1(privKey privacy.PrivateKey, pubKey *operation.Point, amount uint64, msg []byte) (*coin.PlainCoinV1, error) {
	c := new(coin.PlainCoinV1).Init()

	c.SetValue(amount)
	c.SetInfo(msg)
	c.SetPublicKey(pubKey)
	c.SetSNDerivator(operation.RandomScalar())
	c.SetRandomness(operation.RandomScalar())

	//Derive serial number from snDerivator
	c.SetKeyImage(new(operation.Point).Derive(privacy.PedCom.G[0], new(operation.Scalar).FromBytesS(privKey), c.GetSNDerivator()))

	//Create commitment
	err := c.CommitAll()

	if err != nil {
		return nil, err
	}

	return c, nil
}

func createAndSaveTokens(numCoins int, tokenID common.Hash, keySets []*incognitokey.KeySet, testDB *statedb.StateDB, version int) ([]coin.Coin, error) {
	var err error
	if version == coin.CoinVersion1 {
		coinsToBeSaved := make([]coin.Coin, numCoins*len(keySets))
		for i, keySet := range keySets {
			pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
			if err != nil {
				return nil, err
			}
			for j := 0; j < numCoins; j++ {
				amount := uint64(common.RandIntInterval(0, 100000000))
				tmpCoin, err := createSamplePlainCoinV1(keySet.PrivateKey, pubKey, amount, nil)
				tmpCoin2 := new(coin.CoinV1)
				tmpCoin2.CoinDetails = tmpCoin
				if err != nil {
					return nil, err
				}
				coinsToBeSaved[i*numCoins+j] = tmpCoin2
			}
		}
		cmtBytesToBeSaved := make([][]byte, 0)
		for _, coin := range coinsToBeSaved {
			cmtBytesToBeSaved = append(cmtBytesToBeSaved, coin.GetCommitment().ToBytesS())
		}
		err = statedb.StoreCommitments(testDB, tokenID, cmtBytesToBeSaved, 0)

		return coinsToBeSaved, err
	} else {
		coinsToBeSaved := make([]coin.Coin, numCoins*len(keySets))
		for i, keySet := range keySets {
			for j := 0; j < numCoins; j++ {
				amount := uint64(common.RandIntInterval(0, 100000000))
				paymentInfo := key.InitPaymentInfo(keySet.PaymentAddress, amount, []byte("Dummy token"))

				tmpCoin, err := coin.NewCoinFromPaymentInfo(paymentInfo)
				if err != nil {
					return nil, err
				}
				// keyImage, err := tempCoin.ParseKeyImageWithPrivateKey(keySet.PrivateKey)
				// if err != nil {
				// 	return nil, err
				// }
				// tempCoin.SetKeyImage(keyImage)

				tmpCoin.ConcealOutputCoin(keySet.PaymentAddress.GetPublicView())
				coinsToBeSaved[i*numCoins+j] = tmpCoin
			}
		}

		coinsBytesToBeSaved := make([][]byte, 0)
		otasToBeSaved := make([][]byte, 0)
		for _, c := range coinsToBeSaved {
			coinsBytesToBeSaved = append(coinsBytesToBeSaved, c.Bytes())
			otasToBeSaved = append(otasToBeSaved, c.GetPublicKey().ToBytesS())
		}
		err = statedb.StoreOTACoinsAndOnetimeAddresses(testDB, tokenID, 0, coinsBytesToBeSaved, otasToBeSaved, 0)
		if err != nil {
			return nil, err
		}
		return coinsToBeSaved, nil
	}

}

func prepareKeySets(numKeySets int) ([]*incognitokey.KeySet, error) {
	keySets := make([]*incognitokey.KeySet, numKeySets)
	//generate keysets: we want the public key to be in Shard 0
	for i := 0; i < numKeySets; i++ {
		for {
			//generate a private key
			privateKey := key.GeneratePrivateKey(common.RandBytes(32))

			//make keySets from privateKey
			keySet := new(incognitokey.KeySet)
			err := keySet.InitFromPrivateKey(&privateKey)

			if err != nil {
				return nil, err
			}

			//we want the public key to belong to Shard 0
			if keySet.PaymentAddress.Pk[31] == 0 {
				keySets[i] = keySet
				break
			}
		}
	}
	return keySets, nil
}

func createSamplePlainCoinsFromTotalAmount(senderSK privacy.PrivateKey, pubkey *operation.Point, totalAmount uint64, numFeeInputs, version int) ([]coin.PlainCoin, error) {
	coinList := []coin.PlainCoin{}
	tmpAmount := totalAmount / uint64(numFeeInputs)
	if version == coin.CoinVersion1 {
		for i := 0; i < numFeeInputs-1; i++ {
			amount := tmpAmount - uint64(common.RandIntInterval(0, int(tmpAmount)/2))
			coin, err := createSamplePlainCoinV1(senderSK, pubkey, amount, nil)
			if err != nil {
				return nil, err
			}
			coinList = append(coinList, coin)
			totalAmount -= amount
		}
		coin, err := createSamplePlainCoinV1(senderSK, pubkey, totalAmount, nil)
		if err != nil {
			return nil, err
		}
		coinList = append(coinList, coin)
	} else {
		keySet := new(incognitokey.KeySet)
		err := keySet.InitFromPrivateKey(&senderSK)
		if err != nil {
			return nil, err
		}
		for i := 0; i < numFeeInputs-1; i++ {
			amount := uint64(common.RandIntInterval(0, int(totalAmount)-1))
			paymentInfo := key.InitPaymentInfo(keySet.PaymentAddress, amount, []byte("Hello there"))
			coin, err := coin.NewCoinFromPaymentInfo(paymentInfo)
			if err != nil {
				return nil, err
			}
			coinList = append(coinList, coin)
			totalAmount -= amount
		}
		paymentInfo := key.InitPaymentInfo(keySet.PaymentAddress, totalAmount, []byte("Hello there"))
		coin, err := coin.NewCoinFromPaymentInfo(paymentInfo)
		if err != nil {
			return nil, err
		}
		coinList = append(coinList, coin)
	}
	return coinList, nil
}

func createInitTokenParams(theInputCoin coin.Coin, db *statedb.StateDB, tokenID, tokenName string, keySet *incognitokey.KeySet) (*tx_generic.TxTokenParams, *tx_generic.TokenParam, error) {
	msgCipherText := []byte("Testing Init Token")
	initAmount := uint64(1000000000)
	tokenPayments := []*privacy.PaymentInfo{{PaymentAddress: keySet.PaymentAddress, Amount: initAmount, Message: msgCipherText}}

	myOnlyInputCoin, err := theInputCoin.Decrypt(keySet)
	if err != nil {
		return nil, nil, err
	}
	inputCoinsPRV := []coin.PlainCoin{myOnlyInputCoin}
	paymentInfoPRV := []*privacy.PaymentInfo{key.InitPaymentInfo(keySet.PaymentAddress, uint64(100), []byte("test out"))}

	// token param for init new token
	tokenParam := &tx_generic.TokenParam{
		PropertyID:     tokenID,
		PropertyName:   tokenName,
		PropertySymbol: "DEFAULT",
		Amount:         initAmount,
		TokenTxType:    utils.CustomTokenInit,
		Receiver:       tokenPayments,
		TokenInput:     []coin.PlainCoin{},
		Mintable:       false,
		Fee:            0,
	}

	paramToCreateTx := tx_generic.NewTxTokenParams(&keySet.PrivateKey,
		paymentInfoPRV, inputCoinsPRV, 10, tokenParam, db, nil,
		hasPrivacyForPRV, hasPrivacyForToken, shardID, []byte{}, db)
	return paramToCreateTx, tokenParam, nil
}

func createAndSaveCoinV1s(numCoins, numEquals int, privKey privacy.PrivateKey, pubKey *operation.Point, dummyDB *statedb.StateDB) ([]coin.PlainCoin, error) {
	//amount := uint64(common.RandIntInterval(0, 1000000))
	amount := uint64(numCoins * 1000)
	outCoins := []coin.PlainCoin{}
	for i := 0; i < numEquals; i++ {
		coin, err := createSamplePlainCoinV1(privKey, pubKey, 1000, nil)
		if err != nil {
			return nil, err
		}
		outCoins = append(outCoins, coin)
	}
	tmpOutCoins, err := createSamplePlainCoinsFromTotalAmount(privKey, pubKey, amount, numCoins-numEquals, 1)
	for _, coin := range tmpOutCoins {
		outCoins = append(outCoins, coin)
	}
	if err != nil {
		return nil, err
	}

	//save coins and commitment indices onto the database
	commitmentsToBeSaved := [][]byte{}
	coinsToBeSaved := [][]byte{}
	for _, outCoin := range outCoins {
		coinsToBeSaved = append(coinsToBeSaved, outCoin.Bytes())
		commitmentsToBeSaved = append(commitmentsToBeSaved, outCoin.GetCommitment().ToBytesS())
	}
	err = statedb.StoreOutputCoins(dummyDB, common.PRVCoinID, pubKey.ToBytesS(), coinsToBeSaved, shardID)
	if err != nil {
		return nil, err
	}
	err = statedb.StoreCommitments(dummyDB, common.PRVCoinID, commitmentsToBeSaved, shardID)
	if err != nil {
		return nil, err
	}

	return outCoins, nil
}

func createAndSaveTokenCoinV1s(numCoins, numEquals int, privKey privacy.PrivateKey, pubKey *operation.Point, tokenID common.Hash, dummyDB *statedb.StateDB) ([]coin.PlainCoin, error) {
	//amount := uint64(common.RandIntInterval(0, 1000000))
	amount := uint64(numCoins * 1000)
	outCoins := []coin.PlainCoin{}
	for i := 0; i < numEquals; i++ {
		coin, err := createSamplePlainCoinV1(privKey, pubKey, 1000, nil)
		if err != nil {
			return nil, err
		}
		outCoins = append(outCoins, coin)
	}
	tmpOutCoins, err := createSamplePlainCoinsFromTotalAmount(privKey, pubKey, amount, numCoins-numEquals, 1)
	for _, coin := range tmpOutCoins {
		outCoins = append(outCoins, coin)
	}
	if err != nil {
		return nil, err
	}

	//save coins and commitment indices onto the database
	commitmentsToBeSaved := [][]byte{}
	coinsToBeSaved := [][]byte{}
	for _, outCoin := range outCoins {
		coinsToBeSaved = append(coinsToBeSaved, outCoin.Bytes())
		commitmentsToBeSaved = append(commitmentsToBeSaved, outCoin.GetCommitment().ToBytesS())
	}
	err = statedb.StoreOutputCoins(dummyDB, tokenID, pubKey.ToBytesS(), coinsToBeSaved, shardID)
	if err != nil {
		return nil, err
	}
	err = statedb.StoreCommitments(dummyDB, tokenID, commitmentsToBeSaved, shardID)
	if err != nil {
		return nil, err
	}

	return outCoins, nil
}

func createTxPrivacyInitParams(keySet *incognitokey.KeySet, inputCoins []coin.PlainCoin, hasPrivacy bool, numOutputs int) (*incognitokey.KeySet, *tx_generic.TxPrivacyInitParams, error) {
	//initialize payment info of input coins
	paymentInfos := make([]*key.PaymentInfo, numOutputs)
	sumAmount := uint64(0)
	for _, inputCoin := range inputCoins {
		sumAmount += inputCoin.GetValue()
	}
	amount := sumAmount/uint64(numOutputs)
	for i:= 0; i< numOutputs - 1; i++ {
		paymentInfos[i] = key.InitPaymentInfo(keySet.PaymentAddress, amount, []byte("blahblah"))
		sumAmount -= amount
	}
	paymentInfos[numOutputs-1] = key.InitPaymentInfo(keySet.PaymentAddress, sumAmount, []byte("blahblah"))

	//create privacyinitparam
	txPrivacyInitParam := tx_generic.NewTxPrivacyInitParams(&keySet.PrivateKey,
		paymentInfos,
		inputCoins,
		0,
		hasPrivacy,
		dummyDB,
		&common.PRVCoinID,
		nil,
		nil)

	return keySet, txPrivacyInitParam, nil
}

func TestTxVersion1_ValidateTransaction(t *testing.T) {
	keySets, err := prepareKeySets(1)
	assert.Equal(t, nil, err, "prepareKeySets returns an error: %v", err)
	keySet := keySets[0]

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	assert.Equal(t, nil, err, "Cannot parse public key")

	for i:=0; i < numTests; i++ {
		numOfInputs := RandInt() % (maxInputs - minInputs + 1) + minInputs
		numOfOutputs := RandInt() % (maxInputs - minInputs + 1) + minInputs
		coins, err := createAndSaveCoinV1s(100, 0, keySet.PrivateKey, pubKey, dummyDB)
		assert.Equal(t, nil, err, "createAndSaevCoinV1s returns an error: %v", err)

		tx := new(Tx)

		r := common.RandInt() % (100 - numOfInputs)

		inputCoins := coins[r:r+numOfInputs]

		_, txPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins, true, numOfOutputs)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx.Init(txPrivacyParams)
		assert.Equal(t, nil, err, "validateTxParams returns an error: %v", err)

		res, err := tx.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res)

		res, _, err = tx.ValidateTransaction(true, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, true, res)

		testTxV1JsonMarshaler(tx, 25, dummyDB, t)
	}
}

func TestTxVersion1_InputCoinReplication(t *testing.T) {
	keySets, err := prepareKeySets(1)
	assert.Equal(t, nil, err, "prepareKeySets returns an error: %v", err)
	keySet := keySets[0]

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	assert.Equal(t, nil, err, "Cannot parse public key")

	for i:=0; i< numTests; i++ {
		coins, err := createAndSaveCoinV1s(100, 2, keySet.PrivateKey, pubKey, dummyDB)
		assert.Equal(t, nil, err, "createAndSaevCoinV1s returns an error: %v", err)

		// fmt.Println(coins[0].GetValue())

		tx := new(Tx)
		//choose some input coins to spend => make sure that inputCoins[0] and inputCoins[1] have the same amount
		inputCoins := coins[:10]

		_, txPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx_generic.ValidateTxParams(txPrivacyParams)
		assert.Equal(t, nil, err, "validateTxParams returns an error: %v", err)

		err = tx.InitializeTxAndParams(txPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error: %v", err)

		paymentWitnessParamPtr, err := tx.initializePaymentWitnessParam(txPrivacyParams)
		assert.Equal(t, nil, err, "initializePaymentWitnessParam returns an error: %v", err)

		//copy the first coin to the second
		sz := privacy.CommitmentRingSize
		paymentWitnessParamPtr.InputCoins[1] = paymentWitnessParamPtr.InputCoins[0]
		paymentWitnessParamPtr.MyCommitmentIndices[1] = paymentWitnessParamPtr.MyCommitmentIndices[0]
		for i := 0; i < sz; i++ {
			paymentWitnessParamPtr.CommitmentIndices[i+sz] = paymentWitnessParamPtr.CommitmentIndices[i]
			paymentWitnessParamPtr.Commitments[i+sz] = paymentWitnessParamPtr.Commitments[i]
		}

		err = tx.proveAndSignCore(txPrivacyParams, paymentWitnessParamPtr)
		assert.Equal(t, nil, err, "proveAndSignCore returns an error: %v")

		res, err := tx.ValidateSanityData(nil,nil,nil,0)
		assert.Equal(t, false, res)
		if res {
			assert.Equal(t, true, bytes.Equal(paymentWitnessParamPtr.InputCoins[1].Bytes(), paymentWitnessParamPtr.InputCoins[0].Bytes()))
		}
	}
}

func TestTxVersion1_BulletProofCommitmentConsistency(t *testing.T) {
	keySets, err := prepareKeySets(1)
	assert.Equal(t, nil, err, "prepareKeySets returns an error: %v", err)
	keySet := keySets[0]

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	assert.Equal(t, nil, err, "Cannot parse public key")

	coins, err := createAndSaveCoinV1s(numTests*50, 0, keySet.PrivateKey, pubKey, dummyDB)
	assert.Equal(t, nil, err, "createAndSaevCoinV1s returns an error: %v", err)

	for i:=0;i<numTests;i++ {
		//create 2 transactions
		tx1 := new(Tx)
		tx2 := new(Tx)
		inputCoins := coins[i*20 : i*20+10]

		_, txPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx1.Init(txPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error: %v", err)

		res, err := tx1.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res)

		res, _, err = tx1.ValidateTransaction(true, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, true, res)

		proof, ok := tx1.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//retrieve the bulletproof
		oldAggregatedProof, ok := proof.GetAggregatedRangeProof().(*aggregatedrange.AggregatedRangeProof)
		assert.Equal(t, true, ok, "cannot parse bulletproof")

		//create a new transaction and reuse the bulletproof
		inputCoins2 := coins[i*20+10 : i*20+20]

		_, newTxPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins2, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx_generic.ValidateTxParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "validateTxParams returns an error: %v", err)

		err = tx2.InitializeTxAndParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error: %v", err)

		newPaymentWitnessParamPtr, err := tx2.initializePaymentWitnessParam(newTxPrivacyParams)
		assert.Equal(t, nil, err, "initializePaymentWitnessParam returns an error: %v", err)

		err = tx2.proveAndSignCore(newTxPrivacyParams, newPaymentWitnessParamPtr)
		assert.Equal(t, nil, err, "proveAndSignCore returns an error: %v", err)

		proof, ok = tx2.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//assert that tx2 and tx1 have different bulletproofs
		assert.Equal(t, false, bytes.Equal(oldAggregatedProof.Bytes(), proof.Bytes()), "tx2 and tx1 have the same bulletproof")

		proof.SetAggregatedRangeProof(oldAggregatedProof)

		tx2.Sig = nil
		err = tx2.sign()
		assert.Equal(t, nil, err, "sign returns an error: %v", err)

		res, err = tx2.ValidateSanityData(nil, nil, nil, 0)
		//assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, false, res)

	}
}

func TestTxVersion1_SerialNumberProofConsistency(t *testing.T) {
	keySets, err := prepareKeySets(1)
	assert.Equal(t, nil, err, "prepareKeySets returns an error: %v", err)
	keySet := keySets[0]

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	assert.Equal(t, nil, err, "Cannot parse public key")

	coins, err := createAndSaveCoinV1s(numTests*50, 0, keySet.PrivateKey, pubKey, dummyDB)
	assert.Equal(t, nil, err, "createAndSaevCoinV1s returns an error: %v", err)

	for i := 0; i < numTests; i++ {
		//create 2 transactions
		tx1 := new(Tx)
		tx2 := new(Tx)
		inputCoins := coins[i*20 : i*20+10]

		_, txPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx1.Init(txPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error at test number %v: %v", i, err)

		res, err := tx1.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res)

		res, _, err = tx1.ValidateTransaction(true, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, true, res)

		proof, ok := tx1.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//retrieve the serial number proof
		oldSNProof := proof.GetSerialNumberProof()

		//create a new transaction and reuse the serial number proof
		inputCoins2 := coins[i*20+10 : i*20+20]

		_, newTxPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins2, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx_generic.ValidateTxParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "validateTxParams returns an error: %v", err)

		err = tx2.InitializeTxAndParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error: %v", err)

		newPaymentWitnessParamPtr, err := tx2.initializePaymentWitnessParam(newTxPrivacyParams)
		assert.Equal(t, nil, err, "initializePaymentWitnessParam returns an error: %v", err)

		err = tx2.proveAndSignCore(newTxPrivacyParams, newPaymentWitnessParamPtr)
		assert.Equal(t, nil, err, "proveAndSignCore returns an error: %v", err)

		proof, ok = tx2.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//check that the SNProof has changed
		assert.Equal(t, false, bytes.Equal(oldSNProof[0].Bytes(), proof.GetSerialNumberProof()[0].Bytes()))
		assert.Equal(t, false, bytes.Equal(oldSNProof[1].Bytes(), proof.GetSerialNumberProof()[1].Bytes()))
		proof.SetSerialNumberProof(oldSNProof)

		//re-sign the transaction
		tx2.Sig = nil
		err = tx2.sign()
		assert.Equal(t, nil, err, "sign returns an error: %v", err)

		res, err = tx2.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, false, res)
	}
}

func TestTxVersion1_OneOutOfManyProofConsistency(t *testing.T) {
	keySets, err := prepareKeySets(1)
	assert.Equal(t, nil, err, "prepareKeySets returns an error: %v", err)
	keySet := keySets[0]

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	assert.Equal(t, nil, err, "Cannot parse public key")

	coins, err := createAndSaveCoinV1s(numTests*50, 0, keySet.PrivateKey, pubKey, dummyDB)
	assert.Equal(t, nil, err, "createAndSaevCoinV1s returns an error: %v", err)

	for i := 0; i < numTests; i++ {
		//create 2 transactions
		tx1 := new(Tx)
		tx2 := new(Tx)
		inputCoins := coins[i*20 : i*20+10]

		_, txPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx1.Init(txPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error at test number %v: %v", i, err)

		res, err := tx1.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res)

		res, _, err = tx1.ValidateTransaction(true, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, true, res)

		proof, ok := tx1.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//retrieve the serial number proof
		oldOneOfMany := proof.GetOneOfManyProof()

		//create a new transaction and reuse the serial number proof
		inputCoins2 := coins[i*20+10 : i*20+20]

		_, newTxPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins2, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		err = tx_generic.ValidateTxParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "validateTxParams returns an error: %v", err)

		err = tx2.InitializeTxAndParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error: %v", err)

		newPaymentWitnessParamPtr, err := tx2.initializePaymentWitnessParam(newTxPrivacyParams)
		assert.Equal(t, nil, err, "initializePaymentWitnessParam returns an error: %v", err)

		err = tx2.proveAndSignCore(newTxPrivacyParams, newPaymentWitnessParamPtr)
		assert.Equal(t, nil, err, "proveAndSignCore returns an error: %v", err)

		proof, ok = tx2.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//check that the SNProof has changed
		assert.Equal(t, false, bytes.Equal(oldOneOfMany[0].Bytes(), proof.GetSerialNumberProof()[0].Bytes()))
		assert.Equal(t, false, bytes.Equal(oldOneOfMany[1].Bytes(), proof.GetSerialNumberProof()[1].Bytes()))
		proof.SetOneOfManyProof(oldOneOfMany)

		//re-sign the transaction
		tx2.Sig = nil
		err = tx2.sign()
		assert.Equal(t, nil, err, "sign returns an error: %v", err)

		res, err = tx2.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res)

		res, _, err = tx2.ValidateTransaction(true, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		//assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, false, res)
	}
}

func TestTxVersion1_SerialNumberNoPrivacyProofConsistency(t *testing.T) {
	keySets, err := prepareKeySets(1)
	assert.Equal(t, nil, err, "prepareKeySets returns an error: %v", err)
	keySet := keySets[0]

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	assert.Equal(t, nil, err, "Cannot parse public key")

	coins, err := createAndSaveCoinV1s(numTests*50, 0, keySet.PrivateKey, pubKey, dummyDB)
	assert.Equal(t, nil, err, "createAndSaevCoinV1s returns an error: %v", err)

	for i := 0; i < numTests; i++ {
		//create 2 transactions
		tx1 := new(Tx)
		tx2 := new(Tx)
		inputCoins := coins[i*20 : i*20+10]

		_, txPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins, false, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)
		txPrivacyParams.HasPrivacy = false // set hasPrivacy to false to obtain serialNumberNoPrivacyProof

		err = tx1.Init(txPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error at test number %v: %v", i, err)

		res, err := tx1.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res)

		res, _, err = tx1.ValidateTransaction(false, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, true, res)

		proof, ok := tx1.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//retrieve the serial number proof
		oldSNNoPrivacyProof := proof.GetSerialNumberNoPrivacyProof()

		//create a new transaction and reuse the serial number proof
		inputCoins2 := coins[i*20+10 : i*20+20]

		_, newTxPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins2, false, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)
		newTxPrivacyParams.HasPrivacy = false // set hasPrivacy to false to obtain serialNumberNoPrivacyProof

		err = tx_generic.ValidateTxParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "validateTxParams returns an error: %v", err)

		err = tx2.InitializeTxAndParams(newTxPrivacyParams)
		assert.Equal(t, nil, err, "Init returns an error: %v", err)

		newPaymentWitnessParamPtr, err := tx2.initializePaymentWitnessParam(newTxPrivacyParams)
		assert.Equal(t, nil, err, "initializePaymentWitnessParam returns an error: %v", err)

		err = tx2.proveAndSignCore(newTxPrivacyParams, newPaymentWitnessParamPtr)
		assert.Equal(t, nil, err, "proveAndSignCore returns an error: %v", err)

		proof, ok = tx2.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse proof")

		//check that the SNProof has changed
		if len(oldSNNoPrivacyProof) > 0{
			assert.Equal(t, false, bytes.Equal(oldSNNoPrivacyProof[0].Bytes(), proof.GetSerialNumberNoPrivacyProof()[0].Bytes()))
			proof.SetSerialNumberNoPrivacyProof(oldSNNoPrivacyProof)
		}

		//re-sign the transaction
		tx2.Sig = nil
		err = tx2.sign()
		assert.Equal(t, nil, err, "sign returns an error: %v", err)

		res, err = tx2.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res) //This should fail

		res, _, err = tx2.ValidateTransaction(false, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		//assert.Equal(t, nil, err, "ValidateTransaction returns an error: %v", err)
		assert.Equal(t, true, res)
	}
}

func TestTxVersion1_OutputTampered(t *testing.T) {
	//This test will attempt to create a transaction ver1 which has output value larger than sum of input values
	keySets, err := prepareKeySets(1)
	assert.Equal(t, nil, err, "prepareKeySets returns an error: %v", err)
	keySet := keySets[0]

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	assert.Equal(t, nil, err, "Cannot parse public key")

	coins, err := createAndSaveCoinV1s(numTests*50, 0, keySet.PrivateKey, pubKey, dummyDB)
	assert.Equal(t, nil, err, "createAndSaevCoinV1s returns an error: %v", err)

	for i := 0; i < numTests; i++ {
		//create 2 transactions
		tx := new(Tx)
		inputCoins := coins[i*10 : i*10 + 10]

		_, txPrivacyParams, err := createTxPrivacyInitParams(keySet, inputCoins, true, 1)
		assert.Equal(t, nil, err, "createTxPrivacyInitParams returns an error: %v", err)

		//create an output whose amount is larger than sum of all input amounts
		txPrivacyParams.PaymentInfo[0].Amount += 1000

		//initialize transaction and param (initializeTxAndParams without updateParamsWhenOverBalance)
		tx.SetPrivateKey(keySet.PrivateKey)
		if tx.LockTime == 0 {
			tx.LockTime = time.Now().Unix()
		}
		tx.Fee = txPrivacyParams.Fee
		tx.Type = common.TxNormalType
		tx.Metadata = txPrivacyParams.MetaData
		tx.PubKeyLastByteSender = keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		tx.Version = 1
		tx.Info, err = tx_generic.GetTxInfo(txPrivacyParams.Info)
		assert.Equal(t, nil, err, "getTxInfo returns an error: %v", err)

		err = tx.prove(txPrivacyParams)
		assert.Equal(t, nil, err, "tx.Prove returns an error: %v", err)

		proof, ok := tx.Proof.(*zkp.PaymentProof)
		assert.Equal(t, true, ok, "cannot parse payment proof")


		cmOutputValueSum := proof.GetCommitmentOutputValue()
		cmInputValues := proof.GetCommitmentInputValue()
		cmInputSND := proof.GetCommitmentInputSND()
		serialNumberProof := proof.GetSerialNumberProof()

		tmpSNProof := serialnumberprivacy.Copy(*serialNumberProof[0])

		cmInputValueSum := new(operation.Point).Identity()
		for _, cmInputValue := range cmInputValues {
			cmInputValueSum.Add(cmInputValueSum, cmInputValue)
		}

		tmpDiff := new(operation.Point).Sub(cmOutputValueSum[0], cmInputValueSum)
		cmInputValues[0].Add(cmInputValues[0], tmpDiff)
		cmInputSND[0].Sub(cmInputSND[0], tmpDiff)

		proof.SetCommitmentInputValue(cmInputValues)
		proof.SetCommitmentInputSND(cmInputSND)
		serialNumberProof[0] = tmpSNProof
		proof.SetSerialNumberProof(serialNumberProof)


		tx.SetProof(proof)

		tx.Sig = nil
		err = tx.sign()

		assert.Equal(t, nil, err, "tx.sign returns an error: %v", err)

		res, err := tx.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, nil, err, "ValidateSanityData returns an error: %v", err)
		assert.Equal(t, true, res)

		res, _, err = tx.ValidateTransaction(true, dummyDB, dummyDB, 0, &common.PRVCoinID, false, false)
		assert.Equal(t, true, res, err)

	}
}

func testTxV1JsonMarshaler(tx *Tx, count int, db *statedb.StateDB, t *testing.T){
	someInvalidTxs := getCorruptedJsonDeserializedTxs(tx, count, t)
	for _,theInvalidTx := range someInvalidTxs{
		txSpecific, ok := theInvalidTx.(*Tx)
		if !ok{
			fmt.Println("Skipping a transaction from wrong version")
			continue
		}
		// look for potential panics by calling verify
		isSane, _ := txSpecific.ValidateSanityData(nil, nil, nil, 0)
		// if it doesnt pass sanity then the next validation could panic, it's ok by spec
		if !isSane{
			continue
		}
		isValid, _ := txSpecific.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, shardID, false, nil, nil)
		if !isValid{
			continue
		}
		txSpecific.ValidateTxWithBlockChain(nil, nil, nil, shardID, db)
	}
}

func testTxTokenV1JsonMarshaler(tx *TxToken, count int, db *statedb.StateDB, t *testing.T){
	someInvalidTxs := getCorruptedJsonDeserializedTokenTxs(tx, count, t)
	for _,theInvalidTx := range someInvalidTxs{
		txSpecific, ok := theInvalidTx.(*TxToken)
		if !ok{
			fmt.Println("Skipping a transaction from wrong version")
			continue
		}
		// look for potential panics by calling verify
		isSane, _ := txSpecific.ValidateSanityData(nil, nil, nil, 0)
		// if it doesnt pass sanity then the next validation could panic, it's ok by spec
		if !isSane{
			continue
		}
		isValid, _ := txSpecific.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, shardID, false, nil, nil)
		if !isValid{
			continue
		}
		txSpecific.ValidateTxWithBlockChain(nil, nil, nil, shardID, db)
	}
}

func getRandomDigit() rune{

	ind := RandInt() % 10
	return rune(int(rune('0'))+ind)
}

func getRandomLetter() rune{
	ind := RandInt() % 52
	if ind < 26{
		return rune(int(rune('A'))+ind)
	}else{
		return rune(int(rune('a'))+ind-26)
	}
}

func getCorruptedJsonDeserializedTxs(tx *Tx, maxJsonChanges int, t *testing.T) []metadata.Transaction{
	jsonBytes, err := json.Marshal(tx)
	assert.Equal(t, nil, err)
	reconstructedTx := &Tx{}
	err = json.Unmarshal(jsonBytes, reconstructedTx)
	assert.Equal(t, nil, err)
	jsonBytesAgain, err := json.Marshal(reconstructedTx)
	assert.Equal(t, true, bytes.Equal(jsonBytes, jsonBytesAgain))
	var result []metadata.Transaction
	// json bytes are readable strings
	// we try to malleify a letter / digit
	// if that char is part of a key then it's equivalent to deleting that attribute
	for i:=0; i<maxJsonChanges; i++{
		// let the changes stack up many times to exhaust more cases
		s := string(jsonBytesAgain)
		theRunes := []rune(s)
		corruptedIndex := RandInt() % len(theRunes)
		for j:=maxTries;j>0;j--{
			if j==0{
				fmt.Printf("Strange letterless TX with json form : %s\n",s)
				panic("End")
			}
			if unicode.IsLetter(theRunes[corruptedIndex]) || unicode.IsDigit(theRunes[corruptedIndex]){
				break
			}
			// not letter -> retry
			corruptedIndex = RandInt() % len(theRunes)
		}
		// replace this letter with a random one
		if unicode.IsLetter(theRunes[corruptedIndex]){
			theRunes[corruptedIndex] = getRandomLetter()
		}else{
			theRunes[corruptedIndex] = getRandomDigit()
		}


		// reconstructedTx, err = NewTransactionFromJsonBytes([]byte(string(theRunes)))
		err := json.Unmarshal([]byte(string(theRunes)), reconstructedTx)
		if err != nil{
			// fmt.Printf("A byte array failed to deserialize\n")
			continue
		}
		result = append(result,reconstructedTx)
	}
	// fmt.Printf("Made %d dummy faulty txs\n",len(result))
	return result
}

func getCorruptedJsonDeserializedTokenTxs(tx *TxToken, maxJsonChanges int,t *testing.T) []tx_generic.TransactionToken{
	jsonBytes, err := json.Marshal(tx)
	assert.Equal(t, nil, err)
	reconstructedTx := &TxToken{}
	err = json.Unmarshal(jsonBytes, reconstructedTx)
	assert.Equal(t, nil, err)
	jsonBytesAgain, err := json.Marshal(reconstructedTx)
	assert.Equal(t, true, bytes.Equal(jsonBytes, jsonBytesAgain))
	var result []tx_generic.TransactionToken

	for i:=0; i<maxJsonChanges; i++{
		s := string(jsonBytesAgain)
		theRunes := []rune(s)
		corruptedIndex := RandInt() % len(theRunes)
		for j:=maxTries;j>0;j--{
			if j==0{
				fmt.Printf("Strange letterless TX with json form : %s\n",s)
				panic("End")
			}
			if unicode.IsLetter(theRunes[corruptedIndex]) || unicode.IsDigit(theRunes[corruptedIndex]){
				break
			}
			// not letter -> retry
			corruptedIndex = RandInt() % len(theRunes)
		}
		// replace this letter with a random one
		if unicode.IsLetter(theRunes[corruptedIndex]){
			theRunes[corruptedIndex] = getRandomLetter()
		}else{
			theRunes[corruptedIndex] = getRandomDigit()
		}


		// reconstructedTx, err = NewTransactionTokenFromJsonBytes([]byte(string(theRunes)))
		err := json.Unmarshal([]byte(string(theRunes)), reconstructedTx)
		if err != nil{
			// fmt.Printf("A byte array failed to deserialize\n")
			continue
		}
		result = append(result,reconstructedTx)
	}
	return result
}

func RandInt() int {
	return rand.Int()
}