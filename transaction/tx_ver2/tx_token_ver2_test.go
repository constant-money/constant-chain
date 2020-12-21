package tx_ver2

import (
	"bytes"
	"fmt"
	// "math/big"
	"testing"
	// "io/ioutil"
	// "os"
	"encoding/json"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	// "github.com/incognitochain/incognito-chain/trie"
	// "github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/stretchr/testify/assert"
)

func TestPrivacyV2TxToken(t *testing.T) {
	for loop := 0; loop < numOfLoops; loop++ {
		fmt.Printf("\n------------------TxToken Main Test\n")
		var err error
		numOfPrivateKeys := 4
		numOfInputs := 2
		dummyDB, _ := statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)
		preparePaymentKeys(numOfPrivateKeys, t)

		pastCoins := make([]coin.Coin, (10+numOfInputs)*len(dummyPrivateKeys))
		for i, _ := range pastCoins {
			tempCoin, err := coin.NewCoinFromPaymentInfo(paymentInfo[i%len(dummyPrivateKeys)])
			assert.Equal(t, nil, err)
			assert.Equal(t, false, tempCoin.IsEncrypted())
			tempCoin.ConcealOutputCoin(keySets[i%len(dummyPrivateKeys)].PaymentAddress.GetPublicView())
			assert.Equal(t, true, tempCoin.IsEncrypted())
			assert.Equal(t, true, tempCoin.GetSharedRandom() == nil)
			pastCoins[i] = tempCoin
		}

		// store a bunch of sample OTA coins in PRV
		forceSaveCoins(dummyDB, pastCoins, 0, common.PRVCoinID, t)

		// sample message to receiver
		msgCipherText := []byte("haha dummy ciphertext")
		paramToCreateTx, tokenParam := getParamsForTxTokenInit(pastCoins[0], dummyDB)
		// create tx for token init
		tx := &TxToken{}

		fmt.Println("Token Init")
		err = tx.Init(paramToCreateTx)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		// convert to JSON string and revert
		jsb, err := json.Marshal(tx)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		txHash := tx.Hash()
		tx1 := new(TxToken)
		json.Unmarshal(jsb, tx1)
		txHash1 := tx1.Hash()
		assert.Equal(t, txHash, txHash1)

		// size checks
		txActualSize := tx.GetTxActualSize()
		assert.Greater(t, txActualSize, uint64(0))
		// sigPubKey := tx.Tx.GetSigPubKey()
		// assert.Equal(t, common.SigPubKeySize, len(sigPubKey))
		// param checks
		inf := tx.GetTxNormal().GetProof().GetOutputCoins()[0].GetInfo()
		assert.Equal(t, true, bytes.Equal([]byte(inf), msgCipherText))
		retrievedFee := tx.GetTxFee()
		assert.Equal(t, uint64(1000), retrievedFee)
		// theAmount := tx.GetTxTokenData().GetAmount()
		// assert.Equal(t, tokenParam.Amount, theAmount)
		isUniqueReceiver, _, uniqueAmount, tokenID := tx.GetTransferData()
		assert.Equal(t, true, isUniqueReceiver)
		assert.Equal(t, tokenParam.Amount, uniqueAmount)
		assert.Equal(t, tx.GetTokenID(), tokenID)

		// sanity check
		isValidSanity, err := tx.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = hasPrivacyForPRV
		boolParams["isBatch"] = false
		// validate signatures, proofs, etc. Only do after sanity checks are passed
		isValidTxItself, err := tx.ValidateTxByItself(boolParams, dummyDB, nil, nil, shardID, nil, nil)
		assert.Equal(t, true, isValidTxItself)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		// check double spend using `blockchain data` in this db
		err = tx.ValidateTxWithBlockChain(nil, nil, nil, shardID, dummyDB)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		testTxTokenV2JsonMarshaler(tx, 10, dummyDB, t)
		testTxTokenV2DeletedProof(tx, dummyDB, t)
		// which other tests can we use here ?

		// save the fee outputs into the db
		// get output coin token from tx
		tokenOutputs := tx.GetTxTokenData().TxNormal.GetProof().GetOutputCoins()
		feeOutputs := tx.GetTxBase().GetProof().GetOutputCoins()
		forceSaveCoins(dummyDB, feeOutputs, 0, common.PRVCoinID, t)

		feeOutputBytesExtracted := feeOutputs[0].Bytes()
		tokenOutputBytesExtracted := tokenOutputs[0].Bytes()

		// tx token transfer
		paramToCreateTx2, tokenParam2 := getParamForTxTokenTransfer(tx, dummyDB, nil, t)
		_ = tokenParam2
		tx2 := &TxToken{}

		fmt.Println("Token Transfer")
		err = tx2.Init(paramToCreateTx2)
		// should fail because db does not have this token yet
		assert.NotEqual(t, nil, err)
		// store the token
		exists := statedb.PrivacyTokenIDExisted(dummyDB, *tx.GetTokenID())
		assert.Equal(t, false, exists)
		statedb.StorePrivacyToken(dummyDB, *tx.GetTokenID(), tokenParam.PropertyName, tokenParam.PropertySymbol, statedb.InitToken, tokenParam.Mintable, tokenParam.Amount, []byte{}, *tx.Hash())

		statedb.StoreCommitments(dummyDB, *tx.GetTokenID(), [][]byte{tokenOutputs[0].GetCommitment().ToBytesS()}, shardID)
		// check it exists
		exists = statedb.PrivacyTokenIDExisted(dummyDB, *tx.GetTokenID())
		assert.Equal(t, true, exists)
		err = tx2.Init(paramToCreateTx2)
		// still fails because the token's `init` coin (10000 T1) is not stored yet
		assert.NotEqual(t, nil, err)
		// add the coin. Tx creation should succeed  now
		forceSaveCoins(dummyDB, tokenOutputs, 0, common.ConfidentialAssetID, t)
		utils.Logger.Init(activeLogger)
		err = tx2.Init(paramToCreateTx2)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		msgCipherText = []byte("doing a transfer")
		assert.Equal(t, true, bytes.Equal(msgCipherText, tx2.GetTxNormal().GetProof().GetOutputCoins()[0].GetInfo()))

		utils.Logger.Log.Warnf("Token Transfer")
		isValidSanity, err = tx2.ValidateSanityData(nil, nil, nil, 0)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		boolParams["hasPrivacy"] = hasPrivacyForToken
		// before the token init tx is written into db, this should not pass
		isValidTxItself, err = tx2.ValidateTxByItself(boolParams, dummyDB, nil, nil, shardID, nil, nil)
		assert.Equal(t, true, isValidTxItself)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		err = tx2.ValidateTxWithBlockChain(nil, nil, nil, shardID, dummyDB)
		assert.Equal(t, nil, err)
		if err != nil {
			panic(err)
		}

		testTxTokenV2JsonMarshaler(tx2, 10, dummyDB, t)

		testTxTokenV2DeletedProof(tx2, dummyDB, t)
		testTxTokenV2InvalidFee(tx2, dummyDB, t)
		testTxTokenV2OneFakeOutput(tx2, dummyDB, paramToCreateTx2, *tx.GetTokenID(), t)
		testTxTokenV2OneDoubleSpentInput(tx2, dummyDB, feeOutputBytesExtracted, tokenOutputBytesExtracted, tx.GetTokenID(), t)

		// testTxTokenV2Salary(tx.GetTokenID(), dummyDB, t)
		// this negative test below is deprecated by the CA spec
		// testTxTokenV2TransferPRV(dummyDB, t)
	}
}

