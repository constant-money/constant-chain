package metadata

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

//UnStakingMetadata : unstaking metadata
type UnStakingMetadata struct {
	MetadataBase
	CommitteePublicKey string
}

//NewUnStakingMetadata : Constructor of UnStakingMetadata struct
func NewUnStakingMetadata(committeePublicKey string) (*UnStakingMetadata, error) {
	metadataBase := NewMetadataBase(UnStakingMeta)
	return &UnStakingMetadata{
		MetadataBase:       *metadataBase,
		CommitteePublicKey: committeePublicKey,
	}, nil
}

//ValidateMetadataByItself Validate data format/type in unStakingMetadata
func (unStakingMetadata *UnStakingMetadata) ValidateMetadataByItself() bool {
	CommitteePublicKey := new(incognitokey.CommitteePublicKey)
	if unStakingMetadata.Type != UnStakingMeta {
		return false
	}
	if err := CommitteePublicKey.FromString(unStakingMetadata.CommitteePublicKey); err != nil {
		return false
	}
	if !CommitteePublicKey.CheckSanityData() {
		return false
	}
	return true
}

//ValidateTxWithBlockChain Validate Condition to Request Unstake With Blockchain
//- Requested Committee Publickey is in candidate, pending validator,
//- Requested Committee Publickey is in staking tx list,
//- Requester (sender of tx) must be address, which create staking transaction for current requested committee public key
func (unStakingMetadata UnStakingMetadata) ValidateTxWithBlockChain(tx Transaction,
	chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever,
	beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {

	requestedPublicKey := unStakingMetadata.CommitteePublicKey
	committees, err := beaconViewRetriever.GetAllCommitteeValidatorCandidateFlattenListFromDatabase()
	if err != nil {
		return false, NewMetadataTxError(UnStakingRequestNotInCommitteeListError, err)
	}
	// if not found
	if !(common.IndexOfStr(requestedPublicKey, committees) > -1) {
		return false, NewMetadataTxError(UnStakingRequestNotInCommitteeListError, fmt.Errorf("Committee Publickey %+v not found in any committee list of current beacon beststate", requestedPublicKey))
	}

	stakerInfo, has, err := beaconViewRetriever.GetStakerInfo(requestedPublicKey)
	if err != nil {
		return false, NewMetadataTxError(UnStakingRequestGetStakerInfoError, err)
	}
	if !has {
		return false, NewMetadataTxError(UnStakingRequestNotFoundStakerInfoError, fmt.Errorf("Committee Publickey %+v has not staked yet", requestedPublicKey))
	}
	if stakerInfo == nil {
		return false, NewMetadataTxError(UnStakingRequestNotFoundStakerInfoError, fmt.Errorf("Committee Publickey %+v has not staked yet", requestedPublicKey))
	}

	_, _, _, _, stakingTx, err := chainRetriever.GetTransactionByHash(stakerInfo.TxStakingID())
	if err != nil {
		return false, NewMetadataTxError(UnStakingRequestStakingTransactionNotFoundError, err)
	}
	if !bytes.Equal(stakingTx.GetSender(), tx.GetSender()) {
		return false, NewMetadataTxError(UnStakingRequestInvalidTransactionSenderError, fmt.Errorf("Expect %+v to send unstake request but get %+v", stakingTx.GetSender(), tx.GetSender()))
	}

	waitingValidatorsList, err := incognitokey.CommitteeKeyListToString(beaconViewRetriever.CandidateWaitingForNextRandom())
	if err != nil {
		return false, err
	}
	index := common.IndexOfStr(requestedPublicKey, waitingValidatorsList)
	if index == -1 {
		if !stakerInfo.AutoStaking() {
			return false, NewMetadataTxError(UnstakingRequestAlreadyUnstake, errors.New("Public Key Has Already Been Unstaked"))
		}

	}

	return true, nil
}

// ValidateSanityData :
func (unStakingMetadata UnStakingMetadata) ValidateSanityData(
	chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever,
	beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {

	if !unStakingMetadata.ValidateMetadataByItself() {
		return false, false, errors.New("Fail To Validate Metadata By Itself")
	}

	if tx.IsPrivacy() {
		return false, false, errors.New("Unstaking Request Transaction Is No Privacy Transaction")
	}
	return true, true, nil
}

//GetType :
func (unStakingMetadata UnStakingMetadata) GetType() int {
	return unStakingMetadata.Type
}

//CalculateSize :
func (unStakingMetadata *UnStakingMetadata) CalculateSize() uint64 {
	return calculateSize(unStakingMetadata)
}
