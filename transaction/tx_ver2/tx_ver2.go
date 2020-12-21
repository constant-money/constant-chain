package tx_ver2

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/wallet"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/transaction/tx_generic"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
)

// TxSigPubKey of ver2 is array of Indexes in database
type SigPubKey struct {
	Indexes [][]*big.Int
}

type Tx struct {
	tx_generic.TxBase
}

func (sigPub SigPubKey) Bytes() ([]byte, error) {
	n := len(sigPub.Indexes)
	if n == 0 {
		return nil, errors.New("TxSigPublicKeyVer2.ToBytes: Indexes is empty")
	}
	if n > utils.MaxSizeByte {
		return nil, errors.New("TxSigPublicKeyVer2.ToBytes: Indexes is too large, too many rows")
	}
	m := len(sigPub.Indexes[0])
	if m > utils.MaxSizeByte {
		return nil, errors.New("TxSigPublicKeyVer2.ToBytes: Indexes is too large, too many columns")
	}
	for i := 1; i < n; i += 1 {
		if len(sigPub.Indexes[i]) != m {
			return nil, errors.New("TxSigPublicKeyVer2.ToBytes: Indexes is not a rectangle array")
		}
	}

	b := make([]byte, 0)
	b = append(b, byte(n))
	b = append(b, byte(m))
	for i := 0; i < n; i += 1 {
		for j := 0; j < m; j += 1 {
			currentByte := sigPub.Indexes[i][j].Bytes()
			lengthByte := len(currentByte)
			if lengthByte > utils.MaxSizeByte {
				return nil, errors.New("TxSigPublicKeyVer2.ToBytes: IndexesByte is too large")
			}
			b = append(b, byte(lengthByte))
			b = append(b, currentByte...)
		}
	}
	return b, nil
}

func (sigPub *SigPubKey) SetBytes(b []byte) error {
	if len(b) < 2 {
		return errors.New("txSigPubKeyFromBytes: cannot parse length of Indexes, length of input byte is too small")
	}
	n := int(b[0])
	m := int(b[1])
	offset := 2
	indexes := make([][]*big.Int, n)
	for i := 0; i < n; i += 1 {
		row := make([]*big.Int, m)
		for j := 0; j < m; j += 1 {
			if offset >= len(b) {
				return errors.New("txSigPubKeyFromBytes: cannot parse byte length of index[i][j], length of input byte is too small")
			}
			byteLength := int(b[offset])
			offset += 1
			if offset+byteLength > len(b) {
				return errors.New("txSigPubKeyFromBytes: cannot parse big int index[i][j], length of input byte is too small")
			}
			currentByte := b[offset : offset+byteLength]
			offset += byteLength
			row[j] = new(big.Int).SetBytes(currentByte)
		}
		indexes[i] = row
	}
	if sigPub == nil {
		sigPub = new(SigPubKey)
	}
	sigPub.Indexes = indexes
	return nil
}

// ========== GET FUNCTION ===========

func (tx *Tx) GetReceiverData() ([]privacy.Coin, error) {
	if tx.Proof != nil && len(tx.Proof.GetOutputCoins()) > 0 {
		return tx.Proof.GetOutputCoins(), nil
	}
	return nil, nil
}

// ========== CHECK FUNCTION ===========

func (tx *Tx) CheckAuthorizedSender(publicKey []byte) (bool, error) {
	if !tx.Metadata.ShouldSignMetaData() {
		utils.Logger.Log.Error("Check authorized sender failed because tx.Metadata is not appropriate")
		return false, errors.New("Check authorized sender failed because tx.Metadata is not appropriate")
	}
	//meta, ok := tx.Metadata.(*metadata.StopAutoStakingMetadata)
	//if !ok {
	//	utils.Logger.Log.Error("Check authorized sender failed because tx.Metadata is not correct type")
	//	return false, errors.New("Check authorized sender failed because tx.Metadata is not correct type")
	//}
	metaSig := tx.Metadata.GetSig()
	fmt.Println("Metadata Signature", metaSig)
	if metaSig == nil || len(metaSig) == 0 {
		utils.Logger.Log.Error("CheckAuthorizedSender: should have sig for metadata to verify")
		return false, errors.New("CheckAuthorizedSender should have sig for metadata to verify")
	}
	/****** verify Schnorr signature *****/
	verifyKey := new(privacy.SchnorrPublicKey)
	metaSigPublicKey, err := new(privacy.Point).FromBytesS(publicKey)
	if err != nil {
		utils.Logger.Log.Error(err)
		return false, utils.NewTransactionErr(utils.DecompressSigPubKeyError, err)
	}
	verifyKey.Set(metaSigPublicKey)

	signature := new(privacy.SchnSignature)
	if err := signature.SetBytes(metaSig); err != nil {
		newErr := utils.NewTransactionErr(utils.InitTxSignatureFromBytesError, err)
		utils.Logger.Log.Errorf("Metadata Signature Invalid - ", newErr)
		return false, newErr
	}
	fmt.Println("[CheckAuthorizedSender] Metadata Signature - Validate OK")
	return verifyKey.Verify(signature, tx.HashWithoutMetadataSig()[:]), nil
}