func testTxTokenV2TransferPRV(db *statedb.StateDB, t *testing.T) {
	pastCoins, err := createAndSaveTokens(100, common.PRVCoinID, keySets, db, 2)
	if err != nil {
		panic(err)
	}

	//Store token onto the database
	err = statedb.StorePrivacyToken(db, common.PRVCoinID, common.PRVCoinName, common.PRVCoinName, statedb.InitToken, true, uint64(10000000000000000), nil, common.Hash{101})
	if err != nil {
		panic(err)
	}
	res := statedb.PrivacyTokenIDExisted(db, common.PRVCoinID)
	assert.Equal(t, true, res)


	theInputCoin := pastCoins[:3]

	paramToCreateTx, _, err := createTokenTransferParams(theInputCoin, db, common.PRVCoinID.String(), common.PRVCoinName, common.PRVCoinName, keySets[0])
	if err != nil {
		panic(err)
	}
	paramToCreateTx.TokenParams.TokenTxType = utils.CustomTokenTransfer

	tx := &TxToken{}
	err = tx.Init(paramToCreateTx)
	if err != nil {
		panic(err)
	}

	res, err = tx.ValidateSanityData(nil, nil, nil, 0)
	assert.Equal(t, false, res)
}

func testTxTokenV2DeletedProof(txv2 *TxToken, db *statedb.StateDB, t *testing.T) {
	// try setting the proof to nil, then verify
	// it should not go through
	txn, ok := txv2.GetTxNormal().(*Tx)
	assert.Equal(t, true, ok)
	savedProof := txn.GetProof()
	txn.SetProof(nil)
	txv2.SetTxNormal(txn)
	isValid, _ := txv2.ValidateSanityData(nil, nil, nil, 0)
	assert.Equal(t, true, isValid)
	isValidTxItself, err := txv2.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, shardID, false, nil, nil)
	assert.Equal(t, false, isValidTxItself)
	activeLogger.Infof("TEST RESULT : Missing token proof -> %v",err)
	txn.SetProof(savedProof)
	txv2.SetTxNormal(txn)
	isValidTxItself, _ = txv2.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, shardID, false, nil, nil)
	assert.Equal(t, true, isValidTxItself)

	savedProof = txv2.GetTxBase().GetProof()
	txv2.GetTxBase().SetProof(nil)
	isValid, _ = txv2.ValidateSanityData(nil, nil, nil, 0)
	assert.Equal(t, true, isValid)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForPRV
	boolParams["isBatch"] = false

	isValidTxItself, _ := txv2.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
	assert.Equal(t, false, isValidTxItself)
	activeLogger.Infof("TEST RESULT : Missing PRV proof -> %v",err)
	// undo the tampering
	txv2.GetTxBase().SetProof(savedProof)
	isValidTxItself, _ = txv2.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, shardID, false, nil, nil)
	assert.Equal(t, true, isValidTxItself)
}

