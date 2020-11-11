package rpcservice

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"strconv"

	rCommon "github.com/ethereum/go-ethereum/common"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/mempool"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/transaction"
)

type BlockService struct {
	BlockChain *blockchain.BlockChain
	DB         map[int]incdb.Database
	TxMemPool  *mempool.TxPool
	MemCache   *memcache.MemoryCache
}

func (blockService BlockService) GetShardBestStates() map[byte]*blockchain.ShardBestState {
	return blockService.BlockChain.GetClonedAllShardBestState()
}

func (blockService BlockService) GetShardBestStateByShardID(shardID byte) (*blockchain.ShardBestState, error) {
	if blockService.IsShardBestStateNil() {
		return nil, errors.New("Best State shard not existed")
	}
	shard, err := blockService.BlockChain.GetClonedAShardBestState(shardID)
	return shard, err
}

func (blockService BlockService) GetShardBestBlockHashes() map[int]common.Hash {
	bestBlockHashes := make(map[int]common.Hash)
	shards := blockService.BlockChain.GetClonedAllShardBestState()
	for shardID, best := range shards {
		bestBlockHashes[int(shardID)] = best.BestBlockHash
	}
	return bestBlockHashes
}

func (blockService BlockService) GetBeaconBestState() (*blockchain.BeaconBestState, error) {
	if blockService.IsBeaconBestStateNil() {
		Logger.log.Debugf("handleGetBeaconBestState result: %+v", nil)
		return nil, errors.New("Best State beacon not existed")
	}
	return blockService.BlockChain.GetClonedBeaconBestState()
}

func (blockService BlockService) GetBeaconBestBlockHash() (*common.Hash, error) {
	clonedBeaconBestState, err := blockService.BlockChain.GetClonedBeaconBestState()
	if err != nil {
		return nil, err
	}
	return &clonedBeaconBestState.BestBlockHash, nil
}

