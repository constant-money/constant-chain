package internal

import(
	"math/big"
	// "fmt"
	"encoding/json"
	// "strconv"

	"github.com/pkg/errors"
	"github.com/incognitochain/incognito-chain/common"

	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/internal/metadata"
)

const (
	CustomTokenInit = iota
	CustomTokenTransfer
	CustomTokenCrossShard
)


type TxTokenData struct {
	PropertyID     	common.Hash
	PropertyName   	string
	PropertySymbol 	string
	SigPubKey      	[]byte `json:"SigPubKey,omitempty"` // 33 bytes
	Sig            	[]byte `json:"Sig,omitempty"`       //
	Proof          	privacy.Proof

	Type     		int
	Mintable 		bool
}

func (td TxTokenData) Hash() (*common.Hash, error){
	// leave out signature & its public key when hashing tx
	td.Sig = []byte{}
	td.SigPubKey = []byte{}
	inBytes, err := json.Marshal(td)

	if err!=nil{
		return nil, err
	}
	// after this returns, tx is restored since the receiver is not a pointer
	hash := common.HashH(inBytes)
	return &hash, nil
}

func makeTxToken(txPRV *Tx, pubkey, sig []byte, proof privacy.Proof) *Tx{
	result := &Tx{
		Version: 	txPRV.Version,
		Type: 		txPRV.Type,
		LockTime: 	txPRV.LockTime,
		Fee: 		0,
		PubKeyLastByteSender: txPRV.PubKeyLastByteSender,
		Metadata: 	nil,
	}
	var clonedInfo []byte = nil
	if txPRV.Info!=nil{
		clonedInfo = make([]byte, len(txPRV.Info))
		copy(clonedInfo, txPRV.Info)
	}

	var clonedSig []byte = nil
	if sig!=nil{		
		clonedSig = make([]byte, len(sig))
		copy(clonedSig, sig)
	}
	var clonedPk []byte = nil
	if pubkey!=nil{
		clonedPk = make([]byte, len(pubkey))
		copy(clonedPk, pubkey)
	}
	result.Info = clonedInfo
	result.Proof = proof
	result.Sig = clonedSig
	result.SigPubKey = clonedPk
	result.Info = clonedInfo

	return result
}

type TxToken struct {
	Tx 				Tx 					`json:"Tx"`
	TokenData 		TxTokenData 		`json:"TxTokenPrivacyData"`
	cachedTxNormal	*Tx
}
func (tx *TxToken) Hash() *common.Hash{
	firstHash := tx.Tx.Hash()
	secondHash, err := tx.TokenData.Hash()
	if err!=nil{
		return nil
	}
	result := common.HashH(append(firstHash[:], secondHash[:]...))
	return &result
}

func (tx *TxToken) GetTxNormal() *Tx{
	if tx.cachedTxNormal!=nil{
		return tx.cachedTxNormal
	}
	result := makeTxToken(&tx.Tx, tx.TokenData.SigPubKey, tx.TokenData.Sig, tx.TokenData.Proof)
	// tx.cachedTxNormal = result
	return result
}
func (tx *TxToken) SetTxNormal(inTx *Tx) error{
	tx.TokenData.SigPubKey = inTx.SigPubKey
	tx.TokenData.Sig = inTx.Sig
	tx.TokenData.Proof = inTx.Proof
	tx.cachedTxNormal = inTx
	return nil
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
	Metadata           metadata.Metadata
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
	TokenID 	string 					   	`json:"TokenID"`
	TokenPaymentInfo []printedPaymentInfo  	`json:"PaymentInfo"`
	TokenInput	[]CoinInter				   	`json:"InputCoins"`
	TokenCache 	CoinCache 				   	`json:"CoinCache"`
	Type 		int 						`json:"TokenTxType",omitempty`
	TokenName 	string 						`json:"TokenName",omitempty`
	TokenSymbol string 						`json:"TokenSymbol",omitempty`
	Mintable 	bool 						`json:"TokenMintable",omitempty`
}

func (params *TokenInnerParams) GetInputCoins() ([]privacy.PlainCoin, []uint64){
	var resultCoins []privacy.PlainCoin
	var resultIndexes []uint64
	if len(params.TokenInput)==0{
		return []privacy.PlainCoin{}, []uint64{}
	}
	ver := params.TokenInput[0].Version
	for _,ci := range params.TokenInput{
		var c privacy.PlainCoin
		var ind uint64
		var err error
		if ver==2{
			c, ind, err = ci.ToCoin()
			if err!=nil{
				println(err.Error())
				return nil, nil
			}
		}else {
			var temp *privacy.CoinV1
			temp, ind, err = ci.ToCoinV1()
			if err!=nil{
				println(err.Error())
				return nil, nil
			}
			c = temp.CoinDetails
		}
		resultCoins = append(resultCoins, c)
		resultIndexes = append(resultIndexes, ind)
	}
	return resultCoins, resultIndexes
}

