package blsmultisig

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	UnExpectedError = iota
	InvalidPrivateKeyErr
	InvalidPublicKeyErr
	InvalidDataSignErr
	InvalidInputParamsSizeErr
	InvalidCommitteeInfoErr
	DecompressFromByteErr
	MemCacheErr
	JSONError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	Message string
}{
	UnExpectedError:           {-1100, "Unexpected error"},
	InvalidPrivateKeyErr:      {-1101, "Private key is invalid"},
	InvalidPublicKeyErr:       {-1102, "Public key is invalid"},
	InvalidDataSignErr:        {-1103, "Signed data is invalid"},
	InvalidCommitteeInfoErr:   {-1104, "Committee's info is invalid"},
	InvalidInputParamsSizeErr: {-1105, "Len of Input Params is invalid"},
	DecompressFromByteErr:     {-1106, "Decompress bytes array to Elliptic point error"},
	JSONError:                 {-1107, "JSON Marshal, Unmarshal error"},
	MemCacheErr:               {-1108, "Memcache error"},
}

type BLSSignatureError struct {
	Code    int
	Message string
	err     error
}

func (e BLSSignatureError) Error() string {
	return fmt.Sprintf("%d: %s \n %+v", e.Code, e.Message, e.err)
}

func NewBLSSignatureError(key int, err error) error {
	return &BLSSignatureError{
		Code:    ErrCodeMessage[key].Code,
		Message: ErrCodeMessage[key].Message,
		err:     errors.Wrap(err, ErrCodeMessage[key].Message),
	}
}
