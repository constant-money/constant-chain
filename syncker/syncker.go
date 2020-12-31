package syncker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/peerv2"

	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wire"
)

const MAX_S2B_BLOCK = 90
const MAX_CROSSX_BLOCK = 10

type SynckerManagerConfig struct {
	Network    Network
	Blockchain *blockchain.BlockChain
	Consensus  peerv2.ConsensusData
}

type SynckerManager struct {
	isEnabled             bool //0 > stop, 1: running
	config                *SynckerManagerConfig
	BeaconSyncProcess     *BeaconSyncProcess
	ShardSyncProcess      map[int]*ShardSyncProcess
	CrossShardSyncProcess map[int]*CrossShardSyncProcess
	beaconPool            *BlkPool
	shardPool             map[int]*BlkPool
	crossShardPool        map[int]*BlkPool
}

func NewSynckerManager() *SynckerManager {
	s := &SynckerManager{
		ShardSyncProcess:      make(map[int]*ShardSyncProcess),
		shardPool:             make(map[int]*BlkPool),
		CrossShardSyncProcess: make(map[int]*CrossShardSyncProcess),
		crossShardPool:        make(map[int]*BlkPool),
	}
	return s
}

func (synckerManager *SynckerManager) Init(config *SynckerManagerConfig) {
	synckerManager.config = config

	//check preload beacon
	preloadAddr := synckerManager.config.Blockchain.GetConfig().ChainParams.PreloadAddress
	if preloadAddr != "" {
		if err := preloadDatabase(-1, int(config.Blockchain.BeaconChain.GetEpoch()), preloadAddr, config.Blockchain.GetBeaconChainDatabase(), config.Blockchain.GetBTCHeaderChain()); err != nil {
			fmt.Println(err)
			Logger.Infof("Preload beacon fail!")
		} else {
			config.Blockchain.RestoreBeaconViews()
		}
	}

	//init beacon sync process
	synckerManager.BeaconSyncProcess = NewBeaconSyncProcess(synckerManager.config.Network, synckerManager.config.Blockchain, synckerManager.config.Blockchain.BeaconChain)
	synckerManager.beaconPool = synckerManager.BeaconSyncProcess.beaconPool

	//init shard sync process
	for _, chain := range synckerManager.config.Blockchain.ShardChain {
		sid := chain.GetShardID()
		synckerManager.ShardSyncProcess[sid] = NewShardSyncProcess(sid, synckerManager.config.Network, synckerManager.config.Blockchain, synckerManager.config.Blockchain.BeaconChain, chain)
		synckerManager.shardPool[sid] = synckerManager.ShardSyncProcess[sid].shardPool
		synckerManager.CrossShardSyncProcess[sid] = synckerManager.ShardSyncProcess[sid].crossShardSyncProcess
		synckerManager.crossShardPool[sid] = synckerManager.CrossShardSyncProcess[sid].crossShardPool

	}

	//watch commitee change
	go synckerManager.manageSyncProcess()
}

func (synckerManager *SynckerManager) Start() {
	synckerManager.isEnabled = true
}

func (synckerManager *SynckerManager) Stop() {
	synckerManager.isEnabled = false
	synckerManager.BeaconSyncProcess.stop()
	for _, chain := range synckerManager.ShardSyncProcess {
		chain.stop()
	}
}

func (s *SynckerManager) InsertCrossShardBlock(blk *blockchain.CrossShardBlock) {
	s.CrossShardSyncProcess[int(blk.ToShardID)].InsertCrossShardBlock(blk)
}

