package tx_ver2

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"github.com/incognitochain/incognito-chain/trie"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

var (
	numInputs = 5
	// must be 1
	numOutputs         = 1
	numTests           = 10
	unitFeeNativeToken = 100
	//create a sample test DB
	testDB, _ = statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)
)

var _ = func() (_ struct{}) {
	// initialize a `test` db in the OS's tempdir
	// and with it, a db access wrapper that reads/writes our transactions
	fmt.Println("This runs before init()!")
	utils.Logger.Init(common.NewBackend(nil).Logger("test", true))
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_statedb_")
	if err != nil {
		panic(err)
	}
	diskBD, _ := incdb.Open("leveldb", dbPath)
	warperDBStatedbTest = statedb.NewDatabaseAccessWarper(diskBD)
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

func createConversionParams(numInputs, numOutputs int, tokenID *common.Hash) (*incognitokey.KeySet,
	[]*privacy.PaymentInfo, *TxConvertVer1ToVer2InitParams, error) {
	var senderSK privacy.PrivateKey
	var keySet *incognitokey.KeySet

	//create a sample test DB
	testDB, _ := statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)

	//generate keyset: we want the public key to be in Shard 0
	for {
		//generate a private key
		senderSK = key.GeneratePrivateKey(common.RandBytes(32))

		//make keySets from privateKey
		keySet = new(incognitokey.KeySet)
		err := keySet.InitFromPrivateKey(&senderSK)

		if err != nil {
			return nil, nil, nil, err
		}

		//we want the public key to belong to Shard 0
		if keySet.PaymentAddress.Pk[31] == 0 {
			break
		}
	}

	//create input coins
	inputCoins := make([]coin.PlainCoin, numInputs)
	sumInput := uint64(0)
	var err error

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	if err != nil {
		return nil, nil, nil, err
	}

	for i := 0; i < numInputs; i++ {
		amount := uint64(common.RandIntInterval(0, 1000))
		inputCoins[i], err = createSamplePlainCoinV1(senderSK, pubKey, amount, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		sumInput += amount
	}

	sumOutput := uint64(0)

	paymentInfo := make([]*key.PaymentInfo, numOutputs)

	for i := 0; i < numOutputs; i++ {
		amount := uint64(common.RandIntInterval(0, int(sumInput-sumOutput)))
		paymentInfo[i] = key.InitPaymentInfo(keySet.PaymentAddress, amount, nil)
		sumOutput += amount
	}

	//calculate sample fee
	fee := sumInput - sumOutput

	//create conversion params
	txConversionParams := NewTxConvertVer1ToVer2InitParams(
		&keySet.PrivateKey,
		paymentInfo,
		inputCoins,
		fee,
		testDB,
		tokenID, // use for prv coin -> nil is valid
		nil,
		nil,
	)

	return keySet, paymentInfo, txConversionParams, nil
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

				fmt.Println("BUGLOG2 cmt in CreateAndSaveCoin", tmpCoin.GetCommitment())
				fmt.Println("BUGLOG2 value and randomness", tmpCoin.GetValue(), tmpCoin.GetRandomness())

				err = tmpCoin.ConcealOutputCoin(keySet.PaymentAddress.GetPublicView())
				if err != nil {
					return nil, err
				}
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

func createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, feeInputVersion int, tokenID *common.Hash) (*incognitokey.KeySet, *TxTokenConvertVer1ToVer2InitParams, error) {
	keySets, err := prepareKeySets(1)
	if err != nil {
		return nil, nil, err
	}
	keySet := keySets[0]
	senderSK := keySet.PrivateKey

	//create input tokens
	inputTokens := make([]coin.PlainCoin, numTokenInputs)
	sumInput := uint64(0)

	pubKey, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
	if err != nil {
		return nil, nil, err
	}

	for i := 0; i < numTokenInputs; i++ {
		amount := uint64(common.RandIntInterval(0, 1000))
		inputTokens[i], err = createSamplePlainCoinV1(senderSK, pubKey, amount, nil)
		if err != nil {
			return nil, nil, err
		}
		sumInput += amount
	}

	//initialize payment info of input token coins
	paymentInfos := coin.CreatePaymentInfosFromPlainCoinsAndAddress(inputTokens, keySet.PaymentAddress, nil)

	//create some fake fee and PRV coins for paying this fee
	realFee := uint64(common.RandIntInterval(0, 1000))

	//we want the sum of these PRV coins to be large than the real fee
	overBalance := uint64(common.RandIntInterval(0, 1000))

	//create PRV fee input coins of version 2 (bigger than realFee => over balance) and store onto the database
	feeInputs, err := createSamplePlainCoinsFromTotalAmount(senderSK, pubKey, realFee+overBalance, numFeeInputs, feeInputVersion)
	for _, feeInput := range feeInputs {
		keyImage, err := feeInput.ParseKeyImageWithPrivateKey(senderSK)
		if err != nil {
			return nil, nil, err
		}
		feeInput.SetKeyImage(keyImage)
	}
	feeInputBytes := [][]byte{}
	otas := [][]byte{}
	for _, feeInput := range feeInputs {
		feeInputBytes = append(feeInputBytes, feeInput.Bytes())
		otas = append(otas, feeInput.GetPublicKey().ToBytesS())
	}
	err = statedb.StoreOTACoinsAndOnetimeAddresses(testDB, common.PRVCoinID, 0, feeInputBytes, otas, 0)
	if err != nil {
		return nil, nil, err
	}

	if err != nil {
		return nil, nil, err
	}

	//create a return payment for the sender if overbalance > 0
	feePayments := []*privacy.PaymentInfo{}
	if overBalance > 0 {
		for i := 0; i < numFeePayments-1; i++ {
			amount := uint64(common.RandIntInterval(0, int(overBalance)-1))
			feePayment := key.InitPaymentInfo(keySet.PaymentAddress, amount, []byte("Returning over balance to the sender"))
			feePayments = append(feePayments, feePayment)
			overBalance -= amount
		}
		feePayment := key.InitPaymentInfo(keySet.PaymentAddress, overBalance, []byte("Returning over balance to the sender"))
		feePayments = append(feePayments, feePayment)
	}

	//create conversion params
	txTokenConversionParams := NewTxTokenConvertVer1ToVer2InitParams(&keySet.PrivateKey,
		feeInputs,
		feePayments,
		inputTokens,
		paymentInfos,
		realFee,
		testDB,
		bridgeDB,
		tokenID,
		nil,
		nil)

	return keySet, txTokenConversionParams, nil
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

func TestInitializeTxConversion(t *testing.T) {
	for i := 0; i < numTests; i++ {
		_, _, txConversionParams, err := createConversionParams(numInputs, numOutputs, &common.PRVCoinID)
		assert.Equal(t, nil, err, "createConversionParams returns an error: %v", err)

		//Test initializeTxConversion
		txConversionOutput := new(Tx)
		err = initializeTxConversion(txConversionOutput, txConversionParams)

		assert.Equal(t, nil, err, "initializeTxConversion returns an error: %v", err)
	}
}

func TestProveVerifyTxNormalConversion(t *testing.T) {
	for i := 0; i < numTests; i++ {
		txConvertOutput := new(Tx)

		_, _, txConvertParams, err := createConversionParams(numInputs, numOutputs, &common.PRVCoinID)
		assert.Equal(t, nil, err, "createConversionParams returns an error: %v", err)

		err = initializeTxConversion(txConvertOutput, txConvertParams)
		assert.Equal(t, nil, err, "initializeTxConversion returns an error: %v", err)

		err = proveConversion(txConvertOutput, txConvertParams)
		assert.Equal(t, nil, err, "proveConversion returns an error: %v", err)

		res, err := tx_generic.ValidateSanity(txConvertOutput, nil, nil, nil, 0)
		assert.Equal(t, true, err == nil, "validateConversionVer1ToVer2 should not return any error")
		assert.Equal(t, true, res, "validateConversionVer1ToVer2 should be true")

		res, err = tx_generic.MdValidate(txConvertOutput, false, txConvertParams.stateDB, nil, 0)
		// res, err := validateConversionVer1ToVer2(txConvertOutput, txConvertParams.stateDB, 0, &common.PRVCoinID)
		assert.Equal(t, true, err == nil, "validateConversionVer1ToVer2 should not return any error")
		assert.Equal(t, true, res, "validateConversionVer1ToVer2 should be true")

		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = false
		boolParams["isNewTransaction"] = true
		res, err = txConvertOutput.ValidateTxByItself(boolParams, txConvertParams.stateDB, nil, nil, 0, nil, nil)

	}
}

func TestProveVerifyTxNormalConversionTampered(t *testing.T) {
	for i := 0; i < numTests; i++ {
		txConversionOutput := new(Tx)
		_, _, txConversionParams, err := createConversionParams(numInputs, numOutputs, &common.PRVCoinID)
		assert.Equal(t, nil, err, "createConversionParams returns an error: %v", err)

		senderSk := txConversionParams.senderSK
		testDB := txConversionParams.stateDB

		//Initialize conversion and prove
		err = InitConversion(txConversionOutput, txConversionParams)
		assert.Equal(t, nil, err, "proveConversion returns an error: %v", err)

		m := common.RandInt()

		switch m % 5 { //Change this if you want to test a specific case

		//tamper with fee
		case 0:
			fmt.Println("------------------Tampering with fee-------------------")
			txConversionOutput.Fee = uint64(common.RandIntInterval(0, 1000))

			//Re-sign transaction
			txConversionOutput.Sig, txConversionOutput.SigPubKey, err = tx_generic.SignNoPrivacy(senderSk, txConversionOutput.Hash()[:])
			assert.Equal(t, nil, err)

		//tamper with randomness
		case 1:
			fmt.Println("------------------Tampering with randomness-------------------")
			inputCoins := txConversionOutput.Proof.GetInputCoins()
			for j := 0; j < numInputs; j++ {
				inputCoins[j].SetRandomness(operation.RandomScalar())
			}

			//Re-sign transaction
			txConversionOutput.Sig, txConversionOutput.SigPubKey, err = tx_generic.SignNoPrivacy(senderSk, txConversionOutput.Hash()[:])
			assert.Equal(t, nil, err)

		//attempt to convert used coins (used serial numbers)
		case 2:
			fmt.Println("------------------Tampering with serial numbers-------------------")
			usedIndex := common.RandInt() % len(txConversionOutput.Proof.GetInputCoins())
			inputCoin := txConversionOutput.Proof.GetInputCoins()[usedIndex]

			err := statedb.StoreSerialNumbers(testDB, *txConversionParams.tokenID, [][]byte{inputCoin.GetKeyImage().ToBytesS()}, 0)
			assert.Equal(t, true, err == nil)

		//tamper with OTAs
		case 3:
			fmt.Println("------------------Tampering with OTAs-------------------")
			usedIndex := common.RandInt() % len(txConversionOutput.Proof.GetOutputCoins())
			outputCoin := txConversionOutput.Proof.GetOutputCoins()[usedIndex]

			forceSaveCoins(testDB, []coin.Coin{outputCoin}, 0, *txConversionParams.tokenID, t)
			// assert.Equal(t, true, err == nil)

		//tamper with commitment of output
		case 4:
			fmt.Println("------------------Tampering with commitment-------------------")
			//fmt.Println("Tx hash before altered", txConversionOutput.Hash().String())
			newOutputCoins := []coin.Coin{}
			outputCoin := txConversionOutput.Proof.GetOutputCoins()[0]
			newOutputCoin, ok := outputCoin.(*coin.CoinV2)
			assert.Equal(t, true, ok)

			//Attempt to alter some output coins!
			value := new(operation.Scalar).Add(newOutputCoin.GetAmount(), new(operation.Scalar).FromUint64(100))
			randomness := outputCoin.GetRandomness()
			newCommitment := operation.PedCom.CommitAtIndex(value, randomness, operation.PedersenValueIndex)
			newOutputCoin.SetCommitment(newCommitment)
			newOutputCoins = append(newOutputCoins, newOutputCoin)
			txConversionOutput.Proof.SetOutputCoins(newOutputCoins)
			//fmt.Println("Tx hash after altered", txConversionOutput.Hash().String())

			//Re-sign transaction
			txConversionOutput.Sig, txConversionOutput.SigPubKey, err = tx_generic.SignNoPrivacy(senderSk, txConversionOutput.Hash()[:])
			assert.Equal(t, nil, err)
		default:

		}
		//Attempt to verify
		isValidSanity, _ := txConversionOutput.ValidateSanityData(nil, nil, nil, 0)
		var isValid bool
		if !isValidSanity {
			isValid = false
		} else {
			boolParams := make(map[string]bool)
			boolParams["hasPrivacy"] = false
			boolParams["isNewTransaction"] = true
			isValid, err = txConversionOutput.ValidateTxByItself(boolParams, txConversionParams.stateDB, nil, nil, 0, nil, nil)
			tx_generic.MdValidate(txConversionOutput, false, txConversionParams.stateDB, nil, 0)
		}
		// assert.Equal(t, false, isValid, "validateConversionVer1ToVer2 on corrupted data should not be true")
		if isValid {
			jsb, _ := json.Marshal(txConversionOutput)
			fmt.Printf("Unexpected error in case %d : %v\nTransaction : %s\n", m%5, err, string(jsb))
			// return
		}
		//This validation should return an error
		assert.Equal(t, false, isValid)
		//Result should be false
		// assert.Equal(t, false, res)
	}
}

func TestValidateTxTokenConversion(t *testing.T) {
	for i := 0; i < 100; i++ {
		r := common.RandBytes(1)[0]
		tokenID := &common.Hash{r}

		m := common.RandInt()
		switch m % 6 {
		//attempt to use a large number of input tokens
		case 0:
			numTokenInputs, numFeeInputs, numFeePayments := 256, 5, 10
			_, txTokenConversionParams, err := createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, 2, tokenID)
			assert.Equal(t, nil, err, "createTokenConversionParams returns an error: %v", err)

			//Test validate txTokenConversionParams
			err = validateTxTokenConvertVer1ToVer2Params(txTokenConversionParams)
			assert.Equal(t, true, err != nil)
		//attempt to use a large number of input PRV fee coins
		case 1:
			numTokenInputs, numFeeInputs, numFeePayments := 5, 256, 10
			_, txTokenConversionParams, err := createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, 2, tokenID)
			assert.Equal(t, nil, err, "createTokenConversionParams returns an error: %v", err)

			//Test validate txTokenConversionParams
			err = validateTxTokenConvertVer1ToVer2Params(txTokenConversionParams)
			assert.Equal(t, true, err != nil)
		//attempt to return a large number of output coins
		case 2:
			numTokenInputs, numFeeInputs, numFeePayments := 5, 10, 256
			_, txTokenConversionParams, err := createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, 2, tokenID)
			assert.Equal(t, nil, err, "createTokenConversionParams returns an error: %v", err)

			//Test validate txTokenConversionParams
			err = validateTxTokenConvertVer1ToVer2Params(txTokenConversionParams)
			assert.Equal(t, true, err != nil)
		//attempt to use PRV coins ver 1 as fee inputs
		case 3:
			// TODO: reconsider this test
			// storeOTA cannot be used to store coinv1 or the db will be corrupted
			return
			numTokenInputs, numFeeInputs, numFeePayments := common.RandIntInterval(0, 255), common.RandIntInterval(0, 255), common.RandIntInterval(0, 255)
			_, txTokenConversionParams, err := createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, 1, tokenID)
			assert.Equal(t, nil, err, "createTokenConversionParams returns an error: %v", err)

			//Test validate txTokenConversionParams
			err = validateTxTokenConvertVer1ToVer2Params(txTokenConversionParams)
			assert.Equal(t, true, err != nil)
		//attempt to tamper with the total amount
		case 4:
			numTokenInputs, numFeeInputs, numFeePayments := common.RandIntInterval(0, 255), common.RandIntInterval(0, 255), common.RandIntInterval(0, 255)
			_, txTokenConversionParams, err := createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, 2, tokenID)
			assert.Equal(t, nil, err, "createTokenConversionParams returns an error: %v", err)

			txTokenConversionParams.tokenParams.tokenPayments[0].Amount += uint64(common.RandIntInterval(1, 1000))
			//Test validate txTokenConversionParams
			err = validateTxTokenConvertVer1ToVer2Params(txTokenConversionParams)
			assert.Equal(t, true, err != nil)
			// fmt.Println(err)
		//testing as usual
		default:
			numTokenInputs, numFeeInputs, numFeePayments := common.RandIntInterval(0, 100), common.RandIntInterval(0, 100), common.RandIntInterval(0, 100)
			_, txTokenConversionParams, err := createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, 2, tokenID)
			assert.Equal(t, nil, err, "createTokenConversionParams returns an error: %v", err)

			//Test validate txTokenConversionParams
			err = validateTxTokenConvertVer1ToVer2Params(txTokenConversionParams)
			assert.Equal(t, nil, err, "validateTxTokenConvertVer1ToVer2Params returns an error: %v", err)
		}

	}
}

