package committeestate

import (
	"github.com/incognitochain/incognito-chain/blockchain/signaturecounter"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type BeaconCommitteeStateEnvironment struct {
	BeaconHeight                       uint64
	Epoch                              uint64
	BeaconHash                         common.Hash
	BeaconInstructions                 [][]string
	EpochBreakPointSwapNewKey          []uint64
	RandomNumber                       int64
	IsFoundRandomNumber                bool
	IsBeaconRandomTime                 bool
	AssignOffset                       int
	DefaultOffset                      int
	SwapOffset                         int
	ActiveShards                       int
	MinShardCommitteeSize              int
	MinBeaconCommitteeSize             int
	MaxBeaconCommitteeSize             int
	MaxShardCommitteeSize              int
	ConsensusStateDB                   *statedb.StateDB
	IsReplace                          bool
	newAllCandidateSubstituteCommittee []string
	newUnassignedCommonPool            []string
	newAllSubstituteCommittees         []string
	LatestShardsState                  map[byte][]types.ShardState
	SwapSubType                        uint
	ShardID                            byte
	TotalReward                        map[common.Hash]uint64
	IsSplitRewardForCustodian          bool
	PercentCustodianReward             uint64
	DAOPercent                         int
	NumberOfFixedBeaconBlockValidator  uint64
	NumberOfFixedShardBlockValidator   int
	MissingSignaturePenalty            map[string]signaturecounter.Penalty
	DcsMinShardCommitteeSize           int
	DcsMaxShardCommitteeSize           int
	SwapRuleV3Epoch                    uint64
	SwapRuleV2Epoch                    uint64
}

type BeaconCommitteeStateHash struct {
	BeaconCommitteeAndValidatorHash common.Hash
	BeaconCandidateHash             common.Hash
	ShardCandidateHash              common.Hash
	ShardCommitteeAndValidatorHash  common.Hash
	AutoStakeHash                   common.Hash
}

func NewBeaconCommitteeStateEnvironmentForUpdateDB(
	statedb *statedb.StateDB,
) *BeaconCommitteeStateEnvironment {
	return &BeaconCommitteeStateEnvironment{
		ConsensusStateDB: statedb,
	}
}

func NewBeaconCommitteeStateEnvironmentForSwapRule(currentEpoch, swapRuleV3Epoch, swapRuleV2Epoch uint64) *BeaconCommitteeStateEnvironment {
	return &BeaconCommitteeStateEnvironment{
		Epoch:           currentEpoch,
		SwapRuleV2Epoch: swapRuleV2Epoch,
		SwapRuleV3Epoch: swapRuleV3Epoch,
	}
}
