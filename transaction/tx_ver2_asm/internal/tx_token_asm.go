package internal

import(
	"math/big"
	"errors"
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
)

const (
	CustomTokenInit = iota
	CustomTokenTransfer
	CustomTokenCrossShard
)

type TxToken struct {
	*Tx
	TxTokenData TxTokenData `json:"TxTokenPrivacyData"`
}

type TxTokenData struct {
	// TxNormal is the normal transaction, it will never be token transaction
	TxNormal       *Tx
	PropertyID     common.Hash // = hash of TxCustomTokenprivacy data
	PropertyName   string
	PropertySymbol string

	Type     int    // action type
	Mintable bool   // default false
	Amount   uint64 // init amount
}
func (txData TxTokenData) String() string {
	record := txData.PropertyName
	record += txData.PropertySymbol
	record += fmt.Sprintf("%d", txData.Amount)
	if txData.TxNormal.Proof != nil {
		inputCoins := txData.TxNormal.Proof.GetInputCoins()
		outputCoins := txData.TxNormal.Proof.GetOutputCoins()
		for _, out := range outputCoins {
			publicKeyBytes := []byte{}
			if out.GetPublicKey() != nil {
				publicKeyBytes = out.GetPublicKey().ToBytesS()
			}
			record += string(publicKeyBytes)
			record += strconv.FormatUint(out.GetValue(), 10)
		}
		for _, in := range inputCoins {
			publicKeyBytes := []byte{}
			if in.GetPublicKey() != nil {
				publicKeyBytes = in.GetPublicKey().ToBytesS()
			}
			record += string(publicKeyBytes)
			if in.GetValue() > 0 {
				record += strconv.FormatUint(in.GetValue(), 10)
			}
		}
	}
	return record
}
func (txData TxTokenData) Hash() (*common.Hash, error) {
	point := privacy.HashToPoint([]byte(txData.String()))
	hash := new(common.Hash)
	err := hash.SetBytes(point.ToBytesS())
	if err != nil {
		return nil, err
	}
	return hash, nil
}

type TxTokenParams struct {
	SenderKey          *privacy.PrivateKey
	PaymentInfo        []*privacy.PaymentInfo
	InputCoin          []privacy.PlainCoin
	FeeNativeCoin      uint64
	TokenParams        *TokenParam
	HasPrivacyCoin     bool
	HasPrivacyToken    bool
	ShardID            byte
	Info               []byte
}

// CustomTokenParamTx - use for rpc request json body
type TokenParam struct {
	PropertyID     string                 `json:"TokenID"`
	PropertyName   string                 `json:"TokenName"`
	PropertySymbol string                 `json:"TokenSymbol"`
	Amount         uint64                 `json:"TokenAmount"`
	TokenTxType    int                    `json:"TokenTxType"`
	Receiver       []*privacy.PaymentInfo `json:"TokenReceiver"`
	TokenInput     []privacy.PlainCoin    `json:"TokenInput"`
	Mintable       bool                   `json:"TokenMintable"`
	Fee            uint64                 `json:"TokenFee"`
}

type TokenInnerParams struct{
	TokenID 	string 					   `json:"token_id"`
	TokenPaymentInfo []privacy.PaymentInfo `json:"token_payments"`
	TokenInput	[]CoinInter				   `json:"token_input_coins"`
	TokenCache 	CoinCache 				   `json:"token_cache"`
}

func (params *TokenInnerParams) GetInputCoins() []privacy.PlainCoin{
	var result []privacy.PlainCoin
	for _,ci := range params.TokenInput{
		c := ci.GetCoinV2()
		result = append(result,c)
	}
	return result
}

func (params *TokenInnerParams) GetCompatTokenParams() *TokenParam{
	var tpInfos []*privacy.PaymentInfo
	for _, payInf := range params.TokenPaymentInfo{
		tpInfos = append(tpInfos, &payInf)
	}
	// var tid common.Hash 
	// tid.NewHashFromStr(params.TokenID)

	return &TokenParam{PropertyID: params.TokenID, PropertyName: "", PropertySymbol: "", Amount: 0, TokenTxType: CustomTokenTransfer, Receiver: tpInfos, TokenInput: params.GetInputCoins(), Mintable: false, Fee: 0}
}

