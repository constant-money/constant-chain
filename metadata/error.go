package metadata

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	UnexpectedError = iota

	IssuingEthRequestDecodeInstructionError
	IssuingEthRequestUnmarshalJsonError
	IssuingEthRequestNewIssuingETHRequestFromMapEror
	IssuingEthRequestValidateTxWithBlockChainError
	IssuingEthRequestValidateSanityDataError
	IssuingEthRequestBuildReqActionsError
	IssuingEthRequestVerifyProofAndParseReceipt

	IssuingRequestDecodeInstructionError
	IssuingRequestUnmarshalJsonError
	IssuingRequestNewIssuingRequestFromMapEror
	IssuingRequestValidateTxWithBlockChainError
	IssuingRequestValidateSanityDataError
	IssuingRequestBuildReqActionsError
	IssuingRequestVerifyProofAndParseReceipt

	BeaconBlockRewardNewBeaconBlockRewardInfoFromStrError
	BeaconBlockRewardBuildInstructionForBeaconBlockRewardError

	StopAutoStakingRequestNotInCommitteeListError
	StopAutoStakingRequestGetStakingTransactionError
	StopAutoStakingRequestStakingTransactionNotFoundError
	StopAutoStakingRequestInvalidTransactionSenderError
	StopAutoStakingRequestNoAutoStakingAvaiableError
	StopAutoStakingRequestTypeAssertionError
	StopAutoStakingRequestAlreadyStopError

	WrongIncognitoDAOPaymentAddressError

	// pde
	PDEWithdrawalRequestFromMapError
	CouldNotGetExchangeRateError
	RejectInvalidFee
	PDEFeeWithdrawalRequestFromMapError

	// portal
	PortalRequestPTokenParamError
	PortalRedeemRequestParamError
	PortalRedeemLiquidateExchangeRatesParamError

	// Unstake
	UnStakingRequestNotInCommitteeListError
	UnStakingRequestGetStakerInfoError
	UnStakingRequestNotFoundStakerInfoError
	UnStakingRequestStakingTransactionNotFoundError
	UnStakingRequestInvalidTransactionSenderError
	UnStakingRequestNoAutoStakingAvaiableError
	UnStakingRequestTypeAssertionError
	UnStakingRequestAlreadyStopError
	UnStakingRequestInvalidFormatRequestKey
)

