package jsonresult

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/transaction"
)

type TransactionDetail struct {
	BlockHash   string `json:"BlockHash"`
	BlockHeight uint64 `json:"BlockHeight"`
	TxSize      uint64 `json:"TxSize"`
	Index       uint64 `json:"Index"`
	ShardID     byte   `json:"ShardID"`
	Hash        string `json:"Hash"`
	Version     int8   `json:"Version"`
	Type        string `json:"Type"` // Transaction type
	LockTime    string `json:"LockTime"`
	Fee         uint64 `json:"Fee"` // Fee applies: always consant
	Image       string `json:"Image"`

	IsPrivacy       bool          `json:"IsPrivacy"`
	Proof           privacy.Proof `json:"Proof"`
	ProofDetail     ProofDetail   `json:"ProofDetail"`
	InputCoinPubKey string        `json:"InputCoinPubKey"`
	SigPubKey       string        `json:"SigPubKey,omitempty"` // 64 bytes
	Sig             string        `json:"Sig,omitempty"`       // 64 bytes

	Metadata                      string      `json:"Metadata"`
	CustomTokenData               string      `json:"CustomTokenData"`
	PrivacyCustomTokenID          string      `json:"PrivacyCustomTokenID"`
	PrivacyCustomTokenName        string      `json:"PrivacyCustomTokenName"`
	PrivacyCustomTokenSymbol      string      `json:"PrivacyCustomTokenSymbol"`
	PrivacyCustomTokenData        string      `json:"PrivacyCustomTokenData"`
	PrivacyCustomTokenProofDetail ProofDetail `json:"PrivacyCustomTokenProofDetail"`
	PrivacyCustomTokenIsPrivacy   bool        `json:"PrivacyCustomTokenIsPrivacy"`
	PrivacyCustomTokenFee         uint64      `json:"PrivacyCustomTokenFee"`

	IsInMempool bool `json:"IsInMempool"`
	IsInBlock   bool `json:"IsInBlock"`

	Info string `json:"Info"`
}

