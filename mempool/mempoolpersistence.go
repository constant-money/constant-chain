package mempool

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/blockchain"
	"strings"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/databasemp/lvdb"
	"github.com/incognitochain/incognito-chain/transaction"
)

type TempDesc struct {
	StartTime     time.Time
	IsPushMessage bool
	Height        uint64
	Fee           uint64
	FeePerKB      int32
}

// addTransactionToDatabaseMempool - Add a transaction data into mempool database
func (tp *TxPool) addTransactionToDatabaseMempool(txHash *common.Hash, txDesc TxDesc) error {
	tx := txDesc.Desc.Tx
	tempDesc := TempDesc{
		StartTime:     txDesc.StartTime,
		IsPushMessage: txDesc.IsFowardMessage,
		Height:        txDesc.Desc.Height,
		Fee:           txDesc.Desc.Fee,
		FeePerKB:      txDesc.Desc.FeePerKB,
	}

	if tx.GetType() == common.TxNormalType || tx.GetType() == common.TxCustomTokenPrivacyType {
		valueTx, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		valueDesc, err := json.Marshal(tempDesc)
		if err != nil {
			return err
		}
		err = tp.config.DataBaseMempool.AddTransaction(txHash, tx.GetType(), valueTx, valueDesc)
		if err != nil {
			return err
		}
	}
	return nil
}

// getTransactionFromDatabaseMempool - get tx from mempool database
func (tp *TxPool) getTransactionFromDatabaseMempool(txHash *common.Hash) (*TxDesc, error) {
	value, err := tp.config.DataBaseMempool.GetTransaction(txHash)
	values := strings.Split(string(value), string(lvdb.Splitter))
	if len(values) != 3 {
		return nil, err
	}
	txDesc, err := unMarshallTxDescFromDatabase(values[0], []byte(values[1]), []byte(values[2]))
	if err != nil {
		return nil, err
	}
	return txDesc, nil
}

// resetDatabaseMempool - reset data in data mempool
func (tp *TxPool) resetDatabaseMempool() error {
	return tp.config.DataBaseMempool.Reset()
}

// loadDatabaseMP - Get all tx in mempool database persistence
func (tp *TxPool) loadDatabaseMP() ([]TxDesc, error) {
	txDescs := []TxDesc{}
	allTxHashes, allTxs, err := tp.config.DataBaseMempool.Load()
	ttl := time.Duration(tp.config.TxLifeTime) * time.Second
	if err != nil {
		return txDescs, err
	}
	for index, tx := range allTxs {
		values := strings.Split(string(tx), string(lvdb.Splitter))
		txDesc, err := unMarshallTxDescFromDatabase(values[0], []byte(values[1]), []byte(values[2]))
		if err != nil {
			Logger.log.Error(err)
			txHash, err := common.Hash{}.NewHash(allTxHashes[index][3:])
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			// fail to ummarshall transaction then remove
			err1 := tp.removeTransactionFromDatabaseMP(txHash)
			if err1 != nil {
				Logger.log.Error(err1)
			}
			continue
		}
		//if transaction is timeout then remove
		if time.Since(txDesc.StartTime) > ttl {
			err1 := tp.removeTransactionFromDatabaseMP(txDesc.Desc.Tx.Hash())
			if err1 != nil {
				Logger.log.Error(err1)
			}
		}
		//if not validated by current blockchain db then remove
		senderShardID := common.GetShardIDFromLastByte(txDesc.Desc.Tx.GetSenderAddrLastByte())
		beaconView := tp.config.BlockChain.BeaconChain.GetFinalView().(*blockchain.BeaconBestState)
		shardView := tp.config.BlockChain.ShardChain[senderShardID].GetBestView().(*blockchain.ShardBestState)
		err = tp.validateTransaction(shardView, beaconView, txDesc.Desc.Tx, -1, false, false)
		if err != nil {
			Logger.log.Error(err)
			err1 := tp.removeTransactionFromDatabaseMP(txDesc.Desc.Tx.Hash())
			if err1 != nil {
				Logger.log.Error(err1)
			}
			continue
		}

		err = tp.addTx(txDesc, false)
		if err != nil {
			Logger.log.Error(err)
		}
		txDescs = append(txDescs, *txDesc)
	}
	return txDescs, nil
}

// removeTransactionFromDatabaseMP - remove tx from mempool db persistence
func (tp *TxPool) removeTransactionFromDatabaseMP(txHash *common.Hash) error {
	if has, _ := tp.config.DataBaseMempool.HasTransaction(txHash); has {
		err := tp.config.DataBaseMempool.RemoveTransaction(txHash)
		return err
	}
	return nil
}

// unMarshallTxDescFromDatabase - convert tx data in mempool database persistence into TxDesc
func unMarshallTxDescFromDatabase(txType string, valueTx []byte, valueDesc []byte) (*TxDesc, error) {
	var err error
	txDesc := TxDesc{}

	switch txType {
	case common.TxNormalType:
		{
			txDesc.Desc.Tx, err = transaction.NewTransactionFromJsonBytes(valueTx)
			if err != nil {
				return nil, err
			}
		}
	case common.TxCustomTokenPrivacyType:
		{
			txDesc.Desc.Tx, err = transaction.NewTransactionTokenFromJsonBytes(valueTx)
			if err != nil {
				return nil, err
			}
		}
	}
	tempDesc := TempDesc{}
	err = json.Unmarshal(valueDesc, &tempDesc)
	if err != nil {
		return nil, err
	}
	txDesc.IsFowardMessage = tempDesc.IsPushMessage
	txDesc.StartTime = tempDesc.StartTime
	txDesc.Desc.Height = tempDesc.Height
	txDesc.Desc.Fee = tempDesc.Fee
	txDesc.Desc.FeePerKB = tempDesc.FeePerKB
	return &txDesc, nil
}
