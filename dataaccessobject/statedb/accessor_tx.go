package statedb

import (
	"fmt"
	"bytes"
	"errors"
	"math/big"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
)

func StoreSerialNumbers(stateDB *StateDB, tokenID common.Hash, serialNumbers [][]byte, shardID byte) error {
	tokenID = common.ConfidentialAssetID
	for _, serialNumber := range serialNumbers {
		key := GenerateSerialNumberObjectKey(tokenID, shardID, serialNumber)
		value := NewSerialNumberStateWithValue(tokenID, shardID, serialNumber)
		err := stateDB.SetStateObject(SerialNumberObjectType, key, value)
		if err != nil {
			return NewStatedbError(StoreSerialNumberError, err)
		}
	}
	return nil
}

func HasSerialNumber(stateDB *StateDB, tokenID common.Hash, serialNumber []byte, shardID byte) (bool, error) {
	// db key for version 2
	caID := common.ConfidentialAssetID
	key := GenerateSerialNumberObjectKey(caID, shardID, serialNumber)
	s, has, err := stateDB.getSerialNumberState(key)
	if err != nil {
		return false, NewStatedbError(GetSerialNumberError, err)
	}
	if has {
		if bytes.Compare(s.SerialNumber(), serialNumber) != 0 {
			panic("same key wrong value")
			return false, nil
		}
		return has, nil
	}
	// db key for version 1
	key = GenerateSerialNumberObjectKey(tokenID, shardID, serialNumber)
	s, has, err = stateDB.getSerialNumberState(key)
	if err != nil {
		return false, NewStatedbError(GetSerialNumberError, err)
	}
	if has && bytes.Compare(s.SerialNumber(), serialNumber) != 0 {
		panic("same key wrong value")
		return false, nil
	}
	return has, nil
}

func ListSerialNumber(stateDB *StateDB, tokenID common.Hash, shardID byte) (map[string]struct{}, error) {
	tempSerialNumbers := stateDB.getAllSerialNumberByPrefix(tokenID, shardID)
	m := make(map[string]struct{})
	for _, tempSerialNumber := range tempSerialNumbers {
		serialNumber := base58.Base58Check{}.Encode(tempSerialNumber, common.Base58Version)
		m[serialNumber] = struct{}{}
	}
	return m, nil
}

func StoreCommitments(stateDB *StateDB, tokenID common.Hash, commitments [][]byte, shardID byte) error {
	commitmentLengthKey := GenerateCommitmentLengthObjectKey(tokenID, shardID)
	commitmentLength, has, err := stateDB.getCommitmentLengthState(commitmentLengthKey)
	if err != nil {
		return NewStatedbError(GetCommitmentLengthError, err)
	}
	if !has {
		commitmentLength.SetBytes([]byte{0})
	} else {
		temp := commitmentLength.Uint64() + 1
		commitmentLength = new(big.Int).SetUint64(temp)
	}
	for _, commitment := range commitments {
		// store commitment
		keyCommitment := GenerateCommitmentObjectKey(tokenID, shardID, commitment)
		valueCommitment := NewCommitmentStateWithValue(tokenID, shardID, commitment, commitmentLength)
		err = stateDB.SetStateObject(CommitmentObjectType, keyCommitment, valueCommitment)
		if err != nil {
			return NewStatedbError(StoreCommitmentError, err)
		}

		// store commitment index
		keyCommitmentIndex := GenerateCommitmentIndexObjectKey(tokenID, shardID, commitmentLength)
		valueCommitmentIndex := keyCommitment
		err = stateDB.SetStateObject(CommitmentIndexObjectType, keyCommitmentIndex, valueCommitmentIndex)
		if err != nil {
			return NewStatedbError(StoreCommitmentIndexError, err)
		}

		// store commitment length
		valueCommitmentLength := commitmentLength
		err = stateDB.SetStateObject(CommitmentLengthObjectType, commitmentLengthKey, valueCommitmentLength)
		if err != nil {
			return NewStatedbError(StoreCommitmentLengthError, err)
		}
		temp2 := commitmentLength.Uint64() + 1
		commitmentLength = new(big.Int).SetUint64(temp2)
	}
	return nil
}

