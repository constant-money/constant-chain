package metadata

import (
	"strconv"
	"encoding/json"

	"github.com/incognitochain/incognito-chain/common"
)

// PDEWithdrawalRequest - privacy dex withdrawal request
type PDEWithdrawalRequest struct {
	WithdrawerAddressStr  string
	WithdrawalToken1IDStr string
	WithdrawalToken2IDStr string
	WithdrawalShareAmt    uint64
	MetadataBase
}

type PDEWithdrawalRequestAction struct {
	Meta    PDEWithdrawalRequest
	TxReqID common.Hash
	ShardID byte
}

type PDEWithdrawalAcceptedContent struct {
	WithdrawalTokenIDStr string
	WithdrawerAddressStr string
	DeductingPoolValue   uint64
	DeductingShares      uint64
	PairToken1IDStr      string
	PairToken2IDStr      string
	TxReqID              common.Hash
	ShardID              byte
}

func NewPDEWithdrawalRequest(
	withdrawerAddressStr string,
	withdrawalToken1IDStr string,
	withdrawalToken2IDStr string,
	withdrawalShareAmt uint64,
	metaType int,
) (*PDEWithdrawalRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}
	pdeWithdrawalRequest := &PDEWithdrawalRequest{
		WithdrawerAddressStr:  withdrawerAddressStr,
		WithdrawalToken1IDStr: withdrawalToken1IDStr,
		WithdrawalToken2IDStr: withdrawalToken2IDStr,
		WithdrawalShareAmt:    withdrawalShareAmt,
	}
	pdeWithdrawalRequest.MetadataBase = metadataBase
	return pdeWithdrawalRequest, nil
}

func (*PDEWithdrawalRequest) ShouldSignMetaData() bool { return true }
func (pc PDEWithdrawalRequest) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.WithdrawerAddressStr
	record += pc.WithdrawalToken1IDStr
	record += pc.WithdrawalToken2IDStr
	record += strconv.FormatUint(pc.WithdrawalShareAmt, 10)
	if pc.Sig != nil && len(pc.Sig) != 0 {
		record += string(pc.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc PDEWithdrawalRequest) HashWithoutSig() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.WithdrawerAddressStr
	record += pc.WithdrawalToken1IDStr
	record += pc.WithdrawalToken2IDStr
	record += strconv.FormatUint(pc.WithdrawalShareAmt, 10)

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc *PDEWithdrawalRequest) UnmarshalJSON(raw []byte) error{
	var temp struct{
		WithdrawerAddressStr  string
		WithdrawalToken1IDStr string
		WithdrawalToken2IDStr string
		WithdrawalShareAmt    uintMaybeString
		MetadataBase
	}
	err := json.Unmarshal(raw, &temp)
	*pc = PDEWithdrawalRequest{
		WithdrawerAddressStr: temp.WithdrawerAddressStr,
		WithdrawalToken1IDStr: temp.WithdrawalToken1IDStr,
		WithdrawalToken2IDStr: temp.WithdrawalToken2IDStr,
		WithdrawalShareAmt: uint64(temp.WithdrawalShareAmt),
		MetadataBase: temp.MetadataBase,
	}
	return err
}

