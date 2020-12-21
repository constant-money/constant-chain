package internal

import(
	// "encoding/base64"
	// "encoding/json"
	// "math/big"
	// "fmt"
	// "strconv"
	"time"

	// "github.com/pkg/errors"
	"github.com/incognitochain/incognito-chain/common"
	// "github.com/incognitochain/incognito-chain/common/base58"
	// "github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/schnorr"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/serialnumbernoprivacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2"
	// "github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/internal/metadata"
	// "github.com/incognitochain/incognito-chain/wallet"

)

func SignNoPrivacy(privKey *privacy.PrivateKey, hashedMessage []byte) (signatureBytes []byte, sigPubKey []byte, err error) {
	sk := new(privacy.Scalar).FromBytesS(*privKey)
	r := new(privacy.Scalar).FromUint64(0)
	sigKey := new(schnorr.SchnorrPrivateKey)
	sigKey.Set(sk, r)
	signature, err := sigKey.Sign(hashedMessage)
	if err != nil {
		return nil, nil, err
	}
	signatureBytes = signature.Bytes()
	sigPubKey = sigKey.GetPublicKey().GetPublicKey().ToBytesS()
	return signatureBytes, sigPubKey, nil
}

func initializeTxConversion(tx *Tx, params *TxPrivacyInitParams, paymentsPtr *[]printedPaymentInfo) error {
	var err error
	// Get Keyset from param
	skBytes := *params.SenderSK
	senderPaymentAddress := privacy.GeneratePaymentAddress(skBytes)
	// Tx: initialize some values
	tx.Fee = params.Fee
	tx.Version = 2
	tx.Type = common.TxConversionType
	tx.PubKeyLastByteSender = senderPaymentAddress.Pk[len(senderPaymentAddress.Pk)-1]
	tx.LockTime = time.Now().Unix()
	tx.Info = params.Info
	// Params: update balance if overbalance
	if err = updateParamsWhenOverBalance(paymentsPtr, params, senderPaymentAddress); err != nil {
		return err
	}
	return nil
}

func getOutputcoinsFromPaymentInfo(paymentInfos []*privacy.PaymentInfo, tokenID *common.Hash) ([]*privacy.CoinV2, error) {
	var err error
	isPRV := (tokenID==nil) || (*tokenID==common.PRVCoinID)
	c := make([]*privacy.CoinV2, len(paymentInfos))
	// println("token is", tokenID.String())
	for i := 0; i < len(paymentInfos); i += 1 {
		if isPRV{
			c[i], err = privacy.NewCoinFromPaymentInfo(paymentInfos[i])
			if err != nil {
				return nil, err
			}
		}else{
			createdCACoin, _, err := privacy.NewCoinCA(paymentInfos[i], tokenID)
			if err!=nil{
				return nil, err
			}
			createdCACoin.SetPlainTokenID(tokenID)
			c[i] = createdCACoin
		}
	}
	return c, nil
}

func proveConversionAsm(tx *Tx, params *TxPrivacyInitParams) error {
	lenInputs := len(params.InputCoins)
	inputCoins := params.InputCoins
	var err error
	outputCoins, err := getOutputcoinsFromPaymentInfo(params.PaymentInfo, params.TokenID)
	if err != nil {
		return err
	}
	serialnumberWitness := make([]*serialnumbernoprivacy.SNNoPrivacyWitness, lenInputs)
	for i := 0; i < len(inputCoins); i++ {
		/***** Build witness for proving that serial number is derived from the committed derivator *****/
		serialnumberWitness[i] = new(serialnumbernoprivacy.SNNoPrivacyWitness)
		serialnumberWitness[i].Set(inputCoins[i].GetKeyImage(), inputCoins[i].GetPublicKey(),
			inputCoins[i].GetSNDerivator(), new(privacy.Scalar).FromBytesS(*params.SenderSK))
	}
	tx.Proof, err = privacy_v2.ProveConversion(inputCoins, outputCoins, serialnumberWitness)
	if err != nil {
		return err
	}
	if tx.Sig, tx.SigPubKey, err = SignNoPrivacy(params.SenderSK, tx.Hash()[:]); err != nil {
		return err
	}
	// println("After creation, converted output coin is")
	// ci := GetCoinInter(outputCoins[0])
	// jsb, _ := json.Marshal(ci)
	// println(string(jsb))
	return nil
}

func InitConversionASM(tx *Tx, params *InitParamsAsm) error {
	gParams := params.GetGenericParams()
	if err := initializeTxConversion(tx, gParams, &params.PaymentInfo); err != nil {
		return err
	}
	if err := proveConversionAsm(tx, gParams); err != nil {
		return err
	}
	// jsb, _ := json.Marshal(tx)
	// utils.Logger.Log.Infof("Init conversion complete ! -> %s", string(jsb))

	return nil
}

func (txToken *TxToken) initTokenConversion(txNormal *Tx, params *InitParamsAsm) error {
	txToken.TokenData.Type = CustomTokenTransfer
	txToken.TokenData.PropertyName = ""
	txToken.TokenData.PropertySymbol = ""
	txToken.TokenData.Mintable = false
	propertyID, _ := common.TokenStringToHash(params.TokenParams.TokenID)
	txToken.TokenData.PropertyID = *propertyID
	temp := params.TokenParams.GetCompatTokenParams()
	txConvertParams := NewTxParams(
		&params.SenderSK,
		temp.Receiver,
		temp.TokenInput,
		0,
		false,
		propertyID,
		nil,
		params.Info,
	)
	
	if err := initializeTxConversion(txNormal, txConvertParams, &params.TokenParams.TokenPaymentInfo); err != nil {
		return err
	}
	txNormal.Type = TxTokenConversionType
	if err := proveConversionAsm(txNormal, txConvertParams); err != nil {
		return err
	}
	// if err := InitConversion(txNormal, txConvertParams); err != nil {
	// 	return utils.NewTransactionErr(utils.PrivacyTokenInitTokenDataError, err)
	// }
	err := txToken.SetTxNormal(txNormal)
	return err
}

func (txToken *TxToken) initPRVFeeConversion(feeTx *Tx, params *InitParamsAsm) ([]privacy.PlainCoin, []uint64, []*privacy.CoinV2, error) {
	feeTx.Version = 2
	feeTx.Type = common.TxTokenConversionType
	inps, inputIndexes, outs, err := feeTx.provePRV(params)
	if err != nil {
		return nil, nil, nil, err
	}

	return inps, inputIndexes, outs, nil
}

func InitTokenConversionASM(txToken *TxToken, params *InitParamsAsm) error {
	params.HasPrivacy = false
	txPrivacyParams := params.GetGenericParams()
	// Init tx and params (tx and params will be changed)
	tx := &Tx{}
	if err := tx.initializeTxAndParams(txPrivacyParams, &params.PaymentInfo); err != nil {
		return err
	}
	// Init PRV Fee
	inps, inputIndexes, outs, err := txToken.initPRVFeeConversion(tx, params)
	if err != nil {
		return err
	}
	txn := makeTxToken(tx, nil, nil, nil)
	// Init Token
	if err := txToken.initTokenConversion(txn, params); err != nil {
		return err
	}
	tdh, err := txToken.TokenData.Hash()
	if err!=nil{
		return err
	}
	message := common.HashH(append(tx.Hash()[:], tdh[:]...))
	err = tx.sign(inps, inputIndexes, outs, params, message[:])
	if err!=nil{
		return err
	}
	txToken.Tx = *tx
	return nil
}