func NewTransactionDetail(tx metadata.Transaction, blockHash *common.Hash, blockHeight uint64, index int, shardID byte) (*TransactionDetail, error) {
	var result *TransactionDetail
	blockHashStr := ""
	if blockHash != nil {
		blockHashStr = blockHash.String()
	}
	switch tx.GetType() {
	case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType, common.TxConversionType:
		{
			var sigPubKeyStr string
			txVersion := tx.GetVersion()
			if txVersion == 1 {
				sigPubKeyStr = base58.Base58Check{}.Encode(tx.GetSigPubKey(), 0x0)
			} else {
				sigPubKey := new(transaction.TxSigPubKeyVer2)
				if err := sigPubKey.SetBytes(tx.GetSigPubKey()); err != nil {
					sigPubKeyStr = "[]"
				} else {
					if temp, err := json.Marshal(sigPubKey); err != nil {
						sigPubKeyStr = "[]"
					} else {
						sigPubKeyStr = string(temp)
					}
				}
			}

			result = &TransactionDetail{
				BlockHash:   blockHashStr,
				BlockHeight: blockHeight,
				Index:       uint64(index),
				TxSize:      tx.GetTxActualSize(),
				ShardID:     shardID,
				Hash:        tx.Hash().String(),
				Version:     tx.GetVersion(),
				Type:        tx.GetType(),
				LockTime:    time.Unix(tx.GetLockTime(), 0).Format(common.DateOutputFormat),
				Fee:         tx.GetTxFee(),
				IsPrivacy:   tx.IsPrivacy(),
				Proof:       tx.GetProof(),
				SigPubKey:   sigPubKeyStr,
				Sig:         base58.Base58Check{}.Encode(tx.GetSig(), 0x0),
				Info:        string(tx.GetInfo()),
			}
			inputCoins := result.Proof.GetInputCoins()
			if result.Proof != nil && len(inputCoins) > 0 && inputCoins[0].GetPublicKey() != nil {
				result.InputCoinPubKey = base58.Base58Check{}.Encode(inputCoins[0].GetPublicKey().ToBytesS(), common.ZeroByte)
			}
			meta := tx.GetMetadata()
			if meta != nil {
				metaData, _ := json.MarshalIndent(meta, "", "\t")
				result.Metadata = string(metaData)
			}
			if result.Proof != nil {
				result.ProofDetail.ConvertFromProof(result.Proof)
			}
		}
	case common.TxCustomTokenPrivacyType, common.TxTokenConversionType:
		{
			txToken, ok := tx.(transaction.TransactionToken)
			if !ok {
				return nil, errors.New("cannot detect transaction type")
			}
			txTokenData := transaction.GetTxTokenDataFromTransaction(tx)
			result = &TransactionDetail{
				BlockHash:                blockHashStr,
				BlockHeight:              blockHeight,
				Index:                    uint64(index),
				TxSize:                   tx.GetTxActualSize(),
				ShardID:                  shardID,
				Hash:                     tx.Hash().String(),
				Version:                  tx.GetVersion(),
				Type:                     tx.GetType(),
				LockTime:                 time.Unix(tx.GetLockTime(), 0).Format(common.DateOutputFormat),
				Fee:                      tx.GetTxFee(),
				Proof:                    txToken.GetTxBase().GetProof(),
				SigPubKey:                base58.Base58Check{}.Encode(tx.GetSigPubKey(), 0x0),
				Sig:                      base58.Base58Check{}.Encode(tx.GetSig(), 0x0),
				Info:                     string(tx.GetInfo()),
				IsPrivacy:                tx.IsPrivacy(),
				PrivacyCustomTokenSymbol: txTokenData.PropertySymbol,
				PrivacyCustomTokenName:   txTokenData.PropertyName,
				PrivacyCustomTokenID:     txTokenData.PropertyID.String(),
				PrivacyCustomTokenFee:    txTokenData.TxNormal.GetTxFee(),
			}
			
			if result.Proof != nil {
				inputCoins := result.Proof.GetInputCoins()
				if len(inputCoins) > 0 && inputCoins[0].GetPublicKey() != nil {
					result.InputCoinPubKey = base58.Base58Check{}.Encode(inputCoins[0].GetPublicKey().ToBytesS(), common.ZeroByte)
				}
			}


			tokenData, _ := json.MarshalIndent(txTokenData, "", "\t")
			result.PrivacyCustomTokenData = string(tokenData)
			if tx.GetMetadata() != nil {
				metaData, _ := json.MarshalIndent(tx.GetMetadata(), "", "\t")
				result.Metadata = string(metaData)
			}
			if result.Proof != nil {
				result.ProofDetail.ConvertFromProof(result.Proof)
			}
			result.PrivacyCustomTokenIsPrivacy = txTokenData.TxNormal.IsPrivacy()
			if txTokenData.TxNormal.GetProof() != nil {
				result.PrivacyCustomTokenProofDetail.ConvertFromProof(txTokenData.TxNormal.GetProof())
			}
		}
	default:
		{
			return nil, errors.New("Tx type is invalid")
		}
	}
	return result, nil
}

type ProofDetail struct {
	InputCoins  []CoinRPC
	OutputCoins []CoinRPC
}

func (proofDetail *ProofDetail) ConvertFromProof(proof privacy.Proof) {
	inputCoins := proof.GetInputCoins()
	outputCoins := proof.GetOutputCoins()

	proofDetail.InputCoins = make([]CoinRPC, len(inputCoins))
	for i, input := range inputCoins {
		proofDetail.InputCoins[i] = ParseCoinRPCInput(input)
	}

	proofDetail.OutputCoins = make([]CoinRPC, len(outputCoins))
	for i, output := range outputCoins {
		proofDetail.OutputCoins[i] = ParseCoinRPCOutput(output)
	}
}

func ParseCoinRPCInput(inputCoin coin.PlainCoin) CoinRPC {
	var coinrpc CoinRPC
	if inputCoin.GetVersion() == 1 {
		coinrpc = new(CoinRPCV1)
	} else {
		coinrpc = new(CoinRPCV2)
	}
	return coinrpc.SetInputCoin(inputCoin)
}

func ParseCoinRPCOutput(outputCoin coin.Coin) CoinRPC {
	var coinrpc CoinRPC
	if outputCoin.GetVersion() == 1 {
		coinrpc = new(CoinRPCV1)
	} else {
		coinrpc = new(CoinRPCV2)
	}
	return coinrpc.SetOutputCoin(outputCoin)
}

