package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"github.com/binance-chain/go-sdk/types/msg"
	"github.com/incognitochain/incognito-chain/relaying/bnb"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"

	//"github.com/binance-chain/go-sdk/types/msg"
	//"github.com/incognitochain/incognito-chain/relaying/bnb"
	//btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"strconv"
)

// beacon build new instruction from instruction received from Shard Block
func buildCustodianDepositInst(
	custodianAddressStr string,
	depositedAmount uint64,
	remoteAddresses map[string]string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	custodianDepositContent := metadata.PortalCustodianDepositContent{
		IncogAddressStr: custodianAddressStr,
		RemoteAddresses: remoteAddresses,
		DepositedAmount: depositedAmount,
		TxReqID:         txReqID,
		ShardID:         shardID,
	}
	custodianDepositContentBytes, _ := json.Marshal(custodianDepositContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(custodianDepositContentBytes),
	}
}

func buildRequestPortingInst(
	metaType int,
	shardID byte,
	reqStatus string,
	uniqueRegisterId string,
	incogAddressStr string,
	pTokenId string,
	registerAmount uint64,
	portingFee uint64,
	custodian []*statedb.MatchingPortingCustodianDetail,
	txReqID common.Hash,
) []string {
	portingRequestContent := metadata.PortalPortingRequestContent{
		UniqueRegisterId: uniqueRegisterId,
		IncogAddressStr:  incogAddressStr,
		PTokenId:         pTokenId,
		RegisterAmount:   registerAmount,
		PortingFee:       portingFee,
		Custodian:        custodian,
		TxReqID:          txReqID,
		ShardID:          shardID,
	}

	portingRequestContentBytes, _ := json.Marshal(portingRequestContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		reqStatus,
		string(portingRequestContentBytes),
	}
}

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildReqPTokensInst(
	uniquePortingID string,
	tokenID string,
	incogAddressStr string,
	portingAmount uint64,
	portingProof string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	reqPTokenContent := metadata.PortalRequestPTokensContent{
		UniquePortingID: uniquePortingID,
		TokenID:         tokenID,
		IncogAddressStr: incogAddressStr,
		PortingAmount:   portingAmount,
		PortingProof:    portingProof,
		TxReqID:         txReqID,
		ShardID:         shardID,
	}
	reqPTokenContentBytes, _ := json.Marshal(reqPTokenContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(reqPTokenContentBytes),
	}
}

func buildCustodianWithdrawInst(
	metaType int,
	shardID byte,
	reqStatus string,
	paymentAddress string,
	amount uint64,
	remainFreeCollateral uint64,
	txReqID common.Hash,
) []string {
	content := metadata.PortalCustodianWithdrawRequestContent{
		PaymentAddress:       paymentAddress,
		Amount:               amount,
		RemainFreeCollateral: remainFreeCollateral,
		TxReqID:              txReqID,
		ShardID:              shardID,
	}

	contentBytes, _ := json.Marshal(content)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		reqStatus,
		string(contentBytes),
	}
}