func testTxTokenV2InvalidFee(txv2 *TxToken, db *statedb.StateDB, t *testing.T) {
	// a set of init params where fee is changed so mlsag should verify to false
	// let's say someone tried to use this invalid fee for tx
	// we should encounter an error here

	// set fee to increase by 1000PRV
	savedFee := txv2.GetTxBase().GetTxFee()
	txv2.GetTxBase().SetTxFee(savedFee + 1000)

	// sanity should pass
	isValidSanity, err := txv2.ValidateSanityData(nil, nil, nil, 0)
	assert.Equal(t, true, isValidSanity)
	assert.Equal(t, nil, err)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForPRV
	boolParams["isBatch"] = false

	// should reject at signature since fee & output doesn't sum to input
	isValidTxItself, err := txv2.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
	assert.Equal(t, false, isValidTxItself)
	activeLogger.Infof("TEST RESULT : Invalid fee -> %v",err)

	// undo the tampering
	txv2.GetTxBase().SetTxFee(savedFee)
	isValidTxItself, _ = txv2.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
	assert.Equal(t, true, isValidTxItself)
}

func testTxTokenV2OneFakeOutput(txv2 *TxToken, db *statedb.StateDB, params *tx_generic.TxTokenParams, fakingTokenID common.Hash, t *testing.T) {
	// similar to the above. All these verifications should fail
	var err error
	var isValid bool
	txn, ok := txv2.GetTxNormal().(*Tx)
	assert.Equal(t, true, ok)
	outs := txn.Proof.GetOutputCoins()
	tokenOutput, ok := outs[0].(*coin.CoinV2)
	savedCoinBytes := tokenOutput.Bytes()
	assert.Equal(t, true, ok)
	tokenOutput.Decrypt(keySets[0])
	// set amount from 69 to 690
	tokenOutput.SetValue(690)
	tokenOutput.SetSharedRandom(operation.RandomScalar())
	tokenOutput.ConcealOutputCoin(keySets[0].PaymentAddress.GetPublicView())
	txv2.SetTxNormal(txn)
	// here ring is broken so signing will err
	err = resignUnprovenTxToken([]*incognitokey.KeySet{keySets[0]}, txv2, params, nil)
	assert.NotEqual(t, nil, err)
	// isValid, err = txv2.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, 0, false, nil, nil)
	// verify must fail
	// assert.Equal(t, false, isValid)
	activeLogger.Infof("TEST RESULT : Fake output (wrong amount) -> %v",err)
	// undo the tampering
	tokenOutput.SetBytes(savedCoinBytes)
	outs[0] = tokenOutput
	txn.Proof.SetOutputCoins(outs)
	err = resignUnprovenTxToken([]*incognitokey.KeySet{keySets[0]}, txv2, params, nil)
	assert.Equal(t, nil, err)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = true
	boolParams["isBatch"] = false

	isValid, err = txv2.ValidateTxByItself(boolParams, db, nil, nil, 0, nil, nil)
	assert.Equal(t, true, isValid)

	// now instead of changing amount, we change the OTA public key
	theProof := txn.GetProof()
	outs = theProof.GetOutputCoins()
	tokenOutput, ok = outs[0].(*coin.CoinV2)
	savedCoinBytes = tokenOutput.Bytes()
	assert.Equal(t, true, ok)
	payInf := &privacy.PaymentInfo{PaymentAddress: keySets[0].PaymentAddress, Amount: uint64(69), Message: []byte("doing a transfer")}
	// totally fresh OTA of the same amount, meant for the same PaymentAddress
	// newCoin, err := coin.NewCoinFromPaymentInfo(payInf)
	newCoin, _, err := createUniqueOTACoinCA(payInf, &fakingTokenID, db)
	assert.Equal(t, nil, err)
	newCoin.ConcealOutputCoin(keySets[0].PaymentAddress.GetPublicView())
	theProofSpecific, ok := theProof.(*privacy.ProofV2)
	theBulletProof, ok := theProofSpecific.GetAggregatedRangeProof().(*privacy.AggregatedRangeProofV2)
	cmsv := theBulletProof.GetCommitments()
	cmsv[0] = newCoin.GetCommitment()
	outs[0] = newCoin
	theProof.SetOutputCoins(outs)
	txv2.SetTxNormal(txn)
	err = resignUnprovenTxToken([]*incognitokey.KeySet{keySets[0]}, txv2, params, nil)
	assert.Equal(t, nil, err)
	isValid, err = txv2.ValidateTxByItself(boolParams, db, nil, nil, 0, nil, nil)
	// verify must fail
	assert.Equal(t, false, isValid)
	activeLogger.Infof("Fake output (wrong receiving OTA) -> %v",err)
	// undo the tampering
	tokenOutput.SetBytes(savedCoinBytes)
	outs[0] = tokenOutput
	cmsv[0] = tokenOutput.GetCommitment()
	theProof.SetOutputCoins(outs)
	txv2.SetTxNormal(txn)
	err = resignUnprovenTxToken([]*incognitokey.KeySet{keySets[0]}, txv2, params, nil)
	assert.Equal(t, nil, err)
	isValid, err = txv2.ValidateTxByItself(boolParams, db, nil, nil, 0, nil, nil)
	assert.Equal(t, true, isValid)
}

