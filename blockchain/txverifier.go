package blockchain

import (
	"fmt"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/txpool"
	"github.com/pkg/errors"
)

type TxsVerifier struct {
	txDB   *statedb.StateDB
	txPool txpool.TxPool
}

func (v *TxsVerifier) UpdateTransactionStateDB(
	newSDB *statedb.StateDB,
) {
	v.txDB = newSDB
}

func NewTxsVerifier(
	txDB *statedb.StateDB,
	tp txpool.TxPool,
) txpool.TxVerifier {
	x := &TxsVerifier{
		txDB:   txDB,
		txPool: tp,
	}
	return x
}

func (v *TxsVerifier) LoadCommitment(
	tx metadata.Transaction,
	shardViewRetriever metadata.ShardViewRetriever,
) bool {
	sDB := v.txDB
	if shardViewRetriever != nil {
		sDB = shardViewRetriever.GetCopiedTransactionStateDB()
	}
	err := tx.LoadCommitment(sDB.Copy())
	if err != nil {
		fmt.Println("Can not load commitment of this tx %v, error: %v", tx.Hash().String(), err)
		return false
	}
	return true
}

func (v *TxsVerifier) LoadCommitmentForTxs(
	txs []metadata.Transaction,
	shardViewRetriever metadata.ShardViewRetriever,
) bool {
	sDB := v.txDB
	if shardViewRetriever != nil {
		sDB = shardViewRetriever.GetCopiedTransactionStateDB()
	}
	for _, tx := range txs {
		err := tx.LoadCommitment(sDB.Copy())
		if err != nil {
			fmt.Printf("[testNewPool] Can not load commitment of this tx %v, error: %v\n", tx.Hash().String(), err)
			return false
		}
	}
	return true
}

func (v *TxsVerifier) ValidateTxsSig(
	txs []metadata.Transaction,
	errCh chan error,
	doneCh chan interface{},
) {
	for _, tx := range txs {
		go func() {
			ok, err := tx.VerifySigTx()
			if !ok || err != nil {
				errCh <- errors.Errorf("Signature of tx %v is not valid, result %v, error %v", tx.Hash().String(), ok, err)
			} else {
				doneCh <- nil
			}
		}()
	}
}

func (v *TxsVerifier) ValidateWithoutChainstate(tx metadata.Transaction) (bool, error) {
	ok, err := tx.ValidateSanityDataByItSelf()
	if !ok || err != nil {
		return ok, err
	}
	return tx.ValidateTxCorrectness()
}

func (v *TxsVerifier) ValidateWithChainState(
	tx metadata.Transaction,
	chainRetriever metadata.ChainRetriever,
	shardViewRetriever metadata.ShardViewRetriever,
	beaconViewRetriever metadata.BeaconViewRetriever,
	beaconHeight uint64,
) (bool, error) {
	ok, err := tx.ValidateSanityDataWithBlockchain(
		chainRetriever,
		shardViewRetriever,
		beaconViewRetriever,
		beaconHeight,
	)
	if !ok || err != nil {
		return ok, err
	}
	return tx.ValidateDoubleSpendWithBlockChain(shardViewRetriever.GetCopiedTransactionStateDB())
}

func (v *TxsVerifier) ValidateBlockTransactions(
	chainRetriever metadata.ChainRetriever,
	shardViewRetriever metadata.ShardViewRetriever,
	beaconViewRetriever metadata.BeaconViewRetriever,
	txs []metadata.Transaction,
) bool {
	fmt.Printf("[testNewPool] Total txs %v\n", len(txs))
	if len(txs) == 0 {
		return true
	}
	_, newTxs := v.txPool.CheckValidatedTxs(txs)
	// fmt.Println("Is Validated")
	errCh := make(chan error)
	doneCh := make(chan interface{}, len(txs)+2*len(newTxs))
	numOfValidTxs := 0
	timeout := time.After(10 * time.Second)
	v.LoadCommitmentForTxs(
		txs,
		shardViewRetriever,
	)
	v.ValidateTxsSig(
		newTxs,
		errCh,
		doneCh,
	)
	v.validateTxsWithoutChainstate(
		newTxs,
		errCh,
		doneCh,
	)
	v.validateTxsWithChainstate(
		txs,
		chainRetriever,
		shardViewRetriever,
		beaconViewRetriever,
		errCh,
		doneCh,
	)
	fmt.Println("[testNewPool] wait!")
	for {
		select {
		case err := <-errCh:
			fmt.Println(err)
			return false
		case <-doneCh:
			numOfValidTxs++
			fmt.Printf("[testNewPool] %v %v\n", numOfValidTxs, len(txs))
			if numOfValidTxs == len(txs) {
				fmt.Println("[testNewPool] wait!")
				ok, err := v.checkDoubleSpendInListTxs(txs)
				if (!ok) || (err != nil) {
					fmt.Println(err)
					return false
				}
				return true
			}
		case <-timeout:
			fmt.Println("Timeout!!!")
			return false
		}
	}
}

