package privacy

import (
	"fmt"
	"github.com/pkg/errors"
)

const (
	UnexpectedErr = iota
	InvalidOutputValue
	MarshalPaymentProofErr
	UnmarshalPaymentProofErr
	SetBytesProofErr
	EncryptOutputCoinErr
	DecryptOutputCoinErr
	DecompressTransmissionKeyErr
	VerifySerialNumberNoPrivacyProofFailedErr
	VerifyCoinCommitmentInputFailedErr
	VerifyCoinCommitmentOutputFailedErr
	VerifyAmountNoPrivacyFailedErr
	VerifyOneOutOfManyProofFailedErr
	VerifySerialNumberPrivacyProofFailedErr
	VerifyAggregatedProofFailedErr
	VerifyAggregatedProofNewFailedErr
	VerifyAmountPrivacyFailedErr
	CalInnerProductErr
	ProveSerialNumberNoPrivacyErr
	ProveOneOutOfManyErr
	ProveSerialNumberPrivacyErr
	ProveAggregatedRangeErr
	InvalidInputToSetBytesErr
	CommitNewOutputCoinNoPrivacyErr
	ConvertMultiSigToBytesErr
	SignMultiSigErr
	InvalidLengthMultiSigErr
	InvalidMultiSigErr
)

var ErrCodeMessage = map[int]struct {
	Code    int
	Message string
}{
	UnexpectedErr: {-9000, "Unexpected error"},

	InvalidOutputValue:              {-9001, "Invalid output value"},
	MarshalPaymentProofErr:          {-9002, "Marshal payment proof error"},
	UnmarshalPaymentProofErr:        {-9003, "Unmarshal payment proof error"},
	SetBytesProofErr:                {-9004, "Set bytes payment proof error"},
	EncryptOutputCoinErr:            {-9005, "Encrypt output coins error"},
	DecryptOutputCoinErr:            {-9006, "Decrypt output coins error"},
	DecompressTransmissionKeyErr:    {-9007, "Can not decompress transmission key error"},
	CalInnerProductErr:              {-9008, "Calculate inner product between two vectors error"},
	InvalidInputToSetBytesErr:       {-9009, "Length of input data is zero, can not set bytes"},
	CommitNewOutputCoinNoPrivacyErr: {-9010, "Can not commit output coin's details when creating tx without privacy"},
	ConvertMultiSigToBytesErr:       {-9011, "Can not convert multi sig to bytes array"},
	SignMultiSigErr:                 {-9012, "Can not sign multi sig"},
	InvalidLengthMultiSigErr:        {-9013, "Invalid length of multi sig signature"},
	InvalidMultiSigErr:              {-9014, "invalid multiSig for converting to bytes array"},

	ProveSerialNumberNoPrivacyErr: {-9100, "Proving serial number no privacy proof error"},
	ProveOneOutOfManyErr:          {-9101, "Proving one out of many proof error"},
	ProveSerialNumberPrivacyErr:   {-9102, "Proving serial number privacy proof error"},
	ProveAggregatedRangeErr:       {-9103, "Proving aggregated range proof error"},

	VerifySerialNumberNoPrivacyProofFailedErr: {-9201, "Verify serial number no privacy proof failed"},
	VerifyCoinCommitmentInputFailedErr:        {-9202, "Verify coin commitment of input coin failed"},
	VerifyCoinCommitmentOutputFailedErr:       {-9203, "Verify coin commitment of output coin failed"},
	VerifyAmountNoPrivacyFailedErr:            {-9204, "Sum of input coins' amount is not equal sum of output coins' amount"},
	VerifyOneOutOfManyProofFailedErr:          {-9205, "Verify one out of many proof failed"},
	VerifySerialNumberPrivacyProofFailedErr:   {-9206, "Verify serial number privacy proof failed"},
	VerifyAggregatedProofFailedErr:            {-9207, "Verify aggregated proof failed"},
	VerifyAggregatedProofNewFailedErr:            {-9209, "Verify aggregated proof new failed"},
	VerifyAmountPrivacyFailedErr:              {-9208, "Sum of input coins' amount is not equal sum of output coins' amount when creating private tx"},
}

type PrivacyError struct {
	Code    int
	Message string
	err     error
}

func (e PrivacyError) Error() string {
	return fmt.Sprintf("%+v: %+v %+v", e.Code, e.Message, e.err)
}

func (e PrivacyError) GetCode() int {
	return e.Code
}

func NewPrivacyErr(key int, err error) *PrivacyError {
	return &PrivacyError{
		err:     errors.Wrap(err, ErrCodeMessage[key].Message),
		Code:    ErrCodeMessage[key].Code,
		Message: ErrCodeMessage[key].Message,
	}
}
