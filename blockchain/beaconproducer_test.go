package blockchain

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain/committeestate"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/jrick/logrotate/rotator"
	"github.com/stretchr/testify/assert"
)

// initLogRotator initializes the logging rotater to write logs to logFile and
// create roll files in the same directory.  It must be called before the
// package-global log rotater variables are used.
func initLogRotator(logFile string) {
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(common.ExitByLogging)
	}
	r, err := rotator.New(logFile, 10*1024, false, 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(common.ExitByLogging)
	}

	logRotator = r
}

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

var logRotator *rotator.Rotator

func (logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	logRotator.Write(p)
	return len(p), nil
}

func initLog() {
	initLogRotator("./blockchain.log")
	blockchainLogger := common.NewBackend(logWriter{}).Logger("Blockchain log ", false)
	Logger.Init(blockchainLogger)
}

var (
	key0                                                         = "121VhftSAygpEJZ6i9jGkCVwX5W7tZY6McnoXxZ3xArZQcKduS78P6F6B6T8sjNkoxN7pfjJruViCG3o4X5CiEtHCv9Ufnqp7W3qB9WkuSbGnEKtsNNGpHxJEpdEw4saeueY6kRhqFDcF2NQjgocGLyZsc5Ea6KPBj56kMtUtfcois8pBuFPn2udAsSza7HpkiW7e9kYmzu6Nqnca2jPc8ugCJYHsQDtjmzENC1tje2dfFzCnfkHqam8342bF2wEJgiEwTkkZBY2uLkbQT2X39tSsfzmbqjfrEExjorhFA5yx2ZpKrsA4H9sE34Khy8RradfGCK4L6J4gz1G7YQJ1v2hihEsw3D2fp5ktUh46sicTLmTQ2sfzjnNgMq5uAZ2cJx3HeNiETJ65RVR9J71ujLzdw8xDZvbAPRsdB11Hj2KgKFR"
	key                                                          = "121VhftSAygpEJZ6i9jGkEKLMQTKTiiHzeUfeuhpQCcLZtys8FazpWwytpHebkAwgCxvqgUUF13fcSMtp5dgV1YkbRMj3z42TW2EebzAaiGg2DkGPodckN2UsbqhVDibpMgJUHVkLXardemfLdgUqWGtymdxaaRyPM38BAZcLpo2pAjxKv5vG5Uh9zHMkn7ZHtdNHmBmhG8B46UeiGBXYTwhyMe9KGS83jCMPAoUwHhTEXj5qQh6586dHjVxwEkRzp7SKn9iG1FFWdJ97xEkP2ezAapNQ46quVrMggcHFvoZofs1xdd4o5vAmPKnPTZtGTKunFiTWGnpSG9L6r5QpcmapqvRrK5SiuFhNM5DqgzUeHBb7fTfoiWd2N29jkbTGSq8CPUSjx3zdLR9sZguvPdnAA8g25cFPGSZt8aEnFJoPRzM"
	key2                                                         = "121VhftSAygpEJZ6i9jGkEqPGAXcmKffwMbzpwxnEfzJxen4oZKPukWAUBbqvV5xPnowZ2eQmAj2mEebG2oexebQPh1MPFC6vEZAk6i7AiRPrZmfaRrRVrBp4WXnVJmL3xK4wzTfkR2rZkhUmSZm112TTyhDNkDQSaBGJkexrPbryqUygazCA2eyo6LnK5qs7jz2RhhsWqUTQ3sQJUuFcYdf2pSnYwhqZqphDCSRizDHeysaua5L7LwS8fY7KZHhPgTuFjvUWWnWSRTmV8u1dTY5kcmMdDZsPiyN9WfqjgVoTFNALjFG8U4GMvzV3kKwVVjuPMsM2XqyPDVpdNQUgLnv2bJS8Tr22A9NgF1FQfWyAny1DYyY3N5H3tfCggsybzZXzrbYPPgokvEynac91y8hPkRdgKW1e7FHzuBnEisPuKzy"
	key3                                                         = "121VhftSAygpEJZ6i9jGkGLcYhJBeaJTGY5aFjqQA2WwyxU69Utrviuy9AJ3ATkeEyigVGScQUZw22cD1HeFKiyASYAs82WEamujt3nefYA9FPhURBpRTn6jDmGKUdb4QNbs7HVCJkRRaL9aktg1yaQaZE8TJFg2UeE9tBqUdmvD8fy36aDCYM5W86jaTVCXeEJQWPxUunP2EEL3e283PJ8zqPeBkpoFvkvhB28Hk3oRDeCCTC7QhbaV18ayKeToYqAxoUMBBihanfA33ixeX1daeKpajLCgDZ6jrfphwdYwQbf7dMcZ2NVvQ1a5JUCTJUZypwgKRt8tnTAKCowt2L1KNGP4NJJZm61cfHAGbKRyG9QxCJgK2SdMKsKPVefZSc9LbVaB7VeBby5LHxvMoCD7bN7g1HYRp4BX9n1fZJUeEkVa"
	key4                                                         = "121VhftSAygpEJZ6i9jGkDjJj7e2cfgQvrLsPsmLhGMmGD9U9Knffa1MZAw79EijnpueVfTStN2VYt5jRqEr2DTjVqzUinwHVKWH4Tg4szHUntiBdWeqzNC4E8iiwC9Y2KtcRr3hBkpfqvyuBvchigatrigRvFVWu8H2RQqjvopLL51DQ4LFD87L9Zgj9HhasMeyr6f37yirs47JgtGs4BM7EhhpM5zD3TCsFabPphtwDKnfuLMaGzoAw5fM8zEXvdLMuohk96oayjdYothncdtZom17DxB1Mmw535eEjxBwz9ELoZRKk3LYiheSd4xGN9QsxrT2WnZCTd8B5QktARte5S91QYvRMixKC8UEuovQhXt8jMZNkq7CmMeXoybfYdmNaAHuqbY1QeUT2AgaqPho4ay3z5eeKRhnB28H18RGWQ1L"
	key5                                                         = "121VhftSAygpEJZ6i9jGkS3D5FaqhzhSF79YFYKHLTHMY5erPhm5vT9VxMtFdWbUVfmhKvfKosXiUKsygyw8knbejNWineCFpx35KegXBbfnVv6AcE3KD4Rs46pDKrqDvWmpaPJoUDdiJeVPQQsFuTykMGs1txt14hhnWMWx9Bf8caDpxR3SKQY7PyHbEhRhdasL3eJC3X1y83PkzJistXPHFdoK4bszU5iE8EiMrXP5GiHTLLTyTpRxScg6AVnrFnmRzPsEMXJAz5zmwUkxwGNrj5iBi7ZJBHo5m3bTVYdQqTSBgVXSqAGZ6fPqAXPGkH6NfgGeZhXR33D3Q4JhEoBs4QWnr89gaVUDAwGXFoXEVfmGwGFFy7jPdLYKuc2u1yJ9YRa1MbSxcPLATui2wmN73UAFa6uSUdN71rCDHJEfCxpS"
	key6                                                         = "121VhftSAygpEJZ6i9jGkQXi69eX7p8fmJosf8F4KEdBSqfh3cGxGMd6sGa4hfXTg9vxq16AN97mrqerdNM6ZUGBDgPAipbaGznaHSC8gE7gBpSrVKbNb93nwXSBHSBKFVC6MK5NAFN6bpK25YHrmC248FPr3VQMf9tfG57P5TTH7KWr4bn7v2YbTxNRkZFD9JwkTmwXAwEfWJ12rrc1kMDUkAkrSYYhmpykXTjK9wEBkKFA2z5rnw24cBVL9Tt6M2BEqUM3tuEoUfhiA6E6tdPAkYc7LusTjwikzpwRbVYi4cVMCmC7Dd2UccaA2iiotuyP85AYQSUaHzV2MaF2Cv7GtLqTMm6bRqvpetU1kpkunEnQmAuLVLG7QHPRVKdkX6wRYBE6uRcJ1FaejVbbrF3Tgyh6dsMhRVgEvvvocYPULcJ5"
	incKey0, incKey, incKey2, incKey3, incKey4, incKey5, incKey6 *incognitokey.CommitteePublicKey
	paymentAddreessKey0                                          = "12Rs8bHvYZELqHrv28bYezBQQpteZUEbYjUf2oqV9pJm6Gx4sD4n9mr4UgQe5cDeP9A2x1DsB4mbJ9LT8x2ShaY41cZJWrL7RpFpp2v"
)