// happens after txTransfer in test
// we create a second transfer, then try to reuse fee input / token input
func testTxTokenV2OneDoubleSpentInput(tokenTx *TxToken, db *statedb.StateDB, feeOutputBytesExtracted, tokenOutputBytesExtracted []byte, tokenIDExtracted *common.Hash, t *testing.T) {
	// save both fee&token outputs from previous tx
	otaBytes := [][]byte{tokenTx.GetTxNormal().GetProof().GetInputCoins()[0].GetKeyImage().ToBytesS()}
	statedb.StoreSerialNumbers(db, common.ConfidentialAssetID, otaBytes, 0)
	otaBytes = [][]byte{tokenTx.GetTxBase().GetProof().GetInputCoins()[0].GetKeyImage().ToBytesS()}
	statedb.StoreSerialNumbers(db, common.PRVCoinID, otaBytes, 0)

	tokenOutputs := tokenTx.GetTxNormal().GetProof().GetOutputCoins()
	feeOutputs := tokenTx.GetTxBase().GetProof().GetOutputCoins()
	forceSaveCoins(db, feeOutputs, 0, common.PRVCoinID, t)
	forceSaveCoins(db, tokenOutputs, 0, common.ConfidentialAssetID, t)

	// firstly, using the output coins to create new tx should be successful
	utils.Logger.Log.Debugf("Negative test : Double-spending tx for token %s", tokenIDExtracted.String())
	pr, _ := getParamForTxTokenTransfer(tokenTx, db, tokenIDExtracted, t)
	tx := &TxToken{}
	err := tx.Init(pr)
	assert.Equal(t, nil, err)
	isValidSanity, err := tx.ValidateSanityData(nil, nil, nil, 0)
	assert.Equal(t, true, isValidSanity)
	assert.Equal(t, nil, err)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForToken
	boolParams["isBatch"] = false
	isValidTxItself, err := tx.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
	assert.Equal(t, true, isValidTxItself)
	assert.Equal(t, nil, err)
	err = tx.ValidateTxWithBlockChain(nil, nil, nil, 0, db)
	assert.Equal(t, nil, err)

	// now we try to swap in a used input for txfee
	doubleSpendingFeeInput := &coin.CoinV2{}
	doubleSpendingFeeInput.SetBytes(feeOutputBytesExtracted)
	pc, _ := doubleSpendingFeeInput.Decrypt(keySets[0])
	pr.InputCoin = []coin.PlainCoin{pc}
	tx = &TxToken{}
	err = tx.Init(pr)
	assert.Equal(t, nil, err)
	isValidSanity, err = tx.ValidateSanityData(nil, nil, nil, 0)
	assert.Equal(t, true, isValidSanity)
	assert.Equal(t, nil, err)

	boolParams["hasPrivacy"] = hasPrivacyForPRV
	isValidTxItself, err = tx.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
	assert.Equal(t, true, isValidTxItself)
	assert.Equal(t, nil, err)
	err = tx.ValidateTxWithBlockChain(nil, nil, nil, 0, db)
	activeLogger.Infof("Swap with spent Fee Input -> %v", err)
	assert.NotEqual(t, nil, err)

	// now we try to swap in a used token input
	doubleSpendingTokenInput := &coin.CoinV2{}
	doubleSpendingTokenInput.SetBytes(tokenOutputBytesExtracted)
	pc, _ = doubleSpendingTokenInput.Decrypt(keySets[0])
	pr.TokenParams.TokenInput = []coin.PlainCoin{pc}
	tx = &TxToken{}
	err = tx.Init(pr)
	assert.Equal(t, nil, err)
	isValidSanity, err = tx.ValidateSanityData(nil, nil, nil, 0)
	assert.Equal(t, true, isValidSanity)
	assert.Equal(t, nil, err)
	isValidTxItself, err = tx.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
	assert.Equal(t, true, isValidTxItself)
	assert.Equal(t, nil, err)
	err = tx.ValidateTxWithBlockChain(nil, nil, nil, 0, db)
	activeLogger.Infof("Swap with spent Token Input of same TokenID underneath -> %v", err)
	assert.NotEqual(t, nil, err)
	if err == nil {
		fmt.Println(err)
		panic("Test Terminated Early : Double Spent")
	}
}

