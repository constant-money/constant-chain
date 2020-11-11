package instruction

import (
	"reflect"
	"testing"
)

func TestValidateRandomInstructionSanity(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Length of instructions != 5",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] != RANDOM_ACTION",
			args: args{
				instruction: []string{ASSIGN_ACTION, "3157440766", "637918", "3157440766", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[1]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "abdcd", "637918", "3157440766", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[2]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "dasda", "3157440766", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[3]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "aca", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[4]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766", "acxac"},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766", "3157440766"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateRandomInstructionSanity(tt.args.instruction); (err != nil) != tt.wantErr {
				t.Errorf("ValidateRandomInstructionSanity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//Random instruction fomart:
//["random-action", btc_none, btc_height, btc_check_point_time, btc_block_time]
func TestValidateAndImportRandomInstructionFromString(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *RandomInstruction
		wantErr bool
	}{
		{
			name: "Length of instructions != 5",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "instruction[0] != RANDOM_ACTION",
			args: args{
				instruction: []string{ASSIGN_ACTION, "3157440766", "637918", "3157440766", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[1]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "abdcd", "637918", "3157440766", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[2]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "dasda", "3157440766", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[3]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "aca", "3157440766"},
			},
			wantErr: true,
		},
		{
			name: "type(instruction[4]) != int",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766", "acxac"},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766", "3157440766"},
			},
			want: &RandomInstruction{
				BtcNonce:       3157440766,
				BtcBlockHeight: 637918,
				CheckPointTime: 3157440766,
				BtcBlockTime:   3157440766,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndImportRandomInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndImportRandomInstructionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateAndImportRandomInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImportRandomInstructionFromString(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name string
		args args
		want *RandomInstruction
	}{
		{
			name: "Valid Input",
			args: args{
				instruction: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766", "3157440766"},
			},
			want: &RandomInstruction{
				BtcNonce:       3157440766,
				BtcBlockHeight: 637918,
				CheckPointTime: 3157440766,
				BtcBlockTime:   3157440766,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImportRandomInstructionFromString(tt.args.instruction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportRandomInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomInstruction_ToString(t *testing.T) {
	type args struct {
		instruction *RandomInstruction
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Valid Input",
			args: args{
				instruction: &RandomInstruction{
					BtcNonce:       3157440766,
					BtcBlockHeight: 637918,
					CheckPointTime: 3157440766,
					BtcBlockTime:   3157440766,
				},
			},
			want: []string{RANDOM_ACTION, "3157440766", "637918", "3157440766", "3157440766"},
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
