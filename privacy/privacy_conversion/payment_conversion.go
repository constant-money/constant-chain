package privacy_conversion

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_util"
	zkp "github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/oneoutofmany"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/serialnumberprivacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/utils"
	"github.com/incognitochain/incognito-chain/privacy/proof/agg_interface"
	"math/big"
	"strconv"
)

// For conversion proof, its version will be counted down from 255 -> 0
// It should contain inputCoins of v1 and outputCoins of v2 because it convert v1 to v2
type ConversionProof struct {
	Version           uint8
	inputCoins        []*coin.PlainCoinV1
	outputCoins       []*coin.CoinV1
	oneOfManyProof    []*oneoutofmany.OneOutOfManyProof
	serialNumberProof []*serialnumberprivacy.SNPrivacyProof
	comInputSecretKey *operation.Point
	comInputSND       []*operation.Point
	comInputValue     []*operation.Point
	comInputShardID   *operation.Point
	comIndices        []uint64
}

func (proof ConversionProof) Init() {
	proof.Version = ConversionProofVersion
	proof.inputCoins = make([]*coin.PlainCoinV1, 0)
	proof.outputCoins = make([]*coin.CoinV1, 0)
	proof.oneOfManyProof = make([]*oneoutofmany.OneOutOfManyProof, 0)
	proof.serialNumberProof = make([]*serialnumberprivacy.SNPrivacyProof, 0)
	proof.comInputSecretKey = new(operation.Point).Identity()
	proof.comInputSND = make([]*operation.Point, 0)
	proof.comInputValue = make([]*operation.Point, 0)
	proof.comInputShardID = new(operation.Point).Identity()
	proof.comInputSecretKey = new(operation.Point).Identity()
	proof.comInputSecretKey = new(operation.Point).Identity()
}

func (proof ConversionProof) MarshalJSON() ([]byte, error) {
	data := proof.Bytes()
	//temp := base58.Base58Check{}.Encode(data, common.ZeroByte)
	temp := base64.StdEncoding.EncodeToString(data)
	return json.Marshal(temp)
}

func (proof *ConversionProof) UnmarshalJSON(data []byte) error {
	dataStr := common.EmptyString
	errJson := json.Unmarshal(data, &dataStr)
	if errJson != nil {
		return errJson
	}

	//temp, _, err := base58.Base58Check{}.Decode(dataStr)
	temp, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return err
	}

	errSetBytes := proof.SetBytes(temp)
	if errSetBytes != nil {
		return errSetBytes
	}

	return nil
}

func (proof ConversionProof) Bytes() []byte {
	var proofBytes []byte

	//Version
	proofBytes = append(proofBytes, ConversionProofVersion)

	//InputCoins
	proofBytes = append(proofBytes, byte(len(proof.inputCoins)))
	for i := 0; i < len(proof.inputCoins); i++ {
		inCoinBytes := proof.inputCoins[i].Bytes()
		lenInCoinBytes := make([]byte, 0)
		lenInCoinBytes = append(lenInCoinBytes, common.IntToBytes(len(inCoinBytes))...)

		proofBytes = append(proofBytes, lenInCoinBytes...)
		proofBytes = append(proofBytes, inCoinBytes...)
	}

	//OutputCoins
	proofBytes = append(proofBytes, byte(len(proof.outputCoins)))
	for _, outCoin := range proof.outputCoins {
		outCoinBytes := outCoin.Bytes()
		lenOutCoinBytes := make([]byte, 0)
		lenOutCoinBytes = append(lenOutCoinBytes, common.IntToBytes(len(outCoinBytes))...)

		proofBytes = append(proofBytes, lenOutCoinBytes...)
		proofBytes = append(proofBytes, outCoinBytes...)
	}

	//OneOfManyProofs
	proofBytes = append(proofBytes, byte(len(proof.oneOfManyProof)))
	for i := 0; i < len(proof.oneOfManyProof); i++ {
		oneOfManyProof := proof.oneOfManyProof[i].Bytes()
		proofBytes = append(proofBytes, common.IntToBytes(utils.OneOfManyProofSize)...)
		proofBytes = append(proofBytes, oneOfManyProof...)
	}

	//SerialNumberProofs
	proofBytes = append(proofBytes, byte(len(proof.serialNumberProof)))
	for i := 0; i < len(proof.serialNumberProof); i++ {
		serialNumberProof := proof.serialNumberProof[i].Bytes()
		proofBytes = append(proofBytes, common.IntToBytes(utils.SnPrivacyProofSize)...)
		proofBytes = append(proofBytes, serialNumberProof...)
	}

	//CommitmentInputSecretKey
	if proof.comInputSecretKey != nil {
		comInputSKBytes := proof.comInputSecretKey.ToBytesS()
		proofBytes = append(proofBytes, byte(operation.Ed25519KeySize))
		proofBytes = append(proofBytes, comInputSKBytes...)
	} else {
		proofBytes = append(proofBytes, byte(0))
	}

	//CommitmentInputSNDs
	proofBytes = append(proofBytes, byte(len(proof.comInputSND)))
	for _, comInputSND := range proof.comInputSND {
		comInputSNDBytes := comInputSND.ToBytesS()
		proofBytes = append(proofBytes, byte(operation.Ed25519KeySize))
		proofBytes = append(proofBytes, comInputSNDBytes...)
	}

	//CommitmentInputValues
	proofBytes = append(proofBytes, byte(len(proof.comInputValue)))
	for _, comInputValue := range proof.comInputValue {
		comInputValueBytes := comInputValue.ToBytesS()
		proofBytes = append(proofBytes, byte(operation.Ed25519KeySize))
		proofBytes = append(proofBytes, comInputValueBytes...)
	}

	//CommitmentInputShardID
	if proof.comInputShardID != nil {
		comInputShardIDBytes := proof.comInputShardID.ToBytesS()
		proofBytes = append(proofBytes, byte(operation.Ed25519KeySize))
		proofBytes = append(proofBytes, comInputShardIDBytes...)
	} else {
		proofBytes = append(proofBytes, byte(0))
	}

	//CommitmentIndices
	for i := 0; i < len(proof.comIndices); i++ {
		proofBytes = append(proofBytes, common.AddPaddingBigInt(big.NewInt(int64(proof.comIndices[i])), common.Uint64Size)...)
	}

	return proofBytes
}