func getParamForTxTokenTransfer(txTokenInit *TxToken, db *statedb.StateDB, specifiedTokenID *common.Hash, t *testing.T) (*tx_generic.TxTokenParams, *tx_generic.TokenParam) {
	transferAmount := uint64(69)
	msgCipherText := []byte("doing a transfer")
	paymentInfo2 := []*privacy.PaymentInfo{{PaymentAddress: keySets[0].PaymentAddress, Amount: transferAmount, Message: msgCipherText}}

	feeOutputs := txTokenInit.GetTxBase().GetProof().GetOutputCoins()
	prvCoinsToPayTransfer := make([]coin.PlainCoin, 0)
	tokenOutputs := txTokenInit.GetTxTokenData().TxNormal.GetProof().GetOutputCoins()
	tokenCoinsToTransfer := make([]coin.PlainCoin, 0)
	for _, c := range feeOutputs {
		pc, _ := c.Decrypt(keySets[0])
		fmt.Printf("Tx Fee : %x has received %d in PRV\n", pc.GetPublicKey().ToBytesS(), pc.GetValue())
		prvCoinsToPayTransfer = append(prvCoinsToPayTransfer, pc)
	}
	for _, c := range tokenOutputs {
		pc, err := c.Decrypt(keySets[0])
		fmt.Printf("Tx Token : %x has received %d in token T1\n", pc.GetPublicKey().ToBytesS(), pc.GetValue())
		// cv2, _ := pc.(*coin.CoinV2)
		assert.Equal(t, nil, err)
		tokenCoinsToTransfer = append(tokenCoinsToTransfer, pc)
	}
	// token param for transfer token
	if specifiedTokenID==nil{
		specifiedTokenID = txTokenInit.GetTokenID()
	}
	tokenParam2 := &tx_generic.TokenParam{
		PropertyID:     specifiedTokenID.String(),
		PropertyName:   "Token 1",
		PropertySymbol: "T1",
		Amount:         transferAmount,
		TokenTxType:    utils.CustomTokenTransfer,
		Receiver:       paymentInfo2,
		TokenInput:     tokenCoinsToTransfer,
		Mintable:       false,
		Fee:            0,
	}

	paramToCreateTx2 := tx_generic.NewTxTokenParams(&keySets[0].PrivateKey,
		[]*key.PaymentInfo{}, prvCoinsToPayTransfer, 15, tokenParam2, db, nil,
		hasPrivacyForPRV, hasPrivacyForToken, shardID, []byte{}, db)
	return paramToCreateTx2, tokenParam2
}

