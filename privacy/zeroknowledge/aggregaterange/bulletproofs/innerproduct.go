package bulletproofs

import (
	"errors"
	"github.com/incognitochain/incognito-chain/privacy"
	"math"
)

type InnerProductWitness struct {
	a []*privacy.Scalar
	b []*privacy.Scalar
	p *privacy.Point
}

type InnerProductProof struct {
	l []*privacy.Point
	r []*privacy.Point
	a *privacy.Scalar
	b *privacy.Scalar
	p *privacy.Point
}

func (inner *InnerProductProof) Init() *InnerProductProof {
	if inner == nil {
		inner = new(InnerProductProof)
	}
	inner.l = []*privacy.Point{}
	inner.r = []*privacy.Point{}
	inner.a = new(privacy.Scalar)
	inner.b = new(privacy.Scalar)
	inner.p = new(privacy.Point).Identity()

	return inner
}

func (proof InnerProductProof) ValidateSanity() bool {
	if len(proof.l) != len(proof.r) {
		return false
	}
	for i := 0; i < len(proof.l); i++ {
		if !proof.l[i].PointValid() || !proof.r[i].PointValid() {
			return false
		}
	}
	if !proof.a.ScalarValid() || !proof.b.ScalarValid() {
		return false
	}
	return proof.p.PointValid()
}

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

	proof.l = make([]*privacy.Point, lenLArray)
	for i := 0; i < lenLArray; i++ {
		if offset+privacy.Ed25519KeySize > len(bytes){
			return errors.New("Inner Product Proof byte unmarshaling failed")
		}
		proof.l[i], err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset += privacy.Ed25519KeySize
	}

	proof.r = make([]*privacy.Point, lenLArray)
	for i := 0; i < lenLArray; i++ {
		if offset+privacy.Ed25519KeySize > len(bytes){
			return errors.New("Inner Product Proof byte unmarshaling failed")
		}
		proof.r[i], err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset += privacy.Ed25519KeySize
	}

	if offset+privacy.Ed25519KeySize > len(bytes){
		return errors.New("Inner Product Proof byte unmarshaling failed")
	}
	proof.a = new(privacy.Scalar).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	offset += privacy.Ed25519KeySize

	if offset+privacy.Ed25519KeySize > len(bytes){
		return errors.New("Inner Product Proof byte unmarshaling failed")
	}
	proof.b = new(privacy.Scalar).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	offset += privacy.Ed25519KeySize

	if offset+privacy.Ed25519KeySize > len(bytes){
		return errors.New("Inner Product Proof byte unmarshaling failed")
	}
	proof.p, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}

	return nil
}

