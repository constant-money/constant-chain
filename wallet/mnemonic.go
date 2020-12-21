package wallet

import (
// 	"crypto/rand"
// 	"crypto/sha512"
// 	"encoding/binary"
// 	"errors"
	"math/big"
// 	"strings"

// 	"golang.org/x/crypto/pbkdf2"
)

var (
	// Some bitwise operands for working with big.Ints
	last11BitsMask          = big.NewInt(2047)
	rightShift11BitsDivider = big.NewInt(2048)
	bigOne                  = big.NewInt(1)
	bigTwo                  = big.NewInt(2)

	// wordList is the set of words to use
	wordList []string

	// wordMap is a reverse lookup map for wordList
	wordMap map[string]int
)

// func init() {
// 	list := NewWordList("english")
// 	wordList = list
// 	wordMap = map[string]int{}
// 	for i, v := range wordList {
// 		wordMap[v] = i
// 	}
// }

// type MnemonicGenerator struct{}

// // NewEntropy will create random Entropy bytes
// // so long as the requested size bitSize is an appropriate size.
// // bitSize has to be a multiple 32 and be within the inclusive range of {128, 256}
// func (mnemonicGen *MnemonicGenerator) newEntropy(bitSize int) ([]byte, error) {
// 	err := validateEntropyBitSize(bitSize)
// 	if err != nil {
// 		return nil, NewWalletError(NewEntropyError, err)
// 	}

// 	// create bytes array for Entropy from bitSize
// 	entropy := make([]byte, bitSize/8)
// 	// random bytes array
// 	_, err = rand.Read(entropy)
// 	if err != nil {
// 		return nil, NewWalletError(NewEntropyError, err)
// 	}
// 	return entropy, nil
// }

// // NewMnemonic will return a string consisting of the Mnemonic words for
// // the given Entropy.
// // If the provide Entropy is invalid, an error will be returned.
// func (mnemonicGen *MnemonicGenerator) newMnemonic(entropy []byte) (string, error) {
// 	// Compute some lengths for convenience
// 	entropyBitLength := len(entropy) * 8
// 	checksumBitLength := entropyBitLength / 32
// 	sentenceLength := (entropyBitLength + checksumBitLength) / 11

// 	err := validateEntropyBitSize(entropyBitLength)
// 	if err != nil {
// 		return "", NewWalletError(NewMnemonicError, err)
// 	}

// 	// Add checksum to Entropy
// 	entropy = addChecksum(entropy)

// 	// Break Entropy up into sentenceLength chunks of 11 bits
// 	// For each word AND mask the rightmost 11 bits and find the word at that index
// 	// Then bitshift Entropy 11 bits right and repeat
// 	// Add to the last empty slot so we can work with LSBs instead of MSB

// 	// Entropy as an int so we can bitmask without worrying about bytes slices
// 	entropyInt := new(big.Int).SetBytes(entropy)

// 	// Slice to hold words in
// 	words := make([]string, sentenceLength)

// 	// Throw away big int for AND masking
// 	word := big.NewInt(0)

// 	for i := sentenceLength - 1; i >= 0; i-- {
// 		// Get 11 right most bits and bitshift 11 to the right for next time
// 		word.And(entropyInt, last11BitsMask)
// 		entropyInt.Div(entropyInt, rightShift11BitsDivider)

// 		// Get the bytes representing the 11 bits as a 2 byte slice
// 		wordBytes := padByteSlice(word.Bytes(), 2)

// 		// Convert bytes to an index and add that word to the list
// 		words[i] = wordList[binary.BigEndian.Uint16(wordBytes)]
// 	}

// 	return strings.Join(words, " "), nil
// }

// // MnemonicToByteArray takes a Mnemonic string and turns it into a byte array
// // suitable for creating another Mnemonic.
// // An error is returned if the Mnemonic is invalid.
// func (mnemonicGen *MnemonicGenerator) mnemonicToByteArray(mnemonic string, raw ...bool) ([]byte, error) {
// 	var (
// 		mnemonicSlice    = strings.Split(mnemonic, " ")
// 		entropyBitSize   = len(mnemonicSlice) * 11
// 		checksumBitSize  = entropyBitSize % 32
// 		fullByteSize     = (entropyBitSize-checksumBitSize)/8 + 1
// 		checksumByteSize = fullByteSize - (fullByteSize % 4)
// 	)