func TestInitializeTxTokenConversion(t *testing.T) {
	//prepare some fake keysets and fake tokens to save on database
	fmt.Println("====== INIT TEST =====")
	numKeySets := 1
	numTokens := 1
	numCoinsPerKeySet := 1

	//generate sample keysets
	keySets, err := prepareKeySets(numKeySets)
	assert.Equal(t, nil, err, "prepareKeySets returns an errors: %v", err)

	//create and save some PRV coins
	coins, err := createAndSaveTokens(numCoinsPerKeySet, common.PRVCoinID, keySets, testDB, 2)
	assert.Equal(t, nil, err, "createAndSaveTokens returns an error: %v", err)

	//init some token
	tokenIDs := make([]*common.Hash, numTokens)
	for i := 0; i < numTokens; i++ {
		// tokenID := &common.Hash{common.RandBytes(1)[0]}
		tokenName := "Token" + string(i)
		theInputCoin, ok := coins[i].(coin.Coin)
		assert.Equal(t, true, ok, "Cannot parse coin")
		paramToCreateTx, tokenParam, err := createInitTokenParams(theInputCoin, testDB, "", tokenName, keySets[0])

		assert.Equal(t, nil, err, fmt.Sprintf("createInitTokenParams failed with error %v", err))
		if err != nil{
			return
		}

		tx := &TxToken{}

		err = tx.Init(paramToCreateTx)
		if err != nil {
			jsb, _ := json.Marshal(coins[i])
			fmt.Printf("Loop %d : Init returns an error: %v\nThat coin is %s\n", i, err, string(jsb))
			return
		}

		tokenOutputs := tx.GetTxNormal().GetProof().GetOutputCoins()
		feeOutputs := tx.GetTxBase().GetProof().GetOutputCoins()
		forceSaveCoins(testDB, feeOutputs, 0, common.PRVCoinID, t)
		statedb.StorePrivacyToken(testDB, *tx.GetTokenID(), tokenParam.PropertyName, tokenParam.PropertySymbol, statedb.InitToken, tokenParam.Mintable, tokenParam.Amount, []byte{}, *tx.Hash())

		fmt.Printf("Token %d : ID %s\n", i, hex.EncodeToString(tx.GetTokenID()[:]))
		tokenExisted := statedb.PrivacyTokenIDExisted(testDB, *tx.GetTokenID())
		assert.Equal(t, true, tokenExisted)

		statedb.StoreCommitments(testDB, *tx.GetTokenID(), [][]byte{tokenOutputs[0].GetCommitment().ToBytesS()}, shardID)
		tokenIDs[i] = tx.GetTokenID()
	}

	fmt.Println("====== INIT TEST FINISHED =====")

	for i := 0; i < numTests; i++ {
		randomTokenIndex := common.RandInt() % numTokens
		tokenID := tokenIDs[randomTokenIndex]
		assert.Equal(t, true, statedb.PrivacyTokenIDExisted(testDB, *tokenID))

		fmt.Printf("Test #%d: Convert token %d\n", i, randomTokenIndex)
		numTokenInputs, numFeeInputs, numFeePayments := common.RandIntInterval(1, 80), common.RandIntInterval(1, 10), common.RandIntInterval(1, 3)
		mySupposedlyPersonalKey, txTokenConversionParams, err := createTokenConversionParams(numTokenInputs, numFeeInputs, numFeePayments, 2, tokenID)
		assert.Equal(t, nil, err, "createTokenConversionParams returns an error: %v", err)

		txToken := new(TxToken)
		err = InitTokenConversion(txToken, txTokenConversionParams)
		assert.Equal(t, nil, err, "initTokenConversion returns an error: %v", err)
		if err != nil {
			return
		}

		isValidSanity, err := txToken.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)

		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = false
		boolParams["isNewTransaction"] = true

		// validate signatures, proofs, etc. Only do after sanity checks are passed
		isValidTxItself, err := txToken.ValidateTxByItself(boolParams, testDB, nil, nil, shardID, nil, nil)
		assert.Equal(t, true, isValidTxItself)
		assert.Equal(t, nil, err)

		_ = mySupposedlyPersonalKey
		testProveVerifyTxTokenConversionTampered(mySupposedlyPersonalKey, txToken, txTokenConversionParams, tokenID, t)
		// res, err := txToken.ValidateTransaction(false, testDB, bridgeDB, 0, &common.PRVCoinID, false, true)
		// assert.Equal(t, true, res, "ValidateTransaction returns an error: %v", err)

	}
}

