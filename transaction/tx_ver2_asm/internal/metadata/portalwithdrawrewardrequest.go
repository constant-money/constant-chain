package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
)

// PortalRequestWithdrawReward - custodians request withdraw reward
// metadata - custodians request withdraw reward - create normal tx with this metadata
type PortalRequestWithdrawReward struct {
	MetadataBase
	CustodianAddressStr string
	TokenID             common.Hash
}

// PortalRequestWithdrawRewardAction - shard validator creates instruction that contain this action content
// it will be append to ShardToBeaconBlock
type PortalRequestWithdrawRewardAction struct {
	Meta    PortalRequestWithdrawReward
	TxReqID common.Hash
	ShardID byte
}

// PortalRequestWithdrawRewardContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalRequestWithdrawRewardContent struct {
	CustodianAddressStr string
	TokenID             common.Hash
	RewardAmount        uint64
	TxReqID             common.Hash
	ShardID             byte
}

// PortalRequestWithdrawRewardStatus - Beacon tracks status of request unlock collateral amount into db
type PortalRequestWithdrawRewardStatus struct {
	Status              byte
	CustodianAddressStr string
	TokenID             common.Hash
	RewardAmount        uint64
	TxReqID             common.Hash
}

func NewPortalRequestWithdrawReward(
	metaType int,
	incogAddressStr string,
	tokenID common.Hash) (*PortalRequestWithdrawReward, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}
	meta := &PortalRequestWithdrawReward{
		CustodianAddressStr: incogAddressStr,
		TokenID:             tokenID,
	}
	meta.MetadataBase = metadataBase
	return meta, nil
}

func (*PortalRequestWithdrawReward) ShouldSignMetaData() bool { return true }


func (meta PortalRequestWithdrawReward) Hash() *common.Hash {
	record := meta.MetadataBase.Hash().String()
	record += meta.CustodianAddressStr
	record += meta.TokenID.String()
	if meta.Sig != nil && len(meta.Sig) != 0 {
		record += string(meta.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (meta PortalRequestWithdrawReward) HashWithoutSig() *common.Hash {
	record := meta.MetadataBase.Hash().String()
	record += meta.CustodianAddressStr
	record += meta.TokenID.String()
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
