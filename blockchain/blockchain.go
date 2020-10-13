package blockchain

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/incognitochain/incognito-chain/blockchain/btc"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/multiview"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/pubsub"
	bnbrelaying "github.com/incognitochain/incognito-chain/relaying/bnb"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
	"github.com/incognitochain/incognito-chain/transaction"

	"github.com/pkg/errors"
)

type BlockChain struct {
	BeaconChain *BeaconChain
	ShardChain  []*ShardChain
	config      Config
	cQuitSync   chan struct{}

	IsTest bool
}

// Config is a descriptor which specifies the blockchain instance configuration.
type Config struct {
	BTCChain          *btcrelaying.BlockChain
	BNBChainState     *bnbrelaying.BNBChainState
	DataBase          map[int]incdb.Database
	MemCache          *memcache.MemoryCache
	Interrupt         <-chan struct{}
	ChainParams       *Params
	GenesisParams     *GenesisParams
	RelayShards       []byte
	NodeMode          string
	BlockGen          *BlockGenerator
	TxPool            TxPool
	TempTxPool        TxPool
	CRemovedTxs       chan metadata.Transaction
	FeeEstimator      map[byte]FeeEstimator
	IsBlockGenStarted bool
	PubSubManager     *pubsub.PubSubManager
	RandomClient      btc.RandomClient
	Syncker           Syncker
	Server            Server
	ConsensusEngine   ConsensusEngine
	Highway           Highway
}

func NewBlockChain(config *Config, isTest bool) *BlockChain {
	bc := &BlockChain{}
	bc.config = *config
	bc.config.IsBlockGenStarted = false
	bc.IsTest = isTest
	bc.cQuitSync = make(chan struct{})
	bc.GetBeaconBestState().Params = make(map[string]string)
	bc.GetBeaconBestState().ShardCommittee = make(map[byte][]incognitokey.CommitteePublicKey)
	bc.GetBeaconBestState().ShardPendingValidator = make(map[byte][]incognitokey.CommitteePublicKey)
	return bc
}

/*
Init - init a blockchain view from config
*/
func (blockchain *BlockChain) Init(config *Config) error {
	// Enforce required config fields.
	if config.DataBase == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Database is not config"))
	}
	if config.ChainParams == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Chain parameters is not config"))
	}
	blockchain.config = *config
	blockchain.config.IsBlockGenStarted = false
	blockchain.IsTest = false
	// Initialize the chain state from the passed database.  When the db
	// does not yet contain any chain state, both it and the chain state
	// will be initialized to contain only the genesis block.
	if err := blockchain.initChainState(); err != nil {
		return err
	}
	blockchain.cQuitSync = make(chan struct{})
	return nil
}

// initChainState attempts to load and initialize the chain state from the
// database.  When the db does not yet contain any chain state, both it and the
// chain state are initialized to the genesis block.
func (blockchain *BlockChain) initChainState() error {
	// Determine the state of the chain database. We may need to initialize
	// everything from scratch or upgrade certain buckets.
	blockchain.BeaconChain = NewBeaconChain(multiview.NewMultiView(), blockchain.config.BlockGen, blockchain, common.BeaconChainKey)
	if err := blockchain.RestoreBeaconViews(); err != nil {
		Logger.log.Error("debug restore beacon fail, init", err)
		err := blockchain.initBeaconState()
		if err != nil {
			Logger.log.Error("debug beacon state init error")
			return err
		}
	}
	Logger.log.Infof("Init Beacon View height %+v", blockchain.BeaconChain.GetFinalViewHeight())

	blockchain.ShardChain = make([]*ShardChain, blockchain.GetBeaconBestState().ActiveShards)
	for shard := 1; shard <= blockchain.GetBeaconBestState().ActiveShards; shard++ {
		shardID := byte(shard - 1)
		blockchain.ShardChain[shardID] = NewShardChain(shard-1, multiview.NewMultiView(), blockchain.config.BlockGen, blockchain, common.GetShardChainKey(shardID))
		if err := blockchain.RestoreShardViews(shardID); err != nil {
			Logger.log.Error("debug restore shard fail, init")
			err := blockchain.initShardState(shardID)
			if err != nil {
				Logger.log.Error("debug shard state init error")
				return err
			}
		}
		Logger.log.Infof("Init Shard View shardID %+v, height %+v", shardID, blockchain.ShardChain[shardID].GetFinalViewHeight())
	}

	return nil
}

