package wallet

// import (
// 	"crypto/rand"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"github.com/incognitochain/incognito-chain/common"
// 	"golang.org/x/crypto/pbkdf2"
// 	"strings"
// )

// // deriveKey receives passPhrase and salt
// // if salt is empty array, we will random 8-byte array for salt
// // and return new 32-byte key using pbkdf2 method and salt
// func deriveKey(passPhrase string, salt []byte) ([]byte, []byte) {
// 	if len(salt) == 0 {
// 		salt = make([]byte, 8)
// 		rand.Read(salt)
// 	}
// 	return pbkdf2.Key([]byte(passPhrase), salt, 1000, common.AESKeySize, sha256.New), salt
// }

// // EncryptByPassPhrase receives passphrase and plaintext
// // it generates AES key from passPhrase and encrypt the plaintext
// // it returns encrypted plaintext (ciphertext) in string
// func encryptByPassPhrase(passphrase string, plaintext []byte) (string, error) {
// 	if len(plaintext) == 0 {
// 		return common.EmptyString, NewWalletError(InvalidPlaintextErr, nil)
// 	}

// 	// generate key from pass phrase
// 	key, salt := deriveKey(passphrase, nil)

// 	// init aes with key
// 	aes := common.AES{
// 		Key: key,
// 	}

// 	// encrypt plaintext
// 	cipherText, err := aes.Encrypt(plaintext)
// 	if err != nil {
// 		return common.EmptyString, err
// 	}

// 	// return data ciphertext and salt to generate decryption key
// 	saltEncode := hex.EncodeToString(salt)
// 	cipherTextEncode := hex.EncodeToString(cipherText)
// 	return saltEncode + "-" + cipherTextEncode, nil
// }

// // DecryptByPassPhrase receives passPhrase and ciphertext (in hex encode to string)
// // it generates AES key from passPhrase and decrypt the ciphertext
// // and returns plain text in bytes array
// func decryptByPassPhrase(passPhrase string, cipherText string) ([]byte, error) {
// 	arr := strings.Split(cipherText, "-")

// 	salt, err := hex.DecodeString(arr[0])
// 	if err != nil {
// 		return []byte{}, err
// 	}

// 	data, err := hex.DecodeString(arr[1])
// 	if err != nil {
// 		return []byte{}, err
// 	}

// 	// generate key from pass phrase and salt
// 	key, salt := deriveKey(passPhrase, salt)

// 	// init aes with key
// 	aes := common.AES{
// 		Key: key,
// 	}

// 	// decrypt ciphertext
// 	plaintext, err := aes.Decrypt(data)
// 	if err != nil {
// 		return []byte{}, err
// 	}

// 	return plaintext, nil
// }
