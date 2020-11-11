package statedb

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	ErrInvalidByteArrayType                   = "invalid byte array type"
	ErrInvalidHashType                        = "invalid hash type"
	ErrInvalidBigIntType                      = "invalid big int type"
	ErrInvalidCommitteeStateType              = "invalid committee state type"
	ErrInvalidStakerInfoType                  = "invalid staker info type"
	ErrInvalidPaymentAddressType              = "invalid payment address type"
	ErrInvalidIncognitoPublicKeyType          = "invalid incognito public key type"
	ErrInvalidCommitteeRewardStateType        = "invalid reward receiver state type "
	ErrInvalidRewardRequestStateType          = "invalid reward request state type"
	ErrInvalidBlackListProducerStateType      = "invalid black list producer state type"
	ErrInvalidSerialNumberStateType           = "invalid serial number state type"
	ErrInvalidCommitmentStateType             = "invalid commitment state type"
	ErrInvalidSNDerivatorStateType            = "invalid snderivator state type"
	ErrInvalidOutputCoinStateType             = "invalid output coin state type"
	ErrInvalidTokenStateType                  = "invalid token state type"
	ErrInvalidWaitingPDEContributionStateType = "invalid waiting pde contribution state type"
	ErrInvalidPDEPoolPairStateType            = "invalid pde pool pair state type"
	ErrInvalidPDEShareStateType               = "invalid pde shard state type"
	ErrInvalidPDEStatusStateType              = "invalid pde status state type"
	ErrInvalidBridgeEthTxStateType            = "invalid bridge eth tx state type"
	ErrInvalidBridgeTokenInfoStateType        = "invalid bridge token info state type"
	ErrInvalidBridgeStatusStateType           = "invalid bridge status state type"
	ErrInvalidBurningConfirmStateType         = "invalid burning confirm state type"
	ErrInvalidTokenTransactionStateType       = "invalid token transaction state type"
	//A
	ErrInvalidFinalExchangeRatesStateType  = "invalid final exchange rates state type"
	ErrInvalidLiquidationExchangeRatesType = "invalid liquidation exchange rates type"
	ErrInvalidWaitingPortingRequestType    = "invalid waiting porting request type"
	//B
	ErrInvalidPortalStatusStateType           = "invalid portal status state type"
	ErrInvalidPortalCustodianStateType        = "invalid portal custodian state type"
	ErrInvalidPortalWaitingRedeemRequestType  = "invalid portal waiting redeem request type"
	ErrInvalidPortalRewardInfoStateType       = "invalid portal reward info state type"
	ErrInvalidPortalLockedCollateralStateType = "invalid portal locked collateral state type"
	ErrInvalidRewardFeatureStateType          = "invalid feature reward state type"
	ErrInvalidPDETradingFeeStateType          = "invalid pde trading fee state type"
	ErrInvalidBlockHashType                   = "invalid block hash type"
)
const (
	InvalidByteArrayTypeError = iota
	InvalidHashTypeError
	InvalidBigIntTypeError
	InvalidCommitteeStateTypeError
	InvalidPaymentAddressTypeError
	InvalidIncognitoPublicKeyTypeError
	InvalidCommitteeRewardStateTypeError
	InvalidRewardRequestStateTypeError
	InvalidBlackListProducerStateTypeError
	InvalidSerialNumberStateTypeError
	InvalidCommitmentStateTypeError
	InvalidSNDerivatorStateTypeError
	InvalidOutputCoinStateTypeError
	// general error
	MethodNotSupportError
	// transaction related error
	StoreSerialNumberError
	GetSerialNumberError
	StoreCommitmentError
	GetCommitmentError
	StoreCommitmentIndexError
	GetCommitmentIndexError
	StoreCommitmentLengthError
	GetCommitmentLengthError
	StoreOutputCoinError
	GetOutputCoinError
	StoreSNDerivatorError
	GetSNDerivatorError
	StorePrivacyTokenError
	StorePrivacyTokenTransactionError
	GetPrivacyTokenError
	GetPrivacyTokenTxsError
	PrivacyTokenIDExistedError
	// Consensus Related Error
	StoreBlockHashError
	GetBlockHashError
	StoreBeaconCommitteeError
	GetBeaconCommitteeError
	StoreShardCommitteeError
	GetShardCommitteeError
	StoreAllShardCommitteeError
	StoreNextEpochCandidateError
	StoreOneShardSubstitutesValidatorError
	StoreBeaconSubstitutesValidatorError
	StoreCurrentEpochCandidateError
	StoreRewardRequestError
	GetRewardRequestError
	StoreCommitteeRewardError
	GetCommitteeRewardError
	ListCommitteeRewardError
	RemoveCommitteeRewardError
	StoreBlackListProducersError

	DeleteBeaconCommitteeError
	DeleteOneShardCommitteeError
	DeleteAllShardCommitteeError
	DeleteNextEpochShardCandidateError
	DeleteCurrentEpochShardCandidateError
	DeleteNextEpochBeaconCandidateError
	DeleteCurrentEpochBeaconCandidateError
	DeleteAllShardSubstitutesValidatorError
	DeleteBeaconSubstituteValidatorError
	// pdex error
	StoreWaitingPDEContributionError
	StorePDEPoolPairError
	StorePDEShareError
	GetPDEPoolForPairError
	TrackPDEStatusError
	GetPDEStatusError
	// bridge error
	BridgeInsertETHTxHashIssuedError
	IsETHTxHashIssuedError
	IsBridgeTokenExistedByTypeError
	CanProcessCIncTokenError
	CanProcessTokenPairError
	UpdateBridgeTokenInfoError
	GetAllBridgeTokensError
	TrackBridgeReqWithStatusError
	GetBridgeReqWithStatusError
	// burning confirm
	StoreBurningConfirmError
	GetBurningConfirmError

	//portal
	StoreCustodianDepositStateError
	StoreCustodianStateError
	StoreWaitingRedeemRequestError
	StoreRedeemRequestStateError
	TrackCustodianDepositError
	TrackReqPTokenError
	GetItemPortalByKeyError
	GetItemPortalByKeyNotFound
	GetCustodianDepositStatusError
	StoreReqPTokenStatusError
	GetReqPTokenStatusError
	StoreRedeemRequestError
	TrackRedeemReqByTxReqIDError
	TrackReqUnlockCollateralByTxReqIDError
	GetReqUnlockCollateralStatusError
	TrackLiquidateCustodianError
	StorePortalRewardError
	StorePortalStatusError
	StorePortalTxStatusError
	GetPortalStatusError
	GetPortalStatusNotFoundError
	GetPortalRedeemRequestStatusError
	StorePortalRedeemRequestStatusError
	StorePortalCustodianDepositStatusError
	GetPortalCustodianDepositStatusError
	StorePortalRequestPTokenStatusError
	GetPortalRequestPTokenStatusError
	GetPortalRedeemRequestByTxIDStatusError
	StorePortalRedeemRequestByTxIDStatusError
	GetPortalRequestUnlockCollateralStatusError
	StorePortalRequestUnlockCollateralStatusError
	GetPortalLiquidationCustodianRunAwayStatusError
	StorePortalLiquidationCustodianRunAwayStatusError
	GetPortalExpiredPortingReqStatusError
	StorePortalExpiredPortingReqStatusError
	GetPortalRequestWithdrawRewardStatusError
	StorePortalRequestWithdrawRewardStatusError
	StoreLockedCollateralStateError
	GetLockedCollateralStateError
	StorePortalReqMatchingRedeemByTxIDStatusError
	GetPortalReqMatchingRedeemByTxIDStatusError
	GetPortalTopupWaitingPortingStatusError
	GetPortalRedeemRequestFromLiquidationByTxIDStatusError

	//porting request
	GetPortingRequestTxStatusError
	GetPortingRequestStatusError
	StorePortingRequestStateError
	StoreWaitingPortingRequestError
	//exchange rates
	GetPortalFinalExchangeRatesStateError
	StorePortalExchangeRatesStatusError
	StoreExchangeRatesRequestStateError
	StoreFinalExchangeRatesStateError

	//liquidation exchange rates
	GetPortalLiquidationExchangeRatesPoolError
	GetLiquidationTopPercentileExchangeRatesStatusError
	StoreLiquidateTopPercentileExchangeRatesError
	StoreLiquidateExchangeRatesPoolError
	//liquidation custodian deposit
	StoreLiquidationCustodianDepositError
	//liquidation user redeem
	StoreRedeemLiquidationExchangeRatesError

	//custodian withdraw
	StorePortalCustodianWithdrawRequestStatusError
	GetPortalCustodianWithdrawStatusError

	// feature rewards
	StoreRewardFeatureError
	GetRewardFeatureError
	GetAllRewardFeatureError
	ResetAllFeatureRewardByTokenIDError
	GetRewardFeatureAmountByTokenIDError

	// PDEX v2
	StorePDETradingFeeError

	InvalidStakerInfoTypeError
	StoreMemberCommonShardPoolError
	StoreMemberShardPoolError
	StoreMemberCommonBeaconPoolError
	StoreMemberBeaconPoolError
	DeleteMemberCommonShardPoolError
	DeleteMemberShardPoolError
	DeleteMemberCommonBeaconPoolError
	DeleteMemberBeaconPoolError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	message string
}{
	MethodNotSupportError: {-1, "Method Not Support"},
	// -1xxx reposistory level
	InvalidByteArrayTypeError:              {-1000, "invalid byte array type"},
	InvalidHashTypeError:                   {-1001, "invalid hash type"},
	InvalidBigIntTypeError:                 {-1002, "invalid big int type"},
	InvalidCommitteeStateTypeError:         {-1003, "invalid committee state type"},
	InvalidPaymentAddressTypeError:         {-1004, "invalid payment address type"},
	InvalidIncognitoPublicKeyTypeError:     {-1005, "invalid incognito public key type"},
	InvalidCommitteeRewardStateTypeError:   {-1006, "invalid reward receiver state type "},
	InvalidRewardRequestStateTypeError:     {-1007, "invalid reward request state type"},
	InvalidBlackListProducerStateTypeError: {-1008, "invalid black list producer state type"},
	InvalidSerialNumberStateTypeError:      {-1009, "invalid serial number state type"},
	InvalidCommitmentStateTypeError:        {-1010, "invalid commitment state type"},
	InvalidSNDerivatorStateTypeError:       {-1011, "invalid snderivator state type"},
	InvalidOutputCoinStateTypeError:        {-1011, "invalid output coin state type"},
	// -2xxx transaction error
	StoreSerialNumberError:            {-2000, "Store Serial Number Error"},
	GetSerialNumberError:              {-2001, "Get Serial Number Error"},
	StoreCommitmentError:              {-2002, "Store Commitment Error"},
	GetCommitmentError:                {-2003, "Get Commitment Error"},
	StoreCommitmentIndexError:         {-2004, "Store Commitment Index Error"},
	GetCommitmentIndexError:           {-2005, "Get Commitment Index Error"},
	StoreCommitmentLengthError:        {-2006, "Store Commitment Length Error"},
	GetCommitmentLengthError:          {-2007, "Get Commitment Length Error"},
	StoreOutputCoinError:              {-2008, "Store Output Coin Error"},
	GetOutputCoinError:                {-2009, "Get Output Coin Error"},
	StoreSNDerivatorError:             {-2010, "Store SNDeriavator Error"},
	GetSNDerivatorError:               {-2011, "Get SNDeriavator Error"},
	StorePrivacyTokenError:            {-2012, "Store Privacy Token Error"},
	StorePrivacyTokenTransactionError: {-2013, "Store Privacy Token Transaction Error"},
	GetPrivacyTokenError:              {-2014, "Get Privacy Token Error"},
	GetPrivacyTokenTxsError:           {-2015, "Get Privacy Token Txs Error"},
	PrivacyTokenIDExistedError:        {-2016, "Privacy Token ID Existed Error"},
	// -3xxx: consensus error
	StoreBeaconCommitteeError:              {-3000, "Store Beacon Committee Error"},
	GetBeaconCommitteeError:                {-3001, "Get Beacon Committee Error"},
	StoreShardCommitteeError:               {-3002, "Store Shard Committee Error"},
	GetShardCommitteeError:                 {-3003, "Get Shard Committee Error"},
	StoreAllShardCommitteeError:            {-3004, "Store All Shard Committee Error"},
	StoreNextEpochCandidateError:           {-3005, "Store Next Epoch Candidate Error"},
	StoreCurrentEpochCandidateError:        {-3006, "Store Next Current Candidate Error"},
	StoreRewardRequestError:                {-3007, "Store Reward Request Error"},
	GetRewardRequestError:                  {-3008, "Get Reward Request Error"},
	StoreCommitteeRewardError:              {-3009, "Store Committee Reward Error"},
	GetCommitteeRewardError:                {-3010, "Get Committee Reward Error"},
	ListCommitteeRewardError:               {-3011, "List Committee Reward Error"},
	RemoveCommitteeRewardError:             {-3012, "Remove Committee Reward Error"},
	StoreBlackListProducersError:           {-3013, "Store Black List Producers Error"},
	StoreOneShardSubstitutesValidatorError: {-3014, "Store One Shard Substitutes Validator Error"},
	StoreBeaconSubstitutesValidatorError:   {-3014, "Store Beacon Substitutes Validator Error"},
	// -4xxx: pdex error
	StoreWaitingPDEContributionError: {-4000, "Store Waiting PDEX Contribution Error"},
	StorePDEPoolPairError:            {-4001, "Store PDEX Pool Pair Error"},
	StorePDEShareError:               {-4002, "Store PDEX Share Error"},
	GetPDEPoolForPairError:           {-4003, "Get PDEX Pool Pair Error"},
	TrackPDEStatusError:              {-4004, "Track PDEX Status Error"},
	GetPDEStatusError:                {-4005, "Get PDEX Status Error"},
	// -5xxx: bridge error
	BridgeInsertETHTxHashIssuedError: {-5000, "Bridge Insert ETH Tx Hash Issued Error"},
	IsETHTxHashIssuedError:           {-5001, "Is ETH Tx Hash Issued Error"},
	IsBridgeTokenExistedByTypeError:  {-5002, "Is Bridge Token Existed By Type Error"},
	CanProcessCIncTokenError:         {-5003, "Can Process Centralized Inc Token Error"},
	CanProcessTokenPairError:         {-5004, "Can Process Token Pair Error"},
	UpdateBridgeTokenInfoError:       {-5005, "Update Bridge Token Info Error"},
	GetAllBridgeTokensError:          {-5006, "Get All Bridge Tokens Error"},
	TrackBridgeReqWithStatusError:    {-5007, "Track Bridge Request With Status Error"},
	GetBridgeReqWithStatusError:      {-5008, "Get Bridge Request With Status Error"},
	// -6xxx: burning confirm
	StoreBurningConfirmError: {-6000, "Store Burning Confirm Error"},
	GetBurningConfirmError:   {-6001, "Get Burning Confirm Error"},

	//portal
	StoreWaitingPortingRequestError:                        {-14001, "Store waiting porting requests error"},
	StoreWaitingRedeemRequestError:                         {-14002, "Store waiting redeem requests error"},
	StoreRedeemRequestStateError:                           {-14003, "Store redeem request error"},
	StoreFinalExchangeRatesStateError:                      {-14004, "Store final exchange rates request error"},
	StorePortalCustodianWithdrawRequestStatusError:         {-14005, "Store portal custodian withdraw request status error"},
	StorePortalRewardError:                                 {-14006, "Store portal reward error"},
	StoreLiquidateTopPercentileExchangeRatesError:          {-14007, "Store liquidate top percentile exchange rates error"},
	StoreLiquidateExchangeRatesPoolError:                   {-14008, "Store liquidate exchange rates pool error"},
	StoreRedeemLiquidationExchangeRatesError:               {-14009, "Store redeem liquidation exchange rates error"},
	StoreLiquidationCustodianDepositError:                  {-14010, "Store liquidation custodian deposit error"},
	StoreCustodianStateError:                               {-14011, "Store custodian state error"},
	StorePortalStatusError:                                 {-14012, "Store portal status error"},
	GetPortalStatusError:                                   {-14013, "Get portal status error"},
	GetPortalRedeemRequestStatusError:                      {-14014, "Get portal redeem request status error"},
	StorePortalRedeemRequestStatusError:                    {-14015, "Store portal redeem request status error"},
	GetPortalCustodianDepositStatusError:                   {-14016, "Get portal custodian deposit status error"},
	StorePortalCustodianDepositStatusError:                 {-14017, "Store portal custodian deposit status error"},
	StorePortalRequestPTokenStatusError:                    {-14018, "Store portal request ptoken status error"},
	GetPortalRequestPTokenStatusError:                      {-14019, "Get portal request ptoken status error"},
	GetPortalRedeemRequestByTxIDStatusError:                {-14020, "Get portal redeem request by txid status error"},
	StorePortalRedeemRequestByTxIDStatusError:              {-14021, "Store portal redeem request by txid status error"},
	GetPortalRequestUnlockCollateralStatusError:            {-14022, "Get portal request unlock collateral status error"},
	StorePortalRequestUnlockCollateralStatusError:          {-14023, "Store portal request unlock collateral status error"},
	GetPortalLiquidationCustodianRunAwayStatusError:        {-14024, "Get portal liquidation custodian run away status error"},
	StorePortalLiquidationCustodianRunAwayStatusError:      {-14025, "Store portal liquidation custodian run away status error"},
	GetPortalExpiredPortingReqStatusError:                  {-14026, "Get portal expired porting request status error"},
	StorePortalExpiredPortingReqStatusError:                {-14027, "Store portal expired porting request status error"},
	GetPortalRequestWithdrawRewardStatusError:              {-14028, "Get portal request withdraw reward status error"},
	StorePortalRequestWithdrawRewardStatusError:            {-14029, "Store portal request withdraw reward status error"},
	StorePortalTxStatusError:                               {-14030, "Store portal Tx status error"},
	StorePortalExchangeRatesStatusError:                    {-14031, "Store portal exchange rates status error"},
	GetPortalLiquidationExchangeRatesPoolError:             {-14032, "Get portal liquidation exchange rates pool error"},
	GetPortingRequestStatusError:                           {-14033, "Get portal porting request status error"},
	GetPortingRequestTxStatusError:                         {-14034, "Get portal porting request tx status error"},
	GetLiquidationTopPercentileExchangeRatesStatusError:    {-14035, "Get liquidation tp ex change rates status error"},
	GetPortalStatusNotFoundError:                           {-14036, "Get portal status not found error"},
	GetPortalCustodianWithdrawStatusError:                  {-14037, "Get portal custodian withdraw status error"},
	StoreLockedCollateralStateError:                        {-14038, "Store locked collateral state error"},
	GetLockedCollateralStateError:                          {-14039, "Get locked collateral state error"},
	StorePortalReqMatchingRedeemByTxIDStatusError:          {-14040, "Store req matching redeem request error"},
	GetPortalReqMatchingRedeemByTxIDStatusError:            {-14041, "Get req matching redeem request error"},
	GetPortalTopupWaitingPortingStatusError:                {-14042, "Get custodian top up for waiting porting error"},
	GetPortalRedeemRequestFromLiquidationByTxIDStatusError: {-14043, "Get portal redeem req from liquidation pool status error"},

	StoreRewardFeatureError:              {-15000, "Store reward feature state error"},
	GetRewardFeatureError:                {-15001, "Get reward feature state error"},
	GetAllRewardFeatureError:             {-15002, "Get all reward feature state error"},
	ResetAllFeatureRewardByTokenIDError:  {-15003, "Reset all reward feature state by tokenID error"},
	GetRewardFeatureAmountByTokenIDError: {-15004, "Get reward feature amount by tokenID error"},
	InvalidStakerInfoTypeError:           {-15005, "Staker info invalid"},
}

type StatedbError struct {
	err     error
	Code    int
	Message string
}

func (e StatedbError) GetErrorCode() int {
	return e.Code
}
func (e StatedbError) GetError() error {
	return e.err
}
func (e StatedbError) GetMessage() string {
	return e.Message
}

func (e StatedbError) Error() string {
	return fmt.Sprintf("%d: %+v", e.Code, e.err)
}

func NewStatedbError(key int, err error, params ...interface{}) *StatedbError {
	return &StatedbError{
		err:     errors.Wrap(err, ErrCodeMessage[key].message),
		Code:    ErrCodeMessage[key].Code,
		Message: fmt.Sprintf(ErrCodeMessage[key].message, params),
	}
}
