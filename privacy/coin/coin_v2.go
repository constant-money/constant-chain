package coin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

type TxRandom [TxRandomGroupSize]byte

func NewTxRandom() *TxRandom {
	txRandom := new(operation.Point).Identity()
	index := uint32(0)

	res := new(TxRandom)
	res.SetTxRandomPoint(txRandom)
	res.SetIndex(index)
	return res
}

func (t TxRandom) GetTxRandomPoint() (*operation.Point, error) {
	return new(operation.Point).FromBytesS(t[:operation.Ed25519KeySize])
}

func (t TxRandom) GetIndex() (uint32, error) {
	return common.BytesToUint32(t[operation.Ed25519KeySize:])
}

func (t *TxRandom) SetTxRandomPoint(txRandom *operation.Point) {
	txRandomBytes := txRandom.ToBytesS()
	copy(t[:operation.Ed25519KeySize], txRandomBytes)
}

func (t *TxRandom) SetIndex(index uint32) {
	indexBytes := common.Uint32ToBytes(index)
	copy(t[operation.Ed25519KeySize:], indexBytes)
}

func (t TxRandom) Bytes() []byte {
	return t[:]
}

func (t *TxRandom) SetBytes(b []byte) error {
	if b == nil || len(b) != TxRandomGroupSize {
		return errors.New("Cannnot SetByte to TxRandom. Input is invalid")
	}
	_, err := new(operation.Point).FromBytesS(b[:operation.Ed25519KeySize])
	if err != nil {
		errStr := fmt.Sprintf("Cannot TxRandomGroupSize.SetBytes: bytes is not operation.Point err: %v", err)
		return errors.New(errStr)
	}
	copy(t[:], b)
	return nil
}

// CoinV2 is the struct that will be stored to db
// If not privacy, mask and amount will be the original randomness and value
// If has privacy, mask and amount will be as paper monero
type CoinV2 struct {
	// Public
	version    uint8
	info       []byte
	publicKey  *operation.Point
	commitment *operation.Point
	keyImage   *operation.Point

	// sharedRandom and txRandom is shared secret between receiver and giver
	// sharedRandom is only visible when creating coins, when it is broadcast to network, it will be set to null
	sharedRandom *operation.Scalar // r
	txRandom     *TxRandom         // rG + index

	// mask = randomness
	// amount = value
	mask   *operation.Scalar
	amount *operation.Scalar
	// tag is nil unless confidential asset
	assetTag *operation.Point
}

func (c CoinV2) ParsePrivateKeyOfCoin(privKey key.PrivateKey) (*operation.Scalar, error) {
	readonlyKey := key.GenerateViewingKey(privKey[:])
	txRandomPoint, index, err := c.GetTxRandomDetail()
	if err != nil {
		return nil, err
	}
	rK := new(operation.Point).ScalarMult(txRandomPoint, readonlyKey.GetPrivateView()) //rG * k = rK
	H := operation.HashToScalar(append(rK.ToBytesS(), common.Uint32ToBytes(index)...))        // Hash(rK, index)

	k := new(operation.Scalar).FromBytesS(privKey)
	return new(operation.Scalar).Add(H, k), nil // Hash(rK, index) + privSpend
}

func (c CoinV2) ParseKeyImageWithPrivateKey(privKey key.PrivateKey) (*operation.Point, error) {
	k, err := c.ParsePrivateKeyOfCoin(privKey)
	if err != nil {
		err := errors.New("Cannot init keyset from privateKey")
		return nil, errhandler.NewPrivacyErr(errhandler.InvalidPrivateKeyErr, err)
	}
	Hp := operation.HashToPoint(c.GetPublicKey().ToBytesS())
	return new(operation.Point).ScalarMult(Hp, k), nil
}

// AdditionalData of concealData should be publicView of the receiver
func (c *CoinV2) ConcealOutputCoin(additionalData interface{}) error {
	// If this coin is already encrypted or it is created by other person then cannot conceal
	if c.IsEncrypted() || c.GetSharedRandom() == nil {
		return nil
	}
	publicView, ok := additionalData.(*operation.Point)
	if !ok{
		return errors.New("Cannot conceal CoinV2 without receiver view key")
	}

	rK := new(operation.Point).ScalarMult(publicView, c.GetSharedRandom())
	_, index, err := c.GetTxRandomDetail()
	if err  != nil {
		return err
	}
	hash := operation.HashToScalar(append(rK.ToBytesS(), common.Uint32ToBytes(index)...))
	hash = operation.HashToScalar(hash.ToBytesS())
	mask := new(operation.Scalar).Add(c.GetRandomness(), hash)

	hash = operation.HashToScalar(hash.ToBytesS())
	amount := new(operation.Scalar).Add(c.GetAmount(), hash)
	c.SetRandomness(mask)
	c.SetAmount(amount)
	c.SetSharedRandom(nil)
	return nil
}