func getParamsForTxTokenInit(theInputCoin coin.Coin, db *statedb.StateDB) (*tx_generic.TxTokenParams, *tx_generic.TokenParam) {
	msgCipherText := []byte("haha dummy ciphertext")
	initAmount := uint64(10000)
	tokenPayments := []*privacy.PaymentInfo{{PaymentAddress: keySets[0].PaymentAddress, Amount: initAmount, Message: msgCipherText}}

	myOnlyInputCoin, _ := theInputCoin.Decrypt(keySets[0])
	inputCoinsPRV := []coin.PlainCoin{myOnlyInputCoin}
	paymentInfoPRV := []*privacy.PaymentInfo{key.InitPaymentInfo(keySets[0].PaymentAddress, uint64(15000), []byte("test out"))}

	// token param for init new token
	tokenParam := &tx_generic.TokenParam{
		PropertyID:     "",
		PropertyName:   "Token 1",
		PropertySymbol: "T1",
		Amount:         initAmount,
		TokenTxType:    utils.CustomTokenInit,
		Receiver:       tokenPayments,
		TokenInput:     []coin.PlainCoin{},
		Mintable:       false,
		Fee:            0,
	}

	paramToCreateTx := tx_generic.NewTxTokenParams(&keySets[0].PrivateKey,
		paymentInfoPRV, inputCoinsPRV, 1000, tokenParam, db, nil,
		hasPrivacyForPRV, hasPrivacyForToken, shardID, []byte{}, db)
	return paramToCreateTx, tokenParam
}