func (wit InnerProductWitness) Prove(GParam []*privacy.Point, HParam []*privacy.Point, uParam *privacy.Point, hashCache []byte) (*InnerProductProof, error) {
	if len(wit.a) != len(wit.b) {
		return nil, errors.New("invalid inputs")
	}

	N := len(wit.a)

	a := make([]*privacy.Scalar, N)
	b := make([]*privacy.Scalar, N)

	for i := range wit.a {
		a[i] = new(privacy.Scalar).Set(wit.a[i])
		b[i] = new(privacy.Scalar).Set(wit.b[i])
	}

	p := new(privacy.Point).Set(wit.p)
	G := make([]*privacy.Point, N)
	H := make([]*privacy.Point, N)
	for i := range G {
		G[i] = new(privacy.Point).Set(GParam[i])
		H[i] = new(privacy.Point).Set(HParam[i])
	}

	proof := new(InnerProductProof)
	proof.l = make([]*privacy.Point, 0)
	proof.r = make([]*privacy.Point, 0)
	proof.p = new(privacy.Point).Set(wit.p)

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
		L.Add(L, new(privacy.Point).ScalarMult(uParam, cL))
		proof.l = append(proof.l, L)

		R, err := encodeVectors(a[nPrime:], b[:nPrime], G[:nPrime], H[nPrime:])
		if err != nil {
			return nil, err
		}
		R.Add(R, new(privacy.Point).ScalarMult(uParam, cR))
		proof.r = append(proof.r, R)

		x := generateChallenge(hashCache, []*privacy.Point{L, R})
		hashCache = new(privacy.Scalar).Set(x).ToBytesS()

		xInverse := new(privacy.Scalar).Invert(x)
		xSquare := new(privacy.Scalar).Mul(x, x)
		xSquareInverse := new(privacy.Scalar).Mul(xInverse, xInverse)

		// calculate GPrime, HPrime, PPrime for the next loop
		GPrime := make([]*privacy.Point, nPrime)
		HPrime := make([]*privacy.Point, nPrime)

		for i := range GPrime {
			GPrime[i] = new(privacy.Point).AddPedersen(xInverse, G[i], x, G[i+nPrime])
			HPrime[i] = new(privacy.Point).AddPedersen(x, H[i], xInverse, H[i+nPrime])
		}

		// x^2 * l + P + xInverse^2 * r
		PPrime := new(privacy.Point).AddPedersen(xSquare, L, xSquareInverse, R)
		PPrime.Add(PPrime, p)

		// calculate aPrime, bPrime
		aPrime := make([]*privacy.Scalar, nPrime)
		bPrime := make([]*privacy.Scalar, nPrime)

		for i := range aPrime {
			aPrime[i] = new(privacy.Scalar).Mul(a[i], x)
			aPrime[i] = new(privacy.Scalar).MulAdd(a[i+nPrime], xInverse, aPrime[i])

			bPrime[i] = new(privacy.Scalar).Mul(b[i], xInverse)
			bPrime[i] = new(privacy.Scalar).MulAdd(b[i+nPrime], x, bPrime[i])
		}

		a = aPrime
		b = bPrime
		p.Set(PPrime)
		G = GPrime
		H = HPrime
		N = nPrime
	}

	proof.a = new(privacy.Scalar).Set(a[0])
	proof.b = new(privacy.Scalar).Set(b[0])

	return proof, nil
}

func (proof InnerProductProof) Verify(GParam []*privacy.Point, HParam []*privacy.Point, uParam *privacy.Point, hashCache []byte) bool {
	//var aggParam = newBulletproofParams(1)
	p := new(privacy.Point)
	p.Set(proof.p)

	n := len(GParam)
	G := make([]*privacy.Point, n)
	H := make([]*privacy.Point, n)
	for i := range G {
		G[i] = new(privacy.Point).Set(GParam[i])
		H[i] = new(privacy.Point).Set(HParam[i])
	}

	for i := range proof.l {
		nPrime := n / 2
		x := generateChallenge(hashCache, []*privacy.Point{proof.l[i], proof.r[i]})
		hashCache = new(privacy.Scalar).Set(x).ToBytesS()
		xInverse := new(privacy.Scalar).Invert(x)
		xSquare := new(privacy.Scalar).Mul(x, x)
		xSquareInverse := new(privacy.Scalar).Mul(xInverse, xInverse)

		// calculate GPrime, HPrime, PPrime for the next loop
		GPrime := make([]*privacy.Point, nPrime)
		HPrime := make([]*privacy.Point, nPrime)

		for j := 0; j < len(GPrime); j++ {
			GPrime[j] = new(privacy.Point).AddPedersen(xInverse, G[j], x, G[j+nPrime])
			HPrime[j] = new(privacy.Point).AddPedersen(x, H[j], xInverse, H[j+nPrime])
		}
		// calculate x^2 * l + P + xInverse^2 * r
		PPrime := new(privacy.Point).AddPedersen(xSquare, proof.l[i], xSquareInverse, proof.r[i])
		PPrime.Add(PPrime, p)

		p = PPrime
		G = GPrime
		H = HPrime
		n = nPrime
	}

	c := new(privacy.Scalar).Mul(proof.a, proof.b)
	rightPoint := new(privacy.Point).AddPedersen(proof.a, G[0], proof.b, H[0])
	rightPoint.Add(rightPoint, new(privacy.Point).ScalarMult(uParam, c))
	res := privacy.IsPointEqual(rightPoint, p)
	if !res {
		Logger.Log.Error("Inner product argument failed:")
		Logger.Log.Error("p: %v\n", p)
		Logger.Log.Error("RightPoint: %v\n", rightPoint)
	}

	return res
}

