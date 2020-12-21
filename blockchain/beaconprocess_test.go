package blockchain

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/incognitokey"
)

// TESTCASE
// 1. RETURN SHARD-CANDIDATES,NO-BEACON-CANDIDATES,NO-ERROR
// 	INPUT: mock 1 stake shard instruction with 3 committee
// 2. RETURN NO-SHARD-CANDIDATES,BEACON-CANDIDATES,NO-ERROR
// 	INPUT: mock 1 stake beacon instruction with 3 committee
// 3. RETURN SHARD-CANDIDATES,BEACON-CANDIDATES,NO-ERROR
// 	INPUT: mock 1 stake beacon instruction with 3 committee & 1 stake shard instruction with 3 committee
// 4. RETURN NO-SHARD-CANDIDATES,NO-BEACON-CANDIDATES,ERROR NOT PASS CONDITION check
// 	len(beaconCandidatesStructs) != len(beaconRewardReceivers) && len(beaconRewardReceivers) != len(beaconAutoReStaking)
// 	len(shardCandidates) != len(shardRewardReceivers) && len(shardRewardReceivers) != len(shardAutoReStaking)
func TestBeaconBestState_processStakeInstruction(t *testing.T) {
	type fields struct {
		AutoStaking            *MapStringBool
		RewardReceiver         map[string]privacy.PaymentAddress
		StakingTx              map[string]common.Hash
		BeaconHeight           uint64
		ShardCommittee         map[byte][]incognitokey.CommitteePublicKey
		ShardPendingValidator  map[byte][]incognitokey.CommitteePublicKey
		BeaconCommittee        []incognitokey.CommitteePublicKey
		BeaconPendingValidator []incognitokey.CommitteePublicKey
	}
	type args struct {
		instruction     []string
		blockchain      *BlockChain
		committeeChange *committeeChange
		autoStaking     map[string]bool
	}

	bc := &BlockChain{}

	wantCandidate, _ := incognitokey.CommitteeBase58KeyListToStruct([]string{"121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7"})
	tests := []struct {
		name   string
		fields fields
		args   args
		want   error
		want1  bool
		want2  []incognitokey.CommitteePublicKey
		want3  []incognitokey.CommitteePublicKey
	}{
		{
			name: "shard candidates only case",
			fields: fields{
				AutoStaking:    NewMapStringBool(),
				RewardReceiver: make(map[string]privacy.PaymentAddress),
				StakingTx:      make(map[string]common.Hash),
			},
			args: args{
				blockchain:      bc,
				committeeChange: &committeeChange{},
				instruction:     []string{"stake", "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7", "shard", "0000000000000000000000000000000000000000000000000000000000000000", "12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX", "false"},
			},
			want:  nil,
			want1: false,
			want2: []incognitokey.CommitteePublicKey{},
			want3: wantCandidate,
		},
		{
			name: "beacon candidates only case",
			fields: fields{
				AutoStaking:    NewMapStringBool(),
				RewardReceiver: make(map[string]privacy.PaymentAddress),
				StakingTx:      make(map[string]common.Hash),
			},
			args: args{
				blockchain:      bc,
				committeeChange: &committeeChange{},
				instruction:     []string{"stake", "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7", "beacon", "0000000000000000000000000000000000000000000000000000000000000000", "12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX", "false"},
			},
			want:  nil,
			want2: wantCandidate,
			want3: []incognitokey.CommitteePublicKey{},
		},
		{
			name: "error case",
			fields: fields{
				AutoStaking:    NewMapStringBool(),
				RewardReceiver: make(map[string]privacy.PaymentAddress),
				StakingTx:      make(map[string]common.Hash),
			},
			args: args{
				blockchain:      bc,
				committeeChange: &committeeChange{},
				instruction:     []string{"stake", "121VhftSAygpEJZ6i9jGk4fj81FpWVTwe3wWDzRZjzdjaQXk9QtGbwNWNwjt3p8zi3p2LRug8m78TDeq4LCAiQT2shDLSrK9sSHBX4DrNgnqsRbkEazrnWapvs7F5CMTPj5kT859WHJV26Wm1P8hwHXpxLwbeMM9n2kJXznTgRJGzdBZ4iY2CTF28s7ADyknqcBJ1RBfEUT9GVeixKC3AKDAna2QqQfdcdFiJaps5PixjJznk7CcTgcYgfPcnysdUgRuygAcbDikvw35KF9jzmeTZWZtbXhbXePhyPP8MuaGwDY75hCiDn1iDEvNHBGMqKJtENq8mfkQTW9GrGu2kkDBmNsmDVannjsbxUuoHU9MT5hYftTcsvyVi4s2S73JbGDNnWD7e3cVwXF8rgYGMFNyYBm3qWB3jobBkGwTPNh5Tpb7,", "beacon", "0000000000000000000000000000000000000000000000000000000000000000", "12S42qYc9pzsfWoxPZ21sVihEHJxYfNzEp1SXNnxvr7CGYMHNWX12ZaQkzcwvTYKAnhiVsDWwSqz5jFo6xuwzXZmz7QX1TnJaWnwEyX", "false,false"},
			},
			want:  NewBlockChainError(StakeInstructionError, fmt.Errorf("Expect Beacon Candidate (length %+v) and Beacon Reward Receiver (length %+v) and Beacon Auto ReStaking (lenght %+v) have equal length", 2, 1, 1)),
			want1: false,
			want2: []incognitokey.CommitteePublicKey{},
			want3: []incognitokey.CommitteePublicKey{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconBestState := &BeaconBestState{
				AutoStaking:            tt.fields.AutoStaking,
				RewardReceiver:         tt.fields.RewardReceiver,
				StakingTx:              tt.fields.StakingTx,
				BeaconHeight:           tt.fields.BeaconHeight,
				ShardCommittee:         tt.fields.ShardCommittee,
				ShardPendingValidator:  tt.fields.ShardPendingValidator,
				BeaconCommittee:        tt.fields.BeaconCommittee,
				BeaconPendingValidator: tt.fields.BeaconPendingValidator,
			}
			got, got1, got2, got3 := beaconBestState.processInstruction(tt.args.instruction, tt.args.blockchain, tt.args.committeeChange, nil, []string{})

			if tt.want1 && got.Error() == tt.want.Error() {
				t.Errorf("processInstruction() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("processInstruction() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("processInstruction() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("processInstruction() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}
