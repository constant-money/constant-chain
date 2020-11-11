package blockchain

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"strconv"
	"testing"

	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type PDEFlowsSuite struct {
	suite.Suite
	currentPDEStateForProducer CurrentPDEState
	currentPDEStateForProcess  CurrentPDEState
}

func (suite *PDEFlowsSuite) SetupSuite() {
	suite.currentPDEStateForProducer = CurrentPDEState{
		WaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
		PDEPoolPairs:            make(map[string]*rawdbv2.PDEPoolForPair),
		PDEShares:               make(map[string]uint64),
	}
	suite.currentPDEStateForProcess = CurrentPDEState{
		WaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
		PDEPoolPairs:            make(map[string]*rawdbv2.PDEPoolForPair),
		PDEShares:               make(map[string]uint64),
	}
}

// All methods that begin with "Test" are run as tests within a
// suite.
func (suite *PDEFlowsSuite) TestSimulatedBeaconBlock1001() {
	fmt.Println("Running testcase: TestSimulatedBeaconBlock1001")
	bc := &BlockChain{}
	shardID := byte(1)
	beaconHeight := uint64(1001)
	contribInst1 := buildPDEContributionAction(
		"unique-pair-1",
		"contributor-address-1",
		1000000000000,
		"token-id-1",
	)
	contribInst2 := buildPDEContributionAction(
		"unique-pair-1",
		"contributor-address-1",
		2000000000000,
		"token-id-2",
	)
	contribInst3 := buildPDEContributionAction(
		"unique-pair-2",
		"contributor-address-2",
		5000000000000,
		"token-id-3",
	)
	tradeInst1 := buildPDETradeReqAction(
		"token-id-4",
		"token-id-3",
		100000,
		"trader-1",
	)
	withdrawalInst1 := buildPDEWithdrawReqAction(
		"withdrawer-address-1",
		"token-id-1",
		1000000000000,
		"token-id-2",
		2000000000000,
	)
	tradeInst2 := buildPDETradeReqAction(
		"token-id-2",
		"token-id-1",
		200000,
		"trader-2",
	)

	insts := [][]string{contribInst1[0], contribInst2[0], contribInst3[0], tradeInst1, tradeInst2, withdrawalInst1}
	newInsts := [][]string{}
	for _, inst := range insts {
		metaType, _ := strconv.Atoi(inst[0])
		contentStr := inst[1]
		newInst := [][]string{}
		var err error
		switch metaType {
		case metadata.PDEContributionMeta:
			newInst, err = bc.buildInstructionsForPDEContribution(contentStr, shardID, metaType)
		case metadata.PDETradeRequestMeta:
			newInst, err = bc.buildInstructionsForPDETrade(contentStr, shardID, metaType, &suite.currentPDEStateForProducer, beaconHeight-1)
		case metadata.PDEWithdrawalRequestMeta:
			newInst, err = bc.buildInstructionsForPDEWithdrawal(contentStr, shardID, metaType, &suite.currentPDEStateForProducer, beaconHeight-1)
		default:
			continue
		}
		suite.Equal(err, nil)
		newInsts = append(newInsts, newInst...)
	}

	suite.Equal(len(newInsts), 5)

	// skip withdrawal inst, and refund for 2 trade insts
	suite.Equal(newInsts[3][2], "refund")
	suite.Equal(newInsts[4][2], "refund")

	suite.Equal(len(suite.currentPDEStateForProducer.WaitingPDEContributions), 0)
	suite.Equal(len(suite.currentPDEStateForProducer.PDEPoolPairs), 0)
	suite.Equal(len(suite.currentPDEStateForProducer.PDEShares), 0)

	for _, inst := range newInsts {
		if len(inst) < 2 {
			continue // Not error, just not PDE instruction
		}
		var err error
		switch inst[0] {
		case strconv.Itoa(metadata.PDEContributionMeta):
			err = bc.processPDEContribution(beaconHeight-1, inst, &suite.currentPDEStateForProcess)
		case strconv.Itoa(metadata.PDETradeRequestMeta):
			err = bc.processPDETrade(beaconHeight-1, inst, &suite.currentPDEStateForProcess)
		case strconv.Itoa(metadata.PDEWithdrawalRequestMeta):
			err = bc.processPDEWithdrawal(beaconHeight-1, inst, &suite.currentPDEStateForProcess)
		}
		suite.Equal(err, nil)
	}

	// check current pde state values
	newPoolPairs := suite.currentPDEStateForProcess.PDEPoolPairs
	newWaitingPDEContribs := suite.currentPDEStateForProcess.WaitingPDEContributions
	newPDEShares := suite.currentPDEStateForProcess.PDEShares

	// waiting contributions
	suite.Equal(len(newWaitingPDEContribs), 1)
	waitingContrib := string(rawdbv2.BuildWaitingPDEContributionKey(beaconHeight-1, "unique-pair-2"))
	suite.Equal(newWaitingPDEContribs[waitingContrib].ContributorAddressStr, "contributor-address-2")
	suite.Equal(newWaitingPDEContribs[waitingContrib].TokenIDStr, "token-id-3")
	suite.Equal(newWaitingPDEContribs[waitingContrib].Amount, uint64(5000000000000))

	// pool pairs
	suite.Equal(len(newPoolPairs), 1)
	poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "token-id-1", "token-id-2"))
	suite.Equal(newPoolPairs[poolPairKey].Token1PoolValue, uint64(1000000000000))
	suite.Equal(newPoolPairs[poolPairKey].Token2PoolValue, uint64(2000000000000))

	// shares
	suite.Equal(len(newPDEShares), 2)
	shareKey1 := string(rawdbv2.BuildPDESharesKey(beaconHeight-1, "token-id-1", "token-id-2", "token-id-1", "contributor-address-1"))
	shareKey2 := string(rawdbv2.BuildPDESharesKey(beaconHeight-1, "token-id-1", "token-id-2", "token-id-2", "contributor-address-1"))
	suite.Equal(newPDEShares[shareKey1], uint64(1000000000000))
	suite.Equal(newPDEShares[shareKey2], uint64(2000000000000))

	// simulate storing pde state to db
	waitingContributionsWithNewKey := make(map[string]*rawdbv2.PDEContribution)
	poolPairsWithNewKey := make(map[string]*rawdbv2.PDEPoolForPair)
	sharesWithNewKey := make(map[string]uint64)
	for contribKey, contribution := range suite.currentPDEStateForProcess.WaitingPDEContributions {
		newKey := replaceNewBCHeightInKeyStr(contribKey, beaconHeight)
		waitingContributionsWithNewKey[newKey] = contribution
	}
	for poolPairKey, poolPair := range suite.currentPDEStateForProcess.PDEPoolPairs {
		newKey := replaceNewBCHeightInKeyStr(poolPairKey, beaconHeight)
		poolPairsWithNewKey[newKey] = poolPair
	}
	for sharesKey, shares := range suite.currentPDEStateForProcess.PDEShares {
		newKey := replaceNewBCHeightInKeyStr(sharesKey, beaconHeight)
		sharesWithNewKey[newKey] = shares
	}
	suite.currentPDEStateForProcess.WaitingPDEContributions = waitingContributionsWithNewKey
	suite.currentPDEStateForProcess.PDEPoolPairs = poolPairsWithNewKey
	suite.currentPDEStateForProcess.PDEShares = sharesWithNewKey

	// deep copy "value" of currentPDEStateForProcess to currentPDEStateForProducer in order to avoid side effect
	currentPDEStateForProcessBytes, _ := json.Marshal(suite.currentPDEStateForProcess)
	json.Unmarshal(currentPDEStateForProcessBytes, &suite.currentPDEStateForProducer)
}

