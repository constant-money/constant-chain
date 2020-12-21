package gomobile

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
	"math/big"
)

func InitPrivacyTokenTx(args string, serverTime int64) (string, error) {
	paramCreateTx, err := InitParamCreatePrivacyTokenTx(args)
	if err != nil {
		return "", err
	}

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

	tokenIDBytes := tx.TxTokenData.PropertyID.GetBytes()

	lockTimeBytes := common.AddPaddingBigInt(new(big.Int).SetInt64(tx.LockTime), 8)
	resBytes := append(txJson, lockTimeBytes...)
	resBytes = append(resBytes, tokenIDBytes...)

	B64Res := base64.StdEncoding.EncodeToString(resBytes)

	return B64Res, nil
}

func InitBurningRequestTx(args string, serverTime int64) (string, error) {
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

	burnerAddressParam, ok := metaDataParam["BurnerAddress"].(string)
	if !ok {
		println("Invalid meta data burner payment address param")
		return "", errors.New("Invalid meta data burner payment address param")
	}
	keyWalletBurner, err := wallet.Base58CheckDeserialize(burnerAddressParam)
	if err != nil {
		return "", nil
	}
	burnerAddress := keyWalletBurner.KeySet.PaymentAddress
	burningAmount, err := common.AssertAndConvertStrToNumber(metaDataParam["BurningAmount"])
	if err != nil {
		println("Invalid meta data burning amount param")
		return "", errors.New("Invalid meta data burning amount param")
	}
	tokenID, ok := metaDataParam["TokenID"].(string)
	if !ok {
		println("Invalid meta data token id param")
		return "", errors.New("Invalid meta data token id param")
	}
	tokenIDHash, err := new(common.Hash).NewHashFromStr(tokenID)
	if err != nil {
		return "", err
	}

	tokenName, ok := metaDataParam["TokenName"].(string)
	if !ok {
		println("Invalid meta data token name param")
		return "", errors.New("Invalid meta data token name param")
	}
	remoteAddress, ok := metaDataParam["RemoteAddress"].(string)
	if !ok {
		println("Invalid meta data remote address param")
		return "", errors.New("Invalid meta data remote address param")
	}

	metaData, err := metadata.NewBurningRequest(burnerAddress, uint64(burningAmount), *tokenIDHash, tokenName, remoteAddress, int(metaDataType))
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

	tokenIDBytes := tx.TxTokenData.PropertyID.GetBytes()

	lockTimeBytes := common.AddPaddingBigInt(new(big.Int).SetInt64(tx.LockTime), 8)
	resBytes := append(txJson, lockTimeBytes...)
	resBytes = append(resBytes, tokenIDBytes...)

	B64Res := base64.StdEncoding.EncodeToString(resBytes)

	return B64Res, nil
}