func (params *TokenInnerParams) GetCompatTokenParams() *TokenParam{
	var tpInfos []*privacy.PaymentInfo
	for _, payInf := range params.TokenPaymentInfo{
		temp, _ := payInf.To()
		tpInfos = append(tpInfos, temp)
	}

	tis, _ := params.GetInputCoins()

	return &TokenParam{PropertyID: params.TokenID, PropertyName: "", PropertySymbol: "", Amount: 0, TokenTxType: params.Type, Receiver: tpInfos, TokenInput: tis, Mintable: false, Fee: 0}
}

func (params *InitParamsAsm) GetCompatTxTokenParams() *TxTokenParams{
	if params.TokenParams==nil{
		return nil
	}
	var pInfos []*privacy.PaymentInfo
	for _, payInf := range params.PaymentInfo{
		temp, _ := payInf.To()
		pInfos = append(pInfos, temp)
	}
	tp := params.TokenParams.GetCompatTokenParams()
	ics, _ := params.GetInputCoins()
	var info []byte = []byte("")
	if len(params.Info)>0{
		info = params.Info
	}
	md, err := metadata.ParseMetadata(params.Metadata)
	if err!=nil{
		println(string(params.Metadata))
		println(err.Error())
		panic("BAD MD. END")
		md = nil
	}
	shardID := byte(0)
	if len(ics)>0{
		otaPub := ics[0].GetPublicKey().ToBytesS()
		shardID = GetShardIDFromLastByte(otaPub[len(otaPub)-1])
	}

	return &TxTokenParams{SenderKey: &params.SenderSK, PaymentInfo: pInfos, InputCoin: ics, FeeNativeCoin: params.Fee, HasPrivacyCoin: params.HasPrivacy, HasPrivacyToken: params.HasPrivacy, ShardID: shardID, Metadata: md, Info: info, TokenParams: tp}
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
			return nil, errors.Errorf("Cannot cast a coin as v2-CA")
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
			// _, _, indexForShard, err := inputCoin_specific.GetTxRandomDetail()
			// if err != nil {
			// 	return nil, err
			// }
			bl, err = privacy.ComputeAssetTagBlinder(sharedSecret)
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
			return nil, errors.Errorf("Cannot cast a coin as v2-CA")
		}
		// lengths between 0 and len(outputCoins) were rejected before
		bl := new(privacy.Scalar).FromUint64(0)
		isUnblinded := privacy.IsPointEqual(rehashed, oc.GetAssetTag())
		if isUnblinded{
		}else{
			var err error
			// _, _, indexForShard, err := oc.GetTxRandomDetail()
			// if err != nil {
			// 	return nil, err
			// }
			bl, err = privacy.ComputeAssetTagBlinder(outputSharedSecrets[i])
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
		return nil, errors.Errorf("Error : need exactly 2 points for MLSAG double-checking")
	}
	match1 := privacy.IsPointEqual(firstCommitmentToZeroRecomputed, commitmentsToZero[0])
	match2 := privacy.IsPointEqual(secondCommitmentToZeroRecomputed, commitmentsToZero[1])
	if !match1 || !match2{
		return nil, errors.Errorf("Error : asset tag sum or commitment sum mismatch, %v, %v", match1, match2)
	}
	privKeyMlsag[len(inputCoins)] 	= assetSum
	privKeyMlsag[len(inputCoins)+1]	= sumRand
	return privKeyMlsag, nil
}

