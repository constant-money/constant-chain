package basemeta

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

func getPDEPoolPair(
	prvIDStr, tokenIDStr string,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (*rawdbv2.PDEPoolForPair, error) {
	var pdePoolForPair rawdbv2.PDEPoolForPair
	var err error
	poolPairBytes := []byte{}
	if beaconHeight == -1 {
		poolPairBytes, err = statedb.GetLatestPDEPoolForPair(stateDB, prvIDStr, tokenIDStr)
	} else {
		poolPairBytes, err = statedb.GetPDEPoolForPair(stateDB, uint64(beaconHeight), prvIDStr, tokenIDStr)
	}
	if err != nil {
		return nil, err
	}
	if len(poolPairBytes) == 0 {
		return nil, NewMetadataTxError(CouldNotGetExchangeRateError, fmt.Errorf("Could not find out pdePoolForPair with token ids: %s & %s", prvIDStr, tokenIDStr))
	}
	err = json.Unmarshal(poolPairBytes, &pdePoolForPair)
	if err != nil {
		return nil, err
	}
	return &pdePoolForPair, nil
}

func isPairValid(poolPair *rawdbv2.PDEPoolForPair, beaconHeight int64) bool {
	if poolPair == nil {
		return false
	}
	prvIDStr := common.PRVCoinID.String()
	if poolPair.Token1IDStr == prvIDStr &&
		poolPair.Token1PoolValue < uint64(common.MinTxFeesOnTokenRequirement) &&
		beaconHeight >= common.BeaconBlockHeighMilestoneForMinTxFeesOnTokenRequirement {
		return false
	}
	if poolPair.Token2IDStr == prvIDStr &&
		poolPair.Token2PoolValue < uint64(common.MinTxFeesOnTokenRequirement) &&
		beaconHeight >= common.BeaconBlockHeighMilestoneForMinTxFeesOnTokenRequirement {
		return false
	}
	return true
}

func convertValueBetweenCurrencies(
	amount uint64,
	currentCurrencyIDStr string,
	tokenID *common.Hash,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (float64, error) {
	prvIDStr := common.PRVCoinID.String()
	tokenIDStr := tokenID.String()
	pdePoolForPair, err := getPDEPoolPair(prvIDStr, tokenIDStr, beaconHeight, stateDB)
	if err != nil {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	if !isPairValid(pdePoolForPair, beaconHeight) {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, errors.New("PRV pool size on pdex is smaller minimum initial adding liquidity amount"))
	}
	invariant := float64(0)
	invariant = float64(pdePoolForPair.Token1PoolValue) * float64(pdePoolForPair.Token2PoolValue)
	if invariant == 0 {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	if pdePoolForPair.Token1IDStr == currentCurrencyIDStr {
		remainingValue := invariant / (float64(pdePoolForPair.Token1PoolValue) + float64(amount))
		if float64(pdePoolForPair.Token2PoolValue) <= remainingValue {
			return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
		}
		return float64(pdePoolForPair.Token2PoolValue) - remainingValue, nil
	}
	remainingValue := invariant / (float64(pdePoolForPair.Token2PoolValue) + float64(amount))
	if float64(pdePoolForPair.Token1PoolValue) <= remainingValue {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	return float64(pdePoolForPair.Token1PoolValue) - remainingValue, nil
}

// return error if there is no exchange rate between native token and privacy token
// beaconHeight = -1: get the latest beacon height
func ConvertNativeTokenToPrivacyToken(
	nativeTokenAmount uint64,
	tokenID *common.Hash,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (float64, error) {
	return convertValueBetweenCurrencies(
		nativeTokenAmount,
		common.PRVCoinID.String(),
		tokenID,
		beaconHeight,
		stateDB,
	)
}

// return error if there is no exchange rate between native token and privacy token
// beaconHeight = -1: get the latest beacon height
func ConvertPrivacyTokenToNativeToken(
	privacyTokenAmount uint64,
	tokenID *common.Hash,
	beaconHeight int64,
	stateDB *statedb.StateDB,
) (float64, error) {
	return convertValueBetweenCurrencies(
		privacyTokenAmount,
		tokenID.String(),
		tokenID,
		beaconHeight,
		stateDB,
	)
}

func IsPortalMetadata(metaType int) bool {
	res, _ := common.SliceExists(portalMetas, metaType)
	return res
}

func IsPortalV4Metadata(metaType int) bool {
	res, _ := common.SliceExists(portalV4Metas, metaType)
	return res
}