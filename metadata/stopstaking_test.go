package metadata_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/metadata/mocks"
)

func TestNewStopAutoStakingMetadata(t *testing.T) {
	type args struct {
		stopStakingType    int
		committeePublicKey string
	}
	tests := []struct {
		name    string
		args    args
		want    *metadata.StopAutoStakingMetadata
		wantErr bool
	}{
		{
			name: "check StakingType error",
			args: args{
				stopStakingType: metadata.BeaconStakingMeta,
			},
			wantErr: true,
		},
		{
			name: "check StakingType success",
			args: args{
				stopStakingType:    metadata.StopAutoStakingMeta,
				committeePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			want: &metadata.StopAutoStakingMetadata{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := metadata.NewStopAutoStakingMetadata(tt.args.stopStakingType, tt.args.committeePublicKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStopAutoStakingMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStopAutoStakingMetadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStopAutoStakingMetadata_ValidateMetadataByItself(t *testing.T) {
	type fields struct {
		MetadataBase       metadata.MetadataBase
		CommitteePublicKey string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Base58CheckDeserialize error",
			fields: fields{
				MetadataBase: metadata.MetadataBase{
					metadata.StopAutoStakingMeta,
				},
				CommitteePublicKey: "blah",
			},
		},
		{
			name: "CheckSanityData error",
			fields: fields{
				MetadataBase: metadata.MetadataBase{
					metadata.StopAutoStakingMeta,
				},
				CommitteePublicKey: "1hm766APBSXcyDbNbPLbb65Hm2DkK35RJp1cwYx95mFExK3VAkE9qfzDJLTKTMiKbscm4zns5QuDpGS4yc5Hi994G1BVVE2hdLgoNJbvxXdbmsRdrwVCENVYJhYk2k1kci7b8ysb9nFXW8fUEJNsBtfQjtXQY7pEqngbwpEFuF45Kj8skjDriKp2Sc9TjxnPw4478dN4h4XYojPaiSo3sJpqJWDfcZ68DqSWuUAud5REAqeBT3sUiyJCpnfZ9Lp2Uk7M7Pc9CeuTZBVfV3M669zpPdErUgWf7VDYe5wujvcMLhqqjvJRe5WREYLjVni1H1d4qhcuzdbPdW8BC4b7xY2qRSBtiFav8tJt7iSdycTeTTsaYN1",
			},
		},
		{
			name: "stopAutoStakingMetadata error",
			fields: fields{MetadataBase: metadata.MetadataBase{
				metadata.ShardStakingMeta,
			},
				CommitteePublicKey: "blah"},
		},
		{
			name: "happy case",
			fields: fields{
				MetadataBase: metadata.MetadataBase{
					metadata.StopAutoStakingMeta,
				},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopAutoStakingMetadata := &metadata.StopAutoStakingMetadata{
				MetadataBase:       tt.fields.MetadataBase,
				CommitteePublicKey: tt.fields.CommitteePublicKey,
			}
			if got := stopAutoStakingMetadata.ValidateMetadataByItself(); got != tt.want {
				t.Errorf("ValidateMetadataByItself() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStopAutoStakingMetadata_ValidateSanityData(t *testing.T) {
	type fields struct {
		MetadataBase       metadata.MetadataBase
		CommitteePublicKey string
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

	chainBase58CheckDeserializeError := &mocks.ChainRetriever{}
	chainBase58CheckDeserializeError.On("GetBurningAddress", uint64(0)).Return("15pABFiJVeh9D5uiipQxBdSVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs")
	txBase58CheckDeserializeError := &mocks.Transaction{}
	txBase58CheckDeserializeError.On("IsPrivacy").Return(false)
	txBase58CheckDeserializeError.On("GetUniqueReceiver").Return(true, []byte{}, uint64(0))

	chainBurningAddressPublicKeyError := &mocks.ChainRetriever{}
	chainBurningAddressPublicKeyError.On("GetBurningAddress", uint64(0)).Return("12RxahVABnAVCGP3LGwCn8jkQxgw7z1x14wztHzn455TTVpi1wBq9YGwkRMQg3J4e657AbAnCvYCJSdA9czBUNuCKwGSRQt55Xwz8WA")
	txBurningAddressPublicKeyError := &mocks.Transaction{}
	txBurningAddressPublicKeyError.On("IsPrivacy").Return(false)
	txBurningAddressPublicKeyError.On("GetUniqueReceiver").Return(true, []byte{99, 183, 246, 161, 68, 172, 228, 222, 153, 9, 172, 39, 208, 245, 167, 79, 11, 2, 114, 65, 241, 69, 85, 40, 193, 104, 199, 79, 70, 4, 53, 0}, uint64(0))

	chainStopAutoStakingMetadataError := &mocks.ChainRetriever{}
	chainStopAutoStakingMetadataError.On("GetBurningAddress", uint64(0)).Return("12RxahVABnAVCGP3LGwCn8jkQxgw7z1x14wztHzn455TTVpi1wBq9YGwkRMQg3J4e657AbAnCvYCJSdA9czBUNuCKwGSRQt55Xwz8WA")
	txStopAutoStakingMetadataError := &mocks.Transaction{}
	txStopAutoStakingMetadataError.On("IsPrivacy").Return(false)
	txStopAutoStakingMetadataError.On("GetUniqueReceiver").Return(true, []byte{127, 76, 149, 36, 97, 166, 59, 24, 204, 39, 108, 209, 42, 199, 106, 173, 88, 95, 221, 184, 142, 215, 198, 51, 10, 150, 125, 89, 73, 86, 24, 0}, uint64(0))

	txStopAutoStakingMetadataError1 := &mocks.Transaction{}
	txStopAutoStakingMetadataError1.On("IsPrivacy").Return(false)
	txStopAutoStakingMetadataError1.On("GetUniqueReceiver").Return(true, []byte{127, 76, 149, 36, 97, 166, 59, 24, 204, 39, 108, 209, 42, 199, 106, 173, 88, 95, 221, 184, 142, 215, 198, 51, 10, 150, 125, 89, 73, 86, 24, 0}, uint64(1))

	chainHappyCase := &mocks.ChainRetriever{}
	chainHappyCase.On("GetBurningAddress", uint64(0)).Return("15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs")
	txHappyCase := &mocks.Transaction{}
	txHappyCase.On("IsPrivacy").Return(false)
	txHappyCase.On("GetUniqueReceiver").Return(true, []byte{99, 183, 246, 161, 68, 172, 228, 222, 153, 9, 172, 39, 208, 245, 167, 79, 11, 2, 114, 65, 241, 69, 85, 40, 193, 104, 199, 79, 70, 4, 53, 0}, uint64(0))

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		want1   bool
		wantErr bool
	}{
		{
			name: "tx.IsPrivacy error",
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
				chainRetriever: chainBase58CheckDeserializeError,
			},
			want:    false,
			want1:   false,
			wantErr: true,
		},
		{
			name: "stopAutoStakingMetadata check burning address error",
			fields: fields{
				MetadataBase: metadata.MetadataBase{metadata.StopAutoStakingMeta},
			},
			args: args{
				tx:             txBurningAddressPublicKeyError,
				chainRetriever: chainBurningAddressPublicKeyError,
			},
			wantErr: true,
		},
		{
			name: "stopAutoStakingMetadata type error",
			fields: fields{
				MetadataBase: metadata.MetadataBase{metadata.ShardStakingMeta},
			},
			args: args{
				tx:             txStopAutoStakingMetadataError,
				chainRetriever: chainStopAutoStakingMetadataError,
			},
			wantErr: true,
		},
		{
			name: "stopAutoStakingMetadata amount error",
			fields: fields{
				MetadataBase: metadata.MetadataBase{metadata.StopAutoStakingMeta},
			},
			args: args{
				tx:             txStopAutoStakingMetadataError1,
				chainRetriever: chainStopAutoStakingMetadataError,
			},
			wantErr: true,
		},
		{
			name: "CommitteePublicKey.FromString error",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6Afv9xkLDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				chainRetriever: chainHappyCase,
				tx:             txHappyCase,
			},
			wantErr: true,
		},
		{
			name: "CommitteePublicKey.CheckSanityData() error",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "1hm766APBSXcyDbNbPLbb65Hm2DkK35RJp1cwYx95mFExK3VAkE9qfzDJLTKTMiKbscm4zns5QuDpGS4yc5Hi994G1BVVE2hdLgoNJbvxXdbmsRdrwVCENVYJhYk2k1kci7b8ysb9nFXW8fUEJNsBtfQjtXQY7pEqngbwpEFuF45Kj8skjDriKp2Sc9TjxnPw4478dN4h4XYojPaiSo3sJpqJWDfcZ68DqSWuUAud5REAqeBT3sUiyJCpnfZ9Lp2Uk7M7Pc9CeuTZBVfV3M669zpPdErUgWf7VDYe5wujvcMLhqqjvJRe5WREYLjVni1H1d4qhcuzdbPdW8BC4b7xY2qRSBtiFav8tJt7iSdycTeTTsaYN1",
			},
			args: args{
				chainRetriever: chainHappyCase,
				tx:             txHappyCase,
			},
			wantErr: true,
		},
		{
			name: "happy case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				chainRetriever: chainHappyCase,
				tx:             txHappyCase,
			},
			want:    true,
			want1:   true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopAutoStakingMetadata := metadata.StopAutoStakingMetadata{
				MetadataBase:       tt.fields.MetadataBase,
				CommitteePublicKey: tt.fields.CommitteePublicKey,
			}
			got, got1, err := stopAutoStakingMetadata.ValidateSanityData(tt.args.chainRetriever, tt.args.shardRetriever, tt.args.beaconRetriever, tt.args.beaconHeight, tt.args.tx)
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

func TestStopAutoStakingMetadata_ValidateTxWithBlockChain(t *testing.T) {
	type fields struct {
		MetadataBase       metadata.MetadataBase
		CommitteePublicKey string
	}
	type args struct {
		tx                  metadata.Transaction
		chainRetriever      metadata.ChainRetriever
		shardViewRetriever  metadata.ShardViewRetriever
		beaconViewRetriever metadata.BeaconViewRetriever
		shardID             byte
		stateDB             *statedb.StateDB
	}

	var stopStakeTx1Meta metadata.Metadata
	stopStakeTx1Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[1],
	}
	stopStakeTx1 := &mocks.Transaction{}
	stopStakeTx1.On("GetMetadata").Return(stopStakeTx1Meta)
	stopStakeTx1.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx1.On("GetSender").Return([]byte("12buoC8Nmh8WbPhSAiF1SSNB8AuxTu3QbX3sSUydqod4y9ws3e3"))

	chain1 := &mocks.ChainRetriever{}
	beacon1 := &mocks.BeaconViewRetriever{}
	shard1 := &mocks.ShardViewRetriever{}
	beacon1.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").Return([]string{}, errors.New("get error"))

	var stopStakeTx2Meta metadata.Metadata
	stopStakeTx2Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[2],
	}
	stopStakeTx2 := &mocks.Transaction{}
	stopStakeTx2.On("GetMetadata").Return(stopStakeTx2Meta)
	stopStakeTx2.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx2.On("GetSender").Return([]byte("12buoC8Nmh8WbPhSAiF1SSNB8AuxTu3QbX3sSUydqod4y9ws3e3"))

	chain2 := &mocks.ChainRetriever{}
	beacon2 := &mocks.BeaconViewRetriever{}
	shard2 := &mocks.ShardViewRetriever{}
	beacon2.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").Return([]string{""}, nil)
	emptyMap := make(map[string]string)
	shard2.On("GetStakingTx").Return(emptyMap)

	chain21 := &mocks.ChainRetriever{}
	beacon21 := &mocks.BeaconViewRetriever{}
	shard21 := &mocks.ShardViewRetriever{}
	beacon21.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").Return([]string{"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"}, nil)
	shard21.On("GetStakingTx").Return(emptyMap)

	chain3 := &mocks.ChainRetriever{}
	beacon3 := &mocks.BeaconViewRetriever{}
	shard3 := &mocks.ShardViewRetriever{}
	beacon3.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").Return([]string{"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"}, nil)
	stakingMap := make(map[string]string)
	stakingMap["121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"] = "asdasdasdas"
	shard3.On("GetStakingTx").Return(stakingMap)

	stakingTxHash, _ := common.Hash{}.NewHashFromStr("9648b1f460d853d878a3b7ab7a926acab5f45c726de0610221f78a95f333c6dc")
	chain4 := &mocks.ChainRetriever{}
	beacon4 := &mocks.BeaconViewRetriever{}
	shard4 := &mocks.ShardViewRetriever{}
	beacon4.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").Return([]string{"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"}, nil)
	stakingMap1 := make(map[string]string)
	stakingMap1["121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"] = "9648b1f460d853d878a3b7ab7a926acab5f45c726de0610221f78a95f333c6dc"
	shard4.On("GetStakingTx").Return(stakingMap1)
	stakingTxHash2, _ := common.Hash{}.NewHashFromStr("9648b1f460d853d878a3b7ab7a926acab5f45c726de0610221f78a95f333c6dc")
	chain4.On("GetTransactionByHash", *stakingTxHash2).Return(byte(0), nil, uint64(0), int(0), nil, errors.New("error"))

	stopStakeTx3 := &mocks.Transaction{}
	stopStakeTx3.On("GetMetadata").Return(stopStakeTx2Meta)
	stopStakeTx3.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx3.On("GetSender").Return([]byte("12buoC8Nmh8WbPhSAiF1SSNB8AuxZu3QbX3sSUydqod4y9ws3e3"))
	chain5 := &mocks.ChainRetriever{}
	beacon5 := &mocks.BeaconViewRetriever{}
	shard5 := &mocks.ShardViewRetriever{}
	beacon5.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").Return([]string{"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"}, nil)
	shard5.On("GetStakingTx").Return(stakingMap1)
	chain5.On("GetTransactionByHash", *stakingTxHash).Return(byte(0), nil, uint64(0), int(0), stopStakeTx3, nil)
	autoStakingList := make(map[string]bool)
	beacon5.On("GetAutoStakingList").Return(autoStakingList)

	stopStakeTx4 := &mocks.Transaction{}
	stopStakeTx4.On("GetSender").Return([]byte("12buoC8Nmh8WbPhSAiF1SSNB8AuxTu3QbX3sSUydqod4y9ws3e3"))
	chain6 := &mocks.ChainRetriever{}
	beacon6 := &mocks.BeaconViewRetriever{}
	shard6 := &mocks.ShardViewRetriever{}
	beacon6.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").
		Return([]string{"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"}, nil).
		Once()
	shard6.On("GetStakingTx").Return(stakingMap1)
	chain6.On("GetTransactionByHash", *stakingTxHash).Return(byte(0), nil, uint64(0), int(0), stopStakeTx4, nil)
	autoStakingList1 := make(map[string]bool)
	beacon6.On("GetAutoStakingList").Return(autoStakingList1)

	chain7 := &mocks.ChainRetriever{}
	beacon7 := &mocks.BeaconViewRetriever{}
	shard7 := &mocks.ShardViewRetriever{}
	beacon7.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").
		Return([]string{"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"}, nil).
		Once()
	shard7.On("GetStakingTx").Return(stakingMap1)
	chain7.On("GetTransactionByHash", *stakingTxHash).Return(byte(0), nil, uint64(0), int(0), stopStakeTx4, nil)
	autoStakingList2 := make(map[string]bool)
	autoStakingList2["121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"] = false
	beacon7.On("GetAutoStakingList").Return(autoStakingList2)

	chain8 := &mocks.ChainRetriever{}
	beacon8 := &mocks.BeaconViewRetriever{}
	shard8 := &mocks.ShardViewRetriever{}
	beacon8.On("GetAllCommitteeValidatorCandidateFlattenListFromDatabase").
		Return([]string{"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"}, nil).
		Once()
	shard8.On("GetStakingTx").Return(stakingMap1)
	chain8.On("GetTransactionByHash", *stakingTxHash).Return(byte(0), nil, uint64(0), int(0), stopStakeTx4, nil)
	autoStakingList3 := make(map[string]bool)
	autoStakingList3["121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc"] = true
	beacon8.On("GetAutoStakingList").Return(autoStakingList3)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:   "check GetAllCommitteeValidatorCandidateFlattenListFromDatabase error case",
			fields: fields{},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain1,
				beaconViewRetriever: beacon1,
				shardViewRetriever:  shard1,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "check common.IndexOfStr error case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "",
			},
			args: args{
				tx:                  stopStakeTx2,
				chainRetriever:      chain2,
				beaconViewRetriever: beacon2,
				shardViewRetriever:  shard2,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "check stakingTx[requestedPublicKey] error case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain21,
				beaconViewRetriever: beacon21,
				shardViewRetriever:  shard21,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "check common.Hash{}.NewHashFromStr() error case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain3,
				beaconViewRetriever: beacon3,
				shardViewRetriever:  shard3,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "check chainRetriever.GetTransactionByHash() error case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain4,
				beaconViewRetriever: beacon4,
				shardViewRetriever:  shard4,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "check bytes.Equal(stakingTx.GetSender(), tx.GetSender()) error case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain5,
				beaconViewRetriever: beacon5,
				shardViewRetriever:  shard5,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "check autoStakingList[stopStakingMetadata.CommitteePublicKey] error case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain6,
				beaconViewRetriever: beacon6,
				shardViewRetriever:  shard6,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "check !isAutoStaking error case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain7,
				beaconViewRetriever: beacon7,
				shardViewRetriever:  shard7,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "happy case",
			fields: fields{
				MetadataBase:       metadata.MetadataBase{metadata.StopAutoStakingMeta},
				CommitteePublicKey: "121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
			},
			args: args{
				tx:                  stopStakeTx1,
				chainRetriever:      chain8,
				beaconViewRetriever: beacon8,
				shardViewRetriever:  shard8,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopAutoStakingMetadata := metadata.StopAutoStakingMetadata{
				MetadataBase:       tt.fields.MetadataBase,
				CommitteePublicKey: tt.fields.CommitteePublicKey,
			}
			got, err := stopAutoStakingMetadata.ValidateTxWithBlockChain(tt.args.tx, tt.args.chainRetriever, tt.args.shardViewRetriever, tt.args.beaconViewRetriever, tt.args.shardID, tt.args.stateDB)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTxWithBlockChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateTxWithBlockChain() got = %v, want %v", got, tt.want)
			}
		})
	}
}
