package blockchain

import (
	"github.com/incognitochain/incognito-chain/multiview"
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/metadata/mocks"
)

var (
	validPrivateKeys = []string{
		"112t8rq19Uu7UGbTApZzZwCAvVszAgRNAzHzr3p8Cu75jPH3h5AUtRXMKiqF3hw8NbEfeLcjtbpeUvJfw4tGj7pbqwDYngc8wB13Gf77o33f",
		"112t8rrEW3NPNgU8xzbeqE7cr4WTT8JvyaQqSZyczA5hBJVvpQMTBVqNfcCdzhvquWCHH11jHihZtgyJqbdWPhWYbmmsw5aV29WSXBEsgbVX",
		"112t8rnky34vDfgZp4MizVfNS1EttVucnwFPL4ULFwtiS4nwvEJryyWASWG5MrVv5hvFn6nAqMqcVorguTjKjiHJmjD55jCTvEjWPEUmnMP1",
	}
	validCommitteePublicKeys = []string{
		"121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7",
		"121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc",
		"121VhftSAygpEJZ6i9jGkRSiFCZCAU54hEi1ZgW1insUMxeB4DgKpTZKejgVi2D7ENHC6XfRwsAcyiEaeiuis9XRU7YTMXUUGi29SzByMnXfGVRsGAb2hew9W32QMi23QDvYjoSVgUH6rSdWX9wGaPyaUV9SoyHng63Ee9zDc8AVFv1xgqbKNE7BquQzYR22j3AypirG2MmYDSUMLe2HJHBkF9Y7UphmFABNeVKhtZTXVQP78SKpfrEHigg4Gzm595EGFWLLekn6Gcs9HZb7B6gusrMfYbACsRSbCXZ6UcpaYEDx91xReAE3SDktmUHdLh2U7JhJpxgKXK4jjtjNbXwjFAbJqi1eATG8oCA2tEtaubNB9aDQMJjnK5if9KUbt92RGk4d94Ff9Gnr9CG7jVFTfem8UNUzK8KiXHvumziwaoiX",
	}
	validCommitteePublicKeyStructs = []incognitokey.CommitteePublicKey{}
	validPaymentAddresses          = []string{
		"12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX",
		"12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC",
		"12S2xCenBEHuyyZQ3VVqfMUvEEwcKL1UawNEkfSX8BL8HpPwSPu3yaYptvRYfuPzr1GUsyGBtUoet5B6VT1nGMLL8xTErYgZr6uuY52",
	}
	validTxHashStr = []string{
		"a9ab827f18423dc1ee50450020cc671ae32d95246a3ed0afc104928d3bbb3f3c",
		"b14366299ab7c701dba456893066705b3e7fa337073402bc98fa5aaa80b5264e",
		"e3fe35c4a116632a8b700c93809215626db7efda04d7ba2dcb35f7f414c5dbeb",
	}
)

