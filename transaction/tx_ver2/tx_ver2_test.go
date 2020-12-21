package tx_ver2

import (
	"bytes"
	"math/big"
	"testing"
	"fmt"
	"io/ioutil"
	"os"
	"encoding/json"
	"unicode"
	"math/rand"
	"bufio"
	// "time"
	// "strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy"
	// "github.com/incognitochain/incognito-chain/privacy/privacy_v2"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/bulletproofs"
	"github.com/incognitochain/incognito-chain/trie"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
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

	allowModifiedTXsToPass = false
	hasPrivacyForPRV   bool = true
	hasPrivacyForToken bool = true
	shardID            byte = byte(0)

	positiveTestsFileName = "./testdata/accepted.txt"
	negativeTestsFileName = "./testdata/rejected.txt"
	b58 = base58.Base58Check{}
)
// variables for initializing stateDB for test
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
	// activeLogger = inactiveLogger
	activeLogger.SetLevel(common.LevelDebug)
	privacy.LoggerV1.Init(inactiveLogger)
	privacy.LoggerV2.Init(activeLogger)
	// can switch between the 2 loggers to mute logs as one wishes
	utils.Logger.Init(activeLogger)
	bulletproofs.Logger.Init(common.NewBackend(nil).Logger("test", true))
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

func forceSaveCoins(db *statedb.StateDB, coinsToBeSaved []coin.Coin, shardID byte, tokenID common.Hash, t *testing.T){
	coinsInBytes := make([][]byte, 0)
	otas := make([][]byte, 0)
	for _,c := range coinsToBeSaved{
		if t!=nil{
			assert.Equal(t,2,int(c.GetVersion()))
		}
		coinsInBytes = append(coinsInBytes, c.Bytes())
		otas = append(otas, c.GetPublicKey().ToBytesS())
	}
	err := statedb.StoreOTACoinsAndOnetimeAddresses(db, tokenID, 0, coinsInBytes, otas, shardID)
	if t!=nil{
		assert.Equal(t,nil,err)
	}
}

func preparePaymentKeys(count int, t *testing.T){
	// create many random private keys
	// then use each privatekey to derive Incognito keyset (various keys for everything inside the protocol)
	// we ensure they all belong in shard 0 for this test

	// PaymentInfo is like `intent` for making Coin.
	// the paymentInfo slice here will be used to create pastCoins & inputCoins
	// we populate `value` fields with some arbitrary, big-enough constant (here, 4000*len)
	// `message` field can be anything
	dummyPrivateKeys = make([]*key.PrivateKey,count)
	keySets = make([]*incognitokey.KeySet,len(dummyPrivateKeys))
	paymentInfo = make([]*key.PaymentInfo, len(dummyPrivateKeys))
	for i := 0; i < count; i += 1 {
		for{
			privateKey := key.GeneratePrivateKey(common.RandBytes(32))
			dummyPrivateKeys[i] = &privateKey
			keySets[i] = new(incognitokey.KeySet)
			err := keySets[i].InitFromPrivateKey(dummyPrivateKeys[i])
			if t!=nil{
				assert.Equal(t, nil, err)
			}
			paymentInfo[i] = key.InitPaymentInfo(keySets[i].PaymentAddress, uint64(4000*len(dummyPrivateKeys)), []byte("test in"))
			pkb := []byte(paymentInfo[i].PaymentAddress.Pk)
			if common.GetShardIDFromLastByte(pkb[len(pkb)-1])==shardID{
				break
			}
		}
	}
	// fmt.Println("Key & PaymentInfo generation finished")
}