type CoinRPC interface {
	SetInputCoin(coin.PlainCoin) CoinRPC
	SetOutputCoin(coin.Coin) CoinRPC
}

func EncodeBase58Check(b []byte) string {
	if b == nil || len(b) == 0 {
		return ""
	}
	return base58.Base58Check{}.Encode(b, 0x0)
}

func OperationPointPtrToBase58(point *operation.Point) string {
	if point == nil || point.IsIdentity()  {
		return ""
	} else {
		return EncodeBase58Check(point.ToBytesS())
	}
}

func OperationScalarPtrToBase58(scalar *operation.Scalar) string {
	if scalar == nil {
		return ""
	} else {
		return EncodeBase58Check(scalar.ToBytesS())
	}
}

func (c *CoinRPCV1) SetInputCoin(inputCoin coin.PlainCoin) CoinRPC {
	coinv1 := inputCoin.(*coin.PlainCoinV1)

	c.Version = coinv1.GetVersion()
	c.PublicKey = OperationPointPtrToBase58(coinv1.GetPublicKey())
	c.Commitment = OperationPointPtrToBase58(coinv1.GetCommitment())
	c.SNDerivator = OperationScalarPtrToBase58(coinv1.GetSNDerivator())
	c.KeyImage = OperationPointPtrToBase58(coinv1.GetKeyImage())
	c.Randomness = OperationScalarPtrToBase58(coinv1.GetRandomness())
	c.Value = coinv1.GetValue()
	c.Info = EncodeBase58Check(coinv1.GetInfo())
	return c
}

func (c *CoinRPCV1) SetOutputCoin(inputCoin coin.Coin) CoinRPC {
	coinv1 := inputCoin.(*coin.CoinV1)

	c.Version = coinv1.GetVersion()
	c.PublicKey = OperationPointPtrToBase58(coinv1.GetPublicKey())
	c.Commitment = OperationPointPtrToBase58(coinv1.GetCommitment())
	c.SNDerivator = OperationScalarPtrToBase58(coinv1.GetSNDerivator())
	c.KeyImage = OperationPointPtrToBase58(coinv1.GetKeyImage())
	c.Randomness = OperationScalarPtrToBase58(coinv1.GetRandomness())
	c.Value = coinv1.CoinDetails.GetValue()
	c.Info = EncodeBase58Check(coinv1.GetInfo())
	if coinv1.CoinDetailsEncrypted != nil {
		c.CoinDetailsEncrypted = EncodeBase58Check(coinv1.CoinDetailsEncrypted.Bytes())
	} else {
		c.CoinDetailsEncrypted = ""
	}
	return c
}

func (c *CoinRPCV2) SetInputCoin(inputCoin coin.PlainCoin) CoinRPC {
	return c.SetOutputCoin(inputCoin.(coin.Coin))
}

func (c *CoinRPCV2) SetOutputCoin(outputCoin coin.Coin) CoinRPC {
	coinv2 := outputCoin.(*coin.CoinV2)

	c.Version = coinv2.GetVersion()
	c.Info = EncodeBase58Check(coinv2.GetInfo())
	c.PublicKey = OperationPointPtrToBase58(coinv2.GetPublicKey())
	c.Commitment = OperationPointPtrToBase58(coinv2.GetCommitment())
	c.KeyImage = OperationPointPtrToBase58(coinv2.GetKeyImage())
	c.TxRandom = EncodeBase58Check(coinv2.GetTxRandom().Bytes())
	c.Value = strconv.FormatUint(coinv2.GetValue(), 10)
	c.Randomness = OperationScalarPtrToBase58(coinv2.GetRandomness())
	//txRandomPoint, index, _ := coinv2.GetTxRandomDetail()
	//c.TxRandom = OperationPointPtrToBase58(txRandomPoint)
	//c.Index = index

	return c
}

type CoinRPCV1 struct {
	Version              uint8
	PublicKey            string
	Commitment           string
	SNDerivator          string
	KeyImage             string
	Randomness           string
	Value                uint64
	Info                 string
	CoinDetailsEncrypted string
}

type CoinRPCV2 struct {
	Version    uint8
	Index      uint32
	Info       string
	PublicKey  string
	Commitment string
	KeyImage   string
	TxRandom   string

	Value 		 string
	Randomness   string
}
