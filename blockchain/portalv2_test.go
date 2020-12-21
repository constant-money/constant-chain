package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	typesBNB "github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/types/msg"
	txBNB "github.com/binance-chain/go-sdk/types/tx"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/relaying/bnb"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/types"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type PortalTestSuite struct {
	suite.Suite
	currentPortalStateForProducer CurrentPortalState
	currentPortalStateForProcess  CurrentPortalState
	sdb                           *statedb.StateDB
	portalParams                  PortalParams
	blockChain                    *BlockChain
}

const USER1_INC_ADDRESS = "12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC"
const USER2_INC_ADDRESS = "12S1a8VnkwhDTQWZ5PhdpySwiFZj7p8sKdG7oAQFZ3dLsWaV6fhDWk5aSFHpt1jcPBjY4sYgwqAqRzx3oTYDZCvCei1LSCdJARXWiyK"

func (s *PortalTestSuite) SetupTest() {
	dbPath, err := ioutil.TempDir(os.TempDir(), "portal_test_statedb_")
	if err != nil {
		panic(err)
	}
	diskBD, _ := incdb.Open("leveldb", dbPath)
	warperDBStatedbTest := statedb.NewDatabaseAccessWarper(diskBD)
	emptyRoot := common.HexToHash(common.HexEmptyRoot)
	stateDB, _ := statedb.NewWithPrefixTrie(emptyRoot, warperDBStatedbTest)

	s.sdb = stateDB

	finalExchangeRate := statedb.NewFinalExchangeRatesStateWithValue(
		map[string]statedb.FinalExchangeRatesDetail{
			common.PRVIDStr:       {Amount: 1000000},
			common.PortalBNBIDStr: {Amount: 20000000},
			common.PortalBTCIDStr: {Amount: 10000000000},
		})
	s.currentPortalStateForProducer = CurrentPortalState{
		CustodianPoolState:         map[string]*statedb.CustodianState{},
		WaitingPortingRequests:     map[string]*statedb.WaitingPortingRequest{},
		WaitingRedeemRequests:      map[string]*statedb.RedeemRequest{},
		MatchedRedeemRequests:      map[string]*statedb.RedeemRequest{},
		FinalExchangeRatesState:    finalExchangeRate,
		LiquidationPool:            map[string]*statedb.LiquidationPool{},
		LockedCollateralForRewards: new(statedb.LockedCollateralState),
		ExchangeRatesRequests:      map[string]*metadata.ExchangeRatesRequestStatus{},
	}
	s.currentPortalStateForProcess = CurrentPortalState{
		CustodianPoolState:         map[string]*statedb.CustodianState{},
		WaitingPortingRequests:     map[string]*statedb.WaitingPortingRequest{},
		WaitingRedeemRequests:      map[string]*statedb.RedeemRequest{},
		MatchedRedeemRequests:      map[string]*statedb.RedeemRequest{},
		FinalExchangeRatesState:    finalExchangeRate,
		LiquidationPool:            map[string]*statedb.LiquidationPool{},
		LockedCollateralForRewards: new(statedb.LockedCollateralState),
		ExchangeRatesRequests:      map[string]*metadata.ExchangeRatesRequestStatus{},
	}
	s.portalParams = PortalParams{
		TimeOutCustodianReturnPubToken:       24 * time.Hour,
		TimeOutWaitingPortingRequest:         24 * time.Hour,
		TimeOutWaitingRedeemRequest:          15 * time.Minute,
		MaxPercentLiquidatedCollateralAmount: 120,
		MaxPercentCustodianRewards:           10,
		MinPercentCustodianRewards:           1,
		MinLockCollateralAmountInEpoch:       5000 * 1e9, // 5000 prv
		MinPercentLockedCollateral:           200,
		TP120:                                120,
		TP130:                                130,
		MinPercentPortingFee:                 0.01,
		MinPercentRedeemFee:                  0.01,
	}
	s.blockChain = &BlockChain{
		config: Config{
			ChainParams: &Params{
				MinBeaconBlockInterval: 40 * time.Second,
				Epoch:                  100,
			},
		},
	}
}

/*
 Utility functions
*/

func exchangeRates(amount uint64, tokenIDFrom string, tokenIDTo string, finalExchangeRate *statedb.FinalExchangeRatesState) uint64 {
	rateFrom := finalExchangeRate.Rates()[tokenIDFrom].Amount
	rateTo := finalExchangeRate.Rates()[tokenIDTo].Amount
	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amount), new(big.Int).SetUint64(rateFrom))
	return new(big.Int).Div(tmp, new(big.Int).SetUint64(rateTo)).Uint64()
}

func getLockedCollateralAmount(
	portingAmount uint64, tokenID string, finalExchangeRate *statedb.FinalExchangeRatesState, percent uint64) uint64 {

	amount := new(big.Int).Mul(new(big.Int).SetUint64(portingAmount), new(big.Int).SetUint64(percent))
	amount = amount.Div(amount, new(big.Int).SetUint64(100))
	return exchangeRates(amount.Uint64(), tokenID, common.PRVIDStr, finalExchangeRate)
}

func getMinFee(amount uint64, tokenID string, finalExchangeRate *statedb.FinalExchangeRatesState, percent float64) uint64 {
	amountInPRV := exchangeRates(amount, tokenID, common.PRVIDStr, finalExchangeRate)
	fee := float64(amountInPRV) * percent / float64(100)
	return uint64(math.Round(fee))
}

func getUnlockAmount(totalLockedAmount uint64, totalPTokenAmount uint64, pTokenAmount uint64) uint64 {
	amount := new(big.Int).Mul(new(big.Int).SetUint64(pTokenAmount), new(big.Int).SetUint64(totalLockedAmount))
	amount = amount.Div(amount, new(big.Int).SetUint64(totalPTokenAmount))
	return amount.Uint64()
}

func (s *PortalTestSuite) TestGetLockedCollateralAmount() {
	portingAmount := uint64(0.3 * 1e9)
	tokenID := common.PortalBNBIDStr
	percent := s.portalParams.MinPercentLockedCollateral
	amount := getLockedCollateralAmount(portingAmount, tokenID, s.currentPortalStateForProducer.FinalExchangeRatesState, percent)
	fmt.Println("Result from TestGetLockedCollateralAmount: ", amount)
}

func (s *PortalTestSuite) TestGetMinFee() {
	amount := uint64(0.25 * 1e9)
	tokenID := common.PortalBTCIDStr
	percent := s.portalParams.MinPercentPortingFee

	fee := getMinFee(amount, tokenID, s.currentPortalStateForProducer.FinalExchangeRatesState, percent)
	fmt.Println("Result from TestGetMinFee: ", fee)
}

func (s *PortalTestSuite) TestGetUnlockAmount() {
	totalLockedAmount := uint64(40000000000)
	totalPTokenAmount := uint64(1 * 1e9)
	pTokenAmount := uint64(0.3 * 1e9)

	unlockAmount := getUnlockAmount(totalLockedAmount, totalPTokenAmount, pTokenAmount)
	fmt.Println("Result from TestGetUnlockAmount: ", unlockAmount)
}

func (s *PortalTestSuite) TestExchangeRate() {
	s.currentPortalStateForProducer.FinalExchangeRatesState = statedb.NewFinalExchangeRatesStateWithValue(
		map[string]statedb.FinalExchangeRatesDetail{
			common.PRVIDStr:       {Amount: 1000000},
			common.PortalBNBIDStr: {Amount: 40000000},
			common.PortalBTCIDStr: {Amount: 10000000000},
		})
	amount := uint64(0.7 * 1e9)
	tokenIDFrom := common.PortalBNBIDStr
	tokenIDTo := common.PRVIDStr
	convertAmount := exchangeRates(amount, tokenIDFrom, tokenIDTo, s.currentPortalStateForProducer.FinalExchangeRatesState)
	convertAmount = convertAmount * 120 / 100
	fmt.Println("Result from TestExchangeRate: ", convertAmount)
}