func TestSigPubKeyCreationAndMarshalling(t *testing.T) {
// here m, n are not very specific so we give them generous range
	m := RandInt() % (maxPrivateKeys - minInputs + 1) + minInputs
	n := RandInt() % (maxPrivateKeys - minInputs + 1) + minInputs
	var err error
	for i := 0; i < numOfLoops; i += 1 {
		fmt.Printf("\n------------------TxToken SigPubKey Test\n")
		maxLen := new(big.Int)
		maxLen.SetString("1000000000000000000", 10)
		indexes := make([][]*big.Int, n)
		for i := 0; i < n; i += 1 {
			row := make([]*big.Int, m)
			for j := 0; j < m; j += 1 {
				row[j], err = common.RandBigIntMaxRange(maxLen)
				assert.Equal(t, nil, err, "Should not have any bug when Randomizing Int Max Range")
			}
			indexes[i] = row
		}

		txSig := new(SigPubKey)
		txSig.Indexes = indexes

		b, err := txSig.Bytes()
		assert.Equal(t, nil, err, "Should not have any bug when txSig.ToBytes")

		txSig2 := new(SigPubKey)
		err = txSig2.SetBytes(b)
		assert.Equal(t, nil, err, "Should not have any bug when txSig.FromBytes")

		b2, err := txSig2.Bytes()
		assert.Equal(t, nil, err, "Should not have any bug when txSig2.ToBytes")
		assert.Equal(t, true, bytes.Equal(b, b2))

		n1 := len(txSig.Indexes)
		m1 := len(txSig.Indexes[0])
		n2 := len(txSig2.Indexes)
		m2 := len(txSig2.Indexes[0])

		assert.Equal(t, n1, n2, "Two Indexes length should be equal")
		assert.Equal(t, m1, m2, "Two Indexes length should be equal")
		for i := 0; i < n; i += 1 {
			for j := 0; j < m; j += 1 {
				b1 := txSig.Indexes[i][j].Bytes()
				b2 := txSig2.Indexes[i][j].Bytes()
				assert.Equal(t, true, bytes.Equal(b1, b2), "Indexes[i][j] should be equal for every i j")
			}
		}
	}
	fmt.Println("SigPubKey Marshalling Test successful")
}

// tx salary is just a validator printing block rewards in PRV, without privacy
// no need for dummy input
func TestTxV2Salary(t *testing.T){
	numOfPrivateKeys := 2

	for loop := 0; loop < numOfLoops; loop++ {
		fmt.Printf("\n------------------Tx Salary Test\n")
		var err error
		preparePaymentKeys(numOfPrivateKeys,t)

	// create 2 otaCoins, the second one will already be stored in the db
		theCoins := make([]*coin.CoinV2, 2)
		theCoinsGeneric := make([]coin.Coin,2)
		for i, _ := range theCoins {
			var tempCoin *coin.CoinV2
			var err error
			for{
				tempCoin,err = coin.NewCoinFromPaymentInfo(paymentInfo[i])
				otaPublicKeyBytes := tempCoin.GetPublicKey().ToBytesS()
				// want an OTA in shard 0
				if otaPublicKeyBytes[31]==0{
					break
				}
			}
			assert.Equal(t, nil, err)
			assert.Equal(t, false, tempCoin.IsEncrypted())
			tempCoin.ConcealOutputCoin(keySets[i].PaymentAddress.GetPublicView())
			assert.Equal(t, true, tempCoin.IsEncrypted())
			assert.Equal(t, true, tempCoin.GetSharedRandom() == nil)
			_, err = tempCoin.Decrypt(keySets[i])
			assert.Equal(t,nil,err)
			theCoins[i] = tempCoin
			theCoinsGeneric[i] = tempCoin
		}
		forceSaveCoins(dummyDB, []coin.Coin{theCoinsGeneric[1]}, 0, common.PRVCoinID, t)

		// creating the TX object
		tx := &Tx{}
		// actually making the salary TX
		err = tx.InitTxSalary(theCoins[0], dummyPrivateKeys[0], dummyDB, nil)

		isValid,err := tx.ValidateTxSalary(dummyDB)
		assert.Equal(t,nil,err)
		assert.Equal(t,true,isValid)

		testTxV2JsonMarshaler(tx, 10, dummyDB, t)
		malTx := &Tx{}
		// this other coin is already in db so it must be rejected
		err = malTx.InitTxSalary(theCoins[1], dummyPrivateKeys[0], dummyDB, nil)
		assert.NotEqual(t,nil,err)
	}
}

