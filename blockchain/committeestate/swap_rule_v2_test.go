package committeestate

import (
	"github.com/incognitochain/incognito-chain/instruction"
	"reflect"
	"testing"
)

func Test_createSwapShardInstructionV2(t *testing.T) {

	initLog()
	initPublicKey()

	type args struct {
		shardID                byte
		substitutes            []string
		committees             []string
		maxCommitteeSize       int
		minCommitteeSize       int
		typeIns                int
		numberOfFixedValidator int
	}
	tests := []struct {
		name  string
		args  args
		want  *instruction.SwapShardInstruction
		want1 []string
		want2 []string
		want3 []string
	}{
		{
			name: "len(subtitutes) == len(committeess) == 0",
			args: args{
				shardID:                0,
				substitutes:            []string{},
				committees:             []string{},
				maxCommitteeSize:       6,
				typeIns:                instruction.SWAP_BY_END_EPOCH,
				numberOfFixedValidator: 0,
			},
			want:  instruction.NewSwapShardInstruction(),
			want1: []string{},
			want2: []string{},
			want3: []string{},
		},
		{
			name: "swapOffset == 1, currentSize > minSize, no vacant slot, no fixed node",
			args: args{
				shardID:                0,
				substitutes:            []string{key5, key6},
				committees:             []string{key, key2, key3, key4},
				maxCommitteeSize:       4,
				minCommitteeSize:       0,
				typeIns:                instruction.SWAP_BY_END_EPOCH,
				numberOfFixedValidator: 0,
			},
			want: instruction.NewSwapShardInstructionWithValue(
				[]string{key5},
				[]string{key},
				int(0),
				instruction.SWAP_BY_END_EPOCH),
			want1: []string{key2, key3, key4, key5},
			want2: []string{key6},
			want3: []string{key},
		},
		{
			name: "swapOffset == 1, currentSize > minSize, one vacant slot, no fixed node",
			args: args{
				shardID:                0,
				substitutes:            []string{key5, key6},
				committees:             []string{key, key2, key3, key4},
				maxCommitteeSize:       5,
				minCommitteeSize:       0,
				typeIns:                instruction.SWAP_BY_END_EPOCH,
				numberOfFixedValidator: 0,
			},
			want: instruction.NewSwapShardInstructionWithValue(
				[]string{key5, key6},
				[]string{key},
				int(0),
				instruction.SWAP_BY_END_EPOCH),
			want1: []string{key2, key3, key4, key5, key6},
			want2: []string{},
			want3: []string{key},
		},
		//{
		//	name: "int((len(committees) + len(subtitutes)) / 3) > maxCommitteeSize && <= len(substitute)",
		//	args: args{
		//		shardID:                0,
		//		substitutes:            []string{key5, key6, key7, key8, key9, key10, key11, key12},
		//		committees:             []string{key, key2, key3, key4},
		//		maxCommitteeSize:       4,
		//		typeIns:                instruction.SWAP_BY_END_EPOCH,
		//		numberOfFixedValidator: 0,
		//	},
		//	want: instruction.NewSwapShardInstructionWithValue(
		//		[]string{key5, key6, key7, key8},
		//		[]string{key, key2, key3, key4},
		//		int(0),
		//		instruction.SWAP_BY_END_EPOCH),
		//	want1: []string{
		//		key9, key10, key11, key12,
		//	},
		//	want2: []string{},
		//	want3: []string{},
		//},
		//{
		//	name: "int((len(committees) + len(subtitutes)) / 3) < maxCommitteeSize && > len(substitute), with NO vacant slot",
		//	args: args{
		//		shardID:                0,
		//		substitutes:            []string{key10, key11, key12},
		//		committees:             []string{key, key2, key3, key4, key5, key6, key7, key8, key9},
		//		maxCommitteeSize:       9,
		//		typeIns:                instruction.SWAP_BY_END_EPOCH,
		//		numberOfFixedValidator: 0,
		//	},
		//	want: instruction.NewSwapShardInstructionWithValue(
		//		[]string{key10, key11, key12},
		//		[]string{key, key2, key3},
		//		int(0),
		//		instruction.SWAP_BY_END_EPOCH),
		//	want1: []string{},
		//	want2: []string{},
		//	want3: []string{},
		//},
		//{
		//	name: "int((len(committees) + len(subtitutes)) / 3) < maxCommitteeSize && > len(substitute), with vacant slot",
		//	args: args{
		//		shardID:                0,
		//		substitutes:            []string{key10, key11, key12},
		//		committees:             []string{key, key2, key3, key4, key5, key6, key7, key8, key9},
		//		maxCommitteeSize:       11,
		//		typeIns:                instruction.SWAP_BY_END_EPOCH,
		//		numberOfFixedValidator: 0,
		//	},
		//	want: instruction.NewSwapShardInstructionWithValue(
		//		[]string{key10, key11, key12},
		//		[]string{key},
		//		int(0),
		//		instruction.SWAP_BY_END_EPOCH),
		//	want1: []string{},
		//	want2: []string{},
		//	want3: []string{},
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3 := createSwapShardInstructionV2(tt.args.shardID, tt.args.substitutes, tt.args.committees, tt.args.minCommitteeSize, tt.args.maxCommitteeSize, tt.args.typeIns, tt.args.numberOfFixedValidator)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createSwapShardInstructionV2() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("createSwapShardInstructionV2() got1 = %v, want1 %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("createSwapShardInstructionV2() got2 = %v, want2 %v", got2, tt.want3)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("createSwapShardInstructionV2() got3 = %v, want3 %v", got3, tt.want3)
			}
		})
	}
}