// periodically check user commmittee status, enable shard sync process if needed (beacon always start)
func (synckerManager *SynckerManager) manageSyncProcess() {
	defer time.AfterFunc(time.Second*5, synckerManager.manageSyncProcess)

	//check if enable
	if !synckerManager.isEnabled || synckerManager.config == nil {
		return
	}

	chainValidator := synckerManager.config.Consensus.GetOneValidatorForEachConsensusProcess()

	if beaconChain, ok := chainValidator[-1]; ok {
		synckerManager.BeaconSyncProcess.isCommittee = (beaconChain.State.Role == common.CommitteeRole)
	}

	preloadAddr := synckerManager.config.Blockchain.GetConfig().ChainParams.PreloadAddress
	synckerManager.BeaconSyncProcess.start()

	wg := sync.WaitGroup{}
	wantedShard := synckerManager.config.Blockchain.GetWantedShard(synckerManager.BeaconSyncProcess.isCommittee)
	for chainID, _ := range chainValidator {
		wantedShard[byte(chainID)] = struct{}{}
	}
	for sid, syncProc := range synckerManager.ShardSyncProcess {
		wg.Add(1)
		go func(sid int, syncProc *ShardSyncProcess) {
			defer wg.Done()
			if _, ok := wantedShard[byte(sid)]; ok {
				//check preload shard
				if preloadAddr != "" {
					if syncProc.status != RUNNING_SYNC { //run only when start
						if err := preloadDatabase(sid, int(syncProc.Chain.GetEpoch()), preloadAddr, synckerManager.config.Blockchain.GetShardChainDatabase(byte(sid)), nil); err != nil {
							fmt.Println(err)
							Logger.Infof("Preload shard %v fail!", sid)
						} else {
							synckerManager.config.Blockchain.RestoreShardViews(byte(sid))
						}
					}
				}
				syncProc.start()
			} else {
				syncProc.stop()
			}
			if chain, ok := chainValidator[sid]; ok {
				syncProc.isCommittee = chain.State.Role == common.CommitteeRole || chain.State.Role == common.PendingRole
			}

		}(sid, syncProc)
	}
	wg.Wait()

}

//Process incomming broadcast block
func (synckerManager *SynckerManager) ReceiveBlock(blk interface{}, peerID string) {
	switch blk.(type) {
	case *blockchain.BeaconBlock:
		beaconBlk := blk.(*blockchain.BeaconBlock)
		fmt.Printf("syncker: receive beacon block %d \n", beaconBlk.GetHeight())
		//create fake s2b pool peerstate
		if synckerManager.BeaconSyncProcess != nil {
			synckerManager.beaconPool.AddBlock(beaconBlk)
			synckerManager.BeaconSyncProcess.beaconPeerStateCh <- &wire.MessagePeerState{
				Beacon: wire.ChainState{
					Timestamp: beaconBlk.Header.Timestamp,
					BlockHash: *beaconBlk.Hash(),
					Height:    beaconBlk.GetHeight(),
				},
				SenderID:  peerID,
				Timestamp: time.Now().Unix(),
			}
		}

	case *blockchain.ShardBlock:

		shardBlk := blk.(*blockchain.ShardBlock)
		//fmt.Printf("syncker: receive shard block %d \n", shardBlk.GetHeight())
		if synckerManager.shardPool[shardBlk.GetShardID()] != nil {
			synckerManager.shardPool[shardBlk.GetShardID()].AddBlock(shardBlk)
			if synckerManager.ShardSyncProcess[shardBlk.GetShardID()] != nil {
				synckerManager.ShardSyncProcess[shardBlk.GetShardID()].shardPeerStateCh <- &wire.MessagePeerState{
					Shards: map[byte]wire.ChainState{
						byte(shardBlk.GetShardID()): {
							Timestamp: shardBlk.Header.Timestamp,
							BlockHash: *shardBlk.Hash(),
							Height:    shardBlk.GetHeight(),
						},
					},
					SenderID:  peerID,
					Timestamp: time.Now().Unix(),
				}
			}
		}

	case *blockchain.CrossShardBlock:
		csBlk := blk.(*blockchain.CrossShardBlock)
		if synckerManager.CrossShardSyncProcess[int(csBlk.ToShardID)] != nil {
			fmt.Printf("crossdebug: receive block from %d to %d (%synckerManager)\n", csBlk.Header.ShardID, csBlk.ToShardID, csBlk.Hash().String())
			synckerManager.crossShardPool[int(csBlk.ToShardID)].AddBlock(csBlk)
		}
	}
}

