package transaction

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	zkp "github.com/incognitochain/incognito-chain/privacy/zeroknowledge"
	"github.com/incognitochain/incognito-chain/wallet"
)

type Tx struct {
	// Basic data, required
	Version  int8   `json:"Version"`
	Type     string `json:"Type"` // Transaction type
	LockTime int64  `json:"LockTime"`
	Fee      uint64 `json:"Fee"` // Fee applies: always consant
	Info     []byte // 512 bytes
	// Sign and Privacy proof, required
	SigPubKey            []byte `json:"SigPubKey, omitempty"` // 33 bytes
	Sig                  []byte `json:"Sig, omitempty"`       //
	Proof                *zkp.PaymentProof
	PubKeyLastByteSender byte
	// Metadata, optional
	Metadata metadata.Metadata
	// private field, not use for json parser, only use as temp variable
	sigPrivKey       []byte       // is ALWAYS private property of struct, if privacy: 64 bytes, and otherwise, 32 bytes
	cachedHash       *common.Hash // cached hash data of tx
	cachedActualSize *uint64      // cached actualsize data for tx
}

func (tx *Tx) UnmarshalJSON(data []byte) error {
	type Alias Tx
	temp := &struct {
		Metadata *json.RawMessage
		*Alias
	}{
		Alias: (*Alias)(tx),
	}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		Logger.log.Error("UnmarshalJSON tx", string(data))
		return NewTransactionErr(UnexpectedError, err)
	}
	if temp.Metadata == nil {
		tx.SetMetadata(nil)
		return nil
	}

	meta, parseErr := metadata.ParseMetadata(temp.Metadata)
	if parseErr != nil {
		Logger.log.Error(parseErr)
		return parseErr
	}
	tx.SetMetadata(meta)
	return nil
}

type TxPrivacyInitParams struct {
	senderSK    *privacy.PrivateKey
	paymentInfo []*privacy.PaymentInfo
	inputCoins  []*privacy.InputCoin
	fee         uint64
	hasPrivacy  bool
	stateDB     *statedb.StateDB
	tokenID     *common.Hash // default is nil -> use for prv coin
	metaData    metadata.Metadata
	info        []byte // 512 bytes
}

func NewTxPrivacyInitParams(senderSK *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoins []*privacy.InputCoin,
	fee uint64,
	hasPrivacy bool,
	stateDB *statedb.StateDB,
	tokenID *common.Hash, // default is nil -> use for prv coin
	metaData metadata.Metadata,
	info []byte) *TxPrivacyInitParams {
	params := &TxPrivacyInitParams{
		stateDB:     stateDB,
		tokenID:     tokenID,
		hasPrivacy:  hasPrivacy,
		inputCoins:  inputCoins,
		fee:         fee,
		metaData:    metaData,
		paymentInfo: paymentInfo,
		senderSK:    senderSK,
		info:        info,
	}
	return params
}

