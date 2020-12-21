package blockchain

import (
	"encoding/hex"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/pkg/errors"
)

// FlattenAndConvertStringInst receives a slice of insts; concats each inst ([]string) and converts to []byte to build merkle tree later
func FlattenAndConvertStringInst(insts [][]string) ([][]byte, error) {
	flattenInsts := [][]byte{}
	for _, inst := range insts {
		d, err := DecodeInstruction(inst)
		if err != nil {
			return nil, err
		}
		flattenInsts = append(flattenInsts, d)
	}
	return flattenInsts, nil
}

// decodeInstruction appends all part of an instruction and decode them if necessary (for special instruction that needed to be decoded before submitting to Ethereum)
func DecodeInstruction(inst []string) ([]byte, error) {
	flatten := []byte{}
	var err error
	switch inst[0] {
	case strconv.Itoa(metadata.BeaconSwapConfirmMeta), strconv.Itoa(metadata.BridgeSwapConfirmMeta):
		flatten, err = decodeSwapConfirmInst(inst)
		if err != nil {
			return nil, err
		}

	case strconv.Itoa(metadata.BurningConfirmMeta), strconv.Itoa(metadata.BurningConfirmForDepositToSCMeta), strconv.Itoa(metadata.BurningConfirmMetaV2), strconv.Itoa(metadata.BurningConfirmForDepositToSCMetaV2):
		var err error
		flatten, err = decodeBurningConfirmInst(inst)
		if err != nil {
			return nil, err
		}

	default:
		for _, part := range inst {
			flatten = append(flatten, []byte(part)...)
		}
	}
	return flatten, nil
}

// decodeSwapConfirmInst flattens all parts of a swap confirm instruction, decodes and concats it
func decodeSwapConfirmInst(inst []string) ([]byte, error) {
	m, errMeta := strconv.Atoi(inst[0])
	s, errShard := strconv.Atoi(inst[1])
	metaType := byte(m)
	shardID := byte(s)
	height, _, errHeight := base58.Base58Check{}.Decode(inst[2])
	numVals, _, errNumVals := base58.Base58Check{}.Decode(inst[3])
	// Special case: instruction storing beacon/bridge's committee => decode and sign on that instead
	// We need to decode and then submit the pubkeys to Ethereum because we can't decode it on smart contract
	addrs, errAddrs := parseAndPadAddress(inst[4])
	if err := common.CheckError(errMeta, errShard, errHeight, errNumVals, errAddrs); err != nil {
		err = errors.Wrapf(err, "inst: %+v", inst)
		BLogger.log.Error(err)
		return nil, err
	}

	flatten := []byte{}
	flatten = append(flatten, metaType)
	flatten = append(flatten, shardID)
	flatten = append(flatten, toBytes32BigEndian(height)...)
	flatten = append(flatten, toBytes32BigEndian(numVals)...)
	flatten = append(flatten, addrs...)
	return flatten, nil
}

// parseAndPadAddress decodes a list of address of a committee, pads each of them
// to 32 bytes and concat them together
func parseAndPadAddress(instContent string) ([]byte, error) {
	addrPacked, _, err := base58.DecodeCheck(instContent)
	if err != nil {
		return nil, errors.Wrapf(err, "instContent: %v", instContent)
	}
	if len(addrPacked)%20 != 0 {
		return nil, errors.Errorf("invalid packed eth addresses length: %x", addrPacked)
	}
	addrs := []byte{}
	for i := 0; i*20 < len(addrPacked); i++ {
		addr := toBytes32BigEndian(addrPacked[i*20 : (i+1)*20])
		addrs = append(addrs, addr...)
	}
	return addrs, nil
}

// decodeBurningConfirmInst decodes and flattens a BurningConfirm instruction
func decodeBurningConfirmInst(inst []string) ([]byte, error) {
	if len(inst) < 8 {
		return nil, errors.New("invalid length of BurningConfirm inst")
	}
	m, errMeta := strconv.Atoi(inst[0])
	s, errShard := strconv.Atoi(inst[1])
	metaType := byte(m)
	shardID := byte(s)
	tokenID, _, errToken := base58.Base58Check{}.Decode(inst[2])
	remoteAddr, errAddr := decodeRemoteAddr(inst[3])
	amount, _, errAmount := base58.Base58Check{}.Decode(inst[4])
	txID, errTx := common.Hash{}.NewHashFromStr(inst[5])
	incTokenID, _, errIncToken := base58.Base58Check{}.Decode(inst[6])
	height, _, errHeight := base58.Base58Check{}.Decode(inst[7])
	if err := common.CheckError(errMeta, errShard, errToken, errAddr, errAmount, errTx, errIncToken, errHeight); err != nil {
		err = errors.Wrapf(err, "inst: %+v", inst)
		BLogger.log.Error(err)
		return nil, err
	}

	BLogger.log.Infof("Decoded BurningConfirm inst, amount: %d, remoteAddr: %x, tokenID: %x", big.NewInt(0).SetBytes(amount), remoteAddr, tokenID)
	flatten := []byte{}
	flatten = append(flatten, metaType)
	flatten = append(flatten, shardID)
	flatten = append(flatten, toBytes32BigEndian(tokenID)...)
	flatten = append(flatten, remoteAddr...)
	flatten = append(flatten, toBytes32BigEndian(amount)...)
	flatten = append(flatten, txID[:]...)
	flatten = append(flatten, incTokenID...)
	flatten = append(flatten, toBytes32BigEndian(height)...)
	return flatten, nil
}

