package mlsag

import (
	"testing"
	"encoding/json"
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/stretchr/testify/assert"
	"crypto/rand"
)

// TEST DURATION NOTE : 1000 iterations = 190sec

var (
	maxPrivateKeys = 15
	minPrivateKeys = 1
	maxTotalLayers  = 10
	minTotalLayers  = 1
	numOfLoops     = 1000
)

var (
	signatureStrings = []string{"\"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\""}
)

func TestWorkflowMlsag(t *testing.T) {
	message := make([]byte,32)
	
	for loopCount:=0;loopCount<=numOfLoops;loopCount++{
		keyInputs := []*operation.Scalar{}

		// ring params : #private keys, #fake layers, pi
		// are picked randomly in their domain
		numOfPrivateKeys := common.RandInt() % (maxPrivateKeys-minPrivateKeys+1) + minPrivateKeys
		for i := 0; i < numOfPrivateKeys; i += 1 {
			privateKey := operation.RandomScalar()
			keyInputs = append(keyInputs, privateKey)
		}
		numOfLayers := common.RandInt() % (maxTotalLayers-minTotalLayers+1) + minTotalLayers
		pi := common.RandInt() % numOfLayers
		ring := NewRandomRing(keyInputs, numOfLayers, pi)
		signer := NewMlsag(keyInputs, ring, pi)

		// take a random 20-byte message
		rand.Read(message)


		s := common.HashH(message)
		signature, err := signer.Sign(s[:])
		assert.Equal(t, nil, err, "There should not be any error when sign")

		s2 := common.HashH(message)
		check, err := Verify(signature, ring, s2[:])
		assert.Equal(t, nil, err, "There should not be any error when verify")
		assert.Equal(t, true, check, "It should verify correctly")
	}
}

func TestDumpSig(t *testing.T){
	for _, sigStr := range signatureStrings{
		var sigByteHolder []byte = make([]byte, 100)
		err := json.Unmarshal([]byte(sigStr), &sigByteHolder)
		if err!=nil{
			panic(err)
		}
		sig := &MlsagSig{}
		_, err = sig.FromBytes(sigByteHolder)
		fmt.Printf("\n\nBegin Signature\n")
		fmt.Printf("  %x\n--Key Images\n", sig.c.ToBytesS())
		for _, k := range sig.keyImages{
			fmt.Printf("  %x\n", k.ToBytesS())
		}
		fmt.Printf("\n--R Matrix\n")
		for _, row := range sig.r{
			fmt.Printf("\n  --Row--\n")
			for _, ele := range row{
				fmt.Printf("    %x\n", ele.ToBytesS())
			}
		}
		fmt.Printf("End Signature\n")
	}
}