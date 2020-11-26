package blsmultisig

import (
	"bytes"
	"errors"
	"math/big"
	"sync"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
)

// Sign return BLS signature
func Sign(data, skBytes []byte, selfIdx int, committee []PublicKey) ([]byte, error) {
	if len(skBytes) != CSKSz {
		return []byte{0}, NewBLSSignatureError(InvalidPrivateKeyErr, errors.New(ErrCodeMessage[InvalidPrivateKeyErr].Message))
	}
	sk := B2I(skBytes)
	if selfIdx >= len(committee) || (selfIdx < 0) || (len(committee) < 1) {
		return []byte{0}, NewBLSSignatureError(InvalidCommitteeInfoErr, errors.New(ErrCodeMessage[InvalidCommitteeInfoErr].Message))
	}
	dataPn := B2G1P(data)
	aiSk := AiGen(committee, selfIdx)
	aiSk.Mul(aiSk, sk)
	aiSk.Mod(aiSk, bn256.Order)
	sig := dataPn.ScalarMult(dataPn, aiSk)
	return CmprG1(sig), nil
}

// SingleSign return BLS signature
// func SingleSign(data, skBytes []byte, selfIdx int, committee []PublicKey) ([]byte, error) {
// 	if len(skBytes) != CSKSz {
// 		return []byte{0}, NewBLSSignatureError(InvalidPrivateKeyErr, errors.New(ErrCodeMessage[InvalidPrivateKeyErr].Message))
// 	}
// 	sk := B2I(skBytes)
// 	if selfIdx >= len(committee) || (selfIdx < 0) || (len(committee) < 1) {
// 		return []byte{0}, NewBLSSignatureError(InvalidCommitteeInfoErr, errors.New(ErrCodeMessage[InvalidCommitteeInfoErr].Message))
// 	}
// 	dataPn := B2G1P(data)
// 	// aiSk := AiGen(committee, selfIdx)
// 	// aiSk.Mul(aiSk, sk)
// 	// aiSk.Mod(aiSk, bn256.Order)
// 	sig := dataPn.ScalarMult(dataPn, sk)
// 	return CmprG1(sig), nil
// }

// Verify verify BLS sig on given data and list public key
func Verify(sig, data []byte, signersIdx []int, committee []PublicKey) (bool, error) {

	for _, idx := range signersIdx {
		if (idx < 0) || (idx >= len(committee)) {
			return false, NewBLSSignatureError(InvalidCommitteeInfoErr, errors.New(ErrCodeMessage[InvalidCommitteeInfoErr].Message))
		}
	}
	if len(signersIdx) > len(committee) || (len(committee) < 1) {
		return false, NewBLSSignatureError(InvalidCommitteeInfoErr, errors.New(ErrCodeMessage[InvalidCommitteeInfoErr].Message))
	}
	wg := sync.WaitGroup{}
	// done := make(chan bool, 1)
	// errChan := make(chan error, 1)
	var err error
	lPair := new(bn256.GT)
	rPair := new(bn256.GT)
	sigPn := new(bn256.G1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		gG2Pn := new(bn256.G2)
		gG2Pn.ScalarBaseMult(big.NewInt(1))
		sigPn, err = DecmprG1(sig)
		if err != nil {
			return
			// errChan <- err
			// return false, NewBLSSignatureError(DecompressFromByteErr, err)
		}
		lPair = bn256.Pair(sigPn, gG2Pn)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		apk := APKGen(committee, signersIdx)
		dataPn := B2G1P(data)
		rPair = bn256.Pair(dataPn, apk)
	}()
	// go func() {
	wg.Wait()
	// close(done)
	// }()
	// select {
	// case <-done:
	// case err := <-errChan:
	if err != nil {
		return false, NewBLSSignatureError(DecompressFromByteErr, err)
	}
	// }
	if !bytes.Equal(lPair.Marshal(), rPair.Marshal()) {
		return false, nil
	}
	// fmt.Printf("ConsLog %v %v %v %v %v %v %v\n", e1.Seconds(), e2.Seconds(), e3.Seconds(), e4.Seconds(), e5.Seconds(), e6.Seconds(), e7.Seconds())
	return true, nil
}

// Verify verify BLS sig on given data and list public key
// func SingleVerify(sig, data []byte, signersIdx []int, committee []PublicKey) (bool, error) {
// 	// if len(skBytes) != CSKSz {
// 	// 	return []byte{0}, NewBLSSignatureError(InvalidPrivateKeyErr, errors.New(ErrCodeMessage[InvalidPrivateKeyErr].Message))
// 	// }
// 	// sk := B2I(skBytes)
// 	for _, idx := range signersIdx {
// 		if (idx < 0) || (idx >= len(committee)) {
// 			return false, NewBLSSignatureError(InvalidCommitteeInfoErr, errors.New(ErrCodeMessage[InvalidCommitteeInfoErr].Message))
// 		}
// 	}
// 	if len(signersIdx) > len(committee) || (len(committee) < 1) {
// 		return false, NewBLSSignatureError(InvalidCommitteeInfoErr, errors.New(ErrCodeMessage[InvalidCommitteeInfoErr].Message))
// 	}
// 	gG2Pn := new(bn256.G2)
// 	gG2Pn.ScalarBaseMult(big.NewInt(1))
// 	sigPn, err := DecmprG1(sig)
// 	if err != nil {
// 		return false, NewBLSSignatureError(DecompressFromByteErr, err)
// 	}
// 	lPair := bn256.Pair(sigPn, gG2Pn)
// 	// apk := APKGen(committee, signersIdx)
// 	pk, _ := DecmprG2(committee[signersIdx[0]])
// 	dataPn := B2G1P(data)
// 	rPair := bn256.Pair(dataPn, pk)
// 	if !reflect.DeepEqual(lPair.Marshal(), rPair.Marshal()) {
// 		return false, nil
// 	}
// 	return true, nil
// }

// Combine combine list of bls signature
func Combine(sigs [][]byte) ([]byte, error) {
	cSigPn, err := DecmprG1(sigs[0])
	if err != nil {
		return []byte{0}, NewBLSSignatureError(DecompressFromByteErr, err)
	}
	for i := 1; i < len(sigs); i++ {
		tmp, err := DecmprG1(sigs[i])
		if err != nil {
			return []byte{0}, NewBLSSignatureError(DecompressFromByteErr, err)
		}
		cSigPn.Add(cSigPn, tmp)
	}
	return CmprG1(cSigPn), nil
}
