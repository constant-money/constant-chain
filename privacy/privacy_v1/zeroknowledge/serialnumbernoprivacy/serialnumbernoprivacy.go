package serialnumbernoprivacy

import (
	"errors"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/utils"
)

type SerialNumberNoPrivacyStatement struct {
	output *operation.Point
	vKey   *operation.Point
	input  *operation.Scalar
}

// SNNoPrivacyWitness is a protocol for Zero-knowledge Proof of Knowledge of one out of many commitments containing 0
// include Witness: CommitedValue, r []byte
type SNNoPrivacyWitness struct {
	stmt SerialNumberNoPrivacyStatement
	seed *operation.Scalar
}

// serialNumberNNoPrivacyProof contains Proof's value
type SNNoPrivacyProof struct {
	// general info
	stmt SerialNumberNoPrivacyStatement

	tSeed   *operation.Point
	tOutput *operation.Point

	zSeed *operation.Scalar
}

func (proof SNNoPrivacyProof) GetVKey() *operation.Point {
	return proof.stmt.vKey
}

func (proof SNNoPrivacyProof) GetOutput() *operation.Point{
	return proof.stmt.output
}

func (proof SNNoPrivacyProof) GetInput() *operation.Scalar{
	return proof.stmt.input
}


func (proof SNNoPrivacyProof) ValidateSanity() bool {
	if !proof.stmt.output.PointValid() {
		return false
	}
	if !proof.stmt.vKey.PointValid() {
		return false
	}
	if !proof.stmt.input.ScalarValid() {
		return false
	}

	if !proof.tSeed.PointValid() {
		return false
	}
	if !proof.tOutput.PointValid() {
		return false
	}
	return proof.zSeed.ScalarValid()
}

func (pro SNNoPrivacyProof) isNil() bool {
	if pro.stmt.output == nil {
		return true
	}
	if pro.stmt.vKey == nil {
		return true
	}
	if pro.stmt.input == nil {
		return true
	}
	if pro.tSeed == nil {
		return true
	}
	if pro.tOutput == nil {
		return true
	}
	if pro.zSeed == nil {
		return true
	}
	return false
}

func (pro *SNNoPrivacyProof) Init() *SNNoPrivacyProof {
	pro.stmt.output = new(operation.Point)
	pro.stmt.vKey = new(operation.Point)
	pro.stmt.input = new(operation.Scalar)

	pro.tSeed = new(operation.Point)
	pro.tOutput = new(operation.Point)

	pro.zSeed = new(operation.Scalar)

	return pro
}

// Set sets Witness
func (wit *SNNoPrivacyWitness) Set(
	output *operation.Point,
	vKey *operation.Point,
	input *operation.Scalar,
	seed *operation.Scalar) {

	if wit == nil {
		wit = new(SNNoPrivacyWitness)
	}

	wit.stmt.output = output
	wit.stmt.vKey = vKey
	wit.stmt.input = input

	wit.seed = seed
}

// Set sets Proof
func (pro *SNNoPrivacyProof) Set(
	output *operation.Point,
	vKey *operation.Point,
	input *operation.Scalar,
	tSeed *operation.Point,
	tOutput *operation.Point,
	zSeed *operation.Scalar) {

	if pro == nil {
		pro = new(SNNoPrivacyProof)
	}

	pro.stmt.output = output
	pro.stmt.vKey = vKey
	pro.stmt.input = input

	pro.tSeed = tSeed
	pro.tOutput = tOutput

	pro.zSeed = zSeed
}

func (pro SNNoPrivacyProof) Bytes() []byte {
	// if proof is nil, return an empty array
	if pro.isNil() {
		return []byte{}
	}

	var bytes []byte
	bytes = append(bytes, pro.stmt.output.ToBytesS()...)
	bytes = append(bytes, pro.stmt.vKey.ToBytesS()...)
	bytes = append(bytes, pro.stmt.input.ToBytesS()...)

	bytes = append(bytes, pro.tSeed.ToBytesS()...)
	bytes = append(bytes, pro.tOutput.ToBytesS()...)

	bytes = append(bytes, pro.zSeed.ToBytesS()...)

	return bytes
}

func (pro *SNNoPrivacyProof) SetBytes(bytes []byte) error {
	// if len(bytes) == 0 {
	// 	return errors.New("Bytes array is empty")
	// }
	if len(bytes) < operation.Ed25519KeySize*6{
		return errors.New("Not enough bytes to unmarshal Serial Number No Privacy Proof")
	}

	offset := 0
	var err error
	pro.stmt.output, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += operation.Ed25519KeySize

	pro.stmt.vKey, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += operation.Ed25519KeySize

	pro.stmt.input.FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	offset += operation.Ed25519KeySize

	pro.tSeed, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += operation.Ed25519KeySize

	pro.tOutput, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += operation.Ed25519KeySize

	pro.zSeed = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])

	return nil
}

