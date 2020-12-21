package serialnumberprivacy

import (
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/utils"
)

type SerialNumberPrivacyStatement struct {
	sn       *operation.Point // serial number
	comSK    *operation.Point // commitment to private key
	comInput *operation.Point // commitment to input of the pseudo-random function
}

type SNPrivacyWitness struct {
	stmt *SerialNumberPrivacyStatement // statement to be proved

	sk     *operation.Scalar // private key
	rSK    *operation.Scalar // blinding factor in the commitment to private key
	input  *operation.Scalar // input of pseudo-random function
	rInput *operation.Scalar // blinding factor in the commitment to input
}

type SNPrivacyProof struct {
	stmt *SerialNumberPrivacyStatement // statement to be proved

	tSK    *operation.Point // random commitment related to private key
	tInput *operation.Point // random commitment related to input
	tSN    *operation.Point // random commitment related to serial number

	zSK     *operation.Scalar // first challenge-dependent information to open the commitment to private key
	zRSK    *operation.Scalar // second challenge-dependent information to open the commitment to private key
	zInput  *operation.Scalar // first challenge-dependent information to open the commitment to input
	zRInput *operation.Scalar // second challenge-dependent information to open the commitment to input
}

// ValidateSanity validates sanity of proof
func (proof SNPrivacyProof) ValidateSanity() bool {
	if !proof.stmt.sn.PointValid() {
		return false
	}
	if !proof.stmt.comSK.PointValid() {
		return false
	}
	if !proof.stmt.comInput.PointValid() {
		return false
	}
	if !proof.tSK.PointValid() {
		return false
	}
	if !proof.tInput.PointValid() {
		return false
	}
	if !proof.tSN.PointValid() {
		return false
	}
	if !proof.zSK.ScalarValid() {
		return false
	}
	if !proof.zRSK.ScalarValid() {
		return false
	}
	if !proof.zInput.ScalarValid() {
		return false
	}
	if !proof.zRInput.ScalarValid() {
		return false
	}
	return true
}

func (proof SNPrivacyProof) isNil() bool {
	if proof.stmt.sn == nil {
		return true
	}
	if proof.stmt.comSK == nil {
		return true
	}
	if proof.stmt.comInput == nil {
		return true
	}
	if proof.tSK == nil {
		return true
	}
	if proof.tInput == nil {
		return true
	}
	if proof.tSN == nil {
		return true
	}
	if proof.zSK == nil {
		return true
	}
	if proof.zRSK == nil {
		return true
	}
	if proof.zInput == nil {
		return true
	}
	return proof.zRInput == nil
}

// Init inits Proof
func (proof *SNPrivacyProof) Init() *SNPrivacyProof {
	proof.stmt = new(SerialNumberPrivacyStatement)

	proof.tSK = new(operation.Point)
	proof.tInput = new(operation.Point)
	proof.tSN = new(operation.Point)

	proof.zSK = new(operation.Scalar)
	proof.zRSK = new(operation.Scalar)
	proof.zInput = new(operation.Scalar)
	proof.zRInput = new(operation.Scalar)

	return proof
}

func (proof SNPrivacyProof) GetComSK() *operation.Point {
	return proof.stmt.comSK
}

func (proof SNPrivacyProof) GetSN() *operation.Point {
	return proof.stmt.sn
}

func (proof SNPrivacyProof) GetComInput() *operation.Point {
	return proof.stmt.comInput
}



// Set sets Statement
func (stmt *SerialNumberPrivacyStatement) Set(
	SN *operation.Point,
	comSK *operation.Point,
	comInput *operation.Point) {
	stmt.sn = SN
	stmt.comSK = comSK
	stmt.comInput = comInput
}

// Set sets Witness
func (wit *SNPrivacyWitness) Set(
	stmt *SerialNumberPrivacyStatement,
	SK *operation.Scalar,
	rSK *operation.Scalar,
	input *operation.Scalar,
	rInput *operation.Scalar) {

	wit.stmt = stmt
	wit.sk = SK
	wit.rSK = rSK
	wit.input = input
	wit.rInput = rInput
}

// Set sets Proof
func (proof *SNPrivacyProof) Set(
	stmt *SerialNumberPrivacyStatement,
	tSK *operation.Point,
	tInput *operation.Point,
	tSN *operation.Point,
	zSK *operation.Scalar,
	zRSK *operation.Scalar,
	zInput *operation.Scalar,
	zRInput *operation.Scalar) {
	proof.stmt = stmt
	proof.tSK = tSK
	proof.tInput = tInput
	proof.tSN = tSN

	proof.zSK = zSK
	proof.zRSK = zRSK
	proof.zInput = zInput
	proof.zRInput = zRInput
}

