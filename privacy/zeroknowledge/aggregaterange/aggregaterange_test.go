package aggregaterange

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	m.Run()
}

var _ = func() (_ struct{}) {
	fmt.Println("This runs before init()!")
	privacy.Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

func TestPad(t *testing.T) {
	data := []struct {
		number       int
		paddedNumber int
	}{
		{1000, 1024},
		{3, 4},
		{5, 8},
	}

	for _, item := range data {
		num := pad(item.number)
		assert.Equal(t, item.paddedNumber, num)
	}
}

func TestPowerVector(t *testing.T) {
	twoVector := powerVector(new(privacy.Scalar).FromUint64(2), 5)
	assert.Equal(t, 5, len(twoVector))
}

func TestInnerProduct(t *testing.T) {
	for j := 0; j < 100; j++ {
		n := maxExp
		a := make([]*privacy.Scalar, n)
		b := make([]*privacy.Scalar, n)
		uinta := make([]uint64, n)
		uintb := make([]uint64, n)
		uintc := uint64(0)
		for i := 0; i < n; i++ {
			uinta[i] = uint64(rand.Intn(100000000))
			uintb[i] = uint64(rand.Intn(100000000))
			a[i] = new(privacy.Scalar).FromUint64(uinta[i])
			b[i] = new(privacy.Scalar).FromUint64(uintb[i])
			uintc += uinta[i] * uintb[i]
		}

		c, _ := innerProduct(a, b)
		assert.Equal(t, new(privacy.Scalar).FromUint64(uintc), c)
	}
}

func TestEncodeVectors(t *testing.T) {
	for i := 0; i < 100; i++ {
		var AggParam = newBulletproofParams(1)
		n := maxExp
		a := make([]*privacy.Scalar, n)
		b := make([]*privacy.Scalar, n)
		G := make([]*privacy.Point, n)
		H := make([]*privacy.Point, n)

		for i := range a {
			a[i] = privacy.RandomScalar()
			b[i] = privacy.RandomScalar()
			G[i] = new(privacy.Point).Set(AggParam.g[i])
			H[i] = new(privacy.Point).Set(AggParam.h[i])
		}

		actualRes, err := encodeVectors(a, b, G, H)
		if err != nil {
			privacy.Logger.Log.Info("Err: %v\n", err)
		}

		expectedRes := new(privacy.Point).Identity()
		for i := 0; i < n; i++ {
			expectedRes.Add(expectedRes, new(privacy.Point).ScalarMult(G[i], a[i]))
			expectedRes.Add(expectedRes, new(privacy.Point).ScalarMult(H[i], b[i]))
		}

		assert.Equal(t, expectedRes, actualRes)
	}
}

func TestAggregatedRangeProveVerify(t *testing.T) {
	for i := 0; i < 10; i++ {
		//prepare witness for Aggregated range protocol
		wit := new(AggregatedRangeWitness)
		numValue := rand.Intn(maxOutputNumber)
		values := make([]uint64, numValue)
		rands := make([]*privacy.Scalar, numValue)

		for i := range values {
			values[i] = uint64(rand.Uint64())
			rands[i] = privacy.RandomScalar()
		}
		wit.Set(values, rands)

		// proving
		proof, err := wit.Prove()
		assert.Equal(t, nil, err)

		// verify the proof
		res, err := proof.Verify()
		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)

		// validate sanity for proof
		isValidSanity := proof.ValidateSanity()
		assert.Equal(t, true, isValidSanity)

		// convert proof to bytes array
		bytes := proof.Bytes()
		expectProofSize := EstimateMultiRangeProofSize(numValue)
		assert.Equal(t, int(expectProofSize), len(bytes))

		// new aggregatedRangeProof from bytes array
		proof2 := new(AggregatedRangeProof)
		proof2.SetBytes(bytes)

		// verify the proof
		res, err = proof2.Verify()
		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)
	}
}

func TestAggregatedRangeProveVerifyUltraFast(t *testing.T) {
	count := 10
	proofs := make([]*AggregatedRangeProof, 0)

	for i := 0; i < count; i++ {
		//prepare witness for Aggregated range protocol
		wit := new(AggregatedRangeWitness)
		numValue := rand.Intn(maxOutputNumber)
		values := make([]uint64, numValue)
		rands := make([]*privacy.Scalar, numValue)

		for i := range values {
			values[i] = uint64(rand.Uint64())
			rands[i] = privacy.RandomScalar()
		}
		wit.Set(values, rands)

		// proving
		proof, err := wit.Prove()
		assert.Equal(t, nil, err)

		proofs = append(proofs, proof)
	}
	// verify the proof faster
	res, err, _ := VerifyBatch(proofs)
	assert.Equal(t, true, res)
	assert.Equal(t, nil, err)
}

