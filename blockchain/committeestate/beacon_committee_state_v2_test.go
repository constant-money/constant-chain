package committeestate

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/stretchr/testify/assert"
)

func SampleCandidateList(len int) []string {
	res := []string{}
	for i := 0; i < len; i++ {
		res = append(res, fmt.Sprintf("committeepubkey%v", i))
	}
	return res
}

func GetMinMaxRange(sizeMap map[byte]int) int {
	min := -1
	max := -1
	for _, v := range sizeMap {
		if min == -1 {
			min = v
		}
		if min > v {
			min = v
		}
		if max < v {
			max = v
		}
	}
	return max - min
}

func TestBeaconCommitteeStateV2_processStakeInstruction(t *testing.T) {

	initStateDB()
	initPublicKey()

	sDB, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)

	paymentAddress := privacy.GeneratePaymentAddress([]byte{1})
	txHash, err := common.Hash{}.NewHashFromStr("123")

	type fields struct {
		beaconCommittee            []incognitokey.CommitteePublicKey
		shardCommittee             map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
		shardCommonPool            []incognitokey.CommitteePublicKey
		numberOfAssignedCandidates int
		autoStake                  map[string]bool
		rewardReceiver             map[string]privacy.PaymentAddress
		stakingTx                  map[string]common.Hash
	}
	type args struct {
		stakeInstruction *instruction.StakeInstruction
		committeeChange  *CommitteeChange
		env              *BeaconCommitteeStateEnvironment
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		want           *CommitteeChange
		wantSideEffect *fields
		wantErr        bool
	}{
		{
			name: "Valid Input",
			fields: fields{
				shardCommittee:  map[byte][]incognitokey.CommitteePublicKey{},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{},
				autoStake:       map[string]bool{},
				rewardReceiver:  map[string]privacy.PaymentAddress{},
				stakingTx:       map[string]common.Hash{},
			},
			args: args{
				stakeInstruction: &instruction.StakeInstruction{
					PublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey,
					},
					PublicKeys: []string{key},
					RewardReceiverStructs: []privacy.PaymentAddress{
						paymentAddress,
					},
					AutoStakingFlag: []bool{true},
					TxStakeHashes: []common.Hash{
						*txHash,
					},
					TxStakes: []string{"123"},
				},
				committeeChange: &CommitteeChange{},
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB: sDB,
				},
			},
			want: &CommitteeChange{
				NextEpochShardCandidateAdded: []incognitokey.CommitteePublicKey{
					*incKey,
				},
			},
			wantSideEffect: &fields{
				shardCommittee:  map[byte][]incognitokey.CommitteePublicKey{},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{},
				shardCommonPool: []incognitokey.CommitteePublicKey{*incKey},
				autoStake: map[string]bool{
					key: true,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key: *txHash,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV2{
				beaconCommittee:            tt.fields.beaconCommittee,
				shardCommittee:             tt.fields.shardCommittee,
				shardSubstitute:            tt.fields.shardSubstitute,
				shardCommonPool:            tt.fields.shardCommonPool,
				numberOfAssignedCandidates: tt.fields.numberOfAssignedCandidates,
				autoStake:                  tt.fields.autoStake,
				rewardReceiver:             tt.fields.rewardReceiver,
				stakingTx:                  tt.fields.stakingTx,
			}
			got, err := b.processStakeInstruction(tt.args.stakeInstruction, tt.args.committeeChange, tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(b.shardCommonPool, tt.wantSideEffect.shardCommonPool) {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction(), shardCommonPool = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(b.shardCommittee, tt.wantSideEffect.shardCommittee) {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction(), shardCommittee = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(b.shardSubstitute, tt.wantSideEffect.shardSubstitute) {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction(), shardSubstitute = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(b.rewardReceiver, tt.wantSideEffect.rewardReceiver) {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction(), rewardReceiver = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(b.autoStake, tt.wantSideEffect.autoStake) {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction(), autoStake = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(b.stakingTx, tt.wantSideEffect.stakingTx) {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction(), stakingTx = %v, want %v", got, tt.want)
			}
			_, has, _ := statedb.GetStakerInfo(tt.args.env.ConsensusStateDB, key)
			if has {
				t.Errorf("BeaconCommitteeStateV2.processStakeInstruction(), StoreStakerInfo found, %+v", key)
			}
		})
	}
}

func TestBeaconCommitteeStateV2_processAssignWithRandomInstruction(t *testing.T) {

	initLog()
	initPublicKey()

	committeeChangeValidInput := NewCommitteeChange()
	committeeChangeValidInput.NextEpochShardCandidateRemoved = []incognitokey.CommitteePublicKey{
		*incKey2,
	}
	committeeChangeValidInput.ShardSubstituteAdded[1] = []incognitokey.CommitteePublicKey{
		*incKey2,
	}

	type fields struct {
		beaconCommittee            []incognitokey.CommitteePublicKey
		shardCommittee             map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
		shardCommonPool            []incognitokey.CommitteePublicKey
		numberOfAssignedCandidates int
		autoStake                  map[string]bool
		rewardReceiver             map[string]privacy.PaymentAddress
		stakingTx                  map[string]common.Hash
	}
	type args struct {
		rand            int64
		activeShards    int
		committeeChange *CommitteeChange
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantSideEffect *fields
		want           *CommitteeChange
	}{
		{
			name: "Valid Input",
			fields: fields{
				shardCommonPool: []incognitokey.CommitteePublicKey{
					*incKey2,
				},
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey,
						*incKey5,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey6,
					},
				},
				numberOfAssignedCandidates: 1,
			},
			args: args{
				rand:            10000,
				activeShards:    2,
				committeeChange: NewCommitteeChange(),
			},
			wantSideEffect: &fields{
				shardCommonPool: []incognitokey.CommitteePublicKey{},
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey,
						*incKey5,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey6,
					},
					1: []incognitokey.CommitteePublicKey{
						*incKey2,
					},
				},
				numberOfAssignedCandidates: 0,
			},
			want: committeeChangeValidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV2{
				beaconCommittee:            tt.fields.beaconCommittee,
				shardCommittee:             tt.fields.shardCommittee,
				shardSubstitute:            tt.fields.shardSubstitute,
				shardCommonPool:            tt.fields.shardCommonPool,
				numberOfAssignedCandidates: tt.fields.numberOfAssignedCandidates,
				autoStake:                  tt.fields.autoStake,
				rewardReceiver:             tt.fields.rewardReceiver,
				stakingTx:                  tt.fields.stakingTx,
			}
			if got := b.processAssignWithRandomInstruction(tt.args.rand, tt.args.activeShards, tt.args.committeeChange); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BeaconCommitteeStateV2.processAssignWithRandomInstruction() = %v, want %v", got, tt.want)
			}
			if b.numberOfAssignedCandidates != tt.wantSideEffect.numberOfAssignedCandidates {
				t.Errorf("BeaconCommitteeStateV2.processAssignWithRandomInstruction(), numberOfAssignedCandidates = %v, want %v", b.numberOfAssignedCandidates, tt.wantSideEffect.numberOfAssignedCandidates)
			}
			for shardID, gotV := range b.shardSubstitute {
				wantV := tt.wantSideEffect.shardSubstitute[shardID]
				if !reflect.DeepEqual(gotV, wantV) {
					t.Errorf("BeaconCommitteeStateV2.processAssignWithRandomInstruction(), shardSubstitute = %v, want %v", gotV, wantV)
				}
			}
			for shardID, gotV := range b.shardCommittee {
				wantV := tt.wantSideEffect.shardCommittee[shardID]
				if !reflect.DeepEqual(gotV, wantV) {
					t.Errorf("BeaconCommitteeStateV2.processAssignWithRandomInstruction(), shardSubstitute = %v, want %v", gotV, wantV)
				}
			}
		})
	}
}