func TestCreateShardInstructionsFromTransactionAndInstruction(t *testing.T) {
	type args struct {
		transactions []metadata.Transaction
		bc           *BlockChain
		shardID      byte
	}

	validTxHash := []*common.Hash{}
	for _, txHashStr := range validTxHashStr {
		txHash, _ := common.Hash{}.NewHashFromStr(txHashStr)
		validTxHash = append(validTxHash, txHash)
	}

	happyCaseBC := &BlockChain{
		BeaconChain: &BeaconChain{
			multiView: multiview.NewMultiView(),
		},
	}

	//staking happy case args start
	stakingTx1 := &mocks.Transaction{}
	var stakingTx1Meta metadata.Metadata
	stakingTx1Meta = &metadata.StakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.ShardStakingMeta,
		},
		FunderPaymentAddress:         validPaymentAddresses[0],
		RewardReceiverPaymentAddress: validPaymentAddresses[0],
		StakingAmountShard:           1750000000000,
		AutoReStaking:                false,
		CommitteePublicKey:           validCommitteePublicKeys[0],
	}
	stakingTx1.On("GetMetadataType").Return(metadata.ShardStakingMeta)
	stakingTx1.On("GetMetadata").Once().Return(nil)
	stakingTx1.On("GetMetadata").Twice().Return(stakingTx1Meta)
	stakingTx1.On("GetMetadata").Times(3).Return(stakingTx1Meta)

	stakingTx1.On("Hash").Return(validTxHash[0])

	stakingTx2 := &mocks.Transaction{}
	var stakingTx2Meta metadata.Metadata
	stakingTx2Meta = &metadata.StakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.ShardStakingMeta,
		},
		FunderPaymentAddress:         validPaymentAddresses[1],
		RewardReceiverPaymentAddress: validPaymentAddresses[1],
		StakingAmountShard:           1750000000000,
		AutoReStaking:                true,
		CommitteePublicKey:           validCommitteePublicKeys[1],
	}
	stakingTx2.On("GetMetadataType").Return(metadata.ShardStakingMeta)
	stakingTx2.On("GetMetadata").Once().Return(nil)
	stakingTx2.On("GetMetadata").Twice().Return(stakingTx2Meta)
	stakingTx2.On("GetMetadata").Times(3).Return(stakingTx2Meta)
	stakingTx2.On("Hash").Return(validTxHash[1])

	stakingTx3 := &mocks.Transaction{}
	var stakingTx3Meta metadata.Metadata
	stakingTx3Meta = &metadata.StakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.ShardStakingMeta,
		},
		FunderPaymentAddress:         validPaymentAddresses[2],
		RewardReceiverPaymentAddress: validPaymentAddresses[2],
		StakingAmountShard:           1750000000000,
		AutoReStaking:                false,
		CommitteePublicKey:           validCommitteePublicKeys[2],
	}
	stakingTx3.On("GetMetadataType").Return(metadata.ShardStakingMeta)
	stakingTx3.On("GetMetadata").Once().Return(nil)
	stakingTx3.On("GetMetadata").Twice().Return(stakingTx3Meta)
	stakingTx3.On("GetMetadata").Times(3).Return(stakingTx3Meta)
	stakingTx3.On("Hash").Return(validTxHash[2])
	//staking happy case args end

	//stop auto staking case args start
	stopStakeTx1 := &mocks.Transaction{}
	var stopStakeTx1Meta metadata.Metadata
	stopStakeTx1Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[0],
	}
	stopStakeTx1.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx1.On("GetMetadata").Once().Return(nil)
	stopStakeTx1.On("GetMetadata").Twice().Return(stopStakeTx1Meta)
	stopStakeTx1.On("GetMetadata").Times(3).Return(stopStakeTx1Meta)

	stopStakeTx2 := &mocks.Transaction{}
	var stopStakeTx2Meta metadata.Metadata
	stopStakeTx2Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[1],
	}
	stopStakeTx2.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx2.On("GetMetadata").Once().Return(nil)
	stopStakeTx2.On("GetMetadata").Twice().Return(stopStakeTx2Meta)
	stopStakeTx2.On("GetMetadata").Times(3).Return(stopStakeTx2Meta)

	stopStakeTx3 := &mocks.Transaction{}
	var stopStakeTx3Meta metadata.Metadata
	stopStakeTx3Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[2],
	}
	stopStakeTx3.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx3.On("GetMetadata").Once().Return(nil)
	stopStakeTx3.On("GetMetadata").Twice().Return(stopStakeTx3Meta)
	stopStakeTx3.On("GetMetadata").Times(3).Return(stopStakeTx3Meta) //staking happy case args start

	stakingTx4 := &mocks.Transaction{}
	var stakingTx4Meta metadata.Metadata
	stakingTx4Meta = &metadata.StakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.ShardStakingMeta,
		},
		FunderPaymentAddress:         validPaymentAddresses[0],
		RewardReceiverPaymentAddress: validPaymentAddresses[0],
		StakingAmountShard:           1750000000000,
		AutoReStaking:                false,
		CommitteePublicKey:           validCommitteePublicKeys[0],
	}
	stakingTx4.On("GetMetadataType").Return(metadata.ShardStakingMeta)
	stakingTx4.On("GetMetadata").Once().Return(nil)
	stakingTx4.On("GetMetadata").Twice().Return(stakingTx4Meta)
	stakingTx4.On("GetMetadata").Times(4).Return(stakingTx4Meta)
	stakingTx4.On("Hash").Return(validTxHash[0])

	stakingTx5 := &mocks.Transaction{}
	var stakingTx5Meta metadata.Metadata
	stakingTx5Meta = &metadata.StakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.ShardStakingMeta,
		},
		FunderPaymentAddress:         validPaymentAddresses[1],
		RewardReceiverPaymentAddress: validPaymentAddresses[1],
		StakingAmountShard:           1750000000000,
		AutoReStaking:                true,
		CommitteePublicKey:           validCommitteePublicKeys[1],
	}
	stakingTx5.On("GetMetadataType").Return(metadata.ShardStakingMeta)
	stakingTx5.On("GetMetadata").Once().Return(nil)
	stakingTx5.On("GetMetadata").Twice().Return(stakingTx5Meta)
	stakingTx5.On("GetMetadata").Times(3).Return(stakingTx5Meta)
	stakingTx5.On("Hash").Return(validTxHash[1])

	stakingTx6 := &mocks.Transaction{}
	var stakingTx6Meta metadata.Metadata
	stakingTx6Meta = &metadata.StakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.ShardStakingMeta,
		},
		FunderPaymentAddress:         validPaymentAddresses[2],
		RewardReceiverPaymentAddress: validPaymentAddresses[2],
		StakingAmountShard:           1750000000000,
		AutoReStaking:                false,
		CommitteePublicKey:           validCommitteePublicKeys[2],
	}
	stakingTx6.On("GetMetadataType").Return(metadata.ShardStakingMeta)
	stakingTx6.On("GetMetadata").Once().Return(nil)
	stakingTx6.On("GetMetadata").Twice().Return(stakingTx6Meta)
	stakingTx6.On("GetMetadata").Times(3).Return(stakingTx6Meta)
	stakingTx6.On("Hash").Return(validTxHash[2])
	//staking happy case args end

	//stop auto staking case args start
	stopStakeTx4 := &mocks.Transaction{}
	var stopStakeTx4Meta metadata.Metadata
	stopStakeTx4Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[0],
	}
	stopStakeTx4.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx4.On("GetMetadata").Once().Return(nil)
	stopStakeTx4.On("GetMetadata").Twice().Return(stopStakeTx4Meta)
	stopStakeTx4.On("GetMetadata").Times(3).Return(stopStakeTx4Meta)

	stopStakeTx5 := &mocks.Transaction{}
	var stopStakeTx5Meta metadata.Metadata
	stopStakeTx5Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[1],
	}
	stopStakeTx5.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx5.On("GetMetadata").Once().Return(nil)
	stopStakeTx5.On("GetMetadata").Twice().Return(stopStakeTx5Meta)
	stopStakeTx5.On("GetMetadata").Times(3).Return(stopStakeTx5Meta)

	stopStakeTx6 := &mocks.Transaction{}
	var stopStakeTx6Meta metadata.Metadata
	stopStakeTx6Meta = &metadata.StopAutoStakingMetadata{
		MetadataBase: metadata.MetadataBase{
			metadata.StopAutoStakingMeta,
		},
		CommitteePublicKey: validCommitteePublicKeys[2],
	}
	stopStakeTx6.On("GetMetadataType").Return(metadata.StopAutoStakingMeta)
	stopStakeTx6.On("GetMetadata").Once().Return(nil)
	stopStakeTx6.On("GetMetadata").Twice().Return(stopStakeTx6Meta)
	stopStakeTx6.On("GetMetadata").Times(3).Return(stopStakeTx6Meta)
	//stop auto staking case args end
	tests := []struct {
		name             string
		args             args
		wantInstructions [][]string
		wantErr          bool
	}{
		{
			name: "staking happy case",
			args: args{
				transactions: []metadata.Transaction{
					stakingTx1, stakingTx2, stakingTx3,
				},
				bc:      happyCaseBC,
				shardID: 0,
			},
			wantInstructions: [][]string{{"stake", "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7,121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc,121VhftSAygpEJZ6i9jGkRSiFCZCAU54hEi1ZgW1insUMxeB4DgKpTZKejgVi2D7ENHC6XfRwsAcyiEaeiuis9XRU7YTMXUUGi29SzByMnXfGVRsGAb2hew9W32QMi23QDvYjoSVgUH6rSdWX9wGaPyaUV9SoyHng63Ee9zDc8AVFv1xgqbKNE7BquQzYR22j3AypirG2MmYDSUMLe2HJHBkF9Y7UphmFABNeVKhtZTXVQP78SKpfrEHigg4Gzm595EGFWLLekn6Gcs9HZb7B6gusrMfYbACsRSbCXZ6UcpaYEDx91xReAE3SDktmUHdLh2U7JhJpxgKXK4jjtjNbXwjFAbJqi1eATG8oCA2tEtaubNB9aDQMJjnK5if9KUbt92RGk4d94Ff9Gnr9CG7jVFTfem8UNUzK8KiXHvumziwaoiX", "shard", "a9ab827f18423dc1ee50450020cc671ae32d95246a3ed0afc104928d3bbb3f3c,b14366299ab7c701dba456893066705b3e7fa337073402bc98fa5aaa80b5264e,e3fe35c4a116632a8b700c93809215626db7efda04d7ba2dcb35f7f414c5dbeb", "12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX,12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC,12S2xCenBEHuyyZQ3VVqfMUvEEwcKL1UawNEkfSX8BL8HpPwSPu3yaYptvRYfuPzr1GUsyGBtUoet5B6VT1nGMLL8xTErYgZr6uuY52", "false,true,false"}},
			wantErr:          false,
		},
		{
			name: "stop auto stake happy case",
			args: args{
				transactions: []metadata.Transaction{
					stopStakeTx1, stopStakeTx2, stopStakeTx3,
				},
				bc:      happyCaseBC,
				shardID: 0,
			},
			wantInstructions: [][]string{{"stopautostake", "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7,121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc,121VhftSAygpEJZ6i9jGkRSiFCZCAU54hEi1ZgW1insUMxeB4DgKpTZKejgVi2D7ENHC6XfRwsAcyiEaeiuis9XRU7YTMXUUGi29SzByMnXfGVRsGAb2hew9W32QMi23QDvYjoSVgUH6rSdWX9wGaPyaUV9SoyHng63Ee9zDc8AVFv1xgqbKNE7BquQzYR22j3AypirG2MmYDSUMLe2HJHBkF9Y7UphmFABNeVKhtZTXVQP78SKpfrEHigg4Gzm595EGFWLLekn6Gcs9HZb7B6gusrMfYbACsRSbCXZ6UcpaYEDx91xReAE3SDktmUHdLh2U7JhJpxgKXK4jjtjNbXwjFAbJqi1eATG8oCA2tEtaubNB9aDQMJjnK5if9KUbt92RGk4d94Ff9Gnr9CG7jVFTfem8UNUzK8KiXHvumziwaoiX"}},
			wantErr:          false,
		},
		{
			name: "staking & stop auto stake happy case",
			args: args{
				transactions: []metadata.Transaction{
					stakingTx4, stakingTx5, stakingTx6, stopStakeTx4, stopStakeTx5, stopStakeTx6,
				},
				bc:      happyCaseBC,
				shardID: 0,
			},
			wantInstructions: [][]string{{"stake", "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7,121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc,121VhftSAygpEJZ6i9jGkRSiFCZCAU54hEi1ZgW1insUMxeB4DgKpTZKejgVi2D7ENHC6XfRwsAcyiEaeiuis9XRU7YTMXUUGi29SzByMnXfGVRsGAb2hew9W32QMi23QDvYjoSVgUH6rSdWX9wGaPyaUV9SoyHng63Ee9zDc8AVFv1xgqbKNE7BquQzYR22j3AypirG2MmYDSUMLe2HJHBkF9Y7UphmFABNeVKhtZTXVQP78SKpfrEHigg4Gzm595EGFWLLekn6Gcs9HZb7B6gusrMfYbACsRSbCXZ6UcpaYEDx91xReAE3SDktmUHdLh2U7JhJpxgKXK4jjtjNbXwjFAbJqi1eATG8oCA2tEtaubNB9aDQMJjnK5if9KUbt92RGk4d94Ff9Gnr9CG7jVFTfem8UNUzK8KiXHvumziwaoiX", "shard", "a9ab827f18423dc1ee50450020cc671ae32d95246a3ed0afc104928d3bbb3f3c,b14366299ab7c701dba456893066705b3e7fa337073402bc98fa5aaa80b5264e,e3fe35c4a116632a8b700c93809215626db7efda04d7ba2dcb35f7f414c5dbeb", "12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX,12RrjUWjyCNPXoCChrpEVLxucs3WEw9KyFxzP3UrdRzped2UouDzBM9gNugySqt4RpmgkqL1H7xxE8PfNmDwAatnSXPUVdNomBK1yYC,12S2xCenBEHuyyZQ3VVqfMUvEEwcKL1UawNEkfSX8BL8HpPwSPu3yaYptvRYfuPzr1GUsyGBtUoet5B6VT1nGMLL8xTErYgZr6uuY52", "false,true,false"}, {"stopautostake", "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7,121VhftSAygpEJZ6i9jGkCFHRkD4yhxxccAqVjQTWR9gy7skM1KcNf3uGLpX1NvojmHqs9bWwsPfvyBmer39YNBPwBHpgXg1Qku4EDhtUBZnGw2PZGMF7DMCrYa27GNS97uA9WC5z55YuCDA4WsnKfoEEuCFDNUN3iSCeUyrQ4SF5smx9CwBYX6AWAMAvNDPKf4tCuc7Wiafv9xkLKuHSFr7jaxBfg4rdaxtwXzR5eMpFDDpiXz6hQmdcee8xSXQRKceiafg9RMiuqLxDzx9tmLKvBD5TJq4G76LB3rrVmsYwMo1fY4RZLpiYn6AstAfca5EVnMeexueSAE5sam3Lsq8mq5poJfsW6KXzAbsmFPSsSjhmQ4wGhSXoKSap331gBMuuy7KtmVwQAPpwuFPo9hi7RBgrrn1ssdCdjYSwE226Ekc,121VhftSAygpEJZ6i9jGkRSiFCZCAU54hEi1ZgW1insUMxeB4DgKpTZKejgVi2D7ENHC6XfRwsAcyiEaeiuis9XRU7YTMXUUGi29SzByMnXfGVRsGAb2hew9W32QMi23QDvYjoSVgUH6rSdWX9wGaPyaUV9SoyHng63Ee9zDc8AVFv1xgqbKNE7BquQzYR22j3AypirG2MmYDSUMLe2HJHBkF9Y7UphmFABNeVKhtZTXVQP78SKpfrEHigg4Gzm595EGFWLLekn6Gcs9HZb7B6gusrMfYbACsRSbCXZ6UcpaYEDx91xReAE3SDktmUHdLh2U7JhJpxgKXK4jjtjNbXwjFAbJqi1eATG8oCA2tEtaubNB9aDQMJjnK5if9KUbt92RGk4d94Ff9Gnr9CG7jVFTfem8UNUzK8KiXHvumziwaoiX"}},
			wantErr:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInstructions, err := CreateShardInstructionsFromTransactionAndInstruction(tt.args.transactions, tt.args.bc, tt.args.shardID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShardInstructionsFromTransactionAndInstruction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotInstructions, tt.wantInstructions) {
				t.Errorf("CreateShardInstructionsFromTransactionAndInstruction() gotInstructions = %v, want %v", gotInstructions, tt.wantInstructions)
			}
		})
	}
}
