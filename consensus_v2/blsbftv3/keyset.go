package blsbftv3

import (
	"fmt"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/consensus/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/consensus/signatureschemes/bridgesig"
	"github.com/incognitochain/incognito-chain/consensus_v2/consensustypes"
	signatureschemes2 "github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wallet"
)

func (e *BLSBFT_V3) LoadUserKeys(miningKey []signatureschemes2.MiningKey) error {
	e.UserKeySet = miningKey
	return nil
}

func (e *BLSBFT_V3) GetUserPublicKey() *incognitokey.CommitteePublicKey {
	if e.UserKeySet != nil && len(e.UserKeySet) > 0 {
		return e.UserKeySet[0].GetPublicKey()
	}
	return nil
}

func (e BLSBFT_V3) SignData(data []byte) (string, error) {
	if e.UserKeySet != nil && len(e.UserKeySet) > 0 {
		result, err := e.UserKeySet[0].BriSignData(data)
		if err != nil {
			return "", NewConsensusError(SignDataError, err)
		}
		return base58.Base58Check{}.Encode(result, common.Base58Version), nil
	}
	return "", NewConsensusError(SignDataError, fmt.Errorf("No validator key"))

}

func combineVotes(votes map[string]BFTVote, committee []string) (aggSig []byte, brigSigs [][]byte, validatorIdx []int, err error) {
	var blsSigList [][]byte
	for validator, _ := range votes {
		if index := common.IndexOfStr(validator, committee); index != -1 {
			validatorIdx = append(validatorIdx, index)
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

func GetMiningKeyFromPrivateSeed(privateSeed string) (*consensustypes.MiningKey, error) {
	var miningKey consensustypes.MiningKey
	privateSeedBytes, _, err := base58.Base58Check{}.Decode(privateSeed)
	if err != nil {
		return nil, NewConsensusError(LoadKeyError, err)
	}

	blsPriKey, blsPubKey := blsmultisig.KeyGen(privateSeedBytes)

	miningKey.PriKey = map[string][]byte{}
	miningKey.PubKey = map[string][]byte{}
	miningKey.PriKey[common.BlsConsensus] = blsmultisig.SKBytes(blsPriKey)
	miningKey.PubKey[common.BlsConsensus] = blsmultisig.PKBytes(blsPubKey)
	bridgePriKey, bridgePubKey := bridgesig.KeyGen(privateSeedBytes)
	miningKey.PriKey[common.BridgeConsensus] = bridgesig.SKBytes(&bridgePriKey)
	miningKey.PubKey[common.BridgeConsensus] = bridgesig.PKBytes(&bridgePubKey)

	return &miningKey, nil
}

func LoadUserKeyFromIncPrivateKey(privateKey string) (string, error) {
	wl, err := wallet.Base58CheckDeserialize(privateKey)
	if err != nil {
		return "", NewConsensusError(LoadKeyError, err)
	}
	privateSeedBytes := common.HashB(common.HashB(wl.KeySet.PrivateKey))
	if err != nil {
		return "", NewConsensusError(LoadKeyError, err)
	}
	privateSeed := base58.Base58Check{}.Encode(privateSeedBytes, common.Base58Version)
	return privateSeed, nil
}
