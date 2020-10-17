package metadata

import (
	"bytes"

	"github.com/incognitochain/incognito-chain/common"
)

type AccumulatedValues struct {
	UniqETHTxsUsed   [][]byte
	DBridgeTokenPair map[string][]byte
	CBridgeTokens    []*common.Hash
}

func (ac AccumulatedValues) CanProcessTokenPair(
	externalTokenID []byte,
	incTokenID common.Hash,
) (bool, error) {
	incTokenIDStr := incTokenID.String()
	for _, tokenID := range ac.CBridgeTokens {
		if bytes.Equal(tokenID[:], incTokenID[:]) {
			return false, nil
		}
	}
	bridgeTokenPair := ac.DBridgeTokenPair
	if existedExtTokenID, found := bridgeTokenPair[incTokenIDStr]; found {
		if bytes.Equal(existedExtTokenID, externalTokenID) {
			return true, nil
		}
		return false, nil
	}
	for _, existedExtTokenID := range bridgeTokenPair {
		if !bytes.Equal(existedExtTokenID, externalTokenID) {
			continue
		}
		return false, nil
	}
	return true, nil
}

func (ac AccumulatedValues) CanProcessCIncToken(
	incTokenID common.Hash,
) bool {
	incTokenIDStr := incTokenID.String()
	_, found := ac.DBridgeTokenPair[incTokenIDStr]
	return !found
}