// ========== NORMAL INIT FUNCTIONS ==========

func createPrivKeyMlsag(inputCoins []privacy.PlainCoin, outputCoins []*privacy.CoinV2, senderSK *privacy.PrivateKey) ([]*privacy.Scalar, error) {
	sumRand := new(privacy.Scalar).FromUint64(0)
	for _, in := range inputCoins {
		sumRand.Add(sumRand, in.GetRandomness())
	}
	for _, out := range outputCoins {
		sumRand.Sub(sumRand, out.GetRandomness())
	}

	privKeyMlsag := make([]*privacy.Scalar, len(inputCoins)+1)
	for i := 0; i < len(inputCoins); i += 1 {
		var err error
		privKeyMlsag[i], err = inputCoins[i].ParsePrivateKeyOfCoin(*senderSK)
		if err != nil {
			utils.Logger.Log.Errorf("Cannot parse private key of coin %v", err)
			return nil, err
		}
	}
	privKeyMlsag[len(inputCoins)] = sumRand
	return privKeyMlsag, nil
}

func (tx *Tx) Init(paramsInterface interface{}) error {
	params, ok := paramsInterface.(*tx_generic.TxPrivacyInitParams)
	if !ok {
		return errors.New("params of tx Init is not TxPrivacyInitParam")
	}

	jsb, _ := json.Marshal(params)
	utils.Logger.Log.Infof("Create TX v2 with params %s", string(jsb))
	if err := tx_generic.ValidateTxParams(params); err != nil {
		return err
	}

	// Init tx and params (tx and params will be changed)
	if err := tx.InitializeTxAndParams(params); err != nil {
		return err
	}

	// Check if this tx is nonPrivacyNonInput
	// Case 1: tx ptoken transfer with ptoken fee
	// Case 2: tx Reward
	// If it is non privacy non input then return
	if check, err := tx.IsNonPrivacyNonInput(params); check {
		return err
	}
	if err := tx.prove(params); err != nil {
		return err
	}
	jsb, _ = json.Marshal(tx)
	utils.Logger.Log.Infof("TX Creation complete ! The resulting transaction is : %s", string(jsb))
	txSize := tx.GetTxActualSize()
	if txSize > common.MaxTxSize {
		return utils.NewTransactionErr(utils.ExceedSizeTx, nil, strconv.Itoa(int(txSize)))
	}

	return nil
}