func (params *InitParamsAsm) GetCompatTxTokenParams() *TxTokenParams{
	if params.TokenParams==nil{
		return nil
	}
	var pInfos []*privacy.PaymentInfo
	for _, payInf := range params.PaymentInfo{
		pInfos = append(pInfos, &payInf)
	}
	tp := params.TokenParams.GetCompatTokenParams()

	return &TxTokenParams{SenderKey: &params.SenderSK, PaymentInfo: pInfos, InputCoin: params.GetInputCoins(), FeeNativeCoin: params.Fee, HasPrivacyCoin: params.HasPrivacy, HasPrivacyToken: params.HasPrivacy, ShardID: 0, Info: params.Info, TokenParams: tp}
}

func createPrivKeyMlsagCA(inputCoins []privacy.PlainCoin, outputCoins []*privacy.CoinV2, outputSharedSecrets []*privacy.Point, params *TxPrivacyInitParams, shardID byte, commitmentsToZero []*privacy.Point) ([]*privacy.Scalar, error) {
	senderSK := params.SenderSK
	tokenID := params.TokenID
	if tokenID==nil{
		tokenID = &PRVCoinID
	}
	rehashed := privacy.HashToPoint(tokenID[:])
	sumRand := new(privacy.Scalar).FromUint64(0)

	privKeyMlsag := make([]*privacy.Scalar, len(inputCoins)+2)
	sumInputAssetTagBlinders := new(privacy.Scalar).FromUint64(0)
	numOfInputs := new(privacy.Scalar).FromUint64(uint64(len(inputCoins)))
	numOfOutputs := new(privacy.Scalar).FromUint64(uint64(len(outputCoins)))
	mySkBytes := (*senderSK)[:]
	for i := 0; i < len(inputCoins); i += 1 {
		var err error
		privKeyMlsag[i], err = inputCoins[i].ParsePrivateKeyOfCoin(*senderSK)
		if err != nil {
			return nil, err
		}

		inputCoin_specific, ok := inputCoins[i].(*privacy.CoinV2)
		if !ok || inputCoin_specific.GetAssetTag()==nil{
			return nil, errors.New("Cannot cast a coin as v2-CA")
		}

		isUnblinded := privacy.IsPointEqual(rehashed, inputCoin_specific.GetAssetTag())
		if isUnblinded{
		}

		sharedSecret := new(privacy.Point).Identity()
		bl := new(privacy.Scalar).FromUint64(0)
		if !isUnblinded{
			sharedSecret, err = inputCoin_specific.RecomputeSharedSecret(mySkBytes)
			if err != nil {
				return nil, err
			}
			_, indexForShard, err := inputCoin_specific.GetTxRandomDetail()
			if err != nil {
				return nil, err
			}
			bl, err = privacy.ComputeAssetTagBlinder(sharedSecret, indexForShard)
			if err != nil {
				return nil, err
			}
		}

		v := inputCoin_specific.GetAmount()
		effectiveRCom := new(privacy.Scalar).Mul(bl,v)
		effectiveRCom.Add(effectiveRCom, inputCoin_specific.GetRandomness())

		sumInputAssetTagBlinders.Add(sumInputAssetTagBlinders, bl)
		sumRand.Add(sumRand, effectiveRCom)
	}
	sumInputAssetTagBlinders.Mul(sumInputAssetTagBlinders, numOfOutputs)

	sumOutputAssetTagBlinders := new(privacy.Scalar).FromUint64(0)
	for i, oc := range outputCoins{
		if oc.GetAssetTag()==nil{
			return nil, errors.New("Cannot cast a coin as v2-CA")
		}
		// lengths between 0 and len(outputCoins) were rejected before
		bl := new(privacy.Scalar).FromUint64(0)
		isUnblinded := privacy.IsPointEqual(rehashed, oc.GetAssetTag())
		if isUnblinded{
		}else{
			_, indexForShard, err := oc.GetTxRandomDetail()
			if err != nil {
				return nil, err
			}
			bl, err = privacy.ComputeAssetTagBlinder(outputSharedSecrets[i], indexForShard)
			if err != nil {
				return nil, err
			}
		}
		v := oc.GetAmount()
		effectiveRCom := new(privacy.Scalar).Mul(bl,v)
		effectiveRCom.Add(effectiveRCom, oc.GetRandomness())
		sumOutputAssetTagBlinders.Add(sumOutputAssetTagBlinders, bl)
		sumRand.Sub(sumRand, effectiveRCom)
	}
	sumOutputAssetTagBlinders.Mul(sumOutputAssetTagBlinders, numOfInputs)

	// 2 final elements in `private keys` for MLSAG
	assetSum := new(privacy.Scalar).Sub(sumInputAssetTagBlinders, sumOutputAssetTagBlinders)
	firstCommitmentToZeroRecomputed := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], assetSum)
	secondCommitmentToZeroRecomputed := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], sumRand)
	if len(commitmentsToZero)!=2{
		return nil, errors.New("Error : need exactly 2 points for MLSAG double-checking")
	}
	match1 := privacy.IsPointEqual(firstCommitmentToZeroRecomputed, commitmentsToZero[0])
	match2 := privacy.IsPointEqual(secondCommitmentToZeroRecomputed, commitmentsToZero[1])
	if !match1 || !match2{
		return nil, errors.New("Error : asset tag sum or commitment sum mismatch")
	}
	privKeyMlsag[len(inputCoins)] 	= assetSum
	privKeyMlsag[len(inputCoins)+1]	= sumRand
	return privKeyMlsag, nil
}

