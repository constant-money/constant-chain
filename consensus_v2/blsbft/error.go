package blsbft

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	UnExpectedError = iota
	ConsensusTypeNotExistError
	ProducerSignatureError
	CommitteeSignatureError
	CombineSignatureError
	SignDataError
	LoadKeyError
	ConsensusAlreadyStartedError
	ConsensusAlreadyStoppedError
	DecodeValidationDataError
	EncodeValidationDataError
	BlockCreationError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	message string
}{
	UnExpectedError:              {-1000, "Unexpected error"},
	ConsensusTypeNotExistError:   {-1001, "Consensus type isn't exist"},
	ProducerSignatureError:       {-1002, "Producer signature error"},
	CommitteeSignatureError:      {-1003, "Committee signature error"},
	CombineSignatureError:        {-1004, "Combine signature error"},
	SignDataError:                {-1005, "Sign data error"},
	LoadKeyError:                 {-1006, "Load key error"},
	ConsensusAlreadyStartedError: {-1007, "consensus already started error"},
	ConsensusAlreadyStoppedError: {-1008, "consensus already stopped error"},
	DecodeValidationDataError:    {-1009, "Decode Validation Data error"},
	EncodeValidationDataError:    {-1010, "Encode Validation Data Error"},
	BlockCreationError:           {-1011, "Block Creation Error"},
}

type ConsensusError struct {
	Code    int
	Message string
	err     error
}

func (e ConsensusError) Error() string {
	return fmt.Sprintf("%d: %s \n %+v", e.Code, e.Message, e.err)
}

func NewConsensusError(key int, err error) error {
	return &ConsensusError{
		Code:    ErrCodeMessage[key].Code,
		Message: ErrCodeMessage[key].message,
		err:     errors.Wrap(err, ErrCodeMessage[key].message),
	}
}
