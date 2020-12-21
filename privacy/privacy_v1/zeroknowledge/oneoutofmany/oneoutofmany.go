package oneoutofmany

import (
	"math/big"

	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_util"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/oneoutofmany/polynomial"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/utils"
	"github.com/pkg/errors"
)

// This protocol proves in zero-knowledge that one-out-of-N commitments contains 0

// Statement to be proved
type OneOutOfManyStatement struct {
	Commitments []*operation.Point
}

// Statement's witness
type OneOutOfManyWitness struct {
	stmt        *OneOutOfManyStatement
	rand        *operation.Scalar
	indexIsZero uint64
}

// Statement's proof
type OneOutOfManyProof struct {
	Statement      *OneOutOfManyStatement
	cl, ca, cb, cd []*operation.Point
	f, za, zb      []*operation.Scalar
	zd             *operation.Scalar
}

func (proof OneOutOfManyProof) ValidateSanity() bool {
	if len(proof.cl) != privacy_util.CommitmentRingSizeExp || len(proof.ca) != privacy_util.CommitmentRingSizeExp ||
		len(proof.cb) != privacy_util.CommitmentRingSizeExp || len(proof.cd) != privacy_util.CommitmentRingSizeExp ||
		len(proof.f) != privacy_util.CommitmentRingSizeExp || len(proof.za) != privacy_util.CommitmentRingSizeExp ||
		len(proof.zb) != privacy_util.CommitmentRingSizeExp {
		return false
	}

	for i := 0; i < len(proof.cl); i++ {
		if !proof.cl[i].PointValid() {
			return false
		}
		if !proof.ca[i].PointValid() {
			return false
		}
		if !proof.cb[i].PointValid() {
			return false
		}
		if !proof.cd[i].PointValid() {
			return false
		}

		if !proof.f[i].ScalarValid() {
			return false
		}
		if !proof.za[i].ScalarValid() {
			return false
		}
		if !proof.zb[i].ScalarValid() {
			return false
		}
	}

	return proof.zd.ScalarValid()
}

func (proof OneOutOfManyProof) isNil() bool {
	if proof.cl == nil {
		return true
	}
	if proof.ca == nil {
		return true
	}
	if proof.cb == nil {
		return true
	}
	if proof.cd == nil {
		return true
	}
	if proof.f == nil {
		return true
	}
	if proof.za == nil {
		return true
	}
	if proof.zb == nil {
		return true
	}
	return proof.zd == nil
}

func (proof *OneOutOfManyProof) Init() *OneOutOfManyProof {
	proof.zd = new(operation.Scalar)
	proof.Statement = new(OneOutOfManyStatement)

	return proof
}

// Set sets Statement
func (stmt *OneOutOfManyStatement) Set(commitments []*operation.Point) {
	stmt.Commitments = commitments
}

// Set sets Witness
func (wit *OneOutOfManyWitness) Set(commitments []*operation.Point, rand *operation.Scalar, indexIsZero uint64) {
	wit.stmt = new(OneOutOfManyStatement)
	wit.stmt.Set(commitments)

	wit.indexIsZero = indexIsZero
	wit.rand = rand
}

// Set sets Proof
func (proof *OneOutOfManyProof) Set(
	commitments []*operation.Point,
	cl, ca, cb, cd []*operation.Point,
	f, za, zb []*operation.Scalar,
	zd *operation.Scalar) {

	proof.Statement = new(OneOutOfManyStatement)
	proof.Statement.Set(commitments)

	proof.cl, proof.ca, proof.cb, proof.cd = cl, ca, cb, cd
	proof.f, proof.za, proof.zb = f, za, zb
	proof.zd = zd
}

// Bytes converts one of many proof to bytes array
func (proof OneOutOfManyProof) Bytes() []byte {
	// if proof is nil, return an empty array
	if proof.isNil() {
		return []byte{}
	}

	// N = 2^n
	n := privacy_util.CommitmentRingSizeExp

	var bytes []byte

	// convert array cl to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, proof.cl[i].ToBytesS()...)
	}
	// convert array ca to bytes array
	for i := 0; i < n; i++ {
		//fmt.Printf("proof.ca[i]: %v\n", proof.ca[i])
		//fmt.Printf("proof.ca[i]: %v\n", proof.ca[i].Compress())
		bytes = append(bytes, proof.ca[i].ToBytesS()...)
	}

	// convert array cb to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, proof.cb[i].ToBytesS()...)
	}

	// convert array cd to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, proof.cd[i].ToBytesS()...)
	}

	// convert array f to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, proof.f[i].ToBytesS()...)
	}

	// convert array za to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, proof.za[i].ToBytesS()...)
	}

	// convert array zb to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, proof.zb[i].ToBytesS()...)
	}

	// convert array zd to bytes array
	bytes = append(bytes, proof.zd.ToBytesS()...)

	return bytes
}

