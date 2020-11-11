package blockchain

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/instruction"
	"sort"
	"strings"
)

func sortMapStringUint8Keys(m map[string]uint8) map[string]uint8 {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedMap := make(map[string]uint8)
	for _, k := range keys {
		sortedMap[k] = m[k]
	}
	return sortedMap
}

func (blockchain *BlockChain) buildBadProducersWithPunishment(
	isBeacon bool,
	shardID int,
	committee []string,
) map[string]uint8 {
	slashLevels := blockchain.config.ChainParams.SlashLevels
	if len(slashLevels) == 0 {
		return make(map[string]uint8)
	}
	numOfBlocksByProducers := map[string]uint64{}
	if isBeacon {
		if blockchain.GetBeaconBestState() == nil {
			numOfBlocksByProducers = make(map[string]uint64)
		} else {
			numOfBlocksByProducers = blockchain.GetBeaconBestState().NumOfBlocksByProducers
		}

	} else {
		if blockchain.GetBestStateShard(byte(shardID)) == nil {
			numOfBlocksByProducers = make(map[string]uint64)
		} else {
			numOfBlocksByProducers = blockchain.GetBestStateShard(byte(shardID)).NumOfBlocksByProducers
		}

	}
	// numBlkPerEpoch := blockchain.config.ChainParams.Epoch
	numBlkPerEpoch := uint64(0)
	for _, numBlk := range numOfBlocksByProducers {
		numBlkPerEpoch += numBlk
	}
	badProducersWithPunishment := make(map[string]uint8)
	committeeLen := len(committee)
	if committeeLen == 0 {
		return badProducersWithPunishment
	}
	expectedNumBlkByEachProducer := numBlkPerEpoch / uint64(committeeLen)

	if expectedNumBlkByEachProducer == 0 {
		return badProducersWithPunishment
	}
	for _, producer := range committee {
		numBlk, found := numOfBlocksByProducers[producer]
		if !found {
			numBlk = 0
		}
		if numBlk >= expectedNumBlkByEachProducer {
			continue
		}
		missingPercent := uint8((-(numBlk - expectedNumBlkByEachProducer) * 100) / expectedNumBlkByEachProducer)
		var selectedSlLev *SlashLevel
		for _, slLev := range slashLevels {
			if missingPercent >= slLev.MinRange {
				selectedSlLev = &slLev
			}
		}
		if selectedSlLev != nil {
			badProducersWithPunishment[producer] = selectedSlLev.PunishedEpoches
		}
	}
	return sortMapStringUint8Keys(badProducersWithPunishment)
}

func (blockchain *BlockChain) getUpdatedProducersBlackList(slashStateDB *statedb.StateDB, isBeacon bool, shardID int, committee []string, beaconHeight uint64) (map[string]uint8, error) {
	producersBlackList := statedb.GetProducersBlackList(slashStateDB, beaconHeight)
	if isBeacon {
		punishedProducersFinished := []string{}
		for producer, punishedEpoches := range producersBlackList {
			if punishedEpoches == 1 {
				punishedProducersFinished = append(punishedProducersFinished, producer)
			}
		}
		for _, producer := range punishedProducersFinished {
			delete(producersBlackList, producer)
		}
	}
	badProducersWithPunishment := blockchain.buildBadProducersWithPunishment(isBeacon, shardID, committee)
	for producer, punishedEpoches := range badProducersWithPunishment {
		epoches, found := producersBlackList[producer]
		if !found || epoches < punishedEpoches {
			producersBlackList[producer] = punishedEpoches
		}
	}
	return sortMapStringUint8Keys(producersBlackList), nil
}

func (blockchain *BlockChain) processForSlashing(slashStateDB *statedb.StateDB, beaconBlock *types.BeaconBlock) error {
	var err error
	punishedProducersFinished := []string{}
	fliterPunishedProducersFinished := []string{}
	beaconHeight := beaconBlock.GetHeight()
	producersBlackList := statedb.GetProducersBlackList(slashStateDB, beaconHeight-1)
	chainParamEpoch := blockchain.config.ChainParams.Epoch
	newBeaconHeight := beaconBlock.GetHeight()
	if newBeaconHeight%uint64(chainParamEpoch) == 0 { // end of epoch
		for producer := range producersBlackList {
			producersBlackList[producer]--
			if producersBlackList[producer] == 0 {
				punishedProducersFinished = append(punishedProducersFinished, producer)
			}
		}
		for _, producer := range punishedProducersFinished {
			delete(producersBlackList, producer)
		}
	}

	for _, inst := range beaconBlock.GetInstructions() {
		if len(inst) == 0 {
			continue
		}
		if inst[0] != instruction.SWAP_ACTION {
			continue
		}
		badProducersWithPunishmentBytes := []byte{}
		if len(inst) == 6 && inst[3] == "shard" {
			badProducersWithPunishmentBytes = []byte(inst[5])
		}
		if len(inst) == 5 && inst[3] == "beacon" {
			badProducersWithPunishmentBytes = []byte(inst[4])
		}
		if len(badProducersWithPunishmentBytes) == 0 {
			continue
		}

		var badProducersWithPunishment map[string]uint8
		err = json.Unmarshal(badProducersWithPunishmentBytes, &badProducersWithPunishment)
		if err != nil {
			return err
		}
		for producer, punishedEpoches := range badProducersWithPunishment {
			epoches, found := producersBlackList[producer]
			if !found || epoches < punishedEpoches {
				producersBlackList[producer] = punishedEpoches
			}
		}
	}
	for _, punishedProducerFinished := range punishedProducersFinished {
		flag := false
		for producerBlaskList, _ := range producersBlackList {
			if strings.Compare(producerBlaskList, punishedProducerFinished) == 0 {
				flag = true
				break
			}
		}
		if !flag {
			fliterPunishedProducersFinished = append(fliterPunishedProducersFinished, punishedProducerFinished)
		}
	}
	statedb.RemoveProducerBlackList(slashStateDB, fliterPunishedProducersFinished)
	err = statedb.StoreProducersBlackList(slashStateDB, beaconHeight, producersBlackList)
	return err
}
