package blockchain

import (
	"strconv"
	"strings"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/transaction"
)

func CreateGenesisBeaconBlock(
	version int,
	net uint16,
	genesisBlockTime string,
	genesisParams *GenesisParams,
) *types.BeaconBlock {
	inst := [][]string{}
	shardAutoStaking := []string{}
	beaconAutoStaking := []string{}
	txStakes := []string{}
	for i := 0; i < len(genesisParams.PreSelectShardNodeSerializedPubkey); i++ {
		shardAutoStaking = append(shardAutoStaking, "false")
		txStakes = append(txStakes, "d0e731f55fa6c49f602807a6686a7ac769de4e04882bb5eaf8f4fe209f46535d")
	}
	for i := 0; i < len(genesisParams.PreSelectBeaconNodeSerializedPubkey); i++ {
		beaconAutoStaking = append(beaconAutoStaking, "false")
		txStakes = append(txStakes, "d0e731f55fa6c49f602807a6686a7ac769de4e04882bb5eaf8f4fe209f46535d")
	}
	// build validator beacon
	// test generate public key in utility/generateKeys
	beaconAssignInstruction := []string{instruction.STAKE_ACTION}
	beaconAssignInstruction = append(beaconAssignInstruction, strings.Join(genesisParams.PreSelectBeaconNodeSerializedPubkey[:], ","))
	beaconAssignInstruction = append(beaconAssignInstruction, "beacon")
	beaconAssignInstruction = append(beaconAssignInstruction, strings.Join(txStakes[:], ","))
	beaconAssignInstruction = append(beaconAssignInstruction, strings.Join(genesisParams.PreSelectBeaconNodeSerializedPaymentAddress[:], ","))
	beaconAssignInstruction = append(beaconAssignInstruction, strings.Join(beaconAutoStaking[:], ","))

	shardAssignInstruction := []string{instruction.STAKE_ACTION}
	shardAssignInstruction = append(shardAssignInstruction, strings.Join(genesisParams.PreSelectShardNodeSerializedPubkey[:], ","))
	shardAssignInstruction = append(shardAssignInstruction, "shard")
	shardAssignInstruction = append(shardAssignInstruction, strings.Join(txStakes[:], ","))
	shardAssignInstruction = append(shardAssignInstruction, strings.Join(genesisParams.PreSelectShardNodeSerializedPaymentAddress[:], ","))
	shardAssignInstruction = append(shardAssignInstruction, strings.Join(shardAutoStaking[:], ","))

	inst = append(inst, beaconAssignInstruction)
	inst = append(inst, shardAssignInstruction)

	// init network param
	inst = append(inst, []string{instruction.SET_ACTION, "randomnumber", strconv.Itoa(int(0))})

	layout := "2006-01-02T15:04:05.000Z"
	str := genesisBlockTime
	genesisTime, err := time.Parse(layout, str)

	if err != nil {
		Logger.log.Error(err)
	}
	body := types.BeaconBody{ShardState: nil, Instructions: inst}
	header := types.BeaconHeader{
		Timestamp:                       genesisTime.Unix(),
		Version:                         version,
		Epoch:                           1,
		Height:                          1,
		Round:                           1,
		PreviousBlockHash:               common.Hash{},
		BeaconCommitteeAndValidatorRoot: common.Hash{},
		BeaconCandidateRoot:             common.Hash{},
		ShardCandidateRoot:              common.Hash{},
		ShardCommitteeAndValidatorRoot:  common.Hash{},
		ShardStateHash:                  common.Hash{},
		InstructionHash:                 common.Hash{},
	}

	block := &types.BeaconBlock{
		Body:   body,
		Header: header,
	}

	return block
}

func GetBeaconSwapInstructionKeyListV2(genesisParams *GenesisParams, epoch uint64) ([]string, []string) {
	newCommittees := genesisParams.SelectBeaconNodeSerializedPubkeyV2[epoch]
	newRewardReceivers := genesisParams.SelectBeaconNodeSerializedPaymentAddressV2[epoch]

	// TODO - in next replacement of committee validator key -> need read oldCommittees from prev-committee instead of from genesis block
	oldCommittees := genesisParams.PreSelectBeaconNodeSerializedPubkey
	// TODO

	beaconSwapInstructionKeyListV2 := []string{instruction.SWAP_ACTION, strings.Join(newCommittees, ","), strings.Join(oldCommittees, ","), "beacon", "", "", strings.Join(newRewardReceivers, ",")}
	return beaconSwapInstructionKeyListV2, newCommittees
}

func CreateGenesisShardBlock(
	version int,
	net uint16,
	genesisBlockTime string,
	icoParams *GenesisParams,
) *types.ShardBlock {
	body := types.ShardBody{}
	layout := "2006-01-02T15:04:05.000Z"
	str := genesisBlockTime
	genesisTime, err := time.Parse(layout, str)
	if err != nil {
		Logger.log.Error(err)
	}
	header := types.ShardHeader{
		Timestamp:         genesisTime.Unix(),
		Version:           version,
		BeaconHeight:      1,
		Epoch:             1,
		Round:             1,
		Height:            1,
		PreviousBlockHash: common.Hash{},
	}

	for _, tx := range icoParams.InitialIncognito {
		testSalaryTX := transaction.Tx{}
		testSalaryTX.UnmarshalJSON([]byte(tx))
		body.Transactions = append(body.Transactions, &testSalaryTX)
	}

	block := &types.ShardBlock{
		Body:   body,
		Header: header,
	}

	return block
}

func GetShardSwapInstructionKeyListV2(genesisParams *GenesisParams, epoch uint64, minCommitteeSize int, activeShard int) (map[byte][]string, map[byte][]string) {
	allShardSwapInstructionKeyListV2 := make(map[byte][]string)
	allShardNewKeyListV2 := make(map[byte][]string)
	selectShardNodeSerializedPubkeyV2 := genesisParams.SelectShardNodeSerializedPubkeyV2[epoch]
	selectShardNodeSerializedPaymentAddressV2 := genesisParams.SelectShardNodeSerializedPaymentAddressV2[epoch]
	preSelectShardNodeSerializedPubkey := genesisParams.PreSelectShardNodeSerializedPubkey
	shardCommitteeSize := minCommitteeSize
	for i := 0; i < activeShard; i++ {
		shardID := byte(i)
		newCommittees := selectShardNodeSerializedPubkeyV2[:shardCommitteeSize]

		// TODO - in next replacement of committee validator key -> need read oldCommittees from prev-committee instead of from genesis block
		oldCommittees := preSelectShardNodeSerializedPubkey[:shardCommitteeSize]
		// TODO

		newRewardReceiver := selectShardNodeSerializedPaymentAddressV2[:shardCommitteeSize]
		shardSwapInstructionKeyListV2 := []string{instruction.SWAP_ACTION, strings.Join(newCommittees, ","), strings.Join(oldCommittees, ","), "shard", strconv.Itoa(i), "", strings.Join(newRewardReceiver, ",")}
		allShardNewKeyListV2[shardID] = newCommittees
		selectShardNodeSerializedPubkeyV2 = selectShardNodeSerializedPubkeyV2[shardCommitteeSize:]
		preSelectShardNodeSerializedPubkey = preSelectShardNodeSerializedPubkey[shardCommitteeSize:]
		selectShardNodeSerializedPaymentAddressV2 = selectShardNodeSerializedPaymentAddressV2[shardCommitteeSize:]
		allShardSwapInstructionKeyListV2[shardID] = shardSwapInstructionKeyListV2
	}
	return allShardSwapInstructionKeyListV2, allShardNewKeyListV2
}