// SetBytes converts an array of bytes to an object of OneOutOfManyProof
func (proof *OneOutOfManyProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	n := privacy_util.CommitmentRingSizeExp

	offset := 0
	var err error

	// get cl array
	proof.cl = make([]*operation.Point, n)
	for i := 0; i < n; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("One-out-of-many Proof byte unmarshaling failed")
		}
		proof.cl[i], err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset = offset + operation.Ed25519KeySize
	}

	// get ca array
	proof.ca = make([]*operation.Point, n)
	for i := 0; i < n; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("One-out-of-many Proof byte unmarshaling failed")
		}
		proof.ca[i], err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset = offset + operation.Ed25519KeySize
	}

	// get cb array
	proof.cb = make([]*operation.Point, n)
	for i := 0; i < n; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("One-out-of-many Proof byte unmarshaling failed")
		}
		proof.cb[i], err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset = offset + operation.Ed25519KeySize
	}

	// get cd array
	proof.cd = make([]*operation.Point, n)
	for i := 0; i < n; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("One-out-of-many Proof byte unmarshaling failed")
		}
		proof.cd[i], err = new(operation.Point).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset = offset + operation.Ed25519KeySize
	}

	// get f array
	proof.f = make([]*operation.Scalar, n)
	for i := 0; i < n; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("One-out-of-many Proof byte unmarshaling failed")
		}
		proof.f[i] = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		offset = offset + operation.Ed25519KeySize
	}

	// get za array
	proof.za = make([]*operation.Scalar, n)
	for i := 0; i < n; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("One-out-of-many Proof byte unmarshaling failed")
		}
		proof.za[i] = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		offset = offset + operation.Ed25519KeySize
	}

	// get zb array
	proof.zb = make([]*operation.Scalar, n)
	for i := 0; i < n; i++ {
		if offset+operation.Ed25519KeySize > len(bytes){
			return errors.New("One-out-of-many Proof byte unmarshaling failed")
		}
		proof.zb[i] = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])
		offset = offset + operation.Ed25519KeySize
	}

	// get zd
	if offset+operation.Ed25519KeySize > len(bytes){
		return errors.New("One-out-of-many Proof byte unmarshaling failed")
	}
	proof.zd = new(operation.Scalar).FromBytesS(bytes[offset : offset+operation.Ed25519KeySize])

	return nil
}