func TestSnapshotShardCommonPoolV2(t *testing.T) {

	initPublicKey()
	initLog()

	type args struct {
		shardCommonPool        []incognitokey.CommitteePublicKey
		shardCommittee         map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute        map[byte][]incognitokey.CommitteePublicKey
		numberOfFixedValidator int
		minCommitteeSize       int
	}
	tests := []struct {
		name                           string
		args                           args
		wantNumberOfAssignedCandidates int
	}{
		{
			name: "number of assigned candidates < number of committee in shard pool",
			args: args{
				shardCommonPool: []incognitokey.CommitteePublicKey{
					*incKey8, *incKey9, *incKey10,
				},
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey0, *incKey2, *incKey3,
					},
					1: []incognitokey.CommitteePublicKey{
						*incKey4, *incKey5, *incKey6, *incKey7,
					},
				},
				shardSubstitute:        map[byte][]incognitokey.CommitteePublicKey{},
				numberOfFixedValidator: 1,
				minCommitteeSize:       3,
			},
			wantNumberOfAssignedCandidates: 2,
		},
		{
			name: "number of assigned candidates > number of committee in shard pool",
			args: args{
				shardCommonPool: []incognitokey.CommitteePublicKey{
					*incKey8, *incKey9, *incKey10,
				},
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey0, *incKey2, *incKey3, *incKey11, *incKey12,
					},
					1: []incognitokey.CommitteePublicKey{
						*incKey4, *incKey5, *incKey6, *incKey7, *incKey13, *incKey14,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey15, *incKey16,
					},
					1: []incognitokey.CommitteePublicKey{
						*incKey17, *incKey18,
					},
				},
				numberOfFixedValidator: 4,
				minCommitteeSize:       6,
			},
			wantNumberOfAssignedCandidates: 3,
		},
		{
			name: "First time assign candidates",
			args: args{
				shardCommonPool: []incognitokey.CommitteePublicKey{
					*incKey4,
				},
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey0, *incKey2, *incKey3,
					},
					1: []incognitokey.CommitteePublicKey{
						*incKey, *incKey0, *incKey2, *incKey3,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
					1: []incognitokey.CommitteePublicKey{},
				},
				numberOfFixedValidator: 0,
				minCommitteeSize:       4,
			},
			wantNumberOfAssignedCandidates: 1,
		},
		{
			name: "assign 0 candidates",
			args: args{
				shardCommonPool: []incognitokey.CommitteePublicKey{},
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey0, *incKey2, *incKey3,
					},
					1: []incognitokey.CommitteePublicKey{
						*incKey, *incKey0, *incKey2, *incKey3,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
					1: []incognitokey.CommitteePublicKey{},
				},
				numberOfFixedValidator: 0,
				minCommitteeSize:       4,
			},
			wantNumberOfAssignedCandidates: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotNumberOfAssignedCandidates := SnapshotShardCommonPoolV2(tt.args.shardCommonPool, tt.args.shardCommittee, tt.args.shardSubstitute, tt.args.numberOfFixedValidator, tt.args.minCommitteeSize); gotNumberOfAssignedCandidates != tt.wantNumberOfAssignedCandidates {
				t.Errorf("SnapshotShardCommonPoolV2() = %v, want %v", gotNumberOfAssignedCandidates, tt.wantNumberOfAssignedCandidates)
			}
		})
	}
}

func TestBeaconCommitteeEngineV2_GenerateAllSwapShardInstructions(t *testing.T) {

	initPublicKey()
	initLog()

	type fields struct {
		beaconHeight                      uint64
		beaconHash                        common.Hash
		finalBeaconCommitteeStateV2       *BeaconCommitteeStateV2
		uncommittedBeaconCommitteeStateV2 *BeaconCommitteeStateV2
	}
	type args struct {
		env *BeaconCommitteeStateEnvironment
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*instruction.SwapShardInstruction
		wantErr bool
	}{
		{
			name: "len(subtitutes) == len(committeess) == 0",
			fields: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{},
					},
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					NumberOfFixedShardBlockValidators: 0,
					ActiveShards:                      2,
				},
			},
			want:    []*instruction.SwapShardInstruction{},
			wantErr: false,
		},
		{
			name: "Valid Input",
			fields: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey6, *incKey7, *incKey8, *incKey9,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey10,
						},
					},
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					NumberOfFixedShardBlockValidators: 0,
					ActiveShards:                      2,
					MaxShardCommitteeSize:             4,
				},
			},
			want: []*instruction.SwapShardInstruction{
				&instruction.SwapShardInstruction{
					InPublicKeys: []string{
						key5,
					},
					InPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
					OutPublicKeys: []string{
						key,
					},
					OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey,
					},
					ChainID: 0,
					Type:    instruction.SWAP_BY_END_EPOCH,
				},
				&instruction.SwapShardInstruction{
					InPublicKeys: []string{
						key10,
					},
					InPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey10,
					},
					OutPublicKeys: []string{
						key6,
					},
					OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey6,
					},
					ChainID: 1,
					Type:    instruction.SWAP_BY_END_EPOCH,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &BeaconCommitteeEngineV2{
				beaconHeight:                      tt.fields.beaconHeight,
				beaconHash:                        tt.fields.beaconHash,
				finalBeaconCommitteeStateV2:       tt.fields.finalBeaconCommitteeStateV2,
				uncommittedBeaconCommitteeStateV2: tt.fields.uncommittedBeaconCommitteeStateV2,
			}
			got, err := engine.GenerateAllSwapShardInstructions(tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeaconCommitteeEngineV2.GenerateAllRequestShardSwapInstruction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i, v := range got {
				if !reflect.DeepEqual(*v, *tt.want[i]) {
					t.Errorf("*v = %v, want %v", *v, *tt.want[i])
					return
				}
			}
		})
	}
}

