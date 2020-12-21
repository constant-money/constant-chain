package consensus_v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common/base58"
	signatureschemes2 "github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/bridgesig"
	"github.com/incognitochain/incognito-chain/wallet"
	"strings"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus_v2/blsbft"
	"github.com/incognitochain/incognito-chain/consensus_v2/blsbftv2"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

func GetMiningKeyFromPrivateSeed(privateSeed string) (*signatureschemes2.MiningKey, error) {
	var miningKey signatureschemes2.MiningKey
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

func (engine *Engine) GetCurrentMiningPublicKey() (publickey string, keyType string) {
	if engine != nil && engine.GetMiningPublicKeys() != nil {
		name := engine.consensusName
		pubkey := engine.GetMiningPublicKeys().GetMiningKeyBase58(name)
		return pubkey, name
	}
	return "", ""
}

func (engine *Engine) GetMiningPublicKeyByConsensus(consensusName string) (publickey string, err error) {
	keyBytes := map[string][]byte{}
	if engine != nil && engine.currentMiningProcess != nil {
		keytype := engine.currentMiningProcess.GetConsensusName()
		lightweightKey, exist := engine.GetMiningPublicKeys().MiningPubKey[common.BridgeConsensus]
		if !exist {
			return "", blsbft.NewConsensusError(blsbft.LoadKeyError, errors.New("Lightweight key not found"))
		}
		keyBytes[keytype], exist = engine.GetMiningPublicKeys().MiningPubKey[keytype]
		if !exist {
			return "", blsbft.NewConsensusError(blsbft.LoadKeyError, errors.New("Key not found"))
		}
		keyBytes[common.BridgeConsensus] = lightweightKey
		// pubkey := engine.userMiningPublicKeys[keytype]
		// return pubkey.GetMiningKeyBase58(keytype), keytype
	}
	res, err := json.Marshal(keyBytes)
	if err != nil {
		return "", blsbft.NewConsensusError(blsbft.UnExpectedError, err)
	}
	return string(res), nil
}

func (s *Engine) GetMiningPublicKeys() *incognitokey.CommitteePublicKey {
	return s.userMiningPublicKeys
}

func (s *Engine) GetNodeMiningPublicKeys() (userPks []*incognitokey.CommitteePublicKey) {
	for _, v := range s.validators {
		userPks = append(userPks, v.MiningKey.GetPublicKey())
	}
	return userPks
}

func (engine *Engine) GetAllMiningPublicKeys() []string {
	return []string{}
}

func (engine *Engine) SignDataWithCurrentMiningKey(
	data []byte,
) (
	publicKeyStr string,
	publicKeyType string,
	signature string,
	err error,
) {
	publicKeyStr = ""
	publicKeyType = ""
	signature = ""
	if engine != nil && engine.currentMiningProcess != nil {
		publicKeyType = engine.config.Blockchain.GetBeaconBestState().ConsensusAlgorithm
		publicKeyStr, err = engine.GetMiningPublicKeyByConsensus(publicKeyType)
		if err != nil {
			return
		}
		signature, err = engine.currentMiningProcess.SignData(data)
	}
	return
}

func (engine *Engine) VerifyData(data []byte, sig string, publicKey string, consensusType string) error {
	mapPublicKey := map[string][]byte{}
	err := json.Unmarshal([]byte(publicKey), &mapPublicKey)
	if err != nil {
		return blsbft.NewConsensusError(blsbft.LoadKeyError, err)
	}
	return engine.currentMiningProcess.ValidateData(data, sig, string(mapPublicKey[common.BridgeConsensus]))
}

func (engine *Engine) ValidateProducerPosition(blk common.BlockInterface, lastProposerIdx int, committee []incognitokey.CommitteePublicKey, minCommitteeSize int) error {

	//check producer,proposer,agg sig with this version
	producerPosition := blsbft.GetProposerIndexByRound(lastProposerIdx, blk.GetRound(), len(committee))
	if blk.GetVersion() == 1 {
		tempProducer, err := committee[producerPosition].ToBase58()
		if err != nil {
			return fmt.Errorf("Cannot base58 a committee")
		}
		if strings.Compare(tempProducer, blk.GetProducer()) != 0 {
			return fmt.Errorf("Expect Producer Public Key to be equal but get %+v From Index, %+v From Header", tempProducer, blk.GetProducer())
		}
	} else {
		//validate producer
		producer := blk.GetProducer()
		produceTime := blk.GetProduceTime()
		tempProducerID := blockchain.GetProposerByTimeSlot(common.CalculateTimeSlot(produceTime), minCommitteeSize)
		tempProducer := committee[tempProducerID]
		b58Str, _ := tempProducer.ToBase58()
		if strings.Compare(b58Str, producer) != 0 {
			return fmt.Errorf("Expect Producer Public Key to be equal but get %+v From Index, %+v From Header", b58Str, producer)
		}

		//validate proposer
		proposer := blk.GetProposer()
		proposeTime := blk.GetProposeTime()
		tempProducerID = blockchain.GetProposerByTimeSlot(common.CalculateTimeSlot(proposeTime), minCommitteeSize)
		tempProducer = committee[tempProducerID]
		b58Str, _ = tempProducer.ToBase58()
		if strings.Compare(b58Str, proposer) != 0 {
			return fmt.Errorf("Expect Proposer Public Key to be equal but get %+v From Index, %+v From Header", b58Str, proposer)
		}
	}

	return nil
}

func (engine *Engine) ValidateProducerSig(block common.BlockInterface, consensusType string) error {
	if block.GetVersion() == 1 {
		return blsbft.ValidateProducerSig(block)
	} else if block.GetVersion() == 2 {
		return blsbftv2.ValidateProducerSig(block)
	}
	return fmt.Errorf("Wrong block version: %v", block.GetVersion())
}

func (engine *Engine) ValidateBlockCommitteSig(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	if block.GetVersion() == 1 {
		return blsbft.ValidateCommitteeSig(block, committee)
	} else if block.GetVersion() == 2 {
		return blsbftv2.ValidateCommitteeSig(block, committee)
	}
	return fmt.Errorf("Wrong block version: %v", block.GetVersion())
}

func (engine *Engine) GenMiningKeyFromPrivateKey(privateKey string) (string, error) {
	privateSeed, err := LoadUserKeyFromIncPrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	return privateSeed, nil
}

func (engine *Engine) ExtractBridgeValidationData(block common.BlockInterface) ([][]byte, []int, error) {
	if block.GetVersion() == 1 {
		return blsbft.ExtractBridgeValidationData(block)
	} else if block.GetVersion() == 2 {
		return blsbftv2.ExtractBridgeValidationData(block)
	}
	return nil, nil, blsbft.NewConsensusError(blsbft.ConsensusTypeNotExistError, errors.New(block.GetConsensusType()))
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
