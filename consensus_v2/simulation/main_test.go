package main

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus_v2"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

var startTimeSlot uint64

type TimeSlotScenerio struct {
	ProposingScenerio []int                     // offset from position of current timeslot proposer
	VotingScenerios   map[string][]int          // offset from position of current timeslot proposer
	ExpectedOutput    map[string]TimeSlotOutput // map[offset from position of current timeslot proposer]TimeSlotOutput
}

type TimeSlotOutput struct {
	BestHeight    uint64
	BestTimeslot  uint64
	FinalHeight   uint64
	FinalTimeslot uint64
	ViewCount     int
}

type testScenerio struct {
	Name              string
	Committee         []string
	TimeSlots         int // how many timeslot
	TimeSlotScenerios map[int]TimeSlotScenerio
}

func Test_Main4Committee_Case1(t *testing.T) {
	committee := []string{
		"112t8rnXB47RhSdyVRU41TEf78nxbtWGtmjutwSp9YqsNaCpFxQGXcnwcXTtBkCGDk1KLBRBeWMvb2aXG5SeDUJRHtFV8jTB3weHEkbMJ1AL",
		"112t8rnXVdfBqBMigSs5fm9NSS8rgsVVURUxArpv6DxYmPZujKqomqUa2H9wh1zkkmDGtDn2woK4NuRDYnYRtVkUhK34TMfbUF4MShSkrCw5",
		"112t8rnXi8eKJ5RYJjyQYcFMThfbXHgaL6pq5AF5bWsDXwfsw8pqQUreDv6qgWyiABoDdphvqE7NFr9K92aomX7Gi5Nm1e4tEoV3qRLVdfSR",
		"112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43",
		"112t8s9hr9GWdfMBwwEGK12wSqvKeqpkw7jHzgHsK47EeTUcpnPAkQuzZa2xYcwHfrWtSZ6QZPeehkuDRN2u4e72HuEj7w6aKBSy4yUAZ2U3",
		"112t8rr9XGZLzuqjU2f59ey8gdyngZS3mWpwgoNzxPmNwAu8xmAQ87nnduVbZmU4Bhqnej4XTLQuS93yaG2iCGq3UXJSbBdZ8chqzhia4UuM",
		"112t8rrASXvBAtZ3dBTwXp6NH8KsX4dgmghUu36HtaPRJGvqeBqSSKb8yi7NUuNwUa58eKcyLGsXWtqfYVTgiPvAZ11GADLRZSHUNb9nssFw",
		"112t8rzc1pPSajQtjYVctFY5MGRgv2tqpRyD5zAZbwGXyh5Fum5Nafkn86iTw9w8RUhRnYH3wFLnFaZpDpb61gi3vBeQFTnzzyEErtd7jiBD",
	}
	additionalCommittee := []string{
		"112t8sJ4kBcdPD3xjqpDE2rQJXws7uYbaBnDx17zHjXM5v7Xitciozih6qnxyMazD8b6xu2c6nB5NAceKuRsKwqqLsL8cDD6pevrcomQhSuj",
		"112t8s1PX87jAEsotY2jVKcbY11yRgpDkrsVDaNU59hUJYg3FjiM5BHeqY5tyszkmVGy94ReCuhmgARN8W3cDUDJyen5XEWWK8D8RTeL4JEr",
		"112t8rw2S2U1UqMPuSZkvN4ag5gWz9twGzhfGkpPi9hrug87Co4bbi1vmCxKMDPQPGV97acVHJLjbqWWJgJhvgvKnpwkj7VUWadSPUf81tso",
		"112t8rqgs3FEcd1249ReCaMr4zbGYMRdFDMxqKGDM5nKR7AV4x3TQRMfGm9S8VEDfoZyr9fMBMpmkq94TzZ2tUGogrJo3vwWVn8mafdx86iW",
		"112t8sSnofyEiraUFykMfYYER2agCbJaYjMHBUmL5oWsCH5SoFVg1NVYt9i39wYrygbhoXFXm378vcRD3Qbdx6Rsbm47tv5K8hR6QnHXe4mo",
	}

	committee0 := []string{
		"112t8rnXB47RhSdyVRU41TEf78nxbtWGtmjutwSp9YqsNaCpFxQGXcnwcXTtBkCGDk1KLBRBeWMvb2aXG5SeDUJRHtFV8jTB3weHEkbMJ1AL",
		"112t8rnXVdfBqBMigSs5fm9NSS8rgsVVURUxArpv6DxYmPZujKqomqUa2H9wh1zkkmDGtDn2woK4NuRDYnYRtVkUhK34TMfbUF4MShSkrCw5",
		"112t8rnXi8eKJ5RYJjyQYcFMThfbXHgaL6pq5AF5bWsDXwfsw8pqQUreDv6qgWyiABoDdphvqE7NFr9K92aomX7Gi5Nm1e4tEoV3qRLVdfSR",
		"112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43",
	}
	testScn0 := testScenerio{
		Name:      "test0",
		Committee: committee0,
		TimeSlots: 10,
		TimeSlotScenerios: map[int]TimeSlotScenerio{
			2: {
				ProposingScenerio: []int{0, 1, 2, 3},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3},
				},
			},
			3: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 3, BestTimeslot: 3, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			4: {
				ProposingScenerio: []int{0, 1, 2, 3},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3},
				},
			},
			5: {
				ProposingScenerio: []int{0, 1, 2, 3},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3},
				},
			},
			6: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			7: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"all": {BestHeight: 4, BestTimeslot: 7, FinalHeight: 3, FinalTimeslot: 3, ViewCount: 2},
				},
			},
		},
	}

	RunSimulation(&testScn0, t)

	testScn := testScenerio{
		Name:      "test1",
		Committee: committee,
		TimeSlots: 7,
		TimeSlotScenerios: map[int]TimeSlotScenerio{
			2: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7, 8},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7},
				},
			},
			3: {
				ProposingScenerio: []int{1},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 3, 4, 5, 6, 7},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"2": {BestHeight: 3, BestTimeslot: 3, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			4: {
				ProposingScenerio: []int{1},
				VotingScenerios: map[string][]int{
					"all": {1},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"1": {BestHeight: 3, BestTimeslot: 3, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"0": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"2": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"3": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"4": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"5": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"6": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"7": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			5: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"all": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 3, ViewCount: 2},
				},
			},
		},
	}
	RunSimulation(&testScn, t)

	testScn2 := testScenerio{
		Name:      "test2",
		Committee: committee,
		TimeSlots: 9,
		TimeSlotScenerios: map[int]TimeSlotScenerio{
			2: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7},
				},
			},
			3: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 4, 5, 6, 7},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"3": {BestHeight: 3, BestTimeslot: 3, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			4: {
				ProposingScenerio: []int{2},
				VotingScenerios: map[string][]int{
					"all": {2},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"1": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"0": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"2": {BestHeight: 3, BestTimeslot: 3, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"3": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"4": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"5": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"6": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"7": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			5: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7},
				},
			},
			6: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3, 4, 5, 6, 7},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 4, BestTimeslot: 6, FinalHeight: 3, FinalTimeslot: 3, ViewCount: 2},
					"1": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
					"3": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
					"2": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
					"4": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
					"5": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
					"6": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
					"7": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
				},
			},
			7: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"1": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"3": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"2": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"4": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"5": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"6": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"7": {BestHeight: 4, BestTimeslot: 6, FinalHeight: 3, FinalTimeslot: 3, ViewCount: 2},
				},
			},
		},
	}
	RunSimulation(&testScn2, t)

	testScn3 := testScenerio{
		Name:      "test3",
		Committee: committee,
		TimeSlots: 14,
		TimeSlotScenerios: map[int]TimeSlotScenerio{
			2: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7},
				},
			},
			3: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3, 4, 5, 6, 7},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 3, BestTimeslot: 3, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			4: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"1": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"3": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"2": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"4": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"5": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"6": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"7": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
				},
			},
			5: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7},
				},
			},
			6: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3, 4, 5, 6, 7},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 4, BestTimeslot: 6, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"5": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
				},
			},
			7: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"1": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"2": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"3": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"4": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"5": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"6": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"7": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 3},
				},
			},
			8: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7},
				},
			},
			9: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 3, 4, 5, 6, 7},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"1": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"2": {BestHeight: 5, BestTimeslot: 9, FinalHeight: 4, FinalTimeslot: 5, ViewCount: 2},
					"3": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"4": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"5": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 3},
					"6": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
					"7": {BestHeight: 4, BestTimeslot: 5, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
				},
			},
			10: {
				ProposingScenerio: []int{1, 2, 3, 4, 5, 6, 7},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3, 4, 5, 6, 7},
				},
			},
			11: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"all": {BestHeight: 6, BestTimeslot: 11, FinalHeight: 5, FinalTimeslot: 9, ViewCount: 2},
				},
			},
		},
	}
	RunSimulation(&testScn3, t)

	testScn4 := testScenerio{
		Name:      "test4",
		Committee: append(committee, additionalCommittee...),
		TimeSlots: 14,
		TimeSlotScenerios: map[int]TimeSlotScenerio{
			2: {
				ProposingScenerio: []int{1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
			},
			3: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
			},
			4: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 3, BestTimeslot: 4, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			5: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0":  {BestHeight: 3, BestTimeslot: 3, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"12": {BestHeight: 3, BestTimeslot: 4, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
				},
			},
			6: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"0":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"1":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"2":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"3":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"4":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"5":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"6":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"7":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"8":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"9":  {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"10": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 2},
					"11": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
					"12": {BestHeight: 3, BestTimeslot: 2, FinalHeight: 2, FinalTimeslot: 1, ViewCount: 3},
				},
			},
			7: {
				ProposingScenerio: []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
			},
			8: {
				ProposingScenerio: []int{1},
				VotingScenerios: map[string][]int{
					"all": {0, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"2": {BestHeight: 4, BestTimeslot: 8, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
				},
			},
			9: {
				ProposingScenerio: []int{},
				VotingScenerios: map[string][]int{
					"all": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				},
				ExpectedOutput: map[string]TimeSlotOutput{
					"0": {BestHeight: 4, BestTimeslot: 7, FinalHeight: 3, FinalTimeslot: 2, ViewCount: 2},
				},
			},
			10: {
				ExpectedOutput: map[string]TimeSlotOutput{
					"0":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"1":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"2":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"3":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"4":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"5":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"6":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"7":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"8":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"9":  {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"10": {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"11": {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
					"12": {BestHeight: 5, BestTimeslot: 10, FinalHeight: 4, FinalTimeslot: 8, ViewCount: 2},
				},
			},
		},
	}
	RunSimulation(&testScn4, t)

}
func RunSimulation(testScn *testScenerio, t *testing.T) {
	simulation = nil
	committeePkStruct := []incognitokey.CommitteePublicKey{}
	committeePkBytes := [][]byte{}
	for _, v := range testScn.Committee {
		p, _ := consensus_v2.LoadUserKeyFromIncPrivateKey(v)
		m, _ := consensus_v2.GetMiningKeyFromPrivateSeed(p)
		pk := m.GetPublicKey()
		committeePkStruct = append(committeePkStruct, *pk)
		committeePkBytes = append(committeePkBytes, pk.MiningPubKey["bls"])
	}
	nodeList := []*Node{}

	for i, v := range testScn.Committee {
		p, _ := consensus_v2.LoadUserKeyFromIncPrivateKey(v)
		m, _ := consensus_v2.GetMiningKeyFromPrivateSeed(p)
		ni := NewNode(committeePkStruct, m, i, testScn.Name)
		nodeList = append(nodeList, ni)
	}
	var startNode = func() {
		for i, v := range nodeList {
			v.nodeList = nodeList
			fmt.Println("node start", i)
			go v.Start()
		}
	}

	GetSimulation().nodeList = nodeList
	startTimeSlot = uint64(common.CalculateTimeSlot(time.Now().Unix()))
	GetSimulation().setStartTimeSlot(uint64(startTimeSlot))
	var setTimeSlot = func(s int) uint64 {
		return startTimeSlot + uint64(s) - 1
	}
	var setProposeCommunication = func(timeslot uint64, scenario []int) {
		if GetSimulation().scenario.proposeComm[timeslot] == nil {
			GetSimulation().scenario.proposeComm[timeslot] = []int{}
		}
		GetSimulation().scenario.proposeComm[timeslot] = scenario
	}
	var setVoteCommunication = func(timeslot uint64, nodeID int, scenario []int) {
		if GetSimulation().scenario.voteComm[timeslot] == nil {
			GetSimulation().scenario.voteComm[timeslot] = make(map[string][]int)
		}
		GetSimulation().scenario.voteComm[timeslot][fmt.Sprintf("%d", nodeID)] = scenario
	}

	var timeslot uint64
	for i := 1; i <= testScn.TimeSlots; i++ {
		timeslot = setTimeSlot(i)
		slotProducerIdx := GetIndexOfBytes(nodeList[0].chain.GetBestView().GetProposerByTimeSlot(int64(timeslot), 2).MiningPubKey["bls"], committeePkBytes)
		if scenerio, ok := testScn.TimeSlotScenerios[i]; ok {
			pComm := make([]int, len(testScn.Committee))
			for a := range pComm {
				pComm[a] = 1
			}

			for _, p := range scenerio.ProposingScenerio {
				pComm[(slotProducerIdx+p)%len(testScn.Committee)] = 0
			}
			setProposeCommunication(timeslot, pComm)
			if len(scenerio.VotingScenerios) > 0 {
				if vaComm, ok := scenerio.VotingScenerios["all"]; ok {
					vComm := make([]int, len(testScn.Committee))
					for a := range vComm {
						vComm[a] = 1
					}
					for _, p := range vaComm {
						vComm[(slotProducerIdx+p)%len(testScn.Committee)] = 0
					}
					for n := 0; n < len(testScn.Committee); n++ {
						setVoteCommunication(timeslot, n, vComm)
					}
				} else {
					for n := 0; n < len(testScn.Committee); n++ {
						if nComm, ok := scenerio.VotingScenerios[strconv.Itoa(n)]; ok {
							vComm := make([]int, len(testScn.Committee))
							for a := range vComm {
								vComm[a] = 1
							}
							for _, p := range nComm {
								vComm[(slotProducerIdx+p)%len(testScn.Committee)] = 0
							}
							setVoteCommunication(timeslot, n, vComm)
						}
					}
				}
			}
		}
	}

	GetSimulation().setMaxTimeSlot(timeslot)
	startNode()
	lastTimeSlot := uint64(0)
	for {
		curTimeSlotTime := uint64(common.CalculateTimeSlot(time.Now().Unix()))
		curTimeSlot := (curTimeSlotTime - startTimeSlot) + 1
		if lastTimeSlot != curTimeSlot {
			time.AfterFunc(time.Millisecond*1500, func() {
				slotProducerIdx := GetIndexOfBytes(nodeList[0].chain.GetBestView().GetProposerByTimeSlot(int64(curTimeSlotTime), 2).MiningPubKey["bls"], committeePkBytes)
				fmt.Println("==========================")
				fmt.Printf("Timeslot is: %v\n", curTimeSlot)
				fmt.Printf("Proposer is: %d\n", slotProducerIdx)
				expectedOutput := true
				if _, ok := testScn.TimeSlotScenerios[int(curTimeSlot)]; ok {
					if len(testScn.TimeSlotScenerios[int(curTimeSlot)].ExpectedOutput) > 0 {
						if output, ok := testScn.TimeSlotScenerios[int(curTimeSlot)].ExpectedOutput["all"]; ok {
							for i := 0; i < len(testScn.Committee); i++ {
								if nodeList[i].chain.GetBestView().GetHeight() != output.BestHeight {
									expectedOutput = false
									fmt.Println("BestHeight not match!")
									break
								}
								if uint64(common.CalculateTimeSlot(nodeList[i].chain.GetBestView().GetBlock().GetProduceTime()))-startTimeSlot+1 != output.BestTimeslot {
									expectedOutput = false
									fmt.Println("BestTimeslot not match!")
									break
								}
								if nodeList[i].chain.GetFinalView().GetHeight() != output.FinalHeight {
									expectedOutput = false
									fmt.Println("FinalHeight not match!")
									break
								}
								if uint64(common.CalculateTimeSlot(nodeList[i].chain.GetFinalView().GetBlock().GetProduceTime()))-startTimeSlot+1 != output.FinalTimeslot {
									expectedOutput = false
									fmt.Println("FinalHeight not match!")
									break
								}
								if len(nodeList[i].chain.multiview.GetAllViewsWithBFS()) != output.ViewCount {
									expectedOutput = false
									fmt.Println("Viewcount not match!")
									break
								}
							}
						} else {
							for nodeID, output := range testScn.TimeSlotScenerios[int(curTimeSlot)].ExpectedOutput {
								nodeIDOffset, _ := strconv.Atoi(nodeID)
								if nodeList[(slotProducerIdx+nodeIDOffset)%len(testScn.Committee)].chain.GetBestView().GetHeight() != output.BestHeight {
									expectedOutput = false
									fmt.Println("BestHeight not match!")
									break
								}
								if uint64(common.CalculateTimeSlot(nodeList[(slotProducerIdx+nodeIDOffset)%len(testScn.Committee)].chain.GetBestView().GetBlock().GetProduceTime()))-startTimeSlot+1 != output.BestTimeslot {
									expectedOutput = false
									fmt.Println("BestTimeslot not match!")
									break
								}
								if nodeList[(slotProducerIdx+nodeIDOffset)%len(testScn.Committee)].chain.GetFinalView().GetHeight() != output.FinalHeight {
									expectedOutput = false
									fmt.Println("FinalHeight not match!")
									break
								}
								if uint64(common.CalculateTimeSlot(nodeList[(slotProducerIdx+nodeIDOffset)%len(testScn.Committee)].chain.GetFinalView().GetBlock().GetProduceTime()))-startTimeSlot+1 != output.FinalTimeslot {
									expectedOutput = false
									fmt.Println("FinalHeight not match!")
									break
								}
								if len(nodeList[(slotProducerIdx+nodeIDOffset)%len(testScn.Committee)].chain.multiview.GetAllViewsWithBFS()) != output.ViewCount {
									expectedOutput = false
									fmt.Println("Viewcount not match!")
									break
								}
							}
						}
					}
				}
				fmt.Printf("ExpectedOutput met: %v\n", expectedOutput)

				for i := 0; i < len(testScn.Committee); i++ {
					fmt.Printf("Node %v: \n -Best view height: %d:%d\n -Final view height: %d:%d\n -View count: %v\n", i, nodeList[i].chain.GetBestView().GetHeight(), uint64(common.CalculateTimeSlot(nodeList[i].chain.GetBestView().GetBlock().GetProduceTime()))-startTimeSlot+1, nodeList[i].chain.GetFinalView().GetHeight(), uint64(common.CalculateTimeSlot(nodeList[i].chain.GetFinalView().GetBlock().GetProduceTime()))-startTimeSlot+1, len(nodeList[i].chain.multiview.GetAllViewsWithBFS()))
				}
				fmt.Println("==========================")
			})
		}
		lastTimeSlot = curTimeSlot
		time.Sleep(1 * time.Millisecond)
		if curTimeSlot == uint64(testScn.TimeSlots) {
			for i, v := range nodeList {
				fmt.Println("node stop", i)
				go v.Stop()
			}
			return
		}
	}
}

//func Test_Main4BeaconCommittee_ScenarioA(t *testing.T) {
//	committee := []string{
//		"112t8rnXB47RhSdyVRU41TEf78nxbtWGtmjutwSp9YqsNaCpFxQGXcnwcXTtBkCGDk1KLBRBeWMvb2aXG5SeDUJRHtFV8jTB3weHEkbMJ1AL",
//		"112t8rnXVdfBqBMigSs5fm9NSS8rgsVVURUxArpv6DxYmPZujKqomqUa2H9wh1zkkmDGtDn2woK4NuRDYnYRtVkUhK34TMfbUF4MShSkrCw5",
//		"112t8rnXi8eKJ5RYJjyQYcFMThfbXHgaL6pq5AF5bWsDXwfsw8pqQUreDv6qgWyiABoDdphvqE7NFr9K92aomX7Gi5Nm1e4tEoV3qRLVdfSR",
//		"112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43",
//	}
//	committeePkStruct := []incognitokey.CommitteePublicKey{}
//	for _, v := range committee {
//		p, _ := blsbftv2.LoadUserKeyFromIncPrivateKey(v)
//		m, _ := blsbftv2.GetMiningKeyFromPrivateSeed(p)
//		committeePkStruct = append(committeePkStruct, m.GetPublicKey())
//	}
//	nodeList := []*Node{}
//	genesisTime, _ := time.Parse(app.GENESIS_TIMESTAMP, blockchain.TestnetGenesisBlockTime)
//	for {
//		if int(common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), blsbftv2.TIMESLOT))%len(committee) == len(committee)-1 {
//			break
//		} else {
//			time.Sleep(1 * time.Millisecond)
//		}
//	}
//
//	for i, _ := range committee {
//		ni := NewNodeBeacon(committeePkStruct, committee, i)
//		nodeList = append(nodeList, ni)
//	}
//	var startNode = func() {
//		for _, v := range nodeList {
//			v.nodeList = nodeList
//			go v.Start()
//		}
//	}
//	GetSimulation().nodeList = nodeList
//	//simulation
//	rootTimeSlot := nodeList[0].chain.GetBestView().GetRootTimeSlot()
//	currentTimeSlot := common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), 3)
//	startTimeSlot := rootTimeSlot + currentTimeSlot
//	fmt.Println("root Time slot", rootTimeSlot)
//	GetSimulation().setStartTimeSlot(startTimeSlot)
//	var setTimeSlot = func(s int) uint64 {
//		return startTimeSlot + uint64(s)
//	}
//	var setProposeCommunication = func(timeslot uint64, nodeID int, scenario []int) {
//		if GetSimulation().scenario.proposeComm[timeslot] == nil {
//			GetSimulation().scenario.proposeComm[timeslot] = make(map[string][]int)
//		}
//		GetSimulation().scenario.proposeComm[timeslot][fmt.Sprintf("%d", int(int(startTimeSlot)+nodeID)%len(committee))] = scenario
//	}
//	var setVoteCommunication = func(timeslot uint64, nodeID int, scenario []int) {
//		if GetSimulation().scenario.voteComm[timeslot] == nil {
//			GetSimulation().scenario.voteComm[timeslot] = make(map[string][]int)
//		}
//		GetSimulation().scenario.voteComm[timeslot][fmt.Sprintf("%d", int(int(startTimeSlot)+nodeID)%len(committee))] = scenario
//	}
//
//	for _, v := range nodeList {
//		v.consensusEngine.Logger.Info("\n\n")
//		v.consensusEngine.Logger.Info("===============================")
//		v.consensusEngine.Logger.Info("\n\n")
//		fmt.Printf("Node %s log is %s\n", v.id, fmt.Sprintf("log%s.log", v.id))
//	}
//
//	/*
//		START YOUR SIMULATION HERE
//	*/
//	timeslot := setTimeSlot(1) //normal communication, full connect by default
//
//	timeslot = setTimeSlot(2)
//	setProposeCommunication(timeslot, 1, []int{0, 0, 0, 1})
//	setVoteCommunication(timeslot, 3, []int{0, 1, 0, 0})
//	//
//	timeslot = setTimeSlot(3)
//	setVoteCommunication(timeslot, 0, []int{0, 0, 0, 0})
//	setVoteCommunication(timeslot, 1, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 2, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 3, []int{1, 0, 0, 0})
//	//
//	timeslot = setTimeSlot(4)
//	setVoteCommunication(timeslot, 1, []int{0, 1, 1, 1})
//	setVoteCommunication(timeslot, 2, []int{0, 1, 1, 1})
//	setVoteCommunication(timeslot, 3, []int{0, 1, 1, 1})
//
//	timeslot = setTimeSlot(5)
//	setProposeCommunication(timeslot, 0, []int{0, 0, 1, 0})
//	setVoteCommunication(timeslot, 2, []int{1, 0, 0, 0})
//
//	timeslot = setTimeSlot(6)
//	setVoteCommunication(timeslot, 0, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 1, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 3, []int{1, 0, 0, 0})
//
//	timeslot = setTimeSlot(7)
//	setVoteCommunication(timeslot, 1, []int{0, 1, 1, 1})
//	setVoteCommunication(timeslot, 2, []int{0, 1, 1, 1})
//	setVoteCommunication(timeslot, 3, []int{0, 1, 1, 1})
//
//	timeslot = setTimeSlot(8)
//	setProposeCommunication(timeslot, 3, []int{0, 0, 0, 0})
//
//	timeslot = setTimeSlot(9)
//	setProposeCommunication(timeslot, 0, []int{0, 0, 1, 0})
//	setVoteCommunication(timeslot, 2, []int{1, 0, 0, 0})
//
//	timeslot = setTimeSlot(10)
//	setVoteCommunication(timeslot, 0, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 1, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 3, []int{1, 0, 0, 0})
//
//	timeslot = setTimeSlot(11)
//	timeslot = setTimeSlot(12)
//
//	/*
//		END YOUR SIMULATION HERE
//	*/
//	GetSimulation().setMaxTimeSlot(timeslot)
//	startNode()
//	go func() {
//		lastTimeSlot := uint64(0)
//		for {
//			curTimeSlot := (common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), blsbftv2.TIMESLOT) - startTimeSlot) + 1
//			if lastTimeSlot != curTimeSlot {
//				time.AfterFunc(time.Millisecond*500, func() {
//					fmt.Printf("Best view height: %d. Final view height: %d\n", fullnode.GetBestView().GetHeight(), fullnode.GetFinalView().GetHeight())
//				})
//			}
//			for _, v := range nodeList {
//				if lastTimeSlot != curTimeSlot && curTimeSlot <= GetSimulation().maxTimeSlot {
//					v.consensusEngine.Logger.Info("========================================")
//					v.consensusEngine.Logger.Info("SIMULATION NODE", v.id, "TIMESLOT", curTimeSlot)
//					v.consensusEngine.Logger.Info("========================================")
//				}
//
//			}
//			lastTimeSlot = curTimeSlot
//			time.Sleep(1 * time.Millisecond)
//		}
//	}()
//	select {}
//}
//
//func Test_Main4BeaconCommittee_ScenarioB(t *testing.T) {
//	committee := []string{
//		"112t8rnXB47RhSdyVRU41TEf78nxbtWGtmjutwSp9YqsNaCpFxQGXcnwcXTtBkCGDk1KLBRBeWMvb2aXG5SeDUJRHtFV8jTB3weHEkbMJ1AL",
//		"112t8rnXVdfBqBMigSs5fm9NSS8rgsVVURUxArpv6DxYmPZujKqomqUa2H9wh1zkkmDGtDn2woK4NuRDYnYRtVkUhK34TMfbUF4MShSkrCw5",
//		"112t8rnXi8eKJ5RYJjyQYcFMThfbXHgaL6pq5AF5bWsDXwfsw8pqQUreDv6qgWyiABoDdphvqE7NFr9K92aomX7Gi5Nm1e4tEoV3qRLVdfSR",
//		"112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43",
//	}
//	committeePkStruct := []incognitokey.CommitteePublicKey{}
//	for _, v := range committee {
//		p, _ := blsbftv2.LoadUserKeyFromIncPrivateKey(v)
//		m, _ := blsbftv2.GetMiningKeyFromPrivateSeed(p)
//		committeePkStruct = append(committeePkStruct, m.GetPublicKey())
//	}
//	nodeList := []*Node{}
//	genesisTime, _ := time.Parse(app.GENESIS_TIMESTAMP, blockchain.TestnetGenesisBlockTime)
//	for {
//		if int(common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), blsbftv2.TIMESLOT))%len(committee) == len(committee)-1 {
//			break
//		} else {
//			time.Sleep(1 * time.Millisecond)
//		}
//	}
//
//	for i, _ := range committee {
//		ni := NewNodeBeacon(committeePkStruct, committee, i)
//		nodeList = append(nodeList, ni)
//	}
//	var startNode = func() {
//		for _, v := range nodeList {
//			v.nodeList = nodeList
//			go v.Start()
//		}
//	}
//	GetSimulation().nodeList = nodeList
//	//simulation
//	rootTimeSlot := nodeList[0].chain.GetBestView().GetRootTimeSlot()
//	currentTimeSlot := common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), 3)
//	startTimeSlot := rootTimeSlot + currentTimeSlot
//	fmt.Println("root Time slot", rootTimeSlot)
//	GetSimulation().setStartTimeSlot(startTimeSlot)
//	var setTimeSlot = func(s int) uint64 {
//		return startTimeSlot + uint64(s)
//	}
//	var setProposeCommunication = func(timeslot uint64, nodeID int, scenario []int) {
//		if GetSimulation().scenario.proposeComm[timeslot] == nil {
//			GetSimulation().scenario.proposeComm[timeslot] = make(map[string][]int)
//		}
//		GetSimulation().scenario.proposeComm[timeslot][fmt.Sprintf("%d", int(int(startTimeSlot)+nodeID)%len(committee))] = scenario
//	}
//	var setVoteCommunication = func(timeslot uint64, nodeID int, scenario []int) {
//		if GetSimulation().scenario.voteComm[timeslot] == nil {
//			GetSimulation().scenario.voteComm[timeslot] = make(map[string][]int)
//		}
//		GetSimulation().scenario.voteComm[timeslot][fmt.Sprintf("%d", int(int(startTimeSlot)+nodeID)%len(committee))] = scenario
//	}
//
//	for _, v := range nodeList {
//		v.consensusEngine.Logger.Info("\n\n")
//		v.consensusEngine.Logger.Info("===============================")
//		v.consensusEngine.Logger.Info("\n\n")
//		fmt.Printf("Node %s log is %s\n", v.id, fmt.Sprintf("log%s.log", v.id))
//	}
//
//	/*
//		START YOUR SIMULATION HERE
//	*/
//	timeslot := setTimeSlot(1) //normal communication, full connect by default
//
//	timeslot = setTimeSlot(2)
//	setProposeCommunication(timeslot, 1, []int{0, 0, 0, 1})
//	setVoteCommunication(timeslot, 3, []int{0, 1, 0, 0})
//	//
//	timeslot = setTimeSlot(3)
//	setVoteCommunication(timeslot, 0, []int{0, 0, 0, 0})
//	setVoteCommunication(timeslot, 1, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 2, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 3, []int{1, 0, 0, 0})
//	//
//	timeslot = setTimeSlot(4)
//
//	timeslot = setTimeSlot(5)
//	setProposeCommunication(timeslot, 0, []int{0, 0, 1, 0})
//	setVoteCommunication(timeslot, 2, []int{1, 0, 0, 0})
//
//	timeslot = setTimeSlot(6)
//	setVoteCommunication(timeslot, 0, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 1, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 2, []int{1, 0, 0, 0})
//	setVoteCommunication(timeslot, 3, []int{1, 0, 0, 0})
//
//	timeslot = setTimeSlot(7)
//	setVoteCommunication(timeslot, 1, []int{0, 1, 1, 1})
//	setVoteCommunication(timeslot, 2, []int{0, 1, 1, 1})
//	setVoteCommunication(timeslot, 3, []int{0, 1, 1, 1})
//
//	timeslot = setTimeSlot(8)
//	setProposeCommunication(timeslot, 3, []int{0, 1, 0, 0})
//	setProposeCommunication(timeslot, 1, []int{0, 0, 0, 1})
//
//	timeslot = setTimeSlot(9)
//	setProposeCommunication(timeslot, 0, []int{0, 0, 0, 0})
//
//	timeslot = setTimeSlot(10)
//	setProposeCommunication(timeslot, 1, []int{0, 1, 1, 1})
//	timeslot = setTimeSlot(11)
//	setProposeCommunication(timeslot, 2, []int{0, 1, 1, 1})
//	timeslot = setTimeSlot(12)
//	setProposeCommunication(timeslot, 3, []int{0, 1, 1, 1})
//
//	timeslot = setTimeSlot(13)
//
//	/*
//		END YOUR SIMULATION HERE
//	*/
//	GetSimulation().setMaxTimeSlot(timeslot)
//	startNode()
//	go func() {
//		lastTimeSlot := uint64(0)
//		for {
//			curTimeSlot := (common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), blsbftv2.TIMESLOT) - startTimeSlot) + 1
//			if lastTimeSlot != curTimeSlot {
//				time.AfterFunc(time.Millisecond*500, func() {
//					fmt.Printf("Best view height: %d. Final view height: %d\n", fullnode.GetBestView().GetHeight(), fullnode.GetFinalView().GetHeight())
//				})
//			}
//			for _, v := range nodeList {
//				if lastTimeSlot != curTimeSlot && curTimeSlot <= GetSimulation().maxTimeSlot {
//					v.consensusEngine.Logger.Info("========================================")
//					v.consensusEngine.Logger.Info("SIMULATION NODE", v.id, "TIMESLOT", curTimeSlot)
//					v.consensusEngine.Logger.Info("========================================")
//				}
//			}
//			lastTimeSlot = curTimeSlot
//			time.Sleep(1 * time.Millisecond)
//		}
//	}()
//	select {}
//}
//
//func Test_Main4BeaconCommittee_ScenarioC(t *testing.T) {
//	committee := []string{
//		"112t8rnXB47RhSdyVRU41TEf78nxbtWGtmjutwSp9YqsNaCpFxQGXcnwcXTtBkCGDk1KLBRBeWMvb2aXG5SeDUJRHtFV8jTB3weHEkbMJ1AL",
//		"112t8rnXVdfBqBMigSs5fm9NSS8rgsVVURUxArpv6DxYmPZujKqomqUa2H9wh1zkkmDGtDn2woK4NuRDYnYRtVkUhK34TMfbUF4MShSkrCw5",
//		"112t8rnXi8eKJ5RYJjyQYcFMThfbXHgaL6pq5AF5bWsDXwfsw8pqQUreDv6qgWyiABoDdphvqE7NFr9K92aomX7Gi5Nm1e4tEoV3qRLVdfSR",
//		"112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43",
//	}
//	committeePkStruct := []incognitokey.CommitteePublicKey{}
//	for _, v := range committee {
//		p, _ := blsbftv2.LoadUserKeyFromIncPrivateKey(v)
//		m, _ := blsbftv2.GetMiningKeyFromPrivateSeed(p)
//		committeePkStruct = append(committeePkStruct, m.GetPublicKey())
//	}
//	nodeList := []*Node{}
//	genesisTime, _ := time.Parse(app.GENESIS_TIMESTAMP, blockchain.TestnetGenesisBlockTime)
//	for {
//		if int(common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), blsbftv2.TIMESLOT))%len(committee) == len(committee)-1 {
//			break
//		} else {
//			time.Sleep(1 * time.Millisecond)
//		}
//	}
//
//	for i, _ := range committee {
//		ni := NewNodeBeacon(committeePkStruct, committee, i)
//		nodeList = append(nodeList, ni)
//	}
//	var startNode = func() {
//		for _, v := range nodeList {
//			v.nodeList = nodeList
//			go v.Start()
//		}
//	}
//	GetSimulation().nodeList = nodeList
//	//simulation
//	rootTimeSlot := nodeList[0].chain.GetBestView().GetRootTimeSlot()
//	currentTimeSlot := common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), 3)
//	startTimeSlot := rootTimeSlot + currentTimeSlot
//	fmt.Println("root Time slot", rootTimeSlot)
//	GetSimulation().setStartTimeSlot(startTimeSlot)
//	var setTimeSlot = func(s int) uint64 {
//		return startTimeSlot + uint64(s)
//	}
//	var setProposeCommunication = func(timeslot uint64, nodeID int, scenario []int) {
//		if GetSimulation().scenario.proposeComm[timeslot] == nil {
//			GetSimulation().scenario.proposeComm[timeslot] = make(map[string][]int)
//		}
//		GetSimulation().scenario.proposeComm[timeslot][fmt.Sprintf("%d", int(int(startTimeSlot)+nodeID)%len(committee))] = scenario
//	}
//	//var setVoteCommunication = func(timeslot uint64, nodeID int, scenario []int) {
//	//	if GetSimulation().scenario.voteComm[timeslot] == nil {
//	//		GetSimulation().scenario.voteComm[timeslot] = make(map[string][]int)
//	//	}
//	//	GetSimulation().scenario.voteComm[timeslot][fmt.Sprintf("%d", int(int(startTimeSlot)+nodeID)%len(committee))] = scenario
//	//}
//
//	for _, v := range nodeList {
//		v.consensusEngine.Logger.Info("\n\n")
//		v.consensusEngine.Logger.Info("===============================")
//		v.consensusEngine.Logger.Info("\n\n")
//		fmt.Printf("Node %s log is %s\n", v.id, fmt.Sprintf("log%s.log", v.id))
//	}
//
//	/*
//		START YOUR SIMULATION HERE
//	*/
//	timeslot := setTimeSlot(1)
//	setProposeCommunication(timeslot, 0, []int{0, 0, 0, 0})
//	setProposeCommunication(timeslot, 1, []int{0, 0, 0, 0})
//	setProposeCommunication(timeslot, 2, []int{0, 0, 0, 0})
//	setProposeCommunication(timeslot, 3, []int{0, 0, 0, 0})
//
//	timeslot = setTimeSlot(100) //normal communication, full connect by default
//
//	/*
//		END YOUR SIMULATION HERE
//	*/
//	GetSimulation().setMaxTimeSlot(timeslot)
//	startNode()
//	go func() {
//		lastTimeSlot := uint64(0)
//		for {
//			curTimeSlot := (common.GetTimeSlot(genesisTime.Unix(), time.Now().Unix(), blsbftv2.TIMESLOT) - startTimeSlot) + 1
//			if lastTimeSlot != curTimeSlot {
//				time.AfterFunc(time.Millisecond*500, func() {
//					fmt.Printf("Best view height: %d. Final view height: %d\n", fullnode.GetBestView().GetHeight(), fullnode.GetFinalView().GetHeight())
//				})
//			}
//			for _, v := range nodeList {
//				if lastTimeSlot != curTimeSlot && curTimeSlot <= GetSimulation().maxTimeSlot {
//					v.consensusEngine.Logger.Info("========================================")
//					v.consensusEngine.Logger.Info("SIMULATION NODE", v.id, "TIMESLOT", curTimeSlot)
//					v.consensusEngine.Logger.Info("========================================")
//				}
//			}
//			lastTimeSlot = curTimeSlot
//			time.Sleep(1 * time.Millisecond)
//		}
//	}()
//	select {}
//}
