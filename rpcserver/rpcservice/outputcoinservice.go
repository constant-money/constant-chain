package rpcservice

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/wallet"
)

type CoinService struct {
	BlockChain *blockchain.BlockChain
}

func (coinService CoinService) ListDecryptedOutputCoinsByKeySet(keySet *incognitokey.KeySet, shardID byte, shardHeight uint64) ([]coin.PlainCoin, uint64, error) {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, 0, err
	}
	plainCoins, coins, fh, err := coinService.BlockChain.GetListDecryptedOutputCoinsByKeyset(keySet, shardID, prvCoinID, shardHeight)
	if err != nil{
		return nil, 0, err
	}

	if len(coins) != 0{
		return nil, 0, errors.New("need private key to proceed")
	}

	return plainCoins, fh, nil
}

func (coinService CoinService) ListUnspentOutputCoinsByKey(listKeyParams []interface{}, tokenID *common.Hash) (*jsonresult.ListOutputCoins, *RPCError) {
	result := &jsonresult.ListOutputCoins{
		Outputs: make(map[string][]jsonresult.OutCoin),
	}
	for _, keyParam := range listKeyParams {
		keys, ok := keyParam.(map[string]interface{})
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Params %+v", keyParam))
		}
		// get keyset only contain private key by deserializing
		privateKeyStr, ok := keys["PrivateKey"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("private key is invalid"))
		}
		keyWallet, err := wallet.Base58CheckDeserialize(privateKeyStr)
		if err != nil || keyWallet.KeySet.PrivateKey == nil {
			Logger.log.Error("Check Deserialize err", err)
			return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Private key is invalid, error %+v", err))
		}
		keySetTmp, shardID, err := GetKeySetFromPrivateKey(keyWallet.KeySet.PrivateKey)
		if err != nil {
			return nil, NewRPCError(ListUnspentOutputCoinsByKeyError, err)
		}
		keyWallet.KeySet = *keySetTmp

		// get shard height
		shardHeightTemp, ok := keys["StartHeight"].(float64)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid height param"))
		}
		shardHeight := uint64(shardHeightTemp)

		outCoins, fh, err := coinService.ListDecryptedOutputCoinsByKeySet(&keyWallet.KeySet, shardID, shardHeight)
		if err != nil {
			return nil, NewRPCError(ListUnspentOutputCoinsByKeyError, err)
		}
		result.ToHeight = shardHeight
		result.FromHeight = fh
		item := make([]jsonresult.OutCoin, 0)
		for _, outCoin := range outCoins {
			if outCoin.GetValue() == 0{
				continue
			}
			tmp := jsonresult.NewOutCoin(outCoin)
			db := coinService.BlockChain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()

			if outCoin.GetVersion() == 2{
				tmpCoin, ok := outCoin.(*coin.CoinV2)
				if !ok{
					continue
				}

				publicKeyBytes := tmpCoin.GetPublicKey().ToBytesS()
				idx, err := statedb.GetOTACoinIndex(db, common.PRVCoinID, publicKeyBytes)
				if err != nil{
					return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
				}

				tmp.Index = base58.Base58Check{}.Encode(idx.Bytes(), common.ZeroByte)
				if tmpCoin.GetSharedRandom() != nil{
					tmp.SharedRandom = base58.Base58Check{}.Encode(tmpCoin.GetSharedRandom().ToBytesS(), common.ZeroByte)
				}
				if tmpCoin.GetTxRandom() != nil{
					tmp.TxRandom = base58.Base58Check{}.Encode(tmpCoin.GetTxRandom().Bytes(), common.ZeroByte)
				}
			}

			item = append(item, tmp)

		}
		result.Outputs[privateKeyStr] = item
	}
	return result, nil
}

