package debugtool

import (
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	zkp "github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
	"github.com/incognitochain/incognito-chain/rpcserver"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
	"io/ioutil"
	math2 "math"
	"math/big"
	"os"
	"time"
	"fmt"
)


//To be refactored later
func CreateTxPrivacyInitParams(db *statedb.StateDB, keySet *incognitokey.KeySet, paymentAddress privacy.PaymentAddress, inputCoins []coin.PlainCoin, hasPrivacy bool, fee, amount uint64, tokenID common.Hash) ([]*privacy.PaymentInfo, *transaction.TxPrivacyInitParams, error) {
	//initialize payment info of input coins
	paymentInfos := make([]*privacy.PaymentInfo, 1)

	paymentInfos[0] = key.InitPaymentInfo(paymentAddress, amount, []byte("I want to change this world"))

	//Calculate overbalance
	sumOutputValue := uint64(0)
	for _, p := range paymentInfos {
		sumOutputValue += p.Amount
	}

	// Calculate sum of all input coins' value
	sumInputValue := uint64(0)
	for _, coin := range inputCoins {
		sumInputValue += coin.GetValue()
	}

	overBalance := sumInputValue - sumOutputValue - fee
	if overBalance < 0 {
		return nil, nil, errors.New("sumInputs less than sumOutputs + fee")
	}

	if overBalance > 0 {
		paymentInfo := new(privacy.PaymentInfo)
		paymentInfo.Amount = overBalance
		paymentInfo.PaymentAddress = keySet.PaymentAddress
		paymentInfo.Message = nil

		paymentInfos = append(paymentInfos, paymentInfo)
	}

	//create privacyinitparam
	txPrivacyInitParam := transaction.NewTxPrivacyInitParams(&keySet.PrivateKey,
		paymentInfos,
		inputCoins,
		fee,
		hasPrivacy,
		db,
		&tokenID,
		nil,
		nil)

	return paymentInfos, txPrivacyInitParam, nil
}

