package blockchain

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
	"sort"
)

func createCrossShardByteArray(txList []metadata.Transaction, fromShardID byte) []byte {
	crossIDs := []byte{}
	byteMap := make([]byte, common.MaxShardNumber)
	for _, tx := range txList {
		if tx.GetProof() != nil {
			for _, outCoin := range tx.GetProof().GetOutputCoins() {
				lastByte := outCoin.CoinDetails.GetPubKeyLastByte()
				shardID := common.GetShardIDFromLastByte(lastByte)
				byteMap[common.GetShardIDFromLastByte(shardID)] = 1
			}
		}

		switch tx.GetType() {
		case common.TxCustomTokenPrivacyType:
			{
				customTokenTx := tx.(*transaction.TxCustomTokenPrivacy)
				if customTokenTx.TxPrivacyTokenData.TxNormal.GetProof() != nil {
					for _, outCoin := range customTokenTx.TxPrivacyTokenData.TxNormal.GetProof().GetOutputCoins() {
						lastByte := outCoin.CoinDetails.GetPubKeyLastByte()
						shardID := common.GetShardIDFromLastByte(lastByte)
						byteMap[common.GetShardIDFromLastByte(shardID)] = 1
					}
				}
			}
		}
	}

	for k := range byteMap {
		if byteMap[k] == 1 && k != int(fromShardID) {
			crossIDs = append(crossIDs, byte(k))
		}
	}
	return crossIDs
}

func createShardSwapActionForKeyListV2(
	genesisParam *GenesisParams,
	shardCommittees []string,
	minCommitteeSize int,
	activeShard int,
	shardID byte,
	epoch uint64,
) ([]string, []string) {
	swapInstruction, newShardCommittees := GetShardSwapInstructionKeyListV2(genesisParam, epoch, minCommitteeSize, activeShard)
	remainShardCommittees := shardCommittees[minCommitteeSize:]
	return swapInstruction[shardID], append(newShardCommittees[shardID], remainShardCommittees...)
}

func checkReturnStakingTxExistence(txId string, shardBlock *types.ShardBlock) bool {
	for _, tx := range shardBlock.Body.Transactions {
		if tx.GetMetadata() != nil {
			if tx.GetMetadata().GetType() == metadata.ReturnStakingMeta {
				if returnStakingMeta, ok := tx.GetMetadata().(*metadata.ReturnStakingMetadata); ok {
					if returnStakingMeta.TxID == txId {
						return true
					}
				}
			}
		}
	}
	return false
}

func getRequesterFromPKnCoinID(pk privacy.PublicKey, coinID common.Hash) string {
	requester := base58.Base58Check{}.Encode(pk, common.Base58Version)
	return fmt.Sprintf("%s-%s", requester, coinID.String())
}

func reqTableFromReqTxs(
	transactions []metadata.Transaction,
) map[string]metadata.Transaction {
	txRequestTable := map[string]metadata.Transaction{}
	for _, tx := range transactions {
		if tx.GetMetadataType() == metadata.WithDrawRewardRequestMeta {
			requestMeta := tx.GetMetadata().(*metadata.WithDrawRewardRequest)
			key := getRequesterFromPKnCoinID(requestMeta.PaymentAddress.Pk, requestMeta.TokenID)
			txRequestTable[key] = tx
		}
	}
	return txRequestTable
}

func filterReqTxs(
	transactions []metadata.Transaction,
	txRequestTable map[string]metadata.Transaction,
) []metadata.Transaction {
	res := []metadata.Transaction{}
	for _, tx := range transactions {
		if tx.GetMetadataType() == metadata.WithDrawRewardRequestMeta {
			requestMeta := tx.GetMetadata().(*metadata.WithDrawRewardRequest)
			key := getRequesterFromPKnCoinID(requestMeta.PaymentAddress.Pk, requestMeta.TokenID)
			txReq, ok := txRequestTable[key]
			if !ok {
				continue
			}
			cmp, err := txReq.Hash().Cmp(tx.Hash())
			if (err != nil) || (cmp != 0) {
				continue
			}
		}
		res = append(res, tx)
	}
	return res
}

//====================New Merkle Tree================
// CreateShardTxRoot create root hash for cross shard transaction
// this root hash will be used be received shard
func CreateShardTxRoot(txList []metadata.Transaction) ([]common.Hash, []common.Hash) {
	//calculate output coin hash for each shard
	crossShardDataHash := getCrossShardDataHash(txList)
	// calculate merkel path for a shardID
	// step 1: calculate merkle data : [1, 2, 3, 4, 12, 34, 1234]
	/*
			   	1234=hash(12,34)
			   /			  \
		  12=hash(1,2)	 34=hash(3,4)
			 / \	 		 / \
			1	2			3	4
	*/
	merkleTree := Merkle{}
	merkleData := merkleTree.BuildMerkleTreeOfHashes2(crossShardDataHash, common.MaxShardNumber)
	return crossShardDataHash, merkleData
}
func GetMerklePathCrossShard(txList []metadata.Transaction, shardID byte) (merklePathShard []common.Hash, merkleShardRoot common.Hash) {
	_, merkleTree := CreateShardTxRoot(txList)
	merklePathShard, merkleShardRoot = Merkle{}.GetMerklePathForCrossShard(common.MaxShardNumber, merkleTree, shardID)
	return merklePathShard, merkleShardRoot
}

