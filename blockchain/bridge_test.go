package blockchain

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/mocks"
	"github.com/pkg/errors"
)

func TestGenerateInstruction(t *testing.T) {
	testCases := []struct {
		desc    string
		pending int
		val     int
		out     int // -1: expecting no SwapConfirm inst
		err     bool
	}{
		{
			desc:    "In 1, out 0",
			pending: 2,
			val:     22,
			out:     23,
		},
		{
			desc:    "Committee not changed",
			pending: 0,
			val:     22,
			out:     -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			bc, shardID, beaconHeight, beaconBlocks, shardPendingValidator, shardCommittee := getGenerateInstructionTestcase(tc.pending, tc.val)

			insts, _, _, err := bc.generateInstruction(
				shardID,
				beaconHeight,
				beaconBlocks,
				shardPendingValidator,
				shardCommittee,
			)
			isErr := err != nil
			if isErr != tc.err {
				t.Error(errors.Errorf("expect error = %t, got %v", tc.err, err))
			}
			if tc.err {
				return
			}

			if tc.out < 0 {
				if len(insts) > 1 {
					t.Errorf("expect no SwapConfirm inst, found: %+v", insts[1:])
				}
				return
			}

			if len(insts) != 2 { // Swap and SwapConfirm insts
				t.Errorf("expect 2 insts, found %d: %+v", len(insts), insts)
			}

			// Check number of validators in instruction
			sc := insts[1]
			b, _, _ := base58.DecodeCheck(sc[3])
			exp := int64(tc.out)
			if numVals := (&big.Int{}).SetBytes(b); numVals.Int64() != exp {
				t.Errorf("expected %d validators, found %d: %+v", exp, numVals, sc)
			}

		})
	}
}

func getGenerateInstructionTestcase(pending, val int) (
	*BlockChain,
	byte,
	uint64,
	[]*BeaconBlock,
	[]string,
	[]string,
) {
	beaconHeight := uint64(100)
	db := &mocks.DatabaseInterface{}
	db.On("GetProducersBlackList", beaconHeight).Return(nil, nil)
	bc := &BlockChain{
		config: Config{
			ChainParams: &Params{
				Epoch:      100,
				Offset:     1,
				SwapOffset: 1,
			},
			DataBase: db,
		},
		BestState: &BestState{
			Shard: map[byte]*ShardBestState{
				byte(1): &ShardBestState{
					ShardHeight:            1000,
					NumOfBlocksByProducers: map[string]uint64{},
					MaxShardCommitteeSize:  TestNetShardCommitteeSize,
					MinShardCommitteeSize:  TestNetMinShardCommitteeSize,
				},
			},
		},
	}

	shardID := byte(1)
	beaconBlocks := []*BeaconBlock{}
	vals := keyStore()
	shardPendingValidator := vals[:pending]
	shardCommittee := vals[pending : pending+val]
	return bc, shardID, beaconHeight, beaconBlocks, shardPendingValidator, shardCommittee
}

func keyStore() []string {
	return []string{
		"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",
	}
}

func TestParseAndConcatPubkeys(t *testing.T) {
	testCases := []struct {
		desc string
		vals []string
		out  []byte
		err  bool
	}{
		{
			desc: "Valid validators",
			vals: getCommitteeKeys(),
			out:  getCommitteeAddresses(),
		},
		{
			desc: "Invalid validator keys",
			vals: func() []string {
				vals := getCommitteeKeys()
				vals[0] = vals[0] + "a"
				return vals
			}(),
			err: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			addrs, err := parseAndConcatPubkeys(tc.vals)
			isErr := err != nil
			if isErr != tc.err {
				t.Error(errors.Errorf("expect error = %t, got %v", tc.err, err))
			}
			if tc.err {
				return
			}
			if !bytes.Equal(addrs, tc.out) {
				t.Errorf("invalid committee addresses, expect %x, got %x", tc.out, addrs)
			}
		})
	}
}

