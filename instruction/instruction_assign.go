package instruction

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/incognitokey"
)

var (
	ErrAssignInstruction = errors.New("assign instruction error")
)

//AssignInstruction :
//Assign instruction format:
// ["assign action", publickeys, shard or beacon chain, shard_id]
type AssignInstruction struct {
	ChainID               int
	ShardCandidates       []string
	ShardCandidatesStruct []incognitokey.CommitteePublicKey
}

func NewAssignInstructionWithValue(chainID int, shardCandidates []string) *AssignInstruction {
	assignInstruction := &AssignInstruction{}
	assignInstruction.SetChainID(chainID)
	assignInstruction.SetShardCandidates(shardCandidates)
	return assignInstruction
}

func NewAssignInstruction() *AssignInstruction {
	return &AssignInstruction{}
}

func (a *AssignInstruction) GetType() string {
	return ASSIGN_ACTION
}

func (a *AssignInstruction) IsEmpty() bool {
	return reflect.DeepEqual(a, NewAssignInstruction()) ||
		len(a.ShardCandidates) == 0 && len(a.ShardCandidatesStruct) == 0
}

func (a *AssignInstruction) ToString() []string {
	assignInstructionStr := []string{ASSIGN_ACTION}
	assignInstructionStr = append(assignInstructionStr, strings.Join(a.ShardCandidates, SPLITTER))
	assignInstructionStr = append(assignInstructionStr, "shard")
	assignInstructionStr = append(assignInstructionStr, fmt.Sprintf("%v", a.ChainID))
	return assignInstructionStr
}

func (a *AssignInstruction) SetChainID(chainID int) *AssignInstruction {
	a.ChainID = chainID
	return a
}

func (a *AssignInstruction) SetShardCandidates(shardCandidates []string) *AssignInstruction {
	a.ShardCandidates = shardCandidates
	return a
}

func ValidateAndImportAssignInstructionFromString(instruction []string) (*AssignInstruction, error) {
	if err := ValidateAssignInstructionSanity(instruction); err != nil {
		return nil, err
	}
	return ImportAssignInstructionFromString(instruction)
}

// ImportAssignInstructionFromString is unsafe method
func ImportAssignInstructionFromString(instruction []string) (*AssignInstruction, error) {
	assignIntruction := NewAssignInstruction()
	tempShardID := instruction[3]
	chainID, _ := strconv.Atoi(tempShardID)
	assignIntruction.SetChainID(chainID)
	assignIntruction.SetShardCandidates(strings.Split(instruction[1], SPLITTER))

	shardPendingValidatorStruct, err := incognitokey.CommitteeBase58KeyListToStruct(assignIntruction.ShardCandidates)
	if err != nil {
		return nil, err
	}

	assignIntruction.ShardCandidatesStruct = shardPendingValidatorStruct

	return assignIntruction, err
}

//ValidateAssignInstructionSanity ...
func ValidateAssignInstructionSanity(instruction []string) error {
	if len(instruction) != 4 {
		return fmt.Errorf("%+v: invalid length, %+v", ErrAssignInstruction, instruction)
	}
	if instruction[0] != ASSIGN_ACTION {
		return fmt.Errorf("%+v: invalid assign action, %+v", ErrAssignInstruction, instruction)
	}
	if instruction[2] != SHARD_INST {
		return fmt.Errorf("%+v: invalid assign chain ID, %+v", ErrAssignInstruction, instruction)
	}
	if _, err := strconv.Atoi(instruction[3]); err != nil {
		return fmt.Errorf("%+v: invalid assign shard ID, err %+v, %+v", ErrAssignInstruction, err, instruction)
	}
	return nil
}
