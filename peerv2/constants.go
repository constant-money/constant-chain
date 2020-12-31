package peerv2

import "time"

// block type
const (
	blockShard         = 0
	crossShard         = 1
	shardToBeacon      = 2
	blockbeacon        = 3
	MaxCallRecvMsgSize = 50 << 20 // 50 MBs per gRPC response
	MaxConnectionRetry = 6        // connect to new highway after 6 failed retries
)

var (
	RegisterTimestep          = 2 * time.Second  // Re-register to highway
	ReconnectHighwayTimestep  = 1 * time.Second  // Check libp2p connection
	UpdateHighwayListTimestep = 10 * time.Minute // RPC to update list of highways
	RequesterDialTimestep     = 10 * time.Second // Check gRPC connection
	MaxTimePerRequest         = 30 * time.Second // Time per request
	DialTimeout               = 5 * time.Second  // Timeout for dialing's context
	RequesterKeepaliveTime    = 10 * time.Minute
	RequesterKeepaliveTimeout = 30 * time.Second
	defaultMaxBlkReqPerPeer   = 900
	defaultMaxBlkReqPerTime   = 900

	IgnoreRPCDuration = 60 * time.Minute  // Ignore an address after a failed RPC
	IgnoreHWDuration  = 360 * time.Minute // Ignore a highway when cannot connect
)
