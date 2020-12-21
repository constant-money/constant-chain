package rpcservice

import (
	"encoding/hex"
	"errors"
	"math/rand"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/wallet"
)

type WalletService struct {
	Wallet     *wallet.Wallet
	BlockChain *blockchain.BlockChain
}

func (walletService WalletService) ListAccounts() (jsonresult.ListAccounts, *RPCError) {
	result := jsonresult.ListAccounts{
		Accounts:   make(map[string]uint64),
		WalletName: walletService.Wallet.Name,
	}
	accounts := walletService.Wallet.ListAccounts()
	for accountName, account := range accounts {
		lastByte := account.Key.KeySet.PaymentAddress.Pk[len(account.Key.KeySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		prvCoinID := &common.Hash{}
		err := prvCoinID.SetBytes(common.PRVCoinID[:])
		if err != nil {
			return jsonresult.ListAccounts{}, NewRPCError(TokenIsInvalidError, err)
		}

		// List account should get all balance so startHeight = 0
		outCoins, err := walletService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.Key.KeySet, shardIDSender, prvCoinID, true)
		if err != nil {
			return jsonresult.ListAccounts{}, NewRPCError(UnexpectedError, err)
		}
		amount := uint64(0)
		for _, out := range outCoins {
			amount += out.GetValue()
		}
		result.Accounts[accountName] = amount
	}

	return result, nil
}

func (walletService WalletService) SubmitKey(keyStr string) (struct{}, *RPCError) {
	// this function accepts a private key or a hex-encoded OTA key
	var otaKey privacy.OTAKey
	keySet, shardIDSender, err := GetKeySetFromPrivateKeyParams(keyStr)
	if err==nil{
		otaKey = keySet.OTAKey
	}else{
		keySlice, err := hex.DecodeString(keyStr)
		if err!=nil || len(keySlice)!=64{
			return struct{}{}, NewRPCError(InvalidSenderViewingKeyError, errors.New("OTA key must be hex-encoded 64 bytes"))
		}
		var b [64]byte
		copy(b[:], keySlice)
		otaKey = blockchain.OTAKeyFromRaw(b)
		shardIDSender = common.GetShardIDFromLastByte(keySlice[len(keySlice)-1])
	}
	result := struct{}{}
	
	err = walletService.BlockChain.SubmitOTAKey(otaKey, shardIDSender)
	if err != nil {
		return struct{}{}, NewRPCError(UnexpectedError, err)
	}
	
	return result, nil
}

func (walletService WalletService) GetAccount(paymentAddrStr string) (string, error) {
	if paymentAddrStr == "" {
		return "", NewRPCError(RPCInvalidParamsError, errors.New("payment address is invalid"))
	}

	for _, account := range walletService.Wallet.MasterAccount.Child {
		address := account.Key.Base58CheckSerialize(wallet.PaymentAddressType)
		if address == paymentAddrStr {
			return account.Name, nil
		}
	}
	return "", nil
}

func (walletService WalletService) GetAddressesByAccount(accountName string) ([]wallet.KeySerializedData, error) {
	return walletService.Wallet.GetAddressesByAccName(accountName), nil
}

func (walletService *WalletService) GetAccountAddress(accountName string) (wallet.KeySerializedData, error) {
	activeShards := walletService.BlockChain.GetBeaconBestState().ActiveShards
	shardID := walletService.Wallet.GetConfig().ShardID
	// if shardID is nil -> create with any shard
	if shardID != nil {
		// if shardID is configured with not nil
		shardIDInt := int(*shardID)
		// check with activeshards
		if shardIDInt >= activeShards || shardIDInt <= 0 {
			randShard := rand.Int31n(int32(activeShards))
			temp := byte(randShard)
			shardID = &temp
		}
	}
	result := walletService.Wallet.GetAddressByAccName(accountName, shardID)
	return result, nil
}

func (walletService WalletService) DumpPrivkey(param string) wallet.KeySerializedData {
	return walletService.Wallet.DumpPrivateKey(param)
}

func (walletService *WalletService) ImportAccount(privateKey string, accountName string, passPhrase string) (wallet.KeySerializedData, error) {
	account, err := walletService.Wallet.ImportAccount(privateKey, accountName, passPhrase)
	if err != nil {
		return wallet.KeySerializedData{}, err
	}
	result := wallet.KeySerializedData{
		PaymentAddress: account.Key.Base58CheckSerialize(wallet.PaymentAddressType),
		Pubkey:         hex.EncodeToString(account.Key.KeySet.PaymentAddress.Pk),
		ReadonlyKey:    account.Key.Base58CheckSerialize(wallet.ReadonlyKeyType),
	}

	return result, nil
}

func (walletService *WalletService) RemoveAccount(privateKey string, passPhrase string) (bool, *RPCError) {
	err := walletService.Wallet.RemoveAccount(privateKey, passPhrase)
	if err != nil {
		return false, NewRPCError(UnexpectedError, err)
	}
	return true, nil
}

func (walletService WalletService) GetBalanceByPrivateKey(privateKey string) (uint64, *RPCError) {
	keySet, shardIDSender, err := GetKeySetFromPrivateKeyParams(privateKey)
	if err != nil {
		return uint64(0), NewRPCError(RPCInvalidParamsError, err)
	}
	if keySet == nil {
		return uint64(0), NewRPCError(InvalidSenderPrivateKeyError, err)
	}
	prvCoinID := &common.Hash{}
	err = prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return uint64(0), NewRPCError(TokenIsInvalidError, err)
	}

	// Get balance by private key should return all tokens belong to this private key, so start at 0
	outcoints, err := walletService.BlockChain.TryGetAllOutputCoinsByKeyset(keySet, shardIDSender, prvCoinID, true)
	if err != nil {
		return uint64(0), NewRPCError(UnexpectedError, err)
	}

	balance := uint64(0)
	for _, out := range outcoints {
		balance += out.GetValue()
	}

	return balance, nil
}

