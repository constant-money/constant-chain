package jsonresult

import (
	"sort"

	"github.com/incognitochain/incognito-chain/metadata"
)

type GetMempoolInfo struct {
	Size          int                `json:"Size"`
	Bytes         uint64             `json:"Bytes"`
	Usage         uint64             `json:"Usage"`
	MaxMempool    uint64             `json:"MaxMempool"`
	MempoolMinFee uint64             `json:"MempoolMinFee"`
	MempoolMaxFee uint64             `json:"MempoolMaxFee"`
	ListTxs       []GetMempoolInfoTx `json:"ListTxs"`
}

func NewGetMempoolInfo(txMempool interface {
	MaxFee() uint64
	ListTxsDetail() []metadata.Transaction
	Count() int
	Size() uint64
}) *GetMempoolInfo {
	result := &GetMempoolInfo{
		Size:          txMempool.Count(),
		Bytes:         txMempool.Size(),
		MempoolMaxFee: txMempool.MaxFee(),
	}
	// get list data from mempool
	listTxsDetail := txMempool.ListTxsDetail()
	if len(listTxsDetail) > 0 {
		result.ListTxs = make([]GetMempoolInfoTx, 0)
		for _, tx := range listTxsDetail {
			item := NewGetMempoolInfoTx(tx)
			result.ListTxs = append(result.ListTxs, *item)
		}
	}
	// sort for time
	if len(result.ListTxs) > 0 {
		sort.Slice(result.ListTxs, func(i, j int) bool {
			return result.ListTxs[i].LockTime >= result.ListTxs[j].LockTime
		})
	}
	return result
}

type GetMempoolInfoTx struct {
	TxID     string `json:"TxID"`
	LockTime int64  `json:"LockTime"`
}

func NewGetMempoolInfoTx(tx metadata.Transaction) *GetMempoolInfoTx {
	result := &GetMempoolInfoTx{
		LockTime: tx.GetLockTime(),
		TxID:     tx.Hash().String(),
	}
	return result
}

type GetRawMempoolResult struct {
	TxHashes []string
}

func NewGetRawMempoolResult(txMemPool interface{ ListTxs() []string }) *GetRawMempoolResult {
	result := &GetRawMempoolResult{
		TxHashes: txMemPool.ListTxs(),
	}
	return result
}

type GetPendingTxsInBlockgenResult struct {
	TxHashes []string
}

func NewGetPendingTxsInBlockgenResult(txs []metadata.Transaction) GetPendingTxsInBlockgenResult {
	txHashes := []string{}
	for _, tx := range txs {
		txHash := tx.Hash().String()
		txHashes = append(txHashes, txHash)
	}
	return GetPendingTxsInBlockgenResult{TxHashes: txHashes}
}
