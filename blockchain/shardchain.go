package blockchain

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/incdb"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/multiview"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type ShardChain struct {
	shardID   int
	multiView *multiview.MultiView

	BlockGen    *BlockGenerator
	Blockchain  *BlockChain
	hashHistory *lru.Cache
	ChainName   string
	Ready       bool

	insertLock sync.Mutex
}

func NewShardChain(shardID int, multiView *multiview.MultiView, blockGen *BlockGenerator, blockchain *BlockChain, chainName string) *ShardChain {
	return &ShardChain{shardID: shardID, multiView: multiView, BlockGen: blockGen, Blockchain: blockchain, ChainName: chainName}
}

func (chain *ShardChain) GetDatabase() incdb.Database {
	return chain.Blockchain.GetShardChainDatabase(byte(chain.shardID))
}

func (chain *ShardChain) GetFinalView() multiview.View {
	return chain.multiView.GetFinalView()
}

func (chain *ShardChain) GetBestView() multiview.View {
	return chain.multiView.GetBestView()
}

func (chain *ShardChain) GetViewByHash(hash common.Hash) multiview.View {
	return chain.multiView.GetViewByHash(hash)
}

func (chain *ShardChain) GetBestState() *ShardBestState {
	return chain.multiView.GetBestView().(*ShardBestState)
}

func (s *ShardChain) GetEpoch() uint64 {
	return s.GetBestState().Epoch
}

func (s *ShardChain) InsertBatchBlock([]common.BlockInterface) (int, error) {
	panic("implement me")
}

func (s *ShardChain) GetCrossShardState() map[byte]uint64 {

	res := make(map[byte]uint64)
	for index, key := range s.GetBestState().BestCrossShard {
		res[index] = key
	}
	return res
}

func (s *ShardChain) GetAllViewHash() (res []common.Hash) {
	for _, v := range s.multiView.GetAllViewsWithBFS() {
		res = append(res, *v.GetHash())
	}
	return
}

func (s *ShardChain) GetBestViewHeight() uint64 {
	return s.CurrentHeight()
}

func (s *ShardChain) GetFinalViewHeight() uint64 {
	return s.GetFinalView().GetHeight()
}

func (s *ShardChain) GetBestViewHash() string {
	return s.GetBestState().Hash().String()
}

func (s *ShardChain) GetFinalViewHash() string {
	return s.GetBestState().Hash().String()
}
func (chain *ShardChain) GetLastBlockTimeStamp() int64 {
	return chain.GetBestState().BestBlock.Header.Timestamp
}

func (chain *ShardChain) GetMinBlkInterval() time.Duration {
	return chain.GetBestState().BlockInterval
}

func (chain *ShardChain) GetMaxBlkCreateTime() time.Duration {
	return chain.GetBestState().BlockMaxCreateTime
}

func (chain *ShardChain) IsReady() bool {
	return chain.Ready
}

func (chain *ShardChain) SetReady(ready bool) {
	chain.Ready = ready
}

func (chain *ShardChain) CurrentHeight() uint64 {
	return chain.GetBestState().BestBlock.Header.Height
}

func (chain *ShardChain) GetCommittee() []incognitokey.CommitteePublicKey {
	result := []incognitokey.CommitteePublicKey{}
	return append(result, chain.GetBestState().ShardCommittee...)
}

func (chain *ShardChain) GetCommitteeByHeight(h uint64) ([]incognitokey.CommitteePublicKey, error) {
	bcStateRootHash := chain.Blockchain.GetBeaconBestState().ConsensusStateDBRootHash
	bcDB := chain.Blockchain.GetBeaconChainDatabase()
	bcStateDB, err := statedb.NewWithPrefixTrie(bcStateRootHash, statedb.NewDatabaseAccessWarper(bcDB))
	if err != nil {
		return nil, err
	}
	return statedb.GetOneShardCommittee(bcStateDB, byte(chain.shardID)), nil
}

func (chain *ShardChain) GetPendingCommittee() []incognitokey.CommitteePublicKey {
	result := []incognitokey.CommitteePublicKey{}
	return append(result, chain.GetBestState().ShardPendingValidator...)
}

func (chain *ShardChain) GetCommitteeSize() int {
	return len(chain.GetBestState().ShardCommittee)
}

func (chain *ShardChain) GetPubKeyCommitteeIndex(pubkey string) int {
	for index, key := range chain.GetBestState().ShardCommittee {
		if key.GetMiningKeyBase58(chain.GetBestState().ConsensusAlgorithm) == pubkey {
			return index
		}
	}
	return -1
}

func (chain *ShardChain) GetLastProposerIndex() int {
	return chain.GetBestState().ShardProposerIdx
}

