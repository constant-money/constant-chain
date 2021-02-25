package transaction

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy/coin"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/tx_ver1"
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2"
	"github.com/incognitochain/incognito-chain/transaction/utils"
)

var Logger = &utils.Logger

type TxVersion1 		= tx_ver1.Tx
type TxVersion2 		= tx_ver2.Tx
type TxTokenVersion1 	= tx_ver1.TxToken
type TxTokenVersion2	= tx_ver2.TxToken
type TransactionToken 	= tx_generic.TransactionToken
type TokenParam 		= tx_generic.TokenParam
type TxTokenParams 		= tx_generic.TxTokenParams
type TxTokenData 		= tx_generic.TxTokenData
type TxSigPubKeyVer2	= tx_ver2.SigPubKey

func BuildCoinBaseTxByCoinID(params *BuildCoinBaseTxByCoinIDParams) (metadata.Transaction, error) {
	paymentInfo := &privacy.PaymentInfo{
		PaymentAddress: *params.payToAddress,
		Amount: params.amount,
	}
	otaCoin, err := privacy.NewCoinFromPaymentInfo(paymentInfo)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot get new coin from amount and receiver")
		return nil, err
	}
	params.meta.SetSharedRandom(otaCoin.GetSharedRandom().ToBytesS())

	switch params.txType {
	case utils.NormalCoinType:
		tx := new(TxVersion2)
		err = tx.InitTxSalary(otaCoin, params.payByPrivateKey, params.transactionStateDB, params.meta)
		return tx, err
	case utils.CustomTokenPrivacyType:
		var propertyID [common.HashSize]byte
		copy(propertyID[:], params.coinID[:])
		propID := common.Hash(propertyID)
		tx := new(TxTokenVersion2)
		err = tx.InitTxTokenSalary(otaCoin, params.payByPrivateKey, params.transactionStateDB,
			params.meta, &propID, params.coinName)
		return tx, err
	}
	return nil, nil
}

// use this to generate the output coin in salary TXs
// when ReceiverAddress equals nil, we use public key & txRandom
type TxSalaryOutputParams struct{
	Amount 				uint64
	ReceiverAddress 	*privacy.PaymentAddress
	PublicKey 			*privacy.Point
	TxRandom 			*privacy.TxRandom
	TokenID 			*common.Hash
	Info 				[]byte
	Type 				string
}

func (pr TxSalaryOutputParams) getVersion() int{
	// can create mint TX iff the version is 1 or 2
	if pr.PublicKey!=nil && pr.TxRandom!=nil{
		return 2
	}
	if _, err := metadata.AssertPaymentAddressAndTxVersion(*pr.ReceiverAddress, 2); err == nil{
		return 2
	}
	if _, err := metadata.AssertPaymentAddressAndTxVersion(*pr.ReceiverAddress, 1); err == nil{
		return 1
	} else {
		Logger.Log.Errorf("AssertPaymentAddressAndTxVersion error: %v\n", err)
		return 0
	}
}

func (pr TxSalaryOutputParams) generateOutputCoin() (*privacy.CoinV2, error){
	var err error
	var outputCoin *privacy.CoinV2
	isPRV := (pr.TokenID==nil) || (*pr.TokenID==common.PRVCoinID)
	if pr.ReceiverAddress==nil{
		outputCoin = privacy.NewCoinFromAmountAndTxRandomBytes(pr.Amount, pr.PublicKey, pr.TxRandom, pr.Info)
	}else{
		outputCoin, err = privacy.NewCoinFromPaymentInfo(&privacy.PaymentInfo{Amount: pr.Amount, PaymentAddress: *pr.ReceiverAddress})
		if err != nil{
			return nil, err
		}
	}
	// for salary TX, tokenID is never blinded; so when making coin for token transactions, we set an unblinded asset tag
	if !isPRV{
		err := outputCoin.SetPlainTokenID(pr.TokenID)
		if err!=nil{
			return nil, err
		}
	}
	return outputCoin, err
}

