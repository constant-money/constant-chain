package blockchain

// import (
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"github.com/incognitochain/incognito-chain/common"
// 	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
// 	"github.com/incognitochain/incognito-chain/metadata"
// 	mocks "github.com/incognitochain/incognito-chain/mocks"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/suite"
// 	"strconv"
// 	"testing"
// )

// // Define the suite, and absorb the built-in basic suite
// // functionality from testify - including a T() method which
// // returns the current testing context
// type PortalProducerSuite struct {
// 	suite.Suite
// 	currentPortalState *CurrentPortalState
// }

// func (suite *PortalProducerSuite) SetupTest() {
// 	suite.currentPortalState = &CurrentPortalState{
// 		CustodianPoolState:      map[string]*statedb.CustodianState{},
// 		ExchangeRatesRequests:   map[string]*metadata.ExchangeRatesRequestStatus{},
// 		FinalExchangeRatesState: map[string]*statedb.FinalExchangeRatesState{},
// 		WaitingPortingRequests:  map[string]*statedb.WaitingPortingRequest{},
// 		WaitingRedeemRequests:   map[string]*statedb.RedeemRequest{},
// 		LiquidationPool:         map[string]*statedb.LiquidationPool{},
// 	}
// }

// /************************ Porting request test ************************/
// type PortingRequestExcepted struct {
// 	Metadata    string
// 	ChainStatus string
// 	Custodian1  []string
// 	Custodian2  []string
// }

// type LiquidationExchangeRatesExcepted struct {
// 	TpValue         int
// 	Custodian1      []string
// 	Custodian2      []string
// 	LiquidationPool []uint64
// }

// type PortingRequestTestCase struct {
// 	TestCaseName string
// 	Input        func() metadata.PortalUserRegisterAction
// 	Output       func() PortingRequestExcepted
// }

// type AutoLiquidationExchangeRatesTestCase struct {
// 	TestCaseName string
// 	Input        map[string]uint64
// 	Output       func() LiquidationExchangeRatesExcepted
// }

// func (suite *PortalProducerSuite) SetupExchangeRates(beaconHeight uint64) {
// 	rates := make(map[string]statedb.FinalExchangeRatesDetail)
// 	rates["b832e5d3b1f01a4f0623f7fe91d6673461e1f5d37d91fe78c5c2e6183ff39696"] = statedb.FinalExchangeRatesDetail{
// 		Amount: 8000000000,
// 	}
// 	rates["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] = statedb.FinalExchangeRatesDetail{
// 		Amount: 20000000,
// 	}
// 	rates["0000000000000000000000000000000000000000000000000000000000000004"] = statedb.FinalExchangeRatesDetail{
// 		Amount: 500000,
// 	}

// 	exchangeRates := make(map[string]*statedb.FinalExchangeRatesState)
// 	exchangeRatesKey := statedb.GeneratePortalFinalExchangeRatesStateObjectKey(beaconHeight)
// 	exchangeRates[exchangeRatesKey.String()] = statedb.NewFinalExchangeRatesStateWithValue(rates)

// 	suite.currentPortalState.FinalExchangeRatesState = exchangeRates
// }

// func (suite *PortalProducerSuite) SetupExchangeRatesWithValue(beaconHeight uint64, btc uint64, bnb uint64, prv uint64) {
// 	rates := make(map[string]statedb.FinalExchangeRatesDetail)
// 	rates["b832e5d3b1f01a4f0623f7fe91d6673461e1f5d37d91fe78c5c2e6183ff39696"] = statedb.FinalExchangeRatesDetail{
// 		Amount: btc,
// 	}
// 	rates["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] = statedb.FinalExchangeRatesDetail{
// 		Amount: bnb,
// 	}
// 	rates["0000000000000000000000000000000000000000000000000000000000000004"] = statedb.FinalExchangeRatesDetail{
// 		Amount: prv,
// 	}

// 	exchangeRates := make(map[string]*statedb.FinalExchangeRatesState)
// 	exchangeRatesKey := statedb.GeneratePortalFinalExchangeRatesStateObjectKey(beaconHeight)
// 	exchangeRates[exchangeRatesKey.String()] = statedb.NewFinalExchangeRatesStateWithValue(rates)

// 	suite.currentPortalState.FinalExchangeRatesState = exchangeRates
// }

// func (suite *PortalProducerSuite) SetupOneCustodian(beaconHeight uint64) {
// 	remoteAddresses := make([]statedb.RemoteAddress, 0)
// 	remoteAddresses = append(
// 		remoteAddresses,
// 		*statedb.NewRemoteAddressWithValue("b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b", "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf234"),
// 	)

// 	custodianKey := statedb.GenerateCustodianStateObjectKey(beaconHeight, "12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ")
// 	newCustodian := statedb.NewCustodianStateWithValue(
// 		"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ",
// 		100000,
// 		100000,
// 		nil,
// 		nil,
// 		remoteAddresses,
// 		0,
// 	)

// 	custodian := make(map[string]*statedb.CustodianState)
// 	custodian[custodianKey.String()] = newCustodian
// 	suite.currentPortalState.CustodianPoolState = custodian
// }

// func (suite *PortalProducerSuite) SetupMultipleCustodian(beaconHeight uint64) {
// 	remoteAddresses := make([]statedb.RemoteAddress, 0)
// 	remoteAddresses = append(
// 		remoteAddresses,
// 		*statedb.NewRemoteAddressWithValue("b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b", "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf234"),
// 	)

// 	custodianKey := statedb.GenerateCustodianStateObjectKey(beaconHeight, "12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ")
// 	newCustodian := statedb.NewCustodianStateWithValue(
// 		"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ",
// 		100000,
// 		100000,
// 		nil,
// 		nil,
// 		remoteAddresses,
// 		0,
// 	)

// 	custodianKey2 := statedb.GenerateCustodianStateObjectKey(beaconHeight, "12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy")
// 	newCustodian2 := statedb.NewCustodianStateWithValue(
// 		"12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy",
// 		90000,
// 		90000,
// 		nil,
// 		nil,
// 		remoteAddresses,
// 		0,
// 	)

// 	custodian := make(map[string]*statedb.CustodianState)
// 	custodian[custodianKey.String()] = newCustodian
// 	custodian[custodianKey2.String()] = newCustodian2
// 	suite.currentPortalState.CustodianPoolState = custodian
// }

