package blockchain

import (
	"io/ioutil"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/stretchr/testify/suite"
	"strconv"
	"testing"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type PDETestSuiteV2 struct {
	suite.Suite
	currentPDEStateForProducer CurrentPDEState
	currentPDEStateForProcess  CurrentPDEState
	sdb *statedb.StateDB
}

func (s *PDETestSuiteV2) SetupTest() {
	dbPath, err := ioutil.TempDir(os.TempDir(), "test_statedb_")
	if err != nil {
		panic(err)
	}
	diskBD, _ := incdb.Open("leveldb", dbPath)
	warperDBStatedbTest := statedb.NewDatabaseAccessWarper(diskBD)
	emptyRoot := common.HexToHash(common.HexEmptyRoot)
	stateDB, _ := statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)

	s.sdb = stateDB
	s.currentPDEStateForProducer = CurrentPDEState{
		WaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
		DeletedWaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
		PDEPoolPairs:            make(map[string]*rawdbv2.PDEPoolForPair),
		PDEShares:               make(map[string]uint64),
		PDETradingFees:          make(map[string]uint64),
	}
	s.currentPDEStateForProcess = CurrentPDEState{
		WaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
		DeletedWaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
		PDEPoolPairs:            make(map[string]*rawdbv2.PDEPoolForPair),
		PDEShares:               make(map[string]uint64),
		PDETradingFees:          make(map[string]uint64),
	}
}

func buildPDEPRVRequiredContributionAction(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
) []string {
	metadataBase := metadata.MetadataBase{
		Type: metadata.PDEPRVRequiredContributionRequestMeta,
	}
	pdeContribution := metadata.PDEContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
	}
	pdeContribution.MetadataBase = metadataBase
	actionContent := metadata.PDEContributionAction{
		Meta:    pdeContribution,
		TxReqID: common.Hash{},
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(metadata.PDEPRVRequiredContributionRequestMeta), actionContentBase64Str}
	return action
}

func getMatchedNReturnedContributionMetaFromInst(contentInst string) metadata.PDEMatchedNReturnedContribution {
	var data metadata.PDEMatchedNReturnedContribution
	_ = json.Unmarshal([]byte(contentInst), &data)
	return data
}

func getTradeMetaFromAction(contentInst string) metadata.PDECrossPoolTradeRequest {
	var data metadata.PDECrossPoolTradeRequestAction
	dataBytes, _ := base64.StdEncoding.DecodeString(contentInst)
	_ = json.Unmarshal(dataBytes, &data)
	return data.Meta
}

func getPDECrossPoolTradeAcceptedContentFromInst(contentInst string) []metadata.PDECrossPoolTradeAcceptedContent {
	var data []metadata.PDECrossPoolTradeAcceptedContent
	_ = json.Unmarshal([]byte(contentInst), &data)
	return data
}

func getPDERefundCrossPoolTradeContentFromInst(contentInst string) metadata.PDERefundCrossPoolTrade {
	var data metadata.PDERefundCrossPoolTrade
	_ = json.Unmarshal([]byte(contentInst), &data)
	return data
}

func getPDETradeFeeDistributionFromInst(contentInst string) []*tradingFeeForContributorByPair {
	var data []*tradingFeeForContributorByPair
	_ = json.Unmarshal([]byte(contentInst), &data)
	return data
}

func getPDEWithdrawAcceptedContentFromInst(contentInst string) metadata.PDEWithdrawalAcceptedContent {
	var data metadata.PDEWithdrawalAcceptedContent
	_ = json.Unmarshal([]byte(contentInst), &data)
	return data
}

func buildPDECrossPoolTradeReqAction(
	tokenIDToBuyStr string,
	tokenIDToSellStr string,
	sellAmount uint64,
	minAcceptableAmount uint64,
	tradingFee uint64,
	traderAddressStr string,
) []string {
	metadataBase := metadata.MetadataBase{
		Type: metadata.PDECrossPoolTradeRequestMeta,
	}
	pdeTradeRequest := metadata.PDECrossPoolTradeRequest{
		TokenIDToBuyStr:     tokenIDToBuyStr,
		TokenIDToSellStr:    tokenIDToSellStr,
		SellAmount:          sellAmount,
		MinAcceptableAmount: minAcceptableAmount,
		TradingFee:          tradingFee,
		TraderAddressStr:    traderAddressStr,
	}
	pdeTradeRequest.MetadataBase = metadataBase
	actionContent := metadata.PDECrossPoolTradeRequestAction{
		Meta:    pdeTradeRequest,
		TxReqID: common.Hash{},
		ShardID: 1,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(metadata.PDECrossPoolTradeRequestMeta), actionContentBase64Str}
}

func buildPDEFeeWithdrawalRequestAction(
	withdrawerAddressStr string,
	withdrawalToken1IDStr string,
	withdrawalToken2IDStr string,
	withdrawalFeeAmt uint64,
) []string {
	feeWithdrawalRequest := metadata.PDEFeeWithdrawalRequest{
		WithdrawerAddressStr:  withdrawerAddressStr,
		WithdrawalToken1IDStr: withdrawalToken1IDStr,
		WithdrawalToken2IDStr: withdrawalToken2IDStr,
		WithdrawalFeeAmt:      withdrawalFeeAmt,
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PDEFeeWithdrawalRequestMeta,
		},
	}
	actionContent := metadata.PDEFeeWithdrawalRequestAction{
		Meta:    feeWithdrawalRequest,
		TxReqID: common.Hash{},
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(metadata.PDEFeeWithdrawalRequestMeta), actionContentBase64Str}
	return action
}

