package operation

import (
	"github.com/pkg/errors"
)

const (
	PedersenPrivateKeyIndex = byte(0x00)
	PedersenValueIndex      = byte(0x01)
	PedersenSndIndex        = byte(0x02)
	PedersenShardIDIndex    = byte(0x03)
	PedersenRandomnessIndex = byte(0x04)
)

var PedCom PedersenCommitment = NewPedersenParams()

// PedersenCommitment represents the parameters for the commitment
type PedersenCommitment struct {
	G []*Point // generators
	// G[0]: public key
	// G[1]: Value
	// G[2]: SNDerivator
	// G[3]: ShardID
	// G[4]: Randomness
}

var GBase, HBase, RandomBase *Point

func NewPedersenParams() PedersenCommitment {
	var pcm PedersenCommitment
	const capacity = 5 // fixed value = 5
	pcm.G = make([]*Point, capacity)
	pcm.G[0] = new(Point).ScalarMultBase(new(Scalar).FromUint64(1))

	for i := 1; i < len(pcm.G); i++ {
		pcm.G[i] = HashToPointFromIndex(int64(i), CStringBulletProof)
	}
	GBase = new(Point).Set(pcm.G[1])
	HBase = new(Point).Set(pcm.G[4])
	return pcm
}

// CommitAll commits a list of PCM_CAPACITY value(s)
func (com PedersenCommitment) CommitAll(openings []*Scalar) (*Point, error) {
	if len(openings) != len(com.G) {
		return nil, errors.New("invalid length of openings to commit")
	}

	commitment := new(Point).ScalarMult(com.G[0], openings[0])

	for i := 1; i < len(com.G); i++ {
		commitment.Add(commitment, new(Point).ScalarMult(com.G[i], openings[i]))
	}
	return commitment, nil
}

// CommitAtIndex commits specific value with index and returns 34 bytes
// g^v x h^rand
func (com PedersenCommitment) CommitAtIndex(value, rand *Scalar, index byte) *Point {
	//commitment := new(Point).ScalarMult(com.G[index], value)
	//commitment.Add(commitment, new(Point).ScalarMult(com.G[PedersenRandomnessIndex], rand))
	//
	//return commitment
	return new(Point).AddPedersen(value, com.G[index], rand, com.G[PedersenRandomnessIndex])
}
