package common

const PortalBTCIDStr = "ef5947f70ead81a76a53c7c8b7317dd5245510c665d3a13921dc9a581188728b"
const PortalBNBIDStr = "6abd698ea7ddd1f98b1ecaaddab5db0453b8363ff092f0d8d7d4c6b1155fb693"

var PortalV4SupportedIncTokenIDs = []string{
	PortalBTCIDStr, // pBTC
	PortalBNBIDStr, // pBNB
}

const (
	// status of unshield processing - used to store db
	PortalUnshieldReqWaitingStatus   = 0
	PortalUnshieldReqProcessedStatus = 1
	PortalUnshieldReqCompletedStatus = 2

	// status of batching unshield processing by batchID - used to store db
	PortalBatchUnshieldReqProcessedStatus = 0
	PortalBatchUnshieldReqCompletedStatus = 1

	// status of portal request - used to store db
	PortalRequestRejectedStatus = 0
	PortalRequestAcceptedStatus = 1

	// status of portal request - used to append to beacon instructions
	PortalRequestAcceptedChainStatus = "accepted"
	PortalRequestRejectedChainStatus = "rejected"

	PortalProducerInstSuccessChainStatus = "success"
	PortalProducerInstFailedChainStatus  = "failed"
)