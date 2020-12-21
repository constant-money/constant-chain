package bnb

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestDecodePubKeyValidator(t *testing.T) {
	b64EncodePubKey := "uND4Li1FIzpmjmEe9RZGZlKr53zLP8ZHUP8DSQCZpN4="

	pubKeyBytes, _ := base64.StdEncoding.DecodeString(b64EncodePubKey)
	fmt.Printf("pubKeyBytes: %#v\n", pubKeyBytes)
}

func TestNewValidatorSet(t *testing.T) {
	validatorsTestnet, err := NewFixedValidators(TestnetBNBChainID)
	fmt.Printf("ERR: %v\n", err);
	fmt.Printf("validatorsTestnet %+v\n", validatorsTestnet)
}
