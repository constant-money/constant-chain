package coin

import (
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

type CoinV2 struct {
	// Version should be described here as a reminder
	// SetBytes and FromBytes of CoinV1 and CoinV2 will use this first byte as version
	version    uint8
	shardId    uint8
	mask       *operation.Scalar
	amount     *operation.Scalar
	txRandom   *operation.Point
	publicKey  *operation.Point // R^o = H_n(r * K_B^v )G + K_B^s
	commitment *operation.Point
	index      uint8
	info       []byte //256 bytes
}

func (this CoinV2) GetVersion() uint8               { return 2 }
func (this CoinV2) GetShardId() uint8               { return this.shardId }
func (this CoinV2) GetMask() *operation.Scalar      { return this.mask }
func (this CoinV2) GetAmount() *operation.Scalar    { return this.amount }
func (this CoinV2) GetTxRandom() *operation.Point   { return this.txRandom }
func (this CoinV2) GetPublicKey() *operation.Point  { return this.publicKey }
func (this CoinV2) GetCommitment() *operation.Point { return this.commitment }
func (this CoinV2) GetIndex() uint8                 { return this.index }
func (this CoinV2) GetInfo() []byte                 { return this.info }

func (this *CoinV2) SetVersion()                               { this.version = 2 }
func (this *CoinV2) SetMask(mask *operation.Scalar)            { this.mask.Set(mask) }
func (this *CoinV2) SetShardId(shardId uint8)                  { this.shardId = shardId }
func (this *CoinV2) SetAmount(amount *operation.Scalar)        { this.amount.Set(amount) }
func (this *CoinV2) SetTxRandom(txRandom *operation.Point)     { this.txRandom.Set(txRandom) }
func (this *CoinV2) SetPublicKey(publicKey *operation.Point)   { this.publicKey.Set(publicKey) }
func (this *CoinV2) SetCommitment(commitment *operation.Point) { this.commitment.Set(commitment) }
func (this *CoinV2) SetIndex(index uint8)                      { this.index = index }

func (this *CoinV2) SetInfo(b []byte) error {
	if len(b) > MaxSizeInfoCoin {
		return errors.New("Cannot set info to CoinV2, info is longer than 255")
	}
	this.info = make([]byte, len(b))
	copy(this.info, b)
	return nil
}

func NewCoinv2(shardId uint8, mask *operation.Scalar, amount *operation.Scalar, txRandom *operation.Point, publicKey *operation.Point, commitment *operation.Point, index uint8, info []byte) *CoinV2 {
	return &CoinV2{
		2,
		shardId,
		mask,
		amount,
		txRandom,
		publicKey,
		commitment,
		index,
		info,
	}
}

// Init (Coin) initializes a coin
func (this *CoinV2) Init() *CoinV2 {
	if this == nil {
		this = new(CoinV2)
	}
	this.version = uint8(2)
	this.mask = new(operation.Scalar).FromUint64(0)
	this.amount = new(operation.Scalar).FromUint64(0)
	this.txRandom = new(operation.Point).Identity()
	this.publicKey = new(operation.Point).Identity()
	this.commitment = new(operation.Point).Identity()
	this.index = uint8(0)
	this.info = []byte{}
	return this
}

// Bytes converts a coin's details to a bytes array
// Each fields in coin is saved in len - body format
func (this *CoinV2) Bytes() []byte {
	var coinBytes []byte
	coinBytes = append(coinBytes, this.GetVersion())
	shardId := byte(this.GetShardId())
	coinBytes = append(coinBytes, shardId)

	if this.mask != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, this.mask.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}
	if this.amount != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, this.amount.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}
	if this.txRandom != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, this.txRandom.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}
	if this.publicKey != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, this.publicKey.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}
	if this.commitment != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, this.commitment.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}
	coinBytes = append(coinBytes, this.index)
	if len(this.info) > 0 {
		byteLengthInfo := byte(getMin(len(this.info), MaxSizeInfoCoin))
		coinBytes = append(coinBytes, byteLengthInfo)
		infoBytes := this.info[0:byteLengthInfo]
		coinBytes = append(coinBytes, infoBytes...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	return coinBytes
}

func (this *CoinV2) SetBytes(coinBytes []byte) error {
	if len(coinBytes) == 0 {
		return errors.New("coinBytes is empty")
	}
	if coinBytes[0] != 2 {
		return errors.New("The version of this byte is not 2, it should be 2")
	}

	if this == nil {
		this = new(CoinV2)
	}
	var err error

	this.SetVersion()
	shardId := coinBytes[1]
	this.SetShardId(shardId)

	// The first 2 bytes are: 1 byte for version, 1 byte for shard id
	offset := 2
	this.mask, err = parseScalarForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 mask error: " + err.Error())
	}
	this.amount, err = parseScalarForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 amount error: " + err.Error())
	}
	this.txRandom, err = parsePointForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 txRandom error: " + err.Error())
	}
	this.publicKey, err = parsePointForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 publicKey error: " + err.Error())
	}
	this.commitment, err = parsePointForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 commitment error: " + err.Error())
	}

	if offset >= len(coinBytes) {
		return errors.New("Offset is larger than len(bytes), cannot parse index")
	}
	this.index = coinBytes[offset]
	offset++
	this.info, err = parseInfoForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 info error: " + err.Error())
	}

	return nil
}

// HashH returns the SHA3-256 hashing of coin bytes array
func (this *CoinV2) HashH() *common.Hash {
	hash := common.HashH(this.Bytes())
	return &hash
}

func (coin *CoinV2) GetCoinValue(privateKey *key.PrivateKey) uint64 {
	// PRIVACY TODO
	return 0
}