package metadata

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/pkg/errors"
	"strconv"
)

func ParseMetadata(meta interface{}) (basemeta.Metadata, error) {
	if meta == nil {
		return nil, nil
	}

	mtTemp := map[string]interface{}{}
	metaInBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(metaInBytes, &mtTemp)
	if err != nil {
		return nil, err
	}
	var md basemeta.Metadata
	switch int(mtTemp["Type"].(float64)) {
	case basemeta.PortalBurnPTokenMeta:
		md = &PortalUnshieldRequest{}
	default:
		Logger.log.Debug("[db] parse meta err: %+v\n", meta)
		return nil, errors.Errorf("Could not parse metadata with type: %d", int(mtTemp["Type"].(float64)))
	}

	err = json.Unmarshal(metaInBytes, &md)
	if err != nil {
		return nil, err
	}
	return md, nil
}

// TODO: add more meta data types
var portalMetasV4 = []string{
	strconv.Itoa(basemeta.PortalUnshieldBatchingMeta),
}

func HasPortalInstructionsV4(instructions [][]string) bool {
	for _, inst := range instructions {
		for _, meta := range portalMetasV4 {
			if len(inst) > 0 && inst[0] == meta {
				return true
			}
		}
	}
	return false
}

func IsRequireBeaconSigForPortalV4Meta(inst []string) bool {
	isExist, _ := common.SliceExists(portalMetasV4, inst[0])
	return isExist
}