func (proof *ConversionProof) SetBytes(proofBytes []byte) *errhandler.PrivacyError {
	var lenData int
	if len(proofBytes) == 0 {
		return errhandler.NewPrivacyErr(errhandler.InvalidInputToSetBytesErr, fmt.Errorf("length of proof is zero"))
	}
	var err error
	offset := 0

	//Set version
	proof.Version = proofBytes[offset]
	offset += 1

	//Set input coins
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range input coins"))
	}
	numInputCoins := int(proofBytes[offset])
	offset += 1
	inputCoins := make([]coin.PlainCoin, 0)
	for i := 0; i < numInputCoins; i++ {
		//try 1-byte length
		if offset >= len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range input coins"))
		}
		lenData = int(proofBytes[offset])
		offset += 1

		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range input coins"))
		}
		coinBytes := proofBytes[offset : offset+lenData]
		tmpInCoin, err := coin.NewPlainCoinFromByte(coinBytes)
		if err != nil {
			//try 2-byte length
			if offset+1 > len(proofBytes) {
				return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range input coins"))
			}
			lenData = common.BytesToInt(proofBytes[offset-1 : offset+1])
			offset += 1

			if offset+lenData > len(proofBytes) {
				return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range input coins"))
			}
			coinBytes = proofBytes[offset : offset+lenData]
			tmpInCoin, err = coin.NewPlainCoinFromByte(coinBytes)
			if err != nil {
				return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("NewPlainCoinFromByte error: %v", err))
			}
		}
		inputCoins = append(inputCoins, tmpInCoin)
		offset += lenData
	}
	err = proof.SetInputCoins(inputCoins)
	if err != nil {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("SetInputCoins error: %v", err))
	}

	//Set output coins
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range output coins"))
	}
	numOutputCoins := int(proofBytes[offset])
	offset += 1
	outputCoins := make([]coin.Coin, 0)
	for i := 0; i < numOutputCoins; i++ {
		tmpOutCoin := new(coin.CoinV1)
		//try 1-byte length
		if offset >= len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range output coins"))
		}
		lenData = int(proofBytes[offset])
		offset += 1

		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range output coins"))
		}
		coinBytes := proofBytes[offset : offset+lenData]
		err = tmpOutCoin.SetBytes(coinBytes)
		if err != nil {
			//try 2-byte length
			if offset+1 > len(proofBytes) {
				return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range output coins"))
			}
			lenData = common.BytesToInt(proofBytes[offset-1 : offset+1])
			offset += 1

			if offset+lenData > len(proofBytes) {
				return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range output coins"))
			}
			coinBytes = proofBytes[offset : offset+lenData]
			err = tmpOutCoin.SetBytes(coinBytes)
			if err != nil {
				return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("outputCoin SetBytes error: %v", err))
			}
		}
		outputCoins = append(outputCoins, tmpOutCoin)
		offset += lenData
	}
	err = proof.SetOutputCoins(outputCoins)
	if err != nil {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("SetOutputCoins error: %v", err))
	}

	//Set oneOfManyProofs
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range one out of many proof"))
	}
	numOneOfManyProofs := int(proofBytes[offset])
	offset += 1
	proof.oneOfManyProof = make([]*oneoutofmany.OneOutOfManyProof, numOneOfManyProofs)
	for i := 0; i < numOneOfManyProofs; i++ {
		if offset+2 > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range one out of many proof"))
		}
		lenData = common.BytesToInt(proofBytes[offset : offset+2])
		offset += 2
		proof.oneOfManyProof[i] = new(oneoutofmany.OneOutOfManyProof).Init()

		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range one out of many proof"))
		}
		err := proof.oneOfManyProof[i].SetBytes(proofBytes[offset : offset+lenData])
		if err != nil {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, err)
		}
		offset += lenData
	}

	// Set serialNumberProofs
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range serial number proof"))
	}
	numSNProofs := int(proofBytes[offset])
	offset += 1
	proof.serialNumberProof = make([]*serialnumberprivacy.SNPrivacyProof, numSNProofs)
	for i := 0; i < numSNProofs; i++ {
		if offset+2 > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range serial number proof"))
		}
		lenData = common.BytesToInt(proofBytes[offset : offset+2])
		offset += 2
		proof.serialNumberProof[i] = new(serialnumberprivacy.SNPrivacyProof).Init()

		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range serial number proof"))
		}
		err := proof.serialNumberProof[i].SetBytes(proofBytes[offset : offset+lenData])
		if err != nil {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, err)
		}
		offset += lenData
	}

	//Set comInputSecretKey
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins private key"))
	}
	lenData = int(proofBytes[offset])
	offset += 1
	if lenData > 0 {
		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins private key"))
		}

		proof.comInputSecretKey, err = new(operation.Point).FromBytesS(proofBytes[offset : offset+lenData])
		if err != nil {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, err)
		}

		offset += lenData
	}

	//Set comInputSNDs
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins snd"))
	}
	numComInputSNDs := int(proofBytes[offset])
	offset += 1
	proof.comInputSND = make([]*operation.Point, numComInputSNDs)
	for i := 0; i < numComInputSNDs; i++ {
		if offset >= len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins snd"))
		}
		lenData = int(proofBytes[offset])
		offset += 1

		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins snd"))
		}

		proof.comInputSND[i], err = new(operation.Point).FromBytesS(proofBytes[offset : offset+lenData])
		if err != nil {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, err)
		}

		offset += lenData
	}

	//Set comInputValues
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins value"))
	}
	numComInputValues := int(proofBytes[offset])
	offset += 1
	proof.comInputValue = make([]*operation.Point, numComInputValues)
	for i := 0; i < numComInputValues; i++ {
		if offset >= len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins value"))
		}
		lenData = int(proofBytes[offset])
		offset += 1

		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins value"))
		}

		proof.comInputValue[i], err = new(operation.Point).FromBytesS(proofBytes[offset : offset+lenData])
		if err != nil {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, err)
		}

		offset += lenData
	}

	//Set comInputShardID
	if offset >= len(proofBytes) {
		return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins shardid"))
	}
	lenData = int(proofBytes[offset])
	offset += 1
	if lenData > 0 {
		if offset+lenData > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment input coins shardid"))
		}
		proof.comInputShardID, err = new(operation.Point).FromBytesS(proofBytes[offset : offset+lenData])

		if err != nil {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, err)
		}
		offset += lenData
	}

	//Set comIndices
	proof.comIndices = make([]uint64, len(proof.oneOfManyProof)*privacy_util.CommitmentRingSize)
	for i := 0; i < len(proof.oneOfManyProof)*privacy_util.CommitmentRingSize; i++ {
		if offset+common.Uint64Size > len(proofBytes) {
			return errhandler.NewPrivacyErr(errhandler.SetBytesProofErr, fmt.Errorf("out of range commitment indices"))
		}
		proof.comIndices[i] = new(big.Int).SetBytes(proofBytes[offset : offset+common.Uint64Size]).Uint64()
		offset = offset + common.Uint64Size
	}

	return nil
}