func HasCommitment(stateDB *StateDB, tokenID common.Hash, commitment []byte, shardID byte) (bool, error) {
	key := GenerateCommitmentObjectKey(tokenID, shardID, commitment)
	c, has, err := stateDB.getCommitmentState(key)
	if err != nil {
		return false, NewStatedbError(GetCommitmentError, err)
	}
	if has && bytes.Compare(c.Commitment(), commitment) != 0 {
		panic("same key wrong value")
		return false, nil
	}
	return has, nil
}

func HasOTACoinIndex(stateDB *StateDB, tokenID common.Hash, index uint64, shardID byte) (bool, error) {
	otaCoinIndexTemp := new(big.Int).SetUint64(index)
	key := GenerateOTACoinIndexObjectKey(tokenID, shardID, otaCoinIndexTemp)
	c, has, err := stateDB.getOTACoinIndexState(key)
	if err != nil {
		return false, NewStatedbError(GetOTACoinIndexError, err)
	}
	if has && c.Index().Uint64() != index {
		panic("same key wrong value")
		return false, nil
	}
	return has, nil
}

func HasCommitmentIndex(stateDB *StateDB, tokenID common.Hash, commitmentIndex uint64, shardID byte) (bool, error) {
	commitmentIndexTemp := new(big.Int).SetUint64(commitmentIndex)
	key := GenerateCommitmentIndexObjectKey(tokenID, shardID, commitmentIndexTemp)
	c, has, err := stateDB.getCommitmentIndexState(key)
	if err != nil {
		return false, NewStatedbError(GetCommitmentIndexError, err)
	}
	//fmt.PRintln(c)
	//fmt.Println("CIndex =", c.GetIndex().Uint64())
	//fmt.Println("commitmentIndex =", commitmentIndex)

	if has && c.Index().Uint64() != commitmentIndex {
		fmt.Println(has)
		fmt.Println("CIndexByte =", c.Index())
		fmt.Println("CIndex =", c.Index().Uint64())
		fmt.Println("commitmentIndex =", commitmentIndex)
		panic("same key wrong value")
		return false, nil
	}
	return has, nil
}

func GetOTACoinByIndex(stateDB *StateDB, tokenID common.Hash, index uint64, shardID byte) ([]byte, error) {
	otaCoinIndexTemp := new(big.Int).SetUint64(index)
	key := GenerateOTACoinIndexObjectKey(tokenID, shardID, otaCoinIndexTemp)
	c, has, err := stateDB.getOTACoinIndexState(key)
	if err != nil {
		return []byte{}, NewStatedbError(GetOTACoinIndexError, err)
	}
	if !has {
		return []byte{}, NewStatedbError(GetOTACoinIndexError, errors.New("no value exist"))
	}
	if c.Index().Uint64() != index {
		panic("same key wrong value")
		return []byte{}, nil
	}
	return c.outputCoin, nil
}

func GetCommitmentByIndex(stateDB *StateDB, tokenID common.Hash, commitmentIndex uint64, shardID byte) ([]byte, error) {
	commitmentIndexTemp := new(big.Int).SetUint64(commitmentIndex)
	key := GenerateCommitmentIndexObjectKey(tokenID, shardID, commitmentIndexTemp)
	c, has, err := stateDB.getCommitmentIndexState(key)
	if err != nil {
		return []byte{}, NewStatedbError(GetCommitmentIndexError, err)
	}
	if !has {
		return []byte{}, NewStatedbError(GetCommitmentIndexError, errors.New("no value exist"))
	}
	//fmt.Println("CommitmentIndex =", commitmentIndex)
	//fmt.Println("Cindex =", c.index)
	//if c.GetIndex().Uint64() != commitmentIndex {
	//	panic("same key wrong value")
	//	return []byte{}, nil
	//}
	return c.commitment, nil
}

func GetOTACoinIndex(stateDB *StateDB, tokenID common.Hash, ota []byte) (*big.Int, error) {
	key := GenerateOnetimeAddressObjectKey(tokenID, ota)
	c, has, err := stateDB.getOnetimeAddressState(key)
	if err != nil {
		return nil, NewStatedbError(GetOTACoinIndexError, err)
	}
	if !has {
		return nil, NewStatedbError(GetOTACoinIndexError, errors.New("no value exist"))
	}
	if bytes.Compare(c.publicKey, ota) != 0 {
		panic("same key wrong value")
		return nil, nil
	}
	return c.Index(), nil
}