func cloneMap(m map[string]uint64) map[string]uint64 {
	if m == nil {
		return nil
	}
	newMap := make(map[string]uint64, len(m))
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func cloneCustodians(custodians map[string]*statedb.CustodianState) map[string]*statedb.CustodianState {
	newCustodians := make(map[string]*statedb.CustodianState, len(custodians))
	for key, cus := range custodians {
		newCustodians[key] = statedb.NewCustodianStateWithValue(
			cus.GetIncognitoAddress(),
			cus.GetTotalCollateral(),
			cus.GetFreeCollateral(),
			cloneMap(cus.GetHoldingPublicTokens()),
			cloneMap(cus.GetLockedAmountCollateral()),
			cus.GetRemoteAddresses(),
			cloneMap(cus.GetRewardAmount()),
		)
	}
	return newCustodians
}

func cloneMatchingPortingCustodians(custodians []*statedb.MatchingPortingCustodianDetail) []*statedb.MatchingPortingCustodianDetail {
	newMatchingCustodians := make([]*statedb.MatchingPortingCustodianDetail, len(custodians))
	for i, cus := range custodians {
		newMatchingCustodians[i] = &statedb.MatchingPortingCustodianDetail{
			IncAddress:             cus.IncAddress,
			RemoteAddress:          cus.RemoteAddress,
			Amount:                 cus.Amount,
			LockedAmountCollateral: cus.LockedAmountCollateral,
		}
	}
	return newMatchingCustodians
}

func cloneWPortingRequests(wPortingReqs map[string]*statedb.WaitingPortingRequest) map[string]*statedb.WaitingPortingRequest {
	newReqs := make(map[string]*statedb.WaitingPortingRequest, len(wPortingReqs))
	for key, req := range wPortingReqs {
		newReqs[key] = statedb.NewWaitingPortingRequestWithValue(
			req.UniquePortingID(),
			req.TxReqID(),
			req.TokenID(),
			req.PorterAddress(),
			req.Amount(),
			cloneMatchingPortingCustodians(req.Custodians()),
			req.PortingFee(),
			req.BeaconHeight(),
		)
	}
	return newReqs
}

func cloneRedeemRequests(redeemReqs map[string]*statedb.RedeemRequest) map[string]*statedb.RedeemRequest {
	newReqs := make(map[string]*statedb.RedeemRequest, len(redeemReqs))
	for key, req := range redeemReqs {
		newReqs[key] = statedb.NewRedeemRequestWithValue(
			req.GetUniqueRedeemID(),
			req.GetTokenID(),
			req.GetRedeemerAddress(),
			req.GetRedeemerRemoteAddress(),
			req.GetRedeemAmount(),
			req.GetCustodians(),
			req.GetRedeemFee(),
			req.GetBeaconHeight(),
			req.GetTxReqID(),
		)
	}
	return newReqs
}

// buildBNBProofFromTxs build a bnb proof for unit tests
func buildBNBProofFromTxs(blockHeight int64, txs *types.Txs, indexTx int) *bnb.BNBProof {
	proof := txs.Proof(indexTx)

	return &bnb.BNBProof{
		Proof:       &proof,
		BlockHeight: blockHeight,
	}
}

func createSendMsg(fromAddr string, transferInfo map[string]int64) msg.SendMsg {
	fromAccAddr, _ := typesBNB.AccAddressFromHex(fromAddr)

	transfer := make([]msg.Transfer, 0)
	totalAmountTransfer := int64(0)
	for toAddrStr, amount := range transferInfo {
		toAddr, _ := typesBNB.AccAddressFromHex(toAddrStr)
		transfer = append(transfer, msg.Transfer{
			ToAddr: toAddr,
			Coins: typesBNB.Coins{
				typesBNB.Coin{
					Denom:  bnb.DenomBNB,
					Amount: amount,
				},
			},
		})
		totalAmountTransfer += amount
	}

	fromCoins := typesBNB.Coins{
		typesBNB.Coin{
			Denom:  bnb.DenomBNB,
			Amount: totalAmountTransfer,
		},
	}

	sendMsg := msg.CreateSendMsg(fromAccAddr, fromCoins, transfer)
	return sendMsg
}

func createTxs(fromAddr string, tranferInfo map[string]int64, memo string) *types.Txs {
	// create SendMsg
	sendMsg := createSendMsg(fromAddr, tranferInfo)

	// create StdTx
	stdTx := txBNB.NewStdTx([]msg.Msg{sendMsg}, []txBNB.StdSignature{}, memo, int64(0), []byte{})

	txBytes, _ := types.GetCodec().MarshalBinaryLengthPrefixed(stdTx)
	txs := &types.Txs{txBytes}
	return txs
}

// createMemo create memo for porting tx or redeem tx
// if custodianIncAddr is empty, create memo for porting
func createMemo(id string, custodianIncAddr string) string {
	type PortingMemoBNB struct {
		PortingID string `json:"PortingID"`
	}

	type RedeemMemoBNB struct {
		RedeemID                  string `json:"RedeemID"`
		CustodianIncognitoAddress string `json:"CustodianIncognitoAddress"`
	}

	memo := ""

	if custodianIncAddr == "" {
		memoPorting := PortingMemoBNB{PortingID: id}
		memoPortingBytes, _ := json.Marshal(memoPorting)
		memo = base64.StdEncoding.EncodeToString(memoPortingBytes)
	} else {
		memoRedeem := RedeemMemoBNB{RedeemID: id, CustodianIncognitoAddress: custodianIncAddr}
		memoRedeemBytes, _ := json.Marshal(memoRedeem)
		memoRedeemHash := common.HashB(memoRedeemBytes)
		memo = base64.StdEncoding.EncodeToString(memoRedeemHash)
	}

	return memo
}

func buildBNBProof(blockHeight int64, fromAddr string, transferInfo map[string]int64, id string, msg string) (string, []byte) {
	indexTx := 0

	// build memo attach to tx
	memo := createMemo(id, msg)

	txs := createTxs(fromAddr, transferInfo, memo)

	bnbProof := buildBNBProofFromTxs(blockHeight, txs, indexTx)
	bnbProofBytes, _ := json.Marshal(bnbProof)
	bnbProofStr := base64.StdEncoding.EncodeToString(bnbProofBytes)
	rootHash := txs.Hash()

	fmt.Println("Result from TestBuildBNBProof bnbProofStr: ", bnbProofStr)
	fmt.Printf("Result from TestBuildBNBProof rootHash: %#v\n", txs.Hash())

	return bnbProofStr, rootHash
}

func (s *PortalTestSuite) TestBuildBNBProof() {
	//buildBNBProof()
}

func producerPortalInstructions(
	blockchain *BlockChain,
	beaconHeight uint64,
	insts [][]string,
	portalStateDB *statedb.StateDB,
	currentPortalState *CurrentPortalState,
	portalParams PortalParams,
	shardID byte,
	newMatchedRedeemReqIDs []string,
) ([][]string, error) {
	var err error
	var newInst [][]string
	var newInsts [][]string
	for _, inst := range insts {
		switch inst[0] {
		//exchange rates
		case strconv.Itoa(metadata.PortalExchangeRatesMeta):
			newInst, err = blockchain.buildInstructionsForExchangeRates(
				inst[1], shardID, metadata.PortalExchangeRatesMeta, currentPortalState, beaconHeight, portalParams)

		// custodians deposit collateral
		case strconv.Itoa(metadata.PortalCustodianDepositMeta):
			newInst, err = blockchain.buildInstructionsForCustodianDeposit(
				inst[1], shardID, metadata.PortalCustodianDepositMeta, currentPortalState, beaconHeight, portalParams)
		// porting request
		case strconv.Itoa(metadata.PortalUserRegisterMeta):
			newInst, err = blockchain.buildInstructionsForPortingRequest(
				portalStateDB, inst[1], shardID, metadata.PortalUserRegisterMeta, currentPortalState, beaconHeight, portalParams)
		// submit proof to request ptokens
		case strconv.Itoa(metadata.PortalUserRequestPTokenMeta):
			newInst, err = blockchain.buildInstructionsForReqPTokens(
				portalStateDB, inst[1], shardID, metadata.PortalUserRequestPTokenMeta, currentPortalState, beaconHeight, portalParams)

		// redeem request
		case strconv.Itoa(metadata.PortalRedeemRequestMeta):
			newInst, err = blockchain.buildInstructionsForRedeemRequest(
				portalStateDB, inst[1], shardID, metadata.PortalRedeemRequestMeta, currentPortalState, beaconHeight, portalParams)
		// custodian request matching waiting redeem requests
		case strconv.Itoa(metadata.PortalReqMatchingRedeemMeta):
			newInst, newMatchedRedeemReqIDs, err = blockchain.buildInstructionsForReqMatchingRedeem(
				portalStateDB, inst[1], shardID, metadata.PortalReqMatchingRedeemMeta, currentPortalState, beaconHeight, portalParams, newMatchedRedeemReqIDs)
		// submit proof to request unlock collateral
		case strconv.Itoa(metadata.PortalRequestUnlockCollateralMeta):
			newInst, err = blockchain.buildInstructionsForReqUnlockCollateral(
				portalStateDB, inst[1], shardID, metadata.PortalRequestUnlockCollateralMeta, currentPortalState, beaconHeight, portalParams)

		// custodian request withdraw reward
		case strconv.Itoa(metadata.PortalRequestWithdrawRewardMeta):
			newInst, err = blockchain.buildInstructionsForReqWithdrawPortalReward(
				inst[1], shardID, metadata.PortalRequestWithdrawRewardMeta, currentPortalState, beaconHeight)
		// custodian request withdraw collaterals
		case strconv.Itoa(metadata.PortalCustodianWithdrawRequestMeta):
			newInst, err = blockchain.buildInstructionsForCustodianWithdraw(
				inst[1], shardID, metadata.PortalCustodianWithdrawRequestMeta, currentPortalState, beaconHeight, portalParams)

		// custodian top-up collaterals for holding public tokens
		case strconv.Itoa(metadata.PortalLiquidationCustodianDepositMetaV2):
			newInst, err = blockchain.buildInstructionsForLiquidationCustodianDeposit(
				inst[1], shardID, metadata.PortalCustodianWithdrawRequestMeta, currentPortalState, beaconHeight, portalParams)
		// custodian top-up collaterals for waiting porting requests
		case strconv.Itoa(metadata.PortalTopUpWaitingPortingRequestMeta):
			newInst, err = blockchain.buildInstsForTopUpWaitingPorting(
				inst[1], shardID, metadata.PortalTopUpWaitingPortingRequestMeta, currentPortalState, beaconHeight, portalParams)
		// user redeem ptokens from liquidation pool to get PRV
		case strconv.Itoa(metadata.PortalRedeemLiquidateExchangeRatesMeta):
			newInst, err = blockchain.buildInstructionsForLiquidationRedeemPTokenExchangeRates(
				inst[1], shardID, metadata.PortalRedeemLiquidateExchangeRatesMeta, currentPortalState, beaconHeight, portalParams)
		}

		if err != nil {
			Logger.log.Error(err)
			return newInsts, err
		}

		newInsts = append(newInsts, newInst...)
	}

	return newInsts, nil
}

func processPortalInstructions(
	blockchain *BlockChain,
	beaconHeight uint64,
	insts [][]string,
	portalStateDB *statedb.StateDB,
	currentPortalState *CurrentPortalState,
	portalParams PortalParams,
	updatingInfoByTokenID map[common.Hash]UpdatingInfo,
) error {
	var err error
	for _, inst := range insts {
		if len(inst) < 4 {
			continue // Not error, just not Portal instruction
		}

		var err error
		switch inst[0] {
		//porting request
		case strconv.Itoa(metadata.PortalUserRegisterMeta):
			err = blockchain.processPortalUserRegister(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		//exchange rates
		case strconv.Itoa(metadata.PortalExchangeRatesMeta):
			err = blockchain.processPortalExchangeRates(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		//custodian withdraw
		case strconv.Itoa(metadata.PortalCustodianWithdrawRequestMeta):
			err = blockchain.processPortalCustodianWithdrawRequest(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		//liquidation exchange rates
		case strconv.Itoa(metadata.PortalLiquidateTPExchangeRatesMeta):
			err = blockchain.processLiquidationTopPercentileExchangeRates(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		//liquidation custodian deposit
		case strconv.Itoa(metadata.PortalLiquidationCustodianDepositMetaV2):
			err = blockchain.processPortalLiquidationCustodianDeposit(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		//waiting porting top up
		case strconv.Itoa(metadata.PortalTopUpWaitingPortingRequestMeta):
			err = blockchain.processPortalTopUpWaitingPorting(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		//liquidation user redeem
		case strconv.Itoa(metadata.PortalRedeemLiquidateExchangeRatesMeta):
			err = blockchain.processPortalRedeemLiquidateExchangeRates(portalStateDB, beaconHeight, inst, currentPortalState, portalParams, updatingInfoByTokenID)
		//custodian deposit
		case strconv.Itoa(metadata.PortalCustodianDepositMeta):
			err = blockchain.processPortalCustodianDeposit(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		// request ptoken
		case strconv.Itoa(metadata.PortalUserRequestPTokenMeta):
			err = blockchain.processPortalUserReqPToken(portalStateDB, beaconHeight, inst, currentPortalState, portalParams, updatingInfoByTokenID)
		// redeem request
		case strconv.Itoa(metadata.PortalRedeemRequestMeta):
			err = blockchain.processPortalRedeemRequest(portalStateDB, beaconHeight, inst, currentPortalState, portalParams, updatingInfoByTokenID)
		// request unlock collateral
		case strconv.Itoa(metadata.PortalRequestUnlockCollateralMeta):
			err = blockchain.processPortalUnlockCollateral(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		// liquidation custodian run away
		case strconv.Itoa(metadata.PortalLiquidateCustodianMeta):
			err = blockchain.processPortalLiquidateCustodian(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		// portal reward
		case strconv.Itoa(metadata.PortalRewardMeta):
			err = blockchain.processPortalReward(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		// request withdraw reward
		case strconv.Itoa(metadata.PortalRequestWithdrawRewardMeta):
			err = blockchain.processPortalWithdrawReward(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		// expired waiting porting request
		case strconv.Itoa(metadata.PortalExpiredWaitingPortingReqMeta):
			err = blockchain.processPortalExpiredPortingRequest(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		// total custodian reward instruction
		case strconv.Itoa(metadata.PortalTotalRewardCustodianMeta):
			err = blockchain.processPortalTotalCustodianReward(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		// custodian request matching waiting redeem requests
		case strconv.Itoa(metadata.PortalReqMatchingRedeemMeta):
			err = blockchain.processPortalReqMatchingRedeem(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		case strconv.Itoa(metadata.PortalPickMoreCustodianForRedeemMeta):
			err = blockchain.processPortalPickMoreCustodiansForTimeOutWaitingRedeemReq(portalStateDB, beaconHeight, inst, currentPortalState, portalParams)
		}

		if err != nil {
			Logger.log.Error(err)
			return nil
		}
	}

	//save final exchangeRates
	blockchain.pickExchangesRatesFinal(currentPortalState)

	// update info of bridge portal token
	for _, updatingInfo := range updatingInfoByTokenID {
		var updatingAmt uint64
		var updatingType string
		if updatingInfo.countUpAmt > updatingInfo.deductAmt {
			updatingAmt = updatingInfo.countUpAmt - updatingInfo.deductAmt
			updatingType = "+"
		}
		if updatingInfo.countUpAmt < updatingInfo.deductAmt {
			updatingAmt = updatingInfo.deductAmt - updatingInfo.countUpAmt
			updatingType = "-"
		}
		err := statedb.UpdateBridgeTokenInfo(
			portalStateDB,
			updatingInfo.tokenID,
			updatingInfo.externalTokenID,
			updatingInfo.isCentralized,
			updatingAmt,
			updatingType,
		)
		if err != nil {
			return err
		}
	}

	// store updated currentPortalState to leveldb with new beacon height
	err = storePortalStateToDB(portalStateDB, currentPortalState)
	if err != nil {
		Logger.log.Error(err)
	}

	return nil
}

func buildPortalRelayExchangeRateAction(
	incAddressStr string,
	rates []*metadata.ExchangeRateInfo,
	shardID byte,
) []string {
	data := metadata.PortalExchangeRates{
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PortalExchangeRatesMeta,
		},
		SenderAddress: incAddressStr,
		Rates:         rates,
	}

	actionContent := metadata.PortalExchangeRatesAction{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(metadata.PortalExchangeRatesMeta), actionContentBase64Str}
}

func buildPortalCustodianDepositAction(
	incAddressStr string,
	remoteAddress map[string]string,
	depositAmount uint64,
	shardID byte,
) []string {
	data := metadata.PortalCustodianDeposit{
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PortalCustodianDepositMeta,
		},
		IncogAddressStr: incAddressStr,
		RemoteAddresses: remoteAddress,
		DepositedAmount: depositAmount,
	}

	actionContent := metadata.PortalCustodianDepositAction{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(metadata.PortalCustodianDepositMeta), actionContentBase64Str}
}

func buildPortalUserRegisterAction(
	portingID string,
	incAddressStr string,
	pTokenID string,
	portingAmount uint64,
	portingFee uint64,
	shardID byte,
) []string {
	data := metadata.PortalUserRegister{
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PortalUserRegisterMeta,
		},
		UniqueRegisterId: portingID,
		IncogAddressStr:  incAddressStr,
		PTokenId:         pTokenID,
		RegisterAmount:   portingAmount,
		PortingFee:       portingFee,
	}

	actionContent := metadata.PortalUserRegisterAction{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(metadata.PortalUserRegisterMeta), actionContentBase64Str}
}

func buildPortalUserReqPTokenAction(
	portingID string,
	incAddressStr string,
	pTokenID string,
	portingAmount uint64,
	portingProof string,
	shardID byte,
) []string {
	data := metadata.PortalRequestPTokens{
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PortalUserRequestPTokenMeta,
		},
		UniquePortingID: portingID,
		TokenID:         pTokenID,
		IncogAddressStr: incAddressStr,
		PortingAmount:   portingAmount,
		PortingProof:    portingProof,
	}

	actionContent := metadata.PortalRequestPTokensAction{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(metadata.PortalUserRequestPTokenMeta), actionContentBase64Str}
}

func buildPortalTopupCustodianAction(
	incAddressStr string,
	ptokenID string,
	depositAmount uint64,
	shardID byte,
	freeCollateralAmount uint64,
) []string {
	data := metadata.PortalLiquidationCustodianDepositV2{
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PortalLiquidationCustodianDepositMetaV2,
		},
		IncogAddressStr:      incAddressStr,
		PTokenId:             ptokenID,
		DepositedAmount:      depositAmount,
		FreeCollateralAmount: freeCollateralAmount,
	}

	actionContent := metadata.PortalLiquidationCustodianDepositActionV2{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(metadata.PortalLiquidationCustodianDepositMetaV2), actionContentBase64Str}
}

func buildTopupWaitingPortingAction(
	incAddressStr string,
	portingID string,
	ptokenID string,
	depositAmount uint64,
	shardID byte,
	freeCollateralAmount uint64,
) []string {
	data := metadata.PortalTopUpWaitingPortingRequest{
		MetadataBase: metadata.MetadataBase{
			Type: metadata.PortalTopUpWaitingPortingRequestMeta,
		},
		IncogAddressStr:      incAddressStr,
		PortingID:            portingID,
		PTokenID:             ptokenID,
		DepositedAmount:      depositAmount,
		FreeCollateralAmount: freeCollateralAmount,
	}

	actionContent := metadata.PortalTopUpWaitingPortingRequestAction{
		Meta:    data,
		TxReqID: common.Hash{},
		ShardID: shardID,
	}
	actionContentBytes, _ := json.Marshal(actionContent)
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	return []string{strconv.Itoa(metadata.PortalTopUpWaitingPortingRequestMeta), actionContentBase64Str}
}

/*
	Feature 0: Relay exchange rate
*/
type TestCaseRelayExchangeRate struct {
	senderAddressStr string
	rates            []*metadata.ExchangeRateInfo
}

func buildPortalExchangeRateActionsFromTcs(tcs []TestCaseRelayExchangeRate, shardID byte) [][]string {
	insts := [][]string{}

	for _, tc := range tcs {
		inst := buildPortalRelayExchangeRateAction(tc.senderAddressStr, tc.rates, shardID)
		insts = append(insts, inst)
	}

	return insts
}

func (s *PortalTestSuite) TestRelayExchangeRate() {
	fmt.Println("Running TestRelayExchangeRate - beacon height 999 ...")
	bc := s.blockChain
	beaconHeight := uint64(999)
	shardID := byte(0)
	newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	// build test cases
	testcases := []TestCaseRelayExchangeRate{
		// valid
		{
			senderAddressStr: "feeder1",
			rates: []*metadata.ExchangeRateInfo{
				{
					PTokenID: common.PRVIDStr,
					Rate:     1000000,
				},
				{
					PTokenID: common.PortalBNBIDStr,
					Rate:     20000000,
				},
				{
					PTokenID: common.PortalBTCIDStr,
					Rate:     10000000000,
				},
			},
		},
	}

	// build actions from testcases
	insts := buildPortalExchangeRateActionsFromTcs(testcases, shardID)

	// producer instructions
	newInsts, err := producerPortalInstructions(
		bc, beaconHeight, insts, s.sdb, &s.currentPortalStateForProducer, s.portalParams, shardID, newMatchedRedeemReqIDs)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(1, len(newInsts))
	s.Equal(nil, err)

	//exchangeRateKey := statedb.GeneratePortalFinalExchangeRatesStateObjectKey().String()

	finalExchangeRate := statedb.NewFinalExchangeRatesStateWithValue(
		map[string]statedb.FinalExchangeRatesDetail{
			common.PRVIDStr:       {Amount: 1000000},
			common.PortalBNBIDStr: {Amount: 20000000},
			common.PortalBTCIDStr: {Amount: 10000000000},
		})

	s.Equal(finalExchangeRate, s.currentPortalStateForProcess.FinalExchangeRatesState)
}

/*
	Feature 1: Custodians deposit collateral (PRV)
*/

type TestCaseCustodianDeposit struct {
	custodianIncAddress string
	remoteAddress       map[string]string
	depositAmount       uint64
}

func buildCustodianDepositActionsFromTcs(tcs []TestCaseCustodianDeposit, shardID byte) [][]string {
	insts := [][]string{}

	for _, tc := range tcs {
		inst := buildPortalCustodianDepositAction(tc.custodianIncAddress, tc.remoteAddress, tc.depositAmount, shardID)
		insts = append(insts, inst)
	}

	return insts
}

func (s *PortalTestSuite) TestCustodianDepositCollateral() {
	fmt.Println("Running TestCustodianDepositCollateral - beacon height 1000 ...")
	bc := s.blockChain
	beaconHeight := uint64(1000)
	shardID := byte(0)
	newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	// build test cases
	testcases := []TestCaseCustodianDeposit{
		// valid
		{
			custodianIncAddress: "custodianIncAddress1",
			remoteAddress: map[string]string{
				common.PortalBNBIDStr: "bnbAddress1",
				common.PortalBTCIDStr: "btcAddress1",
			},
			depositAmount: 5000 * 1e9,
		},
		// custodian deposit more with new remote addresses
		// expect don't change to new remote addresses,
		// custodian is able to update new remote addresses when total collaterals is empty
		{
			custodianIncAddress: "custodianIncAddress1",
			remoteAddress: map[string]string{
				common.PortalBNBIDStr: "bnbAddress2",
				common.PortalBTCIDStr: "btcAddress2",
			},
			depositAmount: 2000 * 1e9,
		},
		// new custodian supply only bnb address
		{
			custodianIncAddress: "custodianIncAddress2",
			remoteAddress: map[string]string{
				common.PortalBNBIDStr: "bnbAddress2",
			},
			depositAmount: 1000 * 1e9,
		},
		// new custodian supply only btc address
		{
			custodianIncAddress: "custodianIncAddress3",
			remoteAddress: map[string]string{
				common.PortalBTCIDStr: "btcAddress3",
			},
			depositAmount: 10000 * 1e9,
		},
	}

	// build actions from testcases
	insts := buildCustodianDepositActionsFromTcs(testcases, shardID)

	// producer instructions
	newInsts, err := producerPortalInstructions(
		bc, beaconHeight, insts, s.sdb, &s.currentPortalStateForProducer, s.portalParams, shardID, newMatchedRedeemReqIDs)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(4, len(newInsts))
	s.Equal(nil, err)

	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 7000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		}, nil)

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1000*1e9, 1000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		}, nil)

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 10000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		}, nil)

	s.Equal(3, len(s.currentPortalStateForProducer.CustodianPoolState))
	s.Equal(custodian1, s.currentPortalStateForProducer.CustodianPoolState[custodianKey1])
	s.Equal(custodian2, s.currentPortalStateForProducer.CustodianPoolState[custodianKey2])
	s.Equal(custodian3, s.currentPortalStateForProducer.CustodianPoolState[custodianKey3])

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/*
	Feature 2: Users create porting request
*/

type TestCaseRequestPorting struct {
	portingID     string
	incAddressStr string
	pTokenID      string
	portingAmount uint64
	portingFee    uint64
}

func buildRequestPortingActionsFromTcs(tcs []TestCaseRequestPorting, shardID byte) [][]string {
	insts := [][]string{}

	for _, tc := range tcs {
		inst := buildPortalUserRegisterAction(
			tc.portingID, tc.incAddressStr, tc.pTokenID, tc.portingAmount, tc.portingFee, shardID)
		insts = append(insts, inst)
	}

	return insts
}

func (s *PortalTestSuite) SetupTestPortingRequest() {
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 7000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		}, map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1000*1e9, 1000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		}, map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 10000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		}, map[string]uint64{})

	custodians := map[string]*statedb.CustodianState{
		custodianKey1: custodian1,
		custodianKey2: custodian2,
		custodianKey3: custodian3,
	}

	s.currentPortalStateForProducer.CustodianPoolState = custodians
	s.currentPortalStateForProcess.CustodianPoolState = cloneCustodians(custodians)
}

func (s *PortalTestSuite) TestPortingRequest() {
	fmt.Println("Running TestPortingRequest - beacon height 1001 ...")
	bc := s.blockChain
	beaconHeight := uint64(1001)
	shardID := byte(0)
	newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	s.SetupTestPortingRequest()

	// build test cases
	testcases := []TestCaseRequestPorting{
		// valid porting request with 0.01% porting fee
		{
			portingID:     "porting-bnb-1",
			incAddressStr: "userIncAddress1",
			pTokenID:      common.PortalBNBIDStr,
			portingAmount: 1 * 1e9,
			portingFee:    2000000,
		},
		// invalid porting request with duplicate porting ID
		{
			portingID:     "porting-bnb-1",
			incAddressStr: "userIncAddress2",
			pTokenID:      common.PortalBNBIDStr,
			portingAmount: 1 * 1e9,
			portingFee:    2000000,
		},
		// invalid porting request with invalid porting fee
		{
			portingID:     "porting-bnb-2",
			incAddressStr: "userIncAddress2",
			pTokenID:      common.PortalBNBIDStr,
			portingAmount: 1 * 1e9,
			portingFee:    1999999,
		},
		// valid porting request with >0.01% porting fee
		{
			portingID:     "porting-btc-2",
			incAddressStr: "userIncAddress2",
			pTokenID:      common.PortalBTCIDStr,
			portingAmount: 0.1 * 1e9,
			portingFee:    100000001,
		},
		// invalid porting request: custodians' collateral not enough for the porting amount
		{
			portingID:     "porting-btc-3",
			incAddressStr: "userIncAddress3",
			pTokenID:      common.PortalBTCIDStr,
			portingAmount: 1 * 1e9,
			portingFee:    1000000000,
		},
	}

	// build actions from testcases
	insts := buildRequestPortingActionsFromTcs(testcases, shardID)

	// producer instructions
	newInsts, err := producerPortalInstructions(
		bc, beaconHeight-1, insts, s.sdb, &s.currentPortalStateForProducer, s.portalParams, shardID, newMatchedRedeemReqIDs)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(5, len(newInsts))
	s.Equal(nil, err)

	wPortingReqKey1 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-bnb-1").String()
	wPortingRequest1 := statedb.NewWaitingPortingRequestWithValue(
		"porting-bnb-1", common.Hash{}, common.PortalBNBIDStr,
		"userIncAddress1", 1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress1",
				RemoteAddress:          "bnbAddress1",
				Amount:                 1 * 1e9,
				LockedAmountCollateral: 40000000000,
			},
		}, 2000000, beaconHeight)

	wPortingReqKey2 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-btc-2").String()
	wPortingRequest2 := statedb.NewWaitingPortingRequestWithValue(
		"porting-btc-2", common.Hash{}, common.PortalBTCIDStr,
		"userIncAddress2", 0.1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress3",
				RemoteAddress:          "btcAddress3",
				Amount:                 0.1 * 1e9,
				LockedAmountCollateral: 2000000000000,
			},
		}, 100000001, beaconHeight)

	s.Equal(2, len(s.currentPortalStateForProducer.WaitingPortingRequests))
	s.Equal(wPortingRequest1, s.currentPortalStateForProducer.WaitingPortingRequests[wPortingReqKey1])
	s.Equal(wPortingRequest2, s.currentPortalStateForProducer.WaitingPortingRequests[wPortingReqKey2])

	// custodian state after matching porting requests
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 6960000000000,
		nil,
		map[string]uint64{
			common.PortalBNBIDStr: 40000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		}, map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1000*1e9, 1000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		}, map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 8000000000000,
		nil,
		map[string]uint64{
			common.PortalBTCIDStr: 2000000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		}, map[string]uint64{})

	s.Equal(custodian1, s.currentPortalStateForProducer.CustodianPoolState[custodianKey1])
	s.Equal(custodian2, s.currentPortalStateForProducer.CustodianPoolState[custodianKey2])
	s.Equal(custodian3, s.currentPortalStateForProducer.CustodianPoolState[custodianKey3])

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/*
	Feature 3: Users submit proof to request pTokens after sending public tokens to custodians
*/
type TestCaseRequestPtokens struct {
	portingID     string
	incAddressStr string
	pTokenID      string
	portingAmount uint64

	blockHeight  int64
	transferInfo map[string]int64
	portingProof string
	rootHash     []byte
}

