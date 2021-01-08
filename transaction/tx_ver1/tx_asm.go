package tx_ver1

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/wallet"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
)

type TxPrivacyInitParamsForASM struct {
	txParam             tx_generic.TxPrivacyInitParams
	commitmentIndices   []uint64
	commitmentBytes     [][]byte
	myCommitmentIndices []uint64
	sndOutputs          []*privacy.Scalar
}

func NewTxPrivacyInitParamsForASM(
	senderSK *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoins []privacy.PlainCoin,
	fee uint64,
	hasPrivacy bool,
	tokenID *common.Hash, // default is nil -> use for prv coin
	metaData metadata.Metadata,
	info []byte,
	commitmentIndices []uint64,
	commitmentBytes [][]byte,
	myCommitmentIndices []uint64,
	sndOutputs []*privacy.Scalar) *TxPrivacyInitParamsForASM {

	txParam := tx_generic.TxPrivacyInitParams{
		SenderSK:    senderSK,
		PaymentInfo: paymentInfo,
		InputCoins:  inputCoins,
		Fee:         fee,
		HasPrivacy:  hasPrivacy,
		TokenID:     tokenID,
		MetaData:    metaData,
		Info:        info,
	}
	return &TxPrivacyInitParamsForASM{
		txParam:             txParam,
		commitmentIndices:   commitmentIndices,
		commitmentBytes:     commitmentBytes,
		myCommitmentIndices: myCommitmentIndices,
		sndOutputs:          sndOutputs,
	}
}

func (param *TxPrivacyInitParamsForASM) SetMetaData(meta metadata.Metadata) {
	param.txParam.MetaData = meta
}

// return bool indicates that after initialization, should we continue the function "Init" or not
func initializeTxAndParamsASM(tx *tx_generic.TxBase, params *TxPrivacyInitParamsForASM) error {
	txParams := &params.txParam
	err := tx.InitializeTxAndParams(txParams)
	if txParams.HasPrivacy {
		// Check number of list of random commitments, list of random commitment indices
		if len(params.commitmentIndices) != len(params.txParam.InputCoins)*privacy.CommitmentRingSize {
			return utils.NewTransactionErr(utils.RandomCommitmentError, nil)
		}

		if len(params.myCommitmentIndices) != len(params.txParam.InputCoins) {
			return utils.NewTransactionErr(utils.RandomCommitmentError, errors.New("number of list my commitment indices must be equal to number of input coins"))
		}
	}
	return err
}

type TxPrivacyTokenInitParamsForASM struct {
	//senderKey       *privacy.PrivateKey
	//tokenPayments     []*privacy.PaymentInfo
	//inputCoin       []*privacy.PlainCoinV1
	//feeNativeCoin   uint64
	//tokenParams     *TokenParam
	//transactionStateDB              database.DatabaseInterface
	//metaData        metadata.Metadata
	//hasPrivacyCoin  bool
	//hasPrivacyToken bool
	//shardID         byte
	//info            []byte

	txParam                           tx_generic.TxTokenParams
	commitmentIndicesForNativeToken   []uint64
	commitmentBytesForNativeToken     [][]byte
	myCommitmentIndicesForNativeToken []uint64
	sndOutputsForNativeToken          []*privacy.Scalar

	commitmentIndicesForPToken   []uint64
	commitmentBytesForPToken     [][]byte
	myCommitmentIndicesForPToken []uint64
	sndOutputsForPToken          []*privacy.Scalar
}

func (param *TxPrivacyTokenInitParamsForASM) SetMetaData(meta metadata.Metadata) {
	param.txParam.MetaData = meta
}

func NewTxPrivacyTokenInitParamsForASM(
	senderKey *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoin []privacy.PlainCoin,
	feeNativeCoin uint64,
	tokenParams *tx_generic.TokenParam,
	metaData metadata.Metadata,
	hasPrivacyCoin bool,
	hasPrivacyToken bool,
	shardID byte,
	info []byte,
	commitmentIndicesForNativeToken []uint64,
	commitmentBytesForNativeToken [][]byte,
	myCommitmentIndicesForNativeToken []uint64,
	sndOutputsForNativeToken []*privacy.Scalar,

	commitmentIndicesForPToken []uint64,
	commitmentBytesForPToken [][]byte,
	myCommitmentIndicesForPToken []uint64,
	sndOutputsForPToken []*privacy.Scalar) *TxPrivacyTokenInitParamsForASM {

	txParam := tx_generic.NewTxTokenParams(senderKey, paymentInfo, inputCoin, feeNativeCoin, tokenParams, nil, metaData, hasPrivacyCoin, hasPrivacyToken, shardID, info, nil)
	params := &TxPrivacyTokenInitParamsForASM{
		txParam:                           *txParam,
		commitmentIndicesForNativeToken:   commitmentIndicesForNativeToken,
		commitmentBytesForNativeToken:     commitmentBytesForNativeToken,
		myCommitmentIndicesForNativeToken: myCommitmentIndicesForNativeToken,
		sndOutputsForNativeToken:          sndOutputsForNativeToken,

		commitmentIndicesForPToken:   commitmentIndicesForPToken,
		commitmentBytesForPToken:     commitmentBytesForPToken,
		myCommitmentIndicesForPToken: myCommitmentIndicesForPToken,
		sndOutputsForPToken:          sndOutputsForPToken,
	}
	return params
}


