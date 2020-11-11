package instruction

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/incognitokey"
)

//SwapShardInstruction Shard Swap Instruction
type SwapShardInstruction struct {
	InPublicKeys        []string
	InPublicKeyStructs  []incognitokey.CommitteePublicKey
	OutPublicKeys       []string
	OutPublicKeyStructs []incognitokey.CommitteePublicKey
	ChainID             int // which shard
	Type                int // sub type of swap shard instruction
}

func NewSwapShardInstructionWithValue(
	inPublicKeys, outPublicKeys []string,
	chainID, typeIns int) *SwapShardInstruction {
	swapShardInstruction := &SwapShardInstruction{
		ChainID: chainID,
		Type:    typeIns,
	}
	swapShardInstruction, _ = swapShardInstruction.SetInPublicKeys(inPublicKeys)
	swapShardInstruction, _ = swapShardInstruction.SetOutPublicKeys(outPublicKeys)
	return swapShardInstruction
}

func NewSwapShardInstruction() *SwapShardInstruction {
	return &SwapShardInstruction{}
}

func (s *SwapShardInstruction) GetType() string {
	return SWAP_SHARD_ACTION
}

func (s *SwapShardInstruction) ToString() []string {
	SwapShardInstructionStr := []string{SWAP_SHARD_ACTION}
	SwapShardInstructionStr = append(SwapShardInstructionStr, strings.Join(s.InPublicKeys, SPLITTER))
	SwapShardInstructionStr = append(SwapShardInstructionStr, strings.Join(s.OutPublicKeys, SPLITTER))
	SwapShardInstructionStr = append(SwapShardInstructionStr, fmt.Sprintf("%v", s.ChainID))
	SwapShardInstructionStr = append(SwapShardInstructionStr, fmt.Sprintf("%v", s.Type))
	return SwapShardInstructionStr
}

func (s *SwapShardInstruction) IsEmpty() bool {
	return reflect.DeepEqual(s, NewSwapInstruction()) ||
		len(s.OutPublicKeyStructs) == 0 && len(s.OutPublicKeys) == 0 &&
			len(s.InPublicKeyStructs) == 0 && len(s.InPublicKeys) == 0
}

func (s *SwapShardInstruction) SetInPublicKeys(inPublicKeys []string) (*SwapShardInstruction, error) {
	s.InPublicKeys = inPublicKeys
	inPublicKeyStructs, err := incognitokey.CommitteeBase58KeyListToStruct(inPublicKeys)
	if err != nil {
		return nil, err
	}
	s.InPublicKeyStructs = inPublicKeyStructs
	return s, nil
}

func (s *SwapShardInstruction) SetOutPublicKeys(outPublicKeys []string) (*SwapShardInstruction, error) {
	s.OutPublicKeys = outPublicKeys
	outPublicKeyStructs, err := incognitokey.CommitteeBase58KeyListToStruct(outPublicKeys)
	if err != nil {
		return nil, err
	}
	s.OutPublicKeyStructs = outPublicKeyStructs
	return s, nil
}

func (s *SwapShardInstruction) SetChainID(chainID int) *SwapShardInstruction {
	s.ChainID = chainID
	return s
}

func (s *SwapShardInstruction) SetType(typeIns int) *SwapShardInstruction {
	s.Type = typeIns
	return s
}

func ValidateAndImportSwapShardInstructionFromString(instruction []string) (*SwapShardInstruction, error) {
	if err := ValidateSwapShardInstructionSanity(instruction); err != nil {
		return nil, err
	}
	return ImportSwapShardInstructionFromString(instruction), nil
}

func ImportSwapShardInstructionFromString(instruction []string) *SwapShardInstruction {
	swapShardInstruction := NewSwapShardInstruction()
	inPublicKeys := []string{}
	outPublicKeys := []string{}

	if len(instruction[1]) > 0 {
		inPublicKeys = strings.Split(instruction[1], SPLITTER)
	}
	swapShardInstruction, _ = swapShardInstruction.SetInPublicKeys(inPublicKeys)
	if len(instruction[2]) > 0 {
		outPublicKeys = strings.Split(instruction[2], SPLITTER)
	}
	swapShardInstruction, _ = swapShardInstruction.SetOutPublicKeys(outPublicKeys)
	swapShardInstruction.ChainID, _ = strconv.Atoi(instruction[3])
	swapShardInstruction.Type, _ = strconv.Atoi(instruction[4])

	return swapShardInstruction
}

// validate SwapShard instruction sanity
// new reward receiver only present in replace committee
// beaconproducer.go: 356 - 367
func ValidateSwapShardInstructionSanity(instruction []string) error {
	if len(instruction) != 5 {
		return fmt.Errorf("invalid instruction length %+v, %+v, expect %+v", len(instruction), instruction, 6)
	}
	if instruction[0] != SWAP_SHARD_ACTION {
		return fmt.Errorf("invalid SwapShard action, %+v", instruction)
	}
	_, err1 := incognitokey.CommitteeBase58KeyListToStruct(strings.Split(instruction[1], ","))
	if err1 != nil {
		return fmt.Errorf("invalid SwapShard in public key type, %+v, %+v", err1, instruction)
	}
	_, err2 := incognitokey.CommitteeBase58KeyListToStruct(strings.Split(instruction[2], ","))
	if err2 != nil {
		return fmt.Errorf("invalid SwapShard out public key type, %+v, %+v", err2, instruction)
	}
	_, err3 := strconv.ParseUint(instruction[3], 10, 64)
	if err3 != nil {
		return fmt.Errorf("invalid SwapShard chainID, %+v, %+v", err3, instruction)
	}
	_, err5 := strconv.ParseInt(instruction[4], 10, 64)
	if err5 != nil {
		return fmt.Errorf("invalid RequestShardSwap type, %+v, %+v", err5, instruction)
	}
	return nil
}