func (v *TxsVerifier) validateTxsWithoutChainstate(
	txs []metadata.Transaction,
	errCh chan error,
	doneCh chan interface{},
) {
	for _, tx := range txs {
		go func() {
			ok, err := v.ValidateWithoutChainstate(tx)
			if !ok || err != nil {
				errCh <- errors.Errorf("[testNewPool] This list txs contains a invalid tx %v, validate result %v, error %v", tx.Hash().String(), ok, err)
			}
			if err != nil {
				fmt.Printf("[testNewPool] Validate tx %v return error %v:\n", tx.Hash().String(), err)
			} else {
				fmt.Printf("[testNewPool] Validate tx %v\n", tx.Hash().String())
				doneCh <- nil

			}
		}()
	}
}

func (v *TxsVerifier) validateTxsWithChainstate(
	txs []metadata.Transaction,
	cView metadata.ChainRetriever,
	sView metadata.ShardViewRetriever,
	bcView metadata.BeaconViewRetriever,
	errCh chan error,
	doneCh chan interface{},
) {
	// MAX := runtime.NumCPU() - 1
	// nWorkers := make(chan int, MAX)
	for _, tx := range txs {
		// nWorkers <- 1
		go func() {
			ok, err := v.ValidateWithChainState(
				tx,
				cView,
				sView,
				bcView,
				sView.GetBeaconHeight(),
			)
			if !ok || err != nil {
				errCh <- errors.Errorf("[testNewPool] This list txs contains a invalid tx %v, validate result %v, error %v", tx.Hash().String(), ok, err)
				if err != nil {
					fmt.Printf("[testNewPool] Validate tx %v return error %v:\n", tx.Hash().String(), err)
				}
			} else {
				fmt.Printf("[testNewPool] Validate tx %v\n", tx.Hash().String())
				doneCh <- nil
			}
			// <-nWorkers
		}()
	}
}

func (v *TxsVerifier) checkDoubleSpendInListTxs(
	txs []metadata.Transaction,
) (
	bool,
	error,
) {
	mapForChkDbSpend := map[[privacy.Ed25519KeySize]byte]interface{}{}
	for _, tx := range txs {

		prf := tx.GetProof()
		if prf == nil {
			continue
		}
		iCoins := prf.GetInputCoins()
		oCoins := prf.GetOutputCoins()
		for _, iCoin := range iCoins {
			if _, ok := mapForChkDbSpend[iCoin.CoinDetails.GetSerialNumber().ToBytes()]; ok {
				return false, errors.Errorf("List txs contain double spend tx %v", tx.Hash().String())
			} else {
				mapForChkDbSpend[iCoin.CoinDetails.GetSerialNumber().ToBytes()] = nil
			}
		}
		for _, oCoin := range oCoins {
			if _, ok := mapForChkDbSpend[oCoin.CoinDetails.GetSNDerivator().ToBytes()]; ok {
				return false, errors.Errorf("List txs contain double spend tx %v", tx.Hash().String())
			} else {
				mapForChkDbSpend[oCoin.CoinDetails.GetSNDerivator().ToBytes()] = nil
			}
		}
		if tx.GetType() == common.TxCustomTokenPrivacyType {
			txNormal := tx.(*transaction.TxCustomTokenPrivacy).TxPrivacyTokenData.TxNormal
			iCoins := txNormal.GetProof().GetInputCoins()
			oCoins := txNormal.GetProof().GetOutputCoins()
			for _, iCoin := range iCoins {
				if _, ok := mapForChkDbSpend[iCoin.CoinDetails.GetSerialNumber().ToBytes()]; ok {
					return false, errors.Errorf("List txs contain double spend tx %v", tx.Hash().String())
				} else {
					mapForChkDbSpend[iCoin.CoinDetails.GetSerialNumber().ToBytes()] = nil
				}
			}
			for _, oCoin := range oCoins {
				if _, ok := mapForChkDbSpend[oCoin.CoinDetails.GetSNDerivator().ToBytes()]; ok {
					return false, errors.Errorf("List txs contain double spend tx %v", tx.Hash().String())
				} else {
					mapForChkDbSpend[oCoin.CoinDetails.GetSNDerivator().ToBytes()] = nil
				}
			}
		}
	}
	return true, nil
}