// Init - init value for tx from inputcoin(old output coin from old tx)
// create new outputcoin and build privacy proof
// if not want to create a privacy tx proof, set hashPrivacy = false
// database is used like an interface which use to query info from transactionStateDB in building tx
func (tx *Tx) Init(params *TxPrivacyInitParams) error {
	Logger.log.Debugf("CREATING TX........\n")
	tx.Version = txVersion
	var err error
	if len(params.inputCoins) > 255 {
		return NewTransactionErr(InputCoinIsVeryLargeError, nil, strconv.Itoa(len(params.inputCoins)))
	}
	if len(params.paymentInfo) > 254 {
		return NewTransactionErr(PaymentInfoIsVeryLargeError, nil, strconv.Itoa(len(params.paymentInfo)))
	}
	limitFee := uint64(0)
	estimateTxSizeParam := NewEstimateTxSizeParam(len(params.inputCoins), len(params.paymentInfo),
		params.hasPrivacy, nil, nil, limitFee)
	if txSize := EstimateTxSize(estimateTxSizeParam); txSize > common.MaxTxSize {
		return NewTransactionErr(ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}

	if params.tokenID == nil {
		// using default PRV
		params.tokenID = &common.Hash{}
		err := params.tokenID.SetBytes(common.PRVCoinID[:])
		if err != nil {
			return NewTransactionErr(TokenIDInvalidError, err, params.tokenID.String())
		}
	}

	// Calculate execution time
	start := time.Now()

	if tx.LockTime == 0 {
		tx.LockTime = time.Now().Unix()
	}

	// create sender's key set from sender's spending key
	senderFullKey := incognitokey.KeySet{}
	err = senderFullKey.InitFromPrivateKey(params.senderSK)
	if err != nil {
		Logger.log.Error(errors.New(fmt.Sprintf("Can not import Private key for sender keyset from %+v", params.senderSK)))
		return NewTransactionErr(PrivateKeySenderInvalidError, err)
	}
	// get public key last byte of sender
	pkLastByteSender := senderFullKey.PaymentAddress.Pk[len(senderFullKey.PaymentAddress.Pk)-1]

	// init info of tx
	tx.Info = []byte{}
	lenTxInfo := len(params.info)

	if lenTxInfo > 0 {
		if lenTxInfo > MaxSizeInfo {
			return NewTransactionErr(ExceedSizeInfoTxError, nil)
		}

		tx.Info = params.info
	}

	// set metadata
	tx.Metadata = params.metaData

	// set tx type
	tx.Type = common.TxNormalType
	Logger.log.Debugf("len(inputCoins), fee, hasPrivacy: %d, %d, %v\n", len(params.inputCoins), params.fee, params.hasPrivacy)

	if len(params.inputCoins) == 0 && params.fee == 0 && !params.hasPrivacy {
		Logger.log.Debugf("len(inputCoins) == 0 && fee == 0 && !hasPrivacy\n")
		tx.Fee = params.fee
		tx.sigPrivKey = *params.senderSK
		tx.PubKeyLastByteSender = pkLastByteSender
		err := tx.signTx()
		if err != nil {
			Logger.log.Error(errors.New(fmt.Sprintf("Cannot sign tx %v\n", err)))
			return NewTransactionErr(SignTxError, err)
		}
		return nil
	}

	shardID := common.GetShardIDFromLastByte(pkLastByteSender)
	var commitmentIndexs []uint64   // array index random of commitments in transactionStateDB
	var myCommitmentIndexs []uint64 // index in array index random of commitment in transactionStateDB

	if params.hasPrivacy {
		if len(params.inputCoins) == 0 {
			return NewTransactionErr(RandomCommitmentError, fmt.Errorf("input is empty"))
		}
		randomParams := NewRandomCommitmentsProcessParam(params.inputCoins, privacy.CommitmentRingSize, params.stateDB, shardID, params.tokenID)
		commitmentIndexs, myCommitmentIndexs, _ = RandomCommitmentsProcess(randomParams)

		// Check number of list of random commitments, list of random commitment indices
		if len(commitmentIndexs) != len(params.inputCoins)*privacy.CommitmentRingSize {
			return NewTransactionErr(RandomCommitmentError, nil)
		}

		if len(myCommitmentIndexs) != len(params.inputCoins) {
			return NewTransactionErr(RandomCommitmentError, errors.New("number of list my commitment indices must be equal to number of input coins"))
		}
	}

	// Calculate execution time for creating payment proof
	startPrivacy := time.Now()

	// Calculate sum of all output coins' value
	sumOutputValue := uint64(0)
	for _, p := range params.paymentInfo {
		sumOutputValue += p.Amount
	}

	// Calculate sum of all input coins' value
	sumInputValue := uint64(0)
	for _, coin := range params.inputCoins {
		sumInputValue += coin.CoinDetails.GetValue()
	}
	Logger.log.Debugf("sumInputValue: %d\n", sumInputValue)

	// Calculate over balance, it will be returned to sender
	overBalance := int64(sumInputValue - sumOutputValue - params.fee)

	// Check if sum of input coins' value is at least sum of output coins' value and tx fee
	if overBalance < 0 {
		return NewTransactionErr(WrongInputError, errors.New(fmt.Sprintf("input value less than output value. sumInputValue=%d sumOutputValue=%d fee=%d", sumInputValue, sumOutputValue, params.fee)))
	}

	// if overBalance > 0, create a new payment info with pk is sender's pk and amount is overBalance
	if overBalance > 0 {
		changePaymentInfo := new(privacy.PaymentInfo)
		changePaymentInfo.Amount = uint64(overBalance)
		changePaymentInfo.PaymentAddress = senderFullKey.PaymentAddress
		params.paymentInfo = append(params.paymentInfo, changePaymentInfo)
	}

	// create new output coins
	outputCoins := make([]*privacy.OutputCoin, len(params.paymentInfo))

	// create SNDs for output coins
	ok := true
	sndOuts := make([]*privacy.Scalar, 0)

	for ok {
		for i := 0; i < len(params.paymentInfo); i++ {
			sndOut := privacy.RandomScalar()
			for {

				ok1, err := CheckSNDerivatorExistence(params.tokenID, sndOut, params.stateDB)
				if err != nil {
					Logger.log.Error(err)
				}
				// if sndOut existed, then re-random it
				if ok1 {
					sndOut = privacy.RandomScalar()
				} else {
					break
				}
			}
			sndOuts = append(sndOuts, sndOut)
		}

		// if sndOuts has two elements that have same value, then re-generates it
		ok = privacy.CheckDuplicateScalarArray(sndOuts)
		if ok {
			sndOuts = make([]*privacy.Scalar, 0)
		}
	}

	// create new output coins with info: Pk, value, last byte of pk, snd
	for i, pInfo := range params.paymentInfo {
		outputCoins[i] = new(privacy.OutputCoin)
		outputCoins[i].CoinDetails = new(privacy.Coin)
		outputCoins[i].CoinDetails.SetValue(pInfo.Amount)
		if len(pInfo.Message) > 0 {
			if len(pInfo.Message) > privacy.MaxSizeInfoCoin {
				return NewTransactionErr(ExceedSizeInfoOutCoinError, nil)
			}
		}
		outputCoins[i].CoinDetails.SetInfo(pInfo.Message)

		PK, err := new(privacy.Point).FromBytesS(pInfo.PaymentAddress.Pk)
		if err != nil {
			Logger.log.Error(errors.New(fmt.Sprintf("can not decompress public key from %+v", pInfo.PaymentAddress)))
			return NewTransactionErr(DecompressPaymentAddressError, err, pInfo.PaymentAddress)
		}
		outputCoins[i].CoinDetails.SetPublicKey(PK)
		outputCoins[i].CoinDetails.SetSNDerivator(sndOuts[i])
	}

	// assign fee tx
	tx.Fee = params.fee

	// create zero knowledge proof of payment
	tx.Proof = &zkp.PaymentProof{}

	// get list of commitments for proving one-out-of-many from commitmentIndexs
	commitmentProving := make([]*privacy.Point, len(commitmentIndexs))
	for i, cmIndex := range commitmentIndexs {
		temp, err := statedb.GetCommitmentByIndex(params.stateDB, *params.tokenID, cmIndex, shardID)
		if err != nil {
			Logger.log.Error(fmt.Errorf("can not get commitment from index=%d shardID=%+v", cmIndex, shardID))
			return NewTransactionErr(CanNotGetCommitmentFromIndexError, err, cmIndex, shardID)
		}
		commitmentProving[i] = new(privacy.Point)
		commitmentProving[i], err = commitmentProving[i].FromBytesS(temp)
		if err != nil {
			Logger.log.Error(errors.New(fmt.Sprintf("can not get commitment from index=%d shardID=%+v value=%+v", cmIndex, shardID, temp)))
			return NewTransactionErr(CanNotDecompressCommitmentFromIndexError, err, cmIndex, shardID, temp)
		}
	}

	// prepare witness for proving
	witness := new(zkp.PaymentWitness)
	paymentWitnessParam := zkp.PaymentWitnessParam{
		HasPrivacy:              params.hasPrivacy,
		PrivateKey:              new(privacy.Scalar).FromBytesS(*params.senderSK),
		InputCoins:              params.inputCoins,
		OutputCoins:             outputCoins,
		PublicKeyLastByteSender: pkLastByteSender,
		Commitments:             commitmentProving,
		CommitmentIndices:       commitmentIndexs,
		MyCommitmentIndices:     myCommitmentIndexs,
		Fee:                     params.fee,
	}
	err = witness.Init(paymentWitnessParam)
	if err.(*privacy.PrivacyError) != nil {
		Logger.log.Error(err)
		jsonParam, _ := json.MarshalIndent(paymentWitnessParam, common.EmptyString, "  ")
		return NewTransactionErr(InitWithnessError, err, string(jsonParam))
	}

	tx.Proof, err = witness.Prove(params.hasPrivacy)
	if err.(*privacy.PrivacyError) != nil {
		Logger.log.Error(err)
		jsonParam, _ := json.MarshalIndent(paymentWitnessParam, common.EmptyString, "  ")
		return NewTransactionErr(WithnessProveError, err, params.hasPrivacy, string(jsonParam))
	}

	Logger.log.Debugf("DONE PROVING........\n")

	// set private key for signing tx
	if params.hasPrivacy {
		randSK := witness.GetRandSecretKey()
		tx.sigPrivKey = append(*params.senderSK, randSK.ToBytesS()...)

		// encrypt coin details (Randomness)
		// hide information of output coins except coin commitments, public key, snDerivators
		for i := 0; i < len(tx.Proof.GetOutputCoins()); i++ {
			err = tx.Proof.GetOutputCoins()[i].Encrypt(params.paymentInfo[i].PaymentAddress.Tk)
			if err.(*privacy.PrivacyError) != nil {
				Logger.log.Error(err)
				return NewTransactionErr(EncryptOutputError, err)
			}
			tx.Proof.GetOutputCoins()[i].CoinDetails.SetSerialNumber(nil)
			tx.Proof.GetOutputCoins()[i].CoinDetails.SetValue(0)
			tx.Proof.GetOutputCoins()[i].CoinDetails.SetRandomness(nil)
		}

		// hide information of input coins except serial number of input coins
		for i := 0; i < len(tx.Proof.GetInputCoins()); i++ {
			tx.Proof.GetInputCoins()[i].CoinDetails.SetCoinCommitment(nil)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetValue(0)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetSNDerivator(nil)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetPublicKey(nil)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetRandomness(nil)
		}

	} else {
		tx.sigPrivKey = []byte{}
		randSK := big.NewInt(0)
		tx.sigPrivKey = append(*params.senderSK, randSK.Bytes()...)
	}

	// sign tx
	tx.PubKeyLastByteSender = pkLastByteSender
	err = tx.signTx()
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(SignTxError, err)
	}

	elapsedPrivacy := time.Since(startPrivacy)
	elapsed := time.Since(start)
	Logger.log.Debugf("Creating payment proof time %s", elapsedPrivacy)
	Logger.log.Debugf("Successfully Creating normal tx %+v in %s time", *tx.Hash(), elapsed)
	return nil
}

// signTx - signs tx
func (tx *Tx) signTx() error {
	//Check input transaction
	if tx.Sig != nil {
		return NewTransactionErr(UnexpectedError, errors.New("input transaction must be an unsigned one"))
	}

	/****** using Schnorr signature *******/
	// sign with sigPrivKey
	// prepare private key for Schnorr
	sk := new(privacy.Scalar).FromBytesS(tx.sigPrivKey[:common.BigIntSize])
	r := new(privacy.Scalar).FromBytesS(tx.sigPrivKey[common.BigIntSize:])
	sigKey := new(privacy.SchnorrPrivateKey)
	sigKey.Set(sk, r)

	// save public key for verification signature tx
	tx.SigPubKey = sigKey.GetPublicKey().GetPublicKey().ToBytesS()

	// signing
	if Logger.log != nil {
		Logger.log.Debugf(tx.Hash().String())
	}
	signature, err := sigKey.Sign(tx.Hash()[:])
	if err != nil {
		return err
	}

	// convert signature to byte array
	tx.Sig = signature.Bytes()

	return nil
}

// verifySigTx - verify signature on tx
func (tx *Tx) verifySigTx() (bool, error) {
	// check input transaction
	if tx.Sig == nil || tx.SigPubKey == nil {
		return false, NewTransactionErr(UnexpectedError, errors.New("input transaction must be an signed one"))
	}

	var err error
	res := false

	/****** verify Schnorr signature *****/
	// prepare Public key for verification
	verifyKey := new(privacy.SchnorrPublicKey)
	sigPublicKey, err := new(privacy.Point).FromBytesS(tx.SigPubKey)

	if err != nil {
		Logger.log.Error(err)
		return false, NewTransactionErr(DecompressSigPubKeyError, err)
	}
	verifyKey.Set(sigPublicKey)

	// convert signature from byte array to SchnorrSign
	signature := new(privacy.SchnSignature)
	err = signature.SetBytes(tx.Sig)
	if err != nil {
		Logger.log.Error(err)
		return false, NewTransactionErr(InitTxSignatureFromBytesError, err)
	}

	// verify signature
	/*Logger.log.Debugf(" VERIFY SIGNATURE ----------- HASH: %v\n", tx.Hash()[:])
	if tx.Proof != nil {
		Logger.log.Debugf(" VERIFY SIGNATURE ----------- TX Proof bytes before verifing the signature: %v\n", tx.Proof.Bytes())
	}
	Logger.log.Debugf(" VERIFY SIGNATURE ----------- TX meta: %v\n", tx.Metadata)*/
	res = verifyKey.Verify(signature, tx.Hash()[:])

	return res, nil
}

