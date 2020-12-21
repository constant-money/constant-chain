package wallet

import (
	"fmt"
	"github.com/pkg/errors"
)

const (
	InvalidChecksumErr = iota
	WrongPassphraseErr
	ExistedAccountErr
	ExistedAccountNameErr
	UnexpectedErr
	EmptyWalletNameErr
	NotFoundAccountErr
	JsonMarshalErr
	JsonUnmarshalErr
	WriteFileErr
	ReadFileErr
	AESEncryptErr
	AESDecryptErr
	InvalidKeyTypeErr
	InvalidPlaintextErr
	NewChildKeyError
	NewEntropyError
	NewMnemonicError
	MnemonicInvalidError
	InvalidSeserializedKey
)

var ErrCodeMessage = map[int]struct {
	code    int
	message string
}{
	UnexpectedErr: {-1, "Unexpected error"},

	InvalidChecksumErr:     {-1000, "Checksum does not match"},
	WrongPassphraseErr:     {-1001, "Wrong passphrase"},
	ExistedAccountErr:      {-1002, "Existed account"},
	ExistedAccountNameErr:  {-1002, "Existed account name"},
	EmptyWalletNameErr:     {-1003, "Wallet name is empty"},
	NotFoundAccountErr:     {-1004, "Account wallet is not found"},
	JsonMarshalErr:         {-1005, "Can not json marshal"},
	JsonUnmarshalErr:       {-1006, "Can not json unmarshal"},
	WriteFileErr:           {-1007, "Can not write file"},
	ReadFileErr:            {-1008, "Can not read file"},
	AESEncryptErr:          {-1009, "Can not AES encrypt data"},
	AESDecryptErr:          {-1010, "Can not AES decrypt data"},
	InvalidKeyTypeErr:      {-1011, "Serialized key type is invalid"},
	InvalidPlaintextErr:    {-1012, "Plaintext is invalid"},
	NewChildKeyError:       {-1013, "Can not create new child key"},
	NewEntropyError:        {-1014, "Can not create entropy"},
	NewMnemonicError:       {-1015, "Can not create mnemonic"},
	MnemonicInvalidError:   {-1016, "Mnemonic is invalid"},
	InvalidSeserializedKey: {-1016, "Serialized key is invalid"},
}

type WalletError struct {
	code    int
	message string
	err     error
}

func (e WalletError) Error() string {
	return fmt.Sprintf("%+v: %+v", e.code, e.message)
}

func (e WalletError) GetCode() int {
	return e.code
}

func NewWalletError(key int, err error) *WalletError {
	return &WalletError{
		err:     errors.Wrap(err, ErrCodeMessage[key].message),
		code:    ErrCodeMessage[key].code,
		message: ErrCodeMessage[key].message,
	}
}
