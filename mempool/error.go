package mempool

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	RejectDuplicateTx = iota
	RejectInvalidTx
	RejectSanityTx
	RejectSanityTxLocktime
	RejectSalaryTx
	RejectDuplicateStakePubkey
	RejectDuplicateRequestStopAutoStaking
	RejectVersion
	RejectInvalidTxType
	RejectDoubleSpendWithMempoolTx
	RejectDoubleSpendWithBlockchainTx
	RejectMetadataWithBlockchainTx
	RejectInvalidFee
	RejectInvalidSize
	CanNotCheckDoubleSpend
	DatabaseError
	MarshalError
	UnmarshalShardCommitteeError
	DuplicateBlockError
	OldBlockError
	MaxPoolSizeError
	UnexpectedTransactionError
	TransactionNotFoundError
	RejectTestTransactionError
	WrongShardIDError
	HashError
	RejectReplacementTxError
	GetStakingMetadataError
	WalletKeySerializedError
	GetBeaconBlockHashFromDatabaseError
	FetchBeaconBlockFromDatabaseError
	UnmarshalBeaconBlockError
	FindBeaconHeightForCrossShardBlockError
	ValidateAggSignatureForCrossShardBlockError
	DuplicateSerialNumbersHashError
	CouldNotGetExchangeRateError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	Message string
}{
	RejectDuplicateTx:                           {-1000, "Reject duplicate tx in mempool"},
	RejectInvalidTx:                             {-1001, "Reject invalid tx"},
	RejectSanityTx:                              {-1002, "Reject not sansity tx"},
	RejectSalaryTx:                              {-1003, "Reject salary tx"},
	RejectInvalidFee:                            {-1004, "Reject invalid fee"},
	RejectVersion:                               {-1005, "Reject invalid version"},
	CanNotCheckDoubleSpend:                      {-1006, "Can not check double spend"},
	DatabaseError:                               {-1007, "Database Error"},
	RejectDuplicateStakePubkey:                  {-1008, "Reject Duplicate Stake Error"},
	DuplicateBlockError:                         {-1009, "Duplicate Block Error"},
	OldBlockError:                               {-1010, "Old Block Error"},
	MaxPoolSizeError:                            {-1011, "Max Pool Size Error"},
	UnexpectedTransactionError:                  {-1012, "Unexpected Transaction Error"},
	TransactionNotFoundError:                    {-1013, "Transaction Not Found Error"},
	RejectTestTransactionError:                  {-1014, "Reject Test Transaction Error"},
	RejectInvalidTxType:                         {-1015, "Reject Invalid Tx Type"},
	RejectDoubleSpendWithMempoolTx:              {-1016, "Reject Double Spend With Other Tx in mempool"},
	RejectDoubleSpendWithBlockchainTx:           {-1017, "Reject Double Spend With Current Blockchain"},
	WrongShardIDError:                           {-1018, "Reject Cross Shard Block With Same ShardID in Pool"},
	MarshalError:                                {-1019, "Marshal Error"},
	UnmarshalShardCommitteeError:                {-1020, "Unmarshal Shard Committee Error"},
	HashError:                                   {-1021, "Hash Error"},
	RejectReplacementTxError:                    {-1022, "Replacement or Cancel Tx Error"},
	GetStakingMetadataError:                     {-1023, "Get Staking Metadata Error"},
	WalletKeySerializedError:                    {-1024, "Wallet Key Serialized Error"},
	GetBeaconBlockHashFromDatabaseError:         {-1025, "Get Beacon Block Hash From Database Error"},
	FetchBeaconBlockFromDatabaseError:           {-1026, "Fetch Beacon Block From Database Error"},
	UnmarshalBeaconBlockError:                   {-1027, "Unmarshal Beacon Block Error"},
	FindBeaconHeightForCrossShardBlockError:     {-1028, "Find Beacon Height For Cross Shard Block Error"},
	ValidateAggSignatureForCrossShardBlockError: {-1029, "Validate Agg Signature For Cross Shard Block Error"},
	RejectDuplicateRequestStopAutoStaking:       {-1030, "Reject Duplicate Request Stop Auto Staking"},
	DuplicateSerialNumbersHashError:             {-1031, "Duplicate Serial Numbers Hash Error"},
	CouldNotGetExchangeRateError:                {-1032, "Could not get the exchange rate error"},
	RejectSanityTxLocktime:                      {-1033, "Wrong tx locktime"},
	RejectMetadataWithBlockchainTx:              {-1034, "Reject invalid metadata with blockchain"},
}

type MempoolTxError struct {
	Code    int    // The code to send with reject messages
	Message string // Human readable message of the issue
	Err     error
}

// Error satisfies the error interface and prints human-readable errors.
func (e MempoolTxError) Error() string {
	return fmt.Sprintf("%d: %s %+v", e.Code, e.Message, e.Err)
}

func NewMempoolTxError(key int, err error) *MempoolTxError {
	return &MempoolTxError{
		Code:    ErrCodeMessage[key].Code,
		Message: ErrCodeMessage[key].Message,
		Err:     errors.Wrap(err, ErrCodeMessage[key].Message),
	}
}

type BlockPoolError struct {
	Code    int
	Message string
	Err     error
}

func (e *BlockPoolError) Error() string {
	return fmt.Sprintf("%d: %s %+v", e.Code, e.Message, e.Err)
}

func (e *BlockPoolError) Init(key int, err error) {
	e.Code = ErrCodeMessage[key].Code
	e.Message = ErrCodeMessage[key].Message
	e.Err = errors.Wrap(err, e.Message)
}
