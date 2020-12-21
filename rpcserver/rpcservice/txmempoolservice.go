package rpcservice

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy/coin"
)

type TxMemPoolService struct {
	TxMemPool MempoolInterface
}

func (txMemPoolService TxMemPoolService) GetPoolCandidate() map[common.Hash]string {
	return txMemPoolService.TxMemPool.GetClonedPoolCandidate()
}


func (txMemPoolService TxMemPoolService) FilterMemPoolPlainCoinsToSpent(outCoins []coin.PlainCoin) ([]coin.PlainCoin, error) {
	remainOutputCoins := make([]coin.PlainCoin, 0)

	for _, outCoin := range outCoins {
		if txMemPoolService.TxMemPool.ValidateSerialNumberHashH(outCoin.GetKeyImage().ToBytesS()) == nil {
			remainOutputCoins = append(remainOutputCoins, outCoin)
		}
	}
	return remainOutputCoins, nil
}

func (txMemPoolService TxMemPoolService) GetNumberOfTxsInMempool() int {
	return len(txMemPoolService.TxMemPool.ListTxs())
}

func (txMemPoolService TxMemPoolService) MempoolEntry(txIDString string) (metadata.Transaction, byte, *RPCError) {
	txID, err := common.Hash{}.NewHashFromStr(txIDString)
	if err != nil {
		Logger.log.Debugf("handleMempoolEntry result: nil %+v", err)
		return nil, byte(0), NewRPCError(RPCInvalidParamsError, err)
	}

	txInPool, err := txMemPoolService.TxMemPool.GetTx(txID)
	if err != nil {
		Logger.log.Error(err)
		return nil, byte(0), NewRPCError(GeTxFromPoolError, err)
	}
	shardIDTemp := common.GetShardIDFromLastByte(txInPool.GetSenderAddrLastByte())

	return txInPool, shardIDTemp, nil
}

func (txMemPoolService *TxMemPoolService) RemoveTxInMempool(txIDString string) (bool, *RPCError) {
	txID, err := common.Hash{}.NewHashFromStr(txIDString)
	if err != nil {
		Logger.log.Debugf("RemoveTxInMempool result: nil %+v", err)
		return false, NewRPCError(RPCInvalidParamsError, err)
	}

	tempTx, err := txMemPoolService.TxMemPool.GetTx(txID)
	if err != nil {
		return false, NewRPCError(GeTxFromPoolError, err)
	}
	txMemPoolService.TxMemPool.RemoveTx([]metadata.Transaction{tempTx}, false)
	txMemPoolService.TxMemPool.TriggerCRemoveTxs(tempTx)

	return true, nil
}
