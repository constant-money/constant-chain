package serialnumberprivacy

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/utils"
	"github.com/stretchr/testify/assert"
)

func TestPKSNPrivacy(t *testing.T) {
	for i := 0; i < 1000; i++ {
		sk := key.GeneratePrivateKey(common.RandBytes(31))
		skScalar := new(operation.Scalar).FromBytesS(sk)
		if skScalar.ScalarValid() == false {
			fmt.Println("Invalid scala key value")
		}

		SND := operation.RandomScalar()
		rSK := operation.RandomScalar()
		rSND := operation.RandomScalar()

		serialNumber := new(operation.Point).Derive(operation.PedCom.G[operation.PedersenPrivateKeyIndex], skScalar, SND)
		comSK := operation.PedCom.CommitAtIndex(skScalar, rSK, operation.PedersenPrivateKeyIndex)
		comSND := operation.PedCom.CommitAtIndex(SND, rSND, operation.PedersenSndIndex)

		stmt := new(SerialNumberPrivacyStatement)
		stmt.Set(serialNumber, comSK, comSND)

		witness := new(SNPrivacyWitness)
		witness.Set(stmt, skScalar, rSK, SND, rSND)

		// proving
		start := time.Now()
		proof, err := witness.Prove(nil)
		assert.Equal(t, nil, err)

		end := time.Since(start)
		fmt.Printf("Serial number proving time: %v\n", end)

		//validate sanity proof
		isValidSanity := proof.ValidateSanity()
		assert.Equal(t, true, isValidSanity)

		// convert proof to bytes array
		proofBytes := proof.Bytes()
		assert.Equal(t, utils.SnPrivacyProofSize, len(proofBytes))

		// new SNPrivacyProof to set bytes array
		proof2 := new(SNPrivacyProof).Init()
		err = proof2.SetBytes(proofBytes)
		assert.Equal(t, nil, err)
		assert.Equal(t, proof, proof2)

		start = time.Now()
		res, err := proof2.Verify(nil)
		end = time.Since(start)
		fmt.Printf("Serial number verification time: %v\n", end)
		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)
	}
}
