package blockchain

import (
	"strconv"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
)

type BlockGenerator struct {
	// blockpool   BlockPool
	txPool      TxPool
	syncker     Syncker
	chain       *BlockChain
	CQuit       chan struct{}
	CPendingTxs <-chan metadata.Transaction
	CRemovedTxs <-chan metadata.Transaction
	PendingTxs  map[common.Hash]metadata.Transaction
	mtx         sync.RWMutex
}

func NewBlockGenerator(txPool TxPool, chain *BlockChain, syncker Syncker, cPendingTxs chan metadata.Transaction, cRemovedTxs chan metadata.Transaction) (*BlockGenerator, error) {
	return &BlockGenerator{
		txPool:      txPool,
		syncker:     syncker,
		chain:       chain,
		PendingTxs:  make(map[common.Hash]metadata.Transaction),
		CPendingTxs: cPendingTxs,
		CRemovedTxs: cRemovedTxs,
	}, nil
}

func (blockGenerator *BlockGenerator) Start(cQuit chan struct{}) {
	Logger.log.Critical("Block Gen is starting")
	for w := 0; w < WorkerNumber; w++ {
		go blockGenerator.AddTransactionV2Worker(blockGenerator.CPendingTxs)
	}
	for w := 0; w < WorkerNumber; w++ {
		go blockGenerator.RemoveTransactionV2Worker(blockGenerator.CRemovedTxs)
	}
	for {
		select {
		case <-cQuit:
			return
		}
	}
}
func (blockGenerator *BlockGenerator) AddTransactionV2(tx metadata.Transaction) {
	blockGenerator.mtx.Lock()
	defer blockGenerator.mtx.Unlock()
	blockGenerator.PendingTxs[*tx.Hash()] = tx
}
func (blockGenerator *BlockGenerator) AddTransactionV2Worker(cPendingTx <-chan metadata.Transaction) {
	for tx := range cPendingTx {
		blockGenerator.AddTransactionV2(tx)
		time.Sleep(time.Nanosecond)
	}
}
func (blockGenerator *BlockGenerator) RemoveTransactionV2(tx metadata.Transaction) {
	blockGenerator.mtx.Lock()
	defer blockGenerator.mtx.Unlock()
	delete(blockGenerator.PendingTxs, *tx.Hash())
}
func (blockGenerator *BlockGenerator) RemoveTransactionV2Worker(cRemoveTx <-chan metadata.Transaction) {
	for tx := range cRemoveTx {
		blockGenerator.RemoveTransactionV2(tx)
		time.Sleep(time.Nanosecond)
	}
}
func (blockGenerator *BlockGenerator) GetPendingTxsV2(shardID byte) []metadata.Transaction {
	blockGenerator.mtx.Lock()
	defer blockGenerator.mtx.Unlock()
	pendingTxs := []metadata.Transaction{}
	for _, tx := range blockGenerator.PendingTxs {
		txShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
		if shardID != 255 && txShardID != shardID {
			continue
		}
		// TODO: remove after test
		if string(tx.GetInfo()) != "" {
			i, e := strconv.Atoi(string(tx.GetInfo()))
			if e != nil || time.Now().Unix() < int64(i) {
				continue
			}
		}
		// End Remove
		pendingTxs = append(pendingTxs, tx)
	}
	return pendingTxs
}
