package instruction

import (
	"fmt"
	"strconv"
)

//Random Instruction which get nonce from bitcoin block
type RandomInstruction struct {
	BtcNonce       int64
	BtcBlockHeight int
	CheckPointTime int64
	BtcBlockTime   int64
}

func NewRandomInstructionWithValue(btcNonce int64, btcBlockHeight int, checkPointTime int64, btcBlockTime int64) *RandomInstruction {
	return &RandomInstruction{BtcNonce: btcNonce, BtcBlockHeight: btcBlockHeight, CheckPointTime: checkPointTime, BtcBlockTime: btcBlockTime}
}

func NewRandomInstruction() *RandomInstruction {
	s := &RandomInstruction{}
	return s
}

func (s *RandomInstruction) GetType() string {
	return RANDOM_ACTION
}

func (s *RandomInstruction) SetNonce(n int64) *RandomInstruction {
	s.BtcNonce = n
	return s
}

func (s *RandomInstruction) SetBtcBlockHeight(n int) *RandomInstruction {
	s.BtcBlockHeight = n
	return s
}

func (s *RandomInstruction) SetBtcBlockTime(n int64) *RandomInstruction {
	s.BtcBlockTime = n
	return s
}

func (s *RandomInstruction) SetCheckPointTime(n int64) *RandomInstruction {
	s.CheckPointTime = n
	return s
}

func (s *RandomInstruction) ToString() []string {
	strs := []string{}
	strs = append(strs, RANDOM_ACTION)
	strs = append(strs, strconv.FormatInt(s.BtcNonce, 10))
	strs = append(strs, strconv.Itoa(s.BtcBlockHeight))
	strs = append(strs, strconv.FormatInt(s.CheckPointTime, 10))
	strs = append(strs, strconv.FormatInt(s.BtcBlockTime, 10))
	return strs
}

func ValidateAndImportRandomInstructionFromString(instruction []string) (*RandomInstruction, error) {
	if err := ValidateRandomInstructionSanity(instruction); err != nil {
		return nil, err
	}
	return ImportRandomInstructionFromString(instruction), nil
}

// ImportRandomInstructionFromString is unsafe method
func ImportRandomInstructionFromString(instruction []string) *RandomInstruction {
	btcNonce, _ := strconv.ParseInt(instruction[1], 10, 64)
	btcBlockHeight, _ := strconv.ParseInt(instruction[2], 10, 64)
	checkPointTime, _ := strconv.ParseInt(instruction[3], 10, 64)
	btcTimeStamp, _ := strconv.ParseInt(instruction[4], 10, 64)
	r := NewRandomInstructionWithValue(btcNonce, int(btcBlockHeight), checkPointTime, btcTimeStamp)
	return r
}

func ValidateRandomInstructionSanity(instruction []string) error {
	if len(instruction) < 2 {
		return fmt.Errorf("invalid length, %+v", instruction)
	}
	if instruction[0] != RANDOM_ACTION {
		return fmt.Errorf("invalid random action, %+v", instruction)
	}
	if _, err := strconv.ParseInt(instruction[1], 10, 64); err != nil {
		return fmt.Errorf("invalid btc nonce value, %s", err)
	}
	return nil
}
