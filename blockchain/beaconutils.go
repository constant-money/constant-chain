package blockchain

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
)

func GetStakingCandidate(beaconBlock BeaconBlock) ([]string, []string) {
	beacon := []string{}
	shard := []string{}
	beaconBlockBody := beaconBlock.Body
	for _, v := range beaconBlockBody.Instructions {
		if len(v) < 1 {
			continue
		}
		if v[0] == StakeAction && v[2] == "beacon" {
			beacon = strings.Split(v[1], ",")
		}
		if v[0] == StakeAction && v[2] == "shard" {
			shard = strings.Split(v[1], ",")
		}
	}

	return beacon, shard
}

// assignShardCandidate Assign Candidates Into Shard Pending Validator List
// Each Shard Pending Validator List has a limit
// If a candidate is assigned into shard which Pending Validator List has reach its limit then candidate will get back into candidate list
// Otherwise, candidate will be converted to shard pending validator
// - return param #1: remain shard candidate (not assign yet)
// - return param #2: assigned candidate
func assignShardCandidate(candidates []string, numberOfPendingValidator map[byte]int, rand int64, testnetAssignOffset int, activeShards int) ([]string, map[byte][]string) {
	assignedCandidates := make(map[byte][]string)
	remainShardCandidates := []string{}
	shuffledCandidate := shuffleShardCandidate(candidates, rand)
	for _, candidate := range shuffledCandidate {
		shardID := calculateCandidateShardID(candidate, rand, activeShards)
		if numberOfPendingValidator[shardID]+1 > testnetAssignOffset {
			remainShardCandidates = append(remainShardCandidates, candidate)
			continue
		} else {
			assignedCandidates[shardID] = append(assignedCandidates[shardID], candidate)
			numberOfPendingValidator[shardID] += 1
		}
	}
	return remainShardCandidates, assignedCandidates
}

// shuffleShardCandidate Shuffle Position Of Shard Candidates in List with Random Number
func shuffleShardCandidate(candidates []string, rand int64) []string {
	m := make(map[string]string)
	temp := []string{}
	shuffledCandidates := []string{}
	for _, candidate := range candidates {
		seed := strconv.Itoa(int(rand)) + candidate
		hash := common.HashH([]byte(seed)).String()
		m[hash] = candidate
		temp = append(temp, hash)
	}
	if len(m) != len(temp) {
		fmt.Println(candidates)
		panic("Failed To Shuffle Shard Candidate Before Assign to Shard")
	}
	sort.Strings(temp)
	for _, key := range temp {
		shuffledCandidates = append(shuffledCandidates, m[key])
	}
	if len(shuffledCandidates) != len(candidates) {
		panic("Failed To Shuffle Shard Candidate Before Assign to Shard")
	}
	return shuffledCandidates
}

// Formula ShardID: LSB[hash(candidatePubKey+randomNumber)]
// Last byte of hash(candidatePubKey+randomNumber)
func calculateCandidateShardID(candidate string, rand int64, activeShards int) (shardID byte) {
	seed := candidate + strconv.Itoa(int(rand))
	hash := common.HashB([]byte(seed))
	shardID = byte(int(hash[len(hash)-1]) % activeShards)
	Logger.log.Critical("calculateCandidateShardID/shardID", shardID)
	return shardID
}

func filterValidators(
	validators []string,
	producersBlackList map[string]uint8,
	isExistenceIncluded bool,
) []string {
	resultingValidators := []string{}
	for _, pv := range validators {
		_, found := producersBlackList[pv]
		if (found && isExistenceIncluded) || (!found && !isExistenceIncluded) {
			resultingValidators = append(resultingValidators, pv)
		}
	}
	return resultingValidators
}

func isBadProducer(badProducers []string, producer string) bool {
	for _, badProducer := range badProducers {
		if badProducer == producer {
			return true
		}
	}
	return false
}

func CreateBeaconSwapActionForKeyListV2(
	genesisParam *GenesisParams,
	pendingValidator []string,
	beaconCommittees []string,
	minCommitteeSize int,
	epoch uint64,
) ([]string, []string, []string) {
	newPendingValidator := pendingValidator
	swapInstruction, newBeaconCommittees := GetBeaconSwapInstructionKeyListV2(genesisParam, epoch)
	remainBeaconCommittees := beaconCommittees[minCommitteeSize:]
	return swapInstruction, newPendingValidator, append(newBeaconCommittees, remainBeaconCommittees...)
}

