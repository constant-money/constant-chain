package aggregaterange

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge/aggregaterange/bulletproofs"
	"github.com/pkg/errors"
)

// This protocol proves in zero-knowledge that a list of committed values falls in [0, 2^64)

type AggregatedRangeWitness struct {
	values []uint64
	rands  []*privacy.Scalar
}

type AggregatedRangeProof struct {
	cmsValue          []*privacy.Point
	a                 *privacy.Point
	s                 *privacy.Point
	t1                *privacy.Point
	t2                *privacy.Point
	tauX              *privacy.Scalar
	tHat              *privacy.Scalar
	mu                *privacy.Scalar
	innerProductProof *InnerProductProof
}

func (proof AggregatedRangeProof) GetCmValues() []*privacy.Point {
	return proof.cmsValue
}

func (proof AggregatedRangeProof) ValidateSanity() bool {
	for i := 0; i < len(proof.cmsValue); i++ {
		if !proof.cmsValue[i].PointValid() {
			return false
		}
	}
	if !proof.a.PointValid() {
		return false
	}
	if !proof.s.PointValid() {
		return false
	}
	if !proof.t1.PointValid() {
		return false
	}
	if !proof.t2.PointValid() {
		return false
	}
	if !proof.tauX.ScalarValid() {
		return false
	}
	if !proof.tHat.ScalarValid() {
		return false
	}
	if !proof.mu.ScalarValid() {
		return false
	}

	return proof.innerProductProof.ValidateSanity()
}

func (proof *AggregatedRangeProof) Init() {
	proof.a = new(privacy.Point).Identity()
	proof.s = new(privacy.Point).Identity()
	proof.t1 = new(privacy.Point).Identity()
	proof.t2 = new(privacy.Point).Identity()
	proof.tauX = new(privacy.Scalar)
	proof.tHat = new(privacy.Scalar)
	proof.mu = new(privacy.Scalar)
	proof.innerProductProof = new(InnerProductProof)
}

func (proof AggregatedRangeProof) IsNil() bool {
	if proof.a == nil {
		return true
	}
	if proof.s == nil {
		return true
	}
	if proof.t1 == nil {
		return true
	}
	if proof.t2 == nil {
		return true
	}
	if proof.tauX == nil {
		return true
	}
	if proof.tHat == nil {
		return true
	}
	if proof.mu == nil {
		return true
	}
	return proof.innerProductProof == nil
}

func (proof AggregatedRangeProof) Bytes() []byte {
	var res []byte

	if proof.IsNil() {
		return []byte{}
	}

	res = append(res, byte(len(proof.cmsValue)))
	for i := 0; i < len(proof.cmsValue); i++ {
		res = append(res, proof.cmsValue[i].ToBytesS()...)
	}

	res = append(res, proof.a.ToBytesS()...)
	res = append(res, proof.s.ToBytesS()...)
	res = append(res, proof.t1.ToBytesS()...)
	res = append(res, proof.t2.ToBytesS()...)

	res = append(res, proof.tauX.ToBytesS()...)
	res = append(res, proof.tHat.ToBytesS()...)
	res = append(res, proof.mu.ToBytesS()...)
	res = append(res, proof.innerProductProof.Bytes()...)

	return res

}

func (proof *AggregatedRangeProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	lenValues := int(bytes[0])
	offset := 1
	var err error

	proof.cmsValue = make([]*privacy.Point, lenValues)
	for i := 0; i < lenValues; i++ {
		proof.cmsValue[i], err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset += privacy.Ed25519KeySize
	}

	proof.a, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacy.Ed25519KeySize

	proof.s, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacy.Ed25519KeySize

	proof.t1, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacy.Ed25519KeySize

	proof.t2, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacy.Ed25519KeySize

	proof.tauX = new(privacy.Scalar).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	offset += privacy.Ed25519KeySize

	proof.tHat = new(privacy.Scalar).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	offset += privacy.Ed25519KeySize

	proof.mu = new(privacy.Scalar).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	offset += privacy.Ed25519KeySize

	proof.innerProductProof = new(InnerProductProof)
	proof.innerProductProof.SetBytes(bytes[offset:])

	//privacy.Logger.Log.Debugf("AFTER SETBYTES ------------ %v\n", proof.Bytes())
	return nil
}

