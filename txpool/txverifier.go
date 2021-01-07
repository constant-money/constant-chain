package txpool

import (
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
)

// type BlockTxsVerifier interface {
// 	ValidateBlockTransactions(
// 		txP TxPool,
// 		sView interface{},
// 		bcView interface{},
// 		txs []metadata.Transaction,
// 	) bool
// 	ValidateBatchRangeProof([]metadata.Transaction) (bool, error)
// }

// type TxVerifier interface {
// 	ValidateAuthentications(metadata.Transaction) (bool, error)
// 	ValidateDataCorrectness(metadata.Transaction) (bool, error)
// 	ValidateTxZKProof(metadata.Transaction) (bool, error)

// 	ValidateWithBlockChain(
// 		tx metadata.Transaction,
// 		sView interface{},
// 		bcView interface{},
// 	) (bool, error)

// 	ValidateDoubleSpend(
// 		txs []metadata.Transaction,
// 		sView interface{},
// 		bcView interface{},
// 	) (bool, error)

// 	ValidateTxAndAddToListTxs(
// 		txNew metadata.Transaction,
// 		txs []metadata.Transaction,
// 		sView interface{},
// 		bcView interface{},
// 		better func(txA, txB metadata.Transaction) bool, // return true if we want txA when txA and txB is double spending
// 	) (bool, error)

// 	FilterDoubleSpend(
// 		txs []metadata.Transaction,
// 		sView interface{},
// 		bcView interface{},
// 		better func(txA, txB metadata.Transaction) bool, // return true if we want txA when txA and txB is double spending
// 	) ([]metadata.Transaction, error)
// }

type TxsVerifier struct {
	txDB statedb.StateDB
}

func (v *TxsVerifier) ValidateWithoutChainState(tx metadata.Transaction) (bool, error) {
	if err := tx.LoadCommitment(&v.txDB); err != nil {
		return false, err
	}
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
	//Get state db from beaconview
	if err := tx.LoadCommitment(&v.txDB); err != nil {
		return false, err
	}

	ok, err := tx.ValidateSanityDataWithBlockchain(
		chainRetriever,
		shardViewRetriever,
		beaconViewRetriever,
		beaconHeight,
	)
	if !ok || err != nil {
		return ok, err
	}
	return tx.ValidateDoubleSpendWithBlockChain(&v.txDB)
}

// func (v *TxsVerifier) ValidateBlockTransactions(
// 	sView interface{},
// 	bcView interface{},
// 	txs []metadata.Transaction,
// ) bool {
// 	v.
// 	panic("Implement me")
// 	// return false
// }

func (v *TxsVerifier) ValidateTxAndAddToListTxs(
	txNew metadata.Transaction,
	txs []metadata.Transaction,
	sView interface{},
	bcView interface{},
	better func(txA, txB metadata.Transaction) bool, // return true if we want txA when txA and txB is double spending
) (
	bool,
	error,
) {
	panic("Implement me")
	// return false, nil
}

func (v *TxsVerifier) FilterDoubleSpend(
	txs []metadata.Transaction,
	sView interface{},
	bcView interface{},
	better func(txA, txB metadata.Transaction) bool, // return true if we want txA when txA and txB is double spending
) (
	[]metadata.Transaction,
	error,
) {
	panic("Implement me")
	// return false, nil
}