func (walletService WalletService) GetBalanceByPaymentAddress(paymentAddress string) (uint64, *RPCError) {
	keySet, shardIDSender, err := GetKeySetFromPaymentAddressParam(paymentAddress)
	if err != nil {
		return uint64(0), NewRPCError(RPCInvalidParamsError, errors.New("payment address is invalid"))
	}

	prvCoinID := &common.Hash{}
	err1 := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err1 != nil {
		return uint64(0), NewRPCError(TokenIsInvalidError, err1)
	}

	// Get balance should get all, so start from zero
	outcoints, err := walletService.BlockChain.TryGetAllOutputCoinsByKeyset(keySet, shardIDSender, prvCoinID, true)
	Logger.log.Debugf("OutCoins: %+v", outcoints)
	Logger.log.Debugf("shardIDSender: %+v", shardIDSender)
	Logger.log.Debugf("accountWithPaymentAddress.KeySet: %+v", keySet)
	Logger.log.Debugf("paymentAddressParam: %+v", paymentAddress)
	if err != nil {
		return uint64(0), NewRPCError(UnexpectedError, err)
	}
	balance := uint64(0)
	for _, out := range outcoints {
		balance += out.GetValue()
	}

	return balance, nil
}

func (walletService WalletService) GetBalance(accountName string) (uint64, *RPCError) {
	prvCoinID := &common.Hash{}
	err1 := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err1 != nil {
		return uint64(0), NewRPCError(TokenIsInvalidError, err1)
	}

	balance := uint64(0)
	if accountName == "*" {
		// get balance for all accounts in wallet
		for _, account := range walletService.Wallet.MasterAccount.Child {
			lastByte := account.Key.KeySet.PaymentAddress.Pk[len(account.Key.KeySet.PaymentAddress.Pk)-1]
			shardIDSender := common.GetShardIDFromLastByte(lastByte)

			// Get balance should get all, so start from zero
			outCoins, err := walletService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.Key.KeySet, shardIDSender, prvCoinID, true)
			if err != nil {
				return uint64(0), NewRPCError(UnexpectedError, err)
			}
			for _, out := range outCoins {
				balance += out.GetValue()
			}
		}
	} else {
		for _, account := range walletService.Wallet.MasterAccount.Child {
			if account.Name == accountName {
				// get balance for accountName in wallet
				lastByte := account.Key.KeySet.PaymentAddress.Pk[len(account.Key.KeySet.PaymentAddress.Pk)-1]
				shardIDSender := common.GetShardIDFromLastByte(lastByte)

				// Get balance should get all, so start from zero
				outCoins, err := walletService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.Key.KeySet, shardIDSender, prvCoinID, true)
				if err != nil {
					return uint64(0), NewRPCError(UnexpectedError, err)
				}
				for _, out := range outCoins {
					balance += out.GetValue()
				}
				break
			}
		}
	}

	return balance, nil
}

func (walletService WalletService) GetReceivedByAccount(accountName string) (uint64, *RPCError) {
	balance := uint64(0)
	for _, account := range walletService.Wallet.MasterAccount.Child {
		if account.Name == accountName {
			// get balance for accountName in wallet
			lastByte := account.Key.KeySet.PaymentAddress.Pk[len(account.Key.KeySet.PaymentAddress.Pk)-1]
			shardIDSender := common.GetShardIDFromLastByte(lastByte)
			prvCoinID := &common.Hash{}
			err1 := prvCoinID.SetBytes(common.PRVCoinID[:])
			if err1 != nil {
				return uint64(0), NewRPCError(TokenIsInvalidError, err1)
			}

			// Get balance should get all, so start from zero
			outCoins, err := walletService.BlockChain.TryGetAllOutputCoinsByKeyset(&account.Key.KeySet, shardIDSender, prvCoinID, true)
			if err != nil {
				return uint64(0), NewRPCError(UnexpectedError, err)
			}
			for _, out := range outCoins {
				balance += out.GetValue()
			}
			break
		}
	}
	return balance, nil
}