func (proof InnerProductProof) VerifyFaster(GParam []*privacy.Point, HParam []*privacy.Point, uParam *privacy.Point, hashCache []byte) bool {
	//var aggParam = newBulletproofParams(1)
	p := new(privacy.Point)
	p.Set(proof.p)
	n := len(GParam)
	G := make([]*privacy.Point, n)
	H := make([]*privacy.Point, n)
	s := make([]*privacy.Scalar, n)
	sInverse := make([]*privacy.Scalar, n)

	for i := range G {
		G[i] = new(privacy.Point).Set(GParam[i])
		H[i] = new(privacy.Point).Set(HParam[i])
		s[i] = new(privacy.Scalar).FromUint64(1)
		sInverse[i] = new(privacy.Scalar).FromUint64(1)
	}
	logN := int(math.Log2(float64(n)))
	xList := make([]*privacy.Scalar, logN)
	xInverseList := make([]*privacy.Scalar, logN)
	xSquareList := make([]*privacy.Scalar, logN)
	xInverseSquare_List := make([]*privacy.Scalar, logN)

	//a*s ; b*s^-1

	for i := range proof.l {
		// calculate challenge x = hash(hash(G || H || u || p) || x || l || r)
		xList[i] = generateChallenge(hashCache, []*privacy.Point{proof.l[i], proof.r[i]})
		hashCache = new(privacy.Scalar).Set(xList[i]).ToBytesS()

		xInverseList[i] = new(privacy.Scalar).Invert(xList[i])
		xSquareList[i] = new(privacy.Scalar).Mul(xList[i], xList[i])
		xInverseSquare_List[i] = new(privacy.Scalar).Mul(xInverseList[i], xInverseList[i])

		//Update s, s^-1
		for j := 0; j < n; j++ {
			if j&int(math.Pow(2, float64(logN-i-1))) != 0 {
				s[j] = new(privacy.Scalar).Mul(s[j], xList[i])
				sInverse[j] = new(privacy.Scalar).Mul(sInverse[j], xInverseList[i])
			} else {
				s[j] = new(privacy.Scalar).Mul(s[j], xInverseList[i])
				sInverse[j] = new(privacy.Scalar).Mul(sInverse[j], xList[i])
			}
		}
	}

	// Compute (g^s)^a (h^-s)^b u^(ab) = p l^(x^2) r^(-x^2)
	c := new(privacy.Scalar).Mul(proof.a, proof.b)
	rightHSPart1 := new(privacy.Point).MultiScalarMult(s, G)
	rightHSPart1.ScalarMult(rightHSPart1, proof.a)
	rightHSPart2 := new(privacy.Point).MultiScalarMult(sInverse, H)
	rightHSPart2.ScalarMult(rightHSPart2, proof.b)

	rightHS := new(privacy.Point).Add(rightHSPart1, rightHSPart2)
	rightHS.Add(rightHS, new(privacy.Point).ScalarMult(uParam, c))

	leftHSPart1 := new(privacy.Point).MultiScalarMult(xSquareList, proof.l)
	leftHSPart2 := new(privacy.Point).MultiScalarMult(xInverseSquare_List, proof.r)

	leftHS := new(privacy.Point).Add(leftHSPart1, leftHSPart2)
	leftHS.Add(leftHS, proof.p)

	res := privacy.IsPointEqual(rightHS, leftHS)
	if !res {
		Logger.Log.Error("Inner product argument failed:")
		Logger.Log.Error("LHS: %v\n", leftHS)
		Logger.Log.Error("RHS: %v\n", rightHS)
	}

	return res
}