func (tx *Tx) signOnMessage(inp []privacy.PlainCoin, out []*privacy.CoinV2, params *tx_generic.TxPrivacyInitParams, hashedMessage []byte) error {
	if tx.Sig != nil {
		return utils.NewTransactionErr(utils.UnexpectedError, errors.New("input transaction must be an unsigned one"))
	}
	ringSize := privacy.RingSize

	// Generate Ring
	piBig,piErr := common.RandBigIntMaxRange(big.NewInt(int64(ringSize)))
	if piErr!=nil{
		return piErr
	}
	var pi int = int(piBig.Int64())
	shardID := common.GetShardIDFromLastByte(tx.PubKeyLastByteSender)
	ring, indexes, err := generateMlsagRingWithIndexes(inp, out, params, pi, shardID, ringSize)
	if err != nil {
		utils.Logger.Log.Errorf("generateMlsagRingWithIndexes got error %v ", err)
		return err
	}

	// Set SigPubKey
	txSigPubKey := new(SigPubKey)
	txSigPubKey.Indexes = indexes
	tx.SigPubKey, err = txSigPubKey.Bytes()
	if err != nil {
		utils.Logger.Log.Errorf("tx.SigPubKey cannot parse from Bytes, error %v ", err)
		return err
	}

	// Set sigPrivKey
	privKeysMlsag, err := createPrivKeyMlsag(inp, out, params.SenderSK)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot create private key of mlsag: %v", err)
		return err
	}
	sag := mlsag.NewMlsag(privKeysMlsag, ring, pi)
	sk, err := privacy.ArrayScalarToBytes(&privKeysMlsag)
	if err != nil {
		utils.Logger.Log.Errorf("tx.SigPrivKey cannot parse arrayScalar to Bytes, error %v ", err)
		return err
	}
	tx.SetPrivateKey(sk)

	// Set Signature
	mlsagSignature, err := sag.Sign(hashedMessage)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot signOnMessage mlsagSignature, error %v ", err)
		return err
	}
	// inputCoins already hold keyImage so set to nil to reduce size
	mlsagSignature.SetKeyImages(nil)
	tx.Sig, err = mlsagSignature.ToBytes()

	return err
}

func (tx *Tx) signMetadata(privateKey *privacy.PrivateKey) error {
	// signOnMessage meta data
	metaSig := tx.Metadata.GetSig()
	if metaSig != nil && len(metaSig) > 0 {
		return utils.NewTransactionErr(utils.UnexpectedError, errors.New("meta.Sig should be empty or nil"))
	}

	/****** using Schnorr signature *******/
	sk := new(privacy.Scalar).FromBytesS(*privateKey)
	r := new(privacy.Scalar).FromUint64(0)
	sigKey := new(privacy.SchnorrPrivateKey)
	sigKey.Set(sk, r)

	// signing
	signature, err := sigKey.Sign(tx.HashWithoutMetadataSig()[:])
	if err != nil {
		return err
	}

	// convert signature to byte array
	tx.Metadata.SetSig(signature.Bytes())
	fmt.Println("Signature Detail", tx.Metadata.GetSig())
	return nil
}

func (tx *Tx) prove(params *tx_generic.TxPrivacyInitParams) error {
	outputCoins, err := utils.NewCoinV2ArrayFromPaymentInfoArray(params.PaymentInfo, params.TokenID, params.StateDB)
	if err != nil {
		utils.Logger.Log.Errorf("Cannot parse outputCoinV2 to outputCoins, error %v ", err)
		return err
	}

	// inputCoins is plainCoin because it may have coinV1 with coinV2
	inputCoins := params.InputCoins

	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, nil, false, params.PaymentInfo)
	if err != nil {
		utils.Logger.Log.Errorf("Error in privacy_v2.Prove, error %v ", err)
		return err
	}

	if tx.ShouldSignMetaData() {
		if err := tx.signMetadata(params.SenderSK); err != nil {
			utils.Logger.Log.Error("Cannot signOnMessage txMetadata in shouldSignMetadata")
			return err
		}
	}
	err = tx.signOnMessage(inputCoins, outputCoins, params, tx.Hash()[:])
	return err
}

// func (tx *Tx) proveASM(params *tx_generic.TxPrivacyInitParamsForASM) error {
// 	return tx.prove(&params.txParam)
// }

