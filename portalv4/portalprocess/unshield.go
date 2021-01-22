package portalprocess

import (
	"encoding/base64"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/portal"
)

/* =======
Portal Burn PToken Processor
======= */
type portalBurnPTokenRequestProcessor struct {
	*portalInstProcessor
}

func (p *portalBurnPTokenRequestProcessor) GetActions() map[byte][][]string {
	return p.actions
}

func (p *portalBurnPTokenRequestProcessor) PutAction(action []string, shardID byte) {
	_, found := p.actions[shardID]
	if !found {
		p.actions[shardID] = [][]string{action}
	} else {
		p.actions[shardID] = append(p.actions[shardID], action)
	}
}

func (p *portalBurnPTokenRequestProcessor) PrepareDataForBlockProducer(stateDB *statedb.StateDB, contentStr string) (map[string]interface{}, error) {
	return nil, nil
}

func (p *portalBurnPTokenRequestProcessor) BuildNewInsts(
	bc bMeta.ChainRetriever,
	contentStr string,
	shardID byte,
	currentPortalV4State *CurrentPortalV4State,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	portalParams portal.PortalParams,
	optionalData map[string]interface{},
) ([][]string, error) {
	_, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while decoding content string of portal porting request action: %+v", err)
		return [][]string{}, nil
	}

	return [][]string{nil}, nil
}

func (p *portalBurnPTokenRequestProcessor) ProcessInsts(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	instructions []string,
	currentPortalV4State *CurrentPortalV4State,
	portalParams portal.PortalParams,
	updatingInfoByTokenID map[common.Hash]bMeta.UpdatingInfo,
) error {

	if currentPortalV4State == nil {
		Logger.log.Errorf("current portal state is nil")
		return nil
	}

	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	return nil
}