func (pr TxSalaryOutputParams) BuildTxSalary(privateKey *privacy.PrivateKey, stateDB *statedb.StateDB, mdMaker func(privacy.Coin) metadata.Metadata) (metadata.Transaction, error) {
	var res metadata.Transaction
	isPRV := (pr.TokenID==nil) || (*pr.TokenID==common.PRVCoinID)
	switch pr.getVersion(){
	case 1:
		if isPRV{
			temp := new(TxVersion1)
			err := temp.InitTxSalary(pr.Amount, pr.ReceiverAddress, privateKey, stateDB, mdMaker(nil))
			if err != nil {
				Logger.Log.Errorf("Cannot build Tx Salary v1. Err: %v", err)
				return nil, err
			}
			// handle for "return staking" TX; need to set the transaction type
			if pr.Type==common.TxReturnStakingType{
				temp.Type = common.TxReturnStakingType
			}
			res = temp
		}else{
			shardID := common.GetShardIDFromLastByte(pr.ReceiverAddress.Pk[len(pr.ReceiverAddress.Pk)-1])
			tokenParams := &tx_generic.TokenParam{
				PropertyID:     pr.TokenID.String(),
				PropertyName:   "",
				PropertySymbol: "",
				Amount:         pr.Amount,
				TokenTxType:    CustomTokenInit,
				Receiver:       []*privacy.PaymentInfo{&privacy.PaymentInfo{PaymentAddress: *pr.ReceiverAddress, Amount: pr.Amount}},
				TokenInput:     []privacy.PlainCoin{},
				Mintable:       true,
			}
			temp := new(TxTokenVersion1)
			err := temp.Init(
				tx_generic.NewTxTokenParams(privateKey,
					[]*privacy.PaymentInfo{},
					nil,
					0,
					tokenParams,
					stateDB,
					mdMaker(nil),
					false,
					false,
					shardID,
					[]byte{},
					nil,
				))
			if err != nil {
				Logger.Log.Errorf("Cannot build Tx Token Salary v1. Err: %v", err)
				return nil, err
			}
			res = temp
		}
	case 2:
		otaCoin, err := pr.generateOutputCoin()
		if err != nil {
			Logger.Log.Errorf("Cannot create coin for TX salary. Err: %v", err)
			return nil, err
		}
		md := mdMaker(otaCoin)
		
		if isPRV{
			temp := new(TxVersion2)
			err = temp.InitTxSalary(otaCoin, privateKey, stateDB, md)
			if err != nil {
				Logger.Log.Errorf("Cannot build Tx Salary v2. Err: %v", err)
				return nil, err
			}
			// handle for "return staking" TX; need to set the transaction type
			if pr.Type==common.TxReturnStakingType{
				temp.Type = common.TxReturnStakingType
				temp.Sig = nil
				temp.SigPubKey = nil
				if temp.Sig, temp.SigPubKey, err = tx_generic.SignNoPrivacy(privateKey, temp.Hash()[:]); err != nil {
					return nil, utils.NewTransactionErr(utils.SignTxError, err)
				}
				// valid, err = temp.ValidateTxSalary(stateDB)
				// Logger.Log.Debugf("Verify Salary TX for return staking : %v, %v", valid, err)
			}
			res = temp
		}else{
			temp := new(TxTokenVersion2)
			err = temp.InitTxTokenSalary(otaCoin, privateKey, stateDB, md, pr.TokenID, "")
			if err != nil {
				Logger.Log.Errorf("Cannot build Tx Token Salary v2. Err: %v", err)
				return nil, err
			}
			res = temp
		}
	default:
		Logger.Log.Errorf("Cannot build Tx Salary - invalid parameters %v", pr)
		return nil, errors.New("Cannot build Tx Salary - invalid parameters")
	}
	jsb, _ := json.MarshalIndent(res, "", "\t")
	Logger.Log.Infof("Built new salary transaction %s", string(jsb))
	return res, nil
}

// Used to parse json
type txJsonDataVersion struct {
	Version int8 `json:"Version"`
	Type    string
}

