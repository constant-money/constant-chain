package instruction

import (
	"reflect"
	"strings"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/stretchr/testify/assert"
)

func TestValidateAndImportStakeInstructionFromString(t *testing.T) {

	initPublicKey()
	initPaymentAddress()
	initTxHash()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *StakeInstruction
		wantErr bool
	}{
		{
			name: "len(instruction) != 6",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] != STAKE_ACTION",
			args: args{
				instruction: []string{ASSIGN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Invalid chain id",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					"test",
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Invalid public key type",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{"key1", "key2", "key3", "key4"}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of tx stakes and public key is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of public key and reward address is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of public key and tx stop auto staking before is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of tx stakes and reward address is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of tx stakes and stop auto staking request is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of reward address and stop auto staking request is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{
						txHash1, txHash2, txHash3, txHash4,
					}, SPLITTER),
					strings.Join([]string{
						paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4,
					}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			want: &StakeInstruction{
				PublicKeys: []string{key1, key2, key3, key4},
				PublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1,
					*incKey2,
					*incKey3,
					*incKey4,
				},
				Chain: SHARD_INST,
				TxStakes: []string{
					txHash1, txHash2, txHash3, txHash4,
				},
				TxStakeHashes: []common.Hash{
					*incTxHash1,
					*incTxHash2,
					*incTxHash3,
					*incTxHash4,
				},
				RewardReceiverStructs: []privacy.PaymentAddress{
					*incPaymentAddress1,
					*incPaymentAddress2,
					*incPaymentAddress3,
					*incPaymentAddress4,
				},
				RewardReceivers: []string{
					paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4,
				},
				AutoStakingFlag: []bool{true, true, true, true},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndImportStakeInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndImportStakeInstructionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				assert.Equal(t, got.RewardReceiverStructs, tt.want.RewardReceiverStructs)
				t.Errorf("ValidateAndImportStakeInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateStakeInstructionSanity(t *testing.T) {

	initPublicKey()
	initPaymentAddress()
	initTxHash()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "len(instruction) != 6",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] != STAKE_ACTION",
			args: args{
				instruction: []string{ASSIGN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Invalid chain id",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					"test",
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Invalid public key type",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{"key1", "key2", "key3", "key4"}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of tx stakes and public key is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of public key and reward address is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of public key and tx stop auto staking before is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of tx stakes and reward address is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of tx stakes and stop auto staking request is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
					strings.Join([]string{"true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Length of reward address and stop auto staking request is not similar",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
					strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3"}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{
						txHash1, txHash2, txHash3, txHash4}, SPLITTER),
					strings.Join([]string{
						paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateStakeInstructionSanity(tt.args.instruction); (err != nil) != tt.wantErr {
				t.Errorf("ValidateStakeInstructionSanity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportStakeInstructionFromString(t *testing.T) {

	initPublicKey()
	initPaymentAddress()
	initTxHash()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name string
		args args
		want *StakeInstruction
	}{
		{
			name: "Valid Input",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{
						txHash1, txHash2, txHash3, txHash4}, SPLITTER),
					strings.Join([]string{
						paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			want: &StakeInstruction{
				PublicKeys: []string{key1, key2, key3, key4},
				PublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1,
					*incKey2,
					*incKey3,
					*incKey4,
				},
				Chain: SHARD_INST,
				TxStakeHashes: []common.Hash{
					*incTxHash1, *incTxHash2, *incTxHash3, *incTxHash4,
				},
				TxStakes: []string{txHash1, txHash2, txHash3, txHash4},
				RewardReceivers: []string{
					paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4,
				},
				RewardReceiverStructs: []privacy.PaymentAddress{
					*incPaymentAddress1, *incPaymentAddress2, *incPaymentAddress3, *incPaymentAddress4,
				},
				AutoStakingFlag: []bool{true, true, true, true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImportStakeInstructionFromString(tt.args.instruction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportStakeInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStakeInstruction_ToString(t *testing.T) {
	initPublicKey()
	type args struct {
		instruction *StakeInstruction
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Valid Input",
			args: args{
				instruction: &StakeInstruction{
					PublicKeys: []string{key1, key2, key3, key4},
					PublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey1,
						*incKey2,
						*incKey3,
						*incKey4,
					},
					Chain:           SHARD_INST,
					TxStakes:        []string{"tx1", "tx2", "tx3", "tx4"},
					RewardReceivers: []string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"},
					AutoStakingFlag: []bool{true, true, true, true},
				},
			},
			want: []string{STAKE_ACTION,
				strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
				SHARD_INST,
				strings.Join([]string{"tx1", "tx2", "tx3", "tx4"}, SPLITTER),
				strings.Join([]string{"reward-addr1", "reward-addr2", "reward-addr3", "reward-addr4"}, SPLITTER),
				strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.instruction.ToString(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImportInitStakeInstructionFromString(t *testing.T) {

	initPublicKey()
	initPaymentAddress()
	initTxHash()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name string
		args args
		want *StakeInstruction
	}{
		{
			name: "Valid Input - auto stake = true",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{
						txHash1, txHash2, txHash3, txHash4}, SPLITTER),
					strings.Join([]string{
						paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4}, SPLITTER),
					strings.Join([]string{"true", "true", "true", "true"}, SPLITTER)},
			},
			want: &StakeInstruction{
				PublicKeys: []string{key1, key2, key3, key4},
				PublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1,
					*incKey2,
					*incKey3,
					*incKey4,
				},
				Chain: SHARD_INST,
				TxStakes: []string{
					txHash1, txHash2, txHash3, txHash4,
				},
				TxStakeHashes: []common.Hash{
					*incTxHash1,
					*incTxHash2,
					*incTxHash3,
					*incTxHash4,
				},
				RewardReceiverStructs: []privacy.PaymentAddress{
					*incPaymentAddress1,
					*incPaymentAddress2,
					*incPaymentAddress3,
					*incPaymentAddress4,
				},
				RewardReceivers: []string{
					paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4,
				},
				AutoStakingFlag: []bool{true, true, true, true},
			},
		},
		{
			name: "Valid Input - auto stake = false",
			args: args{
				instruction: []string{STAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					SHARD_INST,
					strings.Join([]string{
						txHash1, txHash2, txHash3, txHash4}, SPLITTER),
					strings.Join([]string{
						paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4}, SPLITTER),
					strings.Join([]string{"false", "false", "false", "false"}, SPLITTER)},
			},
			want: &StakeInstruction{
				PublicKeys: []string{key1, key2, key3, key4},
				PublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1,
					*incKey2,
					*incKey3,
					*incKey4,
				},
				Chain: SHARD_INST,
				TxStakes: []string{
					txHash1, txHash2, txHash3, txHash4,
				},
				TxStakeHashes: []common.Hash{
					*incTxHash1,
					*incTxHash2,
					*incTxHash3,
					*incTxHash4,
				},
				RewardReceiverStructs: []privacy.PaymentAddress{
					*incPaymentAddress1,
					*incPaymentAddress2,
					*incPaymentAddress3,
					*incPaymentAddress4,
				},
				RewardReceivers: []string{
					paymentAddress1, paymentAddress2, paymentAddress3, paymentAddress4,
				},
				AutoStakingFlag: []bool{false, false, false, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImportInitStakeInstructionFromString(tt.args.instruction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportInitStakeInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
