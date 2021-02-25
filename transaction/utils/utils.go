package utils

import (
	"encoding/json"
	"fmt"

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

func ParseProof(p interface{}, ver int8) (privacy.Proof, error) {
	// If transaction is nonPrivacyNonInput then we do not have proof, so parse it as nil
	if p == nil {
		return nil, nil
	}

	Logger.Log.Infof("Parsing proof: %v\n", p)

	proofInBytes, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	if string(proofInBytes)=="null"{
		return nil, nil
	}

	var res privacy.Proof
	switch ver {
	case 1:
		res = new(privacy.ProofV1)
		err = json.Unmarshal(proofInBytes, res)
		if err != nil {
			Logger.Log.Errorf("cannot parse to a ProofV1: %v\n", err)

			res = new(privacy.ConversionProof)
			err = json.Unmarshal(proofInBytes, res)
			if err != nil {
				Logger.Log.Errorf("cannot parse to a ConversionProof: %v\n", err)
				return nil, err
			}
		}
	case 2:
		res = new(privacy.ProofV2)
		res.Init()
		err = json.Unmarshal(proofInBytes, res)
		if err != nil {
			Logger.Log.Errorf("cannot parse to a ProofV2: %v\n", err)
			return nil, err
		}
	default:
		Logger.Log.Errorf("proof version %v not valid\n", ver)
		return nil, fmt.Errorf("proof version %v not valid\n", ver)
	}

	return res, nil
}