func (wit *AggregatedRangeWitness) Set(values []uint64, rands []*privacy.Scalar) {
	numValue := len(values)
	wit.values = make([]uint64, numValue)
	wit.rands = make([]*privacy.Scalar, numValue)

	for i := range values {
		wit.values[i] = values[i]
		wit.rands[i] = new(privacy.Scalar).Set(rands[i])
	}
}

func (wit AggregatedRangeWitness) ProveOld() (*AggregatedRangeProof, error) {
	proof := new(AggregatedRangeProof)

	numValue := len(wit.values)
	if numValue > maxOutputNumber {
		return nil, errors.New("Must less than maxOutputNumber")
	}
	numValuePad := pad(numValue)
	aggParam := new(bulletproofParams)
	aggParam.g = AggParam.g[0 : numValuePad*maxExp]
	aggParam.h = AggParam.h[0 : numValuePad*maxExp]
	aggParam.u = AggParam.u
	csByteH := []byte{}
	csByteG := []byte{}
	for i := 0; i < len(aggParam.g); i++ {
		csByteG = append(csByteG, aggParam.g[i].ToBytesS()...)
		csByteH = append(csByteH, aggParam.h[i].ToBytesS()...)
	}
	aggParam.cs = append(aggParam.cs, csByteG...)
	aggParam.cs = append(aggParam.cs, csByteH...)
	aggParam.cs = append(aggParam.cs, aggParam.u.ToBytesS()...)

	values := make([]uint64, numValuePad)
	rands := make([]*privacy.Scalar, numValuePad)

	for i := range wit.values {
		values[i] = wit.values[i]
		rands[i] = new(privacy.Scalar).Set(wit.rands[i])
	}

	for i := numValue; i < numValuePad; i++ {
		values[i] = uint64(0)
		rands[i] = new(privacy.Scalar).FromUint64(0)
	}

	proof.cmsValue = make([]*privacy.Point, numValue)
	for i := 0; i < numValue; i++ {
		proof.cmsValue[i] = privacy.PedCom.CommitAtIndex(new(privacy.Scalar).FromUint64(values[i]), rands[i], privacy.PedersenValueIndex)
	}

	n := maxExp
	// Convert values to binary array
	aL := make([]*privacy.Scalar, numValuePad*n)
	for i, value := range values {
		tmp := privacy.ConvertUint64ToBinary(value, n)
		for j := 0; j < n; j++ {
			aL[i*n+j] = tmp[j]
		}
	}

	twoNumber := new(privacy.Scalar).FromUint64(2)
	twoVectorN := powerVector(twoNumber, n)

	aR := make([]*privacy.Scalar, numValuePad*n)

	for i := 0; i < numValuePad*n; i++ {
		aR[i] = new(privacy.Scalar).Sub(aL[i], new(privacy.Scalar).FromUint64(1))
	}

	// random alpha
	alpha := privacy.RandomScalar()

	// Commitment to aL, aR: A = h^alpha * G^aL * H^aR
	A, err := encodeVectors(aL, aR, aggParam.g, aggParam.h)
	if err != nil {
		return nil, err
	}
	A.Add(A, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], alpha))
	proof.a = A

	// Random blinding vectors sL, sR
	sL := make([]*privacy.Scalar, n*numValuePad)
	sR := make([]*privacy.Scalar, n*numValuePad)
	for i := range sL {
		sL[i] = privacy.RandomScalar()
		sR[i] = privacy.RandomScalar()
	}

	// random rho
	rho := privacy.RandomScalar()

	// Commitment to sL, sR : S = h^rho * G^sL * H^sR
	S, err := encodeVectors(sL, sR, aggParam.g, aggParam.h)
	if err != nil {
		return nil, err
	}
	S.Add(S, new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenRandomnessIndex], rho))
	proof.s = S

	// challenge y, z
	y := generateChallenge([][]byte{aggParam.cs, A.ToBytesS(), S.ToBytesS()})
	z := generateChallenge([][]byte{aggParam.cs, A.ToBytesS(), S.ToBytesS(), y.ToBytesS()})

	zNeg := new(privacy.Scalar).Sub(new(privacy.Scalar).FromUint64(0), z)
	zSquare := new(privacy.Scalar).Mul(z, z)

	// l(X) = (aL -z*1^n) + sL*X
	yVector := powerVector(y, n*numValuePad)

	l0 := vectorAddScalar(aL, zNeg)
	l1 := sL

	// r(X) = y^n hada (aR +z*1^n + sR*X) + z^2 * 2^n
	hadaProduct, err := hadamardProduct(yVector, vectorAddScalar(aR, z))
	if err != nil {
		return nil, err
	}

	vectorSum := make([]*privacy.Scalar, n*numValuePad)
	zTmp := new(privacy.Scalar).Set(z)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		for i := 0; i < n; i++ {
			vectorSum[j*n+i] = new(privacy.Scalar).Mul(twoVectorN[i], zTmp)
		}
	}

	r0, err := vectorAdd(hadaProduct, vectorSum)
	if err != nil {
		return nil, err
	}

	r1, err := hadamardProduct(yVector, sR)
	if err != nil {
		return nil, err
	}

	//t(X) = <l(X), r(X)> = t0 + t1*X + t2*X^2

	//calculate t0 = v*z^2 + delta(y, z)
	deltaYZ := new(privacy.Scalar).Sub(z, zSquare)

	// innerProduct1 = <1^(n*m), y^(n*m)>
	innerProduct1 := new(privacy.Scalar).FromUint64(0)
	for i := 0; i < n*numValuePad; i++ {
		innerProduct1.Add(innerProduct1, yVector[i])
	}

	deltaYZ.Mul(deltaYZ, innerProduct1)

	// innerProduct2 = <1^n, 2^n>
	innerProduct2 := new(privacy.Scalar).FromUint64(0)
	for i := 0; i < n; i++ {
		innerProduct2.Add(innerProduct2, twoVectorN[i])
	}

	sum := new(privacy.Scalar).FromUint64(0)
	zTmp = new(privacy.Scalar).Set(zSquare)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		sum.Add(sum, zTmp)
	}
	sum.Mul(sum, innerProduct2)
	deltaYZ.Sub(deltaYZ, sum)

	// t1 = <l1, r0> + <l0, r1>
	innerProduct3, err := innerProduct(l1, r0)
	if err != nil {
		return nil, err
	}

	innerProduct4, err := innerProduct(l0, r1)
	if err != nil {
		return nil, err
	}

	t1 := new(privacy.Scalar).Add(innerProduct3, innerProduct4)

	// t2 = <l1, r1>
	t2, err := innerProduct(l1, r1)
	if err != nil {
		return nil, err
	}

	// commitment to t1, t2
	tau1 := privacy.RandomScalar()
	tau2 := privacy.RandomScalar()

	proof.t1 = privacy.PedCom.CommitAtIndex(t1, tau1, privacy.PedersenValueIndex)
	proof.t2 = privacy.PedCom.CommitAtIndex(t2, tau2, privacy.PedersenValueIndex)

	// challenge x = hash(G || H || A || S || T1 || T2)
	x := generateChallenge([][]byte{aggParam.cs, proof.a.ToBytesS(), proof.s.ToBytesS(), proof.t1.ToBytesS(), proof.t2.ToBytesS()})

	xSquare := new(privacy.Scalar).Mul(x, x)

	// lVector = aL - z*1^n + sL*x
	lVector, err := vectorAdd(vectorAddScalar(aL, zNeg), vectorMulScalar(sL, x))
	if err != nil {
		return nil, err
	}

	// rVector = y^n hada (aR +z*1^n + sR*x) + z^2*2^n
	tmpVector, err := vectorAdd(vectorAddScalar(aR, z), vectorMulScalar(sR, x))
	if err != nil {
		return nil, err
	}
	rVector, err := hadamardProduct(yVector, tmpVector)
	if err != nil {
		return nil, err
	}

	vectorSum = make([]*privacy.Scalar, n*numValuePad)
	zTmp = new(privacy.Scalar).Set(z)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		for i := 0; i < n; i++ {
			vectorSum[j*n+i] = new(privacy.Scalar).Mul(twoVectorN[i], zTmp)
		}
	}

	rVector, err = vectorAdd(rVector, vectorSum)
	if err != nil {
		return nil, err
	}

	// tHat = <lVector, rVector>
	proof.tHat, err = innerProduct(lVector, rVector)
	if err != nil {
		return nil, err
	}

	// blinding value for tHat: tauX = tau2*x^2 + tau1*x + z^2*rand
	proof.tauX = new(privacy.Scalar).Mul(tau2, xSquare)
	proof.tauX.Add(proof.tauX, new(privacy.Scalar).Mul(tau1, x))
	zTmp = new(privacy.Scalar).Set(z)
	tmpBN := new(privacy.Scalar)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		proof.tauX.Add(proof.tauX, tmpBN.Mul(zTmp, rands[j]))
	}

	// alpha, rho blind A, S
	// mu = alpha + rho*x
	proof.mu = new(privacy.Scalar).Mul(rho, x)
	proof.mu.Add(proof.mu, alpha)

	// instead of sending left vector and right vector, we use inner sum argument to reduce proof size from 2*n to 2(log2(n)) + 2
	innerProductWit := new(InnerProductWitness)
	innerProductWit.a = lVector
	innerProductWit.b = rVector
	innerProductWit.p, err = encodeVectors(lVector, rVector, aggParam.g, aggParam.h)
	if err != nil {
		return nil, err
	}
	innerProductWit.p = innerProductWit.p.Add(innerProductWit.p, new(privacy.Point).ScalarMult(aggParam.u, proof.tHat))

	proof.innerProductProof, err = innerProductWit.Prove(aggParam)
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func (wit AggregatedRangeWitness) Prove() (*AggregatedRangeProof, error) {
	wit2 := new(bulletproofs.AggregatedRangeWitness)
	wit2.Set(wit.values, wit.rands)

	proof2, err := wit2.Prove()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("cannot prove bulletproof v2. Error %v", err))
	}
	proof2Bytes := proof2.Bytes()
	proof := new(AggregatedRangeProof)
	err = proof.SetBytes(proof2Bytes)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, errors.New(fmt.Sprintf("cannot convert proof ver 2  to ver 1. Error %v", err))
	}
	return proof, nil
}

