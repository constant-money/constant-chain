package metadata

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/pkg/errors"
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
		md = &PortalBurnPToken{}
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
