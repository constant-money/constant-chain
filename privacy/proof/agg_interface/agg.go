package agg_interface

type AggregatedRangeProof interface {
	Init()
	IsNil() bool
	Bytes() []byte
	SetBytes([]byte) error
	// Verify() (bool, error)
}

// type AggregatedRangeProofV1 = aggregatedrange.AggregatedRangeProof
// type AggregatedRangeProofV2 = bulletproofs.AggregatedRangeProof