/*
// createChainState initializes both the database and the chain state to the
// genesis block.  This includes creating the necessary buckets and inserting
// the genesis block, so it must only be called on an uninitialized database.
*/
func (blockchain *BlockChain) initShardState(shardID byte) error {
	initShardState := NewBestStateShardWithConfig(shardID, blockchain.config.ChainParams)
	// Create a new block from genesis block and set it as best block of chain
	initShardBlock := ShardBlock{}
	initShardBlock = *blockchain.config.ChainParams.GenesisShardBlock
	initShardBlock.Header.ShardID = shardID
	initShardBlockHeight := initShardBlock.Header.Height
	_, newShardCandidate := GetStakingCandidate(*blockchain.config.ChainParams.GenesisBeaconBlock)
	newShardCandidateStructs := []incognitokey.CommitteePublicKey{}
	for _, candidate := range newShardCandidate {
		key := incognitokey.CommitteePublicKey{}
		err := key.FromBase58(candidate)
		if err != nil {
			return err
		}
		newShardCandidateStructs = append(newShardCandidateStructs, key)
	}
	initShardState.ShardCommittee = append(initShardState.ShardCommittee, newShardCandidateStructs[int(shardID)*blockchain.config.ChainParams.MinShardCommitteeSize:(int(shardID)*blockchain.config.ChainParams.MinShardCommitteeSize)+blockchain.config.ChainParams.MinShardCommitteeSize]...)
	beaconBlocks, err := blockchain.GetBeaconBlockByHeight(initShardBlockHeight)
	genesisBeaconBlock := beaconBlocks[0]
	if err != nil {
		return NewBlockChainError(FetchBeaconBlockError, err)
	}
	err = initShardState.initShardBestState(blockchain, blockchain.GetShardChainDatabase(shardID), &initShardBlock, genesisBeaconBlock)
	if err != nil {
		return err
	}
	committeeChange := newCommitteeChange()
	committeeChange.shardCommitteeAdded[shardID] = initShardState.GetShardCommittee()
	err = blockchain.processStoreShardBlock(initShardState, &initShardBlock, committeeChange, genesisBeaconBlock)
	if err != nil {
		return err
	}
	return nil
}

func (blockchain *BlockChain) initBeaconState() error {
	initBeaconBestState := NewBeaconBestStateWithConfig(blockchain.config.ChainParams)
	initBlock := blockchain.config.ChainParams.GenesisBeaconBlock
	err := initBeaconBestState.initBeaconBestState(initBlock, blockchain, blockchain.GetBeaconChainDatabase())
	if err != nil {
		return err
	}
	initBlockHash := initBeaconBestState.BestBlock.Header.Hash()
	initBlockHeight := initBeaconBestState.BestBlock.Header.Height
	// Insert new block into beacon chain
	if err := statedb.StoreAllShardCommittee(initBeaconBestState.consensusStateDB, initBeaconBestState.ShardCommittee, initBeaconBestState.RewardReceiver, initBeaconBestState.AutoStaking); err != nil {
		return err
	}
	if err := statedb.StoreBeaconCommittee(initBeaconBestState.consensusStateDB, initBeaconBestState.BeaconCommittee, initBeaconBestState.RewardReceiver, initBeaconBestState.AutoStaking); err != nil {
		return err
	}
	consensusRootHash, err := initBeaconBestState.consensusStateDB.Commit(true)
	if err != nil {
		return err
	}
	err = initBeaconBestState.consensusStateDB.Database().TrieDB().Commit(consensusRootHash, false)
	if err != nil {
		return err
	}
	initBeaconBestState.consensusStateDB.ClearObjects()
	if err := rawdbv2.StoreBeaconBlock(blockchain.GetBeaconChainDatabase(), initBlockHeight, initBlockHash, &initBeaconBestState.BestBlock); err != nil {
		Logger.log.Error("Error store beacon block", initBeaconBestState.BestBlockHash, "in beacon chain")
		return err
	}
	if err := rawdbv2.StoreBeaconBlockIndex(blockchain.GetBeaconChainDatabase(), initBlockHeight, initBlockHash); err != nil {
		return err
	}
	// State Root Hash
	if err := rawdbv2.StoreBeaconConsensusStateRootHash(blockchain.GetBeaconChainDatabase(), initBlockHeight, consensusRootHash); err != nil {
		return err
	}
	if err := rawdbv2.StoreBeaconRewardStateRootHash(blockchain.GetBeaconChainDatabase(), initBlockHeight, common.EmptyRoot); err != nil {
		return err
	}
	if err := rawdbv2.StoreBeaconFeatureStateRootHash(blockchain.GetBeaconChainDatabase(), initBlockHeight, common.EmptyRoot); err != nil {
		return err
	}
	if err := rawdbv2.StoreBeaconSlashStateRootHash(blockchain.GetBeaconChainDatabase(), initBlockHeight, common.EmptyRoot); err != nil {
		return err
	}
	// Insert new block into beacon chain
	blockchain.BeaconChain.multiView.AddView(initBeaconBestState)
	if err := blockchain.BackupBeaconViews(blockchain.GetBeaconChainDatabase()); err != nil {
		Logger.log.Error("Error Store best state for block", blockchain.GetBeaconBestState().BestBlockHash, "in beacon chain")
		return NewBlockChainError(UnExpectedError, err)
	}

	return nil
}

