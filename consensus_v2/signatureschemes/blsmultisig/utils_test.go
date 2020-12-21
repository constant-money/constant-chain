package blsmultisig

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
)

func TestI2Bytes(t *testing.T) {
	type args struct {
		bn     *big.Int
		length int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// {
		// 	name: "Happy case",
		// 	args: args{
		// 		bn:     bn256.Order,
		// 		length: 32,
		// 	},
		// 	want: []byte{143, 181, 1, 227, 74, 163, 135, 249, 170, 111, 236, 184, 97, 132, 220, 33, 46, 141, 142, 18, 248, 43, 57, 36, 26, 46, 244, 91, 87, 172, 114, 97},
		// },
		{
			name: "Case 1 byte",
			args: args{
				bn:     big.NewInt(1),
				length: 32,
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := I2Bytes(tt.args.bn, tt.args.length); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("I2Bytes() = %v, want %v", got, tt.want)
			} else {
				t.Logf("I2Bytes(%v) = %v", tt.args.bn.Bytes(), got)
			}
		})
	}
}

func Test_printBit(t *testing.T) {
	type args struct {
		bn *big.Int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "bn256.P",
			args: args{
				bn: bn256.P,
			},
		},
		{
			name: "bn256.Order",
			args: args{
				bn: bn256.Order,
			},
		},
		{
			name: "1",
			args: args{
				bn: big.NewInt(1),
			},
		},
	}
	fmt.Println(bn256.P.Cmp(bn256.Order))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printBit(tt.args.bn)
		})
	}
}
