package metadata

import (
	"encoding/json"
	"strconv"
	"github.com/pkg/errors"
)

func CalculateSize(meta Metadata) uint64 {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return 0
	}
	return uint64(len(metaBytes))
}

func ParseMetadata(raw json.RawMessage) (Metadata, error) {
	var err error
	if raw == nil || string(raw)=="null" {
		return nil, nil
	}

	mtTemp := map[string]interface{}{}
	err = json.Unmarshal(raw, &mtTemp)
	if err != nil {
		return nil, err
	}
	var md Metadata
	typeFloat, ok := mtTemp["Type"].(float64)
	if !ok{
		return nil, errors.Errorf("Could not parse metadata with type: %v", mtTemp["Type"])
	}
	theType := int(typeFloat)
	switch theType {
	case IssuingRequestMeta:
		md = &IssuingRequest{}
	case IssuingResponseMeta:
		md = &IssuingResponse{}
	case ContractingRequestMeta:
		md = &ContractingRequest{}
	case IssuingETHRequestMeta:
		md = &IssuingETHRequest{}
	case IssuingETHResponseMeta:
		md = &IssuingETHResponse{}
	case BeaconSalaryResponseMeta:
		md = &BeaconBlockSalaryRes{}
	case BurningRequestMeta:
		md = &BurningRequest{}
	case BurningRequestMetaV2:
		md = &BurningRequest{}
	case ShardStakingMeta:
		md = &StakingMetadata{}
	case BeaconStakingMeta:
		md = &StakingMetadata{}
	case ReturnStakingMeta:
		md = &ReturnStakingMetadata{}
	case WithDrawRewardRequestMeta:
		md = &WithDrawRewardRequest{}
	case WithDrawRewardResponseMeta:
		md = &WithDrawRewardResponse{}
	case StopAutoStakingMeta:
		md = &StopAutoStakingMetadata{}
	case PDEContributionMeta:
		md = &PDEContribution{}
	case PDEPRVRequiredContributionRequestMeta:
		md = &PDEContribution{}
	case PDETradeRequestMeta:
		md = &PDETradeRequest{}
	case PDETradeResponseMeta:
		md = &PDETradeResponse{}
	case PDECrossPoolTradeRequestMeta:
		md = &PDECrossPoolTradeRequest{}
	case PDECrossPoolTradeResponseMeta:
		md = &PDECrossPoolTradeResponse{}
	case PDEWithdrawalRequestMeta:
		md = &PDEWithdrawalRequest{}
	case PDEWithdrawalResponseMeta:
		md = &PDEWithdrawalResponse{}
	case PDEFeeWithdrawalRequestMeta:
		md = &PDEFeeWithdrawalRequest{}
	case PDEFeeWithdrawalResponseMeta:
		md = &PDEFeeWithdrawalResponse{}
	case PDEContributionResponseMeta:
		md = &PDEContributionResponse{}
	case PortalCustodianDepositMeta:
		md = &PortalCustodianDeposit{}
	case PortalUserRegisterMeta:
		md = &PortalUserRegister{}
	case PortalUserRequestPTokenMeta:
		md = &PortalRequestPTokens{}
	case PortalCustodianDepositResponseMeta:
		md = &PortalCustodianDepositResponse{}
	case PortalUserRequestPTokenResponseMeta:
		md = &PortalRequestPTokensResponse{}
	case PortalRedeemRequestMeta:
		md = &PortalRedeemRequest{}
	case PortalRedeemRequestResponseMeta:
		md = &PortalRedeemRequestResponse{}
	case PortalRequestUnlockCollateralMeta:
		md = &PortalRequestUnlockCollateral{}
	case PortalExchangeRatesMeta:
		md = &PortalExchangeRates{}
	case RelayingBNBHeaderMeta:
		md = &RelayingHeader{}
	case RelayingBTCHeaderMeta:
		md = &RelayingHeader{}
	case PortalCustodianWithdrawRequestMeta:
		md = &PortalCustodianWithdrawRequest{}
	case PortalCustodianWithdrawResponseMeta:
		md = &PortalCustodianWithdrawResponse{}
	case PortalLiquidateCustodianMeta:
		md = &PortalLiquidateCustodian{}
	case PortalLiquidateCustodianResponseMeta:
		md = &PortalLiquidateCustodianResponse{}
	case PortalRequestWithdrawRewardMeta:
		md = &PortalRequestWithdrawReward{}
	case PortalRequestWithdrawRewardResponseMeta:
		md = &PortalWithdrawRewardResponse{}
	case PortalRedeemLiquidateExchangeRatesMeta:
		md = &PortalRedeemLiquidateExchangeRates{}
	case PortalRedeemLiquidateExchangeRatesResponseMeta:
		md = &PortalRedeemLiquidateExchangeRatesResponse{}
	case PortalLiquidationCustodianDepositMetaV2:
		md = &PortalLiquidationCustodianDepositV2{}
	case PortalLiquidationCustodianDepositResponseMetaV2:
		md = &PortalLiquidationCustodianDepositResponseV2{}
	case PortalLiquidationCustodianDepositMeta:
		md = &PortalLiquidationCustodianDeposit{}
	case PortalLiquidationCustodianDepositResponseMeta:
		md = &PortalLiquidationCustodianDepositResponse{}
	case BurningForDepositToSCRequestMeta:
		md = &BurningRequest{}
	case BurningForDepositToSCRequestMetaV2:
		md = &BurningRequest{}
	case PortalPortingResponseMeta:
		md = &PortalFeeRefundResponse{}
	case PortalReqMatchingRedeemMeta:
		md = &PortalReqMatchingRedeem{}
	case PortalTopUpWaitingPortingRequestMeta:
		md = &PortalTopUpWaitingPortingRequest{}
	case PortalTopUpWaitingPortingResponseMeta:
		md = &PortalTopUpWaitingPortingResponse{}
	default:
		return nil, errors.Errorf("Could not parse metadata with type: %d", theType)
	}

	err = json.Unmarshal(raw, &md)
	if err != nil {
		println(err.Error())
		return nil, err
	}
	return md, nil
}

var bridgeMetas = []string{
	strconv.Itoa(BeaconSwapConfirmMeta),
	strconv.Itoa(BridgeSwapConfirmMeta),
	strconv.Itoa(BurningConfirmMeta),
	strconv.Itoa(BurningConfirmForDepositToSCMeta),
}

func HasBridgeInstructions(instructions [][]string) bool {
	for _, inst := range instructions {
		for _, meta := range bridgeMetas {
			if len(inst) > 0 && inst[0] == meta {
				return true
			}
		}
	}
	return false
}

type uintMaybeString uint64
func (u uintMaybeString) MarshalJSON() ([]byte, error){
	return json.Marshal(u)
}
func (u *uintMaybeString) UnmarshalJSON(raw []byte) error{
	var theNum uint64
	err := json.Unmarshal(raw, &theNum)
	if err!=nil{
		var theStr string
		json.Unmarshal(raw, &theStr)
		temp, err := strconv.ParseUint(theStr, 10, 64)
		*u = uintMaybeString(temp)
		return err
	}
	*u = uintMaybeString(theNum)	
	return err
}

