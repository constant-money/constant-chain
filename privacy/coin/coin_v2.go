package coin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

type TxRandom [TxRandomGroupSize]byte

func NewTxRandom() *TxRandom {
	txRandom := new(operation.Point).Identity()
	index := uint32(0)

	res := new(TxRandom)
	res.SetTxConcealRandomPoint(txRandom)
	res.SetIndex(index)
	return res
}

func (t TxRandom) GetTxConcealRandomPoint() (*operation.Point, error) {
	return new(operation.Point).FromBytesS(t[operation.Ed25519KeySize+4:])
}


func (t TxRandom) GetTxOTARandomPoint() (*operation.Point, error) {
	return new(operation.Point).FromBytesS(t[:operation.Ed25519KeySize])
}

func (t TxRandom) GetIndex() (uint32, error) {
	return common.BytesToUint32(t[operation.Ed25519KeySize:operation.Ed25519KeySize + 4])
}

func (t *TxRandom) SetTxConcealRandomPoint(txConcealRandom *operation.Point) {
	txRandomBytes := txConcealRandom.ToBytesS()
	copy(t[operation.Ed25519KeySize+4:], txRandomBytes)
}

func (t *TxRandom) SetTxOTARandomPoint(txRandom *operation.Point) {
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
	_, err = new(operation.Point).FromBytesS(b[operation.Ed25519KeySize+4:])
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
	sharedConcealRandom *operation.Scalar //rConceal: shared random when concealing output coin and blinding assetTag
	sharedRandom        *operation.Scalar // rOTA: shared random when creating one-time-address
	txRandom            *TxRandom         // rConceal*G + rOTA*G + index

	// mask = randomness
	// amount = value
	mask   *operation.Scalar
	amount *operation.Scalar
	// tag is nil unless confidential asset
	assetTag *operation.Point
}

//Retrieve the private OTA key of coin from the Master PrivateKey
func (c CoinV2) ParsePrivateKeyOfCoin(privKey key.PrivateKey) (*operation.Scalar, error) {
	keySet := new(incognitokey.KeySet)
	if err := keySet.InitFromPrivateKey(&privKey); err != nil {
		err := errors.New("Cannot init keyset from privateKey")
		return nil, errhandler.NewPrivacyErr(errhandler.InvalidPrivateKeyErr, err)
	}
	_, txRandomOTAPoint, index, err := c.GetTxRandomDetail()
	if err != nil {
		return nil, err
	}
	rK := new(operation.Point).ScalarMult(txRandomOTAPoint, keySet.OTAKey.GetOTASecretKey()) //(r_ota*G) * k = r_ota * K
	H := operation.HashToScalar(append(rK.ToBytesS(), common.Uint32ToBytes(index)...))     // Hash(r_ota*K, index)

	k := new(operation.Scalar).FromBytesS(privKey)
	return new(operation.Scalar).Add(H, k), nil // Hash(rK, index) + privSpend
}

//Retrieve the keyImage of coin from the Master PrivateKey
func (c CoinV2) ParseKeyImageWithPrivateKey(privKey key.PrivateKey) (*operation.Point, error) {
	k, err := c.ParsePrivateKeyOfCoin(privKey)
	if err != nil {
		err := errors.New("Cannot init keyset from privateKey")
		return nil, errhandler.NewPrivacyErr(errhandler.InvalidPrivateKeyErr, err)
	}
	Hp := operation.HashToPoint(c.GetPublicKey().ToBytesS())
	return new(operation.Point).ScalarMult(Hp, k), nil
}

// Conceal the amount of coin using the publicView of the receiver
//
//	- AdditionalData: must be the publicView of the receiver
func (c *CoinV2) ConcealOutputCoin(additionalData interface{}) error {
	// If this coin is already encrypted or it is created by other person then cannot conceal
	if c.IsEncrypted() || c.GetSharedConcealRandom() == nil {
		return nil
	}
	publicView, ok := additionalData.(*operation.Point)
	if !ok{
		return errors.New("Cannot conceal CoinV2 without receiver view key")
	}

	rK := new(operation.Point).ScalarMult(publicView, c.GetSharedConcealRandom()) //rK = sharedConcealRandom * publicView

	hash := operation.HashToScalar(rK.ToBytesS()) //hash(rK)
	hash = operation.HashToScalar(hash.ToBytesS())
	mask := new(operation.Scalar).Add(c.GetRandomness(), hash) //mask = c.mask + hash

	hash = operation.HashToScalar(hash.ToBytesS())
	amount := new(operation.Scalar).Add(c.GetAmount(), hash) //amount = c.amout + hash
	c.SetRandomness(mask)
	c.SetAmount(amount)
	c.SetSharedConcealRandom(nil)
	c.SetSharedRandom(nil)
	return nil
}

// Conceal the input coin of a transaction: keep only the keyImage + publicOTA
func (c *CoinV2) ConcealInputCoin() {
	c.SetValue(0)
	c.SetRandomness(nil)
	c.SetPublicKey(nil)
	c.SetCommitment(nil)
	c.SetTxRandomDetail(new(operation.Point).Identity(), new(operation.Point).Identity(), 0)
}

