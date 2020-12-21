package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/privacy/coin"

	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
)

func FetchBeaconBlockFromHeight(blockchain *BlockChain, from uint64, to uint64) ([]*BeaconBlock, error) {
	beaconBlocks := []*BeaconBlock{}
	for i := from; i <= to; i++ {
		beaconHash, err := blockchain.GetBeaconBlockHashByHeight(blockchain.BeaconChain.GetFinalView(), blockchain.BeaconChain.GetBestView(), i)
		if err != nil {
			return nil, err
		}
		beaconBlockBytes, err := rawdbv2.GetBeaconBlockByHash(blockchain.GetBeaconChainDatabase(), *beaconHash)
		if err != nil {
			return beaconBlocks, err
		}
		beaconBlock := BeaconBlock{}
		err = json.Unmarshal(beaconBlockBytes, &beaconBlock)
		if err != nil {
			return beaconBlocks, NewBlockChainError(UnmashallJsonShardBlockError, err)
		}
		beaconBlocks = append(beaconBlocks, &beaconBlock)
	}
	return beaconBlocks, nil
}

func CreateCrossShardByteArray(txList []metadata.Transaction, fromShardID byte) ([]byte, error) {
	crossIDs := []byte{}
	byteMap := make([]byte, common.MaxShardNumber)
	for _, tx := range txList {
		var prvProof privacy.Proof
		switch tx.GetType() {
		case common.TxCustomTokenPrivacyType, common.TxTokenConversionType:
			{
				customTokenTx, ok := tx.(transaction.TransactionToken)
				if !ok {
					return nil, errors.New("Cannot cast transaction")
				}
				prvProof = customTokenTx.GetTxBase().GetProof()
				proof := customTokenTx.GetTxTokenData().TxNormal.GetProof()
				if proof != nil {
					for _, outCoin := range proof.GetOutputCoins() {
						if shardID, err := outCoin.GetShardID(); err != nil {
							panic("Cannot get shardID")
						} else {
							byteMap[common.GetShardIDFromLastByte(shardID)] = 1
						}
					}
				}
			}
		default:
			prvProof = tx.GetProof()
		}
		if prvProof != nil {
			for _, outCoin := range prvProof.GetOutputCoins() {
				shardID, err := outCoin.GetShardID()
				if err == nil {
					byteMap[common.GetShardIDFromLastByte(shardID)] = 1
				}
			}
		}
	}

	for k := range byteMap {
		if byteMap[k] == 1 && k != int(fromShardID) {
			crossIDs = append(crossIDs, byte(k))
		}
	}
	return crossIDs, nil
}

// CreateSwapInstruction creates swap instruction and return new validator list
// Return param:
// #1: swap instruction
// #2: new pending validator list after swapped
// #3: new committees after swapped
// #4: error
func CreateSwapInstruction(
	pendingValidator []string,
	commitees []string,
	maxCommitteeSize int,
	minCommitteeSize int,
	shardID byte,
	producersBlackList map[string]uint8,
	badProducersWithPunishment map[string]uint8,
	offset int,
	swapOffset int,
) ([]string, []string, []string, error) {
	newPendingValidator, newShardCommittees, shardSwapedCommittees, shardNewCommittees, err := SwapValidator(pendingValidator, commitees, maxCommitteeSize, minCommitteeSize, offset, producersBlackList, swapOffset)
	if err != nil {
		return nil, nil, nil, err
	}
	badProducersWithPunishmentBytes, err := json.Marshal(badProducersWithPunishment)
	if err != nil {
		return nil, nil, nil, err
	}
	swapInstruction := []string{"swap", strings.Join(shardNewCommittees, ","), strings.Join(shardSwapedCommittees, ","), "shard", strconv.Itoa(int(shardID)), string(badProducersWithPunishmentBytes)}
	return swapInstruction, newPendingValidator, newShardCommittees, nil
}

func CreateShardSwapActionForKeyListV2(
	genesisParam *GenesisParams,
	pendingValidator []string,
	shardCommittees []string,
	minCommitteeSize int,
	activeShard int,
	shardID byte,
	epoch uint64,
) ([]string, []string, []string) {
	newPendingValidator := pendingValidator
	swapInstruction, newShardCommittees := GetShardSwapInstructionKeyListV2(genesisParam, epoch, minCommitteeSize, activeShard)
	remainShardCommittees := shardCommittees[minCommitteeSize:]
	return swapInstruction[shardID], newPendingValidator, append(newShardCommittees[shardID], remainShardCommittees...)
}

