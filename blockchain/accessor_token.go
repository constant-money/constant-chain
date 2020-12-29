package blockchain

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

func (blockchain *BlockChain) ListAllPrivacyCustomTokenAndPRV() (map[common.Hash]*statedb.TokenState, error) {
	tokenStates := make(map[common.Hash]*statedb.TokenState)
	for i := 0; i < blockchain.GetBeaconBestState().ActiveShards; i++ {
		shardID := byte(i)
		m, err := blockchain.ListPrivacyCustomTokenAndPRVByShardID(shardID)
		if err != nil {
			return nil, err
		}
		for newK, newV := range m {
			if v, ok := tokenStates[newK]; !ok {
				tokenStates[newK] = newV
			} else {
				if v.PropertyName() == "" && newV.PropertyName() != "" {
					v.SetPropertyName(newV.PropertyName())
				}
				if v.PropertySymbol() == "" && newV.PropertySymbol() != "" {
					v.SetPropertySymbol(newV.PropertySymbol())
				}
				v.AddTxs(newV.Txs())
			}
		}
	}
	return tokenStates, nil
}

func (blockchain *BlockChain) ListAllPrivacyCustomTokenAndPRVWithTxs() (map[common.Hash]*statedb.TokenState, error) {
	tokenStates := make(map[common.Hash]*statedb.TokenState)
	for i := 0; i < blockchain.GetBeaconBestState().ActiveShards; i++ {
		shardID := byte(i)
		m, err := blockchain.ListPrivacyCustomTokenAndPRVByShardIDWithTxs(shardID)
		if err != nil {
			return nil, err
		}
		for newK, newV := range m {
			if v, ok := tokenStates[newK]; !ok {
				tokenStates[newK] = newV
			} else {
				if v.PropertyName() == "" && newV.PropertyName() != "" {
					v.SetPropertyName(newV.PropertyName())
				}
				if v.PropertySymbol() == "" && newV.PropertySymbol() != "" {
					v.SetPropertySymbol(newV.PropertySymbol())
				}
				v.AddTxs(newV.Txs())
			}
		}
	}
	return tokenStates, nil
}

// ListCustomToken - return all custom token which existed in network
func (blockchain *BlockChain) ListPrivacyCustomTokenAndPRVByShardID(shardID byte) (map[common.Hash]*statedb.TokenState, error) {
	tokenStates := statedb.ListPrivacyToken(blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB())
	return tokenStates, nil
}

// ListCustomToken - return all custom token which existed in network
func (blockchain *BlockChain) ListPrivacyCustomTokenAndPRVByShardIDWithTxs(shardID byte) (map[common.Hash]*statedb.TokenState, error) {
	tokenStates := statedb.ListPrivacyTokenWithTxs(blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB())
	return tokenStates, nil
}

func (blockchain *BlockChain) ListPrivacyTokenAndBridgeTokenAndPRVByShardID(shardID byte) ([]common.Hash, error) {
	tokenIDs := []common.Hash{}
	tokenStates, err := blockchain.ListPrivacyCustomTokenAndPRVByShardID(shardID)
	if err != nil {
		return nil, err
	}
	for k, _ := range tokenStates {
		tokenIDs = append(tokenIDs, k)
	}
	brigdeTokenIDs, _, err := blockchain.GetAllBridgeTokens()
	if err != nil {
		return nil, err
	}
	for _, bridgeTokenID := range brigdeTokenIDs {
		if _, found := tokenStates[bridgeTokenID]; !found {
			tokenIDs = append(tokenIDs, bridgeTokenID)
		}
	}
	return tokenIDs, nil
}

// Check Privacy Custom token ID is existed
func (blockchain *BlockChain) PrivacyCustomTokenIDExistedV2(tokenID *common.Hash, shardID byte) bool {
	return statedb.PrivacyTokenIDExisted(blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB(), *tokenID)
}

// Check Privacy Custom token ID is existed
func (blockchain *BlockChain) GetPrivacyTokenState(tokenID common.Hash, shardID byte) (*statedb.TokenState, bool, error) {
	return statedb.GetPrivacyTokenState(blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB(), tokenID)
}

func (blockchain *BlockChain) GetAllBridgeTokens() ([]common.Hash, []*rawdbv2.BridgeTokenInfo, error) {
	bridgeTokenIDs := []common.Hash{}
	allBridgeTokens := []*rawdbv2.BridgeTokenInfo{}
	bridgeStateDB := blockchain.GetBeaconBestState().GetBeaconFeatureStateDB()
	allBridgeTokensBytes, err := statedb.GetAllBridgeTokens(bridgeStateDB)
	if err != nil {
		return bridgeTokenIDs, allBridgeTokens, err
	}
	err = json.Unmarshal(allBridgeTokensBytes, &allBridgeTokens)
	if err != nil {
		return bridgeTokenIDs, allBridgeTokens, err
	}
	for _, bridgeTokenInfo := range allBridgeTokens {
		bridgeTokenIDs = append(bridgeTokenIDs, *bridgeTokenInfo.TokenID)
	}
	return bridgeTokenIDs, allBridgeTokens, nil
}

func (blockchain *BlockChain) PrivacyTokenIDExistedInAllShards(curView *BeaconBestState, tokenID common.Hash) (bool, error) {
	for shardID, shardHash := range curView.BestShardHash {
		db := blockchain.GetShardChainDatabase(shardID)
		shardRootHash, err := GetShardRootsHashByBlockHash(db, shardID, shardHash)
		if err != nil {
			return false, err
		}
		stateDB, err := statedb.NewWithPrefixTrie(shardRootHash.TransactionStateDBRootHash,
			statedb.NewDatabaseAccessWarper(db))
		if err != nil {
			return false, err
		}
		isExist := statedb.PrivacyTokenIDExisted(stateDB, tokenID)
		if isExist {
			return true, nil
		}
	}
	return false, nil
}
