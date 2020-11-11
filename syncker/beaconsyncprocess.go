package syncker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"os"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/wire"
)

type BeaconPeerState struct {
	Timestamp      int64
	BestViewHash   string
	BestViewHeight uint64
	processed      bool
}

type BeaconSyncProcess struct {
	status              string //stop, running
	isCommittee         bool
	isCatchUp           bool
	beaconPeerStates    map[string]BeaconPeerState //sender -> state
	beaconPeerStateCh   chan *wire.MessagePeerState
	server              Server
	chain               Chain
	beaconPool          *BlkPool
	actionCh            chan func()
	lastCrossShardState map[byte]map[byte]uint64
}

func NewBeaconSyncProcess(server Server, chain BeaconChainInterface) *BeaconSyncProcess {

	var isOutdatedBlock = func(blk interface{}) bool {
		if blk.(*types.BeaconBlock).GetHeight() < chain.GetFinalViewHeight() {
			return true
		}
		return false
	}

	s := &BeaconSyncProcess{
		status:              STOP_SYNC,
		server:              server,
		chain:               chain,
		beaconPool:          NewBlkPool("BeaconPool", isOutdatedBlock),
		beaconPeerStates:    make(map[string]BeaconPeerState),
		beaconPeerStateCh:   make(chan *wire.MessagePeerState),
		actionCh:            make(chan func()),
		lastCrossShardState: make(map[byte]map[byte]uint64),
	}
	go s.syncBeacon()
	go s.insertBeaconBlockFromPool()
	go s.updateConfirmCrossShard()

	go func() {
		ticker := time.NewTicker(time.Millisecond * 500)
		for {
			select {
			case f := <-s.actionCh:
				f()
			case beaconPeerState := <-s.beaconPeerStateCh:
				Logger.Debugf("Got new peerstate, last height %v", beaconPeerState.Beacon.Height)
				s.beaconPeerStates[beaconPeerState.SenderID] = BeaconPeerState{
					Timestamp:      beaconPeerState.Timestamp,
					BestViewHash:   beaconPeerState.Beacon.BlockHash.String(),
					BestViewHeight: beaconPeerState.Beacon.Height,
				}
				s.chain.SetReady(true)
			case <-ticker.C:
				for sender, ps := range s.beaconPeerStates {
					if ps.Timestamp < time.Now().Unix()-10 {
						delete(s.beaconPeerStates, sender)
					}
				}
			}
			if s.status != RUNNING_SYNC {
				time.Sleep(time.Second)
				continue
			}
		}
	}()

	return s
}

func (s *BeaconSyncProcess) start() {
	if s.status == RUNNING_SYNC {
		return
	}
	s.status = RUNNING_SYNC

}

func (s *BeaconSyncProcess) stop() {
	s.status = STOP_SYNC
}

//helper function to access map in atomic way
func (s *BeaconSyncProcess) getBeaconPeerStates() map[string]BeaconPeerState {
	res := make(chan map[string]BeaconPeerState)
	s.actionCh <- func() {
		ps := make(map[string]BeaconPeerState)
		for k, v := range s.beaconPeerStates {
			ps[k] = v
		}
		res <- ps
	}
	return <-res
}

type NextCrossShardInfo struct {
	NextCrossShardHeight uint64
	NextCrossShardHash   string
	ConfirmBeaconHeight  uint64
	ConfirmBeaconHash    string
}

type LastCrossShardBeaconProcess struct {
	BeaconHeight        uint64
	LastCrossShardState map[byte]map[byte]uint64
}

