package statedb

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

var (
	blockHashByIndexPrefix             = []byte("block-hash-by-index-")
	committeePrefix                    = []byte("shard-com-")
	substitutePrefix                   = []byte("shard-sub-")
	nextShardCandidatePrefix           = []byte("next-sha-cand-")
	currentShardCandidatePrefix        = []byte("cur-sha-cand-")
	nextBeaconCandidatePrefix          = []byte("next-bea-cand-")
	currentBeaconCandidatePrefix       = []byte("cur-bea-cand-")
	committeeRewardPrefix              = []byte("committee-reward-")
	rewardRequestPrefix                = []byte("reward-request-")
	blackListProducerPrefix            = []byte("black-list-")
	serialNumberPrefix                 = []byte("serial-number-")
	commitmentPrefix                   = []byte("com-value-")
	commitmentIndexPrefix              = []byte("com-index-")
	commitmentLengthPrefix             = []byte("com-length-")
	snDerivatorPrefix                  = []byte("sn-derivator-")
	outputCoinPrefix                   = []byte("output-coin-")
	tokenPrefix                        = []byte("token-")
	tokenTransactionPrefix             = []byte("token-transaction-")
	waitingPDEContributionPrefix       = []byte("waitingpdecontribution-")
	pdePoolPrefix                      = []byte("pdepool-")
	pdeSharePrefix                     = []byte("pdeshare-")
	pdeTradingFeePrefix                = []byte("pdetradingfee-")
	pdeTradeFeePrefix                  = []byte("pdetradefee-")
	pdeContributionStatusPrefix        = []byte("pdecontributionstatus-")
	pdeTradeStatusPrefix               = []byte("pdetradestatus-")
	pdeWithdrawalStatusPrefix          = []byte("pdewithdrawalstatus-")
	pdeStatusPrefix                    = []byte("pdestatus-")
	bridgeEthTxPrefix                  = []byte("bri-eth-tx-")
	bridgeCentralizedTokenInfoPrefix   = []byte("bri-cen-token-info-")
	bridgeDecentralizedTokenInfoPrefix = []byte("bri-de-token-info-")
	bridgeStatusPrefix                 = []byte("bri-status-")
	burnPrefix                         = []byte("burn-")
	stakerInfoPrefix                   = common.HashB([]byte("stk-info-"))[:prefixHashKeyLength]

	// portal
	portalFinaExchangeRatesStatePrefix                   = []byte("portalfinalexchangeratesstate-")
	portalExchangeRatesRequestStatusPrefix               = []byte("portalexchangeratesrequeststatus-")
	portalPortingRequestStatusPrefix                     = []byte("portalportingrequeststatus-")
	portalPortingRequestTxStatusPrefix                   = []byte("portalportingrequesttxstatus-")
	portalCustodianWithdrawStatusPrefix                  = []byte("portalcustodianwithdrawstatus-")
	portalCustodianWithdrawStatusPrefixV3                = []byte("portalcustodianwithdrawstatusv3-")
	portalLiquidationTpExchangeRatesStatusPrefix         = []byte("portalliquidationtpexchangeratesstatus-")
	portalLiquidationTpExchangeRatesStatusPrefixV3       = []byte("portalliquidationbyratesstatusv3-")
	portalLiquidationExchangeRatesPoolPrefix             = []byte("portalliquidationexchangeratespool-")
	portalLiquidationCustodianDepositStatusPrefix        = []byte("portalliquidationcustodiandepositstatus-")
	portalLiquidationCustodianDepositStatusPrefixV3      = []byte("portalliquidationcustodiandepositstatusv3-")
	portalTopUpWaitingPortingStatusPrefix                = []byte("portaltopupwaitingportingstatus-")
	portalTopUpWaitingPortingStatusPrefixV3              = []byte("portaltopupwaitingportingstatusv3-")
	portalLiquidationRedeemRequestStatusPrefix           = []byte("portalliquidationredeemrequeststatus-")
	portalLiquidationRedeemRequestStatusPrefixV3         = []byte("portalliquidationredeemrequeststatusv3-")
	portalWaitingPortingRequestPrefix                    = []byte("portalwaitingportingrequest-")
	portalCustodianStatePrefix                           = []byte("portalcustodian-")
	portalWaitingRedeemRequestsPrefix                    = []byte("portalwaitingredeemrequest-")
	portalMatchedRedeemRequestsPrefix                    = []byte("portalmatchedredeemrequest-")
	portalUnlockOverRateCollateralsRequestStatusPrefix   = []byte("portalunlockoverratecollateralsstatus-")
	portalUnlockOverRateCollateralsRequestTxStatusPrefix = []byte("portalunlockoverratecollateralstxstatus-")
	portalWaitingUnshieldRequestsPrefix                  = []byte("portalwaitingunshieldrequest-")
	portalUnshieldRequestsProcessedPrefix                = []byte("portalunshieldrequestsprocessed-")
	portalUTXOStatePrefix                                = []byte("portalutxo-")
	portalShieldRequestPrefix                            = []byte("portalshieldrequest-")

	portalStatusPrefix                           = []byte("portalstatus-")
	portalCustodianDepositStatusPrefix           = []byte("custodiandeposit-")
	portalCustodianDepositStatusPrefixV3         = []byte("custodiandepositv3-")
	portalRequestPTokenStatusPrefix              = []byte("requestptoken-")
	portalRedeemRequestStatusPrefix              = []byte("redeemrequest-")
	portalRedeemRequestStatusByTxReqIDPrefix     = []byte("redeemrequestbytxid-")
	portalRequestUnlockCollateralStatusPrefix    = []byte("requestunlockcollateral-")
	portalRequestWithdrawRewardStatusPrefix      = []byte("requestwithdrawportalreward-")
	portalReqMatchingRedeemStatusByTxReqIDPrefix = []byte("reqmatchredeembytxid-")

	// liquidation for portal
	portalLiquidateCustodianRunAwayPrefix = []byte("portalliquidaterunaway-")
	portalExpiredPortingReqPrefix         = []byte("portalexpiredportingreq-")

	// reward for portal
	portalRewardInfoStatePrefix       = []byte("portalreward-")
	portalLockedCollateralStatePrefix = []byte("portallockedcollateral-")

	// reward for features in network (such as portal, pdex, etc)
	rewardFeatureStatePrefix = []byte("rewardfeaturestate-")
	// feature names
	PortalRewardName = "portal"

	portalExternalTxPrefix      = []byte("portalexttx-")
	portalConfirmProofPrefix    = []byte("portalproof-")
	withdrawCollateralProofType = []byte("0-")

	// portal v4
	portalUnshieldRequestStatusPrefix            = []byte("unshieldrequest-")
	portalBatchUnshieldRequestStatusPrefix            = []byte("batchunshieldrequest-")
	portalUnshielReplacementFeeBatchStatusPrefix = []byte("unshieldrequestbatchreplacementfeeprocessed-")
	portalUnshielSubmitConfirmedTxStatusPrefix   = []byte("unshieldrequestsubmitconfirmedtx-")
)

