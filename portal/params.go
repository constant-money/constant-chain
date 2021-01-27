package portal

import (
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"github.com/incognitochain/incognito-chain/portal/portaltokens"
	"sort"
	"time"
)

type PortalCollateral struct {
	ExternalTokenID string
	Decimal         uint8
}

type RelayingParams struct {
	BNBRelayingHeaderChainID string
	BTCRelayingHeaderChainID string
	BTCDataFolderName        string
	BNBFullNodeProtocol      string
	BNBFullNodeHost          string
	BNBFullNodePort          string
}

type PortalParams struct {
	TimeOutCustodianReturnPubToken       time.Duration
	TimeOutWaitingPortingRequest         time.Duration
	TimeOutWaitingRedeemRequest          time.Duration
	MaxPercentLiquidatedCollateralAmount uint64
	MaxPercentCustodianRewards           uint64
	MinPercentCustodianRewards           uint64
	MinLockCollateralAmountInEpoch       uint64
	MinPercentLockedCollateral           uint64
	TP120                                uint64
	TP130                                uint64
	MinPercentPortingFee                 float64
	MinPercentRedeemFee                  float64
	SupportedCollateralTokens            []PortalCollateral
	MinPortalFee                         uint64 // nano PRV

	PortalTokens                 map[string]portaltokens.PortalTokenProcessor
	PortalFeederAddress          string
	PortalETHContractAddressStr  string // smart contract of ETH for portal
	MinUnlockOverRateCollaterals uint64

	RelayingParams RelayingParams
}

func GetLatestPortalParams(params map[uint64]PortalParams) PortalParams {
	if len(params) == 1 {
		return params[0]
	}

	bchs := []uint64{}
	for bch := range params {
		bchs = append(bchs, bch)
	}
	sort.Slice(bchs, func(i, j int) bool {
		return bchs[i] > bchs[j]
	})

	bchKey := bchs[0]
	return params[bchKey]
}

func (p PortalParams) GetSupportedCollateralTokenIDs() []string {
	tokenIDs := []string{}
	for _, col := range p.SupportedCollateralTokens {
		tokenIDs = append(tokenIDs, col.ExternalTokenID)
	}
	return tokenIDs
}

func (p PortalParams) IsSupportedTokenCollateralV3(externalTokenID string) bool {
	isSupported, _ := common.SliceExists(p.GetSupportedCollateralTokenIDs(), externalTokenID)
	return isSupported
}

func (p PortalParams) IsPortalToken(tokenIDStr string) bool {
	isExisted, _ := common.SliceExists(pCommon.PortalSupportedIncTokenIDs, tokenIDStr)
	return isExisted
}

func (p PortalParams) IsPortalExchangeRateToken(tokenIDStr string) bool {
	return p.IsPortalToken(tokenIDStr) || tokenIDStr == common.PRVIDStr || p.IsSupportedTokenCollateralV3(tokenIDStr)
}

func (p PortalParams) GetMinAmountPortalToken(tokenIDStr string) (uint64, error) {
	portalToken, ok := p.PortalTokens[tokenIDStr]
	if !ok {
		return 0, errors.New("TokenID is invalid")
	}
	return portalToken.GetMinTokenAmount(), nil
}
