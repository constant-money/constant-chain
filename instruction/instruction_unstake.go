package instruction

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

//UnstakeInstruction : Hold and verify data for unstake action
type UnstakeInstruction struct {
	CommitteePublicKeys       []string
	CommitteePublicKeysStruct []incognitokey.CommitteePublicKey
}

//NewUnstakeInstructionWithValue : Constructor with value
func NewUnstakeInstructionWithValue(committeePublicKeys []string) *UnstakeInstruction {
	unstakeInstruction := &UnstakeInstruction{}
	unstakeInstruction.SetCommitteePublicKeys(committeePublicKeys)
	return unstakeInstruction
}

func (unstakeInstruction *UnstakeInstruction) SetCommitteePublicKeys(publicKeys []string) error {
	if publicKeys == nil {
		return fmt.Errorf("Public key is null")
	}
	unstakeInstruction.CommitteePublicKeys = publicKeys
	publicKeyStructs, err := incognitokey.CommitteeBase58KeyListToStruct(publicKeys)
	if err != nil {
		return err
	}
	unstakeInstruction.CommitteePublicKeysStruct = publicKeyStructs
	return nil
}

//NewUnstakeInstruction : Default constructor
func NewUnstakeInstruction() *UnstakeInstruction {
	return &UnstakeInstruction{}
}

//GetType : Get type of unstake instruction
func (unstakeIns *UnstakeInstruction) GetType() string {
	return UNSTAKE_ACTION
}

//ToString : Convert class to string
func (unstakeIns *UnstakeInstruction) ToString() []string {
	unstakeInstructionStr := []string{UNSTAKE_ACTION}
	unstakeInstructionStr = append(unstakeInstructionStr, strings.Join(unstakeIns.CommitteePublicKeys, SPLITTER))
	return unstakeInstructionStr
}

func (unstakeIns *UnstakeInstruction) IsEmpty() bool {
	return reflect.DeepEqual(unstakeIns, NewUnstakeInstruction()) ||
		len(unstakeIns.CommitteePublicKeysStruct) == 0 && len(unstakeIns.CommitteePublicKeys) == 0
}

//ValidateAndImportUnstakeInstructionFromString : Validate and import unstake instruction from string
func ValidateAndImportUnstakeInstructionFromString(instruction []string) (*UnstakeInstruction, error) {
	if err := ValidateUnstakeInstructionSanity(instruction); err != nil {
		return nil, err
	}
	return ImportUnstakeInstructionFromString(instruction), nil
}

//ImportUnstakeInstructionFromString : Import unstake instruction from string
func ImportUnstakeInstructionFromString(instruction []string) *UnstakeInstruction {
	unstakeInstruction := NewUnstakeInstruction()
	if len(instruction) > 1 {
		if len(instruction[1]) > 0 {
			committeePublicKeys := strings.Split(instruction[1], SPLITTER)
			unstakeInstruction.CommitteePublicKeys = committeePublicKeys
			unstakeInstruction.CommitteePublicKeysStruct, _ = incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
		}
	}
	return unstakeInstruction
}

//ValidateUnstakeInstructionSanity : Validate unstake instruction data type
func ValidateUnstakeInstructionSanity(instruction []string) error {
	if instruction == nil {
		return fmt.Errorf("Instruction is null")
	}
	if len(instruction) != 2 {
		return fmt.Errorf("invalid length, %+v", instruction)
	}
	if instruction[0] != UNSTAKE_ACTION {
		return fmt.Errorf("invalid stop auto stake action, %+v", instruction)
	}
	_, err := incognitokey.CommitteeBase58KeyListToStruct(strings.Split(instruction[1], SPLITTER))
	if err != nil {
		return err
	}
	return nil
}

//InsertIntoStateDB : insert unstake instruction to statedb
func (unstakeIns *UnstakeInstruction) InsertIntoStateDB(sDB *statedb.StateDB) error {
	return nil
}