func TestBuildSwapConfirmInstruction(t *testing.T) {
	testCases := []struct {
		desc string
		in   *swapInput
		err  bool
	}{
		{
			desc: "Valid swap confirm instruction",
			in:   getSwapBeaconConfirmInput(),
		},
		{
			desc: "Invalid committee",
			in: func() *swapInput {
				s := getSwapBeaconConfirmInput()
				s.vals[0] = s.vals[0] + "A"
				return s
			}(),
			err: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			inst, err := buildSwapConfirmInstruction(tc.in.meta, tc.in.vals, tc.in.startHeight)
			isErr := err != nil
			if isErr != tc.err {
				t.Error(errors.Errorf("expect error = %t, got %v", tc.err, err))
			}
			if tc.err {
				return
			}
			checkSwapConfirmInst(t, inst, tc.in)
		})
	}
}

func checkSwapConfirmInst(t *testing.T, inst []string, input *swapInput) {
	if meta, err := strconv.Atoi(inst[0]); err != nil || meta != input.meta {
		t.Errorf("invalid meta, expect %d, got %d, err = %v", input.meta, meta, err)
	}
	if bridgeID, err := strconv.Atoi(inst[1]); err != nil || bridgeID != 1 {
		t.Errorf("invalid bridgeID, expect %d, got %d, err = %v", 1, bridgeID, err)
	}
	if b, ok := checkDecodeB58(t, inst[2]); ok {
		if h := big.NewInt(0).SetBytes(b); h.Uint64() != input.startHeight {
			t.Errorf("invalid height, expect %d, got %d", input.startHeight, h)
		}
	}
	if b, ok := checkDecodeB58(t, inst[3]); ok {
		if numVals := big.NewInt(0).SetBytes(b); numVals.Uint64() != uint64(len(input.vals)) {
			t.Errorf("invalid #vals, expect %d, got %d", len(input.vals), numVals)
		}
	}
	if b, ok := checkDecodeB58(t, inst[4]); ok {
		if !bytes.Equal(b, getCommitteeAddresses()) {
			t.Errorf("invalid committee addresses, expect %x, got %x", input.vals, b)
		}
	}
}

func getSwapBeaconConfirmInput() *swapInput {
	return &swapInput{
		meta:        70,
		vals:        getCommitteeKeys(),
		startHeight: 123,
	}
}

func getSwapBridgeConfirmInput() *swapInput {
	return &swapInput{
		meta:        71,
		vals:        getCommitteeKeys(),
		startHeight: 123,
	}
}

type swapInput struct {
	meta        int
	vals        []string
	startHeight uint64
}

func getCommitteeAddresses() []byte {
	comm := []string{
		"9BC0faE7BB432828759B6e391e0cC99995057791",
		"6cbc2937FEe477bbda360A842EeEbF92c2FAb613",
		"cabF3DB93eB48a61d41486AcC9281B6240411403",
	}
	addrs := []byte{}
	for _, c := range comm {
		addr, _ := hex.DecodeString(c)
		addrs = append(addrs, addr...)
	}
	return addrs
}

func getCommitteeKeys() []string {
	return []string{
		"121VhftSAygpEJZ6i9jGk9depfiEJfPCUqMoeDS3QJgAURzB7XZFeoaQtPuXYTAd46CNDt5FS1fNgKkEdKcX4PbwxDoL8hACe1bdNoRaGnwvU4wHHY2TxY3kxpTe7w6GxMzGBLwb9GEoiRCh1r2RdxWNvAHMhHMPNzBBfRtJ45iXXtJYgbbB1rUqbGiCV4TDgt5QV3v4KZFYoTiXmURyqXbQeVJkkABRu1BR16HDrfGcNi5LL3s8Z8iemeTm8F1FAvrXdWBeqsTEQeqHuUrY6s5cPVCnTfuCDRRSJFDhLx33CmTiWux8vYdWfNKFuX1E8hJU2vaSgFzjypTWsZb814FMxsztHoq1ibnAXKfbXgZxj9RwjecXWe7285WWEHZsLcWZ3ncW1x6Bga5ZDVQX1zeQh88kSnsebxmfGwQzV8HWikRM",
		"121VhftSAygpEJZ6i9jGk9dvuAMKafpQ1EiTVzFiUVLDYBAfmjkidFCjAJD7UpJQkeakutbs4MfGx1AizjdQ49WY2TWDw2q5sMNsoeHSPSE3Qaqxd45HRAdHH2A7cWseo4sMAVWchFuRaoUJrTB36cqjXVKet1aK8sQJQbPwmnrmHnztsaEw6Soi6vg7TkoG96HJwxQVZaUtWfPpWBZQje5SnLyB15VYqs7KBSK2Fqz4jk2L18idrxXojQQYRfigfdNrLsjwT7FMJhNkN31YWiCs47yZX9hzixqwj4DpsmHQqM1S7FmNApWGePXT86woSTL9yUqAYaA9xXkYDPsajjbxag7vqDyGtbanG7rzZSP3L93oiR4bFxmstYyghsezoXVUoJs9wy98JGH3MmDgZ8gK64sAAsgAu6Lk4AjvkreEyK4K",
		"121VhftSAygpEJZ6i9jGk9dPdVubogXXJe23BYZ1uBiJq4x6aLuEar5iRzsk1TfR995g4C18bPV8yi8frkoeJdPfK2a9CAfaroJdgmBHSUi1yVVAWttSDDAT5PEbr1XhnTGmP1Z82dPwKucctwLwRzDTkBXPfWXwMpYCLs21umN8zpuoR47xZhMqDEN2ZAuWcjZhnBDoxpnmhDgoRBe7QwL2KGGGyBVhXJHc4P15V8msCLxArxKX9U2TT2bQMpw18p25vkfDX7XB2ZyozZox46cKj8PTVf2BjAhMzk5dghb3ipX4kp4p8cpVSnSpsGB8UJwer4LxHhN2sRDm88M8PH3xxtAgs1RZBmPH6EojnbxxU5XZgGtouRda1tjEp5jFDgp2h87gY5VzEME9u5FEKyiAjR1Ye7429PGTmiSf48mtm1xW",
	}
}

