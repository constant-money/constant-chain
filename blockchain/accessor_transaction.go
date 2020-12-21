package blockchain

import (
	"bytes"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/pkg/errors"
	"sort"
	"strconv"
	"strings"
)

var(
	EnableIndexingCoinByOTAKey bool
	outcoinReindexer *coinReindexer
)

// DecryptOutputCoinByKey process outputcoin to get outputcoin data which relate to keyset
// Param keyset: (private key, payment address, read only key)
// in case private key: return unspent outputcoin tx
// in case read only key: return all outputcoin tx with amount value
// in case payment address: return all outputcoin tx with no amount value
func DecryptOutputCoinByKey(transactionStateDB *statedb.StateDB, outCoin privacy.Coin, keySet *incognitokey.KeySet, tokenID *common.Hash, shardID byte) (privacy.PlainCoin, error) {
	result, err := outCoin.Decrypt(keySet)
	if err != nil {
		Logger.log.Errorf("Cannot decrypt output coin by key %v", err)
		return nil, err
	}
	keyImage := result.GetKeyImage()
	if keyImage != nil {
		ok, err := statedb.HasSerialNumber(transactionStateDB, *tokenID, keyImage.ToBytesS(), shardID)
		if err != nil {
			Logger.log.Errorf("There is something wrong when check key image %v", err)
			return nil, err
		} else if ok {
			// The KeyImage is valid but already spent
			return nil, nil
		}
	}
	return result, nil
}

func storePRV(transactionStateRoot *statedb.StateDB) error {
	tokenID := common.PRVCoinID
	name := common.PRVCoinName
	symbol := common.PRVCoinName
	tokenType := 0
	mintable := false
	amount := uint64(0)
	info := []byte{}
	txHash := common.Hash{}
	err := statedb.StorePrivacyToken(transactionStateRoot, tokenID, name, symbol, tokenType, mintable, amount, info, txHash)
	if err != nil {
		return err
	}
	return nil
}

func (blockchain *BlockChain) GetTransactionByHash(txHash common.Hash) (byte, common.Hash, uint64, int, metadata.Transaction, error) {
	for _, i := range blockchain.GetShardIDs() {
		shardID := byte(i)
		blockHash, index, err := rawdbv2.GetTransactionByHash(blockchain.GetShardChainDatabase(shardID), txHash)
		if err != nil {
			continue
		}
		// error is nil
		shardBlock, _, err := blockchain.GetShardBlockByHashWithShardID(blockHash, shardID)
		if err != nil {
			continue
		}
		return shardBlock.Header.ShardID, blockHash, shardBlock.GetHeight(), index, shardBlock.Body.Transactions[index], nil
	}
	return byte(255), common.Hash{}, 0, -1, nil, NewBlockChainError(GetTransactionFromDatabaseError, fmt.Errorf("Not found transaction with tx hash %+v", txHash))
}

func (blockchain *BlockChain) GetTransactionByHashWithShardID(txHash common.Hash, shardID byte) (common.Hash, int, metadata.Transaction, error) {
	blockHash, index, err := rawdbv2.GetTransactionByHash(blockchain.GetShardChainDatabase(shardID), txHash)
	if err != nil {
		return common.Hash{}, -1, nil, NewBlockChainError(GetTransactionFromDatabaseError, fmt.Errorf("Not found transaction with tx hash %+v", txHash))
	}
	// error is nil
	shardBlock, _, err := blockchain.GetShardBlockByHashWithShardID(blockHash, shardID)
	if err != nil {
		return common.Hash{}, -1, nil, NewBlockChainError(GetTransactionFromDatabaseError, fmt.Errorf("Not found transaction with tx hash %+v", txHash))
	}
	return blockHash, index, shardBlock.Body.Transactions[index], nil
}

// GetTransactionHashByReceiver - return list tx id which receiver get from any sender
// this feature only apply on full node, because full node get all data from all shard
func (blockchain *BlockChain) GetTransactionHashByReceiver(keySet *incognitokey.KeySet) (map[byte][]common.Hash, error) {
	result := make(map[byte][]common.Hash)
	for _, i := range blockchain.GetShardIDs() {
		shardID := byte(i)
		var err error
		resultTemp, err := rawdbv2.GetTxByPublicKey(blockchain.GetShardChainDatabase(shardID), keySet.PaymentAddress.Pk)
		if err == nil {
			if resultTemp == nil || len(resultTemp) == 0 {
				continue
			}
			result[shardID] = resultTemp[shardID]
		}
	}
	return result, nil
}