//Process incomming broadcast peerstate
func (synckerManager *SynckerManager) ReceivePeerState(peerState *wire.MessagePeerState) {
	//beacon
	if peerState.Beacon.Height != 0 && synckerManager.BeaconSyncProcess != nil {
		synckerManager.BeaconSyncProcess.beaconPeerStateCh <- peerState
	}
	//shard
	for sid, _ := range peerState.Shards {
		if synckerManager.ShardSyncProcess[int(sid)] != nil {
			// b, _ := json.Marshal(peerState)
			// fmt.Println("[debugshard]: receive peer state", string(b))
			synckerManager.ShardSyncProcess[int(sid)].shardPeerStateCh <- peerState
		}

	}
}

//Get Crossshard Block for creating shardblock block
func (synckerManager *SynckerManager) GetCrossShardBlocksForShardProducer(toShard byte, limit map[byte][]uint64) map[byte][]interface{} {
	//get last confirm crossshard -> process request until retrieve info
	res := make(map[byte][]interface{})

	lastRequestCrossShard := synckerManager.config.Blockchain.ShardChain[int(toShard)].GetCrossShardState()
	bc := synckerManager.config.Blockchain
	beaconDB := bc.GetBeaconChainDatabase()
	for i := 0; i < synckerManager.config.Blockchain.GetActiveShardNumber(); i++ {
		for {
			if i == int(toShard) {
				break
			}

			//if limit has 0 length, we should break now
			if limit != nil && len(res[byte(i)]) >= len(limit[byte(i)]) {
				break
			}

			requestHeight := lastRequestCrossShard[byte(i)]
			nextCrossShardInfo := synckerManager.config.Blockchain.FetchNextCrossShard(i, int(toShard), requestHeight)
			if nextCrossShardInfo == nil {
				break
			}
			if requestHeight == nextCrossShardInfo.NextCrossShardHeight {
				break
			}

			Logger.Info("nextCrossShardInfo.NextCrossShardHeight", i, toShard, requestHeight, nextCrossShardInfo)

			beaconHash, _ := common.Hash{}.NewHashFromStr(nextCrossShardInfo.ConfirmBeaconHash)
			beaconBlockBytes, err := rawdbv2.GetBeaconBlockByHash(beaconDB, *beaconHash)
			if err != nil {
				break
			}

			beaconBlock := new(blockchain.BeaconBlock)
			json.Unmarshal(beaconBlockBytes, beaconBlock)

			for _, shardState := range beaconBlock.Body.ShardState[byte(i)] {
				if shardState.Height == nextCrossShardInfo.NextCrossShardHeight {
					if synckerManager.crossShardPool[int(toShard)].HasHash(shardState.Hash) {
						//validate crossShardBlock before add to result
						blkXShard := synckerManager.crossShardPool[int(toShard)].GetBlock(shardState.Hash)
						beaconConsensusRootHash, err := bc.GetBeaconConsensusRootHash(bc.GetBeaconBestState(), beaconBlock.GetHeight()-1)
						if err != nil {
							Logger.Error("Cannot get beacon consensus root hash from block ", beaconBlock.GetHeight()-1)
							return nil
						}
						beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconConsensusRootHash, statedb.NewDatabaseAccessWarper(bc.GetBeaconChainDatabase()))
						committee := statedb.GetOneShardCommittee(beaconConsensusStateDB, byte(i))
						err = bc.ShardChain[byte(i)].ValidateBlockSignatures(blkXShard.(common.BlockInterface), committee)
						if err != nil {
							Logger.Error("Validate crossshard block fail", blkXShard.GetHeight(), blkXShard.Hash())
							return nil
						}
						//add to result list
						res[byte(i)] = append(res[byte(i)], blkXShard)
						//has block in pool, update request pointer
						lastRequestCrossShard[byte(i)] = nextCrossShardInfo.NextCrossShardHeight
					}
					break
				}
			}

			//cannot append crossshard for a shard (no block in pool, validate error) => break process for this shard
			if requestHeight == lastRequestCrossShard[byte(i)] {
				break
			}

			if len(res[byte(i)]) >= MAX_CROSSX_BLOCK {
				break
			}
		}
	}
	return res
}

