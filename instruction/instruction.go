package instruction

import (
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/pkg/errors"
)

type Instruction interface {
	GetType() string
	ToString() []string
}

type ViewEnvironment struct {
	beaconCommittee                        []incognitokey.CommitteePublicKey
	beaconSubstitute                       []incognitokey.CommitteePublicKey
	candidateShardWaitingForCurrentRandom  []incognitokey.CommitteePublicKey
	candidateBeaconWaitingForCurrentRandom []incognitokey.CommitteePublicKey
	candidateShardWaitingForNextRandom     []incognitokey.CommitteePublicKey
	candidateBeaconWaitingForNextRandom    []incognitokey.CommitteePublicKey
	shardCommittee                         map[byte][]incognitokey.CommitteePublicKey
	shardSubstitute                        map[byte][]incognitokey.CommitteePublicKey
}

type CommitteeStateInstruction struct {
	SwapInstructions          []*SwapInstruction
	StakeInstructions         []*StakeInstruction
	AssignInstructions        []*AssignInstruction
	StopAutoStakeInstructions []*StopAutoStakeInstruction
}

func IsConsensusInstruction(action string) bool {
	return action == RANDOM_ACTION ||
		action == SWAP_ACTION ||
		action == STAKE_ACTION ||
		action == ASSIGN_ACTION ||
		action == STOP_AUTO_STAKE_ACTION ||
		action == SET_ACTION ||
		action == SWAP_SHARD_ACTION ||
		action == UNSTAKE_ACTION
}

// the order of instruction must always be maintain
func (i *CommitteeStateInstruction) ToString(action string) [][]string {
	instructions := [][]string{}
	switch action {
	case ASSIGN_ACTION:
		for _, assignInstruction := range i.AssignInstructions {
			instructions = append(instructions, assignInstruction.ToString())
		}
	case SWAP_ACTION:
		for _, swapInstruction := range i.SwapInstructions {
			instructions = append(instructions, swapInstruction.ToString())
		}
	case STAKE_ACTION:
		for _, stakeInstruction := range i.StakeInstructions {
			instructions = append(instructions, stakeInstruction.ToString())
		}
	case STOP_AUTO_STAKE_ACTION:
		for _, stopAutoStakeInstruction := range i.StopAutoStakeInstructions {
			instructions = append(instructions, stopAutoStakeInstruction.ToString())
		}
	}
	return [][]string{}
}

// FilterInstructions filter duplicate instruction
// duplicate instruction is result from delay of shard and beacon
func (i *CommitteeStateInstruction) ValidateAndFilterStakeInstructionsV1(v *ViewEnvironment) {
	panic("implement me")
}

func ValidateAndImportInstructionFromString(inst []string) (
	Instruction,
	error,
) {
	switch inst[0] {
	case STAKE_ACTION:
		stakeInstruction, err := ValidateAndImportStakeInstructionFromString(inst)
		if err != nil {
			return nil, errors.Errorf("SKIP stake instruction %+v, error %+v", inst, err)
		}
		return stakeInstruction, nil
	case SWAP_ACTION:
		swapInstruction, err := ValidateAndImportSwapInstructionFromString(inst)
		if err != nil {
			return nil, errors.Errorf("SKIP swap instruction %+v, error %+v", inst, err)
		}
		return swapInstruction, nil
	case STOP_AUTO_STAKE_ACTION:
		stopAutoStakeInstruction, err := ValidateAndImportStopAutoStakeInstructionFromString(inst)
		if err != nil {
			return nil, errors.Errorf("SKIP stop auto stake instruction %+v, error %+v", inst, err)
		}
		return stopAutoStakeInstruction, nil
	}
	return nil, nil
}