// GetTransactionHashByReceiverV2 - return list tx id which a receiver receives from any senders in paging fashion
// this feature only apply on full node, because full node get all data from all shard
func (blockchain *BlockChain) GetTransactionHashByReceiverV2(
	keySet *incognitokey.KeySet,
	skip, limit uint,
) (map[byte][]common.Hash, error) {
	result := make(map[byte][]common.Hash)
	for _, i := range blockchain.GetShardIDs() {
		shardID := byte(i)
		var err error
		var resultTemp map[byte][]common.Hash
		resultTemp, skip, limit, err = rawdbv2.GetTxByPublicKeyV2(blockchain.GetShardChainDatabase(shardID), keySet.PaymentAddress.Pk, skip, limit)
		if err == nil {
			if resultTemp == nil || len(resultTemp) == 0 {
				continue
			}
			result[shardID] = resultTemp[shardID]
		}
		if limit == 0 {
			break
		}
	}
	return result, nil
}

func (blockchain *BlockChain) ValidateResponseTransactionFromTxsWithMetadata(shardBlock *ShardBlock) error {
	// filter double withdraw request
	withdrawReqTable := make(map[string]privacy.PaymentAddress)
	for _, tx := range shardBlock.Body.Transactions {
		if tx.GetMetadataType() == metadata.WithDrawRewardRequestMeta {
			metaRequest := tx.GetMetadata().(*metadata.WithDrawRewardRequest)
			mapKey := fmt.Sprintf("%s-%s", base58.Base58Check{}.Encode(metaRequest.PaymentAddress .Pk, common.Base58Version), metaRequest.TokenID.String())
			if _, ok := withdrawReqTable[mapKey]; !ok {
				withdrawReqTable[mapKey] = metaRequest.PaymentAddress
			}
		}
	}
	// check tx withdraw response valid with the corresponding request
	for _, tx := range shardBlock.Body.Transactions {
		if tx.GetMetadataType() == metadata.WithDrawRewardResponseMeta {
			//check valid info with tx request
			metaResponse := tx.GetMetadata().(*metadata.WithDrawRewardResponse)
			mapKey := fmt.Sprintf("%s-%s", base58.Base58Check{}.Encode(metaResponse.RewardPublicKey, common.Base58Version), metaResponse.TokenID.String())
			rewardPaymentAddress, ok := withdrawReqTable[mapKey]
			if !ok {
				return errors.Errorf("[Mint Withdraw Reward] This response dont match with any request in this block - Reward Address: %v", mapKey)
			} else {
				delete(withdrawReqTable, mapKey)
			}
			isMinted, mintCoin, coinID, err := tx.GetTxMintData()
			//check tx mint
			if err != nil || !isMinted {
				return errors.Errorf("[Mint Withdraw Reward] It is not tx mint with error: %v", err)
			}
			//check tokenID
			if cmp, err := metaResponse.TokenID.Cmp(coinID); err != nil || cmp != 0 {
				return errors.Errorf("[Mint Withdraw Reward] Token dont match: %v and %v", metaResponse.TokenID.String(), coinID.String())
			}

			//check amount & receiver
			rewardAmount, err := statedb.GetCommitteeReward(blockchain.GetBestStateShard(shardBlock.Header.ShardID).GetShardRewardStateDB(),
				base58.Base58Check{}.Encode(metaResponse.RewardPublicKey, common.Base58Version), *coinID)
			if err != nil {
				return errors.Errorf("[Mint Withdraw Reward] Cannot get reward amount")
			}
			if ok := mintCoin.CheckCoinValid(rewardPaymentAddress, metaResponse.SharedRandom, rewardAmount); !ok {
				return errors.Errorf("[Mint Withdraw Reward] Mint Coin is invalid for receiver or amount")
			}
		}
	}

	return nil
}

func (blockchain *BlockChain) ValidateResponseTransactionFromBeaconInstructions(
	curView *ShardBestState,
	shardBlock *ShardBlock,
	beaconBlocks []*BeaconBlock,
	shardID byte,
) error {
	//mainnet have two block return double when height < REPLACE_STAKINGTX
	if len(beaconBlocks) > 0 && beaconBlocks[0].GetHeight() < blockchain.config.ChainParams.ReplaceStakingTxHeight {
		return nil
	}
	return blockchain.ValidateReturnStakingTxFromBeaconInstructions(
		curView,
		beaconBlocks,
		shardBlock,
		shardID,
	)
}