// GetCommitmentIndex - return index of commitment in db list
func GetCommitmentIndex(stateDB *StateDB, tokenID common.Hash, commitment []byte, shardID byte) (*big.Int, error) {
	key := GenerateCommitmentObjectKey(tokenID, shardID, commitment)
	c, has, err := stateDB.getCommitmentState(key)
	if err != nil {
		return nil, NewStatedbError(GetCommitmentError, err)
	}
	if !has {
		return nil, NewStatedbError(GetCommitmentError, errors.New("no value exist"))
	}
	if bytes.Compare(c.Commitment(), commitment) != 0 {
		panic("same key wrong value")
		return nil, nil
	}
	return c.Index(), nil
}

func GetOTACoinLength(stateDB *StateDB, tokenID common.Hash, shardID byte) (*big.Int, error) {
	key := GenerateOTACoinLengthObjectKey(tokenID, shardID)
	length, has, err := stateDB.getOTACoinLengthState(key)
	if err != nil {
		return nil, NewStatedbError(GetOTACoinLengthError, err)
	}
	if !has {
		return new(big.Int).SetUint64(0), NewStatedbError(GetOTACoinLengthError, errors.New("no value exist"))
	}
	return new(big.Int).SetUint64(length.Uint64() + 1), nil
}

// GetCommitmentIndex - return index of commitment in db list
func GetCommitmentLength(stateDB *StateDB, tokenID common.Hash, shardID byte) (*big.Int, error) {
	key := GenerateCommitmentLengthObjectKey(tokenID, shardID)
	length, has, err := stateDB.getCommitmentLengthState(key)
	if err != nil {
		return nil, NewStatedbError(GetCommitmentLengthError, err)
	}
	if !has {
		return new(big.Int).SetUint64(0), NewStatedbError(GetCommitmentLengthError, errors.New("no value exist"))
	}
	return new(big.Int).SetUint64(length.Uint64() + 1), nil
}

func ListCommitment(stateDB *StateDB, tokenID common.Hash, shardID byte) (map[string]uint64, error) {
	m := stateDB.getAllCommitmentStateByPrefix(tokenID, shardID)
	return m, nil
}

// ListCommitmentIndices -  return all commitment index and its value
func ListCommitmentIndices(stateDB *StateDB, tokenID common.Hash, shardID byte) (map[uint64]string, error) {
	m := stateDB.getAllCommitmentStateByPrefix(tokenID, shardID)
	reverseM := make(map[uint64]string)
	for k, v := range m {
		reverseM[v] = k
	}
	return reverseM, nil
}

// outputCoins and otas should have the same length
func StoreOTACoinsAndOnetimeAddresses(stateDB *StateDB, tokenID common.Hash, height uint64, outputCoins [][]byte, otas [][]byte, shardID byte) error {
	otaCoinLengthKey := GenerateOTACoinLengthObjectKey(tokenID, shardID)
	otaCoinLength, has, err := stateDB.getOTACoinLengthState(otaCoinLengthKey)
	if err != nil {
		return NewStatedbError(GetOTACoinLengthError, err)
	}
	if !has {
		otaCoinLength.SetBytes([]byte{0})
	} else {
		temp := otaCoinLength.Uint64() + 1
		otaCoinLength = new(big.Int).SetUint64(temp)
	}
	heightBytes := common.Uint64ToBytes(height)
	for index := 0; index < len(outputCoins); index += 1 {
		outputCoin, ota := outputCoins[index], otas[index]

		// Store OnetimeAddress
		key := GenerateOnetimeAddressObjectKey(tokenID, ota)
		value := NewOnetimeAddressStateWithValue(tokenID, ota, otaCoinLength)
		if err := stateDB.SetStateObject(OnetimeAddressObjectType, key, value); err != nil {
			return NewStatedbError(StoreOnetimeAddressError, err)
		}

		// Store OTACoin
		keyOTACoin := GenerateOTACoinObjectKey(tokenID, shardID, heightBytes, outputCoin)
		valueOTACoin := NewOTACoinStateWithValue(tokenID, shardID, heightBytes, outputCoin, otaCoinLength)
		if err := stateDB.SetStateObject(OTACoinObjectType, keyOTACoin, valueOTACoin); err != nil {
			return NewStatedbError(StoreOTACoinError, err)
		}
		//fmt.Println("Key = ", key[:])
		//fmt.Println("Value = ", value)
		//fmt.Println("KeyPrefix = ", GetOTACoinPrefix(tokenID, shardID, heightBytes))
		//keySuffix := common.HashH(outputCoin)
		//fmt.Println("KeySuffix = ", keySuffix[:][:20])

		// store otacoin index
		keyOTAIndex := GenerateOTACoinIndexObjectKey(tokenID, shardID, otaCoinLength)
		valueOTACoinIndex := keyOTACoin
		if err := stateDB.SetStateObject(OTACoinIndexObjectType, keyOTAIndex, valueOTACoinIndex); err != nil {
			return NewStatedbError(StoreOTACoinIndexError, err)
		}

		// store otacoin length
		if err := stateDB.SetStateObject(OTACoinLengthObjectType, otaCoinLengthKey, otaCoinLength); err != nil {
			return NewStatedbError(StoreOTACoinLengthError, err)
		}

		// Caution: ask Hieu before change these lines
		temp2 := otaCoinLength.Uint64() + 1
		otaCoinLength = new(big.Int).SetUint64(temp2)
	}
	//fmt.Println("The database length currently is", otaCoinLength)
	return nil
}