func TestBeaconCommitteeStateV2_processSwapShardInstruction(t *testing.T) {

	initPublicKey()
	initLog()
	initStateDB()

	sDB, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)

	paymentAddress := privacy.GeneratePaymentAddress([]byte{1})

	hash, _ := common.Hash{}.NewHashFromStr("123")
	hash6, _ := common.Hash{}.NewHashFromStr("456")
	statedb.StoreStakerInfoV1(
		sDB,
		[]incognitokey.CommitteePublicKey{*incKey, *incKey6},
		map[string]privacy.PaymentAddress{
			incKey.GetIncKeyBase58():  paymentAddress,
			incKey6.GetIncKeyBase58(): paymentAddress,
		},
		map[string]bool{
			key:  true,
			key6: false,
		},
		map[string]common.Hash{
			key:  *hash,
			key6: *hash6,
		},
	)

	rootHash, _ := sDB.Commit(true)
	sDB.Database().TrieDB().Commit(rootHash, false)

	type fields struct {
		beaconCommittee            []incognitokey.CommitteePublicKey
		shardCommittee             map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
		shardCommonPool            []incognitokey.CommitteePublicKey
		numberOfAssignedCandidates int
		autoStake                  map[string]bool
		rewardReceiver             map[string]privacy.PaymentAddress
		stakingTx                  map[string]common.Hash
	}

	type args struct {
		swapShardInstruction      *instruction.SwapShardInstruction
		returnStakingInstructions map[byte]*instruction.ReturnStakeInstruction
		env                       *BeaconCommitteeStateEnvironment
		committeeChange           *CommitteeChange
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want1   *CommitteeChange
		want2   map[byte]*instruction.ReturnStakeInstruction
		wantErr bool
	}{
		{
			name: "Swap Out Not Valid In List Committees Public Key",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				autoStake:      map[string]bool{},
				rewardReceiver: map[string]privacy.PaymentAddress{},
				stakingTx:      map[string]common.Hash{},
			},
			args: args{
				swapShardInstruction: &instruction.SwapShardInstruction{
					OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey6,
					},
				},
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB:                  sDB,
					NumberOfFixedShardBlockValidators: 0,
				},
				committeeChange:           NewCommitteeChange(),
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1:   nil,
			want2:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: true,
		},
		{
			name: "Swap In Not Valid In List Substitutes Public Key",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				autoStake:      map[string]bool{},
				rewardReceiver: map[string]privacy.PaymentAddress{},
				stakingTx:      map[string]common.Hash{},
			},
			args: args{
				swapShardInstruction: &instruction.SwapShardInstruction{
					InPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey6,
					},
				},
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB:                  sDB,
					NumberOfFixedShardBlockValidators: 0,
				},
				committeeChange:           NewCommitteeChange(),
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1:   nil,
			want2:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: true,
		},
		{
			name: "Swap Out But Not found In Committee List",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey7, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				autoStake:      map[string]bool{},
				rewardReceiver: map[string]privacy.PaymentAddress{},
				stakingTx:      map[string]common.Hash{},
			},
			args: args{
				swapShardInstruction: instruction.NewSwapShardInstructionWithValue(
					[]string{key5},
					[]string{key7},
					0,
					instruction.SWAP_BY_END_EPOCH,
				),
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB:                  sDB,
					NumberOfFixedShardBlockValidators: 0,
				},
				committeeChange:           NewCommitteeChange(),
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1:   nil,
			want2:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: true,
		},
		{
			name: "Valid Input [Back To Common Pool And Re-assign]",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				autoStake:      map[string]bool{},
				rewardReceiver: map[string]privacy.PaymentAddress{},
				stakingTx:      map[string]common.Hash{},
			},
			args: args{
				swapShardInstruction: &instruction.SwapShardInstruction{
					OutPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey,
					},
					InPublicKeyStructs: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
					OutPublicKeys: []string{key},
				},
				env: &BeaconCommitteeStateEnvironment{
					NumberOfFixedShardBlockValidators: 0,
					ConsensusStateDB:                  sDB,
					RandomNumber:                      5000,
					ActiveShards:                      1,
				},
				committeeChange: &CommitteeChange{
					ShardSubstituteAdded:   map[byte][]incognitokey.CommitteePublicKey{},
					ShardSubstituteRemoved: map[byte][]incognitokey.CommitteePublicKey{},
					ShardCommitteeAdded:    map[byte][]incognitokey.CommitteePublicKey{},
					ShardCommitteeRemoved:  map[byte][]incognitokey.CommitteePublicKey{},
				},
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1: &CommitteeChange{
				ShardSubstituteAdded: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey,
					},
				},
				ShardSubstituteRemoved: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				ShardCommitteeAdded: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				ShardCommitteeRemoved: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey,
					},
				},
			},
			want2:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: false,
		},
		{
			name: "Valid Input [Swap Out]",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey6, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				autoStake:      map[string]bool{},
				rewardReceiver: map[string]privacy.PaymentAddress{},
				stakingTx:      map[string]common.Hash{},
			},
			args: args{
				swapShardInstruction: instruction.NewSwapShardInstructionWithValue(
					[]string{key5},
					[]string{key6},
					0,
					instruction.SWAP_BY_END_EPOCH,
				),
				env: &BeaconCommitteeStateEnvironment{
					NumberOfFixedShardBlockValidators: 0,
					ConsensusStateDB:                  sDB,
					RandomNumber:                      5000,
					ActiveShards:                      1,
				},
				committeeChange: &CommitteeChange{
					ShardSubstituteAdded:   map[byte][]incognitokey.CommitteePublicKey{},
					ShardSubstituteRemoved: map[byte][]incognitokey.CommitteePublicKey{},
					ShardCommitteeAdded:    map[byte][]incognitokey.CommitteePublicKey{},
					ShardCommitteeRemoved:  map[byte][]incognitokey.CommitteePublicKey{},
				},
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1: &CommitteeChange{
				ShardSubstituteAdded: map[byte][]incognitokey.CommitteePublicKey{},
				ShardSubstituteRemoved: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				ShardCommitteeAdded: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey5,
					},
				},
				ShardCommitteeRemoved: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey6,
					},
				},
			},
			want2: map[byte]*instruction.ReturnStakeInstruction{
				0: instruction.NewReturnStakeInsWithValue(
					[]string{key6},
					0,
					[]string{hash6.String()},
				),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV2{
				beaconCommittee:            tt.fields.beaconCommittee,
				shardCommittee:             tt.fields.shardCommittee,
				shardSubstitute:            tt.fields.shardSubstitute,
				shardCommonPool:            tt.fields.shardCommonPool,
				numberOfAssignedCandidates: tt.fields.numberOfAssignedCandidates,
				autoStake:                  tt.fields.autoStake,
				rewardReceiver:             tt.fields.rewardReceiver,
				stakingTx:                  tt.fields.stakingTx,
			}
			got1, got2, err := b.processSwapShardInstruction(tt.args.swapShardInstruction, tt.args.env, tt.args.committeeChange, tt.args.returnStakingInstructions)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeaconCommitteeStateV2.processSwapShardInstruction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("BeaconCommitteeStateV2.processSwapShardInstruction() = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("BeaconCommitteeStateV2.processSwapShardInstruction() = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestBeaconCommitteeEngineV2_UpdateCommitteeState(t *testing.T) {
	hash, _ := common.Hash{}.NewHashFromStr("123")

	initPublicKey()
	initStateDB()
	initLog()

	sDB, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)
	paymentAddress0, err := wallet.Base58CheckDeserialize(paymentAddreessKey0)
	assert.Nil(t, err)
	rewardReceiverkey0 := incKey0.GetIncKeyBase58()

	committeeChangeProcessStakeInstruction := NewCommitteeChange()
	committeeChangeProcessStakeInstruction.NextEpochShardCandidateAdded = []incognitokey.CommitteePublicKey{
		*incKey0,
	}

	committeeChangeProcessRandomInstruction := NewCommitteeChange()
	committeeChangeProcessRandomInstruction.NextEpochShardCandidateRemoved = []incognitokey.CommitteePublicKey{
		*incKey6,
	}
	committeeChangeProcessRandomInstruction.ShardSubstituteAdded[0] = []incognitokey.CommitteePublicKey{
		*incKey6,
	}

	committeeChangeProcessStopAutoStakeInstruction := NewCommitteeChange()
	committeeChangeProcessStopAutoStakeInstruction.StopAutoStake = []string{key5}

	committeeChangeProcessSwapShardInstruction := NewCommitteeChange()
	committeeChangeProcessSwapShardInstruction.ShardCommitteeAdded[0] = []incognitokey.CommitteePublicKey{
		*incKey4,
	}
	committeeChangeProcessSwapShardInstruction.ShardSubstituteAdded[0] = []incognitokey.CommitteePublicKey{
		*incKey0,
	}
	committeeChangeProcessSwapShardInstruction.ShardSubstituteRemoved[0] = []incognitokey.CommitteePublicKey{
		*incKey4,
	}
	committeeChangeProcessSwapShardInstruction.ShardCommitteeRemoved[0] = []incognitokey.CommitteePublicKey{
		*incKey0,
	}

	committeeChangeProcessUnstakeInstruction := NewCommitteeChange()
	committeeChangeProcessUnstakeInstruction.NextEpochShardCandidateRemoved = []incognitokey.CommitteePublicKey{*incKey0}
	committeeChangeProcessUnstakeInstruction.Unstake = []string{key0}

	statedb.StoreStakerInfoV1(
		sDB,
		[]incognitokey.CommitteePublicKey{*incKey0},
		map[string]privacy.PaymentAddress{
			rewardReceiverkey0: paymentAddress0.KeySet.PaymentAddress,
		},
		map[string]bool{
			key0: true,
		},
		map[string]common.Hash{
			key0: *hash,
		},
	)

	finalMu := &sync.RWMutex{}
	unCommitteedMu := &sync.RWMutex{}

	type fields struct {
		beaconHeight                      uint64
		beaconHash                        common.Hash
		finalBeaconCommitteeStateV2       *BeaconCommitteeStateV2
		uncommittedBeaconCommitteeStateV2 *BeaconCommitteeStateV2
		version                           uint
	}
	type args struct {
		env *BeaconCommitteeStateEnvironment
	}
	tests := []struct {
		name               string
		fields             fields
		fieldsAfterProcess fields
		args               args
		want               *BeaconCommitteeStateHash
		want1              *CommitteeChange
		want2              [][]string
		wantErr            bool
	}{
		{
			name: "Process Stake Instruction",
			fields: fields{
				beaconHash:   *hash,
				version:      SLASHING_VERSION,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey5,
						},
					},
					autoStake:      map[string]bool{},
					rewardReceiver: map[string]privacy.PaymentAddress{},
					stakingTx:      map[string]common.Hash{},
					mu:             finalMu,
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					mu: unCommitteedMu,
				},
			},
			fieldsAfterProcess: fields{
				beaconHash:                  *hash,
				version:                     SLASHING_VERSION,
				beaconHeight:                10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey5,
						},
					},
					shardCommonPool: []incognitokey.CommitteePublicKey{
						*incKey0,
					},
					autoStake: map[string]bool{
						key0: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						incKey0.GetIncKeyBase58(): paymentAddress0.KeySet.PaymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key0: *hash,
					},
					mu: unCommitteedMu,
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{
							instruction.STAKE_ACTION,
							key0,
							instruction.SHARD_INST,
							hash.String(),
							paymentAddreessKey0,
							"true",
						},
					},
					ConsensusStateDB: sDB,
				},
			},
			want:    &BeaconCommitteeStateHash{},
			want1:   committeeChangeProcessStakeInstruction,
			want2:   [][]string{},
			wantErr: false,
		},
		{
			name: "Process Random Instruction",
			fields: fields{
				beaconHash:   *hash,
				version:      SLASHING_VERSION,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{},
					},
					shardCommonPool: []incognitokey.CommitteePublicKey{
						*incKey6,
					},
					mu:                         finalMu,
					autoStake:                  map[string]bool{},
					rewardReceiver:             map[string]privacy.PaymentAddress{},
					stakingTx:                  map[string]common.Hash{},
					numberOfAssignedCandidates: 1,
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					mu: unCommitteedMu,
				},
			},
			fieldsAfterProcess: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					numberOfAssignedCandidates: 0,
					beaconCommittee:            []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey6,
						},
					},
					shardCommonPool: []incognitokey.CommitteePublicKey{},
					mu:              unCommitteedMu,
					autoStake:       map[string]bool{},
					rewardReceiver:  map[string]privacy.PaymentAddress{},
					stakingTx:       map[string]common.Hash{},
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{
							instruction.RANDOM_ACTION,
							"800000",
							"120000",
							"350000",
							"190000",
						},
					},
					ActiveShards:          1,
					BeaconHeight:          100,
					MaxShardCommitteeSize: 5,
				},
			},
			want:    &BeaconCommitteeStateHash{},
			want1:   committeeChangeProcessRandomInstruction,
			want2:   [][]string{},
			wantErr: false,
		},
		{
			name: "Process Stop Auto Stake Instruction",
			fields: fields{
				beaconHash:   *hash,
				version:      SLASHING_VERSION,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey5,
						},
					},
					shardCommonPool: []incognitokey.CommitteePublicKey{},
					mu:              finalMu,
					autoStake: map[string]bool{
						key5: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						key5: paymentAddress0.KeySet.PaymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key5: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					mu: unCommitteedMu,
				},
			},
			fieldsAfterProcess: fields{
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey5,
						},
					},
					shardCommonPool: []incognitokey.CommitteePublicKey{},
					mu:              unCommitteedMu,
					autoStake: map[string]bool{
						key5: false,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						key5: paymentAddress0.KeySet.PaymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key5: *hash,
					},
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{
							instruction.STOP_AUTO_STAKE_ACTION,
							key5,
						},
					},
					ConsensusStateDB: sDB,
				},
			},
			want:    &BeaconCommitteeStateHash{},
			want1:   committeeChangeProcessStopAutoStakeInstruction,
			want2:   [][]string{},
			wantErr: false,
		},
		{
			name: "Process Swap Shard Instructions",
			fields: fields{
				beaconHeight: 5,
				beaconHash:   *hash,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommonPool: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey4,
						},
					},
					mu: finalMu,
					autoStake: map[string]bool{
						key0: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						key0: paymentAddress0.KeySet.PaymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key0: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					mu: unCommitteedMu,
				},
				version: SLASHING_VERSION,
			},
			fieldsAfterProcess: fields{
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommonPool: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0,
						},
					},
					autoStake: map[string]bool{
						key0: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						key0: paymentAddress0.KeySet.PaymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key0: *hash,
					},
					mu: unCommitteedMu,
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{
							instruction.SWAP_SHARD_ACTION,
							key4,
							key0,
							"0",
							"1",
						},
					},
					ActiveShards:     1,
					ConsensusStateDB: sDB,
					RandomNumber:     5000,
				},
			},
			want:    &BeaconCommitteeStateHash{},
			want1:   committeeChangeProcessSwapShardInstruction,
			want2:   [][]string{},
			wantErr: false,
		},
		{
			name: "Process Unstake Instruction",
			fields: fields{
				beaconHash:   *hash,
				version:      SLASHING_VERSION,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey5,
						},
					},
					shardCommonPool: []incognitokey.CommitteePublicKey{
						*incKey0,
					},
					mu: finalMu,
					autoStake: map[string]bool{
						key0: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						incKey0.GetIncKeyBase58(): paymentAddress0.KeySet.PaymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key0: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					mu: unCommitteedMu,
				},
			},
			fieldsAfterProcess: fields{
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey, *incKey2, *incKey3, *incKey4,
						},
					},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey5,
						},
					},
					shardCommonPool: make([]incognitokey.CommitteePublicKey, 0, 1),
					autoStake:       map[string]bool{},
					rewardReceiver:  map[string]privacy.PaymentAddress{},
					stakingTx:       map[string]common.Hash{},
					mu:              unCommitteedMu,
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{
							instruction.UNSTAKE_ACTION,
							key0,
						},
					},
					unassignedCommonPool: []string{key0},
					ConsensusStateDB:     sDB,
				},
			},
			want:  &BeaconCommitteeStateHash{},
			want1: committeeChangeProcessUnstakeInstruction,
			want2: [][]string{
				[]string{
					instruction.RETURN_ACTION,
					key0,
					"0",
					hash.String(),
					"100",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &BeaconCommitteeEngineV2{
				beaconHeight:                      tt.fields.beaconHeight,
				beaconHash:                        tt.fields.beaconHash,
				finalBeaconCommitteeStateV2:       tt.fields.finalBeaconCommitteeStateV2,
				uncommittedBeaconCommitteeStateV2: tt.fields.uncommittedBeaconCommitteeStateV2,
			}
			_, got1, got2, err := engine.UpdateCommitteeState(tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeaconCommitteeEngineV2.UpdateCommitteeState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("BeaconCommitteeEngineV2.UpdateCommitteeState() got1 = %v, want1 = %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("BeaconCommitteeEngineV2.UpdateCommitteeState() got2 = %v, want2 = %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(tt.fields.uncommittedBeaconCommitteeStateV2,
				tt.fieldsAfterProcess.uncommittedBeaconCommitteeStateV2) {
				t.Errorf(`BeaconCommitteeEngineV2.UpdateCommitteeState() tt.fields.uncommittedBeaconCommitteeStateV2 = %v, 
					tt.fieldsAfterProcess.uncommittedBeaconCommitteeStateV2 = %v`,
					tt.fields.uncommittedBeaconCommitteeStateV2, tt.fieldsAfterProcess.uncommittedBeaconCommitteeStateV2)
			}
		})
	}
}

