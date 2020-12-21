package blockchain

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/trie"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

var (
	wrarperDB      statedb.DatabaseAccessWarper
	diskDB         incdb.Database
	committeesKeys []incognitokey.CommitteePublicKey
	rewardReceiver map[string]string
)

var _ = func() (_ struct{}) {
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_reward")
	if err != nil {
		panic(err)
	}
	diskDB, _ = incdb.Open("leveldb", dbPath)
	wrarperDB = statedb.NewDatabaseAccessWarper(diskDB)
	trie.Logger.Init(common.NewBackend(nil).Logger("test", true))
	dataaccessobject.Logger.Init(common.NewBackend(nil).Logger("test", true))
	committeesKeysStr := []string{
		"121VhftSAygpEJZ6i9jGkGco4dFKpqVXZA6nmGjRKYWR7Q5NngQSX1adAfYY3EGtS32c846sAxYSKGCpqouqmJghfjtYfHEPZTRXctAcc6bYhR3d1YpB6m3nNjEdTYWf85agBq5QnVShMjBRFf54dK25MAazxBSYmpowxwiaEnEikpQah2W4LY9P9vF9HJuLUZ4BnknoXXK3BVkGHsimy5RXtvNet2LqXZgZWHX5CDj31q7kQ2jUGJHr862MgsaHfT4Qq8o4u71nhgtzKBYgw9fvXqJUU6EVynqJCVdqaDXmUvjanGkaZb9vQjaXVoHyf6XRxVSbQBTS5G7eb4D4V3RucXRLQp34KTadmmNQUxnCoPQztVcuDQwNqy9zRXPPAdw7pWvv7P7p4HuQVAHKqvJskMNk3v971WBH5VpZA1XMkmtu",
		"121VhftSAygpEJZ6i9jGk4diwdFxA6whUVx3P9GmT35Lw6txpbDmeVgSJ4qUwSHPAep8FedvNrZfGB1eoXZXnCwwHVQs7htn7XigUSowaRJyXVf9n42Auhk65GJbxnE7C2t8HWjW3N97m4TejbAQoR5WoWSeaixXRSimadBeWVF4cgZxPUvLuPsSfGYWi4DQ4GwJhpSLNEbite3NseJBDM5N7DGas6mn9roe2jcSYSVyFRR87fqHMfPhhyMQ7k21up58RtMa3tRsEBDBRmKZgeaKr67MuBbEFKJw1Hh8fwbRVaFKeD38EAG9oykANrTmBvZXk4gU8Dvm3uJEJLX7iwDLVxgSDaNYtaYAoePD4dbgWmvotELQW2kJaQ7DEmttV7ZgukQCVPg36pHbDF8oijr5bobgLhft3ajJy5x8mMpuRDYy",
		"121VhftSAygpEJZ6i9jGkMJDUd7LzasxGKrkhLZ2s9mw9YtWaeRVURg9TNxrsxhYLHLSmj1Vh6bpwtpu1i9A7kAM1DzYpTPDws7PftpqS7h9nyHGAu1GLEha57mEEv9U8AWopiQa1Bp5AUq4dQd7o65Ub4S6zMb7aj1Rc7xZ8HRfXScXNdG2cvUEGZXrqS9dF6VwxfdjQa6PKymVHagd2MTL4vaWVcHp9SyKHQqhzkkafKZcb2MruPLPodLnm6Zvd4EqRJYGEZoWWSfSoxsmZsY63W6dsWwpJdZ7sMjuHmqvo1ousSHXnNcGV7LYsuhuwqaMshDsMHdJsFsDErRzTWhwi6jCRUwmaGV82c8JVP3L5HLPdMEJSCa79GwZQNCnkzyHv8DDdX4ptgkzfwQd6bCaNE7DbUUz7yTDa9quVTuYjRWU",
	}
	rewardReceiver = make(map[string]string)
	rewardReceiver["1jTb2CjHbu6dZbuPrumUvtfMkE2qkk39F3M3cnAmsm1kvTYhEY"] = "12RvZy9CVoYK9CWt6JSJhPV2WQjp21UjywYf3YZMoUUaDgGMWeQ1qF9WHZmuPHTyZ28B4Za9hDmxdVLDpy5nJXgTYXANqenbY7Kt4zy"
	rewardReceiver["12bGNd9ofTJSbZYB2BXtaAQpsRV4a3KJ3xP5kLQmoxYBMaqhtXw"] = "12S3wJTUwb8RjHPheQFwer9UPJgs3k1puFnuyAodokcJ7zEcPn6qL72kWCACuDEh5NYrKmz3ctdzd4W2L5P1rbwP75H2D117PVjuS7x"
	rewardReceiver["122yPg6oriAu4uqYeRNr1VN2DScMQAY6xnc1eLaJLr7MPsAsSoJ"] = "12RyAEaUz4sErApu1f23PEydvotxDnC5gHoWDy5Th7JQuoT57oUowk8eSQN44ojPj3wZ5sEYFcLeFU5R8zgiXkSbAuY367Tek31gM1z"
	committeesKeys, _ = incognitokey.CommitteeBase58KeyListToStruct(committeesKeysStr)
	return
}()

