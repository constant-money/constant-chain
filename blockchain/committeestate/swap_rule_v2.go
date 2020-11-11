package committeestate

import (
	"fmt"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/instruction"
)

// createSwapShardInstructionV2 create swap instruction and new substitutes list
// swap procedure is only completed after both swap out and swap in is performed
// Output
// #1: swap instruction
// #2: new committee list
// #3: new substitute list
// #3: swapped out committee list
func createSwapShardInstructionV2(
	shardID byte,
	substitutes, committees []string,
	minCommitteeSize int,
	maxCommitteeSize int,
	typeIns int,
	numberOfFixedValidator int,
) (*instruction.SwapShardInstruction, []string, []string, []string) {
	committees, swappedOutCommittees :=
		normalSwapOut(committees, substitutes, minCommitteeSize, numberOfFixedValidator)

	newCommittees, newSubstitutes, swapInCommittees :=
		swapInAfterSwapOut(committees, substitutes, maxCommitteeSize)

	if len(swapInCommittees) == 0 && len(swappedOutCommittees) == 0 {
		return instruction.NewSwapShardInstruction(), newCommittees, newSubstitutes, swappedOutCommittees
	}

	swapShardInstruction := instruction.NewSwapShardInstructionWithValue(
		swapInCommittees,
		swappedOutCommittees,
		int(shardID),
		typeIns,
	)

	return swapShardInstruction, newCommittees, newSubstitutes, swappedOutCommittees
}

// removeValidatorV2 remove validator and return removed list
// return validator list after remove
// parameter:
// #1: list of full validator
// #2: list of removed validator
func removeValidatorV2(validators []string, removedValidators []string) ([]string, error) {
	// if number of pending validator is less or equal than offset, set offset equal to number of pending validator
	for _, removedValidator := range removedValidators {
		found := false
		index := 0
		for i, validator := range validators {
			if validator == removedValidator {
				found = true
				index = i
				break
			}
		}
		if found {
			validators = append(validators[:index], validators[index+1:]...)
		} else {
			return []string{}, fmt.Errorf("Try to remove validator %+v but not found in list %+v", removedValidator, validators)
		}
	}
	return validators, nil
}

// getSwapOutOffset assumes that numberOfFixedValidator <= minCommitteeSize and won't replace fixed nodes
// CONDITION:
// #1 swapOutOffset <= floor(numberOfCommittees/3)
// #2 swapOutOffset <=  numberOfCommittees - numberOfFixedValidator
// #3 committees length after both swap out and swap in must remain >= minCommitteeSize
// #4 swap operation must begin from start position (which is fixed node validator position) as a queue
// #5 number of swap out nodes >= number of swap in nodes
func getSwapOutOffset(numberOfSubstitutes, numberOfCommittees, numberOfFixedValidator, minCommitteeSize int) int {

	swapOffset := numberOfCommittees / MAX_SWAP_OR_ASSIGN_PERCENT
	if swapOffset == 0 {
		return 0
	}

	if swapOffset > numberOfCommittees-numberOfFixedValidator {
		swapOffset = numberOfCommittees - numberOfFixedValidator
	}

	noReplaceOffset := 0
	for swapOffset > 0 && numberOfCommittees > minCommitteeSize {
		swapOffset--
		noReplaceOffset++
		numberOfCommittees--
	}

	replaceSwapOffset := swapOffset
	if numberOfSubstitutes < swapOffset {
		replaceSwapOffset = numberOfSubstitutes
	}

	return noReplaceOffset + replaceSwapOffset
}