func (proof ConversionProof) IsPrivacy() bool {
	return true
}

// GET/SET function
func (proof *ConversionProof) GetVersion() uint8 { return ConversionProofVersion }
func (proof ConversionProof) GetAggregatedRangeProof() agg_interface.AggregatedRangeProof {
	return nil
}
func (proof ConversionProof) GetOneOfManyProof() []*oneoutofmany.OneOutOfManyProof {
	return proof.oneOfManyProof
}
func (proof ConversionProof) GetSerialNumberProof() []*serialnumberprivacy.SNPrivacyProof {
	return proof.serialNumberProof
}
func (proof ConversionProof) GetCommitmentInputSecretKey() *operation.Point {
	return proof.comInputSecretKey
}
func (proof ConversionProof) GetCommitmentInputSND() []*operation.Point {
	return proof.comInputSND
}
func (proof ConversionProof) GetCommitmentInputShardID() *operation.Point {
	return proof.comInputShardID
}
func (proof ConversionProof) GetCommitmentInputValue() []*operation.Point {
	return proof.comInputValue
}
func (proof ConversionProof) GetCommitmentIndices() []uint64 { return proof.comIndices }
func (proof ConversionProof) GetInputCoins() []coin.PlainCoin {
	res := make([]coin.PlainCoin, 0)
	for _, inputCoin := range proof.inputCoins {
		res = append(res, inputCoin)
	}
	return res
}
func (proof ConversionProof) GetOutputCoins() []coin.Coin {
	res := make([]coin.Coin, len(proof.outputCoins))
	for i := 0; i < len(proof.outputCoins); i += 1 {
		res[i] = proof.outputCoins[i]
	}
	return res
}

