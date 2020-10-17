package metadata

import (
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

// RelayingHeader - relaying header chain
// metadata - create normal tx with this metadata
type RelayingHeader struct {
	MetadataBase
	IncogAddressStr string
	Header          string
	BlockHeight     uint64
}

// RelayingHeaderAction - shard validator creates instruction that contain this action content
// it will be append to ShardToBeaconBlock
type RelayingHeaderAction struct {
	Meta    RelayingHeader
	TxReqID common.Hash
	ShardID byte
}

// RelayingHeaderContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and refund status
type RelayingHeaderContent struct {
	IncogAddressStr string
	Header          string
	BlockHeight     uint64
	TxReqID         common.Hash
}

// RelayingHeaderStatus - Beacon tracks status of custodian deposit tx into db
type RelayingHeaderStatus struct {
	Status          byte
	IncogAddressStr string
	Header          string
	BlockHeight     uint64
}

func NewRelayingHeader(
	metaType int,
	incognitoAddrStr string,
	header string,
	blockHeight uint64,
) (*RelayingHeader, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	relayingHeader := &RelayingHeader{
		IncogAddressStr: incognitoAddrStr,
		Header:          header,
		BlockHeight:     blockHeight,
	}
	relayingHeader.MetadataBase = metadataBase
	return relayingHeader, nil
}

func (rh RelayingHeader) Hash() *common.Hash {
	record := rh.MetadataBase.Hash().String()
	record += rh.IncogAddressStr
	record += rh.Header
	record += strconv.Itoa(int(rh.BlockHeight))

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}