// func (suite *PortalProducerSuite) SetupMultipleCustodianContainPToken(beaconHeight uint64) {
// 	remoteAddresses := make([]statedb.RemoteAddress, 0)
// 	remoteAddresses = append(
// 		remoteAddresses,
// 		*statedb.NewRemoteAddressWithValue("b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b", "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf234"),
// 	)

// 	exchangeRatesKey := statedb.GeneratePortalFinalExchangeRatesStateObjectKey(beaconHeight)

// 	convertExchangeRatesObj := NewConvertExchangeRatesObject(suite.currentPortalState.FinalExchangeRatesState[exchangeRatesKey.String()])
// 	totalPTokenAfterUp150PercentUnit64 := up150Percent(1000)   //return nano pBTC, pBNB
// 	totalPTokenAfterUp150PercentUnit64_2 := up150Percent(2000) //return nano pBTC, pBNB

// 	totalPRV, _ := convertExchangeRatesObj.ExchangePToken2PRVByTokenId("b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b", totalPTokenAfterUp150PercentUnit64)
// 	totalPRV_2, _ := convertExchangeRatesObj.ExchangePToken2PRVByTokenId("b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b", totalPTokenAfterUp150PercentUnit64_2)

// 	custodianKey := statedb.GenerateCustodianStateObjectKey(beaconHeight, "12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ")
// 	newCustodian := statedb.NewCustodianStateWithValue(
// 		"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ",
// 		100000,
// 		100000,
// 		map[string]uint64{
// 			"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b": 1000,
// 		},
// 		map[string]uint64{
// 			"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b": totalPRV,
// 		},
// 		remoteAddresses,
// 		0,
// 	)

// 	custodianKey2 := statedb.GenerateCustodianStateObjectKey(beaconHeight, "12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy")
// 	newCustodian2 := statedb.NewCustodianStateWithValue(
// 		"12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy",
// 		90000,
// 		90000,
// 		map[string]uint64{
// 			"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b": 2000,
// 		},
// 		map[string]uint64{
// 			"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b": totalPRV_2,
// 		},
// 		remoteAddresses,
// 		0,
// 	)

// 	custodian := make(map[string]*statedb.CustodianState)
// 	custodian[custodianKey.String()] = newCustodian
// 	custodian[custodianKey2.String()] = newCustodian2
// 	suite.currentPortalState.CustodianPoolState = custodian
// }

// func (suite *PortalProducerSuite) SetupMockBlockChain(trieMock *mocks.Trie) *BlockChain {
// 	root := common.Hash{}
// 	wrapperDBMock := new(mocks.DatabaseAccessWarper)
// 	wrapperDBMock.On("OpenPrefixTrie", root).Return(
// 		trieMock,
// 		nil,
// 	)

// 	wrapperDBMock.On("CopyTrie", trieMock).Return(
// 		trieMock,
// 		nil,
// 	)

// 	root1 := common.Hash{}
// 	stateDb, _ := statedb.NewWithPrefixTrie(root1, wrapperDBMock)

// 	beaconBestState := &BeaconBestState{
// 		featureStateDB: stateDb,
// 	}

// 	bestState := &BestState{
// 		Beacon: beaconBestState,
// 	}

// 	blockChain := &BlockChain{
// 		BestState: bestState,
// 	}

// 	return blockChain
// }

// func (suite *PortalProducerSuite) TestBuildInstructionsForPortingRequest() {
// 	happyCases := []PortingRequestTestCase{
// 		{
// 			"happy_case_1",
// 			func() metadata.PortalUserRegisterAction {
// 				meta, _ := metadata.NewPortalUserRegister(
// 					"1",
// 					"12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC", //100.000 prv
// 					"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b",
// 					1000,
// 					4,
// 					metadata.PortalUserRegisterMeta,
// 				)

// 				actionContent := metadata.PortalUserRegisterAction{
// 					Meta:    *meta,
// 					TxReqID: *meta.Hash(),
// 					ShardID: 1,
// 				}
// 				return actionContent
// 			},
// 			func() PortingRequestExcepted {
// 				return PortingRequestExcepted{
// 					Metadata:    strconv.Itoa(metadata.PortalUserRegisterMeta),
// 					ChainStatus: common.PortalPortingRequestAcceptedChainStatus,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"40000", //free collateral
// 						"1000",  //hold pToken
// 						"60000", //lock prv amount
// 					},
// 				}
// 			},
// 		},
// 		{
// 			"happy_case_2",
// 			func() metadata.PortalUserRegisterAction {
// 				meta, _ := metadata.NewPortalUserRegister(
// 					"2",
// 					"12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC", //100.000 prv
// 					"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b",
// 					100,
// 					4,
// 					metadata.PortalUserRegisterMeta,
// 				)

// 				actionContent := metadata.PortalUserRegisterAction{
// 					Meta:    *meta,
// 					TxReqID: *meta.Hash(),
// 					ShardID: 1,
// 				}
// 				return actionContent
// 			},
// 			func() PortingRequestExcepted {
// 				return PortingRequestExcepted{
// 					Metadata:    strconv.Itoa(metadata.PortalUserRegisterMeta),
// 					ChainStatus: common.PortalPortingRequestAcceptedChainStatus,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"34000", //free collateral
// 						"1100",  //hold pToken
// 						"66000", //lock prv amount
// 					},
// 				}
// 			},
// 		},
// 	}

// 	//reset
// 	suite.SetupTest()
// 	suite.SetupExchangeRates(1)
// 	suite.SetupOneCustodian(1)
// 	suite.verifyPortingRequest(happyCases)

// 	pickMultipleCustodianCases := []PortingRequestTestCase{
// 		{
// 			"pick_multiple_custodian_case_1",
// 			func() metadata.PortalUserRegisterAction {
// 				meta, _ := metadata.NewPortalUserRegister(
// 					"1",
// 					"12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC", //100.000 prv
// 					"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b",
// 					2000,
// 					8,
// 					metadata.PortalUserRegisterMeta,
// 				)