func (blockchain *BlockChain) GetClonedBeaconBestState() (*BeaconBestState, error) {
	result := NewBeaconBestState()
	err := result.cloneBeaconBestStateFrom(blockchain.GetBeaconBestState())
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetReadOnlyShard - return a copy of Shard of BestState
func (blockchain *BlockChain) GetClonedAllShardBestState() map[byte]*ShardBestState {
	result := make(map[byte]*ShardBestState)
	for _, v := range blockchain.ShardChain {
		result[byte(v.GetShardID())] = v.GetBestState()
	}
	return result
}

// GetReadOnlyShard - return a copy of Shard of BestState
func (blockchain *BlockChain) GetClonedAShardBestState(shardID byte) (*ShardBestState, error) {
	return blockchain.ShardChain[int(shardID)].GetBestState(), nil
}

func (blockchain *BlockChain) GetCurrentBeaconBlockHeight(shardID byte) uint64 {
	return blockchain.GetBeaconBestState().BestBlock.Header.Height
}

func (blockchain BlockChain) RandomCommitmentsProcess(usableInputCoins []coin.PlainCoin, randNum int, shardID byte, tokenID *common.Hash) (commitmentIndexs []uint64, myCommitmentIndexs []uint64, commitments [][]byte) {
	param := transaction.NewRandomCommitmentsProcessParam(usableInputCoins, randNum, blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB(), shardID, tokenID)
	return transaction.RandomCommitmentsProcess(param)
}

func (blockchain BlockChain) RandomCommitmentsAndPublicKeysProcess(numOutputs int, shardID byte, tokenID *common.Hash) ([]uint64, [][]byte, [][]byte, [][]byte, error) {
	db := blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()
	lenOTA, err := statedb.GetOTACoinLength(db, *tokenID, shardID)
	if err != nil || lenOTA == nil {
		return nil, nil, nil, nil, err
	}

	indices := make([]uint64, 0)
	publicKeys := make([][]byte, 0)
	commitments := make([][]byte, 0)
	assetTags := make([][]byte, 0)
	// these coins either all have asset tags or none does
	var hasAssetTags bool = true
	for i:=0;i<numOutputs;i++{
		idx, _ := common.RandBigIntMaxRange(lenOTA)
		coinBytes, err := statedb.GetOTACoinByIndex(db, *tokenID, idx.Uint64(), shardID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		coinDB := new(coin.CoinV2)
		if err := coinDB.SetBytes(coinBytes); err != nil {
			return nil, nil, nil, nil , err
		}
		publicKey := coinDB.GetPublicKey()
		commitment := coinDB.GetCommitment()

		indices = append(indices, idx.Uint64())
		publicKeys = append(publicKeys, publicKey.ToBytesS())
		commitments = append(commitments, commitment.ToBytesS())

		if hasAssetTags{
			assetTag := coinDB.GetAssetTag()
			if assetTag==nil{
				assetTags = append(assetTags, assetTag.ToBytesS())
			}else{
				hasAssetTags = false
			}
		}
	}

	return indices, publicKeys, commitments, assetTags, nil
}

func (blockchain *BlockChain) GetActiveShardNumber() int {
	return blockchain.GetBeaconBestState().ActiveShards
}

func (blockchain *BlockChain) GetShardIDs() []int {
	shardIDs := []int{}
	for i := 0; i < blockchain.GetActiveShardNumber(); i++ {
		shardIDs = append(shardIDs, i)
	}
	return shardIDs
}

// -------------- Start of Blockchain retriever's implementation --------------
func (blockchain *BlockChain) SetIsBlockGenStarted(value bool) {
	blockchain.config.IsBlockGenStarted = value
}

func (blockchain *BlockChain) AddTxPool(txpool TxPool) {
	blockchain.config.TxPool = txpool
}

func (blockchain *BlockChain) AddTempTxPool(temptxpool TxPool) {
	blockchain.config.TempTxPool = temptxpool
}

func (blockchain *BlockChain) SetFeeEstimator(feeEstimator FeeEstimator, shardID byte) {
	if len(blockchain.config.FeeEstimator) == 0 {
		blockchain.config.FeeEstimator = make(map[byte]FeeEstimator)
	}
	blockchain.config.FeeEstimator[shardID] = feeEstimator
}

func (blockchain *BlockChain) InitChannelBlockchain(cRemovedTxs chan metadata.Transaction) {
	blockchain.config.CRemovedTxs = cRemovedTxs
}

// -------------- End of Blockchain retriever's implementation --------------

// -------------- Start of Blockchain BackUp And Restore --------------
func CalculateNumberOfByteToRead(amountBytes int) []byte {
	var result = make([]byte, 8)
	binary.LittleEndian.PutUint32(result, uint32(amountBytes))
	return result
}

func GetNumberOfByteToRead(value []byte) (int, error) {
	var result uint32
	err := binary.Read(bytes.NewBuffer(value), binary.LittleEndian, &result)
	if err != nil {
		return -1, err
	}
	return int(result), nil
}

func (blockchain *BlockChain) BackupShardChain(writer io.Writer, shardID byte) error {
	bestStateBytes, err := rawdbv2.GetShardBestState(blockchain.GetShardChainDatabase(shardID), shardID)
	if err != nil {
		return err
	}
	shardBestState := &ShardBestState{}
	err = json.Unmarshal(bestStateBytes, shardBestState)
	bestShardHeight := shardBestState.ShardHeight
	var i uint64
	for i = 1; i < bestShardHeight; i++ {
		shardBlocks, err := blockchain.GetShardBlockByHeight(i, shardID)
		if err != nil {
			return err
		}
		var shardBlock *ShardBlock
		for _, v := range shardBlocks {
			shardBlock = v
		}
		data, err := json.Marshal(shardBlocks)
		if err != nil {
			return err
		}
		_, err = writer.Write(CalculateNumberOfByteToRead(len(data)))
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			return err
		}
		if i%100 == 0 {
			Logger.log.Infof("Backup Shard %+v Block %+v", shardBlock.Header.ShardID, i)
		}
		if i == bestShardHeight-1 {
			Logger.log.Infof("Finish Backup Shard %+v with Block %+v", shardBlock.Header.ShardID, i)
		}
	}
	return nil
}

func (blockchain *BlockChain) BackupBeaconChain(writer io.Writer) error {
	bestStateBytes, err := rawdbv2.GetBeaconBestState(blockchain.GetBeaconChainDatabase())
	if err != nil {
		return err
	}
	beaconBestState := &BeaconBestState{}
	err = json.Unmarshal(bestStateBytes, beaconBestState)
	bestBeaconHeight := beaconBestState.BeaconHeight
	var i uint64
	for i = 1; i < bestBeaconHeight; i++ {
		beaconBlocks, err := blockchain.GetBeaconBlockByHeight(i)
		if err != nil {
			return err
		}
		beaconBlock := beaconBlocks[0]
		data, err := json.Marshal(beaconBlock)
		if err != nil {
			return err
		}
		numOfByteToRead := CalculateNumberOfByteToRead(len(data))
		_, err = writer.Write(numOfByteToRead)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			return err
		}
		if i%100 == 0 {
			Logger.log.Infof("Backup Beacon Block %+v", i)
		}
		if i == bestBeaconHeight-1 {
			Logger.log.Infof("Finish Backup Beacon with Block %+v", i)
		}
	}
	return nil
}

