package rpcservice

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	UnexpectedError = iota
	AlreadyStartedError
	JsonError

	RPCInvalidRequestError
	RPCMethodNotFoundError
	RPCInvalidParamsError
	RPCInvalidMethodPermissionError
	RPCInternalError
	RPCParseError

	InvalidTypeError
	AuthFailError
	InvalidSenderPrivateKeyError
	InvalidSenderViewingKeyError
	InvalidReceiverPaymentAddressError
	ListTokenNotFoundError
	CanNotSignError
	GetOutputCoinError
	GetOutputCoinsVer1Error
	GetOutputCoinsVer2Error
	CreateTxDataError
	SendTxDataError
	Base58ChedkDataOfTxInvalid
	JsonDataOfTxInvalid
	TxTypeInvalidError
	TxNotExistedInMemAndBLockError
	UnsubcribeError
	SubcribeError
	NetworkError
	TokenIsInvalidError
	GetClonedBeaconBestStateError
	GetBeaconBestBlockHashError
	GetBeaconBestBlockError
	GetClonedShardBestStateError
	GetShardBlockByHeightError
	GetShardBestBlockError
	GetLatestFinalizedShardBlockError
	GetShardBlockByHashError
	GetBeaconBlockByHashError
	GetBeaconBlockByHeightError
	GeTxFromPoolError
	NoSwapConfirmInst
	GetKeySetFromPrivateKeyError
	GetPDEStateError
	ListCommitteeRewardError
	GetRewardAmountError
	ListDecryptedOutputCoinsByKeyError
	ListUnspentOutputCoinsByKeyError
	SendRawTransactionError
	BuildTokenParamError
	BuildPrivacyTokenParamError
	GetListPrivacyCustomTokenBalanceError
	GetPrivacyTokenError
	// reject tx
	RejectInvalidTxFeeTooLargeError
	RejectInvalidTxFeeError
	RejectInvalidTxSizeError
	RejectInvalidTxTypeError
	RejectInvalidTxError
	RejectDoubleSpendTxError
	RejectDuplicateTxInPoolError
	RejectInvalidTxVersionError
	RejectSanityTxLocktime
	RejectReplacementTx
	TxPoolRejectTxError
	RejectInvalidFeeError

	//portal
	GetFinalExchangeRatesError
	ConvertExchangeRatesError
	GetPortingRequestFeesError
	GetExchangeRatesIsEmpty
	GetPortingRequestError
	GetPortingRequestIsEmpty

	GetCustodianWithdrawError
	GetReqPTokenStatusError
	GetCustodianDepositError
	GetPortalStateError
	GetReqUnlockCollateralStatusError
	GetReqRedeemStatusError
	GetRequestWithdrawRewardStatusError
	GetPortalRewardError
	GetReqMatchingRedeemStatusError
	GetReqRedeemFromLiquidationPoolStatusError

	GetCustodianLiquidationStatusError
	GetTpExchangeRatesLiquidationError
	GetTpExchangeRatesLiquidationByTokenIdError
	GetExchangeRatesLiquidationPoolError

	GetAmountNeededForCustodianDepositLiquidationError
	GetCustodianTopupStatusError
	GetCustodianTopupWaitingPortingStatusError
	GetAmountTopUpWaitingPortingError

	// relaying
	GetRelayingBNBHeaderByBlockHeightError
	GetBTCRelayingBestState
	GetBTCBlockByHash
	GetRelayingBNBHeaderError
	GetLatestBNBHeaderBlockHeightError

	// feature reward
	GetRewardFeatureByFeatureNameError

	// Init State Root hash
	InitStateRootHash

	GetAllBeaconViews

	// Restore
	RestoreBeaconCommittee
	RestoreShardCommittee
	RestoreBeaconPendingValidator
	RestoreShardPendingValidator
	RestoreCandidateBeaconWaitingForCurrentRandom
	RestoreCandidateBeaconWaitingForNextRandom
	RestoreCandidateShardWaitingForCurrentRandom
	RestoreCandidateShardWaitingForNextRandom

	GetTotalStakerError
)

