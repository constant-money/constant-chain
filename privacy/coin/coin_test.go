package coin

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/stretchr/testify/assert"
	"math/big"

	"github.com/incognitochain/incognito-chain/privacy/key"
	"testing"
)

//func TestIsCoinBelong(t *testing.T) {
//	privateKey0 := key.GeneratePrivateKey([]byte{0})
//	keyset0 := new(incognitokey.KeySet)
//	err := keyset0.InitFromPrivateKey(&privateKey0)
//	assert.Equal(t, nil, err)
//
//	privateKey1 := key.GeneratePrivateKey([]byte{1})
//	keyset1 := new(incognitokey.KeySet)
//	err = keyset1.InitFromPrivateKey(&privateKey1)
//	assert.Equal(t, nil, err)
//
//	paymentInfo0 := key.InitPaymentInfo(keyset0.PaymentAddress, 10, []byte{})
//	c0, err := NewCoinFromPaymentInfo(paymentInfo0)
//	assert.Equal(t, nil, err)
//	assert.Equal(t, false, c0.IsEncrypted())
//	c0.ConcealOutputCoin(keyset0.PaymentAddress.GetPublicView())
//
//	paymentInfo1 := key.InitPaymentInfo(keyset1.PaymentAddress, 10, []byte{})
//	c1, err := NewCoinFromPaymentInfo(paymentInfo1)
//	assert.Equal(t, nil, err)
//	assert.Equal(t, false, c1.IsEncrypted())
//	c1.ConcealOutputCoin(keyset1.PaymentAddress.GetPublicView())
//
//	assert.Equal(t, true, IsCoinBelongToViewKey(c0, keyset0.ReadonlyKey))
//	assert.Equal(t, true, IsCoinBelongToViewKey(c1, keyset1.ReadonlyKey))
//	assert.Equal(t, false, IsCoinBelongToViewKey(c0, keyset1.ReadonlyKey))
//	assert.Equal(t, false, IsCoinBelongToViewKey(c1, keyset0.ReadonlyKey))
//}


// TEST VER 2

func getRandomCoinV2() *CoinV2 {
	c := new(CoinV2)

	c.version = uint8(2)
	c.mask = operation.RandomScalar()
	c.amount = operation.RandomScalar()
	c.txRandom = NewTxRandom()
	c.publicKey = operation.RandomPoint()
	c.commitment = operation.RandomPoint()
	c.info = []byte{1, 2, 3, 4, 5}
	return c
}

