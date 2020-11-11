package instruction

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/incognitochain/incognito-chain/incognitokey"
)

type StopAutoStakeInstruction struct {
	CommitteePublicKeys       []string
	CommitteePublicKeysStruct []incognitokey.CommitteePublicKey
}

func NewStopAutoStakeInstructionWithValue(publicKeys []string) *StopAutoStakeInstruction {
	res := &StopAutoStakeInstruction{}
	res.SetPublicKeys(publicKeys)
	return res
}

func NewStopAutoStakeInstruction() *StopAutoStakeInstruction {
	return &StopAutoStakeInstruction{}
}

func (s *StopAutoStakeInstruction) GetType() string {
	return STOP_AUTO_STAKE_ACTION
}

func (s *StopAutoStakeInstruction) IsEmpty() bool {
	return reflect.DeepEqual(s, NewStopAutoStakeInstruction()) || len(s.CommitteePublicKeys) == 0
}

func (s *StopAutoStakeInstruction) SetPublicKeys(publicKeys []string) (*StopAutoStakeInstruction, error) {
	s.CommitteePublicKeys = publicKeys
	publicKeyStructs, err := incognitokey.CommitteeBase58KeyListToStruct(publicKeys)
	if err != nil {
		return nil, err
	}
	s.CommitteePublicKeysStruct = publicKeyStructs
	return s, nil
}

func (s *StopAutoStakeInstruction) ToString() []string {
	stopAutoStakeInstructionStr := []string{STOP_AUTO_STAKE_ACTION}
	stopAutoStakeInstructionStr = append(stopAutoStakeInstructionStr, strings.Join(s.CommitteePublicKeys, SPLITTER))
	return stopAutoStakeInstructionStr
}

func ValidateAndImportStopAutoStakeInstructionFromString(instruction []string) (*StopAutoStakeInstruction, error) {
	if err := ValidateStopAutoStakeInstructionSanity(instruction); err != nil {
		return nil, err
	}
	return ImportStopAutoStakeInstructionFromString(instruction), nil
}

func ImportStopAutoStakeInstructionFromString(instruction []string) *StopAutoStakeInstruction {
	stopAutoStakeInstruction := NewStopAutoStakeInstruction()
	if len(instruction[1]) > 0 {
		publicKeys := strings.Split(instruction[1], SPLITTER)
		stopAutoStakeInstruction, _ = stopAutoStakeInstruction.SetPublicKeys(publicKeys)
	}
	return stopAutoStakeInstruction
}

func ValidateStopAutoStakeInstructionSanity(instruction []string) error {
	if len(instruction) != 2 {
		return fmt.Errorf("invalid length, %+v", instruction)
	}
	if instruction[0] != STOP_AUTO_STAKE_ACTION {
		return fmt.Errorf("invalid stop auto stake action, %+v", instruction)
	}
	publicKeys := strings.Split(instruction[1], SPLITTER)
	_, err := incognitokey.CommitteeBase58KeyListToStruct(publicKeys)
	if err != nil {
		return err
	}
	return nil
}