func buildPDEWithdrawalRequestAction(
	withdrawerAddressStr string,
	withdrawalToken1IDStr string,
	withdrawalToken2IDStr string,
	withdrawalShareAmt uint64,
) []string {
	feeWithdrawalRequest := metadata.PDEWithdrawalRequest{
		WithdrawerAddressStr:  withdrawerAddressStr,
		WithdrawalToken1IDStr: withdrawalToken1IDStr,
		WithdrawalToken2IDStr: withdrawalToken2IDStr,
		WithdrawalShareAmt:    withdrawalShareAmt,
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PDEWithdrawalRequestMeta,
		},
	}
	actionContent := metadata.PDEWithdrawalRequestAction{
		Meta:    feeWithdrawalRequest,
		TxReqID: common.Hash{},
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(metadata.PDEWithdrawalRequestMeta), actionContentBase64Str}
	return action
}

func processAllNewInsts(
	blockchain *BlockChain,
	insts [][]string,
	pdexStateDB *statedb.StateDB,
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
) {
	for _, inst := range insts {
		if len(inst) < 2 {
			continue // Not error, just not PDE instruction
		}
		var err error
		switch inst[0] {
		case strconv.Itoa(metadata.PDEContributionMeta):
			err = blockchain.processPDEContributionV2(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDEPRVRequiredContributionRequestMeta):
			err = blockchain.processPDEContributionV2(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDETradeRequestMeta):
			err = blockchain.processPDETrade(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDECrossPoolTradeRequestMeta):
			err = blockchain.processPDECrossPoolTrade(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDEWithdrawalRequestMeta):
			err = blockchain.processPDEWithdrawal(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDEFeeWithdrawalRequestMeta):
			err = blockchain.processPDEFeeWithdrawal(pdexStateDB, beaconHeight, inst, currentPDEState)
		case strconv.Itoa(metadata.PDETradingFeesDistributionMeta):
			err = blockchain.processPDETradingFeesDistribution(pdexStateDB, beaconHeight, inst, currentPDEState)
		}
		if err != nil {
			fmt.Printf("An error occured while process instruction: %v\n", inst)
		}
	}
}

