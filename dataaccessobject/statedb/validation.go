package statedb

import (
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/privacy"
)

type StateObjectValidation struct {
	version int
}

func (v StateObjectValidation) Version() int {
	return v.version
}

func (v *StateObjectValidation) SetVersion(version int) {
	v.version = version
}

var SoValidation *StateObjectValidation

func init() {
	SoValidation = &StateObjectValidation{version: defaultVersion}
	return
}

func (s StateObjectValidation) ValidatePaymentAddressSanity(v privacy.PaymentAddress) error {
	if len(v.Pk) == 0 || len(v.Tk) == 0 {
		return fmt.Errorf("length public key %+v, length transmission key %+v", len(v.Pk), len(v.Tk))
	}
	return nil
}

func (s StateObjectValidation) ValidateIncognitoPublicKeySanity(v string) error {
	res, ver, err := base58.Base58Check{}.Decode(v)
	if err != nil {
		return err
	}
	if ver != common.Base58Version {
		return fmt.Errorf("want version %+v got version %+v", common.Base58Version, ver)
	}
	if len(res) != 32 {
		return fmt.Errorf("length public key %+v", len(res))
	}
	return nil
}