// decodeRemoteAddr converts address string to 32 bytes slice
func decodeRemoteAddr(addr string) ([]byte, error) {
	remoteAddr, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	addrFixedLen := [32]byte{}
	copy(addrFixedLen[32-len(remoteAddr):], remoteAddr)
	return addrFixedLen[:], nil
}

// toBytes32BigEndian converts a []byte to uint256 of Ethereum
func toBytes32BigEndian(b []byte) []byte {
	a := [32]byte{}
	copy(a[32-len(b):], b)
	return a[:]
}

// pickInstructionWithType finds all instructions of a specific type in a list
func pickInstructionWithType(
	insts [][]string,
	typeToFind string,
) [][]string {
	found := [][]string{}
	for _, inst := range insts {
		instType := inst[0]
		if instType != typeToFind {
			continue
		}
		found = append(found, inst)
	}
	return found
}

// pickInstructionFromBeaconBlocks extracts all instructions of a specific type
func pickInstructionFromBeaconBlocks(beaconBlocks []*BeaconBlock, instType string) [][]string {
	insts := [][]string{}
	for _, block := range beaconBlocks {
		found := pickInstructionWithType(block.Body.Instructions, instType)
		if len(found) > 0 {
			insts = append(insts, found...)
		}
	}
	return insts
}

// pickBurningConfirmInstruction finds all BurningConfirmMeta instructions
func pickBurningConfirmInstructionV1(
	beaconBlocks []*BeaconBlock,
	height uint64,
) [][]string {
	metas := []string{
		strconv.Itoa(metadata.BurningConfirmMeta),
		strconv.Itoa(metadata.BurningConfirmForDepositToSCMeta),
	}

	insts := [][]string{}
	for _, meta := range metas {
		instOfMeta := pickInstructionFromBeaconBlocks(beaconBlocks, meta)
		insts = append(insts, instOfMeta...)
	}

	// Replace beacon block height with shard's
	h := big.NewInt(0).SetUint64(height)
	for _, inst := range insts {
		inst[len(inst)-1] = base58.Base58Check{}.Encode(h.Bytes(), 0x00)
	}
	return insts
}

// pickBridgeSwapConfirmInst finds all BridgeSwapConfirmMeta instructions in a shard to beacon block
func pickBridgeSwapConfirmInst(
	instructions [][]string,
) [][]string {
	metaType := strconv.Itoa(metadata.BridgeSwapConfirmMeta)
	return pickInstructionWithType(instructions, metaType)
}

// parseAndConcatPubkeys parses pubkeys of a commmittee (stored as string), converts them to addresses and concat them together
func parseAndConcatPubkeys(vals []string) ([]byte, error) {
	addrs := []byte{}
	for _, val := range vals {
		cKey := &incognitokey.CommitteePublicKey{}
		if err := cKey.FromBase58(val); err != nil {
			return nil, err
		}
		miningKey := cKey.MiningPubKey[common.BridgeConsensus]
		pk, err := crypto.DecompressPubkey(miningKey)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot decompress miningKey %v", miningKey)
		}
		addr := crypto.PubkeyToAddress(*pk)

		addrs = append(addrs, addr[:]...)
	}
	return addrs, nil
}

// buildSwapConfirmInstruction builds a confirm instruction for either beacon
// or bridge committee swap
func buildSwapConfirmInstruction(meta int, currentValidators []string, startHeight uint64) ([]string, error) {
	comm, err := parseAndConcatPubkeys(currentValidators)
	if err != nil {
		return nil, err
	}

	// Convert startHeight to big.Int to get bytes later
	height := big.NewInt(0).SetUint64(startHeight)

	// Save number of validators as bytes and parse on Ethereum
	numVals := big.NewInt(int64(len(currentValidators)))

	bridgeID := byte(common.BridgeShardID)
	return []string{
		strconv.Itoa(meta),
		strconv.Itoa(int(bridgeID)),
		base58.Base58Check{}.Encode(height.Bytes(), 0x00),
		base58.Base58Check{}.Encode(numVals.Bytes(), 0x00),
		base58.Base58Check{}.Encode(comm, 0x00),
	}, nil
}

// buildBeaconSwapConfirmInstruction stores in an instruction the list of
// new beacon validators and the block that they start signing on
func buildBeaconSwapConfirmInstruction(currentValidators []string, blockHeight uint64) ([]string, error) {
	BLogger.log.Infof("New beaconComm - startHeight: %d comm: %x", blockHeight+1, currentValidators)
	return buildSwapConfirmInstruction(metadata.BeaconSwapConfirmMeta, currentValidators, blockHeight+1)
}

// buildBridgeSwapConfirmInstruction stores in an instruction the list of
// new bridge validators and the block that they start signing on
func buildBridgeSwapConfirmInstruction(currentValidators []string, blockHeight uint64) ([]string, error) {
	BLogger.log.Infof("New bridgeComm - startHeight: %d comm: %x", blockHeight+1, currentValidators)
	return buildSwapConfirmInstruction(metadata.BridgeSwapConfirmMeta, currentValidators, blockHeight+1)
}
