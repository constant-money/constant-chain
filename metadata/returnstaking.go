package metadata

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
	"reflect"
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

func (sbsRes ReturnStakingMetadata) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, stateDB *statedb.StateDB) bool {
	// no need to have fee for this tx
	return true
}

func (sbsRes ReturnStakingMetadata) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	return true, nil
}

func (sbsRes ReturnStakingMetadata) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	if len(sbsRes.StakerAddress.Pk) == 0 {
		return false, false, errors.New("Wrong request info's producer address")
	}
	if len(sbsRes.StakerAddress.Tk) == 0 {
		return false, false, errors.New("Wrong request info's producer address")
	}
	if sbsRes.TxID == "" {
		return false, false, errors.New("Wrong request info's Tx staking")
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
	if sbsRes.SharedRandom != nil && len(sbsRes.SharedRandom) > 0 {
		record += string(sbsRes.SharedRandom)
	}
	// final hash
	record += sbsRes.MetadataBase.Hash().String()
	hash := common.HashH([]byte(record))
	return &hash
}

func (sbsRes ReturnStakingMetadata) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *MintData, shardID byte,
	tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever,
	beaconViewRetriever BeaconViewRetriever) (bool, error) {

	stakingTx := shardViewRetriever.GetStakingTx()
	for key, value := range stakingTx {
		committeePublicKey := incognitokey.CommitteePublicKey{}
		err := committeePublicKey.FromString(key)
		if err != nil {
			return false, err
		}
		if reflect.DeepEqual(sbsRes.StakerAddress.Pk, committeePublicKey.IncPubKey) && (sbsRes.TxID == value) {
			autoStakingList := beaconViewRetriever.GetAutoStakingList()
			if autoStakingList[key] {
				return false, errors.New("Can not return staking amount for candidate: AutoStaking = true.")
			}

			if _, ok := mintData.ReturnStaking[sbsRes.TxID]; !ok {
				mintData.ReturnStaking[sbsRes.TxID] = true
			} else {
				return false, errors.New("Return Staking: Double mint transaction return staking.")
			}

			isMinted, mintCoin, coinID, err := tx.GetTxMintData()
			//check tx mint
			if err != nil || !isMinted {
				return false, errors.Errorf("Return Staking: It is not tx mint with error: %v", err)
			}
			if cmp, err := coinID.Cmp(&common.PRVCoinID); err != nil || cmp != 0 {
				return false, errors.Errorf("Return Staking: Must mint PRV only")
			}

			txIDReq, err := common.Hash{}.NewHashFromStr(value)
			if err != nil {
				return false, errors.New("Return Staking: Cannot Convert TxID from string to common.Hash")
			}
			_, _, _, _, txReq, err := chainRetriever.GetTransactionByHash(*txIDReq)
			if err != nil {
				return false, errors.Errorf("Return Staking: Cannot get tx request from tx hash %v", value)
			}
			_, burnCoin, _, err := txReq.GetTxBurnData()
			if err != nil {
				return false, errors.Errorf("Return Staking: Cannot get burn data from Tx Staking")
			}
			if ok := mintCoin.CheckCoinValid(sbsRes.StakerAddress, sbsRes.SharedRandom, burnCoin.GetValue()); !ok {
				return false, errors.New("Return Staking: Mint Coin is invalid for corresponding Staker Payment Address")
			}
			fmt.Print("Check Mint Return Staking Valid OK OK OK")

			return true, nil
		}

	}
	return false, errors.New("Can not find any staking information of this publickey")
}

func (sbsRes *ReturnStakingMetadata) SetSharedRandom(r []byte) {
	sbsRes.SharedRandom = r
}

