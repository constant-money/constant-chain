package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/portalv4"
	pv4Common "github.com/incognitochain/incognito-chain/portalv4/common"
	pv4Meta "github.com/incognitochain/incognito-chain/portalv4/metadata"
	portalprocessv4 "github.com/incognitochain/incognito-chain/portalv4/portalprocess"
	portalTokensV4 "github.com/incognitochain/incognito-chain/portalv4/portaltokens"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"
)

var _ = func() (_ struct{}) {
	Logger.Init(common.NewBackend(nil).Logger("test", true))
	portalprocessv4.Logger.Init(common.NewBackend(nil).Logger("test", true))
	portalTokensV4.Logger.Init(common.NewBackend(nil).Logger("test", true))
	pv4Meta.Logger.Init(common.NewBackend(nil).Logger("test", true))
	//pv4Common.Logger.Init(common.NewBackend(nil).Logger("test", true))
	Logger.log.Info("This runs before init()!")
	return
}()

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type PortalTestSuiteV4 struct {
	suite.Suite
	currentPortalStateForProducer portalprocessv4.CurrentPortalV4State
	currentPortalStateForProcess  portalprocessv4.CurrentPortalV4State

	sdb          *statedb.StateDB
	portalParams portalv4.PortalParams
	blockChain   *BlockChain
}

const USER_BTC_ADDRESS_1 = "12ok3D39W4AZj4aF2rmgzqys3BB4uhcXVN"

func (s *PortalTestSuiteV4) SetupTest() {
	dbPath, err := ioutil.TempDir(os.TempDir(), "portal_test_statedb_")
	if err != nil {
		panic(err)
	}
	diskBD, _ := incdb.Open("leveldb", dbPath)
	warperDBStatedbTest := statedb.NewDatabaseAccessWarper(diskBD)
	emptyRoot := common.HexToHash(common.HexEmptyRoot)
	stateDB, _ := statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)

	s.sdb = stateDB

	s.currentPortalStateForProducer = portalprocessv4.CurrentPortalV4State{
		WaitingUnshieldRequests:   map[string]map[string]*statedb.WaitingUnshieldRequest{},
		UTXOs:                     map[string]map[string]*statedb.UTXO{},
		ProcessedUnshieldRequests: map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{},
		ShieldingExternalTx:       map[string]map[string]*statedb.ShieldingRequest{},
	}
	s.currentPortalStateForProcess = portalprocessv4.CurrentPortalV4State{
		WaitingUnshieldRequests:   map[string]map[string]*statedb.WaitingUnshieldRequest{},
		UTXOs:                     map[string]map[string]*statedb.UTXO{},
		ProcessedUnshieldRequests: map[string]map[string]*statedb.ProcessedUnshieldRequestBatch{},
		ShieldingExternalTx:       map[string]map[string]*statedb.ShieldingRequest{},
	}
	s.portalParams = portalv4.PortalParams{
		MultiSigAddresses: map[string]string{
			pv4Common.PortalBTCIDStr: "",
		},
		MultiSigScriptHexEncode: map[string]string{
			pv4Common.PortalBTCIDStr: "",
		},
		PortalTokens: map[string]portalTokensV4.PortalTokenProcessor{
			pv4Common.PortalBTCIDStr: &portalTokensV4.PortalBTCTokenProcessor{
				&portalTokensV4.PortalToken{
					ChainID:        "Bitcoin-Testnet",
					MinTokenAmount: 10,
				},
			},
		},
		FeeUnshields: map[string]uint64{
			pv4Common.PortalBTCIDStr: 100000, // in nano pBTC - 10000 satoshi ~ 4 usd
		},
		BatchNumBlks:               45,
		PortalReplacementAddress:   "",
		MaxFeeForEachStep:          0,
		TimeSpaceForFeeReplacement: 0,
	}
	s.blockChain = &BlockChain{
		config: Config{
			ChainParams: &Params{
				MinBeaconBlockInterval: 40 * time.Second,
				MinShardBlockInterval:  40 * time.Second,
				Epoch:                  100,
				PortalV4Params: map[uint64]portalv4.PortalParams{
					0: s.portalParams,
				},
			},
		},
	}
}

type portalV4InstForProducer struct {
	inst         []string
	optionalData map[string]interface{}
}

