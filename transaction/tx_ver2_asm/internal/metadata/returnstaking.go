package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
)

type ReturnStakingMetadata struct {
	MetadataBase
	TxID          string
	StakerAddress privacy.PaymentAddress
	SharedRandom []byte
}

func NewReturnStaking(txID string, producerAddress privacy.PaymentAddress, metaType int, ) *ReturnStakingMetadata {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &ReturnStakingMetadata{
		TxID:          txID,
		StakerAddress: producerAddress,
		MetadataBase:  metadataBase,
	}
}

func (sbsRes ReturnStakingMetadata) Hash() *common.Hash {
	record := sbsRes.StakerAddress.String()
	record += sbsRes.TxID
	if sbsRes.SharedRandom != nil && len(sbsRes.SharedRandom) > 0 {
		record += string(sbsRes.SharedRandom)
	}
	// final hash
	record += sbsRes.MetadataBase.Hash().String()
	hash := common.HashH([]byte(record))
	return &hash
}
func (sbsRes *ReturnStakingMetadata) SetSharedRandom(r []byte) {
	sbsRes.SharedRandom = r
}