func (proof AggregatedRangeProof) VerifyOld() (bool, error) {
	numValue := len(proof.cmsValue)
	if numValue > maxOutputNumber {
		return false, errors.New("Must less than maxOutputNumber")
	}
	numValuePad := pad(numValue)
	aggParam := new(bulletproofParams)
	aggParam.g = AggParam.g[0 : numValuePad*maxExp]
	aggParam.h = AggParam.h[0 : numValuePad*maxExp]
	aggParam.u = AggParam.u
	csByteH := []byte{}
	csByteG := []byte{}
	for i := 0; i < len(aggParam.g); i++ {
		csByteG = append(csByteG, aggParam.g[i].ToBytesS()...)
		csByteH = append(csByteH, aggParam.h[i].ToBytesS()...)
	}
	aggParam.cs = append(aggParam.cs, csByteG...)
	aggParam.cs = append(aggParam.cs, csByteH...)
	aggParam.cs = append(aggParam.cs, aggParam.u.ToBytesS()...)

	tmpcmsValue := proof.cmsValue

	for i := numValue; i < numValuePad; i++ {
		identity := new(privacy.Point).Identity()
		tmpcmsValue = append(tmpcmsValue, identity)
	}

	n := maxExp
	oneNumber := new(privacy.Scalar).FromUint64(1)
	twoNumber := new(privacy.Scalar).FromUint64(2)
	oneVector := powerVector(oneNumber, n*numValuePad)
	oneVectorN := powerVector(oneNumber, n)
	twoVectorN := powerVector(twoNumber, n)

	// recalculate challenge y, z
	y := generateChallenge([][]byte{aggParam.cs, proof.a.ToBytesS(), proof.s.ToBytesS()})
	z := generateChallenge([][]byte{aggParam.cs, proof.a.ToBytesS(), proof.s.ToBytesS(), y.ToBytesS()})
	zSquare := new(privacy.Scalar).Mul(z, z)

	// challenge x = hash(G || H || A || S || T1 || T2)
	//fmt.Printf("T2: %v\n", proof.t2)
	x := generateChallenge([][]byte{aggParam.cs, proof.a.ToBytesS(), proof.s.ToBytesS(), proof.t1.ToBytesS(), proof.t2.ToBytesS()})
	xSquare := new(privacy.Scalar).Mul(x, x)

	yVector := powerVector(y, n*numValuePad)
	// HPrime = H^(y^(1-i)
	HPrime := make([]*privacy.Point, n*numValuePad)
	yInverse := new(privacy.Scalar).Invert(y)
	expyInverse := new(privacy.Scalar).FromUint64(1)
	for i := 0; i < n*numValuePad; i++ {
		HPrime[i] = new(privacy.Point).ScalarMult(aggParam.h[i], expyInverse)
		expyInverse.Mul(expyInverse, yInverse)
	}

	// g^tHat * h^tauX = V^(z^2) * g^delta(y,z) * T1^x * T2^(x^2)
	deltaYZ := new(privacy.Scalar).Sub(z, zSquare)

	// innerProduct1 = <1^(n*m), y^(n*m)>
	innerProduct1, err := innerProduct(oneVector, yVector)
	if err != nil {
		return false, privacy.NewPrivacyErr(privacy.CalInnerProductErr, err)
	}

	deltaYZ.Mul(deltaYZ, innerProduct1)

	// innerProduct2 = <1^n, 2^n>
	innerProduct2, err := innerProduct(oneVectorN, twoVectorN)
	if err != nil {
		return false, privacy.NewPrivacyErr(privacy.CalInnerProductErr, err)
	}

	sum := new(privacy.Scalar).FromUint64(0)
	zTmp := new(privacy.Scalar).Set(zSquare)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		sum.Add(sum, zTmp)
	}
	sum.Mul(sum, innerProduct2)
	deltaYZ.Sub(deltaYZ, sum)

	left1 := privacy.PedCom.CommitAtIndex(proof.tHat, proof.tauX, privacy.PedersenValueIndex)

	right1 := new(privacy.Point).ScalarMult(proof.t2, xSquare)
	right1.Add(right1, new(privacy.Point).AddPedersen(deltaYZ, privacy.PedCom.G[privacy.PedersenValueIndex], x, proof.t1))

	expVector := vectorMulScalar(powerVector(z, numValuePad), zSquare)
	right1.Add(right1, new(privacy.Point).MultiScalarMult(expVector, tmpcmsValue))

	if !privacy.IsPointEqual(left1, right1) {
		privacy.Logger.Log.Errorf("verify aggregated range proof statement 1 failed")
		return false, errors.New("verify aggregated range proof statement 1 failed")
	}

	innerProductArgValid := proof.innerProductProof.Verify(aggParam)
	if !innerProductArgValid {
		privacy.Logger.Log.Errorf("verify aggregated range proof statement 2 failed")
		return false, errors.New("verify aggregated range proof statement 2 failed")
	}

	return true, nil
}