func TestBeaconCommitteeStateV2_processAfterSwap(t *testing.T) {

	initPublicKey()
	initLog()
	initStateDB()

	sDB, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)
	sDB2, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)
	sDB3, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)

	paymentAddress := privacy.GeneratePaymentAddress([]byte{1})

	hash, err := common.Hash{}.NewHashFromStr("123")
	hash6, err := common.Hash{}.NewHashFromStr("456")
	statedb.StoreStakerInfoV1(
		sDB,
		[]incognitokey.CommitteePublicKey{*incKey, *incKey6},
		map[string]privacy.PaymentAddress{
			incKey.GetIncKeyBase58():  paymentAddress,
			incKey6.GetIncKeyBase58(): paymentAddress,
		},
		map[string]bool{
			key:  true,
			key6: false,
		},
		map[string]common.Hash{
			key:  *hash,
			key6: *hash6,
		},
	)
	statedb.StoreStakerInfoV1(
		sDB2,
		[]incognitokey.CommitteePublicKey{*incKey, *incKey6},
		map[string]privacy.PaymentAddress{
			incKey.GetIncKeyBase58():  paymentAddress,
			incKey6.GetIncKeyBase58(): paymentAddress,
		},
		map[string]bool{
			key:  false,
			key6: false,
		},
		map[string]common.Hash{
			key:  *hash,
			key6: *hash6,
		},
	)
	statedb.StoreStakerInfoV1(
		sDB3,
		[]incognitokey.CommitteePublicKey{*incKey, *incKey6},
		map[string]privacy.PaymentAddress{
			incKey.GetIncKeyBase58():  paymentAddress,
			incKey6.GetIncKeyBase58(): paymentAddress,
		},
		map[string]bool{
			key:  false,
			key6: false,
		},
		map[string]common.Hash{
			key:  *hash,
			key6: *hash6,
		},
	)

	rootHash, _ := sDB.Commit(true)
	sDB.Database().TrieDB().Commit(rootHash, false)

	rootHash2, _ := sDB2.Commit(true)
	sDB2.Database().TrieDB().Commit(rootHash2, false)

	rootHash3, _ := sDB3.Commit(true)
	sDB3.Database().TrieDB().Commit(rootHash3, false)

	type fields struct {
		beaconCommittee            []incognitokey.CommitteePublicKey
		shardCommittee             map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
		shardCommonPool            []incognitokey.CommitteePublicKey
		numberOfAssignedCandidates int
		autoStake                  map[string]bool
		rewardReceiver             map[string]privacy.PaymentAddress
		stakingTx                  map[string]common.Hash
		mu                         *sync.RWMutex
	}
	type args struct {
		env                     *BeaconCommitteeStateEnvironment
		outPublicKeys           []string
		outPublicKeyStructs     []incognitokey.CommitteePublicKey
		shardID                 byte
		committeeChange         *CommitteeChange
		returnStakeInstructions map[byte]*instruction.ReturnStakeInstruction
	}
	tests := []struct {
		name               string
		fields             fields
		fieldsAfterProcess fields
		args               args
		want1              *CommitteeChange
		want2              map[byte]*instruction.ReturnStakeInstruction
		wantErr            bool
	}{
		{
			name: "[Back To Substitute] Not Found Staker Info",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
				},
				autoStake: map[string]bool{
					key:  true,
					key8: false,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key:  *hash,
					key6: *hash6,
				},
			},
			fieldsAfterProcess: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
				},
				autoStake: map[string]bool{
					key:  true,
					key8: false,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key:  *hash,
					key6: *hash6,
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB: sDB,
					ActiveShards:     1,
					RandomNumber:     10000,
				},
				outPublicKeyStructs:     []incognitokey.CommitteePublicKey{*incKey5},
				outPublicKeys:           []string{key5},
				shardID:                 0,
				committeeChange:         &CommitteeChange{},
				returnStakeInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1:   &CommitteeChange{},
			want2:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: true,
		},
		{
			name: "[Swap Out] Return Staking Amount",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey2, *incKey3, *incKey4, *incKey5,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
				},
				autoStake: map[string]bool{
					key:  true,
					key5: true,
					key8: true,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key:  *hash,
					key5: *hash6,
					key6: *hash6,
				},
			},
			fieldsAfterProcess: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey2, *incKey3, *incKey4, *incKey5,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
				},
				autoStake: map[string]bool{
					key5: true,
					key8: true,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{},
				stakingTx: map[string]common.Hash{
					key5: *hash6,
					key6: *hash6,
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB: sDB2,
					ActiveShards:     1,
					RandomNumber:     10000,
				},
				outPublicKeyStructs:     []incognitokey.CommitteePublicKey{*incKey},
				outPublicKeys:           []string{key},
				shardID:                 0,
				committeeChange:         &CommitteeChange{},
				returnStakeInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1: &CommitteeChange{},
			want2: map[byte]*instruction.ReturnStakeInstruction{
				0: instruction.NewReturnStakeInsWithValue(
					[]string{key},
					0,
					[]string{hash.String()},
				),
			},
			wantErr: false,
		},
		{
			name: "[Swap Out] Not Found Staker Info",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
				},
				autoStake: map[string]bool{
					key:  true,
					key8: false,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key: *hash,
				},
			},
			fieldsAfterProcess: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey, *incKey2, *incKey3, *incKey4,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
				},
				autoStake: map[string]bool{
					key:  true,
					key8: false,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key: *hash,
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB: sDB,
					ActiveShards:     1,
					RandomNumber:     10000,
				},
				outPublicKeyStructs:     []incognitokey.CommitteePublicKey{*incKey5},
				outPublicKeys:           []string{key5},
				shardID:                 0,
				committeeChange:         &CommitteeChange{},
				returnStakeInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1:   &CommitteeChange{},
			want2:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: true,
		},
		{
			name: "[Back To Substitute] Valid Input",
			fields: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey2, *incKey3, *incKey4, *incKey5,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{},
				},
				autoStake: map[string]bool{
					key:  true,
					key8: false,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key:  *hash,
					key6: *hash6,
				},
			},
			fieldsAfterProcess: fields{
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey2, *incKey3, *incKey4, *incKey5,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey,
					},
				},
				autoStake: map[string]bool{
					key:  true,
					key8: false,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key:  *hash,
					key6: *hash6,
				},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB: sDB,
					ActiveShards:     1,
					RandomNumber:     10000,
				},
				outPublicKeyStructs: []incognitokey.CommitteePublicKey{*incKey},
				outPublicKeys:       []string{key},
				shardID:             0,
				committeeChange: &CommitteeChange{
					ShardSubstituteAdded: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{},
					},
				},
				returnStakeInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want1: &CommitteeChange{
				ShardSubstituteAdded: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey,
					},
				},
			},
			want2:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV2{
				beaconCommittee:            tt.fields.beaconCommittee,
				shardCommittee:             tt.fields.shardCommittee,
				shardSubstitute:            tt.fields.shardSubstitute,
				shardCommonPool:            tt.fields.shardCommonPool,
				numberOfAssignedCandidates: tt.fields.numberOfAssignedCandidates,
				autoStake:                  tt.fields.autoStake,
				rewardReceiver:             tt.fields.rewardReceiver,
				stakingTx:                  tt.fields.stakingTx,
				mu:                         tt.fields.mu,
			}
			got1, got2, err := b.processAfterSwap(tt.args.env, tt.args.outPublicKeys, tt.args.outPublicKeyStructs, tt.args.shardID, tt.args.committeeChange, tt.args.returnStakeInstructions)
			if (err != nil) != tt.wantErr {
				t.Errorf("processAfterSwap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.fields, tt.fieldsAfterProcess) {
				t.Errorf("processAfterSwap() tt.fields = %v, tt.fieldsAfterProcess %v", tt.fields, tt.fieldsAfterProcess)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("processAfterSwap() got1 = %v, want1 %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("processAfterSwap() got2 = %v, want2 %v", got2, tt.want2)
			}
		})
	}
}

