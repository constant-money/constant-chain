package metadata

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
)

type ReturnStakingMetadata struct {
	MetadataBase
	TxID          string
	StakerAddress privacy.PaymentAddress // Payment Address of funder, not staker
}

func NewReturnStaking(
	txID string,
	producerAddress privacy.PaymentAddress,
	metaType int,
) *ReturnStakingMetadata {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &ReturnStakingMetadata{
		TxID:          txID,
		StakerAddress: producerAddress,
		MetadataBase:  metadataBase,
	}
}

func NewReturnStakingMetaFromStakingTx(
	txStake Transaction,
) *ReturnStakingMetadata {
	metadataBase := MetadataBase{
		Type: ReturnStakingMeta,
	}
	meta := txStake.GetMetadata()
	stakeMeta, ok := meta.(*StakingMetadata)
	if !ok {
		return nil
	}
	funder, _ := wallet.PaymentAddressFromString(stakeMeta.FunderPaymentAddress)
	return &ReturnStakingMetadata{
		TxID:          txStake.Hash().String(),
		StakerAddress: funder,
		MetadataBase:  metadataBase,
	}
}

func (sbsRes ReturnStakingMetadata) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, stateDB *statedb.StateDB) bool {
	// no need to have fee for this tx
	return true
}

func (sbsRes ReturnStakingMetadata) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	return true, nil
}

// pk: 32, tk: 32
func (sbsRes ReturnStakingMetadata) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {

	if len(sbsRes.StakerAddress.Pk) != common.PublicKeySize {
		return false, false, errors.New("Wrong request info's producer address")
	}

	if len(sbsRes.StakerAddress.Tk) != common.TransmissionKeySize {
		return false, false, errors.New("Wrong request info's producer address")
	}

	if sbsRes.TxID == "" {
		return false, false, errors.New("Wrong request info's Tx staking")
	}

	_, err := common.Hash{}.NewHashFromStr(sbsRes.TxID)
	if err != nil {
		return false, false, errors.New("Wrong request info's Tx staking hash")
	}

	return false, true, nil
}

func (sbsRes ReturnStakingMetadata) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return true
}

func (sbsRes ReturnStakingMetadata) Hash() *common.Hash {
	record := sbsRes.StakerAddress.String()
	record += sbsRes.TxID

	// final hash
	record += sbsRes.MetadataBase.Hash().String()
	hash := common.HashH([]byte(record))
	return &hash
}