func update(currentState *CurrentPDEState) {
	currentState.PDEShares["pdeshare-1001-token-id-1-token-id-2-token-id-2-contributor-address-1"] = 1234567
}

func (suite *PDEFlowsSuite) TestSimulatedBeaconBlock1002() {
	fmt.Println("Running testcase: TestSimulatedBeaconBlock1002")
	bc := &BlockChain{}
	shardID := byte(1)
	beaconHeight := uint64(1002)
	tradeInst1 := buildPDETradeReqAction(
		"token-id-1",
		"token-id-2",
		100000,
		"trader-1",
	)
	contribInst1 := buildPDEContributionAction( // contribute to the same token of last contribInst3 of block 1001
		"unique-pair-2",
		"contributor-address-3",
		4000000000000,
		"token-id-3",
	)
	contribInst2 := buildPDEContributionAction( // contribute to the remaining token of last contribInst3 of block 1001
		"unique-pair-2",
		"contributor-address-4",
		10000000000000,
		"token-id-4",
	)
	contribInst3 := buildPDEContributionAction( // contribute to the same token of last contribInst3 of block 1001
		"unique-pair-3",
		"contributor-address-5",
		4000000000000,
		"token-id-2",
	)
	tradeInst2 := buildPDETradeReqAction(
		"token-id-4",
		"token-id-3",
		400000,
		"trader-2",
	)
	tradeInst3 := buildPDETradeReqAction(
		"token-id-2",
		"token-id-1",
		300000,
		"trader-3",
	)
	contribInst4 := buildPDEContributionAction( // contribute to the remaining token of last contribInst3 of block 1001
		"unique-pair-3",
		"contributor-address-6",
		10000000000000,
		"token-id-1",
	)
	contribInst5 := buildPDEContributionAction(
		"unique-pair-4",
		"contributor-address-3",
		3000000000000,
		"token-id-3",
	)

	tradeInst4 := buildPDETradeReqAction(
		"token-id-3",
		"token-id-4",
		600000,
		"trader-4",
	)
	tradeInst5 := buildPDETradeReqAction(
		"token-id-3",
		"token-id-5",
		600000,
		"trader-5",
	)
	withdrawalInst1 := buildPDEWithdrawReqAction(
		"withdrawer-address-1",
		"token-id-1",
		1000000000000,
		"token-id-2",
		2000000000000,
	)
	withdrawalInst2 := buildPDEWithdrawReqAction(
		"contributor-address-1",
		"token-id-1",
		500000000000,
		"token-id-2",
		1000000000000,
	)
	withdrawalInst3 := buildPDEWithdrawReqAction(
		"contributor-address-1",
		"token-id-1",
		500000000000,
		"token-id-3",
		1000000000000,
	)

	// simulate beacon block producer
	insts := [][]string{tradeInst1, contribInst1[0], contribInst2[0], contribInst3[0], tradeInst2, tradeInst3, contribInst4[0], contribInst5[0], tradeInst4, tradeInst5, withdrawalInst1, withdrawalInst2, withdrawalInst3}
	newInsts := [][]string{}
	for _, inst := range insts {
		metaType, _ := strconv.Atoi(inst[0])
		contentStr := inst[1]
		newInst := [][]string{}
		var err error
		switch metaType {
		case metadata.PDEContributionMeta:
			newInst, err = bc.buildInstructionsForPDEContribution(contentStr, shardID, metaType)
		case metadata.PDETradeRequestMeta:
			newInst, err = bc.buildInstructionsForPDETrade(contentStr, shardID, metaType, &suite.currentPDEStateForProducer, beaconHeight-1)
		case metadata.PDEWithdrawalRequestMeta:
			newInst, err = bc.buildInstructionsForPDEWithdrawal(contentStr, shardID, metaType, &suite.currentPDEStateForProducer, beaconHeight-1)
		default:
			continue
		}
		suite.Equal(err, nil)
		newInsts = append(newInsts, newInst...)
	}

	suite.Equal(len(newInsts), 12)

	suite.Equal(newInsts[4][2], "refund")
	suite.Equal(newInsts[8][2], "refund")
	suite.Equal(newInsts[9][2], "refund")
	suite.Equal(newInsts[10][0], strconv.Itoa(metadata.PDEWithdrawalRequestMeta))
	suite.Equal(newInsts[11][0], strconv.Itoa(metadata.PDEWithdrawalRequestMeta))

	poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "token-id-1", "token-id-2"))
	suite.Equal(suite.currentPDEStateForProducer.PDEPoolPairs[poolPairKey].Token1PoolValue, uint64(500000125063))
	suite.Equal(suite.currentPDEStateForProducer.PDEPoolPairs[poolPairKey].Token2PoolValue, uint64(999999750751))

	sharesKey1 := string(rawdbv2.BuildPDESharesKey(beaconHeight-1, "token-id-1", "token-id-2", "token-id-1", "contributor-address-1"))
	sharesKey2 := string(rawdbv2.BuildPDESharesKey(beaconHeight-1, "token-id-1", "token-id-2", "token-id-2", "contributor-address-1"))
	suite.Equal(suite.currentPDEStateForProducer.PDEShares[sharesKey1], uint64(500000000000))
	suite.Equal(suite.currentPDEStateForProducer.PDEShares[sharesKey2], uint64(1000000000000))

	// simulate beacon block process
	for _, inst := range newInsts {
		if len(inst) < 2 {
			continue // Not error, just not PDE instruction
		}
		var err error
		switch inst[0] {
		case strconv.Itoa(metadata.PDEContributionMeta):
			err = bc.processPDEContribution(beaconHeight-1, inst, &suite.currentPDEStateForProcess)
		case strconv.Itoa(metadata.PDETradeRequestMeta):
			err = bc.processPDETrade(beaconHeight-1, inst, &suite.currentPDEStateForProcess)
		case strconv.Itoa(metadata.PDEWithdrawalRequestMeta):
			err = bc.processPDEWithdrawal(beaconHeight-1, inst, &suite.currentPDEStateForProcess)
		}
		suite.Equal(err, nil)
	}

	suite.Equal(len(suite.currentPDEStateForProcess.WaitingPDEContributions), 1)
	waitingContributionKey := string(rawdbv2.BuildWaitingPDEContributionKey(beaconHeight-1, "unique-pair-4"))
	suite.Equal(suite.currentPDEStateForProcess.WaitingPDEContributions[waitingContributionKey].ContributorAddressStr, "contributor-address-3")
	suite.Equal(suite.currentPDEStateForProcess.WaitingPDEContributions[waitingContributionKey].TokenIDStr, "token-id-3")
	suite.Equal(suite.currentPDEStateForProcess.WaitingPDEContributions[waitingContributionKey].Amount, uint64(3000000000000))

	suite.Equal(len(suite.currentPDEStateForProcess.PDEPoolPairs), 2)
	poolPairKey1 := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "token-id-1", "token-id-2"))
	poolPairKey2 := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "token-id-3", "token-id-4"))
	suite.Equal(suite.currentPDEStateForProcess.PDEPoolPairs[poolPairKey1].Token1PoolValue, uint64(10500000125063))
	suite.Equal(suite.currentPDEStateForProcess.PDEPoolPairs[poolPairKey1].Token2PoolValue, uint64(4999999750751))
	suite.Equal(suite.currentPDEStateForProcess.PDEPoolPairs[poolPairKey2].Token1PoolValue, uint64(9000000000000))
	suite.Equal(suite.currentPDEStateForProcess.PDEPoolPairs[poolPairKey2].Token2PoolValue, uint64(10000000000000))

	suite.Equal(len(suite.currentPDEStateForProcess.PDEShares), 6)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestPDEFlowsSuite(t *testing.T) {
	fmt.Println("Initialized...")
	suite.Run(t, new(PDEFlowsSuite))
}