// 	// Pre validate that the Mnemonic is well formed and only contains words that
// 	// are present in the word list
// 	if !mnemonicGen.isMnemonicValid(mnemonic) {
// 		return nil, NewWalletError(MnemonicInvalidError, errors.New("invalid menomic"))
// 	}

// 	// Convert word indices to a `big.Int` representing the Entropy
// 	checksummedEntropy := big.NewInt(0)
// 	modulo := big.NewInt(2048)
// 	for _, v := range mnemonicSlice {
// 		index := big.NewInt(int64(wordMap[v]))
// 		checksummedEntropy.Mul(checksummedEntropy, modulo)
// 		checksummedEntropy.Add(checksummedEntropy, index)
// 	}

// 	// Calculate the unchecksummed Entropy so we can validate that the checksum is
// 	// correct
// 	checksumModulo := big.NewInt(0).Exp(bigTwo, big.NewInt(int64(checksumBitSize)), nil)
// 	rawEntropy := big.NewInt(0).Div(checksummedEntropy, checksumModulo)

// 	// Convert `big.Int`s to byte padded byte slices
// 	rawEntropyBytes := padByteSlice(rawEntropy.Bytes(), checksumByteSize)
// 	checksummedEntropyBytes := padByteSlice(checksummedEntropy.Bytes(), fullByteSize)

// 	// ValidateTransaction that the checksum is correct
// 	newChecksummedEntropyBytes := padByteSlice(addChecksum(rawEntropyBytes), fullByteSize)
// 	if !compareByteSlices(checksummedEntropyBytes, newChecksummedEntropyBytes) {
// 		return nil, errors.New("checksum incorrect")
// 	}

// 	if raw != nil && raw[0] {
// 		return rawEntropyBytes, nil
// 	}
// 	return checksummedEntropyBytes, nil
// }

// // NewSeed creates a hashed Seed output given a provided string and password.
// // No checking is performed to validate that the string provided is a valid Mnemonic.
// func (mnemonicGen *MnemonicGenerator) NewSeed(mnemonic string, password string) []byte {
// 	return pbkdf2.Key([]byte(mnemonic), []byte("Mnemonic"+password), 2048, seedKeyLen, sha512.New)
// }

// // IsMnemonicValid attempts to verify that the provided Mnemonic is valid.
// // Validity is determined by both the number of words being appropriate,
// // and that all the words in the Mnemonic are present in the word list.
// func (mnemonicGen *MnemonicGenerator) isMnemonicValid(mnemonic string) bool {
// 	// Create a list of all the words in the Mnemonic sentence
// 	words := strings.Fields(mnemonic)

// 	// Get word count
// 	wordCount := len(words)

// 	// The number of words should be 12, 15, 18, 21 or 24
// 	if wordCount%3 != 0 || wordCount < 12 || wordCount > 24 {
// 		return false
// 	}

// 	// Check if all words belong in the wordlist
// 	for _, word := range words {
// 		if _, ok := wordMap[word]; !ok {
// 			return false
// 		}
// 	}

// 	return true
// }

// // validateEntropyBitSize ensures that Entropy is the correct size for being a
// // Mnemonic.
// func validateEntropyBitSize(bitSize int) error {
// 	if (bitSize%32) != 0 || bitSize < 128 || bitSize > 256 {
// 		return errors.New("entropy length must be [128, 256] and a multiple of 32")
// 	}
// 	return nil
// }

// // splitMnemonicWords splits mnemonic string into list of words in that mnemonic string
// func (mnemonicGen *MnemonicGenerator) splitMnemonicWords(mnemonic string) ([]string, bool) {
// 	// Create a list of all the words in the Mnemonic sentence
// 	words := strings.Fields(mnemonic)

// 	//Get num of words
// 	numOfWords := len(words)

// 	// The number of words should be 12, 15, 18, 21 or 24
// 	if numOfWords%3 != 0 || numOfWords < 12 || numOfWords > 24 {
// 		return nil, false
// 	}
// 	return words, true
// }
