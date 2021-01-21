package rpcserver

import "github.com/incognitochain/incognito-chain/rpcserver/rpcservice"

type httpHandler func(*HttpServer, interface{}, <-chan struct{}) (interface{}, *rpcservice.RPCError)
type wsHandler func(*WsServer, interface{}, string, chan RpcSubResult, <-chan struct{})

// Commands valid for normal user
var HttpHandler = map[string]httpHandler{
	//Test Rpc Server
	testHttpServer: (*HttpServer).handleTestHttpServer,

	//profiling
	startProfiling: (*HttpServer).handleStartProfiling,
	stopProfiling:  (*HttpServer).handleStopProfiling,
	exportMetrics:  (*HttpServer).handleExportMetrics,

	// node
	getNodeRole:              (*HttpServer).handleGetNodeRole,
	getNetworkInfo:           (*HttpServer).handleGetNetWorkInfo,
	getConnectionCount:       (*HttpServer).handleGetConnectionCount,
	getAllConnectedPeers:     (*HttpServer).handleGetAllConnectedPeers,
	getInOutMessages:         (*HttpServer).handleGetInOutMessages,
	getInOutMessageCount:     (*HttpServer).handleGetInOutMessageCount,
	getAllPeers:              (*HttpServer).handleGetAllPeers,
	estimateFee:              (*HttpServer).handleEstimateFee,
	estimateFeeV2:            (*HttpServer).handleEstimateFeeV2,
	estimateFeeWithEstimator: (*HttpServer).handleEstimateFeeWithEstimator,
	getActiveShards:          (*HttpServer).handleGetActiveShards,
	getMaxShardsNumber:       (*HttpServer).handleGetMaxShardsNumber,

	//tx pool
	getRawMempool:           (*HttpServer).handleGetRawMempool,
	getNumberOfTxsInMempool: (*HttpServer).handleGetNumberOfTxsInMempool,
	getMempoolEntry:         (*HttpServer).handleMempoolEntry,
	removeTxInMempool:       (*HttpServer).handleRemoveTxInMempool,
	getMempoolInfo:          (*HttpServer).handleGetMempoolInfo,
	getPendingTxsInBlockgen: (*HttpServer).handleGetPendingTxsInBlockgen,

	// block pool ver.2
	// getCrossShardPoolStateV2:    (*HttpServer).handleGetCrossShardPoolStateV2,
	// getShardPoolStateV2:         (*HttpServer).handleGetShardPoolStateV2,
	// getBeaconPoolStateV2:        (*HttpServer).handleGetBeaconPoolStateV2,
	// // ver.1
	// //getCrossShardPoolState:    (*HttpServer).handleGetCrossShardPoolState,
	// getNextCrossShard: (*HttpServer).handleGetNextCrossShard,

	//backup and preload
	setBackup:       (*HttpServer).handleSetBackup,
	getLatestBackup: (*HttpServer).handleGetLatestBackup,
	// block
	getBestBlock:                (*HttpServer).handleGetBestBlock,
	getBestBlockHash:            (*HttpServer).handleGetBestBlockHash,
	retrieveBlock:               (*HttpServer).handleRetrieveBlock,
	retrieveBlockByHeight:       (*HttpServer).handleRetrieveBlockByHeight,
	retrieveBeaconBlock:         (*HttpServer).handleRetrieveBeaconBlock,
	retrieveBeaconBlockByHeight: (*HttpServer).handleRetrieveBeaconBlockByHeight,
	getBlocks:                   (*HttpServer).handleGetBlocks,
	getBlockChainInfo:           (*HttpServer).handleGetBlockChainInfo,
	getBlockCount:               (*HttpServer).handleGetBlockCount,
	getBlockHash:                (*HttpServer).handleGetBlockHash,
	checkHashValue:              (*HttpServer).handleCheckHashValue, // get data in blockchain from hash value
	getBlockHeader:              (*HttpServer).handleGetBlockHeader, // Current committee, next block committee and candidate is included in block header
	getCrossShardBlock:          (*HttpServer).handleGetCrossShardBlock,

	// transaction
	listOutputCoins:                           (*HttpServer).handleListOutputCoins,
	createRawTransaction:                      (*HttpServer).handleCreateRawTransaction,
	sendRawTransaction:                        (*HttpServer).handleSendRawTransaction,
	createAndSendTransaction:                  (*HttpServer).handleCreateAndSendTx,
	createAndSendTransactionV2:                (*HttpServer).handleCreateAndSendTxV2,
	getTransactionByHash:                      (*HttpServer).handleGetTransactionByHash,
	gettransactionhashbyreceiver:              (*HttpServer).handleGetTransactionHashByReceiver,
	gettransactionhashbyreceiverv2:            (*HttpServer).handleGetTransactionHashByReceiverV2,
	gettransactionbyreceiver:                  (*HttpServer).handleGetTransactionByReceiver,
	gettransactionbyreceiverv2:                (*HttpServer).handleGetTransactionByReceiverV2,
	createAndSendStakingTransaction:           (*HttpServer).handleCreateAndSendStakingTx,
	createAndSendStakingTransactionV2:         (*HttpServer).handleCreateAndSendStakingTxV2,
	createAndSendStopAutoStakingTransaction:   (*HttpServer).handleCreateAndSendStopAutoStakingTransaction,
	createAndSendStopAutoStakingTransactionV2: (*HttpServer).handleCreateAndSendStopAutoStakingTransactionV2,
	randomCommitments:                         (*HttpServer).handleRandomCommitments,
	hasSerialNumbers:                          (*HttpServer).handleHasSerialNumbers,
	hasSnDerivators:                           (*HttpServer).handleHasSnDerivators,
	listSerialNumbers:                         (*HttpServer).handleListSerialNumbers,
	listCommitments:                           (*HttpServer).handleListCommitments,
	listCommitmentIndices:                     (*HttpServer).handleListCommitmentIndices,
	decryptoutputcoinbykeyoftransaction:       (*HttpServer).handleDecryptOutputCoinByKeyOfTransaction,

	//======Testing and Benchmark======
	getAndSendTxsFromFile:   (*HttpServer).handleGetAndSendTxsFromFile,
	getAndSendTxsFromFileV2: (*HttpServer).handleGetAndSendTxsFromFileV2,
	unlockMempool:           (*HttpServer).handleUnlockMempool,
	getAutoStakingByHeight:  (*HttpServer).handleGetAutoStakingByHeight,
	getCommitteeState:       (*HttpServer).handleGetCommitteeState,
	getRewardAmountByEpoch:  (*HttpServer).handleGetRewardAmountByEpoch,

	//=================================

	// Beststate
	getCandidateList:         (*HttpServer).handleGetCandidateList,
	getCommitteeList:         (*HttpServer).handleGetCommitteeList,
	getShardBestState:        (*HttpServer).handleGetShardBestState,
	getShardBestStateDetail:  (*HttpServer).handleGetShardBestStateDetail,
	getBeaconBestState:       (*HttpServer).handleGetBeaconBestState,
	getBeaconBestStateDetail: (*HttpServer).handleGetBeaconBestStateDetail,
	// getBeaconPoolState:            (*HttpServer).handleGetBeaconPoolState,
	// getShardPoolState:             (*HttpServer).handleGetShardPoolState,
	// getShardPoolLatestValidHeight: (*HttpServer).handleGetShardPoolLatestValidHeight,
	canPubkeyStake:      (*HttpServer).handleCanPubkeyStake,
	getTotalTransaction: (*HttpServer).handleGetTotalTransaction,

	// custom token which support privacy
	createRawPrivacyCustomTokenTransaction:       (*HttpServer).handleCreateRawPrivacyCustomTokenTransaction,
	sendRawPrivacyCustomTokenTransaction:         (*HttpServer).handleSendRawPrivacyCustomTokenTransaction,
	createAndSendPrivacyCustomTokenTransaction:   (*HttpServer).handleCreateAndSendPrivacyCustomTokenTransaction,
	createAndSendPrivacyCustomTokenTransactionV2: (*HttpServer).handleCreateAndSendPrivacyCustomTokenTransactionV2,
	listPrivacyCustomToken:                       (*HttpServer).handleListPrivacyCustomToken,
	getPrivacyCustomToken:                        (*HttpServer).handleGetPrivacyCustomToken,
	listPrivacyCustomTokenByShard:                (*HttpServer).handleListPrivacyCustomTokenByShard,
	privacyCustomTokenTxs:                        (*HttpServer).handlePrivacyCustomTokenDetail,
	getListPrivacyCustomTokenBalance:             (*HttpServer).handleGetListPrivacyCustomTokenBalance,
	getBalancePrivacyCustomToken:                 (*HttpServer).handleGetBalancePrivacyCustomToken,

	// Bridge
	createIssuingRequest:              (*HttpServer).handleCreateIssuingRequest,
	sendIssuingRequest:                (*HttpServer).handleSendIssuingRequest,
	createAndSendIssuingRequest:       (*HttpServer).handleCreateAndSendIssuingRequest,
	createAndSendIssuingRequestV2:     (*HttpServer).handleCreateAndSendIssuingRequestV2,
	createAndSendContractingRequest:   (*HttpServer).handleCreateAndSendContractingRequest,
	createAndSendContractingRequestV2: (*HttpServer).handleCreateAndSendContractingRequestV2,
	checkETHHashIssued:                (*HttpServer).handleCheckETHHashIssued,
	getAllBridgeTokens:                (*HttpServer).handleGetAllBridgeTokens,
	getETHHeaderByHash:                (*HttpServer).handleGetETHHeaderByHash,
	getBridgeReqWithStatus:            (*HttpServer).handleGetBridgeReqWithStatus,
	generateTokenID:                   (*HttpServer).handleGenerateTokenID,

	// wallet
	getPublicKeyFromPaymentAddress:     (*HttpServer).handleGetPublicKeyFromPaymentAddress,
	defragmentAccount:                  (*HttpServer).handleDefragmentAccount,
	defragmentAccountV2:                (*HttpServer).handleDefragmentAccountV2,
	defragmentAccountToken:             (*HttpServer).handleDefragmentAccountToken,
	defragmentAccountTokenV2:           (*HttpServer).handleDefragmentAccountTokenV2,
	getStackingAmount:                  (*HttpServer).handleGetStakingAmount,
	hashToIdenticon:                    (*HttpServer).handleHashToIdenticon,
	createAndSendBurningRequest:        (*HttpServer).handleCreateAndSendBurningRequest,
	createAndSendBurningRequestV2:      (*HttpServer).handleCreateAndSendBurningRequestV2,
	createAndSendTxWithIssuingETHReq:   (*HttpServer).handleCreateAndSendTxWithIssuingETHReq,
	createAndSendTxWithIssuingETHReqV2: (*HttpServer).handleCreateAndSendTxWithIssuingETHReqV2,

	// Incognito -> Ethereum bridge
	getBeaconSwapProof:       (*HttpServer).handleGetBeaconSwapProof,
	getLatestBeaconSwapProof: (*HttpServer).handleGetLatestBeaconSwapProof,
	getBridgeSwapProof:       (*HttpServer).handleGetBridgeSwapProof,
	getLatestBridgeSwapProof: (*HttpServer).handleGetLatestBridgeSwapProof,
	getBurnProof:             (*HttpServer).handleGetBurnProof,

	//reward
	CreateRawWithDrawTransaction: (*HttpServer).handleCreateAndSendWithDrawTransaction,
	getRewardAmount:              (*HttpServer).handleGetRewardAmount,
	getRewardAmountByPublicKey:   (*HttpServer).handleGetRewardAmountByPublicKey,
	listRewardAmount:             (*HttpServer).handleListRewardAmount,

	// mining info
	getMiningInfo:               (*HttpServer).handleGetMiningInfo,
	enableMining:                (*HttpServer).handleEnableMining,
	getChainMiningStatus:        (*HttpServer).handleGetChainMiningStatus,
	getPublickeyMining:          (*HttpServer).handleGetPublicKeyMining,
	getPublicKeyRole:            (*HttpServer).handleGetPublicKeyRole,
	getRoleByValidatorKey:       (*HttpServer).handleGetValidatorKeyRole,
	getIncognitoPublicKeyRole:   (*HttpServer).handleGetIncognitoPublicKeyRole,
	getMinerRewardFromMiningKey: (*HttpServer).handleGetMinerRewardFromMiningKey,
	getProducersBlackList:       (*HttpServer).handleGetProducersBlackList,
	getProducersBlackListDetail: (*HttpServer).handleGetProducersBlackListDetail,

	// pde
	getPDEState:                                (*HttpServer).handleGetPDEState,
	createAndSendTxWithWithdrawalReq:           (*HttpServer).handleCreateAndSendTxWithWithdrawalReq,
	createAndSendTxWithWithdrawalReqV2:         (*HttpServer).handleCreateAndSendTxWithWithdrawalReqV2,
	createAndSendTxWithPDEFeeWithdrawalReq:     (*HttpServer).handleCreateAndSendTxWithPDEFeeWithdrawalReq,
	createAndSendTxWithPTokenTradeReq:          (*HttpServer).handleCreateAndSendTxWithPTokenTradeReq,
	createAndSendTxWithPTokenCrossPoolTradeReq: (*HttpServer).handleCreateAndSendTxWithPTokenCrossPoolTradeReq,
	createAndSendTxWithPRVTradeReq:             (*HttpServer).handleCreateAndSendTxWithPRVTradeReq,
	createAndSendTxWithPRVCrossPoolTradeReq:    (*HttpServer).handleCreateAndSendTxWithPRVCrossPoolTradeReq,
	createAndSendTxWithPTokenContribution:      (*HttpServer).handleCreateAndSendTxWithPTokenContribution,
	createAndSendTxWithPRVContribution:         (*HttpServer).handleCreateAndSendTxWithPRVContribution,
	createAndSendTxWithPTokenContributionV2:    (*HttpServer).handleCreateAndSendTxWithPTokenContributionV2,
	createAndSendTxWithPRVContributionV2:       (*HttpServer).handleCreateAndSendTxWithPRVContributionV2,
	getPDEContributionStatus:                   (*HttpServer).handleGetPDEContributionStatus,
	getPDEContributionStatusV2:                 (*HttpServer).handleGetPDEContributionStatusV2,
	getPDETradeStatus:                          (*HttpServer).handleGetPDETradeStatus,
	getPDEWithdrawalStatus:                     (*HttpServer).handleGetPDEWithdrawalStatus,
	getPDEFeeWithdrawalStatus:                  (*HttpServer).handleGetPDEFeeWithdrawalStatus,
	convertPDEPrices:                           (*HttpServer).handleConvertPDEPrices,
	extractPDEInstsFromBeaconBlock:             (*HttpServer).handleExtractPDEInstsFromBeaconBlock,

	getBurningAddress: (*HttpServer).handleGetBurningAddress,

	// portal
	getPortalState:                                (*HttpServer).handleGetPortalState,
	createAndSendTxWithCustodianDeposit:           (*HttpServer).handleCreateAndSendTxWithCustodianDeposit,
	getPortalCustodianDepositStatus:               (*HttpServer).handleGetPortalCustodianDepositStatus,
	createAndSendRegisterPortingPublicTokens:      (*HttpServer).handleCreateAndSendTxPortingRequest,
	createAndSendTxWithReqPToken:                  (*HttpServer).handleCreateAndSendTxWithReqPToken,
	createAndSendTxWithReqPTokenV4:                (*HttpServer).handleCreateAndSendTxWithReqPTokenV4,
	createAndSendPortalExchangeRates:              (*HttpServer).handleCreateAndSendTxWithPortalExchangeRate,
	getPortalFinalExchangeRates:                   (*HttpServer).handleGetPortalFinalExchangeRates,
	getPortalPortingRequestByKey:                  (*HttpServer).handleGetPortingRequestStatusByTxID,
	getPortalPortingRequestByPortingId:            (*HttpServer).handleGetPortingRequestStatusByPortingId,
	convertExchangeRates:                          (*HttpServer).handleConvertExchangeRates,
	getPortalReqPTokenStatus:                      (*HttpServer).handleGetPortalReqPTokenStatus,
	getPortingRequestFees:                         (*HttpServer).handleGetPortingRequestFees,
	createAndSendTxWithRedeemReq:                  (*HttpServer).handleCreateAndSendTxWithRedeemReq,
	createAndSendTxWithReqUnlockCollateral:        (*HttpServer).handleCreateAndSendTxWithReqUnlockCollateral,
	getPortalReqUnlockCollateralStatus:            (*HttpServer).handleGetPortalReqUnlockCollateralStatus,
	getPortalReqRedeemStatus:                      (*HttpServer).handleGetReqRedeemStatusByRedeemID,
	createAndSendCustodianWithdrawRequest:         (*HttpServer).handleCreateAndSendTxWithCustodianWithdrawRequest,
	getCustodianWithdrawByTxId:                    (*HttpServer).handleGetCustodianWithdrawRequestStatusByTxId,
	getCustodianLiquidationStatus:                 (*HttpServer).handleGetCustodianLiquidationStatus,
	createAndSendTxWithReqWithdrawRewardPortal:    (*HttpServer).handleCreateAndSendTxWithReqWithdrawRewardPortal,
	getLiquidationExchangeRatesPool:               (*HttpServer).handleGetLiquidationExchangeRatesPool,
	createAndSendTxRedeemFromLiquidationPoolV3:    (*HttpServer).handleCreateAndSendTxRedeemFromLiquidationPoolV3,
	createAndSendCustodianTopup:                   (*HttpServer).handleCreateAndSendCustodianTopup,
	createAndSendTopUpWaitingPorting:              (*HttpServer).handleCreateAndSendTopUpWaitingPorting,
	createAndSendCustodianTopupV3:                 (*HttpServer).handleCreateAndSendCustodianTopupV3,
	createAndSendTopUpWaitingPortingV3:            (*HttpServer).handleCreateAndSendTopUpWaitingPortingV3,
	getTopupAmountForCustodian:                    (*HttpServer).handleGetTopupAmountForCustodianState,
	getPortalReward:                               (*HttpServer).handleGetPortalReward,
	getRequestWithdrawPortalRewardStatus:          (*HttpServer).handleGetRequestWithdrawPortalRewardStatus,
	createAndSendTxWithReqMatchingRedeem:          (*HttpServer).handleCreateAndSendTxWithReqMatchingRedeem,
	getReqMatchingRedeemStatus:                    (*HttpServer).handleGetReqMatchingRedeemStatusByTxID,
	getPortalCustodianTopupStatus:                 (*HttpServer).handleGetPortalCustodianTopupStatus,
	getPortalCustodianTopupStatusV3:               (*HttpServer).handleGetPortalCustodianTopupStatusV3,
	getPortalCustodianTopupWaitingPortingStatus:   (*HttpServer).handleGetPortalCustodianTopupWaitingPortingStatus,
	getPortalCustodianTopupWaitingPortingStatusV3: (*HttpServer).handleGetPortalCustodianTopupWaitingPortingStatusV3,
	getAmountTopUpWaitingPorting:                  (*HttpServer).handleGetAmountTopUpWaitingPorting,
	getPortalReqRedeemByTxIDStatus:                (*HttpServer).handleGetReqRedeemStatusByTxID,
	getReqRedeemFromLiquidationPoolByTxIDStatus:   (*HttpServer).handleGetReqRedeemFromLiquidationPoolByTxIDStatus,
	getReqRedeemFromLiquidationPoolByTxIDStatusV3: (*HttpServer).handleGetReqRedeemFromLiquidationPoolByTxIDStatusV3,
	createAndSendTxWithCustodianDepositV3:         (*HttpServer).handleCreateAndSendTxWithCustodianDepositV3,
	getPortalCustodianDepositStatusV3:             (*HttpServer).handleGetPortalCustodianDepositStatusV3,
	checkPortalExternalHashSubmitted:              (*HttpServer).handleCheckPortalExternalHashSubmitted,
	createAndSendTxWithCustodianWithdrawRequestV3: (*HttpServer).handleCreateAndSendTxWithCustodianWithdrawRequestV3,
	getCustodianWithdrawRequestStatusV3ByTxId:     (*HttpServer).handleGetCustodianWithdrawRequestStatusV3ByTxId,
	getPortalWithdrawCollateralProof:              (*HttpServer).handleGetPortalWithdrawCollateralProof,
	createAndSendUnlockOverRateCollaterals:        (*HttpServer).handleCreateAndSendTxWithPortalCusUnlockOverRateCollaterals,
	getPortalUnlockOverRateCollateralsStatus:      (*HttpServer).handleGetPortalReqUnlockOverRateCollateralStatus,

	// relaying
	createAndSendTxWithRelayingBNBHeader: (*HttpServer).handleCreateAndSendTxWithRelayingBNBHeader,
	createAndSendTxWithRelayingBTCHeader: (*HttpServer).handleCreateAndSendTxWithRelayingBTCHeader,
	getRelayingBNBHeaderState:            (*HttpServer).handleGetRelayingBNBHeaderState,
	getRelayingBNBHeaderByBlockHeight:    (*HttpServer).handleGetRelayingBNBHeaderByBlockHeight,
	getBTCRelayingBestState:              (*HttpServer).handleGetBTCRelayingBestState,
	getBTCBlockByHash:                    (*HttpServer).handleGetBTCBlockByHash,
	getLatestBNBHeaderBlockHeight:        (*HttpServer).handleGetLatestBNBHeaderBlockHeight,

	// incognnito mode for sc
	getBurnProofForDepositToSC:                  (*HttpServer).handleGetBurnProofForDepositToSC,
	createAndSendBurningForDepositToSCRequest:   (*HttpServer).handleCreateAndSendBurningForDepositToSCRequest,
	createAndSendBurningForDepositToSCRequestV2: (*HttpServer).handleCreateAndSendBurningForDepositToSCRequestV2,

	//new pool info
	getBeaconPoolInfo:     (*HttpServer).hanldeGetBeaconPoolInfo,
	getShardPoolInfo:      (*HttpServer).hanldeGetShardPoolInfo,
	getCrossShardPoolInfo: (*HttpServer).hanldeGetCrossShardPoolInfo,
	getAllView:            (*HttpServer).hanldeGetAllView,
	getAllViewDetail:      (*HttpServer).hanldeGetAllViewDetail,

	// feature reward
	getRewardFeature: (*HttpServer).handleGetRewardFeature,

	// get committeeByHeight

	getTotalStaker: (*HttpServer).handleGetTotalStaker,

	//validators state
	getValKeyState: (*HttpServer).handleGetValKeyState,
}