// All methods that begin with "Test" are run as tests within a
// suite.
// test contribution
func (s *PDETestSuiteV2) TestSimulatedBeaconBlock1001() {
	fmt.Println("Running testcase: TestSimulatedBeaconBlock1001")
	bc := &BlockChain{}
	shardID := byte(1)
	beaconHeight := uint64(1001)

	// case 1: (valid) one of two contributions (in an unique ID) is PRV contribution
	contribInst1 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		1000000000000,
		common.PRVIDStr,
	)
	contribInst2 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		2000000000000,
		"tokenID1",
	)

	// case 2: (invalid) there is no PRV contribution in an unique ID contribution
	contribInst3 := buildPDEPRVRequiredContributionAction(
		"unique-pair-2",
		"contributorAddress1",
		1000000000000,
		"tokenID1",
	)
	contribInst4 := buildPDEPRVRequiredContributionAction(
		"unique-pair-2",
		"contributorAddress1",
		2000000000000,
		"tokenID2",
	)

	// case 3: (valid) contribute on the existing pool pair with correct proportion
	contribInst5 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		500000000000,
		common.PRVIDStr,
	)
	contribInst6 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		1000000000000,
		"tokenID1",
	)

	// case 4: (valid) contribute on the existing pool pair with incorrect proportion
	contribInst7 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		1000000000000,
		common.PRVIDStr,
	)
	contribInst8 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		3000000000000,
		"tokenID1",
	)

	// case 5: (invalid) different contributor address
	contribInst9 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		1000000000000,
		common.PRVIDStr,
	)
	contribInst10 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress2",
		2000000000000,
		"tokenID2",
	)

	insts := [][]string{
		contribInst1, contribInst2,
		contribInst3, contribInst4,
		contribInst5, contribInst6,
		contribInst7, contribInst8,
		contribInst9, contribInst10,
	}

	newInsts := [][]string{}
	for _, inst := range insts {
		metaType, _ := strconv.Atoi(inst[0])
		contentStr := inst[1]
		newInst := [][]string{}
		var err error
		switch metaType {
		case metadata.PDEPRVRequiredContributionRequestMeta:
			newInst, err = bc.buildInstructionsForPDEContribution(contentStr, shardID, metaType, &s.currentPDEStateForProducer, beaconHeight-1, true)
		//case metadata.PDETradeRequestMeta:
		//	newInst, err = bc.buildInstructionsForPDETrade(contentStr, shardID, metaType, &suite.currentPDEStateForProducer, beaconHeight-1)
		//case metadata.PDEWithdrawalRequestMeta:
		//	newInst, err = bc.buildInstructionsForPDEWithdrawal(contentStr, shardID, metaType, &suite.currentPDEStateForProducer, beaconHeight-1)
		default:
			continue
		}
		s.Equal(err, nil)
		newInsts = append(newInsts, newInst...)
	}

	s.Equal(len(newInsts), 14)

	// run functions for processing block with its instructions
	processAllNewInsts(
		bc,
		newInsts,
		s.sdb,
		beaconHeight - 1,
		&s.currentPDEStateForProcess,
	)

	s.currentPDEStateForProcess.DeletedWaitingPDEContributions = make(map[string]*rawdbv2.PDEContribution)
	s.Equal(true, reflect.DeepEqual(&s.currentPDEStateForProcess, &s.currentPDEStateForProducer))

	// check result of functions
	// case 1:
	s.Equal("waiting", newInsts[0][2])
	s.Equal("matched", newInsts[1][2])

	// case 2:
	s.Equal("waiting", newInsts[2][2])
	s.Equal("refund", newInsts[3][2])
	s.Equal("refund", newInsts[4][2])

	// case 3:
	s.Equal("waiting", newInsts[5][2])
	s.Equal("matchedNReturned", newInsts[6][2])
	s.Equal("matchedNReturned", newInsts[7][2])

	incomingContrib := getMatchedNReturnedContributionMetaFromInst(newInsts[6][3])
	s.Equal(uint64(1000000000000), incomingContrib.ActualContributedAmount)
	s.Equal(uint64(0), incomingContrib.ReturnedContributedAmount)

	waitingContrib := getMatchedNReturnedContributionMetaFromInst(newInsts[7][3])
	s.Equal(uint64(500000000000), waitingContrib.ActualContributedAmount)
	s.Equal(uint64(0), waitingContrib.ReturnedContributedAmount)

	// case 4:
	s.Equal("waiting", newInsts[8][2])
	s.Equal("matchedNReturned", newInsts[9][2])
	s.Equal("matchedNReturned", newInsts[10][2])

	incomingContrib2 := getMatchedNReturnedContributionMetaFromInst(newInsts[9][3])
	s.Equal(uint64(2000000000000), incomingContrib2.ActualContributedAmount)
	s.Equal(uint64(1000000000000), incomingContrib2.ReturnedContributedAmount)

	waitingContrib2 := getMatchedNReturnedContributionMetaFromInst(newInsts[10][3])
	s.Equal(uint64(1000000000000), waitingContrib2.ActualContributedAmount)
	s.Equal(uint64(0), waitingContrib2.ReturnedContributedAmount)

	// case 5:
	s.Equal("waiting", newInsts[11][2])
	s.Equal("refund", newInsts[12][2])
	s.Equal("refund", newInsts[13][2])

	// at the end
	poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "tokenID1", common.PRVIDStr))
	fmt.Printf("s.currentPDEStateForProducer.PDEPoolPairs: %v\n", s.currentPDEStateForProducer.PDEPoolPairs)
	s.Equal(s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey].Token1PoolValue, uint64(2500000000000))
	s.Equal(s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey].Token2PoolValue, uint64(5000000000000))
}