func generateMlsagRingCA(inputCoins []privacy.PlainCoin, outputCoins []*privacy.CoinV2, params *TokenInnerParams, pi int, shardID byte, ringSize int) (*mlsag.Ring, [][]*big.Int, []*privacy.Point, error) {
	coinCache := params.TokenCache
	l := len(coinCache.PublicKeys)
	if len(coinCache.Commitments)!=l || len(coinCache.AssetTags)!=l{
		err := errors.New("Length mismatch in coin cache")
		return nil, nil, nil, err
	}
	randRange := big.NewInt(0).SetUint64(uint64(l))
	outputCoinsAsGeneric := make([]privacy.Coin, len(outputCoins))
	for i:=0;i<len(outputCoins);i++{
		outputCoinsAsGeneric[i] = outputCoins[i]
	}
	sumOutputsWithFee := calculateSumOutputsWithFee(outputCoinsAsGeneric, 0)
	inCount := new(privacy.Scalar).FromUint64(uint64(len(inputCoins)))
	outCount := new(privacy.Scalar).FromUint64(uint64(len(outputCoins)))
	sumOutputAssetTags := new(privacy.Point).Identity()
	for _, oc := range outputCoins{
		sumOutputAssetTags.Add(sumOutputAssetTags, oc.GetAssetTag())
	}
	sumOutputAssetTags.ScalarMult(sumOutputAssetTags, inCount)

	indexes := make([][]*big.Int, ringSize)
	ring := make([][]*privacy.Point, ringSize)
	var lastTwoColumnsCommitmentToZero []*privacy.Point
	for i := 0; i < ringSize; i += 1 {
		sumInputs := new(privacy.Point).Identity()
		sumInputs.Sub(sumInputs, sumOutputsWithFee)
		sumInputAssetTags := new(privacy.Point).Identity()

		row := make([]*privacy.Point, len(inputCoins))
		rowIndexes := make([]*big.Int, len(inputCoins))
		if i == pi {
			for j := 0; j < len(inputCoins); j += 1 {
				row[j] = inputCoins[j].GetPublicKey()
				publicKeyBytes := inputCoins[j].GetPublicKey().ToBytesS()
				key := b58.Encode(publicKeyBytes, common.ZeroByte)
				val, exists := coinCache.PkToIndexMap[key]
				if !exists{
					err := errors.New(fmt.Sprintf("Cannot find a coin's index using cached index map - %s", key))
					return nil, nil, nil, err
				}
				rowIndexes[j] = big.NewInt(0).SetUint64(val)
				sumInputs.Add(sumInputs, inputCoins[j].GetCommitment())
				inputCoin_specific, ok := inputCoins[j].(*privacy.CoinV2)
				if !ok{
					return nil, nil, nil, errors.New("Cannot cast a coin as v2")
				}
				sumInputAssetTags.Add(sumInputAssetTags, inputCoin_specific.GetAssetTag())
			}
		} else {
			for j := 0; j < len(inputCoins); j += 1 {
				temp, _ := RandBigIntMaxRange(randRange)
				pos := int(temp.Uint64())
				pkBytes := coinCache.PublicKeys[pos]
				key := b58.Encode(pkBytes, common.ZeroByte)
				val, exists := coinCache.PkToIndexMap[key]
				if !exists{
					err := errors.New(fmt.Sprintf("Cannot find a coin's index using cached index map - %s", key))
					return nil, nil, nil, err
				}
				rowIndexes[j] = big.NewInt(0).SetUint64(val)
				row[j], _ = new(privacy.Point).FromBytesS(pkBytes)

				commitmentBytes := coinCache.Commitments[pos]
				commitment, _ := new(privacy.Point).FromBytesS(commitmentBytes)
				sumInputs.Add(sumInputs, commitment)
				assetTagBytes := coinCache.AssetTags[pos]
				assetTag, _ := new(privacy.Point).FromBytesS(assetTagBytes)
				sumInputAssetTags.Add(sumInputAssetTags, assetTag)
			}
		}
		sumInputAssetTags.ScalarMult(sumInputAssetTags, outCount)

		assetSum := new(privacy.Point).Sub(sumInputAssetTags, sumOutputAssetTags)
		row = append(row, assetSum)
		row = append(row, sumInputs)
		if i==pi{
			lastTwoColumnsCommitmentToZero = []*privacy.Point{assetSum, sumInputs}
		}

		ring[i] = row
		indexes[i] = rowIndexes
	}
	return mlsag.NewRing(ring), indexes, lastTwoColumnsCommitmentToZero, nil
}