// Commands that are available to a limited user
var LimitedHttpHandler = map[string]httpHandler{
	// local WALLET
	listAccounts:                     (*HttpServer).handleListAccounts,
	getAccount:                       (*HttpServer).handleGetAccount,
	getAddressesByAccount:            (*HttpServer).handleGetAddressesByAccount,
	getAccountAddress:                (*HttpServer).handleGetAccountAddress,
	dumpPrivkey:                      (*HttpServer).handleDumpPrivkey,
	importAccount:                    (*HttpServer).handleImportAccount,
	removeAccount:                    (*HttpServer).handleRemoveAccount,
	listUnspentOutputCoins:           (*HttpServer).handleListUnspentOutputCoins,
	getBalance:                       (*HttpServer).handleGetBalance,
	getBalanceByPrivatekey:           (*HttpServer).handleGetBalanceByPrivatekey,
	getBalanceByPaymentAddress:       (*HttpServer).handleGetBalanceByPaymentAddress,
	getReceivedByAccount:             (*HttpServer).handleGetReceivedByAccount,
	setTxFee:                         (*HttpServer).handleSetTxFee,
	convertNativeTokenToPrivacyToken: (*HttpServer).handleConvertNativeTokenToPrivacyToken,
	convertPrivacyTokenToNativeToken: (*HttpServer).handleConvertPrivacyTokenToNativeToken,
}

