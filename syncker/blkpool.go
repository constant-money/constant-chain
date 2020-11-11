package syncker

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"time"

	"github.com/incognitochain/incognito-chain/common"
)

type BlkPool struct {
	action            chan func()
	blkPoolByHash     map[string]types.BlockPoolInterface // hash -> block
	blkPoolByPrevHash map[string][]string                 // prevhash -> []nexthash
}

func NewBlkPool(name string, IsOutdatedBlk func(interface{}) bool) *BlkPool {
	pool := new(BlkPool)
	pool.action = make(chan func())
	pool.blkPoolByHash = make(map[string]types.BlockPoolInterface)
	pool.blkPoolByPrevHash = make(map[string][]string)
	go pool.Start()

	//remove outdated block in pool, only trigger if pool has more than 1000 blocks
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		for {
			<-ticker.C
			//remove block from blkPoolByHash
			if pool.GetPoolSize() > 100 {
				blkList := pool.GetBlockList()
				for _, blk := range blkList {
					if IsOutdatedBlk(blk) {
						pool.RemoveBlock(blk.Hash())
					}
				}
			}

			//remove prehash block pointer if it point to nothing
			if len(pool.blkPoolByPrevHash) > 100 {
				blkList := pool.GetPrevHashPool()
				for prevhash, hashes := range blkList {
					stillPointToABlock := false
					for _, hash := range hashes {
						h, _ := common.Hash{}.NewHashFromStr(hash)
						if pool.HasHash(*h) {
							stillPointToABlock = true
						}
					}
					if !stillPointToABlock {
						pool.RemovePrevHash(prevhash)
					}

				}
			}
		}
	}()
	return pool
}

func (pool *BlkPool) Start() {
	ticker := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case f := <-pool.action:
			f()
		case <-ticker.C:
		}
	}
}

func (pool *BlkPool) GetPoolSize() int {
	return len(pool.blkPoolByHash)
}

func (pool *BlkPool) GetPrevHashPool() map[string][]string {
	res := make(chan map[string][]string)
	pool.action <- func() {
		prevHashPool := make(map[string][]string)
		for preBlk, blks := range pool.blkPoolByPrevHash {
			prevHashPool[preBlk] = blks
		}
		res <- prevHashPool
	}
	return <-res
}

func (pool *BlkPool) GetBlockList() []types.BlockPoolInterface {
	res := make(chan []types.BlockPoolInterface)
	pool.action <- func() {
		blkList := []types.BlockPoolInterface{}
		for _, blk := range pool.blkPoolByHash {
			blkList = append(blkList, blk)
		}
		res <- blkList
	}
	return <-res
}

func (pool *BlkPool) AddBlock(blk types.BlockPoolInterface) {
	pool.action <- func() {
		prevHash := blk.GetPrevHash().String()
		hash := blk.Hash().String()
		//if exists, return
		if _, ok := pool.blkPoolByHash[hash]; ok {
			return
		}
		pool.blkPoolByHash[hash] = blk
		//insert into prehash datastructure
		if common.IndexOfStr(hash, pool.blkPoolByPrevHash[prevHash]) > -1 {
			return
		}
		pool.blkPoolByPrevHash[prevHash] = append(pool.blkPoolByPrevHash[prevHash], hash)
		//fmt.Println("Syncker: add block to pool", blk.GetHeight())
	}
}

func (pool *BlkPool) GetBlock(hash common.Hash) types.BlockPoolInterface {
	res := make(chan types.BlockPoolInterface)
	pool.action <- func() {
		blk, _ := pool.blkPoolByHash[hash.String()]
		res <- blk
	}
	return <-res
}

func (pool *BlkPool) HasHash(hash common.Hash) bool {
	res := make(chan bool)
	pool.action <- func() {
		_, ok := pool.blkPoolByHash[hash.String()]
		res <- ok
	}
	return <-res
}

func (pool *BlkPool) RemoveBlock(hash *common.Hash) {
	pool.action <- func() {
		delete(pool.blkPoolByHash, hash.String())
	}
}

func (pool *BlkPool) RemovePrevHash(hash string) {
	pool.action <- func() {
		delete(pool.blkPoolByPrevHash, hash)
	}
}

func (pool *BlkPool) GetPoolInfo() []types.BlockPoolInterface {
	res := make(chan []types.BlockPoolInterface)
	pool.action <- func() {
		res <- GetPoolInfo(pool.blkPoolByHash)
	}
	return <-res
}

func (pool *BlkPool) GetLatestHeight(currentHash string) uint64 {
	longest := pool.GetLongestChain(currentHash)
	if len(longest) > 0 {
		return longest[len(longest)-1].GetHeight()
	}
	return 0
}

// END OF COMMON FUNCTION =======================================================================

// START OF SPECIAL CASE FUNCTION =======================================================================

//When get s2b block for producer
//Get Block from current hash to final block
func (pool *BlkPool) GetFinalBlockFromBlockHash(currentHash string) []types.BlockPoolInterface {
	res := make(chan []types.BlockPoolInterface)
	pool.action <- func() {
		res <- GetFinalBlockFromBlockHash_v1(currentHash, pool.blkPoolByHash, pool.blkPoolByPrevHash)
	}
	return <-res
}

//When get last block for s2b synchronization
//Get longest branch in pool
func (pool *BlkPool) GetLongestChain(currentHash string) []types.BlockPoolInterface {
	res := make(chan []types.BlockPoolInterface)
	pool.action <- func() {
		res <- GetLongestChain(currentHash, pool.blkPoolByHash, pool.blkPoolByPrevHash)
	}
	return <-res
}

func (pool *BlkPool) GetBlockByPrevHash(prevHash common.Hash) []types.BlockPoolInterface {
	res := make(chan []types.BlockPoolInterface)
	pool.action <- func() {
		res <- GetBlksByPrevHash(prevHash.String(), pool.blkPoolByHash, pool.blkPoolByPrevHash)
	}
	return <-res
}

func (pool *BlkPool) GetAllViewByHash(rHash string) []types.BlockPoolInterface {
	res := make(chan []types.BlockPoolInterface)
	pool.action <- func() {
		res <- GetAllViewFromHash(rHash, pool.blkPoolByHash, pool.blkPoolByPrevHash)
	}
	return <-res
}