func (tx *Tx) proveCA(params_compat *TxPrivacyInitParams, params_token *TokenInnerParams) (bool, error) {
	var err error
	var outputCoins 	[]*privacy.CoinV2
	var sharedSecrets 	[]*privacy.Point
	// var numOfCoinsBurned uint = 0
	// we do not handle burning yet; we reject it
	// var isBurning bool = false
	var tid common.Hash = *params_compat.TokenID
	for _,inf := range params_compat.PaymentInfo{
		c, ss, err := privacy.NewCoinCA(inf, &tid)
		if err != nil {
			return false, err
		}
		if ss==nil{
			return false, errors.New("No burning in this client")
		}
		sharedSecrets 	= append(sharedSecrets, ss)
		outputCoins 	= append(outputCoins, c)
	}
	inputCoins := params_compat.InputCoins
	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, sharedSecrets, true, params_compat.PaymentInfo)
	if err != nil {
		return false, err
	}

	// also ignore the metadata cases
	// if tx.ShouldSignMetaData() {
	// 	if err := tx.signMetadata(params.SenderKey); err != nil {
	// 		utils.Logger.Log.Error("Cannot signOnMessage txMetadata in shouldSignMetadata")
	// 		return false, err
	// 	}
	// }
	err = tx.signCA(inputCoins, outputCoins, sharedSecrets, params_compat, params_token, tx.Hash()[:])
	return false, err
}

func (tx *Tx) signCA(inp []privacy.PlainCoin, out []*privacy.CoinV2, outputSharedSecrets []*privacy.Point, params_compat *TxPrivacyInitParams, params_token *TokenInnerParams, hashedMessage []byte) error {
	if tx.Sig != nil {
		return errors.New("input transaction must be an unsigned one")
	}
	ringSize := privacy.RingSize
	// default to privacy = true
	// if !params.HasPrivacy {
	// 	ringSize = 1
	// }

	// Generate Ring
	piBig,piErr := RandBigIntMaxRange(big.NewInt(int64(ringSize)))
	if piErr!=nil{
		return piErr
	}
	var pi int = int(piBig.Int64())
	shardID := common.GetShardIDFromLastByte(tx.PubKeyLastByteSender)
	ring, indexes, commitmentsToZero, err := generateMlsagRingCA(inp, out, params_token, pi, shardID, ringSize)
	if err != nil {
		return err
	}

	// Set SigPubKey
	txSigPubKey := new(SigPubKey)
	txSigPubKey.Indexes = indexes
	tx.SigPubKey, err = txSigPubKey.Bytes()
	if err != nil {
		return err
	}

	// Set sigPrivKey
	privKeysMlsag, err := createPrivKeyMlsagCA(inp, out, outputSharedSecrets, params_compat, shardID, commitmentsToZero)
	if err != nil {
		return err
	}
	sag := mlsag.NewMlsag(privKeysMlsag, ring, pi)
	sk, err := privacy.ArrayScalarToBytes(&privKeysMlsag)
	if err != nil {
		return err
	}
	tx.sigPrivKey = sk

	// Set Signature
	mlsagSignature, err := sag.SignConfidentialAsset(hashedMessage)
	if err != nil {
		return err
	}
	// inputCoins already hold keyImage so set to nil to reduce size
	mlsagSignature.SetKeyImages(nil)
	tx.Sig, err = mlsagSignature.ToBytes()

	return err
}