func (proof *ConversionProof) SetSerialNumberProof(serialNumberProof []*serialnumberprivacy.SNPrivacyProof) {
	proof.serialNumberProof = serialNumberProof
}
func (proof *ConversionProof) SetOneOfManyProof(oneOfManyProof []*oneoutofmany.OneOutOfManyProof) {
	proof.oneOfManyProof = oneOfManyProof
}
func (proof *ConversionProof) SetCommitmentShardID(comInputShardID *operation.Point) {
	proof.comInputShardID = comInputShardID
}
func (proof *ConversionProof) SetCommitmentInputSND(comInputSND []*operation.Point) {
	proof.comInputSND = comInputSND
}
func (proof *ConversionProof) SetCommitmentInputValue(comInputValue []*operation.Point) {
	proof.comInputValue = comInputValue
}
func (proof *ConversionProof) SetCommitmentInputSecretKey(comInputSecretKey *operation.Point) {
	proof.comInputSecretKey = comInputSecretKey
}
func (proof *ConversionProof) SetInputCoins(inputCoins []coin.PlainCoin) error {
	var err error
	proof.inputCoins = make([]*coin.PlainCoinV1, 0)
	for _, inCoin := range inputCoins {
		inCoinBytes := inCoin.Bytes()
		tmpInCoin := new(coin.PlainCoinV1)
		if err = tmpInCoin.SetBytes(inCoinBytes); err != nil {
			return fmt.Errorf("set input coin %v error: %v", inCoin, err)
		}
		proof.inputCoins = append(proof.inputCoins, tmpInCoin)
	}
	return nil
}
func (proof *ConversionProof) SetOutputCoins(outputCoins []coin.Coin) error {
	var err error
	proof.outputCoins = make([]*coin.CoinV1, 0)
	for _, outCoin := range outputCoins {
		outCoinBytes := outCoin.Bytes()
		tmpOutCoin := new(coin.CoinV1)
		if err = tmpOutCoin.SetBytes(outCoinBytes); err != nil {
			return fmt.Errorf("set output coin %v error: %v", outCoin, err)
		}
		proof.outputCoins = append(proof.outputCoins, tmpOutCoin)
	}
	return nil
}
func (proof *ConversionProof) SetCommitmentIndices(comIndices []uint64) {
	proof.comIndices = comIndices
}

