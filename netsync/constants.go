package netsync

import "time"

const (
	workers                = 5
	messageLiveTime        = 40 * time.Second  // in second
	messageCleanupInterval = 300 * time.Second //in second
)

// block type
// Please modify peerv2/constants.go if you change something here.
const (
	blockShard = 0
	crossShard = 1
)