func (coinService CoinService) ListOutputCoinsByKey(listKeyParams []interface{}, tokenID common.Hash) (*jsonresult.ListOutputCoins, *RPCError) {
	result := &jsonresult.ListOutputCoins{
		Outputs: make(map[string][]jsonresult.OutCoin),
	}
	for _, keyParam := range listKeyParams {
		keys, ok := keyParam.(map[string]interface{})
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid params: %+v", keyParam))
		}
		// get OTA secretKey deserializing (compulsory for V2, optional for V1)
		var otaKey *wallet.KeyWallet
		var err error
		otaKeyStr, ok := keys["OTASecretKey"].(string)
		if !ok || otaKeyStr == "" {
			Logger.log.Info("otaKey is optional")
		} else {
			otaKey, err = wallet.Base58CheckDeserialize(otaKeyStr)
			if err != nil {
				Logger.log.Debugf("otaKey is invalid: err: %+v", err)
				return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
			}
		}
		// get keyset only contain read only key by deserializing (optional)
		var readonlyKey *wallet.KeyWallet
		readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
		if !ok || readonlyKeyStr == "" {
			Logger.log.Info("Read onlyKey is optional")
		} else {
			readonlyKey, err = wallet.Base58CheckDeserialize(readonlyKeyStr)
			if err != nil {
				Logger.log.Debugf("Read onlyKey is invalid: err: %+v", err)
				return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
			}
		}
		// get keyset only contain public key by deserializing (required)
		paymentAddressStr, ok := keys["PaymentAddress"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid payment address"))
		}
		paymentAddressKey, err := wallet.Base58CheckDeserialize(paymentAddressStr)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
		}

		// Get start height
		startHeightTemp, ok := keys["StartHeight"].(float64)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid start height"))
		}
		startHeight := uint64(startHeightTemp)

		// create a key set
		keySet := incognitokey.KeySet{
			PaymentAddress: paymentAddressKey.KeySet.PaymentAddress,
		}
		// readonly key is optional
		if readonlyKey != nil && len(readonlyKey.KeySet.ReadonlyKey.Rk) > 0 {
			keySet.ReadonlyKey = readonlyKey.KeySet.ReadonlyKey
		}
		if otaKey != nil && otaKey.KeySet.OTAKey.GetOTASecretKey()!= nil {
			keySet.OTAKey = otaKey.KeySet.OTAKey
		}

		lastByte := keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		plainOutputCoins, outputCoins, fh, err := coinService.BlockChain.GetListDecryptedOutputCoinsByKeyset(&keySet, shardIDSender, &tokenID, startHeight)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
		}
		result.ToHeight = startHeight
		result.FromHeight = fh
		item := make([]jsonresult.OutCoin, 0)

		//If the ReadonlyKey is provided, return decrypted coins
		if len(outputCoins) == 0{
			for _, outCoin := range plainOutputCoins {
				tmp := jsonresult.NewOutCoin(outCoin)
				db := coinService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB()

				if outCoin.GetVersion() == 2{
					tmpCoin, ok := outCoin.(*coin.CoinV2)
					if !ok{
						continue
					}

					//Retrieve coin's index
					publicKeyBytes := tmpCoin.GetPublicKey().ToBytesS()
					idx, err := statedb.GetOTACoinIndex(db, tokenID, publicKeyBytes)
					if err != nil{
						return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
					}

					tmp.Index = base58.Base58Check{}.Encode(idx.Bytes(), common.ZeroByte)
					if tmpCoin.GetSharedRandom() != nil{
						tmp.SharedRandom = base58.Base58Check{}.Encode(tmpCoin.GetSharedRandom().ToBytesS(), common.ZeroByte)
					}
				}

				item = append(item, tmp)
			}
			if readonlyKey != nil && len(readonlyKey.KeySet.ReadonlyKey.Rk) > 0 {
				result.Outputs[readonlyKeyStr] = item
			} else {
				result.Outputs[paymentAddressStr] = item
			}
		}else{//ReadonlyKey is not provided, return raw coins
			for _, outCoin := range outputCoins {
				tmp := jsonresult.NewOutCoin(outCoin)
				db := coinService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB()

				if outCoin.GetVersion() == 2{
					tmpCoin, ok := outCoin.(*coin.CoinV2)
					if !ok{
						continue
					}

					//Retrieve coin's index
					publicKeyBytes := tmpCoin.GetPublicKey().ToBytesS()
					idx, err := statedb.GetOTACoinIndex(db, tokenID, publicKeyBytes)
					if err != nil{
						return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
					}

					tmp.Index = base58.Base58Check{}.Encode(idx.Bytes(), common.ZeroByte)
					if tmpCoin.GetSharedRandom() != nil{
						tmp.SharedRandom = base58.Base58Check{}.Encode(tmpCoin.GetSharedRandom().ToBytesS(), common.ZeroByte)
					}
				}

				item = append(item, tmp)
			}
			result.Outputs[paymentAddressStr] = item
		}
	}
	return result, nil
}

