package blsmultisig

import (
	"errors"
	"reflect"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
)

// genKeyAndGetBytes take a number of for-loop for test flow KeyGen -> Bytes -> Check Key-bytes length
// the first loop take seed as input of KeyGen, after that, seed is a random bytes
func genKeyAndGetBytes(seed []byte, loop int) ([]byte, error) {
	internalseed := seed
	for i := 0; i < loop; i++ {
		sk, pk := KeyGen(internalseed)
		skBytes := SKBytes(sk)
		pkBytes := PKBytes(pk)
		if (len(skBytes) != CSKSz) || (len(pkBytes) != CPKSz) {
			return internalseed, errors.New(CErr + CErrInps)
		}
		internalseed = common.HashB(append(seed, append(skBytes, pkBytes...)...))
	}
	return []byte{0}, nil
}

func Test_genKeyAndGetBytes(t *testing.T) {
	type args struct {
		seed []byte
		loop int
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Test with 10000 loop, start with [3, 4, 5, 6]",
			args: args{
				seed: []byte{3, 4, 5, 6},
				loop: 1000,
			},
			want:    []byte{0},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genKeyAndGetBytes(tt.args.seed, tt.args.loop)
			if (err != nil) != tt.wantErr {
				t.Errorf("genKeyAndGetBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genKeyAndGetBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
