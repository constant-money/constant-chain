package portalprocess

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/portal"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	portalMeta "github.com/incognitochain/incognito-chain/portal/metadata"
	"github.com/pkg/errors"
)

type CurrentPortalState struct {
	CustodianPoolState      map[string]*statedb.CustodianState        // key : hash(custodian_address)
	WaitingPortingRequests  map[string]*statedb.WaitingPortingRequest // key : hash(UniquePortingID)
	WaitingRedeemRequests   map[string]*statedb.RedeemRequest         // key : hash(UniqueRedeemID)
	MatchedRedeemRequests   map[string]*statedb.RedeemRequest         // key : hash(UniquePortingID)
	FinalExchangeRatesState *statedb.FinalExchangeRatesState
	LiquidationPool         map[string]*statedb.LiquidationPool // key : hash(beaconHeight || TxID)
	// it used for calculate reward for custodian at the end epoch
	LockedCollateralForRewards *statedb.LockedCollateralState
	//Store temporary exchange rates requests
	ExchangeRatesRequests map[string]*portalMeta.ExchangeRatesRequestStatus // key : hash(beaconHeight | TxID)
	MultisigWalletsState  map[string]*statedb.MultisigWalletsState          // key: hash(TokenID)
}

type CustodianStateSlice struct {
	Key   string
	Value *statedb.CustodianState
}

func InitCurrentPortalStateFromDB(
	stateDB *statedb.StateDB,
) (*CurrentPortalState, error) {
	custodianPoolState, err := statedb.GetCustodianPoolState(stateDB)
	if err != nil {
		return nil, err
	}
	waitingPortingReqs, err := statedb.GetWaitingPortingRequests(stateDB)
	if err != nil {
		return nil, err
	}
	waitingRedeemReqs, err := statedb.GetWaitingRedeemRequests(stateDB)
	if err != nil {
		return nil, err
	}
	matchedRedeemReqs, err := statedb.GetMatchedRedeemRequests(stateDB)
	if err != nil {
		return nil, err
	}
	finalExchangeRates, err := statedb.GetFinalExchangeRatesState(stateDB)
	if err != nil {
		return nil, err
	}
	liquidateExchangeRatesPool, err := statedb.GetLiquidateExchangeRatesPool(stateDB)
	if err != nil {
		return nil, err
	}
	lockedCollateralState, err := statedb.GetLockedCollateralStateByBeaconHeight(stateDB)
	if err != nil {
		return nil, err
	}

	return &CurrentPortalState{
		CustodianPoolState:         custodianPoolState,
		WaitingPortingRequests:     waitingPortingReqs,
		WaitingRedeemRequests:      waitingRedeemReqs,
		MatchedRedeemRequests:      matchedRedeemReqs,
		FinalExchangeRatesState:    finalExchangeRates,
		ExchangeRatesRequests:      make(map[string]*portalMeta.ExchangeRatesRequestStatus),
		LiquidationPool:            liquidateExchangeRatesPool,
		LockedCollateralForRewards: lockedCollateralState,
	}, nil
}

func StorePortalStateToDB(
	stateDB *statedb.StateDB,
	currentPortalState *CurrentPortalState,
) error {
	err := statedb.StoreCustodianState(stateDB, currentPortalState.CustodianPoolState)
	if err != nil {
		return err
	}
	err = statedb.StoreBulkWaitingPortingRequests(stateDB, currentPortalState.WaitingPortingRequests)
	if err != nil {
		return err
	}
	err = statedb.StoreWaitingRedeemRequests(stateDB, currentPortalState.WaitingRedeemRequests)
	if err != nil {
		return err
	}
	err = statedb.StoreMatchedRedeemRequests(stateDB, currentPortalState.MatchedRedeemRequests)
	if err != nil {
		return err
	}
	err = statedb.StoreBulkFinalExchangeRatesState(stateDB, currentPortalState.FinalExchangeRatesState)
	if err != nil {
		return err
	}
	err = statedb.StoreBulkLiquidateExchangeRatesPool(stateDB, currentPortalState.LiquidationPool)
	if err != nil {
		return err
	}
	err = statedb.StoreLockedCollateralState(stateDB, currentPortalState.LockedCollateralForRewards)
	if err != nil {
		return err
	}

	return nil
}

// calculate matched public token amount and locked PRV and locked token collaterals respectively
// of one custodian that match to porting request
func calMatchedPubTokenAmountAndLockCollateralsForPorting(
	portingAmount uint64,
	totalLockCollateralInUSDT uint64, matchLockCollateralInUSDT uint64,
	convertRateTool *PortalExchangeRateTool, custodianState *statedb.CustodianState,
) (uint64, uint64, map[string]uint64, error) {
	// matched public token amount is calculated by percent matchLockCollateralInUSDT of totalLockCollateralInUSDT
	tmp := new(big.Int).Mul(new(big.Int).SetUint64(matchLockCollateralInUSDT), new(big.Int).SetUint64(portingAmount))
	pubTokenAmountCanBeHold := tmp.Div(tmp, new(big.Int).SetUint64(totalLockCollateralInUSDT)).Uint64()

	lockPRVCollateral := uint64(0)
	lockTokenCollaterals := map[string]uint64{}

	remainLockCollateralInUSDT := matchLockCollateralInUSDT

	// lock collateral PRV first
	freePRVCollateral := custodianState.GetFreeCollateral()
	if freePRVCollateral > 0 {
		freePRVCollateralInUSDT, _ := convertRateTool.ConvertToUSD(common.PRVIDStr, freePRVCollateral)
		if freePRVCollateralInUSDT >= matchLockCollateralInUSDT {
			lockPRVCollateral, _ = convertRateTool.ConvertFromUSD(common.PRVIDStr, matchLockCollateralInUSDT)
			return pubTokenAmountCanBeHold, lockPRVCollateral, lockTokenCollaterals, nil
		} else {
			lockPRVCollateral = custodianState.GetFreeCollateral()
			remainLockCollateralInUSDT = matchLockCollateralInUSDT - freePRVCollateralInUSDT
		}
	}

	// lock other token collaterals
	freeTokenCollaterals := custodianState.GetFreeTokenCollaterals()
	sortedTokenIDs := []string{}
	for tokenID := range freeTokenCollaterals {
		sortedTokenIDs = append(sortedTokenIDs, tokenID)
	}
	sort.Strings(sortedTokenIDs)
	for _, tokenID := range sortedTokenIDs {
		amount := freeTokenCollaterals[tokenID]
		if amount == 0 {
			continue
		}
		freeTokenInUSDT, _ := convertRateTool.ConvertToUSD(tokenID, amount)
		lockTokenCollateralAmt := uint64(0)
		if freeTokenInUSDT >= remainLockCollateralInUSDT {
			lockTokenCollateralAmt, _ = convertRateTool.ConvertFromUSD(tokenID, remainLockCollateralInUSDT)
			remainLockCollateralInUSDT = 0
		} else {
			lockTokenCollateralAmt = amount
			remainLockCollateralInUSDT -= freeTokenInUSDT
		}
		lockTokenCollaterals[tokenID] = lockTokenCollateralAmt
		if remainLockCollateralInUSDT == 0 {
			break
		}
	}

	if remainLockCollateralInUSDT > 0 {
		return 0, 0, nil, errors.New("Not enough free collaterals to lock for porting request")
	}

	return pubTokenAmountCanBeHold, lockPRVCollateral, lockTokenCollaterals, nil
}

// pickUpCustodianForPorting pick up custodians for matching to porting request
func pickUpCustodianForPorting(
	portingAmount uint64, portalTokenID string,
	custodianPool map[string]*statedb.CustodianState,
	exchangeRate *statedb.FinalExchangeRatesState,
	portalParams portal.PortalParams) ([]*statedb.MatchingPortingCustodianDetail, error) {
	if len(custodianPool) == 0 {
		return nil, errors.New("pickUpCustodianForPorting: Custodian pool is empty")
	}
	if exchangeRate == nil {
		return nil, errors.New("pickUpCustodianForPorting: Current exchange rate is nil")
	}

	// convert free collaterals of custodians to usdt to compare and sort descending
	convertRateTool := NewPortalExchangeRateTool(exchangeRate, portalParams)
	type custodianTotalCollateral struct {
		custodianKey string
		amountInUSDT uint64
	}
	sortedCusCollaterals := make([]custodianTotalCollateral, 0)
	for cusKey, cusDetail := range custodianPool {
		if cusDetail.GetRemoteAddresses() != nil && cusDetail.GetRemoteAddresses()[portalTokenID] != "" {
			collateralInUSDT, err := convertAllFreeCollateralsToUSDT(convertRateTool, cusDetail)
			if err != nil {
				return nil, errors.New("pickUpCustodianForPorting: Error when converting free collateral to USDT")
			}

			sortedCusCollaterals = append(sortedCusCollaterals,
				custodianTotalCollateral{
					custodianKey: cusKey,
					amountInUSDT: collateralInUSDT,
				})
		}
	}
	if len(sortedCusCollaterals) == 0 {
		return nil, errors.New("pickUpCustodianForPorting: There is no custodian supply remote address for porting tokenID")
	}
	sort.Slice(sortedCusCollaterals, func(i, j int) bool {
		if sortedCusCollaterals[i].amountInUSDT > sortedCusCollaterals[j].amountInUSDT {
			return true
		} else if (sortedCusCollaterals[i].amountInUSDT == sortedCusCollaterals[j].amountInUSDT) &&
			(sortedCusCollaterals[i].custodianKey < sortedCusCollaterals[j].custodianKey) {
			return true
		}
		return false
	})

	// convert porting amount (up to percent) to USDT
	portAmtInUSDT, _ := convertRateTool.ConvertToUSD(portalTokenID, UpPercent(portingAmount, portalParams.MinPercentLockedCollateral))

	// choose the custodian that has free collateral
	matchCustodians := make([]*statedb.MatchingPortingCustodianDetail, 0)

	isChooseOneCustodian := false
	if sortedCusCollaterals[0].amountInUSDT >= portAmtInUSDT {
		isChooseOneCustodian = true
	}

	actualHoldPubToken := uint64(0)
	remainPortAmtInUSDT := portAmtInUSDT
	for i, cus := range sortedCusCollaterals {
		pickedCus := cus
		if cus.amountInUSDT > portAmtInUSDT && i != len(sortedCusCollaterals)-1 {
			continue
		} else if cus.amountInUSDT < portAmtInUSDT && isChooseOneCustodian && i > 0 {
			pickedCus = sortedCusCollaterals[i-1]
		}

		custodianState := custodianPool[pickedCus.custodianKey]

		matchPortAmtInUSDT := pickedCus.amountInUSDT
		if pickedCus.amountInUSDT > remainPortAmtInUSDT {
			matchPortAmtInUSDT = remainPortAmtInUSDT
		}

		holdPublicToken, lockPRVCollateral, lockTokenColaterals, err := calMatchedPubTokenAmountAndLockCollateralsForPorting(
			portingAmount, portAmtInUSDT, matchPortAmtInUSDT, convertRateTool, custodianState)
		if err != nil {
			return nil, err
		}
		actualHoldPubToken += holdPublicToken

		matchCus := statedb.MatchingPortingCustodianDetail{
			IncAddress:             custodianState.GetIncognitoAddress(),
			RemoteAddress:          custodianState.GetRemoteAddresses()[portalTokenID],
			Amount:                 holdPublicToken,
			LockedAmountCollateral: lockPRVCollateral,
			LockedTokenCollaterals: lockTokenColaterals,
		}
		matchCustodians = append(matchCustodians, &matchCus)

		remainPortAmtInUSDT -= matchPortAmtInUSDT
		if remainPortAmtInUSDT == 0 {
			if actualHoldPubToken < portingAmount {
				matchCustodians[0].Amount = matchCustodians[0].Amount + portingAmount - actualHoldPubToken
			}
			break
		}
	}
	if remainPortAmtInUSDT > 0 {
		return nil, errors.New("pickUpCustodianForPorting: Not enough custodians for matching to porting request")
	}

	return matchCustodians, nil
}

// UpdateCustodianStateAfterMatchingPortingRequest updates current portal state after matching porting request
func UpdateCustodianStateAfterMatchingPortingRequest(
	currentPortalState *CurrentPortalState,
	matchCus *statedb.MatchingPortingCustodianDetail,
	portalTokenID string) error {
	custodianKey := statedb.GenerateCustodianStateObjectKey(matchCus.IncAddress).String()
	custodian, ok := currentPortalState.CustodianPoolState[custodianKey]
	if !ok {
		return errors.New("Custodian not found")
	}
	// lock PRV collateral
	if matchCus.LockedAmountCollateral > 0 {
		freeCollateral := custodian.GetFreeCollateral() - matchCus.LockedAmountCollateral

		lockPRVCollateral := custodian.GetLockedAmountCollateral()
		if lockPRVCollateral == nil {
			lockPRVCollateral = make(map[string]uint64)
		}
		lockPRVCollateral[portalTokenID] += matchCus.LockedAmountCollateral

		custodian.SetFreeCollateral(freeCollateral)
		custodian.SetLockedAmountCollateral(lockPRVCollateral)
	}

	// lock token collaterals
	if len(matchCus.LockedTokenCollaterals) > 0 {
		freeTokenCollaterals := custodian.GetFreeTokenCollaterals()
		lockTokenCollaterals := custodian.GetLockedTokenCollaterals()
		if lockTokenCollaterals == nil {
			lockTokenCollaterals = map[string]map[string]uint64{}
		}
		if lockTokenCollaterals[portalTokenID] == nil {
			lockTokenCollaterals[portalTokenID] = map[string]uint64{}
		}
		for collateralTokenID, amount := range matchCus.LockedTokenCollaterals {
			freeTokenCollaterals[collateralTokenID] -= amount
			lockTokenCollaterals[portalTokenID][collateralTokenID] += amount
		}

		custodian.SetFreeTokenCollaterals(freeTokenCollaterals)
		custodian.SetLockedTokenCollaterals(lockTokenCollaterals)
	}

	// Note: don't update holding public tokens to avoid this custodian match to redeem request before receiving pubtokens from users
	currentPortalState.CustodianPoolState[custodianKey] = custodian

	return nil
}