func (proof AggregatedRangeProof) Verify() (bool, error) {
	proof2 := new(bulletproofs.AggregatedRangeProof)
	err := proof2.SetBytes(proof.Bytes())
	if err != nil {
		return false, errors.New(fmt.Sprintf("cannot convert proof from v1 to v2. Error %v", err))
	}
	return proof2.VerifyFaster()
}

func VerifyBatchOld(proofs []*AggregatedRangeProof) (bool, error, int) {
	innerProductProofs := make([]*InnerProductProof, 0)
	csList := make([][]byte, 0)
	for k, proof := range proofs {
		numValue := len(proof.cmsValue)
		if numValue > maxOutputNumber {
			return false, errors.New("Must less than maxOutputNumber"), k
		}
		numValuePad := pad(numValue)
		aggParam := new(bulletproofParams)
		aggParam.g = AggParam.g[0 : numValuePad*maxExp]
		aggParam.h = AggParam.h[0 : numValuePad*maxExp]
		aggParam.u = AggParam.u
		csByteH := []byte{}
		csByteG := []byte{}
		for i := 0; i < len(aggParam.g); i++ {
			csByteG = append(csByteG, aggParam.g[i].ToBytesS()...)
			csByteH = append(csByteH, aggParam.h[i].ToBytesS()...)
		}
		aggParam.cs = append(aggParam.cs, csByteG...)
		aggParam.cs = append(aggParam.cs, csByteH...)
		aggParam.cs = append(aggParam.cs, aggParam.u.ToBytesS()...)

		tmpcmsValue := proof.cmsValue

		for i := numValue; i < numValuePad; i++ {
			identity := new(privacy.Point).Identity()
			tmpcmsValue = append(tmpcmsValue, identity)
		}

		n := maxExp
		oneNumber := new(privacy.Scalar).FromUint64(1)
		twoNumber := new(privacy.Scalar).FromUint64(2)
		oneVector := powerVector(oneNumber, n*numValuePad)
		oneVectorN := powerVector(oneNumber, n)
		twoVectorN := powerVector(twoNumber, n)

		// recalculate challenge y, z
		y := generateChallenge([][]byte{aggParam.cs, proof.a.ToBytesS(), proof.s.ToBytesS()})
		z := generateChallenge([][]byte{aggParam.cs, proof.a.ToBytesS(), proof.s.ToBytesS(), y.ToBytesS()})
		zSquare := new(privacy.Scalar).Mul(z, z)

		// challenge x = hash(G || H || A || S || T1 || T2)
		//fmt.Printf("T2: %v\n", proof.t2)
		x := generateChallenge([][]byte{aggParam.cs, proof.a.ToBytesS(), proof.s.ToBytesS(), proof.t1.ToBytesS(), proof.t2.ToBytesS()})
		xSquare := new(privacy.Scalar).Mul(x, x)

		yVector := powerVector(y, n*numValuePad)
		// HPrime = H^(y^(1-i)
		HPrime := make([]*privacy.Point, n*numValuePad)
		yInverse := new(privacy.Scalar).Invert(y)
		expyInverse := new(privacy.Scalar).FromUint64(1)
		for i := 0; i < n*numValuePad; i++ {
			HPrime[i] = new(privacy.Point).ScalarMult(aggParam.h[i], expyInverse)
			expyInverse.Mul(expyInverse, yInverse)
		}

		// g^tHat * h^tauX = V^(z^2) * g^delta(y,z) * T1^x * T2^(x^2)
		deltaYZ := new(privacy.Scalar).Sub(z, zSquare)

		// innerProduct1 = <1^(n*m), y^(n*m)>
		innerProduct1, err := innerProduct(oneVector, yVector)
		if err != nil {
			return false, privacy.NewPrivacyErr(privacy.CalInnerProductErr, err), k
		}

		deltaYZ.Mul(deltaYZ, innerProduct1)

		// innerProduct2 = <1^n, 2^n>
		innerProduct2, err := innerProduct(oneVectorN, twoVectorN)
		if err != nil {
			return false, privacy.NewPrivacyErr(privacy.CalInnerProductErr, err), k
		}

		sum := new(privacy.Scalar).FromUint64(0)
		zTmp := new(privacy.Scalar).Set(zSquare)
		for j := 0; j < numValuePad; j++ {
			zTmp.Mul(zTmp, z)
			sum.Add(sum, zTmp)
		}
		sum.Mul(sum, innerProduct2)
		deltaYZ.Sub(deltaYZ, sum)

		left1 := privacy.PedCom.CommitAtIndex(proof.tHat, proof.tauX, privacy.PedersenValueIndex)

		right1 := new(privacy.Point).ScalarMult(proof.t2, xSquare)
		right1.Add(right1, new(privacy.Point).AddPedersen(deltaYZ, privacy.PedCom.G[privacy.PedersenValueIndex], x, proof.t1))

		expVector := vectorMulScalar(powerVector(z, numValuePad), zSquare)
		right1.Add(right1, new(privacy.Point).MultiScalarMult(expVector, tmpcmsValue))

		if !privacy.IsPointEqual(left1, right1) {
			privacy.Logger.Log.Errorf("verify aggregated range proof statement 1 failed index %d", k)
			return false, fmt.Errorf("verify aggregated range proof statement 1 failed index %d", k), k
		}

		innerProductProofs = append(innerProductProofs, proof.innerProductProof)
		csList = append(csList, aggParam.cs)
	}

	innerProductArgsValid := VerifyBatchingInnerProductProofs(innerProductProofs, csList)
	if !innerProductArgsValid {
		privacy.Logger.Log.Errorf("verify batch aggregated range proofs statement 2 failed")
		return false, errors.New("verify batch aggregated range proofs statement 2 failed"), -1
	}

	return true, nil, -1
}

func VerifyBatch(proofs []*AggregatedRangeProof) (bool, error, int) {
	proofs2 := make([]*bulletproofs.AggregatedRangeProof, len(proofs))
	for i, proof := range proofs {
		proofs2[i] = new(bulletproofs.AggregatedRangeProof)
		err := proofs2[i].SetBytes(proof.Bytes())
		if err != nil {
			return false, errors.New(fmt.Sprintf("cannot convert proof from v1 to v2. Error %v", err)), i
		}
	}
	return bulletproofs.VerifyBatch(proofs2)
}
