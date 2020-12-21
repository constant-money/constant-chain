package proof

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/proof/agg_interface"
)

// Paymentproof
type Proof interface {
	GetVersion() uint8
	Init()
	GetInputCoins() []coin.PlainCoin
	GetOutputCoins() []coin.Coin
	GetAggregatedRangeProof() agg_interface.AggregatedRangeProof

	SetInputCoins([]coin.PlainCoin) error
	SetOutputCoins([]coin.Coin) error

	Bytes() []byte
	SetBytes(proofBytes []byte) *errhandler.PrivacyError

	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error

	IsPrivacy() bool
	ValidateSanity(interface{}) (bool, error)

	Verify(boolParams map[string]bool, pubKey key.PublicKey, fee uint64, shardID byte, tokenID *common.Hash, additionalData interface{}) (bool, error)
}