func GetCommitteePrefixWithRole(role int, shardID int) []byte {
	switch role {
	case NextEpochShardCandidate:
		temp := []byte(string(nextShardCandidatePrefix))
		h := common.HashH(temp)
		return h[:][:prefixHashKeyLength]
	case CurrentEpochShardCandidate:
		temp := []byte(string(currentShardCandidatePrefix))
		h := common.HashH(temp)
		return h[:][:prefixHashKeyLength]
	case NextEpochBeaconCandidate:
		temp := []byte(string(nextBeaconCandidatePrefix))
		h := common.HashH(temp)
		return h[:][:prefixHashKeyLength]
	case CurrentEpochBeaconCandidate:
		temp := []byte(string(currentBeaconCandidatePrefix))
		h := common.HashH(temp)
		return h[:][:prefixHashKeyLength]
	case SubstituteValidator:
		temp := []byte(string(substitutePrefix) + strconv.Itoa(shardID))
		h := common.HashH(temp)
		return h[:][:prefixHashKeyLength]
	case CurrentValidator:
		temp := []byte(string(committeePrefix) + strconv.Itoa(shardID))
		h := common.HashH(temp)
		return h[:][:prefixHashKeyLength]
	default:
		panic("role not exist: " + strconv.Itoa(role))
	}
}