func TestBeaconCommitteeEngineV2_BuildIncurredInstructions(t *testing.T) {

	initPublicKey()
	initStateDB()
	initLog()

	sDB, _ := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	rewardReceiverkey := incKey.GetIncKeyBase58()
	paymentAddress := privacy.GeneratePaymentAddress([]byte{1})

	hash, _ := common.Hash{}.NewHashFromStr("123")
	err := statedb.StoreStakerInfoV1(
		sDB,
		[]incognitokey.CommitteePublicKey{*incKey},
		map[string]privacy.PaymentAddress{
			rewardReceiverkey: paymentAddress,
		},
		map[string]bool{
			key: true,
		},
		map[string]common.Hash{
			key: *hash,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	type fields struct {
		beaconHeight                      uint64
		beaconHash                        common.Hash
		finalBeaconCommitteeStateV2       *BeaconCommitteeStateV2
		uncommittedBeaconCommitteeStateV2 *BeaconCommitteeStateV2
	}
	type args struct {
		env *BeaconCommitteeStateEnvironment
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    [][]string
		wantErr bool
	}{
		{
			name: "Environment Is Null",
			fields: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					shardCommittee:  map[byte][]incognitokey.CommitteePublicKey{},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{},
					shardCommonPool: []incognitokey.CommitteePublicKey{*incKey},
					autoStake: map[string]bool{
						key: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						incKey.GetIncKeyBase58(): paymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args:    args{},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name: "Length Of Beacon Instructions Is 0",
			fields: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					shardCommittee:  map[byte][]incognitokey.CommitteePublicKey{},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{},
					shardCommonPool: []incognitokey.CommitteePublicKey{*incKey},
					autoStake: map[string]bool{
						key: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						incKey.GetIncKeyBase58(): paymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{},
				},
			},
			want:    [][]string{},
			wantErr: false,
		},

		{
			name: "Invalid Unstake Instruction Format",
			fields: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					shardCommittee:  map[byte][]incognitokey.CommitteePublicKey{},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{},
					shardCommonPool: []incognitokey.CommitteePublicKey{*incKey},
					autoStake: map[string]bool{
						key: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						incKey.GetIncKeyBase58(): paymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{instruction.UNSTAKE_ACTION},
					},
				},
			},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name: "Error In Processing Unstake Instruction",
			fields: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					shardCommittee:  map[byte][]incognitokey.CommitteePublicKey{},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{},
					shardCommonPool: []incognitokey.CommitteePublicKey{*incKey2},
					autoStake: map[string]bool{
						key2: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						incKey2.GetIncKeyBase58(): paymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key2: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{
							instruction.UNSTAKE_ACTION,
							key2,
						},
					},
					ConsensusStateDB:     sDB,
					unassignedCommonPool: []string{key2},
				},
			},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name: "Valid Input",
			fields: fields{
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					shardCommittee:  map[byte][]incognitokey.CommitteePublicKey{},
					shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{},
					shardCommonPool: []incognitokey.CommitteePublicKey{*incKey},
					autoStake: map[string]bool{
						key: true,
					},
					rewardReceiver: map[string]privacy.PaymentAddress{
						incKey.GetIncKeyBase58(): paymentAddress,
					},
					stakingTx: map[string]common.Hash{
						key: *hash,
					},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					BeaconInstructions: [][]string{
						[]string{
							instruction.UNSTAKE_ACTION,
							key,
						},
					},
					ConsensusStateDB:     sDB,
					unassignedCommonPool: []string{key},
				},
			},
			want: [][]string{
				[]string{
					instruction.RETURN_ACTION,
					key,
					"0",
					hash.String(),
					"100",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &BeaconCommitteeEngineV2{
				beaconHeight:                      tt.fields.beaconHeight,
				beaconHash:                        tt.fields.beaconHash,
				finalBeaconCommitteeStateV2:       tt.fields.finalBeaconCommitteeStateV2,
				uncommittedBeaconCommitteeStateV2: tt.fields.uncommittedBeaconCommitteeStateV2,
			}
			got, err := engine.BuildIncurredInstructions(tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildIncurredInstructions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildIncurredInstructions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBeaconCommitteeStateV2_processUnstakeInstruction(t *testing.T) {

	// Init data for testcases
	initStateDB()
	initPublicKey()
	initLog()

	rewardReceiverkey := incKey.GetIncKeyBase58()
	paymentAddress := privacy.GeneratePaymentAddress([]byte{1})

	validSDB, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)

	hash, err := common.Hash{}.NewHashFromStr("123")
	statedb.StoreStakerInfoV1(
		validSDB,
		[]incognitokey.CommitteePublicKey{*incKey},
		map[string]privacy.PaymentAddress{
			rewardReceiverkey: paymentAddress,
		},
		map[string]bool{
			key: true,
		},
		map[string]common.Hash{
			key: *hash,
		},
	)
	committeePublicKeyWrongFormat := incognitokey.CommitteePublicKey{}
	committeePublicKeyWrongFormat.MiningPubKey = nil

	type fields struct {
		beaconCommittee            []incognitokey.CommitteePublicKey
		shardCommittee             map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
		shardCommonPool            []incognitokey.CommitteePublicKey
		numberOfAssignedCandidates int
		autoStake                  map[string]bool
		rewardReceiver             map[string]privacy.PaymentAddress
		stakingTx                  map[string]common.Hash
		mu                         *sync.RWMutex
	}
	type args struct {
		unstakeInstruction        *instruction.UnstakeInstruction
		returnStakingInstructions map[byte]*instruction.ReturnStakeInstruction
		env                       *BeaconCommitteeStateEnvironment
		committeeChange           *CommitteeChange
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *CommitteeChange
		want1   map[byte]*instruction.ReturnStakeInstruction
		wantErr bool
	}{
		{
			name: "[Subtitute List] Invalid Format Of Committee Public Key In Unstake Instruction",
			fields: fields{
				shardCommonPool:            []incognitokey.CommitteePublicKey{*incKey},
				numberOfAssignedCandidates: 0,
			},
			args: args{
				unstakeInstruction: &instruction.UnstakeInstruction{
					CommitteePublicKeys: []string{"123"},
				},
				env: &BeaconCommitteeStateEnvironment{
					unassignedCommonPool: []string{"123"},
				},
				committeeChange:           &CommitteeChange{},
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want:    &CommitteeChange{},
			want1:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: true,
		},
		{
			name: "[Subtitute List] Can't find staker info in database",
			fields: fields{
				shardCommonPool: []incognitokey.CommitteePublicKey{*incKey},
				autoStake: map[string]bool{
					key: true,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					rewardReceiverkey: paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key: *hash,
				},
			},
			args: args{
				unstakeInstruction: &instruction.UnstakeInstruction{
					CommitteePublicKeys:       []string{key2},
					CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey2},
				},
				env: &BeaconCommitteeStateEnvironment{
					unassignedCommonPool: []string{key2},
					ConsensusStateDB:     validSDB,
				},
				committeeChange:           &CommitteeChange{},
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want:    &CommitteeChange{},
			want1:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: true,
		},
		{
			name: "Valid Input Key In Candidates List",
			fields: fields{
				shardCommonPool: []incognitokey.CommitteePublicKey{*incKey},
				autoStake: map[string]bool{
					key: true,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					rewardReceiverkey: paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key: *hash,
				},
			},
			args: args{
				committeeChange: &CommitteeChange{},
				unstakeInstruction: &instruction.UnstakeInstruction{
					CommitteePublicKeys:       []string{key},
					CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey},
				},
				env: &BeaconCommitteeStateEnvironment{
					unassignedCommonPool: []string{key},
					ConsensusStateDB:     validSDB,
				},
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want: &CommitteeChange{
				NextEpochShardCandidateRemoved: []incognitokey.CommitteePublicKey{*incKey},
				Unstake:                        []string{key},
			},
			want1: map[byte]*instruction.ReturnStakeInstruction{
				0: instruction.NewReturnStakeInsWithValue(
					[]string{key},
					0,
					[]string{hash.String()},
				),
			},
			wantErr: false,
		},
		{
			name: "Valid Input Key In Validators List",
			fields: fields{
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{*incKey},
				},
				autoStake: map[string]bool{
					key: true,
				},
			},
			args: args{
				unstakeInstruction: &instruction.UnstakeInstruction{
					CommitteePublicKeys:       []string{key},
					CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey},
				},
				env: &BeaconCommitteeStateEnvironment{
					allSubstituteCommittees: []string{key},
				},
				committeeChange:           &CommitteeChange{},
				returnStakingInstructions: make(map[byte]*instruction.ReturnStakeInstruction),
			},
			want: &CommitteeChange{
				StopAutoStake: []string{key},
			},
			want1:   make(map[byte]*instruction.ReturnStakeInstruction),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV2{
				beaconCommittee:            tt.fields.beaconCommittee,
				shardCommittee:             tt.fields.shardCommittee,
				shardSubstitute:            tt.fields.shardSubstitute,
				shardCommonPool:            tt.fields.shardCommonPool,
				numberOfAssignedCandidates: tt.fields.numberOfAssignedCandidates,
				autoStake:                  tt.fields.autoStake,
				rewardReceiver:             tt.fields.rewardReceiver,
				stakingTx:                  tt.fields.stakingTx,
				mu:                         tt.fields.mu,
			}
			got, got1, err := b.processUnstakeInstruction(tt.args.unstakeInstruction, tt.args.env, tt.args.committeeChange, tt.args.returnStakingInstructions)
			if (err != nil) != tt.wantErr {
				t.Errorf("processUnstakeInstruction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processUnstakeInstruction() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("processUnstakeInstruction() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestBeaconCommitteeStateV2_processStopAutoStakeInstruction(t *testing.T) {

	initPublicKey()

	type fields struct {
		beaconCommittee            []incognitokey.CommitteePublicKey
		shardCommittee             map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
		shardCommonPool            []incognitokey.CommitteePublicKey
		numberOfAssignedCandidates int
		autoStake                  map[string]bool
		rewardReceiver             map[string]privacy.PaymentAddress
		stakingTx                  map[string]common.Hash
		mu                         *sync.RWMutex
	}
	type args struct {
		stopAutoStakeInstruction *instruction.StopAutoStakeInstruction
		env                      *BeaconCommitteeStateEnvironment
		committeeChange          *CommitteeChange
	}
	tests := []struct {
		name               string
		fields             fields
		fieldsAfterProcess fields
		args               args
		want               *CommitteeChange
	}{
		{
			name:               "Not Found In List Subtitutes",
			fields:             fields{},
			fieldsAfterProcess: fields{},
			args: args{
				stopAutoStakeInstruction: &instruction.StopAutoStakeInstruction{
					CommitteePublicKeys: []string{key},
				},
				env: &BeaconCommitteeStateEnvironment{
					allCandidateSubstituteCommittee: []string{key2},
				},
				committeeChange: &CommitteeChange{},
			},
			want: &CommitteeChange{},
		},
		{
			name: "Found In List Subtitutes",
			fields: fields{
				autoStake: map[string]bool{
					key: true,
				},
			},
			fieldsAfterProcess: fields{
				autoStake: map[string]bool{
					key: false,
				},
			},
			args: args{
				stopAutoStakeInstruction: &instruction.StopAutoStakeInstruction{
					CommitteePublicKeys: []string{key},
				},
				env: &BeaconCommitteeStateEnvironment{
					allCandidateSubstituteCommittee: []string{key},
				},
				committeeChange: &CommitteeChange{},
			},
			want: &CommitteeChange{
				StopAutoStake: []string{key},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV2{
				beaconCommittee:            tt.fields.beaconCommittee,
				shardCommittee:             tt.fields.shardCommittee,
				shardSubstitute:            tt.fields.shardSubstitute,
				shardCommonPool:            tt.fields.shardCommonPool,
				numberOfAssignedCandidates: tt.fields.numberOfAssignedCandidates,
				autoStake:                  tt.fields.autoStake,
				rewardReceiver:             tt.fields.rewardReceiver,
				stakingTx:                  tt.fields.stakingTx,
				mu:                         tt.fields.mu,
			}
			if got := b.processStopAutoStakeInstruction(tt.args.stopAutoStakeInstruction, tt.args.env, tt.args.committeeChange); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processStopAutoStakeInstruction() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.fields, tt.fieldsAfterProcess) {
				t.Errorf("processAfterSwap() tt.fields = %v, tt.fieldsAfterProcess %v", tt.fields, tt.fieldsAfterProcess)
			}
		})
	}
}

func TestBeaconCommitteeStateV2_clone(t *testing.T) {

	initPublicKey()
	initLog()
	initStateDB()

	paymentAddress := privacy.GeneratePaymentAddress([]byte{1})
	hash, _ := common.Hash{}.NewHashFromStr("123")
	hash6, _ := common.Hash{}.NewHashFromStr("456")

	type fields struct {
		beaconCommittee            []incognitokey.CommitteePublicKey
		shardCommittee             map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute            map[byte][]incognitokey.CommitteePublicKey
		shardCommonPool            []incognitokey.CommitteePublicKey
		numberOfAssignedCandidates int
		autoStake                  map[string]bool
		rewardReceiver             map[string]privacy.PaymentAddress
		stakingTx                  map[string]common.Hash
	}
	type args struct {
		newB *BeaconCommitteeStateV2
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "[valid input] full data",
			fields: fields{
				beaconCommittee: []incognitokey.CommitteePublicKey{
					*incKey6, *incKey7, *incKey8,
				},
				shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey3, *incKey4, *incKey5,
					},
				},
				shardSubstitute: map[byte][]incognitokey.CommitteePublicKey{
					0: []incognitokey.CommitteePublicKey{
						*incKey,
					},
				},
				shardCommonPool: []incognitokey.CommitteePublicKey{
					*incKey2,
				},
				autoStake: map[string]bool{
					key:  true,
					key8: false,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					incKey.GetIncKeyBase58(): paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key:  *hash,
					key6: *hash6,
				},
				numberOfAssignedCandidates: 1,
			},
			args: args{
				newB: NewBeaconCommitteeStateV2(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV2{
				beaconCommittee:            tt.fields.beaconCommittee,
				shardCommittee:             tt.fields.shardCommittee,
				shardSubstitute:            tt.fields.shardSubstitute,
				shardCommonPool:            tt.fields.shardCommonPool,
				numberOfAssignedCandidates: tt.fields.numberOfAssignedCandidates,
				autoStake:                  tt.fields.autoStake,
				rewardReceiver:             tt.fields.rewardReceiver,
				stakingTx:                  tt.fields.stakingTx,
			}
			tt.args.newB.mu = nil
			if b.clone(tt.args.newB); !reflect.DeepEqual(b, tt.args.newB) {
				t.Errorf("clone() = %v, \n"+
					"want %v", tt.args.newB, b)
			}
		})
	}
}
