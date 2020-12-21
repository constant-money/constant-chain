package utils

import (
	"encoding/json"
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

func NewCoinUniqueOTABasedOnPaymentInfo(paymentInfo *privacy.PaymentInfo, tokenID *common.Hash, stateDB *statedb.StateDB) (*privacy.CoinV2, error) {
	for {
		c, err := privacy.NewCoinFromPaymentInfo(paymentInfo)
		if err != nil {
			Logger.Log.Errorf("Cannot parse coin based on payment info err: %v", err)
			return nil, err
		}
		// If previously created coin is burning address
		if wallet.IsPublicKeyBurningAddress(c.GetPublicKey().ToBytesS()) {
			return c, nil // No need to check db
		}
		// Onetimeaddress should be unique
		publicKeyBytes := c.GetPublicKey().ToBytesS()
		found, err := statedb.HasOnetimeAddress(stateDB, *tokenID, publicKeyBytes)
		if err != nil {
			Logger.Log.Errorf("Cannot check public key existence in DB, err %v", err)
			return nil, err
		}
		if !found {
			return c, nil
		}
	}
}

func NewCoinV2ArrayFromPaymentInfoArray(paymentInfo []*privacy.PaymentInfo, tokenID *common.Hash, stateDB *statedb.StateDB) ([]*privacy.CoinV2, error) {
	outputCoins := make([]*privacy.CoinV2, len(paymentInfo))
	for index, info := range paymentInfo {
		var err error
		outputCoins[index], err = NewCoinUniqueOTABasedOnPaymentInfo(info, tokenID, stateDB)
		if err != nil {
			Logger.Log.Errorf("Cannot create coin with unique OTA, error: %v", err)
			return nil, err
		}
	}
	return outputCoins, nil
}

func ParseProof(p interface{}, ver int8, txType string) (privacy.Proof, error) {
	// If transaction is nonPrivacyNonInput then we do not have proof, so parse it as nil
	if p == nil {
		return nil, nil
	}

	proofInBytes, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	if string(proofInBytes)=="null"{
		return nil, nil
	}


	var res privacy.Proof
	switch txType {
	case common.TxConversionType:
		if ver == TxConversionVersion12Number {
			res = new(privacy.ProofForConversion)
			res.Init()
		} else {
			return nil, errors.New("ParseProof: TxConversion version is incorrect")
		}
	default:
		switch ver {
		case TxVersion1Number, TxVersion0Number:
			res = new(privacy.ProofV1)
		case TxVersion2Number:
			res = new(privacy.ProofV2)
			res.Init()
		default:
			return nil, errors.New("ParseProof: Tx.Version is incorrect")
		}
	}

	err = json.Unmarshal(proofInBytes, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}