func TestPrivacyV2TxPRV(t *testing.T){
	numOfPrivateKeys := RandInt() % (maxPrivateKeys - minPrivateKeys + 1) + minPrivateKeys
	numOfInputs := RandInt() % (maxInputs - minInputs + 1) + minInputs
	// dummyDB, _ = statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)
	for loop := 0; loop < numOfLoops; loop++ {
		fmt.Printf("\n------------------Tx Main Test\n")
		fmt.Printf("Number of inputs  : %d\n", numOfInputs)
		fmt.Printf("Number of outputs : %d\n", numOfPrivateKeys)
		var err error
		preparePaymentKeys(numOfPrivateKeys,t)

		// pastCoins are coins we forcefully write into the dummyDB to simulate the db having OTAs in the past
		// we make sure there are a lot - and a lot - of past coins from all those simulated private keys
		pastCoins := make([]coin.Coin, (10+numOfInputs)*len(dummyPrivateKeys))
		for i, _ := range pastCoins {
			tempCoin,err := coin.NewCoinFromPaymentInfo(paymentInfo[i%len(dummyPrivateKeys)])
			// tempCoin, _, err := createUniqueOTACoinCA(paymentInfo[i%len(dummyPrivateKeys)], &common.PRVCoinID, dummyDB)
			assert.Equal(t, nil, err)
			assert.Equal(t, false, tempCoin.IsEncrypted())

			// to obtain a PlainCoin to feed into input of TX, we need to conceal & decrypt it (it makes sure all fields are right, as opposed to just casting the type to PlainCoin)
			tempCoin.ConcealOutputCoin(keySets[i%len(dummyPrivateKeys)].PaymentAddress.GetPublicView())
			assert.Equal(t, true, tempCoin.IsEncrypted())
			assert.Equal(t, true, tempCoin.GetSharedRandom() == nil)
			pastCoins[i] = tempCoin
		}

		// in this test, we randomize the length of inputCoins so we feel safe fixing the length of outputCoins to equal len(dummyPrivateKeys)
		// since the function `tx.Init` takes output's paymentinfo and creates outputCoins inside of it, we only create the paymentinfo here
		paymentInfoOut := make([]*key.PaymentInfo, len(dummyPrivateKeys))
		for i, _ := range dummyPrivateKeys {
			paymentInfoOut[i] = key.InitPaymentInfo(keySets[i].PaymentAddress,uint64(3000*numOfInputs),[]byte("test out"))
			// fmt.Println(paymentInfo[i])
		}

		// use the db's interface to write our simulated pastCoins to the database
		// we do need to re-format the data into bytes first
		forceSaveCoins(dummyDB, pastCoins, 0, common.PRVCoinID, t)


		// now we take some of those stored coins to use as TX input
		// for the TX to be valid, these inputs must associate to one same private key
		// (it's guaranteed by our way of indexing the pastCoins array)
		inputCoins := make([]coin.PlainCoin,numOfInputs)
		for i,_ := range inputCoins{
			var err error
			inputCoins[i],err = pastCoins[i*len(dummyPrivateKeys)].Decrypt(keySets[0])
			assert.Equal(t,nil,err)
		}

		// now we calculate the fee = sum(Input) - sum(Output)
		sumIn := uint64(4000*len(dummyPrivateKeys)*numOfInputs)
		sumOut := uint64(4000*len(dummyPrivateKeys)*numOfInputs)
		assert.Equal(t,true,sumIn >= sumOut)

		initializingParams := tx_generic.NewTxPrivacyInitParams(dummyPrivateKeys[0],
			paymentInfoOut,inputCoins,
			sumIn-sumOut,hasPrivacyForPRV,
			dummyDB,
			&common.PRVCoinID,
			nil,
			[]byte{},
		)

		// initializingParamsAgain := tx_generic.NewTxPrivacyInitParams(dummyPrivateKeys[0],
		// 	paymentInfoOut,inputCoins,
		// 	sumIn-sumOut,hasPrivacyForPRV,
		// 	nil,
		// 	&common.PRVCoinID,
		// 	nil,
		// 	[]byte{},
		// )

		// var inputCoinsAgain [][]byte
		// for _, c := range initializingParams.InputCoins{
		// 	inputCoinsAgain = append(inputCoinsAgain, c.Bytes())
		// }

		// var dumpedPastCoins [][]byte
		// var dumpedIndexes []uint64
		// for _, c := range pastCoins{
		// 	dumpedPastCoins = append(dumpedPastCoins, c.Bytes())
		// 	ind, err := statedb.GetOTACoinIndex(dummyDB, common.PRVCoinID, c.GetPublicKey().ToBytesS())
		// 	assert.Equal(t, nil, err)
		// 	dumpedIndexes = append(dumpedIndexes, ind.Uint64())
		// }

		// creating the TX object
		tx := &Tx{}
		// actually making the TX
		// `Init` function will also create all necessary proofs and attach them to the TX
		err = tx.Init(initializingParams)
		if err != nil{
			panic(err)
		}
		assert.Equal(t,nil,err)

		// verify the TX
		// params : hasPrivacy bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB,
		// 			shardID byte (we're testing with only 1 shard),
		//			tokenID *common.Hash (set to nil, meaning we use PRV),
		//			isBatch bool, isNewTransaction bool
		isValid,err := tx.ValidateSanityData(nil,nil,nil,0)
		assert.Equal(t,nil,err)
		assert.Equal(t,true,isValid)

		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = hasPrivacyForPRV
		boolParams["isNewTransaction"] = true
		// isValid,err = tx.ValidateTransaction(true,dummyDB,nil,0,nil,false,true)
		isValid,err = tx.ValidateTxByItself(boolParams, dummyDB, nil, nil, shardID, nil, nil)
		if err!=nil{
			panic(err)
		}
		assert.Equal(t,nil,err)
		assert.Equal(t,true,isValid)
		err = tx.ValidateTxWithBlockChain(nil, nil, nil, shardID, dummyDB)
		if err!=nil{
			panic(err)
		}
		dumpTransaction(tx, positiveTestsFileName)

		// first, test the json marshaller
		testTxV2JsonMarshaler(tx, 10, dummyDB, t)

		testTxV2DeletedProof(tx, t)
		testTxV2DuplicateInput(dummyDB, inputCoins, paymentInfoOut, t)
		testTxV2InvalidFee(dummyDB, inputCoins, paymentInfoOut, t)
		testTxV2OneFakeInput(tx, dummyDB, initializingParams, pastCoins, t)
		testTxV2OneFakeOutput(tx, dummyDB, initializingParams, paymentInfoOut, t)
		testTxV2OneDoubleSpentInput(dummyDB, inputCoins, paymentInfoOut, pastCoins, t)
		_,_ = isValid,err
	}
}

