package gomobile

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wallet"

	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/pkg/errors"
	"math/big"
)

func InitPrivacyTx(args string, serverTime int64) (string, error) {
	paramCreateTx, err := InitParamCreatePrivacyTx(args)
	if err != nil {
		return "", err
	}

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

func Staking(args string, serverTime int64) (string, error) {
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

	funderPaymentAddress, ok := metaDataParam["FunderPaymentAddress"].(string)
	if !ok {
		println("Invalid meta data funder payment address param")
		return "", errors.New("Invalid meta data funder payment address param")
	}
	rewardReceiverPaymentAddress, ok := metaDataParam["RewardReceiverPaymentAddress"].(string)
	if !ok {
		println("Invalid meta data reward receiver payment address param")
		return "", errors.New("Invalid meta data reward receiver payment address param")
	}
	stakingAmountShard, err := common.AssertAndConvertStrToNumber(metaDataParam["StakingAmountShard"])
	if err != nil {
		println("Invalid meta data staking amount param")
		return "", errors.New("Invalid meta data staking amount param")
	}
	committeePublicKey, ok := metaDataParam["CommitteePublicKey"].(string)
	if !ok {
		println("Invalid meta data committee public key param")
		return "", errors.New("Invalid meta data committee public key param")
	}
	autoReStaking, ok := metaDataParam["AutoReStaking"].(bool)
	if !ok {
		println("Invalid meta data auto restaking param")
		return "", errors.New("Invalid meta data auto restaking param")
	}

	metaData, err := metadata.NewStakingMetadata(int(metaDataType), funderPaymentAddress, rewardReceiverPaymentAddress, uint64(stakingAmountShard), committeePublicKey, autoReStaking)
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

func StopAutoStaking(args string, serverTime int64) (string, error) {
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

	committeePublicKey, ok := metaDataParam["CommitteePublicKey"].(string)
	if !ok {
		println("Invalid meta data committee public key param")
		return "", errors.New("Invalid meta data committee public key param")
	}

	metaData, err := metadata.NewStopAutoStakingMetadata(int(metaDataType), committeePublicKey)
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

func InitWithdrawRewardTx(args string, serverTime int64) (string, error) {
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

	paymentAddressParam, ok := metaDataParam["PaymentAddress"].(string)
	if !ok {
		println("Invalid meta data payment address param")
		return "", errors.New("Invalid meta data payment address param")
	}
	keyWallet, err := wallet.Base58CheckDeserialize(paymentAddressParam)
	if err != nil {
		return "", nil
	}
	paymentAddress := keyWallet.KeySet.PaymentAddress

	tokenIDParam, ok := metaDataParam["TokenID"].(string)
	if !ok {
		println("Invalid meta data token id param")
		return "", errors.New("Invalid meta data token id param")
	}

	tokenId, err := new(common.Hash).NewHashFromStr(tokenIDParam)
	if err != nil {
		return "", err
	}

	tmp := &metadata.WithDrawRewardRequest{
		PaymentAddress: paymentAddress,
		MetadataBase:   *metadata.NewMetadataBase(int(metaDataType)),
		TokenID:        *tokenId,
		Version:        1,
	}

	paramCreateTx, err := InitParamCreatePrivacyTx(args)
	if err != nil {
		return "", err
	}

	paramCreateTx.SetMetaData(tmp)

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