func TestBuildBeaconSwapConfirmInstruction(t *testing.T) {
	testCases := []struct {
		desc string
		in   *swapInput
		err  bool
	}{
		{
			desc: "Valid swap confirm instruction",
			in:   getSwapBeaconConfirmInput(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			inst, err := buildBeaconSwapConfirmInstruction(tc.in.vals, tc.in.startHeight)
			isErr := err != nil
			if isErr != tc.err {
				t.Error(errors.Errorf("expect error = %t, got %v", tc.err, err))
			}
			if tc.err {
				return
			}
			tc.in.startHeight += 1 // new committee starts signing next block
			checkSwapConfirmInst(t, inst, tc.in)
		})
	}
}

func TestBuildBridgeSwapConfirmInstruction(t *testing.T) {
	testCases := []struct {
		desc string
		in   *swapInput
		err  bool
	}{
		{
			desc: "Valid swap confirm instruction",
			in:   getSwapBridgeConfirmInput(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			inst, err := buildBridgeSwapConfirmInstruction(tc.in.vals, tc.in.startHeight)
			isErr := err != nil
			if isErr != tc.err {
				t.Error(errors.Errorf("expect error = %t, got %v", tc.err, err))
			}
			if tc.err {
				return
			}
			tc.in.startHeight += 1 // new committee starts signing next block
			checkSwapConfirmInst(t, inst, tc.in)
		})
	}
}

func TestPickBridgeSwapConfirmInst(t *testing.T) {
	testCases := []struct {
		desc  string
		insts [][]string
		out   [][]string
	}{
		{
			desc:  "No swap inst",
			insts: [][]string{[]string{"1", "2"}, []string{"3", "4"}},
		},
		{
			desc:  "Check metaType",
			insts: [][]string{[]string{"70", "2"}, []string{"71", "1", "2", "3", "4"}},
			out:   [][]string{[]string{"71", "1", "2", "3", "4"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			insts := pickBridgeSwapConfirmInst(tc.insts)
			if len(tc.out) != len(insts) {
				t.Errorf("incorrect number of insts, expect %d, got %d", len(tc.out), len(insts))
			}
			for i, inst := range insts {
				if strings.Join(inst, "") != strings.Join(tc.out[i], "") {
					t.Errorf("incorrect bridge swap inst, expect %s, got %s", tc.out[i], inst)
				}
			}
		})
	}
}

func TestParseAndPadAddress(t *testing.T) {
	testCases := []struct {
		desc string
		inst string
		err  bool
	}{
		{
			desc: "Valid instruction",
			inst: base58.EncodeCheck(getCommitteeAddresses()),
		},
		{
			desc: "Decode fail",
			inst: func() string {
				inst := base58.EncodeCheck(getCommitteeAddresses())
				inst = inst + "a"
				return inst
			}(),
			err: true,
		},
		{
			desc: "Invalid address length",
			inst: func() string {
				addrs := getCommitteeAddresses()
				addrs = append(addrs, byte(123))
				return base58.EncodeCheck(addrs)
			}(),
			err: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			addrs, err := parseAndPadAddress(tc.inst)
			isErr := err != nil
			if isErr != tc.err {
				t.Error(errors.Errorf("expect error = %t, got %v", tc.err, err))
			}
			if tc.err {
				return
			}

			checkPaddedAddresses(t, addrs, tc.inst)
		})
	}
}

