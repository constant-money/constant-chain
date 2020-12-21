package metadata

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/incognitochain/incognito-chain/wallet"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type StopAutoStakingMetadata struct {
	MetadataBase
	CommitteePublicKey string
}

func (meta *StopAutoStakingMetadata) Hash() *common.Hash {
	record := strconv.Itoa(meta.Type)
	data := []byte(record)
	data = append(data, meta.Sig...)
	hash := common.HashH(data)
	return &hash
}

func (meta *StopAutoStakingMetadata) HashWithoutSig() *common.Hash {
	return meta.MetadataBase.Hash()
}

func (*StopAutoStakingMetadata) ShouldSignMetaData() bool { return true }

func NewStopAutoStakingMetadata(stopStakingType int, committeePublicKey string) (*StopAutoStakingMetadata, error) {
	if stopStakingType != StopAutoStakingMeta {
		return nil, errors.New("invalid stop staking type")
	}
	metadataBase := NewMetadataBase(stopStakingType)
	return &StopAutoStakingMetadata{
		MetadataBase:       *metadataBase,
		CommitteePublicKey: committeePublicKey,
	}, nil
}

/*
 */
func (stopAutoStakingMetadata *StopAutoStakingMetadata) ValidateMetadataByItself() bool {
	CommitteePublicKey := new(incognitokey.CommitteePublicKey)
	if err := CommitteePublicKey.FromString(stopAutoStakingMetadata.CommitteePublicKey); err != nil {
		return false
	}
	if !CommitteePublicKey.CheckSanityData() {
		return false
	}
	return (stopAutoStakingMetadata.Type == StopAutoStakingMeta)
}

//ValidateTxWithBlockChain Validate Condition to Request Stop AutoStaking With Blockchain
//- Requested Committee Publickey is in candidate, pending validator,
//- Requested Committee Publickey is in staking tx list,
//- Requester (sender of tx) must be address, which create staking transaction for current requested committee public key
//- Not yet requested to stop auto-restaking
func (stopAutoStakingMetadata StopAutoStakingMetadata) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	stopStakingMetadata, ok := tx.GetMetadata().(*StopAutoStakingMetadata)
	if !ok {
		return false, NewMetadataTxError(StopAutoStakingRequestTypeAssertionError, fmt.Errorf("Expect *StopAutoStakingMetadata type but get %+v", reflect.TypeOf(tx.GetMetadata())))
	}
	requestedPublicKey := stopStakingMetadata.CommitteePublicKey
	committees, err := beaconViewRetriever.GetAllCommitteeValidatorCandidateFlattenListFromDatabase()
	if err != nil {
		return false, NewMetadataTxError(StopAutoStakingRequestNotInCommitteeListError, err)
	}
	// if not found
	if !(common.IndexOfStr(requestedPublicKey, committees) > -1) {
		return false, NewMetadataTxError(StopAutoStakingRequestNotInCommitteeListError, fmt.Errorf("Committee Publickey %+v not found in any committee list of current beacon beststate", requestedPublicKey))
	}
	stakingTx := shardViewRetriever.GetStakingTx()
	if tempStakingTxHash, ok := stakingTx[requestedPublicKey]; !ok {
		return false, NewMetadataTxError(StopAutoStakingRequestStakingTransactionNotFoundError, fmt.Errorf("No Committe Publickey %+v found in StakingTx of Shard %+v", requestedPublicKey, shardID))
	} else {
		stakingTxHash, err := common.Hash{}.NewHashFromStr(tempStakingTxHash)
		if err != nil {
			return false, err
		}
		_, _, _, _, stakingTx, err := chainRetriever.GetTransactionByHash(*stakingTxHash)
		if err != nil {
			return false, NewMetadataTxError(StopAutoStakingRequestStakingTransactionNotFoundError, err)
		}

		stakingMetadata := stakingTx.GetMetadata().(*StakingMetadata)
		funderPaymentAddress := stakingMetadata.FunderPaymentAddress
		funderWallet, err := wallet.Base58CheckDeserialize(funderPaymentAddress)
		if err != nil || funderWallet == nil {
			return false, errors.New("Invalid Funder Payment Address, Failed to Deserialized Into Key Wallet")
		}

		if ok, err := tx.CheckAuthorizedSender(funderWallet.KeySet.PaymentAddress.Pk); !ok || err != nil {
			fmt.Println("Tx Meatadata Sig", tx.GetMetadata().GetSig())
			fmt.Println("Check authorized sender:", ok, err)
			return false, NewMetadataTxError(StopAutoStakingRequestInvalidTransactionSenderError, fmt.Errorf("CheckAuthorizedSender fail"))
		}
		//if !bytes.Equal(stakingTx.GetSender(), txr.GetSender()) {
		//}
	}
	autoStakingList := beaconViewRetriever.GetAutoStakingList()
	if isAutoStaking, ok := autoStakingList[stopStakingMetadata.CommitteePublicKey]; !ok {
		return false, NewMetadataTxError(StopAutoStakingRequestNoAutoStakingAvaiableError, fmt.Errorf("Committe Publickey %+v already request stop auto re-staking", stopStakingMetadata.CommitteePublicKey))
	} else {
		if !isAutoStaking {
			return false, NewMetadataTxError(StopAutoStakingRequestAlreadyStopError, fmt.Errorf("Auto Staking for Committee Public Key %+v already stop", stopAutoStakingMetadata.CommitteePublicKey))
		}
	}
	return true, nil
}

// Have only one receiver
// Have only one amount corresponding to receiver
// Receiver Is Burning Address
func (stopAutoStakingMetadata StopAutoStakingMetadata) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	isBurned, burnCoin, tokenID, err := tx.GetTxBurnData()
	if err != nil {
		return false, false, errors.New("Error Cannot get burn data from tx")
	}
	if !isBurned {
		return false, false, errors.New("Error StopAutoStaking tx should be a burn tx")
	}
	if !bytes.Equal(tokenID[:], common.PRVCoinID[:]) {
		return false, false, errors.New("Error StopAutoStaking tx should transfer PRV only")
	}
	if stopAutoStakingMetadata.Type != StopAutoStakingMeta && burnCoin.GetValue() != StopAutoStakingAmount {
		return false, false, errors.New("receiver amount should be zero")
	}
	CommitteePublicKey := new(incognitokey.CommitteePublicKey)
	if err := CommitteePublicKey.FromString(stopAutoStakingMetadata.CommitteePublicKey); err != nil {
		return false, false, err
	}
	if !CommitteePublicKey.CheckSanityData() {
		return false, false, errors.New("Invalid Commitee Public Key of Candidate who join consensus")
	}
	return true, true, nil
}

func (stopAutoStakingMetadata StopAutoStakingMetadata) GetType() int {
	return stopAutoStakingMetadata.Type
}

func (stopAutoStakingMetadata *StopAutoStakingMetadata) CalculateSize() uint64 {
	return calculateSize(stopAutoStakingMetadata)
}
