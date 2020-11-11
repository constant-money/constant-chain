package committeestate

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
)

func TestBeaconCommitteeStateV1_processUnstakeChange(t *testing.T) {
	initStateDB()
	initPublicKey()

	sDB, err := statedb.NewWithPrefixTrie(emptyRoot, wrarperDB)
	assert.Nil(t, err)
	rewardReceiver := incKey.GetIncKeyBase58()
	paymentAddress := privacy.GeneratePaymentAddress([]byte{1})
	hash, err := common.Hash{}.NewHashFromStr("123")

	type fields struct {
		beaconCommittee             []incognitokey.CommitteePublicKey
		beaconSubstitute            []incognitokey.CommitteePublicKey
		nextEpochShardCandidate     []incognitokey.CommitteePublicKey
		currentEpochShardCandidate  []incognitokey.CommitteePublicKey
		nextEpochBeaconCandidate    []incognitokey.CommitteePublicKey
		currentEpochBeaconCandidate []incognitokey.CommitteePublicKey
		shardCommittee              map[byte][]incognitokey.CommitteePublicKey
		shardSubstitute             map[byte][]incognitokey.CommitteePublicKey
		autoStake                   map[string]bool
		rewardReceiver              map[string]privacy.PaymentAddress
		stakingTx                   map[string]common.Hash
		mu                          *sync.RWMutex
	}
	type args struct {
		committeeChange *CommitteeChange
		env             *BeaconCommitteeStateEnvironment
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *CommitteeChange
		wantErr bool
	}{
		{
			name:   "Invalid Format Of Public Key In List Unstake Of CommitteeChange",
			fields: fields{},
			args: args{
				committeeChange: &CommitteeChange{
					Unstake: []string{"123"},
				},
				env: &BeaconCommitteeStateEnvironment{},
			},
			want: &CommitteeChange{
				Unstake: []string{"123"},
			},
			wantErr: true,
		},
		{
			name:   "Error When Store Staker Info",
			fields: fields{},
			args: args{
				committeeChange: &CommitteeChange{
					Unstake: []string{key},
				},
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB: sDB,
				},
			},
			want: &CommitteeChange{
				Unstake: []string{key},
			},
			wantErr: true,
		},
		{
			name: "Valid Input",
			fields: fields{
				autoStake: map[string]bool{
					key: true,
				},
				rewardReceiver: map[string]privacy.PaymentAddress{
					rewardReceiver: paymentAddress,
				},
				stakingTx: map[string]common.Hash{
					key: *hash,
				},
			},
			args: args{
				committeeChange: &CommitteeChange{
					Unstake: []string{key},
				},
				env: &BeaconCommitteeStateEnvironment{
					ConsensusStateDB: sDB,
				},
			},
			want: &CommitteeChange{
				Unstake: []string{key},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeStateV1{
				beaconCommittee:             tt.fields.beaconCommittee,
				beaconSubstitute:            tt.fields.beaconSubstitute,
				nextEpochShardCandidate:     tt.fields.nextEpochShardCandidate,
				currentEpochShardCandidate:  tt.fields.currentEpochShardCandidate,
				nextEpochBeaconCandidate:    tt.fields.nextEpochBeaconCandidate,
				currentEpochBeaconCandidate: tt.fields.currentEpochBeaconCandidate,
				shardCommittee:              tt.fields.shardCommittee,
				shardSubstitute:             tt.fields.shardSubstitute,
				autoStake:                   tt.fields.autoStake,
				rewardReceiver:              tt.fields.rewardReceiver,
				stakingTx:                   tt.fields.stakingTx,
				mu:                          tt.fields.mu,
			}
			got, err := b.processUnstakeChange(tt.args.committeeChange, tt.args.env)

			sDB.ClearObjects() // Clear objects for next test case

			if (err != nil) != tt.wantErr {
				t.Errorf("BeaconCommitteeStateV1.processUnstakeChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BeaconCommitteeStateV1.processUnstakeChange() = %v, want %v", got, tt.want)
			}
		})
	}
}