var ErrCodeMessage = map[int]struct {
	Code    int
	Message string
}{
	UnexpectedError: {-1, "Unexpected error"},

	// -1xxx issuing eth request
	IssuingEthRequestDecodeInstructionError:          {-1001, "Can not decode instruction"},
	IssuingEthRequestUnmarshalJsonError:              {-1002, "Can not unmarshall json"},
	IssuingEthRequestNewIssuingETHRequestFromMapEror: {-1003, "Can no new issuing eth request from map"},
	IssuingEthRequestValidateTxWithBlockChainError:   {-1004, "Validate tx with block chain error"},
	IssuingEthRequestValidateSanityDataError:         {-1005, "Validate sanity data error"},
	IssuingEthRequestBuildReqActionsError:            {-1006, "Build request action error"},
	IssuingEthRequestVerifyProofAndParseReceipt:      {-1007, "Verify proof and parse receipt"},

	// -2xxx issuing eth request
	IssuingRequestDecodeInstructionError:        {-2001, "Can not decode instruction"},
	IssuingRequestUnmarshalJsonError:            {-2002, "Can not unmarshall json"},
	IssuingRequestNewIssuingRequestFromMapEror:  {-2003, "Can no new issuing eth request from map"},
	IssuingRequestValidateTxWithBlockChainError: {-2004, "Validate tx with block chain error"},
	IssuingRequestValidateSanityDataError:       {-2005, "Validate sanity data error"},
	IssuingRequestBuildReqActionsError:          {-2006, "Build request action error"},
	IssuingRequestVerifyProofAndParseReceipt:    {-2007, "Verify proof and parse receipt"},

	// -3xxx beacon block reward
	BeaconBlockRewardNewBeaconBlockRewardInfoFromStrError:      {-3000, "Can not new beacon block reward from string"},
	BeaconBlockRewardBuildInstructionForBeaconBlockRewardError: {-3001, "Can not build instruction for beacon block reward"},

	// -4xxx staking error
	StopAutoStakingRequestNotInCommitteeListError:         {-4000, "Stop Auto-Staking Request Not In Committee List Error"},
	StopAutoStakingRequestStakingTransactionNotFoundError: {-4001, "Stop Auto-Staking Request Staking Transaction Not Found Error"},
	StopAutoStakingRequestInvalidTransactionSenderError:   {-4002, "Stop Auto-Staking Request Invalid Transaction Sender Error"},
	StopAutoStakingRequestNoAutoStakingAvaiableError:      {-4003, "Stop Auto-Staking Request No Auto Staking Avaliable Error"},
	StopAutoStakingRequestTypeAssertionError:              {-4004, "Stop Auto-Staking Request Type Assertion Error"},
	StopAutoStakingRequestAlreadyStopError:                {-4005, "Stop Auto Staking Request Already Stop Error"},
	StopAutoStakingRequestGetStakingTransactionError:      {-4006, "Stop Auto Staking Request Get Staking Transaction Error"},
	UnStakingRequestNotInCommitteeListError:               {-4100, "Unstaking Request Not In Committee List Error"},
	UnStakingRequestGetStakerInfoError:                    {-4101, "Unstaking Request Get Staker Info Error"},
	UnStakingRequestNotFoundStakerInfoError:               {-4102, "Unstaking Request Not Found Staker Info Error"},
	UnStakingRequestStakingTransactionNotFoundError:       {-4103, "Unstaking Request Staking Transaction Not Found Error"},
	UnStakingRequestInvalidTransactionSenderError:         {-4104, "Unstaking Request Invalid Transaction Sender Error"},
	UnStakingRequestNoAutoStakingAvaiableError:            {-4105, "UnStaking Request No Auto Staking Available Error"},
	UnStakingRequestTypeAssertionError:                    {-4106, "UnStaking Request Type Assertion Error"},
	UnStakingRequestAlreadyStopError:                      {-4107, "UnStaking Request Already Stop Error"},
	UnStakingRequestInvalidFormatRequestKey:               {-4108, "Unstaking Request Key Is Invalid Format"},
	// -5xxx dev reward error
	WrongIncognitoDAOPaymentAddressError: {-5001, "Invalid dev account"},

	// pde
	PDEWithdrawalRequestFromMapError: {-6001, "PDE withdrawal request Error"},
	CouldNotGetExchangeRateError:     {-6002, "Could not get the exchange rate error"},
	RejectInvalidFee:                 {-6003, "Reject invalid fee"},

	// portal
	PortalRequestPTokenParamError:                {-7001, "Portal request ptoken param error"},
	PortalRedeemRequestParamError:                {-7002, "Portal redeem request param error"},
	PortalRedeemLiquidateExchangeRatesParamError: {-7003, "Portal redeem liquidate exchange rates param error"},
}

type MetadataTxError struct {
	Code    int    // The code to send with reject messages
	Message string // Human readable message of the issue
	Err     error
}

// Error satisfies the error interface and prints human-readable errors.
func (e MetadataTxError) Error() string {
	return fmt.Sprintf("%d: %s %+v", e.Code, e.Message, e.Err)
}

func NewMetadataTxError(key int, err error, params ...interface{}) *MetadataTxError {
	return &MetadataTxError{
		Code:    ErrCodeMessage[key].Code,
		Message: fmt.Sprintf(ErrCodeMessage[key].Message, params),
		Err:     errors.Wrap(err, ErrCodeMessage[key].Message),
	}
}
