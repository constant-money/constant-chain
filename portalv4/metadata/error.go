package metadata

import (
	"fmt"
	"github.com/pkg/errors"
)

const (
	UnexpectedError = iota

	PortalBurnPTokenMetaError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	Message string
}{
	UnexpectedError: {-17000, "Unexpected error"},

	PortalBurnPTokenMetaError: {-17001, "Portal burn ptoken metadata error"},
}

type PortalV4MetadataError struct {
	Code    int    // The code to send with reject messages
	Message string // Human readable message of the issue
	Err     error
}

// Error satisfies the error interface and prints human-readable errors.
func (e PortalV4MetadataError) Error() string {
	return fmt.Sprintf("%d: %s %+v", e.Code, e.Message, e.Err)
}

func NewPortalV4MetadataError(key int, err error, params ...interface{}) *PortalV4MetadataError {
	return &PortalV4MetadataError{
		Code:    ErrCodeMessage[key].Code,
		Message: fmt.Sprintf(ErrCodeMessage[key].Message, params),
		Err:     errors.Wrap(err, ErrCodeMessage[key].Message),
	}
}