func (blockchain *BlockChain) InitTxSalaryByCoinID(
	payToAddress *privacy.PaymentAddress,
	amount uint64,
	payByPrivateKey *privacy.PrivateKey,
	transactionStateDB *statedb.StateDB,
	bridgeStateDB *statedb.StateDB,
	meta *metadata.WithDrawRewardResponse,
	coinID common.Hash,
	shardID byte,
) (metadata.Transaction, error) {
	txType := -1
	if res, err := coinID.Cmp(&common.PRVCoinID); err == nil && res == 0 {
		txType = transaction.NormalCoinType
	}
	if txType == -1 {
		tokenIDs, err := blockchain.ListPrivacyTokenAndBridgeTokenAndPRVByShardID(shardID)
		if err != nil {
			return nil, err
		}
		// coinID must not equal to PRVCoinID
		for _, tokenID := range tokenIDs {
			if res, err := coinID.Cmp(&tokenID); err == nil && res == 0 {
				txType = transaction.CustomTokenPrivacyType
				break
			}
		}
	}
	if txType == -1 {
		return nil, errors.Errorf("Invalid token ID when InitTxSalaryByCoinID. Got %v", coinID)
	}
	buildCoinBaseParams := transaction.NewBuildCoinBaseTxByCoinIDParams(payToAddress,
		amount,
		payByPrivateKey,
		transactionStateDB,
		meta,
		coinID,
		txType,
		coinID.String(),
		shardID,
		bridgeStateDB)
	return transaction.BuildCoinBaseTxByCoinID(buildCoinBaseParams)
}

// @Notice: change from body.Transaction -> transactions
func (blockchain *BlockChain) BuildResponseTransactionFromTxsWithMetadata(view *ShardBestState, transactions []metadata.Transaction, blkProducerPrivateKey *privacy.PrivateKey, shardID byte) ([]metadata.Transaction, error) {
	withdrawReqTable := make(map[string]metadata.Transaction)
	for _, tx := range transactions {
		if tx.GetMetadataType() == metadata.WithDrawRewardRequestMeta {
			metaReq := tx.GetMetadata().(*metadata.WithDrawRewardRequest)
			mapKey := fmt.Sprintf("%s-%s", base58.Base58Check{}.Encode(metaReq.PaymentAddress.Pk, common.Base58Version), metaReq.TokenID.String())
			if _, ok := withdrawReqTable[mapKey]; !ok {
				withdrawReqTable[mapKey] = tx
			}
		}
	}
	txsResponse := []metadata.Transaction{}
	for _, txRequest := range withdrawReqTable {
		txResponse, err := blockchain.buildWithDrawTransactionResponse(view, &txRequest, blkProducerPrivateKey, shardID)
		if err != nil {
			Logger.log.Errorf("[Withdraw Reward] Build transactions response for tx %v return errors %v", txRequest, err)
			continue
		}
		txsResponse = append(txsResponse, txResponse)
		Logger.log.Infof("[Withdraw Reward] - BuildWithDrawTransactionResponse for tx %+v, ok: %+v\n", txRequest, txResponse)
	}
	return append(transactions, txsResponse...), nil
}

func queryDbCoinVer1(pubkey []byte, shardID byte, tokenID *common.Hash, db *statedb.StateDB) ([]privacy.Coin, error) {
	outCoinsBytes, err := statedb.GetOutcoinsByPubkey(db, *tokenID, pubkey, shardID)
	if err != nil {
		Logger.log.Error("GetOutcoinsBytesByKeyset Get by PubKey", err)
		return nil, err
	}
	var outCoins []privacy.Coin
	for _, item := range outCoinsBytes{
		outCoin, err := coin.NewCoinFromByte(item)
		if err != nil {
			Logger.log.Errorf("Cannot create coin from byte %v", err)
			return nil, err
		}
		outCoins = append(outCoins, outCoin)
	}
	return outCoins, nil
}

type coinMatcher func(*privacy.CoinV2, map[string]interface{}) bool