// buildInstructionsForCustodianDeposit builds instruction for custodian deposit action
func (blockchain *BlockChain) buildInstructionsForCustodianDeposit(
	contentStr string,
	shardID byte,
	metaType int,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	portalParams PortalParams,
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal custodian deposit action: %+v", err)
		return [][]string{}, nil
	}
	var actionData metadata.PortalCustodianDepositAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal custodian deposit action: %+v", err)
		return [][]string{}, nil
	}

	if currentPortalState == nil {
		Logger.log.Errorf("WARN - [buildInstructionsForCustodianDeposit]: Current Portal state is null.")
		// need to refund collateral to custodian
		inst := buildCustodianDepositInst(
			actionData.Meta.IncogAddressStr,
			actionData.Meta.DepositedAmount,
			actionData.Meta.RemoteAddresses,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalCustodianDepositRefundChainStatus,
		)
		return [][]string{inst}, nil
	}
	meta := actionData.Meta

	keyCustodianState := statedb.GenerateCustodianStateObjectKey(meta.IncogAddressStr)
	keyCustodianStateStr := keyCustodianState.String()

	newCustodian := new(statedb.CustodianState)
	if currentPortalState.CustodianPoolState[keyCustodianStateStr] == nil {
		// new custodian
		newCustodian = statedb.NewCustodianStateWithValue(
			meta.IncogAddressStr, meta.DepositedAmount, meta.DepositedAmount,
			nil, nil,
			meta.RemoteAddresses, nil)
	} else {
		// custodian deposited before
		custodian := currentPortalState.CustodianPoolState[keyCustodianStateStr]
		totalCollateral := custodian.GetTotalCollateral() + meta.DepositedAmount
		freeCollateral := custodian.GetFreeCollateral() + meta.DepositedAmount
		holdingPubTokens := custodian.GetHoldingPublicTokens()
		lockedAmountCollateral := custodian.GetLockedAmountCollateral()
		rewardAmount := custodian.GetRewardAmount()
		remoteAddresses := custodian.GetRemoteAddresses()
		// if total collateral is zero, custodians are able to update remote addresses
		if custodian.GetTotalCollateral() == 0 {
			if len(meta.RemoteAddresses) > 0 {
				remoteAddresses = meta.RemoteAddresses
			}
		} else {
			sortedTokenIDs := make([]string, 0)
			for tokenID := range meta.RemoteAddresses {
				sortedTokenIDs = append(sortedTokenIDs, tokenID)
			}

			for _, tokenID := range sortedTokenIDs {
				if remoteAddresses[tokenID] == "" {
					remoteAddresses[tokenID] = meta.RemoteAddresses[tokenID]
				}
			}
		}
		newCustodian = statedb.NewCustodianStateWithValue(meta.IncogAddressStr, totalCollateral, freeCollateral,
			holdingPubTokens, lockedAmountCollateral, remoteAddresses, rewardAmount)
	}
	// update state of the custodian
	currentPortalState.CustodianPoolState[keyCustodianStateStr] = newCustodian

	inst := buildCustodianDepositInst(
		actionData.Meta.IncogAddressStr,
		actionData.Meta.DepositedAmount,
		newCustodian.GetRemoteAddresses(),
		actionData.Meta.Type,
		shardID,
		actionData.TxReqID,
		common.PortalCustodianDepositAcceptedChainStatus,
	)
	return [][]string{inst}, nil
}