func producerPortalInstructionsV4(
	blockchain basemeta.ChainRetriever,
	beaconHeight uint64,
	shardHeights map[byte]uint64,
	insts []portalV4InstForProducer,
	currentPortalState *portalprocessv4.CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	shardID byte,
	pm *portalprocessv4.PortalV4Manager,
) ([][]string, error) {
	var newInsts [][]string

	for _, item := range insts {
		inst := item.inst
		optionalData := item.optionalData

		metaType, _ := strconv.Atoi(inst[0])
		contentStr := inst[1]
		portalProcessor := pm.PortalInstructions[metaType]
		newInst, err := portalProcessor.BuildNewInsts(
			blockchain,
			contentStr,
			shardID,
			currentPortalState,
			beaconHeight,
			shardHeights,
			portalParams,
			optionalData,
		)
		if err != nil {
			Logger.log.Error(err)
			return newInsts, err
		}

		newInsts = append(newInsts, newInst...)
	}

	return newInsts, nil
}

func processPortalInstructionsV4(
	blockchain basemeta.ChainRetriever,
	beaconHeight uint64,
	insts [][]string,
	portalStateDB *statedb.StateDB,
	currentPortalState *portalprocessv4.CurrentPortalV4State,
	portalParams portalv4.PortalParams,
	pm *portalprocessv4.PortalV4Manager,
) error {
	updatingInfoByTokenID := map[common.Hash]basemeta.UpdatingInfo{}
	for _, inst := range insts {
		if len(inst) < 4 {
			continue // Not error, just not Portal instruction
		}

		var err error
		metaType, _ := strconv.Atoi(inst[0])
		processor := pm.PortalInstructions[metaType]
		if processor != nil {
			err = processor.ProcessInsts(portalStateDB, beaconHeight, inst, currentPortalState, portalParams, updatingInfoByTokenID)
			if err != nil {
				Logger.log.Errorf("Process portal instruction err: %v, inst %+v", err, inst)
			}
			continue
		}
	}
	// update info of bridge portal token
	for _, updatingInfo := range updatingInfoByTokenID {
		var updatingAmt uint64
		var updatingType string
		if updatingInfo.CountUpAmt > updatingInfo.DeductAmt {
			updatingAmt = updatingInfo.CountUpAmt - updatingInfo.DeductAmt
			updatingType = "+"
		}
		if updatingInfo.CountUpAmt < updatingInfo.DeductAmt {
			updatingAmt = updatingInfo.DeductAmt - updatingInfo.CountUpAmt
			updatingType = "-"
		}
		err := statedb.UpdateBridgeTokenInfo(
			portalStateDB,
			updatingInfo.TokenID,
			updatingInfo.ExternalTokenID,
			updatingInfo.IsCentralized,
			updatingAmt,
			updatingType,
		)
		if err != nil {
			return err
		}
	}

	// store updated currentPortalState to leveldb with new beacon height
	err := portalprocessv4.StorePortalV4StateToDB(portalStateDB, currentPortalState)
	if err != nil {
		Logger.log.Error(err)
	}

	return nil
}

/*
	Feature 6: Users redeem request
*/
type TestCaseUnshieldRequest struct {
	tokenID string
	unshieldAmount uint64
	incAddressStr string
	remoteAddress string
	txId string
	isExisted bool
}

type ExpectedResultUnshieldRequest struct {
	waitingUnshieldReqs map[string]map[string]*statedb.WaitingUnshieldRequest
	//custodianPool     map[string]*statedb.CustodianState
	//waitingRedeemReq  map[string]*statedb.RedeemRequest
	numBeaconInsts    uint
	statusInsts       []string
}

func (s *PortalTestSuiteV4) SetupTestUnshieldRequest() {
	// do nothing
}