// swap return argument
// #1 remaining pendingValidators
// #2 new currentValidators
// #3 swapped out validator
// #4 incoming validator
// #5 error
// REVIEW: @hung
// - should check offset <= goodPendingValidators
func swap(
	badPendingValidators []string,
	goodPendingValidators []string,
	currentGoodProducers []string,
	currentBadProducers []string,
	maxCommittee int,
	offset int,
) ([]string, []string, []string, []string, error) {
	// if swap offset = 0 then do nothing
	if offset == 0 {
		// return pendingValidators, currentGoodProducers, currentBadProducers, []string{}, errors.New("no pending validator for swapping")
		return append(goodPendingValidators, badPendingValidators...), currentGoodProducers, currentBadProducers, []string{}, nil
	}
	if offset > maxCommittee {
		return append(goodPendingValidators, badPendingValidators...), currentGoodProducers, currentBadProducers, []string{}, errors.New("try to swap too many validators")
	}
	tempValidators := []string{}
	swapValidator := currentBadProducers
	diff := maxCommittee - len(currentGoodProducers)
	if diff >= offset {
		tempValidators = append(tempValidators, goodPendingValidators[:offset]...)
		currentGoodProducers = append(currentGoodProducers, tempValidators...)
		goodPendingValidators = goodPendingValidators[offset:]
		return append(goodPendingValidators, badPendingValidators...), currentGoodProducers, swapValidator, tempValidators, nil
	}
	offset -= diff
	tempValidators = append(tempValidators, goodPendingValidators[:diff]...)
	goodPendingValidators = goodPendingValidators[diff:]
	currentGoodProducers = append(currentGoodProducers, tempValidators...)

	// out pubkey: swapped out validator
	swapValidator = append(swapValidator, currentGoodProducers[:offset]...)
	// unqueue validator with index from 0 to offset-1 from currentValidators list
	currentGoodProducers = currentGoodProducers[offset:]
	// in pubkey: unqueue validator with index from 0 to offset-1 from pendingValidators list
	tempValidators = append(tempValidators, goodPendingValidators[:offset]...)
	// enqueue new validator to the remaning of current validators list
	currentGoodProducers = append(currentGoodProducers, goodPendingValidators[:offset]...)
	// save new pending validators list
	goodPendingValidators = goodPendingValidators[offset:]
	return append(goodPendingValidators, badPendingValidators...), currentGoodProducers, swapValidator, tempValidators, nil
}

// SwapValidator consider these list as queue structure
// unqueue a number of validator out of currentValidators list
// enqueue a number of validator into currentValidators list <=> unqueue a number of validator out of pendingValidators list
// return value: #1 remaining pendingValidators, #2 new currentValidators #3 swapped out validator, #4 incoming validator #5 error
func SwapValidator(
	pendingValidators []string,
	currentValidators []string,
	maxCommittee int,
	minCommittee int,
	offset int,
	producersBlackList map[string]uint8,
	swapOffset int,
) ([]string, []string, []string, []string, error) {
	goodPendingValidators := filterValidators(pendingValidators, producersBlackList, false)
	badPendingValidators := filterValidators(pendingValidators, producersBlackList, true)
	currentBadProducers := filterValidators(currentValidators, producersBlackList, true)
	currentGoodProducers := filterValidators(currentValidators, producersBlackList, false)
	goodPendingValidatorsLen := len(goodPendingValidators)
	currentGoodProducersLen := len(currentGoodProducers)
	// number of good producer more than minimum needed producer to continue
	if currentGoodProducersLen >= minCommittee {
		// current number of good producer reach maximum committee size => swap
		if currentGoodProducersLen == maxCommittee {
			offset = swapOffset
		}
		// if not then number of good producer are less than maximum committee size
		// push more pending validator into committee list

		// if number of current good pending validators are less than maximum push offset
		// then push all good pending validator into committee
		if offset > goodPendingValidatorsLen {
			offset = goodPendingValidatorsLen
		}
		return swap(badPendingValidators, goodPendingValidators, currentGoodProducers, currentBadProducers, maxCommittee, offset)
	}

	minProducersNeeded := minCommittee - currentGoodProducersLen
	if len(pendingValidators) >= minProducersNeeded {
		if offset < minProducersNeeded {
			offset = minProducersNeeded
		} else if offset > goodPendingValidatorsLen {
			offset = goodPendingValidatorsLen
		}
		return swap(badPendingValidators, goodPendingValidators, currentGoodProducers, currentBadProducers, maxCommittee, offset)
	}

	producersNumCouldBeSwapped := len(goodPendingValidators) + len(currentValidators) - minCommittee
	swappedProducers := []string{}
	remainingProducers := []string{}
	for _, producer := range currentValidators {
		if isBadProducer(currentBadProducers, producer) && len(swappedProducers) < producersNumCouldBeSwapped {
			swappedProducers = append(swappedProducers, producer)
			continue
		}
		remainingProducers = append(remainingProducers, producer)
	}
	newProducers := append(remainingProducers, goodPendingValidators...)
	return badPendingValidators, newProducers, swappedProducers, goodPendingValidators, nil
}

