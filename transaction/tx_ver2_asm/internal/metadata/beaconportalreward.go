package metadata

type PortalRewardInfo struct {
	rewards map[string]uint64 // tokenID : amount
}
type PortalRewardContent struct {
	BeaconHeight uint64
	Rewards      map[string]*PortalRewardInfo // custodian incognito address : reward infos
}

func NewPortalReward(beaconHeight uint64, rewardInfos map[string]*PortalRewardInfo) *PortalRewardContent {
	return &PortalRewardContent{
		BeaconHeight: beaconHeight,
		Rewards:      rewardInfos,
	}
}

type PortalTotalCustodianReward struct {
	Rewards map[string]uint64
}

func NewPortalTotalCustodianReward(rewards map[string]uint64) *PortalTotalCustodianReward {
	return &PortalTotalCustodianReward{
		Rewards: rewards,
	}
}