//Get Crossshard Block for validating shardblock block
func (synckerManager *SynckerManager) GetCrossShardBlocksForShardValidator(toShard byte, list map[byte][]uint64) (map[byte][]interface{}, error) {
	crossShardPoolLists := synckerManager.GetCrossShardBlocksForShardProducer(toShard, list)

	missingBlocks := compareListsByHeight(crossShardPoolLists, list)
	// synckerManager.config.Server.
	if len(missingBlocks) > 0 {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		synckerManager.StreamMissingCrossShardBlock(ctx, toShard, missingBlocks)
		//Logger.Info("debug finish stream missing crossX block")

		crossShardPoolLists = synckerManager.GetCrossShardBlocksForShardProducer(toShard, list)
		//Logger.Info("get crosshshard block for shard producer", crossShardPoolLists)
		missingBlocks = compareListsByHeight(crossShardPoolLists, list)

		if len(missingBlocks) > 0 {
			return nil, errors.New("Unable to sync required block in time")
		}
	}

	for sid, heights := range list {
		if len(crossShardPoolLists[sid]) != len(heights) {
			return nil, fmt.Errorf("CrossShard list not match sid:%v pool:%v producer:%v", sid, len(crossShardPoolLists[sid]), len(heights))
		}
	}

	return crossShardPoolLists, nil
}

//Stream Missing CrossShard Block
func (synckerManager *SynckerManager) StreamMissingCrossShardBlock(ctx context.Context, toShard byte, missingBlock map[byte][]uint64) {
	for fromShard, missingHeight := range missingBlock {
		//fmt.Println("debug stream missing crossshard block", int(fromShard), int(toShard), missingHeight)
		ch, err := synckerManager.config.Network.RequestCrossShardBlocksViaStream(ctx, "", int(fromShard), int(toShard), missingHeight)
		if err != nil {
			fmt.Println("Syncker: create channel fail")
			return
		}
		//receive
		for {
			select {
			case blk := <-ch:
				if !isNil(blk) {
					Logger.Infof("Receive crosshard block from shard %v ->  %v, hash %v", fromShard, toShard, blk.(common.BlockPoolInterface).Hash().String())
					synckerManager.crossShardPool[int(toShard)].AddBlock(blk.(common.BlockPoolInterface))
				} else {
					//Logger.Info("Block is nil, break stream")
					return
				}
			}
		}
	}
}

////Get Crossshard Block for validating shardblock block
//func (synckerManager *SynckerManager) GetCrossShardBlocksForShardValidatorByHash(toShard byte, list map[byte][]common.Hash) (map[byte][]interface{}, error) {
//	crossShardPoolLists := synckerManager.GetCrossShardBlocksForShardProducer(toShard)
//
//	missingBlocks := compareLists(crossShardPoolLists, list)
//	// synckerManager.config.Server.
//	if len(missingBlocks) > 0 {
//		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
//		synckerManager.StreamMissingCrossShardBlockByHash(ctx, toShard, missingBlocks)
//		fmt.Println("debug finish stream missing s2b block")
//
//		crossShardPoolLists = synckerManager.GetCrossShardBlocksForShardProducer(toShard)
//		missingBlocks = compareLists(crossShardPoolLists, list)
//
//		if len(missingBlocks) > 0 {
//			return nil, errors.New("Unable to sync required block in time")
//		}
//	}
//	return crossShardPoolLists, nil
//}
//
////Stream Missing CrossShard Block
//func (synckerManager *SynckerManager) StreamMissingCrossShardBlockByHash(ctx context.Context, toShard byte, missingBlock map[byte][]common.Hash) {
//	fmt.Println("debug stream missing crossshard block", missingBlock)
//	wg := sync.WaitGroup{}
//	for i, v := range missingBlock {
//		wg.Add(1)
//		go func(sid byte, list []common.Hash) {
//			defer wg.Done()
//			hashes := [][]byte{}
//			for _, h := range list {
//				hashes = append(hashes, h.Bytes())
//			}
//			ch, err := synckerManager.config.Node.RequestCrossShardBlocksByHashViaStream(ctx, "", int(sid), int(toShard), hashes)
//			if err != nil {
//				fmt.Println("Syncker: create channel fail")
//				return
//			}
//			//receive
//			for {
//				select {
//				case blk := <-ch:
//					if !isNil(blk) {
//						synckerManager.crossShardPool[int(toShard)].AddBlock(blk.(common.BlockPoolInterface))
//					} else {
//						return
//					}
//				}
//			}
//		}(i, v)
//	}
//	wg.Wait()
//}

