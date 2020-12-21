package tx_generic

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"github.com/incognitochain/incognito-chain/wallet"
)

func VerifyTxCreatedByMiner(tx metadata.Transaction, mintdata *metadata.MintData, shardID byte, bcr metadata.ChainRetriever, accumulatedValues *metadata.AccumulatedValues, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever) (bool, error)  {
	if tx.IsPrivacy() {
		return true, nil
	}
	proof := tx.GetProof()
	meta := tx.GetMetadata()

	inputCoins := make([]privacy.PlainCoin, 0)
	outputCoins := make([]privacy.Coin, 0)
	if tx.GetProof() != nil {
		inputCoins = tx.GetProof().GetInputCoins()
		outputCoins = tx.GetProof().GetOutputCoins()
	}
	if proof != nil && len(inputCoins) == 0 && len(outputCoins) > 0 { // coinbase tx
		if meta == nil {
			return false, nil
		}
		if !meta.IsMinerCreatedMetaType() {
			return false, nil
		}
	}

	//if type is reward and not have metadata
	if tx.GetType() == common.TxRewardType && meta == nil  {
		return false, nil
	}
	//if type is return staking and not have metadata
	if tx.GetType() == common.TxReturnStakingType && (meta == nil || (meta.GetType() != metadata.ReturnStakingMeta)) {
		return false, nil
	}

	if meta != nil {
		ok, err := meta.VerifyMinerCreatedTxBeforeGettingInBlock(mintdata, shardID, tx, bcr, accumulatedValues, retriever, viewRetriever)
		if err != nil {
			utils.Logger.Log.Error(err)
			return false, utils.NewTransactionErr(utils.VerifyMinerCreatedTxBeforeGettingInBlockError, err)
		}
		return ok, nil
	}
	return true, nil
}
func GetTxMintData(tx metadata.Transaction, tokenID *common.Hash) (bool, privacy.Coin, *common.Hash, error) {
	outputCoins, err := tx.GetReceiverData()
	if err != nil {
		utils.Logger.Log.Error("GetTxMintData: Cannot get receiver data")
		return false, nil, nil, err
	}
	if len(outputCoins) != 1 {
		utils.Logger.Log.Error("GetTxMintData : Should only have one receiver")
		return false, nil, nil, errors.New("Error Tx mint has more than one receiver")
	}
	if inputCoins := tx.GetProof().GetInputCoins(); len(inputCoins) > 0 {
		return false, nil, nil, errors.New("Error this is not Tx mint")
	}
	return true, outputCoins[0], tokenID, nil
}

func GetTxBurnData(tx metadata.Transaction) (bool, privacy.Coin, *common.Hash, error) {
	outputCoins, err := tx.GetReceiverData()
	if err != nil {
		utils.Logger.Log.Errorf("Cannot get receiver data, error %v", err)
		return false, nil, nil, err
	}
	// remove rule only accept maximum 2 outputs in tx burn
	//if len(outputCoins) > 2 {
	//	utils.Logger.Log.Error("GetAndCheckBurning receiver: More than 2 receivers")
	//	return false, nil, nil, err
	//}
	for _, coin := range outputCoins {
		if wallet.IsPublicKeyBurningAddress(coin.GetPublicKey().ToBytesS()) {
			return true, coin, &common.PRVCoinID, nil
		}
	}
	return false, nil, nil, nil
}

