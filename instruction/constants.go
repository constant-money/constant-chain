package instruction

const (
	SWAP_SHARD_ACTION      = "swapshard"
	SWAP_ACTION            = "swap"
	RANDOM_ACTION          = "random"
	STAKE_ACTION           = "stake"
	ASSIGN_ACTION          = "assign"
	STOP_AUTO_STAKE_ACTION = "stopautostake"
	SET_ACTION             = "set"
	RETURN_ACTION          = "return"
	UNSTAKE_ACTION         = "unstake"
	SHARD_INST             = "shard"
	BEACON_INST            = "beacon"
	SPLITTER               = ","
	TRUE                   = "true"
	FALSE                  = "false"
)

const (
	BEACON_CHAIN_ID = -1
)

//Swap Instruction Sub Type
const (
	SWAP_BY_END_EPOCH = iota
	SWAP_BY_SLASHING
	SWAP_BY_INCREASE_COMMITTEES_SIZE
	SWAP_BY_DECREASE_COMMITTEES_SIZE
)