func (chain *ShardChain) CreateNewBlock(version int, proposer string, round int, startTime int64) (common.BlockInterface, error) {
	Logger.log.Infof("Begin Start New Block Shard %+v", time.Unix(startTime, 0))
	time1 := time.Now()
	newBlock, err := chain.Blockchain.NewBlockShard(chain.GetBestState(), version, proposer, round, startTime)
	Logger.log.Infof("Finish New Block Shard %+v", time.Since(time1).Seconds())
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	if version == 2 {
		newBlock.Header.Proposer = proposer
		newBlock.Header.ProposeTime = startTime
	}

	Logger.log.Infof("Finish Create New Block")
	return newBlock, nil
}

func (chain *ShardChain) CreateNewBlockFromOldBlock(oldBlock common.BlockInterface, proposer string, startTime int64) (common.BlockInterface, error) {
	b, _ := json.Marshal(oldBlock)
	newBlock := new(ShardBlock)
	json.Unmarshal(b, &newBlock)
	newBlock.Header.Proposer = proposer
	newBlock.Header.ProposeTime = startTime
	return newBlock, nil
}

// func (chain *ShardChain) ValidateAndInsertBlock(block common.BlockInterface) error {
// 	//@Bahamoot review later
// 	chain.lock.Lock()
// 	defer chain.lock.Unlock()
// 	var shardBestState ShardBestState
// 	shardBlock := block.(*ShardBlock)
// 	shardBestState.cloneShardBestStateFrom(chain.BestState)
// 	producerPublicKey := shardBlock.Header.Producer
// 	producerPosition := (shardBestState.ShardProposerIdx + shardBlock.Header.Round) % len(shardBestState.ShardCommittee)
// 	tempProducer := shardBestState.ShardCommittee[producerPosition].GetMiningKeyBase58(shardBestState.ConsensusAlgorithm)
// 	if strings.Compare(tempProducer, producerPublicKey) != 0 {
// 		return NewBlockChainError(BeaconBlockProducerError, fmt.Errorf("Expect Producer Public Key to be equal but get %+v From Index, %+v From Header", tempProducer, producerPublicKey))
// 	}
// 	if err := chain.ValidateBlockSignatures(block, shardBestState.ShardCommittee); err != nil {
// 		return err
// 	}
// 	return chain.Blockchain.InsertShardBlock(shardBlock, false)
// }

func (chain *ShardChain) ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	if err := chain.Blockchain.config.ConsensusEngine.ValidateProducerSig(block, chain.GetConsensusType()); err != nil {
		return err
	}

	if err := chain.Blockchain.config.ConsensusEngine.ValidateBlockCommitteSig(block, committee); err != nil {
		return err
	}
	return nil
}

func (chain *ShardChain) InsertBlk(block common.BlockInterface, shouldValidate bool) error {
	err := chain.Blockchain.InsertShardBlock(block.(*ShardBlock), shouldValidate)
	if err != nil {
		Logger.log.Error(err)
	}
	return err
}

func (chain *ShardChain) CheckExistedBlk(block common.BlockInterface) bool {
	blkHash := block.Hash()
	_, err := rawdbv2.GetBeaconBlockByHash(chain.Blockchain.GetShardChainDatabase(byte(chain.shardID)), *blkHash)
	return err == nil
}

func (chain *ShardChain) InsertAndBroadcastBlock(block common.BlockInterface) error {
	go chain.Blockchain.config.Server.PushBlockToAll(block, false)
	err := chain.Blockchain.InsertShardBlock(block.(*ShardBlock), false)
	if err != nil {
		Logger.log.Error(err)
		return err
	}
	return nil
}

func (chain *ShardChain) GetActiveShardNumber() int {
	return 0
}

func (chain *ShardChain) GetChainName() string {
	return chain.ChainName
}

func (chain *ShardChain) GetConsensusType() string {
	return chain.GetBestState().ConsensusAlgorithm
}

func (chain *ShardChain) GetShardID() int {
	return chain.shardID
}

func (chain *ShardChain) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	var shardBlk ShardBlock
	err := json.Unmarshal(blockString, &shardBlk)
	if err != nil {
		return nil, err
	}
	return &shardBlk, nil
}

func (chain *ShardChain) ValidatePreSignBlock(block common.BlockInterface) error {
	err := chain.Blockchain.VerifyPreSignShardBlock(block.(*ShardBlock), byte(block.(*ShardBlock).GetShardID()))
	if err != nil {
		Logger.log.Error("ValidatePreSignBlock Shard", err)
	}
	return err
}

func (chain *ShardChain) GetAllView() []multiview.View {
	return chain.multiView.GetAllViewsWithBFS()
}