//TODO:
// current implement: backup all view data
// Optimize: backup view -> backup view hash instead of view
// restore: get view from hash and create new view, then insert into multiview
/*
Backup all BeaconView into Database
*/
func (blockchain *BlockChain) BackupBeaconViews(db incdb.KeyValueWriter) error {
	allViews := []*BeaconBestState{}
	for _, v := range blockchain.BeaconChain.multiView.GetAllViewsWithBFS() {
		allViews = append(allViews, v.(*BeaconBestState))
	}
	b, _ := json.Marshal(allViews)
	return rawdbv2.StoreBeaconBestState(db, b)
}

/*
Restart all BeaconView from Database
*/
func (blockchain *BlockChain) RestoreBeaconViews() error {
	allViews := []*BeaconBestState{}
	b, err := rawdbv2.GetBeaconBestState(blockchain.GetBeaconChainDatabase())
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &allViews)
	if err != nil {
		return err
	}
	for _, v := range allViews {
		if !blockchain.BeaconChain.multiView.AddView(v) {
			panic("Restart beacon views fail")
		}
		err := v.InitStateRootHash(blockchain)
		if err != nil {
			panic(err)
		}
		currentPDEState, err := InitCurrentPDEStateFromDB(v.featureStateDB, v.BeaconHeight)
		if err != nil {
			Logger.log.Error(err)
			return nil
		}
		v.currentPDEState = currentPDEState
	}
	return nil
}

