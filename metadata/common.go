package metadata

import (
	"encoding/json"
	"fmt"
	ec "github.com/ethereum/go-ethereum/common"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"strconv"

	"github.com/pkg/errors"
)

func calculateSize(meta Metadata) uint64 {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return 0
	}
	return uint64(len(metaBytes))
}

func ParseMetadata(meta interface{}) (Metadata, error) {

	if meta == nil {
		return nil, nil
	}

	mtTemp := map[string]interface{}{}
	metaInBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(metaInBytes, &mtTemp)
	if err != nil {
		return nil, err
	}

	var md Metadata
	typeFloat,ok := mtTemp["Type"].(float64)
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
	case PortalRequestPortingMeta, PortalRequestPortingMetaV3:
		md = &PortalUserRegister{}
	case PortalUserRequestPTokenMeta:
		md = &PortalRequestPTokens{}
	case PortalCustodianDepositResponseMeta:
		md = &PortalCustodianDepositResponse{}
	case PortalUserRequestPTokenResponseMeta:
		md = &PortalRequestPTokensResponse{}
	case PortalRedeemRequestMeta, PortalRedeemRequestMetaV3:
		md = &PortalRedeemRequest{}
	case PortalRedeemRequestResponseMeta:
		md = &PortalRedeemRequestResponse{}
	case PortalRequestUnlockCollateralMeta, PortalRequestUnlockCollateralMetaV3:
		md = &PortalRequestUnlockCollateral{}
	case PortalExchangeRatesMeta:
		md = &PortalExchangeRates{}
	case PortalUnlockOverRateCollateralsMeta:
		md = &PortalUnlockOverRateCollaterals{}
	case RelayingBNBHeaderMeta:
		md = &RelayingHeader{}
	case RelayingBTCHeaderMeta:
		md = &RelayingHeader{}
	case PortalCustodianWithdrawRequestMeta:
		md = &PortalCustodianWithdrawRequest{}
	case PortalCustodianWithdrawResponseMeta:
		md = &PortalCustodianWithdrawResponse{}
	case PortalLiquidateCustodianMeta, PortalLiquidateCustodianMetaV3:
		md = &PortalLiquidateCustodian{}
	case PortalLiquidateCustodianResponseMeta:
		md = &PortalLiquidateCustodianResponse{}
	case PortalRequestWithdrawRewardMeta:
		md = &PortalRequestWithdrawReward{}
	case PortalRequestWithdrawRewardResponseMeta:
		md = &PortalWithdrawRewardResponse{}
	case PortalRedeemFromLiquidationPoolMeta:
		md = &PortalRedeemLiquidateExchangeRates{}
	case PortalRedeemFromLiquidationPoolResponseMeta:
		md = &PortalRedeemLiquidateExchangeRatesResponse{}
	case PortalCustodianTopupMetaV2:
		md = &PortalLiquidationCustodianDepositV2{}
	case PortalCustodianTopupResponseMetaV2:
		md = &PortalLiquidationCustodianDepositResponseV2{}
	case PortalCustodianTopupMeta:
		md = &PortalLiquidationCustodianDeposit{}
	case PortalCustodianTopupResponseMeta:
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
	case PortalCustodianDepositMetaV3:
		md = &PortalCustodianDepositV3{}
	case PortalCustodianWithdrawRequestMetaV3:
		md = &PortalCustodianWithdrawRequestV3{}
	case PortalRedeemFromLiquidationPoolMetaV3:
		md = &PortalRedeemFromLiquidationPoolV3{}
	case PortalRedeemFromLiquidationPoolResponseMetaV3:
		md = &PortalRedeemFromLiquidationPoolResponseV3{}
	case PortalCustodianTopupMetaV3:
		md = &PortalLiquidationCustodianDepositV3{}
	case PortalTopUpWaitingPortingRequestMetaV3:
		md = &PortalTopUpWaitingPortingRequestV3{}
	case ConvertingRequestMeta:
		md = &ConvertingRequest{}
	case ConvertingResponseMeta:
		md = &ConvertingResponse{}
	default:
		Logger.log.Debug("[db] parse meta err: %+v\n", meta)
		return nil, errors.Errorf("Could not parse metadata with type: %d", theType)
	}

	err = json.Unmarshal(metaInBytes, &md)
	if err != nil {
		return nil, err
	}
	
	return md, nil
}

