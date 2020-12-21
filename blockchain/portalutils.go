package blockchain

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/metadata/rpccaller"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
	"math"
	"math/big"
	"sort"
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
	ExchangeRatesRequests map[string]*metadata.ExchangeRatesRequestStatus // key : hash(beaconHeight | TxID)
}

type CustodianStateSlice struct {
	Key   string
	Value *statedb.CustodianState
}

type RedeemMemoBNB struct {
	RedeemID                  string `json:"RedeemID"`
	CustodianIncognitoAddress string `json:"CustodianIncognitoAddress"`
}

type PortingMemoBNB struct {
	PortingID string `json:"PortingID"`
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
		ExchangeRatesRequests:      make(map[string]*metadata.ExchangeRatesRequestStatus),
		LiquidationPool:            liquidateExchangeRatesPool,
		LockedCollateralForRewards: lockedCollateralState,
	}, nil
}

func storePortalStateToDB(
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

func sortCustodianByAmountAscent(
	metadata metadata.PortalUserRegister,
	custodianState map[string]*statedb.CustodianState,
	custodianStateSlice *[]CustodianStateSlice) {
	//convert to slice

	var result []CustodianStateSlice
	for k, v := range custodianState {
		//check pTokenId, select only ptokenid
		if v.GetRemoteAddresses()[metadata.PTokenId] == "" {
			continue
		}

		item := CustodianStateSlice{
			Key:   k,
			Value: v,
		}
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Value.GetFreeCollateral() < result[j].Value.GetFreeCollateral() {
			return true
		} else if (result[i].Value.GetFreeCollateral() == result[j].Value.GetFreeCollateral()) &&
			(result[i].Value.GetIncognitoAddress() < result[j].Value.GetIncognitoAddress()) {
			return true
		}
		return false
	})

	*custodianStateSlice = result
}

func pickUpCustodians(
	metadata metadata.PortalUserRegister,
	exchangeRate *statedb.FinalExchangeRatesState,
	custodianStateSlice []CustodianStateSlice,
	currentPortalState *CurrentPortalState,
	portalParams PortalParams,
) ([]*statedb.MatchingPortingCustodianDetail, error) {
	//get multiple custodian
	custodians := make([]*statedb.MatchingPortingCustodianDetail, 0)
	remainPTokens := metadata.RegisterAmount
	convertExchangeRatesObj := NewConvertExchangeRatesObject(exchangeRate)
	for i := len(custodianStateSlice) - 1; i >= 0; i-- {
		custodianItem := custodianStateSlice[i]
		freeCollaterals := custodianItem.Value.GetFreeCollateral()
		if freeCollaterals == 0 {
			continue
		}

		Logger.log.Infof("Porting request, pick multiple custodian key: %v, has collateral %v", custodianItem.Key, freeCollaterals)

		collateralsInPToken, err := convertExchangeRatesObj.ExchangePRV2PTokenByTokenId(metadata.PTokenId, freeCollaterals)
		if err != nil {
			Logger.log.Errorf("Failed to convert prv collaterals to PToken - with error %v", err)
			return nil, err
		}

		pTokenCustodianCanHold := down150Percent(collateralsInPToken, uint64(portalParams.MinPercentLockedCollateral))
		if pTokenCustodianCanHold > remainPTokens {
			pTokenCustodianCanHold = remainPTokens
			Logger.log.Infof("Porting request, custodian key: %v, ptoken amount is more larger than remain so custodian can keep ptoken  %v", custodianItem.Key, pTokenCustodianCanHold)
		} else {
			Logger.log.Infof("Porting request, pick multiple custodian key: %v, can keep ptoken %v", custodianItem.Key, pTokenCustodianCanHold)
		}

		totalPTokenAfterUp150Percent := up150Percent(pTokenCustodianCanHold, uint64(portalParams.MinPercentLockedCollateral))
		neededCollaterals, err := convertExchangeRatesObj.ExchangePToken2PRVByTokenId(metadata.PTokenId, totalPTokenAfterUp150Percent)
		if err != nil {
			Logger.log.Errorf("Failed to convert PToken to prv - with error %v", err)
			return nil, err
		}
		Logger.log.Infof("Porting request, custodian key: %v, to keep ptoken %v need prv %v", custodianItem.Key, pTokenCustodianCanHold, neededCollaterals)

		if freeCollaterals >= neededCollaterals {
			remoteAddr := custodianItem.Value.GetRemoteAddresses()[metadata.PTokenId]
			if remoteAddr == "" {
				Logger.log.Errorf("Remote address in tokenID %v of custodian %v is null", metadata.PTokenId, custodianItem.Value.GetIncognitoAddress())
				return nil, fmt.Errorf("Remote address in tokenID %v of custodian %v is null", metadata.PTokenId, custodianItem.Value.GetIncognitoAddress())
			}
			custodians = append(
				custodians,
				&statedb.MatchingPortingCustodianDetail{
					IncAddress:             custodianItem.Value.GetIncognitoAddress(),
					RemoteAddress:          remoteAddr,
					Amount:                 pTokenCustodianCanHold,
					LockedAmountCollateral: neededCollaterals,
				},
			)

			remainPTokens = remainPTokens - pTokenCustodianCanHold
			if remainPTokens == 0 {
				break
			}
		}
	}

	if remainPTokens  > 0 {
		Logger.log.Errorf("Not enough custodians for porting amount %v", metadata.RegisterAmount)
		return nil, fmt.Errorf("Not enough custodians for porting amount %v", metadata.RegisterAmount)
	}

	return custodians, nil
}

