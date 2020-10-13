package bulletproofs

import (
	"errors"
	// "math"

	"github.com/incognitochain/incognito-chain/privacy/operation"
)

type InnerProductWitness struct {
	a []*operation.Scalar
	b []*operation.Scalar
	p *operation.Point
}

type InnerProductProof struct {
	l []*operation.Point
	r []*operation.Point
	a *operation.Scalar
	b *operation.Scalar
	p *operation.Point
}

func (inner *InnerProductProof) Init() *InnerProductProof {
	if inner == nil {
		inner = new(InnerProductProof)
	}
	inner.l = []*operation.Point{}
	inner.r = []*operation.Point{}
	inner.a = new(operation.Scalar)
	inner.b = new(operation.Scalar)
	inner.p = new(operation.Point).Identity()

	return inner
}

// func (proof InnerProductProof) ValidateSanity() bool {
// 	if len(proof.l) != len(proof.r) {
// 		return false
// 	}
// 	for i := 0; i < len(proof.l); i++ {
// 		if !proof.l[i].PointValid() || !proof.r[i].PointValid() {
// 			return false
// 		}
// 	}
// 	if !proof.a.ScalarValid() || !proof.b.ScalarValid() {
// 		return false
// 	}
// 	return proof.p.PointValid()
// }

func (proof InnerProductProof) Bytes() []byte {
	var res []byte

	res = append(res, byte(len(proof.l)))
	for _, l := range proof.l {
		res = append(res, l.ToBytesS()...)
	}

	for _, r := range proof.r {
		res = append(res, r.ToBytesS()...)
	}

	res = append(res, proof.a.ToBytesS()...)
	res = append(res, proof.b.ToBytesS()...)
	res = append(res, proof.p.ToBytesS()...)

	return res
}

func (proof *InnerProductProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	lenLArray := int(bytes[0])
	offset := 1
	var err error

	proof.l = make([]*operation.Point, lenLArray)
	for i := 0; i < lenLArray; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("Inner Product Proof byte unmarshaling failed")
		}
		proof.l[i], err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset += operation.Ed25519KeySize
	}

	proof.r = make([]*operation.Point, lenLArray)
	for i := 0; i < lenLArray; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("Inner Product Proof byte unmarshaling failed")
		}
		proof.r[i], err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset += operation.Ed25519KeySize
	}

	if offset+operation.Ed25519KeySize > len(bytes){
		return errors.New("Inner Product Proof byte unmarshaling failed")
	}
	proof.a = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	offset += operation.Ed25519KeySize

	if offset+operation.Ed25519KeySize > len(bytes){
		return errors.New("Inner Product Proof byte unmarshaling failed")
	}
	proof.b = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	offset += operation.Ed25519KeySize

	if offset+operation.Ed25519KeySize > len(bytes){
		return errors.New("Inner Product Proof byte unmarshaling failed")
	}
	proof.p, err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
	if err != nil {
		return err
	}

	return nil
}