func testTxTokenV2Salary(tokenID *common.Hash, db *statedb.StateDB, t *testing.T) {
	numOfPrivateKeys := 2
	fmt.Printf("\n------------------TxToken Salary Test\n")
	var err error
	preparePaymentKeys(numOfPrivateKeys, t)

	// create 2 otaCoins, the second one will already be stored in the db
	theCoins := make([]*coin.CoinV2, 2)
	theCoinsGeneric := make([]coin.Coin, 2)
	for i, _ := range theCoins {
		var tempCoin *coin.CoinV2
		var err error
		for {
			tempCoin, err = coin.NewCoinFromPaymentInfo(paymentInfo[i])
			otaPublicKeyBytes := tempCoin.GetPublicKey().ToBytesS()
			// want an OTA in shard 0
			if otaPublicKeyBytes[31] == 0 {
				break
			}
		}
		assert.Equal(t, nil, err)
		assert.Equal(t, false, tempCoin.IsEncrypted())
		// tempCoin.ConcealData(keySets[i].PaymentAddress.GetPublicView())
		// assert.Equal(t, true, tempCoin.IsEncrypted())
		// assert.Equal(t, true, tempCoin.GetSharedRandom() == nil)
		// _, err = tempCoin.Decrypt(keySets[i])
		// assert.Equal(t,nil,err)
		theCoins[i] = tempCoin
		theCoinsGeneric[i] = tempCoin
	}
	forceSaveCoins(db, []coin.Coin{theCoinsGeneric[1]}, 0, *tokenID, t)

	// creating the TX object
	txsal := &TxToken{}
	// actually making the salary TX
	err = txsal.InitTxTokenSalary(theCoins[0], dummyPrivateKeys[0], db, nil, tokenID, "Token 1")
	if err != nil {
		fmt.Println(err)
		panic("Test Terminated Early")
	}
	assert.Equal(t, nil, err)

	testTxTokenV2JsonMarshaler(txsal, 10, db, t)
	// someInvalidTxs := getCorruptedJsonDeserializedTokenTxs(txsal, t)
	// for _,theInvalidTx := range someInvalidTxs{
	// 	txSpecific, ok := theInvalidTx.(*TxToken)
	// 	// assert.Equal(t, true, ok)
	// 	if !ok{
	// 		// it's a txToken but not ver2 for some reason. We ignore for now
	// 		continue
	// 	}
	// 	// look for potential panics by calling verify
	// 	isSane, _ := txSpecific.ValidateSanityData(nil, nil, nil, 0)
	// 	// if it doesnt pass sanity then the next validation could panic, it's ok by spec
	// 	_ = isSane
	// }
	// isValidSanity, err := txsal.ValidateSanityData(nil, nil, nil, 0)
	// assert.Equal(t, true, isValidSanity)
	// assert.Equal(t, nil, err)

	// verify function for txTokenV2Salary is out of scope, so we exit here

}

func resignUnprovenTxToken(decryptingKeys []*incognitokey.KeySet, txToken *TxToken, params *tx_generic.TxTokenParams, nonPrivacyParams *tx_generic.TxPrivacyInitParams) error {
	var err error
	txOuter := &txToken.Tx
	txOuter.SetCachedHash(nil)

	txn, ok := txToken.GetTxNormal().(*Tx)
	if !ok {
		activeLogger.Errorf("Test Error : cast")
		return utils.NewTransactionErr(-1000, nil, "Cast failed")
	}
	txn.SetCachedHash(nil)

	// NOTE : hasPrivacy has been deprecated in the real flow.
	if nonPrivacyParams == nil {
		propertyID, _ := common.TokenStringToHash(params.TokenParams.PropertyID)
		paramsInner := tx_generic.NewTxPrivacyInitParams(
			params.SenderKey,
			params.TokenParams.Receiver,
			params.TokenParams.TokenInput,
			params.TokenParams.Fee,
			true,
			params.TransactionStateDB,
			propertyID,
			nil,
			nil,
		)
		_ = paramsInner
		paramsOuter := tx_generic.NewTxPrivacyInitParams(
			params.SenderKey,
			params.PaymentInfo,
			params.InputCoin,
			params.FeeNativeCoin,
			false,
			params.TransactionStateDB,
			&common.PRVCoinID,
			params.MetaData,
			params.Info,
		)
		err = resignUnprovenTx(decryptingKeys, txOuter, paramsOuter, &txToken.TokenData, false)
		err = resignUnprovenTx(decryptingKeys, txn, paramsInner, nil, true)
		txToken.SetTxNormal(txn)
		txToken.Tx = *txOuter
		return err
	} else {
		paramsOuter := nonPrivacyParams
		err := resignUnprovenTx(decryptingKeys, txOuter, paramsOuter, &txToken.TokenData, false)
		txToken.Tx = *txOuter
		return err
	}

	// txTokenDataHash, err := txToken.TxTokenData.Hash()

}

