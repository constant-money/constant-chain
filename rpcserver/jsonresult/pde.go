package jsonresult

import "github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"

type CurrentPDEState struct {
	WaitingPDEContributions map[string]*rawdbv2.PDEContribution `json:"WaitingPDEContributions"`
	PDEPoolPairs            map[string]*rawdbv2.PDEPoolForPair  `json:"PDEPoolPairs"`
	PDEShares               map[string]uint64                   `json:"PDEShares"`
	PDETradingFees          map[string]uint64                   `json:"PDETradingFees"`
	BeaconTimeStamp         int64                               `json:"BeaconTimeStamp"`
}
