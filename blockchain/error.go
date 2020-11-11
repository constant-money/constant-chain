// Copyright (c) 2014-2016 The thaibaoautonomous developers
// Use of this source Code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	UnExpectedError = iota
	UpdateMerkleTreeForBlockError
	UnmashallJsonShardBlockError
	MashallJsonShardBlockError
	UnmashallJsonShardBestStateError
	MashallJsonShardBestStateError
	UnmashallJsonBeaconBlockError
	MashallJsonBeaconBlockError
	UnmashallJsonBeaconBestStateError
	MashallJsonBeaconBestStateError
	MashallJsonError
	CanNotCheckDoubleSpendError
	HashError
	WrongVersionError
	WrongBlockHeightError
	DatabaseError
	EpochError
	WrongTimestampError
	InstructionHashError
	ShardStateHashError
	RandomError
	VerificationError
	ShardError
	BeaconError
	SignatureError
	CrossShardBlockError
	CandidateError
	ShardIDError
	ProducerError
	ShardStateError
	TransactionFromNewBlockError
	GenerateInstructionError
	ProcessInstructionFromBeaconError
	SwapError
	DuplicateShardBlockError
	CommitteeOrValidatorError
	ShardBlockSanityError
	StoreIncomingCrossShardError
	DeleteIncomingCrossShardError
	WrongShardIDError
	CloneShardBestStateError
	CloneBeaconBestStateError
	ShardBestStateNotCompatibleError
	RegisterEstimatorFeeError
	FetchPreviousBlockError
	TransactionRootHashError
	ShardTransactionRootHashError
	CrossShardTransactionRootHashError
	FetchBeaconBlocksError
	WrongBlockTotalFeeError
	ShardIntructionFromTransactionAndInstructionError
	InstructionsHashError
	FlattenAndConvertStringInstError
	InstructionMerkleRootError
	FetchBeaconBlockHashError
	FetchBeaconBlockError
	FetchShardBlockError
	BeaconBlockNotCompatibleError
	SwapInstructionError
	TransactionCreatedByMinerError
	ResponsedTransactionWithMetadataError
	UnmashallJsonShardCommitteesError
	MashallJsonShardCommitteesError
	VerifyCrossShardBlockError
	NextCrossShardBlockError
	FetchShardCommitteeError
	CrossTransactionHashError
	VerifyCrossShardCustomTokenError
	ShardCommitteeRootHashError
	ShardPendingValidatorRootHashError
	ShardStakingTxRootHashError
	BeaconCommitteeAndPendingValidatorRootError
	ShardCommitteeAndPendingValidatorRootError
	ShardCandidateRootError
	BeaconCandidateRootError
	StoreShardBlockError
	StoreBestStateError
	FetchAndStoreTransactionError
	FetchAndStoreCrossTransactionError
	RemoveCommitteeRewardError
	StoreBurningConfirmError
	SwapValidatorError
	CrossShardBitMapError
	ShardCommitteeLengthAndCommitteeIndexError
	UpdateBridgeIssuanceStatusError
	BeaconCommitteeLengthAndCommitteeIndexError
	BuildRewardInstructionError
	GenerateBeaconCommitteeAndValidatorRootError
	GenerateShardCommitteeAndValidatorRootError
	GenerateBeaconCandidateRootError
	GenerateShardCandidateRootError
	GenerateShardStateError
	GenerateShardCommitteeError
	GenerateShardPendingValidatorError
	ProduceSignatureError
	BeaconBestStateBestBlockNotCompatibleError
	BeaconBlockProducerError
	BeaconBlockSignatureError
	WrongEpochError
	GenerateInstructionHashError
	GetShardBlocksForBeaconProcessError
	ShardStateHeightError
	ShardStateCrossShardBitMapError
	ShardBlockSignatureError
	ShardBestStateBeaconHeightNotCompatibleError
	BeaconBestStateBestShardHeightNotCompatibleError
	ProcessRandomInstructionError
	ProcessSwapInstructionError
	AssignValidatorToShardError
	ShuffleBeaconCandidateError
	CleanBackUpError
	BackUpBestStateError
	StoreBeaconBlockError
	StoreBeaconBlockIndexError
	GetStakingTransactionError
	DecodeHashError
	GetTransactionFromDatabaseError
	ProcessBridgeInstructionError
	UpdateDatabaseWithBlockRewardInfoError
	CreateCrossShardBlockError
	VerifyCrossShardBlockShardTxRootError
	WalletKeySerializedError
	InitSalaryTransactionError
	RemoveOldDataAfterProcessingError
	WrongMetadataTypeError
	StakeInstructionError
	StoreRewardReceiverByHeightError
	CreateNormalTokenTxForCrossShardError
	SnapshotCommitteeError
	ExtractPublicKeyFromCommitteeKeyListError
	PendingValidatorRootError
	CommitteeHashError
	StakingTxHashError
	StopAutoStakingRequestHashError
	StopAutoStakingMetadataError
	AutoStakingRootHashError
	FetchAllCommitteeValidatorCandidateError
	BackupFromTxViewPointError
	BackupFromCrossTxViewPointError
	BackupDatabaseFromBeaconInstructionError
	SnapshotRewardReceiverError
	StoreAutoStakingByHeightError
	FetchAutoStakingByHeightError
	ProcessSlashingError
	ConvertCommitteePubKeyToBase58Error
	ConsensusIsOngoingError
	RevertStateError
	NotEnoughRewardError
	InitPDETradeResponseTransactionError
	ProcessPDEInstructionError
	ProcessPortalInstructionError
	InitBeaconStateError
	InitShardStateError
	GetListOutputCoinsByKeysetError
	ProcessSalaryInstructionsError
	GetShardIDFromTxError
	GetValueFromTxError
	ValidateBlockWithPreviousShardBestStateError
	ValidateBlockWithPreviousBeaconBestStateError
	BackUpShardStateError
	BackupCurrentBeaconStateError
	ProcessAutoStakingError
	ProcessPortalRelayingError
	GetTotalLockedCollateralError
	InsertShardBlockError
	GetShardBlockHeightByHashError
	GetShardBlockByHashError
	ResponsedTransactionFromBeaconInstructionsError
	UpdateBeaconCommitteeStateError
	UpdateShardCommitteeStateError
	BuildIncurredInstructionError
	ReturnStakingInstructionHandlerError
	CommitteeFromBlockNotFoundError
)