// Retrieve ring from database using sigpubkey and last column commitment (last column = sumOutputCoinCommitment + fee)
func getRingFromSigPubKeyAndLastColumnCommitment(sigPubKey []byte, sumOutputsWithFee *privacy.Point, transactionStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (*mlsag.Ring, error) {
	txSigPubKey := new(SigPubKey)
	if err := txSigPubKey.SetBytes(sigPubKey); err != nil {
		errStr := fmt.Sprintf("Error when parsing bytes of txSigPubKey %v", err)
		return nil, utils.NewTransactionErr(utils.UnexpectedError, errors.New(errStr))
	}
	indexes := txSigPubKey.Indexes
	n := len(indexes)
	if n == 0 {
		return nil, errors.New("Cannot get ring from Indexes: Indexes is empty")
	}

	m := len(indexes[0])

	ring := make([][]*privacy.Point, n)
	for i := 0; i < n; i += 1 {
		sumCommitment := new(privacy.Point).Identity()
		sumCommitment.Sub(sumCommitment, sumOutputsWithFee)
		row := make([]*privacy.Point, m+1)
		for j := 0; j < m; j += 1 {
			index := indexes[i][j]
			randomCoinBytes, err := statedb.GetOTACoinByIndex(transactionStateDB, *tokenID, index.Uint64(), shardID)
			if err != nil {
				utils.Logger.Log.Errorf("Get random onetimeaddresscoin error %v ", err)
				return nil, err
			}
			randomCoin := new(privacy.CoinV2)
			if err := randomCoin.SetBytes(randomCoinBytes); err != nil {
				utils.Logger.Log.Errorf("Set coin Byte error %v ", err)
				return nil, err
			}
			row[j] = randomCoin.GetPublicKey()
			sumCommitment.Add(sumCommitment, randomCoin.GetCommitment())
		}
		row[m] = new(privacy.Point).Set(sumCommitment)
		ring[i] = row
	}
	return mlsag.NewRing(ring), nil
}

// ========== NORMAL VERIFY FUNCTIONS ==========

func generateMlsagRingWithIndexes(inputCoins []privacy.PlainCoin, outputCoins []*privacy.CoinV2, params *tx_generic.TxPrivacyInitParams, pi int, shardID byte, ringSize int) (*mlsag.Ring, [][]*big.Int, error) {
	lenOTA, err := statedb.GetOTACoinLength(params.StateDB, *params.TokenID, shardID)
	if err != nil || lenOTA == nil {
		utils.Logger.Log.Errorf("Getting length of commitment error, either database length ota is empty or has error, error = %v", err)
		return nil, nil, err
	}
	outputCoinsAsGeneric := make([]privacy.Coin, len(outputCoins))
	for i:=0;i<len(outputCoins);i++{
		outputCoinsAsGeneric[i] = outputCoins[i]
	}
	sumOutputsWithFee := tx_generic.CalculateSumOutputsWithFee(outputCoinsAsGeneric, params.Fee)
	indexes := make([][]*big.Int, ringSize)
	ring := make([][]*privacy.Point, ringSize)
	for i := 0; i < ringSize; i += 1 {
		sumInputs := new(privacy.Point).Identity()
		sumInputs.Sub(sumInputs, sumOutputsWithFee)

		row := make([]*privacy.Point, len(inputCoins))
		rowIndexes := make([]*big.Int, len(inputCoins))
		if i == pi {
			for j := 0; j < len(inputCoins); j += 1 {
				row[j] = inputCoins[j].GetPublicKey()
				publicKeyBytes := inputCoins[j].GetPublicKey().ToBytesS()
				if rowIndexes[j], err = statedb.GetOTACoinIndex(params.StateDB, *params.TokenID, publicKeyBytes); err != nil {
					utils.Logger.Log.Errorf("Getting commitment index error %v ", err)
					return nil, nil, err
				}
				sumInputs.Add(sumInputs, inputCoins[j].GetCommitment())
			}
		} else {
			for j := 0; j < len(inputCoins); j += 1 {
				rowIndexes[j], _ = common.RandBigIntMaxRange(lenOTA)
				coinBytes, err := statedb.GetOTACoinByIndex(params.StateDB, *params.TokenID, rowIndexes[j].Uint64(), shardID)
				if err != nil {
					utils.Logger.Log.Errorf("Get coinv2 by index error %v ", err)
					return nil, nil, err
				}
				coinDB := new(privacy.CoinV2)
				if err := coinDB.SetBytes(coinBytes); err != nil {
					utils.Logger.Log.Errorf("Cannot parse coinv2 byte error %v ", err)
					return nil, nil, err
				}
				row[j] = coinDB.GetPublicKey()
				sumInputs.Add(sumInputs, coinDB.GetCommitment())
			}
		}
		row = append(row, sumInputs)
		ring[i] = row
		indexes[i] = rowIndexes
	}
	return mlsag.NewRing(ring), indexes, nil
}

func getMLSAGSigFromTxSigAndKeyImages(txSig []byte, keyImages []*privacy.Point) (*mlsag.MlsagSig, error) {
	mlsagSig, err := new(mlsag.MlsagSig).FromBytes(txSig)
	if err != nil {
		utils.Logger.Log.Errorf("Has error when converting byte to mlsag signature, err: %v", err)
		return nil, err
	}

	return mlsag.NewMlsagSig(mlsagSig.GetC(), keyImages, mlsagSig.GetR())
}

func (tx *Tx) verifySig(transactionStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash, isNewTransaction bool) (bool, error) {
	// check input transaction
	if tx.Sig == nil || tx.SigPubKey == nil {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("input transaction must be a signed one"))
	}
	var err error

	// Reform Ring
	sumOutputsWithFee := tx_generic.CalculateSumOutputsWithFee(tx.Proof.GetOutputCoins(), tx.Fee)
	ring, err := getRingFromSigPubKeyAndLastColumnCommitment(tx.SigPubKey, sumOutputsWithFee, transactionStateDB, shardID, tokenID)
	if err != nil {
		utils.Logger.Log.Errorf("Error when querying database to construct mlsag ring: %v ", err)
		return false, err
	}

	// Reform MLSAG Signature
	inputCoins := tx.Proof.GetInputCoins()
	keyImages := make([]*privacy.Point, len(inputCoins)+1)
	for i := 0; i < len(inputCoins); i += 1 {
		if inputCoins[i].GetKeyImage()==nil {
			utils.Logger.Log.Errorf("Error when reconstructing mlsagSignature: missing keyImage")
			return false, err
		}
		keyImages[i] = inputCoins[i].GetKeyImage()
	}
	// The last column is gone, so just fill in any value
	keyImages[len(inputCoins)] = privacy.RandomPoint()
	mlsagSignature, err := getMLSAGSigFromTxSigAndKeyImages(tx.Sig, keyImages)
	if err != nil {
		return false, err
	}

	return mlsag.Verify(mlsagSignature, ring, tx.Hash()[:])
}