// UpdateCustodianStateAfterMatchingPortingRequest updates current portal state after requesting ptoken
func UpdateMultisigWalletsStateAfterUserRequestPToken(currentPortalState *CurrentPortalState, multiWalletsKey string, walletAddress string, utxo statedb.UTXO) error {
	wallet, ok := currentPortalState.MultisigWalletsState[multiWalletsKey]
	if !ok {
		return errors.New("[UpdateMultisigWalletsStateAfterUserRequestPToken] MultisigWallet not found")
	}

	curListUTXO := wallet.GetListUTXO()
	if curListUTXO == nil {
		curListUTXO = []statedb.UTXO{}
	}
	curListUTXO = append(curListUTXO, utxo)
	currentPortalState.MultisigWalletsState[multiWalletsKey].SetListUTXO(curListUTXO)

	return nil
}

// UpdateCustodianStateAfterMatchingPortingRequest updates current portal state after requesting ptoken
func UpdateCustodianStateAfterUserRequestPToken(currentPortalState *CurrentPortalState, custodianKey string, pTokenId string, amountPToken uint64) error {
	custodian, ok := currentPortalState.CustodianPoolState[custodianKey]
	if !ok {
		return errors.New("[UpdateCustodianStateAfterUserRequestPToken] Custodian not found")
	}

	holdingPubTokensTmp := custodian.GetHoldingPublicTokens()
	if holdingPubTokensTmp == nil {
		holdingPubTokensTmp = make(map[string]uint64)
	}
	holdingPubTokensTmp[pTokenId] += amountPToken

	currentPortalState.CustodianPoolState[custodianKey].SetHoldingPublicTokens(holdingPubTokensTmp)
	return nil
}

// CalMinPortingFee calculates the minimum porting fee in PRV
func CalMinPortingFee(portingAmount uint64, portalTokenID string, exchangeRate *statedb.FinalExchangeRatesState, portalParam portal.PortalParams) (uint64, error) {
	exchangeTool := NewPortalExchangeRateTool(exchangeRate, portalParam)
	portingAmountInPRV, err := exchangeTool.Convert(portalTokenID, common.PRVIDStr, portingAmount)
	if err != nil {
		Logger.log.Errorf("Error when calculating minimum porting fee %v", err)
		return 0, err
	}

	// can't use big int to calculate porting fee because of common.MinPercentPortingFee < 1
	portingFee := uint64(math.Round(float64(portingAmountInPRV) * portalParam.MinPercentPortingFee / 100))

	if portingFee < portalParam.MinPortalFee {
		return portalParam.MinPortalFee, nil
	}

	return portingFee, nil
}

// CalMinRedeemFee calculates the minimum redeeming fee in PRV
func CalMinRedeemFee(redeemAmountInPToken uint64, portalTokenID string, exchangeRate *statedb.FinalExchangeRatesState, portalParam portal.PortalParams) (uint64, error) {
	exchangeTool := NewPortalExchangeRateTool(exchangeRate, portalParam)
	redeemAmountInPRV, err := exchangeTool.Convert(portalTokenID, common.PRVIDStr, redeemAmountInPToken)
	if err != nil {
		Logger.log.Errorf("Error when calculating minimum redeem fee %v", err)
		return 0, err
	}

	// can't use big int to calculate porting fee because of common.MinPercentRedeemFee < 1
	redeemFee := uint64(math.Round(float64(redeemAmountInPRV) * portalParam.MinPercentRedeemFee / 100))

	if redeemFee < portalParam.MinPortalFee {
		return portalParam.MinPortalFee, nil
	}

	return redeemFee, nil
}

// UpPercent returns the result be up to percent of amount
func UpPercent(amount uint64, percent uint64) uint64 {
	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amount), new(big.Int).SetUint64(percent))
	result := new(big.Int).Div(tmp, new(big.Int).SetUint64(100)).Uint64()
	return result
}

// DownPercent returns the result be down to percent of amount
func DownPercent(amount uint64, percent uint64) uint64 {
	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amount), new(big.Int).SetUint64(100))
	result := new(big.Int).Div(tmp, new(big.Int).SetUint64(percent)).Uint64()
	return result
}

// calMintedPRVCollateralForRedeemFromLiquidationPoolV2 calculated PRV collateral amount will be minted
// when redeeming from liquidation pool v2
func calMintedPRVCollateralForRedeemFromLiquidationPoolV2(redeemAmount uint64, liquidationPool statedb.LiquidationPoolDetail) (uint64, error) {
	if liquidationPool.PubTokenAmount <= 0 {
		return 0, errors.New("Amount of public token in liquidation pool is zero")
	}

	tmp := new(big.Int).Mul(
		new(big.Int).SetUint64(liquidationPool.CollateralAmount),
		new(big.Int).SetUint64(redeemAmount),
	)
	mintedPRVCollateral := new(big.Int).Div(
		tmp,
		new(big.Int).SetUint64(liquidationPool.PubTokenAmount),
	)
	return mintedPRVCollateral.Uint64(), nil
}

//check value is tp120 or tp130 for portal v2
func checkTPRatio(tpValue uint64, portalParams portal.PortalParams) (bool, bool) {
	if tpValue > portalParams.TP120 && tpValue <= portalParams.TP130 {
		return false, true
	}

	if tpValue <= portalParams.TP120 {
		return true, true
	}

	//not found
	return false, false
}

// CalTopupAmountForCustodianState calculates topup amount for one custodian
func CalTopupAmountForCustodianState(
	currentPortalState *CurrentPortalState,
	custodian *statedb.CustodianState,
	exchangeRates *statedb.FinalExchangeRatesState,
	portalTokenId string,
	collateralTokenID string,
	portalParams portal.PortalParams) (uint64, error) {
	exchangeTool := NewPortalExchangeRateTool(exchangeRates, portalParams)

	// get total hold public token
	totalHoldingPublicToken := GetTotalHoldPubTokenAmount(currentPortalState, custodian, portalTokenId)
	if totalHoldingPublicToken == 0 {
		return 0, nil
	}
	// minimum locked collateral amount in ptoken
	minLockedCollateralAmtInPToken := UpPercent(totalHoldingPublicToken, portalParams.MinPercentLockedCollateral)

	// convert minLockedCollateralAmtInPToken to usd
	minLockedCollateralAmtInUSD, err := exchangeTool.ConvertToUSD(portalTokenId, minLockedCollateralAmtInPToken)
	if err != nil {
		return 0, err
	}

	// get total locked collaterals (exclude in waiting portings) in usd
	currentTotalLockedCollateralInUSD, err := getLockCollateralsInUSDExcludeWPortings(exchangeTool, custodian, currentPortalState)
	if err != nil || currentTotalLockedCollateralInUSD == nil {
		return 0, err
	}
	if currentTotalLockedCollateralInUSD[portalTokenId] >= minLockedCollateralAmtInUSD {
		return 0, nil
	}

	// calculate topup amount
	topupAmountInUSDT := minLockedCollateralAmtInUSD - currentTotalLockedCollateralInUSD[portalTokenId]
	if collateralTokenID == "" {
		collateralTokenID = common.PRVIDStr
	}
	topupAmount, err := exchangeTool.ConvertFromUSD(collateralTokenID, topupAmountInUSDT)
	if err != nil {
		return 0, err
	}

	return topupAmount, nil
}

// sortCustodiansByAmountHoldingPubTokenAscent sorts custodians by holding public token amount ascending
func sortCustodiansByAmountHoldingPubTokenAscent(tokenID string, custodians map[string]*statedb.CustodianState) []*CustodianStateSlice {
	sortedCustodians := make([]*CustodianStateSlice, 0)
	for key, value := range custodians {
		if value.GetHoldingPublicTokens()[tokenID] > 0 {
			item := CustodianStateSlice{
				Key:   key,
				Value: value,
			}
			sortedCustodians = append(sortedCustodians, &item)
		}
	}

	sort.Slice(sortedCustodians, func(i, j int) bool {
		if sortedCustodians[i].Value.GetHoldingPublicTokens()[tokenID] < sortedCustodians[j].Value.GetHoldingPublicTokens()[tokenID] {
			return true
		} else if (sortedCustodians[i].Value.GetHoldingPublicTokens()[tokenID] == sortedCustodians[j].Value.GetHoldingPublicTokens()[tokenID]) &&
			(sortedCustodians[i].Value.GetIncognitoAddress() < sortedCustodians[j].Value.GetIncognitoAddress()) {
			return true
		}
		return false
	})

	return sortedCustodians
}

// pickupCustodianForRedeem picks up custodians for redeeming request
func pickupCustodianForRedeem(redeemAmount uint64, tokenID string, portalState *CurrentPortalState) ([]*statedb.MatchingRedeemCustodianDetail, error) {
	custodianPoolState := portalState.CustodianPoolState
	matchedCustodians := make([]*statedb.MatchingRedeemCustodianDetail, 0)

	// sort smallCustodians by amount holding public token
	sortedCustodianSlice := sortCustodiansByAmountHoldingPubTokenAscent(tokenID, custodianPoolState)
	if len(sortedCustodianSlice) == 0 {
		Logger.log.Errorf("There is no suitable custodian in pool for redeem request")
		return nil, errors.New("There is no suitable custodian in pool for redeem request")
	}

	totalMatchedAmount := uint64(0)
	for i := len(sortedCustodianSlice) - 1; i >= 0; i-- {
		custodianKey := sortedCustodianSlice[i].Key
		custodianValue := sortedCustodianSlice[i].Value

		matchedAmount := custodianValue.GetHoldingPublicTokens()[tokenID]
		amountNeedToBeMatched := redeemAmount - totalMatchedAmount
		if matchedAmount > amountNeedToBeMatched {
			matchedAmount = amountNeedToBeMatched
		}

		remoteAddr := custodianValue.GetRemoteAddresses()[tokenID]
		if remoteAddr == "" {
			Logger.log.Errorf("Remote address in tokenID %v of custodian %v is null", tokenID, custodianValue.GetIncognitoAddress())
			return nil, fmt.Errorf("Remote address in tokenID %v of custodian %v is null", tokenID, custodianValue.GetIncognitoAddress())
		}

		matchedCustodians = append(
			matchedCustodians,
			statedb.NewMatchingRedeemCustodianDetailWithValue(
				custodianPoolState[custodianKey].GetIncognitoAddress(), remoteAddr, matchedAmount))

		totalMatchedAmount += matchedAmount
		if totalMatchedAmount >= redeemAmount {
			return matchedCustodians, nil
		}
	}

	Logger.log.Errorf("Not enough amount public token to return user")
	return nil, errors.New("Not enough amount public token to return user")
}

// updateCustodianStateAfterReqUnlockCollateral updates custodian state (amount collaterals) when custodian returns redeemAmount public token to user
func updateCustodianStateAfterReqUnlockCollateral(custodianState *statedb.CustodianState, unlockedAmount uint64, tokenID string) error {
	lockedAmount := custodianState.GetLockedAmountCollateral()
	if lockedAmount == nil {
		return errors.New("[portal-updateCustodianStateAfterReqUnlockCollateral] Locked amount is nil")
	}
	if lockedAmount[tokenID] < unlockedAmount {
		return errors.New("[portal-updateCustodianStateAfterReqUnlockCollateral] Locked amount is less than amount need to unlocked")
	}

	lockedAmount[tokenID] -= unlockedAmount
	custodianState.SetLockedAmountCollateral(lockedAmount)
	custodianState.SetFreeCollateral(custodianState.GetFreeCollateral() + unlockedAmount)
	return nil
}