func Test_sortShardIDByIncreaseOrder(t *testing.T) {
	type args struct {
		arr []int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "testcase 1",
			args: args{
				[]int{15, 15, 3, 30},
			},
			want: []byte{2, 0, 1, 3},
		},
		{
			name: "testcase 2",
			args: args{
				[]int{1, 15, 3, 30},
			},
			want: []byte{0, 2, 1, 3},
		},
		{
			name: "testcase 3",
			args: args{
				[]int{30, 15, 3, 30},
			},
			want: []byte{2, 1, 0, 3},
		},
		{
			name: "testcase 4",
			args: args{
				[]int{30, 15, 45, 20},
			},
			want: []byte{1, 3, 0, 2},
		},
		{
			name: "testcase 5",
			args: args{
				[]int{190, 542, 208, 18, 674, 817, 808, 112},
			},
			want: []byte{3, 7, 0, 2, 1, 4, 6, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sortShardIDByIncreaseOrder(tt.args.arr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortShardIDByIncreaseOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

// [19,54,20,2,67,81,80,11]
func Test_calculateCandidatePosition(t *testing.T) {
	type args struct {
		candidate string
		rand      int64
		total     int
	}
	tests := []struct {
		name    string
		args    args
		wantPos int
	}{
		{
			name: "testcase 1",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGk4KtMcHSGEy6q7Ad5NPjGKakZoNowXd5xokQ3GYNSqLkmkicDFMRzHYk2qvUKfs6PbHFrLjnQNwQwX9inAzeBZdDeyRDNrPymyAwYkb5UDvvsqhx9fWF7Bm3TBYsZ5fKGLe9c5sok2HgKfZ8MUHyxXvYsmoAa4gPwECUULHXDFkh85XtMxEavYda1PMZCXr9fg9e6jV68RaRrNmodnJ77L7zcE9Dev6YAwPpSe3RpfmQ8Dj4tzhuiRuZiD4h1VEkDmhbuExWruL6VTaNpxRBkAhXgiktUS91WcXNq9CQPe793mvxedpJbyLsU5YsCoqw3bch5TUEoR1p9xD7fbzF6PmG868Cx9CJD73R2XFqFvijsLUnpoTVrZPfG9D6jVpCd1AxDGJv74FCWPQhm6xD7sUaRmpD",
				rand:      10000,
				total:     333,
			},
			wantPos: 39,
		},
		{
			name: "testcase 2",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGk4vSqVyeGkELuK9Zz94N2CjypNGdtskQKMFsseWJv377rYY5NGtTqnPq1PaNkGygmPeXdrjLjmCMcRJCWDRe7ie28Y6nm69a96d9JnDDYCvLsUbMMjmfgaPrZMaG1YmruauaeVqAhW8ahrubCtdAGJM9Bbb4yE4Lh4NBmWmQmsFSDJVQDmBTNE6M7ZuvcgQB3o3cMXGPuFpb7CpWBHvvjG15scXDLckgkjmzCgP8DGr72Y82uxeL2YULpToyjQuijYmY1sdHaAT5jdq8wuADYsMg5AthVpZRwNkdECtpVFey55VsG9mG283RwpQMyebqWASJvJPjpwQjTLQJrMcPdYjbyj6UFZFLoHBbj16A2a8awfEeqegR7TzUxMfnPNsBBfBTjEXZG56GFYLpzM1b885D4kkw",
				rand:      10000,
				total:     333,
			},
			wantPos: 52,
		},
		{
			name: "testcase 3",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGkQYyq7HFXd2y35K4p35YyNgoXZENZrgnhVKxLAzznRbDkLFvB6JAHPGg5vNfvesxU8xdKbDvi6ptt2UhU6BbvMEyuDg7ntDsH23pzs6cLbZALgfSFayfF6KvounPNRMkJ2piWYd8k7oXgC5sMhC4PcB6QxxKCW1een7KZKVHNpQQooCVUkTNiSuy25boa2Q3qrnEL6R9MWBykcm6ET14C7JyrrKXEz7oVaub2H3M8ByKCaib4ccGT9PungUPD8hvywNcKYYLXgM1SSt9kZKdhDBttUKa4X4PkmM2Ew9bzZLHQyESciGtgVv3yWsyQeGHMa7zcSrbFNYXdh5GRdqNJ4JwwpZyJvErYT5hy6HCDdBtjzVWLqFWQspGK72nhoJvfGjPMXYLYcvxvhMWA1uc36M4DbDD",
				rand:      10000,
				total:     333,
			},
			wantPos: 116,
		},
		{
			name: "testcase 4",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGk6wTR2KdLEEADpvc7pzYoxaqk4FMh2iHJRopTUaPwhMhGmndM92rZBNtXQfB2PeRo1YSGKHxRByvH92jF6FPqZLYaJPRoWJitWRsf7r3ReXf3Qmb7WwiFpSbNDQvVWy2HqqdVLhEaYaRPbqzVjLJ5c7fcc32FaPcknX1BUz7L6nWz7enaEMQPzt5LxCdu4NkHWwoNBADF938v8S9dztELMoPQiaruboEsiegjVL1PK9iQqpkT68RWJofnzqAS9nGkwn8jHmf2aDbxHJkYwWAsRfXTtTuJMK6K7kNakgssTNUzTKWs2sviYp1tkseUT26kbj4BXi2icnajCeMJDXMKD5YufvHsgJ4pZvNJvDobi7YQf8iweCCSwkABRqDQPY9qqmPzWugc2jLkEAc5iVcdHbpxZQU",
				rand:      10000,
				total:     333,
			},
			wantPos: 185,
		},
		{
			name: "testcase 5",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGkPeatBbQ3kUFg2EqpopZ1MkV3nTUz3KiNTQqaqZKMGn28FmxUYNJ3KFQAA2ocg3psnREazqFGcRQGd8pR7HhvYXrwqcx8BABj2WA7Qyqj5DLGj419GR2GLjueTsTKgtve3voRyj9EhErBCHKZNc1VVtFmfqzGk6kMKwLCcjv3yuSaxdx9k9odgyYcoiAcJwzWanj8r2oPKJK5FDNjLuQ8xRF8gktucz5VB84iTY9DRZ9ua8Wn6RRDR9U5i9gg69Wc4g5pZPv7mc7PZZGLakf941HB4FMKxQqiJLR6imZHyLhHWnMsN17aA5T7JmxH8UdeZXLNj73Komy8pQCGzfKXkn8uVwhwvwvxAxczShKKEAACfMtEAnfsirv6Gi2VL9AtFYq5Jx1vsfB3HBpAxp9xFjV8oHG",
				rand:      10000,
				total:     333,
			},
			wantPos: 114,
		},
		{
			name: "testcase 6",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGkPyc9JTWSLSmivsQGCgeD8vxTbTegwvLCREXrsywGwsgVMqtdYxmsknXmiAw16TAZhRsJ4DXrFiPhjVkt73VvjK1Q1cjcxjA2BkW4NHtAYSeBVkcUuk5einnjbevayfMEQ8WdGZfKMutVA5AMEammuUhC8BybH7o7BnWg43JqmqvaQXAXuFbYTbK1WCVuE9Lpgddv5dv6hpz7Yp8AGp3v2yn1PTrwFDxWvLfD7sL7qj42c7iZq4gZkcbf5CgyJ438eZnbf6g9vUCnKJLhMx9dhbZhZnAV1cbbo7BEJySw2kEQcVma5gnoYBbKtoJ5xRDQRZTwMk3g1a5eJ2u69Ripmv5vA1Cpt1Q9emQiDaw1VMVXHSbiYgEgCcNtZcsmxqYYYFGL8ZLZjL9tck4N4LFziGa6oEB",
				rand:      10000,
				total:     333,
			},
			wantPos: 322,
		},
		{
			name: "testcase 7",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGkBN1bTLWLWx35tLscRLVb77bFkeSLx911CiKs39cR9pA9YsDjrFEbRys9bNEFY8TesFDX3W89M5PuzyVwLgZm51KqSFpYxCXTnJnT9RkT5qr2KfkjbhgpfvvkLJV2YHwyPTbmKnbHcYXLLGzJeE8TogpZDDg38TckC3YR4xXezKaUR2thAfZDwnnSutrprKSkM6aDUP7SeYmqcEUYLN8HmF2wjcstPFfHu2hEY8PLYSbmMYbtPDp5sJnEQHHyfftRZJneaEJci9KiTuBNPfswj3LsKmDAmCZ5zqRkRpYjGyKYDhTWevyRvbf9tZskpfG7tR23VMYoLr5bEXwxUvdSsPpEWAs2xbHAazUk7MytBrVrgbRReeANFZdzRhPacNsgCRPvBzHAeL2eDMrfzH4XYqmfAha",
				rand:      10000,
				total:     333,
			},
			wantPos: 204,
		},
		{
			name: "testcase 8",
			args: args{
				candidate: "121VhftSAygpEJZ6i9jGkBMpYsJSyYtwUxuUPwfNBKqC44vmE4WsqRaJpvSFNZ6S2TDptppCLzZAc6zDxMBnaLaCxuraVhu1tAjqML9cgume5RmE1DviSeD8ZosA7e2Pomn1ijMexkqREiyjFZ6fcMJVafYHeLGM5nxpaJEhr4SRx78YKwxCBwSBTUFB5iE7fXxekhfQQTVgcNBeJE1Zjh7sVYkkS5FkKY5H8q4NHTVMf99DwnqCCFpURLr3qPyrwN3SPHkLV2AbVuA1PYsh2L3mZvmzSrm88phFYhTVgWdfAqwim7CuLx5shj4rvir1qFpqcyrEX3z4276k2XTjcJ1CQsv6vj8vHN4YLTGCpJx6ky2wk74rP32PKHwhQohnUwi6UAgmL1qmWDhpe6ZEjopdseLgheZnoQXLe9cwvtLHq55t",
				rand:      10000,
				total:     333,
			},
			wantPos: 276,
		},
		{
			name: "Temporary",
			args: args{
				candidate: key7,
				rand:      1250000,
				total:     265,
			},
			wantPos: 182,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotPos := calculateCandidatePosition(tt.args.candidate, tt.args.rand, tt.args.total); gotPos != tt.wantPos {
				t.Errorf("calculateCandidatePosition() = %v, want %v", gotPos, tt.wantPos)
			}
		})
	}
}