func (proof ConversionProof) ValidateSanity(additionalData interface{}) (bool, error) {
	if len(proof.inputCoins) > 255 {
		return false, fmt.Errorf("input coins in tx are very large: %v", strconv.Itoa(len(proof.inputCoins)))
	}

	if len(proof.inputCoins) != len(proof.serialNumberProof) || len(proof.inputCoins) != len(proof.oneOfManyProof){
		return false, fmt.Errorf("the number of input coins must be equal to the number of serialnumber proofs and the number of one-of-many proofs")
	}

	// check doubling a input coin in tx
	serialNumbers := make(map[[operation.Ed25519KeySize]byte]bool)
	for i, inCoin := range proof.inputCoins {
		hashSN := inCoin.GetKeyImage().ToBytes()
		if serialNumbers[hashSN] {
			Logger.Log.Errorf("double input in proof - index %v\n", i)
			return false, fmt.Errorf("double input in tx")
		}
		serialNumbers[hashSN] = true
	}

	param, ok := additionalData.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("cannot cast additional data")
	}

	_, ok = param["sigPubKey"]
	if !ok {
		return false, fmt.Errorf("sigPubkey not found")
	}
	sigPubKeyPoint, ok := param["sigPubKey"].(*operation.Point)
	if !ok {
		return false, fmt.Errorf("cannot cast sigPubkey param")
	}

	cmInputSK := proof.comInputSecretKey

	for i, oneOfManyProof := range proof.oneOfManyProof {
		if oneOfManyProof == nil {
			return false, fmt.Errorf("oneOfManyProof %v is nil", i)
		}
		if !oneOfManyProof.ValidateSanity() {
			return false, fmt.Errorf("oneOfManyProof %v sanity rejected", i)
		}
	}

	for i, snProof := range proof.serialNumberProof {
		if snProof == nil {
			return false, fmt.Errorf("snProof %v is nil", i)
		}
		if !operation.IsPointEqual(cmInputSK, snProof.GetComSK()) {
			Logger.Log.Errorf("ComSK in SNProof %v is not equal to commitment of private key\n", i)
			return false, fmt.Errorf("comSK of SNProof %v is not comSK of private key", i)
		}
		if !operation.IsPointEqual(proof.comInputSND[i], snProof.GetComInput()) {
			Logger.Log.Errorf("cmSND in SNProof %v is not equal to commitment of input's SND\n", i)
			return false, fmt.Errorf("cmSND in SNproof %v is not equal to commitment of input's SND", i)
		}
		if !snProof.ValidateSanity() {
			return false, fmt.Errorf("snProof %v sanity rejected", i)
		}
	}

	for i := 0; i < len(proof.inputCoins); i++ {
		if isBadPoint(proof.inputCoins[i].GetKeyImage()) {
			return false, fmt.Errorf("validate sanity Serial number of input coin failed")
		}
	}

	for i := 0; i < len(proof.outputCoins); i++ {
		if isBadPoint(proof.outputCoins[i].CoinDetails.GetPublicKey()) {
			return false, fmt.Errorf("validate sanity Public key of output coin failed")
		}
		if isBadPoint(proof.outputCoins[i].CoinDetails.GetCommitment()) {
			return false, fmt.Errorf("validate sanity Coin commitment of output coin failed")
		}
		if isBadScalar(proof.outputCoins[i].CoinDetails.GetSNDerivator()) {
			return false, fmt.Errorf("validate sanity SNDerivator of output coin failed")
		}
	}

	// check ComInputSK
	if isBadPoint(cmInputSK) {
		return false, fmt.Errorf("validate sanity ComInputSK of proof failed")
	}

	if !operation.IsPointEqual(cmInputSK, sigPubKeyPoint) {
		return false, fmt.Errorf("SigPubKey is not equal to commitment of private key")
	}

	for i := 0; i < len(proof.comInputValue); i++ {
		if isBadPoint(proof.comInputValue[i]) {
			return false, fmt.Errorf("validate sanity ComInputValue of proof failed")
		}
	}

	for i := 0; i < len(proof.comInputSND); i++ {
		if isBadPoint(proof.comInputSND[i]) {
			return false, fmt.Errorf("validate sanity ComInputSND of proof failed")
		}
	}

	//check ComInputShardID
	if isBadPoint(proof.comInputShardID) {
		return false, fmt.Errorf("validate sanity ComInputShardID of proof failed")
	}
	_, ok = param["shardID"]
	if !ok {
		return false, fmt.Errorf("shardID not found")
	}
	shardID, ok := param["shardID"].(byte)
	if !ok {
		return false, fmt.Errorf("cannot cast shardID param")
	}
	fixedRand := zkp.FixedRandomnessShardID
	expectedCMShardID := operation.PedCom.CommitAtIndex(
		new(operation.Scalar).FromUint64(uint64(shardID)),
		fixedRand, operation.PedersenShardIDIndex)

	if !operation.IsPointEqual(expectedCMShardID, proof.comInputShardID) {
		return false, fmt.Errorf("ComInputShardID must be committed with the fixed randomness")
	}

	if len(proof.comIndices) != len(proof.inputCoins)*privacy_util.CommitmentRingSize {
		return false, fmt.Errorf("validate sanity CommitmentIndices of proof failed")

	}

	return true, nil
}