func TestBenchmarkAggregatedRangeProveVerifyUltraFast(t *testing.T) {
	for k := 1; k < 100; k += 5 {
		count := k
		proofs := make([]*AggregatedRangeProof, 0)
		start := time.Now()
		t1 := time.Now().Sub(start)
		for i := 0; i < count; i++ {
			//prepare witness for Aggregated range protocol
			wit := new(AggregatedRangeWitness)
			//numValue := rand.Intn(maxOutputNumber)
			numValue := 2
			values := make([]uint64, numValue)
			rands := make([]*privacy.Scalar, numValue)

			for i := range values {
				values[i] = uint64(rand.Uint64())
				rands[i] = privacy.RandomScalar()
			}
			wit.Set(values, rands)

			// proving
			proof, err := wit.Prove()
			assert.Equal(t, nil, err)
			start := time.Now()
			proof.Verify()
			t1 += time.Now().Sub(start)

			proofs = append(proofs, proof)
		}
		// verify the proof faster
		start = time.Now()
		res, err, _ := VerifyBatch(proofs)
		fmt.Println(t1, time.Now().Sub(start), k)

		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)
	}
}

func TestInnerProductProveVerify(t *testing.T) {
	for k := 0; k < 1; k++ {
		numValue := rand.Intn(maxOutputNumber)
		numValuePad := pad(numValue)
		aggParam := new(bulletproofParams)
		aggParam.g = AggParam.g[0 : numValuePad*maxExp]
		aggParam.h = AggParam.h[0 : numValuePad*maxExp]
		aggParam.u = AggParam.u
		aggParam.cs = AggParam.cs

		wit := new(InnerProductWitness)
		n := maxExp * numValuePad
		wit.a = make([]*privacy.Scalar, n)
		wit.b = make([]*privacy.Scalar, n)

		for i := range wit.a {
			wit.a[i] = new(privacy.Scalar).FromUint64(uint64(rand.Intn(1000000)))
			wit.b[i] = new(privacy.Scalar).FromUint64(uint64(rand.Intn(1000000)))
		}

		c, err := innerProduct(wit.a, wit.b)

		if err != nil {
			privacy.Logger.Log.Info("Err: %v\n", err)
		}
		wit.p = new(privacy.Point).ScalarMult(aggParam.u, c)

		for i := range wit.a {
			wit.p.Add(wit.p, new(privacy.Point).ScalarMult(aggParam.g[i], wit.a[i]))
			wit.p.Add(wit.p, new(privacy.Point).ScalarMult(aggParam.h[i], wit.b[i]))
		}

		proof, err := wit.Prove(aggParam)
		if err != nil {
			fmt.Printf("Err: %v\n", err)
			return
		}
		res2 := proof.Verify(aggParam)
		assert.Equal(t, true, res2)

		bytes := proof.Bytes()
		proof2 := new(InnerProductProof)
		proof2.SetBytes(bytes)
		res3 := proof2.Verify(aggParam)
		assert.Equal(t, true, res3)
		res3prime := proof2.Verify(aggParam)
		assert.Equal(t, true, res3prime)

	}
}

