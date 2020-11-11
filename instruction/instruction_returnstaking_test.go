package instruction

import (
	"log"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

func TestReturnStakeIns_SetPublicKeys(t *testing.T) {

	initPublicKey()

	type fields struct {
		PublicKeys       []string
		PublicKeysStruct []incognitokey.CommitteePublicKey
		ShardID          byte
		StakingTXIDs     []string
		StakingTxHashes  []common.Hash
		PercentReturns   []uint
	}
	type args struct {
		publicKeys []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ReturnStakeInstruction
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
			want: &ReturnStakeInstruction{
				PublicKeys:       []string{},
				PublicKeysStruct: []incognitokey.CommitteePublicKey{},
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
			want: &ReturnStakeInstruction{
				PublicKeys: []string{
					key1, key2, key3, key4,
				},
				PublicKeysStruct: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsI := &ReturnStakeInstruction{
				PublicKeys:       tt.fields.PublicKeys,
				PublicKeysStruct: tt.fields.PublicKeysStruct,
				ShardID:          tt.fields.ShardID,
				StakingTXIDs:     tt.fields.StakingTXIDs,
				StakingTxHashes:  tt.fields.StakingTxHashes,
				PercentReturns:   tt.fields.PercentReturns,
			}
			got, err := rsI.SetPublicKeys(tt.args.publicKeys)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReturnStakeInstruction.SetPublicKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReturnStakeInstruction.SetPublicKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAndImportReturnStakingInstructionFromString(t *testing.T) {

	initPublicKey()
	initTxHash()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *ReturnStakeInstruction
		wantErr bool
	}{
		{
			name: "Length Of List Argument Smaller Than 5",
			args: args{
				instruction: []string{RETURN_ACTION},
			},
			wantErr: true,
		},
		{
			name: "Action Is Not ReturnStakingIns",
			args: args{
				instruction: []string{
					ASSIGN_ACTION,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Tx Hash",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{"xyz", "xyz1", "xyz3", "xyz4"}, SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Percent Return",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"abc", "abc", "abc", "abc"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Length Of Public Key != Length Of Return Staking Txs",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"abc", "abc", "abc", "abc"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Length Of Reward Percent Return != Length Of Return Staking Txs",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"abc", "abc", "abc", "abc"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Length Of Public Key != Length Of Reward Percent Return Staking Tcs",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			want: &ReturnStakeInstruction{
				PublicKeys: []string{
					key1, key2, key3, key4,
				},
				PublicKeysStruct: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				ShardID: 0,
				StakingTXIDs: []string{
					txHash1, txHash2, txHash3, txHash4,
				},
				StakingTxHashes: []common.Hash{
					*incTxHash1, *incTxHash2, *incTxHash3, *incTxHash4,
				},
				PercentReturns: []uint{100, 100, 100, 100},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndImportReturnStakingInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndImportReturnStakingInstructionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateAndImportReturnStakingInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImportReturnStakingInstructionFromString(t *testing.T) {

	initPublicKey()
	initTxHash()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		want    *ReturnStakeInstruction
		wantErr bool
	}{
		{
			name: "Invalid Format Public Keys",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{"123", "123", "123", "123"}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Shard ID is invalid",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					"abc",
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Staking Tx Hash",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						"xyz",
						"xyz1",
						"xyz2",
						"xyz3"},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Percent Returns",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"abc", "abc", "abc", "abc"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			want: &ReturnStakeInstruction{
				PublicKeys: []string{
					key1, key2, key3, key4,
				},
				PublicKeysStruct: []incognitokey.CommitteePublicKey{
					*incKey1, *incKey2, *incKey3, *incKey4,
				},
				ShardID: 0,
				StakingTXIDs: []string{
					txHash1, txHash2, txHash3, txHash4,
				},
				StakingTxHashes: []common.Hash{
					*incTxHash1, *incTxHash2, *incTxHash3, *incTxHash4,
				},
				PercentReturns: []uint{100, 100, 100, 100},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ImportReturnStakingInstructionFromString(tt.args.instruction)
			if (err != nil) != tt.wantErr {
				t.Errorf("ImportReturnStakingInstructionFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ImportReturnStakingInstructionFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateReturnStakingInstructionSanity(t *testing.T) {

	initPublicKey()

	type args struct {
		instruction []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Length Of List Argument Smaller Than 5",
			args: args{
				instruction: []string{RETURN_ACTION},
			},
			wantErr: true,
		},
		{
			name: "Action Is Not ReturnStakingIns",
			args: args{
				instruction: []string{
					ASSIGN_ACTION,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Tx Hash",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{"xyz", "xyz1", "xyz3", "xyz4"}, SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Percent Return",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"abc", "abc", "abc", "abc"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Length Of Public Key != Length Of Return Staking Txs",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"abc", "abc", "abc", "abc"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Length Of Reward Percent Return != Length Of Return Staking Txs",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"abc", "abc", "abc", "abc"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Length Of Public Key != Length Of Reward Percent Return Staking Tcs",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			args: args{
				instruction: []string{
					RETURN_ACTION,
					strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
					strconv.Itoa(0),
					strings.Join([]string{
						txHash1,
						txHash2,
						txHash3,
						txHash4},
						SPLITTER),
					strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateReturnStakingInstructionSanity(tt.args.instruction); (err != nil) != tt.wantErr {
				t.Errorf("ValidateReturnStakingInstructionSanity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReturnStakeIns_ToString(t *testing.T) {
	type fields struct {
		PublicKeys       []string
		PublicKeysStruct []incognitokey.CommitteePublicKey
		ShardID          byte
		StakingTXIDs     []string
		StakingTxHashes  []common.Hash
		PercentReturns   []uint
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "Valid Input",
			fields: fields{
				PublicKeys: []string{
					key1, key2, key3, key4,
				},
				ShardID: 0,
				StakingTXIDs: []string{
					"1", "2", "3", "4",
				},
				PercentReturns: []uint{
					100, 100, 100, 100,
				},
			},
			want: []string{
				RETURN_ACTION,
				strings.Join([]string{key1, key2, key3, key4}, SPLITTER),
				strconv.Itoa(0),
				strings.Join([]string{"1", "2", "3", "4"}, SPLITTER),
				strings.Join([]string{"100", "100", "100", "100"}, SPLITTER),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsI := &ReturnStakeInstruction{
				PublicKeys:       tt.fields.PublicKeys,
				PublicKeysStruct: tt.fields.PublicKeysStruct,
				ShardID:          tt.fields.ShardID,
				StakingTXIDs:     tt.fields.StakingTXIDs,
				StakingTxHashes:  tt.fields.StakingTxHashes,
				PercentReturns:   tt.fields.PercentReturns,
			}
			if got := rsI.ToString(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReturnStakeInstruction.ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReturnStakeIns_SetStakingTXIDs(t *testing.T) {

	initTxHash()

	type fields struct {
		PublicKeys       []string
		PublicKeysStruct []incognitokey.CommitteePublicKey
		ShardID          byte
		StakingTXIDs     []string
		StakingTxHashes  []common.Hash
		PercentReturns   []uint
	}
	type args struct {
		txIDs []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ReturnStakeInstruction
		wantErr bool
	}{
		{
			name:   "NULL List Tx Hashes",
			fields: fields{},
			args: args{
				txIDs: nil,
			},
			wantErr: true,
		},
		{
			name:   "Empty List Tx Hashes",
			fields: fields{},
			args: args{
				txIDs: []string{},
			},
			want: &ReturnStakeInstruction{
				StakingTXIDs:    []string{},
				StakingTxHashes: []common.Hash{},
			},
			wantErr: false,
		},
		{
			name:   "Invalid Format Tx Hashes",
			fields: fields{},
			args: args{
				txIDs: []string{
					"xyz", "xyz1", "xyz2", "xyz3",
				},
			},
			want:    &ReturnStakeInstruction{},
			wantErr: true,
		},
		{
			name:   "Valid Input",
			fields: fields{},
			args: args{
				txIDs: []string{
					txHash1, txHash2, txHash3, txHash4,
				},
			},
			want: &ReturnStakeInstruction{
				StakingTXIDs: []string{
					txHash1, txHash2, txHash3, txHash4,
				},
				StakingTxHashes: []common.Hash{
					*incTxHash1, *incTxHash2, *incTxHash3, *incTxHash4,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsI := &ReturnStakeInstruction{
				PublicKeys:       tt.fields.PublicKeys,
				PublicKeysStruct: tt.fields.PublicKeysStruct,
				ShardID:          tt.fields.ShardID,
				StakingTXIDs:     tt.fields.StakingTXIDs,
				StakingTxHashes:  tt.fields.StakingTxHashes,
				PercentReturns:   tt.fields.PercentReturns,
			}
			got, err := rsI.SetStakingTXIDs(tt.args.txIDs)
			log.Println("tt.name:", tt.name)
			log.Println("err:", err)
			log.Println("tt.wantErr:", tt.wantErr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReturnStakeInstruction.SetStakingTXIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ReturnStakeInstruction.SetStakingTXIDs() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