// ValidateTransaction returns true if transaction is valid:
// - Verify tx signature
// - Verify the payment proof
// ValidateTransaction returns true if transaction is valid:
// - Verify tx signature
// - Verify the payment proof
func (tx *Tx) ValidateTransaction(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	//hasPrivacy = false
	Logger.log.Debugf("VALIDATING TX........\n")
	if tx.IsSalaryTx() {
		return tx.ValidateTxSalary(transactionStateDB)
	}

	var valid bool
	var err error

	valid, err = tx.verifySigTx()
	if !valid {
		if err != nil {
			Logger.log.Errorf("Error verifying signature with tx hash %s: %+v \n", tx.Hash().String(), err)
			return false, NewTransactionErr(VerifyTxSigFailError, err)
		}
		Logger.log.Errorf("FAILED VERIFICATION SIGNATURE with tx hash %s", tx.Hash().String())
		return false, NewTransactionErr(VerifyTxSigFailError, fmt.Errorf("FAILED VERIFICATION SIGNATURE with tx hash %s", tx.Hash().String()))
	}

	if tx.GetType() == common.TxReturnStakingType {
		return true, nil //
	}

	hasPrivacy, ok := boolParams["hasPrivacy"]
	if !ok {
		hasPrivacy = false
	}
	isNewTransaction, ok := boolParams["isNewTransaction"]
	if !ok {
		isNewTransaction = false
	}

	if tx.Proof != nil {
		if tokenID == nil {
			tokenID = &common.Hash{}
			err := tokenID.SetBytes(common.PRVCoinID[:])
			if err != nil {
				Logger.log.Error(err)
				return false, NewTransactionErr(TokenIDInvalidError, err, tokenID.String())
			}
		}

		sndOutputs := make([]*privacy.Scalar, len(tx.Proof.GetOutputCoins()))
		for i := 0; i < len(tx.Proof.GetOutputCoins()); i++ {
			sndOutputs[i] = tx.Proof.GetOutputCoins()[i].CoinDetails.GetSNDerivator()
		}

		if privacy.CheckDuplicateScalarArray(sndOutputs) {
			Logger.log.Errorf("Duplicate output coins' snd\n")
			return false, NewTransactionErr(DuplicatedOutputSndError, errors.New("Duplicate output coins' snd\n"))
		}

		if isNewTransaction {
			for i := 0; i < len(tx.Proof.GetOutputCoins()); i++ {
				// Check output coins' SND is not exists in SND list (Database)
				if ok, err := CheckSNDerivatorExistence(tokenID, tx.Proof.GetOutputCoins()[i].CoinDetails.GetSNDerivator(), transactionStateDB); ok || err != nil {
					if err != nil {
						Logger.log.Error(err)
					}
					Logger.log.Errorf("snd existed: %d\n", i)
					return false, NewTransactionErr(SndExistedError, err, fmt.Sprintf("snd existed: %d\n", i))
				}
			}
		}

		if !hasPrivacy {
			// Check input coins' commitment is exists in cm list (Database)
			for i := 0; i < len(tx.Proof.GetInputCoins()); i++ {
				ok, err := tx.CheckCMExistence(tx.Proof.GetInputCoins()[i].CoinDetails.GetCoinCommitment().ToBytesS(), transactionStateDB, shardID, tokenID)
				if !ok || err != nil {
					if err != nil {
						Logger.log.Error(err)
					}
					return false, NewTransactionErr(InputCommitmentIsNotExistedError, err)
				}
			}
		}
		// Verify the payment proof
		valid, err = tx.Proof.Verify(boolParams, tx.SigPubKey, tx.Fee, transactionStateDB, shardID, tokenID)
		if !valid {
			if err != nil {
				Logger.log.Error(err)
			}
			Logger.log.Error("FAILED VERIFICATION PAYMENT PROOF")
			err1, ok := err.(*privacy.PrivacyError)
			if ok {
				// parse error detail
				if err1.Code == privacy.ErrCodeMessage[privacy.VerifyOneOutOfManyProofFailedErr].Code {
					if isNewTransaction {
						return false, NewTransactionErr(VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
					} else {
						// for old txs which be get from sync block or validate new block
						if tx.LockTime <= ValidateTimeForOneoutOfManyProof {
							// only verify by sign on block because of issue #504(that mean we should pass old tx, which happen before this issue)
							return true, nil
						} else {
							return false, NewTransactionErr(VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
						}
					}
				}
			}
			return false, NewTransactionErr(TxProofVerifyFailError, err, tx.Hash().String())
		} else {
			Logger.log.Debugf("SUCCESSED VERIFICATION PAYMENT PROOF ")
		}
	}
	//@UNCOMMENT: metrics time
	//elapsed := time.Since(start)
	//Logger.log.Debugf("Validation normal tx %+v in %s time \n", *tx.Hash(), elapsed)

	return true, nil
}

func (tx Tx) String() string {
	record := strconv.Itoa(int(tx.Version))

	record += strconv.FormatInt(tx.LockTime, 10)
	record += strconv.FormatUint(tx.Fee, 10)
	if tx.Proof != nil {
		tmp := base64.StdEncoding.EncodeToString(tx.Proof.Bytes())
		//tmp := base58.Base58Check{}.Encode(tx.Proof.Bytes(), 0x00)
		record += tmp
		// fmt.Printf("Proof check base 58: %v\n",tmp)
	}
	if tx.Metadata != nil {
		metadataHash := tx.Metadata.Hash()
		//Logger.log.Debugf("\n\n\n\n test metadata after hashing: %v\n", metadataHash.GetBytes())
		metadataStr := metadataHash.String()
		record += metadataStr
	}

	//TODO: To be uncomment
	// record += string(tx.Info)
	return record
}

func (tx *Tx) Hash() *common.Hash {
	if tx.cachedHash != nil {
		return tx.cachedHash
	}
	inBytes := []byte(tx.String())
	hash := common.HashH(inBytes)
	tx.cachedHash = &hash
	return &hash
}

func (tx Tx) GetSenderAddrLastByte() byte {
	return tx.PubKeyLastByteSender
}

func (tx Tx) GetTxFee() uint64 {
	return tx.Fee
}

func (tx Tx) GetTxFeeToken() uint64 {
	return uint64(0)
}

// GetTxActualSize computes the actual size of a given transaction in kilobyte
func (tx Tx) GetTxActualSize() uint64 {
	//txBytes, _ := json.Marshal(tx)
	//txSizeInByte := len(txBytes)
	//
	//return uint64(math.Ceil(float64(txSizeInByte) / 1024))
	if tx.cachedActualSize != nil {
		return *tx.cachedActualSize
	}
	sizeTx := uint64(1)                // int8
	sizeTx += uint64(len(tx.Type) + 1) // string
	sizeTx += uint64(8)                // int64
	sizeTx += uint64(8)

	sigPubKey := uint64(len(tx.SigPubKey))
	sizeTx += sigPubKey
	sig := uint64(len(tx.Sig))
	sizeTx += sig
	if tx.Proof != nil {
		proof := uint64(len(tx.Proof.Bytes()))
		sizeTx += proof
	}

	sizeTx += uint64(1)
	info := uint64(len(tx.Info))
	sizeTx += info

	meta := tx.Metadata
	if meta != nil {
		metaSize := meta.CalculateSize()
		sizeTx += metaSize
	}
	result := uint64(math.Ceil(float64(sizeTx) / 1024))
	tx.cachedActualSize = &result
	return *tx.cachedActualSize
}

// GetType returns the type of the transaction
func (tx Tx) GetType() string {
	return tx.Type
}

func (tx Tx) ListSerialNumbersHashH() []common.Hash {
	result := []common.Hash{}
	if tx.Proof != nil {
		for _, d := range tx.Proof.GetInputCoins() {
			hash := common.HashH(d.CoinDetails.GetSerialNumber().ToBytesS())
			result = append(result, hash)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

// CheckCMExistence returns true if cm exists in cm list
func (tx Tx) CheckCMExistence(cm []byte, stateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	ok, err := statedb.HasCommitment(stateDB, *tokenID, cm, shardID)
	return ok, err
}

func (tx Tx) CheckTxVersion(maxTxVersion int8) bool {
	return !(tx.Version > maxTxVersion)
}

// func (tx Tx) CheckTransactionFee(minFeePerKbTx uint64) bool {
// 	if tx.IsSalaryTx() {
// 		return true
// 	}
// 	if tx.Metadata != nil {
// 		return tx.Metadata.CheckTransactionFee(&tx, minFeePerKbTx)
// 	}
// 	fullFee := minFeePerKbTx * tx.GetTxActualSize()
// 	return tx.Fee >= fullFee
// }

func (tx Tx) IsSalaryTx() bool {
	// Check normal tx(not an action tx)
	if tx.GetType() != common.TxRewardType {
		return false
	}
	// Check serialNumber in every Descs
	if len(tx.Proof.GetInputCoins()) == 0 {
		return true
	}
	return false
}

func (tx Tx) GetSender() []byte {
	if tx.Proof == nil || len(tx.Proof.GetInputCoins()) == 0 {
		return nil
	}
	if tx.IsPrivacy() {
		return nil
	}
	if len(tx.Proof.GetInputCoins()) == 0 || tx.Proof.GetInputCoins()[0].CoinDetails == nil {
		return nil
	}
	return tx.Proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
}

func (tx Tx) GetReceivers() ([][]byte, []uint64) {
	pubkeys := [][]byte{}
	amounts := []uint64{}
	if tx.Proof != nil && len(tx.Proof.GetOutputCoins()) > 0 {
		for _, coin := range tx.Proof.GetOutputCoins() {
			added := false
			coinPubKey := coin.CoinDetails.GetPublicKey().ToBytesS()
			for i, key := range pubkeys {
				if bytes.Equal(coinPubKey, key) {
					added = true
					amounts[i] += coin.CoinDetails.GetValue()
					break
				}
			}
			if !added {
				pubkeys = append(pubkeys, coinPubKey)
				amounts = append(amounts, coin.CoinDetails.GetValue())
			}
		}
	}
	return pubkeys, amounts
}

func (tx Tx) GetUniqueReceiver() (bool, []byte, uint64) {
	sender := []byte{} // Empty byte slice for coinbase tx
	if tx.Proof != nil && len(tx.Proof.GetInputCoins()) > 0 && !tx.IsPrivacy() {
		sender = tx.Proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	}
	pubkeys, amounts := tx.GetReceivers()
	pubkey := []byte{}
	amount := uint64(0)
	count := 0
	for i, pk := range pubkeys {
		if !bytes.Equal(pk, sender) {
			pubkey = pk
			amount = amounts[i]
			count += 1
		}
	}
	return count == 1, pubkey, amount
}

func (tx Tx) GetTransferData() (bool, []byte, uint64, *common.Hash) {
	unique, pk, amount := tx.GetUniqueReceiver()
	return unique, pk, amount, &common.PRVCoinID
}

func (tx Tx) GetTokenReceivers() ([][]byte, []uint64) {
	return nil, nil
}

func (tx Tx) GetTokenUniqueReceiver() (bool, []byte, uint64) {
	return false, nil, 0
}

func (tx Tx) validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH map[common.Hash][]common.Hash) error {
	if tx.Proof == nil {
		return nil
	}
	temp := make(map[common.Hash]interface{})
	for _, desc := range tx.Proof.GetInputCoins() {
		hash := common.HashH(desc.CoinDetails.GetSerialNumber().ToBytesS())
		temp[hash] = nil
	}

	for _, listSerialNumbers := range poolSerialNumbersHashH {
		for _, serialNumberHash := range listSerialNumbers {
			if _, ok := temp[serialNumberHash]; ok {
				return errors.New("double spend")
			}
		}
	}
	return nil
}

func (tx Tx) ValidateTxWithCurrentMempool(mr metadata.MempoolRetriever) error {
	poolSerialNumbersHashH := mr.GetSerialNumbersHashH()
	return tx.validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH)
}

// ValidateDoubleSpend - check double spend for any transaction type
func (tx Tx) ValidateDoubleSpendWithBlockchain(
	shardID byte,
	stateDB *statedb.StateDB,
	tokenID *common.Hash,
) error {

	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return err
	}
	if tokenID != nil {
		err := prvCoinID.SetBytes(tokenID.GetBytes())
		if err != nil {
			return err
		}
	}
	for i := 0; tx.Proof != nil && i < len(tx.Proof.GetInputCoins()); i++ {
		serialNumber := tx.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber().ToBytesS()
		ok, err := statedb.HasSerialNumber(stateDB, *prvCoinID, serialNumber, shardID)
		if ok || err != nil {
			return errors.New("double spend")
		}
	}
	return nil
}

func (tx Tx) ValidateTxWithBlockChain(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error {
	if tx.GetType() == common.TxRewardType || tx.GetType() == common.TxReturnStakingType {
		return nil
	}

	if tx.Metadata != nil {
		isContinued, err := tx.Metadata.ValidateTxWithBlockChain(&tx, chainRetriever, shardViewRetriever, beaconViewRetriever, shardID, stateDB)
		// fmt.Printf("[transactionStateDB] validate metadata with blockchain: %d %h %t %v\n", tx.GetMetadataType(), tx.Hash(), isContinued, err)
		if err != nil {
			Logger.log.Errorf("[db] validate metadata with blockchain: %d %s %t %v\n", tx.GetMetadataType(), tx.Hash().String(), isContinued, err)
			return NewTransactionErr(RejectTxMedataWithBlockChain, fmt.Errorf("validate metadata of tx %s with blockchain error %+v", tx.Hash().String(), err))
		}
		if !isContinued {
			return nil
		}
	}

	return tx.ValidateDoubleSpendWithBlockchain(shardID, stateDB, nil)
}

func (tx Tx) validateNormalTxSanityData(bcr metadata.ChainRetriever, beaconHeight uint64) (bool, error) {
	//check version
	if tx.Version > txVersion {
		return false, NewTransactionErr(RejectTxVersion, fmt.Errorf("tx version is %d. Wrong version tx. Only support for version >= %d", tx.Version, txVersion))
	}
	// check LockTime before now
	if int64(tx.LockTime) > time.Now().Unix() {
		return false, NewTransactionErr(RejectInvalidLockTime, fmt.Errorf("wrong tx locktime %d", tx.LockTime))
	}

	// check tx size
	actualTxSize := tx.GetTxActualSize()
	if actualTxSize > common.MaxTxSize {
		//fmt.Print(actualTxSize, common.MaxTxSize)
		return false, NewTransactionErr(RejectTxSize, fmt.Errorf("tx size %d kB is too large", actualTxSize))
	}

	// check sanity of Proof
	validateSanityOfProof, err := tx.validateSanityDataOfProof(bcr, beaconHeight)
	if err != nil || !validateSanityOfProof {
		return false, err
	}

	if len(tx.SigPubKey) != common.SigPubKeySize {
		return false, NewTransactionErr(RejectTxPublickeySigSize, fmt.Errorf("wrong tx Sig PK size %d", len(tx.SigPubKey)))
	}
	// check Type is normal or salary tx
	switch tx.Type {
	case common.TxNormalType, common.TxRewardType, common.TxCustomTokenPrivacyType, common.TxReturnStakingType: //is valid
	default:
		return false, NewTransactionErr(RejectTxType, fmt.Errorf("wrong tx type with %s", tx.Type))
	}

	//if txN.Type != common.TxNormalType && txN.Type != common.TxRewardType && txN.Type != common.TxCustomTokenType && txN.Type != common.TxCustomTokenPrivacyType { // only 1 byte
	//	return false, errors.New("wrong tx type")
	//}

	// check info field
	if len(tx.Info) > 512 {
		return false, NewTransactionErr(RejectTxInfoSize, fmt.Errorf("wrong tx info length %d bytes, only support info with max length <= %d bytes", len(tx.Info), 512))
	}

	return true, nil
}

func (txN Tx) validateSanityDataOfProof(bcr metadata.ChainRetriever, beaconHeight uint64) (bool, error) {
	if txN.Proof != nil {
		if len(txN.Proof.GetInputCoins()) > 255 {
			return false, errors.New("Input coins in tx are very large:" + strconv.Itoa(len(txN.Proof.GetInputCoins())))
		}

		if len(txN.Proof.GetOutputCoins()) > 255 {
			return false, errors.New("Output coins in tx are very large:" + strconv.Itoa(len(txN.Proof.GetOutputCoins())))
		}

		// check doubling a input coin in tx
		serialNumbers := make(map[common.Hash]bool)
		for i, inCoin := range txN.Proof.GetInputCoins() {
			hashSN := common.HashH(inCoin.CoinDetails.GetSerialNumber().ToBytesS())
			if serialNumbers[hashSN] {
				Logger.log.Errorf("Double input in tx - txId %v - index %v", txN.Hash().String(), i)
				return false, errors.New("double input in tx")
			}
			serialNumbers[hashSN] = true
		}

		isPrivacy := txN.IsPrivacy()

		if isPrivacy {
			// check cmValue of output coins is equal to comValue in Bulletproof
			cmValueOfOutputCoins := txN.Proof.GetCommitmentOutputValue()
			cmValueInBulletProof := txN.Proof.GetAggregatedRangeProof().GetCmValues()
			if len(cmValueOfOutputCoins) != len(cmValueInBulletProof) {
				return false, errors.New("invalid cmValues in Bullet proof")
			}

			if len(txN.Proof.GetInputCoins()) != len(txN.Proof.GetSerialNumberProof()) || len(txN.Proof.GetInputCoins()) != len(txN.Proof.GetOneOfManyProof()){
				return false, errors.New("the number of input coins must be equal to the number of serialnumber proofs and the number of one-of-many proofs")
			}

			for i := 0; i < len(cmValueOfOutputCoins); i++ {
				if !privacy.IsPointEqual(cmValueOfOutputCoins[i], cmValueInBulletProof[i]) {
					Logger.log.Errorf("cmValue in Bulletproof is not equal to commitment of output's Value - txId %v", txN.Hash().String())
					return false, fmt.Errorf("cmValue %v in Bulletproof is not equal to commitment of output's Value", i)
				}
			}

			if !txN.Proof.GetAggregatedRangeProof().ValidateSanity() {
				return false, errors.New("validate sanity Aggregated range proof failed")
			}

			for i := 0; i < len(txN.Proof.GetOneOfManyProof()); i++ {
				if !txN.Proof.GetOneOfManyProof()[i].ValidateSanity() {
					return false, errors.New("validate sanity One out of many proof failed")
				}
			}

			cmInputSNDs := txN.Proof.GetCommitmentInputSND()
			cmInputSK := txN.Proof.GetCommitmentInputSecretKey()
			for i := 0; i < len(txN.Proof.GetSerialNumberProof()); i++ {
				// check cmSK of input coin is equal to comSK in serial number proof
				if !privacy.IsPointEqual(cmInputSK, txN.Proof.GetSerialNumberProof()[i].GetComSK()) {
					Logger.log.Errorf("ComSK in SNproof is not equal to commitment of private key - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("comSK of SNProof %v is not comSK of input coins", i))
				}

				// check cmSND of input coins is equal to comInputSND in serial number proof
				if !privacy.IsPointEqual(cmInputSNDs[i], txN.Proof.GetSerialNumberProof()[i].GetComInput()) {
					Logger.log.Errorf("cmSND in SNproof is not equal to commitment of input's SND - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("cmSND in SNproof %v is not equal to commitment of input's SND", i))
				}

				// check SN of input coins is equal to the corresponding SN in serial number proof
				if !privacy.IsPointEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber(), txN.Proof.GetSerialNumberProof()[i].GetSN()) {
					Logger.log.Errorf("SN in SNProof is not equal to SN of input coin - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberPrivacyProofFailedErr, fmt.Errorf("SN in SNProof %v is not equal to SN of input coin", i))
				}

				if !txN.Proof.GetSerialNumberProof()[i].ValidateSanity() {
					return false, errors.New("validate sanity Serial number proof failed")
				}
			}

			// check input coins with privacy
			for i := 0; i < len(txN.Proof.GetInputCoins()); i++ {
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber().PointValid() {
					return false, errors.New("validate sanity Serial number of input coin failed")
				}
			}
			// check output coins with privacy
			for i := 0; i < len(txN.Proof.GetOutputCoins()); i++ {
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetPublicKey().PointValid() {
					return false, errors.New("validate sanity Public key of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetCoinCommitment().PointValid() {
					return false, errors.New("validate sanity Coin commitment of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetSNDerivator().ScalarValid() {
					return false, errors.New("validate sanity SNDerivator of output coin failed")
				}
			}
			// check ComInputSK
			if !txN.Proof.GetCommitmentInputSecretKey().PointValid() {
				return false, errors.New("validate sanity ComInputSK of proof failed")
			}

			// check SigPubKey
			sigPubKeyPoint, err := new(privacy.Point).FromBytesS(txN.GetSigPubKey())
			if err != nil {
				Logger.log.Errorf("SigPubKey is invalid - txId %v", txN.Hash().String())
				return false, errors.New("SigPubKey is invalid")
			}
			if !privacy.IsPointEqual(cmInputSK, sigPubKeyPoint) {
				Logger.log.Errorf("SigPubKey is not equal to commitment of private key - txId %v", txN.Hash().String())
				return false, errors.New("SigPubKey is not equal to commitment of private key")
			}

			// check ComInputValue
			for i := 0; i < len(txN.Proof.GetCommitmentInputValue()); i++ {
				if !txN.Proof.GetCommitmentInputValue()[i].PointValid() {
					return false, errors.New("validate sanity ComInputValue of proof failed")
				}
			}
			//check ComInputSND
			for i := 0; i < len(txN.Proof.GetCommitmentInputSND()); i++ {
				if !txN.Proof.GetCommitmentInputSND()[i].PointValid() {
					return false, errors.New("validate sanity ComInputSND of proof failed")
				}
			}

			//check ComInputShardID
			if !txN.Proof.GetCommitmentInputShardID().PointValid() {
				return false, errors.New("validate sanity ComInputShardID of proof failed")
			}

			fixedRand := bcr.GetFixedRandomForShardIDCommitment(beaconHeight)
			if fixedRand != nil {
				shardIDSender := common.GetShardIDFromLastByte(txN.GetSenderAddrLastByte())
				expectedCMShardID := privacy.PedCom.CommitAtIndex(
					new(privacy.Scalar).FromUint64(uint64(shardIDSender)),
					fixedRand, privacy.PedersenShardIDIndex)

				if !privacy.IsPointEqual(expectedCMShardID, txN.Proof.GetCommitmentInputShardID()) {
					return false, errors.New("ComInputShardID must be committed with the fixed randomness")
				}
			}

			// check ComOutputShardID
			for i := 0; i < len(txN.Proof.GetCommitmentOutputShardID()); i++ {
				if !txN.Proof.GetCommitmentOutputShardID()[i].PointValid() {
					return false, errors.New("validate sanity ComOutputShardID of proof failed")
				}
			}
			//check ComOutputSND
			for i := 0; i < len(txN.Proof.GetCommitmentOutputShardID()); i++ {
				if !txN.Proof.GetCommitmentOutputShardID()[i].PointValid() {
					return false, errors.New("validate sanity ComOutputSND of proof failed")
				}
			}
			//check ComOutputValue
			for i := 0; i < len(txN.Proof.GetCommitmentOutputValue()); i++ {
				if !txN.Proof.GetCommitmentOutputValue()[i].PointValid() {
					return false, errors.New("validate sanity ComOutputValue of proof failed")
				}
			}
			if len(txN.Proof.GetCommitmentIndices()) != len(txN.Proof.GetInputCoins())*privacy.CommitmentRingSize {
				return false, errors.New("validate sanity CommitmentIndices of proof failed")

			}
		}

		if !isPrivacy {
			// check SigPubKey
			sigPubKeyPoint, err := new(privacy.Point).FromBytesS(txN.GetSigPubKey())
			if err != nil {
				Logger.log.Errorf("SigPubKey is invalid - txId %v", txN.Hash().String())
				return false, errors.New("SigPubKey is invalid")
			}
			inputCoins := txN.Proof.GetInputCoins()

			if len(inputCoins) != len(txN.Proof.GetSerialNumberNoPrivacyProof()){
				return false, errors.New("the number of input coins must be equal to the number of serialnumbernoprivacy proofs")
			}

			for i := 0; i < len(inputCoins); i++ {
				// check PublicKey of input coin is equal to SigPubKey
				if !privacy.IsPointEqual(inputCoins[i].CoinDetails.GetPublicKey(), sigPubKeyPoint) {
					Logger.log.Errorf("SigPubKey is not equal to public key of input coins - txId %v", txN.Hash().String())
					return false, errors.New("SigPubKey is not equal to public key of input coins")
				}
			}

			for i := 0; i < len(txN.Proof.GetSerialNumberNoPrivacyProof()); i++ {
				// check PK of input coin is equal to vKey in serial number proof
				if !privacy.IsPointEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetPublicKey(), txN.Proof.GetSerialNumberNoPrivacyProof()[i].GetVKey()) {
					Logger.log.Errorf("VKey in SNNoPrivacyProof is not equal public key of sender - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("VKey of SNNoPrivacyProof %v is not public key of sender", i))
				}

				// check SND of input coins is equal to SND in serial number no privacy proof
				if !privacy.IsScalarEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetSNDerivator(), txN.Proof.GetSerialNumberNoPrivacyProof()[i].GetInput()) {
					Logger.log.Errorf("SND in SNNoPrivacyProof is not equal to input's SND - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("SND in SNNoPrivacyProof %v is not equal to input's SND", i))
				}

				// check SND of input coins is equal to SND in serial number no privacy proof
				if !privacy.IsPointEqual(txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber(), txN.Proof.GetSerialNumberNoPrivacyProof()[i].GetOutput()) {
					Logger.log.Errorf("SN in SNNoPrivacyProof is not equal to SN in input coin - txId %v", txN.Hash().String())
					return false, privacy.NewPrivacyErr(privacy.VerifySerialNumberNoPrivacyProofFailedErr, fmt.Errorf("SN in SNNoPrivacyProof %v is not equal to SN in input coin", i))
				}

				if !txN.Proof.GetSerialNumberNoPrivacyProof()[i].ValidateSanity() {
					return false, errors.New("validate sanity Serial number no privacy proof failed")
				}
			}
			// check input coins without privacy
			for i := 0; i < len(txN.Proof.GetInputCoins()); i++ {
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetCoinCommitment().PointValid() {
					return false, errors.New("validate sanity CoinCommitment of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetPublicKey().PointValid() {
					return false, errors.New("validate sanity PublicKey of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetSerialNumber().PointValid() {
					return false, errors.New("validate sanity Serial number of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetRandomness().ScalarValid() {
					return false, errors.New("validate sanity Randomness of input coin failed")
				}
				if !txN.Proof.GetInputCoins()[i].CoinDetails.GetSNDerivator().ScalarValid() {
					return false, errors.New("validate sanity SNDerivator of input coin failed")
				}
			}

			// check output coins without privacy
			for i := 0; i < len(txN.Proof.GetOutputCoins()); i++ {
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetCoinCommitment().PointValid() {
					return false, errors.New("validate sanity CoinCommitment of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetPublicKey().PointValid() {
					return false, errors.New("validate sanity PublicKey of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetRandomness().ScalarValid() {
					return false, errors.New("validate sanity Randomness of output coin failed")
				}
				if !txN.Proof.GetOutputCoins()[i].CoinDetails.GetSNDerivator().ScalarValid() {
					return false, errors.New("validate sanity SNDerivator of output coin failed")
				}
			}
		}
	}
	return true, nil
}

func (tx Tx) ValidateSanityData(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, error) {
	Logger.log.Debugf("\n\n\n START Validating sanity data of metadata %+v\n\n\n", tx.Metadata)
	if tx.Metadata != nil {
		Logger.log.Debug("tx.Metadata.ValidateSanityData")
		isContinued, ok, err := tx.Metadata.ValidateSanityData(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight, &tx)
		Logger.log.Debug("END tx.Metadata.ValidateSanityData")
		if err != nil || !ok || !isContinued {
			return ok, err
		}
	}
	Logger.log.Debugf("\n\n\n END sanity data of metadata%+v\n\n\n")
	return tx.validateNormalTxSanityData(chainRetriever, beaconHeight)
}

func (tx Tx) ValidateTxByItself(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, chainRetriever metadata.ChainRetriever, shardID byte, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever) (bool, error) {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return false, err
	}
	ok, err := tx.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, prvCoinID)
	if !ok {
		return false, err
	}

	hasPrivacy, ok := boolParams["hasPrivacy"]
	if !ok {
		hasPrivacy = false
	}
	if tx.Metadata != nil {
		if hasPrivacy {
			return false, errors.New("metadata can not exist in  privacy tx")
		}
		validateMetadata := tx.Metadata.ValidateMetadataByItself()
		if validateMetadata {
			return validateMetadata, nil
		} else {
			return validateMetadata, NewTransactionErr(UnexpectedError, errors.New("Metadata is invalid"))
		}
	}
	return true, nil
}


// GetMetadataType returns the type of underlying metadata if is existed
func (tx Tx) GetMetadataType() int {
	if tx.Metadata != nil {
		return tx.Metadata.GetType()
	}
	return metadata.InvalidMeta
}

// GetMetadata returns metadata of tx is existed
func (tx Tx) GetMetadata() metadata.Metadata {
	return tx.Metadata
}

// SetMetadata sets metadata to tx
func (tx *Tx) SetMetadata(meta metadata.Metadata) {
	tx.Metadata = meta
}

// GetMetadata returns metadata of tx is existed
func (tx Tx) GetInfo() []byte {
	return tx.Info
}

func (tx Tx) GetLockTime() int64 {
	return tx.LockTime
}

func (tx Tx) GetSigPubKey() []byte {
	return tx.SigPubKey
}

func (tx Tx) GetProof() *zkp.PaymentProof {
	return tx.Proof
}

func (tx Tx) IsPrivacy() bool {
	if tx.Proof == nil || tx.Proof.GetAggregatedRangeProof() == nil || len(tx.Proof.GetOneOfManyProof()) == 0 || len(tx.Proof.GetSerialNumberProof()) == 0 {
		return false
	}
	return true
}

func (tx Tx) ValidateType() bool {
	return tx.Type == common.TxNormalType || tx.Type == common.TxRewardType || tx.Type == common.TxReturnStakingType
}

func (tx Tx) CalculateBurningTxValue(bcr metadata.ChainRetriever, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, uint64) {
	if tx.Proof == nil || len(tx.Proof.GetOutputCoins()) == 0 {
		return false, 0
	}
	//get burning address
	burningAddress := bcr.GetBurningAddress(beaconHeight)
	keyWalletBurningAccount, err := wallet.Base58CheckDeserialize(burningAddress)
	if err != nil {
		return false, 0
	}
	keysetBurningAccount := keyWalletBurningAccount.KeySet
	paymentAddressBurningAccount := keysetBurningAccount.PaymentAddress

	// check burning amount
	totalBurningAmount := uint64(0)
	for _, outCoin := range tx.Proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if bytes.Equal(outPKBytes, paymentAddressBurningAccount.Pk[:]) {
			totalBurningAmount += outCoin.CoinDetails.GetValue()
		}
	}
	if totalBurningAmount > 0 {
		return true, totalBurningAmount
	}
	return false, 0
}

func (tx Tx) IsCoinsBurning(bcr metadata.ChainRetriever, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) bool {
	if tx.Proof == nil || len(tx.Proof.GetOutputCoins()) == 0 {
		return false
	}
	senderPKBytes := []byte{}
	if len(tx.Proof.GetInputCoins()) > 0 {
		senderPKBytes = tx.Proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	}
	//get burning address
	burningAddress := bcr.GetBurningAddress(beaconHeight)
	keyWalletBurningAccount, err := wallet.Base58CheckDeserialize(burningAddress)
	if err != nil {
		return false
	}
	keysetBurningAccount := keyWalletBurningAccount.KeySet
	paymentAddressBurningAccount := keysetBurningAccount.PaymentAddress
	for _, outCoin := range tx.Proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if !bytes.Equal(senderPKBytes, outPKBytes) && !bytes.Equal(outPKBytes, paymentAddressBurningAccount.Pk[:]) {
			return false
		}
	}
	return true
}

func (tx Tx) CalculateTxValue() uint64 {
	if tx.Proof == nil {
		return 0
	}
	if tx.Proof.GetOutputCoins() == nil || len(tx.Proof.GetOutputCoins()) == 0 {
		return 0
	}
	if tx.Proof.GetInputCoins() == nil || len(tx.Proof.GetInputCoins()) == 0 { // coinbase tx
		txValue := uint64(0)
		for _, outCoin := range tx.Proof.GetOutputCoins() {
			txValue += outCoin.CoinDetails.GetValue()
		}
		return txValue
	}

	senderPKBytes := tx.Proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	txValue := uint64(0)
	for _, outCoin := range tx.Proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if bytes.Equal(senderPKBytes, outPKBytes) {
			continue
		}
		txValue += outCoin.CoinDetails.GetValue()
	}
	return txValue
}

// InitTxSalary
// Init salary transaction
// #1 - salary:
// #2 - receiverAddr:
// #3 - privKey:
// #4 - snDerivators:
func (tx *Tx) InitTxSalary(salary uint64, receiverAddr *privacy.PaymentAddress, privateKey *privacy.PrivateKey, stateDB *statedb.StateDB, metaData metadata.Metadata) error {
	var err error
	sndOut := privacy.RandomScalar()
	tx.Version = txVersion
	tx.Type = common.TxRewardType
	if tx.LockTime == 0 {
		tx.LockTime = time.Now().Unix()
	}
	// create new output coins with info: Pk, value, input, randomness, last byte pk, coin commitment
	tx.Proof = new(zkp.PaymentProof)
	tempOutputCoin := make([]*privacy.OutputCoin, 1)
	tempOutputCoin[0] = new(privacy.OutputCoin).Init()
	publicKey, err := new(privacy.Point).FromBytesS(receiverAddr.Pk)
	if err != nil {
		return err
	}
	tempOutputCoin[0].CoinDetails.SetPublicKey(publicKey)
	tempOutputCoin[0].CoinDetails.SetValue(salary)
	tempOutputCoin[0].CoinDetails.SetRandomness(privacy.RandomScalar())
	for {
		tokenID := &common.Hash{}
		err := tokenID.SetBytes(common.PRVCoinID[:])
		if err != nil {
			return NewTransactionErr(TokenIDInvalidError, err, tokenID.String())
		}
		ok, err := CheckSNDerivatorExistence(tokenID, sndOut, stateDB)
		if err != nil {
			return NewTransactionErr(SndExistedError, err)
		}
		if ok {
			sndOut = privacy.RandomScalar()
		} else {
			break
		}
	}
	tempOutputCoin[0].CoinDetails.SetSNDerivator(sndOut)
	// create coin commitment
	err = tempOutputCoin[0].CoinDetails.CommitAll()
	if err != nil {
		return NewTransactionErr(CommitOutputCoinError, err)
	}
	tx.Proof.SetOutputCoins(tempOutputCoin)
	// get last byte
	tx.PubKeyLastByteSender = receiverAddr.Pk[len(receiverAddr.Pk)-1]
	// sign Tx
	tx.SigPubKey = receiverAddr.Pk
	tx.sigPrivKey = *privateKey
	tx.SetMetadata(metaData)
	err = tx.signTx()
	if err != nil {
		return NewTransactionErr(SignTxError, err)
	}
	return nil
}

func (tx Tx) ValidateTxReturnStaking(stateDB *statedb.StateDB) bool {
	return true
}

func (tx Tx) ValidateTxSalary(stateDB *statedb.StateDB) (bool, error) {
	// verify signature
	valid, err := tx.verifySigTx()
	if !valid {
		if err != nil {
			Logger.log.Debugf("Error verifying signature of tx: %+v", err)
			return false, NewTransactionErr(VerifyTxSigFailError, err)
		}
		return false, nil
	}
	// check whether output coin's input exists in input list or not
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return false, NewTransactionErr(TokenIDInvalidError, err, tokenID.String())
	}
	if ok, err := CheckSNDerivatorExistence(tokenID, tx.Proof.GetOutputCoins()[0].CoinDetails.GetSNDerivator(), stateDB); ok || err != nil {
		return false, err
	}
	// check output coin's coin commitment is calculated correctly
	coin := tx.Proof.GetOutputCoins()[0].CoinDetails
	shardID2 := common.GetShardIDFromLastByte(coin.GetPubKeyLastByte())
	cmTmp2 := new(privacy.Point)
	cmTmp2.Add(coin.GetPublicKey(), new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenValueIndex], new(privacy.Scalar).FromUint64(uint64(coin.GetValue()))))
	cmTmp2.Add(cmTmp2, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenSndIndex], coin.GetSNDerivator()))
	cmTmp2.Add(cmTmp2, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenShardIDIndex], new(privacy.Scalar).FromUint64(uint64(shardID2))))
	cmTmp2.Add(cmTmp2, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], coin.GetRandomness()))

	ok := privacy.IsPointEqual(cmTmp2, tx.Proof.GetOutputCoins()[0].CoinDetails.GetCoinCommitment())
	if !ok {
		return ok, NewTransactionErr(UnexpectedError, errors.New("check output coin's coin commitment isn't calculated correctly"))
	}
	return ok, nil
}