// this signs only on the hash of the data in it
func (tx *Tx) proveToken(params *InitParamsAsm) (bool, error) {
	temp := params.GetCompatTxTokenParams()
	var tid common.Hash
	_, err := tid.NewHashFromStr(temp.TokenParams.PropertyID)
	if err!=nil{
		return false, errors.New("Error parsing token id")
	}
	params_compat := &TxPrivacyInitParams{
				SenderSK: temp.SenderKey,
				PaymentInfo: temp.TokenParams.Receiver,
				InputCoins: temp.TokenParams.TokenInput,
				Fee: temp.TokenParams.Fee,
				HasPrivacy: temp.HasPrivacyToken,
				TokenID: &tid,
			}

	// Init tx and params (tx and params will be changed)
	if err := tx.initializeTxAndParams(params_compat); err != nil {
		return false, err
	}
	isBurning, err := tx.proveCA(params_compat, params.TokenParams)
	if err != nil {
		return false, err
	}
	return isBurning, nil
}

func (txToken *TxToken) initToken(params *InitParamsAsm) error {
	txToken.TxTokenData.Type = CustomTokenTransfer
	txToken.TxTokenData.PropertyName = ""
	txToken.TxTokenData.PropertySymbol = ""
	txToken.TxTokenData.Mintable = false

	switch txToken.TxTokenData.Type {
	// this client only supports 1 type : token transfer
	case CustomTokenTransfer:
		{
			propertyID, _ := common.TokenStringToHash(params.TokenParams.TokenID)
			dbFacingTokenID := common.ConfidentialAssetID
			// txp := params.GetCompatTxTokenParams()
			// txParams := &TxPrivacyInitParams{
			// 	SenderSK: txp.SenderKey,
			// 	PaymentInfo: txp.TokenParams.Receiver,
			// 	InputCoins: txp.TokenParams.TokenInput,
			// 	Fee: txp.TokenParams.Fee,
			// 	HasPrivacy: txp.HasPrivacyToken,
			// 	TokenID: propertyID,
			// }
			txNormal := new(Tx)
			isBurning, err := txNormal.proveToken(params)
			if err != nil {
				return errors.New("Prove Token failed")
			}
			txToken.TxTokenData.TxNormal = txNormal
			// tokenID is already hidden in asset tags in coin, here we use the umbrella ID
			if isBurning{
				// show plain tokenID if this is a burning TX
				txToken.TxTokenData.PropertyID = *propertyID
			}else{
				txToken.TxTokenData.PropertyID = dbFacingTokenID
			}
		}
	default:
		return errors.New("can't handle this TokenTxType")
	}
	return nil
}

// this signs on the hash of both sub TXs
func (tx *Tx) provePRV(params *InitParamsAsm, hashedTokenMessage []byte) error {
	var outputCoins []*privacy.CoinV2
	var pInfos []*privacy.PaymentInfo
	for _, payInf := range params.PaymentInfo{
		c, err := privacy.NewCoinFromPaymentInfo(&payInf)
		if err!=nil{
			return err
		}
		outputCoins = append(outputCoins, c)
		pInfos = append(pInfos, &payInf)
	}

	inputCoins := params.GetInputCoins()
	var err error
	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, nil, false, pInfos)
	if err != nil {
		return err
	}

	message := common.HashH(append(tx.Hash()[:], hashedTokenMessage...))
	err = tx.sign(inputCoins, outputCoins, params, message[:])
	return err
}

func (txToken *TxToken) initPRV(feeTx * Tx, params *InitParamsAsm) error {
	txTokenDataHash, err := txToken.TxTokenData.Hash()
	if err != nil {
		return err
	}
	if err := feeTx.provePRV(params, txTokenDataHash[:]); err != nil {
		return err
	}
	// override TxCustomTokenPrivacyType type
	feeTx.Type = TxCustomTokenPrivacyType
	txToken.Tx = feeTx

	return nil
}

func (txToken *TxToken) InitASM(params *InitParamsAsm) error {
	params_compat := params.GetCompatTxTokenParams()
	txPrivacyParams := &TxPrivacyInitParams{
		SenderSK: params_compat.SenderKey,
		PaymentInfo: params_compat.PaymentInfo,
		InputCoins: params_compat.InputCoin,
		Fee: params_compat.FeeNativeCoin,
		HasPrivacy: params_compat.HasPrivacyCoin,
		Info: params_compat.Info,
	}
	// Init tx and params (tx and params will be changed)
	tx := new(Tx)
	if err := tx.initializeTxAndParams(txPrivacyParams); err != nil {
		return err
	}
	if err := txToken.initToken(params); err != nil {
		return err
	}
	if err := txToken.initPRV(tx, params); err != nil {
		return err
	}

	return nil
}