func (c *CoinV2) ConcealInputCoin() {
	c.SetValue(0)
	c.SetRandomness(nil)
	c.SetPublicKey(nil)
	c.SetCommitment(nil)
	c.SetTxRandomDetail(new(operation.Point).Identity(), 0)
}

func (c *CoinV2) Decrypt(keySet IKeySet) (PlainCoin, error) {
	// Must parse keyImage first in any situation
	if len(keySet.GetPrivateKey()) > 0 {
		keyImage, err := c.ParseKeyImageWithPrivateKey(keySet.GetPrivateKey())
		if err != nil {
			errReturn := errors.New("Cannot parse key image with privateKey CoinV2" + err.Error())
			return nil, errhandler.NewPrivacyErr(errhandler.ParseKeyImageWithPrivateKeyErr, errReturn)
		}
		c.SetKeyImage(keyImage)
	}
	if c.IsEncrypted() == false {
		return c, nil
	}
	if keySet == nil {
		err := errors.New("Cannot Decrypt CoinV2: Keyset is empty")
		return nil, errhandler.NewPrivacyErr(errhandler.DecryptOutputCoinErr, err)
	}
	viewKey := keySet.GetReadOnlyKey()
	if len(viewKey.Rk) == 0 && len(keySet.GetPrivateKey()) == 0 {
		err := errors.New("Cannot Decrypt CoinV2: Keyset does not contain viewkey or privatekey")
		return nil, errhandler.NewPrivacyErr(errhandler.DecryptOutputCoinErr, err)
	}

	if len(viewKey.Rk) > 0 {
		txRandomPoint, index, err := c.GetTxRandomDetail()
		if err != nil {
			return nil, err
		}
		rK := new(operation.Point).ScalarMult(txRandomPoint, viewKey.GetPrivateView())

		// Hash multiple times
		hash := operation.HashToScalar(append(rK.ToBytesS(), common.Uint32ToBytes(index)...))
		hash = operation.HashToScalar(hash.ToBytesS())
		randomness := c.GetRandomness().Sub(c.GetRandomness(), hash)

		// Hash 1 more time to get value
		hash = operation.HashToScalar(hash.ToBytesS())
		value := c.GetAmount().Sub(c.GetAmount(), hash)

		commitment := operation.PedCom.CommitAtIndex(value, randomness, operation.PedersenValueIndex)
		// for `confidential asset` coin, we commit differently
		if c.GetAssetTag() != nil{
			com, err := ComputeCommitmentCA(c.GetAssetTag(), randomness, value)
			if err!=nil{
				err := errors.New("Cannot recompute commitment when decrypting")
				return nil, errhandler.NewPrivacyErr(errhandler.DecryptOutputCoinErr, err)
			}
			commitment = com
		}
		if !operation.IsPointEqual(commitment, c.GetCommitment()) {
			err := errors.New("Cannot Decrypt CoinV2: Commitment is not the same after decrypt")
			return nil, errhandler.NewPrivacyErr(errhandler.DecryptOutputCoinErr, err)
		}
		c.SetRandomness(randomness)
		c.SetAmount(value)
	}
	return c, nil
}

// Init (OutputCoin) initializes a output coin
func (c *CoinV2) Init() *CoinV2 {
	if c == nil {
		c = new(CoinV2)
	}
	c.version = 2
	c.info = []byte{}
	c.publicKey = new(operation.Point).Identity()
	c.commitment = new(operation.Point).Identity()
	c.keyImage = new(operation.Point).Identity()
	c.sharedRandom = new(operation.Scalar)
	c.txRandom = NewTxRandom()
	c.mask = new(operation.Scalar)
	c.amount = new(operation.Scalar)
	return c
}

// Get SND will be nil for ver 2
func (c CoinV2) GetSNDerivator() *operation.Scalar { return nil }

