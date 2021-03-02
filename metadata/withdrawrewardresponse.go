package metadata

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/pkg/errors"
	"strconv"
)

type WithDrawRewardResponse struct {
	MetadataBase
	TxRequest       *common.Hash
	TokenID         common.Hash
	RewardPublicKey []byte
	SharedRandom    []byte
	Version         int
}

func NewWithDrawRewardResponse(metaRequest *WithDrawRewardRequest, reqID *common.Hash) (*WithDrawRewardResponse, error) {
	metadataBase := MetadataBase{
		Type: WithDrawRewardResponseMeta,
	}
	result := &WithDrawRewardResponse{
		MetadataBase:    metadataBase,
		TxRequest:       reqID,
		TokenID:         metaRequest.TokenID,
		RewardPublicKey: metaRequest.PaymentAddress.Pk[:],
	}
	result.Version = metaRequest.Version
	if ok, err := common.SliceExists(AcceptedWithdrawRewardRequestVersion, result.Version); !ok || err != nil {
		return nil, errors.Errorf("Invalid version %d", result.Version)
	}
	return result, nil
}

func (withDrawRewardResponse WithDrawRewardResponse) Hash() *common.Hash {
	if withDrawRewardResponse.Version == 1 {
		if withDrawRewardResponse.TxRequest == nil {
			return &common.Hash{}
		}
		bArr := append(withDrawRewardResponse.TxRequest.GetBytes(), withDrawRewardResponse.TokenID.GetBytes()...)
		version := strconv.Itoa(withDrawRewardResponse.Version)
		if len(withDrawRewardResponse.SharedRandom) != 0 {
			bArr = append(bArr, withDrawRewardResponse.SharedRandom...)
		}
		if len(withDrawRewardResponse.RewardPublicKey) != 0 {
			bArr = append(bArr, withDrawRewardResponse.RewardPublicKey...)
		}

		bArr = append(bArr, []byte(version)...)
		txResHash := common.HashH(bArr)
		return &txResHash
	} else {
		return withDrawRewardResponse.TxRequest
	}
}

func (withDrawRewardResponse *WithDrawRewardResponse) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, db *statedb.StateDB) bool {
	return true
}

func (withDrawRewardResponse *WithDrawRewardResponse) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	return true, nil
}

//ValidateSanityData performs the following verifications:
//	1. Check tx type and supported tokenID
//	2. Check the mintedToken and convertedToken are the same
func (withDrawRewardResponse WithDrawRewardResponse) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	//Step 1
	if tx.GetType() == common.TxRewardType && withDrawRewardResponse.TokenID.String() != common.PRVIDStr {
		return false, false, NewMetadataTxError(UnexpectedError, fmt.Errorf("cannot mint token %v in a PRV transaction", withDrawRewardResponse.TokenID.String()))
	}
	if tx.GetType() == common.TxCustomTokenPrivacyType && withDrawRewardResponse.TokenID.String() == common.PRVIDStr {
		return false, false, NewMetadataTxError(UnexpectedError, fmt.Errorf("cannot mint PRV in a token transaction"))
	}

	//Step 2
	if tx.GetTokenID().String() != withDrawRewardResponse.TokenID.String() {
		return false, false, NewMetadataTxError(UnexpectedError, fmt.Errorf("mintedToken and withdrawToken mismatch: %v != %v", tx.GetTokenID().String(), withDrawRewardResponse.TokenID.String()))
	}
	return false, true, nil
}

func (withDrawRewardResponse WithDrawRewardResponse) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return true
}

func (withDrawRewardResponse *WithDrawRewardResponse) SetSharedRandom(r []byte) {
	withDrawRewardResponse.SharedRandom = r
}

//ValidateTxResponse validates if a withdraw response is a reply to the withdraw request by the following checks (logic obtained from accessor_transaction.go):
//	1. Check if the request metadata type is valid
//	2. Check the requested and minted tokenIDs
//	3. Check the reward amount and minted value
//	4. Check if minted coin is valid, i.e the minted coin is for the payment address in the txReq
// TODO: reviews should double-check if the above validations are sufficient
func (withDrawRewardResponse WithDrawRewardResponse) ValidateTxResponse(txReq, txResp Transaction, rewardAmount uint64) (bool, error) {
	Logger.log.Infof("ValidateTxResponse for withdraw req(%v)/resp(%v)\n", txReq.Hash().String(), txResp.Hash().String())
	//Step 1
	if txReq.GetMetadataType() != WithDrawRewardRequestMeta {
		return false, fmt.Errorf("txReq %v, type %v is not a withdraw request", txReq.Hash().String(), txReq.GetMetadataType())
	}
	tmpMeta := txReq.GetMetadata()
	reqMeta, ok := tmpMeta.(*WithDrawRewardRequest)
	if !ok {
		return false, fmt.Errorf("cannot parse metadata of txReq %v: %v", txReq.Hash().String(), tmpMeta)
	}

	//Step 2
	_, mintedCoin, mintedTokenID, err := txResp.GetTxMintData()
	if err != nil {
		return false, fmt.Errorf("cannot get tx minted data of txResp %v: %v", txResp.Hash().String(), err)
	}
	if reqMeta.TokenID.String() != mintedTokenID.String() {
		return false, fmt.Errorf("txReq mintedTokenID and txResp mintedTokenID mismatch: %v != %v", reqMeta.TokenID.String(), mintedTokenID.String())
	}

	//Step 3
	if mintedCoin.GetValue() != rewardAmount {
		return false, fmt.Errorf("reward amount (%v) and minted value (%v) mismatch", rewardAmount, mintedCoin.GetValue())
	}

	//Step 4
	if ok = mintedCoin.CheckCoinValid(reqMeta.PaymentAddress, withDrawRewardResponse.SharedRandom, rewardAmount); !ok {
		Logger.log.Errorf("[Mint Withdraw Reward] CheckMintCoinValid: %v, %v, %v, %v, %v\n", mintedCoin.GetVersion(), mintedCoin.GetValue(), mintedCoin.GetPublicKey(), reqMeta.PaymentAddress, reqMeta.PaymentAddress.GetPublicSpend().ToBytesS())
		return false, fmt.Errorf("the minted coin is not for the requester")
	}

	Logger.log.Infof("Finish ValidateTxResponse of pair req(%v)/resp(%v)\n", txReq.Hash().String(), txResp.Hash().String())

	return true, nil
}