func (tx *Tx) Verify(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, error) {
	var err error
	var valid bool
	if tokenID, err = tx_generic.ParseTokenID(tokenID); err != nil {
		return false, err
	}
	proofAsV2, ok := tx.GetProof().(*privacy.ProofV2)
	if !ok{
		utils.Logger.Log.Errorf("Error in tx %s : ver2 transaction cannot have proofs of any other version - %v", tx.Hash().String(), err)
		return false, utils.NewTransactionErr(utils.UnexpectedError, err)
	}

	isNewTransaction, ok := boolParams["isNewTransaction"]
	if !ok {
		isNewTransaction = false
	}

	isConfAsset, err := proofAsV2.IsConfidentialAsset()
	if err!=nil{
		utils.Logger.Log.Errorf("Error in tx %s : proof is invalid due to inconsistent asset tags - %v", tx.Hash().String(), err)
		return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
	}
	if isConfAsset{
		valid, err = tx.verifySigCA(transactionStateDB, shardID, tokenID, isNewTransaction)
	}else{
		valid, err = tx.verifySig(transactionStateDB, shardID, tokenID, isNewTransaction)
	}
	if !valid {
		// fmt.Printf("Fail with CA = %v and tokenID = %s\n", isConfAsset, tokenID.String())
		if err != nil {
			utils.Logger.Log.Errorf("Error verifying signature ver2 with tx hash %s: %+v \n", tx.Hash().String(), err)
			return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
		}
		utils.Logger.Log.Errorf("FAILED VERIFICATION SIGNATURE ver2 with tx hash %s", tx.Hash().String())
		return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, fmt.Errorf("FAILED VERIFICATION SIGNATURE ver2 with tx hash %s", tx.Hash().String()))
	}

	boolParams["hasConfidentialAsset"] = isConfAsset

	if valid, err := tx.Proof.Verify(boolParams, tx.SigPubKey, tx.Fee, shardID, tokenID, nil); !valid {
		if err != nil {
			utils.Logger.Log.Error(err)
		}
		utils.Logger.Log.Error("FAILED VERIFICATION PAYMENT PROOF VER 2")
		err1, ok := err.(*privacy.PrivacyError)
		if ok {
			// parse error detail
			if err1.Code == privacy.ErrCodeMessage[errhandler.VerifyOneOutOfManyProofFailedErr].Code {
				if isNewTransaction {
					return false, utils.NewTransactionErr(utils.VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
				} else {
					// for old txs which be get from sync block or validate new block
					if tx.LockTime <= utils.ValidateTimeForOneoutOfManyProof {
						// only verify by signOnMessage on block because of issue #504(that mean we should pass old tx, which happen before this issue)
						return true, nil
					} else {
						return false, utils.NewTransactionErr(utils.VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
					}
				}
			}
		}
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, err, tx.Hash().String())
	}
	utils.Logger.Log.Debugf("SUCCESSED VERIFICATION PAYMENT PROOF ")
	return true, nil
}