func queryDbCoinVer2(otaKey privacy.OTAKey, shardID byte, tokenID *common.Hash, shardHeight, destHeight uint64, db *statedb.StateDB, filters ...coinMatcher) ([]privacy.Coin, error) {
	var outCoins []privacy.Coin
	for height := shardHeight; height <= destHeight; height += 1 {
		currentHeightCoins, err := statedb.GetOTACoinsByHeight(db, *tokenID, shardID, height)
		if err != nil {
			Logger.log.Error("Get outcoins ver 2 bytes by keyset get by height", err)
			return nil, err
		}
		params := make(map[string]interface{})
		params["otaKey"] = otaKey
		params["db"] = db
		params["tokenID"] = tokenID
		for _, coinBytes := range currentHeightCoins {
			c, err := coin.NewCoinFromByte(coinBytes)
			if err != nil {
				Logger.log.Error("Get outcoins ver 2 bytes by keyset Parse Coin From Bytes", err)
				return nil, err
			}
			cv2, ok := c.(*privacy.CoinV2)
			if !ok{
				Logger.log.Error("Get outcoins ver 2 bytes by keyset cast coin to version 2", err)
				return nil, errors.New("Cannot cast a coin to version 2")
			}
			pass := true
			for _, f := range filters{
				if !f(cv2, params){
					pass = false
				}
			}
			if pass{
				outCoins = append(outCoins, cv2)
			}
		}
	}
	return outCoins, nil
}

func getCoinFilterByOTAKeyAndToken() coinMatcher{
	return func(c *privacy.CoinV2, kvargs map[string]interface{}) bool{
		entry, exists := kvargs["otaKey"]
		if !exists{
			return false
		}
		vk, ok := entry.(privacy.OTAKey)
		if !ok{
			return false
		}
		entry, exists = kvargs["tokenID"]
		if !exists{
			return false
		}
		tokenID, ok := entry.(*common.Hash)
		if !ok{
			return false
		}
		ks := &incognitokey.KeySet{}
		ks.OTAKey = vk

		if pass, sharedSecret := c.DoesCoinBelongToKeySet(ks); pass {
			pass, _ = c.ValidateAssetTag(sharedSecret, tokenID)
			return pass
		}
		return false
	}
}

//Return all coins belonging to the provided keyset
//
//If there is a ReadonlyKey, return decrypted coins; otherwise, just return raw coins
func (blockchain *BlockChain) getOutputCoins(keyset *incognitokey.KeySet, shardID byte, tokenID *common.Hash, upToHeight uint64, versionsIncluded map[int]bool) ([]privacy.PlainCoin, []privacy.Coin, uint64, error) {
	var outCoins []privacy.Coin
	var lowerHeight uint64 = 0
	if keyset == nil {
		return nil, nil, 0, NewBlockChainError(GetListDecryptedOutputCoinsByKeysetError, fmt.Errorf("invalid key set, got keyset %+v", keyset))
	}
	bss := blockchain.GetBestStateShard(shardID)
	transactionStateDB := blockchain.GetBestStateTransactionStateDB(shardID)

	if versionsIncluded[1]{
		results, err := queryDbCoinVer1(keyset.PaymentAddress.Pk, shardID, tokenID, transactionStateDB)
		if err != nil {
			return nil, nil, 0, err
		}
		outCoins = append(outCoins, results...)
	}
	if versionsIncluded[2]{
		if keyset.OTAKey.GetOTASecretKey() == nil || keyset.OTAKey.GetPublicSpend() == nil {
			return nil, nil, 0, errors.New("OTA secretKey is needed when retrieving coinV2")
		}
		if keyset.PaymentAddress.GetOTAPublicKey() == nil {
			return nil, nil, 0, errors.New("OTA publicKey is needed when retrieving coinV2")
		}
		latest := bss.ShardHeight
		if upToHeight > latest || upToHeight==0{
			upToHeight = latest
		}
		lowerHeight = getLowerHeight(upToHeight)
		fromHeight := uint64(0)
		if lowerHeight!=0{
			fromHeight = lowerHeight
		}
		results, err := queryDbCoinVer2(keyset.OTAKey, shardID, tokenID, fromHeight, upToHeight, transactionStateDB, getCoinFilterByOTAKeyAndToken())
		if err != nil {
			return nil, nil, 0, err
		}
		outCoins = append(outCoins, results...)
	}

	//If ReadonlyKey found, return decrypted coins
	if keyset.ReadonlyKey.GetPrivateView() != nil && keyset.ReadonlyKey.GetPublicSpend() != nil{
		resultPlainCoins := make([]privacy.PlainCoin, 0)
		for _, outCoin := range outCoins {
			decryptedOut, _ := DecryptOutputCoinByKey(transactionStateDB, outCoin, keyset, tokenID, shardID)
			if decryptedOut != nil {
				resultPlainCoins = append(resultPlainCoins, decryptedOut)
			}
		}
		
		return resultPlainCoins, nil, lowerHeight, nil
	}else{//Just return the raw coins
		return nil, outCoins, lowerHeight, nil
	}
	
}

