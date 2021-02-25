package utils

const (
	// txVersion is the current latest supported transaction version.
	CurrentTxVersion                 = 2
	TxVersion0Number                 = 0
	TxVersion1Number                 = 1
	TxVersion2Number                 = 2
	TxConversionNumber               = 1
	ValidateTimeForOneoutOfManyProof = 1574985600 // GMT: Friday, November 29, 2019 12:00:00 AM
)

const (
	CustomTokenInit = iota
	CustomTokenTransfer
	CustomTokenCrossShard
)

const (
	NormalCoinType = iota
	CustomTokenPrivacyType
)

const (
	MaxSizeInfo   = 512
	MaxSizeUint32 = (1 << 32) - 1
	MaxSizeByte   = (1 << 8) - 1
)