// updateCustodianStateAfterReqUnlockCollateralV3 updates custodian state (amount collaterals) when custodian returns redeemAmount public token to user
func updateCustodianStateAfterReqUnlockCollateralV3(
	custodianState *statedb.CustodianState, unlockedAmount uint64, tokenID string,
	portalParams portal.PortalParams, portalState *CurrentPortalState) (map[string]uint64, error) {
	lockedTokenAmounts := custodianState.GetLockedTokenCollaterals()
	lockedPrvAmount := custodianState.GetLockedAmountCollateral()
	tokenAmountsUnlocked := make(map[string]uint64, 0)
	if lockedTokenAmounts == nil && lockedPrvAmount == nil {
		return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] Locked amount is nil")
	}

	// convert free collaterals of custodians to usdt to compare and sort descending
	convertRateTool := NewPortalExchangeRateTool(portalState.FinalExchangeRatesState, portalParams)
	tokenAmountListInWaitingPoring := GetTotalLockedCollateralAmountInWaitingPortingsV3(portalState, custodianState, tokenID)

	// unlock PRV collateral first
	if lockedPrvAmount != nil && lockedPrvAmount[tokenID] > 0 {
		if lockedPrvAmount[tokenID] < tokenAmountListInWaitingPoring[common.PRVIDStr] {
			return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] Locked amount must greater then amount in waiting porting")
		}
		lockedPrvAmountToProcess := lockedPrvAmount[tokenID] - tokenAmountListInWaitingPoring[common.PRVIDStr]
		tokenAmtInUSD, err := convertRateTool.ConvertToUSD(common.PRVIDStr, lockedPrvAmountToProcess)
		if err != nil {
			return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] Can not convert prv to usd")
		}
		prvCollateralAmountToUpdate := uint64(0)
		if unlockedAmount >= tokenAmtInUSD {
			unlockedAmount -= tokenAmtInUSD
			prvCollateralAmountToUpdate = lockedPrvAmountToProcess
		} else {
			prvCollateralAmountToUpdate, err = convertRateTool.ConvertFromUSD(common.PRVIDStr, unlockedAmount)
			unlockedAmount = 0
		}
		if err != nil || prvCollateralAmountToUpdate > lockedPrvAmountToProcess {
			return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] Can not convert usd to collateral prv")
		}
		if prvCollateralAmountToUpdate > 0 {
			// update collateral prv token
			lockedPrvAmount[tokenID] -= prvCollateralAmountToUpdate
			tokenAmountsUnlocked[common.PRVIDStr] = prvCollateralAmountToUpdate
			custodianState.SetLockedAmountCollateral(lockedPrvAmount)
			// update free prv token
			custodianState.SetFreeCollateral(custodianState.GetFreeCollateral() + prvCollateralAmountToUpdate)
		}
	}

	freeTokenCollaterals := custodianState.GetFreeTokenCollaterals()

	if unlockedAmount > 0 {
		if lockedTokenAmounts == nil {
			return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] can not unlock nil tokens")
		}
		// lock other token collaterals
		sortedTokenIDs := []string{}
		for tokenCollateralID := range lockedTokenAmounts[tokenID] {
			sortedTokenIDs = append(sortedTokenIDs, tokenCollateralID)
		}
		sort.Strings(sortedTokenIDs)

		for _, tokenCollateralID := range sortedTokenIDs {
			if lockedTokenAmounts[tokenID][tokenCollateralID] < tokenAmountListInWaitingPoring[tokenCollateralID] {
				return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] Locked amount must greater then amount in waiting porting")
			}
			if lockedTokenAmounts[tokenID][tokenCollateralID] == 0 {
				continue
			}
			lockedTokenAmountToProcess := lockedTokenAmounts[tokenID][tokenCollateralID] - tokenAmountListInWaitingPoring[tokenCollateralID]
			tokenValueLocked, err := convertRateTool.ConvertToUSD(tokenCollateralID, lockedTokenAmountToProcess)
			if err != nil {
				Logger.log.Errorf("[portal-updateCustodianStateAfterReqUnlockCollateralV3] got error %v", err.Error())
				return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] got error while get convert from collateral to USDT ")
			}
			tokenCollateralAmountToUpdate := uint64(0)
			if unlockedAmount >= tokenValueLocked {
				unlockedAmount -= tokenValueLocked
				tokenCollateralAmountToUpdate = lockedTokenAmountToProcess
			} else {
				tokenCollateralAmountToUpdate, err = convertRateTool.ConvertFromUSD(tokenCollateralID, unlockedAmount)
				unlockedAmount = 0
			}
			if err != nil || tokenCollateralAmountToUpdate > lockedTokenAmountToProcess {
				return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] total locked token less than amount to unlock")
			}
			if tokenCollateralAmountToUpdate > 0 {
				lockedTokenAmounts[tokenID][tokenCollateralID] -= tokenCollateralAmountToUpdate
				tokenAmountsUnlocked[tokenCollateralID] = tokenCollateralAmountToUpdate
				freeTokenCollaterals[tokenCollateralID] += tokenCollateralAmountToUpdate
			}
			if unlockedAmount == 0 {
				break
			}
		}

		if unlockedAmount > 0 {
			return nil, errors.New("[portal-updateCustodianStateAfterReqUnlockCollateralV3] not enough collateral tokens to unlock for custodian")
		}
		custodianState.SetLockedTokenCollaterals(lockedTokenAmounts)
		custodianState.SetFreeTokenCollaterals(freeTokenCollaterals)
	}
	return tokenAmountsUnlocked, nil
}

// CalUnlockCollateralAmount returns unlock collateral amount by percentage of redeem amount
func CalUnlockCollateralAmount(
	portalState *CurrentPortalState,
	custodianStateKey string,
	redeemAmount uint64,
	tokenID string) (uint64, error) {
	custodianState := portalState.CustodianPoolState[custodianStateKey]
	if custodianState == nil {
		Logger.log.Errorf("[test][CalUnlockCollateralAmount] Custodian not found %v\n", custodianStateKey)
		return 0, fmt.Errorf("Custodian not found %v\n", custodianStateKey)
	}

	totalHoldingPubToken := GetTotalHoldPubTokenAmount(portalState, custodianState, tokenID)

	totalLockedAmountInWaitingPortings := GetTotalLockedCollateralAmountInWaitingPortings(portalState, custodianState, tokenID)

	if custodianState.GetLockedAmountCollateral()[tokenID] < totalLockedAmountInWaitingPortings {
		Logger.log.Errorf("custodianState.GetLockedAmountCollateral()[tokenID] %v\n", custodianState.GetLockedAmountCollateral()[tokenID])
		Logger.log.Errorf("totalLockedAmountInWaitingPortings %v\n", totalLockedAmountInWaitingPortings)
		return 0, errors.New("[CalUnlockCollateralAmount] Lock amount is invalid")
	}

	if totalHoldingPubToken == 0 {
		Logger.log.Errorf("[CalUnlockCollateralAmount] Total holding public token amount of custodianAddr %v is zero", custodianState.GetIncognitoAddress())
		return 0, errors.New("[CalUnlockCollateralAmount] Total holding public token amount is zero")
	}

	tmp := new(big.Int).Mul(
		new(big.Int).SetUint64(redeemAmount),
		new(big.Int).SetUint64(custodianState.GetLockedAmountCollateral()[tokenID]-totalLockedAmountInWaitingPortings))
	unlockAmount := new(big.Int).Div(tmp, new(big.Int).SetUint64(totalHoldingPubToken)).Uint64()
	if unlockAmount <= 0 {
		Logger.log.Errorf("[CalUnlockCollateralAmount] Can not calculate unlock amount for custodian %v\n", unlockAmount)
		return 0, errors.New("[CalUnlockCollateralAmount] Can not calculate unlock amount for custodian")
	}
	return unlockAmount, nil
}

// CalUnlockCollateralAmountV3 returns unlock collateral amount by percentage of redeem amount in usd
func CalUnlockCollateralAmountV3(
	portalState *CurrentPortalState,
	custodianStateKey string,
	redeemAmount uint64,
	tokenID string,
	portalParams portal.PortalParams) (uint64, map[string]uint64, error) {
	custodianState := portalState.CustodianPoolState[custodianStateKey]
	if custodianState == nil {
		Logger.log.Errorf("[test][CalUnlockCollateralAmount] Custodian not found %v\n", custodianStateKey)
		return 0, nil, fmt.Errorf("Custodian not found %v\n", custodianStateKey)
	}

	totalHoldingPubToken := GetTotalHoldPubTokenAmount(portalState, custodianState, tokenID)
	if totalHoldingPubToken == 0 {
		Logger.log.Errorf("[CalUnlockCollateralAmount] Total holding public token amount of custodianAddr %v is zero", custodianState.GetIncognitoAddress())
		return 0, nil, errors.New("[CalUnlockCollateralAmount] Total holding public token amount is zero")
	}

	// convert free collaterals of custodians to usdt to compare and sort descending
	convertRateTool := NewPortalExchangeRateTool(portalState.FinalExchangeRatesState, portalParams)
	tokenAmountList := GetTotalLockedCollateralAmountInWaitingPortingsV3(portalState, custodianState, tokenID)
	lockedAmountCollateral := uint64(0)
	listLockedTokens := cloneMap(custodianState.GetLockedTokenCollaterals()[tokenID])
	if listLockedTokens == nil {
		listLockedTokens = map[string]uint64{}
	}
	listLockedTokens[common.PRVIDStr] = custodianState.GetLockedAmountCollateral()[tokenID]
	for tokenCollateralID, token := range listLockedTokens {
		if token < tokenAmountList[tokenCollateralID] {
			return 0, nil, errors.New("[CalUnlockCollateralAmountV3] got error while remove locked token porting in waiting state")
		}
		token -= tokenAmountList[tokenCollateralID]
		listLockedTokens[tokenCollateralID] = token
		tokenValueLocked, err := convertRateTool.ConvertToUSD(tokenCollateralID, token)
		if err != nil {
			Logger.log.Errorf("[CalUnlockCollateralAmountV3] got error %v", err.Error())
			return 0, nil, errors.New("[CalUnlockCollateralAmountV3] got error while get convert from collateral to USDT ")
		}
		lockedAmountCollateral += tokenValueLocked
	}

	tmp := new(big.Int).Mul(
		new(big.Int).SetUint64(redeemAmount),
		new(big.Int).SetUint64(lockedAmountCollateral))
	unlockAmount := new(big.Int).Div(tmp, new(big.Int).SetUint64(totalHoldingPubToken)).Uint64()
	if unlockAmount <= 0 {
		Logger.log.Errorf("[CalUnlockCollateralAmount] Can not calculate unlock amount for custodian %v\n", unlockAmount)
		return 0, nil, errors.New("[CalUnlockCollateralAmount] Can not calculate unlock amount for custodian")
	}
	return unlockAmount, listLockedTokens, nil
}

func CalUnlockCollateralAmountAfterLiquidation(
	portalState *CurrentPortalState,
	liquidatedCustodianStateKey string,
	amountPubToken uint64,
	tokenID string,
	exchangeRate *statedb.FinalExchangeRatesState,
	portalParams portal.PortalParams) (uint64, uint64, error) {
	totalUnlockCollateralAmount, err := CalUnlockCollateralAmount(portalState, liquidatedCustodianStateKey, amountPubToken, tokenID)
	if err != nil {
		Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidation error : %v\n", err)
		return 0, 0, err
	}
	exchangeTool := NewPortalExchangeRateTool(exchangeRate, portalParams)

	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amountPubToken), new(big.Int).SetUint64(uint64(portalParams.MaxPercentLiquidatedCollateralAmount)))
	liquidatedAmountInPToken := new(big.Int).Div(tmp, new(big.Int).SetUint64(100)).Uint64()
	liquidatedAmountInPRV, err := exchangeTool.Convert(tokenID, common.PRVIDStr, liquidatedAmountInPToken)
	if err != nil {
		Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidation error converting rate : %v\n", err)
		return 0, 0, err
	}

	if liquidatedAmountInPRV > totalUnlockCollateralAmount {
		liquidatedAmountInPRV = totalUnlockCollateralAmount
	}

	remainUnlockAmountForCustodian := totalUnlockCollateralAmount - liquidatedAmountInPRV
	return liquidatedAmountInPRV, remainUnlockAmountForCustodian, nil
}