func (tx Tx) GetMetadataFromVinsTx(bcr metadata.ChainRetriever, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever) (metadata.Metadata, error) {
	// implement this func if needed
	return nil, nil
}

func (tx Tx) GetTokenID() *common.Hash {
	return &common.PRVCoinID
}

func (tx Tx) VerifyMinerCreatedTxBeforeGettingInBlock(txsInBlock []metadata.Transaction, txsUsed []int, insts [][]string, instsUsed []int, shardID byte, bcr metadata.ChainRetriever, accumulatedValues *metadata.AccumulatedValues, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever) (bool, error) {
	if tx.IsPrivacy() {
		return true, nil
	}
	meta := tx.Metadata
	if tx.Proof != nil && len(tx.Proof.GetInputCoins()) == 0 && len(tx.Proof.GetOutputCoins()) > 0 { // coinbase tx
		if meta == nil {
			return false, nil
		}
		if !meta.IsMinerCreatedMetaType() {
			return false, nil
		}
	}
	//if type is reward and not have metadata
	if tx.GetType() == common.TxRewardType && meta == nil  {
		return false, nil
	}
	//if type is return staking and not have metadata
	if tx.GetType() == common.TxReturnStakingType && (meta == nil || (meta.GetType() != metadata.ReturnStakingMeta)) {
		return false, nil
	}
	if meta != nil {
		ok, err := meta.VerifyMinerCreatedTxBeforeGettingInBlock(txsInBlock, txsUsed, insts, instsUsed, shardID, &tx, bcr, accumulatedValues, nil, nil)
		if err != nil {
			Logger.log.Error(err)
			return false, NewTransactionErr(VerifyMinerCreatedTxBeforeGettingInBlockError, err)
		}
		return ok, nil
	}
	return true, nil
}