func (wit InnerProductWitness) Prove(GParam []*operation.Point, HParam []*operation.Point, uParam *operation.Point, hashCache []byte) (*InnerProductProof, error) {
	if len(wit.a) != len(wit.b) {
		return nil, errors.New("invalid inputs")
	}

	N := len(wit.a)

	a := make([]*operation.Scalar, N)
	b := make([]*operation.Scalar, N)

	for i := range wit.a {
		a[i] = new(operation.Scalar).Set(wit.a[i])
		b[i] = new(operation.Scalar).Set(wit.b[i])
	}

	p := new(operation.Point).Set(wit.p)
	G := make([]*operation.Point, N)
	H := make([]*operation.Point, N)
	for i := range G {
		G[i] = new(operation.Point).Set(GParam[i])
		H[i] = new(operation.Point).Set(HParam[i])
	}

	proof := new(InnerProductProof)
	proof.l = make([]*operation.Point, 0)
	proof.r = make([]*operation.Point, 0)
	proof.p = new(operation.Point).Set(wit.p)

	for N > 1 {
		nPrime := N / 2

		cL, err := innerProduct(a[:nPrime], b[nPrime:])
		if err != nil {
			return nil, err
		}
		cR, err := innerProduct(a[nPrime:], b[:nPrime])
		if err != nil {
			return nil, err
		}

		L, err := encodeVectors(a[:nPrime], b[nPrime:], G[nPrime:], H[:nPrime])
		if err != nil {
			return nil, err
		}
		L.Add(L, new(operation.Point).ScalarMult(uParam, cL))
		proof.l = append(proof.l, L)

		R, err := encodeVectors(a[nPrime:], b[:nPrime], G[:nPrime], H[nPrime:])
		if err != nil {
			return nil, err
		}
		R.Add(R, new(operation.Point).ScalarMult(uParam, cR))
		proof.r = append(proof.r, R)

		x := generateChallenge(hashCache, []*operation.Point{L, R})
		hashCache = new(operation.Scalar).Set(x).ToBytesS()

		xInverse := new(operation.Scalar).Invert(x)
		xSquare := new(operation.Scalar).Mul(x, x)
		xSquareInverse := new(operation.Scalar).Mul(xInverse, xInverse)

		// calculate GPrime, HPrime, PPrime for the next loop
		GPrime := make([]*operation.Point, nPrime)
		HPrime := make([]*operation.Point, nPrime)

		for i := range GPrime {
			GPrime[i] = new(operation.Point).AddPedersen(xInverse, G[i], x, G[i+nPrime])
			HPrime[i] = new(operation.Point).AddPedersen(x, H[i], xInverse, H[i+nPrime])
		}

		// x^2 * l + P + xInverse^2 * r
		PPrime := new(operation.Point).AddPedersen(xSquare, L, xSquareInverse, R)
		PPrime.Add(PPrime, p)

		// calculate aPrime, bPrime
		aPrime := make([]*operation.Scalar, nPrime)
		bPrime := make([]*operation.Scalar, nPrime)

		for i := range aPrime {
			aPrime[i] = new(operation.Scalar).Mul(a[i], x)
			aPrime[i] = new(operation.Scalar).MulAdd(a[i+nPrime], xInverse, aPrime[i])

			bPrime[i] = new(operation.Scalar).Mul(b[i], xInverse)
			bPrime[i] = new(operation.Scalar).MulAdd(b[i+nPrime], x, bPrime[i])
		}

		a = aPrime
		b = bPrime
		p.Set(PPrime)
		G = GPrime
		H = HPrime
		N = nPrime
	}

	proof.a = new(operation.Scalar).Set(a[0])
	proof.b = new(operation.Scalar).Set(b[0])

	return proof, nil
}
// func (proof InnerProductProof) Verify(GParam []*operation.Point, HParam []*operation.Point, uParam *operation.Point, hashCache []byte) bool {
// 	//var aggParam = newBulletproofParams(1)
// 	p := new(operation.Point)
// 	p.Set(proof.p)

// 	n := len(GParam)
// 	G := make([]*operation.Point, n)
// 	H := make([]*operation.Point, n)
// 	for i := range G {
// 		G[i] = new(operation.Point).Set(GParam[i])
// 		H[i] = new(operation.Point).Set(HParam[i])
// 	}

// 	for i := range proof.l {
// 		nPrime := n / 2
// 		x := generateChallenge(hashCache, []*operation.Point{proof.l[i], proof.r[i]})
// 		hashCache = new(operation.Scalar).Set(x).ToBytesS()
// 		xInverse := new(operation.Scalar).Invert(x)
// 		xSquare := new(operation.Scalar).Mul(x, x)
// 		xSquareInverse := new(operation.Scalar).Mul(xInverse, xInverse)

// 		// calculate GPrime, HPrime, PPrime for the next loop
// 		GPrime := make([]*operation.Point, nPrime)
// 		HPrime := make([]*operation.Point, nPrime)

// 		for j := 0; j < len(GPrime); j++ {
// 			GPrime[j] = new(operation.Point).AddPedersen(xInverse, G[j], x, G[j+nPrime])
// 			HPrime[j] = new(operation.Point).AddPedersen(x, H[j], xInverse, H[j+nPrime])
// 		}
// 		// calculate x^2 * l + P + xInverse^2 * r
// 		PPrime := new(operation.Point).AddPedersen(xSquare, proof.l[i], xSquareInverse, proof.r[i])
// 		PPrime.Add(PPrime, p)

// 		p = PPrime
// 		G = GPrime
// 		H = HPrime
// 		n = nPrime
// 	}