func (tx Tx) VerifyMinerCreatedTxBeforeGettingInBlock(mintdata *metadata.MintData, shardID byte, bcr metadata.ChainRetriever, accumulatedValues *metadata.AccumulatedValues, retriever metadata.ShardViewRetriever, viewRetriever metadata.BeaconViewRetriever) (bool, error) {
	return tx_generic.VerifyTxCreatedByMiner(&tx, mintdata, shardID, bcr, accumulatedValues, retriever, viewRetriever)
}

// ========== SALARY FUNCTIONS: INIT AND VALIDATE  ==========

func (tx *Tx) InitTxSalary(otaCoin *privacy.CoinV2, privateKey *privacy.PrivateKey, stateDB *statedb.StateDB, metaData metadata.Metadata) error {
	tokenID := &common.Hash{}
	if err := tokenID.SetBytes(common.PRVCoinID[:]); err != nil {
		return utils.NewTransactionErr(utils.TokenIDInvalidError, err, tokenID.String())
	}
	if found, err := statedb.HasOnetimeAddress(stateDB, *tokenID, otaCoin.GetPublicKey().ToBytesS()); found || err != nil {
		if found {
			return errors.New("Cannot initTxSalary, onetimeaddress already exists in database")
		}
		if err != nil {
			errStr := fmt.Sprintf("Checking onetimeaddress existence in database get error %v", err)
			return errors.New(errStr)
		}
	}

	tx.Version = utils.TxVersion2Number
	tx.Type = common.TxRewardType
	if tx.LockTime == 0 {
		tx.LockTime = time.Now().Unix()
	}

	tempOutputCoin := []privacy.Coin{otaCoin}
	proof := new(privacy.ProofV2)
	proof.Init()
	proof.SetOutputCoins(tempOutputCoin)
	tx.Proof = proof

	publicKeyBytes := otaCoin.GetPublicKey().ToBytesS()
	tx.PubKeyLastByteSender = publicKeyBytes[len(publicKeyBytes)-1]

	// signOnMessage Tx using ver1 schnorr
	tx.SetPrivateKey(*privateKey)
	tx.SetMetadata(metaData)

	var err error
	if tx.Sig, tx.SigPubKey, err = tx_generic.SignNoPrivacy(privateKey, tx.Hash()[:]); err != nil {
		return utils.NewTransactionErr(utils.SignTxError, err)
	}
	return nil
}

func (tx *Tx) ValidateTxSalary(db *statedb.StateDB) (bool, error) {
	// verify signature
	if valid, err := tx_generic.VerifySigNoPrivacy(tx.Sig, tx.SigPubKey, tx.Hash()[:]); !valid {
		if err != nil {
			utils.Logger.Log.Debugf("Error verifying signature of tx: %+v", err)
			return false, utils.NewTransactionErr(utils.VerifyTxSigFailError, err)
		}
		return false, nil
	}
	// check whether output coin's input exists in input list or not
	tokenID := &common.Hash{}
	if err := tokenID.SetBytes(common.PRVCoinID[:]); err != nil {
		return false, utils.NewTransactionErr(utils.TokenIDInvalidError, err, tokenID.String())
	}

	// Check commitment
	outputCoins := tx.Proof.GetOutputCoins()
	if len(outputCoins) != 1 {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("length outputCoins of proof is not 1"))
	}
	outputCoin := outputCoins[0].(*privacy.CoinV2)
	cmpCommitment := privacy.PedCom.CommitAtIndex(outputCoin.GetAmount(), outputCoin.GetRandomness(), privacy.PedersenValueIndex)
	if !privacy.IsPointEqual(cmpCommitment, outputCoin.GetCommitment()) {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("check output coin's coin commitment isn't calculated correctly"))
	}

	// Check shardID
	coinShardID, errShard := outputCoin.GetShardID()
	if errShard != nil {
		errStr := fmt.Sprintf("error when getting coin shardID, err: %v", errShard)
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New(errStr))
	}
	if coinShardID != common.GetShardIDFromLastByte(tx.PubKeyLastByteSender) {
		return false, utils.NewTransactionErr(utils.UnexpectedError, errors.New("output coin's shardID is different from tx pubkey last byte"))
	}

	// Check database for ota
	found, err := statedb.HasOnetimeAddress(db, *tokenID, outputCoin.GetPublicKey().ToBytesS())
	if err != nil {
		utils.Logger.Log.Errorf("Cannot check public key existence in DB, err %v", err)
		return false, err
	}
	if found {
		utils.Logger.Log.Error("ValidateTxSalary got error: found onetimeaddress in database")
		return false, errors.New("found onetimeaddress in database")
	}
	return true, nil
}

