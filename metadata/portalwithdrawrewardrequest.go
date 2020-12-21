package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
	"strconv"
)

// PortalRequestWithdrawReward - custodians request withdraw reward
// metadata - custodians request withdraw reward - create normal tx with this metadata
type PortalRequestWithdrawReward struct {
	MetadataBase
	CustodianAddressStr string
	TokenID             common.Hash
}

// PortalRequestWithdrawRewardAction - shard validator creates instruction that contain this action content
type PortalRequestWithdrawRewardAction struct {
	Meta    PortalRequestWithdrawReward
	TxReqID common.Hash
	ShardID byte
}

// PortalRequestWithdrawRewardContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and rejected status
type PortalRequestWithdrawRewardContent struct {
	CustodianAddressStr string
	TokenID             common.Hash
	RewardAmount        uint64
	TxReqID             common.Hash
	ShardID             byte
}

// PortalRequestWithdrawRewardStatus - Beacon tracks status of request unlock collateral amount into db
type PortalRequestWithdrawRewardStatus struct {
	Status              byte
	CustodianAddressStr string
	TokenID             common.Hash
	RewardAmount        uint64
	TxReqID             common.Hash
}

func NewPortalRequestWithdrawReward(
	metaType int,
	incogAddressStr string,
	tokenID common.Hash) (*PortalRequestWithdrawReward, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}
	meta := &PortalRequestWithdrawReward{
		CustodianAddressStr: incogAddressStr,
		TokenID:             tokenID,
	}
	meta.MetadataBase = metadataBase
	return meta, nil
}

func (*PortalRequestWithdrawReward) ShouldSignMetaData() bool { return true }

func (meta PortalRequestWithdrawReward) ValidateTxWithBlockChain(
	txr Transaction,
	chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (meta PortalRequestWithdrawReward) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, txr Transaction) (bool, bool, error) {
	// validate CustodianAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(meta.CustodianAddressStr)
	if err != nil {
		return false, false, errors.New("Custodian incognito address is invalid")
	}
	incogAddr := keyWallet.KeySet.PaymentAddress
	if len(incogAddr.Pk) == 0 {
		return false, false, errors.New("Custodian incognito address is invalid")
	}
	if ok, err := txr.CheckAuthorizedSender(incogAddr.Pk); err != nil || !ok {
		return false, false, errors.New("Withdraw request sender is unauthorized")
	}

	// check tx type
	if txr.GetType() != common.TxNormalType {
		return false, false, errors.New("tx request withdraw reward must be TxNormalType")
	}

	return true, true, nil
}

func (meta PortalRequestWithdrawReward) ValidateMetadataByItself() bool {
	return meta.Type == PortalRequestWithdrawRewardMeta
}

func (meta PortalRequestWithdrawReward) Hash() *common.Hash {
	record := meta.MetadataBase.Hash().String()
	record += meta.CustodianAddressStr
	record += meta.TokenID.String()
	if meta.Sig != nil && len(meta.Sig) != 0 {
		record += string(meta.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (meta PortalRequestWithdrawReward) HashWithoutSig() *common.Hash {
	record := meta.MetadataBase.Hash().String()
	record += meta.CustodianAddressStr
	record += meta.TokenID.String()
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (meta *PortalRequestWithdrawReward) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PortalRequestWithdrawRewardAction{
		Meta:    *meta,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(PortalRequestWithdrawRewardMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (meta *PortalRequestWithdrawReward) CalculateSize() uint64 {
	return calculateSize(meta)
}