func dumpTransaction(tx *Tx, filename string){
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	txBytes,_ := json.Marshal(tx)
	b58Encoded := b58.Encode(txBytes, common.ZeroByte)
	file.WriteString(b58Encoded)
	// file.WriteString("\n")
	jsonBytes, _, err := b58.Decode(b58Encoded)
	if err!=nil{
		panic("END")
	}
	_=jsonBytes
}

func TestPremadeTransactions(t *testing.T){
	positiveTestFile, err := os.Open(positiveTestsFileName)
	if err!=nil{
		fmt.Println("Cannot open testdata (+)")
		return
	}
	defer positiveTestFile.Close()
	scanner := bufio.NewScanner(positiveTestFile)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForPRV
	boolParams["isNewTransaction"] = true
    for scanner.Scan() {
        b58EncodedTx := scanner.Text()
        jsonBytes, _, err := b58.Decode(b58EncodedTx)
        assert.Equal(t, nil, err)
    	tx := &Tx{}
    	err = json.Unmarshal(jsonBytes, tx)
    	assert.Equal(t, nil, err)
    	isValid,err := tx.ValidateSanityData(nil,nil,nil,0)
		assert.Equal(t,nil,err)
		assert.Equal(t,true,isValid)
		isValid,err = tx.ValidateTxByItself(boolParams, dummyDB, nil, nil, shardID, nil, nil)
		assert.Equal(t,nil,err)
		assert.Equal(t,true,isValid)
		err = tx.ValidateTxWithBlockChain(nil, nil, nil, shardID, dummyDB)
		assert.Equal(t,nil,err)
	}
    // negative
    negativeTestsFile, err := os.Open(negativeTestsFileName)
    if err!=nil{
		fmt.Println("Cannot open testdata (-)")
		return
	}
	defer negativeTestsFile.Close()
	scanner = bufio.NewScanner(negativeTestsFile)
    for scanner.Scan() {
        b58EncodedTx := scanner.Text()
        jsonBytes, _, err := b58.Decode(b58EncodedTx)
        assert.Equal(t, nil, err)
    	tx := &Tx{}
    	err = json.Unmarshal(jsonBytes, tx)
    	assert.Equal(t, nil, err)
    	isValid,_ := tx.ValidateSanityData(nil,nil,nil,0)
		if !isValid{
			continue
		}
		assert.Equal(t,false,isValid)
		isValid,_ = tx.ValidateTxByItself(boolParams, dummyDB, nil, nil, shardID, nil, nil)
		if !isValid{
			continue
		}
		assert.Equal(t,false,isValid)
		err = tx.ValidateTxWithBlockChain(nil, nil, nil, shardID, dummyDB)
		assert.NotEqual(t,nil,err)
    }
}