// For PRV and the Fee inside TokenTx
func NewTransactionFromJsonBytes(data []byte) (metadata.Transaction, error) {
	choices, err := DeserializeTransactionJSON(data)
	if err!=nil{
		return nil, err
	}
	if choices.Version1!=nil{
		return choices.Version1, nil
	}
	if choices.Version2!=nil{
		return choices.Version2, nil
	}
	return nil, errors.New("Cannot parse TX as PRV transaction")
	// txJsonVersion := new(txJsonDataVersion)
	// if err := json.Unmarshal(data, txJsonVersion); err != nil {
	// 	return nil, err
	// }
	// if txJsonVersion.Type == common.TxConversionType || txJsonVersion.Type == common.TxTokenConversionType {
	// 	if txJsonVersion.Version == int8(utils.TxConversionNumber) {
	// 			tx := new(TxVersion2)
	// 			if err := json.Unmarshal(data, tx); err != nil {
	// 				return nil, err
	// 			}
	// 			return tx, nil
	// 	} else {
	// 		return nil, errors.New("Cannot new txConversion from jsonBytes, type is incorrect.")
	// 	}
	// } else {
	// 	switch txJsonVersion.Version {
	// 	case int8(utils.TxVersion1Number), int8(utils.TxVersion0Number):
	// 		tx := new(TxVersion1)
	// 		if err := json.Unmarshal(data, tx); err != nil {
	// 			return nil, err
	// 		}
	// 		return tx, nil
	// 	case int8(utils.TxVersion2Number):
	// 		tx := new(TxVersion2)
	// 		if err := json.Unmarshal(data, tx); err != nil {
	// 			return nil, err
	// 		}
	// 		return tx, nil
	// 	default:
	// 		return nil, errors.New("Cannot new tx from jsonBytes, version is incorrect")
	// 	}
	// }
}

// Return token transaction from bytes
func NewTransactionTokenFromJsonBytes(data []byte) (tx_generic.TransactionToken, error) {
	choices, err := DeserializeTransactionJSON(data)
	if err!=nil{
		return nil, err
	}
	if choices.TokenVersion1!=nil{
		return choices.TokenVersion1, nil
	}
	if choices.TokenVersion2!=nil{
		return choices.TokenVersion2, nil
	}
	return nil, errors.New("Cannot parse TX as token transaction")
	// txJsonVersion := new(txJsonDataVersion)
	// if err := json.Unmarshal(data, txJsonVersion); err != nil {
	// 	return nil, err
	// }

	// if txJsonVersion.Type == common.TxTokenConversionType {
	// 	if txJsonVersion.Version == utils.TxConversionNumber {
	// 		tx := new(TxTokenVersion2)
	// 		if err := json.Unmarshal(data, tx); err != nil {
	// 			return nil, err
	// 		}
	// 		return tx, nil
	// 	} else {
	// 		return nil, errors.New("Cannot new txTokenConversion from jsonBytes, version is incorrect")
	// 	}
	// } else {
	// 	switch txJsonVersion.Version {
	// 	case int8(utils.TxVersion1Number), utils.TxVersion0Number:
	// 		tx := new(TxTokenVersion1)
	// 		if err := json.Unmarshal(data, tx); err != nil {
	// 			return nil, err
	// 		}
	// 		return tx, nil
	// 	case int8(utils.TxVersion2Number):
	// 		tx := new(TxTokenVersion2)
	// 		if err := json.Unmarshal(data, tx); err != nil {
	// 			return nil, err
	// 		}
	// 		return tx, nil
	// 	default:
	// 		return nil, errors.New("Cannot new txToken from bytes because version is incorrect")
	// 	}
	// }
}

