package instruction

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

//SwapInstruction :
//Swap instruction format:
//["swap-action", list-keys-in, list-keys-out, shard or beacon chain, shard_id(optional), "punished public key", "new reward receivers"]
type SwapInstruction struct {
	InPublicKeys        []string
	InPublicKeyStructs  []incognitokey.CommitteePublicKey
	OutPublicKeys       []string
	OutPublicKeyStructs []incognitokey.CommitteePublicKey
	ChainID             int
	// old slashing, never used
	PunishedPublicKeys string
	// this field is only for replace committee
	NewRewardReceivers       []string
	NewRewardReceiverStructs []privacy.PaymentAddress
	IsReplace                bool
}

func NewSwapInstructionWithValue(inPublicKeys []string, outPublicKeys []string, chainID int) *SwapInstruction {
	producersBlackList := make(map[string]uint8)
	badProducersWithPunishmentBytes, _ := json.Marshal(producersBlackList)
	return &SwapInstruction{InPublicKeys: inPublicKeys, OutPublicKeys: outPublicKeys, ChainID: chainID, PunishedPublicKeys: string(badProducersWithPunishmentBytes), NewRewardReceivers: []string{}}
}

func NewSwapInstruction() *SwapInstruction {
	return &SwapInstruction{}
}

func (s *SwapInstruction) GetType() string {
	return SWAP_ACTION
}

func (s *SwapInstruction) ToString() []string {
	swapInstructionStr := []string{SWAP_ACTION}
	swapInstructionStr = append(swapInstructionStr, strings.Join(s.InPublicKeys, SPLITTER))
	swapInstructionStr = append(swapInstructionStr, strings.Join(s.OutPublicKeys, SPLITTER))
	if s.ChainID == BEACON_CHAIN_ID {
		swapInstructionStr = append(swapInstructionStr, BEACON_INST)
		if len(s.NewRewardReceivers) > 0 {
			swapInstructionStr = append(swapInstructionStr, "")
		}
	} else {
		swapInstructionStr = append(swapInstructionStr, SHARD_INST)
		swapInstructionStr = append(swapInstructionStr, fmt.Sprintf("%v", s.ChainID))
	}
	swapInstructionStr = append(swapInstructionStr, s.PunishedPublicKeys)
	if len(s.NewRewardReceivers) > 0 {
		swapInstructionStr = append(swapInstructionStr, strings.Join(s.NewRewardReceivers, SPLITTER))
	}
	return swapInstructionStr
}

func (s *SwapInstruction) SetInPublicKeys(inPublicKeys []string) (*SwapInstruction, error) {
	s.InPublicKeys = inPublicKeys
	inPublicKeyStructs, err := incognitokey.CommitteeBase58KeyListToStruct(inPublicKeys)
	if err != nil {
		return nil, err
	}
	s.InPublicKeyStructs = inPublicKeyStructs
	return s, nil
}

func (s *SwapInstruction) SetOutPublicKeys(outPublicKeys []string) (*SwapInstruction, error) {
	s.OutPublicKeys = outPublicKeys
	outPublicKeyStructs, err := incognitokey.CommitteeBase58KeyListToStruct(outPublicKeys)
	if err != nil {
		return nil, err
	}
	s.OutPublicKeyStructs = outPublicKeyStructs
	return s, nil
}

func (s *SwapInstruction) SetChainID(chainID int) *SwapInstruction {
	s.ChainID = chainID
	return s
}

func (s *SwapInstruction) SetPunishedPublicKeys(punishedPublicKeys string) *SwapInstruction {
	s.PunishedPublicKeys = punishedPublicKeys
	return s
}

func (s *SwapInstruction) SetNewRewardReceivers(newRewardReceivers []string) *SwapInstruction {
	s.NewRewardReceivers = newRewardReceivers
	for _, v := range newRewardReceivers {
		rewardReceiver, _ := wallet.Base58CheckDeserialize(v)
		s.NewRewardReceiverStructs = append(s.NewRewardReceiverStructs, rewardReceiver.KeySet.PaymentAddress)
	}
	return s
}