func (blockchain *BlockChain) GetListDecryptedOutputCoinsVer2ByKeyset(keyset *incognitokey.KeySet, shardID byte, tokenID *common.Hash, startHeight uint64) ([]privacy.PlainCoin, []privacy.Coin, uint64, error) {
	return blockchain.getOutputCoins(keyset, shardID, tokenID, startHeight, map[int]bool{2:true})
}

func (blockchain *BlockChain) GetListDecryptedOutputCoinsVer1ByKeyset(keyset *incognitokey.KeySet, shardID byte, tokenID *common.Hash) ([]privacy.PlainCoin, []privacy.Coin, error) {
	resPlainCoins, resCoins, _, err := blockchain.getOutputCoins(keyset, shardID, tokenID, 0, map[int]bool{1:true})
	return resPlainCoins, resCoins, err
}

//GetListDecryptedOutputCoinsByKeyset - Read all blocks to get txs(not action tx) which can be decrypt by readonly secret key.
//With private-key, we can check unspent tx by check serialNumber from database
//- Param #1: keyset - (priv-key, payment-address, readonlykey)
//in case priv-key: return unspent outputcoin tx
//in case readonly-key: return all outputcoin tx with amount value
//in case payment-address: return all outputcoin tx with no amount value
//- Param #2: coinType - which type of joinsplitdesc(COIN or BOND)
func (blockchain *BlockChain) GetListDecryptedOutputCoinsByKeyset(keyset *incognitokey.KeySet, shardID byte, tokenID *common.Hash, shardHeight uint64) ([]privacy.PlainCoin, []privacy.Coin, uint64, error) {
	return blockchain.getOutputCoins(keyset, shardID, tokenID, shardHeight, map[int]bool{1:true, 2:true})
}

func (blockchain *BlockChain) SubmitOTAKey(theKey privacy.OTAKey, shardID byte) error{
	if !EnableIndexingCoinByOTAKey{
		return errors.New("OTA key submission not supported by this node configuration")
	}
	bss := blockchain.GetBestStateShard(shardID)
	transactionStateDB := blockchain.GetBestStateTransactionStateDB(shardID)

	go outcoinReindexer.ReindexOutcoin(bss.ShardHeight, theKey, transactionStateDB, shardID)
	return nil
}

func (blockchain *BlockChain) TryGetAllOutputCoinsByKeyset(keyset *incognitokey.KeySet, shardID byte, tokenID *common.Hash, withVersion1 bool) ([]privacy.PlainCoin,  error) {
	transactionStateDB := blockchain.GetBestStateTransactionStateDB(shardID)

	if !EnableIndexingCoinByOTAKey{
		return nil, errors.New("Getting all coins not supported by this node configuration")
		// results, _, _, err := blockchain.getOutputCoins(keyset, shardID, tokenID, bss.ShardHeight, map[int]bool{1:withVersion1, 2:true})
		// return results, err
	}

	outCoins, state, err := outcoinReindexer.GetReindexedOutcoin(keyset.OTAKey, tokenID, transactionStateDB, shardID)
	switch state{
	case 2:
		var results []privacy.PlainCoin
		if withVersion1{
			results, _, _, err = blockchain.getOutputCoins(keyset, shardID, tokenID, 0, map[int]bool{1:true})
			if err!=nil{
				return nil, err
			}
		}
		for _, outCoin := range outCoins {
			decryptedOut, _ := DecryptOutputCoinByKey(transactionStateDB, outCoin, keyset, tokenID, shardID)
			if decryptedOut != nil {
				results = append(results, decryptedOut)
			}
		}
		Logger.log.Infof("Retrieved output coins ver2 for view key %v", keyset.OTAKey.GetOTASecretKey())
		return results, nil
	case 1:
		return nil, errors.New("OTA Key indexing is in progress")
	case 0:
		err := blockchain.SubmitOTAKey(keyset.OTAKey, shardID)
		if err==nil{
			return nil, errors.New("Subscribed to OTA key to view all coins")
		}
		return nil, err
	default:
		return nil, errors.New("OTA Key indexing state is corrupted")
	}
}