func UpdateCustodianStateAfterMatchingPortingRequest(currentPortalState *CurrentPortalState, custodianKey string, PTokenId string, lockedAmountCollateral uint64) error {
	custodian, ok := currentPortalState.CustodianPoolState[custodianKey]
	if !ok {
		return errors.New("Custodian not found")
	}

	freeCollateral := custodian.GetFreeCollateral() - lockedAmountCollateral
	custodian.SetFreeCollateral(freeCollateral)

	// don't update holding public tokens to avoid this custodian match to redeem request before receiving pubtokens from users

	//update collateral holded
	if custodian.GetLockedAmountCollateral() == nil {
		totalLockedAmountCollateral := make(map[string]uint64)
		totalLockedAmountCollateral[PTokenId] = lockedAmountCollateral
		custodian.SetLockedAmountCollateral(totalLockedAmountCollateral)
	} else {
		lockedAmount := custodian.GetLockedAmountCollateral()
		lockedAmount[PTokenId] = lockedAmount[PTokenId] + lockedAmountCollateral
		custodian.SetLockedAmountCollateral(lockedAmount)
	}

	currentPortalState.CustodianPoolState[custodianKey] = custodian

	return nil
}
func UpdateCustodianStateAfterUserRequestPToken(currentPortalState *CurrentPortalState, custodianKey string, PTokenId string, amountPToken uint64) error {
	custodian, ok := currentPortalState.CustodianPoolState[custodianKey]
	if !ok {
		return errors.New("[UpdateCustodianStateAfterUserRequestPToken] Custodian not found")
	}

	holdingPubTokensTmp := custodian.GetHoldingPublicTokens()
	if holdingPubTokensTmp == nil {
		holdingPubTokensTmp = make(map[string]uint64)
		holdingPubTokensTmp[PTokenId] = amountPToken
	} else {
		holdingPubTokensTmp[PTokenId] += amountPToken
	}
	currentPortalState.CustodianPoolState[custodianKey].SetHoldingPublicTokens(holdingPubTokensTmp)
	return nil
}

func CalculatePortingFees(totalPToken uint64, minPercent float64) uint64 {
	result := minPercent * float64(totalPToken) / 100
	roundNumber := math.Round(result)
	return uint64(roundNumber)
}

func CalMinPortingFee(portingAmountInPToken uint64, tokenSymbol string, exchangeRate *statedb.FinalExchangeRatesState, minPercent float64) (uint64, error) {
	convertExchangeRatesObj := NewConvertExchangeRatesObject(exchangeRate)
	portingAmountInPRV, err := convertExchangeRatesObj.ExchangePToken2PRVByTokenId(tokenSymbol, portingAmountInPToken)
	if err != nil {
		Logger.log.Errorf("Error when calculating minimum porting fee %v", err)
		return 0, err
	}

	// can't use big int to calculate porting fee because of common.MinPercentPortingFee < 1
	portingFee := uint64(math.Round(float64(portingAmountInPRV) * minPercent / 100))

	return portingFee, nil
}

