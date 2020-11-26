package blockchain

import "github.com/incognitochain/incognito-chain/blockchain/committeestate"

//RestoreBeaconViewStateFromHash ...
func (beaconBestState *BeaconBestState) RestoreBeaconViewStateFromHash(blockchain *BlockChain) error {
	err := beaconBestState.InitStateRootHash(blockchain)
	if err != nil {
		return err
	}
	//best block
	block, _, err := blockchain.GetBeaconBlockByHash(beaconBestState.BestBlockHash)
	if err != nil || block == nil {
		return err
	}
	beaconBestState.BestBlock = *block
	beaconBestState.BeaconHeight = block.GetHeight()
	var beaconCommitteeEngine committeestate.BeaconCommitteeEngine
	if beaconBestState.BeaconHeight >= blockchain.config.ChainParams.ConsensusV3Epoch {
		beaconCommitteeEngine = InitBeaconCommitteeEngineV2(
			beaconBestState,
			blockchain.config.ChainParams,
			blockchain,
		)
	} else {
		beaconCommitteeEngine = InitBeaconCommitteeEngineV1(
			beaconBestState,
		)
	}
	beaconBestState.beaconCommitteeEngine = beaconCommitteeEngine
	return nil
}