func (coinService CoinService) ListCachedOutputCoinsByKey(listKeyParams []interface{}, tokenID common.Hash) (*jsonresult.ListOutputCoins, *RPCError) {
	result := &jsonresult.ListOutputCoins{
		Outputs: make(map[string][]jsonresult.OutCoin),
	}
	for _, keyParam := range listKeyParams {
		keys, ok := keyParam.(map[string]interface{})
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid params: %+v", keyParam))
		}
		// get OTA secretKey deserializing (compulsory for V2, optional for V1)
		var otaKey *wallet.KeyWallet
		var err error
		otaKeyStr, ok := keys["OTASecretKey"].(string)
		if !ok || otaKeyStr == "" {
			Logger.log.Info("otaKey is optional")
		} else {
			otaKey, err = wallet.Base58CheckDeserialize(otaKeyStr)
			if err != nil {
				Logger.log.Debugf("otaKey is invalid: err: %+v", err)
				return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
			}
		}
		// get keyset only contain read only key by deserializing (optional)
		var readonlyKey *wallet.KeyWallet
		readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
		if !ok || readonlyKeyStr == "" {
			Logger.log.Info("Read onlyKey is optional")
		} else {
			readonlyKey, err = wallet.Base58CheckDeserialize(readonlyKeyStr)
			if err != nil {
				Logger.log.Debugf("Read onlyKey is invalid: err: %+v", err)
				return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
			}
		}
		// get keyset only contain public key by deserializing (required)
		paymentAddressStr, ok := keys["PaymentAddress"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid payment address"))
		}
		paymentAddressKey, err := wallet.Base58CheckDeserialize(paymentAddressStr)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
		}

		// Get start height
		// startHeightTemp, ok := keys["StartHeight"].(float64)
		// if !ok {
		// 	return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid start height"))
		// }
		// startHeight := uint64(startHeightTemp)

		// create a key set
		keySet := incognitokey.KeySet{
			PaymentAddress: paymentAddressKey.KeySet.PaymentAddress,
		}
		// readonly key is optional
		if readonlyKey != nil && len(readonlyKey.KeySet.ReadonlyKey.Rk) > 0 {
			keySet.ReadonlyKey = readonlyKey.KeySet.ReadonlyKey
		}
		if otaKey != nil && otaKey.KeySet.OTAKey.GetOTASecretKey()!= nil {
			keySet.OTAKey = otaKey.KeySet.OTAKey
		}

		lastByte := keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		plainOutputCoins, err := coinService.BlockChain.TryGetAllOutputCoinsByKeyset(&keySet, shardIDSender, &tokenID, true)
		var outputCoins []coin.Coin
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
		}
		result.ToHeight = 0
		result.FromHeight = 0
		item := make([]jsonresult.OutCoin, 0)

		//If the ReadonlyKey is provided, return decrypted coins
		if len(outputCoins) == 0{
			for _, outCoin := range plainOutputCoins {
				tmp := jsonresult.NewOutCoin(outCoin)
				db := coinService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB()

				if outCoin.GetVersion() == 2{
					tmpCoin, ok := outCoin.(*coin.CoinV2)
					if !ok{
						continue
					}

					//Retrieve coin's index
					publicKeyBytes := tmpCoin.GetPublicKey().ToBytesS()
					idx, err := statedb.GetOTACoinIndex(db, tokenID, publicKeyBytes)
					if err != nil{
						return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
					}

					tmp.Index = base58.Base58Check{}.Encode(idx.Bytes(), common.ZeroByte)
					if tmpCoin.GetSharedRandom() != nil{
						tmp.SharedRandom = base58.Base58Check{}.Encode(tmpCoin.GetSharedRandom().ToBytesS(), common.ZeroByte)
					}
				}

				item = append(item, tmp)
			}
			if readonlyKey != nil && len(readonlyKey.KeySet.ReadonlyKey.Rk) > 0 {
				result.Outputs[readonlyKeyStr] = item
			} else {
				result.Outputs[paymentAddressStr] = item
			}
		}else{//ReadonlyKey is not provided, return raw coins
			for _, outCoin := range outputCoins {
				tmp := jsonresult.NewOutCoin(outCoin)
				db := coinService.BlockChain.GetBestStateShard(shardIDSender).GetCopiedTransactionStateDB()

				if outCoin.GetVersion() == 2{
					tmpCoin, ok := outCoin.(*coin.CoinV2)
					if !ok{
						continue
					}

					//Retrieve coin's index
					publicKeyBytes := tmpCoin.GetPublicKey().ToBytesS()
					idx, err := statedb.GetOTACoinIndex(db, tokenID, publicKeyBytes)
					if err != nil{
						return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
					}

					tmp.Index = base58.Base58Check{}.Encode(idx.Bytes(), common.ZeroByte)
					if tmpCoin.GetSharedRandom() != nil{
						tmp.SharedRandom = base58.Base58Check{}.Encode(tmpCoin.GetSharedRandom().ToBytesS(), common.ZeroByte)
					}
				}

				item = append(item, tmp)
			}
			result.Outputs[paymentAddressStr] = item
		}
	}
	return result, nil
}