// RemoveValidator remove validator and return removed list
// return: #param1: validator list after remove
// in parameter: #param1: list of full validator
// in parameter: #param2: list of removed validator
// removed validators list must be a subset of full validator list and it must be first in the list
func RemoveValidator(validators []string, removedValidators []string) ([]string, error) {
	// if number of pending validator is less or equal than offset, set offset equal to number of pending validator
	if len(removedValidators) > len(validators) {
		return validators, errors.New("trying to remove too many validators")
	}
	remainingValidators := []string{}
	for _, validator := range validators {
		isRemoved := false
		for _, removedValidator := range removedValidators {
			if strings.Compare(validator, removedValidator) == 0 {
				isRemoved = true
			}
		}
		if !isRemoved {
			remainingValidators = append(remainingValidators, validator)
		}
	}
	return remainingValidators, nil
}

// Shuffle Candidate: suffer candidates with random number and return suffered list
// Candidate Value Concatenate with Random Number
// then Hash and Obtain Hash Value
// Sort Hash Value Then Re-arrange Candidate corresponding to Hash Value
func ShuffleCandidate(candidates []incognitokey.CommitteePublicKey, rand int64) ([]incognitokey.CommitteePublicKey, error) {
	Logger.log.Debug("Beacon Process/Shuffle Candidate: Candidate Before Sort ", candidates)
	hashes := []string{}
	m := make(map[string]incognitokey.CommitteePublicKey)
	sortedCandidate := []incognitokey.CommitteePublicKey{}
	for _, candidate := range candidates {
		candidateStr, _ := candidate.ToBase58()
		seed := candidateStr + strconv.Itoa(int(rand))
		hash := common.HashB([]byte(seed))
		hashes = append(hashes, string(hash[:32]))
		m[string(hash[:32])] = candidate
	}
	sort.Strings(hashes)
	for _, hash := range hashes {
		sortedCandidate = append(sortedCandidate, m[hash])
	}
	Logger.log.Debug("Beacon Process/Shuffle Candidate: Candidate After Sort ", sortedCandidate)
	return sortedCandidate, nil
}

func getStakeValidatorArrayString(v []string) ([]string, []string) {
	beacon := []string{}
	shard := []string{}
	if len(v) > 0 {
		if v[0] == StakeAction && v[2] == "beacon" {
			beacon = strings.Split(v[1], ",")
		}
		if v[0] == StakeAction && v[2] == "shard" {
			shard = strings.Split(v[1], ",")
		}
	}
	return beacon, shard
}

func snapshotCommittee(beaconCommittee []incognitokey.CommitteePublicKey, allShardCommittee map[byte][]incognitokey.CommitteePublicKey) ([]incognitokey.CommitteePublicKey, map[byte][]incognitokey.CommitteePublicKey, error) {
	snapshotBeaconCommittee := []incognitokey.CommitteePublicKey{}
	snapshotAllShardCommittee := make(map[byte][]incognitokey.CommitteePublicKey)
	for _, committee := range beaconCommittee {
		snapshotBeaconCommittee = append(snapshotBeaconCommittee, committee)
	}
	for shardID, shardCommittee := range allShardCommittee {
		clonedShardCommittee := []incognitokey.CommitteePublicKey{}
		snapshotAllShardCommittee[shardID] = []incognitokey.CommitteePublicKey{}
		for _, committee := range shardCommittee {
			clonedShardCommittee = append(clonedShardCommittee, committee)
		}
		snapshotAllShardCommittee[shardID] = clonedShardCommittee
	}

	if !reflect.DeepEqual(beaconCommittee, snapshotBeaconCommittee) {
		return []incognitokey.CommitteePublicKey{}, nil, fmt.Errorf("Failed To Clone Beacon Committee, expect %+v but get %+v", beaconCommittee, snapshotBeaconCommittee)
	}
	if !reflect.DeepEqual(allShardCommittee, snapshotAllShardCommittee) {
		return []incognitokey.CommitteePublicKey{}, nil, fmt.Errorf("Failed To Clone Beacon Committee, expect %+v but get %+v", allShardCommittee, snapshotAllShardCommittee)
	}

	return snapshotBeaconCommittee, snapshotAllShardCommittee, nil
}
func snapshotRewardReceiver(rewardReceiver map[string]privacy.PaymentAddress) (map[string]privacy.PaymentAddress, error) {
	snapshotRewardReceiver := make(map[string]privacy.PaymentAddress)
	for k, v := range rewardReceiver {
		snapshotRewardReceiver[k] = v
	}
	if !reflect.DeepEqual(snapshotRewardReceiver, rewardReceiver) {
		return snapshotRewardReceiver, fmt.Errorf("Failed to Clone Reward Rewards, expect %+v but get %+v", rewardReceiver, snapshotRewardReceiver)
	}
	return snapshotRewardReceiver, nil
}