func resignUnprovenTx(decryptingKeys []*incognitokey.KeySet, tx *Tx, params *tx_generic.TxPrivacyInitParams, tokenData *TxTokenDataVersion2, isCA bool) error {
	tx.SetCachedHash(nil)
	tx.SetSig(nil)
	tx.SetSigPubKey(nil)
	var err error
	outputCoinsGeneric := tx.GetProof().GetOutputCoins()
	var outputCoins []*coin.CoinV2
	// pre-sign, we need unconcealed outputs
	// so receiver privatekeys here are for simulation
	var sharedSecrets []*operation.Point
	for ind, c := range outputCoinsGeneric {
		var dk *incognitokey.KeySet = decryptingKeys[ind % len(decryptingKeys)]
		mySkBytes := dk.PrivateKey[:]
		cv2 := &coin.CoinV2{}
		cv2.SetBytes(c.Bytes())
		cv2.Decrypt(dk)
		sharedSecret, err := cv2.RecomputeSharedSecret(mySkBytes)
		if err!=nil{
			activeLogger.Errorf("TEST : Cannot compute shared secret for coin %v", cv2.Bytes())
			return err
		}
		sharedSecrets = append(sharedSecrets, sharedSecret)
		outputCoins = append(outputCoins, cv2)
	}
	inputCoins := params.InputCoins

	message := tx.Hash()[:]
	if tokenData != nil {
		tdh, err := tokenData.Hash()
		if err != nil {
			panic("Hash failed")
		}
		temp := common.HashH(append(message, tdh[:]...))
		message = temp[:]
	}

	if isCA{
		utils.Logger.Log.Warnf("Re-sign a CA transaction")
		err = tx.signCA(inputCoins, outputCoins, sharedSecrets, params, message[:])
	}else{
		utils.Logger.Log.Warnf("Re-sign a non-CA transaction")
		err = tx.signOnMessage(inputCoins, outputCoins, params, message[:])
	}

	jsb, _ := json.MarshalIndent(tx, "", "\t")
	activeLogger.Debugf("Resigning TX for testing : Rehash message %s\n => %v", string(jsb), tx.Hash())
	return err
}

func createTokenTransferParams(inputCoins []coin.Coin, db *statedb.StateDB, tokenID, tokenName, symbol string, keySet *incognitokey.KeySet) (*tx_generic.TxTokenParams, *tx_generic.TokenParam, error) {
	var err error

	msgCipherText := []byte("Testing Transfer Token")
	transferAmount := uint64(0)
	plainInputCoins := make([]coin.PlainCoin, len(inputCoins))
	for i, inputCoin := range inputCoins {
		plainInputCoins[i], err = inputCoin.Decrypt(keySet)
		if err != nil {
			return nil, nil, err
		}
		if i != 0 {
			transferAmount += plainInputCoins[i].GetValue()
		}
	}

	tokenPayments := []*privacy.PaymentInfo{{PaymentAddress: keySet.PaymentAddress, Amount: transferAmount, Message: msgCipherText}}


	inputCoinsPRV := []coin.PlainCoin{plainInputCoins[0]}
	paymentInfoPRV := []*privacy.PaymentInfo{key.InitPaymentInfo(keySet.PaymentAddress, uint64(10), []byte("test out"))}

	// token param for init new token
	tokenParam := &tx_generic.TokenParam{
		PropertyID:     tokenID,
		PropertyName:   tokenName,
		PropertySymbol: symbol,
		Amount:         transferAmount,
		TokenTxType:    utils.CustomTokenTransfer,
		Receiver:       tokenPayments,
		TokenInput:     plainInputCoins[1:len(inputCoins)],
		Mintable:       false,
		Fee:            0,
	}

	paramToCreateTx := tx_generic.NewTxTokenParams(&keySet.PrivateKey,
		paymentInfoPRV, inputCoinsPRV, 10, tokenParam, db, nil,
		hasPrivacyForPRV, hasPrivacyForToken, shardID, []byte{}, db)
	return paramToCreateTx, tokenParam, nil
}