func buildRequestPtokensActionsFromTcs(tcs []TestCaseRequestPtokens, shardID byte) [][]string {
	insts := [][]string{}

	for _, tc := range tcs {
		tc.portingProof, tc.rootHash = buildBNBProof(tc.blockHeight, "", tc.transferInfo, tc.portingID, "")

		inst := buildPortalUserReqPTokenAction(
			tc.portingID, tc.incAddressStr, tc.pTokenID, tc.portingAmount, tc.portingProof, shardID)
		insts = append(insts, inst)
	}

	return insts
}

func (s *PortalTestSuite) SetupTestRequestPtokens() {
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 6760000000000,
		nil,
		map[string]uint64{
			common.PortalBNBIDStr: 240000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		}, map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1000*1e9, 1000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		}, map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 3000000000000,
		nil,
		map[string]uint64{
			common.PortalBTCIDStr: 7000000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		}, map[string]uint64{})

	custodians := map[string]*statedb.CustodianState{
		custodianKey1: custodian1,
		custodianKey2: custodian2,
		custodianKey3: custodian3,
	}

	wPortingReqKey1 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-bnb-1").String()
	wPortingRequest1 := statedb.NewWaitingPortingRequestWithValue(
		"porting-bnb-1", common.Hash{}, common.PortalBNBIDStr,
		USER1_INC_ADDRESS, 1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress1",
				RemoteAddress:          "bnbAddress1",
				Amount:                 1 * 1e9,
				LockedAmountCollateral: 40000000000,
			},
		}, 2000000, 1000)

	wPortingReqKey2 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-btc-2").String()
	wPortingRequest2 := statedb.NewWaitingPortingRequestWithValue(
		"porting-btc-2", common.Hash{}, common.PortalBTCIDStr,
		USER2_INC_ADDRESS, 0.1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress3",
				RemoteAddress:          "btcAddress3",
				Amount:                 0.1 * 1e9,
				LockedAmountCollateral: 2000000000000,
			},
		}, 100000001, 1000)

	wPortingReqKey3 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-bnb-3").String()
	wPortingRequest3 := statedb.NewWaitingPortingRequestWithValue(
		"porting-bnb-3", common.Hash{}, common.PortalBNBIDStr,
		USER2_INC_ADDRESS, 5*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress1",
				RemoteAddress:          "bnbAddress1",
				Amount:                 5 * 1e9,
				LockedAmountCollateral: 200000000000,
			},
		}, 2000000, 1000)

	wPortingReqKey4 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-btc-4").String()
	wPortingRequest4 := statedb.NewWaitingPortingRequestWithValue(
		"porting-btc-4", common.Hash{}, common.PortalBTCIDStr,
		USER1_INC_ADDRESS, 0.25*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress3",
				RemoteAddress:          "btcAddress3",
				Amount:                 0.25 * 1e9,
				LockedAmountCollateral: 5000000000000,
			},
		}, 250000000, 1020)
	wPortingRequests := map[string]*statedb.WaitingPortingRequest{
		wPortingReqKey1: wPortingRequest1,
		wPortingReqKey2: wPortingRequest2,
		wPortingReqKey3: wPortingRequest3,
		wPortingReqKey4: wPortingRequest4,
	}

	s.currentPortalStateForProducer.CustodianPoolState = custodians
	s.currentPortalStateForProducer.WaitingPortingRequests = wPortingRequests

	s.currentPortalStateForProcess.CustodianPoolState = cloneCustodians(custodians)
	s.currentPortalStateForProcess.WaitingPortingRequests = cloneWPortingRequests(wPortingRequests)
}

