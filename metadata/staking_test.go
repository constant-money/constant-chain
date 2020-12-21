package metadata_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/metadata/mocks"
	"github.com/incognitochain/incognito-chain/trie"
)

var (
	wrarperDB statedb.DatabaseAccessWarper
	diskDB    incdb.Database
)

var _ = func() (_ struct{}) {
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_metadata")
	if err != nil {
		panic(err)
	}
	diskDB, _ = incdb.Open("leveldb", dbPath)
	wrarperDB = statedb.NewDatabaseAccessWarper(diskDB)
	emptyStateDB, _ = statedb.NewWithPrefixTrie(common.EmptyRoot, wrarperDB)
	validCommitteePublicKeyStructs, _ = incognitokey.CommitteeBase58KeyListToStruct(validCommitteePublicKeys)
	metadata.Logger.Init(common.NewBackend(nil).Logger("test", true))
	trie.Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

var (
	emptyStateDB     *statedb.StateDB
	validPrivateKeys = []string{
		"112t8rq19Uu7UGbTApZzZwCAvVszAgRNAzHzr3p8Cu75jPH3h5AUtRXMKiqF3hw8NbEfeLcjtbpeUvJfw4tGj7pbqwDYngc8wB13Gf77o33f",
		"112t8rrEW3NPNgU8xzbeqE7cr4WTT8JvyaQqSZyczA5hBJVvpQMTBVqNfcCdzhvquWCHH11jHihZtgyJqbdWPhWYbmmsw5aV29WSXBEsgbVX",
	}
	validCommitteePublicKeys = []string{
		"121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7",
		"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
		"121VhftSAygpEJZ6i9jGkM3wj8iY8VF2bkfRdzQe5zetb5SHHEwh5Fa7CSnsAwfZB4yoKxJiuo9ak2WNMS513i24fMG9MTYEaarsLM787wmzkEP37jo3A1wEc4Q7EgiGWi1YKYF9R15vwK4dZJvbAUwPHvbXu53QNxvhcApoArM2dhJBrBDA8n6WbyJNcsTBjx2eP5pAKAhsoN29cCNCDT9jRKL2NwizV5pYpGNzGtBwT8wxLFPdueRanFyNZAsZ5jHsTw8XudBCDi4ACrvp8uz66y8PtEyTvtz3htmwKVoy7kF6tqyhFXdXzpccLWJcRRV4vQtpCkhQr1Reysbs6Vxf7Vz69H6jh2ARQSE2YyTWFZ9kBzRh65E4HacLDp7QkBbLDehu3viK6s5H3vB6QUsPrq3kW5eD4dDrqDdGPUqPVdAv",
	}
	validCommitteePublicKeyStructs = []incognitokey.CommitteePublicKey{}
	validPaymentAddresses          = []string{
		"12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX",
		"12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
	}
	validPublicKeys = []string{
		"12buoC8Nmh8WbPhSAiF1SSNB8AuxTu3QbX3sSUydqod4y9ws3e3",
		"1HXXH7MxWGQgg2QZP854WjDYtebEiKDwPjJzFqBpTUJ447GEG2",
	}
	validPrivateSeeds = []string{
		"129pZpqYqYAA8wTAeDKuVwRthoBjNLUFm8FnLwUTkXddUqwShN9",
		"12JqKehM24bfSkfv3FKGtzFw4seoJSJbbgAqaYtX3w6DjVuH8mb",
	}
	invalidCommitteePublicKeys = []string{"121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAnaeixKC3AdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7",
		"1hm766APBSXcyDbNbPLbb65Hm2DkK35RJp1cwYx95mFExK3VAkE9qfzDJLTKTMiKbscm4zns5QuDpGS4yc5Hi994G1BVVE2hdLgoNJbvxXdbmsRdrwVCENVYJhYk2k1kci7b8ysb9nFXW8fUEJNsBtfQjtXQY7pEqngbwpEFuF45Kj8skjDriKp2Sc9TjxnPw4478dN4h4XYojPaiSo3sJpqJWDfcZ68DqSWuUAud5REAqeBT3sUiyJCpnfZ9Lp2Uk7M7Pc9CeuTZBVfV3M669zpPdErUgWf7VDYe5wujvcMLhqqjvJRe5WREYLjVni1H1d4qhcuzdbPdW8BC4b7xY2qRSBtiFav8tJt7iSdycTeTTsaYN1"}
	invalidPaymentAddresses = []string{
		"12S42qYc9pzsfWoxPZ21sVih7CGYMHNWX12SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX"}
)

func TestNewStakingMetadata(t *testing.T) {
	type args struct {
		stakingType                  int
		funderPaymentAddress         string
		rewardReceiverPaymentAddress string
		stakingAmountShard           uint64
		committeePublicKey           string
		autoReStaking                bool
	}
	tests := []struct {
		name    string
		args    args
		want    *metadata.StakingMetadata
		wantErr bool
	}{
		{
			name: "check StakingType error case",
			args: args{
				stakingType:                  65,
				funderPaymentAddress:         validPaymentAddresses[0],
				rewardReceiverPaymentAddress: validPaymentAddresses[0],
				stakingAmountShard:           1750000000000,
				committeePublicKey:           validCommitteePublicKeys[0],
				autoReStaking:                false,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "check StakingType success case",
			args: args{
				stakingType:                  63,
				funderPaymentAddress:         validPaymentAddresses[0],
				rewardReceiverPaymentAddress: validPaymentAddresses[0],
				stakingAmountShard:           1750000000000,
				committeePublicKey:           validCommitteePublicKeys[0],
				autoReStaking:                false,
			},
			want:    &metadata.StakingMetadata{metadata.MetadataBase{63}, "12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX", "12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX", 1750000000000, false, "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := metadata.NewStakingMetadata(tt.args.stakingType, tt.args.funderPaymentAddress, tt.args.rewardReceiverPaymentAddress, tt.args.stakingAmountShard, tt.args.committeePublicKey, tt.args.autoReStaking)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStakingMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStakingMetadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStakingMetadata_ValidateMetadataByItself(t *testing.T) {
	type fields struct {
		MetadataBase                 metadata.MetadataBase
		FunderPaymentAddress         string
		RewardReceiverPaymentAddress string
		StakingAmountShard           uint64
		AutoReStaking                bool
		CommitteePublicKey           string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "check Base58CheckDeserialize error case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{63},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: invalidPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           validCommitteePublicKeys[0],
			},
			want: false,
		},
		{
			name: "check IsInBase58ShortFormat error case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{63},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           invalidCommitteePublicKeys[0],
			},
			want: false,
		},
		{
			name: "check CommitteePublicKey.FromString error case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{63},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           invalidCommitteePublicKeys[0],
			},
			want: false,
		},
		{
			name: "check CommitteePublicKey.CheckSanityData error case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{63},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           invalidCommitteePublicKeys[1],
			},
			want: false,
		},
		{
			name: "happy case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{63},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           validCommitteePublicKeys[0],
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &metadata.StakingMetadata{
				MetadataBase:                 tt.fields.MetadataBase,
				FunderPaymentAddress:         tt.fields.FunderPaymentAddress,
				RewardReceiverPaymentAddress: tt.fields.RewardReceiverPaymentAddress,
				StakingAmountShard:           tt.fields.StakingAmountShard,
				AutoReStaking:                tt.fields.AutoReStaking,
				CommitteePublicKey:           tt.fields.CommitteePublicKey,
			}
			if got := sm.ValidateMetadataByItself(); got != tt.want {
				t.Errorf("ValidateMetadataByItself() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStakingMetadata_ValidateSanityData(t *testing.T) {
	type fields struct {
		MetadataBase                 metadata.MetadataBase
		FunderPaymentAddress         string
		RewardReceiverPaymentAddress string
		StakingAmountShard           uint64
		AutoReStaking                bool
		CommitteePublicKey           string
	}
	type args struct {
		chainRetriever  metadata.ChainRetriever
		shardRetriever  metadata.ShardViewRetriever
		beaconRetriever metadata.BeaconViewRetriever
		beaconHeight    uint64
		tx              metadata.Transaction
	}

	txIsPrivacyError := &mocks.Transaction{}
	txIsPrivacyError.On("IsPrivacy").Return(true)

	txGetUniqueReceiverError := &mocks.Transaction{}
	txGetUniqueReceiverError.On("IsPrivacy").Return(false)
	txGetUniqueReceiverError.On("GetUniqueReceiver").Return(false, []byte{}, uint64(0))

	bcrBase58CheckDeserializeError := &mocks.ChainRetriever{}
	bcrBase58CheckDeserializeError.On("GetBurningAddress", uint64(0)).Return("15pABFiJVeh9D5uiipQxBdSVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs")
	txBase58CheckDeserializeError := &mocks.Transaction{}
	txBase58CheckDeserializeError.On("IsPrivacy").Return(false)
	txBase58CheckDeserializeError.On("GetUniqueReceiver").Return(true, []byte{}, uint64(0))

	bcrBurningAddressPublicKeyError := &mocks.ChainRetriever{}
	bcrBurningAddressPublicKeyError.On("GetBurningAddress", uint64(0)).Return("15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs")
	txBurningAddressPublicKeyError := &mocks.Transaction{}
	txBurningAddressPublicKeyError.On("IsPrivacy").Return(false)
	txBurningAddressPublicKeyError.On("GetUniqueReceiver").Return(true, []byte{0, 183, 246, 161, 68, 172, 228, 222, 153, 9, 172, 39, 208, 245, 167, 79, 11, 2, 114, 65, 241, 69, 85, 40, 193, 104, 199, 79, 70, 4, 53, 0}, uint64(1650000000000))

	bcrGetStakingAmountShardError := &mocks.ChainRetriever{}
	bcrGetStakingAmountShardError.On("GetBurningAddress", uint64(0)).Return("15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs")
	bcrGetStakingAmountShardError.On("GetStakingAmountShard").Return(uint64(1750000000000))
	txGetStakingAmountShardError := &mocks.Transaction{}
	txGetStakingAmountShardError.On("IsPrivacy").Return(false)
	txGetStakingAmountShardError.On("GetUniqueReceiver").Return(true, []byte{99, 183, 246, 161, 68, 172, 228, 222, 153, 9, 172, 39, 208, 245, 167, 79, 11, 2, 114, 65, 241, 69, 85, 40, 193, 104, 199, 79, 70, 4, 53, 0}, uint64(1650000000000))

	txGetStakingAmountBeaconError := &mocks.Transaction{}
	txGetStakingAmountBeaconError.On("IsPrivacy").Return(false)
	txGetStakingAmountBeaconError.On("GetUniqueReceiver").Return(true, []byte{99, 183, 246, 161, 68, 172, 228, 222, 153, 9, 172, 39, 208, 245, 167, 79, 11, 2, 114, 65, 241, 69, 85, 40, 193, 104, 199, 79, 70, 4, 53, 0}, uint64(1750000000000))

	txBase58CheckDeserialize2Error := &mocks.Transaction{}
	txBase58CheckDeserialize2Error.On("IsPrivacy").Return(false)
	txBase58CheckDeserialize2Error.On("GetUniqueReceiver").Return(true, []byte{99, 183, 246, 161, 68, 172, 228, 222, 153, 9, 172, 39, 208, 245, 167, 79, 11, 2, 114, 65, 241, 69, 85, 40, 193, 104, 199, 79, 70, 4, 53, 0}, uint64(1750000000000))

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		want1   bool
		want2   error
		wantErr bool
	}{
		{
			name:   "check tx.IsPrivacy error case",
			fields: fields{},
			args: args{
				tx: txIsPrivacyError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name:   "check tx.GetUniqueReceiver error case",
			fields: fields{},
			args: args{
				tx: txGetUniqueReceiverError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name:   "check wallet.Base58CheckDeserialize error case",
			fields: fields{},
			args: args{
				tx:             txBase58CheckDeserializeError,
				chainRetriever: bcrBase58CheckDeserializeError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name:   "stake check burning address error",
			fields: fields{},
			args: args{
				tx:             txBurningAddressPublicKeyError,
				chainRetriever: bcrBurningAddressPublicKeyError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "check wallet.chainRetriever.GetStakingAmountShard() && Stake Shard error case",
			fields: fields{
				MetadataBase: metadata.MetadataBase{metadata.ShardStakingMeta},
			},
			args: args{
				tx:             txGetStakingAmountShardError,
				chainRetriever: bcrGetStakingAmountShardError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "check wallet.chainRetriever.GetStakingAmountShard() * 3 && Stake Beacon error case",
			fields: fields{
				MetadataBase: metadata.MetadataBase{metadata.BeaconStakingMeta},
			},
			args: args{
				tx:             txGetStakingAmountBeaconError,
				chainRetriever: bcrGetStakingAmountShardError},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "check wallet.Base58CheckDeserialize(funderPaymentAddress) error case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{metadata.ShardStakingMeta},
				FunderPaymentAddress:         "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gRpmgkqD4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
				RewardReceiverPaymentAddress: "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
			},
			args: args{
				tx:             txBase58CheckDeserialize2Error,
				chainRetriever: bcrGetStakingAmountShardError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "check wallet.Base58CheckDeserialize(rewardReceiverPaymentAddress) error case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{metadata.ShardStakingMeta},
				FunderPaymentAddress:         "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
				RewardReceiverPaymentAddress: "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVaNomBK1yYC",
			},
			args: args{
				tx:             txBase58CheckDeserialize2Error,
				chainRetriever: bcrGetStakingAmountShardError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "check CommitteePublicKey.FromString error case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{metadata.ShardStakingMeta},
				FunderPaymentAddress:         "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
				RewardReceiverPaymentAddress: "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
				CommitteePublicKey:           invalidCommitteePublicKeys[0],
			},
			args: args{
				tx:             txBase58CheckDeserialize2Error,
				chainRetriever: bcrGetStakingAmountShardError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "happy case",
			fields: fields{
				MetadataBase:                 metadata.MetadataBase{metadata.ShardStakingMeta},
				FunderPaymentAddress:         "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
				RewardReceiverPaymentAddress: "12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
				CommitteePublicKey:           validCommitteePublicKeys[0],
			},
			args: args{
				tx:             txBase58CheckDeserialize2Error,
				chainRetriever: bcrGetStakingAmountShardError,
			},
			want:    true,
			want1:   true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stakingMetadata := metadata.StakingMetadata{
				MetadataBase:                 tt.fields.MetadataBase,
				FunderPaymentAddress:         tt.fields.FunderPaymentAddress,
				RewardReceiverPaymentAddress: tt.fields.RewardReceiverPaymentAddress,
				StakingAmountShard:           tt.fields.StakingAmountShard,
				AutoReStaking:                tt.fields.AutoReStaking,
				CommitteePublicKey:           tt.fields.CommitteePublicKey,
			}
			got, got1, err := stakingMetadata.ValidateSanityData(tt.args.chainRetriever, tt.args.shardRetriever, tt.args.beaconRetriever, tt.args.beaconHeight, tt.args.tx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSanityData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateSanityData() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ValidateSanityData() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestStakingMetadata_ValidateTxWithBlockChain(t *testing.T) {
	SC := make(map[byte][]incognitokey.CommitteePublicKey)
	SPV := make(map[byte][]incognitokey.CommitteePublicKey)
	happyCaseBeaconRetriever := &mocks.BeaconViewRetriever{}
	happyCaseBeaconRetriever.On("GetAllCommitteeValidatorCandidate").
		Return(SC, SPV, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{},
			nil)
	stakeAlreadyBeaconRetriever := &mocks.BeaconViewRetriever{}
	stakeAlreadyBeaconRetriever.On("GetAllCommitteeValidatorCandidate").
		Return(SC, SPV, []incognitokey.CommitteePublicKey{validCommitteePublicKeyStructs[0]}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{},
			nil)
	getCommitteeErrorBeaconRetriever := &mocks.BeaconViewRetriever{}
	getCommitteeErrorBeaconRetriever.On("GetAllCommitteeValidatorCandidate").
		Return(SC, SPV, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{}, []incognitokey.CommitteePublicKey{},
			errors.New("get committee error"))

	type fields struct {
		MetadataBase                 metadata.MetadataBase
		FunderPaymentAddress         string
		RewardReceiverPaymentAddress string
		StakingAmountShard           uint64
		AutoReStaking                bool
		CommitteePublicKey           string
	}
	type args struct {
		tx                  metadata.Transaction
		chainRetriever      metadata.ChainRetriever
		shardViewRetriever  metadata.ShardViewRetriever
		beaconViewRetriever metadata.BeaconViewRetriever
		b                   byte
		stateDB             *statedb.StateDB
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "happy case",
			fields: fields{
				MetadataBase: metadata.MetadataBase{
					metadata.ShardStakingMeta,
				},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           validCommitteePublicKeys[0],
			},
			args: args{
				tx:                  &mocks.Transaction{},
				beaconViewRetriever: happyCaseBeaconRetriever,
				b:                   0,
				stateDB:             emptyStateDB,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "stake already error case",
			fields: fields{
				MetadataBase: metadata.MetadataBase{
					metadata.ShardStakingMeta,
				},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           validCommitteePublicKeys[0],
			},
			args: args{
				tx:                  &mocks.Transaction{},
				beaconViewRetriever: stakeAlreadyBeaconRetriever,
				b:                   0,
				stateDB:             emptyStateDB,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "get committee error case",
			fields: fields{
				MetadataBase: metadata.MetadataBase{
					metadata.ShardStakingMeta,
				},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           validCommitteePublicKeys[0],
			},
			args: args{
				tx:                  &mocks.Transaction{},
				beaconViewRetriever: getCommitteeErrorBeaconRetriever,
				b:                   0,
				stateDB:             emptyStateDB,
			},
			want:    false,
			wantErr: true,
		},

		{
			name: "CommitteeBase58KeyListToStruct error case",
			fields: fields{
				MetadataBase: metadata.MetadataBase{
					metadata.ShardStakingMeta,
				},
				FunderPaymentAddress:         validPaymentAddresses[0],
				RewardReceiverPaymentAddress: validPaymentAddresses[0],
				StakingAmountShard:           1750000000000,
				AutoReStaking:                false,
				CommitteePublicKey:           invalidCommitteePublicKeys[0],
			},
			args: args{
				tx:                  &mocks.Transaction{},
				beaconViewRetriever: happyCaseBeaconRetriever,
				b:                   0,
				stateDB:             emptyStateDB,
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stakingMetadata := metadata.StakingMetadata{
				MetadataBase:                 tt.fields.MetadataBase,
				FunderPaymentAddress:         tt.fields.FunderPaymentAddress,
				RewardReceiverPaymentAddress: tt.fields.RewardReceiverPaymentAddress,
				StakingAmountShard:           tt.fields.StakingAmountShard,
				AutoReStaking:                tt.fields.AutoReStaking,
				CommitteePublicKey:           tt.fields.CommitteePublicKey,
			}
			got, err := stakingMetadata.ValidateTxWithBlockChain(tt.args.tx, tt.args.chainRetriever, tt.args.shardViewRetriever, tt.args.beaconViewRetriever, tt.args.b, tt.args.stateDB)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTxWithBlockChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateTxWithBlockChain() got = %v, want %v", got, tt.want)
			}
			fmt.Println(err)
		})
	}
}