// Prove produces a proof for the statement
func (wit OneOutOfManyWitness) Prove() (*OneOutOfManyProof, error) {
	// Check the number of Commitment list's elements
	N := len(wit.stmt.Commitments)
	if N != privacy_util.CommitmentRingSize {
		return nil, errors.New("the number of Commitment list's elements must be equal to CMRingSize")
	}

	n := privacy_util.CommitmentRingSizeExp

	// Check indexIsZero
	if wit.indexIsZero > uint64(N) {
		return nil, errors.New("Index is zero must be Index in list of commitments")
	}

	// represent indexIsZero in binary
	indexIsZeroBinary := privacy_util.ConvertIntToBinary(int(wit.indexIsZero), n)

	//
	r := make([]*operation.Scalar, n)
	a := make([]*operation.Scalar, n)
	s := make([]*operation.Scalar, n)
	t := make([]*operation.Scalar, n)
	u := make([]*operation.Scalar, n)

	cl := make([]*operation.Point, n)
	ca := make([]*operation.Point, n)
	cb := make([]*operation.Point, n)
	cd := make([]*operation.Point, n)

	for j := 0; j < n; j++ {
		// Generate random numbers
		r[j] = operation.RandomScalar()
		a[j] = operation.RandomScalar()
		s[j] = operation.RandomScalar()
		t[j] = operation.RandomScalar()
		u[j] = operation.RandomScalar()

		// convert indexIsZeroBinary[j] to operation.Scalar
		indexInt := new(operation.Scalar).FromUint64(uint64(indexIsZeroBinary[j]))

		// Calculate cl, ca, cb, cd
		// cl = Com(l, r)
		cl[j] = operation.PedCom.CommitAtIndex(indexInt, r[j], operation.PedersenPrivateKeyIndex)

		// ca = Com(a, s)
		ca[j] = operation.PedCom.CommitAtIndex(a[j], s[j], operation.PedersenPrivateKeyIndex)

		// cb = Com(la, t)
		la := new(operation.Scalar).Mul(indexInt, a[j])
		//la.Mod(la, operation.Curve.Params().N)
		cb[j] = operation.PedCom.CommitAtIndex(la, t[j], operation.PedersenPrivateKeyIndex)
	}

	// Calculate: cd_k = ci^pi,k
	for k := 0; k < n; k++ {
		// Calculate pi,k which is coefficient of x^k in polynomial pi(x)
		cd[k] = new(operation.Point).Identity()

		for i := 0; i < N; i++ {
			iBinary := privacy_util.ConvertIntToBinary(i, n)
			pik := getCoefficient(iBinary, k, n, a, indexIsZeroBinary)
			cd[k].Add(cd[k], new(operation.Point).ScalarMult(wit.stmt.Commitments[i], pik))
		}

		cd[k].Add(cd[k], operation.PedCom.CommitAtIndex(new(operation.Scalar).FromUint64(0), u[k], operation.PedersenPrivateKeyIndex))
	}

	// Calculate x
	cmtsInBytes := make([][]byte, 0)
	for _, cmts := range wit.stmt.Commitments{
		cmtsInBytes = append(cmtsInBytes, cmts.ToBytesS())
	}
	x := utils.GenerateChallenge(cmtsInBytes)
	for j := 0; j < n; j++ {
		x = utils.GenerateChallenge([][]byte{
			x.ToBytesS(),
			cl[j].ToBytesS(),
			ca[j].ToBytesS(),
			cb[j].ToBytesS(),
			cd[j].ToBytesS(),
		})
	}

	// Calculate za, zb zd
	za := make([]*operation.Scalar, n)
	zb := make([]*operation.Scalar, n)

	f := make([]*operation.Scalar, n)

	for j := 0; j < n; j++ {
		// f = lx + a
		f[j] = new(operation.Scalar).Mul(new(operation.Scalar).FromUint64(uint64(indexIsZeroBinary[j])), x)
		f[j].Add(f[j], a[j])

		// za = s + rx
		za[j] = new(operation.Scalar).Mul(r[j], x)
		za[j].Add(za[j], s[j])

		// zb = r(x - f) + t
		zb[j] = new(operation.Scalar).Sub(x, f[j])
		zb[j].Mul(zb[j], r[j])
		zb[j].Add(zb[j], t[j])
	}

	// zd = rand * x^n - sum_{k=0}^{n-1} u[k] * x^k
	xi := new(operation.Scalar).FromUint64(1)
	sum := new(operation.Scalar).FromUint64(0)
	for k := 0; k < n; k++ {
		tmp := new(operation.Scalar).Mul(xi, u[k])
		sum.Add(sum, tmp)
		xi.Mul(xi, x)
	}
	zd := new(operation.Scalar).Mul(xi, wit.rand)
	zd.Sub(zd, sum)

	proof := new(OneOutOfManyProof).Init()
	proof.Set(wit.stmt.Commitments, cl, ca, cb, cd, f, za, zb, zd)

	return proof, nil
}