func testTxV2DeletedProof(txv2 *Tx, t *testing.T){
	// try setting the proof to nil, then verify
	// it should not go through
	savedProof := txv2.Proof
	txv2.Proof = nil
	isValid,err := txv2.ValidateSanityData(nil,nil,nil,0)
	assert.NotEqual(t,nil,err)
	assert.Equal(t,false,isValid)
	txv2.Proof = savedProof
}

func testTxV2DuplicateInput(db *statedb.StateDB, inputCoins []coin.PlainCoin, paymentInfoOut []*key.PaymentInfo, t *testing.T){
	dup := &coin.CoinV2{}
	dup.SetBytes(inputCoins[0].Bytes())
	// used the same coin twice in inputs
	malInputCoins := append(inputCoins,dup)
	malFeeParams := tx_generic.NewTxPrivacyInitParams(dummyPrivateKeys[0],
		paymentInfoOut,malInputCoins,
		10,true,
		db,
		nil,
		nil,
		[]byte{},
	)
	malTx := &Tx{}
	errMalInit := malTx.Init(malFeeParams)
	assert.Equal(t,nil,errMalInit)
	// sanity should be fine
	isValid,err := malTx.ValidateSanityData(nil,nil,nil,0)
	assert.Equal(t,nil,err)
	assert.Equal(t,true,isValid)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForPRV
	boolParams["isNewTransaction"] = true
	// validate should reject due to Verify() in PaymentProofV2
	isValid,_ = malTx.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	assert.Equal(t,false,isValid)
	dumpTransaction(malTx, negativeTestsFileName)
}

func testTxV2InvalidFee(db *statedb.StateDB, inputCoins []coin.PlainCoin, paymentInfoOut []*key.PaymentInfo, t *testing.T){
	// a set of init params where sum(Input) < fee + sum(Output)
	// let's say someone tried to use this invalid fee for tx
	// we should encounter an error here
	sumIn := uint64(4000*len(dummyPrivateKeys)*len(inputCoins))
	sumOut := uint64(3000*len(paymentInfoOut))
	assert.Equal(t,true,sumIn > sumOut)
	malFeeParams := tx_generic.NewTxPrivacyInitParams(dummyPrivateKeys[0],
		paymentInfoOut,inputCoins,
		sumIn-sumOut+1111,true,
		db,
		nil,
		nil,
		[]byte{},
	)
	malTx := &Tx{}
	_ = malTx.Init(malFeeParams)
	// if errMalInit!=nil{
	// 	panic(errMalInit)
	// }
	// assert.NotEqual(t,nil,errMalInit)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForPRV
	boolParams["isNewTransaction"] = true
	isValid,errMalVerify := malTx.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	assert.NotEqual(t,nil,errMalVerify)
	fmt.Println(errMalVerify)
	assert.Equal(t,false,isValid)
}

