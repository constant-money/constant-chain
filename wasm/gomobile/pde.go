package gomobile

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/pkg/errors"
	"math/big"
)

func InitPEDContributionMetadataFromParam(metaDataParam map[string]interface{}) (*metadata.PDEContribution, error) {
	metaDataType, ok := metaDataParam["Type"].(float64)
	if !ok {
		println("Invalid meta data type param")
		return nil, errors.New("Invalid meta data type param")
	}

	pdeContributionPairID, ok := metaDataParam["PDEContributionPairID"].(string)
	if !ok {
		println("Invalid meta data pde contribution pair ID param")
		return nil, errors.New("Invalid meta data pde contribution pair ID param")
	}
	contributorAddressStr, ok := metaDataParam["ContributorAddressStr"].(string)
	if !ok {
		println("Invalid meta data contributor payment address param")
		return nil, errors.New("Invalid meta data contributor payment address param")
	}
	contributedAmount, err := common.AssertAndConvertStrToNumber(metaDataParam["ContributedAmount"])
	if err != nil {
		println("Invalid meta data contribute amount param")
		return nil, errors.New("Invalid meta data contribute amount param")
	}
	tokenIDStr, ok := metaDataParam["TokenIDStr"].(string)
	if !ok {
		println("Invalid meta data token id string param")
		return nil, errors.New("Invalid meta data token id string param")
	}

	metaData, err := metadata.NewPDEContribution(
		pdeContributionPairID, contributorAddressStr, uint64(contributedAmount), tokenIDStr, int(metaDataType),
	)
	if err != nil {
		return nil, err
	}

	return metaData, nil
}

func InitPRVContributionTx(args string, serverTime int64) (string, error) {
	// parse meta data
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	println("paramMaps:", paramMaps)

	metaDataParam, ok := paramMaps["metaData"].(map[string]interface{})
	if !ok {
		return "", errors.New("Invalid meta data param")
	}

	metaData, err := InitPEDContributionMetadataFromParam(metaDataParam)
	if err != nil {
		return "", err
	}

	paramCreateTx, err := InitParamCreatePrivacyTx(args)
	if err != nil {
		return "", err
	}

	paramCreateTx.SetMetaData(metaData)

	tx := new(transaction.Tx)
	err = tx.InitForASM(paramCreateTx, serverTime)

	if err != nil {
		println("Can not create tx: ", err)
		return "", err
	}

	// serialize tx json
	txJson, err := json.Marshal(tx)
	if err != nil {
		println("Can not marshal tx: ", err)
		return "", err
	}

	lockTimeBytes := common.AddPaddingBigInt(new(big.Int).SetInt64(tx.LockTime), 8)
	resBytes := append(txJson, lockTimeBytes...)

	B64Res := base64.StdEncoding.EncodeToString(resBytes)

	return B64Res, nil
}

func InitPTokenContributionTx(args string, serverTime int64) (string, error) {
	// parse meta data
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	println("paramMaps:", paramMaps)

	metaDataParam, ok := paramMaps["metaData"].(map[string]interface{})
	if !ok {
		return "", errors.New("Invalid meta data param")
	}

	metaData, err := InitPEDContributionMetadataFromParam(metaDataParam)
	if err != nil {
		return "", err
	}

	paramCreateTx, err := InitParamCreatePrivacyTokenTx(args)
	if err != nil {
		return "", err
	}

	paramCreateTx.SetMetaData(metaData)

	tx := new(transaction.TxTokenBase)
	err = tx.InitForASM(paramCreateTx, serverTime)

	if err != nil {
		println("Can not create tx: ", err)
		return "", err
	}

	// serialize tx json
	txJson, err := json.Marshal(tx)
	if err != nil {
		println("Can not marshal tx: ", err)
		return "", err
	}

	lockTimeBytes := common.AddPaddingBigInt(new(big.Int).SetInt64(tx.LockTime), 8)
	resBytes := append(txJson, lockTimeBytes...)

	B64Res := base64.StdEncoding.EncodeToString(resBytes)

	return B64Res, nil
}

func InitPEDTradeRequestMetadataFromParam(metaDataParam map[string]interface{}) (*metadata.PDETradeRequest, error) {
	metaDataType, ok := metaDataParam["Type"].(float64)
	if !ok {
		println("Invalid meta data type param")
		return nil, errors.New("Invalid meta data type param")
	}

	tokenIDToBuyStr, ok := metaDataParam["TokenIDToBuyStr"].(string)
	if !ok {
		println("Invalid meta data token id to buy param")
		return nil, errors.New("Invalid meta data token id to buy param")
	}
	tokenIDToSellStr, ok := metaDataParam["TokenIDToSellStr"].(string)
	if !ok {
		println("Invalid meta data token id to sell param")
		return nil, errors.New("Invalid meta data token id to sell param")
	}
	sellAmount, err := common.AssertAndConvertStrToNumber(metaDataParam["SellAmount"])
	if err != nil {
		println("Invalid meta data sell amount param")
		return nil, errors.New("Invalid meta data sell amount param")
	}

	minAcceptableAmount, err := common.AssertAndConvertStrToNumber(metaDataParam["MinAcceptableAmount"])
	if err != nil {
		println("Invalid meta data min acceptable amount param")
		return nil, errors.New("Invalid meta data min acceptable amount param")
	}

	tradingFee, err := common.AssertAndConvertStrToNumber(metaDataParam["TradingFee"])
	if err != nil {
		println("Invalid meta data trading fee param")
		return nil, errors.New("Invalid meta data trading fee param")
	}
	traderAddressStr, ok := metaDataParam["TraderAddressStr"].(string)
	if !ok {
		println("Invalid meta data trader address string param")
		return nil, errors.New("Invalid meta data trader address string param")
	}

	metaData, err := metadata.NewPDETradeRequest(
		tokenIDToBuyStr, tokenIDToSellStr, uint64(sellAmount), uint64(minAcceptableAmount), uint64(tradingFee), traderAddressStr, int(metaDataType),
	)
	if err != nil {
		return nil, err
	}

	return metaData, nil
}