// Verify verifies a proof output by Prove
func (proof OneOutOfManyProof) Verify() (bool, error) {
	N := len(proof.Statement.Commitments)

	// the number of Commitment list's elements must be equal to CMRingSize
	if N != privacy_util.CommitmentRingSize {
		return false, errors.New("Invalid length of commitments list in one out of many proof")
	}
	n := privacy_util.CommitmentRingSizeExp

	//Calculate x
	cmtsInBytes := make([][]byte, 0)
	for _, cmts := range proof.Statement.Commitments{
		cmtsInBytes = append(cmtsInBytes, cmts.ToBytesS())
	}
	x := utils.GenerateChallenge(cmtsInBytes)
	for j := 0; j < n; j++ {
		x = utils.GenerateChallenge([][]byte{x.ToBytesS(), proof.cl[j].ToBytesS(), proof.ca[j].ToBytesS(), proof.cb[j].ToBytesS(), proof.cd[j].ToBytesS()})
	}

	for i := 0; i < n; i++ {
		//Check cl^x * ca = Com(f, za)
		leftPoint1 := new(operation.Point).ScalarMult(proof.cl[i], x)
		leftPoint1.Add(leftPoint1, proof.ca[i])

		rightPoint1 := operation.PedCom.CommitAtIndex(proof.f[i], proof.za[i], operation.PedersenPrivateKeyIndex)

		if !operation.IsPointEqual(leftPoint1, rightPoint1) {
			Logger.Log.Errorf("verify one out of many proof statement 1 failed")
			return false, errors.New("verify one out of many proof statement 1 failed")
		}

		//Check cl^(x-f) * cb = Com(0, zb)
		xSubF := new(operation.Scalar).Sub(x, proof.f[i])

		leftPoint2 := new(operation.Point).ScalarMult(proof.cl[i], xSubF)
		leftPoint2.Add(leftPoint2, proof.cb[i])
		rightPoint2 := operation.PedCom.CommitAtIndex(new(operation.Scalar).FromUint64(0), proof.zb[i], operation.PedersenPrivateKeyIndex)

		if !operation.IsPointEqual(leftPoint2, rightPoint2) {
			Logger.Log.Errorf("verify one out of many proof statement 2 failed")
			return false, errors.New("verify one out of many proof statement 2 failed")
		}
	}

	leftPoint3 := new(operation.Point).Identity()
	leftPoint32 := new(operation.Point).Identity()

	for i := 0; i < N; i++ {
		iBinary := privacy_util.ConvertIntToBinary(i, n)

		exp := new(operation.Scalar).FromUint64(1)
		fji := new(operation.Scalar).FromUint64(1)
		for j := 0; j < n; j++ {
			if iBinary[j] == 1 {
				fji.Set(proof.f[j])
			} else {
				fji.Sub(x, proof.f[j])
			}

			exp.Mul(exp, fji)
		}

		leftPoint3.Add(leftPoint3, new(operation.Point).ScalarMult(proof.Statement.Commitments[i], exp))
	}

	tmp2 := new(operation.Scalar).FromUint64(1)
	for k := 0; k < n; k++ {
		xk := new(operation.Scalar).Sub(new(operation.Scalar).FromUint64(0), tmp2)
		leftPoint32.Add(leftPoint32, new(operation.Point).ScalarMult(proof.cd[k], xk))
		tmp2.Mul(tmp2, x)
	}

	leftPoint3.Add(leftPoint3, leftPoint32)

	rightPoint3 := operation.PedCom.CommitAtIndex(new(operation.Scalar).FromUint64(0), proof.zd, operation.PedersenPrivateKeyIndex)

	if !operation.IsPointEqual(leftPoint3, rightPoint3) {
		Logger.Log.Errorf("verify one out of many proof statement 3 failed")
		return false, errors.New("verify one out of many proof statement 3 failed")
	}

	return true, nil
}

