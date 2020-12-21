package debugtool

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)

const EQDLProofLength = 96
const VRFProofLength = 128

//Witness for proving equality of discrete logarithms
//i.e. g^x = a and h^x = b
type EQDLWitness struct {
	x    *privacy.Scalar
	g, h *privacy.Point
	a, b *privacy.Point
}


//Proof for discrete logarithm equality with respect to two different bases
//i.e. g^x = a and h^x = b
type EQDLProof struct {
	k      *privacy.Point
	kPrime *privacy.Point
	z      *privacy.Scalar
}

func NewEQDLWitness(x *privacy.Scalar, g, h, a, b *privacy.Point) EQDLWitness {
	return EQDLWitness{x, g, h, a, b}
}

func (eqdlProof EQDLProof) Bytes() []byte {
	res := eqdlProof.k.ToBytesS()
	res = append(res, eqdlProof.kPrime.ToBytesS()...)
	res = append(res, eqdlProof.z.ToBytesS()...)

	return res
}

func (eqdlProof EQDLProof) SetBytes(data []byte) (*EQDLProof, error) {
	if len(data) != EQDLProofLength{
		return nil, fmt.Errorf("length of EQDLProof should be equal to %v", EQDLProofLength)
	}
	k, err := new(privacy.Point).FromBytesS(data[:32])
	if err != nil{
		return nil, err
	}
	if !k.PointValid(){
		return nil, errors.New("EQDLProof Setbytes: k is not valid")
	}

	kPrime, err := new(privacy.Point).FromBytesS(data[32:64])
	if err != nil{
		return nil, err
	}
	if !kPrime.PointValid(){
		return nil, errors.New("EQDLProof Setbytes: kPrime is not valid")
	}

	z := new(privacy.Scalar).FromBytesS(data[64:])
	if !z.ScalarValid(){
		return nil, errors.New("EQDLProof Setbytes: z is not valid")
	}

	return &EQDLProof{k, kPrime, z}, nil
}

func (eqdlWitness EQDLWitness) Prove(msg []byte) *EQDLProof {
	r := privacy.RandomScalar()

	k := new(privacy.Point).ScalarMult(eqdlWitness.g, r)
	kPrime := new(privacy.Point).ScalarMult(eqdlWitness.h, r)

	msgToBeHashed := []byte{}
	msgToBeHashed = append(msgToBeHashed, msg...)
	msgToBeHashed = append(msgToBeHashed, eqdlWitness.g.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, eqdlWitness.a.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, k.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, eqdlWitness.h.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, eqdlWitness.b.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, kPrime.ToBytesS()...)

	c := privacy.HashToScalar(msgToBeHashed)

	z := new(privacy.Scalar).Add(r, new(privacy.Scalar).Mul(eqdlWitness.x, c))

	return &EQDLProof{k, kPrime, z}
}

func (eqdlProof EQDLProof) Verify(msg []byte, g, h, a, b *privacy.Point) (bool, error) {
	msgToBeHashed := []byte{}
	msgToBeHashed = append(msgToBeHashed, msg...)
	msgToBeHashed = append(msgToBeHashed, g.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, a.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, eqdlProof.k.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, h.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, b.ToBytesS()...)
	msgToBeHashed = append(msgToBeHashed, eqdlProof.kPrime.ToBytesS()...)

	c := operation.HashToScalar(msgToBeHashed)

	leftPoint1 := new(privacy.Point).Add(eqdlProof.k, new(privacy.Point).ScalarMult(a, c))
	rightPoint1 := new(privacy.Point).ScalarMult(g, eqdlProof.z)

	if !privacy.IsPointEqual(leftPoint1, rightPoint1) {
		return false, errors.New("EQDLProof: verify first statement FAILED")
	}

	leftPoint2 := new(privacy.Point).Add(eqdlProof.kPrime, new(privacy.Point).ScalarMult(b, c))
	rightPoint2 := new(privacy.Point).ScalarMult(h, eqdlProof.z)

	if !privacy.IsPointEqual(leftPoint2, rightPoint2) {
		return false, errors.New("EQDLProof: verify second statement FAILED")
	}

	return true, nil
}


//Witness for proving the validity of VRF output
//x: the secret key
//g: the base point
type VRFWitness struct {
	x *privacy.Scalar //the privateKey
	g *privacy.Point
}

type VRFProof struct {
	u         *privacy.Point
	eqdlProof *EQDLProof
}

func NewVRFWitness(x *privacy.Scalar, g *privacy.Point) VRFWitness {
	return VRFWitness{x, g}
}

func (vrfProof VRFProof) Bytes() []byte {
	res := vrfProof.u.ToBytesS()
	res = append(res, vrfProof.eqdlProof.Bytes()...)

	return res
}

func (vrfProof VRFProof) SetBytes(data []byte) (*VRFProof, error) {
	if len(data) != VRFProofLength{
		return nil, fmt.Errorf("length of EQDLProof should be equal to %v", EQDLProofLength)
	}
	u, err := new(privacy.Point).FromBytesS(data[:32])
	if err != nil{
		return nil, err
	}
	if !u.PointValid(){
		return nil, errors.New("VRFProof Setbytes: u is not valid")
	}

	eqdlProof, err := new(EQDLProof).SetBytes(data[32:])
	if err != nil{
		return nil, err
	}

	return &VRFProof{u, eqdlProof}, nil
}

//This module implements the VRF algorithm described in the Ouroboros Praos Paper
//https://eprint.iacr.org/2017/573.pdf
func (vrfWitness VRFWitness) Compute(msg []byte) (*privacy.Scalar, *VRFProof, error) {
	hPrime := privacy.HashToPoint(msg)
	u := new(privacy.Point).ScalarMult(hPrime, vrfWitness.x)

	//compute the output of the VRF, with respect to the input msg
	y := operation.HashToScalar(append(msg, u.ToBytesS()...))
	if y == nil{
		return nil, nil, errors.New("hash to scalar INVALID")
	}

	eqdlWitness := EQDLWitness{
		x: vrfWitness.x,
		g: vrfWitness.g,
		h: hPrime,
		a: new(privacy.Point).ScalarMult(vrfWitness.g, vrfWitness.x),
		b: u,
	}

	//Produce the proof for correct computation of y on input msg.
	eqdlProof := eqdlWitness.Prove(msg)

	vrfProof := VRFProof{
		u:         u,
		eqdlProof: eqdlProof,
	}

	return y, &vrfProof, nil
}

func (vrfProof VRFProof) Verify(msg []byte, g, pubKey *privacy.Point, output *privacy.Scalar) (bool, error) {
	y := operation.HashToScalar(append(msg, vrfProof.u.ToBytesS()...))
	if !privacy.IsScalarEqual(y, output) {
		return false, errors.New("VRFProof: verify first statement FAILED")
	}

	hPrime := privacy.HashToPoint(msg)
	return vrfProof.eqdlProof.Verify(msg, g, hPrime, pubKey, vrfProof.u)
}