func (blockchain *BlockChain) buildInstructionsForPortingRequest(
	stateDB *statedb.StateDB,
	contentStr string,
	shardID byte,
	metaType int,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	portalParams PortalParams,
) ([][]string, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while decoding content string of portal porting request action: %+v", err)
		return [][]string{}, nil
	}

	var actionData metadata.PortalUserRegisterAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("Porting request: an error occurred while unmarshal portal porting request action: %+v", err)
		return [][]string{}, nil
	}

	if currentPortalState == nil {
		Logger.log.Warn("Porting request: Current Portal state is null")
		return [][]string{}, nil
	}

	//check unique id from record from db
	portingRequestKeyExist, err := statedb.IsPortingRequestIdExist(stateDB, []byte(actionData.Meta.UniqueRegisterId))
	if err != nil {
		Logger.log.Errorf("Porting request: Get item portal by prefix error: %+v", err)
		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			nil,
			actionData.TxReqID,
		)

		return [][]string{inst}, nil
	}

	if portingRequestKeyExist {
		Logger.log.Errorf("Porting request: Porting request id exist, key %v", actionData.Meta.UniqueRegisterId)
		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			nil,
			actionData.TxReqID,
		)

		return [][]string{inst}, nil
	}

	waitingPortingRequestKey := statedb.GeneratePortalWaitingPortingRequestObjectKey(actionData.Meta.UniqueRegisterId)
	if _, ok := currentPortalState.WaitingPortingRequests[waitingPortingRequestKey.String()]; ok {
		Logger.log.Errorf("Porting request: Waiting porting request exist, key %v", waitingPortingRequestKey)
		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			nil,
			actionData.TxReqID,
		)

		return [][]string{inst}, nil
	}

	//get exchange rates
	exchangeRatesState := currentPortalState.FinalExchangeRatesState
	if exchangeRatesState == nil {
		Logger.log.Errorf("Porting request, exchange rates not found")
		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			nil,
			actionData.TxReqID,
		)

		return [][]string{inst}, nil
	}

	if len(currentPortalState.CustodianPoolState) <= 0 {
		Logger.log.Errorf("Porting request: Custodian not found")
		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			nil,
			actionData.TxReqID,
		)

		return [][]string{inst}, nil
	}

	var sortCustodianStateByFreeCollateral []CustodianStateSlice
	sortCustodianByAmountAscent(actionData.Meta, currentPortalState.CustodianPoolState, &sortCustodianStateByFreeCollateral)

	if len(sortCustodianStateByFreeCollateral) <= 0 {
		Logger.log.Errorf("Porting request, custodian not found")

		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			nil,
			actionData.TxReqID,
		)

		return [][]string{inst}, nil
	}

	//validation porting fees
	exchangePortingFees, err := CalMinPortingFee(actionData.Meta.RegisterAmount, actionData.Meta.PTokenId, exchangeRatesState, portalParams.MinPercentPortingFee)
	if err != nil {
		Logger.log.Errorf("Calculate Porting fee is error %v", err)
		return [][]string{}, nil
	}

	Logger.log.Infof("Porting request, porting fees need %v", exchangePortingFees)

	if actionData.Meta.PortingFee < exchangePortingFees {
		Logger.log.Errorf("Porting request, Porting fees is wrong")

		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			nil,
			actionData.TxReqID,
		)
		return [][]string{inst}, nil
	}

	// pick-up custodians
	pickedCustodians, err := pickUpCustodians(actionData.Meta, exchangeRatesState, sortCustodianStateByFreeCollateral, currentPortalState, portalParams)
	if err != nil || len(pickedCustodians) == 0 {
		Logger.log.Errorf("Porting request: an error occurred while picking up custodians for the porting request: %+v", err)

		inst := buildRequestPortingInst(
			actionData.Meta.Type,
			shardID,
			common.PortalPortingRequestRejectedChainStatus,
			actionData.Meta.UniqueRegisterId,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PTokenId,
			actionData.Meta.RegisterAmount,
			actionData.Meta.PortingFee,
			pickedCustodians,
			actionData.TxReqID,
		)
		return [][]string{inst}, nil
	}

	// Update custodian state after finishing choosing enough custodians for the porting request
	for _, cus := range pickedCustodians {
		cusKey := statedb.GenerateCustodianStateObjectKey(cus.IncAddress).String()
		//update custodian state
		err := UpdateCustodianStateAfterMatchingPortingRequest(currentPortalState, cusKey, actionData.Meta.PTokenId, cus.LockedAmountCollateral)
		if err != nil {
			return nil, err
		}
	}

	inst := buildRequestPortingInst(
		actionData.Meta.Type,
		shardID,
		common.PortalPortingRequestAcceptedChainStatus,
		actionData.Meta.UniqueRegisterId,
		actionData.Meta.IncogAddressStr,
		actionData.Meta.PTokenId,
		actionData.Meta.RegisterAmount,
		actionData.Meta.PortingFee,
		pickedCustodians,
		actionData.TxReqID,
	)

	newPortingRequestStateWaiting := statedb.NewWaitingPortingRequestWithValue(
		actionData.Meta.UniqueRegisterId,
		actionData.TxReqID,
		actionData.Meta.PTokenId,
		actionData.Meta.IncogAddressStr,
		actionData.Meta.RegisterAmount,
		pickedCustodians,
		actionData.Meta.PortingFee,
		beaconHeight+1,
	)

	keyWaitingPortingRequest := statedb.GeneratePortalWaitingPortingRequestObjectKey(actionData.Meta.UniqueRegisterId)
	currentPortalState.WaitingPortingRequests[keyWaitingPortingRequest.String()] = newPortingRequestStateWaiting

	return [][]string{inst}, nil
}