func TestCoinV2BytesAndSetBytes(t *testing.T) {
	for i := 0; i < 5; i += 1 {
		// test byte-marshalling of random plain coins
		coin := getRandomCoinV2()
		b := coin.Bytes()
		fmt.Println("CoinBytes =", b)
		coinByBytes := new(CoinV2).Init()
		err := coinByBytes.SetBytes(b)
		assert.Equal(t, nil, err, "Set Bytes should not have any error")
		assert.Equal(t, coin.GetVersion(), coinByBytes.GetVersion(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.GetRandomness().ToBytesS(), coinByBytes.GetRandomness().ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.amount.ToBytesS(), coinByBytes.amount.ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.publicKey.ToBytesS(), coinByBytes.publicKey.ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.commitment.ToBytesS(), coinByBytes.commitment.ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.info, coinByBytes.info, "FromBytes then SetBytes should be equal")

		rConceal, rOTA, i, err := coin.GetTxRandomDetail()
		rConcealPrime, rOTAPrime, iPrime, errPrime := coinByBytes.GetTxRandomDetail()
		assert.Equal(t, err, nil)
		assert.Equal(t, errPrime, nil)
		assert.Equal(t, true, operation.IsPointEqual(rOTAPrime, rOTA))
		assert.Equal(t, true, operation.IsPointEqual(rConcealPrime, rConceal))
		assert.Equal(t, i, iPrime)


		// test byte-marshalling of concealed coins
		privateKey := key.GeneratePrivateKey([]byte{byte(i)})
		keyset := new(incognitokey.KeySet)
		err = keyset.InitFromPrivateKey(&privateKey)
		assert.Equal(t, nil, err)
		paymentInfo := key.InitPaymentInfo(keyset.PaymentAddress, 3000, []byte{})
		coin,err = NewCoinFromPaymentInfo(paymentInfo)
		coin.ConcealOutputCoin(keyset.PaymentAddress.GetPublicView())
		b = coin.Bytes()
		coinByBytes = new(CoinV2).Init()
		err = coinByBytes.SetBytes(b)
		assert.Equal(t, nil, err, "Set Bytes should not have any error")
		assert.Equal(t, coin.GetVersion(), coinByBytes.GetVersion(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.GetRandomness().ToBytesS(), coinByBytes.GetRandomness().ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.amount.ToBytesS(), coinByBytes.amount.ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.publicKey.ToBytesS(), coinByBytes.publicKey.ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.commitment.ToBytesS(), coinByBytes.commitment.ToBytesS(), "FromBytes then SetBytes should be equal")
		assert.Equal(t, coin.info, coinByBytes.info, "FromBytes then SetBytes should be equal")
		rConceal, rOTA, i, err = coin.GetTxRandomDetail()
		rConcealPrime, rOTAPrime, iPrime, errPrime = coinByBytes.GetTxRandomDetail()
		assert.Equal(t, err, nil)
		assert.Equal(t, errPrime, nil)
		assert.Equal(t, true, operation.IsPointEqual(rOTAPrime, rOTA))
		assert.Equal(t, true, operation.IsPointEqual(rConcealPrime, rConceal))
		assert.Equal(t, i, iPrime)
	}
}

func TestCoinV2CreateCoinAndDecrypt(t *testing.T) {
	for i := 0; i < 20; i += 1 {
		privateKey := key.GeneratePrivateKey([]byte{byte(i)})
		keyset := new(incognitokey.KeySet)
		err := keyset.InitFromPrivateKey(&privateKey)
		assert.Equal(t, nil, err)

		r := common.RandBytes(8)
		val, errB := common.BytesToUint64(r)
		assert.Equal(t, nil, errB)

		paymentInfo := key.InitPaymentInfo(keyset.PaymentAddress, val, []byte{})

		c, err := NewCoinFromPaymentInfo(paymentInfo)
		assert.Equal(t, val, c.GetValue())
		assert.Equal(t, nil, err)
		assert.Equal(t, false, c.IsEncrypted())

		// Conceal
		err = c.ConcealOutputCoin(keyset.PaymentAddress.GetPublicView())
		if err != nil{
			panic(err)
		}
		assert.Equal(t, true, c.IsEncrypted())
		assert.Equal(t, true, c.GetSharedRandom() == nil)
		assert.NotEqual(t, val, c.GetValue())

		// ensure tampered coins fail to decrypt
		testCoinV2ConcealedTampered(c,keyset,t)

		var pc PlainCoin
		pc, err = c.Decrypt(keyset)
		assert.Equal(t, nil, err)
		assert.Equal(t, false, pc.IsEncrypted())
		assert.Equal(t, val, c.GetValue())
	}
}

func testCoinV2ConcealedTampered(c *CoinV2, ks *incognitokey.KeySet, t *testing.T){
	saved := c.GetAmount()
	c.SetAmount(operation.RandomScalar())
	_, err := c.Decrypt(ks)
	assert.NotEqual(t, nil, err)

	// fmt.Println(err)
	c.SetAmount(saved)

	saved = c.GetRandomness()
	c.SetRandomness(operation.RandomScalar())
	_, err = c.Decrypt(ks)
	assert.NotEqual(t, nil, err)
	// fmt.Println(err)
	c.SetRandomness(saved)
}

func TestTxRandomGroup(t *testing.T) {
	for i := 0; i < 5; i += 1 {
		group := NewTxRandom()
		r := operation.RandomPoint()
		i := uint32(common.RandInt() & ((1 << 32) - 1))
		group.SetTxConcealRandomPoint(r)
		group.SetIndex(i)

		rPrime, err := group.GetTxOTARandomPoint()
		assert.Equal(t, err, nil)
		assert.Equal(t, true, operation.IsPointEqual(rPrime, r))

		iPrime, err := group.GetIndex()
		assert.Equal(t, err, nil)
		assert.Equal(t, i, iPrime)

		b := group.Bytes()
		var group2 TxRandom
		err = group2.SetBytes(b)
		assert.Equal(t, nil, err)
		rPrime, err = group.GetTxOTARandomPoint()
		assert.Equal(t, err, nil)
		assert.Equal(t, true, operation.IsPointEqual(rPrime, r))

		iPrime, err = group.GetIndex()
		assert.Equal(t, err, nil)
		assert.Equal(t, i, iPrime)
	}
}

func TestCoinV2_IsCoinBelongToKeySet(t *testing.T) {
	privateKey := operation.RandomScalar().ToBytesS()
	keySet := new(incognitokey.KeySet)
	err := keySet.InitFromPrivateKeyByte(privateKey)
	assert.Equal(t, nil, err, "InitFromPrivateKeyByte returns an error: %v", err)
	assert.Equal(t, true, operation.IsPointEqual(keySet.PaymentAddress.GetOTAPublicKey(), new(operation.Point).ScalarMultBase(keySet.OTAKey.GetOTASecretKey())))
	assert.Equal(t, true, operation.IsPointEqual(keySet.PaymentAddress.GetPublicView(), new(operation.Point).ScalarMultBase(keySet.ReadonlyKey.GetPrivateView())))
	assert.Equal(t, true, operation.IsPointEqual(keySet.PaymentAddress.GetPublicSpend(), new(operation.Point).ScalarMultBase(new(operation.Scalar).FromBytesS(keySet.PrivateKey))))


	paymentInfo := key.PaymentInfo{Amount: 10000, PaymentAddress: keySet.PaymentAddress, Message: []byte("Hello world")}

	tmpCoinV2, err := NewCoinFromPaymentInfo(&paymentInfo)
	assert.Equal(t, nil, err)

	possessed, _ := tmpCoinV2.DoesCoinBelongToKeySet(keySet)
	assert.Equal(t, true, possessed)

}

// TEST COIN VER 1

var PedCom operation.PedersenCommitment = operation.PedCom

//func TestCoinV1CommitAllThenSwitchV2(t *testing.T) {
//	coin := new(PlainCoinV1).Init()
//	seedKey := operation.RandomScalar().ToBytesS()
//	privateKey := key.GeneratePrivateKey(seedKey)
//	publicKey, err := new(operation.Point).FromBytesS(key.GeneratePublicKey(privateKey))
//
//	assert.Equal(t, nil, err)
//
//	// init other fields for coin
//	coin.SetPublicKey(publicKey)
//	coin.SetSNDerivator(operation.RandomScalar())
//	coin.SetRandomness(operation.RandomScalar())
//	coin.SetValue(new(big.Int).SetBytes(common.RandBytes(2)).Uint64())
//	coin.SetKeyImage(new(operation.Point).Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.GetSNDerivator()))
//	coin.SetInfo([]byte("Incognito chain"))
//
//	err = coin.CommitAll()
//	assert.Equal(t, nil, err)
//
//	allcm := coin.GetCommitment()
//	cm := ParseCommitmentToV2WithCoin(coin)
//
//	shardID, shardIDerr := coin.GetShardID()
//	assert.Equal(t, nil, shardIDerr)
//
//	allcm = ParseCommitmentToV2(
//		allcm,
//		coin.GetPublicKey(),
//		coin.GetSNDerivator(),
//		shardID,
//	)
//
//	b1 := allcm.ToBytesS()
//	b2 := cm.ToBytesS()
//	assert.Equal(t, true, bytes.Equal(b1, b2))
//}

func TestCoinV1CommitAll(t *testing.T) {
	for i := 0; i < 3; i++ {
		coin := new(PlainCoinV1).Init()
		seedKey := operation.RandomScalar().ToBytesS()
		privateKey := key.GeneratePrivateKey(seedKey)
		publicKey := key.GeneratePublicKey(privateKey)

		// init other fields for coin
		coin.publicKey.FromBytesS(publicKey)

		coin.snDerivator = operation.RandomScalar()
		coin.randomness = operation.RandomScalar()
		coin.value = new(big.Int).SetBytes(common.RandBytes(2)).Uint64()
		coin.serialNumber = new(operation.Point).Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
		coin.CommitAll()
		coin.info = []byte("Incognito chain")

		cmTmp := coin.GetPublicKey()
		shardID, shardIDerr := coin.GetShardID()
		assert.Equal(t, nil, shardIDerr)

		cmTmp.Add(cmTmp, new(operation.Point).ScalarMult(PedCom.G[PedersenValueIndex], new(operation.Scalar).FromUint64(uint64(coin.GetValue()))))
		cmTmp.Add(cmTmp, new(operation.Point).ScalarMult(PedCom.G[PedersenSndIndex], coin.snDerivator))
		cmTmp.Add(cmTmp, new(operation.Point).ScalarMult(PedCom.G[PedersenShardIDIndex], new(operation.Scalar).FromUint64(uint64(shardID))))
		cmTmp.Add(cmTmp, new(operation.Point).ScalarMult(PedCom.G[PedersenRandomnessIndex], coin.GetRandomness()))

		res := operation.IsPointEqual(cmTmp, coin.GetCommitment())
		assert.Equal(t, true, res)
	}
}

func TestCoinMarshalJSON(t *testing.T) {

	for i := 0; i < 3; i++ {
		coin := new(PlainCoinV1).Init()
		seedKey := operation.RandomScalar().ToBytesS()
		privateKey := key.GeneratePrivateKey(seedKey)
		publicKey := key.GeneratePublicKey(privateKey)

		// init other fields for coin
		coin.publicKey.FromBytesS(publicKey)
		coin.snDerivator = operation.RandomScalar()
		coin.randomness = operation.RandomScalar()
		coin.value = uint64(100)
		coin.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
		coin.CommitAll()
		coin.info = []byte("Incognito chain")

		bytesJSON, err := coin.MarshalJSON()
		assert.Equal(t, nil, err)

		coin2 := new(PlainCoinV1)
		err2 := coin2.UnmarshalJSON(bytesJSON)
		assert.Equal(t, nil, err2)
		assert.Equal(t, coin, coin2)
	}
}

/*
	Unit test for Bytes/SetBytes Coin function
*/

func TestCoinBytesSetBytes(t *testing.T) {

	for i := 0; i < 3; i++ {
		coin := new(PlainCoinV1).Init()
		seedKey := operation.RandomScalar().ToBytesS()
		privateKey := key.GeneratePrivateKey(seedKey)
		publicKey := key.GeneratePublicKey(privateKey)

		// init other fields for coin
		coin.publicKey.FromBytesS(publicKey)
		coin.snDerivator = operation.RandomScalar()
		coin.randomness = operation.RandomScalar()
		coin.value = uint64(100)
		coin.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
		coin.CommitAll()
		coin.info = []byte("Incognito chain")

		// convert coin object to bytes array
		coinBytes := coin.Bytes()

		assert.Greater(t, len(coinBytes), 0)

		// new coin object and set bytes from bytes array
		coin2 := new(PlainCoinV1)
		err := coin2.SetBytes(coinBytes)

		assert.Equal(t, nil, err)
		assert.Equal(t, coin, coin2)
	}
}

func TestCoinBytesSetBytesWithMissingFields(t *testing.T) {
	for i := 0; i < 3; i++ {
		coin := new(PlainCoinV1).Init()
		seedKey := operation.RandomScalar().ToBytesS()
		privateKey := key.GeneratePrivateKey(seedKey)
		publicKey := key.GeneratePublicKey(privateKey)

		// init other fields for coin
		coin.publicKey.FromBytesS(publicKey)
		coin.snDerivator = operation.RandomScalar()
		coin.randomness = operation.RandomScalar()
		coin.value = uint64(100)
		coin.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
		//coin.CommitAll()
		coin.info = []byte("Incognito chain")

		// convert coin object to bytes array
		coinBytes := coin.Bytes()

		assert.Greater(t, len(coinBytes), 0)

		// new coin object and set bytes from bytes array
		coin2 := new(PlainCoinV1).Init()
		err := coin2.SetBytes(coinBytes)

		assert.Equal(t, nil, err)
		assert.Equal(t, coin, coin2)
	}
}

func TestCoinBytesSetBytesWithInvalidBytes(t *testing.T) {
	// init coin with fully fields
	// init public key
	coin := new(PlainCoinV1).Init()
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)
	publicKey := key.GeneratePublicKey(privateKey)

	// init other fields for coin
	coin.publicKey.FromBytesS(publicKey)
	coin.snDerivator = operation.RandomScalar()
	coin.randomness = operation.RandomScalar()
	coin.value = uint64(100)
	coin.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
	coin.CommitAll()
	coin.info = []byte("Incognito chain")

	// convert coin object to bytes array
	coinBytes := coin.Bytes()
	assert.Greater(t, len(coinBytes), 0)

	// edit coinBytes
	coinBytes[len(coinBytes)-2] = byte(12)

	// new coin object and set bytes from bytes array
	coin2 := new(PlainCoinV1).Init()
	err := coin2.SetBytes(coinBytes)

	assert.Equal(t, nil, err)
	assert.NotEqual(t, coin, coin2)
}

func TestCoinBytesSetBytesWithEmptyBytes(t *testing.T) {
	// new coin object and set bytes from bytes array
	coin2 := new(CoinV1).Init()
	err := coin2.SetBytes([]byte{})
	assert.Equal(t, errors.New("coinBytes is empty"), err)
}

/*
	Unit test for Bytes/SetBytes InputCoin function
*/

func TestInputCoinBytesSetBytes(t *testing.T) {
	for i := 0; i < 3; i++ {
		coin := new(PlainCoinV1).Init()
		seedKey := operation.RandomScalar().ToBytesS()
		privateKey := key.GeneratePrivateKey(seedKey)
		publicKey := key.GeneratePublicKey(privateKey)

		// init other fields for coin
		coin.publicKey.FromBytesS(publicKey)

		coin.snDerivator = operation.RandomScalar()
		coin.randomness = operation.RandomScalar()
		coin.value = uint64(100)
		coin.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
		coin.CommitAll()
		coin.info = []byte("Incognito chain")

		// convert coin object to bytes array
		coinBytes := coin.Bytes()

		assert.Greater(t, len(coinBytes), 0)

		// new coin object and set bytes from bytes array
		coin2 := new(PlainCoinV1)
		err := coin2.SetBytes(coinBytes)

		assert.Equal(t, nil, err)
		assert.Equal(t, coin, coin2)
	}
}

func TestInputCoinBytesSetBytesWithMissingFields(t *testing.T) {
	coin := new(PlainCoinV1).Init()
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)
	publicKey := key.GeneratePublicKey(privateKey)

	coin.publicKey.FromBytesS(publicKey)

	coin.snDerivator = operation.RandomScalar()
	coin.randomness = operation.RandomScalar()
	coin.value = uint64(100)
	coin.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
	coin.info = []byte("Incognito chain")

	// convert coin object to bytes array
	coinBytes := coin.Bytes()
	assert.Greater(t, len(coinBytes), 0)

	// new coin object and set bytes from bytes array
	coin2 := new(PlainCoinV1).Init()
	err := coin2.SetBytes(coinBytes)

	assert.Equal(t, nil, err)
	assert.Equal(t, coin, coin2)
}