func GetOTACoinsByHeight(stateDB *StateDB, tokenID common.Hash, shardID byte, height uint64) ([][]byte, error) {
	heightBytes := common.Uint64ToBytes(height)
	otaCoinStates := stateDB.getAllOTACoinsByPrefix(tokenID, shardID, heightBytes)
	onetimeAddressesBytes := [][]byte{}
	for _, otaCoinState := range otaCoinStates {
		onetimeAddressesBytes = append(onetimeAddressesBytes, otaCoinState.OutputCoin())
	}
	return onetimeAddressesBytes, nil
}

func StoreOutputCoins(stateDB *StateDB, tokenID common.Hash, publicKey []byte, outputCoins [][]byte, shardID byte) error {
	for _, outputCoin := range outputCoins {
		key := GenerateOutputCoinObjectKey(tokenID, shardID, publicKey, outputCoin)
		value := NewOutputCoinStateWithValue(tokenID, shardID, publicKey, outputCoin)
		err := stateDB.SetStateObject(OutputCoinObjectType, key, value)
		if err != nil {
			return NewStatedbError(StoreOutputCoinError, err)
		}
	}
	return nil
}

func HasOnetimeAddress(stateDB *StateDB, tokenID common.Hash, ota []byte) (bool, error) {
	key := GenerateOnetimeAddressObjectKey(tokenID, ota)
	onetimeAddressState, has, err := stateDB.getOnetimeAddressState(key)
	if err != nil {
		return false, NewStatedbError(GetSNDerivatorError, err)
	}
	if has && bytes.Compare(onetimeAddressState.PublicKey(), ota) != 0 {
		panic("same key wrong value")
	}
	return has, nil
}

func GetOutcoinsByPubkey(stateDB *StateDB, tokenID common.Hash, publicKey []byte, shardID byte) ([][]byte, error) {
	outputCoinStates := stateDB.getAllOutputCoinState(tokenID, shardID, publicKey)
	o := [][]byte{}
	for _, outputCoinState := range outputCoinStates {
		o = append(o, outputCoinState.OutputCoin())
	}
	return o, nil
}

// StoreSNDerivators - store list serialNumbers by shardID
func StoreSNDerivators(stateDB *StateDB, tokenID common.Hash, snds [][]byte) error {
	for _, snd := range snds {
		key := GenerateSNDerivatorObjectKey(tokenID, snd)
		value := NewSNDerivatorStateWithValue(tokenID, snd)
		err := stateDB.SetStateObject(SNDerivatorObjectType, key, value)
		if err != nil {
			return NewStatedbError(StoreSNDerivatorError, err)
		}
	}
	return nil
}

// HasSNDerivator - Check SnDerivator in list SnDerivators by shardID
func HasSNDerivator(stateDB *StateDB, tokenID common.Hash, snd []byte) (bool, error) {
	key := GenerateSNDerivatorObjectKey(tokenID, snd)
	sndState, has, err := stateDB.getSNDerivatorState(key)
	if err != nil {
		return false, NewStatedbError(GetSNDerivatorError, err)
	}
	if has && bytes.Compare(sndState.Snd(), snd) != 0 {
		panic("same key wrong value")
	}
	return has, nil
}

func ListSNDerivator(stateDB *StateDB, tokenID common.Hash) ([][]byte, error) {
	return stateDB.getAllSNDerivatorStateByPrefix(tokenID), nil
}