// CreateAndSaveTxViewPointFromBlock - fetch data from block, put into txviewpoint variable and save into db
// still storage full data of commitments, serial number, snderivator to check double spend
// this function only work for transaction transfer token/prv within shard
func (blockchain *BlockChain) CreateAndSaveTxViewPointFromBlock(shardBlock *ShardBlock, transactionStateRoot *statedb.StateDB) error {
	// Fetch data from shardBlock into tx View point
	if shardBlock.Header.Height == 1 {
		err := storePRV(transactionStateRoot)
		if err != nil {
			return err
		}
	}
	var err error
	_, allBridgeTokens, err := blockchain.GetAllBridgeTokens()
	if err != nil {
		return err
	}
	view := NewTxViewPoint(shardBlock.Header.ShardID, shardBlock.Header.Height)
	err = view.fetchTxViewPointFromBlock(transactionStateRoot, shardBlock)
	if err != nil {
		return err
	}
	// check privacy custom token
	// sort by index
	indices := []int{}
	for index := range view.privacyCustomTokenViewPoint {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)
	for _, indexTx := range indices {
		privacyCustomTokenSubView := view.privacyCustomTokenViewPoint[int32(indexTx)]
		privacyCustomTokenTx := view.privacyCustomTokenTxs[int32(indexTx)]
		tokenData := privacyCustomTokenTx.GetTxTokenData()
		isBridgeToken := false
		for _, tempBridgeToken := range allBridgeTokens {
			if tempBridgeToken.TokenID != nil && bytes.Equal(tokenData.PropertyID[:], tempBridgeToken.TokenID[:]) {
				isBridgeToken = true
			}
		}
		switch tokenData.Type {
		case transaction.CustomTokenInit:
			{
				tokenID := tokenData.PropertyID
				existed := statedb.PrivacyTokenIDExisted(transactionStateRoot, tokenID)
				if !existed {
					// check is bridge token
					tokenID := tokenData.PropertyID
					name := tokenData.PropertyName
					symbol := tokenData.PropertySymbol
					mintable := tokenData.Mintable
					amount := tokenData.Amount
					info := privacyCustomTokenTx.GetInfo()
					txHash := *privacyCustomTokenTx.Hash()
					tokenType := statedb.InitToken
					if isBridgeToken {
						tokenType = statedb.BridgeToken
					}
					Logger.log.Info("Store custom token when it is issued", tokenData.PropertyID, tokenData.PropertySymbol, tokenData.PropertyName)
					err := statedb.StorePrivacyToken(transactionStateRoot, tokenID, name, symbol, tokenType, mintable, amount, info, txHash)
					if err != nil {
						return err
					}
				}
			}
		case transaction.CustomTokenTransfer:
			{
				Logger.log.Infof("Transfer custom token %+v", privacyCustomTokenTx)
			}
		}
		err = statedb.StorePrivacyTokenTx(transactionStateRoot, tokenData.PropertyID, *privacyCustomTokenTx.Hash())
		if err != nil {
			return err
		}

		err = blockchain.StoreSerialNumbersFromTxViewPoint(transactionStateRoot, *privacyCustomTokenSubView)
		if err != nil {
			return err
		}

		err = blockchain.StoreCommitmentsFromTxViewPoint(transactionStateRoot, *privacyCustomTokenSubView, shardBlock.Header.ShardID)
		if err != nil {
			return err
		}

		err = blockchain.StoreOnetimeAddressesFromTxViewPoint(transactionStateRoot, *privacyCustomTokenSubView, shardBlock.Header.ShardID)
		if err != nil {
			return err
		}

		err = blockchain.StoreSNDerivatorsFromTxViewPoint(transactionStateRoot, *privacyCustomTokenSubView)
		if err != nil {
			return err
		}
	}

	// updateShardBestState the list serialNumber and commitment, snd set using the state of the used tx view point. This
	// entails adding the new
	// ones created by the shardBlock.
	err = blockchain.StoreSerialNumbersFromTxViewPoint(transactionStateRoot, *view)
	if err != nil {
		return err
	}

	err = blockchain.StoreCommitmentsFromTxViewPoint(transactionStateRoot, *view, shardBlock.Header.ShardID)
	if err != nil {
		return err
	}

	err = blockchain.StoreOnetimeAddressesFromTxViewPoint(transactionStateRoot, *view, shardBlock.Header.ShardID)
	if err != nil {
		return err
	}

	err = blockchain.StoreSNDerivatorsFromTxViewPoint(transactionStateRoot, *view)
	if err != nil {
		return err
	}

	err = blockchain.StoreTxByPublicKey(blockchain.GetShardChainDatabase(shardBlock.Header.ShardID), view)
	if err != nil {
		return err
	}
	return nil
}

