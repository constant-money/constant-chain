package blockchain

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/pkg/errors"
)

func (blockchain *BlockChain) buildConvertingTransactionResponse(view *ShardBestState, txRequest *metadata.Transaction, blkProducerPrivateKey *privacy.PrivateKey, shardID byte) (metadata.Transaction, error) {
	tmpTxRequest := *txRequest
	var txParam transaction.TxSalaryOutputParams
	makeMD := func(c privacy.Coin) metadata.Metadata{return nil}

	switch tmpTxRequest.GetMetadataType() {
	case metadata.ConvertingRequestMeta:
		requestDetail, ok := tmpTxRequest.GetMetadata().(*metadata.ConvertingRequest)
		if !ok {
			return nil, fmt.Errorf("not a converting request: %v", tmpTxRequest.GetMetadata())
		}

		pubKeyBytes, _, err := base58.Base58Check{}.Decode(requestDetail.OTAStr)
		if err != nil {
			return nil, fmt.Errorf("cannot decode the OTAStr %v", requestDetail.OTAStr)
		}
		pubKeyPoint, err := new(privacy.Point).FromBytesS(pubKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("cannot set point from bytes: %v", pubKeyBytes)
		}

		txRandomBytes, _, err := base58.Base58Check{}.Decode(requestDetail.TxRandomStr)
		if err != nil {
			return nil, fmt.Errorf("cannot decode the TxRandomStr %v", requestDetail.TxRandomStr)
		}
		txRandom := new(privacy.TxRandom)
		err = txRandom.SetBytes(txRandomBytes)
		if err != nil {
			return nil, fmt.Errorf("cannot set txRandom from bytes: %v", txRandomBytes)
		}

		txParam = transaction.TxSalaryOutputParams{
			Amount:          requestDetail.ConvertedAmount,
			ReceiverAddress: nil,
			PublicKey:       pubKeyPoint,
			TxRandom:        txRandom,
			TokenID:         &requestDetail.TokenID,
		}

		responseMeta, err := metadata.NewConvertingResponse(requestDetail, tmpTxRequest.Hash())
		makeMD = func(c privacy.Coin) metadata.Metadata {
			return responseMeta
		}

		salaryTx, err := txParam.BuildTxSalary(blkProducerPrivateKey, view.GetCopiedTransactionStateDB(), makeMD)
		if err != nil {
			return nil, errors.Errorf("cannot init salary tx for conversion transaction. Error %v", err)
		}
		return salaryTx, nil
	default:
		return nil, fmt.Errorf("can not understand this request (%v)", tmpTxRequest.GetMetadataType())
	}
}