//Sync missing beacon block  from a hash to our final view (skip if we already have)
func (synckerManager *SynckerManager) SyncMissingBeaconBlock(ctx context.Context, peerID string, fromHash common.Hash) {
	requestHash := fromHash
	for {
		ch, err := synckerManager.config.Network.RequestBeaconBlocksByHashViaStream(ctx, peerID, [][]byte{requestHash.Bytes()})
		if err != nil {
			fmt.Println("[Monitor] Syncker: create channel fail")
			return
		}
		blk := <-ch
		if !isNil(blk) {
			if blk.(*blockchain.BeaconBlock).GetHeight() <= synckerManager.config.Blockchain.BeaconChain.GetFinalViewHeight() {
				return
			}
			synckerManager.beaconPool.AddBlock(blk.(common.BlockPoolInterface))
			prevHash := blk.(*blockchain.BeaconBlock).GetPrevHash()
			if v := synckerManager.config.Blockchain.BeaconChain.GetViewByHash(prevHash); v == nil {
				requestHash = prevHash
				continue
			}
		}
		return
	}
}

//Sync back missing shard block from a hash to our final views (skip if we already have)
func (synckerManager *SynckerManager) SyncMissingShardBlock(ctx context.Context, peerID string, sid byte, fromHash common.Hash) {
	requestHash := fromHash
	for {
		ch, err := synckerManager.config.Network.RequestShardBlocksByHashViaStream(ctx, peerID, int(sid), [][]byte{requestHash.Bytes()})
		if err != nil {
			fmt.Println("Syncker: create channel fail")
			return
		}
		blk := <-ch
		if !isNil(blk) {
			if blk.(*blockchain.ShardBlock).GetHeight() <= synckerManager.config.Blockchain.ShardChain[sid].GetFinalViewHeight() {
				return
			}
			synckerManager.shardPool[int(sid)].AddBlock(blk.(common.BlockPoolInterface))
			prevHash := blk.(*blockchain.ShardBlock).GetPrevHash()
			if v := synckerManager.config.Blockchain.ShardChain[sid].GetViewByHash(prevHash); v == nil {
				requestHash = prevHash
				continue
			}
		}
		return

	}
}

//Get Status Function
type syncInfo struct {
	IsSync     bool
	IsLatest   bool
	PoolLength int
}

type SynckerStatusInfo struct {
	Beacon     syncInfo
	Shard      map[int]*syncInfo
	Crossshard map[int]*syncInfo
}

func (synckerManager *SynckerManager) GetSyncStatus(includePool bool) SynckerStatusInfo {
	info := SynckerStatusInfo{}
	info.Beacon = syncInfo{
		IsSync:   synckerManager.BeaconSyncProcess.status == RUNNING_SYNC,
		IsLatest: synckerManager.BeaconSyncProcess.isCatchUp,
	}

	info.Shard = make(map[int]*syncInfo)
	for k, v := range synckerManager.ShardSyncProcess {
		info.Shard[k] = &syncInfo{
			IsSync:   v.status == RUNNING_SYNC,
			IsLatest: v.isCatchUp,
		}
	}

	info.Crossshard = make(map[int]*syncInfo)
	for k, v := range synckerManager.CrossShardSyncProcess {
		info.Crossshard[k] = &syncInfo{
			IsSync:   v.status == RUNNING_SYNC,
			IsLatest: false,
		}
	}

	if includePool {
		info.Beacon.PoolLength = synckerManager.beaconPool.GetPoolSize()
		for k, _ := range synckerManager.ShardSyncProcess {
			info.Shard[k].PoolLength = synckerManager.shardPool[k].GetPoolSize()
		}
		for k, _ := range synckerManager.CrossShardSyncProcess {
			info.Crossshard[k].PoolLength = synckerManager.crossShardPool[k].GetPoolSize()
		}
	}
	return info
}

