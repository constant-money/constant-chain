package rawdbv2

import (
	"fmt"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
)

// key prefix
var (
	// PDE
	WaitingPDEContributionPrefix = []byte("waitingpdecontribution-")
	PDEPoolPrefix                = []byte("pdepool-")
	PDESharePrefix               = []byte("pdeshare-")
	PDETradingFeePrefix          = []byte("pdetradingfee-")
	PDETradeFeePrefix            = []byte("pdetradefee-")
	PDEContributionStatusPrefix  = []byte("pdecontributionstatus-")
	PDETradeStatusPrefix         = []byte("pdetradestatus-")
	PDEWithdrawalStatusPrefix    = []byte("pdewithdrawalstatus-")
	PDEFeeWithdrawalStatusPrefix = []byte("pdefeewithdrawalstatus-")
)

// TODO - change json to CamelCase
type BridgeTokenInfo struct {
	TokenID         *common.Hash `json:"tokenId"`
	Amount          uint64       `json:"amount"`
	ExternalTokenID []byte       `json:"externalTokenId"`
	Network         string       `json:"network"`
	IsCentralized   bool         `json:"isCentralized"`
}

func NewBridgeTokenInfo(tokenID *common.Hash, amount uint64, externalTokenID []byte, network string, isCentralized bool) *BridgeTokenInfo {
	return &BridgeTokenInfo{TokenID: tokenID, Amount: amount, ExternalTokenID: externalTokenID, Network: network, IsCentralized: isCentralized}
}

type PDEContribution struct {
	ContributorAddressStr string
	TokenIDStr            string
	Amount                uint64
	TxReqID               common.Hash
}

func NewPDEContribution(contributorAddressStr string, tokenIDStr string, amount uint64, txReqID common.Hash) *PDEContribution {
	return &PDEContribution{ContributorAddressStr: contributorAddressStr, TokenIDStr: tokenIDStr, Amount: amount, TxReqID: txReqID}
}

type PDEPoolForPair struct {
	Token1IDStr     string
	Token1PoolValue uint64
	Token2IDStr     string
	Token2PoolValue uint64
}

func NewPDEPoolForPair(token1IDStr string, token1PoolValue uint64, token2IDStr string, token2PoolValue uint64) *PDEPoolForPair {
	return &PDEPoolForPair{Token1IDStr: token1IDStr, Token1PoolValue: token1PoolValue, Token2IDStr: token2IDStr, Token2PoolValue: token2PoolValue}
}

func BuildPDESharesKey(
	beaconHeight uint64,
	token1IDStr string,
	token2IDStr string,
	contributedTokenIDStr string,
	contributorAddressStr string,
) []byte {
	beaconHeightBytes := []byte(fmt.Sprintf("%d-", beaconHeight))
	pdeSharesByBCHeightPrefix := append(PDESharePrefix, beaconHeightBytes...)
	tokenIDStrs := []string{token1IDStr, token2IDStr}
	sort.Strings(tokenIDStrs)
	return append(pdeSharesByBCHeightPrefix, []byte(tokenIDStrs[0]+"-"+tokenIDStrs[1]+"-"+contributedTokenIDStr+"-"+contributorAddressStr)...)
}

func BuildPDESharesKeyV2(
	beaconHeight uint64,
	token1IDStr string,
	token2IDStr string,
	contributorAddressStr string,
) []byte {
	beaconHeightBytes := []byte(fmt.Sprintf("%d-", beaconHeight))
	pdeSharesByBCHeightPrefix := append(PDESharePrefix, beaconHeightBytes...)
	tokenIDStrs := []string{token1IDStr, token2IDStr}
	sort.Strings(tokenIDStrs)
	return append(pdeSharesByBCHeightPrefix, []byte(tokenIDStrs[0]+"-"+tokenIDStrs[1]+"-"+contributorAddressStr)...)
}

func BuildPDEPoolForPairKey(
	beaconHeight uint64,
	token1IDStr string,
	token2IDStr string,
) []byte {
	beaconHeightBytes := []byte(fmt.Sprintf("%d-", beaconHeight))
	pdePoolForPairByBCHeightPrefix := append(PDEPoolPrefix, beaconHeightBytes...)
	tokenIDStrs := []string{token1IDStr, token2IDStr}
	sort.Strings(tokenIDStrs)
	return append(pdePoolForPairByBCHeightPrefix, []byte(tokenIDStrs[0]+"-"+tokenIDStrs[1])...)
}

func BuildPDETradingFeeKey(
	beaconHeight uint64,
	token1IDStr string,
	token2IDStr string,
	contributorAddressStr string,
) []byte {
	beaconHeightBytes := []byte(fmt.Sprintf("%d-", beaconHeight))
	pdeTradingFeeByBCHeightPrefix := append(PDETradingFeePrefix, beaconHeightBytes...)
	tokenIDStrs := []string{token1IDStr, token2IDStr}
	sort.Strings(tokenIDStrs)
	return append(pdeTradingFeeByBCHeightPrefix, []byte(tokenIDStrs[0]+"-"+tokenIDStrs[1]+"-"+contributorAddressStr)...)
}

func BuildPDETradeFeesKey(
	beaconHeight uint64,
	token1IDStr string,
	token2IDStr string,
	tokenForFeeIDStr string,
) []byte {
	beaconHeightBytes := []byte(fmt.Sprintf("%d-", beaconHeight))
	pdeTradeFeesByBCHeightPrefix := append(PDETradeFeePrefix, beaconHeightBytes...)
	tokenIDStrs := []string{token1IDStr, token2IDStr}
	sort.Strings(tokenIDStrs)
	return append(pdeTradeFeesByBCHeightPrefix, []byte(tokenIDStrs[0]+"-"+tokenIDStrs[1]+"-"+tokenForFeeIDStr)...)
}

func BuildWaitingPDEContributionKey(
	beaconHeight uint64,
	pairID string,
) []byte {
	beaconHeightBytes := []byte(fmt.Sprintf("%d-", beaconHeight))
	waitingPDEContribByBCHeightPrefix := append(WaitingPDEContributionPrefix, beaconHeightBytes...)
	return append(waitingPDEContribByBCHeightPrefix, []byte(pairID)...)
}