// test trading and withdrawing contribution
func (s *PDETestSuiteV2) TestSimulatedBeaconBlock1002() {
	fmt.Println("Running testcase: TestSimulatedBeaconBlock1002")
	bc := &BlockChain{}
	shardID := byte(1)
	beaconHeight := uint64(1002)

	// set up: add pool pairs
	// pair PRV - tokenID1
	// contributorAddress1 : 50% - 50%
	contribInst1 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		500000000000,
		common.PRVIDStr,
	)
	contribInst2 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		1000000000000,
		"tokenID1",
	)
	// contributorAddress2
	contribInst3 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress2",
		500000000000,
		common.PRVIDStr,
	)
	contribInst4 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress2",
		1000000000000,
		"tokenID1",
	)

	// pair PRV - tokenID2
	// contributorAddress1 30% - 70%
	contribInst5 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		300000000000,
		common.PRVIDStr,
	)
	contribInst6 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress1",
		900000000000,
		"tokenID2",
	)
	// contributorAddress2
	contribInst7 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress2",
		700000000000,
		common.PRVIDStr,
	)
	contribInst8 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress2",
		2100000000000,
		"tokenID2",
	)

	// pair tokenID1 - tokenID3
	contribInst9 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress3",
		1000000000000,
		"tokenID1",
	)
	contribInst10 := buildPDEPRVRequiredContributionAction(
		"unique-pair-1",
		"contributorAddress3",
		1000000000000,
		"tokenID3",
	)

	setUpInsts := [][]string{
		contribInst1, contribInst2,
		contribInst3, contribInst4,
		contribInst5, contribInst6,
		contribInst7, contribInst8,
		contribInst9, contribInst10,
	}

	newSetupInsts := [][]string{}
	for _, inst := range setUpInsts {
		metaType, _ := strconv.Atoi(inst[0])
		contentStr := inst[1]
		newInst := [][]string{}
		var err error
		switch metaType {
		case metadata.PDEPRVRequiredContributionRequestMeta:
			newInst, err = bc.buildInstructionsForPDEContribution(contentStr, shardID, metaType, &s.currentPDEStateForProducer, beaconHeight-1, true)
		default:
			continue
		}
		s.Equal(err, nil)
		newSetupInsts = append(newSetupInsts, newInst...)
	}

	fmt.Println("======== Pool Pair ========")
	PRV_ID1_PairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "tokenID1", common.PRVIDStr))
	fmt.Printf("s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID1_PairKey].Token1PoolValue: %v\n", s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID1_PairKey].Token1PoolValue)
	fmt.Printf("s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID1_PairKey].Token2PoolValue: %v\n", s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID1_PairKey].Token2PoolValue)
	PRV_ID2_PairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "tokenID2", common.PRVIDStr))
	fmt.Printf("s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID2_PairKey].Token1PoolValue: %v\n", s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID2_PairKey].Token1PoolValue)
	fmt.Printf("s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID2_PairKey].Token2PoolValue: %v\n", s.currentPDEStateForProducer.PDEPoolPairs[PRV_ID2_PairKey].Token2PoolValue)
	fmt.Println("============================")

	// test cases for trading
	// case 1 (valid): directly trade, sell PRV to buy tokenID1 with minAcceptableAmount = receivedAmount
	// 500000 PRV => 1998001998 tokenID1
	tradeInst1 := buildPDECrossPoolTradeReqAction(
		"tokenID1",
		common.PRVIDStr,
		1000000000,
		2000000,
		30,
		"trader1",
	)

	// proportion trade6 = trade7
	tradeInst2 := buildPDECrossPoolTradeReqAction(
		common.PRVIDStr,
		"tokenID2",
		2000000000,
		100000,
		49,
		"trader1",
	)
	tradeInst3 := buildPDECrossPoolTradeReqAction(
		"tokenID1",
		"tokenID2",
		2000000000,
		100000,
		50,
		"trader1",
	)
	// invalid minAcceptableAmount
	tradeInst4 := buildPDECrossPoolTradeReqAction(
		"tokenID2",
		"tokenID1",
		2000000000,
		20000000000,
		50,
		"trader1",
	)
	// refund
	tradeInst5 := buildPDECrossPoolTradeReqAction(
		common.PRVIDStr,
		"tokenID3",
		2000000000,
		1998001998,
		50,
		"trader1",
	)
	// refund
	tradeInst6 := buildPDECrossPoolTradeReqAction(
		"tokenID1",
		"tokenID3",
		2000000000,
		1998001998,
		50,
		"trader1",
	)
	tradeInst7 := buildPDECrossPoolTradeReqAction(
		"tokenID1",
		common.PRVIDStr,
		2000000000,
		200000000000,
		50,
		"trader1",
	)

	tradeInsts := [][]string{
		tradeInst1, tradeInst2, tradeInst3, tradeInst4,
		tradeInst5, tradeInst6, tradeInst7,
	}

	pdeTradeActionsByShardID := make(map[byte][][]string, 0)
	pdeTradeActionsByShardID[shardID] = tradeInsts

	// sort trading instructions by fee
	sortedTradableActions, untradableActions := categorizeNSortPDECrossPoolTradeInstsByFee(
		beaconHeight-1,
		&s.currentPDEStateForProducer,
		pdeTradeActionsByShardID,
	)

	s.Equal(5, len(sortedTradableActions))
	s.Equal(2, len(untradableActions))

	// 7 => 6 => 8 => 3 => 1 => 11 => 5 => 2 => 4
	s.Equal(getTradeMetaFromAction(tradeInst3[1]), sortedTradableActions[0].Meta)
	s.Equal(getTradeMetaFromAction(tradeInst2[1]), sortedTradableActions[1].Meta)
	s.Equal(getTradeMetaFromAction(tradeInst4[1]), sortedTradableActions[2].Meta)
	s.Equal(getTradeMetaFromAction(tradeInst1[1]), sortedTradableActions[3].Meta)
	s.Equal(getTradeMetaFromAction(tradeInst7[1]), sortedTradableActions[4].Meta)

	tradableInsts, tradingFeePair := bc.buildInstsForSortedTradableActions(&s.currentPDEStateForProducer, beaconHeight-1, sortedTradableActions)
	untradableInsts := bc.buildInstsForUntradableActions(untradableActions)
	tradingFeesDistInst := bc.buildInstForTradingFeesDist(&s.currentPDEStateForProducer, beaconHeight-1, tradingFeePair)
	newTradeInsts := append(tradableInsts, untradableInsts...)
	newTradeInsts = append(newTradeInsts, tradingFeesDistInst)

	s.Equal(12, len(newTradeInsts))

	// run functions for processing block with its instructions
	processAllNewInsts(
		bc,
		append(newSetupInsts, newTradeInsts...),
		s.sdb,
		beaconHeight - 1,
		&s.currentPDEStateForProcess,
	)

	s.Equal(true, reflect.DeepEqual(&s.currentPDEStateForProcess.PDEPoolPairs, &s.currentPDEStateForProducer.PDEPoolPairs))
	s.Equal(true, reflect.DeepEqual(&s.currentPDEStateForProcess.PDEShares, &s.currentPDEStateForProducer.PDEShares))
	s.Equal(27, int(s.currentPDEStateForProcess.PDETradingFees["pdetradingfee-1001-0000000000000000000000000000000000000000000000000000000000000004-tokenID1-contributorAddress1"]))
	s.Equal(28, int(s.currentPDEStateForProcess.PDETradingFees["pdetradingfee-1001-0000000000000000000000000000000000000000000000000000000000000004-tokenID1-contributorAddress2"]))
	s.Equal(22, int(s.currentPDEStateForProcess.PDETradingFees["pdetradingfee-1001-0000000000000000000000000000000000000000000000000000000000000004-tokenID2-contributorAddress1"]))
	s.Equal(52, int(s.currentPDEStateForProcess.PDETradingFees["pdetradingfee-1001-0000000000000000000000000000000000000000000000000000000000000004-tokenID2-contributorAddress2"]))

	// check new trade insts
	// new trade instruction 0 (for trade request 3)
	// cross pool : token2 => (PRV) => token1
	s.Equal("xPoolTradeAccepted", newTradeInsts[0][2])
	expectTradeContent0 := []metadata.PDECrossPoolTradeAcceptedContent{
		{
			TraderAddressStr: "trader1",
			TokenIDToBuyStr:  common.PRVIDStr,
			ReceiveAmount:    666222518,
			Token1IDStr:      common.PRVIDStr,
			Token2IDStr:      "tokenID2",
			Token1PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "-",
				Value:    666222518,
			},
			Token2PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "+",
				Value:    2000000000,
			},
			ShardID:       shardID,
			RequestedTxID: common.Hash{},
			AddingFee:     25,
		},
		{
			TraderAddressStr: "trader1",
			TokenIDToBuyStr:  "tokenID1",
			ReceiveAmount:    1331557922,
			Token1IDStr:      common.PRVIDStr,
			Token2IDStr:      "tokenID1",
			Token1PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "+",
				Value:    666222518,
			},
			Token2PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "-",
				Value:    1331557922,
			},
			ShardID:       shardID,
			RequestedTxID: common.Hash{},
			AddingFee:     25,
		},
	}
	tradeContent0 := getPDECrossPoolTradeAcceptedContentFromInst(newTradeInsts[0][3])
	s.Equal(expectTradeContent0, tradeContent0)
	// new pool:
	// PRV-ID1: 1000666222518 - 1998668442078
	// PRV-ID2: 999333777482 - 3002000000000

	// new trade instruction 1 (for trade request 2)
	// direct trade : token2 => PRV
	s.Equal("xPoolTradeAccepted", newTradeInsts[1][2])
	expectTradeContent1 := []metadata.PDECrossPoolTradeAcceptedContent{
		{
			TraderAddressStr: "trader1",
			TokenIDToBuyStr:  common.PRVIDStr,
			ReceiveAmount:    665335404,
			Token1IDStr:      common.PRVIDStr,
			Token2IDStr:      "tokenID2",
			Token1PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "-",
				Value:    665335404,
			},
			Token2PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "+",
				Value:    2000000000,
			},
			ShardID:       shardID,
			RequestedTxID: common.Hash{},
			AddingFee:     49,
		},
	}
	tradeContent1 := getPDECrossPoolTradeAcceptedContentFromInst(newTradeInsts[1][3])
	s.Equal(expectTradeContent1, tradeContent1)
	// new pool:
	// PRV-ID1: 1000666222518 - 1998668442078
	// PRV-ID2: 998668442078 - 3004000000000

	// new trade instruction 2,3 (for trade request 4)
	// invalid minAcceptableAmount : token1 => token2
	// refund trading fee and selling amount
	s.Equal("xPoolTradeRefundFee", newTradeInsts[2][2])
	s.Equal("xPoolTradeRefundSellingToken", newTradeInsts[3][2])
	expectRefundFeeContent2 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       common.PRVIDStr,
		Amount:           50,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundFeeContent2 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[2][3])
	s.Equal(expectRefundFeeContent2, actualRefundFeeContent2)

	expectRefundSellingAmountContent2 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       "tokenID1",
		Amount:           2000000000,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundSellingAmountContent2 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[3][3])
	s.Equal(expectRefundSellingAmountContent2, actualRefundSellingAmountContent2)
	// new pool (doesn't change):
	// PRV-ID1: 1000666222518 - 1998668442078
	// PRV-ID2: 998668442078 - 3004000000000

	// new trade instruction 4 (for trade request 1)
	// direct trade : PRV => token1 (valid minAcceptableAmount)
	s.Equal("xPoolTradeAccepted", newTradeInsts[4][2])
	expectTradeContent4 := []metadata.PDECrossPoolTradeAcceptedContent{
		{
			TraderAddressStr: "trader1",
			TokenIDToBuyStr:  "tokenID1",
			ReceiveAmount:    1995343755,
			Token1IDStr:      common.PRVIDStr,
			Token2IDStr:      "tokenID1",
			Token1PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "+",
				Value:    1000000000,
			},
			Token2PoolValueOperation: metadata.TokenPoolValueOperation{
				Operator: "-",
				Value:    1995343755,
			},
			ShardID:       shardID,
			RequestedTxID: common.Hash{},
			AddingFee:     30,
		},
	}
	tradeContent4 := getPDECrossPoolTradeAcceptedContentFromInst(newTradeInsts[4][3])
	s.Equal(expectTradeContent4, tradeContent4)
	//fmt.Println("new pool prv: ", 1998668442078 - 1995343755)
	// new pool:
	// PRV-ID1: 1001666222518 - 1996673098323
	// PRV-ID2: 998668442078 - 3004000000000

	// new trade instruction 5,6 (for trade request 7)
	// invalid minAcceptableAmount : PRV => token1
	// refund trading fee and selling amount
	s.Equal("xPoolTradeRefundFee", newTradeInsts[5][2])
	s.Equal("xPoolTradeRefundSellingToken", newTradeInsts[6][2])
	expectRefundFeeContent5 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       common.PRVIDStr,
		Amount:           50,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundFeeContent5 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[5][3])
	s.Equal(expectRefundFeeContent5, actualRefundFeeContent5)

	expectRefundSellingAmountContent6 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       common.PRVIDStr,
		Amount:           2000000000,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundSellingAmountContent6 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[6][3])
	s.Equal(expectRefundSellingAmountContent6, actualRefundSellingAmountContent6)
	// new pool (doesn't change):
	// PRV-ID1: 1001666222518 - 1996673098323
	// PRV-ID2: 998668442078 - 3004000000000

	// new trade instruction 7,8 (for trade request 5)
	// untradable request: token3 => PRV
	// refund trading fee and selling amount
	s.Equal("xPoolTradeRefundFee", newTradeInsts[7][2])
	s.Equal("xPoolTradeRefundSellingToken", newTradeInsts[8][2])
	expectRefundFeeContent7 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       common.PRVIDStr,
		Amount:           50,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundFeeContent7 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[7][3])
	s.Equal(expectRefundFeeContent7, actualRefundFeeContent7)

	expectRefundSellingAmountContent8 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       "tokenID3",
		Amount:           2000000000,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundSellingAmountContent8 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[8][3])
	s.Equal(expectRefundSellingAmountContent8, actualRefundSellingAmountContent8)
	// new pool (doesn't change):
	// PRV-ID1: 1001666222518 - 1996673098323
	// PRV-ID2: 998668442078 - 3004000000000

	// new trade instruction 9,10 (for trade request 6)
	// untradable request: token3 => token1
	// refund trading fee and selling amount
	s.Equal("xPoolTradeRefundFee", newTradeInsts[9][2])
	s.Equal("xPoolTradeRefundSellingToken", newTradeInsts[10][2])
	expectRefundFeeContent9 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       common.PRVIDStr,
		Amount:           50,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundFeeContent9 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[9][3])
	s.Equal(expectRefundFeeContent9, actualRefundFeeContent9)

	expectRefundSellingAmountContent10 := metadata.PDERefundCrossPoolTrade{
		TraderAddressStr: "trader1",
		TokenIDStr:       "tokenID3",
		Amount:           2000000000,
		ShardID:          shardID,
		TxReqID:          common.Hash{},
	}
	actualRefundSellingAmountContent10 := getPDERefundCrossPoolTradeContentFromInst(newTradeInsts[10][3])
	s.Equal(expectRefundSellingAmountContent10, actualRefundSellingAmountContent10)
	// new pool (doesn't change):
	// PRV-ID1: 1001666222518 - 1996673098323
	// PRV-ID2: 998668442078 - 3004000000000

	fmt.Printf("s.currentPDEStateForProducer.PDEShares %v\n", s.currentPDEStateForProducer.PDEShares)

	// new trade instruction 11 (for trade fee distribution)
	// Note: order of tokenID in pair
	expectTradeFeeDistributionMeta := []*tradingFeeForContributorByPair{
		{
			ContributorAddressStr: "contributorAddress1",
			FeeAmt:                27,
			Token1IDStr:           common.PRVIDStr,
			Token2IDStr:           "tokenID1",
		},
		{
			ContributorAddressStr: "contributorAddress2",
			FeeAmt:                28,
			Token1IDStr:           common.PRVIDStr,
			Token2IDStr:           "tokenID1",
		},
		{
			ContributorAddressStr: "contributorAddress1",
			FeeAmt:                22,
			Token1IDStr:           common.PRVIDStr,
			Token2IDStr:           "tokenID2",
		},
		{
			ContributorAddressStr: "contributorAddress2",
			FeeAmt:                52,
			Token1IDStr:           common.PRVIDStr,
			Token2IDStr:           "tokenID2",
		},
	}

	actualTradeFeeDistributionMeta := getPDETradeFeeDistributionFromInst(newTradeInsts[11][3])
	s.Equal(expectTradeFeeDistributionMeta, actualTradeFeeDistributionMeta)

	// at the end
	// check pool pair
	poolPairKey1 := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "tokenID1", common.PRVIDStr))
	s.Equal(s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey1].Token1PoolValue, uint64(1001666222518))
	s.Equal(s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey1].Token2PoolValue, uint64(1996673098323))

	poolPairKey2 := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight-1, "tokenID2", common.PRVIDStr))
	s.Equal(s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey2].Token1PoolValue, uint64(998668442078))
	s.Equal(s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey2].Token2PoolValue, uint64(3004000000000))
}