// buildInstructionsForCustodianDeposit builds instruction for custodian deposit action
func (blockchain *BlockChain) buildInstructionsForReqPTokens(
	stateDB *statedb.StateDB,
	contentStr string,
	shardID byte,
	metaType int,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	portalParams PortalParams,
) ([][]string, error) {

	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal custodian deposit action: %+v", err)
		return [][]string{}, nil
	}
	var actionData metadata.PortalRequestPTokensAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal custodian deposit action: %+v", err)
		return [][]string{}, nil
	}
	meta := actionData.Meta

	if currentPortalState == nil {
		Logger.log.Warn("WARN - [buildInstructionsForCustodianDeposit]: Current Portal state is null.")
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	// check meta.UniquePortingID is in waiting PortingRequests list in portal state or not
	portingID := meta.UniquePortingID
	keyWaitingPortingRequest := statedb.GeneratePortalWaitingPortingRequestObjectKey(portingID)
	keyWaitingPortingRequestStr := keyWaitingPortingRequest.String()
	waitingPortingRequest := currentPortalState.WaitingPortingRequests[keyWaitingPortingRequestStr]
	if waitingPortingRequest == nil {
		Logger.log.Errorf("PortingID is not existed in waiting porting requests list")
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	//check unique id from record from db
	portingRequest, err := statedb.GetPortalStateStatusMultiple(stateDB, statedb.PortalPortingRequestStatusPrefix(), []byte(meta.UniquePortingID))

	if err != nil {
		Logger.log.Errorf("Can not get porting req status for portingID %v, %v\n", meta.UniquePortingID, err)
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	var portingRequestStatus metadata.PortingRequestStatus
	err = json.Unmarshal(portingRequest, &portingRequestStatus)
	if err != nil {
		Logger.log.Errorf("Has an error occurred while unmarshal PortingRequestStatus: %+v", err)
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	if portingRequestStatus.Status != common.PortalPortingReqWaitingStatus {
		Logger.log.Errorf("PortingID status invalid, expected %v , but got %v\n", common.PortalPortingReqWaitingStatus, portingRequestStatus.Status)
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	// check tokenID
	if meta.TokenID != waitingPortingRequest.TokenID() {
		Logger.log.Errorf("TokenID is not correct in portingID req")
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	// check porting amount
	if meta.PortingAmount != waitingPortingRequest.Amount() {
		Logger.log.Errorf("PortingAmount is not correct in portingID req")
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	if meta.TokenID == common.PortalBTCIDStr {
		btcChain := blockchain.config.BTCChain
		if btcChain == nil {
			Logger.log.Error("BTC relaying chain should not be null")
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}
		// parse PortingProof in meta
		btcTxProof, err := btcrelaying.ParseBTCProofFromB64EncodeStr(meta.PortingProof)
		if err != nil {
			Logger.log.Errorf("PortingProof is invalid %v\n", err)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		isValid, err := btcChain.VerifyTxWithMerkleProofs(btcTxProof)
		if !isValid || err != nil {
			Logger.log.Errorf("Verify btcTxProof failed %v", err)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		// extract attached message from txOut's OP_RETURN
		btcAttachedMsg, err := btcrelaying.ExtractAttachedMsgFromTx(btcTxProof.BTCTx)
		if err != nil {
			Logger.log.Errorf("Could not extract attached message from BTC tx proof with err: %v", err)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		encodedMsg := btcrelaying.HashAndEncodeBase58(meta.UniquePortingID)
		if btcAttachedMsg != encodedMsg {
			Logger.log.Errorf("PortingId in the btc attached message is not matched with portingID in metadata")
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		// check whether amount transfer in txBNB is equal porting amount or not
		// check receiver and amount in tx
		// get list matching custodians in waitingPortingRequest
		custodians := waitingPortingRequest.Custodians()
		outputs := btcTxProof.BTCTx.TxOut
		for _, cusDetail := range custodians {
			remoteAddressNeedToBeTransfer := cusDetail.RemoteAddress
			amountNeedToBeTransfer := cusDetail.Amount
			amountNeedToBeTransferInBTC := btcrelaying.ConvertIncPBTCAmountToExternalBTCAmount(int64(amountNeedToBeTransfer))

			isChecked := false
			for _, out := range outputs {
				addrStr, err := btcChain.ExtractPaymentAddrStrFromPkScript(out.PkScript)
				if err != nil {
					Logger.log.Errorf("[portal] ExtractPaymentAddrStrFromPkScript: could not extract payment address string from pkscript with err: %v\n", err)
					continue
				}
				if addrStr != remoteAddressNeedToBeTransfer {
					continue
				}
				if out.Value < amountNeedToBeTransferInBTC {
					Logger.log.Errorf("BTC-TxProof is invalid - the transferred amount to %s must be equal to or greater than %d, but got %d", addrStr, amountNeedToBeTransferInBTC, out.Value)
					inst := buildReqPTokensInst(
						meta.UniquePortingID,
						meta.TokenID,
						meta.IncogAddressStr,
						meta.PortingAmount,
						meta.PortingProof,
						meta.Type,
						shardID,
						actionData.TxReqID,
						common.PortalReqPTokensRejectedChainStatus,
					)
					return [][]string{inst}, nil
				} else {
					isChecked = true
					break
				}
			}
			if !isChecked {
				Logger.log.Error("BTC-TxProof is invalid")
				inst := buildReqPTokensInst(
					meta.UniquePortingID,
					meta.TokenID,
					meta.IncogAddressStr,
					meta.PortingAmount,
					meta.PortingProof,
					meta.Type,
					shardID,
					actionData.TxReqID,
					common.PortalReqPTokensRejectedChainStatus,
				)
				return [][]string{inst}, nil
			}
		}
		// update holding public token for custodians
		for _, cusDetail := range custodians {
			custodianKey := statedb.GenerateCustodianStateObjectKey(cusDetail.IncAddress)
			UpdateCustodianStateAfterUserRequestPToken(currentPortalState, custodianKey.String(), waitingPortingRequest.TokenID(), cusDetail.Amount)
		}

		inst := buildReqPTokensInst(
			actionData.Meta.UniquePortingID,
			actionData.Meta.TokenID,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PortingAmount,
			actionData.Meta.PortingProof,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensAcceptedChainStatus,
		)

		// remove waiting porting request from currentPortalState
		deleteWaitingPortingRequest(currentPortalState, keyWaitingPortingRequestStr)
		return [][]string{inst}, nil

	} else if meta.TokenID == common.PortalBNBIDStr {
		// parse PortingProof in meta
		txProofBNB, err := bnb.ParseBNBProofFromB64EncodeStr(meta.PortingProof)
		if err != nil {
			Logger.log.Errorf("PortingProof is invalid %v\n", err)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		// check minimum confirmations block of bnb proof
		latestBNBBlockHeight, err2 := blockchain.GetLatestBNBBlkHeight()
		if err2 != nil {
			Logger.log.Errorf("Can not get latest relaying bnb block height %v\n", err)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		if latestBNBBlockHeight < txProofBNB.BlockHeight+bnb.MinConfirmationsBlock {
			Logger.log.Errorf("Not enough min bnb confirmations block %v, latestBNBBlockHeight %v - txProofBNB.BlockHeight %v\n",
				bnb.MinConfirmationsBlock, latestBNBBlockHeight, txProofBNB.BlockHeight)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}
		dataHash, err2 := blockchain.GetBNBDataHash(txProofBNB.BlockHeight)
		if err2 != nil {
			Logger.log.Errorf("Error when get data hash in blockHeight %v - %v\n",
				txProofBNB.BlockHeight, err2)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		isValid, err := txProofBNB.Verify(dataHash)
		if !isValid || err != nil {
			Logger.log.Errorf("Verify txProofBNB failed %v", err)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		// parse Tx from Data in txProofBNB
		txBNB, err := bnb.ParseTxFromData(txProofBNB.Proof.Data)
		if err != nil {
			Logger.log.Errorf("Data in PortingProof is invalid %v", err)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		// check memo attach portingID req:
		memo := txBNB.Memo
		memoBytes, err2 := base64.StdEncoding.DecodeString(memo)
		if err2 != nil {
			Logger.log.Errorf("Can not decode memo in tx bnb proof", err2)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		var portingMemo PortingMemoBNB
		err2 = json.Unmarshal(memoBytes, &portingMemo)
		if err2 != nil {
			Logger.log.Errorf("Can not unmarshal memo in tx bnb proof", err2)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		if portingMemo.PortingID != meta.UniquePortingID {
			Logger.log.Errorf("PortingId in memoTx is not matched with portingID in metadata", err2)
			inst := buildReqPTokensInst(
				meta.UniquePortingID,
				meta.TokenID,
				meta.IncogAddressStr,
				meta.PortingAmount,
				meta.PortingProof,
				meta.Type,
				shardID,
				actionData.TxReqID,
				common.PortalReqPTokensRejectedChainStatus,
			)
			return [][]string{inst}, nil
		}

		// check whether amount transfer in txBNB is equal porting amount or not
		// check receiver and amount in tx
		// get list matching custodians in waitingPortingRequest
		custodians := waitingPortingRequest.Custodians()
		outputs := txBNB.Msgs[0].(msg.SendMsg).Outputs
		for _, cusDetail := range custodians {
			remoteAddressNeedToBeTransfer := cusDetail.RemoteAddress
			amountNeedToBeTransfer := cusDetail.Amount
			amountNeedToBeTransferInBNB := convertIncPBNBAmountToExternalBNBAmount(int64(amountNeedToBeTransfer))

			isChecked := false
			for _, out := range outputs {
				addr, _ := bnb.GetAccAddressString(&out.Address, blockchain.config.ChainParams.BNBRelayingHeaderChainID)
				if addr != remoteAddressNeedToBeTransfer {
					Logger.log.Warnf("[portal] remoteAddressNeedToBeTransfer: %v - addr: %v\n", remoteAddressNeedToBeTransfer, addr)
					continue
				}

				// calculate amount that was transferred to custodian's remote address
				amountTransfer := int64(0)
				for _, coin := range out.Coins {
					if coin.Denom == bnb.DenomBNB {
						amountTransfer += coin.Amount
					}
				}
				if amountTransfer < amountNeedToBeTransferInBNB {
					Logger.log.Errorf("TxProof-BNB is invalid - Amount transfer to %s must be equal to or greater than %d, but got %d",
						addr, amountNeedToBeTransferInBNB, amountTransfer)
					inst := buildReqPTokensInst(
						meta.UniquePortingID,
						meta.TokenID,
						meta.IncogAddressStr,
						meta.PortingAmount,
						meta.PortingProof,
						meta.Type,
						shardID,
						actionData.TxReqID,
						common.PortalReqPTokensRejectedChainStatus,
					)
					return [][]string{inst}, nil
				} else {
					isChecked = true
					break
				}
			}
			if !isChecked {
				Logger.log.Errorf("TxProof-BNB is invalid - Receiver address is invalid, expected %v",
					remoteAddressNeedToBeTransfer)
				inst := buildReqPTokensInst(
					meta.UniquePortingID,
					meta.TokenID,
					meta.IncogAddressStr,
					meta.PortingAmount,
					meta.PortingProof,
					meta.Type,
					shardID,
					actionData.TxReqID,
					common.PortalReqPTokensRejectedChainStatus,
				)
				return [][]string{inst}, nil
			}
		}

		// update holding public token for custodians
		for _, cusDetail := range custodians {
			custodianKey := statedb.GenerateCustodianStateObjectKey(cusDetail.IncAddress)
			UpdateCustodianStateAfterUserRequestPToken(currentPortalState, custodianKey.String(), waitingPortingRequest.TokenID(), cusDetail.Amount)
		}

		inst := buildReqPTokensInst(
			actionData.Meta.UniquePortingID,
			actionData.Meta.TokenID,
			actionData.Meta.IncogAddressStr,
			actionData.Meta.PortingAmount,
			actionData.Meta.PortingProof,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensAcceptedChainStatus,
		)

		// remove waiting porting request from currentPortalState
		deleteWaitingPortingRequest(currentPortalState, keyWaitingPortingRequestStr)
		return [][]string{inst}, nil
	} else {
		Logger.log.Errorf("TokenID is not supported currently on Portal")
		inst := buildReqPTokensInst(
			meta.UniquePortingID,
			meta.TokenID,
			meta.IncogAddressStr,
			meta.PortingAmount,
			meta.PortingProof,
			meta.Type,
			shardID,
			actionData.TxReqID,
			common.PortalReqPTokensRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}
}

func (blockchain *BlockChain) buildInstructionsForExchangeRates(
	contentStr string,
	shardID byte,
	metaType int,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	portalParams PortalParams,
) ([][]string, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occurred while decoding content string of portal exchange rates action: %+v", err)
		return [][]string{}, nil
	}

	var actionData metadata.PortalExchangeRatesAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occurred while unmarshal portal exchange rates action: %+v", err)
		return [][]string{}, nil
	}

	//check key from db
	if currentPortalState.ExchangeRatesRequests != nil {
		_, ok := currentPortalState.ExchangeRatesRequests[actionData.TxReqID.String()]
		if ok {
			Logger.log.Errorf("ERROR: exchange rates key is duplicated")

			portalExchangeRatesContent := metadata.PortalExchangeRatesContent{
				SenderAddress: actionData.Meta.SenderAddress,
				Rates:         actionData.Meta.Rates,
				TxReqID:       actionData.TxReqID,
				LockTime:      actionData.LockTime,
			}

			portalExchangeRatesContentBytes, _ := json.Marshal(portalExchangeRatesContent)

			inst := []string{
				strconv.Itoa(metaType),
				strconv.Itoa(int(shardID)),
				common.PortalExchangeRatesRejectedChainStatus,
				string(portalExchangeRatesContentBytes),
			}

			return [][]string{inst}, nil
		}
	}

	//success
	portalExchangeRatesContent := metadata.PortalExchangeRatesContent{
		SenderAddress: actionData.Meta.SenderAddress,
		Rates:         actionData.Meta.Rates,
		TxReqID:       actionData.TxReqID,
		LockTime:      actionData.LockTime,
	}

	portalExchangeRatesContentBytes, _ := json.Marshal(portalExchangeRatesContent)

	inst := []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		common.PortalExchangeRatesAcceptedChainStatus,
		string(portalExchangeRatesContentBytes),
	}

	//update E-R request
	if currentPortalState.ExchangeRatesRequests != nil {
		currentPortalState.ExchangeRatesRequests[actionData.TxReqID.String()] = metadata.NewExchangeRatesRequestStatus(
			common.PortalExchangeRatesAcceptedStatus,
			actionData.Meta.SenderAddress,
			actionData.Meta.Rates,
		)
	} else {
		//new object
		newExchangeRatesRequest := make(map[string]*metadata.ExchangeRatesRequestStatus)
		newExchangeRatesRequest[actionData.TxReqID.String()] = metadata.NewExchangeRatesRequestStatus(
			common.PortalExchangeRatesAcceptedStatus,
			actionData.Meta.SenderAddress,
			actionData.Meta.Rates,
		)

		currentPortalState.ExchangeRatesRequests = newExchangeRatesRequest
	}

	return [][]string{inst}, nil
}

/**
Validation:
	- verify each instruct belong shard
	- check amount < fee collateral
	- build PortalCustodianWithdrawRequestContent to send beacon
*/
func (blockchain *BlockChain) buildInstructionsForCustodianWithdraw(
	contentStr string,
	shardID byte,
	metaType int,
	currentPortalState *CurrentPortalState,
	beaconHeight uint64,
	portalParams PortalParams,
) ([][]string, error) {
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("Have an error occurred while decoding content string of custodian withdraw request action: %+v", err)
		return [][]string{}, nil
	}

	var actionData metadata.PortalCustodianWithdrawRequestAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("Have an error occurred while unmarshal  custodian withdraw request action: %+v", err)
		return [][]string{}, nil
	}

	if currentPortalState == nil {
		Logger.log.Warn("Current Portal state is null")
		return [][]string{}, nil
	}

	if len(currentPortalState.CustodianPoolState) <= 0 {
		Logger.log.Errorf("Custodian state is empty")

		inst := buildCustodianWithdrawInst(
			actionData.Meta.Type,
			shardID,
			common.PortalCustodianWithdrawRequestRejectedStatus,
			actionData.Meta.PaymentAddress,
			actionData.Meta.Amount,
			0,
			actionData.TxReqID,
		)
		return [][]string{inst}, nil
	}

	custodianKey := statedb.GenerateCustodianStateObjectKey(actionData.Meta.PaymentAddress)
	custodianKeyStr := custodianKey.String()
	custodian, ok := currentPortalState.CustodianPoolState[custodianKeyStr]

	if !ok {
		Logger.log.Errorf("Custodian not found")

		inst := buildCustodianWithdrawInst(
			actionData.Meta.Type,
			shardID,
			common.PortalCustodianWithdrawRequestRejectedStatus,
			actionData.Meta.PaymentAddress,
			actionData.Meta.Amount,
			0,
			actionData.TxReqID,
		)
		return [][]string{inst}, nil
	}

	if actionData.Meta.Amount > custodian.GetFreeCollateral() {
		Logger.log.Errorf("Free Collateral is not enough PRV")

		inst := buildCustodianWithdrawInst(
			actionData.Meta.Type,
			shardID,
			common.PortalCustodianWithdrawRequestRejectedStatus,
			actionData.Meta.PaymentAddress,
			actionData.Meta.Amount,
			0,
			actionData.TxReqID,
		)
		return [][]string{inst}, nil
	}
	//withdraw
	remainFreeCollateral := custodian.GetFreeCollateral() - actionData.Meta.Amount
	totalFreeCollateral := custodian.GetTotalCollateral() - actionData.Meta.Amount

	inst := buildCustodianWithdrawInst(
		actionData.Meta.Type,
		shardID,
		common.PortalCustodianWithdrawRequestAcceptedStatus,
		actionData.Meta.PaymentAddress,
		actionData.Meta.Amount,
		remainFreeCollateral,
		actionData.TxReqID,
	)

	//update free collateral custodian
	custodian.SetFreeCollateral(remainFreeCollateral)
	custodian.SetTotalCollateral(totalFreeCollateral)
	currentPortalState.CustodianPoolState[custodianKeyStr] = custodian
	return [][]string{inst}, nil
}