// TODO PRIVACY, WILL DO THIS LATER BECAUSE IT IS ASM
func (tx *Tx) InitForASM(params *TxPrivacyInitParamsForASM) error {
	utils.Logger.Log.Debugf("CREATING TX........\n")
	txParams := &params.txParam
	if err := tx_generic.ValidateTxParams(txParams); err != nil {
		return err
	}
	if err := initializeTxAndParamsASM(&tx.TxBase, params); err != nil {
		return err
	}

	// Check if this tx is nonPrivacyNonInput
	// Case 1: tx ptoken transfer with ptoken fee
	// Case 2: tx Reward
	if check, err := tx.IsNonPrivacyNonInput(txParams); check {
		return err
	}

	//metaTx, err := NewTransactionFromTxBase(*tx)
	//if err := tx.ProveASM(tx, params); err != nil {
	//	return err
	//}

	return nil
}

// TODO Privacy, WILL DO THIS LATER BECAUSE IT IS ASM
// Init -  build normal tx component and privacy custom token data
func (txToken *TxToken) InitForASM(params *TxPrivacyTokenInitParamsForASM, serverTime int64) error {
	//var err error
	//initTokenParamsASM := NewTxPrivacyInitParamsForASM(
	//	params.txParam.senderKey,
	//	params.txParam.tokenPayments,
	//	params.txParam.inputCoin,
	//	params.txParam.feeNativeCoin,
	//	params.txParam.hasPrivacyCoin,
	//	nil,
	//	params.txParam.metaData,
	//	params.txParam.info,
	//	params.commitmentIndicesForNativeToken,
	//	params.commitmentBytesForNativeToken,
	//	params.myCommitmentIndicesForNativeToken,
	//	params.sndOutputsForNativeToken,
	//)
	//normalTx :=
	//err = normalTx.InitForASM()
	//if err != nil {
	//	return utils.NewTransactionErr(utils.PrivacyTokenInitPRVError, err)
	//}
	//
	//// override TxCustomTokenPrivacyType type
	//normalTx.Type = common.TxCustomTokenPrivacyType
	//txToken.Tx = normalTx
	//
	//// check action type and create privacy custom toke data
	//var handled = false
	//// Add token data component
	//switch params.txParam.tokenParams.TokenTxType {
	//case CustomTokenInit:
	//	// case init a new privacy custom token
	//	{
	//		handled = true
	//		txToken.TxPrivacyTokenDataVersion1 = TxPrivacyTokenDataVersion1{
	//			Type:           params.txParam.tokenParams.TokenTxType,
	//			PropertyName:   params.txParam.tokenParams.PropertyName,
	//			PropertySymbol: params.txParam.tokenParams.PropertySymbol,
	//			Amount:         params.txParam.tokenParams.Amount,
	//		}
	//
	//		// issue token with data of privacy
	//		temp := Tx{}
	//		temp.Type = common.TxNormalType
	//		temp.Proof = privacy.NewProofWithVersion(txToken.Version)
	//		temp.Proof.Init()
	//		tempOutputCoin := make([]privacy.Coin, 1)
	//		c := new(privacy.CoinV1)
	//		c.CoinDetails.SetValue(params.txParam.tokenParams.Amount)
	//		PK, err := new(privacy.Point).FromBytesS(params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk)
	//		if err != nil {
	//			return utils.NewTransactionErr(utils.DecompressPaymentAddressError, err)
	//		}
	//		c.CoinDetails.SetPublicKey(PK)
	//		c.CoinDetails.SetRandomness(privacy.RandomScalar())
	//
	//		// set info coin for output coin
	//		if len(params.txParam.tokenParams.Receiver[0].Message) > 0 {
	//			if len(params.txParam.tokenParams.Receiver[0].Message) > privacy.MaxSizeInfoCoin {
	//				return utils.NewTransactionErr(utils.ExceedSizeInfoOutCoinError, nil)
	//			}
	//			c.CoinDetails.SetInfo(params.txParam.tokenParams.Receiver[0].Message)
	//		}
	//		sndOut := privacy.RandomScalar()
	//		c.CoinDetails.SetSNDerivator(sndOut)
	//		err = c.CoinDetails.CommitAll()
	//		if err != nil {
	//			return utils.NewTransactionErr(utils.CommitOutputCoinError, err)
	//		}
	//
	//		tempOutputCoin[0] = c
	//		if err = temp.Proof.SetOutputCoins(tempOutputCoin); err != nil {
	//			utils.Logger.Log.Errorf("TxPrivacyToken InitASM cannot set output coins: err %v", err)
	//			return err
	//		}
	//
	//		// get last byte
	//		temp.PubKeyLastByteSender = params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk[len(params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk)-1]
	//
	//		// sign Tx
	//		temp.SigPubKey = params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk
	//		temp.sigPrivKey = *params.txParam.senderKey
	//		err = signTx(&temp)
	//		if err != nil {
	//			utils.Logger.Log.Error(errors.New("can't sign this tx"))
	//			return utils.NewTransactionErr(utils.SignTxError, err)
	//		}
	//
	//		txToken.TxPrivacyTokenDataVersion1.TxNormal = temp
	//		hashInitToken, err := txToken.TxPrivacyTokenDataVersion1.Hash()
	//		if err != nil {
	//			utils.Logger.Log.Error(errors.New("can't hash this token data"))
	//			return utils.NewTransactionErr(utils.UnexpectedError, err)
	//		}
	//
	//		if params.txParam.tokenParams.Mintable {
	//			propertyID, err := common.Hash{}.NewHashFromStr(params.txParam.tokenParams.PropertyID)
	//			if err != nil {
	//				return utils.NewTransactionErr(utils.TokenIDInvalidError, err, propertyID.String())
	//			}
	//			txToken.TxPrivacyTokenDataVersion1.PropertyID = *propertyID
	//			txToken.TxPrivacyTokenDataVersion1.Mintable = true
	//		} else {
	//			//NOTICE: @merman update PropertyID calculated from hash of tokendata and shardID
	//			newHashInitToken := common.HashH(append(hashInitToken.GetBytes(), params.txParam.shardID))
	//			txToken.TxPrivacyTokenDataVersion1.PropertyID = newHashInitToken
	//		}
	//	}
	//case CustomTokenTransfer:
	//	{
	//		handled = true
	//		// make a transfering for privacy custom token
	//		// fee always 0 and reuse function of normal tx for custom token ID
	//		temp := Tx{}
	//		propertyID, _ := common.Hash{}.NewHashFromStr(params.txParam.tokenParams.PropertyID)
	//		txToken.TxPrivacyTokenDataVersion1 = TxPrivacyTokenDataVersion1{
	//			Type:           params.txParam.tokenParams.TokenTxType,
	//			PropertyName:   params.txParam.tokenParams.PropertyName,
	//			PropertySymbol: params.txParam.tokenParams.PropertySymbol,
	//			PropertyID:     *propertyID,
	//			Mintable:       params.txParam.tokenParams.Mintable,
	//		}
	//		//err := temp.InitForASM(NewTxPrivacyInitParamsForASM(
	//		//	params.txParam.senderKey,
	//		//	params.txParam.tokenParams.Receiver,
	//		//	params.txParam.tokenParams.TokenInput,
	//		//	params.txParam.tokenParams.Fee,
	//		//	params.txParam.hasPrivacyToken,
	//		//	propertyID,
	//		//	nil,
	//		//	params.txParam.info,
	//		//	params.commitmentIndicesForPToken,
	//		//	params.commitmentBytesForPToken,
	//		//	params.myCommitmentIndicesForPToken,
	//		//	params.sndOutputsForPToken,
	//		//), serverTime)
	//		err := temp.InitForASM(NewTxPrivacyInitParamsForASM(
	//			params.txParam.senderKey,
	//			params.txParam.tokenParams.Receiver,
	//			params.txParam.tokenParams.TokenInput,
	//			params.txParam.tokenParams.Fee,
	//			params.txParam.hasPrivacyToken,
	//			propertyID,
	//			nil,
	//			params.txParam.info,
	//			params.commitmentIndicesForPToken,
	//			params.commitmentBytesForPToken,
	//			params.myCommitmentIndicesForPToken,
	//			params.sndOutputsForPToken,
	//		))
	//		if err != nil {
	//			return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
	//		}
	//		txToken.TxPrivacyTokenDataVersion1.TxNormal = temp
	//	}
	//}
	//
	//if !handled {
	//	return utils.NewTransactionErr(utils.PrivacyTokenTxTypeNotHandleError, errors.New("can't handle this TokenTxType"))
	//}
	return nil
}


// CreateCustomTokenReceiverArray - parse data frm rpc request to create a list vout for preparing to create a custom token tx
// data interface is a map[paymentt-address]{transferring-amount}
func CreateCustomTokenPrivacyReceiverArrayV2(dataReceiver interface{}) ([]*privacy.PaymentInfo, int64, error) {
	if dataReceiver == nil {
		return nil, 0, fmt.Errorf("data receiver is in valid")
	}
	result := []*privacy.PaymentInfo{}
	voutsAmount := int64(0)
	receivers, ok := dataReceiver.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("data receiver is in valid")
	}
	for key, value := range receivers {
		keyWallet, err := wallet.Base58CheckDeserialize(key)
		if err != nil {
			utils.Logger.Log.Errorf("Invalid key in CreateCustomTokenPrivacyReceiverArray %+v", key)
			return nil, 0, err
		}
		keySet := keyWallet.KeySet

		amount, err := common.AssertAndConvertStrToNumber(value)
		if err != nil {
			return nil, 0, err
		}

		temp := &privacy.PaymentInfo{
			PaymentAddress: keySet.PaymentAddress,
			Amount:         amount,
		}
		result = append(result, temp)
		voutsAmount += int64(temp.Amount)
	}
	return result, voutsAmount, nil
}