// Verify verifies a proof output by Prove
func (proof OneOutOfManyProof) VerifyOld() (bool, error) {
	N := len(proof.Statement.Commitments)

	// the number of Commitment list's elements must be equal to CMRingSize
	if N != privacy_util.CommitmentRingSize {
		return false, errors.New("Invalid length of commitments list in one out of many proof")
	}
	n := privacy_util.CommitmentRingSizeExp

	//Calculate x
	x := new(operation.Scalar).FromUint64(0)

	for j := 0; j < n; j++ {
		x = utils.GenerateChallenge([][]byte{x.ToBytesS(), proof.cl[j].ToBytesS(), proof.ca[j].ToBytesS(), proof.cb[j].ToBytesS(), proof.cd[j].ToBytesS()})
	}

	for i := 0; i < n; i++ {
		//Check cl^x * ca = Com(f, za)
		leftPoint1 := new(operation.Point).ScalarMult(proof.cl[i], x)
		leftPoint1.Add(leftPoint1, proof.ca[i])

		rightPoint1 := operation.PedCom.CommitAtIndex(proof.f[i], proof.za[i], operation.PedersenPrivateKeyIndex)

		if !operation.IsPointEqual(leftPoint1, rightPoint1) {
			Logger.Log.Errorf("verifyOld one out of many proof statement 1 failed")
			return false, errors.New("verifyOld one out of many proof statement 1 failed")
		}

		//Check cl^(x-f) * cb = Com(0, zb)
		xSubF := new(operation.Scalar).Sub(x, proof.f[i])

		leftPoint2 := new(operation.Point).ScalarMult(proof.cl[i], xSubF)
		leftPoint2.Add(leftPoint2, proof.cb[i])
		rightPoint2 := operation.PedCom.CommitAtIndex(new(operation.Scalar).FromUint64(0), proof.zb[i], operation.PedersenPrivateKeyIndex)

		if !operation.IsPointEqual(leftPoint2, rightPoint2) {
			Logger.Log.Errorf("verifyOld one out of many proof statement 2 failed")
			return false, errors.New("verifyOld one out of many proof statement 2 failed")
		}
	}

	leftPoint3 := new(operation.Point).Identity()
	leftPoint32 := new(operation.Point).Identity()

	for i := 0; i < N; i++ {
		iBinary := privacy_util.ConvertIntToBinary(i, n)

		exp := new(operation.Scalar).FromUint64(1)
		fji := new(operation.Scalar).FromUint64(1)
		for j := 0; j < n; j++ {
			if iBinary[j] == 1 {
				fji.Set(proof.f[j])
			} else {
				fji.Sub(x, proof.f[j])
			}

			exp.Mul(exp, fji)
		}

		leftPoint3.Add(leftPoint3, new(operation.Point).ScalarMult(proof.Statement.Commitments[i], exp))
	}

	tmp2 := new(operation.Scalar).FromUint64(1)
	for k := 0; k < n; k++ {
		xk := new(operation.Scalar).Sub(new(operation.Scalar).FromUint64(0), tmp2)
		leftPoint32.Add(leftPoint32, new(operation.Point).ScalarMult(proof.cd[k], xk))
		tmp2.Mul(tmp2, x)
	}

	leftPoint3.Add(leftPoint3, leftPoint32)

	rightPoint3 := operation.PedCom.CommitAtIndex(new(operation.Scalar).FromUint64(0), proof.zd, operation.PedersenPrivateKeyIndex)

	if !operation.IsPointEqual(leftPoint3, rightPoint3) {
		Logger.Log.Errorf("verifyOld one out of many proof statement 3 failed")
		return false, errors.New("verifyOld one out of many proof statement 3 failed")
	}

	return true, nil
}

// Get coefficient of x^k in the polynomial p_i(x)
func getCoefficient(iBinary []byte, k int, n int, scLs []*operation.Scalar, l []byte) *operation.Scalar {

	a := make([]*big.Int, len(scLs))
	for i := 0; i < len(scLs); i++ {
		a[i] = privacy_util.ScalarToBigInt(scLs[i])
	}

	//AP2
	curveOrder := polynomial.LInt
	res := polynomial.Poly{big.NewInt(1)}
	var fji polynomial.Poly
	for j := n - 1; j >= 0; j-- {
		fj := polynomial.Poly{a[j], big.NewInt(int64(l[j]))}
		if iBinary[j] == 0 {
			fji = polynomial.Poly{big.NewInt(0), big.NewInt(1)}.Sub(fj, curveOrder)
		} else {
			fji = fj
		}
		res = res.Mul(fji, curveOrder)
	}

	var sc2 *operation.Scalar
	if res.GetDegree() < k {
		sc2 = new(operation.Scalar).FromUint64(0)
	} else {
		sc2 = privacy_util.BigIntToScalar(res[k])
	}
	return sc2
}

func getCoefficientInt(iBinary []byte, k int, n int, a []*big.Int, l []byte) *big.Int {
	res := polynomial.Poly{big.NewInt(1)}
	var fji polynomial.Poly

	for j := n - 1; j >= 0; j-- {
		fj := polynomial.Poly{a[j], big.NewInt(int64(l[j]))}
		if iBinary[j] == 0 {
			fji = polynomial.Poly{big.NewInt(0), big.NewInt(1)}.Sub(fj, polynomial.LInt)
		} else {
			fji = fj
		}

		res = res.Mul(fji, polynomial.LInt)
	}

	if res.GetDegree() < k {
		return big.NewInt(0)
	}
	return res[k]
}