func (s *PortalTestSuite) TestRequestPtokens() {
	fmt.Println("Running TestRequestPtokens - beacon height 1002 ...")
	bc := s.blockChain
	beaconHeight := uint64(1002)
	shardID := byte(0)
	newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	s.SetupTestRequestPtokens()

	// build test cases
	testcases := []TestCaseRequestPtokens{
		// valid request ptokens
		{
			portingID:     "porting-bnb-1",
			incAddressStr: USER1_INC_ADDRESS,
			pTokenID:      common.PortalBNBIDStr,
			portingAmount: 1 * 1e9,
			blockHeight:   1000,
			transferInfo: map[string]int64{
				"bnbAddress1": 1e8,
			},
			portingProof: "",
			rootHash:     nil,
		},
	}

	// build actions from testcases
	insts := buildRequestPtokensActionsFromTcs(testcases, shardID)

	// producer instructions
	newInsts, err := producerPortalInstructions(
		bc, beaconHeight-1, insts, s.sdb, &s.currentPortalStateForProducer, s.portalParams, shardID, newMatchedRedeemReqIDs)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(1, len(newInsts))
	s.Equal(nil, err)
}

/*
	Feature 4: auto-liquidation: the custodians don't send back public token to the users
*/

func (s *PortalTestSuite) SetupTestAutoLiquidation() {
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()
	custodianKey4 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress4").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 6920000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 80000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		},
		map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1000*1e9, 960000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0.6 * 1e9,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 40000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		},
		map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 8000000000000,
		map[string]uint64{
			common.PortalBTCIDStr: 0.1 * 1e9,
		},
		map[string]uint64{
			common.PortalBTCIDStr: 2000000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		},
		map[string]uint64{})

	custodian4 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress4", 5000*1e9, 4960000000000,
		map[string]uint64{
		},
		map[string]uint64{
			common.PortalBNBIDStr: 40000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress4",
		},
		map[string]uint64{})

	custodians := map[string]*statedb.CustodianState{
		custodianKey1: custodian1,
		custodianKey2: custodian2,
		custodianKey3: custodian3,
		custodianKey4: custodian4,
	}

	redeemReqKey1 := statedb.GenerateMatchedRedeemRequestObjectKey("redeem-bnb-1").String()
	redeemRequest1 := statedb.NewRedeemRequestWithValue(
		"redeem-bnb-1", common.PortalBNBIDStr,
		USER1_INC_ADDRESS, "userBNBAddress1", 2.3*1e9,
		[]*statedb.MatchingRedeemCustodianDetail{
			statedb.NewMatchingRedeemCustodianDetailWithValue("custodianIncAddress1", "bnbAddress1", 2*1e9),
			statedb.NewMatchingRedeemCustodianDetailWithValue("custodianIncAddress2", "bnbAddress2", 0.3*1e9),
		}, 4600000, 1000, common.Hash{})

	redeemReqKey2 := statedb.GenerateMatchedRedeemRequestObjectKey("redeem-btc-2").String()
	redeemRequest2 := statedb.NewRedeemRequestWithValue(
		"redeem-btc-2", common.PortalBTCIDStr,
		USER2_INC_ADDRESS, "userBTCAddress2", 0.03*1e9,
		[]*statedb.MatchingRedeemCustodianDetail{
			statedb.NewMatchingRedeemCustodianDetailWithValue("custodianIncAddress3", "btcAddress3", 0.03*1e9),
		}, 30000000, 1500, common.Hash{})

	matchedRedeemRequest := map[string]*statedb.RedeemRequest{
		redeemReqKey1: redeemRequest1,
		redeemReqKey2: redeemRequest2,
	}

	wRedeemReqKey3 := statedb.GenerateWaitingRedeemRequestObjectKey("redeem-bnb-3").String()
	wRedeemRequest3 := statedb.NewRedeemRequestWithValue(
		"redeem-bnb-3", common.PortalBNBIDStr,
		USER1_INC_ADDRESS, "userBNBAddress1", 0.1*1e9,
		[]*statedb.MatchingRedeemCustodianDetail{
			statedb.NewMatchingRedeemCustodianDetailWithValue("custodianIncAddress2", "bnbAddress2", 0.1*1e9),
		}, 4600000, 1500, common.Hash{})

	wRedeemRequests := map[string]*statedb.RedeemRequest{
		wRedeemReqKey3: wRedeemRequest3,
	}

	wPortingReqKey1 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-bnb-1").String()
	wPortingRequest1 := statedb.NewWaitingPortingRequestWithValue(
		"porting-bnb-1", common.Hash{}, common.PortalBNBIDStr,
		"userIncAddress1", 1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress4",
				RemoteAddress:          "bnbAddress4",
				Amount:                 1 * 1e9,
				LockedAmountCollateral: 40000000000,
			},
		}, 2000000, 1500)

	wPortingRequests := map[string]*statedb.WaitingPortingRequest{
		wPortingReqKey1: wPortingRequest1,
	}

	s.currentPortalStateForProducer.CustodianPoolState = custodians
	s.currentPortalStateForProducer.MatchedRedeemRequests = matchedRedeemRequest
	s.currentPortalStateForProducer.WaitingRedeemRequests = wRedeemRequests
	s.currentPortalStateForProducer.WaitingPortingRequests = wPortingRequests

	s.currentPortalStateForProcess.CustodianPoolState = cloneCustodians(custodians)
	s.currentPortalStateForProcess.MatchedRedeemRequests = cloneRedeemRequests(matchedRedeemRequest)
	s.currentPortalStateForProcess.WaitingRedeemRequests = cloneRedeemRequests(wRedeemRequests)
	s.currentPortalStateForProcess.WaitingPortingRequests = cloneWPortingRequests(wPortingRequests)
}