//  getCrossShardDataHash
//	Helper function: group OutputCoin into shard and get the hash of each group
//	Return value
//	 - Array of hash created from 256 group cross shard data hash
//	 - Length array is 256
//	 - Value is sorted as shardID from low to high
//	 - ShardID which have no outputcoin received hash of emptystring value
//
//	Hash Procedure:
//	- For each shard:
//	   CROSS OUTPUT COIN
//		+ Get outputcoin and append to a list of that shard
//		+ Calculate value for Hash:
//		  * if receiver shard has no outcoin then received hash value of empty string
//		  * if receiver shard has >= 1 outcoin then concatenate all outcoin bytes value then hash
//	      * At last, we compress all cross out put coin into a CrossOutputCoinFinalHash
//	   TXTOKENDATA
//		+ Do the same as above
//	   => Then Final Hash of each shard is Hash of value in this order:
//		1. CrossOutputCoinFinalHash
//		2. TxTokenDataVoutFinalHash
//	TxTokenOut DataStructure
//	- Use Only One TxNormalTokenData for one TokenID
//	- Vouts of one tokenID from many transaction will be compress into One Vouts List
//	- Using Key-Value structure for accessing one token ID data:
//	  key: token ID
//	  value: TokenData of that token
func getCrossShardDataHash(txList []metadata.Transaction) []common.Hash {
	// group transaction by shardID
	outCoinEachShard := make([][]privacy.OutputCoin, common.MaxShardNumber)
	txTokenPrivacyDataMap := make([]map[common.Hash]*types.ContentCrossShardTokenPrivacyData, common.MaxShardNumber)
	for _, tx := range txList {
		switch tx.GetType() {
		//==================For PRV Transfer Only
		//TxReturnStakingType cannot be crossshard tx
		case common.TxNormalType, common.TxRewardType:
			{
				// Proof Process
				if tx.GetProof() != nil {
					for _, outCoin := range tx.GetProof().GetOutputCoins() {
						lastByte := outCoin.CoinDetails.GetPubKeyLastByte()
						shardID := common.GetShardIDFromLastByte(lastByte)
						outCoinEachShard[shardID] = append(outCoinEachShard[shardID], *outCoin)
					}
				}
			}
		case common.TxCustomTokenPrivacyType:
			{
				customTokenPrivacyTx := tx.(*transaction.TxCustomTokenPrivacy)
				//==================Proof Process
				if customTokenPrivacyTx.GetProof() != nil {
					for _, outCoin := range customTokenPrivacyTx.GetProof().GetOutputCoins() {
						lastByte := outCoin.CoinDetails.GetPubKeyLastByte()
						shardID := common.GetShardIDFromLastByte(lastByte)
						outCoinEachShard[shardID] = append(outCoinEachShard[shardID], *outCoin)
					}
				}
				//==================Tx Token Privacy Data Process
				if customTokenPrivacyTx.TxPrivacyTokenData.TxNormal.GetProof() != nil {
					for _, outCoin := range customTokenPrivacyTx.TxPrivacyTokenData.TxNormal.GetProof().GetOutputCoins() {
						lastByte := outCoin.CoinDetails.GetPubKeyLastByte()
						shardID := common.GetShardIDFromLastByte(lastByte)
						if txTokenPrivacyDataMap[shardID] == nil {
							txTokenPrivacyDataMap[shardID] = make(map[common.Hash]*types.ContentCrossShardTokenPrivacyData)
						}
						if _, ok := txTokenPrivacyDataMap[shardID][customTokenPrivacyTx.TxPrivacyTokenData.PropertyID]; !ok {
							contentCrossTokenPrivacyData := types.CloneTxTokenPrivacyDataForCrossShard(customTokenPrivacyTx.TxPrivacyTokenData)
							txTokenPrivacyDataMap[shardID][customTokenPrivacyTx.TxPrivacyTokenData.PropertyID] = &contentCrossTokenPrivacyData
						}
						txTokenPrivacyDataMap[shardID][customTokenPrivacyTx.TxPrivacyTokenData.PropertyID].OutputCoin = append(txTokenPrivacyDataMap[shardID][customTokenPrivacyTx.TxPrivacyTokenData.PropertyID].OutputCoin, *outCoin)
					}
				}
			}
		}
	}
	//calcualte hash for each shard
	outputCoinHash := make([]common.Hash, common.MaxShardNumber)
	txTokenOutHash := make([]common.Hash, common.MaxShardNumber)
	txTokenPrivacyOutHash := make([]common.Hash, common.MaxShardNumber)
	combinedHash := make([]common.Hash, common.MaxShardNumber)
	for i := 0; i < common.MaxShardNumber; i++ {
		outputCoinHash[i] = calHashOutCoinCrossShard(outCoinEachShard[i])
		txTokenOutHash[i] = calHashTxTokenDataHashFromMap()
		txTokenPrivacyOutHash[i] = calHashTxTokenPrivacyDataHashFromMap(txTokenPrivacyDataMap[i])

		tmpByte := append(append(outputCoinHash[i].GetBytes(), txTokenOutHash[i].GetBytes()...), txTokenPrivacyOutHash[i].GetBytes()...)
		combinedHash[i] = common.HashH(tmpByte)
	}
	return combinedHash
}

