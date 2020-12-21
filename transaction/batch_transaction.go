package transaction

import (
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/privacy"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction/utils"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v1/zeroknowledge/aggregatedrange"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/bulletproofs"
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
	var bulletProofListVer1 []*privacy.AggregatedRangeProofV1
	var bulletProofListVer2 []*privacy.AggregatedRangeProofV2

	for i, tx := range txList {
		shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
		boolParams["hasPrivacy"] = tx.IsPrivacy()

		ok, batchableProofs, err := tx.ValidateTransaction(boolParams, transactionStateDB, bridgeStateDB, shardID, prvCoinID)
		if !ok {
			return false, err, i
		}
		if tx.GetMetadata() != nil {
			//if hasPrivacy {
			//	return false, errors.New("Metadata can not exist in privacy tx"), i
			//}
			validateMetadata := tx.GetMetadata().ValidateMetadataByItself()
			if !validateMetadata {
				return validateMetadata, utils.NewTransactionErr(utils.UnexpectedError, errors.New("Metadata is invalid")), i
			}
		}

		for _, batchableProof := range batchableProofs{
			bulletproof := batchableProof.GetAggregatedRangeProof()
			if bulletproof == nil {
				return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, fmt.Errorf("Privacy TX Proof missing at index %d", i)), -1
			}
			switch proof_specific := bulletproof.(type) {
			case *privacy.AggregatedRangeProofV1:
				bulletProofListVer1 = append(bulletProofListVer1, proof_specific)
			case *privacy.AggregatedRangeProofV2:
				bulletProofListVer2 = append(bulletProofListVer2, proof_specific)
			}
		}
	}
	//TODO: add go routine
	ok, err, i := aggregatedrange.VerifyBatchingAggregatedRangeProofs(bulletProofListVer1)
	if err != nil {
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, err), -1
	}
	if !ok {
		Logger.Log.Errorf("FAILED VERIFICATION BATCH PAYMENT PROOF VER 1 %d", i)
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, fmt.Errorf("FAILED VERIFICATION BATCH VER 1 PAYMENT PROOF %d", i)), -1
	}
	ok, err, i = bulletproofs.VerifyBatch(bulletProofListVer2)
	if err != nil {
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, err), -1
	}
	if !ok {
		Logger.Log.Errorf("FAILED VERIFICATION BATCH PAYMENT PROOF VER 2 %d", i)
		return false, utils.NewTransactionErr(utils.TxProofVerifyFailError, fmt.Errorf("FAILED VERIFICATION BATCH VER 2 PAYMENT PROOF %d", i)), -1
	}
	fmt.Println("[BUGLOG] Number of tx in batch", len(bulletProofListVer1), len(bulletProofListVer2))
	return true, nil, -1
}
