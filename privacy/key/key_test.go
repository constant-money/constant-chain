package key

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	for i := 0; i < 1; i++ {
		// random seed
		seed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

		// generate private key from seed
		privateKey := GeneratePrivateKey(seed)

		// generate public key from private key
		publicKey := GeneratePublicKey(privateKey)

		// check public key
		publicKeyPrime := new(operation.Point).ScalarMultBase(new(operation.Scalar).FromBytesS(privateKey))
		assert.Equal(t, publicKeyPrime.ToBytes(), SliceToArray(publicKey))

		// generate receiving key from private key
		receivingKey := GenerateReceivingKey(privateKey)

		// generate transmission key from receiving key
		transmissionKey := GenerateTransmissionKey(receivingKey)

		// decompress transmission key to transmissionKeyPoint
		transmissionKeyPrime := new(operation.Point).ScalarMultBase(new(operation.Scalar).FromBytesS(receivingKey))
		assert.Equal(t, transmissionKeyPrime.ToBytes(), SliceToArray(transmissionKey))

		// generate payment address from private key
		paymentAddress := GeneratePaymentAddress(privateKey)
		assert.Equal(t, publicKey, paymentAddress.Pk)
		assert.Equal(t, transmissionKey, paymentAddress.Tk)

		// convert payment address to bytes array
		paymentAddrBytes := paymentAddress.Bytes()

		// new payment address to set bytes
		paymentAddress2 := new(PaymentAddress)
		paymentAddress2.SetBytes(paymentAddrBytes)

		assert.Equal(t, paymentAddress.Pk, paymentAddress2.Pk)
		assert.Equal(t, paymentAddress.Tk, paymentAddress2.Tk)

		fmt.Printf("Private key: %v\n", privateKey)
		fmt.Printf("publicKey: %v\n", publicKey)
		fmt.Printf("receivingKey: %v\n", receivingKey)
		fmt.Printf("transmissionKey: %v\n", transmissionKey)

	}
}