// func (tx Tx) StringWithoutMetadataSig() string {
// 	record := strconv.Itoa(int(tx.Version))
// 	record += strconv.FormatInt(tx.LockTime, 10)
// 	record += strconv.FormatUint(tx.Fee, 10)
// 	if tx.Proof != nil {
// 		record += base64.StdEncoding.EncodeToString(tx.Proof.Bytes())
// 	}
// 	if tx.Metadata != nil {
// 		metadataHash := tx.Metadata.HashWithoutSig()
// 		record += metadataHash.String()
// 	}
// 	record += string(tx.Info)
// 	return record
// }

func (tx Tx) Hash() *common.Hash {
	// leave out signature & its public key when hashing tx
	tx.Sig = []byte{}
	tx.SigPubKey = []byte{}
	inBytes, err := json.Marshal(tx)
	if err!=nil{
		return nil
	}
	hash := common.HashH(inBytes)
	// after this returns, tx is restored since the receiver is not a pointer
	return &hash
}

func (tx Tx) HashWithoutMetadataSig() *common.Hash {
	md := tx.GetMetadata()
	mdHash := md.HashWithoutSig()
	tx.SetMetadata(nil)
	txHash := tx.Hash()
	if mdHash==nil || txHash==nil{
		return nil
	}
	// tx.SetMetadata(md)
	inBytes := append(mdHash[:], txHash[:]...)
	hash := common.HashH(inBytes)
	return &hash
}

// ========== VALIDATE FUNCTIONS ============

func (tx Tx) ValidateSanityData(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, beaconHeight uint64) (bool, error) {
	if tx.Proof == nil {
		return false, errors.New("Tx Privacy Ver 2 must have proof")
	}

	if check, err := tx_generic.ValidateSanity(&tx, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight); !check || err != nil {
		utils.Logger.Log.Errorf("Cannot check sanity of version, size, proof, type and info: err %v", err)
		return false, err
	}

	if check, err := tx_generic.MdValidateSanity(&tx, chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight); !check || err != nil {
		utils.Logger.Log.Errorf("Cannot check sanity of metadata: err %v", err)
		return false, err
	}

	return true, nil
}

// ========== SHARED FUNCTIONS ============

func (tx Tx) GetTxMintData() (bool, privacy.Coin, *common.Hash, error) { return tx_generic.GetTxMintData(&tx, &common.PRVCoinID) }

func (tx Tx) GetTxBurnData() (bool, privacy.Coin, *common.Hash, error) { return tx_generic.GetTxBurnData(&tx) }

func (tx Tx) GetTxFullBurnData() (bool, privacy.Coin, privacy.Coin, *common.Hash, error) {
	isBurn, burnedCoin, burnedToken, err := tx.GetTxBurnData()
	return isBurn, burnedCoin, nil, burnedToken, err
}

func (tx Tx) ValidateTxWithBlockChain(chainRetriever metadata.ChainRetriever, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever, shardID byte, stateDB *statedb.StateDB) error {
	err := tx_generic.MdValidateWithBlockChain(&tx, chainRetriever, shardViewRetriever, beaconViewRetriever, shardID, stateDB)
	if err!=nil{
		return err
	}
	return tx.TxBase.ValidateDoubleSpendWithBlockchain(shardID, stateDB, nil)
}