func (wit SNNoPrivacyWitness) Prove(mess []byte) (*SNNoPrivacyProof, error) {
	// randomness
	eSK := operation.RandomScalar()

	// calculate tSeed = g_SK^eSK
	tSK := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], eSK)

	// calculate tOutput = sn^eSK
	tE := new(operation.Point).ScalarMult(wit.stmt.output, eSK)

	x := new(operation.Scalar)
	if mess == nil {
		// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
		// recheck frombytes is valid scalar
		x = utils.GenerateChallenge([][]byte{wit.stmt.output.ToBytesS(), wit.stmt.vKey.ToBytesS(), tSK.ToBytesS(), tE.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Calculate zSeed = SK * x + eSK
	zSK := new(operation.Scalar).Mul(wit.seed, x)
	zSK.Add(zSK, eSK)

	proof := new(SNNoPrivacyProof).Init()
	proof.Set(wit.stmt.output, wit.stmt.vKey, wit.stmt.input, tSK, tE, zSK)
	return proof, nil
}

func (pro SNNoPrivacyProof) Verify(mess []byte) (bool, error) {
	// re-calculate x = hash(tSeed || tOutput)
	x := new(operation.Scalar)
	if mess == nil {
		// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
		x = utils.GenerateChallenge([][]byte{pro.stmt.output.ToBytesS(), pro.stmt.vKey.ToBytesS(), pro.tSeed.ToBytesS(), pro.tOutput.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Check gSK^zSeed = vKey^x * tSeed
	leftPoint1 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], pro.zSeed)

	rightPoint1 := new(operation.Point).ScalarMult(pro.stmt.vKey, x)
	rightPoint1 = rightPoint1.Add(rightPoint1, pro.tSeed)

	if !operation.IsPointEqual(leftPoint1, rightPoint1) {
		Logger.Log.Errorf("verify serial number no privacy proof statement 1 failed")
		return false, errors.New("verify serial number no privacy proof statement 1 failed")
	}

	// Check sn^(zSeed + x*input) = gSK^x * tOutput
	tmp := new(operation.Scalar).Add(pro.zSeed, new(operation.Scalar).Mul(x, pro.stmt.input))
	leftPoint2 := new(operation.Point).ScalarMult(pro.stmt.output, tmp)

	rightPoint2 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], x)
	rightPoint2 = rightPoint2.Add(rightPoint2, pro.tOutput)

	if !operation.IsPointEqual(leftPoint2, rightPoint2) {
		Logger.Log.Errorf("verify serial number no privacy proof statement 2 failed")
		return false, errors.New("verify serial number no privacy proof statement 2 failed")
	}

	return true, nil
}

func (pro SNNoPrivacyProof) VerifyOld(mess []byte) (bool, error) {
	// re-calculate x = hash(tSeed || tOutput)
	x := new(operation.Scalar)
	if mess == nil {
		// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
		x = utils.GenerateChallenge([][]byte{pro.tSeed.ToBytesS(), pro.tOutput.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Check gSK^zSeed = vKey^x * tSeed
	leftPoint1 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], pro.zSeed)

	rightPoint1 := new(operation.Point).ScalarMult(pro.stmt.vKey, x)
	rightPoint1 = rightPoint1.Add(rightPoint1, pro.tSeed)

	if !operation.IsPointEqual(leftPoint1, rightPoint1) {
		Logger.Log.Errorf("verifyOld serial number no privacy proof statement 1 failed")
		return false, errors.New("verifyOld serial number no privacy proof statement 1 failed")
	}

	// Check sn^(zSeed + x*input) = gSK^x * tOutput
	tmp := new(operation.Scalar).Add(pro.zSeed, new(operation.Scalar).Mul(x, pro.stmt.input))
	leftPoint2 := new(operation.Point).ScalarMult(pro.stmt.output, tmp)

	rightPoint2 := new(operation.Point).ScalarMult(operation.PedCom.G[operation.PedersenPrivateKeyIndex], x)
	rightPoint2 = rightPoint2.Add(rightPoint2, pro.tOutput)

	if !operation.IsPointEqual(leftPoint2, rightPoint2) {
		Logger.Log.Errorf("verifyOld serial number no privacy proof statement 2 failed")
		return false, errors.New("verifyOld serial number no privacy proof statement 2 failed")
	}

	return true, nil
}