func checkPaddedAddresses(t *testing.T, addrs []byte, inst string) {
	b, _, _ := base58.DecodeCheck(inst)
	numAddrs := len(b) / 20
	if len(addrs) != numAddrs*32 {
		t.Fatalf("incorrect padded length, expect %d, got %d", numAddrs*32, len(addrs))
	}
	zero := make([]byte, 12)
	for i := 0; i < numAddrs; i++ {
		prefix := addrs[i*32 : i*32+12]
		if !bytes.Equal(zero, prefix) {
			t.Errorf("address must start with 12 bytes of 0, expect %x, got %x", zero, prefix)
		}

		addr := addrs[i*32+12 : (i+1)*32]
		exp := b[i*20 : (i+1)*20]
		if !bytes.Equal(exp, addr) {
			t.Errorf("wrong address of committee member, expect %x, got %x", exp, addr)
		}
	}
}

func TestDecodeSwapConfirm(t *testing.T) {
	addrs := []string{
		"834f98e1b7324450b798359c9febba74fb1fd888",
		"1250ba2c592ac5d883a0b20112022f541898e65b",
		"2464c00eab37be5a679d6e5f7c8f87864b03bfce",
		"6d4850ab610be9849566c09da24b37c5cfa93e50",
	}
	testCases := []struct {
		desc string
		inst []string
		out  []byte
		err  bool
	}{
		{
			desc: "Swap beacon instruction",
			inst: buildEncodedSwapConfirmInst(70, 1, 123, addrs),
			out:  buildDecodedSwapConfirmInst(70, 1, 123, addrs),
		},
		{
			desc: "Swap bridge instruction",
			inst: buildEncodedSwapConfirmInst(71, 1, 19827312, []string{}),
			out:  buildDecodedSwapConfirmInst(71, 1, 19827312, []string{}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			decoded, err := decodeSwapConfirmInst(tc.inst)
			isErr := err != nil
			if isErr != tc.err {
				t.Error(errors.Errorf("expect error = %t, got %v", tc.err, err))
			}
			if tc.err {
				return
			}

			if !bytes.Equal(decoded, tc.out) {
				t.Errorf("invalid decoded swap inst, expect\n%v, got\n%v", tc.out, decoded)
			}
		})
	}
}

func buildEncodedSwapConfirmInst(meta, shard, height int, addrs []string) []string {
	a := []byte{}
	for _, addr := range addrs {
		d, _ := hex.DecodeString(addr)
		a = append(a, d...)
	}
	inst := []string{
		strconv.Itoa(meta),
		strconv.Itoa(shard),
		base58.EncodeCheck(big.NewInt(int64(height)).Bytes()),
		base58.EncodeCheck(big.NewInt(int64(len(addrs))).Bytes()),
		base58.EncodeCheck(a),
	}
	return inst
}

func buildDecodedSwapConfirmInst(meta, shard, height int, addrs []string) []byte {
	a := []byte{}
	for _, addr := range addrs {
		d, _ := hex.DecodeString(addr)
		a = append(a, toBytes32BigEndian(d)...)
	}
	decoded := []byte{byte(meta)}
	decoded = append(decoded, byte(shard))
	decoded = append(decoded, toBytes32BigEndian(big.NewInt(int64(height)).Bytes())...)
	decoded = append(decoded, toBytes32BigEndian(big.NewInt(int64(len(addrs))).Bytes())...)
	decoded = append(decoded, a...)
	return decoded
}

func checkDecodeB58(t *testing.T, e string) ([]byte, bool) {
	b, _, err := base58.DecodeCheck(e)
	if err != nil {
		t.Error(errors.WithStack(err))
		return nil, false
	}
	return b, true
}