func CalUnlockCollateralAmountAfterLiquidationV3(
	portalState *CurrentPortalState,
	liquidatedCustodianStateKey string,
	amountPubToken uint64,
	tokenID string,
	portalParams portal.PortalParams) (uint64, uint64, map[string]uint64, map[string]uint64, error) {
	totalUnlockCollateralAmount, listAvailableToUnlock, err := CalUnlockCollateralAmountV3(portalState, liquidatedCustodianStateKey, amountPubToken, tokenID, portalParams)
	if err != nil {
		Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidationV3 error : %v\n", err)
		return 0, 0, nil, nil, err
	}

	// convert free collaterals of custodians to usdt to compare and sort descending
	convertRateTool := NewPortalExchangeRateTool(portalState.FinalExchangeRatesState, portalParams)

	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amountPubToken), new(big.Int).SetUint64(portalParams.MaxPercentLiquidatedCollateralAmount))
	liquidatedAmountInPToken := new(big.Int).Div(tmp, new(big.Int).SetUint64(100)).Uint64()
	liquidatedAmountInUSDT, err := convertRateTool.ConvertToUSD(tokenID, liquidatedAmountInPToken)
	if err != nil {
		Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidationV3 error converting rate : %v\n", err)
		return 0, 0, nil, nil, err
	}

	if liquidatedAmountInUSDT > totalUnlockCollateralAmount {
		liquidatedAmountInUSDT = totalUnlockCollateralAmount
	}

	remainUnlockAmountForCustodian := totalUnlockCollateralAmount - liquidatedAmountInUSDT

	tokenIDKeys := make([]string, 0)
	for tokenIDToUnlock := range listAvailableToUnlock {
		if tokenIDToUnlock == common.PRVIDStr {
			continue
		}
		tokenIDKeys = append(tokenIDKeys, tokenIDToUnlock)
	}
	sort.Strings(tokenIDKeys)
	tokenIDKeys = append([]string{common.PRVIDStr}, tokenIDKeys...)
	var liquidatedAmountInPrv, remainUnlockAmountForCustodianInPrv uint64
	liquidatedAmounts := make(map[string]uint64)
	remainUnlockAmountsForCustodian := make(map[string]uint64)

	for _, tokenCollateralID := range tokenIDKeys {
		tokenValueInUSDT, err := convertRateTool.ConvertToUSD(tokenCollateralID, listAvailableToUnlock[tokenCollateralID])
		if err != nil {
			Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidationV3 error converting rate : %v\n", err)
			return 0, 0, nil, nil, err
		}
		if tokenValueInUSDT > 0 {
			if liquidatedAmountInUSDT > 0 {
				if liquidatedAmountInUSDT >= tokenValueInUSDT {
					liquidatedAmountInUSDT -= tokenValueInUSDT
					liquidatedAmounts[tokenCollateralID] = listAvailableToUnlock[tokenCollateralID]
					continue
				} else {
					tokenValueInUSDT -= liquidatedAmountInUSDT
					tokenValueInCollateralToken, err := convertRateTool.ConvertFromUSD(tokenCollateralID, liquidatedAmountInUSDT)
					if err != nil {
						Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidationV3 error converting rate : %v\n", err)
						return 0, 0, nil, nil, err
					}
					if listAvailableToUnlock[tokenCollateralID] < tokenValueInCollateralToken {
						Logger.log.Error("CalUnlockCollateralAmountAfterLiquidationV3 error when calculating liquidated amount")
						return 0, 0, nil, nil, err
					}
					liquidatedAmounts[tokenCollateralID] = tokenValueInCollateralToken
					listAvailableToUnlock[tokenCollateralID] -= tokenValueInCollateralToken
					liquidatedAmountInUSDT = 0
				}
			}

			if remainUnlockAmountForCustodian > 0 {
				if tokenValueInUSDT > 0 {
					if remainUnlockAmountForCustodian >= tokenValueInUSDT {
						remainUnlockAmountForCustodian -= tokenValueInUSDT
						remainUnlockAmountsForCustodian[tokenCollateralID] = listAvailableToUnlock[tokenCollateralID]
					} else {
						tokenValueInCollateralToken, err := convertRateTool.ConvertFromUSD(tokenCollateralID, remainUnlockAmountForCustodian)
						if err != nil {
							Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidationV3 error converting rate : %v\n", err)
							return 0, 0, nil, nil, err
						}
						if listAvailableToUnlock[tokenCollateralID] < tokenValueInCollateralToken {
							Logger.log.Error("CalUnlockCollateralAmountAfterLiquidationV3 error when calculating remain amount")
							return 0, 0, nil, nil, err
						}
						remainUnlockAmountsForCustodian[tokenCollateralID] = tokenValueInCollateralToken
						remainUnlockAmountForCustodian = 0
					}
					if remainUnlockAmountForCustodian == 0 {
						break
					}
				}
			} else {
				break
			}
		}
	}

	if liquidatedAmountInUSDT > 0 || remainUnlockAmountForCustodian > 0 {
		Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidationV3 error not enough locked token to liquidate custodian")
		return 0, 0, nil, nil, err
	}
	liquidatedAmountInPrv = liquidatedAmounts[common.PRVIDStr]
	remainUnlockAmountForCustodianInPrv = remainUnlockAmountsForCustodian[common.PRVIDStr]
	delete(liquidatedAmounts, common.PRVIDStr)
	delete(remainUnlockAmountsForCustodian, common.PRVIDStr)

	return liquidatedAmountInPrv, remainUnlockAmountForCustodianInPrv, liquidatedAmounts, remainUnlockAmountsForCustodian, nil
}

// updateRedeemRequestStatusByRedeemId updates status of redeem request into db
func updateRedeemRequestStatusByRedeemId(redeemID string, newStatus int, db *statedb.StateDB) error {
	redeemRequestBytes, err := statedb.GetPortalRedeemRequestStatus(db, redeemID)
	if err != nil {
		return err
	}
	if len(redeemRequestBytes) == 0 {
		return fmt.Errorf("Not found redeem request from db with redeemId %v\n", redeemID)
	}

	var redeemRequest portalMeta.PortalRedeemRequestStatus
	err = json.Unmarshal(redeemRequestBytes, &redeemRequest)
	if err != nil {
		return err
	}

	redeemRequest.Status = byte(newStatus)
	newRedeemRequest, err := json.Marshal(redeemRequest)
	if err != nil {
		return err
	}
	err = statedb.StorePortalRedeemRequestStatus(db, redeemID, newRedeemRequest)
	if err != nil {
		return err
	}
	return nil
}

func updateCustodianStateAfterLiquidateCustodian(custodianState *statedb.CustodianState, liquidatedAmount uint64, remainUnlockAmountForCustodian uint64, tokenID string) error {
	if custodianState == nil {
		Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodian] custodian not found")
		return errors.New("[updateCustodianStateAfterLiquidateCustodian] custodian not found")
	}
	if custodianState.GetTotalCollateral() < liquidatedAmount {
		Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodian] total collateral less than liquidated amount")
		return errors.New("[updateCustodianStateAfterLiquidateCustodian] total collateral less than liquidated amount")
	}
	lockedAmountTmp := custodianState.GetLockedAmountCollateral()
	if lockedAmountTmp[tokenID] < liquidatedAmount+remainUnlockAmountForCustodian {
		Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodian] locked amount less than total unlock amount")
		return errors.New("[updateCustodianStateAfterLiquidateCustodian] locked amount less than total unlock amount")
	}

	custodianState.SetTotalCollateral(custodianState.GetTotalCollateral() - liquidatedAmount)

	lockedAmountTmp[tokenID] = lockedAmountTmp[tokenID] - liquidatedAmount - remainUnlockAmountForCustodian
	custodianState.SetLockedAmountCollateral(lockedAmountTmp)

	custodianState.SetFreeCollateral(custodianState.GetFreeCollateral() + remainUnlockAmountForCustodian)

	return nil
}

func updateCustodianStateAfterLiquidateCustodianV3(custodianState *statedb.CustodianState, liquidatedAmount, remainUnlockAmountForCustodian uint64, liquidatedAmounts, remainUnlockAmounts map[string]uint64, tokenID string) error {
	if custodianState == nil {
		Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodianV3] custodian not found")
		return errors.New("[updateCustodianStateAfterLiquidateCustodianV3] custodian not found")
	}
	if custodianState.GetTotalCollateral() < liquidatedAmount {
		Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodianV3] total collateral less than liquidated amount")
		return errors.New("[updateCustodianStateAfterLiquidateCustodianV3] total collateral less than liquidated amount")
	}
	lockedAmountTmp := custodianState.GetLockedAmountCollateral()
	if lockedAmountTmp[tokenID] < liquidatedAmount+remainUnlockAmountForCustodian {
		Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodianV3] locked amount less than total unlock amount")
		return errors.New("[updateCustodianStateAfterLiquidateCustodianV3] locked amount less than total unlock amount")
	}

	custodianState.SetTotalCollateral(custodianState.GetTotalCollateral() - liquidatedAmount)

	lockedAmountTmp[tokenID] = lockedAmountTmp[tokenID] - liquidatedAmount - remainUnlockAmountForCustodian
	custodianState.SetLockedAmountCollateral(lockedAmountTmp)

	custodianState.SetFreeCollateral(custodianState.GetFreeCollateral() + remainUnlockAmountForCustodian)

	if len(liquidatedAmounts) > 0 || len(remainUnlockAmounts) > 0 {
		lockedCollaterals := custodianState.GetLockedTokenCollaterals()
		freeCollaterals := custodianState.GetFreeTokenCollaterals()
		totalTokenCollaterals := custodianState.GetTotalTokenCollaterals()
		for tokenCollateralId, tokenValue := range lockedCollaterals[tokenID] {
			if totalTokenCollaterals[tokenCollateralId] < tokenValue {
				Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodianV3] total collateral less than locked amount")
				return errors.New("[updateCustodianStateAfterLiquidateCustodianV3] total collateral less than locked amount")
			}
			if tokenValue < liquidatedAmounts[tokenCollateralId]+remainUnlockAmounts[tokenCollateralId] {
				Logger.log.Errorf("[updateCustodianStateAfterLiquidateCustodianV3] locked amount less than total unlock amount")
				return errors.New("[updateCustodianStateAfterLiquidateCustodianV3] locked amount less than total unlock amount")
			}
			lockedCollaterals[tokenID][tokenCollateralId] = tokenValue - liquidatedAmounts[tokenCollateralId] - remainUnlockAmounts[tokenCollateralId]
			totalTokenCollaterals[tokenCollateralId] = totalTokenCollaterals[tokenCollateralId] - liquidatedAmounts[tokenCollateralId]
		}

		for tokenCollateralId, _ := range freeCollaterals {
			freeCollaterals[tokenCollateralId] += remainUnlockAmounts[tokenCollateralId]
		}

		custodianState.SetFreeTokenCollaterals(freeCollaterals)
		custodianState.SetLockedTokenCollaterals(lockedCollaterals)
		custodianState.SetTotalTokenCollaterals(totalTokenCollaterals)
	}

	return nil
}

func updateCustodianStateUnlockOverRateCollaterals(
	custodianState *statedb.CustodianState, unlockedAmount uint64, unlockedTokensAmount map[string]uint64, tokenID string) error {
	return updateCustodianStateAfterExpiredPortingReq(custodianState, unlockedAmount, unlockedTokensAmount, tokenID)
}

func updateCustodianStateAfterExpiredPortingReq(
	custodianState *statedb.CustodianState, unlockedAmount uint64, unlockedTokensAmount map[string]uint64, tokenID string) error {
	custodianState.SetFreeCollateral(custodianState.GetFreeCollateral() + unlockedAmount)

	if unlockedAmount > 0 {
		lockedAmountTmp := custodianState.GetLockedAmountCollateral()
		if lockedAmountTmp[tokenID] < unlockedAmount {
			return errors.New("[updateCustodianStateAfterExpiredPortingReq] locked amount custodian state less than token locked in porting request")
		}
		lockedAmountTmp[tokenID] -= unlockedAmount
		custodianState.SetLockedAmountCollateral(lockedAmountTmp)
	}

	if len(unlockedTokensAmount) > 0 {
		lockedTokensAmount := custodianState.GetLockedTokenCollaterals()
		freeTokensAmount := custodianState.GetFreeTokenCollaterals()
		for publicTokenId, tokenValue := range unlockedTokensAmount {
			if lockedTokensAmount[tokenID][publicTokenId] < tokenValue {
				return errors.New("[updateCustodianStateAfterExpiredPortingReq] locked amount custodian state less than token locked in porting request")
			}
			lockedTokensAmount[tokenID][publicTokenId] -= tokenValue
			freeTokensAmount[publicTokenId] += tokenValue
		}
		custodianState.SetLockedTokenCollaterals(lockedTokensAmount)
		custodianState.SetFreeTokenCollaterals(freeTokensAmount)
	}
	return nil
}

func removeCustodianFromMatchingRedeemCustodians(
	matchingCustodians []*statedb.MatchingRedeemCustodianDetail,
	custodianIncAddr string) ([]*statedb.MatchingRedeemCustodianDetail, error) {
	matchingCustodiansRes := make([]*statedb.MatchingRedeemCustodianDetail, len(matchingCustodians))
	copy(matchingCustodiansRes, matchingCustodians)

	for i, cus := range matchingCustodiansRes {
		if cus.GetIncognitoAddress() == custodianIncAddr {
			matchingCustodiansRes = append(matchingCustodiansRes[:i], matchingCustodiansRes[i+1:]...)
			return matchingCustodiansRes, nil
		}
	}
	return matchingCustodiansRes, errors.New("Custodian not found in matching redeem custodians")
}

func deleteWaitingRedeemRequest(state *CurrentPortalState, waitingRedeemRequestKey string) {
	delete(state.WaitingRedeemRequests, waitingRedeemRequestKey)
}

func deleteMatchedRedeemRequest(state *CurrentPortalState, matchedRedeemRequestKey string) {
	delete(state.MatchedRedeemRequests, matchedRedeemRequestKey)
}

func deleteWaitingPortingRequest(state *CurrentPortalState, waitingPortingRequestKey string) {
	delete(state.WaitingPortingRequests, waitingPortingRequestKey)
}

func updateCurrentPortalStateOfLiquidationExchangeRates(
	currentPortalState *CurrentPortalState,
	custodianKey string,
	custodianState *statedb.CustodianState,
	tpRatios map[string]portalMeta.LiquidateTopPercentileExchangeRatesDetail,
	remainUnlockAmounts map[string]uint64,
) {
	//update custodian state
	for pTokenId, tpRatioDetail := range tpRatios {
		holdingPubTokenTmp := custodianState.GetHoldingPublicTokens()
		holdingPubTokenTmp[pTokenId] -= tpRatioDetail.HoldAmountPubToken
		custodianState.SetHoldingPublicTokens(holdingPubTokenTmp)

		lockedAmountTmp := custodianState.GetLockedAmountCollateral()
		lockedAmountTmp[pTokenId] = lockedAmountTmp[pTokenId] - tpRatioDetail.HoldAmountFreeCollateral - remainUnlockAmounts[pTokenId]
		custodianState.SetLockedAmountCollateral(lockedAmountTmp)

		custodianState.SetTotalCollateral(custodianState.GetTotalCollateral() - tpRatioDetail.HoldAmountFreeCollateral)
	}

	totalRemainUnlockAmount := uint64(0)
	for _, amount := range remainUnlockAmounts {
		totalRemainUnlockAmount += amount
	}

	custodianState.SetFreeCollateral(custodianState.GetFreeCollateral() + totalRemainUnlockAmount)
	currentPortalState.CustodianPoolState[custodianKey] = custodianState
	//end

	//update LiquidateExchangeRates
	liquidateExchangeRatesKey := statedb.GeneratePortalLiquidationPoolObjectKey()
	liquidateExchangeRates, ok := currentPortalState.LiquidationPool[liquidateExchangeRatesKey.String()]
	if !ok {
		item := make(map[string]statedb.LiquidationPoolDetail)

		for ptoken, liquidateTopPercentileExchangeRatesDetail := range tpRatios {
			item[ptoken] = statedb.LiquidationPoolDetail{
				CollateralAmount: liquidateTopPercentileExchangeRatesDetail.HoldAmountFreeCollateral,
				PubTokenAmount:   liquidateTopPercentileExchangeRatesDetail.HoldAmountPubToken,
			}
		}
		currentPortalState.LiquidationPool[liquidateExchangeRatesKey.String()] = statedb.NewLiquidationPoolWithValue(item)
	} else {
		for ptoken, liquidateTopPercentileExchangeRatesDetail := range tpRatios {
			if _, ok := liquidateExchangeRates.Rates()[ptoken]; !ok {
				liquidateExchangeRates.Rates()[ptoken] = statedb.LiquidationPoolDetail{
					CollateralAmount: liquidateTopPercentileExchangeRatesDetail.HoldAmountFreeCollateral,
					PubTokenAmount:   liquidateTopPercentileExchangeRatesDetail.HoldAmountPubToken,
				}
			} else {
				liquidateExchangeRates.Rates()[ptoken] = statedb.LiquidationPoolDetail{
					CollateralAmount: liquidateExchangeRates.Rates()[ptoken].CollateralAmount + liquidateTopPercentileExchangeRatesDetail.HoldAmountFreeCollateral,
					PubTokenAmount:   liquidateExchangeRates.Rates()[ptoken].PubTokenAmount + liquidateTopPercentileExchangeRatesDetail.HoldAmountPubToken,
				}
			}
		}

		currentPortalState.LiquidationPool[liquidateExchangeRatesKey.String()] = liquidateExchangeRates
	}
	//end
}