// 				actionContent := metadata.PortalUserRegisterAction{
// 					Meta:    *meta,
// 					TxReqID: *meta.Hash(),
// 					ShardID: 1,
// 				}
// 				return actionContent
// 			},
// 			func() PortingRequestExcepted {
// 				return PortingRequestExcepted{
// 					Metadata:    strconv.Itoa(metadata.PortalUserRegisterMeta),
// 					ChainStatus: common.PortalPortingRequestAcceptedChainStatus,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"0",      //free collateral
// 						"1667",   //hold pToken
// 						"100000", //lock prv amount
// 					},
// 					Custodian2: []string{
// 						"12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy", //address
// 						"70000", //free collateral
// 						"333",   //hold pToken
// 						"20000", //lock prv amount
// 					},
// 				}
// 			},
// 		},
// 		{
// 			"pick_a_custodian_case_2",
// 			func() metadata.PortalUserRegisterAction {
// 				meta, _ := metadata.NewPortalUserRegister(
// 					"2",
// 					"12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC", //100.000 prv
// 					"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b",
// 					1000,
// 					4,
// 					metadata.PortalUserRegisterMeta,
// 				)

// 				actionContent := metadata.PortalUserRegisterAction{
// 					Meta:    *meta,
// 					TxReqID: *meta.Hash(),
// 					ShardID: 1,
// 				}
// 				return actionContent
// 			},
// 			func() PortingRequestExcepted {
// 				return PortingRequestExcepted{
// 					Metadata:    strconv.Itoa(metadata.PortalUserRegisterMeta),
// 					ChainStatus: common.PortalPortingRequestAcceptedChainStatus,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"0",      //free collateral
// 						"1667",   //hold pToken
// 						"100000", //lock prv amount
// 					},
// 					Custodian2: []string{
// 						"12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy", //address
// 						"10000", //free collateral
// 						"1333",  //hold pToken
// 						"80000", //lock prv amount
// 					},
// 				}
// 			},
// 		},
// 	}

// 	//reset
// 	suite.SetupTest()
// 	suite.SetupExchangeRates(1)
// 	suite.SetupMultipleCustodian(1)
// 	suite.verifyPortingRequest(pickMultipleCustodianCases)

// 	waitingPortingRequest := []PortingRequestTestCase{
// 		{
// 			"waiting_porting_request_case_1",
// 			func() metadata.PortalUserRegisterAction {
// 				meta, _ := metadata.NewPortalUserRegister(
// 					"1",
// 					"12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC", //100.000 prv
// 					"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b",
// 					2000,
// 					8,
// 					metadata.PortalUserRegisterMeta,
// 				)

// 				actionContent := metadata.PortalUserRegisterAction{
// 					Meta:    *meta,
// 					TxReqID: *meta.Hash(),
// 					ShardID: 1,
// 				}
// 				return actionContent
// 			},
// 			func() PortingRequestExcepted {
// 				return PortingRequestExcepted{
// 					Metadata:    strconv.Itoa(metadata.PortalUserRegisterMeta),
// 					ChainStatus: common.PortalPortingRequestAcceptedChainStatus,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"0",      //free collateral
// 						"1667",   //hold pToken
// 						"100000", //lock prv amount
// 					},
// 					Custodian2: []string{
// 						"12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy", //address
// 						"70000", //free collateral
// 						"333",   //hold pToken
// 						"20000", //lock prv amount
// 					},
// 				}
// 			},
// 		},
// 		{
// 			"waiting_porting_request_exist_case_2",
// 			func() metadata.PortalUserRegisterAction {
// 				meta, _ := metadata.NewPortalUserRegister(
// 					"1",
// 					"12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC", //100.000 prv
// 					"b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b",
// 					1000,
// 					4,
// 					metadata.PortalUserRegisterMeta,
// 				)

// 				actionContent := metadata.PortalUserRegisterAction{
// 					Meta:    *meta,
// 					TxReqID: *meta.Hash(),
// 					ShardID: 1,
// 				}
// 				return actionContent
// 			},
// 			func() PortingRequestExcepted {
// 				return PortingRequestExcepted{
// 					Metadata:    strconv.Itoa(metadata.PortalUserRegisterMeta),
// 					ChainStatus: common.PortalPortingRequestRejectedChainStatus,
// 				}
// 			},
// 		},
// 	}

// 	//reset
// 	suite.SetupTest()
// 	suite.SetupExchangeRates(1)
// 	suite.SetupMultipleCustodian(1)
// 	suite.verifyPortingRequest(waitingPortingRequest)
// }

// func (suite *PortalProducerSuite) TestBuildInstructionsForLiquidationTPExchangeRates() {
// 	//check tp 150
// 	//check tp 130
// 	//check tp 120

// 	//check custodian
// 	//check liquidation pool

// 	exchangeRatesChange := []AutoLiquidationExchangeRatesTestCase{
// 		{
// 			"liquidation_exchange_rates_none_tp150_1",
// 			map[string]uint64{
// 				"btc": 8000000000,
// 				"bnb": 20000000,
// 				"prv": 500000,
// 			},
// 			func() LiquidationExchangeRatesExcepted {
// 				return LiquidationExchangeRatesExcepted{
// 					TpValue: 0,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"100000", //free collateral
// 						"1000",   //hold pToken
// 						"60000",  //lock prv amount
// 					},
// 					Custodian2: nil,
// 					LiquidationPool: []uint64{
// 						0,
// 						0,
// 					},
// 				}
// 			},
// 		},
// 		{
// 			"liquidation_exchange_rates_tp130_2",
// 			map[string]uint64{
// 				"btc": 8000000000,
// 				"bnb": 23000000,
// 				"prv": 500000,
// 			},
// 			func() LiquidationExchangeRatesExcepted {
// 				return LiquidationExchangeRatesExcepted{
// 					TpValue: 130,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"100000", //free collateral
// 						"1000",   //hold pToken
// 						"60000",  //lock prv amount
// 					},
// 					Custodian2: nil,
// 					LiquidationPool: []uint64{
// 						0, //lock ptoken
// 						0, //lock amount collateral
// 					},
// 				}
// 			},
// 		},
// 		{
// 			"liquidation_exchange_rates_tp120_3",
// 			map[string]uint64{
// 				"btc": 8000000000,
// 				"bnb": 25000000,
// 				"prv": 500000,
// 			},
// 			func() LiquidationExchangeRatesExcepted {
// 				return LiquidationExchangeRatesExcepted{
// 					TpValue: 120,
// 					Custodian1: []string{
// 						"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ", //address
// 						"100000", //free collateral
// 						"0",      //hold pToken
// 						"0",      //lock prv amount
// 					},
// 					Custodian2: []string{
// 						"12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy", //address
// 						"90000", //free collateral
// 						"0",     //hold pToken
// 						"0",     //lock prv amount
// 					},
// 					LiquidationPool: []uint64{
// 						3000,   //lock ptoken
// 						180000, //lock amount collateral
// 					},
// 				}
// 			},
// 		},
// 	}

