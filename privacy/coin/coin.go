package coin

import (
	// "encoding/json"
	// "fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

type Coin interface {
	GetVersion() uint8
	GetCommitment() *operation.Point
	GetInfo() []byte
	GetPublicKey() *operation.Point
	GetKeyImage() *operation.Point
	GetValue() uint64
	GetRandomness() *operation.Scalar
	GetShardID() (uint8, error)
	GetSNDerivator() *operation.Scalar
	IsEncrypted() bool

	// DecryptOutputCoinByKey process outputcoin to get outputcoin data which relate to keyset
	// Param keyset: (private key, payment address, read only key)
	// in case private key: return unspent outputcoin tx
	// in case read only key: return all outputcoin tx with amount value
	// in case payment address: return all outputcoin tx with no amount value
	Decrypt(IKeySet) (PlainCoin, error)

	Bytes() []byte
	SetBytes([]byte) error

	CheckCoinValid(key.PaymentAddress, []byte, uint64) bool
}

type PlainCoin interface {
	// Overide
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error

	GetVersion() uint8
	GetCommitment() *operation.Point
	GetInfo() []byte
	GetPublicKey() *operation.Point
	GetValue() uint64
	GetKeyImage() *operation.Point
	GetRandomness() *operation.Scalar
	GetShardID() (uint8, error)
	GetSNDerivator() *operation.Scalar
	IsEncrypted() bool

	SetKeyImage(*operation.Point)
	SetPublicKey(*operation.Point)
	SetCommitment(*operation.Point)
	SetInfo([]byte)
	SetValue(uint64)
	SetRandomness(*operation.Scalar)

	// ParseKeyImage as Mlsag specification
	ParseKeyImageWithPrivateKey(key.PrivateKey) (*operation.Point, error)
	ParsePrivateKeyOfCoin(key.PrivateKey) (*operation.Scalar, error)

	ConcealOutputCoin(additionalData interface{}) error

	Bytes() []byte
	SetBytes([]byte) error
}

type IKeySet interface{
	GetPrivateKey() key.PrivateKey
	GetPaymentAddress() key.PaymentAddress
	GetReadOnlyKey() key.ViewingKey
}

// func NewPlainCoinFromByte(b []byte) (PlainCoin, error) {
// 	version := byte(CoinVersion2)
// 	if len(b)>=1{
// 		version = b[0]
// 	}
// 	var c PlainCoin
// 	if version == CoinVersion2 {
// 		c = new(CoinV2)
// 	} else {
// 		c = new(PlainCoinV1)
// 	}
// 	err := c.SetBytes(b)
// 	return c, err
// }

// First byte should determine the version or json marshal "34"
// func NewCoinFromByte(b []byte) (Coin, error) {
// 	coinV1 := new(CoinV1)
// 	coinV2 := new(CoinV2)
// 	if errV2 := json.Unmarshal(b, coinV2); errV2 != nil {
// 		if errV1 := json.Unmarshal(b, coinV1); errV1 != nil {
// 			version := b[0]
// 			if version == CoinVersion2 {
// 				err := coinV2.SetBytes(b)
// 				fmt.Println("New Coin From Byte V2", coinV2.publicKey, err)
// 				return coinV2, err
// 			} else {
// 				err := coinV1.SetBytes(b)
// 				fmt.Println("New Coin From Byte V1", coinV1.CoinDetails.publicKey, err)
// 				return coinV1, err
// 			}
// 		} else {
// 			fmt.Println("New Coin From Byte V1", coinV1.CoinDetails.publicKey)
// 			return coinV1, nil
// 		}
// 	} else {
// 		fmt.Println("New Coin From Byte V2", coinV2.publicKey)
// 		return coinV2, nil
// 	}
// }

// Check whether the utxo is from this address
func IsCoinBelongToViewKey(coin Coin, viewKey key.ViewingKey) bool {
	if coin.GetVersion() == 1 {
		return operation.IsPointEqual(viewKey.GetPublicSpend(), coin.GetPublicKey())
	} else if coin.GetVersion() == 2 {
		c, err := coin.(*CoinV2)
		if err == false {
			return false
		}
		txRandomPoint, index, err1 :=  c.GetTxRandomDetail()
		if err1 != nil {
			return false
		}
		rK := new(operation.Point).ScalarMult(txRandomPoint, viewKey.GetPrivateView())

		hashed := operation.HashToScalar(
			append(rK.ToBytesS(), common.Uint32ToBytes(index)...),
		)
		HnG := new(operation.Point).ScalarMultBase(hashed)
		KCheck := new(operation.Point).Sub(c.GetPublicKey(), HnG)

		return operation.IsPointEqual(KCheck, viewKey.GetPublicSpend())
	} else {
		return false
	}
}

// func ParseCoinsFromBytes(data []json.RawMessage) ([]Coin, error) {
// 	coinList := make([]Coin, len(data))
// 	for i := 0; i < len(data); i++ {
// 		if coin, err := NewCoinFromByte(data[i]); err != nil {
// 			return nil, err
// 		} else {
// 			coinList[i] = coin
// 		}
// 	}
// 	return coinList, nil
// }