func (s *PDETestSuiteV2) SetupTestWithdraw(beaconHeight uint64) {
	s.currentPDEStateForProducer = CurrentPDEState{
		WaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
		PDEPoolPairs:            make(map[string]*rawdbv2.PDEPoolForPair),
		PDEShares:               make(map[string]uint64),
		PDETradingFees:          make(map[string]uint64),
	}

	//// set up pool pair
	poolPairKey1 := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, common.PRVIDStr, "tokenID1"))
	poolPairKey2 := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, common.PRVIDStr, "tokenID2"))
	s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey1] = &rawdbv2.PDEPoolForPair{
		Token1IDStr:     common.PRVIDStr,
		Token1PoolValue: 1000000000000,
		Token2IDStr:     "tokenID1",
		Token2PoolValue: 2000000000000,
	}
	s.currentPDEStateForProducer.PDEPoolPairs[poolPairKey2] = &rawdbv2.PDEPoolForPair{
		Token1IDStr:     common.PRVIDStr,
		Token1PoolValue: 1000000000000,
		Token2IDStr:     "tokenID2",
		Token2PoolValue: 3000000000000,
	}

	// set up PDEShares
	shareKey1 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, common.PRVIDStr, "tokenID1", "contributorAddress1"))
	shareKey2 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, common.PRVIDStr, "tokenID1", "contributorAddress2"))
	shareKey3 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, common.PRVIDStr, "tokenID2", "contributorAddress1"))
	shareKey4 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, common.PRVIDStr, "tokenID2", "contributorAddress2"))
	s.currentPDEStateForProducer.PDEShares[shareKey1] = 500000000000 // 500 PRV
	s.currentPDEStateForProducer.PDEShares[shareKey2] = 500000000000 // 500 PRV
	s.currentPDEStateForProducer.PDEShares[shareKey3] = 300000000000 // 500 PRV
	s.currentPDEStateForProducer.PDEShares[shareKey4] = 700000000000 // 500 PRV

	// set up PDETradingFees
	tradeFeeKey1 := string(rawdbv2.BuildPDETradingFeeKey(beaconHeight, common.PRVIDStr, "tokenID1", "contributorAddress1"))
	tradeFeeKey2 := string(rawdbv2.BuildPDETradingFeeKey(beaconHeight, common.PRVIDStr, "tokenID1", "contributorAddress2"))
	tradeFeeKey3 := string(rawdbv2.BuildPDETradingFeeKey(beaconHeight, common.PRVIDStr, "tokenID2", "contributorAddress1"))
	tradeFeeKey4 := string(rawdbv2.BuildPDETradingFeeKey(beaconHeight, common.PRVIDStr, "tokenID2", "contributorAddress2"))
	s.currentPDEStateForProducer.PDETradingFees[tradeFeeKey1] = 500
	s.currentPDEStateForProducer.PDETradingFees[tradeFeeKey2] = 234
	s.currentPDEStateForProducer.PDETradingFees[tradeFeeKey3] = 123
	s.currentPDEStateForProducer.PDETradingFees[tradeFeeKey4] = 0
}