// 	suite.SetupTest()
// 	suite.SetupExchangeRates(1)
// 	suite.SetupMultipleCustodianContainPToken(1)
// 	suite.verifyAutoLiquidationExchangeRates(exchangeRatesChange)
// }

// func (suite *PortalProducerSuite) verifyAutoLiquidationExchangeRates(testCases []AutoLiquidationExchangeRatesTestCase) {
// 	beaconHeight := uint64(1)

// 	for _, testCase := range testCases {
// 		suite.SetupExchangeRatesWithValue(beaconHeight, testCase.Input["btc"], testCase.Input["bnb"], testCase.Input["prv"])

// 		value, _ := buildInstForLiquidationTopPercentileExchangeRates(
// 			beaconHeight,
// 			suite.currentPortalState,
// 		)

// 		fmt.Printf("Testcase %v: instruction %#v", testCase.TestCaseName, value)
// 		fmt.Println()

// 		if testCase.Output().TpValue > 0 {
// 			var actionData metadata.PortalLiquidateTopPercentileExchangeRatesContent
// 			json.Unmarshal([]byte(value[0][3]), &actionData)

// 			if actionData.TP["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"].TPKey != testCase.Output().TpValue { //free collateral
// 				suite.T().Errorf("tp is not equal, %v != %v", actionData.TP["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"].TPKey, testCase.Output().TpValue)
// 			}
// 		}

// 		//custodian 1
// 		if testCase.Output().Custodian1 != nil {
// 			custodianKey := statedb.GenerateCustodianStateObjectKey(beaconHeight, testCase.Output().Custodian1[0])
// 			custodian, ok := suite.currentPortalState.CustodianPoolState[custodianKey.String()]
// 			if !ok {
// 				suite.T().Errorf("custodian %v not found", custodianKey.String())
// 			}

// 			holdPublicToken := custodian.GetHoldingPublicTokens()
// 			lockedAmountCollateral := custodian.GetLockedAmountCollateral()
// 			freeCollateral := custodian.GetFreeCollateral()

// 			fmt.Println("custodian 1")
// 			fmt.Println(testCase.Output().Custodian1)
// 			i1, _ := strconv.ParseUint(testCase.Output().Custodian1[1], 10, 64)
// 			i2, _ := strconv.ParseUint(testCase.Output().Custodian1[2], 10, 64)
// 			i3, _ := strconv.ParseUint(testCase.Output().Custodian1[3], 10, 64)

// 			if i1 != freeCollateral { //free collateral
// 				suite.T().Errorf("free collateral is not equal, %v != %v", i1, freeCollateral)
// 			}

// 			if i2 != holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 				suite.T().Errorf("hold public token is not equal, %v != %v", i2, holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 			}

// 			if i3 != lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 				suite.T().Errorf("lock amount collateral is not equal, %v != %v", i3, lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 			}
// 		}

// 		if testCase.Output().Custodian2 != nil {
// 			custodianKey := statedb.GenerateCustodianStateObjectKey(beaconHeight, testCase.Output().Custodian2[0])
// 			custodian, ok := suite.currentPortalState.CustodianPoolState[custodianKey.String()]
// 			if !ok {
// 				suite.T().Errorf("custodian %v not found", custodianKey.String())
// 			}

// 			holdPublicToken := custodian.GetHoldingPublicTokens()
// 			lockedAmountCollateral := custodian.GetLockedAmountCollateral()
// 			freeCollateral := custodian.GetFreeCollateral()

// 			fmt.Println("custodian 1")
// 			fmt.Println(testCase.Output().Custodian1)
// 			i1, _ := strconv.ParseUint(testCase.Output().Custodian2[1], 10, 64)
// 			i2, _ := strconv.ParseUint(testCase.Output().Custodian2[2], 10, 64)
// 			i3, _ := strconv.ParseUint(testCase.Output().Custodian2[3], 10, 64)

// 			if i1 != freeCollateral { //free collateral
// 				suite.T().Errorf("free collateral is not equal, %v != %v", i1, freeCollateral)
// 			}

// 			if i2 != holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 				suite.T().Errorf("hold public token is not equal, %v != %v", i2, holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 			}

// 			if i3 != lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 				suite.T().Errorf("lock amount collateral is not equal, %v != %v", i3, lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 			}
// 		}

// 		//liquidation pool
// 		if testCase.Output().LiquidationPool != nil {
// 			liquidationPoolKey := statedb.GeneratePortalLiquidationPoolObjectKey(beaconHeight)
// 			liquidationPool, ok := suite.currentPortalState.LiquidationPool[liquidationPoolKey.String()]

// 			if ok && testCase.Output().LiquidationPool[0] != liquidationPool.Rates()["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"].PubTokenAmount {
// 				suite.T().Errorf("hold public token is not equal, %v != %v", testCase.Output().LiquidationPool[0], liquidationPool.Rates()["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"].PubTokenAmount)
// 			}

// 			if ok && testCase.Output().LiquidationPool[1] != liquidationPool.Rates()["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"].CollateralAmount {
// 				suite.T().Errorf("hold amount collateral is not equal, %v != %v", testCase.Output().LiquidationPool[1], liquidationPool.Rates()["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"].CollateralAmount)
// 			}
// 		}
// 	}
// }

// func (suite *PortalProducerSuite) verifyPortingRequest(testCases []PortingRequestTestCase) {
// 	trieMock := new(mocks.Trie)
// 	beaconHeight := uint64(1)

// 	for _, testCase := range testCases {
// 		actionContentBytes, _ := json.Marshal(testCase.Input())
// 		actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)

