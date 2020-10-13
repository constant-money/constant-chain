package incognitokey

import (
	"errors"

	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/schnorr"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
)

// KeySet is real raw data of wallet account, which user can use to
// - spend and check double spend coin with private key
// - receive coin with payment address
// - read tx data with readonly key
type KeySet struct {
	PrivateKey     key.PrivateKey
	PaymentAddress key.PaymentAddress
	ReadonlyKey    key.ViewingKey
	OTAKey 		   key.OTAKey
}

func (keySet *KeySet) GetPrivateKey() key.PrivateKey{
	return keySet.PrivateKey
}
func (keySet *KeySet) GetPaymentAddress() key.PaymentAddress{
	return keySet.PaymentAddress
}
func (keySet *KeySet) GetReadOnlyKey() key.ViewingKey{
	return keySet.ReadonlyKey
}

// GenerateKey generates key set from seed in byte array
func (keySet *KeySet) GenerateKey(seed []byte) *KeySet {
	keySet.PrivateKey = key.GeneratePrivateKey(seed)
	keySet.PaymentAddress = key.GeneratePaymentAddress(keySet.PrivateKey[:])
	keySet.ReadonlyKey = key.GenerateViewingKey(keySet.PrivateKey[:])
	keySet.OTAKey = key.GenerateOTAKey(keySet.PrivateKey[:])
	return keySet
}

// InitFromPrivateKeyByte receives private key in bytes array,
// and regenerates payment address and readonly key
// returns error if private key is invalid
func (keySet *KeySet) InitFromPrivateKeyByte(privateKey []byte) error {
	if len(privateKey) != common.PrivateKeySize {
		return NewCashecError(InvalidPrivateKeyErr, nil)
	}

	keySet.PrivateKey = privateKey
	keySet.PaymentAddress = key.GeneratePaymentAddress(keySet.PrivateKey[:])
	keySet.ReadonlyKey = key.GenerateViewingKey(keySet.PrivateKey[:])
	keySet.OTAKey = key.GenerateOTAKey(keySet.PrivateKey[:])
	return nil
}

// InitFromPrivateKey receives private key in PrivateKey type,
// and regenerates payment address and readonly key
// returns error if private key is invalid
func (keySet *KeySet) InitFromPrivateKey(privateKey *key.PrivateKey) error {
	if privateKey == nil || len(*privateKey) != common.PrivateKeySize {
		return NewCashecError(InvalidPrivateKeyErr, nil)
	}

	keySet.PrivateKey = *privateKey
	keySet.PaymentAddress = key.GeneratePaymentAddress(keySet.PrivateKey[:])
	keySet.ReadonlyKey = key.GenerateViewingKey(keySet.PrivateKey[:])
	keySet.OTAKey = key.GenerateOTAKey(keySet.PrivateKey[:])

	return nil
}

// Sign receives data in bytes array and
// returns the signature of that data using Schnorr Signature Scheme with signing key is private key in ketSet
func (keySet KeySet) Sign(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, NewCashecError(InvalidDataSignErr, errors.New("data is empty to sign"))
	}

	hash := common.HashB(data)
	privateKeySig := new(schnorr.SchnorrPrivateKey)
	privateKeySig.Set(new(operation.Scalar).FromBytesS(keySet.PrivateKey), new(operation.Scalar).FromUint64(0))

	signature, err := privateKeySig.Sign(hash)
	if err != nil {
		return []byte{}, NewCashecError(SignError, err)
	}
	return signature.Bytes(), nil
}

// Verify receives data and signature
// It checks whether the given signature is the signature of data
// and was signed by private key corresponding to public key in keySet or not
// func (keySet KeySet) Verify(data, signature []byte) (bool, error) {
// 	hash := common.HashB(data)
// 	isValid := false

// 	pubKeySig := new(schnorr.SchnorrPublicKey)
// 	PK, err := new(operation.Point).FromBytesS(keySet.PaymentAddress.Pk)
// 	if err != nil {
// 		return false, NewCashecError(InvalidVerificationKeyErr, nil)
// 	}
// 	pubKeySig.Set(PK)

// 	signatureSetBytes := new(schnorr.SchnSignature)
// 	err = signatureSetBytes.SetBytes(signature)
// 	if err != nil {
// 		return false, err
// 	}

// 	isValid = pubKeySig.Verify(signatureSetBytes, hash)
// 	return isValid, nil
// }

// GetPublicKeyInBase58CheckEncode returns the public key which is base58 check encoded
func (keySet KeySet) GetPublicKeyInBase58CheckEncode() string {
	return base58.Base58Check{}.Encode(keySet.PaymentAddress.Pk, common.ZeroByte)
}

func (keySet KeySet) GetReadOnlyKeyInBase58CheckEncode() string {
	return base58.Base58Check{}.Encode(keySet.ReadonlyKey.Rk, common.ZeroByte)
}

func (keySet KeySet) GetOTASecretKeyInBase58CheckEncode() string {
	return base58.Base58Check{}.Encode(keySet.OTAKey.OTASecret, common.ZeroByte)
}

// SignDataInBase58CheckEncode receives data and
// returns the signature that is base58 check encoded and is signed by private key in keySet
func (keySet KeySet) SignDataInBase58CheckEncode(data []byte) (string, error) {
	signatureByte, err := keySet.Sign(data)
	if err != nil {
		return common.EmptyString, NewCashecError(SignDataB58Err, err)
	}
	return base58.Base58Check{}.Encode(signatureByte, common.ZeroByte), nil
}

// ValidateDataB58 receives a data, a base58 check encoded signature (sigB58)
// and a base58 check encoded public key (pbkB58)
// It decodes pbkB58 and sigB58
// after that, it verifies the given signature is corresponding to data using verification key is pbkB58
// func ValidateDataB58(publicKeyInBase58CheckEncode string, signatureInBase58CheckEncode string, data []byte) error {
// 	// decode public key (verification key)
// 	decodedPubKey, _, err := base58.Base58Check{}.Decode(publicKeyInBase58CheckEncode)
// 	if err != nil {
// 		return NewCashecError(B58DecodePubKeyErr, nil)
// 	}
// 	validatorKeySet := KeySet{}
// 	validatorKeySet.PaymentAddress.Pk = make([]byte, len(decodedPubKey))
// 	copy(validatorKeySet.PaymentAddress.Pk[:], decodedPubKey)

// 	// decode the signature
// 	decodedSig, _, err := base58.Base58Check{}.Decode(signatureInBase58CheckEncode)
// 	if err != nil {
// 		return NewCashecError(B58DecodeSigErr, nil)
// 	}

// 	// validate the data and signature
// 	isValid, err := validatorKeySet.Verify(data, decodedSig)
// 	if err != nil {
// 		return NewCashecError(B58ValidateErr, nil)
// 	}
// 	if !isValid {
// 		return NewCashecError(InvalidDataValidateErr, nil)
// 	}
// 	return nil
// }
