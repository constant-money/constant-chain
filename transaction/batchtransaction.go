package transaction

import (
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge/aggregaterange"
)

type batchTransaction struct {
	txs []metadata.Transaction
}

func NewBatchTransaction(txs []metadata.Transaction) *batchTransaction {
	return &batchTransaction{txs: txs}
}

func (b *batchTransaction) AddTxs(txs []metadata.Transaction) {
	b.txs = append(b.txs, txs...)
}

func (b *batchTransaction) Validate(transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, boolParams map[string]bool) (bool, error, int) {
	return b.validateBatchTxsByItself(b.txs, transactionStateDB, bridgeStateDB, boolParams)
}

func (b *batchTransaction) validateBatchTxsByItself(txList []metadata.Transaction, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, boolParams map[string]bool) (bool, error, int) {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return false, err, -1
	}
	bulletProofList := make([]*aggregaterange.AggregatedRangeProof, 0)
	for i, tx := range txList {
		shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
		hasPrivacy := tx.IsPrivacy()

		boolParams["hasPrivacy"] = tx.IsPrivacy()
		ok, err := tx.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, prvCoinID)
		if !ok {
			return false, err, i
		}
		if tx.GetMetadata() != nil {
			if hasPrivacy {
				return false, errors.New("Metadata can not exist in privacy tx"), i
			}
			validateMetadata := tx.GetMetadata().ValidateMetadataByItself()
			if !validateMetadata {
				return validateMetadata, NewTransactionErr(UnexpectedError, errors.New("Metadata is invalid")), i
			}
		}

		if hasPrivacy {
			if bulletProof := tx.GetProof().GetAggregatedRangeProof(); bulletProof != nil {
				bulletProofList = append(bulletProofList, bulletProof)
			}
		}
	}
	//TODO: add go routine
	isNewZKP, ok := boolParams["isNewZKP"]
	if !ok {
		isNewZKP = true
	}

	if isNewZKP {
		ok, err, index := aggregaterange.VerifyBatch(bulletProofList)
		if err != nil {
			return false, NewTransactionErr(TxProofVerifyFailError, err), -1
		}
		if !ok {
			Logger.log.Errorf("FAILED VERIFICATION BATCH PAYMENT PROOF VER 2 %d", index)
			return false, NewTransactionErr(TxProofVerifyFailError, fmt.Errorf("FAILED VERIFICATION BATCH PAYMENT PROOF %d", index)), index
		}
		return true, nil, -1
	} else {
		ok, err, index := aggregaterange.VerifyBatchOld(bulletProofList)
		if err != nil {
			return false, NewTransactionErr(TxProofVerifyFailError, err), -1
		}
		if !ok {
			Logger.log.Errorf("FAILED VERIFICATION BATCH PAYMENT PROOF VER 1 %d", index)
			return false, NewTransactionErr(TxProofVerifyFailError, fmt.Errorf("FAILED VERIFICATION BATCH PAYMENT PROOF %d", index)), index
		}
		return true, nil, -1
	}
}