// 		key := statedb.GeneratePortalStatusObjectKey(statedb.PortalPortingRequestStatusPrefix(), []byte(testCase.Input().Meta.UniqueRegisterId))
// 		trieMock.On("TryGet", key[:]).Return(nil, nil)

// 		blockChain := suite.SetupMockBlockChain(trieMock)

// 		value, err := blockChain.buildInstructionsForPortingRequest(
// 			actionContentBase64Str,
// 			testCase.Input().ShardID,
// 			testCase.Input().Meta.Type,
// 			suite.currentPortalState,
// 			beaconHeight,
// 		)

// 		fmt.Printf("Testcase %v: instruction %#v", testCase.TestCaseName, value)
// 		fmt.Println()

// 		assert.Equal(suite.T(), err, nil)

// 		if len(testCase.Output().Metadata) > 0 {
// 			assert.Equal(suite.T(), testCase.Output().Metadata, value[0][0])
// 		}

// 		assert.Equal(suite.T(), strconv.Itoa(1), value[0][1])

// 		if len(testCase.Output().ChainStatus) > 0 {
// 			assert.Equal(suite.T(), testCase.Output().ChainStatus, value[0][2])
// 		}

// 		assert.NotNil(suite.T(), value[0][3])

// 		//test current portal state
// 		var portingRequestContent metadata.PortalPortingRequestContent
// 		json.Unmarshal([]byte(value[0][3]), &portingRequestContent)

// 		prettyJSON, _ := json.MarshalIndent(portingRequestContent, "", "  ")
// 		fmt.Printf("Porting request result: %s\n", string(prettyJSON))

// 		for _, itemCustodian := range portingRequestContent.Custodian {
// 			//update custodian state
// 			custodianKey := statedb.GenerateCustodianStateObjectKey(beaconHeight, itemCustodian.IncAddress)
// 			custodian := suite.currentPortalState.CustodianPoolState[custodianKey.String()]

// 			if testCase.Output().Custodian1 != nil && itemCustodian.IncAddress == testCase.Output().Custodian1[0] {
// 				holdPublicToken := custodian.GetHoldingPublicTokens()
// 				lockedAmountCollateral := custodian.GetLockedAmountCollateral()
// 				freeCollateral := custodian.GetFreeCollateral()

// 				fmt.Println("custodian 1")
// 				fmt.Println(testCase.Output().Custodian1)
// 				i1, _ := strconv.ParseUint(testCase.Output().Custodian1[1], 10, 64)
// 				i2, _ := strconv.ParseUint(testCase.Output().Custodian1[2], 10, 64)
// 				i3, _ := strconv.ParseUint(testCase.Output().Custodian1[3], 10, 64)

// 				if i1 != freeCollateral { //free collateral
// 					suite.T().Errorf("free collateral is not equal, %v != %v", i1, freeCollateral)
// 				}

// 				if i2 != holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 					suite.T().Errorf("hold public token is not equal, %v != %v", i2, holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 				}

// 				if i3 != lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 					suite.T().Errorf("lock amount collateral is not equal, %v != %v", i3, lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 				}
// 			}

// 			if testCase.Output().Custodian2 != nil && itemCustodian.IncAddress == testCase.Output().Custodian2[0] {
// 				holdPublicToken := custodian.GetHoldingPublicTokens()
// 				lockedAmountCollateral := custodian.GetLockedAmountCollateral()
// 				freeCollateral := custodian.GetFreeCollateral()

// 				fmt.Println("custodian 2")
// 				fmt.Println(testCase.Output().Custodian2)
// 				i1, _ := strconv.ParseUint(testCase.Output().Custodian2[1], 10, 64)
// 				i2, _ := strconv.ParseUint(testCase.Output().Custodian2[2], 10, 64)
// 				i3, _ := strconv.ParseUint(testCase.Output().Custodian2[3], 10, 64)

// 				if i1 != freeCollateral { //free collateral
// 					suite.T().Errorf("free collateral is not equal, %v != %v", i1, freeCollateral)
// 				}

// 				if i2 != holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 					suite.T().Errorf("hold public token is not equal, %v != %v", i2, holdPublicToken["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 				}

// 				if i3 != lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] {
// 					suite.T().Errorf("lock amount collateral is not equal, %v != %v", i3, lockedAmountCollateral["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"])
// 				}
// 			}
// 		}
// 	}
// }

// /************************ Custodian deposit test ************************/
// const ShardIDHardCode = 0
// const BeaconHeight = 1
// const BNBTokenID = "b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"
// const BNBRemoteAddress = "tbnb1fau9kq605jwkyfea2knw495we8cpa47r9r6uxv"

// type CustodianDepositOutput struct {
// 	MetadataType            string
// 	ChainStatus             string
// 	CustodianDepositContent string
// 	CustodianPool           map[string]*statedb.CustodianState
// }

// type CustodianDepositInput struct {
// 	IncognitoAddress string
// 	RemoteAddresses  []statedb.RemoteAddress
// 	DepositedAmount  uint64
// }

// type CustodianDepositTestCase struct {
// 	TestCaseName string
// 	Input        CustodianDepositInput
// 	Output       CustodianDepositOutput
// }

// func buildPortalCustodianDepositAction(
// 	incogAddressStr string,
// 	remoteAddresses map[string]string,
// 	depositedAmount uint64,
// ) []string {
// 	custodianDepositMeta, _ := metadata.NewPortalCustodianDeposit(
// 		metadata.PortalCustodianDepositMeta,
// 		incogAddressStr,
// 		remoteAddresses,
// 		depositedAmount,
// 	)

// 	actionContent := metadata.PortalCustodianDepositAction{
// 		Meta:    *custodianDepositMeta,
// 		TxReqID: common.Hash{},
// 		ShardID: byte(ShardIDHardCode),
// 	}
// 	actionContentBytes, _ := json.Marshal(actionContent)

// 	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
// 	action := []string{strconv.Itoa(metadata.PortalCustodianDepositMeta), actionContentBase64Str}
// 	return action
// }