func Test_assignShardCandidateV2(t *testing.T) {

	initPublicKey()

	type args struct {
		candidates         []string
		numberOfValidators []int
		rand               int64
	}
	tests := []struct {
		name string
		args args
		want map[byte][]string
	}{
		{
			name: "testcase 1",
			args: args{
				candidates: []string{
					"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",
				},
				numberOfValidators: []int{19, 54, 20, 2, 67, 81, 80, 11},
				rand:               10000,
			},
			want: map[byte][]string{
				0: {
					"121VhftSAygpEJZ6i9jGkQYyq7HFXd2y35K4p35YyNgoXZENZrgnhVKxLAzznRbDkLFvB6JAHPGg5vNfvesxU8xdKbDvi6ptt2UhU6BbvMEyuDg7ntDsH23pzs6cLbZALgfSFayfF6KvounPNRMkJ2piWYd8k7oXgC5sMhC4PcB6QxxKCW1een7KZKVHNpQQooCVUkTNiSuy25boa2Q3qrnEL6R9MWBykcm6ET14C7JyrrKXEz7oVaub2H3M8ByKCaib4ccGT9PungUPD8hvywNcKYYLXgM1SSt9kZKdhDBttUKa4X4PkmM2Ew9bzZLHQyESciGtgVv3yWsyQeGHMa7zcSrbFNYXdh5GRdqNJ4JwwpZyJvErYT5hy6HCDdBtjzVWLqFWQspGK72nhoJvfGjPMXYLYcvxvhMWA1uc36M4DbDD",
					"121VhftSAygpEJZ6i9jGkPeatBbQ3kUFg2EqpopZ1MkV3nTUz3KiNTQqaqZKMGn28FmxUYNJ3KFQAA2ocg3psnREazqFGcRQGd8pR7HhvYXrwqcx8BABj2WA7Qyqj5DLGj419GR2GLjueTsTKgtve3voRyj9EhErBCHKZNc1VVtFmfqzGk6kMKwLCcjv3yuSaxdx9k9odgyYcoiAcJwzWanj8r2oPKJK5FDNjLuQ8xRF8gktucz5VB84iTY9DRZ9ua8Wn6RRDR9U5i9gg69Wc4g5pZPv7mc7PZZGLakf941HB4FMKxQqiJLR6imZHyLhHWnMsN17aA5T7JmxH8UdeZXLNj73Komy8pQCGzfKXkn8uVwhwvwvxAxczShKKEAACfMtEAnfsirv6Gi2VL9AtFYq5Jx1vsfB3HBpAxp9xFjV8oHG",
				},
				2: {
					"121VhftSAygpEJZ6i9jGk4KtMcHSGEy6q7Ad5NPjGKakZoNowXd5xokQ3GYNSqLkmkicDFMRzHYk2qvUKfs6PbHFrLjnQNwQwX9inAzeBZdDeyRDNrPymyAwYkb5UDvvsqhx9fWF7Bm3TBYsZ5fKGLe9c5sok2HgKfZ8MUHyxXvYsmoAa4gPwECUULHXDFkh85XtMxEavYda1PMZCXr9fg9e6jV68RaRrNmodnJ77L7zcE9Dev6YAwPpSe3RpfmQ8Dj4tzhuiRuZiD4h1VEkDmhbuExWruL6VTaNpxRBkAhXgiktUS91WcXNq9CQPe793mvxedpJbyLsU5YsCoqw3bch5TUEoR1p9xD7fbzF6PmG868Cx9CJD73R2XFqFvijsLUnpoTVrZPfG9D6jVpCd1AxDGJv74FCWPQhm6xD7sUaRmpD",
					"121VhftSAygpEJZ6i9jGk4vSqVyeGkELuK9Zz94N2CjypNGdtskQKMFsseWJv377rYY5NGtTqnPq1PaNkGygmPeXdrjLjmCMcRJCWDRe7ie28Y6nm69a96d9JnDDYCvLsUbMMjmfgaPrZMaG1YmruauaeVqAhW8ahrubCtdAGJM9Bbb4yE4Lh4NBmWmQmsFSDJVQDmBTNE6M7ZuvcgQB3o3cMXGPuFpb7CpWBHvvjG15scXDLckgkjmzCgP8DGr72Y82uxeL2YULpToyjQuijYmY1sdHaAT5jdq8wuADYsMg5AthVpZRwNkdECtpVFey55VsG9mG283RwpQMyebqWASJvJPjpwQjTLQJrMcPdYjbyj6UFZFLoHBbj16A2a8awfEeqegR7TzUxMfnPNsBBfBTjEXZG56GFYLpzM1b885D4kkw",
				},
				3: {
					"121VhftSAygpEJZ6i9jGk6wTR2KdLEEADpvc7pzYoxaqk4FMh2iHJRopTUaPwhMhGmndM92rZBNtXQfB2PeRo1YSGKHxRByvH92jF6FPqZLYaJPRoWJitWRsf7r3ReXf3Qmb7WwiFpSbNDQvVWy2HqqdVLhEaYaRPbqzVjLJ5c7fcc32FaPcknX1BUz7L6nWz7enaEMQPzt5LxCdu4NkHWwoNBADF938v8S9dztELMoPQiaruboEsiegjVL1PK9iQqpkT68RWJofnzqAS9nGkwn8jHmf2aDbxHJkYwWAsRfXTtTuJMK6K7kNakgssTNUzTKWs2sviYp1tkseUT26kbj4BXi2icnajCeMJDXMKD5YufvHsgJ4pZvNJvDobi7YQf8iweCCSwkABRqDQPY9qqmPzWugc2jLkEAc5iVcdHbpxZQU",
					"121VhftSAygpEJZ6i9jGkBN1bTLWLWx35tLscRLVb77bFkeSLx911CiKs39cR9pA9YsDjrFEbRys9bNEFY8TesFDX3W89M5PuzyVwLgZm51KqSFpYxCXTnJnT9RkT5qr2KfkjbhgpfvvkLJV2YHwyPTbmKnbHcYXLLGzJeE8TogpZDDg38TckC3YR4xXezKaUR2thAfZDwnnSutrprKSkM6aDUP7SeYmqcEUYLN8HmF2wjcstPFfHu2hEY8PLYSbmMYbtPDp5sJnEQHHyfftRZJneaEJci9KiTuBNPfswj3LsKmDAmCZ5zqRkRpYjGyKYDhTWevyRvbf9tZskpfG7tR23VMYoLr5bEXwxUvdSsPpEWAs2xbHAazUk7MytBrVrgbRReeANFZdzRhPacNsgCRPvBzHAeL2eDMrfzH4XYqmfAha",
				},
				7: {
					"121VhftSAygpEJZ6i9jGkPyc9JTWSLSmivsQGCgeD8vxTbTegwvLCREXrsywGwsgVMqtdYxmsknXmiAw16TAZhRsJ4DXrFiPhjVkt73VvjK1Q1cjcxjA2BkW4NHtAYSeBVkcUuk5einnjbevayfMEQ8WdGZfKMutVA5AMEammuUhC8BybH7o7BnWg43JqmqvaQXAXuFbYTbK1WCVuE9Lpgddv5dv6hpz7Yp8AGp3v2yn1PTrwFDxWvLfD7sL7qj42c7iZq4gZkcbf5CgyJ438eZnbf6g9vUCnKJLhMx9dhbZhZnAV1cbbo7BEJySw2kEQcVma5gnoYBbKtoJ5xRDQRZTwMk3g1a5eJ2u69Ripmv5vA1Cpt1Q9emQiDaw1VMVXHSbiYgEgCcNtZcsmxqYYYFGL8ZLZjL9tck4N4LFziGa6oEB",
					"121VhftSAygpEJZ6i9jGkBMpYsJSyYtwUxuUPwfNBKqC44vmE4WsqRaJpvSFNZ6S2TDptppCLzZAc6zDxMBnaLaCxuraVhu1tAjqML9cgume5RmE1DviSeD8ZosA7e2Pomn1ijMexkqREiyjFZ6fcMJVafYHeLGM5nxpaJEhr4SRx78YKwxCBwSBTUFB5iE7fXxekhfQQTVgcNBeJE1Zjh7sVYkkS5FkKY5H8q4NHTVMf99DwnqCCFpURLr3qPyrwN3SPHkLV2AbVuA1PYsh2L3mZvmzSrm88phFYhTVgWdfAqwim7CuLx5shj4rvir1qFpqcyrEX3z4276k2XTjcJ1CQsv6vj8vHN4YLTGCpJx6ky2wk74rP32PKHwhQohnUwi6UAgmL1qmWDhpe6ZEjopdseLgheZnoQXLe9cwvtLHq55t",
				},
			},
		},
		{
			name: "8 Shards 8 Candidates Random Number: [500000 .. 1000000] Current Total Validators: [300 .. 400]",
			args: args{
				candidates: []string{
					key, key2, key3, key4, key5, key6, key7, key8,
				},
				numberOfValidators: []int{
					19, 54, 20, 2, 67, 81, 80, 11,
				},
				rand: 800000,
			},
			want: map[byte][]string{
				0: {
					key, key5,
				},
				1: {
					key8,
				},
				2: {
					key4, key6,
				},
				3: {
					key3,
				},
				4: {
					key2,
				},
				7: {
					key7,
				},
			},
		},
		{
			name: "8 Shards 8 Candidates Random Number: [0 .. 500000] Current Total Validators: [300 .. 400]",
			args: args{
				candidates: []string{
					key, key2, key3, key4, key5, key6, key7, key8,
				},
				numberOfValidators: []int{
					19, 54, 20, 2, 67, 81, 80, 11,
				},
				rand: 100000,
			},
			want: map[byte][]string{
				0: {
					key,
					key2,
					key8,
				},
				7: {
					key3, key4, key5, key6,
				},
				4: {
					key7,
				},
			},
		},
		{
			name: "8 Shards 8 Candidates Random Number: [1000000 .. 2000000] Current Total Validators: [300 .. 400]",
			args: args{
				candidates: []string{
					key, key2, key3, key4, key5, key6, key7, key8,
				},
				numberOfValidators: []int{
					19, 54, 20, 2, 67, 81, 80, 11,
				},
				rand: 1250000,
			},
			want: map[byte][]string{
				0: {
					key4,
					key6,
					key8,
				},
				2: {
					key3,
				},
				3: {
					key,
				},
				7: {
					key2,
					key5, key7,
				},
			},
		},
		{
			name: "8 Shards 8 Candidates Random Number: [500000 .. 1000000] Current Total Validators: [200 .. 300]",
			args: args{
				candidates: []string{
					key, key2, key3, key4, key5, key6, key7, key8,
				},
				numberOfValidators: []int{
					50, 33, 29, 47, 15, 2, 25, 64,
				},
				rand: 800000,
			},
			want: map[byte][]string{
				0: {
					key6,
				},
				1: {
					key2, key3, key4,
				},
				4: {
					key, key7,
				},
				5: {
					key5,
				},
				6: {
					key8,
				},
			},
		},
		{
			name: "8 Shards 8 Candidates Random Number: [0 .. 500000] Current Total Validators: [200 .. 300]",
			args: args{
				candidates: []string{
					key, key2, key3, key4, key5, key6, key7, key8,
				},
				numberOfValidators: []int{
					50, 33, 29, 47, 15, 2, 25, 64,
				},
				rand: 100000,
			},
			want: map[byte][]string{
				0: {
					key4,
				},
				2: {
					key,
				},
				5: {
					key7, key8,
				},
				6: {
					key2, key3, key5, key6,
				},
			},
		},
		{
			name: "8 Shards 8 Candidates Random Number: [1000000 .. 2000000] Current Total Validators: [200 .. 300]",
			args: args{
				candidates: []string{
					key,
					key2,
					key3,
					key4,
					key5,
					key6,
					key7,
					key8,
				},
				numberOfValidators: []int{
					50, 33, 29, 47, 15, 2, 25, 64,
				},
				rand: 1250000,
			},
			want: map[byte][]string{
				3: {
					key7,
				},
				4: {
					key,
					key3,
					key5,
					key8,
				},
				5: {
					key2,
					key6,
				},
				6: {
					key4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assignShardCandidateV2(tt.args.candidates, tt.args.numberOfValidators, tt.args.rand)
			if len(got) != len(tt.want) {
				t.Errorf("assignShardCandidateV2() = %v, want %v", got, tt.want)
			}
			for k, gotV := range got {
				wantV, ok := tt.want[k]
				if !ok {
					t.Errorf("assignShardCandidateV2() = %v, want %v", got, tt.want)
				}
				if !reflect.DeepEqual(gotV, wantV) {
					t.Errorf("assignShardCandidateV2() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_removeValidatorV2(t *testing.T) {
	type args struct {
		validators        []string
		removedValidators []string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Remove validators not found in list validators",
			args: args{
				validators:        []string{key},
				removedValidators: []string{key2},
			},
			wantErr: true,
			want:    []string{},
		},
		{
			name: "Found Validators In List Validators",
			args: args{
				validators:        []string{key},
				removedValidators: []string{key},
			},
			wantErr: false,
			want:    []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := removeValidatorV2(tt.args.validators, tt.args.removedValidators)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeValidatorV2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeValidatorV2() = %v, want %v", got, tt.want)
			}
		})
	}
}

//
//func Test_swapCommitteesV2(t *testing.T) {
//
//	initPublicKey()
//	initLog()
//
//	type args struct {
//		committees              []string
//		substitutes             []string
//		maxCommitteeSize        int
//		numberOfFixedValidators uint64
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    []string
//		want1   []string
//		want2   []string
//		want3   []string
//		wantErr bool
//	}{
//		{
//			name: "len(committees) == 0 && len(subtitutes) == 1",
//			args: args{
//				committees:              []string{key, key2, key3, key4},
//				substitutes:             []string{key5},
//				maxCommitteeSize:        4,
//				numberOfFixedValidators: 4,
//			},
//			want:    []string{key, key2, key3, key4},
//			want1:   []string{key5},
//			want2:   []string{},
//			want3:   []string{},
//			wantErr: false,
//		},
//		{
//			name: "Swap Offset == 0",
//			args: args{
//				substitutes:             []string{},
//				committees:              []string{},
//				maxCommitteeSize:        10,
//				numberOfFixedValidators: 0,
//			},
//			want:    []string{},
//			want1:   []string{},
//			want2:   []string{},
//			want3:   []string{},
//			wantErr: false,
//		},
//		{
//			name: "len(committees) < maxCommitteeSize && len(committees) + len(subtitutes) <= maxCommitteeSize",
//			args: args{
//				substitutes:             []string{key5},
//				committees:              []string{key, key2, key3, key4},
//				maxCommitteeSize:        4,
//				numberOfFixedValidators: 0,
//			},
//			want:    []string{key2, key3, key4, key5},
//			want1:   []string{},
//			want2:   []string{key},
//			want3:   []string{key5},
//			wantErr: false,
//		},
//		{
//			name: "swapoffset + len(committees) <= maxCommitteeSize",
//			args: args{
//				substitutes:             []string{key5, key6},
//				committees:              []string{key, key2, key3, key4},
//				maxCommitteeSize:        6,
//				numberOfFixedValidators: 0,
//			},
//			want:    []string{key, key2, key3, key4, key5, key6},
//			want1:   []string{},
//			want2:   []string{},
//			want3:   []string{key5, key6},
//			wantErr: false,
//		},
//		{
//			name: "swapoffset + len(committees) > maxCommitteeSize && len(committees) < maxCommitteeSize",
//			args: args{
//				substitutes:      []string{key5, key6},
//				committees:       []string{key, key2, key3, key4},
//				maxCommitteeSize: 5,
//			},
//			want:    []string{key2, key3, key4, key5, key6},
//			want1:   []string{},
//			want2:   []string{key},
//			want3:   []string{key5, key6},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, got1, got2, got3, err := swapCommitteesV2(tt.args.committees, tt.args.substitutes, tt.args.maxCommitteeSize, tt.args.numberOfFixedValidators)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("swapCommitteesV2() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("swapCommitteesV2() got = %v, want %v", got, tt.want)
//			}
//			if !reflect.DeepEqual(got1, tt.want1) {
//				t.Errorf("swapCommitteesV2() got1 = %v, want %v", got1, tt.want1)
//			}
//			if !reflect.DeepEqual(got2, tt.want2) {
//				t.Errorf("swapCommitteesV2() got2 = %v, want %v", got2, tt.want2)
//			}
//			if !reflect.DeepEqual(got3, tt.want3) {
//				t.Errorf("swapCommitteesV2() got3 = %v, want %v", got3, tt.want3)
//			}
//		})
//	}
//}