func testTxV2OneFakeInput(txv2 *Tx, db *statedb.StateDB, params *tx_generic.TxPrivacyInitParams, pastCoins []coin.Coin, t *testing.T){
	// likewise, if someone took an already proven tx and swaps one input coin
	// for another random coin from outside, the tx cannot go through
	// (here we only meddle with coin-changing - not adding/removing - since length checks are included within mlsag)
	var err error
	theProof := txv2.GetProof()
	inputCoins := theProof.GetInputCoins()
	numOfInputs := len(inputCoins)
	changed := RandInt() % numOfInputs
	saved := inputCoins[changed]
	inputCoins[changed],_ = pastCoins[len(dummyPrivateKeys)*(numOfInputs+1)].Decrypt(keySets[0])
	theProof.SetInputCoins(inputCoins)

	err = resignUnprovenTx(keySets, txv2, params, nil, false)
	assert.Equal(t,nil,err)
	isValid,err := txv2.ValidateSanityData(nil,nil,nil,0)
	assert.Equal(t,nil,err)
	assert.Equal(t,true,isValid)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForPRV
	boolParams["isNewTransaction"] = true
	isValid,err = txv2.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	// should fail at signature since mlsag needs commitments from inputs
	activeLogger.Debugf("TEST RESULT : One faked valid input -> %v",err)
	assert.Equal(t,false,isValid)
	inputCoins[changed] = saved
	theProof.SetInputCoins(inputCoins)
	err = resignUnprovenTx(keySets, txv2, params, nil, false)
	isValid,err = txv2.ValidateSanityData(nil,nil,nil,0)
	assert.Equal(t,nil,err)
	assert.Equal(t,true,isValid)
	isValid,err = txv2.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	assert.Equal(t,nil,err)
	assert.Equal(t,true,isValid)


	// saved = inputCoins[changed]
	// inputCoins[changed] = nil
	// malTx.GetProof().SetInputCoins(inputCoins)
	// isValid,err = malTx.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, byte(0), true, nil, nil)
	// // verify must fail
	// assert.NotEqual(t,nil,err)
	// assert.Equal(t,false,isValid)
	// inputCoins[changed] = saved
}

func testTxV2OneFakeOutput(txv2 *Tx, db *statedb.StateDB, params *tx_generic.TxPrivacyInitParams, paymentInfoOut []*key.PaymentInfo, t *testing.T){
	// similar to the above. All these verifications should fail
	var err error
	outs := txv2.GetProof().GetOutputCoins()
	prvOutput,ok := outs[0].(*coin.CoinV2)
	savedCoinBytes := prvOutput.Bytes()
	assert.Equal(t,true,ok)
	prvOutput.Decrypt(keySets[0])
	// set amount to something wrong
	prvOutput.SetValue(6996)
	prvOutput.SetSharedRandom(operation.RandomScalar())
	prvOutput.ConcealOutputCoin(keySets[0].PaymentAddress.GetPublicView())
	err = resignUnprovenTx(keySets, txv2, params, nil, false)
	assert.Equal(t,nil,err)
	isValid,err := txv2.ValidateSanityData(nil,nil,nil,0)
	assert.Equal(t,nil,err)
	assert.Equal(t,true,isValid)

	boolParams := make(map[string]bool)
	boolParams["hasPrivacy"] = hasPrivacyForPRV
	boolParams["isNewTransaction"] = true
	isValid,err = txv2.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	// verify must fail
	assert.Equal(t,false,isValid)
	// fmt.Printf("Fake output (wrong amount) -> %v\n",err)
	dumpTransaction(txv2, negativeTestsFileName)
	// undo the tampering
	prvOutput.SetBytes(savedCoinBytes)
	outs[0] = prvOutput
	txv2.GetProof().SetOutputCoins(outs)
	err = resignUnprovenTx(keySets, txv2, params, nil, false)
	assert.Equal(t,nil,err)


	isValid,_ = txv2.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	assert.Equal(t,true,isValid)

	// now instead of changing amount, we change the OTA public key
	theProof := txv2.GetProof()
	outs = theProof.GetOutputCoins()
	prvOutput,ok = outs[0].(*coin.CoinV2)
	savedCoinBytes = prvOutput.Bytes()
	assert.Equal(t,true,ok)
	payInf := paymentInfoOut[0]
	// totally fresh OTA of the same amount, meant for the same PaymentAddress
	newCoin,err  := coin.NewCoinFromPaymentInfo(payInf)
	assert.Equal(t,nil,err)
	newCoin.ConcealOutputCoin(keySets[0].PaymentAddress.GetPublicView())
	theProofSpecific, ok := theProof.(*privacy.ProofV2)
	theBulletProof, ok := theProofSpecific.GetAggregatedRangeProof().(*privacy.AggregatedRangeProofV2)
	cmsv := theBulletProof.GetCommitments()
	cmsv[0] = newCoin.GetCommitment()
	outs[0] = newCoin
	txv2.GetProof().SetOutputCoins(outs)
	err = resignUnprovenTx(keySets, txv2, params, nil, false)
	assert.Equal(t,nil,err)
	isValid,err = txv2.ValidateSanityData(nil,nil,nil,0)
	assert.Equal(t,nil,err)
	assert.Equal(t,true,isValid)


	isValid,err = txv2.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	// verify must fail
	assert.Equal(t,false,isValid)
	// fmt.Printf("Fake output (wrong receiving OTA) -> %v\n",err)
	dumpTransaction(txv2, negativeTestsFileName)
	// undo the tampering
	prvOutput.SetBytes(savedCoinBytes)
	outs[0] = prvOutput
	cmsv[0] = prvOutput.GetCommitment()
	txv2.GetProof().SetOutputCoins(outs)
	err = resignUnprovenTx(keySets, txv2, params, nil, false)
	assert.Equal(t,nil,err)
	isValid,_ = txv2.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
	assert.Equal(t,true,isValid)

}