type TxChoice struct{
	Version1 		*TxVersion1 		`json:"TxVersion1,omitempty"`
	TokenVersion1 	*TxTokenVersion1 	`json:"TxTokenVersion1,omitempty"`
	Version2 		*TxVersion2 		`json:"TxVersion2,omitempty"`
	TokenVersion2 	*TxTokenVersion2 	`json:"TxTokenVersion2,omitempty"`
}
func (ch *TxChoice) ToTx() metadata.Transaction{
	if ch==nil{
		return nil
	}
	// `choice` struct only ever contains 1 non-nil field
	if ch.Version1!=nil{
		return ch.Version1
	}
	if ch.Version2!=nil{
		return ch.Version2
	}
	if ch.TokenVersion1!=nil{
		return ch.TokenVersion1
	}
	if ch.TokenVersion2!=nil{
		return ch.TokenVersion2
	}
	return nil
}
func DeserializeTransactionJSON(data []byte) (*TxChoice, error){
	result := &TxChoice{}
	holder := make(map[string]interface{})
	err := json.Unmarshal(data, &holder)
	if err!=nil{
		return nil, err
	}
	_, isTokenTx := holder["TxTokenPrivacyData"]
	_, hasVersionOutside := holder["Version"]
	var verHolder txJsonDataVersion
	json.Unmarshal(data, &verHolder)
	if hasVersionOutside {
		switch verHolder.Version{
		case utils.TxVersion1Number:
			if isTokenTx{
				// token ver 1
				result.TokenVersion1 = &TxTokenVersion1{}
				err := json.Unmarshal(data, result.TokenVersion1)
				return result, err
			}else{
				// tx ver 1
				result.Version1 = &TxVersion1{}
				err := json.Unmarshal(data, result.Version1)
				return result, err
			}
		case utils.TxVersion2Number: // the same as utils.TxConversionNumber
			if isTokenTx{
				// rejected
				return nil, errors.New("Error unmarshalling TX from JSON : misplaced version")
			}else{
				// tx ver 2
				result.Version2 = &TxVersion2{}
				err := json.Unmarshal(data, result.Version2)
				return result, err
			}
		default:
			return nil, errors.New(fmt.Sprintf("Error unmarshalling TX from JSON : wrong version of %d", verHolder.Version))
		}
	}else{
		if isTokenTx{
			// token ver 2
			result.TokenVersion2 = &TxTokenVersion2{}
			err := json.Unmarshal(data, result.TokenVersion2)
			return result, err
		}else{
			return nil, errors.New("Error unmarshalling TX from JSON")
		}
	}

}

type BuildCoinBaseTxByCoinIDParams struct {
	payToAddress       *privacy.PaymentAddress
	amount             uint64
	txRandom           *privacy.TxRandom
	payByPrivateKey    *privacy.PrivateKey
	transactionStateDB *statedb.StateDB
	bridgeStateDB      *statedb.StateDB
	meta               *metadata.WithDrawRewardResponse
	coinID             common.Hash
	txType             int
	coinName           string
	shardID            byte
}

func NewBuildCoinBaseTxByCoinIDParams(payToAddress *privacy.PaymentAddress,
	amount uint64,
	payByPrivateKey *privacy.PrivateKey,
	stateDB *statedb.StateDB,
	meta *metadata.WithDrawRewardResponse,
	coinID common.Hash,
	txType int,
	coinName string,
	shardID byte,
	bridgeStateDB *statedb.StateDB) *BuildCoinBaseTxByCoinIDParams {
	params := &BuildCoinBaseTxByCoinIDParams{
		transactionStateDB: stateDB,
		bridgeStateDB:      bridgeStateDB,
		shardID:            shardID,
		meta:               meta,
		amount:             amount,
		coinID:             coinID,
		coinName:           coinName,
		payByPrivateKey:    payByPrivateKey,
		payToAddress:       payToAddress,
		txType:             txType,
	}
	return params
}

func NewTransactionFromParams(params *tx_generic.TxPrivacyInitParams) (metadata.Transaction, error) {
	inputCoins := params.InputCoins
	ver, err := tx_generic.GetTxVersionFromCoins(inputCoins)
	if err != nil {
		return nil, err
	}

	if ver == 1 {
		return new(TxVersion1), nil
	} else if ver == 2 {
		return new(TxVersion2), nil
	}
	return nil, errors.New("Cannot create new transaction from params, ver is wrong")
}

func NewTransactionTokenFromParams(params *tx_generic.TxTokenParams) (tx_generic.TransactionToken, error) {
	inputCoins := params.InputCoin
	ver, err := tx_generic.GetTxVersionFromCoins(inputCoins)
	if err != nil {
		return nil, err
	}

	fmt.Println("BUGLOG2 coin version", ver)

	if ver == 1 {
		return new(TxTokenVersion1), nil
	} else if ver == 2 {
		return new(TxTokenVersion2), nil
	}
	return nil, errors.New("Something is wrong when NewTransactionFromParams")
}