func (proof SNPrivacyProof) Bytes() []byte {
	// if proof is nil, return an empty array
	if proof.isNil() {
		return []byte{}
	}

	var bytes []byte
	bytes = append(bytes, proof.stmt.sn.ToBytesS()...)
	bytes = append(bytes, proof.stmt.comSK.ToBytesS()...)
	bytes = append(bytes, proof.stmt.comInput.ToBytesS()...)

	bytes = append(bytes, proof.tSK.ToBytesS()...)
	bytes = append(bytes, proof.tInput.ToBytesS()...)
	bytes = append(bytes, proof.tSN.ToBytesS()...)

	bytes = append(bytes, proof.zSK.ToBytesS()...)
	bytes = append(bytes, proof.zRSK.ToBytesS()...)
	bytes = append(bytes, proof.zInput.ToBytesS()...)
	bytes = append(bytes, proof.zRInput.ToBytesS()...)

	return bytes
}

func (proof *SNPrivacyProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return errors.New("Bytes array is empty")
	}
	if len(bytes) < 9*operation.Ed25519KeySize{
		return errors.New("Not enough bytes to unmarshal Serial Number Proof")
	}

	offset := 0
	var err error

	proof.stmt.sn = new(operation.Point)
	proof.stmt.sn, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += operation.Ed25519KeySize

	proof.stmt.comSK = new(operation.Point)
	proof.stmt.comSK, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += operation.Ed25519KeySize
	proof.stmt.comInput = new(operation.Point)
	proof.stmt.comInput, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += operation.Ed25519KeySize
	proof.tSK = new(operation.Point)
	proof.tSK, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += operation.Ed25519KeySize
	proof.tInput = new(operation.Point)
	proof.tInput, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += operation.Ed25519KeySize
	proof.tSN = new(operation.Point)
	proof.tSN, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += operation.Ed25519KeySize
	proof.zSK = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])

	offset += operation.Ed25519KeySize
	proof.zRSK = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])

	offset += operation.Ed25519KeySize
	proof.zInput = new(operation.Scalar).FromBytesS(bytes[offset : offset+common.BigIntSize])

	offset += operation.Ed25519KeySize
	proof.zRInput = new(operation.Scalar).FromBytesS(bytes[offset : offset+common.BigIntSize])

	return nil
}