type TxPrivacyInitParamsForASM struct {
	txParam             TxPrivacyInitParams
	commitmentIndices   []uint64
	commitmentBytes     [][]byte
	myCommitmentIndices []uint64
	sndOutputs          []*privacy.Scalar
}

func NewTxPrivacyInitParamsForASM(
	senderSK *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoins []*privacy.InputCoin,
	fee uint64,
	hasPrivacy bool,
	tokenID *common.Hash, // default is nil -> use for prv coin
	metaData metadata.Metadata,
	info []byte,
	commitmentIndices []uint64,
	commitmentBytes [][]byte,
	myCommitmentIndices []uint64,
	sndOutputs []*privacy.Scalar) *TxPrivacyInitParamsForASM {

	txParam := TxPrivacyInitParams{
		senderSK:    senderSK,
		paymentInfo: paymentInfo,
		inputCoins:  inputCoins,
		fee:         fee,
		hasPrivacy:  hasPrivacy,
		tokenID:     tokenID,
		metaData:    metaData,
		info:        info,
	}
	params := &TxPrivacyInitParamsForASM{
		txParam:             txParam,
		commitmentIndices:   commitmentIndices,
		commitmentBytes:     commitmentBytes,
		myCommitmentIndices: myCommitmentIndices,
		sndOutputs:          sndOutputs,
	}
	return params
}

