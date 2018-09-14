package cashec

import (
	"github.com/ninjadotorg/cash-prototype/common"
	"github.com/ninjadotorg/cash-prototype/privacy/client"
)

type KeySet struct {
	PrivateKey  client.SpendingKey
	PublicKey   client.PaymentAddress
	ReadonlyKey client.ViewingKey
}



/**
GenerateKey - generate key set from seed byte[]
 */
func (self *KeySet) GenerateKey(seed []byte) (*KeySet) {
	copy(self.PrivateKey[:], common.HashB(seed))
	self.PublicKey = client.GenPaymentAddress(self.PrivateKey)
	self.ReadonlyKey = client.GenViewingKey(self.PrivateKey)
	return self
}

/**
ImportFromPrivateKeyByte - from private-key byte[], regenerate pub-key and readonly-key
 */
func (self *KeySet) ImportFromPrivateKeyByte(privateKey []byte) {
	copy(self.PrivateKey[:], privateKey)
	self.PublicKey = client.GenPaymentAddress(self.PrivateKey)
	self.ReadonlyKey = client.GenViewingKey(self.PrivateKey)
}

/**
ImportFromPrivateKeyByte - from private-key data, regenerate pub-key and readonly-key
 */
func (self *KeySet) ImportFromPrivateKey(privateKey *client.SpendingKey) {
	self.PrivateKey = *privateKey
	self.PublicKey = client.GenPaymentAddress(self.PrivateKey)
	self.ReadonlyKey = client.GenViewingKey(self.PrivateKey)
}

// func (self *KeySet) GenerateSignKey() (client.PrivateKey, error){
// 	// Generate signing key
// 	privKey, err := client.GenerateKey(rand.Reader)
// 	return *privKey, err
// }

// func (self *KeySet) Verify(data, signature []byte, pubKey client.PublicKey) (bool, error) {
// 	jsSig := new(JSSig)
// 	err := json.Unmarshal(signature, jsSig)
// 	if err != nil {
// 		return false, err
// 	}
// 	valid := client.VerifySign(&pubKey, data[:], jsSig.R, jsSig.S)
// 	return valid, nil
// }

// func (self *KeySet) Sign(data []byte, privKey client.PrivateKey) ([]byte, error) {
// 	// TODO(@0xkraken): implement signing using keypair
// 	jsSig := *new(JSSig)
// 	jsSig.R, jsSig.S, _= client.Sign(rand.Reader, &privKey, data[:])

// 	signed_data, err := json.Marshal(jsSig)
// 	if err != nil {
// 		return nil, err
// 	}

// 	//Calculate hi
// 	return signed_data, nil
// }