func (wit SNPrivacyWitness) Prove(mess []byte) (*SNPrivacyProof, error) {

	eSK := operation.RandomScalar()
	eSND := operation.RandomScalar()
	dSK := operation.RandomScalar()
	dSND := operation.RandomScalar()

	// calculate tSeed = g_SK^eSK * h^dSK
	tSeed := operation.PedCom.CommitAtIndex(eSK, dSK, operation.PedersenPrivateKeyIndex)

	// calculate tSND = g_SND^eSND * h^dSND
	tInput := operation.PedCom.CommitAtIndex(eSND, dSND, operation.PedersenSndIndex)

	// calculate tSND = g_SK^eSND * h^dSND2
	tOutput := new(operation.Point).ScalarMult(wit.stmt.sn, new(operation.Scalar).Add(eSK, eSND))

	// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
	x := new(operation.Scalar)
	if mess == nil {
		x = utils.GenerateChallenge([][]byte{
			wit.stmt.sn.ToBytesS(),
			wit.stmt.comSK.ToBytesS(),
			tSeed.ToBytesS(),
			tInput.ToBytesS(),
			tOutput.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Calculate zSeed = sk * x + eSK
	zSeed := new(operation.Scalar).Mul(wit.sk, x)
	zSeed.Add(zSeed, eSK)
	//zSeed.Mod(zSeed, operation.Curve.Params().N)

	// Calculate zRSeed = rSK * x + dSK
	zRSeed := new(operation.Scalar).Mul(wit.rSK, x)
	zRSeed.Add(zRSeed, dSK)
	//zRSeed.Mod(zRSeed, operation.Curve.Params().N)

	// Calculate zInput = input * x + eSND
	zInput := new(operation.Scalar).Mul(wit.input, x)
	zInput.Add(zInput, eSND)
	//zInput.Mod(zInput, operation.Curve.Params().N)

	// Calculate zRInput = rInput * x + dSND
	zRInput := new(operation.Scalar).Mul(wit.rInput, x)
	zRInput.Add(zRInput, dSND)
	//zRInput.Mod(zRInput, operation.Curve.Params().N)

	proof := new(SNPrivacyProof).Init()
	proof.Set(wit.stmt, tSeed, tInput, tOutput, zSeed, zRSeed, zInput, zRInput)
	return proof, nil
}

func (proof SNPrivacyProof) Verify(mess []byte) (bool, error) {
	// re-calculate x = hash(tSeed || tInput || tSND2 || tOutput)
	x := new(operation.Scalar)
	if mess == nil {
		x = utils.GenerateChallenge([][]byte{
			proof.stmt.sn.ToBytesS(),
			proof.stmt.comSK.ToBytesS(),
			proof.tSK.ToBytesS(),
			proof.tInput.ToBytesS(),
			proof.tSN.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Check gSND^zInput * h^zRInput = input^x * tInput
	leftPoint1 := operation.PedCom.CommitAtIndex(proof.zInput, proof.zRInput, operation.PedersenSndIndex)

	rightPoint1 := new(operation.Point).ScalarMult(proof.stmt.comInput, x)
	rightPoint1.Add(rightPoint1, proof.tInput)

	if !operation.IsPointEqual(leftPoint1, rightPoint1) {
		//Logger.Log.Errorf("verify serial number privacy proof statement 1 failed")
		return false, errors.New("verify serial number privacy proof statement 1 failed")
	}

	// Check gSK^zSeed * h^zRSeed = vKey^x * tSeed
	leftPoint2 := operation.PedCom.CommitAtIndex(proof.zSK, proof.zRSK, operation.PedersenPrivateKeyIndex)

	rightPoint2 := new(operation.Point).ScalarMult(proof.stmt.comSK, x)
	rightPoint2.Add(rightPoint2, proof.tSK)

	if !operation.IsPointEqual(leftPoint2, rightPoint2) {
		Logger.Log.Errorf("verify serial number privacy proof statement 2 failed")
		return false, errors.New("verify serial number privacy proof statement 2 failed")
	}

	// Check sn^(zSeed + zInput) = gSK^x * tOutput
	leftPoint3 := new(operation.Point).ScalarMult(proof.stmt.sn, new(operation.Scalar).Add(proof.zSK, proof.zInput))

	rightPoint3 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], x)
	rightPoint3.Add(rightPoint3, proof.tSN)

	if !operation.IsPointEqual(leftPoint3, rightPoint3) {
		//privacy.Logger.Log.Errorf("verify serial number privacy proof statement 3 failed")
		return false, errors.New("verify serial number privacy proof statement 3 failed")
	}

	return true, nil
}

func (proof SNPrivacyProof) VerifyOld(mess []byte) (bool, error) {
	// re-calculate x = hash(tSeed || tInput || tSND2 || tOutput)
	x := new(operation.Scalar)
	if mess == nil {
		x = utils.GenerateChallenge([][]byte{
			proof.tSK.ToBytesS(),
			proof.tInput.ToBytesS(),
			proof.tSN.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Check gSND^zInput * h^zRInput = input^x * tInput
	leftPoint1 := operation.PedCom.CommitAtIndex(proof.zInput, proof.zRInput, operation.PedersenSndIndex)

	rightPoint1 := new(operation.Point).ScalarMult(proof.stmt.comInput, x)
	rightPoint1.Add(rightPoint1, proof.tInput)

	if !operation.IsPointEqual(leftPoint1, rightPoint1) {
		//Logger.Log.Errorf("verify serial number privacy proof statement 1 failed")
		return false, errors.New("verifyOld serial number privacy proof statement 1 failed")
	}

	// Check gSK^zSeed * h^zRSeed = vKey^x * tSeed
	leftPoint2 := operation.PedCom.CommitAtIndex(proof.zSK, proof.zRSK, operation.PedersenPrivateKeyIndex)

	rightPoint2 := new(operation.Point).ScalarMult(proof.stmt.comSK, x)
	rightPoint2.Add(rightPoint2, proof.tSK)

	if !operation.IsPointEqual(leftPoint2, rightPoint2) {
		Logger.Log.Errorf("verifyOld serial number privacy proof statement 2 failed")
		return false, errors.New("verifyOld serial number privacy proof statement 2 failed")
	}

	// Check sn^(zSeed + zInput) = gSK^x * tOutput
	leftPoint3 := new(operation.Point).ScalarMult(proof.stmt.sn, new(operation.Scalar).Add(proof.zSK, proof.zInput))

	rightPoint3 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], x)
	rightPoint3.Add(rightPoint3, proof.tSN)

	if !operation.IsPointEqual(leftPoint3, rightPoint3) {
		//privacy.Logger.Log.Errorf("verify serial number privacy proof statement 3 failed")
		return false, errors.New("verifyOld serial number privacy proof statement 3 failed")
	}

	return true, nil
}


func Copy(proof SNPrivacyProof) *SNPrivacyProof{
	tmpProof := new(SNPrivacyProof)
	tmpProof.tInput = new(operation.Point).Set(proof.tInput)
	tmpProof.tSK = new(operation.Point).Set(proof.tSK)
	tmpProof.tSN = new(operation.Point).Set(proof.tSN)
	tmpProof.zInput = new(operation.Scalar).Set(proof.zInput)
	tmpProof.zRInput = new(operation.Scalar).Set(proof.zRInput)
	tmpProof.zSK = new(operation.Scalar).Set(proof.zSK)
	tmpProof.zRSK = new(operation.Scalar).Set(proof.zRSK)

	sn := new(operation.Point).Set(proof.stmt.sn)
	comSK := new(operation.Point).Set(proof.stmt.comSK)
	comInput := new(operation.Point).Set(proof.stmt.comInput)
	tmpProof.stmt = new(SerialNumberPrivacyStatement)
	tmpProof.stmt.Set(sn, comSK, comInput)

	return tmpProof
}