// Standard JSON-RPC 2.0 errors.
var ErrCodeMessage = map[int]struct {
	Code    int
	Message string
}{
	// general
	UnexpectedError:     {-1, "Unexpected error"},
	AlreadyStartedError: {-2, "RPC server is already started"},
	NetworkError:        {-3, "Network Error, failed to send request to RPC server"},
	JsonError:           {-4, "Json error"},

	// validate component -1xxx
	RPCInvalidRequestError:                {-1001, "Invalid request"},
	RPCMethodNotFoundError:                {-1002, "Method not found"},
	RPCInvalidParamsError:                 {-1003, "Invalid parameters"},
	RPCInternalError:                      {-1004, "Internal error"},
	RPCParseError:                         {-1005, "Parse error"},
	InvalidTypeError:                      {-1006, "Invalid type"},
	AuthFailError:                         {-1007, "Auth failure"},
	RPCInvalidMethodPermissionError:       {-1008, "Invalid method permission"},
	InvalidReceiverPaymentAddressError:    {-1009, "Invalid receiver paymentaddress"},
	ListTokenNotFoundError:                {-1010, "Can not find any token"},
	CanNotSignError:                       {-1011, "Can not sign with key"},
	InvalidSenderPrivateKeyError:          {-1012, "Invalid sender's key"},
	GetOutputCoinError:                    {-1013, "Can not get output coin"},
	TxTypeInvalidError:                    {-1014, "Invalid tx type"},
	InvalidSenderViewingKeyError:          {-1015, "Invalid viewing key"},
	RejectInvalidTxFeeError:               {-1016, "Reject invalid fee"},
	TxNotExistedInMemAndBLockError:        {-1017, "Tx is not existed in mem and block"},
	TokenIsInvalidError:                   {-1018, "Token is invalid"},
	GetKeySetFromPrivateKeyError:          {-1019, "Get KeySet From Private Key Error"},
	GetListPrivacyCustomTokenBalanceError: {-1020, "Get List Privacy Custom Token Balance Error"},
	GetPrivacyTokenError:                  {-1021, "Get Privacy Token Error"},
	GetOutputCoinsVer1Error:               {-1022, "Get output coins of ver 1 in database error"},
	GetOutputCoinsVer2Error:			   {-1023, "Get output coins of ver 2 in database error"},
	// for block -2xxx
	GetShardBlockByHeightError:        {-2000, "Get shard block by height error"},
	GetShardBlockByHashError:          {-2001, "Get shard block by hash error"},
	GetShardBestBlockError:            {-2002, "Get shard best block error"},
	GetBeaconBlockByHashError:         {-2003, "Get beacon block by hash error"},
	GetBeaconBlockByHeightError:       {-2004, "Get beacon block by height error"},
	GetBeaconBestBlockHashError:       {-2004, "Get beacon best block hash error"},
	GetBeaconBestBlockError:           {-2005, "Get beacon best block error"},
	GetLatestFinalizedShardBlockError: {-2006, "Get Latest Finalized Shard Block Error"},

	// best state -3xxx
	GetClonedBeaconBestStateError: {-3000, "Get Cloned Beacon Best State Error"},
	GetClonedShardBestStateError:  {-3001, "Get Cloned Shard Best State Error"},

	// tx -4xxx
	CreateTxDataError:                  {-4001, "Can not create tx"},
	SendTxDataError:                    {-4002, "Can not send tx"},
	Base58ChedkDataOfTxInvalid:         {-4003, "Base58Check encode data of tx is invalid, can not decode"},
	JsonDataOfTxInvalid:                {-4004, "Json string data of tx is invalid, can not unmarshal"},
	ListDecryptedOutputCoinsByKeyError: {-4005, "List Output Coins By Key Error"},
	ListUnspentOutputCoinsByKeyError:   {-4006, "List Unspent Output Coins By Key Error"},
	SendRawTransactionError:            {-4007, "Send Raw Transaction Error"},
	BuildTokenParamError:               {-4008, "Build Token Param Error"},
	BuildPrivacyTokenParamError:        {-4009, "Build Privacy Token Param Error"},
	// socket/subcribe -5xxx
	SubcribeError:   {-5000, "Failed to subcribe"},
	UnsubcribeError: {-5001, "Failed to unsubcribe"},

	// tx pool -6xxx
	GeTxFromPoolError:               {-6000, "Get tx from mempool error"},
	TxPoolRejectTxError:             {-6001, "Pool reject tx by unexpected error"},
	RejectInvalidTxSizeError:        {-6002, "Pool reject tx by invalid size"},
	RejectInvalidTxTypeError:        {-6003, "Pool reject tx by invalid type"},
	RejectInvalidTxError:            {-6004, "Pool reject invalid tx: signature, or proof or verify by itself fail"},
	RejectDoubleSpendTxError:        {-6005, "Pool reject double spend tx, double spend with blockchain or mempool"},
	RejectDuplicateTxInPoolError:    {-6006, "Tx already exist in pool"},
	RejectInvalidTxVersionError:     {-6007, "Reject tx by invalid version"},
	RejectSanityTxLocktime:          {-6008, "Reject wrong tx by locktime"},
	RejectReplacementTx:             {-6009, "Reject error replacement or cancel transaction"},
	RejectInvalidFeeError:           {-6010, "Reject Invalid Fee Error"},
	RejectInvalidTxFeeTooLargeError: {-6011, "Reject invalid tx fee because fee is too large"},

	// decentralized bridge
	NoSwapConfirmInst: {-7000, "No swap confirm instruction found in block"},

	// pde
	GetPDEStateError: {-8000, "Get pde state error"},

	//portal
	GetFinalExchangeRatesError:                         {-9000, "Get get final exchange rates error"},
	GetReqPTokenStatusError:                            {-9001, "Get request ptoken status error"},
	GetCustodianDepositError:                           {-9002, "Get custodian deposit status error"},
	GetPortalStateError:                                {-9003, "Get portal state error"},
	GetPortingRequestError:                             {-9004, "Get portal request error"},
	GetReqUnlockCollateralStatusError:                  {-9005, "Get status of request unlock collateral error"},
	GetReqRedeemStatusError:                            {-9006, "Get status of request redeem by redeemId error"},
	GetAmountNeededForCustodianDepositLiquidationError: {-9007, "Get amount needed for custodian deposit liquidation error"},
	GetExchangeRatesLiquidationPoolError:               {-9008, "Get exchange rates liquidation pool error"},
	GetCustodianWithdrawError:                          {-9009, "Get custodian withdraw error"},
	GetPortalRewardError:                               {-9010, "Get portal reward error"},
	GetRequestWithdrawRewardStatusError:                {-9011, "Get request withdraw portal reward error"},
	ConvertExchangeRatesError:                          {-9012, "Converting exchange rates error"},
	GetPortingRequestFeesError:                         {-9013, "Get porting request fees error"},
	GetReqMatchingRedeemStatusError:                    {-9014, "Get req matching redeem status error"},
	GetCustodianTopupStatusError:                       {-9015, "Get custodian top up status error"},
	GetCustodianTopupWaitingPortingStatusError:         {-9016, "Get custodian top up for waiting porting status error"},
	GetAmountTopUpWaitingPortingError:                  {-9017, "Get amount top up for waiting porting error"},
	GetReqRedeemFromLiquidationPoolStatusError:         {-9018, "Get redeem request form liquidation pool status error"},

	// relaying
	GetRelayingBNBHeaderByBlockHeightError: {-10001, "Get relaying bnb header by block height error"},
	GetRelayingBNBHeaderError:              {-10002, "Get relaying bnb header error"},
	GetBTCRelayingBestState:                {-10003, "Get BTC relaying best state error"},
	GetLatestBNBHeaderBlockHeightError:     {-10004, "Get latest bnb header block height error"},
	GetBTCBlockByHash:                      {-10005, "Get BTC block by hash error"},

	// feature reward
	GetRewardFeatureByFeatureNameError: {-11001, "Get feature reward by feature name error"},

	// InitStateRootHash
	InitStateRootHash: {-12000, "Init state root hash error"},

	// Restore
	RestoreBeaconCommittee:                        {-12001, "Restore beacon committee errror"},
	RestoreShardCommittee:                         {-12002, "Restore shard committee error"},
	RestoreBeaconPendingValidator:                 {-12003, "Restore beacon pending validator"},
	RestoreShardPendingValidator:                  {-12004, "Restore shard pending validator"},
	RestoreCandidateBeaconWaitingForCurrentRandom: {-12005, "Restore candidate beacon waiting for current random"},
	RestoreCandidateBeaconWaitingForNextRandom:    {-12006, "Restore candidate beacon waiting for next random"},
	RestoreCandidateShardWaitingForCurrentRandom:  {-12007, "Restore candidate shard waiting for current random"},
	RestoreCandidateShardWaitingForNextRandom:     {-12008, "Restore candidate shard waiting for next random"},
	GetAllBeaconViews:                             {-12009, "Get all beacon views"},
	GetTotalStakerError:                           {-12010, "Get total staker return error"},
}