func (coinService CoinService) ListUnspentOutputTokensByKey(listKeyParams []interface{}) (*jsonresult.ListOutputCoins, *RPCError) {
	result := &jsonresult.ListOutputCoins{
		Outputs: make(map[string][]jsonresult.OutCoin),
	}
	for _, keyParam := range listKeyParams {
		keys, ok := keyParam.(map[string]interface{})
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Invalid Params %+v", keyParam))
		}
		// get keyset only contain private key by deserializing
		privateKeyStr, ok := keys["PrivateKey"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("private key is invalid"))
		}
		keyWallet, err := wallet.Base58CheckDeserialize(privateKeyStr)
		if err != nil || keyWallet.KeySet.PrivateKey == nil {
			Logger.log.Error("Check Deserialize err", err)
			return nil, NewRPCError(RPCInvalidParamsError, fmt.Errorf("Private key is invalid, error %+v", err))
		}
		keySetTmp, shardID, err := GetKeySetFromPrivateKey(keyWallet.KeySet.PrivateKey)
		if err != nil {
			return nil, NewRPCError(ListUnspentOutputCoinsByKeyError, err)
		}
		keyWallet.KeySet = *keySetTmp

		// get shard height
		shardHeightTemp, ok := keys["StartHeight"].(float64)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid height param"))
		}
		shardHeight := uint64(shardHeightTemp)

		//get token ID
		tokenIDParam, ok := keys["tokenID"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid tokenID param"))
		}
		tokenID, err := common.Hash{}.NewHashFromStr(tokenIDParam)
		if err != nil {
			return nil, NewRPCError(RPCInvalidParamsError, err)
		}

		outCoins, _, fh, err := coinService.BlockChain.GetListDecryptedOutputCoinsByKeyset(&keyWallet.KeySet, shardID, tokenID, shardHeight)
		if err != nil {
			return nil, NewRPCError(ListUnspentOutputCoinsByKeyError, err)
		}
		result.ToHeight = shardHeight
		result.FromHeight = fh
		item := make([]jsonresult.OutCoin, 0)
		for _, outCoin := range outCoins {
			if outCoin.GetValue() == 0{
				continue
			}
			tmp := jsonresult.NewOutCoin(outCoin)
			db := coinService.BlockChain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()

			if outCoin.GetVersion() == 2{
				tmpCoin, ok := outCoin.(*coin.CoinV2)
				if !ok{
					continue
				}

				publicKeyBytes := tmpCoin.GetPublicKey().ToBytesS()
				idx, err := statedb.GetOTACoinIndex(db, *tokenID, publicKeyBytes)
				if err != nil{
					return nil, NewRPCError(ListDecryptedOutputCoinsByKeyError, err)
				}

				tmp.Index = base58.Base58Check{}.Encode(idx.Bytes(), common.ZeroByte)
				if tmpCoin.GetSharedRandom() != nil{
					tmp.SharedRandom = base58.Base58Check{}.Encode(tmpCoin.GetSharedRandom().ToBytesS(), common.ZeroByte)
				}
				if tmpCoin.GetTxRandom() != nil{
					tmp.TxRandom = base58.Base58Check{}.Encode(tmpCoin.GetTxRandom().Bytes(), common.ZeroByte)
				}
			}

			item = append(item, tmp)
		}
		result.Outputs[privateKeyStr] = item
	}
	return result, nil
}