func updateCurrentPortalStateAfterLiquidationByRatesV3(
	currentPortalState *CurrentPortalState,
	custodianKey string,
	liquidationInfo map[string]portalMeta.LiquidationByRatesDetailV3,
	remainUnlockAmounts map[string]portalMeta.RemainUnlockCollateral,
) {
	custodianState := currentPortalState.CustodianPoolState[custodianKey]

	freePRVCollateral := custodianState.GetFreeCollateral()
	freeTokenCollaterals := custodianState.GetFreeTokenCollaterals()

	// update custodian state
	for portalTokenID, lInfo := range liquidationInfo {
		// update hold public token amount
		holdingPubTokenTmp := custodianState.GetHoldingPublicTokens()
		holdingPubTokenTmp[portalTokenID] -= lInfo.LiquidatedPubTokenAmount
		custodianState.SetHoldingPublicTokens(holdingPubTokenTmp)

		remainUnlockByPortalTokenID := remainUnlockAmounts[portalTokenID]
		// update locked prv collateral and total prv collateral
		if lInfo.LiquidatedCollateralAmount > 0 || remainUnlockByPortalTokenID.PrvAmount > 0 {
			lockedAmountTmp := custodianState.GetLockedAmountCollateral()
			lockedAmountTmp[portalTokenID] = lockedAmountTmp[portalTokenID] - lInfo.LiquidatedCollateralAmount - remainUnlockByPortalTokenID.PrvAmount
			custodianState.SetLockedAmountCollateral(lockedAmountTmp)
			custodianState.SetTotalCollateral(custodianState.GetTotalCollateral() - lInfo.LiquidatedCollateralAmount)
		}

		// update locked token collaterals and total token collaterals
		if len(lInfo.LiquidatedTokenCollateralsAmount) > 0 || len(remainUnlockByPortalTokenID.TokenAmounts) > 0 {
			lockedTokenAmountTmp := custodianState.GetLockedTokenCollaterals()
			lockedTokenByPortalTokenID := lockedTokenAmountTmp[portalTokenID]
			totalTokenCollaterals := custodianState.GetTotalTokenCollaterals()
			for extTokenID := range lockedTokenByPortalTokenID {
				lockedTokenByPortalTokenID[extTokenID] = lockedTokenByPortalTokenID[extTokenID] - lInfo.LiquidatedTokenCollateralsAmount[extTokenID] - remainUnlockByPortalTokenID.TokenAmounts[extTokenID]
				totalTokenCollaterals[extTokenID] -= lInfo.LiquidatedTokenCollateralsAmount[extTokenID]
			}

			lockedTokenAmountTmp[portalTokenID] = lockedTokenByPortalTokenID
			custodianState.SetLockedTokenCollaterals(lockedTokenAmountTmp)
			custodianState.SetTotalTokenCollaterals(totalTokenCollaterals)
		}

		// update free collaterals
		freePRVCollateral += remainUnlockByPortalTokenID.PrvAmount
		for extTokenID, amount := range remainUnlockByPortalTokenID.TokenAmounts {
			freeTokenCollaterals[extTokenID] += amount
		}
	}
	custodianState.SetFreeCollateral(freePRVCollateral)
	custodianState.SetFreeTokenCollaterals(freeTokenCollaterals)
	currentPortalState.CustodianPoolState[custodianKey] = custodianState

	// update LiquidationPool
	liquidationPoolKey := statedb.GeneratePortalLiquidationPoolObjectKey()
	liquidationPool, ok := currentPortalState.LiquidationPool[liquidationPoolKey.String()]
	if !ok || liquidationPool == nil {
		liquidationPool = new(statedb.LiquidationPool)
	}
	if liquidationPool.Rates() == nil {
		liquidationPool.SetRates(map[string]statedb.LiquidationPoolDetail{})
	}

	liquidationPoolRates := liquidationPool.Rates()
	for portalTokenID, lInfo := range liquidationInfo {
		liquidatedTokenCollateralTmp := liquidationPoolRates[portalTokenID].TokensCollateralAmount
		if liquidatedTokenCollateralTmp == nil {
			liquidatedTokenCollateralTmp = map[string]uint64{}
		}
		for extTokenID, amount := range lInfo.LiquidatedTokenCollateralsAmount {
			liquidatedTokenCollateralTmp[extTokenID] += amount
		}

		liquidationPoolRates[portalTokenID] = statedb.LiquidationPoolDetail{
			CollateralAmount:       liquidationPoolRates[portalTokenID].CollateralAmount + lInfo.LiquidatedCollateralAmount,
			PubTokenAmount:         liquidationPoolRates[portalTokenID].PubTokenAmount + lInfo.LiquidatedPubTokenAmount,
			TokensCollateralAmount: liquidatedTokenCollateralTmp,
		}
	}
	liquidationPool.SetRates(liquidationPoolRates)
	currentPortalState.LiquidationPool[liquidationPoolKey.String()] = liquidationPool
}

func GetTotalLockedCollateralInEpoch(featureStateDB *statedb.StateDB) (uint64, error) {
	currentPortalState, err := InitCurrentPortalStateFromDB(featureStateDB)
	if err != nil {
		return 0, nil
	}

	return currentPortalState.LockedCollateralForRewards.GetTotalLockedCollateralForRewards(), nil
}

func calAndCheckTPRatio(
	portalState *CurrentPortalState,
	custodianState *statedb.CustodianState,
	finalExchange *statedb.FinalExchangeRatesState,
	portalParams portal.PortalParams) (map[string]portalMeta.LiquidateTopPercentileExchangeRatesDetail, error) {
	result := make(map[string]portalMeta.LiquidateTopPercentileExchangeRatesDetail)
	exchangeTool := NewPortalExchangeRateTool(finalExchange, portalParams)

	lockedAmount := make(map[string]uint64)
	for tokenID, amount := range custodianState.GetLockedAmountCollateral() {
		lockedAmount[tokenID] = amount
	}

	holdingPubToken := make(map[string]uint64)
	for tokenID := range custodianState.GetHoldingPublicTokens() {
		holdingPubToken[tokenID] = GetTotalHoldPubTokenAmount(portalState, custodianState, tokenID)
	}

	for _, waitingPortingReq := range portalState.WaitingPortingRequests {
		for _, matchingCus := range waitingPortingReq.Custodians() {
			if matchingCus.IncAddress == custodianState.GetIncognitoAddress() {
				lockedAmount[waitingPortingReq.TokenID()] -= matchingCus.LockedAmountCollateral
				break
			}
		}
	}

	tpListKeys := make([]string, 0)
	for key := range holdingPubToken {
		tpListKeys = append(tpListKeys, key)
	}
	sort.Strings(tpListKeys)
	for _, tokenID := range tpListKeys {
		amountPubToken := holdingPubToken[tokenID]
		amountPRV, ok := lockedAmount[tokenID]
		if !ok {
			Logger.log.Errorf("Invalid locked amount with tokenID %v\n", tokenID)
			return nil, fmt.Errorf("Invalid locked amount with tokenID %v", tokenID)
		}
		if amountPRV <= 0 || amountPubToken <= 0 {
			continue
		}

		// convert amountPubToken to PRV
		amountPTokenInPRV, err := exchangeTool.Convert(tokenID, common.PRVIDStr, amountPubToken)
		if err != nil || amountPTokenInPRV == 0 {
			Logger.log.Errorf("Error when convert exchange rate %v\n", err)
			return nil, fmt.Errorf("Error when convert exchange rate %v", err)
		}

		// amountPRV * 100 / amountPTokenInPRV
		tmp := new(big.Int).Mul(new(big.Int).SetUint64(amountPRV), big.NewInt(100))
		percent := new(big.Int).Div(tmp, new(big.Int).SetUint64(amountPTokenInPRV)).Uint64()

		if tp20, ok := checkTPRatio(percent, portalParams); ok {
			if tp20 {
				result[tokenID] = portalMeta.LiquidateTopPercentileExchangeRatesDetail{
					TPKey:                    int(portalParams.TP120),
					TPValue:                  percent,
					HoldAmountFreeCollateral: lockedAmount[tokenID],
					HoldAmountPubToken:       holdingPubToken[tokenID],
				}
			} else {
				result[tokenID] = portalMeta.LiquidateTopPercentileExchangeRatesDetail{
					TPKey:                    int(portalParams.TP130),
					TPValue:                  percent,
					HoldAmountFreeCollateral: 0,
					HoldAmountPubToken:       0,
				}
			}
		}
	}

	return result, nil
}

func UpdateCustodianStateAfterRejectRedeemRequestByLiquidation(portalState *CurrentPortalState, rejectedRedeemReq *statedb.RedeemRequest, beaconHeight uint64) error {
	tokenID := rejectedRedeemReq.GetTokenID()
	for _, matchingCus := range rejectedRedeemReq.GetCustodians() {
		custodianStateKey := statedb.GenerateCustodianStateObjectKey(matchingCus.GetIncognitoAddress())
		custodianStateKeyStr := custodianStateKey.String()
		custodianState := portalState.CustodianPoolState[custodianStateKeyStr]
		if custodianState == nil {
			return fmt.Errorf("Custodian not found %v", custodianStateKeyStr)
		}

		holdPubTokens := custodianState.GetHoldingPublicTokens()
		if holdPubTokens == nil {
			holdPubTokens = make(map[string]uint64, 0)
			holdPubTokens[tokenID] = matchingCus.GetAmount()
		} else {
			holdPubTokens[tokenID] += matchingCus.GetAmount()
		}

		portalState.CustodianPoolState[custodianStateKeyStr].SetHoldingPublicTokens(holdPubTokens)
	}

	return nil
}

func UpdateCustodianRewards(currentPortalState *CurrentPortalState, rewardInfos map[string]*statedb.PortalRewardInfo) {
	for custodianKey, custodianState := range currentPortalState.CustodianPoolState {
		custodianAddr := custodianState.GetIncognitoAddress()
		if rewardInfos[custodianAddr] == nil {
			continue
		}

		custodianReward := custodianState.GetRewardAmount()
		if custodianReward == nil {
			custodianReward = map[string]uint64{}
		}

		for tokenID, amount := range rewardInfos[custodianAddr].GetRewards() {
			custodianReward[tokenID] += amount
		}
		currentPortalState.CustodianPoolState[custodianKey].SetRewardAmount(custodianReward)
	}
}

// MatchCustodianToWaitingRedeemReq returns amount matching of custodian in redeem request if valid
func MatchCustodianToWaitingRedeemReq(
	custodianAddr string,
	redeemID string,
	portalState *CurrentPortalState) (uint64, bool, error) {
	// check redeemID is in waiting redeem requests or not
	keyWaitingRedeemRequest := statedb.GenerateWaitingRedeemRequestObjectKey(redeemID).String()
	waitingRedeemRequest := portalState.WaitingRedeemRequests[keyWaitingRedeemRequest]
	if waitingRedeemRequest == nil {
		return 0, false, fmt.Errorf("RedeemID is not existed in waiting matching redeem requests list %v\n", redeemID)
	}

	// check Incognito Address is an custodian or not
	custodianStateKey := statedb.GenerateCustodianStateObjectKey(custodianAddr).String()
	custodianState := portalState.CustodianPoolState[custodianStateKey]
	if custodianState == nil {
		return 0, false, fmt.Errorf("custodianState not found %v\n", custodianAddr)
	}

	// calculate amount need to be matched
	totalMatchedAmount := uint64(0)
	for _, cus := range waitingRedeemRequest.GetCustodians() {
		totalMatchedAmount += cus.GetAmount()
	}
	neededMatchingAmountInPToken := waitingRedeemRequest.GetRedeemAmount() - totalMatchedAmount
	if neededMatchingAmountInPToken <= 0 {
		return 0, false, errors.New("Amount need to be matched is not greater than zero")
	}

	holdPubTokenMap := custodianState.GetHoldingPublicTokens()
	if holdPubTokenMap == nil || len(holdPubTokenMap) == 0 {
		return 0, false, errors.New("Holding public token amount of custodian is not valid")
	}
	holdPubTokenAmount := holdPubTokenMap[waitingRedeemRequest.GetTokenID()]
	if holdPubTokenAmount == 0 {
		return 0, false, errors.New("Holding public token amount of custodian is not available")
	}

	if holdPubTokenAmount >= neededMatchingAmountInPToken {
		return neededMatchingAmountInPToken, true, nil
	} else {
		return holdPubTokenAmount, false, nil
	}
}