//initPublicKey init incognito public key for testing by base 58 string
func initPublicKey() {
	incKey0 = new(incognitokey.CommitteePublicKey)
	incKey = new(incognitokey.CommitteePublicKey)
	incKey2 = new(incognitokey.CommitteePublicKey)
	incKey3 = new(incognitokey.CommitteePublicKey)
	incKey4 = new(incognitokey.CommitteePublicKey)
	incKey5 = new(incognitokey.CommitteePublicKey)
	incKey6 = new(incognitokey.CommitteePublicKey)

	err := incKey.FromBase58(key)
	if err != nil {
		panic(err)
	}

	err = incKey2.FromBase58(key2)
	if err != nil {
		panic(err)
	}

	err = incKey3.FromBase58(key3)
	if err != nil {
		panic(err)
	}

	err = incKey4.FromBase58(key4)
	if err != nil {
		panic(err)
	}

	err = incKey5.FromBase58(key5)
	if err != nil {
		panic(err)
	}

	err = incKey6.FromBase58(key6)
	if err != nil {
		panic(err)
	}

	err = incKey0.FromBase58(key0)
	if err != nil {
		panic(err)
	}
}

func TestBeaconBestState_preProcessInstructionsFromShardBlock(t *testing.T) {

	initPublicKey()
	initLog()

	tx, _ := common.Hash{}.NewHashFromStr("123")
	paymentAddress0, err := wallet.Base58CheckDeserialize(paymentAddreessKey0)
	assert.Nil(t, err)

	type fields struct {
		BestBlockHash            common.Hash
		PreviousBestBlockHash    common.Hash
		BestBlock                types.BeaconBlock
		BestShardHash            map[byte]common.Hash
		BestShardHeight          map[byte]uint64
		Epoch                    uint64
		BeaconHeight             uint64
		BeaconProposerIndex      int
		CurrentRandomNumber      int64
		CurrentRandomTimeStamp   int64
		IsGetRandomNumber        bool
		Params                   map[string]string
		MaxBeaconCommitteeSize   int
		MinBeaconCommitteeSize   int
		MaxShardCommitteeSize    int
		MinShardCommitteeSize    int
		ActiveShards             int
		ConsensusAlgorithm       string
		ShardConsensusAlgorithm  map[byte]string
		beaconCommitteeEngine    committeestate.BeaconCommitteeEngine
		LastCrossShardState      map[byte]map[byte]uint64
		ShardHandle              map[byte]bool
		NumOfBlocksByProducers   map[string]uint64
		BlockInterval            time.Duration
		BlockMaxCreateTime       time.Duration
		consensusStateDB         *statedb.StateDB
		ConsensusStateDBRootHash common.Hash
		rewardStateDB            *statedb.StateDB
		RewardStateDBRootHash    common.Hash
		featureStateDB           *statedb.StateDB
		FeatureStateDBRootHash   common.Hash
		slashStateDB             *statedb.StateDB
		SlashStateDBRootHash     common.Hash
	}
	type args struct {
		instructions [][]string
		shardID      byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *shardInstruction
	}{
		{
			name:   "Stake Instruction",
			fields: fields{},
			args: args{
				instructions: [][]string{
					[]string{
						instruction.STAKE_ACTION,
						key0,
						instruction.SHARD_INST,
						tx.String(),
						paymentAddreessKey0,
						"true",
					},
				},
			},
			want: &shardInstruction{
				swapInstructions: make(map[byte][]*instruction.SwapInstruction),
				stakeInstructions: []*instruction.StakeInstruction{
					&instruction.StakeInstruction{
						PublicKeyStructs: []incognitokey.CommitteePublicKey{
							*incKey0,
						},
						PublicKeys: []string{key0},
						RewardReceiverStructs: []privacy.PaymentAddress{
							paymentAddress0.KeySet.PaymentAddress,
						},
						AutoStakingFlag: []bool{true},
						TxStakeHashes: []common.Hash{
							*tx,
						},
						RewardReceivers: []string{paymentAddreessKey0},
						Chain:           instruction.SHARD_INST,
						TxStakes:        []string{tx.String()},
					},
				},
			},
		},
		{
			name:   "Swap Instruction",
			fields: fields{},
			args: args{
				instructions: [][]string{
					[]string{
						instruction.SWAP_ACTION,
						key0,
						key,
						instruction.SHARD_INST,
						"0",
						"",
						paymentAddreessKey0,
					},
				},
			},
			want: &shardInstruction{
				swapInstructions: map[byte][]*instruction.SwapInstruction{
					0: []*instruction.SwapInstruction{
						&instruction.SwapInstruction{
							InPublicKeys:        []string{key0},
							InPublicKeyStructs:  []incognitokey.CommitteePublicKey{*incKey0},
							OutPublicKeys:       []string{key},
							OutPublicKeyStructs: []incognitokey.CommitteePublicKey{*incKey},
							ChainID:             0,
							PunishedPublicKeys:  "",
							// this field is only for replace committee
							NewRewardReceivers:       []string{paymentAddreessKey0},
							NewRewardReceiverStructs: []privacy.PaymentAddress{paymentAddress0.KeySet.PaymentAddress},
							IsReplace:                true,
						},
					},
				},
			},
		},
		{
			name:   "Stop AutoStake Instruction",
			fields: fields{},
			args: args{
				instructions: [][]string{
					[]string{
						instruction.STOP_AUTO_STAKE_ACTION,
						key0,
					},
				},
			},
			want: &shardInstruction{
				stopAutoStakeInstructions: []*instruction.StopAutoStakeInstruction{
					instruction.NewStopAutoStakeInstructionWithValue([]string{key0}),
				},
			},
		},
		{
			name:   "Unstake Instruction",
			fields: fields{},
			args: args{
				instructions: [][]string{
					[]string{
						instruction.UNSTAKE_ACTION,
						key0,
					},
				},
			},
			want: &shardInstruction{
				unstakeInstructions: []*instruction.UnstakeInstruction{
					&instruction.UnstakeInstruction{
						CommitteePublicKeys:       []string{key0},
						CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey0},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconBestState := &BeaconBestState{
				BestBlockHash:            tt.fields.BestBlockHash,
				PreviousBestBlockHash:    tt.fields.PreviousBestBlockHash,
				BestBlock:                tt.fields.BestBlock,
				BestShardHash:            tt.fields.BestShardHash,
				BestShardHeight:          tt.fields.BestShardHeight,
				Epoch:                    tt.fields.Epoch,
				BeaconHeight:             tt.fields.BeaconHeight,
				BeaconProposerIndex:      tt.fields.BeaconProposerIndex,
				CurrentRandomNumber:      tt.fields.CurrentRandomNumber,
				CurrentRandomTimeStamp:   tt.fields.CurrentRandomTimeStamp,
				IsGetRandomNumber:        tt.fields.IsGetRandomNumber,
				Params:                   tt.fields.Params,
				MaxBeaconCommitteeSize:   tt.fields.MaxBeaconCommitteeSize,
				MinBeaconCommitteeSize:   tt.fields.MinBeaconCommitteeSize,
				MaxShardCommitteeSize:    tt.fields.MaxShardCommitteeSize,
				MinShardCommitteeSize:    tt.fields.MinShardCommitteeSize,
				ActiveShards:             tt.fields.ActiveShards,
				ConsensusAlgorithm:       tt.fields.ConsensusAlgorithm,
				ShardConsensusAlgorithm:  tt.fields.ShardConsensusAlgorithm,
				beaconCommitteeEngine:    tt.fields.beaconCommitteeEngine,
				LastCrossShardState:      tt.fields.LastCrossShardState,
				ShardHandle:              tt.fields.ShardHandle,
				NumOfBlocksByProducers:   tt.fields.NumOfBlocksByProducers,
				BlockInterval:            tt.fields.BlockInterval,
				BlockMaxCreateTime:       tt.fields.BlockMaxCreateTime,
				consensusStateDB:         tt.fields.consensusStateDB,
				ConsensusStateDBRootHash: tt.fields.ConsensusStateDBRootHash,
				rewardStateDB:            tt.fields.rewardStateDB,
				RewardStateDBRootHash:    tt.fields.RewardStateDBRootHash,
				featureStateDB:           tt.fields.featureStateDB,
				FeatureStateDBRootHash:   tt.fields.FeatureStateDBRootHash,
				slashStateDB:             tt.fields.slashStateDB,
				SlashStateDBRootHash:     tt.fields.SlashStateDBRootHash,
			}
			got := beaconBestState.preProcessInstructionsFromShardBlock(tt.args.instructions, tt.args.shardID)
			for i, v := range got.stakeInstructions {
				if !reflect.DeepEqual(v, tt.want.stakeInstructions[i]) {
					t.Errorf("got.stakeInstructions = %v, want.stakeInstructions %v", v, tt.want.stakeInstructions[i])
				}
			}
			for i, v := range got.swapInstructions {
				for index, value := range v {
					if !reflect.DeepEqual(value, tt.want.swapInstructions[i][index]) {
						t.Errorf("got.swapInstructions = %v, want.swapInstructions %v", value, tt.want.swapInstructions[i][index])
					}
				}
			}
			for i, v := range got.stopAutoStakeInstructions {
				if !reflect.DeepEqual(v, tt.want.stopAutoStakeInstructions[i]) {
					t.Errorf("got.stopAutoStakeInstructions = %v, want.stopAutoStakeInstructions %v", v, tt.want.stopAutoStakeInstructions[i])
				}
			}
			for i, v := range got.unstakeInstructions {
				if !reflect.DeepEqual(v, tt.want.unstakeInstructions[i]) {
					t.Errorf("got.unstakeInstructions = %v, want.unstakeInstructions %v", v, tt.want.unstakeInstructions[i])
				}
			}
		})
	}
}

func TestBeaconBestState_processStakeInstructionFromShardBlock(t *testing.T) {

	initPublicKey()
	initLog()

	// tx, _ := common.Hash{}.NewHashFromStr("123")
	// paymentAddress0, err := wallet.Base58CheckDeserialize(paymentAddreessKey0)
	// assert.Nil(t, err)

	type fields struct {
		BestBlockHash            common.Hash
		PreviousBestBlockHash    common.Hash
		BestBlock                types.BeaconBlock
		BestShardHash            map[byte]common.Hash
		BestShardHeight          map[byte]uint64
		Epoch                    uint64
		BeaconHeight             uint64
		BeaconProposerIndex      int
		CurrentRandomNumber      int64
		CurrentRandomTimeStamp   int64
		IsGetRandomNumber        bool
		Params                   map[string]string
		MaxBeaconCommitteeSize   int
		MinBeaconCommitteeSize   int
		MaxShardCommitteeSize    int
		MinShardCommitteeSize    int
		ActiveShards             int
		ConsensusAlgorithm       string
		ShardConsensusAlgorithm  map[byte]string
		beaconCommitteeEngine    committeestate.BeaconCommitteeEngine
		LastCrossShardState      map[byte]map[byte]uint64
		ShardHandle              map[byte]bool
		NumOfBlocksByProducers   map[string]uint64
		BlockInterval            time.Duration
		BlockMaxCreateTime       time.Duration
		consensusStateDB         *statedb.StateDB
		ConsensusStateDBRootHash common.Hash
		rewardStateDB            *statedb.StateDB
		RewardStateDBRootHash    common.Hash
		featureStateDB           *statedb.StateDB
		FeatureStateDBRootHash   common.Hash
		slashStateDB             *statedb.StateDB
		SlashStateDBRootHash     common.Hash
	}
	type args struct {
		shardInstructions    *shardInstruction
		validStakePublicKeys []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *shardInstruction
		want1  *duplicateKeyStakeInstruction
	}{
		// {
		// 	name: "Valid Input-v2",
		// 	fields: fields{
		// 		beaconCommitteeEngine: &committeestate.BeaconCommitteeEngineV2{},
		// 	},
		// 	args: args{
		// 		shardInstructions: &shardInstruction{
		// 			stakeInstructions: []*instruction.StakeInstruction{
		// 				&instruction.StakeInstruction{
		// 					PublicKeyStructs: []incognitokey.CommitteePublicKey{
		// 						*incKey0,
		// 					},
		// 					PublicKeys: []string{key0},
		// 					RewardReceiverStructs: []privacy.PaymentAddress{
		// 						paymentAddress0.KeySet.PaymentAddress,
		// 					},
		// 					AutoStakingFlag: []bool{true},
		// 					TxStakeHashes: []common.Hash{
		// 						*tx,
		// 					},
		// 					RewardReceivers: []string{paymentAddreessKey0},
		// 					Chain:           instruction.SHARD_INST,
		// 					TxStakes:        []string{tx.String()},
		// 				},
		// 				&instruction.StakeInstruction{
		// 					PublicKeyStructs: []incognitokey.CommitteePublicKey{
		// 						*incKey0,
		// 					},
		// 					PublicKeys: []string{key0},
		// 					RewardReceiverStructs: []privacy.PaymentAddress{
		// 						paymentAddress0.KeySet.PaymentAddress,
		// 					},
		// 					AutoStakingFlag: []bool{true},
		// 					TxStakeHashes: []common.Hash{
		// 						*tx,
		// 					},
		// 					RewardReceivers: []string{paymentAddreessKey0},
		// 					Chain:           instruction.SHARD_INST,
		// 					TxStakes:        []string{tx.String()},
		// 				},
		// 			},
		// 		},
		// 		validStakePublicKeys: []string{},
		// 	},
		// 	want: &shardInstruction{
		// 		stakeInstructions: []*instruction.StakeInstruction{
		// 			&instruction.StakeInstruction{
		// 				PublicKeyStructs: []incognitokey.CommitteePublicKey{
		// 					*incKey0,
		// 				},
		// 				PublicKeys: []string{key0},
		// 				RewardReceiverStructs: []privacy.PaymentAddress{
		// 					paymentAddress0.KeySet.PaymentAddress,
		// 				},
		// 				AutoStakingFlag: []bool{true},
		// 				TxStakeHashes: []common.Hash{
		// 					*tx,
		// 				},
		// 				RewardReceivers: []string{paymentAddreessKey0},
		// 				Chain:           instruction.SHARD_INST,
		// 				TxStakes:        []string{tx.String()},
		// 			},
		// 		},
		// 	},
		// 	want1: &duplicateKeyStakeInstruction{
		// 		instructions: []*instruction.StakeInstruction{
		// 			&instruction.StakeInstruction{
		// 				PublicKeyStructs: []incognitokey.CommitteePublicKey{
		// 					*incKey0,
		// 				},
		// 				PublicKeys: []string{key0},
		// 				RewardReceiverStructs: []privacy.PaymentAddress{
		// 					paymentAddress0.KeySet.PaymentAddress,
		// 				},
		// 				AutoStakingFlag: []bool{true},
		// 				TxStakeHashes: []common.Hash{
		// 					*tx,
		// 				},
		// 				RewardReceivers: []string{paymentAddreessKey0},
		// 				Chain:           instruction.SHARD_INST,
		// 				TxStakes:        []string{tx.String()},
		// 			},
		// 		},
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconBestState := &BeaconBestState{
				BestBlockHash:            tt.fields.BestBlockHash,
				PreviousBestBlockHash:    tt.fields.PreviousBestBlockHash,
				BestBlock:                tt.fields.BestBlock,
				BestShardHash:            tt.fields.BestShardHash,
				BestShardHeight:          tt.fields.BestShardHeight,
				Epoch:                    tt.fields.Epoch,
				BeaconHeight:             tt.fields.BeaconHeight,
				BeaconProposerIndex:      tt.fields.BeaconProposerIndex,
				CurrentRandomNumber:      tt.fields.CurrentRandomNumber,
				CurrentRandomTimeStamp:   tt.fields.CurrentRandomTimeStamp,
				IsGetRandomNumber:        tt.fields.IsGetRandomNumber,
				Params:                   tt.fields.Params,
				MaxBeaconCommitteeSize:   tt.fields.MaxBeaconCommitteeSize,
				MinBeaconCommitteeSize:   tt.fields.MinBeaconCommitteeSize,
				MaxShardCommitteeSize:    tt.fields.MaxShardCommitteeSize,
				MinShardCommitteeSize:    tt.fields.MinShardCommitteeSize,
				ActiveShards:             tt.fields.ActiveShards,
				ConsensusAlgorithm:       tt.fields.ConsensusAlgorithm,
				ShardConsensusAlgorithm:  tt.fields.ShardConsensusAlgorithm,
				beaconCommitteeEngine:    tt.fields.beaconCommitteeEngine,
				LastCrossShardState:      tt.fields.LastCrossShardState,
				ShardHandle:              tt.fields.ShardHandle,
				NumOfBlocksByProducers:   tt.fields.NumOfBlocksByProducers,
				BlockInterval:            tt.fields.BlockInterval,
				BlockMaxCreateTime:       tt.fields.BlockMaxCreateTime,
				consensusStateDB:         tt.fields.consensusStateDB,
				ConsensusStateDBRootHash: tt.fields.ConsensusStateDBRootHash,
				rewardStateDB:            tt.fields.rewardStateDB,
				RewardStateDBRootHash:    tt.fields.RewardStateDBRootHash,
				featureStateDB:           tt.fields.featureStateDB,
				FeatureStateDBRootHash:   tt.fields.FeatureStateDBRootHash,
				slashStateDB:             tt.fields.slashStateDB,
				SlashStateDBRootHash:     tt.fields.SlashStateDBRootHash,
			}
			got, got1 := beaconBestState.processStakeInstructionFromShardBlock(tt.args.shardInstructions, tt.args.validStakePublicKeys)
			for i, v := range got.stakeInstructions {
				if !reflect.DeepEqual(v, tt.want.stakeInstructions[i]) {
					t.Errorf("v = %v, tt.want.stakeInstructions[i] %v", v, tt.want.stakeInstructions[i])
				}
			}
			for i, v := range got1.instructions {
				if !reflect.DeepEqual(got1, tt.want1.instructions[i]) {
					t.Errorf("v = %v, tt.want1.instructions[i] %v", v, tt.want1.instructions[i])
				}
			}
		})
	}
}

