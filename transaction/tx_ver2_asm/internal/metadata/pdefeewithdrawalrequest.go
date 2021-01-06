package metadata

import (
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

// PDEFeeWithdrawalRequest - privacy dex withdrawal request
type PDEFeeWithdrawalRequest struct {
	WithdrawerAddressStr  string
	WithdrawalToken1IDStr string
	WithdrawalToken2IDStr string
	WithdrawalFeeAmt      uint64
	MetadataBase
}

type PDEFeeWithdrawalRequestAction struct {
	Meta    PDEFeeWithdrawalRequest
	TxReqID common.Hash
	ShardID byte
}

func NewPDEFeeWithdrawalRequest(
	withdrawerAddressStr string,
	withdrawalToken1IDStr string,
	withdrawalToken2IDStr string,
	withdrawalFeeAmt uint64,
	metaType int,
) (*PDEFeeWithdrawalRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}
	pdeFeeWithdrawalRequest := &PDEFeeWithdrawalRequest{
		WithdrawerAddressStr:  withdrawerAddressStr,
		WithdrawalToken1IDStr: withdrawalToken1IDStr,
		WithdrawalToken2IDStr: withdrawalToken2IDStr,
		WithdrawalFeeAmt:      withdrawalFeeAmt,
	}
	pdeFeeWithdrawalRequest.MetadataBase = metadataBase
	return pdeFeeWithdrawalRequest, nil
}

func (*PDEFeeWithdrawalRequest) ShouldSignMetaData() bool { return true }

func (pc PDEFeeWithdrawalRequest) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.WithdrawerAddressStr
	record += pc.WithdrawalToken1IDStr
	record += pc.WithdrawalToken2IDStr
	record += strconv.FormatUint(pc.WithdrawalFeeAmt, 10)
	if pc.Sig != nil && len(pc.Sig) != 0 {
		record += string(pc.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc PDEFeeWithdrawalRequest) HashWithoutSig() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.WithdrawerAddressStr
	record += pc.WithdrawalToken1IDStr
	record += pc.WithdrawalToken2IDStr
	record += strconv.FormatUint(pc.WithdrawalFeeAmt, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
