package tx_ver1

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"strconv"
	"time"
)

//HELPER FUNCTIONS
func validateTxConversionInitParam(params *tx_generic.TxPrivacyInitParams) error {
	if len(params.InputCoins) > 255 {
		return utils.NewTransactionErr(utils.InputCoinIsVeryLargeError, nil, strconv.Itoa(len(params.InputCoins)))
	}
	if len(params.PaymentInfo) != 1 {
		return utils.NewTransactionErr(utils.UnexpectedError, fmt.Errorf("txConvert must only have 1 output"))
	}

	sumInput := uint64(0)
	for _, c := range params.InputCoins {
		if c.GetVersion() != 1 {
			err := fmt.Errorf("TxConversion should only have inputCoins version 1")
			return utils.NewTransactionErr(utils.InvalidInputCoinVersionErr, err)
		}

		sumInput += c.GetValue()
	}

	sumOutput := params.PaymentInfo[0].Amount
	if sumInput != sumOutput + params.Fee {
		err := fmt.Errorf("TxConversion's sum input coin and output coin (with fee) is not the same")
		return utils.NewTransactionErr(utils.SumInputCoinsAndOutputCoinsError, err)
	}

	if params.TokenID == nil {
		// using default PRV
		params.TokenID = &common.Hash{}
		if err := params.TokenID.SetBytes(common.PRVCoinID[:]); err != nil {
			return utils.NewTransactionErr(utils.TokenIDInvalidError, err, params.TokenID.String())
		}
	}

	if params.MetaData != nil && params.MetaData.GetType() != metadata.ConvertingRequestMeta {
		return utils.NewTransactionErr(utils.UnexpectedError, fmt.Errorf("invalid metadata type: want %v, have %v", metadata.ConvertingRequestMeta, params.MetaData))
	}
	return nil
}
func initTxConversion(tx *Tx, params *tx_generic.TxPrivacyInitParams) error {
	var err error

	senderKeySet :=  incognitokey.KeySet{}
	if err = senderKeySet.InitFromPrivateKey(params.SenderSK); err != nil {
		utils.Logger.Log.Errorf("cannot parse private key: %v\n", err)
		return utils.NewTransactionErr(utils.PrivateKeySenderInvalidError, err)
	}

	tx.Fee = params.Fee
	tx.Version = utils.TxVersion1Number
	tx.Type = common.TxNormalType
	tx.Metadata = params.MetaData
	tx.PubKeyLastByteSender = common.GetShardIDFromLastByte(senderKeySet.PaymentAddress.Pk[len(senderKeySet.PaymentAddress.Pk)-1])

	if tx.LockTime == 0 {
		tx.LockTime = time.Now().Unix()
	}
	if tx.Info, err = tx_generic.GetTxInfo(params.Info); err != nil {
		return err
	}
	return nil
}
func initConversionWitness(params *tx_generic.TxPrivacyInitParams, shardID byte) (*privacy.ConversionWitness, error) {
	comIndices, myComIndices, err := parseCommitments(params, shardID)
	if err != nil {
		return nil, err
	}
	comProving, err := parseCommitmentProving(params, shardID, comIndices)
	if err != nil {
		return nil, err
	}
	outputCoins, err := parseOutputCoins(params.PaymentInfo, params.TokenID, params.StateDB)
	if err != nil {
		return nil, err
	}

	inCoinV1s := make([]*privacy.PlainCoinV1, 0)
	for _, inCoin := range params.InputCoins {
		tmpInCoin, ok := inCoin.(*privacy.PlainCoinV1)
		if !ok {
			return nil, fmt.Errorf("TxConvert input coins must be PlainCoinV1's")
		}
		inCoinV1s = append(inCoinV1s, tmpInCoin)
	}

	// prepare witness for proving
	conversionWitnessParam := privacy.ConversionWitnessParam{
		PrivateKey:              new(privacy.Scalar).FromBytesS(*params.SenderSK),
		InputCoins:              inCoinV1s,
		OutputCoins:             outputCoins,
		PublicKeyLastByteSender: shardID,
		Commitments:             comProving,
		CommitmentIndices:       comIndices,
		MyCommitmentIndices:     myComIndices,
		Fee:                     params.Fee,
	}

	conversionWitness := new(privacy.ConversionWitness)
	err1 := conversionWitness.Init(conversionWitnessParam)
	if err1 != nil {
		return nil, fmt.Errorf("witness init returns an error: %v", err1.Error())
	}
	return conversionWitness, nil
}
func proveAndSignConversion(tx *Tx, params *tx_generic.TxPrivacyInitParams) error {
	shardID := common.GetShardIDFromLastByte(tx.PubKeyLastByteSender)

	witness, err := initConversionWitness(params, shardID)
	if err != nil {
		utils.Logger.Log.Errorf("cannot init conversion witness: %v\n", err)
		return utils.NewTransactionErr(utils.InitWithnessError, err)
	}

	conversionProof, err1 := witness.Prove()
	if err1 != nil {
		return utils.NewTransactionErr(utils.WithnessProveError, err1)
	}
	tx.Proof = conversionProof

	randSK := witness.GetRandSecretKey()
	tx.SetPrivateKey(append(*params.SenderSK, randSK.ToBytesS()...))

	// sign tx
	err = tx.sign()
	if err != nil {
		utils.Logger.Log.Error(err)
		return utils.NewTransactionErr(utils.SignTxError, err)
	}
	return nil
}
//END HELPER FUNCTIONS