func (param *TxPrivacyInitParamsForASM) SetMetaData(meta metadata.Metadata) {
	param.txParam.metaData = meta
}

func (tx *Tx) InitForASM(params *TxPrivacyInitParamsForASM, serverTime int64) error {
	//Logger.log.Debugf("CREATING TX........\n")
	tx.Version = txVersion
	var err error

	if len(params.txParam.inputCoins) > 255 {
		return NewTransactionErr(InputCoinIsVeryLargeError, nil, strconv.Itoa(len(params.txParam.inputCoins)))
	}

	if len(params.txParam.paymentInfo) > 254 {
		return NewTransactionErr(PaymentInfoIsVeryLargeError, nil, strconv.Itoa(len(params.txParam.paymentInfo)))
	}

	if params.txParam.tokenID == nil {
		// using default PRV
		params.txParam.tokenID = &common.Hash{}
		err := params.txParam.tokenID.SetBytes(common.PRVCoinID[:])
		if err != nil {
			return NewTransactionErr(TokenIDInvalidError, err, params.txParam.tokenID.String())
		}
	}

	// Calculate execution time
	//start := time.Now()

	if tx.LockTime == 0 {
		tx.LockTime = serverTime
	}

	// create sender's key set from sender's spending key
	senderFullKey := incognitokey.KeySet{}
	err = senderFullKey.InitFromPrivateKey(params.txParam.senderSK)
	if err != nil {
		Logger.log.Error(errors.New(fmt.Sprintf("Can not import Private key for sender keyset from %+v", params.txParam.senderSK)))
		return NewTransactionErr(PrivateKeySenderInvalidError, err)
	}
	// get public key last byte of sender
	pkLastByteSender := senderFullKey.PaymentAddress.Pk[len(senderFullKey.PaymentAddress.Pk)-1]

	// init info of tx
	tx.Info = []byte{}
	if len(params.txParam.info) > 0 {
		tx.Info = params.txParam.info
	}

	// set metadata
	tx.Metadata = params.txParam.metaData

	// set tx type
	tx.Type = common.TxNormalType
	//Logger.log.Debugf("len(inputCoins), fee, hasPrivacy: %d, %d, %v\n", len(params.inputCoins), params.fee, params.hasPrivacy)

	if len(params.txParam.inputCoins) == 0 && params.txParam.fee == 0 && !params.txParam.hasPrivacy {
		//Logger.log.Debugf("len(inputCoins) == 0 && fee == 0 && !hasPrivacy\n")
		tx.Fee = params.txParam.fee
		tx.sigPrivKey = *params.txParam.senderSK
		tx.PubKeyLastByteSender = pkLastByteSender
		err := tx.signTx()
		if err != nil {
			Logger.log.Error(errors.New(fmt.Sprintf("Cannot sign tx %v\n", err)))
			return NewTransactionErr(SignTxError, err)
		}
		return nil
	}

	shardID := common.GetShardIDFromLastByte(pkLastByteSender)

	if params.txParam.hasPrivacy {
		// Check number of list of random commitments, list of random commitment indices
		if len(params.commitmentIndices) != len(params.txParam.inputCoins)*privacy.CommitmentRingSize {
			return NewTransactionErr(RandomCommitmentError, nil)
		}

		if len(params.myCommitmentIndices) != len(params.txParam.inputCoins) {
			return NewTransactionErr(RandomCommitmentError, errors.New("number of list my commitment indices must be equal to number of input coins"))
		}
	}

	// Calculate execution time for creating payment proof
	//startPrivacy := time.Now()

	// Calculate sum of all output coins' value
	sumOutputValue := uint64(0)
	for _, p := range params.txParam.paymentInfo {
		sumOutputValue += p.Amount
	}

	// Calculate sum of all input coins' value
	sumInputValue := uint64(0)
	for _, coin := range params.txParam.inputCoins {
		sumInputValue += coin.CoinDetails.GetValue()
	}
	//Logger.log.Debugf("sumInputValue: %d\n", sumInputValue)

	// Calculate over balance, it will be returned to sender
	overBalance := int64(sumInputValue - sumOutputValue - params.txParam.fee)

	// Check if sum of input coins' value is at least sum of output coins' value and tx fee
	if overBalance < 0 {
		return NewTransactionErr(WrongInputError,
			errors.New(
				fmt.Sprintf("input value less than output value. sumInputValue=%d sumOutputValue=%d fee=%d",
					sumInputValue, sumOutputValue, params.txParam.fee)))
	}

	// if overBalance > 0, create a new payment info with pk is sender's pk and amount is overBalance
	if overBalance > 0 {
		changePaymentInfo := new(privacy.PaymentInfo)
		changePaymentInfo.Amount = uint64(overBalance)
		changePaymentInfo.PaymentAddress = senderFullKey.PaymentAddress
		params.txParam.paymentInfo = append(params.txParam.paymentInfo, changePaymentInfo)
	}

	// create new output coins
	outputCoins := make([]*privacy.OutputCoin, len(params.txParam.paymentInfo))

	// create SNDs for output coins
	sndOuts := params.sndOutputs

	// create new output coins with info: Pk, value, last byte of pk, snd
	for i, pInfo := range params.txParam.paymentInfo {
		outputCoins[i] = new(privacy.OutputCoin)
		outputCoins[i].CoinDetails = new(privacy.Coin)
		outputCoins[i].CoinDetails.SetValue(pInfo.Amount)
		if len(pInfo.Message) > 0 {
			if len(pInfo.Message) > privacy.MaxSizeInfoCoin {
				return NewTransactionErr(ExceedSizeInfoOutCoinError, nil)
			}
			outputCoins[i].CoinDetails.SetInfo(pInfo.Message)
		}

		PK, err := new(privacy.Point).FromBytesS(pInfo.PaymentAddress.Pk)
		if err != nil {
			Logger.log.Error(errors.New(fmt.Sprintf("can not decompress public key from %+v", pInfo.PaymentAddress)))
			return NewTransactionErr(DecompressPaymentAddressError, err, pInfo.PaymentAddress)
		}
		outputCoins[i].CoinDetails.SetPublicKey(PK)
		outputCoins[i].CoinDetails.SetSNDerivator(sndOuts[i])
	}

	// assign fee tx
	tx.Fee = params.txParam.fee

	// create zero knowledge proof of payment
	tx.Proof = &zkp.PaymentProof{}

	// get list of commitments for proving one-out-of-many from commitmentIndexs
	commitmentProving := make([]*privacy.Point, len(params.commitmentBytes))
	for i, cmBytes := range params.commitmentBytes {
		commitmentProving[i] = new(privacy.Point)
		commitmentProving[i], err = commitmentProving[i].FromBytesS(cmBytes)
		if err != nil {
			Logger.log.Error(errors.New(fmt.Sprintf("can not get commitment from index=%d shardID=%+v value=%+v", params.commitmentIndices[i], shardID, cmBytes)))
			return NewTransactionErr(CanNotDecompressCommitmentFromIndexError, err, params.commitmentIndices[i], shardID, cmBytes)
		}
	}

	// prepare witness for proving
	witness := new(zkp.PaymentWitness)
	paymentWitnessParam := zkp.PaymentWitnessParam{
		HasPrivacy:              params.txParam.hasPrivacy,
		PrivateKey:              new(privacy.Scalar).FromBytesS(*params.txParam.senderSK),
		InputCoins:              params.txParam.inputCoins,
		OutputCoins:             outputCoins,
		PublicKeyLastByteSender: pkLastByteSender,
		Commitments:             commitmentProving,
		CommitmentIndices:       params.commitmentIndices,
		MyCommitmentIndices:     params.myCommitmentIndices,
		Fee:                     params.txParam.fee,
	}
	err = witness.Init(paymentWitnessParam)
	if err.(*privacy.PrivacyError) != nil {
		Logger.log.Error(err)
		jsonParam, _ := json.MarshalIndent(paymentWitnessParam, common.EmptyString, "  ")
		return NewTransactionErr(InitWithnessError, err, string(jsonParam))
	}

	tx.Proof, err = witness.Prove(params.txParam.hasPrivacy)
	if err.(*privacy.PrivacyError) != nil {
		Logger.log.Error(err)
		jsonParam, _ := json.MarshalIndent(paymentWitnessParam, common.EmptyString, "  ")
		return NewTransactionErr(WithnessProveError, err, params.txParam.hasPrivacy, string(jsonParam))
	}

	//Logger.log.Debugf("DONE PROVING........\n")

	// set private key for signing tx
	if params.txParam.hasPrivacy {
		randSK := witness.GetRandSecretKey()
		tx.sigPrivKey = append(*params.txParam.senderSK, randSK.ToBytesS()...)

		// encrypt coin details (Randomness)
		// hide information of output coins except coin commitments, public key, snDerivators
		for i := 0; i < len(tx.Proof.GetOutputCoins()); i++ {
			err = tx.Proof.GetOutputCoins()[i].Encrypt(params.txParam.paymentInfo[i].PaymentAddress.Tk)
			if err.(*privacy.PrivacyError) != nil {
				Logger.log.Error(err)
				return NewTransactionErr(EncryptOutputError, err)
			}
			tx.Proof.GetOutputCoins()[i].CoinDetails.SetSerialNumber(nil)
			tx.Proof.GetOutputCoins()[i].CoinDetails.SetValue(0)
			tx.Proof.GetOutputCoins()[i].CoinDetails.SetRandomness(nil)
		}

		// hide information of input coins except serial number of input coins
		for i := 0; i < len(tx.Proof.GetInputCoins()); i++ {
			tx.Proof.GetInputCoins()[i].CoinDetails.SetCoinCommitment(nil)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetValue(0)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetSNDerivator(nil)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetPublicKey(nil)
			tx.Proof.GetInputCoins()[i].CoinDetails.SetRandomness(nil)
		}

	} else {
		tx.sigPrivKey = []byte{}
		randSK := big.NewInt(0)
		tx.sigPrivKey = append(*params.txParam.senderSK, randSK.Bytes()...)
	}

	// sign tx
	tx.PubKeyLastByteSender = pkLastByteSender
	err = tx.signTx()
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(SignTxError, err)
	}

	snProof := tx.Proof.GetSerialNumberProof()
	for i := 0; i < len(snProof); i++ {
		res, _ := snProof[i].Verify(nil)
		println("Verify serial number proof: ", i, ": ", res)
	}

	//elapsedPrivacy := time.Since(startPrivacy)
	//elapsed := time.Since(start)
	//Logger.log.Debugf("Creating payment proof time %s", elapsedPrivacy)
	//Logger.log.Debugf("Successfully Creating normal tx %+v in %s time", *tx.Hash(), elapsed)
	return nil
}

// GetFullTxValues returns both prv and ptoken values
func (tx Tx) GetFullTxValues() (uint64, uint64) {
	return tx.CalculateTxValue(), 0
}

// IsFullBurning returns whether the tx is full burning tx
func (tx Tx) IsFullBurning(
	bcr metadata.ChainRetriever,
	retriever metadata.ShardViewRetriever,
	viewRetriever metadata.BeaconViewRetriever,
	beaconHeight uint64,
) bool {
	return tx.IsCoinsBurning(bcr, retriever, viewRetriever, beaconHeight)
}