func (s *SwapInstruction) SetIsReplace(isReplace bool) *SwapInstruction {
	s.IsReplace = isReplace
	return s
}

func ValidateAndImportSwapInstructionFromString(instruction []string) (*SwapInstruction, error) {
	if err := ValidateSwapInstructionSanity(instruction); err != nil {
		return nil, err
	}
	return ImportSwapInstructionFromString(instruction), nil
}

func ImportSwapInstructionFromString(instruction []string) *SwapInstruction {
	swapInstruction := NewSwapInstruction()

	inPublicKey := []string{}
	outPublicKey := []string{}

	if len(instruction[1]) > 0 {
		inPublicKey = strings.Split(instruction[1], SPLITTER)
	}
	swapInstruction, _ = swapInstruction.SetInPublicKeys(inPublicKey)

	if len(instruction[2]) > 0 {
		outPublicKey = strings.Split(instruction[2], SPLITTER)
	}
	swapInstruction, _ = swapInstruction.SetOutPublicKeys(outPublicKey)

	if len(instruction) == 7 {
		swapInstruction.SetIsReplace(true)
		swapInstruction.SetNewRewardReceivers(strings.Split(instruction[6], SPLITTER))
	} else {
		swapInstruction.SetIsReplace(false)
		swapInstruction.SetNewRewardReceivers([]string{})
	}
	chain := instruction[3]
	if chain == BEACON_INST {
		swapInstruction.SetChainID(BEACON_CHAIN_ID)
		if len(instruction[4]) > 0 {
			swapInstruction.SetPunishedPublicKeys(instruction[4])
		}
	} else {
		chainID, _ := strconv.Atoi(instruction[4])
		swapInstruction.SetChainID(chainID)
		if len(instruction[5]) > 0 {
			swapInstruction.SetPunishedPublicKeys(instruction[5])
		}
	}
	return swapInstruction
}

// validate swap instruction sanity
// new reward receiver only present in replace committee
// beaconproducer.go: 356 - 367
func ValidateSwapInstructionSanity(instruction []string) error {
	if len(instruction) != 5 && len(instruction) != 6 && len(instruction) != 7 {
		return fmt.Errorf("invalid instruction length, %+v, %+v", len(instruction), instruction)
	}
	if instruction[0] != SWAP_ACTION {
		return fmt.Errorf("invalid swap action, %+v", instruction)
	}
	// beacon instruction
	if len(instruction) == 5 && instruction[3] != BEACON_INST {
		return fmt.Errorf("invalid swap beacon instruction, %+v", instruction)
	}
	// shard instruction
	if len(instruction) == 6 {
		if instruction[3] != SHARD_INST {
			return fmt.Errorf("invalid swap shard instruction, %+v", instruction)
		}
		_, err := strconv.Atoi(instruction[4])
		if err != nil {
			return fmt.Errorf("invalid swap shard id, %+v, %+v", err, instruction)
		}
	}
	if len(instruction) == 7 {
		for _, v := range strings.Split(instruction[6], ",") {
			_, err := wallet.Base58CheckDeserialize(v)
			if err != nil {
				return fmt.Errorf("invalid privacy payment address %+v", err)
			}
		}
	}
	_, err1 := incognitokey.CommitteeBase58KeyListToStruct(strings.Split(instruction[1], ","))
	if err1 != nil {
		return fmt.Errorf("invalid swap in public key type, %+v, %+v", err1, instruction)
	}
	_, err2 := incognitokey.CommitteeBase58KeyListToStruct(strings.Split(instruction[2], ","))
	if err2 != nil {
		return fmt.Errorf("invalid swap out public key type, %+v, %+v", err1, instruction)
	}
	return nil
}
