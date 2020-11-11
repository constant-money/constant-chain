package consensustypes

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/consensus/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/consensus/signatureschemes/bridgesig"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type MiningKey struct {
	PriKey map[string][]byte
	PubKey map[string][]byte
}

func (miningKey *MiningKey) GetPublicKey() incognitokey.CommitteePublicKey {
	key := incognitokey.CommitteePublicKey{}
	key.MiningPubKey = make(map[string][]byte)
	key.MiningPubKey[common.BlsConsensus] = miningKey.PubKey[common.BlsConsensus]
	key.MiningPubKey[common.BridgeConsensus] = miningKey.PubKey[common.BridgeConsensus]
	return key
}

func (miningKey *MiningKey) GetPublicKeyBase58() string {
	key := miningKey.GetPublicKey()
	keyBytes, err := json.Marshal(key)
	if err != nil {
		return ""
	}
	return base58.Base58Check{}.Encode(keyBytes, common.ZeroByte)
}

func (miningKey *MiningKey) BLSSignData(
	data []byte,
	selfIdx int,
	committee []blsmultisig.PublicKey,
) (
	[]byte,
	error,
) {
	sigBytes, err := blsmultisig.Sign(data, miningKey.PriKey[common.BlsConsensus], selfIdx, committee)
	if err != nil {
		return nil, err
	}
	return sigBytes, nil
}

func (miningKey *MiningKey) BriSignData(
	data []byte,
) (
	[]byte,
	error,
) {
	sig, err := bridgesig.Sign(miningKey.PriKey[common.BridgeConsensus], data)
	if err != nil {
		return nil, err
	}
	return sig, nil
}