func (c CoinV2) IsEncrypted() bool {
	if c.mask == nil || c.amount == nil{
		return true
	}
	tempCommitment := operation.PedCom.CommitAtIndex(c.amount, c.mask, operation.PedersenValueIndex)
	if c.GetAssetTag() != nil{
		// err is only for nil parameters, which we already checked, so here it is ignored
		com, _ := c.ComputeCommitmentCA()
		tempCommitment = com
	}
	return !operation.IsPointEqual(tempCommitment, c.commitment)
}

func (c CoinV2) GetVersion() uint8                { return 2 }
func (c CoinV2) GetRandomness() *operation.Scalar { return c.mask }
func (c CoinV2) GetAmount() *operation.Scalar       { return c.amount }
func (c CoinV2) GetSharedRandom() *operation.Scalar { return c.sharedRandom }
func (c CoinV2) GetPublicKey() *operation.Point  { return c.publicKey }
func (c CoinV2) GetCommitment() *operation.Point { return c.commitment }
func (c CoinV2) GetKeyImage() *operation.Point { return c.keyImage }
func (c CoinV2) GetInfo() []byte               { return c.info }
func (c CoinV2) GetAssetTag() *operation.Point { return c.assetTag }
func (c CoinV2) GetValue() uint64 {
	if c.IsEncrypted() {
		return 0
	}
	return c.amount.ToUint64Little()
}
func (c CoinV2) GetTxRandom() *TxRandom {return c.txRandom}
func (c CoinV2) GetTxRandomDetail() (*operation.Point, uint32, error) {
	txRandomPoint, err1 := c.txRandom.GetTxRandomPoint()
	index, err2 := c.txRandom.GetIndex()
	if err1 != nil || err2 != nil{
		return nil, 0, errors.New("Cannot Get TxRandom: point or index is wrong")
	}
	return txRandomPoint, index, nil
}
func (c CoinV2) GetShardID() (uint8, error) {
	if c.publicKey == nil {
		return 255, errors.New("Cannot get GetShardID because GetPublicKey of PlainCoin is concealed")
	}
	pubKeyBytes := c.publicKey.ToBytes()
	lastByte := pubKeyBytes[operation.Ed25519KeySize-1]
	shardID := common.GetShardIDFromLastByte(lastByte)
	return shardID, nil
}

func (c *CoinV2) SetVersion(uint8)                               { c.version = 2 }
func (c *CoinV2) SetRandomness(mask *operation.Scalar)           { c.mask = mask }
func (c *CoinV2) SetAmount(amount *operation.Scalar)             { c.amount = amount }
func (c *CoinV2) SetSharedRandom(sharedRandom *operation.Scalar) { c.sharedRandom = sharedRandom }
func (c *CoinV2) SetTxRandom(txRandom *TxRandom) {
	if txRandom == nil {
		c.txRandom = nil
	} else {
		c.txRandom = NewTxRandom()
		c.txRandom.SetBytes(txRandom.Bytes())
	}
}
func (c *CoinV2) SetTxRandomDetail(txRandomPoint *operation.Point, index uint32) {
	res := new(TxRandom)
	res.SetTxRandomPoint(txRandomPoint)
	res.SetIndex(index)
	c.txRandom = res
}

func (c *CoinV2) SetPublicKey(publicKey *operation.Point)   { c.publicKey = publicKey }
func (c *CoinV2) SetCommitment(commitment *operation.Point) { c.commitment = commitment }
func (c *CoinV2) SetKeyImage(keyImage *operation.Point)     { c.keyImage = keyImage }
func (c *CoinV2) SetValue(value uint64)                     { c.amount = new(operation.Scalar).FromUint64(value) }
func (c *CoinV2) SetInfo(b []byte) {
	c.info = make([]byte, len(b))
	copy(c.info, b)
}
func (c *CoinV2) SetAssetTag(at *operation.Point)     { c.assetTag = at }

func (c CoinV2) Bytes() []byte {
	coinBytes := []byte{c.GetVersion()}
	info := c.GetInfo()
	byteLengthInfo := byte(getMin(len(info), MaxSizeInfoCoin))
	coinBytes = append(coinBytes, byteLengthInfo)
	coinBytes = append(coinBytes, info[:byteLengthInfo]...)

	if c.publicKey != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.publicKey.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	if c.commitment != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.commitment.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	if c.keyImage != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.keyImage.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	if c.sharedRandom != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.sharedRandom.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	if c.txRandom != nil {
		coinBytes = append(coinBytes, TxRandomGroupSize)
		coinBytes = append(coinBytes, c.txRandom.Bytes()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	if c.mask != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.mask.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	if c.amount != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.amount.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	if c.assetTag != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.assetTag.ToBytesS()...)
	} else {
		coinBytes = append(coinBytes, byte(0))
	}

	return coinBytes
}