// normalSwapOut swap node out of committee
// after normal swap out, committees list could have invalid length
// Output:
// #1: new committees list
// #2: swapped out committees list
func normalSwapOut(
	committees []string,
	substitutes []string,
	minCommitteeSize int,
	numberOfFixedValidator int,
) (
	[]string,
	[]string,
) {
	if len(committees) == numberOfFixedValidator {
		return committees, []string{}
	}

	fixedCommittees := make([]string, len(committees[:numberOfFixedValidator]))
	copy(fixedCommittees, committees[:numberOfFixedValidator])
	flexCommittees := make([]string, len(committees[numberOfFixedValidator:]))
	copy(flexCommittees, committees[numberOfFixedValidator:])
	normalSwapOutCommittees := []string{}

	swapOutOffset := getSwapOutOffset(len(substitutes), len(committees), numberOfFixedValidator, minCommitteeSize)
	if swapOutOffset > 0 {
		normalSwapOutCommittees = flexCommittees[:swapOutOffset]
		flexCommittees = flexCommittees[swapOutOffset:]
	}

	committees = append(fixedCommittees, flexCommittees...)
	return committees, normalSwapOutCommittees
}

// swapInAfterSwapOut must be perform after normalSwapOut function is executed
// swap in as many as possible
// output:
// #1 new committee list
// #2 new substitutes list
// #3 swapped in committee list (from substitutes)
func swapInAfterSwapOut(committees, substitutes []string, maxCommitteeSize int) (
	[]string,
	[]string,
	[]string,
) {
	vacantSlot := maxCommitteeSize - len(committees)
	if vacantSlot > len(substitutes) {
		vacantSlot = len(substitutes)
	}
	newCommittees := substitutes[:vacantSlot]
	committees = append(committees, newCommittees...)
	substitutes = substitutes[vacantSlot:]
	return committees, substitutes, newCommittees
}

// assignShardCandidateV2 assign unassignedCommonPool into shard pool with random number
func assignShardCandidateV2(candidates []string, numberOfValidators []int, rand int64) map[byte][]string {
	total := 0
	for _, v := range numberOfValidators {
		total += v
	}
	n := byte(len(numberOfValidators))
	sortedShardIDs := sortShardIDByIncreaseOrder(numberOfValidators)
	m := getShardIDPositionFromArray(sortedShardIDs)
	assignedCandidates := make(map[byte][]string)
	candidateRandomShardID := make(map[string]byte)
	for _, candidate := range candidates {
		randomPosition := calculateCandidatePosition(candidate, rand, total)
		shardID := 0
		tempPosition := numberOfValidators[shardID]
		for randomPosition > tempPosition {
			shardID++
			tempPosition += numberOfValidators[shardID]
		}
		candidateRandomShardID[candidate] = byte(shardID)
	}
	for _, candidate := range candidates {
		randomShardID := candidateRandomShardID[candidate]
		assignShardID := sortedShardIDs[n-1-m[randomShardID]]
		assignedCandidates[byte(assignShardID)] = append(assignedCandidates[byte(assignShardID)], candidate)
	}
	return assignedCandidates
}

// calculateCandidatePosition calculate reverse shardID for candidate
// randomPosition = sum(hash(candidate+rand)) % total, if randomPosition == 0 then randomPosition = 1
// randomPosition in range (1, total)
func calculateCandidatePosition(candidate string, rand int64, total int) (pos int) {
	seed := candidate + fmt.Sprintf("%v", rand)
	hash := common.HashB([]byte(seed))
	data := 0
	for _, v := range hash {
		data += int(v)
	}
	pos = data % total
	if pos == 0 {
		pos = 1
	}
	return pos
}

// sortShardIDByIncreaseOrder take an array and sort array, return sorted index of array
func sortShardIDByIncreaseOrder(arr []int) []byte {
	sortedIndex := []byte{}
	tempArr := []struct {
		shardID byte
		value   int
	}{}
	for i, v := range arr {
		tempArr = append(tempArr, struct {
			shardID byte
			value   int
		}{byte(i), v})
	}
	sort.Slice(tempArr, func(i, j int) bool {
		return tempArr[i].value < tempArr[j].value
	})
	for _, v := range tempArr {
		sortedIndex = append(sortedIndex, v.shardID)
	}
	return sortedIndex
}

func getShardIDPositionFromArray(arr []byte) map[byte]byte {
	m := make(map[byte]byte)
	for i, v := range arr {
		m[v] = byte(i)
	}
	return m
}