func GetStakerInfoPrefix() []byte {
	h := common.HashH(stakerInfoPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetStakerInfoKey(stakerPublicKey []byte) common.Hash {
	h := common.HashH(stakerInfoPrefix)
	final := append(h[:][:prefixHashKeyLength], common.HashH(stakerPublicKey).Bytes()[:prefixKeyLength]...)
	finalHash, err := common.Hash{}.NewHash(final)
	if err != nil {
		panic("Create key fail1")
	}
	return *finalHash
}

func GetCommitteeRewardPrefix() []byte {
	h := common.HashH(committeeRewardPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetRewardRequestPrefix(epoch uint64) []byte {
	buf := common.Uint64ToBytes(epoch)
	temp := append(rewardRequestPrefix, buf...)
	h := common.HashH(temp)
	return h[:][:prefixHashKeyLength]
}

func GetBlackListProducerPrefix() []byte {
	h := common.HashH(blackListProducerPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetSerialNumberPrefix(tokenID common.Hash, shardID byte) []byte {
	h := common.HashH(append(serialNumberPrefix, append(tokenID[:], shardID)...))
	return h[:][:prefixHashKeyLength]
}

func GetCommitmentPrefix(tokenID common.Hash, shardID byte) []byte {
	h := common.HashH(append(commitmentPrefix, append(tokenID[:], shardID)...))
	return h[:][:prefixHashKeyLength]
}

func GetCommitmentIndexPrefix(tokenID common.Hash, shardID byte) []byte {
	h := common.HashH(append(commitmentIndexPrefix, append(tokenID[:], shardID)...))
	return h[:][:prefixHashKeyLength]
}

func GetCommitmentLengthPrefix() []byte {
	h := common.HashH(commitmentLengthPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetSNDerivatorPrefix(tokenID common.Hash) []byte {
	h := common.HashH(append(snDerivatorPrefix, tokenID[:]...))
	return h[:][:prefixHashKeyLength]
}

func GetOutputCoinPrefix(tokenID common.Hash, shardID byte, publicKey []byte) []byte {
	h := common.HashH(append(outputCoinPrefix, append(tokenID[:], append(publicKey, shardID)...)...))
	return h[:][:prefixHashKeyLength]
}

func GetTokenPrefix() []byte {
	h := common.HashH(tokenPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetTokenTransactionPrefix(tokenID common.Hash) []byte {
	h := common.HashH(append(tokenTransactionPrefix, tokenID[:]...))
	return h[:][:prefixHashKeyLength]
}

func GetWaitingPDEContributionPrefix() []byte {
	h := common.HashH(waitingPDEContributionPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPDEPoolPairPrefix() []byte {
	h := common.HashH(pdePoolPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPDESharePrefix() []byte {
	h := common.HashH(pdeSharePrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPDETradingFeePrefix() []byte {
	h := common.HashH(pdeTradingFeePrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPDEStatusPrefix() []byte {
	h := common.HashH(pdeStatusPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetBridgeEthTxPrefix() []byte {
	h := common.HashH(bridgeEthTxPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetBridgeTokenInfoPrefix(isCentralized bool) []byte {
	if isCentralized {
		h := common.HashH(bridgeCentralizedTokenInfoPrefix)
		return h[:][:prefixHashKeyLength]
	} else {
		h := common.HashH(bridgeDecentralizedTokenInfoPrefix)
		return h[:][:prefixHashKeyLength]
	}
}

func GetBridgeStatusPrefix() []byte {
	h := common.HashH(bridgeStatusPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetBurningConfirmPrefix() []byte {
	h := common.HashH(burnPrefix)
	return h[:][:prefixHashKeyLength]
}
func WaitingPDEContributionPrefix() []byte {
	return waitingPDEContributionPrefix
}
func PDEPoolPrefix() []byte {
	return pdePoolPrefix
}
func PDESharePrefix() []byte {
	return pdeSharePrefix
}
func PDETradeFeePrefix() []byte {
	return pdeTradeFeePrefix
}
func PDEContributionStatusPrefix() []byte {
	return pdeContributionStatusPrefix
}
func PDETradeStatusPrefix() []byte {
	return pdeTradeStatusPrefix
}
func PDEWithdrawalStatusPrefix() []byte {
	return pdeWithdrawalStatusPrefix
}

// GetWaitingPDEContributionKey: WaitingPDEContributionPrefix - beacon height - pairid
func GetWaitingPDEContributionKey(beaconHeight uint64, pairID string) []byte {
	prefix := append(waitingPDEContributionPrefix, []byte(fmt.Sprintf("%d-", beaconHeight))...)
	return append(prefix, []byte(pairID)...)
}

// GetPDEPoolForPairKey: PDEPoolPrefix - beacon height - token1ID - token2ID
func GetPDEPoolForPairKey(beaconHeight uint64, token1ID string, token2ID string) []byte {
	prefix := append(pdePoolPrefix, []byte(fmt.Sprintf("%d-", beaconHeight))...)
	tokenIDs := []string{token1ID, token2ID}
	sort.Strings(tokenIDs)
	return append(prefix, []byte(tokenIDs[0]+"-"+tokenIDs[1])...)
}

// GetPDEShareKey: PDESharePrefix + beacon height + token1ID + token2ID + contributor address
func GetPDEShareKey(beaconHeight uint64, token1ID string, token2ID string, contributorAddress string) []byte {
	prefix := append(pdeSharePrefix, []byte(fmt.Sprintf("%d-", beaconHeight))...)
	tokenIDs := []string{token1ID, token2ID}
	sort.Strings(tokenIDs)
	return append(prefix, []byte(tokenIDs[0]+"-"+tokenIDs[1]+"-"+contributorAddress)...)
}

// GetPDETradingFeeKey: PDETradingFeePrefix + beacon height + token1ID + token2ID
func GetPDETradingFeeKey(beaconHeight uint64, token1ID string, token2ID string, contributorAddress string) []byte {
	prefix := append(pdeTradingFeePrefix, []byte(fmt.Sprintf("%d-", beaconHeight))...)
	tokenIDs := []string{token1ID, token2ID}
	sort.Strings(tokenIDs)
	return append(prefix, []byte(tokenIDs[0]+"-"+tokenIDs[1]+"-"+contributorAddress)...)
}

func GetPDEStatusKey(prefix []byte, suffix []byte) []byte {
	return append(prefix, suffix...)
}

// Portal
func GetFinalExchangeRatesStatePrefix() []byte {
	h := common.HashH(portalFinaExchangeRatesStatePrefix)
	return h[:][:prefixHashKeyLength]
}

func PortalPortingRequestStatusPrefix() []byte {
	return portalPortingRequestStatusPrefix
}

func PortalPortingRequestTxStatusPrefix() []byte {
	return portalPortingRequestTxStatusPrefix
}

func PortalExchangeRatesRequestStatusPrefix() []byte {
	return portalExchangeRatesRequestStatusPrefix
}

func PortalCustodianWithdrawStatusPrefix() []byte {
	return portalCustodianWithdrawStatusPrefix
}

func PortalCustodianWithdrawStatusPrefixV3() []byte {
	return portalCustodianWithdrawStatusPrefixV3
}

func PortalLiquidationTpExchangeRatesStatusPrefix() []byte {
	return portalLiquidationTpExchangeRatesStatusPrefix
}

func PortalLiquidationTpExchangeRatesStatusPrefixV3() []byte {
	return portalLiquidationTpExchangeRatesStatusPrefixV3
}

func PortalLiquidationCustodianDepositStatusPrefix() []byte {
	return portalLiquidationCustodianDepositStatusPrefix
}
func PortalLiquidationCustodianDepositStatusPrefixV3() []byte {
	return portalLiquidationCustodianDepositStatusPrefixV3
}

func PortalTopUpWaitingPortingStatusPrefix() []byte {
	return portalTopUpWaitingPortingStatusPrefix
}

func PortalTopUpWaitingPortingStatusPrefixV3() []byte {
	return portalTopUpWaitingPortingStatusPrefixV3
}

func PortalLiquidationRedeemRequestStatusPrefix() []byte {
	return portalLiquidationRedeemRequestStatusPrefix
}
func PortalLiquidationRedeemRequestStatusPrefixV3() []byte {
	return portalLiquidationRedeemRequestStatusPrefixV3
}

func GetPortalWaitingPortingRequestPrefix() []byte {
	h := common.HashH(portalWaitingPortingRequestPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPortalLiquidationPoolPrefix() []byte {
	h := common.HashH(portalLiquidationExchangeRatesPoolPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPortalCustodianStatePrefix() []byte {
	h := common.HashH(portalCustodianStatePrefix)
	return h[:][:prefixHashKeyLength]
}

func GetWaitingRedeemRequestPrefix() []byte {
	h := common.HashH(portalWaitingRedeemRequestsPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetMatchedRedeemRequestPrefix() []byte {
	h := common.HashH(portalMatchedRedeemRequestsPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPortalRewardInfoStatePrefix(beaconHeight uint64) []byte {
	h := common.HashH(append(portalRewardInfoStatePrefix, []byte(fmt.Sprintf("%d-", beaconHeight))...))
	return h[:][:prefixHashKeyLength]
}

func GetPortalStatusPrefix() []byte {
	h := common.HashH(portalStatusPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetLockedCollateralStatePrefix() []byte {
	h := common.HashH(portalLockedCollateralStatePrefix)
	return h[:][:prefixHashKeyLength]
}

func GetRewardFeatureStatePrefix(epoch uint64) []byte {
	h := common.HashH(append(rewardFeatureStatePrefix, []byte(fmt.Sprintf("%d-", epoch))...))
	return h[:][:prefixHashKeyLength]
}

func GetPortalExternalTxPrefix() []byte {
	h := common.HashH(portalExternalTxPrefix)
	return h[:][:prefixHashKeyLength]
}

func GetPortalConfirmProofPrefixV3(proofType []byte) []byte {
	h := common.HashH(append(portalConfirmProofPrefix, proofType...))
	return h[:][:prefixHashKeyLength]
}

func GetPortalUnlockOverRateCollateralsPrefix() []byte {
	h := common.HashH(portalUnlockOverRateCollateralsRequestTxStatusPrefix)
	return h[:][:prefixHashKeyLength]
}

func PortalWithdrawCollateralProofType() []byte {
	return withdrawCollateralProofType
}

func PortalCustodianDepositStatusPrefix() []byte {
	return portalCustodianDepositStatusPrefix
}

func PortalCustodianDepositStatusPrefixV3() []byte {
	return portalCustodianDepositStatusPrefixV3
}

func PortalRequestPTokenStatusPrefix() []byte {
	return portalRequestPTokenStatusPrefix
}

func PortalShieldingRequestStatusPrefix() []byte {
	return portalShieldRequestPrefix
}

func PortalRedeemRequestStatusPrefix() []byte {
	return portalRedeemRequestStatusPrefix
}

func PortalRedeemRequestStatusByTxReqIDPrefix() []byte {
	return portalRedeemRequestStatusByTxReqIDPrefix
}

func PortalRequestUnlockCollateralStatusPrefix() []byte {
	return portalRequestUnlockCollateralStatusPrefix
}

func PortalRequestWithdrawRewardStatusPrefix() []byte {
	return portalRequestWithdrawRewardStatusPrefix
}

func PortalLiquidateCustodianRunAwayPrefix() []byte {
	return portalLiquidateCustodianRunAwayPrefix
}

func PortalExpiredPortingReqPrefix() []byte {
	return portalExpiredPortingReqPrefix
}

func PortalReqMatchingRedeemStatusByTxReqIDPrefix() []byte {
	return portalReqMatchingRedeemStatusByTxReqIDPrefix
}

func PortalUnlockOverRateCollateralsRequestStatusPrefix() []byte {
	return portalUnlockOverRateCollateralsRequestStatusPrefix
}

// ================================== Prefix Portal v4 ==================================
func GetWaitingUnshieldRequestPrefix(tokenID string) []byte {
	h := common.HashH(append(portalWaitingUnshieldRequestsPrefix, []byte(tokenID)...))
	return h[:][:prefixHashKeyLength]
}

func GetProcessedUnshieldRequestBatchPrefix(tokenID string) []byte {
	h := common.HashH(append(portalUnshieldRequestsProcessedPrefix, []byte(tokenID)...))
	return h[:][:prefixHashKeyLength]
}

func GetPortalUTXOStatePrefix(tokenID string) []byte {
	h := common.HashH(append(portalUTXOStatePrefix, []byte(tokenID)...))
	return h[:][:prefixHashKeyLength]
}

func GetShieldingRequestPrefix(tokenID string) []byte {
	h := common.HashH(append(portalShieldRequestPrefix, []byte(tokenID)...))
	return h[:][:prefixHashKeyLength]
}

func PortalUnshieldRequestStatusPrefix() []byte {
	return portalUnshieldRequestStatusPrefix
}

func PortalBatchUnshieldRequestStatusPrefix() []byte {
	return portalBatchUnshieldRequestStatusPrefix
}

func PortalUnshielReplacementFeeBatchStatusPrefix() []byte {
	return portalUnshielReplacementFeeBatchStatusPrefix
}

func PortalSubmitConfirmedTxStatusPrefix() []byte {
	return portalUnshielSubmitConfirmedTxStatusPrefix
}

var _ = func() (_ struct{}) {
	m := make(map[string]string)
	prefixs := [][]byte{}
	// Current validator
	for i := -1; i < 256; i++ {
		temp := GetCommitteePrefixWithRole(CurrentValidator, i)
		prefixs = append(prefixs, temp)
		if v, ok := m[string(temp)]; ok {
			panic("shard-com-" + strconv.Itoa(i) + " same prefix " + v)
		}
		m[string(temp)] = "shard-com-" + strconv.Itoa(i)
	}
	// Substitute validator
	for i := -1; i < 256; i++ {
		temp := GetCommitteePrefixWithRole(SubstituteValidator, i)
		prefixs = append(prefixs, temp)
		if v, ok := m[string(temp)]; ok {
			panic("shard-sub-" + strconv.Itoa(i) + " same prefix " + v)
		}
		m[string(temp)] = "shard-sub-" + strconv.Itoa(i)
	}
	// Current Candidate
	tempCurrentCandidate := GetCommitteePrefixWithRole(CurrentEpochShardCandidate, -2)
	prefixs = append(prefixs, tempCurrentCandidate)
	if v, ok := m[string(tempCurrentCandidate)]; ok {
		panic("cur-cand-" + " same prefix " + v)
	}
	m[string(tempCurrentCandidate)] = "cur-cand-"
	// Next candidate
	tempNextCandidate := GetCommitteePrefixWithRole(NextEpochShardCandidate, -2)
	prefixs = append(prefixs, tempNextCandidate)
	if v, ok := m[string(tempNextCandidate)]; ok {
		panic("next-cand-" + " same prefix " + v)
	}
	m[string(tempNextCandidate)] = "next-cand-"
	// reward receiver
	tempRewardReceiver := GetCommitteeRewardPrefix()
	prefixs = append(prefixs, tempRewardReceiver)
	if v, ok := m[string(tempRewardReceiver)]; ok {
		panic("committee-reward-" + " same prefix " + v)
	}
	m[string(tempRewardReceiver)] = "committee-reward-"
	// black list producer
	tempBlackListProducer := GetBlackListProducerPrefix()
	prefixs = append(prefixs, tempBlackListProducer)
	if v, ok := m[string(tempBlackListProducer)]; ok {
		panic("black-list-" + " same prefix " + v)
	}
	m[string(tempBlackListProducer)] = "black-list-"
	for i, v1 := range prefixs {
		for j, v2 := range prefixs {
			if i == j {
				continue
			}
			if bytes.HasPrefix(v1, v2) || bytes.HasPrefix(v2, v1) {
				panic("(prefix: " + fmt.Sprintf("%+v", v1) + ", value: " + m[string(v1)] + ")" + " is prefix or being prefix of " + " (prefix: " + fmt.Sprintf("%+v", v1) + ", value: " + m[string(v2)] + ")")
			}
		}
	}
	return
}()
