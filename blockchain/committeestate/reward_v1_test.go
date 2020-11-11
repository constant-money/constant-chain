package committeestate

import (
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
)

func TestBeaconCommitteeEngineV1_SplitReward(t *testing.T) {

	initLog()

	type args struct {
		env *BeaconCommitteeStateEnvironment
	}
	totalRewardYear1 := make(map[common.Hash]uint64)
	totalRewardYear1[common.PRVCoinID] = 8751970
	beaconRewardYear1 := make(map[common.Hash]uint64)
	beaconRewardYear1[common.PRVCoinID] = 1575354
	daoRewardYear1 := make(map[common.Hash]uint64)
	daoRewardYear1[common.PRVCoinID] = 875197
	custodianRewardYear1 := make(map[common.Hash]uint64)
	shardRewardYear1 := make(map[common.Hash]uint64)
	shardRewardYear1[common.PRVCoinID] = 6301419

	totalRewardYear2 := make(map[common.Hash]uint64)
	totalRewardYear2[common.PRVCoinID] = 7964293
	beaconRewardYear2 := make(map[common.Hash]uint64)
	beaconRewardYear2[common.PRVCoinID] = 1449501
	daoRewardYear2 := make(map[common.Hash]uint64)
	daoRewardYear2[common.PRVCoinID] = 716786
	custodianRewardYear2 := make(map[common.Hash]uint64)
	shardRewardYear2 := make(map[common.Hash]uint64)
	shardRewardYear2[common.PRVCoinID] = 5798006
	tests := []struct {
		name    string
		args    args
		want    map[common.Hash]uint64
		want1   map[common.Hash]uint64
		want2   map[common.Hash]uint64
		want3   map[common.Hash]uint64
		wantErr bool
	}{
		{
			name: "year 1",
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                10,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward:               totalRewardYear1,
				},
			},
			want:  beaconRewardYear1,
			want1: shardRewardYear1,
			want2: daoRewardYear1,
			want3: custodianRewardYear1,
		},
		{
			name: "year 2",
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                9,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward:               totalRewardYear2,
				},
			},
			want:  beaconRewardYear2,
			want1: shardRewardYear2,
			want2: daoRewardYear2,
			want3: custodianRewardYear2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeEngineV1{
				beaconHeight:                      10,
				beaconHash:                        common.Hash{},
				beaconCommitteeStateV1:            &BeaconCommitteeStateV1{},
				uncommittedBeaconCommitteeStateV1: &BeaconCommitteeStateV1{},
				version:                           1,
			}
			got, got1, got2, got3, err := b.SplitReward(tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeaconCommitteeEngineV1.SplitReward() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitReward() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("splitReward() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("splitReward() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("splitReward() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}