func (blockService BlockService) RetrieveShardBlock(hashString string, verbosity string) (*jsonresult.GetShardBlockResult, *RPCError) {
	hash, errH := common.Hash{}.NewHashFromStr(hashString)
	if errH != nil {
		Logger.log.Debugf("handleRetrieveBlock result: %+v, err: %+v", nil, errH)
		return nil, NewRPCError(RPCInvalidParamsError, errH)
	}
	shardBlock, _, errD := blockService.BlockChain.GetShardBlockByHash(*hash)
	if errD != nil {
		Logger.log.Debugf("handleRetrieveBlock result: %+v, err: %+v", nil, errD)
		return nil, NewRPCError(GetShardBlockByHashError, errD)
	}
	result := jsonresult.GetShardBlockResult{}

	shardID := shardBlock.Header.ShardID
	if verbosity == "0" {
		data, err := json.Marshal(shardBlock)
		if err != nil {
			Logger.log.Debugf("handleRetrieveBlock result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(JsonError, err)
		}
		result.Data = hex.EncodeToString(data)
	} else if verbosity == "1" {
		best := blockService.BlockChain.GetBestStateShard(byte(shardID)).BestBlock
		blockHeight := shardBlock.Header.Height
		// Get next block hash unless there are none.
		var nextHashString string
		// if blockHeight < best.Header.GetHeight() {
		if blockHeight < best.Header.Height {
			nextShardBlocks, err := blockService.BlockChain.GetShardBlockByHeight(blockHeight+1, shardID)
			if err != nil {
				return nil, NewRPCError(GetShardBlockByHeightError, err)
			}
			for _, nextShardBlock := range nextShardBlocks {
				h := shardBlock.Hash()
				if nextShardBlock.Header.PreviousBlockHash.IsEqual(h) {
					nextHashString = nextShardBlock.Hash().String()
					break
				}
			}
		}
		result.Hash = shardBlock.Hash().String()
		result.Confirmations = int64(1 + best.Header.Height - blockHeight)
		result.Height = shardBlock.Header.Height
		result.Version = shardBlock.Header.Version
		result.TxRoot = shardBlock.Header.TxRoot.String()
		result.Time = shardBlock.Header.Timestamp
		result.ShardID = shardBlock.Header.ShardID
		result.PreviousBlockHash = shardBlock.Header.PreviousBlockHash.String()
		result.NextBlockHash = nextHashString
		result.BeaconHeight = shardBlock.Header.BeaconHeight
		result.BeaconBlockHash = shardBlock.Header.BeaconHash.String()
		result.ValidationData = shardBlock.ValidationData
		result.Round = shardBlock.Header.Round
		result.CrossShardBitMap = []int{}
		result.Instruction = shardBlock.Body.Instructions
		result.CommitteeFromBlock = shardBlock.Header.CommitteeFromBlock
		if len(shardBlock.Header.CrossShardBitMap) > 0 {
			for _, shardID := range shardBlock.Header.CrossShardBitMap {
				result.CrossShardBitMap = append(result.CrossShardBitMap, int(shardID))
			}
		}
		result.Epoch = shardBlock.Header.Epoch
		result.TxHashes = []string{}
		for _, tx := range shardBlock.Body.Transactions {
			result.TxHashes = append(result.TxHashes, tx.Hash().String())
		}
	} else if verbosity == "2" {
		best := blockService.BlockChain.GetBestStateShard(byte(shardID)).BestBlock

		blockHeight := shardBlock.Header.Height
		// Get next block hash unless there are none.
		var nextHashString string
		if blockHeight < best.Header.Height {
			nextShardBlocks, err := blockService.BlockChain.GetShardBlockByHeight(blockHeight+1, shardID)
			if err != nil {
				return nil, NewRPCError(GetShardBlockByHeightError, err)
			}
			for _, nextShardBlock := range nextShardBlocks {
				h := shardBlock.Hash()
				if nextShardBlock.Header.PreviousBlockHash.IsEqual(h) {
					nextHashString = nextShardBlock.Hash().String()
					break
				}
			}
		}
		result.Hash = shardBlock.Hash().String()
		result.Confirmations = int64(1 + best.Header.Height - blockHeight)
		result.Height = shardBlock.Header.Height
		result.Version = shardBlock.Header.Version
		result.TxRoot = shardBlock.Header.TxRoot.String()
		result.Time = shardBlock.Header.Timestamp
		result.ShardID = shardBlock.Header.ShardID
		result.PreviousBlockHash = shardBlock.Header.PreviousBlockHash.String()
		result.NextBlockHash = nextHashString
		result.BeaconHeight = shardBlock.Header.BeaconHeight
		result.BeaconBlockHash = shardBlock.Header.BeaconHash.String()
		result.ValidationData = shardBlock.ValidationData
		result.Round = shardBlock.Header.Round
		result.CrossShardBitMap = []int{}
		result.Instruction = shardBlock.Body.Instructions
		instructions, err := blockchain.CreateShardInstructionsFromTransactionAndInstruction(shardBlock.Body.Transactions, blockService.BlockChain, shardBlock.Header.ShardID)
		if err == nil {
			result.Instruction = append(result.Instruction, instructions...)
		}
		if len(shardBlock.Header.CrossShardBitMap) > 0 {
			for _, shardID := range shardBlock.Header.CrossShardBitMap {
				result.CrossShardBitMap = append(result.CrossShardBitMap, int(shardID))
			}
		}
		result.Epoch = shardBlock.Header.Epoch
		result.CommitteeFromBlock = shardBlock.Header.CommitteeFromBlock
		result.Txs = make([]jsonresult.GetBlockTxResult, 0)
		for _, tx := range shardBlock.Body.Transactions {
			transactionResult := jsonresult.GetBlockTxResult{}
			transactionResult.Hash = tx.Hash().String()
			switch tx.GetType() {
			case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType:
				txN := tx.(*transaction.Tx)
				data, err := json.Marshal(txN)
				if err != nil {
					return nil, NewRPCError(JsonError, err)
				}
				transactionResult.HexData = hex.EncodeToString(data)
				transactionResult.Locktime = txN.LockTime
			}
			result.Txs = append(result.Txs, transactionResult)
		}
	}
	return &result, nil
}

func (blockService BlockService) RetrieveShardBlockByHeight(blockHeight uint64, shardId int, verbosity string) ([]*jsonresult.GetShardBlockResult, *RPCError) {
	shardBlocks, errD := blockService.BlockChain.GetShardBlockByHeight(blockHeight, byte(shardId))
	if errD != nil {
		Logger.log.Debugf("handleRetrieveBlock result: %+v, err: %+v", nil, errD)
		return nil, NewRPCError(GetShardBlockByHashError, errD)
	}
	result := []*jsonresult.GetShardBlockResult{}
	for _, shardBlock := range shardBlocks {
		res := jsonresult.GetShardBlockResult{}
		shardID := shardBlock.Header.ShardID
		if verbosity == "0" {
			data, err := json.Marshal(shardBlock)
			if err != nil {
				Logger.log.Debugf("handleRetrieveBlock result: %+v, err: %+v", nil, err)
				return nil, NewRPCError(JsonError, err)
			}
			res.Data = hex.EncodeToString(data)
		} else if verbosity == "1" {
			best := blockService.BlockChain.GetBestStateShard(shardID).BestBlock
			// Get next block hash unless there are none.
			var nextHashString string
			// if blockHeight < best.Header.GetHeight() {
			if blockHeight < best.Header.Height {
				nextHashes, err := blockService.BlockChain.GetShardBlockByHeight(blockHeight+1, shardID)
				if err != nil {
					return nil, NewRPCError(GetShardBlockByHeightError, err)
				}
				for _, nextHash := range nextHashes {
					if nextHash.Header.PreviousBlockHash == shardBlock.Header.Hash() {
						nextHashString = nextHash.Hash().String()
					}
				}
			}
			res.Hash = shardBlock.Hash().String()
			res.Confirmations = int64(1 + best.Header.Height - blockHeight)
			res.Height = shardBlock.Header.Height
			res.Version = shardBlock.Header.Version
			res.TxRoot = shardBlock.Header.TxRoot.String()
			res.Time = shardBlock.Header.Timestamp
			res.ShardID = shardBlock.Header.ShardID
			res.PreviousBlockHash = shardBlock.Header.PreviousBlockHash.String()
			res.NextBlockHash = nextHashString
			res.TxHashes = []string{}
			res.BeaconHeight = shardBlock.Header.BeaconHeight
			res.BeaconBlockHash = shardBlock.Header.BeaconHash.String()
			res.ValidationData = shardBlock.ValidationData
			res.Round = shardBlock.Header.Round
			res.CrossShardBitMap = []int{}
			res.Instruction = shardBlock.Body.Instructions
			if len(shardBlock.Header.CrossShardBitMap) > 0 {
				for _, shardID := range shardBlock.Header.CrossShardBitMap {
					res.CrossShardBitMap = append(res.CrossShardBitMap, int(shardID))
				}
			}
			res.Epoch = shardBlock.Header.Epoch

			for _, tx := range shardBlock.Body.Transactions {
				res.TxHashes = append(res.TxHashes, tx.Hash().String())
			}
		} else if verbosity == "2" {
			best := blockService.BlockChain.GetBestStateShard(shardID).BestBlock
			blockHeight := shardBlock.Header.Height
			var nextHashString string
			if blockHeight < best.Header.Height {
				nextHashes, err := blockService.BlockChain.GetShardBlockByHeight(blockHeight+1, shardID)
				if err != nil {
					Logger.log.Debugf("handleRetrieveBlock result: %+v, err: %+v", nil, err)
					return nil, NewRPCError(GetShardBlockByHeightError, err)
				}
				for _, nextHash := range nextHashes {
					if nextHash.Header.PreviousBlockHash == shardBlock.Header.Hash() {
						nextHashString = nextHash.Hash().String()
					}
				}
			}

			res.Hash = shardBlock.Hash().String()
			res.Confirmations = int64(1 + best.Header.Height - blockHeight)
			res.Height = shardBlock.Header.Height
			res.Version = shardBlock.Header.Version
			res.TxRoot = shardBlock.Header.TxRoot.String()
			res.Time = shardBlock.Header.Timestamp
			res.ShardID = shardBlock.Header.ShardID
			res.PreviousBlockHash = shardBlock.Header.PreviousBlockHash.String()
			res.NextBlockHash = nextHashString
			res.BeaconHeight = shardBlock.Header.BeaconHeight
			res.BeaconBlockHash = shardBlock.Header.BeaconHash.String()
			res.ValidationData = shardBlock.ValidationData
			res.Round = shardBlock.Header.Round
			res.CrossShardBitMap = []int{}
			res.Instruction = shardBlock.Body.Instructions
			instructions, err := blockchain.CreateShardInstructionsFromTransactionAndInstruction(shardBlock.Body.Transactions, blockService.BlockChain, shardBlock.Header.ShardID)
			if err == nil {
				res.Instruction = append(res.Instruction, instructions...)
			}
			if len(shardBlock.Header.CrossShardBitMap) > 0 {
				for _, shardID := range shardBlock.Header.CrossShardBitMap {
					res.CrossShardBitMap = append(res.CrossShardBitMap, int(shardID))
				}
			}
			res.Epoch = shardBlock.Header.Epoch
			res.Txs = make([]jsonresult.GetBlockTxResult, 0)
			for _, tx := range shardBlock.Body.Transactions {
				transactionT := jsonresult.GetBlockTxResult{}
				transactionT.Hash = tx.Hash().String()
				switch tx.GetType() {
				case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType:
					txN := tx.(*transaction.Tx)
					data, err := json.Marshal(txN)
					if err != nil {
						return nil, NewRPCError(JsonError, err)
					}
					transactionT.HexData = hex.EncodeToString(data)
					transactionT.Locktime = txN.LockTime
				}
				res.Txs = append(res.Txs, transactionT)
			}
		}
		result = append(result, &res)
	}
	return result, nil
}

func (blockService BlockService) RetrieveBeaconBlock(hashString string) (*jsonresult.GetBeaconBlockResult, *RPCError) {
	hash, errH := common.Hash{}.NewHashFromStr(hashString)
	if errH != nil {
		Logger.log.Debugf("handleRetrieveBeaconBlock result: %+v, err: %+v", nil, errH)
		return nil, NewRPCError(RPCInvalidParamsError, errH)
	}
	block, _, errD := blockService.BlockChain.GetBeaconBlockByHash(*hash)
	if errD != nil {
		Logger.log.Debugf("handleRetrieveBeaconBlock result: %+v, err: %+v", nil, errD)
		return nil, NewRPCError(GetBeaconBlockByHashError, errD)
	}
	bestBeaconBlock := blockService.BlockChain.GetBeaconBestState().BestBlock
	blockHeight := block.Header.Height
	// Get next block hash unless there are none.
	var nextHashString string
	// if blockHeight < best.Header.GetHeight() {
	if blockHeight < bestBeaconBlock.Header.Height {
		nextHashes, err := blockService.BlockChain.GetBeaconBlockByHeight(blockHeight + 1)
		if err != nil {
			Logger.log.Debugf("handleRetrieveBeaconBlock result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(GetBeaconBlockByHeightError, err)
		}
		nextHash := nextHashes[0]
		nextHashString = nextHash.Hash().String()
	}
	blockBytes, errS := json.Marshal(block)
	if errS != nil {
		return nil, NewRPCError(UnexpectedError, errS)
	}
	result := jsonresult.NewGetBlocksBeaconResult(block, uint64(len(blockBytes)), nextHashString)
	return result, nil
}

func (blockService BlockService) RetrieveBeaconBlockByHeight(blockHeight uint64) ([]*jsonresult.GetBeaconBlockResult, *RPCError) {
	var err error
	nextBeaconBlocks := []*types.BeaconBlock{}
	beaconBlocks, err := blockService.BlockChain.GetBeaconBlockByHeight(blockHeight)
	if err != nil {
		Logger.log.Debugf("handleRetrieveBeaconBlock result: %+v, err: %+v", nil, err)
		return nil, NewRPCError(GetBeaconBlockByHashError, err)
	}
	result := []*jsonresult.GetBeaconBlockResult{}
	bestBeaconBlock := blockService.BlockChain.GetBeaconBestState().BestBlock
	// Get next block hash unless there are none.
	if blockHeight < bestBeaconBlock.Header.Height {
		nextBeaconBlocks, err = blockService.BlockChain.GetBeaconBlockByHeight(blockHeight + 1)
		if err != nil {
			Logger.log.Debugf("handleRetrieveBeaconBlock result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(GetBeaconBlockByHeightError, err)
		}
	}
	for _, beaconBlock := range beaconBlocks {
		beaconBlockBytes, errS := json.Marshal(beaconBlock)
		if errS != nil {
			return nil, NewRPCError(UnexpectedError, errS)
		}
		nextHashString := ""
		for _, nextBeaconBlock := range nextBeaconBlocks {
			currentBlockHash := beaconBlock.Header.Hash()
			if nextBeaconBlock.Header.PreviousBlockHash.IsEqual(&currentBlockHash) {
				nextHashString = nextBeaconBlock.Header.Hash().String()
			}
		}
		res := jsonresult.NewGetBlocksBeaconResult(beaconBlock, uint64(len(beaconBlockBytes)), nextHashString)
		result = append(result, res)
	}
	return result, nil
}

func (blockService BlockService) GetBlocks(shardIDParam int, numBlock int) (interface{}, *RPCError) {
	resultShard := make([]jsonresult.GetShardBlockResult, 0)
	resultBeacon := make([]jsonresult.GetBeaconBlockResult, 0)
	var cacheKey []byte
	if shardIDParam != -1 {
		cacheKey = memcache.GetBlocksCachedKey(shardIDParam, numBlock)
		cacheValue, err := blockService.MemCache.Get(cacheKey)
		if err == nil && len(cacheValue) > 0 {
			err1 := json.Unmarshal(cacheValue, &resultShard)
			if err1 != nil {
				Logger.log.Error("Json Unmarshal cache of get shard blocks error", err1)
			} else {
				return resultShard, nil
			}
		}
	} else {
		cacheKey = memcache.GetBlocksCachedKey(shardIDParam, numBlock)
		cacheValue, err := blockService.MemCache.Get(cacheKey)
		if err == nil && len(cacheValue) > 0 {
			err1 := json.Unmarshal(cacheValue, &resultBeacon)
			if err1 != nil {
				Logger.log.Error("Json Unmarshal cache of get beacon blocks error", err1)
			} else {
				return resultBeacon, nil
			}
		}
	}

	if shardIDParam != -1 {
		if len(resultShard) == 0 {
			shardID := byte(shardIDParam)
			clonedShardBestState, err := blockService.BlockChain.GetClonedAShardBestState(shardID)
			if err != nil {
				return nil, NewRPCError(GetClonedShardBestStateError, err)
			}
			bestBlock := clonedShardBestState.BestBlock
			previousHash := bestBlock.Hash()
			for numBlock > 0 {
				numBlock--
				block, size, errD := blockService.BlockChain.GetShardBlockByHash(*previousHash)
				if errD != nil {
					Logger.log.Debugf("handleGetBlocks resultShard: %+v, err: %+v", nil, errD)
					return nil, NewRPCError(GetShardBlockByHashError, errD)
				}
				blockResult := jsonresult.NewGetBlockResult(block, size, common.EmptyString)
				resultShard = append(resultShard, *blockResult)
				previousHash = &block.Header.PreviousBlockHash
				if previousHash.String() == (common.Hash{}).String() {
					break
				}
			}
			Logger.log.Debugf("handleGetBlocks resultShard: %+v", resultShard)
			if len(resultShard) > 0 {
				cacheValue, err := json.Marshal(resultShard)
				if err == nil {
					err1 := blockService.MemCache.PutExpired(cacheKey, cacheValue, 10000)
					if err1 != nil {
						Logger.log.Error("Cache data of shard best state error", err1)
					}
				}
			}
		}
		return resultShard, nil
	} else {
		if len(resultBeacon) == 0 {
			clonedBeaconBestState, err := blockService.BlockChain.GetClonedBeaconBestState()
			if err != nil {
				return nil, NewRPCError(GetClonedBeaconBestStateError, err)
			}
			bestBlock := clonedBeaconBestState.BestBlock
			previousHash := bestBlock.Hash()
			for numBlock > 0 {
				numBlock--
				block, size, errD := blockService.BlockChain.GetBeaconBlockByHash(*previousHash)
				if errD != nil {
					return nil, NewRPCError(GetBeaconBlockByHashError, errD)
				}
				blockResult := jsonresult.NewGetBlocksBeaconResult(block, size, common.EmptyString)
				resultBeacon = append(resultBeacon, *blockResult)
				previousHash = &block.Header.PreviousBlockHash
				if previousHash.String() == (common.Hash{}).String() {
					break
				}
			}
			Logger.log.Debugf("handleGetBlocks resultShard: %+v", resultBeacon)
			if len(resultBeacon) > 0 {
				cacheValue, err := json.Marshal(resultBeacon)
				if err == nil {
					err1 := blockService.MemCache.PutExpired(cacheKey, cacheValue, 10000)
					if err1 != nil {
						Logger.log.Error("Cache data of shard best state error", err1)
					}
				}
			}
		}
		return resultBeacon, nil
	}
}

func (blockService BlockService) IsBeaconBestStateNil() bool {
	return blockService.BlockChain.GetBeaconBestState() == nil
}

func (blockService BlockService) IsShardBestStateNil() bool {
	return blockService.BlockChain.GetBestStateShard(0) == nil
}

func (blockService BlockService) GetValidStakers(publicKeys []string) ([]string, *RPCError) {
	beaconBestState, err := blockService.GetBeaconBestState()
	if err != nil {
		return nil, NewRPCError(GetClonedBeaconBestStateError, err)
	}

	validPublicKeys := beaconBestState.GetValidStakers(publicKeys)

	return validPublicKeys, nil
}

func (blockService BlockService) CheckHashValue(hashStr string) (isTransaction bool, isShardBlock bool, isBeaconBlock bool, err error) {
	isTransaction = false
	isShardBlock = false
	isBeaconBlock = false

	hash, err2 := common.Hash{}.NewHashFromStr(hashStr)
	if err2 != nil {
		err = errors.New("expected hash string value")
		return
	}
	_, _, err = blockService.BlockChain.GetShardBlockByHash(*hash)
	if err == nil {
		isShardBlock = true
		return
	} else {
		_, _, err = blockService.BlockChain.GetBeaconBlockByHash(*hash)
		if err == nil {
			isBeaconBlock = true
			return
		} else {
			_, _, _, _, _, err = blockService.BlockChain.GetTransactionByHash(*hash)
			if err == nil {
				isTransaction = true
			} else {
				err = nil
			}
		}
	}

	return
}

func (blockService BlockService) GetActiveShards() int {
	return blockService.BlockChain.GetBeaconBestState().ActiveShards
}

func (blockService BlockService) ListPrivacyCustomToken() (map[common.Hash]*statedb.TokenState, error) {
	tokenStates, err := blockService.BlockChain.ListAllPrivacyCustomTokenAndPRV()
	if err != nil {
		return tokenStates, err
	}
	delete(tokenStates, common.PRVCoinID)
	return tokenStates, err
}

func (blockService BlockService) ListPrivacyCustomTokenWithTxs() (map[common.Hash]*statedb.TokenState, error) {
	tokenStates, err := blockService.BlockChain.ListAllPrivacyCustomTokenAndPRVWithTxs()
	if err != nil {
		return tokenStates, err
	}
	delete(tokenStates, common.PRVCoinID)
	return tokenStates, err
}

func (blockService BlockService) ListPrivacyCustomTokenWithPRVByShardID(shardID byte) (map[common.Hash]*statedb.TokenState, error) {
	return blockService.BlockChain.ListPrivacyCustomTokenAndPRVByShardID(shardID)
}

// TODO: 0xmerman update to DBV2 later
//func (blockService BlockService) ListPrivacyCustomTokenCached() (map[common.Hash]transaction.TxCustomTokenPrivacy, map[common.Hash]blockchain.CrossShardTokenPrivacyMetaData, error) {
//	listTxInitPrivacyToken := make(map[common.Hash]transaction.TxCustomTokenPrivacy)
//	listTxInitPrivacyTokenCrossShard := make(map[common.Hash]blockchain.CrossShardTokenPrivacyMetaData)
//
//	cachedKeyPrivacyToken := memcache.GetListPrivacyTokenCachedKey()
//	cachedValuePrivacyToken, err := blockService.MemCache.Get(cachedKeyPrivacyToken)
//	if err == nil && len(cachedValuePrivacyToken) > 0 {
//		err1 := json.Unmarshal(cachedValuePrivacyToken, &listTxInitPrivacyToken)
//		if err1 != nil {
//			Logger.log.Error("Json Unmarshal cachedKeyPrivacyToken err", err1)
//		}
//	}
//
//	cachedKeyPrivacyTokenCrossShard := memcache.GetListPrivacyTokenCrossShardCachedKey()
//	cachedValuePrivacyTokenCrossShard, err := blockService.MemCache.Get(cachedKeyPrivacyTokenCrossShard)
//	if err == nil && len(cachedValuePrivacyToken) > 0 {
//		err1 := json.Unmarshal(cachedValuePrivacyTokenCrossShard, &listTxInitPrivacyTokenCrossShard)
//		if err1 != nil {
//			Logger.log.Error("Json Unmarshal cachedKeyPrivacyToken err", err1)
//		}
//	}
//
//	if len(listTxInitPrivacyToken) == 0 || len(listTxInitPrivacyTokenCrossShard) == 0 {
//		listTxInitPrivacyToken, listTxInitPrivacyTokenCrossShard, err = blockService.ListPrivacyCustomToken()
//
//		for k, v := range listTxInitPrivacyToken {
//			temp := v
//			temp.Tx = transaction.Tx{Info: v.Info}
//			temp.TxPrivacyTokenData.TxNormal = transaction.Tx{Info: v.TxPrivacyTokenData.TxNormal.Info}
//			listTxInitPrivacyToken[k] = temp
//		}
//		cachedValuePrivacyToken, err = json.Marshal(listTxInitPrivacyToken)
//		if err == nil {
//			err1 := blockService.MemCache.PutExpired(cachedKeyPrivacyToken, cachedValuePrivacyToken, 60*1000)
//			if err1 != nil {
//				Logger.log.Error("Cached error cachedValuePrivacyToken", err1)
//			}
//		}
//
//		cachedValuePrivacyTokenCrossShard, err = json.Marshal(listTxInitPrivacyTokenCrossShard)
//		if err == nil {
//			err1 := blockService.MemCache.PutExpired(cachedKeyPrivacyTokenCrossShard, cachedValuePrivacyTokenCrossShard, 60*1000)
//			if err1 != nil {
//				Logger.log.Error("Cached error cachedValuePrivacyTokenCrossShard", err1)
//			}
//		}
//	}
//	return listTxInitPrivacyToken, listTxInitPrivacyTokenCrossShard, err
//}

func (blockService BlockService) GetAllCoinIDWithPRV(shardID byte) ([]common.Hash, error) {
	tokenIDs, err := blockService.BlockChain.ListPrivacyTokenAndBridgeTokenAndPRVByShardID(shardID)
	if err != nil {
		return []common.Hash{}, err
	}
	return tokenIDs, nil
}

func (blockService BlockService) GetMinerRewardFromMiningKey(incPublicKey []byte) (map[string]uint64, error) {
	shardID := common.GetShardIDFromLastByte(incPublicKey[len(incPublicKey)-1])
	allCoinIDs, err := blockService.GetAllCoinIDWithPRV(shardID)
	if err != nil {
		return nil, err
	}
	rewardAmountResult := make(map[string]uint64)
	rewardStateDB := blockService.BlockChain.GetBestStateShard(shardID).GetShardRewardStateDB()
	tempIncPublicKey := base58.Base58Check{}.Encode(incPublicKey, common.Base58Version)
	for _, coinID := range allCoinIDs {
		amount, err := statedb.GetCommitteeReward(rewardStateDB, tempIncPublicKey, coinID)
		if err != nil {
			return nil, err
		}
		if coinID == common.PRVCoinID {
			rewardAmountResult["PRV"] = amount
		} else {
			if amount > 0 {
				rewardAmountResult[coinID.String()] = amount
			}
		}
	}
	return rewardAmountResult, nil
}

func (blockService BlockService) ListRewardAmount() (map[string]map[common.Hash]uint64, error) {
	m := make(map[string]map[common.Hash]uint64)
	beaconBestState := blockService.BlockChain.GetBeaconBestState()
	for i := 0; i < beaconBestState.ActiveShards; i++ {
		shardID := byte(i)
		committeeRewardStateDB := blockService.BlockChain.GetBestStateShard(shardID).GetShardRewardStateDB()
		committeeReward := statedb.ListCommitteeReward(committeeRewardStateDB)
		for k, v := range committeeReward {
			m[k] = v
		}
	}
	return m, nil
}

func (blockService BlockService) GetRewardAmount(paymentAddress string) (map[string]uint64, error) {
	rewardAmountResult := make(map[string]uint64)
	keySet, _, err := GetKeySetFromPaymentAddressParam(paymentAddress)
	if err != nil {
		return nil, err
	}
	publicKey := keySet.PaymentAddress.Pk
	if publicKey == nil {
		return rewardAmountResult, nil
	}
	shardID := common.GetShardIDFromLastByte(publicKey[len(publicKey)-1])
	allCoinIDs, err := blockService.BlockChain.ListPrivacyTokenAndBridgeTokenAndPRVByShardID(shardID)
	if err != nil {
		return nil, err
	}
	for _, coinID := range allCoinIDs {
		committeeRewardStateDB := blockService.BlockChain.GetBestStateShard(shardID).GetShardRewardStateDB()
		tempPK := base58.Base58Check{}.Encode(publicKey, common.Base58Version)
		amount, err := statedb.GetCommitteeReward(committeeRewardStateDB, tempPK, coinID)
		if err != nil {
			return nil, err
		}
		if coinID == common.PRVCoinID {
			rewardAmountResult["PRV"] = amount
		} else {
			if amount > 0 {
				rewardAmountResult[coinID.String()] = amount
			}
		}
	}
	return rewardAmountResult, nil
}

func (blockService BlockService) GetRewardAmountByPublicKey(publicKey string) (map[string]uint64, error) {
	rewardAmountResult := make(map[string]uint64)
	tempPK, _, err := base58.Base58Check{}.Decode(publicKey)
	if err != nil {
		return nil, err
	}
	shardID := common.GetShardIDFromLastByte(publicKey[len(tempPK)-1])
	allCoinIDs, err := blockService.BlockChain.ListPrivacyTokenAndBridgeTokenAndPRVByShardID(shardID)
	if err != nil {
		return nil, err
	}
	for _, coinID := range allCoinIDs {
		committeeRewardStateDB := blockService.BlockChain.GetBestStateShard(shardID).GetShardRewardStateDB()
		amount, err := statedb.GetCommitteeReward(committeeRewardStateDB, publicKey, coinID)
		if err != nil {
			return nil, err
		}
		if coinID == common.PRVCoinID {
			rewardAmountResult["PRV"] = amount
		} else {
			if amount > 0 {
				rewardAmountResult[coinID.String()] = amount
			}
		}
	}
	return rewardAmountResult, nil
}

func (blockService BlockService) CanPubkeyStake(publicKey string) (bool, error) {
	canStake := true
	validStakers, err := blockService.GetValidStakers([]string{publicKey})
	if err != nil {
		return false, err
	}
	if len(validStakers) == 0 {
		canStake = false
	} else {
		// get pool candidate
		poolCandidate := blockService.TxMemPool.GetClonedPoolCandidate()
		if common.IndexOfStrInHashMap(publicKey, poolCandidate) > 0 {
			canStake = false
		}
	}
	return canStake, nil
}

func (blockService BlockService) GetBlockHashByHeightV2(shardID int, height uint64) ([]common.Hash, error) {
	var hash *common.Hash
	var err error
	var beaconBlocks []*types.BeaconBlock
	var shardBlocks map[common.Hash]*types.ShardBlock
	res := []common.Hash{}
	isGetBeacon := shardID == -1
	if isGetBeacon {
		beaconBlocks, err = blockService.BlockChain.GetBeaconBlockByHeight(height)
	} else {
		shardBlocks, err = blockService.BlockChain.GetShardBlockByHeight(height, byte(shardID))
	}
	if err != nil {
		Logger.log.Debugf("handleGetBlockHash result: %+v", nil)
		return res, err
	}

	if isGetBeacon {
		for _, beaconBlock := range beaconBlocks {
			hash = beaconBlock.Hash()
			res = append(res, *hash)
		}
	} else {
		for _, shardBlock := range shardBlocks {
			hash = shardBlock.Hash()
			res = append(res, *hash)
		}
	}
	return res, nil
}

func (blockService BlockService) GetShardBlockHeader(getBy string, blockParam string, shardID float64) ([]*types.ShardHeader, int, []string, *RPCError) {
	switch getBy {
	case "blockhash":
		hash := common.Hash{}
		err := hash.Decode(&hash, blockParam)
		Logger.log.Infof("%+v", hash)
		if err != nil {
			Logger.log.Debugf("handleGetBlockHeader result: %+v", nil)
			return nil, 0, []string{}, NewRPCError(RPCInvalidParamsError, errors.New("invalid blockhash format"))
		}
		shardBlock, _, err := blockService.BlockChain.GetShardBlockByHash(hash)
		if err != nil {
			Logger.log.Debugf("handleGetBlockHeader result: %+v", nil)
			return nil, 0, []string{}, NewRPCError(GetShardBlockByHashError, errors.New("blockParam not exist"))
		}
		blockNumber := int(shardBlock.Header.Height) + 1
		return []*types.ShardHeader{&shardBlock.Header}, blockNumber, []string{hash.String()}, nil
	case "blocknum":
		blockNumber, err := strconv.Atoi(blockParam)
		if err != nil {
			Logger.log.Debugf("handleGetBlockHeader result: %+v", nil)
			return nil, 0, []string{}, NewRPCError(RPCInvalidParamsError, errors.New("invalid blocknum format"))
		}
		shard, err := blockService.GetShardBestStateByShardID(byte(shardID))
		if err != nil {
			return nil, 0, []string{}, NewRPCError(GetClonedShardBestStateError, err)
		}
		if uint64(blockNumber-1) > shard.BestBlock.Header.Height || blockNumber <= 0 {
			Logger.log.Debugf("handleGetBlockHeader result: %+v", nil)
			return nil, 0, []string{}, NewRPCError(GetShardBestBlockError, errors.New("Block not exist"))
		}
		shardBlocks, _ := blockService.BlockChain.GetShardBlockByHeight(uint64(blockNumber-1), uint8(shardID))
		shardHeaders := []*types.ShardHeader{}
		hashes := []string{}
		for _, shardBlock := range shardBlocks {
			shardHeaders = append(shardHeaders, &shardBlock.Header)
			hashes = append(hashes, shardBlock.Hash().String())
		}
		return shardHeaders, blockNumber, hashes, nil
	default:
		Logger.log.Debugf("handleGetBlockHeader result: %+v", nil)
		return nil, 0, []string{}, NewRPCError(RPCInvalidParamsError, errors.New("wrong request format"))
	}
}

func (blockService BlockService) GetBurningAddress(beaconHeight uint64) string {
	return blockService.BlockChain.GetBurningAddress(beaconHeight)
}

//============================= Bridge ===============================
func (blockService BlockService) GetBridgeReqWithStatus(txID string) (byte, error) {
	txIDHash, err := common.Hash{}.NewHashFromStr(txID)
	if err != nil {
		return byte(0), err
	}
	status := byte(common.BridgeRequestNotFoundStatus)
	for _, i := range blockService.BlockChain.GetShardIDs() {
		shardID := byte(i)
		bridgeStateDB := blockService.BlockChain.GetBestStateShard(shardID).GetCopiedFeatureStateDB()
		newStatus, err := statedb.GetBridgeReqWithStatus(bridgeStateDB, *txIDHash)
		if err != nil {
			return status, err
		}
		if newStatus == byte(common.BridgeRequestProcessingStatus) {
			status = newStatus
		}
		if newStatus == byte(common.BridgeRequestAcceptedStatus) {
			return newStatus, nil
		}
	}
	if status == common.BridgeRequestNotFoundStatus || status == common.BridgeRequestProcessingStatus {
		bridgeStateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
		bStatus, err := statedb.GetBridgeReqWithStatus(bridgeStateDB, *txIDHash)
		if err != nil {
			return bStatus, err
		}
		if bStatus == common.BridgeRequestRejectedStatus {
			return bStatus, nil
		}
	}
	return status, nil
}

func (blockService BlockService) GetAllBridgeTokens() ([]*rawdbv2.BridgeTokenInfo, error) {
	_, bridgeTokenInfos, err := blockService.BlockChain.GetAllBridgeTokens()
	return bridgeTokenInfos, err
}

func (blockService BlockService) CheckETHHashIssued(data map[string]interface{}) (bool, error) {
	blockHashParam, ok := data["BlockHash"].(string)
	if !ok {
		return false, errors.New("Block hash param is invalid")
	}
	blockHash := rCommon.HexToHash(blockHashParam)

	txIdxParam, ok := data["TxIndex"].(float64)
	if !ok {
		return false, errors.New("Tx index param is invalid")
	}
	txIdx := uint(txIdxParam)
	uniqETHTx := append(blockHash[:], []byte(strconv.Itoa(int(txIdx)))...)
	bridgeStateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	issued, err := statedb.IsETHTxHashIssued(bridgeStateDB, uniqETHTx)
	return issued, err
}

func (blockService BlockService) GetBurningConfirm(txID common.Hash) (uint64, bool, error) {
	// Get from beacon first
	burningConfirmStateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	if res, err := statedb.GetBurningConfirm(burningConfirmStateDB, txID); err == nil {
		return res, true, nil
	}

	// Get from shard
	for i := 0; i < blockService.BlockChain.GetBeaconBestState().ActiveShards; i++ {
		shardID := byte(i)
		burningConfirmStateDB := blockService.BlockChain.GetBestStateShard(shardID).GetCopiedFeatureStateDB()
		res, err := statedb.GetBurningConfirm(burningConfirmStateDB, txID)
		if err == nil {
			return res, false, nil
		}
	}
	return 0, false, fmt.Errorf("Get Burning Confirm of TxID %+v not found", txID)
}

func (blockService BlockService) GetPDEContributionStatus(pdePrefix []byte, pdeSuffix []byte) (*metadata.PDEContributionStatus, error) {
	pdexStateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	pdeStatusContentBytes, err := statedb.GetPDEContributionStatus(pdexStateDB, pdePrefix, pdeSuffix)
	if err != nil {
		return nil, err
	}
	if len(pdeStatusContentBytes) == 0 {
		return nil, nil
	}
	var contributionStatus metadata.PDEContributionStatus
	err = json.Unmarshal(pdeStatusContentBytes, &contributionStatus)
	if err != nil {
		return nil, err
	}
	return &contributionStatus, nil
}

func (blockService BlockService) GetPDEStatus(pdePrefix []byte, pdeSuffix []byte) (byte, error) {
	pdexStateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	return statedb.GetPDEStatus(pdexStateDB, pdePrefix, pdeSuffix)
}

////============================= Slash ===============================
//func (blockService BlockService) GetProducersBlackList(beaconHeight uint64) (map[string]uint8, error) {
//	slashRootHash, err := blockService.BlockChain.GetBeaconSlashRootHash(blockService.BlockChain.GetBeaconBestState().GetBeaconConsensusStateDB(), beaconHeight)
//	if err != nil {
//		return nil, fmt.Errorf("Beacon Slash Root Hash of Height %+v not found ,error %+v", beaconHeight, err)
//	}
//	slashStateDB, err := statedb.NewWithPrefixTrie(slashRootHash, statedb.NewDatabaseAccessWarper(blockService.BlockChain.GetBeaconChainDatabase()))
//	return statedb.GetProducersBlackList(slashStateDB, beaconHeight), nil
//}

//============================= Portal ===============================
func (blockService BlockService) GetCustodianDepositStatus(depositTxID string) (*metadata.PortalCustodianDepositStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetCustodianDepositStatus(stateDB, depositTxID)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalCustodianDepositStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetPortalReqPTokenStatus(reqTxID string) (*metadata.PortalRequestPTokensStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetRequestPTokenStatus(stateDB, reqTxID)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalRequestPTokensStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetPortalReqUnlockCollateralStatus(reqTxID string) (*metadata.PortalRequestUnlockCollateralStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalRequestUnlockCollateralStatus(stateDB, reqTxID)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalRequestUnlockCollateralStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetPortalRedeemReqStatus(redeemID string) (*metadata.PortalRedeemRequestStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalRedeemRequestStatus(stateDB, redeemID)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalRedeemRequestStatus
	if len(data) > 0 {
		err = json.Unmarshal(data, &status)
		if err != nil {
			return nil, err
		}
		return &status, nil
	}

	return nil, nil
}

func (blockService BlockService) GetPortalRedeemReqByTxIDStatus(txID string) (*metadata.PortalRedeemRequestStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalRedeemRequestByTxIDStatus(stateDB, txID)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalRedeemRequestStatus
	if len(data) > 0 {
		err = json.Unmarshal(data, &status)
		if err != nil {
			return nil, err
		}
		return &status, nil
	}

	return nil, nil
}

func (blockService BlockService) GetPortalLiquidationCustodianStatus(redeemID string, custodianIncAddress string) (*metadata.PortalLiquidateCustodianStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalLiquidationCustodianRunAwayStatus(stateDB, redeemID, custodianIncAddress)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalLiquidateCustodianStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetPortalRequestWithdrawRewardStatus(reqTxID string) (*metadata.PortalRequestWithdrawRewardStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalRequestWithdrawRewardStatus(stateDB, reqTxID)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalRequestWithdrawRewardStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetReqMatchingRedeemByTxIDStatus(reqTxID string) (*metadata.PortalReqMatchingRedeemStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalReqMatchingRedeemByTxIDStatus(stateDB, reqTxID)
	if err != nil {
		return nil, err
	}

	var status metadata.PortalReqMatchingRedeemStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetCustodianTopupStatus(txID string) (*metadata.LiquidationCustodianDepositStatusV2, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalStateStatusMultiple(
		stateDB,
		statedb.PortalLiquidationCustodianDepositStatusPrefix(),
		[]byte(txID))
	if err != nil {
		return nil, err
	}

	var status metadata.LiquidationCustodianDepositStatusV2
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetCustodianTopupWaitingPortingStatus(txID string) (*metadata.PortalTopUpWaitingPortingRequestStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetPortalStateStatusMultiple(
		stateDB,
		statedb.PortalTopUpWaitingPortingStatusPrefix(),
		[]byte(txID))
	if err != nil {
		return nil, err
	}

	var status metadata.PortalTopUpWaitingPortingRequestStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

func (blockService BlockService) GetAmountTopUpWaitingPorting(custodianAddr string) (map[string]uint64, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	currentPortalState, err := blockchain.InitCurrentPortalStateFromDB(stateDB)
	if err != nil {
		return nil, err
	}

	custodianKey := statedb.GenerateCustodianStateObjectKey(custodianAddr).String()
	custodianState, ok := currentPortalState.CustodianPoolState[custodianKey]
	if !ok || custodianState == nil {
		return nil, fmt.Errorf("Custodian address %v not found", custodianAddr)
	}

	portalParam := blockService.BlockChain.GetPortalParams(blockService.BlockChain.GetBeaconBestState().BeaconHeight)
	result, err := blockchain.CalAmountTopUpWaitingPortings(currentPortalState, custodianState, portalParam)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (blockService BlockService) GetRedeemReqFromLiquidationPoolByTxIDStatus(txID string) (*metadata.RedeemLiquidateExchangeRatesStatus, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetRedeemRequestFromLiquidationPoolByTxIDStatus(stateDB, txID)
	if err != nil {
		return nil, err
	}

	var status metadata.RedeemLiquidateExchangeRatesStatus
	if len(data) > 0 {
		err = json.Unmarshal(data, &status)
		if err != nil {
			return nil, err
		}
		return &status, nil
	}

	return nil, nil
}

//============================= Reward Feature ===============================
func (blockService BlockService) GetRewardFeatureByFeatureName(featureName string, epoch uint64) (map[string]uint64, error) {
	stateDB := blockService.BlockChain.GetBeaconBestState().GetBeaconFeatureStateDB()
	data, err := statedb.GetRewardFeatureStateByFeatureName(stateDB, featureName, epoch)
	if err != nil {
		return nil, err
	}

	return data.GetTotalRewards(), nil
}
