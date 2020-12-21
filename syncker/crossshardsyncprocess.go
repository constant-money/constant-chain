package syncker

import (
	"context"
	"fmt"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"

	"github.com/incognitochain/incognito-chain/common"
)

type CrossShardSyncProcess struct {
	status           string //stop, running
	blockchain       *blockchain.BlockChain
	network          Network
	shardID          int
	shardSyncProcess *ShardSyncProcess
	beaconChain      BeaconChainInterface
	crossShardPool   *BlkPool
	actionCh         chan func()
}

type CrossXReq struct {
	height uint64
	time   *time.Time
}

func NewCrossShardSyncProcess(network Network, bc *blockchain.BlockChain, shardSyncProcess *ShardSyncProcess, beaconChain BeaconChainInterface) *CrossShardSyncProcess {

	var isOutdatedBlock = func(blk interface{}) bool {
		if blk.(*blockchain.CrossShardBlock).GetHeight() < shardSyncProcess.Chain.GetCrossShardState()[byte(blk.(*blockchain.CrossShardBlock).GetHeight())] {
			return true
		}
		return false
	}

	s := &CrossShardSyncProcess{
		status:           STOP_SYNC,
		blockchain:       bc,
		network:          network,
		beaconChain:      beaconChain,
		shardSyncProcess: shardSyncProcess,
		crossShardPool:   NewBlkPool("crossshard", isOutdatedBlock),
		shardID:          shardSyncProcess.shardID,
		actionCh:         make(chan func()),
	}

	go s.syncCrossShard()

	go func() {
		for {
			f := <-s.actionCh
			f()
		}
	}()

	return s
}

func (s *CrossShardSyncProcess) start() bool {
	if s.status == RUNNING_SYNC {
		return false
	}
	s.status = RUNNING_SYNC

	return true
}

func (s *CrossShardSyncProcess) stop() {
	s.status = STOP_SYNC
}

func (s *CrossShardSyncProcess) InsertCrossShardBlock(blk interface{}) {
	s.crossShardPool.AddBlock(blk.(common.BlockPoolInterface))
}

//check beacon state and retrieve needed crossshard block, then add to request pool
func (s *CrossShardSyncProcess) syncCrossShard() {
	for {
		reqCnt := 0
		//only run when shard is validator and sync shard is finish
		if s.status != RUNNING_SYNC || !s.shardSyncProcess.isCatchUp {
			time.Sleep(time.Second * 5)
			continue
		}

		//get chain crossshard state and collect all missing crossshard block
		lastRequestCrossShard := s.shardSyncProcess.Chain.GetCrossShardState()
		missingCrossShardBlock := make(map[byte][][]byte)
		for i := 0; i < s.blockchain.GetChainParams().ActiveShards; i++ {
			for {
				if i == s.shardID {
					break
				}
				requestHeight := lastRequestCrossShard[byte(i)]
				nextCrossShardInfo := s.blockchain.FetchNextCrossShard(i, int(s.shardID), requestHeight)
				if nextCrossShardInfo == nil {
					break
				}

				lastRequestCrossShard[byte(i)] = nextCrossShardInfo.NextCrossShardHeight
				h, _ := common.Hash{}.NewHashFromStr(nextCrossShardInfo.NextCrossShardHash)
				if s.crossShardPool.GetBlock(*h) != nil {
					continue
				}
				reqCnt++
				blkHash, _ := common.Hash{}.NewHashFromStr(nextCrossShardInfo.NextCrossShardHash)

				missingCrossShardBlock[byte(i)] = append(missingCrossShardBlock[byte(i)], blkHash.Bytes())
			}
			//fmt.Println("debug syncCrossShard", i, len(missingCrossShardBlock[byte(i)]))
			if len(missingCrossShardBlock[byte(i)]) > 0 {
				s.streamMissingCrossShardBlock(i, missingCrossShardBlock[byte(i)])
			}
		}

		//if no request, we wait 5s, before check again
		if reqCnt == 0 {
			time.Sleep(time.Second * 5)
		}
	}
}

func (s *CrossShardSyncProcess) streamMissingCrossShardBlock(fromSID int, hashes [][]byte) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	//stream
	ch, err := s.network.RequestCrossShardBlocksByHashViaStream(ctx, "", fromSID, s.shardID, hashes)
	if err != nil {
		fmt.Println("Syncker: create channel fail")
		return
	}

	//receive
	for blk := range ch {
		if !isNil(blk) {
			fmt.Println("syncker: Insert crossShard block", blk.GetHeight(), blk.Hash().String())
			s.crossShardPool.AddBlock(blk.(common.BlockPoolInterface))
		} else {
			return
		}
	}
}