// 	c := new(operation.Scalar).Mul(proof.a, proof.b)
// 	rightPoint := new(operation.Point).AddPedersen(proof.a, G[0], proof.b, H[0])
// 	rightPoint.Add(rightPoint, new(operation.Point).ScalarMult(uParam, c))
// 	res := operation.IsPointEqual(rightPoint, p)
// 	if !res {
// 		Logger.Log.Error("Inner product argument failed:")
// 		Logger.Log.Error("p: %v\n", p)
// 		Logger.Log.Error("RightPoint: %v\n", rightPoint)
// 	}

// 	return res
// }

// func (proof InnerProductProof) VerifyFaster(GParam []*operation.Point, HParam []*operation.Point, uParam *operation.Point, hashCache []byte) bool {
// 	//var aggParam = newBulletproofParams(1)
// 	p := new(operation.Point)
// 	p.Set(proof.p)
// 	n := len(GParam)
// 	G := make([]*operation.Point, n)
// 	H := make([]*operation.Point, n)
// 	s := make([]*operation.Scalar, n)
// 	sInverse := make([]*operation.Scalar, n)

// 	for i := range G {
// 		G[i] = new(operation.Point).Set(GParam[i])
// 		H[i] = new(operation.Point).Set(HParam[i])
// 		s[i] = new(operation.Scalar).FromUint64(1)
// 		sInverse[i] = new(operation.Scalar).FromUint64(1)
// 	}
// 	logN := int(math.Log2(float64(n)))
// 	xList := make([]*operation.Scalar, logN)
// 	xInverseList := make([]*operation.Scalar, logN)
// 	xSquareList := make([]*operation.Scalar, logN)
// 	xInverseSquare_List := make([]*operation.Scalar, logN)

// 	//a*s ; b*s^-1

// 	for i := range proof.l {
// 		// calculate challenge x = hash(hash(G || H || u || p) || x || l || r)
// 		xList[i] = generateChallenge(hashCache, []*operation.Point{proof.l[i], proof.r[i]})
// 		hashCache = new(operation.Scalar).Set(xList[i]).ToBytesS()

// 		xInverseList[i] = new(operation.Scalar).Invert(xList[i])
// 		xSquareList[i] = new(operation.Scalar).Mul(xList[i], xList[i])
// 		xInverseSquare_List[i] = new(operation.Scalar).Mul(xInverseList[i], xInverseList[i])

// 		//Update s, s^-1
// 		for j := 0; j < n; j++ {
// 			if j&int(math.Pow(2, float64(logN-i-1))) != 0 {
// 				s[j] = new(operation.Scalar).Mul(s[j], xList[i])
// 				sInverse[j] = new(operation.Scalar).Mul(sInverse[j], xInverseList[i])
// 			} else {
// 				s[j] = new(operation.Scalar).Mul(s[j], xInverseList[i])
// 				sInverse[j] = new(operation.Scalar).Mul(sInverse[j], xList[i])
// 			}
// 		}
// 	}

// 	// Compute (g^s)^a (h^-s)^b u^(ab) = p l^(x^2) r^(-x^2)
// 	c := new(operation.Scalar).Mul(proof.a, proof.b)
// 	rightHSPart1 := new(operation.Point).MultiScalarMult(s, G)
// 	rightHSPart1.ScalarMult(rightHSPart1, proof.a)
// 	rightHSPart2 := new(operation.Point).MultiScalarMult(sInverse, H)
// 	rightHSPart2.ScalarMult(rightHSPart2, proof.b)

// 	rightHS := new(operation.Point).Add(rightHSPart1, rightHSPart2)
// 	rightHS.Add(rightHS, new(operation.Point).ScalarMult(uParam, c))

// 	leftHSPart1 := new(operation.Point).MultiScalarMult(xSquareList, proof.l)
// 	leftHSPart2 := new(operation.Point).MultiScalarMult(xInverseSquare_List, proof.r)

// 	leftHS := new(operation.Point).Add(leftHSPart1, leftHSPart2)
// 	leftHS.Add(leftHS, proof.p)

// 	res := operation.IsPointEqual(rightHS, leftHS)
// 	if !res {
// 		Logger.Log.Error("Inner product argument failed:")
// 		Logger.Log.Error("LHS: %v\n", leftHS)
// 		Logger.Log.Error("RHS: %v\n", rightHS)
// 	}

// 	return res
// }