func (s *PDETestSuiteV2) TestSimulatedBeaconBlock1003() {
	fmt.Println("Running testcase: TestSimulatedBeaconBlock1003 - Test withdraw")
	bc := &BlockChain{}
	shardID := byte(1)
	beaconHeight := uint64(1003)

	s.SetupTestWithdraw(beaconHeight - 1)
	stateContent, _ := json.Marshal(s.currentPDEStateForProducer)
	json.Unmarshal(stateContent, &s.currentPDEStateForProcess)

	// s.currentPDEStateForProcess = s.currentPDEStateForProducer

	fmt.Printf("PDPDETradingFees before processing instructions : %v\n", s.currentPDEStateForProducer.PDETradingFees)

	// withdraw
	// invalid pair
	feeWithdrawInst1 := buildPDEFeeWithdrawalRequestAction(
		"contributorAddress2",
		"tokenID1",
		"tokenID2",
		10,
	)
	// invalid amount
	feeWithdrawInst2 := buildPDEFeeWithdrawalRequestAction(
		"contributorAddress2",
		common.PRVIDStr,
		"tokenID1",
		10000,
	)
	// accept
	feeWithdrawInst3 := buildPDEFeeWithdrawalRequestAction(
		"contributorAddress2",
		common.PRVIDStr,
		"tokenID1",
		10,
	)
	// accept
	feeWithdrawInst4 := buildPDEFeeWithdrawalRequestAction(
		"contributorAddress1",
		"tokenID1",
		common.PRVIDStr,
		10,
	)

	// share amount is greater than actual share
	withdrawInst5 := buildPDEWithdrawalRequestAction(
		"contributorAddress1",
		"tokenID1",
		common.PRVIDStr,
		5000000000000,
	)
	// invalid contributor address
	withdrawInst6 := buildPDEWithdrawalRequestAction(
		"contributorAddress10",
		"tokenID1",
		common.PRVIDStr,
		1000000000,
	)
	// invalid pool pair
	withdrawInst7 := buildPDEWithdrawalRequestAction(
		"contributorAddress1",
		"tokenID3",
		common.PRVIDStr,
		5000000000000,
	)
	// valid
	withdrawInst8 := buildPDEWithdrawalRequestAction(
		"contributorAddress2",
		"tokenID1",
		common.PRVIDStr,
		200000000000,
	)

	feeWithdrawInsts := [][]string{
		feeWithdrawInst1, feeWithdrawInst2,
		feeWithdrawInst3, feeWithdrawInst4,
		withdrawInst5, withdrawInst6,
		withdrawInst7, withdrawInst8,
	}

	fmt.Printf("PDEShare before testing: %v\n", s.currentPDEStateForProducer.PDEShares)

	newWithdrawInsts := [][]string{}
	for _, inst := range feeWithdrawInsts {
		metaType, _ := strconv.Atoi(inst[0])
		newInst := [][]string{}
		var err error
		switch metaType {
		case metadata.PDEFeeWithdrawalRequestMeta:
			newInst, err = bc.buildInstructionsForPDEFeeWithdrawal(
				inst[1], shardID, metadata.PDEFeeWithdrawalRequestMeta,
				&s.currentPDEStateForProducer, beaconHeight-1)
		case metadata.PDEWithdrawalRequestMeta:
			newInst, err = bc.buildInstructionsForPDEWithdrawal(
				inst[1], shardID, metadata.PDEWithdrawalRequestMeta,
				&s.currentPDEStateForProducer, beaconHeight-1)
		}
		s.Equal(nil, err)
		newWithdrawInsts = append(newWithdrawInsts, newInst...)
	}

	// run functions for processing block with its instructions
	processAllNewInsts(
		bc,
		newWithdrawInsts,
		s.sdb,
		beaconHeight - 1,
		&s.currentPDEStateForProcess,
	)
	s.Equal(true, reflect.DeepEqual(&s.currentPDEStateForProcess, &s.currentPDEStateForProducer))

	// check newWithdrawInsts
	s.Equal("rejected", newWithdrawInsts[0][2])
	s.Equal("rejected", newWithdrawInsts[1][2])
	s.Equal("accepted", newWithdrawInsts[2][2])
	s.Equal("accepted", newWithdrawInsts[3][2])

	s.Equal("accepted", newWithdrawInsts[4][2])
	s.Equal("accepted", newWithdrawInsts[5][2])
	s.Equal("rejected", newWithdrawInsts[6][2])
	s.Equal("rejected", newWithdrawInsts[7][2])
	s.Equal("accepted", newWithdrawInsts[8][2])
	s.Equal("accepted", newWithdrawInsts[9][2])

	expectWithdrawContent4 := metadata.PDEWithdrawalAcceptedContent{
		WithdrawalTokenIDStr: common.PRVIDStr,
		WithdrawerAddressStr: "contributorAddress1",
		DeductingPoolValue:   500000000000,
		DeductingShares:      500000000000,
		PairToken1IDStr:      "tokenID1",
		PairToken2IDStr:      common.PRVIDStr,
		TxReqID:              common.Hash{},
		ShardID:              shardID,
	}
	actualWithdrawContent4 := getPDEWithdrawAcceptedContentFromInst(newWithdrawInsts[4][3])
	s.Equal(expectWithdrawContent4, actualWithdrawContent4)

	expectWithdrawContent5 := metadata.PDEWithdrawalAcceptedContent{
		WithdrawalTokenIDStr: "tokenID1",
		WithdrawerAddressStr: "contributorAddress1",
		DeductingPoolValue:   1000000000000,
		DeductingShares:      0,
		PairToken1IDStr:      "tokenID1",
		PairToken2IDStr:      common.PRVIDStr,
		TxReqID:              common.Hash{},
		ShardID:              shardID,
	}
	actualWithdrawContent5 := getPDEWithdrawAcceptedContentFromInst(newWithdrawInsts[5][3])
	s.Equal(expectWithdrawContent5, actualWithdrawContent5)

	//check PDEShares
	shareKey1 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight-1, common.PRVIDStr, "tokenID1", "contributorAddress1"))
	shareKey2 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight-1, common.PRVIDStr, "tokenID1", "contributorAddress2"))
	shareKey3 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight-1, common.PRVIDStr, "tokenID2", "contributorAddress1"))
	shareKey4 := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight-1, common.PRVIDStr, "tokenID2", "contributorAddress2"))
	s.Equal(uint64(0), s.currentPDEStateForProducer.PDEShares[shareKey1])
	s.Equal(uint64(300000000000), s.currentPDEStateForProducer.PDEShares[shareKey2])
	s.Equal(uint64(300000000000), s.currentPDEStateForProducer.PDEShares[shareKey3])
	s.Equal(uint64(700000000000), s.currentPDEStateForProducer.PDEShares[shareKey4])
}

func TestPDETestSuiteV2(t *testing.T) {
	suite.Run(t, new(PDETestSuiteV2))
}
