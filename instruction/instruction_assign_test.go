package instruction

import (
	"reflect"
	"strings"
	"testing"

	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/stretchr/testify/assert"
)

//TestValidateAndImportAssignInstructionFromString ...
func TestValidateAndImportAssignInstructionFromString(t *testing.T) {

	initPublicKey()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *AssignInstruction
		wantErr bool
	}{
		{
			name: "Length of instruction != 4",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] != ASSIGN_ACTION",
			args: args{
				instruction: []string{RANDOM_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "0"},
			},
			wantErr: true,
		},
		{
			name: "instruction[2] != SHARD_INST",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), BEACON_INST, "0"},
			},
			wantErr: true,
		},
		{
			name: "instruction[3] is not integer",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "not's ok"},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "0"},
			},
			want: &AssignInstruction{
				ChainID: 0,
				ShardCandidates: []string{
					key1, key2, key3, key4,
				},
				ShardCandidatesStruct: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndImportAssignInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				assert.Equal(t, tt.want, got)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestValidateAssignInstructionSanity(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Length of instruction != 4",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] != ASSIGN_ACTION",
			args: args{
				instruction: []string{RANDOM_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "0"},
			},
			wantErr: true,
		},
		{
			name: "instruction[2] != SHARD_INST",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), BEACON_INST, "0"},
			},
			wantErr: true,
		},
		{
			name: "instruction[3] is not integer",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "not's ok"},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "0"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateAssignInstructionSanity(tt.args.instruction); (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssignInstructionSanity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportAssignInstructionFromString(t *testing.T) {

	initPublicKey()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name string
		args args
		want *AssignInstruction
	}{
		{
			name: "Valid Input",
			args: args{
				instruction: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "0"},
			},
			want: &AssignInstruction{
				ChainID:         0,
				ShardCandidates: []string{key1, key2, key3, key4},
				ShardCandidatesStruct: []incognitokey.CommitteePublicKey{
					*incKey1,
					*incKey2,
					*incKey3,
					*incKey4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := ImportAssignInstructionFromString(tt.args.instruction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportAssignInstructionFromString() = %v, want %v", got, tt.want)
				assert.Nil(t, err, "Error is not null")
			}
		})
	}
}

func TestAssignInstruction_ToString(t *testing.T) {
	type args struct {
		instruction *AssignInstruction
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Valid Input",
			args: args{
				instruction: &AssignInstruction{
					ChainID:         0,
					ShardCandidates: []string{key1, key2, key3, key4},
				},
			},
			want: []string{ASSIGN_ACTION, strings.Join([]string{key1, key2, key3, key4}, SPLITTER), SHARD_INST, "0"},
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