func testTxV2OneDoubleSpentInput(db *statedb.StateDB, inputCoins []coin.PlainCoin, paymentInfoOut []*key.PaymentInfo, pastCoins []coin.Coin, t *testing.T){
	// similar to the above. All these verifications should fail
		changed := RandInt() % len(inputCoins)
		malInputParams := tx_generic.NewTxPrivacyInitParams(dummyPrivateKeys[0],
			paymentInfoOut,inputCoins,
			1,true,
			db,
			nil,
			nil,
			[]byte{},
		)
		malTx := &Tx{}
		err := malTx.Init(malInputParams)
		assert.Equal(t,nil,err)
		otaBytes := malTx.GetProof().GetInputCoins()[changed].GetKeyImage().ToBytesS()
		statedb.StoreSerialNumbers(db, common.ConfidentialAssetID, [][]byte{otaBytes}, 0)

		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = hasPrivacyForPRV
		boolParams["isBatch"] = true
		boolParams["isNewTransaction"] = true
		isValid,err := malTx.ValidateTxByItself(boolParams, db, nil, nil, byte(0), nil, nil)
		// verify by itself passes
		if err!=nil{
			panic(err)
		}
		assert.Equal(t,nil,err)
		assert.Equal(t,true,isValid)

		// verify with blockchain fails
		err = malTx.ValidateTxWithBlockChain(nil, nil ,nil, 0, db)
		assert.NotEqual(t,nil,err)

}

func testTxV2JsonMarshaler(tx *Tx, count int, db *statedb.StateDB, t *testing.T){
	for i:=0;i<count;i++{
		someInvalidTxs := getCorruptedJsonDeserializedTxs(tx, count, t)
		for _,theInvalidTx := range someInvalidTxs{
			txSpecific, ok := theInvalidTx.(*Tx)
			if !ok{
				continue
			}
			// look for potential panics by calling verify
			isValid, _ := txSpecific.ValidateSanityData(nil, nil, nil, 0)
			// if it doesnt pass sanity then the next validation could panic, it's ok by spec
			if !isValid{
				continue
			}
			isValid, _ = txSpecific.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, shardID, false, nil, nil)
			if !isValid{
				continue
			}
			errAlreadyInChain := txSpecific.ValidateTxWithBlockChain(nil, nil, nil, shardID, db)
			if !allowModifiedTXsToPass && errAlreadyInChain==nil{
				// make sure it's different
				s1 := formatTx(tx)
				s2 := formatTx(txSpecific)
				if bytes.Equal([]byte(s1), []byte(s2)){
					continue
				}
				// the forged TX somehow is valid after all 3 checks, we caught a bug
				fmt.Printf("Original TX : %s\nChanged TX (still valid) : %s\n", s1, s2)
				panic("END TEST : a mal-TX was accepted")
			}
		}
		// look for potential panics by calling verify
		isSane, _ := txSpecific.ValidateSanityData(nil, nil, nil, 0)
		// if it doesnt pass sanity then the next validation could panic, it's ok by spec
		if !isSane{
			continue
		}

		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = hasPrivacyForPRV
		isSane, _ = txSpecific.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
		if !isSane{
			continue
		}
		txSpecific.ValidateTxWithBlockChain(nil, nil, nil, shardID, db)
	}
}