func TestInputCoinBytesSetBytesWithInvalidBytes(t *testing.T) {
	coin := new(PlainCoinV1).Init()
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)
	publicKey := key.GeneratePublicKey(privateKey)

	coin.publicKey.FromBytesS(publicKey)

	coin.snDerivator = operation.RandomScalar()
	coin.randomness = operation.RandomScalar()
	coin.value = uint64(100)
	coin.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.snDerivator)
	coin.info = []byte("Incognito chain")

	// convert coin object to bytes array
	coinBytes := coin.Bytes()
	assert.Greater(t, len(coinBytes), 0)

	// edit coinBytes
	coinBytes[len(coinBytes)-2] = byte(12)

	// new coin object and set bytes from bytes array
	coin2 := new(PlainCoinV1).Init()
	err := coin2.SetBytes(coinBytes)

	assert.Equal(t, nil, err)
	assert.NotEqual(t, coin, coin2)
}

func TestInputCoinBytesSetBytesWithEmptyBytes(t *testing.T) {
	// new coin object and set bytes from bytes array
	coin2 := new(PlainCoinV1).Init()
	err := coin2.SetBytes([]byte{})
	assert.Equal(t, errors.New("coinBytes is empty"), err)
}

/*
	Unit test for Bytes/SetBytes OutputCoin function
*/
func TestOutputCoinBytesSetBytes(t *testing.T) {
	coin := new(CoinV1).Init()
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)
	publicKey := key.GeneratePublicKey(privateKey)
	paymentAddr := key.GeneratePaymentAddress(privateKey)

	coin.CoinDetails.publicKey.FromBytesS(publicKey)

	coin.CoinDetails.snDerivator = operation.RandomScalar()
	coin.CoinDetails.randomness = operation.RandomScalar()
	coin.CoinDetails.value = uint64(100)
	coin.CoinDetails.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(operation.Scalar).FromBytesS(privateKey), coin.CoinDetails.snDerivator)
	//coin.CoinDetails.CommitAll()
	coin.CoinDetails.info = []byte("Incognito chain")
	coin.Encrypt(paymentAddr.Tk)

	// convert coin object to bytes array
	coinBytes := coin.Bytes()

	assert.Greater(t, len(coinBytes), 0)

	// new coin object and set bytes from bytes array
	coin2 := new(CoinV1)
	err := coin2.SetBytes(coinBytes)

	assert.Equal(t, nil, err)
	assert.Equal(t, coin, coin2)
}

