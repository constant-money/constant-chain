package bridgesig

import (
	"reflect"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func Sign(keyBytes []byte, data []byte) ([]byte, error) {
	if len(keyBytes) != CSKSz {
		return []byte{0}, NewBriSignatureError(InvalidPrivateKeyErr, nil)
	}
	sk, err := ethcrypto.ToECDSA(keyBytes)
	if err != nil {
		return nil, err
	}
	hash := ethcrypto.Keccak256Hash(data)
	sig, err := ethcrypto.Sign(hash.Bytes(), sk)
	if err != nil {
		return []byte{0}, NewBriSignatureError(SignDataErr, err)
	}
	return sig, nil
}

func Verify(pubkeyBytes []byte, data []byte, sig []byte) (bool, error) {
	//fmt.Println(sig, len(sig))
	//fmt.Println(pubkeyBytes, len(pubkeyBytes))
	//fmt.Println(data, len(data))
	hash := ethcrypto.Keccak256Hash(data)
	pk, err := ethcrypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		return false, NewBriSignatureError(InvalidSignatureErr, err)
	}
	if !reflect.DeepEqual(pubkeyBytes, ethcrypto.CompressPubkey(pk)) {
		return false, nil
	}
	return true, nil
}