func InitPRVTradeTx(args string, serverTime int64) (string, error) {
	// parse meta data
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	println("paramMaps:", paramMaps)

	metaDataParam, ok := paramMaps["metaData"].(map[string]interface{})
	if !ok {
		return "", errors.New("Invalid meta data param")
	}

	metaData, err := InitPEDTradeRequestMetadataFromParam(metaDataParam)
	if err != nil {
		return "", err
	}

	paramCreateTx, err := InitParamCreatePrivacyTx(args)
	if err != nil {
		return "", err
	}

	paramCreateTx.SetMetaData(metaData)

	tx := new(transaction.Tx)
	err = tx.InitForASM(paramCreateTx, serverTime)

	if err != nil {
		println("Can not create tx: ", err)
		return "", err
	}

	// serialize tx json
	txJson, err := json.Marshal(tx)
	if err != nil {
		println("Can not marshal tx: ", err)
		return "", err
	}

	lockTimeBytes := common.AddPaddingBigInt(new(big.Int).SetInt64(tx.LockTime), 8)
	resBytes := append(txJson, lockTimeBytes...)

	B64Res := base64.StdEncoding.EncodeToString(resBytes)

	return B64Res, nil
}

func InitPTokenTradeTx(args string, serverTime int64) (string, error) {
	// parse meta data
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	println("paramMaps:", paramMaps)

	metaDataParam, ok := paramMaps["metaData"].(map[string]interface{})
	if !ok {
		return "", errors.New("Invalid meta data param")
	}

	metaData, err := InitPEDTradeRequestMetadataFromParam(metaDataParam)
	if err != nil {
		return "", err
	}

	paramCreateTx, err := InitParamCreatePrivacyTokenTx(args)
	if err != nil {
		return "", err
	}

	paramCreateTx.SetMetaData(metaData)

	tx := new(transaction.TxTokenBase)
	err = tx.InitForASM(paramCreateTx, serverTime)

	if err != nil {
		println("Can not create tx: ", err)
		return "", err
	}

	// serialize tx json
	txJson, err := json.Marshal(tx)
	if err != nil {
		println("Can not marshal tx: ", err)
		return "", err
	}

	lockTimeBytes := common.AddPaddingBigInt(new(big.Int).SetInt64(tx.LockTime), 8)
	resBytes := append(txJson, lockTimeBytes...)

	B64Res := base64.StdEncoding.EncodeToString(resBytes)

	return B64Res, nil
}

func WithdrawDexTx(args string, serverTime int64) (string, error) {
	// parse meta data
	bytes := []byte(args)
	println("Bytes: %v\n", bytes)

	paramMaps := make(map[string]interface{})

	err := json.Unmarshal(bytes, &paramMaps)
	if err != nil {
		println("Error can not unmarshal data : %v\n", err)
		return "", err
	}

	println("paramMaps:", paramMaps)

	metaDataParam, ok := paramMaps["metaData"].(map[string]interface{})
	if !ok {
		return "", errors.New("Invalid meta data param")
	}

	metaDataType, ok := metaDataParam["Type"].(float64)
	if !ok {
		println("Invalid meta data type param")
		return "", errors.New("Invalid meta data type param")
	}

	withdrawerAddressStr, ok := metaDataParam["WithdrawerAddressStr"].(string)
	if !ok {
		println("Invalid meta data withdrawerAddressStr param")
		return "", errors.New("Invalid meta data withdrawerAddressStr param")
	}
	withdrawalToken1IDStr, ok := metaDataParam["WithdrawalToken1IDStr"].(string)
	if !ok {
		println("Invalid meta data withdrawalToken1IDStr param")
		return "", errors.New("Invalid meta data withdrawalToken1IDStr param")
	}
	withdrawalToken2IDStr, ok := metaDataParam["WithdrawalToken2IDStr"].(string)
	if !ok {
		println("Invalid meta data withdrawalToken2IDStr param")
		return "", errors.New("Invalid meta data withdrawalToken2IDStr param")
	}
	withdrawalShareAmt, err := common.AssertAndConvertStrToNumber(metaDataParam["WithdrawalShareAmt"])
	if err != nil {
		println("Invalid meta data withdrawalShareAmt param")
		return "", errors.New("Invalid meta data withdrawalShareAmt param")
	}
	metaData, err := metadata.NewPDEWithdrawalRequest(
		withdrawerAddressStr, withdrawalToken1IDStr,
		withdrawalToken2IDStr, uint64(withdrawalShareAmt), int(metaDataType),
	)
	if err != nil {
		return "", err
	}

	paramCreateTx, err := InitParamCreatePrivacyTx(args)
	if err != nil {
		return "", err
	}

	paramCreateTx.SetMetaData(metaData)

	tx := new(transaction.Tx)
	err = tx.InitForASM(paramCreateTx, serverTime)

	if err != nil {
		println("Can not create tx: ", err)
		return "", err
	}

	// serialize tx json
	txJson, err := json.Marshal(tx)
	if err != nil {
		println("Can not marshal tx: ", err)
		return "", err
	}

	lockTimeBytes := common.AddPaddingBigInt(new(big.Int).SetInt64(tx.LockTime), 8)
	resBytes := append(txJson, lockTimeBytes...)

	B64Res := base64.StdEncoding.EncodeToString(resBytes)

	return B64Res, nil
}