func generateMlsagRingCA(inputCoins []privacy.PlainCoin, inputIndexes []uint64, outputCoins []*privacy.CoinV2, params *TokenInnerParams, pi int, shardID byte, ringSize int) (*mlsag.Ring, [][]*big.Int, []*privacy.Point, error) {
	coinCache := params.TokenCache
	l := len(coinCache.PublicKeys)
	if len(coinCache.Commitments)!=l || len(coinCache.AssetTags)!=l{
		err := errors.Errorf("Length mismatch in coin cache")
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
				
				rowIndexes[j] = big.NewInt(0).SetUint64(inputIndexes[j])
				sumInputs.Add(sumInputs, inputCoins[j].GetCommitment())
				inputCoin_specific, ok := inputCoins[j].(*privacy.CoinV2)
				if !ok{
					return nil, nil, nil, errors.Errorf("Cannot cast a coin as v2")
				}
				sumInputAssetTags.Add(sumInputAssetTags, inputCoin_specific.GetAssetTag())
			}
		} else {
			for j := 0; j < len(inputCoins); j += 1 {
				temp, _ := RandBigIntMaxRange(randRange)
				pos := int(temp.Uint64())
				pkBytes := coinCache.PublicKeys[pos]
				rowIndexes[j] = big.NewInt(0).SetUint64(coinCache.Indexes[pos])
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
	var numOfCoinsBurned uint = 0
	var isBurning bool = false
	var tid common.Hash = *params_compat.TokenID
	for _,inf := range params_compat.PaymentInfo{
		c, ss, err := privacy.NewCoinCA(inf, &tid)
		if err != nil {
			return false, err
		}
		if ss==nil{
			isBurning = true
			numOfCoinsBurned += 1
		}
		sharedSecrets 	= append(sharedSecrets, ss)
		outputCoins 	= append(outputCoins, c)
	}
	inputCoins, inputIndexes := params_token.GetInputCoins()
	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, sharedSecrets, true, params_compat.PaymentInfo)
	if err != nil {
		return false, err
	}
	if numOfCoinsBurned>1{
		return false, errors.Errorf("output must not have more than 1 burned coin")
	}

	
	jsb, _ := json.Marshal(tx)
	println("will sign CA on TX and hash", string(jsb), tx.Hash().String())
	err = tx.signCA(inputCoins, inputIndexes, outputCoins, sharedSecrets, params_compat, params_token, tx.Hash()[:])
	return isBurning, err
}

