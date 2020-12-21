package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

/*
listaccount RPC lists accounts and their balances.

Parameter #1—the minimum number of confirmations a transaction must have
Parameter #2—whether to include watch-only addresses in results
Result—a list of accounts and their balances

*/
func (httpServer *HttpServer) handleListAccounts(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	result, err := httpServer.walletService.ListAccounts()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/*
this submits a chain-facing `OTA key` to view its balances later

Parameter #1—the OTA key that will be submitted
Result—success or error

*/
func (httpServer *HttpServer) handleSubmitKey(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams==nil || len(arrayParams)!=1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array with 1 element"))
	}
	key, ok := arrayParams[0].(string)
	if !ok{
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("OTA key is invalid"))
	}
	result, err := httpServer.walletService.SubmitKey(key)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/*
getaccount RPC returns the name of the account associated with the given address.
- Param #1: address
*/
func (httpServer *HttpServer) handleGetAccount(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramTemp, ok := params.(string)
	if !ok {
		return nil, nil
	}

	accountName, _ := httpServer.walletService.GetAccount(paramTemp)
	if accountName != "" {
		return accountName, nil
	}

	return nil, nil
}

/*
getaddressesbyaccount RPC returns a list of every address assigned to a particular account.

Parameter #1—the account name
Result—a list of addresses
*/
func (httpServer *HttpServer) handleGetAddressesByAccount(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramTemp, ok := params.(string)
	if !ok {
		return nil, nil
	}

	addresses, err := httpServer.walletService.GetAddressesByAccount(paramTemp)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.GetAddressesByAccount{}
	result.Addresses = addresses
	return result, nil
}

/*
getaccountaddress RPC returns the current coin address for receiving payments to this account.
If the account doesn’t exist, it creates both the account and a new address for receiving payment.
Once a payment has been received to an address, future calls to this RPC for the same account will return a different address.
Parameter #1—an account name
Result—a incognito address
*/
func (httpServer *HttpServer) handleGetAccountAddress(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	accountName, ok := params.(string)
	if !ok {
		return nil, nil
	}

	result, err := httpServer.walletService.GetAccountAddress(accountName)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return result, nil
}

/*
 dumpprivkey RPC returns the wallet-import-format (WIP) private key corresponding to an address. (But does not remove it from the wallet.)

Parameter #1—the address corresponding to the private key to get
Result—the private key
*/
func (httpServer *HttpServer) handleDumpPrivkey(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramTemp, ok := params.(string)
	if !ok {
		return nil, nil
	}
	result := httpServer.walletService.DumpPrivkey(paramTemp)
	return result, nil
}

/*
handleImportAccount - import a new account by private-key
- Param #1: private-key string
- Param #2: account name
- Param #3: passPhrase of wallet
*/
func (httpServer *HttpServer) handleImportAccount(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 3 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	privateKey, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("privateKey is invalid"))
	}

	accountName, ok := arrayParams[1].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("accountName is invalid"))
	}

	passPhrase, ok := arrayParams[2].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("passPhrase is invalid"))
	}

	result, err := httpServer.walletService.ImportAccount(privateKey, accountName, passPhrase)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleRemoveAccount(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}

	privateKey, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("privateKey is invalid"))
	}

	passPhrase, ok := arrayParams[1].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("passPhrase is invalid"))
	}

	return httpServer.walletService.RemoveAccount(privateKey, passPhrase)
}

// handleGetBalanceByPrivatekey -  return balance of private key
func (httpServer *HttpServer) handleGetBalanceByPrivatekey(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	// all component
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) != 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}
	// param #1: private key of sender
	senderKeyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("invalid private key"))
	}

	return httpServer.walletService.GetBalanceByPrivateKey(senderKeyParam)
}

// handleGetBalanceByPaymentAddress -  return balance of paymentaddress
func (httpServer *HttpServer) handleGetBalanceByPaymentAddress(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {

	// all component
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) != 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}
	// param #1: private key of sender
	paymentAddressParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("payment address is invalid"))
	}

	return httpServer.walletService.GetBalanceByPaymentAddress(paymentAddressParam)
}

/*
handleGetBalance - RPC gets the balances in decimal
*/
func (httpServer *HttpServer) handleGetBalance(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	if httpServer.config.Wallet == nil {
		return uint64(0), rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("wallet is not existed"))
	}
	if len(httpServer.config.Wallet.MasterAccount.Child) == 0 {
		return uint64(0), rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("no account is existed"))
	}

	// convert component to array
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 3 {
		return uint64(0), rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}
	// Param #1: account "*" for all or a particular account
	accountName, ok := arrayParams[0].(string)
	if !ok {
		return uint64(0), rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("accountName is invalid"))
	}

	// Param #2: the minimum number of confirmations an output must have
	minTemp, ok := arrayParams[1].(float64)
	if !ok {
		return uint64(0), rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min is invalid"))
	}
	min := int(minTemp)
	_ = min

	// Param #3: passphrase to access local wallet of node
	passPhrase, ok := arrayParams[2].(string)
	if !ok {
		return uint64(0), rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("passPhrase is invalid"))
	}

	if passPhrase != httpServer.config.Wallet.PassPhrase {
		return uint64(0), rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("password phrase is wrong for local wallet"))
	}

	return httpServer.walletService.GetBalance(accountName)
}