func CreateCoinFromJSONOutcoin(jsonOutCoin jsonresult.OutCoin) (coin.PlainCoin, *big.Int, error) {
	var output coin.PlainCoin
	var keyImage, pubkey, cm *operation.Point
	var snd, randomness *operation.Scalar
	var info []byte
	var err error
	var idx *big.Int
	var sharedRandom *operation.Scalar
	var txRandom *coin.TxRandom

	value, ok := math.ParseUint64(jsonOutCoin.Value)
	if !ok {
		return nil, nil, errors.New("Cannot parse value")
	}

	if len(jsonOutCoin.KeyImage) == 0 {
		keyImage = nil
	} else {
		keyImageInBytes, err := DecodeBase58Check(jsonOutCoin.KeyImage)
		if err != nil {
			return nil, nil, err
		}
		keyImage, err = new(operation.Point).FromBytesS(keyImageInBytes)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(jsonOutCoin.Commitment) == 0 {
		cm = nil
	} else {
		cmInbytes, err := DecodeBase58Check(jsonOutCoin.Commitment)
		if err != nil {
			return nil, nil, err
		}
		cm, err = new(operation.Point).FromBytesS(cmInbytes)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(jsonOutCoin.PublicKey) == 0 {
		pubkey = nil
	} else {
		pubkeyInBytes, err := DecodeBase58Check(jsonOutCoin.PublicKey)
		if err != nil {
			return nil, nil, err
		}
		pubkey, err = new(operation.Point).FromBytesS(pubkeyInBytes)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(jsonOutCoin.Randomness) == 0 {
		randomness = nil
	} else {
		randomnessInBytes, err := DecodeBase58Check(jsonOutCoin.Randomness)
		if err != nil {
			return nil, nil, err
		}
		randomness = new(operation.Scalar).FromBytesS(randomnessInBytes)
	}

	if len(jsonOutCoin.SNDerivator) == 0 {
		snd = nil
	} else {
		sndInBytes, err := DecodeBase58Check(jsonOutCoin.SNDerivator)
		if err != nil {
			return nil, nil, err
		}
		snd = new(operation.Scalar).FromBytesS(sndInBytes)
	}

	if len(jsonOutCoin.Info) == 0 {
		info = []byte{}
	} else {
		info, err = DecodeBase58Check(jsonOutCoin.Info)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(jsonOutCoin.Index) == 0 {
		idx = nil
	} else {
		idxInBytes, err := DecodeBase58Check(jsonOutCoin.Index)
		if err != nil {
			return nil, nil, err
		}
		idx = new(big.Int).SetBytes(idxInBytes)
	}

	if len(jsonOutCoin.SharedRandom) == 0 {
		sharedRandom = nil
	}else{
		sharedRandomInBytes, err := DecodeBase58Check(jsonOutCoin.SharedRandom)
		if err != nil {
			return nil, nil, err
		}
		sharedRandom = new(operation.Scalar).FromBytesS(sharedRandomInBytes)
	}

	if len(jsonOutCoin.TxRandom) == 0 {
		sharedRandom = nil
	}else{
		sharedRandomInBytes, err := DecodeBase58Check(jsonOutCoin.TxRandom)
		if err != nil {
			return nil, nil, err
		}
		txRandom = new(coin.TxRandom)
		err = txRandom.SetBytes(sharedRandomInBytes)
		if err != nil{
			return nil, nil, err
		}
	}

	switch jsonOutCoin.Version {
	case "1":
		output = new(coin.PlainCoinV1)
		tmpOutput, _ := output.(*coin.PlainCoinV1)
		tmpOutput.SetSNDerivator(snd)

	case "2":
		output = new(coin.CoinV2)
		tmpOutput, _ := output.(*coin.CoinV2)
		tmpOutput.SetTxRandom(txRandom)
		tmpOutput.SetSharedRandom(sharedRandom)
	}

	output.SetCommitment(cm)
	output.SetValue(value)
	output.SetKeyImage(keyImage)
	output.SetRandomness(randomness)
	output.SetPublicKey(pubkey)
	output.SetInfo(info)

	return output, idx, nil
}

func DivideCoins(coins []coin.PlainCoin, listIndices []uint64) ([]coin.PlainCoin, []coin.PlainCoin, []uint64, []uint64) {
	coinV1s := []coin.PlainCoin{}
	coinV2s := []coin.PlainCoin{}
	listIndicesV1 := []uint64{}
	listIndicesV2 := []uint64{}

	for i, coin := range coins {
		if coin.GetVersion() == 1 {
			coinV1s = append(coinV1s, coin)
			listIndicesV1 = append(listIndicesV1, listIndices[i])
		} else {
			coinV2s = append(coinV2s, coin)
			listIndicesV2 = append(listIndicesV2, listIndices[i])
		}
	}

	return coinV1s, coinV2s, listIndicesV1, listIndicesV2
}

func ChooseCoinsToSpend(coins []coin.PlainCoin, amount uint64, overbalanced bool) ([]coin.PlainCoin, error) {
	sumValue := uint64(0)
	res := []coin.PlainCoin{}
	for _, coin := range coins {
		sumValue += coin.GetValue()

		if sumValue > amount {
			res = append(res, coin)
		}
	}
	if sumValue < amount && !overbalanced {
		return nil, errors.New("not enough coins to spend")
	}

	return res, nil
}

func CreateSampleCoin(privKey privacy.PrivateKey, pubKey *privacy.Point, amount uint64, msg []byte, version int) (coin.Coin, error) {
	if version == 1 {
		c := new(coin.PlainCoinV1).Init()

		c.SetValue(amount)
		c.SetInfo(msg)
		c.SetPublicKey(pubKey)
		c.SetSNDerivator(privacy.RandomScalar())
		c.SetRandomness(privacy.RandomScalar())

		//Derive serial number from snDerivator
		c.SetKeyImage(new(privacy.Point).Derive(privacy.PedCom.G[0], new(privacy.Scalar).FromBytesS(privKey), c.GetSNDerivator()))

		//Create commitment
		err := c.CommitAll()

		if err != nil {
			return nil, err
		}

		cOut := new(coin.CoinV1).Init()
		cOut.CoinDetails = c

		return cOut, nil
	} else {
		c := new(coin.CoinV2).Init()

		c.SetValue(amount)
		c.SetInfo(msg)
		c.SetPublicKey(pubKey)
		c.SetRandomness(privacy.RandomScalar())

		keyImage, err := c.ParseKeyImageWithPrivateKey(privKey)
		if err != nil {
			return nil, err
		}
		c.SetKeyImage(keyImage)

		return c, nil
	}
}

func CreateSampleCoinsFromTotalAmount(senderSK privacy.PrivateKey, pubkey *privacy.Point, totalAmount uint64, numFeeInputs, version int) ([]coin.Coin, error) {
	coinList := []coin.Coin{}
	tmpAmount := totalAmount / uint64(numFeeInputs)
	for i := 0; i < numFeeInputs-1; i++ {
		amount := tmpAmount - uint64(RandIntInterval(0, int(tmpAmount)/2))
		coin, err := CreateSampleCoin(senderSK, pubkey, amount, nil, version)
		if err != nil {
			return nil, err
		}
		coinList = append(coinList, coin)
		totalAmount -= amount
	}
	coin, err := CreateSampleCoin(senderSK, pubkey, totalAmount, nil, version)
	if err != nil {
		return nil, err
	}
	coinList = append(coinList, coin)

	return coinList, nil
}

func CreateAndSaveCoins(numCoins, numEquals int, privKey privacy.PrivateKey, pubKey *privacy.Point, db *statedb.StateDB, version int, tokenID common.Hash) ([]coin.Coin, error) {
	shardID := byte(0)

	amount := uint64(numCoins * 1000)
	outCoins := []coin.Coin{}
	for i := 0; i < numEquals; i++ {
		coin, err := CreateSampleCoin(privKey, pubKey, 1000, nil, version)
		if err != nil {
			return nil, err
		}
		outCoins = append(outCoins, coin)
	}
	tmpOutCoins, err := CreateSampleCoinsFromTotalAmount(privKey, pubKey, amount, numCoins-numEquals, version)
	for _, coin := range tmpOutCoins {
		outCoins = append(outCoins, coin)
	}
	if err != nil {
		return nil, err
	}

	//save coins and commitment indices onto the database
	commitmentsToBeSaved := [][]byte{}
	coinsToBeSaved := [][]byte{}
	OTAToBeSaved := [][]byte{}
	for _, outCoin := range outCoins {
		coinsToBeSaved = append(coinsToBeSaved, outCoin.Bytes())
		commitmentsToBeSaved = append(commitmentsToBeSaved, outCoin.GetCommitment().ToBytesS())
		OTAToBeSaved = append(OTAToBeSaved, outCoin.GetPublicKey().ToBytesS())
	}
	if version == 1 {
		SNDToBeSaved := [][]byte{}
		for _, outCoin := range outCoins {
			tmpCoin, ok := outCoin.(*coin.CoinV1)
			if !ok {
				return nil, errors.New("cannot parse CoinV1")
			}
			SNDToBeSaved = append(SNDToBeSaved, tmpCoin.GetSNDerivator().ToBytesS())

		}

		err = statedb.StoreSNDerivators(db, tokenID, SNDToBeSaved)
		if err != nil {
			return nil, err
		}

		err = statedb.StoreOutputCoins(db, tokenID, pubKey.ToBytesS(), coinsToBeSaved, shardID)
		if err != nil {
			return nil, err
		}
		err = statedb.StoreCommitments(db, tokenID, commitmentsToBeSaved, shardID)
		if err != nil {
			return nil, err
		}
	} else {
		err = statedb.StoreOTACoinsAndOnetimeAddresses(db, tokenID, 0, coinsToBeSaved, OTAToBeSaved, shardID)
	}

	return outCoins, nil
}

func InitDatabase() (*statedb.StateDB, error) {
	emptyRoot := common.HexToHash(common.HexEmptyRoot)
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_statedb_")
	if err != nil {
		panic(err)
	}
	diskBD, _ := incdb.Open("leveldb", dbPath)
	wrapperDB := statedb.NewDatabaseAccessWarper(diskBD)
	db, err := statedb.NewWithPrefixTrie(emptyRoot, wrapperDB)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InitParam(tx metadata.Transaction, fee uint64, keySet *incognitokey.KeySet, version int8) {
	if version == 1 {
		tmpTx, _ := tx.(*transaction.TxVersion1)
		tmpTx.Fee = fee
		tmpTx.Type = common.TxNormalType
		tmpTx.Metadata = nil
		tmpTx.PubKeyLastByteSender = keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		if tmpTx.LockTime == 0 {
			tmpTx.LockTime = time.Now().Unix()
		}
		tmpTx.Version = version
		tmpTx.Info = nil
	} else {
		tmpTx, _ := tx.(*transaction.TxVersion2)
		tmpTx.Fee = fee
		tmpTx.Type = common.TxNormalType
		tmpTx.Metadata = nil
		tmpTx.PubKeyLastByteSender = keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		if tmpTx.LockTime == 0 {
			tmpTx.LockTime = time.Now().Unix()
		}
		tmpTx.Version = version
		tmpTx.Info = nil
	}

}

func MarshalTransaction(tx metadata.Transaction) ([]byte, error) {
	txBytes, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	result := jsonresult.NewCreateTransactionResult(tx.Hash(), common.EmptyString, txBytes, 0)

	base58Result := result.Base58CheckData
	return []byte(base58Result), nil
}

func PrepareInputCoins(listCoins []coin.PlainCoin, amount uint64, listIndices []uint64, keySet *incognitokey.KeySet) ([]uint64, []coin.PlainCoin, error) {
	coinsToSpend, err := ChooseCoinsToSpend(listCoins, amount, false)
	if err != nil {
		return nil, nil, err
	}

	myListIndices := make([]uint64, len(coinsToSpend))
	for i, c := range coinsToSpend {
		for j, cV2 := range listCoins {
			if operation.IsPointEqual(c.GetPublicKey(), cV2.GetPublicKey()) {
				myListIndices[i] = listIndices[j]
				break
			}
		}
	}

	if len(myListIndices) != len(coinsToSpend) {
		return nil, nil, errors.New("cannot find input coin")
	}

	inputCoins := []coin.PlainCoin{}
	if listCoins[0].GetVersion() == 1{
		for i := 0; i < len(coinsToSpend); i++ {
			tmpCoin := new(coin.PlainCoinV1)

			err = tmpCoin.SetBytes(coinsToSpend[i].Bytes())
			if err != nil {
				return nil, nil, err
			}

			keyImage, err := tmpCoin.ParseKeyImageWithPrivateKey(keySet.PrivateKey)
			if err != nil {
				return nil, nil, err
			}
			tmpCoin.SetKeyImage(keyImage)

			inputCoins = append(inputCoins, tmpCoin)
		}
	}else{
		for i := 0; i < len(coinsToSpend); i++ {
			tmpCoinToSpend, ok := coinsToSpend[i].(*coin.CoinV2)
			if !ok {
				return nil, nil, errors.New("Must be coinV2")
			}

			tmpCoin := new(coin.CoinV2)
			tmpCoin.SetVersion(2)
			tmpCoin.SetCommitment(tmpCoinToSpend.GetCommitment())
			tmpCoin.SetPublicKey(tmpCoinToSpend.GetPublicKey())
			tmpCoin.SetRandomness(tmpCoinToSpend.GetRandomness())
			tmpCoin.SetValue(tmpCoinToSpend.GetValue())
			tmpCoin.SetSharedRandom(tmpCoinToSpend.GetSharedRandom())
			tmpCoin.SetTxRandom(tmpCoinToSpend.GetTxRandom())
			tmpCoin.SetKeyImage(tmpCoinToSpend.GetKeyImage())

			inputCoins = append(inputCoins, tmpCoin)
		}
	}

	return myListIndices, inputCoins, nil
}


//For txver1
func ParseOutputCoinV1s(paymentInfo []*privacy.PaymentInfo) ([]*coin.CoinV1, error) {
	outputCoins := []*coin.CoinV1{}
	for _, pInfo := range paymentInfo {
		sndOut := privacy.RandomScalar()
		tmpCoin := new(coin.CoinV1)
		tmpCoin.CoinDetails = new(coin.PlainCoinV1)
		tmpCoin.CoinDetails.SetValue(pInfo.Amount)
		if len(pInfo.Message) > 0 {
			if len(pInfo.Message) > privacy.MaxSizeInfoCoin {
				return nil, errors.New("size of info is too big")
			}
		}
		tmpCoin.CoinDetails.SetInfo(pInfo.Message)

		PK, err := new(privacy.Point).FromBytesS(pInfo.PaymentAddress.Pk)
		if err != nil {
			return nil, errors.New("cannot parse public key")
		}
		tmpCoin.CoinDetails.SetPublicKey(PK)
		tmpCoin.CoinDetails.SetSNDerivator(sndOut)
		outputCoins = append(outputCoins, tmpCoin)
	}
	return outputCoins, nil
}

func ParseIndicesAndCommitmentsFromJson(jsonRespondInBytes []byte) ([]uint64, []uint64, []*operation.Point, error) {
	var jsonRespond rpcserver.JsonResponse
	err := json.Unmarshal(jsonRespondInBytes, &jsonRespond)
	if err != nil {
		return nil, nil, nil, err
	}

	var msg json.RawMessage
	err = json.Unmarshal(jsonRespond.Result, &msg)
	if err != nil {
		return nil, nil, nil, err
	}

	var result jsonresult.RandomCommitmentResult
	err = json.Unmarshal(msg, &result)
	if err != nil {
		return nil, nil, nil, err
	}

	commitmentIndices := result.CommitmentIndices
	myCommitmentIndices := result.MyCommitmentIndexs

	commitments := make([]*operation.Point, len(result.Commitments))

	for i, cm := range result.Commitments {
		cmInbytes, err := DecodeBase58Check(cm)
		if err != nil {
			return nil, nil, nil, err
		}

		commitments[i], err = new(operation.Point).FromBytesS(cmInbytes)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return commitmentIndices, myCommitmentIndices, commitments, nil
}

func ProveAndSign(witness *zkp.PaymentWitness, paymentInfos []*privacy.PaymentInfo, tx *transaction.TxVersion1, keySet *incognitokey.KeySet) error {
	paymentProof, err := witness.Prove(true, paymentInfos)
	if err != nil {
		return err
	}
	tx.Proof = paymentProof
	sigPrivate := append(keySet.PrivateKey, witness.GetRandSecretKey().ToBytesS()...)
	err1 := tx.Sign(sigPrivate)
	if err1 != nil {
		return errors.New(err1.Error())
	}
	return nil
}


//For txver2
func ParseOutputCoinV2s(paymentInfo []*privacy.PaymentInfo) ([]*coin.CoinV2, error) {
	outputCoins := []*coin.CoinV2{}
	for _, pInfo := range paymentInfo {
		c, err := coin.NewCoinFromPaymentInfo(pInfo)
		if err != nil {
			return nil, err
		}
		outputCoins = append(outputCoins, c)
	}
	return outputCoins, nil
}

func CreatePrivKeyMlsag(inputCoins []coin.PlainCoin, outputCoins []*coin.CoinV2, senderSK *key.PrivateKey) ([]*operation.Scalar, error) {
	sumRand := new(operation.Scalar).FromUint64(0)
	for _, in := range inputCoins {
		sumRand.Add(sumRand, in.GetRandomness())
	}
	for _, out := range outputCoins {
		sumRand.Sub(sumRand, out.GetRandomness())
	}

	privKeyMlsag := make([]*operation.Scalar, len(inputCoins)+1)
	for i := 0; i < len(inputCoins); i += 1 {
		var err error
		privKeyMlsag[i], err = inputCoins[i].ParsePrivateKeyOfCoin(*senderSK)
		if err != nil {
			return nil, err
		}
	}
	privKeyMlsag[len(inputCoins)] = sumRand
	return privKeyMlsag, nil
}

func GenerateMLSAGWithIndices(inputCoins []coin.PlainCoin, outputCoins []*coin.CoinV2, fee uint64, pi int, commitmentIndices []uint64, myIndices []uint64, commitments []*operation.Point, publicKeys []*operation.Point) (*mlsag.Ring, [][]*big.Int){
	sumOutputsWithFee := CalculateSumOutputsWithFee(coin.CoinV2ArrayToCoinArray(outputCoins), fee)
	ringSize := privacy.RingSize
	indices := make([][]*big.Int, ringSize)
	ring := make([][]*operation.Point, ringSize)

	for i := 0; i < ringSize; i += 1 {
		sumInputs := new(operation.Point).Identity()
		sumInputs.Sub(sumInputs, sumOutputsWithFee)

		row := make([]*operation.Point, len(inputCoins))
		rowIndices := make([]*big.Int, len(inputCoins))
		if i == pi {
			for j := 0; j < len(inputCoins); j += 1 {
				row[j] = inputCoins[j].GetPublicKey()
				rowIndices[j] = new(big.Int).SetUint64(myIndices[j])
				sumInputs.Add(sumInputs, inputCoins[j].GetCommitment())
			}
		} else {
			for j := 0; j < len(inputCoins); j += 1 {
				//r := common.RandInt() % len(commitmentIndices)
				r := i*len(inputCoins)+j
				rowIndices[j] = new(big.Int).SetUint64(commitmentIndices[r])
				row[j] = publicKeys[r]
				sumInputs.Add(sumInputs, commitments[r])
			}
		}
		row = append(row, sumInputs)
		ring[i] = row
		indices[i] = rowIndices
	}

	return mlsag.NewRing(ring), indices
}

func CalculateSumOutputsWithFee(outputCoins []coin.Coin, fee uint64) *operation.Point {
	sumOutputsWithFee := new(operation.Point).Identity()
	for i := 0; i < len(outputCoins); i += 1 {
		sumOutputsWithFee.Add(sumOutputsWithFee, outputCoins[i].GetCommitment())
	}
	feeCommitment := new(operation.Point).ScalarMult(
		operation.PedCom.G[operation.PedersenValueIndex],
		new(operation.Scalar).FromUint64(fee),
	)
	sumOutputsWithFee.Add(sumOutputsWithFee, feeCommitment)
	return sumOutputsWithFee
}

func ParseCommitmentsAndPublicKeysFromJson(jsonRespondInBytes []byte) ([]uint64, []*operation.Point, []*operation.Point, error) {
	var jsonRespond rpcserver.JsonResponse
	err := json.Unmarshal(jsonRespondInBytes, &jsonRespond)
	if err != nil {
		return nil, nil, nil, err
	}

	var msg json.RawMessage
	err = json.Unmarshal(jsonRespond.Result, &msg)
	if err != nil {
		return nil, nil, nil, err
	}

	var result jsonresult.RandomCommitmentAndPublicKeyResult
	err = json.Unmarshal(msg, &result)
	if err != nil {
		return nil, nil, nil, err
	}

	commitmentIndices := result.CommitmentIndices
	publicKeys := make([]*operation.Point, len(result.PublicKeys))
	commitments := make([]*operation.Point, len(result.Commitments))

	if len(commitmentIndices) != len(publicKeys) || len(commitmentIndices) != len(commitments) || len(commitments) != len(publicKeys) {
		return nil, nil, nil, errors.New("length mismatch!")
	}

	for i, cm := range result.Commitments {
		cmInbytes, err := DecodeBase58Check(cm)
		if err != nil {
			return nil, nil, nil, err
		}

		pkInbytes, err := DecodeBase58Check(result.PublicKeys[i])
		if err != nil {
			return nil, nil, nil, err
		}

		commitments[i], err = new(operation.Point).FromBytesS(cmInbytes)
		if err != nil {
			return nil, nil, nil, err
		}
		publicKeys[i], err = new(operation.Point).FromBytesS(pkInbytes)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return commitmentIndices, publicKeys, commitments, nil
}


/*Debug tool functions*/
//For txver1
func (tool *DebugTool) InitTxVer1(tx *transaction.TxVersion1, keyWallet *wallet.KeyWallet, paymentString string, inputCoins []coin.PlainCoin, amount uint64, tokenIDString string) error {
	senderPaymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	keySet := &keyWallet.KeySet

	fee := uint64(100)

	walletReceiver, err := wallet.Base58CheckDeserialize(paymentString)
	if err != nil {
		return err
	}

	paymentInfos, _, err := CreateTxPrivacyInitParams(nil, keySet, walletReceiver.KeySet.PaymentAddress, inputCoins, true, fee, amount, common.PRVCoinID)
	if err != nil {
		return err
	}
	paymentInfos[0].Amount = amount - fee

	InitParam(tx, fee, keySet, 1)

	witness, err1 := tool.InitPaymentWitness(tokenIDString, senderPaymentAddress, inputCoins, paymentInfos, keySet, fee)
	if err1 != nil {
		return err1
	}

	err2 := ProveAndSign(witness, paymentInfos, tx, keySet)
	if err2 != nil {
		return err2
	}
	return nil
}

func (this *DebugTool) GetRandomCommitment(tokenID, paymentAddress string, input []coin.PlainCoin) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	coins := []jsonresult.OutCoin{}
	for _, outCoin := range input {
		var tmpCoin jsonresult.OutCoin
		if outCoinV1, ok := outCoin.(*coin.PlainCoinV1); ok {
			tmpOutCoin := new(coin.CoinV1).Init()
			tmpOutCoin.CoinDetails = outCoinV1
			tmpCoin = jsonresult.NewOutCoin(tmpOutCoin)
		}else if outCoinV2, ok := outCoin.(*coin.CoinV2); ok {
			tmpCoin = jsonresult.NewOutCoin(outCoinV2)
		}

		tmpCoin.Info = base58.Base58Check{}.Encode([]byte(""), common.ZeroByte)
		coins = append(coins, tmpCoin)
	}

	r := new(rpcserver.JsonRequest)
	r.Id = ""
	r.Jsonrpc = "1.0"
	r.Method = "randomcommitments"

	r.Params = []interface{}{paymentAddress, coins, tokenID}

	rInBytes, err := json.MarshalIndent(r, "\t", "\t")
	if err != nil {
		return nil, err
	}

	query := string(rInBytes)

	//fmt.Println(query)
	//return nil, nil
	return this.SendPostRequestWithQuery(query)
}

func (tool *DebugTool) InitPaymentWitness(tokenIDString string, senderPaymentAddress string, inputCoins []coin.PlainCoin, paymentInfos []*privacy.PaymentInfo, keySet *incognitokey.KeySet, fee uint64) (*zkp.PaymentWitness, error) {
	//Get random commitments to create one-of-many proofs
	jsonRespondInBytes, err := tool.GetRandomCommitment(tokenIDString, senderPaymentAddress, inputCoins)
	if err != nil {
		return nil, err
	}

	commitmentIndices, myCommitmentIndices, commitments, err := ParseIndicesAndCommitmentsFromJson(jsonRespondInBytes)
	if err != nil {
		return nil, err
	}

	outputCoins, err := ParseOutputCoinV1s(paymentInfos)
	if err != nil {
		return nil, err
	}

	// PrepareTransaction witness for proving
	paymentWitnessParam := zkp.PaymentWitnessParam{
		HasPrivacy:              true,
		PrivateKey:              new(operation.Scalar).FromBytesS(keySet.PrivateKey),
		InputCoins:              inputCoins,
		OutputCoins:             outputCoins,
		PublicKeyLastByteSender: keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1],
		Commitments:             commitments,
		CommitmentIndices:       commitmentIndices,
		MyCommitmentIndices:     myCommitmentIndices,
		Fee:                     fee,
	}

	witness := new(zkp.PaymentWitness)
	err1 := witness.Init(paymentWitnessParam)
	if err1 != nil {
		return nil, err1
	}
	return witness, nil
}

//For txver2
func (tool *DebugTool) SignTransactionV2(tx *transaction.TxVersion2, tokenIDString string, senderPaymentAddress string, inputCoins []coin.PlainCoin, outputCoins []*coin.CoinV2, fee uint64, listIndicesV2 []uint64, keySet *incognitokey.KeySet) error {
	hashedMessage := tx.Hash()[:]

	jsonRespondInBytes, err := tool.GetRandomCommitmentsAndPublicKeys(tokenIDString, senderPaymentAddress, 100*len(inputCoins))
	if err != nil {
		return err
	}

	commitmentIndices, publicKeys, commitments, err := ParseCommitmentsAndPublicKeysFromJson(jsonRespondInBytes)
	if err != nil {
		return err
	}

	fmt.Println("BUGLOG2 InputCoin publicKey")
	for i, inputCoin := range inputCoins{
		fmt.Println(inputCoin.GetPublicKey(), listIndicesV2[i])
	}

	pi := common.RandInt() % privacy.RingSize
	ring, indices := GenerateMLSAGWithIndices(inputCoins, outputCoins, fee, pi, commitmentIndices, listIndicesV2, commitments, publicKeys)

	fmt.Println("BUGLOG2 ring")
	for _, keys :=range ring.GetKeys(){
		for _, keyImage := range keys{
			fmt.Println("BUGLOG2", keyImage.ToBytesS())
		}
		fmt.Println("BUGLOG2")
	}

	fmt.Println("BUGLOG2 indices in Prove", indices)


	txSigPubKey := new(transaction.TxSigPubKeyVer2)
	txSigPubKey.Indexes = indices
	tx.SigPubKey, err = txSigPubKey.Bytes()
	if err != nil {
		return err
	}

	privKeysMlsag, err := CreatePrivKeyMlsag(inputCoins, outputCoins, &keySet.PrivateKey)
	if err != nil {
		return err
	}
	sag := mlsag.NewMlsag(privKeysMlsag, ring, pi)
	sigPrivate, err := privacy.ArrayScalarToBytes(&privKeysMlsag)
	if err != nil {
		fmt.Println(sigPrivate)
		return err
	}

	// Set Signature
	mlsagSignature, err := sag.Sign(hashedMessage)
	if err != nil {
		return err
	}

	res, err := mlsag.Verify(mlsagSignature, ring, hashedMessage)
	fmt.Println("Verify sig", res, err)

	// inputCoins already hold keyImage so set to nil to reduce size
	mlsagSignature.SetKeyImages(nil)
	tx.Sig, err = mlsagSignature.ToBytes()
	return nil
}

func (this *DebugTool) GetRandomCommitmentsAndPublicKeys(tokenID, paymentAddress string, numOutputs int) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}

	r := new(rpcserver.JsonRequest)
	r.Id = ""
	r.Jsonrpc = "1.0"
	r.Method = "randomcommitmentsandpublickeys"

	r.Params = []interface{}{paymentAddress, numOutputs, tokenID}

	rInBytes, err := json.MarshalIndent(r, "\t", "\t")
	if err != nil {
		return nil, err
	}

	query := string(rInBytes)
	//return nil, nil
	return this.SendPostRequestWithQuery(query)
}

//Common
func (tool *DebugTool) PrepareTransaction(privateKey string, tokenIDString string) (*wallet.KeyWallet, string, *operation.Point, []coin.PlainCoin, []coin.PlainCoin, []uint64, []uint64, error) {
	listOutputCoins, listIndices, err := tool.GetPlainOutputCoin(privateKey, tokenIDString)
	if err != nil {
		return nil, "", nil, nil, nil, nil, nil, err
	}

	keyWallet, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		return nil, "", nil, nil, nil, nil, nil, err
	}

	keySet := &keyWallet.KeySet
	err = keySet.InitFromPrivateKey(&keySet.PrivateKey)
	if err != nil {
		return nil, "", nil, nil, nil, nil, nil, err
	}

	senderPaymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)

	pubkey, err := new(privacy.Point).FromBytesS(keySet.PaymentAddress.Pk)
	if err != nil {
		return nil, "", nil, nil, nil, nil, nil, err
	}

	if err != nil {
		return nil, "", nil, nil, nil, nil, nil, err
	}

	coinV1s, coinV2s, listIndicesV1, listIndicesV2 := DivideCoins(listOutputCoins, listIndices)
	return keyWallet, senderPaymentAddress, pubkey, coinV1s, coinV2s, listIndicesV1, listIndicesV2, nil
}

func (tool *DebugTool) GetPlainOutputCoin(privateKey, tokenID string) ([]coin.PlainCoin, []uint64, error) {
	outputCoins := []coin.PlainCoin{}
	respondInBytes, err := tool.GetListUnspentOutputTokens(privateKey, tokenID)
	if err != nil {
		return nil, nil, err
	}

	respond, err := ParseResponse(respondInBytes)
	if err != nil {
		return nil, nil, err
	}

	var msg json.RawMessage
	err = json.Unmarshal(respond.Result, &msg)
	if err != nil {
		return nil, nil, err
	}

	var tmp jsonresult.ListOutputCoins
	err = json.Unmarshal(msg, &tmp)
	if err != nil {
		return nil, nil, err
	}

	listOutputCoins := tmp.Outputs
	listIndices := make([]uint64, 0)
	for _, value := range listOutputCoins {
		for _, outCoin := range value {
			out, idx, err := CreateCoinFromJSONOutcoin(outCoin)
			if err != nil {
				return nil, nil, err
			}
			outputCoins = append(outputCoins, out)
			if out.GetVersion() == 2{
				if idx == nil {
					return nil, nil, errors.New("Coin must have index to proceed!")
				}
				listIndices = append(listIndices, idx.Uint64())
			}else{
				listIndices = append(listIndices, math2.MaxUint64) //Garbage value - we dont actually need it.
			}

		}
	}

	return outputCoins, listIndices, nil
}