func (s *PortalTestSuite) TestAutoLiquidationCustodian() {
	fmt.Println("Running TestAutoLiquidationCustodian - beacon height 3161 ...")
	bc := s.blockChain
	beaconHeight := uint64(3161) // ~ after 24 hours from redeem request
	//shardID := byte(0)
	//newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	s.SetupTestAutoLiquidation()

	// producer instructions
	newInsts, err := bc.checkAndBuildInstForCustodianLiquidation(
		beaconHeight-1, &s.currentPortalStateForProducer, s.portalParams)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(2, len(newInsts))
	s.Equal(nil, err)

	//// remain waiting porting request
	//redeemReqKey2 := statedb.GenerateMatchedRedeemRequestObjectKey("redeem-btc-2").String()
	s.Equal(1, len(s.currentPortalStateForProducer.MatchedRedeemRequests))
	s.Equal(1, len(s.currentPortalStateForProducer.WaitingRedeemRequests))
	//s.Equal(1, len(s.currentPortalStateForProducer.WaitingPortingRequests))

	//custodian state after auto liquidation
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 6952000000000, 6952000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		},
		map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 992800000000, 964800000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0.6 * 1e9,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 28000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		},
		map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 8000000000000,
		map[string]uint64{
			common.PortalBTCIDStr: 0.1 * 1e9,
		},
		map[string]uint64{
			common.PortalBTCIDStr: 2000000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		},
		map[string]uint64{})

	s.Equal(custodian1, s.currentPortalStateForProducer.CustodianPoolState[custodianKey1])
	s.Equal(custodian2, s.currentPortalStateForProducer.CustodianPoolState[custodianKey2])
	s.Equal(custodian3, s.currentPortalStateForProducer.CustodianPoolState[custodianKey3])

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/*
	Feature 5: auto-liquidation: the proportion between the collateral and public token is drop down below 120%
*/