func (synckerManager *SynckerManager) IsChainReady(chainID int) bool {
	if chainID == -1 {
		return synckerManager.BeaconSyncProcess.isCatchUp
	} else if chainID >= 0 {
		return synckerManager.ShardSyncProcess[chainID].isCatchUp
	}
	return false
}

type TmpBlock struct {
	Height  uint64
	BlkHash *common.Hash
	PreHash common.Hash
	ShardID int
	Round   int
}

func (blk *TmpBlock) GetHeight() uint64 {
	return blk.Height
}

func (blk *TmpBlock) Hash() *common.Hash {
	return blk.BlkHash
}

func (blk *TmpBlock) GetPrevHash() common.Hash {
	return blk.PreHash
}

func (blk *TmpBlock) GetShardID() int {
	return blk.ShardID
}
func (blk *TmpBlock) GetRound() int {
	return 1
}

func (synckerManager *SynckerManager) GetPoolInfo(poolType byte, sID int) []common.BlockPoolInterface {
	switch poolType {
	case BeaconPoolType:
		if synckerManager.BeaconSyncProcess != nil {
			if synckerManager.BeaconSyncProcess.beaconPool != nil {
				return synckerManager.BeaconSyncProcess.beaconPool.GetPoolInfo()
			}
		}
	case ShardPoolType:
		if syncProcess, ok := synckerManager.ShardSyncProcess[sID]; ok {
			if syncProcess.shardPool != nil {
				return syncProcess.shardPool.GetPoolInfo()
			}
		}
	case CrossShardPoolType:
		if syncProcess, ok := synckerManager.ShardSyncProcess[sID]; ok {
			if syncProcess.shardPool != nil {
				res := []common.BlockPoolInterface{}
				for fromSID, blksPool := range synckerManager.crossShardPool {
					for _, blk := range blksPool.GetBlockList() {
						res = append(res, &TmpBlock{
							Height:  blk.GetHeight(),
							BlkHash: blk.Hash(),
							PreHash: common.Hash{},
							ShardID: fromSID,
						})
					}
				}
				return res
			}
		}
	}
	return []common.BlockPoolInterface{}
}

func (synckerManager *SynckerManager) GetPoolLatestHeight(poolType byte, bestHash string, sID int) uint64 {
	switch poolType {
	case BeaconPoolType:
		if synckerManager.BeaconSyncProcess != nil {
			if synckerManager.BeaconSyncProcess.beaconPool != nil {
				return synckerManager.BeaconSyncProcess.beaconPool.GetLatestHeight(bestHash)
			}
		}
	case ShardPoolType:
		if syncProcess, ok := synckerManager.ShardSyncProcess[sID]; ok {
			if syncProcess.shardPool != nil {
				return syncProcess.shardPool.GetLatestHeight(bestHash)
			}
		}
	case CrossShardPoolType:
		//TODO
		return 0
	}
	return 0
}

func (synckerManager *SynckerManager) GetAllViewByHash(poolType byte, bestHash string, sID int) []common.BlockPoolInterface {
	switch poolType {
	case BeaconPoolType:
		if synckerManager.BeaconSyncProcess != nil {
			if synckerManager.BeaconSyncProcess.beaconPool != nil {
				return synckerManager.BeaconSyncProcess.beaconPool.GetAllViewByHash(bestHash)
			}
		}
	case ShardPoolType:
		if syncProcess, ok := synckerManager.ShardSyncProcess[sID]; ok {
			if syncProcess.shardPool != nil {
				return syncProcess.shardPool.GetAllViewByHash(bestHash)
			}
		}
	default:
		//TODO
		return []common.BlockPoolInterface{}
	}
	return []common.BlockPoolInterface{}
}