/*
handleGetReceivedByAccount -  RPC returns the total amount received by addresses in a
particular account from transactions with the specified number of confirmations. It does not count salary transactions.
*/
func (httpServer *HttpServer) handleGetReceivedByAccount(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	balance := uint64(0)

	if httpServer.config.Wallet == nil {
		return balance, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("wallet is not existed"))
	}
	if len(httpServer.config.Wallet.MasterAccount.Child) == 0 {
		return balance, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("no account is existed"))
	}

	// convert component to array
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 3 {
		return balance, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}
	// Param #1: account "*" for all or a particular account
	accountName, ok := arrayParams[0].(string)
	if !ok {
		return balance, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("accountName is invalid"))
	}

	// Param #2: the minimum number of confirmations an output must have
	minTemp, ok := arrayParams[1].(float64)
	if !ok {
		return balance, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min is invalid"))
	}
	min := int(minTemp)
	_ = min

	// Param #3: passphrase to access local wallet of node
	passPhrase, ok := arrayParams[2].(string)
	if !ok {
		return balance, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("passPhrase is invalid"))
	}

	if passPhrase != httpServer.config.Wallet.PassPhrase {
		return balance, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("password phrase is wrong for local wallet"))
	}

	return httpServer.walletService.GetReceivedByAccount(accountName)
}

/*
handleSetTxFee - RPC sets the transaction fee per kilobyte paid more by transactions created by this wallet. default is 1 coin per 1 kb
*/
func (httpServer *HttpServer) handleSetTxFee(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramTmp, ok := params.(float64)
	if !ok {
		return false, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param is invalid"))
	}

	httpServer.config.Wallet.GetConfig().IncrementalFee = uint64(paramTmp)
	err := httpServer.config.Wallet.Save(httpServer.config.Wallet.PassPhrase)
	return err == nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
}

func (httpServer *HttpServer) handleListPrivacyCustomToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	getCountTxs := false
	if len(arrayParams) == 1 {
		getCountTxs = false //not use anymore
	}
	listPrivacyToken := make(map[common.Hash]*statedb.TokenState)
	var err error
	if getCountTxs {
		listPrivacyToken, err = httpServer.blockService.ListPrivacyCustomTokenWithTxs()
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
		}
	} else {
		listPrivacyToken, err = httpServer.blockService.ListPrivacyCustomToken()
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
		}
	}
	result := jsonresult.ListCustomToken{ListCustomToken: []jsonresult.CustomToken{}}
	for _, tokenState := range listPrivacyToken {
		item := jsonresult.NewPrivacyToken(tokenState)
		result.ListCustomToken = append(result.ListCustomToken, *item)
	}
	// overwrite amounts with bridge tokens
	allBridgeTokens, err := httpServer.blockService.GetAllBridgeTokens()
	if err != nil {
		return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	for _, bridgeToken := range allBridgeTokens {
		if _, ok := listPrivacyToken[*bridgeToken.TokenID]; ok {
			continue
		}
		item := jsonresult.CustomToken{
			ID:            bridgeToken.TokenID.String(),
			IsPrivacy:     true,
			IsBridgeToken: true,
		}
		if item.Name == "" {
			txs, _, err := httpServer.txService.PrivacyCustomTokenDetail(item.ID)
			if err != nil {
				Logger.log.Error(err)
			} else {
				if len(txs) > 1 {
					initTx := txs[0]
					var err2 *rpcservice.RPCError
					tx, err2 := httpServer.txService.GetTransactionByHash(initTx.String())
					if err2 != nil {
						Logger.log.Error(err)
					} else {
						metaData := make(map[string]interface{})
						err1 := json.Unmarshal([]byte(tx.Metadata), &metaData)
						if err1 != nil {
							Logger.log.Error(err)
						} else {
							var ok bool
							item.Name, ok = metaData["TokenName"].(string)
							if !ok {
								Logger.log.Error("Not found token name")
							}
							item.Symbol, ok = metaData["TokenSymbol"].(string)
							if !ok {
								Logger.log.Error("Not found token symbol")
							} else {
								item.Symbol = item.Name
							}
						}
					}
				}
			}
		}
		result.ListCustomToken = append(result.ListCustomToken, item)
	}
	for index, _ := range result.ListCustomToken {
		if !getCountTxs {
			result.ListCustomToken[index].ListTxs = []string{}
		}
		result.ListCustomToken[index].Image = common.Render([]byte(result.ListCustomToken[index].ID))
		for _, bridgeToken := range allBridgeTokens {
			if result.ListCustomToken[index].ID == bridgeToken.TokenID.String() {
				result.ListCustomToken[index].Amount = bridgeToken.Amount
				result.ListCustomToken[index].IsBridgeToken = true
				break
			}
		}
	}
	return result, nil
}

