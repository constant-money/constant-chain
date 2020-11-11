package instruction

import (
	"github.com/incognitochain/incognito-chain/incognitokey"
	"reflect"
	"strings"
	"testing"
)

func TestValidateStopAutoStakeInstructionSanity(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Length of instruction is not equal with 2",
			args: args{
				instruction: []string{},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] is not stop auto staking action",
			args: args{
				instruction: []string{RANDOM_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{STOP_AUTO_STAKE_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER)},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateStopAutoStakeInstructionSanity(tt.args.instruction); (err != nil) != tt.wantErr {
				t.Errorf("ValidateStopAutoStakeInstructionSanity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAndImportStopAutoStakeInstructionFromString(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *StopAutoStakeInstruction
		wantErr bool
	}{
		{
			name: "Length of instruction is not equal with 2",
			args: args{
				instruction: []string{},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] is not stop auto staking action",
			args: args{
				instruction: []string{RANDOM_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER)},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{STOP_AUTO_STAKE_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER)},
			},
			want: &StopAutoStakeInstruction{
				CommitteePublicKeys:       []string{key1, key2, key3, key4},
				CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey1, *incKey2, *incKey3, *incKey4},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndImportStopAutoStakeInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndImportStopAutoStakeInstructionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateAndImportStopAutoStakeInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

//Stop auto staking instruction format:
//["stop_auto_staking_action", list_public_keys]
func TestImportStopAutoStakeInstructionFromString(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name string
		args args
		want *StopAutoStakeInstruction
	}{
		{
			name: "One stop auto stake instruction",
			args: args{
				instruction: []string{STOP_AUTO_STAKE_ACTION, strings.Join([]string{key1}, SPLITTER)},
			},
			want: &StopAutoStakeInstruction{
				CommitteePublicKeys:       []string{key1},
				CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey1},
			},
		},
		{
			name: "Many keys",
			args: args{
				instruction: []string{STOP_AUTO_STAKE_ACTION, strings.Join([]string{key1, key2, key3}, SPLITTER)},
			},
			want: &StopAutoStakeInstruction{
				CommitteePublicKeys:       []string{key1, key2, key3},
				CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey1, *incKey2, *incKey3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImportStopAutoStakeInstructionFromString(tt.args.instruction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportStopAutoStakeInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStopAutoStakeInstruction_ToString(t *testing.T) {
	type args struct {
		instruction *StopAutoStakeInstruction
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "One stop auto stake instruction",
			args: args{
				instruction: &StopAutoStakeInstruction{
					CommitteePublicKeys: []string{key1},
				},
			},
			want: []string{STOP_AUTO_STAKE_ACTION, strings.Join([]string{key1}, SPLITTER)},
		},
		{
			name: "Many keys",
			args: args{
				instruction: &StopAutoStakeInstruction{
					CommitteePublicKeys: []string{key1, key2, key3},
				},
			},
			want: []string{STOP_AUTO_STAKE_ACTION, strings.Join([]string{key1, key2, key3}, SPLITTER)},
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