func UpdatePortalStateAfterCustodianReqMatchingRedeem(
	custodianAddr string,
	redeemID string,
	matchedAmount uint64,
	isEnoughCustodians bool,
	portalState *CurrentPortalState) (*statedb.RedeemRequest, error) {
	// check redeemID is in waiting redeem requests or not
	keyWaitingRedeemRequest := statedb.GenerateWaitingRedeemRequestObjectKey(redeemID).String()
	waitingRedeemRequest := portalState.WaitingRedeemRequests[keyWaitingRedeemRequest]
	if waitingRedeemRequest == nil {
		return nil, fmt.Errorf("RedeemID is not existed in waiting matching redeem requests list %v\n", redeemID)
	}

	// update custodian state
	custodianStateKey := statedb.GenerateCustodianStateObjectKey(custodianAddr).String()
	custodianState := portalState.CustodianPoolState[custodianStateKey]
	err := UpdateCustodianStateAfterMatchingRedeemReq(custodianState, matchedAmount, waitingRedeemRequest.GetTokenID())
	if err != nil {
		return nil, fmt.Errorf("Error wahne updating custodian state %v\n", err)
	}

	// update matching custodians in waiting redeem request
	matchingCus := waitingRedeemRequest.GetCustodians()
	if matchingCus == nil {
		matchingCus = make([]*statedb.MatchingRedeemCustodianDetail, 0)
	}
	matchingCus = append(matchingCus,
		statedb.NewMatchingRedeemCustodianDetailWithValue(custodianAddr, custodianState.GetRemoteAddresses()[waitingRedeemRequest.GetTokenID()], matchedAmount))
	waitingRedeemRequest.SetCustodians(matchingCus)

	if isEnoughCustodians {
		deleteWaitingRedeemRequest(portalState, keyWaitingRedeemRequest)
		keyMatchedRedeemRequest := statedb.GenerateMatchedRedeemRequestObjectKey(redeemID).String()
		portalState.MatchedRedeemRequests[keyMatchedRedeemRequest] = waitingRedeemRequest
	}

	return waitingRedeemRequest, nil
}

func UpdateCustodianStateAfterMatchingRedeemReq(custodianState *statedb.CustodianState, matchingAmount uint64, tokenID string) error {
	if custodianState == nil {
		return fmt.Errorf("custodianState not found %v\n", custodianState)
	}

	// update custodian state
	holdingPubTokenTmp := custodianState.GetHoldingPublicTokens()
	if holdingPubTokenTmp == nil {
		return errors.New("Holding public token of custodian is null")
	}
	if holdingPubTokenTmp[tokenID] < matchingAmount {
		return fmt.Errorf("Holding public token %v is less than matching amount %v : ", holdingPubTokenTmp[tokenID], matchingAmount)
	}
	holdingPubTokenTmp[tokenID] -= matchingAmount
	custodianState.SetHoldingPublicTokens(holdingPubTokenTmp)

	return nil
}

func UpdatePortalStateAfterPickMoreCustodiansForWaitingRedeemReq(
	moreCustodians []*statedb.MatchingRedeemCustodianDetail,
	waitingRedeem *statedb.RedeemRequest,
	portalState *CurrentPortalState) (*statedb.RedeemRequest, error) {
	// update custodian state
	for _, cus := range moreCustodians {
		custodianStateKey := statedb.GenerateCustodianStateObjectKey(cus.GetIncognitoAddress()).String()
		err := UpdateCustodianStateAfterMatchingRedeemReq(portalState.CustodianPoolState[custodianStateKey], cus.GetAmount(), waitingRedeem.GetTokenID())
		if err != nil {
			Logger.log.Errorf("Error when update custodian state for timeout redeem request %v\n", err)
			return nil, err
		}
	}

	// move waiting redeem request from waiting list to matched list
	waitingRedeemKey := statedb.GenerateWaitingRedeemRequestObjectKey(waitingRedeem.GetUniqueRedeemID()).String()
	deleteWaitingRedeemRequest(portalState, waitingRedeemKey)

	matchedCustodians := waitingRedeem.GetCustodians()
	if matchedCustodians == nil {
		matchedCustodians = make([]*statedb.MatchingRedeemCustodianDetail, 0)
	}
	matchedCustodians = append(matchedCustodians, moreCustodians...)
	waitingRedeem.SetCustodians(matchedCustodians)

	matchedRedeemKey := statedb.GenerateMatchedRedeemRequestObjectKey(waitingRedeem.GetUniqueRedeemID()).String()
	portalState.MatchedRedeemRequests[matchedRedeemKey] = waitingRedeem

	return waitingRedeem, nil
}

// get total porting token in waiting porting requests
func GetTotalLockedCollateralAmountInWaitingPortings(portalState *CurrentPortalState, custodianState *statedb.CustodianState, tokenID string) uint64 {
	totalLockedAmountInWaitingPortings := uint64(0)
	for _, waitingPortingReq := range portalState.WaitingPortingRequests {
		if waitingPortingReq.TokenID() != tokenID {
			continue
		}
		for _, cus := range waitingPortingReq.Custodians() {
			if cus.IncAddress == custodianState.GetIncognitoAddress() {
				totalLockedAmountInWaitingPortings += cus.LockedAmountCollateral
				break
			}
		}
	}

	return totalLockedAmountInWaitingPortings
}

// get total porting tokens in waiting porting requests v3
func GetTotalLockedCollateralAmountInWaitingPortingsV3(portalState *CurrentPortalState, custodianState *statedb.CustodianState, tokenID string) map[string]uint64 {
	var tokenAmountList map[string]uint64
	for _, waitingPortingReq := range portalState.WaitingPortingRequests {
		if waitingPortingReq.TokenID() != tokenID {
			continue
		}
		for _, cus := range waitingPortingReq.Custodians() {
			if cus.IncAddress == custodianState.GetIncognitoAddress() {
				tokenAmountList = cloneMap(cus.LockedTokenCollaterals)
				tokenAmountList[common.PRVIDStr] = cus.LockedAmountCollateral
				break
			}
		}
	}

	return tokenAmountList
}

// GetTotalHoldPubTokenAmount returns total holding public token amount (include both waiting and matched redeem requests)
func GetTotalHoldPubTokenAmount(portalState *CurrentPortalState, custodianState *statedb.CustodianState, tokenID string) uint64 {
	holdPubToken := custodianState.GetHoldingPublicTokens()
	totalHoldingPubTokenAmount := uint64(0)
	if holdPubToken != nil {
		totalHoldingPubTokenAmount += holdPubToken[tokenID]
	}

	for _, waitingRedeemReq := range portalState.WaitingRedeemRequests {
		if waitingRedeemReq.GetTokenID() != tokenID {
			continue
		}

		for _, cus := range waitingRedeemReq.GetCustodians() {
			if cus.GetIncognitoAddress() != custodianState.GetIncognitoAddress() {
				continue
			}
			totalHoldingPubTokenAmount += cus.GetAmount()
			break
		}
	}

	for _, matchedRedeemReq := range portalState.MatchedRedeemRequests {
		if matchedRedeemReq.GetTokenID() != tokenID {
			continue
		}

		for _, cus := range matchedRedeemReq.GetCustodians() {
			if cus.GetIncognitoAddress() != custodianState.GetIncognitoAddress() {
				continue
			}
			totalHoldingPubTokenAmount += cus.GetAmount()
			break
		}
	}

	return totalHoldingPubTokenAmount
}

func GetTotalHoldPubTokenAmountExcludeMatchedRedeemReqs(portalState *CurrentPortalState, custodianState *statedb.CustodianState, tokenID string) uint64 {
	//todo: check
	holdPubToken := custodianState.GetHoldingPublicTokens()
	totalHoldingPubTokenAmount := uint64(0)
	if holdPubToken != nil {
		totalHoldingPubTokenAmount += holdPubToken[tokenID]
	}

	for _, waitingRedeemReq := range portalState.WaitingRedeemRequests {
		if waitingRedeemReq.GetTokenID() != tokenID {
			continue
		}

		for _, cus := range waitingRedeemReq.GetCustodians() {
			if cus.GetIncognitoAddress() != custodianState.GetIncognitoAddress() {
				continue
			}
			totalHoldingPubTokenAmount += cus.GetAmount()
			break
		}
	}

	return totalHoldingPubTokenAmount
}

func GetTotalMatchingPubTokenInWaitingPortings(portalState *CurrentPortalState, custodianState *statedb.CustodianState, tokenID string) uint64 {
	totalMatchingPubTokenAmount := uint64(0)

	for _, waitingPortingReq := range portalState.WaitingPortingRequests {
		if waitingPortingReq.TokenID() != tokenID {
			continue
		}

		for _, cus := range waitingPortingReq.Custodians() {
			if cus.IncAddress != custodianState.GetIncognitoAddress() {
				continue
			}
			totalMatchingPubTokenAmount += cus.Amount
			break
		}
	}

	return totalMatchingPubTokenAmount
}

func UpdateLockedCollateralForRewards(currentPortalState *CurrentPortalState, portalParam portal.PortalParams) {
	exchangeTool := NewPortalExchangeRateTool(currentPortalState.FinalExchangeRatesState, portalParam)

	totalLockedCollateralAmount := currentPortalState.LockedCollateralForRewards.GetTotalLockedCollateralForRewards()
	lockedCollateralDetails := currentPortalState.LockedCollateralForRewards.GetLockedCollateralDetail()

	for _, custodianState := range currentPortalState.CustodianPoolState {
		lockedCollaterals := custodianState.GetLockedAmountCollateral()
		if lockedCollaterals == nil || len(lockedCollaterals) == 0 {
			continue
		}

		for tokenID := range lockedCollaterals {
			holdPubTokenAmount := GetTotalHoldPubTokenAmount(currentPortalState, custodianState, tokenID)
			matchingPubTokenAmount := GetTotalMatchingPubTokenInWaitingPortings(currentPortalState, custodianState, tokenID)
			totalPubToken := holdPubTokenAmount + matchingPubTokenAmount
			pubTokenAmountInPRV, err := exchangeTool.Convert(tokenID, common.PRVIDStr, totalPubToken)
			if err != nil {
				Logger.log.Errorf("Error when converting public token to prv: %v", err)
			}
			lockedCollateralDetails[custodianState.GetIncognitoAddress()] += pubTokenAmountInPRV
			totalLockedCollateralAmount += pubTokenAmountInPRV
		}
	}

	currentPortalState.LockedCollateralForRewards.SetTotalLockedCollateralForRewards(totalLockedCollateralAmount)
	currentPortalState.LockedCollateralForRewards.SetLockedCollateralDetail(lockedCollateralDetails)
}

func UpdateLockedCollateralForRewardsV3(currentPortalState *CurrentPortalState, portalParam portal.PortalParams) {
	exchangeTool := NewPortalExchangeRateTool(currentPortalState.FinalExchangeRatesState, portalParam)

	totalLockedCollateralAmount := currentPortalState.LockedCollateralForRewards.GetTotalLockedCollateralForRewards()
	lockedCollateralDetails := currentPortalState.LockedCollateralForRewards.GetLockedCollateralDetail()
	if lockedCollateralDetails == nil {
		lockedCollateralDetails = map[string]uint64{}
	}
	portalTokenIDs := pCommon.PortalSupportedIncTokenIDs
	for _, custodianState := range currentPortalState.CustodianPoolState {
		for _, tokenID := range portalTokenIDs {
			holdPubTokenAmount := GetTotalHoldPubTokenAmount(currentPortalState, custodianState, tokenID)
			if holdPubTokenAmount == 0 {
				continue
			}
			pubTokenAmountInUSDT, err := exchangeTool.ConvertToUSD(tokenID, holdPubTokenAmount)
			if err != nil {
				Logger.log.Errorf("Error when converting public token to prv: %v", err)
			}
			lockedCollateralDetails[custodianState.GetIncognitoAddress()] += pubTokenAmountInUSDT
			totalLockedCollateralAmount += pubTokenAmountInUSDT
		}
	}

	currentPortalState.LockedCollateralForRewards.SetTotalLockedCollateralForRewards(totalLockedCollateralAmount)
	currentPortalState.LockedCollateralForRewards.SetLockedCollateralDetail(lockedCollateralDetails)
}