func (httpServer *HttpServer) handleGetPrivacyCustomToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) != 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("params is invalid"))
	}
	tokenIDStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("params type is invalid"))
	}
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("tokenID to hash failed %+v", err))
	}
	for _, i := range httpServer.blockService.BlockChain.GetShardIDs() {
		shardID := byte(i)
		tokenState, isExist, err := httpServer.blockService.BlockChain.GetPrivacyTokenState(*tokenID, shardID)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.GetPrivacyTokenError, err)
		}
		if isExist {
			customToken := jsonresult.NewPrivacyToken(tokenState)
			return jsonresult.NewGetCustomToken(isExist, *customToken), nil
		}
	}
	return jsonresult.NewGetCustomToken(false, jsonresult.CustomToken{}), nil
}

func (httpServer *HttpServer) handleListPrivacyCustomTokenByShard(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("params is invalid"))
	}
	shardID, ok := arrayParams[0].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("params type is invalid"))
	}
	listPrivacyToken, err := httpServer.blockService.ListPrivacyCustomTokenWithPRVByShardID(byte(shardID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListTokenNotFoundError, err)
	}
	result := jsonresult.ListCustomToken{ListCustomToken: []jsonresult.CustomToken{}}
	for _, tokenState := range listPrivacyToken {
		item := jsonresult.NewPrivacyToken(tokenState)
		result.ListCustomToken = append(result.ListCustomToken, *item)
	}
	// overwrite amounts with bridge tokens
	allBridgeTokens, err := httpServer.blockService.GetAllBridgeTokens()
	if err != nil {
		return false, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	for index, _ := range result.ListCustomToken {
		result.ListCustomToken[index].Image = common.Render([]byte(result.ListCustomToken[index].ID))
		for _, bridgeToken := range allBridgeTokens {
			if result.ListCustomToken[index].ID == bridgeToken.TokenID.String() {
				result.ListCustomToken[index].Amount = bridgeToken.Amount
				result.ListCustomToken[index].IsBridgeToken = true
				break
			}
		}
	}
	return result, nil
}

// handleGetPublicKeyFromPaymentAddress - return base58check encode of public key which is got from payment address
func (httpServer *HttpServer) handleGetPublicKeyFromPaymentAddress(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("params is invalid"))
	}
	paymentAddress, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("paymentAddress is invalid"))
	}

	keySet, _, err := rpcservice.GetKeySetFromPaymentAddressParam(paymentAddress)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewGetPublicKeyFromPaymentAddressResult(keySet.PaymentAddress.Pk[:])

	return result, nil
}

// ------------------------------------ Defragment output coin of account by combine many input coin in to 1 output coin --------------------
/*
handleDefragmentAccount
*/
func (httpServer *HttpServer) handleDefragmentAccount(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	var err error
	data, err := httpServer.createRawDefragmentAccountTransaction(params, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:    sendResult.(jsonresult.CreateTransactionResult).TxID,
		ShardID: tx.ShardID,
	}
	return result, nil
}

/*
// createRawDefragmentAccountTransaction.
*/
func (httpServer *HttpServer) createRawDefragmentAccountTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	var err error
	tx, err := httpServer.txService.BuildRawDefragmentAccountTransaction(params, nil)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Critical(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	byteArrays, err := json.Marshal(tx)
	if err != nil {
		// return hex for a new tx
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	txShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
		ShardID:         txShardID,
	}
	return result, nil
}

// defragment for token
func (httpServer *HttpServer) handleDefragmentAccountToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	var err error
	data, err := httpServer.createRawDefragmentAccountTokenTransaction(params, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionTokenResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:    sendResult.(jsonresult.CreateTransactionTokenResult).TxID,
		ShardID: tx.ShardID,
	}
	return result, nil
}

// createRawDefragmentAccountTokenTransaction
func (httpServer *HttpServer) createRawDefragmentAccountTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	var err error
	tx, err := httpServer.txService.BuildRawDefragmentPrivacyCustomTokenTransaction(params, nil)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Error(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	byteArrays, err := json.Marshal(tx)
	if err != nil {
		Logger.log.Error(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	result := jsonresult.CreateTransactionTokenResult{
		ShardID:         common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte()),
		TxID:            tx.Hash().String(),
		TokenID:         tx.GetTxTokenData().PropertyID.String(),
		TokenName:       tx.GetTxTokenData().PropertyName,
		TokenAmount:     tx.GetTxTokenData().Amount,
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

// ----------------------------- End ------------------------------------
