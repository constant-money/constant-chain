package key

import (
	"encoding/hex"
	"errors"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

func SliceToArray(slice []byte) [operation.Ed25519KeySize]byte {
	var array [operation.Ed25519KeySize]byte
	copy(array[:], slice)
	return array
}

func ArrayToSlice(array [operation.Ed25519KeySize]byte) []byte {
	var slice []byte
	slice = array[:]
	return slice
}

// 32-byte spending key
type PrivateKey []byte

// 32-byte public key
type PublicKey []byte

// 32-byte receiving key
type ReceivingKey []byte

// 32-byte transmission key
type TransmissionKey []byte

// 32-byte public OTA key: used to generate one-time-address coin (for senders of COINV2 only)
type PublicOTAKey []byte

// 32-byte secret key: used to check if a coin belongs to a receiver (for receivers of COIN2 only)
type PrivateOTAKey []byte

// ViewingKey is a public/private key pair to encrypt coins in an outgoing transaction
type ViewingKey struct {
	Pk PublicKey    // 33 bytes, use to receive coin
	Rk ReceivingKey // 32 bytes, use to decrypt pointByte
}

func (viewKey ViewingKey) GetPublicSpend() *operation.Point {
	pubSpend, _ := new(operation.Point).FromBytesS(viewKey.Pk)
	return pubSpend
}

func (viewKey ViewingKey) GetPrivateView() *operation.Scalar {
	return new(operation.Scalar).FromBytesS(viewKey.Rk)
}

// OTAKey is a pair of keys used to recover coin's one-time-address
type OTAKey struct{
	pk        PublicKey //32 bytes: used to
	otaSecret PrivateOTAKey
}

func (otaKey OTAKey) GetPublicSpend() *operation.Point {
	pubSpend, err := new(operation.Point).FromBytesS(otaKey.pk)
	if err != nil{
		return nil
	}
	if pubSpend.PointValid(){
		return pubSpend
	}
	return nil
}

func (otaKey OTAKey) GetOTASecretKey() *operation.Scalar {
	otaSecret := new(operation.Scalar).FromBytesS(otaKey.otaSecret)
	if otaSecret.ScalarValid(){
		return otaSecret
	}
	return nil
}

func (otaKey *OTAKey) SetOTASecretKey(otaSecretKey []byte){
	if len(otaKey.otaSecret) == 0{
		otaKey.otaSecret = append([]byte{}, otaSecretKey...)
	}
}

func (otaKey *OTAKey) SetPublicSpend(publicSpend []byte){
	if len(otaKey.pk) == 0{
		otaKey.pk = append([]byte{}, publicSpend...)
	}
}

// PaymentAddress is an address of the payee
type PaymentAddress struct {
	Pk PublicKey       // 32 bytes, use to receive coin (CoinV1)
	Tk TransmissionKey // 32 bytes, use to encrypt pointByte
	OTAPublic PublicOTAKey //32 bytes, used to receive coin (CoinV2)
}

// Bytes converts payment address to bytes array
func (addr *PaymentAddress) Bytes() []byte {
	res := append(addr.Pk[:], addr.Tk[:]...)
	if addr.OTAPublic != nil {
		return append(res, addr.OTAPublic[:]...)
	}
	return res
}

// SetBytes reverts bytes array to payment address
func (addr *PaymentAddress) SetBytes(bytes []byte) error {
	if len(bytes) != 2*operation.Ed25519KeySize && len(bytes) != 3*operation.Ed25519KeySize{
		return errors.New("length of payment address not valid")
	}
	// the first 33 bytes are public key
	addr.Pk = bytes[:operation.Ed25519KeySize]
	// the last 33 bytes are transmission key
	addr.Tk = bytes[operation.Ed25519KeySize:2*operation.Ed25519KeySize]
	if len(bytes) == 3 * operation.Ed25519KeySize{
		addr.OTAPublic = bytes[2*operation.Ed25519KeySize:]
	}else{
		addr.OTAPublic = nil
	}
	return nil
}

// String encodes a payment address as a hex string
func (addr PaymentAddress) String() string {
	byteArrays := addr.Bytes()
	return hex.EncodeToString(byteArrays[:])
}

func (addr PaymentAddress) GetPublicSpend() *operation.Point {
	pubSpend, _ := new(operation.Point).FromBytesS(addr.Pk)
	return pubSpend
}

func (addr PaymentAddress) GetPublicView() *operation.Point {
	pubView, _ := new(operation.Point).FromBytesS(addr.Tk)
	return pubView
}

func(addr PaymentAddress) GetOTAPublicKey() *operation.Point{
	encryptionKey, _ := new(operation.Point).FromBytesS(addr.OTAPublic)
	return  encryptionKey
}

// PaymentInfo contains an address of a payee and a value of coins he/she will receive
type PaymentInfo struct {
	PaymentAddress PaymentAddress
	Amount         uint64
	Message        []byte // 512 bytes
}

func InitPaymentInfo(addr PaymentAddress, amount uint64, message []byte) *PaymentInfo {
	return &PaymentInfo{
		PaymentAddress: addr,
		Amount: amount,
		Message: message,
	}
}

// GeneratePrivateKey generates a random 32-byte spending key
func GeneratePrivateKey(seed []byte) PrivateKey {
	bip32PrivKey := operation.HashToScalar(seed)
	privateKey := bip32PrivKey.ToBytesS()
	return privateKey
}

// GeneratePublicKey computes a 32-byte public-key corresponding to a spending key
func GeneratePublicKey(privateKey []byte) PublicKey {
	privScalar := new(operation.Scalar).FromBytesS(privateKey)
	publicKey := new(operation.Point).ScalarMultBase(privScalar)
	return publicKey.ToBytesS()
}

// GenerateReceivingKey generates a 32-byte receiving key
func GenerateReceivingKey(privateKey []byte) ReceivingKey {
	receivingKey := operation.HashToScalar(privateKey[:])
	return receivingKey.ToBytesS()
}

// GenerateTransmissionKey computes a 33-byte transmission key corresponding to a receiving key
func GenerateTransmissionKey(receivingKey []byte) TransmissionKey {
	receiScalar := new(operation.Scalar).FromBytesS(receivingKey)
	transmissionKey := new(operation.Point).ScalarMultBase(receiScalar)
	return transmissionKey.ToBytesS()
}

// GenerateViewingKey generates a viewingKey corresponding to a spending key
func GenerateViewingKey(privateKey []byte) ViewingKey {
	var viewingKey ViewingKey
	viewingKey.Pk = GeneratePublicKey(privateKey)
	viewingKey.Rk = GenerateReceivingKey(privateKey)
	return viewingKey
}

func GeneratePrivateOTAKey(privateKey []byte) PrivateOTAKey {
	privateOTAKey := append(privateKey, []byte(operation.CStringOTA)...)
	privateOTAKey = operation.HashToScalar(privateOTAKey).ToBytesS()
	return privateOTAKey
}

func GeneratePublicOTAKey(privateOTAKey PrivateOTAKey) PublicOTAKey {
	privateOTAScalar := new(operation.Scalar).FromBytesS(privateOTAKey)
	return new(operation.Point).ScalarMultBase(privateOTAScalar).ToBytesS()
}

func GenerateOTAKey(privateKey []byte) OTAKey{
	var otaKey OTAKey
	otaKey.pk = GeneratePublicKey(privateKey)
	otaKey.otaSecret = GeneratePrivateOTAKey(privateKey)
	return otaKey
}

// GeneratePaymentAddress generates a payment address corresponding to a spending key
func GeneratePaymentAddress(privateKey []byte) PaymentAddress {
	var paymentAddress PaymentAddress
	paymentAddress.Pk = GeneratePublicKey(privateKey)
	paymentAddress.Tk = GenerateTransmissionKey(GenerateReceivingKey(privateKey))
	paymentAddress.OTAPublic = GeneratePublicOTAKey(GeneratePrivateOTAKey(privateKey))
	return paymentAddress
}