func (blockchain *BlockChain) StoreSerialNumbersFromTxViewPoint(stateDB *statedb.StateDB, view TxViewPoint) error {
	if len(view.listSerialNumbers) > 0 {
		err := statedb.StoreSerialNumbers(stateDB, *view.tokenID, view.listSerialNumbers, view.shardID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (blockchain *BlockChain) StoreSNDerivatorsFromTxViewPoint(stateDB *statedb.StateDB, view TxViewPoint) error {
	keys := make([]string, 0, len(view.mapCommitments))
	for k := range view.mapCommitments {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		snDsArray := view.mapSnD[k]
		err := statedb.StoreSNDerivators(stateDB, *view.tokenID, snDsArray)
		if err != nil {
			return err
		}
	}
	return nil
}

func (blockchain *BlockChain) StoreTxByPublicKey(db incdb.Database, view *TxViewPoint) error {
	for data := range view.txByPubKey {
		dataArr := strings.Split(data, "_")
		pubKey, _, err := base58.Base58Check{}.Decode(dataArr[0])
		if err != nil {
			return err
		}
		txIDInByte, _, err := base58.Base58Check{}.Decode(dataArr[1])
		if err != nil {
			return err
		}
		txID := common.Hash{}
		err = txID.SetBytes(txIDInByte)
		if err != nil {
			return err
		}
		shardID, _ := strconv.Atoi(dataArr[2])

		err = rawdbv2.StoreTxByPublicKey(db, pubKey, txID, byte(shardID))
		if err != nil {
			return err
		}
	}
	return nil
}

func (blockchain *BlockChain) StoreOnetimeAddressesFromTxViewPoint(stateDB *statedb.StateDB, view TxViewPoint, shardID byte) error {
	// commitment and output are the same key in map
	keys := make([]string, 0, len(view.mapCommitments))
	for k := range view.mapCommitments {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Start to store to db
	for _, publicKey := range keys {
		publicKeyBytes, _, err := base58.Base58Check{}.Decode(publicKey)
		if err != nil {
			return err
		}
		publicKeyShardID := common.GetShardIDFromLastByte(publicKeyBytes[len(publicKeyBytes)-1])
		if publicKeyShardID == shardID {
			// outputs
			outputCoinArray := view.mapOutputCoins[publicKey]
			otaCoinArray := make([][]byte, 0)
			onetimeAddressArray := make([][]byte, 0)
			for _, outputCoin := range outputCoinArray {
				if outputCoin.GetVersion() != 2 {
					continue
				}
				if EnableIndexingCoinByOTAKey{
					handler := func(k, v interface{}) bool{
						vkArr, ok := k.([64]byte)
						if !ok{
							return false
						}
						processing, ok := v.(int)
						if !ok{
							return false
						}
						if processing!=1 && processing!=2{
							return false
						}
						otaKey := OTAKeyFromRaw(vkArr)
						ks := &incognitokey.KeySet{}
						ks.OTAKey = otaKey
						belongs, _ := outputCoin.DoesCoinBelongToKeySet(ks)
						if belongs{
							outcoinReindexer.StoreReindexedOutputCoins(otaKey, []privacy.Coin{outputCoin}, shardID)
						}
						return true
					}
					outcoinReindexer.ManagedOTAKeys.Range(handler)
				}
				otaCoinArray = append(otaCoinArray, outputCoin.Bytes())
				onetimeAddressArray = append(onetimeAddressArray, outputCoin.GetPublicKey().ToBytesS())
			}
			if err = statedb.StoreOTACoinsAndOnetimeAddresses(stateDB, *view.tokenID, view.height, otaCoinArray, onetimeAddressArray, publicKeyShardID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (blockchain *BlockChain) StoreCommitmentsFromTxViewPoint(stateDB *statedb.StateDB, view TxViewPoint, shardID byte) error {
	// commitment and output are the same key in map
	keys := make([]string, 0, len(view.mapCommitments))
	for k := range view.mapCommitments {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Start to store to db
	for _, publicKey := range keys {
		publicKeyBytes, _, err := base58.Base58Check{}.Decode(publicKey)
		if err != nil {
			return err
		}
		publicKeyShardID := common.GetShardIDFromLastByte(publicKeyBytes[len(publicKeyBytes)-1])
		if publicKeyShardID == shardID {
			// outputs
			outputCoinArray := view.mapOutputCoins[publicKey]
			outputCoinBytesArray := make([][]byte, 0)
			for _, outputCoin := range outputCoinArray {
				if outputCoin.GetVersion() == 1 {
					outputCoinBytesArray = append(outputCoinBytesArray, outputCoin.Bytes())
				}
			}
			err = statedb.StoreOutputCoins(stateDB, *view.tokenID, publicKeyBytes, outputCoinBytesArray, publicKeyShardID)
			if err != nil {
				return err
			}

			// commitment
			commitmentsArray := view.mapCommitments[publicKey]
			err = statedb.StoreCommitments(stateDB, *view.tokenID, commitmentsArray, view.shardID)
			if err != nil {
				return err
			}

			// clear cached data
			if blockchain.config.MemCache != nil {
				cachedKey := memcache.GetListOutputcoinCachedKey(publicKeyBytes, view.tokenID, publicKeyShardID)
				if ok, e := blockchain.config.MemCache.Has(cachedKey); ok && e == nil {
					er := blockchain.config.MemCache.Delete(cachedKey)
					if er != nil {
						Logger.log.Error("can not delete memcache", "GetListOutputcoinCachedKey", base58.Base58Check{}.Encode(cachedKey, 0x0))
					}
				}
			}
		}
	}
	return nil
}

func (blockchain *BlockChain) CreateAndSaveCrossTransactionViewPointFromBlock(shardBlock *ShardBlock, transactionStateRoot *statedb.StateDB) error {
	Logger.log.Critical("Fetch Cross transaction", shardBlock.Body.CrossTransactions)
	// Fetch data from block into tx View point
	view := NewTxViewPoint(shardBlock.Header.ShardID, shardBlock.Header.Height)
	err := view.fetchCrossTransactionViewPointFromBlock(transactionStateRoot, shardBlock)
	if err != nil {
		Logger.log.Error("CreateAndSaveCrossTransactionCoinViewPointFromBlock ", err)
		return err
	}

	// sort by index
	indices := []int{}
	for index := range view.privacyCustomTokenViewPoint {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)

	for _, index := range indices {
		privacyCustomTokenSubView := view.privacyCustomTokenViewPoint[int32(index)]
		// 0xsirrush updated: check existed tokenID
		tokenID := *privacyCustomTokenSubView.tokenID
		existed := statedb.PrivacyTokenIDExisted(transactionStateRoot, tokenID)
		if !existed {
			Logger.log.Info("Store custom token when it is issued ", tokenID, privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertyName, privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertySymbol, privacyCustomTokenSubView.privacyCustomTokenMetadata.Amount, privacyCustomTokenSubView.privacyCustomTokenMetadata.Mintable)
			name := privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertyName
			symbol := privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertySymbol
			mintable := privacyCustomTokenSubView.privacyCustomTokenMetadata.Mintable
			amount := privacyCustomTokenSubView.privacyCustomTokenMetadata.Amount
			info := []byte{}
			if err := statedb.StorePrivacyToken(transactionStateRoot, tokenID, name, symbol, statedb.CrossShardToken, mintable, amount, info, common.Hash{}); err != nil {
				return err
			}
		}
		// Store both commitment and outcoin
		err = blockchain.StoreCommitmentsFromTxViewPoint(transactionStateRoot, *privacyCustomTokenSubView, shardBlock.Header.ShardID)
		if err != nil {
			return err
		}

		err = blockchain.StoreOnetimeAddressesFromTxViewPoint(transactionStateRoot, *privacyCustomTokenSubView, shardBlock.Header.ShardID)
		if err != nil {
			return err
		}

		// store snd
		err = blockchain.StoreSNDerivatorsFromTxViewPoint(transactionStateRoot, *privacyCustomTokenSubView)
		if err != nil {
			return err
		}
	}
	// store commitment
	err = blockchain.StoreCommitmentsFromTxViewPoint(transactionStateRoot, *view, shardBlock.Header.ShardID)
	if err != nil {
		return err
	}

	// store otas
	err = blockchain.StoreOnetimeAddressesFromTxViewPoint(transactionStateRoot, *view, shardBlock.Header.ShardID)
	if err != nil {
		return err
	}

	// store snd
	err = blockchain.StoreSNDerivatorsFromTxViewPoint(transactionStateRoot, *view)
	if err != nil {
		return err
	}
	return nil
}