func CalMinRedeemFee(redeemAmountInPToken uint64, tokenSymbol string, exchangeRate *statedb.FinalExchangeRatesState, minPercent float64) (uint64, error) {
	convertExchangeRatesObj := NewConvertExchangeRatesObject(exchangeRate)
	redeemAmountInPRV, err := convertExchangeRatesObj.ExchangePToken2PRVByTokenId(tokenSymbol, redeemAmountInPToken)
	if err != nil {
		Logger.log.Errorf("Error when calculating minimum redeem fee %v", err)
		return 0, err
	}

	// can't use big int to calculate porting fee because of common.MinPercentRedeemFee < 1
	redeemFee := uint64(math.Round(float64(redeemAmountInPRV) * minPercent / 100))

	return redeemFee, nil
}

/*
	up 150%
*/
func up150Percent(amount uint64, percent uint64) uint64 {
	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amount), new(big.Int).SetUint64(percent))
	result := new(big.Int).Div(tmp, new(big.Int).SetUint64(100)).Uint64()
	return result //return nano pBTC, pBNB
}

func down150Percent(amount uint64, percent uint64) uint64 {
	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amount), new(big.Int).SetUint64(100))
	result := new(big.Int).Div(tmp, new(big.Int).SetUint64(percent)).Uint64()
	return result
}

func calTotalLiquidationByExchangeRates(RedeemAmount uint64, liquidateExchangeRates statedb.LiquidationPoolDetail) (uint64, error) {
	//todo: need review divide operator
	// prv  ------   total token
	// ?		     amount token

	if liquidateExchangeRates.PubTokenAmount <= 0 {
		return 0, errors.New("Can not divide 0")
	}

	tmp := new(big.Int).Mul(
		new(big.Int).SetUint64(liquidateExchangeRates.CollateralAmount),
		new(big.Int).SetUint64(RedeemAmount),
	)
	totalPrv := new(big.Int).Div(
		tmp,
		new(big.Int).SetUint64(liquidateExchangeRates.PubTokenAmount),
	)
	return totalPrv.Uint64(), nil
}

//check value is tp120 or tp130
func checkTPRatio(tpValue uint64, portalParams PortalParams) (bool, bool) {
	if tpValue > portalParams.TP120 && tpValue <= portalParams.TP130 {
		return false, true
	}

	if tpValue <= portalParams.TP120 {
		return true, true
	}

	//not found
	return false, false
}

func CalAmountNeededDepositLiquidate(currentPortalState *CurrentPortalState, custodian *statedb.CustodianState, exchangeRates *statedb.FinalExchangeRatesState, pTokenId string, portalParams PortalParams) (uint64, error) {
	if custodian.GetHoldingPublicTokens() == nil {
		return 0, nil
	}
	totalHoldingPublicToken := GetTotalHoldPubTokenAmount(currentPortalState, custodian, pTokenId)
	totalPToken := up150Percent(totalHoldingPublicToken, portalParams.MinPercentLockedCollateral)
	convertExchangeRatesObj := NewConvertExchangeRatesObject(exchangeRates)
	totalPRV, err := convertExchangeRatesObj.ExchangePToken2PRVByTokenId(pTokenId, totalPToken)
	if err != nil {
		return 0, err
	}

	totalLockedAmountInWaitingPorting := GetTotalLockedCollateralAmountInWaitingPortings(currentPortalState, custodian, pTokenId)

	lockedAmountMap := custodian.GetLockedAmountCollateral()
	if lockedAmountMap == nil {
		return 0, errors.New("Locked amount is nil")
	}
	lockedAmount := lockedAmountMap[pTokenId] - totalLockedAmountInWaitingPorting
	if lockedAmount >= totalPRV {
		return 0, nil
	}

	totalAmountNeeded := totalPRV - lockedAmount

	return totalAmountNeeded, nil
}

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