func (s *PortalTestSuite) SetupTestAutoLiquidationByExchangeRate() {
	s.SetupTestAutoLiquidation()
	finalExchangeRate := statedb.NewFinalExchangeRatesStateWithValue(
		map[string]statedb.FinalExchangeRatesDetail{
			common.PRVIDStr:       {Amount: 1000000},
			common.PortalBNBIDStr: {Amount: 40000000},
			common.PortalBTCIDStr: {Amount: 10000000000},
		})

	s.currentPortalStateForProducer.FinalExchangeRatesState = finalExchangeRate
	s.currentPortalStateForProcess.FinalExchangeRatesState = finalExchangeRate
}

func (s *PortalTestSuite) TestAutoLiquidationByExchangeRate() {
	fmt.Println("Running TestAutoLiquidationCustodian - beacon height 1501 ...")
	bc := s.blockChain
	beaconHeight := uint64(1501)
	//shardID := byte(0)
	//newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	s.SetupTestAutoLiquidationByExchangeRate()

	// producer instructions
	newInsts, err := buildInstForLiquidationTopPercentileExchangeRates(
		beaconHeight-1, &s.currentPortalStateForProducer, s.portalParams)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(2, len(newInsts))
	s.Equal(nil, err)

	// remain waiting redeem requests and matched redeem requests
	s.Equal(2, len(s.currentPortalStateForProducer.MatchedRedeemRequests))
	s.Equal(0, len(s.currentPortalStateForProducer.WaitingRedeemRequests))
	s.Equal(1, len(s.currentPortalStateForProducer.WaitingPortingRequests))

	//custodian state after auto liquidation
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()
	custodianKey4 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress4").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 6920000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 80000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		},
		map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 972000000000, 960000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 12000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		},
		map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 8000000000000,
		map[string]uint64{
			common.PortalBTCIDStr: 0.1 * 1e9,
		},
		map[string]uint64{
			common.PortalBTCIDStr: 2000000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		},
		map[string]uint64{})

	custodian4 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress4", 5000*1e9, 4960000000000,
		map[string]uint64{
		},
		map[string]uint64{
			common.PortalBNBIDStr: 40000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress4",
		},
		map[string]uint64{})

	rates := map[string]statedb.LiquidationPoolDetail{
		common.PortalBNBIDStr: {
			CollateralAmount: 28000000000,
			PubTokenAmount:   0.7 * 1e9,
		},
	}
	liquidationPool := statedb.NewLiquidationPoolWithValue(rates)
	liquidationPoolKey := statedb.GeneratePortalLiquidationPoolObjectKey().String()

	s.Equal(custodian1, s.currentPortalStateForProducer.CustodianPoolState[custodianKey1])
	s.Equal(custodian2, s.currentPortalStateForProducer.CustodianPoolState[custodianKey2])
	s.Equal(custodian3, s.currentPortalStateForProducer.CustodianPoolState[custodianKey3])
	s.Equal(custodian4, s.currentPortalStateForProducer.CustodianPoolState[custodianKey4])
	s.Equal(liquidationPool, s.currentPortalStateForProducer.LiquidationPool[liquidationPoolKey])

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/*
	Feature 6: auto-liquidation: the custodian top up the collaterals
*/