func CalAmountTopUpWaitingPortings(
	portalState *CurrentPortalState,
	custodianState *statedb.CustodianState,
	portalParam portal.PortalParams,
	collateralTokenID string) (map[string]uint64, error) {

	result := make(map[string]uint64)
	exchangeTool := NewPortalExchangeRateTool(portalState.FinalExchangeRatesState, portalParam)

	for _, waitingPorting := range portalState.WaitingPortingRequests {
		for _, cus := range waitingPorting.Custodians() {
			if cus.IncAddress != custodianState.GetIncognitoAddress() {
				continue
			}

			// get total locked colalteral in waiting porting request
			lockedPRVCollateralInUSDT, err := exchangeTool.ConvertToUSD(common.PRVIDStr, cus.LockedAmountCollateral)
			if err != nil {
				Logger.log.Errorf("[calAmountTopUpWaitingPortings] Error when converting PRV to USDT %v", err)
				return nil, err
			}
			lockedTokenCollateralsInUSDT, err := exchangeTool.ConvertMapTokensToUSD(cus.LockedTokenCollaterals)
			if err != nil {
				Logger.log.Errorf("[calAmountTopUpWaitingPortings] Error when converting external tokens to USDT %v", err)
				return nil, err
			}
			totalLockedCollateralInUSDT := lockedPRVCollateralInUSDT + lockedTokenCollateralsInUSDT

			// get min locked collaterals in usdt
			minCollateralAmountInUSDT, err := exchangeTool.ConvertToUSD(
				waitingPorting.TokenID(),
				UpPercent(cus.Amount, portalParam.MinPercentLockedCollateral))
			if err != nil {
				Logger.log.Errorf("[calAmountTopUpWaitingPortings] Error when converting ptoken to PRV %v", err)
				return result, err
			}

			// calculate topup amount
			if minCollateralAmountInUSDT > cus.LockedAmountCollateral {
				if collateralTokenID == "" {
					collateralTokenID = common.PRVIDStr
				}
				topupAmountInUSDT := minCollateralAmountInUSDT - totalLockedCollateralInUSDT
				topupAmount, err := exchangeTool.ConvertFromUSD(collateralTokenID, topupAmountInUSDT)
				if err != nil {
					Logger.log.Errorf("[calAmountTopUpWaitingPortings] Error when converting topup amount to token %v", err)
					return result, err
				}

				result[waitingPorting.UniquePortingID()] = topupAmount
			}
			break
		}
	}

	return result, nil
}

func CloneRedeemRequests(redeemReqs map[string]*statedb.RedeemRequest) map[string]*statedb.RedeemRequest {
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
			req.ShardID(),
			req.ShardHeight(),
			req.GetRedeemerExternalAddress(),
		)
	}
	return newReqs
}

func getNewMatchedRedeemReqIDs(
	existRedeemReqs map[string]*statedb.RedeemRequest,
	newRedeemReqs map[string]*statedb.RedeemRequest) []string {
	newIDs := []string{}

	m := map[string]bool{}
	for _, req := range existRedeemReqs {
		m[req.GetUniqueRedeemID()] = true
	}

	for _, req := range newRedeemReqs {
		newID := req.GetUniqueRedeemID()
		if m[newID] == false {
			newIDs = append(newIDs, newID)
		}
	}

	return newIDs
}

func addCustodianToPool(
	custodianPool map[string]*statedb.CustodianState,
	custodianIncAddr string,
	depositAmount uint64,
	collateralTokenID string,
	remoteAddresses map[string]string,
) *statedb.CustodianState {
	keyCustodianState := statedb.GenerateCustodianStateObjectKey(custodianIncAddr)
	keyCustodianStateStr := keyCustodianState.String()
	existCustodian := custodianPool[keyCustodianStateStr]

	// check collateral token ID
	isPRVCollateral := collateralTokenID == common.PRVIDStr

	// the custodian hasn't deposited before
	newCustodian := statedb.NewCustodianState()
	if existCustodian == nil {
		newCustodian.SetIncognitoAddress(custodianIncAddr)
		newCustodian.SetRemoteAddresses(remoteAddresses)

		if isPRVCollateral {
			newCustodian.SetTotalCollateral(depositAmount)
			newCustodian.SetFreeCollateral(depositAmount)
		} else {
			totalTokenColaterals := map[string]uint64{
				collateralTokenID: depositAmount,
			}
			freeTokenCollaterals := map[string]uint64{
				collateralTokenID: depositAmount,
			}
			newCustodian.SetTotalTokenCollaterals(totalTokenColaterals)
			newCustodian.SetFreeTokenCollaterals(freeTokenCollaterals)
		}
	} else {
		newCustodian.SetIncognitoAddress(custodianIncAddr)
		newCustodian.SetHoldingPublicTokens(existCustodian.GetHoldingPublicTokens())
		newCustodian.SetLockedAmountCollateral(existCustodian.GetLockedAmountCollateral())
		newCustodian.SetLockedTokenCollaterals(existCustodian.GetLockedTokenCollaterals())
		newCustodian.SetRewardAmount(existCustodian.GetRewardAmount())

		updateRemoteAddresses := existCustodian.GetRemoteAddresses()
		if len(remoteAddresses) > 0 {
			// if total collateral is zero, custodians are able to update remote addresses
			if existCustodian.IsEmptyCollaterals() {
				updateRemoteAddresses = remoteAddresses
			} else {
				sortedTokenIDs := make([]string, 0)
				for tokenID := range remoteAddresses {
					sortedTokenIDs = append(sortedTokenIDs, tokenID)
				}

				for _, tokenID := range sortedTokenIDs {
					if updateRemoteAddresses[tokenID] != "" {
						continue
					}
					updateRemoteAddresses[tokenID] = remoteAddresses[tokenID]
				}
			}
		}
		newCustodian.SetRemoteAddresses(updateRemoteAddresses)

		if isPRVCollateral {
			newCustodian.SetTotalCollateral(existCustodian.GetTotalCollateral() + depositAmount)
			newCustodian.SetFreeCollateral(existCustodian.GetFreeCollateral() + depositAmount)
			newCustodian.SetTotalTokenCollaterals(existCustodian.GetTotalTokenCollaterals())
			newCustodian.SetFreeTokenCollaterals(existCustodian.GetFreeTokenCollaterals())
		} else {
			newCustodian.SetTotalCollateral(existCustodian.GetTotalCollateral())
			newCustodian.SetFreeCollateral(existCustodian.GetFreeCollateral())

			tmpTotalTokenCollaterals := existCustodian.GetTotalTokenCollaterals()
			if tmpTotalTokenCollaterals == nil {
				tmpTotalTokenCollaterals = map[string]uint64{}
			}
			tmpTotalTokenCollaterals[collateralTokenID] += depositAmount

			tmpFreeTokenCollaterals := existCustodian.GetFreeTokenCollaterals()
			if tmpFreeTokenCollaterals == nil {
				tmpFreeTokenCollaterals = map[string]uint64{}
			}
			tmpFreeTokenCollaterals[collateralTokenID] += depositAmount
			newCustodian.SetTotalTokenCollaterals(tmpTotalTokenCollaterals)
			newCustodian.SetFreeTokenCollaterals(tmpFreeTokenCollaterals)
		}
	}

	return newCustodian
}

func UpdateCustodianStateAfterWithdrawCollateral(
	custodian *statedb.CustodianState,
	collateralTokenID string,
	amount uint64) *statedb.CustodianState {
	if collateralTokenID == common.PRVIDStr {
		custodian.SetTotalCollateral(custodian.GetTotalCollateral() - amount)
		custodian.SetFreeCollateral(custodian.GetFreeCollateral() - amount)
	} else {
		freeTokenCollaterals := custodian.GetFreeTokenCollaterals()
		freeTokenCollaterals[collateralTokenID] -= amount
		totalTokenCollaterals := custodian.GetTotalTokenCollaterals()
		totalTokenCollaterals[collateralTokenID] -= amount

		custodian.SetTotalTokenCollaterals(totalTokenCollaterals)
		custodian.SetFreeTokenCollaterals(freeTokenCollaterals)
	}

	return custodian
}

/*
================== Portal v3 ==================
*/

// convertAllFreeCollateralsToUSDT converts all collaterals of custodian to USD
func convertAllFreeCollateralsToUSDT(convertRateTool *PortalExchangeRateTool, custodian *statedb.CustodianState) (uint64, error) {
	res := uint64(0)
	prvCollateralInUSDT, err := convertRateTool.ConvertToUSD(common.PRVIDStr, custodian.GetFreeCollateral())
	if err != nil {
		return 0, err
	}
	res += prvCollateralInUSDT

	tokenCollaterals := custodian.GetFreeTokenCollaterals()
	for tokenID, amount := range tokenCollaterals {
		amountInUSDT, err := convertRateTool.ConvertToUSD(tokenID, amount)
		if err != nil {
			return 0, err
		}

		res += amountInUSDT
	}
	return res, nil
}

// getAllLockCollateralsInUSD converts all lock collaterals of custodian to USD
func getAllLockCollateralsInUSD(convertRateTool *PortalExchangeRateTool, custodian *statedb.CustodianState) (map[string]uint64, error) {
	res := map[string]uint64{}

	// locked PRV collaterals
	lockedPRVCollateral := custodian.GetLockedAmountCollateral()
	if lockedPRVCollateral != nil && len(lockedPRVCollateral) > 0 {
		for portalTokenID, amount := range lockedPRVCollateral {
			prvCollateralInUSDT, err := convertRateTool.ConvertToUSD(common.PRVIDStr, amount)
			if err != nil {
				return nil, err
			}
			res[portalTokenID] += prvCollateralInUSDT
		}
	}

	// token collaterals
	tokenCollaterals := custodian.GetLockedTokenCollaterals()
	if tokenCollaterals == nil || len(tokenCollaterals) == 0 {
		return res, nil
	}

	for portalTokenID, m := range tokenCollaterals {
		tokenCollateralsInUSDT, err := convertRateTool.ConvertMapTokensToUSD(m)
		if err != nil {
			return nil, err
		}
		res[portalTokenID] += tokenCollateralsInUSDT
	}
	return res, nil
}

// getLockCollateralsInWPortingInUSD converts locked collaterals in waiting porting requests of custodian to USD
func getLockCollateralsInWPortingInUSD(
	convertRateTool *PortalExchangeRateTool,
	custodian *statedb.CustodianState,
	portalState *CurrentPortalState) (map[string]uint64, error) {
	res := map[string]uint64{}

	waitingPortingReqs := portalState.WaitingPortingRequests
	if len(waitingPortingReqs) == 0 {
		return res, nil
	}

	for _, wReq := range portalState.WaitingPortingRequests {
		portalTokenID := wReq.TokenID()
		for _, matchingCus := range wReq.Custodians() {
			if matchingCus.IncAddress == custodian.GetIncognitoAddress() {
				// prv collateral
				prvCollateralInUSDT, err := convertRateTool.ConvertToUSD(common.PRVIDStr, matchingCus.LockedAmountCollateral)
				if err != nil {
					Logger.log.Errorf("Error when convert lock collaterals in waiting porting to usdt: %v", err)
					return nil, fmt.Errorf("Error when convert lock collaterals in waiting porting to usdt: %v", err)
				}

				// token collaterals
				tokenCollateralInUSDT, err := convertRateTool.ConvertMapTokensToUSD(matchingCus.LockedTokenCollaterals)
				if err != nil {
					Logger.log.Errorf("Error when convert lock collaterals in waiting porting to usdt: %v", err)
					return nil, fmt.Errorf("Error when convert lock collaterals in waiting porting to usdt: %v", err)
				}
				res[portalTokenID] = res[portalTokenID] + prvCollateralInUSDT + tokenCollateralInUSDT
				break
			}
		}
	}
	return res, nil
}

// getLockCollateralsInUSDExcludeWPortings converts all locked collaterals (exclude locked in waitying porting) to USD
func getLockCollateralsInUSDExcludeWPortings(
	convertRateTool *PortalExchangeRateTool,
	custodian *statedb.CustodianState,
	portalState *CurrentPortalState) (map[string]uint64, error) {
	allLockCollaterals, err := getAllLockCollateralsInUSD(convertRateTool, custodian)
	if err != nil {
		Logger.log.Errorf("Error when convert lock collaterals exclude waiting porting to usdt: %v", err)
		return nil, fmt.Errorf("Error when convert lock collaterals exclude waiting porting to usdt: %v", err)
	}

	lockCollateralsInWPorting, err := getLockCollateralsInWPortingInUSD(convertRateTool, custodian, portalState)
	if err != nil {
		Logger.log.Errorf("Error when convert lock collaterals exclude waiting porting to usdt: %v", err)
		return nil, fmt.Errorf("Error when convert lock collaterals exclude waiting porting to usdt: %v", err)
	}

	for portalTokenID, _ := range allLockCollaterals {
		allLockCollaterals[portalTokenID] -= lockCollateralsInWPorting[portalTokenID]
	}
	return allLockCollaterals, nil
}

func GetHoldPubTokensByCustodian(portalState *CurrentPortalState, custodianState *statedb.CustodianState) (
	totalHoldPubToken map[string]uint64,
	holdPubTokenInWaitingRedeems map[string]uint64,
	holdPubTokenInMatchedRedeems map[string]uint64,
	waitingRedeemIDs []string) {
	totalHoldPubToken = map[string]uint64{}
	holdPubTokenInWaitingRedeems = map[string]uint64{}
	holdPubTokenInMatchedRedeems = map[string]uint64{}
	waitingRedeemIDs = []string{}

	// from custodian state
	for tokenID, amount := range custodianState.GetHoldingPublicTokens() {
		totalHoldPubToken[tokenID] += amount
	}

	// waiting redeem requests
	for _, waitingRedeemReq := range portalState.WaitingRedeemRequests {
		tokenID := waitingRedeemReq.GetTokenID()
		for _, cus := range waitingRedeemReq.GetCustodians() {
			if cus.GetIncognitoAddress() != custodianState.GetIncognitoAddress() {
				continue
			}
			totalHoldPubToken[tokenID] += cus.GetAmount()
			holdPubTokenInWaitingRedeems[tokenID] += cus.GetAmount()
			waitingRedeemIDs = append(waitingRedeemIDs, waitingRedeemReq.GetUniqueRedeemID())
			break
		}
	}

	for _, matchedRedeemReq := range portalState.MatchedRedeemRequests {
		tokenID := matchedRedeemReq.GetTokenID()
		for _, cus := range matchedRedeemReq.GetCustodians() {
			if cus.GetIncognitoAddress() != custodianState.GetIncognitoAddress() {
				continue
			}
			totalHoldPubToken[tokenID] += cus.GetAmount()
			holdPubTokenInMatchedRedeems[tokenID] += cus.GetAmount()
			break
		}
	}

	return
}

