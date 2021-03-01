package metadata

import (
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
)

type MetadataBase struct {
	Type int
	Sig []byte
}

func (mb *MetadataBase) SetSig(sig []byte) { mb.Sig = sig }

func (mb MetadataBase) GetSig() []byte { return mb.Sig }

func (mb *MetadataBase) ShouldSignMetaData() bool { return false }

func NewMetadataBase(thisType int) *MetadataBase {
	return &MetadataBase{Type: thisType, Sig: []byte{}}
}

func (mb MetadataBase) IsMinerCreatedMetaType() bool {
	metaType := mb.GetType()
	for _, mType := range minerCreatedMetaTypes {
		if metaType == mType {
			return true
		}
	}
	return false
}

func (mb *MetadataBase) CalculateSize() uint64 {
	return 0
}

func (mb *MetadataBase) Validate() error {
	return nil
}

func (mb *MetadataBase) Process() error {
	return nil
}

func (mb MetadataBase) GetType() int {
	return mb.Type
}

func (mb MetadataBase) Hash() *common.Hash {
	record := strconv.Itoa(mb.Type)
	data := []byte(record)
	hash := common.HashH(data)
	return &hash
}

func (mb MetadataBase) HashWithoutSig() *common.Hash {
	return mb.Hash()
}

func (mb MetadataBase) CheckTransactionFee(tx Transaction, minFeePerKbTx uint64, beaconHeight int64, stateDB *statedb.StateDB) bool {
	if tx.GetMetadataType() == ConvertingRequestMeta || tx.GetMetadataType() == ConvertingResponseMeta {
		return true
	}
	// normal privacy tx
	txFee := tx.GetTxFee()
	fullFee := minFeePerKbTx * tx.GetTxActualSize()
	return !(txFee < fullFee)
}

func (mb *MetadataBase) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	return [][]string{}, nil
}

func (mb MetadataBase) VerifyMinerCreatedTxBeforeGettingInBlock(mintData *MintData, shardID byte, tx Transaction, chainRetriever ChainRetriever, ac *AccumulatedValues, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever) (bool, error) {
	return true, nil
}