func TestInnerProductProveVerifyUltraFast(t *testing.T) {
	proofs := make([]*InnerProductProof, 0)
	csList := make([][]byte, 0)
	count := 15
	for k := 0; k < count; k++ {
		numValue := rand.Intn(maxOutputNumber)
		numValuePad := pad(numValue)
		aggParam := new(bulletproofParams)
		aggParam.g = AggParam.g[0 : numValuePad*maxExp]
		aggParam.h = AggParam.h[0 : numValuePad*maxExp]
		aggParam.u = AggParam.u
		aggParam.cs = AggParam.cs

		wit := new(InnerProductWitness)
		n := maxExp * numValuePad
		wit.a = make([]*privacy.Scalar, n)
		wit.b = make([]*privacy.Scalar, n)

		for i := range wit.a {
			wit.a[i] = new(privacy.Scalar).FromUint64(uint64(rand.Intn(1000000)))
			wit.b[i] = new(privacy.Scalar).FromUint64(uint64(rand.Intn(1000000)))
		}

		c, err := innerProduct(wit.a, wit.b)
		if err != nil {
			privacy.Logger.Log.Info("Err: %v\n", err)
		}
		if k == 0 {
			wit.p = new(privacy.Point).ScalarMult(aggParam.u, c.Add(c, new(privacy.Scalar).FromUint64(1)))
		} else {
			wit.p = new(privacy.Point).ScalarMult(aggParam.u, c)
		}

		for i := range wit.a {
			wit.p.Add(wit.p, new(privacy.Point).ScalarMult(aggParam.g[i], wit.a[i]))
			if k == count-1 {
				wit.p.Add(wit.p, new(privacy.Point).ScalarMult(aggParam.h[i], wit.a[i]))
			} else {
				wit.p.Add(wit.p, new(privacy.Point).ScalarMult(aggParam.h[i], wit.b[i]))
			}
		}

		proof, err := wit.Prove(aggParam)
		if err != nil {
			fmt.Printf("Err: %v\n", err)
			return
		}
		proofs = append(proofs, proof)
		csList = append(csList, aggParam.cs)
	}
	res := VerifyBatchingInnerProductProofs(proofs, csList)
	assert.Equal(t, false, res)
	res = VerifyBatchingInnerProductProofs(proofs[1:], csList[1:])
	assert.Equal(t, false, res)
	res = VerifyBatchingInnerProductProofs(proofs[:len(proofs)-1], csList[:len(proofs)-1])
	assert.Equal(t, false, res)
	res = VerifyBatchingInnerProductProofs(proofs[1:len(proofs)-1], csList[1:len(proofs)-1])
	assert.Equal(t, true, res)
}
func benchmarkAggRangeProof_Proof(numberofOutput int, b *testing.B) {
	wit := new(AggregatedRangeWitness)
	values := make([]uint64, numberofOutput)
	rands := make([]*privacy.Scalar, numberofOutput)

	for i := range values {
		values[i] = uint64(rand.Uint64())
		rands[i] = privacy.RandomScalar()
	}
	wit.Set(values, rands)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wit.Prove()
	}
}

func TestAnStrictInnerProductProveVerifyUltraFast(t *testing.T) {
	proofs := make([]*InnerProductProof, 0)
	csList := make([][]byte, 0)
	count := 5
	for k := 0; k < count; k++ {
		numValue := rand.Intn(maxOutputNumber)
		numValuePad := pad(numValue)
		aggParam := new(bulletproofParams)
		aggParam.g = AggParam.g[0 : numValuePad*maxExp]
		aggParam.h = AggParam.h[0 : numValuePad*maxExp]
		aggParam.u = AggParam.u
		aggParam.cs = AggParam.cs
		wit := new(InnerProductWitness)
		n := maxExp * numValuePad
		wit.a = make([]*privacy.Scalar, n)
		wit.b = make([]*privacy.Scalar, n)
		for i := range wit.a {
			wit.a[i] = new(privacy.Scalar).FromUint64(uint64(rand.Intn(1000000)))
			wit.b[i] = new(privacy.Scalar).FromUint64(uint64(rand.Intn(1000000)))
		}
		c, err := innerProduct(wit.a, wit.b)
		if err != nil {
			privacy.Logger.Log.Info("Err: %v\n", err)
		}
		wit.p = new(privacy.Point).ScalarMult(aggParam.u, c)
		for i := range wit.a {
			wit.p.Add(wit.p, new(privacy.Point).ScalarMult(aggParam.g[i], wit.a[i]))
			wit.p.Add(wit.p, new(privacy.Point).ScalarMult(aggParam.h[i], wit.b[i]))
		}
		proof, err := wit.Prove(aggParam)
		if err != nil {
			fmt.Printf("Err: %v\n", err)
			return
		}
		proofs = append(proofs, proof)
		csList = append(csList, aggParam.cs)
	}
	res := VerifyBatchingInnerProductProofs(proofs, csList)
	assert.Equal(t, true, res)
	for j := 0; j < 50; j += 1 {
		i := common.RandInt() % len(proofs)
		r := common.RandInt() % 5
		if r == 0 {
			ran := common.RandInt() % len(proofs[i].l)
			remember := proofs[i].l[ran]
			proofs[i].l[ran] = obfuscatePoint(proofs[i].l[ran])
			assert.NotEqual(t, remember, proofs[i].l[ran])
			res := VerifyBatchingInnerProductProofs(proofs, csList)
			assert.Equal(t, false, res)
			proofs[i].l[ran] = remember
		} else if r == 1 {
			ran := common.RandInt() % len(proofs[i].r)
			remember := proofs[i].r[ran]
			proofs[i].r[ran] = obfuscatePoint(proofs[i].r[ran])
			assert.NotEqual(t, remember, proofs[i].r[ran])
			res := VerifyBatchingInnerProductProofs(proofs, csList)
			assert.Equal(t, false, res)
			proofs[i].r[ran] = remember
		} else if r == 2 {
			remember := proofs[i].a
			proofs[i].a = obfuscateScalar(proofs[i].a)
			assert.NotEqual(t, remember, proofs[i].a)
			res := VerifyBatchingInnerProductProofs(proofs, csList)
			assert.Equal(t, false, res)
			proofs[i].a = remember
		} else if r == 3 {
			remember := proofs[i].b
			proofs[i].b = obfuscateScalar(proofs[i].b)
			assert.NotEqual(t, remember, proofs[i].b)
			res := VerifyBatchingInnerProductProofs(proofs, csList)
			assert.Equal(t, false, res)
			proofs[i].b = remember
		} else if r == 4 {
			remember := proofs[i].p
			proofs[i].p = obfuscatePoint(proofs[i].p)
			assert.NotEqual(t, remember, proofs[i].p)
			res := VerifyBatchingInnerProductProofs(proofs, csList)
			assert.Equal(t, false, res)
			proofs[i].p = remember
		}
	}
	res = VerifyBatchingInnerProductProofs(proofs, csList)
	assert.Equal(t, true, res)
}
func obfuscatePoint(value *privacy.Point) *privacy.Point {
	for {
		k := value.GetKey()
		r := common.RandInt() % len(k)
		i := common.RandInt() % 8
		k[r] ^= (1 << uint8(i))
		after, err := new(privacy.Point).SetKey(&k)
		if err == nil {
			return after
		}
	}
}
func obfuscateScalar(value *privacy.Scalar) *privacy.Scalar {
	for {
		k := value.GetKey()
		r := common.RandInt() % len(k)
		i := common.RandInt() % 8
		k[r] ^= (1 << uint8(i))
		after, err := new(privacy.Scalar).SetKey(&k)
		if err == nil {
			return after
		}
	}
}

