package bulletproofs

import (
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/privacy_util"
	"github.com/pkg/errors"
)

// ConvertIntToBinary represents a integer number in binary
func ConvertUint64ToBinary(number uint64, n int) []*privacy.Scalar {
	if number == 0 {
		res := make([]*privacy.Scalar, n)
		for i := 0; i < n; i++ {
			res[i] = new(privacy.Scalar).FromUint64(0)
		}
		return res
	}

	binary := make([]*privacy.Scalar, n)

	for i := 0; i < n; i++ {
		binary[i] = new(privacy.Scalar).FromUint64(number % 2)
		number = number / 2
	}
	return binary
}

func computeHPrime(y *privacy.Scalar, N int, H []*privacy.Point) []*privacy.Point {
	yInverse := new(privacy.Scalar).Invert(y)
	HPrime := make([]*privacy.Point, N)
	expyInverse := new(privacy.Scalar).FromUint64(1)
	for i := 0; i < N; i++ {
		HPrime[i] = new(privacy.Point).ScalarMult(H[i], expyInverse)
		expyInverse.Mul(expyInverse, yInverse)
	}
	return HPrime
}

func computeDeltaYZ(z, zSquare *privacy.Scalar, yVector []*privacy.Scalar, N int) (*privacy.Scalar, error) {
	oneNumber := new(privacy.Scalar).FromUint64(1)
	twoNumber := new(privacy.Scalar).FromUint64(2)
	oneVectorN := powerVector(oneNumber, privacy_util.MaxExp)
	twoVectorN := powerVector(twoNumber, privacy_util.MaxExp)
	oneVector := powerVector(oneNumber, N)

	deltaYZ := new(privacy.Scalar).Sub(z, zSquare)
	// ip1 = <1^(n*m), y^(n*m)>
	var ip1, ip2 *privacy.Scalar
	var err error
	if ip1, err = innerProduct(oneVector, yVector); err != nil {
		return nil, err
	} else if ip2, err = innerProduct(oneVectorN, twoVectorN); err != nil {
		return nil, err
	} else {
		deltaYZ.Mul(deltaYZ, ip1)
		sum := new(privacy.Scalar).FromUint64(0)
		zTmp := new(privacy.Scalar).Set(zSquare)
		for j := 0; j < int(N/privacy_util.MaxExp); j++ {
			zTmp.Mul(zTmp, z)
			sum.Add(sum, zTmp)
		}
		sum.Mul(sum, ip2)
		deltaYZ.Sub(deltaYZ, sum)
	}
	return deltaYZ, nil
}

func innerProduct(a []*privacy.Scalar, b []*privacy.Scalar) (*privacy.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Incompatible sizes of a and b")
	}
	result := new(privacy.Scalar).FromUint64(uint64(0))
	for i := range a {
		//res = a[i]*b[i] + res % l
		result.MulAdd(a[i], b[i], result)
	}
	return result, nil
}

func vectorAdd(a []*privacy.Scalar, b []*privacy.Scalar) ([]*privacy.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Incompatible sizes of a and b")
	}
	result := make([]*privacy.Scalar, len(a))
	for i := range a {
		result[i] = new(privacy.Scalar).Add(a[i], b[i])
	}
	return result, nil
}

func setAggregateParams(N int) *bulletproofParams {
	aggParam := new(bulletproofParams)
	aggParam.g = AggParam.g[0:N]
	aggParam.h = AggParam.h[0:N]
	aggParam.u = AggParam.u
	aggParam.cs = AggParam.cs
	return aggParam
}

func roundUpPowTwo(v int) int {
	if v == 0 {
		return 1
	} else {
		v--
		v |= v >> 1
		v |= v >> 2
		v |= v >> 4
		v |= v >> 8
		v |= v >> 16
		v++
		return v
	}
}

func hadamardProduct(a []*privacy.Scalar, b []*privacy.Scalar) ([]*privacy.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("Invalid input")
	}
	result := make([]*privacy.Scalar, len(a))
	for i := 0; i < len(result); i++ {
		result[i] = new(privacy.Scalar).Mul(a[i], b[i])
	}
	return result, nil
}

// powerVector calculates base^n
func powerVector(base *privacy.Scalar, n int) []*privacy.Scalar {
	result := make([]*privacy.Scalar, n)
	result[0] = new(privacy.Scalar).FromUint64(1)
	if n > 1 {
		result[1] = new(privacy.Scalar).Set(base)
		for i := 2; i < n; i++ {
			result[i] = new(privacy.Scalar).Mul(result[i-1], base)
		}
	}
	return result
}

// vectorAddScalar adds a vector to a big int, returns big int array
func vectorAddScalar(v []*privacy.Scalar, s *privacy.Scalar) []*privacy.Scalar {
	result := make([]*privacy.Scalar, len(v))
	for i := range v {
		result[i] = new(privacy.Scalar).Add(v[i], s)
	}
	return result
}

// vectorMulScalar mul a vector to a big int, returns a vector
func vectorMulScalar(v []*privacy.Scalar, s *privacy.Scalar) []*privacy.Scalar {
	result := make([]*privacy.Scalar, len(v))
	for i := range v {
		result[i] = new(privacy.Scalar).Mul(v[i], s)
	}
	return result
}

// CommitAll commits a list of PCM_CAPACITY value(s)
func encodeVectors(l []*privacy.Scalar, r []*privacy.Scalar, g []*privacy.Point, h []*privacy.Point) (*privacy.Point, error) {
	if len(l) != len(r) || len(g) != len(l) || len(h) != len(g) {
		return nil, errors.New("Invalid input")
	}
	tmp1 := new(privacy.Point).MultiScalarMult(l, g)
	tmp2 := new(privacy.Point).MultiScalarMult(r, h)
	res := new(privacy.Point).Add(tmp1, tmp2)
	return res, nil
}

// bulletproofParams includes all generator for aggregated range proof
func newBulletproofParams(m int) *bulletproofParams {
	maxExp := privacy_util.MaxExp
	numCommitValue := privacy_util.NumBase
	maxOutputCoin := privacy_util.MaxOutputCoin
	capacity := maxExp * m // fixed value
	param := new(bulletproofParams)
	param.g = make([]*privacy.Point, capacity)
	param.h = make([]*privacy.Point, capacity)
	csByte := []byte{}

	for i := 0; i < capacity; i++ {
		param.g[i] = privacy.HashToPointFromIndex(int64(numCommitValue+i), privacy.CStringBulletProof)
		param.h[i] = privacy.HashToPointFromIndex(int64(numCommitValue+i+maxOutputCoin*maxExp), privacy.CStringBulletProof)
		csByte = append(csByte, param.g[i].ToBytesS()...)
		csByte = append(csByte, param.h[i].ToBytesS()...)
	}

	param.u = new(privacy.Point)
	param.u = privacy.HashToPointFromIndex(int64(numCommitValue+2*maxOutputCoin*maxExp), privacy.CStringBulletProof)
	csByte = append(csByte, param.u.ToBytesS()...)

	param.cs = privacy.HashToPoint(csByte)
	return param
}

func generateChallenge(hashCache []byte, values []*privacy.Point) *privacy.Scalar {
	bytes := []byte{}
	bytes = append(bytes, hashCache...)
	for i := 0; i < len(values); i++ {
		bytes = append(bytes, values[i].ToBytesS()...)
	}
	hash := privacy.HashToScalar(bytes)
	return hash
}