func MdValidateWithBlockChain(tx metadata.Transaction, chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error {
	if tx.GetType() == common.TxRewardType || tx.GetType() == common.TxReturnStakingType {
		return nil
	}
	meta := tx.GetMetadata()
	if meta != nil {
		isContinued, err := meta.ValidateTxWithBlockChain(tx, chainRetriever, shardViewRetriever, beaconViewRetriever, shardID, stateDB)
		fmt.Printf("[transactionStateDB] validate metadata with blockchain: %d %h %t %v\n", tx.GetMetadataType(), tx.Hash(), isContinued, err)
		if err != nil {
			utils.Logger.Log.Errorf("[db] validate metadata with blockchain: %d %s %t %v\n", tx.GetMetadataType(), tx.Hash().String(), isContinued, err)
			return utils.NewTransactionErr(utils.RejectTxMedataWithBlockChain, fmt.Errorf("validate metadata of tx %s with blockchain error %+v", tx.Hash().String(), err))
		}
		if !isContinued {
			return nil
		}
	}
	return nil
}

func MdValidate(tx metadata.Transaction, hasPrivacy bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte) (bool, error) {
	meta := tx.GetMetadata()
	if meta != nil {
		if hasPrivacy && tx.GetVersion() <= 1{
			return false, errors.New("Metadata can only exist in non-privacy tx")
		}
		validateMetadata := meta.ValidateMetadataByItself()
		if validateMetadata {
			return validateMetadata, nil
		} else {
			return validateMetadata, utils.NewTransactionErr(utils.UnexpectedError, errors.New("Metadata is invalid"))
		}
	}
	return true, nil
}

// func validateTransaction(tx metadata.Transaction, hasPrivacy bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash, isBatch bool, isNewTransaction bool) (bool, error) {
// 	switch tx.GetType() {
// 	case common.TxRewardType:
// 		return tx.ValidateTxSalary(transactionStateDB)
// 	case common.TxReturnStakingType:
// 		return tx.ValidateTxReturnStaking(transactionStateDB), nil
// 	case common.TxConversionType:
// 		return validateConversionVer1ToVer2(tx, transactionStateDB, shardID, tokenID)
// 	}
// 	// fmt.Printf("TokenID here is %s\n", tokenID.String())
// 	return tx.Verify(hasPrivacy, transactionStateDB, bridgeStateDB, shardID, tokenID, isBatch, isNewTransaction)
// }

func MdValidateSanity(tx metadata.Transaction, chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) ( bool, error) {
	meta := tx.GetMetadata()
	if meta != nil {
		isValid, ok, err := meta.ValidateSanityData(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight, tx )
		if err != nil || !ok || !isValid {
			return ok, err
		}
	}
	return true, nil
}

func ValidateSanity(tx metadata.Transaction, chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, error) {
	//check version
	if tx.GetVersion() > utils.TxVersion2Number {
		return false, utils.NewTransactionErr(utils.RejectTxVersion, fmt.Errorf("tx version is %d. Wrong version tx. Only support for version <= %d", tx.GetVersion(), utils.CurrentTxVersion))
	}
	// check LockTime before now
	if tx.GetLockTime() > time.Now().Unix() {
		return false, utils.NewTransactionErr(utils.RejectInvalidLockTime, fmt.Errorf("wrong tx locktime %d", tx.GetLockTime()))
	}

	// check tx size
	actualTxSize := tx.GetTxActualSize()
	if actualTxSize > common.MaxTxSize {
		return false, utils.NewTransactionErr(utils.RejectTxSize, fmt.Errorf("tx size %d kB is too large", actualTxSize))
	}

	// check sanity of Proof
	if tx.GetProof() != nil {
		shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
		additionalData := make(map[string]interface{})

		if tx.GetVersion() <= 1{
			if chainRetriever != nil {
				additionalData["isNewZKP"] = chainRetriever.IsAfterNewZKPCheckPoint(beaconHeight)
			}
			sigPubKey, err :=  new(operation.Point).FromBytesS(tx.GetSigPubKey())
			if err != nil {
				return false, errors.New("SigPubKey is invalid")
			}
			additionalData["sigPubKey"] = sigPubKey
		}

		additionalData["shardID"] = shardID

		ok, err := tx.GetProof().ValidateSanity(additionalData)
		if !ok || err != nil {
			s := ""
			if !ok {
				s = fmt.Sprintf("ValidateSanity Proof got error: ok = false; %s\n", err.Error())
			} else {
				s = fmt.Sprintf("ValidateSanity Proof got error: error %s\n", err.Error())
			}
			return false, errors.New(s)
		}
	}

	// check Type is normal or salary tx
	switch tx.GetType() {
	case common.TxNormalType, common.TxRewardType, common.TxCustomTokenPrivacyType, common.TxTokenConversionType, common.TxReturnStakingType, common.TxConversionType: //is valid
	default:
		return false, utils.NewTransactionErr(utils.RejectTxType, fmt.Errorf("wrong tx type with %s", tx.GetType()))
	}

	// check info field
	info := tx.GetInfo()
	if len(info) > utils.MaxSizeInfo {
		return false, utils.NewTransactionErr(utils.RejectTxInfoSize, fmt.Errorf("wrong tx info length %d bytes, only support info with max length <= %d bytes", len(info), 512))
	}
	return true, nil
}

func GetTxActualSizeInBytes(tx metadata.Transaction) uint64{
	if tx == nil {
		return uint64(0)
	}
	var sizeTx = uint64(0)
	txTokenBase, ok := tx.(*TxTokenBase)
	if ok { //TxTokenBase
		sizeTx += GetTxActualSizeInBytes(txTokenBase.Tx)

		if &txTokenBase.TxTokenData != nil {
			sizeTx += GetTxActualSizeInBytes(txTokenBase.TxTokenData.TxNormal)
			sizeTx += uint64(len(txTokenBase.TxTokenData.PropertyName))
			sizeTx += uint64(len(txTokenBase.TxTokenData.PropertySymbol))
			sizeTx += uint64(len(txTokenBase.TxTokenData.PropertyID))
			sizeTx += 4 // Type
			sizeTx += 1 // Mintable
			sizeTx += 8 // Amount
		}
		meta := txTokenBase.GetMetadata()
		if meta != nil {
			sizeTx += meta.CalculateSize()
		}

		return sizeTx
	}else{ //TxBase
		sizeTx += uint64(1) //version
		sizeTx += uint64(len(tx.GetType()) + 1) //type string
		sizeTx += uint64(8) //locktime
		sizeTx += uint64(8) //fee
		sizeTx += uint64(len(tx.GetInfo())) //info

		sizeTx += uint64(len(tx.GetSigPubKey())) //sigpubkey
		sizeTx += uint64(len(tx.GetSig())) //signature
		sizeTx += uint64(1) //pubkeylastbytesender

		//paymentproof
		if tx.GetProof() != nil {
			sizeTx += uint64(len(tx.GetProof().Bytes()))
		}

		//metadata
		if tx.GetMetadata() != nil {
			sizeTx += tx.GetMetadata().CalculateSize()
		}

		return sizeTx
	}
}