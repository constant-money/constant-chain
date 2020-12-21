package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type CurrentPDEState struct {
	WaitingPDEContributions        map[string]*rawdbv2.PDEContribution
	DeletedWaitingPDEContributions map[string]*rawdbv2.PDEContribution
	PDEPoolPairs                   map[string]*rawdbv2.PDEPoolForPair
	PDEShares                      map[string]uint64
	PDETradingFees                 map[string]uint64
}

func (s *CurrentPDEState) Copy() *CurrentPDEState {
	v := new(CurrentPDEState)
	b := new(bytes.Buffer)
	err := gob.NewEncoder(b).Encode(s)
	if err != nil {
		return nil
	}
	err = gob.NewDecoder(bytes.NewBuffer(b.Bytes())).Decode(v)
	if err != nil {
		return nil
	}
	return v
}

type DeductingAmountsByWithdrawal struct {
	Token1IDStr string
	PoolValue1  uint64
	Token2IDStr string
	PoolValue2  uint64
	Shares      uint64
}

type DeductingAmountsByWithdrawalWithPRVFee struct {
	Token1IDStr   string
	PoolValue1    uint64
	Token2IDStr   string
	PoolValue2    uint64
	Shares        uint64
	FeeTokenIDStr string
	FeeAmount     uint64
}

func InitCurrentPDEStateFromDB(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
) (*CurrentPDEState, error) {
	waitingPDEContributions, err := statedb.GetWaitingPDEContributions(stateDB, beaconHeight)
	if err != nil {
		return nil, err
	}
	pdePoolPairs, err := statedb.GetPDEPoolPair(stateDB, beaconHeight)
	if err != nil {
		return nil, err
	}
	pdeShares, err := statedb.GetPDEShares(stateDB, beaconHeight)
	if err != nil {
		return nil, err
	}
	pdeTradingFees, err := statedb.GetPDETradingFees(stateDB, beaconHeight)
	if err != nil {
		return nil, err
	}
	return &CurrentPDEState{
		WaitingPDEContributions:        waitingPDEContributions,
		PDEPoolPairs:                   pdePoolPairs,
		PDEShares:                      pdeShares,
		PDETradingFees:                 pdeTradingFees,
		DeletedWaitingPDEContributions: make(map[string]*rawdbv2.PDEContribution),
	}, nil
}

func storePDEStateToDB(
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
) error {
	var err error
	statedb.DeleteWaitingPDEContributions(stateDB, currentPDEState.DeletedWaitingPDEContributions)
	err = statedb.StoreWaitingPDEContributions(stateDB, beaconHeight, currentPDEState.WaitingPDEContributions)
	if err != nil {
		return err
	}
	err = statedb.StorePDEPoolPairs(stateDB, beaconHeight, currentPDEState.PDEPoolPairs)
	if err != nil {
		return err
	}
	err = statedb.StorePDEShares(stateDB, beaconHeight, currentPDEState.PDEShares)
	if err != nil {
		return err
	}
	err = statedb.StorePDETradingFees(stateDB, beaconHeight, currentPDEState.PDETradingFees)
	if err != nil {
		return err
	}
	return nil
}

func replaceNewBCHeightInKeyStr(key string, newBeaconHeight uint64) string {
	parts := strings.Split(key, "-")
	if len(parts) <= 1 {
		return key
	}
	parts[1] = fmt.Sprintf("%d", newBeaconHeight)
	newKey := ""
	for idx, part := range parts {
		if idx == len(parts)-1 {
			newKey += part
			continue
		}
		newKey += (part + "-")
	}
	return newKey
}

func addShareAmountUpV2(
	beaconHeight uint64,
	token1IDStr string,
	token2IDStr string,
	contributedTokenIDStr string,
	contributorAddrStr string,
	amt uint64,
	currentPDEState *CurrentPDEState,
) {
	pdeShareOnTokenPrefix := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, token1IDStr, token2IDStr, ""))
	totalSharesOnToken := uint64(0)
	for key, value := range currentPDEState.PDEShares {
		if strings.Contains(key, pdeShareOnTokenPrefix) {
			totalSharesOnToken += value
		}
	}
	pdeShareKey := string(rawdbv2.BuildPDESharesKeyV2(beaconHeight, token1IDStr, token2IDStr, contributorAddrStr))
	if totalSharesOnToken == 0 {
		currentPDEState.PDEShares[pdeShareKey] = amt
		return
	}
	poolPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, token1IDStr, token2IDStr))
	poolPair, found := currentPDEState.PDEPoolPairs[poolPairKey]
	if !found || poolPair == nil {
		currentPDEState.PDEShares[pdeShareKey] = amt
		return
	}
	poolValue := poolPair.Token1PoolValue
	if poolPair.Token2IDStr == contributedTokenIDStr {
		poolValue = poolPair.Token2PoolValue
	}
	if poolValue == 0 {
		currentPDEState.PDEShares[pdeShareKey] = amt
	}
	increasingAmt := big.NewInt(0)

	increasingAmt.Mul(new(big.Int).SetUint64(totalSharesOnToken), new(big.Int).SetUint64(amt))
	increasingAmt.Div(increasingAmt, new(big.Int).SetUint64(poolValue))

	currentShare, found := currentPDEState.PDEShares[pdeShareKey]
	addedUpAmt := increasingAmt.Uint64()
	if found {
		addedUpAmt += currentShare
	}
	currentPDEState.PDEShares[pdeShareKey] = addedUpAmt
}

func updateWaitingContributionPairToPoolV2(
	beaconHeight uint64,
	waitingContribution1 *rawdbv2.PDEContribution,
	waitingContribution2 *rawdbv2.PDEContribution,
	currentPDEState *CurrentPDEState,
) {
	addShareAmountUpV2(
		beaconHeight,
		waitingContribution1.TokenIDStr,
		waitingContribution2.TokenIDStr,
		waitingContribution1.TokenIDStr,
		waitingContribution1.ContributorAddressStr,
		waitingContribution1.Amount,
		currentPDEState,
	)

	waitingContributions := []*rawdbv2.PDEContribution{waitingContribution1, waitingContribution2}
	sort.Slice(waitingContributions, func(i, j int) bool {
		return waitingContributions[i].TokenIDStr < waitingContributions[j].TokenIDStr
	})
	pdePoolForPairKey := string(rawdbv2.BuildPDEPoolForPairKey(beaconHeight, waitingContributions[0].TokenIDStr, waitingContributions[1].TokenIDStr))
	pdePoolForPair, found := currentPDEState.PDEPoolPairs[pdePoolForPairKey]
	if !found || pdePoolForPair == nil {
		storePDEPoolForPair(
			pdePoolForPairKey,
			waitingContributions[0].TokenIDStr,
			waitingContributions[0].Amount,
			waitingContributions[1].TokenIDStr,
			waitingContributions[1].Amount,
			currentPDEState,
		)
		return
	}
	storePDEPoolForPair(
		pdePoolForPairKey,
		waitingContributions[0].TokenIDStr,
		pdePoolForPair.Token1PoolValue+waitingContributions[0].Amount,
		waitingContributions[1].TokenIDStr,
		pdePoolForPair.Token2PoolValue+waitingContributions[1].Amount,
		currentPDEState,
	)
}