func (s *PortalTestSuite) SetupTestTopupCustodian() {
	s.SetupTestAutoLiquidationByExchangeRate()
}

type TestCaseTopupCustodian struct {
	incAddressStr        string
	ptokenID             string
	depositAmount        uint64
	freeCollateralAmount uint64
}

func buildTopupCustodianActionsFromTcs(tcs []TestCaseTopupCustodian, shardID byte) [][]string {
	insts := [][]string{}

	for _, tc := range tcs {
		inst := buildPortalTopupCustodianAction(tc.incAddressStr, tc.ptokenID, tc.depositAmount, shardID, tc.freeCollateralAmount)
		insts = append(insts, inst)
	}

	return insts
}

type TestCaseTopupWaitingPorting struct {
	incAddressStr        string
	portingID            string
	ptokenID             string
	depositAmount        uint64
	freeCollateralAmount uint64
}

func buildTopupWaitingPortingActionsFromTcs(tcs []TestCaseTopupWaitingPorting, shardID byte) [][]string {
	insts := [][]string{}

	for _, tc := range tcs {
		inst := buildTopupWaitingPortingAction(tc.incAddressStr, tc.portingID, tc.ptokenID, tc.depositAmount, shardID, tc.freeCollateralAmount)
		insts = append(insts, inst)
	}

	return insts
}
func (s *PortalTestSuite) TestTopupCustodian() {
	fmt.Println("Running TestAutoLiquidationCustodian - beacon height 1501 ...")
	bc := s.blockChain
	beaconHeight := uint64(1501)
	shardID := byte(0)
	newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	s.SetupTestAutoLiquidationByExchangeRate()

	// build test cases for topup custodian
	testcases := []TestCaseTopupCustodian{
		// topup by burning more collaterals
		{
			incAddressStr:        "custodianIncAddress2",
			ptokenID:             common.PortalBNBIDStr,
			depositAmount:        500 * 1e9,
			freeCollateralAmount: 0,
		},
		// topup by using free collaterals
		{
			incAddressStr:        "custodianIncAddress2",
			ptokenID:             common.PortalBNBIDStr,
			depositAmount:        0,
			freeCollateralAmount: 500 * 1e9,
		},
	}

	// build actions from testcases
	insts := buildTopupCustodianActionsFromTcs(testcases, shardID)

	// build test cases for topup waiting porting
	testcases2 := []TestCaseTopupWaitingPorting{
		// topup by burning more collaterals
		{
			incAddressStr:        "custodianIncAddress4",
			portingID:            "porting-bnb-1",
			ptokenID:             common.PortalBNBIDStr,
			depositAmount:        20 * 1e9,
			freeCollateralAmount: 0,
		},
		// topup by using free collaterals
		{
			incAddressStr:        "custodianIncAddress4",
			portingID:            "porting-bnb-1",
			ptokenID:             common.PortalBNBIDStr,
			depositAmount:        0,
			freeCollateralAmount: 50 * 1e9,
		},
	}

	// build actions from testcases2
	insts2 := buildTopupWaitingPortingActionsFromTcs(testcases2, shardID)

	insts = append(insts, insts2...)

	// producer instructions
	newInsts, err := producerPortalInstructions(
		bc, beaconHeight, insts, s.sdb, &s.currentPortalStateForProducer, s.portalParams, shardID, newMatchedRedeemReqIDs)

	// check liquidation by exchange rates
	newInstsForLiquidationByExchangeRate, err := buildInstForLiquidationTopPercentileExchangeRates(
		beaconHeight-1, &s.currentPortalStateForProducer, s.portalParams)

	s.Equal(0, len(newInstsForLiquidationByExchangeRate))

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(4, len(newInsts))
	s.Equal(nil, err)

	// remain waiting redeem requests and matched redeem requests
	s.Equal(2, len(s.currentPortalStateForProducer.MatchedRedeemRequests))
	s.Equal(1, len(s.currentPortalStateForProducer.WaitingRedeemRequests))
	s.Equal(1, len(s.currentPortalStateForProducer.WaitingPortingRequests))

	//custodian state after auto liquidation
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()
	custodianKey4 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress4").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 6920000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 80000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		},
		map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1500*1e9, 460000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0.6 * 1e9,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 1040000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		},
		map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 8000000000000,
		map[string]uint64{
			common.PortalBTCIDStr: 0.1 * 1e9,
		},
		map[string]uint64{
			common.PortalBTCIDStr: 2000000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		},
		map[string]uint64{})

	custodian4 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress4", 5020*1e9, 4910000000000,
		map[string]uint64{
		},
		map[string]uint64{
			common.PortalBNBIDStr: 110000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress4",
		},
		map[string]uint64{})

	wPortingReqKey1 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-bnb-1").String()
	wPortingRequest1 := statedb.NewWaitingPortingRequestWithValue(
		"porting-bnb-1", common.Hash{}, common.PortalBNBIDStr,
		"userIncAddress1", 1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress4",
				RemoteAddress:          "bnbAddress4",
				Amount:                 1 * 1e9,
				LockedAmountCollateral: 110000000000,
			},
		}, 2000000, 1500)

	wPortingRequests := map[string]*statedb.WaitingPortingRequest{
		wPortingReqKey1: wPortingRequest1,
	}

	s.Equal(custodian1, s.currentPortalStateForProducer.CustodianPoolState[custodianKey1])
	s.Equal(custodian2, s.currentPortalStateForProducer.CustodianPoolState[custodianKey2])
	s.Equal(custodian3, s.currentPortalStateForProducer.CustodianPoolState[custodianKey3])
	s.Equal(custodian4, s.currentPortalStateForProducer.CustodianPoolState[custodianKey4])
	s.Equal(0, len(s.currentPortalStateForProducer.LiquidationPool))
	s.Equal(wPortingRequests, s.currentPortalStateForProducer.WaitingPortingRequests)

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/**
	Feature 7: Porting request timeout - after 21 hours
 **/

func (s *PortalTestSuite) SetupTestPortingRequestExpired() {
	s.SetupTestRequestPtokens()
}

func (s *PortalTestSuite) TestPortingRequestExpired() {
	fmt.Println("Running TestPortingRequestExpired - beacon height 3161 ...")
	bc := s.blockChain
	beaconHeight := uint64(3161) // after 24 hours from requesting porting (bch = 100)
	//shardID := byte(0)
	//newMatchedRedeemReqIDs := []string{}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	s.SetupTestPortingRequestExpired()

	// producer instructions
	newInsts, err := bc.checkAndBuildInstForExpiredWaitingPortingRequest(
		beaconHeight-1, &s.currentPortalStateForProducer, s.portalParams)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(3, len(newInsts))
	s.Equal(nil, err)

	// remain waiting redeem requests and matched redeem requests
	s.Equal(1, len(s.currentPortalStateForProducer.WaitingPortingRequests))

	//custodian state after auto liquidation
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000*1e9, 7000*1e9,
		nil,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		}, map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1000*1e9, 1000*1e9,
		nil, nil,
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		}, map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000*1e9, 5000000000000,
		nil,
		map[string]uint64{
			common.PortalBTCIDStr: 5000000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		}, map[string]uint64{})

	s.Equal(custodian1, s.currentPortalStateForProducer.CustodianPoolState[custodianKey1])
	s.Equal(custodian2, s.currentPortalStateForProducer.CustodianPoolState[custodianKey2])
	s.Equal(custodian3, s.currentPortalStateForProducer.CustodianPoolState[custodianKey3])

	s.Equal(s.currentPortalStateForProcess, s.currentPortalStateForProducer)
}