func (proof ConversionProof) Verify(boolParams map[string]bool, pubKey key.PublicKey, fee uint64, shardID byte, tokenID *common.Hash, additionalData interface{}) (bool, error) {
	Logger.Log.Infof("Begin verifying ConversionProof\n")
	if len(proof.outputCoins) != 1 {
		return false, errhandler.NewPrivacyErr(errhandler.UnexpectedErr, fmt.Errorf("number of output coins (%v) must be 1", len(proof.outputCoins)))
	}

	// verify for input coins
	commitmentsPtr := additionalData.(*[][privacy_util.CommitmentRingSize]*operation.Point)
	commitments := *commitmentsPtr

	for i := 0; i < len(proof.oneOfManyProof); i++ {
		proof.oneOfManyProof[i].Statement.Commitments = commitments[i][:]
		valid, err := proof.oneOfManyProof[i].Verify()
		if !valid {
			Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: One out of many failed")
			return false, errhandler.NewPrivacyErr(errhandler.VerifyOneOutOfManyProofFailedErr, err)
		}
		// Verify for the Proof that input coins' serial number is derived from the committed derivator
		valid, err = proof.serialNumberProof[i].Verify(nil)
		if !valid {
			Logger.Log.Errorf("VERIFICATION PAYMENT PROOF: Serial number privacy failed")
			return false, errhandler.NewPrivacyErr(errhandler.VerifySerialNumberPrivacyProofFailedErr, err)
		}
	}

	comInputValueSum := new(operation.Point).Identity()
	for _, comInputValue := range proof.comInputValue {
		comInputValueSum.Add(comInputValueSum, comInputValue)
	}

	outCoin := proof.outputCoins[0]
	if outCoin.IsEncrypted() {
		return false, fmt.Errorf("output of conversionproof should not be encrypted")
	}

	shardID, err := outCoin.GetShardID()
	if err != nil {
		Logger.Log.Errorf("cannot get shardID of output coin: %v\n", err)
		return false, err
	}
	comOutputSK := outCoin.CoinDetails.GetPublicKey()
	comOutputValue := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenValueIndex], new(operation.Scalar).FromUint64(outCoin.CoinDetails.GetValue()))
	comOutputSND := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenSndIndex], outCoin.CoinDetails.GetSNDerivator())
	comOutputShardID := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenShardIDIndex], new(operation.Scalar).FromBytes([operation.Ed25519KeySize]byte{shardID}))
	comOutputRandomness := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenRandomnessIndex], outCoin.CoinDetails.GetRandomness())

	tmpOutputCommitment := new(operation.Point).Add(comOutputSK, comOutputValue)
	tmpOutputCommitment.Add(tmpOutputCommitment, comOutputSND)
	tmpOutputCommitment.Add(tmpOutputCommitment, comOutputShardID)
	tmpOutputCommitment.Add(tmpOutputCommitment, comOutputRandomness)

	if !operation.IsPointEqual(tmpOutputCommitment, outCoin.GetCommitment()) {
		Logger.Log.Errorf("wrong output coin commitment\n")
		return false, errhandler.NewPrivacyErr(errhandler.VerifyCoinCommitmentOutputFailedErr, nil)
	}

	if fee > 0 {
		comFee := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenValueIndex], new(operation.Scalar).FromUint64(fee))
		comOutputValue.Add(comOutputValue, comFee)
	}

	if !operation.IsPointEqual(comInputValueSum, comOutputValue) {
		Logger.Log.Debugf("comInputValueSum: ", comInputValueSum)
		Logger.Log.Debugf("comOutputValueSum: ", comOutputValue)
		Logger.Log.Error("VERIFICATION PAYMENT PROOF: Sum of input coins' value is not equal to sum of output coins' value")
		return false, errhandler.NewPrivacyErr(errhandler.VerifyAmountPrivacyFailedErr, nil)
	}

	return true, nil
}

//HELPER FUNCTIONS
func isBadScalar(sc *operation.Scalar) bool {
	if sc == nil || !sc.ScalarValid() {
		return true
	}
	return false
}

func isBadPoint(point *operation.Point) bool {
	if point == nil || !point.PointValid() {
		return true
	}
	return false
}