func testProveVerifyTxTokenConversionTampered(decryptingKey *incognitokey.KeySet, txConversionOutput *TxToken, convParam *TxTokenConvertVer1ToVer2InitParams, tokenID *common.Hash, t *testing.T) {
	m := common.RandInt()
	testDB := convParam.stateDB
	txPrivacyParams := tx_generic.NewTxPrivacyInitParams(
		convParam.senderSK, convParam.feePayments, convParam.feeInputs, convParam.fee,
		false, convParam.stateDB, &common.PRVCoinID, convParam.metaData, convParam.info,
	)
	var err error
	txInner, ok := txConversionOutput.GetTxNormal().(*Tx)
	assert.Equal(t, true, ok)

	switch m % 5 { //Change this if you want to test a specific case

	//tamper with fee
	case 0:
		fmt.Println("------------------Tampering with fee-------------------")
		txConversionOutput.GetTxBase().SetTxFee(uint64(common.RandIntInterval(0, 1000)))

		//Re-sign transaction
		// txInner.Sig, txInner.SigPubKey, err = tx_generic.SignNoPrivacy(convParam.senderSK, txInner.Hash()[:])
		// assert.Equal(t, nil, err)
		// txConversionOutput.SetTxNormal(txInner)
		resignUnprovenTxToken([]*incognitokey.KeySet{decryptingKey}, txConversionOutput, nil, txPrivacyParams)

		// fmt.Println(err)
		// if err!=nil{
		// 	return
		// }
		// assert.Equal(t, nil, err)
		// sig, sigPubKey, err := signNoPrivacy(senderSk, txConversionOutput.Hash()[:])
		// txConversionOutput.SetSig(sig)
		// txConversionOutput.SetSigPubKey(sigPubKey)
		// assert.Equal(t, nil, err)

	//tamper with randomness
	case 1:
		fmt.Println("------------------Tampering with randomness-------------------")
		// txNormal is of type `convert`, so its inputs' randomness field matters
		// meanwhile txFee has version 2, which means feeInput's randomness becomes irrelevant after the proof is made
		bothInputCoins := txConversionOutput.GetTxTokenData().TxNormal.GetProof().GetInputCoins()
		usedIndex := common.RandInt() % len(bothInputCoins)
		bothInputCoins[usedIndex].SetRandomness(operation.RandomScalar())

		//Re-sign transaction
		txInner.Sig, txInner.SigPubKey, err = tx_generic.SignNoPrivacy(convParam.senderSK, txInner.Hash()[:])
		assert.Equal(t, nil, err)
		resignUnprovenTxToken([]*incognitokey.KeySet{decryptingKey}, txConversionOutput, nil, txPrivacyParams)

	//attempt to convert used coins (used serial numbers)
	case 2:
		fmt.Println("------------------Tampering with serial numbers-------------------")
		bothInputCoins := append(txConversionOutput.Tx.GetProof().GetInputCoins(), txConversionOutput.GetTxNormal().GetProof().GetInputCoins()...)
		usedIndex := common.RandInt() % len(bothInputCoins)
		inputCoin := bothInputCoins[usedIndex]

		// these errors are not important
		err := statedb.StoreSerialNumbers(testDB, *tokenID, [][]byte{inputCoin.GetKeyImage().ToBytesS()}, 0)
		err = statedb.StoreSerialNumbers(testDB, common.PRVCoinID, [][]byte{inputCoin.GetKeyImage().ToBytesS()}, 0)
		assert.Equal(t, true, err == nil)

	//tamper with OTAs
	case 3:
		fmt.Println("------------------Tampering with OTAs-------------------")
		bothOutputCoins := append(txConversionOutput.Tx.GetProof().GetOutputCoins(), txConversionOutput.GetTxTokenData().TxNormal.GetProof().GetOutputCoins()...)
		usedIndex := common.RandInt() % len(bothOutputCoins)
		outputCoin := bothOutputCoins[usedIndex]

		forceSaveCoins(testDB, []coin.Coin{outputCoin}, 0, *tokenID, t)
		forceSaveCoins(testDB, []coin.Coin{outputCoin}, 0, common.PRVCoinID, t)
		// assert.Equal(t, true, err == nil)

	//tamper with commitment of output
	case 4:
		fmt.Println("------------------Tampering with commitment-------------------")
		//fmt.Println("Tx hash before altered", txConversionOutput.Hash().String())
		bothOutputCoins := append(txConversionOutput.Tx.GetProof().GetOutputCoins(), txConversionOutput.GetTxTokenData().TxNormal.GetProof().GetOutputCoins()...)
		usedIndex := common.RandInt() % len(bothOutputCoins)
		outputCoin := bothOutputCoins[usedIndex]
		outputCoinSpecific, ok := outputCoin.(*coin.CoinV2)
		assert.Equal(t, true, ok)

		//Attempt to alter some output coins!
		outputCoinSpecific.SetCommitment(operation.RandomPoint())
		//Re-sign transaction
		txInner.Sig, txInner.SigPubKey, err = tx_generic.SignNoPrivacy(convParam.senderSK, txInner.Hash()[:])
		assert.Equal(t, nil, err)
		resignUnprovenTxToken([]*incognitokey.KeySet{decryptingKey}, txConversionOutput, nil, txPrivacyParams)
	default:

	}
	//Attempt to verify
	var isValid bool
	isValidAlready, err := txConversionOutput.ValidateSanityData(nil, nil, nil, 0)
	fmt.Printf("First error: %v\n", err)
	if !isValidAlready {
		isValid = false
	} else {
		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = false
		// validate signatures, proofs, etc. Only do after sanity checks are passed
		isValidAlready, err = txConversionOutput.ValidateTxByItself(boolParams, testDB, nil, nil, shardID, nil, nil)
	}
	fmt.Printf("Second error: %v\n", err)
	// assert.Equal(t, false, isValid, "validateConversionVer1ToVer2 on corrupted data should not be true")
	if !isValidAlready {
		isValid = false
	} else {
		err := txConversionOutput.ValidateTxWithBlockChain(nil, nil, nil, shardID, testDB)
		isValid = (err == nil)
	}
	fmt.Printf("Third error: %v\n", err)
	if isValid {
		jsb, _ := json.Marshal(txConversionOutput)
		fmt.Printf("Unexpected error in case %d . Transaction : %s\n", m%5, string(jsb))
		// return
	}
	//This validation should return an error
	assert.Equal(t, false, isValid)
	//Result should be false
	// assert.Equal(t, false, res)

}
