package blsmultisig

import (
	"crypto/rand"
	"errors"
	"reflect"
	"testing"

	"fmt"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
)

func cmptPnG1(oddPoint byte, loop int) (*bn256.G1, error) {
	tests := make([]*bn256.G1, loop)
	var err error
	for i := 0; i < loop; i++ {
		_, tests[i], err = bn256.RandomG1(rand.Reader)
		if err != nil {
			return tests[i], err
		}
		for ; tests[i].Marshal()[63]&1 != oddPoint; _, tests[i], _ = bn256.RandomG1(rand.Reader) {
		}
		cmprBytesArr := CmprG1(tests[i])
		pnDeCmpr, err := DecmprG1(cmprBytesArr)
		if err != nil {
			return tests[i], err
		}
		if !reflect.DeepEqual(pnDeCmpr.Marshal(), tests[i].Marshal()) {
			return tests[i], errors.New("Not equal")
		}
	}
	return nil, nil
}

func Test_cmptPnG1(t *testing.T) {
	type args struct {
		oddPoint byte
		loop     int
	}
	tests := []struct {
		name    string
		args    args
		want    *bn256.G1
		wantErr bool
	}{
		{
			name: "Test with 1000 odd point",
			args: args{
				oddPoint: 1,
				loop:     1000,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test with 1000 even point",
			args: args{
				oddPoint: 0,
				loop:     1000,
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cmptPnG1(tt.args.oddPoint, tt.args.loop)
			if (err != nil) != tt.wantErr {
				t.Errorf("cmptPnG1() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cmptPnG1() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func cmptPnG2(oddPoint byte, loop int) (*bn256.G2, error) {
// 	tests := make([]*bn256.G2, loop)
// 	var err error
// 	for i := 0; i < loop; i++ {
// 		_, tests[i], err = bn256.RandomG2(rand.Reader)
// 		if err != nil {
// 			return tests[i], err
// 		}
// 		fmt.Println(tests[i].Marshal())
// 		for ; tests[i].Marshal()[63]&1 != oddPoint; _, tests[i], _ = bn256.RandomG2(rand.Reader) {
// 		}
// 		cmprBytesArr := CmprG2(tests[i])
// 		pnDeCmpr, err := DecmprG2(cmprBytesArr)
// 		if err != nil {
// 			return tests[i], err
// 		}
// 		if !reflect.DeepEqual(pnDeCmpr.Marshal(), tests[i].Marshal()) {
// 			return tests[i], errors.New("Not equal")
// 		}
// 	}
// 	return nil, nil
// }

// func Test_cmptPnG2(t *testing.T) {
// 	type args struct {
// 		oddPoint byte
// 		loop     int
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		want    *bn256.G1
// 		wantErr bool
// 	}{
// 		{
// 			name: "Test with 10000 odd point",
// 			args: args{
// 				oddPoint: 1,
// 				loop:     10000,
// 			},
// 			want:    nil,
// 			wantErr: false,
// 		},
// 		{
// 			name: "Test with 10000 even point",
// 			args: args{
// 				oddPoint: 0,
// 				loop:     10000,
// 			},
// 			want:    nil,
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := cmptPnG2(tt.args.oddPoint, tt.args.loop)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("cmptPnG2() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("cmptPnG2() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func TestBn256G2(t *testing.T) {
	sk, pk, _ := bn256.RandomG2(rand.Reader)

	pkCompressed := CmprG2(pk)
	fmt.Printf("sk: %v\n", sk)
	fmt.Printf("pkCompressed: %v\n", pkCompressed)
}