// func buildPortalCustodianDepositContent(
// 	custodianAddressStr string,
// 	remoteAddresses []statedb.RemoteAddress,
// 	depositedAmount uint64,
// ) string {
// 	custodianDepositContent := metadata.PortalCustodianDepositContent{
// 		IncogAddressStr: custodianAddressStr,
// 		RemoteAddresses: remoteAddresses,
// 		DepositedAmount: depositedAmount,
// 		TxReqID:         common.Hash{},
// 		ShardID:         byte(ShardIDHardCode),
// 	}
// 	custodianDepositContentBytes, _ := json.Marshal(custodianDepositContent)
// 	return string(custodianDepositContentBytes)
// }

// func getTestCasesForCustodianDeposit() []*CustodianDepositTestCase {
// 	testcases := []*CustodianDepositTestCase{
// 		{
// 			TestCaseName: "Custodian deposit when custodian pool is empty",
// 			Input: CustodianDepositInput{
// 				IncognitoAddress: "12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ",
// 				RemoteAddresses: []statedb.RemoteAddress{
// 					*statedb.NewRemoteAddressWithValue(
// 						BNBTokenID,
// 						BNBRemoteAddress),
// 				},
// 				DepositedAmount: 1000 * 1e9,
// 			},
// 			Output: CustodianDepositOutput{
// 				MetadataType:            strconv.Itoa(metadata.PortalCustodianDepositMeta),
// 				ChainStatus:             common.PortalCustodianDepositAcceptedChainStatus,
// 				CustodianDepositContent: "",
// 			},
// 		},
// 		{
// 			TestCaseName: "Custodian deposit when custodian pool has one custodian before",
// 			Input: CustodianDepositInput{
// 				IncognitoAddress: "12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy",
// 				RemoteAddresses: []statedb.RemoteAddress{
// 					*statedb.NewRemoteAddressWithValue(
// 						BNBTokenID,
// 						BNBRemoteAddress),
// 				},
// 				DepositedAmount: 2000 * 1e9,
// 			},
// 			Output: CustodianDepositOutput{
// 				MetadataType:            strconv.Itoa(metadata.PortalCustodianDepositMeta),
// 				ChainStatus:             common.PortalCustodianDepositAcceptedChainStatus,
// 				CustodianDepositContent: "",
// 			},
// 		}, {
// 			TestCaseName: "Custodian deposit more",
// 			Input: CustodianDepositInput{
// 				IncognitoAddress: "12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ",
// 				RemoteAddresses: []statedb.RemoteAddress{
// 					*statedb.NewRemoteAddressWithValue(
// 						BNBTokenID,
// 						BNBRemoteAddress),
// 				},
// 				DepositedAmount: 3000 * 1e9,
// 			},
// 			Output: CustodianDepositOutput{
// 				MetadataType:            strconv.Itoa(metadata.PortalCustodianDepositMeta),
// 				ChainStatus:             common.PortalCustodianDepositAcceptedChainStatus,
// 				CustodianDepositContent: "",
// 			},
// 		},
// 	}

// 	custodianPool := make(map[string]*statedb.CustodianState, 0)
// 	for i := 0; i < len(testcases); i++ {
// 		testcases[i].Output.CustodianDepositContent = buildPortalCustodianDepositContent(
// 			testcases[i].Input.IncognitoAddress,
// 			testcases[i].Input.RemoteAddresses,
// 			testcases[i].Input.DepositedAmount)

// 		custodianKey := statedb.GenerateCustodianStateObjectKey(testcases[i].Input.IncognitoAddress)
// 		if custodianPool[custodianKey.String()] == nil {
// 			custodianState := statedb.NewCustodianStateWithValue(
// 				testcases[i].Input.IncognitoAddress,
// 				testcases[i].Input.DepositedAmount,
// 				testcases[i].Input.DepositedAmount,
// 				nil, nil,
// 				testcases[i].Input.RemoteAddresses,
// 				0,
// 			)
// 			custodianPool[custodianKey.String()] = custodianState
// 		} else {
// 			custodianPool[custodianKey.String()].SetFreeCollateral(custodianPool[custodianKey.String()].GetFreeCollateral() + testcases[i].Input.DepositedAmount)
// 			custodianPool[custodianKey.String()].SetTotalCollateral(custodianPool[custodianKey.String()].GetTotalCollateral() + testcases[i].Input.DepositedAmount)
// 		}
// 		custodianPoolTmp := map[string]*statedb.CustodianState{}
// 		for key, cus := range custodianPool {
// 			custodianPoolTmp[key] = statedb.NewCustodianStateWithValue(
// 				cus.GetIncognitoAddress(),
// 				cus.GetTotalCollateral(),
// 				cus.GetFreeCollateral(),
// 				cus.GetHoldingPublicTokens(),
// 				cus.GetLockedAmountCollateral(),
// 				cus.GetRemoteAddresses(),
// 				cus.GetRewardAmount(),
// 			)
// 		}
// 		testcases[i].Output.CustodianPool = custodianPoolTmp
// 	}
// 	return testcases
// }

// func (suite *PortalProducerSuite) TestCustodianDeposit() {
// 	testcases := getTestCasesForCustodianDeposit()

// 	for _, tc := range testcases {
// 		fmt.Printf("[Custodian deposit] Running test case: %v\n", tc.TestCaseName)
// 		// build custodian deposit action
// 		action := buildPortalCustodianDepositAction(tc.Input.IncognitoAddress, tc.Input.RemoteAddresses, tc.Input.DepositedAmount)

// 		// beacon build new instruction for the action
// 		bc := BlockChain{}
// 		shardID := byte(ShardIDHardCode)
// 		metaType, _ := strconv.Atoi(action[0])
// 		contentStr := action[1]
// 		newInsts, err := bc.buildInstructionsForCustodianDeposit(contentStr, shardID, metaType, suite.currentPortalState, uint64(BeaconHeight))

// 		// compare results to Outputs of test case
// 		suite.Nil(err)
// 		suite.Equal(1, len(newInsts))
// 		newInst := newInsts[0]
// 		suite.Equal(tc.Output.MetadataType, newInst[0])
// 		suite.Equal(tc.Output.ChainStatus, newInst[2])
// 		suite.Equal(tc.Output.CustodianDepositContent, newInst[3])
// 		suite.EqualValues(tc.Output.CustodianPool, suite.currentPortalState.CustodianPoolState)
// 	}
// }