func Test_getNoBlkPerYear(t *testing.T) {
	type args struct {
		blockCreationTimeSeconds uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "40s",
			args: args{blockCreationTimeSeconds: 40},
			want: 788940,
		},
		{
			name: "10s",
			args: args{blockCreationTimeSeconds: 10},
			want: 3155760,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNoBlkPerYear(tt.args.blockCreationTimeSeconds); got != tt.want {
				t.Errorf("getNoBlkPerYear() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockChain_getRewardAmount(t *testing.T) {
	numberOfBlockPerYear := getNoBlkPerYear(40)
	type fields struct {
		config Config
	}
	type args struct {
		blkHeight uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint64
	}{
		{
			name: "Mainnet year 1",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				1,
			},
			want: 1386666000,
		},
		{
			name: "Mainnet year 1",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear,
			},
			want: 1386666000,
		},
		{
			name: "Mainnet year 2",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear + 1,
			},
			want: 1261866060,
		},
		{
			name: "Mainnet year 2",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear * 2,
			},
			want: 1261866060,
		},
		{
			name: "Mainnet year 3",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear*2 + 1,
			},
			want: 1148298114,
		},
		{
			name: "Mainnet year 3",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear * 3,
			},
			want: 1148298114,
		},
		{
			name: "Mainnet year 4",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear*3 + 1,
			},
			want: 1044951283,
		},
		{
			name: "Mainnet year 4",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear * 4,
			},
			want: 1044951283,
		},
		{
			name: "Mainnet year 5",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear*4 + 1,
			},
			want: 950905667,
		},
		{
			name: "Mainnet year 5",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear * 5,
			},
			want: 950905667,
		},
		{
			name: "Mainnet year 6",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear*5 + 1,
			},
			want: 865324156,
		},
		{
			name: "Mainnet year 6",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear * 6,
			},
			want: 865324156,
		},
		{
			name: "Mainnet year 7",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear*6 + 1,
			},
			want: 787444981,
		},
		{
			name: "Mainnet year 7",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear * 7,
			},
			want: 787444981,
		},
		{
			name: "Mainnet year 8",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear*7 + 1,
			},
			want: 716574932,
		},
		{
			name: "Mainnet year 8",
			fields: fields{
				config: Config{
					ChainParams: &Params{
						MinBeaconBlockInterval: MainnetMinBeaconBlkInterval,
						BasicReward:            MainnetBasicReward,
					},
				},
			},
			args: args{
				numberOfBlockPerYear * 8,
			},
			want: 716574932,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockchain := &BlockChain{
				config: tt.fields.config,
			}
			if got := blockchain.getRewardAmount(tt.args.blkHeight); got != tt.want {
				t.Errorf("getRewardAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_splitReward(t *testing.T) {
	type args struct {
		totalReward               *map[common.Hash]uint64
		numberOfActiveShards      int
		devPercent                int
		isSplitRewardForCustodian bool
		percentCustodianRewards   uint64
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
		want    *map[common.Hash]uint64
		want1   *map[common.Hash]uint64
		want2   *map[common.Hash]uint64
		want3   *map[common.Hash]uint64
		wantErr bool
	}{
		{
			name: "year 1",
			args: args{
				totalReward:               &totalRewardYear1,
				numberOfActiveShards:      8,
				devPercent:                10,
				isSplitRewardForCustodian: false,
				percentCustodianRewards:   0,
			},
			want:  &beaconRewardYear1,
			want1: &daoRewardYear1,
			want2: &custodianRewardYear1,
			want3: &shardRewardYear1,
		},
		{
			name: "year 2",
			args: args{
				totalReward:               &totalRewardYear2,
				numberOfActiveShards:      8,
				devPercent:                9,
				isSplitRewardForCustodian: false,
				percentCustodianRewards:   0,
			},
			want:  &beaconRewardYear2,
			want1: &daoRewardYear2,
			want2: &custodianRewardYear2,
			want3: &shardRewardYear2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := splitReward(tt.args.totalReward, tt.args.numberOfActiveShards, tt.args.devPercent, tt.args.isSplitRewardForCustodian, tt.args.percentCustodianRewards)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitReward() error = %v, wantErr %v", err, tt.wantErr)
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
			if !reflect.DeepEqual(tt.args.totalReward, tt.want3) {
				t.Errorf("splitReward() totalReward = %v, want %v", tt.args.totalReward, tt.want3)
			}
		})
	}
}

func Test_getPercentForIncognitoDAO(t *testing.T) {
	type args struct {
		blockHeight uint64
		blkPerYear  uint64
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "year 1",
			args: args{
				blockHeight: 788940,
				blkPerYear:  788940,
			},
			want: 10,
		},
		{
			name: "year 2-1",
			args: args{
				blockHeight: 788941,
				blkPerYear:  788940,
			},
			want: 9,
		}, {
			name: "year 2-2",
			args: args{
				blockHeight: 1577880,
				blkPerYear:  788940,
			},
			want: 9,
		},
		{
			name: "year 3-1",
			args: args{
				blockHeight: 1577881,
				blkPerYear:  788940,
			},
			want: 8,
		},
		{
			name: "year 3-2",
			args: args{
				blockHeight: 2366820,
				blkPerYear:  788940,
			},
			want: 8,
		},
		{
			name: "year 4-1",
			args: args{
				blockHeight: 2366821,
				blkPerYear:  788940,
			},
			want: 7,
		},
		{
			name: "year 4-2",
			args: args{
				blockHeight: 3155760,
				blkPerYear:  788940,
			},
			want: 7,
		},
		{
			name: "year 5-1",
			args: args{
				blockHeight: 3155761,
				blkPerYear:  788940,
			},
			want: 6,
		},
		{
			name: "year 5-2",
			args: args{
				blockHeight: 3944700,
				blkPerYear:  788940,
			},
			want: 6,
		},
		{
			name: "year 6-1",
			args: args{
				blockHeight: 3944701,
				blkPerYear:  788940,
			},
			want: 5,
		},
		{
			name: "year 6-2",
			args: args{
				blockHeight: 4733640,
				blkPerYear:  788940,
			},
			want: 5,
		},
		{
			name: "year 7",
			args: args{
				blockHeight: 5522580,
				blkPerYear:  788940,
			},
			want: 4,
		},
		{
			name: "year 8",
			args: args{
				blockHeight: 6311520,
				blkPerYear:  788940,
			},
			want: 3,
		},
		{
			name: "year 9",
			args: args{
				blockHeight: 7100460,
				blkPerYear:  788940,
			},
			want: 3,
		},
		{
			name: "year 10",
			args: args{
				blockHeight: 7889400,
				blkPerYear:  788940,
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPercentForIncognitoDAO(tt.args.blockHeight, tt.args.blkPerYear); got != tt.want {
				t.Errorf("getPercentForIncognitoDAO() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockChain_addShardRewardRequestToBeacon(t *testing.T) {
	config := Config{}
	config.ChainParams = &ChainMainParam
	sDB, _ := statedb.NewWithPrefixTrie(common.EmptyRoot, wrarperDB)
	acceptedBlockRewardInfoBase := metadata.NewAcceptedBlockRewardInfo(0, make(map[common.Hash]uint64), 2)
	acceptedBlockRewardInfoBaseInst, _ := acceptedBlockRewardInfoBase.GetStringFormat()
	txFee := make(map[common.Hash]uint64)
	txFee1 := uint64(10000)
	txFee[common.PRVCoinID] = txFee1
	acceptedBlockRewardInfo1 := metadata.NewAcceptedBlockRewardInfo(0, txFee, 2)
	acceptedBlockRewardInfo1Inst, _ := acceptedBlockRewardInfo1.GetStringFormat()
	type fields struct {
		config Config
	}
	type args struct {
		beaconBlock   *BeaconBlock
		rewardStateDB *statedb.StateDB
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "add base reward",
			fields: fields{
				config: config,
			},
			args: args{
				beaconBlock: &BeaconBlock{
					Body: BeaconBody{
						Instructions: [][]string{acceptedBlockRewardInfoBaseInst},
					},
					Header: BeaconHeader{
						Epoch: 1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "add base reward + 10000",
			fields: fields{
				config: config,
			},
			args: args{
				beaconBlock: &BeaconBlock{
					Body: BeaconBody{
						Instructions: [][]string{acceptedBlockRewardInfo1Inst},
					},
					Header: BeaconHeader{
						Epoch: 1,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockchain := &BlockChain{
				config: tt.fields.config,
			}
			if err := blockchain.addShardRewardRequestToBeacon(tt.args.beaconBlock, sDB); (err != nil) != tt.wantErr {
				t.Errorf("addShardRewardRequestToBeacon() error = %v, wantErr %v", err, tt.wantErr)
			}
			rootHash, _ := sDB.Commit(true)
			_ = sDB.Database().TrieDB().Commit(rootHash, false)
		})
	}
	reward, err := statedb.GetRewardOfShardByEpoch(sDB, 1, 0, common.PRVCoinID)
	if err != nil {
		t.Error(err)
	}
	wantReward := 1386666000*2 + txFee1
	if reward != wantReward {
		t.Errorf("addShardRewardRequestToBeacon() got base reward = %v, want %v", reward, wantReward)
	}
}

func TestBlockChain_buildInstRewardForBeacons(t *testing.T) {
	type fields struct {
		BestState *BestState
	}
	fields1 := fields{
		BestState: &BestState{Beacon: &BeaconBestState{BeaconCommittee: committeesKeys}},
	}
	totalReward1 := make(map[common.Hash]uint64)
	totalReward1_1 := make(map[common.Hash]uint64)
	totalReward1[common.PRVCoinID] = 900
	totalReward1_1[common.PRVCoinID] = 300
	rewardInst1_1, _ := metadata.BuildInstForBeaconReward(totalReward1_1, committeesKeys[0].GetNormalKey())
	rewardInst1_2, _ := metadata.BuildInstForBeaconReward(totalReward1_1, committeesKeys[1].GetNormalKey())
	rewardInst1_3, _ := metadata.BuildInstForBeaconReward(totalReward1_1, committeesKeys[2].GetNormalKey())
	type args struct {
		epoch       uint64
		totalReward map[common.Hash]uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    [][]string
		wantErr bool
	}{
		{
			name:   "committee len 3",
			fields: fields1,
			args: args{
				epoch:       1,
				totalReward: totalReward1,
			},
			want:    [][]string{rewardInst1_1, rewardInst1_2, rewardInst1_3},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockchain := &BlockChain{
				BestState: tt.fields.BestState,
			}
			got, err := blockchain.buildInstRewardForBeacons(tt.args.epoch, tt.args.totalReward)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildInstRewardForBeacons() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildInstRewardForBeacons() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockChain_buildInstRewardForIncDAO(t *testing.T) {
	type fields struct {
		config Config
	}
	fields1 := fields{
		config: Config{
			ChainParams: &ChainMainParam,
		},
	}
	totalReward1 := make(map[common.Hash]uint64)
	wantReward1 := uint64(256)
	totalReward1[common.PRVCoinID] = wantReward1
	rewardInst1_DAO, _ := metadata.BuildInstForIncDAOReward(totalReward1, ChainMainParam.IncognitoDAOAddress)
	type args struct {
		epoch       uint64
		totalReward map[common.Hash]uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    [][]string
		wantErr bool
	}{
		{
			name:   "build DAO mainnet",
			fields: fields1,
			args: args{
				epoch:       1,
				totalReward: totalReward1,
			},
			want:    [][]string{rewardInst1_DAO},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockchain := &BlockChain{
				config: tt.fields.config,
			}
			got, err := blockchain.buildInstRewardForIncDAO(tt.args.epoch, tt.args.totalReward)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildInstRewardForIncDAO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildInstRewardForIncDAO() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockChain_buildInstRewardForShards(t *testing.T) {
	type args struct {
		epoch        uint64
		totalRewards []map[common.Hash]uint64
	}
	totalRewardShard0_1 := make(map[common.Hash]uint64)
	totalRewardShard0_1[common.PRVCoinID] = 1000
	totalRewardShard1_1 := make(map[common.Hash]uint64)
	totalRewardShard1_1[common.PRVCoinID] = 1123
	rewardInstShard0_1, _ := metadata.BuildInstForShardReward(totalRewardShard0_1, 1, 0)
	rewardInstShard1_1, _ := metadata.BuildInstForShardReward(totalRewardShard1_1, 1, 1)
	tests := []struct {
		name    string
		args    args
		want    [][]string
		wantErr bool
	}{
		{
			name: "shard 0,1",
			args: args{
				1,
				[]map[common.Hash]uint64{totalRewardShard0_1, totalRewardShard1_1},
			},
			want:    [][]string{rewardInstShard0_1[0], rewardInstShard1_1[0]},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockchain := &BlockChain{}
			got, err := blockchain.buildInstRewardForShards(tt.args.epoch, tt.args.totalRewards)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildInstRewardForShards() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildInstRewardForShards() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockChain_addShardCommitteeReward(t *testing.T) {
	sDB, _ := statedb.NewWithPrefixTrie(common.EmptyRoot, wrarperDB)
	totalRewardShard1_1 := make(map[common.Hash]uint64)
	wantReward := uint64(1000)
	totalRewardShard1_1[common.PRVCoinID] = wantReward
	rewardInstShard1_1 := &metadata.ShardBlockRewardInfo{
		Epoch:       1,
		ShardReward: totalRewardShard1_1,
	}
	type args struct {
		rewardStateDB             *statedb.StateDB
		shardID                   byte
		rewardInfoShardToProcess  *metadata.ShardBlockRewardInfo
		committeeOfShardToProcess []incognitokey.CommitteePublicKey
		rewardReceiver            map[string]string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "candidate shard 1",
			args: args{
				rewardStateDB:             sDB,
				shardID:                   1,
				rewardInfoShardToProcess:  rewardInstShard1_1,
				committeeOfShardToProcess: committeesKeys,
				rewardReceiver:            rewardReceiver,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockchain := &BlockChain{}
			if err := blockchain.addShardCommitteeReward(tt.args.rewardStateDB, tt.args.shardID, tt.args.rewardInfoShardToProcess, tt.args.committeeOfShardToProcess, tt.args.rewardReceiver); (err != nil) != tt.wantErr {
				t.Errorf("addShardCommitteeReward() error = %v, wantErr %v", err, tt.wantErr)
			}
			rootHash, _ := sDB.Commit(true)
			_ = sDB.Database().TrieDB().Commit(rootHash, false)
		})
	}
	reward, err := statedb.GetCommitteeReward(sDB, committeesKeys[2].GetIncKeyBase58(), common.PRVCoinID)
	if err != nil {
		t.Errorf("addShardCommitteeReward() error = %v, wantErr %v", err, nil)
	}
	if reward != wantReward/3 {
		t.Errorf("addShardCommitteeReward() reward = %v, wantReward %v", reward, wantReward/3)
	}
}
