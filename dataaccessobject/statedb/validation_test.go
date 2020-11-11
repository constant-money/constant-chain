package statedb

import (
	"testing"
)

func TestValidation_ValidatePaymentAddressSanity(t *testing.T) {
	str1 := receiverPaymentAddressStructs[0]

	err := SoValidation.ValidatePaymentAddressSanity(str1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidation_ValidateIncognitoPublicKeySanity(t *testing.T) {
	str1 := incognitoPublicKeys[0]
	str2 := str1[1:]
	str3 := str1[2:]
	str4 := str1[:len(str1)-1]
	str5 := str1[:len(str1)-2]
	err := SoValidation.ValidateIncognitoPublicKeySanity(str1)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range incognitoPublicKeys[1:] {
		err := SoValidation.ValidateIncognitoPublicKeySanity(v)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = SoValidation.ValidateIncognitoPublicKeySanity(str2)
	if err == nil {
		t.Fatal(err)
	}
	err = SoValidation.ValidateIncognitoPublicKeySanity(str3)
	if err == nil {
		t.Fatal(err)
	}
	err = SoValidation.ValidateIncognitoPublicKeySanity(str4)
	if err == nil {
		t.Fatal(err)
	}
	err = SoValidation.ValidateIncognitoPublicKeySanity(str5)
	if err == nil {
		t.Fatal(err)
	}
}