/**
	Feature 8: Custodian rewards from DAO funds and porting/redeem fee
 **/

func (s *PortalTestSuite) SetupTestCustodianRewards() {
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	custodian1 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress1", 7000000000000, 6708000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 292000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress1",
			common.PortalBTCIDStr: "btcAddress1",
		}, map[string]uint64{})

	custodian2 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress2", 1000000000000, 972000000000,
		map[string]uint64{
			common.PortalBNBIDStr: 0,
		},
		map[string]uint64{
			common.PortalBNBIDStr: 28000000000,
		},
		map[string]string{
			common.PortalBNBIDStr: "bnbAddress2",
		}, map[string]uint64{})

	custodian3 := statedb.NewCustodianStateWithValue(
		"custodianIncAddress3", 10000000000000, 7988000000000,
		nil,
		map[string]uint64{
			common.PortalBTCIDStr: 2012000000000,
		},
		map[string]string{
			common.PortalBTCIDStr: "btcAddress3",
		}, map[string]uint64{})

	custodians := map[string]*statedb.CustodianState{
		custodianKey1: custodian1,
		custodianKey2: custodian2,
		custodianKey3: custodian3,
	}

	// waiting porting requests
	wPortingReqKey1 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-bnb-1").String()
	wPortingRequest1 := statedb.NewWaitingPortingRequestWithValue(
		"porting-bnb-1", common.Hash{}, common.PortalBNBIDStr,
		USER1_INC_ADDRESS, 1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress1",
				RemoteAddress:          "bnbAddress1",
				Amount:                 0.3 * 1e9,
				LockedAmountCollateral: 12000000000,
			},
			{
				IncAddress:             "custodianIncAddress2",
				RemoteAddress:          "bnbAddress1",
				Amount:                 0.7 * 1e9,
				LockedAmountCollateral: 28000000000,
			},
		}, 2000000, 1000)

	wPortingReqKey2 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-btc-2").String()
	wPortingRequest2 := statedb.NewWaitingPortingRequestWithValue(
		"porting-btc-2", common.Hash{}, common.PortalBTCIDStr,
		USER2_INC_ADDRESS, 0.1*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress3",
				RemoteAddress:          "btcAddress3",
				Amount:                 0.1 * 1e9,
				LockedAmountCollateral: 2000000000000,
			},
		}, 100000001, 1000)

	wPortingReqKey3 := statedb.GeneratePortalWaitingPortingRequestObjectKey("porting-bnb-3").String()
	wPortingRequest3 := statedb.NewWaitingPortingRequestWithValue(
		"porting-bnb-3", common.Hash{}, common.PortalBNBIDStr,
		USER2_INC_ADDRESS, 5*1e9,
		[]*statedb.MatchingPortingCustodianDetail{
			{
				IncAddress:             "custodianIncAddress1",
				RemoteAddress:          "bnbAddress1",
				Amount:                 5 * 1e9,
				LockedAmountCollateral: 200000000000,
			},
		}, 2000000, 900)

	wPortingRequests := map[string]*statedb.WaitingPortingRequest{
		wPortingReqKey1: wPortingRequest1,
		wPortingReqKey2: wPortingRequest2,
		wPortingReqKey3: wPortingRequest3,
	}

	// matched redeem requests
	redeemReqKey1 := statedb.GenerateMatchedRedeemRequestObjectKey("redeem-bnb-1").String()
	redeemRequest1 := statedb.NewRedeemRequestWithValue(
		"redeem-bnb-1", common.PortalBNBIDStr,
		USER1_INC_ADDRESS, "userBNBAddress1", 2.3*1e9,
		[]*statedb.MatchingRedeemCustodianDetail{
			statedb.NewMatchingRedeemCustodianDetailWithValue("custodianIncAddress1", "bnbAddress1", 2*1e9),
			statedb.NewMatchingRedeemCustodianDetailWithValue("custodianIncAddress2", "bnbAddress2", 0.3*1e9),
		}, 4600000, 990, common.Hash{})

	matchedRedeemRequest := map[string]*statedb.RedeemRequest{
		redeemReqKey1: redeemRequest1,
	}

	// locked collaterals
	lockedCollateralDetail := map[string]uint64{
		"custodianIncAddress1": 292000000000,
		"custodianIncAddress2": 28000000000,
		"custodianIncAddress3": 2012000000000,
	}
	totalLockedCollateralInEpoch := uint64(2332000000000)
	s.currentPortalStateForProducer.LockedCollateralForRewards = statedb.NewLockedCollateralStateWithValue(
		totalLockedCollateralInEpoch, lockedCollateralDetail)
	s.currentPortalStateForProducer.CustodianPoolState = custodians
	s.currentPortalStateForProducer.WaitingPortingRequests = wPortingRequests
	s.currentPortalStateForProducer.MatchedRedeemRequests = matchedRedeemRequest

	s.currentPortalStateForProcess.CustodianPoolState = cloneCustodians(custodians)
	s.currentPortalStateForProcess.WaitingPortingRequests = cloneWPortingRequests(wPortingRequests)
	s.currentPortalStateForProcess.MatchedRedeemRequests = cloneRedeemRequests(matchedRedeemRequest)
	s.currentPortalStateForProcess.LockedCollateralForRewards = statedb.NewLockedCollateralStateWithValue(
		totalLockedCollateralInEpoch, cloneMap(lockedCollateralDetail))
}

func (s *PortalTestSuite) TestCustodianRewards() {
	fmt.Println("Running TestCustodianRewards - beacon height 1000 ...")
	bc := s.blockChain
	beaconHeight := uint64(1000) // after 24 hours from requesting porting (bch = 100)
	//shardID := byte(0)
	newMatchedRedeemReqIDs := []string{"redeem-bnb-1"}
	rewardForCustodianByEpoch := map[common.Hash]uint64{
		common.PRVCoinID: 100000000000, // 100 prv
		common.Hash{1}:   200000,
	}
	updatingInfoByTokenID := map[common.Hash]UpdatingInfo{}

	s.SetupTestCustodianRewards()

	// producer instructions
	newInsts, err := bc.buildPortalRewardsInsts(
		beaconHeight-1, &s.currentPortalStateForProducer, rewardForCustodianByEpoch, newMatchedRedeemReqIDs)
	s.Equal(nil, err)

	// process new instructions
	err = processPortalInstructions(
		bc, beaconHeight-1, newInsts, s.sdb, &s.currentPortalStateForProcess, s.portalParams, updatingInfoByTokenID)

	// check results
	s.Equal(2, len(newInsts))
	s.Equal(nil, err)

	//custodian state after auto liquidation
	custodianKey1 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress1").String()
	custodianKey2 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress2").String()
	custodianKey3 := statedb.GenerateCustodianStateObjectKey("custodianIncAddress3").String()

	reward1 := map[string]uint64{
		common.PRVIDStr:         12526040824,
		common.Hash{1}.String(): 25044,
	}

	reward2 := map[string]uint64{
		common.PRVIDStr:         1202686106,
		common.Hash{1}.String(): 2401,
	}
	reward3 := map[string]uint64{
		common.PRVIDStr:         86377873071,
		common.Hash{1}.String(): 172555,
	}

	s.Equal(reward1, s.currentPortalStateForProducer.CustodianPoolState[custodianKey1].GetRewardAmount())
	s.Equal(reward2, s.currentPortalStateForProducer.CustodianPoolState[custodianKey2].GetRewardAmount())
	s.Equal(reward3, s.currentPortalStateForProducer.CustodianPoolState[custodianKey3].GetRewardAmount())

	s.Equal(reward1, s.currentPortalStateForProcess.CustodianPoolState[custodianKey1].GetRewardAmount())
	s.Equal(reward2, s.currentPortalStateForProcess.CustodianPoolState[custodianKey2].GetRewardAmount())
	s.Equal(reward3, s.currentPortalStateForProcess.CustodianPoolState[custodianKey3].GetRewardAmount())
}

func TestPortalSuite(t *testing.T) {
	suite.Run(t, new(PortalTestSuite))
}