func GetTxTokenDataFromTransaction(tx metadata.Transaction) *tx_generic.TxTokenData {
	if tx.GetType() != common.TxCustomTokenPrivacyType {
		return nil
	}
	switch tx_specific := tx.(type) {
	case *TxTokenVersion1:
		// txTemp := tx.(*TxTokenVersion1)
		return &tx_specific.TxTokenData
	// } else if tx.GetVersion() == utils.TxVersion2Number || tx.GetVersion() == utils.TxConversionNumber {
	case *TxTokenVersion2:
		// txTemp := tx.(*TxTokenVersion2)
		res := tx_specific.GetTxTokenData()
		return &res
	default:
		return nil
	}
}

func GetFullBurnData(tx metadata.Transaction) (bool, coin.Coin, coin.Coin, *common.Hash, error) {

	switch  tx.GetType() {
	case common.TxNormalType:
		isBurned, burnPrv, _, err :=  tx.GetTxBurnData()
		if err != nil || isBurned == false {
			return false, nil, nil, nil, err
		}
		return true, burnPrv, nil, nil, err
	case common.TxCustomTokenPrivacyType:
		if txTmp, ok := tx.(TransactionToken); !ok {
			return false, nil, nil, nil, fmt.Errorf("tx is not tp")
		} else {
			isBurnToken, burnToken, burnedTokenID, err1 :=  txTmp.GetTxBurnData()
			isBurnPrv, burnPrv, _, err2 := txTmp.GetTxBase().GetTxBurnData()

			if err1 != nil && err2 != nil {
				return false, nil, nil, nil, fmt.Errorf("%v and %v", err1, err2)
			}

			return isBurnPrv || isBurnToken, burnToken, burnPrv, burnedTokenID, nil
		}

	default:
		return false, nil, nil, nil, nil
	}
}

// func (txToken *tx_generic.TxTokenBase) UnmarshalJSON(data []byte) error {
// 	var err error
// 	if txToken.Tx, err = NewTransactionFromJsonBytes(data); err != nil {
// 		return err
// 	}
// 	temp := &struct {
// 		TxTokenData tx_generic.TxTokenData `json:"TxTokenPrivacyData"`
// 	}{}
// 	err = json.Unmarshal(data, &temp)
// 	if err != nil {
// 		Logger.Log.Error(err)
// 		return NewTransactionErr(PrivacyTokenJsonError, err)
// 	}
// 	TxTokenDataJson, err := json.MarshalIndent(temp.tx_generic.TxTokenData, "", "\t")
// 	if err != nil {
// 		Logger.Log.Error(err)
// 		return NewTransactionErr(UnexpectedError, err)
// 	}
// 	err = json.Unmarshal(TxTokenDataJson, &txToken.tx_generic.TxTokenData)
// 	if err != nil {
// 		Logger.Log.Error(err)
// 		return NewTransactionErr(PrivacyTokenJsonError, err)
// 	}

// 	// TODO: hotfix, remove when fixed this issue
// 	if txToken.Tx.GetMetadata() != nil && txToken.Tx.GetMetadata().GetType() == 81 {
// 		if txToken.tx_generic.TxTokenData.Amount == 37772966455153490 {
// 			txToken.tx_generic.TxTokenData.Amount = 37772966455153487
// 		}
// 	}
// 	return nil
// }

// func (txData *tx_generic.TxTokenData) UnmarshalJSON(data []byte) error {
// 	type Alias tx_generic.TxTokenData
// 	temp := &struct {
// 		TxNormal json.RawMessage
// 		*Alias
// 	}{
// 		Alias: (*Alias)(txData),
// 	}
// 	err := json.Unmarshal(data, temp)
// 	if err != nil {
// 		Logger.Log.Error("UnmarshalJSON tx", string(data))
// 		return utils.NewTransactionErr(utils.UnexpectedError, err)
// 	}

// 	if txData.TxNormal, err = NewTransactionFromJsonBytes(temp.TxNormal); err != nil {
// 		Logger.Log.Error(err)
// 		return err
// 	}
// 	return nil
// }