package bridgesig

import (
	"fmt"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/common"
)

var listPKsBytes [][]byte
var listSKsBytes [][]byte

func genKey(seed []byte, size int) error {
	internalseed := seed
	listPKsBytes = make([][]byte, size)
	listSKsBytes = make([][]byte, size)
	for i := 0; i < size; i++ {
		sk, pk := KeyGen(internalseed)
		listSKsBytes[i] = SKBytes(&sk)
		listPKsBytes[i] = PKBytes(&pk)
		internalseed = common.HashB(append(seed, append(listSKsBytes[i], listPKsBytes[i]...)...))
	}

	return nil
}

func Test_flowECDSASignVerify(t *testing.T) {
	size := 10
	err := genKey([]byte{0, 1, 2, 3, 4}, size)
	if err != nil {
		return
	}
	// return
	data := []byte{1, 2, 3, 4}
	sigs := make([][]byte, size)
	for i := 0; i < size; i++ {
		sigs[i], err = Sign(listSKsBytes[i], data)
	}
	for i := 0; i < size; i++ {
		start := time.Now()
		res, err := Verify(listPKsBytes[i], data, sigs[i])
		t := time.Now().Sub(start)
		fmt.Println(res, err, t.Seconds()*1000)
	}
}
