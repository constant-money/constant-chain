package portalprocess

import (
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/portal"
	portalMeta "github.com/incognitochain/incognito-chain/portal/metadata"
)

type relayingProcessor interface {
	getActions() [][]string
	putAction(action []string)
	buildRelayingInst(
		bc bMeta.ChainRetriever,
		relayingHeaderAction portalMeta.RelayingHeaderAction,
		relayingState *RelayingHeaderChainState,
	) [][]string
	buildHeaderRelayingInst(
		senderAddressStr string,
		header string,
		blockHeight uint64,
		metaType int,
		shardID byte,
		txReqID common.Hash,
		status string,
	) []string
}

type portalInstructionProcessor interface {
	GetActions() map[byte][][]string
	PutAction(action []string, shardID byte)
	// get necessary db from stateDB to verify instructions when producing new block
	PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error)
	// validate and create new instructions in new beacon blocks
	BuildNewInsts(
		bc bMeta.ChainRetriever,
		contentStr string,
		shardID byte,
		currentPortalState *CurrentPortalState,
		beaconHeight uint64,
		shardHeights map[byte]uint64,
		portalParams portal.PortalParams,
		optionalData map[string]interface{},
	) ([][]string, error)
	// process instructions that confirmed in beacon blocks
	ProcessInsts(
		stateDB *statedb.StateDB,
		beaconHeight uint64,
		instructions []string,
		currentPortalState *CurrentPortalState,
		portalParams portal.PortalParams,
		updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
	) error
}

type portalInstProcessor struct {
	actions map[byte][][]string
}

type PortalManager struct {
	RelayingChains     map[int]relayingProcessor
	PortalInstructions map[int]portalInstructionProcessor
}

func NewPortalManager() *PortalManager {
	rbnbChain := &relayingBNBChain{
		relayingChain: &relayingChain{
			actions: [][]string{},
		},
	}
	rbtcChain := &relayingBTCChain{
		relayingChain: &relayingChain{
			actions: [][]string{},
		},
	}

	relayingChainProcessor := map[int]relayingProcessor{
		bMeta.RelayingBNBHeaderMeta: rbnbChain,
		bMeta.RelayingBTCHeaderMeta: rbtcChain,
	}

	portalInstProcessor := map[int]portalInstructionProcessor{
		bMeta.PortalExchangeRatesMeta: &portalExchangeRateProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalCustodianDepositMeta: &portalCustodianDepositProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalCustodianDepositMetaV3: &portalCustodianDepositProcessorV3{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalCustodianWithdrawRequestMeta: &portalRequestWithdrawCollateralProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalCustodianWithdrawRequestMetaV3: &portalRequestWithdrawCollateralProcessorV3{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalRequestPortingMetaV3: &portalPortingRequestProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalUserRequestPTokenMeta: &portalRequestPTokenProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalUserRequestPTokenMetaV4: &portalRequestPTokenProcessorV4{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},

		bMeta.PortalRedeemRequestMetaV3: &portalRedeemRequestProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},

		bMeta.PortalReqMatchingRedeemMeta: &portalRequestMatchingRedeemProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalRequestUnlockCollateralMetaV3: &portalRequestUnlockCollateralProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalRequestWithdrawRewardMeta: &portalReqWithdrawRewardProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalRedeemFromLiquidationPoolMeta: &portalRedeemFromLiquidationPoolProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalCustodianTopupMetaV2: &portalCustodianTopupProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalTopUpWaitingPortingRequestMeta: &portalTopupWaitingPortingReqProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalRedeemFromLiquidationPoolMetaV3: &portalRedeemFromLiquidationPoolProcessorV3{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalCustodianTopupMetaV3: &portalCustodianTopupProcessorV3{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalTopUpWaitingPortingRequestMetaV3: &portalTopupWaitingPortingReqProcessorV3{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalPickMoreCustodianForRedeemMeta: &portalPickMoreCustodianForRedeemProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalLiquidateCustodianMetaV3: &portalLiquidationCustodianRunAwayProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalExpiredWaitingPortingReqMeta: &portalExpiredWaitingPortingProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalLiquidateTPExchangeRatesMeta: &portalLiquidationByRatesProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalLiquidateByRatesMetaV3: &portalLiquidationByRatesV3Processor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
		bMeta.PortalUnlockOverRateCollateralsMeta: &portalCusUnlockOverRateCollateralsProcessor{
			portalInstProcessor: &portalInstProcessor{
				actions: map[byte][][]string{},
			},
		},
	}

	return &PortalManager{
		RelayingChains:     relayingChainProcessor,
		PortalInstructions: portalInstProcessor,
	}
}

func (pm PortalManager) GetPortalInstProcessorByMetaType(metaType int) portalInstructionProcessor {
	// PortalRequestPortingMeta and PortalRequestPortingMetaV3 use the same processor
	if metaType == bMeta.PortalRequestPortingMeta {
		metaType = bMeta.PortalRequestPortingMetaV3
	}

	// PortalRedeemRequestMeta and PortalRedeemRequestMetaV3 use the same processor
	if metaType == bMeta.PortalRedeemRequestMeta {
		metaType = bMeta.PortalRedeemRequestMetaV3
	}

	// PortalRequestUnlockCollateralMeta and PortalRequestUnlockCollateralMetaV3 use the same processor
	if metaType == bMeta.PortalRequestUnlockCollateralMeta {
		metaType = bMeta.PortalRequestUnlockCollateralMetaV3
	}

	// PortalLiquidateCustodianMeta and PortalLiquidateCustodianMetaV3 use the same processor
	if metaType == bMeta.PortalLiquidateCustodianMeta {
		metaType = bMeta.PortalLiquidateCustodianMetaV3
	}

	return pm.PortalInstructions[metaType]
}