func TestOutputCoinBytesSetBytesWithMissingFields(t *testing.T) {
	coin := new(CoinV1).Init()
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)
	publicKey := key.GeneratePublicKey(privateKey)
	paymentAddr := key.GeneratePaymentAddress(privateKey)

	coin.CoinDetails.publicKey.FromBytesS(publicKey)

	coin.CoinDetails.snDerivator = operation.RandomScalar()
	coin.CoinDetails.randomness = operation.RandomScalar()
	coin.CoinDetails.value = uint64(100)
	//coin.CoinDetails.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(Scalar).FromBytes(SliceToArray(privateKey)), coin.CoinDetails.snDerivator)
	//coin.CoinDetails.CommitAll()
	coin.CoinDetails.info = []byte("Incognito chain")
	coin.Encrypt(paymentAddr.Tk)

	// convert coin object to bytes array
	coinBytes := coin.Bytes()
	assert.Greater(t, len(coinBytes), 0)

	// new coin object and set bytes from bytes array
	coin2 := new(CoinV1).Init()
	err := coin2.SetBytes(coinBytes)

	assert.Equal(t, nil, err)
	assert.Equal(t, coin, coin2)
}

func TestOutputCoinBytesSetBytesWithInvalidBytes(t *testing.T) {
	coin := new(CoinV1).Init()
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)
	publicKey := key.GeneratePublicKey(privateKey)
	paymentAddr := key.GeneratePaymentAddress(privateKey)

	coin.CoinDetails.publicKey.FromBytesS(publicKey)

	coin.CoinDetails.snDerivator = operation.RandomScalar()
	coin.CoinDetails.randomness = operation.RandomScalar()
	coin.CoinDetails.value = uint64(100)
	//coin.CoinDetails.serialNumber = PedCom.G[0].Derive(PedCom.G[0], new(Scalar).FromBytes(SliceToArray(privateKey)), coin.CoinDetails.snDerivator)
	//coin.CoinDetails.CommitAll()
	coin.CoinDetails.info = []byte("Incognito chain")
	coin.Encrypt(paymentAddr.Tk)

	// convert coin object to bytes array
	coinBytes := coin.Bytes()
	assert.Greater(t, len(coinBytes), 0)

	// edit coinBytes
	coinBytes[len(coinBytes)-2] = byte(12)

	// new coin object and set bytes from bytes array
	coin2 := new(CoinV1).Init()
	err := coin2.SetBytes(coinBytes)

	assert.Equal(t, nil, err)
	assert.NotEqual(t, coin, coin2)
}

