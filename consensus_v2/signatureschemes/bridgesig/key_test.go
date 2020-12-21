package bridgesig

import (
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestKeyGen(t *testing.T) {
	type args struct {
		seed []byte
	}
	tests := []struct {
		name  string
		args  args
		want  ecdsa.PrivateKey
		want1 ecdsa.PublicKey
	}{
		// TODO: Add test cases.
	}
	x, y := KeyGen([]byte{0, 1, 2, 3, 4})
	xBytes := SKBytes(&x)
	yBytes := PKBytes(&y)
	fmt.Println(hexutil.Encode(xBytes))
	fmt.Println(hexutil.Encode(yBytes))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := KeyGen(tt.args.seed)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KeyGen() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("KeyGen() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