// convertIncPBNBAmountToExternalBNBAmount converts amount in inc chain (decimal 9) to amount in bnb chain (decimal 8)
func convertIncPBNBAmountToExternalBNBAmount(incPBNBAmount int64) int64 {
	return incPBNBAmount / 10 // incPBNBAmount / 1^9 * 1^8
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

func CalUnlockCollateralAmountAfterLiquidation(
	portalState *CurrentPortalState,
	liquidatedCustodianStateKey string,
	amountPubToken uint64,
	tokenID string,
	exchangeRate *statedb.FinalExchangeRatesState,
	portalParams PortalParams) (uint64, uint64, error) {
	totalUnlockCollateralAmount, err := CalUnlockCollateralAmount(portalState, liquidatedCustodianStateKey, amountPubToken, tokenID)
	if err != nil {
		Logger.log.Errorf("CalUnlockCollateralAmountAfterLiquidation error : %v\n", err)
		return 0, 0, err
	}
	convertExchangeRatesObj := NewConvertExchangeRatesObject(exchangeRate)
	tmp := new(big.Int).Mul(new(big.Int).SetUint64(amountPubToken), new(big.Int).SetUint64(uint64(portalParams.MaxPercentLiquidatedCollateralAmount)))
	liquidatedAmountInPToken := new(big.Int).Div(tmp, new(big.Int).SetUint64(100)).Uint64()
	liquidatedAmountInPRV, err := convertExchangeRatesObj.ExchangePToken2PRVByTokenId(tokenID, liquidatedAmountInPToken)
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

// updateRedeemRequestStatusByRedeemId updates status of redeem request into db
func updateRedeemRequestStatusByRedeemId(redeemID string, newStatus int, db *statedb.StateDB) error {
	redeemRequestBytes, err := statedb.GetPortalRedeemRequestStatus(db, redeemID)
	if err != nil {
		return err
	}
	if len(redeemRequestBytes) == 0 {
		return fmt.Errorf("Not found redeem request from db with redeemId %v\n", redeemID)
	}

	var redeemRequest metadata.PortalRedeemRequestStatus
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

func updateCustodianStateAfterExpiredPortingReq(
	custodianState *statedb.CustodianState, unlockedAmount uint64, tokenID string) {
	custodianState.SetFreeCollateral(custodianState.GetFreeCollateral() + unlockedAmount)

	lockedAmountTmp := custodianState.GetLockedAmountCollateral()
	lockedAmountTmp[tokenID] -= unlockedAmount
	custodianState.SetLockedAmountCollateral(lockedAmountTmp)
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

type ConvertExchangeRatesObject struct {
	finalExchangeRates *statedb.FinalExchangeRatesState
}

func NewConvertExchangeRatesObject(finalExchangeRates *statedb.FinalExchangeRatesState) *ConvertExchangeRatesObject {
	return &ConvertExchangeRatesObject{finalExchangeRates: finalExchangeRates}
}

func (c ConvertExchangeRatesObject) ExchangePToken2PRVByTokenId(pTokenId string, value uint64) (uint64, error) {
	switch pTokenId {
	case common.PortalBTCIDStr:
		result, err := c.ExchangeBTC2PRV(value)
		if err != nil {
			return 0, err
		}

		return result, nil
	case common.PortalBNBIDStr:
		result, err := c.ExchangeBNB2PRV(value)
		if err != nil {
			return 0, err
		}

		return result, nil
	}

	return 0, errors.New("Ptoken is not support")
}

func (c *ConvertExchangeRatesObject) ExchangePRV2PTokenByTokenId(pTokenId string, value uint64) (uint64, error) {
	switch pTokenId {
	case common.PortalBTCIDStr:
		return c.ExchangePRV2BTC(value)
	case common.PortalBNBIDStr:
		return c.ExchangePRV2BNB(value)
	}

	return 0, errors.New("Ptoken is not support")
}

func (c *ConvertExchangeRatesObject) convert(value uint64, ratesFrom uint64, RatesTo uint64) (uint64, error) {
	//convert to pusdt
	total := new(big.Int).Mul(new(big.Int).SetUint64(value), new(big.Int).SetUint64(ratesFrom))

	if RatesTo <= 0 {
		return 0, errors.New("Can not divide zero")
	}

	//pusdt -> new coin
	roundNumber := new(big.Int).Div(total, new(big.Int).SetUint64(RatesTo))
	return roundNumber.Uint64(), nil

}

func (c *ConvertExchangeRatesObject) ExchangeBTC2PRV(value uint64) (uint64, error) {
	//input : nano
	//todo: check rates exist
	BTCRates := c.finalExchangeRates.Rates()[common.PortalBTCIDStr].Amount //return nano pUSDT
	PRVRates := c.finalExchangeRates.Rates()[common.PRVIDStr].Amount       //return nano pUSDT
	valueExchange, err := c.convert(value, BTCRates, PRVRates)

	if err != nil {
		return 0, err
	}
	//nano
	return valueExchange, nil
}

func (c *ConvertExchangeRatesObject) ExchangeBNB2PRV(value uint64) (uint64, error) {
	BNBRates := c.finalExchangeRates.Rates()[common.PortalBNBIDStr].Amount
	PRVRates := c.finalExchangeRates.Rates()[common.PRVIDStr].Amount

	valueExchange, err := c.convert(value, BNBRates, PRVRates)

	if err != nil {
		return 0, err
	}

	return valueExchange, nil
}

func (c *ConvertExchangeRatesObject) ExchangePRV2BTC(value uint64) (uint64, error) {
	//input nano
	BTCRates := c.finalExchangeRates.Rates()[common.PortalBTCIDStr].Amount //return nano pUSDT
	PRVRates := c.finalExchangeRates.Rates()[common.PRVIDStr].Amount       //return nano pUSDT

	valueExchange, err := c.convert(value, PRVRates, BTCRates)

	if err != nil {
		return 0, err
	}

	return valueExchange, nil
}

func (c *ConvertExchangeRatesObject) ExchangePRV2BNB(value uint64) (uint64, error) {
	BNBRates := c.finalExchangeRates.Rates()[common.PortalBNBIDStr].Amount
	PRVRates := c.finalExchangeRates.Rates()[common.PRVIDStr].Amount

	valueExchange, err := c.convert(value, PRVRates, BNBRates)
	if err != nil {
		return 0, err
	}
	return valueExchange, nil
}

func updateCurrentPortalStateOfLiquidationExchangeRates(
	currentPortalState *CurrentPortalState,
	custodianKey string,
	custodianState *statedb.CustodianState,
	tpRatios map[string]metadata.LiquidateTopPercentileExchangeRatesDetail,
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

func getTotalLockedCollateralInEpoch(featureStateDB *statedb.StateDB) (uint64, error) {
	currentPortalState, err := InitCurrentPortalStateFromDB(featureStateDB)
	if err != nil {
		return 0, nil
	}

	return currentPortalState.LockedCollateralForRewards.GetTotalLockedCollateralForRewards(), nil
}

// GetBNBHeader calls RPC to fullnode bnb to get bnb header by block height
func (blockchain *BlockChain) GetBNBHeader(
	blockHeight int64,
) (*types.Header, error) {
	bnbFullNodeAddress := rpccaller.BuildRPCServerAddress(
		blockchain.GetConfig().ChainParams.BNBFullNodeProtocol,
		blockchain.GetConfig().ChainParams.BNBFullNodeHost,
		blockchain.GetConfig().ChainParams.BNBFullNodePort,
	)
	bnbClient := client.NewHTTP(bnbFullNodeAddress, "/websocket")
	result, err := bnbClient.Block(&blockHeight)
	if err != nil {
		Logger.log.Errorf("An error occured during calling status method: %s", err)
		return nil, fmt.Errorf("error occured during calling status method: %s", err)
	}
	return &result.Block.Header, nil
}

// GetBNBHeader calls RPC to fullnode bnb to get bnb data hash in header
func (blockchain *BlockChain) GetBNBDataHash(
	blockHeight int64,
) ([]byte, error) {
	header, err := blockchain.GetBNBHeader(blockHeight)
	if err != nil {
		return nil, err
	}
	if header.DataHash == nil {
		return nil, errors.New("Data hash is nil")
	}
	return header.DataHash, nil
}

// GetBNBHeader calls RPC to fullnode bnb to get latest bnb block height
func (blockchain *BlockChain) GetLatestBNBBlkHeight() (int64, error) {
	bnbFullNodeAddress := rpccaller.BuildRPCServerAddress(
		blockchain.GetConfig().ChainParams.BNBFullNodeProtocol,
		blockchain.GetConfig().ChainParams.BNBFullNodeHost,
		blockchain.GetConfig().ChainParams.BNBFullNodePort)
	bnbClient := client.NewHTTP(bnbFullNodeAddress, "/websocket")
	result, err := bnbClient.Status()
	if err != nil {
		Logger.log.Errorf("An error occured during calling status method: %s", err)
		return 0, fmt.Errorf("error occured during calling status method: %s", err)
	}
	return result.SyncInfo.LatestBlockHeight, nil
}

func calAndCheckTPRatio(
	portalState *CurrentPortalState,
	custodianState *statedb.CustodianState,
	finalExchange *statedb.FinalExchangeRatesState,
	portalParams PortalParams) (map[string]metadata.LiquidateTopPercentileExchangeRatesDetail, error) {
	result := make(map[string]metadata.LiquidateTopPercentileExchangeRatesDetail)
	convertExchangeRatesObj := NewConvertExchangeRatesObject(finalExchange)

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
		amountPTokenInPRV, err := convertExchangeRatesObj.ExchangePToken2PRVByTokenId(tokenID, amountPubToken)
		if err != nil || amountPTokenInPRV == 0 {
			Logger.log.Errorf("Error when convert exchange rate %v\n", err)
			return nil, fmt.Errorf("Error when convert exchange rate %v", err)
		}

		// amountPRV * 100 / amountPTokenInPRV
		tmp := new(big.Int).Mul(new(big.Int).SetUint64(amountPRV), big.NewInt(100))
		percent := new(big.Int).Div(tmp, new(big.Int).SetUint64(amountPTokenInPRV)).Uint64()

		if tp20, ok := checkTPRatio(percent, portalParams); ok {
			if tp20 {
				result[tokenID] = metadata.LiquidateTopPercentileExchangeRatesDetail{
					TPKey:                    int(portalParams.TP120),
					TPValue:                  percent,
					HoldAmountFreeCollateral: lockedAmount[tokenID],
					HoldAmountPubToken:       holdingPubToken[tokenID],
				}
			} else {
				result[tokenID] = metadata.LiquidateTopPercentileExchangeRatesDetail{
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
	// check Incognito Address is an custodian or not
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

func UpdateLockedCollateralForRewards(currentPortalState *CurrentPortalState) {
	exchangeRate := NewConvertExchangeRatesObject(currentPortalState.FinalExchangeRatesState)

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
			pubTokenAmountInPRV, err := exchangeRate.ExchangePToken2PRVByTokenId(tokenID, totalPubToken)
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

func CalAmountTopUpWaitingPortings(
	portalState *CurrentPortalState,
	custodianState *statedb.CustodianState, portalParam PortalParams) (map[string]uint64, error) {

	result := make(map[string]uint64)
	convertExchangeRatesObj := NewConvertExchangeRatesObject(portalState.FinalExchangeRatesState)

	for _, waitingPorting := range portalState.WaitingPortingRequests {
		for _, cus := range waitingPorting.Custodians() {
			if cus.IncAddress != custodianState.GetIncognitoAddress() {
				continue
			}

			minCollateralAmount, err := convertExchangeRatesObj.ExchangePToken2PRVByTokenId(
				waitingPorting.TokenID(),
				up150Percent(cus.Amount, portalParam.MinPercentLockedCollateral))
			if err != nil {
				Logger.log.Errorf("[calAmountTopUpWaitingPortings] Error when converting ptoken to PRV %v", err)
				return result, err
			}

			if minCollateralAmount <= cus.LockedAmountCollateral {
				break
			}

			result[waitingPorting.UniquePortingID()] = minCollateralAmount - cus.LockedAmountCollateral
		}
	}

	return result, nil
}