// CreateShardInstructionsFromTransactionAndInstruction create instruction from transactions in shard block
// Stake:
//  ["stake", "pubkey1,pubkey2,..." "shard" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,..." "autostaking1,autostaking2,..."]
//	["stake", "pubkey1,pubkey2,..." "beacon" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,..." "autostaking1,autostaking2,..."]
// Stop Auto Staking:
//	["stopautostaking" "pubkey1,pubkey2,..."]
func CreateShardInstructionsFromTransactionAndInstruction(transactions []metadata.Transaction, bc *BlockChain, shardID byte) (instructions [][]string, err error) {
	// Generate stake action
	stakeShardPublicKey := []string{}
	stakeBeaconPublicKey := []string{}
	stakeShardTxID := []string{}
	stakeBeaconTxID := []string{}
	stakeShardRewardReceiver := []string{}
	stakeBeaconRewardReceiver := []string{}
	stakeShardAutoStaking := []string{}
	stakeBeaconAutoStaking := []string{}
	stopAutoStaking := []string{}
	for _, tx := range transactions {
		metadataValue := tx.GetMetadata()
		if metadataValue != nil {
			actionPairs, err := metadataValue.BuildReqActions(tx, bc, nil, bc.BeaconChain.GetFinalView().(*BeaconBestState), shardID)
			Logger.log.Infof("Build Request Action Pairs %+v, metadata value %+v", actionPairs, metadataValue)
			if err == nil {
				instructions = append(instructions, actionPairs...)
			} else {
				Logger.log.Errorf("Build Request Action Error %+v", err)
			}
		}
		switch tx.GetMetadataType() {
		case metadata.ShardStakingMeta:
			stakingMetadata, ok := tx.GetMetadata().(*metadata.StakingMetadata)
			if !ok {
				return nil, fmt.Errorf("Expect metadata type to be *metadata.StakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata()))
			}
			stakeShardPublicKey = append(stakeShardPublicKey, stakingMetadata.CommitteePublicKey)
			stakeShardTxID = append(stakeShardTxID, tx.Hash().String())
			stakeShardRewardReceiver = append(stakeShardRewardReceiver, stakingMetadata.RewardReceiverPaymentAddress)
			if stakingMetadata.AutoReStaking {
				stakeShardAutoStaking = append(stakeShardAutoStaking, "true")
			} else {
				stakeShardAutoStaking = append(stakeShardAutoStaking, "false")
			}
		case metadata.BeaconStakingMeta:
			stakingMetadata, ok := tx.GetMetadata().(*metadata.StakingMetadata)
			if !ok {
				return nil, fmt.Errorf("Expect metadata type to be *metadata.StakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata()))
			}
			stakeBeaconPublicKey = append(stakeBeaconPublicKey, stakingMetadata.CommitteePublicKey)
			stakeBeaconTxID = append(stakeBeaconTxID, tx.Hash().String())
			stakeBeaconRewardReceiver = append(stakeBeaconRewardReceiver, stakingMetadata.RewardReceiverPaymentAddress)
			if stakingMetadata.AutoReStaking {
				stakeBeaconAutoStaking = append(stakeBeaconAutoStaking, "true")
			} else {
				stakeBeaconAutoStaking = append(stakeBeaconAutoStaking, "false")
			}
		case metadata.StopAutoStakingMeta:
			{
				stopAutoStakingMetadata, ok := tx.GetMetadata().(*metadata.StopAutoStakingMetadata)
				if !ok {
					return nil, fmt.Errorf("Expect metadata type to be *metadata.StopAutoStakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata()))
				}
				stopAutoStaking = append(stopAutoStaking, stopAutoStakingMetadata.CommitteePublicKey)
			}
		}
	}
	if !reflect.DeepEqual(stakeShardPublicKey, []string{}) {
		if len(stakeShardPublicKey) != len(stakeShardTxID) && len(stakeShardTxID) != len(stakeShardRewardReceiver) && len(stakeShardRewardReceiver) != len(stakeShardAutoStaking) {
			return nil, NewBlockChainError(StakeInstructionError, fmt.Errorf("Expect public key list (length %+v) and reward receiver list (length %+v), auto restaking (length %+v) to be equal", len(stakeShardPublicKey), len(stakeShardRewardReceiver), len(stakeShardAutoStaking)))
		}
		stakeShardPublicKey, err = incognitokey.ConvertToBase58ShortFormat(stakeShardPublicKey)
		if err != nil {
			return nil, fmt.Errorf("Failed To Convert Stake Shard Public Key to Base58 Short Form")
		}
		// ["stake", "pubkey1,pubkey2,..." "shard" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,..." "flag1,flag2,..."]
		instruction := []string{StakeAction, strings.Join(stakeShardPublicKey, ","), "shard", strings.Join(stakeShardTxID, ","), strings.Join(stakeShardRewardReceiver, ","), strings.Join(stakeShardAutoStaking, ",")}
		instructions = append(instructions, instruction)
	}
	if !reflect.DeepEqual(stakeBeaconPublicKey, []string{}) {
		if len(stakeBeaconPublicKey) != len(stakeBeaconTxID) && len(stakeBeaconTxID) != len(stakeBeaconRewardReceiver) && len(stakeBeaconRewardReceiver) != len(stakeBeaconAutoStaking) {
			return nil, NewBlockChainError(StakeInstructionError, fmt.Errorf("Expect public key list (length %+v) and reward receiver list (length %+v), auto restaking (length %+v) to be equal", len(stakeBeaconPublicKey), len(stakeBeaconRewardReceiver), len(stakeBeaconAutoStaking)))
		}
		stakeBeaconPublicKey, err = incognitokey.ConvertToBase58ShortFormat(stakeBeaconPublicKey)
		if err != nil {
			return nil, fmt.Errorf("Failed To Convert Stake Beacon Public Key to Base58 Short Form")
		}
		// ["stake", "pubkey1,pubkey2,..." "beacon" "txStake1,txStake2,..." "rewardReceiver1,rewardReceiver2,..." "flag1,flag2,..."]
		instruction := []string{StakeAction, strings.Join(stakeBeaconPublicKey, ","), "beacon", strings.Join(stakeBeaconTxID, ","), strings.Join(stakeBeaconRewardReceiver, ","), strings.Join(stakeBeaconAutoStaking, ",")}
		instructions = append(instructions, instruction)
	}
	if !reflect.DeepEqual(stopAutoStaking, []string{}) {
		// ["stopautostaking" "pubkey1,pubkey2,..."]
		instruction := []string{StopAutoStake, strings.Join(stopAutoStaking, ",")}
		instructions = append(instructions, instruction)
	}
	return instructions, nil
}

func checkReturnStakingTxExistence(txId string, shardBlock *ShardBlock) bool {
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

// VerifyCrossShardBlockUTXO Calculate Final Hash as Hash of:
//	1. CrossTransactionFinalHash
//	2. TxTokenDataVoutFinalHash
//	3. CrossTxTokenPrivacyData
// These hashes will be calculated as comment in getCrossShardDataHash function
func VerifyCrossShardBlockUTXO(block *CrossShardBlock, merklePathShard []common.Hash) bool {
	var outputCoinHash common.Hash
	var txTokenDataHash common.Hash
	var txTokenPrivacyDataHash common.Hash
	outCoins := block.CrossOutputCoin
	outputCoinHash = calHashOutCoinCrossShard(outCoins)
	txTokenDataHash = calHashTxTokenDataHashList()
	txTokenPrivacyDataList := block.CrossTxTokenPrivacyData
	txTokenPrivacyDataHash = calHashTxTokenPrivacyDataHashList(txTokenPrivacyDataList)
	tmpByte := append(append(outputCoinHash.GetBytes(), txTokenDataHash.GetBytes()...), txTokenPrivacyDataHash.GetBytes()...)
	finalHash := common.HashH(tmpByte)
	return Merkle{}.VerifyMerkleRootFromMerklePath(finalHash, merklePathShard, block.Header.ShardTxRoot, block.ToShardID)
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
	outCoinEachShard := make([][]coin.Coin, common.MaxShardNumber)
	txTokenPrivacyDataMap := make([]map[common.Hash]*ContentCrossShardTokenPrivacyData, common.MaxShardNumber)
	for _, tx := range txList {
		switch tx.GetType() {
		//==================For PRV Transfer Only
		//TxReturnStakingType cannot be crossshard tx
		case common.TxNormalType, common.TxRewardType, common.TxConversionType:
			{
				// Proof Process
				if tx.GetProof() != nil {
					for _, outCoin := range tx.GetProof().GetOutputCoins() {
						shardID, err := outCoin.GetShardID()
						if err == nil {
							outCoinEachShard[shardID] = append(outCoinEachShard[shardID], outCoin)
						}
					}
				}
			}
		case common.TxCustomTokenPrivacyType, common.TxTokenConversionType:
			{
				customTokenPrivacyTx, ok := tx.(transaction.TransactionToken)
				if !ok {
					continue
				}
				//==================Proof Process
				txProof := customTokenPrivacyTx.GetTxBase().GetProof()
				if txProof != nil {
					for _, outCoin := range txProof.GetOutputCoins() {
						shardID, err := outCoin.GetShardID()
						if err == nil {
							outCoinEachShard[shardID] = append(outCoinEachShard[shardID], outCoin)
						}
					}
				}
				//==================Tx Token Privacy Data Process
				

				txTokenData := customTokenPrivacyTx.GetTxTokenData()
				txTokenProof := txTokenData.TxNormal.GetProof()
				if txTokenProof != nil {
					for _, outCoin := range txTokenProof.GetOutputCoins() {
						shardID, err := outCoin.GetShardID()
						if err == nil {
							if txTokenPrivacyDataMap[shardID] == nil {
								txTokenPrivacyDataMap[shardID] = make(map[common.Hash]*ContentCrossShardTokenPrivacyData)
							}
							if _, ok := txTokenPrivacyDataMap[shardID][txTokenData.PropertyID]; !ok {
								contentCrossTokenPrivacyData := cloneTxTokenPrivacyDataForCrossShard(txTokenData)
								txTokenPrivacyDataMap[shardID][txTokenData.PropertyID] = &contentCrossTokenPrivacyData
							}
							txTokenPrivacyDataMap[shardID][txTokenData.PropertyID].OutputCoin = append(txTokenPrivacyDataMap[shardID][txTokenData.PropertyID].OutputCoin, outCoin)
						}
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

// getCrossShardData get cross data (send to a shard) from list of transaction:
// 1. (Privacy) PRV: Output coin
// 2. Tx Custom Token: Tx Token Data
// 3. Privacy Custom Token: Token Data + Output coin
func getCrossShardData(txList []metadata.Transaction, shardID byte) ([]coin.Coin, []ContentCrossShardTokenPrivacyData, error) {
	coinList := []coin.Coin{}
	txTokenPrivacyDataMap := make(map[common.Hash]*ContentCrossShardTokenPrivacyData)
	var txTokenPrivacyDataList []ContentCrossShardTokenPrivacyData
	for _, tx := range txList {
		var prvProof privacy.Proof
		
		if tx.GetType() == common.TxCustomTokenPrivacyType || tx.GetType() == common.TxTokenConversionType {
			customTokenPrivacyTx, ok := tx.(transaction.TransactionToken)
			if !ok {
				return nil, nil, errors.New("Cannot cast transaction")
			}
			prvProof = customTokenPrivacyTx.GetTxBase().GetProof()
			txTokenData := customTokenPrivacyTx.GetTxTokenData()
			txTokenProof := txTokenData.TxNormal.GetProof()
			if txTokenProof != nil {
				for _, outCoin := range txTokenProof.GetOutputCoins() {
					coinShardID, err := outCoin.GetShardID()
					if err == nil && coinShardID == shardID {
						if _, ok := txTokenPrivacyDataMap[txTokenData.PropertyID]; !ok {
							contentCrossTokenPrivacyData := cloneTxTokenPrivacyDataForCrossShard(txTokenData)
							txTokenPrivacyDataMap[txTokenData.PropertyID] = &contentCrossTokenPrivacyData
						}
						txTokenPrivacyDataMap[txTokenData.PropertyID].OutputCoin = append(txTokenPrivacyDataMap[txTokenData.PropertyID].OutputCoin, outCoin)
					}
				}
			}
		}else{
			prvProof = tx.GetProof()
		}
		if prvProof != nil {
			for _, outCoin := range prvProof.GetOutputCoins() {
				coinShardID, err := outCoin.GetShardID()
				if err == nil && coinShardID == shardID {
					coinList = append(coinList, outCoin)
				}
			}
		}
	}
	if len(txTokenPrivacyDataMap) != 0 {
		for _, value := range txTokenPrivacyDataMap {
			txTokenPrivacyDataList = append(txTokenPrivacyDataList, *value)
		}
		sort.SliceStable(txTokenPrivacyDataList[:], func(i, j int) bool {
			return txTokenPrivacyDataList[i].PropertyID.String() < txTokenPrivacyDataList[j].PropertyID.String()
		})
	}
	return coinList, txTokenPrivacyDataList, nil
}

func calHashOutCoinCrossShard(outCoins []coin.Coin) common.Hash {
	tmpByte := []byte{}
	var outputCoinHash common.Hash
	if len(outCoins) != 0 {
		for _, outCoin := range outCoins {
			if outCoin == nil {
				for i := 0; i < len(outCoins); i++ {
					fmt.Println("Outcoin ", i, outCoins[i])
				}
				fmt.Println("Number of outcoins: ", len(outCoins))
			}
			tmpByte = append(tmpByte, outCoin.Bytes()...)
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

func calHashTxTokenDataHashList() common.Hash {
	return common.HashH([]byte(""))
}

func calHashTxTokenPrivacyDataHashFromMap(txTokenPrivacyDataMap map[common.Hash]*ContentCrossShardTokenPrivacyData) common.Hash {
	if len(txTokenPrivacyDataMap) == 0 {
		return common.HashH([]byte(""))
	}
	var txTokenPrivacyDataList []ContentCrossShardTokenPrivacyData
	for _, value := range txTokenPrivacyDataMap {
		txTokenPrivacyDataList = append(txTokenPrivacyDataList, *value)
	}
	sort.SliceStable(txTokenPrivacyDataList[:], func(i, j int) bool {
		return txTokenPrivacyDataList[i].PropertyID.String() < txTokenPrivacyDataList[j].PropertyID.String()
	})
	return calHashTxTokenPrivacyDataHashList(txTokenPrivacyDataList)
}

func calHashTxTokenPrivacyDataHashList(txTokenPrivacyDataList []ContentCrossShardTokenPrivacyData) common.Hash {
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

func cloneTxTokenPrivacyDataForCrossShard(txTokenPrivacyData transaction.TxTokenData) ContentCrossShardTokenPrivacyData {
	newContentCrossTokenPrivacyData := ContentCrossShardTokenPrivacyData{
		PropertyID:     txTokenPrivacyData.PropertyID,
		PropertyName:   txTokenPrivacyData.PropertyName,
		PropertySymbol: txTokenPrivacyData.PropertySymbol,
		Mintable:       txTokenPrivacyData.Mintable,
		Amount:         txTokenPrivacyData.Amount,
		Type:           transaction.CustomTokenCrossShard,
	}
	newContentCrossTokenPrivacyData.OutputCoin = []coin.Coin{}
	return newContentCrossTokenPrivacyData
}
func CreateMerkleCrossOutputCoin(crossOutputCoins map[byte][]CrossOutputCoin) (*common.Hash, error) {
	if len(crossOutputCoins) == 0 {
		res, err := generateZeroValueHash()

		return &res, err
	}
	keys := []int{}
	crossOutputCoinHashes := []*common.Hash{}
	for k := range crossOutputCoins {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		for _, value := range crossOutputCoins[byte(shardID)] {
			hash := value.Hash()
			hashByte := hash.GetBytes()
			newHash, err := common.Hash{}.NewHash(hashByte)
			if err != nil {
				return &common.Hash{}, NewBlockChainError(HashError, err)
			}
			crossOutputCoinHashes = append(crossOutputCoinHashes, newHash)
		}
	}
	merkle := Merkle{}
	merkleTree := merkle.BuildMerkleTreeOfHashes(crossOutputCoinHashes, len(crossOutputCoinHashes))
	return merkleTree[len(merkleTree)-1], nil
}

func VerifyMerkleCrossOutputCoin(crossOutputCoins map[byte][]CrossOutputCoin, rootHash common.Hash) bool {
	res, err := CreateMerkleCrossOutputCoin(crossOutputCoins)
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
func CreateMerkleCrossTransaction(crossTransactions map[byte][]CrossTransaction) (*common.Hash, error) {
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

func VerifyMerkleCrossTransaction(crossTransactions map[byte][]CrossTransaction, rootHash common.Hash) bool {
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