var WsHandler = map[string]wsHandler{
	testSubcrice:                                (*WsServer).handleTestSubcribe,
	subcribeNewShardBlock:                       (*WsServer).handleSubscribeNewShardBlock,
	subcribeNewBeaconBlock:                      (*WsServer).handleSubscribeNewBeaconBlock,
	subcribePendingTransaction:                  (*WsServer).handleSubscribePendingTransaction,
	subcribeShardCandidateByPublickey:           (*WsServer).handleSubcribeShardCandidateByPublickey,
	subcribeShardCommitteeByPublickey:           (*WsServer).handleSubcribeShardCommitteeByPublickey,
	subcribeShardPendingValidatorByPublickey:    (*WsServer).handleSubcribeShardPendingValidatorByPublickey,
	subcribeBeaconCandidateByPublickey:          (*WsServer).handleSubcribeBeaconCandidateByPublickey,
	subcribeBeaconPendingValidatorByPublickey:   (*WsServer).handleSubcribeBeaconPendingValidatorByPublickey,
	subcribeBeaconCommitteeByPublickey:          (*WsServer).handleSubcribeBeaconCommitteeByPublickey,
	subcribeMempoolInfo:                         (*WsServer).handleSubcribeMempoolInfo,
	subcribeCrossOutputCoinByPrivateKey:         (*WsServer).handleSubcribeCrossOutputCoinByPrivateKey,
	subcribeCrossCustomTokenPrivacyByPrivateKey: (*WsServer).handleSubcribeCrossCustomTokenPrivacyByPrivateKey,
	subcribeShardBestState:                      (*WsServer).handleSubscribeShardBestState,
	subcribeBeaconBestState:                     (*WsServer).handleSubscribeBeaconBestState,
	subcribeBeaconPoolBeststate:                 (*WsServer).handleSubscribeBeaconPoolBestState,
	subcribeShardPoolBeststate:                  (*WsServer).handleSubscribeShardPoolBeststate,
}
