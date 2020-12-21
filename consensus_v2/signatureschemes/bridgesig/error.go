package bridgesig

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	UnExpectedError = iota
	InvalidPrivateKeyErr
	InvalidPublicKeyErr
	SignDataErr
	InvalidDataSignErr
	InvalidInputParamsSizeErr
	InvalidCommitteeInfoErr
	InvalidSignatureErr
	DecompressFromByteErr
	JSONError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	message string
}{
	UnExpectedError:           {-1200, "Unexpected error"},
	InvalidPrivateKeyErr:      {-1201, "Private key is invalid"},
	InvalidPublicKeyErr:       {-1202, "Public key is invalid"},
	InvalidDataSignErr:        {-1203, "Signed data is invalid"},
	InvalidCommitteeInfoErr:   {-1204, "Committee's info is invalid"},
	InvalidInputParamsSizeErr: {-1205, "Len of Input Params is invalid"},
	DecompressFromByteErr:     {-1206, "Decompress bytes array to Elliptic point error"},
	JSONError:                 {-1207, "JSON Marshal, Unmarshal error"},
	InvalidSignatureErr:       {-1208, "Invalid signature"},
	SignDataErr:               {-1209, "Can not sign data"},
}

type BriSignatureError struct {
	Code    int
	Message string
	err     error
}

func (e BriSignatureError) Error() string {
	return fmt.Sprintf("%d: %s \n %+v", e.Code, e.Message, e.err)
}

func NewBriSignatureError(key int, err error) error {
	return &BriSignatureError{
		Code:    ErrCodeMessage[key].Code,
		Message: ErrCodeMessage[key].message,
		err:     errors.Wrap(err, ErrCodeMessage[key].message),
	}
}
