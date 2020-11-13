package instruction

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

//StakeInstruction :
//StakeInstruction Format:
// ["STAKE_ACTION", list_public_keys, chain or beacon, list_txs, list_reward_addresses, list_autostaking_status(boolean)]
type StakeInstruction struct {
	PublicKeys            []string
	PublicKeyStructs      []incognitokey.CommitteePublicKey
	Chain                 string
	TxStakes              []string
	TxStakeHashes         []common.Hash
	RewardReceivers       []string
	RewardReceiverStructs []privacy.PaymentAddress
	AutoStakingFlag       []bool
}

func NewStakeInstructionWithValue(
	publicKeys []string, chain string,
	txStakes []string, rewardReceivers []string,
	autoStakingFlag []bool) *StakeInstruction {
	stakeInstruction := &StakeInstruction{}
	stakeInstruction.SetPublicKeys(publicKeys)
	stakeInstruction.SetChain(chain)
	stakeInstruction.SetTxStakes(txStakes)
	stakeInstruction.SetRewardReceivers(rewardReceivers)
	stakeInstruction.SetAutoStakingFlag(autoStakingFlag)
	return stakeInstruction
}

func NewStakeInstruction() *StakeInstruction {
	return &StakeInstruction{}
}

func (s *StakeInstruction) IsEmpty() bool {
	return reflect.DeepEqual(s, NewStakeInstruction()) ||
		len(s.PublicKeyStructs) == 0 && len(s.PublicKeys) == 0
}

func (s *StakeInstruction) GetType() string {
	return STAKE_ACTION
}

func (s *StakeInstruction) SetPublicKeys(publicKeys []string) (*StakeInstruction, error) {
	s.PublicKeys = publicKeys
	publicKeyStructs, err := incognitokey.CommitteeBase58KeyListToStruct(publicKeys)
	if err != nil {
		return nil, err
	}
	s.PublicKeyStructs = publicKeyStructs
	return s, nil
}

func (s *StakeInstruction) SetChain(chain string) *StakeInstruction {
	s.Chain = chain
	return s
}

func (s *StakeInstruction) SetTxStakes(txStakes []string) *StakeInstruction {
	s.TxStakes = txStakes
	for _, txStake := range txStakes {
		res, _ := common.Hash{}.NewHashFromStr(txStake)
		s.TxStakeHashes = append(s.TxStakeHashes, *res)
		s.TxStakes = append(s.TxStakes, txStake)
	}
	return s
}

func (s *StakeInstruction) SetRewardReceivers(rewardReceivers []string) *StakeInstruction {
	s.RewardReceivers = rewardReceivers
	for _, v := range rewardReceivers {
		wl, _ := wallet.Base58CheckDeserialize(v)
		s.RewardReceiverStructs = append(s.RewardReceiverStructs, wl.KeySet.PaymentAddress)
	}
	return s
}

func (s *StakeInstruction) SetAutoStakingFlag(autoStakingFlag []bool) *StakeInstruction {
	s.AutoStakingFlag = autoStakingFlag
	return s
}

func (s *StakeInstruction) ToString() []string {
	stakeInstructionStr := []string{STAKE_ACTION}
	stakeInstructionStr = append(stakeInstructionStr, strings.Join(s.PublicKeys, SPLITTER))
	stakeInstructionStr = append(stakeInstructionStr, s.Chain)
	stakeInstructionStr = append(stakeInstructionStr, strings.Join(s.TxStakes, SPLITTER))
	stakeInstructionStr = append(stakeInstructionStr, strings.Join(s.RewardReceivers, SPLITTER))
	tempStopAutoStakeFlag := []string{}
	for _, v := range s.AutoStakingFlag {
		if v == true {
			tempStopAutoStakeFlag = append(tempStopAutoStakeFlag, TRUE)
		} else {
			tempStopAutoStakeFlag = append(tempStopAutoStakeFlag, FALSE)
		}
	}
	stakeInstructionStr = append(stakeInstructionStr, strings.Join(tempStopAutoStakeFlag, SPLITTER))
	return stakeInstructionStr
}