// /************************ Redeem request test ************************/
// type RedeemRequestOutput struct {
// 	MetadataType         string
// 	ChainStatus          string
// 	RedeemRequestContent string
// 	CustodianPool        map[string]*statedb.CustodianState
// 	WaitingRedeemRequest map[string]*statedb.RedeemRequest
// }

// type RedeemRequestInput struct {
// 	UniqueRedeemID     string
// 	TokenID            string
// 	RedeemAmount       uint64
// 	RedeemerIncAddress string
// 	RedeemerRemoteAddr string
// 	RedeemFee          uint64
// }

// type RedeemRequestTestCase struct {
// 	TestCaseName string
// 	Input        RedeemRequestInput
// 	Output       RedeemRequestOutput
// }

// func buildPortalRedeemRequestAction(
// 	uniqueRedeemID string,
// 	tokenID string,
// 	redeemAmount uint64,
// 	incAddressStr string,
// 	remoteAddr string,
// 	redeemFee uint64,
// ) []string {
// 	redeemRequestMeta, _ := metadata.NewPortalRedeemRequest(
// 		metadata.PortalRedeemRequestMeta,
// 		uniqueRedeemID,
// 		tokenID,
// 		redeemAmount,
// 		incAddressStr,
// 		remoteAddr,
// 		redeemFee,
// 	)

// 	actionContent := metadata.PortalRedeemRequestAction{
// 		Meta:    *redeemRequestMeta,
// 		TxReqID: common.Hash{},
// 		ShardID: byte(ShardIDHardCode),
// 	}
// 	actionContentBytes, _ := json.Marshal(actionContent)
// 	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
// 	action := []string{strconv.Itoa(metadata.PortalRedeemRequestMeta), actionContentBase64Str}
// 	return action
// }

// func buildPortalRedeemRequestContent(
// 	uniqueRedeemID string,
// 	tokenID string,
// 	redeemAmount uint64,
// 	incAddressStr string,
// 	remoteAddr string,
// 	redeemFee uint64,
// 	matchingCustodianDetail []*statedb.MatchingRedeemCustodianDetail,
// ) string {
// 	redeemRequestContent := metadata.PortalRedeemRequestContent{
// 		UniqueRedeemID:          uniqueRedeemID,
// 		TokenID:                 tokenID,
// 		RedeemAmount:            redeemAmount,
// 		RedeemerIncAddressStr:   incAddressStr,
// 		RemoteAddress:           remoteAddr,
// 		RedeemFee:               redeemFee,
// 		MatchingCustodianDetail: matchingCustodianDetail,
// 		TxReqID:                 common.Hash{},
// 		ShardID:                 byte(ShardIDHardCode),
// 	}
// 	redeemRequestContentBytes, _ := json.Marshal(redeemRequestContent)
// 	return string(redeemRequestContentBytes)
// }

// func (suite *PortalProducerSuite) SetupRedeemRequest(beaconHeight uint64) {
// 	// set up exchange rates
// 	rates := make(map[string]statedb.FinalExchangeRatesDetail)
// 	rates["b832e5d3b1f01a4f0623f7fe91d6673461e1f5d37d91fe78c5c2e6183ff39696"] = statedb.FinalExchangeRatesDetail{
// 		Amount: 8000000000,
// 	}
// 	rates["b2655152784e8639fa19521a7035f331eea1f1e911b2f3200a507ebb4554387b"] = statedb.FinalExchangeRatesDetail{
// 		Amount: 20000000,
// 	}
// 	rates["0000000000000000000000000000000000000000000000000000000000000004"] = statedb.FinalExchangeRatesDetail{
// 		Amount: 500000,
// 	}

// 	exchangeRates := make(map[string]*statedb.FinalExchangeRatesState)
// 	exchangeRatesKey := statedb.GeneratePortalFinalExchangeRatesStateObjectKey(beaconHeight)
// 	exchangeRates[exchangeRatesKey.String()] = statedb.NewFinalExchangeRatesStateWithValue(rates)
// 	suite.currentPortalState.FinalExchangeRatesState = exchangeRates

// 	// set up custodian pool
// 	remoteAddresses := make([]statedb.RemoteAddress, 0)
// 	remoteAddresses = append(
// 		remoteAddresses,
// 		*statedb.NewRemoteAddressWithValue(BNBTokenID, BNBRemoteAddress),
// 	)

// 	custodianStates := []*statedb.CustodianState{
// 		statedb.NewCustodianStateWithValue(
// 			"12RuEdPjq4yxivzm8xPxRVHmkL74t4eAdUKPdKKhMEnpxPH3k8GEyULbwq4hjwHWmHQr7MmGBJsMpdCHsYAqNE18jipWQwciBf9yqvQ",
// 			1000*1e9,
// 			400*1e9,
// 			map[string]uint64{
// 				BNBTokenID: 10 * 1e9, // hold 10 BNB
// 			},
// 			map[string]uint64{
// 				BNBTokenID: 600 * 1e9, // lock 600 PRV
// 			},
// 			remoteAddresses,
// 			0,
// 		),
// 		statedb.NewCustodianStateWithValue(
// 			"12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy",
// 			5000*1e9,
// 			2000*1e9,
// 			map[string]uint64{
// 				BNBTokenID: 50 * 1e9, // hold 50 BNB
// 			},
// 			map[string]uint64{
// 				BNBTokenID: 3000 * 1e9, // lock 3000 PRV
// 			},
// 			remoteAddresses,
// 			0,
// 		),
// 	}

// 	custodian := make(map[string]*statedb.CustodianState)
// 	for _, cus := range custodianStates {
// 		custodianKey := statedb.GenerateCustodianStateObjectKey(beaconHeight, cus.GetIncognitoAddress())
// 		custodian[custodianKey.String()] = cus
// 	}

// 	suite.currentPortalState.CustodianPoolState = custodian
// }

