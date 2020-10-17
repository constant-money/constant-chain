package metadata

import (
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