var ErrCodeMessage = map[int]struct {
	Code    int
	message string
}{
	UnExpectedError:                                   {-1000, "Unexpected error"},
	UpdateMerkleTreeForBlockError:                     {-1001, "updateShardBestState Merkle Commitments Tree For Block is failed"},
	UnmashallJsonShardBlockError:                      {-1002, "Unmarshall Json Shard Block Is Failed"},
	MashallJsonShardBlockError:                        {-1003, "Marshall Json Shard Block Is Failed"},
	UnmashallJsonShardBestStateError:                  {-1004, "Unmarshall Json Shard Best State Is Failed"},
	MashallJsonShardBestStateError:                    {-1005, "Marshall Json Shard Best State Is Failed"},
	UnmashallJsonBeaconBlockError:                     {-1006, "Unmarshall Json Beacon Block Is Failed"},
	MashallJsonBeaconBlockError:                       {-1007, "Marshall Json Beacon Block Is Failed"},
	UnmashallJsonBeaconBestStateError:                 {-1008, "Unmarshall Json Beacon Best State Is Failed"},
	MashallJsonBeaconBestStateError:                   {-1009, "Marshall Json Beacon Best State Is Failed"},
	CanNotCheckDoubleSpendError:                       {-1010, "CanNotCheckDoubleSpend Error"},
	HashError:                                         {-1011, "Hash error"},
	WrongVersionError:                                 {-1012, "Version error"},
	WrongBlockHeightError:                             {-1013, "Wrong Block Height Error"},
	DatabaseError:                                     {-1014, "Database Error"},
	EpochError:                                        {-1015, "Epoch Error"},
	WrongTimestampError:                               {-1016, "Timestamp Error"},
	InstructionHashError:                              {-1017, "Instruction Hash Error"},
	ShardStateHashError:                               {-1018, "ShardState Hash Error"},
	RandomError:                                       {-1019, "Random Number Error"},
	VerificationError:                                 {-1020, "Verify Block Error"},
	BeaconError:                                       {-1021, "Beacon Error"},
	CrossShardBlockError:                              {-1022, "CrossShardBlockError"},
	SignatureError:                                    {-1023, "Signature Error"},
	CandidateError:                                    {-1024, "Candidate Error"},
	ShardIDError:                                      {-1025, "ShardID Error"},
	ProducerError:                                     {-1026, "Producer Error"},
	ShardStateError:                                   {-1027, "Shard State Error"},
	TransactionFromNewBlockError:                      {-1028, "Transaction invalid"},
	GenerateInstructionError:                          {-1029, "Instruction Error"},
	SwapError:                                         {-1030, "Swap Error"},
	MashallJsonError:                                  {-1031, "MashallJson Error"},
	DuplicateShardBlockError:                          {-1032, "Duplicate Shard Block Error"},
	CommitteeOrValidatorError:                         {-1033, "Committee or Validator Error"},
	ShardBlockSanityError:                             {-1034, "Shard Block Sanity Data Error"},
	StoreIncomingCrossShardError:                      {-1035, "Store Incoming Cross Shard Block Error"},
	DeleteIncomingCrossShardError:                     {-1036, "Delete Incoming Cross Shard Block Error"},
	WrongShardIDError:                                 {-1037, "Wrong Shard ID Error"},
	CloneShardBestStateError:                          {-1038, "Clone Shard Best State Error"},
	CloneBeaconBestStateError:                         {-1039, "Clone Beacon Best State Error"},
	ShardBestStateNotCompatibleError:                  {-1040, "New Block and Shard Best State Is NOT Compatible"},
	RegisterEstimatorFeeError:                         {-1041, "Register Fee Estimator Error"},
	FetchPreviousBlockError:                           {-1042, "Failed To Fetch Previous Block Error -> need to make request a new Pre Block for checking fork/revert"},
	TransactionRootHashError:                          {-1043, "Transaction Root Hash Error"},
	ShardTransactionRootHashError:                     {-1044, "Shard Transaction Root Hash Error"},
	CrossShardTransactionRootHashError:                {-1045, "Cross Shard Transaction Root Hash Error"},
	FetchBeaconBlocksError:                            {-1046, "Fetch Beacon Blocks Error"},
	FetchBeaconBlockHashError:                         {-1047, "Fetch Beacon Block Hash Error"},
	FetchBeaconBlockError:                             {-1048, "Fetch Beacon Block Error"},
	WrongBlockTotalFeeError:                           {-1049, "Wrong Block Total Fee Error"},
	ShardIntructionFromTransactionAndInstructionError: {-1050, "Shard Instruction From Transaction And Instruction Error"},
	InstructionsHashError:                             {-1051, "Instruction Hash Error"},
	FlattenAndConvertStringInstError:                  {-1052, "Flatten And Convert String Instruction Error"},
	InstructionMerkleRootError:                        {-1053, "Instruction Merkle Root Error"},
	BeaconBlockNotCompatibleError:                     {-1054, "Beacon Block Not Compatible Error"},
	SwapInstructionError:                              {-1055, "Swap Instruction Error"},
	TransactionCreatedByMinerError:                    {-1056, "Transaction Created By Miner Error"},
	ResponsedTransactionWithMetadataError:             {-1057, "Responsed Transaction With Metadata Error"},
	UnmashallJsonShardCommitteesError:                 {-1058, "Unmashall Json Shard Committees Error"},
	MashallJsonShardCommitteesError:                   {-1059, "Mashall Json Shard Committees Error"},
	VerifyCrossShardBlockError:                        {-1060, "Verify Cross Shard Block Error"},
	NextCrossShardBlockError:                          {-1061, "Next Cross Shard Block Error"},
	FetchShardCommitteeError:                          {-1062, "Fetch Shard Committee Error"},
	CrossTransactionHashError:                         {-1063, "Cross Transaction Hash Error"},
	VerifyCrossShardCustomTokenError:                  {-1064, "Verify Cross Shard Custom Token Error"},
	ShardCommitteeRootHashError:                       {-1065, "Shard Committee Root Hash Error"},
	ShardPendingValidatorRootHashError:                {-1066, "Shard Pending Validator Root Hash Error"},
	StoreShardBlockError:                              {-1067, "Store Shard Block Error"},
	StoreBestStateError:                               {-1068, "Store Shard Shard Best State Error"},
	FetchAndStoreTransactionError:                     {-1069, "Fetch And Store Transaction Error"},
	FetchAndStoreCrossTransactionError:                {-1070, "Fetch And Store Cross Transaction Error"},
	RemoveCommitteeRewardError:                        {-1071, "Remove Committee Reward Error"},
	StoreBurningConfirmError:                          {-1072, "Store Burning Confirm Error"},
	SwapValidatorError:                                {-1073, "Swap Validator Error"},
	CrossShardBitMapError:                             {-1074, "Cross Shard Bitmap Error"},
	ShardCommitteeLengthAndCommitteeIndexError:        {-1075, "Shard Committee Length And Committee Index Error"},
	BuildRewardInstructionError:                       {-1076, "Build Reward Transaction Error"},
	GenerateBeaconCommitteeAndValidatorRootError:      {-1077, "Generate Beacon Committee And Validator Root Error"},
	GenerateShardCommitteeAndValidatorRootError:       {-1078, "Generate Shard Committee And Validator Root Error"},
	GenerateBeaconCandidateRootError:                  {-1079, "Generate Beacon Candidate Root Error"},
	GenerateShardCandidateRootError:                   {-1080, "Generate Shard Candidate Root Error"},
	GenerateShardStateError:                           {-1081, "Generate Shard State Error"},
	GenerateShardCommitteeError:                       {-1082, "Generate Shard Committee Root Error"},
	GenerateShardPendingValidatorError:                {-1083, "Generate Shard Pending Validator Root Error"},
	ProduceSignatureError:                             {-1084, "Produce Signature Error"},
	BeaconBestStateBestBlockNotCompatibleError:        {-1085, "New Beacon Block and Beacon Best State Is NOT Compatible"},
	BeaconBlockProducerError:                          {-1086, "Beacon Block Producer Error"},
	BeaconBlockSignatureError:                         {-1087, "Beacon Block Signature Error"},
	WrongEpochError:                                   {-1088, "Wrong Epoch Error"},
	GenerateInstructionHashError:                      {-1089, "Generate Instruction Hash Error"},
	ShardStateHeightError:                             {-1090, "Generate Instruction Hash Error"},
	ShardStateCrossShardBitMapError:                   {-1091, "Shard State Cross Shard BitMap Error"},
	BeaconCommitteeLengthAndCommitteeIndexError:       {-1092, "Shard Committee Length And Committee Index Error"},
	ShardBlockSignatureError:                          {-1093, "Shard Block Signature Error"},
	ShardBestStateBeaconHeightNotCompatibleError:      {-1094, "Shard BestState Beacon Height Not Compatible Error"},
	BeaconBestStateBestShardHeightNotCompatibleError:  {-1095, "Beacon BestState Best Shard Height Not Compatible Error"},
	BeaconCommitteeAndPendingValidatorRootError:       {-1096, "Beacon Committee And Pending Validator Root Hash Error"},
	ShardCommitteeAndPendingValidatorRootError:        {-1097, "Shard Committee And Pending Validator Root Hash Error"},
	ShardCandidateRootError:                           {-1098, "Shard Candidate Root Hash Error"},
	ProcessRandomInstructionError:                     {-1099, "Process Random Instruction Error"},
	ProcessSwapInstructionError:                       {-1100, "Process Swap Instruction Error"},
	AssignValidatorToShardError:                       {-1101, "Assign Validator To Shard Error"},
	ShuffleBeaconCandidateError:                       {-1102, "Shuffle Beacon Candidate Error"},
	CleanBackUpError:                                  {-1103, "Clean Back Up Error"},
	BackUpBestStateError:                              {-1104, "Back Up Best State Error"},
	ProcessBridgeInstructionError:                     {-1105, "Process Bridge Instruction Error"},
	UpdateDatabaseWithBlockRewardInfoError:            {-1106, "Update Database With Block Reward Info Error"},
	CreateCrossShardBlockError:                        {-1107, "Create Cross Shard Block Error"},
	VerifyCrossShardBlockShardTxRootError:             {-1108, "Verify Cross Shard Block ShardTxRoot Error"},
	GetStakingTransactionError:                        {-1110, "Get Staking Transaction Error"},
	DecodeHashError:                                   {-1111, "Decode Hash Error"},
	GetTransactionFromDatabaseError:                   {-1112, "Get Transaction From Database Error"},
	FetchShardBlockError:                              {-1113, "Fetch Shard Block Error"},
	WalletKeySerializedError:                          {-1114, "Wallet Key Serialized Error"},
	InitSalaryTransactionError:                        {-1115, "Init Salary Transaction Error"},
	RemoveOldDataAfterProcessingError:                 {-1116, "Remove Old Data After Processing Error"},
	WrongMetadataTypeError:                            {-1117, "Wrong Metadata Type Error"},
	StakeInstructionError:                             {-1118, "Stake Instruction Error"},
	StoreRewardReceiverByHeightError:                  {-1119, "Store Reward Receiver By Height Error"},
	CreateNormalTokenTxForCrossShardError:             {-1120, "Create Normal Token Tx For Cross Shard Error"},
	SnapshotCommitteeError:                            {-1121, "Snapshot Committee Error"},
	ExtractPublicKeyFromCommitteeKeyListError:         {-1122, "Extract Public Key From Committee Key List"},
	PendingValidatorRootError:                         {-1123, "Pending Validator Root Error"},
	CommitteeHashError:                                {-1124, "Committee Root Hash Error"},
	StakingTxHashError:                                {-1124, "Staking Tx Root Hash Error"},
	StopAutoStakingRequestHashError:                   {-1125, "Stop Auto Staking Request Root Hash Error"},
	StopAutoStakingMetadataError:                      {-1126, "StopAutoStake Metadata Error"},
	AutoStakingRootHashError:                          {-1127, "Auto Re Staking Root Hash Error"},
	FetchAllCommitteeValidatorCandidateError:          {-1128, "Fetch All Committee Validator Candidate Error"},
	BackupFromTxViewPointError:                        {-1129, "Create Backup From TxViewPoint Error"},
	BackupFromCrossTxViewPointError:                   {-1130, "Create Backup From CrossTxViewPoint Error"},
	BackupDatabaseFromBeaconInstructionError:          {-1131, "Backup Database From BeaconInstruction Error"},
	SnapshotRewardReceiverError:                       {-1132, "Snapshot Reward Receiver Error"},
	StoreAutoStakingByHeightError:                     {-1133, "Store Auto Staking By Height Error"},
	FetchAutoStakingByHeightError:                     {-1134, "Fetch Auto Staking By Height Error"},
	ProcessSlashingError:                              {-1135, "Process slashing Error"},
	ConvertCommitteePubKeyToBase58Error:               {-1136, "Convert committee pub key to base58 Error"},
	ConsensusIsOngoingError:                           {-1137, "Consensus Is Ongoing Error"},
	GetShardBlocksForBeaconProcessError:               {-1138, "Get Shard To Beacon Blocks Error"},
	RevertStateError:                                  {-1139, "Revert State Error"},
	NotEnoughRewardError:                              {-1140, "Not enough reward Error"},
	InitPDETradeResponseTransactionError:              {-1141, "Init PDE trade response tx Error"},
	ProcessPDEInstructionError:                        {-1142, "Process PDE instruction Error"},
	ProcessPortalInstructionError:                     {-1143, "Process Portal instruction Error"},
	InitBeaconStateError:                              {-1144, "Init Beacon State Error"},
	InitShardStateError:                               {-1144, "Init Shard State Error"},
	ProcessSalaryInstructionsError:                    {-1145, "Proccess Salary Instruction Error"},
	GetShardIDFromTxError:                             {-1146, "Get ShardID From Tx Error"},
	GetValueFromTxError:                               {-1147, "Get Value From Tx Error"},
	ValidateBlockWithPreviousShardBestStateError:      {-1148, "Validate Block With Previous Shard Best State Error"},
	BackUpShardStateError:                             {-1149, "Back Up Shard State Error"},
	ValidateBlockWithPreviousBeaconBestStateError:     {-1150, "Validate Block With Previous Beacon Best State Error"},
	BackupCurrentBeaconStateError:                     {-1151, "Backup Current Beacon State Error"},
	ProcessAutoStakingError:                           {-1152, "Process Auto Staking Error"},
	ProcessPortalRelayingError:                        {-1153, "Process Portal Relaying Error"},
	InsertShardBlockError:                             {-1154, "Insert Shard Block Error"},
	GetShardBlockHeightByHashError:                    {-1155, "Get Shard Block Height By Hash Error"},
	GetShardBlockByHashError:                          {-1156, "Get Shard Block By Hash Error"},
	ProcessInstructionFromBeaconError:                 {-1157, "Process Instruction From Beacon Error"},
	ShardStakingTxRootHashError:                       {-1158, "Build Shard StakingTX error"},
	BuildIncurredInstructionError:                     {-1159, "Build Incurred Instructions error"},
	ReturnStakingInstructionHandlerError:              {-1160, "Return Staking Instruction Handler error"},
	CommitteeFromBlockNotFoundError:                   {-1161, "Committee From Beacon Block Not Found Error"},
	GetListOutputCoinsByKeysetError:                   {-2000, "Get List Output Coins By Keyset Error"},
	GetTotalLockedCollateralError:                     {-3000, "Get Total Locked Collateral Error"},
	ResponsedTransactionFromBeaconInstructionsError:   {-3100, "Build Transaction Response From Beacon Instructions Error"},
	UpdateBeaconCommitteeStateError:                   {-4000, "Update Beacon Committee State Error"},
	UpdateShardCommitteeStateError:                    {-4001, "Update Shard Committee State Error"},
}

type BlockChainError struct {
	Code    int
	Message string
	err     error
}

func (e BlockChainError) Error() string {
	return fmt.Sprintf("%d: %s \n %+v", e.Code, e.Message, e.err)
}

func NewBlockChainError(key int, err error) *BlockChainError {
	return &BlockChainError{
		Code:    ErrCodeMessage[key].Code,
		Message: ErrCodeMessage[key].message,
		err:     errors.Wrap(err, ErrCodeMessage[key].message),
	}
}
