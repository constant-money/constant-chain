package metadata

import (
	"encoding/json"
	"errors"
)

type StakingMetadata struct {
	MetadataBase
	FunderPaymentAddress         string
	RewardReceiverPaymentAddress string
	StakingAmountShard           uint64
	AutoReStaking                bool
	CommitteePublicKey           string
	// CommitteePublicKey PublicKeys of a candidate who join consensus, base58CheckEncode
	// CommitteePublicKey string <= encode byte <= mashal struct
}

func NewStakingMetadata(
	stakingType int,
	funderPaymentAddress string,
	rewardReceiverPaymentAddress string,
	// candidatePaymentAddress string,
	stakingAmountShard uint64,
	committeePublicKey string,
	autoReStaking bool,
) (
	*StakingMetadata,
	error,
) {
	if stakingType != ShardStakingMeta && stakingType != BeaconStakingMeta {
		return nil, errors.New("invalid staking type")
	}
	metadataBase := NewMetadataBase(stakingType)
	return &StakingMetadata{
		MetadataBase:                 *metadataBase,
		FunderPaymentAddress:         funderPaymentAddress,
		RewardReceiverPaymentAddress: rewardReceiverPaymentAddress,
		StakingAmountShard:           stakingAmountShard,
		CommitteePublicKey:           committeePublicKey,
		AutoReStaking:                autoReStaking,
	}, nil
}
func (stakingMetadata StakingMetadata) GetType() int {
	return stakingMetadata.Type
}

func (stakingMetadata StakingMetadata) GetBeaconStakeAmount() uint64 {
	return stakingMetadata.StakingAmountShard * 3
}

func (stakingMetadata StakingMetadata) GetShardStateAmount() uint64 {
	return stakingMetadata.StakingAmountShard
}

func (sm *StakingMetadata) UnmarshalJSON(raw []byte) error{
	var temp struct{
		MetadataBase
		FunderPaymentAddress         string
		RewardReceiverPaymentAddress string
		StakingAmountShard           uintMaybeString
		AutoReStaking                bool
		CommitteePublicKey           string
	}
	err := json.Unmarshal(raw, &temp)
	*sm = StakingMetadata{
		MetadataBase: temp.MetadataBase,
		FunderPaymentAddress: temp.FunderPaymentAddress,
		RewardReceiverPaymentAddress: temp.RewardReceiverPaymentAddress,
		StakingAmountShard: uint64(temp.StakingAmountShard),
		AutoReStaking: temp.AutoReStaking,
		CommitteePublicKey: temp.CommitteePublicKey,
	}
	return err
}