func (c *CoinV2) SetBytes(coinBytes []byte) error {
	var err error
	if c == nil {
		return errors.New("Cannot set bytes for unallocated CoinV2")
	}
	if len(coinBytes) == 0 {
		return errors.New("coinBytes is empty")
	}
	if coinBytes[0] != 2 {
		return errors.New("coinBytes version is not 2")
	}
	c.SetVersion(coinBytes[0])

	offset := 1
	c.info, err = parseInfoForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 info error: " + err.Error())
	}

	c.publicKey, err = parsePointForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 publicKey error: " + err.Error())
	}
	c.commitment, err = parsePointForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 commitment error: " + err.Error())
	}
	c.keyImage, err = parsePointForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 keyImage error: " + err.Error())
	}
	c.sharedRandom, err = parseScalarForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 mask error: " + err.Error())
	}

	if offset >= len(coinBytes) {
		return errors.New("Offset is larger than len(bytes), cannot parse txRandom")
	}
	if coinBytes[offset] != TxRandomGroupSize {
		return errors.New("SetBytes CoinV2 TxRandomGroup error: length of TxRandomGroup is not correct")
	}
	offset += 1
	if offset+TxRandomGroupSize > len(coinBytes) {
		return errors.New("SetBytes CoinV2 TxRandomGroup error: length of coinBytes is too small")
	}
	c.txRandom = NewTxRandom()
	err = c.txRandom.SetBytes(coinBytes[offset : offset+TxRandomGroupSize])
	if err != nil {
		return errors.New("SetBytes CoinV2 TxRandomGroup error: " + err.Error())
	}
	offset += TxRandomGroupSize

	if err != nil {
		return errors.New("SetBytes CoinV2 txRandom error: " + err.Error())
	}
	c.mask, err = parseScalarForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 mask error: " + err.Error())
	}
	c.amount, err = parseScalarForSetBytes(&coinBytes, &offset)
	if err != nil {
		return errors.New("SetBytes CoinV2 amount error: " + err.Error())
	}
	
	if offset >=len(coinBytes){
		// for parsing old serialization, which does not have assetTag field
		c.assetTag = nil
	}else{
		c.assetTag, err = parsePointForSetBytes(&coinBytes, &offset)
		if err != nil {
			return errors.New("SetBytes CoinV2 assetTag error: " + err.Error())
		}
	}
	return nil
}

// HashH returns the SHA3-256 hashing of coin bytes array
func (c *CoinV2) HashH() *common.Hash {
	hash := common.HashH(c.Bytes())
	return &hash
}

func (c CoinV2) MarshalJSON() ([]byte, error) {
	data := c.Bytes()
	temp := base58.Base58Check{}.Encode(data, common.ZeroByte)
	return json.Marshal(temp)
}

func (c *CoinV2) UnmarshalJSON(data []byte) error {
	dataStr := ""
	_ = json.Unmarshal(data, &dataStr)
	temp, _, err := base58.Base58Check{}.Decode(dataStr)
	if err != nil {
		return err
	}
	err = c.SetBytes(temp)
	if err != nil {
		return err
	}
	return nil
}

func (c *CoinV2) CheckCoinValid(paymentAdd key.PaymentAddress, sharedRandom []byte, amount uint64) bool {
	if c.GetValue() != amount {
		return false
	}
	// check one-time address is corresponding to paymentaddress
	r := new(operation.Scalar).FromBytesS(sharedRandom)
	if !r.ScalarValid() {
		return false
	}

	rK := new(operation.Point).ScalarMult(paymentAdd.GetPublicView(), r)
	txRandomPoint, index, err := c.GetTxRandomDetail()
	if err  != nil {
		return false
	}
	if !operation.IsPointEqual(new(operation.Point).ScalarMultBase(r), txRandomPoint) {
		return false
	}

	hash := operation.HashToScalar(append(rK.ToBytesS(), common.Uint32ToBytes(index)...))
	HrKG := new(operation.Point).ScalarMultBase(hash)
	tmpPubKey := new(operation.Point).Add(HrKG, paymentAdd.GetPublicSpend())
	return bytes.Equal(tmpPubKey.ToBytesS(), c.publicKey.ToBytesS())
}