// func getTestCasesForRedeemRequest() []*RedeemRequestTestCase {
// 	testcases := []*RedeemRequestTestCase{
// 		{
// 			TestCaseName: "Redeem request matches to one custodian",
// 			Input: RedeemRequestInput{
// 				UniqueRedeemID:     "1",
// 				TokenID:            BNBTokenID,
// 				RedeemAmount:       1 * 1e9,
// 				RedeemerIncAddress: "12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC",
// 				RedeemerRemoteAddr: BNBRemoteAddress,
// 				RedeemFee:          0.004 * 1e9,
// 			},
// 			Output: RedeemRequestOutput{
// 				MetadataType:         strconv.Itoa(metadata.PortalRedeemRequestMeta),
// 				ChainStatus:          common.PortalRedeemRequestAcceptedChainStatus,
// 				RedeemRequestContent: "",
// 			},
// 		},
// 		{
// 			TestCaseName: "Redeem request matches to two custodians",
// 			Input: RedeemRequestInput{
// 				UniqueRedeemID:     "1",
// 				TokenID:            BNBTokenID,
// 				RedeemAmount:       51 * 1e9,
// 				RedeemerIncAddress: "12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC",
// 				RedeemerRemoteAddr: BNBRemoteAddress,
// 				RedeemFee:          0.204 * 1e9,
// 			},
// 			Output: RedeemRequestOutput{
// 				MetadataType:         strconv.Itoa(metadata.PortalRedeemRequestMeta),
// 				ChainStatus:          common.PortalRedeemRequestAcceptedChainStatus,
// 				RedeemRequestContent: "",
// 			},
// 		},
// 		{
// 			TestCaseName: "Redeem request with fee less than min redeem fee",
// 			Input: RedeemRequestInput{
// 				UniqueRedeemID:     "1",
// 				TokenID:            BNBTokenID,
// 				RedeemAmount:       1 * 1e9,
// 				RedeemerIncAddress: "12S5pBBRDf1GqfRHouvCV86sWaHzNfvakAWpVMvNnWu2k299xWCgQzLLc9wqPYUHfMYGDprPvQ794dbi6UU1hfRN4tPiU61txWWenhC",
// 				RedeemerRemoteAddr: BNBRemoteAddress,
// 				RedeemFee:          0.003 * 1e9,
// 			},
// 			Output: RedeemRequestOutput{
// 				MetadataType:         strconv.Itoa(metadata.PortalRedeemRequestMeta),
// 				ChainStatus:          common.PortalRedeemRequestRejectedChainStatus,
// 				RedeemRequestContent: "",
// 			},
// 		},
// 	}

// 	//custodianPool := make(map[string]*statedb.CustodianState, 0)
// 	//for i := 0; i < len(testcases); i++ {
// 	//	testcases[i].Output.CustodianDepositContent = buildPortalCustodianDepositContent(
// 	//		testcases[i].Input.IncognitoAddress,
// 	//		testcases[i].Input.RemoteAddresses,
// 	//		testcases[i].Input.DepositedAmount)
// 	//
// 	//	custodianKey := statedb.GenerateCustodianStateObjectKey(uint64(BeaconHeight), testcases[i].Input.IncognitoAddress)
// 	//	if custodianPool[custodianKey.String()] == nil {
// 	//		custodianState := statedb.NewCustodianStateWithValue(
// 	//			testcases[i].Input.IncognitoAddress,
// 	//			testcases[i].Input.DepositedAmount,
// 	//			testcases[i].Input.DepositedAmount,
// 	//			nil, nil,
// 	//			testcases[i].Input.RemoteAddresses,
// 	//			0,
// 	//		)
// 	//		custodianPool[custodianKey.String()] = custodianState
// 	//	} else {
// 	//		custodianPool[custodianKey.String()].SetFreeCollateral(custodianPool[custodianKey.String()].GetFreeCollateral() + testcases[i].Input.DepositedAmount)
// 	//		custodianPool[custodianKey.String()].SetTotalCollateral(custodianPool[custodianKey.String()].GetTotalCollateral() + testcases[i].Input.DepositedAmount)
// 	//	}
// 	//	custodianPoolTmp := map[string]*statedb.CustodianState{}
// 	//	for key, cus := range custodianPool {
// 	//		custodianPoolTmp[key] = statedb.NewCustodianStateWithValue(
// 	//			cus.GetIncognitoAddress(),
// 	//			cus.GetTotalCollateral(),
// 	//			cus.GetFreeCollateral(),
// 	//			cus.GetHoldingPublicTokens(),
// 	//			cus.GetLockedAmountCollateral(),
// 	//			cus.GetRemoteAddresses(),
// 	//			cus.GetRewardAmount(),
// 	//		)
// 	//	}
// 	//	testcases[i].Output.CustodianPool = custodianPoolTmp
// 	//}
// 	return testcases
// }

// func (suite *PortalProducerSuite) TestRedeemRequest() {
// 	//testcases := getTestCasesForRedeemRequest()
// 	//
// 	//for _, tc := range testcases {
// 	//	fmt.Printf("[Redeem Request] Running test case: %v\n", tc.TestCaseName)
// 	//	// build redeem request action
// 	//	action := buildPortalRedeemRequestAction(
// 	//		tc.Input.UniqueRedeemID, tc.Input.TokenID, tc.Input.RedeemAmount,
// 	//		tc.Input.RedeemerIncAddress, tc.Input.RedeemerRemoteAddr,  tc.Input.RedeemFee)
// 	//
// 	//	// beacon build new instruction for the action
// 	//	bc := BlockChain{}
// 	//	shardID := byte(ShardIDHardCode)
// 	//	metaType, _ := strconv.Atoi(action[0])
// 	//	contentStr := action[1]
// 	//	newInsts, err := bc.buildInstructionsForRedeemRequest(statedb, contentStr, shardID, metaType, suite.currentPortalState, uint64(BeaconHeight))
// 	//
// 	//	// compare results to Outputs of test case
// 	//	suite.Nil(err)
// 	//	suite.Equal(1, len(newInsts))
// 	//	newInst := newInsts[0]
// 	//	suite.Equal(tc.Output.MetadataType, newInst[0])
// 	//	suite.Equal(tc.Output.ChainStatus, newInst[2])
// 	//	suite.Equal(tc.Output.CustodianDepositContent, newInst[3])
// 	//	suite.EqualValues(tc.Output.CustodianPool, suite.currentPortalState.CustodianPoolState)
// 	//}
// }

// /************************ Run suite test ************************/
// // In order for 'go test' to run this suite, we need to create
// // a normal test function and pass our suite to suite.Run
// func TestPortalProducerSuite(t *testing.T) {
// 	suite.Run(t, new(PortalProducerSuite))
// }