/*
Backup shard views
*/
func (blockchain *BlockChain) BackupShardViews(db incdb.KeyValueWriter, shardID byte) error {
	allViews := []*ShardBestState{}
	for _, v := range blockchain.ShardChain[shardID].multiView.GetAllViewsWithBFS() {
		allViews = append(allViews, v.(*ShardBestState))
	}
	fmt.Println("debug BackupShardViews", len(allViews))
	return rawdbv2.StoreShardBestState(db, shardID, allViews)
}

/*
Restart all BeaconView from Database
*/
func (blockchain *BlockChain) RestoreShardViews(shardID byte) error {
	allViews := []*ShardBestState{}
	b, err := rawdbv2.GetShardBestState(blockchain.GetShardChainDatabase(shardID), shardID)
	if err != nil {
		fmt.Println("debug Cannot see shard best state")
		return err
	}
	err = json.Unmarshal(b, &allViews)
	if err != nil {
		fmt.Println("debug Cannot unmarshall shard best state", string(b))
		return err
	}
	fmt.Println("debug RestoreShardViews", len(allViews))
	for _, v := range allViews {
		if !blockchain.ShardChain[shardID].multiView.AddView(v) {
			panic("Restart shard views fail")
		}
		err := v.InitStateRootHash(blockchain.GetShardChainDatabase(shardID), blockchain)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

// -------------- End of Blockchain BackUp And Restore --------------

func (blockchain *BlockChain) GetNodeMode() string {
	return blockchain.config.NodeMode
}

func (blockchain *BlockChain) GetWantedShard() map[byte]struct{} {
	res := map[byte]struct{}{}
	for _, sID := range blockchain.config.RelayShards {
		res[sID] = struct{}{}
	}
	return res
}

// GetConfig returns blockchain's config
func (blockchain *BlockChain) GetConfig() *Config {
	return &blockchain.config
}

// GetPortalParams returns portal params in beaconheight
func (blockchain *BlockChain) GetPortalParams(beaconHeight uint64) PortalParams {
	portalParamMap := blockchain.GetConfig().ChainParams.PortalParams
	// only has one value - default value
	if len(portalParamMap) == 1 {
		return portalParamMap[0]
	}

	bchs := []uint64{}
	for bch := range portalParamMap {
		bchs = append(bchs, bch)
	}
	sort.Slice(bchs, func(i, j int) bool {
		return bchs[i] < bchs[j]
	})

	bchKey := bchs[len(bchs)-1]
	for i := len(bchs) - 1; i >= 0; i-- {
		if beaconHeight < bchs[i] {
			continue
		}
		bchKey = bchs[i]
		break
	}

	return portalParamMap[bchKey]
}

func (blockchain *BlockChain) GetBeaconChainDatabase() incdb.Database {
	return blockchain.config.DataBase[common.BeaconChainDataBaseID]
}

func (blockchain *BlockChain) GetShardChainDatabase(shardID byte) incdb.Database {
	return blockchain.config.DataBase[int(shardID)]
}
