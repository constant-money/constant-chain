package zkp

import (
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
)

var (
	fixedRandomnessShardID       = new(privacy.Scalar).FromBytesS([]byte{0x60, 0xa2, 0xab, 0x35, 0x26, 0x9, 0x97, 0x7c, 0x6b, 0xe1, 0xba, 0xec, 0xbf, 0x64, 0x27, 0x2, 0x6a, 0x9c, 0xe8, 0x10, 0x9e, 0x93, 0x4a, 0x0, 0x47, 0x83, 0x15, 0x48, 0x63, 0xeb, 0xda, 0x6})
	beaconHeightBreakPointNewZKP = uint64(0)
	isInitCheckPoint             = false
)

const validateTimeForOneoutOfManyProof = int64(1574985600)

func InitCheckpoint(
	bcHeightNewZKP uint64,
) error {
	if isInitCheckPoint {
		return errors.Errorf("Can not init twice")
	}
	isInitCheckPoint = true
	beaconHeightBreakPointNewZKP = bcHeightNewZKP
	return nil
}

func IsNewZKP(bcHeight uint64) bool {
	return (bcHeight >= beaconHeightBreakPointNewZKP)
}

func IsNewOneOfManyProof(lockTime int64) bool {
	return (lockTime >= validateTimeForOneoutOfManyProof)
}

func GetFixedRandomnessShardID() privacy.Scalar {
	return *fixedRandomnessShardID
}