func TestBeaconBestState_processStopAutoStakeInstructionFromShardBlock(t *testing.T) {

	initLog()
	initPublicKey()

	type fields struct {
		BestBlockHash            common.Hash
		PreviousBestBlockHash    common.Hash
		BestBlock                types.BeaconBlock
		BestShardHash            map[byte]common.Hash
		BestShardHeight          map[byte]uint64
		Epoch                    uint64
		BeaconHeight             uint64
		BeaconProposerIndex      int
		CurrentRandomNumber      int64
		CurrentRandomTimeStamp   int64
		IsGetRandomNumber        bool
		Params                   map[string]string
		MaxBeaconCommitteeSize   int
		MinBeaconCommitteeSize   int
		MaxShardCommitteeSize    int
		MinShardCommitteeSize    int
		ActiveShards             int
		ConsensusAlgorithm       string
		ShardConsensusAlgorithm  map[byte]string
		beaconCommitteeEngine    committeestate.BeaconCommitteeEngine
		LastCrossShardState      map[byte]map[byte]uint64
		ShardHandle              map[byte]bool
		NumOfBlocksByProducers   map[string]uint64
		BlockInterval            time.Duration
		BlockMaxCreateTime       time.Duration
		consensusStateDB         *statedb.StateDB
		ConsensusStateDBRootHash common.Hash
		rewardStateDB            *statedb.StateDB
		RewardStateDBRootHash    common.Hash
		featureStateDB           *statedb.StateDB
		FeatureStateDBRootHash   common.Hash
		slashStateDB             *statedb.StateDB
		SlashStateDBRootHash     common.Hash
	}
	type args struct {
		shardInstructions              *shardInstruction
		allCommitteeValidatorCandidate []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *shardInstruction
	}{
		{
			name:   "Valid Input",
			fields: fields{},
			args: args{
				shardInstructions: &shardInstruction{
					stopAutoStakeInstructions: []*instruction.StopAutoStakeInstruction{
						instruction.NewStopAutoStakeInstructionWithValue([]string{key}),
					},
				},
				allCommitteeValidatorCandidate: []string{key},
			},
			want: &shardInstruction{
				stopAutoStakeInstructions: []*instruction.StopAutoStakeInstruction{
					instruction.NewStopAutoStakeInstructionWithValue([]string{key}),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconBestState := &BeaconBestState{
				BestBlockHash:            tt.fields.BestBlockHash,
				PreviousBestBlockHash:    tt.fields.PreviousBestBlockHash,
				BestBlock:                tt.fields.BestBlock,
				BestShardHash:            tt.fields.BestShardHash,
				BestShardHeight:          tt.fields.BestShardHeight,
				Epoch:                    tt.fields.Epoch,
				BeaconHeight:             tt.fields.BeaconHeight,
				BeaconProposerIndex:      tt.fields.BeaconProposerIndex,
				CurrentRandomNumber:      tt.fields.CurrentRandomNumber,
				CurrentRandomTimeStamp:   tt.fields.CurrentRandomTimeStamp,
				IsGetRandomNumber:        tt.fields.IsGetRandomNumber,
				Params:                   tt.fields.Params,
				MaxBeaconCommitteeSize:   tt.fields.MaxBeaconCommitteeSize,
				MinBeaconCommitteeSize:   tt.fields.MinBeaconCommitteeSize,
				MaxShardCommitteeSize:    tt.fields.MaxShardCommitteeSize,
				MinShardCommitteeSize:    tt.fields.MinShardCommitteeSize,
				ActiveShards:             tt.fields.ActiveShards,
				ConsensusAlgorithm:       tt.fields.ConsensusAlgorithm,
				ShardConsensusAlgorithm:  tt.fields.ShardConsensusAlgorithm,
				beaconCommitteeEngine:    tt.fields.beaconCommitteeEngine,
				LastCrossShardState:      tt.fields.LastCrossShardState,
				ShardHandle:              tt.fields.ShardHandle,
				NumOfBlocksByProducers:   tt.fields.NumOfBlocksByProducers,
				BlockInterval:            tt.fields.BlockInterval,
				BlockMaxCreateTime:       tt.fields.BlockMaxCreateTime,
				consensusStateDB:         tt.fields.consensusStateDB,
				ConsensusStateDBRootHash: tt.fields.ConsensusStateDBRootHash,
				rewardStateDB:            tt.fields.rewardStateDB,
				RewardStateDBRootHash:    tt.fields.RewardStateDBRootHash,
				featureStateDB:           tt.fields.featureStateDB,
				FeatureStateDBRootHash:   tt.fields.FeatureStateDBRootHash,
				slashStateDB:             tt.fields.slashStateDB,
				SlashStateDBRootHash:     tt.fields.SlashStateDBRootHash,
			}
			if got := beaconBestState.processStopAutoStakeInstructionFromShardBlock(tt.args.shardInstructions, tt.args.allCommitteeValidatorCandidate); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BeaconBestState.processStopAutoStakeInstructionFromShardBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBeaconBestState_processUnstakeInstructionFromShardBlock(t *testing.T) {

	initLog()
	initPublicKey()

	type fields struct {
		BestBlockHash            common.Hash
		PreviousBestBlockHash    common.Hash
		BestBlock                types.BeaconBlock
		BestShardHash            map[byte]common.Hash
		BestShardHeight          map[byte]uint64
		Epoch                    uint64
		BeaconHeight             uint64
		BeaconProposerIndex      int
		CurrentRandomNumber      int64
		CurrentRandomTimeStamp   int64
		IsGetRandomNumber        bool
		Params                   map[string]string
		MaxBeaconCommitteeSize   int
		MinBeaconCommitteeSize   int
		MaxShardCommitteeSize    int
		MinShardCommitteeSize    int
		ActiveShards             int
		ConsensusAlgorithm       string
		ShardConsensusAlgorithm  map[byte]string
		beaconCommitteeEngine    committeestate.BeaconCommitteeEngine
		LastCrossShardState      map[byte]map[byte]uint64
		ShardHandle              map[byte]bool
		NumOfBlocksByProducers   map[string]uint64
		BlockInterval            time.Duration
		BlockMaxCreateTime       time.Duration
		consensusStateDB         *statedb.StateDB
		ConsensusStateDBRootHash common.Hash
		rewardStateDB            *statedb.StateDB
		RewardStateDBRootHash    common.Hash
		featureStateDB           *statedb.StateDB
		FeatureStateDBRootHash   common.Hash
		slashStateDB             *statedb.StateDB
		SlashStateDBRootHash     common.Hash
	}
	type args struct {
		shardInstructions              *shardInstruction
		allCommitteeValidatorCandidate []string
		shardID                        byte
		validUnstakePublicKeys         map[string]bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *shardInstruction
	}{
		{
			name:   "Valid Input",
			fields: fields{},
			args: args{
				shardInstructions: &shardInstruction{
					unstakeInstructions: []*instruction.UnstakeInstruction{
						&instruction.UnstakeInstruction{
							CommitteePublicKeys:       []string{key0},
							CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey0},
						},
					},
				},
				allCommitteeValidatorCandidate: []string{key0},
				shardID:                        0,
				validUnstakePublicKeys:         map[string]bool{},
			},
			want: &shardInstruction{
				unstakeInstructions: []*instruction.UnstakeInstruction{
					&instruction.UnstakeInstruction{
						CommitteePublicKeys:       []string{key0},
						CommitteePublicKeysStruct: []incognitokey.CommitteePublicKey{*incKey0},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconBestState := &BeaconBestState{
				BestBlockHash:            tt.fields.BestBlockHash,
				PreviousBestBlockHash:    tt.fields.PreviousBestBlockHash,
				BestBlock:                tt.fields.BestBlock,
				BestShardHash:            tt.fields.BestShardHash,
				BestShardHeight:          tt.fields.BestShardHeight,
				Epoch:                    tt.fields.Epoch,
				BeaconHeight:             tt.fields.BeaconHeight,
				BeaconProposerIndex:      tt.fields.BeaconProposerIndex,
				CurrentRandomNumber:      tt.fields.CurrentRandomNumber,
				CurrentRandomTimeStamp:   tt.fields.CurrentRandomTimeStamp,
				IsGetRandomNumber:        tt.fields.IsGetRandomNumber,
				Params:                   tt.fields.Params,
				MaxBeaconCommitteeSize:   tt.fields.MaxBeaconCommitteeSize,
				MinBeaconCommitteeSize:   tt.fields.MinBeaconCommitteeSize,
				MaxShardCommitteeSize:    tt.fields.MaxShardCommitteeSize,
				MinShardCommitteeSize:    tt.fields.MinShardCommitteeSize,
				ActiveShards:             tt.fields.ActiveShards,
				ConsensusAlgorithm:       tt.fields.ConsensusAlgorithm,
				ShardConsensusAlgorithm:  tt.fields.ShardConsensusAlgorithm,
				beaconCommitteeEngine:    tt.fields.beaconCommitteeEngine,
				LastCrossShardState:      tt.fields.LastCrossShardState,
				ShardHandle:              tt.fields.ShardHandle,
				NumOfBlocksByProducers:   tt.fields.NumOfBlocksByProducers,
				BlockInterval:            tt.fields.BlockInterval,
				BlockMaxCreateTime:       tt.fields.BlockMaxCreateTime,
				consensusStateDB:         tt.fields.consensusStateDB,
				ConsensusStateDBRootHash: tt.fields.ConsensusStateDBRootHash,
				rewardStateDB:            tt.fields.rewardStateDB,
				RewardStateDBRootHash:    tt.fields.RewardStateDBRootHash,
				featureStateDB:           tt.fields.featureStateDB,
				FeatureStateDBRootHash:   tt.fields.FeatureStateDBRootHash,
				slashStateDB:             tt.fields.slashStateDB,
				SlashStateDBRootHash:     tt.fields.SlashStateDBRootHash,
			}
			got := beaconBestState.processUnstakeInstructionFromShardBlock(tt.args.shardInstructions, tt.args.allCommitteeValidatorCandidate, tt.args.shardID, tt.args.validUnstakePublicKeys)
			for i, v := range got.unstakeInstructions {
				if !reflect.DeepEqual(v, tt.want.unstakeInstructions[i]) {
					t.Errorf("v = %v, want.tt.want.unstakeInstructions[i] %v", v, tt.want.unstakeInstructions[i])
				}
			}
		})
	}
}

func Test_shardInstruction_compose(t *testing.T) {
	type fields struct {
		stakeInstructions         []*instruction.StakeInstruction
		unstakeInstructions       []*instruction.UnstakeInstruction
		swapInstructions          map[byte][]*instruction.SwapInstruction
		stopAutoStakeInstructions []*instruction.StopAutoStakeInstruction
	}
	tests := []struct {
		name               string
		fields             fields
		fieldsAfterProcess fields
	}{
		{
			name: "compose stake instructions",
			fields: fields{
				stakeInstructions: []*instruction.StakeInstruction{
					&instruction.StakeInstruction{
						PublicKeys: []string{"key1", "key2"},
					},
					&instruction.StakeInstruction{
						PublicKeys: []string{"key3", "key4"},
					},
				},
			},
			fieldsAfterProcess: fields{
				stakeInstructions: []*instruction.StakeInstruction{
					&instruction.StakeInstruction{
						PublicKeys: []string{
							"key1", "key2", "key3", "key4",
						},
					},
				},
			},
		},
		{
			name: "compose unstake instructions",
			fields: fields{
				unstakeInstructions: []*instruction.UnstakeInstruction{
					&instruction.UnstakeInstruction{
						CommitteePublicKeys: []string{"key1", "key2"},
					},
					&instruction.UnstakeInstruction{
						CommitteePublicKeys: []string{"key3", "key4"},
					},
				},
			},
			fieldsAfterProcess: fields{
				unstakeInstructions: []*instruction.UnstakeInstruction{
					&instruction.UnstakeInstruction{
						CommitteePublicKeys: []string{
							"key1", "key2", "key3", "key4",
						},
					},
				},
			},
		},
		{
			name: "compose stop auto stake instructions",
			fields: fields{
				stopAutoStakeInstructions: []*instruction.StopAutoStakeInstruction{
					&instruction.StopAutoStakeInstruction{
						CommitteePublicKeys: []string{"key1", "key2"},
					},
					&instruction.StopAutoStakeInstruction{
						CommitteePublicKeys: []string{"key3", "key4"},
					},
				},
			},
			fieldsAfterProcess: fields{
				stopAutoStakeInstructions: []*instruction.StopAutoStakeInstruction{
					&instruction.StopAutoStakeInstruction{
						CommitteePublicKeys: []string{
							"key1", "key2", "key3", "key4",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shardInstruction := &shardInstruction{
				stakeInstructions:         tt.fields.stakeInstructions,
				unstakeInstructions:       tt.fields.unstakeInstructions,
				swapInstructions:          tt.fields.swapInstructions,
				stopAutoStakeInstructions: tt.fields.stopAutoStakeInstructions,
			}
			shardInstruction.compose()
			for i, v := range shardInstruction.stakeInstructions {
				if !reflect.DeepEqual(v, tt.fieldsAfterProcess.stakeInstructions[i]) {
					t.Errorf("got = %v, want %v", v, tt.fieldsAfterProcess.stakeInstructions[i])
				}
			}
		})
	}
}