//Decrypt a coin using the corresponding KeySet
func (c *CoinV2) Decrypt(keySet *incognitokey.KeySet) (PlainCoin, error) {
	if keySet == nil {
		err := errors.New("Cannot Decrypt CoinV2: Keyset is empty")
		return nil, errhandler.NewPrivacyErr(errhandler.DecryptOutputCoinErr, err)
	}

	// Must parse keyImage first in any situation
	if len(keySet.PrivateKey) > 0 {
		keyImage, err := c.ParseKeyImageWithPrivateKey(keySet.PrivateKey)
		if err != nil {
			errReturn := errors.New("Cannot parse key image with privateKey CoinV2" + err.Error())
			return nil, errhandler.NewPrivacyErr(errhandler.ParseKeyImageWithPrivateKeyErr, errReturn)
		}
		c.SetKeyImage(keyImage)
	}

	if !c.IsEncrypted() {
		return c, nil
	}

	viewKey := keySet.ReadonlyKey
	if len(viewKey.Rk) == 0 && len(keySet.PrivateKey) == 0 {
		err := errors.New("Cannot Decrypt CoinV2: Keyset does not contain viewkey or privatekey")
		return nil, errhandler.NewPrivacyErr(errhandler.DecryptOutputCoinErr, err)
	}

	if viewKey.GetPrivateView()!= nil {
		txConcealRandomPoint, err := c.GetTxRandom().GetTxConcealRandomPoint()
		if err != nil {
			return nil, err
		}
		rK := new(operation.Point).ScalarMult(txConcealRandomPoint, viewKey.GetPrivateView())

		// Hash multiple times
		hash := operation.HashToScalar(rK.ToBytesS())
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
func (c CoinV2) GetSharedConcealRandom() *operation.Scalar { return c.sharedConcealRandom }
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
func (c CoinV2) GetTxRandomDetail() (*operation.Point, *operation.Point, uint32, error) {
	txRandomOTAPoint, err1 := c.txRandom.GetTxOTARandomPoint()
	txRandomConcealPoint, err2 := c.txRandom.GetTxConcealRandomPoint()
	index, err3 := c.txRandom.GetIndex()
	if err1 != nil || err2 != nil || err3 != nil{
		return nil, nil, 0, errors.New("Cannot Get TxRandom: point or index is wrong")
	}
	return txRandomConcealPoint, txRandomOTAPoint, index, nil
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
func (c CoinV2) GetCoinDetailEncrypted() []byte {
	return nil
}


func (c *CoinV2) SetVersion(uint8)                               { c.version = 2 }
func (c *CoinV2) SetRandomness(mask *operation.Scalar)           { c.mask = mask }
func (c *CoinV2) SetAmount(amount *operation.Scalar)             { c.amount = amount }
func (c *CoinV2) SetSharedRandom(sharedRandom *operation.Scalar) { c.sharedRandom = sharedRandom }
func (c *CoinV2) SetSharedConcealRandom(sharedConcealRandom *operation.Scalar) { c.sharedConcealRandom = sharedConcealRandom }
func (c *CoinV2) SetTxRandom(txRandom *TxRandom) {
	if txRandom == nil {
		c.txRandom = nil
	} else {
		c.txRandom = NewTxRandom()
		c.txRandom.SetBytes(txRandom.Bytes())
	}
}
func (c *CoinV2) SetTxRandomDetail(txConcealRandomPoint, txRandomPoint *operation.Point, index uint32) {
	res := new(TxRandom)
	res.SetTxConcealRandomPoint(txConcealRandomPoint)
	res.SetTxOTARandomPoint(txRandomPoint)
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

	if c.sharedConcealRandom != nil {
		coinBytes = append(coinBytes, byte(operation.Ed25519KeySize))
		coinBytes = append(coinBytes, c.sharedConcealRandom.ToBytesS()...)
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

	c.sharedConcealRandom, err = parseScalarForSetBytes(&coinBytes, &offset)
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

	rK := new(operation.Point).ScalarMult(paymentAdd.GetOTAPublicKey(), r)
	_, txOTARandomPoint, index,  err := c.GetTxRandomDetail()
	if err  != nil {
		return false
	}
	if !operation.IsPointEqual(new(operation.Point).ScalarMultBase(r), txOTARandomPoint) {
		return false
	}

	hash := operation.HashToScalar(append(rK.ToBytesS(), common.Uint32ToBytes(index)...))
	HrKG := new(operation.Point).ScalarMultBase(hash)
	tmpPubKey := new(operation.Point).Add(HrKG, paymentAdd.GetPublicSpend())
	return bytes.Equal(tmpPubKey.ToBytesS(), c.publicKey.ToBytesS())
}

// Check whether the utxo is from this keyset
func (c *CoinV2) DoesCoinBelongToKeySet(keySet *incognitokey.KeySet) (bool, *operation.Point) {
	_, txOTARandomPoint, index, err1 :=  c.GetTxRandomDetail()
	if err1 != nil {
		return false, nil
	}

	//Check if the utxo belong to this one-time-address
	rK := new(operation.Point).ScalarMult(txOTARandomPoint, keySet.OTAKey.GetOTASecretKey())

	hashed := operation.HashToScalar(
		append(rK.ToBytesS(), common.Uint32ToBytes(index)...),
	)

	HnG := new(operation.Point).ScalarMultBase(hashed)
	KCheck := new(operation.Point).Sub(c.GetPublicKey(), HnG)

	////Retrieve the sharedConcealRandomPoint for generating the blinded assetTag
	//var rSharedConcealPoint *operation.Point
	//if keySet.ReadonlyKey.GetPrivateView() != nil {
	//	rSharedConcealPoint = new(operation.Point).ScalarMult(txConcealRandomPoint, keySet.ReadonlyKey.GetPrivateView())
	//}

	return operation.IsPointEqual(KCheck, keySet.OTAKey.GetPublicSpend()), rK
}
