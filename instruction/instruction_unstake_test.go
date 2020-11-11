package instruction

import (
	"reflect"
	"strings"
	"testing"

	"github.com/incognitochain/incognito-chain/incognitokey"
)

func TestUnstakeInstruction_ToString(t *testing.T) {
	type fields struct {
		CommitteePublicKeys       []string
		CommitteePublicKeysStruct []incognitokey.CommitteePublicKey
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "Empty List Committee Public Keys",
			fields: fields{
				CommitteePublicKeys: []string{},
			},
			want: []string{UNSTAKE_ACTION, ""},
		},
		{
			name: "Valid Input",
			fields: fields{
				CommitteePublicKeys: []string{
					key1, key2, key3, key4,
				},
			},
			want: []string{
				UNSTAKE_ACTION,
				strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstakeIns := &UnstakeInstruction{
				CommitteePublicKeys: tt.fields.CommitteePublicKeys,
			}
			if got := unstakeIns.ToString(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnstakeInstruction.ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAndImportUnstakeInstructionFromString(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *UnstakeInstruction
		wantErr bool
	}{
		{
			name: "Null List Instruction",
			args: args{
				instruction: nil,
			},
			wantErr: true,
		},
		{
			name: "Empty List Instruction",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
				},
			},
			wantErr: true,
		},
		{
			name: "Action is not unstake",
			args: args{
				instruction: []string{
					RETURN_ACTION,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Format Public Keys",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
					strings.Join([]string{"123", key2, key3, key4}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
				},
			},
			want: &UnstakeInstruction{
				CommitteePublicKeys: []string{
					key1, key2, key3, key4,
				},
				CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndImportUnstakeInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndImportUnstakeInstructionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateAndImportUnstakeInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImportUnstakeInstructionFromString(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name string
		args args
		want *UnstakeInstruction
	}{
		{
			name: "Empty List Instruction",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
				},
			},
			want: &UnstakeInstruction{},
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
				},
			},
			want: &UnstakeInstruction{
				CommitteePublicKeys: []string{
					key1, key2, key3, key4,
				},
				CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImportUnstakeInstructionFromString(tt.args.instruction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportUnstakeInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateUnstakeInstructionSanity(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Null List Instruction",
			args: args{
				instruction: nil,
			},
			wantErr: true,
		},
		{
			name: "Empty List Instruction",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
				},
			},
			wantErr: true,
		},
		{
			name: "Action is not unstake",
			args: args{
				instruction: []string{
					RETURN_ACTION,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Format Public Keys",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
					strings.Join([]string{"123", key2, key3, key4}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					UNSTAKE_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateUnstakeInstructionSanity(tt.args.instruction); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUnstakeInstructionSanity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnstakeInstruction_SetCommitteePublicKeys(t *testing.T) {
	type fields struct {
		CommitteePublicKeys       []string
		CommitteePublicKeysStruct []incognitokey.CommitteePublicKey
	}
	type args struct {
		publicKeys []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "NULL List Public Keys",
			fields: fields{},
			args: args{
				publicKeys: nil,
			},
			wantErr: true,
		},
		{
			name:   "Empty List Public Keys",
			fields: fields{},
			args: args{
				publicKeys: []string{},
			},
			wantErr: false,
		},
		{
			name:   "Invalid Format Committee Public Key",
			fields: fields{},
			args: args{
				publicKeys: []string{
					"123", key2, key3, key4,
				},
			},
			wantErr: true,
		},
		{
			name:   "Valid Input",
			fields: fields{},
			args: args{
				publicKeys: []string{
					key1, key2, key3, key4,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unstakeInstruction := &UnstakeInstruction{
				CommitteePublicKeys:       tt.fields.CommitteePublicKeys,
				CommitteePublicKeysStruct: tt.fields.CommitteePublicKeysStruct,
			}
			if err := unstakeInstruction.SetCommitteePublicKeys(tt.args.publicKeys); (err != nil) != tt.wantErr {
				t.Errorf("UnstakeInstruction.SetCommitteePublicKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