// calAndCheckLiquidationRatioV3 calculates between amount locked collaterals and amount holding public tokens
// if ratio < 120%, reject waiting redeem reqs that matched to the custodian
// and calculate liquidated amount
func calAndCheckLiquidationRatioV3(
	portalState *CurrentPortalState,
	custodianState *statedb.CustodianState,
	finalExchange *statedb.FinalExchangeRatesState,
	portalParams portal.PortalParams) (map[string]portalMeta.LiquidationByRatesDetailV3, map[string]portalMeta.RemainUnlockCollateral, []string, error) {
	result := make(map[string]portalMeta.LiquidationByRatesDetailV3)
	remainUnlockCollaterals := make(map[string]portalMeta.RemainUnlockCollateral)
	exchangeTool := NewPortalExchangeRateTool(finalExchange, portalParams)

	// locked collaterals in usdt exclude waiting porting requests
	lockedAmount, err := getLockCollateralsInUSDExcludeWPortings(exchangeTool, custodianState, portalState)
	if err != nil {
		Logger.log.Errorf("Error when calculating and checking liquidation ratio v3: %v", err)
		return nil, nil, nil, fmt.Errorf("Error when calculating and checking liquidation ratio v3: %v", err)
	}

	// get all holding public tokens (in both waiting redeem and matched redeem)
	totalHoldPubToken, _, holdPubTokenInMatchedRedeem, wRedeemIDs := GetHoldPubTokensByCustodian(portalState, custodianState)

	// convert totalHoldPubToken to USDT
	totalHoldPubTokenInUSDT := map[string]uint64{}
	for tokenId, amount := range totalHoldPubToken {
		amounInUSDT, err := exchangeTool.ConvertToUSD(tokenId, amount)
		if err != nil {
			Logger.log.Errorf("Error when calculating and checking liquidation ratio v3: %v", err)
			return nil, nil, nil, fmt.Errorf("Error when calculating and checking liquidation ratio v3: %v", err)
		}
		totalHoldPubTokenInUSDT[tokenId] = amounInUSDT
	}

	tokenIDs := make([]string, 0)
	for key := range totalHoldPubTokenInUSDT {
		tokenIDs = append(tokenIDs, key)
	}
	sort.Strings(tokenIDs)
	for _, tokenID := range tokenIDs {
		amountPubTokenInUSDT := totalHoldPubTokenInUSDT[tokenID]
		amountLockedCollateralInUSDT := lockedAmount[tokenID]

		if amountLockedCollateralInUSDT == 0 || amountPubTokenInUSDT == 0 {
			continue
		}

		// calculate ratio between amount locked collaterals and amount holding public tokens
		// amountLockedCollateralInUSDT * 100 / amountPTokenInPRV
		ratioBN := new(big.Int).Mul(new(big.Int).SetUint64(amountLockedCollateralInUSDT), big.NewInt(100))
		ratioBN = ratioBN.Div(ratioBN, new(big.Int).SetUint64(amountPubTokenInUSDT))
		ratio := ratioBN.Uint64()
		if ratio > portalParams.TP120 {
			continue
		}
		Logger.log.Infof("Custodian %v - PortalTokenID %v - Ratio %v", custodianState.GetIncognitoAddress(), tokenID, ratio)

		// calculate liquidated amount hold public tokens (exclude matched redeem reqs, because we don't liquidate matched redeem)
		// and liquidated amount locked collaterals
		liquidatedHoldPubTokenAmount := totalHoldPubToken[tokenID] - holdPubTokenInMatchedRedeem[tokenID]
		if liquidatedHoldPubTokenAmount == 0 {
			continue
		}

		custodianStateKey := statedb.GenerateCustodianStateObjectKey(custodianState.GetIncognitoAddress()).String()
		liquidatedPRVCollateral, remainUnlockPRVCollateral, liquidatedExtTokens, remainUnlockTokenCollaterals, err :=
			CalUnlockCollateralAmountAfterLiquidationV3(
				portalState, custodianStateKey, liquidatedHoldPubTokenAmount, tokenID, portalParams)
		if err != nil {
			Logger.log.Errorf("Error when calculating and checking liquidation ratio v3: %v", err)
			return nil, nil, nil, fmt.Errorf("Error when calculating and checking liquidation ratio v3: %v", err)
		}

		result[tokenID] = portalMeta.LiquidationByRatesDetailV3{
			Ratio:                            ratio,
			LiquidatedPubTokenAmount:         liquidatedHoldPubTokenAmount,
			LiquidatedCollateralAmount:       liquidatedPRVCollateral,
			LiquidatedTokenCollateralsAmount: liquidatedExtTokens,
		}

		remainUnlockCollaterals[tokenID] = portalMeta.RemainUnlockCollateral{
			PrvAmount:    remainUnlockPRVCollateral,
			TokenAmounts: remainUnlockTokenCollaterals,
		}
	}

	return result, remainUnlockCollaterals, wRedeemIDs, nil
}

func calUnlockedCollateralRedeemFromLiquidationPoolV3(
	redeemAmount uint64,
	lInfo statedb.LiquidationPoolDetail,
	exchangeTool PortalExchangeRateTool) (uint64, map[string]uint64, error) {

	if lInfo.PubTokenAmount == 0 {
		return 0, nil, errors.New("Liquidation pool is invalid")
	}

	// calculate total liquidated collaterals in usdt
	liquidatedPRVCollateralInUSDT, err := exchangeTool.ConvertToUSD(common.PRVIDStr, lInfo.CollateralAmount)
	if err != nil {
		return 0, nil, err
	}
	liquidatedTokenCollateralsInUSDT, err := exchangeTool.ConvertMapTokensToUSD(lInfo.TokensCollateralAmount)
	if err != nil {
		return 0, nil, err
	}
	liquidatedCollateralAmountInUSDT := liquidatedPRVCollateralInUSDT + liquidatedTokenCollateralsInUSDT

	// calculate unlocked collaterals by percent of redeemAmount
	unlockedAmountInUSDT := new(big.Int).Mul(
		new(big.Int).SetUint64(redeemAmount),
		new(big.Int).SetUint64(liquidatedCollateralAmountInUSDT),
	)
	unlockedAmountInUSDT = unlockedAmountInUSDT.Div(
		unlockedAmountInUSDT,
		new(big.Int).SetUint64(lInfo.PubTokenAmount))

	return exchangeTool.ConvertMapTokensFromUSD(unlockedAmountInUSDT.Uint64(), lInfo.CollateralAmount, lInfo.TokensCollateralAmount)
}

func UpdateLiquidationPoolAfterRedeemFrom(
	currentPortalState *CurrentPortalState,
	liquidationPool *statedb.LiquidationPool,
	portalTokenID string,
	redeemAmount uint64,
	mintedPRVCollateral uint64,
	unlockedTokenCollaterals map[string]uint64) {
	liquidationInfoByPortalTokenID := liquidationPool.Rates()[portalTokenID]

	updatedTokensCollateral := liquidationInfoByPortalTokenID.TokensCollateralAmount
	for tokenID, amount := range unlockedTokenCollaterals {
		updatedTokensCollateral[tokenID] -= amount
	}
	liquidationPool.Rates()[portalTokenID] = statedb.LiquidationPoolDetail{
		CollateralAmount:       liquidationInfoByPortalTokenID.CollateralAmount - mintedPRVCollateral,
		PubTokenAmount:         liquidationInfoByPortalTokenID.PubTokenAmount - redeemAmount,
		TokensCollateralAmount: updatedTokensCollateral,
	}
	liquidateExchangeRatesKey := statedb.GeneratePortalLiquidationPoolObjectKey().String()
	currentPortalState.LiquidationPool[liquidateExchangeRatesKey] = liquidationPool
}

// UpdateCustodianAfterTopup - v2 and v3
func UpdateCustodianAfterTopup(
	currentPortalState *CurrentPortalState,
	custodian *statedb.CustodianState,
	portalTokenID string,
	depositAmount uint64,
	freeCollateralAmount uint64,
	collateralTokenID string) (uint64, error) {

	topUpAmt := depositAmount + freeCollateralAmount
	if collateralTokenID == common.PRVIDStr {
		// v2: topup PRV collateral
		custodian.SetTotalCollateral(custodian.GetTotalCollateral() + depositAmount)
		if freeCollateralAmount > 0 {
			custodian.SetFreeCollateral(custodian.GetFreeCollateral() - freeCollateralAmount)
		}
		lockedPRVCollateral := custodian.GetLockedAmountCollateral()
		if lockedPRVCollateral == nil {
			lockedPRVCollateral = map[string]uint64{}
		}
		lockedPRVCollateral[portalTokenID] += topUpAmt
		custodian.SetLockedAmountCollateral(lockedPRVCollateral)
	} else {
		// v3: topup token collaterals
		totalTokenCollaterals := custodian.GetTotalTokenCollaterals()
		if totalTokenCollaterals == nil {
			return 0, errors.New("UpdateCustodianAfterTopup total token collaterals is empty")
		}
		totalTokenCollaterals[collateralTokenID] += depositAmount
		custodian.SetTotalTokenCollaterals(totalTokenCollaterals)

		if freeCollateralAmount > 0 {
			freeTokenCollaterals := custodian.GetFreeTokenCollaterals()
			if freeTokenCollaterals == nil {
				return 0, errors.New("UpdateCustodianAfterTopup free token collaterals is empty")
			}
			freeTokenCollaterals[collateralTokenID] -= freeCollateralAmount
			custodian.SetFreeTokenCollaterals(freeTokenCollaterals)
		}

		lockedTokenCollaterals := custodian.GetLockedTokenCollaterals()
		if lockedTokenCollaterals == nil {
			lockedTokenCollaterals = map[string]map[string]uint64{}
		}
		if lockedTokenCollaterals[portalTokenID] == nil {
			lockedTokenCollaterals[portalTokenID] = map[string]uint64{}
		}
		lockedTokenCollaterals[portalTokenID][collateralTokenID] += topUpAmt
		custodian.SetLockedTokenCollaterals(lockedTokenCollaterals)
	}

	custodianKeyStr := statedb.GenerateCustodianStateObjectKey(custodian.GetIncognitoAddress()).String()
	currentPortalState.CustodianPoolState[custodianKeyStr] = custodian
	return topUpAmt, nil
}

// UpdateCustodianAfterTopup - v2 and v3
func UpdateCustodianAfterTopupWaitingPorting(
	currentPortalState *CurrentPortalState,
	waitingPortingReq *statedb.WaitingPortingRequest,
	custodian *statedb.CustodianState,
	portalTokenID string,
	depositAmount uint64,
	freeCollateralAmount uint64,
	collateralTokenID string) error {

	// update custodian state
	topUpAmt, err := UpdateCustodianAfterTopup(currentPortalState, custodian, portalTokenID, depositAmount, freeCollateralAmount, collateralTokenID)
	if err != nil {
		return err
	}

	// update waiting porting req
	matchedCustodians := waitingPortingReq.Custodians()
	for _, cus := range matchedCustodians {
		if cus.IncAddress != custodian.GetIncognitoAddress() {
			continue
		}

		if collateralTokenID == common.PRVIDStr {
			cus.LockedAmountCollateral += topUpAmt
		} else {
			if cus.LockedTokenCollaterals == nil {
				cus.LockedTokenCollaterals = map[string]uint64{}
			}
			cus.LockedTokenCollaterals[collateralTokenID] += topUpAmt
		}
		waitingPortingReq.SetCustodians(matchedCustodians)
		break
	}
	return nil
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

func cloneMapOfMap(m map[string]map[string]uint64) map[string]map[string]uint64 {
	if m == nil {
		return nil
	}
	newMap := make(map[string]map[string]uint64, len(m))
	for k, v := range m {
		newMap[k] = cloneMap(v)
	}
	return newMap
}

func CloneCustodians(custodians map[string]*statedb.CustodianState) map[string]*statedb.CustodianState {
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
			cloneMap(cus.GetTotalTokenCollaterals()),
			cloneMap(cus.GetFreeTokenCollaterals()),
			cloneMapOfMap(cus.GetLockedTokenCollaterals()),
		)
	}
	return newCustodians
}

func CloneMultisigWallet(wallets map[string]*statedb.MultisigWalletsState) map[string]*statedb.MultisigWalletsState {
	newWallets := make(map[string]*statedb.MultisigWalletsState, len(wallets))
	for key, wallet := range wallets {
		newWallets[key] = statedb.NewMultisigWalletsStateWithValue(
			wallet.GetListUTXO(),
		)
	}
	return newWallets
}

func cloneMatchingPortingCustodians(custodians []*statedb.MatchingPortingCustodianDetail) []*statedb.MatchingPortingCustodianDetail {
	newMatchingCustodians := make([]*statedb.MatchingPortingCustodianDetail, len(custodians))
	for i, cus := range custodians {
		newMatchingCustodians[i] = &statedb.MatchingPortingCustodianDetail{
			IncAddress:             cus.IncAddress,
			RemoteAddress:          cus.RemoteAddress,
			Amount:                 cus.Amount,
			LockedAmountCollateral: cus.LockedAmountCollateral,
			LockedTokenCollaterals: cus.LockedTokenCollaterals,
		}
	}
	return newMatchingCustodians
}

func CloneWPortingRequests(wPortingReqs map[string]*statedb.WaitingPortingRequest) map[string]*statedb.WaitingPortingRequest {
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
			req.ShardHeight(),
			req.ShardID(),
		)
	}
	return newReqs
}