func testTxTokenV2JsonMarshaler(tx *TxToken, count int, db *statedb.StateDB, t *testing.T){
	for i:=0;i<count;i++{
		someInvalidTxs := getCorruptedJsonDeserializedTokenTxs(tx, count, t)
		for _,theInvalidTx := range someInvalidTxs{
			txSpecific, ok := theInvalidTx.(*TxToken)
			if !ok{
				continue
			}
			// look for potential panics by calling verify
			isValid, _ := txSpecific.ValidateSanityData(nil, nil, nil, 0)
			// if it doesnt pass sanity then the next validation could panic, it's ok by spec
			if !isValid{
				continue
			}
			isValid, _ = txSpecific.ValidateTxByItself(hasPrivacyForPRV, db, nil, nil, shardID, false, nil, nil)
			if !isValid{
				continue
			}
			errAlreadyInChain := txSpecific.ValidateTxWithBlockChain(nil, nil, nil, shardID, db)
			if !allowModifiedTXsToPass && errAlreadyInChain==nil{
				// make sure it's different
				s1 := formatTx(tx)
				s2 := formatTx(txSpecific)
				if bytes.Equal([]byte(s1), []byte(s2)){
					continue
				}
				// the forged TX somehow is valid after all 3 checks, we caught a bug
				fmt.Printf("Original TX : %s\nChanged TX (still valid) : %s\n", s1, s2)
				panic("END TEST : a mal-TXTOKEN was accepted")
			}
		}
		// look for potential panics by calling verify
		isSane, _ := txSpecific.ValidateSanityData(nil, nil, nil, 0)
		// if it doesnt pass sanity then the next validation could panic, it's ok by spec
		if !isSane{
			continue
		}
		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = hasPrivacyForPRV

		isSane, _ = txSpecific.ValidateTxByItself(boolParams, db, nil, nil, shardID, nil, nil)
		if !isSane{
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
	s := string(jsonBytesAgain)
	theRunes := []rune(s)
	for i:=0; i<maxJsonChanges; i++{
		// let the changes stack up many times to exhaust more cases
		corruptedIndex := RandInt() % len(theRunes)
		for j:=maxTries;j>0;j--{
			if j==0{
				activeLogger.Warnf("Max changes exceeded with : %s\n",s)
				return result
			}
			if unicode.IsLetter(theRunes[corruptedIndex]) || unicode.IsDigit(theRunes[corruptedIndex]){
				break
			}
			// not letter -> retry
			corruptedIndex = RandInt() % len(theRunes)
		}
		// replace this letter with a random one
		var newRune rune
		if unicode.IsLetter(theRunes[corruptedIndex]){
			newRune = getRandomLetter()
		}else{
			newRune = getRandomDigit()
		}
		if theRunes[corruptedIndex]==newRune{
			// remove that char
			theRunes = append(theRunes[:corruptedIndex], theRunes[corruptedIndex+1:]...)
		}else{
			theRunes[corruptedIndex] = newRune
		}
		temp := &Tx{}
		err := json.Unmarshal([]byte(string(theRunes)), temp)
		if err != nil{
			continue
		}
		result = append(result,temp)
	}
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

	s := string(jsonBytesAgain)
	theRunes := []rune(s)
	for i:=0; i<maxJsonChanges; i++{
		corruptedIndex := RandInt() % len(theRunes)
		for j:=maxTries;j>0;j--{
			if j==0{
				activeLogger.Warnf("Max changes exceeded with : %s\n",s)
				return result
			}
			if unicode.IsLetter(theRunes[corruptedIndex]) || unicode.IsDigit(theRunes[corruptedIndex]){
				break
			}
			// not letter -> retry
			corruptedIndex = RandInt() % len(theRunes)
		}
		// replace this letter with a random one
		var newRune rune
		if unicode.IsLetter(theRunes[corruptedIndex]){
			newRune = getRandomLetter()
		}else{
			newRune = getRandomDigit()
		}
		if theRunes[corruptedIndex]==newRune{
			// remove that char
			theRunes = append(theRunes[:corruptedIndex], theRunes[corruptedIndex+1:]...)
		}else{
			theRunes[corruptedIndex] = newRune
		}
		temp := &TxToken{}
		err := json.Unmarshal([]byte(string(theRunes)), temp)
		if err != nil{
			continue
		}
		result = append(result,temp)
	}
	return result
}

func RandInt() int {
	return rand.Int()
}

func formatTx(tx metadata.Transaction) string{
	jsb, _ := json.Marshal(tx)
	return string(jsb)
}

