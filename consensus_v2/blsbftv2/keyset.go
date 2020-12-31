package blsbftv2

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common/base58"
	signatureschemes2 "github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

func (e *BLSBFT_V2) LoadUserKeys(miningKey []signatureschemes2.MiningKey) error {
	e.UserKeySet = miningKey
	return nil
}

func (e *BLSBFT_V2) GetUserPublicKey() *incognitokey.CommitteePublicKey {
	if e.UserKeySet != nil && len(e.UserKeySet) > 0 {
		return e.UserKeySet[0].GetPublicKey()
	}
	return nil
}

func (e BLSBFT_V2) SignData(data []byte) (string, error) {
	if e.UserKeySet != nil && len(e.UserKeySet) > 0 {
		result, err := e.UserKeySet[0].BriSignData(data)
		if err != nil {
			return "", NewConsensusError(SignDataError, err)
		}
		return base58.Base58Check{}.Encode(result, common.Base58Version), nil
	}
	return "", NewConsensusError(SignDataError, fmt.Errorf("No validator key"))

}

func CombineVotes(votes map[string]*BFTVote, committee []string) (aggSig []byte, brigSigs [][]byte, validatorIdx []int, err error) {
	var blsSigList [][]byte
	for validator, vote := range votes {
		if vote.IsValid == 1 {
			validatorIdx = append(validatorIdx, common.IndexOfStr(validator, committee))
		}
	}
	sort.Ints(validatorIdx)
	for _, idx := range validatorIdx {
		blsSigList = append(blsSigList, votes[committee[idx]].BLS)
		brigSigs = append(brigSigs, votes[committee[idx]].BRI)
	}
	aggSig, err = blsmultisig.Combine(blsSigList)
	if err != nil {
		return nil, nil, nil, NewConsensusError(CombineSignatureError, err)
	}
	return
}