func TestOutputCoinBytesSetBytesWithEmptyBytes(t *testing.T) {
	// new coin object and set bytes from bytes array
	coin2 := new(CoinV1).Init()
	err := coin2.SetBytes([]byte{})

	assert.Equal(t, errors.New("coinBytes is empty"), err)
}

func debugInterface(a interface{}) {
	d, _ := json.Marshal(a)
	fmt.Println(string(d))
}

/*
	Unit test for Encrypt/Decrypt OutputCoin
*/
func TestOutputCoinEncryptDecrypt(t *testing.T) {
	// prepare key
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)

	keySet := new(incognitokey.KeySet)
	err := keySet.InitFromPrivateKey(&privateKey)
	assert.Equal(t, nil, err)

	paymentAddress := key.GeneratePaymentAddress(privateKey)

	for i := 0; i < 3; i++ {
		// new output coin with value and randomness
		coin := new(CoinV1).Init()
		coin.CoinDetails.randomness = operation.RandomScalar()
		coin.CoinDetails.value = new(big.Int).SetBytes(common.RandBytes(2)).Uint64()
		coin.CoinDetails.publicKey.FromBytesS(paymentAddress.Pk)

		// encrypt output coins
		err := coin.Encrypt(paymentAddress.Tk)
		assert.Equal(t, (*errhandler.PrivacyError)(nil), err)

		// convert output coin to bytes array
		coinBytes := coin.Bytes()

		// create new output coin to test
		coin2 := new(CoinV1)
		err2 := coin2.SetBytes(coinBytes)
		assert.Equal(t, nil, err2)

		decrypted, err3 := coin2.Decrypt(keySet)
		assert.Equal(t, nil, err3)

		assert.Equal(t, coin.CoinDetails.randomness, decrypted.GetRandomness())
		assert.Equal(t, coin.CoinDetails.value, decrypted.GetValue())
	}
}