func ValidateAndImportStakeInstructionFromString(instruction []string) (*StakeInstruction, error) {
	if err := ValidateStakeInstructionSanity(instruction); err != nil {
		return nil, err
	}
	return ImportStakeInstructionFromString(instruction), nil
}

// ImportStakeInstructionFromString is unsafe method
func ImportStakeInstructionFromString(instruction []string) *StakeInstruction {
	stakeInstruction := NewStakeInstruction()
	stakeInstruction, _ = stakeInstruction.SetPublicKeys(strings.Split(instruction[1], SPLITTER))
	stakeInstruction.SetTxStakes(strings.Split(instruction[3], SPLITTER))
	stakeInstruction.SetRewardReceivers(strings.Split(instruction[4], SPLITTER))
	tempAutoStakes := strings.Split(instruction[5], SPLITTER)
	autoStakeFlags := []bool{}
	for _, v := range tempAutoStakes {
		if v == TRUE {
			autoStakeFlags = append(autoStakeFlags, true)
		} else {
			autoStakeFlags = append(autoStakeFlags, false)
		}
	}
	stakeInstruction.SetAutoStakingFlag(autoStakeFlags)
	stakeInstruction.SetChain(instruction[2])
	return stakeInstruction
}

// validate stake instruction sanity
// beaconprocess.go: 1122 - 1165
// beaconproducer.go: 386
func ValidateStakeInstructionSanity(instruction []string) error {
	if len(instruction) != 6 {
		return fmt.Errorf("invalid length, %+v", instruction)
	}
	if instruction[0] != STAKE_ACTION {
		return fmt.Errorf("invalid stake action, %+v", instruction)
	}
	if instruction[2] != SHARD_INST && instruction[2] != BEACON_INST {
		return fmt.Errorf("invalid chain id, %+v", instruction)
	}
	publicKeys := strings.Split(instruction[1], SPLITTER)
	txStakes := strings.Split(instruction[3], SPLITTER)
	for _, txStake := range txStakes {
		_, err := common.Hash{}.NewHashFromStr(txStake)
		if err != nil {
			return fmt.Errorf("invalid tx stake %+v", err)
		}
	}
	rewardReceivers := strings.Split(instruction[4], SPLITTER)
	for _, v := range rewardReceivers {
		_, err := wallet.Base58CheckDeserialize(v)
		if err != nil {
			return fmt.Errorf("invalid privacy payment address %+v", err)
		}
	}
	autoStakes := strings.Split(instruction[5], SPLITTER)
	_, err := incognitokey.CommitteeBase58KeyListToStruct(publicKeys)
	if err != nil {
		return fmt.Errorf("invalid public key type,err %+v, %+v", err, instruction)
	}
	if len(publicKeys) != len(txStakes) {
		return fmt.Errorf("invalid public key & tx stake length, %+v", instruction)
	}
	if len(rewardReceivers) != len(txStakes) {
		return fmt.Errorf("invalid reward receivers & tx stake length, %+v", instruction)
	}
	if len(rewardReceivers) != len(autoStakes) {
		return fmt.Errorf("invalid reward receivers & tx auto staking length, %+v", instruction)
	}
	return nil
}

// ImportStakeInstructionFromString is unsafe method
func ImportInitStakeInstructionFromString(instruction []string) *StakeInstruction {
	stakeInstruction := NewStakeInstruction()
	stakeInstruction, _ = stakeInstruction.SetPublicKeys(strings.Split(instruction[1], SPLITTER))
	stakeInstruction.SetTxStakes(strings.Split(instruction[3], SPLITTER))
	stakeInstruction.SetRewardReceivers(strings.Split(instruction[4], SPLITTER))
	tempAutoStakes := strings.Split(instruction[5], SPLITTER)
	autoStakeFlags := []bool{}
	for _, v := range tempAutoStakes {
		if v == TRUE {
			autoStakeFlags = append(autoStakeFlags, true)
		} else {
			autoStakeFlags = append(autoStakeFlags, false)
		}
	}
	stakeInstruction.SetAutoStakingFlag(autoStakeFlags)
	stakeInstruction.SetChain(instruction[2])
	return stakeInstruction
}
