package portalprocess

import (
	"sort"

	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/portalv4"
)

func CollectPortalV4Insts(pm *PortalV4Manager, metaType int, action []string, shardID byte) {
	switch metaType {
	// shield
	case basemeta.PortalShieldingRequestMeta:
		pm.PortalInstructions[basemeta.PortalShieldingRequestMeta].PutAction(action, shardID)
	// unshield
	case basemeta.PortalBurnPTokenMeta:
		pm.PortalInstructions[basemeta.PortalBurnPTokenMeta].PutAction(action, shardID)
	// replacement fee
	case basemeta.PortalReplacementFeeRequestMeta:
		pm.PortalInstructions[basemeta.PortalReplacementFeeRequestMeta].PutAction(action, shardID)
	// submit confirmed tx
	case basemeta.PortalSubmitConfirmedTxMeta:
		pm.PortalInstructions[basemeta.PortalSubmitConfirmedTxMeta].PutAction(action, shardID)

	default:
		return
	}
}

func buildNewPortalV4InstsFromActions(
	p portalInstructionProcessor,
	bc basemeta.ChainRetriever,
	stateDB *statedb.StateDB,
	currentPortalState *CurrentPortalV4State,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portalv4.PortalParams) ([][]string, error) {

	instructions := [][]string{}
	actions := p.GetActions()
	var shardIDKeys []int
	for k := range actions {
		shardIDKeys = append(shardIDKeys, int(k))
	}

	sort.Ints(shardIDKeys)
	for _, value := range shardIDKeys {
		shardID := byte(value)
		actions := actions[shardID]
		for _, action := range actions {
			contentStr := action[1]
			optionalData, err := p.PrepareDataForBlockProducer(stateDB, contentStr)
			if err != nil {
				Logger.log.Errorf("Error when preparing data before processing instruction %+v", err)
				continue
			}
			newInst, err := p.BuildNewInsts(
				bc,
				contentStr,
				shardID,
				currentPortalState,
				beaconHeight,
				shardHeights,
				portalParams,
				optionalData,
			)
			if err != nil {
				Logger.log.Errorf("Error when building new instructions : %v", err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	return instructions, nil
}

// handle portal instructions for block producer
func HandlePortalV4Insts(
	bc basemeta.ChainRetriever,
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	currentPortalState *CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	pm *PortalV4Manager,
) ([][]string, error) {
	instructions := [][]string{}

	// producer portal instructions for actions from shards
	// sort metadata type map to make it consistent for every run
	var metaTypes []int
	for metaType := range pm.PortalInstructions {
		metaTypes = append(metaTypes, metaType)
	}
	sort.Ints(metaTypes)
	for _, metaType := range metaTypes {
		actions := pm.PortalInstructions[metaType]
		newInst, err := buildNewPortalV4InstsFromActions(
			actions,
			bc,
			stateDB,
			currentPortalState,
			beaconHeight,
			shardHeights,
			portalParams)

		if err != nil {
			Logger.log.Error(err)
		}
		if len(newInst) > 0 {
			instructions = append(instructions, newInst...)
		}
	}

	return instructions, nil
}