// RPCError represents an error that is used as a part of a JSON-RPC JsonResponse
// object.
type RPCError struct {
	Code       int    `json:"Code,omitempty"`
	Message    string `json:"Message,omitempty"`
	StackTrace string `json:"StackTrace"`

	err error `json:"Err"`
}

func GetErrorCode(err int) int {
	return ErrCodeMessage[err].Code
}

// Guarantee RPCError satisifies the builtin error interface.
var _, _ error = RPCError{}, (*RPCError)(nil)

// Error returns a string describing the RPC error.  This satisifies the
// builtin error interface.
func (e RPCError) Error() string {
	return fmt.Sprintf("%d: %+v %+v", e.Code, e.err, e.StackTrace)
}

func (e RPCError) GetErr() error {
	return e.err
}

// NewRPCError constructs and returns a new JSON-RPC error that is suitable
// for use in a JSON-RPC JsonResponse object.
func NewRPCError(key int, err error, param ...interface{}) *RPCError {
	e := &RPCError{
		Code: ErrCodeMessage[key].Code,
		err:  errors.Wrap(err, ErrCodeMessage[key].Message),
	}
	if len(param) > 0 {
		e.Message = fmt.Sprintf(ErrCodeMessage[key].Message, param)
	} else {
		e.Message = ErrCodeMessage[key].Message
	}
	return e
}

// internalRPCError is a convenience function to convert an internal error to
// an RPC error with the appropriate Code set.  It also logs the error to the
// RPC server subsystem since internal errors really should not occur.  The
// context parameter is only used in the log Message and may be empty if it's
// not needed.
func InternalRPCError(errStr, context string) *RPCError {
	logStr := errStr
	if context != "" {
		logStr = context + ": " + errStr
	}
	Logger.log.Info(logStr)
	return NewRPCError(RPCInternalError, errors.New(errStr))
}
