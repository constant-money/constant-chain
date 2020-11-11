package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	types2 "github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
	"github.com/tendermint/tendermint/types"
	"strconv"
)

func (blockchain *BlockChain) processRelayingInstructions(block *types2.BeaconBlock) error {
	relayingState, err := blockchain.InitRelayingHeaderChainStateFromDB()
	if err != nil {
		Logger.log.Error(err)
		return nil
	}

	// because relaying instructions in received beacon block were sorted already as desired so dont need to do sorting again over here
	for _, inst := range block.Body.Instructions {
		if len(inst) < 4 {
			continue // Not error, just not relaying instruction
		}
		var err error
		switch inst[0] {
		//case strconv.Itoa(metadata.RelayingBNBHeaderMeta):
		//	err = blockchain.processRelayingBNBHeaderInst(inst, relayingState)
		case strconv.Itoa(metadata.RelayingBTCHeaderMeta):
			err = blockchain.processRelayingBTCHeaderInst(inst, relayingState)
		}
		if err != nil {
			Logger.log.Error(err)
		}
	}

	// store updated relayingState to leveldb with new beacon height
	//err = relayingState.BNBHeaderChain.StoreBNBChainState()
	//if err != nil {
	//	Logger.log.Error(err)
	//}
	return nil
}

func (blockchain *BlockChain) processRelayingBTCHeaderInst(
	instruction []string,
	relayingState *RelayingHeaderChainState,
) error {
	Logger.log.Info("[BTC Relaying] - Processing processRelayingBTCHeaderInst...")
	btcHeaderChain := relayingState.BTCHeaderChain
	if btcHeaderChain == nil {
		return errors.New("[processRelayingBTCHeaderInst] BTC Header chain instance should not be nil")
	}

	if len(instruction) != 4 {
		return nil // skip the instruction
	}

	var relayingHeaderContent metadata.RelayingHeaderContent
	err := json.Unmarshal([]byte(instruction[3]), &relayingHeaderContent)
	if err != nil {
		return err
	}

	headerBytes, err := base64.StdEncoding.DecodeString(relayingHeaderContent.Header)
	if err != nil {
		return err
	}
	var msgBlk *wire.MsgBlock
	err = json.Unmarshal(headerBytes, &msgBlk)
	if err != nil {
		return err
	}
	block := btcutil.NewBlock(msgBlk)
	isMainChain, isOrphan, err := btcHeaderChain.ProcessBlockV2(block, btcrelaying.BFNone)
	if err != nil {
		Logger.log.Errorf("ProcessBlock fail with error: %v", err)
		return err
	}
	Logger.log.Infof("ProcessBlock (%s) success with result: isMainChain: %v, isOrphan: %v", block.Hash(), isMainChain, isOrphan)
	return nil
}

func (blockchain *BlockChain) processRelayingBNBHeaderInst(
	instructions []string,
	relayingState *RelayingHeaderChainState,
) error {
	if relayingState == nil {
		Logger.log.Errorf("relaying block state is nil")
		return errors.New("relaying block state is nil")
	}
	if len(instructions) != 4 {
		return nil // skip the instruction
	}

	// unmarshal instructions content
	var actionData metadata.RelayingHeaderContent
	err := json.Unmarshal([]byte(instructions[3]), &actionData)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal bnb block meta data %v - %v\n", instructions[3], err)
		return err
	}

	var block types.Block
	blockBytes, err := base64.StdEncoding.DecodeString(actionData.Header)
	if err != nil {
		Logger.log.Errorf("Can not decode bnb block %v - %v\n", actionData.Header, err)
		return err
	}
	err = json.Unmarshal(blockBytes, &block)
	if err != nil {
		Logger.log.Errorf("Can not unmarshal bnb block %v - %v\n", string(blockBytes), err)
		return err
	}

	reqStatus := instructions[2]
	if reqStatus == common.RelayingHeaderConsideringChainStatus {
		err := relayingState.BNBHeaderChain.ProcessNewBlock(&block, blockchain.config.ChainParams.BNBRelayingHeaderChainID)
		if err != nil {
			Logger.log.Errorf("Error when process new block %v\n", err)
			return err
		}
	}

	return nil
}