func benchmarkAggRangeProof_Verify(numberofOutput int, b *testing.B) {
	wit := new(AggregatedRangeWitness)
	values := make([]uint64, numberofOutput)
	rands := make([]*privacy.Scalar, numberofOutput)

	for i := range values {
		values[i] = uint64(common.RandInt64())
		rands[i] = privacy.RandomScalar()
	}
	wit.Set(values, rands)
	proof, _ := wit.Prove()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		proof.Verify()
	}
}

func benchmarkAggRangeProof_VerifyFaster(numberofOutput int, b *testing.B) {
	wit := new(AggregatedRangeWitness)
	values := make([]uint64, numberofOutput)
	rands := make([]*privacy.Scalar, numberofOutput)

	for i := range values {
		values[i] = uint64(common.RandInt64())
		rands[i] = privacy.RandomScalar()
	}
	wit.Set(values, rands)
	proof, _ := wit.Prove()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		proof.Verify()
	}
}

func BenchmarkAggregatedRangeWitness_Prove1(b *testing.B) { benchmarkAggRangeProof_Proof(1, b) }
func BenchmarkAggregatedRangeProof_Verify1(b *testing.B)  { benchmarkAggRangeProof_Verify(1, b) }
func BenchmarkAggregatedRangeProof_VerifyFaster1(b *testing.B) {
	benchmarkAggRangeProof_VerifyFaster(1, b)
}

func BenchmarkAggregatedRangeWitness_Prove2(b *testing.B) { benchmarkAggRangeProof_Proof(2, b) }
func BenchmarkAggregatedRangeProof_Verify2(b *testing.B)  { benchmarkAggRangeProof_Verify(2, b) }
func BenchmarkAggregatedRangeProof_VerifyFaster2(b *testing.B) {
	benchmarkAggRangeProof_VerifyFaster(2, b)
}

func BenchmarkAggregatedRangeWitness_Prove4(b *testing.B) { benchmarkAggRangeProof_Proof(4, b) }
func BenchmarkAggregatedRangeProof_Verify4(b *testing.B)  { benchmarkAggRangeProof_Verify(4, b) }
func BenchmarkAggregatedRangeProof_VerifyFaster4(b *testing.B) {
	benchmarkAggRangeProof_VerifyFaster(4, b)
}

func BenchmarkAggregatedRangeWitness_Prove8(b *testing.B) { benchmarkAggRangeProof_Proof(8, b) }
func BenchmarkAggregatedRangeProof_Verify8(b *testing.B)  { benchmarkAggRangeProof_Verify(8, b) }
func BenchmarkAggregatedRangeProof_VerifyFaster8(b *testing.B) {
	benchmarkAggRangeProof_VerifyFaster(8, b)
}

func BenchmarkAggregatedRangeWitness_Prove16(b *testing.B) { benchmarkAggRangeProof_Proof(16, b) }
func BenchmarkAggregatedRangeProof_Verify16(b *testing.B)  { benchmarkAggRangeProof_Verify(16, b) }
func BenchmarkAggregatedRangeProof_VerifyFaster16(b *testing.B) {
	benchmarkAggRangeProof_VerifyFaster(16, b)
}