//INIT FUNCTIONS
func (tx *Tx) InitConversion(params *tx_generic.TxPrivacyInitParams) error {
	var err error

	if err = validateTxConversionInitParam(params); err != nil {
		return err
	}

	if err = initTxConversion(tx, params); err != nil {
		return err
	}

	if err = proveAndSignConversion(tx, params); err != nil {
		return err
	}

	jsb, _ := json.Marshal(tx)
	utils.Logger.Log.Infof("Init conversion complete: %s", string(jsb))
	txSize := tx.GetTxActualSize()
	if txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}
	return nil
}

func (txToken *TxToken) InitTokenConversion(params *tx_generic.TxTokenParams) error {
	txFeeParam := tx_generic.NewTxPrivacyInitParams(
		params.SenderKey,
		nil,
		nil,
		0,
		true,
		params.TransactionStateDB,
		nil,
		params.MetaData,
		params.Info,
	)
	txToken.Tx = new(Tx)
	if err := txToken.Tx.Init(txFeeParam); err != nil {
		return utils.NewTransactionErr(utils.PrivacyTokenInitPRVError, err)
	}
	txToken.Tx.SetType(common.TxCustomTokenPrivacyType)

	txToken.TxTokenData.SetType(1)
	txToken.TxTokenData.SetPropertyName("")
	txToken.TxTokenData.SetPropertySymbol("")

	propertyID, _ := common.Hash{}.NewHashFromStr(params.TokenParams.PropertyID)
	existed := statedb.PrivacyTokenIDExisted(params.TransactionStateDB, *propertyID)
	if !existed {
		isBridgeToken := false
		allBridgeTokensBytes, err := statedb.GetAllBridgeTokens(params.BridgeStateDB)
		if err != nil {
			return utils.NewTransactionErr(utils.TokenIDExistedError, err)
		}
		if len(allBridgeTokensBytes) > 0 {
			var allBridgeTokens []*rawdbv2.BridgeTokenInfo
			err = json.Unmarshal(allBridgeTokensBytes, &allBridgeTokens)
			if err != nil {
				return utils.NewTransactionErr(utils.TokenIDExistedError, err)
			}
			for _, bridgeTokens := range allBridgeTokens {
				if propertyID.IsEqual(bridgeTokens.TokenID) {
					isBridgeToken = true
					break
				}
			}
		}
		if !isBridgeToken {
			return utils.NewTransactionErr(utils.TokenIDExistedError, fmt.Errorf("invalid Token ID"))
		}
	}

	txToken.TxTokenData.SetPropertyID(*propertyID)
	txToken.TxTokenData.SetMintable(false)

	txNormal := new(Tx)
	err := txNormal.InitConversion(tx_generic.NewTxPrivacyInitParams(params.SenderKey,
		params.TokenParams.Receiver,
		params.TokenParams.TokenInput,
		0,
		true,
		params.TransactionStateDB,
		propertyID,
		nil,
		nil))
	if err != nil {
		return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
	}

	txToken.TxTokenData.TxNormal = txNormal
	return nil
}
//END INIT FUNCTIONS

//VALIDATE FUNCTIONS
func ValidateConversionTransaction(tx Tx, db *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	jsb, _ := json.Marshal(tx)
	utils.Logger.Log.Infof("Begin verifying TX %s", string(jsb))

	var err error
	if valid, err := tx.verifySig(); !valid {
		if err != nil {
			utils.Logger.Log.Errorf("Error verifying signature ver1 with tx hash %s: %+v \n", tx.Hash().String(), err)
			return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
		}
		utils.Logger.Log.Errorf("FAILED VERIFICATION SIGNATURE ver1 with tx hash %s", tx.Hash().String())
		return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, fmt.Errorf("FAILED VERIFICATION SIGNATURE ver1 with tx hash %s", tx.Hash().String()))
	}

	tokenID, err = tx_generic.ParseTokenID(tokenID)
	if err != nil {
		return false, err
	}

	outputCoins := tx.Proof.GetOutputCoins()
	outputCoinsAsV1 := make([]*privacy.CoinV1, len(outputCoins))
	for i := 0; i < len(outputCoins); i += 1 {
		c, ok := outputCoins[i].(*privacy.CoinV1)
		if !ok{
			return false, utils.NewTransactionErr(utils.UnexpectedError, nil, fmt.Sprintf("Error when casting a coin to ver1"))
		}
		outputCoinsAsV1[i] = c
	}
	if err := validateSndFromOutputCoin(outputCoinsAsV1); err != nil {
		return false, err
	}

	commitments, err := getCommitmentsInDatabase(tx.Proof, db, shardID, tokenID)
	if err != nil {
		return false, err
	}

	if valid, err := tx.Proof.Verify(nil, tx.SigPubKey, tx.Fee, shardID, tokenID, commitments); !valid {
		if err != nil {
			utils.Logger.Log.Error(err)
		}
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, err, tx.Hash().String())
	}
	utils.Logger.Log.Debugf("SUCCESSED VERIFICATION PAYMENT PROOF ")
	return true, nil
}
//END VALIDATE FUNCTIONS