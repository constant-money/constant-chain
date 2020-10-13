package mlsag

import (
	"errors"
	// "bytes"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/operation"
)


func (ml *Mlsag) SignConfidentialAsset(message []byte) (*MlsagSig, error) {
	if len(message) != common.HashSize {
		return nil, errors.New("Cannot mlsag sign the message because its length is not 32, maybe it has not been hashed")
	}
	var message32byte [32]byte
	copy(message32byte[:], message)

	alpha, r := ml.createRandomChallenges()          // step 2 in paper
	c, err := ml.calculateCCA(message32byte, alpha, r) // step 3 and 4 in paper

	if err != nil {
		return nil, err
	}
	return &MlsagSig{
		c[0], ml.keyImages, r,
	}, nil
}

// func VerifyConfidentialAsset(sig *MlsagSig, K *Ring, message []byte) (bool, error) {
// 	if len(message) != common.HashSize {
// 		return false, errors.New("Cannot mlsag verify the message because its length is not 32, maybe it has not been hashed")
// 	}
// 	message32byte := [32]byte{}
// 	copy(message32byte[:], message)
// 	b1 := verifyKeyImages(sig.keyImages)
// 	b2, err := verifyRingCA(sig, K, message32byte)
// 	return (b1 && b2), err
// }

func NewMlsagCA(privateKeys []*operation.Scalar, R *Ring, pi int) *Mlsag {
	return &Mlsag{
		R,
		pi,
		ParseKeyImagesCA(privateKeys),
		privateKeys,
	}
}

func ParseKeyImagesCA(privateKeys []*operation.Scalar) []*operation.Point {
	m := len(privateKeys)

	result := make([]*operation.Point, m)
	for i := 0; i < m; i += 1 {
		publicKey := parsePublicKey(privateKeys[i], i >= m-2)
		hashPoint := operation.HashToPoint(publicKey.ToBytesS())
		result[i] = new(operation.Point).ScalarMult(hashPoint, privateKeys[i])
	}
	return result
}

func calculateNextCCA(digest [common.HashSize]byte, r []*operation.Scalar, c *operation.Scalar, K []*operation.Point, keyImages []*operation.Point) (*operation.Scalar, error) {
	if len(r) != len(K) || len(r) != len(keyImages) {
		return nil, errors.New("Error in MLSAG: Calculating next C must have length of r be the same with length of ring R and same with length of keyImages")
	}
	var b []byte
	b = append(b, digest[:]...)

	// Below is the mathematics within the Monero paper:
	// If you are reviewing my code, please refer to paper
	// rG: r*G
	// cK: c*R
	// rG_cK: rG + cK
	//
	// HK: H_p(K_i)
	// rHK: r_i*H_p(K_i)
	// cKI: c*R~ (KI as keyImage)
	// rHK_cKI: rHK + cKI

	// Process columns before the last
	for i := 0; i < len(K)-2; i += 1 {
		rG := new(operation.Point).ScalarMultBase(r[i])
		cK := new(operation.Point).ScalarMult(K[i], c)
		rG_cK := new(operation.Point).Add(rG, cK)

		HK := operation.HashToPoint(K[i].ToBytesS())
		rHK := new(operation.Point).ScalarMult(HK, r[i])
		cKI := new(operation.Point).ScalarMult(keyImages[i], c)
		rHK_cKI := new(operation.Point).Add(rHK, cKI)

		b = append(b, rG_cK.ToBytesS()...)
		b = append(b, rHK_cKI.ToBytesS()...)
	}

	// Process last column
	rG := new(operation.Point).ScalarMult(
		operation.PedCom.G[operation.PedersenRandomnessIndex],
		r[len(K)-2],
	)
	cK := new(operation.Point).ScalarMult(K[len(K)-2], c)
	rG_cK := new(operation.Point).Add(rG, cK)
	b = append(b, rG_cK.ToBytesS()...)

	rG = new(operation.Point).ScalarMult(
		operation.PedCom.G[operation.PedersenRandomnessIndex],
		r[len(K)-1],
	)
	cK = new(operation.Point).ScalarMult(K[len(K)-1], c)
	rG_cK = new(operation.Point).Add(rG, cK)
	b = append(b, rG_cK.ToBytesS()...)

	return operation.HashToScalar(b), nil
}

func calculateFirstCCA(digest [common.HashSize]byte, alpha []*operation.Scalar, K []*operation.Point) (*operation.Scalar, error) {
	if len(alpha) != len(K) {
		return nil, errors.New("Error in MLSAG: Calculating first C must have length of alpha be the same with length of ring R")
	}
	var b []byte
	b = append(b, digest[:]...)

	// Process columns before the last
	for i := 0; i < len(K)-2; i += 1 {
		alphaG := new(operation.Point).ScalarMultBase(alpha[i])

		H := operation.HashToPoint(K[i].ToBytesS())
		alphaH := new(operation.Point).ScalarMult(H, alpha[i])

		b = append(b, alphaG.ToBytesS()...)
		b = append(b, alphaH.ToBytesS()...)
	}

	// Process last column
	alphaG := new(operation.Point).ScalarMult(
		// TODO : which g here ?
		operation.PedCom.G[operation.PedersenRandomnessIndex],
		alpha[len(K)-2],
	)
	b = append(b, alphaG.ToBytesS()...)
	alphaG = new(operation.Point).ScalarMult(
		operation.PedCom.G[operation.PedersenRandomnessIndex],
		alpha[len(K)-1],
	)
	b = append(b, alphaG.ToBytesS()...)

	return operation.HashToScalar(b), nil
}

func (this *Mlsag) calculateCCA(message [common.HashSize]byte, alpha []*operation.Scalar, r [][]*operation.Scalar) ([]*operation.Scalar, error) {
	m := len(this.privateKeys)
	n := len(this.R.keys)

	c := make([]*operation.Scalar, n)
	firstC, err := calculateFirstCCA(
		message,
		alpha,
		this.R.keys[this.pi],
	)
	if err != nil {
		return nil, err
	}

	var i int = (this.pi + 1) % n
	c[i] = firstC
	for next := (i + 1) % n; i != this.pi; {
		nextC, err := calculateNextCCA(
			message,
			r[i], c[i],
			(*this.R).keys[i],
			this.keyImages,
		)
		if err != nil {
			return nil, err
		}
		c[next] = nextC
		i = next
		next = (next + 1) % n
	}

	for i := 0; i < m; i += 1 {
		ck := new(operation.Scalar).Mul(c[this.pi], this.privateKeys[i])
		r[this.pi][i] = new(operation.Scalar).Sub(alpha[i], ck)
	}


	return c, nil
}

// func verifyRingCA(sig *MlsagSig, R *Ring, message [common.HashSize]byte) (bool, error) {
// 	c := *sig.c
// 	cBefore := *sig.c
// 	if len(R.keys) != len(sig.r){
// 		return false, errors.New("MLSAG Error : Malformed Ring")
// 	}
// 	for i := 0; i < len(sig.r); i += 1 {
// 		nextC, err := calculateNextCCA(
// 			message,
// 			sig.r[i], &c,
// 			R.keys[i],
// 			sig.keyImages,
// 		)
// 		if err != nil {
// 			return false, err
// 		}
// 		c = *nextC
// 	}
// 	return bytes.Equal(c.ToBytesS(), cBefore.ToBytesS()), nil
// }