func calHashOutCoinCrossShard(outCoins []privacy.OutputCoin) common.Hash {
	tmpByte := []byte{}
	var outputCoinHash common.Hash
	if len(outCoins) != 0 {
		for _, outCoin := range outCoins {
			coin := &outCoin

			tmpByte = append(tmpByte, coin.Bytes()...)
		}
		outputCoinHash = common.HashH(tmpByte)
	} else {
		outputCoinHash = common.HashH([]byte(""))
	}
	return outputCoinHash
}

func calHashTxTokenDataHashFromMap() common.Hash {
	return common.HashH([]byte(""))
}

func calHashTxTokenPrivacyDataHashFromMap(txTokenPrivacyDataMap map[common.Hash]*types.ContentCrossShardTokenPrivacyData) common.Hash {
	if len(txTokenPrivacyDataMap) == 0 {
		return common.HashH([]byte(""))
	}
	var txTokenPrivacyDataList []types.ContentCrossShardTokenPrivacyData
	for _, value := range txTokenPrivacyDataMap {
		txTokenPrivacyDataList = append(txTokenPrivacyDataList, *value)
	}
	sort.SliceStable(txTokenPrivacyDataList[:], func(i, j int) bool {
		return txTokenPrivacyDataList[i].PropertyID.String() < txTokenPrivacyDataList[j].PropertyID.String()
	})
	return calHashTxTokenPrivacyDataHashList(txTokenPrivacyDataList)
}

func calHashTxTokenPrivacyDataHashList(txTokenPrivacyDataList []types.ContentCrossShardTokenPrivacyData) common.Hash {
	tmpByte := []byte{}
	if len(txTokenPrivacyDataList) != 0 {
		for _, txTokenPrivacyData := range txTokenPrivacyDataList {
			tempHash := txTokenPrivacyData.Hash()
			tmpByte = append(tmpByte, tempHash.GetBytes()...)

		}
	} else {
		return common.HashH([]byte(""))
	}
	return common.HashH(tmpByte)
}

func calHashTxTokenDataHashList() common.Hash {
	return common.HashH([]byte(""))
}

func CreateMerkleCrossTransaction(crossTransactions map[byte][]types.CrossTransaction) (*common.Hash, error) {
	if len(crossTransactions) == 0 {
		res, err := generateZeroValueHash()
		return &res, err
	}
	keys := []int{}
	crossTransactionHashes := []*common.Hash{}
	for k := range crossTransactions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		for _, value := range crossTransactions[byte(shardID)] {
			hash := value.Hash()
			crossTransactionHashes = append(crossTransactionHashes, &hash)
		}
	}
	merkle := Merkle{}
	merkleTree := merkle.BuildMerkleTreeOfHashes(crossTransactionHashes, len(crossTransactionHashes))
	return merkleTree[len(merkleTree)-1], nil
}

func VerifyMerkleCrossTransaction(crossTransactions map[byte][]types.CrossTransaction, rootHash common.Hash) bool {
	res, err := CreateMerkleCrossTransaction(crossTransactions)
	if err != nil {
		return false
	}
	hashByte := rootHash.GetBytes()
	newHash, err := common.Hash{}.NewHash(hashByte)
	if err != nil {
		return false
	}
	return newHash.IsEqual(res)
}

//updateCommiteesWithAddedAndRemovedListValidator :
func updateCommiteesWithAddedAndRemovedListValidator(
	source,
	addedCommittees,
	removedCommittees []incognitokey.CommitteePublicKey) ([]incognitokey.CommitteePublicKey, error) {
	newShardPendingValidator := []incognitokey.CommitteePublicKey{}
	m := make(map[string]bool)
	for _, v := range removedCommittees {
		str, err := v.ToBase58()
		if err != nil {
			return nil, err
		}
		m[str] = true
	}
	for _, v := range source {
		str, err := v.ToBase58()
		if err != nil {
			return nil, err
		}
		if m[str] == false {
			newShardPendingValidator = append(newShardPendingValidator, v)
		}
	}
	newShardPendingValidator = append(newShardPendingValidator, addedCommittees...)

	return newShardPendingValidator, nil
}