var bridgeMetas = []string{
	strconv.Itoa(BeaconSwapConfirmMeta),
	strconv.Itoa(BridgeSwapConfirmMeta),
	strconv.Itoa(BurningConfirmMeta),
	strconv.Itoa(BurningConfirmForDepositToSCMeta),
	strconv.Itoa(BurningConfirmMetaV2),
	strconv.Itoa(BurningConfirmForDepositToSCMetaV2),
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

// TODO: add more meta data types
var portalMetas = []string{
	strconv.Itoa(PortalCustodianWithdrawConfirmMetaV3),
	strconv.Itoa(PortalRedeemFromLiquidationPoolConfirmMetaV3),
	strconv.Itoa(PortalLiquidateRunAwayCustodianConfirmMetaV3),
}

func HasPortalInstructions(instructions [][]string) bool {
	for _, inst := range instructions {
		for _, meta := range portalMetas {
			if len(inst) > 0 && inst[0] == meta {
				return true
			}
		}
	}
	return false
}

// Validate portal external addresses for collateral tokens (ETH/ERC20)
func ValidatePortalExternalAddress(chainName string, tokenID string, address string) (bool, error) {
	switch chainName {
	case common.ETHChainName:
		return ec.IsHexAddress(address), nil
	}
	return true, nil
}

// Validate portal remote addresses for portal tokens (BTC, BNB)
func ValidatePortalRemoteAddresses(remoteAddresses map[string]string, chainRetriever ChainRetriever) (bool, error) {
	if len(remoteAddresses) == 0 {
		return false, errors.New("remote addresses should be at least one address")
	}
	for tokenID, remoteAddr := range remoteAddresses {
		if !IsPortalToken(tokenID) {
			return false, errors.New("TokenID in remote address is invalid")
		}
		if len(remoteAddr) == 0 {
			return false, errors.New("Remote address is invalid")
		}
		if !IsValidPortalRemoteAddress(chainRetriever, remoteAddr, tokenID) {
			return false, fmt.Errorf("Remote address %v is not a valid address of tokenID %v", remoteAddr, tokenID)
		}
	}

	return true, nil
}

//Checks if a string payment address is supported by the underlying transaction.
//
//TODO: try another approach since the function itself is too complicated.
func AssertPaymentAddressAndTxVersion(paymentAddress interface{}, version int8) (privacy.PaymentAddress, error) {
	var addr privacy.PaymentAddress
	var ok bool
	//try to parse the payment address
	if addr, ok = paymentAddress.(privacy.PaymentAddress); !ok {
		//try the pointer
		if tmpAddr, ok := paymentAddress.(*privacy.PaymentAddress); !ok {
			//try the string one
			addrStr, ok := paymentAddress.(string)
			if !ok {
				return privacy.PaymentAddress{}, errors.New(fmt.Sprintf("cannot parse payment address - %v: Not a payment address or string address (txversion %v)", paymentAddress, version))
			}
			keyWallet, err := wallet.Base58CheckDeserialize(addrStr)
			if err != nil {
				return privacy.PaymentAddress{}, err
			}
			addr = keyWallet.KeySet.PaymentAddress
		} else {
			addr = *tmpAddr
		}
	}

	//Always check public spend and public view keys
	if addr.GetPublicSpend() == nil || addr.GetPublicView() == nil {
		return privacy.PaymentAddress{}, errors.New("PublicSpend or PublicView not found")
	}

	//If tx is in version 2, PublicOTAKey must not be nil
	if version == 2 {
		if addr.GetOTAPublicKey() == nil {
			return privacy.PaymentAddress{}, errors.New("PublicOTAKey not found")
		}
	}

	return addr, nil
}