//watching confirm beacon block and update cross shard info (which beacon confirm crossshard block N of shard X)
func (s *BeaconSyncProcess) updateConfirmCrossShard() {
	state := rawdbv2.GetLastBeaconStateConfirmCrossShard(s.server.GetBeaconChainDatabase())
	lastBeaconStateConfirmCrossX := new(LastCrossShardBeaconProcess)
	_ = json.Unmarshal(state, &lastBeaconStateConfirmCrossX)
	lastBeaconHeightConfirmCrossX := uint64(1)
	if lastBeaconStateConfirmCrossX.BeaconHeight != 0 {
		s.lastCrossShardState = lastBeaconStateConfirmCrossX.LastCrossShardState
		lastBeaconHeightConfirmCrossX = lastBeaconStateConfirmCrossX.BeaconHeight
	}
	fmt.Println("lastBeaconHeightConfirmCrossX", lastBeaconHeightConfirmCrossX)
	for {
		if lastBeaconHeightConfirmCrossX > s.chain.GetFinalViewHeight() {
			//fmt.Println("DEBUG:larger than final view", s.chain.GetFinalViewHeight())
			time.Sleep(time.Second * 5)
			continue
		}
		beaconBlock, err := s.server.FetchConfirmBeaconBlockByHeight(lastBeaconHeightConfirmCrossX)
		if err != nil || beaconBlock == nil {
			//fmt.Println("DEBUG: cannot find beacon block", lastBeaconHeightConfirmCrossX)
			time.Sleep(time.Second * 5)
			continue
		}
		err = processBeaconForConfirmmingCrossShard(s.server.GetBeaconChainDatabase(), beaconBlock, s.lastCrossShardState)
		if err == nil {
			lastBeaconHeightConfirmCrossX++
			if lastBeaconHeightConfirmCrossX%1000 == 0 {
				fmt.Println("store lastBeaconHeightConfirmCrossX", lastBeaconHeightConfirmCrossX)
				rawdbv2.StoreLastBeaconStateConfirmCrossShard(s.server.GetBeaconChainDatabase(), LastCrossShardBeaconProcess{lastBeaconHeightConfirmCrossX, s.lastCrossShardState})
			}
		} else {
			fmt.Println(err)
			time.Sleep(time.Second * 5)
		}

	}
}

func processBeaconForConfirmmingCrossShard(database incdb.Database, beaconBlock *types.BeaconBlock, lastCrossShardState map[byte]map[byte]uint64) error {
	if beaconBlock != nil && beaconBlock.Body.ShardState != nil {
		for fromShard, shardBlocks := range beaconBlock.Body.ShardState {
			for _, shardBlock := range shardBlocks {
				for _, toShard := range shardBlock.CrossShard {

					if fromShard == toShard {
						continue
					}
					if lastCrossShardState[fromShard] == nil {
						lastCrossShardState[fromShard] = make(map[byte]uint64)
					}
					lastHeight := lastCrossShardState[fromShard][toShard] // get last cross shard height from shardID  to crossShardShardID
					waitHeight := shardBlock.Height

					info := NextCrossShardInfo{
						waitHeight,
						shardBlock.Hash.String(),
						beaconBlock.GetHeight(),
						beaconBlock.Hash().String(),
					}
					fmt.Println("DEBUG: processBeaconForConfirmmingCrossShard ", fromShard, toShard, info)
					b, _ := json.Marshal(info)
					fmt.Println("debug StoreCrossShardNextHeight", fromShard, toShard, lastHeight, string(b))
					err := rawdbv2.StoreCrossShardNextHeight(database, fromShard, toShard, lastHeight, b)
					if err != nil {
						return err
					}
					if lastCrossShardState[fromShard] == nil {
						lastCrossShardState[fromShard] = make(map[byte]uint64)
					}
					lastCrossShardState[fromShard][toShard] = waitHeight //update lastHeight to waitHeight
				}
			}
		}
	}
	return nil
}

//periodically check pool and insert into pool (in case some fork block or block come early)
var insertBeaconTimeCache, _ = lru.New(1000)