func (tx Tx) ValidateTransaction(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash) (bool, []privacy.Proof, error) {
	switch tx.GetType() {
	case common.TxRewardType:
		valid, err := tx.ValidateTxSalary(transactionStateDB)
		return valid, nil, err
	case common.TxReturnStakingType:
		return tx.ValidateTxReturnStaking(transactionStateDB), nil, nil
	case common.TxConversionType:
		valid, err := validateConversionVer1ToVer2(&tx, transactionStateDB, shardID, tokenID)
		return valid, nil ,err
	default:
		valid, err := tx.Verify(boolParams, transactionStateDB, bridgeStateDB, shardID, tokenID)
		resultProofs := []privacy.Proof{}
		isBatch, ok := boolParams["isBatch"]
		if !ok {
			isBatch = false
		}
		if isBatch{
			if tx.GetProof()!=nil{
				resultProofs = append(resultProofs, tx.GetProof())
			}
		}
		return valid, resultProofs, err
	}
	// return validateTransaction(&tx, hasPrivacy, transactionStateDB, bridgeStateDB, shardID, tokenID, isBatch, isNewTransaction)
}

func (tx Tx) ValidateTxByItself(boolParams map[string]bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, chainRetriever metadata.ChainRetriever, shardID byte, shardViewRetriever metadata.ShardViewRetriever, beaconViewRetriever metadata.BeaconViewRetriever) (bool, error) {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return false, err
	}
	valid, _, err := tx.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, prvCoinID)
	if !valid {
		return false, err
	}

	hasPrivacy, ok := boolParams["hasPrivacy"]
	if !ok {
		hasPrivacy = false
	}
	valid, err = tx_generic.MdValidate(&tx, hasPrivacy, transactionStateDB, bridgeStateDB, shardID)
	if !valid {
		return false, err
	}
	return true, nil
}

func (tx Tx) GetTxActualSize() uint64 {
	jsb, err := json.Marshal(tx)
	if err!=nil{
		return 0
	}
	return uint64(math.Ceil(float64(len(jsb)) / 1024))
}

func (tx Tx) ListOTAHashH() []common.Hash {
	result := make([]common.Hash, 0)
	if tx.Proof != nil {
		for _, outputCoin := range tx.Proof.GetOutputCoins() {
			//Discard coins sent to the burning address
			if wallet.IsPublicKeyBurningAddress(outputCoin.GetPublicKey().ToBytesS()){
				continue
			}
			hash := common.HashH(outputCoin.GetPublicKey().ToBytesS())
			result = append(result, hash)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

func (tx Tx) validateDuplicateOTAsWithCurrentMempool(poolOTAHashH map[common.Hash][]common.Hash) error {
	if tx.Proof == nil {
		return nil
	}
	temp := make(map[common.Hash]interface{})
	for _, outputCoin := range tx.Proof.GetOutputCoins() {
		//Skip coins sent to the burning address
		if wallet.IsPublicKeyBurningAddress(outputCoin.GetPublicKey().ToBytesS()){
			continue
		}
		hash := common.HashH(outputCoin.GetPublicKey().ToBytesS())
		temp[hash] = nil
	}

	for _, listOTAs := range poolOTAHashH {
		for _, otaHash := range listOTAs {
			if _, ok := temp[otaHash]; ok {
				return fmt.Errorf("duplicate OTA with current mempool %v",
					otaHash.String())
			}
		}
	}
	return nil
}

func (tx Tx) ValidateTxWithCurrentMempool(mr metadata.MempoolRetriever) error {
	if tx.Proof == nil {
		return nil
	}

	//Check if any OTA has been produced in current mempool
	poolSNDOutputsHashH := mr.GetOTAHashH()
	err := tx.validateDuplicateOTAsWithCurrentMempool(poolSNDOutputsHashH)
	if err != nil {
		return err
	}

	//Check if any serial number has been used in current mempool
	temp := make(map[common.Hash]interface{})
	for _, desc := range tx.Proof.GetInputCoins() {
		hash := common.HashH(desc.GetKeyImage().ToBytesS())
		temp[hash] = nil
	}
	poolSerialNumbersHashH := mr.GetSerialNumbersHashH()
	for _, listSerialNumbers := range poolSerialNumbersHashH {
		for _, serialNumberHash := range listSerialNumbers {
			if _, ok := temp[serialNumberHash]; ok {
				return errors.New("double spend in mempool")
			}
		}
	}
	return nil
}

