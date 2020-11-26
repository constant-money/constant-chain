package blsmultisig

import (
	"math/big"
	"sync"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"github.com/incognitochain/incognito-chain/common"
)

// KeyGen take an input seed and return BLS Key
func KeyGen(seed []byte) (*big.Int, *bn256.G2) {
	sk := SKGen(seed)
	return sk, PKGen(sk)
}

// SKGen take a seed and return BLS secret key
func SKGen(seed []byte) *big.Int {
	sk := big.NewInt(0)
	sk.SetBytes(common.HashB(seed))
	for {
		if sk.Cmp(bn256.Order) == -1 {
			break
		}
		sk.SetBytes(Hash4Bls(sk.Bytes()))
	}
	return sk
}

// PKGen take a secret key and return BLS public key
func PKGen(sk *big.Int) *bn256.G2 {
	pk := new(bn256.G2)
	pk = pk.ScalarBaseMult(sk)
	return pk
}

// AKGen take a seed and return BLS secret key
func AKGen(idxPKByte []byte, combinedPKBytes []byte) (*bn256.G2, *big.Int) {
	// cal akByte
	akByte := []byte{}
	akByte = append(akByte, idxPKByte...)
	akByte = append(akByte, combinedPKBytes...)
	akByte = Hash4Bls(akByte)

	// cal akBInt
	akBInt := B2I(akByte)

	var pkPn *bn256.G2
	pkPn, _ = DecmprG2(idxPKByte)

	result := new(bn256.G2)
	result.ScalarMult(pkPn, akBInt)

	return result, akBInt
}

// ListAKGen take a seed and return BLS secret key
// func APKGen(committee []PublicKey, idx []int) *bn256.G2 {
// 	// apk := new(bn256.G2)
// 	apk, _ := AKGen(committee, idx[0])
// 	// apk.ScalarMult(CommonAPs[signerIdx[0]], big.NewInt(1))
// 	wg := sync.WaitGroup{}
// 	apkTmpList := make([]*bn256.G2, len(idx)-1)
// 	for i := 1; i < len(idx); i++ {
// 		wg.Add(1)
// 		go func(index int) {
// 			apkTmp, _ := AKGen(committee, idx[index])
// 			apkTmpList[index-1] = apkTmp
// 			wg.Done()
// 		}(i)
// 	}
// 	wg.Wait()
// 	for _, apkTmp := range apkTmpList {
// 		apk.Add(apk, apkTmp)
// 	}

// 	return apk
// }

func APKGen(committee []PublicKey, idx []int) *bn256.G2 {
	apkTmpList := make([]*bn256.G2, len(idx))

	// pre-calculate for combined committee
	combinedCommittee := []byte{}
	for i := 0; i < len(committee); i++ {
		combinedCommittee = append(combinedCommittee, committee[i]...)
	}

	// async to process
	wg := sync.WaitGroup{}
	wg.Add(len(idx))
	for i := 0; i < len(idx); i++ {

		committeeByte := committee[idx[i]]
		go func(index int, committeeByte []byte, combinedCommittee []byte, wg *sync.WaitGroup) {
			defer wg.Done()
			apkTmpList[index], _ = AKGen(committeeByte, combinedCommittee)
		}(i, committeeByte, combinedCommittee, &wg)
	}
	wg.Wait()

	// get final result
	res := new(bn256.G2)
	res.Unmarshal(apkTmpList[0].Marshal())
	for i := 1; i < len(idx); i++ {
		res.Add(res, apkTmpList[i])
	}
	return res
}

func AiGen(listPKBytes []PublicKey, id int) *big.Int {
	akByte := []byte{}
	akByte = append(akByte, listPKBytes[id]...)
	for i := 0; i < len(listPKBytes); i++ {
		akByte = append(akByte, listPKBytes[i]...)
	}
	akByte = Hash4Bls(akByte)
	akBInt := B2I(akByte)
	return akBInt
}

// SKBytes take input secretkey integer and return secretkey bytes
func SKBytes(sk *big.Int) SecretKey {
	return I2Bytes(sk, CSKSz)
}

// PKBytes take input publickey point and return publickey bytes
func PKBytes(pk *bn256.G2) PublicKey {
	return CmprG2(pk)
}