func (tx *Tx) signCA(inp []privacy.PlainCoin, inputIndexes []uint64, out []*privacy.CoinV2, outputSharedSecrets []*privacy.Point, params_compat *TxPrivacyInitParams, params_token *TokenInnerParams, hashedMessage []byte) error {
	if tx.Sig != nil {
		return errors.Errorf("input transaction must be an unsigned one")
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
	ring, indexes, commitmentsToZero, err := generateMlsagRingCA(inp, inputIndexes, out, params_token, pi, shardID, ringSize)
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
	tid, err := getTokenIDFromString(params.TokenParams.TokenID)
	if err!=nil{
		return false, errors.Errorf("Error parsing token id")
	}
	// paying fee using pToken is not supported
	feeToken := uint64(0)
	params_compat := NewTxParams(temp.SenderKey, temp.TokenParams.Receiver, temp.TokenParams.TokenInput, feeToken, temp.HasPrivacyToken, &tid, nil, temp.Info)

	// Init tx and params (tx and params will be changed)
	if err := tx.initializeTxAndParams(params_compat, &params.TokenParams.TokenPaymentInfo); err != nil {
		return false, err
	}
	tx.Type = common.TxCustomTokenPrivacyType

	isBurning, err := tx.proveCA(params_compat, params.TokenParams)
	if err != nil {
		return false, err
	}
	return isBurning, nil
}

func (txToken *TxToken) initToken(txNormal *Tx, params *InitParamsAsm) error {
	if params.TokenParams.Type==CustomTokenInit && len(params.TokenParams.TokenName)>0 && len(params.TokenParams.TokenSymbol)>0{
		txToken.TokenData.Type = CustomTokenInit
		txToken.TokenData.PropertyName = params.TokenParams.TokenName
		txToken.TokenData.PropertySymbol = params.TokenParams.TokenSymbol
		txToken.TokenData.Mintable = params.TokenParams.Mintable
	}else{
		txToken.TokenData.Type = CustomTokenTransfer
		txToken.TokenData.PropertyName = ""
		txToken.TokenData.PropertySymbol = ""
		txToken.TokenData.Mintable = false
	}

	switch txToken.TokenData.Type {
	case CustomTokenInit:
		temp := txNormal
		temp.Proof = new(privacy.ProofV2)
		temp.Proof.Init()
		params_compat := params.GetCompatTxTokenParams()
		// set output coins; hash everything but commitment; save the hash to compute the new token ID later
		message := []byte{}
		if len(params_compat.TokenParams.Receiver[0].Message) > 0 {
			message = params_compat.TokenParams.Receiver[0].Message
		}
		tempPaymentInfo := &privacy.PaymentInfo{PaymentAddress: params_compat.TokenParams.Receiver[0].PaymentAddress, Amount: params_compat.TokenParams.Receiver[0].Amount, Message: message}
		createdTokenCoin, errCoin := privacy.NewCoinFromPaymentInfo(tempPaymentInfo)
		if errCoin != nil {
			return errCoin
		}
		if err := temp.Proof.SetOutputCoins([]privacy.Coin{createdTokenCoin}); err != nil {
			return err
		}
		// the coin was copied onto the proof
		theCoinOnProof, ok := temp.Proof.GetOutputCoins()[0].(*privacy.CoinV2)
		if !ok {
			return errors.Errorf("Cannot create init-token TX - coin should have been ver2")
		}
		theCoinOnProof.SetCommitment(new(privacy.Point).Identity())
		hashInitToken, err := txToken.TokenData.Hash()
		if err != nil {
			return err
		}
		temp.Sig = []byte{}
		temp.SigPubKey = []byte{}

		var plainTokenID *common.Hash
		if params_compat.TokenParams.Mintable {
			propertyID, err := getTokenIDFromString(params_compat.TokenParams.PropertyID)
			if err != nil {
				return err
			}
			plainTokenID = &propertyID
		} else {
			//NOTICE: @merman update PropertyID calculated from hash of tokendata and shardID
			newHashInitToken := common.HashH(append(hashInitToken.GetBytes(), params_compat.ShardID))
			plainTokenID = &newHashInitToken		}

		// set the unblinded asset tag
		err = theCoinOnProof.SetPlainTokenID(plainTokenID)
		if err != nil {
			return err
		}
		txToken.TokenData.PropertyID = *plainTokenID
		txToken.SetTxNormal(temp)
		return nil
	case CustomTokenTransfer:
		propertyID, _ := common.TokenStringToHash(params.TokenParams.TokenID)
		dbFacingTokenID := common.ConfidentialAssetID

		isBurning, err := txNormal.proveToken(params)
		if err != nil {
			println(err.Error())
			return errors.Errorf("Prove Token failed - %v", err)
		}
		
		// tokenID is already hidden in asset tags in coin, here we use the umbrella ID
		if isBurning{
			// show plain tokenID if this is a burning TX
			txToken.TokenData.PropertyID = *propertyID
		}else{
			txToken.TokenData.PropertyID = dbFacingTokenID
		}
		txToken.SetTxNormal(txNormal)
		return nil
	default:
		return errors.Errorf("can't handle this TokenTxType")
	}
}

// this signs on the hash of both sub TXs
func (tx *Tx) provePRV(params *InitParamsAsm) ([]privacy.PlainCoin, []uint64, []*privacy.CoinV2, error) {
	var outputCoins []*privacy.CoinV2
	var pInfos []*privacy.PaymentInfo
	for _, payInf := range params.PaymentInfo{
		temp, _ := payInf.To()
		c, err := privacy.NewCoinFromPaymentInfo(temp)
		if err!=nil{
			return nil, nil, nil, err
		}
		outputCoins = append(outputCoins, c)
		pInfos = append(pInfos, temp)
	}

	inputCoins, inputIndexes := params.GetInputCoins()
	var err error
	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, nil, false, pInfos)
	if err != nil {
		return nil, nil, nil, err
	}
	if tx.ShouldSignMetaData() {
		if err := tx.signMetadata(&params.SenderSK); err != nil {
			return nil, nil, nil, err
		}
	}
	return inputCoins, inputIndexes, outputCoins, nil
}

func (txToken *TxToken) initPRV(feeTx * Tx, params *InitParamsAsm) ([]privacy.PlainCoin, []uint64, []*privacy.CoinV2, error) {
	feeTx.Type = common.TxCustomTokenPrivacyType
	inps, inputIndexes, outs, err := feeTx.provePRV(params)
	if err != nil {
		return nil, nil, nil, err
	}

	return inps, inputIndexes, outs, nil
}

func (txToken *TxToken) InitASM(params *InitParamsAsm) error {
	params_compat := params.GetCompatTxTokenParams()
	txPrivacyParams := NewTxParams(params_compat.SenderKey, params_compat.PaymentInfo, params_compat.InputCoin, params_compat.FeeNativeCoin, false, nil, params_compat.Metadata, params_compat.Info)
	// Init tx and params (tx and params will be changed)
	tx := new(Tx)
	if err := tx.initializeTxAndParams(txPrivacyParams, &params.PaymentInfo); err != nil {
		return err
	}
	inps, inputIndexes, outs, err := txToken.initPRV(tx, params)
	if err != nil {
		return err
	}
	txn := makeTxToken(tx, nil, nil, nil)
	// Init, prove and sign(CA) Token
	if err := txToken.initToken(txn, params); err != nil {
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
	if err!=nil{
		return err
	}
	return nil
}