package metadata

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
	"strconv"
)

type WithDrawRewardRequest struct {
	MetadataBase
	PaymentAddress privacy.PaymentAddress
	TokenID common.Hash
	Version int
}

func (withdrawRequestMetadata *WithDrawRewardRequest) UnmarshalJSON(data []byte) error {
	tmp :=  &struct {MetadataBase
					PaymentAddress privacy.PaymentAddress
					TokenID common.Hash
					Version int}{}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if tmp.PaymentAddress.Pk == nil && tmp.PaymentAddress.Tk == nil {
		tmpOld :=  &struct {MetadataBase
							privacy.PaymentAddress
							TokenID common.Hash
							Version int}{}
		if err := json.Unmarshal(data, &tmpOld); err != nil {
			return err
		}

		tmp.PaymentAddress.Tk = tmpOld.Tk
		tmp.PaymentAddress.Pk = tmpOld.Pk
	}


	withdrawRequestMetadata.Type = tmp.Type
	withdrawRequestMetadata.PaymentAddress = tmp.PaymentAddress
	withdrawRequestMetadata.TokenID = tmp.TokenID
	withdrawRequestMetadata.Version = tmp.Version
	return nil
}


func (withDrawRewardRequest WithDrawRewardRequest) Hash() *common.Hash {
	if withDrawRewardRequest.Version == 1 {
		bArr := append(withDrawRewardRequest.PaymentAddress.Bytes(), withDrawRewardRequest.TokenID.GetBytes()...)
		if withDrawRewardRequest.Sig != nil && len(withDrawRewardRequest.Sig) != 0 {
			bArr = append(bArr, withDrawRewardRequest.Sig...)
		}
		txReqHash := common.HashH(bArr)
		return &txReqHash
	} else {
		record := strconv.Itoa(withDrawRewardRequest.Type)
		data := []byte(record)
		hash := common.HashH(data)
		return &hash
	}
}

func (withDrawRewardRequest WithDrawRewardRequest) HashWithoutSig() *common.Hash {
	if withDrawRewardRequest.Version == 1 {
		bArr := append(withDrawRewardRequest.PaymentAddress.Bytes(), withDrawRewardRequest.TokenID.GetBytes()...)
		txReqHash := common.HashH(bArr)
		return &txReqHash
	} else {
		record := strconv.Itoa(withDrawRewardRequest.Type)
		data := []byte(record)
		hash := common.HashH(data)
		return &hash
	}
}

func (*WithDrawRewardRequest) ShouldSignMetaData() bool { return true }

func NewWithDrawRewardRequest(tokenIDStr string, paymentAddStr string, version float64, metaType int) (*WithDrawRewardRequest, error) {
	metadataBase := NewMetadataBase(metaType)
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, errors.New("token ID is invalid")
	}
	paymentAddWallet, err := wallet.Base58CheckDeserialize(paymentAddStr)
	if err != nil {
		return nil, errors.New("payment address is invalid")
	}
	ok, err := common.SliceExists(AcceptedWithdrawRewardRequestVersion, int(version))
	if !ok || err != nil {
		return nil, errors.Errorf("Invalid version %v", version)
	}

	withdrawRewardRequest := &WithDrawRewardRequest{
		MetadataBase: *metadataBase,
		TokenID:  *tokenID,
		PaymentAddress: paymentAddWallet.KeySet.PaymentAddress,
		Version: int(version),
	}
	return withdrawRewardRequest, nil
}

func (withDrawRewardRequest WithDrawRewardRequest) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, stateDB *statedb.StateDB) bool {
	return true
}

func (withDrawRewardRequest WithDrawRewardRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	//check authorized sender
	if ok, err := tx.CheckAuthorizedSender(withDrawRewardRequest.PaymentAddress.Pk); err != nil || !ok {
		return false, fmt.Errorf("public key in withdraw reward request metadata is unauthorized. Error %v, OK %v", err, ok)
	}

	//check token valid (!= PRV)
	tokenIDReq :=  withDrawRewardRequest.TokenID
	isTokenValid := false
	if !tokenIDReq.IsEqual(&common.PRVCoinID) {
		allTokenID, err := chainRetriever.ListPrivacyTokenAndBridgeTokenAndPRVByShardID(common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte()))
		if err != nil {
			return false, err
		}
		for _, availableCoinID := range allTokenID {
			if cmp := tokenIDReq.IsEqual(&availableCoinID); cmp {
				isTokenValid = true
				break
			}
		}
		if !isTokenValid {
			return false, errors.New("Invalid TokenID, maybe this coin not available at current shard")
		}
	}
	pubKeyReqStr := base58.Base58Check{}.Encode(withDrawRewardRequest.PaymentAddress.Pk, common.Base58Version)
	//check available reward
	if rewardAmount, err := statedb.GetCommitteeReward(shardViewRetriever.GetShardRewardStateDB(), pubKeyReqStr, tokenIDReq); err != nil {
		return false, err
	} else if rewardAmount <= 0 {
		return false, errors.New("Not enough reward ")
	} else {
		return true, nil
	}
}

func (withDrawRewardRequest WithDrawRewardRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	return false, true, nil
}

func (withDrawRewardRequest WithDrawRewardRequest) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return true
}

func (withDrawRewardRequest WithDrawRewardRequest) GetType() int {
	return withDrawRewardRequest.Type
}

func (withDrawRewardRequest *WithDrawRewardRequest) CalculateSize() uint64 {
	return calculateSize(withDrawRewardRequest)
}