func TestOutputCoinEncryptDecryptWithUnmatchedKey(t *testing.T) {
	// prepare key
	seedKey := operation.RandomScalar().ToBytesS()
	privateKey := key.GeneratePrivateKey(seedKey)

	keySet := new(incognitokey.KeySet)
	err := keySet.InitFromPrivateKey(&privateKey)
	assert.Equal(t, nil, err)

	paymentAddress := key.GeneratePaymentAddress(privateKey)

	// new output coin with value and randomness
	coin := new(CoinV1).Init()
	coin.CoinDetails.randomness = operation.RandomScalar()
	coin.CoinDetails.value = new(big.Int).SetBytes(common.RandBytes(2)).Uint64()
	coin.CoinDetails.publicKey.FromBytesS(paymentAddress.Pk)

	// encrypt output coins
	err = coin.Encrypt(paymentAddress.Tk)
	assert.Equal(t, (*errhandler.PrivacyError)(nil), err)

	// convert output coin to bytes array
	coinBytes := coin.Bytes()

	// create new output coin to test
	coin2 := new(CoinV1)
	err2 := coin2.SetBytes(coinBytes)
	assert.Equal(t, nil, err2)

	// edit receiving key to be unmatched with transmission key
	keySet.ReadonlyKey.Rk[0] = 12
	decrypted, err3 := coin2.Decrypt(keySet)
	assert.Equal(t, nil, err3)
	assert.NotEqual(t, coin.CoinDetails.randomness, decrypted.GetRandomness())
	assert.NotEqual(t, coin.CoinDetails.value, decrypted.GetValue())
}
