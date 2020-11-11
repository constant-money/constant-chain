package committeestate

import (
	"reflect"
	"sync"
	"testing"

	"github.com/incognitochain/incognito-chain/incognitokey"

	"github.com/incognitochain/incognito-chain/common"
)

func TestBeaconCommitteeEngineV2_SplitReward(t *testing.T) {

	hash, _ := common.Hash{}.NewHashFromStr("123")

	initLog()
	initPublicKey()

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
		want    map[common.Hash]uint64
		want1   map[common.Hash]uint64
		want2   map[common.Hash]uint64
		want3   map[common.Hash]uint64
		wantErr bool
	}{
		{
			name: "Year 1",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                10,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 1093996,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 51054,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 933543,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 109399,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 2",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                9,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 995536,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 46975,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 858963,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 89598,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 3",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                8,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 905938,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 43217,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 790246,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 72475,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 4",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                7,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 824403,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 39755,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 726940,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 57708,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 5",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                6,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 750207,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 36566,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 668629,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 45012,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 6",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                5,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 682688,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 33629,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 614925,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 34134,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 7",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                4,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 621246,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 30925,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 565472,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 24849,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 8",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                3,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 565334,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 28435,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 519939,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 16960,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
		{
			name: "Year 9",
			fields: fields{
				beaconHash:   *hash,
				beaconHeight: 10,
				finalBeaconCommitteeStateV2: &BeaconCommitteeStateV2{
					beaconCommittee: []incognitokey.CommitteePublicKey{
						*incKey0, *incKey, *incKey2, *incKey3, *incKey, *incKey2, *incKey3,
					},
					shardCommittee: map[byte][]incognitokey.CommitteePublicKey{
						0: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						1: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						2: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						3: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						4: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						5: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						6: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
						7: []incognitokey.CommitteePublicKey{
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey0, *incKey, *incKey2, *incKey3, *incKey4, *incKey5,
							*incKey4, *incKey5,
						},
					},
					mu: &sync.RWMutex{},
				},
				uncommittedBeaconCommitteeStateV2: &BeaconCommitteeStateV2{},
			},
			args: args{
				env: &BeaconCommitteeStateEnvironment{
					DAOPercent:                3,
					ActiveShards:              8,
					IsSplitRewardForCustodian: false,
					PercentCustodianReward:    0,
					TotalReward: map[common.Hash]uint64{
						common.PRVCoinID: 514454,
					},
				},
			},
			want: map[common.Hash]uint64{
				common.PRVCoinID: 25876,
			},
			want1: map[common.Hash]uint64{
				common.PRVCoinID: 473145,
			},
			want2: map[common.Hash]uint64{
				common.PRVCoinID: 15433,
			},
			want3:   map[common.Hash]uint64{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BeaconCommitteeEngineV2{
				beaconHeight:                      tt.fields.beaconHeight,
				beaconHash:                        tt.fields.beaconHash,
				finalBeaconCommitteeStateV2:       tt.fields.finalBeaconCommitteeStateV2,
				uncommittedBeaconCommitteeStateV2: tt.fields.uncommittedBeaconCommitteeStateV2,
			}
			got, got1, got2, got3, err := b.SplitReward(tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeaconCommitteeEngineV2.SplitReward() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BeaconCommitteeEngineV2.SplitReward() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("BeaconCommitteeEngineV2.SplitReward() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("BeaconCommitteeEngineV2.SplitReward() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("BeaconCommitteeEngineV2.SplitReward() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}