func buildTestCaseAndExpectedResultUnshieldRequest() ([]TestCaseUnshieldRequest, *ExpectedResultUnshieldRequest) {
	beaconHeight := uint64(1003)
	// build test cases
	testcases := []TestCaseUnshieldRequest{
		// valid unshield request
		{
			tokenID:                 pv4Common.PortalBNBIDStr,
			unshieldAmount:            1 * 1e9,
			incAddressStr: USER_INC_ADDRESS_1,
			remoteAddress:           USER_BTC_ADDRESS_1,
			txId: common.HashH([]byte{1}).String(),
			isExisted: false,
		},
		// valid unshield request
		{
			tokenID:                 pv4Common.PortalBNBIDStr,
			unshieldAmount:            0.5 * 1e9,
			incAddressStr: USER_INC_ADDRESS_1,
			remoteAddress:           USER_BTC_ADDRESS_1,
			txId: common.HashH([]byte{2}).String(),
			isExisted: false,
		},
		// invalid unshield request
		{
			tokenID:                 pv4Common.PortalBNBIDStr,
			unshieldAmount:            1 * 1e9,
			incAddressStr: USER_INC_ADDRESS_1,
			remoteAddress:           USER_BTC_ADDRESS_1,
			txId: common.HashH([]byte{3}).String(),
			isExisted: true,
		},
	}

	// build expected results
	// waiting unshielding requests
	waitingUnshieldReqKey1 := statedb.GenerateWaitingUnshieldRequestObjectKey(pv4Common.PortalBTCIDStr, common.HashH([]byte{1}).String()).String()
	waitingUnshieldReq1 := statedb.NewWaitingUnshieldRequestStateWithValue(
		USER_BTC_ADDRESS_1,  1 * 1e9, common.HashH([]byte{1}).String(), beaconHeight)
	waitingUnshieldReqKey2 := statedb.GenerateWaitingUnshieldRequestObjectKey(pv4Common.PortalBTCIDStr, common.HashH([]byte{2}).String()).String()
	waitingUnshieldReq2 := statedb.NewWaitingUnshieldRequestStateWithValue(
		USER_BTC_ADDRESS_1,  0.5 * 1e9, common.HashH([]byte{2}).String(), beaconHeight)

	expectedRes := &ExpectedResultUnshieldRequest{
		waitingUnshieldReqs: map[string]map[string]*statedb.WaitingUnshieldRequest{
			pv4Common.PortalBTCIDStr: {
				waitingUnshieldReqKey1: waitingUnshieldReq1,
				waitingUnshieldReqKey2: waitingUnshieldReq2,
			},
		},
		numBeaconInsts: 3,
		statusInsts: []string{
			pv4Common.PortalRequestAcceptedChainStatus,
			pv4Common.PortalRequestAcceptedChainStatus,
			pv4Common.PortalRequestRejectedChainStatus,
		},
	}

	return testcases, expectedRes
}

func buildPortalUnshieldRequestAction(
	tokenID string,
	unshieldAmount uint64,
	incAddressStr string,
	remoteAddress string,
	txID string,
	shardID byte,
) []string {
	data := pv4Meta.PortalUnshieldRequest{
		MetadataBase: basemeta.MetadataBase{
			Type: basemeta.PortalBurnPTokenMeta,
		},
		IncAddressStr:  incAddressStr,
		RemoteAddress:  remoteAddress,
		TokenID:        tokenID,
		UnshieldAmount: unshieldAmount,
	}
	txIDHash, _ := common.Hash{}.NewHashFromStr(txID)
	actionContent := pv4Meta.PortalUnshieldRequestAction{
		Meta:        data,
		TxReqID:     *txIDHash,
		ShardID:     shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(basemeta.PortalBurnPTokenMeta), actionContentBase64Str}
}

func buildUnshieldRequestActionsFromTcs(tcs []TestCaseUnshieldRequest, shardID byte, shardHeight uint64) []portalV4InstForProducer {
	insts := []portalV4InstForProducer{}

	for _, tc := range tcs {
		inst := buildPortalUnshieldRequestAction(
			tc.tokenID, tc.unshieldAmount, tc.incAddressStr, tc.remoteAddress, tc.txId, shardID)
		insts = append(insts, portalV4InstForProducer{
			inst:         inst,
			optionalData: map[string]interface{}{
				"isExistUnshieldID" : tc.isExisted,
			},
		})
	}

	return insts
}

func (s *PortalTestSuiteV4) TestUnshieldRequest() {
	fmt.Println("Running TestUnshieldRequest - beacon height 1003 ...")
	bc := s.blockChain
	pm := portalprocessv4.NewPortalV4Manager()
	beaconHeight := uint64(1003)
	shardHeight := uint64(1003)
	shardHeights := map[byte]uint64{
		0: uint64(1003),
	}
	shardID := byte(0)

	s.SetupTestUnshieldRequest()

	// build test cases
	testcases, expectedResult := buildTestCaseAndExpectedResultUnshieldRequest()

	// build actions from testcases
	instsForProducer := buildUnshieldRequestActionsFromTcs(testcases, shardID, shardHeight)

	// producer instructions
	newInsts, err := producerPortalInstructionsV4(
		bc, beaconHeight-1, shardHeights, instsForProducer, &s.currentPortalStateForProducer, s.portalParams, shardID, pm)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructionsV4(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, pm)

	// check results
	s.Equal(expectedResult.numBeaconInsts, uint(len(newInsts)))
	s.Equal(nil, err)

	for i, inst := range newInsts {
		s.Equal(expectedResult.statusInsts[i], inst[2], "Instruction index %v", i)
	}

	s.Equal(expectedResult.waitingUnshieldReqs, s.currentPortalStateForProducer.WaitingUnshieldRequests)

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

func TestPortalSuiteV4(t *testing.T) {
	suite.Run(t, new(PortalTestSuiteV4))
}