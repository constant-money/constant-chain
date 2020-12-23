package schnorr

import (
	"crypto/subtle"
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

// SchnorrPublicKey represents Schnorr Publickey
// PK = G^SK + H^R
type SchnorrPublicKey struct {
	publicKey *operation.Point
	g, h      *operation.Point
}

func (schnorrPubKey SchnorrPublicKey) GetPublicKey() *operation.Point {
	return schnorrPubKey.publicKey
}

// SchnorrPrivateKey represents Schnorr Privatekey
type SchnorrPrivateKey struct {
	privateKey *operation.Scalar
	randomness *operation.Scalar
	publicKey  *SchnorrPublicKey
}

func (schnPrivKey SchnorrPrivateKey) GetPublicKey() *SchnorrPublicKey {
	return schnPrivKey.publicKey
}

// SchnSignature represents Schnorr Signature
type SchnSignature struct {
	e, z1, z2 *operation.Scalar
}

// Set sets Schnorr private key
func (privateKey *SchnorrPrivateKey) Set(sk *operation.Scalar, r *operation.Scalar) {
	pedRandom := operation.PedCom.G[operation.PedersenRandomnessIndex].GetKey()
	pedPrivate := operation.PedCom.G[operation.PedersenPrivateKeyIndex].GetKey()

	privateKey.privateKey = sk
	privateKey.randomness = r
	privateKey.publicKey = new(SchnorrPublicKey)
	privateKey.publicKey.g, _ = new(operation.Point).SetKey(&pedPrivate)
	privateKey.publicKey.h, _ = new(operation.Point).SetKey(&pedRandom)
	privateKey.publicKey.publicKey = new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], sk)
	privateKey.publicKey.publicKey.Add(privateKey.publicKey.publicKey, new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenRandomnessIndex], r))
}

// Set sets Schnorr public key
func (publicKey *SchnorrPublicKey) Set(pk *operation.Point) {
	pubKey := pk.GetKey()
	pedRandom := operation.PedCom.G[operation.PedersenRandomnessIndex].GetKey()
	pedPrivate := operation.PedCom.G[operation.PedersenPrivateKeyIndex].GetKey()

	publicKey.publicKey, _ = new(operation.Point).SetKey(&pubKey)
	publicKey.g, _ = new(operation.Point).SetKey(&pedPrivate)
	publicKey.h, _ = new(operation.Point).SetKey(&pedRandom)
}

//Sign is function which using for signing on hash array by private key
func (privateKey SchnorrPrivateKey) Sign(data []byte) (*SchnSignature, error) {
	if len(data) != common.HashSize {
		return nil, errhandler.NewPrivacyErr(errhandler.UnexpectedErr, errors.New("hash length must be 32 bytes"))
	}

	signature := new(SchnSignature)

	// has privacy
	if !privateKey.randomness.IsZero() {
		// generates random numbers s1, s2 in [0, Curve.Params().N - 1]

		s1 := operation.RandomScalar()
		s2 := operation.RandomScalar()

		// t = s1*G + s2*H
		t := new(operation.Point).ScalarMult(privateKey.publicKey.g, s1)
		t.Add(t, new(operation.Point).ScalarMult(privateKey.publicKey.h, s2))

		// E is the hash of elliptic point t and data need to be signed
		msg := append(t.ToBytesS(), data...)

		signature.e = operation.HashToScalar(msg)

		signature.z1 = new(operation.Scalar).Mul(privateKey.privateKey, signature.e)
		signature.z1 = new(operation.Scalar).Sub(s1, signature.z1)

		signature.z2 = new(operation.Scalar).Mul(privateKey.randomness, signature.e)
		signature.z2 = new(operation.Scalar).Sub(s2, signature.z2)

		return signature, nil
	}

	// generates random numbers s, k2 in [0, Curve.Params().N - 1]
	s := operation.RandomScalar()

	// t = s*G
	t := new(operation.Point).ScalarMult(privateKey.publicKey.g, s)

	// E is the hash of elliptic point t and data need to be signed
	msg := append(t.ToBytesS(), data...)
	signature.e = operation.HashToScalar(msg)

	// Z1 = s - e*sk
	signature.z1 = new(operation.Scalar).Mul(privateKey.privateKey, signature.e)
	signature.z1 = new(operation.Scalar).Sub(s, signature.z1)

	signature.z2 = nil

	return signature, nil
}

//Verify is function which using for verify that the given signature was signed by by privatekey of the public key
func (publicKey SchnorrPublicKey) Verify(signature *SchnSignature, data []byte) bool {
	if signature == nil {
		return false
	}
	rv := new(operation.Point).ScalarMult(publicKey.publicKey, signature.e)
	rv.Add(rv, new(operation.Point).ScalarMult(publicKey.g, signature.z1))
	if signature.z2 != nil {
		rv.Add(rv, new(operation.Point).ScalarMult(publicKey.h, signature.z2))
	}
	msg := append(rv.ToBytesS(), data...)

	ev := operation.HashToScalar(msg)
	return subtle.ConstantTimeCompare(ev.ToBytesS(), signature.e.ToBytesS()) == 1
}

func (sig SchnSignature) Bytes() []byte {
	bytes := append(sig.e.ToBytesS(), sig.z1.ToBytesS()...)
	// Z2 is nil when has no privacy
	if sig.z2 != nil {
		bytes = append(bytes, sig.z2.ToBytesS()...)
	}
	return bytes
}

func (sig *SchnSignature) SetBytes(bytes []byte) error {
	if len(bytes) != 2*operation.Ed25519KeySize && len(bytes) != 3 * operation.Ed25519KeySize{
		return errhandler.NewPrivacyErr(errhandler.InvalidInputToSetBytesErr, nil)
	}
	sig.e = new(operation.Scalar).FromBytesS(bytes[0:operation.Ed25519KeySize])
	sig.z1 = new(operation.Scalar).FromBytesS(bytes[operation.Ed25519KeySize : 2*operation.Ed25519KeySize])
	if len(bytes) == 3*operation.Ed25519KeySize {
		sig.z2 = new(operation.Scalar).FromBytesS(bytes[2*operation.Ed25519KeySize:])
	} else {
		sig.z2 = nil
	}

	return nil
}