func (s *BeaconSyncProcess) insertBeaconBlockFromPool() {

	insertCnt := 0
	defer func() {
		if insertCnt > 0 {
			s.insertBeaconBlockFromPool()
		} else {
			time.AfterFunc(time.Second*2, s.insertBeaconBlockFromPool)
		}
	}()
	//Logger.Debugf("insertBeaconBlockFromPool Start")
	//loop all current views, if there is any block connect to the view
	for _, viewHash := range s.chain.GetAllViewHash() {
		blks := s.beaconPool.GetBlockByPrevHash(viewHash)
		for _, blk := range blks {
			if blk == nil {
				continue
			}
			//Logger.Debugf("insertBeaconBlockFromPool blk %v %v", blk.GetHeight(), blk.Hash().String())
			//if already insert and error, last time insert is < 10s then we skip
			insertTime, ok := insertBeaconTimeCache.Get(viewHash.String())
			if ok && time.Since(insertTime.(time.Time)).Seconds() < 10 {
				continue
			}

			//fullnode delay 1 block (make sure insert final block)
			if os.Getenv("FULLNODE") != "" {
				preBlk := s.beaconPool.GetBlockByPrevHash(*blk.Hash())
				if len(preBlk) == 0 {
					continue
				}
			}

			insertBeaconTimeCache.Add(viewHash.String(), time.Now())
			insertCnt++
			//must validate this block when insert
			if err := s.chain.InsertBlk(blk.(types.BlockInterface), true); err != nil {
				Logger.Error("Insert beacon block from pool fail", blk.GetHeight(), blk.Hash(), err)
				continue
			}
			s.beaconPool.RemoveBlock(blk.Hash())
		}
	}

}

//sync beacon
func (s *BeaconSyncProcess) syncBeacon() {
	for {
		requestCnt := 0
		if s.status != RUNNING_SYNC {
			s.isCatchUp = false
			time.Sleep(time.Second)
			continue
		}

		for peerID, pState := range s.getBeaconPeerStates() {
			requestCnt += s.streamFromPeer(peerID, pState)
		}

		//last check, if we still need to sync more
		if requestCnt > 0 {
			s.isCatchUp = false
		} else {
			if len(s.beaconPeerStates) > 0 {
				s.isCatchUp = true
			}
			time.Sleep(time.Second * 5)
		}
	}
}

func (s *BeaconSyncProcess) streamFromPeer(peerID string, pState BeaconPeerState) (requestCnt int) {
	if pState.processed {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	blockBuffer := []types.BlockInterface{}
	defer func() {
		if requestCnt == 0 {
			pState.processed = true
		}
		cancel()
	}()

	toHeight := pState.BestViewHeight
	//process param

	//fullnode delay 1 block (make sure insert final block)
	if os.Getenv("FULLNODE") != "" {
		toHeight = toHeight - 1
	}

	if toHeight <= s.chain.GetBestViewHeight() {
		return
	}

	//stream
	ch, err := s.server.RequestBeaconBlocksViaStream(ctx, "", s.chain.GetFinalViewHeight()+1, toHeight)
	if err != nil {
		fmt.Println("Syncker: create channel fail")
		return
	}

	//receive
	requestCnt++
	insertTime := time.Now()
	for {
		select {
		case blk := <-ch:
			if !isNil(blk) {
				//Logger.Infof("Syncker beacon receive block %v", blk.GetHeight())
				blockBuffer = append(blockBuffer, blk)
			}

			if uint64(len(blockBuffer)) >= blockchain.DefaultMaxBlkReqPerPeer || (len(blockBuffer) > 0 && (isNil(blk) || time.Since(insertTime) > time.Millisecond*2000)) {
				insertBlkCnt := 0
				for {
					time1 := time.Now()
					if successBlk, err := InsertBatchBlock(s.chain, blockBuffer); err != nil {
						if successBlk == 0 {
							fmt.Println(err)
						}
						return
					} else {
						insertBlkCnt += successBlk
						Logger.Infof("Syncker Insert %d beacon block (from %d to %d) elaspse %f \n", successBlk, blockBuffer[0].GetHeight(), blockBuffer[len(blockBuffer)-1].GetHeight(), time.Since(time1).Seconds())
						if successBlk >= len(blockBuffer) || successBlk == 0 {
							break
						}
						blockBuffer = blockBuffer[successBlk:]
					}
				}
				insertTime = time.Now()
				blockBuffer = []types.BlockInterface{}
			}
			if isNil(blk) && len(blockBuffer) == 0 {
				return
			}
		}
	}
}
