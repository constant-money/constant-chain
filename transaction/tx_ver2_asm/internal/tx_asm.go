package internal

import(
	"encoding/base64"
	"encoding/json"
	"math/big"
	// "fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
	"github.com/incognitochain/incognito-chain/transaction/tx_ver2_asm/internal/metadata"
	"github.com/incognitochain/incognito-chain/wallet"

)

const MaxSizeByte = (1 << 8) - 1
var b58 = base58.Base58Check{}

type b58CompatBytes []byte
func (b b58CompatBytes) MarshalJSON() ([]byte, error){
	var res string
	if len([]byte(b))==0{
		res = ""
	}else{
		res = b58.Encode(b, common.ZeroByte)
	}
	// println(res)
	return json.Marshal(res)
}
func (b *b58CompatBytes) UnmarshalJSON(src []byte) error{
	var theStr string
	json.Unmarshal(src, &theStr)
	if len(theStr)==0{
		*b = b58CompatBytes([]byte{})
		return nil
	}
	// println(theStr)
	res, _, err := b58.Decode(theStr)
	*b = res
	return err
}

func (b b58CompatBytes) IsBlank() bool{
	return len([]byte(b))==0
}

type SigPubKey struct {
	Indexes [][]*big.Int
}
func (sigPub SigPubKey) Bytes() ([]byte, error) {
	n := len(sigPub.Indexes)
	if n == 0 {
		return nil, errors.New("TxSigPublicKeyVer2.ToBytes: Indexes is empty")
	}
	if n > MaxSizeByte {
		return nil, errors.New("TxSigPublicKeyVer2.ToBytes: Indexes is too large, too many rows")
	}
	m := len(sigPub.Indexes[0])
	if m > MaxSizeByte {
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
			if lengthByte > MaxSizeByte {
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

type Tx struct {
	// Basic data, required
	Version  int8   `json:"Version"`
	Type     string `json:"Type"` // Transaction type
	LockTime int64  `json:"LockTime"`
	Fee      uint64 `json:"Fee"` // Fee applies: always consant
	Info     []byte // 512 bytes
	
	// Sign and Privacy proof, required
	SigPubKey            []byte `json:"SigPubKey"` // 33 bytes
	Sig                  []byte `json:"Sig"`       //
	Proof                privacy.Proof
	PubKeyLastByteSender byte
	Metadata metadata.Metadata
	// private field, not use for json parser, only use as temp variable
	sigPrivKey       []byte       // is ALWAYS private property of struct, if privacy: 64 bytes, and otherwise, 32 bytes
}

type CoinCache struct{
	PublicKeys 		[]b58CompatBytes 			`json:"PublicKeys"`
	Commitments 	[]b58CompatBytes 			`json:"Commitments"`
	AssetTags		[]b58CompatBytes 			`json:"AssetTags,omitempty"`
	Indexes 		[]uint64 					`json:"Indexes"`
}
func MakeCoinCache() *CoinCache{
	return &CoinCache{
		PublicKeys: nil,
		Commitments: nil,
		AssetTags: nil,
		Indexes: nil,
	}
}

var b64 = base64.StdEncoding
var genericError = errors.New("Generic error for ASM")

// []byte equivalents are by default encoded with base64 when handled by JSON
type InitParamsAsm struct{
	SenderSK    privacy.PrivateKey		`json:"SenderSK"`
	PaymentInfo []printedPaymentInfo	`json:"PaymentInfo"`
	InputCoins  []CoinInter 	 		`json:"InputCoins"`
	Fee         uint64 					`json:"Fee"`
	HasPrivacy  bool 					`json:"HasPrivacy,omitempty"`
	TokenID     string 					`json:"TokenID,omitempty"`
	Metadata    json.RawMessage			`json:"Metadata,omitempty"`
	Info        []byte 					`json:"Info,omitempty"`
	Kvargs		map[string]interface{} 	`json:"Kvargs,omitempty"`

	Cache 		CoinCache 				`json:"CoinCache"`
	TokenParams *TokenInnerParams 		`json:"TokenParams,omitempty"`
}

type TxPrivacyInitParams struct {
	SenderSK    *privacy.PrivateKey
	PaymentInfo []*privacy.PaymentInfo
	InputCoins  []privacy.PlainCoin
	Fee         uint64
	HasPrivacy  bool
	TokenID     *common.Hash
	Metadata 	metadata.Metadata
	Info        []byte
}
func NewTxParams(sk *privacy.PrivateKey, pInfos []*privacy.PaymentInfo, inputs []privacy.PlainCoin, fee uint64, isPriv bool, tokenID *common.Hash, md metadata.Metadata, info []byte) *TxPrivacyInitParams{
	if info==nil{
		info = []byte("")
	}
	return &TxPrivacyInitParams{
		SenderSK: sk,
		PaymentInfo: pInfos,
		InputCoins: inputs,
		Fee: fee,
		HasPrivacy: isPriv,
		TokenID: tokenID,
		Metadata: md,
		Info: info,
	}
}

func (params *InitParamsAsm) GetInputCoins() ([]privacy.PlainCoin, []uint64){
	var resultCoins []privacy.PlainCoin
	var resultIndexes []uint64
	if len(params.InputCoins)==0{
		return []privacy.PlainCoin{}, []uint64{}
	}
	ver := params.InputCoins[0].Version
	for _,ci := range params.InputCoins{
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

func (params *InitParamsAsm) GetGenericParams() *TxPrivacyInitParams{
	var pInfos []*privacy.PaymentInfo
	for _, payInf := range params.PaymentInfo{
		temp, _ := payInf.To()
		pInfos = append(pInfos, temp)
	}
	tid, err := getTokenIDFromString(params.TokenID)
	if err!=nil{
		println(err.Error())
		return nil
	}

	md, err := metadata.ParseMetadata(params.Metadata)
	if err!=nil{
		println(string(params.Metadata))
		println(err.Error())
		panic("BAD MD. END")
		md = nil
	}
	var info []byte = []byte("")
	if len(params.Info)>0{
		info = params.Info
	}
	ics, _ := params.GetInputCoins()
	return NewTxParams(&params.SenderSK, pInfos, ics, params.Fee, params.HasPrivacy, &tid, md, info)
}

type printedPaymentInfo struct {
	PaymentAddress json.RawMessage 		`json:"PaymentAddress"`
	Amount         string 				`json:"Amount"`
	Message        []byte 			 	`json:"Message",omitempty`
}

func (pp printedPaymentInfo) To() (*privacy.PaymentInfo, error){
	result := &privacy.PaymentInfo{}
	var theStr string
	err := json.Unmarshal(pp.PaymentAddress, &theStr)
	if err!=nil{
		return nil, err
	}

	kw, err := wallet.Base58CheckDeserialize(theStr)
	if err!=nil{
		return nil, err
	}
	result.PaymentAddress = kw.KeySet.PaymentAddress
	num, err := strconv.ParseUint(pp.Amount, 10, 64)
	if err!=nil{
		return nil, err
	}
	result.Amount 	= num
	result.Message 	= pp.Message
	return result, nil
}
func (pp *printedPaymentInfo) From(pInf *privacy.PaymentInfo) {
	kw := &wallet.KeyWallet{}
	kw.KeySet = incognitokey.KeySet{}
	kw.KeySet.PaymentAddress = pInf.PaymentAddress
	paStr := kw.Base58CheckSerialize(wallet.PaymentAddressType)
	result := printedPaymentInfo{}
	result.PaymentAddress, _ = json.Marshal(paStr)
	result.Amount  = strconv.FormatUint(pInf.Amount, 10)
	result.Message = pInf.Message
	*pp = result
}

type printedUintStr uint64
func (u printedUintStr) MarshalJSON() ([]byte, error){
	return json.Marshal(strconv.FormatUint(uint64(u), 10))
}
func (u *printedUintStr) UnmarshalJSON(raw []byte) error{
	var theStr string
	json.Unmarshal(raw, &theStr)
	temp, err := strconv.ParseUint(theStr, 10, 64)
	*u = printedUintStr(temp)
	return err
}

type CoinInter struct {
	Version    		printedUintStr	`json:"Version"`
	Info       		b58CompatBytes 	`json:"Info"`
	Index      		b58CompatBytes 	`json:"Index"`
	PublicKey  		b58CompatBytes 	`json:"PublicKey"`
	Commitment 		b58CompatBytes 	`json:"Commitment"`
	KeyImage   		b58CompatBytes 	`json:"KeyImage"`

	SharedRandom 	b58CompatBytes 	`json:"SharedRandom"`
	SharedConcealRandom 	b58CompatBytes 	`json:"SharedConcealRandom"`
	TxRandom     	b58CompatBytes 	`json:"TxRandom"`
	Mask    		b58CompatBytes 	`json:"Randomness"`
	Value 			printedUintStr 	`json:"Value"`
	Amount 			b58CompatBytes 	`json:"CoinDetailsEncrypted"`

	// for v1
	SNDerivator     b58CompatBytes 	`json:"SNDerivator"`
	// tag is nil unless confidential asset
	AssetTag  		b58CompatBytes 	`json:"AssetTag"`
}
func (c CoinInter) ToCoin() (*privacy.CoinV2, uint64, error){
	var err error
	var p *privacy.Point
	result := &privacy.CoinV2{}
	result.SetVersion(uint8(c.Version))
	result.SetInfo(c.Info)
	if c.PublicKey.IsBlank(){
		result.SetPublicKey(nil)
	}else{
		p, err = (&privacy.Point{}).FromBytesS(c.PublicKey)
		if err!=nil{
			return nil, 0, err
		}
		result.SetPublicKey(p)
	}

	if c.Commitment.IsBlank(){
		result.SetCommitment(nil)
	}else{
		p, err = (&privacy.Point{}).FromBytesS(c.Commitment)
		if err!=nil{
			return nil, 0, err
		}
		result.SetCommitment(p)
	}

	if c.KeyImage.IsBlank(){
		result.SetKeyImage(nil)
	}else{
		p, err = (&privacy.Point{}).FromBytesS(c.KeyImage)
		if err!=nil{
			return nil, 0, err
		}
		result.SetKeyImage(p)
	}
	if c.SharedRandom.IsBlank(){
		result.SetSharedRandom(nil)
	}else{
		result.SetSharedRandom((&privacy.Scalar{}).FromBytesS(c.SharedRandom))
	}
	if c.SharedConcealRandom.IsBlank(){
		result.SetSharedConcealRandom(nil)
	}else{
		result.SetSharedConcealRandom((&privacy.Scalar{}).FromBytesS(c.SharedConcealRandom))
	}

	if c.Amount.IsBlank(){
		result.SetAmount(nil)
	}else{
		result.SetAmount((&privacy.Scalar{}).FromBytesS(c.Amount))
	}

	if c.TxRandom.IsBlank(){
		result.SetTxRandom(nil)
	}else{
		txr := (&privacy.TxRandom{})
		err = txr.SetBytes(c.TxRandom)
		if err!=nil{
			return nil, 0, err
		}	
		result.SetTxRandom(txr)
	}

	if c.Mask.IsBlank(){
		result.SetRandomness(nil)
	}else{
		result.SetRandomness((&privacy.Scalar{}).FromBytesS(c.Mask))
	}

	if c.AssetTag.IsBlank(){
		result.SetAssetTag(nil)
	}else{
		p, err = (&privacy.Point{}).FromBytesS(c.AssetTag)
		if err!=nil{
			return nil, 0, err
		}
		result.SetAssetTag(p)
	}
	ind := big.NewInt(0).SetBytes(c.Index)
	return result, ind.Uint64(), nil
}
func GetCoinInter(coin privacy.PlainCoin) CoinInter{
	var amount []byte = ScalarToBytes(nil)
	var txr []byte = ScalarToBytes(nil)
	cv2, ok := coin.(*privacy.CoinV2)
	if ok{
		amount = ScalarToBytes(cv2.GetAmount())
		txr = coin.GetTxRandom().Bytes()
	}

	return CoinInter{
		Version: printedUintStr(coin.GetVersion()),
		Info: coin.GetInfo(),
		PublicKey: PointToBytes(coin.GetPublicKey()),
		Commitment: PointToBytes(coin.GetCommitment()),
		KeyImage: PointToBytes(coin.GetKeyImage()),
		SharedRandom: ScalarToBytes(coin.GetSharedRandom()),
		SharedConcealRandom: ScalarToBytes(coin.GetSharedConcealRandom()),
		TxRandom: txr,
		Mask: ScalarToBytes(coin.GetRandomness()),
		Value: printedUintStr(coin.GetValue()),
		Amount: amount,
		AssetTag: PointToBytes(coin.GetAssetTag()),

		SNDerivator: ScalarToBytes(coin.GetSNDerivator()),
	}
}

func (c CoinInter) ToCoinV1() (*privacy.CoinV1, uint64, error){
	var err error
	var p *privacy.Point
	result := &privacy.CoinV1{}
	result.Init()
	result.CoinDetails.SetInfo(c.Info)
	if c.PublicKey.IsBlank(){
		result.CoinDetails.SetPublicKey(nil)
	}else{
		p, err = (&privacy.Point{}).FromBytesS(c.PublicKey)
		if err!=nil{
			return nil, 0, err
		}
		result.CoinDetails.SetPublicKey(p)
	}

	if c.Commitment.IsBlank(){
		result.CoinDetails.SetCommitment(nil)
	}else{
		p, err = (&privacy.Point{}).FromBytesS(c.Commitment)
		if err!=nil{
			return nil, 0, err
		}
		result.CoinDetails.SetCommitment(p)
	}

	if c.KeyImage.IsBlank(){
		result.CoinDetails.SetKeyImage(nil)
	}else{
		p, err = (&privacy.Point{}).FromBytesS(c.KeyImage)
		if err!=nil{
			return nil, 0, err
		}
		result.CoinDetails.SetKeyImage(p)
	}

	result.CoinDetails.SetValue(uint64(c.Value))
	if c.Mask.IsBlank(){
		result.CoinDetails.SetRandomness(nil)
	}else{
		result.CoinDetails.SetRandomness((&privacy.Scalar{}).FromBytesS(c.Mask))
	}

	if c.SNDerivator.IsBlank(){
		result.CoinDetails.SetSNDerivator(nil)
	}else{
		result.CoinDetails.SetSNDerivator((&privacy.Scalar{}).FromBytesS(c.SNDerivator))
	}
	result.CoinDetailsEncrypted.SetBytes([]byte(c.Amount))

	ind := big.NewInt(0).SetBytes(c.Index)
	return result, ind.Uint64(), nil
}

func ScalarToBytes(sc *privacy.Scalar) []byte{
	if sc==nil{
		return []byte{}
	}
	return sc.ToBytesS()
}
func PointToBytes(sc *privacy.Point) []byte{
	if sc==nil{
		return []byte{}
	}
	return sc.ToBytesS()
}

// func (ci CoinInter) Bytes() []byte{
// 	c, _, err := ci.ToCoin()
// 	if err!=nil{
// 		return nil
// 	}
// 	return c.Bytes()
// }
// func pad(arr []byte, length int) []byte{
// 	result := make([]byte, length)
// 	copy(result[length-len(arr):length], arr)
// 	return result
// }
// func (ci *CoinInter) SetBytes(coinBytes []byte) error {
// 	c := &privacy.CoinV2{}
// 	err := c.SetBytes(coinBytes)
// 	if err!=nil{
// 		return err
// 	}
// 	*ci = GetCoinInter(c)
// 	return nil
// }

func (tx Tx) Hash() *common.Hash {
	// leave out signature & its public key when hashing tx
	tx.Sig = []byte{}
	tx.SigPubKey = []byte{}
	inBytes, err := json.Marshal(tx)
	// println(fmt.Sprintf("Will hash TX : %s", string(inBytes)))
	if err!=nil{
		return nil
	}
	hash := common.HashH(inBytes)
	// after this returns, tx is restored since the receiver is not a pointer
	// println(fmt.Sprintf("Hash : %s", hash.String()))
	return &hash
}

func (tx Tx) HashWithoutMetadataSig() *common.Hash {
	md := tx.Metadata
	mdHash := md.HashWithoutSig()
	tx.Metadata = nil
	txHash := tx.Hash()
	if mdHash==nil || txHash==nil{
		return nil
	}
	// tx.SetMetadata(md)
	inBytes := append(mdHash[:], txHash[:]...)
	hash := common.HashH(inBytes)
	return &hash
}

func generateMlsagRing(inputCoins []privacy.PlainCoin, inputIndexes []uint64, outputCoins []*privacy.CoinV2, params *InitParamsAsm, pi int, shardID byte, ringSize int) (*mlsag.Ring, [][]*big.Int, *privacy.Point, error) {
	coinCache := params.Cache
	l := len(coinCache.PublicKeys)
	if len(coinCache.Commitments)!=l || len(coinCache.Indexes)!=l{
		err := errors.New("Length mismatch in coin cache")
		return nil, nil, nil, err
	}
	randRange := big.NewInt(0).SetUint64(uint64(l))
	outputCoinsAsGeneric := make([]privacy.Coin, len(outputCoins))
	for i:=0;i<len(outputCoins);i++{
		outputCoinsAsGeneric[i] = outputCoins[i]
	}
	sumOutputsWithFee := calculateSumOutputsWithFee(outputCoinsAsGeneric, params.Fee)

	indexes := make([][]*big.Int, ringSize)
	ring := make([][]*privacy.Point, ringSize)
	var commitmentToZero *privacy.Point
	for i := 0; i < ringSize; i += 1 {
		sumInputs := new(privacy.Point).Identity()
		sumInputs.Sub(sumInputs, sumOutputsWithFee)

		row := make([]*privacy.Point, len(inputCoins))
		rowIndexes := make([]*big.Int, len(inputCoins))
		if i == pi {
			for j := 0; j < len(inputCoins); j += 1 {
				row[j] = inputCoins[j].GetPublicKey()

				rowIndexes[j] = big.NewInt(0).SetUint64(inputIndexes[j])
				sumInputs.Add(sumInputs, inputCoins[j].GetCommitment())
			}
		} else {
			for j := 0; j < len(inputCoins); j += 1 {
				temp, _ := RandBigIntMaxRange(randRange)
				pos := int(temp.Uint64())
				pkBytes := coinCache.PublicKeys[pos]
				commitmentBytes := coinCache.Commitments[pos]
				// key := b58.Encode(pkBytes, common.ZeroByte)
				rowIndexes[j] = big.NewInt(0).SetUint64(coinCache.Indexes[pos])
				row[j], _ = new(privacy.Point).FromBytesS(pkBytes)

				commitment, _ := new(privacy.Point).FromBytesS(commitmentBytes)
				sumInputs.Add(sumInputs, commitment)
			}
		}
		row = append(row, sumInputs)
		if i==pi{
			commitmentToZero = sumInputs
		}
		ring[i] = row
		indexes[i] = rowIndexes
	}
	return mlsag.NewRing(ring), indexes, commitmentToZero, nil
}

func (tx *Tx) proveAsm(params *InitParamsAsm) error {
	var outputCoins []*privacy.CoinV2
	var pInfos []*privacy.PaymentInfo
	for _, payInf := range params.PaymentInfo{
		temp, _ := payInf.To()
		c, err := privacy.NewCoinFromPaymentInfo(temp)
		if err!=nil{
			return err
		}
		outputCoins = append(outputCoins, c)
		pInfos = append(pInfos, temp)
	}
	inputCoins, inputIndexes := params.GetInputCoins()
	var err error
	tx.Proof, err = privacy.ProveV2(inputCoins, outputCoins, nil, false, pInfos)
	if err != nil {
		return err
	}
	if tx.ShouldSignMetaData() {
		if err := tx.signMetadata(&params.SenderSK); err != nil {
			return err
		}
	}

	err = tx.sign(inputCoins, inputIndexes, outputCoins, params, tx.Hash()[:])
	return err
}

func (tx Tx) ShouldSignMetaData() bool {
	if tx.Metadata == nil {
		return false
	}
	return tx.Metadata.ShouldSignMetaData()
}

func (tx *Tx) sign(inp []privacy.PlainCoin, inputIndexes []uint64, out []*privacy.CoinV2, params *InitParamsAsm, hashedMessage []byte) error {
	if tx.Sig != nil {
		return errors.New("Re-signing TX is not allowed")
	}
	ringSize := privacy.RingSize
	

	// Generate Ring
	piBig, piErr := RandBigIntMaxRange(big.NewInt(int64(ringSize)))
	if piErr!=nil{
		return piErr
	}
	var pi int = int(piBig.Int64())
	shardID := GetShardIDFromLastByte(tx.PubKeyLastByteSender)
	ring, indexes, commitmentToZero, err := generateMlsagRing(inp, inputIndexes, out, params, pi, shardID, ringSize)
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
	privKeysMlsag, err := createPrivKeyMlsag(inp, out, &params.SenderSK, commitmentToZero)
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
	mlsagSignature, err := sag.Sign(hashedMessage)
	if err != nil {
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
		return errors.Errorf("meta.Sig should be empty or nil")
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
	// println("Signed MD")
	// println(tx.Metadata.GetSig())
	return nil
}

func (tx *Tx) InitASM(params *InitParamsAsm) error {
	gParams := params.GetGenericParams()
	// Init tx and params (tx and params will be changed)
	if err := tx.initializeTxAndParams(gParams, &params.PaymentInfo); err != nil {
		return err
	}
	// if check, err := tx.IsNonPrivacyNonInput(innerParams); check {
	// 	return err
	// }

	if err := tx.proveAsm(params); err != nil {
		return err
	}

	return nil
}

func (tx *Tx) initializeTxAndParams(params_compat *TxPrivacyInitParams, paymentsPtr *[]printedPaymentInfo) error {
	var err error
	// Get Keyset from param
	skBytes := *params_compat.SenderSK
	senderPaymentAddress := privacy.GeneratePaymentAddress(skBytes)
	tx.sigPrivKey = skBytes
	// Tx: initialize some values
	// non-zero means it was set before
	if tx.LockTime==0{
		tx.LockTime = time.Now().Unix()
	}
	tx.Fee = params_compat.Fee
	// normal type indicator
	tx.Type = TxNormalType
	tx.Metadata = params_compat.Metadata
	tx.PubKeyLastByteSender = senderPaymentAddress.Pk[len(senderPaymentAddress.Pk)-1]
	// we don't support version 1
	tx.Version = 2
	tx.Info = params_compat.Info
	// Params: update balance if overbalance
	if err = updateParamsWhenOverBalance(paymentsPtr, params_compat, senderPaymentAddress); err != nil {
		return err
	}

	return nil
}

func calculateSumOutputsWithFee(outputCoins []privacy.Coin, fee uint64) *privacy.Point {
	sumOutputsWithFee := new(privacy.Point).Identity()
	for i := 0; i < len(outputCoins); i += 1 {
		sumOutputsWithFee.Add(sumOutputsWithFee, outputCoins[i].GetCommitment())
	}
	feeCommitment := new(privacy.Point).ScalarMult(
		privacy.PedCom.G[privacy.PedersenValueIndex],
		new(privacy.Scalar).FromUint64(fee),
	)
	sumOutputsWithFee.Add(sumOutputsWithFee, feeCommitment)
	return sumOutputsWithFee
}

func createPrivKeyMlsag(inputCoins []privacy.PlainCoin, outputCoins []*privacy.CoinV2, senderSK *privacy.PrivateKey, commitmentToZero *privacy.Point) ([]*privacy.Scalar, error) {
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
			return nil, err
		}
	}
	commitmentToZeroRecomputed := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], sumRand)
	match := privacy.IsPointEqual(commitmentToZeroRecomputed, commitmentToZero)
	if !match{
		panic("FOUND IT. END")
		return nil, errors.Errorf("Error : asset tag sum or commitment sum mismatch")
	}
	privKeyMlsag[len(inputCoins)] = sumRand
	return privKeyMlsag, nil
}

func updateParamsWhenOverBalance(originPInfs *[]printedPaymentInfo, gParams *TxPrivacyInitParams, senderPaymentAddree privacy.PaymentAddress) error {
	// Calculate sum of all output coins' value
	sumOutputValue := uint64(0)
	for _, p := range *originPInfs {
		pInf, _ := p.To()
		sumOutputValue += pInf.Amount
	}

	// Calculate sum of all input coins' value
	sumInputValue := uint64(0)
	for _, coin := range gParams.InputCoins {
		// fmt.Printf("Input amount is %v - Mask is %v\nAsset tag is %v\n", coin.GetValue(), coin.GetRandomness(), coin.(*privacy.CoinV2).GetAssetTag())
		sumInputValue += coin.GetValue()
	}

	overBalance := int64(sumInputValue - sumOutputValue - gParams.Fee)
	// Check if sum of input coins' value is at least sum of output coins' value and tx fee
	if overBalance < 0 {
		return errors.New("Output + Fee > Input")
	}
	// Create a new payment to sender's pk where amount is overBalance if > 0
	if overBalance > 0 {
		// Should not check error because have checked before
		temp := new(privacy.PaymentInfo)
		temp.Amount = uint64(overBalance)
		temp.PaymentAddress = senderPaymentAddree
		changePaymentInfo := &printedPaymentInfo{}

		changePaymentInfo.From(temp)
		// println("change to", string(changePaymentInfo.PaymentAddress), "with amount", overBalance)
		gParams.PaymentInfo = append(gParams.PaymentInfo, temp)
		*originPInfs = append(*originPInfs, *changePaymentInfo)
	}

	return nil
}

func getTokenIDFromString(s string) (common.Hash, error){
	if len(s)==0{
		return common.PRVCoinID, nil
	}else{
		res, err := common.Hash{}.NewHashFromStr(s)
		return *res, err
	}
}