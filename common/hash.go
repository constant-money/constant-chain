package common

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/sha3"

	// "github.com/ethereum/go-ethereum/crypto"
)

var InvalidMaxHashSizeErr = errors.New("invalid max hash size")
var InvalidHashSizeErr = errors.New("invalid hash size")
var NilHashErr = errors.New("input hash is nil")

type Hash [HashSize]byte

// MarshalText converts hashObj string to bytes array
func (hashObj Hash) MarshalText() ([]byte, error) {
	return []byte(hashObj.String()), nil
}

// UnmarshalText reverts bytes array to hashObj
func (hashObj Hash) UnmarshalText(text []byte) error {
	copy(hashObj[:], text)
	return nil
}

// UnmarshalJSON unmarshal json data to hashObj
func (hashObj *Hash) UnmarshalJSON(data []byte) error {
	hashString := ""
	_ = json.Unmarshal(data, &hashString)
	hashObj.Decode(hashObj, hashString)
	return nil
}

// Format writes first few bytes of hash for debugging
func (hashObj *Hash) Format(f fmt.State, c rune) {
	if c == 'h' {
		t := hashObj.String()
		f.Write([]byte(t[:8]))
	} else {
		m := "%"
		for i := 0; i < 128; i++ {
			if f.Flag(i) {
				m += string(i)
			}
		}
		m += string(c)
		fmt.Fprintf(f, m, hashObj[:])
	}
}

// SetBytes sets the bytes array which represent the hash.
func (hashObj *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HashSize {
		return InvalidHashSizeErr
	}
	copy(hashObj[:], newHash)

	return nil
}

// GetBytes returns bytes array of hashObj
func (hashObj *Hash) GetBytes() []byte {
	newBytes := []byte{}
	newBytes = make([]byte, len(hashObj))
	copy(newBytes, hashObj[:])
	return newBytes
}

// NewHash receives a bytes array and returns a corresponding object Hash
func (hashObj Hash) NewHash(newHash []byte) (*Hash, error) {
	err := hashObj.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &hashObj, err
}

// String returns the Hash as the hexadecimal string of the byte-reversed hash.
func (hashObj Hash) String() string {
	for i := 0; i < HashSize/2; i++ {
		hashObj[i], hashObj[HashSize-1-i] = hashObj[HashSize-1-i], hashObj[i]
	}
	return hex.EncodeToString(hashObj[:])
}

// IsEqual returns true if target is the same as hashObj.
func (hashObj *Hash) IsEqual(target *Hash) bool {
	if hashObj == nil && target == nil {
		return true
	}
	if hashObj == nil || target == nil {
		return false
	}
	return *hashObj == *target
}

// NewHashFromStr creates a Hash from a hash string.  The string should be
// the hexadecimal string of a byte-reversed hash, but any missing characters
// result in zero padding at the end of the Hash.
func (hashObj Hash) NewHashFromStr(hash string) (*Hash, error) {
	err := hashObj.Decode(&hashObj, hash)
	if err != nil {
		return nil, err
	}
	return &hashObj, nil
}

// Decode decodes the byte-reversed hexadecimal string encoding of a Hash to a
// destination.
func (hashObj *Hash) Decode(dst *Hash, src string) error {
	// Return error if hash string is too long.
	if len(src) > MaxHashStringSize {
		return InvalidMaxHashSizeErr
	}

	// Hex decoder expects the hash to be a multiple of two.  When not, pad
	// with a leading zero.
	var srcBytes []byte
	if len(src)%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+len(src))
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}

	// Hex decode the source bytes to a temporary destination.
	var reversedHash Hash
	_, err := hex.Decode(reversedHash[HashSize-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return err
	}

	// Reverse copy from the temporary hash to destination.  Because the
	// temporary was zeroed, the written result will be correctly padded.
	for i, b := range reversedHash[:HashSize/2] {
		dst[i], dst[HashSize-1-i] = reversedHash[HashSize-1-i], b
	}

	return nil
}

// Cmp compare two hashes
// hash = target : return 0
// hash > target : return 1
// hash < target : return -1
func (hashObj *Hash) Cmp(target *Hash) (int, error) {
	if hashObj == nil || target == nil {
		return 0, NilHashErr
	}
	for i := 0; i < HashSize; i++ {
		if hashObj[i] > target[i] {
			return 1, nil
		}
		if hashObj[i] < target[i] {
			return -1, nil
		}
	}
	return 0, nil
}

// Keccak256 returns Keccak256 hash as a Hash object for storing and comparing
// func Keccak256(data ...[]byte) Hash {
// 	h := crypto.Keccak256(data...)
// 	r := Hash{}
// 	copy(r[:], h)
// 	return r
// }

// func HashArrayInterface(target interface{}) (Hash, error) {
// 	arr := InterfaceSlice(target)
// 	//if len(arr) == 0 {
// 	//	return Hash{}, errors.New("interface input is not an array")
// 	//}
// 	temp := []byte{0}
// 	for value := range arr {
// 		valueBytes, err := json.Marshal(&value)
// 		if err != nil {
// 			return Hash{}, err
// 		}
// 		temp = append(temp, valueBytes...)
// 	}
// 	return HashH(temp), nil
// }

func HashArrayOfHashArray(target []Hash) Hash {
	temp := []byte{0}
	for _, hash := range target {
		temp = append(temp, hash[:]...)
	}
	return HashH(temp)
}

func BytesToHash(b []byte) Hash {
	var h Hash
	_ = h.SetBytes(b)
	//if err != nil {
	//	panic(err)
	//}
	return h
}

func (h Hash) Bytes() []byte { return h[:] }

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h Hash) {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}
