package instruction

import (
	"reflect"
	"strings"
	"testing"

	"github.com/incognitochain/incognito-chain/incognitokey"
)

func TestSwapShardInstruction_ToString(t *testing.T) {
	type fields struct {
		InPublicKeys        []string
		InPublicKeyStructs  []incognitokey.CommitteePublicKey
		OutPublicKeys       []string
		OutPublicKeyStructs []incognitokey.CommitteePublicKey
		ChainID             int
		Type                int
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "Valid Input",
			fields: fields{
				InPublicKeys: []string{
					"key1", "key2", "key3", "key4",
				},
				OutPublicKeys: []string{
					"key1", "key2", "key3", "key4",
				},
				ChainID: 0,
				Type:    SWAP_BY_END_EPOCH,
			},
			want: []string{
				SWAP_SHARD_ACTION,
				"key1,key2,key3,key4",
				"key1,key2,key3,key4",
				"0",
				"0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SwapShardInstruction{
				InPublicKeys:        tt.fields.InPublicKeys,
				InPublicKeyStructs:  tt.fields.InPublicKeyStructs,
				OutPublicKeys:       tt.fields.OutPublicKeys,
				OutPublicKeyStructs: tt.fields.OutPublicKeyStructs,
				ChainID:             tt.fields.ChainID,
				Type:                tt.fields.Type,
			}
			if got := s.ToString(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SwapShardInstruction.ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSwapShardInstruction_SetInPublicKeys(t *testing.T) {

	initPublicKey()

	type fields struct {
		InPublicKeys        []string
		InPublicKeyStructs  []incognitokey.CommitteePublicKey
		OutPublicKeys       []string
		OutPublicKeyStructs []incognitokey.CommitteePublicKey
		ChainID             int
		Height              uint64
		Type                int
	}
	type args struct {
		inPublicKeys []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SwapShardInstruction
		wantErr bool
	}{
		{
			name:   "Invalid Format Public Key",
			fields: fields{},
			args: args{
				inPublicKeys: []string{
					"key1", "key2", "key3",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "Valid Input",
			fields: fields{},
			args: args{
				inPublicKeys: []string{
					key1, key2, key3, key4,
				},
			},
			want: &SwapShardInstruction{
				InPublicKeys: []string{
					key1, key2, key3, key4,
				},
				InPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SwapShardInstruction{
				InPublicKeys:        tt.fields.InPublicKeys,
				InPublicKeyStructs:  tt.fields.InPublicKeyStructs,
				OutPublicKeys:       tt.fields.OutPublicKeys,
				OutPublicKeyStructs: tt.fields.OutPublicKeyStructs,
				ChainID:             tt.fields.ChainID,
				Type:                tt.fields.Type,
			}
			got, err := s.SetInPublicKeys(tt.args.inPublicKeys)
			if (err != nil) != tt.wantErr {
				t.Errorf("SwapShardInstruction.SetInPublicKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SwapShardInstruction.SetInPublicKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSwapShardInstruction_SetOutPublicKeys(t *testing.T) {
	type fields struct {
		InPublicKeys        []string
		InPublicKeyStructs  []incognitokey.CommitteePublicKey
		OutPublicKeys       []string
		OutPublicKeyStructs []incognitokey.CommitteePublicKey
		ChainID             int
		Height              uint64
		Type                int
	}
	type args struct {
		outPublicKeys []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SwapShardInstruction
		wantErr bool
	}{
		{
			name:   "Invalid Format Public Key",
			fields: fields{},
			args: args{
				outPublicKeys: []string{
					"key1", "key2", "key3",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "Valid Input",
			fields: fields{},
			args: args{
				outPublicKeys: []string{
					key1, key2, key3, key4,
				},
			},
			want: &SwapShardInstruction{
				OutPublicKeys: []string{
					key1, key2, key3, key4,
				},
				OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SwapShardInstruction{
				InPublicKeys:        tt.fields.InPublicKeys,
				InPublicKeyStructs:  tt.fields.InPublicKeyStructs,
				OutPublicKeys:       tt.fields.OutPublicKeys,
				OutPublicKeyStructs: tt.fields.OutPublicKeyStructs,
				ChainID:             tt.fields.ChainID,
				Type:                tt.fields.Type,
			}
			got, err := s.SetOutPublicKeys(tt.args.outPublicKeys)
			if (err != nil) != tt.wantErr {
				t.Errorf("SwapShardInstruction.SetOutPublicKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SwapShardInstruction.SetOutPublicKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSwapShardInstructionSanity(t *testing.T) {
	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "len of instructions is != 5",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 0 is not Swap Shard Action",
			args: args{
				instruction: []string{
					ASSIGN_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 1 is not committee public key format",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{"asdasd"},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 2 is not committee public key format",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{"dasdasdas"},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 3 is not uint",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"dsaasd",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 4 is not int",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"dasdas",
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateSwapShardInstructionSanity(tt.args.instruction); (err != nil) != tt.wantErr {
				t.Errorf("ValidateSwapShardInstructionSanity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportSwapShardInstructionFromString(t *testing.T) {

	initPublicKey()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name string
		args args
		want *SwapShardInstruction
	}{
		{
			name: "Length In Public Keys > 0 && Length Out Public Keys == 0",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"",
					"0",
					"0",
				},
			},
			want: &SwapShardInstruction{
				InPublicKeys: []string{
					key1, key2, key3, key4,
				},
				InPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				OutPublicKeyStructs: []incognitokey.CommitteePublicKey{},
				OutPublicKeys:       []string{},
				ChainID:             0,
				Type:                0,
			},
		},
		{
			name: "Length In Public Keys == 0 && Length Out Public Keys > 0",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					"",
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			want: &SwapShardInstruction{
				InPublicKeys:       []string{},
				InPublicKeyStructs: []incognitokey.CommitteePublicKey{},
				OutPublicKeys: []string{
					key1, key2, key3, key4,
				},
				OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				ChainID: 0,
				Type:    0,
			},
		},
		{
			name: "Length In Public Keys == 0 && Length Out Public Keys == 0",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					"",
					"",
					"0",
					"0",
				},
			},
			want: &SwapShardInstruction{
				InPublicKeys:        []string{},
				InPublicKeyStructs:  []incognitokey.CommitteePublicKey{},
				OutPublicKeys:       []string{},
				OutPublicKeyStructs: []incognitokey.CommitteePublicKey{},
				ChainID:             0,
				Type:                0,
			},
		},
		{
			name: "Length In Public Keys > 0 && Length Out Public Keys > 0",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			want: &SwapShardInstruction{
				OutPublicKeys: []string{
					key1, key2, key3, key4,
				},
				OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				InPublicKeys: []string{
					key1, key2, key3, key4,
				},
				InPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				ChainID: 0,
				Type:    0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImportSwapShardInstructionFromString(tt.args.instruction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportSwapShardInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAndImportSwapShardInstructionFromString(t *testing.T) {

	initPublicKey()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *SwapShardInstruction
		wantErr bool
	}{
		{
			name: "len of instructions is != 5",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 0 is not Swap Shard Action",
			args: args{
				instruction: []string{
					ASSIGN_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 1 is not committee public key format",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{"asdasd"},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 2 is not committee public key format",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{"dasdasdas"},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 3 is not uint",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"dsaasd",
					"0",
				},
			},
			wantErr: true,
		},
		{
			name: "Element 4 is not int",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"dasdas",
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					SWAP_SHARD_ACTION,
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					strings.Join(
						[]string{key1, key2, key3, key4},
						SPLITTER),
					"0",
					"0",
				},
			},
			wantErr: false,
			want: &SwapShardInstruction{
				OutPublicKeys: []string{
					key1, key2, key3, key4,
				},
				OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				InPublicKeys: []string{
					key1, key2, key3, key4,
				},
				InPublicKeyStructs: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				ChainID: 0,
				Type:    0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndImportSwapShardInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndImportSwapShardInstructionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateAndImportSwapShardInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
