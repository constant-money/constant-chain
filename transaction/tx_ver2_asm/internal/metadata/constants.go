package metadata

// import "github.com/incognitochain/incognito-chain/common"

const (
	InvalidMeta = 1

	IssuingRequestMeta     = 24
	IssuingResponseMeta    = 25
	ContractingRequestMeta = 26
	BurningRequestMeta     = 27
	IssuingETHRequestMeta  = 80
	IssuingETHResponseMeta = 81

	ShardBlockReward             = 36
	AcceptedBlockRewardInfoMeta  = 37
	ShardBlockSalaryResponseMeta = 38
	BeaconRewardRequestMeta      = 39
	BeaconSalaryResponseMeta     = 40
	ReturnStakingMeta            = 41
	IncDAORewardRequestMeta      = 42
	ShardBlockRewardRequestMeta  = 43
	WithDrawRewardRequestMeta    = 44
	WithDrawRewardResponseMeta   = 45

	//statking
	ShardStakingMeta    = 63
	StopAutoStakingMeta = 127
	BeaconStakingMeta   = 64

	// Incognito -> Ethereum bridge
	BeaconSwapConfirmMeta = 70
	BridgeSwapConfirmMeta = 71
	BurningConfirmMeta    = 72

	// pde
	PDEContributionMeta         = 90
	PDETradeRequestMeta         = 91
	PDETradeResponseMeta        = 92
	PDEWithdrawalRequestMeta    = 93
	PDEWithdrawalResponseMeta   = 94
	PDEContributionResponseMeta = 95

	// portal
	PortalCustodianDepositMeta                      = 100
	PortalUserRegisterMeta                          = 101
	PortalUserRequestPTokenMeta                     = 102
	PortalCustodianDepositResponseMeta              = 103
	PortalUserRequestPTokenResponseMeta             = 104
	PortalExchangeRatesMeta                         = 105
	PortalRedeemRequestMeta                         = 106
	PortalRedeemRequestResponseMeta                 = 107
	PortalRequestUnlockCollateralMeta               = 108
	PortalCustodianWithdrawRequestMeta              = 110
	PortalCustodianWithdrawResponseMeta             = 111
	PortalLiquidateCustodianMeta                    = 112
	PortalLiquidateCustodianResponseMeta            = 113
	PortalLiquidateTPExchangeRatesMeta              = 114
	PortalExpiredWaitingPortingReqMeta              = 116
	PortalRewardMeta                                = 117
	PortalRequestWithdrawRewardMeta                 = 118
	PortalRequestWithdrawRewardResponseMeta         = 119
	PortalRedeemLiquidateExchangeRatesMeta          = 120
	PortalRedeemLiquidateExchangeRatesResponseMeta  = 121
	PortalLiquidationCustodianDepositMeta           = 122
	PortalLiquidationCustodianDepositResponseMeta   = 123
	PortalTotalRewardCustodianMeta                  = 124
	PortalPortingResponseMeta                       = 125
	PortalReqMatchingRedeemMeta                     = 126
	PortalPickMoreCustodianForRedeemMeta            = 128
	PortalLiquidationCustodianDepositMetaV2         = 129
	PortalLiquidationCustodianDepositResponseMetaV2 = 130

	//Note: don't use this metadata type for others
	PortalResetPortalDBMeta = 199

	// relaying
	RelayingBNBHeaderMeta = 200
	RelayingBTCHeaderMeta = 201

	PortalTopUpWaitingPortingRequestMeta  = 202
	PortalTopUpWaitingPortingResponseMeta = 203

	// incognito mode for smart contract
	BurningForDepositToSCRequestMeta = 96
	BurningConfirmForDepositToSCMeta = 97
)

var minerCreatedMetaTypes = []int{
	ShardBlockReward,
	BeaconSalaryResponseMeta,
	IssuingResponseMeta,
	IssuingETHResponseMeta,
	ReturnStakingMeta,
	WithDrawRewardResponseMeta,
	PDETradeResponseMeta,
	PDEWithdrawalResponseMeta,
	PDEContributionResponseMeta,
	PortalUserRequestPTokenResponseMeta,
	PortalCustodianDepositResponseMeta,
	PortalRedeemRequestResponseMeta,
	PortalCustodianWithdrawResponseMeta,
	PortalLiquidateCustodianResponseMeta,
	PortalRequestWithdrawRewardResponseMeta,
	PortalRedeemLiquidateExchangeRatesResponseMeta,
	PortalLiquidationCustodianDepositResponseMeta,
	PortalLiquidationCustodianDepositResponseMetaV2,
	PortalPortingResponseMeta,
	PortalTopUpWaitingPortingResponseMeta,
}

// Special rules for shardID: stored as 2nd param of instruction of BeaconBlock
const (
	AllShards  = -1
	BeaconOnly = -2
)
//const (
//	EthereumLightNodeProtocol = "http"
//	EthereumLightNodePort     = "8545"
//)
const (
	StopAutoStakingAmount = 0
	ETHConfirmationBlocks = 15
)

var AcceptedWithdrawRewardRequestVersion = []int{0, 1}
